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
	word string   // cmdDefine: the headword
	name string   // cmdCommand: the command name, "" for a bare "/"
	args []string // cmdCommand: everything after the name
}

type replKind int

const (
	cmdNothing replKind = iota // blank line, nothing to replay
	cmdDefine                  // define this word
	cmdReplay                  // replay the current word
	cmdCommand                 // a /-prefixed command
)

// parseREPLLine is the loop's decision table, kept pure so it is a unit test
// rather than something only reachable through a fake terminal.
//
// hasCurrent is passed in rather than read from session state, so the function
// has no memory and "blank line with nothing to replay" is an ordinary case.
func parseREPLLine(line string, hasCurrent bool) replCommand {
	// Commands are decided FIRST and HERE. This function is the one place both
	// loops route a submitted line through, so putting the "/" test anywhere
	// else means the raw editor and the line loop disagree about what a line
	// means — the defect #4 spent ten rounds paying for, in a different shape.
	if name, args, ok := parseCommandLine(line); ok {
		return replCommand{kind: cmdCommand, name: name, args: args}
	}
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

// NOTE: #2's eraseLineAndStepBack is DELETED, not ported. It stepped back over
// the terminal's echo of Enter to place the indicator on the prompt line, and
// broke whenever the user typed during playback because cooked-mode echo moves
// the cursor asynchronously. Raw mode does not echo, so the frame is simply
// rendered with the indicator where the prompt would be — the arithmetic has
// nothing left to correct for.

// repl reads words until the input ends or the context is cancelled.
//
// It reads stdin unconditionally and prompts only when interactive, so there is
// no separate batch path to keep in sync and the whole loop is testable from a
// string.
func repl(ctx context.Context, cancel context.CancelFunc, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
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

	if terminalUI {
		// Raw mode: keystrokes, a rendered frame, no terminal echo. Everything
		// #2 did with cursor arithmetic against an echoed Enter is gone.
		return replRaw(ctx, cancel, d, opt, stdin, stdout, stderr)
	}
	return replLines(ctx, d, opt, stdin, stdout, stderr, !interactive, terminalUI)
}

// replLines is the line-oriented loop: piped input, a redirected stdout, or a
// terminal we could not put into raw mode. Reads whole lines, draws no UI.
// pipedInput and showPrompt are SEPARATE parameters on purpose. They answer
// different questions and this repo has now conflated them four times:
//
//	pipedInput — is stdin a pipe?  → does a failed lookup set the exit code
//	showPrompt — do we own the terminal (stdin AND stdout)? → may we draw UI
//
// Collapsing them writes a prompt to stdout whenever stdin is a tty, which
// pollutes `define > out.txt`. That is the same bug as #2 close rounds 2, 4 and
// 5, and it recurred here the moment the two were passed as one flag.
func replLines(ctx context.Context, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer, pipedInput, showPrompt bool) int {
	// Wrapped HERE rather than in the caller, so the line the tests exercise is
	// the line production runs — #2's I-1 lesson, applied to both loops.
	d.audio = newCachingAudioSource(d.audio)
	lines, errc := scanLines(stdin)
	// Exiting 0 at EOF is right for a human at a prompt — a typo is not a failed
	// session. It is wrong for `echo word | define`, which README presents as
	// interchangeable with `define word` and documents as exiting 1 on an unknown
	// word. Track failures and report them only on the non-interactive path.
	var anyFailed bool
	var current string

	for {
		if showPrompt {
			fmt.Fprint(stdout, prompt)
		}
		select {
		case <-ctx.Done():
			return 0
		case err := <-errc:
			if err != nil {
				fmt.Fprintf(stderr, "define: reading input: %v\n", err)
				return 1
			}
			if pipedInput && anyFailed {
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
				// This path is only reached when we do NOT own the terminal, so
				// there is no transient UI to place: play and report.
				playAnnounced(ctx, d, opt, current, indicator{}, stdout, stderr)
			case cmdCommand:
				// The piped loop dispatches too. `echo /history | define` must
				// not reach the dictionary, and a first draft of #15 put this
				// only in the raw editor's submit path (PQ-2).
				if dispatchCommand(cmd, commands, commandCtx{
					stdout: stdout, stderr: stderr, width: terminalWidth(stdout),
				}) != 0 {
					anyFailed = true
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
