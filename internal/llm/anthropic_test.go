package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func client(t *testing.T, base string, tweak ...func(*llm.Config)) llm.Client {
	t.Helper()
	c := llm.Config{
		BaseURL: base, APIKey: "sk-test-1234567890", Model: "claude-opus-5",
		Effort: "high", MaxTokens: 8192, Timeout: 30 * time.Second,
	}
	for _, f := range tweak {
		f(&c)
	}
	return llm.New(c)
}

// CONTENT comes from a committed capture, never a literal — a Reply{Text:
// "flattering, servile"} would assert against my guess at what a model says.
func TestCompleteRoundTrip(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-thinking.json")
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "which also fits"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text == "" {
		t.Error("Text is empty — the text blocks were not collected")
	}
	if len(got.Blocks) < 2 {
		t.Errorf("Blocks = %d, want the thinking block preserved too", len(got.Blocks))
	}
	if got.ID == "" {
		t.Error("ID not carried — it is the only handle that correlates with the proxy's logs")
	}
	if got.Usage.Duration == 0 {
		t.Error("Duration not recorded")
	}
	if got.Usage.ThinkingTokens == 0 {
		t.Error("ThinkingTokens not carried through")
	}
	if got.Usage.PreambleTokens() == 0 {
		t.Error("PreambleTokens is 0 — the ~1,900-token proxy preamble is invisible")
	}
	reqs := f.Requests()
	if len(reqs) != 1 || reqs[0].Path != "/v1/messages" {
		t.Fatalf("requests = %+v", reqs)
	}
	if reqs[0].Headers.Get("anthropic-version") == "" {
		t.Error("anthropic-version header absent — the SDK is expected to send it")
	}
	// The credential must reach the WIRE, under the header the service reads.
	// Deleting option.WithAPIKey was previously caught only by the SDK's own
	// client-side validation, so a key sent under the wrong header would have
	// passed everything here.
	if got := reqs[0].Headers.Get("x-api-key"); got != "sk-test-1234567890" {
		t.Errorf("x-api-key = %q, want the configured key", got)
	}
}

// The regression this exists for: content[0] is a THINKING block on any
// non-trivial prompt, so a naive implementation returns "" live while passing
// against a single-block fake.
func TestTextSkipsThinkingBlock(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-thinking.json")
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "which also fits"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Blocks[0].Type != "thinking" {
		t.Fatalf("capture changed shape; Blocks[0] = %q", got.Blocks[0].Type)
	}
	if got.Text == got.Blocks[0].Text {
		t.Error("Text is the thinking block's content — it must be the TEXT blocks")
	}
	if !strings.Contains(got.Text, "bsequious") {
		t.Errorf("Text = %q, want the answer from the capture", got.Text)
	}
}

// A thinking block's signature must survive verbatim: #16's multi-turn
// follow-ups have to echo it back byte for byte or the continuation is rejected.
func TestThinkingSignatureIsPreserved(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-thinking.json")
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Blocks) == 0 {
		t.Fatal("no blocks")
	}
	if got.Blocks[0].Signature == "" {
		t.Fatal("signature dropped — the thinking block cannot be echoed back")
	}
	if len(got.Blocks[0].Raw) == 0 || !strings.Contains(string(got.Blocks[0].Raw), got.Blocks[0].Signature) {
		t.Error("Raw does not carry the signature; byte-for-byte round-trip is impossible")
	}
}

// The capture that decodes and lies.
func TestTruncatedIsNotSuccess(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-truncated.json")
	_, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "near-synonym?"})
	if !errors.Is(err, llm.ErrTruncated) {
		t.Fatalf("err = %v, want ErrTruncated — a cut-off answer that happens to parse is not an answer", err)
	}
}

// Blocks preserve arrival order, whatever it is. The truncated capture is
// [thinking, text, thinking], so an implementation assuming thinking-then-text
// fails here.
func TestBlocksPreserveArrivalOrder(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-truncated.json")
	// ErrTruncated is expected here — the capture is the truncation specimen —
	// so the error is checked rather than discarded, and Blocks is length-checked
	// before indexing. The earlier version panicked on any unexpected failure,
	// which aborts the package run and masks every other result.
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if !errors.Is(err, llm.ErrTruncated) {
		t.Fatalf("err = %v, want ErrTruncated", err)
	}
	var kinds []string
	for _, b := range got.Blocks {
		kinds = append(kinds, b.Type)
	}
	if len(kinds) != 3 || kinds[2] != "thinking" {
		t.Errorf("Blocks = %v, want the trailing thinking block preserved", kinds)
	}
}

// The retry belongs to the SDK; this asserts we actually get it, and that the
// retried request arrives INTACT (a consumed body would arrive empty).
func TestRetriesOn429(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("hello", llmtest.Reply{Status: 429})
	f.ThenServeRecorded("hello", "message-thinking.json")
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text == "" {
		t.Error("retry produced no text")
	}
	reqs := f.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2 (a retry)", len(reqs))
	}
	if reqs[1].Prompt() != "hello" {
		t.Errorf("retried body = %q, want the original prompt", reqs[1].Prompt())
	}
	// The credential must survive the retry too — a header rebuilt on the second
	// attempt is exactly the kind of thing that only breaks under load.
	if got := reqs[1].Headers.Get("x-api-key"); got != "sk-test-1234567890" {
		t.Errorf("retry sent x-api-key = %q", got)
	}
}

func TestRefusalBecomesErrRefused(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: "", Stop: "refusal"})
	_, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if !errors.Is(err, llm.ErrRefused) {
		t.Fatalf("err = %v, want ErrRefused", err)
	}
}

// A 400 is OUR bug and must stay loud: degrading here would hide it behind the
// same silence as flight mode.
func TestBadRequestIsLoud(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Status: 400})
	_, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if !errors.Is(err, llm.ErrRequest) {
		t.Fatalf("err = %v, want ErrRequest", err)
	}
	if errors.Is(err, llm.ErrUnavailable) {
		t.Error("a 400 must NOT be absorbable as ErrUnavailable")
	}
}

func TestServerDownIsUnavailable(t *testing.T) {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close() // nothing is listening now
	_, err := client(t, "http://"+addr).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestStreamDeltasConcatToText(t *testing.T) {
	f := llmtest.NewFake(t)
	var got strings.Builder
	resp, err := client(t, f.URL).Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"},
		func(s string) { got.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != resp.Text {
		t.Errorf("deltas = %q, final Text = %q — they must agree", got.String(), resp.Text)
	}
	if resp.Text == "" {
		t.Error("no text accumulated from the recorded stream")
	}
}

// Thinking deltas are accumulated but never forwarded: a learner watching a
// question appear must not see the model's reasoning scroll past.
//
// This asserts the CALL COUNT, not the content, and that is the whole point. The
// first version compared onDelta's output against the thinking block's text —
// and the recorded capture's thinking_delta carries "" (the model returns an
// empty thinking body even under display:summarized), so there was nothing to
// leak and the assertion could never fire. Mutation-checked: forwarding thinking
// deltas as well left that version GREEN. An observable both implementations
// produce cannot distinguish between them, however true the assertion looks.
//
// The counts come from the capture: 5 text_delta frames, 1 thinking_delta.
func TestStreamDoesNotForwardThinkingDeltas(t *testing.T) {
	f := llmtest.NewFake(t)
	textDeltas := countFrames(t, "text_delta")
	thinkingDeltas := countFrames(t, "thinking_delta")
	if thinkingDeltas == 0 {
		t.Fatal("the capture has no thinking_delta frames; this test cannot detect forwarding")
	}

	calls := 0
	_, err := client(t, f.URL).Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"},
		func(string) { calls++ })
	if err != nil {
		t.Fatal(err)
	}
	if calls != textDeltas {
		t.Errorf("onDelta called %d times, want %d (one per text_delta); "+
			"%d thinking_delta frames must not reach it", calls, textDeltas, thinkingDeltas)
	}
}

// countFrames counts SSE delta frames of a given type in the committed capture,
// so the expected numbers come from the artifact rather than from a literal that
// silently rots when the capture is re-recorded.
func countFrames(t *testing.T, kind string) int {
	t.Helper()
	return strings.Count(string(llmtest.Capture(t, "stream-sample.sse")), `"type":"`+kind+`"`)
}

// kbench's EXACT failure mode, ported: a peer that dribbles headers one byte at
// a time is bounded by no socket timeout, and in Python it took three attempts
// to bound it (measured there at 25.2s against a 2s deadline, then 1004s against
// 180s). Go's net/http honours ctx through the header read, so this passes as
// written — which is precisely why it is ASSERTED rather than assumed. If Go
// ever stops honouring it, this test is the alarm.
func TestSlowHeadersStillHitTheDeadline(t *testing.T) {
	srv := dribblingServer(t)
	c := client(t, srv, func(c *llm.Config) { c.Timeout = 2 * time.Second })

	start := time.Now()
	_, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "hello"})
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Fatalf("took %s against a 2s deadline — the header read is unbounded", elapsed)
	}
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}

// A deadline is the wrong instrument for a stream: minutes of headroom for a
// long answer, seconds of patience for a dead one. StallAfter is the difference,
// and it is the one bound the SDK does not provide.
//
// A stall BEFORE any frame is a genuine outage: nothing was ever reached.
func TestStreamStallBeforeAnyFrameIsUnavailable(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{StallEarly: true})
	c := stallClient(t, f.URL)
	start := time.Now()
	_, err := c.Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"}, nil)
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("stall took %s to detect with StallAfter=500ms", elapsed)
	}
}

// A stall AFTER answer text has arrived is a truncation, not an outage — the
// service is demonstrably reachable — and the text already received must survive.
//
// This case was UNREACHABLE for a whole milestone: the fake stalled on the first
// content_block_delta, which in the capture is a thinking delta, so only the
// before-any-text path ever ran and the defect below hid behind it. The
// production code returned Response{} + ErrUnavailable here, discarding an
// answer the caller had already watched arrive.
func TestStreamStallAfterTextSalvagesAndTruncates(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Stall: true})

	var seen strings.Builder
	resp, err := stallClient(t, f.URL).Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"},
		func(s string) { seen.WriteString(s) })

	if !errors.Is(err, llm.ErrTruncated) {
		t.Fatalf("err = %v, want ErrTruncated — frames arrived, so the service was reachable", err)
	}
	if errors.Is(err, llm.ErrUnavailable) {
		t.Error("a stall after text must not read as an outage: the caller should skip the question, not stop trying")
	}
	if seen.Len() == 0 {
		t.Fatal("no deltas were delivered; this test is not exercising the after-text case")
	}
	if resp.Text != seen.String() {
		t.Errorf("Response.Text = %q but the caller saw %q — the partial answer was discarded", resp.Text, seen.String())
	}
}

func stallClient(t *testing.T, url string) llm.Client {
	return client(t, url, func(c *llm.Config) {
		c.StallAfter = 500 * time.Millisecond
		c.Timeout = 30 * time.Second // deliberately far larger; StallAfter must fire first
	})
}

// Retries spend the SAME budget as the call. The SDK selects on ctx.Done()
// during backoff (requestconfig.go); this asserts the behaviour, not the reading.
func TestRetryBackoffRespectsTheDeadline(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("hello", llmtest.Reply{Status: 429}, llmtest.Reply{Status: 429}, llmtest.Reply{Status: 429})
	c := client(t, f.URL, func(c *llm.Config) { c.Timeout = time.Second })
	start := time.Now()
	_, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "hello"})
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("backoff ran %s past a 1s deadline", elapsed)
	}
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}

// A malformed frame ends the stream — the SDK's SSE decoder owns framing and
// there is no skipping a frame it refuses, so the Lua-side lesson that a bad
// chunk must never abort a stream does NOT transfer to this SDK.
//
// What must not happen is losing the answer that already arrived. A stream that
// dies mid-reply is a truncation, not an outage (kbench's is_outage draws the
// same line), so the partial text comes back WITH ErrTruncated.
func TestMalformedFrameTruncatesButDoesNotLoseText(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{JunkFrame: true})
	resp, err := client(t, f.URL).Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"}, nil)
	if !errors.Is(err, llm.ErrTruncated) {
		t.Fatalf("err = %v, want ErrTruncated", err)
	}
	if errors.Is(err, llm.ErrUnavailable) {
		t.Error("a mid-reply truncation must not read as an outage — they need opposite responses")
	}
	if resp.Text == "" {
		t.Error("the text that arrived before the junk frame was thrown away")
	}
}

func TestCancellationReturnsPromptly(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Stall: true})
	ctx, cancel := context.WithCancel(t.Context())
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	start := time.Now()
	_, err := client(t, f.URL, func(c *llm.Config) { c.StallAfter = 0 }).
		Stream(ctx, llm.Request{Task: "t", Prompt: "x"}, nil)
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("cancellation took %s", elapsed)
	}
	if err == nil {
		t.Error("expected an error after cancellation")
	}
}

// The schema reaches the wire as output_config.format.schema. Without this a
// silently-dropped schema looks identical to a working one until a consumer
// starts getting free prose.
func TestSchemaReachesTheWire(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: `{"ok":true}`})
	schema := map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}}
	_, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x", Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	got := f.Requests()[0].Schema()
	if got == nil {
		t.Fatal("output_config.format.schema absent from the request")
	}
	if got["type"] != "object" {
		t.Errorf("schema = %+v", got)
	}
}

// dribblingServer writes one header byte at a time, forever. No socket timeout
// can bound a peer like this; only a deadline that covers the header read can.
func dribblingServer(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				c.Write([]byte("HTTP/1.1 200 OK\r\nX-Pad: "))
				for {
					if _, err := c.Write([]byte("a")); err != nil {
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
			}(conn)
		}
	}()
	return "http://" + l.Addr().String()
}

// BR-22's rule, applied: a behaviour the code singles out as load-bearing needs a
// fixture that separates it from the alternative it warns against. Where no
// committed capture exhibits that shape, the fixture is CONSTRUCTED — block count
// and header presence are transport shape, not judgment.

// "Text is every text block JOINED, not Content[0].Text" is asserted by the
// atlas, the package doc and Response.Text's own comment — and was untestable,
// because all three captures carry exactly one text block. A first-block-only
// implementation passed everything.
func TestTextJoinsEveryTextBlock(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: "alpha beta gamma delta", SplitText: 3})
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if err != nil {
		t.Fatal(err)
	}
	var texts int
	for _, b := range got.Blocks {
		if b.Type == "text" {
			texts++
		}
	}
	if texts < 3 {
		t.Fatalf("fixture served %d text blocks, want 3 — this test cannot detect the defect", texts)
	}
	if got.Text != "alpha beta gamma delta" {
		t.Errorf("Text = %q — every text block must be joined, not just the first", got.Text)
	}
}

// The system prompt must reach the wire. Recorded.System() existed for exactly
// this assertion and had zero callers, so deleting the System plumbing left the
// suite green — and M2's Task[T] sets System on every call, where a silently
// dropped system prompt looks identical to a working one.
func TestSystemPromptReachesTheWire(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: "ok"})
	_, err := client(t, f.URL).Complete(t.Context(), llm.Request{
		Task: "t", System: "You are a lexicographer.", Prompt: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Requests()[0].System(); got != "You are a lexicographer." {
		t.Errorf("system on the wire = %q, want the configured system prompt", got)
	}
}

// OnSlow/Progress existed with no test at all: neutering watch() left the suite
// green, and the phase vocabulary drifted from what the tree emits as a result.
func TestOnSlowReportsThePhase(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Stall: true})

	var mu sync.Mutex
	var seen []llm.Progress
	c := llm.New(llm.Config{
		BaseURL: f.URL, APIKey: "sk-test-1234567890",
		StallAfter: 3 * time.Second, Timeout: 30 * time.Second,
		SlowEvery: 200 * time.Millisecond,
		OnSlow: func(p llm.Progress) {
			mu.Lock()
			seen = append(seen, p)
			mu.Unlock()
		},
	})
	_, _ = c.Stream(t.Context(), llm.Request{Task: "author-cloze", Prompt: "x"}, nil)

	mu.Lock()
	defer mu.Unlock()
	if len(seen) == 0 {
		t.Fatal("OnSlow never fired on a stalled call")
	}
	for _, p := range seen {
		if p.Phase != "streaming" {
			t.Errorf("phase = %q, want streaming (the tree emits only waiting|streaming)", p.Phase)
		}
		if p.Task != "author-cloze" {
			t.Errorf("task = %q, want the request's task", p.Task)
		}
		if p.Elapsed == 0 {
			t.Error("Elapsed is zero")
		}
	}
}

// StopDetails exists so ErrRefused can say WHY. Nothing referenced it, and the
// fake's refusal emitted no stop_details, so the populated branch was unreachable
// from either side of the seam.
func TestRefusalCarriesItsReason(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Stop: "refusal"})
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if !errors.Is(err, llm.ErrRefused) {
		t.Fatalf("err = %v, want ErrRefused", err)
	}
	if got.StopDetails == nil {
		t.Fatal("StopDetails is nil — ErrRefused can say that a call was refused but not why")
	}
	if got.StopDetails.Category == "" || got.StopDetails.Explanation == "" {
		t.Errorf("StopDetails = %+v, want category and explanation", got.StopDetails)
	}
}

// Out-of-range Config values are as ordinary as unset ones, and one of them used
// to panic on the watcher goroutine where no caller could recover: cmp.Or only
// replaces the ZERO value, so a negative SlowEvery reached time.NewTicker.
func TestNegativeConfigValuesDoNotPanic(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: "ok"})
	c := llm.New(llm.Config{
		BaseURL: f.URL, APIKey: "sk-test-1234567890",
		SlowEvery: -1, MaxTokens: -5,
		OnSlow: func(llm.Progress) {},
	})
	if _, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"}); err != nil {
		t.Fatalf("negative config values broke the call: %v", err)
	}
}

// The fixture is deliberately NOT ASCII. splitInto chopped byte offsets, so
// multibyte text was cut mid-rune and json.Marshal substituted U+FFFD — invisible
// to an ASCII fixture, while every committed capture contains em-dashes.
func TestTextJoinsMultibyteBlocksWithoutCorruption(t *testing.T) {
	const answer = "obséquieux — très flagorneur, vraiment"
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: answer, SplitText: 3})
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != answer {
		t.Errorf("Text = %q, want %q — blocks were split mid-rune", got.Text, answer)
	}
	if strings.ContainsRune(got.Text, '�') {
		t.Error("replacement character in the joined text")
	}
}

// A .json capture on a streaming request used to be silently swapped for
// stream-sample.sse, so a test would assert against a response it never asked for.
func TestJSONCaptureOnAStreamingRequestFailsLoudly(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Capture: "message-thinking.json"})
	_, err := client(t, f.URL).Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"}, nil)
	if err == nil {
		t.Fatal("a .json capture was silently served as the SSE sample")
	}
	if !strings.Contains(err.Error(), "STREAMING request") {
		t.Errorf("err = %v, want the fake's explanation", err)
	}
}

// InputTokens and CacheCreationTokens were only ever asserted through
// PreambleTokens(), which SUMS CacheCreation + CacheRead — so swapping the two
// assignments in buildResponse was invisible to the whole suite.
func TestUsageFieldsAreCarriedIndividually(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-thinking.json")
	got, err := client(t, f.URL).Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
	if err != nil {
		t.Fatal(err)
	}
	raw := struct {
		Usage struct {
			InputTokens         int64 `json:"input_tokens"`
			OutputTokens        int64 `json:"output_tokens"`
			CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadTokens     int64 `json:"cache_read_input_tokens"`
			Details             struct {
				ThinkingTokens int64 `json:"thinking_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	}{}
	if err := json.Unmarshal(llmtest.Capture(t, "message-thinking.json"), &raw); err != nil {
		t.Fatal(err)
	}
	for _, f := range []struct {
		name      string
		got, want int64
	}{
		{"InputTokens", got.Usage.InputTokens, raw.Usage.InputTokens},
		{"OutputTokens", got.Usage.OutputTokens, raw.Usage.OutputTokens},
		{"CacheCreationTokens", got.Usage.CacheCreationTokens, raw.Usage.CacheCreationTokens},
		{"CacheReadTokens", got.Usage.CacheReadTokens, raw.Usage.CacheReadTokens},
		{"ThinkingTokens", got.Usage.ThinkingTokens, raw.Usage.Details.ThinkingTokens},
	} {
		if f.got != f.want {
			t.Errorf("%s = %d, want %d (from the capture)", f.name, f.got, f.want)
		}
	}
}

type vetoResult struct {
	Fits   bool   `json:"fits"`
	Reason string `json:"reason"`
}

func TestRunDecodesIntoTheResultType(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("near-synonym", llmtest.Reply{Text: `{"fits":true,"reason":"both describe servile flattery"}`})
	got, err := llm.Run(t.Context(), client(t, f.URL), llm.Task[vetoResult]{
		Name: "veto-distractor", Prompt: "Is obsequious a near-synonym of sycophantic?",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Fits || got.Reason == "" {
		t.Errorf("got %+v", got)
	}
	// The schema derived from vetoResult must reach the wire, or the constraint
	// exists only in our imagination.
	schema := f.Requests()[0].Schema()
	if schema == nil {
		t.Fatal("no schema on the request")
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["fits"]; !ok {
		t.Errorf("schema does not describe the result type: %+v", schema)
	}
}

// THE ordering test. A truncated structured answer parses cleanly, so a
// decode-first implementation returns success on garbage and nothing downstream
// can tell. The capture is the specimen: it decodes to
// {"verdict":"yes","reason":": Ā"} with both required fields present.
func TestRunChecksTheStopReasonBeforeDecoding(t *testing.T) {
	type verdict struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	f := llmtest.NewFake(t)
	f.ServeRecorded("message-truncated.json")
	_, err := llm.Run(t.Context(), client(t, f.URL), llm.Task[verdict]{
		Name: "veto-distractor", Prompt: "near-synonym?",
	})
	if !errors.Is(err, llm.ErrTruncated) {
		t.Fatalf("err = %v, want ErrTruncated — this payload DECODES, so only the stop reason can catch it", err)
	}
	if errors.Is(err, llm.ErrMalformed) {
		t.Error("reported as malformed; the answer parsed fine, it was cut short")
	}
}

// Every error names its task, or a failure in a batch of five authoring calls is
// unattributable.
func TestRunErrorsNameTheTask(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: "not json at all"})
	_, err := llm.Run(t.Context(), client(t, f.URL), llm.Task[vetoResult]{Name: "author-cloze", Prompt: "x"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "author-cloze") {
		t.Errorf("err = %v, want the task name", err)
	}
}

// An unavailable seam must stay recognisable THROUGH Run, or consumers cannot
// degrade — the property #12 and #13 both depend on.
func TestRunPreservesTheTaxonomy(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Status: 503})
	_, err := llm.Run(t.Context(), client(t, f.URL), llm.Task[vetoResult]{Name: "veto", Prompt: "x"})
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable to survive Run's wrapping", err)
	}
}

// The cassette key must be the EFFECTIVE request, not the caller's raw one:
// two clients with different default models would otherwise share a recording
// while putting different models on the wire.
func TestRequestInContextCarriesTheEffectiveModel(t *testing.T) {
	var mu sync.Mutex
	var seen []llm.Request
	// Answers 200 rather than erroring: a transport error is RETRIED, and an
	// earlier version of this test blocked forever when the retries overran a
	// buffered channel. The double must not perturb the thing it observes.
	capture := roundTripStub(func(req *http.Request) (*http.Response, error) {
		if r, ok := llm.RequestFromContext(req.Context()); ok {
			mu.Lock()
			seen = append(seen, r)
			mu.Unlock()
		}
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"type":"message","role":"assistant","id":"m","model":"x",` +
					`"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn",` +
					`"usage":{"input_tokens":1,"output_tokens":1}}`)),
		}, nil
	})

	for _, model := range []string{"claude-opus-5", "claude-sonnet-5"} {
		c := llm.New(llm.Config{
			BaseURL: "http://example", APIKey: "sk-test-1234567890",
			Model: model, Timeout: 10 * time.Second, Transport: capture,
		})
		// The Request leaves Model unset, so the CONFIG decides what is sent.
		if _, err := c.Complete(t.Context(), llm.Request{Task: "veto", Prompt: "x"}); err != nil {
			t.Fatalf("model %s: %v", model, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 {
		t.Fatalf("captured %d requests, want 2", len(seen))
	}
	a, b := seen[0], seen[1]
	if a.Model == "" || b.Model == "" {
		t.Fatalf("the context carried an unresolved model: %q / %q", a.Model, b.Model)
	}
	if a.Model == b.Model {
		t.Fatal("two configs with different models produced the same effective request")
	}
	if llm.RequestHash(a) == llm.RequestHash(b) {
		t.Error("different effective models share a cassette key; one recording would answer for both")
	}
}

type roundTripStub func(*http.Request) (*http.Response, error)

func (f roundTripStub) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The matrix, both directions. Fixing one direction was the instance fix: frame
// choreography scripted against Complete was just as silently ignored as an
// invented body scripted against Stream.
func TestMisappliedReplyFieldsFailLoudlyOnBothPaths(t *testing.T) {
	cases := []struct {
		name      string
		reply     llmtest.Reply
		streaming bool
		wantIn    string
	}{
		{"invented text on a stream", llmtest.Reply{Text: "invented"}, true, "STREAMING"},
		{"a split body on a stream", llmtest.Reply{Text: "a b c", SplitText: 3}, true, "STREAMING"},
		{"a .json capture on a stream", llmtest.Reply{Capture: "message-thinking.json"}, true, "STREAMING"},
		{"a stall on a non-stream", llmtest.Reply{Stall: true}, false, "frame choreography"},
		{"an early stall on a non-stream", llmtest.Reply{StallEarly: true}, false, "frame choreography"},
		{"a junk frame on a non-stream", llmtest.Reply{JunkFrame: true}, false, "frame choreography"},
		{"an .sse capture on a non-stream", llmtest.Reply{Capture: "stream-sample.sse"}, false, "frame choreography"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := llmtest.NewFake(t)
			f.Script("x", c.reply)
			cl := client(t, f.URL)
			var err error
			if c.streaming {
				_, err = cl.Stream(t.Context(), llm.Request{Task: "t", Prompt: "x"}, nil)
			} else {
				_, err = cl.Complete(t.Context(), llm.Request{Task: "t", Prompt: "x"})
			}
			if err == nil {
				t.Fatal("the field was silently ignored")
			}
			if !strings.Contains(err.Error(), c.wantIn) {
				t.Errorf("err = %v, want it to mention %q", err, c.wantIn)
			}
			// A harness mistake must stay loud: absorbable as ErrUnavailable it
			// would look exactly like flight mode.
			if !errors.Is(err, llm.ErrRequest) {
				t.Errorf("err = %v, want ErrRequest", err)
			}
		})
	}
}

// A per-request override is a SECOND door into a Config field, and normalising
// only Config left it open: a negative Request.MaxTokens reached the wire as -5
// with a nil error.
func TestNegativeRequestOverridesDoNotReachTheWire(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("x", llmtest.Reply{Text: "ok"})
	_, err := client(t, f.URL).Complete(t.Context(), llm.Request{
		Task: "t", Prompt: "x", MaxTokens: -5,
	})
	if err != nil {
		t.Fatal(err)
	}
	mt, ok := f.Requests()[0].Body["max_tokens"].(float64)
	if !ok {
		t.Fatalf("max_tokens absent from the request: %+v", f.Requests()[0].Body)
	}
	if mt <= 0 {
		t.Errorf("max_tokens = %v reached the wire; a negative override must take the default", mt)
	}
}
