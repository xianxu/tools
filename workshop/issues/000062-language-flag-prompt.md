---
id: 000062
status: open
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
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

- [ ] Design the shared indicator and flag/fallback mapping, implement prompt wiring, and verify language changes and terminal geometry.

## Log

### 2026-09-14

- Requested by the user as a separate future feature during bilingual Spanish work. Task capture only; implementation has not started.
