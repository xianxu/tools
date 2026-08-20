package main

import (
	"slices"
	"strings"
	"testing"
)

func TestAudioCandidatesOrder(t *testing.T) {
	got := AudioCandidates("Sycophantic", "us") // input case must not matter
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
	for _, u := range AudioCandidates("sycophantic", "gb") {
		if !strings.Contains(u, "_gb_") && !strings.Contains(u, "--_gb_") {
			t.Errorf("locale not applied: %s", u)
		}
	}
	// An empty locale defaults to us rather than producing "_en__1".
	for _, u := range AudioCandidates("sycophantic", "") {
		if strings.Contains(u, "__") {
			t.Errorf("empty locale leaked into %s", u)
		}
	}
}

// A one-letter word must not panic on the two-letter shard prefix.
func TestAudioCandidatesShortWord(t *testing.T) {
	got := AudioCandidates("a", "us")
	if len(got) == 0 {
		t.Fatal("no candidates for a single-letter word")
	}
	if !strings.Contains(got[0], "/audio/a/a_en_us_1.mp3") {
		t.Errorf("unexpected shard for a single-letter word: %s", got[0])
	}
	if AudioCandidates("", "us") != nil {
		t.Error("empty word should yield no candidates")
	}
}

// Multi-word headwords exist (hot dog) and must not produce a URL with a space.
func TestAudioCandidatesMultiWord(t *testing.T) {
	for _, u := range AudioCandidates("hot dog", "us") {
		if strings.Contains(u, " ") {
			t.Errorf("unescaped space in %s", u)
		}
	}
	if got := AudioCandidates("hot dog", "us")[0]; !strings.Contains(got, "hot_dog_en_us_1.mp3") {
		t.Errorf("multi-word slug = %s", got)
	}
}
