package store_test

import (
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// The band parse is the boundary the whole distractor rule rests on: everything
// downstream compares bands, and a comparison is only meaningful over the six.
func TestParseBandAcceptsTheScaleAndRefusesEverythingElse(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want store.Band
	}{
		{"A1", store.A1},
		{"C2", store.C2},
		// Case and space are transcription noise, not a different answer.
		{"c1", store.C1},
		{" B2 ", store.B2},
		{"b1\n", store.B1},
	} {
		got, ok := store.ParseBand(tc.in)
		if !ok || got != tc.want {
			t.Errorf("ParseBand(%q) = %q, %v; want %q, true", tc.in, got, ok, tc.want)
		}
	}

	// Every one of these is a REAL answer to a badly-posed question, which is
	// exactly why they must be refused rather than coerced: an unorderable value
	// reaching Rank would sort below A1 and pitch selection at the floor.
	for _, in := range []string{
		"", "B2+", "intermediate", "C1-C2", "A", "3", "advanced", "B2 (high)", "D1", "A0",
	} {
		if got, ok := store.ParseBand(in); ok {
			t.Errorf("ParseBand(%q) = %q, true; want refused", in, got)
		}
	}
}

// Rank and Below are the arithmetic "the learner's band, or one below" compiles
// to, so their edges are the rule's edges.
func TestBandRankAndBelow(t *testing.T) {
	bands := store.Bands()
	if len(bands) != 6 {
		t.Fatalf("Bands() has %d entries, want the 6-point CEFR scale", len(bands))
	}
	for i, b := range bands {
		if got := b.Rank(); got != i {
			t.Errorf("%q.Rank() = %d, want %d", b, got, i)
		}
	}

	// A non-band ranks -1 and NOT 0, so a comparison written without a guard
	// fails loudly rather than treating unparsed input as A1.
	if got := store.Band("B2+").Rank(); got != -1 {
		t.Errorf("Band(\"B2+\").Rank() = %d, want -1", got)
	}

	for _, tc := range []struct {
		in   store.Band
		want store.Band
		ok   bool
	}{
		{store.C2, store.C1, true},
		{store.B1, store.A2, true},
		{store.A2, store.A1, true},
		// A1 has nothing below it; the caller must fall back to the learner's own
		// band rather than inventing a level.
		{store.A1, "", false},
		{store.Band("nonsense"), "", false},
	} {
		got, ok := tc.in.Below()
		if got != tc.want || ok != tc.ok {
			t.Errorf("%q.Below() = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// The domain set is closed, and an unrecognised value DEGRADES rather than
// widening it — the difference between a worse question and a broken measure.
func TestParseDomainIsClosedAndDegradesToGeneral(t *testing.T) {
	got, ok := store.ParseDomain("Law")
	if !ok || got != store.Domain("Law") {
		t.Errorf("ParseDomain(\"Law\") = %q, %v; want Law, true", got, ok)
	}

	// Case-folded, because the failure this guards is exactly "Medicine" and
	// "medicine" counting as two in a measure that counts distinct domains.
	for _, in := range []string{"medicine", "MEDICINE", " Medicine "} {
		got, ok := store.ParseDomain(in)
		if !ok || got != store.Domain("Medicine") {
			t.Errorf("ParseDomain(%q) = %q, %v; want Medicine, true", in, got, ok)
		}
	}

	if got, ok := store.ParseDomain("general"); !ok || got != store.DomainGeneral {
		t.Errorf("ParseDomain(\"general\") = %q, %v; want general, true", got, ok)
	}

	// Not an error, and not a new domain. A word whose domain we cannot name
	// draws from general vocabulary — a worse question, and a fine one.
	for _, in := range []string{"", "business news", "med", "Crypto", "Astrology"} {
		got, ok := store.ParseDomain(in)
		if ok {
			t.Errorf("ParseDomain(%q) reported a match — the set must stay closed", in)
		}
		if got != store.DomainGeneral {
			t.Errorf("ParseDomain(%q) = %q, want the general fallback", in, got)
		}
	}
}

// Domains() excludes general, because the two are asked for in different places:
// a scanner matching NOAD's printed prose wants the labels a dictionary actually
// prints, and "general" is what we call the ABSENCE of one.
func TestDomainsExcludesGeneralAndIsNotEmpty(t *testing.T) {
	ds := store.Domains()
	if len(ds) == 0 {
		t.Fatal("Domains() is empty; every test reading it would pass vacuously")
	}
	for _, d := range ds {
		if d == store.DomainGeneral {
			t.Error("Domains() includes general; NOAD never prints it as a field label")
		}
		if strings.TrimSpace(string(d)) != string(d) {
			t.Errorf("domain %q carries surrounding space", d)
		}
		if _, ok := store.ParseDomain(string(d)); !ok {
			t.Errorf("Domains() yields %q, which ParseDomain refuses", d)
		}
	}
}

// Callers must not be able to corrupt the closed sets by writing to what they
// were handed — the reason both accessors copy.
func TestVocabularyAccessorsCopy(t *testing.T) {
	ds := store.Domains()
	ds[0] = "Astrology"
	if store.Domains()[0] == "Astrology" {
		t.Error("Domains() hands out the backing array; a caller can widen the closed set")
	}
	bs := store.Bands()
	bs[0] = "Z9"
	if store.Bands()[0] == "Z9" {
		t.Error("Bands() hands out the backing array; a caller can rewrite the scale")
	}
}
