package main

import (
	"slices"
	"strings"
	"testing"
)

// typeKeys drives the editor the way the loop does: resolve candidates from
// history against whatever the walk is anchored on, then Apply.
func typeKeys(h History, keys ...Key) (Editor, Action) {
	e := NewEditor()
	act := ActNone
	for _, k := range keys {
		e, act = Apply(e, k, h.Prefix(e.WalkBase()))
	}
	return e, act
}

func runes(s string) []Key {
	var ks []Key
	for _, r := range s {
		ks = append(ks, Key{Kind: KeyRune, Rune: r})
	}
	return ks
}

func hist(words ...string) History {
	h := &memHistory{}
	for _, w := range words {
		h.Add(w, true)
	}
	return h
}

func TestEditorInsertsAtCursorNotJustAtEnd(t *testing.T) {
	e, _ := typeKeys(hist(), append(runes("ac"), Key{Kind: KeyLeft}, Key{Kind: KeyRune, Rune: 'b'})...)
	if e.String() != "abc" {
		t.Errorf("got %q, want %q", e.String(), "abc")
	}
	if e.Cursor != 2 {
		t.Errorf("cursor = %d, want 2", e.Cursor)
	}
}

func TestEditorBoundariesAreNoOpsNotPanics(t *testing.T) {
	// Backspace at 0, Delete at end, Left at 0, Right at end on an empty line.
	e, _ := typeKeys(hist(),
		Key{Kind: KeyBackspace}, Key{Kind: KeyDelete},
		Key{Kind: KeyLeft}, Key{Kind: KeyRight})
	if e.String() != "" || e.Cursor != 0 {
		t.Errorf("got %q cursor %d, want empty", e.String(), e.Cursor)
	}
}

func TestEditorHomeEndAndDelete(t *testing.T) {
	e, _ := typeKeys(hist(), append(runes("word"), Key{Kind: KeyHome}, Key{Kind: KeyDelete})...)
	if e.String() != "ord" {
		t.Errorf("got %q, want %q", e.String(), "ord")
	}
	e, _ = typeKeys(hist(), append(runes("word"), Key{Kind: KeyHome}, Key{Kind: KeyEnd})...)
	if e.Cursor != 4 {
		t.Errorf("cursor = %d, want 4", e.Cursor)
	}
}

func TestEditorActions(t *testing.T) {
	if _, a := typeKeys(hist(), append(runes("hi"), Key{Kind: KeyEnter})...); a != ActSubmit {
		t.Errorf("Enter = %v, want ActSubmit", a)
	}
	if _, a := typeKeys(hist(), Key{Kind: KeyInterrupt}); a != ActInterrupt {
		t.Errorf("Ctrl-C = %v, want ActInterrupt", a)
	}
	if _, a := typeKeys(hist(), Key{Kind: KeyEOF}); a != ActEOF {
		t.Errorf("Ctrl-D on empty = %v, want ActEOF", a)
	}
	// Ctrl-D on a NON-empty line does nothing. The asymmetry is standard and
	// easy to get wrong — quitting mid-word would be infuriating.
	if e, a := typeKeys(hist(), append(runes("hi"), Key{Kind: KeyEOF})...); a != ActNone || e.String() != "hi" {
		t.Errorf("Ctrl-D on %q = %v, want ActNone", e.String(), a)
	}
}

func TestEditorIgnoresUnknownKeys(t *testing.T) {
	e, _ := typeKeys(hist(), append(runes("ok"), Key{Kind: KeyUnknown, Raw: []byte("\x1b[5~")})...)
	if e.String() != "ok" {
		t.Errorf("an unmodelled key altered the line: %q", e.String())
	}
}

// --- history walk -----------------------------------------------------------

func TestHistoryWalkNewestFirst(t *testing.T) {
	h := hist("alpha", "beta", "gamma")
	e, _ := typeKeys(h, Key{Kind: KeyUp})
	if e.String() != "gamma" {
		t.Errorf("first Up = %q, want gamma", e.String())
	}
	e, _ = typeKeys(h, Key{Kind: KeyUp}, Key{Kind: KeyUp})
	if e.String() != "beta" {
		t.Errorf("second Up = %q, want beta", e.String())
	}
}

// Walking back past the newest entry must restore what was typed before the
// walk began — losing the draft is the classic history bug.
func TestHistoryWalkRestoresTheDraft(t *testing.T) {
	h := hist("alpha", "beta")
	keys := append(runes("dra"), Key{Kind: KeyUp}, Key{Kind: KeyUp}, Key{Kind: KeyDown}, Key{Kind: KeyDown})
	e, _ := typeKeys(h, keys...)
	if e.String() != "dra" {
		t.Errorf("got %q, want the draft %q restored", e.String(), "dra")
	}
	if e.Cursor != 3 {
		t.Errorf("cursor = %d, want 3", e.Cursor)
	}
}

func TestHistoryWalkExhaustsWithoutWrapping(t *testing.T) {
	h := hist("only")
	e, _ := typeKeys(h, Key{Kind: KeyUp}, Key{Kind: KeyUp}, Key{Kind: KeyUp})
	if e.String() != "only" {
		t.Errorf("got %q — the walk wrapped or overran", e.String())
	}
}

// With text typed, Up walks only entries starting with it — zsh's
// history-beginning-search-backward, not a plain walk.
func TestHistoryPrefixSearch(t *testing.T) {
	h := hist("sycophantic", "banana", "syzygy", "cherry")
	e, _ := typeKeys(h, append(runes("sy"), Key{Kind: KeyUp})...)
	if e.String() != "syzygy" {
		t.Errorf("first Up with prefix sy = %q, want syzygy", e.String())
	}
	e, _ = typeKeys(h, append(runes("sy"), Key{Kind: KeyUp}, Key{Kind: KeyUp})...)
	if e.String() != "sycophantic" {
		t.Errorf("second Up = %q, want sycophantic", e.String())
	}
	e, _ = typeKeys(h, append(runes("sy"), Key{Kind: KeyUp}, Key{Kind: KeyUp}, Key{Kind: KeyUp})...)
	if e.String() != "sycophantic" {
		t.Errorf("walk left the matching set: %q", e.String())
	}
}

func TestHistoryDedupesKeepingNewest(t *testing.T) {
	h := hist("word", "other", "word")
	if got := h.Prefix(""); !slices.Equal(got, []string{"word", "other"}) {
		t.Errorf("Prefix = %v, want [word other]", got)
	}
}

// --- autosuggestion ---------------------------------------------------------

func TestSuggestionOffersNewestMatch(t *testing.T) {
	h := hist("sycophantic", "sympathy")
	e, _ := typeKeys(h, runes("sy")...)
	if got := Suggestion(e, h.Prefix(e.WalkBase())); got != "mpathy" {
		t.Errorf("suggestion = %q, want %q", got, "mpathy")
	}
}

func TestSuggestionAcceptedByRightAndEnd(t *testing.T) {
	h := hist("sycophantic")
	for _, k := range []Key{{Kind: KeyRight}, {Kind: KeyEnd}} {
		e, _ := typeKeys(h, append(runes("syc"), k)...)
		if e.String() != "sycophantic" {
			t.Errorf("key %v: got %q, want the suggestion accepted", k.Kind, e.String())
		}
	}
}

func TestSuggestionNotAcceptedByOtherKeys(t *testing.T) {
	h := hist("sycophantic")
	for _, k := range []Key{{Kind: KeyLeft}, {Kind: KeyBackspace}, {Kind: KeyRune, Rune: 'x'}, {Kind: KeyHome}} {
		e, _ := typeKeys(h, append(runes("syc"), k)...)
		if strings.HasPrefix(e.String(), "sycophantic") {
			t.Errorf("key %v accepted the suggestion: %q", k.Kind, e.String())
		}
	}
}

// The most damaging possible bug in this issue: submitting a word the user
// never typed.
func TestEnterSubmitsOnlyWhatWasTyped(t *testing.T) {
	h := hist("sycophantic")
	e, act := typeKeys(h, append(runes("syc"), Key{Kind: KeyEnter})...)
	if act != ActSubmit {
		t.Fatalf("act = %v", act)
	}
	if e.String() != "syc" {
		t.Errorf("submitted %q — Enter accepted the suggestion", e.String())
	}
}

func TestSuggestionSuppressedMidLineAndWhenUnmatched(t *testing.T) {
	h := hist("sycophantic")
	e, _ := typeKeys(h, append(runes("syc"), Key{Kind: KeyLeft})...)
	if got := Suggestion(e, h.Prefix(e.WalkBase())); got != "" {
		t.Errorf("suggested %q with the cursor mid-line", got)
	}
	e, _ = typeKeys(h, runes("zzz")...)
	if got := Suggestion(e, h.Prefix(e.WalkBase())); got != "" {
		t.Errorf("suggested %q with no match", got)
	}
	e, _ = typeKeys(h)
	if got := Suggestion(e, h.Prefix(e.WalkBase())); got != "" {
		t.Errorf("suggested %q on an empty line", got)
	}
}
