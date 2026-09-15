package play

// LanguageRole describes ownership without importing session language codes.
type LanguageRole uint8

const (
	Neutral LanguageRole = iota
	Target
	English
)

// LanguageSpan addresses exact UTF-8 bytes in Presentation.Text. Uncovered text
// is neutral; AnswerStyled protects semantic answer styles from language tint.
type LanguageSpan struct {
	Start, End   int
	Role         LanguageRole
	AnswerStyled bool
}
type Presentation struct {
	Text  string
	Spans []LanguageSpan
}

func (p promptBuilder) presentation() Presentation { return Presentation{Text: p.s, Spans: p.spans} }
func (p *promptBuilder) owned(s string, role LanguageRole, answerStyled bool) {
	start := len(p.s)
	p.text(s)
	if len(s) > 0 {
		p.spans = append(p.spans, LanguageSpan{start, len(p.s), role, answerStyled})
	}
}
func (p *promptBuilder) option(i int, s string) {
	p.text(optionLine(i, ""))
	p.owned(s, Target, false)
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
