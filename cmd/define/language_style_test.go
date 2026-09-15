package main

import (
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"testing"
)

func TestLanguageTextValidation(t *testing.T) {
	for _, tc := range []struct {
		text  string
		spans []languageSpan
		valid bool
	}{
		{"árbol", []languageSpan{{0, 6, "es"}}, true},
		{"árbol", []languageSpan{{1, 6, "es"}}, false},
		{"word", []languageSpan{{0, 5, "en"}}, false},
		{"word", []languageSpan{{0, 3, "en"}, {2, 4, "es"}}, false},
		{"word", []languageSpan{{-1, 2, "en"}}, false},
		{"word", []languageSpan{{3, 2, "en"}}, false},
		{"word", nil, true},
		{"\x1b[32mword", []languageSpan{{2, 9, "en"}}, false},
	} {
		if got := validateLanguageText(languageText{tc.text, tc.spans}); got != tc.valid {
			t.Errorf("%q %+v: valid=%v want %v", tc.text, tc.spans, got, tc.valid)
		}
	}
}

func TestLanguageTintStyle(t *testing.T) {
	for _, tc := range []struct {
		name, text, want string
		lang             store.Lang
	}{
		{"plain", "hola", languageDark + "hola" + languageOff, "es"},
		{"unknown", "hola", "hola", ""},
		{"other", "hello", "hello", "en"},
		{"invalid tag", "hola", "hola", "\x1b[31m"},
		{"line whitespace", "  hola mundo  \n  adiós \n", "  " + languageDark + "hola mundo" + languageOff + "  \n  " + languageDark + "adiós" + languageOff + " \n", "es"},
		{"foreground reset", knownOn + "hola" + sgrOff + " mundo", knownOn + languageDark + "hola" + languageOff + sgrOff + languageDark + " mundo" + languageOff, "es"},
		{"semantic background", "\x1b[48;5;22mhola\x1b[0m mundo", "\x1b[48;5;22mhola\x1b[0m" + languageDark + " mundo" + languageOff, "es"},
		{"combined reset foreground", "\x1b[0;32mhola", "\x1b[0;32m" + languageDark + "hola" + languageOff, "es"},
		{"rgb foreground", "\x1b[38;2;0;0;0mhola", "\x1b[38;2;0;0;0m" + languageDark + "hola" + languageOff, "es"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lt := languageText{tc.text, []languageSpan{{0, len(tc.text), tc.lang}}}
			got := styleLanguageText(lt, tintPolicy{"es", languageDark})
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
			if stripANSI(got) != stripANSI(tc.text) || visibleCells(got) != visibleCells(tc.text) {
				t.Fatal("style changed text or width")
			}
		})
	}
}

func TestLanguageTintMixedAndSelection(t *testing.T) {
	lt := languageText{"A menudo means often.", []languageSpan{{0, 8, "es"}, {8, 21, "en"}}}
	got := styleLanguageText(lt, tintPolicy{"es", languageDark})
	if want := languageDark + "A menudo" + languageOff + " means often."; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := styleLanguageText(lt, tintPolicy{"en", languageLight}); got != "A menudo"+languageLight+" means often."+languageOff {
		t.Fatalf("English %q", got)
	}
	for _, p := range []tintPolicy{{"es", ""}, {"es", "\x1b[2J"}} {
		if styleLanguageText(lt, p) != lt.text {
			t.Fatal("disabled/invalid palette styled text")
		}
	}
	frame := newSelectionFrame(30, 1, []selectionRow{{styled: got, selectable: true}})
	text, err := selectedText(frame, selectionPoint{0, 0}, selectionPoint{0, 20})
	if err != nil || text != lt.text {
		t.Fatalf("copy=%q err=%v", text, err)
	}
	selected := frame.highlightRow(0, selectionPoint{0, 0}, selectionPoint{0, 7})
	if !strings.Contains(selected, "\x1b[7m") || stripANSI(selected) != lt.text {
		t.Fatalf("selection=%q", selected)
	}
}

func TestLanguageTintProfile(t *testing.T) {
	for _, tc := range []struct {
		profile, want string
		bad           bool
	}{{"dark", languageDark, false}, {"light", languageLight, false}, {"off", "", false}, {"bogus", "", true}} {
		got, err := tintProfile(tc.profile)
		if got != tc.want || (err != nil) != tc.bad {
			t.Errorf("%q: %q %v", tc.profile, got, err)
		}
	}
	for _, lang := range []store.Lang{"es", "en"} {
		p := (options{color: true, tintBackground: languageDark}).tintFor(lang)
		if p.lang != lang || p.background != languageDark {
			t.Errorf("policy=%+v", p)
		}
	}
	if p := (options{tintBackground: languageDark}).tintFor("es"); p.background != "" {
		t.Fatal("no-color tint")
	}
}
