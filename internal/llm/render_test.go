package llm

import (
	"strings"
	"testing"
)

func schemaFixture() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"zebra":   map[string]any{"type": "string"},
			"apple":   map[string]any{"type": "boolean"},
			"mango":   map[string]any{"type": "integer"},
			"kiwi":    map[string]any{"type": "string"},
			"banana":  map[string]any{"type": "number"},
			"quince":  map[string]any{"type": "string"},
			"lychee":  map[string]any{"type": "string"},
			"pomelo":  map[string]any{"type": "string"},
			"tangelo": map[string]any{"type": "string"},
		},
		"required": []string{"zebra", "apple"},
	}
}

// The property the whole cassette scheme rests on: a Go map has no iteration
// order, so an unsorted render would hash differently run to run and every
// cassette would miss at random. Nine keys, so a chance pass is unlikely.
func TestRenderIsStableUnderMapIterationOrder(t *testing.T) {
	r := Request{Task: "t", Model: "claude-opus-5", Prompt: "p", Schema: schemaFixture()}
	first := RequestHash(r)
	for i := 0; i < 50; i++ {
		// A fresh map each time: Go randomises iteration order per map instance,
		// so re-hashing the SAME map would not exercise this.
		r.Schema = schemaFixture()
		if got := RequestHash(r); got != first {
			t.Fatalf("hash changed on iteration %d: %s != %s", i, got, first)
		}
	}
}

// Golden and Cassette must move TOGETHER. The coupling is the requirement, so it
// is asserted rather than trusted: if they could diverge, a prompt edit would
// show in one artifact and not the other.
func TestAPromptEditMovesBothTheRenderAndTheHash(t *testing.T) {
	a := Request{Task: "veto", Model: "claude-opus-5", Prompt: "Is obsequious a near-synonym?"}
	b := a
	b.Prompt = "Is ephemeral a near-synonym?"

	if RenderRequest(a) == RenderRequest(b) {
		t.Error("the rendered form did not change with the prompt")
	}
	if RequestHash(a) == RequestHash(b) {
		t.Error("the hash did not change with the prompt")
	}
}

// Every field that changes what was ASKED must reach the hash. A field that
// silently does not is a cassette that keeps matching after a real change.
func TestEveryMeaningfulFieldReachesTheHash(t *testing.T) {
	base := Request{
		Task: "t", Model: "claude-opus-5", Effort: "high",
		System: "sys", Prompt: "prompt", Schema: schemaFixture(),
	}
	for _, c := range []struct {
		name   string
		mutate func(*Request)
	}{
		{"Task", func(r *Request) { r.Task = "other" }},
		{"Model", func(r *Request) { r.Model = "claude-sonnet-5" }},
		{"Effort", func(r *Request) { r.Effort = "max" }},
		{"System", func(r *Request) { r.System = "different" }},
		{"Prompt", func(r *Request) { r.Prompt = "different" }},
		{"Schema", func(r *Request) { r.Schema = map[string]any{"type": "string"} }},
	} {
		got := base
		c.mutate(&got)
		if RequestHash(got) == RequestHash(base) {
			t.Errorf("%s does not reach the hash — a cassette would survive changing it", c.name)
		}
	}
}

// MaxTokens deliberately does NOT reach the hash: it changes how much room the
// answer had, not what was asked, and including it would invalidate every
// cassette the moment a default moved.
func TestMaxTokensDoesNotReachTheHash(t *testing.T) {
	a := Request{Task: "t", Prompt: "p", MaxTokens: 1024}
	b := a
	b.MaxTokens = 8192
	if RequestHash(a) != RequestHash(b) {
		t.Error("MaxTokens reaches the hash; a default change would invalidate every cassette")
	}
}

func TestRenderIsHumanReadable(t *testing.T) {
	got := RenderRequest(Request{Task: "veto", Model: "m", Effort: "high", System: "s", Prompt: "p"})
	for _, want := range []string{"task:   veto", "--- system ---", "--- prompt ---", "--- schema ---", "(none)"} {
		if !strings.Contains(got, want) {
			t.Errorf("render is missing %q:\n%s", want, got)
		}
	}
}
