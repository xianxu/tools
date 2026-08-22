package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"

	// Embedded so the DST rows below cannot silently skip on a machine without a
	// zone database. A guard that reports nothing when it cannot run certifies
	// nothing — the lesson #4 landed, applied to a test's inputs.
	_ "time/tzdata"
)

func la(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("no zone database: %v", err)
	}
	return loc
}

// "The last two days" is a LOCAL-CALENDAR question, and the event log is a set
// of UTC-NAMED files whose records carry their own offset. Every row here exists
// because one of those two facts breaks a plausible implementation.
func TestHistoryWindow(t *testing.T) {
	loc := la(t)
	at := func(s string) time.Time {
		ts, err := time.ParseInLocation("2006-01-02T15:04:05", s, loc)
		if err != nil {
			t.Fatal(err)
		}
		return ts
	}

	for _, tc := range []struct {
		name string
		now  string
		days int
		want string
	}{
		// Local midnight, not now-minus-N×24h. A window that starts at 00:30
		// silently drops everything you did before half past midnight yesterday.
		{"two days is today plus yesterday, from local midnight", "2026-08-21T00:30:00", 2, "2026-08-20T00:00:00"},
		{"one day is today only", "2026-08-21T00:30:00", 1, "2026-08-21T00:00:00"},
		{"a week", "2026-08-21T20:00:00", 7, "2026-08-15T00:00:00"},

		// Spring forward: 2026-03-08 is a 23-hour local day. Subtracting 24h from
		// midnight on the 9th lands at 01:00 on the 8th and loses an hour of it.
		{"a 23-hour local day still starts at midnight", "2026-03-09T12:00:00", 2, "2026-03-08T00:00:00"},
		// Fall back: 2026-11-01 is 25 hours.
		{"a 25-hour local day still starts at midnight", "2026-11-02T12:00:00", 2, "2026-11-01T00:00:00"},

		// Defensive: parseHistoryArgs refuses these, so this is belt-and-braces
		// for any other caller.
		{"zero clamps to one day", "2026-08-21T20:00:00", 0, "2026-08-21T00:00:00"},
		{"negative clamps to one day", "2026-08-21T20:00:00", -3, "2026-08-21T00:00:00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, want := historyWindow(at(tc.now), tc.days), at(tc.want)
			if !got.Equal(want) {
				t.Errorf("historyWindow(%s, %d) = %s, want %s", tc.now, tc.days, got, want)
			}
			// The instant must also carry the LOCAL zone, not UTC: the store
			// compares it against timestamps that kept their offsets.
			if got.Location() != want.Location() {
				t.Errorf("location = %v, want %v", got.Location(), want.Location())
			}
		})
	}
}

// The case that a filename-based filter gets wrong, which is why the store
// compares timestamps and never day-file names.
//
// A lookup at 19:50 local TODAY is written to TOMORROW's UTC-named file. A
// filter over the local day names [08-20, 08-21] never opens 08-22.yaml, so a
// lookup from ten minutes ago vanishes and /history reads "nothing today".
func TestWindowIncludesThisEveningDespiteTomorrowsFilename(t *testing.T) {
	loc := la(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, loc)
	lookup := time.Date(2026, 8, 21, 19, 50, 0, 0, loc)

	if file := lookup.UTC().Format("2006-01-02") + ".yaml"; file != "2026-08-22.yaml" {
		t.Fatalf("premise wrong: the lookup is stored in %s", file)
	}
	since := historyWindow(now, 2)
	if lookup.Before(since) {
		t.Errorf("a lookup from ten minutes ago (%s) fell outside the window starting %s", lookup, since)
	}
}

func TestParseHistoryArgs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    int
		wantErr string
	}{
		{"no argument is two days", nil, 2, ""},
		{"a bare number", []string{"7"}, 7, ""},
		{"--days N", []string{"--days", "7"}, 7, ""},
		{"--days=N", []string{"--days=7"}, 7, ""},
		{"the upper bound itself is fine", []string{"3650"}, 3650, ""},

		{"zero", []string{"0"}, 0, "0"},
		{"negative", []string{"-3"}, 0, "-3"},
		{"not a number", []string{"zzz"}, 0, "zzz"},
		// AddDate NORMALISES a year-overflowing date rather than failing, so an
		// unbounded value returns a garbage instant and prints an empty list
		// that reads as "you have no history".
		{"absurdly large is refused by name", []string{"999999999"}, 0, "3650"},
		{"--days with no value", []string{"--days"}, 0, "--days"},
		{"too many arguments", []string{"7", "8"}, 0, "8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseHistoryArgs(tc.args)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Errorf("days = %d, want %d", got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("want an error mentioning %q, got days = %d", tc.wantErr, got)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not name the operand %q", err, tc.wantErr)
			}
		})
	}
}

func ev(word string, found bool, at time.Time) store.ReviewEvent {
	return store.ReviewEvent{Word: word, Kind: store.EventLookedUp, Found: found, At: at}
}

// Membership and ordering are DIFFERENT time facts about the same word, and
// conflating them is the bug this test exists for. A word belongs in the list
// because it was queried inside the window; it sits where it does because of
// when it was FIRST ever seen — so a word you keep returning to holds its
// original position instead of churning to the top (operator, 2026-08-20).
func TestSummariseLookups(t *testing.T) {
	loc := la(t)
	day := func(d, h int) time.Time { return time.Date(2026, 8, d, h, 0, 0, 0, loc) }
	since := time.Date(2026, 8, 20, 0, 0, 0, 0, loc) // a two-day window ending the 21st

	got := summariseLookups([]store.ReviewEvent{
		// Seen long before the window, queried again inside it: it appears, but
		// ordered by the OLD sighting, so it sorts last.
		ev("perennial", true, day(1, 9)),
		ev("perennial", true, day(21, 9)),

		// Case and spacing collapse to one row via store.Key.
		ev("Sycophantic", true, day(20, 8)),
		ev("sycophantic", true, day(21, 10)),

		// Only ever seen before the window: absent.
		ev("obsolete", true, day(2, 9)),

		// A typo: recorded as history for up-arrow recall, never vocabulary.
		ev("sykophantic", false, day(21, 11)),

		// First seen inside the window, most recently of all: sorts first.
		ev("defenestrate", true, day(21, 12)),
	}, since)

	var words []string
	for _, r := range got {
		words = append(words, r.Word)
	}
	want := []string{"defenestrate", "sycophantic", "perennial"}
	if !reflect.DeepEqual(words, want) {
		t.Errorf("rows = %v, want %v", words, want)
	}
	for _, r := range got {
		switch r.Word {
		case "perennial":
			if !r.FirstAt.Equal(day(1, 9)) {
				t.Errorf("perennial FirstAt = %v, want the sighting from the 1st", r.FirstAt)
			}
			if r.Lookups != 2 {
				t.Errorf("perennial Lookups = %d, want 2", r.Lookups)
			}
		case "sycophantic":
			if !r.FirstAt.Equal(day(20, 8)) {
				t.Errorf("sycophantic FirstAt = %v, want the 20th", r.FirstAt)
			}
			if r.Lookups != 2 {
				t.Errorf("sycophantic Lookups = %d, want 2 (case-folded)", r.Lookups)
			}
		}
	}
}

// Ordering must be TOTAL, or the output flickers between runs on a tie.
func TestSummariseLookupsBreaksTiesByWord(t *testing.T) {
	loc := la(t)
	at := time.Date(2026, 8, 21, 9, 0, 0, 0, loc)
	got := summariseLookups([]store.ReviewEvent{
		ev("zebra", true, at), ev("apple", true, at), ev("mango", true, at),
	}, at.Add(-time.Hour))

	var words []string
	for _, r := range got {
		words = append(words, r.Word)
	}
	if !reflect.DeepEqual(words, []string{"apple", "mango", "zebra"}) {
		t.Errorf("tie order = %v, want alphabetical", words)
	}
}

func TestRenderHistory(t *testing.T) {
	loc := la(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, loc)
	d := func(day, h int) time.Time { return time.Date(2026, 8, day, h, 0, 0, 0, loc) }

	got := renderHistory([]historyRow{
		{Word: "defenestrate", FirstAt: d(21, 12), LastAt: d(21, 12), Lookups: 1},
		{Word: "sycophantic", FirstAt: d(20, 8), LastAt: d(21, 10), Lookups: 2},
		{Word: "perennial", FirstAt: d(1, 9), LastAt: d(21, 9), Lookups: 4},
	}, now, 0)

	want := []string{
		"  defenestrate  today",
		"  sycophantic   yesterday   2×",
		"  perennial     Aug 1       4×",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("renderHistory =\n  %q\nwant\n  %q", got, want)
	}
}

// Relative dates are computed on LOCAL CALENDAR DAYS, not elapsed hours: a
// lookup at 23:50 last night is "yesterday" at 00:10 even though it was twenty
// minutes ago, and one at 00:10 this morning is "today" at 23:50 though it was
// most of a day ago.
func TestRenderHistoryRelativeDatesAreCalendarDays(t *testing.T) {
	loc := la(t)
	for _, tc := range []struct {
		name string
		now  time.Time
		at   time.Time
		want string
	}{
		{"twenty minutes ago but yesterday", time.Date(2026, 8, 21, 0, 10, 0, 0, loc), time.Date(2026, 8, 20, 23, 50, 0, 0, loc), "yesterday"},
		{"most of a day ago but today", time.Date(2026, 8, 21, 23, 50, 0, 0, loc), time.Date(2026, 8, 21, 0, 10, 0, 0, loc), "today"},

		// Across a spring-forward, two local midnights one calendar day apart
		// are 23 HOURS, so dividing elapsed hours by 24 truncates to 0 and every
		// date reads a day too recent for a week afterwards. This is the same
		// trap historyWindow avoids with AddDate, twenty lines above.
		{"the day after a spring-forward", time.Date(2026, 3, 9, 12, 0, 0, 0, loc), time.Date(2026, 3, 8, 12, 0, 0, 0, loc), "yesterday"},
		{"two days across a spring-forward", time.Date(2026, 3, 9, 12, 0, 0, 0, loc), time.Date(2026, 3, 7, 12, 0, 0, 0, loc), "Saturday"},
		{"a week across a spring-forward is a date", time.Date(2026, 3, 9, 12, 0, 0, 0, loc), time.Date(2026, 3, 2, 12, 0, 0, 0, loc), "Mar 2"},
		// And a fall-back day is 25 hours, which truncation also gets wrong the
		// other way for the >7 boundary.
		{"the day after a fall-back", time.Date(2026, 11, 2, 12, 0, 0, 0, loc), time.Date(2026, 11, 1, 12, 0, 0, 0, loc), "yesterday"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := renderHistory([]historyRow{{Word: "w", FirstAt: tc.at, LastAt: tc.at, Lookups: 1}}, tc.now, 0)
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Errorf("renderHistory = %q, want it to say %q", got, tc.want)
			}
		})
	}
}

// End to end against store.Mem — production code behind the interface YAML
// implements (ARCH-MOCK), with a clock a test can stand still.
func historyRig(t *testing.T, now time.Time) (commandCtx, store.Store, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	st := store.NewMem()
	var out, errb bytes.Buffer
	return commandCtx{deck: st, clock: store.FixedClock(now), stdout: &out, stderr: &errb}, st, &out, &errb
}

func TestRunHistory(t *testing.T) {
	loc := la(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, loc)
	c, st, out, errb := historyRig(t, now)

	for _, e := range []store.ReviewEvent{
		ev("perennial", true, time.Date(2026, 8, 1, 9, 0, 0, 0, loc)),
		ev("perennial", true, time.Date(2026, 8, 21, 9, 0, 0, 0, loc)),
		ev("sycophantic", true, time.Date(2026, 8, 20, 8, 0, 0, 0, loc)),
		ev("sykophantic", false, time.Date(2026, 8, 21, 11, 0, 0, 0, loc)),
		// TODAY's evening, stored in TOMORROW's UTC-named day file. The reason
		// this whole feature reads timestamps and never filenames.
		ev("defenestrate", true, time.Date(2026, 8, 21, 19, 50, 0, 0, loc)),
		ev("obsolete", true, time.Date(2026, 8, 2, 9, 0, 0, 0, loc)),
	} {
		if err := st.AppendEvent(e); err != nil {
			t.Fatal(err)
		}
	}

	if code := runHistory(c, nil); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	got := out.String()
	for _, want := range []string{"defenestrate", "sycophantic", "perennial"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q from:\n%s", want, got)
		}
	}
	if strings.Contains(got, "sykophantic") {
		t.Errorf("a failed lookup entered the words-queried view:\n%s", got)
	}
	if strings.Contains(got, "obsolete") {
		t.Errorf("a word last seen outside the window appeared:\n%s", got)
	}
	if i, j := strings.Index(got, "defenestrate"), strings.Index(got, "perennial"); i > j {
		t.Errorf("ordering is not newest-first-sighting:\n%s", got)
	}
}

func TestRunHistoryWindowAndErrors(t *testing.T) {
	loc := la(t)
	now := time.Date(2026, 8, 21, 20, 0, 0, 0, loc)

	t.Run("a wider window reaches further back", func(t *testing.T) {
		c, st, out, _ := historyRig(t, now)
		_ = st.AppendEvent(ev("perennial", true, time.Date(2026, 8, 18, 9, 0, 0, 0, loc)))
		if runHistory(c, nil); strings.Contains(out.String(), "perennial") {
			t.Error("two days should not reach 2026-08-18")
		}
		out.Reset()
		if code := runHistory(c, []string{"7"}); code != 0 || !strings.Contains(out.String(), "perennial") {
			t.Errorf("--days 7 did not reach it: %q", out.String())
		}
	})

	t.Run("a bad window names the operand", func(t *testing.T) {
		c, _, _, errb := historyRig(t, now)
		if code := runHistory(c, []string{"zzz"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errb.String(), "zzz") {
			t.Errorf("stderr does not name the operand: %q", errb.String())
		}
	})

	t.Run("an empty window says so rather than printing nothing", func(t *testing.T) {
		c, _, out, _ := historyRig(t, now)
		if code := runHistory(c, nil); code != 0 {
			t.Errorf("exit = %d, want 0 — an empty deck is not an error", code)
		}
		if !strings.Contains(out.String(), "nothing looked up") {
			t.Errorf("silence instead of an explanation: %q", out.String())
		}
	})

	// BR-15's lesson applied forward: the same fact must not be stated twice.
	t.Run("the no-deck message is shared with --forget", func(t *testing.T) {
		for _, noCapture := range []bool{false, true} {
			var errb bytes.Buffer
			c := commandCtx{clock: store.FixedClock(now), stdout: &bytes.Buffer{}, stderr: &errb, noCapture: noCapture}
			if code := runHistory(c, nil); code != 1 {
				t.Errorf("exit = %d, want 1", code)
			}
			if got, want := strings.TrimSpace(errb.String()), noDeckMessage(noCapture); got != want {
				t.Errorf("message = %q, want the shared %q", got, want)
			}
		}
	})
}
