package main

import (
	"bytes"
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
		// crlfWriter's case, which the screen replaces on this path: a reply
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
			wantRow := got.rows - menuHeight(tc.menu, tc.termCols) - want.rows + want.cursorRow
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

// menuHeight is what the menu costs the frame, for the test's own arithmetic —
// deliberately recomputed from the FITTED menu rather than read out of Paint, so
// the assertion cannot agree with the code by construction.
func menuHeight(menu []string, cols int) int {
	n := 0
	for _, m := range menu {
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
