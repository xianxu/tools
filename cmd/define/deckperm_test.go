package main

import "testing"

// counter records how many times the question was actually put to the user, so
// "asked at most once" is a fact a test can assert rather than a hope.
type counter struct {
	allow bool
	calls int
}

func (c *counter) ask() bool { c.calls++; return c.allow }

// ONE DIRECTORY, ONE QUESTION. A lookup writes a word, an event and possibly a
// fact; asking three times for one intent is the shape this memo prevents.
func TestDeckPermissionAsksOnce(t *testing.T) {
	c := &counter{allow: true}
	p := newDeckPermission(c.ask)
	for i := 0; i < 5; i++ {
		if !p.allowed() {
			t.Fatal("allowed() = false, want true")
		}
	}
	if c.calls != 1 {
		t.Errorf("asked %d times, want exactly 1", c.calls)
	}
}

func TestDeckPermissionRemembersNo(t *testing.T) {
	c := &counter{allow: false}
	p := newDeckPermission(c.ask)
	for i := 0; i < 3; i++ {
		if p.allowed() {
			t.Fatal("allowed() = true after a decline")
		}
	}
	if c.calls != 1 {
		t.Errorf("asked %d times, want exactly 1 — a decline is an answer, not a "+
			"reason to keep asking", c.calls)
	}
}

// A NIL PERMISSION AND A NIL ASKER BOTH ALLOW.
//
// This is what keeps M1 a true no-op: every existing `deps{newStore: openStore}`
// in the suite passes no permission, and must behave exactly as it does today.
func TestDeckPermissionNilMeansAllow(t *testing.T) {
	var nilPerm *deckPermission
	if !nilPerm.allowed() {
		t.Error("a nil permission must allow — the wiring predates the policy")
	}
	if !newDeckPermission(nil).allowed() {
		t.Error("a nil asker must allow — there is no question to put")
	}
}

// saving() READS WITHOUT RESOLVING, and that is the whole reason it exists
// separately from allowed() (#50 PQ-6).
//
// --stats renders an empty screen and must say whether anything is being saved.
// If it learned that by calling allowed(), asking would become a side effect of
// READING — and `define --stats` in the wrong directory would prompt to create a
// deck it will never write to. That is the false alarm the lazy design exists to
// prevent, and false alarms train people to answer yes.
func TestSavingDoesNotResolve(t *testing.T) {
	c := &counter{allow: true}
	p := newDeckPermission(c.ask)

	allowed, decided := p.saving()
	if decided {
		t.Error("saving() reported a decision before one was made")
	}
	if allowed {
		t.Error("saving() reported allowed on an undecided permission")
	}
	if c.calls != 0 {
		t.Errorf("saving() put the question to the user %d time(s) — reading whether "+
			"anything is being saved must not ASK whether to start saving", c.calls)
	}
}

func TestSavingReportsTheAnswerOnceResolved(t *testing.T) {
	for _, tc := range []struct{ allow bool }{{true}, {false}} {
		c := &counter{allow: tc.allow}
		p := newDeckPermission(c.ask)
		p.resolve()
		allowed, decided := p.saving()
		if !decided {
			t.Errorf("allow=%v: saving() says undecided after resolve()", tc.allow)
		}
		if allowed != tc.allow {
			t.Errorf("allow=%v: saving() = %v", tc.allow, allowed)
		}
		if c.calls != 1 {
			t.Errorf("allow=%v: asked %d times", tc.allow, c.calls)
		}
	}
}

// resolve() IS IDEMPOTENT, because it is called once above the loop-shell choice
// and a write may still call allowed() afterwards.
func TestResolveIsIdempotent(t *testing.T) {
	c := &counter{allow: true}
	p := newDeckPermission(c.ask)
	p.resolve()
	p.resolve()
	p.allowed()
	if c.calls != 1 {
		t.Errorf("asked %d times, want 1", c.calls)
	}
}

// A nil permission must survive every entry point, not just allowed().
func TestNilPermissionSurvivesEveryOperation(t *testing.T) {
	var p *deckPermission
	p.resolve()
	allowed, decided := p.saving()
	if !allowed || !decided {
		t.Errorf("nil permission saving() = (%v, %v), want (true, true) — no policy "+
			"means saving, settled", allowed, decided)
	}
}
