package play

import (
	"strings"
	"testing"
)

func TestRecallGrade(t *testing.T) {
	r := NewRecall("obsequious", "fawning attentively")

	for _, tc := range []struct {
		name   string
		key    rune
		want   Verdict
		wantOK bool
	}{
		{"y is correct", 'y', Correct, true},
		{"Y is correct too — a session is typed fast", 'Y', Correct, true},
		{"n is wrong", 'n', Wrong, true},
		{"N is wrong too", 'N', Wrong, true},
		// A stray key must NOT become a silent wrong answer: that would demote a
		// word the learner never rated and corrupt the schedule.
		{"an unrelated key means nothing", 'q', Skipped, false},
		{"a digit means nothing to THIS form", '3', Skipped, false},
		{"space means nothing", ' ', Skipped, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := r.Grade(tc.key)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("Grade(%q) = (%v, %v), want (%v, %v)", tc.key, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// The entire point of a recall form: the prompt must not give away the answer.
// A careless implementation shows the whole entry and the exercise evaporates.
func TestRecallPromptHidesTheDefinition(t *testing.T) {
	r := NewRecall("obsequious", "fawning attentively in a servile way")

	if got := r.Prompt(); got != "obsequious" {
		t.Errorf("Prompt() = %q, want just the word", got)
	}
	if strings.Contains(r.Prompt(), "fawning") {
		t.Error("the prompt leaks the definition — there is nothing left to recall")
	}
	if !strings.Contains(r.Reveal(), "fawning") {
		t.Error("Reveal does not show the definition")
	}
}

func TestRecallWordIsTheKeyTheAnswerIsRecordedAgainst(t *testing.T) {
	if got := NewRecall("hot dog", "a sausage").Word(); got != "hot dog" {
		t.Errorf("Word() = %q", got)
	}
}
