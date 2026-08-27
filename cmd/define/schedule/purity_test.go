package schedule_test

import (
	"os/exec"
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

	allowed := map[string]bool{
		"time": true,
		"github.com/xianxu/tools/cmd/define/store": true,
	}
	var got []string
	for _, imp := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if imp = strings.TrimSpace(imp); imp == "" {
			continue
		}
		got = append(got, imp)
		if !allowed[imp] {
			t.Errorf("schedule imports %q — this package must stay pure over store and time; "+
				"anything that needs IO belongs in its caller", imp)
		}
	}

	// Guard against the guard passing vacuously: if go list ever returns
	// nothing, an empty import set would satisfy every assertion above.
	if len(got) == 0 {
		t.Fatal("go list reported no imports at all — this test would assert nothing")
	}
}
