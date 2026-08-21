package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
