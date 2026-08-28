package main

import "errors"

// ErrNoEntry means the dictionary has no entry for the word. It is a normal
// outcome, not a malfunction: NOAD genuinely lacks recent coinages such as
// "rizz", and the CLI reports it as a clean non-zero exit.
var ErrNoEntry = errors.New("no dictionary entry")

// ErrLookupFailed separates a malfunction from a word the dictionary simply does
// not have.
//
// Platform-neutral, beside ErrNoEntry, because the decision that USES the
// distinction (foldLookupError) is platform-neutral too. It lived in the darwin
// file while its only consumer moved out, which would have left the pure
// function unable to compile off darwin.
var ErrLookupFailed = errors.New("dictionary lookup failed")

// Dictionary resolves a word to a raw dictionary entry.
//
// The seam exists so that ParseEntry never touches CoreServices and the parser
// tests never touch the system dictionary (ARCH-PURE). Everything downstream of
// Lookup operates on a plain string.
type Dictionary interface {
	Lookup(word string) (string, error)
}
