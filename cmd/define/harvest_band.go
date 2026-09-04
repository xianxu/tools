package main

import (
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// bandTaskName names this request: the golden file, the cassette key, the usage
// label. Part of the contract rather than a label.
const bandTaskName = "word-band"

// bandClaim is the typed answer: what level a word sits at, and what subject
// field it belongs to.
//
// ONE call for both, because they are one judgement about the word and two
// would double the cost of the only per-word work this issue does. The domain
// half is often already answered by the dictionary before this is ever sent —
// see renderBandPrompt.
type bandClaim struct {
	Band   string `json:"band"`
	Domain string `json:"domain"`
}

// bandSystem names no language, because the facts this task produces are stored
// PER-LANGUAGE and a Spanish working directory is a shipped path — the deck,
// the language setting and the learner model are all scoped to it (#23). An
// English-asserting prompt writing a Spanish word's facts would be wrong FOREVER — the cache is never re-examined, and
// the plan's own word for undoing a bad forever-cache is "a migration". The
// language travels in the prompt body instead, where the word is.
//
// CEFR is a Council of Europe framework defined for many languages, so the scale
// itself needs no per-language wording.
const bandSystem = "You place vocabulary on the CEFR scale. " +
	"Answer about the WORD as a learner would meet it, not about the rarest sense a dictionary records."

// renderBandPrompt builds the request for one word. Pure — a word in, an
// llm.Request out — so the golden is what the transport actually sends.
//
// lang is threaded rather than assumed: see bandSystem.
//
// knownDomain is the dictionary's answer when it had one. NOAD prints a subject
// field on specialist senses, readGloss already extracts it, and re-asking a
// model to derive a fact the dictionary printed is a call this issue should not
// make: it costs money, it is slower, and it is LESS reliable than the
// editorial label it would be second-guessing. When it is set, the prompt says
// so and asks only for the band.
func renderBandPrompt(lang store.Lang, word, gloss string, knownDomain store.Domain) llm.Request {
	var b strings.Builder
	if lang == "" {
		lang = store.DefaultLang
	}

	fmt.Fprintf(&b, "## The word\n\n%s\n\nIt is a word of the language with IETF code `%s`, "+
		"and the band must be that language's CEFR scale.\n\n", word, lang)
	if strings.TrimSpace(gloss) != "" {
		fmt.Fprintf(&b, "Its dictionary sense:\n\n%s\n\n", strings.TrimSpace(gloss))
	}

	b.WriteString("## What to answer\n\n")
	b.WriteString("**band** — the CEFR level at which a learner would be expected to KNOW this word: " +
		"one of A1, A2, B1, B2, C1, C2, and nothing else. Not a range, not a `+`, not a word like " +
		"\"intermediate\". Judge the word's ordinary current usage.\n\n")

	if knownDomain != "" {
		// Stated rather than omitted: the model reads better with the domain in
		// front of it as CONTEXT for the band — a Law word is banded differently
		// from a general one — and telling it the answer is already known stops
		// it spending the field on a guess.
		fmt.Fprintf(&b, "**domain** — already known from the dictionary: `%s`. Repeat it exactly.\n\n", knownDomain)
	} else {
		b.WriteString("**domain** — the subject field this word belongs to, from EXACTLY this list:\n\n")
		for _, d := range store.Domains() {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		b.WriteString("\nIf the word is ordinary vocabulary rather than a specialist term of one of those " +
			"fields, answer `general`. `general` is the common and correct answer — do not reach for a " +
			"field a word merely touches.\n\n")
	}

	// The SCHEMA travels with the request, for the reason renderReflectPrompt
	// gives: the golden exists so a field added without thought shows up in its
	// diff, and the shape is what the model is actually constrained by.
	schema, err := llm.SchemaFor[bandClaim]()
	if err != nil {
		// Unreachable for a struct of two strings, and llm.Run computes the real
		// one anyway — this copy only travels so the golden records it.
		schema = nil
	}
	return llm.Request{
		Task:   bandTaskName,
		System: bandSystem,
		Prompt: b.String(),
		Schema: schema,
	}
}

// agreement is the fraction of assignments that match the modal band.
//
// 1.0 is perfect stability; 1/N is noise. THE MODE, not the first answer:
// "what does this model usually say" is the question the cache's premise rests
// on, and anchoring on a single run would measure that run's luck.
//
// WHAT THIS MEASURES, exactly, and the limit is the operator's decision of
// 2026-09-04 recorded on the issue: it is STABILITY, not correctness. A band is
// assigned once and reused forever, so the property the cache depends on is that
// this model gives this word the same band each time — which is what this
// reports. It CANNOT detect a model that is confidently and consistently wrong,
// and every downstream use of a band rests on the scale being right. A uniformly
// skewed scale scores 1.0 here. If distractors ever read as mispitched, this is
// the first thing to suspect and a hand-labelled sample is the thing to build.
//
// Values that are not bands are counted as assignments and can never be the
// mode, so a model answering "B2+" half the time scores 0.5 rather than 1.0 —
// refusals are instability, which is exactly what they are.
func agreement(bands []store.Band) float64 {
	if len(bands) == 0 {
		return 0
	}
	counts := map[store.Band]int{}
	for _, b := range bands {
		// Keyed on the PARSED band, not the raw answer. ParseBand documents case
		// and surrounding space as transcription noise rather than a different
		// answer, so counting the raw string would score a perfectly stable model
		// that varied its casing at 0.67 — below the conformance floor whose
		// prescribed remedy is the expensive hand-labelled sample this issue
		// defers. Two spellings of one fact, one canonical and one not (ARCH-DRY).
		if parsed, ok := store.ParseBand(string(b)); ok {
			counts[parsed]++
		}
	}
	best := 0
	for _, n := range counts {
		if n > best {
			best = n
		}
	}
	return float64(best) / float64(len(bands))
}
