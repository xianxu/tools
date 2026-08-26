package main

import (
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// deckEvidence is what the model is shown: one row per word, plus the totals.
//
// NOT the raw event log. A year of it is thousands of rows, and the claims we
// want back are about WORDS — how often, how recently, in what company — not
// about individual lookups (#17 D6).
type deckEvidence struct {
	// Words is the evidence a claim may cite, and nothing else may be cited.
	// historyRow, not a new row type: /history already needed exactly this
	// shape, and a second one would be a second answer to "what do we know
	// about this word".
	Words []historyRow
	// Lookups counts every FOUND lookup in the log, including words the deck no
	// longer holds — the total is a fact about the learner, not about the deck.
	Lookups int
	// Questions counts #16's asked events, apart from lookups: a question is not
	// a lookup, and conflating them inflates every share the model reasons over.
	Questions int
	// From..To is the span the claims may speak about. Zero on an empty deck,
	// deliberately: a zero-day window is something a prompt would assert over.
	From, To time.Time
}

// foldLookups summarises the store for one --reflect run.
//
// The per-word fold is summariseLookups', called with a zero `since` so it
// covers the whole log: the same EventLookedUp-and-Found filter, the same
// store.Key keying and the same FirstAt/LastAt/Lookups accumulation /history
// needs. Writing a second one would have been a second answer to "how many
// times has this learner looked this up" (ARCH-DRY).
//
// What this adds is the part that is genuinely new: the questions count, the
// window, and D7's rule — the DECK decides which words are evidence, the LOG
// decides how many times and when. A word `--forget` removed therefore stops
// being evidence while its history still counts toward the totals, which is
// exactly what --forget promises: the deck is a working set, the log is history.
//
// Pure: the instant is a parameter rather than a clock, so "what does the window
// mean when the deck spans one day" is a table row and not a timing test.
func foldLookups(deck []store.Word, events []store.ReviewEvent, now time.Time) deckEvidence {
	var ev deckEvidence
	for _, e := range events {
		switch {
		case e.Kind == store.EventAsked:
			ev.Questions++
		case e.Kind == store.EventLookedUp && e.Found:
			ev.Lookups++
		}
	}

	inDeck := make(map[string]bool, len(deck))
	for _, w := range deck {
		if k := store.Key(w.Text); k != "" {
			inDeck[k] = true
		}
	}
	for _, r := range summariseLookups(events, time.Time{}) {
		if !inDeck[r.Word] {
			continue // forgotten: history, not evidence
		}
		ev.Words = append(ev.Words, r)
		if ev.From.IsZero() || r.FirstAt.Before(ev.From) {
			ev.From = r.FirstAt
		}
		if r.LastAt.After(ev.To) {
			ev.To = r.LastAt
		}
	}
	return ev
}

// learnerModel is the typed answer.
//
// SchemaFor[learnerModel] reflects the JSON schema from this struct, so the
// shape has one source and a field added without thought shows up in the
// golden's diff (internal/llm's typed-task contract).
type learnerModel struct {
	Level   levelClaim    `json:"level"`
	Domains []domainClaim `json:"domains"`
}

// levelClaim is the working band, and it is a CLAIM: it carries the deck words
// it was read off, and is dropped if none of them exist.
type levelClaim struct {
	Band      string   `json:"band"`
	Rationale string   `json:"rationale"`
	Evidence  []string `json:"evidence"`
}

// domainClaim is one domain the learner reads in, with the share of the deck it
// accounts for, the words that are the evidence, and what authoring should DO
// about it — the directive is the whole reason the model is worth generating.
type domainClaim struct {
	Name      string   `json:"name"`
	Share     float64  `json:"share"`
	Evidence  []string `json:"evidence"`
	Directive string   `json:"directive"`
}

// checkEvidence enforces D1: a claim may cite only words the deck holds.
//
// The model may READ the deck and may not ADD to it — the same rule this project
// applies to distractors, which are selected and never invented. Without it,
// "every claim names its evidence" is a formatting convention: a fabricated
// sailing domain citing `luffing` and `clew` is exactly as checkable-looking as
// a real one, and the file's whole promise is that a claim CAN be checked.
//
// A partially supported claim keeps the evidence that exists rather than being
// dropped whole: the domain may well be real even where one citation is not.
// Returns what it dropped, so the caller can say so out loud rather than
// silently shipping a shorter file.
func checkEvidence(m learnerModel, deck map[string]bool) (learnerModel, []string) {
	var dropped []string
	// Matched on store.Key, the deck's own identity: raw matching would drop
	// every capitalised or double-spaced citation as if it were invented.
	supported := func(words []string) []string {
		var out []string
		for _, w := range words {
			if deck[store.Key(w)] {
				out = append(out, w)
			}
		}
		return out
	}

	if ev := supported(m.Level.Evidence); len(ev) > 0 {
		m.Level.Evidence = ev
	} else {
		dropped = append(dropped, "level "+m.Level.Band+": no evidence in the deck")
		m.Level = levelClaim{}
	}

	var kept []domainClaim
	for _, d := range m.Domains {
		ev := supported(d.Evidence)
		if len(ev) == 0 {
			dropped = append(dropped, "domain "+d.Name+": no evidence in the deck")
			continue
		}
		d.Evidence = ev
		kept = append(kept, d)
	}
	m.Domains = kept
	return m, dropped
}
