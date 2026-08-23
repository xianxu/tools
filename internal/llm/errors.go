package llm

import (
	"errors"
	"fmt"
	"net/http"
)

// The taxonomy. The load-bearing distinction is the first two: one is a normal
// outcome a caller absorbs, the other is a defect a caller must not hide.
//
// This is deliberately the same split fetch.go draws between ErrNoAudio ("this
// word has no recording" — normal) and ErrFetchFailed ("the network is down" —
// not). Collapsing the two is what makes a broken feature look like a quiet one.
var (
	// ErrUnavailable means the model could not be reached or is not configured.
	// Every consumer degrades on this: no key, no network, rate-limited, 5xx.
	// The learner's review is never blocked on a third party.
	ErrUnavailable = errors.New("llm: unavailable")
	// ErrRequest means WE sent something the provider rejected — a bad schema, an
	// unknown model, a malformed body. Loud on purpose: degrading here would hide
	// our own bug behind the same silence as a flight-mode fallback.
	ErrRequest = errors.New("llm: bad request")
	// ErrRefused means the model declined. Distinct from ErrRequest (the request
	// was well-formed) and from ErrUnavailable (the service is fine).
	ErrRefused = errors.New("llm: refused")
	// ErrMalformed means the answer did not decode into the caller's type. The
	// caller skips this question; it does not crash the session.
	ErrMalformed = errors.New("llm: malformed response")
	// ErrTruncated means the answer was CUT OFF — "max_tokens", or a stream that
	// died mid-reply. Separate from ErrMalformed because a truncated structured
	// answer commonly PARSES: the specimen at llmtest/testdata/message-truncated.json
	// decodes cleanly with every required field present. Only the stop reason can
	// catch that, so the transport is the only place it can be caught.
	ErrTruncated = errors.New("llm: truncated response")
)

// transient mirrors the set the TRANSPORT itself retries (the SDK's shouldRetry:
// 408, 409, 429, every 5xx). Derived rather than restated — a second independent
// list is how 408 came to be retried twice and then reported as our bad request
// (ARCH-DRY: one answer to "is this worth retrying").
func transient(status int) bool {
	switch {
	case status == http.StatusRequestTimeout, // 408 — retried by the transport
		status == http.StatusConflict,        // 409 — retried by the transport
		status == http.StatusTooManyRequests, // 429
		status >= 500:
		return true
	}
	return false
}

// classifyStatus maps an HTTP status (0 when the request never got one) onto the
// taxonomy, preserving the cause so errors.Is reaches it.
func classifyStatus(status int, cause error) error {
	switch {
	case status == 0: // never reached the server: DNS, refused, timeout
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	case transient(status):
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		// A missing or wrong credential is an operator configuration state, not a
		// crash: `define` still defines words.
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	default:
		return fmt.Errorf("%w: %w", ErrRequest, cause)
	}
}

// classifyStop maps a stop_reason onto the taxonomy.
//
// Enumerated rather than defaulted-to-success: an unrecognised stop reason
// returns ErrMalformed naming itself, so a new one surfaces on first sight
// instead of passing garbage to a learner.
func classifyStop(stop string) error {
	switch stop {
	case "end_turn", "stop_sequence", "tool_use", "":
		return nil
	case "max_tokens":
		return fmt.Errorf("%w: hit max_tokens", ErrTruncated)
	case "refusal":
		return ErrRefused
	default:
		return fmt.Errorf("%w: unknown stop_reason %q", ErrMalformed, stop)
	}
}

// ErrorForStop exposes the stop-reason mapping to llmtest, so a replayed
// recording reconstructs the same taxonomy member the live call produced —
// otherwise a recorded refusal would come back as a generic failure and a
// consumer's degradation path would go untested.
func ErrorForStop(stop string) error { return classifyStop(stop) }
