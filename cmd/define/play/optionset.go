package play

// optionSet is the numbered, digit-graded machinery both option forms share.
//
// THE CONTRACT LIVES ONCE, and that is the reason this type exists rather than
// two forms each carrying an `options`/`chosen` pair. The rules are subtle, they
// are documented across four paragraphs of Question's doc comment, and none of
// them is obvious from reading an implementation:
//
//   - options are numbered from 1, and a digit past the end is a STRAY KEY
//     rather than a wrong answer — a young deck can offer two options, and
//     grading a `3` there would demote a word the learner never answered about;
//   - Grade is asked BEFORE the reveal, not only after;
//   - the session's reserved keys (Enter, space, `d`, Ctrl-C) are never in the
//     answer set;
//   - Keys names only the digits that actually work, which on a young deck is
//     fewer than four;
//   - the pick is REMEMBERED, because the reveal names what they chose.
//
// A second implementation is where a documented subtlety goes to be forgotten.
// The alternative — two copies, ~30 lines apiece — was considered and rejected
// for exactly that: the second copy would be written by someone reading the
// first, and the comments explaining WHY do not survive that trip.
//
// It carries no prompt and no reveal. Those are where the forms genuinely
// differ, and a shared type that reached into them would be the wrong seam.
type optionSet struct {
	options []Option
	chosen  int // -1 until graded
}

func newOptionSet(options []Option) optionSet {
	return optionSet{options: options, chosen: -1}
}

// Options is the option set, in the order the learner sees it.
//
// Exported for two consumers that both need to know WHERE an option is on
// screen: tests, which cannot know which digit is correct once the set is
// shuffled, and #38, which marks the option lines clickable.
func (o *optionSet) Options() []Option { return o.options }

// Grade reads 1-N and nothing else.
//
// A digit past the end of the option set returns false rather than a verdict:
// D9 allows a two-option question on a young deck, and pressing `3` there is a
// stray key, not a wrong answer. Grading it would demote a word the learner
// never actually answered about.
//
// The session RESERVES Enter, space, `d` and Ctrl-C (question.go:74-77), and
// digits collide with none of them.
func (o *optionSet) Grade(k rune) (Verdict, bool) {
	i := int(k - '1')
	if i < 0 || i >= len(o.options) {
		return Skipped, false
	}
	o.chosen = i
	if o.options[i].Correct {
		return Correct, true
	}
	return Wrong, true
}

// keysFor names the digits that actually work, which on a young deck is fewer
// than four (D9). Telling a learner "1-4" beside a two-option question invites a
// keystroke that does nothing.
//
// The what — "pick the definition", "pick the word" — is the FORM's, because it
// is the only part that differs and the only part a learner reads for meaning.
func (o *optionSet) keysFor(what string) string {
	// No branch for fewer than two options: the builders refuse below two, so
	// such a set is not constructible through production. One built by hand gets
	// "1-1", which is honest about what it would actually grade.
	return "1-" + string(rune('0'+len(o.options))) + " = " + what
}

// correctIndex is where the answer sits, or -1. Both forms' reveals need it and
// neither should walk the slice itself.
func (o *optionSet) correctIndex() int {
	for i, opt := range o.options {
		if opt.Correct {
			return i
		}
	}
	return -1
}

// wrongPick is the index they chose when it was wrong, or -1 — the condition
// both reveals branch on to say "you chose".
func (o *optionSet) wrongPick() int {
	if o.chosen < 0 || o.chosen >= len(o.options) || o.options[o.chosen].Correct {
		return -1
	}
	return o.chosen
}
