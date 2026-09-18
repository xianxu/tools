package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestOutputLayoutWrapsPaintAndActions(t *testing.T) {
	// The second source row's paint is the variable: tinted, it must reach
	// physical row 2 and not row 3; untinted, row 2 must not take row 0's. (Before
	// #70 the two source rows carried different shades, which told them apart in
	// one pass; a row now keeps one bit, so both values of it are run.)
	for _, second := range []bool{true, false} {
		source := "  uno dos tres cuatro cinco seis\n\nnext"
		o := renderedOutput{text: source, rows: []rowPaint{{tinted: true, exclusions: []cellRange{{22, 27}}}, {tinted: second}}, regions: []Region{{Line: 0, Col: 22, Width: 5, Text: "cinco"}, {Line: 2, Col: 0, Width: 4, Text: "next"}}}
		got := layoutOutput(o, 20)
		if got.text != "  uno dos tres\n  cuatro cinco seis\n\nnext" {
			t.Fatalf("layout = %q", got.text)
		}
		if len(got.rows) != 4 || !got.rows[0].tinted || !got.rows[1].tinted || got.rows[2].tinted != second || got.rows[3].tinted {
			t.Fatalf("paint projection (second row tinted=%v): %+v", second, got.rows)
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
}

func TestOutputLayoutWideCoordinatesAndInvalidMetadata(t *testing.T) {
	o := renderedOutput{text: "  界 a\n", rows: []rowPaint{{tinted: true, exclusions: []cellRange{{2, 4}}}}, regions: []Region{{Line: 0, Col: 2, Width: 2, Text: "界"}}}
	got := layoutOutput(o, 20)
	if !reflect.DeepEqual(got.regions, o.regions) || !reflect.DeepEqual(got.rows[0], o.rows[0]) {
		t.Fatalf("wide display geometry changed: %+v", got)
	}
	// A bad background STRING was once a case here; since #70 a paint is only a
	// bit, so that input is unrepresentable rather than filtered.
	for _, rows := range [][]rowPaint{{{tinted: true, exclusions: []cellRange{{2, 1}}}}, {{}, {}, {}}} {
		o.rows = rows
		got = layoutOutput(o, 20)
		if got.text != o.text {
			t.Error("invalid metadata loses source")
		}
		for _, p := range got.rows {
			if p.tinted || len(p.exclusions) > 0 {
				t.Errorf("invalid metadata retained: %+v", p)
			}
		}
	}
}

func TestOutputSerializationKeepsSourceAndClipboardUnpadded(t *testing.T) {
	o := renderedOutput{text: "  hola\n\nfin\n", rows: []rowPaint{{tinted: true}, {tinted: true}, {tinted: true}}}
	got := serializeOutput(o, 20, store.SchemeDark)
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
	if serializeOutput(o, 0, store.SchemeDark) != o.text {
		t.Error("plain serialization adds padding")
	}
	f := newSelectionFrame(20, 3, []selectionRow{{styled: "  hola", selectable: true}, {styled: "", selectable: true}, {styled: "fin", selectable: true}})
	copied, err := selectedText(f, selectionPoint{row: 0, col: 0}, selectionPoint{row: 2, col: 19})
	if err != nil || copied != "  hola\n\nfin" {
		t.Fatalf("clipboard=%q, %v", copied, err)
	}
}

func TestOutputSerializationPreservesStylesAcrossRows(t *testing.T) {
	o := renderedOutput{text: "\x1b[31muno\ndos\x1b[0m", rows: []rowPaint{{tinted: true}, {tinted: true}}}
	lines := strings.Split(serializeOutput(o, 20, store.SchemeDark), "\n")
	for _, line := range lines {
		cells, _ := rowTestCells(t, line, 20)
		if cells[0].fg != 31 {
			t.Fatalf("producer foreground lost: %q", line)
		}
	}
}

func TestOutputLayoutWidthChangesDoNotPadStoredText(t *testing.T) {
	o := renderedOutput{text: "  one two three four five six", rows: []rowPaint{{tinted: true}}}
	narrow := layoutOutput(o, 20)
	if narrow.text != "  one two three four\n  five six" {
		t.Fatalf("narrow text=%q", narrow.text)
	}
	wide := layoutOutput(narrow, 40)
	if wide.text != narrow.text {
		t.Error("historical rows reflowed")
	}
	for _, width := range []int{20, 40, 20} {
		serialized := serializeOutput(wide, width, store.SchemeDark)
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
