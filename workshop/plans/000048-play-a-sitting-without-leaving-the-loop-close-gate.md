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

## Open findings

- **BR-1** [Minor] `refusal-enumeration-incomplete` the two entry points would print different sentences for the same nil deck
- **BR-2** [Minor] `hand-swept-surface-unnamed` Step 8 is the bare item "README + atlas" and names no section for the hand-swept half
- **BR-3** [Minor] `terminal-ownership-unstated` the sittingInPlace signature still takes sess/stdout, the pair the stated rule says it must not build a console from
- **BR-4** [Minor] `plan-cites-unwritten-test` the plan's Verification section backticks a test that does not exist, so TestPlanCitesTestsThatExist is red on main
- **BR-5** [Critical] `borrowed-channel-swallows-owners-events` a resize consumed during a sitting is never handed back, so the REPL screen paints at a stale shape for the rest of the session
- **BR-6** [Important] `plan-named-test-not-written` all three of runPlayCommand's refusals are unpinned - the plan's Chunk 1 tests were never written
- **BR-7** [Important] `plan-named-test-not-written` the scoped interrupt has no test - removing Set/restore reddens nothing
- **BR-8** [Important] `test-name-overclaims-what-it-pins` TestASittingRecordsTheSameEventFromEitherDoor never invokes either door
- **BR-9** [Important] `atlas-stale-for-new-surface` atlas update appears missing for console.newSitting, liveScreen.suspend/resume and sittingInPlace
- **BR-10** [Minor] `plan-table-stale` the plan's entity tables have no suspend/resume row and a stale sittingInPlace signature
- **BR-11** [Minor] `capability-wider-than-its-doc` newSitting is installed on every console newConsole builds, including --play's and the board's
- **BR-12** [Minor] `refusal-wording-diverges` --play and /play give different sentences, streams and exit codes for a deckless directory
- **BR-13** [Minor] `test-helper-duplicated` play_cmd_test.go re-implements reviewEvents and duplicates the AST-parse preamble
