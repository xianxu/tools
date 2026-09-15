package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	spanishEnglishDictionaryID = "com.apple.dictionary.OxfordSpanish"
	bilingualMaxRecords        = 64
	bilingualMaxBytes          = 1 << 20
	bilingualEntryNamespace    = "http://www.apple.com/DTDs/DictionaryService-1.0.rng"
)

var (
	ErrBilingualUnavailable = errors.New("Spanish–English dictionary unavailable")
	ErrBilingualUnsupported = errors.New("Spanish–English dictionary API unavailable")
	ErrBilingualMalformed   = errors.New("unsupported Spanish–English dictionary record")
	ErrBilingualLimit       = errors.New("Spanish–English dictionary result exceeds limit")
)

// bilingualRecord owns both representations of the same native record. HTML
// supplies verified direction/title; Text is the dictionary's own rendering.
type bilingualRecord struct {
	HTML string `json:"html"`
	Text string `json:"text"`
}

type recordSource interface {
	Records(word string) ([]bilingualRecord, error)
}

type bilingualIdentity struct {
	id, title string
	spanish   bool
}

// bilingualRecordIdentity accepts the dictionary's html/body/d:entry shape.
// Reading a nested span's ID would let an English entry impersonate Spanish.
// A full token walk checks truncation and duplicate entries; encoding/xml does
// not fetch external DTDs or expand external entities.
func bilingualRecordIdentity(raw string) (bilingualIdentity, error) {
	var identity bilingualIdentity
	decoder := xml.NewDecoder(strings.NewReader(raw))
	var path []xml.Name
	roots, entries := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return identity, ErrBilingualMalformed
		}
		switch token := token.(type) {
		case xml.StartElement:
			seenAttrs := map[xml.Name]bool{}
			for _, attr := range token.Attr {
				if seenAttrs[attr.Name] {
					return identity, ErrBilingualMalformed
				}
				seenAttrs[attr.Name] = true
			}
			path = append(path, token.Name)
			if len(path) > 64 {
				return identity, ErrBilingualMalformed
			}
			if len(path) == 1 {
				roots++
				if token.Name.Local != "html" || (token.Name.Space != "" && token.Name.Space != "http://www.w3.org/1999/xhtml") {
					return identity, ErrBilingualMalformed
				}
			}
			if token.Name.Local == "entry" && token.Name.Space == bilingualEntryNamespace {
				if len(path) != 3 || path[1].Local != "body" || path[1].Space != path[0].Space {
					return identity, ErrBilingualMalformed
				}
				entries++
				for _, a := range token.Attr {
					if a.Name.Local == "id" && a.Name.Space == "" {
						identity.id = a.Value
					}
					if a.Name.Local == "title" && a.Name.Space == bilingualEntryNamespace {
						identity.title = a.Value
					}
				}
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		case xml.CharData:
			if len(path) == 0 && strings.TrimSpace(string(token)) != "" {
				return identity, ErrBilingualMalformed
			}
		}
	}
	if roots != 1 || entries != 1 || strings.TrimSpace(identity.title) == "" {
		return identity, ErrBilingualMalformed
	}
	var suffix string
	if strings.HasPrefix(identity.id, "s_b-es-en") {
		identity.spanish = true
		suffix = strings.TrimPrefix(identity.id, "s_b-es-en")
	} else if strings.HasPrefix(identity.id, "e_b-en-es") {
		suffix = strings.TrimPrefix(identity.id, "e_b-en-es")
	} else {
		return identity, ErrBilingualMalformed
	}
	if suffix == "" || strings.IndexFunc(suffix, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return identity, ErrBilingualMalformed
	}
	return identity, nil
}

// spanishTitleFold is deliberately stricter than audio's
// differsOnlyByDiacritics candidate heuristic: that helper admits any non-ASCII
// substitution (even a→é), which cannot establish dictionary word identity.
// Fold only Spanish orthographic marks, including their decomposed forms.
func spanishTitleFold(s string) string {
	var out strings.Builder
	var previous rune
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á':
			r = 'a'
		case 'é':
			r = 'e'
		case 'í':
			r = 'i'
		case 'ó':
			r = 'o'
		case 'ú', 'ü':
			r = 'u'
		case 'ñ':
			r = 'n'
		case '\u0301':
			if strings.ContainsRune("aeiou", previous) {
				continue
			}
		case '\u0303':
			if previous == 'n' {
				continue
			}
		case '\u0308':
			if previous == 'u' {
				continue
			}
		}
		out.WriteRune(r)
		previous = r
	}
	return out.String()
}

// selectSpanishRecords keeps all homographs at the best verified title match.
// Exact query spelling precedes accent equivalents, then the primary entry's
// canonical headword. It never guesses an inflection from unrelated candidates.
func selectSpanishRecords(records []bilingualRecord, word, canonical string) ([]string, error) {
	if len(records) > bilingualMaxRecords || len(word) > bilingualMaxBytes || len(canonical) > bilingualMaxBytes {
		return nil, ErrBilingualLimit
	}
	type selected struct {
		id, text string
		rank     int
	}
	matches := map[string]selected{}
	best := 5
	malformed := false
	foldedWord, foldedCanonical := spanishTitleFold(word), spanishTitleFold(canonical)
	for _, r := range records {
		if len(r.HTML) > bilingualMaxBytes || len(r.Text) > bilingualMaxBytes {
			return nil, ErrBilingualLimit
		}
		identity, err := bilingualRecordIdentity(r.HTML)
		if err != nil || strings.TrimSpace(r.Text) == "" || !utf8.ValidString(r.Text) {
			malformed = true
			continue
		}
		if !identity.spanish {
			continue
		}
		rank := 5
		switch {
		case strings.EqualFold(identity.title, word):
			rank = 0
		case spanishTitleFold(identity.title) == foldedWord:
			rank = 1
		case canonical != "" && strings.EqualFold(identity.title, canonical):
			rank = 2
		case canonical != "" && spanishTitleFold(identity.title) == foldedCanonical:
			rank = 3
		}
		if rank == 5 {
			continue
		}
		best = min(best, rank)
		old, exists := matches[identity.id]
		if !exists || rank < old.rank || rank == old.rank && r.Text < old.text {
			matches[identity.id] = selected{identity.id, r.Text, rank}
		}
	}
	var ordered []selected
	for _, m := range matches {
		if m.rank == best {
			ordered = append(ordered, m)
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].id < ordered[j].id })
	if len(ordered) == 0 {
		if malformed {
			return nil, ErrBilingualMalformed
		}
		return nil, fmt.Errorf("Spanish–English %q: %w", word, ErrNoEntry)
	}
	out := make([]string, len(ordered))
	for i, m := range ordered {
		out[i] = m.text
	}
	return out, nil
}
