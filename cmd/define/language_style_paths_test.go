package main

import (
	"bytes"
	"github.com/creack/pty"
	"io"
	"strings"
	"testing"
)

func TestLanguageTintInvocation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		term, want string
		redirect   bool
	}{
		{"default", nil, "xterm-256color", languageDark, false},
		{"light", []string{"-language-tint", "light"}, "xterm-256color", languageLight, false},
		{"off", []string{"-language-tint", "off"}, "xterm-256color", "", false},
		{"plain", []string{"-no-color"}, "xterm-256color", "", false},
		{"dumb", nil, "dumb", "", false},
		{"redirect", nil, "xterm-256color", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)
			d := testDeps(t)
			var capture, errout bytes.Buffer
			args := append([]string{"-no-audio"}, tc.args...)
			args = append(args, "sycophantic")
			var code int
			if tc.redirect {
				code = run(t.Context(), args, d, strings.NewReader(""), &capture, &errout)
			} else {
				master, slave, err := pty.Open()
				if err != nil {
					t.Fatal(err)
				}
				defer master.Close()
				defer slave.Close()
				done := make(chan struct{})
				go func() { defer close(done); io.Copy(&capture, master) }()
				code = run(t.Context(), args, d, strings.NewReader(""), slave, &errout)
				slave.Close()
				<-done
			}
			if code != 0 {
				t.Fatalf("exit=%d stderr=%s", code, &errout)
			}
			text := capture.String()
			if !strings.Contains(text, "sycophantic") {
				t.Fatal("missing definition")
			}
			if tc.want != "" && !strings.Contains(text, tc.want) {
				t.Fatalf("missing tint %q: %q", tc.want, text)
			}
			if tc.want == "" && (strings.Contains(text, languageDark) || strings.Contains(text, languageLight)) {
				t.Fatalf("unexpected tint: %q", text)
			}
			if (tc.name == "plain" || tc.redirect) && strings.Contains(text, "\x1b") {
				t.Fatalf("ANSI leaked: %q", text)
			}
		})
	}
}

func TestLanguageTintInvalidFlagBeforeStore(t *testing.T) {
	d := testDeps(t)
	opened := false
	d.newStore = func(options, io.Writer, *deckPermission) storeDeps { opened = true; return storeDeps{} }
	var out, errout bytes.Buffer
	code := run(t.Context(), []string{"-language-tint", "bogus", "sycophantic"}, d, strings.NewReader(""), &out, &errout)
	if code != 2 || opened || !strings.Contains(errout.String(), "invalid language tint") {
		t.Fatalf("code=%d opened=%v stderr=%s", code, opened, &errout)
	}
}
