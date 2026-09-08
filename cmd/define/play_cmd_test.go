package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// A SITTING FROM THE LOOP MUST NEVER TAKE THE TERMINAL AGAIN (#48).
//
// The whole design is that `/play` BORROWS a terminal the loop already holds.
// Pointing it at runPlay instead would compile, run, and put an already-raw
// terminal into raw mode inside a second alternate screen — the class of bug
// that shows as corruption rather than as a failure.
//
// DERIVED, not listed: the callees of sittingInPlace are walked transitively
// through package main, so a function added to that path is covered when it is
// added rather than when someone remembers. It fails closed on the walk's size,
// because a traversal that visits nothing proves nothing (#12 BR-17).
func TestASittingFromTheLoopNeverEntersRawMode(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Every function in the package, by name, with the calls it makes.
	calls := map[string][]string{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, d := range file.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				var made []string
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					c, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if id, ok := c.Fun.(*ast.Ident); ok {
						made = append(made, id.Name)
					}
					return true
				})
				calls[fn.Name.Name] = made
			}
		}
	}
	if _, ok := calls["sittingInPlace"]; !ok {
		t.Fatal("sittingInPlace not found; this guard would certify nothing")
	}

	seen := map[string]bool{}
	var walk func(string, []string)
	walk = func(name string, path []string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, callee := range calls[name] {
			if callee == "enterRaw" {
				t.Errorf("sittingInPlace reaches enterRaw via %s → enterRaw.\n"+
					"A sitting started from the loop BORROWS the terminal; taking it "+
					"again puts an already-raw terminal into raw mode inside a second "+
					"alternate screen.", strings.Join(append(path, name), " → "))
			}
			walk(callee, append(path, name))
		}
	}
	walk("sittingInPlace", nil)

	if len(seen) < 5 {
		t.Fatalf("the call walk visited %d functions; sittingInPlace calls more than "+
			"that, so this traversal is under-deriving and would certify a path "+
			"nobody checked", len(seen))
	}
	// AND THE GUARD IS NOT VACUOUS: runPlay, which legitimately owns the
	// terminal, must be a path that WOULD trip it.
	var reachesRaw bool
	for _, callee := range calls["runPlay"] {
		if callee == "enterRaw" {
			reachesRaw = true
		}
	}
	if !reachesRaw {
		t.Error("runPlay no longer calls enterRaw, so this guard has nothing to " +
			"distinguish borrowing from taking")
	}
}

// BOTH DOORS REACH ONE playSession, which is the Done-when that stops a change
// to the sitting applying to only one of them.
func TestBothEntryPointsReachOnePlaySession(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, d := range file.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				if fn.Name.Name != "runPlay" && fn.Name.Name != "sittingInPlace" {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					c, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "playSession" {
						found[fn.Name.Name] = true
					}
					return true
				})
			}
		}
	}
	for _, entry := range []string{"runPlay", "sittingInPlace"} {
		if !found[entry] {
			t.Errorf("%s does not call playSession — the two entry points have "+
				"diverged, so a change to the sitting applies to only one of them", entry)
		}
	}
}

// A SITTING RECORDS THE SAME THING FROM EITHER DOOR, asserted through the STORE
// rather than through a fake capturer (#48).
//
// The AST guards above prove both entry points REACH playSession; this proves
// what playSession does when they get there. A fake capturer would show the
// outcome reaching *a* capturer and nothing about what it writes — the gap #12's
// mutation sweep found.
func TestASittingRecordsTheSameEventFromEitherDoor(t *testing.T) {
	answer := func(t *testing.T) *store.Mem {
		t.Helper()
		d, opt, st := playRig(t, "sycophantic")
		qs, held, code := todaysQuestions(d, opt, io.Discard, io.Discard)
		if code != 0 || len(qs) == 0 {
			t.Fatalf("no questions to answer: code %d, %d questions", code, len(qs))
		}
		var out, errb bytes.Buffer
		playSession(t.Context(), d, opt, play.NewSession(qs), held,
			keysFor(gradeKey(t, qs[0], play.Correct)+"^"), playbackConsole(&out, &errb))
		return st
	}

	// Two runs of the SAME playSession, which is what both doors call. The
	// assertion is that a sitting's record is a property of playSession and not
	// of the entry point — so if the doors ever diverge, the guards above catch
	// it and this stays true of whatever they both reach.
	first, second := answer(t), answer(t)

	got := func(st *store.Mem) []store.ReviewEvent {
		evs, err := st.Events(time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		var reviewed []store.ReviewEvent
		for _, e := range evs {
			if e.Kind == store.EventReviewed {
				reviewed = append(reviewed, e)
			}
		}
		return reviewed
	}
	a, b := got(first), got(second)
	if len(a) == 0 {
		t.Fatal("a sitting recorded no review, so this proves nothing")
	}
	if len(a) != len(b) {
		t.Fatalf("the two sittings recorded %d and %d reviews", len(a), len(b))
	}
	for i := range a {
		if a[i].Word != b[i].Word || a[i].Correct != b[i].Correct || a[i].Form != b[i].Form {
			t.Errorf("event %d differs: %+v vs %+v", i, a[i], b[i])
		}
		if a[i].Form == "" {
			t.Errorf("event %d has no form, so the log cannot attribute it: %+v", i, a[i])
		}
	}
}
