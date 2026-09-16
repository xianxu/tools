package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"io"
	"slices"
	"strings"
)

type outputWriter interface{ WriteOutput(renderedOutput) error }

func writeOutput(w io.Writer, o renderedOutput, width int) error {
	if sink, ok := w.(outputWriter); ok {
		return sink.WriteOutput(o)
	}
	if sink, ok := w.(regionWriter); ok {
		o = layoutOutput(o, width)
		sink.WriteRegions(serializeOutput(o, width), o.regions)
		return nil
	}
	text := serializeOutput(o, width)
	n, err := io.WriteString(w, text)
	if err == nil && n < len(text) {
		return io.ErrShortWrite
	}
	return err
}

func paintAt(rows []rowPaint, at int) rowPaint {
	if at >= 0 && at < len(rows) {
		return rows[at]
	}
	return rowPaint{}
}

func (l *liveScreen) OutputWidth() int { l.mu.Lock(); defer l.mu.Unlock(); return l.cols }

func (l *liveScreen) WriteOutput(o renderedOutput) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.s.pinned {
		o = layoutOutput(o, l.cols)
	}
	o.text = strings.Join(outputStyledRows(o.text), "\n")
	l.invalidateSelectionLocked()
	base := len(l.s.lines)
	if l.s.partial && base > 0 {
		base--
	}
	l.s.addRegions(o.regions)
	// Metadata is attached under the same lock as its unpadded source. A partial
	// plain prefix cannot inherit ownership from a later structured fragment.
	partial := l.s.partial
	l.s.invalidatePartialPaint(o.text)
	_, err := l.s.Write([]byte(o.text))
	if l.s.paints == nil {
		l.s.paints = make(map[int]rowPaint)
	}
	for i, p := range o.rows {
		if base+i >= len(l.s.lines) {
			break
		}
		if i == 0 && partial {
			continue
		}
		p.exclusions = slices.Clone(p.exclusions)
		l.s.paints[base+i] = p
	}
	l.throttledPaint()
	return err
}

func (s *screen) paintedTranscript(width int) string {
	var b strings.Builder
	for i, line := range outputStyledRows(strings.Join(s.lines, "\n")) {
		if len(s.lines) == 0 {
			break
		}
		b.WriteString(paintLanguageRow(clipVisible(line, width), s.paints[i], width))
		b.WriteByte('\n')
	}
	return b.String()
}

// DrawOutput keeps live-edge metadata outside the transcript and source text.
func (l *liveScreen) DrawOutput(prompt, footer renderedOutput) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prompt = prompt.text
	l.footer = strings.Split(footer.text, "\n")
	if footer.text == "" {
		l.footer = nil
	}
	l.s.promptPaint = cloneRowPaints(prompt.rows)
	l.s.footerPaint = cloneRowPaints(footer.rows)
	l.repaint()
}

func normalizedLang(lang store.Lang) store.Lang {
	if lang == "" {
		lang = store.DefaultLang
	}
	v, err := store.ParseLang(string(lang))
	if err != nil {
		return ""
	}
	return v
}

func paintOutputChunk(text string, p rowPaint, width int) string {
	if p.background == "" {
		return text
	}
	rows := selectionPhysicalRows(text, width)
	if rows == nil {
		return text
	}
	var b strings.Builder
	start := 0
	for _, row := range rows {
		b.WriteString(paintLanguageRow(row, sliceRowPaint(p, start, visibleCells(row)), width))
		start += visibleCells(row)
	}
	return b.String()
}
func sliceRowPaint(p rowPaint, start, width int) rowPaint {
	out := rowPaint{background: p.background}
	for _, r := range p.exclusions {
		a, b := max(r.start, start), min(r.end, start+width)
		if a < b {
			out.exclusions = append(out.exclusions, cellRange{a - start, b - start})
		}
	}
	return out
}

func cloneRowPaints(rows []rowPaint) []rowPaint {
	out := slices.Clone(rows)
	for i := range out {
		out[i].exclusions = slices.Clone(out[i].exclusions)
	}
	return out
}

// Compatibility string renderers have no viewport. Paint their source width;
// production writers retain metadata until their actual terminal boundary.
func renderOutputText(o renderedOutput) string {
	lines := outputStyledRows(o.text)
	for i, line := range lines {
		lines[i] = paintLanguageRow(line, paintAt(o.rows, i), visibleCells(line))
	}
	return strings.Join(lines, "\n")
}

func (l *liveScreen) PaintedTranscript() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s.paintedTranscript(l.cols)
}
func (l *liveScreen) OutputTranscript() renderedOutput {
	l.mu.Lock()
	defer l.mu.Unlock()
	o := renderedOutput{text: l.s.Transcript(), rows: make([]rowPaint, len(l.s.lines))}
	for i := range o.rows {
		o.rows[i] = l.s.paints[i]
	}
	o.rows = cloneRowPaints(o.rows)
	for i, regions := range l.s.regions {
		for _, r := range regions {
			r.Line = i
			o.regions = append(o.regions, r)
		}
	}
	return o
}

// A terminator commits an existing row without adding unknown source prose.
// Appending any other source bytes invalidates ownership of that partial row;
// later rows start independently. Plain and structured writers share this rule.
func (s *screen) invalidatePartialPaint(text string) {
	if !s.partial {
		return
	}
	prefix, _, _ := strings.Cut(text, "\n")
	if strings.Trim(prefix, "\r") != "" {
		delete(s.paints, len(s.lines)-1)
	}
}
