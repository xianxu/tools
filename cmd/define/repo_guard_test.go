package main_test

import (
	"bufio"
	"bytes"
	"io"
	"os"
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

// A compiled binary must never be tracked. `go build ./` inside cmd/define
// writes an extensionless binary named after the directory, and `git add -A`
// swept a 9.6 MB one in.
//
// This covers the INDEX — the state a .gitignore line keeps clean, and the last
// moment the mistake is free. TestNoBinariesInHistory covers where the cost
// actually lives.
func TestNoCommittedBinaries(t *testing.T) {
	dir := repoRoot(t)
	files := strings.Split(string(git(t, "-C", dir, "ls-files", "-z")), "\x00")

	scanned := 0
	for _, f := range files {
		if f == "" {
			continue
		}
		fh, err := os.Open(filepath.Join(dir, f))
		if err != nil {
			continue // deleted or a dangling symlink; not this test's business
		}
		var head [4]byte
		n, _ := fh.Read(head[:])
		fh.Close()
		if n < 4 {
			continue
		}
		scanned++
		if isExecutableImage(head[:]) {
			t.Errorf("compiled binary is tracked: %s", f)
		}
	}
	if scanned == 0 {
		t.Fatal("no files were read; this test would pass vacuously")
	}
}

// The index is not where this class costs anything. Removing a binary in a
// FOLLOW-UP commit leaves the blob reachable, so every clone still pays: the
// 9.6 MB artifact took this branch's clone to 5.9 MB against main's 604 KB
// while `git ls-files` reported it zero times and the index guard above was
// green. The fix is to rewrite the commit that adds it, and this is the test
// that says whether that worked.
func TestNoBinariesInHistory(t *testing.T) {
	dir := repoRoot(t)

	// rev-list --objects prints "<sha> <path>" for blobs and trees, bare shas
	// for commits.
	paths := map[string]string{}
	var req strings.Builder
	for _, line := range strings.Split(string(git(t, "-C", dir, "rev-list", "--objects", "HEAD")), "\n") {
		sha, path, ok := strings.Cut(line, " ")
		if !ok || path == "" {
			continue
		}
		if _, seen := paths[sha]; seen {
			continue
		}
		paths[sha] = path
		req.WriteString(sha + "\n")
	}
	if len(paths) == 0 {
		t.Fatal("rev-list returned no path-bearing objects; this test would pass vacuously")
	}

	cmd := exec.Command("git", "-C", dir, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(req.String())
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("git cat-file: %v", err)
	}
	defer cmd.Wait()

	// Each record is "<sha> <type> <size>\n" followed by <size> bytes and a \n.
	r := bufio.NewReader(pipe)
	blobs := 0
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
			continue // "<sha> missing"
		}
		size, err := strconv.Atoi(f[2])
		if err != nil {
			t.Fatalf("unparsable size in %q", header)
		}
		body := make([]byte, size+1) // +1 for the record's trailing newline
		if _, err := io.ReadFull(r, body); err != nil {
			t.Fatalf("reading object %s: %v", f[0], err)
		}
		if f[1] != "blob" {
			continue
		}
		blobs++
		if isExecutableImage(body) {
			t.Errorf("compiled binary in history: %s (blob %s, %d bytes) — reachable from HEAD, "+
				"so it is fetched by every clone. Rewrite the commit that adds it; deleting it "+
				"in a later commit does not remove the cost.", paths[f[0]], f[0][:8], size)
		}
	}
	if blobs == 0 {
		t.Fatal("no blobs were read; this test would pass vacuously")
	}
}
