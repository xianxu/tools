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
func replRaw(ctx context.Context, interrupts *interrupter, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
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

	keys := readKeys(ctx, f, interrupts)
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
	// The RAW editor is the only thing that recalls, so it is the only thing
	// that pays for reading the log. replLines never touches history at all.
	hist.Load()
	e := NewEditor()
	var sess session
	// Apply gets the candidate list computed BEFORE the keystroke, which is
	// correct: it is deciding what to do with that keystroke given the line as
	// it stands, and a history walk anchors on it. draw() computes its own from
	// the line AFTER — passing it this one is what made the grey tail disagree
	// with what Tab accepted.
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
	// draw takes NO match list on purpose. It used to accept one, and one caller
	// passed the list computed BEFORE the keystroke was applied — so the grey
	// tail was rendered against the previous line. Both lists were history until
	// #15 and a stale superset usually had the same first match, so nothing
	// showed; command mode made the stale list come from a different NAMESPACE
	// and typing "/" suggested "/history" out of recall while the menu under it
	// listed commands and Tab accepted "/help".
	//
	// Computing here means there is one answer to "what does the current line
	// match", and no way to hand this function a stale one.
	draw := func() {
		// The menu is painted FIRST and the prompt line last, so RenderLine
		// leaves the cursor where the user is typing.
		paintMenu(menuLines(e.String(), commands, opt.width))
		fmt.Fprint(stdout, RenderLine(e, Suggestion(e, completionsFor(e.WalkBase(), hist, commands)), opt.color))
	}
	draw()

	// One report for "raw mode could not be re-entered", because the editor would
	// then keep drawing frames a cooked terminal echoes over — silently unusable.
	// #16 took this from two copies to four, and M2's streaming adds a fifth.
	lostTerminal := func(err error) int {
		finish()
		fmt.Fprintf(stderr, "define: lost the terminal: %v\n", err)
		return 1
	}

	// ONE entry into the ask path for this loop, reached from two places: a
	// forced question ("?…") and a dictionary miss that reads as one. M2 hangs
	// the interrupter and the streaming writer here, so a second copy of this
	// wiring would be a second copy of Ctrl-C's meaning (ARCH-DRY).
	//
	// It runs COOKED for the same reason a command does: in raw mode "\n" is a
	// line feed with no carriage return.
	// The caller supplies the leading "\r\n", because the unforced route has
	// already written one before submitLine — emitting a second here put the two
	// routes' output at different heights, which is exactly the divergence one
	// shared closure exists to prevent.
	askInSession := func(q question) error {
		// Streamed in RAW mode through crlfWriter rather than under cooked():
		// deltas arrive continuously and flapping the terminal per delta is not
		// a thing, and staying raw is also what keeps the key reader seeing
		// bytes — which is what lets Ctrl-C mean something narrower here (D6).
		ask(ctx, d, opt, &sess, &crlfWriter{w: stdout}, &crlfWriter{w: stderr}, q)
		var err error
		if err != nil {
			return err
		}
		fmt.Fprint(stdout, "\r\n")
		draw()
		return nil
	}

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
				cmd := parseREPLLine(e.String(), sess.hasCurrent())
				// Whatever happens next writes below this line, so the dropdown
				// has to go before any of it.
				clearMenu()
				// Redraw the committed line with NO suggestion before advancing:
				// the grey tail was never accepted, so leaving it in scrollback
				// claims the user typed something they did not.
				submitted := e
				e = NewEditor()
				if cmd.kind == cmdDefine || cmd.kind == cmdCommand || cmd.kind == cmdAsk {
					fmt.Fprint(stdout, RenderLine(submitted, "", opt.color))
				}
				if cmd.kind == cmdCommand {
					// Commands print multiple lines, so they run COOKED for the
					// same reason a definition does — in raw mode "\n" is a line
					// feed with no carriage return.
					fmt.Fprint(stdout, "\r\n")
					hist.Add(cmd.recallLine()) // up-arrow recalls "/history" too
					if err := cooked(func() {
						cc := newCommandCtx(d, opt, stdout, stderr)
						// opt is this loop's own copy, so a command can change
						// the session by writing through here.
						cc.setTimes = func(n int) { opt.times = n }
						dispatchCommand(cmd, commands, cc)
					}); err != nil {
						return lostTerminal(err)
					}
					fmt.Fprint(stdout, "\r\n")
					draw()
					continue
				}
				if cmd.kind == cmdAsk {
					// Up-arrow must recall the question you just asked, exactly
					// as it recalls a /command. Added here rather than inside
					// askInSession, which the unforced route reaches AFTER
					// submitLine has already recorded the line.
					hist.Add(cmd.recallLine())
					fmt.Fprint(stdout, "\r\n")
					if err := askInSession(question{text: cmd.question, forced: true}); err != nil {
						return lostTerminal(err)
					}
					continue
				}
				if cmd.kind != cmdDefine {
					// cmdReplay and cmdNothing both stay on this line: the
					// indicator is drawn over the prompt, then the prompt back.
					if cmd.note != "" {
						// eraseLine like replayInPlace's siblings: without it the
						// note is appended to the line the user typed and reads
						// as `› ?define: type a question after "?"`.
						fmt.Fprintf(stderr, "%sdefine: %s\r\n", eraseLine, nothingSays(cmd, true))
						draw()
						continue
					}
					replayInPlace(ctx, d, opt, sess.current, stdout, stderr)
					draw()
					continue
				}
				// In RAW mode "\n" is a line feed only — no carriage return — so
				// the next line would start at the current column. Everything
				// written before we drop back to cooked mode needs "\r\n".
				fmt.Fprint(stdout, "\r\n")
				out, err := submitLine(ctx, cooked, d, opt, cmd, hist, &sess, stdout, stderr)
				if err != nil {
					return lostTerminal(err)
				}
				if out.ask != "" {
					// The unforced route into the SAME closure the forced one
					// uses. submitLine has already recorded the line for recall
					// and left the session's current word alone.
					if err := askInSession(question{text: out.ask}); err != nil {
						return lostTerminal(err)
					}
					continue
				}
				// A blank line between the entry and the next prompt: without it
				// the prompt butts against the last line of the definition and
				// reads as part of it.
				fmt.Fprint(stdout, "\r\n")
				draw()
				continue
			}
			draw()
		}
	}
}

// replayInPlace speaks the current word again without moving the cursor off the
// prompt line. Stays in RAW mode throughout: Ctrl-C must reach the key reader as
// a byte while playback blocks.
func replayInPlace(ctx context.Context, d deps, opt options, current string, stdout, stderr io.Writer) {
	switch {
	case current == "":
		// Through nothingSays, not a copy of its sentence: this is the live path
		// for a bare Enter with nothing current, and a byte-identical duplicate
		// is what made "the one place that answers this" false the moment it was
		// written (BR-20).
		fmt.Fprintf(stderr, "%sdefine: %s\r\n", eraseLine, nothingSays(replCommand{}, true))
	case opt.noAudio || opt.times <= 0:
		fmt.Fprint(stderr, eraseLine+"define: nothing to replay: audio is off\r\n")
	default:
		playAnnounced(ctx, d, opt, current, indicator{show: true, erase: eraseLine}, stdout, stderr)
	}
}

// submitLine handles one submitted word.
func submitLine(ctx context.Context, cooked func(func()) error, d deps, opt options, cmd replCommand,
	hist History, sess *session, stdout, stderr io.Writer) (lookupOutcome, error) {
	line := cmd.word

	// Render in COOKED mode so newlines translate, but play in RAW mode so
	// Ctrl-C arrives as a byte the key reader can act on. Playback is the part
	// that blocks for seconds; doing it cooked made cancellation depend on a
	// signal that raw mode exists to replace.
	var out lookupOutcome
	if err := cooked(func() { out = lookupAndRender(d, opt, cmd, stdout, stderr) }); err != nil {
		return out, err
	}
	if out.play {
		playAnnounced(ctx, d, opt, line, indicator{show: true, before: "\r\n", erase: eraseLine}, stdout, stderr)
	}
	// Recorded whatever it turned out to be — a typo you want to edit and retry,
	// and a question you want to ask again, are both worth an Up-arrow. Stored
	// in its re-submittable form, so a forced line comes back still forced.
	hist.Add(cmd.recallLine())
	// A QUESTION IS NOT A LOOKUP, and out.code is 0 for both: the ask outcome
	// carries no failure, so testing the code alone would make the question the
	// current word and have the next bare Enter "replay" it.
	if out.ask == "" && out.code == 0 {
		sess.sawLookup(line, out)
	}
	return out, nil
}
