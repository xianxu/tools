---
id: '000041'
status: done
started: 2026-08-31T13:26:27-07:00
created: 2026-08-31
updated: 2026-08-31
estimate_hours: 3.31
actual_hours: 4.77
---

# play mode paints frames through screen, and gains a status bar

## Problem

There are two interactive surfaces in this binary and they render two different
ways. The editor loop draws whole frames through `screen`/`liveScreen` (`#30`);
`--play` writes lines through `crlfWriter` to a scrolling terminal. `atlas/define.md`
records the consequence directly: *"`play_loop.go` writes through `crlfWriter` to
a scrolling terminal and therefore owns no coordinates. `#30` D5a predicted this
seam."*

Three things follow from owning no coordinates, and they are all user-visible:

- **A long entry scrolls the question away.** Form 2.3's reveal shows the whole
  rendered definition; on a word like `run` or `bank` that is far more than a
  screenful, and the word being asked about is gone off the top. `screen` already
  has a viewport and paging; `--play` cannot use them.
- **There is nowhere to put a status bar.** A learner in a sitting cannot see how
  many words are left, how many are due today, or what the day costs. That number
  exists once `#39` lands and has nowhere to go.
- **Every future mode inherits the limitation.** `#40`'s board is a grid with a
  cursor; a grid cannot be drawn by appending lines.
