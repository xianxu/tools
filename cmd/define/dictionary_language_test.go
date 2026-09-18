package main

import (
	"path/filepath"
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

func TestDictionarySourceProvenanceCorpus(t *testing.T) {
	paths, err := filepath.Glob("testdata/bilingual/*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		if name == "manifest" {
			continue
		}
		for _, record := range bilingualFixture(t, name) {
			id, err := bilingualRecordIdentity(record.HTML)
			if err != nil || !id.spanish {
				continue
			}
			source := bilingualLanguageText(record)
			if source.text != record.Text || len(source.spans) == 0 {
				t.Errorf("%s %s lost all provenance", name, id.id)
			}
			record.Text = "unrelated " + record.Text
			if got := bilingualLanguageText(record); len(got.spans) != 0 {
				t.Errorf("%s trusted misaligned text", name)
			}
		}
	}
}

func TestDictionaryMonolingualOriginAndDisabledTint(t *testing.T) {
	// The per-fragment tint this test also asserted is gone (#70): production
	// tints whole sections, pinned by TestDefinitionOutputUniformSections.
	entry := ParseEntry("word | wərd | noun a spoken token: use a word. ORIGIN from German Wort.")
	out, _ := Render(entry, RenderOpts{Color: false, Language: "en"})
	if strings.Contains(out, "\x1b") {
		t.Fatal("no-color emitted ANSI")
	}
}

func TestDictionaryProjectionExactOccurrenceAndFallback(t *testing.T) {
	source := languageText{text: "red red", spans: []languageSpan{{start: 0, end: 3, lang: "es"}, {start: 4, end: 7, lang: "en"}}}
	if got := projectDictionaryText(source, "red green"); len(got.spans) != 0 || got.text != "red green" {
		t.Fatal("transformed text should stay intact and neutral")
	}
}

func TestDictionaryDefinitionsRetainSourceAndRegions(t *testing.T) {
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
	english, _ := renderDefinitions(set, RenderOpts{Color: true, Language: "es", Tint: tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}})
	assertDictionaryTint(t, english, "subir a la red", true)
	assertDictionaryTint(t, english, "to go up to", true)
	opt.Tint = tintPolicy{}
	untinted, baseline := renderDefinitions(set, opt)
	if !reflect.DeepEqual(regions, baseline) || stripEscapes(out) != stripEscapes(untinted) {
		t.Fatal("provenance changed region behavior")
	}
	for _, region := range regions {
		if region.Kind == RegionWord && region.Text == "net" {
			t.Fatal("supplement acquired a target-deck action")
		}
	}
}

func TestDictionaryParserSourceOffsets(t *testing.T) {
	for _, raw := range []string{
		"red noun 1 red: [label] : red. | red. 2 red: red.",
		"red | rɛd | noun 1 red: [label] : red. | red. verb [with object] | rɛd | 1 red: red.",
	} {
		entry := ParseEntry(raw)
		for _, block := range entry.Blocks {
			for _, sense := range block.Senses {
				if !sense.sourceKnown || raw[sense.sourceAt:sense.sourceAt+len(sense.Gloss)] != sense.Gloss {
					t.Fatalf("wrong gloss offset: %+v", sense)
				}
				for _, example := range sense.Examples {
					if !example.sourceKnown || raw[example.sourceAt:example.sourceAt+len(example.Text)] != example.Text {
						t.Fatalf("wrong example offset: %+v", example)
					}
				}
			}
		}
	}
	entry := ParseEntry("red noun (past red | rɛd |) a word")
	if entry.Blocks[0].Senses[0].sourceKnown {
		t.Fatal("rewritten source retained an unproven offset")
	}
}

func TestDictionaryProjectionDoesNotJoinWords(t *testing.T) {
	for _, pair := range [][2]string{{"an other", "another"}, {"another", "an other"}} {
		source := languageText{text: pair[0], spans: []languageSpan{{start: 0, end: len(pair[0]), lang: "es"}}}
		if got := projectDictionaryText(source, pair[1]); len(got.spans) > 0 {
			t.Fatalf("trusted changed word boundaries: %q → %q", pair[0], pair[1])
		}
	}
}

func TestDictionaryUnprovenSectionAlignmentStaysNeutral(t *testing.T) {
	source := languageText{text: "red noun other", spans: []languageSpan{{start: 0, end: 3, lang: "es"}}}
	set := definitionSet{sections: []definitionSection{{err: ErrNoEntry}, {entries: []string{"red noun changed"}, source: []languageText{source}}}}
	out, _ := renderDefinitions(set, RenderOpts{Color: true, Language: "es", Tint: tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}})
	if strings.Contains(out, languageDark) {
		t.Fatal("mismatched record text retained ownership")
	}
}

func TestDictionaryDuplicateRecordOwnershipIsDeterministic(t *testing.T) {
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
