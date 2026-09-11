package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestParseLangArgs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    store.Lang
		set     bool
		wantErr string
	}{
		{"no argument reports", nil, "", false, ""},
		{"a language", []string{"es"}, "es", true, ""},
		{"case-folded", []string{"ES"}, "es", true, ""},
		{"not a tag", []string{"spanish"}, "", false, "spanish"},
		// The value becomes a path segment; traversal is refused by name so the
		// learner sees what was rejected rather than a generic complaint.
		{"traversal", []string{"../etc"}, "", false, "../etc"},
		{"too many arguments", []string{"es", "en"}, "", false, "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, set, err := parseLangArgs(tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want an error naming %q, got %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not name %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want || set != tc.set {
				t.Errorf("= (%q, %v), want (%q, %v)", got, set, tc.want, tc.set)
			}
		})
	}
}

func TestRunLang(t *testing.T) {
	t.Run("reports the current language", func(t *testing.T) {
		var out bytes.Buffer
		c := commandCtx{lang: "es", stdout: &out, stderr: &bytes.Buffer{}}
		if code := runLang(c, nil); code != 0 {
			t.Errorf("exit = %d", code)
		}
		if !strings.Contains(out.String(), "es") {
			t.Errorf("did not report the language: %q", out.String())
		}
	})

	t.Run("switching calls setLang and says so", func(t *testing.T) {
		var out bytes.Buffer
		var got store.Lang
		c := commandCtx{lang: "en", stdout: &out, stderr: &bytes.Buffer{},
			setLang: func(l store.Lang) error { got = l; return nil }}
		if code := runLang(c, []string{"es"}); code != 0 {
			t.Errorf("exit = %d", code)
		}
		if got != "es" {
			t.Errorf("setLang got %q, want es", got)
		}
		if !strings.Contains(out.String(), "es") {
			t.Errorf("said %q", out.String())
		}
	})

	// A directory with no lang.txt is ALREADY "en" by default, so a no-op switch
	// is the only way to put that on the record. Skipping the write would leave
	// the learner no way to make the implicit default explicit.
	t.Run("re-declaring the current language still persists it", func(t *testing.T) {
		var out bytes.Buffer
		called := false
		c := commandCtx{lang: "en", stdout: &out, stderr: &bytes.Buffer{},
			setLang: func(store.Lang) error { called = true; return nil }}
		if code := runLang(c, []string{"en"}); code != 0 {
			t.Errorf("exit = %d", code)
		}
		if !called {
			t.Error("a no-op switch wrote nothing, so the default stays implicit")
		}
		if out.String() == "" {
			t.Error("said nothing; the learner asked for a state and is in it")
		}
	})

	t.Run("a bad tag is refused by name and changes nothing", func(t *testing.T) {
		var out, errb bytes.Buffer
		called := false
		c := commandCtx{lang: "en", stdout: &out, stderr: &errb,
			setLang: func(store.Lang) error { called = true; return nil }}
		if code := runLang(c, []string{"spanish"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if called {
			t.Error("a rejected tag still reached setLang")
		}
		if !strings.Contains(errb.String(), "spanish") {
			t.Errorf("stderr %q does not name the rejected tag", errb.String())
		}
	})

	// The one case /lang refuses: nowhere to persist to. Contrast /sound, which
	// refuses whenever there is no SESSION — language survives the session, so a
	// one-shot run has durable work to do and must not be told no.
	t.Run("no directory to save to is refused, not silently accepted", func(t *testing.T) {
		var out, errb bytes.Buffer
		c := commandCtx{lang: "en", stdout: &out, stderr: &errb}
		if code := runLang(c, []string{"es"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errb.String(), "-lang") {
			t.Errorf("stderr %q does not point at the flag that works here", errb.String())
		}
	})

	t.Run("a failed write is reported, not swallowed", func(t *testing.T) {
		var out, errb bytes.Buffer
		c := commandCtx{lang: "en", stdout: &out, stderr: &errb,
			setLang: func(store.Lang) error { return os.ErrPermission }}
		if code := runLang(c, []string{"es"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if out.String() != "" {
			t.Errorf("claimed a switch that failed: %q", out.String())
		}
	})
}

// The durable half works without a loop, because a one-shot lookup has no
// session to inherit a mode from. This is the whole reason language persists and
// /sound does not.
func TestLangCommandPersistsFromAOneShotRun(t *testing.T) {
	dir := t.TempDir()
	// ALREADY A DECK, because since #50 a directory is not written to until
	// someone says so — and this test is about whether the setting PERSISTS, not
	// about whether the directory may become a deck. Without this the run is a
	// one-shot with no terminal, which correctly declines and writes nothing;
	// TestPersistLangIsGated covers that path deliberately.
	if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
		t.Fatal(err)
	}
	// testDeps loads the fixture corpus by a path relative to the package
	// directory, so it is built BEFORE chdir, not after.
	d := testDeps(t)
	d.newStore = openStore
	t.Chdir(dir)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"/lang", "es"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if got := store.ReadLang(dir); got != "es" {
		t.Errorf("after a one-shot /lang es, the directory says %q", got)
	}
	// And the next invocation inherits it, which is the point.
	if _, err := os.Stat(filepath.Join(dir, store.LangFileName())); err != nil {
		t.Errorf("the setting is not on disk: %v", err)
	}
}

// -lang applies to THIS invocation and does NOT persist: a script must be able
// to ask a question without mutating state.
func TestLangFlagDoesNotPersist(t *testing.T) {
	dir := t.TempDir()
	// testDeps loads the fixture corpus by a path relative to the package
	// directory, so it is built BEFORE chdir, not after.
	d := testDeps(t)
	d.newStore = openStore
	t.Chdir(dir)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-lang", "es", "/lang"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "es") {
		t.Errorf("-lang did not reach the session: %q", out.String())
	}
	// Derived, not spelled: a negative assertion against a hand-typed name
	// passes VACUOUSLY the moment the writer renames the file.
	if _, err := os.Stat(filepath.Join(dir, store.LangFileName())); !os.IsNotExist(err) {
		t.Error("-lang wrote the setting; it is for one invocation")
	}
}

// An invalid -lang is refused BEFORE anything opens a directory — the same rule
// the --forget usage checks follow: a mistyped flag must not be the thing that
// creates words/ in someone's directory.
func TestBadLangFlagIsRefusedBeforeADirectoryIsTouched(t *testing.T) {
	dir := t.TempDir()
	d := testDeps(t)
	t.Chdir(dir)

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-lang", "../etc", "sycophantic"}, d,
		strings.NewReader(""), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "words")); !os.IsNotExist(err) {
		t.Error("a rejected -lang still created the deck directory")
	}
}
