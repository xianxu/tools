package store

import (
	"fmt"
	"strings"
)

// Lang is a validated language tag — the name of a deck, the language of a
// recording, and the language of a dictionary.
//
// A named type rather than a bare string because it becomes a PATH SEGMENT:
// words/<lang>/. An unvalidated string there is a directory traversal, which is
// why ParseLang is the only way to make one from input.
type Lang string

// DefaultLang is what a directory with no setting means.
//
// English, and the reason is a fact about the FILES rather than a preference: a
// deck written before this existed was filed by a tool that only ever consulted
// the English dictionary and requested _en_us_ recordings. Whatever its
// headwords are, its entries are English.
const DefaultLang Lang = "en"

// ParseLang validates and normalises a language tag.
//
// Deliberately NOT checked against a list of known languages: the CDN and the
// installed dictionaries own what exists, and a hardcoded list here would be a
// restatement of a fact they own — one that goes stale the day a dictionary is
// installed. An unknown-but-well-formed tag degrades to "no recording, no
// dictionary", which is honest and is what a learner sees anyway.
func ParseLang(s string) (Lang, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", fmt.Errorf("empty language")
	}
	if len(s) != 2 || !isASCIILower(s[0]) || !isASCIILower(s[1]) {
		return "", fmt.Errorf("not a language tag: %q (want two letters, like en or es)", s)
	}
	return Lang(s), nil
}

func isASCIILower(b byte) bool { return b >= 'a' && b <= 'z' }
