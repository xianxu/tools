// Package llm is the one way anything in tools/ talks to a language model.
//
// It owns the transport, the error taxonomy, and the typed-task layer. It owns
// no prompts: a prompt is domain knowledge and lives with the consumer that
// needs it, as a Task[T] whose result type IS its schema.
//
// It discovers advertised local models lazily on first use, but does not restart
// or heal the proxy it talks to.
// Parley owns that ladder, with a repair budget and one-shot guards; a second
// healing mechanism here would be a parallel track where one already works. Our
// contribution is an error that says WHICH thing is wrong, so the right
// machinery acts on it.
//
// # Stability
//
// The import path is public (tools#85). v0.x may change incompatibly at a minor
// version bump, so a consumer pins an exact version.
//
// # Concurrency
//
// A Client from New, the auto-discovering one included, is safe for concurrent
// use by any number of goroutines. Calls share no per-call state: each has its
// own stall timer, accumulator and callback, so one call's events never reach
// another's callback, and cancelling one call's context affects that call alone.
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
	// Stream delivers events as they arrive and still returns the final
	// Response. onEvent runs synchronously on the calling goroutine, is never
	// called after Stream returns, and may be nil. Thinking is accumulated into
	// Blocks but never forwarded. TextOnly adapts a plain text callback.
	Stream(ctx context.Context, r Request, onEvent func(Delta)) (Response, error)
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

	// Messages is the conversation so far, oldest first. Prompt, when non-empty,
	// is appended as a final user text turn, so a single-turn caller sets Prompt
	// alone. A request with neither is refused with ErrRequest before sending.
	Messages []Message
	Prompt   string

	// Tools the model may call. ToolChoice is "" (the provider default, auto),
	// "auto", "any", "none", or one declared tool's Name to force it. Because
	// keywords and names share the string, a Tool named auto, any or none is
	// refused with ErrRequest, as is a ToolChoice naming no declared tool.
	Tools      []Tool
	ToolChoice string

	// Schema, when non-nil, constrains output. Callers must STILL handle
	// ErrMalformed: an intermediary may drop the field.
	Schema map[string]any

	// adaptiveThinking records the effective wire mode for request hashing.
	adaptiveThinking bool
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
	// ToolCalls is every tool_use block, in arrival order. Stop "tool_use" is a
	// success: run the tools, append Message() and a user turn of ToolResult
	// blocks, and call again.
	ToolCalls []ToolCall
	Usage     Usage
}

// Message returns the assistant turn to append to the next Request.Messages.
// Every block keeps its Raw, so thinking signatures survive the round trip.
func (r Response) Message() Message {
	return Message{Role: RoleAssistant, Content: append([]Block(nil), r.Blocks...)}
}

// Role is who said a Message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn of a conversation. Content reuses Block, so an assistant
// turn goes back exactly as it arrived.
type Message struct {
	Role    Role
	Content []Block
}

// Tool is one function the model may call.
type Tool struct {
	Name        string
	Description string
	// InputSchema is the JSON Schema of the call's input; nil means
	// {"type":"object"}.
	InputSchema map[string]any
}

// ToolCall is one call the model made.
type ToolCall struct {
	ID   string
	Name string
	// Input is the complete JSON object the model sent; "{}" when it sent none.
	Input json.RawMessage
}

// Text is an outgoing text block.
func Text(s string) Block { return Block{Type: "text", Text: s} }

// ToolResult is an outgoing tool_result block answering the call callID.
func ToolResult(callID, content string, isError bool) Block {
	return Block{Type: "tool_result", CallID: callID, Text: content, IsError: isError}
}

// Block is one content block, preserved as received.
//
// Raw is load-bearing: a text block carries its content under "text" and a
// thinking block under "thinking", so one Text field cannot round-trip both.
// Text is the decoded convenience; Raw is what gets echoed back.
//
// Raw is opaque provider data. A consumer carries it from a Response back to the
// next Request and never authors it: an outgoing block with Raw is sent as
// exactly those bytes, and one without is rendered from its typed fields (build
// those with Text and ToolResult).
type Block struct {
	Type      string          // "text" | "thinking" | "tool_use" | "tool_result" | future
	Text      string          // text/thinking content, or a tool_result's content
	Signature string          // thinking blocks carry one; it must round-trip unchanged
	ToolCall  *ToolCall       // set when Type == "tool_use"
	CallID    string          // set when Type == "tool_result"
	IsError   bool            // tool_result only
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
	// Attempts counts the HTTP attempts actually dispatched (those
	// Config.BeforeAttempt admitted and sent), retries included. It is set on
	// every return path, errors too: a call whose first attempt the hook rejected
	// reports 0.
	Attempts int
}

// Attempt describes one physical HTTP attempt, for Config.BeforeAttempt.
type Attempt struct {
	Task   string // Request.Task
	Model  string // the effective model
	Number int    // 0 for the first attempt, 1.. for retries
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
