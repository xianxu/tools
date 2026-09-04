package store

import "strings"

// The two closed vocabularies a harvested fact is written in: a CEFR band and a
// subject domain.
//
// They live in store rather than beside their producers because the STORE is
// what decides which values may be written to disk, and both are persisted
// forever. A parse that refuses at the boundary is the whole point: everything
// downstream compares bands arithmetically and counts domains, and neither
// survives an open vocabulary.

// Band is a CEFR level.
//
// ONE scale for the word and the learner, which is what makes "the learner's
// band, or one below" arithmetic rather than a mapping nothing reconciles. #17
// assigns the learner a band as free-form prose; ParseBand is what turns that
// into this type, so the two cannot drift into separate spellings (#10 PQ-1).
type Band string

const (
	A1 Band = "A1"
	A2 Band = "A2"
	B1 Band = "B1"
	B2 Band = "B2"
	C1 Band = "C1"
	C2 Band = "C2"
)

// bands is the scale in order, lowest first. Rank and Below derive from it, so
// the ordering has one source and an inserted level cannot be half-adopted.
var bands = []Band{A1, A2, B1, B2, C1, C2}

// Bands returns the scale, lowest first.
func Bands() []Band { return append([]Band(nil), bands...) }

// ParseBand reads a band, and REFUSES anything that is not one of the six.
//
// A model returning "B2+", "intermediate" or "C1-C2" is giving a real answer to
// a badly-posed question, and accepting it would put an unorderable value into
// the comparison every distractor rule depends on. Refuse here, leave the word
// unbanded, and let it be re-asked — an unbanded word costs one call, a
// nonsense band corrupts selection silently.
//
// Case and surrounding space are forgiven because they are transcription noise
// rather than a different answer: "c1 " is the same claim as "C1".
func ParseBand(s string) (Band, bool) {
	up := strings.ToUpper(strings.TrimSpace(s))
	for _, b := range bands {
		if string(b) == up {
			return b, true
		}
	}
	return "", false
}

// Rank is the band's position on the scale, 0 for A1 through 5 for C2, or -1
// for a value that is not a band.
//
// The -1 is deliberate and must be checked: it makes an unparsed band sort below
// A1 rather than silently equal to it, so a comparison written without thinking
// fails loudly instead of pitching every distractor at the floor.
func (b Band) Rank() int {
	for i, x := range bands {
		if x == b {
			return i
		}
	}
	return -1
}

// Below is the band one step down, and reports false at A1 (and for a non-band).
//
// One BELOW rather than above: a distractor the learner does not know is
// unrejectable — they eliminate it by ignorance rather than by knowing it does
// not fit, which teaches nothing. At A1 there is no lower band and the caller
// must fall back to the learner's own band rather than inventing one.
func (b Band) Below() (Band, bool) {
	r := b.Rank()
	if r <= 0 {
		return "", false
	}
	return bands[r-1], true
}

// Domain is the subject field a word belongs to, from a CLOSED set.
//
// Closed because topicSpread — the one measure this issue takes with NO model —
// counts distinct domains over a batch, and an open vocabulary lets "Medicine",
// "medicine" and "med" report a spread of three where there is one. A measure
// that can be inflated by casing is not a check on the model, it is a mirror.
type Domain string

// DomainGeneral is the fallback, and the only value here that is not NOAD's.
//
// It is a real answer, not a failure: most words carry no subject field at all,
// and "general vocabulary" is exactly what the distractor rule offers when a
// word has no register of its own to draw from.
const DomainGeneral Domain = "general"

// noadDomains — NOAD's subject-field labels, capitalized in its own style.
//
// A CLOSED TABLE this repo owns. #35 argued the shape when originLanguages faced
// the same objection: a table restating a fact the CDN owns goes stale, while
// one reading a dictionary's EDITORIAL PROSE does not — the set of field labels
// a dictionary prints is stable, small, and ours to read. A missing row costs a
// word its domain and it degrades to general; it never produces a wrong
// question, which is what makes an incomplete table an acceptable state rather
// than a latent bug.
//
// This is the SINGLE source. cmd/define/glosslabel.go reads NOAD's prose and
// needs the same labels ordered longest-first for prefix matching; it derives
// that ordering from this slice rather than restating the vocabulary, so a label
// added here reaches the scanner without a second edit (ARCH-DRY).
var noadDomains = []Domain{
	"American football", "Psychoanalysis", "Archaeology", "Mathematics",
	"Architecture", "Linguistics", "Philosophy", "Psychology", "Statistics",
	"Astronomy", "Chemistry", "Computing", "Geometry", "Heraldry", "Medicine",
	"Military", "Nautical", "Politics", "Theology", "Anatomy", "Baseball",
	"Biology", "Ecology", "Finance", "Geology", "Grammar", "Cricket", "Physics",
	"Printing", "Botany", "Mining", "Music", "Zoology", "Bridge", "Chess",
	"Sports", "Law",
}

// Domains returns every subject field a word may carry, EXCLUDING general.
//
// Excluding it because the two are asked for in different places: a scanner
// matching NOAD's prose wants the labels a dictionary actually prints, and
// "general" is never one of them — it is what we call the absence of a label.
func Domains() []Domain { return append([]Domain(nil), noadDomains...) }

// ParseDomain reads a domain, refusing anything outside the closed set.
//
// An unrecognised value is NOT an error and NOT a new domain: it becomes
// general, reported by the false return. That is the degradation the table's
// comment promises — a word whose domain we cannot name draws from general
// vocabulary, which is a worse question than a specialist one and a fine one
// nonetheless. Widening the set on the model's say-so is the failure this
// function exists to prevent.
func ParseDomain(s string) (Domain, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return DomainGeneral, false
	}
	for _, d := range noadDomains {
		if strings.EqualFold(string(d), t) {
			return d, true
		}
	}
	if strings.EqualFold(t, string(DomainGeneral)) {
		return DomainGeneral, true
	}
	return DomainGeneral, false
}
