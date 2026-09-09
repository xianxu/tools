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
		done <- sittingInPlace(t.Context(), d, opt, keys, &interrupter{}, repl, resizes, &tty, &errb)
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

// THE SITTING HANDS THE INTERRUPT BACK (#48 BR-7).
//
// interrupter.Set scopes Ctrl-C to the sitting and restore() returns it to the
// loop. In the real path readKeys fires the interrupter and WITHHOLDS the key
// when a scope consumed it (rawterm.go:104-110), so the sitting's context
// cancels and the loop's does not — that end-to-end behaviour is what the
// operator's pty run exercised.
//
// What is deterministic in-process, and what can silently break, is the RESTORE:
// a Set without its restore leaves the sitting's cancel installed after the
// sitting is gone, so the next Ctrl-C at the prompt cancels a dead context and
// the loop never quits. This drives the real door and asserts both halves —
// the loop's cancel untouched during, and working after.
//
// Two earlier versions of this test were wrong in ways worth recording: one fed
// Key{KeyInterrupt} straight into the channel, which bypasses the interrupter
// entirely and passed with Set/restore deleted; the other waited on HasScope,
// which this test had already made true by installing the loop's own cancel.
func TestASittingHandsTheInterruptBack(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")

	var loopCancelled bool
	interrupts := &interrupter{}
	interrupts.Set(func() { loopCancelled = true })

	var tty, errb bytes.Buffer
	repl := newLiveScreen(&tty, 24, 80)
	sittingInPlace(t.Context(), d, opt, keysFor("^"), interrupts, repl, nil, &tty, &errb)

	if loopCancelled {
		t.Error("the loop's cancel ran during the sitting — Ctrl-C would quit define " +
			"instead of ending the review")
	}
	// RESTORED: the loop's own interrupt works again, and is the one that fires.
	if !interrupts.Fire() {
		t.Fatal("nothing consumed the interrupt after the sitting; the sink was left empty")
	}
	if !loopCancelled {
		t.Error("the interrupt was not handed back to the loop — the next Ctrl-C at " +
			"the prompt would cancel the sitting's dead context and never quit")
	}
}

// THE THREE REFUSALS, which the plan's Chunk 1 named and the first build never
// wrote (#48 BR-6).
func TestSlashPlayRefusals(t *testing.T) {
	live := func() commandCtx {
		return commandCtx{deck: store.NewMem(), startSitting: func() {}, stderr: io.Discard}
	}

	t.Run("no terminal is a refusal", func(t *testing.T) {
		// A nil capability IS the refusal — the rule setTimes states. One check
		// covers the one-shot path, a pipe, and the line-mode REPL, which has no
		// raw terminal at all.
		var errb bytes.Buffer
		c := live()
		c.startSitting, c.stderr = nil, &errb
		if code := runPlayCommand(c, nil); code != 1 {
			t.Errorf("exit = %d, want 1", code)
		}
		if !strings.Contains(errb.String(), "terminal") {
			t.Errorf("the refusal does not name the cause: %q", errb.String())
		}
	})

	t.Run("no deck is a SECOND, separate refusal", func(t *testing.T) {
		// todaysQuestions dereferences the deck, so a sitting without one panics
		// rather than degrading. The capability says the terminal can host a
		// sitting and nothing about there being one to review.
		var errb bytes.Buffer
		c := live()
		c.deck, c.stderr, c.noCapture = nil, &errb, true
		if code := runPlayCommand(c, nil); code != 1 {
			t.Errorf("exit = %d, want 1", code)
		}
		if !strings.Contains(errb.String(), "DEFINE_NO_CAPTURE") {
			t.Errorf("the refusal does not name which cause: %q", errb.String())
		}
	})

	t.Run("an argument is a usage error", func(t *testing.T) {
		var errb bytes.Buffer
		c := live()
		c.stderr = &errb
		if code := runPlayCommand(c, []string{"sycophantic"}); code != 2 {
			t.Errorf("exit = %d, want 2", code)
		}
		if !strings.Contains(errb.String(), "/play") {
			t.Errorf("the refusal does not name the command: %q", errb.String())
		}
	})

	t.Run("otherwise it RECORDS rather than performs", func(t *testing.T) {
		var started bool
		c := live()
		c.startSitting = func() { started = true }
		if code := runPlayCommand(c, nil); code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
		if !started {
			t.Error("the command did not record the intent, so the loop has nothing to do")
		}
	})
}
