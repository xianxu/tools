package conformance

import (
	"errors"
	"testing"
)

// The text a reader of a red log sees, asserted directly.
//
// This exists because the first attempt asserted it through a substitute
// *testing.T, which records THAT a test failed and not WHY — so the check
// collapsed into comparing StrictEnv with its own literal and could not fail
// (PQ-1). Splitting the message out is what makes the claim checkable at all.
func TestMessage(t *testing.T) {
	boom := errors.New("dial tcp: refused")

	for _, tc := range []struct {
		name   string
		reason string
		err    error
		strict bool
		want   string
	}{
		{
			name: "default, with a cause", reason: "network unavailable", err: boom, strict: false,
			want: "network unavailable: dial tcp: refused",
		},
		{
			name:   "strict names the variable, so the reader knows which mode failed them",
			reason: "network unavailable", err: boom, strict: true,
			want: "network unavailable: dial tcp: refused (CONFORMANCE_STRICT is set)",
		},
		{
			name: "no cause to report, default", reason: "master is not a terminal", err: nil, strict: false,
			want: "master is not a terminal",
		},
		{
			name: "no cause to report, strict", reason: "master is not a terminal", err: nil, strict: true,
			want: "master is not a terminal (CONFORMANCE_STRICT is set)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := message(tc.reason, tc.err, tc.strict); got != tc.want {
				t.Errorf("message() = %q, want %q", got, tc.want)
			}
		})
	}
}
