---
id: 000008
status: open
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
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

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
