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
	sess, err := enterRaw(f)
	if err != nil {
		fmt.Fprintf(stderr, "define: could not enter raw mode: %v\n", err)
		return 1
	}
	defer sess.restore()

	// EVERY byte of session output goes through crlfWriter.
	//
	// In raw mode a bare \n moves down WITHOUT returning to column 0, so a
	// multi-line definition cascades diagonally across the screen — each line
	// starting where the last one ended. #16 built this writer for exactly that
	// and the atlas documents it; the first version of this loop put \r\n in its
	// own format strings and forgot that Render's output has bare newlines
	// throughout, which is most of what a session prints.
	return playSession(ctx, d, opt, play.NewSession(questions),
		readKeys(ctx, f, interrupts), rawTerm{sess: sess, f: f},
		&crlfWriter{w: stdout}, &crlfWriter{w: stderr})
}

// rawTerm is the terminal a session borrows during playback and takes back
// after.
//
// It carries the FILE, because re-entering raw mode has to use the descriptor
// runPlay was handed — the first version hardcoded os.Stdin, which is right only
// by coincidence and silently wrong for any caller given another descriptor.
type rawTerm struct {
	sess *rawSession
	f    *os.File
}

// playSession drives the state machine and performs its outcomes.
//
// Split from runPlay so a test can drive a whole session with a scripted key
// channel and no terminal at all — the setup in runPlay is the part that needs
// one. Its own doc block, because it had been swallowed into rawTerm's.
func playSession(ctx context.Context, d deps, opt options, s play.Session,
	keys <-chan Key, raw rawTerm, stdout, stderr io.Writer) int {

	draw(stdout, s)
	for !s.Done {
		// Cancellation is checked BEFORE the select, not only inside it.
		//
		// select picks uniformly at random among ready cases, so with a
		// cancelled context and a key already buffered it would sometimes grade
		// one more answer after Ctrl-C — recording a verdict for a word the
		// learner had stopped on. Caught as an intermittent test failure, which
		// is the only way a random-choice bug ever shows up.
		if ctx.Err() != nil {
			return finish(stdout, s)
		}
		var k Key
		select {
		case <-ctx.Done():
			return finish(stdout, s)
		case got, ok := <-keys:
			if !ok {
				return finish(stdout, s)
			}
			k = got
		}

		in, ok := toInput(k)
		if !ok {
			continue
		}
		var outs []play.Outcome
		s, outs = play.Apply(s, in)

		// EVERY outcome, in order. One input can owe more than one: a miss on a
		// hidden word records the verdict and then reveals. The record comes
		// first, so it is written before anything that can block on the terminal.
		for _, out := range outs {
			switch out.Kind {
			case play.OutcomeRecord:
				// RECORDED NOW, before the next question is drawn. That is what makes
				// Ctrl-C lossless by construction rather than by a flush, and the
				// loop never inspects the verdict — a skip produced no outcome at
				// all, so there is nothing to filter here.
				d.capture.CaptureReview(out.Word, out.Verdict == play.Correct, opt)
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
				if !opt.noAudio && opt.times > 0 {
					// Cooked for playback, as #16 established: the indicator and any
					// warning are written for a human to read.
					//
					// s.Current() is still the RIGHT word: every input that emits
					// OutcomeReveal leaves the session on its question — a peek
					// does not advance, and a miss deliberately does not either.
					// If a future input ever advances AND reveals, the audio would
					// play for the next word; carry the word on the outcome then,
					// the way OutcomeRecord and OutcomeDrop already do.
					word := s.Current().Word()
					if raw.sess != nil {
						raw.sess.restore()
					}
					playAnnounced(ctx, d, opt, word, defaultIndicator(opt), stdout, stderr)
					if raw.sess != nil {
						again, err := enterRaw(raw.f)
						if err != nil {
							// REPORTED, not dropped. Without raw mode readKeys is
							// line-buffered, so every keystroke appears to do nothing
							// until Enter — the session looks frozen and nothing says
							// why. Ending is honest; pretending to continue is not.
							// EXIT 1, like the failure to enter raw mode in the first
							// place. Both are "this session cannot continue because
							// the terminal is gone", and returning 0 from one of them
							// tells a script the session ended normally when it did
							// not (BR-24).
							fmt.Fprintf(stderr, "define: lost the terminal after playback: %v\n", err)
							finish(stdout, s)
							return 1
						}
						*raw.sess = *again
					}
				}
			}
		}
		draw(stdout, s)
	}
	return finish(stdout, s)
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

	var qs []play.Question
	for _, key := range keys {
		text, err := d.dict.Lookup(key)
		if err != nil {
			// A word in the deck the dictionary no longer knows. Skip it rather
			// than failing the session: the other words are still worth review.
			fmt.Fprintf(stderr, "define: skipping %q: %v\n", key, err)
			continue
		}
		rendered := Render(ParseEntry(text), RenderOpts{
			Color: opt.color, Width: opt.width, Vocab: vocabularyFor(d, opt),
		})
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

func draw(w io.Writer, s play.Session) {
	q := s.Current()
	if q == nil {
		return
	}
	// Plain \n throughout: the caller wraps stdout in crlfWriter, so translation
	// happens in ONE place over every byte — including Render's, which is where
	// the newlines actually are.
	fmt.Fprintf(w, "\n%s\n", q.Prompt())
	if s.Revealed {
		fmt.Fprintf(w, "\n%s\n", q.Reveal())
	}
	if s.Graded {
		// Answered, and the answer is on screen. The only thing left is to read
		// it and move on — offering y/n here would invite a second verdict on a
		// question that already has one.
		fmt.Fprint(w, "\nany key = next word, d = remove from deck, Ctrl-C to stop\n")
		return
	}
	// The GRADING keys, whether or not the definition is showing.
	//
	// This line used to appear only AFTER a reveal, and an unrevealed word said
	// "Enter or space to reveal" instead — so every correct answer cost a
	// keystroke that carried no information, and the slow one at that, since a
	// reveal fetches and plays the pronunciation. A learner who wants to check
	// before rating still can; they simply no longer have to (#24).
	fmt.Fprint(w, "\ny = got it, n = missed it, d = remove from deck, Ctrl-C to stop\n")
}

func finish(w io.Writer, s play.Session) int {
	fmt.Fprintf(w, "\n%d right, %d wrong\n", s.Right, s.Wrong)
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
