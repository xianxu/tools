package main

import (
	"unicode"

	"github.com/xianxu/tools/cmd/define/store"
)

// knownOn is the style for a word the learner has looked up: bold green.
//
// Green over the alternatives because it reads as "known" and is instantly
// separable from the grey suggestion and the plain bold of ordinary input. It is
// the one accent the prompt line does not already use — amber is the
// part-of-speech label in definition bodies, cyan is the prompt itself.
const knownOn = "\x1b[1;32m"

// isWordRune decides what counts as part of a word, and it is the single
// predicate both the span matcher and the streaming writer tokenise by.
//
// Apostrophes and hyphens are INSIDE a word, so `don't` and `hot-dog` are each
// one token. That matters for hyphens especially: a hyphenated deck entry has
// `hot-dog` as its store.Key, so splitting on the hyphen would make it
// unmatchable. Space-separated phrases like `hot dog` are handled a level up, by
// the phrase window — a different mechanism for a genuinely different thing.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '-'
}

// wordRun is one token's byte range in the text it came from.
type wordRun struct{ start, end int }

// wordRuns returns the maximal runs of word characters, as byte offsets, with
// leading and trailing joiners trimmed.
//
// The trim matters because apostrophes and hyphens are word characters only
// INSIDE a word: `don't` is one token, but `'obsequious'` must tokenize to the
// word, not to the word wearing quotes, or a quoted deck word never matches.
// Definition bodies quote and dash routinely, so this is load-bearing from M2 on.
// A run that is nothing but joiners (`--`) trims to empty and is dropped.
//
// Byte offsets rather than rune indices because every consumer slices the
// original string with them, and converting to runes and back is where
// multi-byte text gets corrupted.
func wordRuns(text string) []wordRun {
	var out []wordRun
	start, in := 0, false
	for i, r := range text {
		switch {
		case isWordRune(r) && !in:
			start, in = i, true
		case !isWordRune(r) && in:
			out = appendTrimmed(out, text, start, i)
			in = false
		}
	}
	if in {
		out = appendTrimmed(out, text, start, len(text))
	}
	return out
}

// isJoiner reports the word characters that may not stand at a token's edge.
func isJoiner(r rune) bool { return r == '\'' || r == '-' }

func appendTrimmed(out []wordRun, text string, start, end int) []wordRun {
	for start < end && isJoiner(rune(text[start])) {
		start++
	}
	for end > start && isJoiner(rune(text[end-1])) {
		end--
	}
	if start >= end {
		return out
	}
	return append(out, wordRun{start, end})
}

// span is one run of text plus whether it is a word the learner knows.
//
// Concatenating every span's text reproduces the input EXACTLY. That invariant
// is the whole safety story: a renderer joins spans back into what the user
// sees, so a lossy split silently corrupts a definition. FuzzHighlightSpans
// defends it, the same way FuzzTrailingSegments defends #20's head+text rule.
type span struct {
	text  string
	known bool
}

// phraseGap reports whether the text between two tokens lets them be one phrase.
//
// Only spaces and tabs qualify. store.Key collapses ALL whitespace, so without
// this a Render-wrapped "hot\n  dog" would form the candidate "hot dog", match,
// and paint a green run straight through the wrap indent. Punctuation is
// excluded for the same reason in miniature: "hot, dog" is not "hot dog".
func phraseGap(gap string) bool {
	if gap == "" {
		return false // tokens cannot be adjacent; wordRuns are maximal
	}
	for _, r := range gap {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

// highlightSpans splits text into alternating known/unknown runs.
//
// Longest match wins at each position: with both `hot` and `hot dog` in the set,
// "one hot dog please" highlights the phrase, not the first word of it. That is
// why the loop counts DOWN from MaxPhraseWords rather than up.
func highlightSpans(text string, v Vocabulary) []span {
	if v == nil {
		return nonEmptySpan(text, false)
	}
	runs := wordRuns(text)
	maxWords := v.MaxPhraseWords()
	var out []span
	emitted := 0 // byte offset up to which text has been turned into spans

	for i := 0; i < len(runs); i++ {
		n := maxWords
		if rest := len(runs) - i; n > rest {
			n = rest
		}
		for ; n >= 1; n-- {
			last := i + n - 1
			if !phraseRunsJoin(text, runs[i:last+1]) {
				continue
			}
			start, end := runs[i].start, runs[last].end
			if !v.Has(store.Key(text[start:end])) {
				continue
			}
			out = append(out, nonEmptySpan(text[emitted:start], false)...)
			out = append(out, span{text: text[start:end], known: true})
			emitted = end
			i = last // continue after the phrase, not inside it
			break
		}
	}
	return append(out, nonEmptySpan(text[emitted:], false)...)
}

// phraseRunsJoin reports whether consecutive tokens are separated only by
// phrase-legal gaps, so slicing from the first to the last yields a candidate
// key rather than a span of unrelated text.
func phraseRunsJoin(text string, runs []wordRun) bool {
	for i := 1; i < len(runs); i++ {
		if !phraseGap(text[runs[i-1].end:runs[i].start]) {
			return false
		}
	}
	return true
}

// nonEmptySpan keeps the "no empty span" half of the invariant in one place, so
// no caller has to remember to check.
func nonEmptySpan(text string, known bool) []span {
	if text == "" {
		return nil
	}
	return []span{{text: text, known: known}}
}
