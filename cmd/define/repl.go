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

// repl reads words until the input ends or the context is cancelled.
//
// It reads stdin unconditionally and prompts only when interactive, so there is
// no separate batch path to keep in sync and the whole loop is testable from a
// string.
func repl(ctx context.Context, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
	interactive := d.stdinIsTerminal != nil && d.stdinIsTerminal()

	// Cache behind the seam, so the fake CDN's own request recorder proves that a
	// replay costs no second fetch.
	d.audio = newCachingAudioSource(d.audio)

	lines, errc := scanLines(stdin)
	var current string

	for {
		if interactive {
			fmt.Fprint(stdout, prompt)
		}
		select {
		case <-ctx.Done():
			if interactive {
				fmt.Fprintln(stdout)
			}
			return 0
		case err := <-errc:
			if err != nil {
				fmt.Fprintf(stderr, "define: reading input: %v\n", err)
				return 1
			}
			if interactive {
				fmt.Fprintln(stdout)
			}
			return 0
		case line := <-lines:
			switch cmd := parseREPLLine(line, current != ""); cmd.kind {
			case cmdNothing:
				fmt.Fprintln(stderr, "define: type a word, or press return to replay the last one")
			case cmdReplay:
				// Silent: no definition, no announcement, nothing written to
				// stdout at all. Pressing return means "say it again" — the screen
				// should look exactly as it did, so what you are hearing stays
				// next to what you are reading.
				if err := replay(ctx, d, opt, current); err != nil {
					fmt.Fprintf(stderr, "define: %s\n", err)
				}
			case cmdDefine:
				// Only a successful lookup becomes the current word, so a typo
				// does not cost you the word you were listening to.
				if defineOnce(ctx, d, opt, cmd.word, stdout, stderr) == 0 {
					current = cmd.word
				}
			}
		}
	}
}

// replay speaks the current word again, writing nothing to stdout.
func replay(ctx context.Context, d deps, opt options, word string) error {
	if opt.noAudio || opt.times <= 0 {
		return fmt.Errorf("nothing to replay: audio is off")
	}
	return speak(ctx, d, word, opt.locale, opt.times)
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
