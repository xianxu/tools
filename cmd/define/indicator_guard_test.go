package main_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// nonScreenFiles are the files whose playback writes to a REAL stdout, and they
// are the exemptions — everything else in package main is in scope.
//
// AN ALLOWLIST, INVERTED FROM THE FIRST DRAFT, and the inversion is the whole
// point (#44 BR-6). That draft named the two screen-hosted files, which made the
// guard's own SCOPE a hand-maintained enumeration — the very shape this issue
// exists to argue against, one level up: a third screen-hosted file would simply
// not be checked, silently. This package's purity guard is an allowlist for the
// same reason, and it is the precedent this cites.
//
// So a new file defaults INTO the rule. Exempting one is a deliberate edit here,
// with a reason:
//
//   - main.go — the one-shot path (`define <word>`), writing to the real stdout,
//     where "\n" genuinely is cursor movement.
//   - repl.go — the piped loop, where the indicator is a RECORD rather than
//     ephemeral UI and is written as a plain line with no escape at all.
var nonScreenFiles = map[string]string{
	"main.go": "the one-shot path writes to a real stdout",
	"repl.go": "the piped loop keeps the indicator as a record",
}

// EVERY indicator ARGUMENT in a screen-hosted file is screenIndicator() (#44).
//
// Inside an append-only buffer a newline is CONTENT, not cursor movement:
// `screen.Write` splits on the erase gesture and `eraseOpenLine` takes back only
// the line the indicator itself was writing, so an `ind.before` of "\n" is
// already a completed line — scrollback — by the time the erase arrives. One row
// leaks per playback.
//
// BY ARGUMENT TYPE, NOT BY CALLEE NAME, and that distinction is the whole value
// of this guard. The obvious version — "every call to playAnnounced passes
// screenIndicator()" — was written first and was blind to exactly the site the
// issue was reported from: the sitting's click goes through `playRegion`, a
// forwarder, so a walk keyed on the callee never reached it. Naming a callee
// enumerates instances; naming the type states the class.
//
// The types are resolved WITHIN THE PACKAGE, by reading each callee's own
// declaration, rather than through go/types. Every indicator-taking function
// lives in package main, so a package-local resolution is complete for the rule
// as stated — and it stays a test with no build-tag and no toolchain dependency.
// A callee from another package taking an indicator would be invisible here,
// which is a real limit and is why the type is declared in this package.
func TestEveryScreenPlaybackTakesTheScreenIndicator(t *testing.T) {
	root := repoRoot(t)
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, filepath.Join(root, "cmd", "define"), nil, 0)
	if err != nil {
		t.Fatalf("parsing cmd/define: %v", err)
	}
	var files map[string]*ast.File
	for name, pkg := range pkgs {
		if name == "main" {
			files = pkg.Files
		}
	}
	if len(files) == 0 {
		t.Fatal("package main parsed to no files; this guard would certify nothing")
	}

	// Which parameter positions of which functions are typed `indicator`.
	takesIndicator := map[string][]int{}
	for _, f := range files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Type.Params == nil {
				continue
			}
			pos := 0
			for _, field := range fn.Type.Params.List {
				n := max(len(field.Names), 1)
				if id, ok := field.Type.(*ast.Ident); ok && id.Name == "indicator" {
					for i := range n {
						takesIndicator[fn.Name.Name] = append(takesIndicator[fn.Name.Name], pos+i)
					}
				}
				pos += n
			}
		}
	}
	if len(takesIndicator) == 0 {
		t.Fatal("no function in package main takes an indicator; this guard would certify nothing")
	}

	checked, scanned := 0, 0
	for path, f := range files {
		base := filepath.Base(path)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		if why, exempt := nonScreenFiles[base]; exempt {
			t.Logf("skipping %s: %s", base, why)
			continue
		}
		scanned++
		// A function that itself takes an indicator may forward its own
		// parameter — the argument is then a bare identifier and the obligation
		// belongs to whoever supplied it. No such function survives in these
		// files today; the arm states the class rather than waiting for one.
		var enclosingTakesOne bool
		ast.Inspect(f, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				enclosingTakesOne = fn.Name != nil && len(takesIndicator[fn.Name.Name]) > 0
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			for _, i := range takesIndicator[callee.Name] {
				if i >= len(call.Args) {
					continue // variadic or a compile error; not this guard's business
				}
				checked++
				arg := call.Args[i]
				if inner, ok := arg.(*ast.CallExpr); ok {
					if id, ok := inner.Fun.(*ast.Ident); ok && id.Name == "screenIndicator" {
						continue
					}
				}
				if _, ok := arg.(*ast.Ident); ok && enclosingTakesOne {
					continue
				}
				t.Errorf("%s: %s is handed an indicator that is not screenIndicator().\n"+
					"This file writes into a screen, where a newline is CONTENT: any "+
					"`before` is committed to the append-only buffer and the erase "+
					"cannot take it back, so the frame drifts up a row per playback.",
					fset.Position(arg.Pos()), callee.Name)
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("every file in package main was exempted; this guard would certify nothing")
	}
	if checked == 0 {
		t.Fatalf("scanned %d files and found no indicator argument in any of them — "+
			"either the exemptions have swallowed the rule, or `indicator` moved and "+
			"this guard is now checking a name nothing uses", scanned)
	}
	// The exemptions must still NAME REAL FILES. A renamed or deleted exemption is
	// a rule quietly widened or a scope quietly narrowed, and either should be a
	// deliberate edit rather than a silent one.
	for base := range nonScreenFiles {
		if _, ok := files[filepath.Join(root, "cmd", "define", base)]; !ok {
			t.Errorf("nonScreenFiles exempts %s, which is not in package main — "+
				"if it moved, move the exemption with it", base)
		}
	}
}
