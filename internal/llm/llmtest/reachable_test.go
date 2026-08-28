package llmtest

import (
	"github.com/xianxu/tools/internal/conformance"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The distinction this helper exists to make, asserted in both directions.
//
// Untested, it silently did the wrong thing: an earlier version probed with a
// real API call and skipped on ErrUnavailable, so a renamed model — which this
// proxy answers with 502 — would have SKIPPED the suite instead of failing it.

func TestSkipsOnlyWhenNothingIsListening(t *testing.T) {
	// Neutralize CONFORMANCE_STRICT, because it deliberately INVERTS what this
	// test asserts: under strict an absent dependency fails rather than skips
	// (internal/conformance). Without this the helper's contract is asserted
	// against whatever the ambient environment happens to be — which is exactly
	// how a green `go test` and a red `CONFORMANCE_STRICT=1 go test` came to
	// disagree about the same unchanged code.
	t.Setenv(conformance.StrictEnv, "")

	// A closed port: nothing there, so the suite must skip.
	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		SkipIfUnreachable(fake, "http://127.0.0.1:1")
	}()
	<-done
	if !fake.Skipped() {
		t.Error("a closed port did not skip; a stopped proxy would report drift")
	}
}

func TestDoesNotSkipWhenTheServiceAnswersWithAnError(t *testing.T) {
	// A server that answers 502 for everything — what this proxy does for a
	// renamed or withdrawn model. That is DRIFT, and it must reach the suite.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"unknown provider for model x"}}`))
	}))
	defer srv.Close()

	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		SkipIfUnreachable(fake, srv.URL)
	}()
	<-done
	if fake.Skipped() {
		t.Error("skipped on a service that ANSWERED — a renamed model would be swallowed as 'unreachable'")
	}
	if fake.Failed() {
		t.Error("failed on a reachable service; the verdict belongs to the suite, not the probe")
	}
}

// ...and under strict it FAILS instead, which is the whole point of the mode.
//
// This is the only unit test of that inversion at a real call site: a stopped
// proxy must not let a conformance run report green when it is asked to mean
// "it ran".
func TestStrictTurnsAnUnreachableServiceIntoAFailure(t *testing.T) {
	t.Setenv(conformance.StrictEnv, "1")

	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		SkipIfUnreachable(fake, "http://127.0.0.1:1")
	}()
	<-done

	if fake.Skipped() {
		t.Error("a closed port skipped under CONFORMANCE_STRICT; green would mean 'did not run'")
	}
	if !fake.Failed() {
		t.Error("a closed port neither skipped nor failed under CONFORMANCE_STRICT")
	}
}
