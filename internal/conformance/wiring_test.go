package conformance_test

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
)

// SkipOrFail must actually USE message() — pinned by re-execing the test binary
// and reading what `go test` printed.
//
// Splitting message() out made the TEXT assertable (that was PQ-1's fix), and
// then nothing pinned the wiring: bypassing message() inside SkipOrFail and
// formatting inline left `go test ./...` green in both modes. An extraction that
// makes a thing testable is not the same as testing it, and the seam introduced
// by the extraction is exactly where the coverage went.
//
// A substitute *testing.T cannot do this job — it records THAT a test skipped or
// failed and exposes no reader for the reason, which is the whole reason
// message() was extracted. The only place the text is observable is a real test
// binary's output, so this re-execs itself: the helper branch calls SkipOrFail
// for real, the parent reads the transcript.
func TestSkipOrFailPrintsTheMessage(t *testing.T) {
	const marker = "CONFORMANCE_WIRING_HELPER"

	// The child selector is DERIVED from this test's name, not a literal copy of
	// it. A literal silently decays on rename: the child then matches zero tests,
	// prints nothing, and the assertions below report "SkipOrFail is not routing
	// through message()" — a confident wrong cause (BR-11), the same
	// misattribution BR-9 fixed for a spawn failure.
	selector := "^" + regexp.QuoteMeta(t.Name()) + "$"

	// The helper branch: run inside the child process, where the skip or failure
	// is genuine and `go test -v` prints its reason.
	if os.Getenv(marker) != "" {
		conformance.SkipOrFail(t, "network unavailable", errors.New("dial refused"))
		return
	}

	// The strict suffix must be asserted in BOTH directions — present under
	// strict, ABSENT by default.
	//
	// Asserting only presence made the default row unable to fail for the mode it
	// names, because its want string is a PREFIX of the strict message (BR-6).
	// Measured: mutating SkipOrFail so every offline skip announces
	// "(CONFORMANCE_STRICT is set)" left both packages and `go test ./...` green
	// in both env states. That also means the row could not detect its own
	// CONFORMANCE_STRICT="" override failing to beat an inherited =1 — the
	// ambient-environment bug this whole issue is about, in the test written to
	// prove it fixed.
	strictSuffix := "(" + conformance.StrictEnv + " is set)"

	for _, tc := range []struct {
		name       string
		strict     string
		want       string
		wantSuffix bool
	}{
		{
			name:       "default: the skip carries reason and cause, and does NOT claim strict",
			strict:     "",
			want:       "network unavailable: dial refused",
			wantSuffix: false,
		},
		{
			name:       "strict: the failure also names the variable",
			strict:     "1",
			want:       "network unavailable: dial refused",
			wantSuffix: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run="+selector, "-test.v")
			cmd.Env = append(os.Environ(),
				marker+"=1",
				conformance.StrictEnv+"="+tc.strict,
			)
			// The error is kept, not discarded: a non-zero exit is EXPECTED under
			// strict, but so is a spawn failure, and swallowing both reports an
			// empty transcript as "not routing through message()" — a confident
			// wrong cause for whoever is reading CI (BR-9).
			out, runErr := cmd.CombinedOutput()
			transcript := string(out)

			// Distinguish "the child ran and printed the wrong thing" from "the
			// child never ran at all" before blaming the wiring.
			if !strings.Contains(transcript, "RUN   "+t.Name()[:strings.Index(t.Name(), "/")]) {
				t.Fatalf("the child process ran no test for selector %q (exec: %v) — "+
					"this is a harness fault, not a wiring defect.\n--- transcript ---\n%s",
					selector, runErr, transcript)
			}
			if !strings.Contains(transcript, tc.want) {
				t.Errorf("the test binary never printed %q (exec: %v).\nSkipOrFail is not routing through message().\n--- transcript ---\n%s",
					tc.want, runErr, transcript)
			}
			// BR-10: t.Helper() must attribute the skip/failure to the CALLER.
			// Deleting it left both packages and go test ./... green in both env
			// states — the location a reader needs in order to find which check
			// bailed was verified by reading, never by a test. Measured: with the
			// call, the transcript says wiring_test.go:NN; without it,
			// conformance.go:NN.
			if !strings.Contains(transcript, "wiring_test.go:") {
				t.Errorf("the message is not attributed to the caller — t.Helper() is missing from SkipOrFail.\n--- transcript ---\n%s",
					transcript)
			}
			if strings.Contains(transcript, "conformance.go:") {
				t.Errorf("the message is attributed to the helper's own file; t.Helper() is not in effect.\n--- transcript ---\n%s",
					transcript)
			}
			if got := strings.Contains(transcript, strictSuffix); got != tc.wantSuffix {
				t.Errorf("strict suffix %q present = %v, want %v (exec: %v).\n--- transcript ---\n%s",
					strictSuffix, got, tc.wantSuffix, runErr, transcript)
			}
		})
	}
}
