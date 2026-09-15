package main

import (
	"html"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

const languageBodyLimit = 16 << 10
const languageHeaderLimit = 64

type languageDecodeState uint8

const (
	decodeNeutral languageDecodeState = iota
	decodeSegment
	decodeRecovery
)

type languageDecodeEvent uint8

const (
	decodeText languageDecodeEvent = iota
	decodeOpen
	decodeClose
	decodeBadHeader
	decodeLimit
	decodeFinish
)

type languageDecodeEffect uint8

const (
	decodeNoEffect languageDecodeEffect = iota
	decodeEmitNeutral
	decodeAppend
	decodeFlushOwned
	decodeFlushNeutral
)

// stepLanguageDecode is the single owner of annotation transitions. limit in
// neutral/recovery is rejected (false), so malformed internal events cannot
// unexpectedly alter ownership.
func stepLanguageDecode(s languageDecodeState, e languageDecodeEvent) (languageDecodeState, languageDecodeEffect, bool) {
	switch e {
	case decodeText:
		if s == decodeSegment {
			return s, decodeAppend, true
		}
		return s, decodeEmitNeutral, true
	case decodeOpen:
		if s == decodeNeutral {
			return decodeSegment, decodeNoEffect, true
		}
		if s == decodeSegment {
			return decodeRecovery, decodeFlushNeutral, true
		}
		return s, decodeNoEffect, true
	case decodeClose:
		if s == decodeSegment {
			return decodeNeutral, decodeFlushOwned, true
		}
		return decodeNeutral, decodeNoEffect, true
	case decodeBadHeader:
		if s == decodeSegment {
			return decodeRecovery, decodeFlushNeutral, true
		}
		return decodeRecovery, decodeNoEffect, true
	case decodeLimit:
		if s == decodeSegment {
			return decodeRecovery, decodeFlushNeutral, true
		}
		return s, decodeNoEffect, false
	case decodeFinish:
		if s == decodeSegment {
			return decodeNeutral, decodeFlushNeutral, true
		}
		return decodeNeutral, decodeNoEffect, true
	}
	return s, decodeNoEffect, false
}

type languageDecoder struct {
	state        languageDecodeState
	body         strings.Builder
	lang         store.Lang
	marker       string
	filter       answerTextFilter
	emit         func(languageText)
	entity       string
	entityLang   store.Lang
	literal      answerTextFilter
	literalLang  store.Lang
	literalText  strings.Builder
	literalSpans []languageSpan
}

func newLanguageDecoder(emit func(languageText)) *languageDecoder {
	d := &languageDecoder{emit: emit}
	d.filter.emit = d.lex
	d.literal.emit = func(s string) {
		start := d.literalText.Len()
		d.literalText.WriteString(s)
		if d.literalLang != "" {
			n := len(d.literalSpans)
			if n > 0 && d.literalSpans[n-1].end == start && d.literalSpans[n-1].lang == d.literalLang {
				d.literalSpans[n-1].end += len(s)
			} else {
				d.literalSpans = append(d.literalSpans, languageSpan{start: start, end: start + len(s), lang: d.literalLang})
			}
		}
	}

	return d
}
func (d *languageDecoder) Write(s string) { d.filter.Write(s) }
func (d *languageDecoder) Finish() {
	d.filter.Finish()
	if d.marker != "" && !strings.HasPrefix(d.marker, "[lang") && !strings.HasPrefix(d.marker, "[/lang") {
		d.event(decodeText, d.marker, "")
	}
	d.marker = ""
	d.event(decodeFinish, "", "")
	d.flushEntity()
	d.literal.Finish()
	d.flushLiteral()
}
func (d *languageDecoder) event(e languageDecodeEvent, text string, lang store.Lang) {
	old := d.state
	next, effect, ok := stepLanguageDecode(old, e)
	if !ok {
		return
	}
	d.state = next
	switch effect {
	case decodeEmitNeutral:
		d.output(text, "")
	case decodeAppend:
		if d.body.Len()+len(text) > languageBodyLimit {
			d.event(decodeLimit, "", "")
			d.output(text, "")
		} else {
			d.body.WriteString(text)
		}
	case decodeFlushOwned:
		d.output(d.body.String(), d.lang)
		d.body.Reset()
		d.lang = ""
	case decodeFlushNeutral:
		d.output(d.body.String(), "")
		d.body.Reset()
		d.lang = ""
	}
	if e == decodeOpen && old == decodeNeutral {
		d.lang = lang
	}
}
func (d *languageDecoder) lex(s string) {
	// The upstream filter emits complete runes, so candidate/body limits never
	// split UTF-8. The marker grammar itself is ASCII.
	if d.marker == "" {
		if s == "[" {
			d.marker = s
		} else {
			d.event(decodeText, s, "")
		}
		return
	}
	d.marker += s
	m := d.marker
	reserved := strings.HasPrefix(m, "[lang") || strings.HasPrefix(m, "[/lang")
	possible := strings.HasPrefix("[lang=", m) || strings.HasPrefix("[/lang]", m)
	if m == "[/lang]" {
		d.marker = ""
		d.event(decodeClose, "", "")
		return
	}
	if reserved && strings.HasSuffix(m, "]") {
		d.marker = ""
		if strings.HasPrefix(m, "[lang=") {
			code := m[6 : len(m)-1]
			if code == "und" {
				d.event(decodeOpen, "", "")
				return
			}
			if len(code) == 2 {
				if lang, err := store.ParseLang(code); err == nil {
					d.event(decodeOpen, "", lang)
					return
				}
			}
		}
		d.event(decodeBadHeader, "", "")
		return
	}
	if reserved && len(m) >= languageHeaderLimit {
		d.marker = ""
		d.event(decodeBadHeader, "", "")
		return
	}
	if !reserved && !possible {
		d.marker = ""
		d.event(decodeText, m[:1], "")
		for _, r := range m[1:] {
			d.lex(string(r))
		}
	}
}

// Entity decoding is after marker parsing: escaped marker examples are literal.
// A second control filter prevents numeric entities from injecting controls.
func (d *languageDecoder) output(s string, lang store.Lang) {
	defer d.flushLiteral()
	if d.entity != "" && d.entityLang != lang {
		d.flushEntity()
	}
	for _, r := range s {
		if d.entity == "" {
			if r == '&' {
				d.entity = "&"
				d.entityLang = lang
			} else {
				d.literalLang = lang
				d.literal.Write(string(r))
			}
			continue
		}
		if r == '&' {
			d.flushEntity()
			d.entity = "&"
			d.entityLang = lang
			continue
		}
		d.entity += string(r)
		if r == ';' {
			v := html.UnescapeString(d.entity)
			d.entity = ""
			d.literalLang = lang
			d.literal.Write(v)
		} else if len(d.entity) >= 64 || r == '\n' || r == ' ' {
			d.flushEntity()
		}
	}
}
func (d *languageDecoder) flushEntity() {
	if d.entity != "" {
		d.literalLang = d.entityLang
		d.literal.Write(d.entity)
		d.entity = ""
	}
}

func (d *languageDecoder) flushLiteral() {
	if d.literalText.Len() > 0 {
		d.emit(languageText{text: d.literalText.String(), spans: d.literalSpans})
		d.literalText.Reset()
		d.literalSpans = nil
	}
}
