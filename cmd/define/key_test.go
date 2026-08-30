package main

import (
	"regexp"
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
		// The click, which M2.2 gave a meaning. It was KeyUnknown through M1,
		// deliberately: coordinates mean nothing until something can look them
		// up, and a Key kind nothing reads is a kind that drifts.
		{"a left press is a click", "\x1b[<0;10;5M", KeyClick, 10},
		// A press and a release BOTH arrive. Acting on both would play every
		// recording twice, so the release is consumed and dropped.
		{"the release is not a second click", "\x1b[<0;10;5m", KeyUnknown, 10},
		// Middle pastes and right opens a menu, in every terminal a user knows;
		// taking either would break a gesture this program did not invent.
		{"the middle button is not ours", "\x1b[<1;10;5M", KeyUnknown, 10},
		{"the right button is not ours", "\x1b[<2;10;5M", KeyUnknown, 10},
		{"a shift-click is still a click", "\x1b[<4;10;5M", KeyClick, 10},
		{"a horizontal wheel is inert", "\x1b[<66;10;5M", KeyUnknown, 11},
		{"a malformed report is inert, not partial", "\x1b[<;;M", KeyUnknown, 6},
		// A field this program could not read WHOLE must not become a
		// coordinate: acting on half a number is worse than not acting. Note
		// the length — "x" is itself a valid CSI final byte (0x40–0x7E), so the
		// scanner ends the sequence there and this is 7 bytes, not 11. A real
		// terminal never sends it; the row exists so a hand-rolled parameter
		// reader cannot start guessing.
		{"a field with trailing junk is inert", "\x1b[<0;1x;5M", KeyUnknown, 7},
		{"too few parameters is inert", "\x1b[<0;5M", KeyUnknown, 7},
		{"a fourth parameter is inert", "\x1b[<0;5;5;5M", KeyUnknown, 11},
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
//
// It also must never report a coordinate it did not read. That is the failure a
// MOUSE decoder has and a key decoder does not: a key that decodes wrongly is a
// wrong character, while a click that decodes wrongly plays the recording for
// something the user never pointed at — silently, because the click looked
// exactly like a click.
func FuzzDecodeMouseIsBounded(f *testing.F) {
	for _, seed := range []string{
		"\x1b[<64;10;5M", "\x1b[<0;1;1m", "\x1b[<0;1;1M", "\x1b[<999999999;0;0M",
		"\x1b[<", "\x1b[<;;;;M", "\x1b[M \x21\x21", "\x1b[M", "\x1b[<0;1x;5M",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		k, n := decodeKey([]byte(in))
		if n < 0 || n > len(in) {
			t.Fatalf("decodeKey(%q) consumed %d of %d bytes", in, n, len(in))
		}
		switch k.Kind {
		case KeyWheelUp, KeyWheelDown, KeyClick:
			seq := in[:n]
			// It came from something that really was a mouse report: either the
			// SGR form, which ends in its own final byte, or the six-byte X10
			// form. Anything else answering "click" is the decoder inventing a
			// gesture out of typed text.
			sgr := strings.HasPrefix(seq, "\x1b[<") && (seq[n-1] == 'M' || seq[n-1] == 'm')
			x10 := strings.HasPrefix(seq, "\x1b[M") && n == 6
			if !sgr && !x10 {
				t.Fatalf("decodeKey(%q) called %q a mouse event", in, seq)
			}
		}
		// A click's coordinates are always inside the screen. Negative ones
		// would index backwards through the region map.
		if k.Kind == KeyClick && (k.Row < 0 || k.Col < 0) {
			t.Fatalf("decodeKey(%q) reported a click at row %d col %d", in, k.Row, k.Col)
		}
		// And nothing but a click carries a position, so a consumer cannot read
		// one off a key that never had it.
		if k.Kind != KeyClick && (k.Row != 0 || k.Col != 0) {
			t.Fatalf("decodeKey(%q) put a position on a %v", in, k.Kind)
		}
	})
}

// The X10 mouse report, which mode 1000 answers in when a terminal ignores 1006.
//
// ESC[M is followed by THREE RAW BYTES that belong to no CSI grammar. Before
// #30's rework the scan stopped at "M" as a final byte and handed the payload to
// the line as text — a left click typed " !!" into the word being looked up.
// This is #14's family one encoding over, and the rule it leaves behind is that
// for every mode we enable, the decoder answers every encoding that mode can
// reply in.
func TestDecodeX10Mouse(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want KeyKind
		n    int
	}{
		// Button 0 at (1,1): 32+0, 32+1, 32+1.
		{"a left click is consumed WHOLE, payload and all", "\x1b[M \x21\x21", KeyClick, 6},
		{"an X10 wheel up scrolls", "\x1b[M\x60\x21\x21", KeyWheelUp, 6},
		{"an X10 wheel down scrolls", "\x1b[M\x61\x21\x21", KeyWheelDown, 6},
		// Coordinates are raw bytes and may be anything ≥ 32, including bytes
		// that look like the start of a UTF-8 rune.
		// The payload is raw BYTES, not text: a coordinate byte may look like the
		// start of a UTF-8 rune and must never be decoded as one.
		{"a click at a high column", "\x1b[M \xc3\xa9", KeyClick, 6},
		{"a partial report waits rather than half-decoding", "\x1b[M ", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, n := decodeKey([]byte(tc.in))
			if n != tc.n {
				t.Fatalf("decodeKey(%q) consumed %d, want %d — the payload would reach the line as text", tc.in, n, tc.n)
			}
			if n > 0 && k.Kind != tc.want {
				t.Errorf("decodeKey(%q) = kind %v, want %v", tc.in, k.Kind, tc.want)
			}
		})
	}
}

// The whole point, asserted end to end: a click leaves NOTHING for the editor to
// type. This is the observable the operator would meet — characters appearing in
// the word being looked up — rather than a byte count.
func TestX10ClickTypesNothing(t *testing.T) {
	buf := []byte("\x1b[M \x21\x21")
	var typed []rune
	for len(buf) > 0 {
		k, n := decodeKey(buf)
		if n == 0 {
			t.Fatalf("decoder stalled with %q left", buf)
		}
		buf = buf[n:]
		if k.Kind == KeyRune {
			typed = append(typed, k.Rune)
		}
	}
	if len(typed) != 0 {
		t.Errorf("a click typed %q into the line", string(typed))
	}
}

// A click carries WHERE, and the conversion to 0-based happens once — here, at
// the boundary — rather than at whichever consumer remembers (#30 M2.2).
//
// Terminals report 1-based columns and rows. An off-by-one here is a click that
// lands on the wrong word, which is the whole failure this issue exists to
// avoid, and it is silent: every region is one cell to the left of where the
// pointer was.
func TestClickCarriesItsPosition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		in       string
		row, col int
	}{
		{"the top-left cell is 0,0", "\x1b[<0;1;1M", 0, 0},
		{"column and row, in that order on the wire", "\x1b[<0;10;5M", 4, 9},
		// Past 223, where the X10 encoding wraps — which is why 1006 is asked
		// for alongside 1000.
		{"a wide terminal", "\x1b[<0;300;120M", 119, 299},
		// The legacy encoding, offset by 32 and then by the 1-based convention.
		{"X10 reports the same cell", "\x1b[M \x21\x21", 0, 0},
		{"X10, a few cells in", "\x1b[M \x2b\x26", 5, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, n := decodeKey([]byte(tc.in))
			if k.Kind != KeyClick {
				t.Fatalf("decodeKey(%q) = kind %v consumed %d, want a click", tc.in, k.Kind, n)
			}
			if k.Row != tc.row || k.Col != tc.col {
				t.Errorf("click at row %d col %d, want row %d col %d — every region would be off by that much",
					k.Row, k.Col, tc.row, tc.col)
			}
		})
	}
}

// Both fuzz findings, kept as rows so a rewrite of the decoder meets them
// directly rather than waiting for the corpus to rediscover them.
func TestMouseDecoderRejectsWhatNoTerminalSends(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		// SS3 rather than CSI. decodeEscape handles both in one branch because
		// the arrow keys arrive both ways; a mouse report is always CSI.
		{"an SS3 sequence is not a mouse report", "\x1bO<0;1;1M"},
		// X10 spends one byte per coordinate, offset by 32, and 1-based — so a
		// byte of 0x20 is wire coordinate 0, and 0-1 is row -1, which would
		// index backwards through the region map.
		{"an X10 coordinate below the origin", "\x1b[M00 "},
		{"an SGR coordinate of zero", "\x1b[<0;0;1M"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k, n := decodeKey([]byte(tc.in))
			if k.Kind == KeyClick || k.Kind == KeyWheelUp || k.Kind == KeyWheelDown {
				t.Errorf("decodeKey(%q) = %v at row %d col %d — a gesture invented from bytes no terminal sends",
					tc.in, k.Kind, k.Row, k.Col)
			}
			if n == 0 || n > len(tc.in) {
				t.Errorf("decodeKey(%q) consumed %d of %d — the tail would reach the line as text", tc.in, n, len(tc.in))
			}
		})
	}
}

// EVERY ENCODING THE MODES WE ENABLE CAN REPLY IN IS DECODED (#30 M2.6).
//
// This is the rule M1's shipped Critical left behind, made mechanical. Enabling
// mode 1000 without decoding its native X10 form let a click type " !!" into the
// word being looked up: the mode was asked for, and its answer was not read.
//
// So the modes are read OFF THE CONSTANT the program actually sends. Adding a
// mode to mouseOn without adding a row here reddens the suite, which is the only
// version of this rule that survives the next person to enable something.
func TestEveryEnabledMouseModeIsDecoded(t *testing.T) {
	// What each mode can answer in, and one well-formed sample of it.
	//
	// 1005 (UTF-8) and 1015 (urxvt) are deliberately absent: a terminal uses
	// them only when ASKED, and this program never asks. That is why the table
	// is keyed on what we enable rather than on what exists.
	replies := map[string][]struct{ encoding, sample string }{
		"1000": {
			{"X10, the default reply to 1000", "\x1b[M \x21\x21"},
			{"SGR, once 1006 is also on", "\x1b[<0;1;1M"},
		},
		"1006": {
			{"SGR extended coordinates", "\x1b[<0;300;120M"},
		},
	}

	modes := regexp.MustCompile(`\x1b\[\?(\d+)h`).FindAllStringSubmatch(mouseOn, -1)
	if len(modes) == 0 {
		t.Fatal("no modes found in mouseOn; this test would be vacuous")
	}
	for _, m := range modes {
		mode := m[1]
		rows, listed := replies[mode]
		if !listed {
			t.Errorf("mode %s is enabled but no row here says what it can REPLY in. "+
				"Enabling a mode without decoding its answer is how a click came to type "+
				"characters into the line (#30 M1.4b); add the encoding and a sample.", mode)
			continue
		}
		for _, r := range rows {
			t.Run(mode+" — "+r.encoding, func(t *testing.T) {
				k, n := decodeKey([]byte(r.sample))
				if n != len(r.sample) {
					t.Errorf("decodeKey(%q) consumed %d of %d — the tail reaches the line as text",
						r.sample, n, len(r.sample))
				}
				if k.Kind == KeyRune {
					t.Errorf("decodeKey(%q) produced the rune %q — the report was typed rather than read",
						r.sample, k.Rune)
				}
			})
		}
	}
}
