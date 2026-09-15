package main

import (
	"errors"
	"strings"
	"testing"
)

type definitionFake struct {
	primary                       string
	primaryErr                    error
	entries                       []string
	englishErr                    error
	primaryCalls, supplementCalls int
}

func (d *definitionFake) Lookup(string) (string, error) {
	d.primaryCalls++
	return d.primary, d.primaryErr
}
func (d *definitionFake) supplement(string, string) definitionSection {
	d.supplementCalls++
	return definitionSection{label: "English — Oxford Spanish–English", entries: d.entries, err: d.englishErr}
}
func (d *definitionFake) primaryLabel() string { return "Spanish — Larousse" }

func TestDefinitionAvailability(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		primaryErr, englishErr error
		on                     bool
		successes              int
	}{
		{"both", nil, nil, true, 2}, {"primary only", nil, ErrNoEntry, true, 1},
		{"English only", ErrNoEntry, nil, true, 1}, {"neither", ErrNoEntry, ErrNoEntry, true, 0},
		{"supplement broken", nil, errors.New("broken"), true, 1}, {"off", nil, nil, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &definitionFake{entries: []string{"mesa noun table"}, englishErr: tc.englishErr}
			set := definitionsFor(d, "mesa", "mesa nombre femenino mueble", tc.primaryErr, tc.on)
			success := 0
			for _, s := range set.sections {
				if s.err == nil && len(s.entries) > 0 {
					success++
				}
			}
			if success != tc.successes || (set.err != nil) != (success == 0) {
				t.Fatalf("set=%+v", set)
			}
			if d.supplementCalls != boolInt(tc.on) {
				t.Fatal("supplement gating")
			}
		})
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func TestDefinitionRendering(t *testing.T) {
	primary := "mesa nombre femenino mueble"
	english := "mesa feminine noun table | a red table"
	d := &definitionFake{entries: []string{english}}
	v := &memVocabulary{}
	v.Add("red")
	v.Add("mueble")
	for _, width := range []int{0, 20, 40, 80} {
		text, rs := renderDefinitions(definitionsFor(d, "mesa", primary, nil, true), RenderOpts{Word: "mesa", Width: width, Vocab: v, Color: true})
		plain := unstyled(text)
		es := strings.Index(plain, "Spanish")
		en := strings.Index(plain, "English")
		if es < 0 || en <= es || !strings.Contains(plain, "table") {
			t.Fatalf("%q", plain)
		}
		lines := strings.Split(plain, "\n")
		for _, r := range rs {
			if r.Line < 0 || r.Line >= len(lines) {
				t.Fatalf("bad region %+v", r)
			}
			// These fixtures use single-cell runes, so use an independent
			// coordinate oracle rather than the production cell slicer.
			cells := []rune(lines[r.Line])
			if r.Col < 0 || r.Col+r.Width > len(cells) || string(cells[r.Col:r.Col+r.Width]) != r.Text {
				t.Fatalf("region %+v on %q", r, lines[r.Line])
			}
			if r.Text == "red" {
				t.Fatal("English prose got Spanish word region")
			}
		}
		mono, mrs := renderDefinitions(definitionsFor(d, "mesa", primary, nil, false), RenderOpts{Word: "mesa", Width: width})
		want, wrs := Render(ParseEntry(primary), RenderOpts{Word: "mesa", Width: width})
		if mono != want || len(mrs) != len(wrs) {
			t.Fatalf("off changed output %q", mono)
		}
	}
}

func TestBilingualLookupPaths(t *testing.T) {
	for _, tc := range []struct {
		name     string
		off, raw bool
		english  bool
	}{
		{"on", false, false, true}, {"off", true, false, false}, {"raw", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &definitionFake{primary: "mesa nombre femenino mueble", entries: []string{"mesa feminine noun table"}}
			on := !tc.off
			d := deps{dict: fake, bilingual: &on, langDeps: langDeps{capture: noopCapturer{}}}
			var out, errout strings.Builder
			result := lookupAndRender(d, options{raw: tc.raw}, replCommand{word: "mesa", literal: true}, &out, &errout)
			if result.code != 0 || strings.Contains(out.String(), "English") != tc.english {
				t.Fatalf("out=%q err=%q", out.String(), errout.String())
			}
			if fake.primaryCalls != 1 || fake.supplementCalls != boolInt(tc.english) {
				t.Fatalf("calls primary=%d supplemental=%d", fake.primaryCalls, fake.supplementCalls)
			}
		})
	}
}

func TestDefinitionProtectedSpan(t *testing.T) {
	v := &memVocabulary{}
	v.Add("red")
	for _, tc := range []struct {
		text, protected string
		want            int
	}{
		{"red\nred\nred", "red\nred\n", 1}, {"red red red", "red red", 1}, {"x red\nred end", "red\nred", 0},
	} {
		got := wordRegionsOutside(tc.text, tc.protected, v)
		if len(got) != tc.want {
			t.Fatalf("%q / %q got %v want %d", tc.text, tc.protected, got, tc.want)
		}
	}
}

func TestBilingualAvailabilityThroughLookup(t *testing.T) {
	for _, tc := range []struct {
		name       string
		primaryErr error
		installed  bool
		records    []bilingualRecord
		failure    error
		wantCode   int
		needle     string
	}{
		{"both", nil, true, bilingualFixture(t, "mesa"), nil, 0, "table"},
		{"Spanish missing", ErrNoEntry, true, bilingualFixture(t, "mesa"), nil, 0, "table"},
		{"English missing", nil, false, nil, nil, 0, "enable Spanish–English"},
		{"both missing", ErrNoEntry, false, nil, nil, 1, "English dictionary unavailable"},
		{"English word missing", nil, true, nil, nil, 0, "no English explanation"},
		{"English malformed", nil, true, []bilingualRecord{{HTML: "broken", Text: "bad"}}, nil, 0, "English explanation unavailable"},
		{"English native failure", nil, true, nil, ErrLookupFailed, 0, "English explanation unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			primary := &definitionFake{primary: "mesa nombre femenino mueble", primaryErr: tc.primaryErr}
			source := &fakeRecordSource{installed: tc.installed, entries: map[string][]bilingualRecord{"mesa": tc.records}, failures: map[string]error{"mesa": tc.failure}}
			capture := &countingCapturer{}
			d := deps{dict: spanishDefinitions{primary, source}, langDeps: langDeps{capture: capture}}
			var out, errout strings.Builder
			result := lookupAndRender(d, options{}, replCommand{word: "mesa", literal: true}, &out, &errout)
			if result.code != tc.wantCode || !strings.Contains(out.String()+errout.String(), tc.needle) {
				t.Fatalf("code=%d out=%q err=%q", result.code, out.String(), errout.String())
			}
			if primary.primaryCalls != 1 || len(source.calls) != 1 || len(capture.calls) != 1 {
				t.Fatalf("primary=%d supplemental=%v capture=%v", primary.primaryCalls, source.calls, capture.calls)
			}
			if tc.primaryErr == nil && !strings.Contains(out.String(), "mueble") {
				t.Fatal("valid Spanish explanation suppressed")
			}
		})
	}
}

func FuzzDefinitionRendering(f *testing.F) {
	for _, s := range []string{"madrugar verbo intransitivo levantarse temprano", "mesa feminine noun table", "árbol noun tree", "字 noun árbol"} {
		f.Add(s, uint8(40))
	}
	f.Fuzz(func(t *testing.T, raw string, w uint8) {
		if len(raw) > 4096 || strings.ContainsAny(raw, "\x1b\r\x00") {
			return
		}
		set := definitionSet{labeled: true, sections: []definitionSection{{label: "Spanish", entries: []string{raw}}, {label: "English", entries: []string{raw}}}}
		text, _ := renderDefinitions(set, RenderOpts{Width: 20 + int(w), Color: true})
		if i := subsequenceGap(alnum(raw), alnum(unstyled(text))); i >= 0 {
			t.Fatalf("lost input at %d: %q -> %q", i, raw, text)
		}
	})
}
