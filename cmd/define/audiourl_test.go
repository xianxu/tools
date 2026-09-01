package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestAudioCandidatesOrder(t *testing.T) {
	got := AudioCandidates("Sycophantic", voice{Lang: "en", Locale: "us"}) // input case must not matter
	want := []string{
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_1.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_2.mp3",
		audioBase + "/sounds/oxford/sycophantic--_us_1.mp3",
		audioBase + "/sounds/oxford/sycophantic--_us_2.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestAudioCandidatesLocale(t *testing.T) {
	for _, u := range AudioCandidates("sycophantic", voice{Lang: "en", Locale: "gb"}) {
		if !strings.Contains(u, "_gb_") && !strings.Contains(u, "--_gb_") {
			t.Errorf("locale not applied: %s", u)
		}
	}
	// An empty locale defaults to us rather than producing "_en__1".
	for _, u := range AudioCandidates("sycophantic", voice{Lang: "en"}) {
		if strings.Contains(u, "__") {
			t.Errorf("empty locale leaked into %s", u)
		}
	}
}

// A one-letter word must not panic on the two-letter shard prefix.
func TestAudioCandidatesShortWord(t *testing.T) {
	got := AudioCandidates("a", voice{Lang: "en", Locale: "us"})
	if len(got) == 0 {
		t.Fatal("no candidates for a single-letter word")
	}
	if !strings.Contains(got[0], "/audio/a/a_en_us_1.mp3") {
		t.Errorf("unexpected shard for a single-letter word: %s", got[0])
	}
	if AudioCandidates("", voice{Lang: "en", Locale: "us"}) != nil {
		t.Error("empty word should yield no candidates")
	}
}

// Multi-word headwords are keyed with the spaces REMOVED — "hot dog" is `hotdog`.
//
// This test used to assert `hot_dog_en_us_1.mp3` and so PINNED a bug for the life
// of the feature: every multi-word headword in the corpus (hot dog, a priori)
// missed, and missed silently, because a miss is a supported outcome here.
//
// Two things made it invisible, and both are worth naming since neither is
// specific to this file. The assertion above it — "must not produce a URL with a
// space" — is satisfied by ANY separator, so it looked like coverage while
// discriminating nothing. And the value it did assert came from the comment on
// the implementation rather than from the server: the underscore was never
// measured, it was assumed, and the test then froze the assumption.
//
// So the shard is asserted too. `hotdog` shards under `ho`; `hot_dog` would shard
// under `ho` as well, which is precisely why the shard could not catch this and
// the filename has to.
func TestAudioCandidatesMultiWord(t *testing.T) {
	got := AudioCandidates("hot dog", voice{Lang: "en", Locale: "us"})
	if len(got) == 0 {
		t.Fatal("no candidates")
	}
	// Measured against the live CDN, 2026-08-30: hotdog_en_us_1 is 200 while
	// hot_dog_en_us_1 and hot-dog_en_us_1 are both 404. Pinned live by
	// TestCDNKeysPhrasesWithoutSeparators in fetch_conformance_test.go.
	if !strings.Contains(got[0], "/ho/hotdog_en_us_1.mp3") {
		t.Errorf("multi-word key = %s, want the spaces removed", got[0])
	}
	for _, u := range got {
		// No space, and no separator standing in for one either.
		if strings.Contains(u, " ") || strings.Contains(u, "%20") {
			t.Errorf("unescaped space in %s", u)
		}
		if strings.Contains(u, "hot_dog") || strings.Contains(u, "hot-dog") {
			t.Errorf("%s uses a separator the CDN 404s on", u)
		}
	}
}

// One language, and the legacy path for English only.
//
// Still true after #29, and enumerated in that issue's sweep rather than left
// unexamined: this documents AudioCandidates, which really does build for one
// voice. The cross-language walk is utterance.Candidates, one level up.
//
// Measured 2026-08-28: madrugar--_us_1 and madrugar--_es_1 are BOTH 404 while
// sycophantic--_us_1 is 200, so /sounds/oxford/ is an English-only generation.
// A Spanish candidate there is a guaranteed miss at ~450ms, which is the whole
// cost of guessing instead of asking.
func TestAudioCandidatesSpanish(t *testing.T) {
	got := AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"})
	want := []string{
		audioBase + "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_2.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// English is unchanged, legacy pair included. This is the regression that must
// not happen: every recording the tool plays today comes from these four URLs.
func TestAudioCandidatesEnglishUnchanged(t *testing.T) {
	got := AudioCandidates("sycophantic", voice{Lang: "en", Locale: "us"})
	want := []string{
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_1.mp3",
		audioBase + "/pronunciation/2022-03-02/audio/sy/sycophantic_en_us_2.mp3",
		audioBase + "/sounds/oxford/sycophantic--_us_1.mp3",
		audioBase + "/sounds/oxford/sycophantic--_us_2.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The locale policy: -locale is honoured for EVERY language, and nothing here
// whitelists which pairs exist.
//
// It was English-only for one milestone — #23 M1's D2, an explicit interim rule
// written to be replaced here. #27 replaced it, and deliberately did NOT replace
// it with a table of valid pairs: that would restate a fact the CDN owns and go
// stale when Google adds a variant, which is the same argument ParseLang makes
// for not enumerating languages. Using a different philosophy for the adjacent
// field would be the inconsistency.
//
// Measured 2026-08-28: es_es and es_us both serve (including the phonemic pair
// cazar/casar), es_419 and es_mx do not, and en_gb serves on both the 2022 and
// legacy paths. The rows below encode the POLICY, not that measurement — the
// conformance tests own the measurement.
func TestLocaleFor(t *testing.T) {
	for _, tc := range []struct {
		name string
		lang store.Lang
		flag string
		want string
	}{
		{name: "English defaults to us", lang: "en", want: "us"},
		{name: "English honours the flag", lang: "en", flag: "gb", want: "gb"},
		{name: "Spanish defaults to es", lang: "es", want: "es"},
		// THE change #27 makes. This was refused with a diagnostic before, and
		// es_us is a real recording — the Latin American seseo, phonemically
		// distinct from Castilian es_es, which is the whole point of the flag.
		{name: "Spanish honours the flag: seseo", lang: "es", flag: "us", want: "us"},
		{name: "an unknown language is its own locale", lang: "de", want: "de"},
		{name: "and honours the flag too", lang: "fr", flag: "ca", want: "ca"},
		// NOT rejected. es_gb 404s and degrades to the warning every missing
		// recording produces; the CDN is the authority on what exists, not a
		// table here.
		{name: "an unserved pair is built, not refused", lang: "es", flag: "gb", want: "gb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := localeFor(tc.lang, tc.flag); got != tc.want {
				t.Errorf("localeFor(%q, %q) = %q, want %q", tc.lang, tc.flag, got, tc.want)
			}
		})
	}
}

// The Spanish locales build the URLs the CDN actually serves — the pair this
// issue exists to make reachable.
func TestAudioCandidatesSpanishLocales(t *testing.T) {
	for _, tc := range []struct {
		locale string
		want   string
	}{
		{locale: "es", want: audioBase + "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3"},
		{locale: "us", want: audioBase + "/pronunciation/2022-03-02/audio/ma/madrugar_es_us_1.mp3"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			got := AudioCandidates("madrugar", voice{Lang: "es", Locale: tc.locale})
			if len(got) == 0 || got[0] != tc.want {
				t.Errorf("first candidate = %v, want %q", got, tc.want)
			}
			// Still no legacy pair for Spanish, whatever the locale.
			for _, u := range got {
				if strings.Contains(u, "/sounds/oxford/") {
					t.Errorf("a Spanish candidate reached the English-only legacy path: %s", u)
				}
			}
		})
	}
}

// The mix-up the struct exists to prevent: "es" is a legal value of both fields,
// so a transposed pair must not silently build a plausible URL.
func TestVoiceForBuildsBothFieldsTogether(t *testing.T) {
	v := voiceFor("es", "")
	if v.Lang != "es" || v.Locale != "es" {
		t.Errorf("voiceFor(es) = %+v", v)
	}
	if got := AudioCandidates("madrugar", v); len(got) != 2 {
		t.Errorf("a Spanish voice produced %d candidates, want the 2022 pair only", len(got))
	}
}

// The integration assertion no unit test covers: what the FETCH LOOP actually
// asks the CDN for, given a language.
//
// AudioCandidates being right is not the deliverable — the deliverable is that
// a Spanish session never spends a request on an English URL. Measured
// 2026-08-28, a miss costs ~300-600ms against ~40ms for a hit, so the two legacy
// URLs a language-blind version would try are most of a second per lookup, every
// lookup, for a guaranteed 404.
// The scope is "the session's language" — meaning NO -pron was given. With one,
// the loop deliberately asks for another language first (#29,
// TestAnUtteranceAsksTheSourceFirstAndFallsBackToTheSession). The name says
// "only" and that remains true of every case below, all of which build their
// candidates straight from AudioCandidates; it would be false of an utterance
// that was handed a source. A name asserting an invariant the tree no longer
// holds is the same defect as a comment doing it.
func TestTheFetchLoopAsksOnlyForTheSessionsLanguageWhenNoneWasNamed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		v       voice
		present string
		want    []string
	}{
		{
			name:    "Spanish asks the 2022 path only",
			v:       voice{Lang: "es", Locale: "es"},
			present: "/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3",
			want:    []string{"/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3"},
		},
		{
			// The negative half, and the one that matters: a Spanish word with no
			// recording must NOT fall back onto the English-only legacy path.
			name: "a Spanish miss never reaches the legacy path",
			v:    voice{Lang: "es", Locale: "es"},
			want: []string{
				"/pronunciation/2022-03-02/audio/ma/madrugar_es_es_1.mp3",
				"/pronunciation/2022-03-02/audio/ma/madrugar_es_es_2.mp3",
			},
		},
		{
			name: "English still walks all four",
			v:    voice{Lang: "en", Locale: "us"},
			want: []string{
				"/pronunciation/2022-03-02/audio/ma/madrugar_en_us_1.mp3",
				"/pronunciation/2022-03-02/audio/ma/madrugar_en_us_2.mp3",
				"/sounds/oxford/madrugar--_us_1.mp3",
				"/sounds/oxford/madrugar--_us_2.mp3",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			present := map[string][]byte{}
			if tc.present != "" {
				present[tc.present] = []byte("ID3 fake mp3 body")
			}
			cdn := newFakeCDN(t, present)

			var candidates []string
			for _, u := range AudioCandidates("madrugar", tc.v) {
				candidates = append(candidates, cdn.URL+stripHost(t, u, audioBase))
			}
			_, _, _ = cdn.source().Fetch(t.Context(), candidates)

			if got := cdn.Requested(); !slices.Equal(got, tc.want) {
				t.Errorf("requested:\n  %s\nwant:\n  %s",
					strings.Join(got, "\n  "), strings.Join(tc.want, "\n  "))
			}
		})
	}
}

// The locale is user input and reaches a URL path, so it is escaped like the
// word. Unescaped, a value containing "/" rewrites the path instead of missing
// cleanly — and a clean miss is the contract, since nothing whitelists which
// locales exist.
func TestAudioCandidatesEscapesTheLocale(t *testing.T) {
	for _, u := range AudioCandidates("madrugar", voice{Lang: "es", Locale: "a/b"}) {
		if strings.Contains(u, "_a/b_") {
			t.Errorf("an unescaped locale rewrote the path: %s", u)
		}
		if !strings.Contains(u, "a%2Fb") {
			t.Errorf("the locale was not escaped: %s", u)
		}
	}
}

// The ORDER is the contract: each spelling costs two CDN requests before the
// next is tried, at ~300–600ms per miss (#29).
func TestSourceSpellingsPutsTheAccentedFormFirst(t *testing.T) {
	for _, tc := range []struct {
		name, typed, raw string
		want             []string
	}{
		{
			"the headword carries the accent the typist omitted",
			// jalapeño_es_es is a 200; jalapeno_es_es is a 404 — measured
			// 2026-08-28. The typed form stays as the last resort.
			"jalapeno",
			"jalapeño ja·la·pe·ño | ˌhaləˈpān(y)ō | noun a chili pepper.",
			[]string{"jalapeño", "jalapeno"},
		},
		{
			// This cell is the one a headword-first ordering gets wrong: NOAD
			// heads this entry `cafe` and files `café` as the alternative, and it
			// is `café` the CDN serves.
			"the alternative carries it instead, so the alternative goes first",
			"cafe",
			"cafe ca·fe | kaˈfā | (also café) noun 1 a small restaurant.",
			[]string{"café", "cafe"},
		},
		{
			"nothing differs, so there is one spelling and no wasted request",
			"arrondissement",
			"arrondissement ar·ron·disse·ment | əˈrändəsmənt | noun a district.",
			[]string{"arrondissement"},
		},
		{
			"a capitalised headword is lowered — Señor_es_es is 404, señor_es_es is 200",
			"senor",
			"Señor Se·ñor | sānˈyôr | noun a title for a Spanish man.",
			[]string{"señor", "senor"},
		},
		{
			"an unparseable entry still leaves the typed word to try",
			"ciao",
			"",
			[]string{"ciao"},
		},
		{
			"among equals the source order stands — sorting is a partition, not a rank",
			"cafe",
			"café ca·fé | kaˈfā | (also cafè) noun 1 a small restaurant.",
			[]string{"café", "cafè", "cafe"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SourceSpellings(tc.typed, ParseEntry(tc.raw))
			if !slices.Equal(got, tc.want) {
				t.Errorf("SourceSpellings(%q) = %q, want %q", tc.typed, got, tc.want)
			}
		})
	}
}

func TestAnUtteranceAsksTheSourceFirstAndFallsBackToTheSession(t *testing.T) {
	en := voice{Lang: "en", Locale: "us"}
	es := voice{Lang: "es", Locale: "es"}
	it := voice{Lang: "it", Locale: "it"}

	// The no-regression assertion for every caller that never asks for a source:
	// the walk must be the SAME BYTES it was before #29, not merely equivalent.
	t.Run("no source asked for leaves the ordinary walk untouched", func(t *testing.T) {
		u := utterance{Word: "sycophantic", Session: en}
		if got, want := u.Candidates(), AudioCandidates("sycophantic", en); !slices.Equal(got, want) {
			t.Errorf("an utterance with no source changed the ordinary walk:\ngot  %q\nwant %q", got, want)
		}
	})

	t.Run("every source spelling is tried before the session's own", func(t *testing.T) {
		u := utterance{Word: "jalapeno", Spellings: []string{"jalapeño", "jalapeno"}, Source: es, Session: en}
		var want []string
		want = append(want, AudioCandidates("jalapeño", es)...)
		want = append(want, AudioCandidates("jalapeno", es)...)
		want = append(want, AudioCandidates("jalapeno", en)...)
		if got := u.Candidates(); !slices.Equal(got, want) {
			t.Errorf("walk order wrong:\ngot  %q\nwant %q", got, want)
		}
		// The walk ENDS at the session's recording, which is what makes a source
		// miss degrade rather than go silent: fr coverage is partial (hotel and
		// debut are 404 on fr_fr, 200 on en_us) and Italian is absent entirely.
		// By MEMBERSHIP, not by matching "_en_us_" in the string. An English walk
		// ends on the LEGACY path — /sounds/oxford/jalapeno--_us_2.mp3 — which
		// carries no such marker, so a substring check calls the right answer
		// wrong. Same rule spokeSource follows one level down.
		got := u.Candidates()
		if last := got[len(got)-1]; !slices.Contains(AudioCandidates(u.Word, u.Session), last) {
			t.Errorf("the walk does not end at the session's recording: %q", last)
		}
	})

	t.Run("a source equal to the session is not asked for twice", func(t *testing.T) {
		u := utterance{Word: "madrugar", Spellings: []string{"madrugar"}, Source: es, Session: es}
		if got, want := u.Candidates(), AudioCandidates("madrugar", es); !slices.Equal(got, want) {
			t.Errorf("/pron es in a Spanish session doubled the walk:\ngot  %q\nwant %q", got, want)
		}
	})

	// By MEMBERSHIP in the list actually built, never by reading the URL: the
	// report this feeds is a record, and a record has to be true.
	t.Run("spokeSource answers by membership, not by parsing", func(t *testing.T) {
		u := utterance{Word: "ciao", Spellings: []string{"ciao"}, Source: it, Session: en}
		if src := AudioCandidates("ciao", it)[0]; !u.spokeSource(src) {
			t.Errorf("a source URL was not recognised as one: %q", src)
		}
		if ses := AudioCandidates("ciao", en)[0]; u.spokeSource(ses) {
			t.Errorf("the session's URL was reported as a source one: %q", ses)
		}
		if u.spokeSource("https://example.invalid/nothing.mp3") {
			t.Error("a URL in neither list was reported as a source one")
		}
		// And with no source asked for, nothing is a source — including the URL
		// that will actually answer.
		plain := utterance{Word: "ciao", Session: en}
		if plain.spokeSource(AudioCandidates("ciao", en)[0]) {
			t.Error("an utterance with no source claimed one spoke")
		}
	})
}
