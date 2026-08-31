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

// PickOptions builds one question.
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
func PickOptions(target Candidate, pool []Candidate, seed uint64) []Option {
	used := map[string]bool{target.Word: true}
	var distractors []Candidate
	rng := newPRNG(seed)

	// THE SEED DRIVES SELECTION, not just the order the options end up in.
	//
	// Both passes below walk this permutation rather than `pool` directly. The
	// first version seeded only the final shuffle, and the effect was severe
	// enough to defeat the form: `pool` is built ONCE per sitting, so scanning
	// it in fixed order made every question take the same first-matching
	// domain / register / general candidate. Measured over a 20-word deck, 17
	// of 20 questions shared one distractor set — the only variation was which
	// slot the answer landed in, so after question one the learner could answer
	// the rest by elimination without knowing a word.
	//
	// A PERMUTATION rather than shuffling the slice: `choiceFor` reuses the
	// sitting's pool across every question, and reordering the caller's slice
	// under it would make each question's selection depend on the ones before.
	order := make([]int, len(pool))
	for i := range order {
		order[i] = i
	}
	for i := len(order) - 1; i > 0; i-- {
		j := rng.intn(i + 1)
		order[i], order[j] = order[j], order[i]
	}

	take := func(c Candidate) {
		used[c.Word] = true
		distractors = append(distractors, c)
	}

	// Pass 1: one per axis, scarcest axis first.
	for _, want := range distractorAxes {
		if len(distractors) >= maxOptions-1 {
			break
		}
		for _, i := range order {
			if c := pool[i]; c.Axis == want && !used[c.Word] {
				take(c)
				break
			}
		}
	}
	// Pass 2: fill what is left.
	for _, i := range order {
		if len(distractors) >= maxOptions-1 {
			break
		}
		if c := pool[i]; !used[c.Word] {
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
	rng.shuffleOptions(opts)
	return opts
}

// prng is a hand-rolled xorshift64.
//
// NOT math/rand, and the reason is stronger than this package's import guard.
// Done-when 3 claims a fixed seed gives the same question — forever, so that a
// question can be reproduced from a log. math/rand's sequence for a given seed
// is a property of the Go runtime, and the top-level generator's behaviour has
// already changed once across versions. A PRNG defined here is pinned by this
// repo's own tests, which is the guarantee the Done-when actually needs.
//
// Not cryptographic and does not need to be: it is choosing which of four
// definitions goes first.
type prng struct{ state uint64 }

func newPRNG(seed uint64) *prng {
	// A zero seed is a FIXED POINT for xorshift — it emits zeros forever and
	// shuffles nothing, silently. Zero is also exactly what a caller passes
	// before wiring a real seed, so this is the case that would ship.
	if seed == 0 {
		seed = 0x9E3779B97F4A7C15 // the golden ratio; any nonzero constant
	}
	return &prng{state: seed}
}

func (p *prng) next() uint64 {
	p.state ^= p.state << 13
	p.state ^= p.state >> 7
	p.state ^= p.state << 17
	return p.state
}

func (p *prng) intn(n int) int { return int(p.next() % uint64(n)) }

func (p *prng) shuffleOptions(opts []Option) {
	for i := len(opts) - 1; i > 0; i-- {
		j := p.intn(i + 1)
		opts[i], opts[j] = opts[j], opts[i]
	}
}

// SampleStrings moves a uniform sample of n entries to the front of ss, in
// place, deterministically under seed.
//
// A PARTIAL Fisher-Yates: only the first n positions are drawn, so sampling 40
// words out of a deck of three thousand costs 40 swaps rather than three
// thousand. Exported because the deck lives in package main and this is the one
// piece of the sampling that has to be seeded the same way the shuffle is —
// ARCH-DRY, rather than main growing a second PRNG that drifts from this one.
func SampleStrings(ss []string, n int, seed uint64) {
	if n > len(ss) {
		n = len(ss)
	}
	p := newPRNG(seed)
	for i := 0; i < n; i++ {
		j := i + p.intn(len(ss)-i)
		ss[i], ss[j] = ss[j], ss[i]
	}
}
