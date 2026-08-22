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
    - "n": 4
      timestamp: "2026-08-21T17:17:54-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: The plan now carries a per-function "what its test exists to catch" table (plan lines 154-165) plus one for M3.
          round: 4
        - id: BR-7
          disposition: addressed
          note: width is read by renderHistory and menuLines, and newCommandCtx takes opt.width rather than re-deriving via terminalWidth.
          round: 4
        - id: BR-8
          disposition: addressed
          note: 'Mutation-verified: restoring strings.ToLower(prefix) in commandCompletions reddens TestCommandCompletions.'
          round: 4
        - id: BR-9
          disposition: addressed
          note: 'Mutation-verified: restoring dispatchCommand''s len(near) != len(cmds) inference reddens TestNearMissWithOneRegisteredCommand.'
          round: 4
        - id: BR-10
          disposition: addressed
          note: TestEveryRegisteredCommandIsRunnable iterates the live registry and Fatals on an empty one, so it cannot pass vacuously.
          round: 4
        - id: BR-11
          disposition: addressed
          note: Both arg comparisons now use reflect.DeepEqual; sameSet survives only in TestCompletionsFor, where the data really is unordered.
          round: 4
        - id: BR-12
          disposition: addressed
          note: The count is gone from command_test.go:130 — the comment no longer names a number that a fifth call site can falsify.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: Behaviour verified correct for `define /help`, but no test pins it (deleting main.go:292-294 leaves the suite green) and `define /history 7` still prints a usage dump — see the entry-mode-inconsistency rule finding.
          round: 4
        - id: BR-14
          disposition: addressed
          note: main.go:4-13 now has stdlib in one group and the store import in the second, no stray blank line.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: A prose section on the / surface did land (README:92-115), but the editor key table (README:42-51) still has no / row and its "Enter | define what you typed" row is still incomplete — the specific structure the finding named is untouched.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: 'The cmdCode plumbing is present and `echo /qqqqqq | define` exits 2 as probed, but nothing pins it: deleting the `if cmdCode != 0 { return cmdCode }` block leaves the suite green (mutation applied, BUILD_OK, -count=1 ok).'
          round: 4
      findings:
        - id: BR-17
          severity: Critical
          title: relativeDay divides a Duration by 24h, so every /history date is off by one for a week after each spring-forward
          detail: |-
            history_cmd.go:191 computes `int(dayOf(now).Sub(dayOf(at)).Hours() / 24)`. Two local
            midnights one calendar day apart are 23h across a spring-forward, and int(23.0/24) is 0.
            Measured against the shipped function body in America/Los_Angeles: at=2026-03-08,
            now=2026-03-09 prints "today" for a lookup that was yesterday; at=2026-03-07 prints
            "yesterday" for two days ago; at=2026-03-02, now=2026-03-09 prints "Monday" though the
            days<7 rule should give "Mar 2". The function's own comment says it is computed on local
            calendar days rather than elapsed hours, which is the contract it breaks, and it is a
            second implementation of what historyWindow already gets right with AddDate (ARCH-DRY).
            TestRenderHistoryRelativeDatesAreCalendarDays has two rows, both in August, while the
            sibling TestHistoryWindow has four DST rows and the file already embeds time/tzdata.
          family: elapsed-hours-for-calendar-days
          round: 4
        - id: BR-18
          severity: Critical
          title: The developer's deck is committed under cmd/define/, and the conformance suite rewrites those tracked files
          detail: |-
            cmd/define/events/2026-08-21.yaml, events/2026-08-22.yaml and words/sycophantic.yaml are
            tracked, added by fc071af and 1ff0d5e — two separate commits in this window. No test or
            fixture references them. Reproduced: after `go test -tags conformance -run PTY`, git status
            reports both events/2026-08-22.yaml and words/sycophantic.yaml MODIFIED (lookups 12 to 14),
            because pty_conformance_test.go:47 launches ../../bin/define with the test's cwd, and define
            writes its deck to the current directory. So the suite is not idempotent against the index
            and `git add -A` sweeps deck churn into unrelated commits. .gitignore:16-19 anticipates this
            class but anchors at the root, which does not cover cmd/*/. It is also the ARCH-MOCK gap:
            the live conformance flow and the in-process fake do not share a storage boundary.
          family: runtime-output-tracked-in-source-tree
          round: 4
        - id: BR-19
          severity: Important
          title: Four loop-shell wirings shipped this round with no test that fails without them
          detail: |-
            This is the 4th finding in family unpinned-production-wiring (BR-2 first, BR-16's fix
            another). Do NOT fix the four instances individually. Measured, each mutation applied with
            BUILD_OK and the full suite green at -count=1: replraw.go:166 cc.setTimes deleted;
            main.go:292-294 one-shot dispatch deleted; repl.go:139-142 cmdCode return deleted;
            replraw.go:167 hist.Add deleted. The setTimes one is the sharpest — without it, /sound 1 at
            the real TUI prompt prints "needs an interactive session", the opposite of M3's requirement,
            and TestSoundChangesPlaybackForTheRestOfTheSession drives replLines, the piped loop.
            THE RULE - a value or effect that only a loop shell supplies must be pinned by a test that
            drives that loop shell (runEditor, replLines, run), never by one that builds the callee's
            context by hand. Every commandCtx literal in a test (sound_cmd_test.go:52,64,77,88;
            commandloop_test.go:131; history_cmd_test.go:271,354) is where the rule breaks invisibly.
            The rule-level fix is one table in commandloop_test.go running the same command scripts
            through BOTH runEditor and replLines, so a field wired in one loop and not the other reddens
            by construction.
          family: unpinned-production-wiring
          round: 4
        - id: BR-20
          severity: Important
          title: define /history 7 prints a usage dump while echo '/history 7' | define runs it
          detail: |-
            This is the 2nd finding in family entry-mode-inconsistency (BR-13 first). Do NOT patch this
            instance. Probed against the built binary - `define /history 7` exits 2 with the full usage
            text; `echo '/history 7' | define` prints "nothing looked up in the last 7 days" and exits 0.
            main.go:266 rejects fs.NArg() > 1 before main.go:292 tests for a command, and main.go:292
            classifies fs.Arg(0) alone rather than a line. README:96 claims all three entry modes are
            "one thing" and atlas/define.md says "Every entry mode reaches it" — true only for
            zero-argument commands.
            THE RULE - every entry mode must hand parseREPLLine the same input, a whole LINE. The two
            loops do; the argv path hands it one token. Classify strings.Join(fs.Args(), " ") when
            fs.Arg(0) opens command mode and dispatch before the arity guard, rather than extending the
            arity guard each time a command grows an argument. BR-13's fix stopped at the zero-argument
            case, which is why the family recurred.
          family: entry-mode-inconsistency
          round: 4
        - id: BR-21
          severity: Important
          title: The plan's Core-concepts table names none of the entities the two scope events added, and six step boxes are unticked
          detail: |-
            This is the 2nd finding in family plan-artifact-lags-code (BR-5 first). Do NOT just tick the
            boxes. menuLines and menuNameWidth (M1b) and parseSoundArgs, runSound, soundTimes and the
            whole sound_cmd.go file (M3) appear in no Core-concepts row, though three of them do appear
            in the test-strategy tables. Six "- [ ]" boxes remain at plan lines 277, 300, 311, 312, 326
            — Task 6 Step 3 and the Step-5 commit boxes of Tasks 4-7 — which sdlc close's plan-unchecked
            guard reads.
            THE RULE - the Core-concepts table is the greppable contract this gate cross-checks against
            the filesystem, so a scope event that adds an entity adds its row in the same commit as the
            code, and a task's boxes are ticked by the commit that lands it. The M1b and M3 Revisions
            entries describe the work in prose but never amend the table, which is what made both gaps
            possible.
          family: plan-artifact-lags-code
          round: 4
        - id: BR-22
          severity: Minor
          title: --sound 1000 is accepted while /sound 1000 is refused at the 20 cap
          detail: |-
            Probed both. sound_cmd.go:33 bounds /sound at maxSoundTimes = 20; main.go:228 rejects only
            negatives for -sound/-times. atlas/define.md calls them the same setting.
          family: same-setting-two-validation-policies
          round: 4
        - id: BR-23
          severity: Minor
          title: 'define -times -1 reports "define: -sound must not be negative"'
          detail: |-
            main.go:229. main_test.go:253 asserts only the exit code, so the message text is unpinned in
            either direction.
          family: error-names-a-flag-the-user-did-not-type
          round: 4
        - id: BR-24
          severity: Minor
          title: menuLines truncates in bytes while renderHistory truncates in runes, and renderHistory measures columns in bytes but pads in runes
          detail: |-
            command.go:239 slices line[:width] (bytes, can split a rune); history_cmd.go:172 slices
            runes. history_cmd.go's nameW/dateW use len() while %-*s pads in runes, so a non-ASCII
            headword widens the whole gutter — alignment survives, the column is just wider than asked
            for. One shared truncateToWidth helper covers both (ARCH-DRY).
          family: duplicated-width-arithmetic
          round: 4
        - id: BR-25
          severity: Minor
          title: renderHistory's truncation branch is never exercised — both test call sites pass width 0
          detail: |-
            history_cmd_test.go:227 and :250 pass 0, and nothing asserts runHistory forwards c.width.
            The equivalent branch in menuLines is covered by TestMenuLines' width=14 row.
          family: uncovered-branch-at-boundary
          round: 4
      blocked: true
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

## Round 4 — 2026-08-21T17:17:54-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — The plan now carries a per-function "what its test exists to catch" table (plan lines 154-165) plus one for M3.
- BR-7 — addressed — width is read by renderHistory and menuLines, and newCommandCtx takes opt.width rather than re-deriving via terminalWidth.
- BR-8 — addressed — Mutation-verified: restoring strings.ToLower(prefix) in commandCompletions reddens TestCommandCompletions.
- BR-9 — addressed — Mutation-verified: restoring dispatchCommand's len(near) != len(cmds) inference reddens TestNearMissWithOneRegisteredCommand.
- BR-10 — addressed — TestEveryRegisteredCommandIsRunnable iterates the live registry and Fatals on an empty one, so it cannot pass vacuously.
- BR-11 — addressed — Both arg comparisons now use reflect.DeepEqual; sameSet survives only in TestCompletionsFor, where the data really is unordered.
- BR-12 — addressed — The count is gone from command_test.go:130 — the comment no longer names a number that a fifth call site can falsify.
- BR-13 — not-addressed — Behaviour verified correct for `define /help`, but no test pins it (deleting main.go:292-294 leaves the suite green) and `define /history 7` still prints a usage dump — see the entry-mode-inconsistency rule finding.
- BR-14 — addressed — main.go:4-13 now has stdlib in one group and the store import in the second, no stray blank line.
- BR-15 — not-addressed — A prose section on the / surface did land (README:92-115), but the editor key table (README:42-51) still has no / row and its "Enter | define what you typed" row is still incomplete — the specific structure the finding named is untouched.
- BR-16 — not-addressed — The cmdCode plumbing is present and `echo /qqqqqq | define` exits 2 as probed, but nothing pins it: deleting the `if cmdCode != 0 { return cmdCode }` block leaves the suite green (mutation applied, BUILD_OK, -count=1 ok).

### Raised

- **BR-17** [Critical] `elapsed-hours-for-calendar-days` relativeDay divides a Duration by 24h, so every /history date is off by one for a week after each spring-forward
  history_cmd.go:191 computes `int(dayOf(now).Sub(dayOf(at)).Hours() / 24)`. Two local
  midnights one calendar day apart are 23h across a spring-forward, and int(23.0/24) is 0.
  Measured against the shipped function body in America/Los_Angeles: at=2026-03-08,
  now=2026-03-09 prints "today" for a lookup that was yesterday; at=2026-03-07 prints
  "yesterday" for two days ago; at=2026-03-02, now=2026-03-09 prints "Monday" though the
  days<7 rule should give "Mar 2". The function's own comment says it is computed on local
  calendar days rather than elapsed hours, which is the contract it breaks, and it is a
  second implementation of what historyWindow already gets right with AddDate (ARCH-DRY).
  TestRenderHistoryRelativeDatesAreCalendarDays has two rows, both in August, while the
  sibling TestHistoryWindow has four DST rows and the file already embeds time/tzdata.
- **BR-18** [Critical] `runtime-output-tracked-in-source-tree` The developer's deck is committed under cmd/define/, and the conformance suite rewrites those tracked files
  cmd/define/events/2026-08-21.yaml, events/2026-08-22.yaml and words/sycophantic.yaml are
  tracked, added by fc071af and 1ff0d5e — two separate commits in this window. No test or
  fixture references them. Reproduced: after `go test -tags conformance -run PTY`, git status
  reports both events/2026-08-22.yaml and words/sycophantic.yaml MODIFIED (lookups 12 to 14),
  because pty_conformance_test.go:47 launches ../../bin/define with the test's cwd, and define
  writes its deck to the current directory. So the suite is not idempotent against the index
  and `git add -A` sweeps deck churn into unrelated commits. .gitignore:16-19 anticipates this
  class but anchors at the root, which does not cover cmd/*/. It is also the ARCH-MOCK gap:
  the live conformance flow and the in-process fake do not share a storage boundary.
- **BR-19** [Important] `unpinned-production-wiring` Four loop-shell wirings shipped this round with no test that fails without them
  This is the 4th finding in family unpinned-production-wiring (BR-2 first, BR-16's fix
  another). Do NOT fix the four instances individually. Measured, each mutation applied with
  BUILD_OK and the full suite green at -count=1: replraw.go:166 cc.setTimes deleted;
  main.go:292-294 one-shot dispatch deleted; repl.go:139-142 cmdCode return deleted;
  replraw.go:167 hist.Add deleted. The setTimes one is the sharpest — without it, /sound 1 at
  the real TUI prompt prints "needs an interactive session", the opposite of M3's requirement,
  and TestSoundChangesPlaybackForTheRestOfTheSession drives replLines, the piped loop.
  THE RULE - a value or effect that only a loop shell supplies must be pinned by a test that
  drives that loop shell (runEditor, replLines, run), never by one that builds the callee's
  context by hand. Every commandCtx literal in a test (sound_cmd_test.go:52,64,77,88;
  commandloop_test.go:131; history_cmd_test.go:271,354) is where the rule breaks invisibly.
  The rule-level fix is one table in commandloop_test.go running the same command scripts
  through BOTH runEditor and replLines, so a field wired in one loop and not the other reddens
  by construction.
- **BR-20** [Important] `entry-mode-inconsistency` define /history 7 prints a usage dump while echo '/history 7' | define runs it
  This is the 2nd finding in family entry-mode-inconsistency (BR-13 first). Do NOT patch this
  instance. Probed against the built binary - `define /history 7` exits 2 with the full usage
  text; `echo '/history 7' | define` prints "nothing looked up in the last 7 days" and exits 0.
  main.go:266 rejects fs.NArg() > 1 before main.go:292 tests for a command, and main.go:292
  classifies fs.Arg(0) alone rather than a line. README:96 claims all three entry modes are
  "one thing" and atlas/define.md says "Every entry mode reaches it" — true only for
  zero-argument commands.
  THE RULE - every entry mode must hand parseREPLLine the same input, a whole LINE. The two
  loops do; the argv path hands it one token. Classify strings.Join(fs.Args(), " ") when
  fs.Arg(0) opens command mode and dispatch before the arity guard, rather than extending the
  arity guard each time a command grows an argument. BR-13's fix stopped at the zero-argument
  case, which is why the family recurred.
- **BR-21** [Important] `plan-artifact-lags-code` The plan's Core-concepts table names none of the entities the two scope events added, and six step boxes are unticked
  This is the 2nd finding in family plan-artifact-lags-code (BR-5 first). Do NOT just tick the
  boxes. menuLines and menuNameWidth (M1b) and parseSoundArgs, runSound, soundTimes and the
  whole sound_cmd.go file (M3) appear in no Core-concepts row, though three of them do appear
  in the test-strategy tables. Six "- [ ]" boxes remain at plan lines 277, 300, 311, 312, 326
  — Task 6 Step 3 and the Step-5 commit boxes of Tasks 4-7 — which sdlc close's plan-unchecked
  guard reads.
  THE RULE - the Core-concepts table is the greppable contract this gate cross-checks against
  the filesystem, so a scope event that adds an entity adds its row in the same commit as the
  code, and a task's boxes are ticked by the commit that lands it. The M1b and M3 Revisions
  entries describe the work in prose but never amend the table, which is what made both gaps
  possible.
- **BR-22** [Minor] `same-setting-two-validation-policies` --sound 1000 is accepted while /sound 1000 is refused at the 20 cap
  Probed both. sound_cmd.go:33 bounds /sound at maxSoundTimes = 20; main.go:228 rejects only
  negatives for -sound/-times. atlas/define.md calls them the same setting.
- **BR-23** [Minor] `error-names-a-flag-the-user-did-not-type` define -times -1 reports "define: -sound must not be negative"
  main.go:229. main_test.go:253 asserts only the exit code, so the message text is unpinned in
  either direction.
- **BR-24** [Minor] `duplicated-width-arithmetic` menuLines truncates in bytes while renderHistory truncates in runes, and renderHistory measures columns in bytes but pads in runes
  command.go:239 slices line[:width] (bytes, can split a rune); history_cmd.go:172 slices
  runes. history_cmd.go's nameW/dateW use len() while %-*s pads in runes, so a non-ASCII
  headword widens the whole gutter — alignment survives, the column is just wider than asked
  for. One shared truncateToWidth helper covers both (ARCH-DRY).
- **BR-25** [Minor] `uncovered-branch-at-boundary` renderHistory's truncation branch is never exercised — both test call sites pass width 0
  history_cmd_test.go:227 and :250 pass 0, and nothing asserts runHistory forwards c.width.
  The equivalent branch in menuLines is covered by TestMenuLines' width=14 row.

## Open findings

- **BR-13** [Minor] `entry-mode-inconsistency` define /help as a one-shot argument still goes to the dictionary
- **BR-15** [Minor] `fix-appended-not-integrated` The BR-4 README fix was appended to the sh block instead of integrated into the key table
- **BR-16** [Minor] `loop-collapses-callee-exit-code` A piped unknown command exits 1 where dispatchCommand computes 2 and README documents 2
- **BR-17** [Critical] `elapsed-hours-for-calendar-days` relativeDay divides a Duration by 24h, so every /history date is off by one for a week after each spring-forward
- **BR-18** [Critical] `runtime-output-tracked-in-source-tree` The developer's deck is committed under cmd/define/, and the conformance suite rewrites those tracked files
- **BR-19** [Important] `unpinned-production-wiring` Four loop-shell wirings shipped this round with no test that fails without them
- **BR-20** [Important] `entry-mode-inconsistency` define /history 7 prints a usage dump while echo '/history 7' | define runs it
- **BR-21** [Important] `plan-artifact-lags-code` The plan's Core-concepts table names none of the entities the two scope events added, and six step boxes are unticked
- **BR-22** [Minor] `same-setting-two-validation-policies` --sound 1000 is accepted while /sound 1000 is refused at the 20 cap
- **BR-23** [Minor] `error-names-a-flag-the-user-did-not-type` define -times -1 reports "define: -sound must not be negative"
- **BR-24** [Minor] `duplicated-width-arithmetic` menuLines truncates in bytes while renderHistory truncates in runes, and renderHistory measures columns in bytes but pads in runes
- **BR-25** [Minor] `uncovered-branch-at-boundary` renderHistory's truncation branch is never exercised — both test call sites pass width 0
