package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/schedule"
	"github.com/xianxu/tools/cmd/define/store"
)

// runPlay is a review session: today's queue, one question at a time.
//
// The division of labour, which is the whole design: schedule decides WHICH
// words are worth asking about, play decides how to ask and what an answer
// means, and this owns the terminal, the store and the audio. Neither of those
// packages can reach any of these things, which is enforced rather than asserted
// (cmd/define/puretest).
func runPlay(ctx context.Context, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
	// No deck, no session. DEFINE_NO_CAPTURE is one cause and openStore's Getwd
	// failure is another, so the guard is on the PRECONDITION rather than on the
	// flag — keying it on the env var would hand a nil store to the queue builder
	// on the other path and panic.
	if d.deck == nil {
		fmt.Fprintln(stdout, "define: no deck in this directory, so there is nothing to review")
		return 0
	}

	// EVERY PRECONDITION FOR OWNING THE TERMINAL IS SETTLED HERE, before the
	// command does any WORK — the same rule main.go states for usage errors
	// ("settled BEFORE a store is opened"). Above `todaysQuestions`, which reads
	// a file per deck word and a file per day of log, and above anything that
	// paints; below the deck-absent guard, which costs nothing and answers a
	// question about the DIRECTORY that a reader in the wrong one needs more
	// than they need to be told about their terminal.
	//
	// `repl` computes `terminalUI := interactive && opt.tty` and falls back to
	// the line loop, with a comment recording that this family — gating cursor
	// control on the wrong stream — has already shipped three times here.
	// `--play` gated on stdin alone, which was harmless while it appended lines
	// and emitted no escapes at all; a sitting now takes the alternate screen,
	// reports the mouse and paints `ESC[H ESC[J` frames.
	//
	// Three causes, told apart because their fixes differ. A sitting the learner
	// cannot see is not a sitting, so this REFUSES rather than degrading: there
	// is no line-mode fallback to fall to, and pretending otherwise would write
	// the frames anyway.
	f, isFile := stdin.(*os.File)
	if !isFile || d.stdinIsTerminal == nil || !d.stdinIsTerminal() {
		// A review is a conversation with a person. Piped input would answer
		// questions it never saw.
		fmt.Fprintln(stderr, "define: --play needs a terminal")
		return 1
	}
	if !isTerminal(stdout) {
		// A redirected stdout would otherwise collect frames at a fabricated 80
		// columns.
		fmt.Fprintln(stderr, "define: --play draws a full screen, so its output must be a terminal")
		return 1
	}
	if !opt.tty {
		// -no-color means "emit no ANSI", which main.go's own comment says
		// "disables cursor control too — the flag exists for terminals that
		// mangle escapes". Such a terminal is exactly where a full-screen
		// sitting would be unusable rather than merely ugly.
		fmt.Fprintln(stderr, "define: --play draws a full screen, which -no-color turns off; run it without -no-color")
		return 1
	}

	questions, held, code := todaysQuestions(d, opt, stdout, stderr)
	if code != 0 || len(questions) == 0 {
		return code
	}

	// Same interrupt shape as repl (#16 D5), and now literally the same body —
	// see detachedInterrupts for why the detach happens before the loop starts.
	ctx, interrupts, cancel := detachedInterrupts(ctx, d)
	defer cancel()

	sess, err := enterRaw(f, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "define: could not enter raw mode: %v\n", err)
		return 1
	}
	defer sess.restore()

	// THE SCREEN, and it is the LAST writer of line endings in this binary (D1).
	//
	// A sitting used to write lines to a scrolling terminal through a translating
	// writer: in raw mode a bare \n moves down WITHOUT returning to column 0, so
	// a multi-line definition cascaded diagonally across the screen. The screen
	// places every row itself, so nothing on this path depends on the line
	// discipline any more — and with the editor already converted (#30 D5), that
	// writer had no caller left and is deleted rather than kept for a third.
	// The SHARED builder, with the pinned screen as its one argument: a status
	// bar belongs at the terminal's bottom edge, where the editor's dropdown
	// belongs under the line being typed (D3a). Everything else about taking a
	// terminal is the same question, and the first version of this file answered
	// it a second time (BR-7).
	return playSession(ctx, d, opt, play.NewSession(questions), held,
		readKeys(ctx, f, interrupts), newConsole(ctx, d, sess, stdout, newPinnedScreen))
}

// playSession drives the state machine and performs its outcomes.
//
// Split from runPlay so a test can drive a whole session with a scripted key
// channel and no terminal at all — the setup in runPlay is the part that needs
// one.
//
// It takes a `console` rather than a pair of writers and a borrowed terminal:
// the same type the editor loop takes, which is the whole of #41's claim that
// there is ONE way to draw in this binary.
func playSession(ctx context.Context, d deps, opt options, s play.Session, held *sittingDeck,
	keys <-chan Key, con console) int {

	view, stdout, stderr := con.view, con.stdout, con.stderr

	// THE BAR'S FIGURES, refreshed in memory (D7).
	//
	// `budget` and `total` are fixed for the sitting; `load`, `fresh` and `done`
	// move only when the deck does. So this is called from the two outcomes that
	// change it — a record and a drop — and NOT once per frame: a frame is per
	// keystroke, and an O(deck) walk per keystroke is a cost the plan's own
	// table says this path does not pay.
	var fig sittingFigures
	refresh := func() {
		// The WHOLE struct, then the two the deck does not know. Copying named
		// fields out of a fresh figures() means a field added later goes
		// silently stale in the bar, and a stale number on screen is worse than
		// no number.
		fig = held.figures(opt.count)
		fig.total = sittingWords(s.Questions)
		// ANSWERED, which is what the Spec's bar says. A dropped word is not an
		// answer: the learner curated it away rather than being asked about it.
		fig.done = s.Right + s.Wrong
	}
	// The FIRST frame derives from the same expression as every later one:
	// refresh() is exactly right at init, because Right+Wrong is zero there.
	// Hand-copying its first lines here is how a third field added to it would
	// be stale before the sitting's first keystroke.
	refresh()

	// THE QUESTION IS A BUFFER LINE AND THE KEYS ARE THE LIVE EDGE (D4).
	//
	// This is the change of model, and the place a naive port breaks. The old
	// draw() wrote the prompt, the reveal and the keys on EVERY call, which is
	// correct for a scrolling terminal and would, against a line buffer, file a
	// copy of the question per keystroke.
	//
	// So the loop tracks which question it has already written. `written` is that
	// question's index, and -1 means none — the state is one int, and it lives
	// here because "perform the outcomes" already does.
	written := -1
	// THE CHROME'S PALETTE, resolved once for the sitting. From `main`, because
	// `main` owns the terminal's colours — the same seam `boardPalette` sits on,
	// and the same reason: a form or a formatter choosing its own escape
	// sequences would be a second owner of a decision `newPalette` already makes
	// for every other surface (#44).
	pal := newPalette(opt.color)
	// boardWhole is whether the CURRENT board is drawn in full, recomputed by
	// every frame and read by the key loop (R17). False only after a resize has
	// shrunk the terminal under a board already in play — the board is not
	// re-selected then, because its marks are already in the log.
	boardWhole := true
	// relearn is the words this board has been marked `no` on, emptied into the
	// transcript when it closes. A slice on the loop rather than state on the
	// form: see relearnLine.
	var relearn []string
	// `show` rather than `draw`: the package-level draw() is gone, and a closure
	// wearing its name would read as the same thing narrowed rather than as the
	// different thing it is — this one decides what has changed, where that one
	// printed everything every time.
	show := func() {
		q := s.Current()
		// A GRID IS THE LIVE EDGE AND NOT A BUFFER LINE (#40 D10).
		//
		// The buffer is append-only, which is what makes a click's coordinates
		// exact — and also what would freeze a grid the moment it was written.
		// Its marks change as they land, so it is rebuilt into the FOOTER on
		// every frame instead, and nothing about it reaches the transcript
		// except the relearn line it writes as it closes.
		if g, ok := q.(play.Grid); ok {
			// LAID OUT FOR THE TERMINAL AS IT IS, at DRAW time (R17).
			//
			// The resize case used to be the only place that told a form its
			// width, which fixed the board that happened to be ON SCREEN when
			// the window changed and no other: a board that became current
			// afterwards was built by todaysQuestions at the old width and
			// painted with rows too wide for the terminal, so its rows wrapped,
			// a footer entry stopped being one physical row, and the wrong-word
			// click was back. `show` is the ONE place that draws, so it is the
			// only place that can promise this for every board.
			//
			// Idempotent: Resize returns immediately when the width is the one
			// the form already has, which is every frame but the first after a
			// change.
			termRows, termCols := view.Size()
			g.Resize(termCols)
			// AND WHETHER IT FITS, which decides one thing: whether Enter may
			// spend it. A shrunken terminal drops trailing footer rows, so some
			// grid rows are simply not drawn — and Enter takes every unmarked
			// word as Wrong, including words the learner never saw. That is a
			// box halved per word on a keystroke meaning "ask me these again".
			boardWhole = boardFitsIn(q, termRows, termCols)
			// NO BLANK BUFFER LINE HERE ANY MORE (#44). A board used to write one
			// the first time it was drawn, because it writes nothing else to the
			// buffer and its grid would otherwise begin immediately under the
			// previous question's last line. That gap is now the FRAME's, held
			// for every form by `chromeGap` — so this was one form's exception to
			// a rule the frame did not yet have, and it also spent a buffer line
			// on it, which the exit transcript then carried.
			view.Draw(asChrome(boardPrompt(q, boardWhole), pal), boardFooter(q, fig, pal))
			return
		}
		if q != nil && written != s.Index {
			written = s.Index
			// Plain \n: the screen places every row, so nothing here decides
			// where a line goes (D1).
			//
			// THE PROMPT WORD IS CLICKABLE (T4), when the prompt has one.
			writeRendered(stdout, "\n"+q.Prompt()+"\n", promptRegions(q))
		}
		// The grading keys are the PROMPT and the bar is the FOOTER, which gets
		// the order of sacrifice right for free (D3): Paint clips the prompt last
		// and drops footer rows first, and a learner who cannot see the keys
		// cannot answer at all, while one who cannot see their daily load loses
		// nothing this minute.
		view.Draw(asChrome(livePrompt(s), pal), []string{asChrome(sittingBar(fig), pal)})
	}

	// Every exit is the summary and THEN the terminal, in that order. The summary
	// is a buffer write like any other and handing the terminal back is what
	// prints the buffer (D5) — reversed, a sitting's last line would be written
	// into a screen nobody paints again and would be missing from the transcript
	// as well as from the terminal.
	over := func() int {
		code := finish(stdout, s, fig)
		con.finish()
		return code
	}

	show()
	for !s.Done {
		// Cancellation is checked BEFORE the select, not only inside it.
		//
		// select picks uniformly at random among ready cases, so with a
		// cancelled context and a key already buffered it would sometimes grade
		// one more answer after Ctrl-C — recording a verdict for a word the
		// learner had stopped on. Caught as an intermittent test failure, which
		// is the only way a random-choice bug ever shows up.
		if ctx.Err() != nil {
			return over()
		}
		var k Key
		select {
		case <-ctx.Done():
			return over()
		case sz := <-con.resizes:
			// A frame is drawn for a SHAPE, and both halves of it go wrong: too
			// tall and the terminal scrolls, which moves every row the sitting
			// believes it placed; too narrow and the bar and the keys are laid
			// out against a width that is not there.
			//
			// The SHAPE, and nothing else. The editor also re-derives opt.width
			// here because it renders new entries mid-session; a sitting renders
			// its whole queue up front and wraps at the screen's Write against
			// the screen's own cols, which Resize is what updates. Setting a
			// second width here would be a second answer to the same question.
			view.Resize(sz.rows, sz.cols)
			// NOTHING IS TOLD ABOUT THE FORM HERE, and that is R17.
			//
			// This case used to relayout the board — correctly for the one on
			// screen, and for no other: a board that became current later was
			// built at the old width and painted too wide, so its rows wrapped
			// and a click on a continuation row meant a different word. `show`
			// is the one place that draws, so it is the only place that can
			// promise a layout for every board, and it does it every frame.
			//
			// AND NO RE-SELECTION. The board stays, at the new shape, even if the
			// terminal is now too short to draw it whole — D15's rule holding
			// rather than bending. Selection chooses the form for words not yet
			// asked; this board's marks are already in the log, so "send it to
			// 2.3 instead" would mean re-asking answered words. What a short
			// terminal loses is the bar, then the panel, then grid rows — and
			// Enter is held while any of them are missing, which is what makes
			// those losses survivable rather than "harmless" (R18).
			show()
			continue
		case got, ok := <-keys:
			if !ok {
				return over()
			}
			k = got
		}

		// A VIEWPORT GESTURE NEVER REACHES play (D6), through the SAME helper
		// the editor uses.
		//
		// It changes what you are LOOKING AT, not what you are answering, and
		// `play` is mechanically guarded pure — main owns the terminal and knows
		// Ctrl-C is 0x03; play must not. A `play.Input` kind for "page up" would
		// put a display concept inside the pure package and every future form
		// would inherit it, which is why an earlier draft routing these through
		// toInput was reversed.
		//
		// This is what a long reveal needed: form 2.3 on a word like `run` is
		// several screenfuls, and before frames the question simply scrolled off
		// the top with no way back. The first cut of it was a second copy of the
		// editor's four-case switch, which is how the two loops would come to
		// disagree about which direction a page goes (BR-1).
		if viewportGesture(view, k) {
			continue
		}

		// A CLICK NEVER ANSWERS A FORM THAT DID NOT ASK FOR IT (D8, T7; #40 D11).
		//
		// #38 shipped this as "a click ACTS and never answers", and the reason
		// stands unchanged for every form that holds one word: hearing the word
		// is what `y`/`n` are answering ABOUT, so a click that recorded a review
		// would corrupt the schedule silently — the worst kind of bug here,
		// because the damage is to data the learner cannot see.
		//
		// A board asks for it. Its cells ARE its answers, so the click is offered
		// to the form FIRST and falls through to playRegion when the form
		// declines — which every existing form does, by not being a grid. That is
		// why #38's row is still green untouched, and being untouched is the
		// proof that the seam widened rather than branched.
		//
		// A click on nothing is nothing: no beep, no message. Pointing at
		// ordinary text is not an error.
		var in play.Input
		if k.Kind == KeyClick {
			cell, marks := formCell(view, s.Current(), k)
			if !marks {
				if r, ok := view.RegionAtRow(k.Row, k.Col); ok {
					// The indicator is playRegion's own now (#44). It used to be
					// passed, and this site passed `defaultIndicator` under a
					// comment saying that was "what every other playback on this
					// path already uses" — true of the ONE-SHOT path and false of
					// every screen, which is how a blank line came to be committed
					// per click.
					playRegion(ctx, d, opt, r, "", stdout, stderr)
					show()
				}
				continue
			}
			in = play.Input{Kind: play.InputMark, Cell: cell}
		} else {
			var ok bool
			if in, ok = toInput(k); !ok {
				continue
			}
			// ENTER IS HELD WHILE THE BOARD IS NOT DRAWN IN FULL (R17).
			//
			// It takes every unmarked word as Wrong, and on a shrunken terminal
			// some of those words were never on screen — so one keystroke would
			// halve the box of words the learner had no chance to look at. The
			// loop refuses rather than the session, for the same reason a
			// viewport gesture never reaches `play` (D6): what was DRAWN is the
			// terminal's business and the pure package must not learn about it.
			//
			// Marking still works, and Ctrl-C is still free — so nothing is
			// stuck. The prompt says why, because a key that silently stops
			// working is the thing a learner blames themselves for.
			if in.Kind == play.InputFinish && !boardWhole {
				if _, isGrid := s.Current().(play.Grid); isGrid {
					continue
				}
			}
		}

		// The question the keystroke is ABOUT, read before Apply moves on. A
		// reveal never advances, so s.Current() would answer the same — but
		// reading it after would make that a fact about Apply that this loop
		// silently depends on, and BR-4 is what a nil Current() costs.
		asked := s.Current()
		var outs []play.Outcome
		s, outs = play.Apply(s, in)
		// THE BOARD'S OUTCOME, KEPT (D10). Its grid never reaches the buffer, so
		// without this a swept board would leave the transcript with no trace
		// that the sitting happened at all.
		_, wasGrid := asked.(play.Grid)
		if wasGrid {
			for _, out := range outs {
				if out.Kind == play.OutcomeRecord && out.Verdict == play.Wrong {
					relearn = append(relearn, out.Word)
				}
			}
		}

		// EVERY outcome, ONCE, IN ORDER.
		//
		// Widening Apply to []Outcome moved three obligations from the type system
		// into this loop, and each one is a row that has to be pinned HERE, at the
		// consumer — the pure tests cannot see any of them, because they assert
		// what Apply RETURNS, not what the caller does with it:
		//
		//	membership — every outcome is performed. Dropping either end of the
		//	             slice left the whole suite green (BR-8); now
		//	             TestAMissPlaysThePronunciationAndRecordsIt asserts the
		//	             play AND the event, so both `outs[:1]` and
		//	             `outs[len(outs)-1:]` redden it.
		//	order      — the record is written BEFORE the reveal, which is what
		//	             makes a miss survive anything that goes wrong while the
		//	             answer is being shown. Reversing this iteration also left
		//	             the whole suite green (BR-13); the pin is now
		//	             TestAMissRecordsBeforeItPlays, which observes the
		//	             store from INSIDE playback — #41 deleted the terminal
		//	             hand-back that used to make the order observable by
		//	             failing (D5a), and an order that is load-bearing needs a
		//	             pin that does not depend on a branch being reachable.
		//	once       — no outcome is performed twice. A duplicated record is a
		//	             second review event for one answer, which Fold would read
		//	             as another review; the event-count assertions redden it.
		//
		// The enumeration is written down because BR-8 fixed membership and left
		// order in the tree — a widened contract has more than one way to be
		// betrayed by its caller, and finding them one review at a time is what
		// this comment is here to stop.
		for _, out := range outs {
			switch out.Kind {
			case play.OutcomeRecord:
				// RECORDED NOW, before the next question is drawn. That is what makes
				// Ctrl-C lossless by construction rather than by a flush, and the
				// loop never inspects the verdict — a skip produced no outcome at
				// all, so there is nothing to filter here.
				// The whole outcome: it already carries the word, the verdict,
				// the axis a miss was missed on and whether the answer was
				// unaided. One argument cannot be passed in the wrong order.
				d.capture.CaptureReview(out, opt)
				// ...and the same transition on the copy in hand, so the bar
				// shows the cost AFTER this answer without reading anything.
				held.answered(out, d.clock.Now())
				refresh()
			case play.OutcomeFlag:
				// Its own arm, calling its own verb — exactly as OutcomeDrop
				// calls Forget rather than routing through CaptureReview. A flag
				// records no verdict, so nothing reaches Fold and the word
				// neither promotes nor demotes.
				d.capture.CaptureFlag(out, opt)
				refresh()
				fmt.Fprintf(stdout, "\nflagged %q as a bad question\n", out.Word)
			case play.OutcomeDrop:
				// Through the store's own Forget, which is --forget's path: the deck
				// loses the word and the events keep it. Reported, because removing
				// something on one keystroke should say so.
				if removed, err := d.deck.Forget(out.Word); err != nil {
					fmt.Fprintf(stderr, "define: could not remove %q: %v\n", out.Word, err)
				} else if removed {
					held.dropped(out.Word)
					refresh()
					fmt.Fprintf(stdout, "\nremoved %q from the deck\n", out.Word)
				}

			case play.OutcomeReveal:
				// The answer joins the transcript, ONCE — this is the only place
				// it is written, which is what keeps a reveal out of the buffer
				// until it is earned and out of it twice however many keys follow.
				if asked != nil {
					// THE REVEALED DEFINITION CARRIES ITS REGIONS (T5), shifted
					// by however many lines the form prepends: `Choice.Reveal()`
					// names the right option and what you picked before the entry
					// begins, and those coordinates belong to the render, not to
					// the reveal.
					reveal := "\n" + asked.Reveal() + "\n"
					writeRendered(stdout, reveal, held.marksIn(asked.Word(), reveal))
				}
				// THE PREDICATE, not a fifth hand-copy of `!opt.noAudio &&
				// opt.times > 0` (T0). playAnnounced applies it itself, so being
				// below the guard is impossible; this asks only to decide whether
				// there is anything to announce at all.
				if opt.playsAudio() {
					// RAW THROUGHOUT, and that is D5a's whole content.
					//
					// This used to call restore(), play in cooked mode, and
					// enterRaw again — because "the indicator and any warning are
					// written for a human to read" and needed newline translation.
					// Under the frame model the indicator is a frame write like
					// any other, and screen.Write already honours its `\r\x1b[K`
					// erase by taking the open line back. So the dance goes, and
					// with it the "lost the terminal after playback" branch: it
					// existed only because the terminal had been handed back and
					// might not come back, and nothing is handed back any more.
					//
					// Keeping it would have been worse than redundant. enterAlt is
					// opt-in on rawSession and restore() leaves the alternate
					// screen, so a frame-drawing sitting would have lost the alt
					// screen on its first reveal and painted every frame after it
					// over the user's scrollback (PQ-1).
					//
					// The word comes from the OUTCOME, not from s.Current(): the
					// failure mode of reading it back off the session is a
					// nil-interface panic at the end of the queue (BR-4).
					//
					// No source language here: a review session is the deck's own
					// language throughout, and #29's -pron is a per-lookup flag that
					// --play has no line to carry.
					playAnnounced(ctx, d, opt, utteranceFor(out.Word, "", "", opt),
						screenIndicator(), stdout, stderr)
				}
			}
		}
		// WRITTEN AS THE BOARD CLOSES, before the next question is drawn — so the
		// line sits in the transcript where the board was, rather than after
		// whatever came next. `asked` is the form the keystroke was about, and a
		// board leaves the screen the moment its last word is answered.
		if wasGrid && s.Current() != asked {
			if line := relearnLine(relearn); line != "" {
				fmt.Fprintf(stdout, "\n%s\n", line)
			}
			relearn = nil
		}
		show()
	}
	return over()
}

// toInput translates a decoded terminal Key into play's own Input.
//
// THIS is where main.Key stops. play must not know that Ctrl-C is 0x03 or that
// Enter may be \r — it is pure, and a package that knew those could not be
// tested without a terminal. So the translation happens once, here, at the
// boundary that already decodes escape sequences.
func toInput(k Key) (play.Input, bool) {
	switch k.Kind {
	case KeyInterrupt, KeyEOF:
		return play.Input{Kind: play.InputQuit}, true
	case KeyEnter:
		// ENTER IS NOT SPACE ANY MORE (D14). The two were one kind while every
		// form held one word and one answer; a board takes every unmarked word
		// as No when it is finished, and leaving them merged fired that on the
		// most careless key there is. Apply treats InputFinish exactly as
		// InputReveal for every form that is not a Batch, which is what keeps
		// 2.1 and 2.3 from noticing.
		return play.Input{Kind: play.InputFinish}, true
	case KeyTab:
		// Dropped on the floor until now — key.go decoded it and nothing here
		// had a case for it (D13).
		return play.Input{Kind: play.InputToggle}, true
	case KeyRune:
		switch k.Rune {
		case ' ':
			return play.Input{Kind: play.InputReveal}, true
		case 'd', 'D':
			// THE RUNE RIDES ALONG, because `d` is only the session's key while
			// the form has a current word to remove. On a form holding many it is
			// an ordinary graded key, and Apply cannot ask the form about a
			// keystroke this did not carry.
			return play.Input{Kind: play.InputDrop, Rune: k.Rune}, true
		}
		return play.Input{Kind: play.InputRune, Rune: k.Rune}, true
	}
	return play.Input{}, false
}

// boardPrompt is a board's prompt row: its own keys, or the reason Enter is
// held (R17).
//
// The REASON, not a silent refusal. A learner who presses Enter on a board that
// will not commit needs to know the window is the problem — and this row is the
// one `Paint` clips last, so it is the right place to say it.
//
// It replaces the form's keys rather than joining them, because the two would
// not both fit at the width where this happens, and a prompt that wraps is a
// frame one row taller than the board was budgeted for.
func boardPrompt(q play.Question, whole bool) string {
	if whole {
		return gradePrompt(q)
	}
	return boardRefusal
}

// boardRefusal is the prompt row when the window cannot show the whole board.
//
// NO WIDER THAN THE KEYS ROW IT REPLACES, which is a budget constraint rather
// than a style one: `Paint` charges the frame for the prompt it is given, and a
// taller replacement would drop one more footer row than the fit was computed
// against. Pinned by TestTheRefusalRowIsNoWiderThanTheKeysRow, because "these
// two strings are the same width" is not a fact anyone will re-check by eye.
const boardRefusal = "window too short for the whole board — mark what you see, Ctrl-C to stop"

// boardFitsIn is THE answer to "can this board be drawn whole here", and it is
// asked at both moments: at SELECTION, where a board that does not fit is not
// offered (D15), and at every DRAW, where the answer decides whether Enter may
// spend it (R17).
//
// One helper because it was two spellings of one formula, and they had already
// diverged: the selection copy refused a terminal under `minWrapWidth` and the
// draw-time copy did not, so a board narrowed below that by a resize still
// reported itself whole. Not reachable as harm today — the row arithmetic turns
// the answer false well before the words become unreadable — which is the reason
// to consolidate it rather than a reason not to (ARCH-DRY).
func boardFitsIn(q play.Question, termRows, termCols int) bool {
	g, ok := q.(play.Grid)
	if !ok || termCols < minWrapWidth {
		// Below minWrapWidth this program already treats the terminal as too
		// narrow to lay text out at all, and a board there is columns of stubs.
		return false
	}
	return fitsABoard(termRows, g.Rows(), displayRows(gradePrompt(q), termCols))
}

// boardPalette is how a board's marks are painted, and it is the ONE place this
// program decides that.
//
// GREEN for yes and RED for no — the conventions a terminal reader already has,
// and the pair a learner does not have to be taught. A drop is struck out rather
// than given a third hue, because a third colour would need teaching and would
// compete with those two. Bold on the two ratings, because the grid's unmarked
// cells are ordinary weight and a marked one should separate at a glance rather
// than on inspection.
//
// It comes from `main` because `main` owns the terminal: `play` is mechanically
// guarded pure, and a form choosing its own escape sequences would be a second
// owner of a decision `newPalette` already makes for every other surface. The
// board takes finished sequences, exactly as `Choice` takes finished options.
//
// Empty under `-no-color`, which `--play` refuses to run with (BR-3) — so this
// is belt rather than a reachable state, and the board then draws in plain text
// with its keys intact, which is still usable.
func boardPalette(opt options) play.Palette {
	if !opt.color {
		return play.Palette{}
	}
	// DIM STRIKETHROUGH for a removal, not a third hue. Green and red are the two
	// conventions a terminal reader already has and a learner does not have to be
	// taught; a third colour would need teaching and would compete with them. A
	// word on its way out of the deck reads as struck out, which is what it is
	// (#42).
	return play.Palette{Yes: "\x1b[1;32m", No: "\x1b[1;31m", Drop: "\x1b[2;9m", Off: "\x1b[0m"}
}

// boardFooter is the live edge for a board: everything the FORM draws, then the
// bar.
//
// One line of assembly, and that is the point. The grid and the panel are the
// board's own rendering — a form owns how it looks, and a loop composing it out
// of accessors would make the board's appearance a thing two files agree about,
// on the surface where disagreeing marks the wrong word. All this adds is the
// bar, which belongs to the sitting rather than to the question.
//
// THE FORM IS FIRST, which is load-bearing rather than aesthetic: formCell reads
// a footer entry index straight back as a grid row, so anything above it would
// silently shift every cell. It is also the order of value that fitFooter drops
// from: the bar goes first, then the panel, then grid rows.
//
// A BOARD CAN END UP IN A FOOTER THAT DROPS ROWS, and D15's "never" was measured
// wrong (R11). It holds at SELECTION — `packBoards` refuses a board the terminal
// cannot draw whole — and a resize afterwards is a shape nobody chose.
//
// What the order buys is that the losses are SURVIVABLE in sequence: the bar (a
// figure), the panel (cosmetic), then grid rows. An earlier version of this
// comment called them "harmless", which was checked against the CLICK map —
// FooterRowAt answers nothing for a row that was never painted — and was false
// of the SWEEP, which does not go through that map at all: Enter took every
// unmarked word including ones the window never drew. That is why Enter is now
// held while the board is not whole (R17), and why a safety word has to name the
// path it was checked on.
//
// The one thing that must not go is the statement of what a click will MEAN, and
// that is why the mode moved to the prompt row, which Paint clips last.
//
// THE PALETTE is threaded in rather than reached for, on the same seam
// `boardPalette` sits on: `main` owns the terminal's colours and the form takes
// finished sequences. It styles only the BAR — the grid above it is the board's
// own rendering, already painted through `play.Palette` (#44).
func boardFooter(q play.Question, fig sittingFigures, pal palette) []string {
	return append(strings.Split(q.Prompt(), "\n"), asChrome(sittingBar(fig), pal))
}

// barRows is the ONE row the bar is guaranteed below the board.
//
// A minimum rather than a measurement, and that asymmetry with promptRows is
// deliberate. `fitFooter` drops whole rows from the END, and the board's own
// rows come FIRST — so if the bar turns out to need three rows on a narrow
// terminal, the bar is what gets dropped and the board is still whole. That is
// the sacrifice order D10 wanted, and it means the bar's real height cannot cost
// the board anything. The prompt is different: it sits ABOVE the footer and
// takes its share off the top, so its real height has to be charged.
const barRows = 1

// fitsABoard reports whether a terminal this tall can draw a board of boardRows
// WHOLE, given a keys prompt of promptRows (D15).
//
// A board that cannot be drawn whole is not a board, so the words go to form 2.3
// for that sitting instead — which is a complete answer rather than a degraded
// one. The alternative was a floor in fitFooter, and it would have broken the
// budget Paint rests on: footerRows would exceed what the prompt left, the
// terminal would scroll to fit it, and a click at viewport row R would stop
// meaning the word drawn there. Refusing to offer the board keeps fitFooter's
// guarantee true rather than negotiating with it.
//
// THE CHROME GAP IS NOT A TERM HERE, and that is a PROOF rather than an
// oversight (#44 PQ-8). `grantedGap` hands out the row only when two rows survive
// past the prompt and the footer, so whenever the gap exists this sum already had
// slack for it: `T-P-F >= 2` gives `F+P+1 <= T-1`, and whenever it does not exist
// the sum is unchanged. Charging it here would therefore alter no answer while
// LOOKING like the two consumers had been reconciled — and at `{T:8, F:7, P:1}`,
// the exact row the table below pins, a naive charge would refuse a board that
// `Paint` goes on to draw whole, swapping the keys row for `boardRefusal` and
// holding Enter over a grid with every cell on screen.
//
// `fitFooter` gets its budget before the gap does, so the gap can never cost the
// board a row either: a board this says is whole is a board the footer had room
// for. TestTheChromeGapNeverChangesWhetherABoardFits is the pin, because "these
// two formulas agree" is not a fact anyone will re-derive by eye.
//
// promptRows is MEASURED and passed in, because a constant here was a second
// owner of a height `displayRows` already computes. It was 1, and the board's
// keys line is seventy-six columns while the board was offered from twenty — so
// on a narrow terminal `fitFooter` silently dropped the bar, then the panel, and
// then — while the live mark still had a footer row of its own, before R11 moved
// it onto the prompt — the one statement of what the next click would mean,
// while every mark is irreversible.
func fitsABoard(termRows, boardRows, promptRows int) bool {
	return boardRows+promptRows+barRows <= termRows
}

// formCell offers a click to the form on screen and reports which of its cells
// was hit, if any.
//
// TWO questions, and each is asked of the only thing that can answer it. The
// SCREEN says which footer entry the pointer was on, because the screen laid the
// footer out; the FORM says which cell is at that spot, because the form decided
// where its words are printed. The loop knows neither and does the subtraction
// between them — the grid is drawn as the FIRST footer entries, so a footer row
// below Rows() is the bar rather than a word.
//
// False for every form that is not a grid, which is every form but the board,
// and false is what leaves #38's behaviour exactly as it was.
func formCell(view display, q play.Question, k Key) (int, bool) {
	g, ok := q.(play.Grid)
	if !ok {
		// Also the nil case, at the end of a queue: a nil Question is not a Grid.
		return 0, false
	}
	row, offset, ok := view.FooterRowAt(k.Row)
	if !ok {
		return 0, false
	}
	// A CONTINUATION ROW IS NOT A TARGET (R9).
	//
	// `k.Col` is a column of the TERMINAL, and it only means a column of the
	// form's own line while that line is drawn on one physical row. Once an entry
	// wraps, column 4 of its second row is column cols+4 of the line, and acting
	// on it lands a permanent mark on whatever word happens to sit at column 4.
	//
	// The board keeps its rows fitting by relaying out on resize, so this should
	// never fire — which is exactly why it is here. That guarantee lives in
	// another package and depends on the loop remembering to pass the resize on;
	// a click is irreversible, and "should never happen" is not a thing to bet
	// one on.
	if offset != 0 {
		return 0, false
	}
	// NO SECOND BOUND HERE. A `row >= g.Rows()` guard was written first and a
	// mutation showed it changed nothing: CellAt already refuses a row past the
	// grid, because the grid is the thing that knows how tall it is. Two owners
	// of one bound is how a chrome row comes to be a cell on the day one of them
	// is edited.
	return g.CellAt(row, k.Col)
}

// sittingWords is how many WORDS the sitting will ask about, which is not the
// same as how many questions it holds (D8).
//
// `fig.done` was already a word count — every mark scores — so a board made the
// bar compare words against slots, and a twenty-word sitting with one board in
// it read "0 of 2". The budget is untouched: schedule.Queue returns that many
// keys whatever they are packed into. This is the number on screen, and it is
// the number the whole load argument is about.
func sittingWords(qs []play.Question) int {
	n := 0
	for _, q := range qs {
		if b, ok := q.(play.Batch); ok {
			n += b.Words()
			continue
		}
		n++
	}
	return n
}

// relearnLine is what a board leaves in the transcript as it closes (D10).
//
// The live edge is ephemeral by design — that is what bought marks that change
// as they land — and for a grid that is right, because a grid of sixteen words
// is not something to scroll back to. The OUTCOME is worth keeping: the words
// marked `no` are the ones the sitting was actually about.
//
// Built from the outcomes the loop RECORDED rather than from the board's marks,
// and that is not an arbitrary choice between two equal sources: it makes the
// transcript name exactly what reached the log. A line assembled from the form
// could disagree with the events, and a transcript that disagrees with the log
// is worse than no transcript.
func relearnLine(words []string) string {
	if len(words) == 0 {
		return ""
	}
	return "relearn: " + strings.Join(words, ", ")
}

// boardBox is the box at which a word becomes eligible for the board (D4).
//
// A STARTING NUMBER, not a derived one, and the only figure in this issue with
// no argument under it. What is argued is the shape of the risk: a wrong `Yes`
// sends a word to box+1 where a real test would have sent it to box/2, so the
// cost is the DELAY it buys — 0 days at box 0, 4 at box 3, 165 at box 10 — while
// the CHANCE of a wrong yes falls as the box rises. The product peaks in the
// middle, around boxes 5-7, which no single floor expresses well.
//
// Boxes 0 and 1 are literally free: the ladder waits one day at both, so a
// wrongly promoted new word comes back tomorrow regardless. Three is past those
// two free rungs, at a four-day interval and roughly three recalls of history.
//
// It is meant to be REPLACED BY EVIDENCE rather than by argument, which is what
// ReviewEvent.Form (D4a) is for. The operator confirmed the floor stands until
// the log can answer.
const boardBox = 3

// packBoards splits triage words into the largest boards this terminal can draw
// WHOLE, and returns the ones it cannot draw at all.
//
// A CHUNK THAT DOES NOT FIT SHRINKS rather than going back to being asked one at
// a time, which is `#40` D15 widened rather than replaced. D15 sent an unfittable
// chunk to form 2.3 — a complete answer then, and not available now for the words
// `#42` newly routes here, since they arrive precisely because no 2.3 could be
// built. Shrinking is monotone, so the search terminates and finds the largest
// fit: fewer words never need MORE rows, because `cols` is capped at four and a
// subset's longest word is no longer than the whole's.
//
// SO THE LEFTOVER CASE IS ALL-OR-NOTHING, which is worth knowing rather than
// discovering. A one-word board's height does not depend on the word — one grid
// row plus two of chrome, and the word is truncated into the width — so either
// this terminal can draw a board or it can draw none, and `undrawable` is empty
// or everything.
//
// The caller owes those words a form: form 2.3 where one can be built, and a skip
// only where neither is possible. Returning them rather than skipping them here is
// what keeps that decision at the one place that has the entries (#42 PQ-1).
func packBoards(words []string, opt options) (boards [][]string, undrawable []string) {
	for len(words) > 0 {
		n := min(len(words), play.MaxBoardWords)
		for n > 0 && !boardFits(words[:n], opt) {
			n--
		}
		if n == 0 {
			return boards, words
		}
		boards = append(boards, words[:n])
		words = words[n:]
	}
	return boards, nil
}

// boardFits reports whether this terminal can draw a board of these words whole.
//
// It builds a PROBE and asks it TWO questions, because both heights belong to
// something else: the board owns its layout — how many columns fit, and
// therefore how many rows — and `displayRows` owns how tall a line is once the
// terminal has wrapped it. Neither is re-derived here.
//
// Glosses are left out of the probe: the panel is one row whatever it says, so
// they cannot change the answer.
//
// The width check is separate and blunt: below minWrapWidth this program already
// treats the terminal as too narrow to lay text out at all, and a board there
// would be columns of truncated stubs.
func boardFits(words []string, opt options) bool {
	cells := make([]play.Cell, len(words))
	for i, w := range words {
		cells[i] = play.Cell{Word: w}
	}
	// NO PALETTE on the probe: escape sequences cost no columns, so they cannot
	// change how tall the board is, which is the only thing being asked here.
	// THE SAME QUESTION THE FRAME WILL ASK, through the same helper — so a board
	// offered at selection is one the draw agrees is whole, and neither can
	// acquire a rule the other lacks.
	return boardFitsIn(play.NewBoard(cells, opt.width, play.Palette{}), opt.rows, opt.width)
}

// todaysQuestions builds the queue: fold the log, ask the schedule, render each
// word's definition.
//
// It RETURNS the deck and the folded progress rather than discarding them, and
// that is the whole of D7. Deck() reads one file per WORD and Events() one file
// per DAY of history, so a sitting that recomputed its cost figures per answer
// would pay ~5,700 file reads per question on a 5,000-word deck with two years
// of log — with a person waiting. These two reads are the sitting's only ones,
// and everything after them happens in memory.
func todaysQuestions(d deps, opt options, stdout, stderr io.Writer) ([]play.Question, *sittingDeck, int) {
	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(stderr, "define: could not read the deck: %v\n", err)
		return nil, &sittingDeck{}, 1
	}
	events, err := d.deck.Events(anyTime)
	if err != nil {
		// A log we cannot read means no progress, which makes every word look
		// new — wrong, but reviewing the wrong order beats refusing to review.
		// The sitting still runs, and its reviews are still recorded; only the
		// cost figures are computed against an empty history.
		fmt.Fprintf(stderr, "define: could not read the review log (%v); treating every word as new\n", err)
	}
	held := &sittingDeck{deck: deck, prog: schedule.Fold(events), marks: map[string]clickable{}}
	now := d.clock.Now()
	keys := schedule.Queue(deck, held.prog, now, opt.count)
	if len(keys) == 0 {
		fmt.Fprintln(stdout, emptyQueueReason(len(deck), opt.count))
		return nil, held, 0
	}

	// The distractor pool for the whole sitting, built ONCE. Per-question it
	// would be one dictionary lookup per deck word per due word — quadratic in a
	// deck that only grows (ARCH-CONSTRAINTS).
	//
	// Seeded on the DAY, so today's sitting draws the same sample whichever
	// order the words come up in, and a different one tomorrow.
	day := now.Format("2006-01-02")
	pool := buildPool(d, deck, seedFor("pool", day))

	// THE FORM IS PICKED AFTER THE LOOKUP, and that is #42's structural change.
	//
	// Selection ran on KEYS, before anything was fetched, because the box was
	// all it consulted. The rule is now one sentence — **2.3 tests you, the board
	// triages you** — and its second half is a question about the ENTRY: whether
	// the deck can build a real test out of it. That is not knowable until the
	// entry is parsed, so selection moved here.
	//
	// A MATURE WORD IS NEVER LOOKED UP FOR ITS OPTIONS. Its box alone sends it to
	// a board (D4), so asking `optionsFor` would be pool work for a form it will
	// not take. It is looked up once below, for its gloss, exactly as before.
	var qs []play.Question
	marks := map[string]clickable{}
	var triage []string
	// unaskable counts words dropped for a reason that is NOT a lookup failure, so
	// an empty sitting can name the cause it actually established rather than the
	// only cause that used to exist.
	var unaskable int
	// The entries parsed on THIS pass, so a word that changes hands is not looked
	// up twice. A young word that turns out to be untestable goes to the board
	// loop, which needs the same entry for its gloss.
	parsed := map[string]Entry{}

	// ask builds form 2.3 for a word whose entry is already in hand, or reports
	// that it cannot. ONE place renders, so the click map and the question cannot
	// be built from different strings.
	ask := func(key string, entry Entry) play.Question {
		opts := optionsFor(key, entry, pool, seedFor(key, day))
		if opts == nil {
			return nil
		}
		// THE REGIONS, kept rather than discarded (D7). `play` must never see
		// them — it is mechanically guarded pure and `Region` lives in main — so
		// the loop keeps its own word→regions map, built here, where the entry is
		// rendered and the coordinates are true.
		rendered, rs := Render(entry, RenderOpts{
			// Word is IDENTITY, not presentation, and RenderOpts says so: a
			// click on the headword replays the word the deck holds, and
			// deriving it from the entry instead lets the two disagree —
			// `jalapeno` in the deck against `jalapeño` on the head line, for
			// which the CDN answers different URLs.
			Word:  key,
			Color: opt.color, Width: opt.width, Vocab: vocabularyFor(d, opt),
		})
		marks[key] = clickable{text: rendered, regions: rs}
		return play.NewChoice(key, rendered, opts)
	}

	for _, key := range keys {
		if held.prog[key].Box >= boardBox {
			triage = append(triage, key)
			continue
		}
		text, err := d.dict.Lookup(key)
		if err != nil {
			// A word in the deck the dictionary no longer knows. Skip it rather
			// than failing the session: the other words are still worth review.
			fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
			continue
		}
		entry := ParseEntry(text)
		// AN AUTHORED ITEM BEATS A DEFINITION MATCH, which is the whole of #12's
		// clause on the selection rule.
		//
		// #10 exists because a definition match is the WEAKER test — it asks
		// which gloss belongs to a word, where a cloze asks which word belongs to
		// a sentence. The item was authored to be the better question, so
		// preferring 2.3 when both are available would make #10 decoration.
		//
		// The board still triages mature words above (#42's rule, unchanged):
		// that rule was hard-won and changing it is a different issue with its
		// own evidence.
		if q := clozeAsk(d, opt, key, entry, marks, day, stderr); q != nil {
			qs = append(qs, q)
			continue
		}
		if q := ask(key, entry); q != nil {
			qs = append(qs, q)
			continue
		}
		// Young, and no real test can be built for it (`fallbackReasons`). It is
		// TRIAGED rather than asked to rate itself — which is #42 — and the
		// entry travels with it so the board loop does not re-fetch.
		parsed[key] = entry
		triage = append(triage, key)
	}

	boards, undrawable := packBoards(triage, opt)

	// A TERMINAL TOO SHORT FOR ANY BOARD, which is all-or-nothing (see
	// packBoards). D15's fallback holds for the words it always covered: form 2.3
	// where one can be built. Only a word that can be neither drawn nor tested is
	// skipped, and it says which — both conditions, because both must hold.
	for _, key := range undrawable {
		entry, ok := parsed[key]
		if !ok {
			text, err := d.dict.Lookup(key)
			if err != nil {
				fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
				continue
			}
			entry = ParseEntry(text)
		}
		if q := ask(key, entry); q != nil {
			qs = append(qs, q)
			continue
		}
		unaskable++
		fmt.Fprintf(stderr, "define: skipping %q: this window is too short to draw a board "+
			"and no multiple choice can be built for it\n", key)
	}

	// THE BOARDS, and they are the only questions that need no render: nothing
	// about a board reaches the buffer, so there is no click map to build and no
	// definition to wrap. One gloss each — for the panel — is the whole of what
	// the form takes, and targetCandidate is the same sense form 2.3 asks about.
	//
	// A word the dictionary no longer knows is skipped exactly as it is above.
	// The board only gets SHORTER for it, so the fit packBoards already checked
	// still holds.
	for _, chunk := range boards {
		var cells []play.Cell
		for _, key := range chunk {
			entry, ok := parsed[key]
			if !ok {
				text, err := d.dict.Lookup(key)
				if err != nil {
					fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
					continue
				}
				entry = ParseEntry(text)
			}
			var gloss string
			if c, ok := targetCandidate(key, entry); ok {
				gloss = c.Gloss
			}
			cells = append(cells, play.Cell{Word: key, Gloss: gloss})
		}
		if len(cells) > 0 {
			qs = append(qs, play.NewBoard(cells, opt.width, boardPalette(opt)))
		}
	}
	if len(qs) == 0 {
		// NOT "nothing due today": words WERE due. Saying nothing is due would
		// send the learner away believing their deck is clear when it is not.
		//
		// AND THE CAUSE IS THE ONE THE CODE ESTABLISHED. This said "none could be
		// looked up" for every empty sitting, which was true while a failed lookup
		// was the only way to drop a word — and #42 added a second: a word the
		// dictionary answers fine, for which no test can be built and which this
		// window cannot draw a board for. Reporting a dictionary failure then
		// sends the learner to check their dictionary about a window that is too
		// narrow. The population is exactly this issue's subject: a young deck on
		// a narrow terminal, which ran a full sitting before #42.
		if unaskable > 0 && unaskable == len(keys) {
			fmt.Fprintf(stderr, "define: %d words are due but none can be asked in this "+
				"window — make it taller or wider\n", len(keys))
		} else {
			fmt.Fprintf(stderr, "define: %d words are due but none could be looked up\n", len(keys))
		}
		return nil, held, 1
	}
	held.marks = marks
	return qs, held, 0
}

// clickable is one entry's rendered text and the spans in it a click can act on.
//
// The TEXT is kept beside the regions because their coordinates are relative to
// it, and the loop writes something LARGER — a form's reveal prepends its own
// lines. Finding the render inside the reveal is how the offset is computed, and
// keeping both is what makes that possible without the form giving up ownership
// of its own output.
type clickable struct {
	text    string
	regions []Region
}

// sittingDeck is the deck as ONE sitting sees it: read once at the start and
// kept in memory for the rest of it (D7).
//
// Passed by POINTER everywhere. Its mutators have pointer receivers while the
// value was being copied into the loop, so `prog` was shared and `deck` was
// half-shared — a drop reached the caller only because slices.DeleteFunc
// compacts the backing array in place. Nothing production depended on that, and
// a test did (BR-5); `#40` would have been the second consumer to meet it.
//
// The point is not caching. It is that the same transition schedule.Fold applies
// to a logged event is applied HERE the instant an answer lands — so the cost
// the bar shows during a sitting and the cost the next sitting derives from the
// log are the same number by construction, not by two pieces of code agreeing.
type sittingDeck struct {
	deck []store.Word
	prog map[string]schedule.Progress
	// marks is what each word's rendered entry OFFERS, keyed by store.Key —
	// built once where the entries are rendered, because that is the only place
	// the coordinates are true (D7).
	marks map[string]clickable
}

// answered applies one graded answer, exactly as folding the event it produced
// would. schedule.GradeOf is the shared rule; capture.go writes the same two
// booleans into the log.
func (sd *sittingDeck) answered(out play.Outcome, at time.Time) {
	key := store.Key(out.Word)
	if key == "" {
		return
	}
	if sd.prog == nil {
		// A sitting whose log read failed folds to an empty map, and a caller
		// that built this value itself has none. Filling it here rather than
		// returning early: a silent no-op would make the figures look computed.
		sd.prog = map[string]schedule.Progress{}
	}
	sd.prog[key] = schedule.Answer(sd.prog[key], schedule.GradeOf(out.Verdict == play.Correct, out.Unaided), at)
}

// dropped removes a word the learner curated away mid-sitting.
//
// Without this the bar would keep charging the deck for a word that is no longer
// in it, and disagree with the deck the learner just edited. The store's Forget
// is the durable half; this is the same edit to the copy in hand.
func (sd *sittingDeck) dropped(word string) {
	key := store.Key(word)
	sd.deck = slices.DeleteFunc(sd.deck, func(w store.Word) bool { return store.Key(w.Text) == key })
}

// marksIn is the clicked spans for a word, moved into the coordinates of the
// text about to be WRITTEN.
//
// A form's reveal is larger than the render it contains — `Choice.Reveal()` puts
// the correct option and the learner's pick above it — so every region's Line
// moves by the number of lines before the render begins. Located rather than
// counted from a formula: the form owns its own layout, and a formula here would
// be a second copy of it that a new form silently invalidates.
func (sd *sittingDeck) marksIn(word, written string) []Region {
	c, ok := sd.marks[store.Key(word)]
	if !ok || c.text == "" || len(c.regions) == 0 {
		return nil
	}
	at := strings.Index(written, c.text)
	if at < 0 {
		// The render is not in what is being written — a form that reworded its
		// reveal, or a word with no entry. No regions rather than wrong ones.
		return nil
	}
	off := strings.Count(written[:at], "\n")
	out := make([]Region, len(c.regions))
	for i, r := range c.regions {
		r.Line += off
		out[i] = r
	}
	return out
}

// promptRegions is what a form's PROMPT offers to a click.
//
// LOCATED, NOT ASSUMED — the rule marksIn states thirty lines up ("the form owns
// its own layout, and a formula here would be a second copy of it that a new
// form silently invalidates"), applied to the one place that was still using the
// formula.
//
// The formula was `Choice.Prompt()`'s shape read as every form's: line 0 is the
// headword, at column 0, as wide as the word. Cloze's prompt is a blanked
// sentence, so the region landed on the sentence's first eleven cells — and a
// click there SPOKE THE ANSWER, while the underline advertised its length. That
// is precisely what Blank exists to prevent, arriving by a path Blank cannot see.
// Cloze's own doc comment said the premise did not hold here; saying it is not
// the same as acting on it.
//
// So the region is issued only when the claim it makes is TRUE. The predicate is
// the region's own coordinates read back as a sentence — "line 0 begins with the
// headword, and the first visibleCells(word) cells of it are that word" — which
// is why TestAPromptRegionCoversTheTextItClaims can assert exactly the same
// thing over every form without knowing which forms have one.
//
// Searching the whole prompt instead would be WORSE than the formula: a cloze
// prompt does contain its answer, among the options, so "find the word" would
// underline the correct option. The claim is about a POSITION, so a position is
// what gets checked.
//
// Nothing to worry about with wrapping: a headword is never wide enough to wrap.
func promptRegions(q play.Question) []Region {
	word := q.Word()
	if word == "" {
		return nil
	}
	line0, _, _ := strings.Cut(q.Prompt(), "\n")
	if !strings.HasPrefix(line0, word) {
		return nil
	}
	// Line 1, not 0: the write leads with a blank line, and addRegions anchors at
	// the line the write STARTS on.
	return []Region{{
		Kind: RegionHeadword, Text: word, Word: word,
		Line: 1, Col: 0, Width: visibleCells(word),
	}}
}

// figures is what the bar and the summary are built from: a walk over the deck
// slice, no IO. A few thousand iterations of at most twenty integer
// multiplications, which is why it is charged per ANSWER rather than per frame —
// there is simply no reason to redo it more often.
func (sd *sittingDeck) figures(budget int) sittingFigures {
	return sittingFigures{
		load:   schedule.DailyLoad(sd.deck, sd.prog),
		fresh:  schedule.SustainableNewWords(budget, sd.deck, sd.prog),
		budget: budget,
	}
}

// anyTime is the zero time, which Events(since) treats as "everything".
//
// Spelled directly rather than as store.Word{}.FirstSeen, which was a way of
// saying time.Time{} that made a reader chase a struct field to learn nothing.
var anyTime time.Time

// livePrompt is what to press RIGHT NOW: the frame's prompt row, rewritten on
// every keystroke and never filed in the buffer.
//
// It replaces the old draw(), which wrote the question, the reveal AND the keys
// on every call. Those three had different lifetimes all along — the question
// and the answer are the transcript, the keys are the live edge — and a
// scrolling terminal was simply unable to express the difference (D4).
//
// The GRADING keys are offered whether or not the definition is showing. That
// line used to appear only AFTER a reveal, and an unrevealed word said "Enter or
// space to reveal" instead — so every correct answer cost a keystroke that
// carried no information, and the slow one at that, since a reveal fetches and
// plays the pronunciation (#24).
func livePrompt(s play.Session) string {
	q := s.Current()
	if q == nil {
		// Between the last answer and the summary. An empty prompt is a frame
		// with nothing to press, which is the truth for that moment.
		return ""
	}
	if s.Graded {
		// Answered, and the answer is on screen. The only thing left is to read
		// it and move on — offering y/n here would invite a second verdict on a
		// question that already has one.
		//
		// EXCEPT the flag, which is not a second verdict and is most useful
		// exactly here: a learner discovers a question was broken by reading the
		// reveal. Derived from the FORM rather than a constant, because "any key
		// = next word" is a lie on a form where `?` does something else — the
		// same class gradePrompt below was created to fix.
		return gradedPromptFor(q)
	}
	return gradePrompt(q)
}

// gradedPromptFor is the post-answer line, naming the flag when the form has one.
//
// A function rather than the const it used to be, for the reason gradePrompt is:
// a keys line promising a key that does nothing — or omitting one that does
// something — is a bug no test could see, because every test types the keys the
// line names.
func gradedPromptFor(q play.Question) string {
	if play.CanFlag(q) {
		return "any key = next word, " + flagKeys + sessionKeys
	}
	return gradedPrompt
}

// The two prompt lines livePrompt returns, named because README.md quotes them
// VERBATIM and doc_sync_test.go pins that — a hand-maintained restatement of a
// fact the code owns will drift, so the restatement is made to derive.
//
// Three findings in the `doc-sweep-incomplete` family said the same thing about
// this exact line: the flow reversal reached the README table and not the form's
// doc comments, then reached the doc comments and not the two test citations.
// Sweeping is what kept failing; a consumer that fails the build does not.
const (
	// quitKey is the one reserved key true of EVERY form without exception,
	// which is what earned it a name of its own (#40 D12).
	quitKey = "Ctrl-C to stop"
	// sessionKeys are the keys the SESSION reserves, true of every form that
	// holds ONE word (question.go:74-77). The form's own keys are prepended by
	// gradePrompt — this half does not vary per form, and a form restating it
	// would be two owners of one fact.
	sessionKeys = "d = remove from deck, " + quitKey
	// gradedPrompt is shown once the answer is in and the definition is up.
	// DERIVED from the pair above, so the wording cannot drift between them.
	gradedPrompt = "any key = next word, " + sessionKeys
	// flagKeys names the bad-question gesture for a form that has one (#12).
	// After a verdict is when a learner discovers a question was broken, so this
	// is the state where naming it matters most — and "any key = next word" is
	// actively wrong there, because `?` does something else.
	flagKeys = string(play.FlagKey) + " = bad question, "
)

// reservedKeys is the session's own half of the prompt, for THIS form.
//
// `d` NAMES NO WORD on a form holding many — a grid has no single current word,
// so Apply refuses the drop rather than guessing which cell it meant (#40 D12).
// Offering it here would be the exact bug gradePrompt was created to fix: a keys
// line promising a key that does nothing. The reserved set shrank for one kind
// of form, so the sentence about it had to stop being a constant.
func reservedKeys(q play.Question) string {
	if _, ok := q.(play.Batch); ok {
		return quitKey
	}
	return sessionKeys
}

// gradePrompt is what to press while a verdict is still owed: the FORM's answer
// keys, then the session's reserved ones.
//
// A function rather than the const it used to be. The const spelled form 2.1's
// y/n, so the moment a second form shipped the learner was being told to press a
// key that did nothing — a bug no test could see, because every test typed the
// keys the const named.
func gradePrompt(q play.Question) string {
	// No nil guard: draw returns before this when Current() is nil, so a nil
	// here would be a bug in the loop rather than a state to render politely.
	// The guard that was here shipped as dead code and would have hidden that.
	return q.Keys() + ", " + reservedKeys(q)
}

// finish prints the sitting's score AND what the deck now costs per day.
//
// The cost line is the number that should govern how many new words a learner
// takes on, and before #39 it was computed nowhere and shown nowhere — a
// growing backlog was the only way to discover it, which is the worst possible
// feedback loop for a tool whose entire subject is spaced feedback.
//
// IT NO LONGER READS ANYTHING, and #39 T7's reasoning is what survives the
// change. That version re-read the deck and the log here on the grounds that
// "the answers just given have changed every box involved, so the figure the
// learner should see is the one AFTER today". True, and the loop's in-memory
// copy already IS that figure: it is updated by the same schedule.Answer the
// fold applies (D7). #39 had no such copy to use, which is what made the re-read
// the only way rather than the wrong way — and it was also the second `Deck()`
// and second `Events()` in a sitting that claims to pay for one of each.
//
// The failure branches go with the reads. There is nothing left here that can
// fail, so a summary can no longer cost a sitting whose reviews are recorded.
func finish(w io.Writer, s play.Session, fig sittingFigures) int {
	fmt.Fprintf(w, "\n%d right, %d wrong\n", s.Right, s.Wrong)
	// Through the SHARED formatter, so this line and the pinned bar cannot
	// describe the same deck differently or word the -count assumption two ways.
	// The budget is -count: the number of questions a sitting asks, which is the
	// DAILY budget for a learner who sits down once a day, and it is NAMED in
	// the line rather than hidden so someone who sits twice knows to double it.
	fmt.Fprintln(w, sittingSummary(fig))
	return 0
}

// emptyQueueReason names WHY the sitting is empty.
//
// "Nothing due today" is a statement about the SCHEDULE, and it was being
// printed for three different situations: an empty deck, a budget of zero, and
// the schedule genuinely having nothing. The first two are the learner's own
// input coming back at them wearing the schedule's clothes — `--play -count 0`
// reported a clear deck while every word in it was outstanding (BR-46).
//
// Same rule as the all-lookups-fail branch above: when the sitting offers fewer
// words than the deck made due, the message names the actual cause, and this
// sentence is reserved for the case where the cause really is the schedule.
func emptyQueueReason(deckSize, budget int) string {
	switch {
	case deckSize == 0:
		return "define: the deck is empty — look a word up first and it will be scheduled"
	case budget == 0:
		return "define: -count 0, so no words were offered"
	default:
		return "define: nothing due today"
	}
}
