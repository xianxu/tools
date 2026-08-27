// Package nothing exists to be ACCEPTED with ZERO imports.
//
// ImportsOnly treats no-imports as a PASS, not as a failed measurement — see its
// doc comment for why. That decision was pinned only by cmd/define/play, which
// imports nothing TODAY. The pure fixture next door was created for exactly this
// reason ("a change to production code silently changes what this test asserts")
// and the reasoning was not carried to the second case; #7 adds a form package
// that will likely give play its first import, at which point the zero-import
// path would have gone quiet with nothing to say so (BR-41).
package nothing

func Nothing() int { return 0 }
