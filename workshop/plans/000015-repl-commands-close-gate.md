---
gate: boundary-review
issue: 15
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-21T16:11:01-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 1, 3 and 4 enumerate test cases in prose instead of one strategy line per risky function
          detail: |-
            The case lists are a lossy pre-image of code that will exist within the hour,
            and they systematically miss the malformed-input class. Compress to the
            adversarial input class plus the mechanical guard, per function.
            (carried from plan-quality PQ-4, deferred to the boundary review)
          family: test-strategy-not-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-21T16:11:01-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: Command type-ahead works in the editor but no test pins the wiring
          detail: |-
            Verified by mutation: replacing all five completionsFor(e.WalkBase(), hist, commands)
            calls in runEditor (replraw.go:79,92,124,131,153) with hist.Prefix(e.WalkBase()) leaves
            the whole suite green with -count=1. TestCompletionsFor covers the pure switch only, so
            M1's headline Done-when "type-ahead narrows" is asserted nowhere in the production path.
            A runEditor test scripting "/he" and asserting greyOn+"lp" passes on HEAD and fails under
            the mutation (both verified).
          family: unpinned-production-wiring
          round: 2
        - id: BR-3
          severity: Important
          title: TestRawEditorDispatchesCommands passes with dispatch removed
          detail: |-
            commandloop_test.go:73-83 asserts stdout contains "/help", but runEditor echoes the
            committed line via RenderLine (replraw.go:108) before dispatching, so the assertion is
            satisfied by the echo. Verified: neutering the dispatchCommand call inside the cooked
            closure leaves this test PASS. Assert on runHelp's output shape ("  /help " with its
            %-10s padding) or on an occurrence count instead.
          family: assertion-cannot-distinguish-paths
          round: 2
        - id: BR-4
          severity: Important
          title: README and --help never mention the / command surface shipped at M1
          detail: |-
            README.md contains no occurrence of "/help" or command mode, and the usage text at
            main.go:171-186 is unchanged, while atlas/define.md gained a full "## Command mode"
            section in the same commit. The binary now ships a surface a reader types; two lines in
            the key table plus one sentence closes it. The plan defers this to M2 Task 8 Steps 2-3,
            which is what makes the two artifacts disagree.
          family: docs-consumer-not-updated
          round: 2
        - id: BR-5
          severity: Important
          title: All fifteen M1 step boxes in the plan are unticked at the boundary
          detail: |-
            workshop/plans/000015-repl-commands-plan.md:184-221 shows Tasks 1-3 entirely as "- [ ]"
            while the issue's M1 bullet is ticked and the code is shipped. AGENTS.md section 8 asks
            for per-milestone ticking and the close gate's plan-unchecked guard reads these boxes.
          family: plan-artifact-lags-code
          round: 2
        - id: BR-6
          severity: Important
          title: commandCtx is constructed twice, and M2 adds two more fields to it
          detail: |-
            repl.go:150-152 and replraw.go:117-119 carry the identical commandCtx literal. Plan
            Tasks 4 and 7 add deck and clock; wiring one site and not the other is the same
            two-loops-disagree failure this milestone's design exists to prevent (ARCH-DRY).
            Extract newCommandCtx(d, stdout, stderr) while it is a three-line change.
          family: parallel-construction-drift
          round: 2
        - id: BR-7
          severity: Minor
          title: commandCtx.width is written at both call sites and read by nothing
          family: dead-field-at-boundary
          round: 2
        - id: BR-8
          severity: Minor
          title: Completion lowercases only the input while dispatch uses EqualFold and Suggestion is case-sensitive
          detail: |-
            commandCompletions (command.go:73) lowercases the prefix but not c.name, so an uppercase
            registry name would silently never complete; and because Suggestion compares
            case-sensitively, the {"HIS", ["/history"]} guarantee at command_test.go:63 is
            unreachable through the grey tail (Up-arrow does reach it).
          family: case-policy-inconsistent
          round: 2
        - id: BR-9
          severity: Minor
          title: dispatchCommand infers "nothing was close" from len(near) == len(cmds)
          detail: |-
            command.go:161. nearestCommands knows which branch it took and discards it; with two
            commands both within edit distance 2 the caller prints the menu form instead of
            "did you mean".
          family: caller-rederives-callee-knowledge
          round: 2
        - id: BR-10
          severity: Minor
          title: Nothing checks that every commands row has a non-nil run
          family: registry-row-unvalidated
          round: 2
        - id: BR-11
          severity: Minor
          title: sameSet compares ordered args in repl_test.go:29 and commandloop_test.go:43
          family: set-compare-for-ordered-data
          round: 2
        - id: BR-12
          severity: Minor
          title: command_test.go:60-62 comment describes no row in the table it heads
          family: comment-orphaned-by-insertion
          round: 2
        - id: BR-13
          severity: Minor
          title: define /help as a one-shot argument still goes to the dictionary
          detail: |-
            Acceptable under the Spec's "line" framing, but worth a doc line once /history exists:
            define /history will fail with "no dictionary entry" while echo /history | define runs it.
          family: entry-mode-inconsistency
          round: 2
        - id: BR-14
          severity: Minor
          title: main.go:4-12 has a stray blank line and the store import inside the stdlib group
          family: import-grouping
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-21T16:22:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan Tasks 1/3/5/6/7 still enumerate cases in prose; no fuzz target named for either parser.
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'Mutation-verified: all five completionsFor sites reverted to hist.Prefix, BUILD_OK, TestEditorSuggestsFromCommands FAILS.'
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'Mutation-verified: dispatch neutered in runEditor, BUILD_OK, TestRawEditorDispatchesCommands FAILS on "list the commands".'
          round: 3
        - id: BR-4
          disposition: addressed
          note: README.md:31 and main.go usage both name the / surface; see the new fix-appended-not-integrated finding for the residue.
          round: 3
        - id: BR-5
          disposition: addressed
          note: All fifteen M1 step boxes ticked at plan lines 184-221.
          round: 3
        - id: BR-6
          disposition: addressed
          note: newCommandCtx at command.go:148, used at repl.go:155 and replraw.go:117; M2 field additions are now compiler-enforced.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: Still unread, and newCommandCtx now re-derives width per dispatch — a second source beside opt.width.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: 'Probed live: /HEL + Tab does not accept, then dispatches and fails; /HELP + Enter works. Unchanged.'
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: 'Worse than measured: with one command, len(near)==len(cmds) always, so "did you mean" is unreachable in production at M1.'
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: No nil-run check anywhere; a row without run panics at dispatch.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: sameSet still compares ordered args at repl_test.go:30 and commandloop_test.go:47.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: Prevalence now 2 — command_test.go:93 "the four call sites in replraw.go" was made wrong by the cmdCommand branch's fifth.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: 'Probed: `define /help` exits 1 with "define: /help: no dictionary entry".'
          round: 3
        - id: BR-14
          disposition: not-addressed
          note: main.go:3-16 unchanged — blank line after "context", store import inside the stdlib group.
          round: 3
      findings:
        - id: BR-15
          severity: Minor
          title: The BR-4 README fix was appended to the sh block instead of integrated into the key table
          detail: |-
            README.md:31 adds a second "define" row to a shell block whose every other line is a
            distinct invocation, so the block now lists the same command twice with a continuation
            comment. The editor key table at README.md:44-51 — the doc's actual structure for "what
            can I type" — gained no row for /, and its "Enter | define what you typed" row is now
            incomplete, since Enter also dispatches a command. The finding asked for two lines in
            the key table; the fix landed elsewhere.
          family: fix-appended-not-integrated
          round: 3
        - id: BR-16
          severity: Minor
          title: A piped unknown command exits 1 where dispatchCommand computes 2 and README documents 2
          detail: |-
            Probed: `echo /qqqqqq | define` exits 1. dispatchCommand returns 2 (repl.go:155 discards
            it into anyFailed), and README documents 2 as the usage-error code, so a script cannot
            tell "no such command" from "no dictionary entry". Defensible for a multi-line loop, but
            it is script-visible behaviour and no test asserts it in either direction.
          family: loop-collapses-callee-exit-code
          round: 3
      boundary: M1
      blocked: false
---

# Gate ledger — tools#15 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-21T16:11:01-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-strategy-not-enumeration` Tasks 1, 3 and 4 enumerate test cases in prose instead of one strategy line per risky function
  The case lists are a lossy pre-image of code that will exist within the hour,
  and they systematically miss the malformed-input class. Compress to the
  adversarial input class plus the mechanical guard, per function.
  (carried from plan-quality PQ-4, deferred to the boundary review)

## Round 2 — 2026-08-21T16:11:01-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `unpinned-production-wiring` Command type-ahead works in the editor but no test pins the wiring
  Verified by mutation: replacing all five completionsFor(e.WalkBase(), hist, commands)
  calls in runEditor (replraw.go:79,92,124,131,153) with hist.Prefix(e.WalkBase()) leaves
  the whole suite green with -count=1. TestCompletionsFor covers the pure switch only, so
  M1's headline Done-when "type-ahead narrows" is asserted nowhere in the production path.
  A runEditor test scripting "/he" and asserting greyOn+"lp" passes on HEAD and fails under
  the mutation (both verified).
- **BR-3** [Important] `assertion-cannot-distinguish-paths` TestRawEditorDispatchesCommands passes with dispatch removed
  commandloop_test.go:73-83 asserts stdout contains "/help", but runEditor echoes the
  committed line via RenderLine (replraw.go:108) before dispatching, so the assertion is
  satisfied by the echo. Verified: neutering the dispatchCommand call inside the cooked
  closure leaves this test PASS. Assert on runHelp's output shape ("  /help " with its
  %-10s padding) or on an occurrence count instead.
- **BR-4** [Important] `docs-consumer-not-updated` README and --help never mention the / command surface shipped at M1
  README.md contains no occurrence of "/help" or command mode, and the usage text at
  main.go:171-186 is unchanged, while atlas/define.md gained a full "## Command mode"
  section in the same commit. The binary now ships a surface a reader types; two lines in
  the key table plus one sentence closes it. The plan defers this to M2 Task 8 Steps 2-3,
  which is what makes the two artifacts disagree.
- **BR-5** [Important] `plan-artifact-lags-code` All fifteen M1 step boxes in the plan are unticked at the boundary
  workshop/plans/000015-repl-commands-plan.md:184-221 shows Tasks 1-3 entirely as "- [ ]"
  while the issue's M1 bullet is ticked and the code is shipped. AGENTS.md section 8 asks
  for per-milestone ticking and the close gate's plan-unchecked guard reads these boxes.
- **BR-6** [Important] `parallel-construction-drift` commandCtx is constructed twice, and M2 adds two more fields to it
  repl.go:150-152 and replraw.go:117-119 carry the identical commandCtx literal. Plan
  Tasks 4 and 7 add deck and clock; wiring one site and not the other is the same
  two-loops-disagree failure this milestone's design exists to prevent (ARCH-DRY).
  Extract newCommandCtx(d, stdout, stderr) while it is a three-line change.
- **BR-7** [Minor] `dead-field-at-boundary` commandCtx.width is written at both call sites and read by nothing
- **BR-8** [Minor] `case-policy-inconsistent` Completion lowercases only the input while dispatch uses EqualFold and Suggestion is case-sensitive
  commandCompletions (command.go:73) lowercases the prefix but not c.name, so an uppercase
  registry name would silently never complete; and because Suggestion compares
  case-sensitively, the {"HIS", ["/history"]} guarantee at command_test.go:63 is
  unreachable through the grey tail (Up-arrow does reach it).
- **BR-9** [Minor] `caller-rederives-callee-knowledge` dispatchCommand infers "nothing was close" from len(near) == len(cmds)
  command.go:161. nearestCommands knows which branch it took and discards it; with two
  commands both within edit distance 2 the caller prints the menu form instead of
  "did you mean".
- **BR-10** [Minor] `registry-row-unvalidated` Nothing checks that every commands row has a non-nil run
- **BR-11** [Minor] `set-compare-for-ordered-data` sameSet compares ordered args in repl_test.go:29 and commandloop_test.go:43
- **BR-12** [Minor] `comment-orphaned-by-insertion` command_test.go:60-62 comment describes no row in the table it heads
- **BR-13** [Minor] `entry-mode-inconsistency` define /help as a one-shot argument still goes to the dictionary
  Acceptable under the Spec's "line" framing, but worth a doc line once /history exists:
  define /history will fail with "no dictionary entry" while echo /history | define runs it.
- **BR-14** [Minor] `import-grouping` main.go:4-12 has a stray blank line and the store import inside the stdlib group

## Round 3 — 2026-08-21T16:22:34-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan Tasks 1/3/5/6/7 still enumerate cases in prose; no fuzz target named for either parser.
- BR-2 — addressed — Mutation-verified: all five completionsFor sites reverted to hist.Prefix, BUILD_OK, TestEditorSuggestsFromCommands FAILS.
- BR-3 — addressed — Mutation-verified: dispatch neutered in runEditor, BUILD_OK, TestRawEditorDispatchesCommands FAILS on "list the commands".
- BR-4 — addressed — README.md:31 and main.go usage both name the / surface; see the new fix-appended-not-integrated finding for the residue.
- BR-5 — addressed — All fifteen M1 step boxes ticked at plan lines 184-221.
- BR-6 — addressed — newCommandCtx at command.go:148, used at repl.go:155 and replraw.go:117; M2 field additions are now compiler-enforced.
- BR-7 — not-addressed — Still unread, and newCommandCtx now re-derives width per dispatch — a second source beside opt.width.
- BR-8 — not-addressed — Probed live: /HEL + Tab does not accept, then dispatches and fails; /HELP + Enter works. Unchanged.
- BR-9 — not-addressed — Worse than measured: with one command, len(near)==len(cmds) always, so "did you mean" is unreachable in production at M1.
- BR-10 — not-addressed — No nil-run check anywhere; a row without run panics at dispatch.
- BR-11 — not-addressed — sameSet still compares ordered args at repl_test.go:30 and commandloop_test.go:47.
- BR-12 — not-addressed — Prevalence now 2 — command_test.go:93 "the four call sites in replraw.go" was made wrong by the cmdCommand branch's fifth.
- BR-13 — not-addressed — Probed: `define /help` exits 1 with "define: /help: no dictionary entry".
- BR-14 — not-addressed — main.go:3-16 unchanged — blank line after "context", store import inside the stdlib group.

### Raised

- **BR-15** [Minor] `fix-appended-not-integrated` The BR-4 README fix was appended to the sh block instead of integrated into the key table
  README.md:31 adds a second "define" row to a shell block whose every other line is a
  distinct invocation, so the block now lists the same command twice with a continuation
  comment. The editor key table at README.md:44-51 — the doc's actual structure for "what
  can I type" — gained no row for /, and its "Enter | define what you typed" row is now
  incomplete, since Enter also dispatches a command. The finding asked for two lines in
  the key table; the fix landed elsewhere.
- **BR-16** [Minor] `loop-collapses-callee-exit-code` A piped unknown command exits 1 where dispatchCommand computes 2 and README documents 2
  Probed: `echo /qqqqqq | define` exits 1. dispatchCommand returns 2 (repl.go:155 discards
  it into anyFailed), and README documents 2 as the usage-error code, so a script cannot
  tell "no such command" from "no dictionary entry". Defensible for a multi-line loop, but
  it is script-visible behaviour and no test asserts it in either direction.

## Open findings

- **BR-1** [Minor] `test-strategy-not-enumeration` Tasks 1, 3 and 4 enumerate test cases in prose instead of one strategy line per risky function
- **BR-7** [Minor] `dead-field-at-boundary` commandCtx.width is written at both call sites and read by nothing
- **BR-8** [Minor] `case-policy-inconsistent` Completion lowercases only the input while dispatch uses EqualFold and Suggestion is case-sensitive
- **BR-9** [Minor] `caller-rederives-callee-knowledge` dispatchCommand infers "nothing was close" from len(near) == len(cmds)
- **BR-10** [Minor] `registry-row-unvalidated` Nothing checks that every commands row has a non-nil run
- **BR-11** [Minor] `set-compare-for-ordered-data` sameSet compares ordered args in repl_test.go:29 and commandloop_test.go:43
- **BR-12** [Minor] `comment-orphaned-by-insertion` command_test.go:60-62 comment describes no row in the table it heads
- **BR-13** [Minor] `entry-mode-inconsistency` define /help as a one-shot argument still goes to the dictionary
- **BR-14** [Minor] `import-grouping` main.go:4-12 has a stray blank line and the store import inside the stdlib group
- **BR-15** [Minor] `fix-appended-not-integrated` The BR-4 README fix was appended to the sh block instead of integrated into the key table
- **BR-16** [Minor] `loop-collapses-callee-exit-code` A piped unknown command exits 1 where dispatchCommand computes 2 and README documents 2
