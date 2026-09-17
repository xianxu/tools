package main

import (
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// exchange is one question and the answer it got, within a session.
type exchange struct {
	Question string
	Answer   string
}

// askContext is everything the model is told, as DATA.
//
// A struct rather than a string built at the call site, because it is what makes
// "what did we send" assertable without a socket: the golden renders this, and
// the test that proves the deck and the learner model reached the prompt reads
// the recorded request rather than trusting the code that built it.
//
// Every field is optional except Question. The context is whatever this
// directory happens to hold — a fresh process in a fresh directory answers with
// none of it, which is the property that makes the context inspectable as files
// rather than trapped in a long-running process.
type askContext struct {
	Question string
	Language store.Lang
	// CurrentWord and CurrentEntry are set when the question FOLLOWS a lookup,
	// which is the common case: you look a word up, then ask about it.
	CurrentWord  string
	CurrentEntry string
	// SessionWords are this session's lookups, newest last. DeckWords are the
	// store's, which survive the process.
	SessionWords []string
	DeckWords    []string
	// UserModel is the learner model verbatim (#17 writes it). Passed whole rather
	// than parsed: it is prose for a model to read, and a parser here would be a
	// second definition of a format #17 owns.
	UserModel string
	Turns     []exchange
}

// The section headers, named so the tests can assert a section is OMITTED rather
// than rendered empty — an empty "## The learner" says there IS a model and it
// is blank, which is a different claim and produces a confident generic answer.
const (
	headerCurrent = "## The word on screen"
	headerLearner = "## The learner"
	headerSession = "## Looked up this session"
	headerDeck    = "## Recently in the deck"
	headerEarlier = "## Earlier in this conversation"
)

// maxTurns bounds the transcript carried into a follow-up. Enough for "give me
// three more examples" to resolve against what was just said, short enough that
// a long session does not grow the prompt without limit.
const maxTurns = 20

// askTask names this request. It keys the cassette and names the golden, so it
// is part of the contract rather than a label.
const askTask = "console-question"

// recentTurns keeps the newest maxTurns exchanges, oldest first.
//
// Newest rather than oldest, because a follow-up resolves against what was JUST
// said; order preserved, because "three more examples" only means anything after
// the answer it follows.
func recentTurns(all []exchange) []exchange {
	if len(all) <= maxTurns {
		return all
	}
	return all[len(all)-maxTurns:]
}

// recentDeck picks the n most RECENTLY seen deck words and returns them
// oldest-first, so the section reads forwards.
//
// Pure, and here beside recentTurns rather than in the IO gatherer, because it
// is a POLICY: which words the model is told about. Inlined in gatherAskContext
// it was only reachable through a real store and a wire-level fake, which is
// why nothing caught it taking the wrong end of a newest-first slice and
// labelling the result "Recently in the deck" (BR-22, then BR-41).
//
// Store.Deck() is ordered by LastSeen DESCENDING — the head is the newest.
func recentDeck(deck []store.Word, n int) []string {
	if len(deck) > n {
		deck = deck[:n]
	}
	words := make([]string, 0, len(deck))
	for i := len(deck) - 1; i >= 0; i-- {
		words = append(words, deck[i].Text)
	}
	return words
}

// renderAskPrompt turns the context into the request that will be sent.
//
// Pure: a struct in, an llm.Request out, no IO and no clock. That is what lets
// the golden be the request the transport hashes rather than a string this file
// concatenates for the test's benefit.
//
// The prompt lives HERE and not in internal/llm: the transport owns the wire,
// the consumer owns the domain (AGENTS.local.md).
func renderAskPrompt(c askContext) llm.Request {
	var b strings.Builder
	lang := c.Language
	if lang == "" {
		lang = store.DefaultLang
	}
	b.WriteString("## Study language\n" + string(lang) + "\n\n")
	section := func(header, body string) {
		if strings.TrimSpace(body) == "" {
			return // omitted entirely — see the note on the headers above
		}
		b.WriteString(header)
		b.WriteString("\n")
		b.WriteString(strings.TrimRight(body, "\n"))
		b.WriteString("\n\n")
	}

	if c.CurrentWord != "" {
		section(headerCurrent, c.CurrentWord+"\n\n"+c.CurrentEntry)
	}
	section(headerLearner, c.UserModel)
	section(headerSession, strings.Join(c.SessionWords, ", "))
	section(headerDeck, strings.Join(c.DeckWords, ", "))

	var turns strings.Builder
	for _, e := range recentTurns(c.Turns) {
		turns.WriteString("Q: " + e.Question + "\nA: " + e.Answer + "\n\n")
	}
	section(headerEarlier, turns.String())

	b.WriteString("## The question\n")
	b.WriteString(c.Question)
	b.WriteString("\n")

	return llm.Request{
		Task:   askTask,
		System: askSystem,
		Prompt: b.String(),
	}
}

// The shared clauses. Both system prompts are BUILT from these rather than each
// spelling them out: #67 added a second prompt that restated the level default,
// the dictionary-authority rule and the whole language grammar, so a change to any
// of them had to be made twice and would drift the first time it was not.
const (
	// sharedLevel is the level policy, including the default when no learner
	// model exists. Two settings in OPPOSITE directions, and reading them as one
	// is the mistake: hold the language, drop the assumed background.
	sharedLevel = `The context blocks tell you who is asking. Pitch the answer at the level the
learner model implies.

If the learner model is absent, assume a curious reader who is going to college
but does not have the background yet. That is TWO settings in opposite
directions, and reading it as one is the mistake to avoid. Hold the language:
they are a capable reader, so do not simplify your sentences and never replace a
hard word with an easy one — least of all the word being explained. Drop the
assumed background: do not take a field's terms as known. And be curious: offer
the one connecting fact that makes the answer land, rather than stopping at the
question's edge.`

	// sharedAuthority is who wins when the model and the dictionary disagree.
	sharedAuthority = `Never invent a definition that contradicts a dictionary entry you were given —
it is authoritative and you are not. Say plainly when you are unsure.`

	// sharedLanguageGrammar is the [lang=xx] contract languageDecoder parses. It
	// is a WIRE FORMAT, so a second spelling of it is a second format.
	sharedLanguageGrammar = `Annotate the language of your answer using short nonnested passages:
[lang=es]Spanish text[/lang] and [lang=en]English text[/lang]. Use a two-letter
language code, or und when unknown. Place boundaries at actual language changes,
including an inline phrase in another language. Keep passages below 4000 characters.
These annotations describe the language you write; they do not change which
languages or proportions the question calls for. Untagged prose is neutral.
To discuss these reserved markers literally, escape their brackets as ` + escLeft + ` and
` + escRight + `. Never emit terminal escape sequences or control characters.`
)

// askSystem is the standing instruction for a console question.
var askSystem = strings.Join([]string{
	`You are helping someone build vocabulary in the study language named in the context, inside a dictionary tool.`,
	`Answer the question directly and briefly — a few sentences, not an essay. Prefer
concrete usage over abstract definition: show the word working in a sentence
rather than describing what it does.`,
	sharedLevel + `

Draw comparisons from words they have actually looked up when a comparison helps.`,
	sharedAuthority,
	sharedLanguageGrammar,
}, "\n\n")
