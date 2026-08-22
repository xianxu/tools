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
    - "n": 5
      timestamp: "2026-08-21T17:39:51-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: addressed
          note: 'Probed: `define /help` runs and exits 0; pinned by TestOneShotCommandGoesThroughRun (deleting the dispatch block reddens it).'
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: The sh block is fine now, but README.md:43-50 still has no / row and "Enter | define what you typed" still omits command dispatch — the exact structure the finding named.
          round: 5
        - id: BR-16
          disposition: addressed
          note: Probed `echo /qqqqqq | define` exits 2; mutation-verified — deleting the cmdCode return reddens TestPipedLoopReturnsTheCommandsExitCode.
          round: 5
        - id: BR-17
          disposition: addressed
          note: 'Mutation-verified: reverting math.Round to truncation reddens three DST rows. Residual exactness gap raised separately as a Minor.'
          round: 5
        - id: BR-18
          disposition: addressed
          note: Files untracked, TestNoTrackedRuntimeState plant-verified to fail, and a real-pty conformance run leaves git status clean with no deck under cmd/define/. History residue raised separately.
          round: 5
        - id: BR-19
          disposition: not-addressed
          note: All four instances are now mutation-verified pinned, but the RULE the finding demanded is absent from workshop/lessons.md and was violated in the same commit — BR-20's fix and clearMenu are both unpinned (measured).
          round: 5
        - id: BR-20
          disposition: not-addressed
          note: 'Behaviour correct when probed, but unpinned: reverting the arity-guard exemption restores the reported usage dump, and classifying fs.Arg(0) makes /history 7 print a 2-day window — full suite green under both.'
          round: 5
        - id: BR-21
          disposition: addressed
          note: Zero unticked boxes remain, and every entity named in the Core-concepts table exists at its stated path (grep-verified).
          round: 5
        - id: BR-22
          disposition: not-addressed
          note: 'Re-probed: --sound 1000 accepted, /sound 1000 refused at the 20 cap.'
          round: 5
        - id: BR-23
          disposition: not-addressed
          note: 'Re-probed: define -times -1 still reports "define: -sound must not be negative".'
          round: 5
        - id: BR-24
          disposition: not-addressed
          note: command.go:239 still slices bytes, history_cmd.go:178 still slices runes.
          round: 5
        - id: BR-25
          disposition: not-addressed
          note: Both renderHistory call sites still pass width 0, and nothing asserts runHistory forwards c.width.
          round: 5
      findings:
        - id: BR-26
          severity: Important
          title: A command line skips the "usage errors are settled before a store is opened" invariant and reads the whole event log
          detail: |-
            main.go:272 exempts cmdCommand from the arity guard, so a command line reaches
            main.go:279 d.withStore before anything validates it. Measured with the same torn-log
            fixture TestUsageErrorsDoNotOpenTheLog uses - `define /qqqqqq`, `define /qqqqqq zzz`
            and `define /history zzz` all print "recovered 0 event(s), dropped 1 torn record(s)"
            before their usage error. main.go:252 states the contract four lines above the guard
            that breaks it. The directory-creation half still holds (NewYAML is lazy, confirmed),
            but the read half does not, and capture_test.go:423 has rows for -forget and two words
            and none for a command, so the drift is untested in either direction. Fix by resolving
            the command name against `commands` before withStore, and add the three rows.
          family: guard-bypassed-by-new-kind
          round: 5
        - id: BR-27
          severity: Important
          title: --days and --days=N ship and are tested but appear in no user-facing doc, while the plan box promising them is ticked
          detail: |-
            This is the 2nd finding in family docs-consumer-not-updated (BR-4 first). Do NOT patch
            this instance. parseHistoryArgs accepts 7, --days 7 and --days=7, all pinned by
            TestParseHistoryArgs; grep finds zero occurrences of --days in README.md, atlas/define.md
            and the --help text, and /help's summary is "words looked up recently". Only the
            positional [N] form is documented. Plan line 352 reads "- [x] Step 2: README: what
            /history shows, what it omits and why, --days" - a ticked box asserting the one
            deliverable that did not land.
            THE RULE - a flag or argument form is not shipped until at least one consumer a user can
            reach derives it, and the plan box is ticked by the commit that updates that consumer,
            not by the commit that adds the parser (ARCH-PURPOSE shadow-sweep: parser and tests
            derive, README/atlas/--help//help do not).
          family: docs-consumer-not-updated
          round: 5
        - id: BR-28
          severity: Important
          title: The deck blobs are still reachable from HEAD, and the guard written for BR-18 checks only the index
          detail: |-
            This is the 2nd finding in family runtime-output-tracked-in-source-tree (BR-18 first). Do
            NOT patch this instance. `git show 1ff0d5e:cmd/define/words/sycophantic.yaml` still
            returns the word with its timestamps and lookups: 12; the two events files likewise, added
            by fc071af and 1ff0d5e. 4b019e0 removed them from the index only. TestNoTrackedRuntimeState
            reads `git ls-files` - the index - while its sibling TestNoBinariesInHistory walks history
            precisely because, in its own words, deleting in a later commit does not remove the cost.
            THE RULE - a guard for a committed-artifact class must check the scope where that class's
            cost lives: the index for what is checked out, history for what every clone fetches. A fix
            that only untracks is incomplete against a history-scoped guard. Cheap now while the branch
            is unmerged; permanent after.
          family: runtime-output-tracked-in-source-tree
          round: 5
        - id: BR-29
          severity: Minor
          title: relativeDay's rounding is a heuristic where an exact computation is one line away, and the comment asserts an exactness the code lacks
          detail: |-
            This is the 2nd finding in family elapsed-hours-for-calendar-days (BR-17 first). Do NOT
            patch this instance. history_cmd.go:199's comment claims "The true gap is always N days
            +/- 1 hour, which makes rounding exact"; measured in Pacific/Apia, at=2011-12-29,
            now=2011-12-31 prints "yesterday" for a two-calendar-day gap, because the 2011 date-line
            change deleted 2011-12-30 and only 24 hours elapsed.
            THE RULE - count calendar days on calendar-day numbers, never on elapsed time: normalise
            both dates into a fixed-offset zone before differencing, so no offset arithmetic enters
            the count at all. Verified - building dayOf in time.UTC and dropping math.Round returns
            "Thursday" for Apia, keeps the full suite green including all four DST rows, and removes
            the math import.
          family: elapsed-hours-for-calendar-days
          round: 5
        - id: BR-30
          severity: Minor
          title: replraw.go:67 still claims candidates are resolved once per keystroke, but draw() now computes its own list
          detail: |-
            This is the 2nd finding in family comment-orphaned-by-insertion (BR-12 first). Do NOT patch
            this instance. The comment reads "Resolve candidates ONCE per keystroke and use the same
            slice for both the state machine and the suggestion. Querying twice doubled the work the
            History seam will do once #3 backs it with a store." Since 1ff0d5e removed draw's parameter,
            completionsFor runs twice per keystroke - once for Apply at replraw.go:129 and once inside
            draw - and #3 has landed.
            THE RULE - when a fix changes what a block does, the comment above it is part of the diff.
            A comment that survives a behaviour change is a false claim about the code beneath it,
            which is exactly what let BR-17's DST bug read as correct for four rounds.
          family: comment-orphaned-by-insertion
          round: 5
      blocked: true
    - "n": 6
      timestamp: "2026-08-21T18:00:13-07:00"
      agent: claude
      dispose:
        - id: BR-15
          disposition: not-addressed
          note: README.md:43-50 still has no / row and "Enter | define what you typed (never the suggestion)" still omits command dispatch.
          round: 6
        - id: BR-19
          disposition: addressed
          note: Rule recorded at lessons.md:367; both new instances mutation-verified (arity exemption and clearMenu each redden a loop-driving test).
          round: 6
        - id: BR-20
          disposition: addressed
          note: Both halves mutation-verified — reverting the arity exemption and classifying fs.Arg(0) each redden TestOneShotCommandTakesArguments.
          round: 6
        - id: BR-22
          disposition: not-addressed
          note: Re-probed against the built binary — "define --sound 1000 /sound" prints "playing 1000x" and exits 0.
          round: 6
        - id: BR-23
          disposition: not-addressed
          note: 'Re-probed — "define -times -1 x" still reports "define: -sound must not be negative".'
          round: 6
        - id: BR-24
          disposition: not-addressed
          note: command.go:255 still slices bytes, history_cmd.go:177 still slices runes.
          round: 6
        - id: BR-25
          disposition: not-addressed
          note: Both renderHistory call sites still pass width 0, and nothing asserts runHistory forwards c.width.
          round: 6
        - id: BR-26
          disposition: not-addressed
          note: Two of the three measured cases are fixed and mutation-pinned, but "define /history zzz" still reads the whole log before its usage error, and that is the one row the finding asked for that was not added.
          round: 6
        - id: BR-27
          disposition: addressed
          note: --days now derives in three reachable consumers — README.md:102, atlas/define.md:401, and the --help text at main.go:206.
          round: 6
        - id: BR-28
          disposition: addressed
          note: git rev-list --objects HEAD returns zero deck objects, the three named commits no longer exist, and TestNoRuntimeStateInHistory covers the history scope.
          round: 6
        - id: BR-29
          disposition: not-addressed
          note: history_cmd.go:199 still uses math.Round on elapsed hours, and the exactness comment is unchanged.
          round: 6
        - id: BR-30
          disposition: not-addressed
          note: replraw.go:67-69 unchanged; completionsFor still runs twice per keystroke, at :113 and :129.
          round: 6
      findings:
        - id: BR-31
          severity: Important
          title: The needsDeck exemption strands deps.clock, so a registry row with needsDeck false gets a nil clock and panics
          detail: |-
            This is the 2nd finding in family guard-bypassed-by-new-kind (BR-26 first). Do NOT patch
            this instance. main.go:285 skips withStore for a command that does not need the deck, and
            withStore (main.go:99) is the ONLY thing that ever sets deps.clock - realDeps at main.go:50
            does not. Measured: adding a registry row with needsDeck false whose run calls c.clock.Now()
            makes "define -no-audio /probe" panic with a nil pointer dereference, while the full suite
            stays green. capture_test.go:468 and :494 exist to prevent exactly this and both pass,
            because they exercise openStore and withStore, the path the exemption routes around. The
            issue's Done-when "adding a second command needs no change to the dispatch loop" now carries
            an undocumented precondition that neither the commandCtx doc comment nor atlas/define.md's
            needsDeck paragraph states.
            THE RULE - when a new kind is exempted from a shared setup path, enumerate everything that
            path guaranteed and re-supply it; the exemption drops invariants you were not thinking about,
            not just the cost you were avoiding. Rule-level fix - the clock has no store dependency, so
            supply it outside withStore (in run before the branch, or as a fallback in newCommandCtx) so
            no exemption can strand it, and extend TestEveryRegisteredCommandIsRunnable to assert the ctx
            a one-shot actually builds carries every field a row may read.
          family: guard-bypassed-by-new-kind
          round: 6
        - id: BR-32
          severity: Important
          title: /history reads the whole event log twice and prints every store warning twice
          detail: |-
            Measured against a torn-log fixture - "define -no-audio /history" and "echo /history |
            define" each emit "recovered 1 event(s), dropped 1 torn record(s)" TWICE before the
            output, while a plain word lookup emits it once and "define /help" not at all. Read 1 is
            newStoreHistory calling st.Events(time.Time{}) at history_store.go:31, built by withStore
            because commandNeedsDeck said yes; read 2 is runHistory calling c.deck.Events(time.Time{})
            at history_cmd.go:228. In the one-shot path the first read is pure waste, and replLines
            never touches d.history at all so it is waste in the piped path too. atlas/define.md:446
            and plan line 374 both record "One read per /history on a personal word list is the right
            trade today" as the accepted cost, so the recorded trade is not the one the code makes.
            Fix by gating on the specific dependency rather than the whole store bundle, or by handing
            runHistory the events storeHistory already read, and assert the warning COUNT in
            TestCommandThatReadsNothingDoesNotOpenTheLog, which today only checks presence.
          family: same-source-read-twice
          round: 6
        - id: BR-33
          severity: Important
          title: The atlas never learned about the runtime-state guards, and the README still teaches the deprecated flag name
          detail: |-
            This is the 3rd finding in family docs-consumer-not-updated (BR-4, BR-27). Do NOT patch
            these instances. Measured prevalence, both from this window - (1) atlas/repo-guards.md
            documents one guard class with an index/history table naming TestNoCommittedBinaries and
            TestNoBinariesInHistory; this window added a second class with the identical split
            (TestNoTrackedRuntimeState / TestNoRuntimeStateInHistory) and neither that page nor
            atlas/index.md:13-14 ("no executable image in the index or reachable from HEAD") mentions
            it, so the index understates what the file it points at enforces. (2) README.md:84 still
            reads "Flags are session settings - define -times 1 opens the loop with single playback",
            teaching the name README.md:118 calls "the older name for --sound".
            THE RULE - a surface is not shipped until every doc that already describes its class is
            updated in the same commit, where "its class" means the page that is wrong by omission,
            not only the page that names the new thing. BR-4 updated the README because the finding
            named the README and BR-27 updated three places because the finding listed three; neither
            adopted the sweep. Enforcement worth having (ARCH-PURPOSE) - atlas/repo-guards.md's table
            is a hand-maintained restatement of the Test functions in repo_guard_test.go, so a test
            that greps the table for every "func TestNo..." in that file makes the page derive.
          family: docs-consumer-not-updated
          round: 6
        - id: BR-34
          severity: Important
          title: The Core-concepts table lost five entities again, and the plan has no Revisions entry for rounds 5 or 6
          detail: |-
            This is the 3rd finding in family plan-artifact-lags-code (BR-5, BR-21). Do NOT just add
            the rows. commandNeedsDeck, command.needsDeck, editDistance, historyPaths and
            TestNoRuntimeStateInHistory appear in no row (grep-verified, zero occurrences of each name
            in the plan); four of the five were added by e5719f9, the commit that closed BR-21's own
            family. The plan's last Revisions heading is "close round 4", so rounds 5 and 6 - which
            produced the needsDeck design, the history rewrite and the --days docs - are unrecorded.
            Box-ticking is clean and every entity the table DOES name exists at its stated path.
            THE RULE - the Core-concepts table is a hand-maintained restatement of what the package
            declares, so it drifts by default; the fix is to make it derive, not to remember harder.
            BR-21 was disposed addressed and the very next commit re-broke it, which is the measurement
            that a habit-level fix does not hold. A guard comparing the table's Name-and-path pairs
            against top-level declarations in the named files would fail the same commit that adds an
            unnamed entity - the same shape as TestNoTrackedRuntimeState.
          family: plan-artifact-lags-code
          round: 6
        - id: BR-35
          severity: Minor
          title: Two registry lookups carry independently-written matching rules and nothing pins that they agree
          detail: |-
            This is the 2nd finding in family parallel-construction-drift (BR-6 first). Do NOT patch
            this instance. commandNeedsDeck (command.go:192-199) and dispatchCommand (command.go:211-215)
            each walk cmds with strings.EqualFold. Mutating commandNeedsDeck to an exact comparison
            leaves the ENTIRE suite green while "define /HISTORY" reports "define: no deck in this
            directory" and exits 1 - a lie about the user's data - because dispatch still accepts
            loosely and the deck gate no longer does. atlas/define.md states the loose-accept policy as
            a design decision, so the two lookups implementing it must agree by construction.
            THE RULE - two sites that must agree about one fact should be one site: a single
            findCommand(name, cmds) (command, bool) that both call, so the matching rule cannot drift.
          family: parallel-construction-drift
          round: 6
      blocked: true
    - "n": 7
      timestamp: "2026-08-21T18:10:14-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
    - "n": 8
      timestamp: "2026-08-21T18:24:16-07:00"
      agent: claude
      dispose:
        - id: BR-15
          disposition: not-addressed
          note: README.md:42-49 still has no / row and "Enter | define what you typed (never the suggestion)" still omits command dispatch.
          round: 8
        - id: BR-22
          disposition: not-addressed
          note: Re-probed against the built binary — "define --sound 1000 /sound" prints "playing 1000x" and exits 0, while "/sound 1000" is refused at 20.
          round: 8
        - id: BR-23
          disposition: not-addressed
          note: 'Re-probed — "define -times -1 x" still reports "define: -sound must not be negative".'
          round: 8
        - id: BR-24
          disposition: not-addressed
          note: command.go:238-239 still slices bytes, history_cmd.go:177-178 still slices runes, and runHelp's hardcoded %-10s at command.go:211 is now a third name-column width beside menuNameWidth.
          round: 8
        - id: BR-25
          disposition: not-addressed
          note: history_cmd_test.go:225 and :268 still pass width 0, and nothing asserts runHistory forwards c.width.
          round: 8
        - id: BR-26
          disposition: addressed
          note: Probed with the torn-log fixture — /qqqqqq, /qqqqqq zzz and /history zzz all read nothing now; mutation-verified (reverting Load to eager reddens TestCommandThatReadsNothingDoesNotOpenTheLog). The three rows were still not added; raised separately.
          round: 8
        - id: BR-29
          disposition: not-addressed
          note: Reproduced in Pacific/Apia — at=2011-12-29, now=2011-12-31 prints "yesterday" for a two-calendar-day gap. history_cmd.go:199 unchanged, exactness comment unchanged.
          round: 8
        - id: BR-30
          disposition: not-addressed
          note: replraw.go:70 unchanged; completionsFor still runs twice per keystroke, at :113 and :129.
          round: 8
        - id: BR-31
          disposition: addressed
          note: needsDeck deleted and withStore is unconditional; mutation-verified two ways — removing withStore's clock fallback reddens TestWithStoreCarriesTheClock, and nil-ing clock in newCommandCtx panics the suite through run().
          round: 8
        - id: BR-32
          disposition: addressed
          note: Probed — /history warns ONCE now, one-shot and piped; pinned by the Load-laziness mutation. The warning-COUNT assertion the finding asked for is still absent; raised separately.
          round: 8
        - id: BR-33
          disposition: not-addressed
          note: repo-guards.md gained the section and README.md:84 now teaches --sound, but atlas/index.md:13-14 — named explicitly in the finding — still mentions only the executable-image class, and the same commit created three fresh stale claims in atlas/define.md.
          round: 8
        - id: BR-34
          disposition: not-addressed
          note: commandNeedsDeck and needsDeck became correct by deletion rather than maintenance; historyPaths and TestNoRuntimeStateInHistory landed; editDistance is still absent (grep-verified), and the derive-guard the rule called for was not written.
          round: 8
        - id: BR-35
          disposition: addressed
          note: commandNeedsDeck is deleted, so exactly one strings.EqualFold walk over cmds remains (command.go:195) — the two rules cannot drift by construction.
          round: 8
      findings:
        - id: BR-36
          severity: Important
          title: atlas/define.md documents needsDeck, a field the same commit deleted, and two more paragraphs describe a load that no longer happens at construction
          detail: |-
            This is the 4th finding in family docs-consumer-not-updated (BR-4, BR-27, BR-33). Do NOT
            patch these instances. Measured prevalence, all four from 8f04434 - the commit whose
            message closes BR-33: (1) atlas/define.md:404-405 "A command declares needsDeck when it
            reads the store. Opening the store constructs storeHistory, which READS the whole event
            log" - grep needsDeck over cmd/ returns zero hits and opening the store reads nothing;
            (2) atlas/define.md:307 "loads the log once at construction"; (3) atlas/define.md:284
            "since #4 it only reads at construction" - both falsified by History.Load; (4)
            atlas/index.md:13-14 still names only the executable-image class, the exact instance
            BR-33 listed and the one half of it that was not fixed.
            THE RULE, stated by BR-33 and still unadopted - a surface is not shipped until every doc
            that already describes its class is updated in the same commit, where "its class" means
            the page that is wrong by omission or contradiction, not only the page that names the new
            thing. Three rounds of fixing exactly the pages a finding lists is the measurement that
            the sweep is not happening. Cheap durable half - when a commit DELETES an identifier,
            grep atlas/ and README.md for it before committing; enforcement worth having
            (ARCH-PURPOSE) is the repo-guards table guard BR-33 already described.
          family: docs-consumer-not-updated
          round: 8
        - id: BR-37
          severity: Important
          title: editDistance is still missing from the Core-concepts table, and runHelp was never in it
          detail: |-
            This is the 4th finding in family plan-artifact-lags-code (BR-5, BR-21, BR-34). Do NOT
            just add the rows. Of BR-34's five named entities, two (commandNeedsDeck,
            command.needsDeck) became correct by DELETION rather than maintenance, two (historyPaths,
            TestNoRuntimeStateInHistory) were added, and editDistance at command.go:124 is still
            absent - grep-verified, zero occurrences in the plan. Cross-checking every top-level
            declaration in command.go, history_cmd.go and sound_cmd.go against both tables surfaces
            a sixth the finding never named - runHelp at command.go:211, the /help command itself,
            which has no row and never had one. Box-ticking is clean, the round-5/6 Revisions entry
            landed, and every entity the tables DO name exists at its stated path.
            THE RULE, unchanged from BR-34 and now four rounds deep - the table is a hand-maintained
            restatement of what the package declares, so it drifts by default; the fix is to make it
            derive, not to remember harder. A guard comparing the table's Name-and-path pairs against
            top-level declarations in the named files would have failed the commit that added
            editDistance and the commit that added runHelp - the same shape as TestNoTrackedRuntimeState.
          family: plan-artifact-lags-code
          round: 8
        - id: BR-38
          severity: Minor
          title: history_store.go's type comment says the log is read at construction; its own Load doc fifteen lines below says it moved out of the constructor
          detail: |-
            This is the 3rd finding in family comment-orphaned-by-insertion (BR-12, BR-30). Do NOT
            patch this instance. history_store.go:21-22 reads "the log is read once at construction
            and everything after is memory", while history_store.go:35-38 - written by the same
            commit - reads "It used to happen in the constructor". A direct contradiction inside one
            file, in the doc comment of the type the change is about.
            THE RULE, already stated for BR-30 and measurably unadopted at prevalence 3 - when a fix
            changes what a block does, every comment describing that block is part of the diff,
            including the TYPE's own doc comment, which is the one nobody re-reads because it sits
            above the declaration rather than above the changed lines. The same two sentences are
            restated at atlas/define.md:284 and :307, which is why this and the docs finding are one
            change to make.
          family: comment-orphaned-by-insertion
          round: 8
        - id: BR-39
          severity: Minor
          title: The specific inputs BR-26 and BR-32 measured are still entered by no fixture; both regressions restore silently
          detail: |-
            This is the 2nd finding in family uncovered-branch-at-boundary (BR-25 first). Do NOT just
            add these rows. Two mutations, each compiled with the full suite green at -count=1:
            (1) inserting a second c.deck.Events(time.Time{}) into runHistory restores BR-32's exact
            doubled-warning symptom - the count assertion that finding asked for was never added;
            (2) moving the Events read above parseHistoryArgs in runHistory restores BR-26's exact
            "define /history zzz reads the whole log before its usage error" symptom - the three rows
            that finding asked for were never added. The behaviour is correct in both cases and the
            shared mechanism (History.Load laziness) IS pinned, so this is about the specific cases,
            not the fix.
            THE RULE - when a finding states the inputs it measured, those inputs ARE the regression
            fixture; a fix is closed by a test that runs them, not by a change that happens to make
            them pass. BR-25 is the same shape at a different site - renderHistory's width branch is
            prose-correct and fixture-free because both call sites pass 0.
          family: uncovered-branch-at-boundary
          round: 8
        - id: BR-40
          severity: Minor
          title: storeHistory.Load mutates lines and loaded without taking the mutex that Add and Prefix both take
          detail: |-
            history_store.go:40. Load appends to h.lines and sets h.loaded outside h.mu, while
            Add (:62) and Prefix (:73) both lock. Unreachable concurrently today - the only production
            caller is runEditor at replraw.go:67, before the key loop starts, and go test -race is
            green - so this is latent, not live. But a struct that locks for some mutations and not
            others invites the next caller to assume the wrong thing, and Load is the newest method
            on it.
          family: partial-lock-discipline
          round: 8
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

## Round 5 — 2026-08-21T17:39:51-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — addressed — Probed: `define /help` runs and exits 0; pinned by TestOneShotCommandGoesThroughRun (deleting the dispatch block reddens it).
- BR-15 — not-addressed — The sh block is fine now, but README.md:43-50 still has no / row and "Enter | define what you typed" still omits command dispatch — the exact structure the finding named.
- BR-16 — addressed — Probed `echo /qqqqqq | define` exits 2; mutation-verified — deleting the cmdCode return reddens TestPipedLoopReturnsTheCommandsExitCode.
- BR-17 — addressed — Mutation-verified: reverting math.Round to truncation reddens three DST rows. Residual exactness gap raised separately as a Minor.
- BR-18 — addressed — Files untracked, TestNoTrackedRuntimeState plant-verified to fail, and a real-pty conformance run leaves git status clean with no deck under cmd/define/. History residue raised separately.
- BR-19 — not-addressed — All four instances are now mutation-verified pinned, but the RULE the finding demanded is absent from workshop/lessons.md and was violated in the same commit — BR-20's fix and clearMenu are both unpinned (measured).
- BR-20 — not-addressed — Behaviour correct when probed, but unpinned: reverting the arity-guard exemption restores the reported usage dump, and classifying fs.Arg(0) makes /history 7 print a 2-day window — full suite green under both.
- BR-21 — addressed — Zero unticked boxes remain, and every entity named in the Core-concepts table exists at its stated path (grep-verified).
- BR-22 — not-addressed — Re-probed: --sound 1000 accepted, /sound 1000 refused at the 20 cap.
- BR-23 — not-addressed — Re-probed: define -times -1 still reports "define: -sound must not be negative".
- BR-24 — not-addressed — command.go:239 still slices bytes, history_cmd.go:178 still slices runes.
- BR-25 — not-addressed — Both renderHistory call sites still pass width 0, and nothing asserts runHistory forwards c.width.

### Raised

- **BR-26** [Important] `guard-bypassed-by-new-kind` A command line skips the "usage errors are settled before a store is opened" invariant and reads the whole event log
  main.go:272 exempts cmdCommand from the arity guard, so a command line reaches
  main.go:279 d.withStore before anything validates it. Measured with the same torn-log
  fixture TestUsageErrorsDoNotOpenTheLog uses - `define /qqqqqq`, `define /qqqqqq zzz`
  and `define /history zzz` all print "recovered 0 event(s), dropped 1 torn record(s)"
  before their usage error. main.go:252 states the contract four lines above the guard
  that breaks it. The directory-creation half still holds (NewYAML is lazy, confirmed),
  but the read half does not, and capture_test.go:423 has rows for -forget and two words
  and none for a command, so the drift is untested in either direction. Fix by resolving
  the command name against `commands` before withStore, and add the three rows.
- **BR-27** [Important] `docs-consumer-not-updated` --days and --days=N ship and are tested but appear in no user-facing doc, while the plan box promising them is ticked
  This is the 2nd finding in family docs-consumer-not-updated (BR-4 first). Do NOT patch
  this instance. parseHistoryArgs accepts 7, --days 7 and --days=7, all pinned by
  TestParseHistoryArgs; grep finds zero occurrences of --days in README.md, atlas/define.md
  and the --help text, and /help's summary is "words looked up recently". Only the
  positional [N] form is documented. Plan line 352 reads "- [x] Step 2: README: what
  /history shows, what it omits and why, --days" - a ticked box asserting the one
  deliverable that did not land.
  THE RULE - a flag or argument form is not shipped until at least one consumer a user can
  reach derives it, and the plan box is ticked by the commit that updates that consumer,
  not by the commit that adds the parser (ARCH-PURPOSE shadow-sweep: parser and tests
  derive, README/atlas/--help//help do not).
- **BR-28** [Important] `runtime-output-tracked-in-source-tree` The deck blobs are still reachable from HEAD, and the guard written for BR-18 checks only the index
  This is the 2nd finding in family runtime-output-tracked-in-source-tree (BR-18 first). Do
  NOT patch this instance. `git show 1ff0d5e:cmd/define/words/sycophantic.yaml` still
  returns the word with its timestamps and lookups: 12; the two events files likewise, added
  by fc071af and 1ff0d5e. 4b019e0 removed them from the index only. TestNoTrackedRuntimeState
  reads `git ls-files` - the index - while its sibling TestNoBinariesInHistory walks history
  precisely because, in its own words, deleting in a later commit does not remove the cost.
  THE RULE - a guard for a committed-artifact class must check the scope where that class's
  cost lives: the index for what is checked out, history for what every clone fetches. A fix
  that only untracks is incomplete against a history-scoped guard. Cheap now while the branch
  is unmerged; permanent after.
- **BR-29** [Minor] `elapsed-hours-for-calendar-days` relativeDay's rounding is a heuristic where an exact computation is one line away, and the comment asserts an exactness the code lacks
  This is the 2nd finding in family elapsed-hours-for-calendar-days (BR-17 first). Do NOT
  patch this instance. history_cmd.go:199's comment claims "The true gap is always N days
  +/- 1 hour, which makes rounding exact"; measured in Pacific/Apia, at=2011-12-29,
  now=2011-12-31 prints "yesterday" for a two-calendar-day gap, because the 2011 date-line
  change deleted 2011-12-30 and only 24 hours elapsed.
  THE RULE - count calendar days on calendar-day numbers, never on elapsed time: normalise
  both dates into a fixed-offset zone before differencing, so no offset arithmetic enters
  the count at all. Verified - building dayOf in time.UTC and dropping math.Round returns
  "Thursday" for Apia, keeps the full suite green including all four DST rows, and removes
  the math import.
- **BR-30** [Minor] `comment-orphaned-by-insertion` replraw.go:67 still claims candidates are resolved once per keystroke, but draw() now computes its own list
  This is the 2nd finding in family comment-orphaned-by-insertion (BR-12 first). Do NOT patch
  this instance. The comment reads "Resolve candidates ONCE per keystroke and use the same
  slice for both the state machine and the suggestion. Querying twice doubled the work the
  History seam will do once #3 backs it with a store." Since 1ff0d5e removed draw's parameter,
  completionsFor runs twice per keystroke - once for Apply at replraw.go:129 and once inside
  draw - and #3 has landed.
  THE RULE - when a fix changes what a block does, the comment above it is part of the diff.
  A comment that survives a behaviour change is a false claim about the code beneath it,
  which is exactly what let BR-17's DST bug read as correct for four rounds.

## Round 6 — 2026-08-21T18:00:13-07:00 (claude) — BLOCKED

### Disposed

- BR-15 — not-addressed — README.md:43-50 still has no / row and "Enter | define what you typed (never the suggestion)" still omits command dispatch.
- BR-19 — addressed — Rule recorded at lessons.md:367; both new instances mutation-verified (arity exemption and clearMenu each redden a loop-driving test).
- BR-20 — addressed — Both halves mutation-verified — reverting the arity exemption and classifying fs.Arg(0) each redden TestOneShotCommandTakesArguments.
- BR-22 — not-addressed — Re-probed against the built binary — "define --sound 1000 /sound" prints "playing 1000x" and exits 0.
- BR-23 — not-addressed — Re-probed — "define -times -1 x" still reports "define: -sound must not be negative".
- BR-24 — not-addressed — command.go:255 still slices bytes, history_cmd.go:177 still slices runes.
- BR-25 — not-addressed — Both renderHistory call sites still pass width 0, and nothing asserts runHistory forwards c.width.
- BR-26 — not-addressed — Two of the three measured cases are fixed and mutation-pinned, but "define /history zzz" still reads the whole log before its usage error, and that is the one row the finding asked for that was not added.
- BR-27 — addressed — --days now derives in three reachable consumers — README.md:102, atlas/define.md:401, and the --help text at main.go:206.
- BR-28 — addressed — git rev-list --objects HEAD returns zero deck objects, the three named commits no longer exist, and TestNoRuntimeStateInHistory covers the history scope.
- BR-29 — not-addressed — history_cmd.go:199 still uses math.Round on elapsed hours, and the exactness comment is unchanged.
- BR-30 — not-addressed — replraw.go:67-69 unchanged; completionsFor still runs twice per keystroke, at :113 and :129.

### Raised

- **BR-31** [Important] `guard-bypassed-by-new-kind` The needsDeck exemption strands deps.clock, so a registry row with needsDeck false gets a nil clock and panics
  This is the 2nd finding in family guard-bypassed-by-new-kind (BR-26 first). Do NOT patch
  this instance. main.go:285 skips withStore for a command that does not need the deck, and
  withStore (main.go:99) is the ONLY thing that ever sets deps.clock - realDeps at main.go:50
  does not. Measured: adding a registry row with needsDeck false whose run calls c.clock.Now()
  makes "define -no-audio /probe" panic with a nil pointer dereference, while the full suite
  stays green. capture_test.go:468 and :494 exist to prevent exactly this and both pass,
  because they exercise openStore and withStore, the path the exemption routes around. The
  issue's Done-when "adding a second command needs no change to the dispatch loop" now carries
  an undocumented precondition that neither the commandCtx doc comment nor atlas/define.md's
  needsDeck paragraph states.
  THE RULE - when a new kind is exempted from a shared setup path, enumerate everything that
  path guaranteed and re-supply it; the exemption drops invariants you were not thinking about,
  not just the cost you were avoiding. Rule-level fix - the clock has no store dependency, so
  supply it outside withStore (in run before the branch, or as a fallback in newCommandCtx) so
  no exemption can strand it, and extend TestEveryRegisteredCommandIsRunnable to assert the ctx
  a one-shot actually builds carries every field a row may read.
- **BR-32** [Important] `same-source-read-twice` /history reads the whole event log twice and prints every store warning twice
  Measured against a torn-log fixture - "define -no-audio /history" and "echo /history |
  define" each emit "recovered 1 event(s), dropped 1 torn record(s)" TWICE before the
  output, while a plain word lookup emits it once and "define /help" not at all. Read 1 is
  newStoreHistory calling st.Events(time.Time{}) at history_store.go:31, built by withStore
  because commandNeedsDeck said yes; read 2 is runHistory calling c.deck.Events(time.Time{})
  at history_cmd.go:228. In the one-shot path the first read is pure waste, and replLines
  never touches d.history at all so it is waste in the piped path too. atlas/define.md:446
  and plan line 374 both record "One read per /history on a personal word list is the right
  trade today" as the accepted cost, so the recorded trade is not the one the code makes.
  Fix by gating on the specific dependency rather than the whole store bundle, or by handing
  runHistory the events storeHistory already read, and assert the warning COUNT in
  TestCommandThatReadsNothingDoesNotOpenTheLog, which today only checks presence.
- **BR-33** [Important] `docs-consumer-not-updated` The atlas never learned about the runtime-state guards, and the README still teaches the deprecated flag name
  This is the 3rd finding in family docs-consumer-not-updated (BR-4, BR-27). Do NOT patch
  these instances. Measured prevalence, both from this window - (1) atlas/repo-guards.md
  documents one guard class with an index/history table naming TestNoCommittedBinaries and
  TestNoBinariesInHistory; this window added a second class with the identical split
  (TestNoTrackedRuntimeState / TestNoRuntimeStateInHistory) and neither that page nor
  atlas/index.md:13-14 ("no executable image in the index or reachable from HEAD") mentions
  it, so the index understates what the file it points at enforces. (2) README.md:84 still
  reads "Flags are session settings - define -times 1 opens the loop with single playback",
  teaching the name README.md:118 calls "the older name for --sound".
  THE RULE - a surface is not shipped until every doc that already describes its class is
  updated in the same commit, where "its class" means the page that is wrong by omission,
  not only the page that names the new thing. BR-4 updated the README because the finding
  named the README and BR-27 updated three places because the finding listed three; neither
  adopted the sweep. Enforcement worth having (ARCH-PURPOSE) - atlas/repo-guards.md's table
  is a hand-maintained restatement of the Test functions in repo_guard_test.go, so a test
  that greps the table for every "func TestNo..." in that file makes the page derive.
- **BR-34** [Important] `plan-artifact-lags-code` The Core-concepts table lost five entities again, and the plan has no Revisions entry for rounds 5 or 6
  This is the 3rd finding in family plan-artifact-lags-code (BR-5, BR-21). Do NOT just add
  the rows. commandNeedsDeck, command.needsDeck, editDistance, historyPaths and
  TestNoRuntimeStateInHistory appear in no row (grep-verified, zero occurrences of each name
  in the plan); four of the five were added by e5719f9, the commit that closed BR-21's own
  family. The plan's last Revisions heading is "close round 4", so rounds 5 and 6 - which
  produced the needsDeck design, the history rewrite and the --days docs - are unrecorded.
  Box-ticking is clean and every entity the table DOES name exists at its stated path.
  THE RULE - the Core-concepts table is a hand-maintained restatement of what the package
  declares, so it drifts by default; the fix is to make it derive, not to remember harder.
  BR-21 was disposed addressed and the very next commit re-broke it, which is the measurement
  that a habit-level fix does not hold. A guard comparing the table's Name-and-path pairs
  against top-level declarations in the named files would fail the same commit that adds an
  unnamed entity - the same shape as TestNoTrackedRuntimeState.
- **BR-35** [Minor] `parallel-construction-drift` Two registry lookups carry independently-written matching rules and nothing pins that they agree
  This is the 2nd finding in family parallel-construction-drift (BR-6 first). Do NOT patch
  this instance. commandNeedsDeck (command.go:192-199) and dispatchCommand (command.go:211-215)
  each walk cmds with strings.EqualFold. Mutating commandNeedsDeck to an exact comparison
  leaves the ENTIRE suite green while "define /HISTORY" reports "define: no deck in this
  directory" and exits 1 - a lie about the user's data - because dispatch still accepts
  loosely and the deck gate no longer does. atlas/define.md states the loose-accept policy as
  a design decision, so the two lookups implementing it must agree by construction.
  THE RULE - two sites that must agree about one fact should be one site: a single
  findCommand(name, cmds) (command, bool) that both call, so the matching rule cannot drift.

## Round 7 — 2026-08-21T18:10:14-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 8 — 2026-08-21T18:24:16-07:00 (claude) — passed

### Disposed

- BR-15 — not-addressed — README.md:42-49 still has no / row and "Enter | define what you typed (never the suggestion)" still omits command dispatch.
- BR-22 — not-addressed — Re-probed against the built binary — "define --sound 1000 /sound" prints "playing 1000x" and exits 0, while "/sound 1000" is refused at 20.
- BR-23 — not-addressed — Re-probed — "define -times -1 x" still reports "define: -sound must not be negative".
- BR-24 — not-addressed — command.go:238-239 still slices bytes, history_cmd.go:177-178 still slices runes, and runHelp's hardcoded %-10s at command.go:211 is now a third name-column width beside menuNameWidth.
- BR-25 — not-addressed — history_cmd_test.go:225 and :268 still pass width 0, and nothing asserts runHistory forwards c.width.
- BR-26 — addressed — Probed with the torn-log fixture — /qqqqqq, /qqqqqq zzz and /history zzz all read nothing now; mutation-verified (reverting Load to eager reddens TestCommandThatReadsNothingDoesNotOpenTheLog). The three rows were still not added; raised separately.
- BR-29 — not-addressed — Reproduced in Pacific/Apia — at=2011-12-29, now=2011-12-31 prints "yesterday" for a two-calendar-day gap. history_cmd.go:199 unchanged, exactness comment unchanged.
- BR-30 — not-addressed — replraw.go:70 unchanged; completionsFor still runs twice per keystroke, at :113 and :129.
- BR-31 — addressed — needsDeck deleted and withStore is unconditional; mutation-verified two ways — removing withStore's clock fallback reddens TestWithStoreCarriesTheClock, and nil-ing clock in newCommandCtx panics the suite through run().
- BR-32 — addressed — Probed — /history warns ONCE now, one-shot and piped; pinned by the Load-laziness mutation. The warning-COUNT assertion the finding asked for is still absent; raised separately.
- BR-33 — not-addressed — repo-guards.md gained the section and README.md:84 now teaches --sound, but atlas/index.md:13-14 — named explicitly in the finding — still mentions only the executable-image class, and the same commit created three fresh stale claims in atlas/define.md.
- BR-34 — not-addressed — commandNeedsDeck and needsDeck became correct by deletion rather than maintenance; historyPaths and TestNoRuntimeStateInHistory landed; editDistance is still absent (grep-verified), and the derive-guard the rule called for was not written.
- BR-35 — addressed — commandNeedsDeck is deleted, so exactly one strings.EqualFold walk over cmds remains (command.go:195) — the two rules cannot drift by construction.

### Raised

- **BR-36** [Important] `docs-consumer-not-updated` atlas/define.md documents needsDeck, a field the same commit deleted, and two more paragraphs describe a load that no longer happens at construction
  This is the 4th finding in family docs-consumer-not-updated (BR-4, BR-27, BR-33). Do NOT
  patch these instances. Measured prevalence, all four from 8f04434 - the commit whose
  message closes BR-33: (1) atlas/define.md:404-405 "A command declares needsDeck when it
  reads the store. Opening the store constructs storeHistory, which READS the whole event
  log" - grep needsDeck over cmd/ returns zero hits and opening the store reads nothing;
  (2) atlas/define.md:307 "loads the log once at construction"; (3) atlas/define.md:284
  "since #4 it only reads at construction" - both falsified by History.Load; (4)
  atlas/index.md:13-14 still names only the executable-image class, the exact instance
  BR-33 listed and the one half of it that was not fixed.
  THE RULE, stated by BR-33 and still unadopted - a surface is not shipped until every doc
  that already describes its class is updated in the same commit, where "its class" means
  the page that is wrong by omission or contradiction, not only the page that names the new
  thing. Three rounds of fixing exactly the pages a finding lists is the measurement that
  the sweep is not happening. Cheap durable half - when a commit DELETES an identifier,
  grep atlas/ and README.md for it before committing; enforcement worth having
  (ARCH-PURPOSE) is the repo-guards table guard BR-33 already described.
- **BR-37** [Important] `plan-artifact-lags-code` editDistance is still missing from the Core-concepts table, and runHelp was never in it
  This is the 4th finding in family plan-artifact-lags-code (BR-5, BR-21, BR-34). Do NOT
  just add the rows. Of BR-34's five named entities, two (commandNeedsDeck,
  command.needsDeck) became correct by DELETION rather than maintenance, two (historyPaths,
  TestNoRuntimeStateInHistory) were added, and editDistance at command.go:124 is still
  absent - grep-verified, zero occurrences in the plan. Cross-checking every top-level
  declaration in command.go, history_cmd.go and sound_cmd.go against both tables surfaces
  a sixth the finding never named - runHelp at command.go:211, the /help command itself,
  which has no row and never had one. Box-ticking is clean, the round-5/6 Revisions entry
  landed, and every entity the tables DO name exists at its stated path.
  THE RULE, unchanged from BR-34 and now four rounds deep - the table is a hand-maintained
  restatement of what the package declares, so it drifts by default; the fix is to make it
  derive, not to remember harder. A guard comparing the table's Name-and-path pairs against
  top-level declarations in the named files would have failed the commit that added
  editDistance and the commit that added runHelp - the same shape as TestNoTrackedRuntimeState.
- **BR-38** [Minor] `comment-orphaned-by-insertion` history_store.go's type comment says the log is read at construction; its own Load doc fifteen lines below says it moved out of the constructor
  This is the 3rd finding in family comment-orphaned-by-insertion (BR-12, BR-30). Do NOT
  patch this instance. history_store.go:21-22 reads "the log is read once at construction
  and everything after is memory", while history_store.go:35-38 - written by the same
  commit - reads "It used to happen in the constructor". A direct contradiction inside one
  file, in the doc comment of the type the change is about.
  THE RULE, already stated for BR-30 and measurably unadopted at prevalence 3 - when a fix
  changes what a block does, every comment describing that block is part of the diff,
  including the TYPE's own doc comment, which is the one nobody re-reads because it sits
  above the declaration rather than above the changed lines. The same two sentences are
  restated at atlas/define.md:284 and :307, which is why this and the docs finding are one
  change to make.
- **BR-39** [Minor] `uncovered-branch-at-boundary` The specific inputs BR-26 and BR-32 measured are still entered by no fixture; both regressions restore silently
  This is the 2nd finding in family uncovered-branch-at-boundary (BR-25 first). Do NOT just
  add these rows. Two mutations, each compiled with the full suite green at -count=1:
  (1) inserting a second c.deck.Events(time.Time{}) into runHistory restores BR-32's exact
  doubled-warning symptom - the count assertion that finding asked for was never added;
  (2) moving the Events read above parseHistoryArgs in runHistory restores BR-26's exact
  "define /history zzz reads the whole log before its usage error" symptom - the three rows
  that finding asked for were never added. The behaviour is correct in both cases and the
  shared mechanism (History.Load laziness) IS pinned, so this is about the specific cases,
  not the fix.
  THE RULE - when a finding states the inputs it measured, those inputs ARE the regression
  fixture; a fix is closed by a test that runs them, not by a change that happens to make
  them pass. BR-25 is the same shape at a different site - renderHistory's width branch is
  prose-correct and fixture-free because both call sites pass 0.
- **BR-40** [Minor] `partial-lock-discipline` storeHistory.Load mutates lines and loaded without taking the mutex that Add and Prefix both take
  history_store.go:40. Load appends to h.lines and sets h.loaded outside h.mu, while
  Add (:62) and Prefix (:73) both lock. Unreachable concurrently today - the only production
  caller is runEditor at replraw.go:67, before the key loop starts, and go test -race is
  green - so this is latent, not live. But a struct that locks for some mutations and not
  others invites the next caller to assume the wrong thing, and Load is the newest method
  on it.

## Open findings

- **BR-15** [Minor] `fix-appended-not-integrated` The BR-4 README fix was appended to the sh block instead of integrated into the key table
- **BR-22** [Minor] `same-setting-two-validation-policies` --sound 1000 is accepted while /sound 1000 is refused at the 20 cap
- **BR-23** [Minor] `error-names-a-flag-the-user-did-not-type` define -times -1 reports "define: -sound must not be negative"
- **BR-24** [Minor] `duplicated-width-arithmetic` menuLines truncates in bytes while renderHistory truncates in runes, and renderHistory measures columns in bytes but pads in runes
- **BR-25** [Minor] `uncovered-branch-at-boundary` renderHistory's truncation branch is never exercised — both test call sites pass width 0
- **BR-29** [Minor] `elapsed-hours-for-calendar-days` relativeDay's rounding is a heuristic where an exact computation is one line away, and the comment asserts an exactness the code lacks
- **BR-30** [Minor] `comment-orphaned-by-insertion` replraw.go:67 still claims candidates are resolved once per keystroke, but draw() now computes its own list
- **BR-33** [Important] `docs-consumer-not-updated` The atlas never learned about the runtime-state guards, and the README still teaches the deprecated flag name
- **BR-34** [Important] `plan-artifact-lags-code` The Core-concepts table lost five entities again, and the plan has no Revisions entry for rounds 5 or 6
- **BR-36** [Important] `docs-consumer-not-updated` atlas/define.md documents needsDeck, a field the same commit deleted, and two more paragraphs describe a load that no longer happens at construction
- **BR-37** [Important] `plan-artifact-lags-code` editDistance is still missing from the Core-concepts table, and runHelp was never in it
- **BR-38** [Minor] `comment-orphaned-by-insertion` history_store.go's type comment says the log is read at construction; its own Load doc fifteen lines below says it moved out of the constructor
- **BR-39** [Minor] `uncovered-branch-at-boundary` The specific inputs BR-26 and BR-32 measured are still entered by no fixture; both regressions restore silently
- **BR-40** [Minor] `partial-lock-discipline` storeHistory.Load mutates lines and loaded without taking the mutex that Add and Prefix both take
