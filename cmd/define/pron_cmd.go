package main

import (
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

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
		// The message points at where the answer IS. #29 chose a declared
		// language over an inferred one because ORIGIN and the CDN both fail to
		// tell a live loanword from a naturalised one, so the entry on screen is
		// what the learner reads the language off.
		return "", fmt.Errorf("which language? /pron fr — the entry's ORIGIN says which")
	default:
		return "", fmt.Errorf("/pron takes one language, not %q", strings.Join(args, " "))
	}
	return store.ParseLang(args[0])
}

// runPron is /pron: hear the current word in another language, once.
//
// It does NOT play, and that is the whole shape of it. Commands are dispatched
// inside the raw editor's cooked block, where playing would hand Ctrl-C back to
// the line discipline — the discipline swallows the byte, the key reader sees
// nothing, and the session looks frozen for the length of the recording.
// workshop/lessons.md has this as "Raw mode: render cooked, play raw". So this
// records the request and the LOOP performs it, in raw mode, through the very
// same replay a bare Enter uses.
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
	if c.replay == nil {
		fmt.Fprintln(c.stderr, "define: /pron replays the word you just looked up; there is none yet")
		return 2
	}
	c.replay(lang)
	return 0
}
