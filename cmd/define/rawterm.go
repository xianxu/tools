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
	// modes is which modes this session actually took, by name.
	//
	// A MAP rather than three bools: the enter/leave pairs and the teardown order
	// are derived from enabledModes, and a bool per mode is the parallel
	// enumeration that derivation exists to remove.
	modes map[string]bool
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
	// Every mode goes back, in enabledModes' own order — which IS the teardown
	// order, and the reasons are stated there rather than restated here.
	r.leaveModes()
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

// Button-event motion tracking (1002) and SGR coordinates (1006). Held drags
// belong to the shared selection router; idle hover is not reported.
const (
	mouseOn  = "\x1b[?1002h\x1b[?1006h"
	mouseOff = "\x1b[?1006l\x1b[?1002l"
)

// Bracketed paste (2004). The terminal wraps pasted text in ESC[200~ / ESC[201~
// so a program can tell it from typing — which is the whole point: without it a
// pasted newline is a carriage return and submits the line mid-paste.
const (
	pasteOn  = "\x1b[?2004h"
	pasteOff = "\x1b[?2004l"
)

// enabledModes is every mode the program asks a terminal for, IN TEARDOWN ORDER.
//
// ONE list, so everything derives from it instead of hand-writing a case per
// mode. #67 paid for not having it three times: mode 2004 was nearly enabled
// where the decoder guard could not see it, the paste enable shipped as wiring no test
// exercised, and the teardown order lived only in restore()'s call sequence.
//
// The ORDER is the teardown order, and it is load-bearing. Mouse reporting goes
// back first: a terminal left reporting it writes escapes into whatever runs
// next, and unlike raw mode there is no `reset` reflex because the shell still
// looks fine. Bracketed paste carries the identical hazard. The alternate screen
// goes last, before raw mode ends, so a cooked terminal never briefly draws into
// a buffer about to be discarded. Entering runs the list in reverse, so the alt
// screen is taken first and the input modes apply to it.
//
// `replies` marks a mode the terminal ANSWERS in. Those owe the decoder a case,
// which TestEveryEnabledInputModeIsDecoded derives from here; the alternate
// screen replies with nothing, so it has no encoding to decode.
var enabledModes = []struct {
	name, on, off string
	replies       bool
}{
	{"mouse reporting", mouseOn, mouseOff, true},
	{"bracketed paste", pasteOn, pasteOff, true},
	{"alternate screen", altScreenOn, altScreenOff, false},
}

// backgroundQuery asks the terminal for its background colour (OSC 11, #70).
// The answer arrives as input, whenever it arrives; nothing waits for it.
const backgroundQuery = "\x1b]11;?\x1b\\"

// terminalQueries is every QUESTION this program asks a terminal. Not modes —
// there is nothing to tear down — but they share enabledModes' obligation: a
// terminal that answers owes the decoder a case, and
// TestEveryEnabledInputModeIsDecoded derives from both lists. Every query here
// is about the tint today, so one condition (wantsBackground) governs them all;
// a query with a different condition gets its own.
var terminalQueries = []struct{ name, query string }{
	{"background colour", backgroundQuery},
}

// ask writes a query where the modes go. NEVER through a screen: scanEscape
// reads ESC ] as a 2-byte escape, so the rest would be painted as text.
func (r *rawSession) ask(query string) {
	if r == nil || r.control == nil {
		return
	}
	fmt.Fprint(r.control, query)
}

// wantsBackground: ask only where a tint can appear — colour on, the tint on,
// not -raw. TERM=dumb already turned tintOn off at flag parse.
func wantsBackground(opt options) bool { return opt.tty && opt.color && opt.tintOn && !opt.raw }

// enterModes takes every mode, in reverse teardown order.
//
// A LOOP over the list rather than three calls, so a mode added to the list is
// taken without anyone remembering to add a fourth call — which is the failure
// the paste enable already had once.
func (r *rawSession) enterModes() {
	for i := len(enabledModes) - 1; i >= 0; i-- {
		m := enabledModes[i]
		r.enterMode(m.name, m.on)
	}
}

// enterMode writes one mode and records it ONLY on a successful write, so
// restore never sends a leave for a mode the terminal never entered.
func (r *rawSession) enterMode(name, on string) {
	if r == nil || r.control == nil || r.modes[name] {
		return
	}
	if _, err := fmt.Fprint(r.control, on); err != nil {
		return
	}
	if r.modes == nil {
		r.modes = map[string]bool{}
	}
	r.modes[name] = true
}

// leaveModes gives them all back, in the list's own order.
func (r *rawSession) leaveModes() {
	for _, m := range enabledModes {
		if r == nil || r.control == nil || !r.modes[m.name] {
			continue
		}
		fmt.Fprint(r.control, m.off)
		r.modes[m.name] = false
	}
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
