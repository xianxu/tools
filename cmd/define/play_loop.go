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
		if _, ok := q.(play.Grid); ok {
			view.Draw(livePrompt(s), boardFooter(q, fig))
			return
		}
		if q != nil && written != s.Index {
			written = s.Index
			// Plain \n: the screen places every row, so nothing here decides
			// where a line goes (D1).
			//
			// THE PROMPT WORD IS CLICKABLE (T4). Both forms put the headword on
			// their first line at column 0 — `Recall.Prompt()` IS the word, and
			// `Choice.Prompt()` is the word, a blank, then the options — and the
			// leading "\n" of this write puts it on line 1. That is the whole
			// region-finding problem for a prompt: nothing to search for, no
			// offsets to survive a wrap, because a headword is never wide enough
			// to wrap.
			// Line 1, not 0: the write leads with a blank line, and addRegions
			// anchors at the line the write STARTS on.
			writeRendered(stdout, "\n"+q.Prompt()+"\n", []Region{{
				Kind: RegionHeadword, Text: q.Word(), Word: q.Word(),
				Line: 1, Col: 0, Width: visibleCells(q.Word()),
			}})
		}
		// The grading keys are the PROMPT and the bar is the FOOTER, which gets
		// the order of sacrifice right for free (D3): Paint clips the prompt last
		// and drops footer rows first, and a learner who cannot see the keys
		// cannot answer at all, while one who cannot see their daily load loses
		// nothing this minute.
		view.Draw(livePrompt(s), []string{sittingBar(fig)})
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
			// AND THE FORM, if it lays itself out (R9). A board built for eighty
			// columns has 74-column rows; at forty the terminal wraps each into
			// two, a footer entry stops being one physical row, and a click on
			// the continuation carries a column that means another word. Its
			// marks are already in the log and cannot be retracted, so the board
			// is relaid out rather than replaced.
			//
			// `sz.cols` rather than `opt.width`: this is the width the SCREEN
			// paints at, and the board has to agree with the paint, not with a
			// wrap policy that answers 0 on a narrow terminal.
			if g, ok := s.Current().(play.Grid); ok {
				g.Resize(sz.cols)
				// AND NO RE-SELECTION. The board stays, at the new shape, even
				// if the terminal is now too short to draw it whole — which is
				// D15's rule holding rather than bending. `boardsFor` chooses
				// the form for words that have not been asked yet; this board's
				// marks are already in the log and cannot be retracted, so
				// "send it to 2.3 instead" would mean re-asking words already
				// answered. What a too-short terminal loses is listed in
				// boardFooter, in the order it loses it, and the row that says
				// what a click means is not in the footer at all.
			}
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
					// The record-shaped indicator, not the editor's erasable one:
					// a sitting's `♫ playing 3×` is a frame write like any other
					// (D5a), and defaultIndicator is what every other playback on
					// this path already uses.
					playRegion(ctx, d, opt, r, "", defaultIndicator(opt), stdout, stderr)
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
						defaultIndicator(opt), stdout, stderr)
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
			return play.Input{Kind: play.InputDrop}, true
		}
		return play.Input{Kind: play.InputRune, Rune: k.Rune}, true
	}
	return play.Input{}, false
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
// wrong (R11). It holds at SELECTION — boardsFor refuses a board the terminal
// cannot draw whole — and a resize afterwards is a shape nobody chose. What the
// order buys is that the losses are harmless in sequence: the bar (a figure), the
// panel (cosmetic), then grid rows, which are conspicuously absent and, because
// they were never painted, are not clickable either (FooterRowAt answers nothing
// for a row fitFooter dropped). The one thing that must not go is the statement
// of what a click will MEAN, and that is why the mode moved to the prompt row,
// which Paint clips last.
func boardFooter(q play.Question, fig sittingFigures) []string {
	return append(strings.Split(q.Prompt(), "\n"), sittingBar(fig))
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
// promptRows is MEASURED and passed in, because a constant here was a second
// owner of a height `displayRows` already computes. It was 1, and the board's
// keys line is seventy-six columns while the board was offered from twenty — so
// on a narrow terminal `fitFooter` silently dropped the bar, then the panel, and
// at twenty-five columns the TOGGLE, which is the one owner of which mark is
// live while every mark is irreversible.
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
// below Rows() is the toggle or the bar rather than a word.
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
	// of one bound is how the toggle row comes to be a cell on the day one of
	// them is edited.
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

// boardsFor splits today's keys into the ones swept on a board and the ones
// asked one at a time (D4).
//
// SINGLES FIRST, THEN BOARDS, and the order is a choice rather than an
// accident. Packing is the point of the whole form — sixteen words for sixteen
// keystrokes is what makes a large deck affordable — and packing cannot preserve
// the queue's interleaving, because a board formed from words scattered through
// the queue has to sit somewhere. So retrieval gets the learner's freshest
// attention and the maintenance sweep comes after.
//
// The counter-argument is real and now measurable: a tired learner marks
// everything yes, which is the illusion-of-knowing the Spec worries about.
// `ReviewEvent.Form` is what will eventually say whether it happens.
//
// A CHUNK THAT WILL NOT FIT GOES BACK TO SINGLES (D15). A board that cannot be
// drawn whole is not a board, and form 2.3 is a complete answer rather than a
// degraded one.
func boardsFor(keys []string, prog map[string]schedule.Progress, opt options) (single []string, boards [][]string) {
	var eligible []string
	for _, k := range keys {
		if prog[k].Box < boardBox {
			single = append(single, k)
			continue
		}
		eligible = append(eligible, k)
	}
	for len(eligible) > 0 {
		n := min(len(eligible), play.MaxBoardWords)
		chunk := eligible[:n]
		eligible = eligible[n:]
		if boardFits(chunk, opt) {
			boards = append(boards, chunk)
			continue
		}
		single = append(single, chunk...)
	}
	return single, boards
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
	if opt.width < minWrapWidth {
		return false
	}
	cells := make([]play.Cell, len(words))
	for i, w := range words {
		cells[i] = play.Cell{Word: w}
	}
	probe := play.NewBoard(cells, opt.width)
	// THE PROMPT THE LOOP WILL ACTUALLY DRAW, measured at this width — the same
	// expression livePrompt returns for this form, so the two cannot disagree.
	return fitsABoard(opt.rows, probe.Rows(), displayRows(gradePrompt(probe), opt.width))
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

	// THE BOX PICKS THE FORM (D4), and this is the first time anything in this
	// program has consulted one to choose HOW to ask. Selection was a capability
	// question until now — form 2.3 when the deck can supply distractors, 2.1
	// when it cannot — and nothing looked at a box at all.
	single, boards := boardsFor(keys, held.prog, opt)

	var qs []play.Question
	marks := map[string]clickable{}
	for _, key := range single {
		text, err := d.dict.Lookup(key)
		if err != nil {
			// A word in the deck the dictionary no longer knows. Skip it rather
			// than failing the session: the other words are still worth review.
			fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
			continue
		}
		entry := ParseEntry(text)
		// THE REGIONS, kept rather than discarded (D7). `play` must never see
		// them — it is mechanically guarded pure and `Region` lives in main — so
		// the loop keeps its own word→regions map, built here, where the entry is
		// rendered and the coordinates are true.
		rendered, rs := Render(entry, RenderOpts{
			// Word is IDENTITY, not presentation, and RenderOpts says so: a
			// click on the headword replays the word the deck holds, and
			// deriving it from the entry instead lets the two disagree —
			// `jalapeno` in the deck against `jalapeño` on the head line, for
			// which the CDN answers different URLs. Empty means "no click map
			// wanted", which was true of `--play` until this issue.
			Word:  key,
			Color: opt.color, Width: opt.width, Vocab: vocabularyFor(d, opt),
		})
		marks[key] = clickable{text: rendered, regions: rs}
		// Form 2.3 when the deck can supply distractors, form 2.1 when it
		// cannot (D9). A young deck is a NORMAL state, not an error, and the
		// fallback is invisible to the learner — the sitting stays the length
		// the schedule asked for either way.
		if q := choiceFor(key, rendered, entry, pool, seedFor(key, day)); q != nil {
			qs = append(qs, q)
			continue
		}
		qs = append(qs, play.NewRecall(key, rendered))
	}
	// THE BOARDS, and they are the only questions that need no render: nothing
	// about a board reaches the buffer, so there is no click map to build and no
	// definition to wrap. One gloss each — for the panel — is the whole of what
	// the form takes, and targetCandidate is the same sense form 2.3 asks about.
	//
	// A word the dictionary no longer knows is skipped exactly as it is above.
	// The board only gets SHORTER for it, so the fit boardsFor already checked
	// still holds.
	for _, chunk := range boards {
		var cells []play.Cell
		for _, key := range chunk {
			text, err := d.dict.Lookup(key)
			if err != nil {
				fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
				continue
			}
			var gloss string
			if c, ok := targetCandidate(key, ParseEntry(text)); ok {
				gloss = c.Gloss
			}
			cells = append(cells, play.Cell{Word: key, Gloss: gloss})
		}
		if len(cells) > 0 {
			qs = append(qs, play.NewBoard(cells, opt.width))
		}
	}
	if len(qs) == 0 {
		// NOT "nothing due today": words WERE due, and every one of them failed
		// to look up. Saying nothing is due would send the learner away believing
		// their deck is clear when the dictionary is the problem.
		fmt.Fprintf(stderr, "define: %d words are due but none could be looked up\n", len(keys))
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
		return gradedPrompt
	}
	return gradePrompt(q)
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
