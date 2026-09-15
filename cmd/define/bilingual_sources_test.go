package main

import (
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
