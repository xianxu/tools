package store

import "time"

// Clock is injected everywhere a time is stamped.
//
// Not a convenience: every consumer of this store is date-driven — "which words
// are due today" is the whole of #5 — and a wall clock makes that untestable.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// SystemClock is the real one. Tests pass a fake.
func SystemClock() Clock { return systemClock{} }

// FixedClock returns a Clock frozen at t, for tests and for reproducing a day.
func FixedClock(t time.Time) Clock { return fixed{t} }

type fixed struct{ t time.Time }

func (f fixed) Now() time.Time { return f.t }

// StartOfDay is local midnight of t's date, in t's own location.
//
// The ONE answer to "which local day is this instant in". #15 needed it twice
// for /history and wrote it inline twice in two different shapes; #5 needs the
// same idea for "has this word's interval elapsed", and a third encoding of one
// concept in one binary is exactly the drift this package keeps warning about
// elsewhere.
//
// Two things make it less obvious than it looks, both learned in #15:
//
//   - It is a local-CALENDAR question, so the boundary is local midnight rather
//     than "now minus N×24h". A learner who reviews at 9am Monday and sits down
//     at 8am Tuesday must find the word due; under 24h arithmetic they would not.
//   - The location is PRESERVED. Callers compare the result against timestamps
//     that kept their own offsets, never against day-file names, which are UTC
//     and belong to the wrong calendar for this question.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// DaysBetween counts local calendar days from a to b, signed.
//
// AddDate-shaped rather than a Duration divided by 24h, and that is load-bearing:
// a DST day is 23 or 25 hours, so dividing would report a day that did not pass
// or miss one that did. Counting by stepping the calendar is what "a day later"
// means to a reader, which is the standard #5's Spec sets for every scheduling
// answer.
func DaysBetween(a, b time.Time) int {
	a, b = StartOfDay(a), StartOfDay(b)
	if a.Equal(b) {
		return 0
	}
	sign, from, to := 1, a, b
	if b.Before(a) {
		sign, from, to = -1, b, a
	}
	n := 0
	for from.Before(to) {
		from = from.AddDate(0, 0, 1)
		n++
	}
	return sign * n
}
