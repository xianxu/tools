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
	// By POSITION, not by matching word runs. A mark can be a PHRASE — a drag
	// produces one span covering several words — and the first version compared
	// each mark to a whole word run, so a dragged phrase matched nothing and was
	// silently dropped from the prompt (#67, C-B).
	byLine := map[int][]passageSpan{}
	for _, sp := range m.ordered() {
		byLine[sp.line] = append(byLine[sp.line], sp)
	}
	var out strings.Builder
	for i := range p.lines {
		if i > 0 {
			out.WriteString("\n")
		}
		line := p.line(i)
		at := 0
		for _, sp := range byLine[i] {
			if sp.start < at || sp.end > len(line) || sp.start >= sp.end {
				continue // overlapping or out of range: the set is a set, not a stack
			}
			out.WriteString(escapeReservedBrackets(line[at:sp.start]))
			out.WriteString(selOpen + escapeReservedBrackets(line[sp.start:sp.end]) + selClose)
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
// COMPOSED from the shared clauses rather than restating them: the level policy,
// the dictionary-authority rule and the language grammar are the same rules, and
// a second spelling of a wire format is a second format. What is written out here
// is only what actually differs — the reader is working through a text, the marks
// are what they could not follow, and the answer covers BOTH.
var passageSystem = strings.Join([]string{
	`You are helping someone read, inside a dictionary tool. They have pasted a passage they are working through.`,
	`Words and phrases they could not follow are wrapped in ` + selOpen + `…` + selClose + `. Answer BOTH:
first what the passage as a whole is saying, then each marked span in the context
of that passage — which sense is in play here, and how the marked words relate to
each other. Pick the sense the passage actually uses; do not list the others.

A marked span may be a PHRASE. Explain it as one thing, not word by word.

Answer directly and briefly — a few sentences, not an essay. Do not restate the
passage back at them. If nothing is marked, explain the passage as a whole.`,
	sharedLevel,
	sharedAuthority,
	sharedLanguageGrammar,
}, "\n\n")
