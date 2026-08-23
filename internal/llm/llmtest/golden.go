package llmtest

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm"
)

// update rewrites goldens and re-records cassettes. One flag for both, because
// they are two views of the same request and must never be refreshed apart.
var update = flag.Bool("update", false, "rewrite golden files and cassettes")

// Updating reports whether -update was passed. Consumers that reach the live
// service to re-record read this.
func Updating() bool { return *update }

// AssertGolden compares a Request's canonical form with testdata/golden/<name>.txt.
//
// This is the mechanism the issue's "prompt regressions are visible in a diff"
// requires. The golden FILES live with the consumer that owns the prompt —
// #10's authoring prompt, #13's grading prompt — because a prompt is domain
// knowledge; this package owns only the comparison.
//
// It renders through llm.RenderRequest, the same function llm.RequestHash hashes,
// so a prompt edit cannot move one artifact without moving the other.
func AssertGolden(t *testing.T, dir, name string, r llm.Request) {
	t.Helper()
	path := filepath.Join(dir, "golden", name+".txt")
	got := llm.RenderRequest(r)

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("golden: %v", err)
		}
		t.Logf("golden: wrote %s", path)
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: %v\n\nthe prompt renders as:\n%s\nrun with -update to record it", err, got)
	}
	if string(want) != got {
		t.Errorf("prompt changed for %q.\n\n--- recorded ---\n%s\n--- now ---\n%s\n"+
			"If the change is intended, re-run with -update — and note that the cassette "+
			"key moves with it, so any recording for this request must be refreshed too.",
			name, indent(string(want)), indent(got))
	}
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n  ")
}
