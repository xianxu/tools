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
		d.entries[strings.TrimSuffix(filepath.Base(p), ".txt")] = string(b)
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
