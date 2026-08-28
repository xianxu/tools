package store

import "testing"

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
func TestAcceptedLangIsASafePathSegment(t *testing.T) {
	for _, in := range []string{"en", "es", "ZZ", "  de  "} {
		l, err := ParseLang(in)
		if err != nil {
			t.Fatalf("ParseLang(%q): %v", in, err)
		}
		for _, bad := range []string{"/", `\`, ".", ".."} {
			if string(l) == bad || len(string(l)) != 2 {
				t.Errorf("ParseLang(%q) = %q, which is not a two-letter path segment", in, l)
			}
		}
	}
}
