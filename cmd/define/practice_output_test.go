package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"testing"
)

func TestPracticeOutputClassifiesPhysicalRows(t *testing.T) {
	text := "  uno dos tres cuatro cinco seis seven"
	foreign := strings.Index(text, "seven")
	p := play.Presentation{Text: text, Spans: []play.LanguageSpan{{Start: 2, End: foreign, Role: play.Target}, {Start: foreign, End: len(text), Role: play.English}}}
	o := renderPracticeOutput(p, "es", "es", tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "", 20)
	lines := strings.Split(o.text, "\n")
	if len(lines) < 2 {
		t.Fatalf("no physical wrap: %q", o.text)
	}
	if !o.rows[0].tinted || o.rows[len(lines)-1].tinted {
		t.Fatalf("ownership: %+v / %q", o.rows, o.text)
	}
	if strings.Contains(o.text, languageDark) {
		t.Fatal("paint leaked into source")
	}
}
func TestPracticeDefinitionRegionsOverrideMixedProse(t *testing.T) {
	text := "Oxford\nhola hello\n\nfin end\n"
	p := play.Presentation{Text: text, Regions: []play.PresentationRegion{{Start: 0, End: len(text), Tinted: true}}}
	o := renderPracticeOutput(p, "en", "es", tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeLight)}, nil, surfaceProse, "", 30)
	for i := 0; i < 4; i++ {
		cells, _ := rowTestCells(t, paintLanguageRow(strings.Split(o.text, "\n")[i], o.rows[i], 30, store.SchemeLight), 30)
		for col, c := range cells {
			if c.bg != 254 {
				t.Fatalf("row %d col %d background %d", i, col, c.bg)
			}
		}
	}
	for _, q := range []interface{ RevealPresentation() play.Presentation }{play.NewChoice("hola", text, nil).WithDefinitionRegions(p.Regions), play.NewCloze("hola", "___", "hola", text, nil).WithDefinitionRegions(p.Regions)} {
		reveal := q.RevealPresentation()
		if len(reveal.Regions) != 1 || reveal.Text[reveal.Regions[0].Start:reveal.Regions[0].End] != text {
			t.Fatalf("embedded metadata lost: %+v", reveal)
		}
	}
}
func TestPracticeUnknownProseAndAnswerExclusions(t *testing.T) {
	p := play.Presentation{Text: "hola unknown", Spans: []play.LanguageSpan{{Start: 0, End: 4, Role: play.Target}}}
	o := renderPracticeOutput(p, "es", "es", tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "", 30)
	if o.rows[0].tinted {
		t.Fatal("unknown substantive prose tinted")
	}
	p = play.Presentation{Text: "hola bien", Spans: []play.LanguageSpan{{Start: 0, End: 4, Role: play.Target}, {Start: 5, End: 9, Role: play.Target, AnswerStyled: true}}}
	o = renderPracticeOutput(p, "es", "es", tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "", 30)
	cells, _ := rowTestCells(t, paintLanguageRow(o.text, o.rows[0], 30, store.SchemeDark), 30)
	for i, c := range cells {
		want := 236
		if i >= 5 && i < 9 {
			want = -1
		}
		if c.bg != want {
			t.Fatalf("cell %d bg %d want %d", i, c.bg, want)
		}
	}
}

func TestPracticeChromeOutput(t *testing.T) {
	var b practiceBuilder
	b.english("next word")
	pal := newPalette(true)
	o := practiceChromeOutput(b.Presentation, deps{lang: "en"}, options{color: true, tintOn: true}, pal)
	if !strings.Contains(o.text, pal.dim) || !o.rows[0].tinted {
		t.Fatalf("chrome or metadata missing: %+v", o)
	}
}
