package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/tools/internal/llm"
)

// runLLMCheck answers "did my model configuration take", which is otherwise only
// discoverable by triggering a feature that quietly degrades.
//
// It exists because every model-shaped feature in this tool is DESIGNED to fail
// silently: no key means #12 skips its veto and #13 skips its form, by intent.
// That is right for a review session and wrong for an operator who has just
// edited a config and wants to know whether it worked. This is the one surface
// where an unavailable seam is loud.
// ctx is run's signal context, and taking it is the whole point: main installs
// signal.NotifyContext so Ctrl-C cancels rather than killing. Deriving from
// context.Background() instead discards that, and a hung endpoint then holds the
// terminal for the full Timeout with Ctrl-C doing nothing — measured at 5s+
// against a socket that accepts and never answers.
func runLLMCheck(ctx context.Context, getenv func(string) string, newClient func(llm.Config) llm.Client, stdout, stderr io.Writer) int {
	cfg, err := llm.Resolve(getenv)
	if err != nil {
		// Non-zero, and the message says what to do. A cheerful empty result is
		// the failure mode AGENTS.local.md explicitly forbids for these tools.
		fmt.Fprintf(stderr, "define: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "  base url  %s\n", cfg.BaseURL)
	fmt.Fprintf(stdout, "  model     %s (effort %s)\n", cfg.Model, cfg.Effort)
	// Redacted, always. A diagnostic that prints a credential is a diagnostic you
	// cannot paste into an issue.
	fmt.Fprintf(stdout, "  key       %s\n", llm.Redact(cfg.APIKey))

	// parent is kept so the two reasons ctx can be done stay distinguishable.
	// Checking the DERIVED ctx.Err() treats both alike — and the one it silences,
	// our own deadline, is the failure a hung endpoint actually produces, so the
	// guard that was meant to keep an interrupt quiet was silencing the loudest
	// case instead.
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	start := time.Now()
	// MaxTokens comes from the resolved Config. Hardcoding it here meant
	// --llm-check reported on a request the operator had not configured, which is
	// the opposite of a diagnostic's job.
	resp, err := newClient(cfg).Complete(ctx, llm.Request{
		Task:   "llm-check",
		Prompt: "Reply with exactly the word PONG and nothing else.",
	})
	if err != nil {
		// An interrupt is the user's own keypress, not a failure to report — the
		// same distinction playAnnounced draws for interrupted playback. A
		// DEADLINE is a failure, and a loud one: it is what a hung proxy looks
		// like, and silence there is the worst possible answer from a diagnostic.
		if parent.Err() != nil {
			return 1
		}
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintf(stderr, "define: llm check timed out after %s (DEFINE_LLM_* points at %s)\n",
				took(start), cfg.BaseURL)
			return 1
		}
		fmt.Fprintf(stderr, "define: llm check failed after %s: %v\n", took(start), err)
		return 1
	}

	fmt.Fprintf(stdout, "  latency   %s\n", took(start))
	fmt.Fprintf(stdout, "  tokens    %d in, %d out (%d thinking)\n",
		resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.ThinkingTokens)
	// The preamble is the proxy's, not ours. Surfaced because our System field is
	// ADDITIVE to a prompt we do not control, and nobody should tune a prompt
	// without knowing that.
	fmt.Fprintf(stdout, "  preamble  %d tokens injected upstream (not ours)\n", resp.Usage.PreambleTokens())
	fmt.Fprintf(stdout, "  answer    %q\n", truncate(resp.Text, 60))
	fmt.Fprintln(stdout, "  ok")
	return 0
}

func took(start time.Time) time.Duration {
	return time.Since(start).Round(time.Millisecond)
}
