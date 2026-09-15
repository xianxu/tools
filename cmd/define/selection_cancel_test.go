package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSelectionCancellationBeforeAdmission(t *testing.T) {
	for _, wire := range []string{"x", "\x1b[5~", "\x1b[<64;1;1M", "\x03"} {
		for _, full := range []bool{false, true} {
			for _, scoped := range []bool{false, true} {
				t.Run(fmt.Sprintf("%q/full=%v/scoped=%v", wire, full, scoped), func(t *testing.T) {
					live := newLiveScreen(io.Discard, 5, 40)
					live.Draw("select me", nil)
					board := newMemoryClipboard()
					router := newPointerRouter(live, board)
					defer router.Stop()
					input, write := io.Pipe()
					defer input.Close()
					defer write.Close()
					interrupts := &interrupter{}
					if scoped {
						defer interrupts.Set(func() {})()
					}
					keys := readInput(t.Context(), input, interrupts, router)
					if full {
						io.WriteString(write, strings.Repeat("x", 257))
						waitFor(t, func() bool { live.mu.Lock(); defer live.mu.Unlock(); return live.selectionNotice != "" })
						// Queue remains full after dismissing the overflow warning. Start a fresh
						// drag on the new visible frame; the next rejected event must cancel it.
						router.route(Key{Kind: KeyPointerPress})
						router.route(Key{Kind: KeyPointerRelease})
					}

					router.route(Key{Kind: KeyPointerPress})
					router.route(Key{Kind: KeyPointerMotion, Col: 3})
					live.mu.Lock()
					dragging := live.gesture.dragging
					live.mu.Unlock()
					if !dragging {
						t.Fatal("premise: no active drag")
					}
					io.WriteString(write, wire+"\x1b[<0;4;1m")
					write.Close()
					for range keys {
					} // decoder EOF is the deterministic observation barrier
					selectionAssertNoCopy(t, router, board)
				})
			}
		}
	}
}

func TestSelectionScopedSignalCancelsBeforeForeground(t *testing.T) {
	signals := make(chan os.Signal)
	ctx, interrupts, cancel := detachedInterrupts(context.Background(), deps{notifySignals: func(...os.Signal) <-chan os.Signal { return signals }})
	defer cancel()
	defer close(signals)
	live := newLiveScreen(io.Discard, 5, 40)
	live.Draw("select me", nil)
	board := newMemoryClipboard()
	router := newPointerRouter(live, board)
	defer router.Stop()
	input, write := io.Pipe()
	defer input.Close()
	defer write.Close()
	readInput(ctx, input, interrupts, router)
	cancelled := make(chan struct{})
	restore := interrupts.Set(func() { close(cancelled) })
	defer restore()
	router.route(Key{Kind: KeyPointerPress})
	router.route(Key{Kind: KeyPointerMotion, Col: 3})
	signals <- os.Interrupt
	awaitActivity(t, cancelled)
	// The foreground has not repainted or finished. Signal observation alone
	// must make release inert, including its queued clipboard effect.
	router.route(Key{Kind: KeyPointerRelease, Col: 3})
	selectionAssertNoCopy(t, router, board)
}

func selectionAssertNoCopy(t *testing.T, router *pointerRouter, board *memoryClipboard) {
	t.Helper()
	if err := router.clipboard.Submit("barrier", nil); err != nil {
		t.Fatal(err)
	}
	if got := clipboardAwait(t, board.started); got != "barrier" {
		t.Fatalf("cancelled selection reached clipboard: %q", got)
	}
	board.release <- nil
	router.Stop()
	board.mu.Lock()
	defer board.mu.Unlock()
	if len(board.writes) != 1 {
		t.Fatalf("unexpected clipboard attempts: %v", board.writes)
	}
}
