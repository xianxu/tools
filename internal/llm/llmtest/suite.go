package llmtest

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
)

// Suite is every obligation an llm.Client must meet, stated so that BOTH the
// fake and the live service can satisfy it.
//
// That constraint is what makes it useful: assertions are about SHAPE — text is
// non-empty, usage is non-zero, deltas concatenate to the final text, a bad
// model does not panic — never about specific content, so the same file runs in
// the normal suite against the fake and under `-tags conformance` against the
// live proxy. "The fake behaves like the real thing" becomes a test rather than
// a claim, which is the same job storetest.Suite does for Store.
//
// It has already earned its keep once: writing it surfaced that the proxy
// answers 502 for an unknown model where api.anthropic.com answers 400.
func Suite(t *testing.T, newClient func(t *testing.T) llm.Client) {
	t.Helper()

	t.Run("a completion returns text and usage", func(t *testing.T) {
		c := newClient(t)
		got, err := c.Complete(ctx(t), llm.Request{
			Task: "suite-complete", Prompt: "Name one synonym for sycophantic. Answer in one word.",
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if strings.TrimSpace(got.Text) == "" {
			t.Error("Text is empty")
		}
		if got.Usage.OutputTokens == 0 {
			t.Error("OutputTokens is 0 — usage is not being carried through")
		}
		if got.Usage.Duration == 0 {
			t.Error("Duration is 0")
		}
	})

	t.Run("every block type is one we know", func(t *testing.T) {
		c := newClient(t)
		got, err := c.Complete(ctx(t), llm.Request{
			Task: "suite-blocks", Prompt: "Which is closer to sycophantic: obsequious or ephemeral? Explain briefly.",
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if len(got.Blocks) == 0 {
			t.Fatal("no blocks preserved")
		}
		for _, b := range got.Blocks {
			switch b.Type {
			case "text", "thinking", "redacted_thinking":
			default:
				// Surfaces a new block type HERE rather than in a learner's
				// session. Deliberately not asserting an ORDER: the committed
				// captures show [thinking,text], [text] and [thinking,text,thinking].
				t.Errorf("unknown block type %q — reconcile it against the response model", b.Type)
			}
			if len(b.Raw) == 0 {
				t.Errorf("block %q has no Raw — it cannot be echoed back", b.Type)
			}
		}
	})

	t.Run("a schema'd request decodes", func(t *testing.T) {
		c := newClient(t)
		got, err := c.Complete(ctx(t), llm.Request{
			Task:   "suite-schema",
			Prompt: "Is obsequious a near-synonym of sycophantic? Answer with the schema.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"verdict": map[string]any{"type": "string"},
					"reason":  map[string]any{"type": "string"},
				},
				"required":             []string{"verdict", "reason"},
				"additionalProperties": false,
			},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		var out struct {
			Verdict string `json:"verdict"`
			Reason  string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(got.Text)), &out); err != nil {
			t.Fatalf("schema'd response does not decode: %v\nbody: %q", err, got.Text)
		}
		if out.Verdict == "" {
			t.Error("verdict field empty")
		}
	})

	t.Run("stream deltas concatenate to the final text", func(t *testing.T) {
		c := newClient(t)
		var seen strings.Builder
		got, err := c.Stream(ctx(t), llm.Request{
			Task: "suite-stream", Prompt: "Say: one two three",
		}, func(s string) { seen.WriteString(s) })
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if seen.String() != got.Text {
			t.Errorf("deltas = %q but final Text = %q — they must agree", seen.String(), got.Text)
		}
		if got.Text == "" {
			t.Error("no text streamed")
		}
	})

	t.Run("a cancelled context returns promptly", func(t *testing.T) {
		c := newClient(t)
		cctx, cancel := context.WithCancel(context.Background())
		cancel() // already dead
		start := time.Now()
		_, err := c.Complete(cctx, llm.Request{Task: "suite-cancel", Prompt: "hello"})
		if err == nil {
			t.Fatal("a cancelled context produced no error")
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("took %s to notice cancellation", elapsed)
		}
	})

	t.Run("an unknown model errors without panicking", func(t *testing.T) {
		c := newClient(t)
		_, err := c.Complete(ctx(t), llm.Request{
			Task: "suite-badmodel", Model: "claude-not-a-real-model", Prompt: "hello",
		})
		if err == nil {
			t.Fatal("an unknown model succeeded")
		}
		// NOT asserted: which taxonomy member. Measured 2026-08-22, the proxy
		// answers 502 for this, so it lands in ErrUnavailable — where a direct
		// api.anthropic.com 400 would land in ErrRequest. Asserting either would
		// make this suite pass against one backend and fail against the other,
		// which is exactly what it exists to prevent. The obligation is that a
		// typo is an error rather than a silent success or a panic.
		if !errors.Is(err, llm.ErrUnavailable) && !errors.Is(err, llm.ErrRequest) {
			t.Errorf("err = %v, want a taxonomy member", err)
		}
	})
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	return c
}
