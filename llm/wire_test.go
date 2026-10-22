package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// wire renders the params a Request would send, as decoded JSON, with no
// server: the translation is pure, so its rules are table rows.
func wire(t *testing.T, r Request) map[string]any {
	t.Helper()
	a := &anthropicClient{cfg: Config{Model: "claude-opus-5", Effort: "high", MaxTokens: 100}}
	b, err := json.Marshal(a.params(a.effective(r)))
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode params: %v\n%s", err, b)
	}
	return out
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWireMessages(t *testing.T) {
	thinking := json.RawMessage(`{"type":"thinking","thinking":"","signature":"sig-ABC"}`)
	weather := Tool{Name: "weather", Description: "look up weather",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}}}

	cases := []struct {
		name string
		req  Request
		// want maps a top-level params key to its expected JSON; "" means absent.
		want map[string]string
	}{
		{
			name: "prompt only is today's single user turn",
			req:  Request{Prompt: "hello"},
			want: map[string]string{
				"messages":    `[{"content":[{"text":"hello","type":"text"}],"role":"user"}]`,
				"tools":       "",
				"tool_choice": "",
			},
		},
		{
			name: "history then prompt, Raw sent verbatim",
			req: Request{
				Messages: []Message{
					{Role: RoleUser, Content: []Block{Text("first")}},
					{Role: RoleAssistant, Content: []Block{{Type: "thinking", Raw: thinking}, Text("answer")}},
				},
				Prompt: "second",
			},
			want: map[string]string{
				"messages": `[{"content":[{"text":"first","type":"text"}],"role":"user"},` +
					`{"content":[{"signature":"sig-ABC","thinking":"","type":"thinking"},{"text":"answer","type":"text"}],"role":"assistant"},` +
					`{"content":[{"text":"second","type":"text"}],"role":"user"}]`,
			},
		},
		{
			name: "tool use and tool result round trip",
			req: Request{
				Messages: []Message{
					{Role: RoleUser, Content: []Block{Text("weather in Oslo?")}},
					{Role: RoleAssistant, Content: []Block{{Type: "tool_use",
						ToolCall: &ToolCall{ID: "tu_1", Name: "weather", Input: json.RawMessage(`{"city":"Oslo"}`)}}}},
					{Role: RoleUser, Content: []Block{ToolResult("tu_1", "rain", false), ToolResult("tu_2", "boom", true)}},
				},
				Tools: []Tool{weather},
			},
			want: map[string]string{
				"messages": `[{"content":[{"text":"weather in Oslo?","type":"text"}],"role":"user"},` +
					`{"content":[{"id":"tu_1","input":{"city":"Oslo"},"name":"weather","type":"tool_use"}],"role":"assistant"},` +
					`{"content":[{"content":"rain","tool_use_id":"tu_1","type":"tool_result"},` +
					`{"content":"boom","is_error":true,"tool_use_id":"tu_2","type":"tool_result"}],"role":"user"}]`,
				"tools": `[{"description":"look up weather","input_schema":{"properties":{"city":{"type":"string"}},"type":"object"},"name":"weather"}]`,
			},
		},
		{
			name: "a nil input schema is an object",
			req:  Request{Prompt: "x", Tools: []Tool{{Name: "ping"}}},
			want: map[string]string{"tools": `[{"input_schema":{"type":"object"},"name":"ping"}]`},
		},
		{name: "choice auto", req: Request{Prompt: "x", Tools: []Tool{weather}, ToolChoice: "auto"},
			want: map[string]string{"tool_choice": `{"type":"auto"}`}},
		{name: "choice any", req: Request{Prompt: "x", Tools: []Tool{weather}, ToolChoice: "any"},
			want: map[string]string{"tool_choice": `{"type":"any"}`}},
		{name: "choice none", req: Request{Prompt: "x", Tools: []Tool{weather}, ToolChoice: "none"},
			want: map[string]string{"tool_choice": `{"type":"none"}`}},
		{name: "choice forces a named tool", req: Request{Prompt: "x", Tools: []Tool{weather}, ToolChoice: "weather"},
			want: map[string]string{"tool_choice": `{"name":"weather","type":"tool"}`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkRequest(tc.req); err != nil {
				t.Fatalf("checkRequest refused a valid request: %v", err)
			}
			got := wire(t, tc.req)
			for key, want := range tc.want {
				v, ok := got[key]
				if want == "" {
					if ok {
						t.Errorf("%s present (%s), want absent", key, jsonOf(t, v))
					}
					continue
				}
				if !ok {
					t.Fatalf("%s absent, want %s", key, want)
				}
				if g := jsonOf(t, v); g != want {
					t.Errorf("%s =\n  %s\nwant\n  %s", key, g, want)
				}
			}
		})
	}
}

// A thinking block's Raw reaches the wire unchanged: same keys, same order, same
// values (upstream checks the signature against its content). Only insignificant
// whitespace goes, because encoding/json compacts a RawMessage it embeds.
func TestWireRawIsByteForByte(t *testing.T) {
	raw := json.RawMessage(`{"type":"thinking", "thinking":"", "signature":"Zx==", "zeta":1}`)
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		t.Fatal(err)
	}
	got := string(messageJSON(Message{Role: RoleAssistant, Content: []Block{{Type: "thinking", Raw: raw}}}))
	if !strings.Contains(got, compact.String()) {
		t.Errorf("rendered %s, want it to contain %s verbatim", got, compact.String())
	}
}

func TestCheckRequestRefusals(t *testing.T) {
	cases := []struct {
		name string
		req  Request
	}{
		{"neither messages nor prompt", Request{}},
		{"tool named auto", Request{Prompt: "x", Tools: []Tool{{Name: "auto"}}}},
		{"tool named any", Request{Prompt: "x", Tools: []Tool{{Name: "any"}}}},
		{"tool named none", Request{Prompt: "x", Tools: []Tool{{Name: "none"}}}},
		{"tool with no name", Request{Prompt: "x", Tools: []Tool{{}}}},
		{"tool declared twice", Request{Prompt: "x", Tools: []Tool{{Name: "a"}, {Name: "a"}}}},
		{"choice names no declared tool", Request{Prompt: "x", Tools: []Tool{{Name: "a"}}, ToolChoice: "b"}},
		{"choice names a tool with none declared", Request{Prompt: "x", ToolChoice: "b"}},
		{"unknown role", Request{Messages: []Message{{Role: "system", Content: []Block{Text("x")}}}}},
		{"empty turn", Request{Messages: []Message{{Role: RoleUser}}}},
		{"tool_result without a call id", Request{Messages: []Message{{Role: RoleUser, Content: []Block{ToolResult("", "x", false)}}}}},
		{"tool_use without a call", Request{Messages: []Message{{Role: RoleAssistant, Content: []Block{{Type: "tool_use"}}}}}},
		{"unknown block type without Raw", Request{Messages: []Message{{Role: RoleUser, Content: []Block{{Type: "image"}}}}}},
		{"Raw that is not JSON", Request{Messages: []Message{{Role: RoleUser, Content: []Block{{Type: "text", Raw: json.RawMessage("{nope")}}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkRequest(tc.req); !errors.Is(err, ErrRequest) {
				t.Errorf("checkRequest = %v, want ErrRequest", err)
			}
		})
	}
}

// The single-turn path keeps the SDK's own constructor, so its params are the
// same value they were before Messages existed.
func TestWirePromptOnlyMatchesTheSDKConstructor(t *testing.T) {
	got := wireMessages(Request{Prompt: "hello"})
	want := []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("hello"))}
	if jsonOf(t, got) != jsonOf(t, want) {
		t.Errorf("prompt-only messages = %s, want %s", jsonOf(t, got), jsonOf(t, want))
	}
}
