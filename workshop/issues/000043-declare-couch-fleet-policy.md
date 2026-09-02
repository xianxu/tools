---
id: 000043
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T20:23:56-07:00
---

# Declare Couch fleet policy

## Problem

Couch refuses to start an actor in the tools repository because tools has no
`.sdlc/fleet.json` declaration. Fleet admission intentionally fails closed when
the repository has not declared its concurrency policy.

## Spec

Declare tools as a repo-keyed fleet with capacity one. When one tools actor is
active, Couch rejects another actor for any tools worktree; parked actors do not
consume capacity.

## Done when

- `sdlc fleet policy --path /Users/xianxu/workspace/tools --json` returns a valid repo-keyed, capacity-one policy.
- Couch can proceed past fleet-policy resolution when starting a tools actor.

## Plan

- [ ] Add `.sdlc/fleet.json` using the established repo-keyed bounded policy.
- [ ] Validate the declaration through `sdlc fleet policy`.

## Log

### 2026-09-01

The operator approved the same single-thread-per-repository declaration used by
Pair and Parley. This keeps the admission fact in the repository-owned policy
source (`ARCH-DRY`).
