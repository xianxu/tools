package main

// selectionPoint uses physical, zero-based terminal cells.
type selectionPoint struct{ row, col int }
type selectionEvent uint8

const (
	selectionPress selectionEvent = iota
	selectionMotion
	selectionRelease
	selectionCancel
)

type selectionEffect uint8

const (
	selectionNone selectionEffect = iota
	selectionClick
	selectionCopy
)

type selectionGesture struct {
	anchor, end                selectionPoint
	frame                      uint64
	active, dragging, selected bool
}

// pointerClick starts as an ownership ticket. Validation fills its immutable
// target while holding the active router and screen locks together.
type pointerClick struct {
	screen                    *liveScreen
	owner, frame              uint64
	point                     selectionPoint
	region                    Region
	hasRegion                 bool
	footerEntry, footerOffset int
	footer, retry             bool
	// line is the absolute BUFFER line the gesture landed on, or -1.
	//
	// A Region's own Line is relative to the render it came from, so it cannot
	// say which passage it belongs to — and a superseded passage's regions stay
	// in the map. Without the absolute line, clicking a word in an OLD passage
	// resolved against the current one and marked an unrelated word (#67, C-A:
	// measured alpha -> zulu).
	line int
	// dragAnchor and dragEnd are a drag's ends in absolute buffer coordinates,
	// set only when hasDrag. The marks a drag produces are derived by
	// marksForDrag, which is where the decided single-span rule lives.
	dragAnchor, dragEnd selectionPoint
	hasDrag             bool
}

// selectionStep owns the click/drag decision. A drag cannot become a click by
// returning to its anchor, and invalidation consumes its eventual release.
func selectionStep(g selectionGesture, event selectionEvent, p selectionPoint, frame uint64, width, height int) (selectionGesture, selectionEffect) {
	if event == selectionCancel || width <= 0 || height <= 0 {
		return selectionGesture{}, selectionNone
	}
	if event == selectionPress {
		g = selectionGesture{}
		if p.row >= 0 && p.row < height && p.col >= 0 && p.col < width {
			g = selectionGesture{anchor: p, end: p, frame: frame, active: true}
		}
		return g, selectionNone
	}
	if !g.active {
		return g, selectionNone
	}
	if g.frame != frame {
		return selectionGesture{}, selectionNone
	}
	p.row = max(0, min(p.row, height-1))
	p.col = max(0, min(p.col, width-1))
	if event != selectionMotion && event != selectionRelease {
		return g, selectionNone
	}
	g.end = p
	g.dragging = g.dragging || p != g.anchor
	if event == selectionMotion {
		return g, selectionNone
	}
	g.active = false
	g.selected = g.dragging
	if g.dragging {
		return g, selectionCopy
	}
	return g, selectionClick
}
