package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fake lives in a _test.go file so it never links into the shipped binary.
// fakeDictionary serves the committed corpus in testdata/entries, captured from
// the real API by testdata/capture.sh. Fixtures are real captured output rather
// than hand-written approximations, so the parser is developed against text the
// system actually produces (ARCH-MOCK).
type fakeDictionary struct {
	entries map[string]string
}

// loadFakeDictionary reads every fixture under dir. It fails on an empty corpus:
// a silently-empty fake would make the no-data-loss invariant vacuously green,
// which is the exact failure testdata/capture.sh's byte floor guards against.
func loadFakeDictionary(dir string) (*fakeDictionary, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no fixtures in %s — run testdata/capture.sh (unsandboxed)", dir)
	}
	d := &fakeDictionary{entries: make(map[string]string, len(paths))}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, fmt.Errorf("empty fixture %s — re-run testdata/capture.sh", p)
		}
		// Lowercase at LOAD, matching what Lookup does to the query. The real
		// dependency is case-insensitive (verified: capture.py Amazon and
		// capture.py amazon both return the entry), so a fake keyed by the raw
		// filename stem diverges from it — iPhone, iPad, MacBook and Amazon were
		// all present on disk yet unreachable through the seam.
		d.entries[strings.ToLower(strings.TrimSuffix(filepath.Base(p), ".txt"))] = string(b)
	}
	return d, nil
}

func (d *fakeDictionary) Lookup(word string) (string, error) {
	if s, ok := d.entries[strings.ToLower(word)]; ok {
		return s, nil
	}
	return "", ErrNoEntry
}

func testDict(t *testing.T) *fakeDictionary {
	t.Helper()
	d, err := loadFakeDictionary("testdata/entries")
	if err != nil {
		t.Fatalf("loadFakeDictionary: %v", err)
	}
	return d
}

func TestFakeDictionaryLoadsRealFixtures(t *testing.T) {
	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("corpus is empty — later invariant tests would be vacuous")
	}
	got, err := d.Lookup("sycophantic")
	if err != nil {
		t.Fatalf("Lookup(sycophantic): %v", err)
	}
	if !strings.Contains(got, "ˌsikəˈfan(t)ik") {
		t.Errorf("fixture missing the NOAD IPA, got %q", got)
	}
}

func TestFakeDictionaryMissIsErrNoEntry(t *testing.T) {
	if _, err := testDict(t).Lookup("rizz"); !errors.Is(err, ErrNoEntry) {
		t.Errorf("want ErrNoEntry, got %v", err)
	}
}

func TestLoadFakeDictionaryRejectsEmptyCorpus(t *testing.T) {
	if _, err := loadFakeDictionary(t.TempDir()); err == nil {
		t.Error("want an error for an empty corpus, got nil")
	}
}

// Every fixture on disk must be reachable through the seam, not merely present
// in the map. Four were not, which silently disabled the end-to-end coverage of
// the no-pronunciation path they had been added for.
func TestEveryFixtureIsReachableViaLookup(t *testing.T) {
	d := testDict(t)
	for key := range d.entries {
		if _, err := d.Lookup(key); err != nil {
			t.Errorf("fixture %q unreachable via Lookup: %v", key, err)
		}
	}
	// And the capitalized spellings a user would actually type.
	for _, w := range []string{"iPhone", "MacBook", "Amazon", "iPad"} {
		if _, err := d.Lookup(w); err != nil {
			t.Errorf("Lookup(%q) failed: %v", w, err)
		}
	}
}
