package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/xianxu/tools/cmd/define/schedule"
)

// runStats prints one screen answering "is any of this working".
//
// The shape mirrors runReflect (reflect.go:338): a mode, a store read, a pure
// transformation, a render. Two writers because a DIAGNOSTIC IS NOT OUTPUT — a
// `--stats > report.txt` must not have "could not read the log" in the middle of
// its figures.
//
// EXIT CODES. 0 when the screen printed, including on an empty deck: there is
// nothing wrong with having done nothing yet, and a new learner meeting exit 1
// would read it as breakage. 1 when the log could not be read, because the
// figures would then be silently LOW rather than absent, and a wrong number is
// worse than a refusal. A nil deck is 0 — a statement about the directory, not a
// failure.
func runStats(ctx context.Context, d deps, opt options, out, errOut io.Writer) int {
	if d.deck == nil {
		// noDeckMessage, not a fourth copy of the sentence: the two causes need
		// different words (DEFINE_NO_CAPTURE opened nothing on purpose; an
		// ordinary directory simply has no deck), and its own comment records
		// that "the same fact stated in two places is how the atlas
		// contradictions in #4 started".
		fmt.Fprintln(out, noDeckMessage(opt.noCapture))
		return 0
	}

	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the deck: %v\n", err)
		return 1
	}
	// The WHOLE log: a streak is a fact about all of history, and #8's figures
	// are not windowed. Events reads every day file regardless of `since`
	// (history_cmd.go:100 records the same), so the zero time costs nothing extra.
	events, err := d.deck.Events(time.Time{})
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the log: %v\n", err)
		return 1
	}

	// d.clock, never time.Now(): withStore fills it unconditionally
	// (main.go:214), so a fallback here would be a SECOND source for the one
	// thing every figure on this screen is measured against — and the one a test
	// controls. A screen whose "today" disagrees with the sitting's is a screen
	// whose streak disagrees.
	now := d.clock.Now()
	for _, line := range renderStats(schedule.Summarise(events, deck, now), now) {
		fmt.Fprintln(out, line)
	}
	return 0
}

// renderStats turns the figures into lines.
//
// SEPARATE FROM THE FOLD so the numbers are testable without parsing a screen —
// the Spec asks for exactly this split, and it is why every assertion in
// stats_test.go reads a field rather than a substring.
func renderStats(s schedule.Stats, now time.Time) []string {
	if s.Known == 0 && s.ActiveDays == 0 {
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
	forms := make([]string, 0, len(s.Accuracy))
	for f, a := range s.Accuracy {
		if a.Attempts > 0 {
			forms = append(forms, f)
		}
	}
	if len(forms) == 0 {
		return nil
	}
	sort.Strings(forms)

	out := make([]string, 0, len(forms))
	for _, f := range forms {
		a := s.Accuracy[f]
		out = append(out, fmt.Sprintf("  %-16s %3.0f%%  (%d of %d)",
			formLabel(f), a.Rate()*100, a.Correct, a.Attempts))
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
