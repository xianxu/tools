package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/schedule"
	"github.com/xianxu/tools/cmd/define/store"
)

func statsAt(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, time.UTC)
}

// AN EMPTY DECK SAYS SO RATHER THAN PRINTING ZEROES — a Done-when row, and the
// state every new learner is in. Seven figures all reading 0 says "this is
// broken" to the one person guaranteed to see it.
func TestStatsOnAnEmptyDeckSaysSoRatherThanPrintingZeros(t *testing.T) {
	lines := renderStats(schedule.Summarise(nil, nil, statsAt(2026, time.June, 10)), statsAt(2026, time.June, 10))
	joined := strings.Join(lines, "\n")
	if len(lines) == 0 {
		t.Fatal("an empty deck rendered nothing at all")
	}
	if strings.Contains(joined, " 0") {
		t.Errorf("the empty screen prints zeroes:\n%s", joined)
	}
	// And it points somewhere, because "nothing yet" without a next step is a
	// dead end.
	if !strings.Contains(joined, "define") {
		t.Errorf("the empty screen suggests no next step:\n%s", joined)
	}
}

// EVERY FIELD OF Stats REACHES THE SCREEN.
//
// Derived by REFLECTION rather than by listing the fields, which is the
// correction #12 BR-17 forced: a hand-listed extent is half a guard, and a field
// added without a line would otherwise be silently invisible.
//
// It works by giving each field a distinctive value and looking for its trace,
// so it tests the RENDERER rather than restating the layout.
func TestEveryStatsFieldIsRendered(t *testing.T) {
	now := statsAt(2026, time.June, 10)
	full := schedule.Stats{
		Known: 41, Mastered: 17, ActiveDays: 23,
		CurrentStreak: 5, LongestStreak: 12, Added: 85, AddedPerDay: 3.7,
		Accuracy: map[string]schedule.FormAccuracy{"meaning": {Attempts: 9, Correct: 7}},
		FirstDay: statsAt(2026, time.January, 3),
		LastDay:  statsAt(2026, time.June, 9),
	}
	got := strings.Join(renderStats(full, now), "\n")

	// LastDay is deliberately not on the screen: "since <first>" answers the
	// question a learner asks, and the last active day is the streak's job.
	// Named here so the omission is a decision rather than an oversight.
	const notShown = "LastDay"

	v := reflect.ValueOf(full)
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		if name == notShown {
			continue
		}
		var want string
		switch name {
		case "Known":
			want = "41"
		case "Mastered":
			want = "17"
		case "ActiveDays":
			want = "23"
		case "CurrentStreak":
			want = "5 days"
		case "LongestStreak":
			want = "longest 12"
		case "Added":
			// Not a row of its own: it GATES the rate, and its trace on screen is
			// that the rate appears at all. A deck with no recorded adds shows no
			// rate, which TestTheRateIsHiddenWhenNothingWasEverAdded pins.
			want = "words/day"
		case "AddedPerDay":
			want = "3.7"
		case "Accuracy":
			want = "meaning"
		case "FirstDay":
			want = "since"
		default:
			t.Errorf("Stats gained the field %q and this guard has no expectation for it — "+
				"add one, or add it to notShown with a reason", name)
			continue
		}
		if !strings.Contains(got, want) {
			t.Errorf("field %s does not reach the screen (looked for %q):\n%s", name, want, got)
		}
	}
}

// A HAND-EDITED FORM NAME CANNOT REACH THE TERMINAL RAW (ARCH-SECURE).
//
// ReviewEvent.Form is a bare string in a file the README invites editing, and
// this screen is the first path that prints it — #12's BR-15, one field over.
func TestAFormNameCannotCarryEscapes(t *testing.T) {
	s := schedule.Stats{
		Known: 1,
		Accuracy: map[string]schedule.FormAccuracy{
			"meaning\x1b[2J\x1b[H": {Attempts: 1, Correct: 1},
			"":                     {Attempts: 1, Correct: 0},
		},
	}
	got := strings.Join(renderStats(s, statsAt(2026, time.June, 10)), "\n")
	if strings.ContainsAny(got, "\x1b\a") {
		t.Errorf("a control rune reached the screen: %q", got)
	}
	// The row is still SHOWN — a name nobody recognises is how a learner
	// discovers a stale or hand-edited log.
	if !strings.Contains(got, "meaning") {
		t.Errorf("the row was dropped rather than neutralised:\n%s", got)
	}
	if !strings.Contains(got, "(unnamed form)") {
		t.Errorf("an empty form name left a blank column, which reads as a bug:\n%s", got)
	}
}

// THE SCREEN IS STABLE BETWEEN RUNS. Map order would reshuffle the accuracy rows
// every time and make a learner think something had changed.
func TestAccuracyRowsAreOrdered(t *testing.T) {
	s := schedule.Stats{Known: 3, Accuracy: map[string]schedule.FormAccuracy{
		"meaning": {Attempts: 1, Correct: 1},
		"board":   {Attempts: 1, Correct: 1},
		"cloze":   {Attempts: 1, Correct: 1},
	}}
	first := strings.Join(renderStats(s, statsAt(2026, time.June, 10)), "\n")
	for i := 0; i < 20; i++ {
		if again := strings.Join(renderStats(s, statsAt(2026, time.June, 10)), "\n"); again != first {
			t.Fatalf("the screen changed between runs:\n%s\n---\n%s", first, again)
		}
	}
	if strings.Index(first, "board") > strings.Index(first, "cloze") {
		t.Errorf("rows are not sorted by name:\n%s", first)
	}
}

// The longest streak is shown only when it is LONGER: "3 days (longest 3)" tells
// the learner nothing and reads as a rebuke.
func TestTheLongestStreakIsShownOnlyWhenItIsLonger(t *testing.T) {
	equal := schedule.Stats{Known: 1, CurrentStreak: 3, LongestStreak: 3}
	if got := streakPhrase(equal); strings.Contains(got, "longest") {
		t.Errorf("streakPhrase = %q; a longest equal to the current says nothing", got)
	}
	if got := streakPhrase(schedule.Stats{CurrentStreak: 1}); got != "1 day" {
		t.Errorf("streakPhrase = %q, want %q", got, "1 day")
	}
}

// --- the shell -------------------------------------------------------------

// A NIL DECK IS NOT A FAILURE, and the two causes say different things.
func TestStatsWithNoDeckExplainsWhichCause(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		noCapture  bool
	}{
		{"no directory", "no deck in this directory", false},
		{"capture off", "DEFINE_NO_CAPTURE", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			d := testDeps(t)
			d.deck = nil
			code := runStats(t.Context(), d, options{noCapture: tc.noCapture}, &out, &errb)
			if code != 0 {
				t.Errorf("exit = %d, want 0 — an empty directory is a statement, not a failure", code)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Errorf("said %q, want it to name the cause %q", out.String(), tc.want)
			}
		})
	}
}

// A DIAGNOSTIC IS NOT OUTPUT: a --stats piped to a file must not have "could not
// read the log" in the middle of its figures.
func TestStatsSendsFailuresToStderrAndExitsNonZero(t *testing.T) {
	var out, errb bytes.Buffer
	d := testDeps(t)
	d.deck = failingStore{}
	code := runStats(t.Context(), d, options{}, &out, &errb)

	if code != 1 {
		t.Errorf("exit = %d, want 1 — silently low figures are worse than a refusal", code)
	}
	if out.Len() != 0 {
		t.Errorf("a diagnostic reached stdout: %q", out.String())
	}
	if !strings.Contains(errb.String(), "define:") {
		t.Errorf("stderr said %q, want a named failure", errb.String())
	}
}

// END TO END through run(), which is what proves the mode is wired rather than
// merely written.
func TestStatsRunsThroughTheMode(t *testing.T) {
	st := store.NewMem()
	if err := st.Upsert(store.Word{Text: "sycophantic"}); err != nil {
		t.Fatal(err)
	}
	if err := st.AppendEvent(store.ReviewEvent{
		Word: "sycophantic", Kind: store.EventLookedUp, Found: true,
		At: statsAt(2026, time.June, 9),
	}); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.deck = st
	d.clock = store.FixedClock(statsAt(2026, time.June, 10))

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-stats"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "words") {
		t.Errorf("--stats printed no figures:\n%s", out.String())
	}
}

// A MODE TAKES NO WORD, the rule --reflect and --play already state.
func TestStatsRefusesAWord(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-stats", "sycophantic"}, testDeps(t), strings.NewReader(""), &out, &errb)
	if code != 2 {
		t.Errorf("exit = %d, want 2 — a mode plus a word is two commands on one line", code)
	}
	if !strings.Contains(errb.String(), "--stats") {
		t.Errorf("the refusal does not name the flag: %q", errb.String())
	}
}

// THE RATE IS HIDDEN WHEN THE LOG RECORDS NO ADDS.
//
// Found by running --stats on a deck seeded programmatically: ten words and
// "words/day 0.0", which reads as a broken figure rather than as the missing
// history it is. A hand-seeded or pre-log deck is the reachable case — the
// README invites editing that directory.
func TestTheRateIsHiddenWhenNothingWasEverAdded(t *testing.T) {
	noAdds := schedule.Stats{Known: 10, ActiveDays: 1, CurrentStreak: 1}
	if got := strings.Join(renderStats(noAdds, statsAt(2026, time.June, 10)), "\n"); strings.Contains(got, "words/day") {
		t.Errorf("a deck with no recorded adds shows a rate:\n%s", got)
	}
	// AND A SLOW LEARNER STILL SEES IT: gated on the COUNT, not the rate, so a
	// rate that rounds to 0.0 is still shown.
	slow := schedule.Stats{Known: 10, ActiveDays: 200, Added: 3, AddedPerDay: 0.015}
	if got := strings.Join(renderStats(slow, statsAt(2026, time.June, 10)), "\n"); !strings.Contains(got, "words/day") {
		t.Errorf("a slow learner's rate was hidden:\n%s", got)
	}
}
