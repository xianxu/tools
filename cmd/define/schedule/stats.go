package schedule

import (
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Stats is every figure `--stats` shows, folded out of the event log.
//
// NOTHING IS STORED. #3 chose an append-only log precisely so there would be no
// counters to keep in sync, and the drift a counter permits is invisible: a
// number that is merely wrong looks exactly like a number that is right. Every
// field here is derived on read, and the cost of that is measured in
// milliseconds (see Summarise).
type Stats struct {
	// Known is how many words are IN THE DECK, not how many the log mentions.
	Known int
	// Mastered is how many of those have stopped needing attention, as
	// schedule.Mastered decides it — never as a box comparison written here.
	Mastered int
	// ActiveDays is the number of local calendar days carrying at least one
	// event.
	ActiveDays int
	// CurrentStreak counts back from today, and does NOT require today: a
	// learner who reviewed yesterday and has not yet sat down keeps their
	// streak. Breaking it at midnight would punish them for the hour in which
	// they read this screen.
	CurrentStreak int
	// LongestStreak is the longest run of consecutive active days ever.
	LongestStreak int
	// Added is how many words the log records being looked up for the first
	// time. It exists so the renderer can tell "none recorded" from "genuinely
	// slow" — a deck whose words arrived before the log did, or was seeded by
	// hand, has words but no adds, and a rate of 0.0 beside a deck of 41 reads
	// as a bug rather than as the absence it is.
	Added int
	// AddedPerDay is words added per active day — over the window the learner
	// has actually been using this, not since the epoch. An average diluted by
	// dormant months answers a question nobody asked.
	AddedPerDay float64
	// Accuracy is per form NAME, keyed by what the log already records
	// (store.ReviewEvent.Form, a string) so a form added to `play` appears here
	// with no edit.
	Accuracy map[string]FormAccuracy
	// FirstDay and LastDay bound the record, and are the zero time when there is
	// nothing to bound.
	FirstDay, LastDay time.Time
}

// FormAccuracy is how a learner does on one form.
type FormAccuracy struct {
	// Attempts counts REVIEWED events only. A flagged question is not an
	// attempt: #12 records it with its option set and moves on without marking
	// the learner wrong, because a broken question is not evidence about them.
	// Counting it here would let bad material lower an accuracy score.
	Attempts int
	Correct  int
}

// Rate is correct over attempts, or 0 when there are none.
func (f FormAccuracy) Rate() float64 {
	if f.Attempts == 0 {
		return 0
	}
	return float64(f.Correct) / float64(f.Attempts)
}

// Summarise folds the log and the deck into one screen's worth of figures.
//
// IT TAKES BOTH, and the asymmetry is the design rather than an inconvenience:
//
//   - KNOWN AND MASTERED COME FROM THE DECK. `--forget` removes a word and
//     deliberately leaves its events — the deck is a working set, the log is
//     history — so a log-only count reports words the learner has deleted.
//   - ADDED-PER-DAY COMES FROM THE LOG, for the opposite reason. It asks what
//     HAPPENED, and a forgotten word was still a word added that day; folding
//     the deck for it would rewrite the past every time --forget ran.
//
// PURE, INCLUDING THE CLOCK. `now` is a parameter rather than an injected
// interface, so every timezone and DST case is a table row instead of a fake.
// The issue's Done-when asks for a fake clock; this is stronger, and the issue
// records the deviation.
func Summarise(events []store.ReviewEvent, deck []store.Word, now time.Time) Stats {
	// ONE FILTER, EVERY FIGURE. Fold ran on the RAW slice while the loop below
	// ran on countable's survivors, so a hand-edited future timestamp was
	// excluded from the streak and included in the box — the mastered count and
	// the active days disagreeing about which events are real (#8 BR-6).
	//
	// Filtering once, here, is also the only place that can: countable needs
	// `now`, which Fold has no business knowing.
	events = countableEvents(events, now)
	prog := Fold(events)
	s := Stats{Accuracy: map[string]FormAccuracy{}}

	for _, w := range deck {
		key := store.Key(w.Text)
		if key == "" {
			continue
		}
		s.Known++
		if Mastered(prog[key]) {
			s.Mastered++
		}
	}

	days := map[civilDay]bool{}
	// seen is the words the log has already introduced, so "added" counts a word
	// ONCE. This walks the events rather than asking Fold, because Progress
	// carries no first-seen instant (Box, MaxBox, LastReviewed — progress.go:52)
	// — a draft of this file assumed it did and the compiler said otherwise.
	//
	// It relies on the log being CHRONOLOGICAL, which the seam promises
	// (store/store.go:18) and both implementations keep (mem.go:105,
	// yaml.go:353). Out of order, this would credit the wrong day; it would still
	// count each word once.
	seen := map[string]bool{}
	var added int
	for _, e := range events {
		// No filter here: countableEvents already ran, so every event in this
		// slice is one every figure agrees is real.
		//
		// IN now's LOCATION FIRST, then as a DATE. Both halves are load-bearing
		// and each fixes a different bug.
		//
		// The location, because store.DaysBetween's doc settles whose calendar
		// this is — "b's location defines the calendar… the caller's question is
		// always how many days have passed for the LEARNER, and the learner is
		// wherever now is" (clock.go:46).
		//
		// The DATE, because an instant can fail to exist. This was keyed by
		// store.StartOfDay's time.Time, and in a zone whose DST transition
		// happens at LOCAL MIDNIGHT that midnight is not a real instant:
		// time.Date normalises it, and in Havana StartOfDay(2026-03-08) returns
		// 2026-03-07T23:00 — a key on the PREVIOUS DAY'S date. Santiago
		// (2026-09-06) and Beirut (2026-03-29) do the same. A run spanning such a
		// date then reads as broken, which is the streak silently lying about the
		// learner. A civil date cannot fail to exist.
		day := civilDayOf(e.At.In(now.Location()))
		days[day] = true
		// FirstDay/LastDay stay INSTANTS because relativeDay renders them, and
		// they are built from the event rather than from the date so the zone a
		// reader sees is the learner's.
		atLocal := e.At.In(now.Location())
		if s.FirstDay.IsZero() || atLocal.Before(s.FirstDay) {
			s.FirstDay = atLocal
		}
		if atLocal.After(s.LastDay) {
			s.LastDay = atLocal
		}
		switch e.Kind {
		case store.EventLookedUp:
			// A word is ADDED the first time it is looked up and found. Counting
			// every lookup would make a learner who re-reads one entry look
			// prolific.
			if key := store.Key(e.Word); e.Found && key != "" && !seen[key] {
				seen[key] = true
				added++
			}
		case store.EventReviewed:
			a := s.Accuracy[e.Form]
			a.Attempts++
			if e.Correct {
				a.Correct++
			}
			s.Accuracy[e.Form] = a
		}
	}

	s.Added = added
	s.ActiveDays = len(days)
	s.CurrentStreak, s.LongestStreak = streaks(days, now)
	if s.ActiveDays > 0 {
		s.AddedPerDay = float64(added) / float64(s.ActiveDays)
	}
	return s
}

// countable reports whether an event can be counted, and at what instant.
//
// THE LOG IS HAND-EDITABLE INPUT. `events/` is plain YAML in a directory the
// README invites editing, and a bad timestamp poisons every figure SILENTLY:
//
//   - a zero At lands on year 1, which makes ActiveDays span two millennia and
//     the streak arithmetic meaningless;
//   - an At after now leaves a gap in the calendar that nothing can close, so
//     the current streak reads as broken forever. That is also the shape a
//     clock-skewed machine writes.
//
// Skipped rather than clamped: a clamped event is a fabricated fact, and the
// figures degrade to "fewer events" rather than to a wrong number — the same
// choice sanitiseItem and readCapped make one package over (ARCH-SECURE).
func countable(e store.ReviewEvent, now time.Time) bool {
	return !e.At.IsZero() && !e.At.After(now)
}

// countableEvents is the filter applied ONCE, so every figure folds the same
// events — the boxes as well as the days.
func countableEvents(events []store.ReviewEvent, now time.Time) []store.ReviewEvent {
	out := make([]store.ReviewEvent, 0, len(events))
	for _, e := range events {
		if countable(e, now) {
			out = append(out, e)
		}
	}
	return out
}

// streaks walks the active days and returns the current and longest runs.
//
// LOCAL CALENDAR DAYS, walked with AddDate rather than with a duration.
//
// AddDate moves the CALENDAR DATE and lets the offset change underneath, which
// is what "the day before" means to a reader. `Sub()/24h` passes every
// whole-hour test and then breaks on the 23- and 25-hour DST days — the bug the
// Done-when asks to be verified against, and the one historyWindow's own comment
// records having met (history_cmd.go:24-34).
//
// It does not call store.DaysBetween: that answers "how many days between these
// two instants", and this needs "is the previous day present", which is a map
// lookup once every key is in one location (see Summarise). The RULE is
// DaysBetween's — b's location defines the calendar — applied where the keys are
// built.
//
// TODAY IS NOT REQUIRED for the current streak. A learner who reviewed yesterday
// still has it; the streak breaks when a day is MISSED, not when a day has not
// yet been used.
func streaks(days map[civilDay]bool, now time.Time) (current, longest int) {
	if len(days) == 0 {
		return 0, 0
	}
	today := civilDayOf(now)

	// The current run walks back from today, allowing today itself to be empty.
	for d := today; ; d = d.add(-1) {
		if days[d] {
			current++
			continue
		}
		if d == today {
			// Today unused is not a break — yet.
			continue
		}
		break
	}

	// The longest run is over the days themselves, so it needs no ordering: for
	// each day that STARTS a run (its predecessor is absent), walk forward.
	for d := range days {
		if days[d.add(-1)] {
			continue
		}
		n := 0
		for c := d; days[c]; c = c.add(1) {
			n++
		}
		if n > longest {
			longest = n
		}
	}
	return current, longest
}

// civilDay is a calendar date — no instant, no zone, no offset.
//
// THE KEY MUST BE A DATE, NOT A MOMENT. Keying the day set by time.Time was
// wrong twice: its equality includes the *Location pointer, so one calendar day
// written in two offsets made two keys; and worse, an instant can fail to
// EXIST. Where a DST transition happens at local midnight — Havana 2026-03-08,
// Santiago 2026-09-06, Beirut 2026-03-29 — there is no 00:00, and time.Date
// normalises it into the previous day, so the day's key landed on its
// neighbour's date and any run through it read as broken.
//
// A date has neither problem: it is three integers, comparable by ==, and
// 2026-03-08 exists in every zone whatever its clocks did that night.
type civilDay struct {
	year  int
	month time.Month
	day   int
}

func civilDayOf(t time.Time) civilDay {
	y, m, d := t.Date()
	return civilDay{year: y, month: m, day: d}
}

// add moves by whole calendar days.
//
// The arithmetic runs at NOON UTC deliberately. AddDate on a real zone can land
// in a gap or a fold; at noon in a zone that has neither, "the day before" is
// always the date a reader would name, and the result is converted straight back
// to a date. This is calendar arithmetic, so it is done in the one place where
// no clock has ever been moved.
func (c civilDay) add(days int) civilDay {
	t := time.Date(c.year, c.month, c.day, 12, 0, 0, 0, time.UTC).AddDate(0, 0, days)
	return civilDayOf(t)
}
