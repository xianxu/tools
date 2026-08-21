package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

// replRaw is the interactive loop when we own the terminal: keystrokes in, a
// rendered frame out.
//
// The whole editing model is Apply + RenderLine, both pure — this function is
// the thin shell that reads keys, draws frames, and dispatches lookups.
func replRaw(ctx context.Context, cancel context.CancelFunc, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
	f, ok := stdin.(*os.File)
	if !ok {
		// Not a real terminal handle (a test harness, a wrapper): fall back to the
		// line path rather than pretending raw mode succeeded.
		return replLines(ctx, d, opt, stdin, stdout, stderr, false /*stdin is a tty*/, true /*we own it*/)
	}
	sess, err := enterRaw(f)
	if err != nil {
		fmt.Fprintf(stderr, "define: cannot enter raw mode (%v); falling back to line input\n", err)
		return replLines(ctx, d, opt, stdin, stdout, stderr, false /*stdin is a tty*/, true /*we own it*/)
	}
	defer sess.restore()

	keys := readKeys(ctx, f, cancel)
	// cooked drops raw mode around a lookup so the definition scrolls normally,
	// then re-enters for the next frame.
	// Dropping and re-entering raw mode around a lookup. If re-entry fails the
	// editor would keep drawing frames a cooked terminal echoes over — silently
	// unusable — so say so and stop rather than swallow it.
	cooked := func(run func()) error {
		sess.restore()
		run()
		s, err := enterRaw(f)
		if err != nil {
			return err
		}
		*sess = *s
		return nil
	}
	return runEditor(ctx, keys, d, opt, cooked, sess.restore, stdout, stderr)
}

// runEditor is the editor loop with the terminal factored out: keys arrive on a
// channel, `cooked` runs a lookup outside raw mode, and `finish` restores the
// terminal. Tests drive it with a scripted channel and no terminal at all —
// which is the whole point of keeping Apply and RenderLine pure.
func runEditor(ctx context.Context, keys <-chan Key, d deps, opt options,
	cooked func(func()) error, finish func(), stdout, stderr io.Writer) int {

	// Same wiring as replLines: cached behind the seam, in the function that uses
	// it, so a test driving runEditor gets exactly what production gets.
	d.audio = newCachingAudioSource(d.audio)
	// The seam is filled at the boundary (#3). The loop never decides where
	// history lives — which is what let persistence land without the editor
	// changing at all.
	hist := d.history
	if hist == nil {
		hist = &memHistory{}
	}
	e := NewEditor()
	var current string
	// Resolve candidates ONCE per keystroke and use the same slice for both the
	// state machine and the suggestion. Querying twice doubled the work the
	// History seam will do once #3 backs it with a store.
	// The command menu is a dropdown, not scrollback: drawn BELOW the prompt
	// line and erased on every redraw. menuDrawn is how many rows are currently
	// on screen under the cursor.
	//
	// Known limit, shared with the erase arithmetic elsewhere in this file: if
	// the menu does not fit below the cursor the terminal scrolls, and the
	// cursor-up count then lands a row off. It self-corrects on the next
	// keystroke, because the prompt line is fully rewritten each time.
	menuDrawn := 0
	paintMenu := func(lines []string) {
		// Erase max(previous, new) rows, so a list that SHRINKS as you type
		// leaves nothing of the longer one behind.
		n := menuDrawn
		if len(lines) > n {
			n = len(lines)
		}
		if n == 0 {
			return
		}
		for i := 0; i < n; i++ {
			fmt.Fprint(stdout, "\r\n"+eraseLine)
			if i < len(lines) {
				fmt.Fprint(stdout, lines[i])
			}
		}
		fmt.Fprintf(stdout, "\x1b[%dA\r", n) // back up to the prompt line
		menuDrawn = len(lines)
	}
	clearMenu := func() { paintMenu(nil) }
	draw := func(matches []string) {
		// The menu is painted FIRST and the prompt line last, so RenderLine
		// leaves the cursor where the user is typing.
		paintMenu(menuLines(e.String(), commands, opt.width))
		fmt.Fprint(stdout, RenderLine(e, Suggestion(e, matches), opt.color))
	}
	draw(completionsFor(e.WalkBase(), hist, commands))

	for {
		select {
		case <-ctx.Done():
			finish() // before anything else can write: never exit leaving raw mode on
			fmt.Fprintln(stdout)
			return 0
		case k, open := <-keys:
			if !open {
				finish()
				fmt.Fprintln(stdout)
				return 0
			}
			matches := completionsFor(e.WalkBase(), hist, commands)
			var act Action
			e, act = Apply(e, k, matches)
			switch act {
			case ActInterrupt, ActEOF:
				clearMenu()
				finish()
				fmt.Fprintln(stdout)
				return 0
			case ActSubmit:
				// Route through the SAME decision table the line loop uses.
				// Bypassing it meant the interactive path quietly disagreed about
				// what a line means — no trimming, and "hot  dog" not collapsed to
				// the multi-word headword the dictionary actually has (ARCH-DRY).
				cmd := parseREPLLine(e.String(), current != "")
				line := cmd.word
				// Whatever happens next writes below this line, so the dropdown
				// has to go before any of it.
				clearMenu()
				// Redraw the committed line with NO suggestion before advancing:
				// the grey tail was never accepted, so leaving it in scrollback
				// claims the user typed something they did not.
				submitted := e
				e = NewEditor()
				if cmd.kind == cmdDefine || cmd.kind == cmdCommand {
					fmt.Fprint(stdout, RenderLine(submitted, "", opt.color))
				}
				if cmd.kind == cmdCommand {
					// Commands print multiple lines, so they run COOKED for the
					// same reason a definition does — in raw mode "\n" is a line
					// feed with no carriage return.
					fmt.Fprint(stdout, "\r\n")
					hist.Add(submitted.String()) // up-arrow recalls "/history" too
					if err := cooked(func() {
						dispatchCommand(cmd, commands, newCommandCtx(stdout, stderr))
					}); err != nil {
						finish()
						fmt.Fprintf(stderr, "define: lost the terminal: %v\n", err)
						return 1
					}
					fmt.Fprint(stdout, "\r\n")
					draw(completionsFor(e.WalkBase(), hist, commands))
					continue
				}
				if cmd.kind != cmdDefine {
					// cmdReplay and cmdNothing both stay on this line: the
					// indicator is drawn over the prompt, then the prompt back.
					replayInPlace(ctx, d, opt, current, stdout, stderr)
					draw(completionsFor(e.WalkBase(), hist, commands))
					continue
				}
				// In RAW mode "\n" is a line feed only — no carriage return — so
				// the next line would start at the current column. Everything
				// written before we drop back to cooked mode needs "\r\n".
				fmt.Fprint(stdout, "\r\n")
				if err := submitLine(ctx, cooked, d, opt, line, hist, &current, stdout, stderr); err != nil {
					// Raw mode could not be re-entered. The editor would keep
					// drawing frames a cooked terminal echoes over — silently
					// unusable — so stop and say why rather than swallow it.
					finish()
					fmt.Fprintf(stderr, "define: lost the terminal: %v\n", err)
					return 1
				}
				// A blank line between the entry and the next prompt: without it
				// the prompt butts against the last line of the definition and
				// reads as part of it.
				fmt.Fprint(stdout, "\r\n")
				draw(completionsFor(e.WalkBase(), hist, commands))
				continue
			}
			draw(matches)
		}
	}
}

// replayInPlace speaks the current word again without moving the cursor off the
// prompt line. Stays in RAW mode throughout: Ctrl-C must reach the key reader as
// a byte while playback blocks.
func replayInPlace(ctx context.Context, d deps, opt options, current string, stdout, stderr io.Writer) {
	switch {
	case current == "":
		fmt.Fprint(stderr, eraseLine+"define: type a word, or press return to replay the last one\r\n")
	case opt.noAudio || opt.times <= 0:
		fmt.Fprint(stderr, eraseLine+"define: nothing to replay: audio is off\r\n")
	default:
		playAnnounced(ctx, d, opt, current, indicator{show: true, erase: eraseLine}, stdout, stderr)
	}
}

// submitLine handles one submitted word.
func submitLine(ctx context.Context, cooked func(func()) error, d deps, opt options, line string,
	hist History, current *string, stdout, stderr io.Writer) error {

	// Render in COOKED mode so newlines translate, but play in RAW mode so
	// Ctrl-C arrives as a byte the key reader can act on. Playback is the part
	// that blocks for seconds; doing it cooked made cancellation depend on a
	// signal that raw mode exists to replace.
	var code int
	var play bool
	if err := cooked(func() { code, play = lookupAndRender(d, opt, line, stdout, stderr) }); err != nil {
		return err
	}
	if play {
		playAnnounced(ctx, d, opt, line, indicator{show: true, before: "\r\n", erase: eraseLine}, stdout, stderr)
	}
	hist.Add(line)
	if code == 0 {
		*current = line
	}
	return nil
}
