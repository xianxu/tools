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

// THE SITTING DOOR IS ACTUALLY DRIVEN, and it records what a sitting records.
//
// The first version of this test called playSession twice and asserted the two
// runs matched — a pin that cannot fail, wearing a name that claimed otherwise,
// with a comment rationalising it (#48 BR-8). sittingInPlace takes its terminal
// as parameters, so it can be driven over a buffer with a scripted key channel;
// there was never a reason not to.
//
// Asserted through the STORE rather than a fake capturer: a fake shows the
// outcome reaching *a* capturer and nothing about what it writes, which is the
// gap #12's sweep found.
func TestTheSittingDoorRecordsWhatItAnswers(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	qs, _, code := todaysQuestions(d, opt, io.Discard, io.Discard)
	if code != 0 || len(qs) == 0 {
		t.Fatalf("no questions: code %d, %d questions", code, len(qs))
	}

	var tty, errb bytes.Buffer
	repl := newLiveScreen(&tty, 24, 80)
	keys := keysFor(gradeKey(t, qs[0], play.Correct) + "^")

	sittingInPlace(t.Context(), d, opt, keys, &interrupter{}, repl, nil, &tty, &errb)

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
	if len(reviewed) == 0 {
		t.Fatal("a sitting driven through sittingInPlace recorded no review")
	}
	if reviewed[0].Form == "" {
		t.Errorf("the event has no form, so the log cannot attribute it: %+v", reviewed[0])
	}
	if !reviewed[0].Correct {
		t.Errorf("a correct answer was recorded wrong: %+v", reviewed[0])
	}
}

// THE EDITOR'S SCREEN COMES BACK AT THE TERMINAL'S CURRENT SHAPE (#48 BR-5).
//
// The resize channel is borrowed, so a SIGWINCH during a sitting is consumed by
// the sitting and applied to ITS screen. Without handing the shape back, the
// editor repaints at the pre-sitting size for the rest of the session.
func TestTheEditorScreenTakesTheShapeTheSittingEndedWith(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, _, _ := todaysQuestions(d, opt, io.Discard, io.Discard)
	if len(qs) == 0 {
		t.Fatal("no questions, so this proves nothing")
	}

	var tty, errb bytes.Buffer
	repl := newLiveScreen(&tty, 24, 80)

	// A resize arrives while the sitting owns the terminal, and it must be
	// CONSUMED before the interrupt — otherwise select could take either and the
	// test would pass or fail at random. So the keys channel stays empty until
	// the resize has actually been read, which is the only ordering the sitting
	// can be forced into from outside.
	resizes := make(chan winSize, 1)
	resizes <- winSize{rows: 40, cols: 120}
	keys := make(chan Key)

	done := make(chan int, 1)
	go func() {
		c, _ := sittingInPlace(t.Context(), d, opt, keys, &interrupter{}, repl, resizes, &tty, &errb)
		done <- c
	}()
	for i := 0; len(resizes) > 0; i++ {
		if i > 2000 {
			t.Fatal("the sitting never read the resize")
		}
		time.Sleep(time.Millisecond)
	}
	keys <- Key{Kind: KeyInterrupt}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the sitting did not end on Ctrl-C")
	}

	rows, cols := repl.Size()
	if rows != 40 || cols != 120 {
		t.Errorf("the editor's screen is %dx%d, want 40x120 — a resize the sitting "+
			"consumed was never handed back, so the prompt repaints at a shape the "+
			"terminal no longer has", rows, cols)
	}
}

// CTRL-C ENDS THE SITTING, NOT THE LOOP — both halves (#48 BR-7).
//
// The property has two, and a test asserting only one passes on code that has
// neither. FOUR earlier attempts each passed while wrong: feeding
// Key{KeyInterrupt} into the channel bypasses the interrupter entirely; waiting
// on a HasScope helper this test had itself made true fired before the sitting
// was in the picture; racing two goroutines over one bytes.Buffer hung; and
// asserting only the restore passed with Set AND restore both deleted, because
// never scoping also leaves the loop's cancel installed.
//
// So: fire the interrupt WHILE the sitting owns it, ordered deterministically by
// making the sitting consume a resize first — a channel read the test can
// observe, unlike a buffer it must not race.
func TestASittingScopesTheInterruptAndHandsItBack(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")

	var loopCancelled bool
	interrupts := &interrupter{}
	interrupts.Set(func() { loopCancelled = true })

	var tty, errb bytes.Buffer
	repl := newLiveScreen(&tty, 24, 80)
	resizes := make(chan winSize, 1)
	resizes <- winSize{rows: 30, cols: 100}
	keys := make(chan Key)

	done := make(chan struct{})
	go func() {
		sittingInPlace(t.Context(), d, opt, keys, interrupts, repl, resizes, &tty, &errb)
		close(done)
	}()

	// The sitting has read the resize, so it is inside playSession with its own
	// cancel installed.
	deadline := time.Now().Add(5 * time.Second)
	for len(resizes) > 0 {
		if time.Now().After(deadline) {
			t.Fatal("the sitting never read the resize, so it never took the interrupt over")
		}
		time.Sleep(time.Millisecond)
	}

	// HALF ONE: the interrupt reaches the SITTING. In the real path readKeys
	// fires the sink and withholds the key when a scope consumed it
	// (rawterm.go:104-110), which is what ends the sitting and leaves the loop
	// alone.
	if !interrupts.Fire() {
		t.Fatal("nothing consumed the interrupt; the sitting installed no scope")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		close(keys)
		t.Fatal("the sitting did not end when its own interrupt fired")
	}
	if loopCancelled {
		t.Error("the interrupt reached the LOOP's cancel — Ctrl-C would quit define " +
			"instead of ending the review")
	}

	// HALF TWO: it is handed back. A Set without its restore leaves a dead
	// context installed, and the next Ctrl-C at the prompt never quits.
	if !interrupts.Fire() || !loopCancelled {
		t.Error("the interrupt was not restored to the loop after the sitting")
	}
}

// A SHAPE IS APPLIED IN ONE PLACE, BY BOTH ROUTES (#48 BR-14).
//
// A terminal shape now arrives twice: from the loop's own resize case, and
// handed back when a sitting ends — the sitting CONSUMED the resize, because the
// channel is borrowed and whoever reads it takes the value. The first fix handed
// back the screen's shape and not opt.width, so an entry looked up after a
// sitting wrapped at the pre-sitting width: the instance fixed, the class not.
//
// So the guard is that both routes go through applyShape, derived from the
// source rather than listed — a third route added later is covered when it is
// added.
func TestBothShapeRoutesGoThroughOnePlace(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "replraw.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var applyCalls, bareResizes int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		// applyShape's OWN Resize is the one legitimate call — it is the place
		// everything else is required to go through.
		inApplyShape := fn.Name.Name == "applyShape"
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch f := call.Fun.(type) {
			case *ast.Ident:
				if f.Name == "applyShape" {
					applyCalls++
				}
			case *ast.SelectorExpr:
				// view.Resize(...) elsewhere is a second place that knows what a
				// shape means, and it is the one that forgets opt.width.
				if f.Sel.Name == "Resize" && !inApplyShape {
					bareResizes++
				}
			}
			return true
		})
	}
	if applyCalls < 2 {
		t.Errorf("applyShape is called %d times in replraw.go; both the resize case "+
			"and the post-sitting hand-back must use it, or opt.width is set on one "+
			"route and not the other", applyCalls)
	}
	if bareResizes != 0 {
		t.Errorf("replraw.go calls view.Resize directly %d time(s) — a shape applied "+
			"outside applyShape is a shape whose width policy was forgotten", bareResizes)
	}
}

// AND THE POLICY ITSELF: below the wrap floor, wrapping is OFF while the frame
// still fits the columns that exist.
func TestApplyShapeSetsBothWidths(t *testing.T) {
	var tty bytes.Buffer
	view := newLiveScreen(&tty, 24, 80)

	opt := options{width: 80}
	applyShape(&opt, view, winSize{rows: 40, cols: 120})
	if opt.width != 120 {
		t.Errorf("opt.width = %d, want 120", opt.width)
	}
	if r, c := view.Size(); r != 40 || c != 120 {
		t.Errorf("view is %dx%d, want 40x120", r, c)
	}

	applyShape(&opt, view, winSize{rows: 40, cols: minWrapWidth - 1})
	if opt.width != 0 {
		t.Errorf("opt.width = %d, want 0 — below the floor an entry cannot be broken "+
			"and stay readable", opt.width)
	}
	if _, c := view.Size(); c != minWrapWidth-1 {
		t.Errorf("view cols = %d, want %d — the frame still has to fit the columns "+
			"that exist", c, minWrapWidth-1)
	}
}
