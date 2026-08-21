// Package store persists the vocabulary deck as YAML files in a directory.
//
// The directory is a parameter, not a policy: `define` writes where it was
// started, and who decides that stays a one-line question at the boundary.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode"
)

// Word is one entry in the deck.
//
// It holds what is true about the WORD, not about studying it. Box numbers and
// intervals belong to the scheduler (#5); putting them here would make this file
// the schedule's home and force that issue to migrate it.
type Word struct {
	Text      string    `yaml:"text"`
	FirstSeen time.Time `yaml:"first_seen"`
	LastSeen  time.Time `yaml:"last_seen"`
	Lookups   int       `yaml:"lookups"`
}

// Key is the identity of a word: case- and space-normalised text. `Define` and
// `define` are one entry.
func Key(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

// Slug returns the filename for a key: exactly one safe path element, always.
//
// The rule, decided in the plan rather than left to the implementation because
// it fixes filenames on disk:
//
//  1. spaces become "-", so `hot dog` → `hot-dog.yaml` — readable;
//  2. if replacing "-" with " " does not return the key, a short hash is
//     appended, so the hyphenated word `hot-dog` cannot share a file with the
//     two-word `hot dog`;
//  3. anything containing a path separator, or resolving to ""/"."/"..", takes
//     the hash form too.
//
// Every file also stores `text:`, so the key is recoverable from content and the
// filename is only an index — which is what keeps a future naming change a
// rename rather than a data migration.
func Slug(key string) string {
	base := strings.ReplaceAll(key, " ", "-")
	if reversible(key, base) && safeElement(base) {
		return base
	}
	sum := sha256.Sum256([]byte(key))
	short := hex.EncodeToString(sum[:])[:6]
	if !safeElement(base) {
		base = sanitize(base)
	}
	if base == "" {
		return "w-" + short
	}
	return base + "-" + short
}

func reversible(key, base string) bool {
	return strings.ReplaceAll(base, "-", " ") == key
}

// safeElement rejects anything that is not a single, ordinary path component.
func safeElement(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	if strings.ContainsAny(s, `/\`) || strings.HasPrefix(s, ".") {
		return false
	}
	for _, r := range s {
		if r == 0 || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == 0 || unicode.IsControl(r):
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), ".-")
}
