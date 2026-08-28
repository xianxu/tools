package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// An interrupted write leaves a temp file. It must never be read as a word.
//
// INTERNAL, and that is the fix rather than a preference. The external version
// planted its fixture at words/.tmp-halfwritten — the pre-#23 flat path — while
// wordsDir() had become words/<lang>/. Deck() never listed it, so the test went
// on passing while exercising nothing. Both the directory and the temp name are
// now obtained from the functions that produce them, so a layout or prefix
// change reddens this instead of silently relocating the fixture out of the code
// under test.
//
// The body is DELIBERATELY VALID for a word. The old fixture was unparseable, so
// removing Deck()'s suffix check left it warned-and-skipped and the suite green
// either way — the guard the comment described was not actually pinned. With a
// readable body the suffix check is the only thing standing between this file
// and the deck, which is what makes the assertion below load-bearing.
func TestDeckIgnoresInterruptedWrites(t *testing.T) {
	dir := t.TempDir()
	y := NewYAML(dir, DefaultLang, nil)
	if err := y.Upsert(Word{Text: "good", LastSeen: time.Now()}); err != nil {
		t.Fatal(err)
	}

	f, err := newTempFile(y.wordsDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("text: bad\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if filepath.Dir(f.Name()) != y.wordsDir() {
		t.Fatalf("the fixture landed at %s, not in the deck directory %s", f.Name(), y.wordsDir())
	}

	deck, err := y.Deck()
	if err != nil {
		t.Fatalf("a leftover temp file broke the deck: %v", err)
	}
	if len(deck) != 1 || deck[0].Text != "good" {
		t.Errorf("deck = %+v, want only the completed write — a half-written file was read as a word", deck)
	}
	// The fixture must still be there: Deck() skips it, it does not clean up.
	if _, err := os.Stat(f.Name()); err != nil {
		t.Errorf("Deck() removed the temp file: %v", err)
	}
}
