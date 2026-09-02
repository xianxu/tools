package schedule

import (
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Grade is HOW an answer was given, not merely whether it was right.
//
// Three values rather than a bool, because the ladder now promotes by an amount
// that depends on the manner of the answer, and a bool cannot carry three
// states.
//
// `#40` was expected to extend this seam with a fourth, `GradeUnsure`, and it
// did NOT: the operator deleted the third mark — *"I guess unsure means no"* —
// so form 2.5's board grades `Correct` and `Wrong` like everything else. The
// seam is still the right one to extend; nothing has needed to yet.
type Grade int

const (
	// GradeWrong is the ZERO value deliberately. A caller that forgets to set a
	// grade demotes rather than promotes, which is the safe direction: a word
	// wrongly demoted comes back sooner, while a word wrongly promoted silently
	// disappears for months.
	GradeWrong Grade = iota
	// GradeCorrect is a right answer that needed help — the learner revealed
	// first, or the form only asked them to rate themselves.
	GradeCorrect
	// GradeUnaided is a right answer given COLD, and it is an observation
	// rather than a claim: the form compared the learner's answer to one it
	// already knew, and no reveal preceded it. Form 2.1 can never produce this
	// (its `y` is self-report), which `play`'s SelfRated interface enforces.
	GradeUnaided
)

// MasteredBox is where a word stops being highlighted.
//
// Box 9 is reached on day 108 after nine correct recalls, the last of which came
// after a 42-day gap — which is the part that matters. A bar that could be
// cleared without ever surviving a long gap would certify words the learner last
// saw three weeks ago.
const MasteredBox = 9

// Progress is one word's schedule state, DERIVED from the event log.
//
// No review count: #8 can fold the log for that directly, and a field with no
// reader is a field that goes stale. `Streak` was deleted for exactly that
// reason when `Mastered` stopped reading it.
type Progress struct {
	Box int
	// MaxBox is the highest box this word has ever reached, and it is the
	// express lane back after a lapse.
	//
	// Storage strength survives when retrieval strength does not, which is why
	// relearning is faster than learning — Ebbinghaus called it savings. So a
	// word below its own high-water mark climbs two rungs per correct answer
	// instead of one. It ERODES by one on every lapse, so a word that keeps
	// failing gradually loses the lane and is eventually relearned properly
	// rather than being waved back up forever.
	MaxBox       int
	LastReviewed time.Time
}

// Answer is the transition: one review, one new state.
//
// Correct climbs one rung, or TWO while below MaxBox. Unaided climbs two. Wrong
// HALVES the box — one sentence, and it scales: box 12 (281 days) falls to box 6
// (16 days), a real relearning interval, while box 2 falls to box 1, barely a
// nudge. A fixed step cannot be both.
//
// THE HALVING AND THE EXPRESS LANE ARE A PAIR. A gentle single-step demotion
// needs no lane because it never travels far; halving without one would make a
// single slip cost most of a year. Removing either alone is worse than removing
// both, which TestRecoveryFromALapse is there to catch.
func Answer(p Progress, g Grade, at time.Time) Progress {
	switch g {
	case GradeWrong:
		box := clampBox(p.Box / 2)
		return Progress{Box: box, MaxBox: max(box, p.MaxBox-1), LastReviewed: at}
	case GradeUnaided:
		return promote(p, 2, at)
	default:
		// Two rungs while below the high-water mark, one at or above it.
		step := 1
		if p.Box < p.MaxBox {
			step = 2
		}
		return promote(p, step, at)
	}
}

// promote caps the step at two. An unaided answer BELOW MaxBox does not climb
// four: the two reasons to move faster are the same reason — this word is
// easier than a new one — and stacking them would let a word skip most of the
// ladder on a single lucky sitting.
func promote(p Progress, step int, at time.Time) Progress {
	box := clampBox(p.Box + step)
	return Progress{Box: box, MaxBox: max(box, p.MaxBox), LastReviewed: at}
}

// Due reports whether this word's interval has elapsed.
//
// LOCAL CALENDAR days, via store.StartOfDay, never N×24h. A learner who reviews
// at 9am Monday and sits down at 8am Tuesday must find a box-0 word due; under
// hour arithmetic they would not, and the tool would feel broken in the most
// ordinary use there is.
//
// The zero Progress is due, which is how a word you looked up but never reviewed
// reaches the queue at all.
func Due(p Progress, now time.Time) bool {
	if p.LastReviewed.IsZero() {
		return true
	}
	return store.DaysBetween(p.LastReviewed, now) >= IntervalDays(p.Box)
}

// Mastered is whether a word has stopped needing attention on screen.
//
// A LABEL, never a removal, and queue.go states the reason: "a word never
// offered can never be answered wrong, so it could never be demoted, and the
// learner's mastered count could only ever grow while their actual recall
// decayed." A mastered word keeps being reviewed — at box 9 that is three times
// a year, which rounds to nothing — and keeps serving as a distractor.
//
// Exported and defined once because two consumers need the same answer: #6's
// --play decides what to stop highlighting and #8's --stats reports how many
// words are known. Two conditions written separately would drift, and the drift
// would show as a stats screen disagreeing with the review queue.
func Mastered(p Progress) bool {
	return p.Box >= MasteredBox
}

// Fold derives every word's Progress from the event log.
//
// DERIVED, never stored — store's event.go states the rule for the whole store:
// the log is "deliberately the ONLY record of activity ... storing counters
// alongside would create a second source of truth that drifts". A Box field on
// store.Word would be exactly that, and it would go wrong invisibly.
//
// It APPLIES Answer rather than reimplementing the transition, so there is one
// encoding of "correct promotes, wrong demotes and resets" in the package.
//
// ORDERING CONTRACT: events are consumed in the order given, which is the
// chronological order store.Store.Events documents. It is NOT order-independent
// and cannot be — two reviews sharing a timestamp with different outcomes fold
// differently by order, and ReviewEvent carries no tiebreaker.
//
// Only EventReviewed participates. A lookup or a question is activity, not
// assessment: they say what the learner is working on, which is #17's signal,
// not what they know.
func Fold(events []store.ReviewEvent) map[string]Progress {
	out := map[string]Progress{}
	for _, e := range events {
		if e.Kind != store.EventReviewed {
			continue
		}
		key := store.Key(e.Word)
		if key == "" {
			continue
		}
		out[key] = Answer(out[key], gradeOf(e), e.At)
	}
	return out
}

// GradeOf is how the two booleans an answer is recorded as read as a rung on the
// ladder.
//
// EXPORTED because there are two callers and they must not be able to disagree.
// gradeOf reconstructs it from a logged event; `--play`'s loop applies it to its
// in-memory copy of the progress map the instant an answer lands, so the cost
// figures a sitting SHOWS cannot drift from the ones the next sitting DERIVES
// from the log (#41 D7). Two spellings of one rule is how they would.
//
// Booleans rather than a ReviewEvent, so the caller with an outcome in hand does
// not have to build a log entry it is not writing.
func GradeOf(correct, unaided bool) Grade {
	switch {
	case !correct:
		return GradeWrong
	case unaided:
		return GradeUnaided
	default:
		return GradeCorrect
	}
}

// gradeOf reconstructs how an answer was given from what the log recorded.
//
// The log stores two booleans rather than the enum, because `Correct` predates
// this and rewriting history is not on offer. An event written before `unaided`
// existed reads as GradeCorrect — the conservative reading, which promotes one
// rung rather than two, so re-folding an old log can only make words due SOONER
// than the new ladder would otherwise say.
func gradeOf(e store.ReviewEvent) Grade { return GradeOf(e.Correct, e.Unaided) }
