package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strings"
	"testing"
)

// withDeck is deps carrying a vocabulary, which is what writeWords derives from.
// One helper so no row builds the struct by hand and gets it subtly wrong.
func withDeck(words ...string) deps {
	return deps{langDeps: langDeps{vocab: deckOf(words...)}}
}

func deckOf(words ...string) Vocabulary {
	v := &memVocabulary{}
	for _, w := range words {
		v.Add(w)
	}
	return v
}

// THE ROW THAT DECIDES THE DESIGN (#46).
//
// The reveal contains an entry Render has already highlighted, so the text
// carries "\x1b[1;32m" runs. A walk over raw bytes tokenises the letters inside
// an escape as words and reports columns that count escape bytes as cells —
// producing regions that underline a `[1;32m` and clicks that land on nothing.
// Everything downstream still renders, which is why this is a test and not a
// thing anyone would notice.
func TestASpanWalkSkipsEscapeSequences(t *testing.T) {
	v := deckOf("keel", "mesa")
	plain := "the keel and the mesa"
	coloured := "the " + knownOn + "keel" + sgrOff + " and the " + knownOn + "mesa" + sgrOff

	want := deckSpans(plain, v)
	got := deckSpans(coloured, v)
	if len(want) != 2 {
		t.Fatalf("the plain walk found %d spans, want 2: %+v", len(want), want)
	}
	if len(got) != len(want) {
		t.Fatalf("coloured text yielded %d spans, plain yielded %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("span %d differs once the text is coloured:\n coloured %+v\n plain    %+v\n"+
				"Columns must be DISPLAY CELLS, so an escape sequence occupies none.",
				i, got[i], want[i])
		}
	}
	// And nothing was matched INSIDE an escape: every span's text is a real word.
	for _, s := range got {
		if strings.ContainsAny(s.Text, "\x1b[") {
			t.Errorf("span %+v matched inside an escape sequence", s)
		}
	}
}

// Longest phrase wins, INHERITED from highlightSpans rather than re-decided.
func TestAPhraseBeatsItsFirstWord(t *testing.T) {
	got := deckSpans("one hot dog please", deckOf("hot", "hot dog"))
	if len(got) != 1 || got[0].Text != "hot dog" {
		t.Errorf("got %+v, want the phrase — the matcher's rule must carry through the locator", got)
	}
}

// Coordinates are per LINE, on the line the word is on.
func TestSpansAreLocatedOnTheirOwnLine(t *testing.T) {
	got := deckSpans("alpha\nthe keel\n", deckOf("keel"))
	if len(got) != 1 {
		t.Fatalf("got %d spans, want 1: %+v", len(got), got)
	}
	if got[0].Line != 1 || got[0].Col != 4 || got[0].Width != 4 {
		t.Errorf("got line %d col %d width %d, want 1/4/4", got[0].Line, got[0].Col, got[0].Width)
	}
}

// Only DECK words. The walk marks what the learner is learning, not every word.
func TestOnlyDeckWordsAreSpanned(t *testing.T) {
	if got := deckSpans("the keel and the boat", deckOf("keel")); len(got) != 1 {
		t.Errorf("got %d spans, want 1 — a non-deck word was marked: %+v", len(got), got)
	}
	if got := deckSpans("nothing here", nil); got != nil {
		t.Errorf("a nil vocabulary produced %+v; nil is the one representation of "+
			"'nothing to highlight'", got)
	}
}

// EVERY REGION COVERS THE TEXT IT CLAIMS (#12 BR-14), now on coloured text.
func TestWordRegionsCoverTheTextTheyClaim(t *testing.T) {
	v := deckOf("keel", "mesa")
	text := "the " + knownOn + "keel" + sgrOff + " and the mesa\nagain the keel"
	lines := strings.Split(text, "\n")
	rs := wordRegions(text, v)
	if len(rs) != 3 {
		t.Fatalf("got %d regions, want 3: %+v", len(rs), rs)
	}
	for _, r := range rs {
		if got := cellSlice(stripEscapes(lines[r.Line]), r.Col, r.Width); got != r.Text {
			t.Errorf("region %+v claims %q but %q is written there", r, r.Text, got)
		}
	}
}

// A LINE'S REGIONS MUST BE DISJOINT (#46 PQ-2).
//
// markClickable sorts by column and walks ONE cursor left to right, so a second
// region at a column already passed can never match — and every LATER region on
// that line silently loses its underline. RegionAt compounds it by resolving in
// append order rather than by narrowest span. Neither was wrong while one
// producer owned a line.
func TestALinesRegionsAreDisjointAndAscending(t *testing.T) {
	// The concrete case: a Choice prompt's headword IS a deck word, so both
	// producers emit at line 1 column 0.
	head := []Region{{Kind: RegionHeadword, Text: "keel", Word: "keel", Line: 1, Col: 0, Width: 4}}
	got := mergeRegions(head, wordRegions("\nkeel\n\nthe mesa", deckOf("keel", "mesa")))

	byLine := map[int][]Region{}
	for _, r := range got {
		byLine[r.Line] = append(byLine[r.Line], r)
	}
	for ln, rs := range byLine {
		for i := range rs {
			for j := i + 1; j < len(rs); j++ {
				if rs[i].Col < rs[j].Col+rs[j].Width && rs[j].Col < rs[i].Col+rs[i].Width {
					t.Errorf("line %d has overlapping regions %+v and %+v — "+
						"markClickable would drop every later region on this line",
						ln, rs[i], rs[j])
				}
			}
		}
	}
	// PRECEDENCE: the headword survives, because it carries the LOOKUP KEY, which
	// is more specific than a text match (`define jalapeno` renders `jalapeño`).
	var kept Region
	for _, r := range got {
		if r.Line == 1 && r.Col == 0 {
			kept = r
		}
	}
	if kept.Kind != RegionHeadword {
		t.Errorf("the text match displaced the headword region: %+v", kept)
	}
	// And the OTHER deck word still got one.
	if len(got) != 2 {
		t.Errorf("got %d regions, want 2 (the headword and the mesa): %+v", len(got), got)
	}
}

// EVERY FORM IS CLASSIFIED, over the extent TestEveryFormIsEnrolled makes
// mechanical — so a form added to play cannot arrive without an answer to
// "is this text the deck itself".
func TestEveryFormHasASurface(t *testing.T) {
	declared := declaredForms(t)
	// The classification is total by construction (surfaceOf has a default), so
	// what this asserts is that someone DECIDED: every form name must appear in
	// surfaceOf's source, or it is taking the default by accident.
	src := readSource(t, "deckwords.go")
	for name := range declared {
		if !strings.Contains(src, `"`+name+`"`) && !strings.Contains(src, "// unclassified: "+name) {
			t.Errorf("form %q is declared in play/ but surfaceOf names no case for it, "+
				"so it takes the prose default by accident rather than by decision.\n"+
				"Add a case, or a `// unclassified: %s` comment saying prose is right for it.",
				name, name)
		}
	}
}

// A cloze's options are CLICKABLE BUT NOT COLOURED — the operator's rule, and
// the case it was asked for.
//
// BOTH HALVES. The first version asserted only the colour half, so a change that
// removed every click target from a cloze would have left it green — and
// "clickable" is the half the operator asked for first.
func TestClozeOptionsAreClickableButNotColoured(t *testing.T) {
	// The click half, through the write door a sitting actually uses.
	rw := &recordingRegionWriter{}
	prompt := "\nThe Times dismissed it as ___.\n\n1  keel\n2  mesa\n"
	writeWords(rw, prompt, nil, deps{langDeps: langDeps{vocab: deckOf("keel", "mesa")}}, options{color: true}, surfaceOf("cloze"), "")
	if len(rw.regions) != 2 {
		t.Errorf("a cloze prompt offered %d click targets, want 2: %+v", len(rw.regions), rw.regions)
	}
	for _, r := range rw.regions {
		if r.Kind != RegionWord {
			t.Errorf("region %+v is not a deck word", r)
		}
	}
	if strings.Contains(rw.String(), knownOn) {
		t.Errorf("the cloze prompt was coloured: %q", rw.String())
	}

	// And the rule itself.
	if surfaceOf("cloze").admitsColour() {
		t.Error("a cloze admits colour; all four options are deck words, so every one would go green")
	}
	if surfaceOf("board").admitsColour() {
		t.Error("a board admits colour; every cell is a deck word by construction")
	}
	if !surfaceOf("meaning").admitsColour() {
		t.Error("a meaning question refuses colour; its options are DEFINITIONS, " +
			"prose in which a known word is worth spotting")
	}
}

// readSource is a source file's text, for the guards that check a DECISION was
// made rather than a behaviour produced.
func readSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("%s unreadable: %v", name, err)
	}
	return string(b)
}

// A CLICK DOES NOT DEPEND ON COLOUR (#46).
//
// vocabularyFor returns nil when colour is off — correctly, for its own caller:
// reading the deck to inject a style nobody will see is IO for a disabled
// feature. A second consumer arrived that has nothing to do with style, and
// taking the vocabulary from there would have made `--no-color` silently remove
// every click target as well.
func TestClicksSurviveNoColour(t *testing.T) {
	v := deckOf("keel")
	_ = v
	var withColour, without strings.Builder
	writeWords(&withColour, "the keel", nil, deps{langDeps: langDeps{vocab: v}}, options{color: true}, surfaceProse, "")
	writeWords(&without, "the keel", nil, deps{langDeps: langDeps{vocab: v}}, options{color: false}, surfaceProse, "")

	if !strings.Contains(withColour.String(), knownOn) {
		t.Error("colour on produced no highlight")
	}
	if strings.Contains(without.String(), knownOn) {
		t.Errorf("colour off still emitted ANSI: %q", without.String())
	}
	// The regions are what a click resolves against, and they are identical.
	if got := len(wordRegions("the keel", v)); got != 1 {
		t.Errorf("got %d regions, want 1 — a click target must not depend on colour", got)
	}
}

// A SURFACE THAT REFUSES COLOUR STILL OFFERS CLICKS. This is the operator's
// case, stated at the write door rather than at the rule.
func TestADeckSurfaceIsClickableButUncoloured(t *testing.T) {
	v := deckOf("keel", "mesa")
	var out strings.Builder
	rw := &recordingRegionWriter{}
	writeWords(rw, "keel\nmesa", nil, deps{langDeps: langDeps{vocab: v}}, options{color: true}, surfaceDeck, "")
	writeWords(&out, "keel\nmesa", nil, deps{langDeps: langDeps{vocab: v}}, options{color: true}, surfaceDeck, "")

	if strings.Contains(out.String(), knownOn) {
		t.Errorf("a deck surface was coloured: %q — every word there is a deck word, "+
			"so colour marks everything and distinguishes nothing", out.String())
	}
	if len(rw.regions) != 2 {
		t.Errorf("a deck surface offered %d click targets, want 2: %+v", len(rw.regions), rw.regions)
	}
}

// recordingRegionWriter captures what writeRendered hands the screen.
type recordingRegionWriter struct {
	strings.Builder
	regions []Region
}

func (w *recordingRegionWriter) WriteRegions(text string, rs []Region) {
	w.WriteString(text)
	w.regions = append(w.regions, rs...)
}

// EVERY DECK WORD IS CLICKABLE WHEREVER IT IS WRITTEN, on every surface the
// write door serves. Driven through writeWords rather than through the rule, so
// it fails if a call site stops passing the vocabulary.
func TestEveryDeckWordInASittingIsClickable(t *testing.T) {
	v := deckOf("keel", "mesa", "sycophantic")
	for _, tc := range []struct {
		name string
		text string
		sf   surface
		want int
	}{
		{"a cloze prompt", "\nthe ___ shifts\n\n1  keel\n2  mesa\n", surfaceDeck, 2},
		{"a meaning prompt", "\nsycophantic\n\n1  a keel is a thing\n", surfaceProse, 2},
		// EXACT TOKENS. `keels` is not `keel` to the matcher — isWordRune
		// tokenises whole words and does no inflection — so a fixture using
		// the inflected form would assert the wrong count for the right
		// reason.
		{"a reveal", "\nThe keel shifts sharply.\n\nyou chose\n1  mesa\n", surfaceProse, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rw := &recordingRegionWriter{}
			writeWords(rw, tc.text, nil, deps{langDeps: langDeps{vocab: v}}, options{color: true}, tc.sf, "")
			if len(rw.regions) != tc.want {
				t.Errorf("got %d click targets, want %d: %+v", len(rw.regions), tc.want, rw.regions)
			}
		})
	}
}

// A MEANING QUESTION'S GLOSSES ARE COLOURED, which is the contrast that makes
// the cloze rule a rule rather than a special case: 2.3's options are
// DEFINITIONS, prose in which a word you know is a discovery.
func TestChoiceOptionGlossesAreColoured(t *testing.T) {
	var out strings.Builder
	writeWords(&out, "\nsycophantic\n\n1  a part of a keel, in a boat\n",
		nil, deps{langDeps: langDeps{vocab: deckOf("keel")}}, options{color: true}, surfaceOf("meaning"), "")
	if !strings.Contains(out.String(), knownOn+"keel") {
		t.Errorf("a gloss was not coloured: %q", out.String())
	}
}

// A BOARD'S CELLS CARRY NO DECK COLOUR even though every one is a deck word.
//
// NOT A VACUOUS PIN. A board renders through boardFooter into the FOOTER and
// never reaches the write door, so "the board is not coloured" is true today for
// a reason that has nothing to do with the rule. This asserts the reachable
// thing instead — that routing a board's cells through the door still produces
// no colour — which is what reddens the day someone wires the footer through it.
func TestBoardCellsCarryNoDeckColourEvenWhenTheyAreDeckWords(t *testing.T) {
	var out strings.Builder
	writeWords(&out, "0  keel   1  mesa\n", nil, deps{langDeps: langDeps{vocab: deckOf("keel", "mesa")}}, options{color: true}, surfaceOf("board"), "")
	if strings.Contains(out.String(), knownOn) {
		t.Errorf("a board's cells were coloured: %q — every cell is a deck word, "+
			"so colour marks everything and distinguishes nothing", out.String())
	}
}

// THE EMBEDDED RENDER IS NOT RE-COLOURED.
//
// Render colours the entry with a per-region BASE style (amber part-of-speech
// labels, the example style) and ANSI DOES NOT NEST — a second flat pass over
// the same bytes produces escapes inside escapes, which still renders and reads
// wrong. So the write door colours only OUTSIDE the range Render produced, and
// finds that range rather than assuming where it starts.
func TestTheRenderedEntryIsNotRecoloured(t *testing.T) {
	v := deckOf("keel", "mesa")
	// What Render would have handed back: already highlighted.
	already := "keel\n  noun\n    a mesa is not a " + knownOn + "keel" + sgrOff
	text := "\nThe keel shifts sharply.\n\n" + already + "\n"

	var out strings.Builder
	writeWords(&out, text, nil, deps{langDeps: langDeps{vocab: v}}, options{color: true}, surfaceProse, already)
	got := out.String()

	// The render arrives byte-for-byte as Render produced it.
	if !strings.Contains(got, already) {
		t.Errorf("the rendered entry was rewritten:\n got %q\n want it to contain %q", got, already)
	}
	// Nothing nested: no highlight opens immediately inside another.
	if strings.Contains(got, knownOn+knownOn) || strings.Contains(got, knownOn+"keel"+knownOn) {
		t.Errorf("colour was nested: %q", got)
	}
	// And the text OUTSIDE the render still got its colour — otherwise this
	// would pass by colouring nothing at all.
	before, _, _ := strings.Cut(got, already)
	if !strings.Contains(before, knownOn) {
		t.Errorf("nothing outside the render was coloured, so this proves nothing: %q", before)
	}
}

// EVERY writeWords CALL SITE PASSES A REAL VOCABULARY (#46 BR-21).
//
// The review found that nilling the vocabulary at all three sites left the WHOLE
// SUITE green: every M2 test built its own call, so they proved the door works
// and nothing about whether production opens it. That is the same finding M1's
// BR-3 made about the disk cache's wiring, and I fixed the instance there
// without applying the class here.
//
// PARSED, not listed. The extent is every writeWords call in non-test code, so a
// fourth site added later is checked the moment it exists — the correction #12's
// BR-17 made when a regex-scraped extent silently under-derived. It reads the
// ARGUMENT rather than driving the loops, because `isTerminal(stdout)` is a real
// syscall with no seam and a sitting cannot be driven in-process.
func TestEveryWriteWordsCallSitePassesAVocabulary(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	sites := 0
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				id, ok := call.Fun.(*ast.Ident)
				if !ok || id.Name != "writeWords" || len(call.Args) < 4 {
					return true
				}
				sites++
				// Argument 3 is the DEPS the loop is holding, and it must be a
				// plain identifier — the `d` that reached the function. Anything
				// else is a site building its own, which is how a site ends up
				// marking nothing.
				//
				// Why this shape: the first version took a `Vocabulary` argument
				// and this guard checked its SOURCE TOKEN, so `nil` reddened it
				// while `vocabularyFor(d, opt)` did not — even though that returns
				// nil whenever colour is off and would silently remove every click
				// target on `--no-color`. Removing the argument fixed that, and
				// then `deps{}` slipped through in its place. A composite literal
				// here is the same mistake wearing the new type.
				switch arg := call.Args[3].(type) {
				case *ast.Ident:
					if arg.Name == "nil" {
						t.Errorf("%s: writeWords is called with nil deps, so no deck word "+
							"at that site is clickable or coloured.", fset.Position(call.Pos()))
					}
				default:
					t.Errorf("%s: writeWords is passed a constructed deps (%T) rather than "+
						"the one the loop holds, so its vocabulary is whatever that literal "+
						"carries — which is how a site comes to mark nothing.",
						fset.Position(call.Pos()), arg)
				}
				return true
			})
		}
	}
	if sites < 3 {
		t.Fatalf("found %d writeWords call sites, want at least 3 (the lookup entry, the "+
			"sitting's prompt, the sitting's reveal) — the derivation is under-deriving "+
			"and this guard would certify nothing", sites)
	}
}
