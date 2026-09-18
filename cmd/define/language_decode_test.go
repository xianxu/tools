package main

import (
	"reflect"
	"strings"
	"testing"
)

// spanAccumulator joins a decoder's emissions into one languageText, merging
// adjacent runs of the same language.
//
// Merging is not cosmetic since #72: a passage is emitted as it arrives, one span
// per rune, so "the longest span" is 3 bytes for every answer ever written unless
// adjacent runs are joined back up. The conformance recorder learned that the
// expensive way, declaring a perfectly good Spanish answer fragmented.
//
// ONE copy, because there were three — this, decodedChunksBounded's clone, and a
// third open-coded in the conformance recorder (ARCH-DRY).
func spanAccumulator(out *languageText) func(languageText) {
	return func(v languageText) {
		n := len(out.text)
		out.text += v.text
		for _, sp := range v.spans {
			sp.start += n
			sp.end += n
			if last := len(out.spans) - 1; last >= 0 && out.spans[last].end == sp.start && out.spans[last].lang == sp.lang {
				out.spans[last].end = sp.end
				continue
			}
			out.spans = append(out.spans, sp)
		}
	}
}

// decodeChunks runs a string through the decoder in two writes, calling after()
// at every point the decoder is between events — which is where any claim about
// what it RETAINS has to hold.
func decodeChunks(s string, split int, after func(*languageDecoder)) languageText {
	var out languageText
	d := newLanguageDecoder(spanAccumulator(&out))
	d.Write(s[:split])
	after(d)
	d.Write(s[split:])
	after(d)
	d.Finish()
	after(d)
	return out
}

func decodedChunks(s string, split int) languageText {
	return decodeChunks(s, split, func(*languageDecoder) {})
}

// longestPassage reports the longest single-language run and the total decoded
// length — the ONE definition of "a dominant passage", shared by the guard that
// promotes a capture and the guard that replays it, so they cannot disagree
// about the same file. Both measure DECODED text; the marker bytes a raw capture
// carries are not text anyone sees.
func longestPassage(v languageText) (longest, total int) {
	for _, sp := range v.spans {
		if n := sp.end - sp.start; n > longest {
			longest = n
		}
	}
	return longest, len(v.text)
}

// dominantPassage is that rule's threshold, stated once.
func dominantPassage(v languageText) bool {
	longest, total := longestPassage(v)
	return total >= 500 && longest*10 >= total*6
}
func TestLanguageDecoderRecoveryAndSplits(t *testing.T) {
	for _, tt := range []struct {
		in, want string
		owned    string
	}{
		{"hello [lang=es]¡hola![/lang] world", "hello ¡hola! world", "¡hola!"},
		// Ownership is decided at the OPEN since #72, so text that reached the
		// reader before a passage went wrong keeps the language it was announced
		// with — `red` below, and the whole of `unfinished`. Only text arriving
		// AFTER the malformed event is denied ownership, which is what keeps
		// `son` neutral in both the nested and the bad-header row.
		{"[lang=es]red[lang=en]son[/lang] [lang=es]mesa[/lang][/lang]", "redson mesa", "redmesa"},
		{"[lang=es]unfinished[/lang", "unfinished", "unfinished"},
		{"[lang=bad]red[lang=es]son[/lang][lang=es]mesa[/lang]", "redsonmesa", "mesa"},
		{"literal [bracket] &#91;lang=es&#93;plain&#91;/lang&#93;", "literal [bracket] [lang=es]plain[/lang]", ""},
		{"[lang=und]maybe[/lang]", "maybe", ""},
		// Formerly the over-limit row: 16385 bytes exceeded the 16 KiB body bound
		// and the passage was abandoned to neutral. Nothing accumulates now, so
		// there is no bound to exceed and a long passage is simply a long
		// passage — every byte of it owned, and every byte already on screen.
		{"[lang=es]" + strings.Repeat("a", 16385) + "[/lang][lang=es]sí[/lang]", strings.Repeat("a", 16385) + "sí", strings.Repeat("a", 16385) + "sí"},
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

// decodedChunksBounded is decodedChunks with the retention invariant asserted
// after every write.
//
// The invariant lives here, in the fuzz path, because that is where it can be
// wrong: the bound is a property of the marker and entity GRAMMARS, and a
// hand-written case only ever exercises the fragments its author thought of. It
// replaced `d.body.Len() < languageBodyLimit`, which named a field — and so
// stopped checking anything the moment #72 deleted that field, rather than
// failing.
func decodedChunksBounded(t *testing.T, s string, split int) languageText {
	t.Helper()
	return decodeChunks(s, split, func(d *languageDecoder) {
		if got := d.retained(); got > maxLanguageDecoderRetained {
			t.Fatalf("decoder retained %d bytes, bound %d", got, maxLanguageDecoderRetained)
		}
	})
}

func FuzzLanguageDecoderChunks(f *testing.F) {
	f.Add("[lang=es]café[/lang]", uint16(5))
	// Own-at-open forms (#72): a passage that never closes, a nested one, a long
	// one that would once have tripped the 16 KiB body bound, and an entity
	// inside a passage — the four shapes whose ownership or retention the change
	// altered.
	f.Add("[lang=es]sin cerrar", uint16(3))
	f.Add("[lang=es]uno[lang=en]dos[/lang]", uint16(11))
	f.Add("[lang=es]"+strings.Repeat("a", 20000)+"[/lang]", uint16(9))
	f.Add("[lang=es]&#91;no&#93;[/lang]", uint16(13))
	f.Fuzz(func(t *testing.T, s string, n uint16) {
		if len(s) > 65536 {
			return
		}
		a := decodedChunksBounded(t, s, 0)
		b := decodedChunksBounded(t, s, int(n)%(len(s)+1))
		if a.text != b.text || !reflect.DeepEqual(a.spans, b.spans) {
			t.Fatal("chunk-dependent decode")
		}
	})
}

func TestLanguageDecodeTransitions(t *testing.T) {
	// Independent table, stated rather than read off the implementation — it is
	// the oracle, so deriving it from the code under test would assert nothing.
	// Columns: text, open, close, badHeader, finish.
	//
	// The function is TOTAL since #72: decodeLimit was the only event a state
	// could refuse, and it had one producer — the append arm that no longer
	// exists. Segment/text is the cell this issue turned over, from "accumulate"
	// to "emit, owned by the announced language".
	next := [3][5]languageDecodeState{
		{decodeNeutral, decodeSegment, decodeNeutral, decodeRecovery, decodeNeutral},
		{decodeSegment, decodeRecovery, decodeNeutral, decodeRecovery, decodeNeutral},
		{decodeRecovery, decodeRecovery, decodeNeutral, decodeRecovery, decodeNeutral},
	}
	effects := [3][5]languageDecodeEffect{
		{decodeEmitNeutral, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect},
		{decodeEmitOwned, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect},
		{decodeEmitNeutral, decodeNoEffect, decodeNoEffect, decodeNoEffect, decodeNoEffect},
	}
	for state := decodeNeutral; state <= decodeRecovery; state++ {
		for event := decodeText; event <= decodeFinish; event++ {
			n, e := stepLanguageDecode(state, event)
			if n != next[state][event] || e != effects[state][event] {
				t.Fatalf("state %d event %d: %d %d", state, event, n, e)
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
		if got := d.retained(); got > maxLanguageDecoderRetained {
			t.Fatalf("decoder retained %d bytes, bound %d", got, maxLanguageDecoderRetained)
		}
	}
	// A quarter-megabyte passage no longer overflows into recovery, because
	// nothing accumulates to overflow (#72). It stays an open, owned passage —
	// and every byte of it has already reached the sink.
	if d.state != decodeSegment {
		t.Fatalf("state %d: a long passage stays a passage now that nothing buffers", d.state)
	}
	d.Write("[/lang]")
	if d.state != decodeNeutral {
		t.Fatal("close did not end the passage")
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

// TestOwnedTextIsEmittedBeforeItsCloseMarker is the observable every earlier
// streaming test lacked (#72).
//
// They all assert the answer's final CONTENT, and a decoder that buffers a whole
// passage produces byte-identical final content — which is how a ten-second
// blank screen passed a green suite. What has to be asserted is WHEN text
// reaches the sink, relative to the close marker that used to release it.
//
// Measured before the fix: 0 emissions until `[/lang]`, on a stream whose deltas
// arrived 60 ms apart for ten seconds.
func TestOwnedTextIsEmittedBeforeItsCloseMarker(t *testing.T) {
	var emitted []languageText
	d := newLanguageDecoder(func(v languageText) { emitted = append(emitted, v) })

	d.Write("[lang=es]")
	d.Write("hola ")
	d.Write("mundo")
	if len(emitted) == 0 {
		t.Fatal("nothing reached the sink while the passage was still open")
	}

	// And it arrives OWNED, not as neutral text later relabelled: ownership is
	// decided by the opening marker, which is the whole of the change.
	var text strings.Builder
	var owned strings.Builder
	for _, v := range emitted {
		text.WriteString(v.text)
		for _, sp := range v.spans {
			if sp.lang != "es" {
				t.Fatalf("emitted span owned by %q, want es", sp.lang)
			}
			owned.WriteString(v.text[sp.start:sp.end])
		}
	}
	if text.String() != "hola mundo" || owned.String() != "hola mundo" {
		t.Fatalf("emitted %q (owned %q) want all of \"hola mundo\" owned", text.String(), owned.String())
	}
}
