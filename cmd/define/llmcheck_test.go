package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm"
)

type checkClient struct {
	resp llm.Response
	err  error
}

func (c checkClient) Complete(context.Context, llm.Request) (llm.Response, error) {
	return c.resp, c.err
}
func (c checkClient) Stream(context.Context, llm.Request, func(string)) (llm.Response, error) {
	return c.resp, c.err
}

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

const testKey = "sk-ant-api03-SUPERSECRETVALUE"

func TestLLMCheckReportsAHealthyConfiguration(t *testing.T) {
	var out, errOut bytes.Buffer
	code := runLLMCheck(
		envOf(map[string]string{"DEFINE_LLM_API_KEY": testKey}),
		func(llm.Config) llm.Client {
			return checkClient{resp: llm.Response{
				Text: "PONG",
				Usage: llm.Usage{
					InputTokens: 22, OutputTokens: 5, ThinkingTokens: 0,
					CacheReadTokens: 1902,
				},
			}}
		}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut.String())
	}
	for _, want := range []string{"127.0.0.1:8317", "claude-opus-5", "1902", "PONG", "ok"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output is missing %q:\n%s", want, out.String())
		}
	}
}

// The credential must never reach the output. Asserted by grepping for the key
// literal, because "we call Redact" is a claim about code and this is a claim
// about bytes.
func TestLLMCheckNeverPrintsTheKey(t *testing.T) {
	var out, errOut bytes.Buffer
	runLLMCheck(
		envOf(map[string]string{"DEFINE_LLM_API_KEY": testKey}),
		func(llm.Config) llm.Client { return checkClient{resp: llm.Response{Text: "PONG"}} },
		&out, &errOut)

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
		client llm.Client
		wantIn string
	}{
		{
			name:   "no key names both variables and where to find one",
			env:    nil,
			client: checkClient{resp: llm.Response{Text: "PONG"}},
			wantIn: "DEFINE_LLM_API_KEY",
		},
		{
			name:   "unreachable names the failure",
			env:    map[string]string{"DEFINE_LLM_API_KEY": testKey},
			client: checkClient{err: errors.New("llm: unavailable: connection refused")},
			wantIn: "connection refused",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := runLLMCheck(envOf(c.env), func(llm.Config) llm.Client { return c.client }, &out, &errOut)
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
