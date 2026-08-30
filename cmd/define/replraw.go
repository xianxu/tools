package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/xianxu/tools/cmd/define/store"
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
		return replLines(ctx, interrupts, d, opt, stdin, stdout, stderr, false /*stdin is a tty*/, true /*we own it*/)
	}
	sess, err := enterRaw(f)
	if err != nil {
		fmt.Fprintf(stderr, "define: cannot enter raw mode (%v); falling back to line input\n", err)
		return replLines(ctx, interrupts, d, opt, stdin, stdout, stderr, false /*stdin is a tty*/, true /*we own it*/)
	}
	defer sess.restore()

	keys := readKeys(ctx, f, interrupts)
	// The alternate screen, and with it the END of the cooked/raw dance (#30 D4).
	// `cooked()` existed so a definition's bare "\n"s translated while it was
	// printed; here the screen places every line itself, so nothing depends on
	// the line discipline and raw mode is continuous — which is what "render
	// cooked, play raw" wanted all along.
	sess.enterAlt()
	// And the mouse, whose wheel this loop needs (M1.4b): inside the alternate
	// screen a terminal sends the wheel as ARROW KEYS unless asked to report the
	// mouse, and Up/Down here are the history walk — so a scroll walked history.
	// The bytes are identical, so nothing could tell them apart; the report is
	// the only way to be handed the gesture the user actually made.
	sess.enterMouse()
	live := newLiveScreen(stdout, terminalRows(stdout))
	// The shape is MEASURED here, where the terminal is, and delivered to the
	// loop as a value — so the loop's new select case knows nothing about
	// os/signal and everything about what it has to redraw.
	resizes := watchResize(ctx, d.notifySignals, func() winSize {
		return winSize{rows: terminalRows(stdout), cols: terminalWidth(stdout)}
	})
	finish := func() {
		// Painting stops BEFORE the terminal is handed back: a frame drawn after
		// restore lands on the normal screen, over whatever was there before.
		live.Stop()
		sess.restore()
	}
	// BOTH streams are the screen, stderr included (D5b). A diagnostic written
	// straight to the terminal while the alternate screen is up lands wherever
	// the cursor happens to be and corrupts the frame; through the screen it is a
	// buffer line like any other, and survives to the exit transcript, where
	// today it is simply gone. The one-shot and piped paths keep the real stderr
	// (D6), so a script's `2>` is untouched.
	return runEditor(ctx, keys, interrupts, d, opt, live, resizes, finish, live, live)
}

// wheelLines is how far one wheel event moves the viewport.
//
// Three, which is what the terminal itself means by a notch: left to its own
// devices in the alternate screen it translates one notch into THREE arrow keys.
// Matching that keeps the gesture feeling like the terminal's own scroll rather
// than like this program's idea of one.
const wheelLines = 3

// display is the loop's whole view of the terminal: one frame out, and a
// viewport it can move.
//
// An interface rather than the concrete screen, for the reason runEditor takes a
// key CHANNEL rather than a file: the loop is drivable with no terminal at all,
// which is what keeps Apply and RenderLine testable as pure functions.
// liveScreen is the production implementation.
type display interface {
	// Draw puts one frame on the screen: the buffer's visible tail, the prompt,
	// and the command menu under it.
	Draw(prompt string, menu []string)
	// Page moves the viewport by whole screenfuls — positive is BACKWARD, toward
	// older text, which is the direction "page up" means to a reader. How tall a
	// page is belongs to the screen; the loop only knows a key was pressed.
	Page(n int)
	// Scroll moves it by LINES, in the same direction. The wheel is a finer
	// gesture than the page keys and a screenful per notch would be unusable.
	Scroll(lines int)
	// Resize sets the terminal's height. The caller redraws — it has just
	// learned the new WIDTH as well, and the live edge is rendered against that.
	Resize(rows int)
}

// runEditor is the editor loop with the terminal factored out: keys arrive on a
// channel, `view` is the screen it draws on and scrolls, and `finish` restores
// the terminal. Tests drive it with a scripted channel and no terminal at all —
// which is the whole point of keeping Apply and RenderLine pure.
// interrupts sits beside keys because they are two halves of one story: the
// channel carries the byte transport, and the sink decides what an interrupt
// from EITHER transport means while this loop owns the foreground.
//
// stdout and stderr are the SCREEN in production (D5, D5b) and plain buffers in
// tests. The loop cannot tell the difference, which is the point: every writer
// it had keeps writing, and only the destination changed.
func runEditor(ctx context.Context, keys <-chan Key, interrupts *interrupter, d deps, opt options,
	view display, resizes <-chan winSize, finish func(), stdout, stderr io.Writer) int {
	if interrupts == nil {
		// A loop with no sink still runs; nothing can scope an interrupt, which
		// is the honest behaviour for a caller that supplied no cancellation.
		interrupts = &interrupter{}
	}

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
	// The highlight set, resolved the same way every other render path resolves
	// it. Not a local policy: vocabularyFor owns "loaded, and only with colour",
	// so a path that forgets to ask cannot silently render against an empty set.
	voc := vocabularyFor(d, opt)
	e := NewEditor()
	var sess session
	// Apply gets the candidate list computed BEFORE the keystroke, which is
	// correct: it is deciding what to do with that keystroke given the line as
	// it stands, and a history walk anchors on it. draw() computes its own from
	// the line AFTER — passing it this one is what made the grey tail disagree
	// with what Tab accepted.
	//
	// The menu is a DROPDOWN drawn under the prompt, and it is now an argument to
	// one frame rather than rows this loop tracks. paintMenu counted the rows it
	// had drawn so it could erase exactly that many, and carried a documented
	// known limit for when the count was wrong ("if the menu does not fit below
	// the cursor the terminal scrolls, and the cursor-up count then lands a row
	// off"). A whole frame cannot be off by a row, because it never counts rows
	// it drew earlier — so the arithmetic, its limit, and clearMenu with them,
	// are deleted rather than ported.
	//
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
		// completionsFor rather than candidatesFor: draw renders only the grey
		// tail, so resolving the pair here would build a recall list per
		// keystroke that nothing reads. Same function that fills .complete, so
		// the two paths cannot disagree.
		//
		// The prompt and the menu go to the FRAME rather than to stdout, and
		// that is the whole of this loop's change: they are the live edge,
		// rewritten on every keystroke, so buffering them would file a copy of
		// the prompt per character typed.
		view.Draw(RenderLine(e, Suggestion(e, completionsFor(e.WalkBase(), hist, commands)), voc, opt.color),
			menuLines(e.String(), commands, opt.width))
	}
	draw()

	// ONE entry into the ask path for this loop, reached from two places: a
	// forced question ("?…") and a dictionary miss that reads as one. It used to
	// hang the streaming writers too, because a raw terminal was what made them
	// differ from the line loop's; the screen took that job (D5), and what is
	// left here is the pair of routes. The interrupt SCOPE does not hang here
	// either — askScoped owns that, so both loops cannot drift on an ordering
	// that is silent when wrong.
	//
	// The caller supplies the leading "\r\n", because the unforced route has
	// already written one before submitLine — emitting a second here put the two
	// routes' output at different heights, which is exactly the divergence one
	// shared closure exists to prevent.
	askInSession := func(q question) {
		// Ctrl-C here means "stop this answer", not "quit" — the one place in
		// this program where it means something narrower. askScoped owns the
		// sequence and its ordering; the reader swallows the interrupt it
		// consumed rather than also handing it to this loop, where it would quit
		// the session the moment the answer ended (#16 D5).
		//
		// Streamed straight into the screen, with NO crlfWriter (#30 D5). It
		// wrapped this path because a raw terminal needs the carriage half of
		// every line break and a stream cannot be flapped cooked per delta; the
		// screen now owns where a line goes, and two owners of line endings is
		// how they drift. `--play` keeps its own crlfWriter, because it keeps
		// drawing its own frames (D5a).
		askScoped(ctx, interrupts, func(qctx context.Context) int {
			return ask(qctx, d, opt, &sess, stdout, stderr, q)
		})

		fmt.Fprint(stdout, "\r\n")
		draw()
	}

	// Every exit is finish() and nothing else. The farewell newline the three of
	// them used to write is gone with the alternate screen: leaving it restores
	// the normal buffer exactly as the shell left it, so there is no half-drawn
	// line for a newline to finish — and a write after finish would land in a
	// buffer nobody paints again (#30 D3 prints the transcript there instead).
	for {
		select {
		case <-ctx.Done():
			finish() // before anything else can write: never exit leaving raw mode on
			return 0
		case sz := <-resizes:
			// A frame is drawn for a SHAPE, and both halves of it go wrong. Too
			// tall and the terminal scrolls, which moves every row the app
			// believes it placed; too narrow and the width the next entry wraps
			// to is not the width it is read at.
			//
			// Lines already in the buffer keep the wrapping they were rendered
			// with. Re-wrapping them would mean re-rendering from entries this
			// program does not keep, and a session's scrollback going fluid on
			// every drag is not obviously better than one that reads as a record
			// of what was shown.
			//
			// A resize that arrives while a recording plays waits here until the
			// loop is idle: this case is only reached between keystrokes, so the
			// frame is briefly stale rather than concurrently painted by two
			// goroutines.
			opt.width = sz.cols
			view.Resize(sz.rows)
			draw()
			continue
		case k, open := <-keys:
			if !open {
				finish()
				return 0
			}
			// A VIEWPORT gesture never reaches Apply: it changes what you are
			// looking at, not the line you are typing, so the editor does not
			// have to learn that a screen exists. PageUp/PageDown and the wheel
			// only — the obvious half-page bindings Ctrl-U and Ctrl-D are already
			// the kill and the EOF (key.go), and taking either would be a silent
			// regression in an editor people already use.
			switch k.Kind {
			case KeyPageUp:
				view.Page(1)
				continue
			case KeyPageDown:
				view.Page(-1)
				continue
			case KeyWheelUp:
				view.Scroll(wheelLines)
				continue
			case KeyWheelDown:
				view.Scroll(-wheelLines)
				continue
			}
			cands := candidatesFor(e.WalkBase(), hist, commands)
			var act Action
			e, act = Apply(e, k, cands)
			switch act {
			case ActInterrupt, ActEOF:
				finish()
				return 0
			case ActSubmit:
				// Route through the SAME decision table the line loop uses.
				// Bypassing it meant the interactive path quietly disagreed about
				// what a line means — no trimming, and "hot  dog" not collapsed to
				// the multi-word headword the dictionary actually has (ARCH-DRY).
				cmd := parseREPLLine(e.String(), sess.hasCurrent())
				// Redraw the committed line with NO suggestion before advancing:
				// the grey tail was never accepted, so leaving it in scrollback
				// claims the user typed something they did not.
				submitted := e
				e = NewEditor()
				// THE PROMPT BELONGS TO A LOOP THAT IS WAITING. From here until
				// the next draw() this one is working — looking up, streaming,
				// playing — and every write repaints the frame around whatever
				// live edge was last recorded. Left alone that is the line just
				// submitted, so `arrondissement` appeared a second time under
				// its own definition while the recording played
				// (operator-reported). Blanking it is not a patch on that
				// instance: a prompt drawn while nothing is reading keys invites
				// typing at a line that does not exist.
				view.Draw("", nil)
				if cmd.kind == cmdDefine || cmd.kind == cmdCommand || cmd.kind == cmdAsk {
					fmt.Fprint(stdout, RenderLine(submitted, "", voc, opt.color))
				}
				if cmd.kind == cmdCommand {
					// Commands print multiple lines, and the screen places every
					// one of them — which is why there is no mode to drop into
					// here any more (D4).
					fmt.Fprint(stdout, "\r\n")
					hist.Add(cmd.recallLine()) // up-arrow recalls "/history" too
					var pron store.Lang
					cc := newCommandCtx(d, opt, stdout, stderr)
					// opt is this loop's own copy, so a command can change
					// the session by writing through here.
					cc.setTimes = func(n int) { opt.times = n }
					// And &voc, because THIS loop caches the highlight set in
					// a local before the loop starts (see above). Reassigning
					// d alone would leave the editor highlighting from the
					// previous language's deck — the one thing a deps swap
					// cannot reach.
					cc.setLang = sessionSetLang(&d, &opt, cc.setLang, &voc, stderr)
					cc.entry = sess.entry
					// RECORDED here, PERFORMED below. The hazard this separation
					// was built for is gone — raw mode no longer flaps, so
					// playing where the command runs could not hand Ctrl-C to a
					// line discipline any more (lessons.md, "render cooked, play
					// raw", is answered by D4). It stays because it is also the
					// right shape: a command decides WHAT to replay and the loop
					// owns replaying, so /pron and a bare Enter remain one path.
					if sess.hasCurrent() {
						cc.replay = func(l store.Lang) { pron = l }
					}
					dispatchCommand(cmd, commands, cc)
					if pron != "" {
						// The same replay a bare Enter takes, one parameter apart.
						// No reset: pron is declared inside this block and
						// does not outlive the iteration.
						replayInPlace(ctx, d, opt, sess, pron, stdout, stderr)
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
					askInSession(question{text: cmd.question, forced: true})
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
					replayInPlace(ctx, d, opt, sess, "", stdout, stderr)
					draw()
					continue
				}
				// End the committed prompt line before the entry is written
				// under it.
				fmt.Fprint(stdout, "\r\n")
				out := submitLine(ctx, d, opt, cmd, hist, &sess, stdout, stderr)
				if out.ask != "" {
					// The unforced route into the SAME closure the forced one
					// uses. submitLine has already recorded the line for recall
					// and left the session's current word alone.
					askInSession(question{text: out.ask})
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
//
// `pron` is #29's source language, empty for an ordinary replay. A bare Enter
// and `/pron fr` are therefore ONE replay path with one parameter rather than
// two implementations — #14's "two loops, one decision table", applied before a
// second one could be written. It takes the whole session because the source
// spellings come from the entry, not just the word.
func replayInPlace(ctx context.Context, d deps, opt options, sess session, pron store.Lang, stdout, stderr io.Writer) {
	switch {
	case sess.current == "":
		// Through nothingSays, not a copy of its sentence: this is the live path
		// for a bare Enter with nothing current, and a byte-identical duplicate
		// is what made "the one place that answers this" false the moment it was
		// written (BR-20).
		fmt.Fprintf(stderr, "%sdefine: %s\r\n", eraseLine, nothingSays(replCommand{}, true))
	case opt.noAudio || opt.times <= 0:
		fmt.Fprint(stderr, eraseLine+nothingToReplay+"\r\n")
	default:
		playAnnounced(ctx, d, opt, utteranceFor(sess.current, sess.entry, pron, opt),
			indicator{show: true, erase: eraseLine}, stdout, stderr)
	}
}

// submitLine handles one submitted word.
func submitLine(ctx context.Context, d deps, opt options, cmd replCommand,
	hist History, sess *session, stdout, stderr io.Writer) lookupOutcome {
	line := cmd.word

	// One mode throughout (#30 D4). This used to render under cooked() so a
	// definition's bare "\n"s translated, and play outside it so Ctrl-C reached
	// the key reader as a byte — the pair that lessons.md records as "render
	// cooked, play raw". The screen places lines itself, so nothing here depends
	// on the line discipline and raw mode never drops: the rule is satisfied by
	// there being nothing left to flap.
	out := lookupAndRender(d, opt, cmd, stdout, stderr)
	if out.play {
		playAnnounced(ctx, d, opt, utteranceFor(line, out.entry, cmd.pron, opt),
			indicator{show: true, before: "\r\n", erase: eraseLine}, stdout, stderr)
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
	return out
}
