package play

import (
	// strings in a TEST is not a purity violation: puretest reads .Imports, not
	// .TestImports, and the guard is about what the package ships. Hand-rolling
	// Split and Contains here was ARCH-DRY for no gain.
	"strings"
	"testing"
)

// The word must be ALONE on the first line: #38's region math depends on it
// (D1a), and #38 is parked, so nothing over there would notice if this changed.
func TestChoicePromptKeepsTheWordAloneOnLineOne(t *testing.T) {
	c := NewChoice("sycophantic", "", []Option{
		{Gloss: "behaving in an obsequious way", Correct: true},
		{Gloss: "the offence of taking property", Axis: AxisDomain},
		{Gloss: "a fool or simpleton", Axis: AxisRegister},
		{Gloss: "a light meal in the afternoon", Axis: AxisGeneral},
	})
	lines := strings.Split(c.Prompt(), "\n")
	if lines[0] != "sycophantic" {
		t.Errorf("line 0 = %q, want the word alone — #38 draws a region at (0,0) of width len(word)", lines[0])
	}
	if len(lines) < 6 {
		t.Fatalf("prompt has %d lines, want the word, a blank and four options:\n%s", len(lines), c.Prompt())
	}
	if lines[1] != "" {
		t.Errorf("line 1 = %q, want a blank between the word and the options", lines[1])
	}
	for i, want := range []string{"1", "2", "3", "4"} {
		if got := lines[2+i]; len(got) < 2 || got[:1] != want {
			t.Errorf("option line %d = %q, want it to start with %q", i, got, want)
		}
	}
}

// Only 1-4, and nothing the session reserved (question.go:74-77).
func TestChoiceGradesOnlyItsOwnKeys(t *testing.T) {
	opts := []Option{{Gloss: "right", Correct: true}, {Gloss: "wrong", Axis: AxisDomain}}
	for _, tc := range []struct {
		key      rune
		wantOK   bool
		wantVerd Verdict
	}{
		{'1', true, Correct},
		{'2', true, Wrong},
		{'3', false, Skipped}, // only two options exist
		{'4', false, Skipped},
		{'y', false, Skipped}, // form 2.1's keys are not this form's
		{'d', false, Skipped}, // RESERVED: the session drops on this
		{'\r', false, Skipped},
		{' ', false, Skipped},
		{0x03, false, Skipped},
	} {
		c := NewChoice("w", "", opts)
		v, ok := c.Grade(tc.key)
		if ok != tc.wantOK || v != tc.wantVerd {
			t.Errorf("Grade(%q) = (%v, %v), want (%v, %v)", tc.key, v, ok, tc.wantVerd, tc.wantOK)
		}
	}
}

// D7: the recorded thing is the AXIS of the option they picked.
func TestChoiceReportsTheAxisItWasMissedOn(t *testing.T) {
	opts := []Option{
		{Gloss: "right", Correct: true},
		{Gloss: "a Law sense", Axis: AxisDomain},
		{Gloss: "an archaic sense", Axis: AxisRegister},
	}
	c := NewChoice("w", "", opts)
	if c.MissedAxis() != AxisNone {
		t.Errorf("before grading MissedAxis = %v, want AxisNone", c.MissedAxis())
	}
	c.Grade('3')
	if got := c.MissedAxis(); got != AxisRegister {
		t.Errorf("MissedAxis = %v, want AxisRegister — picking the archaic option IS the finding", got)
	}

	// D8: a correct answer carries no axis, so nothing is written for it.
	right := NewChoice("w", "", opts)
	right.Grade('1')
	if got := right.MissedAxis(); got != AxisNone {
		t.Errorf("a correct answer reports %v, want AxisNone — D8: no axis on the log for a right answer", got)
	}
}

// The reveal has to teach, which is the whole point of the sitting.
func TestChoiceRevealNamesTheAnswerAndWhatTheyPicked(t *testing.T) {
	opts := []Option{{Gloss: "the right one", Correct: true}, {Gloss: "the wrong one", Axis: AxisDomain}}
	c := NewChoice("w", "", opts)
	c.Grade('2')
	rev := c.Reveal()
	if !strings.Contains(rev, "the right one") {
		t.Errorf("Reveal = %q, want it to name the correct gloss", rev)
	}
	if !strings.Contains(rev, "the wrong one") {
		t.Errorf("Reveal = %q, want it to show what they picked — that is the learning moment", rev)
	}
}

// OptionIndent and the prefix optionLine actually writes are ONE fact.
//
// main pre-wraps a gloss to this indent (#41), so a prefix that grew without the
// constant growing would clip every option's first line by the difference —
// silently, and only on a terminal narrow enough to matter.
func TestOptionLineStartsAtOptionIndent(t *testing.T) {
	const gloss = "a definition"
	line := optionLine(0, gloss)
	if got := len(line) - len(gloss); got != OptionIndent {
		t.Errorf("optionLine puts %d columns before the gloss, and OptionIndent says %d — "+
			"main wraps to the constant, so the difference is clipped off every option", got, OptionIndent)
	}
	if line[:OptionIndent] != "1  " {
		t.Errorf("prefix = %q, want the option number and two spaces", line[:OptionIndent])
	}
}
