package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestDictionaryCapturedMixedOwnership(t *testing.T) {
	selected, err := selectedSpanishRecords(bilingualFixture(t, "red"), "red", "")
	if err != nil {
		t.Fatal(err)
	}
	entry := ParseEntry(selected[0].Text)
	entry.source = bilingualLanguageText(selected[0])
	for _, lang := range []store.Lang{"es", "en"} {
		opt := RenderOpts{Color: true, Language: "es", Tint: tintPolicy{lang: lang, background: "\x1b[48;5;236m"}}
		out, regions := Render(entry, opt)
		for _, tc := range []struct {
			text string
			lang store.Lang
		}{{"subir a la red", "es"}, {"to go up to", "en"}, {"caer en las redes de alguien", "es"}, {"to fall into somebody's clutches", "en"}} {
			assertDictionaryTint(t, out, tc.text, lang == tc.lang)
		}
		opt.Tint = tintPolicy{}
		plain, rs := Render(entry, opt)
		if stripEscapes(out) != stripEscapes(plain) || !reflect.DeepEqual(regions, rs) {
			t.Fatal("tint changed prose or region ownership")
		}
	}
}

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
			entry := ParseEntry(record.Text)
			entry.source = source
			out, _ := Render(entry, RenderOpts{Color: true, Tint: tintPolicy{lang: "es", background: "\x1b[48;5;236m"}})
			if !strings.Contains(out, "\x1b[48;5;236m") {
				t.Errorf("%s %s no rendered Spanish", name, id.id)
			}
			record.Text = "unrelated " + record.Text
			if got := bilingualLanguageText(record); len(got.spans) != 0 {
				t.Errorf("%s trusted misaligned text", name)
			}
		}
	}
}

func TestDictionaryMonolingualOriginAndDisabledTint(t *testing.T) {
	entry := ParseEntry("word | wərd | noun a spoken token: use a word. ORIGIN from German Wort.")
	opt := RenderOpts{Color: true, Language: "en", Tint: tintPolicy{lang: "en", background: "\x1b[48;5;236m"}}
	out, _ := Render(entry, opt)
	for _, word := range []string{"word", "a spoken token", "use a word"} {
		assertDictionaryTint(t, out, word, true)
	}
	assertDictionaryTint(t, out, "from German Wort", false)
	assertDictionaryTint(t, out, "wərd", false)
	opt.Color = false
	out, _ = Render(entry, opt)
	if strings.Contains(out, "\x1b") {
		t.Fatal("no-color emitted ANSI")
	}
}

func TestDictionaryProjectionExactOccurrenceAndFallback(t *testing.T) {
	source := languageText{text: "red red", spans: []languageSpan{{start: 0, end: 3, lang: "es"}, {start: 4, end: 7, lang: "en"}}}
	for _, tc := range []struct {
		at   int
		lang store.Lang
	}{{0, "es"}, {4, "en"}} {
		got := dictionaryFragment(source, tc.at, "red")
		if len(got.spans) != 1 || got.spans[0].lang != tc.lang {
			t.Fatalf("offset %d: %+v", tc.at, got)
		}
	}
	if got := dictionaryFragment(source, 1, "red"); len(got.spans) != 0 {
		t.Fatal("searched for a later spelling")
	}
	if got := projectDictionaryText(source, "red green"); len(got.spans) != 0 || got.text != "red green" {
		t.Fatal("transformed text should stay intact and neutral")
	}
}

func TestDictionaryDefinitionsRetainSourceAndRegions(t *testing.T) {
	records := &fakeRecordSource{installed: true, entries: map[string][]bilingualRecord{"red": bilingualFixture(t, "red")}}
	dict := spanishDefinitions{Dictionary: &definitionFake{}, english: records}
	set := definitionsFor(dict, "red", "red nombre femenino tejido", nil, true)
	vocab := &memVocabulary{}
	vocab.Add("red")
	vocab.Add("net")
	opt := RenderOpts{Color: true, Word: "red", Language: "es", Vocab: vocab, Tint: tintPolicy{lang: "es", background: languageDark}}
	out, regions := renderDefinitions(set, opt)
	assertDictionaryTint(t, out, "subir a la red", true)
	assertDictionaryTint(t, out, "to go up to", false)
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
	out, _ := renderDefinitions(set, RenderOpts{Color: true, Language: "es", Tint: tintPolicy{lang: "es", background: languageDark}})
	if strings.Contains(out, languageDark) {
		t.Fatal("mismatched record text retained ownership")
	}
}

func TestDictionaryInlinePronunciationRemainsNeutral(t *testing.T) {
	entry := ParseEntry("red noun (past red | rɛd |) a word DERIVATIVES redden | ˈredən | verb")
	out, _ := Render(entry, RenderOpts{Color: true, Language: "en", Tint: tintPolicy{lang: "en", background: languageDark}})
	assertDictionaryTint(t, out, "rɛd", false)
	assertDictionaryTint(t, out, "ˈredən", false)
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
