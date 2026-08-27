---
gate: boundary-review
issue: 6
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-27T08:51:51-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: CaptureReview(word, correct bool, opt) cannot represent Skipped, yet the plan tests the skip against it
          detail: |-
            Task 2 Step 1 says a skip emits a record Outcome carrying Skipped; Task 3 Step 1 tests "a SKIP records
            nothing at all" against CaptureReview, whose only verdict input is a bool. Decide which side filters:
            Apply emits no record outcome for a skip (test moves to session_test.go), or CaptureReview takes the
            Verdict — which introduces a main-to-play dependency edge the plan never draws.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: signature-cannot-express-contract
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-27T08:51:51-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: puretest has no tests, yet the issue Log and project entry both claim its negative cases are verified in the tree
          detail: |-
            cmd/define/puretest reports "[no test files]". Gutting the t.Errorf in ImportsOnly (puretest.go:52) and
            NoWallClock (puretest.go:76) leaves both play and schedule green, so the shared guard body five packages
            will depend on has zero negative coverage. workshop/issues/000006-vocab-play.md:180 states the extraction
            closed that gap "with the negative cases verified in the tree" and workshop/projects/define-learn.md:527
            repeats it. Fix: take a reporter interface instead of *testing.T, add puretest_test.go driving each guard
            against a testdata subject that is deliberately impure, and assert it reports. Correct BOTH artifacts, not
            one (ARCH-PURPOSE).
          family: claim-without-failing-test
          round: 2
        - id: BR-3
          severity: Important
          title: Answering the last question sets Session.Done but returns OutcomeRecord, so the outcome alone never says the session ended
          detail: |-
            session.go:139-158 — advance sets s.Done and returns OutcomeRecord (or OutcomeNone for a skip);
            OutcomeDone is only produced by a subsequent Apply. A loop switching solely on Outcome.Kind blocks on
            the key channel after the final answer. Session.Done has no doc comment and OutcomeDone's says only
            "the session is over". M2's runPlay is the consumer. Add Outcome.SessionDone, or document the
            two-signal contract on Apply and pin it with a test.
          family: two-signal-termination
          round: 2
        - id: BR-4
          severity: Important
          title: The Core concepts table places Verdict in session.go, claims play imports store, and omits the puretest package
          detail: |-
            plan:21 lists Verdict at cmd/define/play/session.go; it is declared at question.go:21. plan:27 says
            "play imports store, schedule and pure stdlib" and the Test-surface note promises the store SYMBOL
            guard; play imports nothing and purity_test.go:13-19 deliberately omits StoreSymbolsOnly. puretest, a
            new package introduced at this boundary that shells out to the go binary, has no row in either table.
            Every entity exists and is exercised, which is why this is not filed Critical.
          family: plan-table-contradicts-code
          round: 2
        - id: BR-5
          severity: Important
          title: Plan Task 2 Step 1 and Task 3 Step 1 still instruct the superseded skip contract, leaving PQ-3 open into M2
          detail: |-
            plan:98 says "a skip emits a record outcome with Skipped"; session.go:152 emits OutcomeNone, which is
            what the Done-when requires. plan:122 says "the loop drops it", contradicting "ONE filter, in Apply".
            plan:98 also says "the last answer emits quit". The plan-quality ledger lists PQ-3 as open after five
            rounds. M2 is executed from this file, so correct it at this boundary.
          family: signature-cannot-express-contract
          round: 2
        - id: BR-6
          severity: Minor
          title: TestSkipAdvancesButRecordsNothing drives an ungraded key and asserts the session did NOT advance
          detail: |-
            session_test.go:70 — the body drives 's', which Recall does not grade, and fails if s.Index != 0. The
            skip contract is covered at line 84. Rename to TestUngradedKeyIsIgnored.
          family: test-name-contradicts-assertion
          round: 2
        - id: BR-7
          severity: Minor
          title: Form 2.1 has no skip key, so Verdict.Skipped is unreachable in production, and no artifact records the decision
          detail: |-
            recall.go:33 grades only y/Y/n/N. Defensible against the Spec's "self-rate right/wrong", but the issue
            Log at 000006-vocab-play.md:150 frames skipping as a learner action. Record it in the Log or the plan.
          family: unrecorded-scope-decision
          round: 2
        - id: BR-8
          severity: Minor
          title: skipForm duplicates a capability fakeForm already has
          detail: |-
            session_test.go:216 — fakeForm.Grade('3') already returns (Skipped, true) at line 211, so
            TestSkippedVerdictRecordsNothing could drive fakeForm and one double could go.
          family: duplicated-guard-logic
          round: 2
        - id: BR-9
          severity: Minor
          title: Grade overloads Skipped as both a real verdict and the zero value returned with ok == false
          detail: |-
            question.go:45 — callers check ok first so nothing is wrong today, but Verdict's zero value being
            Skipped deserves a sentence in the doc comment.
          family: overloaded-zero-value
          round: 2
        - id: BR-10
          severity: Minor
          title: puretest discards go list stderr, so a failed measurement fatals with only an exit status
          detail: |-
            puretest.go:45 and :122 use exec.Command(...).Output() and report only err. Capture
            exitErr.Stderr into the fatal message.
          family: discarded-error-detail
          round: 2
        - id: BR-11
          severity: Minor
          title: Every Chunk 1 step in the durable plan is still unticked although M1 is complete
          detail: |-
            The issue's Plan M1 row is [x] but workshop/plans/000006-vocab-play-plan.md Chunk 1 Steps 1-9 remain
            "- [ ]". The durable plan is the traceability record.
          family: plan-checkboxes-not-ticked
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-27T10:40:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Apply filters (session.go:161-165); the test lives at session_test.go:88 and reddens without the branch.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: cmd/define/puretest still reports "[no test files]"; issue:183 still claims the negative cases are in the tree.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: No Outcome.SessionDone and no doc on Apply/Session.Done/OutcomeDone; playSession happens to loop on !s.Done.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: plan:21, plan:26 and plan:48 unchanged; still no puretest row, and now no Input/Outcome rows either.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: plan:98 and plan:122 are byte-identical to the round-2 text; PQ-3 is still open.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: session_test.go:74 still named TestSkipAdvancesButRecordsNothing while asserting Index == 0.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: recall.go:33 unchanged; no Log or plan entry records the no-skip-key decision.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: skipForm still at session_test.go:220 beside fakeForm's identical Grade('3') capability.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: question.go:45 doc comment unchanged; Verdict's zero value is still silently Skipped.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: puretest.go:43 and :123 still use .Output(); exitErr.Stderr is still discarded.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: Every step in both chunks is still "- [ ]", now including all of Chunk 2.
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: 'Two Done-when rows are ticked with no test: the audio block has measured zero coverage and -count is never exercised'
          detail: |-
            2nd in this family (with BR-2), so fix the RULE, not the instance. go test -coverprofile shows
            play_loop.go 131.37-142.7 at count 0; playRig (play_loop_test.go:33) hard-codes noAudio true, so
            fakePlayer is installed and never asked anything (ARCH-MOCK). -count is passed as 20 against 2-word
            decks, so neither the bound nor the default is read by a test. Plan Task 4 Step 1b demanded the audio
            row explicitly. The rule: no Done-when row is ticked without a named test in the tree that fails
            without it. I walked the enumeration - rows 3 (audio) and 4 (-count) are the only two with no pinning
            test. Write that row-to-test map into the issue and close both in this round, with BR-2.
          family: claim-without-failing-test
          round: 3
        - id: BR-13
          severity: Important
          title: 'The --play and -count flag wiring at main.go:369 has zero test coverage, the class #21 already shipped twice'
          detail: |-
            No test calls run() with --play; every play test enters at runPlay or playSession with a hand-built
            deps. Plan Task 4 Step 8 committed to "the same entry-path enumeration #21 needed", and
            news_test.go:209-212 records the precedent in this repo's own words. Nothing today catches opt.count
            not being threaded, withStore being dropped, or the dispatch moving below the argument-count switch.
            Fix: drive run(ctx, []string{"--play"}, ...) and run(ctx, []string{"--play","-count","3"}, ...)
            through production wiring with a non-terminal stdin.
          family: dead-entry-path
          round: 3
        - id: BR-14
          severity: Important
          title: define --play sycophantic runs a full session and ignores the word, unlike --forget and --reflect
          detail: |-
            main.go:389-392 states the rule and pins it for --reflect (TestReflectWithAWordIsAUsageError):
            a mode plus a word is two commands on one line, and silently honouring one is how -raw came to mean
            two things in #2. --play is dispatched at main.go:369, above that switch, so it can never reach it.
            Unlike --llm-check, which shares the position, --play writes events - so state changes under a
            misread intent. Fix: add a case for *playFlag and fs.NArg() != 0, and move the dispatch below the
            switch (it already needs withStore, which sits there).
          family: mode-silently-ignores-argument
          round: 3
        - id: BR-15
          severity: Important
          title: The d drop key ships undocumented and both README and atlas describe a prompt line the code no longer prints
          detail: |-
            eba07e0 added d/D to InputDrop, which permanently removes a word from the deck on one keystroke.
            README.md:41-51 never mentions it and its sample transcript prints "Enter or space to reveal, Ctrl-C
            to stop", which draw() (play_loop.go:233) stopped emitting. atlas/define.md:1215 is now false:
            "everything else is a rune for the form to grade" - d and D are not. Neither InputDrop/OutcomeDrop,
            nor the session-output-through-crlfWriter rule, nor the check-cancellation-before-select rule appear
            in the atlas, though all three are this window's durable decisions. Both the atlas and README gates fire.
          family: docs-not-updated-for-new-surface
          round: 3
        - id: BR-16
          severity: Important
          title: play_loop.go:140 swallows the raw-mode re-entry error, leaving the terminal cooked and the session apparently frozen
          detail: |-
            2nd in this family (with BR-10), so fix the RULE, not the instance. "if again, err := enterRaw(os.Stdin);
            err == nil" drops the error entirely: after a failed re-entry readKeys is line-buffered, so every
            keystroke appears to do nothing until Enter, and nothing reaches stderr. The rule: an error is acted on
            or reported, never dropped where it is available. Production sites in this window - play_loop.go:140
            (dropped entirely) and puretest.go:43,123 (exitErr.Stderr discarded). play_loop.go:184-188 is the
            correct shape and should be the model. Sweep both.
          family: discarded-error-detail
          round: 3
        - id: BR-17
          severity: Minor
          title: playSession re-enters raw mode on os.Stdin rather than the file runPlay was handed
          detail: |-
            play_loop.go:140 - the one hard-wired IO reference in an otherwise fully injected loop (ARCH-PURE),
            and the reason the audio branch cannot be driven at all. Thread the *os.File through and the
            audio Done-when becomes testable.
          family: io-not-injected
          round: 3
        - id: BR-18
          severity: Minor
          title: The cancel-before-select fix is pinned only probabilistically - roughly 1 run in 4 reddens
          detail: |-
            play_loop_test.go:218 - with the guard removed, ctx.Done() and a buffered key are both ready and
            select picks at random, so recording an event needs two coin flips to go the wrong way. A
            deterministic form (assert the first scripted key is still buffered on return) fails every time.
          family: probabilistic-regression-test
          round: 3
        - id: BR-19
          severity: Minor
          title: anyTime spells time.Time{} as store.Word{}.FirstSeen, while reflect.go:250 writes the plain form
          detail: |-
            play_loop.go:217 ties an "everything since the beginning" sentinel to an unrelated struct field's
            type. ARCH-DRY: one spelling for one fact.
          family: duplicate-sentinel-spelling
          round: 3
        - id: BR-20
          severity: Minor
          title: toInput reserves Enter, space, d and D from every form, qualifying the second-form Done-when, and no artifact records it
          detail: |-
            2nd in this family (with BR-7), so fix the RULE, not the instance. The rule: a decision that
            constrains a future consumer is recorded in the Log or plan in the same round. Open enumeration is
            exactly two - form 2.1 having no skip key (BR-7), and the loop's reserved key set at
            play_loop.go:157-172. #7 hits the second when it picks its keys.
          family: unrecorded-scope-decision
          round: 3
      blocked: true
---

# Gate ledger — tools#6 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T08:51:51-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `signature-cannot-express-contract` CaptureReview(word, correct bool, opt) cannot represent Skipped, yet the plan tests the skip against it
  Task 2 Step 1 says a skip emits a record Outcome carrying Skipped; Task 3 Step 1 tests "a SKIP records
  nothing at all" against CaptureReview, whose only verdict input is a bool. Decide which side filters:
  Apply emits no record outcome for a skip (test moves to session_test.go), or CaptureReview takes the
  Verdict — which introduces a main-to-play dependency edge the plan never draws.
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-08-27T08:51:51-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `claim-without-failing-test` puretest has no tests, yet the issue Log and project entry both claim its negative cases are verified in the tree
  cmd/define/puretest reports "[no test files]". Gutting the t.Errorf in ImportsOnly (puretest.go:52) and
  NoWallClock (puretest.go:76) leaves both play and schedule green, so the shared guard body five packages
  will depend on has zero negative coverage. workshop/issues/000006-vocab-play.md:180 states the extraction
  closed that gap "with the negative cases verified in the tree" and workshop/projects/define-learn.md:527
  repeats it. Fix: take a reporter interface instead of *testing.T, add puretest_test.go driving each guard
  against a testdata subject that is deliberately impure, and assert it reports. Correct BOTH artifacts, not
  one (ARCH-PURPOSE).
- **BR-3** [Important] `two-signal-termination` Answering the last question sets Session.Done but returns OutcomeRecord, so the outcome alone never says the session ended
  session.go:139-158 — advance sets s.Done and returns OutcomeRecord (or OutcomeNone for a skip);
  OutcomeDone is only produced by a subsequent Apply. A loop switching solely on Outcome.Kind blocks on
  the key channel after the final answer. Session.Done has no doc comment and OutcomeDone's says only
  "the session is over". M2's runPlay is the consumer. Add Outcome.SessionDone, or document the
  two-signal contract on Apply and pin it with a test.
- **BR-4** [Important] `plan-table-contradicts-code` The Core concepts table places Verdict in session.go, claims play imports store, and omits the puretest package
  plan:21 lists Verdict at cmd/define/play/session.go; it is declared at question.go:21. plan:27 says
  "play imports store, schedule and pure stdlib" and the Test-surface note promises the store SYMBOL
  guard; play imports nothing and purity_test.go:13-19 deliberately omits StoreSymbolsOnly. puretest, a
  new package introduced at this boundary that shells out to the go binary, has no row in either table.
  Every entity exists and is exercised, which is why this is not filed Critical.
- **BR-5** [Important] `signature-cannot-express-contract` Plan Task 2 Step 1 and Task 3 Step 1 still instruct the superseded skip contract, leaving PQ-3 open into M2
  plan:98 says "a skip emits a record outcome with Skipped"; session.go:152 emits OutcomeNone, which is
  what the Done-when requires. plan:122 says "the loop drops it", contradicting "ONE filter, in Apply".
  plan:98 also says "the last answer emits quit". The plan-quality ledger lists PQ-3 as open after five
  rounds. M2 is executed from this file, so correct it at this boundary.
- **BR-6** [Minor] `test-name-contradicts-assertion` TestSkipAdvancesButRecordsNothing drives an ungraded key and asserts the session did NOT advance
  session_test.go:70 — the body drives 's', which Recall does not grade, and fails if s.Index != 0. The
  skip contract is covered at line 84. Rename to TestUngradedKeyIsIgnored.
- **BR-7** [Minor] `unrecorded-scope-decision` Form 2.1 has no skip key, so Verdict.Skipped is unreachable in production, and no artifact records the decision
  recall.go:33 grades only y/Y/n/N. Defensible against the Spec's "self-rate right/wrong", but the issue
  Log at 000006-vocab-play.md:150 frames skipping as a learner action. Record it in the Log or the plan.
- **BR-8** [Minor] `duplicated-guard-logic` skipForm duplicates a capability fakeForm already has
  session_test.go:216 — fakeForm.Grade('3') already returns (Skipped, true) at line 211, so
  TestSkippedVerdictRecordsNothing could drive fakeForm and one double could go.
- **BR-9** [Minor] `overloaded-zero-value` Grade overloads Skipped as both a real verdict and the zero value returned with ok == false
  question.go:45 — callers check ok first so nothing is wrong today, but Verdict's zero value being
  Skipped deserves a sentence in the doc comment.
- **BR-10** [Minor] `discarded-error-detail` puretest discards go list stderr, so a failed measurement fatals with only an exit status
  puretest.go:45 and :122 use exec.Command(...).Output() and report only err. Capture
  exitErr.Stderr into the fatal message.
- **BR-11** [Minor] `plan-checkboxes-not-ticked` Every Chunk 1 step in the durable plan is still unticked although M1 is complete
  The issue's Plan M1 row is [x] but workshop/plans/000006-vocab-play-plan.md Chunk 1 Steps 1-9 remain
  "- [ ]". The durable plan is the traceability record.

## Round 3 — 2026-08-27T10:40:55-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Apply filters (session.go:161-165); the test lives at session_test.go:88 and reddens without the branch.
- BR-2 — not-addressed — cmd/define/puretest still reports "[no test files]"; issue:183 still claims the negative cases are in the tree.
- BR-3 — not-addressed — No Outcome.SessionDone and no doc on Apply/Session.Done/OutcomeDone; playSession happens to loop on !s.Done.
- BR-4 — not-addressed — plan:21, plan:26 and plan:48 unchanged; still no puretest row, and now no Input/Outcome rows either.
- BR-5 — not-addressed — plan:98 and plan:122 are byte-identical to the round-2 text; PQ-3 is still open.
- BR-6 — not-addressed — session_test.go:74 still named TestSkipAdvancesButRecordsNothing while asserting Index == 0.
- BR-7 — not-addressed — recall.go:33 unchanged; no Log or plan entry records the no-skip-key decision.
- BR-8 — not-addressed — skipForm still at session_test.go:220 beside fakeForm's identical Grade('3') capability.
- BR-9 — not-addressed — question.go:45 doc comment unchanged; Verdict's zero value is still silently Skipped.
- BR-10 — not-addressed — puretest.go:43 and :123 still use .Output(); exitErr.Stderr is still discarded.
- BR-11 — not-addressed — Every step in both chunks is still "- [ ]", now including all of Chunk 2.

### Raised

- **BR-12** [Important] `claim-without-failing-test` Two Done-when rows are ticked with no test: the audio block has measured zero coverage and -count is never exercised
  2nd in this family (with BR-2), so fix the RULE, not the instance. go test -coverprofile shows
  play_loop.go 131.37-142.7 at count 0; playRig (play_loop_test.go:33) hard-codes noAudio true, so
  fakePlayer is installed and never asked anything (ARCH-MOCK). -count is passed as 20 against 2-word
  decks, so neither the bound nor the default is read by a test. Plan Task 4 Step 1b demanded the audio
  row explicitly. The rule: no Done-when row is ticked without a named test in the tree that fails
  without it. I walked the enumeration - rows 3 (audio) and 4 (-count) are the only two with no pinning
  test. Write that row-to-test map into the issue and close both in this round, with BR-2.
- **BR-13** [Important] `dead-entry-path` The --play and -count flag wiring at main.go:369 has zero test coverage, the class #21 already shipped twice
  No test calls run() with --play; every play test enters at runPlay or playSession with a hand-built
  deps. Plan Task 4 Step 8 committed to "the same entry-path enumeration #21 needed", and
  news_test.go:209-212 records the precedent in this repo's own words. Nothing today catches opt.count
  not being threaded, withStore being dropped, or the dispatch moving below the argument-count switch.
  Fix: drive run(ctx, []string{"--play"}, ...) and run(ctx, []string{"--play","-count","3"}, ...)
  through production wiring with a non-terminal stdin.
- **BR-14** [Important] `mode-silently-ignores-argument` define --play sycophantic runs a full session and ignores the word, unlike --forget and --reflect
  main.go:389-392 states the rule and pins it for --reflect (TestReflectWithAWordIsAUsageError):
  a mode plus a word is two commands on one line, and silently honouring one is how -raw came to mean
  two things in #2. --play is dispatched at main.go:369, above that switch, so it can never reach it.
  Unlike --llm-check, which shares the position, --play writes events - so state changes under a
  misread intent. Fix: add a case for *playFlag and fs.NArg() != 0, and move the dispatch below the
  switch (it already needs withStore, which sits there).
- **BR-15** [Important] `docs-not-updated-for-new-surface` The d drop key ships undocumented and both README and atlas describe a prompt line the code no longer prints
  eba07e0 added d/D to InputDrop, which permanently removes a word from the deck on one keystroke.
  README.md:41-51 never mentions it and its sample transcript prints "Enter or space to reveal, Ctrl-C
  to stop", which draw() (play_loop.go:233) stopped emitting. atlas/define.md:1215 is now false:
  "everything else is a rune for the form to grade" - d and D are not. Neither InputDrop/OutcomeDrop,
  nor the session-output-through-crlfWriter rule, nor the check-cancellation-before-select rule appear
  in the atlas, though all three are this window's durable decisions. Both the atlas and README gates fire.
- **BR-16** [Important] `discarded-error-detail` play_loop.go:140 swallows the raw-mode re-entry error, leaving the terminal cooked and the session apparently frozen
  2nd in this family (with BR-10), so fix the RULE, not the instance. "if again, err := enterRaw(os.Stdin);
  err == nil" drops the error entirely: after a failed re-entry readKeys is line-buffered, so every
  keystroke appears to do nothing until Enter, and nothing reaches stderr. The rule: an error is acted on
  or reported, never dropped where it is available. Production sites in this window - play_loop.go:140
  (dropped entirely) and puretest.go:43,123 (exitErr.Stderr discarded). play_loop.go:184-188 is the
  correct shape and should be the model. Sweep both.
- **BR-17** [Minor] `io-not-injected` playSession re-enters raw mode on os.Stdin rather than the file runPlay was handed
  play_loop.go:140 - the one hard-wired IO reference in an otherwise fully injected loop (ARCH-PURE),
  and the reason the audio branch cannot be driven at all. Thread the *os.File through and the
  audio Done-when becomes testable.
- **BR-18** [Minor] `probabilistic-regression-test` The cancel-before-select fix is pinned only probabilistically - roughly 1 run in 4 reddens
  play_loop_test.go:218 - with the guard removed, ctx.Done() and a buffered key are both ready and
  select picks at random, so recording an event needs two coin flips to go the wrong way. A
  deterministic form (assert the first scripted key is still buffered on return) fails every time.
- **BR-19** [Minor] `duplicate-sentinel-spelling` anyTime spells time.Time{} as store.Word{}.FirstSeen, while reflect.go:250 writes the plain form
  play_loop.go:217 ties an "everything since the beginning" sentinel to an unrelated struct field's
  type. ARCH-DRY: one spelling for one fact.
- **BR-20** [Minor] `unrecorded-scope-decision` toInput reserves Enter, space, d and D from every form, qualifying the second-form Done-when, and no artifact records it
  2nd in this family (with BR-7), so fix the RULE, not the instance. The rule: a decision that
  constrains a future consumer is recorded in the Log or plan in the same round. Open enumeration is
  exactly two - form 2.1 having no skip key (BR-7), and the loop's reserved key set at
  play_loop.go:157-172. #7 hits the second when it picks its keys.

## Open findings

- **BR-2** [Important] `claim-without-failing-test` puretest has no tests, yet the issue Log and project entry both claim its negative cases are verified in the tree
- **BR-3** [Important] `two-signal-termination` Answering the last question sets Session.Done but returns OutcomeRecord, so the outcome alone never says the session ended
- **BR-4** [Important] `plan-table-contradicts-code` The Core concepts table places Verdict in session.go, claims play imports store, and omits the puretest package
- **BR-5** [Important] `signature-cannot-express-contract` Plan Task 2 Step 1 and Task 3 Step 1 still instruct the superseded skip contract, leaving PQ-3 open into M2
- **BR-6** [Minor] `test-name-contradicts-assertion` TestSkipAdvancesButRecordsNothing drives an ungraded key and asserts the session did NOT advance
- **BR-7** [Minor] `unrecorded-scope-decision` Form 2.1 has no skip key, so Verdict.Skipped is unreachable in production, and no artifact records the decision
- **BR-8** [Minor] `duplicated-guard-logic` skipForm duplicates a capability fakeForm already has
- **BR-9** [Minor] `overloaded-zero-value` Grade overloads Skipped as both a real verdict and the zero value returned with ok == false
- **BR-10** [Minor] `discarded-error-detail` puretest discards go list stderr, so a failed measurement fatals with only an exit status
- **BR-11** [Minor] `plan-checkboxes-not-ticked` Every Chunk 1 step in the durable plan is still unticked although M1 is complete
- **BR-12** [Important] `claim-without-failing-test` Two Done-when rows are ticked with no test: the audio block has measured zero coverage and -count is never exercised
- **BR-13** [Important] `dead-entry-path` The --play and -count flag wiring at main.go:369 has zero test coverage, the class #21 already shipped twice
- **BR-14** [Important] `mode-silently-ignores-argument` define --play sycophantic runs a full session and ignores the word, unlike --forget and --reflect
- **BR-15** [Important] `docs-not-updated-for-new-surface` The d drop key ships undocumented and both README and atlas describe a prompt line the code no longer prints
- **BR-16** [Important] `discarded-error-detail` play_loop.go:140 swallows the raw-mode re-entry error, leaving the terminal cooked and the session apparently frozen
- **BR-17** [Minor] `io-not-injected` playSession re-enters raw mode on os.Stdin rather than the file runPlay was handed
- **BR-18** [Minor] `probabilistic-regression-test` The cancel-before-select fix is pinned only probabilistically - roughly 1 run in 4 reddens
- **BR-19** [Minor] `duplicate-sentinel-spelling` anyTime spells time.Time{} as store.Word{}.FirstSeen, while reflect.go:250 writes the plain form
- **BR-20** [Minor] `unrecorded-scope-decision` toInput reserves Enter, space, d and D from every form, qualifying the second-form Done-when, and no artifact records it
