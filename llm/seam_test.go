package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/tools/llm"
	"github.com/xianxu/tools/llm/llmtest"
)

// The tools#85 seam (multi-turn, tools, Delta events, cancel, admission), driven
// against the wire fake so the SDK serialises, retries and parses as it does in
// production.

var weather = llm.Tool{Name: "weather", Description: "look up weather",
	InputSchema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}}

func toolReply() llmtest.Reply {
	return llmtest.Reply{Text: "Checking both.", ToolCalls: []llm.ToolCall{
		{ID: "tu_1", Name: "weather", Input: json.RawMessage(`{"city":"Oslo — Norway"}`)},
		{ID: "tu_2", Name: "weather", Input: json.RawMessage(`{"city":"Bergen"}`)},
	}}
}

func TestToolCallsOnBothPaths(t *testing.T) {
	want := toolReply().ToolCalls
	for _, streaming := range []bool{false, true} {
		f := llmtest.NewFake(t)
		f.Script("weather", toolReply())
		c := client(t, f.URL)
		r := llm.Request{Task: "t", Prompt: "weather in Oslo and Bergen?", Tools: []llm.Tool{weather}, ToolChoice: "auto"}
		var resp llm.Response
		var err error
		var events []llm.Delta
		if streaming {
			resp, err = c.Stream(t.Context(), r, func(d llm.Delta) { events = append(events, d) })
		} else {
			resp, err = c.Complete(t.Context(), r)
		}
		if err != nil {
			t.Fatalf("streaming=%v: %v", streaming, err)
		}
		if resp.Stop != "tool_use" || !reflect.DeepEqual(resp.ToolCalls, want) {
			t.Errorf("streaming=%v: stop %q, calls %+v, want %+v", streaming, resp.Stop, resp.ToolCalls, want)
		}
		got := f.Requests()[0]
		if len(got.Tools()) != 1 || got.Tools()[0]["name"] != "weather" || got.ToolChoice()["type"] != "auto" {
			t.Errorf("tools on the wire = %v, choice %v", got.Tools(), got.ToolChoice())
		}
		if !streaming {
			continue
		}
		var ends []llm.ToolCall
		inputs := map[int]string{}
		var text strings.Builder
		for _, d := range events {
			switch d.Kind {
			case llm.DeltaText:
				if len(ends) > 0 || len(inputs) > 0 {
					t.Error("text arrived after tool input; the fake orders Text first")
				}
				text.WriteString(d.Text)
			case llm.DeltaToolCallInput:
				inputs[d.Index] += d.Text
			case llm.DeltaToolCallEnd:
				if inputs[d.Index] != string(d.Call.Input) {
					t.Errorf("Index %d fragments %q != End %q", d.Index, inputs[d.Index], d.Call.Input)
				}
				ends = append(ends, *d.Call)
			}
		}
		if text.String() != resp.Text || !reflect.DeepEqual(ends, resp.ToolCalls) {
			t.Errorf("events text %q calls %+v; response text %q calls %+v", text.String(), ends, resp.Text, resp.ToolCalls)
		}
	}
}

// A conversation continues by appending resp.Message() and the tool results: the
// thinking block goes back with its signature intact, and the fake matches the
// next reply on the tool result's content.
func TestMultiTurnRoundTrip(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("Which is closer", llmtest.Reply{Capture: "message-thinking.json"})
	f.Script("rain in Oslo", llmtest.Reply{Text: "Bring a coat."})
	c := client(t, f.URL)

	first, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "Which is closer?", Tools: []llm.Tool{weather}})
	if err != nil {
		t.Fatal(err)
	}
	history := []llm.Message{
		{Role: llm.RoleUser, Content: []llm.Block{llm.Text("Which is closer?")}},
		first.Message(),
		{Role: llm.RoleUser, Content: []llm.Block{llm.ToolResult("tu_1", "rain in Oslo", false)}},
	}
	second, err := c.Complete(t.Context(), llm.Request{Task: "t", Messages: history, Tools: []llm.Tool{weather}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Text != "Bring a coat." {
		t.Errorf("second reply %q: the tool-result turn did not match its script", second.Text)
	}
	sent := f.Requests()[1].Messages()
	if len(sent) != 3 || sent[1].Role != "assistant" || sent[2].Content[0]["type"] != "tool_result" {
		t.Fatalf("history on the wire = %+v", sent)
	}
	var thinking map[string]any
	if err := json.Unmarshal(first.Blocks[0].Raw, &thinking); err != nil {
		t.Fatal(err)
	}
	if sent[1].Content[0]["signature"] != thinking["signature"] || thinking["signature"] == "" {
		t.Errorf("thinking signature did not round-trip: sent %v", sent[1].Content[0]["signature"])
	}
}

func TestRequestRefusedBeforeSending(t *testing.T) {
	f := llmtest.NewFake(t)
	c := client(t, f.URL)
	for _, r := range []llm.Request{
		{Task: "t"},
		{Task: "t", Prompt: "x", Tools: []llm.Tool{{Name: "none"}}},
		{Task: "t", Prompt: "x", Tools: []llm.Tool{weather}, ToolChoice: "search"},
	} {
		if _, err := c.Complete(t.Context(), r); !errors.Is(err, llm.ErrRequest) {
			t.Errorf("Complete(%+v) = %v, want ErrRequest", r, err)
		}
		if _, err := c.Stream(t.Context(), r, nil); !errors.Is(err, llm.ErrRequest) {
			t.Errorf("Stream(%+v) = %v, want ErrRequest", r, err)
		}
	}
	if f.Attempts() != 0 {
		t.Errorf("%d requests reached the wire; a refused request must send nothing", f.Attempts())
	}
}

// Cancel points, each driven deterministically by a fake barrier: the outcome is
// ErrCanceled wrapping the caller's ctx error, the partial content is what the
// call had completed, and no callback runs after Stream returns.
func TestCancelOutcomes(t *testing.T) {
	type point struct {
		name      string
		reply     func(started chan struct{}) llmtest.Reply
		deadline  bool
		wantText  string
		wantCalls int
	}
	points := []point{
		{name: "held before any frame", reply: func(s chan struct{}) llmtest.Reply {
			return llmtest.Reply{Started: s, Release: make(chan struct{})}
		}},
		{name: "after the first text", wantText: "Checking both.", reply: func(s chan struct{}) llmtest.Reply {
			r := toolReply()
			r.FinishRelease = make(chan struct{}) // held after the text frame
			return r
		}},
		{name: "mid tool call after one completed", wantText: "Checking both.", wantCalls: 1, reply: func(s chan struct{}) llmtest.Reply {
			r := toolReply()
			r.Stall = true // silent inside the second call, after the first completed
			return r
		}},
		{name: "caller deadline", deadline: true, reply: func(s chan struct{}) llmtest.Reply {
			return llmtest.Reply{Started: s, Release: make(chan struct{})}
		}},
	}
	for _, p := range points {
		t.Run(p.name, func(t *testing.T) {
			f := llmtest.NewFake(t)
			started := make(chan struct{}, 1)
			f.Script("go", p.reply(started))
			c := client(t, f.URL, func(c *llm.Config) { c.StallAfter = -1 })
			ctx, cancel := context.WithCancel(t.Context())
			if p.deadline {
				ctx, cancel = context.WithTimeout(t.Context(), 300*time.Millisecond)
			}
			defer cancel()

			var mu sync.Mutex
			returned := false
			late := 0
			sawEnd := make(chan struct{}, 4)
			sawText := make(chan struct{}, 4)
			done := make(chan struct{})
			var resp llm.Response
			var err error
			go func() {
				defer close(done)
				resp, err = c.Stream(ctx, llm.Request{Task: "t", Prompt: "go"}, func(d llm.Delta) {
					mu.Lock()
					if returned {
						late++
					}
					mu.Unlock()
					switch d.Kind {
					case llm.DeltaToolCallEnd:
						sawEnd <- struct{}{}
					case llm.DeltaText:
						sawText <- struct{}{}
					}
				})
				mu.Lock()
				returned = true
				mu.Unlock()
			}()
			switch {
			case p.deadline:
			case p.wantCalls > 0:
				<-sawEnd
				time.Sleep(50 * time.Millisecond) // the fake is now silent mid-second-call
				cancel()
			case p.wantText != "":
				// The CLIENT's first text event, not the fake's flush: only then is
				// the frame certainly accumulated.
				<-sawText
				cancel()
			default:
				<-started
				cancel()
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Stream did not return promptly after cancel")
			}
			wantCtx := context.Canceled
			if p.deadline {
				wantCtx = context.DeadlineExceeded
			}
			if !errors.Is(err, llm.ErrCanceled) || !errors.Is(err, wantCtx) {
				t.Fatalf("err = %v, want ErrCanceled wrapping %v", err, wantCtx)
			}
			if errors.Is(err, llm.ErrUnavailable) || errors.Is(err, llm.ErrTruncated) {
				t.Errorf("err = %v also reads as an outage or truncation", err)
			}
			if resp.Text != p.wantText || len(resp.ToolCalls) != p.wantCalls {
				t.Errorf("partial = %q with %d calls, want %q with %d", resp.Text, len(resp.ToolCalls), p.wantText, p.wantCalls)
			}
			assertOnlyCompleteBlocks(t, resp)
			mu.Lock()
			defer mu.Unlock()
			if late != 0 {
				t.Errorf("%d callbacks ran after Stream returned", late)
			}
		})
	}
}

// Complete honours the same rule.
func TestCompleteCanceled(t *testing.T) {
	f := llmtest.NewFake(t)
	started := make(chan struct{}, 1)
	f.Script("go", llmtest.Reply{Started: started, Release: make(chan struct{})})
	ctx, cancel := context.WithCancel(t.Context())
	go func() { <-started; cancel() }()
	_, err := client(t, f.URL).Complete(ctx, llm.Request{Task: "t", Prompt: "go"})
	if !errors.Is(err, llm.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want ErrCanceled wrapping context.Canceled", err)
	}
}

// The package's own budget is not the caller's cancel: it keeps its mapping.
func TestOwnTimeoutIsNotCanceled(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("go", llmtest.Reply{Started: make(chan struct{}, 1), Release: make(chan struct{})})
	c := client(t, f.URL, func(c *llm.Config) { c.Timeout = 300 * time.Millisecond })
	_, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "go"})
	if !errors.Is(err, llm.ErrUnavailable) || errors.Is(err, llm.ErrCanceled) {
		t.Errorf("own Timeout = %v, want ErrUnavailable and not ErrCanceled", err)
	}
}

// BeforeAttempt runs once per PHYSICAL attempt; Usage.Attempts counts the ones
// it admitted; a rejection sends nothing more, is not retried, and stays
// identifiable.
func TestBeforeAttempt(t *testing.T) {
	errShed := errors.New("gateway: shed")
	cases := []struct {
		name         string
		statuses     []int
		rejectAt     int // hook rejects this Number; -1 never
		wantNumbers  []int
		wantAttempts int
		wantErr      error
	}{
		{"admitted first try", []int{0}, -1, []int{0}, 1, nil},
		{"two retries then success", []int{503, 503, 0}, -1, []int{0, 1, 2}, 3, nil},
		{"retries exhausted", []int{503}, -1, []int{0, 1, 2}, 3, llm.ErrUnavailable},
		{"rejected on the first attempt", []int{0}, 0, []int{0}, 0, errShed},
		{"rejected on the retry", []int{503, 0}, 1, []int{0, 1}, 1, errShed},
	}
	for _, tc := range cases {
		for _, streaming := range []bool{false, true} {
			t.Run(tc.name, func(t *testing.T) {
				f := llmtest.NewFake(t)
				var replies []llmtest.Reply
				for _, s := range tc.statuses {
					replies = append(replies, llmtest.Reply{Status: s})
				}
				f.Script("go", replies...)
				var mu sync.Mutex
				var numbers []int
				c := client(t, f.URL, func(c *llm.Config) {
					c.BeforeAttempt = func(_ context.Context, a llm.Attempt) error {
						mu.Lock()
						defer mu.Unlock()
						if a.Task != "t" || a.Model != "claude-opus-5" {
							t.Errorf("Attempt = %+v", a)
						}
						numbers = append(numbers, a.Number)
						if a.Number == tc.rejectAt {
							return errShed
						}
						return nil
					}
				})
				r := llm.Request{Task: "t", Prompt: "go"}
				var resp llm.Response
				var err error
				if streaming {
					resp, err = c.Stream(t.Context(), r, nil)
				} else {
					resp, err = c.Complete(t.Context(), r)
				}
				switch {
				case tc.wantErr == nil && err != nil:
					t.Fatalf("err = %v", err)
				case tc.wantErr != nil && !errors.Is(err, tc.wantErr):
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if tc.wantErr == errShed && (!errors.Is(err, llm.ErrUnavailable) || errors.Is(err, llm.ErrCanceled)) {
					t.Errorf("a rejection = %v, want ErrUnavailable wrapping the hook's error", err)
				}
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(numbers, tc.wantNumbers) {
					t.Errorf("hook saw Numbers %v, want %v", numbers, tc.wantNumbers)
				}
				if f.Attempts() != tc.wantAttempts || resp.Usage.Attempts != tc.wantAttempts {
					t.Errorf("wire attempts %d, Usage.Attempts %d, want %d", f.Attempts(), resp.Usage.Attempts, tc.wantAttempts)
				}
			})
		}
	}
}

// A hook that blocks until the caller gives up has rejected nothing: the outcome
// is the caller's cancel. (A rejection made while the call was live winning over
// a later cancel is outcome()'s order, unit-tested in attempt_test.go.)
func TestOutcomeOrder(t *testing.T) {
	f := llmtest.NewFake(t)
	ctx, cancel := context.WithCancel(t.Context())
	c := client(t, f.URL, func(c *llm.Config) {
		c.BeforeAttempt = func(ctx context.Context, _ llm.Attempt) error {
			cancel()
			<-ctx.Done()
			return ctx.Err()
		}
	})
	if _, err := c.Complete(ctx, llm.Request{Task: "t", Prompt: "go"}); !errors.Is(err, llm.ErrCanceled) {
		t.Errorf("blocking hook then cancel = %v, want ErrCanceled", err)
	}

	if f.Attempts() != 0 {
		t.Errorf("%d attempts reached the wire", f.Attempts())
	}
}

// A partial Response reports only complete state: every tool_use block in Blocks
// (and so in Message()) is one of ToolCalls with a valid object input, never the
// half-streamed call the SDK's accumulator opened at content_block_start.
func assertOnlyCompleteBlocks(t *testing.T, resp llm.Response) {
	t.Helper()
	done := map[string]bool{}
	for _, c := range resp.ToolCalls {
		done[c.ID] = true
	}
	for _, b := range resp.Message().Content {
		switch b.Type {
		case "text":
			// A text prefix is the documented salvage.
		case "tool_use":
			if b.ToolCall == nil || !done[b.ToolCall.ID] || !json.Valid(b.ToolCall.Input) {
				t.Errorf("partial Blocks carry an incomplete tool_use: %+v", b.ToolCall)
			}
		case "thinking":
			if b.Signature == "" {
				t.Errorf("partial Blocks carry a thinking block cut before its signature")
			}
		default:
			if !json.Valid(b.Raw) {
				t.Errorf("partial Blocks carry an incomplete %s block", b.Type)
			}
		}
	}
}

// The same input is malformed through Complete and through Stream: one check,
// both paths. A stream that dies on it drops the bad call from Blocks too.
func TestToolInputMustBeAnObjectOnBothPaths(t *testing.T) {
	bad := llmtest.Reply{ToolCalls: []llm.ToolCall{{ID: "tu_bad", Name: "weather", Input: json.RawMessage(`[1,2]`)}}}
	for _, streaming := range []bool{false, true} {
		f := llmtest.NewFake(t)
		f.Script("go", bad)
		c := client(t, f.URL)
		r := llm.Request{Task: "t", Prompt: "go", Tools: []llm.Tool{weather}}
		var resp llm.Response
		var err error
		if streaming {
			resp, err = c.Stream(t.Context(), r, nil)
		} else {
			resp, err = c.Complete(t.Context(), r)
		}
		if !errors.Is(err, llm.ErrMalformed) {
			t.Errorf("streaming=%v: err = %v, want ErrMalformed", streaming, err)
		}
		if streaming {
			assertOnlyCompleteBlocks(t, resp)
			if len(resp.ToolCalls) != 0 {
				t.Errorf("a malformed call was reported: %+v", resp.ToolCalls)
			}
		}
	}
}
