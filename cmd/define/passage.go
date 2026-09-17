package main

import "strings"

// passageSpan is a range of one passage LINE, in bytes.
//
// Bytes rather than cells, because that is what wordRuns speaks and what the
// prompt renderer needs to bracket a mark in place. Cells belong to the screen,
// and the conversion between them has exactly one home: wordAtCell going in,
// cellRangesFor coming back out (#67 M3).
type passageSpan struct {
	line       int
	start, end int
}

// passage is the text being read, and everything derived from it.
//
// It is CHROME, not scrollback. screen.lines is append-only and immutable once
// written, and deck colour is baked in at write time — so a passage printed as
// ordinary output could never re-render, and "marks clear and the words you
// asked about turn green" (#67) would be impossible. It lives in the footer,
// which Draw rebuilds from source on every frame.
type passage struct {
	src   string
	lines []string
	runs  [][]wordRun
}

// newPassage takes text and derives its lines and word runs.
//
// The text is expected to have been through sanitisePasteBody already — the wire
// boundary strips escapes — but nothing here DEPENDS on that: the column table
// is built with the same escape-aware walk the rest of the program uses, so a
// passage that somehow carried an escape still maps clicks to the right word
// rather than silently to the wrong one.
func newPassage(text string) *passage {
	p := &passage{src: text, lines: strings.Split(text, "\n")}
	p.runs = make([][]wordRun, len(p.lines))
	for i, line := range p.lines {
		// wordRuns is THE tokeniser (highlight.go). A second one here would put
		// the same word in two states on one screen — "don't" as one token at
		// the prompt and two in the passage.
		p.runs[i] = wordRuns(line)
	}
	return p
}

func (p *passage) raw() string    { return p.src }
func (p *passage) lineCount() int { return len(p.lines) }
func (p *passage) line(i int) string {
	if i < 0 || i >= len(p.lines) {
		return ""
	}
	return p.lines[i]
}

// spans is every word of a line, as spans this package can carry around.
func (p *passage) spans(line int) []passageSpan {
	if line < 0 || line >= len(p.runs) {
		return nil
	}
	out := make([]passageSpan, 0, len(p.runs[line]))
	for _, r := range p.runs[line] {
		out = append(out, passageSpan{line: line, start: r.start, end: r.end})
	}
	return out
}

// text is the span's own characters.
func (p *passage) text(s passageSpan) string {
	line := p.line(s.line)
	if s.start < 0 || s.end > len(line) || s.start >= s.end {
		return ""
	}
	return line[s.start:s.end]
}

// empty reports a passage with nothing to read, which is not a passage. A paste
// of only whitespace should leave the reader where they were rather than pinning
// a blank region.
func (p *passage) empty() bool { return strings.TrimSpace(p.src) == "" }

// wordAtCell resolves a display COLUMN to the word under it.
//
// The bridge between two coordinate spaces that otherwise never meet: a click
// arrives as cells, and words are byte ranges. Everything that maps a pointer to
// a word goes through here, so a click and any later gesture cannot disagree
// about what was pointed at.
//
// The column must already be corrected for a continuation row — see
// wrappedColumn, which is the other half and is kept separate so the correction
// has one name rather than being inlined at each caller.
func wordAtCell(p *passage, line, col int) (passageSpan, bool) {
	if p == nil || line < 0 || line >= len(p.lines) || col < 0 {
		return passageSpan{}, false
	}
	at, ok := byteAtCell(p.lines[line], col)
	if !ok {
		return passageSpan{}, false
	}
	for _, r := range p.runs[line] {
		if at >= r.start && at < r.end {
			return passageSpan{line: line, start: r.start, end: r.end}, true
		}
	}
	return passageSpan{}, false // whitespace and punctuation are not words
}

// byteAtCell walks display units to find the byte offset a column lands in.
//
// It walks rather than indexing visibleIndex's table because it needs the
// INVERSE of that table, and a wide glyph occupies two columns for one offset —
// so the answer is "the unit whose cell range contains col", not "the offset
// whose column equals col". Written with nextDisplayUnit and escapeLen, the two
// functions that already own this arithmetic.
func byteAtCell(line string, col int) (int, bool) {
	c := 0
	for i := 0; i < len(line); {
		if skip := escapeLen(line[i:]); skip > 0 {
			i += skip
			continue
		}
		size, w := nextDisplayUnit(line[i:])
		if size == 0 {
			break
		}
		if col >= c && col < c+max(w, 1) {
			return i, true
		}
		c += w
		i += size
	}
	return 0, false
}

// wrappedColumn corrects a click's column for the row it landed on.
//
// The screen wraps a passage line across several frame rows, and FooterRowAt
// reports WHICH of them was hit (screen.go) precisely so a caller can do this:
// column 4 of the first continuation is column width+4 of the line. A passage
// wraps on any normal terminal, so this is the common path.
//
// One named function rather than the arithmetic inlined at each call site,
// because two copies of an off-by-one is how a click comes to mark a word the
// reader did not point at.
func wrappedColumn(offset, col, width int) int {
	if offset <= 0 || width <= 0 {
		return col
	}
	return offset*width + col
}

// pasteIsPassage decides whether a paste is reading material or a headword.
//
// It reuses readsAsQuestion's FOUR-WORD FLOOR rather than inventing a threshold
// (#57): one to three words on a single line is a headword shape — `hot dog` and
// `a priori` are dictionary entries, and someone pasting `sycophantic` to look it
// up should get a lookup. Four or more words, or any newline at all, is prose.
//
// The same reasoning the console classifier uses, applied to a different
// question: not "is this a word" but "is this something to read".
func pasteIsPassage(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	if strings.Contains(text, "\n") {
		return true
	}
	return len(strings.Fields(text)) >= 4
}

// passageFooter renders the passage for the pinned region, one entry per LOGICAL
// line — the screen does the wrapping, which is what makes FooterRowAt's
// (entry, offset) answer meaningful.
//
// Deck colour goes on here, at render time, and the footer is rebuilt from
// source on every frame — which is the whole reason the passage lives in the
// footer rather than the buffer. A word that enters the deck between two frames
// is green in the second one, with nothing to invalidate.
func passageFooter(p *passage, v Vocabulary, colour bool) []string {
	if p == nil || p.empty() {
		return nil
	}
	out := make([]string, 0, p.lineCount())
	for i := range p.lines {
		line := p.line(i)
		if colour {
			// highlightRegion is the shared entry seven other callers already
			// use; open-coding the span loop here would be a fourth copy of the
			// ANSI re-open rule.
			line = highlightRegion(line, v, knownOn, "")
		}
		out = append(out, line)
	}
	return out
}
