package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

// replRaw is the interactive loop when we own the terminal: keystrokes in, a
// rendered frame out.
//
// The whole editing model is Apply + RenderLine, both pure — this function is
// the thin shell that reads keys, draws frames, and dispatches lookups.
func replRaw(ctx context.Context, cancel context.CancelFunc, d deps, opt options, stdin io.Reader, stdout, stderr io.Writer) int {
	f, ok := stdin.(*os.File)
	if !ok {
		// Not a real terminal handle (a test harness, a wrapper): fall back to the
		// line path rather than pretending raw mode succeeded.
		return replLines(ctx, d, opt, stdin, stdout, stderr, false /*stdin is a tty*/, true /*we own it*/)
	}
	sess, err := enterRaw(f)
	if err != nil {
		fmt.Fprintf(stderr, "define: cannot enter raw mode (%v); falling back to line input\n", err)
		return replLines(ctx, d, opt, stdin, stdout, stderr, false /*stdin is a tty*/, true /*we own it*/)
	}
	defer sess.restore()

	keys := readKeys(ctx, f, cancel)
	// cooked drops raw mode around a lookup so the definition scrolls normally,
	// then re-enters for the next frame.
	cooked := func(run func()) {
		sess.restore()
		defer func() {
			if s, err := enterRaw(f); err == nil {
				*sess = *s
			}
		}()
		run()
	}
	return runEditor(ctx, keys, d, opt, cooked, sess.restore, stdout, stderr)
}

// runEditor is the editor loop with the terminal factored out: keys arrive on a
// channel, `cooked` runs a lookup outside raw mode, and `finish` restores the
// terminal. Tests drive it with a scripted channel and no terminal at all —
// which is the whole point of keeping Apply and RenderLine pure.
func runEditor(ctx context.Context, keys <-chan Key, d deps, opt options,
	cooked func(func()), finish func(), stdout, stderr io.Writer) int {

	// Same wiring as replLines: cached behind the seam, in the function that uses
	// it, so a test driving runEditor gets exactly what production gets.
	d.audio = newCachingAudioSource(d.audio)
	hist := &memHistory{}
	e := NewEditor()
	var current string
	draw := func() { fmt.Fprint(stdout, RenderLine(e, Suggestion(e, hist.Prefix(e.WalkBase())), opt.color)) }
	draw()

	for {
		select {
		case <-ctx.Done():
			finish() // before anything else can write: never exit leaving raw mode on
			fmt.Fprintln(stdout)
			return 0
		case k, open := <-keys:
			if !open {
				finish()
				fmt.Fprintln(stdout)
				return 0
			}
			var act Action
			e, act = Apply(e, k, hist.Prefix(e.WalkBase()))
			switch act {
			case ActInterrupt, ActEOF:
				finish()
				fmt.Fprintln(stdout)
				return 0
			case ActSubmit:
				line := e.String()
				e = NewEditor()
				fmt.Fprintln(stdout) // commit the input line to scrollback
				submitLine(ctx, cooked, d, opt, line, hist, &current, stdout, stderr)
				draw()
				continue
			}
			draw()
		}
	}
}

// submitLine handles one submitted line: a word is defined, a bare Enter replays.
func submitLine(ctx context.Context, cooked func(func()), d deps, opt options, line string,
	hist *memHistory, current *string, stdout, stderr io.Writer) {

	if line == "" { // bare Enter: replay, screen untouched
		if *current == "" {
			fmt.Fprintln(stderr, "define: type a word, or press return to replay the last one")
			return
		}
		if opt.noAudio || opt.times <= 0 {
			fmt.Fprintln(stderr, "define: nothing to replay: audio is off")
			return
		}
		// Replay also stays raw: same reason.
		playAnnounced(ctx, d, opt, *current, indicator{show: true, erase: eraseLine}, stdout, stderr)
		return
	}

	// Render in COOKED mode so newlines translate, but play in RAW mode so
	// Ctrl-C arrives as a byte the key reader can act on. Playback is the part
	// that blocks for seconds; doing it cooked made cancellation depend on a
	// signal that raw mode exists to replace.
	var code int
	var play bool
	cooked(func() { code, play = lookupAndRender(d, opt, line, stdout, stderr) })
	if play {
		playAnnounced(ctx, d, opt, line, indicator{show: true, before: "\r\n", erase: eraseLine}, stdout, stderr)
	}
	hist.Add(line, code == 0)
	if code == 0 {
		*current = line
	}
}
