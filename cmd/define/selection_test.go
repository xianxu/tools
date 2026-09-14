package main

import "testing"

func TestSelectionGestureExclusiveEffects(t *testing.T) {
	for mask := 0; mask < 256; mask++ {
		g := selectionGesture{}
		g, _ = selectionStep(g, selectionPress, selectionPoint{1, 1}, 7, 4, 3)
		moved := false
		for bit := 0; bit < 8; bit++ {
			p := selectionPoint{1, 1}
			if mask&(1<<bit) != 0 {
				p.col = 2
				moved = true
			}
			var effect selectionEffect
			g, effect = selectionStep(g, selectionMotion, p, 7, 4, 3)
			if effect != selectionNone {
				t.Fatal("motion produced action")
			}
		}
		var effect selectionEffect
		g, effect = selectionStep(g, selectionRelease, selectionPoint{1, 1}, 7, 4, 3)
		want := selectionClick
		if moved {
			want = selectionCopy
		}
		if effect != want {
			t.Fatalf("mask %d effect %v want %v", mask, effect, want)
		}
		_, effect = selectionStep(g, selectionRelease, selectionPoint{1, 1}, 7, 4, 3)
		if effect != selectionNone {
			t.Fatal("duplicate release action")
		}
	}
}
func TestSelectionGestureInvalidationAndClamping(t *testing.T) {
	g, _ := selectionStep(selectionGesture{}, selectionPress, selectionPoint{1, 1}, 3, 4, 3)
	g, e := selectionStep(g, selectionRelease, selectionPoint{100, -10}, 3, 4, 3)
	if e != selectionCopy || g.end != (selectionPoint{2, 0}) {
		t.Fatalf("clamp: %+v %v", g, e)
	}
	g, _ = selectionStep(g, selectionPress, selectionPoint{1, 1}, 3, 4, 3)
	g, e = selectionStep(g, selectionRelease, selectionPoint{1, 1}, 4, 4, 3)
	if e != selectionNone || g.active {
		t.Fatal("stale frame acted")
	}
	g, _ = selectionStep(g, selectionPress, selectionPoint{-1, 0}, 4, 4, 3)
	if g.active {
		t.Fatal("outside press active")
	}
}
func FuzzSelectionStep(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 2})
	f.Fuzz(func(t *testing.T, events []byte) {
		g := selectionGesture{}
		press := false
		drag := false
		for _, b := range events {
			ev := selectionEvent(b % 4)
			p := selectionPoint{int(b % 3), int(b % 4)}
			if ev == selectionPress {
				press = true
				drag = false
			}
			before := g
			var effect selectionEffect
			g, effect = selectionStep(g, ev, p, 1, 4, 3)
			if ev == selectionMotion && before.active && p != before.anchor {
				drag = true
			}
			if effect != selectionNone {
				if ev != selectionRelease || !press {
					t.Fatal("effect without live release")
				}
				if effect == selectionClick && drag {
					t.Fatal("drag clicked")
				}
				press = false
			}
			if ev == selectionCancel {
				press = false
			}
			if g.active && (g.end.row < 0 || g.end.row >= 3 || g.end.col < 0 || g.end.col >= 4) {
				t.Fatal("unbounded endpoint")
			}
		}
	})
}
