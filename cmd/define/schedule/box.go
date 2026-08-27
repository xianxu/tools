// Package schedule decides what is worth the learner's attention today.
//
// Leitner boxes with fixed intervals, chosen over SM-2 deliberately: for a
// personal tool "why is this due?" must be answerable in one sentence, and an
// ease factor cannot be.
//
// ENTIRELY PURE: no IO, no clock of its own, no state. It imports `store` and
// pure standard-library packages only, and every instant arrives as a parameter.
//
// That is not a stylistic preference: every question here is date-driven, and a
// package that could reach a wall clock would be untestable at exactly the point
// where correctness lives. THREE guards enforce it, because a comment cannot —
// an import allowlist, a check that no wall-clock reader (`time.Now`,
// `time.Since`, and the rest) appears in any non-test file, and a store-SYMBOL
// allowlist, since allowlisting `store` would otherwise grant the disk with it.
//
// `play` takes the first two; it names no store symbol, so the third would have
// nothing to check. The count differs per package, which is why it is stated
// per package rather than as a fact about the guards.
//
// Schedule state is DERIVED from the event log, never stored beside it. store's
// own event.go states the rule: the log is "deliberately the ONLY record of
// activity ... storing counters alongside would create a second source of truth
// that drifts". A Box field on store.Word would be exactly that, and it would go
// wrong invisibly — a hand-edited deck file disagreeing with the events that
// produced it.
package schedule

// intervalDays is the Leitner ladder. Fixed, and short enough to read.
//
// One slice, so a per-learner variant later replaces the table rather than the
// logic around it. Nothing here measures whether these suit this learner; #8's
// stats are what would eventually say.
// An ARRAY, not a slice, so len() is a constant expression and LastBox can be a
// const. As a `var` it was exported, mutable, and depended on by every clamp and
// by Mastered — any package could have assigned to it and silently rewritten the
// schedule for the whole process.
var intervalDays = [...]int{1, 3, 7, 14, 30, 90}

// LastBox is the final rung. Reaching it takes LastBox consecutive correct
// answers from box 0.
const LastBox = len(intervalDays) - 1

// IntervalDays is how many local calendar days a word in this box waits.
//
// Clamps rather than panicking on an out-of-range box: Fold derives boxes from
// an append-only log, and a log written by a future version with a longer ladder
// must degrade to "the longest interval we know" rather than crash a review
// session.
func IntervalDays(box int) int {
	return intervalDays[clampBox(box)]
}

func clampBox(box int) int {
	if box < 0 {
		return 0
	}
	if box > LastBox {
		return LastBox
	}
	return box
}
