//go:build conformance

package main

// Live conformance for Google News RSS (ARCH-MOCK, applied to the feed).
//
// The fake models a shape; this asserts the shape is still what the service
// emits. Cadence is on-demand with the rest of the conformance suite — it needs
// network access, which does not belong in merge-check.yml.
//
//	go test -tags conformance -run News ./cmd/define/
//
// It PRINTS what it fetched, which is deliberate: with no user-facing consumer
// until #10, this is the way a person looks at real output.

import (
	"github.com/xianxu/tools/internal/conformance"
	"strings"
	"testing"
)

// coverageFloor is asserted rather than an exact count: the feed is live, and a
// test that pins a number fails on a slow news week rather than on a broken
// contract. The Spec measured 12 to 99 matching headlines per word; a floor well
// under that says "still working" without saying "still identical".
const coverageFloor = 3

func TestNewsFeedStillParsesAndCovers(t *testing.T) {
	// A common word, so a thin week for one topic does not read as a failure.
	const word = "ephemeral"

	body, err := newHTTPFeed().Fetch(t.Context(), word)
	if err != nil {
		// An unreachable feed is an ABSENT DEPENDENCY, not drift — the two are
		// distinguished here and nowhere below, because everything past this
		// point is a statement about the feed's SHAPE and stays a hard failure.
		conformance.SkipOrFail(t, "news feed unreachable", err)
	}
	if len(body) == 0 {
		t.Fatal("empty body")
	}

	items, err := parseRSS(body)
	if err != nil {
		t.Fatalf("the live feed no longer parses: %v — the shape changed", err)
	}
	if len(items) == 0 {
		t.Fatal("the live feed parsed to zero items — the shape changed")
	}

	usages := usagesFrom(items, word)
	t.Logf("%d items, %d usages for %q", len(items), len(usages), word)
	for i, u := range usages {
		if i == 5 {
			t.Logf("  ... and %d more", len(usages)-5)
			break
		}
		t.Logf("  %s  [%s]", u.Text, u.Publisher)
	}
	if len(usages) < coverageFloor {
		t.Errorf("only %d usages, want at least %d — either the feed thinned out or the filter broke",
			len(usages), coverageFloor)
	}

	// The attribution rule, against real titles rather than a fixture: no usage
	// should still be carrying its publisher.
	for _, u := range usages {
		if u.Publisher != "" && strings.HasSuffix(u.Text, " - "+u.Publisher) {
			t.Errorf("usage still carries its attribution: %q", u.Text)
		}
	}

	// The terms the Spec records are in the feed body itself; assert they are
	// still what we think we are agreeing to.
	if !strings.Contains(string(body), "personal, non-commercial use") {
		t.Error("the feed's usage terms no longer say personal, non-commercial use — re-read them")
	}
}
