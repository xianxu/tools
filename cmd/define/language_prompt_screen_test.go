package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Literal physical rows and wire controls are the oracle here: the older
// terminal simulator also uses rune widths and cannot detect a split RI pair.
func TestLanguagePromptPinnedLayoutAndSelection(t *testing.T) {
	var out bytes.Buffer
	live := newPinnedScreen(&out, 6, 6)
	live.s.Write([]byte("one\n"))
	live.s.addRegions([]Region{{Line: 0, Col: 0, Width: 4, Text: "casa", Word: "casa", Lang: "es"}})
	live.s.Write([]byte("casa\n"))
	editor := Editor{Line: []rune("a"), Cursor: 1}
	prefix := languagePrompt("es", true, 6)
	rendered := RenderLine(editor, "bc", nil, false, prefix)
	layout := live.s.layoutSelectionFrame(6, 6, rendered, []string{"abc"}, "", "", false)
	if layout.frame.err != nil {
		t.Fatal(layout.frame.err)
	}
	var rows []string
	for _, row := range layout.frame.rows {
		rows = append(rows, stripEscapes(row.styled))
	}
	want := []string{"one", "casa", "", "🇪🇸 › a", "bc", "abc"}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("physical rows = %q, want %q", rows, want)
	}
	if live.s.footerTop != 5 || !layout.frame.rows[5].footer {
		t.Fatalf("footer row = %d", live.s.footerTop)
	}
	layout.paint(&out, selectionGesture{})
	// The two-row prompt and one-row footer require two cursor-up steps;
	// replaying the prompt then backs over exactly the two completion cells.
	if !strings.HasSuffix(out.String(), "\x1b[2A\r\x1b[K🇪🇸 › abc\x1b[2D") {
		t.Fatalf("cursor restoration = %q", out.String())
	}
	for _, col := range []int{0, 1} {
		p := selectionPoint{3, col}
		got, err := selectedText(layout.frame, p, p)
		if err != nil || got != "🇪🇸" {
			t.Fatalf("prompt flag cell %d = %q: %v", col, got, err)
		}
	}
	got, err := selectedText(layout.frame, selectionPoint{1, 0}, selectionPoint{1, 3})
	if err != nil || got != "casa" {
		t.Fatalf("definition copied = %q: %v", got, err)
	}
	if region, ok := live.s.RegionAt(1, 2); !ok || region.Word != "casa" || region.Lang != "es" {
		t.Fatalf("definition action = %+v, %v", region, ok)
	}
	if len(layout.frame.rows[1].regions) != 1 || layout.frame.rows[1].regions[0].Word != "casa" {
		t.Fatalf("definition snapshot lost action: %+v", layout.frame.rows[1])
	}
}

func TestLanguagePromptPinnedResizeRestoresHistoricalFlag(t *testing.T) {
	var out bytes.Buffer
	live := newPinnedScreen(&out, 10, 12)
	live.s.Write([]byte("🇪🇸 › casa\n"))
	for _, tc := range []struct {
		cols       int
		historical string
		promptRows []string
	}{
		{12, "🇪🇸 › casa", []string{"🇪🇸 › "}},
		{1, "", []string{"[", "e", "s", "]", " ", "›", " "}},
		{12, "🇪🇸 › casa", []string{"🇪🇸 › "}},
	} {
		rendered := RenderLine(NewEditor(), "", nil, false, languagePrompt("es", true, tc.cols))
		layout := live.s.layoutSelectionFrame(10, tc.cols, rendered, nil, "", "", false)
		if layout.frame.err != nil {
			t.Fatalf("width %d: %v", tc.cols, layout.frame.err)
		}
		if len(layout.frame.rows) != 10 {
			t.Fatalf("width %d rows = %d", tc.cols, len(layout.frame.rows))
		}
		if got := stripEscapes(layout.frame.rows[0].styled); got != tc.historical {
			t.Fatalf("width %d history = %q, want %q", tc.cols, got, tc.historical)
		}
		var got []string
		for _, row := range layout.frame.rows[10-len(tc.promptRows):] {
			got = append(got, stripEscapes(row.styled))
		}
		if !reflect.DeepEqual(got, tc.promptRows) {
			t.Fatalf("width %d prompt rows = %q, want %q", tc.cols, got, tc.promptRows)
		}
		if tc.cols == 1 {
			for _, row := range layout.frame.cells {
				for _, cell := range row {
					if strings.ContainsAny(cell.text, "🇪🇸") {
						t.Fatalf("width 1 retained a flag cell: %q", cell.text)
					}
				}
			}
		} else {
			p := selectionPoint{0, 1}
			copied, err := selectedText(layout.frame, p, p)
			if err != nil || copied != "🇪🇸" {
				t.Fatalf("restored historical copy = %q: %v", copied, err)
			}
		}
		if got := live.s.Transcript(); got != "🇪🇸 › casa\n" {
			t.Fatalf("resize changed transcript: %q", got)
		}
	}
}
