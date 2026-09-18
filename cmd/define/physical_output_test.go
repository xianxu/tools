package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"io"
	"strings"
	"testing"
)

func TestDefinitionOutputWrapsLongWordBeforeScreenHistory(t *testing.T) {
	word := "anticonstitucionalmente"
	o := renderDefinitionOutput(definitionSet{sections: []definitionSection{{language: "es", entries: []string{word + " nombre una palabra"}}}}, RenderOpts{Color: true, Width: 20, Word: word, Tint: tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}})
	lines := strings.Split(o.text, "\n")
	for _, line := range lines {
		if visibleCells(line) > 20 {
			t.Fatalf("definition row overflows before screen: %q", line)
		}
	}
	if strings.Join(strings.Fields(stripEscapes(lines[0]+lines[1])), "") != word {
		t.Fatalf("long headword was not physical rows: %q", o.text)
	}
	l := newLiveScreen(io.Discard, 20, 20)
	defer l.Stop()
	l.WriteOutput(o)
	if strings.Join(strings.Fields(stripEscapes(l.s.lines[0]+l.s.lines[1])), "") != word {
		t.Fatal("screen source lost suffix")
	}
	before := l.Transcript()
	l.Resize(20, 10)
	if l.Transcript() != before {
		t.Fatal("historical resize reflowed source")
	}
}
func TestPracticeLongRunOwnershipFollowsPhysicalRows(t *testing.T) {
	text := strings.Repeat("a", 23) + " hello"
	p := play.Presentation{Text: text, Spans: []play.LanguageSpan{{Start: 0, End: 23, Role: play.Target}, {Start: 24, End: len(text), Role: play.English}}}
	o := renderPracticeOutput(p, "es", "es", tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "", 20)
	for _, line := range strings.Split(o.text, "\n") {
		if visibleCells(line) > 20 {
			t.Fatalf("not physical: %q", o.text)
		}
	}
	if !o.rows[0].tinted {
		t.Fatal("first completed target row lost ownership")
	}
}

func TestPracticeActionsFollowPhysicalOutputIntoScreen(t *testing.T) {
	for _, color := range []bool{false, true} {
		l := newLiveScreen(io.Discard, 20, 20)
		defer l.Stop()
		p := play.Presentation{Text: strings.Repeat("a", 23) + "\nhola"}
		writePracticePresentation(l, p, []Region{{Line: 2, Col: 0, Width: 4, Text: "hola"}}, deps{lang: "es"}, options{width: 20, color: color, tintOn: true}, surfaceProse, "", "")
		r, ok := l.s.RegionAt(3, 0)
		if !ok || r.Text != "hola" || stripEscapes(l.s.lines[3]) != "hola" {
			t.Fatalf("color=%v action misplaced: rows=%q regions=%+v", color, l.s.lines, l.s.regions)
		}
		l.Stop()
	}
}

func TestStructuredRegionSinkUsesPhysicalGeometry(t *testing.T) {
	var w recordingRegionWriter
	o := renderedOutput{text: strings.Repeat("a", 23) + "\nhola", regions: []Region{{Line: 1, Col: 0, Width: 4, Text: "hola"}}}
	if err := writeOutput(&w, o, 20, store.SchemeDark); err != nil {
		t.Fatal(err)
	}
	if len(w.regions) != 1 || w.regions[0].Line != 2 {
		t.Fatalf("regions=%+v", w.regions)
	}
}
