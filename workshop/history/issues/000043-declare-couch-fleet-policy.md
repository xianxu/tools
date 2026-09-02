---
id: 000043
status: done
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T20:23:56-07:00
actual_hours: N/A
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

The declaration uses the established version-1 fleet schema already consumed
by `sdlc fleet policy` and Couch. Repository identity is the admission key so a
linked worktree cannot bypass the single active actor limit. Capacity overflow
is rejected rather than provisioning another worktree. No Couch fallback or
repo-name inference is introduced; the checked-in file remains the sole
authority for tools admission behavior.

## Done when

- `sdlc fleet policy --path /Users/xianxu/workspace/tools --json` returns a valid repo-keyed, capacity-one policy.
- Couch can proceed past fleet-policy resolution when starting a tools actor.

## Plan

- [x] Add `.sdlc/fleet.json` using the established repo-keyed bounded policy.
- [x] Validate the declaration through `sdlc fleet policy`.

## Log

### 2026-09-01
- 2026-09-01: closed — One-file repository configuration; jq validated the exact declaration and sdlc fleet policy --path /Users/xianxu/workspace/tools --json returned ok=true with repo-keyed bounded capacity 1 and reject overflow. Actual-time telemetry was unavailable for this worktree, so calibration is not applicable.; review verdict: SHIP

The operator approved the same single-thread-per-repository declaration used by
Pair and Parley. This keeps the admission fact in the repository-owned policy
source (`ARCH-DRY`).

Added the declaration and validated its normalized result through the same
`sdlc fleet policy` boundary Couch invokes.
