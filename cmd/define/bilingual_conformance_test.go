//go:build darwin && conformance

package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/conformance"
)

// Run strictly after macOS upgrades and before dictionary-related releases:
// CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestBilingualNative'
func bilingualNativeProbe(t *testing.T) nativeSpanishEnglishSource {
	t.Helper()
	source := newSpanishEnglishSource().(nativeSpanishEnglishSource)
	records, err := source.Records("red")
	if err != nil {
		conformance.SkipOrFail(t, "installed Oxford Spanish dictionary inaccessible", err)
		return source
	}
	if len(records) == 0 {
		conformance.SkipOrFail(t, "installed Oxford Spanish dictionary returned no probe records", ErrNoEntry)
	}
	return source
}

func TestBilingualNativeDirection(t *testing.T) {
	source := bilingualNativeProbe(t)
	for _, tc := range []struct{ query, canonical, needle string }{
		{"red", "red", "net"}, {"mesa", "mesa", "table"}, {"madrugar", "madrugar", "get up early"},
		{"arbol", "árbol", "tree"}, {"jalapeno", "jalapeño", "pepper"}, {"madrugaste", "madrugar", "get up early"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			records, err := source.Records(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			texts, err := selectSpanishRecords(records, tc.query, tc.canonical)
			if err != nil || !strings.Contains(strings.Join(texts, "\n"), tc.needle) {
				t.Fatalf("selection %q: %v", texts, err)
			}
			if tc.query == "red" {
				if strings.Contains(strings.Join(texts, "\n"), "redder") {
					t.Fatal("English red leaked")
				}
				sawEnglish := false
				for _, record := range records {
					id, e := bilingualRecordIdentity(record.HTML)
					if e == nil && !id.spanish && id.title == "red" {
						sawEnglish = true
					}
				}
				if !sawEnglish {
					t.Fatal("probe no longer exercises both directions")
				}
			}
		})
	}
	// DictionaryServices is synchronous; measure its actual warm queue cost.
	started := time.Now()
	for i := 0; i < 20; i++ {
		records, err := source.Records("mesa")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := selectSpanishRecords(records, "mesa", "mesa"); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(started)
	t.Logf("20 warm native lookups and selections: %s", elapsed)
	if elapsed >= 2*time.Second {
		t.Fatalf("20-word queue exceeded 2s budget: %s", elapsed)
	}
}

func TestBilingualNativeLimits(t *testing.T) {
	source := bilingualNativeProbe(t)
	capSource := source
	capSource.recordLimit = 2
	if _, err := capSource.Records("red"); !errors.Is(err, ErrBilingualLimit) {
		t.Fatalf("record cap: %v", err)
	}
	byteSource := source
	byteSource.byteLimit = 10
	if _, err := byteSource.Records("mesa"); !errors.Is(err, ErrBilingualLimit) {
		t.Fatalf("byte cap: %v", err)
	}
	if _, err := source.Records(strings.Repeat("x", bilingualMaxBytes+1)); !errors.Is(err, ErrBilingualLimit) {
		t.Fatalf("query cap: %v", err)
	}
	absent := source
	absent.dictionaryID = "com.example.nonexistent-tools61-dictionary"
	if _, err := absent.Records("mesa"); !errors.Is(err, ErrBilingualUnavailable) {
		t.Fatalf("absent dictionary: %v", err)
	}
	records, err := source.Records("zzzxxyy")
	if err != nil || len(records) != 0 {
		t.Fatalf("ordinary miss: %d %v", len(records), err)
	}
	if _, err := source.Records("red\x00mesa"); !errors.Is(err, ErrBilingualMalformed) {
		t.Fatalf("embedded null: %v", err)
	}
}
