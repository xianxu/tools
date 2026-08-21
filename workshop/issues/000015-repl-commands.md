---
id: 000015
status: working
deps: [tools#14, tools#3]
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours:
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

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
