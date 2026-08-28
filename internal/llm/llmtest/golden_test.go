package llmtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/tools/internal/llm"
)

func req(prompt string) llm.Request {
	return llm.Request{Task: "veto", Model: "claude-opus-5", Effort: "high", Prompt: prompt}
}

// The golden must FAIL on a changed prompt — planted, observed, removed. A
// golden that cannot fail is a file, not a guard.
func TestGoldenDetectsAChangedPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "golden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "golden", "veto.txt"),
		[]byte(llm.RenderRequest(req("Is obsequious a near-synonym?"))), 0o644); err != nil {
		t.Fatal(err)
	}

	// Unchanged: passes.
	fake := substituteT(t, "", func(ft *testing.T) {
		AssertGolden(ft, dir, "veto", req("Is obsequious a near-synonym?"))
	})
	if fake.Failed() {
		t.Error("an unchanged prompt failed its golden")
	}

	// Changed: fails.
	planted := substituteT(t, "", func(ft *testing.T) {
		AssertGolden(ft, dir, "veto", req("Is ephemeral a near-synonym?"))
	})
	if !planted.Failed() {
		t.Error("a changed prompt passed its golden")
	}
}

// A missing golden names the fix rather than just failing.
//
// Run through substituteT: AssertGolden uses t.Fatalf for a missing file, and
// Fatalf calls runtime.Goexit — which would terminate THIS test rather than the
// substitute's.
//
// Golden checks do not route through the conformance guard (a missing golden is
// an absent IN-REPO artifact, which always fails), so the "" here buys
// uniformity rather than correctness — no test in this package should depend on
// ambient environment.
func TestGoldenMissingFileExplainsItself(t *testing.T) {
	dir := t.TempDir()
	fake := substituteT(t, "", func(ft *testing.T) {
		AssertGolden(ft, dir, "absent", req("x"))
	})
	if !fake.Failed() {
		t.Error("a missing golden did not fail")
	}
}

// The coupling that matters: the golden and the cassette key move TOGETHER,
// because both derive from llm.RenderRequest. If they could diverge, a prompt
// edit would show in one artifact and silently not in the other.
func TestGoldenAndCassetteKeyMoveTogether(t *testing.T) {
	a, b := req("first"), req("second")
	if llm.RenderRequest(a) == llm.RenderRequest(b) {
		t.Fatal("the render did not change")
	}
	if llm.RequestHash(a) == llm.RequestHash(b) {
		t.Error("the cassette key did not move with the prompt")
	}
}
