package main

import (
	"context"
	"errors"
	"io"
	"testing"
)

// TestSelectionNestedSittingBorrowsInputAndClipboard exercises the actual nested
// entrypoint with one decoder and one queue across both ownership transfers.
func TestSelectionNestedSittingBorrowsInputAndClipboard(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	questions, _ := questionsFor(t, d, opt)
	if len(questions) == 0 {
		t.Fatal("empty sitting cannot exercise ownership")
	}
	var tty syncBuf
	parent := newLiveScreen(&tty, 24, 80)
	defer parent.Stop()
	parent.WriteRegions("parent source\n", []Region{{Col: 0, Width: 6, Kind: RegionHeadword, Word: "parent"}})
	parent.Draw("› ", nil)
	board := newMemoryClipboard()
	router := newPointerRouter(parent, board)
	defer router.Stop()
	originalQueue := router.clipboard
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	in, write := io.Pipe()
	defer in.Close()
	defer write.Close()
	interrupts := &interrupter{}
	keys := readInput(ctx, in, interrupts, router)
	a, ok := selectionFind(parent, "parent")
	if !ok {
		t.Fatal("parent text absent")
	}
	// Keep a real decoded release ticket across the nested handoff.
	if _, err := io.WriteString(write, selectionWire(a, a)); err != nil {
		t.Fatal(err)
	}
	parentTicket := clipboardAwait(t, keys)
	if hit, ok := router.resolve(parentTicket); !ok || !hit.hasRegion || hit.region.Word != "parent" {
		t.Fatalf("parent ticket was never valid: %+v %v", hit, ok)
	}
	b := a
	b.col += 5
	if _, err := io.WriteString(write, selectionWire(a, b)); err != nil {
		t.Fatal(err)
	}
	if got := clipboardAwait(t, board.started); got != "parent" {
		t.Fatalf("parent copied %q", got)
	}
	done := make(chan int, 1)
	go func() {
		code, _ := sittingInPlace(ctx, d, opt, keys, interrupts, parent, nil, &tty, io.Discard, router)
		done <- code
	}()
	var nested *liveScreen
	waitFor(t, func() bool {
		router.mu.Lock()
		defer router.mu.Unlock()
		nested = router.active
		return nested != nil && nested != parent
	})
	var nestedStart selectionPoint
	waitFor(t, func() bool { var found bool; nestedStart, found = selectionFind(nested, "sycophantic"); return found })
	if _, ok := router.resolve(parentTicket); ok {
		t.Fatal("parent ticket survived nested entry")
	}
	if router.clipboard != originalQueue {
		t.Fatal("nested sitting replaced shared clipboard queue")
	}
	nestedEnd := nestedStart
	nestedEnd.col += 2
	if _, err := io.WriteString(write, selectionWire(nestedStart, nestedEnd)); err != nil {
		t.Fatal(err)
	}
	// The parent write remains first. Its late failure must not draw over the
	// nested screen; the next write starting proves its callback has completed.
	waitFor(t, func() bool { return len(originalQueue.pending) == 1 })
	nestedPaint := tty.String()
	board.release <- errors.New("parent write failed after handoff")
	if got := clipboardAwait(t, board.started); got != "syc" {
		t.Fatalf("nested copied %q", got)
	}
	if tty.String() != nestedPaint {
		t.Fatal("late parent completion wrote to nested terminal")
	}
	nested.mu.Lock()
	notice := nested.selectionNotice
	nested.mu.Unlock()
	if notice != "" {
		t.Fatalf("parent completion painted nested notice %q", notice)
	}
	if len(reviewEvents(t, st)) != 0 {
		t.Fatal("nested drag graded a question")
	}
	// Ctrl-C is decoded by the borrowed reader and scoped to the sitting.
	if _, err := io.WriteString(write, "\x03"); err != nil {
		t.Fatal(err)
	}
	if code := clipboardAwait(t, done); code != 0 {
		t.Fatalf("sitting exit=%d", code)
	}
	router.mu.Lock()
	active, queue := router.active, router.clipboard
	router.mu.Unlock()
	if active != parent || queue != originalQueue {
		t.Fatal("sitting did not restore parent ownership and queue")
	}
	if _, ok := router.resolve(parentTicket); ok {
		t.Fatal("old parent ticket revived after handback")
	}
	// The same decoder accepts a fresh parent selection while the nested copy
	// remains in flight. Its start is a barrier after that stale completion.
	a, ok = selectionFind(parent, "parent")
	if !ok {
		t.Fatal("parent source lost on handback")
	}
	b = a
	b.col += 5
	if _, err := io.WriteString(write, selectionWire(a, b)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return len(originalQueue.pending) == 1 })
	parentPaint := tty.String()
	board.release <- errors.New("nested write failed after handback")
	if got := clipboardAwait(t, board.started); got != "parent" {
		t.Fatalf("restored parent copied %q", got)
	}
	if tty.String() != parentPaint {
		t.Fatal("late nested completion wrote to restored parent terminal")
	}
	parent.mu.Lock()
	notice = parent.selectionNotice
	failed := parent.failedCopy
	parent.mu.Unlock()
	if notice != "" || failed != "" {
		t.Fatalf("nested completion leaked into parent: notice=%q failed=%q", notice, failed)
	}
	board.release <- nil
}
