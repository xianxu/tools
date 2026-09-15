package main

import (
	"unicode"
	"unicode/utf8"
)

// answerTextFilter discards terminal controls before either storage or display.
// Invalid UTF-8 is replaced one invalid byte at a time; only a partial rune is
// retained. Control strings retain state only, never their untrusted payload.
type answerTextFilter struct {
	emit    func(string)
	state   uint8
	osc     bool
	pending string
}

const (
	textPlain uint8 = iota
	textEscape
	textCSI
	textString
	textStringEscape
)

func (f *answerTextFilter) Write(s string) {
	s = f.pending + s
	f.pending = ""
	for len(s) > 0 {
		if !utf8.FullRuneInString(s) {
			f.pending = s
			return
		}
		r, n := utf8.DecodeRuneInString(s)
		// Raw C1 bytes and UTF-8 encoded C1 controls have the same semantics.
		if r == utf8.RuneError && n == 1 && s[0] >= 0x80 && s[0] <= 0x9f {
			r = rune(s[0])
		}
		s = s[n:]
		f.rune(r)
	}
}
func (f *answerTextFilter) Finish() {
	for range len(f.pending) {
		f.rune(utf8.RuneError)
	}
	f.pending = ""
	f.state = textPlain
}
func (f *answerTextFilter) rune(r rune) {
	if f.state == textString || f.state == textStringEscape {
		if r == 0x9c || f.osc && r == 7 || f.state == textStringEscape && r == '\\' {
			f.state = textPlain
			return
		}
		if r == 27 {
			f.state = textStringEscape
		} else {
			f.state = textString
		}
		return
	}
	if f.state == textEscape {
		f.state = textPlain
		switch r {
		case '[':
			f.state = textCSI
			return
		case ']':
			f.state = textString
			f.osc = true
			return
		case 'P', 'X', '^', '_':
			f.state = textString
			f.osc = false
			return
		}
		if r >= 0x20 && r <= 0x2f {
			f.state = textEscape
			return
		}
		if r >= 0x30 && r <= 0x7e {
			return
		}
	}
	if f.state == textCSI {
		if r >= 0x40 && r <= 0x7e {
			f.state = textPlain
			return
		}
		if r >= 0x20 && r <= 0x3f {
			return
		}
		f.state = textPlain // malformed sequence: reprocess this character as prose
	}
	switch r {
	case 27:
		f.state = textEscape
	case 0x9b:
		f.state = textCSI
	case 0x9d:
		f.state = textString
		f.osc = true
	case 0x90, 0x98, 0x9e, 0x9f:
		f.state = textString
		f.osc = false
	default:
		if r == '\n' || r == '\t' || unicode.IsPrint(r) {
			f.emit(string(r))
		}
	}
}
