---
id: 000003
status: open
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# vocabulary store: per-user YAML deck in a brain, behind a Store seam

## Problem

The vocabulary features all need per-user persistent state, and none of them
should know where it lives or what format it is in. A database is the eventual
destination; YAML files in a git repo are the quick, inspectable start.

## Spec

A `Store` seam with a YAML-in-a-brain backend.

- `Store`: `Deck()`, `Upsert(Word)`, `AppendEvent(ReviewEvent)`, `Events(since)`.
- **Shape is chosen for git.** A brain syncs across machines, so the failure mode
  is a merge conflict, not corruption: one file per word (`words/<slug>.yaml`)
  plus an append-only event log per day (`events/YYYY-MM-DD.yaml`). Neither
  conflicts. A single mutable `vocab.yaml` would conflict on every second machine.
- Per-user by construction — a brain is one person's repo. `nous push`
  ("checkpoint and push the brain containing the current directory") provides
  sync, so **no sync code is written here**.
- Location `$BRAIN/data/vocab/`, resolved via nous; `DEFINE_VOCAB_DIR` overrides.
- **Clock injected from day one.** Spaced repetition is entirely date-driven and
  "due today" is untestable against a wall clock. No component may call
  `time.Now()` directly.
- Fakes: in-memory `Store` for consumers; the YAML backend is tested against a
  temp dir, and a conformance test asserts the two agree.

## Done when

- [ ] Both backends satisfy the same seam and pass one shared conformance suite.
- [ ] A word written on two "machines" (two checkouts) merges without conflict.
- [ ] No production code calls `time.Now()`; the clock is injected everywhere.
- [ ] Store survives an interrupted write (append-only log is never truncated).

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
