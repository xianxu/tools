//go:build darwin && conformance

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/conformance"
)

// Native production lookup, structural render, and final terminal serialization.
// Optional DEFINE_LAYOUT_CAPTURE_PREFIX records the actual outputs for review.
func TestBilingualNativeRendirLayout(t *testing.T) {
	source := bilingualNativeProbe(t)
	records, err := source.Records("rendir")
	if err != nil {
		conformance.SkipOrFail(t, "native rendir record unavailable", err)
		return
	}
	selected, err := selectedSpanishRecords(records, "rendir", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 {
		t.Fatalf("rendir record count changed: %d", len(selected))
	}
	doc, err := parseBilingualDocument(selected[0])
	if err != nil {
		t.Fatal(err)
	}
	structural, _ := renderBilingualDocument(doc, RenderOpts{})
	assertOxfordRows(t, structural)
	if oxfordContent(structural) != oxfordContent(selected[0].Text) {
		t.Fatal("native content lost")
	}
	ids, name := spanishDictionarySources(installedDictionaries())
	if len(ids) == 0 || strings.Contains(name, "unavailable") || strings.Contains(name, "unknown") {
		conformance.SkipOrFail(t, "verified Spanish primary dictionary unavailable", ErrBilingualUnavailable)
		return
	}
	dict, _ := systemDictionary("es", nil)
	primary, err := dict.Lookup("rendir")
	if err != nil {
		conformance.SkipOrFail(t, "Spanish primary rendir unavailable", err)
		return
	}
	set := definitionsFor(dict, "rendir", primary, nil, true)
	if set.err != nil || len(set.sections) != 2 || set.sections[1].err != nil || set.sections[1].formatErr != nil {
		t.Fatalf("native composition failed: %+v", set)
	}
	for _, profile := range []struct {
		name string
		sc   store.Scheme
	}{{"dark", store.SchemeDark}, {"light", store.SchemeLight}} {
		for _, lang := range []store.Lang{"es", "en"} {
			for _, width := range []int{32, 80} {
				// Shared independent cell oracle checks every cell, including indentation,
				// headings, blank rows, wrapped example translations, and right-side fill.
				assertDefinitionSectionCells(t, set, lang, profile.sc, width)
				output := renderDefinitionOutput(set, RenderOpts{Word: "rendir", Color: true, Width: width, Tint: tintPolicy{lang: lang, on: true, scheme: holderFor(profile.sc)}})
				if prefix := os.Getenv("DEFINE_LAYOUT_CAPTURE_PREFIX"); prefix != "" {
					base := fmt.Sprintf("%s-%s-%s-%d", prefix, profile.name, lang, width)
					for suffix, content := range map[string]string{".ansi": serializeOutput(output, width, profile.sc), ".txt": stripEscapes(output.text)} {
						if err := os.WriteFile(base+suffix, []byte(content), 0600); err != nil {
							t.Fatal(err)
						}
					}
					t.Logf("production render capture: %s.{ansi,txt}", base)
				}
			}
		}
	}
}
