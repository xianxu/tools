package main

import (
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

func reflectDay(n int) time.Time {
	return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -n)
}

// D7: the DECK decides which words are evidence, the LOG decides how many times
// and when.
//
// The rule follows from what --forget already promises — it removes a word from
// the deck and deliberately keeps its events, because "the deck is a working
// set, the log is history". So a forgotten word must stop being evidence while
// its history still counts toward the totals.
func TestFoldLookupsTakesMembershipFromTheDeckAndCountsFromTheLog(t *testing.T) {
	deck := []store.Word{
		{Text: "certiorari", FirstSeen: reflectDay(3), LastSeen: reflectDay(1), Lookups: 99},
	}
	events := []store.ReviewEvent{
		{Word: "certiorari", Kind: store.EventLookedUp, Found: true, At: reflectDay(3)},
		{Word: "certiorari", Kind: store.EventLookedUp, Found: true, At: reflectDay(1)},
		// forgotten: --forget removed it from the deck, its events remain
		{Word: "estoppel", Kind: store.EventLookedUp, Found: true, At: reflectDay(2)},
	}

	got := foldLookups(deck, events, reflectDay(0))

	if len(got.Words) != 1 || got.Words[0].Word != "certiorari" {
		t.Fatalf("words = %+v, want only the word still in the deck", got.Words)
	}
	// The COUNT comes from the log, not from store.Word.Lookups — 99 is the
	// deck's own tally and would be a second answer to one question.
	if got.Words[0].Lookups != 2 {
		t.Errorf("lookups = %d, want 2 from the log", got.Words[0].Lookups)
	}
	// The forgotten word still counts toward the total: its history is real.
	if got.Lookups != 3 {
		t.Errorf("total = %d, want every found lookup in the log", got.Lookups)
	}
}

// D6: a miss is not vocabulary and an ask is not a lookup. Conflating them
// inflates every share the model is asked to reason about.
func TestFoldLookupsCountsAsksAndMissesApartFromLookups(t *testing.T) {
	deck := []store.Word{{Text: "sycophantic", FirstSeen: reflectDay(2), LastSeen: reflectDay(2), Lookups: 1}}
	events := []store.ReviewEvent{
		{Word: "sycophantic", Kind: store.EventLookedUp, Found: true, At: reflectDay(2)},
		{Word: "sycophanti", Kind: store.EventLookedUp, Found: false, At: reflectDay(2)},
		{Word: "sycophantic", Kind: store.EventAsked, Question: "vs obsequious?", At: reflectDay(1)},
	}

	got := foldLookups(deck, events, reflectDay(0))

	if got.Lookups != 1 {
		t.Errorf("lookups = %d, want only the found one", got.Lookups)
	}
	if got.Questions != 1 {
		t.Errorf("questions = %d, want the asked event counted apart", got.Questions)
	}
	if len(got.Words) != 1 {
		t.Errorf("words = %+v, want the typo excluded from the evidence", got.Words)
	}
}

func TestFoldLookupsWindowSpansTheEvidence(t *testing.T) {
	deck := []store.Word{
		{Text: "a", FirstSeen: reflectDay(9), LastSeen: reflectDay(9), Lookups: 1},
		{Text: "b", FirstSeen: reflectDay(1), LastSeen: reflectDay(1), Lookups: 1},
	}
	events := []store.ReviewEvent{
		{Word: "a", Kind: store.EventLookedUp, Found: true, At: reflectDay(9)},
		{Word: "b", Kind: store.EventLookedUp, Found: true, At: reflectDay(1)},
	}

	got := foldLookups(deck, events, reflectDay(0))

	if !got.From.Equal(reflectDay(9)) || !got.To.Equal(reflectDay(1)) {
		t.Errorf("window = %v..%v, want oldest..newest across the evidence", got.From, got.To)
	}
}

// An empty fold, not a zero-day window the prompt would then assert over.
func TestFoldLookupsOnAnEmptyDeck(t *testing.T) {
	got := foldLookups(nil, nil, reflectDay(0))

	if len(got.Words) != 0 {
		t.Errorf("words = %+v, want none", got.Words)
	}
	if !got.From.IsZero() || !got.To.IsZero() {
		t.Errorf("window = %v..%v, want zero — there is nothing to speak about", got.From, got.To)
	}
}

// D1: the model may READ the deck and may not ADD to it.
//
// This is the project's existing rule — distractors are selected, never invented
// — applied to the learner model. Without it, "every claim names its evidence"
// is a formatting convention that a plausible hallucination satisfies: a sailing
// domain citing `luffing` and `clew` looks exactly as checkable as a legal one
// citing words the learner actually looked up.
func TestCheckEvidenceDropsClaimsTheDeckCannotSupport(t *testing.T) {
	deck := map[string]bool{"certiorari": true, "dicta": true, "ephemeral": true}
	in := learnerModel{
		Level: levelClaim{Band: "C1", Evidence: []string{"certiorari", "dicta"}},
		Domains: []domainClaim{
			{Name: "law", Share: 0.6, Evidence: []string{"certiorari", "dicta"}},
			{Name: "sailing", Share: 0.2, Evidence: []string{"luffing", "clew"}},
			{Name: "mixed", Share: 0.2, Evidence: []string{"ephemeral", "luffing"}},
		},
	}

	got, dropped := checkEvidence(in, deck)

	if len(got.Domains) != 2 {
		t.Fatalf("domains = %+v, want the wholly-unsupported one dropped", got.Domains)
	}
	if got.Domains[0].Name != "law" {
		t.Errorf("kept %q first, want law", got.Domains[0].Name)
	}
	// A PARTIALLY supported claim keeps the evidence that exists rather than
	// being dropped whole: "mixed" is a real domain, "luffing" is not a word.
	if len(got.Domains[1].Evidence) != 1 || got.Domains[1].Evidence[0] != "ephemeral" {
		t.Errorf("mixed evidence = %v, want only the deck word", got.Domains[1].Evidence)
	}
	if len(dropped) == 0 {
		t.Error("nothing reported: a dropped claim must be sayable out loud")
	}
}

// The level band is a claim like any other. Unsupported, it must not reach the
// file — an asserted band is exactly what the issue's spec forbids.
func TestCheckEvidenceDropsALevelClaimWithNoSupport(t *testing.T) {
	deck := map[string]bool{"certiorari": true}
	in := learnerModel{Level: levelClaim{Band: "C2", Evidence: []string{"luffing"}}}

	got, dropped := checkEvidence(in, deck)

	if got.Level.Band != "" {
		t.Errorf("band = %q, want it dropped — nothing in the deck supports it", got.Level.Band)
	}
	if len(dropped) == 0 {
		t.Error("a dropped level claim must be reported")
	}
}

// The deck's identity is case- and space-normalised (store.Key), so evidence
// must match the same way or every capitalised citation is dropped as invented.
func TestCheckEvidenceMatchesOnTheDeckKey(t *testing.T) {
	deck := map[string]bool{"hot dog": true, "certiorari": true}
	in := learnerModel{
		Level:   levelClaim{Band: "B2", Evidence: []string{"Certiorari"}},
		Domains: []domainClaim{{Name: "food", Share: 0.5, Evidence: []string{"Hot  Dog"}}},
	}

	got, dropped := checkEvidence(in, deck)

	if got.Level.Band != "B2" {
		t.Errorf("a capitalised citation was dropped: %+v", got.Level)
	}
	if len(got.Domains) != 1 {
		t.Errorf("domains = %+v, want the multi-word citation kept", got.Domains)
	}
	if len(dropped) != 0 {
		t.Errorf("dropped %v, want nothing — both cite real deck words", dropped)
	}
}

// An empty deck supports nothing, and must not be read as supporting everything.
func TestCheckEvidenceAgainstAnEmptyDeck(t *testing.T) {
	got, dropped := checkEvidence(learnerModel{
		Level:   levelClaim{Band: "C1", Evidence: []string{"anything"}},
		Domains: []domainClaim{{Name: "law", Evidence: []string{"certiorari"}}},
	}, map[string]bool{})

	if got.Level.Band != "" || len(got.Domains) != 0 {
		t.Errorf("got %+v, want everything dropped", got)
	}
	if len(dropped) != 2 {
		t.Errorf("dropped %v, want both claims reported", dropped)
	}
}
