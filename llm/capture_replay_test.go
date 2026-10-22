package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// The tool-use captures are replayed through the production paths, not only
// text-searched: a real stream through Client.Stream, real messages through
// buildResponse and Message(). A capture that only proves its own shape cannot
// catch a decoder that mishandles it.

func captureClient(t *testing.T, name, contentType string) Client {
	t.Helper()
	data, err := os.ReadFile("llmtest/testdata/" + name)
	if err != nil {
		t.Fatalf("%s: %v (run scripts/llm-probe.sh record)", name, err)
	}
	transport := rendererTransport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{contentType}},
			Body: io.NopCloser(bytes.NewReader(data)), Request: req}, nil
	})
	return New(Config{Model: "test-model", APIKey: "test", Transport: transport})
}

func TestToolStreamCaptureReplaysThroughStream(t *testing.T) {
	c := captureClient(t, "stream-tool-use.sse", "text/event-stream")
	var text strings.Builder
	inputs := map[int]*strings.Builder{}
	fragments := map[int]int{}
	ends := map[int]ToolCall{}
	starts := 0
	resp, err := c.Stream(t.Context(), Request{Prompt: "x", Tools: []Tool{{Name: "weather"}}}, func(d Delta) {
		switch d.Kind {
		case DeltaText:
			text.WriteString(d.Text)
		case DeltaToolCallStart:
			starts++
			inputs[d.Index] = &strings.Builder{}
		case DeltaToolCallInput:
			inputs[d.Index].WriteString(d.Text)
			fragments[d.Index]++
		case DeltaToolCallEnd:
			ends[d.Index] = *d.Call
		}
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if resp.Stop != "tool_use" {
		t.Errorf("Stop = %q, want tool_use", resp.Stop)
	}
	if len(resp.ToolCalls) == 0 || starts != len(resp.ToolCalls) || len(ends) != len(resp.ToolCalls) {
		t.Fatalf("starts=%d ends=%d ToolCalls=%d, want one Start and one End per call", starts, len(ends), len(resp.ToolCalls))
	}
	byID := map[string]ToolCall{}
	for _, tc := range resp.ToolCalls {
		byID[tc.ID] = tc
	}
	for idx, end := range ends {
		if fragments[idx] < 2 {
			t.Errorf("call at index %d arrived in %d input fragments, want several", idx, fragments[idx])
		}
		if inputs[idx].String() != string(end.Input) {
			t.Errorf("fragments join to %s, End.Input = %s", inputs[idx].String(), end.Input)
		}
		if tc, ok := byID[end.ID]; !ok || !jsonEqual(t, end.Input, tc.Input) {
			t.Errorf("End %+v disagrees with Response.ToolCalls %+v", end, resp.ToolCalls)
		}
	}
	if text.String() != resp.Text {
		t.Errorf("text deltas = %q, Response.Text = %q", text.String(), resp.Text)
	}
}

func TestToolRoundCapturesReplayThroughBuildResponse(t *testing.T) {
	turn1, err := buildResponse(loadCapture(t, "message-tool-use.json"), time.Second)
	if err != nil {
		t.Fatalf("turn 1: %v", err)
	}
	if turn1.Stop != "tool_use" || len(turn1.ToolCalls) == 0 {
		t.Fatalf("turn 1 Stop=%q ToolCalls=%d, want a tool call", turn1.Stop, len(turn1.ToolCalls))
	}
	// Message() is what a caller echoes as turn 2's history: the thinking block
	// must go back verbatim (signature included) and the call must render.
	echo := string(messageJSON(turn1.Message()))
	for _, b := range turn1.Blocks {
		if b.Type != "thinking" {
			continue
		}
		var th struct {
			Signature string `json:"signature"`
		}
		if json.Unmarshal(b.Raw, &th) != nil || th.Signature == "" {
			t.Fatalf("thinking block lost its Raw signature: %s", b.Raw)
		}
		if !strings.Contains(echo, th.Signature) {
			t.Error("Message() echo drops the thinking signature")
		}
	}
	if !strings.Contains(echo, turn1.ToolCalls[0].ID) {
		t.Errorf("Message() echo drops the tool call %s: %s", turn1.ToolCalls[0].ID, echo)
	}
	if err := checkRequest(Request{Messages: []Message{{Role: RoleUser, Content: []Block{Text("q")}}, turn1.Message(),
		{Role: RoleUser, Content: []Block{ToolResult(turn1.ToolCalls[0].ID, "ok", false)}}}}); err != nil {
		t.Errorf("turn 2 built from the capture is refused: %v", err)
	}

	turn2, err := buildResponse(loadCapture(t, "message-tool-result.json"), time.Second)
	if err != nil {
		t.Fatalf("turn 2: %v", err)
	}
	if turn2.Stop != "end_turn" || turn2.Text == "" || len(turn2.ToolCalls) != 0 {
		t.Errorf("turn 2 Stop=%q Text=%q ToolCalls=%d, want a plain answer", turn2.Stop, turn2.Text, len(turn2.ToolCalls))
	}
}

func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var x, y bytes.Buffer
	if json.Compact(&x, a) != nil || json.Compact(&y, b) != nil {
		return false
	}
	return x.String() == y.String()
}
