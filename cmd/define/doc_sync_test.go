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
	atlas := string(b)
	want := fmt.Sprintf("<!-- raw-notation-count -->%d<!-- /raw-notation-count -->", knownRawNotationEntries)
	if !strings.Contains(atlas, want) {
		t.Errorf("atlas/define.md does not quote the pinned raw-notation count.\n"+
			"want the marked span to read %q — the ratchet owns this number, the doc consumes it.\n"+
			"If the count moved, knownRawByCause moved first and the atlas follows.", want)
	}
	// EVERY CAUSE, not just the total. Marking only the total let a compensating
	// swap — one cause up, another down — keep the whole suite green while both
	// atlas numbers were wrong, which is the same weakness the per-cause pin was
	// added to close in the code.
	for _, c := range rawCauses {
		if c == causeUnclassified {
			continue // the live run asserts it is zero; it is not a doc figure
		}
		want := fmt.Sprintf("<!-- raw:%s -->%d<!-- /raw:%s -->", c, knownRawByCause[c], c)
		if !strings.Contains(atlas, want) {
			t.Errorf("atlas/define.md does not quote the pinned count for %s.\n"+
				"want the marked span to read %q", c, want)
		}
		// EVERY occurrence, not the first. The count is stated twice for
		// prose-numeral — once in the breakdown and once in the Limits entry —
		// and marking only one leaves the other an unmarked restatement that
		// strings.Contains cannot see.
		open := fmt.Sprintf("<!-- raw:%s -->", c)
		if n, m := strings.Count(atlas, open), strings.Count(atlas, want); n != m {
			t.Errorf("atlas/define.md marks %s %d time(s) but only %d carry the pinned value %d — "+
				"an unmarked or stale restatement is exactly what the markers exist to prevent",
				c, n, m, knownRawByCause[c])
		}
	}
}

// The docs quote the -locale help VERBATIM.
//
// Fourth member of a family this repo keeps re-fixing by hand: a doc restating
// something the code owns. #27 found the locale policy stated in four places —
// the flag, localeFor, the README and the atlas — with nothing keeping them in
// step, and the README's version was two milestones stale ("applies to English
// only", which stopped being true when #23 M1's interim rule was replaced).
//
// Narrow on purpose, like the prompt test above: it pins the one line a reader
// acts on, not the prose around it, which should stay free to be rewritten.
// The same mechanism for -pron (#29), and for the same reason the -locale one
// exists: this policy is stated in the flag, the README and the atlas, and
// nothing but a test keeps three copies in step.
//
// It is a SEPARATE test rather than a row in the one below because the failure
// messages differ — a reader who breaks this one needs to be told about -pron,
// not about locales.
func TestDocsQuoteThePronHelp(t *testing.T) {
	want := "<!-- pron-help -->" + pronHelp + "<!-- /pron-help -->"
	for _, doc := range derivedDocs {
		b, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("%s unreadable: %v", doc, err)
		}
		if !strings.Contains(string(b), want) {
			t.Errorf("%s does not quote the -pron help.\nwant the marked span to read:\n%s\n"+
				"pronHelp owns this text; the docs consume it.", doc, want)
		}
	}
}

func TestDocsQuoteTheLocaleHelp(t *testing.T) {
	// EVERY doc that states the policy, not just the first one wired up. Fixing
	// the README alone left atlas/define.md as the next copy to go stale, which
	// is the same half-fix this family keeps producing.
	want := "<!-- locale-help -->" + localeHelp + "<!-- /locale-help -->"
	for _, doc := range derivedDocs {
		b, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("%s unreadable: %v", doc, err)
		}
		if !strings.Contains(string(b), want) {
			t.Errorf("%s does not quote the -locale help.\nwant the marked span to read:\n%s\n"+
				"localeHelp owns this text; the docs consume it.", doc, want)
		}
	}
}

// The atlas's command list DERIVES from the registry, or it drifts.
//
// Third instance of the `doc-sweep-incomplete` family on this page, and the
// measured shape is what makes a mechanism the right answer rather than a row:
// the list held three of five commands, and the two missing were the two most
// recently added — `/lang` (#23) and `/pron` (#29). Two for two. Every author
// added a command, updated the registry, and did not know this table existed.
//
// So the table is generated here and the page consumes it, exactly as
// TestDocsQuoteTheLocaleHelp does for localeHelp. The next command fails the
// build until the page catches up, which is the only thing that has ever worked
// for this family.
//
// The NAME and SUMMARY only, in registry order. Argument syntax is deliberately
// out: the summary is what /help prints, so padding it with forms would make the
// table stop matching the screen — and the screen is what a reader checks it
// against.
func TestDocsQuoteTheCommandList(t *testing.T) {
	var b strings.Builder
	b.WriteString("<!-- command-list -->\n| command | does |\n|---|---|\n")
	for _, c := range commands {
		fmt.Fprintf(&b, "| `/%s` | %s |\n", c.name, c.summary)
	}
	b.WriteString("<!-- /command-list -->")

	doc := "../../atlas/define.md"
	raw, err := os.ReadFile(doc)
	if err != nil {
		t.Fatalf("%s unreadable: %v", doc, err)
	}
	if !strings.Contains(string(raw), b.String()) {
		t.Errorf("%s does not quote the command list the registry produces.\nwant the "+
			"marked span to read:\n%s\n`commands` owns this list; the page consumes it.",
			doc, b.String())
	}
}
