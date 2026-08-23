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
	// ErrTruncated means the answer was CUT OFF — stop_reason "max_tokens".
	//
	// Separate from ErrMalformed because a truncated structured answer commonly
	// PARSES. Measured, and the specimen is committed at
	// llmtest/testdata/message-truncated.json: a schema'd request hit the cap and
	// returned {"verdict":"yes", "reason":": Ā"} — valid JSON, both required
	// fields present, the reason cut mid-rune. A decoder accepts it and every
	// consumer would have believed it. Nothing downstream can detect this; only
	// the stop reason can, so the transport is the only place it can be caught.
	ErrTruncated = errors.New("llm: truncated response")
)

// classifyStatus maps an HTTP status (0 when the request never got one) onto the
// taxonomy, preserving the cause so errors.Is reaches it.
func classifyStatus(status int, cause error) error {
	switch {
	case status == 0: // never reached the server: DNS, refused, timeout
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	case status == http.StatusTooManyRequests, status >= 500:
		// >= 500 covers 529 "overloaded", which this issue hit live while
		// recording captures — it is a real code, not a hypothetical.
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		// A missing or wrong credential is an operator configuration state, not a
		// crash: `define` still defines words. --llm-check is where it is loud.
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	default:
		return fmt.Errorf("%w: %w", ErrRequest, cause)
	}
}

// classifyStop maps a stop_reason onto the taxonomy. The two that are not
// outcomes are the two that matter.
//
// Enumerated rather than defaulted-to-success: an unrecognised stop_reason is a
// reason to be suspicious, not to proceed. A future stop reason we have never
// seen returns ErrMalformed naming it, so it surfaces on the first occurrence
// instead of silently passing garbage to a learner.
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
