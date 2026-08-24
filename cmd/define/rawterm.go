package main

import (
	"context"
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
}

func enterRaw(f *os.File) (*rawSession, error) {
	st, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	return &rawSession{fd: int(f.Fd()), state: st}, nil
}

func (r *rawSession) restore() {
	if r == nil || r.state == nil {
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
