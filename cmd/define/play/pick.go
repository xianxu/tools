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
	// Word is the DECK KEY — what the learner typed and what the event records.
	Word string
	// Source identifies the dictionary ENTRY this gloss came from, and it is not
	// the same thing as Word.
	//
	// This distinction is the root of three separate defects found one round
	// apart, each looking like a new bug and each the same one: a deck holds
	// KEYS, a dictionary holds ENTRIES, and the mapping is many-to-one.
	// `jalapeño` and `jalapeno` are two keys (store.Key folds case and
	// whitespace, not diacritics) that the dictionary answers with one entry.
	// Dedup on Word alone let that entry supply two options; dedup on Word and
	// Gloss still let it supply two options carrying DIFFERENT SENSES of itself,
	// both of which genuinely define the prompted word, one arbitrarily marked
	// wrong.
	//
	// Keying on the entry is what actually closes it, because the entry is the
	// unit of meaning. Empty means "no entry identity available", and selection
	// then falls back to Word — a caller that forgets to set it is no worse off
	// than before, which keeps this from being a new way to break the form.
	Source string
	Gloss  string
	Axis   Axis
}

// sourceOf is a candidate's entry identity, falling back to its deck key.
//
// The fallback matters: a Candidate built without a Source (a test, or a future
// caller) must still be deduped rather than colliding with every other
// Source-less candidate under the empty string.
func sourceOf(c Candidate) string {
	if c.Source != "" {
		return c.Source
	}
	return c.Word
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
	// DEDUP ON GLOSS TOO, not just on word, and seeded with the ANSWER's gloss.
	//
	// Two deck words can resolve to one dictionary entry: `jalapeño` and
	// `jalapeno` are separate deck keys (store.Key folds case and whitespace but
	// not diacritics) and the dictionary answers both with the same entry, which
	// is documented, production, accent-insensitive behaviour. Keying only on
	// Word then puts BYTE-IDENTICAL glosses in one option set, one marked
	// Correct and one not — so a learner who reads both and picks the other one
	// is recorded as a miss, given a fabricated axis, and has the word demoted.
	//
	// `crossReferenced` cannot catch it: neither headword appears in the shared
	// gloss. Nothing upstream can, either, because both words are real deck
	// entries whose entry genuinely defines them. The set is the only place that
	// can see two options saying the same thing.
	usedGloss := map[string]bool{target.Gloss: true}
	usedSource := map[string]bool{sourceOf(target): true}
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
	shuffle(rng, order)

	take := func(c Candidate) {
		used[c.Word] = true
		usedGloss[c.Gloss] = true
		usedSource[sourceOf(c)] = true
		distractors = append(distractors, c)
	}
	// THREE KEYS, and each one closes a case the others cannot see. Word stops
	// the obvious repeat; Gloss stops two entries that happen to print the same
	// text; Source stops ONE entry supplying two of its own senses under two
	// deck keys, which is the case that reads as two different right answers.
	free := func(c Candidate) bool {
		return !used[c.Word] && !usedGloss[c.Gloss] && !usedSource[sourceOf(c)]
	}

	// Pass 1: one per axis, scarcest axis first.
	for _, want := range distractorAxes {
		if len(distractors) >= maxOptions-1 {
			break
		}
		for _, i := range order {
			if c := pool[i]; c.Axis == want && free(c) {
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
		if c := pool[i]; free(c) {
			take(c)
		}
	}

	// Below one distractor there is nothing to choose BETWEEN, so this is not a
	// question. Returning nothing rather than a one-option question is what lets
	// the caller triage the word on a board (D9, #42) instead of showing a learner
	// a multiple choice with a single answer.
	if len(distractors) == 0 {
		return nil
	}

	opts := []Option{{Gloss: target.Gloss, Word: target.Word, Correct: true}}
	for _, c := range distractors {
		opts = append(opts, Option{Gloss: c.Gloss, Word: c.Word, Axis: c.Axis})
	}
	shuffle(rng, opts)
	return opts
}

// prng is a hand-rolled xorshift64.
//
// NOT math/rand. The binding constraint is this package's empty import
// allowlist, but the sequence being OURS matters on its own: math/rand's output
// for a given seed is a property of the Go runtime, and the top-level
// generator's behaviour has already changed once across versions.
//
// What that buys, stated accurately: the same deck on the same day yields the
// same sitting, so a mid-sitting restart re-asks rather than reshuffles, and
// tests are stable across runs and machines. It does NOT make a recorded
// question re-derivable from the event log — the option set depends on the pool,
// which is the deck at that moment, and the log records none of that.
// TestPRNGSequenceIsPinned is what makes "ours" true rather than asserted.
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

// shuffle is Fisher-Yates over any slice.
//
// Generic because it was written twice — once for the []int permutation that
// drives selection and once for the []Option result — with identical bodies
// (ARCH-DRY). Generics need no import, so the package's empty allowlist was
// never the reason they were separate.
func shuffle[T any](p *prng, xs []T) {
	for i := len(xs) - 1; i > 0; i-- {
		j := p.intn(i + 1)
		xs[i], xs[j] = xs[j], xs[i]
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

// ShuffleInts is the sitting's shuffle, exported for the authoring side.
//
// #10 selects distractors offline and wants the same seeded permutation this
// package uses at review time. It first hand-rolled a copy with a DIFFERENT
// seeding step — "the same algorithm" producing different sequences, which is
// the worst kind of duplicate because the comment claiming kinship is what a
// reader trusts. An int-slice shuffle needs none of the store's vocabulary, so
// the D5a rule ("play imports nothing") is untouched by exporting it, exactly as
// SampleStrings already is.
func ShuffleInts(seed uint64, xs []int) { shuffle(newPRNG(seed), xs) }
