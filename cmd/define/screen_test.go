package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
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
			s.Paint(&b, tc.termRows, "PROMPT", tc.menu)
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
	l := newLiveScreen(&tty, 10)

	l.Draw("› ", nil)
	tty.Reset()
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
