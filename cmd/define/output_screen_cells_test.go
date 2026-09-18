package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func screenOutputRows(t *testing.T, s *screen, width, count int, selected bool) [][]rowTestCell {
	t.Helper()
	layout := s.layoutSelectionFrame(8, width, "", nil, "", "", false)
	gesture := selectionGesture{}
	if selected {
		gesture = selectionGesture{selected: true, frame: 1, anchor: selectionPoint{0, 0}, end: selectionPoint{count - 1, width - 1}}
	}
	var out bytes.Buffer
	layout.paint(&out, gesture)
	body := strings.TrimPrefix(out.String(), cursorHome+eraseDown)
	lines := strings.Split(body, "\r\n")
	if len(lines) < count {
		t.Fatalf("paint has %d rows, need %d: %q", len(lines), count, out.String())
	}
	cells := make([][]rowTestCell, count)
	for i := range cells {
		var end rowTestCell
		cells[i], end = rowTestCells(t, lines[i], width)
		if end.bg != -1 {
			t.Fatalf("row %d leaks background to cursor movement", i)
		}
	}
	return cells
}

func TestOutputScreenFillSelectionAndResize(t *testing.T) {
	l := newLiveScreen(io.Discard, 8, 20)
	defer l.Stop()
	source := "  hola\n\nabcdefghijklmnop\n"
	if err := l.WriteOutput(renderedOutput{text: source, rows: []rowPaint{{tinted: true}, {tinted: true}, {tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{20, 10, 24} {
		l.Resize(8, width)
		for _, selected := range []bool{false, true} {
			cells := screenOutputRows(t, l.s, width, 3, selected)
			for row, line := range cells {
				for col, c := range line {
					if c.bg != 236 {
						t.Fatalf("width=%d selected=%v row=%d col=%d background=%d", width, selected, row, col, c.bg)
					}
				}
			}
			if selected && (!cells[0][0].inverse || cells[0][6].inverse || cells[1][0].inverse) {
				t.Fatalf("inverse should select source cells only: %+v", cells)
			}
		}
		frame := l.s.layoutSelectionFrame(8, width, "", nil, "", "", false).frame
		copied, err := selectedText(frame, selectionPoint{0, 0}, selectionPoint{2, width - 1})
		want := "  hola\n\n" + "abcdefghijklmnop"[:min(width, 16)]
		if err != nil || copied != want {
			t.Fatalf("width %d clipboard=%q (%v), want %q", width, copied, err, want)
		}
		if strings.Join(l.s.lines, "\n")+"\n" != source {
			t.Fatalf("resize mutated source: %q", l.s.lines)
		}
	}
}

func TestOutputScreenImmutablePolicyAndCurrentWidthTranscript(t *testing.T) {
	l := newLiveScreen(io.Discard, 8, 20)
	defer l.Stop()
	// Each row keeps the paint it was WRITTEN with — since #70 that is whether it
	// is tinted (its shade follows the scheme at paint, one for the whole screen),
	// so the second write and the producer's later mutation carry the other bit.
	o := renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}
	if err := l.WriteOutput(o); err != nil {
		t.Fatal(err)
	}
	o.rows[0].tinted = false
	if err := l.WriteOutput(renderedOutput{text: "hello\n", rows: []rowPaint{{}}}); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{12, 24} {
		l.Resize(8, width)
		lines := strings.Split(strings.TrimSuffix(l.PaintedTranscript(), "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("transcript rows=%d", len(lines))
		}
		for row, line := range lines {
			cells, _ := rowTestCells(t, line, width)
			want := 236
			if row == 1 {
				want = -1
			}
			for col, c := range cells {
				if c.bg != want {
					t.Fatalf("width %d row %d col %d bg=%d, want %d", width, row, col, c.bg, want)
				}
			}
		}
	}
}

func TestOutputScreenKeepsProducerStyleAcrossRows(t *testing.T) {
	l := newLiveScreen(io.Discard, 8, 20)
	defer l.Stop()
	if err := l.WriteOutput(renderedOutput{text: "\x1b[31muno\ndos\x1b[0m\n", rows: []rowPaint{{tinted: true}, {tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	cells := screenOutputRows(t, l.s, 20, 2, false)
	if cells[1][0].fg != 31 {
		t.Errorf("live second-row foreground=%d, want red", cells[1][0].fg)
	}
	lines := strings.Split(l.PaintedTranscript(), "\n")
	transcript, _ := rowTestCells(t, lines[1], 20)
	if transcript[0].fg != 31 {
		t.Errorf("transcript second-row foreground=%d, want red", transcript[0].fg)
	}
}

func TestOutputScreenSnapshotsAnswerExclusions(t *testing.T) {
	l := newLiveScreen(io.Discard, 8, 20)
	defer l.Stop()
	exclusions := []cellRange{{0, 1}}
	if err := l.WriteOutput(renderedOutput{text: "ab\n", rows: []rowPaint{{tinted: true, exclusions: exclusions}}}); err != nil {
		t.Fatal(err)
	}
	exclusions[0] = cellRange{1, 2}
	cells := screenOutputRows(t, l.s, 20, 1, false)
	if cells[0][0].bg != -1 || cells[0][1].bg != 236 {
		t.Fatalf("producer mutation recolored stored cells: %+v", cells[0][:2])
	}
}
