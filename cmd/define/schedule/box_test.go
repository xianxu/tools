package schedule

import "testing"

// EVERY interval, not just the ends.
//
// The first version asserted box 0 and the last box only, on the reasoning that
// rows should be decisions rather than data. That was wrong here: the ladder IS
// the model — it is what the Spec names and what a learner experiences — so a
// typo in the middle of it would change the product and redden nothing.
//
// These are literals rather than a recomputation of `floor(1.6^box)`, because a
// test that recomputes the implementation's formula asserts only that Go's
// arithmetic is deterministic. The values were derived once, by hand, and are
// the contract.
func TestBoxIntervals(t *testing.T) {
	want := []int{1, 1, 2, 4, 6, 10, 16, 26, 42, 68, 109, 175, 281, 450, 720, 1152, 1844, 2951, 4722, 7555, 12089}

	if got := ladderLimit; got != len(want)-1 {
		t.Fatalf("ladderLimit = %d, want %d — the ladder changed and this test's expectation did not", got, len(want)-1)
	}
	for box, wantDays := range want {
		if got := IntervalDays(box); got != wantDays {
			t.Errorf("box %d interval = %d days, want %d", box, got, wantDays)
		}
	}
	// Out of range clamps rather than panicking: Fold reads boxes off an
	// append-only log that a future version may have written differently.
	if got := IntervalDays(-3); got != IntervalDays(0) {
		t.Errorf("a negative box gave %d, want box 0's interval", got)
	}
	if got := IntervalDays(ladderLimit + 5); got != IntervalDays(ladderLimit) {
		t.Errorf("a box past the end gave %d, want the last interval", got)
	}
}

// BOXES 0 AND 1 ARE BOTH ONE DAY, and that is the most valuable rung in the
// ladder rather than a rounding artifact.
//
// `floor(1.6^0)` and `floor(1.6^1)` are both 1, so a new word is seen on day 1
// and again on day 2 — which is when the forgetting curve is steepest. The
// obvious "fix", indexing from `1.6^(box+1)`, removes the duplicate and with it
// the entire acquisition density this ladder exists to provide.
//
// Its own test because it looks like a bug to anyone reading the table cold.
func TestTheFirstTwoRungsAreBothOneDay(t *testing.T) {
	if IntervalDays(0) != 1 || IntervalDays(1) != 1 {
		t.Errorf("boxes 0 and 1 are %d and %d days, want 1 and 1 — a new word must be "+
			"seen on day 1 AND day 2", IntervalDays(0), IntervalDays(1))
	}
	if IntervalDays(2) != 2 {
		t.Errorf("box 2 is %d days, want 2 — the ladder is indexed from 1.6^box, not 1.6^(box+1)", IntervalDays(2))
	}
}

// NON-decreasing, not strictly increasing: boxes 0 and 1 tie deliberately.
// A DECREASING step would make a promotion shorten the interval — the whole
// model inverted, silently.
func TestIntervalsNeverShrink(t *testing.T) {
	for b := 1; b <= ladderLimit; b++ {
		if IntervalDays(b) < IntervalDays(b-1) {
			t.Errorf("interval for box %d (%d) is shorter than box %d (%d) — a promotion "+
				"must never bring a word back sooner", b, IntervalDays(b), b-1, IntervalDays(b-1))
		}
	}
	// And it must actually GROW overall, or a "ladder" of equal rungs would pass
	// the check above.
	if IntervalDays(ladderLimit) <= IntervalDays(2) {
		t.Error("the ladder does not grow")
	}
}

// The clamp is ARITHMETIC, not pedagogy — no learner can reach it.
//
// `8^21` overflows int64, so the ladder stops at box 20. That is only acceptable
// because reaching box 20 takes 20,135 days of correct answers: 55 years. A
// future ratio change that made the top rung reachable would turn an overflow
// guard into a silent pedagogical ceiling, which is the thing this issue exists
// to remove.
func TestTheClampIsUnreachable(t *testing.T) {
	const aLifetime = 20000 // days; ~55 years
	cumulative := 0
	for b := 0; b < ladderLimit; b++ {
		cumulative += IntervalDays(b)
	}
	if cumulative < aLifetime {
		t.Errorf("reaching box %d takes %d days (%.1f years) — the clamp is now a "+
			"pedagogical ceiling rather than an overflow guard, which is what this "+
			"ladder is designed not to have", ladderLimit, cumulative, float64(cumulative)/365)
	}
}

func TestBoxClamps(t *testing.T) {
	if got := clampBox(ladderLimit + 1); got != ladderLimit {
		t.Errorf("promotion past the last box gave %d, want %d", got, ladderLimit)
	}
	if got := clampBox(-1); got != 0 {
		t.Errorf("demotion below zero gave %d, want 0", got)
	}
}
