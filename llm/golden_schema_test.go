package llm_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/xianxu/tools/llm"
	"github.com/xianxu/tools/llm/llmtest"
)

// vetoVerdict is a representative result type, not used in production.
//
// #10 defines the real one, at AUTHORING time rather than at review time; this
// shape predates that decision. See #12's Revisions: selection and the veto
// moved to #10, so an item reaches a sitting already finished.
//
// It exists here so the derived schema has a committed snapshot — the property
// schema.go cites to justify reflecting the schema instead of hand-writing one:
// "a struct field added without thought shows up in a diff."
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

// A single-turn request hashes exactly as it did before Messages and Tools
// existed (tools#85). The hash is tied to the COMMITTED golden's bytes, not to a
// fresh render, so a renderer change that moved every cassette key would fail
// here even if someone re-recorded the golden alongside it.
func TestSingleTurnHashUnchanged(t *testing.T) {
	schema, err := llm.SchemaFor[vetoVerdict]()
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile("testdata/golden/schema-veto-verdict.txt")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(golden)
	want := hex.EncodeToString(sum[:])[:12]
	got := llm.RequestHash(llm.Request{
		Task: "veto-distractor", Model: "claude-opus-5", Effort: "high",
		System: "You judge whether a candidate distractor would also fit a cloze blank.",
		Prompt: "Would `obsequious` also correctly fill the blank?",
		Schema: schema,
	})
	if got != want {
		t.Errorf("RequestHash = %s, want %s (the committed golden's): every cassette key moved", got, want)
	}
}
