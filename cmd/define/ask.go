package main

import (
	"fmt"
	"io"
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
// M2 replaces the body with a real request; it must stay the single entry the
// way this is, so the resolve/gather/stream/record sequence cannot fork. The
// terminal-mode wiring (the interrupter, the CRLF writer) belongs to the raw
// loop's closure around this call, not inside it.
func ask(opt options, errOut io.Writer, q question) int {
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
	if q.forced {
		// Nothing may be claimed about the text: the dictionary was skipped.
		fmt.Fprintf(errOut, "define: no model configured; cannot answer `%s`\n", truncateQuestion(q.text))
		return 1
	}
	// The message says what happened rather than what failed: the line was
	// understood as a question, and there is nowhere to send it. Looking a
	// sentence up instead — the pre-#16 behaviour — reported "not found" for
	// something that was never a word.
	fmt.Fprintf(errOut, "define: no model configured; `%s` is not a word\n", truncateQuestion(q.text))
	return 1
}
