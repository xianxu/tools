---
gate: boundary-review
issue: 48
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-08T16:52:21-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: the two entry points would print different sentences for the same nil deck
          detail: |-
            This is the 2nd finding in family refusal-enumeration-incomplete. Per the
            escalation rule I am not asking you to fix this instance — state the rule.
            The rule: one refusal condition gets one sentence from one helper. The
            enumeration is measurable and small — six sites call noDeckMessage
            (main.go:1193, stats.go:43, stats.go:237, history_cmd.go:220, harvest.go:111,
            reflect.go:340) and exactly one hand-rolls it, runPlay at play_loop.go:30
            ("no deck in this directory, so there is nothing to review"). The plan has
            /play use noDeckMessage, so after this issue the same condition reads two
            ways depending on which entry point you took — against Done-when's "a change
            to the sitting cannot apply to only one of them". Either sweep play_loop.go:30
            into the helper in the same round, or state that noDeckMessage owns the cause
            and a caller may append its own consequence, and make both entry points obey it.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: refusal-enumeration-incomplete
          round: 1
        - id: BR-2
          severity: Minor
          title: Step 8 is the bare item "README + atlas" and names no section for the hand-swept half
          detail: |-
            This is the 2nd finding in family hand-swept-surface-unnamed, so the rule rather
            than the instance: a prose surface no test derives is named - file and section - in
            the step that sweeps it, which is this repo's own stated convention at
            doc_sync_test.go:376-380. Here that is the per-command paragraphs at
            cmd/define/README.md:724-760, where /stats, /history and /pron each have one.
            (carried from plan-quality PQ-9, deferred to the boundary review)
          family: hand-swept-surface-unnamed
          round: 1
        - id: BR-3
          severity: Minor
          title: the sittingInPlace signature still takes sess/stdout, the pair the stated rule says it must not build a console from
          detail: |-
            This is the 4th finding in family terminal-ownership-unstated, so the rule not the row — and the plan now STATES the rule correctly (newSitting owns assembly, built where the rawSession and the real stdout are in scope). The signature is the one artifact left contradicting it: replayInPlace's sess is the plain session struct (replraw.go:625, session.go:9) and its stdout at the call site is con.stdout, the liveScreen (replraw.go:507), so an implementer building the pinned screen from those arguments reproduces PQ-2's fabricated 80x24 via defaultCols. Make the signature take the console.
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: terminal-ownership-unstated
          round: 1
        - id: BR-4
          severity: Minor
          title: the plan's Verification section backticks a test that does not exist, so TestPlanCitesTestsThatExist is red on main
          detail: |-
            go test -run TestPlanCitesTestsThatExist ./... fails today: repo_guard_test.go:1353 reports that the plan cites TestSlashPlayRefusesWhereItCannotRun and no such test exists. The plan cites that guard itself (repo_guard_test.go:1306) and then trips it, and its own Verification step 1 demands a clean suite. Un-backtick the not-yet-written names, or note the expected red until Task 1 Step 1 lands.
            (carried from plan-quality PQ-11, deferred to the boundary review)
          family: plan-cites-unwritten-test
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-08T16:52:21-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Critical
          title: a resize consumed during a sitting is never handed back, so the REPL screen paints at a stale shape for the rest of the session
          detail: |-
            sittingInPlace borrows the loop's resizes channel (play_cmd.go:71, replraw.go:103) and playSession
            applies each shape to the sitting's screen only. Nothing writes it back to repl, and resume()
            (screen.go:901) repaints with rows/cols untouched; watchResize only emits on the NEXT SIGWINCH, so the
            editor is stuck at the pre-sitting geometry - too many rows scrolls the terminal, wrong cols mis-wraps
            and puts click coordinates off. screen.go:899 asserts the opposite in a doc comment. Reproduced: a
            scratch test that pre-loads resizes with 40x120, runs sittingInPlace to a scripted interrupt, and reads
            repl.Size() gets 24x80. Fix: capture sitting.Size() and repl.Resize(r, c) before repl.resume(), and
            correct the comment.
          family: borrowed-channel-swallows-owners-events
          round: 2
        - id: BR-6
          severity: Important
          title: all three of runPlayCommand's refusals are unpinned - the plan's Chunk 1 tests were never written
          detail: |-
            Mutation-proven: replacing runPlayCommand's body (play_cmd.go:23-49) with a bare nil check plus
            c.startSitting() - dropping the argument refusal, the no-terminal sentence, the no-deck sentence and
            both non-zero exit codes - leaves the whole ./cmd/define suite green. No test file names runPlayCommand
            or sets startSitting. Chunk 1 Task 1 Step 1 named exactly these three tests and the step is ticked.
          family: plan-named-test-not-written
          round: 2
        - id: BR-7
          severity: Important
          title: the scoped interrupt has no test - removing Set/restore reddens nothing
          detail: |-
            In the same mutation run, deleting restore := interrupts.Set(cancel) and defer restore()
            (play_cmd.go:97-98) reddened nothing. Without them Ctrl-C during a sitting fires the loop's cancel and
            kills define, which is the regression Done-when 2 exists to prevent. pty_conformance_test.go is
            untouched in this window; sittingInPlace and the runEditor dispatch (replraw.go:521-531) have zero
            coverage. Plan Task 2 Steps 2, 4 and 7 and Verification item 4 all claim this.
          family: plan-named-test-not-written
          round: 2
        - id: BR-8
          severity: Important
          title: TestASittingRecordsTheSameEventFromEitherDoor never invokes either door
          detail: |-
            play_cmd_test.go:145-183 calls playSession directly twice with identical inputs and compares the two
            results - a determinism check, not a comparison of entry points. runPlay, sittingInPlace and
            runPlayCommand appear nowhere in it, so the Done-when it is cited against rests entirely on the AST
            guard at play_cmd_test.go:101-142. Rename it to what it pins, or parameterise it over the two doors.
          family: test-name-overclaims-what-it-pins
          round: 2
        - id: BR-9
          severity: Important
          title: atlas update appears missing for console.newSitting, liveScreen.suspend/resume and sittingInPlace
          detail: |-
            Only the /play command-table row was added (atlas/define.md:1119). The type table at atlas/define.md:273
            still lists console as display + resizes + finish + stdout + stderr; suspend/resume gets no entry beside
            handBack/Stop; and the "newConsole is the shared builder both loops now call" claim at
            atlas/define.md:2562 and 2579 now has a deliberate hand-assembled exception whose rationale lives only
            in a function comment.
          family: atlas-stale-for-new-surface
          round: 2
        - id: BR-10
          severity: Minor
          title: the plan's entity tables have no suspend/resume row and a stale sittingInPlace signature
          detail: |-
            PQ-5's revision claims suspend/resume now has "an entity row"; neither the Pure entities nor the
            Integration points table contains one. The sittingInPlace row still reads (ctx, d, opt, sess, keys,
            interrupts, stdout, stderr); the shipped function takes repl, resizes and tty and no sess.
            TestPlanTablesNameEntitiesThatExist passes because it only checks one direction.
          family: plan-table-stale
          round: 2
        - id: BR-11
          severity: Minor
          title: newSitting is installed on every console newConsole builds, including --play's and the board's
          detail: |-
            replraw.go:100-104 sets con.newSitting unconditionally, so --play's pinned console and #40's board carry
            a non-nil sitting factory where a nested sitting cannot run - contradicting the field's own doc at
            replraw.go:165-168 ("nil where one cannot run"). Unreachable today; only runEditor reads it.
          family: capability-wider-than-its-doc
          round: 2
        - id: BR-12
          severity: Minor
          title: --play and /play give different sentences, streams and exit codes for a deckless directory
          detail: |-
            play_loop.go:29-31 prints "no deck in this directory, so there is nothing to review" to stdout and
            returns 0; play_cmd.go:44-47 prints noDeckMessage to stderr and returns 1. Two doors, two answers to the
            same question.
          family: refusal-wording-diverges
          round: 2
        - id: BR-13
          severity: Minor
          title: play_cmd_test.go re-implements reviewEvents and duplicates the AST-parse preamble
          detail: |-
            The inline closure at play_cmd_test.go:172-183 is reviewEvents (play_loop_test.go:81) in the same
            package, and the ParseDir + FuncDecl walk is copied between play_cmd_test.go:23-58 and :104-119
            (ARCH-DRY).
          family: test-helper-duplicated
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-08T18:33:10-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: play_loop.go:30 still hand-rolls the sentence; verified by running both doors — --play prints "no deck in this directory, so there is nothing to review" to stdout and exits 0, /play prints noDeckMessage to stderr and exits 1. No rule stated anywhere.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: README.md:724-730 did get its paragraph, but the plan's Step 8 is still the bare "README + atlas" — the rule the finding asked for was never stated.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: The plan is unmodified in this window; the Integration-points row still reads (ctx, d, opt, sess, keys, interrupts, stdout, stderr). The code got it right, so the predicted defect did not ship — only the artifact contradicts it.
          round: 3
        - id: BR-4
          disposition: addressed
          note: 'Fixed in the window''s base commit adcbcf1; TestPlanCitesTestsThatExist is green. The inverse is now true and folded into the new plan-table-stale finding: the plan cites no shipped test name at all.'
          round: 3
        - id: BR-5
          disposition: addressed
          note: 'Mutation-verified: deleting play_cmd.go:103 reddens TestTheEditorScreenTakesTheShapeTheSittingEndedWith with its own sentence. Only the screen shape is handed back, though — see the new opt.width finding.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: 'Mutation-verified: replacing runPlayCommand''s body with a bare nil check plus c.startSitting() reddens three of TestSlashPlayRefusals'' four subtests, on sentence and exit code.'
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: 'Mutation re-run: with BOTH restore := interrupts.Set(cancel) and defer restore() deleted, TestASittingHandsTheInterruptBack still PASSES. keysFor("^") puts Key{KeyInterrupt} straight on the channel, bypassing the interrupter — the flaw the test''s own comment attributes to version one — and the test installs the loop''s cancel itself, so Fire()''s consumed is true either way. Only the restore half is pinned (dropping defer restore() alone does redden). The pty test the plan names at Task 2 Step 2 and Verification item 4 still does not exist; pty_conformance_test.go has no /play case.'
          round: 3
        - id: BR-8
          disposition: addressed
          note: Renamed to TestTheSittingDoorRecordsWhatItAnswers and now drives sittingInPlace over a buffer with a scripted key channel, asserting through store.Events rather than a fake.
          round: 3
        - id: BR-9
          disposition: addressed
          note: 'atlas/define.md:2605-2626 now names sittingInPlace, the three acquisitions a borrower must not take, suspend/resume, the shape hand-back and the scoped interrupt. Residual: the console block at atlas/define.md:268-274 still enumerates five fields and omits newSitting.'
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: 'Plan unmodified. Root cause now measured: repo_guard_test.go:797 skips status "new" rows while any "- [ ] " remains, and the plan is 0/15 ticked — so both new rows are exempt rather than passing. Rolled into the new plan-table-stale finding.'
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: replraw.go:101-104 still sets con.newSitting unconditionally, so --play's console (play_loop.go:106) carries a factory the field's own doc at replraw.go:166-169 says is nil where a sitting cannot run.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: 'Confirmed by running the built binary in a deckless directory: --play says "no deck in this directory, so there is nothing to review" on stdout, exit 0; /play says noDeckMessage on stderr, exit 1, and names DEFINE_NO_CAPTURE where --play does not.'
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: 'Both duplications survive: play_cmd_test.go:169-181 re-implements reviewEvents (play_loop_test.go:81), and the ParseDir + FuncDecl preamble is still copied between play_cmd_test.go:31-63 and :106-131.'
          round: 3
      findings:
        - id: BR-14
          severity: Important
          title: the borrowed resize hands back the screen shape but not opt.width, so entries looked up after a sitting wrap at the pre-sitting width
          detail: |-
            This is the 2nd finding in family borrowed-channel-swallows-owners-events, so the rule
            rather than the instance. The rule: a borrower that consumes an owner's event stream owes
            back EVERY effect the owner's handler applies - enumerate that handler and mirror it, or
            hand the raw event back instead of one derived value. The enumeration here is exact and
            small. The owner's handler (replraw.go:404-430) applies two pieces of state: the wrap
            policy (opt.width = sz.cols, clamped to 0 below minWrapWidth, lines 424-427) and the
            screen shape (view.Resize, line 428). BR-5's fix hands back the second only, and cannot
            hand back the first - opt reaches sittingInPlace by value through con.newSitting
            (replraw.go:532) and again by value into playSession. So a SIGWINCH consumed by a sitting
            leaves lookupAndRender (replraw.go:685) and newCommandCtx (replraw.go:495) using the
            pre-sitting width for the rest of the session; on a narrowed terminal Paint's clipVisible
            then cuts the over-wide lines at the right edge. Fix the rule: factor lines 424-428 into
            one applyResize(sz, &opt, view) that both the owner's case and the sitting's hand-back
            call, so a third effect added later is handed back by construction.
          family: borrowed-channel-swallows-owners-events
          round: 3
        - id: BR-15
          severity: Important
          title: the durable plan is 0/15 ticked at close, which exempts its two new-entity rows from the guard that would have checked them
          detail: |-
            This is the 2nd finding in family plan-table-stale, so the rule rather than the rows. The
            rule: at a close boundary the durable plan is reconciled with what shipped in ONE sweep -
            checkboxes, entity rows, signatures, test names - because the guards that read plans key
            off "is this plan still in progress". Measured: repo_guard_test.go:797 does
            `if inProgress && status == "new" { continue }`, and inProgress is
            `strings.Contains(body, "- [ ] ")`. The plan has 15 unticked steps and 0 ticked, so BOTH
            new rows (runPlayCommand, sittingInPlace) are skipped outright - that is how BR-3's and
            BR-10's stale rows survived a green suite. Same sweep, same rule: the sittingInPlace row's
            signature still names sess and stdout against a shipped (ctx, d, opt, keys, interrupts,
            repl, resizes, tty, stderr); there is still no suspend/resume entity row despite PQ-5's
            revision claiming one; and the Verification section now backticks no test name at all,
            de-backticked in adcbcf1 "until the tests land" - they have landed, and
            TestPlanCitesTestsThatExist's own message says "write it, or cite the one that shipped".
            Also record what did not ship: the pty test at Task 2 Step 2, and Step 1's shared-half
            extraction (play_loop.go is unmodified).
          family: plan-table-stale
          round: 3
        - id: BR-16
          severity: Minor
          title: console.newSitting takes a parameter it already captures, and is named like a constructor while running a whole sitting
          detail: |-
            replraw.go:101-104 builds the closure over `live`, and runEditor passes stderr := con.stderr
            (replraw.go:532), which IS `live` - so the sixth parameter is always the value already
            captured and can be dropped. Separately, the field is `func(...) int` that performs a
            sitting and returns its exit code, while the plan specified `newSitting func() console`, a
            constructor; a `new*` name on a verb is the one thing a reader of the dispatch at
            replraw.go:532 cannot guess. Rename to runSitting and drop the redundant writer. This is a
            newly-introduced internal seam that downstream work will consume, so the surface is worth
            settling now.
          family: new-seam-surface-unshaped
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-08T18:56:35-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: play_loop.go is unmodified in this window and no artifact states the rule; runPlay:30 still hand-rolls the sentence.
          round: 4
        - id: BR-2
          disposition: addressed
          note: The named surface shipped at cmd/define/README.md:724-730; Step 8's wording is folded into the plan-revision list.
          round: 4
        - id: BR-3
          disposition: addressed
          note: The shipped signature takes repl *liveScreen and the real tty, not sess/stdout; the stale plan ROW is BR-10/BR-15's.
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: TestSlashPlayRefusals was written in 15ecac4 and DELETED in c2e593a; removing the nil-capability refusal now reddens nothing.
          round: 4
        - id: BR-7
          disposition: addressed
          note: Verified by reverting Set/restore in a scratch copy - the test fails at "the sitting did not end when its own interrupt fired".
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: Still no suspend/resume entity row, and line 144's sittingInPlace signature is unchanged.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: replraw.go:123 still sets con.newSitting unconditionally, so --play's console and the board carry it.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: play_loop.go is unmodified; the two doors still differ in sentence, stream and exit code.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: reviewEvents is still re-implemented at play_cmd_test.go:172-178 and ParseDir is still copied at :31 and :107.
          round: 4
        - id: BR-14
          disposition: addressed
          note: Verified by bypassing applyShape on the hand-back route - both clauses of TestBothShapeRoutesGoThroughOnePlace fail.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: Boxes ticked but two of them falsely (Task 2 Steps 1 and 2); signature row, suspend/resume row, Verification backticks and the did-not-ship record are all still missing.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: Still named newSitting for a verb, and still takes the stderr it already captures as live.
          round: 4
      findings:
        - id: BR-17
          severity: Important
          title: the /play dispatch in runEditor is unpinned - making the command a no-op leaves the suite green
          detail: |-
            This is the 3rd finding in family plan-named-test-not-written, so the rule rather
            than the instance. Measured: replacing cc.startSitting = func() { sitting = true }
            (replraw.go:540) with a no-op leaves every non-git-dependent test passing, so /play
            typed at the prompt can silently do nothing. The rule: a plan step is not ticked
            until the test it names exists AND a named mutation of the code it covers reddens
            it. The enumeration is exact - Verification lists 6 items plus 3 Test-surface rows;
            delivered and mutation-checked are items 3, 5 (one door), 6 and the shape rows;
            unmet are item 2 (the refusals, deleted in c2e593a - see BR-6) and item 4 (the pty
            row, pty_conformance_test.go untouched in this window). Two obligations unmet while
            15/15 boxes read done. Adopt the rule at the tick and record the two unmet
            obligations in the plan's Revisions instead of ticking over them.
          family: plan-named-test-not-written
          round: 4
        - id: BR-18
          severity: Important
          title: lessons.md states a Go semantics rule that is false, and records a superseded approach as what worked
          detail: |-
            workshop/lessons.md:4069 says "return code, shape evaluates shape before the defer
            runs, so it hands back the pre-sitting size". With NAMED results - which
            sittingInPlace has - that is not so; I ran it, and the deferred mutation still wins
            for both a bare return and return code, shape. The real requirement is that the
            results be named, which they are. The same wrong claim is in the code comment at
            play_cmd.go:75-78. Separately, lines 4056-4060 record the round-1 approach ("narrow
            the claim to the RESTORE... the end-to-end path is covered by the pty run") that
            BR-7 rejected and c2e593a replaced, and there is no pty run for /play. AGENTS.md
            section 4 makes lessons.md the rule-store, so a wrong rule there misdirects future
            work; rewrite both entries to what actually held.
          family: unverified-mechanism-claim
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-09-08T19:32:27-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: play_loop.go untouched in this window; runPlay:29-31 still hand-rolls the sentence to stdout with exit 0, and noDeckMessage's doc (main.go:1209) is unchanged. Neither the sweep nor the rule-statement happened.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: 'Mutation-measured on a scratch checkout of ea74e04: gutting all three refusals leaves the ./cmd/define failure set byte-identical to baseline. No test names runPlayCommand. Worse, c2e593a TICKED Chunk 1 Step 1 in the same commit whose Revisions entry records this finding.'
          round: 5
        - id: BR-10
          disposition: addressed
          note: Integration points now carries the liveScreen.suspend/resume row, and sittingInPlace's signature matches the shipped one exactly.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: replraw.go:123 still assigns unconditionally; play_loop.go:106 builds --play's console through newConsole, so it carries a non-nil factory the field doc at replraw.go:187-191 says must be nil.
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: Unchanged - stdout/exit 0 versus stderr/exit 1 for the same deckless directory.
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: play_cmd_test.go:170-179 still re-implements reviewEvents (play_loop_test.go:81); the ParseDir preamble is still copied between play_cmd_test.go:32 and :108.
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: Half swept - entity rows, signature, Step 1's DID-NOT-SHIP note, Step 2's rewrite. Not swept, and one moved backwards - Chunk 1 Step 1 was ticked in c2e593a with none of its three tests written; Verification item 4 still claims a pty test Step 2 says was not written; Verification still backticks no test name though ten landed (TestPlanCitesTestsThatExist SKIPS here, no Done-when heading); "The three things that must be built" 2 still specifies newSitting func() console.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: Field still named newSitting while returning (int, winSize); stderr param is still always con.stderr == live, the value the closure already captures.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: The INSTANCE is fixed and I verified it - no-opping cc.startSitting at replraw.go:540 reddens TestTypingSlashPlayRunsASitting and TestTheLoopAppliesTheShapeASittingHandsBack. The escalated RULE was not adopted - the same window ticked Chunk 1 Step 1 with none of its named tests written, and a third site of the same class (newConsole's closure) is unpinned.
          round: 5
        - id: BR-18
          disposition: addressed
          note: lessons.md now states the correct rule and records the earlier error; the round-1 narrowing is recorded as also wrong; play_cmd.go:75-81 corrected. I re-verified the Go semantics independently - named results, defer wins for both bare and explicit return.
          round: 5
      findings:
        - id: BR-19
          severity: Important
          title: newConsole's newSitting closure can stop calling sittingInPlace entirely and the whole suite stays green
          detail: |-
            4th in family - so the rule, not the instance. A capability delivered
            through an injected field is pinned at BOTH ends: the consumer, and the
            production assembly that supplies the real implementation. A test that
            installs its own double for field F proves the consumer and nothing about
            the producer. Enumeration is exact - this issue added three wiring sites,
            all mutation-measured against ea74e04 in a scratch checkout: the loop
            dispatch (replraw.go:540) reddens two tests; runPlayCommand's refusals and
            newConsole's closure (replraw.go:123-126, body replaced with
            "return 0, winSize{}") each leave the failure set byte-identical to
            baseline. Two of three unpinned. TestTypingSlashPlayRunsASitting sets
            con.newSitting itself so it can never see the producer; the two AST guards
            walk sittingInPlace by name whether or not anything reaches it. Cheap
            enforcement already exists in this file - add to
            TestBothEntryPointsReachOnePlaySession that newConsole's body reaches
            sittingInPlace, derived from the source.
          family: plan-named-test-not-written
          round: 5
        - id: BR-20
          severity: Minor
          title: the one-place-for-a-shape guard parses only replraw.go while a second bare Resize lives in play_cmd.go
          detail: |-
            TestBothShapeRoutesGoThroughOnePlace (play_cmd_test.go:319-321) calls
            parser.ParseFile on "replraw.go" alone, and its message says "a shape
            applied outside applyShape is a shape whose width policy was forgotten" -
            but play_cmd.go:116 does exactly that, repl.Resize(r, c) with no opt.width,
            one file over. Benign today because runEditor calls applyShape immediately
            after, so the class the guard names is enforced only where it happens to
            look.
          family: guard-scope-narrower-than-claim
          round: 5
      blocked: false
    - "n": 6
      timestamp: "2026-09-08T21:39:28-07:00"
      agent: claude
      dispose:
        - id: BR-17
          disposition: addressed
          note: 'Mutation-verified: no-op''ing cc.startSitting at replraw.go:540 reddens TestTypingSlashPlayRunsASitting and TestTheLoopAppliesTheShapeASittingHandsBack.'
          round: 6
        - id: BR-19
          disposition: addressed
          note: 'Mutation-verified: replacing newConsole''s newSitting closure body with `return 0, winSize{}` reddens TestTheProducedSittingCapabilityCallsTheRealThing.'
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: 'Re-measured at 116d0f3: gutting runPlayCommand to a bare nil check plus c.startSitting() leaves the whole ./cmd/define suite green. No test names runPlayCommand.'
          round: 6
        - id: BR-15
          disposition: not-addressed
          note: Entity rows and the DID-NOT-SHIP record are fixed; the plan still carries `newSitting func() console`, two pty claims contradicting Task 2 Step 2, no Forget row, no revision for rounds 3-6, and no backticked test names. I confirmed by execution that a bogus `new` row still passes the guard while the plan is inProgress.
          round: 6
        - id: BR-1
          disposition: not-addressed
          note: play_loop.go:30 still hand-rolls the sentence; no rule stated in code, plan or lessons.md.
          round: 6
        - id: BR-12
          disposition: not-addressed
          note: Unchanged - play_loop.go:30 stdout/exit-0 versus play_cmd.go:44-45 stderr/exit-1.
          round: 6
        - id: BR-11
          disposition: not-addressed
          note: replraw.go:123 still assigns con.newSitting unconditionally; the field doc at replraw.go:189-193 is unchanged since 2dc2100.
          round: 6
        - id: BR-13
          disposition: not-addressed
          note: play_cmd_test.go:171-180 still re-implements reviewEvents; the AST preamble is now duplicated four ways (:32, :108, :321, :458).
          round: 6
        - id: BR-16
          disposition: not-addressed
          note: Signature and name unchanged; runEditor still passes con.stderr, which is the `live` screen the closure already captures.
          round: 6
        - id: BR-20
          disposition: not-addressed
          note: Guard still parses replraw.go alone; play_cmd.go:116 still calls repl.Resize directly.
          round: 6
      findings:
        - id: BR-21
          severity: Important
          title: memVocabulary.Forget recounts maxWords without Add's phraseRunsJoin filter, so a drop can widen the phrase window past what Add allows
          detail: |-
            vocab.go:101-106 recomputes maxWords as max(len(wordRuns(w))) over every key, while Add
            (vocab.go:129-131) raises it only when phraseRunsJoin(key, runs) - because a key holding
            other punctuation is permanently unmatchable and counting it makes every stream hold a
            wider window for a match that cannot happen, a cost Add's comment records as measured.
            Executed at 116d0f3: Add("keel"); Add("e.g."); Add("junk") gives MaxPhraseWords()==1, and
            Forget("junk") raises it to 2. Any /play drop on a deck holding one punctuated entry
            widens the streaming renderer's lookahead for the rest of the session. No wrong highlight
            results, so the blast radius is latency (ARCH-CONSTRAINTS), but the invariant is broken
            and the two maintainers of one derived aggregate should be one helper (ARCH-DRY). The
            existing test uses `keel` and `hot dog` and cannot see it - written from the fix's own
            model.
          family: inverse-op-diverges-from-its-pair
          round: 6
        - id: BR-22
          severity: Important
          title: suspend/resume and the sitting's finish are unpinned at the call site, and BR-19's hand-written enumeration of wiring sites was wrong
          detail: |-
            This is the 5th finding in family plan-named-test-not-written. Not asking for these
            instances to be patched - the rule is the deliverable. Measured at 116d0f3, full
            ./cmd/define suite per mutation: deleting repl.suspend() (play_cmd.go:97) and
            repl.resume() (play_cmd.go:117) leaves it GREEN; replacing the sitting console's finish
            body (play_cmd.go:137-140) with a no-op leaves it GREEN; gutting runPlayCommand leaves it
            GREEN (BR-6). So the issue's headline capability - the editor's throttled painter going
            quiet so it cannot land inside the sitting's frame - and the README's promised
            summary-in-the-scrollback are both unpinned. BR-19 stated the both-ends rule and then
            HAND-LISTED the sites ("the enumeration is exact - three wiring sites"); it missed three.
            The rule: an enumeration of sites obliged to obey a capability must be DERIVED from the
            source, not listed from memory - the discipline TestASittingFromTheLoopNeverEntersRawMode
            already applies to callees and TestBothShapeRoutesGoThroughOnePlace to routes. One AST
            guard over sittingInPlace's body (calls suspend, calls resume, writes Transcript() into
            repl), shaped like the drop-arm guard at play_cmd_test.go:557-616, covers the class.
          family: plan-named-test-not-written
          round: 6
      blocked: false
---

# Gate ledger — tools#48 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-08T16:52:21-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `refusal-enumeration-incomplete` the two entry points would print different sentences for the same nil deck
  This is the 2nd finding in family refusal-enumeration-incomplete. Per the
  escalation rule I am not asking you to fix this instance — state the rule.
  The rule: one refusal condition gets one sentence from one helper. The
  enumeration is measurable and small — six sites call noDeckMessage
  (main.go:1193, stats.go:43, stats.go:237, history_cmd.go:220, harvest.go:111,
  reflect.go:340) and exactly one hand-rolls it, runPlay at play_loop.go:30
  ("no deck in this directory, so there is nothing to review"). The plan has
  /play use noDeckMessage, so after this issue the same condition reads two
  ways depending on which entry point you took — against Done-when's "a change
  to the sitting cannot apply to only one of them". Either sweep play_loop.go:30
  into the helper in the same round, or state that noDeckMessage owns the cause
  and a caller may append its own consequence, and make both entry points obey it.
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-2** [Minor] `hand-swept-surface-unnamed` Step 8 is the bare item "README + atlas" and names no section for the hand-swept half
  This is the 2nd finding in family hand-swept-surface-unnamed, so the rule rather
  than the instance: a prose surface no test derives is named - file and section - in
  the step that sweeps it, which is this repo's own stated convention at
  doc_sync_test.go:376-380. Here that is the per-command paragraphs at
  cmd/define/README.md:724-760, where /stats, /history and /pron each have one.
  (carried from plan-quality PQ-9, deferred to the boundary review)
- **BR-3** [Minor] `terminal-ownership-unstated` the sittingInPlace signature still takes sess/stdout, the pair the stated rule says it must not build a console from
  This is the 4th finding in family terminal-ownership-unstated, so the rule not the row — and the plan now STATES the rule correctly (newSitting owns assembly, built where the rawSession and the real stdout are in scope). The signature is the one artifact left contradicting it: replayInPlace's sess is the plain session struct (replraw.go:625, session.go:9) and its stdout at the call site is con.stdout, the liveScreen (replraw.go:507), so an implementer building the pinned screen from those arguments reproduces PQ-2's fabricated 80x24 via defaultCols. Make the signature take the console.
  (carried from plan-quality PQ-10, deferred to the boundary review)
- **BR-4** [Minor] `plan-cites-unwritten-test` the plan's Verification section backticks a test that does not exist, so TestPlanCitesTestsThatExist is red on main
  go test -run TestPlanCitesTestsThatExist ./... fails today: repo_guard_test.go:1353 reports that the plan cites TestSlashPlayRefusesWhereItCannotRun and no such test exists. The plan cites that guard itself (repo_guard_test.go:1306) and then trips it, and its own Verification step 1 demands a clean suite. Un-backtick the not-yet-written names, or note the expected red until Task 1 Step 1 lands.
  (carried from plan-quality PQ-11, deferred to the boundary review)

## Round 2 — 2026-09-08T16:52:21-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Critical] `borrowed-channel-swallows-owners-events` a resize consumed during a sitting is never handed back, so the REPL screen paints at a stale shape for the rest of the session
  sittingInPlace borrows the loop's resizes channel (play_cmd.go:71, replraw.go:103) and playSession
  applies each shape to the sitting's screen only. Nothing writes it back to repl, and resume()
  (screen.go:901) repaints with rows/cols untouched; watchResize only emits on the NEXT SIGWINCH, so the
  editor is stuck at the pre-sitting geometry - too many rows scrolls the terminal, wrong cols mis-wraps
  and puts click coordinates off. screen.go:899 asserts the opposite in a doc comment. Reproduced: a
  scratch test that pre-loads resizes with 40x120, runs sittingInPlace to a scripted interrupt, and reads
  repl.Size() gets 24x80. Fix: capture sitting.Size() and repl.Resize(r, c) before repl.resume(), and
  correct the comment.
- **BR-6** [Important] `plan-named-test-not-written` all three of runPlayCommand's refusals are unpinned - the plan's Chunk 1 tests were never written
  Mutation-proven: replacing runPlayCommand's body (play_cmd.go:23-49) with a bare nil check plus
  c.startSitting() - dropping the argument refusal, the no-terminal sentence, the no-deck sentence and
  both non-zero exit codes - leaves the whole ./cmd/define suite green. No test file names runPlayCommand
  or sets startSitting. Chunk 1 Task 1 Step 1 named exactly these three tests and the step is ticked.
- **BR-7** [Important] `plan-named-test-not-written` the scoped interrupt has no test - removing Set/restore reddens nothing
  In the same mutation run, deleting restore := interrupts.Set(cancel) and defer restore()
  (play_cmd.go:97-98) reddened nothing. Without them Ctrl-C during a sitting fires the loop's cancel and
  kills define, which is the regression Done-when 2 exists to prevent. pty_conformance_test.go is
  untouched in this window; sittingInPlace and the runEditor dispatch (replraw.go:521-531) have zero
  coverage. Plan Task 2 Steps 2, 4 and 7 and Verification item 4 all claim this.
- **BR-8** [Important] `test-name-overclaims-what-it-pins` TestASittingRecordsTheSameEventFromEitherDoor never invokes either door
  play_cmd_test.go:145-183 calls playSession directly twice with identical inputs and compares the two
  results - a determinism check, not a comparison of entry points. runPlay, sittingInPlace and
  runPlayCommand appear nowhere in it, so the Done-when it is cited against rests entirely on the AST
  guard at play_cmd_test.go:101-142. Rename it to what it pins, or parameterise it over the two doors.
- **BR-9** [Important] `atlas-stale-for-new-surface` atlas update appears missing for console.newSitting, liveScreen.suspend/resume and sittingInPlace
  Only the /play command-table row was added (atlas/define.md:1119). The type table at atlas/define.md:273
  still lists console as display + resizes + finish + stdout + stderr; suspend/resume gets no entry beside
  handBack/Stop; and the "newConsole is the shared builder both loops now call" claim at
  atlas/define.md:2562 and 2579 now has a deliberate hand-assembled exception whose rationale lives only
  in a function comment.
- **BR-10** [Minor] `plan-table-stale` the plan's entity tables have no suspend/resume row and a stale sittingInPlace signature
  PQ-5's revision claims suspend/resume now has "an entity row"; neither the Pure entities nor the
  Integration points table contains one. The sittingInPlace row still reads (ctx, d, opt, sess, keys,
  interrupts, stdout, stderr); the shipped function takes repl, resizes and tty and no sess.
  TestPlanTablesNameEntitiesThatExist passes because it only checks one direction.
- **BR-11** [Minor] `capability-wider-than-its-doc` newSitting is installed on every console newConsole builds, including --play's and the board's
  replraw.go:100-104 sets con.newSitting unconditionally, so --play's pinned console and #40's board carry
  a non-nil sitting factory where a nested sitting cannot run - contradicting the field's own doc at
  replraw.go:165-168 ("nil where one cannot run"). Unreachable today; only runEditor reads it.
- **BR-12** [Minor] `refusal-wording-diverges` --play and /play give different sentences, streams and exit codes for a deckless directory
  play_loop.go:29-31 prints "no deck in this directory, so there is nothing to review" to stdout and
  returns 0; play_cmd.go:44-47 prints noDeckMessage to stderr and returns 1. Two doors, two answers to the
  same question.
- **BR-13** [Minor] `test-helper-duplicated` play_cmd_test.go re-implements reviewEvents and duplicates the AST-parse preamble
  The inline closure at play_cmd_test.go:172-183 is reviewEvents (play_loop_test.go:81) in the same
  package, and the ParseDir + FuncDecl walk is copied between play_cmd_test.go:23-58 and :104-119
  (ARCH-DRY).

## Round 3 — 2026-09-08T18:33:10-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — play_loop.go:30 still hand-rolls the sentence; verified by running both doors — --play prints "no deck in this directory, so there is nothing to review" to stdout and exits 0, /play prints noDeckMessage to stderr and exits 1. No rule stated anywhere.
- BR-2 — not-addressed — README.md:724-730 did get its paragraph, but the plan's Step 8 is still the bare "README + atlas" — the rule the finding asked for was never stated.
- BR-3 — not-addressed — The plan is unmodified in this window; the Integration-points row still reads (ctx, d, opt, sess, keys, interrupts, stdout, stderr). The code got it right, so the predicted defect did not ship — only the artifact contradicts it.
- BR-4 — addressed — Fixed in the window's base commit adcbcf1; TestPlanCitesTestsThatExist is green. The inverse is now true and folded into the new plan-table-stale finding: the plan cites no shipped test name at all.
- BR-5 — addressed — Mutation-verified: deleting play_cmd.go:103 reddens TestTheEditorScreenTakesTheShapeTheSittingEndedWith with its own sentence. Only the screen shape is handed back, though — see the new opt.width finding.
- BR-6 — addressed — Mutation-verified: replacing runPlayCommand's body with a bare nil check plus c.startSitting() reddens three of TestSlashPlayRefusals' four subtests, on sentence and exit code.
- BR-7 — not-addressed — Mutation re-run: with BOTH restore := interrupts.Set(cancel) and defer restore() deleted, TestASittingHandsTheInterruptBack still PASSES. keysFor("^") puts Key{KeyInterrupt} straight on the channel, bypassing the interrupter — the flaw the test's own comment attributes to version one — and the test installs the loop's cancel itself, so Fire()'s consumed is true either way. Only the restore half is pinned (dropping defer restore() alone does redden). The pty test the plan names at Task 2 Step 2 and Verification item 4 still does not exist; pty_conformance_test.go has no /play case.
- BR-8 — addressed — Renamed to TestTheSittingDoorRecordsWhatItAnswers and now drives sittingInPlace over a buffer with a scripted key channel, asserting through store.Events rather than a fake.
- BR-9 — addressed — atlas/define.md:2605-2626 now names sittingInPlace, the three acquisitions a borrower must not take, suspend/resume, the shape hand-back and the scoped interrupt. Residual: the console block at atlas/define.md:268-274 still enumerates five fields and omits newSitting.
- BR-10 — not-addressed — Plan unmodified. Root cause now measured: repo_guard_test.go:797 skips status "new" rows while any "- [ ] " remains, and the plan is 0/15 ticked — so both new rows are exempt rather than passing. Rolled into the new plan-table-stale finding.
- BR-11 — not-addressed — replraw.go:101-104 still sets con.newSitting unconditionally, so --play's console (play_loop.go:106) carries a factory the field's own doc at replraw.go:166-169 says is nil where a sitting cannot run.
- BR-12 — not-addressed — Confirmed by running the built binary in a deckless directory: --play says "no deck in this directory, so there is nothing to review" on stdout, exit 0; /play says noDeckMessage on stderr, exit 1, and names DEFINE_NO_CAPTURE where --play does not.
- BR-13 — not-addressed — Both duplications survive: play_cmd_test.go:169-181 re-implements reviewEvents (play_loop_test.go:81), and the ParseDir + FuncDecl preamble is still copied between play_cmd_test.go:31-63 and :106-131.

### Raised

- **BR-14** [Important] `borrowed-channel-swallows-owners-events` the borrowed resize hands back the screen shape but not opt.width, so entries looked up after a sitting wrap at the pre-sitting width
  This is the 2nd finding in family borrowed-channel-swallows-owners-events, so the rule
  rather than the instance. The rule: a borrower that consumes an owner's event stream owes
  back EVERY effect the owner's handler applies - enumerate that handler and mirror it, or
  hand the raw event back instead of one derived value. The enumeration here is exact and
  small. The owner's handler (replraw.go:404-430) applies two pieces of state: the wrap
  policy (opt.width = sz.cols, clamped to 0 below minWrapWidth, lines 424-427) and the
  screen shape (view.Resize, line 428). BR-5's fix hands back the second only, and cannot
  hand back the first - opt reaches sittingInPlace by value through con.newSitting
  (replraw.go:532) and again by value into playSession. So a SIGWINCH consumed by a sitting
  leaves lookupAndRender (replraw.go:685) and newCommandCtx (replraw.go:495) using the
  pre-sitting width for the rest of the session; on a narrowed terminal Paint's clipVisible
  then cuts the over-wide lines at the right edge. Fix the rule: factor lines 424-428 into
  one applyResize(sz, &opt, view) that both the owner's case and the sitting's hand-back
  call, so a third effect added later is handed back by construction.
- **BR-15** [Important] `plan-table-stale` the durable plan is 0/15 ticked at close, which exempts its two new-entity rows from the guard that would have checked them
  This is the 2nd finding in family plan-table-stale, so the rule rather than the rows. The
  rule: at a close boundary the durable plan is reconciled with what shipped in ONE sweep -
  checkboxes, entity rows, signatures, test names - because the guards that read plans key
  off "is this plan still in progress". Measured: repo_guard_test.go:797 does
  `if inProgress && status == "new" { continue }`, and inProgress is
  `strings.Contains(body, "- [ ] ")`. The plan has 15 unticked steps and 0 ticked, so BOTH
  new rows (runPlayCommand, sittingInPlace) are skipped outright - that is how BR-3's and
  BR-10's stale rows survived a green suite. Same sweep, same rule: the sittingInPlace row's
  signature still names sess and stdout against a shipped (ctx, d, opt, keys, interrupts,
  repl, resizes, tty, stderr); there is still no suspend/resume entity row despite PQ-5's
  revision claiming one; and the Verification section now backticks no test name at all,
  de-backticked in adcbcf1 "until the tests land" - they have landed, and
  TestPlanCitesTestsThatExist's own message says "write it, or cite the one that shipped".
  Also record what did not ship: the pty test at Task 2 Step 2, and Step 1's shared-half
  extraction (play_loop.go is unmodified).
- **BR-16** [Minor] `new-seam-surface-unshaped` console.newSitting takes a parameter it already captures, and is named like a constructor while running a whole sitting
  replraw.go:101-104 builds the closure over `live`, and runEditor passes stderr := con.stderr
  (replraw.go:532), which IS `live` - so the sixth parameter is always the value already
  captured and can be dropped. Separately, the field is `func(...) int` that performs a
  sitting and returns its exit code, while the plan specified `newSitting func() console`, a
  constructor; a `new*` name on a verb is the one thing a reader of the dispatch at
  replraw.go:532 cannot guess. Rename to runSitting and drop the redundant writer. This is a
  newly-introduced internal seam that downstream work will consume, so the surface is worth
  settling now.

## Round 4 — 2026-09-08T18:56:35-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — play_loop.go is unmodified in this window and no artifact states the rule; runPlay:30 still hand-rolls the sentence.
- BR-2 — addressed — The named surface shipped at cmd/define/README.md:724-730; Step 8's wording is folded into the plan-revision list.
- BR-3 — addressed — The shipped signature takes repl *liveScreen and the real tty, not sess/stdout; the stale plan ROW is BR-10/BR-15's.
- BR-6 — not-addressed — TestSlashPlayRefusals was written in 15ecac4 and DELETED in c2e593a; removing the nil-capability refusal now reddens nothing.
- BR-7 — addressed — Verified by reverting Set/restore in a scratch copy - the test fails at "the sitting did not end when its own interrupt fired".
- BR-10 — not-addressed — Still no suspend/resume entity row, and line 144's sittingInPlace signature is unchanged.
- BR-11 — not-addressed — replraw.go:123 still sets con.newSitting unconditionally, so --play's console and the board carry it.
- BR-12 — not-addressed — play_loop.go is unmodified; the two doors still differ in sentence, stream and exit code.
- BR-13 — not-addressed — reviewEvents is still re-implemented at play_cmd_test.go:172-178 and ParseDir is still copied at :31 and :107.
- BR-14 — addressed — Verified by bypassing applyShape on the hand-back route - both clauses of TestBothShapeRoutesGoThroughOnePlace fail.
- BR-15 — not-addressed — Boxes ticked but two of them falsely (Task 2 Steps 1 and 2); signature row, suspend/resume row, Verification backticks and the did-not-ship record are all still missing.
- BR-16 — not-addressed — Still named newSitting for a verb, and still takes the stderr it already captures as live.

### Raised

- **BR-17** [Important] `plan-named-test-not-written` the /play dispatch in runEditor is unpinned - making the command a no-op leaves the suite green
  This is the 3rd finding in family plan-named-test-not-written, so the rule rather
  than the instance. Measured: replacing cc.startSitting = func() { sitting = true }
  (replraw.go:540) with a no-op leaves every non-git-dependent test passing, so /play
  typed at the prompt can silently do nothing. The rule: a plan step is not ticked
  until the test it names exists AND a named mutation of the code it covers reddens
  it. The enumeration is exact - Verification lists 6 items plus 3 Test-surface rows;
  delivered and mutation-checked are items 3, 5 (one door), 6 and the shape rows;
  unmet are item 2 (the refusals, deleted in c2e593a - see BR-6) and item 4 (the pty
  row, pty_conformance_test.go untouched in this window). Two obligations unmet while
  15/15 boxes read done. Adopt the rule at the tick and record the two unmet
  obligations in the plan's Revisions instead of ticking over them.
- **BR-18** [Important] `unverified-mechanism-claim` lessons.md states a Go semantics rule that is false, and records a superseded approach as what worked
  workshop/lessons.md:4069 says "return code, shape evaluates shape before the defer
  runs, so it hands back the pre-sitting size". With NAMED results - which
  sittingInPlace has - that is not so; I ran it, and the deferred mutation still wins
  for both a bare return and return code, shape. The real requirement is that the
  results be named, which they are. The same wrong claim is in the code comment at
  play_cmd.go:75-78. Separately, lines 4056-4060 record the round-1 approach ("narrow
  the claim to the RESTORE... the end-to-end path is covered by the pty run") that
  BR-7 rejected and c2e593a replaced, and there is no pty run for /play. AGENTS.md
  section 4 makes lessons.md the rule-store, so a wrong rule there misdirects future
  work; rewrite both entries to what actually held.

## Round 5 — 2026-09-08T19:32:27-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — play_loop.go untouched in this window; runPlay:29-31 still hand-rolls the sentence to stdout with exit 0, and noDeckMessage's doc (main.go:1209) is unchanged. Neither the sweep nor the rule-statement happened.
- BR-6 — not-addressed — Mutation-measured on a scratch checkout of ea74e04: gutting all three refusals leaves the ./cmd/define failure set byte-identical to baseline. No test names runPlayCommand. Worse, c2e593a TICKED Chunk 1 Step 1 in the same commit whose Revisions entry records this finding.
- BR-10 — addressed — Integration points now carries the liveScreen.suspend/resume row, and sittingInPlace's signature matches the shipped one exactly.
- BR-11 — not-addressed — replraw.go:123 still assigns unconditionally; play_loop.go:106 builds --play's console through newConsole, so it carries a non-nil factory the field doc at replraw.go:187-191 says must be nil.
- BR-12 — not-addressed — Unchanged - stdout/exit 0 versus stderr/exit 1 for the same deckless directory.
- BR-13 — not-addressed — play_cmd_test.go:170-179 still re-implements reviewEvents (play_loop_test.go:81); the ParseDir preamble is still copied between play_cmd_test.go:32 and :108.
- BR-15 — not-addressed — Half swept - entity rows, signature, Step 1's DID-NOT-SHIP note, Step 2's rewrite. Not swept, and one moved backwards - Chunk 1 Step 1 was ticked in c2e593a with none of its three tests written; Verification item 4 still claims a pty test Step 2 says was not written; Verification still backticks no test name though ten landed (TestPlanCitesTestsThatExist SKIPS here, no Done-when heading); "The three things that must be built" 2 still specifies newSitting func() console.
- BR-16 — not-addressed — Field still named newSitting while returning (int, winSize); stderr param is still always con.stderr == live, the value the closure already captures.
- BR-17 — not-addressed — The INSTANCE is fixed and I verified it - no-opping cc.startSitting at replraw.go:540 reddens TestTypingSlashPlayRunsASitting and TestTheLoopAppliesTheShapeASittingHandsBack. The escalated RULE was not adopted - the same window ticked Chunk 1 Step 1 with none of its named tests written, and a third site of the same class (newConsole's closure) is unpinned.
- BR-18 — addressed — lessons.md now states the correct rule and records the earlier error; the round-1 narrowing is recorded as also wrong; play_cmd.go:75-81 corrected. I re-verified the Go semantics independently - named results, defer wins for both bare and explicit return.

### Raised

- **BR-19** [Important] `plan-named-test-not-written` newConsole's newSitting closure can stop calling sittingInPlace entirely and the whole suite stays green
  4th in family - so the rule, not the instance. A capability delivered
  through an injected field is pinned at BOTH ends: the consumer, and the
  production assembly that supplies the real implementation. A test that
  installs its own double for field F proves the consumer and nothing about
  the producer. Enumeration is exact - this issue added three wiring sites,
  all mutation-measured against ea74e04 in a scratch checkout: the loop
  dispatch (replraw.go:540) reddens two tests; runPlayCommand's refusals and
  newConsole's closure (replraw.go:123-126, body replaced with
  "return 0, winSize{}") each leave the failure set byte-identical to
  baseline. Two of three unpinned. TestTypingSlashPlayRunsASitting sets
  con.newSitting itself so it can never see the producer; the two AST guards
  walk sittingInPlace by name whether or not anything reaches it. Cheap
  enforcement already exists in this file - add to
  TestBothEntryPointsReachOnePlaySession that newConsole's body reaches
  sittingInPlace, derived from the source.
- **BR-20** [Minor] `guard-scope-narrower-than-claim` the one-place-for-a-shape guard parses only replraw.go while a second bare Resize lives in play_cmd.go
  TestBothShapeRoutesGoThroughOnePlace (play_cmd_test.go:319-321) calls
  parser.ParseFile on "replraw.go" alone, and its message says "a shape
  applied outside applyShape is a shape whose width policy was forgotten" -
  but play_cmd.go:116 does exactly that, repl.Resize(r, c) with no opt.width,
  one file over. Benign today because runEditor calls applyShape immediately
  after, so the class the guard names is enforced only where it happens to
  look.

## Round 6 — 2026-09-08T21:39:28-07:00 (claude) — passed

### Disposed

- BR-17 — addressed — Mutation-verified: no-op'ing cc.startSitting at replraw.go:540 reddens TestTypingSlashPlayRunsASitting and TestTheLoopAppliesTheShapeASittingHandsBack.
- BR-19 — addressed — Mutation-verified: replacing newConsole's newSitting closure body with `return 0, winSize{}` reddens TestTheProducedSittingCapabilityCallsTheRealThing.
- BR-6 — not-addressed — Re-measured at 116d0f3: gutting runPlayCommand to a bare nil check plus c.startSitting() leaves the whole ./cmd/define suite green. No test names runPlayCommand.
- BR-15 — not-addressed — Entity rows and the DID-NOT-SHIP record are fixed; the plan still carries `newSitting func() console`, two pty claims contradicting Task 2 Step 2, no Forget row, no revision for rounds 3-6, and no backticked test names. I confirmed by execution that a bogus `new` row still passes the guard while the plan is inProgress.
- BR-1 — not-addressed — play_loop.go:30 still hand-rolls the sentence; no rule stated in code, plan or lessons.md.
- BR-12 — not-addressed — Unchanged - play_loop.go:30 stdout/exit-0 versus play_cmd.go:44-45 stderr/exit-1.
- BR-11 — not-addressed — replraw.go:123 still assigns con.newSitting unconditionally; the field doc at replraw.go:189-193 is unchanged since 2dc2100.
- BR-13 — not-addressed — play_cmd_test.go:171-180 still re-implements reviewEvents; the AST preamble is now duplicated four ways (:32, :108, :321, :458).
- BR-16 — not-addressed — Signature and name unchanged; runEditor still passes con.stderr, which is the `live` screen the closure already captures.
- BR-20 — not-addressed — Guard still parses replraw.go alone; play_cmd.go:116 still calls repl.Resize directly.

### Raised

- **BR-21** [Important] `inverse-op-diverges-from-its-pair` memVocabulary.Forget recounts maxWords without Add's phraseRunsJoin filter, so a drop can widen the phrase window past what Add allows
  vocab.go:101-106 recomputes maxWords as max(len(wordRuns(w))) over every key, while Add
  (vocab.go:129-131) raises it only when phraseRunsJoin(key, runs) - because a key holding
  other punctuation is permanently unmatchable and counting it makes every stream hold a
  wider window for a match that cannot happen, a cost Add's comment records as measured.
  Executed at 116d0f3: Add("keel"); Add("e.g."); Add("junk") gives MaxPhraseWords()==1, and
  Forget("junk") raises it to 2. Any /play drop on a deck holding one punctuated entry
  widens the streaming renderer's lookahead for the rest of the session. No wrong highlight
  results, so the blast radius is latency (ARCH-CONSTRAINTS), but the invariant is broken
  and the two maintainers of one derived aggregate should be one helper (ARCH-DRY). The
  existing test uses `keel` and `hot dog` and cannot see it - written from the fix's own
  model.
- **BR-22** [Important] `plan-named-test-not-written` suspend/resume and the sitting's finish are unpinned at the call site, and BR-19's hand-written enumeration of wiring sites was wrong
  This is the 5th finding in family plan-named-test-not-written. Not asking for these
  instances to be patched - the rule is the deliverable. Measured at 116d0f3, full
  ./cmd/define suite per mutation: deleting repl.suspend() (play_cmd.go:97) and
  repl.resume() (play_cmd.go:117) leaves it GREEN; replacing the sitting console's finish
  body (play_cmd.go:137-140) with a no-op leaves it GREEN; gutting runPlayCommand leaves it
  GREEN (BR-6). So the issue's headline capability - the editor's throttled painter going
  quiet so it cannot land inside the sitting's frame - and the README's promised
  summary-in-the-scrollback are both unpinned. BR-19 stated the both-ends rule and then
  HAND-LISTED the sites ("the enumeration is exact - three wiring sites"); it missed three.
  The rule: an enumeration of sites obliged to obey a capability must be DERIVED from the
  source, not listed from memory - the discipline TestASittingFromTheLoopNeverEntersRawMode
  already applies to callees and TestBothShapeRoutesGoThroughOnePlace to routes. One AST
  guard over sittingInPlace's body (calls suspend, calls resume, writes Transcript() into
  repl), shaped like the drop-arm guard at play_cmd_test.go:557-616, covers the class.

## Open findings

- **BR-1** [Minor] `refusal-enumeration-incomplete` the two entry points would print different sentences for the same nil deck
- **BR-6** [Important] `plan-named-test-not-written` all three of runPlayCommand's refusals are unpinned - the plan's Chunk 1 tests were never written
- **BR-11** [Minor] `capability-wider-than-its-doc` newSitting is installed on every console newConsole builds, including --play's and the board's
- **BR-12** [Minor] `refusal-wording-diverges` --play and /play give different sentences, streams and exit codes for a deckless directory
- **BR-13** [Minor] `test-helper-duplicated` play_cmd_test.go re-implements reviewEvents and duplicates the AST-parse preamble
- **BR-15** [Important] `plan-table-stale` the durable plan is 0/15 ticked at close, which exempts its two new-entity rows from the guard that would have checked them
- **BR-16** [Minor] `new-seam-surface-unshaped` console.newSitting takes a parameter it already captures, and is named like a constructor while running a whole sitting
- **BR-20** [Minor] `guard-scope-narrower-than-claim` the one-place-for-a-shape guard parses only replraw.go while a second bare Resize lives in play_cmd.go
- **BR-21** [Important] `inverse-op-diverges-from-its-pair` memVocabulary.Forget recounts maxWords without Add's phraseRunsJoin filter, so a drop can widen the phrase window past what Add allows
- **BR-22** [Important] `plan-named-test-not-written` suspend/resume and the sitting's finish are unpinned at the call site, and BR-19's hand-written enumeration of wiring sites was wrong
