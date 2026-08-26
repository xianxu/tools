package main

import "strings"

// sgrState remembers the style currently in effect, so a highlight injected into
// styled text can hand the style back afterwards.
//
// It exists because ANSI does not nest. Writing green inside an italic-green
// example and then closing it with a reset would leave the REST of the example
// unstyled; the writer has to re-open what was in effect, and something has to
// know what that was.
type sgrState struct {
	// open is every SGR seen since the last reset, in order. A terminal composes
	// them (bold, then colour), so resuming means replaying them — keeping only
	// the last would drop the bold that Render opened separately.
	open []string
}

func (s *sgrState) observe(seq string) {
	if !isSGR(seq) {
		// A non-SGR CSI (erase-line, cursor moves) changes no style. Silently
		// ignoring it is the point: the raw loop puts those through this stream.
		return
	}
	if isReset(seq) {
		s.open = s.open[:0]
		return
	}
	s.open = append(s.open, seq)
}

// resume is what to emit to put the style back after a highlight.
func (s sgrState) resume() string { return strings.Join(s.open, "") }

// isSGR reports whether a sequence is a CSI ending in 'm'.
func isSGR(seq string) bool {
	return strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m")
}

// isReset covers both spellings a producer may use: an explicit 0 and the bare
// form, which means the same thing.
func isReset(seq string) bool {
	params := strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b["), "m")
	if params == "" {
		return true
	}
	for _, p := range strings.Split(params, ";") {
		if p != "0" && p != "" {
			return false
		}
	}
	return true
}

// scanEscape returns how many bytes of an escape sequence begin s, or -1 when
// the sequence has not finished arriving.
//
// The -1 is the streaming case and the reason this is not a regexp: a chunk can
// end mid-sequence, and emitting a half-escape would put garbage on the terminal
// and corrupt every style after it. The caller holds the bytes and waits.
func scanEscape(s string) int {
	if len(s) == 0 || s[0] != 0x1b {
		return 0
	}
	if len(s) == 1 {
		return -1
	}
	if s[1] != '[' {
		// A two-byte escape (ESC M, ESC 7, …). Nothing here changes style.
		return 2
	}
	for i := 2; i < len(s); i++ {
		// CSI runs until a final byte in @..~.
		if s[i] >= '@' && s[i] <= '~' {
			return i + 1
		}
	}
	return -1
}
