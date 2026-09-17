package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"syscall"

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
	// control is where the terminal's MODE sequences go — the alternate screen,
	// mouse reporting and bracketed paste. An io.Writer rather than the *os.File, for two
	// reasons. It is the seam that makes the restore protocol assertable in
	// process: the test that named itself the pin for this could not fail,
	// because with a nil file every enter and every leave returned at the same
	// guard and the assertion checked a field nothing had set.
	//
	// And it is where the sequences BELONG: they change the screen the frames
	// are drawn on, so they go to the stream the frames go to. Writing them to
	// the stdin handle worked only because interactive means both are the same
	// tty — an assumption this type had never made before.
	control io.Writer
	alt     bool
	mouse   bool
	paste   bool
}

// enterRaw puts f into raw mode and sends mode sequences to control.
//
// The two are separate because they are separate facts: raw mode is a property
// of the input FD, while the alternate screen and mouse reporting are output the
// user sees. They coincide on a terminal and must not be assumed to.
func enterRaw(f *os.File, control io.Writer) (*rawSession, error) {
	st, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	return &rawSession{fd: int(f.Fd()), state: st, control: control}, nil
}

func (r *rawSession) restore() {
	if r == nil {
		return
	}
	// Mouse reporting goes first of all: a terminal left reporting the mouse
	// sends escape sequences into whatever the user runs next, and unlike raw
	// mode there is no `reset` reflex for it because the shell still looks fine.
	r.leaveMouse()
	// Bracketed paste goes back for the same reason and in the same breath: a
	// terminal left bracketing pastes types ESC[200~ into the next program, and
	// there is no reflex for that either.
	r.leavePaste()
	// The alternate screen goes next, so the terminal is back on the normal
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
	return readInput(ctx, r, interrupts, nil)
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
	if r == nil || r.control == nil || r.alt {
		return
	}
	if _, err := fmt.Fprint(r.control, altScreenOn); err != nil {
		// Not recorded as entered, so restore does not send a leave for a screen
		// the terminal never showed. A dropped error here would make the flag a
		// claim about a write rather than about the terminal.
		return
	}
	r.alt = true
}

// leaveAlt returns to the normal screen. Idempotent, for the same reason
// restore is: it runs from more than one path and claiming a second call is an
// error would make the paths care about each other.
func (r *rawSession) leaveAlt() {
	if r == nil || r.control == nil || !r.alt {
		return
	}
	fmt.Fprint(r.control, altScreenOff)
	r.alt = false
}

// Button-event motion tracking (1002) and SGR coordinates (1006). Held drags
// belong to the shared selection router; idle hover is not reported.
const (
	mouseOn  = "\x1b[?1002h\x1b[?1006h"
	mouseOff = "\x1b[?1006l\x1b[?1002l"
)

// enterMouse asks the terminal to report the mouse.
//
// On rawSession for the same reason enterAlt is: restoration has to be one
// guarantee rather than three that each cover part of the exit paths.
func (r *rawSession) enterMouse() {
	if r == nil || r.control == nil || r.mouse {
		return
	}
	if _, err := fmt.Fprint(r.control, mouseOn); err != nil {
		return // as enterAlt: the flag records the terminal's state, not the attempt
	}
	r.mouse = true
}

// leaveMouse stops it. Idempotent, like the rest of restore.
func (r *rawSession) leaveMouse() {
	if r == nil || r.control == nil || !r.mouse {
		return
	}
	fmt.Fprint(r.control, mouseOff)
	r.mouse = false
}

// Bracketed paste (2004). The terminal wraps pasted text in ESC[200~ / ESC[201~
// so a program can tell it from typing — which is the whole point: without it a
// pasted newline is a carriage return and submits the line mid-paste.
const (
	pasteOn  = "\x1b[?2004h"
	pasteOff = "\x1b[?2004l"
)

// enabledModes is every mode the program asks a terminal for, paired with the
// sequence that gives it back.
//
// ONE list, so the guards DERIVE the set instead of hand-writing a case per mode.
// #67 found the cost of not having it twice over: mode 2004 was nearly enabled
// where TestEveryEnabledInputModeIsDecoded could not see it (it read mouseOn
// alone), and then enterPaste shipped as production wiring no test exercised —
// deleting the call left the whole suite green while the milestone silently
// reverted.
//
// Each entry earns two assertions: newConsole WRITES it, and a real terminal
// gets it back (pty_conformance_test.go). Both are loops over this slice.
var enabledModes = []struct{ name, on, off string }{
	{"alternate screen", altScreenOn, altScreenOff},
	{"mouse reporting", mouseOn, mouseOff},
	{"bracketed paste", pasteOn, pasteOff},
}

// enterPaste asks the terminal to bracket pastes.
//
// A sibling of enterMouse in every respect, including the one that matters: the
// flag is set only on a successful WRITE, so restore never sends a leave for a
// mode the terminal never entered.
func (r *rawSession) enterPaste() {
	if r == nil || r.control == nil || r.paste {
		return
	}
	if _, err := fmt.Fprint(r.control, pasteOn); err != nil {
		return
	}
	r.paste = true
}

// leavePaste stops it. Idempotent, like the rest of restore.
func (r *rawSession) leavePaste() {
	if r == nil || r.control == nil || !r.paste {
		return
	}
	fmt.Fprint(r.control, pasteOff)
	r.paste = false
}

// winSize is the terminal's shape, measured where the signal arrives so the
// editor loop never needs a terminal handle of its own.
type winSize struct{ rows, cols int }

// watchResize reports the terminal's new shape on every SIGWINCH (#30 M1.4).
//
// There was NO resize handling before the screen: the width was read once at
// flag parse and a definition kept the wrapping it was rendered with. A
// full-screen program cannot get away with that — it draws a frame for a height,
// and a frame one row too tall makes the terminal scroll, which moves every row
// the app believes it placed.
//
// Signals in, a measured SHAPE out: the loop's select grows one case and learns
// nothing about os/signal. The channel is the same shape as `keys` for the same
// reason.
func watchResize(ctx context.Context, notify func(...os.Signal) <-chan os.Signal, measure func() winSize) <-chan winSize {
	out := make(chan winSize, 1)
	if notify == nil {
		return out // no signal transport: the shape is whatever it was at startup
	}
	sigs := notify(syscall.SIGWINCH)
	// NOT signal.Stop'd on the way out, and that is a consequence of the seam
	// rather than an oversight: notify hands back a RECEIVE-only channel so a
	// test can supply any source at all, and signal.Stop needs the bidirectional
	// one. Widening the seam to reclaim a registration that lives as long as the
	// process buys nothing here; the day a watcher outlives its program, the
	// seam is what changes.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-sigs:
				if !open {
					return
				}
				// COALESCED. Dragging a window corner fires dozens of signals,
				// and a queue of stale shapes is a queue of wrong frames — only
				// the newest is true. So an unread shape is replaced rather than
				// waited on, which also means this goroutine can never block on
				// a loop that is busy playing a recording.
				// Drop whatever is waiting, then offer the new shape. Two
				// steps, not four: this is the only producer, so nothing can
				// fill the slot between them.
				select {
				case <-out:
				default:
				}
				out <- measure()
			}
		}
	}()
	return out
}
