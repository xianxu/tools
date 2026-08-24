package main

import (
	"strings"

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
	// CurrentWord and CurrentEntry are set when the question FOLLOWS a lookup,
	// which is the common case: you look a word up, then ask about it.
	CurrentWord  string
	CurrentEntry string
	// SessionWords are this session's lookups, newest last. DeckWords are the
	// store's, which survive the process.
	SessionWords []string
	DeckWords    []string
	// UserModel is user-model.md verbatim (#17 writes it). Passed whole rather
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
const maxTurns = 6

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

// askSystem is the standing instruction. Short and specific: the context blocks
// carry everything variable, and a long preamble here would be paid for on every
// question while saying the same thing each time.
//
// "Answer at the level the learner model implies" is the sentence that makes the
// whole adaptive loop worth building — without it the context is decoration.
const askSystem = `You are helping someone build their English vocabulary, inside a dictionary tool.

Answer the question directly and briefly — a few sentences, not an essay. Prefer
concrete usage over abstract definition: show the word working in a sentence
rather than describing what it does.

The context blocks tell you who is asking. Pitch the answer at the level the
learner model implies, and draw comparisons from words they have actually looked
up when a comparison helps. If the learner model is absent, write for a capable
adult reader and do not guess at their level.

Never invent a definition that contradicts the dictionary entry you were given —
it is authoritative and you are not. Say plainly when you are unsure.`
