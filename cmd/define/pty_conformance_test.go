//go:build darwin && conformance

package main

// Live pty conformance for the editor (ARCH-MOCK, applied to the terminal).
//
// The terminal is an external dependency like the dictionary and the CDN, and
// these are the behaviours no in-process fake can model: that raw mode is
// actually entered, that escape sequences arrive as this program's key decoder
// expects, and — the one that matters most — that the terminal is left cooked
// when we exit.
//
// **This suite does NOT pin "Ctrl-C is a byte rather than a signal", though it
// used to claim it did.** Measured: mutating replRaw's `case ActInterrupt,
// ActEOF:` to return 9 leaves all three tests here GREEN, while mutating
// `case <-ctx.Done():` to return 7 reddens them — so the \x03 written to the
// master reaches define as a SIGINT and it leaves through NotifyContext, not
// through the key reader. "Exited, and the terminal is sane" is an observable
// that the byte path, the signal path and a crash all produce, so asserting it
// separates only the crash.
//
// The byte path is pinned in-process by TestEditorLoopCtrlCExitsZero, which
// scripts "syc\x03" through runEditor: the same mutation reddens it with
// "exit = 9, want 0" (verified).
//
// #16 closed the OTHER half of that gap. TestPTYCtrlCMidAnswerKeepsTheSession
// asserts an observable only a SURVIVING session produces — the answer stopped
// AND the next lookup rendered — which "exited cleanly" cannot imitate;
// unscoping the interrupt reddens it through a real pty (verified). It still
// does not distinguish byte from signal, and by design cannot: both transports
// feed one sink (#16 D5), so the outcome is the same either way. What it pins is
// that the scope reaches a real terminal, which no in-process test can say.
//
// Cadence is on-demand with the rest of the conformance suite; it needs a real
// pty and a built binary.
//
//	go test -tags conformance -run PTY ./cmd/define/

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/tools/internal/llm/llmtest"
	"golang.org/x/term"
)

// startDefine launches the built binary on a pty and returns it plus the master.
func startDefine(t *testing.T, args ...string) (*exec.Cmd, *os.File) {
	return startDefineWithEnv(t, nil, args...)
}

// startDefineWithEnv is startDefine plus environment, so a test can point the
// binary's model seam at a fake served from this process.
func startDefineWithEnv(t *testing.T, env []string, args ...string) (*exec.Cmd, *os.File) {
	t.Helper()
	bin, err := filepath.Abs("../../bin/define")
	if err != nil {
		t.Fatalf("resolving the binary: %v", err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("run `make build` first: %v", err)
	}
	cmd := exec.Command(bin, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	// define writes its deck to the CURRENT directory, and a test's cwd is the
	// PACKAGE directory — so this suite used to write a deck into the source
	// tree and then rewrite it on every run, which is how three deck files
	// reached commits. The binary path is absolute for exactly this reason.
	cmd.Dir = t.TempDir()
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

// The command menu against a real terminal. In-process tests prove the BYTES
// are emitted; only a pty proves the cursor arithmetic around them — the menu
// is drawn below the prompt and then the cursor is walked back up, and a
// miscount there leaves the user typing on top of the menu.
func TestPTYCommandMenuAppearsAndClears(t *testing.T) {
	_, f := startDefine(t, "--no-audio")
	out := watch(f)
	out.take(time.Second)

	f.Write([]byte("/"))
	frame := out.take(1500 * time.Millisecond)
	if !strings.Contains(frame, "list the commands") {
		t.Errorf("typing / did not draw the menu: %q", frame)
	}
	// Drawn BELOW: the cursor must come back up by exactly as many rows as were
	// drawn, or the prompt lands in the middle of the menu.
	//
	// Derived from the frame, never hardcoded. The first version asserted
	// "\x1b[1A" and broke the moment a second command was registered — the row
	// count is incidental, the EQUALITY is the invariant.
	rows := strings.Count(frame, "\r\n")
	m := regexp.MustCompile(`\x1b\[(\d+)A`).FindStringSubmatch(frame)
	if m == nil {
		t.Fatalf("the menu never moved the cursor back up: %q", frame)
	}
	if up, _ := strconv.Atoi(m[1]); up != rows {
		t.Errorf("drew %d rows but came back up %d: %q", rows, up, frame)
	}

	// A prefix that matches nothing must clear it.
	f.Write([]byte("zzz"))
	frame = out.take(1500 * time.Millisecond)
	if strings.Contains(frame, "list the commands") {
		t.Errorf("a non-matching prefix redrew the menu: %q", frame)
	}

	f.Write([]byte("\x03"))
}

// Ctrl-C mid-answer returns to the prompt with the session INTACT.
//
// This is the row the header above says the other three cannot supply. They
// assert "exited, and the terminal is sane" — an observable the byte path, the
// signal path and a crash all produce alike, which is why mutating the byte
// branch left them green. "The answer stopped AND the next lookup rendered" is
// produced only by a session that survived, so it separates what they cannot.
//
// The model seam points at a wire-level fake served from this process, so the
// answer is a real recorded stream and the interrupt lands mid-flight.
func TestPTYCtrlCMidAnswerKeepsTheSession(t *testing.T) {
	fake := llmtest.NewFake(t)
	fake.Script("", llmtest.Reply{Capture: "stream-sample.sse", Stall: true})

	_, f := startDefineWithEnv(t, []string{
		"DEFINE_LLM_BASE_URL=" + fake.URL,
		"DEFINE_LLM_API_KEY=pty-conformance",
	}, "--no-audio")
	out := watch(f)
	out.take(300 * time.Millisecond)

	// A forced question, so the dictionary is never consulted and the only way
	// to the model is the branch under test.
	f.WriteString("?why\r")
	answer := out.take(1500 * time.Millisecond)
	if !strings.Contains(answer, "Obsequious") {
		t.Fatalf("the answer never streamed; the seam may be misconfigured:\n%q", answer)
	}

	f.WriteString("\x03") // Ctrl-C: stop the answer, keep the session
	out.take(300 * time.Millisecond)

	f.WriteString("sycophantic\r")
	after := out.take(2 * time.Second)
	if !strings.Contains(after, "sikəˈfan(t)ik") {
		t.Errorf("the session did not survive the interrupt — no definition after Ctrl-C:\n%q", after)
	}
}
