package main

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestSelectionMouseReportsPreserveGesturePhase(t *testing.T) {
	for _, tc := range []struct {
		wire string
		kind KeyKind
	}{
		{"\x1b[<0;3;2M", KeyPointerPress},
		{"\x1b[<32;3;2M", KeyPointerMotion},
		{"\x1b[<0;3;2m", KeyPointerRelease},
		{"\x1b[M #\"", KeyPointerPress},
		{"\x1b[M@#\"", KeyPointerMotion},
		{"\x1b[M##\"", KeyPointerRelease},
		{"\x1b[<1;3;2m", KeyUnknown},
		{"\x1b[<35;3;2M", KeyUnknown},
		{"\x1b[<128;3;2M", KeyUnknown},
	} {
		k, n := decodeKey([]byte(tc.wire))
		if n != len(tc.wire) || k.Kind != tc.kind {
			t.Errorf("%q: kind %v n%d want %v", tc.wire, k.Kind, n, tc.kind)
		}
		if tc.kind != KeyUnknown && (k.Row != 1 || k.Col != 2) {
			t.Errorf("wrong cell %+v", k)
		}
	}
}

func TestSelectionInputNeverWaitsBehindTypeahead(t *testing.T) {
	live := newLiveScreen(io.Discard, 4, 40)
	live.Draw("select this", nil)
	var board clipboardWriter
	router := newPointerRouter(live, board)
	defer router.Stop()
	input := strings.Repeat("x", 257) + "\x1b[<0;1;1M\x1b[<32;6;1M\x1b[<0;6;1m\x03"
	hit := make(chan struct{}, 1)
	interrupts := &interrupter{}
	interrupts.Set(func() { hit <- struct{}{} })
	keys := readInput(t.Context(), strings.NewReader(input), interrupts, router)
	select {
	case <-hit:
	case <-time.After(time.Second):
		t.Fatal("full typeahead stalled interrupt decoding")
	}
	for range keys {
	}
}

func TestSelectionRouterQueuesOnlyCompletedClicks(t *testing.T) {
	live := newLiveScreen(io.Discard, 4, 40)
	live.WriteRegions("hello", []Region{{Col: 0, Width: 5, Kind: RegionHeadword, Word: "hello"}})
	live.Draw("", nil)
	router := newPointerRouter(live, nil)
	defer router.Stop()
	if _, ok := router.route(Key{Kind: KeyPointerPress}); ok {
		t.Fatal("press escaped to loop")
	}
	k, ok := router.route(Key{Kind: KeyPointerRelease})
	if !ok || k.Kind != KeyClick {
		t.Fatal("release did not produce click")
	}
	if hit, ok := router.resolve(k); !ok || !hit.hasRegion || hit.region.Word != "hello" {
		t.Fatalf("wrong hit: %+v, %v", hit, ok)
	}
	live.Write([]byte("changed"))
	live.Draw("", nil)
	if _, ok := router.resolve(k); ok {
		t.Fatal("stale click accepted after repaint")
	}
}
