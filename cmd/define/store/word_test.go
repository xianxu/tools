package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestKeyNormalises(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Define", "define"},
		{"  sycophantic  ", "sycophantic"},
		{"hot   dog", "hot dog"},
		{"HOT DOG", "hot dog"},
	} {
		if got := Key(tc.in); got != tc.want {
			t.Errorf("Key(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSlugIsReadableForOrdinaryWords(t *testing.T) {
	for _, tc := range []struct{ key, want string }{
		{"sycophantic", "sycophantic"},
		{"hot dog", "hot-dog"},
		{"a priori", "a-priori"},
	} {
		if got := Slug(tc.key); got != tc.want {
			t.Errorf("Slug(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

// The collision that matters: a hyphenated word and a two-word phrase normalise
// to the same base. They must never share a file — one would silently overwrite
// the other's definition history.
func TestSlugDoesNotMergeHyphenAndSpace(t *testing.T) {
	a, b := Slug("hot dog"), Slug("hot-dog")
	if a == b {
		t.Fatalf("both slugged to %q — two different words would share a file", a)
	}
	if !strings.HasPrefix(b, "hot-dog-") {
		t.Errorf("Slug(%q) = %q, want the readable base plus a hash", "hot-dog", b)
	}
}

// A slug is always exactly one safe path element. This function turns arbitrary
// dictionary input into a filename, so it is the one place a traversal could
// enter.
func TestSlugIsAlwaysOneSafePathElement(t *testing.T) {
	for _, key := range []string{
		"../etc/passwd", "/absolute", ".", "..", "", "  ", ".hidden",
		"with/slash", `with\backslash`, "with\x00null", "\n\t",
	} {
		got := Slug(key)
		if got == "" || got == "." || got == ".." {
			t.Errorf("Slug(%q) = %q", key, got)
		}
		if strings.ContainsAny(got, `/\`) {
			t.Errorf("Slug(%q) = %q contains a separator", key, got)
		}
		if filepath.Base(got) != got {
			t.Errorf("Slug(%q) = %q is not a single element", key, got)
		}
		if strings.HasPrefix(got, ".") {
			t.Errorf("Slug(%q) = %q is a dotfile", key, got)
		}
	}
}

func TestSlugIsDeterministic(t *testing.T) {
	for _, k := range []string{"hot-dog", "café", "../x"} {
		if Slug(k) != Slug(k) {
			t.Errorf("Slug(%q) is not stable", k)
		}
	}
}

// Non-ASCII headwords exist in the corpus and should stay readable rather than
// being mangled into a hash.
func TestSlugKeepsNonASCIIReadable(t *testing.T) {
	if got := Slug("café"); got != "café" {
		t.Errorf("Slug(café) = %q, want it readable", got)
	}
}

// Slug converts arbitrary text into a path. Fuzz its safety property.
func FuzzSlugIsSafe(f *testing.F) {
	for _, s := range []string{"a", "hot dog", "../x", "", ".", "..", "a/b", "café", "\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, key string) {
		got := Slug(Key(key))
		if got == "" || got == "." || got == ".." || strings.ContainsAny(got, `/\`) {
			t.Fatalf("Slug(%q) = %q is not a safe path element", key, got)
		}
		if filepath.Base(got) != got {
			t.Fatalf("Slug(%q) = %q is not a single element", key, got)
		}
		if strings.HasPrefix(got, ".") {
			t.Fatalf("Slug(%q) = %q is a dotfile", key, got)
		}
	})
}

// The traversal guard, exercised at its own level.
//
// Inlined behind Slug it was unreachable in principle: Slug sanitises first, so
// no test could tell the guard from Slug and deleting it left every test green.
// Taking a slug rather than a key makes it a real net under a future Slug
// regression — these are names Slug does not currently produce, and that is the
// point.
func TestWordFileNameRefusesUnsafeNames(t *testing.T) {
	for _, bad := range []string{
		"", ".", "..", "../etc/passwd", "/absolute", `back\slash`, ".hidden", "a/b",
	} {
		if got, err := wordFileName(bad); err == nil {
			t.Errorf("wordFileName(%q) = %q, want an error", bad, got)
		}
	}
	for _, ok := range []string{"sycophantic", "hot-dog", "café", "hot-dog-3f9a1c"} {
		got, err := wordFileName(ok)
		if err != nil {
			t.Errorf("wordFileName(%q) failed: %v", ok, err)
		}
		if got != ok+".yaml" {
			t.Errorf("wordFileName(%q) = %q", ok, got)
		}
	}
}
