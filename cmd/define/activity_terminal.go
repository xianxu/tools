package main

import (
	"io"
	"sync"
)

// plainActivityHost owns the otherwise empty output line until clear. Callers
// stop synchronously before writing response text to the same terminal.
type plainActivityHost struct {
	out     io.Writer
	mu      sync.Mutex
	current *activityLease
}

func (h *plainActivityHost) begin(frame string) (*activityLease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	prefix := ""
	if h.current != nil {
		close(h.current.revoked)
		h.current = nil
		prefix = "\r"
	}
	lease := &activityLease{revoked: make(chan struct{})}
	lease.set = func(frame string) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.current != lease {
			return nil
		}
		return h.paint("\r" + frame)
	}
	lease.clear = func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.current != lease {
			return
		}
		h.current = nil
		close(lease.revoked)
		_, _ = io.WriteString(h.out, eraseLine)
	}
	if err := h.paint(prefix + frame); err != nil {
		close(lease.revoked)
		// A short/failed write may still have put part of the glyph on screen.
		_, _ = io.WriteString(h.out, eraseLine)
		return nil, err
	}
	h.current = lease
	return lease, nil
}

func (h *plainActivityHost) paint(text string) error {
	w := &activityPaintWriter{writer: h.out}
	_, err := io.WriteString(w, text)
	return err
}
