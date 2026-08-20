package main

import (
	"bytes"
	"strings"
	"testing"
)

func testDeps(t *testing.T) deps { return deps{dict: testDict(t)} }

func TestRunPrintsDefinition(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"sycophantic"}, testDeps(t), &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	got := out.String()
	for _, want := range []string{"sycophantic", "/ˌsikəˈfan(t)ik/", "adjective", "obsequious"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestRunUnknownWordExitsOne(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"rizz"}, testDeps(t), &out, &errb)
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "rizz") {
		t.Errorf("stderr should name the word, got %q", errb.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout should stay empty, got %q", out.String())
	}
}

func TestRunNoArgsIsUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, testDeps(t), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestRunRawPrintsUnparsed(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-raw", "quokka"}, testDeps(t), &out, &errb); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "quok·ka | ˈkwäkə |") {
		t.Errorf("raw output should be the unparsed entry, got %q", out.String())
	}
}

// Colour must be off for a non-TTY writer so piping yields clean text.
func TestRunNoColorWhenNotATerminal(t *testing.T) {
	var out, errb bytes.Buffer
	run([]string{"quokka"}, testDeps(t), &out, &errb)
	if strings.Contains(out.String(), "\x1b[") {
		t.Error("ANSI escapes leaked into non-TTY output")
	}
}
