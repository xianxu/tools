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
	// help is the blanked stem in English, drawn under it BEFORE answering; ""
	// is none. It keeps the blank — main's checker refuses a translation that
	// drops a ___ or names the answer — so it restates the question rather than
	// answering it.
	help string
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

// Blanked is the stem exactly as Prompt shows it: the text a translation of the
// question is made from, since the restored stem carries the answer.
func (c *Cloze) Blanked() string { return c.blanked }

// SetHelp sets the stem's English; "" clears it. One string where Choice takes
// one per option: the stem is a cloze question's only prose, and its options
// are the deck words under test.
func (c *Cloze) SetHelp(english string) { c.help = english }

// Prompt is the blanked sentence, its English under it when SetHelp gave one,
// a blank line, then the numbered words.
//
// UNLIKE Choice, the first line is NOT the headword — showing it would answer
// the question. #38's clickable-prompt premise (line 0, column 0, width
// len(word)) is Choice's and does not hold here; a cloze prompt has no headword
// to click, and #38 must not assume one.
func (c *Cloze) Prompt() string {
	return c.PromptPresentation().Text
}

// HelpLines is which lines of Prompt are English help, as Choice.HelpLines
// defines it. nil when no help is set.
func (c *Cloze) HelpLines() []int {
	return c.render().helps
}

// render is Prompt and HelpLines from one walk, for the reason Choice.render
// gives. The stem may itself span lines, which is exactly the count a separate
// index would get wrong.
func (c *Cloze) render() promptBuilder {
	var p promptBuilder
	p.blanked(c.blanked)
	if c.help != "" {
		p.text("\n")
		p.help(c.help)
	}
	p.text("\n\n")
	for i, o := range c.options {
		p.option(i, o.Word, Target)
		if i < len(c.options)-1 {
			p.text("\n")
		}
	}
	return p
}

// PromptPresentation emits the prompt and its language ownership in one walk.
func (c *Cloze) PromptPresentation() Presentation { return c.render().presentation() }

// Reveal is the sentence RESTORED — which is the whole point of the form, and
// the reason #10 authored a sentence at all: the learner sees the word doing its
// work in the context it was written for.
//
// Then what they picked, when it was wrong, and then the entry. The three-part
// shape is Choice's, for the reason choice.go gives.
func (c *Cloze) Reveal() string { return c.RevealPresentation().Text }
func (c *Cloze) RevealPresentation() Presentation {
	var p promptBuilder
	p.owned(c.restored, Target, false)
	if i := c.wrongPick(); i >= 0 {
		p.text("\n\n")
		p.owned("you chose", English, false)
		p.text("\n")
		p.option(i, c.options[i].Word, Target)
	}
	if c.definition != "" {
		p.text("\n\n" + c.definition)
	}
	return p.presentation()
}

// FlagKey is what the learner presses to say the question itself is broken.
//
// `?` is not reserved by the session (toInput passes it through as an ordinary
// rune) and cannot be confused with a digit. It is mnemonic: this question is
// questionable.
const FlagKey = '?'

// Flag reports whether a keystroke is the bad-question gesture, and returns the
// options that made the question bad.
//
// STATELESS: it answers about the rune it is given rather than remembering one,
// so there is no one-shot to get wrong and Grade never has to see the key.
func (c *Cloze) Flag(k rune) ([]string, bool) {
	if k != FlagKey {
		return nil, false
	}
	words := make([]string, 0, len(c.options))
	for _, o := range c.options {
		words = append(words, o.Word)
	}
	return words, true
}

// Keys names the digits AND the flag, because the keys line is the only place a
// learner is told what a key does. `?` documented nowhere is `?` nobody presses.
func (c *Cloze) Keys() string {
	return c.KeysPresentation().Text
}

// Form names this form in the log (#40 D4a). `cloze` rather than "2.2": a log
// read years later by a script or a person needs no atlas to decode `cloze`.
func (c *Cloze) Form() string { return "cloze" }

func (c *Cloze) KeysPresentation() Presentation {
	base := c.keysPresentation("pick the word")
	p := promptBuilder{s: base.Text, spans: base.Spans}
	p.text(", " + string(FlagKey) + " = ")
	p.owned("bad question", English, false)
	return p.presentation()
}
