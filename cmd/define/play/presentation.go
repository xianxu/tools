package play

// LanguageRole describes ownership without importing session language codes.
type LanguageRole uint8

const (
	Neutral LanguageRole = iota
	Target
	// DictionarySource belongs to the selected dictionary, whose language may
	// differ from the deck or be unknown for an all-active fallback search.
	DictionarySource
	English
	// Decoration is producer-owned numbering, key glyphs, and separators.
	Decoration
)

// LanguageSpan addresses exact UTF-8 bytes in Presentation.Text. Uncovered text
// is neutral; AnswerStyled protects semantic answer styles from language tint.
type LanguageSpan struct {
	Start, End   int
	Role         LanguageRole
	AnswerStyled bool
}

// PresentationRegion marks a byte range and whether its rows are tinted. A
// role, not a colour: the shade is main's business, resolved at paint (#70).
type PresentationRegion struct {
	Start, End int
	Tinted     bool
}

// PresentationOption is an option NUMBER a click can answer with (#80): the
// bytes of its `[k] ` prefix in Presentation.Text, and which option it picks.
//
// Recorded by the form as it writes the prompt, because the form is the only
// thing that knows where it drew its numbers — main re-finding them by shape
// would be a second owner of optionLine's layout. Only a PROMPT carries these:
// a reveal repeats option lines, and a click there must answer nothing.
type PresentationOption struct {
	Start, End int
	Index      int
}

type Presentation struct {
	Regions []PresentationRegion
	Text    string
	Spans   []LanguageSpan
	Options []PresentationOption
}

func (p promptBuilder) presentation() Presentation {
	return Presentation{Text: p.s, Spans: p.spans, Options: p.picks}
}
func (p *promptBuilder) owned(s string, role LanguageRole, answerStyled bool) {
	start := len(p.s)
	p.text(s)
	if len(s) > 0 {
		p.spans = append(p.spans, LanguageSpan{start, len(p.s), role, answerStyled})
	}
}
func (p *promptBuilder) option(i int, s string, role LanguageRole) {
	p.text(optionLine(i, ""))
	p.owned(s, role, false)
}

// pickable is option() on a PROMPT: the same line, with its number recorded as
// something a click answers with. The number's span stops before the option's
// text, so a click on the word is never an answer (#80 hazard 3).
func (p *promptBuilder) pickable(i int, s string, role LanguageRole) {
	start := len(p.s)
	p.option(i, s, role)
	p.picks = append(p.picks, PresentationOption{Start: start, End: start + OptionIndent, Index: i})
}
func (p *promptBuilder) blanked(s string) {
	start := 0
	for i := 0; i+3 <= len(s); i++ {
		if s[i:i+3] == "___" {
			p.owned(s[start:i], Target, false)
			p.text("___")
			i += 2
			start = i + 1
		}
	}
	p.owned(s[start:], Target, false)
}

func withDefinitionRegions(p Presentation, definition string, regions []PresentationRegion) Presentation {
	// Definition is appended verbatim as the final part of the reveal.
	offset := len(p.Text) - len(definition)
	for _, r := range regions {
		r.Start += offset
		r.End += offset
		p.Regions = append(p.Regions, r)
	}
	return p
}
