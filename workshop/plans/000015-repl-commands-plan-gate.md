---
gate: plan-quality
issue: 15
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-21T15:40:26-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: commandCtx names a store.Clock that no existing dep provides, and no task adds it
          detail: |-
            deps (main.go:20-44) and options (main.go:143-160) have no clock field; the
            only store.Clock in the package is unexported inside storeCapturer
            (capture.go:58). Task 5's file list omits main.go, so nothing adds the field,
            defaults it in realDeps/withStore, or plumbs it into runEditor. An
            implementer reaches Task 5 Step 3 and has to invent the wiring.
          family: undeclared-wiring
          round: 1
        - id: PQ-2
          severity: Important
          title: Dispatch lands only in runEditor, so replLines still sends /history to the dictionary
          detail: |-
            replraw.go:96-99 states that both loops route through parseREPLLine on
            purpose, and repl.go:92-100 records that conflating them has been a bug four
            times. With the /-decision only in the raw loop, a /history typed under piped
            stdin, redirected stdout, or a failed enterRaw (replraw.go:20,25) parses as
            cmdDefine and is looked up as a word — falsifying the Done-when "A word
            starting with / is impossible to look up" (ARCH-PURPOSE) and creating a
            second table for what a line means (ARCH-DRY).
          family: single-line-decision-table
          round: 1
        - id: PQ-3
          severity: Important
          title: parseHistoryArgs has no upper bound; a huge days value overflows historyWindow silently
          detail: |-
            The plan rejects only zero, negatives and non-numbers. "/history
            9223372036854775807" passes all three: Atoi succeeds, AddDate(0,0,-(days-1))
            overflows int64 in the seconds computation, since lands at an arbitrary
            instant possibly in the future, and /history prints an empty list rather than
            all history. Bound days, and state the adversarial-input strategy (fuzz
            seeded with malformed and out-of-range forms) for parseHistoryArgs and
            parseCommandLine.
          family: unbounded-parser-input
          round: 1
        - id: PQ-4
          severity: Minor
          title: Tasks 1, 3 and 4 enumerate test cases in prose instead of one strategy line per risky function
          detail: |-
            The case lists are a lossy pre-image of code that will exist within the hour,
            and they systematically miss the malformed-input class. Compress to the
            adversarial input class plus the mechanical guard, per function.
          family: test-strategy-not-enumeration
          round: 1
        - id: PQ-5
          severity: Minor
          title: '"Typing / shows the available commands" has no named mechanism; Suggestion renders one tail, not a list'
          detail: |-
            Suggestion (editor.go:145-156) returns a single grey tail. With a second
            command in the table only the first is ever visible unless Apply's Up/Down
            walk (editor.go:117-139) is the intended list affordance. Invisible with one
            row today; wrong the moment the table grows, which the Done-when invites.
          family: unnamed-ui-affordance
          round: 1
        - id: PQ-6
          severity: Minor
          title: Spec says "Tab or Return accepts" but the plan makes /his + Return suggest rather than run
          detail: |-
            Editor.String() deliberately excludes the suggestion (editor.go:101-104) and
            KeyEnter returns ActSubmit without accepting, so a unique prefix submitted
            with Return reaches dispatchCommand as "his" and gets nearestCommands. That is
            a defensible resolution of the Spec's wording — state it as the deliberate
            choice rather than leaving the Done-when readable both ways.
          family: spec-invariant-conflict
          round: 1
        - id: PQ-7
          severity: Minor
          title: runHistory copies forgetWord's nil-deck message instead of reusing it
          detail: |-
            main.go:400-406 is a two-branch conditional keyed on opt.noCapture. The plan
            says runHistory prints "the same explanatory message", which makes two
            sources for one fact. Extract a shared helper and have both call it
            (ARCH-DRY).
          family: copied-not-extracted
          round: 1
        - id: PQ-8
          severity: Minor
          title: The plan has Risks but no Non-goals section
          detail: |-
            Deliberate omissions are scattered inline (typos excluded from /history, / is
            never a headword). An explicit Non-goals list is where the line-loop
            divergence and the absence of a multi-command list UI would have surfaced
            before implementation.
          family: no-stated-non-goals
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-21T15:46:19-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 4 adds clock to storeDeps, builds it in openStore, deletes the inline main.go:124 construction, and pins it with a test that the supplied clock reaches both capturer and commandCtx.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Dispatch is now a cmdCommand kind on parseREPLLine (the seam at repl.go:30 both loops route through), both loops tested, plus a mutation check on deleting the replLines case.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: maxHistoryDays = 3650 with a refusal naming the limit; days=999999999 is an explicit test case. The fuzz-strategy half of the ask rolls into PQ-4.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Enumeration expanded, not compressed; Task 5's bullets are defensible domain facts, but Tasks 1/6/7 still list cases and no fuzz target is named for either parser.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Now an explicit non-goal — no multi-command list UI, existing suggestion mechanism only, /help prints the table.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: 'First non-goal states it: Tab accepts, Return submits what was typed, per #14''s contract at editor.go:101-104.'
          round: 2
        - id: PQ-7
          disposition: addressed
          note: noDeckMessage(opt) extracted from main.go:399-408, with a test asserting forgetWord and runHistory produce the identical string.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Six-entry Non-goals section; it is where the Return-accepts and list-UI decisions now live.
          round: 2
      findings:
        - id: PQ-9
          severity: Minor
          title: The issue's Plan block still names "the submitLine branch" — the design the plan file rejects
          detail: |-
            This is the 2nd finding in family `spec-invariant-conflict`. Do NOT fix this
            instance. The rule that covers both: the plan file is the design of record, so
            any restatement of it elsewhere must be regenerated when the plan is revised,
            or deleted — a surviving restatement is a stale second source (ARCH-DRY at the
            artifact level). Measured prevalence, all from the round-1 revision: (1) the
            issue's `## Plan` Task 5 names "the `submitLine` branch", which is exactly the
            PQ-2 defect the plan file removed and its Revisions section calls out by name;
            (2) the issue lists 6 tasks against the plan's 8, and its M1 bullet omits the
            dispatch task entirely while its M2 bullet puts `dispatchCommand` in M2, where
            the plan puts it in M1 Task 3; (3) the issue's mutation-check paragraph names
            only the historyWindow mutation, not the cmdCommand-in-replLines one the
            revision added; (4) plan file Chunk 1 says "Task 5 adds it" of the clock, which
            is Task 4; (5) plan file Chunk 1 says "Task 5 lifts it" of noDeckMessage, which
            is Task 7. Non-blocking: the plan file is emphatic, Task 3 is titled "dispatch
            in BOTH loops", and the replLines mutation check would redden if an implementer
            followed the stale row.
          family: spec-invariant-conflict
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-08-21T15:50:09-07:00"
      agent: claude
      dispose:
        - id: PQ-9
          disposition: addressed
          note: Issue Plan now delegates to the plan file as design of record, deleting the restatement class; instances 2-5 also gone.
          round: 3
        - id: PQ-4
          disposition: not-addressed
          note: Tasks 1/3/5/6/7 still enumerate cases in prose; Minor, carried to close review, not blocking.
          round: 3
      blocked: false
content_hash: 455c931bdee579c1c8ed9c4c9ef3bb36af1fcddadbd5f6149599b65bb0916249
---

# Gate ledger — tools#15 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-21T15:40:26-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `undeclared-wiring` commandCtx names a store.Clock that no existing dep provides, and no task adds it
  deps (main.go:20-44) and options (main.go:143-160) have no clock field; the
  only store.Clock in the package is unexported inside storeCapturer
  (capture.go:58). Task 5's file list omits main.go, so nothing adds the field,
  defaults it in realDeps/withStore, or plumbs it into runEditor. An
  implementer reaches Task 5 Step 3 and has to invent the wiring.
- **PQ-2** [Important] `single-line-decision-table` Dispatch lands only in runEditor, so replLines still sends /history to the dictionary
  replraw.go:96-99 states that both loops route through parseREPLLine on
  purpose, and repl.go:92-100 records that conflating them has been a bug four
  times. With the /-decision only in the raw loop, a /history typed under piped
  stdin, redirected stdout, or a failed enterRaw (replraw.go:20,25) parses as
  cmdDefine and is looked up as a word — falsifying the Done-when "A word
  starting with / is impossible to look up" (ARCH-PURPOSE) and creating a
  second table for what a line means (ARCH-DRY).
- **PQ-3** [Important] `unbounded-parser-input` parseHistoryArgs has no upper bound; a huge days value overflows historyWindow silently
  The plan rejects only zero, negatives and non-numbers. "/history
  9223372036854775807" passes all three: Atoi succeeds, AddDate(0,0,-(days-1))
  overflows int64 in the seconds computation, since lands at an arbitrary
  instant possibly in the future, and /history prints an empty list rather than
  all history. Bound days, and state the adversarial-input strategy (fuzz
  seeded with malformed and out-of-range forms) for parseHistoryArgs and
  parseCommandLine.
- **PQ-4** [Minor] `test-strategy-not-enumeration` Tasks 1, 3 and 4 enumerate test cases in prose instead of one strategy line per risky function
  The case lists are a lossy pre-image of code that will exist within the hour,
  and they systematically miss the malformed-input class. Compress to the
  adversarial input class plus the mechanical guard, per function.
- **PQ-5** [Minor] `unnamed-ui-affordance` "Typing / shows the available commands" has no named mechanism; Suggestion renders one tail, not a list
  Suggestion (editor.go:145-156) returns a single grey tail. With a second
  command in the table only the first is ever visible unless Apply's Up/Down
  walk (editor.go:117-139) is the intended list affordance. Invisible with one
  row today; wrong the moment the table grows, which the Done-when invites.
- **PQ-6** [Minor] `spec-invariant-conflict` Spec says "Tab or Return accepts" but the plan makes /his + Return suggest rather than run
  Editor.String() deliberately excludes the suggestion (editor.go:101-104) and
  KeyEnter returns ActSubmit without accepting, so a unique prefix submitted
  with Return reaches dispatchCommand as "his" and gets nearestCommands. That is
  a defensible resolution of the Spec's wording — state it as the deliberate
  choice rather than leaving the Done-when readable both ways.
- **PQ-7** [Minor] `copied-not-extracted` runHistory copies forgetWord's nil-deck message instead of reusing it
  main.go:400-406 is a two-branch conditional keyed on opt.noCapture. The plan
  says runHistory prints "the same explanatory message", which makes two
  sources for one fact. Extract a shared helper and have both call it
  (ARCH-DRY).
- **PQ-8** [Minor] `no-stated-non-goals` The plan has Risks but no Non-goals section
  Deliberate omissions are scattered inline (typos excluded from /history, / is
  never a headword). An explicit Non-goals list is where the line-loop
  divergence and the absence of a multi-command list UI would have surfaced
  before implementation.

## Round 2 — 2026-08-21T15:46:19-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 4 adds clock to storeDeps, builds it in openStore, deletes the inline main.go:124 construction, and pins it with a test that the supplied clock reaches both capturer and commandCtx.
- PQ-2 — addressed — Dispatch is now a cmdCommand kind on parseREPLLine (the seam at repl.go:30 both loops route through), both loops tested, plus a mutation check on deleting the replLines case.
- PQ-3 — addressed — maxHistoryDays = 3650 with a refusal naming the limit; days=999999999 is an explicit test case. The fuzz-strategy half of the ask rolls into PQ-4.
- PQ-4 — not-addressed — Enumeration expanded, not compressed; Task 5's bullets are defensible domain facts, but Tasks 1/6/7 still list cases and no fuzz target is named for either parser.
- PQ-5 — addressed — Now an explicit non-goal — no multi-command list UI, existing suggestion mechanism only, /help prints the table.
- PQ-6 — addressed — First non-goal states it: Tab accepts, Return submits what was typed, per #14's contract at editor.go:101-104.
- PQ-7 — addressed — noDeckMessage(opt) extracted from main.go:399-408, with a test asserting forgetWord and runHistory produce the identical string.
- PQ-8 — addressed — Six-entry Non-goals section; it is where the Return-accepts and list-UI decisions now live.

### Raised

- **PQ-9** [Minor] `spec-invariant-conflict` The issue's Plan block still names "the submitLine branch" — the design the plan file rejects
  This is the 2nd finding in family `spec-invariant-conflict`. Do NOT fix this
  instance. The rule that covers both: the plan file is the design of record, so
  any restatement of it elsewhere must be regenerated when the plan is revised,
  or deleted — a surviving restatement is a stale second source (ARCH-DRY at the
  artifact level). Measured prevalence, all from the round-1 revision: (1) the
  issue's `## Plan` Task 5 names "the `submitLine` branch", which is exactly the
  PQ-2 defect the plan file removed and its Revisions section calls out by name;
  (2) the issue lists 6 tasks against the plan's 8, and its M1 bullet omits the
  dispatch task entirely while its M2 bullet puts `dispatchCommand` in M2, where
  the plan puts it in M1 Task 3; (3) the issue's mutation-check paragraph names
  only the historyWindow mutation, not the cmdCommand-in-replLines one the
  revision added; (4) plan file Chunk 1 says "Task 5 adds it" of the clock, which
  is Task 4; (5) plan file Chunk 1 says "Task 5 lifts it" of noDeckMessage, which
  is Task 7. Non-blocking: the plan file is emphatic, Task 3 is titled "dispatch
  in BOTH loops", and the replLines mutation check would redden if an implementer
  followed the stale row.

## Round 3 — 2026-08-21T15:50:09-07:00 (claude) — passed

### Disposed

- PQ-9 — addressed — Issue Plan now delegates to the plan file as design of record, deleting the restatement class; instances 2-5 also gone.
- PQ-4 — not-addressed — Tasks 1/3/5/6/7 still enumerate cases in prose; Minor, carried to close review, not blocking.

## Open findings

- **PQ-4** [Minor] `test-strategy-not-enumeration` Tasks 1, 3 and 4 enumerate test cases in prose instead of one strategy line per risky function
