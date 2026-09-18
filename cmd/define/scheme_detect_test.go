package main

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestParseBackgroundColour(t *testing.T) {
	light, dark := store.SchemeLight, store.SchemeDark
	for payload, want := range map[string]store.Scheme{
		"rgb:ffff/ffff/ffff": light,
		"rgb:0000/0000/0000": dark,
		"rgb:1e1e/1e1e/1e1e": dark,  // a common dark theme
		"rgb:fdf6/e3e3/e3e3": light, // Solarized-light-ish
		"rgb:f/f/f":          light, // one hex digit per component
		// The boundary that separates Rec. 601 on ENCODED values (0.502 / 0.498)
		// from linear luminance, which would call both dark.
		"rgb:80/80/80": light,
		"rgb:7f/7f/7f": dark,
	} {
		if got, ok := parseBackgroundColour(payload); !ok || got != want {
			t.Errorf("parseBackgroundColour(%q) = %q, %v; want %q", payload, got, ok, want)
		}
	}
	for _, bad := range []string{"rgba:ffff/ffff/ffff/ffff", "#ffffff", "rgb:fffff/0/0", "rgb:ff/ff", "rgb:gg/00/00", "rgb:", "", "rgb:+f/0/0"} {
		if got, ok := parseBackgroundColour(bad); ok {
			t.Errorf("parseBackgroundColour(%q) = %q, want no colour", bad, got)
		}
	}
}
