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

// wrapPassageLines breaks each logical line to width at SPACES.
//
// wrapText is the shared breaker — the terminal would wrap for us, but at the
// COLUMN, splitting words mid-syllable, which is exactly what a passage must not
// do since the reader is going to click the words.
func wrapPassageLines(text string, width int) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if width <= 0 {
			out = append(out, line)
			continue
		}
		out = append(out, strings.Split(wrapText(line, width, 0), "\n")...)
	}
	return out
}

// newPassage takes text and derives its WRAPPED lines and their word runs.
//
// Wrapped HERE, at construction, so a passage line IS a buffer line and IS a
// Region line. The alternative — keeping logical lines and projecting regions
// through a reflow — needs a second coordinate space and a mapping to keep in
// step, which is the kind of thing that ends with a click marking the word above
// the one you pointed at.
//
// It does not reflow on resize, which is the same bargain the rest of the buffer
// makes: a record keeps the shape it was written in (`screen.lines` is immutable),
// and a definition wrapped at 80 stays wrapped at 80.
//
// The text is expected to have been through sanitisePasteBody already — the wire
// boundary strips escapes — but nothing here DEPENDS on that: the column table
// is built with the same escape-aware walk the rest of the program uses, so a
// passage that somehow carried an escape still maps clicks to the right word
// rather than silently to the wrong one.
func newPassage(text string, width int) *passage {
	p := &passage{src: text, lines: wrapPassageLines(text, width)}
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

// markOn is how a MARK is painted: an explicit foreground/background pair.
//
// Not the selection's inverse video (\x1b[7m), for two reasons. Inverse swaps
// foreground and background, so a deck-green word inside it comes out
// green-BACKGROUND; and a mark persists while a drag does not, so the two must be
// distinguishable at a glance. An explicit pair also composes better with the row
// tint — sourceBackground recognises 48 and not 7, so paintLanguageRow correctly
// declines to inject the row background underneath a mark.
const markOn = "\x1b[48;5;24m\x1b[38;5;231m"

// paintMarks splices the mark attribute over the given display-cell ranges of an
// already-styled line.
//
// It works on FINISHED bytes, the way markClickable does, because the buffer's
// text is immutable once written and a mark is transient — it lives between
// marking a word and asking about it. Paint time is where changing decoration
// belongs.
//
// The attribute is RE-ASSERTED after every producer SGR inside a range, which is
// the discipline that lets it survive colours the renderer already baked in:
// ANSI does not nest, so a highlight that trusted nesting would be cancelled by
// the first reset the palette emits inside it.
func paintMarks(line string, ranges []cellRange) string {
	if len(ranges) == 0 {
		return line
	}
	var b strings.Builder
	var style sgrState
	col, next, on := 0, 0, false
	for i := 0; i < len(line); {
		if on && next < len(ranges) && col >= ranges[next].end {
			b.WriteString(sgrOff + style.resume())
			on, next = false, next+1
		}
		if skip := escapeLen(line[i:]); skip > 0 {
			seq := line[i : i+skip]
			style.observe(seq)
			b.WriteString(seq)
			if on && isSGR(seq) {
				b.WriteString(markOn) // re-assert: ANSI does not nest
			}
			i += skip
			continue
		}
		if !on && next < len(ranges) && col >= ranges[next].start && col < ranges[next].end {
			b.WriteString(markOn)
			on = true
		}
		size, w := nextDisplayUnit(line[i:])
		if size == 0 {
			break
		}
		b.WriteString(line[i : i+size])
		i += size
		col += w
	}
	if on {
		b.WriteString(sgrOff + style.resume())
	}
	return b.String()
}

// passageRegions is one clickable region per word of the passage, addressed the
// way Render addresses its own: Line counts newlines within THIS text, and the
// screen adds its own base when it stores them.
//
// Region.Line therefore IS the passage line, which is what lets a click come
// back to a passageSpan without a second table to keep in step.
func passageRegions(p *passage) []Region {
	if p == nil {
		return nil
	}
	var out []Region
	for i := range p.lines {
		line := p.line(i)
		for _, sp := range p.spans(i) {
			col, width, ok := spanCells(line, sp)
			if !ok {
				continue
			}
			out = append(out, Region{
				Kind:  RegionPassageWord,
				Text:  p.text(sp),
				Word:  p.text(sp),
				Line:  i,
				Col:   col,
				Width: width,
			})
		}
	}
	return out
}

// spanCells is the INVERSE of wordAtCell: a byte span to its display column and
// width. Both live here so the two directions cannot disagree about which cells
// a word occupies.
func spanCells(line string, sp passageSpan) (col, width int, ok bool) {
	if sp.start < 0 || sp.end > len(line) || sp.start >= sp.end {
		return 0, 0, false
	}
	for i := 0; i < len(line); {
		size, w := nextDisplayUnit(line[i:])
		if size == 0 {
			break
		}
		switch {
		case i < sp.start:
			col += w
		case i < sp.end:
			width += w
		}
		i += size
	}
	return col, width, width > 0
}

// markCellRanges is the marks in the SCREEN's coordinates: buffer line to the
// display-cell ranges to paint.
//
// Derived on every draw from the session's marks, rather than kept in step by
// hand — the session owns which spans are marked and this is the one place that
// answer is translated, so the two cannot drift.
func markCellRanges(p *passage, m markSet, base int) map[int][]cellRange {
	if p == nil || m.empty() {
		return nil
	}
	out := map[int][]cellRange{}
	for _, sp := range m.ordered() {
		col, width, ok := spanCells(p.line(sp.line), sp)
		if !ok {
			continue
		}
		at := base + sp.line
		out[at] = append(out[at], cellRange{start: col, end: col + width})
	}
	return out
}

// passageText is the passage as it goes into the buffer: its own lines, joined
// with the line ending the raw screen uses, with the deck words coloured.
//
// Coloured AT WRITE TIME, which freezes today's deck into the record — and that
// is right, because it is what a definition already does. A passage is a record
// of something you read, and the deck it was read against is part of that. What
// changes afterwards is the MARK, which is paint-time for exactly that reason.
//
// highlightRegion is the shared entry seven other callers use; open-coding the
// span loop would be a fourth copy of the ANSI re-open rule.
func passageText(p *passage, v Vocabulary, colour bool) string {
	if p == nil {
		return ""
	}
	lines := make([]string, 0, len(p.lines))
	for _, line := range p.lines {
		if colour {
			line = highlightRegion(line, v, knownOn, "")
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\r\n")
}
