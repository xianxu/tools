package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

type lifecycleHost struct {
	mu      sync.Mutex
	lease   *activityLease
	frames  []string
	cleared chan struct{}
	fail    bool
}

func (h *lifecycleHost) begin(frame string) (*activityLease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.frames = append(h.frames, frame)
	l := &activityLease{revoked: make(chan struct{})}
	h.lease = l
	l.set = func(s string) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.frames = append(h.frames, s)
		if h.fail {
			return errors.New("write failed")
		}
		return nil
	}
	l.clear = func() { close(h.cleared) }
	return l, nil
}
func TestActivityFramesAndEligibility(t *testing.T) {
	for i := 0; i < 100; i++ {
		if activityFrame(i) != activityFrame(i+10) || utf8.RuneCountInString(activityFrame(i)) != 1 {
			t.Fatalf("invalid frame %d", i)
		}
	}
	for _, tc := range []struct{ tty, raw, want bool }{{false, false, false}, {false, true, false}, {true, false, true}, {true, true, false}} {
		if got := activityEnabled(options{tty: tc.tty, raw: tc.raw}); got != tc.want {
			t.Fatalf("eligibility %+v = %v", tc, got)
		}
	}
}
func TestActivityDisabledDoesNotStartTicker(t *testing.T) {
	startActivityTicks(context.Background(), nil, func() (<-chan time.Time, func()) { t.Fatal("started disabled ticker"); return nil, nil })()
}
func TestActivityLifecycle(t *testing.T) {
	for _, reason := range []string{"stop", "cancel", "revoke", "write error"} {
		t.Run(reason, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h := &lifecycleHost{cleared: make(chan struct{})}
			ticks := make(chan time.Time)
			tickerStopped := make(chan struct{})
			stop := startActivityTicks(ctx, h, func() (<-chan time.Time, func()) { return ticks, func() { close(tickerStopped) } })
			h.mu.Lock()
			if len(h.frames) != 1 || h.frames[0] != "⠋" {
				t.Fatalf("initial frames %v", h.frames)
			}
			h.mu.Unlock()
			switch reason {
			case "stop":
				stop()
			case "cancel":
				cancel()
			case "revoke":
				close(h.lease.revoked)
			case "write error":
				h.mu.Lock()
				h.fail = true
				h.mu.Unlock()
				ticks <- time.Now()
			}
			select {
			case <-h.cleared:
			case <-time.After(time.Second):
				t.Fatal("activity not cleared")
			}
			stop()
			stop()
			select {
			case <-tickerStopped:
			default:
				t.Fatal("ticker not stopped")
			}
		})
	}
}

func TestActivityTickAdvancesAndStopJoins(t *testing.T) {
	h := &lifecycleHost{cleared: make(chan struct{})}
	ticks := make(chan time.Time)
	stop := startActivityTicks(context.Background(), h, func() (<-chan time.Time, func()) { return ticks, func() {} })
	ticks <- time.Now()
	// Sending on the unbuffered tick channel guarantees the worker entered
	// its write branch before stop waits for cleanup.
	stop()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.frames) != 2 || h.frames[1] != "⠙" {
		t.Fatalf("frames %v", h.frames)
	}
	select {
	case ticks <- time.Now():
		t.Fatal("worker still receiving ticks")
	default:
	}
}

func FuzzActivityFrame(f *testing.F) {
	f.Add(0)
	f.Add(-1)
	f.Add(123456)
	f.Fuzz(func(t *testing.T, index int) {
		frame := activityFrame(index)
		if utf8.RuneCountInString(frame) != 1 {
			t.Fatalf("not one glyph: %q", frame)
		}
		r, _ := utf8.DecodeRuneInString(frame)
		if r < 0x2800 || r > 0x28ff {
			t.Fatalf("not Braille: %q", frame)
		}
		if activityFrame(index%10) != frame {
			t.Fatalf("non-periodic frame %d", index)
		}
	})
}

func TestActivityConcurrentStop(t *testing.T) {
	h := &lifecycleHost{cleared: make(chan struct{})}
	stop := startActivityTicks(context.Background(), h, func() (<-chan time.Time, func()) { return make(chan time.Time), func() {} })
	var callers sync.WaitGroup
	for i := 0; i < 32; i++ {
		callers.Add(1)
		go func() { defer callers.Done(); stop() }()
	}
	callers.Wait()
	select {
	case <-h.cleared:
	default:
		t.Fatal("stop returned before clearing")
	}
}

func TestActivityAlreadyCanceledDoesNotDraw(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := &lifecycleHost{cleared: make(chan struct{})}
	startActivityTicks(ctx, h, func() (<-chan time.Time, func()) { t.Fatal("started canceled activity"); return nil, nil })()
	if len(h.frames) != 0 {
		t.Fatalf("canceled activity drew %v", h.frames)
	}
}
