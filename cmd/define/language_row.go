package main

import (
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// paintLanguageRow composes a row background with the producer's styles.
// Explicit backgrounds and excluded answer cells win; selection inverse and
// foreground styles pass through. Padding exists only in this terminal string.
// A tinted row takes the shade of the scheme it is painted in (#70).
func paintLanguageRow(text string, paint rowPaint, width int, sc store.Scheme) string {
	if width <= 0 {
		return text
	}
	if width > maxSelectionCells || len(text) > maxSelectionSource || !validRowPaint(paint) {
		paint = rowPaint{}
	}
	if !paint.tinted {
		return text
	}
	background, ink := schemeTint(sc), schemeInk(sc)
	var out strings.Builder
	// explicit: the producer set a background; coloured: it set a foreground.
	// Our paired ink is on exactly when filled && !coloured — derived, not
	// stored: every change to coloured is preceded by unfill().
	explicit, coloured, filled := false, false, false
	col, exclude := 0, 0
	unfill := func() {
		if !filled {
			return
		}
		out.WriteString(languageOff)
		if !coloured {
			out.WriteString(inkOff)
		}
		filled = false
	}
	fill := func(w int) {
		for exclude < len(paint.exclusions) && paint.exclusions[exclude].end <= col {
			exclude++
		}
		excluded := exclude < len(paint.exclusions) && paint.exclusions[exclude].start < col+w
		wanted := !explicit && !excluded
		if wanted && !filled {
			out.WriteString(background)
			// Text with no colour of its own takes the tint's paired ink (#70).
			if !coloured {
				out.WriteString(ink)
			}
			filled = true
		}
		if !wanted {
			unfill()
		}
	}
	for i := 0; i < len(text); {
		if n := escapeLen(text[i:]); n > 0 {
			seq := text[i : i+n]
			if !isSGR(seq) {
				out.WriteString("\x1b[0m")
				filled, explicit, coloured = false, false, false
			}
			// Remove only our injected paint before a producer style. This keeps
			// producer resets, explicit answer backgrounds and producer colours
			// authoritative.
			unfill()
			out.WriteString(seq)
			explicit, coloured = sourceColours(seq, explicit, coloured)
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
	explicit, coloured, filled = false, false, false
	for col < width {
		fill(1)
		out.WriteByte(' ')
		col++
	}
	out.WriteString("\x1b[0m")
	return out.String()
}
