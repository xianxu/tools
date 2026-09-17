package main

import "testing"

// Marks are a SET with a TOGGLE, not an append-only list: clicking a marked word
// unmarks it (#67). Ordered by POSITION rather than by insertion, because the
// prompt brackets them in place — a reader marking right-to-left must produce the
// same request as one marking left-to-right, or the same two words asked about
// twice are two different questions.
func TestMarkSetTogglesAndOrdersByPosition(t *testing.T) {
	var m markSet
	later := passageSpan{line: 0, start: 20, end: 30}
	earlier := passageSpan{line: 0, start: 4, end: 9}
	m = m.toggle(later)
	m = m.toggle(earlier)

	got := m.ordered()
	if len(got) != 2 || got[0] != earlier || got[1] != later {
		t.Fatalf("ordered() = %v, want position order [%v %v]", got, earlier, later)
	}
	if m = m.toggle(earlier); len(m.ordered()) != 1 || m.ordered()[0] != later {
		t.Errorf("toggling a marked span did not remove it: %v", m.ordered())
	}
}

// Later LINES sort after earlier ones, whatever order they were marked in.
func TestMarkSetOrdersAcrossLines(t *testing.T) {
	var m markSet
	m = m.toggle(passageSpan{line: 2, start: 0, end: 3})
	m = m.toggle(passageSpan{line: 0, start: 8, end: 11})
	m = m.toggle(passageSpan{line: 0, start: 0, end: 3})
	got := m.ordered()
	want := []passageSpan{{line: 0, start: 0, end: 3}, {line: 0, start: 8, end: 11}, {line: 2, start: 0, end: 3}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ordered() = %v, want %v", got, want)
		}
	}
}

// A click-mark and a drag-mark are the SAME kind of thing — a click is a one-word
// drag (#67). If they were distinct the toggle could not cancel one with the
// other, and the set would accumulate duplicates of the same word.
func TestAClickMarkAndADragMarkAreOneKind(t *testing.T) {
	var m markSet
	span := passageSpan{line: 1, start: 0, end: 5}
	m = m.toggle(span) // as if by click
	m = m.toggle(span) // as if by drag over exactly that span
	if len(m.ordered()) != 0 {
		t.Error("a drag over a click-marked span did not cancel it; they are not one kind")
	}
}

// Marks CLEAR after an ask (#67): the transient state converts into deck
// membership, and a bare Enter afterwards must find nothing to re-ask.
func TestMarkSetClears(t *testing.T) {
	var m markSet
	m = m.toggle(passageSpan{line: 0, start: 0, end: 3})
	if m.empty() {
		t.Fatal("a set with a mark reported itself empty")
	}
	if m = m.clear(); !m.empty() {
		t.Error("clear left marks behind; a bare Enter would re-ask")
	}
}

// A drag runs from anchor to end in CELLS and may cross lines. It becomes one
// mark per line, snapped OUTWARD to whole words — the unit everywhere else is a
// word, so a drag starting mid-word marks that whole word rather than half of it.
func TestMarksForDragSnapsToWordsAndSplitsPerLine(t *testing.T) {
	p := newPassage("the slow precession\nof the equinox")

	// Within one line, starting and ending mid-word.
	got := marksForDrag(p, passageCell{line: 0, col: 5}, passageCell{line: 0, col: 12})
	if len(got) != 1 {
		t.Fatalf("a single-line drag produced %d marks, want 1: %v", len(got), got)
	}
	if text := p.text(got[0]); text != "slow precession" {
		t.Errorf("drag marked %q, want %q — the ends must snap outward to whole words", text, "slow precession")
	}

	// Across a line break: one mark per line, because a bracket cannot span one.
	got = marksForDrag(p, passageCell{line: 0, col: 9}, passageCell{line: 1, col: 4})
	if len(got) != 2 {
		t.Fatalf("a drag across a line break produced %d marks, want 2: %v", len(got), got)
	}
	if a, b := p.text(got[0]), p.text(got[1]); a != "precession" || b != "of the" {
		t.Errorf("drag marked %q and %q, want %q and %q", a, b, "precession", "of the")
	}
}

// A drag over only whitespace marks nothing rather than marking a zero-width
// span: an empty bracket in the prompt would be a question about nothing.
func TestADragOverWhitespaceMarksNothing(t *testing.T) {
	p := newPassage("a    b")
	if got := marksForDrag(p, passageCell{line: 0, col: 2}, passageCell{line: 0, col: 3}); len(got) != 0 {
		t.Errorf("a drag over whitespace produced %v", got)
	}
}

// A backwards drag is the same selection as a forwards one — the reader dragged
// right-to-left, which says nothing about what they meant.
func TestABackwardsDragIsTheSameSelection(t *testing.T) {
	p := newPassage("the slow precession")
	fwd := marksForDrag(p, passageCell{line: 0, col: 4}, passageCell{line: 0, col: 12})
	back := marksForDrag(p, passageCell{line: 0, col: 12}, passageCell{line: 0, col: 4})
	if len(fwd) != 1 || len(back) != 1 || fwd[0] != back[0] {
		t.Errorf("forwards %v != backwards %v", fwd, back)
	}
}
