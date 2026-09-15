package main

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func bilingualFixture(t testing.TB, word string) []bilingualRecord {
	t.Helper()
	b, err := os.ReadFile("testdata/bilingual/" + word + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var records []bilingualRecord
	if err := json.Unmarshal(b, &records); err != nil {
		t.Fatal(err)
	}
	return records
}

func TestBilingualSelection(t *testing.T) {
	for _, tc := range []struct {
		word, canonical, contains string
		count                     int
	}{
		{"red", "red", "net", 1}, {"mesa", "mesa", "table", 1}, {"madrugar", "madrugar", "get up early", 1},
		{"arbol", "árbol", "tree", 1}, {"jalapeno", "jalapeño", "pepper", 1}, {"como", "como", "como", 3},
		{"solo", "solo", "solo", 3}, {"pie", "pie", "pie", 2}, {"son", "son", "son", 2},
		{"madrugaste", "madrugar", "get up early", 1}, {"mesas", "mesa", "table", 1},
	} {
		t.Run(tc.word, func(t *testing.T) {
			records := bilingualFixture(t, tc.word)
			got, err := selectSpanishRecords(records, tc.word, tc.canonical)
			if err != nil || len(got) != tc.count || !strings.Contains(strings.Join(got, "\n"), tc.contains) {
				t.Fatalf("got %q, %v", got, err)
			}
			if tc.word == "red" && strings.Contains(strings.Join(got, ""), "redder") {
				t.Fatal("wrong direction")
			}
			slices.Reverse(records)
			again, err := selectSpanishRecords(append(records, records...), tc.word, tc.canonical)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatalf("permutation/duplicate changed selection: %q %v", again, err)
			}
		})
	}
}

func TestBilingualSelectionRejects(t *testing.T) {
	good := bilingualFixture(t, "mesa")[0]
	for _, tc := range []struct {
		name    string
		records []bilingualRecord
		want    error
	}{
		{"empty", nil, ErrNoEntry},
		{"unrelated", []bilingualRecord{good}, ErrNoEntry},
		{"malformed", []bilingualRecord{{HTML: "<html>", Text: "entry"}}, ErrBilingualMalformed},
		{"nested impostor", []bilingualRecord{{HTML: `<html><body><span><d:entry xmlns:d="http://www.apple.com/DTDs/DictionaryService-1.0.rng" id="s_b-es-en1" d:title="red"/></span></body></html>`, Text: "entry"}}, ErrBilingualMalformed},
		{"unknown direction", []bilingualRecord{{HTML: strings.Replace(good.HTML, "s_b-es-en", "unknown", -1), Text: good.Text}}, ErrBilingualMalformed},
		{"wrong direction", bilingualFixture(t, "red")[:2], ErrNoEntry},
		{"empty text", []bilingualRecord{{HTML: good.HTML}}, ErrBilingualMalformed},
		{"bytes", []bilingualRecord{{HTML: strings.Repeat("x", bilingualMaxBytes+1), Text: "entry"}}, ErrBilingualLimit},
		{"records", make([]bilingualRecord, bilingualMaxRecords+1), ErrBilingualLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := selectSpanishRecords(tc.records, "red", "")
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	if _, err := selectSpanishRecords(bilingualFixture(t, "madrugaste"), "madrugaste", ""); !errors.Is(err, ErrNoEntry) {
		t.Fatalf("guessed inflection: %v", err)
	}
}

func TestBilingualSelectionMatching(t *testing.T) {
	record := func(id, title, text string) bilingualRecord {
		return bilingualRecord{HTML: `<html><body><d:entry xmlns:d="` + bilingualEntryNamespace + `" id="s_b-es-en` + id + `" d:title="` + title + `"/></body></html>`, Text: text}
	}
	for _, tc := range []struct {
		name, word, canonical string
		records               []bilingualRecord
		want                  string
	}{
		{"exact before accent", "como", "comer", []bilingualRecord{record("1", "cómo", "accent"), record("2", "como", "exact"), record("3", "comer", "canonical")}, "exact"},
		{"accent before canonical", "arbol", "otro", []bilingualRecord{record("1", "árbol", "accent"), record("2", "otro", "canonical")}, "accent"},
		{"canonical accent", "mesas", "arbol", []bilingualRecord{record("1", "árbol", "canonical")}, "canonical"},
		{"distinct letters aren't accents", "ça", "xa", []bilingualRecord{record("1", "ña", "wrong")}, ""},
		{"ascii vs unrelated accent", "a", "", []bilingualRecord{record("1", "é", "wrong")}, ""},
		{"combining accent", "a\u0301rbol", "", []bilingualRecord{record("1", "árbol", "accent")}, "accent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := selectSpanishRecords(tc.records, tc.word, tc.canonical)
			if tc.want == "" {
				if !errors.Is(err, ErrNoEntry) {
					t.Fatalf("got %q %v", got, err)
				}
				return
			}
			if err != nil || len(got) != 1 || got[0] != tc.want {
				t.Fatalf("got %q %v", got, err)
			}
		})
	}
}

func TestBilingualMalformedMetadata(t *testing.T) {
	record := bilingualFixture(t, "mesa")[0]
	for _, html := range []string{
		record.HTML + "unexpected trailing text",
		strings.Replace(record.HTML, `<html xmlns:d`, `<html xmlns="urn:impostor" xmlns:d`, 1),
		strings.Replace(record.HTML, `id="s_b-es-en0031072"`, `id="s_b-es-en0031072" id="s_b-es-en999"`, 1),
		strings.Replace(record.HTML, `<body>`, `<body><span>`, 1),
	} {
		if _, err := selectSpanishRecords([]bilingualRecord{{HTML: html, Text: record.Text}}, "mesa", ""); !errors.Is(err, ErrBilingualMalformed) {
			t.Fatalf("accepted malformed root: %v", err)
		}
	}
	// Unusable candidates must not hide an independently verified good record.
	got, err := selectSpanishRecords([]bilingualRecord{{HTML: "bad", Text: "bad"}, record}, "mesa", "")
	if err != nil || len(got) != 1 || got[0] != record.Text {
		t.Fatalf("lost usable result: %q %v", got, err)
	}
}

// fakeRecordSource models changing installation, records and failures between calls.
type fakeRecordSource struct {
	installed bool
	entries   map[string][]bilingualRecord
	failures  map[string]error
	calls     []string
}

func (f *fakeRecordSource) Records(word string) ([]bilingualRecord, error) {
	f.calls = append(f.calls, word)
	if !f.installed {
		return nil, ErrBilingualUnavailable
	}
	if err := f.failures[word]; err != nil {
		return nil, err
	}
	return slices.Clone(f.entries[word]), nil
}

func TestBilingualSourceState(t *testing.T) {
	f := &fakeRecordSource{entries: map[string][]bilingualRecord{"mesa": bilingualFixture(t, "mesa")}}
	if _, err := f.Records("mesa"); !errors.Is(err, ErrBilingualUnavailable) {
		t.Fatal(err)
	}
	f.installed = true
	if rs, err := f.Records("mesa"); err != nil || len(rs) != 1 {
		t.Fatalf("%v %v", rs, err)
	}
	f.failures = map[string]error{"mesa": ErrLookupFailed}
	if _, err := f.Records("mesa"); !errors.Is(err, ErrLookupFailed) {
		t.Fatal(err)
	}
	if len(f.calls) != 3 {
		t.Fatal(f.calls)
	}
}

func FuzzBilingualRecords(f *testing.F) {
	for _, word := range []string{"red", "mesa", "como"} {
		for _, r := range bilingualFixture(f, word) {
			f.Add(r.HTML, r.Text, word)
		}
	}
	f.Add("<html>", "bad", "red")
	f.Add(`<html><body><span><d:entry xmlns:d="`+bilingualEntryNamespace+`" id="s_b-es-en1" d:title="red"/></span></body></html>`, "nested", "red")
	f.Add(`<html attribute="`+strings.Repeat("x", bilingualMaxBytes)+`"/>`, "huge", "red")
	f.Add(strings.Replace(bilingualFixture(f, "mesa")[0].HTML, "s_b-es-en", "unknown", -1), "unknown", "mesa")
	f.Fuzz(func(t *testing.T, html, text, word string) {
		r := bilingualRecord{HTML: html, Text: text}
		got, err := selectSpanishRecords([]bilingualRecord{r, r}, word, "")
		if err == nil && (len(got) != 1 || got[0] != text) {
			t.Fatalf("invented/duplicated text: %q", got)
		}
		again, other := selectSpanishRecords([]bilingualRecord{r, r}, word, "")
		if !reflect.DeepEqual(got, again) || (err == nil) != (other == nil) {
			t.Fatal("nondeterminism")
		}
	})
}
