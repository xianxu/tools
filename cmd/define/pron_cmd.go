package main

import (
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// pronUsage is /pron's row in the registry: THE statement of the command's
// argument rule, which /help pron prints and the docs quote. It is not pronHelp,
// and the distinction is the point: pronHelp documents the -pron FLAG, which
// deliberately does NOT infer (#35 D6).
const pronUsage = "Replay this word once in another language. With nothing, it reads the source " +
	"language off the entry's ORIGIN and says which it chose. It declines when ORIGIN names " +
	"only historical stages (Old French, Latin) or cognates (\"related to Dutch …\"), because " +
	"neither is a language anyone speaks the word in today."

// parsePronArgs reads /pron's language. Pure, so the table is a unit test.
//
// A language is REQUIRED, which is where this differs from parseLangArgs and
// parseSoundArgs. Both of those return a "set" bool because they name a SETTING
// with a current value worth printing — "/lang" alone is a fair question. /pron
// names an ACTION and leaves nothing behind, so there is no current value and a
// bare one is a half-typed command.
func parsePronArgs(args []string) (store.Lang, error) {
	switch len(args) {
	case 1:
	case 0:
		// NO LANGUAGE is a request to INFER, not a usage error (#35). #29 shipped
		// it as an error because it had rejected inference — but that rejection
		// was about inferring on EVERY lookup, where a wrong guess is silent and
		// unasked-for. Here the user typed the gesture and the answer is
		// reported, so the inference is opt-in and auditable. runPron owns the
		// reading; this function only says an argument is optional.
		return "", nil
	default:
		return "", fmt.Errorf("/pron takes one language, not %q", strings.Join(args, " "))
	}
	return store.ParseLang(args[0])
}

// runPron is /pron: hear the current word in another language, once.
//
// It does NOT play, and that is the whole shape of it. It records the request
// and the LOOP performs it, through the very same replay a bare Enter uses — so
// /pron is one parameter of an existing path rather than a second player.
//
// The shape was forced by a hazard that is now gone: commands were dispatched
// inside the raw editor's cooked block, where playing would have handed Ctrl-C
// back to the line discipline — the discipline swallows the byte, the key reader
// sees nothing, and the session looks frozen for the length of the recording
// (workshop/lessons.md, "Raw mode: render cooked, play raw"). #30 D4 deleted the
// cooked block, so there is no wrong mode left to play in. The separation stays
// on its own merits.
//
// A nil c.replay means there is nothing to replay — the one-shot path, or a loop
// before its first lookup. Saying so beats playing silence, which is the call
// /sound already makes for a nil setTimes.
func runPron(c commandCtx, args []string) int {
	lang, err := parsePronArgs(args)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /pron: %v\n", err)
		return 2
	}
	// NOTHING LOOKED UP is answered BEFORE the inference, or bare /pron as the
	// first line of a session blames an entry that does not exist: it printed
	// "this entry has no ORIGIN" when there was no entry at all. `replay` is nil
	// exactly when the session has no current word, so this is the same condition
	// the argument form already used — it was simply below the inference.
	if c.replay == nil {
		fmt.Fprintln(c.stderr, "define: /pron replays the word you just looked up; there is none yet")
		return 2
	}
	// No language: read it off the entry the user is looking at (#35).
	//
	// The REPORT is not decoration. A silent inference cannot be audited, and it
	// is what makes a contested ORIGIN visible — `piano` reads "either from
	// French, or …", and taking the first-named while saying so leaves the hedge
	// where the reader can see it and override in one word.
	if lang == "" {
		got, named, err := OriginLanguage(ParseEntry(c.entry))
		if err != nil {
			fmt.Fprintf(c.stderr, "define: /pron: %v. Name one: /pron fr\n", err)
			return 2
		}
		fmt.Fprintf(c.stdout, "  ORIGIN says %s\n", named)
		lang = got
	}
	c.replay(lang)
	return 0
}
