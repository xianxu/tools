package main

import (
	"encoding/xml"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/store"
)

// bilingualLanguageText trusts only the selected Spanish-to-English record's
// language-bearing classes. Unknown wrappers inherit an established ancestor;
// pronunciation, generated punctuation, and editorial metadata remain neutral.
func bilingualLanguageText(record bilingualRecord) languageText {
	neutral := languageText{text: record.Text}
	if len(record.HTML) > bilingualMaxBytes || len(record.Text) > bilingualMaxBytes {
		return neutral
	}
	id, err := bilingualRecordIdentity(record.HTML)
	if err != nil || !id.spanish {
		return neutral
	}
	decoder := xml.NewDecoder(strings.NewReader(record.HTML))
	var stack []store.Lang
	var out strings.Builder
	var spans []languageSpan
	entryDepth := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return neutral
		}
		switch token := token.(type) {
		case xml.StartElement:
			lang := store.Lang("")
			if len(stack) > 0 {
				lang = stack[len(stack)-1]
			}
			if token.Name.Local == "entry" && token.Name.Space == bilingualEntryNamespace {
				entryDepth = len(stack) + 1
			}
			for _, attr := range token.Attr {
				if attr.Name.Local != "class" {
					continue
				}
				for _, class := range strings.Fields(attr.Value) {
					switch class {
					case "hw", "ex", "idm", "ind":
						lang = "es"
					case "trans":
						lang = "en"
					case "gp", "ph", "prx", "lg", "reg", "lev", "fld", "tgr", "ps", "sn", "underline":
						lang = ""
					}
				}
			}
			stack = append(stack, lang)
		case xml.EndElement:
			if len(stack) == entryDepth {
				entryDepth = 0
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if entryDepth == 0 {
				continue
			}
			start := out.Len()
			out.Write(token)
			if lang := stack[len(stack)-1]; lang != "" {
				spans = append(spans, languageSpan{start: start, end: out.Len(), lang: lang})
			}
		}
	}
	return projectDictionaryText(languageText{text: out.String(), spans: spans}, record.Text)
}

// projectDictionaryText permits only whitespace normalization and ANSI styling.
// It consumes both streams from their current positions, never searching ahead
// for words. Any other transformation makes the whole fragment neutral.
func projectDictionaryText(source languageText, rendered string) languageText {
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
	for j < len(rendered) {
		if n := escapeLen(rendered[j:]); n > 0 {
			j += n
			continue
		}
		r, n := utf8.DecodeRuneInString(rendered[j:])
		if unicode.IsSpace(r) {
			if !inSpace {
				begin := i
				for i < len(source.text) {
					q, m := utf8.DecodeRuneInString(source.text[i:])
					if !unicode.IsSpace(q) {
						break
					}
					i += m
				}
				if begin == i {
					return languageText{text: rendered}
				}
				appendOwned(j, j+n, owner(begin))
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
	if strings.TrimSpace(source.text[i:]) != "" {
		return languageText{text: rendered}
	}
	return result
}

func dictionaryFragment(source languageText, at int, text string) languageText {
	out := languageText{text: text}
	if at < 0 || at > len(source.text) || len(text) > len(source.text)-at || source.text[at:at+len(text)] != text {
		return out
	}
	end := at + len(text)
	first := sort.Search(len(source.spans), func(i int) bool { return source.spans[i].end > at })
	for _, span := range source.spans[first:] {
		if span.start >= end {
			break
		}
		out.spans = append(out.spans, languageSpan{start: max(span.start, at) - at, end: min(span.end, end) - at, lang: span.lang})
	}
	return out
}

func (o RenderOpts) dictionaryText(e Entry, original, rendered string, at int, known bool) string {
	if !o.Color || !known && at < 0 {
		return rendered
	}
	source := languageText{text: original}
	if e.source.text != "" {
		if known {
			source = dictionaryFragment(e.source, at, original)
		}
	} else if o.Language != "" {
		source.spans = []languageSpan{{start: 0, end: len(original), lang: o.Language}}
	}
	return styleLanguageText(projectDictionaryText(source, rendered), o.Tint)
}
