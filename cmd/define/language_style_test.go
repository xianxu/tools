package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"testing"
)

// sourceBackground is the producer-background tracker paintLanguageRow composes
// with. #70 moved these cases here when the per-fragment tint painter, its other
// consumer, was deleted. An explicit producer background must win over the tint,
// and a colour PAYLOAD must never be read as a code of its own.
func TestSourceBackgroundTracksTheProducersBackground(t *testing.T) {
	for _, tc := range []struct {
		name, seq string
		before    bool
		want      bool
	}{
		{"256-colour background", "\x1b[48;5;22m", false, true},
		{"basic background", "\x1b[42m", false, true},
		{"bright background", "\x1b[102m", false, true},
		{"reset", "\x1b[0m", true, false},
		{"default background", "\x1b[49m", true, false},
		{"combined reset and foreground", "\x1b[0;32m", true, false},
		{"an rgb zero is a payload, not a reset", "\x1b[38;2;0;0;0m", true, true},
		{"a 256-colour 48 is a payload, not a background", "\x1b[38;5;48m", false, false},
		{"not SGR", "\x1b[2J", true, true},
	} {
		if got := sourceBackground(tc.seq, tc.before); got != tc.want {
			t.Errorf("%s: sourceBackground(%q, %v) = %v, want %v", tc.name, tc.seq, tc.before, got, tc.want)
		}
	}
}

// Selecting across a tinted row copies its text, not its paint, and the
// selection's inverse still shows over the tint.
func TestSelectionCopiesATintedRowsText(t *testing.T) {
	const text = "A menudo means often."
	row := paintLanguageRow(text, rowPaint{tinted: true}, 30, store.SchemeDark)
	frame := newSelectionFrame(30, 1, []selectionRow{{styled: row, selectable: true}})
	got, err := selectedText(frame, selectionPoint{0, 0}, selectionPoint{0, 20})
	if err != nil || got != text {
		t.Fatalf("copy=%q err=%v", got, err)
	}
	selected := frame.highlightRow(0, selectionPoint{0, 0}, selectionPoint{0, 7})
	if !strings.Contains(selected, "\x1b[7m") || strings.TrimRight(stripANSI(selected), " ") != text {
		t.Fatalf("selection=%q", selected)
	}
}

// tintFor's colour gate is the ONLY gate on the paths with no second one:
// practice output and the answer writer read policy.on alone, so without it
// `define --play -no-color` would print tint escapes. (The lookup path checks
// colour again in renderDefinitionOutput, so the -no-color flag rows cannot
// see this gate go.)
func TestTintForGatesOnColour(t *testing.T) {
	for _, lang := range []store.Lang{"es", "en"} {
		h := holderFor(store.SchemeLight)
		d := deps{lang: lang, scheme: h}
		p := tintFor(d, options{color: true, tintOn: true})
		if p.lang != lang || !p.on || p.scheme != h {
			t.Errorf("colour on: policy=%+v", p)
		}
		if tintFor(d, options{color: false, tintOn: true}).on {
			t.Errorf("%s: no colour, yet the policy tints", lang)
		}
		if tintFor(d, options{color: true, tintOn: false}).on {
			t.Errorf("%s: -language-tint off, yet the policy tints", lang)
		}
	}
}
