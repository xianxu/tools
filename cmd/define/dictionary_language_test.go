package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func assertDictionaryTint(t *testing.T, out, needle string, want bool) {
	t.Helper()
	var plain strings.Builder
	var tint []bool
	active := false
	for i := 0; i < len(out); {
		if n := escapeLen(out[i:]); n > 0 {
			seq := out[i : i+n]
			if strings.Contains(seq, "48;5;") {
				active = true
			} else if seq == "\x1b[49m" || seq == "\x1b[0m" {
				active = false
			}
			i += n
			continue
		}
		plain.WriteByte(out[i])
		tint = append(tint, active)
		i++
	}
	at := strings.Index(plain.String(), needle)
	if at < 0 {
		t.Fatalf("missing %q in %q", needle, plain.String())
	}
	for i := at; i < at+len(needle); i++ {
		if plain.String()[i] == ' ' {
			continue
		}
		if tint[i] != want {
			t.Fatalf("%q byte %d tinted=%v want=%v", needle, i-at, tint[i], want)
		}
	}
}

func TestDictionaryMonolingualOriginAndDisabledTint(t *testing.T) {
	// The per-fragment tint this test also asserted is gone (#70): production
	// tints whole sections, pinned by TestDefinitionOutputUniformSections.
	entry := ParseEntry("word | wərd | noun a spoken token: use a word. ORIGIN from German Wort.")
	out, _ := Render(entry, RenderOpts{Color: false})
	if strings.Contains(out, "\x1b") {
		t.Fatal("no-color emitted ANSI")
	}
}

func TestDictionaryDefinitionsSectionTintKeepsRegions(t *testing.T) {
	records := &fakeRecordSource{installed: true, entries: map[string][]bilingualRecord{"red": bilingualFixture(t, "red")}}
	dict := spanishDefinitions{Dictionary: monolingualDictionary{Dictionary: &definitionFake{}, language: "es"}, english: records}
	set := definitionsFor(dict, "red", "red nombre femenino tejido", nil, true)
	vocab := &memVocabulary{}
	vocab.Add("red")
	vocab.Add("net")
	opt := RenderOpts{Color: true, Word: "red", Vocab: vocab, Tint: tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}}
	out, regions := renderDefinitions(set, opt)
	assertDictionaryTint(t, out, "subir a la red", false)
	assertDictionaryTint(t, out, "to go up to", false)
	english, _ := renderDefinitions(set, RenderOpts{Color: true, Tint: tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}})
	assertDictionaryTint(t, english, "subir a la red", true)
	assertDictionaryTint(t, english, "to go up to", true)
	opt.Tint = tintPolicy{}
	untinted, baseline := renderDefinitions(set, opt)
	if !reflect.DeepEqual(regions, baseline) || stripEscapes(out) != stripEscapes(untinted) {
		t.Fatal("tint changed region behavior")
	}
	for _, region := range regions {
		if region.Kind == RegionWord && region.Text == "net" {
			t.Fatal("supplement acquired a target-deck action")
		}
	}
}

func TestDictionaryDuplicateRecordSelectionIsDeterministic(t *testing.T) {
	selected, err := selectedSpanishRecords(bilingualFixture(t, "red"), "red", "")
	if err != nil {
		t.Fatal(err)
	}
	a := selected[0]
	b := a
	b.HTML = strings.Replace(b.HTML, `class="ex"`, `class="unverified"`, 1)
	first, err := selectedSpanishRecords([]bilingualRecord{a, b}, "red", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := selectedSpanishRecords([]bilingualRecord{b, a}, "red", "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("equal-text duplicate selected different ownership by order")
	}
}
