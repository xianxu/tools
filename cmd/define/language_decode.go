package main

import (
	"html"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/store"
)

const languageHeaderLimit = 64

// maxLanguageDecoderRetained bounds everything this decoder holds BETWEEN
// events, and exists because #72 deleted the segment body — the one quantity
// anything used to assert a bound on.
//
// Its components: a marker candidate and an entity candidate, each stopped by
// its own grammar at 64 bytes, plus a partial rune in each of the two control
// filters (`filter` lexes, `literal` renders). `literalText` is reset by
// `output`'s deferred flush, so it holds nothing once an event returns.
//
// Stated as a TOTAL rather than per field: `d.body.Len() < languageBodyLimit`
// named one field, so it became uncheckable the moment that field went away
// instead of failing. A total survives the next component being added.
const maxLanguageDecoderRetained = 2*languageHeaderLimit + 2*utf8.UTFMax

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
	decodeFinish
)

type languageDecodeEffect uint8

const (
	decodeNoEffect languageDecodeEffect = iota
	decodeEmitNeutral
	decodeEmitOwned
)

// stepLanguageDecode is the single owner of annotation transitions.
//
// OWNERSHIP IS DECIDED WHEN A PASSAGE OPENS (#72). Text inside a segment is
// emitted as it arrives, owned by the language the opening marker announced,
// rather than accumulated and released at the close. Holding it is what made an
// answer invisible for the whole of its generation: measured, the entire
// 1422-byte answer reached the screen in one write at 9.4 s, on a stream whose
// deltas had been arriving 60 ms apart since 1.25 s.
//
// Three things go with the buffer — the body, its 16 KiB bound, and the
// decodeLimit event that bound produced — and this function becomes TOTAL, since
// the only rejected pair was a limit event in a state that could not produce one.
//
// What it costs, stated so it is not mistaken for an oversight: a segment that
// nests, carries a bad header or never closes keeps the language it announced
// instead of degrading to neutral. Painted text cannot be un-painted, so the
// revocation the buffer bought was only ever available by withholding every
// well-formed answer as well. Recovery still denies ownership to text arriving
// AFTER the malformed event; only what was already emitted keeps its language.
func stepLanguageDecode(s languageDecodeState, e languageDecodeEvent) (languageDecodeState, languageDecodeEffect) {
	switch e {
	case decodeText:
		if s == decodeSegment {
			return s, decodeEmitOwned
		}
		return s, decodeEmitNeutral
	case decodeOpen:
		if s == decodeNeutral {
			return decodeSegment, decodeNoEffect
		}
		if s == decodeSegment {
			return decodeRecovery, decodeNoEffect // nested: neither passage is trusted from here
		}
		return s, decodeNoEffect
	case decodeClose:
		return decodeNeutral, decodeNoEffect
	case decodeBadHeader:
		return decodeRecovery, decodeNoEffect
	case decodeFinish:
		return decodeNeutral, decodeNoEffect
	}
	return s, decodeNoEffect
}

type languageDecoder struct {
	state        languageDecodeState
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
	next, effect := stepLanguageDecode(old, e)
	d.state = next
	// ONE clearing rule for d.lang: it belongs to the segment being decoded, so
	// it is dropped on every transition that LEAVES one. It used to be cleared
	// inside the two flush effects — two sites, and the four ways out of a
	// segment (close, nested open, bad header, finish) reached them unevenly.
	if old == decodeSegment && next != decodeSegment {
		d.lang = ""
	}
	switch effect {
	case decodeEmitNeutral:
		d.output(text, "")
	case decodeEmitOwned:
		d.output(text, d.lang)
	}
	if e == decodeOpen && old == decodeNeutral {
		d.lang = lang
	}
}

func (d *languageDecoder) lex(s string) {
	// The upstream filter emits complete runes, so the marker candidate's limit
	// never splits UTF-8. The marker grammar itself is ASCII.
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

// retained reports the bytes held between events — the envelope's oracle.
func (d *languageDecoder) retained() int {
	return len(d.marker) + len(d.entity) + len(d.filter.pending) + len(d.literal.pending) + d.literalText.Len()
}
