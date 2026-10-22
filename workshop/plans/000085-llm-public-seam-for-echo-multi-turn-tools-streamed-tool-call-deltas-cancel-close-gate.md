---
gate: boundary-review
issue: 85
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: cec2c4eb-6a5c-4bab-9f90-62516561042d
            repository: github.com/xianxu/tools
            issue_id: "000085"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/0a791a0ee029577e
            reviewed_head: 12082073a2925d1c8cbf8086d62118446152a51b
            slot: tools:1
            requested_at: "2026-10-22T03:47:14.017665Z"
          phase: finished
          pid: 95561
          started_at: "2026-10-22T03:47:14.192099Z"
          finished_at: "2026-10-22T03:54:18.655298Z"
          session_id: aabeb59d-dd8b-421e-b6ed-b66bc7cc7503
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/tools
        code_head: 12082073a2925d1c8cbf8086d62118446152a51b
        tracker_head: 457f32c6d49af2c20dee3bbf35c2e485059a93ca
        branch_point: 2ff2d9d484a5b709721375699cf377ffb199dd43
        since: "2026-10-21T20:24:53-07:00"
        until: "2026-10-21T20:46:44-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T20:24:53-07:00"
              end: "2026-10-21T20:46:44-07:00"
              began: "2026-10-21T20:24:53-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/tools-slot1/tools
        sources:
            - harness: claude
              id: 38db0b39-19d1-4212-8c30-70640a7a3391
              slot: tools:1
              cwd: /Users/xianxu/workspace/worktree/tools-slot1/tools
              digest: 823c1ed7c8df91dc49c5fb8ef8981a9a94d246871c0b8968d2bc30cffd160527
              started: "2026-10-22T03:23:53.951Z"
              spawned: false
              bytes: 2263991
              windows:
                - start: "2026-10-21T20:24:53-07:00"
                  end: "2026-10-21T20:46:44-07:00"
      "n": 1
      timestamp: "2026-10-21T20:54:19-07:00"
      dispatched: "2026-10-21T20:47:14-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Minor
          title: Path rewrite changed historical accounts to name `llm` where the past measurement was against `internal/llm`
          detail: conformance/conformance.go package doc ("Measured, ... ./llm/... reported ok"), conformance/guard_test.go round-4 comment, and atlas/define.md:2772 now describe past events with a path that did not exist then; keep `internal/llm` (or annotate "then internal/llm") in historical narrative.
          family: historical-prose-rewritten-by-mechanical-rename
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Plan's Milestones list still shows `- [ ] M1` while the issue marks M1 done and every Task 1 step is checked
          detail: workshop/plans/000085-llm-public-seam-plan.md line 369; tick it at milestone close so plan and issue agree.
          family: tracker-checkbox-state-diverges
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: llm/conformance_test.go places the repo-local conformance import inside the stdlib import group
          detail: gofmt-clean but not goimports-grouped; move it into the github.com/xianxu/tools group.
          family: import-grouping
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: 12082073a2925d1c8cbf8086d62118446152a51b
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 394b0419-49e1-4d99-a9cd-e7752a97b1b1
            repository: github.com/xianxu/tools
            issue_id: "000085"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/d10f3a74e7609fba
            reviewed_head: 248c58864b4c19ce4bd4e2f860eca3e0f7d79f31
            slot: tools:1
            requested_at: "2026-10-22T04:20:59.954303Z"
          phase: finished
          pid: 16776
          started_at: "2026-10-22T04:21:00.047884Z"
          finished_at: "2026-10-22T04:31:26.326439Z"
          session_id: 311ecca3-7410-46da-b1fb-c132f79e41b4
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/tools
        code_head: 248c58864b4c19ce4bd4e2f860eca3e0f7d79f31
        tracker_head: 457f32c6d49af2c20dee3bbf35c2e485059a93ca
        branch_point: 2ff2d9d484a5b709721375699cf377ffb199dd43
        since: "2026-10-21T20:24:53-07:00"
        until: "2026-10-21T21:20:40-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T20:24:53-07:00"
              end: "2026-10-21T21:20:40-07:00"
              began: "2026-10-21T20:24:53-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/tools-slot1/tools
        sources:
            - harness: claude
              id: 38db0b39-19d1-4212-8c30-70640a7a3391
              slot: tools:1
              cwd: /Users/xianxu/workspace/worktree/tools-slot1/tools
              digest: ed71e9aee8f1033fdac710a2939df00b36a220877522bfa0bca1a03f83b8ad6f
              started: "2026-10-22T03:23:53.951Z"
              spawned: false
              bytes: 3939023
              windows:
                - start: "2026-10-21T20:24:53-07:00"
                  end: "2026-10-21T21:20:40-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-21T20:47:14-07:00"
              finalized: "2026-10-21T20:54:19-07:00"
              finalizing: true
      "n": 2
      timestamp: "2026-10-21T21:31:27-07:00"
      dispatched: "2026-10-21T21:20:59-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-4
          severity: Critical
          title: Cancelled/truncated/malformed stream's partial Response keeps the half-streamed tool_use in Blocks and Message()
          detail: 'llm/anthropic.go Stream partial() replaces ToolCalls with deltas.done but builds Blocks from the SDK accumulator, which adds tool_use at content_block_start. Reproduced on the fake: Blocks[2] tool_use tu_2 has ToolCall.Input "{\"cit" (invalid JSON), Raw input {}, and Message() echoes it back, which breaks the "dropped, not half-reported" contract. Fix: keep a tool_use block only if its ID is in deltas.done. Regression: assert Blocks in TestCancelOutcomes and on a malformed-End stream.'
          family: partial-response-reports-incomplete-state
          round: 2
        - kind: defect
          id: BR-5
          severity: Important
          title: Suite cancel obligations (single cancel and ConcurrentStreams victim) assert ErrCanceled on a race the plan declares legal (ARCH-ORDER)
          detail: 'llm/llmtest/suite.go cancels inside the callback and requires ErrCanceled. The fake holds the stream with FinishRelease, so that run is deterministic. Live (conformance_test.go:41, M3), a buffered short echo stream can legitimately finish first, and Stream returns success. Fix: on the live path accept ErrCanceled or a complete, consistent success; keep the strict check where the double holds the stream.'
          family: test-observes-one-interleaving
          round: 2
        - kind: defect
          id: BR-6
          severity: Minor
          title: Ticked plan Tasks 2-4 quote commit subjects that were never made (work landed as 46a25c8 and 33f3c3a)
          detail: 'This is the 2nd finding in family tracker-checkbox-state-diverges. Rule: a ticked plan item may quote only artifacts that exist at the head, whether test names, commit subjects or milestone states. Extend the TestPlanCitesTestsThatExist guard to quoted commit subjects, or stop quoting planned commit names in task items. Prevalence: 3 of 3 M2 implementation tasks.'
          family: tracker-checkbox-state-diverges
          round: 2
        - kind: defect
          id: BR-7
          severity: Minor
          title: Complete accepts non-object tool_use input that Stream rejects with ErrMalformed
          detail: buildResponse does not apply the JSON-object check that deltaState.next enforces at End, so the same tool_use input succeeds through Complete and fails through Stream.
          family: inconsistent-validation-across-paths
          round: 2
        - kind: defect
          id: BR-8
          severity: Minor
          title: atlas/llm.md still documents the pre-M2 seam (func(string) Stream, no ErrCanceled or BeforeAttempt)
          detail: M3 (Task 6) explicitly schedules the rewrite. Recorded so the close gate checks it; it does not block M2.
          family: docs-deferred-to-later-milestone
          round: 2
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9ea61b07-7db7-4841-9713-a8a12aaf1f94
            repository: github.com/xianxu/tools
            issue_id: "000085"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/eb13b7b5b9ab1110
            reviewed_head: 9c6d73ae3ff35b36ee8a15f45d2556704445d731
            slot: tools:1
            requested_at: "2026-10-22T04:34:27.826327Z"
          phase: finished
          pid: 19866
          started_at: "2026-10-22T04:34:28.270213Z"
          finished_at: "2026-10-22T04:36:54.116718Z"
          session_id: c09149f5-73ef-4e5f-b559-4cef52b5b6ff
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/tools
        code_head: 9c6d73ae3ff35b36ee8a15f45d2556704445d731
        tracker_head: 457f32c6d49af2c20dee3bbf35c2e485059a93ca
        branch_point: 2ff2d9d484a5b709721375699cf377ffb199dd43
        since: "2026-10-21T20:24:53-07:00"
        until: "2026-10-21T21:33:41-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T20:24:53-07:00"
              end: "2026-10-21T21:33:41-07:00"
              began: "2026-10-21T20:24:53-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/tools-slot1/tools
        sources:
            - harness: claude
              id: 38db0b39-19d1-4212-8c30-70640a7a3391
              slot: tools:1
              cwd: /Users/xianxu/workspace/worktree/tools-slot1/tools
              digest: b24392788e21f1e5b20d360e45560f2a0a3859fe50dcdf93b26a86910a9629c7
              started: "2026-10-22T03:23:53.951Z"
              spawned: false
              bytes: 4198215
              windows:
                - start: "2026-10-21T20:24:53-07:00"
                  end: "2026-10-21T21:33:41-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-21T20:47:14-07:00"
              finalized: "2026-10-21T20:54:19-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-21T21:20:59-07:00"
              finalized: "2026-10-21T21:31:27-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-21T21:36:55-07:00"
      dispatched: "2026-10-21T21:34:27-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-4
          disposition: addressed
          note: partial() filters Blocks via deltaState.complete; reverting it makes TestCancelOutcomes and TestToolInputMustBeAnObjectOnBothPaths fail (Input {"cit / [1,2]).
          round: 3
        - id: BR-5
          disposition: addressed
          note: Suite accepts ErrCanceled or a success with a Stop; the fake still holds both streams with a FinishRelease that is never released, so success can't happen there and the strict side stays enforced.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Tasks 2-5 now say "Committed" instead of quoting subjects; the rule is in workshop/lessons.md; Task 1 quotes an existing subject (1208207).
          round: 3
        - id: BR-7
          disposition: addressed
          note: buildResponse calls the shared toolInputObject; reverting to return nil makes TestToolInputMustBeAnObjectOnBothPaths fail (streaming=false).
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: atlas/llm.md:22 still shows onDelta func(string); scheduled for M3 Task 6, non-blocking.
          round: 3
      findings:
        - kind: defect
          id: BR-9
          severity: Minor
          title: No test pins that a thinking block without a stop is dropped from a partial Response
          detail: 'This is the 2nd finding in family partial-response-reports-incomplete-state. Rule (already in lessons.md and implemented for every block type in deltaState.complete): a partial keeps only blocks that reached content_block_stop, plus answer text. Fix the class in the oracle: extend assertOnlyCompleteBlocks to check every non-text block, not just tool_use, and add one cancel row held mid-thinking.'
          family: partial-response-reports-incomplete-state
          round: 3
        - kind: defect
          id: BR-10
          severity: Minor
          title: Suite subtest name still says a cancelled stream returns ErrCanceled after BR-5 relaxed it
          detail: llm/llmtest/suite.go:196 also accepts a complete success now; rename it to describe both legal outcomes.
          family: test-name-states-stale-contract
          round: 3
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      reviewed: 9c6d73ae3ff35b36ee8a15f45d2556704445d731
      blocked: false
---

# Gate ledger — tools#85 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-21T20:54:19-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `historical-prose-rewritten-by-mechanical-rename` (kind: defect) Path rewrite changed historical accounts to name `llm` where the past measurement was against `internal/llm`
  conformance/conformance.go package doc ("Measured, ... ./llm/... reported ok"), conformance/guard_test.go round-4 comment, and atlas/define.md:2772 now describe past events with a path that did not exist then; keep `internal/llm` (or annotate "then internal/llm") in historical narrative.
- **BR-2** [Minor] `tracker-checkbox-state-diverges` (kind: defect) Plan's Milestones list still shows `- [ ] M1` while the issue marks M1 done and every Task 1 step is checked
  workshop/plans/000085-llm-public-seam-plan.md line 369; tick it at milestone close so plan and issue agree.
- **BR-3** [Minor] `import-grouping` (kind: defect) llm/conformance_test.go places the repo-local conformance import inside the stdlib import group
  gofmt-clean but not goimports-grouped; move it into the github.com/xianxu/tools group.

## Round 2 — 2026-10-21T21:31:27-07:00 (claude) — BLOCKED

### Raised

- **BR-4** [Critical] `partial-response-reports-incomplete-state` (kind: defect) Cancelled/truncated/malformed stream's partial Response keeps the half-streamed tool_use in Blocks and Message()
  llm/anthropic.go Stream partial() replaces ToolCalls with deltas.done but builds Blocks from the SDK accumulator, which adds tool_use at content_block_start. Reproduced on the fake: Blocks[2] tool_use tu_2 has ToolCall.Input "{\"cit" (invalid JSON), Raw input {}, and Message() echoes it back, which breaks the "dropped, not half-reported" contract. Fix: keep a tool_use block only if its ID is in deltas.done. Regression: assert Blocks in TestCancelOutcomes and on a malformed-End stream.
- **BR-5** [Important] `test-observes-one-interleaving` (kind: defect) Suite cancel obligations (single cancel and ConcurrentStreams victim) assert ErrCanceled on a race the plan declares legal (ARCH-ORDER)
  llm/llmtest/suite.go cancels inside the callback and requires ErrCanceled. The fake holds the stream with FinishRelease, so that run is deterministic. Live (conformance_test.go:41, M3), a buffered short echo stream can legitimately finish first, and Stream returns success. Fix: on the live path accept ErrCanceled or a complete, consistent success; keep the strict check where the double holds the stream.
- **BR-6** [Minor] `tracker-checkbox-state-diverges` (kind: defect) Ticked plan Tasks 2-4 quote commit subjects that were never made (work landed as 46a25c8 and 33f3c3a)
  This is the 2nd finding in family tracker-checkbox-state-diverges. Rule: a ticked plan item may quote only artifacts that exist at the head, whether test names, commit subjects or milestone states. Extend the TestPlanCitesTestsThatExist guard to quoted commit subjects, or stop quoting planned commit names in task items. Prevalence: 3 of 3 M2 implementation tasks.
- **BR-7** [Minor] `inconsistent-validation-across-paths` (kind: defect) Complete accepts non-object tool_use input that Stream rejects with ErrMalformed
  buildResponse does not apply the JSON-object check that deltaState.next enforces at End, so the same tool_use input succeeds through Complete and fails through Stream.
- **BR-8** [Minor] `docs-deferred-to-later-milestone` (kind: defect) atlas/llm.md still documents the pre-M2 seam (func(string) Stream, no ErrCanceled or BeforeAttempt)
  M3 (Task 6) explicitly schedules the rewrite. Recorded so the close gate checks it; it does not block M2.

## Round 3 — 2026-10-21T21:36:55-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — partial() filters Blocks via deltaState.complete; reverting it makes TestCancelOutcomes and TestToolInputMustBeAnObjectOnBothPaths fail (Input {"cit / [1,2]).
- BR-5 — addressed — Suite accepts ErrCanceled or a success with a Stop; the fake still holds both streams with a FinishRelease that is never released, so success can't happen there and the strict side stays enforced.
- BR-6 — addressed — Tasks 2-5 now say "Committed" instead of quoting subjects; the rule is in workshop/lessons.md; Task 1 quotes an existing subject (1208207).
- BR-7 — addressed — buildResponse calls the shared toolInputObject; reverting to return nil makes TestToolInputMustBeAnObjectOnBothPaths fail (streaming=false).
- BR-8 — not-addressed — atlas/llm.md:22 still shows onDelta func(string); scheduled for M3 Task 6, non-blocking.

### Raised

- **BR-9** [Minor] `partial-response-reports-incomplete-state` (kind: defect) No test pins that a thinking block without a stop is dropped from a partial Response
  This is the 2nd finding in family partial-response-reports-incomplete-state. Rule (already in lessons.md and implemented for every block type in deltaState.complete): a partial keeps only blocks that reached content_block_stop, plus answer text. Fix the class in the oracle: extend assertOnlyCompleteBlocks to check every non-text block, not just tool_use, and add one cancel row held mid-thinking.
- **BR-10** [Minor] `test-name-states-stale-contract` (kind: defect) Suite subtest name still says a cancelled stream returns ErrCanceled after BR-5 relaxed it
  llm/llmtest/suite.go:196 also accepts a complete success now; rename it to describe both legal outcomes.

## Open findings

- **BR-1** [Minor] `historical-prose-rewritten-by-mechanical-rename` (kind: defect) Path rewrite changed historical accounts to name `llm` where the past measurement was against `internal/llm`
- **BR-2** [Minor] `tracker-checkbox-state-diverges` (kind: defect) Plan's Milestones list still shows `- [ ] M1` while the issue marks M1 done and every Task 1 step is checked
- **BR-3** [Minor] `import-grouping` (kind: defect) llm/conformance_test.go places the repo-local conformance import inside the stdlib import group
- **BR-8** [Minor] `docs-deferred-to-later-milestone` (kind: defect) atlas/llm.md still documents the pre-M2 seam (func(string) Stream, no ErrCanceled or BeforeAttempt)
- **BR-9** [Minor] `partial-response-reports-incomplete-state` (kind: defect) No test pins that a thinking block without a stop is dropped from a partial Response
- **BR-10** [Minor] `test-name-states-stale-contract` (kind: defect) Suite subtest name still says a cancelled stream returns ErrCanceled after BR-5 relaxed it
