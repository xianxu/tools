package main

import "strings"

// paintLanguageRow composes a row background with the producer's styles.
// Explicit backgrounds and excluded answer cells win; selection inverse and
// foreground styles pass through. Padding exists only in this terminal string.
func paintLanguageRow(text string, paint rowPaint, width int) string {
	if width <= 0 {
		return text
	}
	if width > maxSelectionCells || len(text) > maxSelectionSource || !validRowPaint(paint) {
		paint = rowPaint{}
	}
	if paint.background == "" {
		return text
	}
	var out strings.Builder
	explicit, filled := false, false
	col, exclude := 0, 0
	fill := func(w int) {
		for exclude < len(paint.exclusions) && paint.exclusions[exclude].end <= col {
			exclude++
		}
		excluded := exclude < len(paint.exclusions) && paint.exclusions[exclude].start < col+w
		wanted := !explicit && !excluded
		if wanted && !filled {
			out.WriteString(paint.background)
			filled = true
		}
		if !wanted && filled {
			out.WriteString(languageOff)
			filled = false
		}
	}
	for i := 0; i < len(text); {
		if n := escapeLen(text[i:]); n > 0 {
			seq := text[i : i+n]
			if !isSGR(seq) {
				out.WriteString("\x1b[0m")
				filled, explicit = false, false
			}
			// Remove only our injected background before a producer style. This
			// keeps producer resets and explicit answer backgrounds authoritative.
			if filled {
				out.WriteString(languageOff)
				filled = false
			}
			out.WriteString(seq)
			explicit = sourceBackground(seq, explicit)
			i += n
			continue
		}
		n, w := nextDisplayUnit(text[i:])
		if text[i] == '\n' || text[i] == '\r' || col+w > width {
			break
		}
		fill(w)
		out.WriteString(text[i : i+n])
		col += w
		i += n
	}
	// Synthetic spaces must not inherit explicit answer backgrounds or inverse.
	out.WriteString("\x1b[0m")
	explicit = false
	filled = false
	for col < width {
		fill(1)
		out.WriteByte(' ')
		col++
	}
	out.WriteString("\x1b[0m")
	return out.String()
}
