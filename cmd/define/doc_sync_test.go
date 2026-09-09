package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"

	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"sort"
	"strconv"
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
	// THE RESIDUAL IS NO LONGER HUMAN (#12). This comment used to say "a form
	// added to play and not added to this slice is not checked here — that half
	// is human", and #12 then added a form and did not add it here, so neither of
	// its prompt lines was a README consumer. TestEveryFormIsEnrolled below
	// derives the extent from the package's Form() declarations, the same move
	// numRegionKinds makes for region kinds: the set has one source, and a form
	// that is not enrolled fails the build.
	forms := docSyncForms(t)

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
	// THE GRADED LINE IS PER-FORM TOO, since #12: a form with the flag gesture
	// says so after the answer as well, because that is when a learner discovers
	// a question was broken. Checking only the const would have missed exactly
	// the line the close review found wrong.
	gradedSeen := map[string]bool{}
	for _, f := range forms {
		line := gradedPromptFor(f)
		if gradedSeen[line] {
			continue // two forms may legitimately share the post-answer line
		}
		gradedSeen[line] = true
		t.Run("graded: "+line, func(t *testing.T) {
			if !strings.Contains(readme, line) {
				t.Errorf("README.md does not contain the line livePrompt returns for %T once graded:"+
					"\n\t%q", f, line)
			}
		})
	}
}

// docSyncForms is the ONE list of forms the documentation guards walk.
//
// A helper rather than a local, so the keys-line guard and the enrolment guard
// cannot disagree about what the set is — which would put the extent back in two
// places, the thing TestEveryFormIsEnrolled exists to stop.
func docSyncForms(t *testing.T) []play.Question {
	t.Helper()
	return []play.Question{
		play.NewChoice("ephemeral", "", []play.Option{
			{Gloss: "a", Correct: true}, {Gloss: "b"}, {Gloss: "c"}, {Gloss: "d"},
		}),
		// The board's line differs in its RESERVED half too, not only in its own
		// keys: `d` is refused on a form holding many words, so the prompt must
		// not offer it (#40 D12). That is the second thing this row checks, and
		// the reason it is worth adding rather than being the "third form" the
		// comment above calls human.
		play.NewBoard([]play.Cell{{Word: "ephemeral"}, {Word: "quokka"}}, 80, play.Palette{}),
		// The cloze's line differs in its OWN half twice over: it grades digits
		// like 2.3 but says "pick the word", and it is the first form with a
		// gesture that is neither an answer nor reserved, so `?` has to be named
		// or a learner has no source for it (#12).
		play.NewCloze("ephemeral", "a ___ b", "a ephemeral b", "", []play.Option{
			{Word: "ephemeral", Correct: true}, {Word: "quokka"}, {Word: "keel"}, {Word: "mesa"},
		}),
	}
}

// TestEveryFormIsEnrolled makes the doc guard's extent MECHANICAL.
//
// Its sibling above used to record "a form added to play and not added to this
// slice is not checked here — that half is human", and #12 then added a form and
// did not add it. So the set is derived: every type in `play` that declares
// Form() is a form, and every form must appear in the slice the doc guard walks.
//
// Same move numRegionKinds makes for region kinds — the extent has one source,
// and a member added without being enrolled fails the build rather than being
// silently unchecked.
//
// PARSED, NOT SCRAPED (#12 BR-17). The first version matched a regex requiring a
// single-letter pointer receiver, a one-line body and a lowercase literal, all
// at once — and because the assertion is "declared ⊆ enrolled", a form the regex
// MISSED was silence rather than failure. Measured: renaming Cloze's receiver to
// `cz` (still gofmt-clean) and un-enrolling it left this guard, both README
// guards and BR-14's region guard all green. A derivation that can under-derive
// silently is not a derivation; it is the hand-maintained list with extra steps.
//
// go/parser answers the question the regex was approximating — "is there a
// method named Form on some receiver returning a string literal" — with no
// opinion about formatting. And the count assertion below makes it FAIL CLOSED:
// if the parse ever finds fewer forms than are enrolled, the extent is wrong in
// the direction that hides things, and that is now the loud case rather than the
// quiet one.
func TestEveryFormIsEnrolled(t *testing.T) {
	declared := declaredForms(t)
	enrolled := map[string]bool{}
	for _, f := range docSyncForms(t) {
		enrolled[f.Form()] = true
	}
	for name := range declared {
		if !enrolled[name] {
			t.Errorf("form %q is declared in play/ but not enrolled in the doc guard's slice, "+
				"so neither of its prompt lines is checked against README.md", name)
		}
	}
	// FAIL CLOSED. Subset-only is satisfied by finding nothing, which is exactly
	// how a mis-derivation would present.
	if len(declared) != len(enrolled) {
		t.Errorf("the parse found %d form(s) %v but %d are enrolled %v.\n"+
			"Equal counts are the point: a derivation that finds FEWER than are "+
			"enrolled is under-deriving, and every guard built on it is then "+
			"checking a set nobody chose.",
			len(declared), keysOf(declared), len(enrolled), keysOf(enrolled))
	}
}

// declaredForms is every form name `play` declares, read from the AST.
//
// A form is a method named Form with a receiver, no parameters, one string
// result, whose body is a single `return "<literal>"`. That last clause is the
// only shape assumption left, and it is a real one: a Form() computing its name
// would make "the set of forms" unknowable statically, which is worth failing on
// rather than guessing at.
func declaredForms(t *testing.T) map[string]bool {
	t.Helper()
	declared := map[string]bool{}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "play", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing play/: %v", err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, d := range file.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name.Name != "Form" || fn.Body == nil {
					continue
				}
				if len(fn.Body.List) != 1 {
					t.Errorf("%s: Form() has a %d-statement body; this guard can only "+
						"derive a name from a single return of a literal",
						fset.Position(fn.Pos()), len(fn.Body.List))
					continue
				}
				ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
				if !ok || len(ret.Results) != 1 {
					continue
				}
				lit, ok := ret.Results[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Errorf("%s: Form() does not return a string literal, so the set of "+
						"forms cannot be known statically", fset.Position(fn.Pos()))
					continue
				}
				name, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: %v", fset.Position(fn.Pos()), err)
				}
				declared[name] = true
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("no Form() declarations found; this guard would pass vacuously")
	}
	return declared
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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

// THE KEY TABLE DERIVES FROM THE FORMS THAT OWN THE KEYS (#12 BR-16).
//
// README.md's table is what a reader consults for what they can press, and its
// header promises completeness. It was the THIRD hand-maintained home of the
// same fact: BR-1 fixed the prompt lines, BR-10 fixed the enrolment that checks
// them, and this table still listed no `?` and still described the digits as
// "pick the definition" after a second form began grading them.
//
// The rule the three findings share: **every enumeration of live keys must
// derive from the code that owns them, and a new key is not shipped until every
// such enumeration derives.** So this reads each form's own Keys() line, which
// is already the single source the prompt lines come from, splits it into the
// `key = what` pairs the form commits to, and requires the table to name each
// one. A form that grows a gesture reddens this until the table says so.
//
// The WHAT is matched rather than the key glyph: `1`–`4` in the table is a range
// where a form writes "1-4", and a guard matching glyphs would be pinned to the
// table's typography instead of to its content.
func TestREADMEKeyTableNamesEveryLiveKey(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("README.md unreadable: %v", err)
	}
	table := keyTableIn(t, string(b))
	for _, q := range docSyncForms(t) {
		for _, pair := range strings.Split(q.Keys(), ", ") {
			_, what, ok := strings.Cut(pair, " = ")
			if !ok {
				continue
			}
			if !strings.Contains(strings.ToLower(table), strings.ToLower(what)) {
				t.Errorf("README.md's key table does not name %q, which %T offers.\n"+
					"Full keys line: %q\n"+
					"The table is where a reader looks for what they can press; a key that "+
					"works and is listed nowhere is a key nobody presses.", what, q, q.Keys())
			}
		}
	}
}

// keyTableIn is the key table alone, so the guard above cannot be satisfied by
// the same words appearing in prose elsewhere in the README — the mistake
// TestAtlasDescribesEveryRegionKind's comment records ("that word occurs in the
// atlas nineteen times for unrelated reasons, so a docs guard built on it passed
// with the whole section deleted").
func keyTableIn(t *testing.T, readme string) string {
	t.Helper()
	const header = "| key | does |"
	i := strings.Index(readme, header)
	if i < 0 {
		t.Fatalf("README.md has no %q table; this guard would certify nothing", header)
	}
	rest := readme[i:]
	// The table ends at the first blank line — markdown's own rule.
	if j := strings.Index(rest, "\n\n"); j >= 0 {
		rest = rest[:j]
	}
	if strings.Count(rest, "\n") < 3 {
		t.Fatalf("the key table has %d rows; this guard would certify nothing", strings.Count(rest, "\n"))
	}
	return rest
}

// THE STORE LAYOUT DERIVES FROM THE EVENT KINDS (#12 BR-18).
//
// Both blocks — README.md's and the atlas's — read "kinds: looked-up, asked"
// long after `reviewed` and `flagged` existed. The README's is the only
// documentation a human reading their own event log has, so a kind missing from
// it is a row in a file nobody can interpret.
//
// This is the fourth finding in the same family: a document restating a fact the
// code owns. The rule the family produced — **a restatement must derive from the
// code or be swept at the boundary that changed it** — has a derivable half and a
// human half, and this closes the derivable half for event kinds the way
// TestREADMEKeyTableNamesEveryLiveKey closed it for keys. What is left is
// workshop/targets/derived-restatement.md, which enumerates the rest.
func TestStoreLayoutDocsNameEveryEventKind(t *testing.T) {
	kinds := store.EventKinds()
	if len(kinds) == 0 {
		t.Fatal("no event kinds; this guard would certify nothing")
	}
	for _, doc := range []struct{ path, marker string }{
		{"README.md", "events/2026-08-21.yaml"},
		{"../../atlas/define.md", "events/YYYY-MM-DD.yaml"},
	} {
		b, err := os.ReadFile(doc.path)
		if err != nil {
			t.Fatalf("%s unreadable: %v", doc.path, err)
		}
		block := layoutBlockIn(t, doc.path, string(b), doc.marker)
		for _, k := range kinds {
			if !strings.Contains(block, string(k)) {
				t.Errorf("%s's store layout does not name the event kind %q.\n"+
					"It is the block a reader consults to interpret their own log; "+
					"a kind written there by the code and missing here is a row "+
					"nobody can read.", doc.path, k)
			}
		}
	}
}

// layoutBlockIn is the fenced store-layout block alone, so prose elsewhere
// naming a kind in passing cannot satisfy the guard above — the scoping mistake
// TestAtlasDescribesEveryRegionKind's comment records.
func layoutBlockIn(t *testing.T, path, doc, marker string) string {
	t.Helper()
	i := strings.Index(doc, marker)
	if i < 0 {
		t.Fatalf("%s has no %q line; the store layout moved and this guard "+
			"would certify nothing", path, marker)
	}
	start := strings.LastIndex(doc[:i], "```")
	end := strings.Index(doc[i:], "```")
	if start < 0 || end < 0 {
		t.Fatalf("%s: the store layout at %q is not in a fenced block", path, marker)
	}
	return doc[start : i+end]
}

// THE STORE LAYOUTS NAME EVERY RUNTIME DIRECTORY (#46 BR-30).
//
// Both blocks — README.md's and the atlas's — restated `store.RuntimeDirs` by
// hand, and the README had already fallen behind: `usage/` was missing, and
// `audio/` only arrived because a reviewer noticed.
//
// Fourth finding in the family `workshop/targets/derived-restatement.md` exists
// for, and the rule that target states is the fix: DERIVE where the fact is
// machine-readable. `RuntimeDirs` is an exported slice; a directory added to it
// now fails this until both blocks describe it, exactly as
// `TestGitignoreCoversRuntimeDirs` already fails until `.gitignore` does. The
// derived consumer was right through every one of those findings while every
// hand-maintained one was wrong.
func TestStoreLayoutDocsNameEveryRuntimeDir(t *testing.T) {
	if len(store.RuntimeDirs) == 0 {
		t.Fatal("no runtime directories; this guard would certify nothing")
	}
	for _, doc := range []struct{ path, marker string }{
		{"README.md", "events/2026-08-21.yaml"},
		{"../../atlas/define.md", "events/YYYY-MM-DD.yaml"},
	} {
		b, err := os.ReadFile(doc.path)
		if err != nil {
			t.Fatalf("%s unreadable: %v", doc.path, err)
		}
		block := layoutBlockIn(t, doc.path, string(b), doc.marker)
		for _, dir := range store.RuntimeDirs {
			if !strings.Contains(block, dir+"/") {
				t.Errorf("%s's store layout does not describe %q.\n"+
					"It is the block a reader consults to interpret their own "+
					"directory; a directory the program writes and this does not "+
					"name is a file nobody can account for.", doc.path, dir)
			}
		}
	}
}

// TestAtlasListsEveryConformanceCheck derives the atlas's conformance table from
// the files on disk.
//
// THE TABLE LAGGED BY TWO OF NINE and nobody noticed across several issues
// (#49 III): `harvest_conformance_test.go` and `version_conformance_test.go` both
// existed with no row. That is the ordinary fate of a hand-typed enumeration of a
// set the code already knows — the same defect `declaredModes` exists to end one
// directory over, applied to docs.
//
// Only presence is derived, never the prose: what a check ASSERTS is a human
// sentence and belongs to whoever wrote the check. This fails when a file has no
// row at all, which is the half that rots silently.
func TestAtlasListsEveryConformanceCheck(t *testing.T) {
	files, err := filepath.Glob("*_conformance_test.go")
	if err != nil {
		t.Fatalf("globbing conformance files: %v", err)
	}
	// live_property_test.go is conformance by tag rather than by name, and the
	// table has always carried it; deriving it by name alone would drop the row.
	if _, err := os.Stat("live_property_test.go"); err == nil {
		files = append(files, "live_property_test.go")
	}
	if len(files) < 7 {
		t.Fatalf("found %d conformance files %v; this package has at least seven, so "+
			"this derivation is under-deriving and would certify a table nobody wrote",
			len(files), files)
	}

	atlas, err := os.ReadFile(filepath.Join("..", "..", "atlas", "define.md"))
	if err != nil {
		t.Fatalf("reading atlas/define.md: %v", err)
	}
	for _, f := range files {
		name := filepath.Base(f)
		if !strings.Contains(string(atlas), "`"+name+"`") {
			t.Errorf("atlas/define.md has no row for %s. Every conformance check earns "+
				"one, because the table is how a reader learns which assumptions are "+
				"pinned against the live world — and a check with no row is one nobody "+
				"knows to run. Add it beside its siblings in the conformance table.", name)
		}
	}
}
