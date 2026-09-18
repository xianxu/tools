package store

import "testing"

func TestParseScheme(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Scheme
		ok   bool
	}{
		{"dark", SchemeDark, true}, {"light", SchemeLight, true},
		{" Light\n", SchemeLight, true}, {"DARK", SchemeDark, true},
		{"", "", false}, {"auto", "", false}, {"solarized", "", false},
	} {
		got, err := ParseScheme(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseScheme(%q) = %q, %v; want %q, ok=%v", tc.in, got, err, tc.want, tc.ok)
		}
	}
}
