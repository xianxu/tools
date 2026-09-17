package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
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
		sp, ok := (&session{passage: p}).passageSpanAt(r.Line, r)
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

	sp, ok := (&session{passage: p}).passageSpanAt(0, r)
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

// A DRAG produces ONE span — a phrase when it covers several words — which is
// what the operator asked for and what marksForDrag implements.
//
// It did NOT, for one release: production toggled each covered word separately,
// so a drag over "at the zenith of" sent four marks and admitted `at`, `the` and
// `of` to the deck as words. marksForDrag had three passing tests and no
// production caller at all.
func TestADragAcrossAPassageProducesOneSpan(t *testing.T) {
	p := newPassage("he stopped at the zenith of the arc", 0)
	got := marksForDrag(p, passageCell{line: 0, col: 11}, passageCell{line: 0, col: 27})
	if len(got) != 1 {
		t.Fatalf("a drag produced %d marks, want ONE span: %v", len(got), got)
	}
	if text := p.text(got[0]); text != "at the zenith of" {
		t.Errorf("drag marked %q, want %q", text, "at the zenith of")
	}
	// And the span survives into the prompt as a single bracketed phrase.
	var m markSet
	m = m.toggle(got[0])
	if want := "he stopped " + selOpen + "at the zenith of" + selClose + " the arc"; markedPassageText(p, m) != want {
		t.Errorf("prompt =\n%s\nwant\n%s", markedPassageText(p, m), want)
	}
}

// The ANCHOR decides whether a drag marks. A drag that starts in an answer and
// ends over the passage is a COPY — treating it as a mark would swallow the copy
// the reader asked for.
func TestADragThatStartsOutsideThePassageIsNotAMark(t *testing.T) {
	live := newLiveScreen(&bytes.Buffer{}, 24, 80)
	defer live.Stop()
	p := newPassage("the slow precession of the equinox", 0)
	rows := make([]selectionRow, 24)
	rows[0] = selectionRow{selectable: true, styled: "an ordinary line of output"}
	rows[1] = selectionRow{selectable: true, styled: p.line(0), regions: passageRegions(p)}
	live.frame = newSelectionFrame(80, 24, rows)
	live.frameID, live.framePublished = 1, true
	// The buffer behind the frame, and the LIVE passage's range within it:
	// regions alone are not enough, because a superseded passage's rows carry
	// them forever.
	live.s.lines = []string{"an ordinary line of output", p.line(0)}
	live.s.rows, live.s.cols = 24, 80
	live.SetPassage(1, 2, nil)

	if _, _, ok := live.passageDragLocked(selectionPoint{row: 0, col: 0}, selectionPoint{row: 1, col: 10}); ok {
		t.Error("a drag starting in ordinary output was taken as a passage drag; its copy is lost")
	}
	if _, _, ok := live.passageDragLocked(selectionPoint{row: 1, col: 4}, selectionPoint{row: 1, col: 12}); !ok {
		t.Error("a drag starting in the passage was not taken as one")
	}
}

// A superseded passage's regions must not resolve against the CURRENT passage.
// Measured before the fix: clicking `alpha` in the old passage marked `zulu` in
// the new one, because a Region's Line is relative to its own render and nothing
// removes the old regions.
func TestAStalePassagesRegionsDoNotMarkTheCurrentOne(t *testing.T) {
	first := newPassage("alpha beta gamma delta epsilon", 0)
	sess := &session{passage: first, passageBase: 10}
	r := passageRegions(first)[0]

	if _, ok := sess.passageSpanAt(10, r); !ok {
		t.Fatal("a click on the current passage did not resolve")
	}
	// A second paste lands further down the buffer; the first passage's regions
	// are still in the screen's map at their old lines.
	sess.passage = newPassage("zulu yankee xray whiskey victor", 0)
	sess.passageBase = 40
	if sp, ok := sess.passageSpanAt(10, r); ok {
		t.Errorf("a stale region resolved to %q of the current passage", sess.passage.text(sp))
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

// THE DONE-WHEN ROW: "the token AFTER the mark still carries the style it had".
//
// Asserted on the ESCAPES, not on stripped text — stripping escapes is precisely
// what hides a lost style, which is why the row says so in as many words. The
// earlier test checked only that the mark was re-asserted INSIDE the span, so
// replacing paintMarks' `sgrOff + style.resume()` with a bare `sgrOff` left the
// whole suite green while every word after a mark rendered plain.
func TestTheTokenAfterAMarkKeepsItsStyle(t *testing.T) {
	// The deck colour opens before the mark and must still be in force after it.
	line := knownOn + "equinox precession" + sgrOff
	got := paintMarks(line, []cellRange{{start: 0, end: 7}})

	at := strings.Index(got, "precession")
	if at < 0 {
		t.Fatalf("the text was lost: %q", got)
	}
	before := got[:at]
	closed := strings.LastIndex(before, sgrOff)
	if closed < 0 {
		t.Fatalf("the mark never closed: %q", got)
	}
	// Between closing the mark and the next token, the producer's style must be
	// RESUMED. Without it the rest of the line renders plain.
	if !strings.Contains(before[closed:], knownOn) {
		t.Errorf("the style was not resumed after the mark closed, so %q renders plain:\n%q",
			"precession", got)
	}
}

// A drag in a SUPERSEDED passage must not mark the current one. Its rows still
// carry RegionPassageWord — screen.regions is never pruned — so the region kind
// alone answers yes forever; only the live range can refuse it.
//
// Measured before the fix: a drag anchored on the old passage's row marked "zulu
// yankee xray" of the CURRENT one, which then reached the prompt and the deck.
func TestADragInASupersededPassageIsNotAMark(t *testing.T) {
	old := newPassage("alpha beta gamma", 0)
	live := newLiveScreen(&bytes.Buffer{}, 24, 80)
	defer live.Stop()
	rows := make([]selectionRow, 24)
	rows[0] = selectionRow{selectable: true, styled: old.line(0), regions: passageRegions(old)}
	live.frame = newSelectionFrame(80, 24, rows)
	live.frameID, live.framePublished = 1, true
	live.s.lines = []string{old.line(0), "", "zulu yankee xray"}
	live.s.rows, live.s.cols = 24, 80
	// The LIVE passage is further down the buffer; row 0 belongs to the old one.
	live.SetPassage(2, 3, nil)

	if _, _, ok := live.passageDragLocked(selectionPoint{row: 0, col: 0}, selectionPoint{row: 0, col: 10}); ok {
		t.Error("a drag in a superseded passage was taken as a passage drag; it would mark the current one")
	}
}

// passageCell REFUSES a row the live passage does not own, rather than clamping
// it to the nearest line that exists — which is how an out-of-passage drag end
// landed inside the current passage.
func TestPassageCellRefusesARowItDoesNotOwn(t *testing.T) {
	sess := &session{passage: newPassage("alpha beta\ngamma delta", 0), passageBase: 40}
	if _, ok := sess.passageCell(selectionPoint{row: 41, col: 3}); !ok {
		t.Error("a row inside the passage was refused")
	}
	for _, row := range []int{0, 39, 42, 1000} {
		if c, ok := sess.passageCell(selectionPoint{row: row, col: 3}); ok {
			t.Errorf("row %d was accepted as passage line %d; clamping is how a stale drag lands inside", row, c.line)
		}
	}
}

// "A passage was once pasted" is not authority over Enter for the rest of the
// session. Once it has scrolled away a bare Enter must replay again, and the
// nudge must stop pointing at something off-screen.
func TestEnterReplaysAgainOnceThePassageHasScrolledAway(t *testing.T) {
	sess := &session{current: "sycophantic", passage: newPassage("the slow precession", 0), passageBase: 5}

	onScreen := sess.lineState(sess.passageVisible(0, 24))
	if got := parseREPLLine("", onScreen); got.kind != cmdNothing || got.note != noteNothingMarked {
		t.Errorf("with the passage on screen, bare Enter = %v %q; want the nudge", got.kind, got.note)
	}
	scrolledAway := sess.lineState(sess.passageVisible(100, 124))
	if got := parseREPLLine("", scrolledAway); got.kind != cmdReplay {
		t.Errorf("after the passage scrolled away, bare Enter = %v; want cmdReplay", got.kind)
	}
}

// The passage is a RECORD: it goes into the buffer and scrolls away like a
// definition or an answer, not into the footer where it was welded to the prompt
// forever (operator-reported, with a screenshot).
func TestThePassageIsWrittenToTheBufferNotTheFooter(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(Key{Kind: KeyPaste, Raw: []byte("the slow precession of the equinox")})
	runEditor(t.Context(), ks, nil, rig.deps, opt,
		console{view: view, finish: finish, stdout: &out, stderr: &errb})

	if !strings.Contains(out.String(), "precession") {
		t.Errorf("the passage never reached the buffer:\n%s", out.String())
	}
	if joined := strings.Join(view.footer(), "\n"); strings.Contains(joined, "precession") {
		t.Errorf("the passage is in the footer, so it will never scroll away: %q", joined)
	}
}

// MARK WINS over deck colour. An explicit fg/bg pair overrides the foreground it
// resumes over anyway; painting the mark last makes that structural rather than a
// rule someone has to remember.
func TestAMarkedDeckWordRendersAsAMarkNotAsADeckWord(t *testing.T) {
	v := &memVocabulary{}
	v.Add("equinox")
	p := newPassage("the slow equinox", 0)
	line := passageText(p, v, true)
	if !strings.Contains(line, knownOn+"equinox") {
		t.Fatalf("setup: the deck word was not coloured: %q", line)
	}
	col, width, ok := spanCells(p.line(0), p.spans(0)[2])
	if !ok {
		t.Fatal("setup: could not locate the word's cells")
	}
	got := paintMarks(line, []cellRange{{start: col, end: col + width}})
	at := strings.Index(got, "equinox")
	if at < 0 {
		t.Fatalf("the word was lost: %q", got)
	}
	// The mark opens last before the word, so it is what the terminal shows.
	before := got[:at]
	if strings.LastIndex(before, markOn) < strings.LastIndex(before, knownOn) {
		t.Errorf("the deck colour won over the mark:\n%q", got)
	}
}

// An UNBREAKABLE token — a URL, a long identifier — has no space for wrapText to
// break at, so it came back whole and over the margin; clipVisible then truncated
// it at paint time and the tail was neither readable nor clickable. That is the
// half of the operator's wrapping report that survived the first fix.
func TestAnUnbreakableTokenIsBrokenAtTheMargin(t *testing.T) {
	url := "https://example.com/a/very/long/path/that/never/breaks/anywhere"
	p := newPassage("see "+url+" for more", 30)
	for i := range p.lines {
		if got := visibleCells(p.line(i)); got > 30 {
			t.Errorf("line %d is %d cells, past the margin — its tail is unclickable: %q", i, got, p.line(i))
		}
	}
	// Nothing is lost: every character is still there to click.
	var joined string
	for i := range p.lines {
		joined += p.line(i)
	}
	if !strings.Contains(strings.ReplaceAll(joined, " ", ""), strings.ReplaceAll(url, " ", "")) {
		t.Errorf("the token was truncated rather than broken:\n%q", joined)
	}
}

// A wide glyph is never cut in half by the hard break.
func TestHardBreakNeverSplitsAWideGlyph(t *testing.T) {
	for _, line := range hardBreak(strings.Repeat("漢", 20), 7) {
		if visibleCells(line) > 7 {
			t.Errorf("line %q is %d cells, over the margin", line, visibleCells(line))
		}
		if !utf8.ValidString(line) {
			t.Errorf("the break cut a rune: %q", line)
		}
	}
}
