package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/tools/cmd/define/store"
)

func TestLanguagePromptSwitchPaths(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, tc := range []struct {
			name          string
			persistErr    error
			command, next string
		}{
			{"persisted", nil, "es", "🇪🇸 › "},
			{"session only", errDeckDeclined, "es", "🇪🇸 › "},
			{"failed", os.ErrPermission, "es", "🇺🇸 › "},
			{"unknown", nil, "xx", "[xx] › "},
			{"invalid", nil, "nonsense", "🇺🇸 › "},
		} {
			t.Run(bilingualState(raw)+"/"+tc.name, func(t *testing.T) {
				d := testDeps(t)
				d.lang = "en"
				d.persistLang = func(store.Lang) error { return tc.persistErr }
				dict := d.dict
				d.newDict = func(store.Lang, io.Writer) (Dictionary, string) { return dict, "fixture" }
				hist := &memHistory{}
				d.history = hist
				input := "/lang " + tc.command + "\n"
				var out, errout bytes.Buffer
				opt := options{noAudio: true, color: true, tty: true}
				if raw {
					view := paintInto(&out)
					runEditor(t.Context(), scriptKeys(input), &interrupter{}, d, opt, console{view: view, stdout: view, stderr: &errout, finish: func() {}})
					if got := strings.TrimPrefix(stripANSI(view.lastPrompt()), "\r"); got != tc.next {
						t.Errorf("next=%q want %q; %s", got, tc.next, &errout)
					}
					if !strings.Contains(stripANSI(out.String()), "🇺🇸 › /lang "+tc.command+"\r\n") {
						t.Errorf("missing old-language submitted prompt: %q", stripANSI(out.String()))
					}
					if got := hist.Prefix(""); len(got) != 1 || got[0] != strings.TrimSpace(input) {
						t.Errorf("history=%q", got)
					}
				} else {
					replLines(t.Context(), &interrupter{}, d, opt, strings.NewReader(input), &out, &errout, false, true)
					if !strings.HasPrefix(out.String(), "🇺🇸 › ") || !strings.HasSuffix(out.String(), tc.next) {
						t.Errorf("prompts=%q want initial en and final %q", out.String(), tc.next)
					}
				}
			})
		}
	}
}

// run probes stdout itself. A PTY slave gives it a real terminal; scripted
// non-file stdin takes the existing line fallback without needing a console seam.
func TestLanguagePromptStartup(t *testing.T) {
	for _, tc := range []struct {
		name     string
		saved    store.Lang
		args     []string
		stdinTTY bool
		redirect bool
		want     string
		plain    bool
	}{
		{"default", "", nil, true, false, "🇺🇸 › ", false},
		{"saved", "es", nil, true, false, "🇪🇸 › ", false},
		{"override", "es", []string{"-lang", "it"}, true, false, "🇮🇹 › ", false},
		{"no flags", "es", []string{"-no-flags"}, true, false, "[es] › ", false},
		{"no color", "es", []string{"-no-color"}, true, false, "[es] › ", true},
		{"pipe", "es", nil, false, false, "", true},
		{"redirect", "es", nil, true, true, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := testDeps(t)
			d.newStore = openStore
			d.stdinIsTerminal = func() bool { return tc.stdinTTY }
			dir := t.TempDir()
			if tc.saved != "" {
				if err := store.WriteLang(dir, tc.saved); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(dir)
			t.Setenv("DEFINE_NO_BACKGROUND", "1")
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()
			var capture bytes.Buffer
			done := make(chan struct{})
			go func() { defer close(done); io.Copy(&capture, master) }()
			var redirected, errout bytes.Buffer
			var stdout io.Writer = slave
			if tc.redirect {
				stdout = &redirected
			}
			args := append([]string{"-no-audio", "-here"}, tc.args...)
			code := run(t.Context(), args, d, strings.NewReader(""), stdout, &errout)
			slave.Close()
			<-done
			got := capture.String()
			if tc.redirect {
				got = redirected.String()
			}
			if code != 0 {
				t.Fatalf("exit=%d stderr=%s", code, &errout)
			}
			if got != tc.want {
				t.Errorf("stdout=%q want %q", got, tc.want)
			}
			if tc.plain && strings.Contains(got, "\x1b") {
				t.Errorf("plain output has ANSI: %q", got)
			}
		})
	}
}

func TestLanguagePromptResizeUsesActualColumns(t *testing.T) {
	d := testDeps(t)
	d.lang = "es"
	var out syncBuf
	view := paintInto(&out)
	keys := make(chan Key)
	resizes := make(chan winSize)
	done := make(chan int, 1)
	go func() {
		done <- runEditor(t.Context(), keys, &interrupter{}, d, options{color: true, tty: true, noAudio: true}, console{view: view, stdout: view, stderr: &out, finish: func() {}, resizes: resizes})
	}()
	defer func() { close(keys); <-done }()
	for _, tc := range []struct {
		cols int
		want string
	}{{80, "🇪🇸 › "}, {1, "[es] › "}, {2, "🇪🇸 › "}, {80, "🇪🇸 › "}} {
		resizes <- winSize{rows: 24, cols: tc.cols}
		waitFor(t, func() bool { return strings.TrimPrefix(stripANSI(view.lastPrompt()), "\r") == tc.want })
	}
}

func TestLanguagePromptRenderCompletionCursor(t *testing.T) {
	e := Editor{Line: []rune("ca"), Cursor: 1}
	got := RenderLine(e, "t", nil, true, "🇪🇸 › ")
	want := eraseLine + promptOn + "🇪🇸 › " + sgrOff + inputOn + "ca" + sgrOff + greyOn + "t" + greyOff + "\x1b[2D"
	if got != want {
		t.Fatalf("render=%q want %q", got, want)
	}
	if e.String() != "ca" {
		t.Fatalf("editor mutated: %q", e.String())
	}
}
