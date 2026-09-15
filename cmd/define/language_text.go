package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"unicode/utf8"
)

// Language ranges address the exact text supplied by its producer. Missing
// ownership is unknown, never a guess based on the spelling of a word.
type languageSpan struct {
	start, end int
	lang       store.Lang
}
type languageText struct {
	text  string
	spans []languageSpan
}

// A malformed producer range disables decoration, never readable content.
// All walks are linear; offsets may surround ANSI but cannot split its bytes.
func validateLanguageText(t languageText) bool {
	if !utf8.ValidString(t.text) {
		return false
	}
	ends := make([]int, 0, 2*len(t.spans))
	last := 0
	for _, sp := range t.spans {
		if sp.start < last || sp.end < sp.start || sp.end > len(t.text) {
			return false
		}
		for _, at := range []int{sp.start, sp.end} {
			if at < len(t.text) && !utf8.RuneStart(t.text[at]) {
				return false
			}
			ends = append(ends, at)
		}
		last = sp.end
	}
	boundary := 0
	for i := 0; i < len(t.text); {
		n := escapeLen(t.text[i:])
		if n == 0 {
			_, n = utf8.DecodeRuneInString(t.text[i:])
		}
		for boundary < len(ends) && ends[boundary] <= i {
			boundary++
		}
		if boundary < len(ends) && ends[boundary] < i+n {
			return false
		}
		i += n
	}
	return true
}
