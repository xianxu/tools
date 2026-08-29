package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// rawSession puts the terminal into raw mode and guarantees restoration.
//
// A terminal left raw is the worst outcome this tool can produce — the user's
// shell becomes unusable and no later output can fix it. So restore runs from a
// defer AND on the cancellation path, and is idempotent.
type rawSession struct {
	fd    int
	state *term.State
	// f is the terminal itself, kept so restore can also leave the alternate
	// screen. Without it the two guarantees would live in different places and
	// a Ctrl-C would honour one of them.
	f   *os.File
	alt bool
}

func enterRaw(f *os.File) (*rawSession, error) {
	st, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	return &rawSession{fd: int(f.Fd()), state: st, f: f}, nil
}

func (r *rawSession) restore() {
	if r == nil {
		return
	}
	// The alternate screen goes FIRST, so the terminal is back on the normal
	// buffer before raw mode ends — the reverse order leaves a cooked terminal
	// briefly drawing into a buffer that is about to be discarded.
	r.leaveAlt()
	if r.state == nil {
		return
	}
	_ = term.Restore(r.fd, r.state)
	r.state = nil // idempotent: restoring twice is not an error, but claiming to is
}

// readKeys decodes keypresses from r onto a channel.
//
// It also delivers one of the two CANCELLATION transports. In raw mode Ctrl-C
// arrives as byte 0x03 — term.MakeRaw clears ISIG — and the moment that matters
// is during playback, when the loop is blocked inside speak for seconds and
// cannot act on anything, so firing here works regardless of what the loop is
// doing.
//
// It is NOT the only transport, though this comment used to say so: the pty
// suite measured a \x03 arriving as a SIGINT. Both feed the same interrupter,
// which is what decides the meaning (#16 D5).
func readKeys(ctx context.Context, r io.Reader, interrupts *interrupter) <-chan Key {
	// BUFFERED, and that is load-bearing. The loop stops reading while an answer
	// streams; on an unbuffered channel the reader would block on the first key
	// typed during it and never decode the bytes behind — including a Ctrl-C
	// meant to stop that very answer. Buffering also gives type-ahead during a
	// long answer for free: the keys are simply waiting when the prompt returns.
	out := make(chan Key, 256)
	go func() {
		defer close(out)
		var buf []byte
		chunk := make([]byte, 256)
		for {
			n, err := r.Read(chunk)
			if n > 0 {
				buf = append(buf, chunk[:n]...)
				for len(buf) > 0 {
					k, used := decodeKey(buf)
					if used == 0 {
						break // a partial sequence: wait for more bytes
					}
					buf = buf[used:]
					if k.Kind == KeyInterrupt {
						// Fires the sink even mid-playback, when the loop is
						// blocked and cannot act on anything itself.
						if interrupts.Fire() {
							// A scope consumed it — a streaming answer was
							// cancelled. Delivering it as well would have the
							// loop quit the session as soon as the answer ended,
							// which is the opposite of what was asked for.
							continue
						}
					}
					select {
					case out <- k:
					case <-ctx.Done():
						return
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return out
}

// The alternate screen: one buffer, no scrollback, discarded on exit.
//
// #30 takes it so a click's coordinates are exact — with no scrollback there is
// nothing above the viewport for the terminal to show, so nothing but `screen`
// can move the view. The session's own history is not lost: it is printed back
// into the normal buffer on exit (#30 D3), because losing it silently would be a
// regression a user meets immediately.
const (
	altScreenOn  = "\x1b[?1049h"
	altScreenOff = "\x1b[?1049l"
)

// enterAlt switches the terminal to the alternate screen.
//
// It lives on rawSession, and that placement is the whole point: a terminal left
// in the alternate screen is as bad an outcome as one left raw — the user's
// shell keeps working but everything they had scrolled back to is hidden behind
// a buffer nobody is drawing. rawSession already guarantees restoration from a
// defer AND on the cancellation path, so folding this in means the guarantee
// covers both rather than two mechanisms each covering half.
func (r *rawSession) enterAlt() {
	if r == nil || r.f == nil || r.alt {
		return
	}
	fmt.Fprint(r.f, altScreenOn)
	r.alt = true
}

// leaveAlt returns to the normal screen. Idempotent, for the same reason
// restore is: it runs from more than one path and claiming a second call is an
// error would make the paths care about each other.
func (r *rawSession) leaveAlt() {
	if r == nil || r.f == nil || !r.alt {
		return
	}
	fmt.Fprint(r.f, altScreenOff)
	r.alt = false
}
