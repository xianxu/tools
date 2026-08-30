package main

import (
	"strings"
	"testing"
)

func TestDecodeKey(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		want     KeyKind
		wantRune rune
		consumed int
	}{
		{"printable", "a", KeyRune, 'a', 1},
		{"multi-byte rune", "é", KeyRune, 'é', 2},
		{"non-latin rune", "♫", KeyRune, '♫', 3},
		{"up", "\x1b[A", KeyUp, 0, 3},
		{"down", "\x1b[B", KeyDown, 0, 3},
		{"right", "\x1b[C", KeyRight, 0, 3},
		{"left", "\x1b[D", KeyLeft, 0, 3},
		{"home csi", "\x1b[H", KeyHome, 0, 3},
		{"end csi", "\x1b[F", KeyEnd, 0, 3},
		{"home ss3", "\x1bOH", KeyHome, 0, 3},
		{"end ss3", "\x1bOF", KeyEnd, 0, 3},
		{"delete", "\x1b[3~", KeyDelete, 0, 4},
		{"backspace del", "\x7f", KeyBackspace, 0, 1},
		{"backspace bs", "\b", KeyBackspace, 0, 1},
		{"enter cr", "\r", KeyEnter, 0, 1},
		{"enter lf", "\n", KeyEnter, 0, 1},
		{"tab", "\t", KeyTab, 0, 1},
		{"ctrl-c", "\x03", KeyInterrupt, 0, 1},
		{"ctrl-d", "\x04", KeyEOF, 0, 1},
		{"ctrl-u / cmd+delete", "\x15", KeyKillLine, 0, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, n := decodeKey([]byte(tc.in))
			if k.Kind != tc.want || n != tc.consumed {
				t.Errorf("decodeKey(%q) = kind %v consumed %d, want kind %v consumed %d",
					tc.in, k.Kind, n, tc.want, tc.consumed)
			}
			if tc.want == KeyRune && k.Rune != tc.wantRune {
				t.Errorf("rune = %q, want %q", k.Rune, tc.wantRune)
			}
		})
	}
}

// consumed == 0 is how the reader learns to wait. Without it a lone ESC that
// arrives in its own read decodes as Escape-then-junk, so arrow keys break
// precisely when the terminal is slow.
func TestDecodeKeyPartialSequences(t *testing.T) {
	for _, in := range []string{"", "\x1b", "\x1b[", "\x1b[3", "\xe2", "\xe2\x99"} {
		if _, n := decodeKey([]byte(in)); n != 0 {
			t.Errorf("decodeKey(%q) consumed %d, want 0 — a partial sequence must wait", in, n)
		}
	}
}

// An unmodelled sequence must be swallowed WHOLE, never leak its tail as text.
func TestDecodeKeyUnknownSequencesAreInert(t *testing.T) {
	for _, tc := range []struct {
		in       string
		consumed int
	}{
		{"\x1b[15~", 5},  // F5 — the tilde family's unmodelled half
		{"\x1b[1;5C", 6}, // Ctrl-Right
		{"\x1b[200~", 6}, // bracketed-paste start
		{"\x1bZ", 2},     // unknown two-byte
		{"\x00", 1},      // stray control byte
	} {
		k, n := decodeKey([]byte(tc.in))
		if k.Kind != KeyUnknown || n != tc.consumed {
			t.Errorf("decodeKey(%q) = kind %v consumed %d, want KeyUnknown consumed %d",
				tc.in, k.Kind, n, tc.consumed)
		}
	}
}

// The viewport keys (#30 M1.4a). They were always DELIMITED correctly — the CSI
// scan finds the real final byte — and then discarded as KeyUnknown; naming them
// is what makes the buffer reachable by a user.
//
// Their modified forms must NOT become page keys: ESC[5;5~ is Ctrl-PageUp, and a
// decoder that matched a prefix would both scroll and leave "5~" to be typed
// into the line, which is the exact bug ESC[3;5~ shipped once already.
func TestDecodeKeyPageKeys(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want KeyKind
		n    int
	}{
		{"\x1b[5~", KeyPageUp, 4},
		{"\x1b[6~", KeyPageDown, 4},
		{"\x1b[5;5~", KeyUnknown, 6},
		{"\x1b[6;5~", KeyUnknown, 6},
	} {
		k, n := decodeKey([]byte(tc.in))
		if k.Kind != tc.want || n != tc.n {
			t.Errorf("decodeKey(%q) = kind %v consumed %d, want kind %v consumed %d",
				tc.in, k.Kind, n, tc.want, tc.n)
		}
	}
}

// A modified Delete must not be mistaken for plain Delete, and must consume its
// whole sequence. This was a Critical: ESC[3;5~ consumed four bytes and inserted
// "5~" into the word being typed.
func TestDecodeKeyModifiedDeleteIsNotDelete(t *testing.T) {
	k, n := decodeKey([]byte("\x1b[3;5~"))
	if k.Kind == KeyDelete {
		t.Error("Ctrl-Delete decoded as plain Delete")
	}
	if n != 6 {
		t.Errorf("consumed %d of a 6-byte sequence — the tail would reach the line as text", n)
	}
	// Plain Delete still works.
	if k, n := decodeKey([]byte("\x1b[3~")); k.Kind != KeyDelete || n != 4 {
		t.Errorf("plain Delete broke: kind %v consumed %d", k.Kind, n)
	}
}

// The fuzz target's real obligation: no byte of an escape sequence may ever
// surface as a rune, or it lands in the word the user is typing.
func FuzzDecodeKeyNeverLeaksEscapeTails(f *testing.F) {
	for _, s := range []string{"\x1b[3;5~", "\x1b[1;3D", "\x1b[200~", "\x1bO", "\x1b["} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		if len(buf) == 0 || buf[0] != 0x1b {
			return
		}
		k, n := decodeKey(buf)
		if k.Kind == KeyRune {
			t.Fatalf("an escape sequence decoded as the rune %q", k.Rune)
		}
		if n > 0 && n < len(buf) && buf[n] == 0x1b {
			return // next sequence starts cleanly
		}
	})
}

// decodeKey is the one function fed arbitrary bytes from outside the program.
func FuzzDecodeKey(f *testing.F) {
	for _, s := range []string{"a", "\x1b[A", "\x1b[3~", "\x1bO", "é", "\x1b[1;5C", "\x00"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		k, n := decodeKey(buf)
		if n < 0 || n > len(buf) {
			t.Fatalf("consumed %d out of range for %d bytes", n, len(buf))
		}
		if n == 0 && k.Kind != KeyUnknown && k.Kind != 0 {
			t.Fatalf("consumed 0 but returned kind %v", k.Kind)
		}
	})
}

// The WHEEL, reported by the terminal only because #30 M1.4b asks it to.
//
// In the alternate screen a terminal translates the wheel into arrow keys by
// default, and this editor binds Up/Down to the history walk — so a scroll
// walked history, and nothing could tell the two apart because they are the same
// bytes (operator-reported).
//
// The sequence is delimited by the CSI scan, not by a length: "<" is a parameter
// byte and "M"/"m" are final bytes. A malformed report must therefore still be
// CONSUMED WHOLE — the failure #14 shipped was a decoder that ate part of a
// sequence and typed the remainder into the word being looked up.
func TestDecodeWheel(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want KeyKind
		n    int
	}{
		{"wheel up", "\x1b[<64;10;5M", KeyWheelUp, 11},
		{"wheel down", "\x1b[<65;10;5M", KeyWheelDown, 11},
		{"the release form is still the gesture", "\x1b[<64;10;5m", KeyWheelUp, 11},
		// Modifier bits ride along: shift 4, meta 8, ctrl 16. Shift-wheel is a
		// wheel, so they are ignored rather than matched exactly.
		{"shift-wheel down", "\x1b[<69;10;5M", KeyWheelDown, 11},
		{"ctrl-wheel up", "\x1b[<80;10;5M", KeyWheelUp, 11},
		// Coordinates past 223, where the LEGACY encoding wrapped — which is the
		// whole reason 1006 is enabled alongside 1000.
		{"a wide terminal", "\x1b[<64;480;300M", KeyWheelUp, 14},
		// Not the wheel: a click carries coordinates that mean nothing until M2
		// has a region map, so it stays inert — consumed whole.
		{"a left click is inert", "\x1b[<0;10;5M", KeyUnknown, 10},
		{"a horizontal wheel is inert", "\x1b[<66;10;5M", KeyUnknown, 11},
		{"a malformed report is inert, not partial", "\x1b[<;;M", KeyUnknown, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, n := decodeKey([]byte(tc.in))
			if k.Kind != tc.want || n != tc.n {
				t.Errorf("decodeKey(%q) = kind %v consumed %d, want kind %v consumed %d",
					tc.in, k.Kind, n, tc.want, tc.n)
			}
		})
	}
}

// A mouse report must never consume past its final byte, whatever is in it: a
// decoder that over-consumes eats the next keystroke, and one that under-consumes
// types the tail into the line.
func FuzzDecodeWheelIsBounded(f *testing.F) {
	for _, seed := range []string{"\x1b[<64;10;5M", "\x1b[<0;1;1m", "\x1b[<999999999;0;0M", "\x1b[<", "\x1b[<;;;;M"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		k, n := decodeKey([]byte(in))
		if n < 0 || n > len(in) {
			t.Fatalf("decodeKey(%q) consumed %d of %d bytes", in, n, len(in))
		}
		// A wheel answer may only come from a sequence that actually ended in a
		// mouse report's final byte.
		if k.Kind == KeyWheelUp || k.Kind == KeyWheelDown {
			seq := in[:n]
			if !strings.HasPrefix(seq, "\x1b[<") || (seq[n-1] != 'M' && seq[n-1] != 'm') {
				t.Fatalf("decodeKey(%q) called %q a wheel event", in, seq)
			}
		}
	})
}
