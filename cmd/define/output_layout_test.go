package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestOutputLayoutWrapsPaintAndActions(t *testing.T) {
	source := "  uno dos tres cuatro cinco seis\n\nnext"
	o := renderedOutput{text: source, rows: []rowPaint{{background: languageDark, exclusions: []cellRange{{22, 27}}}, {background: languageLight}}, regions: []Region{{Line: 0, Col: 22, Width: 5, Text: "cinco"}, {Line: 2, Col: 0, Width: 4, Text: "next"}}}
	got := layoutOutput(o, 20)
	if got.text != "  uno dos tres\n  cuatro cinco seis\n\nnext" {
		t.Fatalf("layout = %q", got.text)
	}
	if len(got.rows) != 4 || got.rows[0].background != languageDark || got.rows[1].background != languageDark || got.rows[2].background != languageLight || got.rows[3].background != "" {
		t.Fatalf("paint projection: %+v", got.rows)
	}
	if !reflect.DeepEqual(got.rows[1].exclusions, []cellRange{{9, 14}}) {
		t.Errorf("exclusions=%+v", got.rows[1].exclusions)
	}
	if len(got.regions) != 2 || got.regions[0].Line != 1 || got.regions[0].Col != 9 || got.regions[0].Width != 5 || got.regions[1].Line != 3 {
		t.Errorf("click projection=%+v", got.regions)
	}
	if o.text != source {
		t.Fatal("source mutated")
	}
}

func TestOutputLayoutWideCoordinatesAndInvalidMetadata(t *testing.T) {
	o := renderedOutput{text: "  界 a\n", rows: []rowPaint{{background: languageDark, exclusions: []cellRange{{2, 4}}}}, regions: []Region{{Line: 0, Col: 2, Width: 2, Text: "界"}}}
	got := layoutOutput(o, 20)
	if !reflect.DeepEqual(got.regions, o.regions) || !reflect.DeepEqual(got.rows[0], o.rows[0]) {
		t.Fatalf("wide display geometry changed: %+v", got)
	}
	for _, rows := range [][]rowPaint{{{background: "bad"}}, {{background: languageDark, exclusions: []cellRange{{2, 1}}}}, {{}, {}, {}}} {
		o.rows = rows
		got = layoutOutput(o, 20)
		if got.text != o.text {
			t.Error("invalid metadata loses source")
		}
		for _, p := range got.rows {
			if p.background != "" || len(p.exclusions) > 0 {
				t.Errorf("invalid metadata retained: %+v", p)
			}
		}
	}
}

func TestOutputSerializationKeepsSourceAndClipboardUnpadded(t *testing.T) {
	o := renderedOutput{text: "  hola\n\nfin\n", rows: []rowPaint{{background: languageDark}, {background: languageDark}, {background: languageDark}}}
	got := serializeOutput(o, 20)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 || lines[3] != "" {
		t.Fatalf("trailing newline emits extra row: %q", got)
	}
	for _, line := range lines[:3] {
		cells, end := rowTestCells(t, line, 20)
		for _, c := range cells {
			if c.bg != 236 {
				t.Error("unfilled serialized cell")
			}
		}
		if end.bg != -1 {
			t.Error("background leaks to newline")
		}
	}
	if serializeOutput(o, 0) != o.text {
		t.Error("plain serialization adds padding")
	}
	f := newSelectionFrame(20, 3, []selectionRow{{styled: "  hola", selectable: true}, {styled: "", selectable: true}, {styled: "fin", selectable: true}})
	copied, err := selectedText(f, selectionPoint{row: 0, col: 0}, selectionPoint{row: 2, col: 19})
	if err != nil || copied != "  hola\n\nfin" {
		t.Fatalf("clipboard=%q, %v", copied, err)
	}
}

func TestOutputSerializationPreservesStylesAcrossRows(t *testing.T) {
	o := renderedOutput{text: "\x1b[31muno\ndos\x1b[0m", rows: []rowPaint{{background: languageDark}, {background: languageDark}}}
	lines := strings.Split(serializeOutput(o, 20), "\n")
	for _, line := range lines {
		cells, _ := rowTestCells(t, line, 20)
		if cells[0].fg != 31 {
			t.Fatalf("producer foreground lost: %q", line)
		}
	}
}

func TestOutputLayoutWidthChangesDoNotPadStoredText(t *testing.T) {
	o := renderedOutput{text: "  one two three four five six", rows: []rowPaint{{background: languageDark}}}
	narrow := layoutOutput(o, 20)
	if narrow.text != "  one two three four\n  five six" {
		t.Fatalf("narrow text=%q", narrow.text)
	}
	wide := layoutOutput(narrow, 40)
	if wide.text != narrow.text {
		t.Error("historical rows reflowed")
	}
	for _, width := range []int{20, 40, 20} {
		serialized := serializeOutput(wide, width)
		for _, line := range strings.Split(serialized, "\n") {
			cells, _ := rowTestCells(t, line, width)
			for col, c := range cells {
				if c.bg != 236 {
					t.Fatalf("width %d col %d missing fill", width, col)
				}
			}
		}
		if wide.text != narrow.text {
			t.Fatal("paint mutated stored rows")
		}
	}
}
