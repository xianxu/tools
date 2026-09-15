package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// renderedOutput keeps terminal paint separate from unpadded, selectable text.
// Row backgrounds are resolved when produced, so later policy changes do not
// recolor history. Coordinates in exclusions are half-open display columns.
type renderedOutput struct {
	text    string
	regions []Region
	rows    []rowPaint
}
type rowPaint struct {
	background string
	exclusions []cellRange
}
type cellRange struct{ start, end int }

func validRowPaint(p rowPaint) bool {
	if p.background != "" && p.background != languageDark && p.background != languageLight {
		return false
	}
	if len(p.exclusions) > maxSelectionCells {
		return false
	}
	end := 0
	for _, r := range p.exclusions {
		if r.start < end || r.end <= r.start || r.end > maxSelectionCells {
			return false
		}
		end = r.end
	}
	return true
}

// layoutOutput delegates line breaking to the existing geometry owner. A
// display-unit correspondence projects both click ranges and answer exclusions;
// generated hanging indentation has no source cell.
func layoutOutput(o renderedOutput, width int) renderedOutput {
	out := renderedOutput{text: wrapWritten(o.text, width)}
	if len(o.text) > maxSelectionSource || visibleCells(o.text) > maxSelectionCells {
		out.regions = wrapMovedRegions(o.text, o.regions, width)
		return out
	}
	source := strings.Split(o.text, "\n")
	valid := len(o.text) <= maxSelectionSource && len(o.rows) <= len(source)
	if valid {
		for _, p := range o.rows {
			if !validRowPaint(p) {
				valid = false
				break
			}
		}
	}
	lineBase := 0
	for line, s := range source {
		physical := strings.Split(wrapWritten(s, width), "\n")
		origins, ok := outputCellOrigins(s, physical)
		excluded := 0
		for row := range physical {
			p := rowPaint{}
			if valid && ok && line < len(o.rows) {
				p.background = o.rows[line].background
				for col, origin := range origins[row] {
					if origin < 0 {
						continue
					}
					exclusions := o.rows[line].exclusions
					for excluded < len(exclusions) && exclusions[excluded].end <= origin {
						excluded++
					}
					if excluded < len(exclusions) && exclusions[excluded].start <= origin {
						p.exclusions = appendCellRange(p.exclusions, col)
					}
				}
			}
			out.rows = append(out.rows, p)
		}
		for _, r := range o.regions {
			if r.Line != line || r.Col < 0 || r.Width <= 0 || r.Col > maxSelectionCells-r.Width || !ok {
				continue
			}
			for row, cols := range origins {
				var runs []cellRange
				for col, origin := range cols {
					if origin >= r.Col && origin < r.Col+r.Width {
						runs = appendCellRange(runs, col)
					}
				}
				for _, run := range runs {
					moved := r
					moved.Line = lineBase + row
					moved.Col = run.start
					moved.Width = run.end - run.start
					out.regions = append(out.regions, moved)
				}
			}
		}
		lineBase += len(physical)
	}
	return out
}

func appendCellRange(rs []cellRange, col int) []cellRange {
	if len(rs) > 0 && rs[len(rs)-1].end == col {
		rs[len(rs)-1].end++
		return rs
	}
	return append(rs, cellRange{col, col + 1})
}

type outputUnit struct {
	text       string
	col, width int
	space      bool
}

func outputUnits(s string) []outputUnit {
	var units []outputUnit
	col := 0
	for i := 0; i < len(s); {
		if n := escapeLen(s[i:]); n > 0 {
			i += n
			continue
		}
		n, w := nextDisplayUnit(s[i:])
		r, _ := utf8.DecodeRuneInString(s[i:])
		units = append(units, outputUnit{s[i : i+n], col, w, unicode.IsSpace(r)})
		col += w
		i += n
	}
	return units
}

func outputCellOrigins(source string, rows []string) ([][]int, bool) {
	units := outputUnits(source)
	origins := make([][]int, len(rows))
	at := 0
	for row, text := range rows {
		started := row == 0
		for _, u := range outputUnits(text) {
			origin := -1
			if u.space {
				// Continuation indentation was inserted by wrapWritten, not the source.
				if started && at < len(units) && units[at].space {
					origin = units[at].col
					at++
				}
			} else {
				for at < len(units) && units[at].space {
					at++
				}
				if at >= len(units) || units[at].text != u.text {
					return origins, false
				}
				origin = units[at].col
				at++
				started = true
			}
			for j := 0; j < u.width; j++ {
				v := -1
				if origin >= 0 {
					v = origin + j
				}
				origins[row] = append(origins[row], v)
			}
		}
	}
	for at < len(units) && units[at].space {
		at++
	}
	return origins, at == len(units)
}

// outputStyledRows makes each source row independently paintable without
// adding selectable cells. A terminal newline preserves producer SGR state;
// the row painter resets it after padding, so subsequent rows need a replay.
func outputStyledRows(text string) []string {
	lines := strings.Split(text, "\n")
	var style sgrState
	for i, line := range lines {
		if i == len(lines)-1 && line == "" && strings.HasSuffix(text, "\n") {
			continue
		}
		prefix := style.resume()
		for at := 0; at < len(line); {
			if n := escapeLen(line[at:]); n > 0 {
				style.observe(line[at : at+n])
				at += n
			} else {
				n, _ := nextDisplayUnit(line[at:])
				at += n
			}
		}
		lines[i] = prefix + line
	}
	return lines
}

// serializeOutput paints only at the terminal boundary. Width zero is the
// clean pipe/history path and never introduces synthetic spaces.
func serializeOutput(o renderedOutput, width int) string {
	if width <= 0 {
		return o.text
	}
	o = layoutOutput(o, width)
	lines := outputStyledRows(o.text)
	for i, line := range lines {
		// A trailing newline is a terminator, not a newly emitted empty row.
		if i == len(lines)-1 && line == "" && strings.HasSuffix(o.text, "\n") {
			continue
		}
		paint := rowPaint{}
		if i < len(o.rows) {
			paint = o.rows[i]
		}
		lines[i] = paintLanguageRow(line, paint, width)
	}
	return strings.Join(lines, "\n")
}
