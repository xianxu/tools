package store

import (
	"os"
	"path/filepath"
	"testing"
)

// A DECK IS ANY ARTIFACT define WROTE, not just words/.
//
// DERIVED, so the table is the code's own list rather than a second copy of it:
// a seventh RuntimeDirs entry produces a seventh subtest here with no edit to
// isdeck.go. RuntimeDirs and RuntimeFiles grew three times across three issues
// and each growth found a place that had not been told; a predicate that
// re-listed them would be the fourth.
//
// ANY artifact, not all of them. A directory holding only audio/ — one lookup
// that played a recording — is one define has written into, and asking
// permission to create a deck there asks about something that exists.
func TestIsDeckSeesEveryRuntimeDir(t *testing.T) {
	for _, dir := range RuntimeDirs {
		t.Run(dir, func(t *testing.T) {
			root := t.TempDir()
			if IsDeck(root) {
				t.Fatal("an empty directory is not a deck")
			}
			if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
				t.Fatal(err)
			}
			if !IsDeck(root) {
				t.Errorf("a directory holding %s/ is a deck — define wrote it, so define "+
					"has already made this directory its own", dir)
			}
		})
	}
}

// The FILE half, including the per-language family. userModelLegacy is the
// pre-#23 flat name and still counts: a directory carrying one is a deck old
// enough to predate the migration, which is the last directory that should be
// asked whether it wants to become one.
func TestIsDeckSeesRuntimeFiles(t *testing.T) {
	for _, name := range []string{
		langFileName,
		UserModelName(DefaultLang),
		UserModelName(Lang("es")),
		userModelLegacy,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			if !IsDeck(root) {
				t.Errorf("a directory holding %s is a deck", name)
			}
		})
	}
}

// THE ATOMIC-WRITE SHADOW IS DEBRIS, NOT AN ARTIFACT.
//
// A crashed write leaves one behind. Counting it would make a directory that
// never finished becoming a deck read as an established one — skipping the
// question this predicate exists to ask, in exactly the case where something
// already went wrong.
func TestIsDeckIgnoresTheAtomicWriteShadow(t *testing.T) {
	root := t.TempDir()
	f, err := newTempFile(root)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if IsDeck(root) {
		t.Errorf("a leftover %s shadow made an unfinished directory read as a deck; "+
			"the shadow is debris from a crashed write, not an artifact", tmpPattern)
	}
}

func TestIsDeckOnAnEmptyOrMissingDirectory(t *testing.T) {
	if IsDeck(t.TempDir()) {
		t.Error("an empty directory is not a deck")
	}
	// UNREADABLE READS AS NOT-A-DECK, which is the SAFE direction: the caller's
	// response to false is to ASK, and a question is a better answer to a
	// directory we could not read than an assumption.
	if IsDeck(filepath.Join(t.TempDir(), "nope")) {
		t.Error("a directory that does not exist holds no deck")
	}
}

// THE DIRECTORY NAME IS NOT A PATTERN (#50 PQ-5).
//
// The directory comes from os.Getwd(), so it is input this code does not choose.
// An earlier draft globbed filepath.Join(dir, pattern): a cwd containing `[`
// makes that pattern malformed, Glob returns ErrBadPattern, the branch is
// skipped — and an ESTABLISHED deck reads as not-a-deck. The caller then asks to
// create one, and a "yes" runs MkdirAll over a live deck. `*` or `?` in the path
// could match a sibling directory instead.
//
// Matching entry NAMES keeps the path out of the pattern entirely, so no
// directory name can change what this predicate means.
func TestIsDeckWhenTheDirectoryNameLooksLikeAPattern(t *testing.T) {
	// THE ARTIFACT MUST BE A FILE, and that is the whole point of the row.
	//
	// A first draft of this test put RuntimeDirs[0] in the bracket-named
	// directory and passed under the very defect it names — because the directory
	// branch is a map lookup with no pattern in it, so a dir-only case never
	// reaches the globbing at all. The file branch is where a pattern lives, so
	// the file branch is what an adversarial directory name has to be tested
	// against. Verified by reversion: restoring the Glob form reddens these rows.
	for _, name := range []string{"br[acket", "sta*r", "quest?ion", "bra]ce"} {
		for _, artifact := range []string{langFileName, UserModelName(DefaultLang)} {
			t.Run(name+"/"+artifact, func(t *testing.T) {
				root := filepath.Join(t.TempDir(), name)
				if err := os.MkdirAll(root, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, artifact), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				if !IsDeck(root) {
					t.Errorf("a real deck in a directory named %q read as NOT a deck — the "+
						"path reached a glob pattern, and the caller would now offer to "+
						"create a deck on top of this one", name)
				}
			})
		}
	}
}
