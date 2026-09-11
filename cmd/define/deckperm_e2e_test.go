package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// e2eDeps is a real one-shot run: the fixture dictionary, the real openStore, and
// whatever the caller types on stdin.
func e2eDeps(t *testing.T, dir string) deps {
	t.Helper()
	d := testDeps(t) // BEFORE chdir: it loads fixtures relative to the package dir
	d.newStore = openStore
	// THE REAL CAPTURER, not testDeps' noop. withStore fills capture in only when
	// it is nil (main.go:177), and testDeps installs a noopCapturer because most
	// tests are about the DEFINITION half and want no writes at all. Leaving it
	// meant these tests looked a word up, wrote nothing, and therefore never
	// reached the question — three of them "passed" by never exercising the
	// feature. Clearing it is what makes this an end-to-end test.
	d.capture = nil
	t.Chdir(dir)
	return d
}

// DECLINING LEAVES THE DIRECTORY EXACTLY AS IT WAS, verified by LISTING it.
//
// This is the operator's actual complaint — a directory that quietly grew a deck
// — so the assertion is on the filesystem, not on the gate. A test that trusted
// the permission would pass with every write path ungated.
func TestDecliningCreatesNothingOnDisk(t *testing.T) {
	dir := t.TempDir()
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return true }

	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"sycophantic"}, d, strings.NewReader("n\n"), &out, &errb)

	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("declining still wrote %v into the directory", names)
	}
	if !strings.Contains(out.String(), "sycophantic") {
		t.Errorf("the lookup did not print. Declining to SAVE is not declining to "+
			"ANSWER — stdout was %q", out.String())
	}
	if code != 0 {
		t.Errorf("exit = %d, want 0 — the learner answered a question, which is not a "+
			"failure", code)
	}
	if !strings.Contains(errb.String(), "Create one here?") {
		t.Errorf("no question was put; stderr was %q", errb.String())
	}
}

// ACCEPTING CREATES, which is the other half and guards against a gate that
// simply never allows.
func TestAcceptingCreatesTheDeck(t *testing.T) {
	dir := t.TempDir()
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return true }

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"sycophantic"}, d, strings.NewReader("y\n"), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if names := lsNames(t, dir); len(names) == 0 {
		t.Error("accepting created nothing")
	}
	if !store.IsDeck(dir) {
		t.Error("accepting did not produce a deck IsDeck recognises")
	}
}

// NO TERMINAL: nothing created, nothing asked, the lookup still prints. This is
// the operator's stated requirement for the unaskable case.
func TestPipedLookupCreatesNothingAndAsksNothing(t *testing.T) {
	dir := t.TempDir()
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"sycophantic"}, d, strings.NewReader(""), &out, &errb)

	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("a piped lookup created %v", names)
	}
	if strings.Contains(errb.String(), "Create one here?") {
		t.Errorf("a question was put where nobody could answer it: %q", errb.String())
	}
	if !strings.Contains(out.String(), "sycophantic") {
		t.Errorf("the lookup did not print; stdout was %q", out.String())
	}
	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

// --here CREATES UNASKED, on a terminal or not. It is the only path automation
// has, so it must not depend on one.
func TestHereCreatesWithoutAsking(t *testing.T) {
	dir := t.TempDir()
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-here", "sycophantic"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if !store.IsDeck(dir) {
		t.Errorf("--here created no deck; the directory holds %v", lsNames(t, dir))
	}
	if strings.Contains(errb.String(), "Create one here?") {
		t.Errorf("--here asked anyway: %q", errb.String())
	}
}

// AN EXISTING DECK IS NEVER ASKED ABOUT. Without this, every existing user would
// be asked once per directory they already use.
func TestAnExistingDeckIsNeverAskedAbout(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return true }

	var out, errb bytes.Buffer
	// Empty stdin: if it asked, it would read EOF and decline — so a question
	// here would both annoy and silently stop saving.
	if code := run(t.Context(), []string{"sycophantic"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if strings.Contains(errb.String(), "Create one here?") {
		t.Errorf("an established deck was asked about: %q", errb.String())
	}
}

// /lang AFTER A DECLINE WRITES NO lang.txt (#50 PQ-1).
//
// store.WriteLang is a free function, so no amount of wrapping the Store
// interface reaches it — and a one-shot lookup test would never see this path.
func TestLangAfterADeclineWritesNothing(t *testing.T) {
	dir := t.TempDir()
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return true }

	var out, errb bytes.Buffer
	run(t.Context(), []string{"/lang", "es"}, d, strings.NewReader("n\n"), &out, &errb)

	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("/lang after a decline wrote %v — store.WriteLang is a free function "+
			"and needs the permission directly", names)
	}
}

// --stats IN AN UNSAVED DIRECTORY reports empty rather than refusing, and says so
// honestly. This is the operator's design in one assertion: "all others relying
// on history would return empty, as if there's empty history."
func TestStatsInAnUnsavedDirectoryReportsEmpty(t *testing.T) {
	dir := t.TempDir()
	d := e2eDeps(t, dir)
	d.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-stats"}, d, strings.NewReader(""), &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0 — an empty deck is not an error; %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "Nothing yet") {
		t.Errorf("--stats did not report an empty deck; stdout was %q", out.String())
	}
	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("--stats created %v", names)
	}
}
