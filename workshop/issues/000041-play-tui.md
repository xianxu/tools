---
id: 000041
status: open
deps: ["tools#39"]
github_issue:
created: 2026-08-31
updated: 2026-08-31
estimate_hours:
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

## Spec

**`--play` draws through the same seam the editor uses.** Not a new abstraction:
`screen` is a pure line buffer plus viewport with a `display` interface over it
(`Draw`, `Page`, `Scroll`, `Resize`), and `liveScreen` is the only part that
touches a terminal. Adopting it is what closes `#30` D5a's predicted divergence.

**A status bar, pinned to the bottom.** What it shows:

```
18 due · 7 answered · 11 left            ~14 reviews/day at your current mix
```

The right-hand number is `#39`'s load figure — `Σ 1/IntervalDays(box)` over the
deck. It is the number that should govern how many new words the learner takes
on, and today it is computed nowhere and shown nowhere. Hence the dependency.

**The transcript must survive exit, and that is already solved.** `#30` M1.2b is
*"Paint draws a whole frame, and the transcript survives the alt screen"* — a
sitting's words should still be on screen after quitting, not wiped by the
alt-screen restore. Reuse that behaviour rather than re-deciding it.

**Paging comes for free**, and is the point of the first bullet above: a reveal
longer than the viewport pages rather than pushing the question off the top.
`#30` already decodes the wheel and PageUp/PageDown into `Page`/`Scroll`.

**Not in scope:** any new question form. This issue changes how `--play` DRAWS
and nothing about what it asks. `#40` is the first consumer.

## Done when

- [ ] `--play` paints whole frames through `display`; no path appends bare lines.
- [ ] A status bar stays pinned at the bottom across question, reveal and resize.
- [ ] A reveal longer than the terminal PAGES; the prompt word stays on screen.
- [ ] The transcript survives exit, pinned by the same test shape `#30` used.
- [ ] SIGWINCH repaints correctly mid-sitting.
- [ ] A pty conformance test drives a real sitting and asserts the bar is present, updates as answers land, and is still there after a reveal.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-31

Filed from a design conversation. The operator's framing was "full TUI with a
bottom bar of today's coverage and progress"; the enabling change is narrower
than that, because `#30` already built the screen abstraction for the editor and
this is its second consumer.
