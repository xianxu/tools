package main

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

const maxSelectionCells = 262144
const maxSelectionSource = 1 << 20

var errSelectionBounds = errors.New("selection exceeds display limit")

type selectionRow struct {
	styled                    string
	selectable                bool
	regions                   []Region
	footerEntry, footerOffset int
	footer, retry             bool
}

// A cell repeats the same whole glyph for both columns of a wide character.
// start/end address the original styled source, preserving zero-width marks.
type selectionCell struct {
	text       string
	start, end int
}
type selectionFrame struct {
	width, height int
	rows          []selectionRow
	cells         [][]selectionCell
	err           error
}

func newSelectionFrame(width, height int, rows []selectionRow) selectionFrame {
	f := selectionFrame{width: width, height: height}
	if width <= 0 || height <= 0 || height > maxSelectionCells || width > maxSelectionCells/height || len(rows) > height {
		f.err = errSelectionBounds
		return f
	}
	n := 0
	for _, row := range rows {
		if len(row.styled) > maxSelectionSource-n {
			f.err = errSelectionBounds
			return f
		}
		n += len(row.styled)
		if len(row.regions) > (maxSelectionSource-n)/64 {
			f.err = errSelectionBounds
			return f
		}
		n += len(row.regions) * 64
		for _, r := range row.regions {
			for _, s := range []string{r.Text, r.Word, string(r.Lang)} {
				if len(s) > maxSelectionSource-n {
					f.err = errSelectionBounds
					return f
				}
				n += len(s)
			}
		}
	}
	f.rows = make([]selectionRow, len(rows))
	f.cells = make([][]selectionCell, len(rows))
	for i, row := range rows {
		row.styled = strings.Clone(row.styled)
		row.regions = slices.Clone(row.regions)
		for j := range row.regions {
			row.regions[j].Text = strings.Clone(row.regions[j].Text)
			row.regions[j].Word = strings.Clone(row.regions[j].Word)
		}
		f.rows[i] = row
		if row.selectable {
			f.cells[i], f.err = selectionCells(row.styled, width)
			if f.err != nil {
				return f
			}
		}
	}
	return f
}

func selectionCells(styled string, width int) ([]selectionCell, error) {
	if width < 0 || width > maxSelectionCells || len(styled) > maxSelectionSource {
		return nil, errSelectionBounds
	}
	cells := make([]selectionCell, 0, min(width, len(styled)))
	col := 0
	last := -1
	var glyph strings.Builder
	flush := func() {
		if last >= 0 {
			text := glyph.String()
			for j := last; j < len(cells); j++ {
				cells[j].text = text
			}
		}
		glyph.Reset()
	}
	for i := 0; i < len(styled); {
		if skip := escapeLen(styled[i:]); skip > 0 {
			i += skip
			continue
		}
		r, n := utf8.DecodeRuneInString(styled[i:])
		w := cellWidth(r)
		if w == 0 {
			if last >= 0 {
				glyph.WriteString(styled[i : i+n])
				for j := last; j < len(cells); j++ {
					cells[j].end = i + n
				}
			}
			i += n
			continue
		}
		flush()
		if w > width-col {
			last = -1
			break
		}
		last = len(cells)
		glyph.WriteString(styled[i : i+n])
		cell := selectionCell{start: i, end: i + n}
		for j := 0; j < w; j++ {
			cells = append(cells, cell)
		}
		col += w
		i += n
	}
	flush()
	return cells, nil
}

func selectionOrdered(a, b selectionPoint) (selectionPoint, selectionPoint) {
	if a.row > b.row || a.row == b.row && a.col > b.col {
		return b, a
	}
	return a, b
}
func selectedText(f selectionFrame, a, b selectionPoint) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	a, b = selectionOrdered(a, b)
	var out strings.Builder
	included := false
	for row := max(0, a.row); row < len(f.rows) && row <= b.row; row++ {
		if !f.rows[row].selectable {
			continue
		}
		lo, hi := 0, len(f.cells[row])-1
		if row == a.row {
			lo = max(lo, a.col)
		}
		if row == b.row {
			hi = min(hi, b.col)
		}
		if included {
			out.WriteByte('\n')
		}
		included = true
		last := -1
		for c := lo; c <= hi; c++ {
			cell := f.cells[row][c]
			if cell.start == last {
				continue
			}
			last = cell.start
			if len(cell.text) > maxSelectionSource-out.Len() {
				return "", errSelectionBounds
			}
			out.WriteString(cell.text)
		}
		if out.Len() > maxSelectionSource {
			return "", errSelectionBounds
		}
	}
	return out.String(), nil
}

func (f selectionFrame) same(other selectionFrame) bool {
	if f.width != other.width || f.height != other.height || len(f.rows) != len(other.rows) || (f.err != nil) != (other.err != nil) {
		return false
	}
	for i, r := range f.rows {
		s := other.rows[i]
		if r.selectable != s.selectable || r.footer != s.footer || r.retry != s.retry || r.footerEntry != s.footerEntry || r.footerOffset != s.footerOffset || !slices.Equal(r.regions, s.regions) {
			return false
		}
		if !r.selectable {
			continue
		}
		if len(f.cells[i]) != len(other.cells[i]) {
			return false
		}
		for j, c := range f.cells[i] {
			if c.text != other.cells[i][j].text {
				return false
			}
		}
	}
	return true
}

func (f selectionFrame) highlightRow(row int, a, b selectionPoint) string {
	if row < 0 || row >= len(f.rows) {
		return ""
	}
	text := f.rows[row].styled
	if f.err != nil || !f.rows[row].selectable {
		return text
	}
	a, b = selectionOrdered(a, b)
	if row < a.row || row > b.row {
		return text
	}
	lo, hi := 0, len(f.cells[row])-1
	if row == a.row {
		lo = max(lo, a.col)
	}
	if row == b.row {
		hi = min(hi, b.col)
	}
	if lo > hi {
		return text
	}
	start, end := f.cells[row][lo].start, f.cells[row][hi].end
	var out strings.Builder
	var style sgrState
	on := false
	for i := 0; i < len(text); {
		if on && i >= end {
			out.WriteString("\x1b[0m" + style.resume())
			on = false
		}
		if skip := escapeLen(text[i:]); skip > 0 {
			seq := text[i : i+skip]
			style.observe(seq)
			out.WriteString(seq)
			if on && isSGR(seq) {
				out.WriteString("\x1b[7m")
			}
			i += skip
			continue
		}
		if !on && i == start {
			out.WriteString("\x1b[7m")
			on = true
		}
		_, n := utf8.DecodeRuneInString(text[i:])
		out.WriteString(text[i : i+n])
		i += n
	}
	if on {
		out.WriteString("\x1b[0m" + style.resume())
	}
	return out.String()
}
