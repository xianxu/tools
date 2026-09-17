package main

// All methods in this file require liveScreen.mu. The pointer router acquires
// its own lock first; no screen operation calls back into the router or waits
// for clipboard IO.

func (l *liveScreen) invalidateSelectionLocked() {
	l.frameID++
	l.framePublished = false
	l.gesture = selectionGesture{}
}

func (l *liveScreen) observeSelectionSizeLocked(rows, cols int) {
	if l.sizeObserved && l.observedRows == rows && l.observedCols == cols {
		return
	}
	if !l.sizeObserved && l.rows == rows && l.cols == cols {
		return
	}
	l.observedRows, l.observedCols, l.sizeObserved = rows, cols, true
	l.invalidateSelectionLocked()
}

func (l *liveScreen) cancelSelectionLocked(dismiss bool) {
	changed := l.gesture.selected || l.gesture.dragging
	l.gesture = selectionGesture{}
	if dismiss {
		changed = changed || l.selectionNotice != ""
		l.selectionNotice, l.failedCopy = "", ""
		l.copySeq++
	}
	if changed {
		l.repaint()
	}
}

func (l *liveScreen) pointerLocked(event selectionEvent, p selectionPoint) (pointerClick, string) {
	if l.stopped || l.suspended || !l.framePublished {
		l.gesture = selectionGesture{}
		return pointerClick{}, ""
	}
	if event == selectionPress {
		if p.row < 0 || p.row >= l.frame.height || p.col < 0 || p.col >= l.frame.width {
			return pointerClick{}, ""
		}
		retry := p.row >= 0 && p.row < len(l.frame.rows) && l.frame.rows[p.row].retry
		if !retry {
			before := l.frameID
			l.cancelSelectionLocked(true)
			// Dismissal can move text. Do not reinterpret this press against the
			// new layout; the next gesture starts from what is now visible.
			if before != l.frameID {
				return pointerClick{}, ""
			}
		}
	}
	var effect selectionEffect
	l.gesture, effect = selectionStep(l.gesture, event, p, l.frameID, l.frame.width, l.frame.height)
	if l.gesture.dragging && l.frame.err != nil {
		l.gesture = selectionGesture{}
		l.selectionNoticeLocked("Selection exceeds display limit")
		return pointerClick{}, ""
	}
	switch effect {
	case selectionClick:
		click := pointerClick{screen: l, frame: l.frameID, point: l.gesture.end}
		resolved, ok := l.resolvePointerLocked(click)
		if ok && resolved.retry {
			text := l.failedCopy
			l.cancelSelectionLocked(true)
			return pointerClick{}, text
		}
		return click, ""
	case selectionCopy:
		// A DRAG ACROSS A PASSAGE MARKS rather than copies (#67). The screen can
		// tell because the passage is buffer text with its own region kind on
		// every word — there is no separate "which rows are the passage" table to
		// keep in step.
		//
		// The reader is choosing what to ask about; taking the text to the
		// clipboard instead would answer a question they did not ask, and the
		// copy gesture is still there everywhere else.
		// The ANCHOR decides, not overlap: a drag that starts in an answer and
		// ends over the passage is a copy, and treating it as a mark would
		// silently swallow the copy the reader asked for.
		if a, b, ok := l.passageDragLocked(l.gesture.anchor, l.gesture.end); ok {
			l.gesture = selectionGesture{}
			l.repaint()
			return pointerClick{screen: l, frame: l.frameID, hasDrag: true, dragAnchor: a, dragEnd: b, line: a.row}, ""
		}
		text, err := selectedText(l.frame, l.gesture.anchor, l.gesture.end)
		if err != nil {
			l.gesture = selectionGesture{}
			l.selectionNoticeLocked("Selection exceeds copy limit")
			return pointerClick{}, ""
		}
		l.repaint()
		return pointerClick{}, text
	default:
		if l.gesture.dragging || event == selectionCancel {
			l.repaint()
		}
		return pointerClick{}, ""
	}
}

func (l *liveScreen) resolvePointerLocked(click pointerClick) (pointerClick, bool) {
	if click.screen != l || l.stopped || l.suspended || !l.framePublished || click.frame != l.frameID || l.frame.err != nil {
		return pointerClick{}, false
	}
	p := click.point
	if p.row < 0 || p.row >= len(l.frame.rows) || p.col < 0 || p.col >= l.frame.width {
		return pointerClick{}, false
	}
	row := l.frame.rows[p.row]
	click.line = -1
	if line, ok := l.s.LineAt(p.row); ok {
		click.line = line
	}
	click.footer, click.footerEntry, click.footerOffset, click.retry = row.footer, row.footerEntry, row.footerOffset, row.retry
	click.region, click.hasRegion = Region{}, false
	// The original region can extend beyond a paint-time clip. A wide glyph
	// omitted at the edge must not leave an action on its undrawn first cell.
	if p.col >= len(l.frame.cells[p.row]) {
		return click, true
	}
	for _, region := range row.regions {
		if p.col >= region.Col && p.col-region.Col < region.Width {
			click.region, click.hasRegion = region, true
			break
		}
	}
	return click, true
}

func (l *liveScreen) copyStartedLocked() uint64 {
	l.copySeq++
	return l.copySeq
}

func (l *liveScreen) copyFinishedLocked(seq uint64, text string, err error) {
	if seq != l.copySeq || l.stopped || l.suspended {
		return
	}
	if err != nil {
		l.failedCopy = text
		l.selectionNotice = "Copy failed — click to retry"
		l.repaint()
		return
	}
	l.failedCopy, l.selectionNotice = "", ""
	l.repaint()
}

func (l *liveScreen) selectionNoticeLocked(text string) {
	if l.stopped || l.suspended {
		return
	}
	// A different notice is a dismissal of the old retry affordance, not a
	// relabeling of its action. Pending clipboard feedback loses that ownership.
	l.failedCopy = ""
	l.copySeq++
	l.selectionNotice = text
	l.repaint()
}

// passageDragLocked reports a drag that began on a PASSAGE row, with both ends
// converted to absolute buffer coordinates.
//
// The anchor decides. Overlap across every covered row was the first rule, and it
// meant a drag starting in an answer and ending over the passage lost its copy
// and marked passage words instead — a gesture doing something the reader did not
// ask for.
//
// It returns COORDINATES rather than the words it covered, because which words a
// drag means is marksForDrag's answer and there must be exactly one of those.
func (l *liveScreen) passageDragLocked(a, b selectionPoint) (selectionPoint, selectionPoint, bool) {
	a, b = selectionOrdered(a, b)
	if a.row < 0 || a.row >= len(l.frame.rows) || !rowHasPassageWord(l.frame.rows[a.row]) {
		return a, b, false
	}
	toBuffer := func(p selectionPoint) selectionPoint {
		line, ok := l.s.LineAt(p.row)
		if !ok {
			return selectionPoint{row: -1, col: p.col}
		}
		return selectionPoint{row: line, col: p.col}
	}
	return toBuffer(a), toBuffer(b), true
}

func rowHasPassageWord(r selectionRow) bool {
	for _, x := range r.regions {
		if x.Kind == RegionPassageWord {
			return true
		}
	}
	return false
}
