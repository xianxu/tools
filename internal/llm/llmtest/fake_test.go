package llmtest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// The FLOOR: if the fake's own envelope is wrong, every later failure is
// misattributed to the client.
//
// Note what is NOT asserted — a block sequence. The committed captures disagree
// on it ([thinking,text], [text], [thinking,text,thinking]), so asserting one
// here would bake in the very mistake this fake exists to avoid. What is
// asserted is that the served bytes are IDENTICAL to the capture, which cannot
// drift from reality by construction.
func TestServesACaptureVerbatim(t *testing.T) {
	f := NewFake(t)
	f.ServeRecorded("message-thinking.json")

	body := post(t, f.URL, `{"model":"claude-opus-5","messages":[{"role":"user","content":"which also fits"}]}`)
	want := Capture(t, "message-thinking.json")
	if !bytes.Equal(bytes.TrimSpace(body), bytes.TrimSpace(want)) {
		t.Errorf("served body is not the capture\n got %d bytes\nwant %d bytes", len(body), len(want))
	}

	reqs := f.Requests()
	if len(reqs) != 1 {
		t.Fatalf("%d requests recorded, want 1", len(reqs))
	}
	if got := reqs[0].Prompt(); got != "which also fits" {
		t.Errorf("Prompt() = %q", got)
	}
	if reqs[0].Path != "/v1/messages" {
		t.Errorf("Path = %q", reqs[0].Path)
	}
}

// A scripted reply's default shape includes a thinking block, because that is
// what the live service does. A fake whose default is the easy case is how a
// suite goes green against a client that returns "" in production.
func TestScriptedReplyIncludesThinkingByDefault(t *testing.T) {
	f := NewFake(t)
	f.Script("hello", Reply{Text: "answer"})

	var msg struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(post(t, f.URL, `{"messages":[{"role":"user","content":"hello"}]}`), &msg); err != nil {
		t.Fatal(err)
	}
	if len(msg.Content) != 2 || msg.Content[0].Type != "thinking" {
		t.Fatalf("content = %+v, want a leading thinking block", msg.Content)
	}
	if msg.Content[1].Text != "answer" {
		t.Errorf("text block = %q", msg.Content[1].Text)
	}
}

// Matchers are ordered and first-match-wins. With a map this was nondeterministic
// when two keys matched one prompt, and the test flaked rarely enough to look
// like a bug in the code under test.
func TestOverlappingMatchersAreDeterministic(t *testing.T) {
	for i := 0; i < 20; i++ {
		f := NewFake(t)
		f.Script("syco", Reply{Text: "first"})
		f.Script("sycophantic", Reply{Text: "second"})
		body := post(t, f.URL, `{"messages":[{"role":"user","content":"sycophantic"}]}`)
		if !strings.Contains(string(body), "first") {
			t.Fatalf("run %d served the later matcher; ordering is not deterministic", i)
		}
	}
}

// A queue expresses a SEQUENCE — 429 then success — which one canned reply per
// key cannot.
func TestQueueServesInOrder(t *testing.T) {
	f := NewFake(t)
	f.Script("hello", Reply{Status: 429}, Reply{Text: "second try"})

	if code := postStatus(t, f.URL, `{"messages":[{"role":"user","content":"hello"}]}`); code != 429 {
		t.Fatalf("first call = %d, want 429", code)
	}
	if body := post(t, f.URL, `{"messages":[{"role":"user","content":"hello"}]}`); !strings.Contains(string(body), "second try") {
		t.Errorf("second call = %s", body)
	}
}

// The stream path replays the RECORDED frames, which carry a ping event and
// space-padded payloads nobody would have invented.
func TestStreamReplaysRecordedFrames(t *testing.T) {
	f := NewFake(t)
	body := string(post(t, f.URL, `{"stream":true,"messages":[{"role":"user","content":"x"}]}`))
	for _, want := range []string{"message_start", "ping", "thinking_delta", "signature_delta", "text_delta", "message_stop"} {
		if !strings.Contains(body, want) {
			t.Errorf("stream is missing %q — it is not the recorded capture", want)
		}
	}
}

func post(t *testing.T, url, body string) []byte {
	t.Helper()
	resp, err := http.Post(url+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return buf.Bytes()
}

func postStatus(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url+"/v1/messages", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// The fake rejects an unknown model the way the REAL proxy does — 502 with an
// api_error body, measured 2026-08-22, NOT the 400 api.anthropic.com returns
// directly. A fake that answered 400 would make the shared obligation suite pass
// here and fail against the live service, which is the divergence the suite
// exists to catch.
func TestUnknownModelIsRejectedLikeTheProxyDoes(t *testing.T) {
	f := NewFake(t)
	// The fixtures are ORDINARY typos, not strings carrying a magic substring.
	// An earlier fixture was "claude-not-a-real-model", which the pre-fix prefix
	// rule special-cased by name — so reverting that rule left this test green
	// while the fake answered 200 for every other typo.
	for _, bad := range []string{"claude-opus-6", "claude-sonnet-9", "gpt-5.7-imaginary"} {
		code := postStatus(t, f.URL, `{"model":"`+bad+`","messages":[{"role":"user","content":"hi"}]}`)
		if code != 502 {
			t.Errorf("%s: status = %d, want 502 (what cli-proxy-api answers)", bad, code)
		}
	}
	if code := postStatus(t, f.URL, `{"model":"claude-opus-5","messages":[{"role":"user","content":"hi"}]}`); code != 200 {
		t.Errorf("a real model got %d", code)
	}
}
