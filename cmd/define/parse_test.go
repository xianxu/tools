package main

import (
	"slices"
	"strings"
	"testing"
)

func fixture(t *testing.T, word string) string {
	t.Helper()
	s, err := testDict(t).Lookup(word)
	if err != nil {
		t.Fatalf("fixture %s: %v", word, err)
	}
	return s
}

// Rule B's verdict table from the plan. Discriminating on "contains no ASCII
// letters" does NOT work — ˈrekərd and baNGk are mostly ASCII letters — so the
// rule is word-shape: every comma-separated part must be a single token.
func TestIsPronunciation(t *testing.T) {
	tests := []struct {
		inner string
		want  bool
	}{
		{"ˈrekərd", true},
		{"baNGk", true},
		{"rən", true},
		{"set", true},
		{"əˈfem(ə)rəl", true},
		{"ˌsikəˈfan(t)ik", true},
		{"ˌsikəˈfan(t)ək(ə)lē, -ˈfantik(ə)lē", true}, // comma-separated variants
		{"identification was made through dental records", false},
		{"[as modifier] : record profits", false},
		{"she ran the last few yards, breathing heavily", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isPronunciation(tc.inner); got != tc.want {
			t.Errorf("isPronunciation(%q) = %v, want %v", tc.inner, got, tc.want)
		}
	}
}

// Rule A's header table from the plan.
func TestParseHeader(t *testing.T) {
	tests := []struct {
		name, raw, word, syl, homo, ipa string
	}{
		{
			"simple",
			"sycophantic syc·o·phan·tic | ˌsikəˈfan(t)ik | adjective behaving badly.",
			"sycophantic", "syc·o·phan·tic", "", "ˌsikəˈfan(t)ik",
		},
		{
			"homograph, monosyllable, no syllabification",
			"bank 1 | baNGk | noun 1 the land alongside a river.",
			"bank", "", "1", "baNGk",
		},
		{
			"part-of-speech glued to syllabification",
			"record rec·ordnoun | ˈrekərd | 1 a thing constituting evidence.",
			"record", "rec·ord", "", "ˈrekərd",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseEntry(tc.raw)
			if e.Headword() != tc.word {
				t.Errorf("Headword = %q, want %q", e.Headword(), tc.word)
			}
			if e.Syllables() != tc.syl {
				t.Errorf("Syllables = %q, want %q", e.Syllables(), tc.syl)
			}
			if e.Homograph() != tc.homo {
				t.Errorf("Homograph = %q, want %q", e.Homograph(), tc.homo)
			}
			if e.IPA != tc.ipa {
				t.Errorf("IPA = %q, want %q", e.IPA, tc.ipa)
			}
		})
	}
}

// The glued part-of-speech is not decoration: record's body starts at "1 a
// thing…" with no POS token of its own, so a parser that drops the glued "noun"
// yields a first block with an empty POS.
func TestParseGluedPOSOpensFirstBlock(t *testing.T) {
	e := ParseEntry(fixture(t, "record"))
	if len(e.Blocks) < 2 {
		t.Fatalf("want at least 2 blocks, got %d", len(e.Blocks))
	}
	if e.Blocks[0].POS != "noun" {
		t.Errorf("Blocks[0].POS = %q, want %q (from the glued suffix)", e.Blocks[0].POS, "noun")
	}
}

// record's verb block carries its own pronunciation, distinct from the head's.
func TestParsePerBlockIPA(t *testing.T) {
	e := ParseEntry(fixture(t, "record"))
	var verb *Block
	for i := range e.Blocks {
		if e.Blocks[i].POS == "verb" {
			verb = &e.Blocks[i]
			break
		}
	}
	if verb == nil {
		t.Fatal("no verb block")
	}
	if verb.IPA != "rəˈkôrd" {
		t.Errorf("verb block IPA = %q, want %q", verb.IPA, "rəˈkôrd")
	}
	if e.IPA != "ˈrekərd" {
		t.Errorf("entry IPA = %q, want %q", e.IPA, "ˈrekərd")
	}
}

// The interior pipes of a sense separate examples and must not be mistaken for
// a pronunciation span.
func TestParseExamplesSplitOnInteriorPipes(t *testing.T) {
	e := ParseEntry(fixture(t, "record"))
	var found []Example
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			for _, ex := range s.Examples {
				if strings.Contains(ex.Text, "dental records") {
					found = s.Examples
				}
			}
		}
	}
	if len(found) < 3 {
		t.Fatalf("want the sense split into >=3 examples, got %d: %+v", len(found), found)
	}
	for _, ex := range found {
		if isPronunciation(ex.Text) {
			t.Errorf("example %q was shaped like a pronunciation", ex.Text)
		}
	}
}

func TestParseBlocksAndSections(t *testing.T) {
	e := ParseEntry(fixture(t, "ephemeral"))
	var pos []string
	for _, b := range e.Blocks {
		pos = append(pos, b.POS)
	}
	if !slices.Equal(pos, []string{"adjective", "noun"}) {
		t.Errorf("blocks = %v, want [adjective noun]", pos)
	}
	var names []string
	for _, s := range e.Sections {
		names = append(names, s.Name)
	}
	if !slices.Equal(names, []string{"DERIVATIVES", "ORIGIN"}) {
		t.Errorf("sections = %v, want [DERIVATIVES ORIGIN]", names)
	}
	if got := e.Blocks[0].Senses[0].Examples; len(got) != 1 || got[0].Text != "fashions are ephemeral" {
		t.Errorf("examples = %+v", got)
	}
}

func TestParseRawIsRetained(t *testing.T) {
	raw := fixture(t, "quokka")
	if ParseEntry(raw).Raw != raw {
		t.Error("Raw must be retained verbatim")
	}
}

// A grammar label introducing an example belongs outside the quotes, not inside
// them with a stray colon. bank is the fixture the issue's Done-when names.
func TestParseExampleGrammarLabel(t *testing.T) {
	e := ParseEntry(fixture(t, "bank"))
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			for _, ex := range s.Examples {
				if strings.Contains(ex.Text, "[") || strings.HasPrefix(ex.Text, ":") {
					t.Errorf("example text still carries a label or colon: %q", ex.Text)
				}
				if strings.Contains(ex.Text, "bank shot") && ex.Label != "[as modifier]" {
					t.Errorf("label = %q, want %q", ex.Label, "[as modifier]")
				}
			}
		}
	}
}
