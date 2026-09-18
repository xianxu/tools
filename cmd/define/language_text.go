package main

import "github.com/xianxu/tools/cmd/define/store"

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
