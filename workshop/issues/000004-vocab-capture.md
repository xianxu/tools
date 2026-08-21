---
id: 000004
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours:
started: 2026-08-21T10:02:22-07:00
---

# capture looked-up words into the deck

## Problem

A vocabulary deck nobody has to curate is the only one that gets used. Every
`define` lookup is already a signal of interest — it should build the deck.

## Spec

`define <word>` records the lookup into the deck.

- **Capture only on a successful lookup.** Typos exit 1 and never enter, so the
  dictionary is its own spam filter and no validation layer is needed.
- Record lookup count and timestamps: a word looked up three times is a stronger
  signal than one looked up once, and `--play` can order by that.
- `define --forget <word>` removes it; `DEFINE_NO_CAPTURE=1` disables capture.
- Capture must never break a lookup: a store failure warns on stderr and leaves
  the exit code alone, exactly as audio failure does today.
- Applies to REPL lookups (#2) too.

## Done when

- [ ] A successful lookup appears in the deck; a failed one does not.
- [ ] Repeat lookups increment the count rather than duplicating the word.
- [ ] `--forget` removes; `DEFINE_NO_CAPTURE=1` suppresses.
- [ ] A failing store degrades to a warning, never a failed lookup.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
