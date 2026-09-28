---
id: 000074
status: open
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
github_issue:
---

# define: a deck word is a set of senses, not one word

## Problem

The deck models a word as one thing. It is keyed by `store.Key(word)`;
`store.WordFacts` holds exactly one `Band` and one `Domain` per word;
`schedule.Progress` is a `map[string]Progress` keyed by that same word key; and
`store.ReviewEvent` has no sense field.

But learning a word means learning a *sense* of it. NOAD puts a river's edge, a
financial institution, a tier of oars, a shot in pool and an aircraft's tilt in
one `bank` entry. Two consequences follow, and the second is not benign.

**Practice can test a sense the learner never met.** Surfaced while designing
#68: `senseFacts` commits to the FIRST usable gloss and the first domain label in
document order, and that gloss is what `renderAuthorPrompt` hands the model — so
an authored item is internally consistent. `entryUsages` does the opposite: it
walks every block and sense and flattens their examples into one list, discarding
which sense each came from. Measured over the 34 committed entry fixtures, 24
words have examples, 242 in total — `run` 40, `set` 36, `read` 26, `bank` 6
across at least four unrelated senses. Pick a stem from that flattened pool and
the item's stem, gloss, band and domain can each be anchored to a different part
of the entry.

**All senses collapse into one Progress.** A learner who knows one of `bank`'s
senses and meets the other four in review produces four wrong answers, and
`schedule.Progress` cannot tell them apart from four failures on one meaning. The
scheduler reads "does not know `bank`" and moves it up the queue, then re-tests
senses the learner has never encountered. This is invisible to the learner and it
is what would make polysemous words feel punishing.
