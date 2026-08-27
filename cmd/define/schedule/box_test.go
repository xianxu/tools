package schedule

import "testing"

// Rows are the DECISIONS the box model makes, not one row per interval.
func TestBoxIntervals(t *testing.T) {
	if got := IntervalDays(0); got != 1 {
		t.Errorf("box 0 interval = %d days, want 1 — a word just answered should come back tomorrow", got)
	}
	if got := IntervalDays(LastBox); got != 90 {
		t.Errorf("last box interval = %d days, want 90", got)
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
