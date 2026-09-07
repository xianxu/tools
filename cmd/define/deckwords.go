package main

import "strings"

// deckSpan is one word from the learner's deck, LOCATED in rendered text.
//
// The middle of three layers, and the only new one (#46):
//
//  1. the MATCHER — `wordRuns` + `highlightSpans`, which answer "which byte
//     ranges of this text are deck words", longest phrase winning. No ANSI, no
//     coordinates, no terminal. That layer is untouched here and should stay
//     that way: it is the only one that would travel to another program.
//  2. this, the LOCATOR — where those ranges are ON SCREEN.
//  3. the CONSUMERS — a colour and a click, each a loop over these.
//
// Colour and clicks were two independent walks before, covering different
// surfaces, which is exactly why colour reached a rendered entry and clicks
// reached its headword and NEITHER reached a form's own text — the option lines
// a learner is choosing between.
type deckSpan struct {
	// Line is the index within the text, counting "\n".
	Line int
	// Col and Width are DISPLAY CELLS, the units Region speaks in.
	Col, Width int
	// Word is the deck key, which is what a click plays.
	Word string
	// Text is the span exactly as it appears, which is what a region must be
	// able to claim it covers (#12 BR-14).
	Text string
}

// deckSpans finds every deck word in rendered text, with coordinates.
//
// ESCAPE-AWARE WITHOUT TEACHING `highlightSpans` ABOUT ANSI. Per line it takes
// `visibleIndex`, runs the EXISTING matcher over the PLAIN text, and maps each
// known span's byte offset back through the column table. That is `findVisible`'s
// trick generalised from one needle to the whole vocabulary.
//
// It has to be escape-aware because the text this walks is frequently ALREADY
// COLOURED: a form's reveal embeds an entry that `Render` has highlighted, so
// the string carries "\x1b[1;32m" runs. A walk over raw bytes would tokenise the
// letters of an escape sequence as a word, and would report columns that count
// escape bytes as cells — producing regions that underline a `[1;32m` and clicks
// that land on nothing. Everything downstream would still render.
//
// Returns nil for a nil vocabulary, which is the ONE representation of "nothing
// to highlight" this program has (see withStore).
func deckSpans(rendered string, v Vocabulary) []deckSpan {
	if v == nil || rendered == "" {
		return nil
	}
	var out []deckSpan
	for ln, line := range strings.Split(rendered, "\n") {
		plain, cols := visibleIndex(line)
		if plain == "" {
			continue
		}
		at := 0 // byte offset into plain, tracked as spans are consumed
		for _, sp := range highlightSpans(plain, v) {
			if sp.known && at < len(cols) {
				out = append(out, deckSpan{
					Line:  ln,
					Col:   cols[at],
					Width: visibleCells(sp.text),
					Word:  sp.text,
					Text:  sp.text,
				})
			}
			at += len(sp.text)
		}
	}
	return out
}

// wordRegions turns the walk into click regions.
//
// One place builds a Region from a span, so `#12` BR-14's invariant — a region
// covers the text it claims — is provable once rather than per call site.
func wordRegions(rendered string, v Vocabulary) []Region {
	spans := deckSpans(rendered, v)
	if len(spans) == 0 {
		return nil
	}
	out := make([]Region, 0, len(spans))
	for _, s := range spans {
		out = append(out, Region{
			Kind: RegionWord, Text: s.Text, Word: s.Word,
			Line: s.Line, Col: s.Col, Width: s.Width,
		})
	}
	return out
}

// mergeRegions combines two producers' regions for one piece of text, keeping
// them DISJOINT.
//
// `markClickable` has an unwritten precondition and M2 is the first thing able to
// violate it: it sorts by column and walks ONE cursor left to right, so a second
// region at a column the cursor has already passed can never match — and **every
// later region on that line silently loses its underline**. `RegionAt` compounds
// it by resolving in APPEND ORDER rather than by narrowest span.
//
// Neither was wrong while one producer owned a line. A `Choice` prompt is the
// concrete case that makes it wrong now: `RegionHeadword` sits at line 1 column
// 0 and the headword is a deck word, so both producers emit there.
//
// PRECEDENCE: the existing region wins. A `RegionHeadword` carries the entry's
// LOOKUP KEY, which is more specific than a text match — `define jalapeno`
// renders `jalapeño`, and the headword region plays the word the deck holds
// where a text-matched one would play what is on screen.
func mergeRegions(into, add []Region) []Region {
	if len(add) == 0 {
		return into
	}
	out := append([]Region(nil), into...)
	for _, r := range add {
		if !overlapsAny(out, r) {
			out = append(out, r)
		}
	}
	return out
}

// overlapsAny reports whether r shares a cell with anything already on its line.
func overlapsAny(rs []Region, r Region) bool {
	for _, o := range rs {
		if o.Line != r.Line {
			continue
		}
		if r.Col < o.Col+o.Width && o.Col < r.Col+r.Width {
			return true
		}
	}
	return false
}

// surface is what a piece of text IS, which decides whether colour applies.
//
// THE OPERATOR'S RULE, AS A PROPERTY. The ask was "no colour in the cloze,
// because everything there is new words and colour would clog things up". Read
// narrowly that is a flag on one form. Read as a property it is:
//
//	COLOUR MARKS A DECK WORD INSIDE PROSE, WHERE FINDING ONE IS A DISCOVERY.
//	WHERE THE TEXT *IS* THE DECK, COLOUR MARKS EVERYTHING AND DISTINGUISHES
//	NOTHING.
//
// That answers the board too — every cell there is a deck word by construction —
// and predicts the answer for a surface nobody has written yet.
//
// CLICKS ARE NOT SUBJECT TO IT. A click is per-word and costs nothing when it is
// everywhere; colour is a field the eye reads at once and degrades when it is
// everywhere. Cloze options are the case the operator asked for first: clickable,
// uncoloured.
type surface int

const (
	// surfaceProse is text in which a deck word is worth spotting: a definition,
	// an example, a gloss, a restored sentence.
	surfaceProse surface = iota
	// surfaceDeck is text that IS the deck — a cloze's four options, a board's
	// cells. Clickable, never coloured.
	surfaceDeck
)

// admitsColour is the rule in one place.
func (s surface) admitsColour() bool { return s == surfaceProse }

// surfaceOf classifies a form's own text.
//
// DERIVED FROM THE FORM, not from the write site: there is ONE prompt write site
// serving every form, so the site cannot tell a Choice from a Cloze.
//
// It switches on `Form()` — the NAME, a string — never on the concrete type. A
// type switch on a form is forbidden outright by `#6`'s Done-when, and the name
// is what the log and the README already speak in.
//
// The extent is `docSyncForms`, whose completeness `TestEveryFormIsEnrolled`
// guarantees by parsing `play/*.go`. So a new form cannot arrive unclassified —
// `TestEveryFormHasASurface` is what says so.
func surfaceOf(form string) surface {
	switch form {
	case "cloze":
		// Four words drawn from the banded deck. All four would go green.
		return surfaceDeck
	case "board":
		// Every cell is a deck word by construction.
		return surfaceDeck
	case "meaning":
		// PROSE, and the contrast with cloze is the whole rule. Form 2.3's
		// options are DEFINITIONS — sentences in which a deck word appearing is
		// a genuine discovery, and often the most useful thing on the screen.
		return surfaceProse
	}
	// PROSE IS THE DEFAULT, and deliberately so: a form whose text is the deck is
	// the unusual case and the one whose author will be thinking about colour. A
	// new form that is wrong here shows a learner too much green, which they can
	// see; the opposite default would hide the signal silently.
	return surfaceProse
}
