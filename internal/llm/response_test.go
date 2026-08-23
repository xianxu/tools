package llm

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// buildResponse is pure, so it is tested directly — no httptest server, no SDK
// client, no network. That is the whole point of it not being a method: the one
// piece of real transformation logic in this package was previously reachable
// only through an HTTP round trip.
//
// The capture is read from disk rather than through llmtest, which imports this
// package (a cycle). `go test` runs in the package directory, so the relative
// path resolves — a fact this repo has paid for three times over.
func loadCapture(t *testing.T, name string) *anthropic.Message {
	t.Helper()
	raw, err := os.ReadFile("llmtest/testdata/" + name)
	if err != nil {
		t.Fatalf("%s: %v (run scripts/llm-probe.sh record)", name, err)
	}
	var msg anthropic.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return &msg
}

func TestBuildResponseJoinsTextBlocksAndSkipsThinking(t *testing.T) {
	got, err := buildResponse(loadCapture(t, "message-thinking.json"), time.Second)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Text == "" {
		t.Fatal("Text is empty")
	}
	if got.Blocks[0].Type != "thinking" {
		t.Fatalf("capture changed shape: Blocks[0] = %q", got.Blocks[0].Type)
	}
	if strings.HasPrefix(got.Text, got.Blocks[0].Text) && got.Blocks[0].Text != "" {
		t.Error("thinking content leaked into Text")
	}
	// Every text block, joined — not just the first one found.
	var want strings.Builder
	for _, b := range got.Blocks {
		if b.Type == "text" {
			want.WriteString(b.Text)
		}
	}
	if got.Text != want.String() {
		t.Errorf("Text = %q, want every text block joined (%q)", got.Text, want.String())
	}
}

func TestBuildResponsePreservesRawAndSignature(t *testing.T) {
	got, _ := buildResponse(loadCapture(t, "message-thinking.json"), time.Second)
	for _, b := range got.Blocks {
		if len(b.Raw) == 0 {
			t.Errorf("block %q has no Raw — it cannot be echoed back byte for byte", b.Type)
		}
		if b.Type == "thinking" && b.Signature == "" {
			t.Error("thinking block lost its signature")
		}
	}
}

// The truncation specimen: three blocks, thinking LAST, and an error despite a
// perfectly well-formed body.
func TestBuildResponseFlagsTruncationAndKeepsOrder(t *testing.T) {
	got, err := buildResponse(loadCapture(t, "message-truncated.json"), time.Second)
	if err == nil {
		t.Fatal("a max_tokens response returned no error")
	}
	var kinds []string
	for _, b := range got.Blocks {
		kinds = append(kinds, b.Type)
	}
	if len(kinds) != 3 || kinds[2] != "thinking" {
		t.Errorf("blocks = %v, want the trailing thinking block preserved", kinds)
	}
	if got.Text == "" {
		t.Error("the partial answer was discarded")
	}
}

func TestBuildResponseCarriesUsage(t *testing.T) {
	got, _ := buildResponse(loadCapture(t, "message-thinking.json"), 42*time.Millisecond)
	if got.Usage.Duration != 42*time.Millisecond {
		t.Errorf("Duration = %v", got.Usage.Duration)
	}
	if got.Usage.ThinkingTokens == 0 {
		t.Error("ThinkingTokens = 0; the capture has a thinking block")
	}
	if got.Usage.PreambleTokens() == 0 {
		t.Error("PreambleTokens = 0; the proxy preamble is invisible")
	}
	if got.ID == "" {
		t.Error("ID dropped")
	}
}

// New must default EVERY field of Config, not the ones that happened to come up.
//
// Asserted on the constructed client rather than on behaviour: a behavioural
// test needs a deadline to bound it, and that deadline ends the call whether or
// not the default was applied — so it cannot distinguish, and the first version
// of this test stayed green with the default deleted. In-package, this fails the
// moment any default is dropped, which is the class rather than one field.
func TestNewDefaultsEveryConfigField(t *testing.T) {
	c, ok := New(Config{BaseURL: "http://example", APIKey: "sk-test-1234567890"}).(*anthropicClient)
	if !ok {
		t.Fatal("New no longer returns *anthropicClient; update this test")
	}
	for _, f := range []struct {
		name string
		zero bool
	}{
		{"Timeout", c.cfg.Timeout == 0},
		{"StallAfter", c.cfg.StallAfter == 0},
		{"MaxTokens", c.cfg.MaxTokens == 0},
		{"Model", c.cfg.Model == ""},
		{"Effort", c.cfg.Effort == ""},
		{"BaseURL", c.cfg.BaseURL == ""},
	} {
		if f.zero {
			t.Errorf("New left %s unset — a consumer writing the minimal Config silently loses it", f.name)
		}
	}
}

// A negative StallAfter must SURVIVE New, or the documented way to disable stall
// detection is unreachable.
func TestNewPreservesADisabledStallBound(t *testing.T) {
	c := New(Config{BaseURL: "http://example", APIKey: "sk-test-1234567890", StallAfter: -1}).(*anthropicClient)
	if c.cfg.StallAfter >= 0 {
		t.Errorf("StallAfter = %v; a negative value must be preserved as 'disabled'", c.cfg.StallAfter)
	}
}
