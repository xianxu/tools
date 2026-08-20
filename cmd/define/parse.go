package main

import (
	"regexp"
	"strings"
)

// Entry is one parsed NOAD headword.
//
// NOAD returns flat text with no schema, so parsing is best-effort by nature.
// Raw is retained verbatim so the no-data-loss invariant (invariant_test.go) can
// hold the rendered output against the source, and so --raw can print it.
type Entry struct {
	Headword  string
	Syllables string
	Homograph string   // "1" in "bank 1"; see the Non-goals in the plan
	IPA       string   // the entry-level pronunciation
	HeadExtra []string // head tokens we did not classify — kept, never dropped
	Blocks    []Block
	Sections  []Section
	Raw       string
}

// Block is one part-of-speech run within an entry.
type Block struct {
	POS string
	// IPA is the block's own pronunciation when it differs from the entry's.
	// This is not speculative: record's verb block carries |rəˈkôrd| against the
	// head's |ˈrekərd|. Empty means "same as Entry.IPA".
	IPA    string
	Senses []Sense
}

// Sense is a numbered sense or a • sub-sense.
type Sense struct {
	Number   string
	Sub      bool
	Gloss    string
	Examples []string
}

// Section is a trailing all-caps block such as DERIVATIVES or ORIGIN.
type Section struct {
	Name string
	Text string
}

var posWords = []string{
	"plural noun", "noun", "verb", "adjective", "adverb", "pronoun",
	"preposition", "conjunction", "interjection", "exclamation", "determiner",
	"abbreviation", "prefix", "suffix", "symbol", "contraction",
}

var sectionWords = []string{
	"PHRASAL VERBS", "DERIVATIVES", "PHRASES", "ORIGIN", "USAGE",
}

// isPronunciation implements Rule B: a |…| span delimits a pronunciation iff
// every comma-separated part of it is a single token.
//
// The tempting rule — "contains no ASCII letters" — is wrong: ˈrekərd and baNGk
// are mostly ASCII letters. What actually separates the two uses of | is word
// shape. Pronunciations are single tokens (or a comma-separated list of them);
// examples are prose, and prose has interior spaces.
func isPronunciation(inner string) bool {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return false
	}
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if part == "" || strings.ContainsAny(part, " \t") {
			return false
		}
	}
	return true
}

var pipeSpan = regexp.MustCompile(`\|([^|]*)\|`)

// splitLeadingPronunciation returns the text before the first pronunciation
// span, that span's content, and the text after it. ok is false when the text
// holds no pronunciation span (pipes present but prose-shaped do not count).
func splitLeadingPronunciation(s string) (before, ipa, after string, ok bool) {
	for _, m := range pipeSpan.FindAllStringSubmatchIndex(s, -1) {
		inner := s[m[2]:m[3]]
		if isPronunciation(inner) {
			return s[:m[0]], strings.TrimSpace(inner), s[m[1]:], true
		}
	}
	return s, "", "", false
}

// ParseEntry converts raw NOAD text into an Entry. It never discards input: any
// token it cannot classify is preserved (HeadExtra, or an unnumbered sense), so
// the no-data-loss invariant holds even for shapes nobody sampled.
func ParseEntry(raw string) Entry {
	e := Entry{Raw: raw}

	head, ipa, rest, ok := splitLeadingPronunciation(raw)
	if !ok {
		// No pronunciation at all — keep everything as one unstructured block.
		e.Headword = firstToken(raw)
		e.Blocks = []Block{{Senses: parseSenses(strings.TrimSpace(trimFirstToken(raw)))}}
		return e
	}
	e.IPA = ipa

	gluedPOS := parseHead(&e, head)

	body, sections := splitSections(rest)
	e.Sections = sections
	e.Blocks = parseBlocks(body, gluedPOS, ipa)
	return e
}

// parseHead applies Rule A and returns a part-of-speech glued to the
// syllabification, if any ("rec·ordnoun" → "noun").
func parseHead(e *Entry, head string) string {
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return ""
	}
	e.Headword = fields[0]
	bare := strings.ReplaceAll(e.Headword, "·", "")
	var glued string
	for _, tok := range fields[1:] {
		switch {
		case isDigits(tok):
			e.Homograph = tok
		case strings.ReplaceAll(tok, "·", "") == bare:
			e.Syllables = tok
		default:
			// "rec·ordnoun" — syllabification with a POS welded onto the end.
			if stripped := strings.ReplaceAll(tok, "·", ""); strings.HasPrefix(stripped, bare) {
				if suffix := stripped[len(bare):]; matchPOS(suffix) == suffix && suffix != "" {
					e.Syllables = tok[:len(tok)-len(suffix)]
					glued = suffix
					continue
				}
			}
			e.HeadExtra = append(e.HeadExtra, tok)
		}
	}
	return glued
}

// splitSections peels the trailing all-caps sections off the body.
func splitSections(body string) (string, []Section) {
	type hit struct {
		idx  int
		name string
	}
	var hits []hit
	for _, name := range sectionWords {
		if i := indexToken(body, name); i >= 0 {
			hits = append(hits, hit{i, name})
		}
	}
	if len(hits) == 0 {
		return body, nil
	}
	for i := 1; i < len(hits); i++ { // insertion sort: at most 5 entries
		for j := i; j > 0 && hits[j].idx < hits[j-1].idx; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	var sections []Section
	for i, h := range hits {
		end := len(body)
		if i+1 < len(hits) {
			end = hits[i+1].idx
		}
		text := strings.TrimSpace(body[h.idx+len(h.name) : end])
		sections = append(sections, Section{Name: h.name, Text: text})
	}
	return body[:hits[0].idx], sections
}

// parseBlocks splits the body on part-of-speech tokens. gluedPOS opens block 0
// when the head consumed the first POS (record), and entryIPA lets a block
// report only a pronunciation that actually differs.
func parseBlocks(body, gluedPOS, entryIPA string) []Block {
	type mark struct {
		idx int
		pos string
	}
	var marks []mark
	for i := 0; i < len(body); {
		pos, width := posAt(body, i)
		if pos == "" {
			i++
			continue
		}
		marks = append(marks, mark{i, pos})
		i += width
	}

	newBlock := func(pos, text string) Block {
		b := Block{POS: pos}
		// A block's pronunciation follows its POS, but a grammar label may sit
		// between them: "verb [with object] | rəˈkôrd | 1 set down …". The label
		// is folded back into the sense text so nothing is dropped.
		if before, ipa, after, ok := splitLeadingPronunciation(text); ok && isGrammarLabelOnly(before) {
			if ipa != entryIPA {
				b.IPA = ipa
			}
			text = strings.TrimSpace(before) + " " + after
		}
		b.Senses = parseSenses(strings.TrimSpace(text))
		return b
	}

	var blocks []Block
	lead := body
	if len(marks) > 0 {
		lead = body[:marks[0].idx]
	}
	if strings.TrimSpace(lead) != "" || gluedPOS != "" {
		blocks = append(blocks, newBlock(gluedPOS, lead))
	}
	for i, m := range marks {
		end := len(body)
		if i+1 < len(marks) {
			end = marks[i+1].idx
		}
		blocks = append(blocks, newBlock(m.pos, body[m.idx+len(m.pos):end]))
	}
	return blocks
}

var senseSplit = regexp.MustCompile(`(?:^|\s)(\d+)\s|•`)

// parseSenses splits a block into numbered senses and • sub-senses. Text before
// the first marker becomes an unnumbered sense rather than being dropped.
func parseSenses(text string) []Sense {
	if text == "" {
		return nil
	}
	locs := senseSplit.FindAllStringSubmatchIndex(text, -1)
	if len(locs) == 0 {
		return []Sense{newSense("", false, text)}
	}
	var senses []Sense
	if lead := strings.TrimSpace(text[:locs[0][0]]); lead != "" {
		senses = append(senses, newSense("", false, lead))
	}
	for i, loc := range locs {
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		number := ""
		if loc[2] >= 0 {
			number = text[loc[2]:loc[3]]
		}
		senses = append(senses, newSense(number, number == "", strings.TrimSpace(text[loc[1]:end])))
	}
	return senses
}

// newSense splits a sense into its gloss and examples. Examples follow the first
// ":" and are separated by interior pipes (Rule B guarantees those pipes are not
// pronunciations, since parseBlocks already peeled any leading one off).
func newSense(number string, sub bool, text string) Sense {
	s := Sense{Number: number, Sub: sub}
	if i := strings.Index(text, ":"); i >= 0 {
		s.Gloss = strings.TrimSpace(text[:i])
		for _, ex := range strings.Split(text[i+1:], "|") {
			if ex = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(ex), ".")); ex != "" {
				s.Examples = append(s.Examples, ex)
			}
		}
		return s
	}
	s.Gloss = strings.TrimSpace(text)
	return s
}

// --- small helpers ---------------------------------------------------------

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

var bracketed = regexp.MustCompile(`\[[^\]]*\]`)

// isGrammarLabelOnly reports whether s holds nothing but bracketed grammar
// labels ("[with object]", "[as modifier]") and whitespace — i.e. whether a
// pronunciation right after it still belongs to the block that just opened.
func isGrammarLabelOnly(s string) bool {
	return strings.TrimSpace(bracketed.ReplaceAllString(s, "")) == ""
}

func matchPOS(s string) string {
	for _, p := range posWords {
		if s == p {
			return p
		}
	}
	return ""
}

func isBoundary(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == ']' || b == ')'
}

// posAt reports the part-of-speech token starting at i, if any, plus how far to
// advance. Matching is on whole tokens so "verbatim" never reads as "verb".
func posAt(body string, i int) (string, int) {
	if i > 0 && !isBoundary(body[i-1]) {
		return "", 0
	}
	for _, p := range posWords {
		if strings.HasPrefix(body[i:], p) {
			end := i + len(p)
			if end == len(body) || isBoundary(body[end]) {
				return p, len(p)
			}
		}
	}
	return "", 0
}

func indexToken(s, token string) int {
	for i := 0; i+len(token) <= len(s); i++ {
		if s[i:i+len(token)] != token {
			continue
		}
		if i > 0 && !isBoundary(s[i-1]) {
			continue
		}
		if end := i + len(token); end == len(s) || isBoundary(s[end]) {
			return i
		}
	}
	return -1
}

func firstToken(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

func trimFirstToken(s string) string {
	if i := strings.IndexByte(strings.TrimLeft(s, " "), ' '); i >= 0 {
		return strings.TrimLeft(s, " ")[i:]
	}
	return ""
}
