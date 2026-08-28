// Package conformance owns ONE decision: what a live check does when the thing
// it checks against is not there.
//
// A skip reads as green. A conformance run on a machine with no network, no
// dictionary, no model, no terminal or no afplay therefore reports success for
// suites that never executed — and these suites exist because defects invisible
// to everything that WAS running (define #6's CRLF cascade) are exactly what
// they are for. Green has to mean "it ran".
//
// # The two directions are one decision
//
//	CONFORMANCE_STRICT set — an absent dependency is a FAILURE. For CI, and for
//	                         a close that has to mean something.
//	default                — it is a SKIP, so a developer offline still gets a
//	                         useful `go test ./...`.
//
// Both halves matter and each was got wrong separately: the rule was first
// applied at a single pty site while six suites skipped silently, and three
// other suites had the mirror defect — an absent dependency written as an
// unconditional Fatalf, which made the offline suite red rather than skipped.
//
// # Four classes, and a site is placed by asking it the question
//
// Never by which file it lives in; excluding a file by name is what hid the
// third instance of this.
//
//	absent EXTERNAL dependency  — skip; FAIL under strict. Routes through
//	                              SkipOrFail. (no network, no NOAD, no pty, no
//	                              afplay, no model configured)
//	absent IN-REPO artifact     — ALWAYS fail. A committed fixture or doc that is
//	                              not there is a deleted file, not a machine that
//	                              lacks something.
//	SHAPE drift                 — ALWAYS fail. A fixture that no longer
//	                              byte-matches, a feed that stopped parsing: the
//	                              very thing these suites exist to report.
//	INAPPLICABLE table row      — skip, permanently and correctly. No dependency
//	                              is involved and failing it would assert
//	                              something untrue. Marked at the site with
//	                              `conformance:inapplicable`, which is what
//	                              TestEverySkipIsRoutedOrWaived looks for.
//
// Only the first class routes here.
//
// # Why this is a package and not a helper in one directory
//
// The first version lived in cmd/define and the README claimed the guarantee for
// `./...`. Measured, `CONFORMANCE_STRICT=1 go test -tags conformance ./internal/llm/...`
// reported ok with its suites skipped — a documented command whose green meant
// nothing, which is the precise false assurance the rule exists to remove. The
// scope of the guarantee and the scope of the sweep are now the same thing, and
// TestEverySkipIsRoutedOrWaived keeps them that way by failing the build rather
// than by anyone re-reading prose.
package conformance

import (
	"fmt"
	"os"
	"testing"
)

// StrictEnv is the variable that makes a skip a failure.
//
// SET vs UNSET, not true vs false. ANY non-empty value turns strict ON —
// including "0" and "false". This is the ordinary shell convention for a flag
// variable, and it is stated here because it surprises: someone writing
// CONFORMANCE_STRICT=0 to turn the mode OFF turns it on, and gets a red suite
// they did not ask for. The way off is to unset it, or set it empty.
//
//	CONFORMANCE_STRICT=1      strict          CONFORMANCE_STRICT=       default
//	CONFORMANCE_STRICT=0      strict (!)      (unset)                   default
//	CONFORMANCE_STRICT=false  strict (!)
//
// Parsing the value instead would be worse: it invites "true"/"yes"/"on" and a
// table of spellings, and makes a typo silently mean OFF in the mode whose whole
// purpose is that green means something.
const StrictEnv = "CONFORMANCE_STRICT"

// Strict reports whether a missing dependency should fail rather than skip.
//
// See StrictEnv: set-vs-unset, so any non-empty value is on.
func Strict() bool { return os.Getenv(StrictEnv) != "" }

// SkipOrFail handles an absent EXTERNAL dependency: skip, or fail under strict.
//
// Pass a nil err when the absence has no error to report ("master is not a
// terminal on this platform").
func SkipOrFail(t *testing.T, reason string, err error) {
	t.Helper()
	if Strict() {
		t.Fatal(message(reason, err, true))
	}
	t.Skip(message(reason, err, false))
}

// message is the text a reader of a red CI log actually sees, split out so it
// can be ASSERTED.
//
// It was inline, and the test that claimed to pin it could not: a substitute
// *testing.T records that it failed but exposes no reader for the text, so the
// assertion degenerated into comparing StrictEnv with its own literal — true by
// construction, checking nothing (PQ-1). Same vacuous shape #24's PQ-6 caught
// one issue earlier: a check whose subject is unreachable passes for the wrong
// reason.
//
// Naming the variable in the strict message is the load-bearing part. Someone
// reading a failure needs to know that the DEPENDENCY was missing, not the code
// broken, and which mode turned that into a failure.
func message(reason string, err error, strict bool) string {
	if err != nil {
		reason = fmt.Sprintf("%s: %v", reason, err)
	}
	if strict {
		return fmt.Sprintf("%s (%s is set)", reason, StrictEnv)
	}
	return reason
}
