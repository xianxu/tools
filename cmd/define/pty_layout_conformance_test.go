//go:build darwin && conformance

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/tools/internal/conformance"
)

// StartWithSize sets geometry before exec; setting it after starting a one-shot
// command races its initial terminal-size probe. The binary and watcher are the
// same source-built helpers used by the interactive PTY conformance tests.
func TestPTYNativeRendirSectionLayout(t *testing.T) {
	source := bilingualNativeProbe(t)
	records, err := source.Records("rendir")
	if err != nil {
		conformance.SkipOrFail(t, "native rendir unavailable", err)
		return
	}
	selected, err := selectedSpanishRecords(records, "rendir", "")
	if err != nil || len(selected) != 1 {
		t.Fatalf("native rendir selection: %d records, %v", len(selected), err)
	}
	ids, name := spanishDictionarySources(installedDictionaries())
	if len(ids) == 0 || strings.Contains(name, "unavailable") || strings.Contains(name, "unknown") {
		conformance.SkipOrFail(t, "verified Spanish primary unavailable", ErrBilingualUnavailable)
		return
	}
	// -scheme picks the tint's shade; -language-tint off removes it (#70).
	for _, profile := range []struct {
		name, flag string
		bg         int
	}{{"dark", "--scheme=dark", 236}, {"light", "--scheme=light", 254}, {"off", "--language-tint=off", -1}} {
		for _, width := range []int{32, 80} {
			t.Run(fmt.Sprintf("%s-%d", profile.name, width), func(t *testing.T) {
				cmd := exec.Command(builtBinary(t), "--lang=es", "--no-audio", "--no-flags", profile.flag, "rendir")
				cmd.Dir = t.TempDir()
				// Its own config directory, as startDefineBinary gives every launch (#70).
				cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir(), "TERM=xterm-256color", "DEFINE_NO_BACKGROUND=1", "DEFINE_NO_CAPTURE=1", "NO_COLOR=")
				f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 160, Cols: uint16(width)})
				if err != nil {
					conformance.SkipOrFail(t, "no pty available", err)
					return
				}
				t.Cleanup(func() { _ = cmd.Process.Kill(); _ = f.Close() })
				out := watch(f)
				done := make(chan error, 1)
				go func() { done <- cmd.Wait() }()
				select {
				case err := <-done:
					if err != nil {
						t.Fatalf("one-shot exit: %v; output %q", err, out.take(20*time.Millisecond))
					}
				case <-time.After(8 * time.Second):
					t.Fatal("one-shot dictionary lookup did not finish")
				}
				got := out.take(50 * time.Millisecond)
				if dir := os.Getenv("DEFINE_TINT_PTY_CAPTURE_DIR"); dir != "" {
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					path := filepath.Join(dir, fmt.Sprintf("rendir-%s-%d.raw", profile.name, width))
					if err := os.WriteFile(path, []byte(got), 0600); err != nil {
						t.Fatal(err)
					}
					t.Logf("actual entrypoint PTY capture: %s", path)
				}
				got = strings.ReplaceAll(got, "\r", "")
				if strings.Contains(got, "\x1b[?1049h") {
					t.Fatal("one-shot entered interactive alternate screen")
				}
				plain := stripEscapes(got)
				if !strings.Contains(oxfordContent(plain), oxfordContent(selected[0].Text)) {
					t.Fatal("one-shot PTY lost or reordered native Oxford content")
				}
				if !strings.Contains(plain, "A transitive verb") || !strings.Contains(plain, "B intransitive verb") || !strings.Contains(plain, "C (rendirse) pronominal verb") {
					t.Fatalf("native Oxford hierarchy missing: %q", plain)
				}
				lines := strings.Split(got, "\n")
				second := -1
				for i, line := range lines {
					if strings.HasPrefix(stripEscapes(line), "English") {
						second = i
						break
					}
				}
				if second < 1 {
					t.Fatalf("missing Oxford heading: %q", plain)
				}
				for i, line := range lines[:len(lines)-1] {
					want := -1
					if i < second-1 {
						want = profile.bg
					}
					cells, end := rowTestCells(t, line, width)
					for col, c := range cells {
						if c.bg != want {
							t.Fatalf("row %d col %d bg %d want %d: %q", i, col, c.bg, want, stripEscapes(line))
						}
					}
					if end.bg != -1 {
						t.Fatalf("row %d leaked background", i)
					}
					if profile.name == "off" && strings.TrimRight(stripEscapes(line), " ") != stripEscapes(line) {
						t.Fatalf("off profile added right padding to row %d", i)
					}
				}
			})
		}
	}
}
