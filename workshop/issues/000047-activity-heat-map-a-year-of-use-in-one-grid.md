---
id: 000047
status: open
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
---

# activity heat map: a year of use in one grid

## Problem

**Post-MVP.** `#8`'s `--stats` answers "is any of this working" with seven
numbers. It does not answer *when* — and a year of use has a shape that numbers
flatten: the fortnight you stopped, the month you were commuting, the week the
new job started. A streak is one number about that shape; a grid is the shape.

The operator asked for GitHub's contributions graph, applied to learning: one
cell per day for the last year, shaded by how much happened.

## Spec

**One row per weekday, one column per week, one cell per day, shaded by
activity** — the layout GitHub uses, and worth copying rather than reinventing
because a reader already knows how to read it.

### What counts as activity

Three signals the operator named, and **the third does not exist yet**:

| signal | recorded today? |
|---|---|
| a new word looked up | **yes** — `EventLookedUp` with `found: true`, and `#8`'s `Stats.Added` already counts the first one per word |
| a review question answered | **yes** — `EventReviewed`, keyed by form |
| **a pronunciation played** | **NO** — `store.EventKinds()` is `looked-up`, `asked`, `reviewed`, `flagged`. Nothing records a play, at any of the four sites that can start one (`speak`, the REPL's bare Enter, `playRegion`'s click, `--play`'s reveal) |

So this issue has a **prerequisite it owns**: a fifth event kind, appended where
playback actually succeeds. That is a small change and a real one — it makes
"how much did I listen" answerable for the first time, and `#46`'s click-anything
work is what makes it interesting, because a play is now a deliberate gesture on
a specific word rather than a side effect of a lookup.

**Weighting is a decision, not a detail.** GitHub counts contributions equally;
these three are not equal. A review answered is more effort than a word played.
The plan should either weight them explicitly and say why, or show one signal at
a time and let the reader pick — and "one at a time" may well be the better
product, since *"which days did I actually review"* and *"which days did I look
things up"* are different questions.

### Constraints this inherits

- **Every cell folded from the event log, nothing stored** — `#3`'s shape and
  `#8`'s rule. `schedule.Summarise` is the sibling; a `Grid` fold belongs beside
  it and `stats.go`'s day-keying is the part to reuse rather than rewrite.
- **The learner's calendar, not UTC.** `#8` learned this the hard way: the day
  map is keyed by `time.Time`, whose equality includes the location pointer, so
  every event is converted into `now`'s zone first. A grid gets it wrong in a way
  a reader can SEE — a cell in the wrong column — which is at least honest.
- **It must render without colour.** `-no-color` is a supported mode and a
  terminal that mangles escapes is why it exists. Shading needs a monochrome
  fallback (density characters), not an empty grid.
- **A year is 53 columns.** That does not fit an 80-column terminal at two cells
  per day, and truncating silently is the wrong answer. Either the cell is one
  character wide, or the window shrinks to what fits and says so.

## Done when

- [ ] A pronunciation played is recorded, at every site that can play one — a
      derived guard over those sites, not a list someone maintains.
- [ ] A year of days renders as a grid, in the learner's calendar, folded from
      the log with nothing stored.
- [ ] It renders usefully with `-no-color` and in a narrow terminal, and says
      what it did rather than truncating silently.
- [ ] The weighting (or the one-signal-at-a-time choice) is a stated decision
      with a reason, not an accident of implementation.
- [ ] An empty log renders an empty grid, not a panic and not a blank screen.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-09-07 — filed, post-MVP

Operator request, with a screenshot of GitHub's contributions graph: *"file a
task, post mvp, to print a calendar action heat map, similar to github's heatmap.
activity is new words checked, number of review question answered, number of
audio listened (clicked on to listen) etc."*

**Explicitly NOT in `define-learn`'s MVP scope**, which is now `#8` alone. Filed
against the same product and deliberately outside it.

**The audio signal is the interesting part of this issue, not the grid.** Checked
before filing: `store.EventKinds()` carries four kinds and none of them is a
play. So the operator's third signal is unrecorded today, and adding it is this
issue's own prerequisite rather than an assumption it can make. `#46` is what
makes it worth having — a play used to be a side effect of looking a word up, and
is now a deliberate click on a specific word.
