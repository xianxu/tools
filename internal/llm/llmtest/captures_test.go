package llmtest

import (
	"encoding/json"
	"strings"
	"testing"
)

// Every capture must EXHIBIT the shape it was recorded to demonstrate — enforced
// under `go test`, offline, on every run.
//
// scripts/llm-probe.sh has the same declarations, but nothing runs that script
// and it needs the proxy config to start. Given that "a capture recorded under
// conditions that do not elicit the shape the fake models" produced three
// separate plan-gate findings, the durable form of the guard is this one: the
// captures are already embedded, so it costs nothing and cannot be skipped.
//
// These assert SHAPE, never content — the text differs every re-record.
func TestCapturesExhibitTheirShapes(t *testing.T) {
	// RULE: a test helper reading a fixture must FAIL, never panic. An unchecked
	// type assertion on a malformed capture aborts the whole package run and
	// masks every other result — the same class as indexing Blocks[0] after
	// discarding an error.
	blocks := func(t *testing.T, name string) ([]string, map[string]any) {
		t.Helper()
		var m map[string]any
		if err := json.Unmarshal(Capture(t, name), &m); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if m["type"] == "error" {
			t.Fatalf("%s is an upstream error envelope, not a capture — re-record it", name)
		}
		content, ok := m["content"].([]any)
		if !ok {
			t.Fatalf("%s has no content array", name)
		}
		var kinds []string
		for i, c := range content {
			cm, ok := c.(map[string]any)
			if !ok {
				t.Fatalf("%s: content[%d] is not an object", name, i)
			}
			kind, ok := cm["type"].(string)
			if !ok {
				t.Fatalf("%s: content[%d] has no type", name, i)
			}
			kinds = append(kinds, kind)
		}
		return kinds, m
	}
	textOf := func(t *testing.T, m map[string]any) string {
		t.Helper()
		var b strings.Builder
		content, _ := m["content"].([]any)
		for _, c := range content {
			cm, ok := c.(map[string]any)
			if !ok || cm["type"] != "text" {
				continue
			}
			if s, ok := cm["text"].(string); ok {
				b.WriteString(s)
			}
		}
		return b.String()
	}

	t.Run("message-thinking has a thinking block and completed", func(t *testing.T) {
		kinds, m := blocks(t, "message-thinking.json")
		if !contains(kinds, "thinking") {
			t.Errorf("blocks = %v, want a thinking block — the prompt was too easy, and a "+
				"capture without one cannot show that content[0] is not the text", kinds)
		}
		if m["stop_reason"] != "end_turn" {
			t.Errorf("stop_reason = %v, want a healthy end_turn", m["stop_reason"])
		}
	})

	t.Run("message-schema completed and decodes", func(t *testing.T) {
		_, m := blocks(t, "message-schema.json")
		if m["stop_reason"] != "end_turn" {
			t.Errorf("stop_reason = %v — raise max_tokens; thinking shares the budget", m["stop_reason"])
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(textOf(t, m))), &out); err != nil {
			t.Errorf("payload does not decode: %v — this capture exists to show output_config works", err)
		}
	})

	t.Run("message-truncated is still a truncation that PARSES", func(t *testing.T) {
		_, m := blocks(t, "message-truncated.json")
		if m["stop_reason"] != "max_tokens" {
			t.Fatalf("stop_reason = %v — this preserved specimen must not be re-recorded", m["stop_reason"])
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(textOf(t, m))), &out); err != nil {
			t.Errorf("payload no longer parses: %v — the whole point is that a truncated "+
				"answer decodes cleanly, which is why only stop_reason can catch it", err)
		}
	})

	// The three captures must DISAGREE on block shape. If they ever converge, no
	// test can catch an implementation that assumes a fixed sequence — which is
	// the mistake two separate plan rounds made.
	t.Run("the capture set exhibits more than one block shape", func(t *testing.T) {
		seen := map[string]bool{}
		for _, n := range []string{"message-thinking.json", "message-schema.json", "message-truncated.json"} {
			kinds, _ := blocks(t, n)
			seen[strings.Join(kinds, ",")] = true
		}
		if len(seen) < 2 {
			t.Errorf("all captures share one block shape %v — nothing can catch a "+
				"fixed-sequence assumption any more; re-record for variety", seen)
		}
	})

	t.Run("the stream carries thinking and signature deltas", func(t *testing.T) {
		sse := string(Capture(t, "stream-sample.sse"))
		for _, want := range []string{
			"message_start", "content_block_delta", "message_stop",
			"thinking_delta", "signature_delta", "text_delta",
		} {
			if !strings.Contains(sse, want) {
				t.Errorf("stream-sample.sse has no %q — a Stream that drops thinking "+
					"blocks would pass the whole suite (run scripts/llm-probe.sh record)", want)
			}
		}
	})
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}
