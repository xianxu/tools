package play

import "testing"

// The axis has to leave the form and reach the loop, or D7's whole argument
// (record the kind, not a boolean) stops at the type that knows it.
func TestApplyCarriesTheMissedAxisOutOnTheRecord(t *testing.T) {
	opts := []Option{
		{Gloss: "right", Correct: true},
		{Gloss: "a Law sense", Word: "larceny", Axis: AxisDomain},
	}
	s := NewSession([]Question{NewChoice("sycophantic", "", opts)})
	_, outs := Apply(s, Input{Kind: InputRune, Rune: '2'})

	var rec *Outcome
	for i := range outs {
		if outs[i].Kind == OutcomeRecord {
			rec = &outs[i]
		}
	}
	if rec == nil {
		t.Fatalf("no record outcome: %+v", outs)
	}
	if rec.Verdict != Wrong {
		t.Errorf("verdict = %v, want Wrong", rec.Verdict)
	}
	if rec.Axis != AxisDomain {
		t.Errorf("Axis = %v, want AxisDomain — the finding is WHICH option was picked", rec.Axis)
	}
}

// D8: a right answer writes no axis.
func TestACorrectAnswerCarriesNoAxis(t *testing.T) {
	opts := []Option{{Gloss: "right", Correct: true}, {Gloss: "wrong", Axis: AxisDomain}}
	s := NewSession([]Question{NewChoice("w", "", opts)})
	_, outs := Apply(s, Input{Kind: InputRune, Rune: '1'})
	for _, o := range outs {
		if o.Kind == OutcomeRecord && o.Axis != AxisNone {
			t.Errorf("a correct answer recorded axis %v, want none", o.Axis)
		}
	}
}

// The session stays form-AGNOSTIC: a form that cannot say why it was missed is
// not required to, and Apply must not know which forms those are.
func TestAFormWithNoAxisStillRecords(t *testing.T) {
	s := NewSession([]Question{NewRecall("w", "the definition")})
	_, outs := Apply(s, Input{Kind: InputRune, Rune: 'n'})
	found := false
	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			found = true
			if o.Axis != AxisNone {
				t.Errorf("form 2.1 reported axis %v; it has no way to know one", o.Axis)
			}
		}
	}
	if !found {
		t.Error("form 2.1 stopped recording")
	}
}

// UNAIDED is an observation: the form compared the answer to one it knew, and
// no reveal preceded it.
func TestUnaidedIsSetOnAColdCorrectAnswer(t *testing.T) {
	opts := []Option{{Gloss: "right", Correct: true}, {Gloss: "wrong", Axis: AxisDomain}}
	s := NewSession([]Question{NewChoice("w", "", opts)})
	_, outs := Apply(s, Input{Kind: InputRune, Rune: '1'})

	rec := recordIn(t, outs)
	if !rec.Unaided {
		t.Error("a correct answer given without a reveal is not marked unaided")
	}
}

// THE DISCRIMINATING CASE, and the one the obvious implementation gets wrong.
//
// `advance` sets `s.Revealed = false` BEFORE it builds the Record outcome, so
// computing `unaided` where the outcome is constructed yields true for EVERY
// correct answer — including this one. The feature would look like it worked and
// the ladder would run at double speed. Nothing else here would catch it: the
// cold-answer test above is green on the bug, and so is every wrong-answer test.
func TestARevealDisqualifiesUnaided(t *testing.T) {
	opts := []Option{{Gloss: "right", Correct: true}, {Gloss: "wrong", Axis: AxisDomain}}
	s := NewSession([]Question{NewChoice("w", "", opts)})

	s, _ = Apply(s, Input{Kind: InputReveal})
	if !s.Revealed {
		t.Fatal("fixture is wrong: the reveal did not take")
	}
	_, outs := Apply(s, Input{Kind: InputRune, Rune: '1'})

	rec := recordIn(t, outs)
	if rec.Verdict != Correct {
		t.Fatalf("verdict = %v, want Correct", rec.Verdict)
	}
	if rec.Unaided {
		t.Error("an answer given AFTER the reveal is marked unaided — `unaided` is being " +
			"read after advance() has already zeroed s.Revealed")
	}
}

// A SELF-RATED form can never earn it, however cold the answer.
//
// Form 2.1's `y` means "I knew it" with nobody checking. Granting the two-rung
// promotion for that is the same overconfidence the board is denied — one form
// to the left.
func TestSelfRatedFormsNeverEarnUnaided(t *testing.T) {
	for _, q := range []Question{
		NewRecall("w", "the definition"),
		NewChoice("w", "", []Option{{Gloss: "a", Correct: true}, {Gloss: "b", Axis: AxisGeneral}}),
	} {
		s := NewSession([]Question{q})
		key := 'y'
		if _, ok := q.(*Choice); ok {
			key = '1'
		}
		_, outs := Apply(s, Input{Kind: InputRune, Rune: key})
		rec := recordIn(t, outs)

		_, isSelfRated := q.(SelfRated)
		if isSelfRated && rec.Unaided {
			t.Errorf("%T is self-rated and earned unaided — its verdict is the learner's "+
				"claim, not something the form checked", q)
		}
		if !isSelfRated && !rec.Unaided {
			t.Errorf("%T checked the answer and gave it cold, yet did not earn unaided", q)
		}
	}
}

func recordIn(t *testing.T, outs []Outcome) Outcome {
	t.Helper()
	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			return o
		}
	}
	t.Fatalf("no record outcome: %+v", outs)
	return Outcome{}
}
