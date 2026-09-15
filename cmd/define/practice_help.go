package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// Practice help (#61): English beside the deck-language text a learner is asked
// to read, shown BEFORE answering, for a learner who cannot yet read the
// question alone.
//
// Everything in this file is PURE. practice_help_client.go is the thin shell
// that reaches the cache and the model; package play only displays what it is
// given. The split is what lets every rule below be tested with literal inputs.

// helpMode is what kind of text is being translated. It is part of the cache
// key, because the same words as a definition and as a sentence with a hole in
// it are checked differently.
type helpMode string

const (
	helpGloss helpMode = "gloss" // a dictionary definition: a Choice option or a Board cell
	helpCloze helpMode = "cloze" // a sentence with the answer hidden as Blank
)

// helpKey is what a translation is OF: the exact text, its language, its kind.
// It is also the cache key, so a text that changes by one byte is a new key and
// never inherits a stale translation.
type helpKey struct {
	lang store.Lang
	mode helpMode
	text string
}

// helpNeed is one shown text a question wants English for.
//
// answer is the hidden word of a cloze and it NEVER reaches the model: it is
// here only so the local check can refuse a translation that names it. slot is
// the option or cell the text belongs to.
type helpNeed struct {
	key    helpKey
	answer string
	slot   int
}

// The bounds, each with its reason in the plan's operating envelope (#61 PQ-6).
const (
	maxHelpSource     = 4 << 10  // one shown text; a gloss is ~200 bytes, a stem ~300
	maxHelpEnglish    = 8 << 10  // one translation, before folding
	maxHelpBatch      = 16       // texts per model call
	maxHelpBatchBytes = 32 << 10 // source bytes per model call
)

// helpWanted is whether a sitting shows English help: bilingual display on, in
// a deck whose language is not English. English help for an English deck would
// repeat the question.
func helpWanted(d deps) bool {
	return d.bilingualEnabled() && d.lang != "" && d.lang != store.DefaultLang
}

// helpNeedsOf lists what each question would show English for, in display
// order: every option gloss of a Choice, the blanked sentence of a Cloze, every
// cell gloss of a Board.
//
// READ OFF THE QUESTIONS, not off the material they were built from, so what is
// translated is byte for byte what is shown. A Cloze's text is Blanked(), the
// very string its prompt prints.
func helpNeedsOf(qs []play.Question, lang store.Lang) [][]helpNeed {
	out := make([][]helpNeed, len(qs))
	for i, q := range qs {
		switch q := q.(type) {
		case *play.Choice:
			for j, o := range q.Options() {
				out[i] = append(out[i], helpNeed{key: helpKey{lang, helpGloss, o.Gloss}, slot: j})
			}
		case *play.Cloze:
			answer := ""
			for _, o := range q.Options() {
				if o.Correct {
					answer = o.Word
				}
			}
			if q.Blanked() != "" {
				out[i] = append(out[i], helpNeed{key: helpKey{lang, helpCloze, q.Blanked()}, answer: answer})
			}
		case *play.Board:
			for j, c := range q.Cells() {
				if strings.TrimSpace(c.Gloss) != "" {
					out[i] = append(out[i], helpNeed{key: helpKey{lang, helpGloss, c.Gloss}, slot: j})
				}
			}
		}
	}
	return out
}

// helpKeysOf is every distinct key the needs name, in first-shown order, so a
// sitting's requests are deterministic.
func helpKeysOf(needs [][]helpNeed) []helpKey {
	seen := map[helpKey]bool{}
	var keys []helpKey
	for _, qn := range needs {
		for _, n := range qn {
			if !seen[n.key] {
				seen[n.key] = true
				keys = append(keys, n.key)
			}
		}
	}
	return keys
}

// checkHelp turns a translation into text fit to show beside need, or refuses
// it.
//
// THE REPLY IS UNTRUSTED whether it came from the model a second ago or from a
// cache file written last week (ARCH-SECURE): it reaches a terminal, so control
// and bidi-override characters are refused rather than escaped. Whitespace is
// folded to single spaces, because a help line is one line.
//
// A CLOZE TRANSLATION must keep every blank and must not name the hidden word.
// Both failures render perfectly and give the answer away, which is why they are
// checked here rather than trusted to the prompt.
func checkHelp(need helpNeed, english string) (string, bool) {
	if len(english) > maxHelpEnglish || !utf8.ValidString(english) {
		return "", false
	}
	text := strings.Join(strings.Fields(english), " ")
	if text == "" {
		return "", false
	}
	for _, r := range text {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return "", false
		}
	}
	if need.key.mode == helpCloze {
		if strings.Count(text, Blank) != strings.Count(need.key.text, Blank) {
			return "", false
		}
		// wordIndexIn, the package's one answer to "where does this word occur",
		// and NOT stemUsesTheWord: that one refuses any text already holding a
		// blank, which every cloze translation does, so it would never see the
		// leak it was asked about.
		if at, _ := wordIndexIn(text, need.answer); need.answer != "" && at >= 0 {
			return "", false
		}
	}
	return text, true
}

// acceptHelp keeps a candidate translation only if it passes checkHelp for
// EVERY need that shares its text. One key can serve two questions, and a
// translation safe beside one cloze may name the other's answer.
func acceptHelp(needs [][]helpNeed, candidates map[helpKey]string) map[helpKey]string {
	ok := map[helpKey]string{}
	bad := map[helpKey]bool{}
	for _, qn := range needs {
		for _, n := range qn {
			raw, has := candidates[n.key]
			if !has || bad[n.key] {
				continue
			}
			text, good := checkHelp(n, raw)
			if !good {
				bad[n.key] = true
				delete(ok, n.key)
				continue
			}
			ok[n.key] = text
		}
	}
	return ok
}

// applyHelp gives each question the accepted English it can use and reports
// how many questions wanted help and got none.
//
// A CHOICE TAKES ALL OF ITS OPTIONS' ENGLISH OR NONE: English under three of
// four options would single out the fourth. A Board's cells are independent,
// because a board is self-assessment and each cell explains only itself.
func applyHelp(qs []play.Question, needs [][]helpNeed, accepted map[helpKey]string) (missed int) {
	for i, q := range qs {
		if len(needs[i]) == 0 {
			continue
		}
		got := 0
		switch q := q.(type) {
		case *play.Choice:
			help := make([]string, len(q.Options()))
			for _, n := range needs[i] {
				help[n.slot] = accepted[n.key]
			}
			if q.SetHelp(help) {
				got = 1
			}
		case *play.Cloze:
			if e, ok := accepted[needs[i][0].key]; ok {
				q.SetHelp(e)
				got = 1
			}
		case *play.Board:
			for _, n := range needs[i] {
				if e, ok := accepted[n.key]; ok {
					q.SetHelp(n.slot, e)
					got++
				}
			}
		}
		if got == 0 {
			missed++
		}
	}
	return missed
}

// planHelpBatches groups the texts still missing a translation into requests.
//
// Exact duplicates are one text: two options sharing a gloss cost one
// translation. A text over maxHelpSource is left out WHOLE, never truncated,
// because half a sentence translated is a different sentence.
func planHelpBatches(keys []helpKey) (batches [][]helpKey, excluded []helpKey) {
	seen := map[helpKey]bool{}
	var cur []helpKey
	size := 0
	for _, k := range keys {
		if strings.TrimSpace(k.text) == "" || seen[k] {
			continue
		}
		seen[k] = true
		if len(k.text) > maxHelpSource {
			excluded = append(excluded, k)
			continue
		}
		if len(cur) > 0 && (len(cur) == maxHelpBatch || size+len(k.text) > maxHelpBatchBytes) {
			batches = append(batches, cur)
			cur, size = nil, 0
		}
		cur = append(cur, k)
		size += len(k.text)
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches, excluded
}

// helpTaskName keys the golden, the cassette and the usage label.
const helpTaskName = "practice-help"

// helpReply is the model's answer for one batch.
//
// NUMBERED, not positional: a reply that drops or reorders one entry must not
// shift every later translation onto the wrong text.
type helpReply struct {
	Translations []helpTranslation `json:"translations"`
}

type helpTranslation struct {
	N       int    `json:"n"`
	English string `json:"english"`
}

const helpSystem = "You translate short texts from a language exercise into plain English for a learner. " +
	"You translate only what each text says, and you never reveal a hidden word."

// renderHelpPrompt builds the request for one batch. Pure.
//
// The model sees the shown texts and nothing else: no answer, no Correct flag,
// no headword a cloze hides (ARCH-PURE, and the leak it prevents).
func renderHelpPrompt(lang store.Lang, batch []helpKey) llm.Request {
	if lang == "" {
		lang = store.DefaultLang
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## The texts\n\nEvery text is in the language with IETF code `%s`.\n\n", lang)
	for i, k := range batch {
		kind := "a dictionary definition"
		if k.mode == helpCloze {
			kind = "a sentence with a word hidden as " + Blank
		}
		fmt.Fprintf(&b, "%d. (%s) %s\n", i+1, kind, strings.Join(strings.Fields(k.text), " "))
	}
	b.WriteString("\n## What to write\n\n")
	b.WriteString("For each numbered text, one English translation that a beginner can read beside the original.\n\n")
	b.WriteString("1. Translate what the text says. Do not explain it, add examples, or add anything it does not say.\n")
	b.WriteString("2. `" + Blank + "` marks a hidden word. Copy every `" + Blank + "` into the translation exactly, at the matching place. " +
		"Never fill it in, hint at it, or name the hidden word.\n")
	b.WriteString("3. One translation per text, numbered as the text is, each on one line.\n")

	schema, err := llm.SchemaFor[helpReply]()
	if err != nil {
		schema = nil
	}
	return llm.Request{Task: helpTaskName, System: helpSystem, Prompt: b.String(), Schema: schema}
}

// helpTask adapts the rendered request to a typed task, the way harvest's
// tasks do, so the golden and the wire cannot differ.
func helpTask(lang store.Lang, batch []helpKey) llm.Task[helpReply] {
	req := renderHelpPrompt(lang, batch)
	return llm.Task[helpReply]{Name: req.Task, System: req.System, Prompt: req.Prompt}
}

// readHelpReply maps a reply back onto its batch. A number outside the batch,
// or one that appears twice, is refused: which text it belongs to is unknown.
func readHelpReply(batch []helpKey, r helpReply) map[helpKey]string {
	count := map[int]int{}
	for _, t := range r.Translations {
		count[t.N]++
	}
	got := map[helpKey]string{}
	for _, t := range r.Translations {
		if t.N < 1 || t.N > len(batch) || count[t.N] != 1 {
			continue
		}
		got[batch[t.N-1]] = t.English
	}
	return got
}

// sortedHelpKeys orders keys deterministically, for writes whose order the
// cache's eviction reads.
func sortedHelpKeys(m map[helpKey]string) []helpKey {
	keys := make([]helpKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.lang != b.lang {
			return a.lang < b.lang
		}
		if a.mode != b.mode {
			return a.mode < b.mode
		}
		return a.text < b.text
	})
	return keys
}
