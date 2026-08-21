package main

import (
	"strings"
	"testing"
	"time"

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
