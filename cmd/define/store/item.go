package store

import "time"

// WordFacts is what is cached about a word FOREVER: its band and its domain.
//
// One artifact rather than two caches, because they are one judgement about the
// word and they expire together (never). Assigned once and re-read on every
// later run — which is what makes run-to-run model fuzziness acceptable: a
// word's band is a fact about the word, not about the run that asked.
//
// Stored per LANGUAGE, like the deck and unlike the event log. The dividing line
// is DERIVATION (see yaml.go): `red`, `once`, `actual` and `sensible` are real
// words in both English and Spanish with different bands and unrelated
// meanings, and because these are cached forever and replaced rather than
// merged, a collision would be permanent.
type WordFacts struct {
	Band   Band      `yaml:"band"`
	Domain Domain    `yaml:"domain"`
	At     time.Time `yaml:"at"`
}

// Harvested reports whether these facts were ever assigned.
//
// The timestamp is the signal, so absence needs no second return value — unlike
// NewsItems, which times items it does not timestamp itself. A word that has
// never been harvested and a facts file too damaged to parse read the same way
// here ON PURPOSE: both are worth exactly one re-ask, and a half-trusted record
// is worth less than none.
func (f WordFacts) Harvested() bool { return !f.At.IsZero() }

// Form is which kind of question an item was authored for.
//
// It exists so ONE store holds every form's material rather than a second store
// appearing beside this one when #13's free-sentence form arrives. A word may
// hold several items of different forms, and #12 picks among the ones it can
// render.
type Form string

const (
	// FormCloze is #12's blanked sentence: the stem hides the answer.
	FormCloze Form = "cloze"
	// FormSentence is #13's free written sentence, graded by the model. The stem
	// is a prompt rather than a sentence with a hole in it.
	FormSentence Form = "sentence"
)

// Item is one finished practice item, authored offline and stored complete.
//
// COMPLETE is the point. A review sitting must stay instant, free and offline,
// so everything a question needs — the stem, the answer, and the distractors
// already selected and already vetoed — is decided here, ahead of time. Nothing
// downstream reaches for a model to render one.
//
// The item is also the ONLY evidence about itself. The Spec gave up provenance
// deliberately: a model-authored stem has no source URL to inspect when the pool
// goes bad, so when material reads wrong there is nothing to check but the text
// in this struct. That raises the bar on the judges that write it.
type Item struct {
	Word        string    `yaml:"word"`
	Form        Form      `yaml:"form"`
	Stem        string    `yaml:"stem"`
	Answer      string    `yaml:"answer"`
	Distractors []string  `yaml:"distractors,omitempty"`
	At          time.Time `yaml:"at"`
}
