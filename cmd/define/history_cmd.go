package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// maxHistoryDays bounds --days because AddDate NORMALISES a year-overflowing
// date rather than failing: --days 999999999 would return a garbage instant and
// print an empty list, which reads as "you have no history" rather than as the
// refusal it should be.
const maxHistoryDays = 3650

const defaultHistoryDays = 2

// historyWindow is the instant "the last N days" starts, in LOCAL time.
//
// Two things make this less obvious than it looks:
//
//   - It is a local-CALENDAR question, so the boundary is local midnight, not
//     now minus N×24h. A window starting at 00:30 silently drops everything you
//     did before half past midnight on the first day.
//   - AddDate, not a Duration. A DST day is 23 or 25 hours (2026-03-08 and
//     2026-11-01 here), so subtracting 24h from midnight lands at 01:00 or 23:00
//     the previous day. AddDate moves the calendar date and lets the offset
//     change underneath, which is what "the day before" means to a reader.
//
// The result keeps now's location, because the store compares it against
// timestamps that kept their own offsets — never against day-file names, which
// are UTC and therefore belong to the wrong calendar for this question.
func historyWindow(now time.Time, days int) time.Time {
	if days < 1 {
		days = 1
	}
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
}

// parseHistoryArgs reads /history's optional window: "7", "--days 7" or
// "--days=7". Errors name the operand, so the message says what to fix.
func parseHistoryArgs(args []string) (int, error) {
	var raw string
	switch {
	case len(args) == 0:
		return defaultHistoryDays, nil
	case len(args) == 1 && strings.HasPrefix(args[0], "--days="):
		raw = strings.TrimPrefix(args[0], "--days=")
	case len(args) == 1 && args[0] == "--days":
		return 0, fmt.Errorf("--days needs a number of days")
	case len(args) == 1:
		raw = args[0]
	case len(args) == 2 && args[0] == "--days":
		raw = args[1]
	default:
		return 0, fmt.Errorf("/history takes one window, not %q", strings.Join(args[1:], " "))
	}

	days, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number of days", raw)
	}
	if days < 1 {
		return 0, fmt.Errorf("%d is not a number of days; ask for at least 1", days)
	}
	if days > maxHistoryDays {
		return 0, fmt.Errorf("%d is more days than the log can meaningfully hold; the limit is %d", days, maxHistoryDays)
	}
	return days, nil
}

// historyRow is one word in the /history view.
type historyRow struct {
	// Word is the KEY, not a spelling that happened to be typed. The row IS the
	// key — showing "Sycophantic" because that capital arrived first would make
	// the dedupe rule invisible and tie the output to event order.
	Word    string
	FirstAt time.Time // the FIRST time this word was ever looked up
	LastAt  time.Time // the most recent, for the relative date
	Lookups int       // successful lookups in the whole log, not just the window
}

// summariseLookups turns the event log into the /history view.
//
// Membership and ordering are DIFFERENT time facts about the same word, and
// keeping them apart is the whole point:
//
//   - a word is IN the list because it was queried inside the window;
//   - it SITS where it does because of when it was first ever seen.
//
// So a word you keep returning to holds the position its first sighting earned
// rather than churning to the top on every re-read. That is why this takes the
// whole log and not a windowed slice — the first sighting is usually older than
// the window, and Events reads every day file regardless of `since`, so asking
// for all of it costs nothing extra (ARCH-DRY: one source answers both facts).
//
// Only successful lookups count. A typo is recorded — #14's up-arrow recall
// must reach a word you typed and got wrong — but it is not vocabulary, so it
// stays out of the words-queried view. One log, two readers.
func summariseLookups(evs []store.ReviewEvent, since time.Time) []historyRow {
	rows := map[string]*historyRow{}
	for _, e := range evs {
		if e.Kind != store.EventLookedUp || !e.Found {
			continue
		}
		k := store.Key(e.Word)
		if k == "" {
			continue
		}
		r, ok := rows[k]
		if !ok {
			r = &historyRow{Word: k, FirstAt: e.At, LastAt: e.At}
			rows[k] = r
		}
		if e.At.Before(r.FirstAt) {
			r.FirstAt = e.At
		}
		if e.At.After(r.LastAt) {
			r.LastAt = e.At
		}
		r.Lookups++
	}

	var out []historyRow
	for _, r := range rows {
		if !r.LastAt.Before(since) { // queried at least once inside the window
			out = append(out, *r)
		}
	}
	// Newest first sighting at the top. The tiebreak keeps the order TOTAL —
	// without it, map iteration makes the output flicker between runs.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].FirstAt.Equal(out[j].FirstAt) {
			return out[i].FirstAt.After(out[j].FirstAt)
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// renderHistory formats the rows. Pure, so the layout is table-tested without a
// store or a terminal.
//
// The relative date is computed on LOCAL CALENDAR DAYS, not elapsed hours, for
// the same reason the window is: a lookup twenty minutes ago at 23:50 is
// "yesterday" once it is past midnight, and saying "0 days ago" would be true
// and useless.
func renderHistory(rows []historyRow, now time.Time, width int) []string {
	nameW := 0
	for _, r := range rows {
		if len(r.Word) > nameW {
			nameW = len(r.Word)
		}
	}
	dateW := 0
	dates := make([]string, len(rows))
	for i, r := range rows {
		// FirstAt, not LastAt: the list is ORDERED by first sighting, so showing
		// any other date makes the ordering look arbitrary.
		dates[i] = relativeDay(r.FirstAt, now)
		if len(dates[i]) > dateW {
			dateW = len(dates[i])
		}
	}

	out := make([]string, 0, len(rows))
	for i, r := range rows {
		line := fmt.Sprintf("  %-*s  %s", nameW, r.Word, dates[i])
		if r.Lookups > 1 {
			line = fmt.Sprintf("  %-*s  %-*s   %d×", nameW, r.Word, dateW, dates[i], r.Lookups)
		}
		if width > 0 && len([]rune(line)) > width {
			line = string([]rune(line)[:width])
		}
		out = append(out, line)
	}
	return out
}

// relativeDay names a day the way a person would.
func relativeDay(at, now time.Time) string {
	at = at.In(now.Location())
	dayOf := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	}
	switch days := int(dayOf(now).Sub(dayOf(at)).Hours() / 24); {
	case days == 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 7:
		return at.Format("Monday")
	default:
		return at.Format("Jan 2")
	}
}

// runHistory is /history: the words you looked up recently, most-recently-first-
// seen at the top.
func runHistory(c commandCtx, args []string) int {
	days, err := parseHistoryArgs(args)
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /history: %v\n", err)
		return 2
	}
	if c.deck == nil {
		fmt.Fprintln(c.stderr, noDeckMessage(c.noCapture))
		return 1
	}

	// The WHOLE log, deliberately. Membership needs the window but ordering
	// needs each word's first sighting, which is usually older than it — and
	// Events reads every day file whatever `since` says, so asking for all of it
	// costs nothing and keeps both facts on one source.
	evs, err := c.deck.Events(time.Time{})
	if err != nil {
		fmt.Fprintf(c.stderr, "define: /history: %v\n", err)
		return 1
	}

	now := c.clock.Now()
	rows := summariseLookups(evs, historyWindow(now, days))
	if len(rows) == 0 {
		fmt.Fprintf(c.stdout, "  nothing looked up in the last %s\n", days2str(days))
		return 0
	}
	for _, line := range renderHistory(rows, now, c.width) {
		fmt.Fprintln(c.stdout, line)
	}
	return 0
}

func days2str(days int) string {
	if days == 1 {
		return "day"
	}
	return fmt.Sprintf("%d days", days)
}
