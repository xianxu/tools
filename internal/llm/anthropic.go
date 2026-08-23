package llm

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicClient is the real transport.
//
// Retries are the SDK's, not a second hand-rolled loop: it covers 408, 409, 429
// and every 5xx (which includes 529 "overloaded", observed live while recording
// this issue's captures), honours retry-after and x-should-retry, checks
// ctx.Err() before each attempt and selects on ctx.Done() during backoff. So one
// context deadline bounds attempts PLUS backoff PLUS body read — the property
// that in Python needed a worker thread to achieve, and that TestSlowHeaders…
// asserts rather than assumes.
type anthropicClient struct {
	cfg Config
	api anthropic.Client
}

// New builds a Client from resolved config.
func New(c Config) Client {
	// Default EVERY field. Config is this package's whole public input, and a
	// caller writing llm.New(llm.Config{BaseURL: u, APIKey: k}) — the natural
	// first thing to type — must not silently lose the stall bound, which is the
	// one protection the SDK does not supply.
	c.Timeout = cmp.Or(c.Timeout, defaultTimeout)
	c.StallAfter = cmp.Or(c.StallAfter, defaultStallAfter) // negative = disabled, preserved
	// cmp.Or only replaces the ZERO value, so a negative slipped through to
	// time.NewTicker, which panics — on the watcher goroutine, where no caller
	// can recover. Config is public input; out-of-range is as ordinary as unset.
	if c.SlowEvery <= 0 {
		c.SlowEvery = defaultSlowEvery
	}
	c.MaxTokens = cmp.Or(c.MaxTokens, defaultMaxTokens)
	c.Model = cmp.Or(c.Model, defaultModel)
	c.Effort = cmp.Or(c.Effort, defaultEffort)
	c.BaseURL = cmp.Or(c.BaseURL, defaultBaseURL)
	return &anthropicClient{
		cfg: c,
		api: anthropic.NewClient(
			option.WithBaseURL(c.BaseURL),
			option.WithAPIKey(c.APIKey),
			option.WithMaxRetries(2),
			option.WithRequestTimeout(c.Timeout),
		),
	}
}

func (a *anthropicClient) params(r Request) anthropic.MessageNewParams {
	p := anthropic.MessageNewParams{
		Model:     anthropic.Model(cmp.Or(r.Model, a.cfg.Model)),
		MaxTokens: cmp.Or(r.MaxTokens, a.cfg.MaxTokens),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(r.Prompt)),
		},
	}
	if r.System != "" {
		p.System = []anthropic.TextBlockParam{{Text: r.System}}
	}
	// Thinking is left UNSET: on claude-opus-5 that runs adaptive by default,
	// which is what we want, and budget_tokens would be rejected with a 400.
	oc := anthropic.OutputConfigParam{
		Effort: anthropic.OutputConfigEffort(cmp.Or(r.Effort, a.cfg.Effort)),
	}
	if r.Schema != nil {
		oc.Format = anthropic.JSONOutputFormatParam{Schema: r.Schema}
	}
	p.OutputConfig = oc
	return p
}

func (a *anthropicClient) Complete(ctx context.Context, r Request) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	done := a.watch(ctx, r.Task, "waiting")
	start := time.Now()
	msg, err := a.api.Messages.New(ctx, a.params(r))
	done()
	if err != nil {
		return Response{}, a.mapError(err)
	}
	return buildResponse(msg, time.Since(start))
}

func (a *anthropicClient) Stream(ctx context.Context, r Request, onDelta func(string)) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	// stall is the one bound the SDK does not provide, and a total deadline
	// cannot express it: a long answer legitimately takes minutes while a dead
	// connection should fail in seconds. The timer is reset by every event, so it
	// measures SILENCE rather than duration.
	stallCtx, stallCancel := context.WithCancel(ctx)
	defer stallCancel()
	var stalled bool
	var mu sync.Mutex
	reset := func() {}
	if a.cfg.StallAfter > 0 { // negative disables; zero was defaulted by New
		timer := time.AfterFunc(a.cfg.StallAfter, func() {
			mu.Lock()
			stalled = true
			mu.Unlock()
			stallCancel()
		})
		defer timer.Stop()
		reset = func() { timer.Reset(a.cfg.StallAfter) }
	}

	done := a.watch(stallCtx, r.Task, "streaming")
	defer done()

	start := time.Now()
	stream := a.api.Messages.NewStreaming(stallCtx, a.params(r))
	msg := anthropic.Message{}
	// sawEvent is the outage/truncation discriminator. Once a single frame has
	// arrived the service is demonstrably reachable, so a later failure is the
	// reply being cut short — NOT the endpoint being down. kbench's is_outage
	// draws this line deliberately ("408 is absent, and so is every deadline
	// miss"), because conflating the two either parks a healthy run or abandons a
	// recoverable one. Here it decides whether a consumer retries later or skips
	// the question now.
	sawEvent := false
	for stream.Next() {
		event := stream.Current()
		sawEvent = true
		reset()
		// Accumulate BEFORE inspecting: it is what refreshes each block's raw
		// JSON at content_block_stop, which is what makes byte-for-byte thinking
		// preservation possible at all.
		if err := msg.Accumulate(event); err != nil {
			// An event the accumulator rejects is skipped rather than fatal.
			// Note this is NOT the same as a frame the SSE decoder rejects — see
			// the salvage branch below, which is where that lands.
			continue
		}
		if onDelta == nil {
			continue
		}
		if d, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			// ANSWER text only. Thinking deltas are accumulated into Blocks but
			// never forwarded: a learner watching a question appear must not see
			// the model's reasoning scroll past.
			if td, ok := d.Delta.AsAny().(anthropic.TextDelta); ok {
				onDelta(td.Text)
			}
		}
	}
	if err := stream.Err(); err != nil {
		mu.Lock()
		s := stalled
		mu.Unlock()
		if s {
			// A stall is not special-cased: it goes through the SAME discriminator
			// as any other mid-stream failure, because the caller's question is
			// identical — did we ever reach the service?
			partial, _ := buildResponse(&msg, time.Since(start))
			if sawEvent {
				return partial, fmt.Errorf("%w: stream went silent for %s after %d bytes of text",
					ErrTruncated, a.cfg.StallAfter, len(partial.Text))
			}
			return Response{}, fmt.Errorf("%w: stream went silent for %s before any frame (phase streaming)",
				ErrUnavailable, a.cfg.StallAfter)
		}
		// The SDK's SSE decoder owns framing, so a frame it cannot decode ends the
		// stream and there is no skipping it from up here. The Lua-side lesson
		// that a malformed chunk must never abort a stream does NOT transfer to
		// this SDK, and a comment claiming otherwise would describe behaviour we
		// do not have.
		//
		// What does transfer is the half that matters: once the stream started,
		// this is a cut-short reply rather than an outage, and whatever text
		// already arrived is not thrown away. A caller wanting the prefix can have
		// it; one that treats any error as "skip this question" is correct by
		// default.
		if sawEvent {
			partial, _ := buildResponse(&msg, time.Since(start))
			return partial, fmt.Errorf("%w: stream ended mid-reply after %d bytes of text: %w",
				ErrTruncated, len(partial.Text), err)
		}
		return Response{}, a.mapError(err)
	}
	return buildResponse(&msg, time.Since(start))
}

// buildResponse converts an SDK message into our provider-independent shape.
//
// A package-level PURE function, not a method: it touches no IO, and as a method
// the one piece of real transformation logic in this package was reachable only
// through an httptest server (ARCH-PURE). It is now unit-tested directly from a
// committed capture with json.Unmarshal and no server at all.
//
// Shared by both paths so they cannot disagree about how Text is assembled,
// which is the thing most likely to drift: Text is every text block JOINED, in
// arrival order — not Content[0].Text. Measured, the first block is a thinking
// block on any non-trivial prompt, and the committed captures show a thinking
// block arriving AFTER the text too.
func buildResponse(msg *anthropic.Message, took time.Duration) (Response, error) {
	out := Response{
		ID:    msg.ID,
		Model: string(msg.Model),
		Stop:  string(msg.StopReason),
		Usage: Usage{
			InputTokens:         msg.Usage.InputTokens,
			OutputTokens:        msg.Usage.OutputTokens,
			ThinkingTokens:      msg.Usage.OutputTokensDetails.ThinkingTokens,
			CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
			CacheReadTokens:     msg.Usage.CacheReadInputTokens,
			Duration:            took,
		},
	}
	var text strings.Builder
	for _, b := range msg.Content {
		blk := Block{Type: b.Type, Signature: b.Signature, Raw: json.RawMessage(b.RawJSON())}
		switch b.Type {
		case "text":
			blk.Text = b.Text
			text.WriteString(b.Text)
		case "thinking":
			blk.Text = b.Thinking
		}
		out.Blocks = append(out.Blocks, blk)
	}
	out.Text = text.String()
	if msg.StopReason == anthropic.StopReasonRefusal {
		out.StopDetails = &StopDetails{
			Type:        "refusal",
			Category:    string(msg.StopDetails.Category),
			Explanation: msg.StopDetails.Explanation,
		}
	}
	// The stop reason is checked AFTER the response is built, so a caller that
	// wants to inspect a truncated answer still can — but it is returned WITH an
	// error, because a cut-off answer that happens to parse is not an answer.
	if err := classifyStop(out.Stop); err != nil {
		return out, err
	}
	return out, nil
}

// mapError turns an SDK error into the taxonomy.
func (a *anthropicClient) mapError(err error) error {
	var apierr *anthropic.Error
	if errors.As(err, &apierr) {
		return classifyStatus(apierr.StatusCode, err)
	}
	// No status: never reached the server, or the context expired. Both are the
	// ordinary offline state from a consumer's point of view.
	return classifyStatus(0, err)
}

// watch fires Config.OnSlow on a ticker while a call is in flight, so a slow call
// can answer "slow WHERE". Returns a stop function.
//
// Off unless OnSlow is set, and never called on a fast path: this exists for
// diagnosis, not logging.
func (a *anthropicClient) watch(ctx context.Context, task, phase string) func() {
	if a.cfg.OnSlow == nil {
		return func() {}
	}
	start := time.Now()
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(a.cfg.SlowEvery)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				a.cfg.OnSlow(Progress{Task: task, Phase: phase, Elapsed: time.Since(start)})
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(stop) }) }
}
