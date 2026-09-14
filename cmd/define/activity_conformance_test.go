//go:build darwin && conformance

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm/llmtest"
)

// The real raw terminal must animate while the wire is silent, remove its
// transient row before answer text, and remain usable after cancellation.
// startDefineWithEnv routes missing PTY dependencies through SkipOrFail.
func TestPTYActivityWaitingAndCleanup(t *testing.T) {
	for _, cancelWaiting := range []bool{false, true} {
		name := "response"
		if cancelWaiting {
			name = "cancel"
		}
		t.Run(name, func(t *testing.T) {
			fake := llmtest.NewFake(t)
			started := make(chan struct{}, 1)
			release := make(chan struct{})
			afterText := make(chan struct{}, 1)
			finish := make(chan struct{})
			fake.Script("", llmtest.Reply{Capture: "stream-sample.sse", Started: started, Release: release, AfterText: afterText, FinishRelease: finish})
			_, f := startDefineWithEnv(t, []string{
				"DEFINE_LLM_BASE_URL=" + fake.URL,
				"DEFINE_LLM_API_KEY=pty-conformance",
				"DEFINE_LLM_MODEL=claude-opus-5",
				"DEFINE_LLM_PROVIDER=anthropic",
			}, "--no-audio")
			out := watch(f)
			awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "Create one here?") })
			if _, err := f.WriteString("n\n"); err != nil {
				t.Fatal(err)
			}
			awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "› ") })
			if _, err := f.WriteString("?why\r"); err != nil {
				t.Fatal(err)
			}
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("model request did not arrive")
			}
			waiting := awaitActivityPTY(t, out, func(s string) bool {
				seen := 0
				for _, r := range "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏" {
					if strings.ContainsRune(s, r) {
						seen++
					}
				}
				return seen >= 2
			})
			if strings.Contains(waiting, "Obsequious") {
				t.Fatal("answer escaped response barrier")
			}
			if cancelWaiting {
				if _, err := f.WriteString("\x03"); err != nil {
					t.Fatal(err)
				}
			} else {
				close(release)
				select {
				case <-afterText:
				case <-time.After(5 * time.Second):
					t.Fatal("first answer not flushed")
				}
				answer := awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "Obsequious") })
				// At the first answer repaint the activity row must already be absent,
				// even though the upstream stream is still open behind FinishRelease.
				if frame := lastFrame(answer); strings.ContainsAny(frame, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
					t.Fatalf("spinner survives first answer: %q", frame)
				}
				if late := out.take(250 * time.Millisecond); strings.ContainsAny(late, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
					t.Fatalf("spinner tick after answer: %q", late)
				}
				close(finish)
			}
			cleaned := awaitActivityPTY(t, out, func(s string) bool {
				return strings.Contains(lastFrame(s), "› ") && !strings.ContainsAny(lastFrame(s), "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
			})
			if strings.ContainsAny(lastFrame(cleaned), "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
				t.Fatal("spinner remained at prompt")
			}
			if _, err := f.WriteString("still usable"); err != nil {
				t.Fatal(err)
			}
			awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(unstyled(s), "› still usable") })
		})
	}
}

func awaitActivityPTY(t *testing.T, out *ptyOut, ready func(string) bool) string {
	t.Helper()
	var collected strings.Builder
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		collected.WriteString(out.take(20 * time.Millisecond))
		if ready(collected.String()) {
			return collected.String()
		}
	}
	t.Fatalf("terminal did not reach expected state: %q", collected.String())
	return ""
}
