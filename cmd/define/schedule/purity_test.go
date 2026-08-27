package schedule_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The purity claim, ENFORCED.
//
// The plan's first draft said "a test needing a fake would not compile", which is
// false — a Go test file may import anything, and nothing about the language
// stops a production file here from reaching a clock or a disk. This package's
// whole value is that every scheduling question is a pure function of explicit
// inputs, and that claim was going to live in a doc comment.
//
// This repo has now been bitten repeatedly by facts that lived only in comments;
// the fix each time was a test. This is that test.
func TestScheduleImportsOnlyStoreAndTime(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`,
		"github.com/xianxu/tools/cmd/define/schedule").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	// The claim is "no IO and no hidden clock", not "exactly two imports" — and
	// the first version of this allowlist said the latter, then fired on `sort`,
	// which is as pure as arithmetic. A guard that reddens on correct code is a
	// liability: the tempting response is to delete it.
	//
	// Additions must be genuinely pure. `fmt` is NOT (it writes to streams),
	// `os`, `net`, `bufio`, `io` are not, and anything that can name a file is
	// not.
	allowed := map[string]bool{
		"time":   true, // the TYPE; time.Now is banned separately below
		"sort":   true,
		"slices": true,
		"cmp":    true,
		"github.com/xianxu/tools/cmd/define/store": true,
	}
	var got []string
	for _, imp := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if imp = strings.TrimSpace(imp); imp == "" {
			continue
		}
		got = append(got, imp)
		if !allowed[imp] {
			t.Errorf("schedule imports %q — this package must stay pure; "+
				"anything that needs IO belongs in its caller", imp)
		}
	}

	// Guard against the guard passing vacuously: if go list ever returns
	// nothing, an empty import set would satisfy every assertion above.
	if len(got) == 0 {
		t.Fatal("go list reported no imports at all — this test would assert nothing")
	}
}

// The hazard an import list cannot see.
//
// `time` is legitimately imported for the TYPE, so its presence proves nothing —
// but `time.Now()` inside this package would be a clock the caller cannot
// control, which is the one thing that would make every scheduling question
// untestable. Every `now` here arrives as a parameter.
func TestScheduleNeverReadsTheClock(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{.Dir}}`,
		"github.com/xianxu/tools/cmd/define/schedule").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	dir := strings.TrimSpace(string(out))

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		checked++
		// EVERY wall-clock reader, not one spelling. The first version grepped
		// only "time.Now(" — and time.Since, time.Until, time.After and the timer
		// constructors all read the same clock, so the guard covered one door in
		// a room with six.
		for _, banned := range []string{
			"time.Now(", "time.Since(", "time.Until(",
			"time.After(", "time.Tick(", "time.NewTimer(", "time.NewTicker(",
		} {
			if strings.Contains(string(b), banned) {
				t.Errorf("%s calls %s — every instant in this package must arrive as a parameter, "+
					"or the caller cannot control the date and nothing here is testable", name, banned)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no non-test source files found — this test would assert nothing")
	}
}

// The hole the other two guards leave open.
//
// The import allowlist admits `store` WHOLESALE — but `store` is where the disk
// lives. `store.NewYAML(dir, w)` inside this package would open a directory,
// read files and write them, and it would sail past both the import check
// (store is allowed) and the clock check (it names no time function). The purity
// claim would be false while every guard stayed green.
//
// So this guard is about SYMBOLS rather than packages: schedule may use store's
// pure types and helpers, and nothing that can reach a disk. A new symbol has to
// be added here deliberately, which is the point — the decision becomes visible
// instead of implicit.
func TestScheduleUsesOnlyPureStoreSymbols(t *testing.T) {
	out, err := exec.Command("go", "list", "-f", `{{.Dir}}`,
		"github.com/xianxu/tools/cmd/define/schedule").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	dir := strings.TrimSpace(string(out))

	// Data shapes and pure functions. Deliberately absent: Store (its methods do
	// IO), NewYAML, NewMem, SystemClock — anything that opens or constructs a
	// backing store.
	pure := map[string]bool{
		"Key": true, "Word": true, "ReviewEvent": true,
		"EventReviewed": true, "EventLookedUp": true, "EventAsked": true,
		"StartOfDay": true, "DaysBetween": true,
	}
	ref := regexp.MustCompile(`\bstore\.([A-Za-z_][A-Za-z0-9_]*)`)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	found := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			// Comments legitimately name store.Store when explaining the ordering
			// contract; it is CODE that must not reach it.
			if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "//") {
				continue
			}
			for _, m := range ref.FindAllStringSubmatch(line, -1) {
				found++
				if !pure[m[1]] {
					t.Errorf("%s uses store.%s — schedule may use store's pure types and helpers only; "+
						"anything that can reach a disk belongs in the caller", name, m[1])
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no store references found at all — this test would assert nothing")
	}
}
