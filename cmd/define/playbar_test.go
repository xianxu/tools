package main

import (
	"strings"
	"testing"
)

// The bar is one formatter, shared with the sitting summary, so the two cannot
// describe the same deck differently or word the `-count` assumption two ways.
func TestSittingBar(t *testing.T) {
	for _, tc := range []struct {
		name string
		f    sittingFigures
		want []string
		not  []string
	}{
		{
			"a normal sitting",
			sittingFigures{load: 14.2, fresh: 0.9, budget: 20, done: 7, total: 18},
			[]string{"7 of 18", "14 reviews/day", "0.9 new", "20 a sitting"},
			nil,
		},
		{
			// A deck already over budget sustains nothing, and the bar must not
			// print "-3 new words" — schedule.SustainableNewWords floors at zero
			// and the bar says so plainly rather than showing a zero that reads
			// like a rounding artifact.
			"over budget",
			sittingFigures{load: 40, fresh: 0, budget: 20, done: 0, total: 20},
			[]string{"40 reviews/day", "no room for new words"},
			[]string{"0.0 new"},
		},
		{
			// A brand-new deck: everything is box 0, so the load is one per word
			// and looks alarming. It is accurate and transient, and the bar's job
			// is to report rather than to soften.
			"a young deck",
			sittingFigures{load: 10, fresh: 0.9, budget: 20, done: 0, total: 10},
			[]string{"0 of 10", "10 reviews/day"},
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := sittingBar(tc.f)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("bar = %q, want it to contain %q", got, w)
				}
			}
			for _, w := range tc.not {
				if strings.Contains(got, w) {
					t.Errorf("bar = %q, want it NOT to contain %q", got, w)
				}
			}
			if strings.Contains(got, "\n") {
				t.Errorf("bar = %q, want ONE row — Paint charges a footer its real height, "+
					"and a two-row bar would silently take a row from the buffer", got)
			}
		})
	}
}

// The bar and the summary must not word the same assumption differently, which
// is the whole reason they share a formatter.
func TestTheBarAndTheSummaryAgree(t *testing.T) {
	f := sittingFigures{load: 14.2, fresh: 0.9, budget: 20, done: 7, total: 18}
	bar := sittingBar(f)
	summary := sittingSummary(f)

	for _, shared := range []string{"reviews/day", "new words/day", "a sitting"} {
		if !strings.Contains(bar, shared) || !strings.Contains(summary, shared) {
			t.Errorf("%q appears in one of the two but not both:\nbar     %q\nsummary %q", shared, bar, summary)
		}
	}
	// The summary carries the progress differently — it is written after the
	// sitting, so "7 of 18" belongs to the live bar and "7 right, 3 wrong" to
	// the summary's own line, which finish() already prints.
	if strings.Contains(summary, "of 18") {
		t.Errorf("the summary repeats the bar's progress counter: %q", summary)
	}
}
