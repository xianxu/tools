package llmtest

import (
	"testing"

	"github.com/xianxu/tools/internal/conformance"
)

// substituteT runs fn against a throwaway *testing.T on its own goroutine and
// returns it, with CONFORMANCE_STRICT set to strict.
//
// SCOPE: t.Setenv binds to the CALLING test and is restored at that test's
// cleanup — not when fn returns. Two calls in one test therefore leave the last
// mode in force for the remainder, which is fine for the callers here (each
// asserts one mode, or passes "" twice) but is not "for the duration of fn", as
// an earlier version of this comment claimed (BR-8). A test needing two modes
// wants two subtests, so each gets its own cleanup.
//
// Two things are bundled here deliberately.
//
// The GOROUTINE, because the helpers under test end a test through t.Fatalf or
// t.Skip, and both call runtime.Goexit — which would terminate THIS test rather
// than the substitute's. Goexit still runs defers, so the channel closes either
// way.
//
// The ENV, because a test that reaches a conformance-routed helper must state
// which mode it is asserting. Inheriting it from the ambient environment is the
// exact defect this issue exists to fix — and it recurred INSIDE the fix: two of
// the three sites in reachable_test.go were given a t.Setenv by hand and the
// third was missed, so the Done-when's universal claim was false as written
// (BR-1). A parameter cannot be forgotten; a convention can.
//
// It lives in package llmtest, not in a shared test-support package. The plan
// first recorded the opposite — that a helper would force golden_test.go to
// import internal/conformance — and that premise was simply wrong (BR-2):
// golden_test.go and reachable_test.go are the same package, so an unexported
// helper covers both with no new package and no coupling. Only
// internal/conformance's own test is genuinely cross-package, and it keeps the
// idiom inline rather than exporting a test helper from a production package.
func substituteT(t *testing.T, strict string, fn func(*testing.T)) *testing.T {
	t.Helper()
	t.Setenv(conformance.StrictEnv, strict)

	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(fake)
	}()
	<-done
	return fake
}
