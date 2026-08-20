package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// replCommand is what one line of input means.
type replCommand struct {
	kind replKind
	word string
}

type replKind int

const (
	cmdNothing replKind = iota // blank line, nothing to replay
	cmdDefine                  // define this word
	cmdReplay                  // replay the current word
)

// parseREPLLine is the loop's decision table, kept pure so it is a unit test
// rather than something only reachable through a fake terminal.
//
// hasCurrent is passed in rather than read from session state, so the function
// has no memory and "blank line with nothing to replay" is an ordinary case.
func parseREPLLine(line string, hasCurrent bool) replCommand {
	word := strings.TrimSpace(line)
	if word == "" {
		if hasCurrent {
			return replCommand{kind: cmdReplay}
		}
		return replCommand{kind: cmdNothing}
	}
	// Multi-word headwords are real — "hot dog", "a priori" — so the whole line
	// is the word, with interior whitespace collapsed.
	return replCommand{kind: cmdDefine, word: strings.Join(strings.Fields(word), " ")}
}

// maxLineBytes bounds one line of input. bufio.Scanner's default is 64 KB, past
// which it stops with ErrTooLong — and once that happens every later Scan()
// returns false, so the error cannot be recovered from, only reported. Raising
// the cap is the mechanical guard; a line past this is genuinely not a word.
const maxLineBytes = 1 << 20

const prompt = "› "

// eraseLine clears the current line and returns the cursor to its start, so a
// transient indicator can be removed once it has served its purpose.
const eraseLine = "\r\x1b[K"

// eraseLineAndStepBack additionally moves up onto the prompt line and clears it,
// so the prompt can be redrawn in place. A replay therefore leaves the screen
// exactly as it was — the terminal's echo of Enter is undone rather than
// accepted, and the view never scrolls.
//
// This is the only cursor control in the tool; #2 listed it as a non-goal and the
// operator lifted it for exactly this (2026-08-20). Gated on a terminal, so piped
// output never sees an escape sequence.
const eraseLineAndStepBack = eraseLine + "\x1b[A" + eraseLine

// repl reads words until the input ends or the context is cancelled.
//
// It reads stdin unconditionally and prompts only when interactive, so there is
// no separate batch path to keep in sync and the whole loop is testable from a
// string.
func repl(ctx context.Context, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
	interactive := d.stdinIsTerminal != nil && d.stdinIsTerminal()
	// ONE predicate for "there is a human looking at a terminal", used for every
	// byte of interactive UI: the prompt, the indicator, and the cursor control.
	//
	// This family of bug has now appeared three times — cursor control gated on
	// stdin, then the atlas describing that weaker gate, then the prompt itself.
	// Each was a separate fix. The rule underneath all three: UI goes to STDOUT,
	// so stdout must be a terminal; it responds to a human, so stdin must be one
	// too. `define > out.txt` satisfies neither and must stay clean.
	//
	// `interactive` alone survives only where the question really is about stdin:
	// whether a failed lookup should set the exit code.
	terminalUI := interactive && opt.tty

	// Cache behind the seam, so the fake CDN's own request recorder proves that a
	// replay costs no second fetch.
	d.audio = newCachingAudioSource(d.audio)

	lines, errc := scanLines(stdin)
	var current string
	var skipPrompt bool
	// Exiting 0 at EOF is right for a human at a prompt — a typo is not a failed
	// session. It is wrong for `echo word | define`, which README presents as
	// interchangeable with `define word` and documents as exiting 1 on an unknown
	// word. Track failures and report them only on the non-interactive path.
	var anyFailed bool

	for {
		if terminalUI && !skipPrompt {
			fmt.Fprint(stdout, prompt)
		}
		skipPrompt = false
		select {
		case <-ctx.Done():
			if terminalUI {
				fmt.Fprintln(stdout)
			}
			return 0
		case err := <-errc:
			if err != nil {
				fmt.Fprintf(stderr, "define: reading input: %v\n", err)
				return 1
			}
			if terminalUI {
				fmt.Fprintln(stdout)
			}
			if !interactive && anyFailed {
				return 1
			}
			return 0
		case line := <-lines:
			switch cmd := parseREPLLine(line, current != ""); cmd.kind {
			case cmdNothing:
				fmt.Fprintln(stderr, "define: type a word, or press return to replay the last one")
			case cmdReplay:
				if opt.noAudio || opt.times <= 0 {
					fmt.Fprintln(stderr, "define: nothing to replay: audio is off")
					break
				}
				// The indicator REPLACES the prompt rather than appearing below
				// it: the terminal has already echoed Enter onto a new line, so
				// step back over that echo and over the prompt itself before
				// drawing. While the sound plays there is no prompt, which is
				// honest — input is not accepted during playback anyway.
				ind := indicator{show: terminalUI, before: eraseLineAndStepBack, erase: eraseLine}
				if playAnnounced(ctx, d, opt, current, ind, stdout, stderr) && terminalUI {
					// Nothing was reported, so the line we cleared is ours to
					// reclaim: redraw the prompt in place and let the loop skip
					// its own. On a failure we deliberately do NOT, so the
					// diagnostic is not written onto a redrawn prompt and the
					// loop still draws one afterwards.
					fmt.Fprint(stdout, prompt)
					skipPrompt = true
				}
			case cmdDefine:
				// Only a successful lookup becomes the current word, so a typo
				// does not cost you the word you were listening to.
				if defineOnce(ctx, d, opt, cmd.word, stdout, stderr) == 0 {
					current = cmd.word
				} else {
					anyFailed = true
				}
			}
		}
	}
}

// scanLines reads in a goroutine so a pending read cannot swallow cancellation.
//
// The goroutine outlives repl when the context is cancelled — it stays blocked
// on stdin — so it shares nothing mutable with the loop: the current word lives
// in repl only, and these channels carry values.
func scanLines(r io.Reader) (<-chan string, <-chan error) {
	lines := make(chan string)
	errc := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64<<10), maxLineBytes)
		for sc.Scan() {
			lines <- sc.Text()
		}
		errc <- sc.Err()
	}()
	return lines, errc
}
