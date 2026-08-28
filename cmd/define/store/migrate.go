package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MigrateFlatDeck moves a pre-language deck from words/ into words/en/.
//
// Without it a deck written before #23 is ORPHANED rather than lost: Deck() reads
// words/<lang>/, so every existing word would simply stop being there.
//
// There is no language parameter, and that is the point rather than an omission.
// The destination is DefaultLang always, justified by a fact about the files: a
// flat deck was written by a tool that only ever consulted the English
// dictionary and requested _en_us_ recordings. Taking the ACTIVE language would
// mean one `define -lang es` on a first run filed an entire English deck under
// words/es/.
//
// So the migration is language-BLIND, and it says so out loud instead of
// guessing: a Spanish word filed before #23 lands in words/en/ too, and only the
// learner can know that. Inferring it here is exactly the heuristic #23's
// declared mode exists to avoid.
//
// Non-destructive on the one artifact here that cannot be regenerated. It MOVES
// rather than rewrites, never overwrites a destination, and leaves a colliding
// flat file exactly where it is — inert, since nothing reads words/*.yaml any
// more — rather than choosing which copy of the learner's word to destroy.
func MigrateFlatDeck(dir string, warn io.Writer) error {
	src := filepath.Join(dir, RuntimeDirs[0])
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var flat []string
	for _, e := range entries {
		// A subdirectory is another language's deck, not something to move.
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		flat = append(flat, e.Name())
	}
	if len(flat) == 0 {
		return nil
	}

	dst := filepath.Join(src, string(DefaultLang))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	moved, skipped := 0, []string(nil)
	for _, name := range flat {
		to := filepath.Join(dst, name)
		if _, err := os.Stat(to); err == nil {
			// Both copies exist. Keep both: the destination is the live one, and
			// the flat file is unreachable but not destroyed.
			skipped = append(skipped, name)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(filepath.Join(src, name), to); err != nil {
			return err
		}
		moved++
	}

	if warn == nil {
		return nil
	}
	if moved > 0 {
		fmt.Fprintf(warn, "define: moved %d word(s) into %s/%s/ — this migration cannot tell "+
			"languages apart, so a non-English word filed before now is in there too; "+
			"mv it into %s/<lang>/ if so\n", moved, RuntimeDirs[0], DefaultLang, RuntimeDirs[0])
	}
	if len(skipped) > 0 {
		fmt.Fprintf(warn, "define: left %d flat word file(s) in place because %s/%s/ already has "+
			"them: %s\n", len(skipped), RuntimeDirs[0], DefaultLang, strings.Join(skipped, ", "))
	}
	return nil
}
