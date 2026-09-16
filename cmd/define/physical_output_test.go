package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"io"
	"strings"
	"testing"
)

func TestDefinitionOutputWrapsLongWordBeforeScreenHistory(t *testing.T) {
	word := "anticonstitucionalmente"
	o := renderDefinitionOutput(definitionSet{sections: []definitionSection{{language: "es", entries: []string{word + " nombre una palabra"}}}}, RenderOpts{Color: true, Width: 20, Word: word, Tint: tintPolicy{"es", languageDark}})
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
	o := renderPracticeOutput(p, "es", "es", tintPolicy{"es", languageDark}, nil, surfaceProse, "", 20)
	for _, line := range strings.Split(o.text, "\n") {
		if visibleCells(line) > 20 {
			t.Fatalf("not physical: %q", o.text)
		}
	}
	if o.rows[0].background != languageDark {
		t.Fatal("first completed target row lost ownership")
	}
}
