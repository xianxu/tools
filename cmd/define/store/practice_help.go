package store

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// PracticeHelpVersion is the cache policy version; a file with another version
// reads as empty.
//
// A MISS rather than a migration, because this is a cache: every entry can be
// fetched again on demand, so an unrecognised file costs one round of requests,
// while half-understanding another version's file could serve wrong answers.
const PracticeHelpVersion = 1

const practiceHelpFileName = "practice-help.json"

// A cache that grows on every practice turn needs both bounds. Both are enforced
// on WRITE so the file never outgrows them; the byte cap is enforced on READ too,
// because the file is untrusted and a corrupted or hand-edited one must not
// allocate an arbitrary buffer at startup.
const (
	maxPracticeHelpEntries = 512
	maxPracticeHelpBytes   = 1 << 20
)

// PracticeHelpFileName is the per-deck practice-help cache's filename.
func PracticeHelpFileName() string { return practiceHelpFileName }

// HelpEntry is one cached English translation of one exact displayed source text.
//
// The store only persists entries; matching one to what is on screen is the
// caller's job, which is why Mode is opaque here.
type HelpEntry struct {
	Lang    Lang      `json:"lang"`    // source language of Source (e.g. "es")
	Mode    string    `json:"mode"`    // "gloss" or "cloze" (opaque to the store; non-empty)
	Source  string    `json:"source"`  // exact source text
	English string    `json:"english"` // the translation
	At      time.Time `json:"at"`      // when it was stored; eviction drops the oldest
}

// practiceHelpFile is the on-disk shape: compact JSON, entries oldest first, so
// eviction is always a drop from the front.
//
//	{"version":1,"entries":[{"lang":"es","mode":"gloss","source":"…","english":"…","at":"2026-09-01T12:00:00Z"},…]}
//
// Generic over the entry so ONE declaration serves both directions: the writer
// carries entries it has already encoded (and sized), the reader decodes them.
type practiceHelpFile[E any] struct {
	Version int `json:"version"`
	Entries []E `json:"entries"`
}

// ReadPracticeHelp reads the deck's cache as UNTRUSTED input — it lives in the
// learner's directory, where anything may have edited it. A missing, unreadable,
// oversize, malformed, or wrong-version file reads as nil: a cache miss, which
// the caller answers by asking again. An entry a caller could not use is dropped
// rather than failing the entries around it.
func ReadPracticeHelp(dir string) []HelpEntry {
	f, err := os.Open(filepath.Join(dir, practiceHelpFileName))
	if err != nil {
		return nil
	}
	defer f.Close()
	// Limit the read before parsing, as ReadBilingual does; the one byte past the
	// cap is what tells an oversize file from one exactly at it.
	b, err := io.ReadAll(io.LimitReader(f, maxPracticeHelpBytes+1))
	if err != nil {
		return nil
	}
	return parsePracticeHelp(b)
}

// parsePracticeHelp is ReadPracticeHelp without the file, split out as
// parseBilingualSetting is: the fuzzer then spends its budget on the decoder.
// With a disk write per input, minimizing each new input took long enough to
// freeze exploration for most of a 20-second run.
func parsePracticeHelp(b []byte) []HelpEntry {
	if len(b) > maxPracticeHelpBytes {
		return nil
	}
	var file practiceHelpFile[HelpEntry]
	if json.Unmarshal(b, &file) != nil || file.Version != PracticeHelpVersion {
		return nil
	}
	var out []HelpEntry
	for _, e := range file.Entries {
		if e, ok := usableHelpEntry(e); ok {
			out = append(out, e)
		}
	}
	return out
}

// usableHelpEntry is the one predicate both directions share. Read drops what it
// rejects because the file is untrusted; write drops it too, so the entry and
// byte budgets are never spent on an entry the next read would discard.
//
// Lang goes through ParseLang, the package's only way to make a Lang from input:
// the type is a path segment elsewhere, and a decoded "../x" would be one.
func usableHelpEntry(e HelpEntry) (HelpEntry, bool) {
	l, err := ParseLang(string(e.Lang))
	if err != nil || e.Mode == "" || e.Source == "" || e.English == "" {
		return HelpEntry{}, false
	}
	e.Lang = l
	return e, true
}

// WritePracticeHelp atomically replaces the cache with entries, keeping the
// newest maxPracticeHelpEntries by At (a tie goes to the later slice position)
// and dropping the oldest further until the file fits maxPracticeHelpBytes.
//
// The byte budget is counted over each entry's own encoding, newest first, and
// the file is assembled from those same encodings — which is what makes the
// count exact rather than an estimate, and means an entry older than the cut is
// never encoded at all. A failed encode or write returns the error and leaves
// any previous file intact.
func WritePracticeHelp(dir string, entries []HelpEntry) error {
	// A filtered COPY: sorting the caller's slice in place would reorder state
	// it still holds.
	live := make([]HelpEntry, 0, len(entries))
	for _, e := range entries {
		if e, ok := usableHelpEntry(e); ok {
			live = append(live, e)
		}
	}
	// Stable and ascending, so equal times keep slice order and walking back
	// from the end meets the later position first.
	slices.SortStableFunc(live, func(a, b HelpEntry) int { return a.At.Compare(b.At) })

	// Non-nil, so an empty cache encodes as [] rather than null.
	kept := make([]json.RawMessage, 0, min(len(live), maxPracticeHelpEntries))
	frame, err := json.Marshal(practiceHelpFile[json.RawMessage]{Version: PracticeHelpVersion, Entries: kept})
	if err != nil {
		return err
	}
	size := len(frame)
	for i := len(live) - 1; i >= 0 && len(kept) < maxPracticeHelpEntries; i-- {
		raw, err := json.Marshal(live[i])
		if err != nil {
			return err
		}
		grow := len(raw)
		if len(kept) > 0 {
			grow++ // the comma between entries
		}
		if size+grow > maxPracticeHelpBytes {
			break
		}
		size += grow
		kept = append(kept, raw)
	}
	slices.Reverse(kept) // chosen newest first, stored oldest first
	b, err := json.Marshal(practiceHelpFile[json.RawMessage]{Version: PracticeHelpVersion, Entries: kept})
	if err != nil {
		return err
	}
	return writeBytesAtomic(filepath.Join(dir, practiceHelpFileName), b)
}
