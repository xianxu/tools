package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/store"
)

// bilingualLanguageText reuses the structural parser's verified ownership walk.
func bilingualLanguageText(record bilingualRecord) languageText {
	doc, err := parseBilingualDocument(record)
	if err != nil {
		return languageText{text: record.Text}
	}
	return doc.native
}

// projectDictionaryText permits only whitespace normalization and ANSI styling.
// It consumes both streams from their current positions, never searching ahead
// for words. Any other transformation makes the whole fragment neutral.
func projectDictionaryText(source languageText, rendered string) languageText {
	return projectLanguageText(source, rendered, false)
}

// Display layout may insert a physical newline inside an overlong source word.
// Native dictionary correspondence remains stricter: only the owned layout
// transform opts into generated whitespace; glyph matching stays sequential.
func projectDisplayText(source languageText, rendered string) languageText {
	return projectLanguageText(source, rendered, true)
}
func projectLanguageText(source languageText, rendered string, displayWhitespace bool) languageText {
	result := languageText{text: rendered}
	i, j, spanIndex := 0, 0, 0
	inSpace := false
	owner := func(at int) store.Lang {
		for spanIndex < len(source.spans) && source.spans[spanIndex].end <= at {
			spanIndex++
		}
		if spanIndex < len(source.spans) && source.spans[spanIndex].start <= at {
			return source.spans[spanIndex].lang
		}
		return ""
	}
	appendOwned := func(start, end int, lang store.Lang) {
		if lang == "" {
			return
		}
		n := len(result.spans)
		if n > 0 && result.spans[n-1].end == start && result.spans[n-1].lang == lang {
			result.spans[n-1].end = end
		} else {
			result.spans = append(result.spans, languageSpan{start: start, end: end, lang: lang})
		}
	}
	skipSourceStyle := func() {
		for i < len(source.text) {
			n := escapeLen(source.text[i:])
			if n == 0 {
				break
			}
			i += n
		}
	}
	for j < len(rendered) {
		skipSourceStyle()
		if n := escapeLen(rendered[j:]); n > 0 {
			j += n
			continue
		}
		r, n := utf8.DecodeRuneInString(rendered[j:])
		if unicode.IsSpace(r) {
			if !inSpace {
				begin := i
				for i < len(source.text) {
					skipSourceStyle()
					if i == len(source.text) {
						break
					}
					q, m := utf8.DecodeRuneInString(source.text[i:])
					if !unicode.IsSpace(q) {
						break
					}
					i += m
				}
				if begin == i && !displayWhitespace {
					return languageText{text: rendered}
				}
				if begin < i {
					appendOwned(j, j+n, owner(begin))
				}
			}
			// Continuation indentation is generated, so only the first byte run
			// consumes source whitespace and receives its ownership.
			inSpace = true
			j += n
			continue
		}
		inSpace = false
		if i >= len(source.text) || !strings.HasPrefix(source.text[i:], rendered[j:j+n]) {
			return languageText{text: rendered}
		}
		appendOwned(j, j+n, owner(i))
		i += n
		j += n
	}
	skipSourceStyle()
	if strings.TrimSpace(stripEscapes(source.text[i:])) != "" {
		return languageText{text: rendered}
	}
	return result
}
