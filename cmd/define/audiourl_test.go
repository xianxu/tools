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

// Multi-word headwords exist (hot dog) and must not produce a URL with a space.
func TestAudioCandidatesMultiWord(t *testing.T) {
	for _, u := range AudioCandidates("hot dog", voice{Lang: "en", Locale: "us"}) {
		if strings.Contains(u, " ") {
			t.Errorf("unescaped space in %s", u)
		}
	}
	if got := AudioCandidates("hot dog", voice{Lang: "en", Locale: "us"})[0]; !strings.Contains(got, "hot_dog_en_us_1.mp3") {
		t.Errorf("multi-word slug = %s", got)
	}
}

// One language, and the legacy path for English only.
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
		name    string
		lang    store.Lang
		flag    string
		flagSet bool
		want    string
	}{
		{name: "English defaults to us", lang: "en", want: "us"},
		{name: "English honours the flag", lang: "en", flag: "gb", flagSet: true, want: "gb"},
		{name: "Spanish defaults to es", lang: "es", want: "es"},
		// THE change #27 makes. This was refused with a diagnostic before, and
		// es_us is a real recording — the Latin American seseo, phonemically
		// distinct from Castilian es_es, which is the whole point of the flag.
		{name: "Spanish honours the flag: seseo", lang: "es", flag: "us", flagSet: true, want: "us"},
		{name: "an unknown language is its own locale", lang: "de", want: "de"},
		{name: "and honours the flag too", lang: "fr", flag: "ca", flagSet: true, want: "ca"},
		// NOT rejected. es_gb 404s and degrades to the warning every missing
		// recording produces; the CDN is the authority on what exists, not a
		// table here.
		{name: "an unserved pair is built, not refused", lang: "es", flag: "gb", flagSet: true, want: "gb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := localeFor(tc.lang, tc.flag, tc.flagSet); got != tc.want {
				t.Errorf("localeFor(%q, %q, %v) = %q, want %q", tc.lang, tc.flag, tc.flagSet, got, tc.want)
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
	v := voiceFor("es", "", false)
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
func TestTheFetchLoopAsksOnlyForTheSessionsLanguage(t *testing.T) {
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
