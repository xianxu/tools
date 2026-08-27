package main

import (
	"context"
	"fmt"
	"io"
	"os"

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

	questions, code := todaysQuestions(d, opt, stdout, stderr)
	if code != 0 || len(questions) == 0 {
		return code
	}

	// Same interrupt shape as repl (#16 D5): the loop owns what Ctrl-C means, so
	// it must not also be cancelled behind the sink's back.
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	interrupts := &interrupter{fn: cancel}
	if d.notifySignals != nil {
		go func() {
			for range d.notifySignals(os.Interrupt) {
				interrupts.Fire()
			}
		}()
	}

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

	return playSession(ctx, d, opt, play.NewSession(questions),
		readKeys(ctx, f, interrupts), sess, stdout, stderr)
}

// playSession drives the state machine and performs its outcomes.
//
// Split from runPlay so a test can drive a whole session with a scripted key
// channel and no terminal at all — the setup above is the part that needs one.
func playSession(ctx context.Context, d deps, opt options, s play.Session,
	keys <-chan Key, raw *rawSession, stdout, stderr io.Writer) int {

	draw(stdout, s)
	for !s.Done {
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
		var out play.Outcome
		s, out = play.Apply(s, in)

		switch out.Kind {
		case play.OutcomeRecord:
			// RECORDED NOW, before the next question is drawn. That is what makes
			// Ctrl-C lossless by construction rather than by a flush, and the
			// loop never inspects the verdict — a skip produced no outcome at
			// all, so there is nothing to filter here.
			d.capture.CaptureReview(out.Word, out.Verdict == play.Correct, opt)
		case play.OutcomeReveal:
			if !opt.noAudio && opt.times > 0 {
				// Cooked for playback, as #16 established: the indicator and any
				// warning are written for a human to read.
				word := s.Current().Word()
				if raw != nil {
					raw.restore()
				}
				playAnnounced(ctx, d, opt, word, defaultIndicator(opt), stdout, stderr)
				if raw != nil {
					if again, err := enterRaw(os.Stdin); err == nil {
						*raw = *again
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
		if k.Rune == ' ' {
			return play.Input{Kind: play.InputReveal}, true
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
		fmt.Fprintln(stdout, "define: nothing due today")
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
		fmt.Fprintln(stdout, "define: nothing due today")
	}
	return qs, 0
}

// anyTime is the zero time: Events(since) returns everything at or after it.
var anyTime = store.Word{}.FirstSeen

func draw(w io.Writer, s play.Session) {
	q := s.Current()
	if q == nil {
		return
	}
	fmt.Fprintf(w, "\r\n%s\r\n", q.Prompt())
	if s.Revealed {
		fmt.Fprintf(w, "\r\n%s\r\n", q.Reveal())
		fmt.Fprint(w, "\r\ny = got it, n = missed it, Ctrl-C to stop\r\n")
		return
	}
	fmt.Fprint(w, "\r\nEnter or space to reveal, Ctrl-C to stop\r\n")
}

func finish(w io.Writer, s play.Session) int {
	fmt.Fprintf(w, "\r\n%d right, %d wrong\r\n", s.Right, s.Wrong)
	return 0
}
