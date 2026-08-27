package schedule

import "testing"

// EVERY interval, not just the ends.
//
// The first version asserted box 0 and LastBox only, on the reasoning that rows
// should be decisions rather than data. That was wrong here: the ladder IS the
// model — it is what the Spec names and what a learner experiences — so a typo
// in the middle of it would change the product and redden nothing.
func TestBoxIntervals(t *testing.T) {
	want := []int{1, 3, 7, 14, 30, 90}

	if got := LastBox; got != len(want)-1 {
		t.Fatalf("LastBox = %d, want %d — the ladder changed and this test's expectation did not", got, len(want)-1)
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
	if got := IntervalDays(LastBox + 5); got != IntervalDays(LastBox) {
		t.Errorf("a box past the end gave %d, want the last interval", got)
	}
}

// A non-increasing table would make a "promotion" shorten the interval — the
// whole model inverted, silently.
func TestIntervalsAreStrictlyIncreasing(t *testing.T) {
	for b := 1; b <= LastBox; b++ {
		if IntervalDays(b) <= IntervalDays(b-1) {
			t.Errorf("interval for box %d (%d) is not longer than box %d (%d)",
				b, IntervalDays(b), b-1, IntervalDays(b-1))
		}
	}
}

func TestBoxClamps(t *testing.T) {
	if got := clampBox(LastBox + 1); got != LastBox {
		t.Errorf("promotion past the last box gave %d, want %d", got, LastBox)
	}
	if got := clampBox(-1); got != 0 {
		t.Errorf("demotion below zero gave %d, want 0", got)
	}
}
