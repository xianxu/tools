package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzBlankStem — the tenth fuzz target in this repo, and the function that
// earns one.
//
// Its input is MODEL-AUTHORED text; its predecessor `blankOut` shipped a
// slice-bounds panic on `Ⱥ`/`İ` (folding runes change byte length, and an index
// computed on a folded string is invalid for the original); and a leak here is
// SILENT — a question that gives away its answer renders perfectly and grades
// perfectly.
//
// Three invariants, which are the three ways this can be wrong:
//   - it never panics;
//   - the output is valid UTF-8;
//   - no word-boundary occurrence of the answer survives.
func FuzzBlankStem(f *testing.F) {
	// Seeded from the leak table, plus the folding runes that broke the sibling.
	for _, s := range [][2]string{
		{"The aide was sycophantic to a fault.", "sycophantic"},
		{"Sycophantic aides surrounded him.", "sycophantic"},
		{"Shipwrights laid the keels of three frigates.", "keel"},
		{"The keel cracked; the keel was replaced.", "keel"},
		{"The keel's timbers rotted.", "keel"},
		{"Sunset over the set of the play.", "set"},
		{"They sold a hot-dog at the stand.", "hot"},
		{"İstanbul shipwrights laid the keel.", "keel"},
		{"The Ⱥ institute laid the keel.", "keel"},
		{"", ""},
		// THE HANG the fuzz found in eleven seconds: invalid UTF-8 decodes to
		// RuneError, which matches itself and is not a word rune, so the match
		// had no run and the loop never advanced. Seeded so it is a regression.
		{"\xc400000000000000000", "\xf1"},
		// The second find: an answer that is not a word at all — and `---`, the
		// same hole one predicate over, which passes an isWordRune test.
		{"_", "_"},
		{"a --- b", "---"},
	} {
		f.Add(s[0], s[1])
	}

	f.Fuzz(func(t *testing.T, stem, answer string) {
		got := blankStem(stem, answer)

		if !utf8.ValidString(stem) {
			return // garbage in; the invariants below are about real text
		}
		if !utf8.ValidString(got) {
			t.Fatalf("blankStem(%q, %q) produced invalid UTF-8: %q", stem, answer, got)
		}
		if strings.TrimSpace(answer) == "" {
			if got != stem {
				t.Fatalf("an empty answer changed the stem: %q -> %q", stem, got)
			}
			return
		}
		// THE INVARIANT IS ABOUT WORDS, and that is a narrowing rather than an
		// excuse. The fuzz found blankStem("_", "_") = "___" and called it a
		// leak — but `_` is not a word rune, so "does the answer occur as a WORD
		// in the output" is ill-defined for it, and the blank token is itself
		// made of the character in question. The real defect the case exposed is
		// upstream: usableItem now refuses an answer with no word characters,
		// because such an item is not a question. Pinned in TestUsableItem.
		if !hasLetterOrDigit(answer) {
			return
		}
		// THE LEAK, as a property: no word-boundary occurrence survives.
		if i, _ := wordIndexIn(got, answer); i >= 0 {
			t.Fatalf("blankStem(%q, %q) = %q — the answer survives at %d", stem, answer, got, i)
		}
	})
}
