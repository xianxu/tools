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

type Presentation struct {
	Regions []PresentationRegion
	Text    string
	Spans   []LanguageSpan
}

func (p promptBuilder) presentation() Presentation { return Presentation{Text: p.s, Spans: p.spans} }
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
