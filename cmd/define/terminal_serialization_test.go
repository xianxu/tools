package main

import (
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/xianxu/tools/cmd/define/store"
)

// Compare source glyph order independently of physical wrapping or fill spaces.
func terminalSourceGlyphs(s string) string {
	var out strings.Builder
	for _, r := range stripEscapes(s) {
		if !unicode.IsSpace(r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func TestTerminalSerializationPreservesOverlongDictionarySource(t *testing.T) {
	const word = "anticonstitucionalmente"
	set := definitionSet{sections: []definitionSection{{language: "es", entries: []string{word + " adverb " + word + "."}}}}
	// Each scheme's tint, and no tint ("off" paints in dark, which it never shows).
	for _, profile := range []struct {
		name string
		on   bool
		sc   store.Scheme
	}{{"dark", true, store.SchemeDark}, {"light", true, store.SchemeLight}, {"off", false, store.SchemeDark}} {
		t.Run(profile.name, func(t *testing.T) {
			output := renderDefinitionOutput(set, RenderOpts{Color: true, Width: 20, Word: word, Tint: tintPolicy{lang: "es", on: profile.on, scheme: holderFor(profile.sc)}})
			got := serializeOutput(output, 20, profile.sc)
			wantGlyphs := terminalSourceGlyphs(output.text)
			if terminalSourceGlyphs(got) != wantGlyphs {
				t.Fatalf("serialized source lost glyphs: got %q, want %q", terminalSourceGlyphs(got), wantGlyphs)
			}
			if strings.Count(wantGlyphs, word) != 2 {
				t.Fatalf("fixture must exercise headword and body: %q", output.text)
			}
			for _, line := range strings.Split(got, "\n") {
				if visibleCells(line) > 20 {
					t.Fatalf("overlong physical row: %q", line)
				}
			}
			if serializeOutput(output, 0, profile.sc) != output.text {
				t.Fatal("zero-width plain source changed")
			}
		})
	}
}

func TestTerminalSerializationPreservesWideDisplayUnits(t *testing.T) {
	source := "\x1b[31m" + strings.Repeat("界", 11) + "e\u0301" + strings.Repeat("界", 9) + "\x1b[0m"
	// Each scheme's tint, and no tint ("off" paints in dark, which it never shows).
	for _, profile := range []struct {
		tinted bool
		sc     store.Scheme
	}{{true, store.SchemeDark}, {true, store.SchemeLight}, {false, store.SchemeDark}} {
		output := renderedOutput{text: source, rows: []rowPaint{{tinted: profile.tinted}}}
		got := serializeOutput(output, 20, profile.sc)
		if terminalSourceGlyphs(got) != terminalSourceGlyphs(source) {
			t.Fatalf("wide source lost: %q", got)
		}
		lines := strings.Split(got, "\n")
		if len(lines) != 3 {
			t.Fatalf("physical rows=%d want 3: %q", len(lines), got)
		}
		if strings.HasPrefix(stripEscapes(lines[2]), "\u0301") {
			t.Fatal("combining glyph split")
		}
		wantBackground := -1
		if profile.tinted && profile.sc == store.SchemeDark {
			wantBackground = 236
		} else if profile.tinted && profile.sc == store.SchemeLight {
			wantBackground = 254
		}
		for _, line := range lines {
			cells, end := rowTestCells(t, line, 20)
			if cells[0].fg != 31 {
				t.Error("foreground lost at hard wrap")
			}
			for col, c := range cells {
				if c.bg != wantBackground {
					t.Fatalf("physical row column %d background=%d want %d", col, c.bg, wantBackground)
				}
			}
			if profile.tinted && end.bg != -1 {
				t.Error("hard-wrapped row leaks background")
			}
		}
	}
}

func TestTerminalSerializationProjectsHardWrappedActionsAndExclusions(t *testing.T) {
	source := strings.Repeat("a", 19) + "界bcdef"
	output := renderedOutput{text: source, rows: []rowPaint{{tinted: true, exclusions: []cellRange{{19, 23}}}}, regions: []Region{{Line: 0, Col: 19, Width: 4, Text: "界bc"}}}
	got := layoutOutput(output, 20)
	if got.text != strings.Repeat("a", 19)+"\n界bcdef" {
		t.Fatalf("hard wrap=%q", got.text)
	}
	if len(got.rows) != 2 || !got.rows[1].tinted || !reflect.DeepEqual(got.rows[1].exclusions, []cellRange{{0, 4}}) {
		t.Fatalf("paint projection=%+v", got.rows)
	}
	if len(got.regions) != 1 || got.regions[0].Line != 1 || got.regions[0].Col != 0 || got.regions[0].Width != 4 {
		t.Fatalf("action projection=%+v", got.regions)
	}
	exact := layoutOutput(renderedOutput{text: strings.Repeat("a", 20), rows: []rowPaint{{tinted: true}}}, 20)
	if strings.Contains(exact.text, "\n") {
		t.Fatal("exact-width source gained extra row")
	}
}

func TestTerminalSerializationHardWrapKeepsSpaceCoordinates(t *testing.T) {
	source := strings.Repeat(" ", 21) + "hola"
	output := renderedOutput{text: source, rows: []rowPaint{{tinted: true, exclusions: []cellRange{{19, 22}}}}, regions: []Region{{Line: 0, Col: 19, Width: 3}}}
	got := layoutOutput(output, 20)
	if got.text != strings.Repeat(" ", 20)+"\n hola" {
		t.Fatalf("source whitespace changed: %q", got.text)
	}
	if !reflect.DeepEqual(got.rows[0].exclusions, []cellRange{{19, 20}}) || !reflect.DeepEqual(got.rows[1].exclusions, []cellRange{{0, 2}}) {
		t.Fatalf("source whitespace exclusions changed: %+v", got.rows)
	}
	if len(got.regions) != 2 || got.regions[0].Line != 0 || got.regions[0].Col != 19 || got.regions[0].Width != 1 || got.regions[1].Line != 1 || got.regions[1].Col != 0 || got.regions[1].Width != 2 {
		t.Fatalf("split action changed: %+v", got.regions)
	}
}

func TestTerminalSerializationPhysicalRowsKeepExplicitBoundaries(t *testing.T) {
	source := "\n" + strings.Repeat("a", 21) + "\n\n"
	want := []string{"", strings.Repeat("a", 20), "a", "", ""}
	if got := outputWrappedRows(source, 20); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows=%q want %q", got, want)
	}
	if got := strings.Join(outputWrappedRows(source, 0), "\n"); got != source {
		t.Fatal("plain source changed")
	}
	// Width one cannot represent a wide glyph, but it must never lose the source.
	output := renderedOutput{text: "界a", rows: []rowPaint{{tinted: true}}}
	if got := terminalSourceGlyphs(serializeOutput(output, 1, store.SchemeDark)); got != "界a" {
		t.Fatalf("narrow terminal loses glyph: %q", got)
	}
}
