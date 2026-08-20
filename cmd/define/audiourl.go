package main

import (
	"net/url"
	"strings"
)

// audioBase is Google's dictionary pronunciation CDN. The paths are
// undocumented but have been stable for a decade and through two migrations;
// fetch.go's conformance test is what detects a third.
const audioBase = "https://ssl.gstatic.com/dictionary/static"

// AudioCandidates returns the CDN URLs to try, in order, for a word's recorded
// pronunciation. Pure and offline — no request is made here, so the ordering is
// unit-testable without a server.
//
// The order comes from a survey across 10 words: the 2022 generation strictly
// dominates the legacy sounds/oxford paths (gaslighting exists only on 2022;
// defenestrate needs the _2 suffix on the legacy path), so one URL is not
// enough and the fallback policy belongs here rather than in the fetch loop.
func AudioCandidates(word, locale string) []string {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}
	if locale == "" {
		locale = "us"
	}
	// Multi-word headwords are spelled with underscores on the CDN.
	slug := strings.ReplaceAll(word, " ", "_")
	esc := url.PathEscape(slug)

	// The shard is the first two letters — or one, for a single-letter word.
	shard := esc
	if r := []rune(slug); len(r) >= 2 {
		shard = url.PathEscape(string(r[:2]))
	}

	var out []string
	for _, n := range []string{"1", "2"} {
		out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_en_"+locale+"_"+n+".mp3")
	}
	for _, n := range []string{"1", "2"} {
		out = append(out, audioBase+"/sounds/oxford/"+esc+"--_"+locale+"_"+n+".mp3")
	}
	return out
}
