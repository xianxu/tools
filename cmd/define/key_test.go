package main

import "testing"

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
		{"\x1b[5~", 4},   // PageUp
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
