package store

import (
	"fmt"
	"os"
	"path/filepath"
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

// langFile is where a directory's language lives.
//
// Named through RuntimeFiles rather than spelled here, so the guards that keep
// it out of git and the code that writes it cannot disagree about the name.
func langFile(dir string) string { return filepath.Join(dir, RuntimeFiles[1]) }

// ReadLang returns the directory's language, or DefaultLang when there is none.
//
// Absent, empty, unreadable and malformed all mean the default. That is a
// deliberate flattening: the learner typed a word, not a request for a
// configuration audit, and failing a lookup because a one-line settings file has
// a typo in it would be the tool inventing a way to be useless. The cost is that
// a typo is silent — which is why /lang REPORTS the current language, so the
// answer is one command away rather than a mystery.
func ReadLang(dir string) Lang {
	b, err := os.ReadFile(langFile(dir))
	if err != nil {
		return DefaultLang
	}
	l, err := ParseLang(string(b))
	if err != nil {
		return DefaultLang
	}
	return l
}

// WriteLang persists the directory's language.
//
// It validates rather than trusting its caller: this value becomes a path
// segment on the NEXT run, and a store that read it back would have no way to
// tell it was written rather than typed.
func WriteLang(dir string, l Lang) error {
	valid, err := ParseLang(string(l))
	if err != nil {
		return err
	}
	return writeBytesAtomic(langFile(dir), []byte(string(valid)+"\n"))
}
