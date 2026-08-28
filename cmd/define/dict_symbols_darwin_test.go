//go:build darwin

package main

import (
	"os"
	"regexp"
	"testing"
)

// The Go symbol list and the C resolver must name the SAME symbols.
//
// dcsPrivateSymbols is documented as the one producer, and the conformance check
// walks it — but dcs_resolve hand-writes the same three names as C string
// literals, so adding or renaming a dlsym in the preamble would leave that check
// green while the resolver needed a symbol nobody verified. That is BR-7's shape
// exactly — a guard asserting coverage from a hand-typed restatement — applied to
// the list the whole OS-version risk of #23 M2 rests on.
//
// Reading the source is the honest way to close it: cgo gives Go no view of the
// preamble, so the two copies cannot be unified — but they CAN be compared, and
// a comparison is what turns "documented as one producer" into a fact.
//
// In-package and darwin-tagged, because the list and the preamble both are.
func TestTheCResolverAndTheGoSymbolListAgree(t *testing.T) {
	// A path relative to the package directory: go test runs here, and this file
	// is compiled only where that source exists.
	b, err := os.ReadFile("dict_darwin.go")
	if err != nil {
		t.Fatalf("reading dict_darwin.go: %v", err)
	}

	inC := map[string]bool{}
	for _, m := range regexp.MustCompile(`dlsym\(RTLD_DEFAULT,\s*"([A-Za-z_][A-Za-z0-9_]*)"\)`).
		FindAllStringSubmatch(string(b), -1) {
		inC[m[1]] = true
	}
	if len(inC) == 0 {
		t.Fatal("no dlsym calls found in the preamble; this test would pass vacuously")
	}

	inGo := map[string]bool{}
	for _, name := range dcsPrivateSymbols {
		inGo[name] = true
	}
	if len(inGo) == 0 {
		t.Fatal("dcsPrivateSymbols is empty; this test would pass vacuously")
	}

	for name := range inC {
		if !inGo[name] {
			t.Errorf("the C resolver dlsyms %q, which dcsPrivateSymbols does not list — the "+
				"conformance check walks the Go list, so this symbol is verified by nothing", name)
		}
	}
	for name := range inGo {
		if !inC[name] {
			t.Errorf("dcsPrivateSymbols lists %q, which the C resolver does not dlsym — the "+
				"conformance check would report on a symbol the code does not need", name)
		}
	}
}
