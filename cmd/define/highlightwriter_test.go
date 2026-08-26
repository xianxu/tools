package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// writeChunks feeds the writer one chunk at a time, then flushes.
func writeChunks(t *testing.T, v Vocabulary, chunks ...string) string {
	t.Helper()
	var buf bytes.Buffer
	w := newHighlightWriter(&buf, v, knownOn)
	for _, c := range chunks {
		n, err := w.Write([]byte(c))
		if err != nil {
			t.Fatalf("Write(%q) = %v", c, err)
		}
		if n != len(c) {
			t.Fatalf("Write(%q) consumed %d of %d bytes", c, n, len(c))
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() = %v", err)
	}
	return buf.String()
}

// Byte-exact, per contract rule 3: escape-stripped comparison cannot see a
// reorder, which is the failure mode rule 1 exists to prevent.
func TestHighlightWriter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		v      Vocabulary
		chunks []string
		want   string
	}{
		{
			"a known word in one call",
			vocab("obsequious"),
			[]string{"his obsequious day"},
			"his \x1b[1;32mobsequious\x1b[0m day",
		},
		{
			// The whole reason this is a writer rather than a string function.
			"a known word split across two writes",
			vocab("obsequious"),
			[]string{"his obseq", "uious day"},
			"his \x1b[1;32mobsequious\x1b[0m day",
		},
		{
			"nothing known passes through unchanged",
			vocab("obsequious"),
			[]string{"his manner today"},
			"his manner today",
		},
		{
			// ANSI does not nest: the enclosing style must come back.
			"a known word inside a styled run resumes that style",
			vocab("obsequious"),
			[]string{"\x1b[3;32mhis obsequious day\x1b[0m"},
			"\x1b[3;32mhis \x1b[1;32mobsequious\x1b[0m\x1b[3;32m day\x1b[0m",
		},
		{
			"an escape split across two writes passes through intact",
			vocab("obsequious"),
			[]string{"\x1b[3;", "32mobsequious\x1b[0m"},
			"\x1b[3;32m\x1b[1;32mobsequious\x1b[0m\x1b[3;32m\x1b[0m",
		},
		{
			// Contract rule 1: nothing overtakes held text. Holding "hot" while
			// waiting to see whether "dog" follows, then passing the reset
			// straight through, would put the reset BEFORE the word it closes.
			"an escape during a hold does not move ahead of it",
			vocab("hot dog"),
			[]string{"\x1b[1;36mhot\x1b[0m dog"},
			"\x1b[1;36mhot\x1b[0m dog",
		},
		{
			"a phrase spanning a space is highlighted whole",
			vocab("hot dog"),
			[]string{"one hot dog please"},
			"one \x1b[1;32mhot dog\x1b[0m please",
		},
		{
			"a phrase split mid-word across writes",
			vocab("hot dog"),
			[]string{"one hot d", "og please"},
			"one \x1b[1;32mhot dog\x1b[0m please",
		},
		{
			// Contract rule 2: a phrase may not span a line break.
			"a phrase does not span a newline",
			vocab("hot dog"),
			[]string{"hot\n  dog"},
			"hot\n  dog",
		},
		{
			"a word at the very end still highlights after flush",
			vocab("obsequious"),
			[]string{"truly obsequious"},
			"truly \x1b[1;32mobsequious\x1b[0m",
		},
		{
			"empty input",
			vocab("obsequious"),
			[]string{""},
			"",
		},
		{
			"a quoted word still matches",
			vocab("obsequious"),
			[]string{"the word 'obsequious' here"},
			"the word '\x1b[1;32mobsequious\x1b[0m' here",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := writeChunks(t, tc.v, tc.chunks...); got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

// Contract rule 4. crlfWriter — which this will wrap — short-writes and is
// tested for it, and byte loss is this feature's worst failure mode.
func TestHighlightWriterPropagatesDownstreamErrors(t *testing.T) {
	boom := errors.New("pipe closed")
	var written bytes.Buffer
	w := newHighlightWriter(&failAfter{out: &written, after: 3, err: boom}, vocab("obsequious"), knownOn)

	_, err1 := w.Write([]byte("his obsequious day"))
	errF := w.Flush()

	if err1 == nil && errF == nil {
		t.Fatal("a downstream failure never reached the caller")
	}
	// Poisoned: a later Write must not emit anything more, so nothing can be
	// written twice. That is WHY (0, err) is the right answer here while
	// crlfWriter — which can be written to again — returns caller-unit progress.
	// The two writers answer the same question differently on purpose.
	before := written.Len()
	if n, err := w.Write([]byte("more")); err == nil || n != 0 {
		t.Errorf("Write after failure = (%d, %v), want (0, err)", n, err)
	}
	if written.Len() != before {
		t.Error("the writer kept emitting after a downstream failure")
	}
}

// A short write with a nil error is not a success; treating it as one loses
// bytes silently.
func TestHighlightWriterTreatsAShortWriteAsAnError(t *testing.T) {
	// crlf_test.go's fixture, which is the writer this one will actually wrap.
	w := newHighlightWriter(&shortWriter{limit: 2}, vocab("obsequious"), knownOn)

	_, err := w.Write([]byte("his obsequious day"))
	if err == nil {
		err = w.Flush()
	}
	if !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("err = %v, want io.ErrShortWrite", err)
	}
}

type failAfter struct {
	out   *bytes.Buffer
	n     int
	after int
	err   error
}

func (f *failAfter) Write(p []byte) (int, error) {
	if f.n >= f.after {
		return 0, f.err
	}
	f.n++
	return f.out.Write(p)
}

// fuzzDeck is the vocabulary the writer property runs against.
//
// Derived from TestWordRuns' table — the package's own enumeration of word
// character classes — rather than hand-picked. The first version pinned
// vocab("obsequious", "hot dog", "hot"), which holds no joiner and no multi-byte
// entry, so the whole "a chunk splits inside a joiner or a rune" failure class
// was UNREACHABLE: 803k execs proved nothing about it, and two real data-loss
// bugs sat underneath. The vocabulary is part of the input space this property
// quantifies over, so it has to cover the same classes the tokenizer does.
var fuzzDeck = vocab(
	"obsequious", // plain
	"hot dog",    // a phrase, so the hold window is > 1 token
	"hot",        // and its prefix, so longest-match is exercised
	"don't",      // an apostrophe INSIDE a word
	"hot-dog",    // a hyphen inside a word
	"covid-19",   // digits
	"café",       // multi-byte: a chunk can split mid-rune
	"a priori",   // a phrase whose first token is one rune
)

// Chunk-independence is the property; byte identity is what makes it able to
// see a reorder, which visible-text equality cannot.
func FuzzHighlightWriterIsChunkIndependent(f *testing.F) {
	// Seeds carry the shapes the deck can match, including the split points that
	// broke it: a joiner at a chunk edge and a rune cut in half.
	for _, s := range []string{
		"his obsequious day", "\x1b[3;32mhot dog\x1b[0m", "hot\n dog", "'obsequious'", "a", "",
		"don't stop", "a hot-dog stand", "covid-19 era", "café society", "a priori truth",
		"'obsequious' don't café a priori",
	} {
		f.Add(s, 1)
	}
	f.Fuzz(func(t *testing.T, text string, at int) {
		v := fuzzDeck
		if at < 0 || at > len(text) {
			at = len(text) / 2
		}
		one := writeChunks(t, v, text)
		two := writeChunks(t, v, text[:at], text[at:])
		if one != two {
			t.Fatalf("split at %d changed the output:\n one %q\n two %q", at, one, two)
		}
		// And no visible text may be lost, whatever the styling. Both sides are
		// stripped: the INPUT carries escapes too, so comparing against the raw
		// text asserts that highlighting removes styling, which is not the claim.
		if got, want := stripANSI(two), stripANSI(text); got != want {
			t.Fatalf("visible text changed: got %q, want %q", got, want)
		}
	})
}

// Byte-at-a-time is the harshest split schedule there is: every joiner and every
// multi-byte rune is cut. A table can only sample the split points this covers
// exhaustively for a given text.
func TestHighlightWriterByteAtATimeMatchesOneCall(t *testing.T) {
	for _, text := range []string{
		"'obsequious' don't café a priori",
		"a hot-dog and a hot dog",
		"\x1b[3;32mdon't café\x1b[0m",
		"covid-19, obsequious.",
	} {
		t.Run(text, func(t *testing.T) {
			var bytes []string
			for i := 0; i < len(text); i++ {
				bytes = append(bytes, text[i:i+1])
			}
			one := writeChunks(t, fuzzDeck, text)
			if got := writeChunks(t, fuzzDeck, bytes...); got != one {
				t.Errorf("byte-at-a-time differs:\n got  %q\n want %q", got, one)
			}
		})
	}
}

// The motivating case, end to end through the production print path: the
// definition of `sycophantic` uses `obsequious` in its gloss, and a learner who
// has looked that up should see the connection.
//
// Driven through lookupAndRender, not through highlightText — a test that calls
// the helper directly begins after the wiring hop, which is how M1's set-to-screen
// link stayed unpinned.
func TestDefinitionBodyHighlightsADeckWord(t *testing.T) {
	rig, opt, _, _ := editorRig(t, "sycophantic", true)
	rig.deps.vocab = vocab("obsequious")
	var out, errb bytes.Buffer

	lookupAndRender(rig.deps, opt, replCommand{kind: cmdDefine, word: "sycophantic"}, &out, &errb)

	if !strings.Contains(out.String(), "\x1b[1;32mobsequious") {
		t.Errorf("the cross-referenced deck word was not highlighted: %q", out.String())
	}
}

func TestDefinitionHighlightingIsOffWithoutColour(t *testing.T) {
	rig, opt, _, _ := editorRig(t, "sycophantic", true)
	rig.deps.vocab = vocab("obsequious")
	opt.color = false
	var out, errb bytes.Buffer

	lookupAndRender(rig.deps, opt, replCommand{kind: cmdDefine, word: "sycophantic"}, &out, &errb)

	// Absence of ANY escape, not just of green: a test that only checks for the
	// highlight code passes while emitting bold, and Render with Color:false
	// makes the stronger assertion available for free.
	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("-no-color emitted an escape sequence: %q", out.String())
	}
	if !strings.Contains(out.String(), "obsequious") {
		t.Error("the word itself went missing")
	}
}

// Highlighting must not lose a byte of the entry. Render's no-data-loss
// invariant is asserted with colour OFF; this is the same property with
// highlighting ON, and it strips escapes first — the codes carry digits
// (1;32, 0) that alnum() would otherwise read as content.
func TestHighlightingLosesNothing(t *testing.T) {
	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("empty corpus — this test would be vacuous")
	}
	v := vocab("obsequious", "sycophantic", "a priori", "bank")
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			plain := Render(ParseEntry(raw), RenderOpts{Color: false})
			lit := stripANSI(Render(ParseEntry(raw), RenderOpts{Color: true, Vocab: v}))
			if lit != stripANSI(Render(ParseEntry(raw), RenderOpts{Color: true})) {
				t.Errorf("highlighting changed the visible text of %q", word)
			}
			if strings.Contains(plain, "\x1b") {
				t.Fatal("the colour-off baseline carries escapes; this comparison would be vacuous")
			}
			if lit != plain {
				t.Errorf("highlighting changed the visible text of %q", word)
			}
		})
	}
}

// The Spec's out-of-scope list: "Re-styling the headword line. The head is
// already bold cyan; highlighting is for body text, answers, and input."
//
// A learner revisits words constantly, so looking up a word already in the deck
// is the COMMON case, and wrapping the whole rendered string put green inside
// the cyan headword. The decision is now per region (RenderOpts.admitsHighlight)
// and this pins the withhold half; TestDefinitionBodyHighlightsADeckWord pins
// the admit half.
func TestTheHeadwordLineIsNotHighlighted(t *testing.T) {
	raw := fixture(t, "sycophantic")

	out := Render(ParseEntry(raw), RenderOpts{Color: true, Vocab: vocab("sycophantic")})

	head := strings.SplitN(out, "\n", 2)[0]
	if strings.Contains(head, knownOn) {
		t.Errorf("the headword line was highlighted: %q", head)
	}
	// ...and the withhold is scoped to the region, not to the word: the same
	// word inside a sense body still highlights.
	if !strings.Contains(out, knownOn) {
		t.Error("nothing at all was highlighted; the withhold is too wide")
	}
}
