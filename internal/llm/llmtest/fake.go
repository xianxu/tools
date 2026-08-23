// Package llmtest holds the stateful fake every llm.Client test runs against,
// and the obligation suite the fake and the live service must BOTH satisfy.
//
// The fake is an httptest server speaking the Anthropic wire protocol — not a
// stub of llm.Client. That placement is the point: a stubbed Client cannot see a
// mis-serialized output_config, a dropped anthropic-version header, a retry that
// re-sends a consumed body, or an SSE frame the parser mishandles, because all
// of those live BELOW the seam it replaces. Putting the double on the wire means
// production flow and test flow share the same boundary.
//
// CONTENT comes from committed captures, never from literals in a test. You
// cannot fake judgment; you can freeze a real answer and pin our handling of it.
// Only transport SHAPES are invented here — a 429, a refusal, a stalled stream —
// because those are protocol rather than judgment.
package llmtest

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Captures are embedded rather than read from disk so consumers outside this
// package (cmd/define's tests) can use the fake without their own testdata copy.
// go:embed paths are package-relative, so this is the only place they resolve.
//
//go:embed testdata/*.json testdata/*.sse
var captures embed.FS

// Capture reads a committed capture by name. Fails the test rather than
// returning an error: a missing capture is a broken checkout, not a condition to
// handle.
func Capture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := captures.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("llmtest: capture %q: %v (run scripts/llm-probe.sh record)", name, err)
	}
	return b
}

// Recorded is one request as the server saw it.
type Recorded struct {
	Path    string
	Headers http.Header
	Body    map[string]any
}

// System returns the request's system prompt as a single string, or "".
func (r Recorded) System() string { return joinBlocks(r.Body["system"]) }

// Prompt returns the last user message's text.
func (r Recorded) Prompt() string {
	msgs, _ := r.Body["messages"].([]any)
	for i := len(msgs) - 1; i >= 0; i-- {
		m, _ := msgs[i].(map[string]any)
		if m["role"] == "user" {
			return joinBlocks(m["content"])
		}
	}
	return ""
}

// Schema returns the request's output_config.format.schema, or nil.
func (r Recorded) Schema() map[string]any {
	oc, _ := r.Body["output_config"].(map[string]any)
	f, _ := oc["format"].(map[string]any)
	s, _ := f["schema"].(map[string]any)
	return s
}

// Streaming reports whether the request asked for SSE.
func (r Recorded) Streaming() bool { b, _ := r.Body["stream"].(bool); return b }

func joinBlocks(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var b strings.Builder
		for _, e := range t {
			m, _ := e.(map[string]any)
			if s, ok := m["text"].(string); ok {
				b.WriteString(s)
			}
		}
		return b.String()
	}
	return ""
}

// Reply is one scripted answer.
//
// Text/Stop describe an invented but well-formed answer; Status and Body model
// transport failures. Capture names a committed artifact to serve verbatim —
// which is how anything resembling real model output enters a test.
type Reply struct {
	Capture string // committed capture served verbatim; wins over the rest
	Text    string // invented text, wrapped in a realistic envelope
	Stop    string // "" means "end_turn"
	Status  int    // non-zero: reply with this HTTP status instead
	Body    string // non-zero Status only: exact error body

	// NoThinking omits the leading thinking block. The default INCLUDES one,
	// because that is what the live service does on any non-trivial prompt.
	NoThinking bool
	// Stall replays frames up to the first content_block_delta then goes silent
	// without closing — the shape a hung upstream actually has.
	Stall bool
	// JunkFrame inserts an undecodable data line AFTER the first text delta, so
	// the salvage path is genuinely exercised. Injecting it earlier would only
	// prove that a stream with no content yet fails — which is the uninteresting
	// half.
	JunkFrame bool
}

// knownModel mirrors the model list the proxy actually serves (measured via
// GET /v1/models). Deliberately a prefix check rather than the full 31-entry
// list: the point is to reject a typo, not to be a registry that goes stale.
func knownModel(m string) bool {
	for _, p := range []string{"claude-", "gpt-", "gemini-", "grok-"} {
		if strings.HasPrefix(m, p) && !strings.Contains(m, "not-a-real") {
			return true
		}
	}
	return false
}

type matcher struct {
	match string
	queue []Reply
}

// Fake is a stateful Anthropic-shaped server.
//
// State that persists across calls, and why each is needed:
//   - requests, IN ORDER — the ordering is what proves a retry happened, and
//     what lets a consumer assert its prompt rather than assert that some
//     function was called.
//   - a QUEUE of replies per matcher — so a sequence can be scripted: 429 then
//     success proves the SDK's retry reaches us. A single canned reply per key
//     cannot express that.
//   - matchers are ORDERED, first match wins. An earlier draft ranged over a
//     map, so two keys matching one prompt served a random reply and the test
//     flaked once in a while — the worst kind of failure, because it looks like
//     the code.
type Fake struct {
	*httptest.Server

	mu       sync.Mutex
	requests []Recorded
	matchers []matcher
	fallback Reply
	t        *testing.T
	// closing is closed at test cleanup. A stalled handler waits on it rather
	// than sleeping: httptest.Server.Close blocks on active connections, so a
	// sleeping handler turns every stall test into a 30-second cleanup hang.
	closing chan struct{}
}

// NewFake starts a fake and registers cleanup.
func NewFake(t *testing.T) *Fake {
	t.Helper()
	f := &Fake{t: t, fallback: Reply{Text: "ok"}, closing: make(chan struct{})}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(func() { close(f.closing); f.Close() })
	return f
}

// Script queues replies for requests whose prompt contains match.
//
// Appends to an existing matcher with the same key, so two Script calls extend
// one queue rather than creating a shadowed duplicate.
func (f *Fake) Script(match string, replies ...Reply) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.matchers {
		if f.matchers[i].match == match {
			f.matchers[i].queue = append(f.matchers[i].queue, replies...)
			return
		}
	}
	f.matchers = append(f.matchers, matcher{match: match, queue: replies})
}

// ServeRecorded serves a committed capture verbatim for every unmatched request.
// This is how real model output enters a test.
func (f *Fake) ServeRecorded(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fallback = Reply{Capture: name}
}

// ThenServeRecorded queues a capture behind whatever is already scripted for match.
func (f *Fake) ThenServeRecorded(match, name string) {
	f.Script(match, Reply{Capture: name})
}

// Requests returns everything received, in order.
func (f *Fake) Requests() []Recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Recorded(nil), f.requests...)
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	rec := Recorded{Path: r.URL.Path, Headers: r.Header.Clone(), Body: body}
	f.mu.Lock()
	f.requests = append(f.requests, rec)
	reply := f.next(rec.Prompt())
	f.mu.Unlock()

	// An unknown model is rejected the way the REAL proxy rejects one. Measured
	// 2026-08-22: cli-proxy-api answers 502 with
	// {"type":"error","error":{"type":"api_error","message":"unknown provider for
	// model X"}} — NOT the 400 an unknown model gets from api.anthropic.com
	// directly. Modelled here rather than invented, because the whole value of a
	// shared obligation suite is that the fake and the live service answer the
	// same way; a fake that 400s would make the suite pass here and fail there.
	if m, _ := rec.Body["model"].(string); m != "" && !knownModel(m) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"type":"error","error":{"type":"api_error","message":"unknown provider for model %s"}}`, m)
		return
	}
	if reply.Status != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.Status)
		b := reply.Body
		if b == "" {
			b = fmt.Sprintf(`{"type":"error","error":{"type":"api_error","message":"scripted %d"}}`, reply.Status)
		}
		fmt.Fprint(w, b)
		return
	}
	if rec.Streaming() {
		f.serveStream(w, reply)
		return
	}
	f.serveJSON(w, reply)
}

// next pops the queue whose key matches the prompt, else the fallback.
// Callers hold f.mu.
func (f *Fake) next(prompt string) Reply {
	for i := range f.matchers {
		m := &f.matchers[i]
		if len(m.queue) > 0 && strings.Contains(prompt, m.match) {
			r := m.queue[0]
			m.queue = m.queue[1:]
			return r
		}
	}
	return f.fallback
}

func (f *Fake) serveJSON(w http.ResponseWriter, reply Reply) {
	w.Header().Set("Content-Type", "application/json")
	if reply.Capture != "" {
		w.Write(Capture(f.t, reply.Capture))
		return
	}
	stop := reply.Stop
	if stop == "" {
		stop = "end_turn"
	}
	var blocks []map[string]any
	if !reply.NoThinking {
		// The default has a thinking block, because the live service does. A fake
		// whose default is the easy case is how a suite goes green against a
		// client that returns "" in production.
		blocks = append(blocks, map[string]any{
			"type": "thinking", "thinking": "", "signature": "scripted-signature",
		})
	}
	blocks = append(blocks, map[string]any{"type": "text", "text": reply.Text})
	json.NewEncoder(w).Encode(map[string]any{
		"type": "message", "role": "assistant", "id": "msg_fake", "model": "claude-opus-5",
		"content": blocks, "stop_reason": stop,
		"usage": map[string]any{
			"input_tokens": 10, "output_tokens": 20,
			"cache_read_input_tokens":     1902,
			"cache_creation_input_tokens": 0,
			"output_tokens_details":       map[string]any{"thinking_tokens": 5},
		},
	})
}

// serveStream replays the RECORDED frame sequence.
//
// Even the scripted variants are grounded in it: Stall truncates the real frames
// after the first delta, JunkFrame injects one bad line into them. Hand-writing
// an event sequence from memory is how a fake comes to model behaviour the
// service does not have — this capture carries a `ping` event and space-padded
// payloads that no one would have invented.
func (f *Fake) serveStream(w http.ResponseWriter, reply Reply) {
	name := reply.Capture
	if name == "" || !strings.HasSuffix(name, ".sse") {
		name = "stream-sample.sse"
	}
	frames := strings.SplitAfter(string(Capture(f.t, name)), "\n\n")

	w.Header().Set("Content-Type", "text/event-stream")
	flusher, _ := w.(http.Flusher)
	for _, fr := range frames {
		if fr == "" {
			continue
		}
		fmt.Fprint(w, fr)
		if reply.JunkFrame && strings.Contains(fr, "text_delta") {
			// AFTER a text delta: the point is that a stream dying mid-reply keeps
			// what already arrived, which a pre-content injection cannot show.
			fmt.Fprint(w, "event: content_block_delta\ndata: <<not json>>\n\n")
			reply.JunkFrame = false
		}
		if flusher != nil {
			flusher.Flush()
		}
		if reply.Stall && strings.Contains(fr, "content_block_delta") {
			// Silence without closing: what a hung upstream actually looks like.
			// The client's StallAfter (or ctx cancellation) is the only thing that
			// can end this. Waiting on `closing` rather than sleeping keeps
			// httptest.Server.Close from blocking for the sleep's duration.
			select {
			case <-f.closing:
			case <-time.After(30 * time.Second):
			}
			return
		}
	}
}
