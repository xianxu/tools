package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/tools/internal/llm"
)

// question is a line on its way to the model, and HOW it got here.
//
// The route is carried rather than inferred because it changes what can honestly
// be said. An unforced question reached this path because the dictionary missed
// it, so "is not a word" is true by construction. A forced one skipped the
// dictionary entirely — `?why` is a headword — so the same sentence would be a
// claim nothing checked (BR-11).
type question struct {
	text   string
	forced bool
}

// mayAsk is the single answer to "may this session reach the model at all".
//
// One predicate, consulted by BOTH dispatches — the unforced fallback in
// lookupAndRender and the forced "?" route in all three entry modes. Guarding
// only the fallback left three of six cells open: `define -raw "?what is X"`,
// the piped loop and the raw editor all still asked, while README stated the
// absolute (BR-9). A rule stated as an absolute has to be enforced at the point
// the thing happens, not on one route to it.
func mayAsk(opt options) bool { return !opt.raw }

// ask is the ONE entry into the question path, from all six cells of
// {forced, unforced} x {one-shot, piped loop, raw editor}.
//
// It refuses, or hands off to runAsk. The terminal-mode wiring — the interrupt
// sink, the CRLF writer — belongs to the raw loop's closure around the call,
// not inside it, because it is the only caller that owns a terminal.
func ask(ctx context.Context, d deps, opt options, sess *session, out, errOut io.Writer, q question) int {
	if !mayAsk(opt) {
		// -raw's output contract is an unparsed dictionary entry, and a model
		// answer is not one — so the two cannot both be honoured. An explicit
		// "?" alongside it is contradictory input, and this repo says so out
		// loud rather than guessing (as -forget with a word does, and --sound
		// with -times). The UNFORCED route never gets here: it simply does not
		// fall back, so a miss stays a miss and scripts see what they expect.
		fmt.Fprintln(errOut, `define: -raw does not ask; drop -raw, or drop the "?"`)
		return 2
	}
	// Both routes go to the model. The forced/unforced distinction decides what
	// can be SAID when there is no model to reach — see unavailable — and never
	// whether the question is asked at all: M1 short-circuited the forced route
	// here, so `?why` never reached the seam even when one was configured.
	return runAsk(ctx, d, opt, sess, q, out, errOut)
}

// unavailable is what a question gets when there is nowhere to send it.
//
// The message says what happened rather than what failed: the line was
// understood as a question, and no model is configured. Looking a sentence up
// instead — the pre-#16 behaviour — reported "not found" for something that was
// never a word.
func unavailable(errOut io.Writer, q question) int {
	if q.forced {
		// Nothing may be claimed about the text: the dictionary was skipped.
		fmt.Fprintf(errOut, "define: no model configured; cannot answer `%s`\n", truncateQuestion(q.text))
		return 1
	}
	fmt.Fprintf(errOut, "define: no model configured; `%s` is not a word\n", truncateQuestion(q.text))
	return 1
}

// maxContextWords bounds each of the two word lists in the prompt. The deck is
// unbounded on disk and a year of it would crowd out the question.
const maxContextWords = 12

// runAsk is the one place a question reaches the network.
//
// Thin on purpose: resolve, gather, render, stream, record. Every decision it
// looks like it makes — what context to include, how to phrase it, how many
// turns to keep — belongs to renderAskPrompt and is unit-tested without a
// socket (ARCH-PURE).
func runAsk(ctx context.Context, d deps, opt options, sess *session, q question, out, errOut io.Writer) int {
	// A nil seam is "no model wired", not a crash: llm.Resolve dereferences the
	// getenv it is handed, and every test that does not care about the model
	// leaves these unset. Degrading here is also the honest answer for a build
	// where the seam was never wired at all.
	if d.getenv == nil || d.newLLM == nil {
		return unavailable(errOut, q)
	}
	cfg, err := llm.Resolve(d.getenv)
	if err != nil {
		return unavailable(errOut, q)
	}

	req := renderAskPrompt(gatherAskContext(d, sess, q, errOut))
	answer := &strings.Builder{}
	_, err = d.newLLM(cfg).Stream(ctx, req, func(delta string) {
		answer.WriteString(delta)
		fmt.Fprint(out, delta)
	})

	// Asked FIRST, and asked of the CONTEXT rather than the error. A cancelled
	// request never reached a status, and mapError classifies statusless
	// failures as ErrUnavailable — so a decode-the-error version answers the
	// user's own Ctrl-C with "no model configured". The same distinction
	// playAnnounced draws for interrupted playback.
	// Recorded on EVERY path from here, because the question was asked whatever
	// became of the answer — a cancelled or failed one is still what the learner
	// wanted to know, which is the signal #17 reads. Not recorded above this
	// point, where no request was ever sent (README says exactly this).
	defer func() {
		if d.capture != nil {
			d.capture.CaptureAsk(sess.current, q.text, opt)
		}
	}()

	if ctx.Err() != nil {
		if answer.Len() > 0 {
			fmt.Fprintln(out) // close the partial line the stream left open
		}
		return 0
	}
	switch {
	case err == nil:
	case errors.Is(err, llm.ErrUnavailable):
		if answer.Len() > 0 {
			break // the answer arrived; the failure was in the teardown
		}
		return unavailable(errOut, q)
	case errors.Is(err, llm.ErrTruncated):
		// Partial text is kept: once frames have arrived the service is
		// demonstrably reachable, and half an answer beats none.
		fmt.Fprintf(errOut, "define: the answer was cut off\n")
	default:
		// ErrRequest and ErrMalformed are OURS — a bad prompt, a bad model, a
		// body nothing could decode. Loud, per the taxonomy in atlas/llm.md.
		fmt.Fprintf(errOut, "define: %v\n", err)
		return 1
	}

	if answer.Len() > 0 && !strings.HasSuffix(answer.String(), "\n") {
		fmt.Fprintln(out)
	}
	sess.recordExchange(q.text, answer.String())
	return 0
}

// gatherAskContext is the thin IO step: read what the directory holds.
//
// It formats nothing and truncates nothing beyond the counts — those are the
// pure renderer's, which is what keeps "what did we send" assertable from a
// struct literal.
func gatherAskContext(d deps, sess *session, q question, warnOut io.Writer) askContext {
	c := askContext{
		Question:     q.text,
		CurrentWord:  sess.current,
		CurrentEntry: sess.entry,
		SessionWords: lastN(sess.words, maxContextWords),
		Turns:        sess.turns,
	}
	if d.deck == nil {
		return c
	}
	model, err := d.deck.UserModel()
	if err != nil {
		// NOT silently empty. store/yaml.go says why it returns an error at all:
		// answering "" for a model that exists pitches every answer at the wrong
		// level with no way to tell. Warned like a store that cannot be opened.
		fmt.Fprintf(warnOut, "define: could not read user-model.md (%v); answering without it\n", err)
	}
	c.UserModel = model

	if deck, err := d.deck.Deck(); err != nil {
		fmt.Fprintf(warnOut, "define: could not read the deck (%v); answering without it\n", err)
	} else {
		c.DeckWords = recentDeck(deck, maxContextWords)
	}
	return c
}

func lastN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
