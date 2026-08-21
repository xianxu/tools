package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A compiled binary must never be tracked.
//
// A 9.6 MB one reached a commit here: `go build ./` inside cmd/define writes an
// extensionless binary named after the directory, and `git add -A` swept it in.
// A .gitignore line fixes that path; this fixes the class, including cmd/
// directories added later.
//
// The test is on the file's MAGIC BYTES, not on its name. An earlier version
// flagged "extensionless files in a source directory" and produced a false
// positive on a tracked symlink — the question is whether a file is an
// executable image, and that is answerable exactly.
func TestNoCommittedBinaries(t *testing.T) {
	// `go test` runs with cwd set to the PACKAGE directory, so a bare
	// `git ls-files` yields paths relative to it. Resolve the root explicitly;
	// the first version of this test skipped the artifact because of that.
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("not a git checkout: %v", err)
	}
	dir := strings.TrimSpace(string(root))
	out, err := exec.Command("git", "-C", dir, "ls-files", "-z").Output()
	if err != nil {
		t.Skipf("git ls-files: %v", err)
	}

	// Mach-O (both endians, 32/64, and universal) and ELF.
	magics := [][]byte{
		{0xCF, 0xFA, 0xED, 0xFE}, {0xCE, 0xFA, 0xED, 0xFE},
		{0xFE, 0xED, 0xFA, 0xCF}, {0xFE, 0xED, 0xFA, 0xCE},
		{0xCA, 0xFE, 0xBA, 0xBE}, {0x7F, 'E', 'L', 'F'},
	}
	for _, f := range strings.Split(string(out), "\x00") {
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
		for _, m := range magics {
			if bytes.Equal(head[:], m) {
				t.Errorf("compiled binary is tracked: %s", f)
				break
			}
		}
	}
}
