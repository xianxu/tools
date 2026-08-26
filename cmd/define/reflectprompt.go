package main

import (
	"fmt"
	"strings"

	"github.com/xianxu/tools/internal/llm"
)

// reflectTaskName names this request: the golden file, the cassette key, the
// usage label. Part of the contract rather than a label.
const reflectTaskName = "learner-model"

// renderReflectPrompt turns the folded evidence into the request that will be
// sent. Pure — evidence in, an llm.Request out — so the golden is what the
// transport actually sends rather than a string assembled for the test.
//
// The prompt lives here and not in internal/llm: the transport owns the wire,
// the consumer owns the domain (AGENTS.local.md).
func renderReflectPrompt(ev deckEvidence) llm.Request {
	var b strings.Builder

	fmt.Fprintf(&b, "## The deck\n\n%d words, looked up %d times between %s and %s. %d questions asked.\n\n",
		len(ev.Words), ev.Lookups, dateOrNone(ev.From), dateOrNone(ev.To), ev.Questions)
	b.WriteString("| word | times | first | last |\n|---|---|---|---|\n")
	for _, w := range ev.Words {
		fmt.Fprintf(&b, "| %s | %d | %s | %s |\n",
			w.Word, w.Lookups, w.FirstAt.Format("2006-01-02"), w.LastAt.Format("2006-01-02"))
	}

	return llm.Request{Task: reflectTaskName, System: reflectSystem, Prompt: b.String()}
}

// reflectSystem states the two rules the checker then enforces.
//
// Saying them is not guaranteeing them — checkEvidence drops what the deck
// cannot support, because a model asked for evidence will supply plausible
// evidence. But a model told the rule breaks it less often, so not saying them
// is a guaranteed cost where saying them is a free improvement.
//
// The directive is the reason a domain claim is worth generating at all: a share
// tells authoring nothing, and "draw comparables from judicial prose" tells it
// what to do tomorrow.
const reflectSystem = `You are reading one person's vocabulary lookups to describe who they are as a learner.

Two rules, both strict:

1. Cite ONLY words from the list you are given. Every claim you make — the level
   band and each domain — must name the words it was read off, and those words
   must appear in the list. A word that is not in the list is not evidence, and a
   claim you cannot support with listed words is one you must not make.
2. Every domain carries a DIRECTIVE: what authoring practice material for this
   person should do differently because of it. "Law, 42%" is an observation;
   "draw comparables from judicial prose" is usable.

Judge the level from what they reach for, not from how many words they have: a
person looking up rare precise words is not a beginner with a long list. Say the
band plainly (A2 through C2) and say what it was read off.

Name domains the way the person would recognise them — "law", "sailing",
"business news" — not as academic categories. Two or three real ones beat six
speculative. If the deck is genuinely mixed with no pattern, say so with fewer
domains rather than inventing structure.`
