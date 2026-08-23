package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestReadsAsQuestion. Every row here is a line NOAD MISSED — that containment
// is the whole safety argument, so a row for a real headword would be a lie
// about what this function is ever asked. "hot dog" and "use" are lookups
// because the dictionary has them, not because this function was careful; see
// TestConsoleDecisionTable for the end-to-end version.
func TestReadsAsQuestion(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want bool
	}{
		{"trailing question mark", "is it pejorative?", true},
		{"wh opener", "what's the difference to obsequious", true},
		{"contracted aux opener", "isn't that the same as fawning", true},
		{"a wh word ending in n is not a negation", "when is it used", true},
		{"request verb with an object", "use it in a sentence", true},
		{"follow-up request", "give me three more examples", true},
		{"length alone is not a signal", "difference between sycophantic and obsequious please", false},
		{"a typo is not a question", "sycophanti", false},
		{"a bare request verb is a word", "give", false},
		{"a wh word alone is a word", "what", false},
		{"a two-word phrase is a headword shape", "amuse bouche", false},
		{"punctuation only", "?", false},
		{"empty", "", false},
		{"whitespace only", "   \t ", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readsAsQuestion(tc.line); got != tc.want {
				t.Errorf("readsAsQuestion(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

func TestTruncateQuestion(t *testing.T) {
	long := "what is the difference between sycophantic and obsequious"
	got := truncateQuestion(long)
	if r := []rune(got); r[len(r)-1] != '…' {
		t.Errorf("truncateQuestion(%q) = %q, want an ellipsis", long, got)
	}
	if n := len([]rune(got)); n > maxQuestionRunes+1 {
		t.Errorf("truncateQuestion returned %d runes, want <= %d", n, maxQuestionRunes+1)
	}
	if got := truncateQuestion("why"); got != "why" {
		t.Errorf("truncateQuestion(%q) = %q, want it unchanged", "why", got)
	}
	// Rune-safe: cutting mid-rune would print a replacement character.
	if got := truncateQuestion(strings.Repeat("é", 60)); !utf8.ValidString(got) {
		t.Errorf("truncateQuestion produced invalid UTF-8: %q", got)
	}
}
