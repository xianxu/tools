package main

import (
	"slices"
	"strings"
	"testing"
)

// The passage tokenises through wordRuns, the ONE tokeniser (highlight.go).
// Asserted by cases only a shared tokeniser gets right: an apostrophe and a
// hyphen are INSIDE a word, so each of these is a single token. A second
// tokeniser here would split them, and the same word would then be one thing at
// the prompt and another in the passage.
func TestPassageTokenisesLikeTheRestOfTheProgram(t *testing.T) {
	p := newPassage("don't hot-dog me, O'Brien")
	var got []string
	for _, s := range p.spans(0) {
		got = append(got, p.text(s))
	}
	want := []string{"don't", "hot-dog", "me", "O'Brien"}
	if !slices.Equal(got, want) {
		t.Errorf("spans = %q, want %q — this is a second tokeniser, not wordRuns", got, want)
	}
}

func TestPassageSplitsOnNewlines(t *testing.T) {
	p := newPassage("the slow precession\nof the equinox")
	if got := p.lineCount(); got != 2 {
		t.Fatalf("lineCount = %d, want 2", got)
	}
	if got := p.line(1); got != "of the equinox" {
		t.Errorf("line(1) = %q", got)
	}
}

// A click carries a display COLUMN; words are byte ranges. This is the bridge
// between the two coordinate spaces, and the case that proves it is a WIDE
// glyph, where cells and bytes diverge. Columns are ZERO-BASED — the space row
// pins that, since it would resolve to a word under any off-by-one.
func TestWordAtCellSnapsToTheWordUnderTheColumn(t *testing.T) {
	// cols:        0123456789…
	// "the 漢字 of precession"
	//  the=0-2  漢=4-5  字=6-7  of=9-10  precession=12-21
	p := newPassage("the 漢字 of precession")
	for _, tc := range []struct {
		name string
		col  int
		want string
	}{
		{"inside the first word", 1, "the"},
		{"first cell of a wide glyph", 4, "漢字"},
		{"second cell of the same wide glyph", 5, "漢字"},
		{"first cell of the next wide glyph", 6, "漢字"},
		{"a word after the wide run", 12, "precession"},
		{"the last cell of the last word", 21, "precession"},
	} {
		got, ok := wordAtCell(p, 0, tc.col)
		if !ok {
			t.Errorf("%s: wordAtCell(col %d) found nothing, want %q", tc.name, tc.col, tc.want)
			continue
		}
		if p.text(got) != tc.want {
			t.Errorf("%s: wordAtCell(col %d) = %q, want %q", tc.name, tc.col, p.text(got), tc.want)
		}
	}
	for _, col := range []int{3, 8, 11, 22, 900} {
		if s, ok := wordAtCell(p, 0, col); ok {
			t.Errorf("col %d resolved to %q; whitespace and the void are not words", col, p.text(s))
		}
	}
	if _, ok := wordAtCell(p, 7, 0); ok {
		t.Error("a line that does not exist resolved to a word")
	}
}

// A click on a CONTINUATION row carries a column in THAT ROW's coordinate space,
// not the logical line's: column 4 of a continuation is column cols+4 of the
// line. FooterRowAt hands back the offset precisely so a caller can correct for
// it, and a passage wraps on any normal terminal — so this is the common case,
// not an edge. One named place for the arithmetic, or two callers will disagree.
func TestWrappedColumnCorrectsForTheContinuationRow(t *testing.T) {
	for _, tc := range []struct {
		offset, col, width, want int
	}{
		{0, 4, 20, 4},  // the first row is its own coordinate space
		{1, 4, 20, 24}, // one wrap in
		{2, 0, 20, 40}, // two wraps in, at the left edge
		{1, 0, 80, 80}, // a wider terminal moves it further
	} {
		if got := wrappedColumn(tc.offset, tc.col, tc.width); got != tc.want {
			t.Errorf("wrappedColumn(%d, %d, %d) = %d, want %d",
				tc.offset, tc.col, tc.width, got, tc.want)
		}
	}
}

// The passage is built from already-sanitised text (the scanner strips escapes
// at the wire boundary), but it must not ASSUME that: a passage that reached a
// column table through an escape would map every later click to the wrong word.
func TestAPassageWithAnEscapeStillMapsColumnsCorrectly(t *testing.T) {
	p := newPassage(sanitisePasteBody("the \x1b[31mslow\x1b[0m precession"))
	got, ok := wordAtCell(p, 0, 4)
	if !ok || p.text(got) != "slow" {
		t.Errorf("wordAtCell(col 4) = %q ok=%v, want %q", p.text(got), ok, "slow")
	}
}

func TestAnEmptyPassageIsNotAPassage(t *testing.T) {
	if p := newPassage("   \n  "); p.empty() != true {
		t.Error("a passage of only whitespace should report itself empty")
	}
	if p := newPassage("word"); p.empty() {
		t.Error("a passage with a word reported itself empty")
	}
}

// --- the passage on screen --------------------------------------------------

// A paste of reading material becomes the PASSAGE; a paste of a headword goes
// into the line, which is what it was already doing and what someone pasting
// `sycophantic` to look it up expects.
func TestAPasteIsClassifiedByShapeNotByBeingAPaste(t *testing.T) {
	for _, tc := range []struct {
		in      string
		passage bool
		why     string
	}{
		{"sycophantic", false, "one word is a headword"},
		{"hot dog", false, "two words is still a headword — the dictionary has this one"},
		{"a priori thing", false, "three words stays a lookup, matching readsAsQuestion's floor"},
		{"the slow precession of the equinox", true, "four or more words is prose"},
		{"two\nlines", true, "a newline is reading material whatever its length"},
		{"   ", false, "whitespace is nothing"},
	} {
		if got := pasteIsPassage(tc.in); got != tc.passage {
			t.Errorf("pasteIsPassage(%q) = %v, want %v — %s", tc.in, got, tc.passage, tc.why)
		}
	}
}

// --- the passage as a record ------------------------------------------------

// Deck colour is baked in AT WRITE TIME, exactly as a definition's is: the
// passage is a record of something read, and the deck it was read against is
// part of that record.
func TestThePassageIsWrittenWithDeckColour(t *testing.T) {
	v := &memVocabulary{}
	v.Add("equinox")
	got := passageText(newPassage("the slow precession of the equinox"), v, true)
	if !strings.Contains(got, knownOn+"equinox") {
		t.Errorf("the deck word was not coloured:\n%q", got)
	}
	if strings.Contains(got, knownOn+"precession") {
		t.Error("a word that is not in the deck was coloured")
	}
	if plain := passageText(newPassage("the slow equinox"), v, false); strings.Contains(plain, "\x1b") {
		t.Errorf("colour was emitted with colour off: %q", plain)
	}
}

// Every word is clickable, and Region.Line IS the passage line — which is what
// lets a click come back to a span without a second table to keep in step.
func TestPassageRegionsAddressEveryWord(t *testing.T) {
	p := newPassage("the slow precession\nof the equinox")
	rs := passageRegions(p)
	if len(rs) != 6 {
		t.Fatalf("got %d regions, want one per word", len(rs))
	}
	for _, r := range rs {
		if r.Kind != RegionPassageWord {
			t.Errorf("region %q has kind %v", r.Text, r.Kind)
		}
		sp, ok := passageSpanOf(p, r)
		if !ok || p.text(sp) != r.Text {
			t.Errorf("region %q did not round-trip to its span (got %q)", r.Text, p.text(sp))
		}
	}
	if rs[3].Line != 1 {
		t.Errorf("the second line's first region has Line %d, want 1", rs[3].Line)
	}
}

// The mark is painted over FINISHED bytes and RE-ASSERTED after a producer SGR,
// which is what lets it survive the deck colour already baked into the line.
// ANSI does not nest; this does not rely on it.
func TestPaintMarksSurvivesTheDeckColourUnderIt(t *testing.T) {
	line := "the " + knownOn + "equinox" + sgrOff + " tonight"
	got := paintMarks(line, []cellRange{{start: 4, end: 11}})
	at := strings.Index(got, knownOn)
	if at < 0 {
		t.Fatalf("the producer's own colour was dropped: %q", got)
	}
	if !strings.HasPrefix(got[at+len(knownOn):], markOn) {
		t.Errorf("the mark was not re-asserted after the producer SGR; the rest of the span loses it:\n%q", got)
	}
	if stripEscapes(got) != stripEscapes(line) {
		t.Errorf("painting changed the text: %q", stripEscapes(got))
	}
}

// The session owns which spans are marked; this is the one translation into the
// screen's coordinates, so the two cannot drift.
func TestMarkCellRangesTranslatesToBufferLines(t *testing.T) {
	p := newPassage("the slow precession\nof the equinox")
	m := markSet{}.toggle(p.spans(1)[2]) // "equinox", on the second line
	got := markCellRanges(p, m, 40)
	if len(got) != 1 {
		t.Fatalf("got %v, want one line's worth", got)
	}
	rs, ok := got[41]
	if !ok {
		t.Fatalf("marks landed on %v, want buffer line 41 (base 40 + passage line 1)", got)
	}
	if len(rs) != 1 || rs[0].start != 7 || rs[0].end != 14 {
		t.Errorf("cell range = %v, want the cells of %q", rs, "equinox")
	}
}

// A click marks the word under it, and clicking it again unmarks it: the gesture
// is a toggle, and a click-mark and a drag-mark are the same kind of thing.
func TestAClickOnAPassageWordMarksIt(t *testing.T) {
	p := newPassage("the slow precession of the equinox")
	sess := &session{passage: p}
	r := passageRegions(p)[2]

	sp, ok := passageSpanOf(p, r)
	if !ok || p.text(sp) != "precession" {
		t.Fatalf("the region did not resolve to precession: %q", p.text(sp))
	}
	sess.marks = sess.marks.toggle(sp)
	if got := sess.marks.ordered(); len(got) != 1 {
		t.Fatalf("marks = %v", got)
	}
	sess.marks = sess.marks.toggle(sp)
	if !sess.marks.empty() {
		t.Error("clicking a marked word did not unmark it")
	}
}
