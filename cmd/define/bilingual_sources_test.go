package main

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestBilingualSourceAvailabilityNames(t *testing.T) {
	primary := dictMeta{ID: "com.apple.dictionary.es.DGLEV", Langs: []langPair{{Index: "es", Description: "es"}}}
	english := dictMeta{ID: spanishEnglishDictionaryID, Langs: []langPair{{Index: "es", Description: "es"}, {Index: "en", Description: "es"}}}
	for _, tc := range []struct {
		name      string
		installed []dictMeta
		primary   bool
		want      string
	}{
		{"both", []dictMeta{primary, english}, true, "Larousse Diccionario General; English: Oxford Spanish–English"},
		{"primary only", []dictMeta{primary}, true, "Larousse Diccionario General; English: Oxford Spanish–English (unavailable)"},
		{"English only", []dictMeta{english}, false, "Larousse Diccionario General (unavailable); English: Oxford Spanish–English"},
		{"neither", []dictMeta{}, false, "Larousse Diccionario General (unavailable); English: Oxford Spanish–English (unavailable)"},
		{"unknown", nil, false, "Larousse Diccionario General (availability unknown); English: Oxford Spanish–English (availability unknown)"},
		{"wrong primary metadata", []dictMeta{{ID: primary.ID, Langs: english.Langs}, english}, false, "Larousse Diccionario General (unavailable); English: Oxford Spanish–English"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ids, name := spanishDictionarySources(tc.installed)
			if (len(ids) > 0) != tc.primary || name != tc.want {
				t.Fatalf("ids %v name %q; want primary=%t %q", ids, name, tc.primary, tc.want)
			}
			if tc.primary && strings.Join(ids, ",") != primary.ID {
				t.Fatalf("wrong primary %v", ids)
			}
		})
	}
}

func TestBilingualFactoryMetadataDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name               string
		installed          []dictMeta
		diagnostic         string
		installationAdvice bool
	}{
		{"unknown metadata", nil, "dictionary-selection API unavailable", false},
		{"confirmed missing", []dictMeta{}, "enable Spanish (Larousse Diccionario General)", true},
	} {
		for _, englishAvailable := range []bool{true, false} {
			t.Run(tc.name+"/english="+strconv.FormatBool(englishAvailable), func(t *testing.T) {
				source := &fakeRecordSource{installed: englishAvailable, entries: map[string][]bilingualRecord{"mesa": bilingualFixture(t, "mesa")}}
				madePrimary := false
				dictionary, _ := spanishDictionaryFromInstalled(tc.installed, func([]string) Dictionary { madePrimary = true; return &fakeDictionary{} }, source)
				if madePrimary {
					t.Fatal("constructed primary from unavailable metadata")
				}
				on := true
				d := deps{dict: dictionary, bilingual: &on, langDeps: langDeps{capture: noopCapturer{}}}
				var out, errout strings.Builder
				result := lookupAndRender(d, options{}, replCommand{word: "mesa", literal: true}, &out, &errout)
				// A usable supplemental entry must remain visible even when metadata
				// enumeration cannot establish the primary dictionary's availability.
				if (result.code == 0) != englishAvailable {
					t.Fatalf("code=%d stdout=%q stderr=%q", result.code, out.String(), errout.String())
				}
				visible := out.String() + errout.String()
				if !strings.Contains(visible, tc.diagnostic) {
					t.Fatalf("missing diagnosis %q: stdout=%q stderr=%q", tc.diagnostic, out.String(), errout.String())
				}
				if englishAvailable && !strings.Contains(out.String(), "table") {
					t.Fatalf("discarded usable English entry: %q", out.String())
				}
				// Scope to the primary diagnosis: an absent English source legitimately
				// retains its own installation advice in the second section.
				_, primaryErr := dictionary.Lookup("mesa")
				if primaryErr == nil || strings.Contains(primaryErr.Error(), "enable Spanish") != tc.installationAdvice {
					t.Fatalf("primary diagnosis %v", primaryErr)
				}
				if !tc.installationAdvice && strings.Contains(primaryErr.Error(), "wait for the download") {
					t.Fatalf("unknown metadata asserted a missing installation: %v", primaryErr)
				}
				if errors.Is(primaryErr, ErrLookupFailed) == tc.installationAdvice {
					t.Fatalf("metadata failure and confirmed absence have same error category: %v", primaryErr)
				}
				if len(source.calls) != 1 {
					t.Fatalf("supplement fetched %d times", len(source.calls))
				}
			})
		}
	}
}
