package main

import (
	"bytes"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
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
	// Far enough past the last frame that the throttle does not hold this one —
	// what is under test here is that a write SHOWS, not how soon.
	l.painted = time.Now().Add(-paintInterval)
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

// A frame fits the terminal in DISPLAY ROWS, not in lines.
//
// A line wider than the terminal wraps onto a second row, so a frame that
// counted it as one is a frame one row too tall — and the terminal then SCROLLS
// to fit it, which moves every row the app believes it placed. That is the exact
// property the alternate screen was taken for: a click at viewport row R is
// buffer line R+offset only while nothing but this program can move the view.
//
// Two routine ways in, and both are reachable today: narrowing the window (buffer
// lines keep the wrapping they were rendered with, by decision) and typing a line
// longer than the terminal is wide, since the committed line goes to the buffer.
func TestScreenFrameFitsTheTerminalInDisplayRows(t *testing.T) {
	for _, tc := range []struct {
		name           string
		lines          []string
		termRows       int
		termCols       int
		prompt         string
		menu           []string
		wantRowsAtMost int
	}{
		{
			// The reviewer's measurement: ten 200-column lines in an 80-column
			// terminal need 28 display rows when each is counted as one.
			name:     "lines wider than the terminal",
			lines:    repeated(10, strings.Repeat("x", 200)),
			termRows: 10, termCols: 80, prompt: "› ",
			wantRowsAtMost: 10,
		},
		{
			name:     "a prompt longer than the terminal is charged its real height",
			lines:    []string{"a", "b", "c", "d", "e", "f"},
			termRows: 6, termCols: 20, prompt: "› " + strings.Repeat("z", 55),
			wantRowsAtMost: 6,
		},
		{
			name:     "a menu row that wraps is charged too",
			lines:    []string{"a", "b", "c", "d", "e", "f"},
			termRows: 8, termCols: 20, prompt: "› ",
			menu:           []string{strings.Repeat("m", 45)},
			wantRowsAtMost: 8,
		},
		{
			name:     "coloured text is measured by what is VISIBLE",
			lines:    []string{"\x1b[1;36m" + strings.Repeat("c", 100) + "\x1b[0m"},
			termRows: 4, termCols: 40, prompt: "› ",
			wantRowsAtMost: 4,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s screen
			for _, l := range tc.lines {
				s.Write([]byte(l + "\n"))
			}
			var b strings.Builder
			s.Paint(&b, tc.termRows, tc.termCols, tc.prompt, tc.menu)

			// Count what the TERMINAL would count: every row a line occupies
			// once it has wrapped, plus the row each explicit break starts.
			rows := 0
			for _, painted := range strings.Split(b.String(), "\r\n") {
				rows += displayRows(strings.TrimPrefix(painted, cursorHome+eraseDown), tc.termCols)
			}
			if rows > tc.wantRowsAtMost {
				t.Errorf("the frame needs %d display rows in a %d-row terminal — it scrolls, and every placed row moves",
					rows, tc.wantRowsAtMost)
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
	if visibleLen(got) != 10 {
		t.Errorf("clipped to %d visible columns, want 10: %q", visibleLen(got), got)
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
	l := newLiveScreen(&tty, 24, 80)
	l.Draw("› ", nil) // the first frame
	before := tty.painted()

	// A burst, as the ask path delivers it.
	for i := 0; i < 200; i++ {
		l.Write([]byte("token "))
	}
	if got := tty.painted() - before; got > 5 {
		t.Errorf("200 deltas painted %d frames — the throttle is not holding", got)
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

	// The TRAILING half, and it is what makes the throttle safe rather than
	// merely cheap: a held frame goes out whether or not another write follows.
	// The indicator is written and then playback blocks for seconds — a throttle
	// waiting for the next write would hide it for the whole recording.
	l.Write([]byte("  ♫ playing 3×"))
	frames = tty.frames
	waitFor(t, func() bool { return tty.painted() > frames })

	// Stop flushes too, for the exit path.
	l.Write([]byte("the last word\n"))
	frames = tty.painted()
	l.Stop()
	if tty.painted() == frames && l.pending {
		t.Error("Stop left a pending frame unpainted")
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
