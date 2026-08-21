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
- **Location is config, defaulting to the working directory** (operator,
  2026-08-20). `define` does not resolve brains, workspaces or home directories:
  it writes YAML where it was started, and a config file can override that.
- **Durable replication is explicitly NOT this issue's concern.** Running
  `define` inside a directory that happens to be replicated is how state travels
  between machines; that is the operator's choice of cwd, not a feature here. The
  earlier draft resolved a brain path and invoked `nous push` — that was me
  over-reading "use nous to push" as a design requirement when the actual
  constraint was only **YAML files rather than a database**.
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
