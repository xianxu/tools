package main

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// answerWrapWriter holds the unfinished word, not the paragraph. Highlighting
// feeds it logical text; completed words reach the screen at display-cell wrap
// points. Flush releases the last word on every way a model request can end.
// A downstream failure poisons it, just as it poisons highlightWriter.
type answerWrapWriter struct {
	out        io.Writer
	width, col int
	word, gap  strings.Builder
	tail       []byte // only an incomplete rune or escape; rescans stay bounded
	err        error
}

const (
	maxAnswerWrapPending = 64 << 10
	maxAnswerWrapEscape  = 256
)

func newAnswerWrapWriter(out io.Writer, width int) *answerWrapWriter {
	if width < minWrapWidth {
		width = 0
	}
	return &answerWrapWriter{out: out, width: width}
}

func (w *answerWrapWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.width == 0 {
		w.emit(string(p))
	} else {
		for _, b := range p {
			w.tail = append(w.tail, b)
			w.consume()
			if w.err != nil {
				break
			}
			if w.word.Len()+w.gap.Len()+len(w.tail) > maxAnswerWrapPending {
				w.err = fmt.Errorf("answer wrapping: unfinished text exceeds %d bytes", maxAnswerWrapPending)
				break
			}
		}
	}
	if w.err != nil {
		return 0, w.err
	}
	return len(p), nil
}

// consume visits each complete rune once. Only an incomplete escape can need
// repeated scans; its small independent cap bounds that work even for a peer
// delivering one byte per call. scanEscape remains the owner of its grammar.
func (w *answerWrapWriter) consume() {
	for len(w.tail) > 0 && w.err == nil {
		if w.tail[0] == '\x1b' {
			n := scanEscape(string(w.tail))
			if n < 0 {
				if len(w.tail) > maxAnswerWrapEscape {
					w.err = fmt.Errorf("answer wrapping: unfinished escape exceeds %d bytes", maxAnswerWrapEscape)
				}
				return
			}
			w.word.Write(w.tail[:n])
			w.tail = w.tail[n:]
			continue
		}
		if !utf8.FullRune(w.tail) {
			return
		}
		r, n := utf8.DecodeRune(w.tail)
		s := string(w.tail[:n])
		w.tail = w.tail[n:]
		if !unicode.IsSpace(r) {
			w.word.WriteString(s)
			continue
		}
		w.emitWord()
		if r == '\n' {
			w.emit(w.gap.String() + s)
			w.gap.Reset()
			w.col = 0
		} else {
			if r == '\t' {
				s = " " // tabs have terminal-dependent stops; prose uses one space
			}
			w.gap.WriteString(s)
		}
	}
}

func (w *answerWrapWriter) emitWord() {
	if w.word.Len() == 0 || w.err != nil {
		return
	}
	word, gap := w.word.String(), w.gap.String()
	cells := visibleCells(word)
	if cells > 0 && w.col > 0 && w.col+visibleCells(gap)+cells > w.width {
		w.emit("\n")
		w.col, gap = 0, ""
	}
	w.emit(gap + word)
	w.col += visibleCells(gap) + cells
	w.word.Reset()
	w.gap.Reset()
}

func (w *answerWrapWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	// Preserve a malformed final tail verbatim, as highlightWriter does.
	w.word.Write(w.tail)
	w.tail = nil
	w.emitWord()
	w.emit(w.gap.String())
	w.gap.Reset()
	return w.err
}

func (w *answerWrapWriter) emit(s string) {
	if s == "" || w.err != nil {
		return
	}
	n, err := io.WriteString(w.out, s)
	if err == nil && n < len(s) {
		err = io.ErrShortWrite
	}
	w.err = err
}
