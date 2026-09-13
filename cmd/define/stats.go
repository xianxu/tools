package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/xianxu/tools/cmd/define/schedule"
	"github.com/xianxu/tools/cmd/define/store"
)

// runStats prints one screen answering "is any of this working".
//
// The shape mirrors runReflect (reflect.go:338): a mode, a store read, a pure
// transformation, a render. Two writers because a DIAGNOSTIC IS NOT OUTPUT — a
// `--stats > report.txt` must not have "could not read the log" in the middle of
// its figures.
//
// EXIT CODES. 0 when the screen printed, including on an EMPTY deck: there is
// nothing wrong with having done nothing yet, and a new learner meeting exit 1
// would read it as breakage. 1 when the deck or the log could not be read,
// because the figures would then be silently LOW rather than absent, and a wrong
// number is worse than a refusal.
//
// A NIL DECK IS ALSO 1, on stderr, which is a correction: this file first argued
// it was "a statement about the directory, not a failure" and returned 0 to
// stdout. Every sibling disagrees — --forget (main.go:1193), --harvest
// (harvest.go:111), --reflect (reflect.go:340) and /history (history_cmd.go:220)
// all print noDeckMessage to stderr and return 1, unanimously. The case that
// argument was protecting is the EMPTY deck, which still exits 0 with a
// sentence; a nil one means DEFINE_NO_CAPTURE or no directory at all, and a
// script that asked for figures and got none should know.
func runStats(d deps, opt options, out, errOut io.Writer) int {
	if d.deck == nil {
		// noDeckMessage, not a fourth copy of the sentence: the two causes need
		// different words (DEFINE_NO_CAPTURE opened nothing on purpose; an
		// ordinary directory simply has no deck), and its own comment records
		// that "the same fact stated in two places is how the atlas
		// contradictions in #4 started".
		fmt.Fprintln(errOut, noDeckMessage(opt.noCapture))
		return 1
	}

	return printStats(d.deck, d.clock, "--stats", d.deckPermission, out, errOut)
}

// printStats reads, folds and renders — the whole of the screen below the door.
//
// IT IS THE SHARED HALF, and it exists because the comment claiming the halves
// were shared was FALSE. `runStats` and `runStatsCommand` each did their own
// Deck(), Events(), clock read, Summarise and render loop: five duplicated
// statements under a doc comment that said "ONE FOLD, ONE RENDERER, TWO ENTRY
// POINTS… everything below that seam is shared". The boundary review measured it
// (#8 BR-4). A comment asserting DRY is not DRY; this is.
//
// `who` names the caller in a diagnostic, because "define: --stats: …" and
// "define: /stats: …" tell a reader which door they came through — the only
// thing the two genuinely differ in besides where the deck comes from.
func printStats(deckStore store.Store, clock store.Clock, who string, perm *deckPermission, out, errOut io.Writer) int {
	deck, err := deckStore.Deck()
	if err != nil {
		fmt.Fprintf(errOut, "define: %s: could not read the deck: %v\n", who, err)
		return 1
	}
	// The WHOLE log: a streak is a fact about all of history, and #8's figures
	// are not windowed. Events reads every day file regardless of `since`
	// (history_cmd.go:100 records the same), so the zero time costs nothing extra.
	events, err := deckStore.Events(time.Time{})
	if err != nil {
		fmt.Fprintf(errOut, "define: %s: could not read the log: %v\n", who, err)
		return 1
	}
	// The injected clock, never time.Now(): withStore fills it unconditionally
	// (main.go:214) and commandCtx carries it (command.go:156), so a fallback
	// would be a SECOND source for the one thing every figure is measured
	// against — and the one a test controls. A screen whose "today" disagrees
	// with the sitting's is a screen whose streak disagrees.
	now := clock.Now()
	// READ, NEVER RESOLVE. saving() reports what has already been settled; asking
	// via allowed() would make putting the question a side effect of --stats, so
	// reading your figures in the wrong directory would offer to create a deck
	// there (#50 PQ-6).
	allowed, decided := perm.saving()
	for _, line := range renderStats(schedule.Summarise(events, deck, now), now, !decided || allowed) {
		fmt.Fprintln(out, line)
	}
	return 0
}

// renderStats turns the figures into lines.
//
// SEPARATE FROM THE FOLD so the numbers are testable without parsing a screen —
// the Spec asks for exactly this split, and it is why every assertion in
// stats_test.go reads a field rather than a substring.
// `saving` says whether anything written here will survive the process. It
// changes ONE thing — the empty screen's call to action — because that sentence
// is the only part of this render that makes a promise.
func renderStats(s schedule.Stats, now time.Time, saving bool) []string {
	if s.Known == 0 && s.ActiveDays == 0 {
		if !saving {
			// NOT "look a word up and it joins your deck", which is FALSE here:
			// nothing will join anything. This screen is exactly where the person
			// who ran define in the wrong directory ends up, so it is the worst
			// possible place to promise a deck that will not exist.
			return []string{
				"Nothing yet, and nothing is being saved here — this directory is not a deck.",
				"",
				"  cd somewhere/else      and run it there",
				"  define --here          make THIS directory a deck",
			}
		}
		// NOT A SCREEN OF ZEROES. Seven figures all reading 0 says "this is
		// broken" to the one person guaranteed to see it — someone who has just
		// installed it. A sentence says the same thing truthfully and points
		// somewhere.
		return []string{
			"Nothing yet — look a word up and it joins your deck.",
			"",
			"  define sycophantic     look a word up",
			"  define --play          review what is due",
		}
	}

	out := []string{
		fmt.Sprintf("  %-16s %d", "words", s.Known),
		fmt.Sprintf("  %-16s %d", "mastered", s.Mastered),
		fmt.Sprintf("  %-16s %d", "active days", s.ActiveDays),
		fmt.Sprintf("  %-16s %s", "streak", streakPhrase(s)),
	}
	// The RATE IS SHOWN ONLY WHEN THE LOG RECORDS ADDS. A deck seeded before the
	// log existed — or by hand, which the README invites — has words and no
	// adds, and "words/day 0.0" beside "words 41" reads as a broken figure
	// rather than as the missing history it is. Gated on the COUNT, not on the
	// rate: a genuinely slow learner whose rate rounds to 0.0 has still been
	// adding words and should see the row.
	if s.Added > 0 {
		out = append(out, fmt.Sprintf("  %-16s %.1f", "words/day", s.AddedPerDay))
	}
	if !s.FirstDay.IsZero() {
		out = append(out, fmt.Sprintf("  %-16s %s", "since", relativeDay(s.FirstDay, now)))
	}
	if lines := accuracyLines(s); len(lines) > 0 {
		out = append(out, "")
		out = append(out, lines...)
	}
	return out
}

// streakPhrase reads the two streaks as one line, because they are one fact with
// a qualifier rather than two numbers.
//
// The longest is shown only when it is LONGER: "3 days (longest 3)" tells the
// learner nothing and reads as a rebuke for not beating themselves.
func streakPhrase(s schedule.Stats) string {
	cur := fmt.Sprintf("%d %s", s.CurrentStreak, plural(s.CurrentStreak, "day"))
	if s.LongestStreak > s.CurrentStreak {
		return fmt.Sprintf("%s (longest %d)", cur, s.LongestStreak)
	}
	return cur
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// accuracyLines is one row per form the log has seen.
//
// SORTED BY NAME, so the screen is stable between runs: map order would reshuffle
// the rows every time and make a learner think something had changed.
func accuracyLines(s schedule.Stats) []string {
	// KEYED BY THE LABEL, not by the raw form. formLabel neutralises, so two
	// distinct keys can print identically — "meaning" and "meaning\x1b[2J" both
	// render as `meaning` — and grouping after the render is what stops the
	// screen showing the same row twice with different numbers (#8 BR-10).
	byLabel := map[string]schedule.FormAccuracy{}
	for f, a := range s.Accuracy {
		if a.Attempts == 0 {
			continue
		}
		l := formLabel(f)
		acc := byLabel[l]
		acc.Attempts += a.Attempts
		acc.Correct += a.Correct
		byLabel[l] = acc
	}
	if len(byLabel) == 0 {
		return nil
	}
	labels := make([]string, 0, len(byLabel))
	for l := range byLabel {
		labels = append(labels, l)
	}
	sort.Strings(labels)

	out := make([]string, 0, len(labels))
	for _, l := range labels {
		a := byLabel[l]
		out = append(out, fmt.Sprintf("  %-16s %3.0f%%  (%d of %d)",
			l, a.Rate()*100, a.Correct, a.Attempts))
	}
	return out
}

// formLabel is a form name made safe to print.
//
// THE LOG IS HAND-EDITABLE and this is the first path that puts Form on a
// terminal. `ReviewEvent.Form` is a bare string (store/event.go:108), so a
// hand-edited `form: "meaning\x1b[2J"` would clear the display — #12's BR-15,
// one field over. Control runes that are not whitespace are dropped, exactly as
// store's oneLine drops them.
//
// An UNKNOWN name is still shown rather than filtered: a row labelled with a name
// nobody recognises is how a learner discovers a stale or hand-edited log, and
// dropping it would hide the fact. An EMPTY one is labelled, because a blank
// left-hand column reads as a rendering bug.
func formLabel(form string) string {
	safe := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, form)
	safe = strings.Join(strings.Fields(safe), " ")
	if safe == "" {
		return "(unnamed form)"
	}
	return safe
}

// statsUsage is /stats's row in the registry.
const statsUsage = "The deck, streak and accuracy figures for this directory. Takes no arguments."

// runStatsCommand is `/stats`, and it is `--stats` with a different door.
//
// ONE FOLD, ONE RENDERER, TWO ENTRY POINTS — through printStats, which is where
// that claim became true. The flag and the command differ only in where the deck
// and the clock come from (deps for one, commandCtx for the other) and in the
// name a diagnostic carries; everything below that is one function, so a change
// to the screen cannot apply to only one of them. That is the property #48 will need for
// `/play`, arriving here first on the cheaper case.
//
// Adding a command is a row in `commands` plus this function; the dispatch loop
// never changes, which is #16's Done-when.
func runStatsCommand(c commandCtx, args []string) int {
	if len(args) > 0 {
		// The same rule the flag states, for the same reason: /stats reads
		// everything and takes no subject, so a word beside it can only be a
		// misread intent.
		fmt.Fprintf(c.stderr, "define: /stats reads the whole log; it takes no arguments, not %q\n",
			strings.Join(args, " "))
		return 2
	}
	if c.deck == nil {
		fmt.Fprintln(c.stderr, noDeckMessage(c.noCapture))
		return 1
	}
	return printStats(c.deck, c.clock, "/stats", c.deckPermission, c.stdout, c.stderr)
}
