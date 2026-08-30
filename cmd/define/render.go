package main

import (
	"fmt"
	"strings"
	"unicode"
)

// RenderOpts controls presentation only. Render is pure: it never probes the
// terminal — the caller decides Color.
type RenderOpts struct {
	Color bool
	// Vocab highlights the words the learner knows. Nil means no highlighting.
	//
	// It reaches Render rather than wrapping Render's OUTPUT because the
	// decision is per REGION and only Render knows where its regions are.
	// Wrapping the whole string re-styled the headword — which the Spec puts out
	// of scope — because a finished string has no structure left to consult.
	// admitsHighlight below is the table.
	Vocab Vocabulary
	// Width is the terminal width used for word wrapping. 0 disables wrapping,
	// which is what a pipe wants — a consumer re-wraps for itself, and hard
	// breaks baked into piped output cannot be undone.
	Width int
}

// ANSI codes, empty when colour is off so the same format strings serve both.
type palette struct{ head, ipa, pos, num, ex, sect, dim, off string }

func newPalette(on bool) palette {
	if !on {
		return palette{}
	}
	return palette{
		head: "\x1b[1;36m", ipa: "\x1b[35m", pos: "\x1b[1;33m",
		num: "\x1b[1m", ex: "\x1b[3;32m", sect: "\x1b[1;34m",
		dim: "\x1b[2m", off: "\x1b[0m",
	}
}

// admitsHighlight is the per-region decision, written out so it can be read and
// tested rather than inferred from where a call happens to be.
//
// The rule: highlighting marks vocabulary in PROSE. A label is not prose — it is
// the dictionary's own scaffolding, and colouring "adjective" because the
// learner once looked it up says nothing about the word being defined.
//
// The table has to be COMPLETE to be a decision procedure. Its first version
// omitted HeadHomograph, HeadOther and the block label — all withheld by
// construction, so behaviour was right and the enumeration was a subset
// pretending to be the whole.
//
// TestHighlightsAppearOnlyInAdmittedRegions is what makes an omission FAIL: it
// derives the admitted text from the parsed Entry, so a region that starts
// leaking is caught without anyone remembering to add a row here. That test —
// not this comment — is the record. Deliberately no COUNT is written beside the
// rows: a number in prose next to an enumeration is a second source of truth
// nothing checks, and it drifted twice in two rounds (once when the correction
// also split two rows and the arithmetic was not redone).
//
//	region                     decision
//	------------------------   --------
//	headword (HeadWord)        withhold — Spec, out of scope; already bold cyan
//	homograph (HeadHomograph)  withhold — an index number
//	syllabification            withhold — a label
//	POS in the head (HeadPOS)  withhold — a label
//	other head tokens          withhold — carries prose ("read verb (past and
//	  (HeadOther)                         past participle read | red |)") but it
//	                                      is head material, styled as the head
//	entry IPA                  withhold — not prose
//	block POS label            withhold — a label
//	block grammar label        withhold — a label
//	block IPA                  withhold — not prose
//	sense number marker        withhold — scaffolding
//	sense gloss                ADMIT    — prose
//	example label              withhold — a label
//	example text               ADMIT    — prose
//	section name               withhold — a label
//	section text               ADMIT    — prose
//
// -raw is withheld earlier still, at the print site: it is by contract the
// unparsed entry.
func (o RenderOpts) admitsHighlight() bool { return o.Color && o.Vocab != nil }

// prose highlights one admitted region, resuming the style that encloses it.
//
// base is what the terminal is in when the region starts, because ANSI does not
// nest: an example is written inside p.ex, so a highlight closing with a reset
// would leave the rest of the example unstyled.
func (o RenderOpts) prose(s, base string) string {
	if !o.admitsHighlight() {
		return s
	}
	return highlightRegion(s, o.Vocab, knownOn, base)
}

// Render turns an Entry into the printed block.
//
// CORRECTNESS CONSTRAINT, not a style preference: Render must not change case,
// abbreviate, truncate, or reorder. The no-data-loss invariant
// (invariant_test.go) holds the rendered letters and digits against the raw
// entry as an ordered subsequence, and any of those would break it.
func Render(e Entry, opt RenderOpts) string {
	p := newPalette(opt.Color)
	var b strings.Builder

	// Header: walk the head tokens in SOURCE order. NOAD has no fixed field
	// order — "present 1 pres·ent" puts the homograph first, "record rec·ordnoun"
	// welds a POS on, "read verb (past and past participle read | red |)" carries
	// a whole parenthetical — so emitting fields in a guessed order reorders the
	// entry. Nothing is hidden here, including a syllabification equal to the
	// headword: suppression is how content goes missing.
	for i, t := range e.Head {
		sep := " "
		switch {
		case i == 0:
			sep = ""
		case t.Kind == HeadSyllables || t.Kind == HeadPOS:
			sep = "  "
		}
		var color string
		switch t.Kind {
		case HeadWord:
			color = p.head
		case HeadSyllables, HeadHomograph:
			color = p.dim
		case HeadPOS:
			color = p.pos
		}
		if color != "" {
			fmt.Fprintf(&b, "%s%s%s%s", sep, color, t.Text, p.off)
		} else {
			fmt.Fprintf(&b, "%s%s", sep, t.Text)
		}
	}
	b.WriteString("\n")
	if e.IPA != "" {
		fmt.Fprintf(&b, "%s/%s/%s\n", p.ipa, e.IPA, p.off)
	}

	for _, blk := range e.Blocks {
		b.WriteString("\n")
		// The block header is omitted entirely when the POS already appeared in
		// the entry header (record) and there is nothing else to show — an empty
		// indented line reads as a formatting bug.
		var hdr strings.Builder
		if blk.POS != "" && !blk.FromHead {
			fmt.Fprintf(&hdr, "%s%s%s", p.pos, blk.POS, p.off)
		}
		if blk.Label != "" {
			fmt.Fprintf(&hdr, " %s%s%s", p.dim, prettyPronunciations(blk.Label, p), p.off)
		}
		if blk.IPA != "" {
			fmt.Fprintf(&hdr, "  %s/%s/%s", p.ipa, blk.IPA, p.off)
		}
		if h := strings.TrimSpace(hdr.String()); h != "" {
			fmt.Fprintf(&b, "  %s\n", h)
		}

		for _, s := range blk.Senses {
			indent := "    "
			marker := ""
			switch {
			case s.Number != "":
				marker = fmt.Sprintf("%s%s.%s ", p.num, s.Number, p.off)
			case s.Sub:
				indent = "      "
				marker = "• "
			}
			if s.Gloss != "" {
				body := prettyPronunciations(s.Gloss, p)
				lead := len(indent) + visibleCells(marker)
				// Highlight AFTER wrapping: wrapText measures visible columns and
				// breaks at spaces, so a highlight inserted first would widen the
				// text it measures. Wrapping first also means a phrase cannot span
				// a line break, which is the writer's rule 2 falling out rather
				// than being enforced twice.
				fmt.Fprintf(&b, "%s%s%s\n", indent, marker, opt.prose(wrapText(body, opt.Width, lead), ""))
			} else if marker != "" {
				fmt.Fprintf(&b, "%s%s\n", indent, strings.TrimSpace(marker))
			}
			for _, ex := range s.Examples {
				fmt.Fprintf(&b, "%s  ", indent)
				if ex.Label != "" {
					fmt.Fprintf(&b, "%s%s%s ", p.dim, ex.Label, p.off)
				}
				// Quote explicitly, never with %q: strconv.Quote escapes '"' and
				// every rune failing unicode.IsPrint, so NOAD's quoted speech
				// arrived as literal backslashes and a soft hyphen (U+00AD)
				// became five alphanumeric runes the dictionary never returned.
				// Render must not insert content any more than it may drop it.
				// Typographic outer quotes so an example containing quoted speech
				// stays readable: “"This blows," she sighs” rather than
				// ""This blows," she sighs".
				fmt.Fprintf(&b, "%s\u201c%s\u201d%s\n", p.ex,
					opt.prose(wrapText(prettyPronunciations(ex.Text, p), opt.Width, len(indent)+2), p.ex), p.off)
			}
		}
	}

	for _, sec := range e.Sections {
		fmt.Fprintf(&b, "\n  %s%s%s\n", p.sect, sec.Name, p.off)
		if sec.Text != "" {
			// Section text carries the same example separators sense text does
			// ("…played some great football into the bargain | save yourself
			// money…"). Rendering it as one paragraph left NOAD's raw "|" on
			// screen; each segment gets its own line instead.
			for i, seg := range strings.Split(prettyPronunciations(sec.Text, p), "|") {
				if seg = strings.TrimSpace(seg); seg == "" {
					continue
				}
				indent := "    "
				if i > 0 {
					indent = "      "
				}
				fmt.Fprintf(&b, "%s%s\n", indent, opt.prose(wrapText(seg, opt.Width, len(indent)), ""))
			}
		}
	}
	return b.String()
}

// prettyPronunciations is the coloured face of rewritePronunciations.
func prettyPronunciations(s string, p palette) string {
	return rewritePronunciations(s, p.ipa, p.off)
}

// visibleCells is the width of s in TERMINAL COLUMNS, ignoring ANSI escape
// sequences. Wrapping on raw byte length would break early on any coloured line,
// and these lines are coloured; counting runes instead is wrong in both
// directions for a dictionary — see cellWidth.
//
// It is the ONE owner of "how wide is this" in this program. Every wrap, every
// frame budget and every clip reads it, so a line cannot be measured one way
// where it is written and another where it is placed.
func visibleCells(s string) int {
	n, inEsc, inCSI := 0, false, false
	for _, r := range s {
		switch {
		case inCSI:
			// ANY final byte ends a CSI, not just "m". Render emits only SGR, so
			// "m" was enough while this only measured rendered text — but #30's
			// screen measures lines that also carry cursor moves (`\x1b[4D`
			// parks the cursor after a suggestion), and a machine that waits for
			// "m" counts the whole rest of such a line as invisible. A frame
			// budgeted from that measurement is a frame that does not fit.
			if r >= 0x40 && r <= 0x7e {
				inCSI = false
			}
		case inEsc:
			inEsc = false
			inCSI = r == '['
		case r == '\x1b':
			inEsc = true
		default:
			n += cellWidth(r)
		}
	}
	return n
}

// cellWidth is how many terminal columns one rune occupies: 0, 1 or 2.
//
// It exists because this program is a DICTIONARY, and both exceptions are its
// daily traffic. NOAD writes `bänˈZHo͝or` — the o͝o carries a combining double
// breve, a rune that occupies no column of its own — so counting runes reports
// ten columns for nine and cuts text that fits. And a Japanese or Chinese entry
// is full-width: ten runes are twenty columns, so counting runes builds a frame
// twice as tall as it measured, the terminal scrolls, and every row the app
// believes it placed has moved.
//
// Hand-rolled against unicode's own tables rather than taking a dependency for
// it: the ranges below are the East Asian Wide and Fullwidth blocks, and the
// zero-width cases are exactly the categories unicode already names.
func cellWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
		// Non-spacing and enclosing marks compose with the rune before them, and
		// format characters (ZWJ, the bidi controls) are not drawn at all.
		return 0
	case r == '\u200b': // zero-width space, which is Zs rather than Cf
		return 0
	case isWide(r):
		return 2
	}
	return 1
}

// isWide reports the East Asian Wide and Fullwidth ranges — the ones a terminal
// draws in two cells.
func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115f, // Hangul Jamo
		r >= 0x2e80 && r <= 0x303e, // CJK radicals, Kangxi, CJK symbols
		r >= 0x3041 && r <= 0x33ff, // Hiragana, Katakana, Hangul compat, CJK compat
		r >= 0x3400 && r <= 0x4dbf, // CJK Ext A
		r >= 0x4e00 && r <= 0x9fff, // CJK Unified
		r >= 0xa000 && r <= 0xa4cf, // Yi
		r >= 0xac00 && r <= 0xd7a3, // Hangul syllables
		r >= 0xf900 && r <= 0xfaff, // CJK compatibility ideographs
		r >= 0xfe30 && r <= 0xfe6f, // CJK compatibility forms
		r >= 0xff00 && r <= 0xff60, // fullwidth forms
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x1f300 && r <= 0x1f64f, // emoji, which terminals draw double-wide
		r >= 0x1f900 && r <= 0x1f9ff,
		r >= 0x20000 && r <= 0x3fffd: // CJK Ext B and beyond
		return true
	}
	return false
}

// wrapText breaks s at spaces to fit width, continuing on subsequent lines with
// `indent` spaces. Returns s unchanged when width is 0.
//
// The terminal will wrap for us if we do not, but it wraps at the column —
// splitting words mid-syllable ("fing/ers"), which is exactly what a dictionary
// entry must not do.
func wrapText(s string, width, indent int) string {
	if width <= 0 || visibleCells(s)+indent <= width {
		return s
	}
	pad := strings.Repeat(" ", indent)
	var b strings.Builder
	col := indent
	for i, word := range strings.Fields(s) {
		w := visibleCells(word)
		switch {
		case i == 0:
			b.WriteString(word)
			col += w
		case col+1+w <= width:
			b.WriteString(" " + word)
			col += 1 + w
		default:
			// A single word longer than the line still goes on its own line
			// rather than being cut: better to overflow than to split it.
			b.WriteString("\n" + pad + word)
			col = indent + w
		}
	}
	return b.String()
}
