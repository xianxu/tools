package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

type rendererTransport func(*http.Request) (*http.Response, error)

func (f rendererTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProviderRenderingReachesWireAndHash(t *testing.T) {
	for _, provider := range []string{"", "anthropic", "openai", "antigravity"} {
		for _, stream := range []bool{false, true} {
			for _, effort := range []string{"low", "high"} {
				t.Run(provider+"/"+effort+"/stream="+map[bool]string{false: "false", true: "true"}[stream], func(t *testing.T) {
					schema := map[string]any{"type": "object", "properties": map[string]any{"answer": map[string]any{"type": "string"}}, "required": []any{"answer"}}
					before, _ := json.Marshal(schema)
					r := Request{Task: "wire", System: "Domain rules", Prompt: "Answer the question", Schema: schema, Effort: effort}
					calls := 0
					transport := rendererTransport(func(req *http.Request) (*http.Response, error) {
						calls++
						var wire struct {
							Model  string `json:"model"`
							System []struct {
								Text string `json:"text"`
							} `json:"system"`
							Messages []struct {
								Content []struct {
									Text string `json:"text"`
								} `json:"content"`
							} `json:"messages"`
							Thinking     map[string]any `json:"thinking"`
							OutputConfig struct {
								Effort string `json:"effort"`
								Format struct {
									Schema map[string]any `json:"schema"`
								} `json:"format"`
							} `json:"output_config"`
						}
						if err := json.NewDecoder(req.Body).Decode(&wire); err != nil {
							t.Fatal(err)
						}
						effective, ok := RequestFromContext(req.Context())
						if !ok {
							t.Fatal("missing effective request")
						}
						system := ""
						for _, b := range wire.System {
							system += b.Text
						}
						if system != effective.System || wire.Messages[0].Content[0].Text != effective.Prompt {
							t.Error("hashed request differs from HTTP prompt/system")
						}
						reconstructed := effective
						reconstructed.System = system
						reconstructed.Prompt = wire.Messages[0].Content[0].Text
						reconstructed.Model = wire.Model
						reconstructed.Effort = wire.OutputConfig.Effort
						reconstructed.Schema = wire.OutputConfig.Format.Schema
						if RequestHash(reconstructed) != RequestHash(effective) {
							t.Error("wire request hash mismatch")
						}
						nonClaude := provider == "openai" || provider == "antigravity"
						if nonClaude {
							if !strings.Contains(system, "JSON") || !strings.Contains(system, renderSchema(schema)) {
								t.Errorf("missing JSON/schema instructions on wire: %q", system)
							}
							if wire.Thinking["type"] != "adaptive" {
								t.Errorf("thinking = %v, want adaptive", wire.Thinking)
							}
							if !strings.Contains(RenderRequest(effective), "thinking: adaptive\n") {
								t.Error("thinking missing from hash renderer")
							}
						} else {
							if system != r.System || wire.Thinking != nil {
								t.Error("direct Claude rendering changed")
							}
							if strings.Contains(RenderRequest(effective), "thinking:") {
								t.Error("legacy renderer gained thinking line")
							}
						}
						if wire.OutputConfig.Effort != effort {
							t.Error("effort not sent")
						}
						if !reflect.DeepEqual(wire.OutputConfig.Format.Schema, schema) {
							t.Error("schema not sent intact")
						}
						name, ct := "message-thinking.json", "application/json"
						if stream {
							name, ct = "stream-sample.sse", "text/event-stream"
						}
						data, err := os.ReadFile("llmtest/testdata/" + name)
						if err != nil {
							t.Fatal(err)
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{ct}}, Body: io.NopCloser(strings.NewReader(string(data))), Request: req}, nil
					})
					c := New(Config{Model: "test-model", APIKey: "test", provider: provider, Transport: transport})
					var err error
					if stream {
						_, err = c.Stream(t.Context(), r, nil)
					} else {
						_, err = c.Complete(t.Context(), r)
					}
					if err != nil {
						t.Fatal(err)
					}
					if calls != 1 {
						t.Fatalf("calls=%d", calls)
					}
					after, _ := json.Marshal(schema)
					if string(before) != string(after) || r.System != "Domain rules" {
						t.Error("caller request mutated")
					}
				})
			}
		}
	}
}

func FuzzEffectiveSchema(f *testing.F) {
	f.Add([]byte(`{"type":"object","properties":{"x":{"type":"string"}}}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var schema map[string]any
		if json.Unmarshal(data, &schema) != nil || schema == nil {
			return
		}
		before, _ := json.Marshal(schema)
		a := anthropicClient{cfg: Config{provider: "openai"}}
		r := a.effective(Request{System: "domain", Schema: schema})
		if !strings.Contains(r.System, renderSchema(schema)) {
			t.Fatal("schema absent")
		}
		after, _ := json.Marshal(schema)
		if string(before) != string(after) {
			t.Fatal("schema mutated")
		}
		if RequestHash(r) != RequestHash(a.effective(Request{System: "domain", Schema: schema})) {
			t.Fatal("unstable rendering")
		}
	})
}

func TestEffectiveProviderRenderingIsIdempotent(t *testing.T) {
	a := anthropicClient{cfg: Config{provider: "openai", Model: "gpt-5.6", Effort: "high", MaxTokens: 8192}}
	r := a.effective(Request{System: "domain", Prompt: "question", Schema: map[string]any{"type": "object"}})
	if got := a.effective(r); !reflect.DeepEqual(got, r) {
		t.Fatal("repeated effective rendering duplicates instructions")
	}
}
