//go:build conformance

package llm_test

import (
	"os"
	"testing"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
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
//	go test -tags conformance -run Conformance ./internal/llm/
//
// The proxy is the parley-managed cli-proxy-api; its key lives in
// ~/.local/share/nvim/parley/cliproxy/config.yaml. Export it as
// DEFINE_LLM_API_KEY before running.
func TestConformanceAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(os.Getenv)
	if err != nil {
		// Skip, not fail: an unconfigured machine is not a broken one. Same shape
		// cmd/define/fetch_conformance_test.go uses for the CDN.
		t.Skipf("llm seam not configured: %v", err)
	}
	t.Logf("conformance against %s (model %s, key %s)", cfg.BaseURL, cfg.Model, llm.Redact(cfg.APIKey))
	llmtest.Suite(t, func(t *testing.T) llm.Client { return llm.New(cfg) })
}
