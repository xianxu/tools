package play

// Choice is form 2.3: show the word, offer up to four definitions, one right.
//
// UP TO four: a young deck supplies two or three, and Keys() names the digits
// that actually work rather than promising 1-4 (D9).
//
// This is a RECOGNITION test, where form 2.1 is a recall test — which is why
// this one's content is unanswerable unseen and therefore lives in Prompt, as
// the Question interface's doc says it must.
//
// NO IMPORTS, which is a constraint this package's purity guard enforces with an
// empty allowlist (purity_test.go). Everything here is byte arithmetic and
// concatenation. The dictionary work — extracting a gloss, reading NOAD's
// labels, excluding near-synonyms — happens in package main and arrives as
// finished Options, exactly as Board takes a finished gloss (board.go).

// Axis is WHY a distractor is in the option set, and it is the whole of the
// error taxonomy this form can produce (#17 M2, reduced).
//
// The reduction is deliberate and measured, not a shortcut. NOAD labels domain
// and register ITSELF, inline at the head of a sense, so both are free and
// neither creates a second defensible answer — a `Law` sense defines something
// else entirely. Near-synonym collapse and connotation need semantics, so they
// need a model, so they belong to #12 and #13 which have a model veto. This form
// has none by design, which is exactly why it may not reach for them.
type Axis int

const (
	// AxisNone is the CORRECT option, and the zero value. A right answer has no
	// axis to record (D8), so silence is what the log gets — and making silence
	// the zero value means a form that forgets to set one cannot invent a finding.
	AxisNone Axis = iota
	AxisGeneral
	AxisDomain
	AxisRegister
	numAxes // sentinel: the guard derives the set from this, never from a list
)

// distractorAxes is the fill PRIORITY, and the order is the measurement.
//
// Domain first because it is the SCARCE one: counted over the committed corpus,
// register labels outnumber domain roughly five to one, so a set that fills
// register first would almost never leave a domain candidate unused. General
// last because it always succeeds — it is the fallback, and a fallback that
// competes for slots stops being one.
var distractorAxes = []Axis{AxisDomain, AxisRegister, AxisGeneral}

// String is what reaches the event log. Empty for AxisNone so `omitempty` drops
// it (D8): a right answer must not put a word in the log the taxonomy then has
// to filter back out.
func (a Axis) String() string {
	switch a {
	case AxisGeneral:
		return "general"
	case AxisDomain:
		return "domain"
	case AxisRegister:
		return "register"
	}
	return ""
}

// Option is one answer line: the definition shown, and why it is in the set.
//
// Word is the headword the gloss came from. It is NOT what gets recorded — D7
// records the Axis, because "picked the Law one" is a finding a later reader can
// interpret while "picked larceny" is a fact about one question whose option set
// no longer exists. Word is here for the reveal, which names what they picked.
type Option struct {
	Gloss   string
	Word    string
	Axis    Axis
	Correct bool
}

// Choice is one question. Pointer receivers because it REMEMBERS what was
// picked, which a self-rated form never has to.
type Choice struct {
	word    string
	options []Option
	// definition is the whole rendered entry, shown after the answer.
	//
	// The options carry ONE gloss each, which is enough to choose between and
	// not enough to learn from — a learner who just missed a word wants its
	// examples, its other senses and its origin. Form 2.1 revealed the full entry
	// for exactly this reason before #42 deleted it, and a recognition form that
	// revealed less would have taught less than the easier form did.
	definition string
	chosen     int // -1 until graded
}

// NewChoice takes finished options — glosses already extracted, axes already
// assigned, near-synonyms already excluded. See D5: prose does not cross into
// this package.
func NewChoice(word, definition string, options []Option) *Choice {
	return &Choice{word: word, definition: definition, options: options, chosen: -1}
}

// Options is the option set, in the order the learner sees it.
//
// Exported for two consumers that both need to know WHERE an option is on
// screen: tests, which cannot know which digit is correct once the set is
// shuffled, and #38, which will mark the option lines clickable.
func (c *Choice) Options() []Option { return c.options }

func (c *Choice) Word() string { return c.word }

// Prompt is the word ALONE on the first line, a blank, then the numbered
// options.
//
// The first line is load-bearing beyond looking tidy: #38 makes a prompt word
// clickable and computes its region as line 0, column 0, width len(word)
// (recall.go:29 is its premise). #38 is PARKED, so a change here would break it
// silently in an issue nobody is reading. D1a records the constraint on both
// sides.
func (c *Choice) Prompt() string {
	s := c.word + "\n\n"
	for i, o := range c.options {
		s += optionLine(i, o.Gloss)
		if i < len(c.options)-1 {
			s += "\n"
		}
	}
	return s
}

// OptionIndent is how many columns optionLine puts in front of a gloss.
//
// EXPORTED because the caller PRE-WRAPS the gloss and therefore has to know what
// this package will prepend to its first line. That was the alternative #7 named
// — *"the honest fix is to wrap in main and pass pre-wrapped option text, the
// same way the definition already arrives pre-rendered"* — and #41 forced it:
// the terminal used to wrap a long option line, badly but visibly, and a frame
// CLIPS instead, because a line that wraps is a frame one row too tall and the
// terminal then scrolls every placed row (screen.go's whole budget). So an
// unwrapped gloss stopped being ugly and started being missing.
//
// A constant rather than a 3 in main: the prefix's width is this form's fact,
// and two owners of it would drift the day a form numbers past nine.
// TestOptionLineStartsAtOptionIndent is the pin.
const OptionIndent = 3

// optionLine numbers one option. `byte('0'+n)` rather than fmt: this package
// imports nothing, and one digit does not need a formatter (D5a).
//
// The gloss arrives already wrapped to OptionIndent, so its continuation lines
// carry their own padding and this only has to place the number.
func optionLine(i int, gloss string) string {
	return string(rune('0'+i+1)) + "  " + gloss
}

// Reveal names the answer AND what they picked, because the miss is the moment
// the word is actually learned. Showing only the right answer leaves the learner
// to work out which of four they had chosen.
func (c *Choice) Reveal() string {
	var s string
	for i, o := range c.options {
		if o.Correct {
			s = optionLine(i, o.Gloss)
			break
		}
	}
	if c.chosen >= 0 && c.chosen < len(c.options) && !c.options[c.chosen].Correct {
		// The label gets its OWN line, and that is a consequence of the gloss
		// arriving pre-wrapped: "you chose " in front of it would push the first
		// line ten columns past the width it was wrapped to, and a frame clips
		// what does not fit. Wrapping every option ten columns narrower to buy
		// room for one line in one state is the worse trade.
		s += "\n\nyou chose\n" + optionLine(c.chosen, c.options[c.chosen].Gloss)
	}
	if c.definition != "" {
		s += "\n\n" + c.definition
	}
	return s
}

// Keys names the digits that actually work, which on a young deck is fewer than
// four (D9). Telling a learner "1-4" beside a two-option question invites a
// keystroke that does nothing.
func (c *Choice) Keys() string {
	// No branch for fewer than two options: choiceFor refuses below two, so
	// such a Choice is not constructible through production. One built by hand
	// gets "1-1", which is honest about what this form would actually grade.
	return "1-" + string(rune('0'+len(c.options))) + " = pick the definition"
}

// Form names this form in the log (#40 D4a). `meaning` rather than "2.3",
// because that is what the learner types to reach it.
func (c *Choice) Form() string { return "meaning" }

// Grade reads 1-4 and nothing else.
//
// A digit past the end of the option set returns false rather than a verdict:
// D9 allows a two-option question on a young deck, and pressing `3` there is a
// stray key, not a wrong answer. Grading it would demote a word the learner
// never actually answered about.
//
// The session RESERVES Enter, space, `d` and Ctrl-C (question.go:74-77), and
// digits collide with none of them.
func (c *Choice) Grade(k rune) (Verdict, bool) {
	i := int(k - '1')
	if i < 0 || i >= len(c.options) {
		return Skipped, false
	}
	c.chosen = i
	if c.options[i].Correct {
		return Correct, true
	}
	return Wrong, true
}

// MissedAxis is why the option they picked was in the set, or AxisNone.
//
// An OPTIONAL interface (see Missed in session.go) rather than a widening of
// Grade: changing Grade's signature would touch every form for a fact only some
// forms have, and the session would then be carrying a concept form 2.1 has no
// answer for.
func (c *Choice) MissedAxis() Axis {
	if c.chosen < 0 || c.chosen >= len(c.options) || c.options[c.chosen].Correct {
		return AxisNone
	}
	return c.options[c.chosen].Axis
}
