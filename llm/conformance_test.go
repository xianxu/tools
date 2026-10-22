//go:build conformance

package llm_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/xianxu/tools/conformance"
	"github.com/xianxu/tools/llm"
	"github.com/xianxu/tools/llm/llmtest"
)

// Live conformance for the model seam (ARCH-MOCK).
//
// The SAME obligation suite that runs against the wire fake, run against the
// real service. That is what makes "the fake behaves like the real thing" a test
// rather than a claim — and it has already caught one divergence: the proxy
// answers 502 for an unknown model where api.anthropic.com answers 400, which is
// why the fake models the 502.
//
// Cadence is on-demand with the rest of the conformance suite — it needs network
// access and a configured key, neither of which belongs in merge-check.yml.
//
//	go test -tags conformance -run Conformance ./llm/
//
// The proxy is the parley-managed cli-proxy-api; its key lives in
// ~/.local/share/nvim/parley/cliproxy/config.yaml. Export it as
// DEFINE_LLM_API_KEY before running.
func TestConformanceAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(os.Getenv)
	if err != nil {
		// Skip, not fail: an unconfigured machine is not a broken one. Same shape
		// cmd/define/fetch_conformance_test.go uses for the CDN.
		conformance.SkipOrFail(t, "llm seam not configured", err)
	}
	t.Logf("conformance against %s (model %s, key %s)", cfg.BaseURL, cfg.Model, llm.Redact(cfg.APIKey))
	// A stopped proxy would otherwise fail all six obligations, reporting "the
	// fake does not behave like the real thing" when the real thing is simply off.
	llmtest.SkipIfUnreachable(t, cfg.BaseURL)
	llmtest.Suite(t, func(t *testing.T) llm.Client { return llm.New(cfg) })
}

// BeforeAttempt against the real transport: the hook sees every physical
// attempt, numbered from 0, and Usage.Attempts reports exactly the ones it
// admitted. Not part of llmtest.Suite, which receives a built client and cannot
// install a hook (tools#85).
//
//	go test -tags conformance -run BeforeAttemptAgainst ./llm/
func TestBeforeAttemptAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(os.Getenv)
	if err != nil {
		conformance.SkipOrFail(t, "llm seam not configured", err)
	}
	llmtest.SkipIfUnreachable(t, cfg.BaseURL)
	var mu sync.Mutex
	var numbers []int
	cfg.BeforeAttempt = func(_ context.Context, a llm.Attempt) error {
		mu.Lock()
		defer mu.Unlock()
		numbers = append(numbers, a.Number)
		return nil
	}
	resp, err := llm.New(cfg).Complete(t.Context(), llm.Request{Task: "conformance-attempts", Prompt: "Reply with exactly: PONG"})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(numbers) == 0 || numbers[0] != 0 || resp.Usage.Attempts != len(numbers) {
		t.Errorf("hook saw %v, Usage.Attempts = %d; want Numbers from 0 and one Attempt per call", numbers, resp.Usage.Attempts)
	}

	errShed := errors.New("conformance: shed")
	cfg.BeforeAttempt = func(context.Context, llm.Attempt) error { return errShed }
	resp, err = llm.New(cfg).Complete(t.Context(), llm.Request{Task: "conformance-shed", Prompt: "Reply with exactly: PONG"})
	if !errors.Is(err, errShed) || !errors.Is(err, llm.ErrUnavailable) || resp.Usage.Attempts != 0 {
		t.Errorf("rejected call = %v, Attempts %d; want ErrUnavailable wrapping the hook error, 0 attempts", err, resp.Usage.Attempts)
	}
}
