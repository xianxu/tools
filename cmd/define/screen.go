package main

import "strings"

// screen is the interactive loop's line buffer and viewport (#30).
//
// It exists to make a click's coordinates exact. `define` takes the ALTERNATE
// SCREEN, which has no scrollback — there is nothing above the viewport for the
// terminal to show — so nothing can move the view except this type, and a click
// at viewport row R is buffer line `R + offset` by construction rather than by
// tracking something the app never observes.
//
// It is an io.Writer, and that is what keeps this from being a rewrite: Render
// returns a string, the ask path streams, commands and the indicator print, and
// every one of them feeds the buffer unchanged. It REPLACES crlfWriter on this
// path — both translate for a raw terminal, and two owners of line endings is
// how they drift.
//
// Write and Frame do no terminal IO. Paint is the only part that touches a
// terminal, so the arithmetic here is unit-testable with no pty.
type screen struct {
	// lines is everything the session has shown, oldest first. The LAST line may
	// be partial: deltas arrive chunked and a reply split as "one" then " two\n"
	// is one line, not two.
	lines []string
	// partial reports whether the final element is still being written to, so a
	// later Write continues it rather than starting a line.
	partial bool
	// offset is how far back the viewport sits, in lines from the tail. 0 is the
	// bottom — where a session lives — so a fresh screen needs no initialisation.
	offset int
	rows   int
	cols   int
}

// Write appends bytes to the buffer, splitting on newlines.
//
// A bare "\r" is dropped rather than kept: it is the carriage half of a CRLF
// that arrived in a different chunk, which is the case crlfWriter documents
// ("a reply split as \"one\\r\" then \"\\ntwo\" must not become \"one\\r\\r\\ntwo\"").
// Here there is no terminal to position, so the CR carries no information at
// all — a line's placement is Paint's business.
func (s *screen) Write(p []byte) (int, error) {
	text := strings.ReplaceAll(string(p), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "")
	if text == "" {
		return len(p), nil
	}
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		if i == 0 && s.partial && len(s.lines) > 0 {
			s.lines[len(s.lines)-1] += part
			continue
		}
		s.lines = append(s.lines, part)
	}
	// A trailing "\n" ends the last line; anything else leaves it open.
	s.partial = !strings.HasSuffix(text, "\n")
	if !s.partial && len(s.lines) > 0 && s.lines[len(s.lines)-1] == "" {
		// Split leaves an empty tail after a terminating newline. Drop it, or
		// every completed write would add a blank line.
		s.lines = s.lines[:len(s.lines)-1]
	}
	return len(p), nil
}

// Lines is the whole buffer. Present for tests and for the exit transcript
// (D3), which is a loop over exactly this.
func (s *screen) Lines() []string { return s.lines }

// Frame is the rows to paint, oldest first. PURE.
//
// The offset is CLAMPED here rather than at the call sites, because a wheel
// event arrives per notch and a held PageUp repeats — both overshoot routinely,
// and clamping in one place is what stops an overshoot from indexing backwards.
func (s *screen) Frame() []string {
	if s.rows <= 0 {
		return nil
	}
	if len(s.lines) <= s.rows {
		return s.lines
	}
	max := len(s.lines) - s.rows
	off := s.offset
	if off > max {
		off = max
	}
	if off < 0 {
		off = 0
	}
	end := len(s.lines) - off
	return s.lines[end-s.rows : end]
}

// Scroll moves the viewport by n lines — positive is BACKWARD, toward older
// text, which is the direction "scroll up" means to a reader.
func (s *screen) Scroll(n int) {
	s.offset += n
	max := len(s.lines) - s.rows
	if max < 0 {
		max = 0
	}
	if s.offset > max {
		s.offset = max
	}
	if s.offset < 0 {
		s.offset = 0
	}
}
