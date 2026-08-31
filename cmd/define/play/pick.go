package play

// Option selection: the pure heart of form 2.3.
//
// Deck + target + seed in, a question out, and the same three in always give the
// same one — which is what makes the form testable at all, and what Done-when 3
// pins. No imports, per this package's guard: the shuffle is hand-rolled below
// and the reasons are with it.

// Candidate is one possible distractor: a headword, ONE of its glosses, and the
// axis that gloss supplies.
//
// Pre-expanded by package main, one Candidate per (word, sense) pair worth
// offering. That is why this type carries a single Axis rather than a set: D4a
// says a distractor's sense is the one CARRYING the axis being sought, so the
// pairing is decided where the dictionary is, and arrives here already made.
// `record` legitimately appears three times — its Law sense, its plastic-disk
// sense, its best-performance sense — and selection makes sure only one of them
// reaches a given question.
type Candidate struct {
	Word  string
	Gloss string
	Axis  Axis
}

// maxOptions is four. Not a tuning knob: four is what the issue specifies, and
// the number the 1-4 key range in Grade can express.
const maxOptions = 4

// pickOptions builds one question.
//
// Two passes, and the split is the whole design:
//
//  1. one distractor per axis, in distractorAxes' priority order — scarce first,
//     so a plentiful axis cannot crowd out a rare one;
//  2. top up the remaining slots from whatever is left.
//
// Pass 2 exists because three axes and three slots only line up when the deck
// has all three. A deck with no domain-labelled word would otherwise return a
// three-option question forever, which is a worse question than one with two
// general distractors — D2a's "a question is never blocked on an axis being
// available", made concrete.
//
// The result is SHUFFLED, so the answer does not sit in slot 1 every time.
func pickOptions(target Candidate, pool []Candidate, seed uint64) []Option {
	used := map[string]bool{target.Word: true}
	var distractors []Candidate

	take := func(c Candidate) {
		used[c.Word] = true
		distractors = append(distractors, c)
	}

	// Pass 1: one per axis, scarcest axis first.
	for _, want := range distractorAxes {
		if len(distractors) >= maxOptions-1 {
			break
		}
		for _, c := range pool {
			if c.Axis == want && !used[c.Word] {
				take(c)
				break
			}
		}
	}
	// Pass 2: fill what is left, in pool order.
	for _, c := range pool {
		if len(distractors) >= maxOptions-1 {
			break
		}
		if !used[c.Word] {
			take(c)
		}
	}

	// Below one distractor there is nothing to choose BETWEEN, so this is not a
	// question. Returning nothing rather than a one-option question is what lets
	// the caller fall back to form 2.1 (D9) instead of showing a learner a
	// multiple choice with a single answer.
	if len(distractors) == 0 {
		return nil
	}

	opts := []Option{{Gloss: target.Gloss, Word: target.Word, Correct: true}}
	for _, c := range distractors {
		opts = append(opts, Option{Gloss: c.Gloss, Word: c.Word, Axis: c.Axis})
	}
	shuffle(opts, seed)
	return opts
}

// shuffle is Fisher-Yates over a hand-rolled xorshift64.
//
// NOT math/rand, and the reason is stronger than this package's import guard.
// Done-when 3 claims a fixed seed gives the same question — forever, so that a
// question can be reproduced from a log. math/rand's sequence for a given seed
// is a property of the Go runtime, and the top-level generator's behaviour has
// already changed once across versions. A PRNG defined here is pinned by this
// repo's own tests, which is the guarantee the Done-when actually needs.
//
// xorshift64 is not cryptographic and does not need to be: it is choosing which
// of four definitions goes first.
func shuffle(opts []Option, seed uint64) {
	// A zero seed is a fixed point for xorshift — it would emit zeros forever
	// and shuffle nothing, silently. Seed 0 is also exactly what a caller passes
	// before wiring a real one, so this is the case that would ship.
	state := seed
	if state == 0 {
		state = 0x9E3779B97F4A7C15 // any nonzero constant; the golden ratio's
	}
	next := func() uint64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return state
	}
	for i := len(opts) - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		opts[i], opts[j] = opts[j], opts[i]
	}
}
