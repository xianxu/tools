package main_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Mach-O (both endians, 32- and 64-bit, and universal) and ELF.
var executableMagics = [][]byte{
	{0xCF, 0xFA, 0xED, 0xFE}, {0xCE, 0xFA, 0xED, 0xFE},
	{0xFE, 0xED, 0xFA, 0xCF}, {0xFE, 0xED, 0xFA, 0xCE},
	{0xCA, 0xFE, 0xBA, 0xBE}, {0x7F, 'E', 'L', 'F'},
}

// The test is on MAGIC BYTES, not on a filename. An earlier version flagged
// "extensionless files in a source directory" and false-positived on a tracked
// symlink — the question is whether a file is an executable image, and that is
// answerable exactly.
func isExecutableImage(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	for _, m := range executableMagics {
		if bytes.Equal(b[:4], m) {
			return true
		}
	}
	return false
}

// Deliberately Fatal, never Skip. The previous version skipped on any git
// error, so it was a silent no-op in an exported tree or without git on PATH —
// a guard that reports nothing when it cannot run certifies nothing.
func git(t *testing.T, args ...string) []byte {
	t.Helper()
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git %s: %v (this guard must not be skipped)", strings.Join(args, " "), err)
	}
	return out
}

// `go test` runs with cwd set to the PACKAGE directory, so a bare `git ls-files`
// yields paths relative to it. Resolve the root explicitly; the first version of
// this test skipped the very artifact it was written for because of that.
func repoRoot(t *testing.T) string {
	t.Helper()
	return strings.TrimSpace(string(git(t, "rev-parse", "--show-toplevel")))
}

// scanForExecutables reads every blob named by want (sha -> path) out of git and
// returns the ones that are executable images.
//
// It asserts that it CONSUMED THE WHOLE LIST. That assertion is the point: the
// first version of this helper enumerated every object and then reported a clean
// result from a partial scan, because nothing compared the records read against
// the records requested and cat-file's exit status was deferred and dropped.
// Feeding it only the first five shas left the guard GREEN with a planted binary
// still reachable from HEAD — a guard carrying the exact defect it exists to
// catch. A guard that enumerates a work list must assert it reached the end of
// it, and must check the exit status of every process it depends on.
func scanForExecutables(t *testing.T, dir string, want map[string]string) []string {
	t.Helper()
	if len(want) == 0 {
		t.Fatal("nothing to scan; this test would pass vacuously")
	}

	var req strings.Builder
	for sha := range want {
		req.WriteString(sha + "\n")
	}
	cmd := exec.Command("git", "-C", dir, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(req.String())
	var errb bytes.Buffer
	cmd.Stderr = &errb
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("git cat-file: %v", err)
	}

	// Each record is "<sha> <type> <size>\n" then <size> bytes then a newline.
	var found []string
	seen := 0
	r := bufio.NewReader(pipe)
	for {
		header, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading cat-file stream: %v", err)
		}
		f := strings.Fields(header)
		if len(f) < 3 {
			// "<sha> missing": an object git was asked for and could not
			// produce. Never a reason to keep going quietly.
			t.Fatalf("git cat-file could not read %q", strings.TrimSpace(header))
		}
		size, err := strconv.Atoi(f[2])
		if err != nil {
			t.Fatalf("unparsable size in %q", header)
		}
		// The decision needs four bytes; the --batch protocol needs the whole
		// record consumed. Reading the object in full would make this guard's
		// memory scale with exactly the input class it exists to catch — a large
		// enough committed artifact would OOM it rather than be reported — and
		// the cost would grow with history forever, on every `go test`.
		n := 4
		if size < n {
			n = size
		}
		head := make([]byte, n)
		if _, err := io.ReadFull(r, head); err != nil {
			t.Fatalf("reading object %s: %v", f[0], err)
		}
		if _, err := io.CopyN(io.Discard, r, int64(size-n+1)); err != nil { // +1: record's trailing newline
			t.Fatalf("skipping object %s: %v", f[0], err)
		}
		seen++
		if f[1] == "blob" && isExecutableImage(head) {
			found = append(found, fmt.Sprintf("%s (blob %s, %d bytes)", want[f[0]], f[0][:8], size))
		}
	}

	// Both of these are load-bearing, and neither existed in the first version.
	if err := cmd.Wait(); err != nil {
		t.Fatalf("git cat-file exited %v (%s)", err, strings.TrimSpace(errb.String()))
	}
	if seen != len(want) {
		t.Fatalf("scanned %d of %d objects — a partial scan must not report a clean result", seen, len(want))
	}
	return found
}

// A compiled binary must never be tracked. `go build ./` inside cmd/define
// writes an extensionless binary named after the directory, and `git add -A`
// swept a 9.6 MB one in.
//
// This reads the INDEX — the state a .gitignore line keeps clean, and the last
// moment the mistake is free. It reads the index's BLOBS rather than the files
// on disk: the previous version opened each path and skipped anything it could
// not open, so a tracked file missing from the worktree was silently unexamined.
func TestNoCommittedBinaries(t *testing.T) {
	dir := repoRoot(t)

	want := map[string]string{}
	for _, line := range strings.Split(string(git(t, "-C", dir, "ls-files", "-s")), "\n") {
		meta, path, ok := strings.Cut(line, "\t") // "<mode> <sha> <stage>\t<path>"
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) < 2 {
			continue
		}
		want[f[1]] = path // identical content at two paths collapses; either name locates it
	}
	for _, hit := range scanForExecutables(t, dir, want) {
		t.Errorf("compiled binary is tracked: %s", hit)
	}
}

// The index is not where this class costs anything. Removing a binary in a
// FOLLOW-UP commit leaves the blob reachable, so every clone still pays: the
// 9.6 MB artifact took this branch's clone to 5.9 MB against main's 604 KB
// while `git ls-files` reported it zero times and the index guard was green.
// The fix is to rewrite the commit that adds it, and this is the test that says
// whether that worked.
func TestNoBinariesInHistory(t *testing.T) {
	dir := repoRoot(t)

	// rev-list --objects prints "<sha> <path>" for blobs and trees, bare shas
	// for commits.
	want := map[string]string{}
	for _, line := range strings.Split(string(git(t, "-C", dir, "rev-list", "--objects", "HEAD")), "\n") {
		sha, path, ok := strings.Cut(line, " ")
		if !ok || path == "" {
			continue
		}
		if _, dup := want[sha]; dup {
			continue
		}
		want[sha] = path
	}
	for _, hit := range scanForExecutables(t, dir, want) {
		t.Errorf("compiled binary in history: %s — reachable from HEAD, so it is fetched by "+
			"every clone. Rewrite the commit that adds it; deleting it in a later commit does not "+
			"remove the cost.", hit)
	}
}

// A deck is runtime state, never source. Three of its files reached commits
// because .gitignore anchored at the repo root while `go test` and the pty
// suite both run with cwd set to cmd/define/, so the deck they created matched
// no pattern.
//
// Checked against the INDEX by path, not by content: a deck file is perfectly
// valid YAML, so nothing about the file itself says it does not belong.
func TestNoTrackedRuntimeState(t *testing.T) {
	dir := repoRoot(t)
	files := strings.Split(string(git(t, "-C", dir, "ls-files", "-z")), "\x00")

	seen := 0
	for _, f := range files {
		if f == "" {
			continue
		}
		seen++
		for _, p := range strings.Split(filepath.ToSlash(f), "/") {
			if p == "words" || p == "events" {
				t.Errorf("runtime deck state is tracked: %s", f)
				break
			}
		}
	}
	if seen == 0 {
		t.Fatal("no tracked files were examined; this test would pass vacuously")
	}
}
