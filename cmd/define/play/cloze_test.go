package play

import (
	"strings"
	"testing"
)

func clozeOptions() []Option {
	return []Option{
		{Word: "ephemeral"},
		{Word: "sycophantic", Correct: true},
		{Word: "defenestrate"},
	}
}

const (
	clozeBlanked  = "The Times dismissed Oliver Stone's Putin interviews as ___."
	clozeRestored = "The Times dismissed Oliver Stone's Putin interviews as sycophantic."
)

func newTestCloze() *Cloze {
	return NewCloze("sycophantic", clozeBlanked, clozeRestored, "flattering to gain advantage", clozeOptions())
}

// THE PROMPT IS THE QUESTION, and the question must not contain its answer.
func TestClozePromptHidesTheAnswerAndShowsTheWords(t *testing.T) {
	got := newTestCloze().Prompt()

	// The blanked sentence leads. Not the headword — Choice puts the word on
	// line 0 and #38 reads it there, but showing it here would answer the
	// question outright.
	if !strings.HasPrefix(got, clozeBlanked) {
		t.Errorf("prompt does not lead with the blanked sentence:\n%s", got)
	}
	// The answer is absent from the SENTENCE. It is necessarily present among the
	// options — that is what makes it answerable — so the assertion is scoped to
	// the stem, which is the part that must not give it away.
	stem, _, _ := strings.Cut(got, "\n")
	if strings.Contains(strings.ToLower(stem), "sycophantic") {
		t.Errorf("the blanked sentence still contains the answer: %q", stem)
	}
	// Every option is offered, BY WORD — a cloze option is a word, where 2.3's
	// is a definition. Both live in Option; each form renders the field it shows.
	for _, o := range clozeOptions() {
		if !strings.Contains(got, o.Word) {
			t.Errorf("option %q is missing from the prompt:\n%s", o.Word, got)
		}
	}
	// Numbered from 1, like every option form.
	if !strings.Contains(got, optionLine(0, "ephemeral")) {
		t.Errorf("options are not numbered from 1:\n%s", got)
	}
}

// THE REVEAL IS THE PAYLOAD: the sentence with the word doing its work.
func TestClozeRevealRestoresTheSentence(t *testing.T) {
	c := newTestCloze()
	got := c.Reveal()
	if !strings.HasPrefix(got, clozeRestored) {
		t.Errorf("the reveal does not lead with the restored sentence:\n%s", got)
	}
	// And the entry, for the reason choice.go gives: a learner who just missed a
	// word wants more than the one sentence.
	if !strings.Contains(got, "flattering to gain advantage") {
		t.Errorf("the reveal drops the definition:\n%s", got)
	}
	// Nothing about a wrong pick when there was none.
	if strings.Contains(got, "you chose") {
		t.Errorf("an ungraded reveal names a choice:\n%s", got)
	}
}

// A wrong pick is NAMED, because the miss is the moment the word is learned and
// the learner should not have to work out which of three they had picked.
func TestClozeRevealNamesAWrongPick(t *testing.T) {
	c := newTestCloze()
	if v, ok := c.Grade('1'); !ok || v != Wrong {
		t.Fatalf("Grade('1') = %v, %v; want Wrong, true", v, ok)
	}
	got := c.Reveal()
	if !strings.Contains(got, "you chose") || !strings.Contains(got, "ephemeral") {
		t.Errorf("the reveal does not name the wrong pick:\n%s", got)
	}

	// A CORRECT pick names nothing — "you chose" beside the right answer is noise.
	right := newTestCloze()
	if v, ok := right.Grade('2'); !ok || v != Correct {
		t.Fatalf("Grade('2') = %v, %v; want Correct, true", v, ok)
	}
	if strings.Contains(right.Reveal(), "you chose") {
		t.Errorf("a correct pick was told what it chose:\n%s", right.Reveal())
	}
}

// The mechanical contract comes from optionSet and is asserted here too, because
// Cloze is a SECOND consumer and the embedding is the thing under test.
func TestClozeGradesDigitsAndIgnoresStrays(t *testing.T) {
	c := newTestCloze()
	// A digit past the end is a stray key, not a wrong answer: grading it would
	// demote a word the learner never answered about.
	if _, ok := c.Grade('4'); ok {
		t.Error("a digit past the option set was graded")
	}
	// The session's reserved keys must never grade.
	for _, r := range []rune{' ', '\r', 'd', 'y', 'n'} {
		if _, ok := c.Grade(r); ok {
			t.Errorf("%q graded; it is reserved or unrelated", r)
		}
	}
	// The keys line names the flag too: `?` documented nowhere is `?` nobody
	// presses, and the learner has no other source for it.
	if got := c.Keys(); got != "1-3 = pick the word, ? = bad question" {
		t.Errorf("Keys() = %q, want the digits AND the flag", got)
	}
}

// The log name is a NAME, not a form number (#40 D4a).
func TestClozeFormIsNamed(t *testing.T) {
	if got := newTestCloze().Form(); got != "cloze" {
		t.Errorf("Form() = %q, want %q", got, "cloze")
	}
}

// Cloze does NOT implement Missed. Its options come from #10's band-and-domain
// selection, not from NOAD's axes, so "which axis did you confuse" has no answer
// here — and a form answering it would put a fabricated axis in the log.
func TestClozeReportsNoMissedAxis(t *testing.T) {
	c := newTestCloze()
	if _, ok := any(c).(Missed); ok {
		t.Error("Cloze implements Missed; its options carry no axis, so any answer would be invented")
	}
	if got := missedAxis(c); got != AxisNone {
		t.Errorf("missedAxis(cloze) = %v, want AxisNone", got)
	}
}

// THE THREE SESSION STATES a flag can arrive in (PQ-3).
//
// The gate found this: session.go makes every rune advance once s.Graded, so a
// flag pressed AFTER answering would have silently advanced — and after
// answering, having just read the reveal, is exactly when a learner discovers a
// question is broken.
func TestAFlagIsHeardInEverySessionState(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*Session, Question)
	}{
		{"before answering", func(s *Session, q Question) {}},
		{"after answering", func(s *Session, q Question) {
			*s, _ = Apply(*s, Input{Kind: InputRune, Rune: '1'})
		}},
		{"after revealing", func(s *Session, q Question) {
			*s, _ = Apply(*s, Input{Kind: InputReveal})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestCloze()
			s := NewSession([]Question{c})
			tc.setup(&s, c)

			_, outs := Apply(s, Input{Kind: InputRune, Rune: FlagKey})
			if len(outs) != 1 {
				t.Fatalf("got %d outcomes, want 1", len(outs))
			}
			if outs[0].Kind != OutcomeFlag {
				t.Fatalf("outcome = %v, want OutcomeFlag — the flag was swallowed", outs[0].Kind)
			}
			if outs[0].Word != "sycophantic" {
				t.Errorf("flag names word %q", outs[0].Word)
			}
			// THE OPTIONS ARE THE EVIDENCE. A flag naming none is "something was
			// wrong once" — the deliberate opposite of Missed, which records the
			// axis because the option set will not exist later.
			if len(outs[0].Options) != 3 {
				t.Errorf("flag carried %d options, want the whole set: %v", len(outs[0].Options), outs[0].Options)
			}
			// AND IT SCORES NOTHING. No Record, so nothing reaches Fold and the
			// word neither promotes nor demotes: a broken question is not
			// evidence about the learner.
			if outs[0].Kind == OutcomeRecord {
				t.Error("a flag produced a Record; the ladder would move on a broken question")
			}
		})
	}
}

// A form that cannot flag is UNCHANGED — the any-key-advances rule still holds
// for it, which is the row that catches a fix written as "every rune is a flag".
func TestANonFlaggingFormStillAdvancesOnAnyKey(t *testing.T) {
	ch := NewChoice("sycophantic", "def", []Option{
		{Gloss: "a", Correct: true}, {Gloss: "b"},
	})
	s := NewSession([]Question{ch})
	s, _ = Apply(s, Input{Kind: InputRune, Rune: '1'})

	_, outs := Apply(s, Input{Kind: InputRune, Rune: FlagKey})
	if len(outs) != 1 || outs[0].Kind == OutcomeFlag {
		t.Errorf("a non-flagging form produced %v; only forms with the gesture may flag", outs[0].Kind)
	}
}

// The gesture is asked ABOUT A RUNE and the form keeps no state, so there is no
// one-shot to get wrong.
func TestFlagAnswersAboutTheRune(t *testing.T) {
	c := newTestCloze()
	if opts, ok := c.Flag(FlagKey); !ok || len(opts) != 3 {
		t.Errorf("Flag(FlagKey) = %v, %v; want the option set, true", opts, ok)
	}
	// Asking again gives the same answer: stateless, not one-shot.
	if _, ok := c.Flag(FlagKey); !ok {
		t.Error("the gesture stopped answering; it should be stateless")
	}
	for _, r := range []rune{'1', ' ', 'd', 'y'} {
		if _, ok := c.Flag(r); ok {
			t.Errorf("%q was read as the flag gesture", r)
		}
	}
	// AND THE FLAG KEY IS NEVER AN ANSWER: Grade never sees it, so a graded
	// question cannot have its pick moved by one.
	before := c.chosen
	if _, ok := c.Grade(FlagKey); ok {
		t.Error("the flag key graded as an ANSWER; it is not one")
	}
	if c.chosen != before {
		t.Error("the flag key moved the pick")
	}
}

// THE RE-PICK the close review found: the graded branch used to hand every rune
// to Grade to discover a flag, so a stray digit after answering moved the pick
// and changed what the reveal said.
func TestAStrayDigitAfterAnsweringDoesNotRePick(t *testing.T) {
	c := newTestCloze()
	s := NewSession([]Question{c})
	s, _ = Apply(s, Input{Kind: InputRune, Rune: '1'}) // a wrong pick
	before := c.chosen

	Apply(s, Input{Kind: InputRune, Rune: '3'})
	if c.chosen != before {
		t.Errorf("a stray digit moved the pick from %d to %d after the question was graded",
			before, c.chosen)
	}
}
