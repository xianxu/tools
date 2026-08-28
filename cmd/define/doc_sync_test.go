package main

import (
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
