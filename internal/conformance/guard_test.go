package conformance_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The guarantee enforces ITSELF, instead of being swept and then claimed.
//
// Four review rounds of the `check-that-cannot-fail-reads-as-green` family were
// spent on this one rule, and each round fixed the sites the previous round's
// grep happened to reach:
//
//	round 1 — one pty site fixed; six suites still skipped silently.
//	round 2 — all seven routed, enumerated by `grep 't\.Skipf\?('` — which cannot
//	          see the MIRROR defect, so three suites writing an absent dependency
//	          as an unconditional Fatalf went unnoticed.
//	round 3 — the carve-out excluded a file by NAME, hiding a committed fixture
//	          that skipped when deleted.
//	round 4 — the sweep covered cmd/define while the README claimed `./...`;
//	          measured, the documented strict command reported ok with
//	          internal/llm's suites skipped.
//
// Every one of those was a human re-running an enumeration. A grep cannot fail a
// build; this can. It is the same move that ended the `doc-sweep-incomplete`
// family — make the claim a CONSUMER of the thing it claims about.
//
// A skip is legitimate in exactly two shapes, and both are visible here:
// routed through conformance.SkipOrFail (an absent external dependency), or
// marked `conformance:inapplicable` with a reason (a table row that does not
// apply). Anything else fails, including in a package nobody thought to sweep.
func TestEverySkipIsRoutedOrWaived(t *testing.T) {
	root := repoRoot(t)

	// The router itself performs the skip every routed site delegates to.
	const router = "internal/conformance/conformance.go"

	skip := regexp.MustCompile(`\bt\.Skipf?\(|\bt\.SkipNow\(`)

	var unrouted []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "workshop", "atlas", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) == router {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(b), "\n")
		for i, line := range lines {
			if !skip.MatchString(line) {
				continue
			}
			// The waiver marker may sit on the line or in the three above it,
			// so the reason can be written as a comment over the branch.
			lo := max(0, i-3)
			if strings.Contains(strings.Join(lines[lo:i+1], "\n"), "conformance:inapplicable") {
				continue
			}
			unrouted = append(unrouted,
				filepath.ToSlash(rel)+":"+itoa(i+1)+"  "+strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the tree: %v", err)
	}

	if len(unrouted) > 0 {
		t.Errorf("%d skip site(s) neither routed through conformance.SkipOrFail nor waived:\n\t%s\n\n"+
			"Ask the site what a missing thing MEANS there:\n"+
			"  absent EXTERNAL dependency  -> conformance.SkipOrFail(t, reason, err)\n"+
			"  absent IN-REPO artifact     -> t.Fatal; a committed file that is gone is a deleted file\n"+
			"  SHAPE drift                 -> t.Fatal; that is what the check is FOR\n"+
			"  INAPPLICABLE table row      -> keep the skip, and write `conformance:inapplicable — <why>`\n"+
			"An unrouted skip makes `%s=1` report green for a check that did not run.",
			len(unrouted), strings.Join(unrouted, "\n\t"), "CONFORMANCE_STRICT")
	}
}

// repoRoot walks up to the directory holding go.mod.
//
// t.Fatal, not a skip: go.mod is committed, so its absence is a broken checkout
// rather than a dependency this machine lacks — the second of the four classes.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the working directory — broken checkout")
		}
		dir = parent
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
