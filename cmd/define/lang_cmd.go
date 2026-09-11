package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// parseLangArgs reads /lang's optional tag. The second return distinguishes
// "switch to l" from "tell me what it is", the same split parseSoundArgs makes —
// and for the same reason: a report and a mutation are different commands
// wearing one name, so the branch has to be decided from the arguments alone.
func parseLangArgs(args []string) (l store.Lang, set bool, err error) {
	switch len(args) {
	case 0:
		return "", false, nil
	case 1:
	default:
		return "", false, fmt.Errorf("/lang takes one language, not %q", strings.Join(args, " "))
	}
	parsed, err := store.ParseLang(args[0])
	if err != nil {
		return "", false, err
	}
	return parsed, true, nil
}

// runLang is /lang: which language this deck is in, and switching it.
//
// Unlike /sound, this PERSISTS. That is not a stylistic difference — a one-shot
// `define madrugar` has no session to inherit a mode from, so a session-scoped
// language would mean re-declaring it on every lookup, which is the friction the
// mode exists to remove. It is a property of the directory, which is already the
// unit everything else here scopes to.
func runLang(c commandCtx, args []string) int {
	lang, set, err := parseLangArgs(args)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /lang: %v\n", err)
		return 2
	}
	if !set {
		fmt.Fprintf(c.stdout, "  defining in %s\n", c.lang)
		// Which dictionary is answering. The curated list is a short honest list,
		// so on a machine with a different set installed a wrong pick should be
		// findable rather than puzzling — and this is where a learner looks,
		// instead of a line printed on every lookup.
		if c.dictName != "" {
			fmt.Fprintf(c.stdout, "  from %s\n", c.dictName)
		}
		return 0
	}
	if c.setLang == nil {
		// No directory to write to — DEFINE_NO_CAPTURE, or an unopenable one.
		// Accepting silently would be a lie about what it did, exactly as it
		// would be for /sound without a session.
		fmt.Fprintln(c.stderr, "define: /lang cannot save a language here; use -lang es for one run")
		return 2
	}
	if err := c.setLang(lang); err != nil {
		if errors.Is(err, errDeckDeclined) {
			// THE CONFIRMATION IS DERIVED FROM THE EFFECT (#50 BR-18). A one-shot
			// /lang has only a durable effect, so in a directory nobody agreed to
			// write to it is a no-op — and announcing "now defining in es" there
			// was the command reporting a switch it had not made. The session
			// language still changes, which is the honest half, and the sentence
			// says exactly that much.
			fmt.Fprintf(c.stdout, "  defining in %s for this session only, not saved "+
				"(this directory is not a deck)\n", lang)
			return 0
		}
		fmt.Fprintf(c.stderr, "define: /lang: %v\n", err)
		return 2
	}
	if lang == c.lang {
		// Persisted anyway, and the message says which of the two things
		// happened. A directory with no lang.txt is ALREADY "en" by default, so
		// skipping the write here would leave "/lang en" with no way to make that
		// explicit — the one state a learner cannot otherwise reach.
		fmt.Fprintf(c.stdout, "  still defining in %s, now on the record\n", lang)
		return 0
	}
	fmt.Fprintf(c.stdout, "  now defining in %s\n", lang)
	return 0
}
