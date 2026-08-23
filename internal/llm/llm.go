// Package llm is the one way anything in tools/ talks to a language model.
//
// It owns the transport, the error taxonomy, and the typed-task layer. It owns
// no prompts: a prompt is domain knowledge and lives with the consumer that
// needs it, as a Task[T] whose result type IS its schema.
//
// It also deliberately does NOT probe, restart or heal the proxy it talks to.
// Parley owns that ladder, with a repair budget and one-shot guards; a second
// healing mechanism here would be a parallel track where one already works. Our
// contribution is an error that says WHICH thing is wrong, so the right
// machinery acts on it.
package llm

import (
	"context"
	"encoding/json"
	"time"
)

// Client is the transport: a structured answer a program consumes, and prose a
// person reads as it arrives.
//
// Nothing here mentions Anthropic — that is what lets the fake, the real client
// and the live conformance run share one obligation suite (llmtest.Suite).
type Client interface {
	// Complete returns the whole answer at once.
	Complete(ctx context.Context, r Request) (Response, error)
	// Stream delivers text as it arrives and still returns the final Response.
	// onDelta receives ANSWER text only — thinking deltas are accumulated into
	// Blocks but never forwarded — and may be nil.
	Stream(ctx context.Context, r Request, onDelta func(string)) (Response, error)
}

// Request is one call, described independently of any provider's wire format.
type Request struct {
	// Task is a stable identifier ("author-cloze", "veto-distractor"): it names a
	// golden file, keys a cassette, labels usage. NOT sent to the model.
	Task string
	// Per-request overrides; zero means "use the Config default".
	Model     string
	Effort    string
	MaxTokens int64

	System string
	Prompt string

	// Schema, when non-nil, constrains output. Callers must STILL handle
	// ErrMalformed: an intermediary may drop the field.
	Schema map[string]any
}

// Response is one answer.
type Response struct {
	// ID is the provider's message id — the only handle that correlates a call
	// with the proxy's own error logs (~/.cli-proxy-api/logs/error-v1-messages-*).
	ID string
	// Text is every text block joined, in arrival order. NOT Blocks[0].Text — the
	// captures show [thinking,text], [text] and [thinking,text,thinking], so
	// neither presence nor position is fixed.
	Text  string
	Model string
	// Stop is the provider's stop reason, passed through; classifyStop turns it
	// into an error.
	Stop string
	// StopDetails is populated only when Stop == "refusal" — guard every read.
	StopDetails *StopDetails
	// Blocks is every content block as received, unflattened. Kept, not
	// interpreted: a multi-turn follow-up on the same model must echo thinking
	// blocks back byte for byte or the continuation is rejected.
	Blocks []Block
	Usage  Usage
}

// Block is one content block, preserved as received.
//
// Raw is load-bearing: a text block carries its content under "text" and a
// thinking block under "thinking", so one Text field cannot round-trip both.
// Text is the decoded convenience; Raw is what gets echoed back.
type Block struct {
	Type      string          // "text" | "thinking" | anything a future model adds
	Text      string          // "text"/"thinking" content, whichever this block uses
	Signature string          // thinking blocks carry one; it must round-trip unchanged
	Raw       json.RawMessage // exactly as received — what gets echoed back
}

// StopDetails is the structured reason behind a refusal.
type StopDetails struct {
	Type        string
	Category    string // open set: "cyber", "bio", … or ""
	Explanation string
}

// Usage is what the call cost. Reported, not budgeted against: quality wins over
// cost for this tool (operator, 2026-08-22), so this exists to be surfaced by
// whatever diagnostic reads it, not to gate anything.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	Duration     time.Duration
	// ThinkingTokens is billed but invisible — opus-5's thinking block is empty
	// by default — so without it the output count looks inexplicably high.
	ThinkingTokens int64
	// The proxy prepends system preamble we did not send (~1,900 tokens). It
	// lands in cache_creation when cold and cache_read when warm, so both are
	// kept: one field would report zero half the time.
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// PreambleTokens is what arrived that we did not send, cold or warm.
func (u Usage) PreambleTokens() int64 { return u.CacheCreationTokens + u.CacheReadTokens }

// Progress reports where a slow call currently is. Fired on a ticker via
// Config.OnSlow, never on the happy path.
//
// The point is answering "slow WHERE": a call stuck waiting for a first token
// and one stuck mid-stream look identical from outside.
type Progress struct {
	Task string
	// Phase is "waiting" (no response yet) or "streaming" (mid-body). Those are
	// the two the transport can distinguish and the two it emits; listing phases
	// nothing produces would make this doc a wish.
	Phase   string
	Elapsed time.Duration
}
