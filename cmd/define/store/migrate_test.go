package store_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

// A deck written before languages existed moves under words/en/ rather than
// being orphaned — which is what it would otherwise be, since Deck() reads
// words/<lang>/ and would see nothing.
func TestMigrateFlatDeckMovesWordsUnderTheDefaultLanguage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "words", "sycophantic.yaml"), "text: sycophantic\n")

	var warn bytes.Buffer
	if err := store.MigrateToLanguages(dir, &warn); err != nil {
		t.Fatal(err)
	}

	if got := read(t, filepath.Join(dir, "words", "en", "sycophantic.yaml")); got != "text: sycophantic\n" {
		t.Errorf("migrated file = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "words", "sycophantic.yaml")); !os.IsNotExist(err) {
		t.Error("the flat file survived a clean migration; the deck now exists twice")
	}
	// It says what it did: this is the one operation here that touches the
	// learner's own words, and it is language-BLIND — a Spanish word filed before
	// #23 lands in en, and only the learner can know that.
	if !strings.Contains(warn.String(), "1") || !strings.Contains(warn.String(), "words/en") {
		t.Errorf("migration was silent about what it moved: %q", warn.String())
	}

	// And the deck can actually see it now.
	deck, err := store.NewYAML(dir, store.DefaultLang, nil).Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 || deck[0].Text != "sycophantic" {
		t.Errorf("deck after migration = %+v", deck)
	}
}

// A collision NEVER overwrites and NEVER deletes. The destination wins, the flat
// file stays where it is, and the warning names it.
//
// Leaving it is safe rather than sloppy: nothing reads words/*.yaml after #23, so
// a surviving flat file is inert. That is the whole argument for choosing
// "leave it" over "merge it" on the one artifact here that cannot be regenerated.
func TestMigrateFlatDeckNeverOverwritesOrDeletesOnCollision(t *testing.T) {
	dir := t.TempDir()
	flat := filepath.Join(dir, "words", "mesa.yaml")
	dest := filepath.Join(dir, "words", "en", "mesa.yaml")
	writeFile(t, flat, "text: mesa\nlookups: 1\n")
	writeFile(t, dest, "text: mesa\nlookups: 99\n")

	var warn bytes.Buffer
	if err := store.MigrateToLanguages(dir, &warn); err != nil {
		t.Fatal(err)
	}

	if got := read(t, dest); got != "text: mesa\nlookups: 99\n" {
		t.Errorf("the destination was overwritten: %q", got)
	}
	if got := read(t, flat); got != "text: mesa\nlookups: 1\n" {
		t.Errorf("the flat file was destroyed rather than left alone: %q", got)
	}
	if !strings.Contains(warn.String(), "mesa.yaml") {
		t.Errorf("a skipped collision must be named, not silently left: %q", warn.String())
	}
}

// Run twice, same result. The rule that must stop any future version that
// rewrites in place rather than moving.
func TestMigrateFlatDeckIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "words", "sycophantic.yaml"), "text: sycophantic\n")

	for i := 0; i < 2; i++ {
		if err := store.MigrateToLanguages(dir, nil); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	if got := read(t, filepath.Join(dir, "words", "en", "sycophantic.yaml")); got != "text: sycophantic\n" {
		t.Errorf("after two runs, file = %q", got)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "words"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "en" {
		t.Errorf("words/ = %v, want only the en directory", entries)
	}
}

// Nothing to move: no directory created, nothing said. A first run in an empty
// directory must not announce a migration that did not happen.
func TestMigrateFlatDeckIsSilentWhenThereIsNothingToDo(t *testing.T) {
	dir := t.TempDir()
	var warn bytes.Buffer
	if err := store.MigrateToLanguages(dir, &warn); err != nil {
		t.Fatal(err)
	}
	if warn.String() != "" {
		t.Errorf("said %q about an empty directory", warn.String())
	}

	// And with a deck that is already migrated.
	writeFile(t, filepath.Join(dir, "words", "en", "sycophantic.yaml"), "text: sycophantic\n")
	warn.Reset()
	if err := store.MigrateToLanguages(dir, &warn); err != nil {
		t.Fatal(err)
	}
	if warn.String() != "" {
		t.Errorf("said %q about an already-migrated deck", warn.String())
	}
}

// Subdirectories are skipped, not descended into: words/es/ is another
// language's deck, not something to move into en.
func TestMigrateFlatDeckLeavesOtherLanguagesAlone(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "words", "es", "madrugar.yaml"), "text: madrugar\n")

	if err := store.MigrateToLanguages(dir, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(dir, "words", "es", "madrugar.yaml")); got != "text: madrugar\n" {
		t.Errorf("another language's deck was touched: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "words", "en", "madrugar.yaml")); !os.IsNotExist(err) {
		t.Error("a Spanish word was moved into the English deck")
	}
}

// The learner model moves with the deck, because it is DERIVED from the deck.
//
// M1 made the deck per-language and left this file flat, so `--reflect` in
// Spanish overwrote the English model and every English answer was then pitched
// at a Spanish beginner. Same rules as the deck's move: never overwrite, never
// delete, say what happened.
func TestMigrateMovesTheUserModelUnderItsLanguage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user-model.md"), "## Level\nB2\n")

	var warn bytes.Buffer
	if err := store.MigrateToLanguages(dir, &warn); err != nil {
		t.Fatal(err)
	}
	// Read back THROUGH THE STORE, not from a hand-built path: what matters is
	// that the migration's destination is exactly what UserModel() later opens.
	// Asserting a literal name would stay green while the migration wrote a file
	// nothing reads.
	got, err := store.NewYAML(dir, store.DefaultLang, nil).UserModel()
	if err != nil {
		t.Fatal(err)
	}
	if got != "## Level\nB2\n" {
		t.Errorf("the migrated model is not what the store reads: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "user-model.md")); !os.IsNotExist(err) {
		t.Error("the flat model survived; the learner model now exists twice")
	}
	if !strings.Contains(warn.String(), "learner model") {
		t.Errorf("migration was silent about the model: %q", warn.String())
	}
}

// A collision leaves both, like the deck's. The model carries a human-owned
// ## Corrections section that #17 promises never to rewrite, so destroying
// either copy is worse than leaving one unreachable.
func TestMigrateNeverOverwritesAnExistingUserModel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user-model.md"), "old\n")
	if err := store.NewYAML(dir, store.DefaultLang, nil).SetUserModel("current\n"); err != nil {
		t.Fatal(err)
	}

	var warn bytes.Buffer
	if err := store.MigrateToLanguages(dir, &warn); err != nil {
		t.Fatal(err)
	}
	got, err := store.NewYAML(dir, store.DefaultLang, nil).UserModel()
	if err != nil {
		t.Fatal(err)
	}
	if got != "current\n" {
		t.Errorf("the destination was overwritten: %q", got)
	}
	if got := read(t, filepath.Join(dir, "user-model.md")); got != "old\n" {
		t.Errorf("the flat model was destroyed: %q", got)
	}
}

// The bug itself, at the store level: two languages, two models, no clobbering.
func TestTheUserModelIsPerLanguage(t *testing.T) {
	dir := t.TempDir()
	if err := store.NewYAML(dir, store.DefaultLang, nil).SetUserModel("english model\n"); err != nil {
		t.Fatal(err)
	}
	if err := store.NewYAML(dir, "es", nil).SetUserModel("modelo espanol\n"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		lang store.Lang
		want string
	}{{store.DefaultLang, "english model\n"}, {"es", "modelo espanol\n"}} {
		got, err := store.NewYAML(dir, tc.lang, nil).UserModel()
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%s model = %q, want %q — a reflect in one language replaced the other's",
				tc.lang, got, tc.want)
		}
	}
}
