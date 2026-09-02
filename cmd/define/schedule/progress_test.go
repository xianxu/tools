package schedule

import (
	"testing"
	"time"
	// Embedded so the zone tests below RUN rather than skip. The repo already
	// does this in history_cmd_test.go: a t.Skip on the only test pinning a
	// correctness fix is not a pin at all — it is a green result on a machine
	// that never checked.
	_ "time/tzdata"

	"github.com/xianxu/tools/cmd/define/store"
)

var day0 = time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

func at(dayOffset int) time.Time { return day0.AddDate(0, 0, dayOffset) }

func reviewed(word string, correct bool, t time.Time) store.ReviewEvent {
	return store.ReviewEvent{Word: word, Kind: store.EventReviewed, Correct: correct, At: t}
}

func TestAnswerTransitions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		in               Progress
		grade            Grade
		wantBox, wantMax int
	}{
		{"correct promotes one rung", Progress{Box: 0}, GradeCorrect, 1, 1},
		{"correct again", Progress{Box: 1, MaxBox: 1}, GradeCorrect, 2, 2},
		{"unaided promotes TWO", Progress{Box: 2, MaxBox: 2}, GradeUnaided, 4, 4},

		// The express lane: below the high-water mark, a correct answer climbs
		// two. Relearning is faster than learning.
		{"correct climbs two while below MaxBox", Progress{Box: 5, MaxBox: 9}, GradeCorrect, 7, 9},
		{"and one on arrival at it", Progress{Box: 9, MaxBox: 9}, GradeCorrect, 10, 10},

		// The step is CAPPED at two: unaided below MaxBox is not four. The two
		// reasons to move faster are the same reason.
		{"unaided below MaxBox still climbs only two", Progress{Box: 4, MaxBox: 12}, GradeUnaided, 6, 12},

		// Halving scales: harsh where harshness is warranted, gentle where not.
		{"wrong HALVES a high box", Progress{Box: 12, MaxBox: 12}, GradeWrong, 6, 11},
		{"wrong barely moves a low box", Progress{Box: 2, MaxBox: 2}, GradeWrong, 1, 1},
		{"wrong at box 1 lands at zero", Progress{Box: 1, MaxBox: 3}, GradeWrong, 0, 2},

		{"promotion clamps at the ladder limit", Progress{Box: ladderLimit, MaxBox: ladderLimit}, GradeCorrect, ladderLimit, ladderLimit},
		{"demotion clamps at zero", Progress{Box: 0}, GradeWrong, 0, 0},

		// The zero Grade demotes. A caller that forgets to set one must not
		// promote: a word wrongly demoted returns sooner, while one wrongly
		// promoted disappears for months.
		{"the zero Grade is wrong, not correct", Progress{Box: 8, MaxBox: 8}, Grade(0), 4, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Answer(tc.in, tc.grade, at(1))

			if got.Box != tc.wantBox {
				t.Errorf("Box = %d, want %d", got.Box, tc.wantBox)
			}
			if got.MaxBox != tc.wantMax {
				t.Errorf("MaxBox = %d, want %d", got.MaxBox, tc.wantMax)
			}
			if !got.LastReviewed.Equal(at(1)) {
				t.Errorf("LastReviewed = %v, want the time of the answer", got.LastReviewed)
			}
		})
	}
}

// THE RECOVERY WALK, step by step — the plan's worked example, and the pin that
// makes the halving and the express lane a pair.
//
// Removing EITHER fails this: without the lane the walk takes five reviews, and
// without the halving it never leaves the top. A test that asserted only the end
// state would pass on the gentle-demotion design this replaces.
func TestRecoveryFromALapse(t *testing.T) {
	p := Progress{Box: 10, MaxBox: 10}

	p = Answer(p, GradeWrong, at(0))
	if p.Box != 5 || p.MaxBox != 9 {
		t.Fatalf("after the lapse box=%d max=%d, want 5 and 9", p.Box, p.MaxBox)
	}
	// The first retest lands 10 days later, which is where relearning happens —
	// not 175 days later, which is what a single-step demotion would have given.
	if got := IntervalDays(p.Box); got != 10 {
		t.Errorf("the retest interval is %d days, want 10", got)
	}

	for i, want := range []int{7, 9, 10} {
		p = Answer(p, GradeCorrect, at(i+1))
		if p.Box != want {
			t.Fatalf("recovery step %d: box = %d, want %d", i+1, p.Box, want)
		}
	}
	if p.Box != 10 {
		t.Fatalf("recovered to box %d, want 10", p.Box)
	}
}

// The express lane ERODES, so a word that keeps failing is eventually relearned
// properly rather than being waved back up forever.
func TestRepeatedLapsesEndTheExpressLane(t *testing.T) {
	p := Progress{Box: 12, MaxBox: 12}
	for i := 0; i < 6; i++ {
		p = Answer(p, GradeWrong, at(i))
		p = Answer(p, GradeCorrect, at(i))
	}
	if p.MaxBox > 8 {
		t.Errorf("after six lapses MaxBox is still %d — the lane never closes, so a "+
			"chronically failing word keeps skipping the ladder", p.MaxBox)
	}
	if p.Box > p.MaxBox {
		t.Errorf("box %d exceeds MaxBox %d", p.Box, p.MaxBox)
	}
}

func TestDue(t *testing.T) {
	// A word never reviewed is due: the zero Progress means "new", and a word
	// you looked up and never reviewed is exactly what should come first.
	if !Due(Progress{}, day0) {
		t.Error("a never-reviewed word is not due; new words would never be offered")
	}

	// Box 3 waits 4 days. Deliberately not box 1, which waits ONE day and would
	// make "not due yet" and "due" a single calendar day apart — a fixture too
	// tight to distinguish an off-by-one from a working interval.
	p := Progress{Box: 3, MaxBox: 3, LastReviewed: day0}
	if Due(p, at(3)) {
		t.Error("due after 3 days at a 4-day interval")
	}
	if !Due(p, at(4)) {
		t.Error("not due after exactly 4 days")
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
	if Mastered(Progress{Box: MasteredBox - 1}) {
		t.Error("mastered one box short")
	}
	if !Mastered(Progress{Box: MasteredBox}) {
		t.Error("not mastered at the mastery box")
	}
	if !Mastered(Progress{Box: MasteredBox + 4}) {
		t.Error("un-mastered by climbing higher")
	}
	// Reaching MasteredBox requires surviving the box-8 gap, which is what makes
	// the bar mean something. If that gap ever shrank below a month, "mastered"
	// would start certifying words the learner last saw three weeks ago.
	if got := IntervalDays(MasteredBox - 1); got < 30 {
		t.Errorf("the last gap before mastery is %d days, want at least 30 — mastery "+
			"must require surviving a long gap, not merely arriving", got)
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
	if p.Box != 2 {
		t.Errorf("got box %d, want 2 — a lookup is not a wrong answer", p.Box)
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
	for _, tc := range []struct {
		correct, unaided bool
		grade            Grade
	}{
		{true, false, GradeCorrect},
		{true, true, GradeUnaided},
		{false, false, GradeWrong},
		// A wrong answer cannot be unaided; the log could still hold the pair,
		// and `wrong` must win rather than the flag promoting a miss.
		{false, true, GradeWrong},
	} {
		e := store.ReviewEvent{Word: "w", Kind: store.EventReviewed, Correct: tc.correct, Unaided: tc.unaided, At: at(1)}
		got := Fold([]store.ReviewEvent{e})[store.Key("w")]
		want := Answer(Progress{}, tc.grade, at(1))

		if got != want {
			t.Errorf("correct=%v unaided=%v: Fold gave %+v, Answer(%v) gave %+v",
				tc.correct, tc.unaided, got, tc.grade, want)
		}
	}
}

// The Done-when's own row: promotion, demotion and due-dates across a simulated
// multi-week schedule with an explicit clock.
func TestMultiWeekSchedule(t *testing.T) {
	word := "obsequious"
	var events []store.ReviewEvent

	// Six correct answers, each on the day the previous interval came due.
	// Intervals are 1, 1, 2, 4, 6, 10 — so the review days are the running sum.
	reviewDays := []int{0, 1, 3, 7, 13, 23}
	for _, d := range reviewDays {
		events = append(events, reviewed(word, true, at(d)))
	}

	// A due-date assertion at EACH step, which the plan promised and the first
	// version of this test did not do — it checked only the end state, so a wrong
	// interval anywhere in the middle would have passed.
	for i := range reviewDays {
		p := Fold(events[:i+1])[store.Key(word)]
		wantBox := i + 1
		if p.Box != wantBox {
			t.Fatalf("after %d correct answers box = %d, want %d", i+1, p.Box, wantBox)
		}
		iv := IntervalDays(p.Box)
		reviewedOn := reviewDays[i]
		if Due(p, at(reviewedOn+iv-1)) {
			t.Errorf("box %d: due %d days after review, want not due until %d", p.Box, iv-1, iv)
		}
		if !Due(p, at(reviewedOn+iv)) {
			t.Errorf("box %d: not due %d days after review", p.Box, iv)
		}
	}

	p := Fold(events)[store.Key(word)]

	if p.Box != len(reviewDays) {
		t.Fatalf("after %d correct answers box = %d, want %d", len(reviewDays), p.Box, len(reviewDays))
	}
	if p.MaxBox != p.Box {
		t.Errorf("MaxBox = %d after a clean climb to box %d; they must agree", p.MaxBox, p.Box)
	}
	// At box 6 the interval is 16 days: not due the next day, due on day 16.
	if Due(p, at(24)) {
		t.Error("due one day after reaching box 6")
	}
	if !Due(p, at(23+16)) {
		t.Error("not due 16 days after the last review")
	}

	// One miss: the box HALVES, and the word is due sooner because the interval
	// shortened — which is the entire point of the demotion.
	before := p.Box
	events = append(events, reviewed(word, false, at(145)))
	p = Fold(events)[store.Key(word)]

	if p.Box != before/2 {
		t.Errorf("after a miss box = %d, want %d (half of %d)", p.Box, before/2, before)
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

		if p.Box < 0 || p.Box > ladderLimit {
			t.Fatalf("box %d outside the ladder after %d events", p.Box, n)
		}
		if p.MaxBox < p.Box {
			t.Fatalf("MaxBox %d is below Box %d after %d events", p.MaxBox, p.Box, n)
		}
		// Idempotent over a re-fold of the same slice: folding is a pure
		// function of its input, so calling it twice cannot differ.
		if again := Fold(events)[store.Key("w")]; again != p {
			t.Fatalf("Fold is not idempotent: %+v then %+v", p, again)
		}
	})
}

// The consequence of the mixed-zone bug, pinned where a learner would feel it.
//
// A stored event stamp carries a FIXED offset (yaml.v3 parses it that way) while
// `now` comes from the system clock in time.Local. Before store.DaysBetween
// normalised, those two on the same local day counted as one day apart — so a
// box-0 word reviewed this morning was offered again this afternoon, for half the
// year. No existing test crossed zones, which is why it was green.
func TestDueDoesNotFireOnTheDayOfReview(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		// NOT a skip: time/tzdata is embedded above, so a failure here means
		// something is genuinely wrong rather than absent.
		t.Fatalf("loading the zone: %v", err)
	}
	// The stamp's OWN date must differ from its date in the learner's zone, or
	// the two readings agree and the test cannot tell them apart — my first
	// version used a same-date pair and stayed green under the mutation.
	//
	// This is a review recorded while travelling: 22:00 on Aug 3 at UTC-10, which
	// is 01:00 on Aug 4 where the learner now is. The store keeps the offset it
	// was written with, so this is what a real stamp looks like.
	reviewedAt := time.Date(2026, 8, 3, 22, 0, 0, 0, time.FixedZone("HST", -10*3600))
	p := Progress{Box: 0, LastReviewed: reviewedAt} // box 0 waits 1 day

	// Later the SAME local morning, in the learner's calendar.
	sameLocalDay := time.Date(2026, 8, 4, 9, 0, 0, 0, la)
	if Due(p, sameLocalDay) {
		t.Error("due on the same local day it was reviewed — the schedule counted a day that did not pass")
	}

	nextDay := time.Date(2026, 8, 5, 9, 0, 0, 0, la)
	if !Due(p, nextDay) {
		t.Error("not due the next calendar day")
	}
}

// RE-FOLDING AN EXISTING LOG IS SAFE: no migration runs, so every learner's
// boxes are silently re-derived the first time they open a sitting after this
// ships. The direction of that change is what matters.
//
// Under the old ladder (1, 3, 7, 14, 30, 90, clamped at box 5) a word with N
// correct answers sat at box min(N,5); under this one it sits at box N. Below
// box 10 the new interval is SHORTER, so words become due sooner — more review,
// never less, and nothing is silently deferred. A learner with a mature deck
// will meet a large first sitting, which the README warns about; what they will
// not meet is a word quietly disappearing for a year.
func TestReFoldingAnOldLogIsSafe(t *testing.T) {
	oldLadder := []int{1, 3, 7, 14, 30, 90}
	oldInterval := func(n int) int {
		if n > len(oldLadder)-1 {
			n = len(oldLadder) - 1
		}
		return oldLadder[n]
	}

	for n := 1; n <= 9; n++ {
		var events []store.ReviewEvent
		for i := 0; i < n; i++ {
			events = append(events, reviewed("w", true, at(i)))
		}
		p := Fold(events)[store.Key("w")]

		if p.Box != n {
			t.Fatalf("%d corrects gave box %d, want %d", n, p.Box, n)
		}
		now, was := IntervalDays(p.Box), oldInterval(n)
		if now > was {
			t.Errorf("%d correct answers: the interval moved from %d days to %d — re-folding "+
				"an existing log DEFERRED this word, so a learner loses reviews they had "+
				"already earned", n, was, now)
		}
	}

	// And the direction reverses above box 10, which is the intended trade: a
	// word recalled that many times has earned a longer wait than the old
	// ladder's cap could express.
	if IntervalDays(10) <= 90 {
		t.Errorf("box 10 is %d days, want more than the old 90-day cap — the unbounded "+
			"ladder buys nothing if it never exceeds the ceiling it replaced", IntervalDays(10))
	}
}

// THE FORM IS TELEMETRY AND THE SCHEDULER MUST NOT READ IT (#40 D4a).
//
// The field exists so a later query can ask whether board-promoted words lapse
// more than the ones a real retrieval test promoted. The moment the ladder
// branched on it, it would stop being an observation and start being a rule —
// and the deferred remedies it is meant to inform would already be half-chosen.
func TestFoldIgnoresTheFormThatAsked(t *testing.T) {
	at := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	base := []store.ReviewEvent{
		{Word: "keel", Kind: store.EventReviewed, Correct: true, At: at},
		{Word: "keel", Kind: store.EventReviewed, Correct: true, Unaided: true, At: at.Add(24 * time.Hour)},
		{Word: "mesa", Kind: store.EventReviewed, Correct: false, At: at},
	}
	want := Fold(base)

	for _, form := range []string{"recall", "meaning", "board", ""} {
		tagged := make([]store.ReviewEvent, len(base))
		copy(tagged, base)
		for i := range tagged {
			tagged[i].Form = form
		}
		got := Fold(tagged)
		if len(got) != len(want) {
			t.Fatalf("form %q folded to %d words, want %d", form, len(got), len(want))
		}
		for k, p := range want {
			if got[k] != p {
				t.Errorf("form %q changed %q: %+v, want %+v", form, k, got[k], p)
			}
		}
	}
}
