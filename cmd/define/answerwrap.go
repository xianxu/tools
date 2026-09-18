package main

import (
	"fmt"
	"github.com/xianxu/tools/cmd/define/store"
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
	tail       []byte   // only an incomplete rune or escape; rescans stay bounded
	sgr        sgrState // style of emitted text, not of the unfinished word
	err        error
	background bool
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
			w.emit(w.gap.String())
			w.newline()
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
		w.newline()
		w.col, gap = 0, ""
	}
	w.emit(gap + word)
	// A viewport can begin at any physical row. Reopen the emitted style at
	// each boundary, but observe only original escapes: observing our own
	// replay would repeatedly accumulate the same styles in sgrState.
	for rest := word; ; {
		i := strings.IndexByte(rest, '\x1b')
		if i < 0 {
			break
		}
		n := scanEscape(rest[i:])
		if n < 0 {
			break // malformed final tail; preserved, not interpreted
		}
		w.sgr.observe(rest[i : i+n])
		w.background = sourceBackground(rest[i:i+n], w.background)
		rest = rest[i+n:]
	}
	w.col += visibleCells(gap) + cells
	w.word.Reset()
	w.gap.Reset()
}

// Newlines inserted after composition must not carry the language background
// onto terminal padding. Replay the original style for the next physical row.
func (w *answerWrapWriter) newline() {
	if w.background {
		w.emit(languageOff)
	}
	w.emit("\n")
	w.emit(w.sgr.resume())
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

// rowOwnership is independent of terminal styling. Empty decorations never
// establish ownership; unknown or foreign prose absorbs later known prose.
type rowOwnership struct {
	lang  string
	mixed bool
}
type rowOwnershipEvent struct {
	lang        string
	substantive bool
	finalize    bool
}

func advanceRowOwnership(s rowOwnership, e rowOwnershipEvent) rowOwnership {
	if e.finalize {
		return rowOwnership{}
	}
	if !e.substantive || s.mixed {
		return s
	}
	if e.lang == "" || s.lang != "" && s.lang != e.lang {
		return rowOwnership{mixed: true}
	}
	return rowOwnership{lang: e.lang}
}

// isLanguageProse separates unowned list numbers and punctuation from prose.
// Explicit ownership can establish language for any non-whitespace glyph.
func isLanguageProse(r rune, owned bool) bool {
	return unicode.IsLetter(r) || owned && !unicode.IsSpace(r)
}

type answerUnit struct {
	text, lang string
	cells      int
	space      bool
}

// ownedAnswerWrapWriter buffers only the unfinished physical row and word.
// Ownership travels with display units through wrapping, so a later foreign
// word cannot change the immutable rows already sent to either kind of sink.
type ownedAnswerWrapWriter struct {
	out            io.Writer
	width          int
	policy         tintPolicy
	row, word, gap []answerUnit
	col, pending   int
	sgr            sgrState
	err            error
}

func newOwnedAnswerWrapWriter(out io.Writer, width int, p tintPolicy) *ownedAnswerWrapWriter {
	if p.lang == "" {
		p.lang = store.DefaultLang
	}
	if lang, err := store.ParseLang(string(p.lang)); err == nil {
		p.lang = lang
	} else {
		p.on = false
	}
	return &ownedAnswerWrapWriter{out: out, width: width, policy: p}
}
func (w *ownedAnswerWrapWriter) WriteOwned(s, lang string) error {
	if w.err != nil {
		return w.err
	}
	if source, ok := w.out.(interface{ OutputWidth() int }); ok && w.width > 0 {
		width := source.OutputWidth()
		if width > 0 && width != w.width {
			w.resize(width)
		}
	}
	if w.width <= 0 {
		n, err := io.WriteString(w.out, s)
		if err == nil && n != len(s) {
			err = io.ErrShortWrite
		}
		w.err = err
		return err
	}
	for len(s) > 0 && w.err == nil {
		n := escapeLen(s)
		cells := 0
		space := false
		if n == 0 {
			n, cells = nextDisplayUnit(s)
			r, _ := utf8.DecodeRuneInString(s)
			space = unicode.IsSpace(r)
		}
		u := answerUnit{text: s[:n], lang: lang, cells: cells, space: space}
		s = s[n:]
		if u.text == "\t" {
			u.text = " "
			u.cells = 1
		}
		w.pending += len(u.text)
		if w.pending > maxAnswerWrapPending {
			w.err = fmt.Errorf("answer wrapping: unfinished text exceeds %d bytes", maxAnswerWrapPending)
			break
		}
		w.acceptUnit(u)
	}
	return w.err
}
func (w *ownedAnswerWrapWriter) acceptUnit(u answerUnit) {
	if !u.space {
		if len(w.word) > 0 && u.cells > 0 {
			last := &w.word[len(w.word)-1]
			if last.cells > 0 && last.lang == u.lang {
				joined := last.text + u.text
				if n, cells := nextDisplayUnit(joined); n == len(joined) {
					last.text, last.cells = joined, cells
					return
				}
			}
		}
		w.word = append(w.word, u)
		return
	}
	w.emitWord()
	if u.text == "\n" {
		w.appendRow(w.gap)
		w.gap = nil
		w.pending -= len(u.text)
		w.finalize(true)
	} else {
		w.gap = append(w.gap, u)
	}
}
func unitCells(units []answerUnit) (n int) {
	for _, u := range units {
		n += u.cells
	}
	return
}
func unitBytes(units []answerUnit) (n int) {
	for _, u := range units {
		n += len(u.text)
	}
	return
}
func (w *ownedAnswerWrapWriter) emitWord() {
	if len(w.word) == 0 || w.err != nil {
		return
	}
	if w.col > 0 && w.col+unitCells(w.gap)+unitCells(w.word) > w.width {
		w.pending -= unitBytes(w.gap)
		w.gap = nil
		w.finalize(true)
	}
	units := append(w.gap, w.word...)
	w.gap = nil
	w.word = nil
	w.appendRow(units)
}
func (w *ownedAnswerWrapWriter) appendRow(units []answerUnit) {
	for _, u := range units {
		if u.cells > 0 && w.col > 0 && w.col+u.cells > w.width {
			w.finalize(true)
		}
		w.row = append(w.row, u)
		w.col += u.cells
	}
}
func (w *ownedAnswerWrapWriter) finalize(newline bool) {
	if w.err != nil {
		return
	}
	var text strings.Builder
	text.WriteString(w.sgr.resume())
	owner := rowOwnership{}
	for _, u := range w.row {
		text.WriteString(u.text)
		if isSGR(u.text) {
			w.sgr.observe(u.text)
		}
		r, _ := utf8.DecodeRuneInString(u.text)
		owner = advanceRowOwnership(owner, rowOwnershipEvent{lang: u.lang, substantive: u.cells > 0 && isLanguageProse(r, u.lang != "")})
	}
	target := string(w.policy.lang)
	tinted := !owner.mixed && owner.lang == target && w.policy.on
	if newline {
		text.WriteByte('\n')
	}
	w.err = writeOutput(w.out, renderedOutput{text: text.String(), rows: []rowPaint{{tinted: tinted}}}, w.width, w.policy.scheme.Scheme())
	w.pending -= unitBytes(w.row)
	w.row = nil
	w.col = 0
}
func (w *ownedAnswerWrapWriter) resize(width int) {
	// Only uncommitted text is replayed. Already emitted rows remain unchanged.
	pending := append(append(append([]answerUnit{}, w.row...), w.gap...), w.word...)
	w.row = nil
	w.gap = nil
	w.word = nil
	w.col = 0
	w.width = width
	for _, u := range pending {
		w.acceptUnit(u)
	}
}
func (w *ownedAnswerWrapWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	if source, ok := w.out.(interface{ OutputWidth() int }); ok && w.width > 0 {
		if width := source.OutputWidth(); width > 0 && width != w.width {
			w.resize(width)
		}
	}
	w.emitWord()
	w.appendRow(w.gap)
	w.gap = nil
	if len(w.row) > 0 {
		w.finalize(false)
	}
	return w.err
}
