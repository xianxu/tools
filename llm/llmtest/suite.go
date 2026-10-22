package llmtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/tools/llm"
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
		}, llm.TextOnly(func(s string) { seen.WriteString(s) }))
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

	t.Run("tool-call events agree with the response", func(t *testing.T) {
		c := newClient(t)
		var events []llm.Delta
		got, err := c.Stream(ctx(t), echoRequest(7), func(d llm.Delta) { events = append(events, d) })
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if got.Stop != "tool_use" || len(got.ToolCalls) != 1 || got.ToolCalls[0].Name != "echo" {
			t.Fatalf("stop %q calls %+v, want one echo call", got.Stop, got.ToolCalls)
		}
		starts, inputs, ends := map[int]int{}, map[int]string{}, map[int]llm.ToolCall{}
		for _, d := range events {
			switch d.Kind {
			case llm.DeltaToolCallStart:
				starts[d.Index]++
			case llm.DeltaToolCallInput:
				if starts[d.Index] != 1 {
					t.Errorf("input for Index %d outside its Start/End", d.Index)
				}
				inputs[d.Index] += d.Text
			case llm.DeltaToolCallEnd:
				if _, dup := ends[d.Index]; dup {
					t.Errorf("second End for Index %d", d.Index)
				}
				ends[d.Index] = *d.Call
			}
		}
		if len(ends) != 1 {
			t.Fatalf("%d End events, want 1: %+v", len(ends), events)
		}
		for i, end := range ends {
			if inputs[i] != string(end.Input) || end.ID != got.ToolCalls[0].ID || string(end.Input) != string(got.ToolCalls[0].Input) {
				t.Errorf("Index %d: fragments %q, End %+v, response %+v", i, inputs[i], end, got.ToolCalls[0])
			}
			if n := echoN(t, end.Input); n != 7 {
				t.Errorf("echo input n = %d, want 7", n)
			}
		}
	})

	t.Run("a conversation continues from Message()", func(t *testing.T) {
		c := newClient(t)
		first, err := c.Complete(ctx(t), llm.Request{Task: "suite-turn-1", Prompt: "Name one synonym for happy."})
		if err != nil {
			t.Fatalf("first turn: %v", err)
		}
		second, err := c.Complete(ctx(t), llm.Request{Task: "suite-turn-2", Messages: []llm.Message{
			{Role: llm.RoleUser, Content: []llm.Block{llm.Text("Name one synonym for happy.")}},
			first.Message(),
			{Role: llm.RoleUser, Content: []llm.Block{llm.Text("Name one more, a different one.")}},
		}})
		if err != nil {
			t.Fatalf("second turn, carrying the first's blocks back: %v", err)
		}
		if second.Text == "" {
			t.Error("second turn answered nothing")
		}
	})

	t.Run("a cancelled stream ends in ErrCanceled or a complete success, never a late callback", func(t *testing.T) {
		c := newClient(t)
		cctx, cancel := context.WithCancel(ctx(t))
		defer cancel()
		var mu sync.Mutex
		returned, late := false, 0
		start := time.Now()
		resp, err := c.Stream(cctx, llm.Request{Task: "suite-cancel-stream", Prompt: longPrompt}, func(llm.Delta) {
			mu.Lock()
			defer mu.Unlock()
			if returned {
				late++
			}
			cancel()
		})
		mu.Lock()
		returned = true
		mu.Unlock()
		// Either outcome is legal (ARCH-ORDER): a stream already fully buffered may
		// finish before the cancel lands. What is never legal is a mixture, or a
		// different error. The strict cancel-lands case is pinned on the fake,
		// which holds the stream (TestCancelOutcomes).
		if err != nil && (!errors.Is(err, llm.ErrCanceled) || !errors.Is(err, context.Canceled)) {
			t.Fatalf("err = %v, want ErrCanceled wrapping context.Canceled, or a complete success", err)
		}
		if err == nil && resp.Stop == "" {
			t.Errorf("a success without a stop reason is not complete: %+v", resp)
		}
		if elapsed := time.Since(start); elapsed > 30*time.Second {
			t.Errorf("took %s to return after cancel", elapsed)
		}
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		if late != 0 {
			t.Errorf("%d callbacks after Stream returned", late)
		}
	})

	t.Run("concurrent streams on one client never cross, and cancel alone", func(t *testing.T) {
		c := newClient(t)
		const n, victim = ConcurrentStreams, ConcurrentVictim
		var wg sync.WaitGroup
		errs := make([]error, n)
		saw := make([][]int, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				cctx, cancel := context.WithCancel(ctx(t))
				defer cancel()
				var resp llm.Response
				resp, errs[i] = c.Stream(cctx, echoRequest(i), func(d llm.Delta) {
					if i == victim {
						cancel()
					}
					if d.Kind == llm.DeltaToolCallEnd {
						saw[i] = append(saw[i], echoN(t, d.Call.Input))
					}
				})
				if errs[i] == nil && (len(resp.ToolCalls) != 1 || echoN(t, resp.ToolCalls[0].Input) != i) {
					t.Errorf("stream %d: response calls %+v", i, resp.ToolCalls)
				}
			}(i)
		}
		wg.Wait()
		for i := 0; i < n; i++ {
			if i == victim {
				// ErrCanceled, or a complete success if the short echo stream was
				// already buffered when the cancel landed (ARCH-ORDER); the success
				// path is checked for consistency above, like every other stream.
				if errs[i] != nil && !errors.Is(errs[i], llm.ErrCanceled) {
					t.Errorf("cancelled stream %d: err = %v, want ErrCanceled or success", i, errs[i])
				}
				continue
			}
			if errs[i] != nil {
				t.Errorf("stream %d failed alongside a cancelled sibling: %v", i, errs[i])
			}
			if len(saw[i]) != 1 || saw[i][0] != i {
				t.Errorf("stream %d's callback saw echo inputs %v, want only [%d]", i, saw[i], i)
			}
		}
	})

	t.Run("an unknown model errors without panicking", func(t *testing.T) {
		c := newClient(t)
		_, err := c.Complete(ctx(t), llm.Request{
			Task: "suite-badmodel", Model: "claude-opus-6", Prompt: "hello",
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

// ConcurrentStreams and ConcurrentVictim size the concurrency obligation; a fake
// scripts an echo reply for each n (EchoPrompt), and holds the victim's.
const (
	ConcurrentStreams = 8
	ConcurrentVictim  = 3
)

// EchoPrompt is the prompt that forces an echo call with input {"n": n}; a fake
// scripts a reply on it.
func EchoPrompt(n int) string {
	return fmt.Sprintf("Call the echo tool exactly once with n set to %d. Do not answer in text.", n)
}

// EchoTool is the tool the obligation suite forces.
var EchoTool = llm.Tool{Name: "echo", Description: "Echo a number back.",
	InputSchema: map[string]any{"type": "object", "required": []any{"n"},
		"properties": map[string]any{"n": map[string]any{"type": "integer"}}}}

// echoRequest leaves ToolChoice on "auto" and asks in the prompt instead:
// Anthropic refuses a FORCED choice (any, or a named tool) alongside extended
// thinking, which these models use by default, so a forced obligation would test
// the refusal rather than the seam.
func echoRequest(n int) llm.Request {
	return llm.Request{Task: "suite-echo", Prompt: EchoPrompt(n), Tools: []llm.Tool{EchoTool}, ToolChoice: "auto"}
}

func echoN(t *testing.T, in json.RawMessage) int {
	t.Helper()
	var v struct {
		N *int `json:"n"`
	}
	if err := json.Unmarshal(in, &v); err != nil || v.N == nil {
		t.Errorf("echo input %s has no integer n: %v", in, err)
		return -1
	}
	return *v.N
}

// LongPrompt asks for an answer long enough that a cancel on its first event
// always lands before the stream ends.
const LongPrompt = "Write ten numbered sentences about rivers, one per line."

var longPrompt = LongPrompt
