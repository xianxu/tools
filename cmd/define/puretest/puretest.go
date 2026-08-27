// Package puretest holds the guards a PURE package must pass.
//
// It exists because #5's schedule and #6's play make the same three claims, and
// #7, #12 and #13 each add another form package — so "copy the guards" becomes
// "copy them five times, and let them drift". storetest.Suite is the precedent:
// one conformance body, many callers.
//
// The guards are separate functions rather than one Suite call because a package
// may legitimately need a different allowlist, and a caller should have to name
// what it is claiming.
package puretest

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ImportsOnly asserts the package imports nothing outside allowed.
//
// Nothing about Go enforces a purity claim — a file here may import anything —
// so the claim needs a guard or it is only a comment. #5's plan asserted that a
// test needing a fake "would not compile", which is false, and this is what
// replaced it.
//
// ZERO imports is a PASS, not a suspicious result. The first version fatalled on
// it as a vacuity guard, which fired on #6's play — a package so pure it needs
// nothing at all, which is the strongest possible version of the claim being
// tested. A vacuity guard has to distinguish "the measurement failed" from "the
// answer is legitimately empty", and here `go list` succeeding IS the
// measurement working. What would be vacuous is a caller passing an allowlist so
// wide it cannot fail, and no guard in this file can see that.
func ImportsOnly(t *testing.T, importPath string, allowed []string) {
	t.Helper()
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}

	out, err := exec.Command("go", "list", "-f", `{{join .Imports "\n"}}`, importPath).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", importPath, err)
	}
	for _, imp := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if imp = strings.TrimSpace(imp); imp == "" {
			continue
		}
		if !ok[imp] {
			t.Errorf("%s imports %q — this package must stay pure; anything needing IO belongs in its caller",
				importPath, imp)
		}
	}
}

// wallClock is every way to read the clock. An import list cannot see this:
// `time` is legitimately imported for its types, and time.Since reads the same
// clock as time.Now. #5's first version grepped one spelling and covered one
// door in a room with six.
var wallClock = []string{
	"time.Now(", "time.Since(", "time.Until(",
	"time.After(", "time.Tick(", "time.NewTimer(", "time.NewTicker(",
}

// NoWallClock asserts no non-test file reads the clock.
//
// Every instant must arrive as a parameter, or the caller cannot control the
// date and nothing in the package is testable.
func NoWallClock(t *testing.T, importPath string) {
	t.Helper()
	eachSourceFile(t, importPath, func(name, src string) {
		for _, banned := range wallClock {
			if strings.Contains(src, banned) {
				t.Errorf("%s calls %s — every instant must arrive as a parameter", name, banned)
			}
		}
	})
}

var storeRef = regexp.MustCompile(`\bstore\.([A-Za-z_][A-Za-z0-9_]*)`)

// StoreSymbolsOnly asserts the package names only the pure parts of store.
//
// The hole ImportsOnly leaves open: allowlisting `store` grants its WHOLE
// surface, and store is where the disk lives — store.NewYAML would open, read
// and write files while passing both other guards, leaving the purity claim
// false with everything green. Allowlisting a package grants everything in it,
// so a mixed dependency has to be guarded by symbol.
func StoreSymbolsOnly(t *testing.T, importPath string, allowed []string) {
	t.Helper()
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}

	found := 0
	eachSourceFile(t, importPath, func(name, src string) {
		for _, line := range strings.Split(src, "\n") {
			// Comments legitimately name store.Store when explaining a contract;
			// it is CODE that must not reach it.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, m := range storeRef.FindAllStringSubmatch(line, -1) {
				found++
				if !ok[m[1]] {
					t.Errorf("%s uses store.%s — only store's pure types and helpers may be named here; "+
						"anything that can reach a disk belongs in the caller", name, m[1])
				}
			}
		}
	})
	if found == 0 {
		t.Fatalf("no store references found in %s — this guard would assert nothing", importPath)
	}
}

// eachSourceFile visits the package's non-test .go files.
func eachSourceFile(t *testing.T, importPath string, fn func(name, src string)) {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", `{{.Dir}}`, importPath).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", importPath, err)
	}
	dir := strings.TrimSpace(string(out))

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	seen := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		seen++
		fn(name, string(b))
	}
	if seen == 0 {
		t.Fatalf("no non-test source files in %s — this guard would assert nothing", dir)
	}
}
