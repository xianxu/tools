---
id: 000060
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
started: 2026-09-14T11:00:40-07:00
---

# define: highlight the active board marking option

## Problem

The active board marking option is dimmed with the surrounding instructions.

## Spec

Highlight the bracketed active marking option in the existing cyan heading color. Clear dim before the option and restore dim afterward. Preserve wording, visible width, no-color output, and the small-window refusal.

The highlight follows the existing board mode as Tab cycles yes, no, and drop. It does not change grading colors on marked words or alter input handling. Styling belongs to the caller that already owns terminal colors.

## Done when

- All three marking modes highlight only their active option; surrounding chrome remains dim and tests pass.

## Plan

- [x] Style the selected span at the board prompt presentation seam using the existing palette and bracketed wording (ARCH-DRY, ARCH-PURE).
- [x] Verify all modes, no-color, refusal, and live frame styling; run define tests and close review.

## Log

### 2026-09-14

- Small presentation-only correction: no new state, IO, resources, durable artifacts, or width changes. Existing mode owner and frame lifecycle remain authoritative. Skip the plan judge and estimate for this trivial styling correction; retain the close review. Reuse established palette rather than board grading colors (ARCH-PURPOSE).

- Verified regression fails for all three modes before styling and passes afterward. Full `go test ./cmd/define/...` passed (112.055s); `go vet ./cmd/define/...` and diff whitespace checks passed. Live chrome test confirms the Draw wiring; atlas documents the active-span exception.
