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
	// Named reports the second requirement — a real person, place or institution
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
func renderEntailPrompt(word, stem string) llm.Request {
	var b strings.Builder
	fmt.Fprintf(&b, "## The item\n\nWord: %s\n\nSentence: %s\n\n", word, stem)
	b.WriteString("## What to judge\n\n")
	b.WriteString("**entails** — with the word removed, could a reader who did NOT know it work out " +
		"which word belongs in the blank, from the rest of the sentence alone? If several unrelated " +
		"words would fit equally well, the answer is no. \"His ___ behaviour was noted by all\" is a NO: " +
		"almost any adjective fits.\n\n")
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
func renderVetoPrompt(answer, stem, candidate string) llm.Request {
	var b strings.Builder
	fmt.Fprintf(&b, "## The question\n\nSentence: %s\n\nThe intended answer is **%s**.\n\n"+
		"A wrong answer being considered: **%s**\n\n", blankOut(stem, answer), answer, candidate)
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
	lower, target := strings.ToLower(stem), strings.ToLower(answer)
	i := strings.Index(lower, target)
	if i < 0 {
		return stem
	}
	return stem[:i] + "___" + stem[i+len(target):]
}
