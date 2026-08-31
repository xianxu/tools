// Package schedule decides what is worth the learner's attention today.
//
// Leitner boxes on a GEOMETRIC ladder, chosen over SM-2 deliberately: for a
// personal tool "why is this due?" must be answerable in one sentence, and an
// ease factor cannot be. The sentence here is "each correct recall multiplies
// the wait by 1.6".
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

// The ladder's ratio, as an exact rational: 1.6 = 8/5.
//
// A RATIONAL rather than a float, and the reason is the one `#7` established for
// its PRNG and hash. `math.Pow(1.6, b)` is not guaranteed bit-identical across
// architectures, so a ladder built on it could differ by platform — and a
// schedule that differs by platform is one this repo cannot pin in a table test
// and a learner cannot reason about. `8^b / 5^b` in int64 is exact everywhere,
// needs no import, and leaves this package's allowlist untouched.
const (
	ratioNum = 8
	ratioDen = 5
)

// ladderLimit is where the ARITHMETIC stops, and it is not a pedagogical
// ceiling.
//
// `8^20` is 1.15e18, comfortably inside int64; `8^21` overflows. Box 20 is a
// 12,089-day interval and is reached only after **20,135 days of correct
// answers — 55 years** (pinned by TestTheClampIsUnreachable). So no learner can
// arrive here, and the ladder is unbounded in every sense that matters to one.
//
// NAMED `ladderLimit` AND NOT `maxBox`, deliberately: Progress carries a MaxBox
// field, and `p.Box < maxBox` and `p.Box < p.MaxBox` are both valid Go with
// opposite meanings — the first would grant every word a permanent express lane,
// silently and forever. Two concepts one letter apart is a defect waiting for a
// tired reader.
const ladderLimit = 20

// IntervalDays is how many local calendar days a word in this box waits.
//
// `floor(1.6^box)`, computed as `8^box / 5^box`. The first two rungs are BOTH
// one day — `floor(1.6^0)` and `floor(1.6^1)` are both 1 — and that duplicate is
// the most valuable rung in the ladder rather than an artifact: a new word is
// seen on day 1 and again on day 2, which is when forgetting is steepest.
//
//	box    0  1  2  3  4   5   6   7   8   9   10   11   12
//	wait   1  1  2  4  6  10  16  26  42  68  109  175  281
//	day@   0  1  2  4  8  14  24  40  66 108  176  285  460
//
// Clamps rather than panicking on an out-of-range box: Fold derives boxes from
// an append-only log, and a log written by a future version with a different
// ratio must degrade to the longest interval we can express rather than crash a
// review session.
func IntervalDays(box int) int {
	box = clampBox(box)
	num, den := int64(1), int64(1)
	for i := 0; i < box; i++ {
		num *= ratioNum
		den *= ratioDen
	}
	if d := num / den; d > 1 {
		return int(d)
	}
	// Boxes 0 and 1 both land here: 1/1 and 8/5 both floor to 1.
	return 1
}

func clampBox(box int) int {
	if box < 0 {
		return 0
	}
	if box > ladderLimit {
		return ladderLimit
	}
	return box
}
