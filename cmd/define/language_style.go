package main

import (
	"fmt"
	"github.com/xianxu/tools/cmd/define/store"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	languageDark  = "\x1b[48;5;236m"
	languageLight = "\x1b[48;5;254m"
	languageOff   = "\x1b[49m"
)

type tintPolicy struct {
	lang       store.Lang
	background string
}

// styleLanguageText changes background only. Trusted source styles pass through;
// explicit backgrounds take precedence until their own reset. Unknown regions
// pass through unchanged, including already-rendered bilingual definitions.
func styleLanguageText(t languageText, p tintPolicy) string {
	if p.background != languageDark && p.background != languageLight || !validateLanguageText(t) {
		return t.text
	}
	if p.lang == "" {
		p.lang = store.DefaultLang
	}
	target, err := store.ParseLang(string(p.lang))
	if err != nil {
		return t.text
	}
	spans := make([]languageSpan, 0, len(t.spans))
	for _, sp := range t.spans {
		lang, err := store.ParseLang(string(sp.lang))
		if err == nil && lang == target && sp.start < sp.end {
			spans = append(spans, sp)
		}
	}
	if len(spans) == 0 {
		return t.text
	}
	var out strings.Builder
	open, explicit := false, false
	closeTint := func() {
		if open {
			out.WriteString(languageOff)
			open = false
		}
	}
	first, last := lineInkBounds(t.text, 0)
	at := 0
	for i := 0; i < len(t.text); {
		if n := escapeLen(t.text[i:]); n > 0 {
			seq := t.text[i : i+n]
			closeTint()
			out.WriteString(seq)
			explicit = sourceBackground(seq, explicit)
			i += n
			continue
		}
		n, _ := nextDisplayUnit(t.text[i:])
		for at < len(spans) && spans[at].end <= i {
			at++
		}
		wanted := !explicit && i >= first && i < last && at < len(spans) && spans[at].start <= i && spans[at].end >= i+n
		if wanted && !open {
			out.WriteString(p.background)
			open = true
		}
		if !wanted {
			closeTint()
		}
		out.WriteString(t.text[i : i+n])
		if t.text[i] == '\n' {
			first, last = lineInkBounds(t.text, i+n)
		}
		i += n
	}
	closeTint()
	return out.String()
}

// Ink bounds exclude line indentation and padding, even when ANSI surrounds it.
func lineInkBounds(text string, start int) (first, last int) {
	first, last = len(text), start
	for i := start; i < len(text) && text[i] != '\n'; {
		if n := escapeLen(text[i:]); n > 0 {
			i += n
			continue
		}
		r, n := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsSpace(r) {
			if first == len(text) {
				first = i
			}
			last = i + n
		}
		i += n
	}
	return
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

func tintProfile(name string) (string, error) {
	switch name {
	case "dark":
		return languageDark, nil
	case "light":
		return languageLight, nil
	case "off":
		return "", nil
	default:
		return "", fmt.Errorf("invalid language tint %q: use dark, light, or off", name)
	}
}
func (o options) tintFor(lang store.Lang) tintPolicy {
	background := o.tintBackground
	if !o.color {
		background = ""
	}
	return tintPolicy{lang, background}
}
