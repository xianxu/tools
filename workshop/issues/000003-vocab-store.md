---
id: 000003
status: working
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours: 1.61
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.2  impl=0.04
item: greenfield-go-module     design=0.3  impl=0.32
item: smaller-go-module        design=0.1  impl=0.2
item: milestone-review         design=0.0  impl=0.12
item: milestone-review         design=0.0  impl=0.12
item: atlas-docs               design=0.05 impl=0.06
design-buffer: 0.15
total: 1.61
```

Derivation notes:

- **greenfield-go-module** is the store package: types, two implementations, one
  shared conformance suite. Design takes the ×0.2 spec-quality discount
  (1.5 → 0.3) — the plan fixes the interface, the file layout and the atomicity
  rule. Impl at the top of its band, because atomic writes and skip-and-warn on a
  corrupt file are the parts with real failure modes.
- **smaller-go-module** is `storeHistory` plus the wiring: mirror-shaped, since
  `#14` already consumes the `History` interface and nothing in the editor moves.
- **Two `milestone-review`s.** `#14` budgeted three and used two; `#1` and `#2`
  each needed more than one. This is smaller than `#14`, so two.
- Library-availability check ran: `go.yaml.in/yaml/v3` is already an ariadne
  dependency and is used rather than hand-rolling YAML; no from-scratch halving.
- `familiarity: 1.0` — same package family, same fakes, same test posture.

Σdesign 0.65 × 1.15 = 0.7475; Σimpl 0.86; total **1.61**.

## Plan

See `workshop/plans/000003-vocab-store-plan.md`.

- [ ] `Word`, `ReviewEvent`, `Slug` (fuzzed — it turns text into a filename).
- [ ] `Store` seam + `memStore` + one shared conformance suite.
- [ ] `yamlStore`: one file per word, append-only day logs, atomic writes.
- [ ] `storeHistory` — makes `#14`'s history persistent with no editor change.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
