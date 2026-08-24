package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// replCommand is what one line of input means.
type replCommand struct {
	kind     replKind
	word     string   // cmdDefine: the headword
	question string   // cmdAsk: the question, with any forcing prefix stripped
	name     string   // cmdCommand: the command name, "" for a bare "/"
	args     []string // cmdCommand: everything after the name
	// literal suppresses the question fallback: a "\"-prefixed line the
	// dictionary misses stays a miss. One of #16's two escape hatches — the
	// other is "?", which reaches cmdAsk without a dictionary call at all.
	literal bool
	// note is what to say about a cmdNothing that is not simply a blank line.
	// A field rather than a fourth kind: the loops already do nothing here, and
	// only the wording differs.
	note string
}

type replKind int

const (
	cmdNothing replKind = iota // blank line, nothing to replay
	cmdDefine                  // define this word
	cmdReplay                  // replay the current word
	cmdCommand                 // a /-prefixed command
	cmdAsk                     // a question for the model
)

// noteEmptyQuestion is the hint for a bare "?" — the hatch typed with nothing
// after it. "type a word, or press return to replay the last one" is the wrong
// answer to a key the user pressed on purpose.
const noteEmptyQuestion = `type a question after "?"`

// noteEmptyLiteral is its counterpart for the other hatch.
const noteEmptyLiteral = `type a word after "\"`

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
	// The two escape hatches, decided HERE for the same reason "/" is. They sit
	// AFTER the command test so "/" keeps winning: "/help" is a command, never a
	// question about help.
	//
	// "?" forces a question without consulting the dictionary at all; "\" forces
	// the dictionary and suppresses the question fallback. Neither is the only
	// way to reach its outcome — a bare question still asks and a bare word still
	// looks up — which is what makes them escape hatches rather than syntax.
	trimmed := strings.TrimSpace(line)
	if rest, ok := strings.CutPrefix(trimmed, "?"); ok {
		q := strings.Join(strings.Fields(rest), " ")
		if q == "" {
			return replCommand{kind: cmdNothing, note: noteEmptyQuestion}
		}
		return replCommand{kind: cmdAsk, question: q}
	}
	var literal bool
	if rest, ok := strings.CutPrefix(trimmed, `\`); ok {
		if strings.TrimSpace(rest) == "" {
			// Symmetric with a bare "?": a hatch typed with no payload is a
			// malformed line, whatever the session state. Without this it fell
			// through to the empty-line test and REPLAYED audio when there was a
			// current word, so the two hatches disagreed at the prompt.
			return replCommand{kind: cmdNothing, note: noteEmptyLiteral}
		}
		literal, line = true, rest
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
	return replCommand{kind: cmdDefine, word: strings.Join(strings.Fields(word), " "), literal: literal}
}

// nothingSays is the ONE answer to "this line meant nothing — why, and what
// should the user do about it".
//
// #16 gave that question three sites — the one-shot, the piped loop and the raw
// editor — each with its own default text and line ending, and M2 was about to
// add a fourth. The line ending stays with the caller, because only the raw loop
// needs "\r\n"; the words do not vary by caller and so do not live there.
// inSession, not "canReplay": what varies between the callers is whether there
// IS a loop to press return in, not whether something is currently loaded. A
// note-less cmdNothing is reachable only when nothing is current — a blank line
// WITH a current word parses to cmdReplay — so a "can replay right now" reading
// would have selected the replay sentence in the one state where there is
// definitionally nothing to replay, and both loop sites were passing a literal
// true to a parameter that could not mean what it said.
func nothingSays(c replCommand, inSession bool) string {
	if c.note != "" {
		return c.note
	}
	if inSession {
		return "type a word, or press return to replay the last one"
	}
	return "type a word"
}

// recallLine is the canonical, re-submittable form of this line: what Up-arrow
// must put back so that pressing Enter means what it meant the first time.
//
// A forcing prefix is PART OF THE MEANING. Recall used to store the stripped
// word, so `\how so` came back as `how so` — which re-submits as a QUESTION,
// the exact opposite of what the hatch was typed to force (BR-12). The rule the
// three recall sites now share: what recall stores must re-submit to the same
// meaning. Whitespace is still collapsed, because that changes no meaning.
func (c replCommand) recallLine() string {
	switch c.kind {
	case cmdAsk:
		return "?" + c.question
	case cmdCommand:
		return strings.Join(append([]string{"/" + c.name}, c.args...), " ")
	case cmdDefine:
		if c.literal {
			return `\` + c.word
		}
		return c.word
	}
	return ""
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
func repl(ctx context.Context, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
	// The loop owns what an interrupt means (#16 D5), so it must not ALSO be
	// cancelled behind the sink's back by main's NotifyContext — a SIGINT would
	// end the session while an answer streams, whatever the sink points at.
	//
	// This sits HERE, above the replRaw/replLines choice, because BOTH loops need
	// a transport. Detaching in run() and watching only in replRaw leaves every
	// piped, redirected or raw-mode-fallback run with a ctx.Done() nothing can
	// reach: SIGINT diverted from default termination by NotifyContext, and then
	// delivered to no one (PQ-6).
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
		return replRaw(ctx, interrupts, d, opt, stdin, stdout, stderr)
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
	// A command's exit code survives the loop. Collapsing it into anyFailed made
	// `echo /histry | define` exit 1 where dispatchCommand computes 2 and the
	// README documents 2 for a usage error (BR-16).
	var cmdCode int
	var sess session
	// fail is the ONE place a per-line dispatch's exit code reaches the loop's.
	//
	// The loop's code is not a boolean: a usage error (2) has to survive
	// alongside a lookup failure (1). The comment on cmdCode records BR-16, where
	// collapsing a command's code into anyFailed made `echo /histry | define`
	// exit 1 where dispatch computed 2 — and #16 reintroduced exactly that for
	// questions, because the new branch collapsed ask()'s code the same way
	// (BR-15). One helper, so the next branch cannot repeat it a third time.
	fail := func(code int) {
		if code == 0 {
			return
		}
		anyFailed = true
		if code > cmdCode {
			cmdCode = code
		}
	}
	// ONE entry into the ask path, reached from two places: a forced question
	// ("?…", decided by the parser) and an unforced one (a dictionary miss that
	// reads as a question). They differ only in whether the dictionary was
	// consulted, so wiring them separately would mean maintaining the answer
	// path twice (ARCH-DRY).
	askHere := func(q question) { fail(ask(ctx, d, opt, &sess, stdout, stderr, q)) }

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
				if cmdCode != 0 {
					return cmdCode
				}
				return 1
			}
			return 0
		case line := <-lines:
			switch cmd := parseREPLLine(line, sess.hasCurrent()); cmd.kind {
			case cmdNothing:
				fmt.Fprintf(stderr, "define: %s\n", nothingSays(cmd, true))
				// A hatch typed with no payload is a malformed LINE, the same
				// shape as a malformed /command — so it carries the same usage
				// code, and README's "exit 2" holds piped as well as one-shot.
				if cmd.note != "" {
					fail(2)
				}
			case cmdReplay:
				if opt.noAudio || opt.times <= 0 {
					fmt.Fprintln(stderr, "define: nothing to replay: audio is off")
					break
				}
				// This path is only reached when we do NOT own the terminal, so
				// there is no transient UI to place: play and report.
				playAnnounced(ctx, d, opt, sess.current, indicator{}, stdout, stderr)
			case cmdCommand:
				// The piped loop dispatches too. `echo /history | define` must
				// not reach the dictionary, and a first draft of #15 put this
				// only in the raw editor's submit path (PQ-2).
				cc := newCommandCtx(d, opt, stdout, stderr)
				cc.setTimes = func(n int) { opt.times = n }
				fail(dispatchCommand(cmd, commands, cc))
			case cmdAsk:
				askHere(question{text: cmd.question, forced: true})
			case cmdDefine:
				// Only a successful lookup becomes the current word, so a typo
				// does not cost you the word you were listening to — and neither
				// does a question, which is why the ask branch returns first.
				out := defineOnce(ctx, d, opt, cmd, stdout, stderr)
				switch {
				case out.ask != "":
					askHere(question{text: out.ask})
				case out.code == 0:
					sess.sawLookup(cmd.word, out)
				default:
					fail(out.code)
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
