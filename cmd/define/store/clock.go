package store

import "time"

// Clock is injected everywhere a time is stamped.
//
// Not a convenience: every consumer of this store is date-driven — "which words
// are due today" is the whole of #5 — and a wall clock makes that untestable.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// SystemClock is the real one. Tests pass a fake.
func SystemClock() Clock { return systemClock{} }

// FixedClock returns a Clock frozen at t, for tests and for reproducing a day.
func FixedClock(t time.Time) Clock { return fixed{t} }

type fixed struct{ t time.Time }

func (f fixed) Now() time.Time { return f.t }
