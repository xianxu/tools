package llm_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/xianxu/tools/llm"
	"github.com/xianxu/tools/llm/llmtest"
)

// The obligation suite against the fake. The same suite runs against the live
// proxy under `-tags conformance` (conformance_test.go), which is what makes
// "the fake behaves like the real thing" a test rather than a claim.
func TestSuiteAgainstTheFake(t *testing.T) {
	llmtest.Suite(t, func(t *testing.T) llm.Client {
		f := llmtest.NewFake(t)
		// Content the suite's obligations need. Shape comes from the captures;
		// the schema case needs a decodable payload, which is a transport shape
		// (does output_config round-trip) rather than a judgment.
		f.Script("Name one synonym", llmtest.Reply{Text: "obsequious"})
		f.Script("Which is closer", llmtest.Reply{Capture: "message-thinking.json"})
		f.Script("Is obsequious a near-synonym", llmtest.Reply{Capture: "message-schema.json"})
		f.Script("Name one synonym for happy", llmtest.Reply{Text: "joyful"})
		f.Script("Name one more", llmtest.Reply{Text: "cheerful"})
		// Held after its first frame, so a cancel on that frame always lands
		// mid-stream, as it does against a long live answer.
		f.Script(llmtest.LongPrompt, llmtest.Reply{ToolCalls: []llm.ToolCall{{ID: "tu_long", Name: "echo",
			Input: json.RawMessage(`{"n":0}`)}}, Text: "1. Rivers run.", FinishRelease: make(chan struct{})})
		for n := 0; n < llmtest.ConcurrentStreams; n++ {
			r := llmtest.Reply{ToolCalls: []llm.ToolCall{{ID: fmt.Sprintf("tu_%d", n), Name: "echo",
				Input: json.RawMessage(fmt.Sprintf(`{"n":%d}`, n))}}}
			if n == llmtest.ConcurrentVictim {
				r.FinishRelease = make(chan struct{})
			}
			f.Script(llmtest.EchoPrompt(n), r)
		}
		return llm.New(llm.Config{
			BaseURL: f.URL, APIKey: "sk-test-1234567890", Model: "claude-opus-5",
			Effort: "high", MaxTokens: 8192, Timeout: 30 * time.Second,
			StallAfter: 10 * time.Second,
		})
	})
}
