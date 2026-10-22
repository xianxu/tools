package llmtest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/xianxu/tools/llm"
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

// The last scripted reply is sticky. Without it, scripting one 503 meant attempt
// 1 saw the 503 and the SDK's retry saw the fallback's success — so a test that
// scripted an outage observed a decode error instead.
func TestTheLastScriptedReplyIsSticky(t *testing.T) {
	f := NewFake(t)
	f.Script("hello", Reply{Status: 503})
	for i := 0; i < 3; i++ {
		if code := postStatus(t, f.URL, `{"model":"claude-opus-5","messages":[{"role":"user","content":"hello"}]}`); code != 503 {
			t.Fatalf("request %d = %d, want a sticky 503", i+1, code)
		}
	}
}

// …but a queue with more than one entry still advances, or a scripted SEQUENCE
// (429 then success, which is how the retry is proven) would never reach its
// second entry.
func TestAQueueStillAdvancesWhileItHasEntries(t *testing.T) {
	f := NewFake(t)
	f.Script("hello", Reply{Status: 429}, Reply{Text: "recovered"})
	if code := postStatus(t, f.URL, `{"model":"claude-opus-5","messages":[{"role":"user","content":"hello"}]}`); code != 429 {
		t.Fatalf("first = %d, want 429", code)
	}
	body := post(t, f.URL, `{"model":"claude-opus-5","messages":[{"role":"user","content":"hello"}]}`)
	if !strings.Contains(string(body), "recovered") {
		t.Errorf("second = %s, want the queue to have advanced", body)
	}
}

// A tool-call reply has the protocol's shape on both paths: tool_use blocks after
// the text with stop_reason tool_use, and on a stream each input cut into
// several input_json_delta frames. A fake serving one fragment per call could
// not catch a client that parses a fragment as the whole input.
func TestFakeToolCalls(t *testing.T) {
	calls := []llm.ToolCall{{ID: "tu_1", Name: "weather", Input: json.RawMessage(`{"city":"Oslo"}`)}, {ID: "tu_2", Name: "ping"}}
	f := NewFake(t)
	f.Script("weather", Reply{Text: "Checking.", ToolCalls: calls})

	var msg map[string]any
	if err := json.Unmarshal(post(t, f.URL, `{"messages":[{"role":"user","content":"weather?"}]}`), &msg); err != nil {
		t.Fatal(err)
	}
	if msg["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason = %v", msg["stop_reason"])
	}
	var types []string
	for _, b := range msg["content"].([]any) {
		types = append(types, b.(map[string]any)["type"].(string))
	}
	if strings.Join(types, ",") != "thinking,text,tool_use,tool_use" {
		t.Errorf("blocks = %v", types)
	}

	body := string(post(t, f.URL, `{"stream":true,"messages":[{"role":"user","content":"weather?"}]}`))
	if n := strings.Count(body, "input_json_delta"); n < 4 {
		t.Errorf("%d input_json_delta frames for two calls, want each input in several", n)
	}
	for _, want := range []string{`"type":"tool_use"`, `"stop_reason":"tool_use"`, "message_stop", `"text":"Checking."`} {
		if !strings.Contains(body, want) {
			t.Errorf("tool stream is missing %s", want)
		}
	}
}

// The next reply after a tool round is scripted on the tool result's content.
func TestFakeMatchesToolResult(t *testing.T) {
	f := NewFake(t)
	f.Script("rain in Oslo", Reply{Text: "Bring a coat."})
	body := post(t, f.URL, `{"messages":[{"role":"user","content":"weather?"},`+
		`{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"weather","input":{}}]},`+
		`{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu_1","content":"rain in Oslo"}]}]}`)
	if !strings.Contains(string(body), "Bring a coat.") {
		t.Errorf("tool-result turn did not match: %s", body)
	}
}

func TestRecordedAccessors(t *testing.T) {
	f := NewFake(t)
	post(t, f.URL, `{"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"text","text":"yo"}]}],`+
		`"tools":[{"name":"weather","input_schema":{"type":"object"}}],"tool_choice":{"type":"tool","name":"weather"}}`)
	r := f.Requests()[0]
	msgs := r.Messages()
	if len(msgs) != 2 || msgs[0].Role != "user" || msgs[0].Content[0]["text"] != "hi" || msgs[1].Content[0]["text"] != "yo" {
		t.Errorf("Messages() = %+v", msgs)
	}
	if len(r.Tools()) != 1 || r.Tools()[0]["name"] != "weather" {
		t.Errorf("Tools() = %v", r.Tools())
	}
	if r.ToolChoice()["name"] != "weather" {
		t.Errorf("ToolChoice() = %v", r.ToolChoice())
	}
	if f.Attempts() != 1 {
		t.Errorf("Attempts() = %d", f.Attempts())
	}
}
