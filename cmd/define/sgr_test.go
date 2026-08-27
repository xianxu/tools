package main

import "testing"

// One row per RULE, not one per escape code. What varies is how a sequence
// affects the style that must be restored after a highlight is injected.
func TestSGRState(t *testing.T) {
	for _, tc := range []struct {
		name string
		feed []string
		want string
	}{
		{"nothing seen", nil, ""},
		{"an SGR becomes the style to resume", []string{"\x1b[3;32m"}, "\x1b[3;32m"},
		{"a full reset clears it", []string{"\x1b[3;32m", "\x1b[0m"}, ""},
		{"a bare CSI m is also a reset", []string{"\x1b[1m", "\x1b[m"}, ""},
		// Render opens bold and colour separately in places, and a terminal
		// composes them — so must the resume.
		{"SGRs accumulate until a reset", []string{"\x1b[1m", "\x1b[32m"}, "\x1b[1m\x1b[32m"},
		{"a reset then a new SGR starts over", []string{"\x1b[1m", "\x1b[0m", "\x1b[32m"}, "\x1b[32m"},
		// A non-SGR CSI must not disturb the style: erase-line runs through the
		// same stream in raw mode.
		{"a non-SGR CSI leaves the style alone", []string{"\x1b[3;32m", "\x1b[2J", "\x1b[K"}, "\x1b[3;32m"},
		{"a non-CSI escape leaves the style alone", []string{"\x1b[3;32m", "\x1bM"}, "\x1b[3;32m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s sgrState
			for _, seq := range tc.feed {
				s.observe(seq)
			}
			if got := s.resume(); got != tc.want {
				t.Errorf("resume() = %q, want %q", got, tc.want)
			}
		})
	}
}

// scanEscape is what tells the writer where a sequence ENDS, including when it
// has not arrived yet — the streaming case.
func TestScanEscape(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want int // bytes consumed; -1 means "incomplete, hold everything"
	}{
		{"a complete SGR", "\x1b[1;32mrest", 7},
		{"a complete non-SGR CSI", "\x1b[2Jrest", 4},
		{"CSI with no final byte yet", "\x1b[1;32", -1},
		{"bare ESC at the end of a chunk", "\x1b", -1},
		{"ESC [ and nothing more", "\x1b[", -1},
		{"a two-byte escape", "\x1bMrest", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := scanEscape(tc.in); got != tc.want {
				t.Errorf("scanEscape(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// A reset arriving inside a region clears the ENCLOSING style too.
//
// Without this, highlighting a word after an inner reset resumed to base and
// styled text that is plain in the un-highlighted render — a styling change no
// escape-stripped comparison can see. prettyPronunciations emits an inner reset
// inside every example that mentions a pronunciation.
func TestAnInnerResetClearsTheEnclosingStyle(t *testing.T) {
	s := sgrState{base: "\x1b[3;32m"}

	if got := s.resume(); got != "\x1b[3;32m" {
		t.Fatalf("before any reset, resume() = %q, want the base", got)
	}
	s.observe("\x1b[35m")
	s.observe("\x1b[0m")

	if got := s.resume(); got != "" {
		t.Errorf("after an inner reset, resume() = %q, want nothing — the terminal is plain", got)
	}
}
