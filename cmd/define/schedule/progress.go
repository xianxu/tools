package schedule

import (
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// masteryStreak is how many consecutive correct answers mastery takes, and the
// number needs a reason rather than a taste.
//
// Reaching the last box from box 0 takes LastBox consecutive correct answers, so
// any value at or below that would make Mastered mean "arrived" rather than
// "known". This is "the promotions that reach the 90-day interval, plus two
// confirmations at it" — which is the one-sentence explanation the Spec demands
// of every scheduling answer.
const masteryStreak = 7

// Progress is one word's schedule state, DERIVED from the event log.
//
// No review count: #8 can fold the log for that directly, and a field with no
// reader is a field that goes stale.
type Progress struct {
	Box          int
	Streak       int // consecutive correct; any wrong answer resets it
	LastReviewed time.Time
}

// Answer is the transition: one review, one new state.
//
// Correct promotes a box; wrong demotes ONE box and resets the streak. Demotion
// is a single step rather than a fall to zero because a word at the 90-day
// interval that slips once is not a word you have never seen — the interval is
// where Leitner keeps what you have learned.
func Answer(p Progress, correct bool, at time.Time) Progress {
	if correct {
		return Progress{Box: clampBox(p.Box + 1), Streak: p.Streak + 1, LastReviewed: at}
	}
	return Progress{Box: clampBox(p.Box - 1), Streak: 0, LastReviewed: at}
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

// Mastered is the FINAL box reached with masteryStreak consecutive correct.
//
// Exported and defined once because two consumers need the same answer: #6's
// --play decides what to stop offering, and #8's --stats reports how many words
// are known. Two conditions written separately would drift, and the drift would
// show as a stats screen disagreeing with the review queue.
func Mastered(p Progress) bool {
	return p.Box >= LastBox && p.Streak >= masteryStreak
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
		out[key] = Answer(out[key], e.Correct, e.At)
	}
	return out
}
