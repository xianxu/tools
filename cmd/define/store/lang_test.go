package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A language tag is validated at the EDGE, once, so nothing downstream has to
// wonder whether "ES " or "" is a language.
//
// The rejections carry the weight here. The value becomes a PATH SEGMENT —
// words/<lang>/ — so "../etc" is not a malformed tag, it is a directory
// traversal, and ParseLang being the only way to build a Lang from input is what
// makes that unreachable rather than merely unlikely.
func TestParseLang(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    Lang
		wantErr bool
	}{
		{in: "en", want: "en"},
		{in: "es", want: "es"},
		{in: "ES", want: "es"},    // case-folded: a directory name is lowercase
		{in: " es\n", want: "es"}, // ReadLang hands us a file's bytes, newline and all
		{in: "", wantErr: true},
		{in: "english", wantErr: true},
		{in: "e", wantErr: true},
		{in: "e/s", wantErr: true},
		{in: "../etc", wantErr: true}, // a path segment; traversal is not a language
		{in: "..", wantErr: true},
		{in: "e1", wantErr: true},
	} {
		got, err := ParseLang(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseLang(%q) = %q, want an error", tc.in, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("ParseLang(%q) = (%q, %v), want (%q, nil)", tc.in, got, err, tc.want)
		}
	}
}

// Whatever ParseLang accepts must be safe to join onto a path — the property
// behind the table above, asserted directly so a future relaxation of the tag
// rule cannot quietly reintroduce traversal.
//
// The assertion is that filepath.Join cannot be made to LEAVE the base
// directory, which is the actual danger; an earlier version compared the tag
// against "/" and ".." and was subsumed by its own length check, so it tested
// nothing the table above had not.
func TestAcceptedLangIsASafePathSegment(t *testing.T) {
	base := t.TempDir()
	for _, in := range []string{"en", "es", "ZZ", "  de  "} {
		l, err := ParseLang(in)
		if err != nil {
			t.Fatalf("ParseLang(%q): %v", in, err)
		}
		joined := filepath.Join(base, "words", string(l))
		if !strings.HasPrefix(filepath.Clean(joined), filepath.Clean(base)+string(filepath.Separator)) {
			t.Errorf("ParseLang(%q) = %q, which escapes the deck directory: %s", in, l, joined)
		}
	}
}

// The setting survives the session, because a one-shot lookup has no session to
// inherit from. That is the one way language deliberately differs from /sound.
func TestLangRoundTrips(t *testing.T) {
	dir := t.TempDir()
	if got := ReadLang(dir); got != DefaultLang {
		t.Errorf("an unset directory = %q, want %q", got, DefaultLang)
	}
	if err := WriteLang(dir, Lang("es")); err != nil {
		t.Fatal(err)
	}
	if got := ReadLang(dir); got != "es" {
		t.Errorf("after WriteLang, ReadLang = %q, want es", got)
	}
	// And it is the file the guards know about, not some other name.
	if _, err := os.Stat(filepath.Join(dir, RuntimeFiles[1])); err != nil {
		t.Errorf("the setting is not at %s: %v", RuntimeFiles[1], err)
	}
}

// A garbage, empty or unreadable setting degrades to the default rather than
// failing the lookup: the learner asked for a word, not a configuration audit.
func TestABrokenLangFileFallsBackToTheDefault(t *testing.T) {
	for _, body := range []string{"not-a-language", "", "   ", "../etc", "en es"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, RuntimeFiles[1]), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := ReadLang(dir); got != DefaultLang {
			t.Errorf("ReadLang with %q on disk = %q, want the default", body, got)
		}
	}
}

// WriteLang refuses what ParseLang refuses — the file is written by /lang, and a
// bad value there would become a path segment on the next run.
func TestWriteLangRejectsAnInvalidTag(t *testing.T) {
	dir := t.TempDir()
	if err := WriteLang(dir, Lang("../etc")); err == nil {
		t.Error("WriteLang accepted a traversal as a language")
	}
	if _, err := os.Stat(filepath.Join(dir, RuntimeFiles[1])); !os.IsNotExist(err) {
		t.Error("a rejected language still wrote a file")
	}
}

// Trailing newline included, because ReadLang has to survive both what WriteLang
// produces and what a person types into the file with an editor.
func TestReadLangToleratesHowAPersonWouldWriteIt(t *testing.T) {
	for _, body := range []string{"es", "es\n", " ES \n", "es\r\n"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, RuntimeFiles[1]), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := ReadLang(dir); got != "es" {
			t.Errorf("ReadLang with %q on disk = %q, want es", body, got)
		}
	}
}
