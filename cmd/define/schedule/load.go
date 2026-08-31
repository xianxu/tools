package schedule

import "github.com/xianxu/tools/cmd/define/store"

// What the deck costs, so the learner can see it.
//
// The number that should govern how many new words someone takes on is reviews
// per day, and before this it was computed nowhere and shown nowhere — a growing
// backlog was the only way to discover it, which is the worst possible feedback
// loop for a tool whose whole subject is spaced feedback.

// DailyLoad is how many reviews per day this deck costs in steady state.
//
// A word in box b comes round every IntervalDays(b) days, so it costs
// `1/IntervalDays(b)` reviews per day; the deck's cost is the sum. That is why
// an unbounded geometric ladder is affordable and a capped one is not: with a
// ceiling the stock of parked words grows linearly forever, while with intervals
// that keep growing a word's cost falls as fast as the deck accumulates.
//
// IT TAKES THE DECK, not just the folded progress, and that is the whole
// correctness question. Fold returns an entry only for words that have a review
// event, so a deck of 500 with 50 reviewed folds to 50 entries. Summing the map
// alone would report a tenth of the true cost — and it would understate exactly
// when the warning matters most, which is when a large backlog of never-reviewed
// words has built up. A missing entry is the zero Progress: box 0, a one-day
// interval, one review per day.
func DailyLoad(deck []store.Word, prog map[string]Progress) float64 {
	var load float64
	seen := make(map[string]bool, len(deck))
	for _, w := range deck {
		// Deduped by KEY for the same reason Queue dedupes: the deck can hold
		// two rows that store.Key collapses into one word, and counting both
		// would overstate the cost of a hand-edited deck.
		key := store.Key(w.Text)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		load += 1 / float64(IntervalDays(prog[key].Box))
	}
	return load
}

// SustainableNewWords is how many words a day the learner can take on and still
// finish their reviews.
//
// A word admitted today generates `reviewsInFirstYear()` reviews over its first
// year, and in steady state admitting `a` words a day therefore costs `a × that`
// reviews a day. So whatever budget is left after the existing deck divides by
// it.
//
// Never negative: a deck already over budget sustains nothing, and reporting
// "-3 new words a day" would be a number with no meaning. A zero budget is not
// "unlimited" — the same reading Queue refuses, for the same reason.
func SustainableNewWords(budget int, deck []store.Word, prog map[string]Progress) float64 {
	if budget <= 0 {
		return 0
	}
	spare := float64(budget) - DailyLoad(deck, prog)
	if spare <= 0 {
		return 0
	}
	return spare / float64(reviewsInFirstYear())
}

// reviewsInFirstYear walks the ladder and counts the reviews a brand-new word
// earns in its first 365 days.
//
// DERIVED rather than typed. It is 11 for the current ratio, and writing 11 down
// would make it a second encoding of the ladder that a ratio change would leave
// silently stale — the same defect this package avoids by deriving Progress from
// the log rather than storing it.
func reviewsInFirstYear() int {
	n, day := 0, 0
	for b := 0; b <= ladderLimit; b++ {
		day += IntervalDays(b)
		if day > 365 {
			break
		}
		n++
	}
	return n
}
