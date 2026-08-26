package main

import (
	"strings"
	"unicode"
)

// minInnerSegment is how many runes an INNER segment must have before it is
// allowed to complete anything.
//
// Without a floor, every finished short word grows a grey tail: typing "to"
// mid-sentence suggests "torpid", "is" suggests "island", and the suggestion
// stops meaning "I recognise where you are going" and starts meaning "here is a
// word". Three runes is where a tail stops looking like a preposition.
//
// It deliberately does NOT apply to segment 0, the whole line. There is no
// ambiguity there about what is being completed, and applying it would regress
// single-word typeahead: "ob" completes to "obsequious" today and must keep
// doing so.
const minInnerSegment = 3

// segment is a trailing slice of a line, cut at a word boundary, plus the head
// it was cut from.
//
// head+text == line, always. That invariant is the whole safety story of
// gluing: a candidate is joined back onto head and rendered as the user's own
// line, so if it ever failed, the editor would display text nobody typed.
// FuzzTrailingSegments defends it.
type segment struct {
	head string
	text string
}

// trailingSegments returns the line's trailing word-boundary segments, longest
// first — the whole line, then the line minus its first word, and so on:
//
//	"what is the diff" -> ["what is the diff", "is the diff", "the diff", "diff"]
//
// A segment starts only where a word starts, so a trailing space produces no
// segment. That is load-bearing rather than incidental: History.Prefix("")
// returns EVERY entry by contract, so an empty segment would make every space
// the user types suggest an unrelated line.
func trailingSegments(line string) []segment {
	var out []segment
	atBoundary := true
	for i, r := range line {
		if unicode.IsSpace(r) {
			atBoundary = true
			continue
		}
		if atBoundary {
			out = append(out, segment{head: line[:i], text: line[i:]})
		}
		atBoundary = false
	}
	return out
}

// submissionMarkers are the prefixes recallLine puts on a line to make it
// re-submittable, and that the completion namespace has to see past.
//
// "?" is here because the SYSTEM adds it, not the user: readsAsQuestion
// classifies a bare sentence as a question and recallLine stores it "?…". So
// requiring a "?" to complete a question that was asked without one would make
// past questions effectively uncompletable — the opposite of the point.
// forceLiteral is the same shape: a hatch that disambiguates submission,
// wrapping what is exactly a deck word.
//
// "/" is deliberately NOT here. Commands are a real separate namespace with
// their own completion path in completionsFor, not a marker on a word.
var submissionMarkers = []string{"?", forceLiteral}

// matchesFor returns the history entries that could complete text, with the
// submission markers unwrapped so a marked entry completes as the thing behind
// the marker.
//
// Bare matches come first: an unmarked line is the ordinary case, and Suggestion
// takes the first match.
func matchesFor(hist History, text string) []string {
	out := hist.Prefix(text)
	seen := make(map[string]bool, len(out))
	for _, m := range out {
		seen[m] = true
	}
	for _, mark := range submissionMarkers {
		for _, m := range hist.Prefix(mark + text) {
			unwrapped := strings.TrimPrefix(m, mark)
			if seen[unwrapped] {
				continue
			}
			seen[unwrapped] = true
			out = append(out, unwrapped)
		}
	}
	return out
}

// historyCompletions answers "what could this line become", from the lines that
// were actually submitted.
//
// It walks the trailing segments longest-first and returns the first segment
// that matches anything, glued back onto its head. Longest-first is what keeps
// pre-#20 behaviour intact and gives it precedence: segment 0 IS the whole line,
// so a real past line always beats a word glued onto a head.
//
// Note what is NOT here: any test for the command namespace. completionsFor
// makes that decision once, on the whole line, before calling this. Re-testing
// per segment would let the "7" in "/history 7" fall through to history, and let
// any trailing "/…" complete a command mid-line.
func historyCompletions(base string, hist History) []string {
	for i, s := range trailingSegments(base) {
		if i > 0 && len([]rune(s.text)) < minInnerSegment {
			// Segments only get shorter, so nothing after this clears the floor.
			break
		}
		if ms := matchesFor(hist, s.text); len(ms) > 0 {
			return glue(s.head, ms)
		}
	}
	return nil
}

// glue re-attaches the head a segment was cut from, so what comes back is a
// whole line. That is what lets Suggestion keep matching against the whole typed
// line — the editor never learns about word boundaries, exactly as command mode
// never made it learn about commands.
func glue(head string, ms []string) []string {
	if head == "" {
		return ms
	}
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = head + m
	}
	return out
}
