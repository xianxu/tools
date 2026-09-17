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

// The passage renders under the NORMAL rules, which is what later makes "marks
// clear and the words you asked about turn green" work: the footer is rebuilt
// from source every frame, so the deck it is coloured against is today's.
func TestThePassageRendersWithDeckColour(t *testing.T) {
	v := &memVocabulary{}
	v.Add("equinox")
	got := passageFooter(newPassage("the slow precession of the equinox"), markSet{}, v, true)
	if len(got) != 1 {
		t.Fatalf("passageFooter returned %d lines, want 1", len(got))
	}
	if !strings.Contains(got[0], knownOn+"equinox") {
		t.Errorf("the deck word was not coloured:\n%q", got[0])
	}
	if strings.Contains(got[0], knownOn+"precession") {
		t.Error("a word that is not in the deck was coloured")
	}
}

// Clicks must work with colour OFF, so the footer is built against
// deckVocabulary rather than vocabularyFor — the distinction vocab.go draws for
// exactly this reason. With colour off the text is plain but the words are still
// there to point at.
func TestThePassageRendersWithoutColour(t *testing.T) {
	v := &memVocabulary{}
	v.Add("equinox")
	got := passageFooter(newPassage("the slow equinox"), markSet{}, v, false)
	if strings.Contains(got[0], "\x1b") {
		t.Errorf("colour was emitted with colour off: %q", got[0])
	}
	if got[0] != "the slow equinox" {
		t.Errorf("line = %q, want the plain text", got[0])
	}
}

// THE REGRESSION a first draft of the plan would have shipped. Two of the three
// Draw sites pass a nil footer to blank the PROMPT while the loop is working —
// correctly, since a prompt drawn while nothing reads keys invites typing at a
// line that is not there. But blanking the passage with it would make it vanish
// for the whole lookup/answer window: exactly the interval the feature exists
// for.
func TestThePassageSurvivesALookup(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(append([]Key{{Kind: KeyPaste, Raw: []byte("the slow precession of the equinox")}},
		append(runes("sycophantic"), Key{Kind: KeyEnter})...)...)
	runEditor(t.Context(), ks, nil, rig.deps, opt,
		console{view: view, finish: finish, stdout: &out, stderr: &errb})

	joined := strings.Join(view.footer(), "\n")
	if !strings.Contains(joined, "precession") {
		t.Errorf("the passage was gone after a lookup; last footer = %q", joined)
	}
}

// A passage taller than the rows available is DROPPED at the bottom by
// fitFooter, and FooterRowAt refuses a row that was not drawn — its own comment
// says why: "inventing an entry for it would mark a word that is not on screen".
//
// This pins that the passage path does not route around that. The assertion is
// on the SEAM rather than on pixels: every entry the screen reports must be one
// the passage actually has, whatever the terminal's height.
func TestAClickNeverResolvesToARowThatWasNotDrawn(t *testing.T) {
	p := newPassage(strings.Repeat("a line of the passage\n", 40))
	view := paintInto(&bytes.Buffer{})
	// The screen drew only the first three entries; rows below are not mapped.
	view.footerAt(0, 0)
	view.footerAt(1, 1)
	view.footerAt(2, 2)

	for row := 0; row < 10; row++ {
		entry, offset, ok := view.FooterRowAt(row)
		if !ok {
			continue // a row that was not drawn resolves to nothing, which is right
		}
		if entry >= p.lineCount() {
			t.Fatalf("row %d resolved to entry %d, which the passage does not have", row, entry)
		}
		if _, found := wordAtCell(p, entry, wrappedColumn(offset, 0, 20)); !found && offset == 0 {
			t.Errorf("row %d resolved to entry %d offset %d, which holds no word", row, entry, offset)
		}
	}
}

// --- marks on screen --------------------------------------------------------

// MARK WINS over deck colour. An explicit pair would override the foreground it
// resumes over anyway; making it structural — one styled span per word, closed —
// means nothing nests and no rule has to be remembered.
func TestAMarkedDeckWordRendersAsAMarkNotAsADeckWord(t *testing.T) {
	v := &memVocabulary{}
	v.Add("equinox")
	p := newPassage("the slow equinox")
	m := markSet{}.toggle(p.spans(0)[2])

	got := passageFooter(p, m, v, true)[0]
	if !strings.Contains(got, markOn+"equinox") {
		t.Errorf("the marked word is not painted as a mark:\n%q", got)
	}
	if strings.Contains(got, knownOn+"equinox") {
		t.Error("the marked word kept its deck colour; the mark must win")
	}
}

// A mark is drawn with colour OFF too: it is not decoration, it is the only
// thing on screen saying what the next Enter will ask about.
func TestAMarkIsDrawnWithColourOff(t *testing.T) {
	p := newPassage("the slow equinox")
	m := markSet{}.toggle(p.spans(0)[1])
	if got := passageFooter(p, m, nil, false)[0]; !strings.Contains(got, markOn+"slow") {
		t.Errorf("no mark with colour off: %q", got)
	}
}

// Concatenation reproduces the line exactly once escapes are stripped — the same
// invariant highlightSpans holds, because a renderer that loses a byte corrupts
// the passage silently.
func TestRenderingAPassageLineLosesNothing(t *testing.T) {
	v := &memVocabulary{}
	v.Add("equinox")
	for _, line := range []string{"the slow equinox", "don't hot-dog me, O'Brien", "  leading and trailing  ", ""} {
		p := newPassage(line)
		if p.empty() {
			continue // an empty passage renders nothing, which is its own row above
		}
		m := markSet{}
		if sp := p.spans(0); len(sp) > 0 {
			m = m.toggle(sp[0])
		}
		got := stripEscapes(passageFooter(p, m, v, true)[0])
		if got != line {
			t.Errorf("rendering %q produced %q", line, got)
		}
	}
}

// The three coordinate spaces compose only in markClickedWord, and this is the
// case that would catch an off-by-one: a click on a CONTINUATION row.
func TestAClickInThePassageMarksTheWordUnderIt(t *testing.T) {
	view := paintInto(&bytes.Buffer{})
	view.footerAt(0, 0)
	sess := &session{passage: newPassage("the slow precession of the equinox")}

	if !markClickedWord(view, sess, pointerClick{footer: true, footerEntry: 0, point: selectionPoint{col: 9}}) {
		t.Fatal("a click inside the passage was not taken as one")
	}
	if got := sess.marks.ordered(); len(got) != 1 || sess.passage.text(got[0]) != "precession" {
		t.Fatalf("marks = %v, want precession", got)
	}
	// Clicking it again unmarks it: the gesture is a toggle.
	markClickedWord(view, sess, pointerClick{footer: true, footerEntry: 0, point: selectionPoint{col: 9}})
	if !sess.marks.empty() {
		t.Error("clicking a marked word did not unmark it")
	}
}

// COLLISION 3, held to its scope: outside the passage every click keeps the
// meaning it has today, so markClickedWord must decline it.
func TestAClickOutsideThePassageIsNotAMark(t *testing.T) {
	view := paintInto(&bytes.Buffer{})
	sess := &session{passage: newPassage("the slow precession")}
	if markClickedWord(view, sess, pointerClick{footer: false, point: selectionPoint{col: 1}}) {
		t.Error("a click on the buffer was taken as a mark")
	}
	if markClickedWord(view, sess, pointerClick{footer: true, footerEntry: 7, point: selectionPoint{col: 1}}) {
		t.Error("a click on the command menu was taken as a mark")
	}
	if markClickedWord(&recordDisplay{w: &bytes.Buffer{}}, &session{}, pointerClick{footer: true}) {
		t.Error("a click with no passage was taken as a mark")
	}
}
