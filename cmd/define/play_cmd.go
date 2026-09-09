package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/tools/cmd/define/play"
)

// runPlayCommand is `/play`: today's sitting, from the definition prompt.
//
// IT RECORDS, THE LOOP PERFORMS — `cc.replay`'s shape one size up, and
// `replraw.go`'s comment states the rule for both: *"a command decides WHAT to
// replay and the loop owns replaying, so /pron and a bare Enter remain one
// path."* The loop owns running a sitting, so `--play` and `/play` remain one
// path.
//
// Performing it here would nest a full-screen loop inside a command's lifetime,
// while the keys channel, the console and the interrupter are all locals of
// runEditor.
func runPlayCommand(c commandCtx, args []string) int {
	if len(args) > 0 {
		// The rule --play and --stats already state: a mode plus a word is two
		// commands on one line, and /play reviews what is due rather than a
		// subject you name.
		fmt.Fprintf(c.stderr, "define: /play reviews the words due today; it takes no arguments, not %q\n",
			strings.Join(args, " "))
		return 2
	}
	// TWO REFUSALS, because there are two different reasons a sitting cannot
	// happen and they are not the same question.
	//
	// The capability answers "can this terminal host one" — nil on the one-shot
	// path, in a pipe, and in the line-mode REPL, which has no raw terminal.
	// That is the rule setTimes states: a nil capability IS the refusal.
	if c.startSitting == nil {
		fmt.Fprintln(c.stderr, "define: /play needs a terminal; it draws a full screen")
		return 1
	}
	// The deck answers "is there anything to review" — and todaysQuestions
	// dereferences it, so a sitting without one panics rather than degrading.
	if c.deck == nil {
		fmt.Fprintln(c.stderr, noDeckMessage(c.noCapture))
		return 1
	}
	c.startSitting()
	return 0
}

// sittingInPlace runs one sitting on a terminal the caller ALREADY holds.
//
// The sibling is replayInPlace, and the name is deliberate: same position in the
// loop, same "on the terminal we already hold" contract.
//
// IT BORROWS RATHER THAN TAKES, which is the whole of its difference from
// runPlay. newConsole acquires three things a borrower must not:
//
//   - `finish: onceHandBack(live, sess, stdout)`, which restores the SHARED
//     session and prints the transcript to a cooked terminal — an order whose
//     precondition a no-op restorer would silently break;
//   - a second `watchResize` goroutine, so SIGWINCH would be delivered to two
//     channels and the suspended screen would act on it;
//   - `sess.enterMouse()`, already reported.
//
// So the console is assembled here instead: a pinned screen over the same tty,
// the caller's resize channel borrowed, and a finish that hands the summary UP
// into the caller's buffer rather than out to a terminal it does not own.
func sittingInPlace(ctx context.Context, d deps, opt options, keys <-chan Key,
	interrupts *interrupter, repl *liveScreen, resizes <-chan winSize,
	tty io.Writer, stderr io.Writer) int {

	questions, held, code := todaysQuestions(d, opt, repl, stderr)
	if code != 0 || len(questions) == 0 {
		// todaysQuestions has already said why — emptyQueueReason writes the
		// sentence into the screen the loop is showing, so the learner reads it
		// at the prompt rather than in a screen that flashes and vanishes.
		return code
	}

	// THE EDITOR'S SCREEN GOES QUIET for the duration. Two screens over one
	// terminal, and this one has a throttled painter that fires on its own
	// goroutine; without this its pending frame lands inside the sitting's.
	repl.suspend()
	rows, cols := repl.Size()
	sitting := newPinnedScreen(tty, rows, cols)
	defer func() {
		// THE SHAPE IS HANDED BACK, and this is not tidiness (#48 BR-5).
		//
		// The resize channel is BORROWED, so a SIGWINCH during the sitting is
		// consumed by the sitting's loop and applied to the sitting's screen.
		// The editor's screen never sees it — nothing else is reading that
		// channel — so without this it repaints at the shape the terminal had
		// before the sitting started, for the rest of the session. Too tall and
		// the terminal scrolls; too narrow and the prompt is laid out against a
		// width that is not there.
		//
		// resume() repaints, which is why the shape must be set BEFORE it: a
		// repaint at a stale size is exactly the frame this exists to prevent.
		// The plan claimed resume "takes the new shape" on its own; it does not,
		// and the review measured that.
		repl.Resize(sitting.Size())
		repl.resume()
	}()

	// CTRL-C ENDS THE SITTING, NOT THE PROGRAM. interrupter.Set exists for
	// exactly this — #16 built it so a streaming answer could own the interrupt
	// and hand it back — and the deferred restore is what makes "hand it back"
	// true on every exit path.
	sctx, cancel := context.WithCancel(ctx)
	defer cancel()
	restore := interrupts.Set(cancel)
	defer restore()

	con := console{
		view:    sitting,
		resizes: resizes,
		// The summary goes UP, not out. handBack would restore the shared
		// session and print to a cooked terminal; the loop still owns that
		// terminal, so the sitting's transcript is written into the editor's
		// buffer instead — where it is in the scrollback when the prompt comes
		// back, which is what --play achieves by printing on exit.
		finish: func() {
			sitting.Stop()
			fmt.Fprint(repl, sitting.Transcript())
		},
		stdout: sitting, stderr: sitting,
	}
	return playSession(sctx, d, opt, play.NewSession(questions), held, keys, con)
}
