package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// Every /scheme wiring is pinned by a test that drives the LOOP SHELL supplying
// it (lessons: "Pin a loop shell's wiring with a test that drives that loop
// shell") — the raw editor, the piped loop, and the one-shot run().

// schemeEditor drives runEditor on a real live screen that already shows one
// tinted row, and returns the screen, the terminal bytes and stderr.
func schemeEditor(t *testing.T, d deps, input string) (*liveScreen, *syncBuf, *bytes.Buffer) {
	t.Helper()
	var terminal syncBuf
	var errout bytes.Buffer
	live := newLiveScreen(&terminal, 24, 80)
	live.interval = -1
	live.attachScheme(d.scheme)
	if err := live.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	opt := options{noAudio: true, width: 80, rows: 24, tty: true, color: true, tintOn: true}
	runEditor(t.Context(), keysFor(strings.ReplaceAll(input, "\n", "\r")), &interrupter{}, d, opt,
		console{view: live, stdout: live, stderr: &errout, finish: live.Stop})
	return live, &terminal, &errout
}

// holaRow is the painted row that says hola in a frame, or "".
func holaRow(frame string) string {
	for _, row := range strings.Split(frame, "\r\n") {
		if strings.Contains(stripANSI(row), "hola") {
			return row
		}
	}
	return ""
}

func schemeDeps(t *testing.T, dir string) deps {
	t.Helper()
	d := testDeps(t)
	d.scheme = newSchemeHolder(schemeState{})
	if dir != "" {
		d.configDir = func() (string, bool) { return dir, true }
	}
	return d
}

func TestRawEditorSchemeRepaintsWhatIsOnScreen(t *testing.T) {
	dir := t.TempDir()
	live, terminal, errout := schemeEditor(t, schemeDeps(t, dir), "/scheme light\n")
	row := holaRow(lastFrame(terminal.String()))
	if !strings.Contains(row, languageLight) || strings.Contains(row, languageDark) {
		t.Fatalf("the row already on screen was not repainted light: %q (stderr %q)", row, errout)
	}
	transcript := live.PaintedTranscript()
	if !strings.Contains(transcript, languageLight) || strings.Contains(transcript, languageDark) {
		t.Fatalf("the exit transcript kept the old shade: %q", transcript)
	}
	if !strings.Contains(stripANSI(transcript), "scheme light (saved)") {
		t.Fatalf("no report: %q", stripANSI(transcript))
	}
	if b, err := os.ReadFile(filepath.Join(dir, "scheme")); err != nil || string(b) != "light\n" {
		t.Fatalf("saved %q (%v), want light", b, err)
	}
}

func TestRawEditorSchemeWithNowhereToSave(t *testing.T) {
	live, terminal, _ := schemeEditor(t, schemeDeps(t, ""), "/scheme light\n")
	if !strings.Contains(stripANSI(live.PaintedTranscript()), "scheme light (session only; not saved)") {
		t.Fatalf("report: %q", stripANSI(live.PaintedTranscript()))
	}
	if row := holaRow(lastFrame(terminal.String())); !strings.Contains(row, languageLight) {
		t.Fatalf("a session-only switch still repaints: %q", row)
	}
}

func TestRawEditorSchemeWriteErrorChangesNothing(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A config dir UNDER a regular file: MkdirAll fails, so the save does.
	live, terminal, errout := schemeEditor(t, schemeDeps(t, filepath.Join(file, "define")), "/scheme light\n")
	if !strings.Contains(errout.String(), "define: /scheme:") {
		t.Fatalf("no error reported: %q", errout)
	}
	if row := holaRow(lastFrame(terminal.String())); !strings.Contains(row, languageDark) || strings.Contains(row, languageLight) {
		t.Fatalf("a failed save changed the shade: %q", row)
	}
	if strings.Contains(stripANSI(live.PaintedTranscript()), "saved)") {
		t.Fatalf("something claims a save that failed: %q", stripANSI(live.PaintedTranscript()))
	}
}

func TestRawEditorSchemeReportsNothingDetected(t *testing.T) {
	live, _, _ := schemeEditor(t, schemeDeps(t, t.TempDir()), "/scheme\n")
	want := "scheme dark (default: the terminal has not reported its background)"
	if !strings.Contains(stripANSI(live.PaintedTranscript()), want) {
		t.Fatalf("want %q in %q", want, stripANSI(live.PaintedTranscript()))
	}
}

// schemePiped drives replLines, then looks up a word whose rows are tinted.
func schemePiped(t *testing.T, d deps, input string) (string, string) {
	t.Helper()
	d.dict = tintSourceFixture{d.dict, "en"}
	var out, errout bytes.Buffer
	opt := options{noAudio: true, width: 80, rows: 24, color: true, tintOn: true}
	replLines(t.Context(), &interrupter{}, d, opt, strings.NewReader(input), &out, &errout, false, true)
	return out.String(), errout.String()
}

func TestPipedSchemeSaves(t *testing.T) {
	out, errout := schemePiped(t, schemeDeps(t, t.TempDir()), "/scheme light\nsycophantic\n")
	if !strings.Contains(stripANSI(out), "scheme light (saved)") {
		t.Fatalf("report: %q (stderr %q)", stripANSI(out), errout)
	}
	if !strings.Contains(out, languageLight) || strings.Contains(out, languageDark) {
		t.Fatalf("the lookup after the switch is not light: %q", out)
	}
}

func TestPipedSchemeWithNowhereToSave(t *testing.T) {
	out, errout := schemePiped(t, schemeDeps(t, ""), "/scheme light\nsycophantic\n")
	if !strings.Contains(stripANSI(out), "scheme light (session only; not saved)") {
		t.Fatalf("report: %q (stderr %q)", stripANSI(out), errout)
	}
	if !strings.Contains(out, languageLight) {
		t.Fatalf("a session-only switch still applies to the session: %q", out)
	}
}

func TestOneShotScheme(t *testing.T) {
	dir := t.TempDir()
	oneShot := func(t *testing.T, d deps, args ...string) (string, string, int) {
		t.Helper()
		var out, errout bytes.Buffer
		code := run(t.Context(), args, d, strings.NewReader(""), &out, &errout)
		return out.String(), errout.String(), code
	}
	withDir := func() deps {
		d := testDeps(t)
		d.configDir = func() (string, bool) { return dir, true }
		return d
	}
	if _, errout, code := oneShot(t, withDir(), "/scheme", "light"); code != 0 {
		t.Fatalf("exit %d: %s", code, errout)
	}
	if s, found, err := store.ReadScheme(dir); s != store.SchemeLight || !found || err != nil {
		t.Fatalf("one-shot /scheme light did not save: %q %v %v", s, found, err)
	}
	if out, _, _ := oneShot(t, withDir(), "/scheme"); !strings.Contains(out, "scheme light (saved)") {
		t.Fatalf("bare /scheme: %q", out)
	}
	if out, _, _ := oneShot(t, withDir(), "-scheme", "light", "/scheme", "dark"); !strings.Contains(out, "scheme dark (saved)") {
		t.Fatalf("a flag choice is replaced by /scheme: %q", out)
	}
	if _, errout, code := oneShot(t, testDeps(t), "/scheme", "light"); code != 2 || !strings.Contains(errout, "$XDG_CONFIG_HOME") {
		t.Fatalf("with nowhere to save, a one-shot refuses naming where it looked: exit %d, %q", code, errout)
	}
}
