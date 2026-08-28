package main

import (
	"net/url"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
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
//
// ONE language, never a search across languages: #23 makes the language a
// declared mode, so there is nothing to guess. That is what deleted #27's
// planned voices() fallback — a mode does not need one.
func AudioCandidates(word string, v voice) []string {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}
	if v.Lang == "" {
		v.Lang = store.DefaultLang
	}
	if v.Locale == "" {
		v.Locale = defaultLocale(v.Lang)
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
		out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_"+string(v.Lang)+"_"+v.Locale+"_"+n+".mp3")
	}
	// The legacy generation is ENGLISH-ONLY. Measured 2026-08-28: madrugar--_us_1
	// and madrugar--_es_1 are both 404 while sycophantic--_us_1 is 200. Asking
	// anyway would cost ~450ms per lookup for a guaranteed miss — the exact cost
	// #23's declared mode exists to stop paying.
	if v.Lang == store.DefaultLang {
		for _, n := range []string{"1", "2"} {
			out = append(out, audioBase+"/sounds/oxford/"+esc+"--_"+v.Locale+"_"+n+".mp3")
		}
	}
	return out
}
