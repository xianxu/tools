package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// scriptKeys turns a string into keypresses, with "\r" meaning Enter — the
// scripted key source the plan called for, so the editor loop is testable with
// no terminal anywhere.
func scriptKeys(s string) <-chan Key {
	ch := make(chan Key, len(s)+1)
	for _, r := range s {
		switch r {
		case '\r', '\n':
			ch <- Key{Kind: KeyEnter}
		case '\x03':
			ch <- Key{Kind: KeyInterrupt}
		default:
			ch <- Key{Kind: KeyRune, Rune: r}
		}
	}
	close(ch) // channel close == end of input
	return ch
}

func editorRig(t *testing.T, word string, audioPresent bool) (*audioRig, options, func()) {
	t.Helper()
	rig := newAudioRig(t, word, audioPresent)
	rig.deps.stdinIsTerminal = func() bool { return true }
	// finish is a no-op: there is no real terminal in a test.
	return rig, options{times: 3, locale: "us", tty: true, color: true}, func() {}
}

// recordDisplay is the display double, and it writes the frame where the loop's
// own output goes so a test still reads one session as one stream.
//
// In production the display is liveScreen, which puts the live edge — the prompt
// and the command menu — on a terminal rather than in the buffer (#30 D5), and
// moves a viewport nothing here has. A test has neither, so the halves fold back
// into one writer, which is exactly what stdout was before the screen existed.
// What these tests pin is the LOOP's decisions: what it writes, when it looks
// up, what it replays, and which keys reach the viewport. The screen's own
// arithmetic is pinned in screen_test.go, with no terminal either.
type recordDisplay struct {
	// mu, because a test that drives the loop on another goroutine reads these
	// while the loop writes them — `go test -race` reports it, and `waitFor`
	// spinning on an unsynchronised field can spin to its own timeout.
	mu sync.Mutex
	w  io.Writer
	// prompt and menu are the CURRENT live edge; prompts and menus are every one
	// the loop drew, in order. Both, because "what is on screen now" and "what
	// was on screen while X happened" are different questions.
	prompt  string
	menu    []string
	prompts []string
	menus   [][]string
	pages   []int
	lines   []int
	rows    []int
	cols    []int
	regions []Region
	at      map[[2]int]Region
	// footerRows scripts FooterRowAt: a viewport row to {entry, offset} — which
	// footer entry is drawn there and which of ITS physical rows. Empty means
	// "the live edge is not clickable here", which is the answer for every test
	// that predates the board.
	footerRows map[int][2]int
}

func paintInto(w io.Writer) *recordDisplay { return &recordDisplay{w: w} }

// recordingConsole is the console a test drives, and it takes ONE writer for the
// frame and stdout because that is not incidental: in production the display and
// both streams are a single liveScreen (#30 D5b), so a test that split them
// would read a session the loop never produced.
//
// Written once rather than at every call site, which is the point — the two
// io.Writer parameters below are positionally swappable exactly here, and
// nowhere else.
//
// Named for what it IS rather than for who drives it: #41 made `--play` the
// second loop taking a console, and a sitting's test building an "editorConsole"
// would read as the wrong loop.
//
// A test that ASSERTS on what was drawn keeps the struct literal instead and
// names its own recorder: `view.lastPrompt()` says what it means, and a literal
// with named fields cannot be swapped by position either.
func recordingConsole(out, errb io.Writer, finish func()) console {
	view := paintInto(out)
	// view AND stdout, one object — production's shape, where both are the same
	// liveScreen (#30 D5b). stderr stays its own buffer here, which is the one
	// deliberate departure: a test that wants to tell a diagnostic from the
	// session's output needs them apart, and the loop cannot tell.
	return console{view: view, finish: finish, stdout: view, stderr: errb}
}

// Write makes the recorder the console's STDOUT as well as its display, which
// is production's shape: there, both are one liveScreen. A double that split
// them would never be handed a click map, because the map travels with the text
// through the writer.
func (d *recordDisplay) Write(p []byte) (int, error) { return d.w.Write(p) }

func (d *recordDisplay) Draw(prompt string, menu []string) {
	d.mu.Lock()
	d.prompt, d.menu = prompt, menu
	d.prompts = append(d.prompts, prompt)
	d.menus = append(d.menus, menu)
	d.mu.Unlock()
	// Menu first, prompt last, in the order Paint puts them on a screen: the
	// dropdown hangs below the line you are typing.
	for _, m := range menu {
		fmt.Fprint(d.w, "\r\n"+m)
	}
	fmt.Fprint(d.w, prompt)
}

func (d *recordDisplay) Page(n int) {
	d.mu.Lock()
	d.pages = append(d.pages, n)
	d.mu.Unlock()
}

func (d *recordDisplay) Scroll(n int) {
	d.mu.Lock()
	d.lines = append(d.lines, n)
	d.mu.Unlock()
}

// WriteRegions records the click map a render offered, and writes the text where
// the loop's own output goes — so a test still reads one session as one stream.
func (d *recordDisplay) WriteRegions(text string, rs []Region) {
	d.mu.Lock()
	d.regions = append(d.regions, rs...)
	d.mu.Unlock()
	fmt.Fprint(d.w, text)
}

// RegionAtRow answers a click. A test SCRIPTS the answer — `at` maps a
// (row, col) to whatever should be found there — because what the loop's tests
// pin is what it DOES with a region, and where regions actually live is pinned
// on the real screen in screen_test.go.
func (d *recordDisplay) RegionAtRow(row, col int) (Region, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r, ok := d.at[[2]int{row, col}]
	return r, ok
}

// FooterRowAt is SCRIPTED the same way, and answers "none" until a test says
// otherwise — which is the editor's whole involvement with the live edge's click
// map (#40 D10). `footerAt` is how --play's tests put a board row under a click.
func (d *recordDisplay) FooterRowAt(row int) (int, int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.footerRows[row]
	return e[0], e[1], ok
}

// footerAt scripts a row as an entry's FIRST physical row; footerAtOffset
// scripts a continuation, which is what a wrapped entry produces and what a
// caller acting on a column has to refuse (R9).
func (d *recordDisplay) footerAt(row, entry int) { d.footerAtOffset(row, entry, 0) }

func (d *recordDisplay) footerAtOffset(row, entry, offset int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.footerRows == nil {
		d.footerRows = map[int][2]int{}
	}
	d.footerRows[row] = [2]int{entry, offset}
}

func (d *recordDisplay) offer(row, col int, r Region) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.at == nil {
		d.at = map[[2]int]Region{}
	}
	d.at[[2]int{row, col}] = r
}

// collected is every region the loop was handed, for a test that asserts the
// entry really did carry a click map.
func (d *recordDisplay) collected() []Region {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Region(nil), d.regions...)
}

func (d *recordDisplay) Resize(rows, cols int) {
	d.mu.Lock()
	d.rows = append(d.rows, rows)
	d.cols = append(d.cols, cols)
	d.mu.Unlock()
}

// The readers a test uses, so nothing reaches a field without the lock.

func (d *recordDisplay) livePrompt() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.prompt
}

func (d *recordDisplay) lastPrompt() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.prompts) == 0 {
		return ""
	}
	return d.prompts[len(d.prompts)-1]
}

func (d *recordDisplay) drawnMenus() [][]string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([][]string(nil), d.menus...)
}

func (d *recordDisplay) scrolls() (pages, lines, rows []int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]int(nil), d.pages...), append([]int(nil), d.lines...), append([]int(nil), d.rows...)
}

func TestEditorLoopDefinesTypedWord(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "/ˌsikəˈfan(t)ik/") {
		t.Error("the typed word was not defined")
	}
	if rig.player.count() != 3 {
		t.Errorf("played %d times, want 3", rig.player.count())
	}
}

// A bare Enter replays, and costs no second fetch.
func TestEditorLoopBareEnterReplays(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6", got)
	}
	if got := rig.cdn.Requested(); len(got) != 1 {
		t.Errorf("made %d CDN requests, want 1 — the replay refetched: %v", len(got), got)
	}
	if n := strings.Count(out.String(), "/ˌsikəˈfan(t)ik/"); n != 1 {
		t.Errorf("definition printed %d times, want 1 — replay reprinted it", n)
	}
}

// History is populated from submitted lines, so the next frame can suggest.
func TestEditorLoopFeedsHistory(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\rsyc"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	// The final frame should carry the grey remainder of the earlier word.
	if !strings.Contains(out.String(), greyOn+"ophantic"+greyOff) {
		t.Errorf("no grey suggestion in the final frame: %q", tailOf(out.String()))
	}
}

func TestEditorLoopCtrlCExitsZero(t *testing.T) {
	rig, opt, _ := editorRig(t, "sycophantic", true)
	restored := false
	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("syc\x03"), nil, rig.deps, opt, recordingConsole(&out, &errb, func() { restored = true }))

	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if !restored {
		t.Error("the terminal was not restored — the shell would be left raw")
	}
}

// Cancellation must restore the terminal too. This is the worst failure this
// tool can produce: no later output can fix a shell left in raw mode.
func TestEditorLoopCancellationRestoresTerminal(t *testing.T) {
	rig, opt, _ := editorRig(t, "sycophantic", true)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	restored := false
	var out, errb bytes.Buffer

	runEditor(ctx, make(chan Key), nil, rig.deps, opt, recordingConsole(&out, &errb, func() { restored = true }))
	if !restored {
		t.Error("cancellation exited without restoring the terminal")
	}
}

func tailOf(s string) string {
	if len(s) > 120 {
		return s[len(s)-120:]
	}
	return s
}

// A bare Enter must not advance the prompt: it replays, and the screen ends
// where it started. Operator-reported — each empty return was adding a line.
func TestEditorLoopBareEnterDoesNotAdvance(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))
	baseline := strings.Count(out.String(), "\n")

	rig2, opt2, finish2 := editorRig(t, "sycophantic", true)
	var out2 bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r\r\r\r"), nil, rig2.deps, opt2, recordingConsole(&out2, &bytes.Buffer{}, finish2))

	// Three extra replays, zero extra lines.
	if got := strings.Count(out2.String(), "\n"); got != baseline {
		t.Errorf("three replays added %d newlines, want 0", got-baseline)
	}
	if rig2.player.count() != 12 {
		t.Errorf("played %d times, want 12 (4 × 3)", rig2.player.count())
	}
}

// The loop's own writes still carry "\r\n".
//
// The original reason — a bare "\n" in raw mode moves down without returning to
// column 0 — is gone with the mode it named: the screen strips the carriage
// return and places lines itself (#30 D4/D5). What the bytes still buy is that
// this loop's output is correct on ANY writer, including the plain buffer these
// tests give it and the real stdout the fallback path uses.
func TestEditorLoopUsesCarriageReturnsInRawMode(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	s := out.String()
	// Find the newline the loop writes to commit the input line.
	i := strings.Index(s, "\n")
	if i < 1 {
		t.Fatalf("no newline written: %q", s)
	}
	if s[i-1] != '\r' {
		t.Errorf("bare \\n written in raw mode at %d — the next line would start at the current column: %q", i, s[max(0, i-20):i+1])
	}
}

// The input line must be findable in a screenful of definition text: the prompt
// carries an accent colour and the typed word is bold, so the one line you can
// act on reads differently from everything you cannot.
func TestRenderLineMakesTheInputLineDistinct(t *testing.T) {
	e := NewEditor()
	e.Line = []rune("fold")
	e.Cursor = 4

	got := RenderLine(e, "able", nil, true)
	if !strings.Contains(got, promptOn+prompt) {
		t.Error("the prompt is not accented")
	}
	if !strings.Contains(got, inputOn+"fold") {
		t.Error("typed text is not emphasised")
	}
	if !strings.Contains(got, greyOn+"able") {
		t.Error("the suggestion lost its grey")
	}
	// -no-color must still produce no ANSI beyond the frame control.
	plain := RenderLine(e, "able", nil, false)
	for _, sgr := range []string{promptOn, inputOn, greyOn} {
		if strings.Contains(plain, sgr) {
			t.Errorf("colour leaked into a no-colour render: %q", plain)
		}
	}
	if !strings.Contains(plain, "fold"+"able") {
		t.Errorf("plain render lost content: %q", plain)
	}
}

// The grey tail was never accepted, so committing it to scrollback would claim
// the user typed something they did not. Operator-reported.
func TestEditorLoopCommitsWithoutTheSuggestion(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	// Define the long word, then type a prefix of it and submit.
	runEditor(t.Context(), scriptKeys("sycophantic\rsyc\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	s := out.String()
	// The last frame written before the second submit must carry no grey.
	idx := strings.LastIndex(s, "\r\n")
	if idx < 0 {
		t.Fatal("no committed line")
	}
	head := s[:idx]
	lastFrame := head[strings.LastIndex(head, eraseLine):]
	if strings.Contains(lastFrame, greyOn) {
		t.Errorf("the committed line still showed the suggestion: %q", lastFrame)
	}
	if !strings.Contains(lastFrame, "syc") {
		t.Errorf("committed frame lost the typed text: %q", lastFrame)
	}
}

// C-2: the interactive path must mean the same thing as the line path. It
// bypassed parseREPLLine, so it trimmed nothing and did not collapse interior
// whitespace — "hot  dog" never matched the multi-word headword.
func TestEditorLoopNormalisesTheSubmittedLine(t *testing.T) {
	rig, opt, finish := editorRig(t, "hot dog", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("  hot   dog  \r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	if strings.Contains(errb.String(), "no dictionary entry") {
		t.Errorf("the line was not normalised before lookup: %q", errb.String())
	}
	if !strings.Contains(out.String(), "hot dog") {
		t.Errorf("the multi-word headword was not defined: %q", tailOf(out.String()))
	}
}

// I-2: a bare Enter with nothing defined yet is a hint, not a crash or a lookup.
func TestEditorLoopBareEnterWithNoCurrentWord(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	if !strings.Contains(errb.String(), "press return to replay") {
		t.Errorf("want the hint, got %q", errb.String())
	}
	if rig.player.count() != 0 {
		t.Errorf("played %d times with nothing defined", rig.player.count())
	}
}

// I-2: a failed lookup must not become the current word, so a following bare
// Enter replays the last GOOD word rather than retrying the typo.
func TestEditorLoopFailedLookupKeepsPreviousWord(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\rrizz\r\r"), nil, rig.deps, opt, recordingConsole(&out, &errb, finish))

	if !strings.Contains(errb.String(), "rizz") {
		t.Error("the failed lookup was not reported")
	}
	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6 — the replay did not use the last good word", got)
	}
}

// The loop through a real SCREEN, which is what production hands it (#30 D5).
//
// The rest of this file drives the loop with a plain buffer, which is the right
// instrument for what the loop DECIDES. This one pins where its output LANDS:
// the definition and the line the user committed are buffer lines — they are
// scrollback, and D3 prints them back into the normal screen on exit — while the
// `♫ playing 3×` indicator is drawn and taken back, so the transcript never
// files a claim that playback happened.
func TestEditorLoopWritesThroughAScreen(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	sc := &screen{}
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, console{view: paintInto(io.Discard), finish: finish, stdout: sc, stderr: sc})

	got := sc.Transcript()
	if !strings.Contains(got, "/ˌsikəˈfan(t)ik/") {
		t.Errorf("the definition never reached the buffer: %q", got)
	}
	// The committed line, colours and all, is the FIRST thing in the buffer: the
	// word the user typed stays above the entry it produced.
	first := sc.Lines()[0]
	if !strings.Contains(first, prompt) || !strings.Contains(first, "sycophantic") {
		t.Errorf("the committed line is not the head of the transcript: %q", first)
	}
	if strings.Contains(got, "♫") {
		t.Errorf("the ephemeral indicator survived into the record: %q", got)
	}
	// The prompt is the LIVE EDGE and goes to paint, so a session that typed
	// eleven characters must not hold eleven copies of the prompt.
	if n := strings.Count(got, prompt); n != 1 {
		t.Errorf("the buffer holds %d prompts, want 1 — the live edge was buffered", n)
	}
	if strings.Contains(got, "\x1b[K") {
		t.Errorf("a terminal control sequence was buffered as text: %q", got)
	}
}

// A committed line carries no CURSOR, and the buffer is where that shows.
//
// RenderLine parks the cursor by emitting `ESC[<n>D`, which is right for a line
// being typed and wrong for one being filed: submitted mid-line, that escape
// landed in the buffer and from there in the exit transcript — a control
// sequence stored as text. The earlier version of this test guarded only against
// `ESC[K`, so the fix was deletable with the suite still green.
func TestACommittedLineCarriesNoCursorEscape(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	sc := &screen{}
	// Type the word, walk the cursor back into it, THEN submit.
	ks := keySeq(append(runes("sycophantic"),
		Key{Kind: KeyLeft}, Key{Kind: KeyLeft}, Key{Kind: KeyLeft}, Key{Kind: KeyEnter})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: paintInto(io.Discard), finish: finish, stdout: sc, stderr: sc})

	first := sc.Lines()[0]
	if !strings.Contains(first, "sycophantic") {
		t.Fatalf("the committed line is not the head of the transcript: %q", first)
	}
	// Any CSI that is not a colour: the cut is at the final byte, and only "m"
	// belongs in a record of what was shown.
	for _, seq := range []string{"\x1b[3D", "\x1b[K", "\x1b[A"} {
		if strings.Contains(first, seq) {
			t.Errorf("the committed line carries %q, a control sequence stored as text: %q", seq, first)
		}
	}
}

// writerFunc adapts a function to io.Writer, so a test can observe WHEN a write
// happens rather than only what it said.
type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// The prompt belongs to a loop that is WAITING for a keystroke.
//
// Every write repaints the frame around the live edge last recorded, so a prompt
// left standing through a lookup is redrawn under the definition it produced:
// `arrondissement` appeared twice, once as the entry's headword and once as a
// prompt still holding the line just submitted, while the recording played
// (operator-reported). What is pinned here is the general property rather than
// that instance — nothing is written while a prompt is on the frame, because a
// prompt drawn when nothing is reading keys invites typing at a line that does
// not exist.
func TestNothingIsWrittenWhileAPromptIsShown(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	view := paintInto(io.Discard)
	var whenWritten []string
	w := writerFunc(func(p []byte) (int, error) {
		whenWritten = append(whenWritten, view.livePrompt())
		return len(p), nil
	})
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, console{view: view, finish: finish, stdout: w, stderr: w})

	if len(whenWritten) == 0 {
		t.Fatal("the session wrote nothing at all, so nothing is asserted")
	}
	for i, p := range whenWritten {
		if p != "" {
			t.Fatalf("write %d of %d landed with a prompt on the frame: %q",
				i+1, len(whenWritten), p)
		}
	}
	// And it comes BACK: blanking the live edge for the whole session would
	// satisfy the loop above and leave a session with no prompt at all.
	if view.livePrompt() == "" {
		t.Error("the prompt never returned after the work finished")
	}
}

// The viewport can be MOVED by a user (#30 M1.4a).
//
// Without this M1 ships a scroll model nothing exercises and nobody can reach:
// the alternate screen has no scrollback, so a definition taller than the
// terminal has its head off-screen and the terminal's own scrollback cannot get
// it back. The wheel is M2's; these keys are what make M1 usable on its own.
func TestEditorPageKeysScroll(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(Key{Kind: KeyPageUp}, Key{Kind: KeyPageUp}, Key{Kind: KeyPageDown})
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	pages, _, _ := view.scrolls()
	if want := []int{1, 1, -1}; !slices.Equal(pages, want) {
		t.Errorf("the viewport moved %v, want %v — positive is backward, toward older text", pages, want)
	}
}

// A viewport key is not an EDITOR key: it changes what you are looking at, not
// the line you are typing.
func TestPageKeysDoNotTouchTheLine(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(append(runes("syc"), Key{Kind: KeyPageUp}, Key{Kind: KeyPageDown})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	// The last frame drawn still holds what was typed — a page key that reached
	// Apply would have redrawn something else, or nothing.
	if last := view.lastPrompt(); !strings.Contains(last, "syc") {
		t.Errorf("a page key disturbed the line being typed: %q", last)
	}
}

// Row 4b of M1's done-when, and the reason the plan says PageUp/PageDown ONLY.
//
// Ctrl-U and Ctrl-D are the obvious half-page bindings, and an earlier draft
// proposed exactly that. Both are already bound — 0x15 kills the line, 0x04 ends
// the session on an empty one — so taking either for scrolling would be a silent
// regression in an editor people already use, and a suite that tested only the
// page keys would ship it green.
func TestCtrlDStillEndsTheSession(t *testing.T) {
	rig, opt, _ := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	ended := false
	// A key BEHIND the Ctrl-D, and the channel closed so a loop that ignored it
	// still terminates. Ending the session is not observable through `ended`
	// alone — the closed channel calls finish() too — so what discriminates is
	// whether the key behind it was ever read.
	keys := make(chan Key, 2)
	keys <- Key{Kind: KeyEOF}
	keys <- Key{Kind: KeyRune, Rune: 'x'}
	close(keys)
	code := runEditor(t.Context(), keys, nil, rig.deps, opt, recordingConsole(&out, &errb, func() { ended = true }))

	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if !ended {
		t.Error("the terminal was not restored")
	}
	if len(keys) != 1 {
		t.Error("Ctrl-D did not end the session — the loop read on past it, so it was rebound")
	}
}

func TestCtrlUStillKillsTheLine(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(append(runes("syco"), Key{Kind: KeyKillLine})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	if last := view.lastPrompt(); strings.Contains(last, "syco") {
		t.Errorf("Ctrl-U did not clear the line — it was rebound: %q", last)
	}
}

// The wheel scrolls the buffer. It must NOT walk history (operator-reported).
//
// In the alternate screen a terminal sends the wheel as ARROW KEYS unless asked
// to report the mouse, and Up/Down here are the history walk — so scrolling
// recalled words. Nothing could tell the two apart, because they are the same
// bytes; #30 M1.4b turns mouse reporting on so the gesture arrives as itself.
func TestWheelScrollsRatherThanWalkingHistory(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	// A lookup FIRST, so history has something to walk. With an empty history a
	// wheel that reached the walk would change nothing, and this test would pass
	// for the wrong reason.
	ks := keySeq(append(runes("sycophantic"),
		Key{Kind: KeyEnter}, Key{Kind: KeyWheelUp}, Key{Kind: KeyWheelDown})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	_, lines, _ := view.scrolls()
	if want := []int{wheelLines, -wheelLines}; !slices.Equal(lines, want) {
		t.Errorf("the wheel moved the viewport %v, want %v lines", lines, want)
	}
	if last := view.lastPrompt(); strings.Contains(last, "sycophantic") {
		t.Errorf("the wheel recalled a word from history into the line: %q", last)
	}
}

// A resize repaints, and it changes BOTH halves of the shape (#30 M1.4).
//
// There was no resize handling before the screen: the width was read once at
// flag parse. A full-screen program cannot get away with that — it draws a frame
// for a height, and a frame one row too tall makes the terminal scroll, which
// moves every row the app believes it placed.
func TestEditorResizeRedrawsForTheNewShape(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	opt.width = 80
	var out, errb bytes.Buffer
	view := paintInto(&out)
	resizes := make(chan winSize, 1)
	keys := make(chan Key, 4)

	done := make(chan int, 1)
	go func() {
		done <- runEditor(t.Context(), keys, nil, rig.deps, opt, console{view: view, resizes: resizes, finish: finish, stdout: &out, stderr: &errb})
	}()
	resizes <- winSize{rows: 10, cols: 40}
	// A key AFTER the resize, so the assertion runs on a loop that has certainly
	// processed it — the alternative is sleeping and hoping.
	keys <- Key{Kind: KeyRune, Rune: '/'}
	waitFor(t, func() bool { _, _, rows := view.scrolls(); return len(rows) > 0 })
	close(keys)
	<-done

	_, _, rows := view.scrolls()
	if !slices.Equal(rows, []int{10}) {
		t.Errorf("the screen was told %v rows, want [10]", rows)
	}
	// The WIDTH half, observed where it is actually used: the command menu is
	// truncated to it, so a 40-column terminal must not be handed 80-column rows.
	menus := view.drawnMenus()
	for _, m := range menus[len(menus)-1] {
		if len([]rune(m)) > 40 {
			t.Errorf("a menu row is %d columns wide in a 40-column terminal: %q", len([]rune(m)), m)
		}
	}
}

// Dragging a window corner fires dozens of SIGWINCHes, and a queue of stale
// shapes is a queue of wrong frames — only the newest is true. Coalescing is
// also what keeps this goroutine from blocking on a loop that is busy playing a
// recording for five seconds.
func TestWatchResizeCoalesces(t *testing.T) {
	sigs := make(chan os.Signal, 8)
	shapes := []winSize{{rows: 10, cols: 40}, {rows: 20, cols: 80}, {rows: 30, cols: 120}}
	// The measurements are reported over a CHANNEL rather than counted in a
	// variable the watcher's goroutine writes and this one reads: an
	// unsynchronised counter is a data race, and `waitFor` spinning on one can
	// spin to its own timeout.
	measured := make(chan int, len(shapes))
	n := 0
	out := watchResize(t.Context(), func(...os.Signal) <-chan os.Signal { return sigs },
		func() winSize {
			sz := shapes[n]
			n++
			measured <- n
			return sz
		})
	for range shapes {
		sigs <- syscall.SIGWINCH
	}
	for range shapes {
		select {
		case <-measured:
		case <-time.After(5 * time.Second):
			t.Fatal("the watcher stopped measuring")
		}
	}

	// One shape waiting, and it is the LAST one.
	if got := <-out; got != shapes[len(shapes)-1] {
		t.Errorf("the loop would have redrawn for %v, want the newest shape %v", got, shapes[len(shapes)-1])
	}
	select {
	case extra := <-out:
		t.Errorf("a stale shape was left queued: %v", extra)
	default:
	}
}

// No signal transport, no crash: a channel that never fires leaves the shape as
// it was measured at startup, which is what a test harness and a terminal-less
// caller both get.
func TestWatchResizeWithoutASignalTransport(t *testing.T) {
	out := watchResize(t.Context(), nil, func() winSize { t.Fatal("measured with no transport"); return winSize{} })
	select {
	case sz := <-out:
		t.Errorf("a shape arrived from nowhere: %v", sz)
	default:
	}
}

// Clicking the headword plays it (#30 M2.4, Done-when 1).
//
// The click goes through replayInPlace — the same path a bare Enter takes — so
// it cannot drift from the gesture it is a shortcut for. Asserted on the CDN
// request, which is the observable a user gets: what was actually asked for.
func TestClickOnHeadwordReplays(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	view.offer(3, 5, Region{Kind: RegionHeadword, Text: "sycophantic", Word: "sycophantic"})

	// Look the word up, then click its headword.
	ks := keySeq(append(runes("sycophantic"), Key{Kind: KeyEnter}, Key{Kind: KeyClick, Row: 3, Col: 5})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	// Three for the lookup, three for the click, and NO second fetch: the click
	// replays rather than looking the word up again.
	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6 — the lookup and the click", got)
	}
	if got := rig.cdn.Requested(); len(got) != 1 {
		t.Errorf("made %d CDN requests, want 1 — the click refetched: %v", len(got), got)
	}
}

// Clicking `ORIGIN French` plays it in French (#30 M2.4, Done-when 2).
//
// This is the click the issue was filed for. It reaches #29's mechanism through
// the same replay a `/pron fr` takes — one parameter apart — rather than a
// second path that could disagree about what a source language means.
func TestClickOnOriginLanguagePlaysIt(t *testing.T) {
	// `concrete` is the corpus's French-origin entry: "from French concret or
	// Latin concretus" — a stage before the language, which is also the shape
	// the offset mask has to survive.
	en := voice{Lang: "en", Locale: "us"}
	fr := voice{Lang: "fr", Locale: "fr"}
	english := AudioCandidates("concrete", en)[0]
	french := AudioCandidates("concrete", fr)[0]

	rig := newAudioRigServing(t, english, french)
	rig.deps.stdinIsTerminal = func() bool { return true }
	opt := options{times: 1, tty: true, color: true}

	var out, errb bytes.Buffer
	view := paintInto(&out)
	view.offer(9, 4, Region{Kind: RegionOriginLang, Text: "French", Word: "concrete", Lang: "fr"})

	ks := keySeq(append(runes("concrete"), Key{Kind: KeyEnter}, Key{Kind: KeyClick, Row: 9, Col: 4})...)
	code := runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: func() {}, stdout: &out, stderr: &errb})
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}

	// The lookup asked in the session's language; the click asked in French.
	want := []string{stripHost(t, english, audioBase), stripHost(t, french, audioBase)}
	if got := rig.cdn.Requested(); !slices.Equal(got, want) {
		t.Errorf("the CDN was asked for:\n  %q\nwant:\n  %q", got, want)
	}
}

// A click on ordinary text is NOTHING — not a beep, not a message.
//
// Pointing at a word that offers nothing is not an error, and a program that
// answered one would make the whole screen feel like a minefield.
func TestClickOnNothingIsNothing(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	// No regions offered anywhere.

	ks := keySeq(append(runes("sycophantic"), Key{Kind: KeyEnter}, Key{Kind: KeyClick, Row: 4, Col: 2})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	if got := rig.player.count(); got != 3 {
		t.Errorf("played %d times, want 3 — the click on empty text acted", got)
	}
	if s := errb.String(); strings.Contains(s, "define:") {
		t.Errorf("a click on ordinary text produced a message: %q", s)
	}
}

// A click never reaches the editor: it is a gesture on the SCREEN, so the line
// being typed is untouched — the same rule the viewport keys follow.
func TestClickDoesNotTouchTheLine(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	view.offer(1, 1, Region{Kind: RegionHeadword, Text: "sycophantic", Word: "sycophantic"})

	ks := keySeq(append(runes("syc"), Key{Kind: KeyClick, Row: 1, Col: 1})...)
	runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

	if last := view.lastPrompt(); !strings.Contains(last, "syc") {
		t.Errorf("a click disturbed the line being typed: %q", last)
	}
}

// An ENTRY carries its click map to the screen, which is the wiring between
// Render's regions and the hit test — and the half that a scripted display
// double cannot check for itself.
func TestALookupHandsItsRegionsToTheScreen(t *testing.T) {
	rig, opt, finish := editorRig(t, "concrete", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	runEditor(t.Context(), scriptKeys("concrete\r"), nil, rig.deps, opt,
		console{view: view, finish: finish, stdout: view, stderr: &errb})

	got := view.collected()
	if len(got) == 0 {
		t.Fatal("the entry reached the screen with no click map at all")
	}
	var kinds []RegionKind
	for _, r := range got {
		kinds = append(kinds, r.Kind)
		if r.Word != "concrete" {
			t.Errorf("region %q belongs to %q, want the entry's headword", r.Text, r.Word)
		}
	}
	if !slices.Contains(kinds, RegionHeadword) {
		t.Errorf("no headword region: the primary target is missing (%v)", got)
	}
	if !slices.Contains(kinds, RegionOriginLang) {
		t.Errorf("no ORIGIN language region for `concrete`, whose etymology names French (%v)", got)
	}
}

// Regions are ONE REGISTRY, not two special cases (#30 Done-when 7).
//
// The issue is filed as "the affordance is one mechanism, so a third consumer is
// a row rather than a new feature". A Kind with no action is that promise
// quietly breaking: the region draws, invites a click, and does nothing.
func TestEveryRegionKindIsActionable(t *testing.T) {
	// DERIVED from the registry's own extent, not from a kind named by hand. The
	// predecessor looped `kind <= RegionOriginLang`, which is a second copy of
	// "these are all the kinds" — so a third kind was never exercised and this
	// guard could not fire for the case it exists to catch.
	for kind := RegionKind(0); kind < numRegionKinds; kind++ {
		// A fresh rig per kind, so each count starts from zero rather than from
		// whatever the previous kind left behind.
		rig, opt, finish := editorRig(t, "sycophantic", true)
		var out, errb bytes.Buffer
		view := paintInto(&out)
		view.offer(2, 0, Region{Kind: kind, Text: "sycophantic", Word: "sycophantic", Lang: "fr"})

		ks := keySeq(append(runes("sycophantic"), Key{Kind: KeyEnter}, Key{Kind: KeyClick, Row: 2, Col: 0})...)
		runEditor(t.Context(), ks, nil, rig.deps, opt, console{view: view, finish: finish, stdout: &out, stderr: &errb})

		// Every kind must DO something: the lookup plays 3, so a kind that acted
		// plays more. A kind added with no case in `clicked` reddens here.
		if got := rig.player.count(); got <= 3 {
			t.Errorf("RegionKind %d played nothing when clicked — it draws, invites a click, and does nothing", kind)
		}
	}
}

// ...and the SAME registry answers for a sitting, driven directly (#38 D4).
//
// The loop above proves every kind acts in the EDITOR. `#30` Done-when 7 asks
// for more than that — "one mechanism, so a third consumer is a row rather than
// a new feature" — and the failure it forbids is a kind that acts in one loop
// and not the other. Driving `playRegion` itself is what makes that
// unexpressible: there is only one switch to be missing a case from.
func TestEveryRegionKindIsActionableThroughTheSharedRegistry(t *testing.T) {
	for kind := RegionKind(0); kind < numRegionKinds; kind++ {
		rig, opt, _ := editorRig(t, "sycophantic", true)
		opt.times = 1
		var out, errb bytes.Buffer

		playRegion(t.Context(), rig.deps, opt,
			Region{Kind: kind, Text: "sycophantic", Word: "sycophantic", Lang: "fr"},
			"", defaultIndicator(opt), &out, &errb)

		if rig.player.count() == 0 {
			t.Errorf("RegionKind %d played nothing through playRegion — a sitting draws the "+
				"underline and a click on it does nothing", kind)
		}
	}
}

// The registry REFUSES what it does not know, rather than playing the headword
// as a fallback.
//
// A kind with no row is a bug in the registry, and silently doing something
// plausible is how it would ship: the underline would work, and only the wrong
// recording would say otherwise.
func TestAnUnknownRegionKindPlaysNothing(t *testing.T) {
	rig, opt, _ := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer

	playRegion(t.Context(), rig.deps, opt,
		Region{Kind: numRegionKinds + 7, Text: "sycophantic", Word: "sycophantic"},
		"", defaultIndicator(opt), &out, &errb)

	if got := rig.player.count(); got != 0 {
		t.Errorf("an unknown kind played %d times — the registry guessed instead of refusing", got)
	}
}

// THE WHOLE PATH, on real objects: a terminal's click coordinates reach the word
// they point at (#30 M2, BR-47).
//
// Every other test in this file scripts the display's answer, so the row and
// column are the test's own invention and nothing checks that `Key.Row` is the
// row `screen.LineAt` indexes. The layers were each pinned and the JOINT between
// them was not — the eighth time this issue has produced that finding, and the
// rule it leaves is that the enumeration must be of JOINTS: every place two
// separately-pinned layers exchange a value across a coordinate or unit boundary
// earns a row, and a row is earned only when a test drives both real objects.
//
// So this drives runEditor with a real liveScreen as BOTH view and stdout, reads
// the underlined span out of the painted frame the way a user's eye would, and
// clicks the cell the terminal would report — no coordinate chosen by the test.
func TestAClickAtAPaintedCellPlaysWhatIsUnderIt(t *testing.T) {
	en := voice{Lang: "en", Locale: "us"}
	fr := voice{Lang: "fr", Locale: "fr"}
	english := AudioCandidates("concrete", en)[0]
	french := AudioCandidates("concrete", fr)[0]

	rig := newAudioRigServing(t, english, french)
	rig.deps.stdinIsTerminal = func() bool { return true }
	opt := options{times: 1, tty: true, color: true, width: 76}

	// syncBuf, not bytes.Buffer: the loop paints from its own goroutine while
	// this one reads the frame to find the cell to click. `go test -race` reports
	// the unguarded version, and a test that races is a test that can fail for a
	// reason it is not about.
	var tty syncBuf
	live := newLiveScreen(&tty, 24, 76)
	live.interval = -1 // paint every write, so the frame under test is the real one

	keys := make(chan Key, 32)
	for _, r := range "concrete" {
		keys <- Key{Kind: KeyRune, Rune: r}
	}
	keys <- Key{Kind: KeyEnter}

	done := make(chan int, 1)
	var errb bytes.Buffer
	go func() {
		done <- runEditor(t.Context(), keys, nil, rig.deps, opt,
			console{view: live, finish: func() {}, stdout: live, stderr: &errb})
	}()

	// The cell to click is read from the frame the screen is CURRENTLY showing,
	// atomically — not from the terminal stream. Reading the stream is what a
	// user's eye does, but a stream is a history: the last frame in it while the
	// loop is still writing sits a line behind the state a click resolves
	// against, and this test read row 12 for a span the screen answered at row
	// 11 about one run in five. Frame counts do not fix it either, since a frame
	// from a write still in flight satisfies them.
	//
	// The coordinate is still the SCREEN's and not the test's: it is where the
	// painted frame shows the word. That the frame carries the underline is
	// pinned separately, in screen_test.go.
	// IDLE FIRST. Waiting for the underline to appear is not enough: the loop is
	// still writing after it — the playback indicator, the blank line, the
	// redraw — and each write shifts the buffer, so a frame read mid-sequence is
	// a line behind the state the click will resolve against. That was one run
	// in five, whether the frame came from the stream or from the screen.
	//
	// A marker keystroke is the deterministic signal. Keys are consumed in
	// order, so when its echo reaches the live edge, everything the Enter set in
	// motion has finished and the loop is blocked waiting for the next key.
	keys <- Key{Kind: KeyRune, Rune: 'z'}
	waitFor(t, func() bool { return strings.Contains(livePromptOf(live), "z") })

	row, col, ok := frameCell(live, "French")
	if !ok {
		t.Fatalf("the frame does not show French:\n%s", tty.String())
	}

	// The BYTES a terminal sends for a click on that cell, decoded the way the
	// key reader decodes them — so the 1-based wire convention is part of what
	// this test crosses rather than something it assumes. Constructing the Key
	// directly left that conversion to a different test, which is the seam this
	// row exists to close.
	report := fmt.Sprintf("\x1b[<0;%d;%dM", col+1, row+1)
	click, n := decodeKey([]byte(report))
	if n != len(report) || click.Kind != KeyClick {
		t.Fatalf("decodeKey(%q) = kind %v consumed %d; the report is not a click", report, click.Kind, n)
	}
	keys <- click
	waitFor(t, func() bool { return len(rig.cdn.Requested()) >= 2 })
	close(keys)
	<-done

	want := []string{stripHost(t, english, audioBase), stripHost(t, french, audioBase)}
	if got := rig.cdn.Requested(); !slices.Equal(got, want) {
		t.Errorf("clicking the cell at row %d col %d asked the CDN for:\n  %q\nwant:\n  %q",
			row, col, got, want)
	}
}

// livePromptOf reads the prompt the screen is currently showing, under its lock.
func livePromptOf(l *liveScreen) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.prompt
}

// frameCell is where the screen is showing text right now: its viewport row and
// display column, read under the screen's own lock so the answer cannot be a
// frame the loop has already moved past.
func frameCell(l *liveScreen, text string) (row, col int, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	frame, _ := l.s.visible()
	for r, line := range frame {
		if i := strings.Index(stripEscapes(line), text); i >= 0 {
			return r, visibleCells(stripEscapes(line)[:i]), true
		}
	}
	return 0, 0, false
}
