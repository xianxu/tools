package main

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// HeadKind classifies a token in an entry's head.
type HeadKind int

const (
	HeadWord HeadKind = iota
	HeadHomograph
	HeadSyllables
	HeadPOS
	HeadOther
)

// HeadTok is one head token, kept in SOURCE ORDER.
//
// The order is carried by the data rather than re-derived by the renderer,
// because NOAD does not use a fixed field order: "present 1 pres·ent" puts the
// homograph before the syllabification, "record rec·ordnoun" welds a
// part-of-speech onto it, and "read verb (past and past participle read | red |)"
// puts a whole parenthetical in the head. A renderer that emits fields in a
// fixed order reorders every entry that disagrees with its guess.
type HeadTok struct {
	Kind HeadKind
	Text string
}

// Entry is one parsed NOAD headword.
//
// NOAD returns flat text with no schema, so parsing is best-effort by nature.
// Raw is retained verbatim so the no-data-loss property (invariant_test.go) can
// hold the rendered output against the source, and so --raw can print it.
type Entry struct {
	Head     []HeadTok
	IPA      string // the entry-level pronunciation
	Blocks   []Block
	Sections []Section
	Raw      string
}

// Accessors derive from Head, so there is exactly one representation of the
// head and no way for the fields and the render order to drift apart.
func (e Entry) headOf(k HeadKind) string {
	for _, t := range e.Head {
		if t.Kind == k {
			return t.Text
		}
	}
	return ""
}

func (e Entry) Headword() string  { return e.headOf(HeadWord) }
func (e Entry) Homograph() string { return e.headOf(HeadHomograph) }
func (e Entry) Syllables() string { return e.headOf(HeadSyllables) }
func (e Entry) HeadPOS() string   { return e.headOf(HeadPOS) }

// Block is one part-of-speech run within an entry.
type Block struct {
	POS string
	// FromHead marks the block whose POS came from the entry head, so Render
	// does not print it twice.
	FromHead bool
	// Label is a grammar label sitting between the POS and its pronunciation
	// ("verb [with object] | rəˈkôrd |"). Held separately so Render can emit it
	// in NOAD's order; folding it into the sense text would reorder the entry.
	Label string
	// IPA is the block's own pronunciation, e.g. record's verb block carries
	// |rəˈkôrd| against the head's |ˈrekərd|. Always stored when present, even
	// when it equals the entry's: suppressing it would silently drop input.
	IPA    string
	Senses []Sense
}

// Sense is a numbered sense or a • sub-sense.
type Sense struct {
	Number   string
	Sub      bool
	Gloss    string
	Examples []Example
}

// Example is one usage example, with any grammar label that introduces it.
//
// NOAD writes "4 the cushion of a pool table: [as modifier] : a bank shot",
// where "[as modifier]" qualifies the example rather than being part of it.
// Splitting only on the first ":" left the label and a stray colon inside the
// quotes on 13.8% of entries. Block.Label already models this concept one level
// up; this is its counterpart.
type Example struct {
	Label string
	Text  string
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

// isPronunciation implements the pipe-disambiguation rule.
//
// Two shapes qualify, and the second one matters more than it looks:
//
//  1. every comma-separated part is a single token — ˈrekərd, baNGk;
//  2. a multi-word span carrying a NOAD stress mark (ˈ or ˌ) —
//     "ˈhät ˌdäɡ" (hot dog), "ˌā prīˈôrī" (a priori), "ˌän bəˈhaf əv".
//
// Requiring single tokens alone rejected every multi-word headword's
// pronunciation. That was not merely cosmetic: when an entry's OWN pronunciation
// was the rejected one, findPronunciation walked past it and adopted a
// derivative's, dumping the whole entry into the head line.
//
// The tempting rule — "contains no ASCII letters" — is wrong in the other
// direction: ˈrekərd and baNGk are mostly ASCII letters.
//
// Prose never carries stress marks, which is what makes rule 2 safe: measured
// over every distinct multi-token span in the live dictionary, it accepts the
// genuine pronunciations and rejects the example sentences.
func isPronunciation(inner string) bool {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return false
	}
	// Sentence punctuation never appears inside a pronunciation.
	if strings.ContainsAny(inner, "[]:;.") {
		return false
	}
	for _, part := range strings.Split(inner, ",") {
		fields := strings.Fields(part)
		switch {
		case len(fields) == 0:
			return false
		case len(fields) == 1:
			continue // rule 1
		case len(fields) > maxPronunciationWords:
			return false
		}
		// Rule 2: at least one word carries a stress mark. Not *every* word —
		// unstressed particles and dialect labels are normal inside a phrase
		// pronunciation ("ət ˈprez(ə)nt", "BrE ˌeɪɡrəˈmatɪk(ə)l"). Prose carries
		// no stress marks at all, so one is enough to separate the two.
		marked := false
		for _, f := range fields {
			if strings.ContainsAny(f, "ˈˌ") {
				marked = true
				break
			}
		}
		if !marked {
			return false
		}
	}
	return true
}

// maxPronunciationWords bounds rule 2. The longest genuine phrase pronunciations
// in NOAD run to six words; beyond that a stress-marked run is not a headword.
const maxPronunciationWords = 8

// findPronunciation returns the text before the first pronunciation span at
// paren depth zero, that span's content, and the text after it.
//
// Depth matters: `read` returns "read verb (past and past participle read
// | red |) [with object] | rēd | …", where the first pronunciation-shaped span
// belongs to a parenthesised inflected form, not to the entry.
func findPronunciation(s string) (before, ipa, after string, ok bool) {
	return findPronunciationIn(s, len(s))
}

// headLimit is where an entry's head can no longer be: the first trailing
// section (DERIVATIVES, ORIGIN, …) or the first part-of-speech word that opens a
// new block after a sentence end.
//
// Without this bound, an entry carrying no pronunciation of its own reaches past
// every block and section boundary and adopts one from DERIVATIVES — "concrete"
// rendered its adjective, noun and verb blocks as one head line and displayed
// the pronunciation of *concreteness*. Bounding the search turns that class of
// failure from a total collapse into a missing IPA, which is the failure mode to
// prefer.
func headLimit(s string) int {
	limit := len(s)
	for _, name := range sectionWords {
		if i := indexToken(s, name); i >= 0 && i < limit {
			limit = i
		}
	}
	for i := 0; i < len(s) && i < limit; i++ {
		if pos, _ := posAt(s, i); pos != "" && afterSentenceEnd(s, i) {
			limit = i
			break
		}
	}
	return limit
}

// afterSentenceEnd reports whether the previous non-space character ends a
// sentence. Unlike opensBlock this does NOT accept ")" or ":" — inside the head
// those still belong to the head ("(subject to) adjective").
func afterSentenceEnd(s string, i int) bool {
	for j := i - 1; j >= 0; j-- {
		switch s[j] {
		case ' ', '\t', '\n':
			continue
		case '.':
			return true
		default:
			return false
		}
	}
	return false
}

func findPronunciationIn(s string, limit int) (before, ipa, after string, ok bool) {
	depth := 0
	for i := 0; i < len(s) && i < limit; i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '|':
			if depth != 0 {
				continue
			}
			j := strings.IndexByte(s[i+1:], '|')
			if j < 0 {
				continue
			}
			if inner := s[i+1 : i+1+j]; isPronunciation(inner) {
				return s[:i], strings.TrimSpace(inner), s[i+1+j+1:], true
			}
		}
	}
	return s, "", "", false
}

// ParseEntry converts raw NOAD text into an Entry. It never discards input: any
// token it cannot classify becomes a HeadOther token or an unnumbered sense, so
// the no-data-loss property holds even for shapes nobody sampled.
func ParseEntry(raw string) Entry {
	e := Entry{Raw: raw}

	head, ipa, rest, ok := findPronunciationIn(raw, headLimit(raw))
	if !ok {
		// No pronunciation to delimit the head, so the head is the maximal prefix
		// of head-SHAPED tokens instead. "iPhone\nA combination mobile phone…"
		// stops at "A"; "concrete con·creteadjective existing…" still splits the
		// welded part-of-speech off the syllabification.
		var body string
		e.Head, body = splitHeadByShape(raw)
		// Sections must be peeled here too. Skipping it left DERIVATIVES inside a
		// block, where its pronunciation pipes were consumed as example
		// separators and surfaced as quoted "-ˈkälik(ə)lē" examples.
		body, e.Sections = splitSections(body)
		e.Blocks = parseBlocks(body, e.HeadPOS(), "")
		return e
	}
	e.IPA = ipa
	e.Head = parseHead(rewritePronunciations(head, "", ""))
	body, sections := splitSections(rest)
	e.Sections = sections
	e.Blocks = parseBlocks(body, e.HeadPOS(), ipa)
	return e
}

// splitHeadByShape consumes head-shaped tokens (headword, homograph,
// syllabification, part-of-speech) from the front of raw and returns them plus
// the remaining body. Used only when no pronunciation delimits the head.
func splitHeadByShape(raw string) ([]HeadTok, string) {
	rest := strings.TrimLeftFunc(raw, unicode.IsSpace)
	word, after := splitFirstToken(rest)
	if word == "" {
		return nil, ""
	}
	head := word
	for {
		tok, remainder := splitFirstToken(after)
		if tok == "" {
			break
		}
		// Reuse the one classifier: a token is head-shaped iff parseHead gives it
		// a kind other than HeadOther.
		toks := parseHead(word + " " + tok)
		if len(toks) < 2 || toks[1].Kind == HeadOther {
			break
		}
		head += " " + tok
		after = remainder
	}
	return parseHead(head), after
}

// parseHead classifies the head tokens, preserving source order. Every token
// lands somewhere; HeadOther is the "kept, never dropped" bucket.
func parseHead(head string) []HeadTok {
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return nil
	}
	toks := []HeadTok{{HeadWord, fields[0]}}
	bare := strings.ReplaceAll(fields[0], "·", "")
	seen := map[HeadKind]bool{}

	// add records a single-valued kind once; a second occurrence is preserved as
	// HeadOther rather than overwriting the first.
	add := func(k HeadKind, text string) {
		if seen[k] {
			k = HeadOther
		}
		seen[k] = true
		toks = append(toks, HeadTok{k, text})
	}

	for i, tok := range fields[1:] {
		stripped := strings.ReplaceAll(tok, "·", "")
		switch {
		// A homograph number sits immediately after the headword ("bank 1",
		// "present 1"). A digit further along is something else — in
		// "use verb 1 [with object]" it is sense 1's number.
		case isDigits(tok) && i == 0:
			add(HeadHomograph, tok)
		// A syllabification always carries interpunct dots. Without that
		// requirement a token merely equal to the headword ("read verb (past and
		// past participle read …") would be classified as one and then hidden.
		case strings.Contains(tok, "·") && stripped == bare:
			add(HeadSyllables, tok)
		case strings.Contains(tok, "·") && strings.HasPrefix(stripped, bare) &&
			matchPOS(stripped[len(bare):]) != "":
			// "rec·ordnoun" — syllabification with a POS welded onto the end.
			suffix := stripped[len(bare):]
			add(HeadSyllables, tok[:len(tok)-len(suffix)])
			add(HeadPOS, suffix)
		case matchPOS(tok) != "":
			add(HeadPOS, tok)
		default:
			toks = append(toks, HeadTok{HeadOther, tok})
		}
	}
	return toks
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
		sections = append(sections, Section{h.name, strings.TrimSpace(body[h.idx+len(h.name) : end])})
	}
	return body[:hits[0].idx], sections
}

// parseBlocks splits the body on part-of-speech tokens. gluedPOS opens block 0
// when the head already carried a part-of-speech.
func parseBlocks(body, gluedPOS, entryIPA string) []Block {
	type mark struct {
		idx int
		pos string
	}
	// A part-of-speech WORD is not a part-of-speech BLOCK. NOAD's prose is full
	// of them — "a noun phrase functioning as", "[with adjective or noun
	// modifier]", "(banked as adjective)" — and treating any of them as a block
	// opener orphans the senses that follow under a phantom heading.
	//
	// A real opener is structural, not lexical. It sits at the start of the body
	// or just after a sentence end, and never inside a bracket or paren:
	//
	//	sycophantic … | ˌsikəˈfan(t)ik | adjective behaving …   body start
	//	… in one season. noun an ephemeral plant: …             after "."
	//	… attributes. (subject to) adjective [predicative] …    after ")"
	//
	// Three separate bugs (bank, man/thing, subject) were all this one rule
	// missing, so it is expressed once here rather than patched per shape.
	var marks []mark
	paren, bracket := 0, 0
	for i := 0; i < len(body); {
		switch body[i] {
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		}
		pos, width := posAt(body, i)
		if pos == "" || paren != 0 || bracket != 0 || !opensBlock(body, i) {
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
		// is held separately so it still renders in NOAD's order.
		if before, ipa, after, ok := findPronunciation(text); ok && isGrammarLabelOnly(before) {
			b.IPA = ipa
			b.Label = strings.TrimSpace(before)
			text = after
		}
		// Convert any remaining pronunciation spans BEFORE splitting senses.
		// Sense text splits examples on "|", and an inflection list carries
		// pronunciations inline — "verb (parrots, parroting | ˈperədiNG |,
		// parroted | ˈperədəd |)" — whose pipes would otherwise be consumed as
		// example separators, burying the pronunciation inside a quoted example
		// where nothing downstream can recognise it.
		text = rewritePronunciations(text, "", "")
		b.Senses = parseSenses(strings.TrimSpace(text))
		return b
	}

	var blocks []Block
	lead := body
	if len(marks) > 0 {
		lead = body[:marks[0].idx]
	}
	if strings.TrimSpace(lead) != "" || gluedPOS != "" {
		lb := newBlock(gluedPOS, lead)
		lb.FromHead = gluedPOS != ""
		blocks = append(blocks, lb)
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

// parseSenses splits a block into numbered senses and • sub-senses.
//
// A bare numeral in prose ("she ran in the 200 meters", "Dave has run 42
// marathons") looks exactly like a sense number, so a numbered split is accepted
// only when it opens the block or continues the sequence. Rejected numerals stay
// in the surrounding sense text — they are never dropped.
func parseSenses(text string) []Sense {
	if text == "" {
		return nil
	}
	locs := senseSplit.FindAllStringSubmatchIndex(text, -1)
	var accepted [][]int
	// The sequence does not always start at 1: when the head swallowed sense 1's
	// number ("use verb 1 [with object] | yo͞oz |"), the body opens at 2. Anchoring
	// at 1 unconditionally rejected every sense in such a block and merged them
	// into the preceding text.
	want := firstSenseNumber(locs, text)
	for _, loc := range locs {
		if loc[2] < 0 { // a • sub-sense always splits
			accepted = append(accepted, loc)
			continue
		}
		if n, err := atoi(text[loc[2]:loc[3]]); err == nil && n == want {
			accepted = append(accepted, loc)
			want++
		}
	}
	if len(accepted) == 0 {
		return []Sense{newSense("", false, text)}
	}
	var senses []Sense
	if lead := strings.TrimSpace(text[:accepted[0][0]]); lead != "" {
		senses = append(senses, newSense("", false, lead))
	}
	for i, loc := range accepted {
		end := len(text)
		if i+1 < len(accepted) {
			end = accepted[i+1][0]
		}
		number := ""
		if loc[2] >= 0 {
			number = text[loc[2]:loc[3]]
		}
		senses = append(senses, newSense(number, number == "", strings.TrimSpace(text[loc[1]:end])))
	}
	return senses
}

// firstSenseNumber picks the value the numbered sequence starts at: the first
// numbered candidate when that is 1 or 2, else 1. A larger leading numeral is a
// prose number ("she ran in the 200 meters"), not a sense.
func firstSenseNumber(locs [][]int, text string) int {
	for _, loc := range locs {
		if loc[2] < 0 {
			continue
		}
		if n, err := atoi(text[loc[2]:loc[3]]); err == nil && (n == 1 || n == 2) {
			return n
		}
		return 1
	}
	return 1
}

// newSense splits a sense into its gloss and examples. Examples follow the first
// ":" and are separated by interior pipes (any leading pronunciation span was
// already peeled off by parseBlocks).
func newSense(number string, sub bool, text string) Sense {
	s := Sense{Number: number, Sub: sub}
	if i := strings.Index(text, ":"); i >= 0 {
		s.Gloss = strings.TrimSpace(text[:i])
		for _, seg := range strings.Split(text[i+1:], "|") {
			if ex, ok := newExample(seg); ok {
				s.Examples = append(s.Examples, ex)
			}
		}
		return s
	}
	s.Gloss = strings.TrimSpace(text)
	return s
}

// newExample peels any leading bracketed grammar label (and the colon that
// follows it) off one example segment.
func newExample(seg string) (Example, bool) {
	seg = strings.TrimSpace(seg)
	var label string
	for {
		seg = strings.TrimSpace(seg)
		if !strings.HasPrefix(seg, "[") {
			break
		}
		end := strings.IndexByte(seg, ']')
		if end < 0 {
			break
		}
		label = strings.TrimSpace(label + " " + seg[:end+1])
		seg = strings.TrimSpace(seg[end+1:])
		seg = strings.TrimSpace(strings.TrimPrefix(seg, ":"))
	}
	seg = strings.TrimRight(seg, ".")
	if seg == "" && label == "" {
		return Example{}, false
	}
	return Example{Label: label, Text: seg}, true
}

// --- small helpers ---------------------------------------------------------

var pipeSpanRe = regexp.MustCompile(`\|([^|]*)\|`)

// rewritePronunciations turns NOAD's |ˌsikəˈfan(t)ək(ə)lē| spans into /…/ so the
// whole entry uses one notation — the Google-style /…/ the tool exists to show.
// Spans that are prose (example separators) are left alone; isPronunciation is
// what decides. Only punctuation changes, so this is no-data-loss safe.
//
// Applied at BOTH ends: in the head at parse time (read carries "(past and past
// participle read | red |)") and to labels, glosses, examples and sections at
// render time. One function, two call sites, so the two cannot drift.
func rewritePronunciations(s, pre, post string) string {
	var b strings.Builder
	i := 0
	for {
		open := strings.IndexByte(s[i:], '|')
		if open < 0 {
			break
		}
		open += i
		close := strings.IndexByte(s[open+1:], '|')
		if close < 0 {
			break
		}
		close += open + 1

		if inner := s[open+1 : close]; isPronunciation(inner) {
			b.WriteString(s[i:open])
			b.WriteString(pre + "/" + strings.TrimSpace(inner) + "/" + post)
			i = close + 1
			continue
		}
		// Not a pronunciation. Consume only the OPENING pipe and retry from the
		// next one — swallowing both would misalign every later pair. A PHRASES
		// section interleaves example separators with phrase pronunciations
		// ("… | example. in business | ˌin ˈbiznəs | operating …"), so a
		// left-to-right pairing that consumes two pipes at a time straddles the
		// real spans and leaves them unconverted.
		b.WriteString(s[i : open+1])
		i = open + 1
	}
	b.WriteString(s[i:])
	return b.String()
}

var bracketed = regexp.MustCompile(`\[[^\]]*\]`)

// isGrammarLabelOnly reports whether s holds nothing but bracketed grammar
// labels ("[with object]") and whitespace — i.e. whether a pronunciation right
// after it still belongs to the block that just opened.
func isGrammarLabelOnly(s string) bool {
	return strings.TrimSpace(bracketed.ReplaceAllString(s, "")) == ""
}

// splitFirstToken is the single definition of "the first token and the rest".
// Two helpers previously re-derived this boundary and disagreed on whether a
// newline counts, which silently dropped a word from every entry that had no
// pronunciation (iPhone, iPad, MacBook).
func splitFirstToken(s string) (head, rest string) {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}

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

var errNotNumber = errors.New("not a number")

// atoi parses a small non-negative integer. It exists rather than strconv.Atoi
// so an absurdly long digit run cannot be mistaken for a sense number.
func atoi(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errNotNumber
		}
		n = n*10 + int(r-'0')
		if n > 1<<20 {
			return 0, errNotNumber
		}
	}
	if s == "" {
		return 0, errNotNumber
	}
	return n, nil
}

func matchPOS(s string) string {
	for _, p := range posWords {
		if s == p {
			return p
		}
	}
	return ""
}

// isBoundary is the LEADING boundary test for a part-of-speech token: a POS may
// follow a bracket or paren.
func isBoundary(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == ']' || b == ')'
}

// posAt reports the part-of-speech token starting at i, plus how far to advance.
//
// The TRAILING boundary is whitespace-or-end only. Allowing ']' and ')' made
// "adjective)" inside "(banked as adjective)" open a whole new top-level block,
// orphaning the verb senses after it under a phantom heading whose body was a
// lone ")". Real POS tokens followed by a bracket only occur inside ORIGIN /
// PHRASAL VERBS, which splitSections has already peeled off.
//
// That covers POS-then-bracket only. The mirror case — a POS *inside* a bracket,
// "[with adjective or noun modifier]" — has whitespace on both sides and is
// rejected by the bracket-depth guard in parseBlocks, not here.
func posAt(body string, i int) (string, int) {
	if i > 0 && !isBoundary(body[i-1]) {
		return "", 0
	}
	for _, p := range posWords {
		if !strings.HasPrefix(body[i:], p) {
			continue
		}
		end := i + len(p)
		if end == len(body) || body[end] == ' ' || body[end] == '\t' || body[end] == '\n' {
			return p, len(p)
		}
	}
	return "", 0
}

// opensBlock reports whether position i is a structural block boundary: the
// start of the body, or immediately after a sentence end. The closing paren of
// a lead-in group counts — "attributes. (subject to) adjective [predicative]".
func opensBlock(body string, i int) bool {
	for j := i - 1; j >= 0; j-- {
		switch body[j] {
		case ' ', '\t', '\n':
			continue
		case '.', ')', ':', ';':
			return true
		default:
			return false
		}
	}
	return true // only whitespace before it — the body starts here
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
