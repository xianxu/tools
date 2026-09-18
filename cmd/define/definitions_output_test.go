package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func capturedDefinitionSet(t *testing.T) definitionSet {
	t.Helper()
	primary, err := testDictFor(t, "es").Lookup("mesa")
	if err != nil {
		t.Fatal(err)
	}
	dict := spanishDefinitions{Dictionary: monolingualDictionary{Dictionary: &definitionFake{}, language: "es"}, english: &fakeRecordSource{installed: true, entries: map[string][]bilingualRecord{"mesa": bilingualFixture(t, "mesa")}}}
	return definitionsFor(dict, "mesa", primary, nil, true)
}

// Check emitted cells, independently of the row metadata that requested paint.
func assertDefinitionSectionCells(t *testing.T, set definitionSet, lang store.Lang, sc store.Scheme, width int) renderedOutput {
	t.Helper()
	opt := RenderOpts{Word: entryIdentity(ParseEntry(set.sections[0].entries[0])), Color: true, Width: width, Tint: tintPolicy{lang: lang, on: true, scheme: holderFor(sc)}}
	output := renderDefinitionOutput(set, opt)
	serialized := serializeOutput(output, width, sc)
	lines := strings.Split(serialized, "\n")
	plain := strings.Split(stripEscapes(serialized), "\n")
	second := -1
	for i, line := range plain {
		if strings.HasPrefix(line, "English") {
			second = i
			break
		}
	}
	if second < 1 {
		t.Fatalf("missing second section heading: %q", plain)
	}
	// The separator is inserted as the opening row of the second section.
	second--
	wantTint := 236
	if sc == store.SchemeLight {
		wantTint = 254
	}
	for i, line := range lines[:len(lines)-1] {
		want := -1
		if i < second && lang == "es" || i >= second && lang == "en" {
			want = wantTint
		}
		cells, end := rowTestCells(t, line, width)
		for col, c := range cells {
			if c.bg != want {
				t.Fatalf("%s width=%d row=%d col=%d bg=%d want %d text=%q", lang, width, i, col, c.bg, want, plain[i])
			}
		}
		if end.bg != -1 {
			t.Fatalf("row %d leaks background to newline", i)
		}
	}
	if strings.Contains(output.text, schemeTint(sc)) {
		t.Fatal("source text contains paint")
	}
	if serializeOutput(output, 0, sc) != output.text {
		t.Fatal("pipe adds paint/padding")
	}
	return output
}

func TestDefinitionOutputUniformSections(t *testing.T) {
	set := capturedDefinitionSet(t)
	for _, sc := range []store.Scheme{store.SchemeDark, store.SchemeLight} {
		for _, lang := range []store.Lang{"es", "en"} {
			for _, width := range []int{32, 80} {
				output := assertDefinitionSectionCells(t, set, lang, sc, width)
				if len(output.regions) < 2 {
					t.Fatal("section headword actions missing")
				}
				heads := 0
				for _, r := range output.regions {
					if r.Kind == RegionHeadword {
						heads++
						if r.Word != "mesa" || r.Text != "mesa" || r.Col != 0 || r.Width != 4 {
							t.Fatalf("wrong headword action: %+v", r)
						}
					}
				}
				if heads != 2 {
					t.Fatalf("headword actions=%d want 2", heads)
				}
				plain := strings.Split(stripEscapes(output.text), "\n")
				for _, region := range output.regions {
					if region.Line >= len(plain) || region.Col < 0 || region.Col+region.Width > visibleCells(plain[region.Line]) {
						t.Fatalf("invalid action %+v", region)
					}
				}
				untinted := renderDefinitionOutput(set, RenderOpts{Word: "mesa", Color: true, Width: width})
				if !reflect.DeepEqual(output.regions, untinted.regions) || output.text != untinted.text {
					t.Fatal("paint changed text/actions")
				}
			}
		}
	}
}

func TestDefinitionOutputFormattingFallbackPreservesPrimary(t *testing.T) {
	set := capturedDefinitionSet(t)
	record := bilingualFixture(t, "mesa")[0]
	// Retain a valid native identity but corrupt HTML-to-Text correspondence.
	record.Text = "unmatched source " + record.Text
	provider := spanishDefinitions{english: &fakeRecordSource{installed: true, entries: map[string][]bilingualRecord{"mesa": {record}}}}
	set.sections[1] = provider.supplement("mesa", set.sections[0].entries[0])
	if set.sections[1].formatErr == nil {
		t.Fatal("missing structural failure")
	}
	output := renderDefinitionOutput(set, RenderOpts{Word: "mesa", Color: true, Width: 80, Tint: tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}})
	plain := stripEscapes(output.text)
	if !strings.Contains(oxfordContent(plain), oxfordContent(record.Text)) {
		t.Fatal("fallback dropped native source content")
	}
	if !strings.Contains(plain, "unmatched source") || !strings.Contains(plain, "Oxford formatting unavailable") {
		t.Fatal("readable source fallback or diagnostic lost")
	}
	if !strings.Contains(plain, "Spanish") || len(output.regions) == 0 {
		t.Fatal("primary lost on supplement formatting failure")
	}
	for _, row := range output.rows {
		if row.tinted {
			t.Fatal("unproven fallback acquired tint")
		}
	}
}

func TestDefinitionOutputUnknownPrimaryIsNeutral(t *testing.T) {
	set := capturedDefinitionSet(t)
	set.sections[0].language = ""
	output := renderDefinitionOutput(set, RenderOpts{Color: true, Width: 80, Tint: tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}})
	for _, row := range output.rows {
		if row.tinted {
			t.Fatal("display label established ownership for unverified source")
		}
	}
}

func TestDefinitionOutputDisabledTintKeepsCleanLayout(t *testing.T) {
	set := capturedDefinitionSet(t)
	for _, opt := range []RenderOpts{{Width: 32, Tint: tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}}, {Color: true, Width: 32, Tint: tintPolicy{lang: "es"}}} {
		output := renderDefinitionOutput(set, opt)
		for _, row := range output.rows {
			if row.tinted {
				t.Fatal("disabled tint retained paint")
			}
		}
		rendered := serializeOutput(output, 32, opt.Tint.scheme.Scheme())
		if strings.Contains(rendered, languageDark) || strings.Contains(rendered, languageLight) {
			t.Fatal("disabled tint emitted background")
		}
		if !opt.Color && strings.Contains(rendered, "\x1b") {
			t.Fatal("no-color output contains escapes")
		}
		if stripEscapes(rendered) != stripEscapes(wrapWritten(output.text, 32)) {
			t.Fatal("disabled tint changed clean layout")
		}
	}
}
