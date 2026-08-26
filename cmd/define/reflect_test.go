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
