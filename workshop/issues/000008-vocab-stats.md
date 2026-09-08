---
id: 000008
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-09-07
estimate_hours: 3.71
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source`, so the per-primitive hours are provisional; they are
derived against `#12` (4.54/3.70) and `#46` (7.02/5.91), the two most recent
closes in this repo, using the same primitives on the same codebase.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.30 impl=0.06
item: greenfield-go-module     design=0.06 impl=0.28
item: smaller-go-module        design=0.03 impl=0.14
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.02 impl=0.12
item: greenfield-go-module     design=0.04 impl=0.20
item: smaller-go-module        design=0.03 impl=0.14
item: cross-cutting-refactor   design=0.03 impl=0.16
item: atlas-docs               design=0.02 impl=0.06
item: smaller-go-module        design=0.0  impl=0.20
item: ux-rename-iteration      design=0.0  impl=0.15
item: milestone-review         design=0.0  impl=0.60
item: milestone-review         design=0.0  impl=0.85
design-buffer: 0.15
total: 3.71
```

| row | the work |
|---|---|
| `issue-spec` 0.30/0.06 | the Spec was written in August and needed no rework; three plan-quality rounds, all of them about CITATION discipline rather than design. Below `#12`'s 0.35 because the design was settled from the first draft — what changed was the evidence for its claims. |
| `greenfield-go-module` 0.06/0.28 | `schedule/stats.go`: `Stats`, `Summarise`, `FormAccuracy`, `activeDays`. |
| `smaller-go-module` 0.03/0.14 | the two streaks, over `StartOfDay`/`DaysBetween` rather than re-derived. |
| `smaller-go-module` 0.02/0.12 | accuracy by form and added-per-day. |
| `smaller-go-module` 0.02/0.12 | the ARCH-SECURE skips — zero `At`, future `At`, control runes in `Form` — each pinned. |
| `greenfield-go-module` 0.04/0.20 | `cmd/define/stats.go`: `renderStats` and the empty-deck screen. |
| `smaller-go-module` 0.03/0.14 | `runStats`: the shell, the exit codes, `noDeckMessage`. |
| `cross-cutting-refactor` 0.03/0.16 | the mode registration, plus **making `TestModeCollision` actually derive** — a live bug this issue trips, so it is work this issue owns rather than a side quest. |
| `atlas-docs` 0.02/0.06 | README's command list and the atlas. |
| `smaller-go-module` 0.0/0.20 | the mutation sweeps: `DaysBetween` against a Duration fold, each skip, the mode guard. |
| `ux-rename-iteration` 0.0/0.15 | the hand-run. `#12`'s deviation 3 records that pricing this at nothing is the omission `#10` had already made, and a stats SCREEN is the case where it matters most. |
| `milestone-review` 0.0/0.60 + 0.0/0.85 | the close boundary, run + remediation — the house pair `#12` and `#42` set. |

**Reconciliation.** Σdesign = 0.55, Σimpl = 3.08.
0.55 × 1.15 + 3.08 × 1.0 = **3.71**.

**Read against the trailing record.** This repo's ledger: `#42` 0.36, `#44` 0.47,
`#10` 0.62, `#12` 1.23, `#46` 1.18 — median ≈ 0.62, which at 3.71 predicts
roughly 6h actual. That gap is the within-session parallelism `#117`'s ledger
exists to instrument, not a reason to inflate the primitives; multiplying the
rows to meet it would destroy the only signal the ledger carries.

**Why this is the smallest issue in the project.** Single-pass, one new package
file, and the two hardest parts — the DST-correct calendar and the mastery
predicate — are being CALLED rather than written. The plan's first section is
what the issue refuses to reimplement, and that is most of why the number is low.

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

## Revisions

### 2026-09-07 — two Done-when rows, read precisely

Both are deviations the plan makes deliberately; recorded here so the close gate
reads criteria that match what shipped rather than a plan contradicting them.

**"Every figure derived from events, none stored separately" — the second half is
the invariant, the first is a shorthand.** `Known` and `Mastered` come from the
DECK (folded through `schedule.Fold` for the box, but enumerated from the deck),
because the log alone cannot answer them: `--forget` removes a word and
deliberately leaves its events, so a log-only count reports words the learner has
deleted. Nothing is STORED — no counter, no cache, no second source of truth —
which is what the row exists to protect. `AddedPerDay` does count from the log,
for the opposite reason: it asks what HAPPENED, and a forgotten word was still
added that day.

**"Streak arithmetic verified... with a fake clock" — `now` is a PARAMETER
instead.** `Summarise(events, deck, now)` is pure, so every timezone and DST case
is a table row rather than a clock double. That is stronger than the row asks
for: no interface to inject, no fake to keep honest, and the DST days are named
constants in the test. `store.FixedClock` remains available and is simply not
needed at this seam.
