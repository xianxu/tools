package play

import "testing"

func rune_(r rune) Input { return Input{Kind: InputRune, Rune: r} }

var reveal = Input{Kind: InputReveal}
var quit = Input{Kind: InputQuit}

// drive runs a whole input sequence and collects what the loop was told to do.
func drive(s Session, ins ...Input) (Session, []Outcome) {
	var out []Outcome
	for _, in := range ins {
		var o Outcome
		s, o = Apply(s, in)
		out = append(out, o)
	}
	return s, out
}

func records(outs []Outcome) []Outcome {
	var rec []Outcome
	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			rec = append(rec, o)
		}
	}
	return rec
}

func twoQuestions() Session {
	return NewSession([]Question{
		NewRecall("obsequious", "fawning"),
		NewRecall("ephemeral", "short-lived"),
	})
}

func TestRevealThenGradeAdvancesAndRecords(t *testing.T) {
	s, outs := drive(twoQuestions(), reveal, rune_('y'))

	rec := records(outs)
	if len(rec) != 1 {
		t.Fatalf("got %d record outcomes, want 1: %+v", len(rec), outs)
	}
	if rec[0].Word != "obsequious" || rec[0].Verdict != Correct {
		t.Errorf("recorded %+v, want obsequious/Correct", rec[0])
	}
	if s.Index != 1 {
		t.Errorf("index = %d, want 1 — the session did not advance", s.Index)
	}
	if s.Revealed {
		t.Error("still revealed on the next question — the answer is showing before it is asked")
	}
	if s.Right != 1 {
		t.Errorf("Right = %d, want 1", s.Right)
	}
}

// A learner cannot rate what they have not seen. Recording here would file a
// verdict about a word still hidden.
func TestGradingBeforeRevealIsIgnored(t *testing.T) {
	s, outs := drive(twoQuestions(), rune_('y'), rune_('n'))

	if rec := records(outs); len(rec) != 0 {
		t.Errorf("recorded %+v before reveal", rec)
	}
	if s.Index != 0 {
		t.Errorf("index = %d, want 0 — a mis-keystroke advanced the session", s.Index)
	}
}

// A skip is not an assessment: schedule.Fold would read a recorded skip as a
// miss and demote a word the learner was honest about.
func TestSkipAdvancesButRecordsNothing(t *testing.T) {
	s, outs := drive(twoQuestions(), reveal, rune_('s'))

	// 's' is not a key Recall grades, so it is ignored entirely rather than
	// skipping — this asserts the CURRENT contract: only y/n advance form 2.1.
	if s.Index != 0 {
		t.Errorf("index = %d, want 0 — an ungraded key advanced the session", s.Index)
	}
	if rec := records(outs); len(rec) != 0 {
		t.Errorf("recorded %+v for a key the form does not use", rec)
	}
}

// The verdict-level rule, driven through a form that DOES produce Skipped.
func TestSkippedVerdictRecordsNothing(t *testing.T) {
	s := NewSession([]Question{&skipForm{}})

	next, outs := drive(s, reveal, rune_('x'))

	if rec := records(outs); len(rec) != 0 {
		t.Errorf("a Skipped verdict was recorded: %+v", rec)
	}
	if next.Index != 1 {
		t.Errorf("index = %d, want 1 — a skip must still advance", next.Index)
	}
	if next.Right != 0 || next.Wrong != 0 {
		t.Errorf("a skip counted in the tally: right %d wrong %d", next.Right, next.Wrong)
	}
}

func TestAnsweringTheLastQuestionEndsTheSession(t *testing.T) {
	s, outs := drive(twoQuestions(), reveal, rune_('y'), reveal, rune_('n'))

	if !s.Done {
		t.Error("session is not done after the last answer")
	}
	if len(records(outs)) != 2 {
		t.Errorf("got %d records, want 2", len(records(outs)))
	}
	if s.Right != 1 || s.Wrong != 1 {
		t.Errorf("tally right %d wrong %d, want 1/1", s.Right, s.Wrong)
	}
	// And nothing after Done does anything.
	after, o := Apply(s, rune_('y'))
	if o.Kind != OutcomeDone || after.Index != s.Index {
		t.Errorf("input after Done changed things: %+v", o)
	}
}

func TestQuitEndsTheSessionAtAnyPoint(t *testing.T) {
	for _, name := range []string{"before reveal", "after reveal"} {
		t.Run(name, func(t *testing.T) {
			s := twoQuestions()
			if name == "after reveal" {
				s, _ = drive(s, reveal)
			}

			next, o := Apply(s, quit)

			if !next.Done {
				t.Error("quit did not end the session")
			}
			if o.Kind != OutcomeDone {
				t.Errorf("outcome = %v, want OutcomeDone", o.Kind)
			}
		})
	}
}

// Revealing is what triggers the pronunciation, and revealing twice must not
// play it twice.
func TestRevealIsIdempotent(t *testing.T) {
	_, outs := drive(twoQuestions(), reveal, reveal)

	if outs[0].Kind != OutcomeReveal {
		t.Errorf("first reveal = %v, want OutcomeReveal", outs[0].Kind)
	}
	if outs[1].Kind != OutcomeNone {
		t.Errorf("second reveal = %v, want OutcomeNone — the word would play twice", outs[1].Kind)
	}
}

// "Nothing due today" is the expected state most days, not an error.
func TestEmptyQueueIsImmediatelyDone(t *testing.T) {
	s := NewSession(nil)

	if !s.Done {
		t.Error("an empty session is not done")
	}
	if s.Current() != nil {
		t.Error("an empty session has a current question")
	}
}

// THE DONE-WHEN: "adding a second form requires no change to the loop".
//
// This is the only honest way to test that before a second form exists. fakeForm
// grades entirely different keys — digits, not y/n — and produces all three
// verdicts. The same session behaviour must fall out. If any key semantics had
// leaked into Apply, this fails.
func TestSessionIsFormAgnostic(t *testing.T) {
	s := NewSession([]Question{&fakeForm{word: "alpha"}, &fakeForm{word: "beta"}})

	next, outs := drive(s, reveal, rune_('1'), reveal, rune_('2'))

	rec := records(outs)
	if len(rec) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(rec), outs)
	}
	if rec[0].Word != "alpha" || rec[0].Verdict != Correct {
		t.Errorf("first record %+v, want alpha/Correct", rec[0])
	}
	if rec[1].Word != "beta" || rec[1].Verdict != Wrong {
		t.Errorf("second record %+v, want beta/Wrong", rec[1])
	}
	if !next.Done || next.Right != 1 || next.Wrong != 1 {
		t.Errorf("done %v right %d wrong %d", next.Done, next.Right, next.Wrong)
	}
	// And y/n — form 2.1's keys — mean NOTHING here, which is the proof that the
	// session never learned them.
	ignored := NewSession([]Question{&fakeForm{word: "alpha"}})
	after, outs2 := drive(ignored, reveal, rune_('y'))
	if len(records(outs2)) != 0 || after.Index != 0 {
		t.Error("'y' graded a form that does not use it — key semantics leaked into the session")
	}
}

// fakeForm uses digits and produces every verdict, sharing no key with Recall.
type fakeForm struct{ word string }

func (f *fakeForm) Word() string   { return f.word }
func (f *fakeForm) Prompt() string { return "which one?" }
func (f *fakeForm) Reveal() string { return "it was the first" }
func (f *fakeForm) Grade(r rune) (Verdict, bool) {
	switch r {
	case '1':
		return Correct, true
	case '2':
		return Wrong, true
	case '3':
		return Skipped, true
	}
	return Skipped, false
}

// skipForm grades one key and always skips, so the Skipped path is reachable.
type skipForm struct{}

func (skipForm) Word() string   { return "skipped" }
func (skipForm) Prompt() string { return "?" }
func (skipForm) Reveal() string { return "!" }
func (skipForm) Grade(r rune) (Verdict, bool) {
	if r == 'x' {
		return Skipped, true
	}
	return Skipped, false
}
