package store_test

import (
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// StartOfDay is the one answer to "which local day is this instant in".
//
// #15 needed it twice for /history and wrote it inline twice, in two different
// shapes — one building a local midnight, one normalising to UTC before
// dividing. #5 needs the same idea for "has this word's interval elapsed", and a
// third encoding of one concept in one binary is the drift this package's own
// comments keep warning about.
func TestStartOfDay(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("no tzdata: %v", err)
	}

	for _, tc := range []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			"midday becomes local midnight of the same date",
			time.Date(2026, 8, 26, 13, 45, 30, 500, la),
			time.Date(2026, 8, 26, 0, 0, 0, 0, la),
		},
		{
			// The case /history's comment names: a window starting at 00:30
			// silently drops everything done before half past midnight.
			"just after midnight is already that day",
			time.Date(2026, 8, 26, 0, 30, 0, 0, la),
			time.Date(2026, 8, 26, 0, 0, 0, 0, la),
		},
		{
			"just before midnight is still the previous day",
			time.Date(2026, 8, 26, 23, 59, 59, 0, la),
			time.Date(2026, 8, 26, 0, 0, 0, 0, la),
		},
		{
			// A DST day is 23 hours here. Midnight still exists and is still
			// where the day starts; what changes is how long the day runs.
			"the spring-forward day still starts at local midnight",
			time.Date(2026, 3, 8, 14, 0, 0, 0, la),
			time.Date(2026, 3, 8, 0, 0, 0, 0, la),
		},
		{
			"the fall-back day still starts at local midnight",
			time.Date(2026, 11, 1, 14, 0, 0, 0, la),
			time.Date(2026, 11, 1, 0, 0, 0, 0, la),
		},
		{
			"the location is preserved, not normalised to UTC",
			time.Date(2026, 8, 26, 13, 0, 0, 0, time.UTC),
			time.Date(2026, 8, 26, 0, 0, 0, 0, time.UTC),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := store.StartOfDay(tc.in)
			if !got.Equal(tc.want) {
				t.Errorf("StartOfDay(%v) = %v, want %v", tc.in, got, tc.want)
			}
			if got.Location() != tc.in.Location() {
				t.Errorf("location = %v, want %v — the caller compares this against timestamps that kept their own offsets",
					got.Location(), tc.in.Location())
			}
		})
	}
}

func TestStartOfDayIsIdempotent(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("no tzdata: %v", err)
	}
	in := time.Date(2026, 8, 26, 13, 45, 0, 0, la)

	once := store.StartOfDay(in)
	twice := store.StartOfDay(once)

	if !once.Equal(twice) {
		t.Errorf("StartOfDay is not idempotent: %v then %v", once, twice)
	}
}

// Counting days must survive a DST boundary, which is the property that makes
// AddDate the right tool and a Duration the wrong one: a 23-hour day would make
// "one day later" land at 23:00 the same evening.
func TestDaysBetweenAcrossDST(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("no tzdata: %v", err)
	}

	// 2026-03-08 is the 23-hour spring-forward day here.
	before := time.Date(2026, 3, 7, 9, 0, 0, 0, la)
	after := time.Date(2026, 3, 9, 9, 0, 0, 0, la)

	if got := store.DaysBetween(before, after); got != 2 {
		t.Errorf("DaysBetween across spring-forward = %d, want 2 calendar days", got)
	}
	if got := store.DaysBetween(after, before); got != -2 {
		t.Errorf("DaysBetween backwards = %d, want -2", got)
	}
	if got := store.DaysBetween(before, before); got != 0 {
		t.Errorf("DaysBetween(t, t) = %d, want 0", got)
	}
}
