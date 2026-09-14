//go:build conformance

package llm_test

import (
	"errors"
	"os"
	"testing"

	"github.com/xianxu/tools/internal/llm"
)

// Opt-in live contract check; an absent catalog is a visible skip, never evidence
// of successful inference. Ordinary tests use the stateful wire fake.
func TestConformanceAutoSelection(t *testing.T) {
	cfg, err := llm.Resolve(os.Getenv)
	if err != nil {
		t.Skipf("local model configuration unavailable: %v", err)
	}
	if !cfg.AutoModel {
		t.Skip("automatic default-local discovery is not configured")
	}
	c := llm.New(cfg)
	resp, err := c.Complete(t.Context(), llm.Request{Prompt: "Reply with PONG."})
	if errors.Is(err, llm.ErrUnavailable) {
		t.Skipf("no available live provider: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	chosen := llm.SelectionOf(c)
	if chosen.ID == "" || chosen.Provider == "" || resp.Text == "" {
		t.Fatalf("missing selection or answer: %+v", chosen)
	}
	t.Logf("verified live selection %s / %s; other providers not tested", chosen.Provider, chosen.ID)
	var streamed string
	_, err = c.Stream(t.Context(), llm.Request{Prompt: "Reply with PONG."}, func(s string) { streamed += s })
	if err != nil || streamed == "" {
		t.Fatalf("stream: %v (empty=%t)", err, streamed == "")
	}
	type pong struct {
		Answer string `json:"answer"`
	}
	got, err := llm.Run(t.Context(), c, llm.Task[pong]{Name: "discovery-conformance", Prompt: "Set answer to PONG."})
	if err != nil || got.Answer != "PONG" {
		t.Fatalf("structured answer=%q err=%v", got.Answer, err)
	}
}
