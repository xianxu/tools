package play

// Cloze is form 2.2: the word's own sentence with the word blanked out, and N
// words to choose from.
//
// It is the harder recognition direction. Form 2.3 asks which DEFINITION matches
// a word; this asks which WORD fits a sentence — which is closer to using the
// word, and is why #10 authored a sentence rather than trusting a definition
// match to be the whole test.
//
// Pointer receivers because it remembers what was picked, exactly as Choice
// does.
//
// It takes FINISHED material. The stem was written offline, the distractors were
// selected from the banded deck and vetoed one at a time, and the blanking
// happened in main — this package parses no prose and reaches for nothing (D5,
// D5a). A sitting showing a cloze question makes no model call and no network
// call.
type Cloze struct {
	optionSet
	word string
	// blanked is the stem with the answer hidden — what the learner is asked.
	blanked string
	// restored is the same sentence with the word in it, shown at the reveal.
	//
	// Held rather than re-derived by substituting into `blanked`: the blank
	// consumed an inflected tail (`keels` became `___`, not `___s`), so putting
	// the answer back where the blank is would produce a sentence the author
	// never wrote. The original is the only thing that reads correctly.
	restored string
	// definition is the rendered entry, shown after the answer.
	//
	// choice.go argues this at length and the argument carries over unchanged: a
	// learner who just missed a word wants its other senses, its examples and its
	// origin, and a recognition form revealing less than the retired form 2.1
	// would teach less than the easier form did. The restored sentence is the
	// payload; this is what you read when the payload was not enough.
	definition string
	// flagged is set by Grade when the learner pressed FlagKey, and read once by
	// Flagged. Grade is the only place a keystroke reaches this form.
	flagged bool
}

// NewCloze takes a finished item: the stem already blanked, the same stem
// unblanked, and options already selected, vetoed and ordered.
func NewCloze(word, blanked, restored, definition string, options []Option) *Cloze {
	return &Cloze{
		optionSet:  newOptionSet(options),
		word:       word,
		blanked:    blanked,
		restored:   restored,
		definition: definition,
	}
}

func (c *Cloze) Word() string { return c.word }

// Grade reads the digits through the embedded set, and intercepts FlagKey.
//
// It answers FALSE for the flag, which is the whole trick: false means "this key
// meant nothing to me as an ANSWER", so the session records no verdict — and the
// flag itself travels out through Flagging instead. A form returning a verdict
// here would put "the question was broken" into the ladder.
func (c *Cloze) Grade(k rune) (Verdict, bool) {
	if k == FlagKey {
		c.flagged = true
		return Skipped, false
	}
	return c.optionSet.Grade(k)
}

// Prompt is the blanked sentence, a blank line, then the numbered words.
//
// UNLIKE Choice, the first line is NOT the headword — showing it would answer
// the question. #38's clickable-prompt premise (line 0, column 0, width
// len(word)) is Choice's and does not hold here; a cloze prompt has no headword
// to click, and #38 must not assume one.
func (c *Cloze) Prompt() string {
	s := c.blanked + "\n\n"
	for i, o := range c.options {
		s += optionLine(i, o.Word)
		if i < len(c.options)-1 {
			s += "\n"
		}
	}
	return s
}

// Reveal is the sentence RESTORED — which is the whole point of the form, and
// the reason #10 authored a sentence at all: the learner sees the word doing its
// work in the context it was written for.
//
// Then what they picked, when it was wrong, and then the entry. The three-part
// shape is Choice's, for the reason choice.go gives.
func (c *Cloze) Reveal() string {
	s := c.restored
	if i := c.wrongPick(); i >= 0 {
		s += "\n\nyou chose\n" + optionLine(i, c.options[i].Word)
	}
	if c.definition != "" {
		s += "\n\n" + c.definition
	}
	return s
}

// FlagKey is what the learner presses to say the question itself is broken.
//
// `?` is not reserved by the session (toInput passes it through as an ordinary
// rune) and cannot be confused with a digit. It is mnemonic: this question is
// questionable.
const FlagKey = '?'

// Flagged reports whether the last keystroke called this question broken, and
// returns the options that made it so.
//
// ONE-SHOT: cleared as it is read, because advance asks after every mark and an
// implementation that kept answering would record the same flag repeatedly.
func (c *Cloze) Flagged() ([]string, bool) {
	if !c.flagged {
		return nil, false
	}
	c.flagged = false
	words := make([]string, 0, len(c.options))
	for _, o := range c.options {
		words = append(words, o.Word)
	}
	return words, true
}

// Keys names the digits, and what they mean for this form.
func (c *Cloze) Keys() string { return c.keysFor("pick the word") }

// Form names this form in the log (#40 D4a). `cloze` rather than "2.2": a log
// read years later by a script or a person needs no atlas to decode `cloze`.
func (c *Cloze) Form() string { return "cloze" }
