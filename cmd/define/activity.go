package main

import (
	"context"
	"io"
	"sync"
	"time"
)

// activityHost owns presentation and serializes leases. Replacing a lease
// revokes its worker; a stale lease must never draw or clear a newer one.
type activityHost interface {
	begin(string) (*activityLease, error)
}

type activityLease struct {
	revoked chan struct{}
	set     func(string) error
	clear   func()
}

func activityFrame(index int) string {
	const frames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	index %= 10
	if index < 0 {
		index += 10
	}
	return frames[index*3 : (index+1)*3]
}

func activityEnabled(opt options) bool { return opt.tty && !opt.raw }

// startActivity draws immediately, including time spent discovering a model.
// Stop waits for both animation and clearing before the caller writes output.
func startActivity(ctx context.Context, host activityHost) func() {
	return startActivityTicks(ctx, host, func() (<-chan time.Time, func()) {
		ticker := time.NewTicker(80 * time.Millisecond)
		return ticker.C, ticker.Stop
	})
}

func startActivityTicks(ctx context.Context, host activityHost, newTicker func() (<-chan time.Time, func())) func() {
	noop := func() {}
	if host == nil || ctx.Err() != nil {
		return noop
	}
	lease, err := host.begin(activityFrame(0))
	if err != nil {
		return noop
	}
	ticks, stopTicker := newTicker()
	stopped := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer lease.clear()
		defer stopTicker()
		for index := 1; ; {
			select {
			case <-stopped:
				return
			case <-ctx.Done():
				return
			case <-lease.revoked:
				return
			case <-ticks:
				// A tick racing cleanup may be selected once; lease identity and the
				// synchronous stop join keep it from surviving into response output.
				if err := lease.set(activityFrame(index)); err != nil {
					return
				}
				index = (index + 1) % 10
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(stopped) }); <-done }
}

// activityPaintWriter records the first terminal error, including short writes.
// Both hosts use it so animation stops even when a writer omits its error.
type activityPaintWriter struct {
	writer io.Writer
	err    error
}

func (w *activityPaintWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.writer.Write(p)
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}
