---
id: '000039'
status: done
started: 2026-08-31T11:01:05-07:00
created: 2026-08-31
updated: 2026-08-31
estimate_hours: 2.94
actual_hours: 1.59
---

# spaced repetition: one unbounded ladder, confidence-driven promotion, and a visible daily budget

## Problem

`define`'s schedule is a Leitner ladder of `1, 3, 7, 14, 30, 90` days with a
mastery bar of seven consecutive correct answers. Three things are wrong with it,
and they were found by using the tool rather than by reading the code.

**It starts backing off immediately.** The first three rungs are 1, 3 and 7 days,
which assumes the word is already learned and only needs protecting. A word met
once yesterday is not learned. Pimsleur's graduated interval recall and Anki's
FSRS both spend heavily in the first days and then back off; this ladder never
spends.

**It has a ceiling, and a ceiling is unsustainable at any admission rate.** In
steady state, with `a` new words per day over `N` rungs:

```
daily reviews = a × N            (climbing: every word passes each rung once)
              + stock / I_top    (everything parked at the top)
```

The second term grows LINEARLY forever. At 7 new words/day with a 90-day top
rung, the accumulated stock alone costs ~55 reviews/day after two years and there
is no budget left for anything new. The ladder needs no ceiling: if intervals
keep growing geometrically, a word of age τ is reviewed at roughly `1/τ` per day
and the total load integrates to `a × ln(T)` — logarithmic, so a fixed daily
budget supports a nearly constant new-word rate indefinitely.

**Mastery costs 235 days and one slip near the end costs six months.** Seven
CONSECUTIVE correct on a ladder whose top rung is 90 days means five promotions
to climb (55 days) plus two confirmations at 90 days each. A wrong answer resets
the streak to zero.

There is also no way for the learner to see what any of this costs them. The
number that should govern how many new words they take on — reviews per day — is
not computed or shown anywhere.
