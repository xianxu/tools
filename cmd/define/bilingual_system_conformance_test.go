//go:build darwin && conformance

package main

import (
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
)

// Exercise the assembled factory, not only the native record adapter: primary
// Lookup stays Spanish, and only enabled display asks for English supplementation.
func TestBilingualNativeSystemDictionary(t *testing.T) {
	installed := installedDictionaries()
	ids, name := spanishDictionarySources(installed)
	if len(ids) == 0 || strings.Contains(name, "unavailable") || strings.Contains(name, "unknown") {
		conformance.SkipOrFail(t, "both Spanish dictionaries required for assembled lookup", ErrBilingualUnavailable)
		return
	}
	dictionary, reported := systemDictionary("es", nil)
	if reported != name {
		t.Fatalf("factory name %q differs from installed selection %q", reported, name)
	}
	primary, err := dictionary.Lookup("red")
	if err != nil {
		conformance.SkipOrFail(t, "Spanish primary dictionary inaccessible", err)
		return
	}
	direct, err := (selectedDictionary{ids: ids}).Lookup("red")
	if err != nil {
		t.Fatal(err)
	}
	if primary != direct {
		t.Fatal("raw Lookup changed primary dictionary content")
	}
	off := definitionsFor(dictionary, "red", primary, nil, false)
	if off.err != nil || off.labeled || len(off.sections) != 1 || len(off.sections[0].entries) != 1 || off.sections[0].entries[0] != primary {
		t.Fatalf("off changed primary definition: %#v", off)
	}
	on := definitionsFor(dictionary, "red", primary, nil, true)
	if on.err != nil || !on.labeled || len(on.sections) != 2 {
		t.Fatalf("on composition: %#v", on)
	}
	if on.sections[0].entries[0] != primary || !strings.HasPrefix(on.sections[0].label, "Spanish") {
		t.Fatal("Spanish not first")
	}
	if on.sections[1].err != nil {
		t.Fatal(on.sections[1].err)
	}
	english := strings.Join(on.sections[1].entries, "\n")
	if !strings.Contains(english, "net") || strings.Contains(english, "redder") {
		t.Fatalf("wrong English direction %q", english)
	}
}
