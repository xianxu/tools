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
