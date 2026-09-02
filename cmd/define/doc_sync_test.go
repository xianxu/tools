package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
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
	b, err := os.ReadFile("README.md")
	if err != nil {
		// NOT a skip: the README is in the repo, so an unreadable one is a
		// broken checkout or a moved file, never an absent dependency.
		t.Fatalf("README.md unreadable: %v", err)
	}
	readme := string(b)

	// EVERY SHIPPED FORM's prompt, composed exactly as the loop composes it.
	//
	// This used to read two consts. That was wrong in a way the README could not
	// show: the grading prompt is now per-FORM, because a const naming form
	// 2.1's y/n was printed under form 2.3's numbered options and told the
	// learner to press a key that did nothing.
	//
	// The residual, stated rather than hidden: a THIRD form added to play and
	// not added to this slice is not checked here. That half is human. What is
	// mechanical is that Question.Keys() is on the interface, so a new form
	// cannot compile without writing one, and the rows below fail the build the
	// moment an existing form's wording moves.
	forms := []play.Question{
		play.NewRecall("ephemeral", "lasting for a very short time"),
		play.NewChoice("ephemeral", "", []play.Option{
			{Gloss: "a", Correct: true}, {Gloss: "b"}, {Gloss: "c"}, {Gloss: "d"},
		}),
		// The board's line differs in its RESERVED half too, not only in its own
		// keys: `d` is refused on a form holding many words, so the prompt must
		// not offer it (#40 D12). That is the second thing this row checks, and
		// the reason it is worth adding rather than being the "third form" the
		// comment above calls human.
		play.NewBoard([]play.Cell{{Word: "ephemeral"}, {Word: "quokka"}}, 80, play.Palette{}),
	}
	seen := map[string]bool{}
	for _, f := range forms {
		line := gradePrompt(f)
		t.Run(line, func(t *testing.T) {
			if seen[line] {
				t.Errorf("two forms print the identical prompt %q — one of them is not describing its own keys", line)
			}
			seen[line] = true
			if !strings.Contains(readme, line) {
				t.Errorf("README.md does not contain the line livePrompt returns for %T:\n\t%q\n"+
					"The loop changed and the README did not. Update README.md, or "+
					"change the form's Keys() if the new wording is the intended one.", f, line)
			}
		})
	}
	t.Run("the graded prompt, shown once the answer is up", func(t *testing.T) {
		if !strings.Contains(readme, gradedPrompt) {
			t.Errorf("README.md does not contain:\n\t%q", gradedPrompt)
		}
	})
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

// The atlas quotes the /pron COMMAND's argument rule from the code that owns it.
//
// Third prose site for this one rule — #31's per-command paragraph, the walk
// sentence in the pronunciation section, and the README — and the first two went
// stale the moment #35 made the argument optional. This pins the one that states
// the RULE; the other two are prose about behaviour, swept by hand and named as
// such in #35's plan rather than pretending a mechanism covers them.
func TestDocsQuoteThePronCommandHelp(t *testing.T) {
	want := "<!-- pron-command-help -->" + pronCommandHelp + "<!-- /pron-command-help -->"
	doc := "../../atlas/define.md"
	b, err := os.ReadFile(doc)
	if err != nil {
		t.Fatalf("%s unreadable: %v", doc, err)
	}
	if !strings.Contains(string(b), want) {
		t.Errorf("%s does not quote the /pron argument rule.\nwant the marked span to read:\n%s\n"+
			"pronCommandHelp owns this text; the page consumes it.", doc, want)
	}
}

// The atlas describes EVERY region kind, derived from the registry (#30 BR-39).
//
// The docs-lag family reached two findings before this existed, and the cause
// was structural rather than forgetfulness: `M1.6` made the doc sweep a TASK in
// one milestone, so the next milestone shipped surface with no row to remind
// anyone. A task cannot cover work that has not been planned yet; a guard can.
//
// Same shape as `TestEveryEnabledMouseModeIsDecoded`: the SET has one owner
// (`numRegionKinds`), and the check derives from it rather than restating it. A
// third kind added to the registry reddens this until the atlas says what it
// offers — which is the only version of "keep the docs current" that survives
// the next person to add one.
func TestAtlasDescribesEveryRegionKind(t *testing.T) {
	b, err := os.ReadFile("../../atlas/define.md")
	if err != nil {
		// NOT a skip: the atlas is in the repo, so an unreadable one is a broken
		// checkout or a moved file, never an absent dependency.
		t.Fatalf("atlas/define.md unreadable: %v", err)
	}
	atlas := string(b)
	// The qualified GO IDENTIFIER, not the prose name. Searching for k.String()
	// was the same defect this file's RenderOpts guard had: "headword" occurs in
	// the atlas nineteen times for unrelated reasons, so deleting the whole
	// "## Clickable regions" section left this green. One finding named two
	// sites and only one was swept — the instance rather than the class.
	for k := RegionKind(0); k < numRegionKinds; k++ {
		if !strings.Contains(atlas, k.identifier()) {
			t.Errorf("the atlas does not mention %s, the %q region. A clickable span the "+
				"docs never describe is surface a reader can only find by clicking at "+
				"random.", k.identifier(), k)
		}
	}
}

// Every RenderOpts field is described in the atlas, derived from the struct
// (#30 M2, BR-52).
//
// Fourth finding in the docs-lag family, and the deliverable is the rule rather
// than the two lines it named: `RenderOpts` gained `Word` — the field that
// carries a click's target, and the fix for a Critical — while the atlas went on
// describing the old shape. A doc sweep as a TASK cannot cover a field added by
// a later fix; a guard derived from the declared set can.
//
// Same move as `TestAtlasDescribesEveryRegionKind` and
// `TestEveryEnabledMouseModeIsDecoded`: the SET has one owner, and the check
// reads it rather than restating it. Adding a field reddens this until the atlas
// says what it is for.
//
// Deliberately narrow: it asks only that the NAME appears. Prose cannot be
// checked mechanically, and a guard that pretended to would be theatre — what it
// prevents is a field nobody wrote a sentence about at all.
func TestAtlasDescribesEveryRenderOpt(t *testing.T) {
	b, err := os.ReadFile("../../atlas/define.md")
	if err != nil {
		t.Fatalf("atlas/define.md unreadable: %v", err)
	}
	atlas := string(b)
	rt := reflect.TypeOf(RenderOpts{})
	if rt.NumField() == 0 {
		t.Fatal("RenderOpts has no fields; this guard would certify nothing")
	}
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if !rt.Field(i).IsExported() {
			continue
		}
		// The QUALIFIED name only. The first version also accepted a bare
		// `Field`, which made it vacuous for the very field it was written for:
		// "Word" occurs in the atlas for a dozen unrelated reasons — the word to
		// play, Region.Word — so the guard passed no matter what RenderOpts said.
		// A guard that any prose can satisfy is not checking the tree, which is
		// the rule this file already states one test above.
		if !strings.Contains(atlas, "RenderOpts."+name) {
			t.Errorf("the atlas never mentions RenderOpts.%s. A rendering input nobody "+
				"documented is one the next reader has to infer from the code — and this "+
				"one carried a Critical's fix.", name)
		}
	}
}

// The README must name EVERY reason a word falls back to form 2.1.
//
// Derived from the code's own list rather than checked against a copy in the
// test, because a hand-maintained enumeration is what failed: choiceFor branched
// on three reasons while the README named two and the atlas named one, and the
// third — a dictionary redirect sending `bargainer` to `bargain` — is
// user-visible, since that word silently gets the other form.
func TestREADMENamesEveryFallbackReason(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md unreadable: %v", err)
	}
	readme := strings.ToLower(string(b))
	if len(fallbackReasons) == 0 {
		t.Fatal("no fallback reasons declared; this guard would certify nothing")
	}
	for _, r := range fallbackReasons {
		if !strings.Contains(readme, strings.ToLower(r)) {
			t.Errorf("README.md does not name the fallback reason %q.\n"+
				"A learner whose word silently gets the other form has no way to know why. "+
				"Add it, or change fallbackReasons if the wording moved.", r)
		}
	}
}

// THE README'S BOARD IS DERIVED, NOT DRAWN BY HAND (#40 BR-18).
//
// The block was a hand-maintained picture with no consumer, and it went stale
// the moment an operator sitting changed the design: it still showed the mark
// standing where the key was, and a label row carrying the old hole at `d` —
// both contradicted by the README's own prose eight lines below.
// `doc_sync_test` pinned the prompt LINE and nothing pinned the grid.
//
// So the grid derives, exactly as the prompt lines do: the fenced block's cell
// rows must be what a real `play.Board` over those words draws. The marks are
// deliberately absent from the picture — a marked cell is distinguished by
// COLOUR, which a code fence cannot show, and the prose says so instead.
func TestREADMEDrawsTheBoardTheFormActuallyDraws(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md unreadable: %v", err)
	}
	readme := string(b)
	board := play.NewBoard(boardCells(readmeBoardWords...), defaultCols, play.Palette{})

	grid := strings.Split(board.Prompt(), "\n")
	rows := 0
	for _, row := range grid {
		if row == "" {
			break // the blank that separates the grid from the panel
		}
		rows++
		if !strings.Contains(readme, row) {
			t.Errorf("README.md does not contain the grid row the board draws:\n\t%q\n"+
				"The form changed and the README did not. Regenerate the block from "+
				"play.Board over readmeBoardWords, or change the form if the picture "+
				"is the intended one.", row)
		}
	}
	if rows == 0 {
		t.Fatal("the board drew no grid rows; this test would assert nothing")
	}
	// AND THE PICTURE IS OF AN UNMARKED BOARD, so nothing in it can imply a mark
	// is drawn by a glyph. `[y]`/`[n]` in a cell is the design the sitting
	// deleted.
	for _, gone := range []string{"[y] ", "[n] "} {
		if strings.Contains(readme, gone) {
			t.Errorf("README.md draws %q in a cell — a mark is COLOUR now, and the key stays put", gone)
		}
	}
}
