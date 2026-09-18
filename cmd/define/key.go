package main

import (
	"bytes"
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"unicode/utf8"
)

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
	// Raw pointer phases carry positions. KeyClick is produced only on release.
	KeyClick // completed gesture; only the pointer router creates it
	KeyPointerPress
	KeyPointerMotion
	KeyPointerRelease
	// KeyPaste carries a whole bracketed paste in Raw — TEXT, already stripped
	// of escapes and control runes, not the unmodelled escape tail Raw holds for
	// KeyUnknown.
	//
	// ONE key rather than a rune per character, and that is load-bearing:
	// readInput delivers into a 256-key channel that DROPS THE NEWEST when full
	// (selection_input.go:157,204), so a 1000-rune paste arriving per-rune would
	// lose its tail behind a single "input full" notice.
	KeyPaste
	// KeyPasteRefused is a paste over maxPasteRunes. It carries no text: the
	// refusal is the message, and the caller reports it.
	KeyPasteRefused
	// KeyBackground is a terminal REPORT, not a keystroke: the answer to
	// backgroundQuery (#70), carrying the scheme in Background. Never typing,
	// never an answer, never cancels a gesture.
	KeyBackground
	// numKeyKinds is NOT a kind: it is the registry's extent, so a guard can
	// DERIVE the set rather than restate it — the move numRegionKinds already
	// makes for regions.
	//
	// It exists because #67 added KeyPaste and nothing forced the question "what
	// does a sitting do with this?". The answer was "silently nothing", and the
	// milestone had already turned mode 2004 on for that surface. A sentinel is
	// what turns a new kind into a decision instead of an omission.
	numKeyKinds
)

// Key is one decoded keypress.
//
// Raw carries BYTES, and what they mean depends on the kind. For KeyUnknown they
// are an unmodelled escape sequence, kept so it can be ignored rather than
// inserted as garbage — the failure mode of a decoder that falls through to "it
// must be text". For KeyPaste they are the opposite: sanitised TEXT that Apply
// inserts at the cursor. One field, two meanings, disambiguated by Kind.
type Key struct {
	Kind KeyKind
	Rune rune
	Raw  []byte
	// Row and Col are zero-based terminal cells for raw pointer phases and
	// completed clicks. The wire conversion happens once in clickAt.
	Row, Col int
	click    *pointerClick // immutable ticket for a completed application click
	// Background is what a KeyBackground reported.
	Background store.Scheme
}

// keyDecoder is decodeKey plus the one piece of state a byte stream needs.
//
// A bracketed paste spans reads, so its scanner has to survive between calls —
// everything else here is a pure function of the buffer. The streaming caller
// (readInput) holds one for the life of its goroutine; every other call site
// goes through decodeKey, which allocates a fresh one.
type keyDecoder struct{ paste pasteScanner }

// decodeKey converts the front of buf into a Key, with no carried state.
//
// It is the stateless entry the package has always had, kept because 38 call
// sites and both fuzz targets want exactly that: a fresh decoder cannot be
// mid-paste, so paste state can never leak between fuzz inputs or between
// unrelated tests.
func decodeKey(buf []byte) (Key, int) {
	var d keyDecoder
	return d.decode(buf)
}

// decode converts the front of buf into a Key.
//
// consumed == 0 means buf holds a PREFIX of a longer sequence and the caller
// must read more before deciding. Without that signal a lone ESC arriving in its
// own read decodes as Escape-then-junk, and arrow keys break exactly when the
// terminal is slow.
func (d *keyDecoder) decode(buf []byte) (Key, int) {
	if len(buf) == 0 {
		return Key{}, 0
	}
	// The paste scanner goes FIRST, and its condition is not "the byte is ESC".
	//
	// While draining an over-cap paste the head of the buffer is ordinary body
	// text, so an ESC-only hook would never consult the scanner again and the
	// rest of the paste would arrive as runes — the exact failure the drain
	// exists to prevent. The rule is: every byte while draining, 0x1b otherwise.
	if d.paste.draining || buf[0] == 0x1b {
		if k, used, _ := d.paste.scan(buf); used > 0 {
			return k, used
		} else if d.paste.draining || bytes.HasPrefix(buf, []byte(pasteStart)) {
			// Ours, but incomplete: wait rather than letting the CSI scan below
			// swallow a start marker as an unmodelled sequence.
			return Key{}, 0
		}
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
	case ']':
		return decodeOSC(buf)
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
				// enabling mode 1002 asks for the mouse, and a terminal that
				// honours 1002 but ignores 1006 answers in X10 — ESC[M plus
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

// decodeWheel decodes a complete SGR 1006 report: ESC[<button;col;rowM
// (lowercase m for release). The caller owns framing; this owns button meaning.
func decodeWheel(seq []byte) (Key, bool) {
	// CSI, not SS3. decodeEscape handles `ESC [` and `ESC O` in one branch
	// because the arrow keys arrive both ways, and without this check a
	// hand-typed `ESC O <0;1;1M` would be read as a click — a decoder inventing
	// a gesture out of text nobody made. Found by the fuzzer; no terminal sends
	// it, which is exactly why nothing else would have caught it.
	if len(seq) < 4 || seq[1] != '[' || seq[2] != '<' {
		return Key{}, false
	}
	final := seq[len(seq)-1]
	if final != 'M' && final != 'm' {
		return Key{}, false
	}
	params := parseParams(seq[3 : len(seq)-1])
	if len(params) != 3 {
		return Key{}, false // malformed: inert, and the caller has still consumed it
	}
	if k, ok := wheelFromButton(params[0]); ok {
		return k, true
	}
	kind, ok := pointerButton(params[0], final == 'm', false)
	if !ok {
		return Key{}, false
	}
	k, ok := clickAt(params[1], params[2])
	k.Kind = kind
	return k, ok
}

// clickAt builds a click from WIRE coordinates — 1-based, as every terminal
// reports them — and refuses anything that is not a cell on the screen.
//
// One owner for the conversion and the guard, because there are two encodings
// and they are wrong in different ways. Found by the fuzzer: X10 spends one byte
// per coordinate offset by 32, so a byte of 0x20 decodes to wire coordinate 0,
// and 0 - 1 is row -1 — which would index backwards through the region map. SGR
// can report a literal 0 for the same result.
//
// A refusal is inert, and inert is the right answer: the sequence has still been
// consumed whole, so nothing reaches the line either way.
func clickAt(wireCol, wireRow int) (Key, bool) {
	if wireCol < 1 || wireRow < 1 {
		return Key{}, false
	}
	// 1-based on the wire, 0-based here — converted once, at the boundary,
	// rather than at whichever consumer remembers.
	return Key{Kind: KeyPointerPress, Col: wireCol - 1, Row: wireRow - 1}, true
}

// isClickButton reports whether a button byte is a plain press we act on.
//
// The low two bits are the button (left, middle, right) and bit 5 marks MOTION —
// a drag, which mode 1000 does not report but 1002 would, and which is a
// selection gesture rather than a click. Modifier bits ride along and are
// ignored, as they are for the wheel: a Shift-click is still a click.
func isClickButton(b int) bool {
	if b&64 != 0 || b&32 != 0 { // a wheel event, or motion
		return false
	}
	if b&128 != 0 {
		// Buttons 8-11 (back, forward, and two more) set bit 7, and their low
		// two bits are zero — so a mask of `b&3 == 0` called every one of them a
		// left press while the comment promised "LEFT only". Measured:
		// ESC[<128;5;3M decoded to a click. A browser-back button should not
		// play a recording.
		return false
	}
	// The remaining modifier bits — shift 4, meta 8, ctrl 16 — ride along and are
	// ignored: a Shift-click is still a click.
	return b&3 == 0 // LEFT only: middle pastes and right opens a menu, elsewhere
}

// parseParams splits a mouse report's `1;2;3` parameter list.
//
// Returns nil on anything malformed rather than a partial list, so a caller
// cannot act on coordinates it did not actually read — which is the failure mode
// a mouse decoder has that a key decoder does not.
func parseParams(b []byte) []int {
	var out []int
	for _, field := range strings.Split(string(b), ";") {
		n, ok := atoiPrefix([]byte(field))
		if !ok || len(field) != digits(n) {
			return nil
		}
		out = append(out, n)
	}
	return out
}

// digits is how many characters n was written with, so a field carrying trailing
// junk ("12x") is rejected rather than silently read as 12.
func digits(n int) int {
	if n == 0 {
		return 1
	}
	d := 0
	for ; n > 0; n /= 10 {
		d++
	}
	return d
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
func decodeX10Mouse(buf []byte) (Key, int) {
	if len(buf) < 6 {
		return Key{}, 0
	}
	b := int(buf[3]) - 32
	if k, ok := wheelFromButton(b); ok {
		return k, 6
	}
	if kind, ok := pointerButton(b, false, true); ok {
		if k, ok := clickAt(int(buf[4])-32, int(buf[5])-32); ok {
			k.Kind = kind
			return k, 6
		}
	}
	return Key{Kind: KeyUnknown, Raw: buf[:6]}, 6
}

// pointerButton decodes the shared button bit field. Legacy release has no
// button identity; only the gesture policy can decide whether it ends a drag.
func pointerButton(b int, released, legacy bool) (KeyKind, bool) {
	if b < 0 || b & ^63 != 0 {
		return KeyUnknown, false
	}
	if legacy && b&3 == 3 && b&32 == 0 {
		return KeyPointerRelease, true
	}
	if b&3 != 0 {
		return KeyUnknown, false
	}
	if released {
		if b&32 != 0 {
			return KeyUnknown, false
		}
		return KeyPointerRelease, true
	}
	if b&32 != 0 {
		return KeyPointerMotion, true
	}
	if isClickButton(b) {
		return KeyPointerPress, true
	}
	return KeyUnknown, false
}

func isPointerKey(k KeyKind) bool {
	return k == KeyPointerPress || k == KeyPointerMotion || k == KeyPointerRelease
}

// oscBackgroundReply is the front of the terminal's answer to backgroundQuery.
const oscBackgroundReply = "\x1b]11;"

// maxOSCReply bounds a reply, terminator included. A real one is about 25 bytes
// (rgba: about 30); past this it is not a reply.
const maxOSCReply = 64

// decodeOSC decodes ESC ] … (#70). It SWALLOWS only the reply to the question
// this program asks, and in two steps so no reply FORMAT can leak as typing — in
// a sitting a leaked character is an answer:
//
//  1. swallow: ESC ] 11 ; then bytes in 0x20-0x7E up to BEL, ST (ESC \) or the
//     8-bit ST 0x9C (never legal inside a payload), at most maxOSCReply bytes in
//     all. A sequence longer than that is not a reply (a real one is ~25 bytes),
//     so it decodes as it always has — the one way past this rule, and it takes
//     a terminal no one ships. Any byte outside 0x20-0x7E other than a
//     terminator — Ctrl-C, Enter, DEL, 0x80+, an ESC not followed by \ — ABORTS,
//     and the input decodes exactly as it always has: ESC ] as a 2-byte
//     KeyUnknown, then the rest. So Alt-] with meta-sends-escape, then typing or
//     Ctrl-C, behaves as before; a user would have to type "11;" straight after
//     Alt-] to enter the swallow at all.
//  2. parse: rgb: → KeyBackground; any other payload → KeyUnknown, swallowed,
//     nothing detected.
//
// While the buffer is still a PREFIX of a reply it returns 0 and waits. 0x03 is
// never part of one, so Ctrl-C always aborts in the same pass (lessons: "Trusted
// ANSI parsing is not untrusted control filtering").
func decodeOSC(buf []byte) (Key, int) {
	abort := func() (Key, int) { return Key{Kind: KeyUnknown, Raw: buf[:2]}, 2 }
	for i := 2; i < len(oscBackgroundReply); i++ {
		if i >= len(buf) {
			return Key{}, 0
		}
		if buf[i] != oscBackgroundReply[i] {
			return abort()
		}
	}
	for i := len(oscBackgroundReply); i < len(buf); i++ {
		if i >= maxOSCReply {
			return abort()
		}
		switch c := buf[i]; {
		case c == 0x07 || c == 0x9c:
			return backgroundKey(buf[len(oscBackgroundReply):i], buf[:i+1]), i + 1
		case c == 0x1b:
			if i+1 >= maxOSCReply {
				return abort() // ST would end past the cap
			}
			if i+1 == len(buf) {
				return Key{}, 0
			}
			if buf[i+1] != '\\' {
				return abort()
			}
			return backgroundKey(buf[len(oscBackgroundReply):i], buf[:i+2]), i + 2
		case c < 0x20 || c > 0x7e:
			return abort()
		}
	}
	if len(buf) >= maxOSCReply {
		return abort()
	}
	return Key{}, 0
}

// backgroundKey turns a swallowed reply into a report, or into nothing at all.
func backgroundKey(payload, raw []byte) Key {
	if s, ok := parseBackgroundColour(string(payload)); ok {
		return Key{Kind: KeyBackground, Background: s, Raw: raw}
	}
	return Key{Kind: KeyUnknown, Raw: raw}
}
