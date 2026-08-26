package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// vocab builds a Vocabulary for tests. Every highlight test runs against this
// and touches no IO — the purity boundary is visible from outside (ARCH-PURE).
func vocab(words ...string) Vocabulary {
	v := &memVocabulary{}
	for _, w := range words {
		v.Add(w)
	}
	return v
}

// The table pins the DECISIONS about what a word character is, one row per rule.
// The class of "every other punctuation mark" belongs to FuzzWordRuns, not here.
func TestWordRuns(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want []string
	}{
		{"empty", "", nil},
		{"only spaces", "   ", nil},
		{"only joiners trims to nothing", "-- ... --", nil},
		{"quotes and dashes are trimmed off the edges", "'obsequious' --truly--", []string{"obsequious", "truly"}},
		{"plain words", "his manner", []string{"his", "manner"}},
		{"trailing punctuation is not part of the word", "obsequious, truly.", []string{"obsequious", "truly"}},
		{"an apostrophe is INSIDE a word", "don't", []string{"don't"}},
		{"a hyphen is INSIDE a word", "hot-dog", []string{"hot-dog"}},
		{"digits count", "covid-19", []string{"covid-19"}},
		{"multi-byte runes are one word", "café", []string{"café"}},
		{"parenthesised", "(obsequious)", []string{"obsequious"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := wordRuns(tc.text)
			var got []string
			for _, r := range runs {
				got = append(got, tc.text[r.start:r.end])
			}
			if strings.Join(got, "|") != strings.Join(tc.want, "|") {
				t.Errorf("wordRuns(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

// wordRuns is the byte-offset source of truth for BOTH the span matcher and the
// streaming writer. If they disagree about where a word begins, a word
// highlighted in a definition would not highlight at the prompt — so the offsets
// get a property of their own rather than only being exercised through callers.
func FuzzWordRuns(f *testing.F) {
	for _, s := range []string{"", "   ", "his manner", "don't", "hot-dog", "café", "¿qué tal?", "a-b'c", "--", "covid-19"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		prevEnd := -1
		for i, r := range wordRuns(text) {
			if r.start < 0 || r.end > len(text) || r.start >= r.end {
				t.Fatalf("run %d of %q is %v, want a non-empty range inside the text", i, text, r)
			}
			if r.start < prevEnd {
				t.Fatalf("run %d of %q starts at %d, overlapping the previous end %d", i, text, r.start, prevEnd)
			}
			prevEnd = r.end
			// Slicing must not split a rune, and everything inside must be a word
			// character — otherwise the matcher builds keys out of punctuation.
			for _, ru := range text[r.start:r.end] {
				if !isWordRune(ru) {
					t.Fatalf("run %d of %q contains %q, which is not a word rune", i, text, ru)
				}
			}
			// A run never begins or ends with a joiner: that is what makes
			// '"'"'obsequious'"'"' match a deck key rather than tokenizing with its quotes.
			if isJoiner(rune(text[r.start])) || isJoiner(rune(text[r.end-1])) {
				t.Fatalf("run %d of %q is %q, which has a joiner at an edge", i, text, text[r.start:r.end])
			}
			// Maximal MODULO the trim: a run may be adjacent to a joiner (it was
			// trimmed off), but never to a letter or digit — that would be a word
			// split in half.
			//
			// The property used to demand plain maximality, which the trim made
			// false by design. It stayed green because `go test` runs a fuzz
			// target against its SEED CORPUS only, and no seed happened to put a
			// joiner beside a kept run. Re-fuzzed, not just re-run.
			if r.start > 0 {
				before := []rune(text[:r.start])
				if last := before[len(before)-1]; isWordRune(last) && !isJoiner(last) {
					t.Fatalf("run %d of %q is split from a letter on the left", i, text)
				}
			}
			for _, ru := range text[r.end:] {
				if isWordRune(ru) && !isJoiner(ru) {
					t.Fatalf("run %d of %q is split from a letter on the right", i, text)
				}
				break
			}
		}
	})
}

// spanText joins spans back into what the renderer will show, and marks the
// known ones, so a test asserts on one readable string.
func marked(spans []span) string {
	var b strings.Builder
	for _, s := range spans {
		if s.known {
			b.WriteString("[" + s.text + "]")
			continue
		}
		b.WriteString(s.text)
	}
	return b.String()
}

func TestHighlightSpans(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    Vocabulary
		text string
		want string
	}{
		{"a known word", vocab("obsequious"), "his obsequious manner", "his [obsequious] manner"},
		{"nothing known", vocab("obsequious"), "his manner", "his manner"},
		{"empty vocabulary", vocab(), "his obsequious manner", "his obsequious manner"},
		{"empty text", vocab("obsequious"), "", ""},
		{"case-insensitive", vocab("obsequious"), "Obsequious and OBSEQUIOUS", "[Obsequious] and [OBSEQUIOUS]"},
		{"punctuation around a word", vocab("obsequious"), "(obsequious),", "([obsequious]),"},
		// The fixture holds BOTH "hot" and "hot dog", so the two rules produce
		// different output and the test can tell them apart. A fixture holding
		// only the phrase would pass under either rule (lessons.md, #20).
		{"longest phrase wins", vocab("hot", "hot dog"), "one hot dog please", "one [hot dog] please"},
		{"the short word still matches on its own", vocab("hot", "hot dog"), "too hot today", "too [hot] today"},
		// Exact match only: the operator chose this over a suffix list.
		{"inflections do not match", vocab("obsequious"), "his obsequiousness", "his obsequiousness"},
		{"a longer word is not a match", vocab("ration"), "rationing", "rationing"},
		{"hyphenated entry", vocab("hot-dog"), "a hot-dog stand", "a [hot-dog] stand"},
		// Contract rule 2: a phrase spans only spaces and tabs.
		{"a phrase does not span a newline", vocab("hot dog"), "hot\n  dog", "hot\n  dog"},
		{"a phrase does not span punctuation", vocab("hot dog"), "hot, dog", "hot, dog"},
		{"a phrase does span a run of spaces", vocab("hot dog"), "hot  dog", "[hot  dog]"},
		{"adjacent known words stay separate spans", vocab("hot", "dog"), "hot dog", "[hot] [dog]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := marked(highlightSpans(tc.text, tc.v)); got != tc.want {
				t.Errorf("highlightSpans(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

// A renderer joins spans back into what the user sees. If that join is ever
// lossy, a definition silently loses text — the worst thing this feature can do.
func FuzzHighlightSpans(f *testing.F) {
	for _, s := range []string{"", "   ", "his obsequious manner", "hot dog", "hot\n dog", "café obsequious", "(obsequious),", "a-b'c"} {
		f.Add(s)
	}
	v := vocab("obsequious", "hot", "hot dog", "café", "a-b'c")
	f.Fuzz(func(t *testing.T, text string) {
		spans := highlightSpans(text, v)

		var b strings.Builder
		for i, s := range spans {
			if s.text == "" {
				t.Fatalf("span %d of %q is empty", i, text)
			}
			if i > 0 && spans[i-1].known == s.known {
				t.Fatalf("spans %d and %d of %q share known=%v; spans must be maximal",
					i-1, i, text, s.known)
			}
			b.WriteString(s.text)
		}
		if got := b.String(); got != text {
			t.Fatalf("joining spans of %q gave %q — the render would show text nobody wrote", text, got)
		}
	})
}

// editorOn builds an Editor with the line typed and the cursor at the end.
func editorOn(line string) Editor {
	e := NewEditor()
	e.Line = []rune(line)
	e.Cursor = len(e.Line)
	return e
}

func TestRenderLineHighlightsAKnownWord(t *testing.T) {
	got := RenderLine(editorOn("what is obsequious"), "", vocab("obsequious"), true)

	// Literal bytes, not knownOn+"obsequious". Using the constant compares it
	// against itself: aliasing knownOn to inputOn produces no visible highlight
	// at all and the constant-based assertion still passes. That mutant survived
	// the first version of this test.
	if !strings.Contains(got, "\x1b[1;32mobsequious") {
		t.Errorf("no bold-green highlight on a deck word: %q", got)
	}
	// And the highlight must differ from ordinary input styling, which is the
	// property "it is easier to spot" actually rests on.
	if knownOn == inputOn {
		t.Error("the highlight style is the same as ordinary input; nothing stands out")
	}
}

// The part a naive implementation gets wrong: ANSI does not nest, so after the
// highlight ends the rest of the line must be told to be bold again. Without
// this the tail of every line goes plain after the first known word.
func TestRenderLineResumesInputStyleAfterAHighlight(t *testing.T) {
	got := RenderLine(editorOn("obsequious manner"), "", vocab("obsequious"), true)

	const lit = "\x1b[1;32mobsequious"
	i := strings.Index(got, lit)
	if i < 0 {
		t.Fatalf("no highlight at all: %q", got)
	}
	rest := got[i+len(lit):]
	if !strings.HasPrefix(rest, "\x1b[0m\x1b[1m") {
		t.Errorf("after the highlight the line reads %q, want it closed then returned to bold", rest)
	}
}

func TestRenderLineWithoutColourEmitsNoStyle(t *testing.T) {
	got := RenderLine(editorOn("what is obsequious"), "ly", vocab("obsequious"), false)

	for _, style := range []string{knownOn, inputOn, promptOn, greyOn, sgrOff} {
		if strings.Contains(got, style) {
			t.Errorf("no-colour output carries style %q: %q", style, got)
		}
	}
	if !strings.Contains(got, "what is obsequious") {
		t.Errorf("the typed text is missing: %q", got)
	}
}

// Highlighting adds BYTES but no visible columns, and the cursor-park count is
// in columns. Pinning it against the un-highlighted render is the direct way to
// say "this feature may not move the cursor".
func TestHighlightingDoesNotMoveTheCursor(t *testing.T) {
	e := editorOn("what is obsequious")
	e.Cursor = 4 // mid-line, so there is a real park to compute

	plain := RenderLine(e, "sug", nil, true)
	lit := RenderLine(e, "sug", vocab("obsequious"), true)

	park := func(s string) string {
		i := strings.LastIndex(s, "\x1b[")
		return s[i:]
	}
	if park(plain) != park(lit) {
		t.Errorf("park moved: plain %q vs highlighted %q", park(plain), park(lit))
	}
}

// A nil vocabulary is the "no highlighting configured" case and must render
// exactly as it did before this feature existed.
func TestRenderLineWithoutAVocabularyIsUnchanged(t *testing.T) {
	e := editorOn("what is obsequious")

	if got := RenderLine(e, "", nil, true); !strings.Contains(got, inputOn+"what is obsequious"+sgrOff) {
		t.Errorf("nil vocabulary changed the render: %q", got)
	}
}

// BR-1a: the set→screen link is the M1 Done-when, and nothing pinned it. Both
// "pass nil instead of the vocabulary" and "delete voc.Load()" survived the
// entire suite, because every other test called RenderLine directly. A test that
// reaches around the production wiring cannot see the wiring break — the same
// class as #20's typeKeys.
func TestEditorLoopHighlightsADeckWordOnScreen(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	st := store.NewMem()
	if err := st.Upsert(store.Word{Text: "sycophantic"}); err != nil {
		t.Fatal(err)
	}
	// A storeVocabulary, not a pre-filled memVocabulary: this is what makes
	// deleting voc.Load() redden, since an unloaded store set is empty.
	rig.deps.vocab = newStoreVocabulary(st, nil)

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic"), nil, rig.deps, opt, cooked, finish, &out, &errb)

	if !strings.Contains(out.String(), "\x1b[1;32msycophantic") {
		t.Errorf("the deck word was not highlighted on screen: %q", out.String())
	}
}

// The in-session-growth Done-when, end to end. The first submit echoes BEFORE
// the lookup records anything, so the highlight can only appear on the retype —
// which is exactly the behaviour worth pinning.
func TestEditorLoopHighlightsAWordLookedUpThisSession(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	st := store.NewMem()
	voc := newStoreVocabulary(st, nil)
	rig.deps.vocab = voc
	rig.deps.capture = newStoreCapturer(st, store.FixedClock(aDay), nil, voc)

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\rsycophantic"), nil, rig.deps, opt, cooked, finish, &out, &errb)

	if !strings.Contains(out.String(), "\x1b[1;32msycophantic") {
		t.Errorf("a word looked up this session did not highlight on retype: %q", out.String())
	}
}

// A deck key carrying punctuation cannot currently match, because a phrase spans
// only spaces and tabs. Pinned as KNOWN behaviour so M2 does not rediscover it
// as a bug when definition bodies widen the input.
func TestAPunctuatedKeyIsNotMatchable(t *testing.T) {
	if got := marked(highlightSpans("see e.g. this", vocab("e.g."))); got != "see e.g. this" {
		t.Errorf("got %q — if this now matches, the phrase rule changed and the comment in vocab.go is stale", got)
	}
}
