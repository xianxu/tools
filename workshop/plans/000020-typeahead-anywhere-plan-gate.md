---
gate: plan-quality
issue: 20
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-26T11:12:10-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Gluing candidates changes Up-arrow recall, not just the grey tail — the plan only accounts for Suggestion
          detail: |-
            replraw.go:180 passes completionsFor's list to Apply, which routes KeyUp/KeyDown to
            walk (editor.go:79-82), and walk assigns e.Line = matches[next] (editor.go:38). Glued
            head+candidate entries make Up recall a line the user never typed and Enter submit it;
            cross-segment duplicates also break prefixMatch's deduped contract (history.go:50-51).
            The plan must state whether the walk list stays unglued or synthesized entries are
            accepted deliberately, and cover it in Done-when.
          family: shared-seam-second-consumer
          round: 1
        - id: PQ-2
          severity: Important
          title: Looping completionsFor over segments reaches the command branch mid-line, breaking the "/command untouched" Done-when
          detail: |-
            completionsFor branches on parseCommandLine(base) at command.go:42 before hist.Prefix.
            Per-segment, "/history 7" lets segment "7" fall through to hist.Prefix and glue a
            history match onto a settled command line, and any trailing segment starting with "/"
            completes a command mid-line. State the rule: segment expansion is history-namespace
            only; a command line returns today's answer unchanged. Name completionsFor as a
            unit-tested function alongside trailingSegments.
          family: command-namespace-boundary
          round: 1
        - id: PQ-3
          severity: Minor
          title: '"hist.Add already recorded it" understates what Add stores — recallLine keeps the forcing prefix'
          detail: |-
            The three Add sites (replraw.go:211, :230, :314) pass cmd.recallLine() (repl.go:131-144),
            which stores a forced literal lookup as "\word" and asks/commands as "?…"/"/…". So a
            \-forced lookup is not completable mid-line in-session, contra the spec's claim.
          family: recall-line-not-raw-word
          round: 1
        - id: PQ-4
          severity: Minor
          title: trailingSegments' head+text == line invariant deserves a property or fuzz test, not a table
          detail: |-
            It scans arbitrary user input and its whole safety story is one invariant. The repo
            already has the idiom in invariant_test.go / live_property_test.go; a hand-written table
            is blind to the malformed-input class by construction.
          family: invariant-needs-property-not-table
          round: 1
        - id: PQ-5
          severity: Minor
          title: deps is empty and no branch is named while tools#17 is still working on the current branch and also edits atlas/define.md
          detail: |-
            HEAD is 000017-user-model with issue 17 at status: working; its diff vs main already
            touches atlas/define.md, which plan step 6 sweeps. Sections do not overlap textually,
            but the plan should say this lands on its own branch off main rather than inside #17's.
          family: in-flight-branch-ownership
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-26T11:37:53-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Fixed as the class — Apply takes candidates{recall, complete}; walk gets recall only, and Done-when asserts it.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Segment loop confined to historyCompletions; the parseCommandLine test stays once on the whole line.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: historyCompletions strips the leading backslash, so the spec's claim is true rather than caveated.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: FuzzTrailingSegments added for the head+text == line invariant, matching the repo's existing fuzz idiom.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: 'HEAD is main with #17 M1 merged via PR #9; #17 M2 is blocked on #6, so #20 branches from main uncontended.'
          round: 2
      blocked: false
content_hash: 951917e88869d65767b59750c1e3e9914039560cb982c7bb601ae399b3520451
---

# Gate ledger — tools#20 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T11:12:10-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `shared-seam-second-consumer` Gluing candidates changes Up-arrow recall, not just the grey tail — the plan only accounts for Suggestion
  replraw.go:180 passes completionsFor's list to Apply, which routes KeyUp/KeyDown to
  walk (editor.go:79-82), and walk assigns e.Line = matches[next] (editor.go:38). Glued
  head+candidate entries make Up recall a line the user never typed and Enter submit it;
  cross-segment duplicates also break prefixMatch's deduped contract (history.go:50-51).
  The plan must state whether the walk list stays unglued or synthesized entries are
  accepted deliberately, and cover it in Done-when.
- **PQ-2** [Important] `command-namespace-boundary` Looping completionsFor over segments reaches the command branch mid-line, breaking the "/command untouched" Done-when
  completionsFor branches on parseCommandLine(base) at command.go:42 before hist.Prefix.
  Per-segment, "/history 7" lets segment "7" fall through to hist.Prefix and glue a
  history match onto a settled command line, and any trailing segment starting with "/"
  completes a command mid-line. State the rule: segment expansion is history-namespace
  only; a command line returns today's answer unchanged. Name completionsFor as a
  unit-tested function alongside trailingSegments.
- **PQ-3** [Minor] `recall-line-not-raw-word` "hist.Add already recorded it" understates what Add stores — recallLine keeps the forcing prefix
  The three Add sites (replraw.go:211, :230, :314) pass cmd.recallLine() (repl.go:131-144),
  which stores a forced literal lookup as "\word" and asks/commands as "?…"/"/…". So a
  \-forced lookup is not completable mid-line in-session, contra the spec's claim.
- **PQ-4** [Minor] `invariant-needs-property-not-table` trailingSegments' head+text == line invariant deserves a property or fuzz test, not a table
  It scans arbitrary user input and its whole safety story is one invariant. The repo
  already has the idiom in invariant_test.go / live_property_test.go; a hand-written table
  is blind to the malformed-input class by construction.
- **PQ-5** [Minor] `in-flight-branch-ownership` deps is empty and no branch is named while tools#17 is still working on the current branch and also edits atlas/define.md
  HEAD is 000017-user-model with issue 17 at status: working; its diff vs main already
  touches atlas/define.md, which plan step 6 sweeps. Sections do not overlap textually,
  but the plan should say this lands on its own branch off main rather than inside #17's.

## Round 2 — 2026-08-26T11:37:53-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Fixed as the class — Apply takes candidates{recall, complete}; walk gets recall only, and Done-when asserts it.
- PQ-2 — addressed — Segment loop confined to historyCompletions; the parseCommandLine test stays once on the whole line.
- PQ-3 — addressed — historyCompletions strips the leading backslash, so the spec's claim is true rather than caveated.
- PQ-4 — addressed — FuzzTrailingSegments added for the head+text == line invariant, matching the repo's existing fuzz idiom.
- PQ-5 — addressed — HEAD is main with #17 M1 merged via PR #9; #17 M2 is blocked on #6, so #20 branches from main uncontended.

## Open findings

(none — every finding has been disposed)
