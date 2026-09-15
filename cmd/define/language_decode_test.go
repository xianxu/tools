package main

import (
	"reflect"
	"strings"
	"testing"
)

func decodedChunks(s string, split int) languageText {
	var out languageText
	emit := func(v languageText) {
		n := len(out.text)
		out.text += v.text
		for _, sp := range v.spans {
			sp.start += n
			sp.end += n
			out.spans = append(out.spans, sp)
		}
	}
	d := newLanguageDecoder(emit)
	d.Write(s[:split])
	d.Write(s[split:])
	d.Finish()
	return out
}
func TestLanguageDecoderRecoveryAndSplits(t *testing.T) {
	for _, tt := range []struct {
		in, want string
		owned    string
	}{
		{"hello [lang=es]¡hola![/lang] world", "hello ¡hola! world", "¡hola!"},
		{"[lang=es]red[lang=en]son[/lang] [lang=es]mesa[/lang][/lang]", "redson mesa", "mesa"},
		{"[lang=es]unfinished[/lang", "unfinished", ""},
		{"[lang=bad]red[lang=es]son[/lang][lang=es]mesa[/lang]", "redsonmesa", "mesa"},
		{"literal [bracket] &#91;lang=es&#93;plain&#91;/lang&#93;", "literal [bracket] [lang=es]plain[/lang]", ""},
		{"[lang=und]maybe[/lang]", "maybe", ""},
		{"[lang=es]" + strings.Repeat("a", 16385) + "[/lang][lang=es]sí[/lang]", strings.Repeat("a", 16385) + "sí", "sí"},
	} {
		first := decodedChunks(tt.in, 0)
		if first.text != tt.want {
			t.Fatalf("%q => %q want %q", tt.in, first.text, tt.want)
		}
		var owned strings.Builder
		for _, sp := range first.spans {
			owned.WriteString(first.text[sp.start:sp.end])
		}
		if owned.String() != tt.owned {
			t.Fatalf("owned %q want %q", owned.String(), tt.owned)
		}
		for i := 0; i <= len(tt.in); i++ {
			if len(tt.in) > 1000 && i > 70 && i < len(tt.in)-70 && i%1024 != 0 {
				continue
			}
			got := decodedChunks(tt.in, i)
			if got.text != first.text || !reflect.DeepEqual(got.spans, first.spans) {
				t.Fatalf("split %d differs for %q", i, tt.in)
			}
		}
	}
}
func TestAnswerTextFilterControls(t *testing.T) {
	for _, in := range []string{"a\x1b[31mb\x1b[0mc", "a\x1b]52;c;secret\ab\x1bPpayload\x1b\\c", "a\u009dsecret\u009cb\u0090secret\u009cc", "a\x1bXsecret\x1b\\b\x1b^secret\x1b\\c", "a\x1b_secret\x1b\\bc", "abc\x1b]unterminated"} {
		for i := range len(in) + 1 {
			var b strings.Builder
			f := answerTextFilter{emit: func(s string) { b.WriteString(s) }}
			f.Write(in[:i])
			f.Write(in[i:])
			f.Finish()
			if b.String() != "abc" {
				t.Fatalf("split %d %q => %q", i, in, b.String())
			}
		}
	}
}
func FuzzLanguageDecoderChunks(f *testing.F) {
	f.Add("[lang=es]café[/lang]", uint16(5))
	f.Fuzz(func(t *testing.T, s string, n uint16) {
		if len(s) > 65536 {
			return
		}
		a := decodedChunks(s, 0)
		b := decodedChunks(s, int(n)%(len(s)+1))
		if a.text != b.text || !reflect.DeepEqual(a.spans, b.spans) {
			t.Fatal("chunk-dependent decode")
		}
	})
}

func TestLanguageDecodeTransitions(t *testing.T) {
	// Independent table pins every state/event pair, including illegal limits.
	next := [3][7]languageDecodeState{{0, 1, 0, 2, 0, 0, 0}, {1, 2, 0, 2, 2, 0, 0}, {2, 2, 0, 2, 2, 0, 0}}
	effects := [3][6]languageDecodeEffect{{decodeEmitNeutral, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect}, {decodeAppend, decodeFlushNeutral, decodeFlushOwned, decodeFlushNeutral, decodeFlushNeutral, decodeFlushNeutral}, {decodeEmitNeutral, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect}}
	for state := decodeNeutral; state <= decodeRecovery; state++ {
		for event := decodeText; event <= decodeFinish; event++ {
			n, e, ok := stepLanguageDecode(state, event)
			wantOK := event != decodeLimit || state == decodeSegment
			if n != next[state][event] || e != effects[state][event] || ok != wantOK {
				t.Fatalf("state %d event %d: %d %d %v", state, event, n, e, ok)
			}
		}
	}
}
func TestLanguageDecoderControlEntitiesAndBounds(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"a&#27;]52;c;secret&#7;bc", "abc"},
		{"[lang=es]a\x1b]secret\ab[/lang]", "ab"},
		{"a\xffb\xc3", "a�b�"},
		{"[lang=" + strings.Repeat("x", 58) + "tail[/lang]", "tail"},
		{"[lang=es]hola[/la", "hola[/la"},
	} {
		for i := range len(tt.in) + 1 {
			if got := decodedChunks(tt.in, i); got.text != tt.want {
				t.Fatalf("split %d %q => %q want %q", i, tt.in, got.text, tt.want)
			}
		}
	}
}

func TestAnswerControlPayloadAndAnnotationMemoryAreBounded(t *testing.T) {
	d := newLanguageDecoder(func(languageText) {})
	d.Write("[lang=es]")
	for range 1000 {
		d.Write(strings.Repeat("x", 256))
		if d.body.Len() > languageBodyLimit || len(d.marker) > languageHeaderLimit {
			t.Fatal("annotation buffer grew past bound")
		}
	}
	if d.state != decodeRecovery || d.body.Len() != 0 {
		t.Fatal("overflow failed to stream neutrally")
	}
	d.Write("[/lang]")
	if d.state != decodeNeutral {
		t.Fatal("first close did not recover")
	}
	var out strings.Builder
	f := answerTextFilter{emit: func(s string) { out.WriteString(s) }}
	f.Write("\x1b]52;")
	for range 1000 {
		f.Write(strings.Repeat("secret", 256))
		if len(f.pending) > 3 {
			t.Fatal("control payload retained")
		}
	}
	f.Write("\x1b\\visible")
	f.Finish()
	if out.String() != "visible" {
		t.Fatalf("control payload escaped: %q", out.String())
	}
}
