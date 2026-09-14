package main

import (
	"context"
	"errors"
	"io"
	"sync"
)

// pointerRouter owns the active screen, before input enters a busy loop's queue.
// Lock order is router then screen. Clipboard IO and joins happen outside both.
type pointerRouter struct {
	mu               sync.Mutex
	active           *liveScreen
	owner            uint64
	clipboard        *clipboardQueue
	unwatchInterrupt func()
}

func newPointerRouter(l *liveScreen, w clipboardWriter) *pointerRouter {
	r := &pointerRouter{active: l, owner: 1}
	if w != nil {
		r.clipboard = newClipboardQueue(w)
	}
	return r
}

func (r *pointerRouter) route(k Key) (Key, bool) {
	r.mu.Lock()
	l := r.active
	if l == nil {
		r.mu.Unlock()
		return Key{}, false
	}
	l.mu.Lock()
	var click pointerClick
	var text string
	if isPointerKey(k.Kind) {
		event := selectionPress
		if k.Kind == KeyPointerMotion {
			event = selectionMotion
		}
		if k.Kind == KeyPointerRelease {
			event = selectionRelease
		}
		click, text = l.pointerLocked(event, selectionPoint{k.Row, k.Col})
		click.owner = r.owner
	} else if k.Kind != KeyUnknown {
		cancelPointerInput(l, k, true)
	}
	var seq uint64
	if text != "" {
		seq = l.copyStartedLocked()
	}
	owner := r.owner
	l.mu.Unlock()
	r.mu.Unlock()
	if text != "" {
		done := func(err error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.active != l || r.owner != owner {
				return
			}
			l.mu.Lock()
			defer l.mu.Unlock()
			l.copyFinishedLocked(seq, text, err)
		}
		if r.clipboard == nil {
			done(errors.New("clipboard unavailable"))
		} else if err := r.clipboard.Submit(text, done); err != nil {
			done(err)
		}
	}
	if isPointerKey(k.Kind) {
		if click.screen == nil {
			return Key{}, false
		}
		return Key{Kind: KeyClick, Row: k.Row, Col: k.Col, click: &click}, true
	}
	// Raw KeyClick input cannot invent a completed gesture.
	if k.Kind == KeyClick {
		return Key{}, false
	}
	return k, true
}

func (r *pointerRouter) resolve(k Key) (pointerClick, bool) {
	if r == nil || k.Kind != KeyClick || k.click == nil {
		return pointerClick{}, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil || k.click.screen != r.active || k.click.owner != r.owner {
		return pointerClick{}, false
	}
	r.active.mu.Lock()
	defer r.active.mu.Unlock()
	return r.active.resolvePointerLocked(*k.click)
}

func (r *pointerRouter) observeSize(sz winSize) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		r.active.mu.Lock()
		r.active.observeSelectionSizeLocked(sz.rows, sz.cols)
		r.active.mu.Unlock()
	}
}

func (r *pointerRouter) notice(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		r.active.mu.Lock()
		r.active.selectionNoticeLocked(text)
		r.active.mu.Unlock()
	}
}

func (r *pointerRouter) activate(l *liveScreen) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil {
		r.active.mu.Lock()
		r.active.cancelSelectionLocked(true)
		r.active.mu.Unlock()
	}
	r.owner++
	r.active = l
}

func (r *pointerRouter) Stop() {
	if r == nil {
		return
	}
	r.activate(nil)
	r.mu.Lock()
	unwatch := r.unwatchInterrupt
	r.unwatchInterrupt = nil
	r.mu.Unlock()
	if unwatch != nil {
		unwatch()
	}
	if r.clipboard != nil {
		r.clipboard.Stop()
	}
}

// readInput always decodes pointer and interrupt events even when type-ahead is
// saturated. Accepted ordinary keys remain FIFO; only the newest is refused.
func readInput(ctx context.Context, in io.Reader, interrupts *interrupter, router *pointerRouter) <-chan Key {
	if router != nil && interrupts != nil {
		router.watchInterrupts(interrupts)
	}
	out := make(chan Key, 256)
	go func() {
		defer close(out)
		var buf []byte
		chunk := make([]byte, 256)
		saturated := false
		for {
			if ctx.Err() != nil {
				return
			}
			n, err := in.Read(chunk)
			if n > 0 {
				buf = append(buf, chunk[:n]...)
				for len(buf) > 0 {
					if ctx.Err() != nil {
						return
					}
					k, used := decodeKey(buf)
					if used == 0 {
						break
					}
					buf = buf[used:]
					if k.Kind == KeyInterrupt && interrupts != nil && interrupts.Fire() {
						if router != nil {
							router.route(k)
						}
						continue
					}
					if !isPointerKey(k.Kind) && len(out) == cap(out) {
						if router != nil {
							router.cancelInput(k, false)
						}
						if !saturated && router != nil {
							router.notice("input full — newest key ignored")
						}
						saturated = true
						continue
					}
					if router != nil {
						var deliver bool
						k, deliver = router.route(k)
						if !deliver {
							continue
						}
					} else if isPointerKey(k.Kind) {
						continue
					}
					select {
					case out <- k:
						saturated = false
					case <-ctx.Done():
						return
					default:
						if !saturated && router != nil {
							router.notice("input full — newest key ignored")
						}
						saturated = true
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return out
}

// cancelPointerInput runs under the screen lock for every observed non-pointer
// input, including rejected type-ahead. Rejected input retains its overflow
// notice, but can never retain an unfinished gesture or a stale viewport ticket.
func cancelPointerInput(l *liveScreen, k Key, dismiss bool) {
	if k.Kind == KeyUnknown {
		return
	}
	l.cancelSelectionLocked(dismiss)
	switch k.Kind {
	case KeyPageUp, KeyPageDown, KeyWheelUp, KeyWheelDown, KeyInterrupt:
		l.invalidateSelectionLocked()
	}
}

func (r *pointerRouter) cancelInput(k Key, dismiss bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return
	}
	r.active.mu.Lock()
	defer r.active.mu.Unlock()
	cancelPointerInput(r.active, k, dismiss)
}

func (r *pointerRouter) watchInterrupts(i *interrupter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return
	}
	if r.unwatchInterrupt != nil {
		r.unwatchInterrupt()
	}
	r.unwatchInterrupt = i.Observe(func() { r.cancelInput(Key{Kind: KeyInterrupt}, true) })
}
