package main

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// Write is the seam every caller feeds (#30 D5), so it has to behave like a
// stream rather than like a frame source: deltas arrive chunked and a reply
// split as "one" then " two\n" is ONE line, not two.
func TestScreenWriteBuildsLines(t *testing.T) {
	for _, tc := range []struct {
		name   string
		writes []string
		want   []string
	}{
		{"one whole line", []string{"hello\n"}, []string{"hello"}},
		{"a partial line is not a line yet", []string{"hel"}, []string{"hel"}},
		{"a partial line CONTINUES", []string{"hel", "lo\n"}, []string{"hello"}},
		{"two lines in one write", []string{"a\nb\n"}, []string{"a", "b"}},
		{"a blank line is a line", []string{"a\n\nb\n"}, []string{"a", "", "b"}},
		{"no trailing newline still shows", []string{"a\nb"}, []string{"a", "b"}},
		// The ask path streams token by token; this is that shape.
		{"token stream", []string{"The", " quick", " brown\n", "fox"}, []string{"The quick brown", "fox"}},
		// The split-CRLF case the screen inherited: a reply
		// split as "a\r" then "\nb" must not become two lines plus a stray CR.
		{"a CRLF split across writes", []string{"a\r", "\nb\n"}, []string{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s screen
			for _, w := range tc.writes {
				if _, err := s.Write([]byte(w)); err != nil {
					t.Fatalf("Write: %v", err)
				}
			}
			if got := s.Lines(); !slices.Equal(got, tc.want) {
				t.Errorf("Lines() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Frame is PURE — (lines, offset, rows) in, the rows to paint out — which is
// what keeps the scroll arithmetic out of a pty test.
func TestScreenFrame(t *testing.T) {
	lines := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for _, tc := range []struct {
		name   string
		offset int
		rows   int
		want   []string
	}{
		{"the tail is what you see by default", 0, 3, []string{"7", "8", "9"}},
		{"scrolled up by two", 2, 3, []string{"5", "6", "7"}},
		{"a viewport taller than the buffer shows all of it", 0, 20, lines},
		// Both clamps. An offset past the top must not index backwards, and a
		// negative one must not either — the wheel and PageUp both produce them.
		{"clamped at the top", 999, 3, []string{"0", "1", "2"}},
		{"clamped at the bottom", -999, 3, []string{"7", "8", "9"}},
		{"a zero-row viewport paints nothing", 0, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s screen
			for _, l := range lines {
				s.Write([]byte(l + "\n"))
			}
			s.rows = tc.rows
			s.offset = tc.offset
			if got := s.Frame(); !slices.Equal(got, tc.want) {
				t.Errorf("Frame(offset=%d, rows=%d) = %q, want %q", tc.offset, tc.rows, got, tc.want)
			}
		})
	}
}

// Scroll clamps rather than letting the offset run away, because a wheel event
// arrives per notch and a held PageUp repeats.
func TestScreenScrollClamps(t *testing.T) {
	var s screen
	for i := 0; i < 5; i++ {
		s.Write([]byte("x\n"))
	}
	s.rows = 3
	s.Scroll(100)
	if s.offset != 2 { // 5 lines, 3 rows: the furthest back is 2
		t.Errorf("after scrolling up hard, offset = %d, want 2", s.offset)
	}
	s.Scroll(-100)
	if s.offset != 0 {
		t.Errorf("after scrolling down hard, offset = %d, want 0", s.offset)
	}
}

// It must survive whatever a dictionary entry or a model reply is.
func FuzzScreenWriteDoesNotPanic(f *testing.F) {
	f.Add("plain\n")
	f.Add("\xff\xfe")
	f.Add(strings.Repeat("a\n", 5000))
	f.Fuzz(func(t *testing.T, in string) {
		var s screen
		s.rows = 24
		s.Write([]byte(in))
		_ = s.Frame()
		s.Scroll(1)
		s.Scroll(-1)
	})
}

// Paint is a WHOLE frame: home, clear, buffer tail, prompt, menu.
//
// The model it replaces counted the menu rows it drew so it could erase exactly
// that many, and carried a documented known limit for when the count was wrong.
// A whole-frame redraw cannot be off by a row, and these cases pin the split of
// a fixed terminal height between buffer, prompt and menu.
func TestScreenPaintSplitsTheHeight(t *testing.T) {
	var s screen
	for _, l := range []string{"a", "b", "c", "d", "e"} {
		s.Write([]byte(l + "\n"))
	}
	for _, tc := range []struct {
		name      string
		termRows  int
		menu      []string
		wantLines []string
	}{
		{"no menu: rows-1 of buffer", 4, nil, []string{"c", "d", "e"}},
		{"a menu takes from the buffer, not the prompt", 4, []string{"m1"}, []string{"d", "e"}},
		{"a tall menu can leave no buffer at all", 3, []string{"m1", "m2"}, nil},
		// Losing the line you are typing is worse than losing history you can
		// scroll to, so the prompt survives a terminal too short for anything.
		{"an impossibly short terminal still shows the prompt", 1, []string{"m1", "m2"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			s.Paint(&b, tc.termRows, 80, "PROMPT", tc.menu)
			out := b.String()
			for _, want := range tc.wantLines {
				if !strings.Contains(out, want+"\r\n") {
					t.Errorf("frame is missing buffer line %q:\n%q", want, out)
				}
			}
			if !strings.Contains(out, "PROMPT") {
				t.Errorf("frame has no prompt:\n%q", out)
			}
			if !strings.HasPrefix(out, cursorHome+eraseDown) {
				t.Errorf("frame does not start by clearing the screen: %q", out[:min(20, len(out))])
			}
		})
	}
}

// The transcript is what survives the alternate screen being discarded.
func TestScreenTranscript(t *testing.T) {
	var s screen
	if got := s.Transcript(); got != "" {
		t.Errorf("an empty session has a transcript: %q", got)
	}
	s.Write([]byte("one\ntwo\n"))
	if got, want := s.Transcript(), "one\ntwo\n"; got != want {
		t.Errorf("Transcript() = %q, want %q", got, want)
	}
}

// eraseLine is the ephemeral indicator's own gesture — "take that line back" —
// and the buffer honours it rather than stripping it (#30 M1.3).
//
// Without this the `♫ playing 3×` that a terminal shows and then removes would
// stay in the buffer, and the exit transcript (D3) would file a claim that
// playback happened. That is the "ephemeral UI vs record" doctrine failing in
// the one direction it exists to prevent.
func TestScreenTakesBackAnErasedLine(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write []string
		want  []string
	}{
		{
			"the indicator is drawn and taken back",
			[]string{"entry\r\n", "  ♫ playing 3×", eraseLine},
			[]string{"entry"},
		},
		{
			"a note replaces the line it is written over",
			[]string{"  ♫ playing 3×", eraseLine + "define: nothing to replay\r\n"},
			[]string{"define: nothing to replay"},
		},
		{
			// The gesture addresses the row the cursor sits on, and after a
			// newline that row has nothing written to it yet. A committed line
			// is scrollback and must survive.
			"an erase with no open line leaves scrollback alone",
			[]string{"entry\r\n", eraseLine},
			[]string{"entry"},
		},
		{
			"text after an erase starts a fresh line",
			[]string{"half", eraseLine, "whole\r\n"},
			[]string{"whole"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s screen
			for _, w := range tc.write {
				s.Write([]byte(w))
			}
			if got := s.Lines(); !slices.Equal(got, tc.want) {
				t.Errorf("lines = %q, want %q", got, tc.want)
			}
		})
	}
}

// liveScreen is an io.Writer that SHOWS what is written to it, which is what
// stdout was before the alternate screen. A streamed answer arrives token by
// token and a `♫ playing 3×` has to appear while playback blocks for seconds:
// both reach the terminal by being written, so a write has to repaint.
func TestLiveScreenShowsWhatIsWrittenToIt(t *testing.T) {
	var tty bytes.Buffer
	l := newLiveScreen(&tty, 10, 80)

	l.Draw("› ", nil)
	tty.Reset()
	// The throttle is out of the way: a zero window means every write paints, so
	// what is under test here is that a write SHOWS, not how soon.
	l.interval = -1
	l.Write([]byte("a definition\n"))
	if !strings.Contains(tty.String(), "a definition") {
		t.Errorf("a write did not reach the terminal: %q", tty.String())
	}
	if !strings.Contains(tty.String(), "› ") {
		t.Errorf("the repaint dropped the live edge: %q", tty.String())
	}

	// After the terminal is handed back, painting must stop dead: a frame drawn
	// then lands on the NORMAL screen, over whatever was there before define
	// ran. The buffer keeps accepting writes, because the exit transcript is
	// read from it.
	l.Stop()
	tty.Reset()
	l.Write([]byte("after the end\n"))
	if tty.Len() != 0 {
		t.Errorf("painted after Stop: %q", tty.String())
	}
	if !strings.Contains(l.Transcript(), "after the end") {
		t.Errorf("Stop dropped the write instead of just the paint: %q", l.Transcript())
	}
}

// A page is a SCREENFUL, minus one line of overlap so the eye has an anchor
// across the jump — and the screen computes it, because the loop knows a key was
// pressed and not how tall the viewport is (#30 M1.4a).
func TestScreenPageIsAScreenful(t *testing.T) {
	var s screen
	for i := 0; i < 30; i++ {
		s.Write([]byte("line\n"))
	}
	s.rows = 10

	s.Page(1)
	if s.offset != 9 {
		t.Errorf("one page back = %d lines, want 9 — a screenful with one line of overlap", s.offset)
	}
	s.Page(1)
	if s.offset != 18 {
		t.Errorf("two pages back = %d, want 18", s.offset)
	}
	s.Page(-1)
	if s.offset != 9 {
		t.Errorf("a page forward left the offset at %d, want 9", s.offset)
	}
	// Held keys overshoot routinely, and Scroll clamps both ends.
	s.Page(99)
	if s.offset != len(s.lines)-s.rows {
		t.Errorf("paging past the top left the offset at %d, want %d", s.offset, len(s.lines)-s.rows)
	}
	s.Page(-99)
	if s.offset != 0 {
		t.Errorf("paging past the bottom left the offset at %d, want 0", s.offset)
	}
}

// New output SNAPS the viewport back to the tail.
//
// Everything this program writes answers something the user just typed, so the
// thing they asked for must be the thing they see. Holding the offset would not
// even hold the view still: it is measured from the tail, so the text under the
// reader's eye slides up by a line for every line written.
func TestScreenWriteReturnsToTheTail(t *testing.T) {
	var s screen
	for i := 0; i < 30; i++ {
		s.Write([]byte("old\n"))
	}
	s.rows = 5
	s.Page(2)
	if s.offset == 0 {
		t.Fatal("the viewport did not move, so its return proves nothing")
	}
	s.Write([]byte("the answer\n"))
	if s.offset != 0 {
		t.Errorf("offset = %d after a write, want 0 — the answer was written off-screen", s.offset)
	}
	if got := s.Frame(); got[len(got)-1] != "the answer" {
		t.Errorf("the last visible line is %q, want the line just written", got[len(got)-1])
	}
}

// A frame is a PLACEMENT, not a set of substrings — so these tests interpret the
// bytes Paint emits the way a terminal would, and assert two properties over
// shapes rather than checking that named lines appear somewhere.
//
//  1. the frame never needs more display rows than the terminal has
//  2. it leaves the cursor where the user is typing
//
// Both were breakable while the suite was green: a line wider than the terminal
// wrapped and scrolled the screen away, a menu row taller than the space left
// did the same, the cursor walked back by menu ENTRIES rather than rows, and the
// whole cursor-up-and-reprint block could be deleted without a single failure.
// A frame that scrolls moves every row the app believes it placed, which is the
// one property the alternate screen was taken for and M2's hit test needs.
type frameGeometry struct {
	rows      int // display rows the frame occupies
	cursorRow int
	cursorCol int
}

// readFrame interprets a painted frame as a terminal would: home, erase, text
// with wrapping, and the cursor moves Paint emits.
func readFrame(t *testing.T, frame string, cols int) frameGeometry {
	t.Helper()
	row, col, maxRow := 0, 0, 0
	pending := false // the deferred wrap: the cursor sits in the last column
	rest := frame
	for len(rest) > 0 {
		if strings.HasPrefix(rest, "\x1b[") {
			end := strings.IndexFunc(rest[2:], func(r rune) bool { return r >= 0x40 && r <= 0x7e })
			if end < 0 {
				t.Fatalf("unterminated escape in frame: %q", rest)
			}
			seq, final := rest[2:2+end], rest[2+end]
			rest = rest[3+end:]
			n := 1
			if seq != "" {
				if v, err := strconv.Atoi(seq); err == nil {
					n = v
				}
			}
			switch final {
			case 'H':
				row, col, pending = 0, 0, false
			case 'A':
				row -= n
			case 'B':
				row += n
			case 'D':
				col -= n
			case 'C':
				col += n
			}
			if row < 0 {
				t.Fatalf("the frame moved the cursor above the screen: %q", frame)
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(rest)
		rest = rest[size:]
		switch r {
		case '\r':
			col, pending = 0, false
		case '\n':
			row++
			pending = false
		default:
			// DEFERRED WRAP, which is what a VT100-family terminal actually
			// does: filling the last column leaves the cursor there and the wrap
			// happens only if another character arrives. Modelling it as an
			// immediate wrap would count a line clipped to exactly the width as
			// two rows, and would have this test demand a wasted column.
			if pending {
				row++
				col, pending = 0, false
			}
			w := cellWidth(r)
			if cols > 0 && col+w > cols {
				row++
				col = 0
			}
			col += w
			if cols > 0 && col == cols {
				pending = true
			}
		}
		if row > maxRow {
			maxRow = row
		}
	}
	return frameGeometry{rows: maxRow + 1, cursorRow: row, cursorCol: col}
}

// DONE-WHEN 11: on a PINNED screen the footer sits at the terminal's bottom
// edge, whatever the content is (#41 D3a).
//
// "Pinned to the bottom" is not free, and that was the finding the plan's third
// round turned up: Paint writes the visible frame, then the prompt, then the
// footer, so a one-line question on a 24-row terminal put the bar on row three
// with twenty blank rows beneath it.
func TestAShortQuestionStillPinsTheBar(t *testing.T) {
	const termRows, termCols = 24, 80
	sc := screen{pinned: true}
	sc.Write([]byte("sycophantic\n"))

	var b strings.Builder
	sc.Paint(&b, termRows, termCols, "y = got it, n = missed it", []string{"0 of 18 · ~2 reviews/day"})

	if got := readFrame(t, b.String(), termCols); got.rows != termRows {
		t.Errorf("a one-line question painted a %d-row frame in a %d-row terminal — the bar is "+
			"floating under the content rather than pinned to the bottom", got.rows, termRows)
	}
}

// DONE-WHEN 12: the EDITOR's footer still follows its content.
//
// The other half of the same decision, and the one that makes it a decision
// rather than a fix: a REPL prompt belongs directly under the last output, not
// stranded at the screen's edge. `newPinnedScreen`'s padding leaking into the
// editor would be a change to the appearance of a loop people already use — a
// second issue wearing this one's clothes.
func TestTheEditorsFooterFollowsItsContent(t *testing.T) {
	const termRows, termCols = 24, 80
	var sc screen // NOT pinned: the editor's
	sc.Write([]byte("arrondissement\n"))

	var b strings.Builder
	sc.Paint(&b, termRows, termCols, "› syc", []string{"  /help"})

	// One buffer line, one prompt row, one menu row.
	if got := readFrame(t, b.String(), termCols); got.rows != 3 {
		t.Errorf("the editor's frame is %d rows for one line of output, want 3 — the dropdown "+
			"belongs under the line being typed, not at the bottom of the screen", got.rows)
	}
}

// The padding is ROWS, emitted at paint time, and never LINES in the buffer.
//
// The transcript and the click map must not gain rows that exist only because
// the terminal is tall — so this paints twice at two heights, which is what a
// resize is, and asserts the buffer did not move.
func TestPaddingNeverReachesTheTranscript(t *testing.T) {
	const termCols = 80
	sc := screen{pinned: true}
	sc.Write([]byte("sycophantic\nbehaving in an obsequious way\n"))
	before := len(sc.Lines())

	var tall, short strings.Builder
	sc.Paint(&tall, 40, termCols, "y = got it", []string{"the bar"})
	sc.Paint(&short, 12, termCols, "y = got it", []string{"the bar"})

	// The padding actually happened, or the assertion below is about nothing.
	if got := readFrame(t, tall.String(), termCols); got.rows != 40 {
		t.Fatalf("the tall frame is %d rows, want 40 — nothing was padded, so this test asserts nothing", got.rows)
	}
	if after := len(sc.Lines()); after != before {
		t.Errorf("the buffer went from %d lines to %d across two paints — blank rows are being "+
			"APPENDED, so the exit transcript grows with the terminal's height", before, after)
	}
}

// newPinnedScreen is the difference, and it is visible where the screen is BUILT
// rather than at every paint.
func TestOnlyThePinnedConstructorPads(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(io.Writer) *liveScreen
		want int
	}{
		{"the editor's", func(w io.Writer) *liveScreen { return newLiveScreen(w, 20, 40) }, 2},
		{"--play's", func(w io.Writer) *liveScreen { return newPinnedScreen(w, 20, 40) }, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tty strings.Builder
			live := tc.make(&tty)
			live.interval = -1
			live.Write([]byte("one line\n"))
			tty.Reset()
			live.Draw("the prompt", nil)

			if got := readFrame(t, tty.String(), 40); got.rows != tc.want {
				t.Errorf("frame = %d rows, want %d", got.rows, tc.want)
			}
		})
	}
}

// lastFrame is the most recent WHOLE frame in what a terminal received.
//
// Paint opens every frame with home-and-erase, so the bytes after the last one
// are what is on screen; everything before it has been erased. A pty test that
// searched the accumulated stream would find text the terminal had already
// wiped, which under #41 is most of it.
func lastFrame(painted string) string {
	if i := strings.LastIndex(painted, cursorHome+eraseDown); i >= 0 {
		return painted[i:]
	}
	return painted
}

// THE PINNED SCREEN WRAPS WHATEVER IS WRITTEN TO IT, whoever writes it.
//
// This is the seam fix for a class that took four findings: `Paint` CLIPS a line
// too wide for the terminal, and a sitting writes not only its own text but
// whatever the helpers it calls write. Three earlier versions enforced the wrap
// at call sites — the queue build, then the loop's writes — and each time a
// site outside them was found, most recently `playAnnounced`'s network warning
// at 156 cells in a 40-column terminal. Asserting it HERE covers every caller
// including ones that do not exist yet.
func TestThePinnedScreenWrapsWhateverIsWrittenToIt(t *testing.T) {
	const cols = 40
	long := "define: " + strings.Repeat("a warning from some helper ", 8)

	t.Run("--play's wraps it", func(t *testing.T) {
		var tty strings.Builder
		live := newPinnedScreen(&tty, 24, cols)
		live.interval = -1
		fmt.Fprintln(live, long)

		for _, line := range strings.Split(live.Transcript(), "\n") {
			if n := visibleCells(line); n > cols {
				t.Errorf("a buffer line is %d columns in a %d-column terminal, so Paint clips it: %q", n, cols, line)
			}
		}
		// ...and nothing was lost to the wrap.
		if flat := strings.Join(strings.Fields(live.Transcript()), " "); !strings.Contains(flat, strings.Join(strings.Fields(long), " ")) {
			t.Errorf("the text did not survive wrapping:\n%s", live.Transcript())
		}
	})

	t.Run("the editor's does not", func(t *testing.T) {
		// Its text is pre-wrapped by Render, and its ask path streams token by
		// token — where a chunk ending mid-line has no line to wrap yet.
		var tty strings.Builder
		live := newLiveScreen(&tty, 24, cols)
		live.interval = -1
		fmt.Fprintln(live, long)

		if !strings.Contains(live.Transcript(), long) {
			t.Errorf("the editor's screen rewrote what it was given:\n%s", live.Transcript())
		}
	})
}

func TestPaintFitsTheTerminalAndParksTheCursor(t *testing.T) {
	const prompt = "› syc"
	for _, tc := range []struct {
		name     string
		lines    []string
		termRows int
		termCols int
		prompt   string
		menu     []string
	}{
		{"a short session", []string{"one", "two"}, 10, 80, prompt, nil},
		{"the buffer overflows", repeated(40, "a line"), 10, 80, prompt, nil},
		// The measurement from BR-6/BR-12: ten 200-column lines in 80 columns
		// need 28 rows if a line is counted as one.
		{"lines wider than the terminal", repeated(10, strings.Repeat("x", 200)), 10, 80, prompt, nil},
		{"coloured lines are measured by what is visible", repeated(6, "\x1b[1;36m"+strings.Repeat("c", 100)+"\x1b[0m"), 8, 40, prompt, nil},
		{"CJK is two columns a rune", repeated(10, strings.Repeat("日", 100)), 10, 80, prompt, nil},
		{"a menu under the prompt", []string{"one", "two"}, 10, 80, prompt, []string{"  /help", "  /history"}},
		// BR-32's measurements: the live edge alone taller than the terminal.
		{"a menu row that wraps three ways", []string{"one"}, 12, 15, "› /", []string{strings.Repeat("m", 36), strings.Repeat("n", 36), strings.Repeat("o", 36)}},
		{"more menu than terminal", []string{"one"}, 5, 80, prompt, []string{"a", "b", "c", "d", "e", "f", "g"}},
		{"a prompt longer than the terminal is wide", nil, 6, 20, "› " + strings.Repeat("z", 55), []string{"m1"}},
		{"a terminal too short for anything", []string{"one"}, 2, 20, prompt, []string{"m1", "m2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sc screen
			for _, l := range tc.lines {
				sc.Write([]byte(l + "\n"))
			}
			var b strings.Builder
			sc.Paint(&b, tc.termRows, tc.termCols, tc.prompt, tc.menu)
			got := readFrame(t, b.String(), tc.termCols)

			// 1. It FITS. A taller frame makes the terminal scroll, and every row
			// the app believes it placed moves with it.
			if got.rows > tc.termRows {
				t.Errorf("the frame needs %d display rows in a %d-row terminal", got.rows, tc.termRows)
			}
			// 2. It leaves the cursor at the end of the prompt — where the user
			// is typing. Deleting the cursor-up-and-reprint block, or walking
			// back by menu entries rather than rows, lands it somewhere else.
			// Where the cursor SHOULD be: at the end of the prompt, which is
			// the last cell the prompt occupies. Derived by replaying the
			// prompt through the same terminal model, so this asserts a
			// position rather than a formula copied out of Paint.
			shown := clipVisible(tc.prompt, tc.termRows*tc.termCols)
			want := readFrame(t, shown, tc.termCols)
			if got.cursorCol != want.cursorCol {
				t.Errorf("the cursor rests at column %d, want %d — the next keystroke redraws in the wrong place",
					got.cursorCol, want.cursorCol)
			}
			// The ROW, exactly — not merely "not the last one". An off-by-one
			// UPWARD in the walk-back leaves the cursor above the prompt, where
			// the next redraw overwrites the buffer's last line, and that was
			// caught only where the buffer happened to be empty.
			//
			// The target is the END of the prompt, which is where a person is
			// typing — and for a prompt that wraps, that is its LAST row, not
			// its first. Composed from where the prompt starts in the frame plus
			// where the cursor lands within it.
			wantRow := got.rows - footerHeight(tc.menu, tc.termCols) - want.rows + want.cursorRow
			if wantRow < 0 {
				wantRow = 0
			}
			if got.cursorRow != wantRow {
				t.Errorf("the cursor rests on row %d of a %d-row frame, want %d — the end of the prompt",
					got.cursorRow, got.rows, wantRow)
			}
		})
	}
}

// Clipping happens at PAINT time, so the buffer — and with it the exit
// transcript and M2's click map — keeps the whole text.
func TestScreenClipsTheViewNotTheBuffer(t *testing.T) {
	var s screen
	long := strings.Repeat("x", 200)
	s.Write([]byte(long + "\n"))
	var b strings.Builder
	s.Paint(&b, 10, 80, "› ", nil)

	if strings.Contains(b.String(), long) {
		t.Error("the frame carried the whole over-wide line")
	}
	if got := s.Transcript(); !strings.Contains(got, long) {
		t.Error("clipping reached the buffer: the transcript lost text the session showed")
	}
}

// A cut inside a colour hands the style back, or the terminal keeps painting
// with it to the end of the row and into the next line.
func TestClipVisibleClosesAnOpenStyle(t *testing.T) {
	got := clipVisible("\x1b[1;36m"+strings.Repeat("c", 40)+"\x1b[0m", 10)
	if visibleCells(got) != 10 {
		t.Errorf("clipped to %d visible columns, want 10: %q", visibleCells(got), got)
	}
	if !strings.HasSuffix(got, sgrOff) {
		t.Errorf("a cut inside a style did not close it: %q", got)
	}
	// A line that fits is returned untouched, escapes and all.
	if in := "\x1b[1mshort\x1b[0m"; clipVisible(in, 40) != in {
		t.Errorf("a line that fits was rewritten: %q", clipVisible(in, 40))
	}
}

func repeated(n int, line string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = line
	}
	return out
}

// A frame per streamed delta is a full-screen erase-and-redraw per token —
// hundreds of them, megabytes to the tty, for one answer. So writes are
// throttled, and what that buys is a bound on STALENESS rather than a change of
// behaviour: the unpainted text is never more than one interval's worth, and
// every gesture that ends a burst paints unconditionally.
func TestLiveScreenThrottlesTheRepaintButNeverLosesTheLastWord(t *testing.T) {
	var tty countingWriter
	// A long window, so "pending" is deterministic: no write paints itself and
	// the trailing timer cannot fire during the test. Racing the real 16 ms is
	// how a scheduling hiccup becomes a false failure.
	l := &liveScreen{s: &screen{}, tty: &tty, rows: 24, cols: 80, interval: time.Hour}
	l.Draw("› ", nil) // the first frame
	before := tty.painted()

	// A burst, as the ask path delivers it.
	for i := 0; i < 200; i++ {
		l.Write([]byte("token "))
	}
	if got := tty.painted() - before; got != 0 {
		t.Errorf("200 deltas painted %d frames inside one window — the throttle is not holding", got)
	}
	// The burst is still ALL in the buffer; only the painting was skipped.
	if n := strings.Count(l.Transcript(), "token"); n != 200 {
		t.Errorf("the buffer holds %d tokens, want 200 — the throttle dropped text", n)
	}
	// And the loop's own redraw settles what the throttle held back.
	frames := tty.painted()
	l.Draw("› ", nil)
	if tty.painted() != frames+1 {
		t.Error("Draw did not paint: a burst could end with text the user never sees")
	}

	// Stop flushes what the window is still holding, which is the exit path: the
	// last thing a session showed must not be the thing the throttle held back.
	l.Write([]byte("the last word\n"))
	frames = tty.painted()
	l.Stop()
	// Only the frame count, and only through its accessor: liveScreen's own
	// fields are written by the trailing timer's goroutine. -race happens not to
	// report the direct read because the timer reliably fires after it, which is
	// exactly why the discipline has to be structural rather than observed.
	if tty.painted() == frames {
		t.Error("Stop left a pending frame unpainted")
	}
	if !strings.Contains(l.Transcript(), "the last word") {
		t.Error("the last write never reached the buffer")
	}
}

// countingWriter counts frames. Locked, because the trailing timer paints from
// its own goroutine.
type countingWriter struct {
	mu     sync.Mutex
	frames int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames++
	return len(p), nil
}

func (c *countingWriter) painted() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.frames
}

// The TRAILING half, on its own and on the real clock — it is what makes the
// throttle safe rather than merely cheap.
//
// `♫ playing 3×` is written and then playback blocks for SECONDS. A throttle
// that waited for the next write would hide the indicator for the whole
// recording, which is worse than the redraw storm being fixed.
func TestLiveScreenFlushesAHeldFrameWithNoFurtherWrites(t *testing.T) {
	var tty countingWriter
	l := newLiveScreen(&tty, 24, 80)
	l.Draw("› ", nil)
	before := tty.painted()

	l.Write([]byte("  ♫ playing 3×")) // inside the window: held
	waitFor(t, func() bool { return tty.painted() > before })
}

// footerHeight is what the footer costs the frame, for the test's own
// arithmetic — deliberately recomputed from the FITTED footer rather than read
// out of Paint, so the assertion cannot agree with the code by construction.
func footerHeight(footer []string, cols int) int {
	n := 0
	for _, m := range footer {
		n += displayRows(m, cols)
	}
	return n
}

// The hit test, and the mapping the alternate screen exists to make exact.
//
// A click arrives as a VIEWPORT row; a region was collected against a Render's
// own line numbers. What joins them is that nothing but this type can move the
// view (#30 D1), so the arithmetic is exact rather than a guess about what the
// terminal did.
func TestScreenResolvesAClickToWhatWasRenderedThere(t *testing.T) {
	var s screen
	s.rows = 10
	// A session: a committed line, then an entry with regions, then another.
	s.Write([]byte("› arrondissement\n"))
	s.addRegions([]Region{
		{Kind: RegionHeadword, Text: "arrondissement", Line: 0, Col: 0, Width: 14},
		{Kind: RegionOriginLang, Text: "French", Lang: "fr", Line: 3, Col: 4, Width: 6},
	})
	s.Write([]byte("arrondissement\nnoun\n\n    French, from arrondir.\n"))

	for _, tc := range []struct {
		name     string
		row, col int
		want     string // "" means nothing is offered there
		lang     string
	}{
		{"the headword", 1, 0, "arrondissement", ""},
		{"the last cell of the headword", 1, 13, "arrondissement", ""},
		{"one past its end offers nothing", 1, 14, "", ""},
		{"the ORIGIN language", 4, 4, "French", "fr"},
		{"before it on the same line", 4, 3, "", ""},
		{"a line with no regions", 2, 0, "", ""},
		{"a row past the buffer is not a click target", 9, 0, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			line, ok := s.LineAt(tc.row)
			if !ok {
				if tc.want != "" {
					t.Fatalf("viewport row %d maps to no buffer line", tc.row)
				}
				return
			}
			r, hit := s.RegionAt(line, tc.col)
			if tc.want == "" {
				if hit {
					t.Errorf("row %d col %d offered %q, want nothing", tc.row, tc.col, r.Text)
				}
				return
			}
			if !hit {
				t.Fatalf("row %d col %d offered nothing, want %q", tc.row, tc.col, tc.want)
			}
			if r.Text != tc.want {
				t.Errorf("row %d col %d offered %q, want %q", tc.row, tc.col, r.Text, tc.want)
			}
			if string(r.Lang) != tc.lang {
				t.Errorf("region %q carries language %q, want %q", r.Text, r.Lang, tc.lang)
			}
		})
	}
}

// SCROLLING MOVES THE MAP WITH THE TEXT, which is the property the whole screen
// was built for: a click at viewport row R is buffer line R+offset, exactly,
// because nothing else can scroll.
func TestClicksFollowTheTextWhenScrolled(t *testing.T) {
	var s screen
	s.rows = 5
	for i := 0; i < 20; i++ {
		s.Write([]byte("filler\n"))
	}
	s.addRegions([]Region{{Kind: RegionHeadword, Text: "potassium", Line: 0, Col: 0, Width: 9}})
	s.Write([]byte("potassium\n"))
	for i := 0; i < 10; i++ {
		s.Write([]byte("more\n"))
	}

	// Scroll until the word is on screen, then click it wherever it landed.
	found := false
	for back := 0; back < 20 && !found; back++ {
		s.offset = back
		for row := 0; row < s.rows; row++ {
			line, ok := s.LineAt(row)
			if !ok {
				continue
			}
			if r, hit := s.RegionAt(line, 0); hit && r.Text == "potassium" {
				// And the text really is on that row of the frame.
				if got := s.Frame()[row]; got != "potassium" {
					t.Errorf("the region resolves at row %d, where the screen shows %q", row, got)
				}
				found = true
			}
		}
	}
	if !found {
		t.Error("the region was unreachable at every offset — the click map does not move with the text")
	}
}

// A region belongs to the line the render landed on, not to the render's own
// line 0. The caller ends the prompt line before writing an entry, so the entry
// begins on the next buffer line — an off-by-one here puts every click one line
// above what it points at.
func TestRegionsLandOnTheLinesTheirRenderWroteTo(t *testing.T) {
	var s screen
	s.rows = 20
	s.Write([]byte("first\nsecond\n"))
	s.addRegions([]Region{{Kind: RegionHeadword, Text: "third", Line: 0, Col: 0, Width: 5}})
	s.Write([]byte("third\n"))

	if _, hit := s.RegionAt(2, 0); !hit {
		t.Error("the region is not on line 2, where its render wrote")
	}
	for _, ln := range []int{0, 1, 3} {
		if r, hit := s.RegionAt(ln, 0); hit {
			t.Errorf("line %d offers %q, which was rendered elsewhere", ln, r.Text)
		}
	}

	// And when the writer is MID-LINE, a render's line 0 continues that open
	// line rather than starting a new one. Untested, this branch is arithmetic
	// nobody has checked — and it is one line off in the direction that puts
	// every click above what it points at.
	var open screen
	open.rows = 20
	open.Write([]byte("done\n"))
	open.Write([]byte("still open: ")) // no newline
	open.addRegions([]Region{{Kind: RegionHeadword, Text: "here", Line: 0, Col: 12, Width: 4}})
	open.Write([]byte("here\n"))

	if r, hit := open.RegionAt(1, 12); !hit || r.Text != "here" {
		t.Errorf("a render starting mid-line put its region elsewhere: line 1 offers %q (hit=%v), and the buffer is %q",
			r.Text, hit, open.Lines())
	}
}

// A clickable span is visibly clickable BEFORE it is clicked (#30 Done-when 4).
//
// Without a mark the affordance is invisible: a reader who never moves the mouse
// cannot tell the token is live, and the feature may as well not exist. Static
// rather than on hover, because hover needs mode 1003 — an event per cell the
// pointer crosses — while mode 1000 reports presses only and never tells the app
// where the pointer is.
func TestScreenMarksClickableSpans(t *testing.T) {
	var s screen
	s.rows = 10
	s.addRegions([]Region{
		{Kind: RegionHeadword, Text: "potassium", Word: "potassium", Line: 0, Col: 0, Width: 9},
		{Kind: RegionOriginLang, Text: "French", Word: "x", Lang: "fr", Line: 1, Col: 9, Width: 6},
	})
	s.Write([]byte("potassium is a metal\nfrom the French potasse\n"))

	var b strings.Builder
	s.Paint(&b, 10, 80, "› ", nil)
	frame := b.String()

	// The span is underlined, and the underline ENDS with it: an unterminated
	// attribute runs on through everything painted after it.
	if !strings.Contains(frame, underlineOn+"potassium"+underlineOff) {
		t.Errorf("the headword is not marked as clickable:\n%q", frame)
	}
	if !strings.Contains(frame, underlineOn+"French"+underlineOff) {
		t.Errorf("the ORIGIN language is not marked as clickable:\n%q", frame)
	}
	// And the mark does not move the text: a reader's columns are unchanged, and
	// so is the click map that was built against them.
	for i, line := range strings.Split(strings.TrimSuffix(frame, "\r\n"), "\r\n") {
		if i == 0 {
			line = strings.TrimPrefix(line, cursorHome+eraseDown)
		}
		if got := visibleCells(line); got > 80 {
			t.Errorf("marking widened a line to %d cells", got)
		}
	}
	if plain := stripEscapes(frame); !strings.Contains(plain, "potassium is a metal") {
		t.Errorf("marking altered the text: %q", plain)
	}
}

// The mark is an ATTRIBUTE, so the span keeps the colour the palette gave it.
//
// Turning the underline off with `0` instead of `24` would end that colour too,
// and the rest of the line would go plain — the bug this test exists to prevent.
func TestMarkingKeepsTheSpansOwnColour(t *testing.T) {
	line := "\x1b[1;36mpotassium\x1b[0m is a metal"
	got := markClickable(line, []Region{{Text: "potassium", Col: 0, Width: 9}})

	if !strings.Contains(got, "\x1b[1;36m") {
		t.Errorf("the span lost its colour: %q", got)
	}
	if strings.Contains(got, underlineOff) && strings.Contains(got, underlineOn+"\x1b[0m") {
		t.Errorf("the mark reset the style rather than just the attribute: %q", got)
	}
	if stripEscapes(got) != stripEscapes(line) {
		t.Errorf("marking changed the text: %q", stripEscapes(got))
	}
}

// Marks are placed by COLUMN, so a coloured line — where the bytes before a span
// are not its column — is marked in the right place.
func TestMarkingIsPlacedByColumnNotByByte(t *testing.T) {
	for _, tc := range []struct {
		name, line string
		col, width int
		want       string
	}{
		{"plain", "abc def", 4, 3, "abc " + underlineOn + "def" + underlineOff},
		{"after an escape", "\x1b[1mabc\x1b[0m def", 4, 3, "\x1b[1mabc\x1b[0m " + underlineOn + "def" + underlineOff},
		{"a span at the end closes", "abc def", 4, 3, "def" + underlineOff},
		// Two cells a rune: a mark placed by byte would land three cells early.
		{"after CJK", "日本 def", 5, 3, "日本 " + underlineOn + "def" + underlineOff},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := markClickable(tc.line, []Region{{Text: "def", Col: tc.col, Width: tc.width}})
			if !strings.Contains(got, tc.want) {
				t.Errorf("markClickable(%q) = %q, want it to contain %q", tc.line, got, tc.want)
			}
		})
	}
}

// The PRODUCTION join, on the real liveScreen (#30 M2, BR-37).
//
// The two halves were each pinned and the object that joins them was not: the
// screen's own hit test is checked in this file, the loop's use of it is checked
// against a scripted double, and `liveScreen.WriteRegions` sat between them
// untested. Swapping its two statements — recording the regions AFTER writing
// the text rather than before — left the whole suite green while shifting every
// region forward by the entry's line count in production.
//
// The rule that leaves: a double may not stand in for the object joining two
// separately-pinned halves. A "pinned by" claim holds only when mutating the
// IMPLEMENTING code reddens a named test.
func TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 24, 80)
	l.interval = -1 // paint every write, so the frame is never a throttled one

	// A session as the loop produces it: a committed line, then an entry whose
	// regions are relative to its OWN first line.
	l.Write([]byte("› concrete\r\n"))
	l.WriteRegions("concrete  con·crete\nnoun a building material.\n\n  ORIGIN\n    from French concret.\n",
		[]Region{
			{Kind: RegionHeadword, Text: "concrete", Word: "concrete", Line: 0, Col: 0, Width: 8},
			{Kind: RegionOriginLang, Text: "French", Word: "concrete", Lang: "fr", Line: 4, Col: 9, Width: 6},
		})

	// Resolve through the VIEWPORT, which is what a click gives us.
	for _, tc := range []struct {
		name     string
		row, col int
		want     string
	}{
		{"the headword, one line below the committed line", 1, 0, "concrete"},
		{"the ORIGIN language, four lines further on", 5, 9, "French"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := l.RegionAtRow(tc.row, tc.col)
			if !ok {
				t.Fatalf("row %d col %d offers nothing; the buffer is %q", tc.row, tc.col, l.s.Lines())
			}
			if r.Text != tc.want {
				t.Errorf("row %d col %d offers %q, want %q", tc.row, tc.col, r.Text, tc.want)
			}
		})
	}

	// And the text really is where the click map says it is — the frame shows
	// the marked span on that row, so the two cannot agree with each other while
	// both being wrong about the screen.
	frame := tty.String()
	if !strings.Contains(frame, underlineOn+"French"+underlineOff) {
		t.Errorf("the frame does not mark the span the hit test resolves: %q", frame)
	}
}

// The click map stays attached when the viewport GROWS (#30 M2, BR-42).
//
// `Frame` used to return early — for an empty viewport, and for a buffer that
// fits — without clamping, so a stale offset survived. Growing the viewport
// while scrolled back then left `offset` past the end and the top line negative:
// the underline painted on one row while the region answered on another, which
// is the exact-placement property the alternate screen exists to give.
//
// Three routine ways in, all of them growth: a resize taller, the command menu
// closing, and a wrapped prompt cleared with Ctrl-U.
func TestClickMapSurvivesTheViewportGrowing(t *testing.T) {
	for _, tc := range []struct {
		name string
		grow func(l *liveScreen)
	}{
		{"a resize taller", func(l *liveScreen) { l.Resize(30, 80) }},
		// The menu closing gives its rows back to the buffer, which is the same
		// growth arriving through Paint rather than through SIGWINCH.
		{"the command menu closing", func(l *liveScreen) { l.Draw("› ", nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tty strings.Builder
			l := newLiveScreen(&tty, 12, 80)
			l.interval = -1
			l.Draw("› ", []string{"m1", "m2", "m3"})
			for i := 0; i < 20; i++ {
				l.Write([]byte("filler\n"))
			}
			l.WriteRegions("potassium\n", []Region{
				{Kind: RegionHeadword, Text: "potassium", Word: "potassium", Line: 0, Col: 0, Width: 9},
			})
			l.Page(10) // scroll back, far enough to overshoot when the viewport grows
			tc.grow(l)
			l.Draw("› ", nil) // the redraw the loop performs after any growth

			// Wherever the word is showing now, a click on it must find it —
			// and the row that answers must be the row that SHOWS it. Read
			// through Frame, which both shapes of this code have, so the test
			// can be run against the one it was written to catch.
			frame := l.s.Frame()
			row := -1
			for i, line := range frame {
				if strings.Contains(line, "potassium") {
					row = i
				}
			}
			if row < 0 {
				return // scrolled out of view: nothing to click, which is fine
			}
			r, ok := l.RegionAtRow(row, 0)
			if !ok {
				t.Fatalf("the word shows on row %d and offers nothing: the map detached from the text", row)
			}
			if r.Text != "potassium" {
				t.Errorf("row %d offers %q, want the word shown there", row, r.Text)
			}
		})
	}
}
