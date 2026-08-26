package main

import (
	"io"
	"strings"
	"unicode/utf8"
)

// highlightWriter rewrites words the learner knows as it copies a byte stream,
// in text that already carries ANSI codes and may arrive in pieces.
//
// One mechanism serves both surfaces that need it. A definition is a complete
// string (Render builds it, then it is printed); an answer arrives as stream
// deltas where `obsequious` can land as `obseq` + `uious`. Both need the same
// two hard things — never split an escape sequence, and never highlight a word
// before knowing where it ends — so both get the same writer rather than each
// growing its own. Same shape as crlfWriter, which carries `lastWasCR` across
// writes for exactly this class of reason.
//
// THE CONTRACT (workshop/plans/000021-highlight-learned-plan.md states it
// normatively; this is the summary the code has to keep):
//
//  1. Nothing overtakes held text. An escape arriving while text is held
//     RESOLVES the hold first, then passes through — otherwise a reset can end
//     up before the word it was closing.
//  2. A phrase spans only spaces and tabs. An escape, newline or punctuation
//     closes the window.
//  3. Downstream errors poison the writer: the first failure is remembered and
//     nothing is emitted after it, so no byte can be written twice.
//  4. Flush is part of the contract, not a convenience — held text is invisible
//     until it happens.
type highlightWriter struct {
	out  io.Writer
	v    Vocabulary
	on   string
	sgr  sgrState
	pend []byte
	err  error
}

// newHighlightWriter returns a writer that highlights, or a plain pass-through
// when there is nothing to highlight with.
//
// The guard lives HERE rather than in the region helper because M3 constructs this
// directly for the answer stream: an empty style would otherwise wrap every
// known word in a bare reset, stripping the enclosing style for no benefit.
// A guard only one of two callers gets is not a guard.
func newHighlightWriter(out io.Writer, v Vocabulary, on string) *highlightWriter {
	if on == "" {
		v = nil // nothing to inject: highlightSpans then returns one plain span
	}
	return &highlightWriter{out: out, v: v, on: on}
}

// Write reports (len(p), nil) on success and (0, err) on failure.
//
// The zero is deliberate and differs from crlfWriter in the same package, which
// returns caller-unit PROGRESS on a partial write. The difference is retry:
// crlfWriter can be written to again, so a caller must know how far it got, and
// crlf_test.go defends that. This writer POISONS on the first failure — every
// later Write returns the same error and emits nothing — so there is no retry to
// inform, and no byte can be written twice. Reporting a partial count here would
// invite a resume that the writer will never honour.
func (w *highlightWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	w.pend = append(w.pend, p...)
	if err := w.drain(false); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Flush emits everything still held. Every exit path must call it: a stream that
// ends without one silently drops its last word.
func (w *highlightWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	return w.drain(true)
}

func (w *highlightWriter) drain(final bool) error {
	for len(w.pend) > 0 {
		s := string(w.pend)
		if s[0] == 0x1b {
			n := scanEscape(s)
			if n < 0 {
				if !final {
					return nil // the rest of the sequence has not arrived
				}
				// At flush, a half-escape can only go out as it stands; holding
				// it would lose bytes, which is worse than a stray fragment.
				n = len(s)
			}
			seq := s[:n]
			w.sgr.observe(seq)
			if err := w.emit(seq); err != nil {
				return err
			}
			w.pend = w.pend[n:]
			continue
		}
		region := s
		endsAtBuffer := true
		if i := strings.IndexByte(s, 0x1b); i >= 0 {
			// Rule 1: the escape closes this region, so everything before it is
			// decided NOW and goes out before the escape does.
			region, endsAtBuffer = s[:i], false
		}
		cut := len(region)
		if endsAtBuffer && !final {
			cut = decidedEnd(region, w.v, w.maxPhraseWords())
		}
		if cut == 0 {
			return nil // nothing decidable yet; wait for more or for Flush
		}
		if err := w.emitText(region[:cut]); err != nil {
			return err
		}
		w.pend = w.pend[cut:]
		if cut < len(region) {
			return nil // the rest is held
		}
	}
	return nil
}

func (w *highlightWriter) maxPhraseWords() int {
	if w.v == nil {
		return 0
	}
	return w.v.MaxPhraseWords()
}

// emitText writes one decided region, wrapping known words and handing the
// enclosing style back after each — ANSI does not nest.
func (w *highlightWriter) emitText(text string) error {
	for _, sp := range highlightSpans(text, w.v) {
		if !sp.known {
			if err := w.emit(sp.text); err != nil {
				return err
			}
			continue
		}
		if err := w.emit(w.on + sp.text + sgrOff + w.sgr.resume()); err != nil {
			return err
		}
	}
	return nil
}

// emit is the only place bytes reach the downstream writer, so it is the only
// place that has to know a short write with a nil error is still a failure.
func (w *highlightWriter) emit(s string) error {
	if s == "" {
		return nil
	}
	n, err := io.WriteString(w.out, s)
	if err == nil && n < len(s) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = err
	}
	return err
}

// decidedEnd returns how much of a region can be highlighted now, given that
// more text may still arrive.
//
// Two rules compose, and the second is the one that is easy to miss.
//
// The tail that could still CHANGE must be held: the token touching the end may
// grow another letter, and a phrase containing it can begin up to maxWords-1
// tokens earlier. Text closed by punctuation or a newline cannot change and goes
// out immediately.
//
// But that hold point can fall INSIDE a phrase that is already complete — with
// `hot dog` in the deck, "one hot dog please" holds from `dog` (because `please`
// may grow), which would emit `hot` on its own and lose the match forever. So a
// known span straddling the hold point drags it back to that span's start, where
// it will be re-matched intact once more text arrives. Plain text may be cut
// freely: the token rule already guarantees nothing before it can join a future
// phrase.
func decidedEnd(region string, v Vocabulary, maxWords int) int {
	if maxWords <= 0 {
		return len(region) // nothing to match, so nothing to wait for
	}
	// BOTH release paths ask the same growth question. The first version asked it
	// on only one, and a region of pure punctuation ending in half a rune
	// ("!\xc3") has no tokens at all — so it took the no-token path, was
	// released, and "!über" arriving split lost its match while every byte
	// survived.
	toks := wordRuns(region)
	if len(toks) == 0 {
		// Nothing to hold against yet: release unless the tail could still grow
		// into a token, or is a gap a phrase could span.
		if tokenStillOpen(region) || phraseGapOrEmpty(region) {
			return 0
		}
		return len(region)
	}
	if !tokenStillOpen(region) && !phraseGapOrEmpty(region[toks[len(toks)-1].end:]) {
		return len(region) // punctuation closed the last token
	}
	k := len(toks) - maxWords
	if k < 0 {
		k = 0
	}
	hold := toks[k].start
	off := 0
	for _, sp := range highlightSpans(region, v) {
		end := off + len(sp.text)
		if sp.known && off < hold && end > hold {
			return off
		}
		off = end
	}
	return hold
}

// phraseGapOrEmpty is phraseGap plus the empty case. The matcher's phraseGap
// answers false for "" because wordRuns are maximal so tokens are never
// adjacent; the writer asks the same question about a region's TAIL, where empty
// means "the token runs to the end". One rule, two entry conditions — rather
// than a second near-identical predicate that could drift from it.
func phraseGapOrEmpty(s string) bool { return s == "" || phraseGap(s) }

// tokenStillOpen reports whether the region's last token could grow if more
// bytes arrive, in which case nothing may be released.
//
// Three ways it can be open, and the last two are why this exists — wordRuns
// TRIMS joiners off token edges, so a region ending in one has a trimmed token
// end followed by a character that is not a phrase gap, which reads exactly like
// "punctuation closed the token". It is the opposite: the token is mid-word.
//
//	"don'"  + "t"      -> don't      (a joiner at the end)
//	"hot-"  + "dog"    -> hot-dog
//	"caf\xc3" + "\xa9"  -> café       (an incomplete rune)
//
// Each of those lost its match before this check existed.
func tokenStillOpen(region string) bool {
	if region == "" {
		return false
	}
	if r, size := utf8.DecodeLastRuneInString(region); r == utf8.RuneError && size <= 1 {
		return true // an incomplete final rune: more bytes are coming
	}
	last, _ := utf8.DecodeLastRuneInString(region)
	return isWordRune(last)
}

// highlightRegion runs one complete region through the writer.
//
// The same mechanism the answer stream uses, which is the point: a definition
// region and an answer differ only in whether the text arrives at once, and
// giving them separate implementations would mean two sets of ANSI-resume rules
// to keep in agreement.
//
// base is what the terminal is in when the region starts, and therefore what
// each highlight must hand back — ANSI does not nest, so an example written
// inside p.ex would otherwise lose its style after the first highlighted word.
//
// Returns the input unchanged when there is nothing to highlight, so a caller
// with colour off never pays for a pass that cannot do anything.
func highlightRegion(s string, v Vocabulary, on, base string) string {
	if v == nil || on == "" || s == "" {
		return s
	}
	var b strings.Builder
	w := newHighlightWriter(&b, v, on)
	w.sgr.base = base
	if _, err := io.WriteString(w, s); err != nil {
		return s
	}
	if err := w.Flush(); err != nil {
		return s
	}
	return b.String()
}
