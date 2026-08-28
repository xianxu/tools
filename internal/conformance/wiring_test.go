package conformance_test

import (
	"errors"
	"os"
	"os/exec"
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

	// The helper branch: run inside the child process, where the skip or failure
	// is genuine and `go test -v` prints its reason.
	if os.Getenv(marker) != "" {
		conformance.SkipOrFail(t, "network unavailable", errors.New("dial refused"))
		return
	}

	for _, tc := range []struct {
		name   string
		strict string
		want   string
	}{
		{
			name:   "default: the skip carries reason and cause",
			strict: "",
			want:   "network unavailable: dial refused",
		},
		{
			name:   "strict: the failure also names the variable",
			strict: "1",
			want:   "network unavailable: dial refused (CONFORMANCE_STRICT is set)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestSkipOrFailPrintsTheMessage$", "-test.v")
			cmd.Env = append(os.Environ(),
				marker+"=1",
				conformance.StrictEnv+"="+tc.strict,
			)
			out, _ := cmd.CombinedOutput() // non-zero under strict, by design

			if !strings.Contains(string(out), tc.want) {
				t.Errorf("the test binary never printed %q.\nSkipOrFail is not routing through message().\n--- transcript ---\n%s",
					tc.want, out)
			}
		})
	}
}
