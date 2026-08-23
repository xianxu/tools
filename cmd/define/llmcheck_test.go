package main

import (
	"bytes"
	"context"
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
	code := runLLMCheck(
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
	runLLMCheck(envOf(map[string]string{"DEFINE_LLM_API_KEY": testKey}), fakeClient(t, f), &out, &errOut)

	both := out.String() + errOut.String()
	if strings.Contains(both, testKey) || strings.Contains(both, "SUPERSECRET") {
		t.Fatalf("the key appears in the output:\n%s", both)
	}
}

// A diagnostic must be LOUD. Every model-shaped feature in this tool degrades
// silently by design; this is the one surface where it must not.
func TestLLMCheckIsNonZeroAndSpecificWhenUnavailable(t *testing.T) {
	cases := []struct {
		name   string
		env    map[string]string
		base   string
		wantIn string
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
			code := runLLMCheck(envOf(c.env), func(cfg llm.Config) llm.Client {
				if c.base != "" {
					cfg.BaseURL = c.base
				}
				cfg.Timeout = 5 * time.Second
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
