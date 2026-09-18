package main

import (
	"strconv"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// parseBackgroundColour reads an OSC 11 reply's payload (#70). Only the rgb:
// form with 1-4 hex digits per component is a colour; anything else (rgba:, #hex,
// garbage) is not an answer, and the caller swallows it without detecting.
//
// Classification is Rec. 601 luma on the gamma-ENCODED components, below 0.5
// dark — Neovim's background heuristic. Linear luminance would call a mid-grey
// terminal dark, which is not what its user calls it.
func parseBackgroundColour(payload string) (store.Scheme, bool) {
	rest, ok := strings.CutPrefix(payload, "rgb:")
	if !ok {
		return "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 3 {
		return "", false
	}
	var c [3]float64
	for i, p := range parts {
		if len(p) < 1 || len(p) > 4 {
			return "", false
		}
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil {
			return "", false
		}
		c[i] = float64(v) / float64(uint64(1)<<(4*len(p))-1)
	}
	if 0.299*c[0]+0.587*c[1]+0.114*c[2] < 0.5 {
		return store.SchemeDark, true
	}
	return store.SchemeLight, true
}
