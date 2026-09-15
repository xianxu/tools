package main

import (
	"errors"
	"io"
	"sync"

	"github.com/xianxu/tools/cmd/define/store"
)

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

// dictionaryMu serializes every dictionary lookup in the process (#54). The
// system dictionary is reached through cgo (dict_darwin.go) with no lock of its
// own and no documented thread-safety, and the background job looks words up
// while the editor loop does. ONE package-level mutex rather than one per
// instance, so the loop's dictionary and a job's copy after /lang still take
// turns: they are different values over the same DictionaryServices.
var dictionaryMu sync.Mutex

// lockedDictionary is a Dictionary whose every lookup holds dictionaryMu.
type lockedDictionary struct{ inner Dictionary }

func (l lockedDictionary) Lookup(word string) (string, error) {
	dictionaryMu.Lock()
	defer dictionaryMu.Unlock()
	return l.inner.Lookup(word)
}

// lockedDictionaries wraps a dictionary builder so everything it builds is
// locked. realDeps wraps its one builder with it, and that builder is the seam
// both the startup dictionary and every /lang switch already go through.
func lockedDictionaries(build func(store.Lang, io.Writer) (Dictionary, string)) func(store.Lang, io.Writer) (Dictionary, string) {
	return func(l store.Lang, w io.Writer) (Dictionary, string) {
		d, name := build(l, w)
		locked := lockedDictionary{inner: d}
		if provider, ok := d.(supplementalDictionary); ok {
			return lockedSupplementalDictionary{lockedDictionary: locked, provider: provider}, name
		}
		return locked, name
	}
}

// lockedSupplementalDictionary preserves the optional capability only for
// sources that support it. Supplemental native reads share the primary lock.
type lockedSupplementalDictionary struct {
	lockedDictionary
	provider supplementalDictionary
}

func (d lockedSupplementalDictionary) primaryLabel() string { return d.provider.primaryLabel() }
func (d lockedSupplementalDictionary) supplement(word, primary string) definitionSection {
	dictionaryMu.Lock()
	defer dictionaryMu.Unlock()
	return d.provider.supplement(word, primary)
}

// dictionarySourceLanguage reports producer-owned provenance. An absent
// capability is unknown; neither the study language nor a display label proves
// which language an all-active-dictionaries fallback returned.
func dictionarySourceLanguage(dict Dictionary) store.Lang {
	if source, ok := dict.(interface{ primarySourceLanguage() store.Lang }); ok {
		return source.primarySourceLanguage()
	}
	return ""
}

// monolingualDictionary carries the language verified at selected-ID assembly.
// It wraps only a primary source, before optional supplement adapters are added.
type monolingualDictionary struct {
	Dictionary
	language store.Lang
}

func (d monolingualDictionary) primarySourceLanguage() store.Lang { return d.language }
func (d lockedDictionary) primarySourceLanguage() store.Lang {
	return dictionarySourceLanguage(d.inner)
}
