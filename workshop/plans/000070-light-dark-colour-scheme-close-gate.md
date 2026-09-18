---
gate: boundary-review
issue: 70
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-17T22:58:39-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: 'The plan restates the diff: full implementations, test cases listed in prose, call sites by line number'
          detail: |-
            The gate asks for named functions plus one strategy line per risky function. It does not block this plan, which can be carried out and whose facts were checked; compress in future plans rather than rewriting this one.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: plan-restates-code
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-17T22:58:39-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Minor
          title: schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
          detail: 'The spec''s choice *{value, source: flag|saved|session} is implemented as choice plus chosenBy schemeSource with sourceDefault as the absent marker (cmd/define/scheme.go:30,49). M2''s describeScheme will print this source to the user; give the choice its own source type or use *schemeChoice before M2 builds on it (ARCH-ORDER).'
          family: state-shape-admits-illegal-combinations
          round: 2
        - id: BR-3
          severity: Minor
          title: Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
          detail: TestBoardFooterPutsTheFormsOwnRowsFirst and the fitsABoard row count (play_loop_test.go:2417,3049) call boardFooter (render_helpers_test.go:82), which also now holds the only copy of the load-bearing rationale. Point them at paintedBoardFooterForTest, move the comment to boardFooterOutput, and delete boardFooter (ARCH-DRY). Pre-existing; M1 moved it into a test file.
          family: tests-pin-a-shadow-of-the-live-path
          round: 2
        - id: BR-4
          severity: Minor
          title: The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
          detail: atlas/define.md:503-515 describes the terminal-reported value and the saved and session sources as if they worked now, but none has a production caller until M2/M3. The plan says docs should describe only what each milestone ships; add one clause saying so.
          family: docs-describe-unshipped-surface
          round: 2
        - id: BR-5
          severity: Minor
          title: Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
          detail: The Log's Task 3 entry explains why, but the list of spec revisions the plan makes omits it, so the issue's Done-when (line 320) and Spec section on Repaint still claim what was not done.
          family: spec-done-when-drifts-from-delivery
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
---

# Gate ledger — tools#70 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T22:58:39-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-restates-code` The plan restates the diff: full implementations, test cases listed in prose, call sites by line number
  The gate asks for named functions plus one strategy line per risky function. It does not block this plan, which can be carried out and whose facts were checked; compress in future plans rather than rewriting this one.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-09-17T22:58:39-07:00 (claude) — passed

### Raised

- **BR-2** [Minor] `state-shape-admits-illegal-combinations` schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
  The spec's choice *{value, source: flag|saved|session} is implemented as choice plus chosenBy schemeSource with sourceDefault as the absent marker (cmd/define/scheme.go:30,49). M2's describeScheme will print this source to the user; give the choice its own source type or use *schemeChoice before M2 builds on it (ARCH-ORDER).
- **BR-3** [Minor] `tests-pin-a-shadow-of-the-live-path` Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
  TestBoardFooterPutsTheFormsOwnRowsFirst and the fitsABoard row count (play_loop_test.go:2417,3049) call boardFooter (render_helpers_test.go:82), which also now holds the only copy of the load-bearing rationale. Point them at paintedBoardFooterForTest, move the comment to boardFooterOutput, and delete boardFooter (ARCH-DRY). Pre-existing; M1 moved it into a test file.
- **BR-4** [Minor] `docs-describe-unshipped-surface` The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
  atlas/define.md:503-515 describes the terminal-reported value and the saved and session sources as if they worked now, but none has a production caller until M2/M3. The plan says docs should describe only what each milestone ships; add one clause saying so.
- **BR-5** [Minor] `spec-done-when-drifts-from-delivery` Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
  The Log's Task 3 entry explains why, but the list of spec revisions the plan makes omits it, so the issue's Done-when (line 320) and Spec section on Repaint still claim what was not done.

## Open findings

- **BR-1** [Minor] `plan-restates-code` The plan restates the diff: full implementations, test cases listed in prose, call sites by line number
- **BR-2** [Minor] `state-shape-admits-illegal-combinations` schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
- **BR-3** [Minor] `tests-pin-a-shadow-of-the-live-path` Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
- **BR-4** [Minor] `docs-describe-unshipped-surface` The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
- **BR-5** [Minor] `spec-done-when-drifts-from-delivery` Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
