package main

import "errors"

// ErrNoEntry means the dictionary has no entry for the word. It is a normal
// outcome, not a malfunction: NOAD genuinely lacks recent coinages such as
// "rizz", and the CLI reports it as a clean non-zero exit.
var ErrNoEntry = errors.New("no dictionary entry")

// Dictionary resolves a word to a raw dictionary entry.
//
// The seam exists so that ParseEntry never touches CoreServices and the parser
// tests never touch the system dictionary (ARCH-PURE). Everything downstream of
// Lookup operates on a plain string.
type Dictionary interface {
	Lookup(word string) (string, error)
}
