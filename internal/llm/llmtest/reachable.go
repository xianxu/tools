package llmtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
)

// SkipIfUnreachable ends the test as SKIPPED when the service is not running.
//
// "Not running" is not "changed", and every live suite must draw that line or it
// reports drift on a stopped proxy — telling the operator to re-record captures
// that are perfectly good. Resolving config only proves a key is SET.
//
// One helper rather than a check per suite: the enumeration of live suites grows
// (the obligation suite and the capture-drift suite today), and a per-site check
// is a site the next one forgets.
func SkipIfUnreachable(t *testing.T, c llm.Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := c.Complete(ctx, llm.Request{
		Task: "reachability", Prompt: "Reply with exactly: PONG", MaxTokens: 2048,
	})
	if errors.Is(err, llm.ErrUnavailable) {
		t.Skipf("service unreachable, not drifted: %v", err)
	}
	// Any other error is a real finding and belongs to the suite that follows —
	// this probe deliberately does not fail on it.
}
