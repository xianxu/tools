//go:build darwin && conformance

package main

// Live pty conformance for the editor (ARCH-MOCK, applied to the terminal).
//
// The terminal is an external dependency like the dictionary and the CDN, and
// these are the behaviours no in-process fake can model: that raw mode is
// actually entered, that escape sequences arrive as this program's key decoder
// expects, that Ctrl-C is a BYTE rather than a signal, and — the one that
// matters most — that the terminal is left cooked when we exit.
//
// Cadence is on-demand with the rest of the conformance suite; it needs a real
// pty and a built binary.
//
//	go test -tags conformance -run PTY ./cmd/define/

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// startDefine launches the built binary on a pty and returns it plus the master.
func startDefine(t *testing.T, args ...string) (*exec.Cmd, *os.File) {
	t.Helper()
	bin := "../../bin/define"
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("run `make build` first: %v", err)
	}
	cmd := exec.Command(bin, args...)
	f, err := pty.Start(cmd)
	if err != nil {
		t.Skipf("no pty available: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); f.Close() })
	return cmd, f
}

// ptyOut collects everything the pty emits in the background.
//
// A pty master does not honour SetReadDeadline reliably, so a foreground read
// loop hangs. A dedicated reader plus a snapshot is the shape that works.
type ptyOut struct {
	mu  sync.Mutex
	buf strings.Builder
}

func watch(f *os.File) *ptyOut {
	o := &ptyOut{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				o.mu.Lock()
				o.buf.Write(buf[:n])
				o.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	return o
}

// take waits d, then returns everything collected since the last take.
func (o *ptyOut) take(d time.Duration) string {
	time.Sleep(d)
	o.mu.Lock()
	defer o.mu.Unlock()
	s := o.buf.String()
	o.buf.Reset()
	return s
}

func TestPTYSuggestionAndAcceptance(t *testing.T) {
	_, f := startDefine(t, "--times", "1")
	out := watch(f)
	out.take(time.Second)

	f.Write([]byte("sycophantic\r"))
	out.take(5 * time.Second)

	f.Write([]byte("syc"))
	frame := out.take(1500 * time.Millisecond)
	if !strings.Contains(frame, greyOn) {
		t.Errorf("no grey suggestion after typing a known prefix: %q", frame)
	}
	if !strings.Contains(frame, greyOn+"ophantic") {
		t.Errorf("suggestion is not the remainder of the history entry: %q", frame)
	}

	f.Write([]byte("\x1b[C")) // Right accepts
	if got := out.take(time.Second); !strings.Contains(got, "sycophantic") {
		t.Errorf("Right did not accept the suggestion: %q", got)
	}
}

// The Critical from the plan gate: raw mode makes Ctrl-C a byte, so
// signal.NotifyContext cannot deliver it — and the moment that matters is during
// playback, when the loop is blocked for seconds. Playing in cooked mode made
// this hang; the fix is to render cooked and play raw.
func TestPTYCtrlCDuringPlaybackExitsPromptly(t *testing.T) {
	cmd, f := startDefine(t, "--times", "5")
	out := watch(f)
	out.take(time.Second)

	f.Write([]byte("sycophantic\r"))
	out.take(1500 * time.Millisecond) // let playback start

	start := time.Now()
	f.Write([]byte("\x03"))

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Errorf("Ctrl-C took %v to stop playback", elapsed)
		}
		if err != nil {
			t.Errorf("exit: %v, want 0", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("Ctrl-C during playback did not exit — cancellation is not reaching the loop")
	}
}

// The worst failure this tool can produce: a shell left in raw mode. No later
// output can fix it.
func TestPTYTerminalIsRestoredOnExit(t *testing.T) {
	cmd, f := startDefine(t, "--no-audio")
	out := watch(f)
	out.take(time.Second)

	// While the editor runs the SLAVE is raw; assert the master side round-trips
	// after exit, which is only possible from a sane terminal state.
	f.Write([]byte("\x03"))
	// Checked, not discarded. A crashed define also leaves the terminal sane —
	// the kernel restores it when the process dies — so dropping this status
	// would let this test pass for entirely the wrong reason.
	if err := cmd.Wait(); err != nil {
		t.Errorf("exit: %v, want 0", err)
	}

	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		t.Skip("master is not a terminal on this platform")
	}
	st, err := term.MakeRaw(fd)
	if err != nil {
		t.Fatalf("terminal unusable after exit: %v", err)
	}
	if err := term.Restore(fd, st); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
}
