package play

// Choice is form 2.3: show the word, offer up to four definitions, one right.
//
// UP TO four: a young deck supplies two or three, and Keys() names the digits
// that actually work rather than promising 1-4 (D9).
//
// This is a RECOGNITION test, where form 2.1 was a recall test — which is why
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
	// Help is Gloss in English, for a learner whose deck is not English; "" is
	// none. DISPLAY ONLY: Grade, Keys and Reveal never read it, so a question
	// with help is graded exactly as the same question without.
	Help string
}

// Choice is one question. Pointer receivers because it REMEMBERS what was
// picked, which a self-rated form never has to.
type Choice struct {
	optionSet
	word string
	// definition is the whole rendered entry, shown after the answer.
	//
	// The options carry ONE gloss each, which is enough to choose between and
	// not enough to learn from — a learner who just missed a word wants its
	// examples, its other senses and its origin. Form 2.1 revealed the full entry
	// for exactly this reason before #42 deleted it, and a recognition form that
	// revealed less would have taught less than the easier form did.
	definition string
}

// NewChoice takes finished options — glosses already extracted, axes already
// assigned, near-synonyms already excluded. See D5: prose does not cross into
// this package.
func NewChoice(word, definition string, options []Option) *Choice {
	return &Choice{optionSet: newOptionSet(options), word: word, definition: definition}
}

func (c *Choice) Word() string { return c.word }

// Prompt is the word ALONE on the first line, a blank, then the numbered
// options — each followed, once SetHelp has run, by its English under the
// gloss. The deck's line comes first because it is the question; the English
// is help beneath it.
//
// The first line is load-bearing beyond looking tidy: #38 makes a prompt word
// clickable and computes its region as line 0, column 0, width len(word)
// (form 2.1, since retired is its premise). #38 is PARKED, so a change here would break it
// silently in an issue nobody is reading. D1a records the constraint on both
// sides.
func (c *Choice) Prompt() string {
	return c.PromptPresentation().Text
}

// HelpLines is which lines of Prompt are English help, as 0-based indices into
// its "\n"-split lines, continuations included — so the caller can keep deck
// colouring and word clicks off them. nil when no help is set.
func (c *Choice) HelpLines() []int {
	return c.render().helps
}

// render is Prompt and HelpLines from ONE walk. An index computed beside the
// prompt would be a second owner of its layout, and the day the two drifted a
// click on an English line would act on a deck word.
func (c *Choice) render() promptBuilder {
	var p promptBuilder
	p.owned(c.word, Target, false)
	p.text("\n\n")
	for i, o := range c.options {
		p.option(i, o.Gloss)
		if o.Help != "" {
			p.text("\n")
			p.help(indentHelp(o.Help))
		}
		if i < len(c.options)-1 {
			p.text("\n")
		}
	}
	return p
}

// SetHelp gives every option its English, or none of them.
//
// ALL OR NOTHING because a partial set singles options out: one option without
// English among others with it is a difference a learner reads as a hint. A
// slice of the wrong length, or with an empty entry, returns false and changes
// nothing — including help an earlier call set.
//
// COPIED rather than written through the slice NewChoice was handed: the caller
// may still hold it, and a second question built from it would grow help it was
// never given.
func (c *Choice) SetHelp(help []string) bool {
	if len(help) != len(c.options) {
		return false
	}
	for _, h := range help {
		if h == "" {
			return false
		}
	}
	opts := append([]Option(nil), c.options...)
	for i := range opts {
		opts[i].Help = help[i]
	}
	c.options = opts
	return true
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

// indentHelp puts OptionIndent in front of EVERY line of an English help, so it
// hangs under the gloss it translates rather than under the option number.
//
// Every line, where a gloss arrives with only its continuations padded: a
// gloss's first line is placed by optionLine, and a help has no number to sit
// beside. A help line at column 0 that began with a digit would also read as a
// new option — to the learner, and to main's isOptionLine.
//
// Byte-wise, which is safe: '\n' never occurs inside a UTF-8 sequence.
func indentHelp(help string) string {
	pad := spaces(OptionIndent)
	out, start := pad, 0
	for i := 0; i < len(help); i++ {
		if help[i] == '\n' {
			out += help[start:i+1] + pad
			start = i + 1
		}
	}
	return out + help[start:]
}

// promptBuilder is a prompt being written that knows which line it is on, so a
// form records its help lines as it writes them rather than recounting later.
type promptBuilder struct {
	s     string
	spans []LanguageSpan
	line  int // the 0-based line the next byte lands on
	helps []int
}

// text appends question text: anything that is not help.
func (p *promptBuilder) text(s string) {
	p.s += s
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			p.line++
		}
	}
}

// help appends help, and every line it touches is a help line: the one it
// starts on and one more per newline inside it.
func (p *promptBuilder) help(s string) {
	first := p.line
	p.owned(s, English, false)
	for l := first; l <= p.line; l++ {
		p.helps = append(p.helps, l)
	}
}

// PromptPresentation emits the prompt and its language ownership in one walk.
func (c *Choice) PromptPresentation() Presentation { return c.render().presentation() }

// Reveal names the answer AND what they picked, because the miss is the moment
// the word is actually learned. Showing only the right answer leaves the learner
// to work out which of four they had chosen.
func (c *Choice) Reveal() string { return c.RevealPresentation().Text }
func (c *Choice) RevealPresentation() Presentation {
	var p promptBuilder
	if i := c.correctIndex(); i >= 0 {
		p.option(i, c.options[i].Gloss)
	}
	if i := c.wrongPick(); i >= 0 {
		p.text("\n\n")
		p.owned("you chose", English, false)
		p.text("\n")
		p.option(i, c.options[i].Gloss)
	}
	if c.definition != "" {
		p.text("\n\n" + c.definition)
	}
	return p.presentation()
}

// Keys names the digits, and what they mean for THIS form.
func (c *Choice) Keys() string                   { return c.KeysPresentation().Text }
func (c *Choice) KeysPresentation() Presentation { return c.keysPresentation("pick the definition") }

// Form names this form in the log (#40 D4a). `meaning` rather than "2.3",
// because that is what the learner types to reach it.
func (c *Choice) Form() string { return "meaning" }

// MissedAxis is why the option they picked was in the set, or AxisNone.
//
// An OPTIONAL interface (see Missed in session.go) rather than a widening of
// Grade: changing Grade's signature would touch every form for a fact only some
// forms have, and the session would then be carrying a concept form 2.1 had no
// answer for.
func (c *Choice) MissedAxis() Axis {
	i := c.wrongPick()
	if i < 0 {
		return AxisNone
	}
	return c.options[i].Axis
}
