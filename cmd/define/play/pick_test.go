package play

import "testing"

// A pool with all three axes available, several times over.
func testPool() []Candidate {
	return []Candidate{
		{Word: "larceny", Gloss: "the offence of taking property", Axis: AxisDomain},
		{Word: "record", Gloss: "an official report of proceedings", Axis: AxisDomain},
		{Word: "manservant", Gloss: "a manservant or valet", Axis: AxisRegister},
		{Word: "diarrhea", Gloss: "an unpleasant condition", Axis: AxisRegister},
		{Word: "mesa", Gloss: "an isolated flat-topped hill", Axis: AxisGeneral},
		{Word: "quokka", Gloss: "a small short-tailed wallaby", Axis: AxisGeneral},
		{Word: "parrot", Gloss: "a vividly coloured bird", Axis: AxisGeneral},
	}
}

var target = Candidate{Word: "sycophantic", Gloss: "behaving in an obsequious way"}

// Done-when 1. Exactly one option is the answer, and it is the target's gloss.
func TestPickOptionsHasOneAnswer(t *testing.T) {
	opts := PickOptions(target, testPool(), 42)
	if len(opts) != 4 {
		t.Fatalf("%d options, want 4", len(opts))
	}
	n := 0
	for _, o := range opts {
		if o.Correct {
			n++
			if o.Gloss != target.Gloss {
				t.Errorf("the correct option is %q, want the target's gloss", o.Gloss)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d options marked correct, want exactly 1", n)
	}
}

// Done-when 3. Same seed, same question — order included, since the position of
// the answer is part of what makes a question reproducible.
func TestPickOptionsIsDeterministic(t *testing.T) {
	a := PickOptions(target, testPool(), 7)
	for i := 0; i < 20; i++ {
		b := PickOptions(target, testPool(), 7)
		if len(a) != len(b) {
			t.Fatalf("run %d produced %d options, first run produced %d", i, len(b), len(a))
		}
		for j := range a {
			if a[j] != b[j] {
				t.Fatalf("run %d differs at option %d: %+v vs %+v", i, j, b[j], a[j])
			}
		}
	}
	// And a different seed must actually move something, or "seeded" is a lie.
	same := true
	for s := uint64(1); s < 30 && same; s++ {
		other := PickOptions(target, testPool(), s)
		for j := range a {
			if a[j] != other[j] {
				same = false
			}
		}
	}
	if same {
		t.Error("30 seeds produced the identical question — the seed is not reaching the shuffle")
	}
}

// The answer must not sit in the same slot every time, or the learner learns the
// position instead of the word.
func TestPickOptionsMovesTheAnswerAround(t *testing.T) {
	seen := map[int]bool{}
	for s := uint64(0); s < 200; s++ {
		for i, o := range PickOptions(target, testPool(), s) {
			if o.Correct {
				seen[i] = true
			}
		}
	}
	if len(seen) < 4 {
		t.Errorf("the answer only ever appeared in slots %v, want all four", seen)
	}
}

// D2a: fill domain, then register, then general.
func TestPickOptionsVariesTheAxes(t *testing.T) {
	opts := PickOptions(target, testPool(), 3)
	got := map[Axis]int{}
	for _, o := range opts {
		if !o.Correct {
			got[o.Axis]++
		}
	}
	for _, a := range distractorAxes {
		if got[a] == 0 {
			t.Errorf("no %v distractor, though the pool has one — axes: %v", a, got)
		}
	}
}

// A word may supply several axes, so it can appear in the pool more than once —
// but never twice in one question, which would be two options with one source.
func TestPickOptionsNeverRepeatsAWord(t *testing.T) {
	pool := []Candidate{
		{Word: "record", Gloss: "an official report", Axis: AxisDomain},
		{Word: "record", Gloss: "a thin plastic disk", Axis: AxisGeneral},
		{Word: "record", Gloss: "the best performance", Axis: AxisRegister},
		{Word: "mesa", Gloss: "a flat-topped hill", Axis: AxisGeneral},
	}
	for s := uint64(0); s < 50; s++ {
		seen := map[string]bool{}
		for _, o := range PickOptions(target, pool, s) {
			if o.Word != "" && seen[o.Word] {
				t.Fatalf("seed %d used %q twice", s, o.Word)
			}
			seen[o.Word] = true
		}
	}
}

// D9: a young deck is a NORMAL state. Below two options it is not a question,
// and the caller falls back to form 2.1.
func TestPickOptionsWithATinyDeck(t *testing.T) {
	for _, tc := range []struct{ pool, want int }{
		{0, 0}, // not a question at all
		{1, 2},
		{2, 3},
		{3, 4},
		{9, 4}, // never more than four
	} {
		var pool []Candidate
		for i := 0; i < tc.pool; i++ {
			// DISTINCT glosses. The first version gave every candidate "g",
			// which the gloss-dedup rule correctly collapses to one option —
			// a fixture that could not represent a real deck, where two words
			// sharing a definition is the anomaly this form now guards against.
			pool = append(pool, Candidate{
				Word:  string(rune('a' + i)),
				Gloss: "definition number " + string(rune('a'+i)),
				Axis:  AxisGeneral,
			})
		}
		got := PickOptions(target, pool, 1)
		if len(got) != tc.want {
			t.Errorf("a pool of %d gave %d options, want %d", tc.pool, len(got), tc.want)
		}
	}
}

// Done-when 6, DERIVED from the Axis set rather than from a list here — the same
// shape as #30's registry guards. Adding an axis nothing can select fails this.
//
// THE MEMBERSHIP ASSERTION IS THE LOAD-BEARING ONE, and it is here because the
// behavioural version alone was vacuous. The first draft built a pool entirely
// of the axis under test and asked whether an option carried it; PickOptions'
// second pass fills any remaining slot from ANY candidate, so the answer was yes
// even for an axis absent from distractorAxes entirely. Reshaping the pool does
// not fix it either: pass 2 can always reach the candidate, by design.
//
// So the property is stated directly. distractorAxes is a hand-maintained
// restatement of the Axis set, and this is what makes it derived: every axis
// between AxisGeneral and the numAxes sentinel must appear in the fill order,
// must have a String() that can reach the event log, and must be reachable in a
// real option set.
func TestEveryAxisIsSelectable(t *testing.T) {
	for a := AxisGeneral; a < numAxes; a++ {
		inFillOrder := false
		for _, d := range distractorAxes {
			if d == a {
				inFillOrder = true
			}
		}
		if !inFillOrder {
			t.Errorf("axis %d (%q) is not in distractorAxes, so pass 1 never seeks it — "+
				"it could only ever appear by accident when pass 2 tops up, which is not selection",
				a, a.String())
		}
		if a.String() == "" {
			t.Errorf("axis %d has no String(), so it could never reach the event log", a)
		}
		// And it must actually come out of a real call, so the membership
		// assertion above cannot pass on an axis the selector mishandles.
		pool := []Candidate{
			{Word: "x", Gloss: "gx", Axis: a},
			{Word: "y", Gloss: "gy", Axis: a},
			{Word: "z", Gloss: "gz", Axis: a},
		}
		found := false
		for seed := uint64(1); seed < 40 && !found; seed++ {
			for _, o := range PickOptions(target, pool, seed) {
				if !o.Correct && o.Axis == a {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("no option ever carries %v (%q)", a, a.String())
		}
	}
}

// A SITTING must not be one question repeated. This is the pin for the defect
// the close review measured: the seed used to reach only the final shuffle, so
// every target drew the same first-matching candidate out of the pool the
// sitting builds once — 17 of 20 questions with an identical option set, and a
// learner who could answer the rest by elimination after question one.
func TestPickOptionsVariesTheDistractorsAcrossASitting(t *testing.T) {
	var pool []Candidate
	axes := []Axis{AxisDomain, AxisRegister, AxisGeneral}
	for i := 0; i < 18; i++ {
		pool = append(pool, Candidate{
			Word:  string(rune('a' + i)),
			Gloss: "gloss " + string(rune('a'+i)),
			Axis:  axes[i%3],
		})
	}

	sets := map[string]int{}
	for i := 0; i < 20; i++ {
		// One target per question, as a sitting does, against the SAME pool.
		tgt := Candidate{Word: "target" + string(rune('a'+i)), Gloss: "the answer"}
		var key string
		for _, o := range PickOptions(tgt, pool, seedOf(i)) {
			if !o.Correct {
				key += o.Word + ","
			}
		}
		sets[key]++
	}
	// 20 questions over an 18-candidate pool: with selection seeded, near-total
	// variety. Eight is far below what a correct implementation gives and far
	// above the ONE the defect produced, so this cannot pass on the bug and
	// cannot flake on a shuffle.
	if len(sets) < 8 {
		t.Errorf("20 questions produced only %d distinct distractor sets %v — "+
			"the seed is not reaching SELECTION, so a sitting is one question repeated", len(sets), sets)
	}
}

// seedOf is a per-question seed, as the loop derives one from word + day.
func seedOf(i int) uint64 { return uint64(i)*2654435761 + 7 }

// The PRNG and the option shuffle are GOLDEN, because the argument for
// hand-rolling them is that this repo pins them.
//
// D5a rejects math/rand, and nothing pinned the sequence: changing a shift
// constant left the whole suite green. Comparing two runs inside one binary
// proves only that the code is deterministic, which math/rand also is; what had
// to be pinned is the SPECIFIC sequence.
//
// What that buys, stated accurately — an earlier version of this comment said
// "so a question can be reproduced from a log", which is false and was corrected
// across the whole diff: the option set also depends on the pool, which is the
// deck at that moment, and the log records none of it. The true claim is that
// the same deck on the same day gives the same sitting, so a mid-sitting restart
// re-asks rather than reshuffles, and tests are stable across machines. Changing
// these literals is a deliberate act that changes what today's sitting asks.
func TestPRNGSequenceIsPinned(t *testing.T) {
	p := newPRNG(7)
	want := []uint64{7575888327, 8070950887952051652, 13931920357059763743}
	for i, w := range want {
		if got := p.next(); got != w {
			t.Errorf("next() #%d = %d, want %d — the xorshift constants moved, so every "+
				"question ever recorded now reproduces differently", i, got, w)
		}
	}
	// And a zero seed must not be a fixed point emitting zeros forever.
	z := newPRNG(0)
	if a, b := z.next(), z.next(); a == 0 || a == b {
		t.Errorf("seed 0 degenerated: %d, %d", a, b)
	}
}

// NO TWO OPTIONS MAY SAY THE SAME THING, including the answer.
//
// Two deck words can resolve to one dictionary entry — `jalapeño` and `jalapeno`
// are separate deck keys that the dictionary answers identically — so a set
// deduped on Word alone offers byte-identical glosses, one Correct and one not.
// The learner who picks the other one is recorded as a miss with a fabricated
// axis, and the word is demoted for answering correctly.
func TestPickOptionsNeverRepeatsAGloss(t *testing.T) {
	shared := "a very hot green chili pepper, used especially in Mexican cooking"
	tgt := Candidate{Word: "jalapeno", Gloss: shared}
	pool := []Candidate{
		{Word: "jalapeño", Gloss: shared, Axis: AxisGeneral}, // the same entry
		{Word: "jalapeños", Gloss: shared, Axis: AxisDomain}, // and again
		{Word: "mesa", Gloss: "a flat-topped hill", Axis: AxisGeneral},
		{Word: "quokka", Gloss: "a small wallaby", Axis: AxisRegister},
	}
	for s := uint64(0); s < 60; s++ {
		opts := PickOptions(tgt, pool, s)
		seen := map[string]bool{}
		for _, o := range opts {
			if seen[o.Gloss] {
				t.Fatalf("seed %d: two options share the gloss %q — one is marked Correct "+
					"and the other is not, so reading them both and picking the second "+
					"records a miss for a right answer: %+v", s, o.Gloss, opts)
			}
			seen[o.Gloss] = true
		}
	}
}
