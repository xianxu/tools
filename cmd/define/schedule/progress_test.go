package schedule

import (
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

var day0 = time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

func at(dayOffset int) time.Time { return day0.AddDate(0, 0, dayOffset) }

func reviewed(word string, correct bool, t time.Time) store.ReviewEvent {
	return store.ReviewEvent{Word: word, Kind: store.EventReviewed, Correct: correct, At: t}
}

func TestAnswerTransitions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		in         Progress
		correct    bool
		wantBox    int
		wantStreak int
	}{
		{"correct promotes and counts", Progress{Box: 0, Streak: 0}, true, 1, 1},
		{"correct again", Progress{Box: 1, Streak: 1}, true, 2, 2},
		// One box down, not back to zero: a word at the 90-day interval that
		// slips once is not a word you have never seen, and the interval is
		// where Leitner keeps the information.
		{"wrong demotes ONE box", Progress{Box: 4, Streak: 4}, false, 3, 0},
		{"wrong resets the streak", Progress{Box: 2, Streak: 9}, false, 1, 0},
		{"promotion clamps at the last box", Progress{Box: LastBox, Streak: 8}, true, LastBox, 9},
		{"demotion clamps at zero", Progress{Box: 0, Streak: 0}, false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Answer(tc.in, tc.correct, at(1))

			if got.Box != tc.wantBox {
				t.Errorf("Box = %d, want %d", got.Box, tc.wantBox)
			}
			if got.Streak != tc.wantStreak {
				t.Errorf("Streak = %d, want %d", got.Streak, tc.wantStreak)
			}
			if !got.LastReviewed.Equal(at(1)) {
				t.Errorf("LastReviewed = %v, want the time of the answer", got.LastReviewed)
			}
		})
	}
}

func TestDue(t *testing.T) {
	// A word never reviewed is due: the zero Progress means "new", and a word
	// you looked up and never reviewed is exactly what should come first.
	if !Due(Progress{}, day0) {
		t.Error("a never-reviewed word is not due; new words would never be offered")
	}

	p := Progress{Box: 1, LastReviewed: day0} // box 1 waits 3 days
	if Due(p, at(2)) {
		t.Error("due after 2 days at a 3-day interval")
	}
	if !Due(p, at(3)) {
		t.Error("not due after exactly 3 days")
	}
	if !Due(p, at(30)) {
		t.Error("not due long after the interval")
	}
}

// Local CALENDAR days, not 24h — the reason store.StartOfDay exists.
//
// A learner who reviews at 9am Monday and sits down at 8am Tuesday must find a
// box-0 word due. Under 24h arithmetic they would not, and the tool would feel
// broken in the most ordinary use there is.
func TestDueCountsCalendarDaysNotHours(t *testing.T) {
	monday9am := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	tuesday8am := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC) // 23 hours later

	p := Progress{Box: 0, LastReviewed: monday9am} // box 0 waits 1 day

	if !Due(p, tuesday8am) {
		t.Error("not due 23 hours later on the NEXT calendar day — this is 24h arithmetic, not calendar")
	}
}

func TestMastered(t *testing.T) {
	if Mastered(Progress{Box: LastBox, Streak: masteryStreak - 1}) {
		t.Error("mastered one short of the streak")
	}
	if !Mastered(Progress{Box: LastBox, Streak: masteryStreak}) {
		t.Error("not mastered at the last box with the full streak")
	}
	// Reaching the last box takes LastBox consecutive correct answers, so a
	// streak that merely got you there must NOT count as mastery — otherwise
	// "mastered" means "arrived", which is not what the word means.
	if Mastered(Progress{Box: LastBox, Streak: LastBox}) {
		t.Error("mastered on arrival at the last box; masteryStreak adds nothing")
	}
	if Mastered(Progress{Box: LastBox - 1, Streak: masteryStreak + 10}) {
		t.Error("mastered below the last box")
	}
}

// A lookup or a question is activity, not assessment: they say what the learner
// is working ON, which is #17's signal, not what they KNOW.
//
// The non-review events must come AFTER a promotion for this to discriminate.
// The first version put them first, where a spurious demotion clamps at box 0
// and disappears — so counting every event kind produced the identical answer
// and the mutant survived.
func TestFoldOnlyCountsReviews(t *testing.T) {
	events := []store.ReviewEvent{
		reviewed("obsequious", true, at(0)),
		reviewed("obsequious", true, at(1)), // box 2, streak 2
		{Word: "obsequious", Kind: store.EventLookedUp, Found: true, At: at(2)},
		{Word: "obsequious", Kind: store.EventAsked, Question: "what?", At: at(3)},
	}

	got := Fold(events)

	p := got[store.Key("obsequious")]
	if p.Box != 2 || p.Streak != 2 {
		t.Errorf("got box %d streak %d, want 2/2 — a lookup is not a wrong answer", p.Box, p.Streak)
	}
	// LastReviewed must not move either: the learner did not review on day 3.
	if !p.LastReviewed.Equal(at(1)) {
		t.Errorf("LastReviewed = %v, want the last REVIEW at %v", p.LastReviewed, at(1))
	}
}

func TestFoldNormalisesKeys(t *testing.T) {
	got := Fold([]store.ReviewEvent{
		reviewed("Obsequious", true, at(1)),
		reviewed("obsequious", true, at(2)),
	})

	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1 — Define and define are one word", len(got))
	}
	if p := got[store.Key("obsequious")]; p.Box != 2 {
		t.Errorf("box = %d, want 2 — both events must land on one word", p.Box)
	}
}

// Fold APPLIES Answer; it does not reimplement the transition. Two encodings of
// "correct promotes, wrong demotes and resets" would be two things to keep in
// agreement, and the disagreement would be invisible.
func TestFoldOfOneEventEqualsOneAnswer(t *testing.T) {
	for _, correct := range []bool{true, false} {
		got := Fold([]store.ReviewEvent{reviewed("w", correct, at(1))})[store.Key("w")]
		want := Answer(Progress{}, correct, at(1))

		if got != want {
			t.Errorf("correct=%v: Fold gave %+v, Answer gave %+v", correct, got, want)
		}
	}
}

// The Done-when's own row: promotion, demotion and due-dates across a simulated
// multi-week schedule with an explicit clock.
func TestMultiWeekSchedule(t *testing.T) {
	word := "obsequious"
	var events []store.ReviewEvent

	// Six correct answers, each on the day the previous interval came due.
	// Intervals are 1, 3, 7, 14, 30, 90 — so the review days are the running sum.
	reviewDays := []int{0, 1, 4, 11, 25, 55}
	for _, d := range reviewDays {
		events = append(events, reviewed(word, true, at(d)))
	}

	p := Fold(events)[store.Key(word)]

	if p.Box != LastBox {
		t.Fatalf("after %d correct answers box = %d, want the last box %d", len(reviewDays), p.Box, LastBox)
	}
	if p.Streak != len(reviewDays) {
		t.Errorf("streak = %d, want %d", p.Streak, len(reviewDays))
	}
	// At the 90-day interval it is not due the next day, and is on day 90.
	if Due(p, at(56)) {
		t.Error("due one day after reaching the 90-day interval")
	}
	if !Due(p, at(55+90)) {
		t.Error("not due 90 days after the last review")
	}

	// One miss: down one box, streak gone, and due sooner because the interval
	// shortened — which is the entire point of the demotion.
	events = append(events, reviewed(word, false, at(145)))
	p = Fold(events)[store.Key(word)]

	if p.Box != LastBox-1 {
		t.Errorf("after a miss box = %d, want %d", p.Box, LastBox-1)
	}
	if p.Streak != 0 {
		t.Errorf("after a miss streak = %d, want 0", p.Streak)
	}
	if Mastered(p) {
		t.Error("still mastered after a miss")
	}
	if !Due(p, at(145+30)) {
		t.Error("not due 30 days after the demotion — the shortened interval is the point")
	}
}

// The properties that hold over the WHOLE domain. Permutation-independence is
// deliberately NOT among them: two reviews sharing an At with different Correct
// fold differently by order and ReviewEvent has no tiebreaker, so Fold's
// contract is that it consumes events in the order store.Events returns them.
func FuzzFold(f *testing.F) {
	f.Add(uint16(0), 3)
	f.Add(uint16(0xFFFF), 16)
	f.Add(uint16(0b1010101010101010), 9)

	f.Fuzz(func(t *testing.T, answers uint16, n int) {
		if n < 0 {
			n = -n
		}
		n %= 17
		var events []store.ReviewEvent
		for i := 0; i < n; i++ {
			events = append(events, reviewed("w", answers&(1<<uint(i%16)) != 0, at(i)))
		}

		got := Fold(events)
		p := got[store.Key("w")]

		if p.Box < 0 || p.Box > LastBox {
			t.Fatalf("box %d outside the ladder after %d events", p.Box, n)
		}
		if p.Streak < 0 {
			t.Fatalf("negative streak %d", p.Streak)
		}
		// Idempotent over a re-fold of the same slice: folding is a pure
		// function of its input, so calling it twice cannot differ.
		if again := Fold(events)[store.Key("w")]; again != p {
			t.Fatalf("Fold is not idempotent: %+v then %+v", p, again)
		}
	})
}
