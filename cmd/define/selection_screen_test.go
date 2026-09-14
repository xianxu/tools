package main

import (
	"errors"
	"strings"
	"testing"
)

func TestSelectionLayoutMatchesPaintedRows(t *testing.T) {
	for _, pinned := range []bool{false, true} {
		for rows := 1; rows <= 12; rows++ {
			for cols := 4; cols <= 17; cols++ {
				var tty strings.Builder
				l := newLiveScreen(&tty, rows, cols)
				l.s.pinned = pinned
				if pinned {
					l.s.gap = chromeGap
				}
				l.s.Write([]byte("first\n\n\x1b[32msecond row\x1b[0m\n"))
				l.Draw("\r\x1b[K> typed text\x1b[2D", []string{"footer line"})
				got := readFrame(t, lastFrame(tty.String()), cols)
				if l.frame.err != nil {
					t.Fatal(l.frame.err)
				}
				for row, source := range l.frame.rows {
					if !source.selectable {
						continue
					}
					text, err := selectedText(l.frame, selectionPoint{row, 0}, selectionPoint{row, cols - 1})
					if err != nil {
						t.Fatal(err)
					}
					if strings.TrimRight(text, " ") != got.row(row) {
						t.Fatalf("pinned=%v shape=%dx%d row=%d selected %q painted %q", pinned, rows, cols, row, text, got.row(row))
					}
				}
				for row := 0; row < got.rows; row++ {
					if got.row(row) != "" && (row >= len(l.frame.rows) || !l.frame.rows[row].selectable) {
						t.Fatalf("painted text missing from selectable snapshot: row=%d text=%q", row, got.row(row))
					}
				}
				l.mu.Lock()
				l.pointerLocked(selectionPress, selectionPoint{0, 0})
				l.pointerLocked(selectionMotion, selectionPoint{rows - 1, cols - 1})
				l.mu.Unlock()
				highlighted := readFrame(t, lastFrame(tty.String()), cols)
				if got.cursorRow != highlighted.cursorRow || got.cursorCol != highlighted.cursorCol {
					t.Fatalf("highlight moved prompt cursor at %dx%d: %+v -> %+v", rows, cols, got, highlighted)
				}
				for row := 0; row < rows; row++ {
					if got.row(row) != highlighted.row(row) {
						t.Fatalf("highlight changed visible row %d: %q -> %q", row, got.row(row), highlighted.row(row))
					}
				}
			}
		}
	}
}

func TestLiveSelectionCopiesWithoutChangingTranscriptOrCursor(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 6, 12)
	l.s.Write([]byte("one two\nsecond\n"))
	l.Draw("> typed", []string{"footer"})
	before, transcript := readFrame(t, lastFrame(tty.String()), 12), l.Transcript()
	l.mu.Lock()
	l.pointerLocked(selectionPress, selectionPoint{0, 1})
	l.pointerLocked(selectionMotion, selectionPoint{1, 2})
	if !strings.Contains(lastFrame(tty.String()), "\x1b[7m") {
		t.Fatal("held drag not highlighted before release")
	}
	click, text := l.pointerLocked(selectionRelease, selectionPoint{1, 2})
	l.mu.Unlock()
	if click.screen != nil || text != "ne two\nsec" {
		t.Fatalf("click=%+v text=%q", click, text)
	}
	after := readFrame(t, lastFrame(tty.String()), 12)
	if before.cursorRow != after.cursorRow || before.cursorCol != after.cursorCol {
		t.Fatalf("cursor changed: %+v -> %+v", before, after)
	}
	if l.Transcript() != transcript {
		t.Fatal("highlight changed transcript")
	}
	if !strings.Contains(lastFrame(tty.String()), "\x1b[7m") {
		t.Fatal("drag not highlighted")
	}
}

func TestSelectionLayoutBudgetsWideGlyphSoftWraps(t *testing.T) {
	for _, rows := range []int{2, 3, 4} {
		var tty strings.Builder
		l := newLiveScreen(&tty, rows, 3)
		l.Draw("日日日", []string{"end"})
		got := readFrame(t, lastFrame(tty.String()), 3)
		if got.rows > rows {
			t.Fatalf("wide prompt overflowed %d rows: %+v", rows, got)
		}
		if l.frame.err != nil {
			t.Fatalf("ordinary wide text disabled frame: %v", l.frame.err)
		}
		for row, source := range l.frame.rows {
			if !source.selectable {
				continue
			}
			text, err := selectedText(l.frame, selectionPoint{row, 0}, selectionPoint{row, 2})
			if err != nil {
				t.Fatal(err)
			}
			if text != got.row(row) {
				t.Fatalf("row %d copy %q paint %q", row, text, got.row(row))
			}
		}
	}
}

func TestLiveSelectionTracksContentAndHitTargetsNotCosmeticFrames(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 8, 16)
	l.s.Write([]byte("hello\n"))
	l.s.regions = map[int][]Region{0: {{Text: "hello", Col: 0, Width: 5}}}
	l.Draw(">", []string{"footer wrapped across rows"})
	l.mu.Lock()
	l.pointerLocked(selectionPress, selectionPoint{0, 2})
	click, _ := l.pointerLocked(selectionRelease, selectionPoint{0, 2})
	resolved, ok := l.resolvePointerLocked(click)
	if !ok || !resolved.hasRegion || resolved.region.Text != "hello" {
		t.Fatal("published region lost")
	}
	id := l.frameID
	l.s.lines[0] = "\x1b[32mhello\x1b[0m"
	l.repaint()
	if l.frameID != id {
		t.Fatal("cosmetic SGR invalidated frame")
	}
	l.activityGlyph = "⠋"
	l.repaint()
	id = l.frameID
	l.activityGlyph = "⠙"
	l.repaint()
	if l.frameID != id {
		t.Fatal("activity tick invalidated frame")
	}
	l.s.regions[0][0].Text = "different action"
	l.repaint()
	if _, ok := l.resolvePointerLocked(click); ok {
		t.Fatal("changed target retained old ticket")
	}
	for row, source := range l.frame.rows {
		if !source.footer {
			continue
		}
		ticket := pointerClick{screen: l, frame: l.frameID, point: selectionPoint{row, 1}}
		target, ok := l.resolvePointerLocked(ticket)
		if !ok || !target.footer || target.footerEntry != 0 || target.footerOffset != source.footerOffset {
			t.Fatal("footer continuation lost")
		}
	}
	l.mu.Unlock()
}

func TestLiveSelectionInvalidationAndLifetime(t *testing.T) {
	for _, event := range []string{"write", "scroll", "page", "resize", "suspend", "stop"} {
		t.Run(event, func(t *testing.T) {
			var tty strings.Builder
			l := newLiveScreen(&tty, 4, 12)
			l.s.Write([]byte("first\nsecond\nthird\nfourth\n"))
			l.Draw(">", nil)
			l.mu.Lock()
			l.pointerLocked(selectionPress, selectionPoint{0, 0})
			l.pointerLocked(selectionMotion, selectionPoint{0, 2})
			l.mu.Unlock()
			switch event {
			case "write":
				l.Write([]byte("new"))
			case "scroll":
				l.Scroll(1)
			case "page":
				l.Page(1)
			case "resize":
				l.Resize(5, 12)
			case "suspend":
				l.suspend()
			case "stop":
				l.Stop()
			}
			l.mu.Lock()
			click, text := l.pointerLocked(selectionRelease, selectionPoint{0, 2})
			if click.screen != nil || text != "" || l.gesture.active || l.gesture.selected {
				t.Fatal("invalidated release had an effect")
			}
			l.mu.Unlock()
			l.Stop()
		})
	}
}

type selectionFailTerminal struct {
	strings.Builder
	fail bool
}

func (w *selectionFailTerminal) Write(p []byte) (int, error) {
	if w.fail {
		return 0, errors.New("terminal disconnected")
	}
	return w.Builder.Write(p)
}

func TestLiveSelectionRejectsUnpublishedAndChangedFrames(t *testing.T) {
	var tty selectionFailTerminal
	l := newLiveScreen(&tty, 5, 20)
	l.s.Write([]byte("clickable\n"))
	l.Draw(">", nil)
	l.mu.Lock()
	l.pointerLocked(selectionPress, selectionPoint{0, 1})
	click, _ := l.pointerLocked(selectionRelease, selectionPoint{0, 1})
	if _, ok := l.resolvePointerLocked(click); !ok {
		t.Fatal("fresh click rejected")
	}
	l.observeSelectionSizeLocked(5, 10)
	if _, ok := l.resolvePointerLocked(click); ok {
		t.Fatal("resize observation kept old ticket")
	}
	l.repaint()
	l.pointerLocked(selectionPress, selectionPoint{0, 0})
	if l.gesture.active {
		t.Fatal("old-sized paint accepted after observed resize")
	}
	l.mu.Unlock()
	l.Resize(5, 10)
	l.Draw(">", nil)
	l.mu.Lock()
	l.pointerLocked(selectionPress, selectionPoint{0, 0})
	if !l.gesture.active {
		t.Fatal("matching paint did not restore selection")
	}
	tty.fail = true
	l.repaint()
	if l.gesture.active {
		t.Fatal("failed paint retained gesture")
	}
	l.pointerLocked(selectionPress, selectionPoint{0, 0})
	if l.gesture.active {
		t.Fatal("failed paint published a selectable frame")
	}
	l.mu.Unlock()
}

func TestLiveSelectionFeedbackRetainsRetryPayloadAndRejectsStaleCompletion(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 1, 50)
	l.Draw("> preserved prompt", nil)
	l.mu.Lock()
	seq := l.copyStartedLocked()
	l.copyFinishedLocked(seq, "copy me", errors.New("failed"))
	if l.failedCopy != "copy me" || l.prompt != "> preserved prompt" {
		t.Fatal("failed copy or stored prompt lost")
	}
	if len(l.frame.rows) == 0 || !l.frame.rows[0].retry || l.frame.rows[0].selectable {
		t.Fatal("one-row retry target not drawn as decoration")
	}
	l.pointerLocked(selectionPress, selectionPoint{0, 1})
	click, text := l.pointerLocked(selectionRelease, selectionPoint{0, 1})
	if click.screen != nil || text != "copy me" {
		t.Fatalf("retry click=%+v text=%q", click, text)
	}
	newSeq := l.copyStartedLocked()
	l.copyFinishedLocked(seq, "old", errors.New("late"))
	if l.failedCopy != "" {
		t.Fatal("stale completion replaced feedback")
	}
	l.copyFinishedLocked(newSeq, text, nil)
	if l.failedCopy != "" || l.selectionNotice != "" {
		t.Fatal("success retained failure")
	}
	l.mu.Unlock()
	if !strings.Contains(lastFrame(tty.String()), "> preserved prompt") {
		t.Fatal("prompt not restored")
	}
}

func TestSelectionSnapshotRefusesOversizeWithoutLosingPaint(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 2, maxSelectionCells+1)
	l.Draw("still visible", nil)
	if !strings.Contains(tty.String(), "still visible") || l.frame.err == nil {
		t.Fatal("oversized snapshot must refuse while painting continues")
	}
	l.mu.Lock()
	l.pointerLocked(selectionPress, selectionPoint{0, 0})
	l.pointerLocked(selectionMotion, selectionPoint{0, 1})
	if !strings.Contains(l.selectionNotice, "limit") {
		t.Fatal("attempted oversized drag refused silently")
	}
	l.mu.Unlock()
}

func TestLiveSelectionRejectsClippedRegionAndOutsidePress(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 4, 3)
	l.s.Write([]byte("ab日\n"))
	l.s.regions = map[int][]Region{0: {{Col: 2, Width: 2, Text: "日"}}}
	l.Draw(">", nil)
	l.mu.Lock()
	defer l.mu.Unlock()
	target, ok := l.resolvePointerLocked(pointerClick{screen: l, frame: l.frameID, point: selectionPoint{0, 2}})
	if !ok || target.hasRegion {
		t.Fatal("clipped wide character retained clickable invisible cells")
	}
	l.pointerLocked(selectionPress, selectionPoint{0, 0})
	l.pointerLocked(selectionRelease, selectionPoint{0, 1})
	if !l.gesture.selected {
		t.Fatal("test did not establish selection")
	}
	l.pointerLocked(selectionPress, selectionPoint{-1, 0})
	if !l.gesture.selected {
		t.Fatal("outside press changed selection")
	}
}

func TestLiveSelectionGeneralNoticeDoesNotOfferOldCopyAsRetry(t *testing.T) {
	var tty strings.Builder
	l := newLiveScreen(&tty, 4, 60)
	l.Draw(">", nil)
	l.mu.Lock()
	defer l.mu.Unlock()
	seq := l.copyStartedLocked()
	l.copyFinishedLocked(seq, "old payload", errors.New("copy failed"))
	l.selectionNoticeLocked("Input buffer full")
	if l.failedCopy != "" {
		t.Fatal("unrelated notice retained stale retry payload")
	}
	for _, row := range l.frame.rows {
		if row.retry {
			t.Fatal("unrelated notice offers retry")
		}
	}
}
