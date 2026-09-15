package main

import (
	"io"
	"strings"
)

// activityRow grants decoration only after the original prompt has its rows.
// It returns the glyph separately: RenderLine owns erase and cursor controls.
func activityRow(prompt, glyph string, rows, cols int) string {
	if glyph == "" || rows <= 0 {
		return ""
	}
	if prompt == "" {
		return glyph
	}
	prompt = clipVisible(strings.TrimPrefix(prompt, "\r"), rows*max(cols, 1))
	if displayRows(prompt, cols) >= rows {
		return ""
	}
	return glyph
}

type screenActivityHost struct{ screen *liveScreen }

func (h *screenActivityHost) begin(glyph string) (*activityLease, error) {
	l := h.screen
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.stopped || l.suspended || l.tty == nil {
		return nil, io.ErrClosedPipe
	}
	if l.paintErr != nil {
		return nil, l.paintErr
	}
	l.revokeActivity()
	lease := &activityLease{revoked: make(chan struct{})}
	l.activity, l.activityGlyph = lease, glyph
	lease.set = func(glyph string) error {
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.activity != lease {
			return nil
		}
		if l.paintErr != nil {
			return l.paintErr
		}
		l.activityGlyph = glyph
		l.throttledPaint()
		return l.paintErr
	}
	lease.clear = func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.activity != lease {
			return
		}
		l.revokeActivity()
		l.repaint()
	}
	l.repaint()
	if l.paintErr != nil {
		l.revokeActivity()
		return nil, l.paintErr
	}
	return lease, nil
}

// revokeActivity never waits for a worker; its caller holds the screen lock.
func (l *liveScreen) revokeActivity() {
	if l.activity != nil {
		close(l.activity.revoked)
		l.activity = nil
	}
	l.activityGlyph = ""
}
