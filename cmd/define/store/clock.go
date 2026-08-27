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
// **B'S LOCATION DEFINES THE CALENDAR.** a is converted into it first, which is
// the same choice #15's relativeDay already makes with `at = at.In(now.Location())`.
// The caller's question is always "how many days have passed for the LEARNER",
// and the learner is wherever `now` is.
//
// That normalisation is not decoration, and the first version of this function
// omitted it. Store timestamps keep their own offsets — yaml.v3 parses them into
// FIXED zones — while `now` comes from SystemClock in time.Local, so mixed
// locations are the NORMAL case, not an edge one. Without the conversion, two
// instants on the same local day counted as 1 apart, and `Due` reported a word
// due on the day it was reviewed.
//
// Then the two local dates are projected onto a UTC day index and subtracted.
// That is what #15's dayIndex closure did, and its comment said why: it "takes
// DST out of the arithmetic instead of compensating for it". Consolidating the
// two encodings, I kept the shape that read more cleanly and dropped the one
// whose comment explained itself — and the clean-looking one stepped the
// calendar in a's zone while comparing instants against b's, which is precisely
// the bug. A 23- or 25-hour day cannot perturb a subtraction of two UTC
// midnights.
func DaysBetween(a, b time.Time) int {
	a = a.In(b.Location())
	return int(dayIndex(b) - dayIndex(a))
}

// dayIndex projects a LOCAL date onto a UTC day number, so day arithmetic is
// subtraction rather than a walk that DST can perturb.
func dayIndex(t time.Time) int64 {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400
}
