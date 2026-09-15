package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func selectionWire(a, b selectionPoint) string {
	return fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<32;%d;%dM\x1b[<0;%d;%dm", a.col+1, a.row+1, b.col+1, b.row+1, b.col+1, b.row+1)
}

func selectionFind(l *liveScreen, text string) (selectionPoint, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for row, r := range l.frame.rows {
		plain := stripEscapes(r.styled)
		if i := strings.Index(plain, text); i >= 0 {
			return selectionPoint{row, visibleCells(plain[:i])}, true
		}
	}
	return selectionPoint{}, false
}

func TestSelectionEditorCopiesWhileModelIsBlocked(t *testing.T) {
	d, f, _, _ := askRig(t)
	started, release := make(chan struct{}), make(chan struct{})
	f.Script("", llmtest.Reply{Capture: streamCapture, Started: started, Release: release})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var tty syncBuf
	live := newLiveScreen(&tty, 24, 80)
	live.Write([]byte("copy this text\n"))
	board := newMemoryClipboard()
	router := newPointerRouter(live, board)
	defer router.Stop()
	in, write := io.Pipe()
	defer in.Close()
	defer write.Close()
	interrupts := &interrupter{}
	keys := readInput(ctx, in, interrupts, router)
	done := make(chan int, 1)
	go func() {
		done <- runEditor(ctx, keys, interrupts, d, options{tty: true, noAudio: true, width: 80}, console{view: live, pointer: router, stdout: live, stderr: live, finish: live.Stop})
	}()
	waitFor(t, func() bool { return strings.Contains(livePromptOf(live), "›") })
	io.WriteString(write, "?why\r")
	awaitActivity(t, started)
	a, ok := selectionFind(live, "copy this text")
	if !ok {
		t.Fatal("source text absent")
	}
	b := a
	b.col += 3
	io.WriteString(write, selectionWire(a, b))
	if copied := clipboardAwait(t, board.started); copied != "copy" {
		t.Fatalf("copied %q", copied)
	}
	select {
	case <-done:
		t.Fatal("editor ended before model release")
	default:
	}
	board.release <- nil
	close(release)
	cancel()
	clipboardAwait(t, done)
}

func TestSelectionPracticeDragDoesNotMarkBoard(t *testing.T) {
	d, opt, st := playRig(t, "keel", "mesa")
	q := play.NewBoard(boardCells("keel", "mesa"), 80, play.Palette{})
	_, held := questionsFor(t,d,opt)
	var tty syncBuf
	live := newPinnedScreen(&tty, 24, 80)
	board := newMemoryClipboard()
	router := newPointerRouter(live, board)
	defer router.Stop()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	in, write := io.Pipe()
	defer in.Close()
	defer write.Close()
	keys := readInput(ctx, in, &interrupter{}, router)
	done := make(chan int, 1)
	go func() {
		done <- playSession(ctx, d, opt, play.NewSession([]play.Question{q}), held, keys, console{view: live, pointer: router, stdout: live, stderr: live, finish: live.Stop})
	}()
	var a selectionPoint
	waitFor(t, func() bool { var ok bool; a, ok = selectionFind(live, "keel"); return ok })
	b := a
	b.col += 3
	io.WriteString(write, selectionWire(a, b))
	if copied := clipboardAwait(t, board.started); copied != "keel" {
		t.Fatalf("copied %q", copied)
	}
	board.release <- nil
	if n := len(reviewEvents(t, st)); n != 0 {
		t.Fatalf("drag recorded %d grades", n)
	}
	// A completed board scores unmarked cells Wrong. Dragging must not leave one Right.
	io.WriteString(write, "\r")
	waitFor(t, func() bool { return len(reviewEvents(t, st)) > 0 })
	for _, event := range reviewEvents(t, st) {
		if event.Correct {
			t.Fatal("drag marked a cell right")
		}
	}
	cancel()
	clipboardAwait(t, done)
}
