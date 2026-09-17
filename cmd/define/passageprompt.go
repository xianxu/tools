package main

import (
	"strings"

	"github.com/xianxu/tools/internal/llm"
)

// The reserved-bracket escapes. They existed only as prose inside askSystem's
// raw string, which is why the answer direction had a rule and the prompt
// direction did not. Extracted so both can name the same thing (#67).
const (
	escLeft  = "&#91;"
	escRight = "&#93;"
)

// The selection markers, in the SAME family as the reserved [lang=xx]…[/lang]
// rather than a second bracket dialect, so the model meets one grammar (ARCH-DRY).
const (
	selOpen  = "[sel]"
	selClose = "[/sel]"
)

// passageTask names this request. A task of its own, not console-question: it
// keys a separate golden and a separate cassette, so the existing ask prompt's
// golden is untouched by anything that happens here.
const passageTask = "passage-question"

const headerPassage = "## The passage"

// passageAsk is a question about a passage, as DATA.
//
// It carries the ordinary askContext whole rather than duplicating its fields:
// the context blocks are the same blocks, and forking them is how two prompts
// come to disagree about what the model is told.
type passageAsk struct {
	Context askContext
	Passage *passage
	Marks   markSet
}

// renderPassagePrompt turns a passage and its marks into the request that will be
// sent. Pure, so the golden snapshots the real request rather than a string a
// test concatenated.
func renderPassagePrompt(a passageAsk) llm.Request {
	req := renderAskPrompt(a.Context)
	req.Task = passageTask
	req.System = passageSystem

	// The passage goes in just before the question, so the last thing the model
	// reads is what it was asked — the same ordering renderAskPrompt uses.
	prompt := req.Prompt
	const questionHeader = "## The question"
	at := strings.LastIndex(prompt, questionHeader)
	if at < 0 {
		at = len(prompt)
	}
	body := headerPassage + "\n" + markedPassageText(a.Passage, a.Marks) + "\n\n"
	req.Prompt = prompt[:at] + body + prompt[at:]
	return req
}

// markedPassageText is the passage with its marks bracketed IN PLACE.
//
// In place rather than "the passage, and separately a list of marked words",
// because position is then unambiguous: a word occurring twice needs no
// occurrence index, since the bracket is already at the right one.
//
// Every bracket the passage itself contains is escaped first, so pasted text
// cannot forge a marker — the passage is untrusted input on its way into a prompt
// (ARCH-SECURE), and the escape is the one askSystem already names for the answer
// direction.
func markedPassageText(p *passage, m markSet) string {
	if p == nil {
		return ""
	}
	marked := map[passageSpan]bool{}
	for _, s := range m.ordered() {
		marked[s] = true
	}
	var out strings.Builder
	for i := range p.lines {
		if i > 0 {
			out.WriteString("\n")
		}
		line := p.line(i)
		at := 0
		for _, sp := range p.spans(i) {
			out.WriteString(escapeReservedBrackets(line[at:sp.start]))
			word := escapeReservedBrackets(line[sp.start:sp.end])
			if marked[sp] {
				out.WriteString(selOpen + word + selClose)
			} else {
				out.WriteString(word)
			}
			at = sp.end
		}
		out.WriteString(escapeReservedBrackets(line[at:]))
	}
	return out.String()
}

// escapeReservedBrackets makes pasted text unable to forge a marker.
func escapeReservedBrackets(s string) string {
	return strings.NewReplacer("[", escLeft, "]", escRight).Replace(s)
}

// passageSystem is askSystem's sibling for a passage question.
//
// It states the one thing that differs: the reader is asking about a text they
// are reading, the marks are what they could not follow, and the answer covers
// BOTH the passage and each mark. A passage question is not N word questions —
// the relations among the marked words are most of what a reader is missing, and
// per-span glosses discard exactly that (#67).
const passageSystem = `You are helping someone read, inside a dictionary tool. They have pasted a passage they are working through.

Words and phrases they could not follow are wrapped in [sel]…[/sel]. Answer BOTH:
first what the passage as a whole is saying, then each marked span in the context
of that passage — which sense is in play here, and how the marked words relate to
each other. Pick the sense the passage actually uses; do not list the others.

Answer directly and briefly — a few sentences, not an essay. Do not restate the
passage back at them.

If nothing is marked, explain the passage as a whole.

Never invent a definition that contradicts a dictionary entry you were given — it
is authoritative and you are not. Say plainly when you are unsure.

Annotate the language of your answer using short nonnested passages:
[lang=es]Spanish text[/lang] and [lang=en]English text[/lang]. Use a two-letter
language code, or und when unknown. Place boundaries at actual language changes,
including an inline phrase in another language. Keep passages below 4000 characters.
To discuss these reserved markers literally, escape their brackets as ` + escLeft + ` and
` + escRight + `. Never emit terminal escape sequences or control characters.`
