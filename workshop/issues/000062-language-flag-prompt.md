---
id: 000062
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-15
estimate_hours:
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
