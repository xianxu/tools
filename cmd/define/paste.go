package main

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxPasteRunes bounds one passage: a sentence to a paragraph (#67).
//
// RUNES, not bytes, so a CJK paragraph is a paragraph — a byte cap would refuse
// one at roughly a third of its length, on exactly the decks `/lang` exists for.
//
// A paste over the cap is REFUSED rather than truncated, because a silently
// half-taken passage would produce an answer about text the reader cannot see.
// The refusal still CONSUMES its bytes: they would otherwise arrive as
// keystrokes, and readInput's buffer would grow with the input.
const maxPasteRunes = 1000

// The bracketed-paste markers (DEC mode 2004). The terminal wraps a paste in
// them so a program can tell pasted text from typed text — which is the whole
// point here, since a pasted newline must not submit the line.
const (
	pasteStart = "\x1b[200~"
	pasteEnd   = "\x1b[201~"
)

// pasteScanner decodes a bracketed paste.
//
// IT ACCUMULATES NOTHING, and that is the contract rather than an optimisation.
// readInput re-presents its whole buffer after a short read and advances only on
// consumption (selection_input.go:169-178), so a scanner that also buffered
// internally would see every byte twice: an earlier design did, and a 900-byte
// paste arriving in the real 256-byte chunks came back duplicated and then
// refused. With a 256-byte read and a 1000-rune cap, multi-read is the NORMAL
// path, not an edge.
//
// So the only state is `draining`, which is the one fact the buffer cannot
// carry: that an over-cap paste's bytes are being discarded until its closer
// arrives. The shape is selectionStep's — a decision that is a pure function of
// (state, input), drivable in a test without a terminal (ARCH-ORDER).
type pasteScanner struct{ draining bool }

// scan reports what the head of buf means, in decodeKey's own (Key, int)
// protocol: used == 0 means "a prefix, read more", and never "an empty paste".
//
// It answers for a buffer that is ALREADY known to be ours in one of two ways:
// the head is pasteStart, or a drain is in progress and every byte belongs to it
// until the closer. decodeKey's hook has to honour both — see the comment there,
// because hooking on ESC alone makes the drain unreachable.
func (s *pasteScanner) scan(buf []byte) (Key, int) {
	body, opened := buf, 0
	if !s.draining {
		if !bytes.HasPrefix(buf, []byte(pasteStart)) {
			return Key{}, 0
		}
		opened = len(pasteStart)
		body = buf[opened:]
	}

	if end := bytes.Index(body, []byte(pasteEnd)); end >= 0 {
		// An embedded closer ends the paste — that IS the protocol, and the
		// remainder is ordinary input. Decided rather than inherited, because
		// the remainder can contain a carriage return.
		used := opened + end + len(pasteEnd)
		if s.draining {
			s.draining = false
			return Key{Kind: KeyPasteRefused}, used
		}
		text := body[:end]
		if utf8.RuneCount(text) > maxPasteRunes {
			return Key{Kind: KeyPasteRefused}, used
		}
		return Key{Kind: KeyPaste, Raw: []byte(sanitisePasteBody(string(text)))}, used
	}

	// No closer yet. Under the cap, wait: the caller will re-present with more.
	if !s.draining && utf8.RuneCount(body) <= maxPasteRunes {
		return Key{}, 0
	}

	// Over the cap. Start discarding, and CONSUME, so the caller's buffer stops
	// growing — otherwise the memory bound is a comment rather than a property.
	//
	// Hold back the last len(pasteEnd)-1 bytes in case the closer straddles this
	// read: the same whole-or-nothing care decodeX10Mouse takes (key.go:330),
	// and without it a drain can cut its own exit and never end.
	s.draining = true
	used := opened + len(body) - (len(pasteEnd) - 1)
	if used <= 0 {
		return Key{}, 0
	}
	return Key{Kind: KeyUnknown}, used
}

// sanitisePasteBody is where untrusted bytes become a typed value (ARCH-SECURE).
//
// The body is arbitrary text from the user's clipboard, and it is bound for the
// footer, which passes producer SGR through by construction
// (selection_frame.go:236-245). A pasted escape would recolour the passage and
// defeat the mark painting, which re-asserts over KNOWN producer styling rather
// than arbitrary injected state. Stripping at the boundary makes that
// unrepresentable instead of checked downstream — the same move oneLine makes at
// the store boundary (store/item.go:200).
//
// Newlines and tabs SURVIVE: a passage has lines, and a tab is text. Every other
// control rune goes, including the C1 range utf8 decodes from valid input.
func sanitisePasteBody(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		// escapeLen owns the escape grammar for whole strings (render.go:582);
		// a second one here is exactly the divergence ARCH-DRY names.
		if n := escapeLen(s[i:]); n > 0 {
			i += n
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		i += n
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if r == utf8.RuneError && n == 1 {
			continue // an invalid byte is not text
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
