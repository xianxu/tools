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

// Client is the transport. Two methods, because there are two shapes of answer:
// a structured one a program consumes, and prose a person reads as it arrives.
//
// Nothing here mentions Anthropic. That is what lets the wire fake, the real
// client and the live conformance run share one obligation suite (llmtest.Suite).
type Client interface {
	// Complete returns the whole answer at once.
	Complete(ctx context.Context, r Request) (Response, error)
	// Stream delivers text as it arrives and still returns the final Response, so
	// a caller never needs a second call to learn what the answer cost.
	//
	// onDelta receives ANSWER text only. Thinking deltas are accumulated into
	// Blocks but never forwarded: a learner watching a question appear must not
	// see the model's reasoning scroll past. onDelta may be nil.
	Stream(ctx context.Context, r Request, onDelta func(string)) (Response, error)
}

// Request is one call, described independently of any provider's wire format.
type Request struct {
	// Task is a stable identifier — "author-cloze", "veto-distractor". It names
	// the golden file, keys a cassette, and labels usage. It is NOT sent.
	Task string
	// Model, Effort and MaxTokens are per-request overrides; zero means "use the
	// resolved Config default", so a consumer that does not care says nothing.
	Model     string
	Effort    string
	MaxTokens int64

	System string
	Prompt string

	// Schema, when non-nil, asks the provider to constrain output to it. Callers
	// must still handle ErrMalformed: an intermediary may drop the field, and a
	// constraint we did not verify is not a guarantee.
	Schema map[string]any
}

// Response is one answer.
type Response struct {
	// ID is the provider's message id. Kept because it is the ONLY handle that
	// correlates a call with the proxy's own error logs
	// (~/.cli-proxy-api/logs/error-v1-messages-*.log), which is how the first
	// probe in this issue was diagnosed at all.
	ID string
	// Text is every text block joined, in arrival order. NOT Blocks[0].Text:
	// measured, the first block is a thinking block whenever the prompt is
	// non-trivial, and a thinking block can also arrive AFTER the text — the
	// committed captures show [thinking,text], [text] and [thinking,text,thinking].
	Text  string
	Model string
	// Stop is the provider's stop reason, passed through rather than interpreted.
	// classifyStop is what turns it into an error.
	Stop string
	// StopDetails carries the refusal category and explanation. Populated only
	// when Stop == "refusal", nil otherwise — so every read must guard. Without
	// it ErrRefused says a call was refused but never why, and "why" is the
	// difference between a prompt to fix and a topic to avoid.
	StopDetails *StopDetails
	// Blocks is every content block as received, unflattened.
	//
	// Kept, not interpreted. Two reasons it cannot be derived later from Text: a
	// multi-turn follow-up on the same model must echo thinking blocks back BYTE
	// FOR BYTE or the continuation is rejected, and a library that flattens to a
	// string forecloses that for every future consumer.
	Blocks []Block
	Usage  Usage
}

// Block is one content block, preserved as received.
//
// Raw is the load-bearing field. A text block carries its content under "text"
// and a thinking block under "thinking" — DIFFERENT KEYS, confirmed across the
// committed captures — so a struct with one Text field cannot round-trip both.
// Text is the decoded convenience; Raw is the truth.
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

// Usage is what the call cost. Reported, not budgeted against: the operator
// decided on 2026-08-22 that quality wins over cost for this tool, so this
// exists to be shown (--llm-check, --stats later), not to gate anything.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	Duration     time.Duration
	// ThinkingTokens is billed but invisible: opus-5 returns a thinking block
	// whose text is empty by default. Without this the output token count looks
	// inexplicably high, which is how it becomes folklore.
	ThinkingTokens int64
	// The proxy prepends ~1,900 tokens of system preamble that we did not send.
	// It lands in cache_creation on a cold call and cache_read on a warm one —
	// BOTH captures exist, so one field would report zero half the time and the
	// preamble would look like it came and went.
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// PreambleTokens is what arrived that we did not send, cold or warm.
func (u Usage) PreambleTokens() int64 { return u.CacheCreationTokens + u.CacheReadTokens }

// Progress reports where a slow call currently is. Fired on a ticker via
// Config.OnSlow, never on the happy path.
//
// The point is answering "slow WHERE" — a call stuck connecting and a call stuck
// mid-body look identical from outside, and that ambiguity cost kbench two days.
type Progress struct {
	Task    string
	Phase   string // "connect" | "waiting" | "streaming" | "done"
	Elapsed time.Duration
	Bytes   int
}
