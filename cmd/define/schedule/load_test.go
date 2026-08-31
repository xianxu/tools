package schedule

import (
	"math"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func deckOf(words ...string) []store.Word {
	var d []store.Word
	for _, w := range words {
		d = append(d, store.Word{Text: w})
	}
	return d
}

// THE COST OF A DECK IS Σ 1/interval, and a word reviewed every 68 days costs
// 1/68 of a review per day.
func TestDailyLoad(t *testing.T) {
	deck := deckOf("a", "b")
	prog := map[string]Progress{
		"a": {Box: 5, MaxBox: 5}, // 10-day interval
		"b": {Box: 9, MaxBox: 9}, // 68-day interval
	}
	want := 1.0/10 + 1.0/68
	if got := DailyLoad(deck, prog); math.Abs(got-want) > 1e-9 {
		t.Errorf("DailyLoad = %v, want %v", got, want)
	}
	// An empty deck costs nothing, and must not divide by anything.
	if got := DailyLoad(nil, nil); got != 0 {
		t.Errorf("an empty deck costs %v, want 0", got)
	}
}

// NEVER-REVIEWED DECK WORDS COUNT, and this is the assertion that matters.
//
// Fold returns an entry only for words with a review event, so a deck of 20 with
// 2 reviewed folds to 2 entries. Summing the MAP would report a tenth of the
// true cost — and it would understate exactly when the learner most needs the
// warning, which is when a large backlog of never-reviewed words has built up.
func TestDailyLoadCountsUnreviewedWords(t *testing.T) {
	deck := deckOf("a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	prog := map[string]Progress{"a": {Box: 9, MaxBox: 9}} // one reviewed, nine not

	// Nine unreviewed words are box 0 — a one-day interval, one review each per
	// day — plus the mature one.
	want := 9.0 + 1.0/68
	got := DailyLoad(deck, prog)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("DailyLoad = %v, want %v — a deck of ten with one reviewed must not "+
			"report the cost of one", got, want)
	}
	if got := DailyLoad(deck, prog); got < float64(len(deck))/2 {
		t.Errorf("DailyLoad = %v for a mostly-unreviewed deck of %d; the fold's map "+
			"is being summed instead of the deck", got, len(deck))
	}
}

// reviewsInFirstYear DERIVES from the ladder rather than being typed, so a
// change to the ratio cannot leave it stale.
func TestReviewsInFirstYearDerives(t *testing.T) {
	got := reviewsInFirstYear()

	// Recomputed here by walking the ladder independently of the implementation.
	want, day := 0, 0
	for b := 0; b <= ladderLimit; b++ {
		day += IntervalDays(b)
		if day > 365 {
			break
		}
		want++
	}
	if got != want {
		t.Errorf("reviewsInFirstYear = %d, want %d", got, want)
	}
	// And the value must be plausible: a ladder that reviewed a new word once or
	// fifty times in its first year would make SustainableNewWords nonsense.
	if got < 5 || got > 20 {
		t.Errorf("a new word is reviewed %d times in its first year, which is outside "+
			"any sane range for this ladder", got)
	}
}

// The budget left over, divided by what a new word costs in its first year.
func TestSustainableNewWords(t *testing.T) {
	deck := deckOf("a")
	prog := map[string]Progress{"a": {Box: 9, MaxBox: 9}} // ~0.015/day

	got := SustainableNewWords(20, deck, prog)
	want := (20 - DailyLoad(deck, prog)) / float64(reviewsInFirstYear())
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("SustainableNewWords = %v, want %v", got, want)
	}

	// A deck already over budget sustains NOTHING, and must not report a
	// negative number of words — which would read as "you may add -3 words".
	big := deckOf("a", "b", "c", "d", "e")
	if got := SustainableNewWords(1, big, nil); got != 0 {
		t.Errorf("an over-budget deck sustains %v new words, want 0", got)
	}
	// A budget of zero is not "unlimited", the same reading Queue refuses.
	if got := SustainableNewWords(0, deck, prog); got != 0 {
		t.Errorf("a zero budget sustains %v, want 0", got)
	}
}
