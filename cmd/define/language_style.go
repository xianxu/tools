package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"strconv"
	"strings"
)

const (
	languageDark  = "\x1b[48;5;236m"
	languageLight = "\x1b[48;5;254m"
	languageOff   = "\x1b[49m"
)

// tintPolicy says whether a producer marks the target language's rows as
// tinted (#70): on is the -language-tint setting with colour on. The shade is
// not here — it resolves at paint — but scheme travels with the policy so a
// writer holding only the policy (the answer writer) can paint in it.
type tintPolicy struct {
	lang   store.Lang
	on     bool
	scheme *schemeHolder
}

// schemeTint is the shade a tinted row takes in a scheme: the background the
// terminal's theme cannot remap, so the one colour the scheme decides.
func schemeTint(s store.Scheme) string {
	if s == store.SchemeLight {
		return languageLight
	}
	return languageDark
}

// The ink paired with each tint, and its reset. A FIXED background needs a
// FIXED text colour: the terminal's default foreground is chosen for the
// terminal's own background, not for our tint, so on a mismatch (a light tint
// in a dark theme) default text was white on light grey. The mark pairs 24
// with 231 for the same reason.
const (
	inkOnLight = "\x1b[38;5;235m"
	inkOnDark  = "\x1b[38;5;252m"
	inkOff     = "\x1b[39m"
)

// schemeInk is the text colour for text with no colour of its own on a tinted
// row in scheme s: near-black on the light tint, near-white on the dark one.
func schemeInk(s store.Scheme) string {
	if s == store.SchemeLight {
		return inkOnLight
	}
	return inkOnDark
}

// Background state of the producer, excluding the tint we inject.
func sourceBackground(seq string, active bool) bool {
	bg, _ := sourceColours(seq, active, false)
	return bg
}

// sourceColours is the producer's colour state after seq — whether it has set a
// background, and whether it has set a foreground — excluding what we inject.
// ONE parse for both (#70), so they cannot disagree about what a sequence
// means. Extended colour payloads are skipped, so an RGB zero is never mistaken
// for a reset and the 36 inside 48;5;36 is never a foreground.
func sourceColours(seq string, bg, fg bool) (bool, bool) {
	if !isSGR(seq) {
		return bg, fg
	}
	params := strings.Split(seq[2:len(seq)-1], ";")
	for i := 0; i < len(params); i++ {
		part := strings.Split(params[i], ":")
		code := 0
		if part[0] != "" {
			var err error
			code, err = strconv.Atoi(part[0])
			if err != nil {
				continue
			}
		}
		switch {
		case code == 0:
			bg, fg = false, false
		case code == 49:
			bg = false
		case code == 39:
			fg = false
		case code == 48 || code >= 40 && code <= 47 || code >= 100 && code <= 107:
			bg = true
		case code == 38 || code >= 30 && code <= 37 || code >= 90 && code <= 97:
			fg = true
		}
		if len(part) == 1 && (code == 38 || code == 48 || code == 58) && i+1 < len(params) {
			switch params[i+1] {
			case "5":
				i += 2
			case "2":
				i += 4
			}
		}
	}
	return bg, fg
}

// tintFor is the tint policy for this session's language.
func tintFor(d deps, opt options) tintPolicy {
	return tintPolicy{lang: d.lang, on: opt.color && opt.tintOn, scheme: d.scheme}
}
