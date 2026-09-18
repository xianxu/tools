package main

import (
	"reflect"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// The projection carries ownership through a display transform, and it owes
// that in BOTH directions: a glyph the source did not write is never owned, and
// whitespace the layout generated inside a source word does not cost the word
// its ownership.
//
// The same-length substitution is the case that reaches the glyph comparison.
// A changed-length word or joined words are caught earlier by the length guards
// (`i >= len(source.text)` and the trailing leftover), so a projection that
// stopped comparing glyphs would still pass them. Measured 2026-09-18 (#76):
// with the comparison replaced by `false`, only "red bed" gains ownership.
func TestDisplayProjectionOwnsOnlyMatchingGlyphs(t *testing.T) {
	es, en := store.Lang("es"), store.Lang("en")
	whole := func(text string) languageText {
		return languageText{text: text, spans: []languageSpan{{start: 0, end: len(text), lang: es}}}
	}
	for _, tc := range []struct {
		name     string
		source   languageText
		rendered string
		want     []languageSpan
	}{
		{
			name:     "unchanged text keeps each fragment's own language",
			source:   languageText{text: "red red", spans: []languageSpan{{0, 3, es}, {4, 7, en}}},
			rendered: "red red",
			want:     []languageSpan{{0, 3, es}, {4, 7, en}},
		},
		{
			name:     "styling is skipped, not owned",
			source:   whole("red"),
			rendered: "\x1b[1mred\x1b[0m",
			want:     []languageSpan{{4, 7, es}},
		},
		{name: "same-length substitution", source: whole("red red"), rendered: "red bed"},
		{name: "changed-length word", source: whole("red red"), rendered: "red green"},
		{name: "joined words", source: whole("an other"), rendered: "another"},
		{
			name:     "inserted space inside a source word",
			source:   whole("another"),
			rendered: "an other",
			want:     []languageSpan{{0, 2, es}, {3, 8, es}},
		},
		{
			name:     "inserted newline inside a source word",
			source:   whole("another"),
			rendered: "an\nother",
			want:     []languageSpan{{0, 2, es}, {3, 8, es}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := projectDisplayText(tc.source, tc.rendered)
			if got.text != tc.rendered {
				t.Fatalf("text = %q, want the rendered %q unchanged", got.text, tc.rendered)
			}
			if len(got.spans) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got.spans, tc.want) {
				t.Fatalf("%q → %q: spans = %v, want %v", tc.source.text, tc.rendered, got.spans, tc.want)
			}
		})
	}
}

// For ANY source and rendered text, the projection returns the rendered text
// and spans that address it: in bounds, non-empty, ascending, non-overlapping.
// Nothing here asserts which glyphs are owned; that is the table test's job,
// and a property that re-derived it would only restate the implementation.
func FuzzDisplayProjection(f *testing.F) {
	for _, seed := range [][2]string{
		{"red red", "red red"},
		{"red", "\x1b[1mred\x1b[0m"},
		{"red red", "red bed"},
		{"red red", "red green"},
		{"an other", "another"},
		{"another", "an other"},
		{"another", "an\nother"},
	} {
		f.Add(seed[0], seed[1], uint8(3))
	}
	f.Fuzz(func(t *testing.T, sourceText, rendered string, cut uint8) {
		// Two owners meeting at an arbitrary byte, so the owner lookup is
		// exercised across a language boundary and not only inside one span.
		at := 0
		if len(sourceText) > 0 {
			at = int(cut) % (len(sourceText) + 1)
		}
		var spans []languageSpan
		if at > 0 {
			spans = append(spans, languageSpan{start: 0, end: at, lang: "es"})
		}
		if at < len(sourceText) {
			spans = append(spans, languageSpan{start: at, end: len(sourceText), lang: "en"})
		}
		got := projectDisplayText(languageText{text: sourceText, spans: spans}, rendered)
		if got.text != rendered {
			t.Fatalf("text = %q, want the rendered %q unchanged", got.text, rendered)
		}
		prevEnd := 0
		for i, s := range got.spans {
			if s.start < prevEnd || s.end <= s.start || s.end > len(rendered) {
				t.Fatalf("span %d of %v is not a non-empty, ascending range inside %d bytes", i, got.spans, len(rendered))
			}
			if s.lang == "" {
				t.Fatalf("span %d of %v has no language; unknown ownership is the absence of a span", i, got.spans)
			}
			prevEnd = s.end
		}
	})
}
