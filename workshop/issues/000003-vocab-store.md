---
id: 000003
status: working
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
started: 2026-08-20T17:51:17-07:00
---

# vocabulary store: per-user YAML deck in a brain, behind a Store seam

## Problem

The vocabulary features all need per-user persistent state, and none of them
should know where it lives or what format it is in. A database is the eventual
destination; YAML files in a git repo are the quick, inspectable start.

## Spec

A `Store` seam with a YAML-files-on-disk backend.

- `Store`: `Deck()`, `Upsert(Word)`, `AppendEvent(ReviewEvent)`, `Events(since)`.
- **Location is the working directory** (operator, 2026-08-20). `define` resolves
  no brains, workspaces or home directories: it writes YAML where it was started.
  No config file yet — that arrives if and when `define` needs to run anywhere.
  `NewStore(dir)` takes the directory as a parameter so *who decides `dir`* stays
  a one-line question at the boundary rather than something threaded through.
- **Shape is chosen for a directory that may be replicated.** The operator plans
  to run this inside a synced directory, so the eventual failure mode is a merge
  conflict rather than corruption: one file per word (`words/<slug>.yaml`) plus an
  append-only event log per day (`events/YYYY-MM-DD.yaml`). Neither conflicts; a
  single mutable `vocab.yaml` would, on the second machine. It costs nothing to
  get right now and is expensive to change once files exist.
- **Durable replication is explicitly NOT this issue's concern.** Running `define`
  inside a replicated directory is how state travels between machines; that is a
  choice of cwd, not a feature here. An earlier draft resolved a brain path and
  invoked `nous push` — that was me over-reading "use nous to push" as a design
  requirement when the constraint was only **YAML rather than a database**.
- **Persistent history falls out of this.** `#14` already consumes a `History`
  seam (`Add`, `Prefix`) with a session-scoped implementation. The store satisfies
  that interface, so history becomes durable without the editor changing — and
  without the private history file `#15` forbids.
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
