package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRegionBackgroundCoversTrailingScreenCells(t *testing.T) {
	var out bytes.Buffer
	s := &screen{}
	s.Write([]byte("  hola\n"))
	s.paints = map[int]rowPaint{0: {background: languageDark}}
	s.Paint(&out, 4, 12, "", nil)
	if !strings.Contains(stripEscapes(out.String()), "hola      ") {
		t.Fatalf("row has no full-width paint: %q", out.String())
	}
}

func TestLiveFooterCarriesPaintAndWrappedAnswerExclusions(t *testing.T) {
	l := newLiveScreen(io.Discard, 8, 20)
	defer l.Stop()
	l.DrawOutput(renderedOutput{text: "keys"}, renderedOutput{text: "abcdefghijklmnopqrstuvwxy", rows: []rowPaint{{background: languageDark, exclusions: []cellRange{{21, 23}}}}})
	l.mu.Lock()
	defer l.mu.Unlock()
	frame := l.s.layoutSelectionFrame(8, 20, l.prompt, l.footer, "", "", false).frame
	first := l.s.footerTop
	if first+1 >= len(frame.rows) {
		t.Fatalf("missing wrapped footer: %+v", frame.rows)
	}
	for row := 0; row < 2; row++ {
		source := frame.rows[first+row]
		cells, _ := rowTestCells(t, paintLanguageRow(source.styled, source.paint, 20), 20)
		for col, c := range cells {
			want := 236
			if row == 1 && col >= 1 && col < 3 {
				want = -1
			}
			if c.bg != want {
				t.Fatalf("footer row %d col %d background=%d want=%d", row, col, c.bg, want)
			}
		}
	}
	if len(frame.cells[first+1]) != 5 {
		t.Fatal("footer paint padding became selectable")
	}
}

func TestNestedScreenTransfersUnpaddedMetadata(t *testing.T) {
	child := newLiveScreen(io.Discard, 8, 20)
	defer child.Stop()
	parent := newLiveScreen(io.Discard, 8, 30)
	defer parent.Stop()
	child.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{background: languageDark}}, regions: []Region{{Line: 0, Col: 0, Width: 4, Kind: RegionHeadword, Word: "hola"}}})
	child.Resize(8, 2)
	if child.Transcript() != "hola\n" {
		t.Fatalf("history was clipped/painted: %q", child.Transcript())
	}
	parent.WriteOutput(child.OutputTranscript())
	if parent.Transcript() != "hola\n" {
		t.Fatal("nested copy acquired paint padding")
	}
	cells, _ := rowTestCells(t, strings.Split(parent.PaintedTranscript(), "\n")[0], 30)
	for col, c := range cells {
		if c.bg != 236 {
			t.Fatalf("nested metadata lost col %d", col)
		}
	}
	if len(parent.s.regions[0]) != 1 || parent.s.regions[0][0].Word != "hola" {
		t.Fatal("nested actions lost")
	}
}
