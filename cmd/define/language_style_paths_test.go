package main

import (
	"bytes"
	"github.com/creack/pty"
	"io"
	"os"
	"path/filepath"
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
		// -scheme picks the shade; auto with nothing detected is dark (#70).
		{"default", nil, "xterm-256color", languageDark, false},
		{"scheme dark", []string{"-scheme", "dark"}, "xterm-256color", languageDark, false},
		{"scheme light", []string{"-scheme", "light"}, "xterm-256color", languageLight, false},
		{"scheme auto", []string{"-scheme", "auto"}, "xterm-256color", languageDark, false},
		{"off", []string{"-language-tint", "off"}, "xterm-256color", "", false},
		{"plain", []string{"-no-color"}, "xterm-256color", "", false},
		{"dumb", nil, "dumb", "", false},
		{"redirect", nil, "xterm-256color", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)
			d := testDeps(t)
			d.dict = tintSourceFixture{d.dict, "en"}
			args := append([]string{"-no-audio"}, tc.args...)
			args = append(args, "sycophantic")
			text, errout, code := runLookup(t, args, d, !tc.redirect)
			if code != 0 {
				t.Fatalf("exit=%d stderr=%s", code, errout)
			}
			if !strings.Contains(text, "sycophantic") {
				t.Fatal("missing definition")
			}
			if tc.want != "" && !strings.Contains(text, tc.want) {
				t.Fatalf("missing tint %q: %q", tc.want, text)
			}
			if tc.want != "" {
				wantBackground := 236
				if tc.want == languageLight {
					wantBackground = 254
				}
				for row, line := range strings.Split(strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n"), "\n") {
					cells, end := rowTestCells(t, line, 80)
					for col, c := range cells {
						if c.bg != wantBackground {
							t.Fatalf("row %d col %d background=%d, want %d", row, col, c.bg, wantBackground)
						}
					}
					if end.bg != -1 {
						t.Fatalf("row %d leaks background", row)
					}
				}
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

// A usage error is settled before anything opens a store, and -language-tint's
// old shade values are refused by NAME, pointing at the flag that owns the
// shade now (#70: narrowed to on|off, not aliased).
func TestLanguageTintInvalidFlagBeforeStore(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"-language-tint", "light"}, "-scheme light"},
		{[]string{"-language-tint", "dark"}, "-scheme dark"},
		{[]string{"-language-tint", "bogus"}, "invalid -language-tint"},
		{[]string{"-scheme", "sepia"}, "not a colour scheme"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			d := testDeps(t)
			opened := false
			d.newStore = func(options, io.Writer, *deckPermission) storeDeps { opened = true; return storeDeps{} }
			var out, errout bytes.Buffer
			code := run(t.Context(), append(tc.args, "sycophantic"), d, strings.NewReader(""), &out, &errout)
			if code != 2 || opened || !strings.Contains(errout.String(), tc.want) {
				t.Fatalf("code=%d opened=%v stderr=%s", code, opened, &errout)
			}
		})
	}
}

// runLookup runs define in-process with stdout on a real pty (onTerminal) or a
// plain buffer, returning what reached stdout, stderr and the exit code. Colour
// needs a terminal on stdout, so every test of a painted shade through run()
// goes through here.
func runLookup(t *testing.T, args []string, d deps, onTerminal bool) (string, string, int) {
	t.Helper()
	var capture, errout bytes.Buffer
	if !onTerminal {
		code := run(t.Context(), args, d, strings.NewReader(""), &capture, &errout)
		return capture.String(), errout.String(), code
	}
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	if err := pty.Setsize(slave, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); io.Copy(&capture, master) }()
	code := run(t.Context(), args, d, strings.NewReader(""), slave, &errout)
	slave.Close()
	<-done
	return capture.String(), errout.String(), code
}

// A saved scheme governs a run with no -scheme flag; the flag beats it; a garbled
// file warns ONCE and is ignored; no config directory means dark, silently (#70).
func TestSavedSchemeGovernsALookup(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	for _, tc := range []struct {
		name, saved string // "" writes nothing
		noConfig    bool
		args        []string
		want, not   string
		warnings    int
	}{
		{"saved light", "light\n", false, nil, languageLight, languageDark, 0},
		{"flag beats saved", "light\n", false, []string{"-scheme", "dark"}, languageDark, languageLight, 0},
		{"garbled is ignored", "sepia\n", false, nil, languageDark, languageLight, 1},
		{"no config directory", "", true, nil, languageDark, languageLight, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := testDeps(t)
			d.dict = tintSourceFixture{d.dict, "en"}
			dir := t.TempDir()
			if tc.saved != "" {
				if err := os.WriteFile(filepath.Join(dir, "scheme"), []byte(tc.saved), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if !tc.noConfig {
				d.configDir = func() (string, bool) { return dir, true }
			}
			args := append(append([]string{"-no-audio"}, tc.args...), "sycophantic")
			text, errout, code := runLookup(t, args, d, true)
			if code != 0 {
				t.Fatalf("exit=%d stderr=%s", code, errout)
			}
			if !strings.Contains(text, tc.want) || strings.Contains(text, tc.not) {
				t.Fatalf("want %q and not %q in %q", tc.want, tc.not, text)
			}
			if got := strings.Count(errout, "define: ignoring saved scheme:"); got != tc.warnings {
				t.Fatalf("%d warnings, want %d: %q", got, tc.warnings, errout)
			}
		})
	}
}
