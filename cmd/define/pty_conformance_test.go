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
	"fmt"
	"github.com/xianxu/tools/internal/conformance"
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
	// define writes its deck to the CURRENT directory, and a test's cwd is the
	// PACKAGE directory — so this suite used to write a deck into the source
	// tree and then rewrite it on every run, which is how three deck files
	// reached commits. The binary path is absolute for exactly this reason.
	return startDefineInDir(t, t.TempDir(), env, args...)
}

// startDefineInDir is startDefineWithEnv with the deck directory named, so two
// runs can share one deck.
//
// --play needs a deck that already has a due word in it, and the only honest way
// to get one is to let define capture it: a hand-written words/ file would pin
// this test to a storage format rather than to the behaviour.
func startDefineInDir(t *testing.T, dir string, env []string, args ...string) (*exec.Cmd, *os.File) {
	t.Helper()
	bin := builtBinary(t)
	cmd := exec.Command(bin, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Dir = dir
	f, err := pty.Start(cmd)
	if err != nil {
		conformance.SkipOrFail(t, "no pty available", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); f.Close() })
	return cmd, f
}

// builtBinary is the binary these tests drive, BUILT FROM THE SOURCE IN THE TREE.
//
// It used to be whatever `../../bin/define` happened to be, with a skip if it
// was absent and no check that it was current — so a conformance run could
// validate pre-fix code and report ok. Measured once: the binary on disk was 34
// minutes older than the fix commit under test. A live conformance check that
// cannot fail on the change it exists to check is worse than no check, because
// it reads as evidence.
//
// Built once per run into the test's own temp space, so it cannot be stale and
// cannot collide with `make build`'s output.
var builtBinaryOnce struct {
	sync.Once
	path string
	err  error
}

func builtBinary(t *testing.T) string {
	t.Helper()
	builtBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "define-pty")
		if err != nil {
			builtBinaryOnce.err = err
			return
		}
		path := filepath.Join(dir, "define")
		out, err := exec.Command("go", "build", "-o", path, ".").CombinedOutput()
		if err != nil {
			builtBinaryOnce.err = fmt.Errorf("building define: %v\n%s", err, out)
			return
		}
		builtBinaryOnce.path = path
	})
	if builtBinaryOnce.err != nil {
		t.Fatalf("%v", builtBinaryOnce.err)
	}
	return builtBinaryOnce.path
}

// ptyOut collects everything the pty emits in the background.
//
// A pty master does not honour SetReadDeadline reliably, so a foreground read
// loop hangs. A dedicated reader plus a snapshot is the shape that works — and
// it is the same shape syncBuf provides for the in-process tests, so the
// locking lives in one place rather than two (ARCH-DRY).
type ptyOut struct {
	buf syncBuf
}

func watch(f *os.File) *ptyOut {
	o := &ptyOut{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				o.buf.Write(buf[:n])
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
	return o.buf.TakeAll()
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

	// #20, on a real terminal: the same grey tail mid-sentence. Before #20 the
	// suggestion matched only against the WHOLE line, so it vanished the moment
	// a space was typed — which is exactly where a question gets asked.
	f.Write([]byte("\x15")) // Ctrl-U clears the accepted line
	out.take(500 * time.Millisecond)

	f.Write([]byte("what is a syc"))
	frame = out.take(1500 * time.Millisecond)
	if !strings.Contains(frame, greyOn+"ophantic") {
		t.Errorf("no mid-sentence grey suggestion: %q", frame)
	}

	// Asserted on the TEXT, with styling stripped. The literal-bytes version of
	// this check went red the moment #21 began highlighting deck words inside the
	// typed line: the frame reads "what is a \x1b[1;32msycophantic", so Tab had
	// accepted and the assertion could not see it. Both #20 and #21 merged with
	// this test red, because the conformance suite is on-demand and neither close
	// ran it — a live check that is never run is not a check.
	f.Write([]byte("\t")) // Tab accepts
	if got := out.take(time.Second); !strings.Contains(unstyled(got), "what is a sycophantic") {
		t.Errorf("Tab did not accept mid-sentence: %q", got)
	}
	f.Write([]byte("\x15")) // leave the prompt clean for the next assertion
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
		conformance.SkipOrFail(t, "master is not a terminal on this platform", nil)
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

// --play renders every line at COLUMN 0 on a real terminal.
//
// This is the check that was missing when --play shipped, and its absence is the
// reason the operator found the defect instead of the suite (BR-45). In raw mode
// ONLCR is off, so the tty performs no newline translation: a bare \n written by
// the program arrives at the master as a bare \n and the cursor stays where the
// previous line ended. That is the CRLF cascade — each line starting further
// right than the last, which is what the screenshot showed.
//
// No in-process test can see it. The format strings were correct; it was
// Render's output flowing through a writer that did not translate, and a test
// capturing bytes and printing them through anything OTHER than a raw terminal
// renders \n at column 0 and reports success. The smoke run that "verified" this
// did exactly that. The terminal is an external dependency, and this is the
// observable only it produces (ARCH-MOCK).
func TestPTYPlayRendersEveryLineAtColumnZero(t *testing.T) {
	deck := t.TempDir()

	// Capture a word so the deck has one, the way a learner gets one. A word
	// never reviewed is due immediately (schedule.Queue's fresh tier).
	// Flags BEFORE the word: Go's flag package stops parsing at the first
	// non-flag argument, so "sycophantic --no-audio" prints usage and looks up
	// nothing.
	_, seed := startDefineInDir(t, deck, nil, "--no-audio", "sycophantic")
	seeded := watch(seed).take(3 * time.Second)
	if !strings.Contains(seeded, "sikəˈfan(t)ik") {
		// The DICTIONARY is the absent dependency here, not a defect in the
		// flow under test: DCSCopyTextDefinition returns silence, not an error,
		// without real access to /System/Library/AssetsV2. Written as a hard
		// Fatal this was BR-9's mirror on a darwin host with a pty and no
		// dictionary assets — the non-strict suite red rather than skipped.
		conformance.SkipOrFail(t, fmt.Sprintf(
			"the seeding lookup did not resolve, so no deck was written:\n%q", seeded), nil)
	}
	seed.WriteString("\x04") // EOF: leave the editor, flushing the capture

	_, f := startDefineInDir(t, deck, nil, "--play", "--no-audio")
	out := watch(f)
	shown := out.take(3 * time.Second)
	if !strings.Contains(shown, "sycophantic") {
		t.Fatalf("--play never offered the seeded word; the deck or the queue is the problem:\n%q", shown)
	}

	f.WriteString(" ") // reveal, so the definition renders too
	shown += out.take(2 * time.Second)
	f.WriteString("y")
	shown += out.take(2 * time.Second)

	if bad := bareNewlines(shown); bad != 0 {
		t.Errorf("%d bare newline(s) in --play's output — in raw mode each one leaves the "+
			"cursor where the previous line ended, so every following line starts further "+
			"right. Output:\n%q", bad, shown)
	}
}

// bareNewlines counts \n not preceded by \r.
//
// Written as a count rather than a bool so the failure says how far the cascade
// went, which is the difference between one missed format string and a writer
// that translates nothing.
func bareNewlines(s string) int {
	n := 0
	for i := range s {
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			n++
		}
	}
	return n
}

// unstyled drops SGR sequences so a text assertion survives a styling change.
//
// Only the colour sequences: cursor movement and erasure are what several tests
// here are ABOUT, and stripping those would make those assertions vacuous.
var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func unstyled(s string) string { return sgr.ReplaceAllString(s, "") }

// The grade-first flow on a real terminal.
//
// #6 BR-45 is why this exists in the same milestone as the change rather than
// after it: --play's one shipped defect was invisible to every in-process test
// AND to a byte-capture smoke run, and the operator found it.
// seedDeck captures one word into a fresh deck the way a learner gets one, and
// returns the deck directory.
//
// A word never reviewed is due immediately (schedule.Queue's fresh tier). Flags
// go BEFORE the word: Go's flag package stops parsing at the first non-flag
// argument, so "sycophantic --no-audio" prints usage and looks up nothing.
func seedDeck(t *testing.T) string {
	t.Helper()
	deck := t.TempDir()
	_, seed := startDefineInDir(t, deck, nil, "--no-audio", "sycophantic")
	seeded := watch(seed).take(3 * time.Second)
	if !strings.Contains(seeded, "sikəˈfan(t)ik") {
		// The DICTIONARY is the absent dependency here, not a defect in the flow
		// under test: DCSCopyTextDefinition returns silence, not an error,
		// without real access to /System/Library/AssetsV2. Written as a hard
		// Fatal this was BR-9's mirror on a darwin host with a pty and no
		// dictionary assets — the non-strict suite red rather than skipped.
		conformance.SkipOrFail(t, fmt.Sprintf(
			"the seeding lookup did not resolve, so no deck was written:\n%q", seeded), nil)
	}
	seed.WriteString("\x04") // EOF: leave the editor, flushing the capture
	return deck
}

// A CORRECT answer costs one keystroke and never shows the definition — on a
// REAL terminal.
//
// This is the keystroke the issue exists to make free, and it was asserted only
// in-process: the pty check drove `n` and space, so the Done-when row "a pty
// conformance test drives the new flow" was satisfied by the artifact existing
// rather than by the behaviour running. Its own session and its own deck, so the
// draw order cannot make it flaky.
func TestPTYPlayCorrectAnswerNeverRevealsIt(t *testing.T) {
	deck := seedDeck(t)

	_, f := startDefineInDir(t, deck, nil, "--play", "--no-audio")
	out := watch(f)
	first := out.take(3 * time.Second)
	if !strings.Contains(unstyled(first), "y = got it") {
		t.Fatalf("the grading keys were not offered up front:\n%q", first)
	}

	f.WriteString("y")
	after := out.take(3 * time.Second)
	if strings.Contains(unstyled(first+after), "sikəˈfan(t)ik") {
		t.Errorf("a correct answer showed the definition; that is the step #24 removes:\n%q", after)
	}
	if !strings.Contains(unstyled(after), "1 right, 0 wrong") {
		t.Errorf("y did not record a hit and end the sitting:\n%q", after)
	}
	if bad := bareNewlines(first + after); bad != 0 {
		t.Errorf("%d bare newline(s) — the CRLF cascade is back", bad)
	}
}

func TestPTYPlayGradeFirst(t *testing.T) {
	deck := seedDeck(t)

	_, f := startDefineInDir(t, deck, nil, "--play", "--no-audio")
	out := watch(f)
	first := out.take(3 * time.Second)
	if !strings.Contains(unstyled(first), "y = got it") {
		t.Fatalf("the grading keys were not offered up front:\n%q", first)
	}
	if strings.Contains(unstyled(first), "sikəˈfan(t)ik") {
		t.Errorf("the definition was showing before the learner answered:\n%q", first)
	}

	f.WriteString("n")
	missed := out.take(2 * time.Second)
	if !strings.Contains(unstyled(missed), "sikəˈfan(t)ik") {
		t.Errorf("n did not put the definition on screen:\n%q", missed)
	}
	if !strings.Contains(unstyled(missed), "any key = next word") {
		t.Errorf("the graded prompt did not appear:\n%q", missed)
	}

	// PQ-2 on a real terminal: space is what a learner presses, and it travels
	// as InputReveal rather than as a rune. This is the assertion that would
	// have caught it dead.
	f.WriteString(" ")
	moved := out.take(3 * time.Second)
	if !strings.Contains(unstyled(moved), "0 right, 1 wrong") {
		t.Errorf("space did not move on from the missed word:\n%q", moved)
	}
	if bad := bareNewlines(first + missed + moved); bad != 0 {
		t.Errorf("%d bare newline(s) — the CRLF cascade is back", bad)
	}
}

// Mouse reporting is asked for, and given back (#30 M1.4b).
//
// Giving it back matters more than turning it on: a terminal left reporting the
// mouse writes escape sequences into whatever the user runs next, and unlike raw
// mode there is no `reset` reflex for it, because the shell still looks fine. It
// rides rawSession's restore for exactly that reason, and this is the row that
// says the guarantee reaches a real terminal.
func TestPTYMouseTrackingIsAskedForAndGivenBack(t *testing.T) {
	cmd, f := startDefine(t, "--no-audio")
	out := watch(f)
	started := out.take(time.Second)

	if !strings.Contains(started, mouseOn) {
		t.Errorf("the session never enabled mouse reporting, so the wheel arrives as arrow keys: %q", started)
	}
	f.Write([]byte("\x03"))
	if err := cmd.Wait(); err != nil {
		t.Errorf("exit: %v, want 0", err)
	}
	if rest := out.take(time.Second); !strings.Contains(started+rest, mouseOff) {
		t.Errorf("mouse reporting was left ON: the next program run in this terminal gets escape sequences typed into it: %q", rest)
	}
}

// A resize repaints, driven by a REAL SIGWINCH (#30 M1.4).
//
// This is the one part of the resize path no in-process test can reach: the
// signal itself. TestWatchResizeCoalesces drives the channel, and
// TestEditorResizeRedrawsForTheNewShape drives the loop — but whether a window
// change actually delivers SIGWINCH to this program, and whether the frame that
// follows fits the new window, is a question only a terminal answers.
func TestPTYResizeRepaints(t *testing.T) {
	_, f := startDefine(t, "--no-audio")
	out := watch(f)
	if err := pty.Setsize(f, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		conformance.SkipOrFail(t, "cannot size the pty on this platform", err)
	}
	out.take(time.Second)

	// A long entry, so the buffer is taller than the window it is about to get.
	f.Write([]byte("run\r"))
	out.take(3 * time.Second)

	if err := pty.Setsize(f, &pty.Winsize{Rows: 10, Cols: 80}); err != nil {
		t.Fatalf("resize: %v", err)
	}
	after := out.take(2 * time.Second)
	if !strings.Contains(after, cursorHome) {
		t.Fatalf("nothing was repainted after the window changed — SIGWINCH never reached the select: %q", after)
	}
	// And the frame FITS: a frame drawn for the old height is too tall, the
	// terminal scrolls to fit it, and every row the app believes it placed has
	// moved — which is exactly what makes a click land on the wrong line.
	frame := after[strings.LastIndex(after, cursorHome):]
	if rows := strings.Count(frame, "\r\n") + 1; rows > 10 {
		t.Errorf("the frame is %d rows in a 10-row window: %q", rows, frame)
	}
}

// The session survives the alternate screen (#30 D3, M1.5).
//
// The alt buffer is discarded on the way out, so without the transcript
// everything a session showed is gone the moment it ends — and `define
// arrondissement` used to leave the entry where you could scroll back to it
// tomorrow, or copy from it.
//
// The assertion is deliberately anchored AFTER the teardown sequence: the entry
// appearing anywhere in the stream proves only that it was drawn on the screen
// that is about to be thrown away.
func TestPTYTranscriptIsPrintedOnExit(t *testing.T) {
	cmd, f := startDefine(t, "--no-audio")
	out := watch(f)
	out.take(time.Second)

	f.Write([]byte("sycophantic\r"))
	if drawn := out.take(3 * time.Second); !strings.Contains(drawn, "sikəˈfan(t)ik") {
		t.Fatalf("the word was never defined, so its survival proves nothing: %q", drawn)
	}

	f.Write([]byte("\x03"))
	if err := cmd.Wait(); err != nil {
		t.Errorf("exit: %v, want 0", err)
	}
	rest := out.take(time.Second)
	i := strings.Index(rest, altScreenOff)
	if i < 0 {
		t.Fatalf("the alternate screen was never left: %q", rest)
	}
	after := rest[i:]
	if !strings.Contains(after, "sikəˈfan(t)ik") {
		t.Errorf("the session vanished with the alternate screen — nothing to scroll back to: %q", after)
	}
	// The line the user typed comes back too: an entry with no word above it
	// reads as scrollback from nowhere.
	if !strings.Contains(after, "sycophantic") {
		t.Errorf("the committed line is not in the transcript: %q", after)
	}
}

// A terminal that never reports a mouse behaves exactly as M1 did (#30 M2.6).
//
// The enable is emitted unconditionally, because there is no reliable way to ask
// a terminal whether it will honour it — DECRQM is a round trip terminals answer
// inconsistently, and a private mode nobody implements is ignored rather than
// echoed. So the degrade is not a code path: it is the absence of input. What
// this row asserts is that the absence costs nothing — the session still looks
// words up, still scrolls by key, and still leaves the terminal sane.
func TestPTYWithoutMouseBehavesAsBefore(t *testing.T) {
	cmd, f := startDefine(t, "--no-audio")
	out := watch(f)
	out.take(time.Second)

	// A whole session, and not one mouse report in it.
	f.Write([]byte("sycophantic\r"))
	if got := out.take(3 * time.Second); !strings.Contains(got, "sikəˈfan(t)ik") {
		t.Fatalf("the lookup did not answer: %q", got)
	}
	f.Write([]byte("\x1b[5~")) // PageUp still scrolls
	if got := out.take(time.Second); !strings.Contains(got, cursorHome) {
		t.Errorf("the viewport did not move without a mouse: %q", got)
	}
	f.Write([]byte("syc"))
	if got := out.take(time.Second); !strings.Contains(got, greyOn) {
		t.Errorf("suggestions stopped working without a mouse: %q", got)
	}

	f.Write([]byte("\x03"))
	if err := cmd.Wait(); err != nil {
		t.Errorf("exit: %v, want 0", err)
	}
	rest := out.take(time.Second)
	if !strings.Contains(rest, mouseOff) {
		t.Errorf("tracking was left on by a session that never saw a mouse: %q", rest)
	}
}
