package schedule_test

import (
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/schedule"
	"github.com/xianxu/tools/cmd/define/store"
)

// nyc and kolkata are the two zones this file reasons in: one with DST, one with
// a HALF-HOUR offset. The second exists because a fold built on duration
// arithmetic passes every whole-hour test.
var (
	nyc       = mustLoad("America/New_York")
	kolkata   = mustLoad("Asia/Kolkata")
	kathmandu = mustLoad("Asia/Kathmandu") // +05:45, the quarter-hour case
)

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func at(loc *time.Location, y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, loc)
}

func looked(word string, t time.Time) store.ReviewEvent {
	return store.ReviewEvent{Word: word, Kind: store.EventLookedUp, Found: true, At: t}
}

func reviewed(word, form string, correct bool, t time.Time) store.ReviewEvent {
	return store.ReviewEvent{Word: word, Kind: store.EventReviewed, Found: true, Correct: correct, Form: form, At: t}
}

func deckOf(words ...string) []store.Word {
	out := make([]store.Word, 0, len(words))
	for _, w := range words {
		out = append(out, store.Word{Text: w})
	}
	return out
}

// KNOWN COMES FROM THE DECK, NOT THE LOG, and the two genuinely differ.
//
// `--forget` removes a word and deliberately leaves its events — the deck is a
// working set, the log is history — so a fold over the log alone reports words
// the learner has deleted. That is the most likely wrong number on this screen,
// which is why the signature takes both.
func TestKnownCountsTheDeckNotTheLog(t *testing.T) {
	now := at(nyc, 2026, time.June, 10, 12)
	events := []store.ReviewEvent{
		looked("ephemeral", at(nyc, 2026, time.June, 1, 9)),
		looked("forgotten", at(nyc, 2026, time.June, 2, 9)), // later --forget'd
	}
	// The deck has only the word that was not forgotten.
	got := schedule.Summarise(events, deckOf("ephemeral"), now)

	if got.Known != 1 {
		t.Errorf("Known = %d, want 1 — the log mentions a word the deck no longer holds", got.Known)
	}
	// AND THE LOG STILL COUNTS IT AS ADDED, which is the same asymmetry going the
	// other way: added-per-day asks what HAPPENED, and a forgotten word was still
	// a word added that day. Folding the deck for it would rewrite the past every
	// time --forget ran.
	if got.AddedPerDay == 0 {
		t.Error("AddedPerDay is 0; a forgotten word was still added on the day it was looked up")
	}
}

// MASTERY IS ONE FUNCTION'S ANSWER, and this asserts the agreement rather than
// re-deriving the rule — `Mastered`'s own doc names this issue as its second
// consumer and says the drift "would show as a stats screen disagreeing with the
// review queue".
func TestMasteredAgreesWithTheScheduleFunction(t *testing.T) {
	now := at(nyc, 2026, time.June, 10, 12)
	// A word promoted to the mastered box, and one that lapsed back out of it.
	var events []store.ReviewEvent
	for i := 0; i < 12; i++ {
		events = append(events, reviewed("mastered", "meaning", true, at(nyc, 2026, time.January, 1+i, 9)))
		events = append(events, reviewed("lapsed", "meaning", true, at(nyc, 2026, time.January, 1+i, 9)))
	}
	events = append(events, reviewed("lapsed", "meaning", false, at(nyc, 2026, time.February, 1, 9)))

	deck := deckOf("mastered", "lapsed")
	got := schedule.Summarise(events, deck, now)

	// The reference answer, computed the way --play computes it.
	prog := schedule.Fold(events)
	want := 0
	for _, w := range deck {
		if schedule.Mastered(prog[store.Key(w.Text)]) {
			want++
		}
	}
	if got.Mastered != want {
		t.Errorf("Mastered = %d, want %d — the screen and the queue must give one answer",
			got.Mastered, want)
	}
	// And the fixture reaches the branch: a lapse must actually cost mastery,
	// or this row is comparing two zeros.
	if !schedule.Mastered(prog["mastered"]) || schedule.Mastered(prog["lapsed"]) {
		t.Fatalf("the fixture does not distinguish mastery: mastered=%v lapsed=%v",
			schedule.Mastered(prog["mastered"]), schedule.Mastered(prog["lapsed"]))
	}
}

// A STREAK IS A LOCAL CALENDAR QUESTION.
func TestStreaksCountLocalCalendarDays(t *testing.T) {
	now := at(nyc, 2026, time.June, 5, 12)
	events := []store.ReviewEvent{
		reviewed("a", "meaning", true, at(nyc, 2026, time.June, 3, 23)),
		reviewed("b", "meaning", true, at(nyc, 2026, time.June, 4, 1)),
		reviewed("c", "meaning", true, at(nyc, 2026, time.June, 5, 9)),
	}
	got := schedule.Summarise(events, nil, now)
	if got.ActiveDays != 3 {
		t.Errorf("ActiveDays = %d, want 3", got.ActiveDays)
	}
	if got.CurrentStreak != 3 {
		t.Errorf("CurrentStreak = %d, want 3", got.CurrentStreak)
	}
}

// TODAY IS NOT REQUIRED. A learner who reviewed yesterday and has not yet sat
// down keeps their streak; breaking it at midnight would punish them for the
// hour in which they read this screen.
func TestAStreakSurvivesUntilADayIsMissed(t *testing.T) {
	events := []store.ReviewEvent{
		reviewed("a", "meaning", true, at(nyc, 2026, time.June, 3, 9)),
		reviewed("b", "meaning", true, at(nyc, 2026, time.June, 4, 9)),
	}
	// Today is the 5th and unused: the streak stands at 2.
	if got := schedule.Summarise(events, nil, at(nyc, 2026, time.June, 5, 12)); got.CurrentStreak != 2 {
		t.Errorf("with today unused, CurrentStreak = %d, want 2", got.CurrentStreak)
	}
	// The 6th, with the 5th missed: the streak is gone.
	if got := schedule.Summarise(events, nil, at(nyc, 2026, time.June, 6, 12)); got.CurrentStreak != 0 {
		t.Errorf("after a missed day, CurrentStreak = %d, want 0", got.CurrentStreak)
	}
}

func TestTheLongestStreakSurvivesAGap(t *testing.T) {
	now := at(nyc, 2026, time.June, 20, 12)
	var events []store.ReviewEvent
	for _, d := range []int{1, 2, 3, 4, 5} { // a five-day run, long ago
		events = append(events, reviewed("a", "meaning", true, at(nyc, 2026, time.June, d, 9)))
	}
	for _, d := range []int{19, 20} { // a two-day run, current
		events = append(events, reviewed("a", "meaning", true, at(nyc, 2026, time.June, d, 9)))
	}
	got := schedule.Summarise(events, nil, now)
	if got.LongestStreak != 5 {
		t.Errorf("LongestStreak = %d, want 5", got.LongestStreak)
	}
	if got.CurrentStreak != 2 {
		t.Errorf("CurrentStreak = %d, want 2", got.CurrentStreak)
	}
}

// THE DST ROW THE DONE-WHEN NAMES. 2026-03-08 is 23 hours in New York and
// 2026-11-01 is 25 — the two days on which `now.Sub(then)/24h` lands on the
// wrong calendar date.
func TestStreaksAcrossDSTBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		days []time.Time
		now  time.Time
	}{
		{
			"spring forward, a 23-hour day",
			[]time.Time{
				at(nyc, 2026, time.March, 7, 20),
				at(nyc, 2026, time.March, 8, 20),
				at(nyc, 2026, time.March, 9, 20),
			},
			at(nyc, 2026, time.March, 9, 22),
		},
		{
			"fall back, a 25-hour day",
			[]time.Time{
				at(nyc, 2026, time.October, 31, 20),
				at(nyc, 2026, time.November, 1, 20),
				at(nyc, 2026, time.November, 2, 20),
			},
			at(nyc, 2026, time.November, 2, 22),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []store.ReviewEvent
			for _, d := range tc.days {
				events = append(events, reviewed("a", "meaning", true, d))
			}
			got := schedule.Summarise(events, nil, tc.now)
			if got.ActiveDays != 3 {
				t.Errorf("ActiveDays = %d, want 3 — a DST day is not two days or none", got.ActiveDays)
			}
			if got.CurrentStreak != 3 {
				t.Errorf("CurrentStreak = %d, want 3 — the run crosses the offset change", got.CurrentStreak)
			}
		})
	}
}

// ZONES WHOSE OFFSET IS NOT A WHOLE HOUR. Kolkata is +05:30 and Kathmandu
// +05:45; an implementation that rounded to hours would pass every test above.
func TestStreaksInFractionalOffsetZones(t *testing.T) {
	for _, loc := range []*time.Location{kolkata, kathmandu} {
		t.Run(loc.String(), func(t *testing.T) {
			events := []store.ReviewEvent{
				reviewed("a", "meaning", true, at(loc, 2026, time.June, 3, 23)),
				reviewed("b", "meaning", true, at(loc, 2026, time.June, 4, 0)),
			}
			got := schedule.Summarise(events, nil, at(loc, 2026, time.June, 4, 12))
			if got.ActiveDays != 2 || got.CurrentStreak != 2 {
				t.Errorf("ActiveDays = %d, CurrentStreak = %d, want 2 and 2",
					got.ActiveDays, got.CurrentStreak)
			}
		})
	}
}

// THE CALENDAR IS THE LEARNER'S, whatever zone an event was written in.
//
// The map is keyed by time.Time, whose equality includes the *Location POINTER,
// so an event that came back carrying a fixed +01:00 zone and a `now` in
// time.Local produce two different keys for ONE calendar day unless the fold
// converts first. The streak would then miss days the learner actually used —
// and a YAML log routinely holds mixed offsets, because DST changes the one the
// machine writes.
func TestDaysAreCountedInTheLearnersZone(t *testing.T) {
	fixed := time.FixedZone("+0100", 3600)
	// Two events that are the same New York day, written in different zones.
	events := []store.ReviewEvent{
		reviewed("a", "meaning", true, at(nyc, 2026, time.June, 3, 9)),
		reviewed("b", "meaning", true, at(nyc, 2026, time.June, 3, 14).In(fixed)),
		reviewed("c", "meaning", true, at(nyc, 2026, time.June, 4, 9).In(time.UTC)),
	}
	got := schedule.Summarise(events, nil, at(nyc, 2026, time.June, 4, 20))
	if got.ActiveDays != 2 {
		t.Errorf("ActiveDays = %d, want 2 — the same calendar day written in two zones is one day",
			got.ActiveDays)
	}
	if got.CurrentStreak != 2 {
		t.Errorf("CurrentStreak = %d, want 2 — a zone change must not break a run", got.CurrentStreak)
	}
}

// A FLAGGED QUESTION IS NOT AN ATTEMPT. #12 records it with its option set and
// moves on WITHOUT marking the learner wrong, because a broken question is not
// evidence about them. Counting it here would let bad material lower a score.
func TestAFlaggedQuestionIsNotAnAttempt(t *testing.T) {
	now := at(nyc, 2026, time.June, 10, 12)
	events := []store.ReviewEvent{
		reviewed("a", "cloze", true, at(nyc, 2026, time.June, 1, 9)),
		{Word: "b", Kind: store.EventFlagged, Form: "cloze", At: at(nyc, 2026, time.June, 1, 10)},
	}
	got := schedule.Summarise(events, nil, now)
	if a := got.Accuracy["cloze"]; a.Attempts != 1 || a.Correct != 1 {
		t.Errorf("cloze accuracy = %+v, want 1 of 1 — a flag is not an attempt", a)
	}
	if got.Accuracy["cloze"].Rate() != 1 {
		t.Errorf("Rate = %v, want 1", got.Accuracy["cloze"].Rate())
	}
}

// ACCURACY IS KEYED BY THE FORM THE LOG RECORDS, so a form added to `play`
// appears here with no edit to this package.
func TestAccuracyIsKeyedByTheFormTheLogRecords(t *testing.T) {
	now := at(nyc, 2026, time.June, 10, 12)
	events := []store.ReviewEvent{
		reviewed("a", "meaning", true, at(nyc, 2026, time.June, 1, 9)),
		reviewed("b", "cloze", false, at(nyc, 2026, time.June, 1, 10)),
		// A form nobody has written yet: it must appear rather than be dropped.
		reviewed("c", "unheard-of", true, at(nyc, 2026, time.June, 1, 11)),
	}
	got := schedule.Summarise(events, nil, now)
	for _, form := range []string{"meaning", "cloze", "unheard-of"} {
		if got.Accuracy[form].Attempts != 1 {
			t.Errorf("form %q has %d attempts, want 1 — the key comes from the log, "+
				"not from a list this package maintains", form, got.Accuracy[form].Attempts)
		}
	}
}

// ADDED-PER-DAY IS OVER THE ACTIVE WINDOW, not since the epoch: an average
// diluted by dormant months answers a question nobody asked.
func TestAddedPerDayIsOverTheActiveWindow(t *testing.T) {
	now := at(nyc, 2026, time.June, 30, 12)
	events := []store.ReviewEvent{
		// Four words over two active days, then three dormant weeks.
		looked("a", at(nyc, 2026, time.June, 1, 9)),
		looked("b", at(nyc, 2026, time.June, 1, 10)),
		looked("c", at(nyc, 2026, time.June, 2, 9)),
		looked("d", at(nyc, 2026, time.June, 2, 10)),
	}
	got := schedule.Summarise(events, deckOf("a", "b", "c", "d"), now)
	if got.AddedPerDay != 2 {
		t.Errorf("AddedPerDay = %v, want 2 — four words over the two days actually used, "+
			"not over the month", got.AddedPerDay)
	}
	// A REPEAT LOOKUP IS NOT AN ADD, or a learner re-reading one entry looks
	// prolific.
	events = append(events, looked("a", at(nyc, 2026, time.June, 3, 9)))
	if got := schedule.Summarise(events, deckOf("a", "b", "c", "d"), now); got.AddedPerDay != 4.0/3.0 {
		t.Errorf("AddedPerDay = %v, want 4/3 — the repeat added a day, not a word", got.AddedPerDay)
	}
}

// THE LOG IS HAND-EDITABLE INPUT, and a bad timestamp poisons every figure
// SILENTLY (ARCH-SECURE). Skipped rather than clamped: a clamped event is a
// fabricated fact.
func TestBadTimestampsAreSkippedNotBelieved(t *testing.T) {
	now := at(nyc, 2026, time.June, 10, 12)
	good := reviewed("a", "meaning", true, at(nyc, 2026, time.June, 10, 9))

	t.Run("a zero At", func(t *testing.T) {
		// Year 1 would make ActiveDays span two millennia and the streak
		// arithmetic meaningless.
		events := []store.ReviewEvent{{Word: "z", Kind: store.EventReviewed, Form: "meaning"}, good}
		got := schedule.Summarise(events, nil, now)
		if got.ActiveDays != 1 {
			t.Errorf("ActiveDays = %d, want 1 — an event with no time is not a day", got.ActiveDays)
		}
		if got.Accuracy["meaning"].Attempts != 1 {
			t.Errorf("a dateless event was counted as an attempt: %+v", got.Accuracy["meaning"])
		}
	})

	t.Run("an At after now", func(t *testing.T) {
		// A future event leaves a gap nothing can close, so the current streak
		// would read as broken forever. It is also what a clock-skewed machine
		// writes.
		future := reviewed("f", "meaning", true, at(nyc, 2026, time.July, 1, 9))
		got := schedule.Summarise([]store.ReviewEvent{good, future}, nil, now)
		if got.ActiveDays != 1 {
			t.Errorf("ActiveDays = %d, want 1 — a fold cannot be evidence about the future",
				got.ActiveDays)
		}
		if got.CurrentStreak != 1 {
			t.Errorf("CurrentStreak = %d, want 1 — a future event must not break today's run",
				got.CurrentStreak)
		}
	})
}

// AN EMPTY LOG AND AN EMPTY DECK PRODUCE ZEROES, NOT A PANIC — the state every
// new learner is in, and the Done-when's third row at the fold level.
func TestSummariseOnNothing(t *testing.T) {
	got := schedule.Summarise(nil, nil, at(nyc, 2026, time.June, 10, 12))
	if got.Known != 0 || got.ActiveDays != 0 || got.CurrentStreak != 0 || got.LongestStreak != 0 {
		t.Errorf("empty fold = %+v, want zeroes", got)
	}
	if got.AddedPerDay != 0 {
		t.Errorf("AddedPerDay = %v on an empty log; a division by zero days must not happen",
			got.AddedPerDay)
	}
	if got.Accuracy == nil {
		t.Error("Accuracy is nil; a caller ranging over it should not have to nil-check")
	}
	if !got.FirstDay.IsZero() || !got.LastDay.IsZero() {
		t.Errorf("FirstDay/LastDay = %v/%v, want the zero time", got.FirstDay, got.LastDay)
	}
}
