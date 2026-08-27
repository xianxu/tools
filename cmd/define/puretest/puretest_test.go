package puretest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/puretest"
)

// recorder stands in for *testing.T so a guard can be run against a KNOWN-BAD
// package and asserted to FAIL.
//
// This file is the point. #5 verified its purity guards by mutating the tree in
// a scratch copy and deleting it; #6 M1 did the same for the extracted version
// and then recorded that the negative cases were "verified in the tree", which
// was false — the package had no tests at all. A guard nothing has ever seen
// fail is indistinguishable from a guard that cannot fail.
type recorder struct{ msgs []string }

func (r *recorder) Helper() {}
func (r *recorder) Errorf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
}

// Fatalf panics so control leaves the guard the way it would under testing.T,
// and runGuard recovers it.
func (r *recorder) Fatalf(format string, args ...any) {
	r.msgs = append(r.msgs, fmt.Sprintf(format, args...))
	panic(errFatal)
}

const errFatal = "puretest: fatal"

func runGuard(fn func(t puretest.T)) []string {
	r := &recorder{}
	func() {
		defer func() {
			if v := recover(); v != nil && v != errFatal {
				panic(v)
			}
		}()
		fn(r)
	}()
	return r.msgs
}

func failed(msgs []string, want string) bool {
	for _, m := range msgs {
		if strings.Contains(m, want) {
			return true
		}
	}
	return false
}

const (
	impure = "github.com/xianxu/tools/cmd/define/puretest/testdata/impure"
	clocky = "github.com/xianxu/tools/cmd/define/puretest/testdata/clocky"
	// A dedicated known-GOOD fixture. Pointing this at cmd/define/play would
	// make a change to production code silently change what these tests assert;
	// a guard's positive case has to be as fixed as its negative one.
	pure = "github.com/xianxu/tools/cmd/define/puretest/testdata/pure"
	// The same argument, applied to the OTHER known-good case: zero imports.
	// It was pinned only by cmd/define/play importing nothing today (BR-41).
	nothing = "github.com/xianxu/tools/cmd/define/puretest/testdata/nothing"
)

func TestImportsOnlyRejectsAnIOImport(t *testing.T) {
	msgs := runGuard(func(tt puretest.T) {
		puretest.ImportsOnly(tt, impure, []string{"github.com/xianxu/tools/cmd/define/store"})
	})

	if !failed(msgs, `imports "os"`) {
		t.Errorf("the guard did not reject an os import: %v", msgs)
	}
}

func TestImportsOnlyAcceptsAPurePackage(t *testing.T) {
	if msgs := runGuard(func(tt puretest.T) {
		puretest.ImportsOnly(tt, pure, []string{"sort"})
	}); len(msgs) != 0 {
		t.Errorf("a pure package was rejected: %v", msgs)
	}
}

// The hazard an import list structurally cannot see: `time` is allowed, and
// time.Since reads the same clock as time.Now.
func TestNoWallClockRejectsTimeSince(t *testing.T) {
	msgs := runGuard(func(tt puretest.T) { puretest.NoWallClock(tt, clocky) })

	if !failed(msgs, "time.Since(") {
		t.Errorf("the guard did not reject time.Since: %v", msgs)
	}
	// ...and the import guard alone would have let it through, which is why
	// there are two guards rather than one.
	if msgs := runGuard(func(tt puretest.T) {
		puretest.ImportsOnly(tt, clocky, []string{"time"})
	}); len(msgs) != 0 {
		t.Errorf("the import guard was expected to PASS this package: %v", msgs)
	}
}

func TestNoWallClockAcceptsAPurePackage(t *testing.T) {
	if msgs := runGuard(func(tt puretest.T) { puretest.NoWallClock(tt, pure) }); len(msgs) != 0 {
		t.Errorf("a clock-free package was rejected: %v", msgs)
	}
}

// The hole the import allowlist leaves: allowlisting `store` grants the disk
// with it.
func TestStoreSymbolsOnlyRejectsADiskConstructor(t *testing.T) {
	msgs := runGuard(func(tt puretest.T) {
		puretest.StoreSymbolsOnly(tt, impure, []string{"Store"})
	})

	if !failed(msgs, "store.NewYAML") {
		t.Errorf("the guard did not reject store.NewYAML: %v", msgs)
	}
}

// Every guard must FATAL rather than pass when it finds nothing to check —
// otherwise pointing one at the wrong package reports success.
func TestGuardsRefuseToPassVacuously(t *testing.T) {
	const nowhere = "github.com/xianxu/tools/cmd/define/puretest/testdata/nosuchpackage"

	for _, tc := range []struct {
		name string
		run  func(t puretest.T)
	}{
		{"ImportsOnly", func(tt puretest.T) { puretest.ImportsOnly(tt, nowhere, nil) }},
		{"NoWallClock", func(tt puretest.T) { puretest.NoWallClock(tt, nowhere) }},
		{"StoreSymbolsOnly", func(tt puretest.T) { puretest.StoreSymbolsOnly(tt, nowhere, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if msgs := runGuard(tc.run); len(msgs) == 0 {
				t.Error("the guard passed against a package that does not exist")
			}
		})
	}

	// StoreSymbolsOnly on a package that names no store symbol at all: passing
	// would mean "no violations found" when the truth is "nothing was examined".
	if msgs := runGuard(func(tt puretest.T) {
		puretest.StoreSymbolsOnly(tt, clocky, []string{"Key"})
	}); !failed(msgs, "no store references") {
		t.Errorf("StoreSymbolsOnly passed vacuously on a package with no store references: %v", msgs)
	}
}

// The failed command's own words must survive: exec.Cmd.Output puts them on
// ExitError.Stderr, and discarding it turns "package does not exist" into a bare
// "exit status 1". Second in the discarded-error family this window.
func TestGuardFailureNamesTheUnderlyingError(t *testing.T) {
	msgs := runGuard(func(tt puretest.T) {
		puretest.ImportsOnly(tt, "github.com/xianxu/tools/cmd/define/puretest/testdata/nosuchpackage", nil)
	})

	if !failed(msgs, "no required module") && !failed(msgs, "is not in std") && !failed(msgs, "cannot find") {
		t.Errorf("the failure did not carry go list's own words: %v", msgs)
	}
}

// ZERO imports is a PASS, and it is pinned HERE rather than incidentally by
// whichever production package happens to import nothing this week.
//
// ImportsOnly's doc comment argues the case at length — a package needing
// nothing at all is the STRONGEST form of the claim, not a failed measurement —
// and until now the only thing standing behind that argument was cmd/define/play
// importing nothing today. #7 adds a form package that will likely give play its
// first import, and the zero-import path would then have gone quiet with nothing
// to say so (BR-41).
func TestImportsOnlyAcceptsAPackageWithNoImports(t *testing.T) {
	msgs := runGuard(func(t puretest.T) {
		puretest.ImportsOnly(t, nothing, []string{"sort"})
	})
	if len(msgs) != 0 {
		t.Errorf("zero imports must PASS, got %v", msgs)
	}
}
