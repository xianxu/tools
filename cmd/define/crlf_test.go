package main

import (
	"bytes"
	"io"
	"testing"
)

func TestCRLFWriter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		writes []string
		want   string
	}{
		{"a bare newline gains its carriage return", []string{"one\ntwo\n"}, "one\r\ntwo\r\n"},
		{"an existing CRLF is left alone", []string{"one\r\ntwo"}, "one\r\ntwo"},
		{"a CR split across writes is not doubled", []string{"one\r", "\ntwo"}, "one\r\ntwo"},
		{"a bare CR is left alone", []string{"one\rtwo"}, "one\rtwo"},
		{"no newlines at all", []string{"plain text"}, "plain text"},
		{"an empty write changes nothing", []string{"a", "", "\nb"}, "a\r\nb"},
		{"consecutive newlines each gain one", []string{"\n\n"}, "\r\n\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &crlfWriter{w: &buf}
			for _, s := range tc.writes {
				n, err := io.WriteString(w, s)
				if err != nil {
					t.Fatalf("write %q: %v", s, err)
				}
				// The contract io.Writer callers rely on: n counts the bytes
				// they handed over, not the bytes that reached the terminal.
				if n != len(s) {
					t.Errorf("wrote %q: n = %d, want %d", s, n, len(s))
				}
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// An io.Writer must report progress in the CALLER's units, and a short write is
// not success. Returning 0 on a partial write claims nothing was consumed, so a
// retry duplicates whatever did reach the terminal.
func TestCRLFWriterReportsProgressOnAShortWrite(t *testing.T) {
	// A writer that accepts 5 translated bytes and then stops.
	short := &shortWriter{limit: 5}
	w := &crlfWriter{w: short}

	n, err := io.WriteString(w, "ab\ncd") // translates to "ab\r\ncd", 6 bytes
	if err == nil {
		t.Fatal("a short underlying write was reported as success")
	}
	if n <= 0 || n >= len("ab\ncd") {
		t.Errorf("n = %d, want progress in caller units strictly between 0 and %d", n, len("ab\ncd"))
	}
}

type shortWriter struct{ limit int }

func (s *shortWriter) Write(p []byte) (int, error) {
	if len(p) > s.limit {
		return s.limit, nil // short, no error: the io.Writer contract's other half
	}
	return len(p), nil
}

// consumed() must replay the translation from the writer's ENTRY state.
//
// Re-seeding it from false gets exactly the case lastWasCR exists for backwards:
// "a\r" then "\nb" needs no inserted carriage return, so a fresh-state replay
// counts one byte that was never written and reports the wrong progress — the
// defect the short-write fix's own comment says it prevents.
func TestCRLFWriterProgressAcrossACarriedCR(t *testing.T) {
	short := &shortWriter{limit: 8}
	w := &crlfWriter{w: short}

	// First write ends with a CR, so lastWasCR carries into the second.
	if _, err := io.WriteString(w, "a\r"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	short.limit = 1 // now cut the next one short
	// "\nb" translates to "\nb" — 2 bytes, NOT 3 — because the CR already
	// arrived. One byte lands.
	n, err := io.WriteString(w, "\nb")
	if err == nil {
		t.Fatal("a short underlying write was reported as success")
	}
	if n != 1 {
		t.Errorf("n = %d, want 1 — the carried CR means \"\\n\" cost one byte, not two", n)
	}
}
