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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c43d40aa-b76d-412a-af61-01b654196a3a
            repository: github.com/xianxu/tools
            issue_id: "000085"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/bba02b266d84ebf0
            reviewed_head: abfb3f17c90743f73d0b26a347abbddbbcbaf325
            slot: tools:1
            requested_at: "2026-10-22T06:35:57.753373Z"
          phase: finished
          pid: 62661
          started_at: "2026-10-22T06:35:57.839577Z"
          finished_at: "2026-10-22T06:41:38.004877Z"
          session_id: a9a175ba-738f-47a8-8a2e-a9072ce33950
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/tools
        code_head: abfb3f17c90743f73d0b26a347abbddbbcbaf325
        tracker_head: 211c99c63a7132f24eb1378528ffda826245f690
        branch_point: 32e6e8d9a548edb257c2ee7a4b8e1e2d2a8e0b90
        since: "2026-10-21T20:24:53-07:00"
        until: "2026-10-21T23:35:37-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T20:24:53-07:00"
              end: "2026-10-21T23:35:37-07:00"
              began: "2026-10-21T20:24:53-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/tools-slot1/tools
        sources:
            - harness: claude
              id: 38db0b39-19d1-4212-8c30-70640a7a3391
              slot: tools:1
              cwd: /Users/xianxu/workspace/worktree/tools-slot1/tools
              digest: 9a5440d5f7144420ccc5cd748c54f8614f35b2090e103329ddb80581694db66b
              started: "2026-10-22T03:23:53.951Z"
              spawned: false
              bytes: 5862892
              windows:
                - start: "2026-10-21T20:24:53-07:00"
                  end: "2026-10-21T23:35:37-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-21T20:47:14-07:00"
              finalized: "2026-10-21T20:54:19-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-21T21:20:59-07:00"
              finalized: "2026-10-21T21:31:27-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-21T21:34:27-07:00"
              finalized: "2026-10-21T21:36:55-07:00"
              finalizing: true
      "n": 4
      timestamp: "2026-10-21T23:41:39-07:00"
      dispatched: "2026-10-21T23:35:57-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: conformance/conformance.go:47 and guard_test.go:25 keep internal/llm with a "(then internal/llm)" or "at the time" note; atlas/define.md:2772 says "(as it was then)".
          round: 4
        - id: BR-2
          disposition: addressed
          note: Plan line 370 now reads "- [x] M1". The new M3 divergence is raised separately as the family's 3rd instance.
          round: 4
        - id: BR-3
          disposition: addressed
          note: llm/conformance_test.go imports conformance in the github.com/xianxu/tools group after the stdlib group; capture_conformance_test.go was regrouped in b9b9dc0 too.
          round: 4
        - id: BR-8
          disposition: addressed
          note: atlas/llm.md now documents Stream with func(Delta), TextOnly, ErrCanceled, BeforeAttempt, Usage.Attempts, the fake controls and the new live suites.
          round: 4
        - id: BR-9
          disposition: addressed
          note: '80cae37: assertOnlyCompleteBlocks checks every block type (thinking needs a signature, others need valid Raw), and TestPartialKeepsOnlyCompleteBlocks pins deltaState.complete dropping a thinking block with no stop.'
          round: 4
        - id: BR-10
          disposition: addressed
          note: suite.go:196 is renamed "a cancelled stream ends in ErrCanceled or a complete success, never a late callback".
          round: 4
      findings:
        - kind: defect
          id: BR-11
          severity: Important
          title: New tool-use captures are only text-searched, never decoded by Stream or buildResponse, yet Task 6 "captures replayed" is ticked
          detail: llm/llmtest/captures_test.go:130-159 only does strings.Contains on stream-tool-use.sse, message-tool-use.json and message-tool-result.json. No test runs them through Client.Stream (rendererTransport) or buildResponse, unlike the older captures (response_test.go:35, provider_render_test.go:94). FuzzDeltaAccumulator seeds (delta_test.go:49-51) are still made up although Task 3 says M3 adds real frames. A throwaway replay test in this review passed (fragments join to End Input equal to ToolCalls[0].Input, stop tool_use, thinking Raw and signature kept), so this is missing regression coverage, not a bug. Commit that test, or record in Revisions that the fuzz seed was dropped.
          family: capture-not-replayed-by-production-path
          round: 4
        - kind: defect
          id: BR-12
          severity: Minor
          title: Issue ticks M3 (abfb3f1) while the plan's Milestones still has "- [ ] M3"; both lines include land and tag v0.1.0, which have not happened
          detail: 'This is the 3rd finding in this family, so fix the rule, not the instance. Rule: the issue milestone box and the plan milestone box are one fact. Tick them in the same commit, and only when every deliverable named on the line exists. Fix: move land, tag and go get out of the M3 line in both files (keep them in the unticked Task 6 item), then tick M3 in both together. Tick each Done-when box when its evidence is logged; all five are still unchecked.'
          family: tracker-checkbox-state-diverges
          round: 4
        - kind: defect
          id: BR-13
          severity: Minor
          title: Plan Task 6 still says the tool capture is "forced", but auto ToolChoice shipped and there is no Revisions entry
          detail: The probe and Suite use tool_choice auto because a forced choice is refused alongside extended thinking. The reason is in the issue Log and suite.go:326, but the plan's Revisions section does not record it.
          family: plan-revision-missing-for-deviation
          round: 4
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 7dc54412-685d-4f8b-934f-6ee3fe04bec4
            repository: github.com/xianxu/tools
            issue_id: "000085"
            recipe: milestone-review
            round: 5
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/06a1d7149d731714
            reviewed_head: 74ef0b692d07aac73f4ef9c2e01c6f38dc92fb38
            slot: tools:1
            requested_at: "2026-10-22T06:46:38.584002Z"
          phase: finished
          pid: 70180
          started_at: "2026-10-22T06:46:38.792313Z"
          finished_at: "2026-10-22T06:50:43.945538Z"
          session_id: b9c609d2-8906-4d24-890b-e9fe0deb3b86
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/tools
        code_head: 74ef0b692d07aac73f4ef9c2e01c6f38dc92fb38
        tracker_head: 211c99c63a7132f24eb1378528ffda826245f690
        branch_point: 32e6e8d9a548edb257c2ee7a4b8e1e2d2a8e0b90
        since: "2026-10-21T20:24:53-07:00"
        until: "2026-10-21T23:43:12-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-21T20:24:53-07:00"
              end: "2026-10-21T23:43:12-07:00"
              began: "2026-10-21T20:24:53-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/tools-slot1/tools
        sources:
            - harness: claude
              id: 38db0b39-19d1-4212-8c30-70640a7a3391
              slot: tools:1
              cwd: /Users/xianxu/workspace/worktree/tools-slot1/tools
              digest: 22920c60919d753586bdf1126c24acb72c5146e7cffd93c21634408c25d4e3e7
              started: "2026-10-22T03:23:53.951Z"
              spawned: false
              bytes: 6061387
              windows:
                - start: "2026-10-21T20:24:53-07:00"
                  end: "2026-10-21T23:43:12-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-21T20:47:14-07:00"
              finalized: "2026-10-21T20:54:19-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-21T21:20:59-07:00"
              finalized: "2026-10-21T21:31:27-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-21T21:34:27-07:00"
              finalized: "2026-10-21T21:36:55-07:00"
              finalizing: true
            - dispatched: "2026-10-21T23:35:57-07:00"
              finalized: "2026-10-21T23:41:39-07:00"
              finalizing: false
      "n": 5
      timestamp: "2026-10-21T23:50:44-07:00"
      dispatched: "2026-10-21T23:46:38-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-11
          disposition: addressed
          note: capture_replay_test.go replays all three captures through Stream and buildResponse; mutating delta.go:104 and wire.go blockJSON each turned a test red; the fuzz corpus is seeded from message-tool-use.json.
          round: 5
        - id: BR-12
          disposition: addressed
          note: Issue and plan M3 lines are both ticked in 74ef0b6, with land, tag and go get moved to the unticked Task 6 item; Done-when boxes 1-4 ticked with Log evidence, box 5 (tag) left open.
          round: 5
        - id: BR-13
          disposition: addressed
          note: Task 6 now says auto choice (see Revisions), and the 2026-10-21 (M3) Revisions entry gives the reason (forced tool_choice is refused alongside extended thinking).
          round: 5
      recipe: milestone-review
      isolation: fenced
      reviewed: 74ef0b692d07aac73f4ef9c2e01c6f38dc92fb38
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

## Round 4 — 2026-10-21T23:41:39-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — conformance/conformance.go:47 and guard_test.go:25 keep internal/llm with a "(then internal/llm)" or "at the time" note; atlas/define.md:2772 says "(as it was then)".
- BR-2 — addressed — Plan line 370 now reads "- [x] M1". The new M3 divergence is raised separately as the family's 3rd instance.
- BR-3 — addressed — llm/conformance_test.go imports conformance in the github.com/xianxu/tools group after the stdlib group; capture_conformance_test.go was regrouped in b9b9dc0 too.
- BR-8 — addressed — atlas/llm.md now documents Stream with func(Delta), TextOnly, ErrCanceled, BeforeAttempt, Usage.Attempts, the fake controls and the new live suites.
- BR-9 — addressed — 80cae37: assertOnlyCompleteBlocks checks every block type (thinking needs a signature, others need valid Raw), and TestPartialKeepsOnlyCompleteBlocks pins deltaState.complete dropping a thinking block with no stop.
- BR-10 — addressed — suite.go:196 is renamed "a cancelled stream ends in ErrCanceled or a complete success, never a late callback".

### Raised

- **BR-11** [Important] `capture-not-replayed-by-production-path` (kind: defect) New tool-use captures are only text-searched, never decoded by Stream or buildResponse, yet Task 6 "captures replayed" is ticked
  llm/llmtest/captures_test.go:130-159 only does strings.Contains on stream-tool-use.sse, message-tool-use.json and message-tool-result.json. No test runs them through Client.Stream (rendererTransport) or buildResponse, unlike the older captures (response_test.go:35, provider_render_test.go:94). FuzzDeltaAccumulator seeds (delta_test.go:49-51) are still made up although Task 3 says M3 adds real frames. A throwaway replay test in this review passed (fragments join to End Input equal to ToolCalls[0].Input, stop tool_use, thinking Raw and signature kept), so this is missing regression coverage, not a bug. Commit that test, or record in Revisions that the fuzz seed was dropped.
- **BR-12** [Minor] `tracker-checkbox-state-diverges` (kind: defect) Issue ticks M3 (abfb3f1) while the plan's Milestones still has "- [ ] M3"; both lines include land and tag v0.1.0, which have not happened
  This is the 3rd finding in this family, so fix the rule, not the instance. Rule: the issue milestone box and the plan milestone box are one fact. Tick them in the same commit, and only when every deliverable named on the line exists. Fix: move land, tag and go get out of the M3 line in both files (keep them in the unticked Task 6 item), then tick M3 in both together. Tick each Done-when box when its evidence is logged; all five are still unchecked.
- **BR-13** [Minor] `plan-revision-missing-for-deviation` (kind: defect) Plan Task 6 still says the tool capture is "forced", but auto ToolChoice shipped and there is no Revisions entry
  The probe and Suite use tool_choice auto because a forced choice is refused alongside extended thinking. The reason is in the issue Log and suite.go:326, but the plan's Revisions section does not record it.

## Round 5 — 2026-10-21T23:50:44-07:00 (claude) — passed

### Disposed

- BR-11 — addressed — capture_replay_test.go replays all three captures through Stream and buildResponse; mutating delta.go:104 and wire.go blockJSON each turned a test red; the fuzz corpus is seeded from message-tool-use.json.
- BR-12 — addressed — Issue and plan M3 lines are both ticked in 74ef0b6, with land, tag and go get moved to the unticked Task 6 item; Done-when boxes 1-4 ticked with Log evidence, box 5 (tag) left open.
- BR-13 — addressed — Task 6 now says auto choice (see Revisions), and the 2026-10-21 (M3) Revisions entry gives the reason (forced tool_choice is refused alongside extended thinking).

## Open findings

(none — every finding has been disposed)
