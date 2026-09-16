package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func assertOxfordRows(t *testing.T, out string) {
	t.Helper()
	for _, want := range []string{"  A transitive verb", "  B intransitive verb", "  C (rendirse) pronominal verb", "    2", "      a (Finance) to yield", "      b (producir) to produce", "      a ‹informe› to present", "▸ rendían culto a la Virgen de Guadalupe they worshipped the Virgin of Guadalupe"} {
		found := false
		for _, line := range strings.Split(stripEscapes(out), "\n") {
			if strings.HasSuffix(line, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing structural row %q", want)
		}
	}
}
func TestOxfordNativeStructure(t *testing.T) {
	record := bilingualFixture(t, "rendir")[0]
	doc, err := parseBilingualDocument(record)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := renderBilingualDocument(doc, RenderOpts{})
	if oxfordContent(out) != oxfordContent(record.Text) {
		t.Fatal("source content lost or reordered")
	}
	assertOxfordRows(t, out)
}
func oxfordContent(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, stripEscapes(s))
}

func TestOxfordColorPreservesRowsAndActions(t *testing.T) {
	doc, err := parseBilingualDocument(bilingualFixture(t, "rendir")[0])
	if err != nil {
		t.Fatal(err)
	}
	plain, actions := renderBilingualDocument(doc, RenderOpts{Word: "rendir"})
	colored, got := renderBilingualDocument(doc, RenderOpts{Word: "rendir", Color: true})
	if stripEscapes(colored) != plain {
		t.Fatal("color changed logical text/row whitespace")
	}
	if !reflect.DeepEqual(actions, got) || len(got) != 1 || got[0].Line != 0 || got[0].Col != 0 || got[0].Width != 6 {
		t.Fatalf("headword actions: %+v", got)
	}
	assertOxfordRows(t, colored)
}

func TestOxfordNativeTreeAndLeafConservation(t *testing.T) {
	doc, err := parseBilingualDocument(bilingualFixture(t, "rendir")[0])
	if err != nil {
		t.Fatal(err)
	}
	children := func(n *bilingualNode, class string) []*bilingualNode {
		var out []*bilingualNode
		for _, c := range n.children {
			if c.has(class) {
				out = append(out, c)
			}
		}
		return out
	}
	groups := children(doc.root, "gramb")
	if len(groups) != 3 {
		t.Fatalf("grammatical groups=%d", len(groups))
	}
	for i, want := range []int{5, 4, 2} {
		if got := len(children(groups[i], "semb")); got != want {
			t.Fatalf("group %d senses=%d want %d", i, got, want)
		}
	}
	senses := children(groups[0], "semb")
	for _, i := range []int{1, 3} {
		if got := len(children(senses[i], "semb")); got != 2 {
			t.Fatalf("A%d sub-senses=%d", i+1, got)
		}
	}
	at := 0
	var walk func(*bilingualNode)
	walk = func(n *bilingualNode) {
		if n.tag == "" {
			if n.start != at || n.end < n.start {
				t.Fatalf("leaf gap/overlap at %d: %+v", at, n)
			}
			at = n.end
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(doc.root)
	if at != len(doc.source.text) {
		t.Fatal("unconsumed source leaves")
	}
}

func TestOxfordCorpusConservation(t *testing.T) {
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
			doc, err := parseBilingualDocument(record)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			for _, width := range []int{0, 32, 80} {
				out, _ := renderBilingualDocument(doc, RenderOpts{Color: true, Width: width})
				if oxfordContent(out) != oxfordContent(record.Text) {
					t.Fatalf("%s width %d: content lost/reordered", name, width)
				}
			}
		}
	}
}

func TestOxfordRejectsUnprovenRecords(t *testing.T) {
	native := bilingualFixture(t, "rendir")[0]
	for _, tc := range []struct {
		name   string
		mutate func(*bilingualRecord)
	}{
		{"text mismatch", func(r *bilingualRecord) { r.Text = "wrong " + r.Text }},
		{"word boundary", func(r *bilingualRecord) { r.Text = strings.Replace(r.Text, "to pay", "topay", 1) }},
		{"truncated", func(r *bilingualRecord) { r.HTML = r.HTML[:len(r.HTML)/2] }},
		{"wrong direction", func(r *bilingualRecord) { r.HTML = strings.Replace(r.HTML, "s_b-es-en0040662", "e_b-en-es0040662", 1) }},
		{"duplicate entry", func(r *bilingualRecord) {
			r.HTML = strings.Replace(r.HTML, "</body>", `<d:entry id="s_b-es-en1" d:title="rendir">rendir</d:entry></body>`, 1)
		}},
		{"depth", func(r *bilingualRecord) {
			r.HTML = strings.Replace(r.HTML, "rendir </span>", strings.Repeat("<span>", 65)+"rendir"+strings.Repeat("</span>", 65)+" </span>", 1)
		}},
		{"size", func(r *bilingualRecord) { r.HTML = strings.Repeat(" ", bilingualMaxBytes) + r.HTML }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := native
			tc.mutate(&r)
			if _, err := parseBilingualDocument(r); err == nil {
				t.Fatal("trusted invalid record")
			}
			if got := bilingualLanguageText(r); len(got.spans) != 0 || got.text != r.Text {
				t.Fatal("fallback changed text or retained ownership")
			}
		})
	}
}

func TestOxfordUnknownWrappersAndEmphasis(t *testing.T) {
	r := bilingualFixture(t, "rendir")[0]
	r.HTML = strings.Replace(r.HTML, "they worshipped the Virgin of Guadalupe", "<mystery><strong>they worshipped</strong> the Virgin of Guadalupe</mystery>", 1)
	doc, err := parseBilingualDocument(r)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := renderBilingualDocument(doc, RenderOpts{Color: true})
	if !strings.Contains(out, "\x1b[1mthey worshipped\x1b[0m") {
		t.Fatal("explicit emphasis lost")
	}
	if oxfordContent(out) != oxfordContent(r.Text) {
		t.Fatal("unknown wrapper lost content")
	}
}

func FuzzOxfordDocument(f *testing.F) {
	data, err := os.ReadFile("testdata/bilingual/rendir.json")
	if err != nil {
		f.Fatal(err)
	}
	var records []bilingualRecord
	if err := json.Unmarshal(data, &records); err != nil {
		f.Fatal(err)
	}
	f.Add(records[0].HTML, records[0].Text)
	f.Fuzz(func(t *testing.T, html, text string) {
		record := bilingualRecord{HTML: html, Text: text}
		doc, err := parseBilingualDocument(record)
		if err != nil {
			return
		}
		rendered, _ := renderBilingualDocument(doc, RenderOpts{})
		if oxfordContent(rendered) != oxfordContent(text) {
			t.Fatal("accepted record lost/duplicated/reordered content")
		}
		at := 0
		var walk func(*bilingualNode)
		walk = func(n *bilingualNode) {
			if n.tag == "" {
				if n.start != at || n.end < n.start || n.end > len(doc.source.text) {
					t.Fatal("invalid source leaf")
				}
				at = n.end
			}
			for _, c := range n.children {
				walk(c)
			}
		}
		walk(doc.root)
		if at != len(doc.source.text) {
			t.Fatal("source not fully consumed")
		}
		for _, span := range doc.native.spans {
			if span.start < 0 || span.end < span.start || span.end > len(text) {
				t.Fatal("invalid native ownership")
			}
		}
	})
}

func TestOxfordNativeIdiomsAndEmphasis(t *testing.T) {
	for _, tc := range []struct{ name, needle string }{{"red", "\x1b[4mor\x1b[0m"}, {"mesa", "\x1b[1metc.\x1b[0m"}, {"arbol", "\x1b[3malso "}} {
		selected, err := selectedSpanishRecords(bilingualFixture(t, tc.name), tc.name, "")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := parseBilingualDocument(selected[0])
		if err != nil {
			t.Fatal(err)
		}
		out, _ := renderBilingualDocument(doc, RenderOpts{Color: true})
		if !strings.Contains(out, tc.needle) {
			t.Errorf("%s lost native emphasis %q", tc.name, tc.needle)
		}
		if tc.name == "red" && !strings.Contains(stripEscapes(out), "    caer en las redes de alguien to fall into somebody's clutches\n") {
			t.Fatal("idiom/translation lost structural row")
		}
	}
}
