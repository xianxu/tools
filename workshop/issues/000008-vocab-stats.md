---
id: 000008
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-09-07
estimate_hours:
started: 2026-09-07T19:04:15-07:00
---

# define --stats: deck, streak and mastery statistics

## Problem

Whether any of this is working should be visible in one screen.

## Spec

`define --stats`.

- Words known, added per day, mastered, active days, current and longest streak,
  accuracy by form.
- **All derived from the append-only event log** — a second reason for that
  shape (#3). No separate counters to keep in sync and no drift.
- Pure aggregation over events + clock → a stats struct; rendering is separate,
  so the numbers are testable without parsing a screen.

## Done when

- [ ] Every figure derived from events, none stored separately.
- [ ] Streak arithmetic verified across timezone boundaries and gaps with a fake
      clock.
- [ ] Renders sensibly on an empty deck.

## Plan

Durable design: `workshop/plans/000008-vocab-stats-plan.md`.

Single-pass: one boundary, plain checkboxes (AGENTS.md §3 — tagging Mx would
force a redundant milestone-close on atomic work).

- [ ] The fold — Stats, Summarise, the two streaks, accuracy by form.
- [ ] The screen — renderStats, the -stats flag, docs.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
