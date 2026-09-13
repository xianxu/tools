---
gate: plan-quality
issue: 53
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-12T16:12:42-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: usageText already exists in package main (deckasker_test.go:409); the new renderer will not compile
          detail: func usageText(t *testing.T) string is declared at cmd/define/deckasker_test.go:409 and used at :338. Tests share package main, so the plan's usageText(c command, width int) breaks go test ./cmd/define/ from Task 2 on. Rename the renderer (or the helper) and update the entity table and test bodies.
          family: new-symbol-collides-with-package
          round: 1
        - id: PQ-2
          severity: Important
          title: The plan file reddens TestPlanTablesNameEntitiesThatExist and TestPlanCitesTestsThatExist, so every task's "go test ./cmd/define/ → PASS" is false as written
          detail: The modified row backticks args/usage as existing symbols; pronCommandHelp is marked deleted while still declared (repo_guard_test.go:1115); five inline-backticked Test* names do not exist yet (repo_guard_test.go:1402). Write the row as `command` only, keep pronCommandHelp modified until a Task 4 step flips it, cite unwritten tests without backticks, and log the two extra failures beside the nine so Task 0's exit criterion is honest.
          family: plan-guard-consistency
          round: 1
        - id: PQ-3
          severity: Minor
          title: Changing /help's summary touches eight assertions on "list the commands"; the plan should say the prefix is kept on purpose
          detail: commandloop_test.go:77,92,181,193,202,310 and pty_conformance_test.go:288,309 all Contains-check the old string; "list the commands, or explain one" keeps them green, but lessons.md:403 records this string once matching the wrong row, so state the intent.
          family: unstated-test-impact
          round: 1
        - id: PQ-4
          severity: Minor
          title: 'synopsis()/usageText need one adversarial line: empty args must not leave a trailing space, and widths {0, 20, 80}'
          detail: commandUsageSpan sets the synopsis in backticks, so "/stats " vs "/stats" would silently diverge between doc and screen. wrapText already returns the input at width 0 (render.go:634); a small width table pins the wrapping.
          family: risky-function-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-12T16:23:05-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Renderer is commandUsage; grep confirms no package symbol named commandUsage, findCommand, unknownCommand, asksForUsage, synopsis or any *Usage text.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: TestPlan* and TestNoArtifactNamesARetiredSymbol pass on the current tree; row is `command` only, pronCommandHelp deferred to Task 3, unwritten tests cited without backticks, Log records the two extra failures.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 1 Step 3 states the prefix is kept on purpose and names the eight assertions.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: TestCommandUsageWraps checks the trailing-space case and widths {0, 20, 80} over every row.
          round: 2
      blocked: false
content_hash: ef8a49cac97f3031d47f07e8f6ffde6286039437c70d2a1410ea4c095df5c354
---

# Gate ledger — tools#53 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T16:12:42-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `new-symbol-collides-with-package` usageText already exists in package main (deckasker_test.go:409); the new renderer will not compile
  func usageText(t *testing.T) string is declared at cmd/define/deckasker_test.go:409 and used at :338. Tests share package main, so the plan's usageText(c command, width int) breaks go test ./cmd/define/ from Task 2 on. Rename the renderer (or the helper) and update the entity table and test bodies.
- **PQ-2** [Important] `plan-guard-consistency` The plan file reddens TestPlanTablesNameEntitiesThatExist and TestPlanCitesTestsThatExist, so every task's "go test ./cmd/define/ → PASS" is false as written
  The modified row backticks args/usage as existing symbols; pronCommandHelp is marked deleted while still declared (repo_guard_test.go:1115); five inline-backticked Test* names do not exist yet (repo_guard_test.go:1402). Write the row as `command` only, keep pronCommandHelp modified until a Task 4 step flips it, cite unwritten tests without backticks, and log the two extra failures beside the nine so Task 0's exit criterion is honest.
- **PQ-3** [Minor] `unstated-test-impact` Changing /help's summary touches eight assertions on "list the commands"; the plan should say the prefix is kept on purpose
  commandloop_test.go:77,92,181,193,202,310 and pty_conformance_test.go:288,309 all Contains-check the old string; "list the commands, or explain one" keeps them green, but lessons.md:403 records this string once matching the wrong row, so state the intent.
- **PQ-4** [Minor] `risky-function-strategy` synopsis()/usageText need one adversarial line: empty args must not leave a trailing space, and widths {0, 20, 80}
  commandUsageSpan sets the synopsis in backticks, so "/stats " vs "/stats" would silently diverge between doc and screen. wrapText already returns the input at width 0 (render.go:634); a small width table pins the wrapping.

## Round 2 — 2026-09-12T16:23:05-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Renderer is commandUsage; grep confirms no package symbol named commandUsage, findCommand, unknownCommand, asksForUsage, synopsis or any *Usage text.
- PQ-2 — addressed — TestPlan* and TestNoArtifactNamesARetiredSymbol pass on the current tree; row is `command` only, pronCommandHelp deferred to Task 3, unwritten tests cited without backticks, Log records the two extra failures.
- PQ-3 — addressed — Task 1 Step 3 states the prefix is kept on purpose and names the eight assertions.
- PQ-4 — addressed — TestCommandUsageWraps checks the trailing-space case and widths {0, 20, 80} over every row.

## Open findings

(none — every finding has been disposed)
