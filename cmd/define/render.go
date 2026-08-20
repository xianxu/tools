package main

import (
	"fmt"
	"strings"
)

// RenderOpts controls presentation only. Render is pure: it never probes the
// terminal — the caller decides Color.
type RenderOpts struct {
	Color bool
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
				fmt.Fprintf(&b, "%s%s%s\n", indent, marker, prettyPronunciations(s.Gloss, p))
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
				fmt.Fprintf(&b, "%s\u201c%s\u201d%s\n", p.ex, prettyPronunciations(ex.Text, p), p.off)
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
				fmt.Fprintf(&b, "%s%s\n", indent, seg)
			}
		}
	}
	return b.String()
}

// prettyPronunciations is the coloured face of rewritePronunciations.
func prettyPronunciations(s string, p palette) string {
	return rewritePronunciations(s, p.ipa, p.off)
}
