package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"io"
	"strings"
	"testing"
)

func TestDefinitionUnknownSourceDoesNotInheritStudyLanguage(t *testing.T) {
	raw := "word noun an English definition"
	dict := &fakeDictionary{}
	set := definitionsFor(dict, "word", raw, nil, false)
	out, _ := renderDefinitions(set, RenderOpts{Color: true, Tint: tintPolicy{lang: "it", on: true, scheme: holderFor(store.SchemeDark)}})
	if strings.Contains(out, languageDark) {
		t.Fatal("unknown dictionary source inherited study language")
	}
}

func TestDictionaryAssemblyOwnsVerifiedPrimaryLanguage(t *testing.T) {
	italian := dictMeta{ID: curated["it"][0], Langs: []langPair{{Index: "it", Description: "it"}}}
	english := dictMeta{ID: curated["en"][0], Langs: []langPair{{Index: "en", Description: "en"}}}
	bilingual := italian
	bilingual.Langs = []langPair{{Index: "it", Description: "it"}, {Index: "en", Description: "it"}}
	for _, tc := range []struct {
		name         string
		installed    []dictMeta
		target, want store.Lang
	}{
		{"metadata unavailable", nil, "it", ""},
		{"nothing installed", []dictMeta{}, "it", ""},
		{"Italian unavailable", []dictMeta{english}, "it", ""},
		{"bilingual is unverified", []dictMeta{bilingual}, "it", ""},
		{"verified Italian", []dictMeta{italian, english}, "it", "it"},
		{"verified English", []dictMeta{italian, english}, "en", "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := "word noun source prose"
			selectedCalls := 0
			dict, _, _ := dictionaryFromInstalled(tc.installed, tc.target, func(ids []string) Dictionary {
				selectedCalls++
				if len(ids) == 0 {
					t.Fatal("empty IDs selected")
				}
				return &definitionFake{primary: raw}
			}, &definitionFake{primary: raw})
			if got := dictionarySourceLanguage(dict); got != tc.want {
				t.Fatalf("source=%q want=%q", got, tc.want)
			}
			if (selectedCalls > 0) != (tc.want != "") {
				t.Fatal("provenance and actual selection disagree")
			}
			// Wrappers and misleading status labels cannot invent or change provenance.
			wrapped := untranslatedDefinitions{Dictionary: dict, language: "Italian"}
			factory := lockedDictionaries(func(store.Lang, io.Writer) (Dictionary, string) { return wrapped, "Italian" })
			locked, _ := factory("it", io.Discard)
			if got := dictionarySourceLanguage(locked); got != tc.want {
				t.Fatalf("wrapped source=%q want=%q", got, tc.want)
			}
			text, err := locked.Lookup("word")
			if err != nil {
				t.Fatal(err)
			}
			set := definitionsFor(locked, "word", text, nil, false)
			for _, target := range []store.Lang{"it", "en"} {
				out, _ := renderDefinitions(set, RenderOpts{Color: true, Tint: tintPolicy{lang: target, on: true, scheme: holderFor(store.SchemeDark)}})
				assertDictionaryTint(t, out, "source prose", tc.want != "" && tc.want == target)
			}
		})
	}
}

func TestSelectedSourceLanguageRequiresEveryActualID(t *testing.T) {
	en := dictMeta{ID: curated["en"][0], Langs: []langPair{{Index: "en", Description: "en"}}}
	es := dictMeta{ID: curated["es"][0], Langs: []langPair{{Index: "es", Description: "es"}}}
	for _, tc := range []struct {
		ids  []string
		want store.Lang
	}{
		{nil, ""}, {[]string{en.ID}, "en"}, {[]string{es.ID}, "es"}, {[]string{en.ID, es.ID}, ""}, {[]string{en.ID, "missing"}, ""},
	} {
		if got := selectedSourceLanguage([]dictMeta{en, es}, tc.ids); got != tc.want {
			t.Fatalf("%v source=%q want=%q", tc.ids, got, tc.want)
		}
	}
}

func TestSpanishAssemblyPreservesPrimarySourceMetadata(t *testing.T) {
	primary := dictMeta{ID: curated["es"][0], Langs: []langPair{{Index: "es", Description: "es"}}}
	source := &fakeRecordSource{installed: true, entries: map[string][]bilingualRecord{"red": bilingualFixture(t, "red")}}
	dict, _ := spanishDictionaryFromInstalled([]dictMeta{primary}, func([]string) Dictionary { return &definitionFake{primary: "red nombre femenino tejido"} }, source)
	if got := dictionarySourceLanguage(dict); got != "es" {
		t.Fatalf("Spanish selected source=%q", got)
	}
	absent, _ := spanishDictionaryFromInstalled(nil, func([]string) Dictionary { t.Fatal("constructed unproven primary"); return nil }, source)
	if got := dictionarySourceLanguage(absent); got != "" {
		t.Fatalf("absent primary claimed %q", got)
	}
}
