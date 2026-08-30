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
	// The VIEWPORT keys, a category of their own: they change what you are
	// looking at rather than the line you are typing, so the editor ignores them
	// and the loop hands them to the screen (#30 M1.4a).
	//
	// PageUp/PageDown and nothing else. Ctrl-U and Ctrl-D are the obvious
	// half-page bindings and both are ALREADY TAKEN above — 0x15 kills the line
	// and 0x04 ends the session on an empty one — so rebinding either would be a
	// silent regression in an editor people already use.
	KeyPageUp
	KeyPageDown
	// The WHEEL, which is a viewport gesture like the page keys and not a mouse
	// feature (#30 M1.4b). In the alternate screen a terminal translates the
	// wheel into ARROW KEYS by default, and this program binds Up/Down to the
	// history walk — so a scroll walked history instead. Nothing can tell the
	// two apart, because they are the same bytes; the only way to get the wheel
	// itself is to ask the terminal to report the mouse, which is why tracking
	// is enabled here rather than waiting for the clicks in M2.
	KeyWheelUp
	KeyWheelDown
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
		case 'M':
			if buf[1] == '[' {
				// The X10 mouse report, and the reason this case exists at all:
				// enabling mode 1000 asks for the mouse, and a terminal that
				// honours 1000 but ignores 1006 answers in X10 — ESC[M plus
				// THREE RAW BYTES that are not part of any CSI grammar. The scan
				// below would stop at "M" as a final byte and hand the payload to
				// the line as text: a left click at (1,1) typed " !!" into the
				// word being looked up, and a wheel notch typed "`!!".
				//
				// This is #14's family, one encoding over. The rule it leaves
				// behind: for every mode we ENABLE, the decoder answers every
				// encoding that mode can reply in.
				return decodeX10Mouse(buf)
			}
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
				// The tilde family, delimited by the scan above rather than by a
				// guessed length. ESC[5~/ESC[6~ were already delimited correctly
				// and then discarded as KeyUnknown; naming them is all M1.4a
				// needed from this decoder.
				switch string(seq) {
				case "\x1b[3~":
					return Key{Kind: KeyDelete}, i + 1
				case "\x1b[5~":
					return Key{Kind: KeyPageUp}, i + 1
				case "\x1b[6~":
					return Key{Kind: KeyPageDown}, i + 1
				}
				if k, ok := decodeWheel(seq); ok {
					return k, i + 1
				}
				return Key{Kind: KeyUnknown, Raw: seq}, i + 1
			}
			return Key{Kind: KeyUnknown, Raw: buf[:i+1]}, i + 1 // malformed
		}
		return Key{}, 0 // still incomplete
	}
	return Key{Kind: KeyUnknown, Raw: buf[:2]}, 2
}

// decodeWheel reads an SGR 1006 mouse report and answers only the WHEEL.
//
//	ESC [ < Cb ; Cx ; Cy M     press      (m for release)
//
// The sequence is already DELIMITED by the caller's scan — "<" is a parameter
// byte and "M"/"m" are final bytes — so this only interprets what is inside, and
// cannot consume past the end. That is the guarantee that matters here: #14
// shipped a decoder that assumed a length, ate four bytes of a six-byte
// sequence, and typed the remainder into the word being looked up.
//
// Only the wheel, deliberately. Buttons carry COORDINATES that mean nothing
// until there is a region map to look them up in (M2), and a Key kind nothing
// reads is a kind that drifts. A click therefore stays KeyUnknown — consumed
// whole and inert, which is exactly what it should be for now.
func decodeWheel(seq []byte) (Key, bool) {
	if len(seq) < 4 || seq[2] != '<' {
		return Key{}, false
	}
	final := seq[len(seq)-1]
	if final != 'M' && final != 'm' {
		return Key{}, false
	}
	// Cb only. Cx and Cy are parsed by M2, which has somewhere to put them.
	b, ok := atoiPrefix(seq[3 : len(seq)-1])
	if !ok {
		return Key{}, false // malformed: inert, and the caller has still consumed it
	}
	return wheelFromButton(b)
}

// wheelFromButton reads a mouse report's BUTTON byte, which means the same thing
// in every encoding the terminal might answer in.
//
// One owner, because there are already two encodings and M2 adds buttons to
// both: three spellings of "bit 6 is the wheel, the low two bits are the
// direction" across two decoders is how they come to disagree.
//
// The modifier bits (shift 4, meta 8, ctrl 16) ride along and are IGNORED rather
// than matched exactly — Shift-wheel is still a wheel.
func wheelFromButton(b int) (Key, bool) {
	if b&64 == 0 {
		return Key{}, false
	}
	switch b & 3 {
	case 0:
		return Key{Kind: KeyWheelUp}, true
	case 1:
		return Key{Kind: KeyWheelDown}, true
	}
	return Key{}, false // horizontal wheel: nothing to scroll sideways
}

// atoiPrefix reads the leading decimal number of a parameter list. Returns false
// on anything else, including an empty field or a number long enough to be a
// denial-of-sense rather than a coordinate.
func atoiPrefix(b []byte) (int, bool) {
	n, digits := 0, 0
	for _, c := range b {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
		digits++
		if digits > 6 {
			return 0, false
		}
	}
	if digits == 0 {
		return 0, false
	}
	return n, true
}

// decodeX10Mouse reads the legacy mouse report: ESC[M followed by three bytes,
// each a value offset by 32.
//
// It consumes SIX bytes or none. None means "wait" — the partial-sequence
// protocol decodeKey already has — because a report split across two reads must
// not be half-decoded, and the payload bytes are otherwise indistinguishable
// from typed characters.
//
// Only the wheel is answered, matching decodeWheel: a button carries coordinates
// that mean nothing until M2 can look them up. The rest is inert, which for this
// encoding means CONSUMED rather than ignored.
func decodeX10Mouse(buf []byte) (Key, int) {
	if len(buf) < 6 {
		return Key{}, 0
	}
	if k, ok := wheelFromButton(int(buf[3]) - 32); ok {
		return k, 6
	}
	return Key{Kind: KeyUnknown, Raw: buf[:6]}, 6
}
