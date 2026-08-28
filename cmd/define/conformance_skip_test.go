//go:build conformance

package main

import (
	"os"
	"testing"
)

// skipOrFail is how EVERY conformance suite reacts to a missing dependency.
//
// A skip reads as green. A conformance run on a machine with no network, no
// model, no terminal or no afplay therefore reports success for suites that
// never executed — and these suites exist because #6's CRLF defect was invisible
// to everything that WAS running. Green has to mean "it ran".
//
// With DEFINE_CONFORMANCE_STRICT set — CI, or a close that must mean something —
// the missing dependency is a FAILURE. Without it, a developer on a plane still
// gets a useful `go test ./...`.
//
// ONE body, because the first version of this rule was applied at a single site
// while lessons.md wrote the general form, and six other suites kept skipping
// silently (BR-9).
//
// ENUMERATE BY READING EACH SUITE'S DEPENDENCY PROBE, not by grepping one call
// shape. `grep -rn 't\.Skipf\?(' cmd/define/*_test.go` is what BR-9 prescribed,
// and it found the seven skips while missing three sites with the MIRROR bug —
// dict_conformance, news_conformance and live_property wrote an absent
// dependency as an unconditional t.Fatalf/t.Errorf, so the NON-strict suite
// could never be green offline. Same rule, opposite failure, invisible to a grep
// for the word "Skip". The question to ask of each suite is "what does it do when
// its dependency is missing?", and only reading answers that.
//
// The two directions are one decision, which is why they are one function:
//
//	strict   — a dependency that is not there is a FAILURE. Green means it ran.
//	default  — it is a SKIP. A developer offline still gets a useful `go test`.
//
// What must NOT route here is a check about the dependency's SHAPE: a fixture
// that no longer byte-matches, a feed that stopped parsing, a renderer losing
// content. Those are the drift these suites exist to report, and they stay hard
// failures in both modes. The untagged skips in askrun_test.go and
// render_test.go stay out for the mirror reason — table rows that do not apply
// are not absent dependencies, and failing them would assert something untrue.
func skipOrFail(t *testing.T, reason string, err error) {
	t.Helper()
	if os.Getenv("DEFINE_CONFORMANCE_STRICT") != "" {
		if err != nil {
			t.Fatalf("%s (DEFINE_CONFORMANCE_STRICT is set): %v", reason, err)
		}
		t.Fatalf("%s (DEFINE_CONFORMANCE_STRICT is set)", reason)
	}
	if err != nil {
		t.Skipf("%s: %v", reason, err)
	}
	t.Skip(reason)
}
