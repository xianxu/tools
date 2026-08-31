package play

import "testing"

// The axis has to leave the form and reach the loop, or D7's whole argument
// (record the kind, not a boolean) stops at the type that knows it.
func TestApplyCarriesTheMissedAxisOutOnTheRecord(t *testing.T) {
	opts := []Option{
		{Gloss: "right", Correct: true},
		{Gloss: "a Law sense", Word: "larceny", Axis: AxisDomain},
	}
	s := NewSession([]Question{NewChoice("sycophantic", opts)})
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
	s := NewSession([]Question{NewChoice("w", opts)})
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
