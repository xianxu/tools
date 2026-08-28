package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// minDeckForReflection is the floor. Below it there is nothing worth reflecting
// on, and the failure mode is not an empty file — it is a CONFIDENT one, which
// then steers authoring. Absence already degrades cleanly (#16's ask path reads
// this file and answers generically without it), so writing nothing is strictly
// better than writing something generic and sure of itself.
const minDeckForReflection = 12

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

// DeckLookups counts found lookups of words STILL IN THE DECK — the ones
// From..To actually spans.
//
// Distinct from Lookups because the two have different scopes, and a message
// mixing them asserts a containment neither computes: "looked up N times between
// X and Y" was printed with the whole-log N against a deck-scoped span, so a
// forgotten word looked up years earlier inflated N while leaving X..Y untouched
// (BR-25). The prompt site is the one with teeth — a false premise handed to the
// model that writes the durable artifact.
//
// DERIVED, not stored. The first version was a field, and the golden fixture —
// which builds deckEvidence by hand rather than through foldLookups — silently
// carried 0, turning a true sentence into a different false one. A field would
// have to be kept in step with Words at every construction site, which is the
// list-that-drifts shape this file has now been burned by twice; summing Words
// cannot disagree with Words.
func (e deckEvidence) DeckLookups() int {
	var n int
	for _, w := range e.Words {
		n += w.Lookups
	}
	return n
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
	Band      string `json:"band"`
	Rationale string `json:"rationale"`
	// evidence_words, not "evidence": SchemaFor reflects the field NAME to the
	// model, and "evidence" invited whole sentences here — measured against the
	// live model, which returned "Advanced specialist items looked up:
	// 'certiorari', 'estoppel' — near-native legal register" as one element and
	// lost the entire level claim to the deck check. Domains, which have no
	// rationale field to compete with, cited bare words correctly.
	EvidenceWords []string `json:"evidence_words"`
}

// domainClaim is one domain the learner reads in, with the share of the deck it
// accounts for, the words that are the evidence, and what authoring should DO
// about it — the directive is the whole reason the model is worth generating.
type domainClaim struct {
	Name          string   `json:"name"`
	Share         float64  `json:"share"`
	EvidenceWords []string `json:"evidence_words"`
	Directive     string   `json:"directive"`
}

// cite renders one piece of model text for a DIAGNOSTIC, and neutralises it on
// the way.
//
// The file is not the only structured output here: these messages go to a
// terminal one per line, and a band or a domain name carrying a newline forges a
// second diagnostic — a fabricated "define: ..." line the reader has no way to
// tell from a real one. The sweep that found this also found the frontmatter's
// model name; the class is untrusted text reaching ANY structured output, and
// the file's table was simply the instance a finding named first.
func cite(s string) string {
	if strings.TrimSpace(s) == "" {
		return "nothing"
	}
	return oneLine(s)
}

// citeAll is cite over a list. A claim that cited NOTHING is a different failure
// from one that cited words we do not have, and the message must tell them apart
// or it cannot be acted on.
func citeAll(words []string) string {
	if len(words) == 0 {
		return "nothing"
	}
	out := make([]string, 0, len(words))
	for _, w := range words {
		out = append(out, oneLine(w))
	}
	return strings.Join(out, ", ")
}

// dropClaim is one rejected claim, carried as DATA so that neutralising it
// happens in exactly one place.
//
// Every arm used to build its own string with cite(subject) inline — five call
// sites, each of which had to remember. That is the list-that-drifts shape
// renderUserModel was refactored away from, reproduced here in the diagnostics
// (BR-17). MEASURED at c179efa: unsanitising the share-out-of-range arm reddened
// ZERO tests, because no test constructed a subject carrying a newline at three
// of the five arms.
//
// The claim is stated as a measurement because the first version of this comment
// asserted a broader one — "deleting the sanitiser left the whole suite green" —
// that was false for the fourth site, sanitiseMeta, which a test added the round
// before already covered (BR-21). A comment that asserts coverage is a claim
// about a mutation someone has to have run.
//
// With the subject and citations held as fields, an arm CANNOT forget: String is
// the only path to text, and it is where oneLine happens.
type dropClaim struct {
	Kind    string // authored here: "level" or "domain"
	Subject string // UNTRUSTED — model-supplied, neutralised by String
	Reason  string // authored here; unused when Cites is set
	// Cites marks a claim rejected FOR its citations, and Cited is what it named.
	//
	// The flag exists because an EMPTY Cited is not the absence of a citation
	// complaint — it is a claim that cited nothing at all, which citeAll renders
	// as "nothing" precisely so a reader can tell it apart from a claim that
	// cited words the deck lacks. Branching on len(Cited) instead collapsed the
	// two: the message stopped at "level C1: cites" mid-clause, and citeAll's
	// empty branch became unreachable code with a comment explaining a
	// distinction nothing made any more (BR-22).
	Cites bool
	Cited []string // UNTRUSTED
}

// String is the ONE place a dropped claim becomes text.
func (d dropClaim) String() string {
	msg := d.Kind + " " + cite(d.Subject) + ": "
	if d.Cites {
		return msg + "cites " + citeAll(d.Cited) + ", none of which is in the deck"
	}
	return msg + d.Reason
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
func checkEvidence(m learnerModel, deck map[string]bool) (learnerModel, []dropClaim) {
	var dropped []dropClaim
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

	// The level is a claim like any other, and gets BOTH arms: unsupported
	// evidence, and unusable content. The first version checked only evidence
	// here while checking name-and-directive on domains — an asymmetry with no
	// reason behind it, and a band with an empty rationale renders as "**C1** —"
	// with nothing after the dash.
	switch ev := supported(m.Level.EvidenceWords); {
	case strings.TrimSpace(m.Level.Band) == "" || strings.TrimSpace(m.Level.Rationale) == "":
		dropped = append(dropped, dropClaim{Kind: "level", Subject: m.Level.Band,
			Reason: "no band or no rationale — nothing a reader could check"})
		m.Level = levelClaim{}
	case len(ev) > 0:
		m.Level.EvidenceWords = ev
	default:
		// Naming the REJECTED words, not just the fact: "no evidence in the
		// deck" is the same unactionable shape as a claim that names none, and
		// this message is the only place a person can see WHY the level went
		// missing from their file.
		dropped = append(dropped, dropClaim{Kind: "level", Subject: m.Level.Band,
			Cites: true, Cited: m.Level.EvidenceWords})
		m.Level = levelClaim{}
	}

	var kept []domainClaim
	for _, d := range m.Domains {
		// A claim that cannot be ACTED on is not a claim. Measured against the
		// live model: under a schema that requires every field, it fills the
		// ones it does not believe in — a domain literally named "x", a
		// rationale of "placeholder" — because emitting a stub satisfies the
		// shape. A domain with no name or no directive tells authoring nothing,
		// which is the only reason a domain claim is generated at all.
		if strings.TrimSpace(d.Name) == "" || strings.TrimSpace(d.Directive) == "" {
			dropped = append(dropped, dropClaim{Kind: "domain", Subject: d.Name,
				Reason: "no name or no directive — nothing authoring could act on"})
			continue
		}
		if d.Share < 0 || d.Share > 1 {
			// Rendered as a percentage, so 5.0 becomes "500%" — a number a
			// reader cannot act on and would not believe.
			dropped = append(dropped, dropClaim{Kind: "domain", Subject: d.Name,
				Reason: "share out of range"})
			continue
		}
		ev := supported(d.EvidenceWords)
		if len(ev) == 0 {
			dropped = append(dropped, dropClaim{Kind: "domain", Subject: d.Name,
				Cites: true, Cited: d.EvidenceWords})
			continue
		}
		d.EvidenceWords = ev
		kept = append(kept, d)
	}
	m.Domains = kept
	return m, dropped
}

// runReflect folds the directory into user-model.md.
//
// Thin on purpose: read, fold, ask, check, render, splice, write. Every decision
// it looks like it makes belongs to one of the pure functions above, which is
// what lets "what would this write" be a table test rather than a store and a
// socket (ARCH-PURE).
//
// No model call ever moves onto the lookup or review path: this is a MODE, run
// on demand, and that is the whole reason the analysis is batch (#17's Spec).
func runReflect(ctx context.Context, d deps, opt options, out, errOut io.Writer) int {
	if d.deck == nil {
		fmt.Fprintln(errOut, noDeckMessage(opt.noCapture))
		return 1
	}
	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the deck: %v\n", err)
		return 1
	}
	events, err := d.deck.Events(time.Time{})
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the event log: %v\n", err)
		return 1
	}

	ev := foldLookups(deck, events, d.clock.Now())
	if len(ev.Words) < minDeckForReflection {
		// Said with both numbers, so it reads as a threshold rather than a
		// refusal: the learner can see how far off they are.
		fmt.Fprintf(errOut, "define: %d words in the deck; --reflect needs %d to say anything worth reading\n",
			len(ev.Words), minDeckForReflection)
		return 1
	}

	if d.getenv == nil || d.newLLM == nil {
		return unavailableToReflect(errOut)
	}
	cfg, err := llm.Resolve(d.getenv)
	if err != nil {
		return unavailableToReflect(errOut)
	}
	model, err := llm.Run(ctx, d.newLLM(cfg), llm.Task[learnerModel]{
		Name:   reflectTaskName,
		System: reflectSystem,
		Prompt: renderReflectPrompt(ev).Prompt,
		// Roomier than the 8192 default, because this answer is genuinely long
		// — four domains with paragraph-length directives — and it shares the
		// budget with high-effort thinking. Measured: at the default, runs came
		// back INTERMITTENTLY degenerate, with an empty band and domains named
		// "x", while others were excellent. Not a truncation (Run checks the
		// stop reason first, and it was end_turn), but the same squeeze that
		// produced #11's preserved max_tokens specimen: thinking eats the
		// budget and the answer gets the remainder.
		MaxTokens: 16384,
	})
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the deck's shape: %v\n", err)
		return 1
	}

	inDeck := make(map[string]bool, len(ev.Words))
	for _, w := range ev.Words {
		inDeck[w.Word] = true
	}
	model, dropped := checkEvidence(model, inDeck)
	for _, why := range dropped {
		// Out loud, always: a silently shorter file is indistinguishable from a
		// model that had less to say.
		fmt.Fprintf(errOut, "define: dropped %s\n", why)
	}

	// Nothing survived the check: do not write a file that says "here is what we
	// know about you" and knows nothing. That is D2's floor arriving through a
	// different door — an empty file reads as an answer, and #16's ask path
	// degrades cleanly on absence but not on emptiness.
	if model.Level.Band == "" && len(model.Domains) == 0 {
		fmt.Fprintln(errOut, "define: nothing in the answer survived checking against the deck; user-model.md not written")
		return 1
	}

	existing, err := d.deck.UserModel()
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the existing user-model.md: %v\n", err)
		return 1
	}
	generated := renderUserModel(model, modelMeta{
		Updated:   d.clock.Now(),
		From:      ev.From,
		To:        ev.To,
		Lookups:   ev.DeckLookups(), // BR-25: the count beside window: must be the count within it
		Questions: ev.Questions,
		Model:     cfg.Model,
	})
	if err := d.deck.SetUserModel(spliceCorrections(existing, generated)); err != nil {
		fmt.Fprintf(errOut, "define: could not write user-model.md: %v\n", err)
		return 1
	}

	fmt.Fprintf(out, "define: wrote user-model.md from %d words\n", len(ev.Words))
	return 0
}

// unavailableToReflect is the degradation path, and it says the same thing the
// ask path says for the same condition — one fact, one sentence.
func unavailableToReflect(errOut io.Writer) int {
	fmt.Fprintln(errOut, "define: no model configured; --reflect needs one")
	return 1
}
