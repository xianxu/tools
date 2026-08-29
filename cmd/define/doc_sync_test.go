package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The README's account of the play loop DERIVES from the loop, or it drifts.
//
// This is the general fix for the `doc-sweep-incomplete` family, which reached
// three findings on this one screen before anyone wrote a test: the grade-first
// reversal landed in README and atlas but not the form's doc comments (BR-1),
// then in the doc comments but not the two test citations, and the README's
// audio sentence described a fetch that a `y` no longer performs (BR-3). Every
// instance was found by a human re-reading prose, and every fix was another
// sweep — so the next edit starts the cycle again.
//
// A grep cannot fail a build. This can: the prompt lines are consts in
// play_loop.go, and the README is now a CONSUMER of them rather than a
// restatement. Change what the learner is told and this test names the doc that
// has not caught up.
//
// It is deliberately narrow. It pins the two lines the learner reads off the
// screen and types against — not the surrounding prose, which is explanation and
// should be free to be rewritten.
func TestREADMEQuotesThePromptsTheLoopActuallyPrints(t *testing.T) {
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		// NOT a skip: the README is in the repo, so an unreadable one is a
		// broken checkout or a moved file, never an absent dependency.
		t.Fatalf("README.md unreadable: %v", err)
	}
	readme := string(b)

	for _, tc := range []struct {
		name, line string
	}{
		{"the grading prompt, shown while a verdict is owed", gradePrompt},
		{"the graded prompt, shown once the answer is up", gradedPrompt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(readme, tc.line) {
				t.Errorf("README.md does not contain the line draw() prints:\n\t%q\n"+
					"The loop changed and the README did not. Update README.md, or "+
					"change the const in play_loop.go if the new wording is the "+
					"intended one.", tc.line)
			}
		})
	}
}

// The atlas's raw-notation count DERIVES from the ratchet rather than restating
// it.
//
// Same move as the prompt test above, for the same reason and with a longer
// record behind it: this number has drifted three times. `#26` found it stated
// as 27 in the atlas and again in a parse.go comment while the measured value
// was 26, and the atlas additionally attributed the whole population to one
// cause when it is four.
//
// The marker comment is what makes it machine-checkable without pinning the
// surrounding prose, which should stay free to be rewritten — the narrowness is
// deliberate, exactly as it is for the play-loop prompts.
func TestAtlasQuotesTheRawNotationCount(t *testing.T) {
	b, err := os.ReadFile("../../atlas/define.md")
	if err != nil {
		// NOT a skip: the atlas is in the repo, so an unreadable one is a broken
		// checkout or a moved file, never an absent dependency.
		t.Fatalf("atlas/define.md unreadable: %v", err)
	}
	want := fmt.Sprintf("<!-- raw-notation-count -->%d<!-- /raw-notation-count -->", knownRawNotationEntries)
	if !strings.Contains(string(b), want) {
		t.Errorf("atlas/define.md does not quote the pinned raw-notation count.\n"+
			"want the marked span to read %q — the ratchet owns this number, the doc consumes it.\n"+
			"If the count moved, knownRawByCause moved first and the atlas follows.", want)
	}
}
