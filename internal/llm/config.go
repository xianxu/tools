package llm

import (
	"fmt"
	"time"
)

const (
	// The local cli-proxy-api is the DEFAULT, not a fallback (operator,
	// 2026-08-22): it fronts a subscription plan, so the good model is also the
	// cheap path. api.anthropic.com is reached by setting DEFINE_LLM_BASE_URL.
	//
	// The instance is parley-managed (~/.local/share/nvim/parley/cliproxy/),
	// which carries auto-healing a standalone install does not.
	defaultBaseURL = "http://127.0.0.1:8317"
	defaultModel   = "claude-opus-5"
	defaultEffort  = "high"
	// Generous on purpose, and not arbitrary: with adaptive thinking on,
	// max_tokens must cover the THINKING AND the answer. A 512-token budget
	// during this issue's probes let thinking consume the lot and returned an
	// answer cut mid-rune that still parsed — the specimen is committed at
	// llmtest/testdata/message-truncated.json. Budget for the answer alone and
	// the answer is what gets cut.
	defaultMaxTokens = 8192
	// Total budget for one call: attempts, backoff and body read. Enforced as a
	// context deadline, which in Go covers every phase — including the header
	// read that in Python needed a worker thread to bound.
	defaultTimeout = 5 * time.Minute
	// Silence inside a stream, which defaultTimeout cannot express: a long answer
	// legitimately takes minutes, a dead connection should fail in seconds.
	defaultStallAfter = 90 * time.Second
)

// Config is where the model lives and who we are.
type Config struct {
	BaseURL   string
	APIKey    string
	Model     string
	Effort    string
	MaxTokens int64
	// Timeout is the TOTAL budget for a call: attempts, backoff and body read.
	Timeout time.Duration
	// StallAfter bounds SILENCE inside a stream. Zero disables it.
	StallAfter time.Duration
	// OnSlow, when set, is called on a ticker while a call is still running, with
	// the phase it is in. Optional, off by default, never called on a fast path —
	// this is for answering "slow where", not for logging.
	OnSlow func(Progress)
}

// Resolve reads configuration from an env lookup function.
//
// It takes the lookup rather than calling os.Getenv so precedence is a table
// test instead of a process-state mutation, and so a test can assert what
// happens with NOTHING set without unsetting the developer's own environment
// (ARCH-PURE: this is the pure half; main() supplies os.Getenv).
//
// It deliberately does NOT probe reachability. A probe here would put a network
// round trip on `define <word>`, whose whole promise is that it is instant and
// offline. An unreachable proxy shows up as ErrUnavailable at the first real
// call, which is the same path as having no key at all — one degradation story,
// not two.
func Resolve(getenv func(string) string) (Config, error) {
	first := func(keys ...string) string {
		for _, k := range keys {
			if v := getenv(k); v != "" {
				return v
			}
		}
		return ""
	}
	c := Config{
		BaseURL:    orDefault(first("DEFINE_LLM_BASE_URL"), defaultBaseURL),
		APIKey:     first("DEFINE_LLM_API_KEY", "ANTHROPIC_API_KEY"),
		Model:      orDefault(first("DEFINE_LLM_MODEL"), defaultModel),
		Effort:     orDefault(first("DEFINE_LLM_EFFORT"), defaultEffort),
		MaxTokens:  defaultMaxTokens,
		Timeout:    defaultTimeout,
		StallAfter: defaultStallAfter,
	}
	if c.APIKey == "" {
		// Names what to do, not just what is wrong. The parley pointer is a hint
		// in a message, not a dependency: nothing here reads parley's files.
		return c, fmt.Errorf("%w: no API key — set DEFINE_LLM_API_KEY or ANTHROPIC_API_KEY "+
			"(the parley-managed proxy keeps one in ~/.local/share/nvim/parley/cliproxy/config.yaml)",
			ErrUnavailable)
	}
	return c, nil
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// Redact renders a credential safe to print. Every diagnostic path goes through
// it; --llm-check prints the base URL and this, never the key.
//
// The short-key branch is not defensive padding: the parley proxy's key is four
// characters, so it is the common case here.
func Redact(key string) string {
	if key == "" {
		return "(unset)"
	}
	if len(key) <= 8 {
		return "(set, short)"
	}
	return key[:6] + "…" + key[len(key)-4:]
}
