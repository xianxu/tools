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

// maxPasteBytes is the MEMORY bound on an unterminated paste, in the widest
// bytes a rune can take.
//
// Separate from maxPasteRunes on purpose: the rune cap is semantic and can only
// be judged on complete text, at the closer. This one is judged while the text is
// still arriving and may end mid-rune, so it has to be in bytes or it refuses
// legal pastes (BR-12).
const maxPasteBytes = maxPasteRunes * utf8.UTFMax

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

	// A paste is TEXT, and sanitisePasteBody already says what that means: every
	// control rune but newline and tab is stripped. So a control byte arriving
	// inside an OPEN paste means the terminal never closed it — the paste is
	// malformed, and waiting for a closer that is not coming is not an option.
	//
	// Raw mode disables ISIG, so Ctrl-C is reachable ONLY as a decoded
	// KeyInterrupt. A paste that swallows it makes the program unquittable from
	// the keyboard, which is what an earlier version of this file did: under the
	// cap it returned 0 forever and readInput never advanced its buffer; over the
	// cap `draining` latched and discarded every byte until a closer that never
	// came. Found by the M1 boundary review, reproduced through the real
	// readInput goroutine.
	//
	// Abandoning restores exactly the behaviour before this milestone: the start
	// marker decodes as an inert unmodelled sequence and the rest is ordinary
	// input.
	closer := bytes.Index(body, []byte(pasteEnd))
	if quit := indexPasteAbandon(body); quit >= 0 && (closer < 0 || quit < closer) {
		s.draining = false
		if opened > 0 {
			return Key{Kind: KeyUnknown, Raw: []byte(pasteStart)}, opened
		}
		return Key{}, 0
	}

	if end := closer; end >= 0 {
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

	// No closer yet. This branch is a MEMORY bound and it is measured in BYTES.
	//
	// The semantic cap is in runes, but it cannot be applied here: the buffer may
	// end mid-rune, and utf8.RuneCount counts each orphan byte as a RuneError. A
	// 1000-rune CJK paste split so one scan sees 2999 body bytes counts 1001 —
	// 999 whole runes plus two orphans — latches the drain, and then refuses a
	// paste that fits. A false refusal on exactly the decks /lang exists for,
	// which is the argued point of having a rune cap at all. Found by the M1
	// boundary review (BR-12).
	//
	// So one predicate no longer serves two purposes. Here: bound the memory, in
	// the widest bytes a rune can take. At the closer, where the text is whole:
	// the rune cap, and only there.
	if !s.draining && len(body) <= maxPasteBytes {
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
// The body is arbitrary text from the user's clipboard, and it reaches the screen:
// today the line editor, whose RenderLine opens its own styles around what it
// draws. A pasted escape would leak out of the line and repaint the frame around
// it. Stripping at the boundary makes that unrepresentable instead of checked
// downstream — the same move oneLine makes at the store boundary
// (store/item.go:200), and it holds for any later surface without being restated
// there.
//
// Newlines and tabs SURVIVE: a passage has lines, and a tab is text.
//
// The set removed is unicode.IsControl — category Cc, which is C0 and C1. That is
// the DECISION, not an accident of which predicate came to hand: Cc is what a
// terminal would act on, and it is what makes a pasted escape or a stray NUL
// unable to reach the screen. Format characters (Cf: zero-width joiner, bidi
// marks) are deliberately KEPT, because they are part of the text a reader
// pasted — stripping them would silently alter words in scripts that need them.
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

// pasteLineRunes is a paste on its way into the LINE editor, which holds one
// line.
//
// An interior newline becomes a space rather than submitting — which is the
// whole point of bracketing pastes — and rather than being dropped, because
// "hot\ndog" is two words and joining them into "hotdog" would invent a word.
// parseREPLLine collapses the run afterwards, so this only has to be lossless
// about the boundary, not about the whitespace.
func pasteLineRunes(s string) []rune {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' {
			r = ' '
		}
		out = append(out, r)
	}
	return out
}

// indexPasteAbandon finds the first byte that cannot be inside a paste.
//
// The set is the control bytes the program acts on regardless of mode —
// interrupt and end-of-file — because those are the two the reader must never
// lose. Newline and tab are deliberately absent: they are paste text, which is
// the whole reason bracketing exists.
func indexPasteAbandon(b []byte) int {
	for i, c := range b {
		if c == 0x03 || c == 0x04 {
			return i
		}
	}
	return -1
}
