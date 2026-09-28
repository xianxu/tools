---
id: 000022
status: open
created: 2026-08-26
updated: 2026-08-26
estimate_hours:
github_issue:
---

# active learning set: words graduate out of highlighting

## Problem

#21 highlights every word in the deck, everywhere it appears. That is the right
v1 — one rule, no surprising suppression — but it has a built-in decay: the deck
only grows, so the screen gets greener forever and the signal thins out. A
learner who has looked up 400 words does not need 400 words flagged; they need
the twenty they are still working on.

Operator's framing, which is the point of the issue:

> I imagine as user use this program and some of the words become theirs, and
> those highlighting would stop.

Highlighting should mark *effort still owed*, not *history of contact*.
