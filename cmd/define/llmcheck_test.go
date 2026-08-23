package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// The wire fake, not a stubbed Client. A double that discards its llm.Request
// cannot fail for any reason related to what was asked — MaxTokens, the task name
// and the PONG prompt could all be dropped and every test here would stay green
// while the real flag 400s. llmtest.NewFake records the request, so they cannot.
func fakeClient(t *testing.T, f *llmtest.Fake) func(llm.Config) llm.Client {
	return func(cfg llm.Config) llm.Client {
		cfg.BaseURL = f.URL
		cfg.Timeout = 30 * time.Second
		return llm.New(cfg)
	}
}

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

const testKey = "sk-ant-api03-SUPERSECRETVALUE"

func TestLLMCheckReportsAHealthyConfiguration(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("PONG", llmtest.Reply{Text: "PONG"})

	var out, errOut bytes.Buffer
	code := runLLMCheck(t.Context(),
		envOf(map[string]string{"DEFINE_LLM_API_KEY": testKey}),
		fakeClient(t, f), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut.String())
	}
	for _, want := range []string{"claude-opus-5", "PONG", "ok"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q:\n%s", want, out.String())
		}
	}
	// What was ASKED is now assertable, which is the point of using the wire fake.
	reqs := f.Requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests, want 1", len(reqs))
	}
	if !strings.Contains(reqs[0].Prompt(), "PONG") {
		t.Errorf("prompt = %q, want the PONG probe", reqs[0].Prompt())
	}
	if mt, _ := reqs[0].Body["max_tokens"].(float64); mt < 1024 {
		t.Errorf("max_tokens = %v — too small for a model that thinks before answering", mt)
	}
}

// The credential must never reach the output. Asserted by grepping for the key
// literal, because "we call Redact" is a claim about code and this is a claim
// about bytes.
func TestLLMCheckNeverPrintsTheKey(t *testing.T) {
	var out, errOut bytes.Buffer
	f := llmtest.NewFake(t)
	f.Script("PONG", llmtest.Reply{Text: "PONG"})
	runLLMCheck(t.Context(), envOf(map[string]string{"DEFINE_LLM_API_KEY": testKey}), fakeClient(t, f), &out, &errOut)

	both := out.String() + errOut.String()
	if strings.Contains(both, testKey) || strings.Contains(both, "SUPERSECRET") {
		t.Fatalf("the key appears in the output:\n%s", both)
	}
}

// A diagnostic must be LOUD. Every model-shaped feature in this tool degrades
// silently by design; this is the one surface where it must not.
func TestLLMCheckIsNonZeroAndSpecificWhenUnavailable(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		base    string
		hung    bool // point at a listener that accepts and never answers
		timeout time.Duration
		wantIn  string
	}{
		{
			name:   "no key names both variables and where to find one",
			env:    nil,
			wantIn: "DEFINE_LLM_API_KEY",
		},
		{
			// A real closed port, not a fabricated error: the message a user sees
			// comes from the transport, so inventing one proves nothing about it.
			name:   "unreachable names the failure",
			env:    map[string]string{"DEFINE_LLM_API_KEY": testKey},
			base:   "http://127.0.0.1:1",
			wantIn: "unavailable",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			base := c.base
			if c.hung {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer ln.Close()
				go func() {
					for {
						conn, err := ln.Accept()
						if err != nil {
							return
						}
						_ = conn // hold it open, answer nothing
					}
				}()
				base = "http://" + ln.Addr().String()
			}
			code := runLLMCheck(t.Context(), envOf(c.env), func(cfg llm.Config) llm.Client {
				if base != "" {
					cfg.BaseURL = base
				}
				cfg.Timeout = 5 * time.Second
				if c.timeout != 0 {
					cfg.Timeout = c.timeout
				}
				return llm.New(cfg)
			}, &out, &errOut)
			if code == 0 {
				t.Errorf("exit 0 for an unusable configuration; stdout:\n%s", out.String())
			}
			if !strings.Contains(errOut.String(), c.wantIn) {
				t.Errorf("stderr = %q, want it to mention %q", errOut.String(), c.wantIn)
			}
		})
	}
}

// Driven through run(), not by calling the helper: a wiring only the loop shell
// supplies must be pinned by a test that drives that shell (lessons.md, #15).
// Deleting the dispatch in main.go leaves the tests above green.
func TestLLMCheckIsReachableFromTheFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	// No key in the process env for this test, so the check reports unavailable —
	// which is enough to prove the flag REACHED runLLMCheck, and needs no network.
	t.Setenv("DEFINE_LLM_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")

	code := run(context.Background(), []string{"-llm-check"}, deps{}, strings.NewReader(""), &out, &errOut)
	if code == 0 {
		t.Errorf("exit 0 with no key configured")
	}
	if !strings.Contains(errOut.String(), "API key") {
		t.Errorf("-llm-check did not reach runLLMCheck; stderr = %q", errOut.String())
	}
	// And it must not have looked a word up or opened a deck on the way.
	if strings.Contains(out.String(), "no dictionary entry") {
		t.Error("-llm-check fell through to the lookup path")
	}
}

// Ctrl-C must reach the call. main installs signal.NotifyContext so an interrupt
// cancels rather than kills; deriving the check's context from Background()
// discarded that, and a hung endpoint then held the terminal for the full
// five-minute timeout with Ctrl-C doing nothing — measured against a socket that
// accepts and never answers.
func TestLLMCheckHonoursCancellation(t *testing.T) {
	// A listener that accepts and never responds: the shape of a hung upstream.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c // hold it open, answer nothing
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(300 * time.Millisecond); cancel() }()

	var out, errOut bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runLLMCheck(ctx, envOf(map[string]string{"DEFINE_LLM_API_KEY": testKey}),
			func(cfg llm.Config) llm.Client {
				cfg.BaseURL = "http://" + ln.Addr().String()
				cfg.Timeout = 5 * time.Minute // deliberately long: cancellation must win
				return llm.New(cfg)
			}, &out, &errOut)
	}()

	// Bounded HERE rather than by the package timeout. With the fix reverted the
	// call blocks for the full five minutes, and a package-level timeout failure
	// is indistinguishable from an unrelated hang — so the test names its own
	// cause instead.
	var code int
	select {
	case code = <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("runLLMCheck ignored a cancelled context; Ctrl-C would be swallowed for the whole Timeout")
	}
	if code == 0 {
		t.Error("exit 0 after cancellation")
	}
	// A cancelled context is the user's own keypress, not a failure to report.
	if strings.Contains(errOut.String(), "llm check failed") {
		t.Errorf("reported an interrupt as a failure: %q", errOut.String())
	}
}

// A DEADLINE must be reported, not silenced.
//
// Its own test rather than a row in the table above, because the table's hung
// listener races two timers — the context deadline and the SDK's per-request
// timeout — and whichever wins changes the error. That made the row pass either
// way, so it could not detect the guard being widened back. Here the transport
// blocks until the CONTEXT is done, so the deadline is unambiguously the cause.
//
// The bug this pins: a guard meant to keep an interrupt quiet, written against
// the DERIVED context, also silences the deadline — and a hung proxy is exactly
// what produces a deadline, so the diagnostic exits 1 saying nothing at all.
func TestLLMCheckReportsADeadline(t *testing.T) {
	var out, errOut bytes.Buffer
	// The base URL comes from the ENV, the way an operator sets it — not from a
	// mutation inside the client factory. The first version overrode it in the
	// closure and then asserted the message named it, which production never
	// does: the message reports the RESOLVED config, and resolution had already
	// happened. A fixture shaped differently from production tests the fixture.
	// DEFINE_LLM_TIMEOUT is what makes runLLMCheck's OWN deadline reachable. An
	// earlier version set cfg.Timeout inside the client factory, which shortens
	// only the SDK's per-request timeout — runLLMCheck derives its context from
	// the RESOLVED config, so its deadline stayed at five minutes and never
	// fired. The test then passed whether the guard was right or wrong.
	code := runLLMCheck(context.Background(),
		envOf(map[string]string{
			"DEFINE_LLM_API_KEY":  testKey,
			"DEFINE_LLM_BASE_URL": "http://example.invalid",
			"DEFINE_LLM_TIMEOUT":  "200ms",
		}),
		func(cfg llm.Config) llm.Client {
			cfg.Transport = roundTripBlocker{}
			return llm.New(cfg)
		}, &out, &errOut)

	if code == 0 {
		t.Error("exit 0 on a deadline")
	}
	if !strings.Contains(errOut.String(), "timed out") {
		t.Errorf("stderr = %q, want a timeout message — a diagnostic that exits 1 "+
			"in silence is the worst possible answer for a hung proxy", errOut.String())
	}
	// And it names where it was pointed, since a wrong base URL is the likeliest
	// reason for a hang.
	if !strings.Contains(errOut.String(), "example.invalid") {
		t.Errorf("stderr = %q, want it to name the base URL", errOut.String())
	}
}

// roundTripBlocker sleeps a fixed duration, IGNORING its request context, then
// reports failure.
//
// Ignoring the context is the whole point. Waiting on `r.Context().Done()`
// returns the moment the SDK's own per-request timeout fires — which is derived
// from the same Config.Timeout — so the outer context was still a hair short of
// expiring when the error came back, and a guard checking the DERIVED context
// therefore saw nil and printed the message either way. The test could not tell
// the fix from the bug.
//
// Sleeping past both deadlines makes the outer one unambiguously expired at the
// moment of the check, which is the state the bug needs to be visible.
type roundTripBlocker struct{}

func (roundTripBlocker) RoundTrip(r *http.Request) (*http.Response, error) {
	time.Sleep(700 * time.Millisecond) // > the 300ms deadline, deliberately
	return nil, errors.New("blocked past the deadline")
}
