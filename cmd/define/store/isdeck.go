package store

import (
	"os"
	"path/filepath"
)

// IsDeck reports whether define has already made this directory its own.
//
// DERIVED FROM THE TWO LISTS THAT ALREADY ENUMERATE WHAT WE WRITE, never from a
// hardcoded "words". RuntimeDirs and RuntimeFiles exist precisely because the set
// of artifacts grew three times across three issues and each growth found a place
// that had not been told; a predicate that re-listed them would be the fourth.
//
// ANY artifact counts, not all of them. A directory holding only audio/ — one
// lookup that played a recording and captured nothing else — is one define has
// written into, and asking permission to create a deck there would be asking
// about something that exists.
//
// THE DIRECTORY NAME IS NEVER PART OF A PATTERN. `dir` comes from os.Getwd(), so
// it is input this code does not choose, and an earlier draft globbed
// filepath.Join(dir, pattern): a cwd containing `[` makes that malformed, Glob
// returns ErrBadPattern, the branch is skipped, and an ESTABLISHED deck reads as
// not-a-deck — after which a "yes" runs MkdirAll over a live deck. `*` or `?`
// could match a sibling instead. One ReadDir, then Match against entry NAMES, so
// no directory name can change what this means.
//
// UNREADABLE READS AS FALSE, which is the safe direction: the caller's response
// to false is to ASK, and a question is a better answer to a directory we could
// not read than an assumption is.
func IsDeck(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	dirs := make(map[string]bool, len(RuntimeDirs))
	for _, d := range RuntimeDirs {
		dirs[d] = true
	}
	for _, e := range entries {
		if e.IsDir() {
			if dirs[e.Name()] {
				return true
			}
			continue
		}
		for _, pat := range RuntimeFiles {
			// The atomic-write shadow is DEBRIS, not an artifact. A crashed write
			// leaves one behind, and counting it would make a directory that never
			// finished becoming a deck read as an established one — skipping this
			// question in exactly the case where something already went wrong.
			if pat == tmpPattern {
				continue
			}
			if ok, err := filepath.Match(pat, e.Name()); err == nil && ok {
				return true
			}
		}
	}
	return false
}
