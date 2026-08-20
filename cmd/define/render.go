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

	// Header: headword, syllabification, and the homograph number — the last of
	// these matters because only one homograph is reachable at all, so showing
	// "bank 1" makes the truncation visible instead of silent.
	fmt.Fprintf(&b, "%s%s%s", p.head, e.Headword, p.off)
	if e.Syllables != "" && e.Syllables != e.Headword {
		fmt.Fprintf(&b, "  %s%s%s", p.dim, e.Syllables, p.off)
	}
	if e.Homograph != "" {
		fmt.Fprintf(&b, " %s%s%s", p.dim, e.Homograph, p.off)
	}
	if e.HeadPOS != "" {
		fmt.Fprintf(&b, "  %s%s%s", p.pos, e.HeadPOS, p.off)
	}
	for _, x := range e.HeadExtra {
		fmt.Fprintf(&b, " %s", x)
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
			fmt.Fprintf(&hdr, " %s%s%s", p.dim, blk.Label, p.off)
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
				fmt.Fprintf(&b, "%s%s%s\n", indent, marker, s.Gloss)
			} else if marker != "" {
				fmt.Fprintf(&b, "%s%s\n", indent, strings.TrimSpace(marker))
			}
			for _, ex := range s.Examples {
				fmt.Fprintf(&b, "%s  %s%q%s\n", indent, p.ex, ex, p.off)
			}
		}
	}

	for _, sec := range e.Sections {
		fmt.Fprintf(&b, "\n  %s%s%s\n", p.sect, sec.Name, p.off)
		if sec.Text != "" {
			fmt.Fprintf(&b, "    %s\n", prettyPronunciations(sec.Text, p))
		}
	}
	return b.String()
}

// prettyPronunciations rewrites NOAD's |ˌsikəˈfan(t)ək(ə)lē| spans as /…/ so
// section text matches the header's notation. Only punctuation changes, so the
// no-data-loss invariant is unaffected. Spans that are prose (example
// separators) are left alone — that is exactly what isPronunciation decides.
func prettyPronunciations(s string, p palette) string {
	return pipeSpan.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSpace(strings.Trim(m, "|"))
		if !isPronunciation(inner) {
			return m
		}
		return fmt.Sprintf("%s/%s/%s", p.ipa, inner, p.off)
	})
}
