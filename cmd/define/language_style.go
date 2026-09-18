package main

import (
	"fmt"
	"github.com/xianxu/tools/cmd/define/store"
	"strconv"
	"strings"
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
