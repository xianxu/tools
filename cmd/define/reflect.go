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
