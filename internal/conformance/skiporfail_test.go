package conformance_test

import (
	"errors"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
)

// The inversion this package exists for, asserted in both directions.
//
// It shipped unpinned: the behaviour was measured by running whole suites and
// reading their output, which is why a helper elsewhere could route through it
// and silently change what its own unit test asserted — that test passed under
// `go test` and failed under `CONFORMANCE_STRICT=1`, against unchanged code.
// A rule with two modes needs both modes pinned at the rule, not at its callers.
func TestSkipOrFailBothDirections(t *testing.T) {
	boom := errors.New("no terminal")

	for _, tc := range []struct {
		name, strict string
		err          error
		wantSkip     bool
	}{
		{"default skips, so an offline developer still gets a useful run", "", boom, true},
		{"strict fails, so green cannot mean 'did not run'", "1", boom, false},
		{"a nil err still skips by default", "", nil, true},
		{"a nil err still fails under strict", "1", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(conformance.StrictEnv, tc.strict)

			// SkipOrFail ends its goroutine via runtime.Goexit, so it is called
			// on a fake T in one — the same shape the llmtest helper's own test
			// uses.
			fake := &testing.T{}
			done := make(chan struct{})
			go func() {
				defer close(done)
				conformance.SkipOrFail(fake, "no pty available", tc.err)
			}()
			<-done

			wantFail := !tc.wantSkip // the two outcomes are exclusive and exhaustive
			if got := fake.Skipped(); got != tc.wantSkip {
				t.Errorf("Skipped() = %v, want %v", got, tc.wantSkip)
			}
			if got := fake.Failed(); got != wantFail {
				t.Errorf("Failed() = %v, want %v", got, wantFail)
			}
		})
	}
}

// Strict() is SET vs UNSET — an empty value is off, and every other value,
// INCLUDING "0" and "false", is on.
//
// The name used to advertise only the empty case while the table already pinned
// the surprising half (BR-12): `CONFORMANCE_STRICT=0` turns strict ON. That is
// the ordinary shell convention for a flag variable, but it is exactly the thing
// someone gets wrong when they mean to switch the mode off — so it is named
// here, and documented on StrictEnv where a caller reads it.
func TestStrictIsSetVsUnsetSoEvenZeroTurnsItOn(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{{"", false}, {"1", true}, {"0", true}, {"anything", true}} {
		t.Setenv(conformance.StrictEnv, tc.value)
		if got := conformance.Strict(); got != tc.want {
			t.Errorf("Strict() with %q = %v, want %v", tc.value, got, tc.want)
		}
	}
}
