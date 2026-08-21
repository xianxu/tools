package main

import "unicode/utf8"

// KeyKind is the vocabulary of keypresses this editor understands.
type KeyKind int

const (
	KeyUnknown KeyKind = iota // an escape sequence we do not model — inert, never inserted
	KeyRune
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyBackspace
	KeyDelete
	KeyEnter
	KeyTab
	KeyInterrupt // Ctrl-C: a BYTE in raw mode, not a signal
	KeyEOF       // Ctrl-D
	KeyKillLine  // Ctrl-U — and Cmd+Delete, which terminals send as \x15
)

// Key is one decoded keypress. Raw carries the bytes of an unmodelled sequence
// so it can be ignored rather than inserted as garbage — the failure mode of a
// decoder that falls through to "it must be text".
type Key struct {
	Kind KeyKind
	Rune rune
	Raw  []byte
}

// decodeKey converts the front of buf into a Key.
//
// consumed == 0 means buf holds a PREFIX of a longer sequence and the caller
// must read more before deciding. Without that signal a lone ESC arriving in its
// own read decodes as Escape-then-junk, and arrow keys break exactly when the
// terminal is slow.
func decodeKey(buf []byte) (Key, int) {
	if len(buf) == 0 {
		return Key{}, 0
	}
	switch b := buf[0]; b {
	case 0x03:
		return Key{Kind: KeyInterrupt}, 1
	case 0x04:
		return Key{Kind: KeyEOF}, 1
	case 0x15:
		// Ctrl-U. Ghostty binds super+backspace to this by default
		// (`keybind = super+backspace=text:\x15`), and Terminal.app and iTerm2
		// map Cmd+Delete the same way — so handling the byte covers the gesture
		// without guessing at a key code no terminal actually sends.
		return Key{Kind: KeyKillLine}, 1
	case '\r', '\n':
		return Key{Kind: KeyEnter}, 1
	case '\t':
		return Key{Kind: KeyTab}, 1
	case 0x7f, '\b':
		return Key{Kind: KeyBackspace}, 1
	case 0x1b:
		return decodeEscape(buf)
	}
	if buf[0] < 0x20 {
		return Key{Kind: KeyUnknown, Raw: buf[:1]}, 1 // other control bytes are inert
	}
	r, n := utf8.DecodeRune(buf)
	if r == utf8.RuneError && n <= 1 {
		if !utf8.FullRune(buf) {
			return Key{}, 0 // a multi-byte rune split across reads
		}
		return Key{Kind: KeyUnknown, Raw: buf[:1]}, 1
	}
	return Key{Kind: KeyRune, Rune: r}, n
}

func decodeEscape(buf []byte) (Key, int) {
	if len(buf) < 2 {
		return Key{}, 0
	}
	switch buf[1] {
	case '[', 'O':
		if len(buf) < 3 {
			return Key{}, 0
		}
		switch buf[2] {
		case 'A':
			return Key{Kind: KeyUp}, 3
		case 'B':
			return Key{Kind: KeyDown}, 3
		case 'C':
			return Key{Kind: KeyRight}, 3
		case 'D':
			return Key{Kind: KeyLeft}, 3
		case 'H':
			return Key{Kind: KeyHome}, 3
		case 'F':
			return Key{Kind: KeyEnd}, 3
		}
		// Every other CSI sequence: find its REAL final byte rather than assuming
		// a length. ESC[3~ is Delete, but ESC[3;5~ is Ctrl-Delete — assuming four
		// bytes consumed "ESC[3;" and left "5~" to be inserted into the word as
		// text. Parameter bytes are 0x30-0x3F, intermediates 0x20-0x2F, and the
		// final byte is 0x40-0x7E.
		for i := 2; i < len(buf); i++ {
			c := buf[i]
			if c >= 0x30 && c <= 0x3F || c >= 0x20 && c <= 0x2F {
				continue // parameter or intermediate byte
			}
			if c >= 0x40 && c <= 0x7E {
				seq := buf[:i+1]
				if string(seq) == "\x1b[3~" {
					return Key{Kind: KeyDelete}, i + 1
				}
				return Key{Kind: KeyUnknown, Raw: seq}, i + 1
			}
			return Key{Kind: KeyUnknown, Raw: buf[:i+1]}, i + 1 // malformed
		}
		return Key{}, 0 // still incomplete
	}
	return Key{Kind: KeyUnknown, Raw: buf[:2]}, 2
}
