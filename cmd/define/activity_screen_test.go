package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestActivityScreenGeometry(t *testing.T) {
	for _, pinned := range []bool{false, true} {
		for _, cols := range []int{1, 4, 20} {
			for _, rows := range []int{1, 2, 8} {
				var out bytes.Buffer
				live := newLiveScreen(&out, rows, cols)
				if pinned {
					live.s.pinned = true
					live.s.gap = chromeGap
				}
				live.interval = -1
				prompt := RenderLine(Editor{Line: []rune("ab"), Cursor: 2}, "", nil, false)
				live.Write([]byte("history\n"))
				live.Draw(prompt, []string{"menu"})
				lease, err := (&screenActivityHost{screen: live}).begin("⠋")
				if err != nil {
					t.Fatal(err)
				}
				frame := readFrame(t, lastFrame(out.String()), cols)
				if frame.rows > rows {
					t.Fatalf("%dx%d pinned=%v: overflow %+v", rows, cols, pinned, frame)
				}
				shown := clipVisible(strings.TrimPrefix(prompt, "\r"), rows*cols)
				want := readFrame(t, shown, cols)
				glyph := strings.Contains(lastFrame(out.String()), "⠋")
				if glyph != (displayRows(shown, cols) < rows) {
					t.Fatalf("%dx%d: glyph visibility %v", rows, cols, glyph)
				}
				if frame.cursorCol != want.cursorCol {
					t.Fatalf("cursor col=%d want %d", frame.cursorCol, want.cursorCol)
				}
				for row, line := range frame.text {
					if strings.Contains(line, "menu") {
						if _, _, ok := live.FooterRowAt(row); !ok {
							t.Fatalf("%dx%d pinned=%v footer unmapped row %d frame=%+v raw=%q", rows, cols, pinned, row, frame, lastFrame(out.String()))
						}
					}
					if strings.Contains(line, "⠋") {
						if _, ok := live.RegionAtRow(row, 0); ok {
							t.Fatal("glyph mapped to transcript")
						}
					}
				}
				lease.clear()
				if strings.Contains(lastFrame(out.String()), "⠋") {
					t.Fatal("clear retained glyph")
				}
				if live.Transcript() != "history\n" {
					t.Fatalf("transcript %q", live.Transcript())
				}
				live.Stop()
			}
		}
	}
}

func TestActivityScreenEmptyPromptAndOwnership(t *testing.T) {
	var out bytes.Buffer
	live := newLiveScreen(&out, 10, 20)
	live.interval = time.Hour
	host := &screenActivityHost{screen: live}
	old, err := host.begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	if g := readFrame(t, lastFrame(out.String()), 20); g.rows != 1 {
		t.Fatalf("empty prompt consumes extra row: %+v", g)
	}
	next, err := host.begin("⠙")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-old.revoked:
	default:
		t.Fatal("old lease not revoked")
	}
	old.set("⠹")
	old.clear()
	if !strings.Contains(lastFrame(out.String()), "⠙") {
		t.Fatal("stale lease erased new")
	}
	live.suspend()
	select {
	case <-next.revoked:
	default:
		t.Fatal("suspend did not revoke")
	}
	n := out.Len()
	next.set("⠸")
	live.flush()
	if out.Len() != n {
		t.Fatal("paint while suspended")
	}
	live.resume()
	if strings.Contains(lastFrame(out.String()), "⠙") {
		t.Fatal("resume resurrected")
	}
	last, err := host.begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	last.set("⠙")
	live.Stop()
	select {
	case <-last.revoked:
	default:
		t.Fatal("stop did not revoke")
	}
	n = out.Len()
	last.set("⠹")
	last.clear()
	live.flush()
	if out.Len() != n {
		t.Fatal("paint after stop")
	}
	if strings.Contains(lastFrame(out.String()), "⠋") || strings.Contains(lastFrame(out.String()), "⠙") {
		t.Fatal("stop retained activity")
	}
}

type activityBrokenWriter struct{ err error }

func (w activityBrokenWriter) Write(p []byte) (int, error) { return 0, w.err }
func TestActivityScreenWriteError(t *testing.T) {
	want := errors.New("terminal gone")
	live := newLiveScreen(activityBrokenWriter{want}, 10, 20)
	if _, err := (&screenActivityHost{screen: live}).begin("⠋"); !errors.Is(err, want) {
		t.Fatalf("begin error %v", err)
	}
	var out bytes.Buffer
	live = newLiveScreen(&out, 10, 20)
	live.interval = -1
	lease, err := (&screenActivityHost{screen: live}).begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	live.mu.Lock()
	live.tty = activityBrokenWriter{io.ErrClosedPipe}
	live.mu.Unlock()
	if err := lease.set("⠙"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("tick error %v", err)
	}
	lease.clear()
	live.Stop()
}

func TestActivityScreenPromptAllocation(t *testing.T) {
	for _, glyph := range []string{"", "⠋"} {
		var out bytes.Buffer
		s := &screen{}
		prompt := RenderLine(Editor{Line: []rune("ab"), Cursor: 2}, "", nil, false)
		s.paintActivity(&out, 4, 4, prompt, []string{"menu"}, glyph)
		frame := readFrame(t, out.String(), 4)
		wantPromptRow := 0
		if glyph != "" {
			wantPromptRow = 1
		}
		if frame.rows != wantPromptRow+2 || frame.cursorRow != wantPromptRow || frame.cursorCol != 4 {
			t.Fatalf("glyph=%q allocation/cursor %+v", glyph, frame)
		}
		if frame.text[wantPromptRow] != "› ab" || frame.text[wantPromptRow+1] != "menu" {
			t.Fatalf("chrome %+v", frame)
		}
		if glyph != "" && frame.text[0] != glyph {
			t.Fatalf("glyph overwritten: %+v", frame)
		}
		if _, _, ok := s.FooterRowAt(wantPromptRow + 1); !ok {
			t.Fatal("footer not mapped")
		}
	}
	var out bytes.Buffer
	live := newLiveScreen(&out, 4, 20)
	prompt := RenderLine(Editor{Line: []rune("ab"), Cursor: 1}, "cd", nil, false)
	live.Draw(prompt, nil)
	lease, err := (&screenActivityHost{screen: live}).begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	frame := readFrame(t, lastFrame(out.String()), 20)
	if frame.rows != 2 || frame.cursorRow != 1 || frame.cursorCol != 3 {
		t.Fatalf("suggestion cursor %+v", frame)
	}
	lease.clear()
	live.Stop()
}

func TestActivityScreenResizeScroll(t *testing.T) {
	var out bytes.Buffer
	live := newPinnedScreen(&out, 8, 20)
	live.interval = -1
	live.WriteRegions("one\ntwo\nthree\nfour\nfive\nsix\n", []Region{{Line: 0, Col: 0, Width: 3}})
	prompt := RenderLine(Editor{Line: []rune("ab"), Cursor: 2}, "", nil, false)
	live.Draw(prompt, []string{"menu"})
	lease, err := (&screenActivityHost{screen: live}).begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	live.Scroll(2)
	if !strings.Contains(lastFrame(out.String()), "⠋") {
		t.Fatal("scroll lost activity")
	}
	live.Resize(2, 2)
	live.Draw(prompt, []string{"menu"})
	if strings.Contains(lastFrame(out.String()), "⠋") {
		t.Fatal("activity stole full prompt rows")
	}
	frame := readFrame(t, lastFrame(out.String()), 2)
	if frame.rows != 2 {
		t.Fatalf("resize frame %+v", frame)
	}
	live.Resize(8, 20)
	live.Draw(prompt, []string{"menu"})
	if !strings.Contains(lastFrame(out.String()), "⠋") {
		t.Fatal("resize lost active lease")
	}
	lease.clear()
	live.Stop()
}

func TestActivityScreenRowPolicy(t *testing.T) {
	for _, tc := range []struct {
		prompt, glyph string
		rows, cols    int
		want          string
	}{
		{"", "⠋", 1, 1, "⠋"},
		{"", "⠋", 0, 1, ""},
		{"prompt", "", 3, 4, ""},
		{"abcd", "⠋", 1, 4, ""},
		{"abcd", "⠋", 2, 4, "⠋"},
		{"abcde", "⠋", 2, 4, ""},
		{"\r\x1b[Kabcd", "⠋", 2, 4, "⠋"},
	} {
		if got := activityRow(tc.prompt, tc.glyph, tc.rows, tc.cols); got != tc.want {
			t.Errorf("activityRow(%q,%q,%d,%d)=%q want %q", tc.prompt, tc.glyph, tc.rows, tc.cols, got, tc.want)
		}
	}
}
