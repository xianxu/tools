package play

import (
	"slices"
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

// THE ENGLISH IS HELP, NOT A DIFFERENT QUESTION (#61). Each option's English
// sits under its gloss, deck line first, and nothing the learner is graded on
// moves.
//
// The widths are the ones main pre-wraps to, which is why a gloss here carries
// its own continuation padding and a help does not: SetHelp indents every help
// line itself. Every expected prompt is spelled out rather than derived.
func TestChoiceHelpLines(t *testing.T) {
	const definition = "efímero: que dura poco"
	for _, tc := range []struct {
		width     int
		glosses   [3]string
		helps     [3]string
		plain     string
		helped    string
		helpLines []int
	}{
		{
			width:   80,
			glosses: [3]string{"que dura poco tiempo; pasajero", "año en que se cosecha la uva de un vino", "plato japonés de arroz, como el 寿司"},
			helps:   [3]string{"lasting a very short time; fleeting", "the year a wine's grapes were harvested", "a Japanese rice dish, such as 寿司"},
			plain: "efímero\n\n" +
				"1  que dura poco tiempo; pasajero\n" +
				"2  año en que se cosecha la uva de un vino\n" +
				"3  plato japonés de arroz, como el 寿司",
			helped: "efímero\n\n" +
				"1  que dura poco tiempo; pasajero\n" +
				"   lasting a very short time; fleeting\n" +
				"2  año en que se cosecha la uva de un vino\n" +
				"   the year a wine's grapes were harvested\n" +
				"3  plato japonés de arroz, como el 寿司\n" +
				"   a Japanese rice dish, such as 寿司",
			helpLines: []int{3, 5, 7},
		},
		{
			width:   40,
			glosses: [3]string{"que dura poco tiempo; pasajero", "año en que se cosecha la uva de un\n   vino", "plato japonés de arroz, como el 寿司"},
			helps:   [3]string{"lasting a very short time; fleeting", "the year a wine's grapes were\nharvested", "a Japanese rice dish, such as 寿司"},
			plain: "efímero\n\n" +
				"1  que dura poco tiempo; pasajero\n" +
				"2  año en que se cosecha la uva de un\n" +
				"   vino\n" +
				"3  plato japonés de arroz, como el 寿司",
			helped: "efímero\n\n" +
				"1  que dura poco tiempo; pasajero\n" +
				"   lasting a very short time; fleeting\n" +
				"2  año en que se cosecha la uva de un\n" +
				"   vino\n" +
				"   the year a wine's grapes were\n" +
				"   harvested\n" +
				"3  plato japonés de arroz, como el 寿司\n" +
				"   a Japanese rice dish, such as 寿司",
			helpLines: []int{3, 6, 7, 9},
		},
		{
			width:   20,
			glosses: [3]string{"que dura poco\n   tiempo; pasajero", "año en que se\n   cosecha la uva de\n   un vino", "plato japonés de\n   arroz, como el 寿司"},
			helps:   [3]string{"lasting a very\nshort time;\nfleeting", "the year a wine's\ngrapes were\nharvested", "a Japanese rice\ndish, such as 寿司"},
			plain: "efímero\n\n" +
				"1  que dura poco\n" +
				"   tiempo; pasajero\n" +
				"2  año en que se\n" +
				"   cosecha la uva de\n" +
				"   un vino\n" +
				"3  plato japonés de\n" +
				"   arroz, como el 寿司",
			helped: "efímero\n\n" +
				"1  que dura poco\n" +
				"   tiempo; pasajero\n" +
				"   lasting a very\n" +
				"   short time;\n" +
				"   fleeting\n" +
				"2  año en que se\n" +
				"   cosecha la uva de\n" +
				"   un vino\n" +
				"   the year a wine's\n" +
				"   grapes were\n" +
				"   harvested\n" +
				"3  plato japonés de\n" +
				"   arroz, como el 寿司\n" +
				"   a Japanese rice\n" +
				"   dish, such as 寿司",
			helpLines: []int{4, 5, 6, 10, 11, 12, 15, 16},
		},
	} {
		options := func() []Option {
			return []Option{
				{Gloss: tc.glosses[0], Word: "efímero", Correct: true},
				{Gloss: tc.glosses[1], Word: "añada", Axis: AxisDomain},
				{Gloss: tc.glosses[2], Word: "sushi", Axis: AxisRegister},
			}
		}
		control := NewChoice("efímero", definition, options())
		shared := options()
		c := NewChoice("efímero", definition, shared)

		// NO HELP IS TODAY, byte for byte.
		if got := c.Prompt(); got != tc.plain || got != control.Prompt() {
			t.Errorf("width %d: without help Prompt =\n%s\nwant\n%s", tc.width, got, tc.plain)
		}
		if got := c.HelpLines(); len(got) != 0 {
			t.Errorf("width %d: HelpLines = %v without help, want none", tc.width, got)
		}

		// A PARTIAL SET IS REFUSED WHOLE: one option without English among
		// others with it is a difference a learner reads as a hint.
		for _, bad := range [][]string{
			nil,
			tc.helps[:2],
			{tc.helps[0], "", tc.helps[2]},
			{tc.helps[0], tc.helps[1], tc.helps[2], "and one more"},
		} {
			if c.SetHelp(bad) {
				t.Errorf("width %d: SetHelp(%q) applied; a partial or padded set must be refused", tc.width, bad)
			}
			if got := c.Prompt(); got != tc.plain || len(c.HelpLines()) != 0 {
				t.Errorf("width %d: a refused SetHelp(%q) changed the prompt to\n%s", tc.width, bad, got)
			}
		}

		if !c.SetHelp(tc.helps[:]) {
			t.Fatalf("width %d: SetHelp refused a complete set", tc.width)
		}
		if got := c.Prompt(); got != tc.helped {
			t.Errorf("width %d: with help Prompt =\n%s\nwant\n%s", tc.width, got, tc.helped)
		}
		if got := c.HelpLines(); !slices.Equal(got, tc.helpLines) {
			t.Errorf("width %d: HelpLines = %v, want %v", tc.width, got, tc.helpLines)
		}
		// The indent SetHelp adds is the one main wrapped the English for, so no
		// line is wider than the terminal it was wrapped to.
		for i, line := range strings.Split(c.Prompt(), "\n") {
			if n := columnsIn(line); n > tc.width {
				t.Errorf("width %d: line %d is %d columns: %q", tc.width, i, n, line)
			}
		}
		// "Changes nothing" includes help already set.
		if c.SetHelp([]string{"one"}) || c.Prompt() != tc.helped {
			t.Errorf("width %d: a refused SetHelp disturbed the help already set", tc.width)
		}
		// COPIED: the slice NewChoice was handed is still the caller's.
		for i, o := range shared {
			if o.Help != "" {
				t.Errorf("width %d: SetHelp wrote %q through into the caller's option %d", tc.width, o.Help, i)
			}
		}

		// NOTHING GRADED MOVED: order, keys, verdicts and the reveal.
		for i, o := range c.Options() {
			want := control.Options()[i]
			if o.Gloss != want.Gloss || o.Word != want.Word || o.Axis != want.Axis || o.Correct != want.Correct || o.Help != tc.helps[i] {
				t.Errorf("width %d: option %d is %+v with help, want %+v plus help %q", tc.width, i, o, want, tc.helps[i])
			}
		}
		if got := c.Keys(); got != "1-3 = pick the definition" {
			t.Errorf("width %d: Keys() = %q with help", tc.width, got)
		}
		for _, k := range []rune{'1', '2', '3', '4', 'd', ' '} {
			helped, plain := NewChoice("efímero", definition, options()), NewChoice("efímero", definition, options())
			helped.SetHelp(tc.helps[:])
			hv, hok := helped.Grade(k)
			pv, pok := plain.Grade(k)
			if hv != pv || hok != pok || helped.Reveal() != plain.Reveal() {
				t.Errorf("width %d: Grade(%q) = (%v, %v) with help and (%v, %v) without; reveals\n%s\n---\n%s",
					tc.width, k, hv, hok, pv, pok, helped.Reveal(), plain.Reveal())
			}
		}
		// ...and the reveal is the deck's language alone.
		c.Grade('2')
		wantReveal := "1  " + tc.glosses[0] + "\n\nyou chose\n2  " + tc.glosses[1] + "\n\n" + definition
		if got := c.Reveal(); got != wantReveal {
			t.Errorf("width %d: Reveal with help =\n%s\nwant\n%s", tc.width, got, wantReveal)
		}
	}
}
