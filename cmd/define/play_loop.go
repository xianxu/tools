package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/schedule"
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

	questions, code := todaysQuestions(d, opt, stdout, stderr)
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
	return playSession(ctx, d, opt, play.NewSession(questions),
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
	live := newLiveScreen(stdout, terminalRows(stdout), terminalCols(stdout))
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
func playSession(ctx context.Context, d deps, opt options, s play.Session,
	keys <-chan Key, con console) int {

	view, stdout, stderr := con.view, con.stdout, con.stderr

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
	draw := func() {
		if q := s.Current(); q != nil && written != s.Index {
			written = s.Index
			// Plain \n: the screen places every row, so nothing here decides
			// where a line goes (D1).
			fmt.Fprintf(stdout, "\n%s\n", q.Prompt())
		}
		view.Draw(livePrompt(s), nil)
	}

	// Every exit is the summary and THEN the terminal, in that order. The summary
	// is a buffer write like any other and handing the terminal back is what
	// prints the buffer (D5) — reversed, a sitting's last line would be written
	// into a screen nobody paints again and would be missing from the transcript
	// as well as from the terminal.
	over := func() int {
		code := finish(stdout, s, d, opt)
		con.finish()
		return code
	}

	draw()
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
		case got, ok := <-keys:
			if !ok {
				return over()
			}
			k = got
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
			case play.OutcomeDrop:
				// Through the store's own Forget, which is --forget's path: the deck
				// loses the word and the events keep it. Reported, because removing
				// something on one keystroke should say so.
				if removed, err := d.deck.Forget(out.Word); err != nil {
					fmt.Fprintf(stderr, "define: could not remove %q: %v\n", out.Word, err)
				} else if removed {
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
		draw()
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
func todaysQuestions(d deps, opt options, stdout, stderr io.Writer) ([]play.Question, int) {
	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(stderr, "define: could not read the deck: %v\n", err)
		return nil, 1
	}
	events, err := d.deck.Events(anyTime)
	if err != nil {
		// A log we cannot read means no progress, which makes every word look
		// new — wrong, but reviewing the wrong order beats refusing to review.
		fmt.Fprintf(stderr, "define: could not read the review log (%v); treating every word as new\n", err)
	}
	now := d.clock.Now()
	keys := schedule.Queue(deck, schedule.Fold(events), now, opt.count)
	if len(keys) == 0 {
		fmt.Fprintln(stdout, emptyQueueReason(len(deck), opt.count))
		return nil, 0
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
		if q := choiceFor(key, rendered, entry, pool, seedFor(key, day)); q != nil {
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
		return nil, 1
	}
	return qs, 0
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
// RECOMPUTED here rather than carried down from todaysQuestions, and that is
// correctness rather than convenience: the answers just given have changed
// every box involved, so the figure the learner should see is the one AFTER
// today, not the one the sitting opened with.
//
// A failure to read the deck or the log costs the line, not the sitting. The
// learner has just finished their reviews and those are already recorded; a
// summary that could fail the whole verb would trade something that matters for
// something that does not.
func finish(w io.Writer, s play.Session, d deps, opt options) int {
	fmt.Fprintf(w, "\n%d right, %d wrong\n", s.Right, s.Wrong)

	deck, err := d.deck.Deck()
	if err != nil {
		return 0
	}
	events, err := d.deck.Events(anyTime)
	if err != nil {
		return 0
	}
	prog := schedule.Fold(events)
	load := schedule.DailyLoad(deck, prog)
	// The budget is -count: the number of questions a sitting asks, which is
	// the DAILY budget for a learner who sits down once a day. That assumption
	// is stated in the line rather than hidden, so someone who sits twice knows
	// to double it. Inventing a second flag would give the tool two answers to
	// "how much do I do per day".
	fresh := schedule.SustainableNewWords(opt.count, deck, prog)
	// Through the SHARED formatter, so this line and the pinned bar cannot
	// describe the same deck differently or word the -count assumption two ways.
	fmt.Fprintln(w, sittingSummary(sittingFigures{load: load, fresh: fresh, budget: opt.count}))
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
