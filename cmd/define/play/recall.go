package play

// Recall is form 2.1: show the word, let the learner rate their own recall, and
// show the definition when they missed it.
//
// The rating comes FIRST and the definition is feedback — `y` moves on without
// ever showing it. This is a recall test, so the learner knows the answer before
// they check; grading used to be refused until a reveal, which cost a keystroke
// on every correct answer (#24).
//
// No question generation, no network — which is what makes it the form a session
// can always fall back to, and the reason #6's Done-when asks for a full session
// with the model seam unavailable.
type Recall struct {
	word       string
	definition string
}

// NewRecall takes the word and its ALREADY-RENDERED definition.
//
// Rendering happens in the caller, deliberately: Render needs RenderOpts and the
// dictionary, both IO-shaped, and pulling them in here would end this package's
// purity for a string the caller already has.
func NewRecall(word, definition string) *Recall {
	return &Recall{word: word, definition: definition}
}

func (r *Recall) Word() string   { return r.word }
func (r *Recall) Prompt() string { return r.word }
func (r *Recall) Reveal() string { return r.definition }

// Grade reads the self-rating.
//
// y/n because they are the keys a hand reaches for on a yes/no question, and
// case-insensitively because a session is typed fast. Anything else returns
// false — a stray key is not a silent wrong answer, which would corrupt the
// schedule for a word the learner never rated.
func (r *Recall) Keys() string { return "y = got it, n = missed it" }

// IsSelfRated: this form's verdict is the LEARNER'S CLAIM. `y` means "I knew
// it" and nothing checked. That is why a correct answer here never earns the
// ladder's two-rung promotion, however quickly it came — see play.SelfRated.
func (r *Recall) IsSelfRated() bool { return true }

func (r *Recall) Grade(k rune) (Verdict, bool) {
	switch k {
	case 'y', 'Y':
		return Correct, true
	case 'n', 'N':
		return Wrong, true
	}
	return Skipped, false
}
