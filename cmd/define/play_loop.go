package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
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

	questions, held, code := todaysQuestions(d, opt, stdout, stderr)
	if code != 0 || len(questions) == 0 {
		return code
	}

	// Same interrupt shape as repl (#16 D5), and now literally the same body —
	// see detachedInterrupts for why the detach happens before the loop starts.
	ctx, interrupts, cancel := detachedInterrupts(ctx, d)
	defer cancel()

	f, isFile := stdin.(*os.File)
	if !isFile || d.stdinIsTerminal == nil || !d.stdinIsTerminal() {
		// A review is a conversation with a person. Piped input would answer
		// questions it never saw.
		fmt.Fprintln(stderr, "define: --play needs a terminal")
		return 1
	}
	sess, err := enterRaw(f, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "define: could not enter raw mode: %v\n", err)
		return 1
	}
	defer sess.restore()

	// THE SCREEN, not crlfWriter (D1).
	//
	// A sitting used to write lines to a scrolling terminal, which is why every
	// byte went through crlfWriter: in raw mode a bare \n moves down WITHOUT
	// returning to column 0, so a multi-line definition cascaded diagonally
	// across the screen. The screen places every row itself, so nothing on this
	// path depends on the line discipline any more — and two owners of line
	// endings is how they drift (#30 D5).
	return playSession(ctx, d, opt, play.NewSession(questions), held,
		readKeys(ctx, f, interrupts), playConsole(ctx, d, sess, stdout))
}

// playConsole is the terminal a sitting draws on, built exactly as replRaw
// builds the editor's (D1).
//
// Not a parallel construction: the same alternate screen, the same mouse
// reporting, the same resize watch and the same hand-back. `--play` diverging
// from this is precisely what #41 exists to end, so a difference here would have
// to be argued for rather than merely written.
func playConsole(ctx context.Context, d deps, sess *rawSession, stdout io.Writer) console {
	// The alternate screen, and with it the END of the cooked/raw dance (D5a):
	// playback no longer hands the terminal back, so the alt screen is entered
	// once and left once.
	sess.enterAlt()
	// The mouse, whose wheel pages a long reveal (D6). Inside the alternate
	// screen a terminal sends the wheel as ARROW KEYS unless asked to report the
	// mouse, and the bytes are identical — the report is the only way to be
	// handed the gesture the reader actually made.
	sess.enterMouse()
	// PINNED, unlike the editor's: a status bar belongs at the terminal's bottom
	// edge, where the editor's dropdown belongs under the line being typed (D3a).
	live := newPinnedScreen(stdout, terminalRows(stdout), terminalCols(stdout))
	// MEASURED here, where the terminal is, and delivered to the loop as a value
	// — so the loop's resize case knows nothing about os/signal.
	resizes := watchResize(ctx, d.notifySignals, func() winSize {
		return winSize{rows: terminalRows(stdout), cols: terminalCols(stdout)}
	})
	return console{
		view: live, resizes: resizes,
		// ONCE, and the transcript is why: a sitting's words must still be on
		// screen after quitting, and printing them twice is not something an
		// idempotent restore fixes (D5).
		finish: onceHandBack(live, sess, stdout),
		// BOTH streams are the screen, stderr included: a diagnostic written
		// straight to the terminal while the alternate screen is up lands
		// wherever the cursor happens to be and corrupts the frame.
		stdout: live, stderr: live,
	}
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
func playSession(ctx context.Context, d deps, opt options, s play.Session, held sittingDeck,
	keys <-chan Key, con console) int {

	view, stdout, stderr := con.view, con.stdout, con.stderr

	// THE BAR'S FIGURES, refreshed in memory (D7).
	//
	// `budget` and `total` are fixed for the sitting; `load`, `fresh` and `done`
	// move only when the deck does. So this is called from the two outcomes that
	// change it — a record and a drop — and NOT once per frame: a frame is per
	// keystroke, and an O(deck) walk per keystroke is a cost the plan's own
	// table says this path does not pay.
	fig := held.figures(opt.count)
	fig.total = len(s.Questions)
	refresh := func() {
		f := held.figures(opt.count)
		fig.load, fig.fresh = f.load, f.fresh
		// ANSWERED, which is what the Spec's bar says. A dropped word is not an
		// answer: the learner curated it away rather than being asked about it.
		fig.done = s.Right + s.Wrong
	}

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
	// `show` rather than `draw`: the package-level draw() is gone, and a closure
	// wearing its name would read as the same thing narrowed rather than as the
	// different thing it is — this one decides what has changed, where that one
	// printed everything every time.
	show := func() {
		if q := s.Current(); q != nil && written != s.Index {
			written = s.Index
			// Plain \n: the screen places every row, so nothing here decides
			// where a line goes (D1).
			fmt.Fprintf(stdout, "\n%s\n", q.Prompt())
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
			// The editor's case, mirrored — with one line missing on purpose.
			// It also sets opt.width, the POLICY width new entries wrap to;
			// here every question was rendered before the sitting started, so
			// re-deriving it would change nothing and would claim a re-wrap
			// this loop does not do.
			view.Resize(sz.rows, sz.cols)
			show()
			continue
		case got, ok := <-keys:
			if !ok {
				return over()
			}
			k = got
		}

		// A VIEWPORT GESTURE NEVER REACHES play (D6).
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
		// the top with no way back.
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

		// The question the keystroke is ABOUT, read before Apply moves on. A
		// reveal never advances, so s.Current() would answer the same — but
		// reading it after would make that a fact about Apply that this loop
		// silently depends on, and BR-4 is what a nil Current() costs.
		asked := s.Current()
		in, ok := toInput(k)
		if !ok {
			continue
		}
		var outs []play.Outcome
		s, outs = play.Apply(s, in)

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
		//	             TestAMissIsRecordedBeforeItIsRevealed, which observes the
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
					fmt.Fprintf(stdout, "\n%s\n", asked.Reveal())
				}
				if !opt.noAudio && opt.times > 0 {
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
		return play.Input{Kind: play.InputReveal}, true
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

// todaysQuestions builds the queue: fold the log, ask the schedule, render each
// word's definition.
//
// It RETURNS the deck and the folded progress rather than discarding them, and
// that is the whole of D7. Deck() reads one file per WORD and Events() one file
// per DAY of history, so a sitting that recomputed its cost figures per answer
// would pay ~5,700 file reads per question on a 5,000-word deck with two years
// of log — with a person waiting. These two reads are the sitting's only ones,
// and everything after them happens in memory.
func todaysQuestions(d deps, opt options, stdout, stderr io.Writer) ([]play.Question, sittingDeck, int) {
	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(stderr, "define: could not read the deck: %v\n", err)
		return nil, sittingDeck{}, 1
	}
	events, err := d.deck.Events(anyTime)
	if err != nil {
		// A log we cannot read means no progress, which makes every word look
		// new — wrong, but reviewing the wrong order beats refusing to review.
		// The sitting still runs, and its reviews are still recorded; only the
		// cost figures are computed against an empty history.
		fmt.Fprintf(stderr, "define: could not read the review log (%v); treating every word as new\n", err)
	}
	held := sittingDeck{deck: deck, prog: schedule.Fold(events)}
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

	var qs []play.Question
	for _, key := range keys {
		text, err := d.dict.Lookup(key)
		if err != nil {
			// A word in the deck the dictionary no longer knows. Skip it rather
			// than failing the session: the other words are still worth review.
			fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
			continue
		}
		entry := ParseEntry(text)
		// No regions: `--play` draws its own frames and has no click map (D5a).
		rendered, _ := Render(entry, RenderOpts{
			Color: opt.color, Width: opt.width, Vocab: vocabularyFor(d, opt),
		})
		// Form 2.3 when the deck can supply distractors, form 2.1 when it
		// cannot (D9). A young deck is a NORMAL state, not an error, and the
		// fallback is invisible to the learner — the sitting stays the length
		// the schedule asked for either way.
		if q := choiceFor(key, rendered, entry, pool, seedFor(key, day), opt.width); q != nil {
			qs = append(qs, q)
			continue
		}
		qs = append(qs, play.NewRecall(key, rendered))
	}
	if len(qs) == 0 {
		// NOT "nothing due today": words WERE due, and every one of them failed
		// to look up. Saying nothing is due would send the learner away believing
		// their deck is clear when the dictionary is the problem.
		fmt.Fprintf(stderr, "define: %d words are due but none could be looked up\n", len(keys))
		return nil, held, 1
	}
	return qs, held, 0
}

// sittingDeck is the deck as ONE sitting sees it: read once at the start and
// kept in memory for the rest of it (D7).
//
// The point is not caching. It is that the same transition schedule.Fold applies
// to a logged event is applied HERE the instant an answer lands — so the cost
// the bar shows during a sitting and the cost the next sitting derives from the
// log are the same number by construction, not by two pieces of code agreeing.
type sittingDeck struct {
	deck []store.Word
	prog map[string]schedule.Progress
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
	// sessionKeys are the keys the SESSION reserves, true whatever form is
	// asking (question.go:74-77). The form's own keys are prepended by
	// gradePrompt — this half does not vary, and a form restating it would be
	// two owners of one fact.
	sessionKeys = "d = remove from deck, Ctrl-C to stop"
	// gradedPrompt is shown once the answer is in and the definition is up.
	gradedPrompt = "any key = next word, d = remove from deck, Ctrl-C to stop"
)

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
	return q.Keys() + ", " + sessionKeys
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
