package llm

import (
	"errors"
	"net/http"
	"testing"
)

// The split that matters: a caller degrades on ErrUnavailable and must NOT
// degrade on ErrRequest, because ErrRequest means we sent something wrong and
// silently falling back would hide our own bug forever.
func TestClassifyStatus(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusTooManyRequests, ErrUnavailable},
		{http.StatusInternalServerError, ErrUnavailable},
		{http.StatusBadGateway, ErrUnavailable},
		{http.StatusServiceUnavailable, ErrUnavailable},
		{529, ErrUnavailable}, // Anthropic "overloaded" — observed live during this issue
		{http.StatusUnauthorized, ErrUnavailable},
		{http.StatusForbidden, ErrUnavailable},
		{http.StatusBadRequest, ErrRequest},
		{http.StatusNotFound, ErrRequest},
		{http.StatusUnprocessableEntity, ErrRequest},
	}
	for _, c := range cases {
		got := classifyStatus(c.status, errors.New("boom"))
		if !errors.Is(got, c.want) {
			t.Errorf("classifyStatus(%d) = %v, want %v", c.status, got, c.want)
		}
	}
}

// The cause must survive classification, or a transport failure becomes
// undiagnosable from the error alone.
func TestClassifyKeepsCause(t *testing.T) {
	cause := errors.New("dial tcp 127.0.0.1:8317: connection refused")
	got := classifyStatus(0, cause)
	if !errors.Is(got, ErrUnavailable) || !errors.Is(got, cause) {
		t.Errorf("got %v; want both ErrUnavailable and the cause", got)
	}
}

// A truncated answer commonly PARSES, so nothing downstream can catch it. Only
// the stop reason can, which makes this the transport's job.
func TestClassifyStop(t *testing.T) {
	cases := []struct {
		stop string
		want error // nil means "this is a completed answer"
	}{
		{"end_turn", nil},
		{"stop_sequence", nil},
		{"tool_use", nil},
		{"", nil},
		{"max_tokens", ErrTruncated},
		{"refusal", ErrRefused},
		{"something_new", ErrMalformed}, // never silently succeed on the unknown
	}
	for _, c := range cases {
		got := classifyStop(c.stop)
		if c.want == nil {
			if got != nil {
				t.Errorf("classifyStop(%q) = %v, want nil", c.stop, got)
			}
			continue
		}
		if !errors.Is(got, c.want) {
			t.Errorf("classifyStop(%q) = %v, want %v", c.stop, got, c.want)
		}
	}
}

// An unknown stop reason must name itself, or the first occurrence of a new one
// is undiagnosable.
func TestUnknownStopNamesItself(t *testing.T) {
	err := classifyStop("pause_turn")
	if err == nil || !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
	if got := err.Error(); !contains(got, "pause_turn") {
		t.Errorf("error %q does not name the stop reason it rejected", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
