---
id: 000062
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-15
estimate_hours: 2.338
started: 2026-09-15T12:23:43-07:00
---

# define: show language flag before prompt

## Problem

The interactive prompt does not make the current dictionary language obvious, so switching between decks or languages can leave the user unsure which mode is active.

## Spec

Display a Unicode flag before the interactive prompt to denote the effective language. Update it immediately after /lang changes and initialize it from the resolved startup/deck language. Use one shared language-indicator mapping across prompt renderers. Specify a readable fallback for unmapped languages and terminals where flag emoji width differs; preserve cursor, completion and mouse coordinates.

## Done when

- The prompt visibly identifies the effective language at startup and after a language switch.
- Raw/editor and line-mode prompt paths stay consistent.
- Flag display does not misalign editing, completion or mouse selection.
- Unknown language tags have an explicit fallback.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale, so this estimate is provisional.

Units cover issue/spec work, the prompt policy, the bounded display-unit reader,
the column-consumer integration, interactive rendering and tests, docs, one close
review, and PTY verification. Existing utf8/ANSI and terminal seams cover the
work; no new library would replace the narrow RI-pair policy without expanding
Unicode behavior. Implementation hours are 40% of v2/v2.1 table values;
familiarity is 1.0. Thorough-plan design uses the 0.2 multiplier for implementation
units, preserving the already incurred issue/spec allowance, plus 15% design
buffer. The cross-cutting unit covers all column consumers and the TUI unit
includes both loops and regressions; PTY discovery is a separate allowance.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.04 impl=0.12
item: smaller-go-module design=0.06 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: tui-screen design=0.16 impl=0.32
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.04 impl=0.16
item: real-api-discovery design=0 impl=0.12
design-buffer: 0.15
total: 2.338
```

## Plan

- [ ] Approve and gate [the implementation plan](../plans/000062-language-flag-prompt-plan.md), implement shared prompt indicators and atomic flag geometry, and verify both loop paths before close review.

## Log

### 2026-09-14

- Requested by the user as a separate future feature during bilingual Spanish work. Task capture only; implementation has not started.

## Revisions

### 2026-09-15 — Styling sequence and concrete prompt design

User directed #62 first, then #65; #64 remains the separate adaptive-learning
work. Claimed #62 and entered planning. Proposed prompt examples are `🇪🇸 › `
and `🇺🇸 › `, with `[es] › ` as the text fallback. Known mapping covers en→US,
es→ES, it→IT, fr→FR, de→DE, pt→PT, zh→CN, ja→JP, ko→KR; unmapped valid tags
use their code. These are presentation conventions, not a supported-language
registry or a change to pronunciation locale.

Add `-no-flags` for terminals whose flag rendering is unsuitable; `-no-color`
also uses the text fallback. Neither fallback hides the language. The indicator
is derived from effective session language at each render, including committed
input lines; failed language changes retain the old indicator. Pipes and
redirected output keep the existing no-prompt behavior.

Geometry inspection found two regional-indicator runes already sum to two cells,
but clipping, soft wrapping and selection can split the pair. A shared display
unit helper will keep adjacent pairs whole across all column-based consumers.
Independent boundary tests must detect this defect without sharing the helper
as their oracle. The detailed plan awaits operator approval; no implementation
has started.

### 2026-09-15 — Review corrections

At a true terminal width below two columns, render the code fallback. Preserve
whole historical flags through viewport clipping and widening. Added explicit
entry-point tests for saved deck language and `-lang` overriding it, alongside
the loop tests. These address the fresh plan review findings.

### 2026-09-15 — Implementation approved and plan gate format corrected

Operator approved the plan. Addressed PQ-1 by replacing procedural task inventories
with function-level verification strategies. Plain-mode prompt visibility needs
terminal ownership separated from ANSI/raw-mode permission; tests will exercise
resolved startup through real PTY stdout. Behavior remains the approved design.
