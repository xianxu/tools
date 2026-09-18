package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// The terminal's answer to backgroundQuery (#70) decodes as ONE report, in each
// terminator a terminal may use, and waits while it is still arriving.
func TestDecodeBackgroundReply(t *testing.T) {
	for _, reply := range []string{
		"\x1b]11;rgb:ffff/ffff/ffff\x07",
		"\x1b]11;rgb:ffff/ffff/ffff\x1b\\",
		"\x1b]11;rgb:ffff/ffff/ffff\x9c", // the 8-bit ST
	} {
		k, n := decodeKey([]byte(reply))
		if k.Kind != KeyBackground || k.Background != store.SchemeLight || n != len(reply) {
			t.Errorf("%q: got kind %v %q consuming %d of %d", reply, k.Kind, k.Background, n, len(reply))
		}
		for i := 1; i < len(reply); i++ {
			if _, n := decodeKey([]byte(reply[:i])); n != 0 {
				t.Errorf("the prefix %q consumed %d; a reply still arriving must wait", reply[:i], n)
			}
		}
	}
	if k, _ := decodeKey([]byte("\x1b]11;rgb:0000/0000/0000\x07")); k.Background != store.SchemeDark {
		t.Errorf("a black background reads as %q", k.Background)
	}
}

// Any other reply FORMAT is swallowed whole: nothing detected, and no byte left
// to be typed — in a sitting a leaked digit is an answer.
func TestDecodeBackgroundReplyOtherFormats(t *testing.T) {
	for _, reply := range []string{"\x1b]11;rgba:ffff/ffff/ffff/ffff\x1b\\", "\x1b]11;#ffffff\x07", "\x1b]11;garbage\x07"} {
		k, n := decodeKey([]byte(reply))
		if k.Kind != KeyUnknown || n != len(reply) {
			t.Errorf("%q: kind %v consuming %d of %d, want one KeyUnknown for the whole reply", reply, k.Kind, n, len(reply))
		}
	}
}

// decodeAll decodes buf to the end, failing if any step would wait.
func decodeAll(t *testing.T, buf string) []Key {
	t.Helper()
	var keys []Key
	for len(buf) > 0 {
		k, n := decodeKey([]byte(buf))
		if n == 0 {
			t.Fatalf("decoding waited on %q", buf)
		}
		keys = append(keys, k)
		buf = buf[n:]
	}
	return keys
}

// Everything that is not a reply decodes EXACTLY as before #70: ESC ] as a
// 2-byte unknown, then the rest. So Alt-] then typing, or then Ctrl-C, is
// unchanged, and a byte that cannot continue a reply ends the swallow.
func TestDecodeOSCAbortsAsToday(t *testing.T) {
	keys := decodeAll(t, "\x1b]x")
	if len(keys) != 2 || keys[0].Kind != KeyUnknown || len(keys[0].Raw) != 2 || keys[1].Kind != KeyRune || keys[1].Rune != 'x' {
		t.Fatalf("Alt-] then x: %+v", keys)
	}
	keys = decodeAll(t, "\x1b]11;rgb\x03")
	if keys[0].Kind != KeyUnknown || len(keys[0].Raw) != 2 || keys[len(keys)-1].Kind != KeyInterrupt {
		t.Fatalf("Ctrl-C inside a would-be reply must abort it and still interrupt: %+v", keys)
	}
	for _, s := range []string{"\x1b]11;rgb:\x7f", "\x1b]11;rgb:\x80", "\x1b]11;rgb:f/f/f\x1b[A", "\x1b]11;rgb:f/f/f\r"} {
		if k, n := decodeKey([]byte(s)); k.Kind != KeyUnknown || n != 2 {
			t.Errorf("%q: kind %v consuming %d, want the 2-byte abort", s, k.Kind, n)
		}
	}
}

// A reply is at most 64 bytes, terminator included. A 64-byte sequence cannot
// carry a valid rgb: payload, so the assertion is the CONSUMED count: whole
// (swallowed) at 64, the 2-byte abort at 65 — with each terminator.
func TestDecodeOSCCap(t *testing.T) {
	const prefix = "\x1b]11;"
	for _, tc := range []struct {
		name, term string
	}{{"BEL", "\x07"}, {"ST", "\x1b\\"}} {
		fits := prefix + strings.Repeat("f", 64-len(prefix)-len(tc.term)) + tc.term
		if k, n := decodeKey([]byte(fits)); n != 64 || k.Kind != KeyUnknown {
			t.Errorf("%s at 64 bytes: kind %v consumed %d, want the whole reply", tc.name, k.Kind, n)
		}
		over := prefix + strings.Repeat("f", 65-len(prefix)-len(tc.term)) + tc.term
		if _, n := decodeKey([]byte(over)); n != 2 {
			t.Errorf("%s at 65 bytes: consumed %d, want the 2-byte abort", tc.name, n)
		}
	}
}

// Through the real reader, with the bytes arriving as SEPARATE writes: the
// decoder genuinely waits on a prefix, and Ctrl-C still gets through.
func TestReadInputBackgroundAcrossWrites(t *testing.T) {
	next := func(t *testing.T, keys <-chan Key) Key {
		t.Helper()
		select {
		case k := <-keys:
			return k
		case <-time.After(2 * time.Second):
			t.Fatal("no key arrived")
		}
		return Key{}
	}
	for _, tc := range []struct {
		name   string
		writes []string
		want   []KeyKind
	}{
		{"Alt-] then Ctrl-C", []string{"\x1b]", "\x03"}, []KeyKind{KeyUnknown, KeyInterrupt}},
		{"Alt-] then a", []string{"\x1b]", "a"}, []KeyKind{KeyUnknown, KeyRune}},
		{"a reply in three writes", []string{"\x1b]1", "1;rgb:ffff/ff", "ff/ffff\x1b\\"}, []KeyKind{KeyBackground}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			pr, pw := io.Pipe()
			defer pw.Close()
			keys := readInput(ctx, pr, nil, nil)
			for _, w := range tc.writes {
				if _, err := io.WriteString(pw, w); err != nil {
					t.Fatal(err)
				}
			}
			for _, want := range tc.want {
				if k := next(t, keys); k.Kind != want {
					t.Fatalf("got %v, want %v", k.Kind, want)
				}
			}
		})
	}
}
