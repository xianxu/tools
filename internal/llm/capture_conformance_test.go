//go:build conformance

package llm_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// Drift detection for the committed captures (ARCH-MOCK).
//
// The captures were recorded on 2026-08-22 and the fake is modelled on them, so
// the question this answers is "do they still describe the service". It asserts
// INVARIANTS, never sequences or content: the captures themselves disagree on
// block order, and the text differs every run.
//
// On drift the failure names the fix, because a conformance test that detects a
// problem without saying what to do becomes the test everyone skips.
//
//	go test -tags conformance -run CaptureDrift ./internal/llm/
func TestCaptureDriftAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(os.Getenv)
	if err != nil {
		t.Skipf("llm seam not configured: %v", err)
	}
	c := llm.New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("a non-trivial prompt still returns a thinking block", func(t *testing.T) {
		got, err := c.Complete(ctx, llm.Request{
			Task:   "drift-blocks",
			Prompt: "Which of obsequious, ephemeral, meticulous would ALSO fill the blank in: \"The board produced nothing but ______ agreement — every executive praised a plan they had privately called unworkable.\" Think it through, then answer.",
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		var kinds []string
		var sawThinking, sawText bool
		for _, b := range got.Blocks {
			kinds = append(kinds, b.Type)
			switch b.Type {
			case "thinking":
				sawThinking = true
			case "text":
				sawText = true
			default:
				t.Errorf("unknown block type %q — reconcile it against the response model "+
					"(llm.Block) before it reaches a learner", b.Type)
			}
		}
		if !sawText {
			t.Error("no text block")
		}
		if !sawThinking {
			t.Errorf("blocks = %v, no thinking block: the model may no longer think by "+
				"default, which would make message-thinking.json misleading. "+
				"Re-record: scripts/llm-probe.sh record", kinds)
		}
	})

	t.Run("output_config still survives the proxy", func(t *testing.T) {
		got, err := c.Complete(ctx, llm.Request{
			Task:   "drift-schema",
			Prompt: "Is obsequious a near-synonym of sycophantic? Answer with the schema.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"verdict": map[string]any{"type": "string"}, "reason": map[string]any{"type": "string"}},
				"required":             []string{"verdict", "reason"},
				"additionalProperties": false,
			},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(got.Text)), &out); err != nil {
			t.Fatalf("schema'd response no longer decodes: %v\nbody: %q\n"+
				"the proxy may have stopped passing output_config through; every authored "+
				"item would degrade to free-text parsing", err, got.Text)
		}
	})

	t.Run("the stream still carries thinking and signature deltas", func(t *testing.T) {
		var deltas int
		got, err := c.Stream(ctx, llm.Request{
			Task:   "drift-stream",
			Prompt: "Name one synonym for sycophantic, then explain the nuance in two sentences.",
		}, func(string) { deltas++ })
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if deltas == 0 {
			t.Error("no text deltas delivered")
		}
		var signed bool
		for _, b := range got.Blocks {
			if b.Type == "thinking" && b.Signature != "" {
				signed = true
			}
		}
		if !signed {
			t.Error("no signed thinking block in the stream: #16's byte-for-byte echo " +
				"depends on it. Re-record: scripts/llm-probe.sh record")
		}
	})

	t.Run("the injected preamble has not moved materially", func(t *testing.T) {
		got, err := c.Complete(ctx, llm.Request{Task: "drift-preamble", Prompt: "Reply with exactly: PONG"})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		p := got.Usage.PreambleTokens()
		// Measured at ~1,900 on 2026-08-22. A broad band: the point is to notice a
		// change of kind, not of a few tokens — every prompt in this repo was
		// tuned against a preamble of roughly this size.
		if p < 500 || p > 6000 {
			t.Errorf("proxy preamble = %d tokens, expected ~1,900. The upstream shape may "+
				"have changed; re-check what the proxy prepends before trusting prompt tuning.", p)
		}
		t.Logf("preamble: %d tokens (cache_creation %d + cache_read %d)",
			p, got.Usage.CacheCreationTokens, got.Usage.CacheReadTokens)
	})

	_ = llmtest.Capture // the committed artifacts this run is checking
}
