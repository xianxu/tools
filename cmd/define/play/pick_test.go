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
	opts := pickOptions(target, testPool(), 42)
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
	a := pickOptions(target, testPool(), 7)
	for i := 0; i < 20; i++ {
		b := pickOptions(target, testPool(), 7)
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
		other := pickOptions(target, testPool(), s)
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
		for i, o := range pickOptions(target, testPool(), s) {
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
	opts := pickOptions(target, testPool(), 3)
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
		for _, o := range pickOptions(target, pool, s) {
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
			pool = append(pool, Candidate{Word: string(rune('a' + i)), Gloss: "g", Axis: AxisGeneral})
		}
		got := pickOptions(target, pool, 1)
		if len(got) != tc.want {
			t.Errorf("a pool of %d gave %d options, want %d", tc.pool, len(got), tc.want)
		}
	}
}

// Done-when 6, DERIVED from the Axis set rather than from a list here — the same
// shape as #30's registry guards. Adding an axis nobody can select fails this.
func TestEveryAxisIsSelectable(t *testing.T) {
	for a := AxisGeneral; a < numAxes; a++ {
		pool := []Candidate{
			{Word: "x", Gloss: "gx", Axis: a},
			{Word: "y", Gloss: "gy", Axis: a},
			{Word: "z", Gloss: "gz", Axis: a},
		}
		found := false
		for _, o := range pickOptions(target, pool, 1) {
			if !o.Correct && o.Axis == a {
				found = true
			}
		}
		if !found {
			t.Errorf("no option ever carries %v (%q) — an axis nothing can select is dead vocabulary", a, a.String())
		}
		if a.String() == "" {
			t.Errorf("axis %d has no String(), so it could never reach the event log", a)
		}
	}
}
