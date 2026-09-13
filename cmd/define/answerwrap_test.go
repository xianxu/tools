package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestAnswerWrapChunkIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
		width          int
	}{
		{"words", "alpha beta gamma delta omega", "alpha beta gamma\ndelta omega", 20},
		{"paragraphs", "alpha beta gamma delta\n\nlast word", "alpha beta gamma\ndelta\n\nlast word", 20},
		{"spaces", "  alpha   beta gamma delta", "  alpha   beta gamma\ndelta", 20},
		{"styled unicode", "alpha \x1b[32mcafé\x1b[0m 界界 beta gamma", "alpha \x1b[32mcafé\x1b[0m 界界 beta\ngamma", 20},
		{"combining", "alpha cafe\u0301 gamma delta", "alpha cafe\u0301 gamma\ndelta", 20},
		{"oversized word", "x abcdefghijklmnopqrstuvwxyz y", "x\nabcdefghijklmnopqrstuvwxyz\ny", 20},
		{"oversized first", "abcdefghijklmnopqrstuvwxyz y", "abcdefghijklmnopqrstuvwxyz\ny", 20},
		{"trailing spaces", "alpha beta gamma delta  ", "alpha beta gamma\ndelta  ", 20},
		{"incomplete escape", "alpha \x1b[3", "alpha \x1b[3", 20},
		{"incomplete rune", "alpha \xc3", "alpha \xc3", 20},
		{"pipe", "alpha\t beta gamma delta omega\r\n", "alpha\t beta gamma delta omega\r\n", 0},
		{"tabs", "alpha\tbeta gamma delta", "alpha beta gamma\ndelta", 20},
		{"sub minimum", "alpha beta gamma delta", "alpha beta gamma delta", 19},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := func(parts []string) {
				t.Helper()
				var out bytes.Buffer
				w := newAnswerWrapWriter(&out, tc.width)
				for _, p := range parts {
					if n, err := w.Write([]byte(p)); err != nil || n != len(p) {
						t.Fatalf("Write = %d, %v", n, err)
					}
				}
				for range 2 {
					if err := w.Flush(); err != nil {
						t.Fatal(err)
					}
					if out.String() != tc.want {
						t.Fatalf("chunks %q: got %q, want %q", parts, out.String(), tc.want)
					}
				}
			}
			for i := 0; i <= len(tc.in); i++ {
				check([]string{tc.in[:i], tc.in[i:]})
			}
			var parts []string
			for i := range len(tc.in) {
				parts = append(parts, tc.in[i:i+1])
			}
			check(parts)
		})
	}
}

func TestAnswerWrapStreamsCompletedWords(t *testing.T) {
	var out bytes.Buffer
	w := newAnswerWrapWriter(&out, 20)
	w.Write([]byte("alpha beta gam"))
	if out.String() != "alpha beta" {
		t.Fatalf("completed words not streamed: %q", out.String())
	}
	w.Write([]byte("ma delta "))
	if out.String() != "alpha beta gamma\ndelta" {
		t.Fatalf("streamed wrap = %q", out.String())
	}
}

func TestAnswerWrapContinuationStyle(t *testing.T) {
	t.Run("actual highlighted phrase", func(t *testing.T) {
		var s screen
		w := newAnswerWrapWriter(&s, 20)
		hw := newHighlightWriter(w, vocab("alpha beta"), knownOn)
		if _, err := hw.Write([]byte("12345678901234 alpha beta end")); err != nil {
			t.Fatal(err)
		}
		if err := hw.Flush(); err != nil {
			t.Fatal(err)
		}
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
		var frame bytes.Buffer
		s.Paint(&frame, 2, 20, "> ", nil)
		if strings.Contains(frame.String(), "alpha") || !strings.Contains(frame.String(), knownOn+"beta"+sgrOff+" end") {
			t.Fatalf("scrolled phrase lost its highlight: %q", frame.String())
		}
	})
	for _, tc := range []struct{ name, input, style string }{
		{"wrapped highlight", "12345678901234 alpha beta", knownOn},
		{"explicit newline", "alpha\nbeta", "\x1b[1m\x1b[35m"},
		{"wrapped enclosing style", "12345678901234 alpha beta", "\x1b[1m\x1b[35m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s screen
			w := newAnswerWrapWriter(&s, 20)
			input := tc.style + tc.input + sgrOff + " end"
			for i := range len(input) {
				if _, err := w.Write([]byte(input[i : i+1])); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.Flush(); err != nil {
				t.Fatal(err)
			}
			var frame bytes.Buffer
			s.Paint(&frame, 2, 20, "> ", nil) // only the continuation fits above the prompt
			if strings.Contains(frame.String(), "alpha") || !strings.Contains(frame.String(), tc.style+"beta"+sgrOff+" end") {
				t.Fatalf("continuation lost its independent style: %q", frame.String())
			}
		})
	}
}

func TestAnswerWrapPoisonedOnWriteFailure(t *testing.T) {
	boom := errors.New("closed output")
	for _, width := range []int{0, 20} {
		var out bytes.Buffer
		w := newAnswerWrapWriter(&failAfter{out: &out, after: 1, err: boom}, width)
		w.Write([]byte("alpha "))
		w.Write([]byte("beta "))
		if !errors.Is(w.Flush(), boom) {
			t.Fatal("lost downstream error")
		}
		before := out.String()
		if n, err := w.Write([]byte("again")); n != 0 || !errors.Is(err, boom) {
			t.Fatalf("poisoned Write = %d, %v", n, err)
		}
		w.Flush()
		if out.String() != before {
			t.Fatal("output continued after failure")
		}
	}
	w := newAnswerWrapWriter(&shortWriter{limit: 2}, 20)
	w.Write([]byte("alpha"))
	if !errors.Is(w.Flush(), io.ErrShortWrite) {
		t.Fatal("short write not reported by final flush")
	}
}

func TestAnswerWrapPendingLimits(t *testing.T) {
	for _, input := range []string{strings.Repeat("a", 65536), strings.Repeat(" ", 65536), "\x1b[" + strings.Repeat("1", 254)} {
		var out bytes.Buffer
		atBound := newAnswerWrapWriter(&out, 20)
		if _, err := atBound.Write([]byte(input)); err != nil {
			t.Fatal(err)
		}
		if err := atBound.Flush(); err != nil || out.String() != input {
			t.Fatalf("flush at bound: %v, preserved=%v", err, out.String() == input)
		}
		for _, chunk := range []int{1, len(input)} {
			w := newAnswerWrapWriter(io.Discard, 20)
			for i := 0; i < len(input); i += chunk {
				if _, err := w.Write([]byte(input[i:min(i+chunk, len(input))])); err != nil {
					t.Fatalf("at bound: %v", err)
				}
			}
			if _, err := w.Write([]byte("1")); err == nil {
				t.Fatal("excess pending input was accepted")
			} else if w.Flush() != err {
				t.Fatal("limit error not retained")
			}
		}
	}
	// Capacity limits pending data, not a large chunk of ordinary prose.
	w := newAnswerWrapWriter(io.Discard, 20)
	if _, err := w.Write([]byte(strings.Repeat("word ", 65536))); err != nil {
		t.Fatal(err)
	}
	var pipe bytes.Buffer
	w = newAnswerWrapWriter(&pipe, 0)
	input := strings.Repeat("x", 65537)
	if _, err := w.Write([]byte(input)); err != nil || pipe.String() != input {
		t.Fatalf("pipe limit/pass-through: %v", err)
	}
}
