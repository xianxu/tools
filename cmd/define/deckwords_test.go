package main

import (
	"os"
	"strings"
	"testing"
)

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
func TestClozeOptionsAreClickableButNotColoured(t *testing.T) {
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
	var withColour, without strings.Builder
	writeWords(&withColour, "the keel", nil, v, true, surfaceProse, "")
	writeWords(&without, "the keel", nil, v, false, surfaceProse, "")

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
	writeWords(rw, "keel\nmesa", nil, v, true, surfaceDeck, "")
	writeWords(&out, "keel\nmesa", nil, v, true, surfaceDeck, "")

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
