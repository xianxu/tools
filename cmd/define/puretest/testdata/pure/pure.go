// Package pure exists to be ACCEPTED: it imports one allowlisted stdlib package
// and reads no clock.
//
// A dedicated fixture rather than pointing the known-good cases at a real
// package. cmd/define/play is pure TODAY; using it here would mean a change to
// production code silently changes what this test asserts, and a guard's
// positive case has to be as fixed as its negative one.
package pure

import "sort"

func Sorted(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
