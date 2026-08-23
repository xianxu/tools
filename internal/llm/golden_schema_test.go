package llm_test

import (
	"testing"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// vetoVerdict is a representative result type. It is not used in production —
// #12 will define its own — it exists so the derived schema has a committed
// snapshot, which is the property schema.go cites to justify reflecting the
// schema instead of hand-writing one: "a struct field added without thought shows
// up in a diff."
//
// Without a committed artifact that claim was aspirational: AssertGolden shipped
// with zero golden files anywhere in the tree.
type vetoVerdict struct {
	Fits   bool   `json:"fits"`
	Reason string `json:"reason"`
}

func TestSchemaGoldenIsStable(t *testing.T) {
	schema, err := llm.SchemaFor[vetoVerdict]()
	if err != nil {
		t.Fatal(err)
	}
	// Snapshotted through the SAME renderer a prompt golden uses, so the schema
	// and the prompt it travels with are diffed the same way.
	llmtest.AssertGolden(t, "testdata", "schema-veto-verdict", llm.Request{
		Task: "veto-distractor", Model: "claude-opus-5", Effort: "high",
		System: "You judge whether a candidate distractor would also fit a cloze blank.",
		Prompt: "Would `obsequious` also correctly fill the blank?",
		Schema: schema,
	})
}
