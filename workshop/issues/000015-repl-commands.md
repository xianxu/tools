---
id: 000015
status: working
deps: [tools#14, tools#3]
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours: 1.82
started: 2026-08-21T13:55:32-07:00
---

# REPL command mode: /-prefixed commands with type-ahead, starting with /history

## Problem

Once the REPL has an editor, it needs a way to do things that are not "define
this word" — and a namespace that cannot collide with a word being looked up.

## Spec

A `/` in the first column switches the line to **command mode**.

- Typing `/` shows the available commands; further characters narrow them
  (type-ahead), Tab or Return accepts.
- `/` is unambiguous: no English headword starts with it, so command mode never
  competes with a lookup. (`define` already handles multi-word headwords like
  `hot dog`, so the namespace has to be a character, not a word.)
- An unknown command reports the near matches rather than silently defining
  something.

### First command: `/history`

Words queried **in the last two days**, deduped, sorted reverse-chronologically
by *first* lookup (operator, 2026-08-20 — first, not most recent, so a word you
keep returning to holds its original position rather than churning to the top).

- **Reads the store (`#3`), not a private history file.** `#4` already records
  every successful lookup with timestamps; a second history mechanism would drift
  from it and put the same fact in two places (ARCH-DRY). This issue therefore
  waits on `#3` rather than inventing storage.
- One wrinkle to settle in the plan: the deck records **successful** lookups
  only, but the *editor's* history (`#14`) should recall what you typed, including
  typos. Decide whether they are one source with a success flag, or two —
  `/history` is about words queried, so it reads the deck's view either way.
- The window is `--days N`, defaulting to 2.

## Done when

- [ ] `/` opens command mode; type-ahead narrows; an unknown command suggests.
- [ ] `/history` lists the last two days, deduped, reverse-sorted by first lookup.
- [ ] It reads the store, with no second history mechanism anywhere.
- [ ] A word starting with `/` is impossible to look up — confirmed as acceptable.
- [ ] Adding a second command needs no change to the dispatch loop.

## Plan

**Design of record: `workshop/plans/000015-repl-commands-plan.md`.** That file
owns the task breakdown; this section carries only the boundaries, so there is
one place to revise when the design moves.

Two boundaries on purpose. `#4` closed as one batch and took ten review rounds;
six findings fixed in one commit is six chances to create a seventh.

- [ ] M1 — the `/` namespace and dispatch (plan Tasks 1-3). Complete without the
      store: `/` opens command mode, type-ahead narrows, unknown commands
      suggest, `/help` lists the table. Dispatch lands on `parseREPLLine`'s new
      `cmdCommand` kind so BOTH loops get it.
- [ ] M2 — `/history` (plan Tasks 4-8). One clock on `storeDeps`, the local-time
      window, the summary, `runHistory`, docs.

## Estimate

Method A, primitive decomposition against the plan's eight tasks. Design takes
v2's ×0.2 spec-quality discount (the plan doc resolves the decisions), so the
buffer is v2.1's +15% rather than +30% — layering both would double-count the
same thoroughness. `impl=` is v3.1's 40% of the v2 table.

| task | primitive | design (table → ×0.2) | impl (table → ×0.4) |
|---|---|---|---|
| 1 command table + parsing | smaller-go-module | 0.3 → 0.06 | 0.4 → 0.16 |
| 2 `completionsFor` switch | smaller-go-module | 0.3 → 0.06 | 0.4 → 0.16 |
| 3 `cmdCommand` + both loops | smaller-go-module | 0.3 → 0.06 | 0.4 → 0.16 |
| M1 boundary | milestone-review | 0.1 → 0.02 | 0.35 → 0.14 |
| 4 clock on `storeDeps` | cross-cutting-refactor | 0.3 → 0.06 | 0.3 → 0.12 |
| 5 `historyWindow` + args | smaller-go-module | 0.3 → 0.06 | 0.5 → 0.20 |
| 6 `summariseLookups` | smaller-go-module | 0.3 → 0.06 | 0.4 → 0.16 |
| 7 `runHistory` e2e | smaller-go-module | 0.3 → 0.06 | 0.5 → 0.20 |
| 8 docs + atlas | atlas-docs | 0.1 → 0.02 | 0.15 → 0.06 |
| M2 boundary + close | milestone-review | 0.1 → 0.02 | 0.35 → 0.14 |

No library check applies (Step 2.5): the only non-obvious primitive is local-day
arithmetic, which the standard library covers with `time.Date`/`AddDate` — the
work is choosing the right call, not writing one.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module       design=0.30 impl=0.88
item: milestone-review        design=0.04 impl=0.28
item: cross-cutting-refactor  design=0.06 impl=0.12
item: atlas-docs              design=0.02 impl=0.06
design-buffer: 0.15
total: 1.82
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

**Known bias, recorded rather than corrected for.** `tools#4` closed at 6.56h
against a 1.31h estimate (0.2×), almost entirely in review rounds that
`milestone-review`'s 0.2–0.5h impl does not model — ten boundary rounds on one
issue. The honest move is to apply the method as written and let the calibration
ledger see the miss (`#127` tracks the recalibration; `estimate-source` already
reports the doc as stale), not to inflate this number until it looks right.

## Log

### 2026-08-21 — gates

Plan-quality cleared in 2 rounds. Round 1 raised 3 Important; **PQ-2 was `#4`'s
defect in design form** — dispatch placed in `submitLine`, which only the raw
editor reaches, so `echo /history | define` would have gone to the dictionary.
Fixed by moving it onto `parseREPLLine`, the classifier both loops already share.
PQ-1 (a clock nothing provided) and PQ-3 (unbounded `--days` silently normalised
by `AddDate`) also fixed. See the plan's `## Revisions`.

Estimate-quality returned INFO. Its advisory notes, carried here so the close
review sees them rather than re-deriving them:

- Plan authoring and the plan-gate rounds sit inside the `sdlc actual` window
  (anchored at the claim) but are modelled by no `item:` row — a structural
  est/actual gap that grows with each pre-implementation round.
- `milestone-review` at 0.14h impl per boundary is the number most likely to
  drive a miss, given `#4` spent ten rounds in the same package. Flag it as a
  `milestone-review`-attributable row in `#117`'s ledger at close, not just as a
  whole-issue ratio.
- `M2 boundary + close` is priced like `M1 boundary`, though `sdlc close` carries
  gates M1 does not (actual measurement, atlas, project tick, verified evidence).
- The two required mutation checks and the manual raw-terminal pass are real work
  no row covers (~0.1–0.2h).

Left as derived rather than corrected: inflating an estimate until it looks right
is what makes calibration data useless.

### 2026-08-20

Created as part of the `define-learn` project.
