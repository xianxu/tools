package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSelectionCellsWholeCharacters(t *testing.T) {
	frame := newSelectionFrame(8, 4, []selectionRow{{styled: "\x1b[32mA界e\u0301  z\x1b[0m", selectable: true}, {selectable: false}, {styled: "", selectable: true}, {styled: "end", selectable: true}})
	for _, col := range []int{1, 2} {
		got, err := selectedText(frame, selectionPoint{0, col}, selectionPoint{0, col})
		if err != nil || got != "界" {
			t.Fatalf("half wide: %q %v", got, err)
		}
	}
	got, err := selectedText(frame, selectionPoint{3, 2}, selectionPoint{0, 3})
	if err != nil || got != "e\u0301  z\n\nend" {
		t.Fatalf("backward rows: %q %v", got, err)
	}
	clipped := newSelectionFrame(2, 1, []selectionRow{{styled: "A界", selectable: true}})
	got, _ = selectedText(clipped, selectionPoint{}, selectionPoint{0, 1})
	if got != "A" {
		t.Fatalf("clipped wide: %q", got)
	}
}
func TestSelectionFrameBoundsAndIdentity(t *testing.T) {
	for _, f := range []selectionFrame{newSelectionFrame(int(^uint(0)>>1), 2, nil), newSelectionFrame(1, 1, []selectionRow{{styled: strings.Repeat("x", maxSelectionSource+1)}})} {
		if f.err == nil {
			t.Fatal("unbounded frame accepted")
		}
	}
	a := newSelectionFrame(3, 2, []selectionRow{{styled: "\x1b[31mabc", selectable: true}, {styled: "⠋"}})
	b := newSelectionFrame(3, 2, []selectionRow{{styled: "\x1b[32mabc", selectable: true}, {styled: "⠙"}})
	if !a.same(b) {
		t.Fatal("cosmetic frame invalidates")
	}
	b.rows[0].footer = true
	if a.same(b) {
		t.Fatal("hit metadata ignored")
	}
}
func TestSelectionHighlightPreservesTextAndStyle(t *testing.T) {
	row := "\x1b[32ma\x1b[0m界e\u0301\x1b[2D"
	frame := newSelectionFrame(5, 1, []selectionRow{{styled: row, selectable: true}})
	got := frame.highlightRow(0, selectionPoint{0, 2}, selectionPoint{0, 3})
	if stripEscapes(got) != stripEscapes(row) || !strings.Contains(got, "\x1b[7m") || !strings.HasSuffix(got, "\x1b[2D") {
		t.Fatalf("highlight: %q", got)
	}
}
func FuzzSelectionText(f *testing.F) {
	f.Add("a界e\u0301", uint8(0), uint8(3))
	f.Fuzz(func(t *testing.T, text string, a, b uint8) {
		if len(text) > 4096 || !utf8.ValidString(text) || strings.ContainsAny(text, "\x1b\r\n") {
			return // outside the generated physical-row input domain
		}
		frame := newSelectionFrame(256, 1, []selectionRow{{styled: text, selectable: true}})
		got, err := selectedText(frame, selectionPoint{0, int(a)}, selectionPoint{0, int(b)})
		if err != nil {
			t.Fatal(err)
		}
		if !utf8.ValidString(got) {
			t.Fatal("split UTF-8")
		}
		back, _ := selectedText(frame, selectionPoint{0, int(b)}, selectionPoint{0, int(a)})
		if got != back {
			t.Fatal("direction changes text")
		}
		decorated := newSelectionFrame(256, 1, []selectionRow{{styled: "\x1b[31m" + text + "\x1b[0m", selectable: true}})
		other, _ := selectedText(decorated, selectionPoint{0, int(a)}, selectionPoint{0, int(b)})
		if other != got {
			t.Fatal("ANSI changes text")
		}
	})
}
