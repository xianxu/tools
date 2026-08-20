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
// It also owns CANCELLATION. In raw mode Ctrl-C arrives as byte 0x03, not a
// signal, so signal.NotifyContext never fires — and the moment that matters is
// during playback, when the loop is blocked inside speak for seconds and cannot
// act on anything. Cancelling here fires regardless of what the loop is doing.
func readKeys(ctx context.Context, r io.Reader, cancel context.CancelFunc) <-chan Key {
	out := make(chan Key)
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
						cancel() // reaches the loop even mid-playback
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
