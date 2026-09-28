---
id: '000038'
status: done
started: 2026-08-30T16:08:11-07:00
created: 2026-08-30
updated: 2026-09-01
estimate_hours: 2.83
actual_hours: 4.24
---

# the review loop's words are not clickable, because --play draws its own frames

## Problem

Operator, 2026-08-30, with a screenshot of a `--play` sitting:

> in the --play mode, link the words so user can click on them for
> pronunciation. in the screenshot: link sycophantic, obsequious, constabulary.

`#30` made a rendered entry's tokens clickable in the interactive loop. The
review loop shows a word and asks you about it — and that word is the most
obviously clickable thing in the program, because hearing it is the whole point
of the exercise. It is inert.

**The target is easy; the coordinates are not.** A recall form's prompt IS the
word (`play/recall.go:29`, `Prompt() string { return r.word }`), so there is no
region-finding problem — nothing to search for, no offsets to survive a wrap.

What is missing is the thing `#30` spent a milestone building: **`--play` draws
its own frames.** It writes through `crlfWriter` straight to a terminal that
scrolls underneath it (`play_loop.go:64-65`), so it does not know which row
anything is on. A click reports an absolute row; without an owned screen there
is nothing to resolve it against. That is exactly `#30`'s D1, which chose the
alternate screen so the mapping is exact by construction rather than tracked.

`#30`'s D5a already named this seam and deferred it:

> `#32` keeps its `--play` half, which continues to draw its own frames through
> `crlfWriter` and is untouched by this issue.

So this issue is where that debt comes due. It is ALSO `#30`'s own promise under
test — "the affordance is ONE mechanism with a registry of regions, so a third
consumer is a row rather than a new feature". A third consumer IS a row; a third
FRAME-DRAWER is not. The honest reading is that `--play` has to join the screen,
and then the click is a row.
