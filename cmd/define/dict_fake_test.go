package main

import (
	"errors"
	"strings"
	"testing"
)

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
