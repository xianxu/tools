package main_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// screenHostedFiles are the files whose playback writes into a `screen` rather
// than into a real stdout.
//
// A LIST, because "is this writer a screen" is not a question any type can
// answer here: `playAnnounced` takes an `io.Writer`, and that is right — the
// one-shot and piped paths hand it the real thing and genuinely want the
// cursor-positioning newline `screenIndicator` refuses.
var screenHostedFiles = []string{"play_loop.go", "replraw.go"}

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

	checked := 0
	for _, want := range screenHostedFiles {
		f := fileNamed(t, fset, files, want)
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
	if checked == 0 {
		t.Fatalf("found no indicator arguments in %v; a guard that checks nothing certifies nothing", screenHostedFiles)
	}
}

// fileNamed finds one parsed file by base name, and FATALS when it is absent —
// never skips. A renamed file must break this guard loudly rather than quietly
// dropping the rule it carried.
func fileNamed(t *testing.T, fset *token.FileSet, files map[string]*ast.File, base string) *ast.File {
	t.Helper()
	for path, f := range files {
		if filepath.Base(path) == base {
			return f
		}
	}
	t.Fatalf("%s is not in package main — if it moved, move it in screenHostedFiles too", base)
	return nil
}
