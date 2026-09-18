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

// schemeTint is the shade a tinted row takes in a scheme: the only colour the
// scheme decides, because it is the only one the terminal's theme cannot remap.
func schemeTint(s store.Scheme) string {
	if s == store.SchemeLight {
		return languageLight
	}
	return languageDark
}

// Background state of the producer, excluding the tint we inject. Skip extended
// foreground payloads so an RGB zero is never mistaken for an SGR reset.
func sourceBackground(seq string, active bool) bool {
	if !isSGR(seq) {
		return active
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
		case code == 0 || code == 49:
			active = false
		case code == 48 || code >= 40 && code <= 47 || code >= 100 && code <= 107:
			active = true
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
	return active
}

// tintFor is the tint policy for this session's language.
func tintFor(d deps, opt options) tintPolicy {
	return tintPolicy{lang: d.lang, on: opt.color && opt.tintOn, scheme: d.scheme}
}
