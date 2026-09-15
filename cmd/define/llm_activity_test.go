package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func awaitActivity(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("activity boundary not reached")
	}
}

func TestLLMActivityStreamClearsBeforeText(t *testing.T) {
	f := llmtest.NewFake(t)
	before, release := make(chan struct{}), make(chan struct{})
	after, finish := make(chan struct{}), make(chan struct{})
	f.Script("", llmtest.Reply{Capture: streamCapture, TextStarted: before, TextRelease: release, AfterText: after, FinishRelease: finish})
	var out syncBuf
	client := foregroundClient(fakeClient(t, f)(llm.Config{Model: "claude-opus-5", APIKey: testKey}), &out, options{tty: true})
	delta := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = client.Stream(t.Context(), llm.Request{Prompt: "wait"}, func(s string) {
			if s != "" {
				out.Write([]byte("ANSWER"))
				select {
				case delta <- struct{}{}:
				default:
				}
			}
		})
	}()
	awaitActivity(t, before)
	if !strings.Contains(out.String(), "⠋") {
		t.Fatalf("no waiting glyph: %q", out.String())
	}
	close(release)
	awaitActivity(t, after)
	awaitActivity(t, delta)
	got := out.String()
	if !strings.Contains(got, eraseLine+"ANSWER") {
		t.Fatalf("answer not preceded by synchronous clear: %q", got)
	}
	close(finish)
	awaitActivity(t, done)
	if strings.ContainsAny(out.String()[strings.Index(out.String(), "ANSWER"):], "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
		t.Fatal("spinner reappeared after answer")
	}
}

func TestLLMActivityCompleteAndCancellation(t *testing.T) {
	for _, cancelCall := range []bool{false, true} {
		t.Run(map[bool]string{false: "return", true: "cancel"}[cancelCall], func(t *testing.T) {
			f := llmtest.NewFake(t)
			started, release := make(chan struct{}), make(chan struct{})
			f.Script("", llmtest.Reply{Text: "PONG", Started: started, Release: release})
			var out syncBuf
			client := foregroundClient(fakeClient(t, f)(llm.Config{Model: "claude-opus-5", APIKey: testKey}), &out, options{tty: true})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan struct{})
			go func() {
				defer close(done)
				_, _ = client.Complete(ctx, llm.Request{Prompt: "wait"})
				out.Write([]byte("RETURN"))
			}()
			awaitActivity(t, started)
			if !strings.Contains(out.String(), "⠋") {
				t.Fatal("no spinner during Complete")
			}
			if cancelCall {
				cancel()
			} else {
				close(release)
			}
			awaitActivity(t, done)
			if !strings.Contains(out.String(), eraseLine+"RETURN") {
				t.Fatalf("uncleared return: %q", out.String())
			}
		})
	}
}

func TestLLMActivityDisabledPreservesClient(t *testing.T) {
	original := llm.New(llm.Config{})
	for _, opt := range []options{{}, {tty: true, raw: true}} {
		var out syncBuf
		if foregroundClient(original, &out, opt) != original {
			t.Fatal("disabled activity changed client")
		}
		if out.String() != "" {
			t.Fatal("disabled activity wrote output")
		}
	}
}

func TestLLMActivityDiagnosticDiscovery(t *testing.T) {
	f := llmtest.NewFake(t)
	started, release := make(chan struct{}), make(chan struct{})
	f.SetCatalog([]llm.ModelInfo{{ID: "gemini-3-flash", OwnedBy: "antigravity"}})
	f.Script("PONG", llmtest.Reply{Text: "PONG", Started: started, Release: release})
	var out, errs syncBuf
	done := make(chan struct{})
	go func() {
		defer close(done)
		runLLMCheck(t.Context(), envOf(nil), discoveredClient(f), &out, &errs, options{tty: true})
	}()
	awaitActivity(t, started)
	waiting := out.String()
	close(release)
	awaitActivity(t, done)
	if !strings.Contains(waiting, "⠋") {
		t.Fatal("diagnostic inference has no spinner")
	}
	if !strings.Contains(out.String(), eraseLine+"  provider  antigravity") || !strings.Contains(out.String(), "gemini-3-flash") {
		t.Fatalf("cleanup/provenance missing: %q", out.String())
	}
}

func TestLLMActivityIncludesDiscovery(t *testing.T) {
	f := llmtest.NewFake(t)
	started, release := make(chan struct{}), make(chan struct{})
	f.ConfigureCatalog(llmtest.CatalogOptions{Started: started, Release: release})
	f.Script("", llmtest.Reply{Text: "PONG"})
	var out syncBuf
	client := foregroundClient(discoveredClient(f)(llm.Config{APIKey: testKey}), &out, options{tty: true})
	done := make(chan struct{})
	go func() { defer close(done); _, _ = client.Complete(t.Context(), llm.Request{Prompt: "wait"}) }()
	awaitActivity(t, started)
	waiting := out.String()
	close(release)
	awaitActivity(t, done)
	if !strings.Contains(waiting, "⠋") {
		t.Fatal("model discovery has no spinner")
	}
	if !strings.HasSuffix(out.String(), eraseLine) {
		t.Fatalf("discovery call did not clear: %q", out.String())
	}
}

// This client exercises callback shapes (including nil/empty) that the wire
// parser normally filters. Stateful HTTP tests above own the transport contract.
type activityDeltaClient struct {
	llm.Client
	deltas func(func(string))
}

func (c activityDeltaClient) Stream(_ context.Context, _ llm.Request, delta func(string)) (llm.Response, error) {
	c.deltas(delta)
	return llm.Response{Text: "answer"}, nil
}

func TestLLMActivityEmptyAndNilCallback(t *testing.T) {
	for _, nilCallback := range []bool{false, true} {
		var out syncBuf
		seen := []string{}
		client := foregroundClient(activityDeltaClient{deltas: func(delta func(string)) {
			delta("")
			if strings.Contains(out.String(), eraseLine) {
				t.Error("empty delta stopped spinner")
			}
			delta("answer")
			if !strings.HasSuffix(out.String(), eraseLine) {
				t.Error("answer delta did not stop spinner")
			}
		}}, &out, options{tty: true})
		var callback func(string)
		if !nilCallback {
			callback = func(s string) { seen = append(seen, s) }
		}
		resp, err := client.Stream(t.Context(), llm.Request{}, callback)
		if err != nil || resp.Text != "answer" {
			t.Fatalf("response changed: %+v %v", resp, err)
		}
		if !nilCallback && (len(seen) != 2 || seen[0] != "" || seen[1] != "answer") {
			t.Fatalf("deltas changed: %q", seen)
		}
	}
}
