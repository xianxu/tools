package main

import (
	"bytes"
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
	p := newPassage("don't hot-dog me, O'Brien", 0)
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
	p := newPassage("the slow precession\nof the equinox", 0)
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
	p := newPassage("the 漢字 of precession", 0)
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
	p := newPassage(sanitisePasteBody("the \x1b[31mslow\x1b[0m precession"), 0)
	got, ok := wordAtCell(p, 0, 4)
	if !ok || p.text(got) != "slow" {
		t.Errorf("wordAtCell(col 4) = %q ok=%v, want %q", p.text(got), ok, "slow")
	}
}

func TestAnEmptyPassageIsNotAPassage(t *testing.T) {
	if p := newPassage("   \n  ", 0); p.empty() != true {
		t.Error("a passage of only whitespace should report itself empty")
	}
	if p := newPassage("word", 0); p.empty() {
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
	got := passageText(newPassage("the slow precession of the equinox", 0), v, true)
	if !strings.Contains(got, knownOn+"equinox") {
		t.Errorf("the deck word was not coloured:\n%q", got)
	}
	if strings.Contains(got, knownOn+"precession") {
		t.Error("a word that is not in the deck was coloured")
	}
	if plain := passageText(newPassage("the slow equinox", 0), v, false); strings.Contains(plain, "\x1b") {
		t.Errorf("colour was emitted with colour off: %q", plain)
	}
}

// Every word is clickable, and Region.Line IS the passage line — which is what
// lets a click come back to a span without a second table to keep in step.
func TestPassageRegionsAddressEveryWord(t *testing.T) {
	p := newPassage("the slow precession\nof the equinox", 0)
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
	p := newPassage("the slow precession\nof the equinox", 0)
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
	p := newPassage("the slow precession of the equinox", 0)
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

// A passage WRAPS at construction, so a passage line is a buffer line is a
// Region line. Without it a long line runs off the right edge and the words past
// the margin cannot be clicked at all (operator-reported, with a screenshot).
func TestAPassageWrapsToTheTerminalWidth(t *testing.T) {
	long := "the slow precession of the equinox points westward along the ecliptic over millennia"
	p := newPassage(long, 30)
	if p.lineCount() < 3 {
		t.Fatalf("a %d-cell line at width 30 became %d lines", visibleCells(long), p.lineCount())
	}
	for i := range p.lines {
		if got := visibleCells(p.line(i)); got > 30 {
			t.Errorf("line %d is %d cells wide, past the margin: %q", i, got, p.line(i))
		}
	}
	// Broken at SPACES, never mid-word: the reader is going to click these.
	var words []string
	for i := range p.lines {
		for _, sp := range p.spans(i) {
			words = append(words, p.text(sp))
		}
	}
	if strings.Join(words, " ") != long {
		t.Errorf("wrapping changed the words:\n got %q\nwant %q", strings.Join(words, " "), long)
	}
}

// Width 0 means "do not wrap", which is what a test without a terminal wants —
// the same meaning RenderOpts.Width already carries.
func TestAPassageAtWidthZeroIsNotWrapped(t *testing.T) {
	p := newPassage("a fairly long line of text that would otherwise be broken up", 0)
	if p.lineCount() != 1 {
		t.Errorf("width 0 wrapped anyway: %d lines", p.lineCount())
	}
}

// Every word of a passage is clickable, so underlining them all says nothing.
// The underline is for a span that offers something its neighbours do not.
func TestPassageWordsAreNotUnderlined(t *testing.T) {
	if regionUnderlines(RegionPassageWord) {
		t.Error("passage words are underlined; in a passage that marks every word and reads as noise")
	}
	for _, k := range []RegionKind{RegionHeadword, RegionOriginLang, RegionWord} {
		if !regionUnderlines(k) {
			t.Errorf("%v lost its underline, so its affordance is invisible", k)
		}
	}
	line := markClickable("the slow precession", []Region{{Kind: RegionPassageWord, Text: "slow", Col: 4, Width: 4}})
	if strings.Contains(line, "\x1b[4m") {
		t.Errorf("markClickable underlined a passage word: %q", line)
	}
}

// A DRAG across a passage marks every word it covers, rather than copying the
// text. The reader is choosing what to ask about; taking it to the clipboard
// would answer a question they did not ask.
func TestADragAcrossAPassageMarksItsWords(t *testing.T) {
	p := newPassage("the slow precession of the equinox", 0)
	live := newLiveScreen(&bytes.Buffer{}, 24, 80)
	defer live.Stop()
	rows := make([]selectionRow, 24)
	rows[0] = selectionRow{selectable: true, styled: p.line(0), regions: passageRegions(p)}
	live.frame = newSelectionFrame(80, 24, rows)
	live.frameID, live.framePublished = 1, true

	got := live.passageWordsInLocked(selectionPoint{row: 0, col: 4}, selectionPoint{row: 0, col: 12})
	var words []string
	for _, r := range got {
		words = append(words, r.Text)
	}
	// Whole words at both ends: the drag starts inside "slow" and stops inside
	// "precession", and half a word is not something anyone can ask about.
	if strings.Join(words, " ") != "slow precession" {
		t.Errorf("drag covered %q, want %q", strings.Join(words, " "), "slow precession")
	}
}

// A drag OUTSIDE a passage still copies, which is the gesture everywhere else.
func TestADragOutsideAPassageStillCopies(t *testing.T) {
	live := newLiveScreen(&bytes.Buffer{}, 24, 80)
	defer live.Stop()
	rows := make([]selectionRow, 24)
	rows[0] = selectionRow{selectable: true, styled: "an ordinary line of output",
		regions: []Region{{Kind: RegionWord, Text: "ordinary", Col: 3, Width: 8}}}
	live.frame = newSelectionFrame(80, 24, rows)
	live.frameID, live.framePublished = 1, true

	if got := live.passageWordsInLocked(selectionPoint{row: 0, col: 0}, selectionPoint{row: 0, col: 20}); len(got) != 0 {
		t.Errorf("a drag over a deck word was taken as a passage drag: %v", got)
	}
}

// Dragging back over a marked run clears it — a drag and a click are ONE
// gesture, so either must be able to cancel the other.
func TestDraggingBackOverAMarkedRunClearsIt(t *testing.T) {
	p := newPassage("the slow precession of the equinox", 0)
	var m markSet
	for _, sp := range []passageSpan{p.spans(0)[1], p.spans(0)[2]} {
		m = m.toggle(sp)
	}
	if len(m.ordered()) != 2 {
		t.Fatalf("setup: marks = %v", m.ordered())
	}
	for _, sp := range []passageSpan{p.spans(0)[1], p.spans(0)[2]} {
		m = m.toggle(sp)
	}
	if !m.empty() {
		t.Errorf("dragging back over the run left %v", m.ordered())
	}
}
