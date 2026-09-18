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
    - "n": 3
      timestamp: "2026-09-17T23:46:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The fix belongs in future plans, and no lessons.md rule records it; Chunk 1's code blocks have already drifted from the code (the plan's own Revisions says so), which shows the cost. Minor, not blocking.
          round: 3
      findings:
        - id: BR-6
          severity: Important
          title: M2 plan steps are ticked but the issue Log has no M2 entry
          detail: Task 7 Step 6 says the WriteScheme symlink-to-regular-file behaviour is "noted in the Log"; it is not. Task 11 Step 4 claims the TestPTY conformance run PASSED and should name what ran and what was skipped, but the M1 Log records 13 TestPTY tests already failing, so a clean PASS contradicts it. The mutation record the Done-when requires is also missing. Add an M2 Log entry covering the symlinked-file behaviour, the pty run broken into passed, pre-existing failures and skipped, and the mutation list.
          family: ticked-step-lacks-its-evidence
          round: 3
        - id: BR-7
          severity: Minor
          title: commandCtx session/fullScreen and schemeArg auto/value allow combinations that mean nothing
          detail: 'This is the 2nd finding in this family. Rule: when fields depend on each other, they should be ONE tagged value whose members are exactly the legal combinations. Remaining instances in #70: commandCtx.session plus fullScreen (fullScreen without session is representable), and schemeArg auto plus value (its zero value would save a blank line, which every later startup warns about). Neither is reachable today. Measured prevalence in #70: 3 instances; choice/chosenBy was fixed at M1. Fix: a loopKind enum {oneShot, piped, editor}, and a nil-able choice where nil means auto.'
          family: state-shape-admits-illegal-combinations
          round: 3
        - id: BR-8
          severity: Minor
          title: The startup read skips schemePersister, and the precedence order is tested only through a pty
          detail: 'run() resolves d.configDir and calls store.ReadScheme inline, while deps.schemePersister resolves it again for save and clear (ARCH-DRY). The flag, then saved, then default order lives in run() glue; "flag beats saved" and "garbled file warns once" are pinned only by TestSavedSchemeGovernsALookup through a real pty (ARCH-PURE). Fix: add load() to schemePersister and extract a pure initialSchemeState(flag, persister, warn) that can be unit-tested without a pty.'
          family: store-access-bypasses-its-seam
          round: 3
      boundary: M2
      recipe: milestone-review
      blocked: true
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

## Round 3 — 2026-09-17T23:46:34-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The fix belongs in future plans, and no lessons.md rule records it; Chunk 1's code blocks have already drifted from the code (the plan's own Revisions says so), which shows the cost. Minor, not blocking.

### Raised

- **BR-6** [Important] `ticked-step-lacks-its-evidence` M2 plan steps are ticked but the issue Log has no M2 entry
  Task 7 Step 6 says the WriteScheme symlink-to-regular-file behaviour is "noted in the Log"; it is not. Task 11 Step 4 claims the TestPTY conformance run PASSED and should name what ran and what was skipped, but the M1 Log records 13 TestPTY tests already failing, so a clean PASS contradicts it. The mutation record the Done-when requires is also missing. Add an M2 Log entry covering the symlinked-file behaviour, the pty run broken into passed, pre-existing failures and skipped, and the mutation list.
- **BR-7** [Minor] `state-shape-admits-illegal-combinations` commandCtx session/fullScreen and schemeArg auto/value allow combinations that mean nothing
  This is the 2nd finding in this family. Rule: when fields depend on each other, they should be ONE tagged value whose members are exactly the legal combinations. Remaining instances in #70: commandCtx.session plus fullScreen (fullScreen without session is representable), and schemeArg auto plus value (its zero value would save a blank line, which every later startup warns about). Neither is reachable today. Measured prevalence in #70: 3 instances; choice/chosenBy was fixed at M1. Fix: a loopKind enum {oneShot, piped, editor}, and a nil-able choice where nil means auto.
- **BR-8** [Minor] `store-access-bypasses-its-seam` The startup read skips schemePersister, and the precedence order is tested only through a pty
  run() resolves d.configDir and calls store.ReadScheme inline, while deps.schemePersister resolves it again for save and clear (ARCH-DRY). The flag, then saved, then default order lives in run() glue; "flag beats saved" and "garbled file warns once" are pinned only by TestSavedSchemeGovernsALookup through a real pty (ARCH-PURE). Fix: add load() to schemePersister and extract a pure initialSchemeState(flag, persister, warn) that can be unit-tested without a pty.

## Open findings

- **BR-1** [Minor] `plan-restates-code` The plan restates the diff: full implementations, test cases listed in prose, call sites by line number
- **BR-2** [Minor] `state-shape-admits-illegal-combinations` schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
- **BR-3** [Minor] `tests-pin-a-shadow-of-the-live-path` Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
- **BR-4** [Minor] `docs-describe-unshipped-surface` The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
- **BR-5** [Minor] `spec-done-when-drifts-from-delivery` Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
- **BR-6** [Important] `ticked-step-lacks-its-evidence` M2 plan steps are ticked but the issue Log has no M2 entry
- **BR-7** [Minor] `state-shape-admits-illegal-combinations` commandCtx session/fullScreen and schemeArg auto/value allow combinations that mean nothing
- **BR-8** [Minor] `store-access-bypasses-its-seam` The startup read skips schemePersister, and the precedence order is tested only through a pty
