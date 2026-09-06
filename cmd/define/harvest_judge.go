package main

import (
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// topicSpread is the fraction of a batch's items that carry DISTINCT domains.
//
// 1.0 means every item is from a different subject field; 1/N means the whole
// batch collapsed onto one. Measured with NO MODEL AT ALL, which is Done-when
// 5's second half and the reason this function exists rather than a judge
// scoring its own batch's variety — the self-oracle problem `atlas/define.md`
// records under "Its own oracle".
//
// **The arithmetic only holds because Domain is closed.** Counting distinct
// values over free model text would let `Medicine`, `medicine` and `med` report
// a spread of three where there is one, and this is precisely the measure
// nothing else can catch being wrong. store.ParseDomain is what makes the count
// mean something; the collapse is asserted by a test.
//
// An empty batch is 0 and not 1: "no items" is not perfect variety.
func topicSpread(domains []store.Domain) float64 {
	if len(domains) == 0 {
		return 0
	}
	seen := map[store.Domain]bool{}
	for _, d := range domains {
		// Through the parse, not the raw value. A caller handing this text that
		// never went through the store would otherwise inflate the count by
		// casing alone — the one failure mode a no-model measure has.
		parsed, _ := store.ParseDomain(string(d))
		seen[parsed] = true
	}
	return float64(len(seen)) / float64(len(domains))
}

// entailTaskName / vetoTaskName name their requests.
const (
	entailTaskName = "stem-entails"
	vetoTaskName   = "distractor-veto"
)

// entailVerdict is the judge's answer about one stem.
type entailVerdict struct {
	// Entails is the whole question: could a reader who did not know the word
	// work it out from the rest of the sentence?
	Entails bool `json:"entails"`
	// Glosses reports whether the sentence DEFINES the word rather than using it.
	//
	// A field of its own because the first real batch showed Entails CAUSES this:
	// the cheapest way to make a sentence entail a word is to define the word in
	// it, and half of 20 items came back as appositive glosses that entailed
	// perfectly. A judge scoring only entailment would have passed every one.
	//
	// Inverted sense from the others — true is a REJECTION — because "does it
	// gloss" is the question a reader asks, and phrasing it as "is it clean"
	// would make the prompt argue against itself.
	Glosses bool `json:"glosses"`
	// Named reports the naming requirement — a real person, place or institution
	// the reader can picture — separately, because a stem can satisfy one and
	// fail the other and a single boolean would hide which.
	Named  bool   `json:"named"`
	Reason string `json:"reason"`
}

// vetoVerdict is the judge's answer about one candidate distractor.
//
// A YES/NO ON A CONCRETE PAIR, which is the whole reason a model is trusted
// here: "would this word also fit this blank" is checkable, while "invent three
// wrong answers" is open generation and is how a distractor ends up also
// correct.
type vetoVerdict struct {
	Fits   bool   `json:"fits"`
	Reason string `json:"reason"`
}

const entailSystem = "You judge whether a sentence teaches a word by using it. " +
	"You are strict: most sentences do not."

// renderEntailPrompt asks whether a stem entails its answer. Pure.
//
// lang is threaded for the reason bandSystem states at length: every
// request-rendering function on a per-language path takes a store.Lang and says
// so in the prompt. A Spanish stem judged by a prompt that never names its
// language is judged against English intuitions.
func renderEntailPrompt(lang store.Lang, word, stem string) llm.Request {
	var b strings.Builder
	if lang == "" {
		lang = store.DefaultLang
	}
	fmt.Fprintf(&b, "## The item\n\nLanguage: `%s`\n\nWord: %s\n\nSentence: %s\n\n", lang, word, stem)
	b.WriteString("## What to judge\n\n")
	// THE MULTIPLE-CHOICE FRAMING, and the second checkpoint is what forced it.
	//
	// The first wording asked whether the word was recoverable "from the rest of
	// the sentence alone", and that bar is unreachable without a gloss: nine of
	// twenty items were rejected with reasons like "any migratory fish name would
	// fit, AND NO DEFINITION IS SUPPLIED" — the judge citing the absence of the
	// very thing the gloss rule forbids. The two requirements contradicted each
	// other for every concrete noun.
	//
	// The learner sees FOUR OPTIONS. The question is never "recover this word from
	// the lexicon", it is "is this the right one of these four" — and whether a
	// specific alternative also fits is exactly what the veto asks, per pair,
	// against the options actually offered. So this judge asks the only thing the
	// veto cannot: does the sentence make the word's MEANING do work, or is it
	// merely a place the word can sit?
	b.WriteString("**entails** — does the sentence make the word's MEANING do work?\n\n" +
		"The learner will see this sentence with the word blanked out, beside three other options, " +
		"so the word does NOT have to be the only one in the language that fits — it has to be the " +
		"one this sentence is ABOUT.\n\n" +
		"NO: \"His ___ behaviour was noted by all\" — the word is decorative; the sentence would " +
		"read the same with almost any adjective, and nothing in it is about flattery.\n" +
		"NO: \"___ decreases prosocial intentions\" — a title the word merely appears in.\n" +
		"YES: \"The Times dismissed Oliver Stone's Putin interviews as ___\" — dismissal, a fawning " +
		"interview, a reprinting Kremlin: the sentence is about the quality the word names, even " +
		"though `obsequious` would also fit.\n" +
		"YES: \"Shipwrights at Chatham laid the ___ of their replica frigate, and the first oak " +
		"frames will be bolted to it by spring\" — frames bolted to it, laid first: the sentence " +
		"is about the thing the word names.\n\n" +
		"A near-synonym also fitting is NOT a reason to answer no. That is the veto's job, and it " +
		"runs against the actual options.\n\n")
	b.WriteString("**glosses** — does the sentence DEFINE the word instead of just using it? " +
		"An appositive (\"the alewife, the small silver herring\"), a relative clause " +
		"(\"a mesa, which is a flat-topped hill\") or a contrast that explains it " +
		"(\"a mesa, too broad to be a butte\") are all YES. A sentence that uses the word the way " +
		"a newspaper would, in front of readers assumed to know it, is NO.\n\n" +
		"Yes means the item is REJECTED: a sentence carrying its own definition tests reading, " +
		"not vocabulary.\n\n")
	b.WriteString("**named** — does the sentence name a real person, place or institution the reader " +
		"can picture? \"A manager\" and \"the company\" are NO. \"The Senate Judiciary Committee\" is YES.\n\n")
	b.WriteString("**reason** — one clause, for a human reading a rejected item.\n")

	schema, err := llm.SchemaFor[entailVerdict]()
	if err != nil {
		schema = nil
	}
	return llm.Request{Task: entailTaskName, System: entailSystem, Prompt: b.String(), Schema: schema}
}

const vetoSystem = "You check whether a wrong answer is actually wrong. " +
	"You are looking for the case where it is not."

// renderVetoPrompt asks whether one candidate would ALSO fit the blank. Pure.
//
// One pair per call, deliberately. A batched "which of these four also fit"
// invites the model to rank rather than to judge, and the failure this exists to
// catch is a single candidate that happens to fit — which a ranking hides by
// putting it second.
func renderVetoPrompt(lang store.Lang, answer, stem, candidate string) llm.Request {
	var b strings.Builder
	if lang == "" {
		lang = store.DefaultLang
	}
	fmt.Fprintf(&b, "## The question\n\nLanguage: `%s`\n\nSentence: %s\n\nThe intended answer is **%s**.\n\n"+
		"A wrong answer being considered: **%s**\n\n", lang, blankOut(stem, answer), answer, candidate)
	b.WriteString("## What to judge\n\n")
	b.WriteString("**fits** — would `" + candidate + "` ALSO make the sentence true and natural? " +
		"Answer yes if a careful reader could defend it as correct, even if it is a worse fit than " +
		"the intended answer. A near-synonym almost always fits: `obsequious` fits a blank meant for " +
		"`sycophantic`.\n\n")
	b.WriteString("Yes means the candidate is REJECTED as a wrong answer, because a question with two " +
		"correct options teaches nothing.\n\n")
	b.WriteString("**reason** — one clause.\n")

	schema, err := llm.SchemaFor[vetoVerdict]()
	if err != nil {
		schema = nil
	}
	return llm.Request{Task: vetoTaskName, System: vetoSystem, Prompt: b.String(), Schema: schema}
}

// blankOut replaces the answer with a blank, case-insensitively, so the veto
// judges the QUESTION rather than the sentence with its answer still in it.
//
// Deliberately simple: this is a prompt-shaping helper, not #12's renderer.
// Leaking the answer to the veto would make every candidate look wrong, and
// #12 owns the real blanking — stem, plural, hyphenation — where it is the
// learner who must not see it.
func blankOut(stem, answer string) string {
	lower, target := strings.ToLower(stem), strings.ToLower(strings.TrimSpace(answer))
	// Through wordIndexIn, the same predicate stemUsesTheWord uses. Two spellings of
	// "where is the word" rendered "The settlement was reached" as
	// "The ___tlement was reached".
	i := wordIndexIn(lower, target)
	if i < 0 {
		return stem
	}
	return stem[:i] + "___" + stem[i+len(target):]
}
