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

// The rule that keeps `(also …)` mining honest, case by measured case (#29).
//
// Surveyed over 400 live NOAD entries (2026-08-29), `(also …)` is NOT a spelling
// list: it holds phrases, compounds, derivatives and genuine English variants.
// Each one admitted costs a wasted pair of CDN requests at ~300–600ms per miss.
// This predicate took all three real gains — café, naïve, façade — and, across
// that 400-entry sample, nothing else at all.
func TestDiacriticsOnlyAdmitsTheSameWordInAnotherDress(t *testing.T) {
	for _, tc := range []struct {
		name, alt, head string
		want            bool
	}{
		{"an accent added", "café", "cafe", true},
		{"a diaeresis added", "naïve", "naive", true},
		{"a cedilla added", "façade", "facade", true},
		{"a circumflex added", "rôle", "role", true},
		{"an accent REMOVED — the headword is the accented one", "cliche", "cliché", true},
		{"a tilde, mid-word", "señor", "senor", true},
		{"case is not a difference that matters", "Señor", "senor", true},

		{"a different English spelling is not a diacritic", "advisor", "adviser", false},
		{"nor is another one", "convertor", "converter", false},
		{"a dropped letter changes the length", "caldron", "cauldron", false},
		{"a derivative is a different word", "naïveness", "naive", false},
		{"a compound is not a spelling", "jalapeño pepper", "jalapeño", false},
		{"a phrase certainly is not", "good as gold", "gold", false},
		{"identical is not a DIFFERENT dress", "cafe", "cafe", false},
		{"identical up to case is still not one", "Cafe", "cafe", false},
		{"empty alternative", "", "cafe", false},
		{"empty headword", "café", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := differsOnlyByDiacritics(tc.alt, tc.head); got != tc.want {
				t.Errorf("differsOnlyByDiacritics(%q, %q) = %v, want %v", tc.alt, tc.head, got, tc.want)
			}
		})
	}
}

// The class the table above cannot reach: this predicate runs over arbitrary
// NOAD gloss text, so what matters is not more equality cases but that hostile
// or malformed input cannot panic it or smuggle a non-spelling through (#29).
//
// The interesting cell is INVALID UTF-8, and it is not a crash: []rune turns a
// stray byte into U+FFFD, which is outside ASCII and therefore looks exactly
// like a diacritic. "caf\xff" would be admitted against "cafe" and spend two CDN
// requests on a URL-escaped replacement character. Valid UTF-8 is a precondition
// of being a spelling at all, so it is checked rather than assumed.
func TestDiacriticsOnlyRefusesMalformedInput(t *testing.T) {
	for _, tc := range []struct {
		name, alt, head string
	}{
		{"a stray byte is not a diacritic", "caf\xff", "cafe"},
		{"nor is one in the headword", "café", "caf\xff"},
		{"a decomposed accent is longer than a precomposed one", "café", "café"},
		{"and longer than the bare word", "café", "cafe"},
		{"a zero-width joiner is a rune, so the lengths differ", "cafe‍", "cafe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if differsOnlyByDiacritics(tc.alt, tc.head) {
				t.Errorf("differsOnlyByDiacritics(%q, %q) admitted malformed input", tc.alt, tc.head)
			}
		})
	}
}

// Whatever arrives, it returns. A gloss is untrusted text of unbounded length.
func FuzzDiacriticsOnlyDoesNotPanic(f *testing.F) {
	f.Add("café", "cafe")
	f.Add("caf\xff", "cafe")
	f.Add(strings.Repeat("ñ", 4096), strings.Repeat("n", 4096))
	f.Add("", "")
	f.Fuzz(func(t *testing.T, alt, head string) {
		// Both directions, because AlsoSpellings controls neither argument.
		_ = differsOnlyByDiacritics(alt, head)
		_ = differsOnlyByDiacritics(head, alt)
	})
}

// `(also X)` lands as an UNNUMBERED sense in the POS-less first block — verified
// against the live dictionary 2026-08-29, e.g. cafe parses to
// Blocks[0]{POS:"", Senses:[{Number:"", Gloss:"(also café)"}]}. The accessor
// derives from that shape rather than re-scanning Raw, so it cannot disagree
// with what Render shows (#29).
func TestAlsoSpellingsTakesOnlyTheSourceOrthographies(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      []string
	}{
		{
			"the accented alternative NOAD files under an unaccented head",
			"cafe ca·fe | kaˈfā | (also café) noun 1 a small restaurant.",
			[]string{"café"},
		},
		{
			"a compound is not a spelling",
			"jalapeño ja·la·pe·ño | ˌhaləˈpān(y)ō | (also jalapeño pepper) noun a chili pepper.",
			nil,
		},
		{
			"an English spelling variant is not a source orthography",
			"adviser ad·vis·er | ədˈvīzər | (also advisor) noun a person who advises.",
			nil,
		},
		{
			"no parenthetical at all",
			"arrondissement ar·ron·disse·ment | əˈrändəsmənt | noun an administrative district.",
			nil,
		},
		// ONE gloss, TWO parentheticals — measured, not supposed: ParseEntry
		// returns Gloss == "(also naïve) (also naïveness)" for the first of these.
		// A first-match read passes the first case and silently drops the source
		// spelling in the second, so both orderings are here on purpose.
		{
			"two alternatives, the spelling written first",
			"naive na·ive | näˈēv | (also naïve) (also naïveness) adjective of a person.",
			[]string{"naïve"},
		},
		{
			"two alternatives, the spelling written SECOND",
			"naive na·ive | näˈēv | (also naïveness) (also naïve) adjective of a person.",
			[]string{"naïve"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseEntry(tc.raw).AlsoSpellings()
			if !slices.Equal(got, tc.want) {
				t.Errorf("AlsoSpellings() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A COLON INSIDE PARENTHESES IS NOT THE GLOSS/EXAMPLE BOUNDARY.
//
// Reported from a real sitting: `potassium` rendered as a gloss of "(Symbol" and
// an example beginning "K) the chemical element of atomic number 19…". NOAD
// writes the element symbol as a parenthetical — "(Symbol: K)" — and newSense
// took the FIRST colon anywhere, so it split inside the brackets and threw the
// entire definition into a quoted example.
//
// This is a lookup bug before it is a review bug: `define potassium` printed the
// meaning as an example. Form 2.3 made it visible by offering "(Symbol" as an
// answer, which is why it was noticed.
//
// The file already had delimiterDepths for exactly this class — firstSenseNumber
// uses it to ignore numerals inside brackets — so the fix is to apply the tool
// that was already here rather than to add one.
func TestSenseSplitIgnoresColonsInsideBrackets(t *testing.T) {
	for _, tc := range []struct {
		name, text, wantGloss string
		wantExamples          int
	}{
		{
			"the measured case",
			"(Symbol: K) the chemical element of atomic number 19, a soft silvery-white reactive metal",
			"(Symbol: K) the chemical element of atomic number 19, a soft silvery-white reactive metal", 0,
		},
		{
			"a bracketed label with a colon still opens the examples after it",
			"the land alongside a river: willows lined the bank",
			"the land alongside a river", 1,
		},
		{
			"a parenthetical colon before a real one",
			"(Symbol: Na) a soft metal: sodium burns in air",
			"(Symbol: Na) a soft metal", 1,
		},
		{
			"square brackets too",
			"[Chem.: see note] an element",
			"[Chem.: see note] an element", 0,
		},
		{
			"no colon at all",
			"a small short-tailed wallaby",
			"a small short-tailed wallaby", 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := newSense("1", false, tc.text)
			if got.Gloss != tc.wantGloss {
				t.Errorf("gloss = %q,\n want %q", got.Gloss, tc.wantGloss)
			}
			if len(got.Examples) != tc.wantExamples {
				t.Errorf("%d examples, want %d: %+v", len(got.Examples), tc.wantExamples, got.Examples)
			}
		})
	}
}
