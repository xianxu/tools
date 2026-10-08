package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// copyFixture puts testdata/take.cast and its sidecar captions in a temp dir.
func copyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range []string{"take.cast", "take.captions.txt"} {
		data, err := os.ReadFile(filepath.Join("testdata", f))
		if err != nil {
			t.Fatalf("fixture %s: %v", f, err) // a committed fixture is never optional
		}
		if err := os.WriteFile(filepath.Join(dir, f), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCutCLIDefaultsCaptionsToTheSidecar(t *testing.T) {
	dir := copyFixture(t)
	take, out := filepath.Join(dir, "take.cast"), filepath.Join(dir, "cut.cast")
	// Flags after the positional, as typed.
	code, stdout, stderr := runCLI(t, "cut", take, "-o", out, "--speed", "4")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.HasPrefix(stdout, out+": ") || !strings.Contains(stdout, "4 captions") ||
		!strings.Contains(stdout, "Listing shows <two> files & nothing else.") {
		t.Errorf("summary:\n%s", stdout)
	}

	in := mustParse(t, take)
	got := mustParse(t, out)
	caps, err := parseCaptions("", mustRead(t, sidecarPath(take)))
	if err != nil {
		t.Fatal(err)
	}
	tm := defaultTiming
	tm.Speed = 4
	checkCut(t, in, caps, tm, got)

	// The contract CastEmbed.astro reads: header.captions = [{start, end, text}].
	hc := headerCaptions(t, got)
	if len(hc) != 4 || hc[1].Text != "Listing shows <two> files & nothing else." || !(hc[0].Start < hc[0].End) {
		t.Errorf("header captions = %+v", hc)
	}
	if bytes.Contains(mustRead(t, out), []byte(`\`+`u003c`)) {
		t.Error("captions are HTML-escaped in the output")
	}
}

func TestCutCLIExplicitCaptionsAndErrors(t *testing.T) {
	dir := copyFixture(t)
	take, out := filepath.Join(dir, "take.cast"), filepath.Join(dir, "cut.cast")
	late := filepath.Join(dir, "late.txt")
	if err := os.WriteFile(late, []byte("~9:00  far too late\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCLI(t, "cut", take, late, "-o", out)
	if code != 1 || !strings.HasPrefix(stderr, "castcut: caption at 540.0s is past the end") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("an output file was written for a failed cut")
	}

	if code, _, stderr := runCLI(t, "cut", take, "-o", take); code != 1 || !strings.Contains(stderr, "would overwrite an input") {
		t.Errorf("-o onto the take: exit %d %q", code, stderr)
	}
	for _, args := range [][]string{
		{"cut", "-o", out},                         // no take
		{"cut", take, late, "extra", "-o", out},    // too many
		{"cut", take, "-o", out, "--no-such-flag"}, // unknown flag
	} {
		if code, _, _ := runCLI(t, args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, stderr := runCLI(t, "cut", take, "-o", out, "--speed", "0"); code != 1 || !strings.Contains(stderr, "must be positive") {
		t.Errorf("--speed 0: exit %d %q", code, stderr)
	}
	if code, _, stderr := runCLI(t, "frobnicate"); code != 1 || !strings.Contains(stderr, "unknown command") {
		t.Errorf("unknown command: exit %d %q", code, stderr)
	}
}

func TestCutCLIDefaultsTheOutputBesideTheTake(t *testing.T) {
	dir := copyFixture(t)
	take := filepath.Join(dir, "take.cast")
	code, stdout, stderr := runCLI(t, "cut", take)
	want := filepath.Join(dir, "take-cut.cast")
	if code != 0 || !strings.HasPrefix(stdout, want+": ") {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	first := mustRead(t, want)
	// Re-cutting overwrites the default output; it is derived, not a take.
	if code, _, stderr := runCLI(t, "cut", take, "--speed", "2"); code != 0 {
		t.Fatalf("re-cut: exit %d %s", code, stderr)
	}
	if bytes.Equal(first, mustRead(t, want)) {
		t.Error("re-cut with other pacing did not replace the default output")
	}
}

func TestVersion(t *testing.T) {
	if code, out, _ := runCLI(t, "--version"); code != 0 || out != "castcut built from source\n" {
		t.Errorf("exit %d %q", code, out)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustParse(t *testing.T, path string) Cast {
	t.Helper()
	c, err := parseCast(path, mustRead(t, path))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
