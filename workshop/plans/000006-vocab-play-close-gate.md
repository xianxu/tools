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
    - "n": 4
      timestamp: "2026-08-27T10:56:44-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: not-addressed
          note: go test still reports "cmd/define/puretest [no test files]"; issue:183 and define-learn.md:527 both still claim the negative cases are verified in the tree.
          round: 4
        - id: BR-3
          disposition: not-addressed
          note: session.go untouched since round 3 — no Outcome.SessionDone, no doc on Apply/Session.Done, OutcomeDone still says only "the session is over".
          round: 4
        - id: BR-4
          disposition: not-addressed
          note: plan:21/26/48 byte-identical; still no puretest, Input or Outcome rows, and Verdict is still filed under session.go.
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: plan:98 and plan:122 unchanged; the plan still instructs the superseded skip contract that M2 was executed from.
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: session_test.go:74 still named TestSkipAdvancesButRecordsNothing while asserting Index == 0.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: recall.go:33 unchanged; no Log or plan entry records that form 2.1 has no skip key.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: skipForm still at session_test.go:220 beside fakeForm.Grade('3') at :213.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: question.go:45 and the Verdict doc at :15-21 unchanged; the zero value is still silently Skipped.
          round: 4
        - id: BR-10
          disposition: addressed
          note: stderrOf added (puretest.go:126) and used at both go list sites; unpinned only because puretest still has no test file (BR-2).
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: grep -c '^- \[x\]' on the plan returns 0; both chunks are entirely unticked and there is still no "## Revisions" section.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: playRig still hard-codes noAudio true and count 20; I verified both rows are pinnable today with the existing fakeCDN/fakePlayer seams.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: run() is now driven with "--play sycophantic", but that returns 2 at the usage switch and never reaches the dispatch, withStore or count threading.
          round: 4
        - id: BR-14
          disposition: addressed
          note: Mutation-verified twice in a scratch copy — removing the case reddens the test, and so does moving the dispatch back above the switch.
          round: 4
        - id: BR-15
          disposition: addressed
          note: README gains a key table naming d and -count and its transcript now matches draw()'s actual line; atlas records InputDrop, crlfWriter and cancel-before-select.
          round: 4
        - id: BR-16
          disposition: addressed
          note: Swept as the class — play_loop.go:141-148 reports and ends, puretest uses stderrOf at both sites; NOT pinned by a test, which the new claim-without-failing-test finding names.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: play_loop.go:140 still calls enterRaw(os.Stdin); this is now also the sole blocker on pinning BR-16's fix.
          round: 4
        - id: BR-18
          disposition: not-addressed
          note: play_loop_test.go:218 unchanged; two coin flips still have to land wrong for the missing guard to redden it.
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: play_loop.go:224 still spells the sentinel store.Word{}.FirstSeen.
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: 'The atlas now describes the key mapping, but no artifact records it as a CONSTRAINT on the forms in #7/#12/#13, and BR-7''s sibling is still unrecorded.'
          round: 4
      findings:
        - id: BR-21
          severity: Important
          title: The rule from round 3 was written down and then broken by the commit that closed round 3 — four unpinned claims remain and I measured that three are pinnable today
          detail: |-
            This is the 3rd finding in family claim-without-failing-test (BR-2, BR-12). Do NOT fix this
            instance — the rule is: a claim in an artifact (a Done-when tick, a Log sentence, a fix
            note) is complete only when a named test in the tree fails without it. The enumeration,
            walked in full and measured, is five sites. (1) puretest's three guards, zero tests, both
            the issue Log and the project entry still assert otherwise. (2) the audio Done-when row.
            (3) the -count Done-when row. (4) the --play dispatch through run(). (5) the raw-mode
            re-entry report added by BR-16's own fix. I built a scratch copy and proved 1-4 need no
            structural change: swapping playRig's noAudioSource for the existing newAudioRig fake made
            "audio plays before reveal" pass with player calls = 1 and raw == nil, and a 3-word deck
            with count 2 offered exactly 2 questions. Only site 5 needs a change, and it is exactly
            BR-17 — enterRaw is not injected, so the failure cannot be simulated. Fix the rule: write
            the row-to-test map into the issue, close 1-4 in this round, and either inject enterRaw or
            state in the artifact that site 5 is verified by reading.
          family: claim-without-failing-test
          round: 4
        - id: BR-22
          severity: Important
          title: The M1 row is ticked although the M1 boundary review blocked with four Importants that are still open, and no verdict trailer or close line exists for it
          detail: |-
            This is the 2nd finding in family plan-checkboxes-not-ticked (BR-11). Do NOT fix this
            instance — state the rule: a checkbox in a tracker artifact asserts an EVENT, and it is
            ticked only when the evidence for that event exists. BR-11 is the same rule with the sign
            flipped (work done, box unticked). Measured: issue:69 reads "- [x] M1", but the gate ledger
            records round 2 (boundary M1) as blocked:true with BR-2/3/4/5 raised; git log over the whole
            window carries exactly one Review-Verdict trailer (REWORK, on HEAD); and grep for "closed
            M1" in the issue returns nothing. AGENTS.md 3 says an Mx row commits to its own
            milestone-close producing both. So M2 was built on top of a boundary that never cleared,
            which is how four M1 findings reached round 4 alive.
          family: plan-checkboxes-not-ticked
          round: 4
        - id: BR-23
          severity: Minor
          title: main.go:417 opens the store a second time on the --play path — measured at 2 calls, against 1 on every other path
          detail: |-
            Moving the dispatch below the switch to fix BR-14 left the branch's own
            d = d.withStore(opt, stderr) in place, three lines below the unconditional one at main.go:408.
            I probed it with a counting newStore: --play calls it twice, a plain lookup once. withStore's
            nil guards discard the second storeDeps, so nothing is corrupted, but openStore re-runs Getwd
            and builds a second store.NewYAML plus a second vocabulary set — and on the Getwd-failure path
            it prints "define: no working directory" twice. Delete line 417. Note the entry-path test
            BR-13 asks for is exactly what caught this.
          family: duplicate-initialisation
          round: 4
        - id: BR-24
          severity: Minor
          title: Losing the terminal mid-session reports to stderr and exits 0, while failing to enter raw mode at the start exits 1
          detail: |-
            play_loop.go:146-147 prints "define: lost the terminal after playback" and then returns
            finish(), which is always 0; play_loop.go:57-59 returns 1 for the same class of failure one
            screen earlier. Defensible (the session ran and its events are recorded) but the two should
            agree or the difference should be stated in the comment that already explains why ending is
            honest.
          family: inconsistent-exit-status
          round: 4
        - id: BR-25
          severity: Minor
          title: A deck word the dictionary no longer knows has already spent a -count slot before it is skipped, and that branch has no test
          detail: |-
            play_loop.go:204-211 asks schedule.Queue for opt.count keys and then drops any the dictionary
            cannot resolve, so a learner with three stale entries and -count 20 gets a 17-word sitting
            with no explanation beyond a stderr line. I hit this accidentally while probing -count: three
            deck words, budget 2, one question. Either re-fill from the queue or say in the comment that
            a stale entry costs a slot. No test enters the branch today.
          family: budget-counted-before-filter
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-08-27T11:44:19-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: 'Verified by revert in a scratch copy: gutting ImportsOnly''s t.Errorf reddens TestImportsOnlyRejectsAnIOImport.'
          round: 5
        - id: BR-3
          disposition: addressed
          note: 'Verified by revert: dropping SessionDone from advance reddens TestTheOutcomeThatEndsTheSessionSaysSo.'
          round: 5
        - id: BR-4
          disposition: addressed
          note: Plan table corrected at 6442c6a - Verdict at question.go, Input/Outcome rows, puretest row, "imports NOTHING".
          round: 5
        - id: BR-5
          disposition: not-addressed
          note: plan:102 still says a skip emits a record outcome and the last answer emits quit; plan:126 still says the loop drops it.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: session_test.go:74 name unchanged; a clarifying comment was added but the name still contradicts the assertion.
          round: 5
        - id: BR-7
          disposition: not-addressed
          note: No record of the no-skip-key decision in the issue, plan or atlas.
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: skipForm still at session_test.go:220 beside fakeForm.Grade('3') at :214.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: question.go:15-20 still says nothing about Skipped being the zero value.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: Every checkbox in the durable plan, both chunks, is still "- [ ]".
          round: 5
        - id: BR-12
          disposition: not-addressed
          note: 'Measured: play_loop.go:131.37,151 at coverage 0; playRig still noAudio true + noAudioSource; -count still unexercised.'
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: 'Measured: main.go:416.15,419.3 at coverage 0; only the usage-error path enters run() with --play.'
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: play_loop.go:140 still re-enters raw mode on os.Stdin rather than the file runPlay was handed.
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: play_loop_test.go:218 unchanged; still reddens probabilistically with the guard removed.
          round: 5
        - id: BR-19
          disposition: not-addressed
          note: play_loop.go:224 still spells the zero time as store.Word{}.FirstSeen.
          round: 5
        - id: BR-20
          disposition: not-addressed
          note: No artifact records that toInput reserves Enter, space, d and D from every form.
          round: 5
        - id: BR-21
          disposition: not-addressed
          note: Rule written to lessons.md, enumeration unswept - site 1 closed, sites 2-5 unchanged, no row-to-test map in the issue.
          round: 5
        - id: BR-22
          disposition: not-addressed
          note: Rule stated in lessons.md and issue Revisions, but issue:69 is still "- [x] M1" with no Review-Verdict trailer and no "closed M1" log line.
          round: 5
        - id: BR-23
          disposition: not-addressed
          note: main.go:417 still calls withStore a second time, nine lines below the unconditional call at :408.
          round: 5
        - id: BR-24
          disposition: not-addressed
          note: play_loop.go:146 returns 0, play_loop.go:58 returns 1, for the same class of failure.
          round: 5
        - id: BR-25
          disposition: not-addressed
          note: play_loop.go:204-211 unchanged; the branch is still uncovered and the fall-through prints a misleading "nothing due today".
          round: 5
      findings:
        - id: BR-26
          severity: Important
          title: Mode flags are guarded against a word but not against each other, and "define -raw --play" runs a full session that records nothing
          detail: |-
            2nd in this family (with BR-14), so fix the RULE, not the instance. The rule: when two flags on one
            line cannot both be honoured, the binary refuses with a usage error rather than silently picking one.
            I walked the enumeration over the current flag surface. Guarded: --play+word, --reflect+word,
            -forget+word. NOT guarded: --play+-raw, --play+--reflect (main.go:416 wins), --play+-forget
            (main.go:410 wins), --reflect+-forget, --llm-check+anything (main.go:362 returns first). The -raw
            pairing has teeth - openStore (main.go:188) branches only on noCapture, so with -raw the deck is
            non-nil, runPlay's deck==nil guard at play_loop.go:26 does not fire, a full session runs, and every
            CaptureReview returns at decideCapture(...)==captureNothing (capture.go:124) with no message. The
            learner reviews twenty words, sees the tally, and nothing reaches disk. plan:80 enumerated two
            producers of "there is no deck to review" and missed this third route to the same end state
            (ARCH-PURPOSE). No test covers any mode-plus-mode combination.
          family: mode-silently-ignores-argument
          round: 5
        - id: BR-27
          severity: Minor
          title: The durable plan was rewritten in place at 6442c6a with no "## Revisions" section
          detail: |-
            git diff d008145 HEAD on workshop/plans/000006-vocab-play-plan.md shows two substantive hunks - the
            Core concepts table and the Test-surface paragraph - both overwriting prior text. AGENTS.md section 1
            requires a plan artifact revised mid-stream to append a Revisions entry (timestamp + reason + delta)
            rather than overwrite. The issue file received its Revisions entries this round; the plan did not, so
            the record of what the table used to claim survives only in the boundary ledger.
          family: artifact-revised-without-revision-entry
          round: 5
        - id: BR-28
          severity: Minor
          title: README's new --play section never mentions that the pronunciation plays on every reveal, or that -no-audio applies to a session
          detail: |-
            2nd in this family (with BR-15), so fix the RULE, not the instance. The rule: every user-observable
            behaviour a window introduces is described where the user would look for it, in the same round. I
            walked this window's user-observable surface - --play (README:41), the key table (:54-59), -count
            (:62), the no-deck line, the nothing-due line are all documented; audio-during-a-session is the one
            that is not. README:35 documents -no-audio as a lookup flag only, and the --play section's closing
            line ("No key and no network") reads as if a session is silent. This is the Spec's default-on
            behaviour and the same row BR-12 measures as having zero test coverage.
          family: docs-not-updated-for-new-surface
          round: 5
      blocked: true
    - "n": 6
      timestamp: "2026-08-27T12:08:51-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: 'Revert-verified in a scratch copy at HEAD: gutting the Errorf in ImportsOnly, NoWallClock and StoreSymbolsOnly each reddens its own named test; issue Log and project entry both now true.'
          round: 6
        - id: BR-3
          disposition: addressed
          note: 'Revert-verified: dropping SessionDone from advance''s OutcomeRecord return reddens TestTheOutcomeThatEndsTheSessionSaysSo.'
          round: 6
        - id: BR-4
          disposition: addressed
          note: Table corrected on all four counts; but the same store-SYMBOL claim survives in question.go, purity_test.go and the atlas — raised as a new finding in this family.
          round: 6
        - id: BR-5
          disposition: not-addressed
          note: Task 2 Step 1 fixed; plan:132 still reads "the loop drops it", contradicting plan:76 and session.go:170.
          round: 6
        - id: BR-6
          disposition: addressed
          note: Renamed to TestAnUngradedKeyDoesNotAdvance; the doc comment above it at session_test.go:72 still describes skip semantics — fold into the same edit.
          round: 6
        - id: BR-7
          disposition: not-addressed
          note: No artifact records that form 2.1 offers no skip key; question.go:24 documents only the zero value.
          round: 6
        - id: BR-8
          disposition: addressed
          note: skipForm deleted; TestSkippedVerdictRecordsNothing drives fakeForm's '3' and still reddens when the skip filter is mutated.
          round: 6
        - id: BR-9
          disposition: addressed
          note: question.go:24-27 now states Skipped is deliberately the zero value and why.
          round: 6
        - id: BR-10
          disposition: addressed
          note: stderrOf carries ExitError.Stderr at both sites; revert-verified — disabling it reddens TestGuardFailureNamesTheUnderlyingError.
          round: 6
        - id: BR-11
          disposition: addressed
          note: All Chunk 1 and Chunk 2 checkboxes ticked.
          round: 6
      findings:
        - id: BR-29
          severity: Important
          title: play runs TWO purity guards, but question.go, purity_test.go and the atlas all still claim three
          detail: |-
            2nd in this family (BR-4 was the plan's copy of the same sentence). Do NOT fix the
            instance. Live sites: cmd/define/play/question.go:7-8 "Three guards enforce it — an
            import allowlist, a wall-clock grep, and a store-SYMBOL allowlist"; purity_test.go:11
            "Same three guards as schedule", contradicted two lines later by its own "the store
            guard is deliberately absent"; atlas/define.md:1179 "#6 needs the same three".
            purity_test.go:19-26 runs ImportsOnly and NoWallClock only. The rule: a claim about
            what is ENFORCED is checked against the enforcing code before it is written, and the
            fix sweeps the whole greppable enumeration rather than the site the finding named
            (ARCH-PURPOSE). grep -rn "three guards|store-SYMBOL" returns exactly these three plus
            schedule/purity_test.go:11, where the claim is true.
          family: plan-table-contradicts-code
          round: 6
        - id: BR-30
          severity: Important
          title: Four of the five code fixes in the sweep commit land in branches measured at coverage 0, and two are pinnable today
          detail: |-
            4th in this family. Do NOT fix the instances. Measured with go test -coverprofile:
            play_loop.go:146-166 (BR-17's rawTerm.f and BR-24's exit 1) coverage 0 — every
            playSession test passes rawTerm{}, so raw.sess is nil and that whole block is entered
            by nothing; play_loop.go:234-240 (BR-25's message and exit 1) coverage 0;
            main.go:416-422 (BR-23's duplicate withStore removal) coverage 0; store/yaml.go:158
            not practically forceable. The rule is already in lessons.md; what is missing is the
            operable step — before closing a round, run coverage over the CHANGED lines and list
            every fix landing in a zero-coverage branch as either pinned-this-round or explicitly
            verified-by-reading in the artifact. BR-21 asked for this enumeration and received a
            prose table of err == nil sites instead of a coverage sweep, which is why the family
            recurs. BR-25's is pinnable now (a deck whose only due word the fake dictionary lacks,
            the fixture TestCountBoundsTheSession's comment already names) and BR-23's is pinnable
            now (a counting newStore, which BR-23 itself used to measure the defect).
          family: claim-without-failing-test
          round: 6
        - id: BR-31
          severity: Minor
          title: The cancel-before-select test now claims determinism, and measurement contradicts it
          detail: |-
            2nd in this family (BR-18). Do NOT fix the instance. play_loop_test.go:218 says
            "DETERMINISTIC by construction" and "near-certain rather than occasional". With the
            pre-select guard removed I measured 8/24 runs red using the new four-key buffer and
            7/24 using the original single key — statistically identical, because recording a
            review still needs two consecutive keys wins in the select regardless of buffer depth.
            The rule: a regression test for a nondeterministic bug removes the nondeterminism
            (repeat the scenario N times in-test, or extract the stop decision into a pure
            function and assert it), and no comment asserts a property a measurement refutes.
          family: probabilistic-regression-test
          round: 6
        - id: BR-32
          severity: Minor
          title: The atlas does not record Outcome.SessionDone, the puretest.T seam, or the testdata fixture convention this window introduced
          detail: |-
            3rd in this family. Do NOT fix the instance. The rule: the atlas entry for a surface
            is edited in the same round the surface changes, and "new surface" includes exported
            fields and test-fixture conventions, not only user-typed commands. This window's
            enumeration of new architectural surface absent from atlas/define.md: Outcome
            .SessionDone (new exported field, justified as the affordance downstream forms
            consume), puretest.T plus the testdata known-bad-fixture obligation that a form
            package adding a guard must follow, and the new all-lookups-fail exit-1 path.
          family: docs-not-updated-for-new-surface
          round: 6
        - id: BR-33
          severity: Minor
          title: puretest's known-GOOD fixture is the live play package, so an unrelated change to play reddens puretest's own suite
          detail: |-
            puretest_test.go:57 points TestImportsOnlyAcceptsAPurePackage and
            TestNoWallClockAcceptsAPurePackage at cmd/define/play with an empty allowlist. If #7
            gives play a legitimate import, puretest's suite fails with a message about the wrong
            package. testdata/ already holds the known-bad fixtures; a five-line testdata/pure
            makes the good one owned and portable too (ARCH-MOCK's portable-fixture clause).
          family: production-code-used-as-test-fixture
          round: 6
        - id: BR-34
          severity: Minor
          title: 2a2c837 "wip" carries 520 lines of M2 work with no issue reference, no body and no Co-Authored-By trailer
          detail: |-
            AGENTS.md section 12 makes git log --grep "^#6" the retrieval path for an issue's
            history, and this commit is invisible to it. It touches main.go, play_loop.go,
            play_loop_test.go and both gate ledgers — the largest single hunk of the window.
          family: commit-not-attributable-to-issue
          round: 6
      boundary: M1
      blocked: true
    - "n": 7
      timestamp: "2026-08-27T12:27:09-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: plan:105 now reads "a SKIP emits NO record outcome" and plan:130 replaces "the loop drops it" with "Apply emits no record outcome for one"; no residue found by grep.
          round: 7
        - id: BR-7
          disposition: not-addressed
          note: question.go:30-33 documents why Skipped is the zero value (BR-9's ask), but no artifact records that no shipped form produces Skipped with ok==true; grep of plan/issue/atlas returns nothing.
          round: 7
        - id: BR-29
          disposition: not-addressed
          note: Only question.go was corrected. play/purity_test.go:11 still says "Same three guards as schedule"; atlas:1179 still says "#6 needs the same three" three lines above its own correction at 1187; and schedule/box.go:12 says "TWO guards enforce it" for a package whose purity_test runs three.
          round: 7
        - id: BR-30
          disposition: addressed
          note: Verified by revert in a scratch copy - both new tests redden. Coverage of the two branches is now count 1. Note the enumeration omits store/yaml.go:163-165, also measured count 0.
          round: 7
        - id: BR-31
          disposition: not-addressed
          note: Measured in-process, N=400, pre-select guard removed - 119/400 red (~30%), matching the theoretical 0.25 for two consecutive select wins. play_loop_test.go:219 still says "DETERMINISTIC by construction" and :232 still says "near-certain rather than occasional".
          round: 7
        - id: BR-32
          disposition: not-addressed
          note: grep of atlas/define.md returns no SessionDone, no puretest.T, no testdata known-bad-fixture convention and no all-lookups-fail exit-1 path. The window's atlas edit covers only the guard count and the reserved keys.
          round: 7
        - id: BR-33
          disposition: not-addressed
          note: puretest_test.go:57 still points the known-good fixtures at cmd/define/play with an empty allowlist.
          round: 7
        - id: BR-34
          disposition: not-addressed
          note: A fresh instance landed in this window - 4cf2616 "wip", 507 lines including question.go, play_loop_test.go and the atlas, with no issue reference, no body and no Co-Authored-By trailer.
          round: 7
      findings:
        - id: BR-35
          severity: Important
          title: The plan files puretest under "Pure entities" but it execs `go list` and reads the filesystem
          detail: |-
            This is the 3rd finding in family `plan-table-contradicts-code`. Earlier rounds fixed
            instances. Do NOT fix this instance. The rule: a Core-concepts row's KIND is
            determined by reading the entity's imports, not by where the entity conceptually
            belongs - and the check is mechanical, so it applies to every row at once.
            Measured prevalence: plan:24 lists `puretest` guards under "Pure entities (the
            conceptual core)"; puretest.go:148 and :168 run `exec.Command("go", "list", ...)`
            and `os.ReadFile`, and puretest_test.go drives the real toolchain against real
            packages. The row was ADDED by BR-4's fix, so a fix in this family created a new
            member of it. ARCH-PURE at-review, and ARCH-MOCK: the `go` binary is an external
            dependency consumed with no named seam and no fake.
          family: plan-table-contradicts-code
          round: 7
        - id: BR-36
          severity: Important
          title: The issue Log's round-6 entry silently lost seven inline code spans, making BR-30's own accounting unreadable
          detail: |-
            workshop/issues/000006-vocab-play.md:340-358. Raw text reads "The first attempt
            used , which FAILS the test when consulted", "handing the session a  whose file is
            , so the re-entry genuinely fails", "the duplicate  is idempotent by
            construction", "BR-29:  and the atlas both claimed THREE purity guards where \n
            runs two", and "the decision moved to  - the same two-places-for-one-rule". Line
            341 is also a dangling indented fragment. Seven identifiers are gone. This is the
            artifact BR-30 asked to carry the pinned-vs-verified-by-reading enumeration, so
            the record of the disposition is itself unreadable. The rule: an artifact edit is
            re-read from the file after writing - a write that drops content is not a record,
            and the check is one `sed -n` away.
          family: unreadable-artifact-edit
          round: 7
        - id: BR-37
          severity: Important
          title: The durable plan was substantively revised this window with no `## Revisions` entry
          detail: |-
            This is the 2nd finding in family `artifact-revised-without-revision-entry`.
            Earlier rounds fixed instances. Do NOT fix this instance. The rule (AGENTS.md
            section 1): a plan artifact revised mid-stream gets an appended `## Revisions`
            entry - timestamp, reason, delta - and the trigger is "the diff touched the
            artifact", not "the change felt large". Measured: `grep -n "^## "` on
            workshop/plans/000006-vocab-play-plan.md returns Core concepts / Chunk 1 / Chunk 2
            / Risks and no Revisions section at all, while the window changed 64 lines
            including the Core-concepts table, the "play imports NOTHING" reversal, the
            two-guards paragraph, the reserved-keys paragraph and both skip-contract steps.
            The issue file has a `## Revisions` section covering the same period; the plan
            does not.
          family: artifact-revised-without-revision-entry
          round: 7
        - id: BR-38
          severity: Important
          title: The plan ticks its two gate-invocation steps and the issue ticks M2, though neither gate has produced a verdict trailer or close line
          detail: |-
            This is the 3rd finding in family `plan-checkboxes-not-ticked`. Earlier rounds
            fixed instances. Do NOT fix this instance. The rule: a checkbox asserts an EVENT
            happened, and for a gate step the event has a checkable artifact - so a `- [x]`
            on a `sdlc milestone-close` / `sdlc close` row is only writable after that
            command's `Review-Verdict:` trailer and `## Log` close line exist. Measured:
            `git log main..HEAD --format='%(trailers:key=Review-Verdict)'` is empty for all 20
            branch commits; `grep "closed M1\|closed M2"` on the issue returns nothing; the
            issue frontmatter is still `status: working`. Yet plan:120 is `- [x] Step 9:
            sdlc milestone-close --issue 6 --milestone M1`, plan:163 is `- [x] Step 11: sdlc
            close --issue 6`, and issue:69-70 tick both M1 and M2. BR-11's fix ticked every
            plan checkbox including the two gate rows, which is how fixing one member of this
            family created another.
          family: plan-checkboxes-not-ticked
          round: 7
        - id: BR-39
          severity: Minor
          title: playSession's doc comment is now attached to `type rawTerm`, leaving playSession undocumented
          detail: |-
            cmd/define/play_loop.go:74-90. Inserting rawTerm between playSession's comment and
            its declaration merged the two comment blocks, so godoc renders rawTerm as
            "playSession drives the state machine and performs its outcomes. Split from
            runPlay so a test can drive a whole session ... rawTerm is the terminal a session
            borrows ...". Move the playSession block back down to sit immediately above
            `func playSession`.
          family: comment-detached-from-its-declaration
          round: 7
        - id: BR-40
          severity: Minor
          title: TestAnUngradedKeyDoesNotAdvance kept the skip-rule doc comment from the name it replaced
          detail: |-
            This is the 2nd finding in family `test-name-contradicts-assertion`. Earlier
            rounds fixed instances. Do NOT fix this instance. The rule: a test's NAME and its
            DOC COMMENT are both claims about what the body asserts, so renaming a test means
            re-reading its comment in the same edit. Measured: session_test.go:72-73 still
            reads "A skip is not an assessment: schedule.Fold would read a recorded skip as a
            miss and demote a word the learner was honest about" above a test whose body
            (:75-84) asserts that an UNGRADED key does not advance - the skip claim it
            describes is now tested two functions down in TestSkippedVerdictRecordsNothing.
            BR-6's fix renamed the function and left the comment.
          family: test-name-contradicts-assertion
          round: 7
      boundary: M1
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

## Round 4 — 2026-08-27T10:56:44-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — not-addressed — go test still reports "cmd/define/puretest [no test files]"; issue:183 and define-learn.md:527 both still claim the negative cases are verified in the tree.
- BR-3 — not-addressed — session.go untouched since round 3 — no Outcome.SessionDone, no doc on Apply/Session.Done, OutcomeDone still says only "the session is over".
- BR-4 — not-addressed — plan:21/26/48 byte-identical; still no puretest, Input or Outcome rows, and Verdict is still filed under session.go.
- BR-5 — not-addressed — plan:98 and plan:122 unchanged; the plan still instructs the superseded skip contract that M2 was executed from.
- BR-6 — not-addressed — session_test.go:74 still named TestSkipAdvancesButRecordsNothing while asserting Index == 0.
- BR-7 — not-addressed — recall.go:33 unchanged; no Log or plan entry records that form 2.1 has no skip key.
- BR-8 — not-addressed — skipForm still at session_test.go:220 beside fakeForm.Grade('3') at :213.
- BR-9 — not-addressed — question.go:45 and the Verdict doc at :15-21 unchanged; the zero value is still silently Skipped.
- BR-10 — addressed — stderrOf added (puretest.go:126) and used at both go list sites; unpinned only because puretest still has no test file (BR-2).
- BR-11 — not-addressed — grep -c '^- \[x\]' on the plan returns 0; both chunks are entirely unticked and there is still no "## Revisions" section.
- BR-12 — not-addressed — playRig still hard-codes noAudio true and count 20; I verified both rows are pinnable today with the existing fakeCDN/fakePlayer seams.
- BR-13 — not-addressed — run() is now driven with "--play sycophantic", but that returns 2 at the usage switch and never reaches the dispatch, withStore or count threading.
- BR-14 — addressed — Mutation-verified twice in a scratch copy — removing the case reddens the test, and so does moving the dispatch back above the switch.
- BR-15 — addressed — README gains a key table naming d and -count and its transcript now matches draw()'s actual line; atlas records InputDrop, crlfWriter and cancel-before-select.
- BR-16 — addressed — Swept as the class — play_loop.go:141-148 reports and ends, puretest uses stderrOf at both sites; NOT pinned by a test, which the new claim-without-failing-test finding names.
- BR-17 — not-addressed — play_loop.go:140 still calls enterRaw(os.Stdin); this is now also the sole blocker on pinning BR-16's fix.
- BR-18 — not-addressed — play_loop_test.go:218 unchanged; two coin flips still have to land wrong for the missing guard to redden it.
- BR-19 — not-addressed — play_loop.go:224 still spells the sentinel store.Word{}.FirstSeen.
- BR-20 — not-addressed — The atlas now describes the key mapping, but no artifact records it as a CONSTRAINT on the forms in #7/#12/#13, and BR-7's sibling is still unrecorded.

### Raised

- **BR-21** [Important] `claim-without-failing-test` The rule from round 3 was written down and then broken by the commit that closed round 3 — four unpinned claims remain and I measured that three are pinnable today
  This is the 3rd finding in family claim-without-failing-test (BR-2, BR-12). Do NOT fix this
  instance — the rule is: a claim in an artifact (a Done-when tick, a Log sentence, a fix
  note) is complete only when a named test in the tree fails without it. The enumeration,
  walked in full and measured, is five sites. (1) puretest's three guards, zero tests, both
  the issue Log and the project entry still assert otherwise. (2) the audio Done-when row.
  (3) the -count Done-when row. (4) the --play dispatch through run(). (5) the raw-mode
  re-entry report added by BR-16's own fix. I built a scratch copy and proved 1-4 need no
  structural change: swapping playRig's noAudioSource for the existing newAudioRig fake made
  "audio plays before reveal" pass with player calls = 1 and raw == nil, and a 3-word deck
  with count 2 offered exactly 2 questions. Only site 5 needs a change, and it is exactly
  BR-17 — enterRaw is not injected, so the failure cannot be simulated. Fix the rule: write
  the row-to-test map into the issue, close 1-4 in this round, and either inject enterRaw or
  state in the artifact that site 5 is verified by reading.
- **BR-22** [Important] `plan-checkboxes-not-ticked` The M1 row is ticked although the M1 boundary review blocked with four Importants that are still open, and no verdict trailer or close line exists for it
  This is the 2nd finding in family plan-checkboxes-not-ticked (BR-11). Do NOT fix this
  instance — state the rule: a checkbox in a tracker artifact asserts an EVENT, and it is
  ticked only when the evidence for that event exists. BR-11 is the same rule with the sign
  flipped (work done, box unticked). Measured: issue:69 reads "- [x] M1", but the gate ledger
  records round 2 (boundary M1) as blocked:true with BR-2/3/4/5 raised; git log over the whole
  window carries exactly one Review-Verdict trailer (REWORK, on HEAD); and grep for "closed
  M1" in the issue returns nothing. AGENTS.md 3 says an Mx row commits to its own
  milestone-close producing both. So M2 was built on top of a boundary that never cleared,
  which is how four M1 findings reached round 4 alive.
- **BR-23** [Minor] `duplicate-initialisation` main.go:417 opens the store a second time on the --play path — measured at 2 calls, against 1 on every other path
  Moving the dispatch below the switch to fix BR-14 left the branch's own
  d = d.withStore(opt, stderr) in place, three lines below the unconditional one at main.go:408.
  I probed it with a counting newStore: --play calls it twice, a plain lookup once. withStore's
  nil guards discard the second storeDeps, so nothing is corrupted, but openStore re-runs Getwd
  and builds a second store.NewYAML plus a second vocabulary set — and on the Getwd-failure path
  it prints "define: no working directory" twice. Delete line 417. Note the entry-path test
  BR-13 asks for is exactly what caught this.
- **BR-24** [Minor] `inconsistent-exit-status` Losing the terminal mid-session reports to stderr and exits 0, while failing to enter raw mode at the start exits 1
  play_loop.go:146-147 prints "define: lost the terminal after playback" and then returns
  finish(), which is always 0; play_loop.go:57-59 returns 1 for the same class of failure one
  screen earlier. Defensible (the session ran and its events are recorded) but the two should
  agree or the difference should be stated in the comment that already explains why ending is
  honest.
- **BR-25** [Minor] `budget-counted-before-filter` A deck word the dictionary no longer knows has already spent a -count slot before it is skipped, and that branch has no test
  play_loop.go:204-211 asks schedule.Queue for opt.count keys and then drops any the dictionary
  cannot resolve, so a learner with three stale entries and -count 20 gets a 17-word sitting
  with no explanation beyond a stderr line. I hit this accidentally while probing -count: three
  deck words, budget 2, one question. Either re-fill from the queue or say in the comment that
  a stale entry costs a slot. No test enters the branch today.

## Round 5 — 2026-08-27T11:44:19-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — Verified by revert in a scratch copy: gutting ImportsOnly's t.Errorf reddens TestImportsOnlyRejectsAnIOImport.
- BR-3 — addressed — Verified by revert: dropping SessionDone from advance reddens TestTheOutcomeThatEndsTheSessionSaysSo.
- BR-4 — addressed — Plan table corrected at 6442c6a - Verdict at question.go, Input/Outcome rows, puretest row, "imports NOTHING".
- BR-5 — not-addressed — plan:102 still says a skip emits a record outcome and the last answer emits quit; plan:126 still says the loop drops it.
- BR-6 — not-addressed — session_test.go:74 name unchanged; a clarifying comment was added but the name still contradicts the assertion.
- BR-7 — not-addressed — No record of the no-skip-key decision in the issue, plan or atlas.
- BR-8 — not-addressed — skipForm still at session_test.go:220 beside fakeForm.Grade('3') at :214.
- BR-9 — not-addressed — question.go:15-20 still says nothing about Skipped being the zero value.
- BR-11 — not-addressed — Every checkbox in the durable plan, both chunks, is still "- [ ]".
- BR-12 — not-addressed — Measured: play_loop.go:131.37,151 at coverage 0; playRig still noAudio true + noAudioSource; -count still unexercised.
- BR-13 — not-addressed — Measured: main.go:416.15,419.3 at coverage 0; only the usage-error path enters run() with --play.
- BR-17 — not-addressed — play_loop.go:140 still re-enters raw mode on os.Stdin rather than the file runPlay was handed.
- BR-18 — not-addressed — play_loop_test.go:218 unchanged; still reddens probabilistically with the guard removed.
- BR-19 — not-addressed — play_loop.go:224 still spells the zero time as store.Word{}.FirstSeen.
- BR-20 — not-addressed — No artifact records that toInput reserves Enter, space, d and D from every form.
- BR-21 — not-addressed — Rule written to lessons.md, enumeration unswept - site 1 closed, sites 2-5 unchanged, no row-to-test map in the issue.
- BR-22 — not-addressed — Rule stated in lessons.md and issue Revisions, but issue:69 is still "- [x] M1" with no Review-Verdict trailer and no "closed M1" log line.
- BR-23 — not-addressed — main.go:417 still calls withStore a second time, nine lines below the unconditional call at :408.
- BR-24 — not-addressed — play_loop.go:146 returns 0, play_loop.go:58 returns 1, for the same class of failure.
- BR-25 — not-addressed — play_loop.go:204-211 unchanged; the branch is still uncovered and the fall-through prints a misleading "nothing due today".

### Raised

- **BR-26** [Important] `mode-silently-ignores-argument` Mode flags are guarded against a word but not against each other, and "define -raw --play" runs a full session that records nothing
  2nd in this family (with BR-14), so fix the RULE, not the instance. The rule: when two flags on one
  line cannot both be honoured, the binary refuses with a usage error rather than silently picking one.
  I walked the enumeration over the current flag surface. Guarded: --play+word, --reflect+word,
  -forget+word. NOT guarded: --play+-raw, --play+--reflect (main.go:416 wins), --play+-forget
  (main.go:410 wins), --reflect+-forget, --llm-check+anything (main.go:362 returns first). The -raw
  pairing has teeth - openStore (main.go:188) branches only on noCapture, so with -raw the deck is
  non-nil, runPlay's deck==nil guard at play_loop.go:26 does not fire, a full session runs, and every
  CaptureReview returns at decideCapture(...)==captureNothing (capture.go:124) with no message. The
  learner reviews twenty words, sees the tally, and nothing reaches disk. plan:80 enumerated two
  producers of "there is no deck to review" and missed this third route to the same end state
  (ARCH-PURPOSE). No test covers any mode-plus-mode combination.
- **BR-27** [Minor] `artifact-revised-without-revision-entry` The durable plan was rewritten in place at 6442c6a with no "## Revisions" section
  git diff d008145 HEAD on workshop/plans/000006-vocab-play-plan.md shows two substantive hunks - the
  Core concepts table and the Test-surface paragraph - both overwriting prior text. AGENTS.md section 1
  requires a plan artifact revised mid-stream to append a Revisions entry (timestamp + reason + delta)
  rather than overwrite. The issue file received its Revisions entries this round; the plan did not, so
  the record of what the table used to claim survives only in the boundary ledger.
- **BR-28** [Minor] `docs-not-updated-for-new-surface` README's new --play section never mentions that the pronunciation plays on every reveal, or that -no-audio applies to a session
  2nd in this family (with BR-15), so fix the RULE, not the instance. The rule: every user-observable
  behaviour a window introduces is described where the user would look for it, in the same round. I
  walked this window's user-observable surface - --play (README:41), the key table (:54-59), -count
  (:62), the no-deck line, the nothing-due line are all documented; audio-during-a-session is the one
  that is not. README:35 documents -no-audio as a lookup flag only, and the --play section's closing
  line ("No key and no network") reads as if a session is silent. This is the Spec's default-on
  behaviour and the same row BR-12 measures as having zero test coverage.

## Round 6 — 2026-08-27T12:08:51-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — Revert-verified in a scratch copy at HEAD: gutting the Errorf in ImportsOnly, NoWallClock and StoreSymbolsOnly each reddens its own named test; issue Log and project entry both now true.
- BR-3 — addressed — Revert-verified: dropping SessionDone from advance's OutcomeRecord return reddens TestTheOutcomeThatEndsTheSessionSaysSo.
- BR-4 — addressed — Table corrected on all four counts; but the same store-SYMBOL claim survives in question.go, purity_test.go and the atlas — raised as a new finding in this family.
- BR-5 — not-addressed — Task 2 Step 1 fixed; plan:132 still reads "the loop drops it", contradicting plan:76 and session.go:170.
- BR-6 — addressed — Renamed to TestAnUngradedKeyDoesNotAdvance; the doc comment above it at session_test.go:72 still describes skip semantics — fold into the same edit.
- BR-7 — not-addressed — No artifact records that form 2.1 offers no skip key; question.go:24 documents only the zero value.
- BR-8 — addressed — skipForm deleted; TestSkippedVerdictRecordsNothing drives fakeForm's '3' and still reddens when the skip filter is mutated.
- BR-9 — addressed — question.go:24-27 now states Skipped is deliberately the zero value and why.
- BR-10 — addressed — stderrOf carries ExitError.Stderr at both sites; revert-verified — disabling it reddens TestGuardFailureNamesTheUnderlyingError.
- BR-11 — addressed — All Chunk 1 and Chunk 2 checkboxes ticked.

### Raised

- **BR-29** [Important] `plan-table-contradicts-code` play runs TWO purity guards, but question.go, purity_test.go and the atlas all still claim three
  2nd in this family (BR-4 was the plan's copy of the same sentence). Do NOT fix the
  instance. Live sites: cmd/define/play/question.go:7-8 "Three guards enforce it — an
  import allowlist, a wall-clock grep, and a store-SYMBOL allowlist"; purity_test.go:11
  "Same three guards as schedule", contradicted two lines later by its own "the store
  guard is deliberately absent"; atlas/define.md:1179 "#6 needs the same three".
  purity_test.go:19-26 runs ImportsOnly and NoWallClock only. The rule: a claim about
  what is ENFORCED is checked against the enforcing code before it is written, and the
  fix sweeps the whole greppable enumeration rather than the site the finding named
  (ARCH-PURPOSE). grep -rn "three guards|store-SYMBOL" returns exactly these three plus
  schedule/purity_test.go:11, where the claim is true.
- **BR-30** [Important] `claim-without-failing-test` Four of the five code fixes in the sweep commit land in branches measured at coverage 0, and two are pinnable today
  4th in this family. Do NOT fix the instances. Measured with go test -coverprofile:
  play_loop.go:146-166 (BR-17's rawTerm.f and BR-24's exit 1) coverage 0 — every
  playSession test passes rawTerm{}, so raw.sess is nil and that whole block is entered
  by nothing; play_loop.go:234-240 (BR-25's message and exit 1) coverage 0;
  main.go:416-422 (BR-23's duplicate withStore removal) coverage 0; store/yaml.go:158
  not practically forceable. The rule is already in lessons.md; what is missing is the
  operable step — before closing a round, run coverage over the CHANGED lines and list
  every fix landing in a zero-coverage branch as either pinned-this-round or explicitly
  verified-by-reading in the artifact. BR-21 asked for this enumeration and received a
  prose table of err == nil sites instead of a coverage sweep, which is why the family
  recurs. BR-25's is pinnable now (a deck whose only due word the fake dictionary lacks,
  the fixture TestCountBoundsTheSession's comment already names) and BR-23's is pinnable
  now (a counting newStore, which BR-23 itself used to measure the defect).
- **BR-31** [Minor] `probabilistic-regression-test` The cancel-before-select test now claims determinism, and measurement contradicts it
  2nd in this family (BR-18). Do NOT fix the instance. play_loop_test.go:218 says
  "DETERMINISTIC by construction" and "near-certain rather than occasional". With the
  pre-select guard removed I measured 8/24 runs red using the new four-key buffer and
  7/24 using the original single key — statistically identical, because recording a
  review still needs two consecutive keys wins in the select regardless of buffer depth.
  The rule: a regression test for a nondeterministic bug removes the nondeterminism
  (repeat the scenario N times in-test, or extract the stop decision into a pure
  function and assert it), and no comment asserts a property a measurement refutes.
- **BR-32** [Minor] `docs-not-updated-for-new-surface` The atlas does not record Outcome.SessionDone, the puretest.T seam, or the testdata fixture convention this window introduced
  3rd in this family. Do NOT fix the instance. The rule: the atlas entry for a surface
  is edited in the same round the surface changes, and "new surface" includes exported
  fields and test-fixture conventions, not only user-typed commands. This window's
  enumeration of new architectural surface absent from atlas/define.md: Outcome
  .SessionDone (new exported field, justified as the affordance downstream forms
  consume), puretest.T plus the testdata known-bad-fixture obligation that a form
  package adding a guard must follow, and the new all-lookups-fail exit-1 path.
- **BR-33** [Minor] `production-code-used-as-test-fixture` puretest's known-GOOD fixture is the live play package, so an unrelated change to play reddens puretest's own suite
  puretest_test.go:57 points TestImportsOnlyAcceptsAPurePackage and
  TestNoWallClockAcceptsAPurePackage at cmd/define/play with an empty allowlist. If #7
  gives play a legitimate import, puretest's suite fails with a message about the wrong
  package. testdata/ already holds the known-bad fixtures; a five-line testdata/pure
  makes the good one owned and portable too (ARCH-MOCK's portable-fixture clause).
- **BR-34** [Minor] `commit-not-attributable-to-issue` 2a2c837 "wip" carries 520 lines of M2 work with no issue reference, no body and no Co-Authored-By trailer
  AGENTS.md section 12 makes git log --grep "^#6" the retrieval path for an issue's
  history, and this commit is invisible to it. It touches main.go, play_loop.go,
  play_loop_test.go and both gate ledgers — the largest single hunk of the window.

## Round 7 — 2026-08-27T12:27:09-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — addressed — plan:105 now reads "a SKIP emits NO record outcome" and plan:130 replaces "the loop drops it" with "Apply emits no record outcome for one"; no residue found by grep.
- BR-7 — not-addressed — question.go:30-33 documents why Skipped is the zero value (BR-9's ask), but no artifact records that no shipped form produces Skipped with ok==true; grep of plan/issue/atlas returns nothing.
- BR-29 — not-addressed — Only question.go was corrected. play/purity_test.go:11 still says "Same three guards as schedule"; atlas:1179 still says "#6 needs the same three" three lines above its own correction at 1187; and schedule/box.go:12 says "TWO guards enforce it" for a package whose purity_test runs three.
- BR-30 — addressed — Verified by revert in a scratch copy - both new tests redden. Coverage of the two branches is now count 1. Note the enumeration omits store/yaml.go:163-165, also measured count 0.
- BR-31 — not-addressed — Measured in-process, N=400, pre-select guard removed - 119/400 red (~30%), matching the theoretical 0.25 for two consecutive select wins. play_loop_test.go:219 still says "DETERMINISTIC by construction" and :232 still says "near-certain rather than occasional".
- BR-32 — not-addressed — grep of atlas/define.md returns no SessionDone, no puretest.T, no testdata known-bad-fixture convention and no all-lookups-fail exit-1 path. The window's atlas edit covers only the guard count and the reserved keys.
- BR-33 — not-addressed — puretest_test.go:57 still points the known-good fixtures at cmd/define/play with an empty allowlist.
- BR-34 — not-addressed — A fresh instance landed in this window - 4cf2616 "wip", 507 lines including question.go, play_loop_test.go and the atlas, with no issue reference, no body and no Co-Authored-By trailer.

### Raised

- **BR-35** [Important] `plan-table-contradicts-code` The plan files puretest under "Pure entities" but it execs `go list` and reads the filesystem
  This is the 3rd finding in family `plan-table-contradicts-code`. Earlier rounds fixed
  instances. Do NOT fix this instance. The rule: a Core-concepts row's KIND is
  determined by reading the entity's imports, not by where the entity conceptually
  belongs - and the check is mechanical, so it applies to every row at once.
  Measured prevalence: plan:24 lists `puretest` guards under "Pure entities (the
  conceptual core)"; puretest.go:148 and :168 run `exec.Command("go", "list", ...)`
  and `os.ReadFile`, and puretest_test.go drives the real toolchain against real
  packages. The row was ADDED by BR-4's fix, so a fix in this family created a new
  member of it. ARCH-PURE at-review, and ARCH-MOCK: the `go` binary is an external
  dependency consumed with no named seam and no fake.
- **BR-36** [Important] `unreadable-artifact-edit` The issue Log's round-6 entry silently lost seven inline code spans, making BR-30's own accounting unreadable
  workshop/issues/000006-vocab-play.md:340-358. Raw text reads "The first attempt
  used , which FAILS the test when consulted", "handing the session a  whose file is
  , so the re-entry genuinely fails", "the duplicate  is idempotent by
  construction", "BR-29:  and the atlas both claimed THREE purity guards where \n
  runs two", and "the decision moved to  - the same two-places-for-one-rule". Line
  341 is also a dangling indented fragment. Seven identifiers are gone. This is the
  artifact BR-30 asked to carry the pinned-vs-verified-by-reading enumeration, so
  the record of the disposition is itself unreadable. The rule: an artifact edit is
  re-read from the file after writing - a write that drops content is not a record,
  and the check is one `sed -n` away.
- **BR-37** [Important] `artifact-revised-without-revision-entry` The durable plan was substantively revised this window with no `## Revisions` entry
  This is the 2nd finding in family `artifact-revised-without-revision-entry`.
  Earlier rounds fixed instances. Do NOT fix this instance. The rule (AGENTS.md
  section 1): a plan artifact revised mid-stream gets an appended `## Revisions`
  entry - timestamp, reason, delta - and the trigger is "the diff touched the
  artifact", not "the change felt large". Measured: `grep -n "^## "` on
  workshop/plans/000006-vocab-play-plan.md returns Core concepts / Chunk 1 / Chunk 2
  / Risks and no Revisions section at all, while the window changed 64 lines
  including the Core-concepts table, the "play imports NOTHING" reversal, the
  two-guards paragraph, the reserved-keys paragraph and both skip-contract steps.
  The issue file has a `## Revisions` section covering the same period; the plan
  does not.
- **BR-38** [Important] `plan-checkboxes-not-ticked` The plan ticks its two gate-invocation steps and the issue ticks M2, though neither gate has produced a verdict trailer or close line
  This is the 3rd finding in family `plan-checkboxes-not-ticked`. Earlier rounds
  fixed instances. Do NOT fix this instance. The rule: a checkbox asserts an EVENT
  happened, and for a gate step the event has a checkable artifact - so a `- [x]`
  on a `sdlc milestone-close` / `sdlc close` row is only writable after that
  command's `Review-Verdict:` trailer and `## Log` close line exist. Measured:
  `git log main..HEAD --format='%(trailers:key=Review-Verdict)'` is empty for all 20
  branch commits; `grep "closed M1\|closed M2"` on the issue returns nothing; the
  issue frontmatter is still `status: working`. Yet plan:120 is `- [x] Step 9:
  sdlc milestone-close --issue 6 --milestone M1`, plan:163 is `- [x] Step 11: sdlc
  close --issue 6`, and issue:69-70 tick both M1 and M2. BR-11's fix ticked every
  plan checkbox including the two gate rows, which is how fixing one member of this
  family created another.
- **BR-39** [Minor] `comment-detached-from-its-declaration` playSession's doc comment is now attached to `type rawTerm`, leaving playSession undocumented
  cmd/define/play_loop.go:74-90. Inserting rawTerm between playSession's comment and
  its declaration merged the two comment blocks, so godoc renders rawTerm as
  "playSession drives the state machine and performs its outcomes. Split from
  runPlay so a test can drive a whole session ... rawTerm is the terminal a session
  borrows ...". Move the playSession block back down to sit immediately above
  `func playSession`.
- **BR-40** [Minor] `test-name-contradicts-assertion` TestAnUngradedKeyDoesNotAdvance kept the skip-rule doc comment from the name it replaced
  This is the 2nd finding in family `test-name-contradicts-assertion`. Earlier
  rounds fixed instances. Do NOT fix this instance. The rule: a test's NAME and its
  DOC COMMENT are both claims about what the body asserts, so renaming a test means
  re-reading its comment in the same edit. Measured: session_test.go:72-73 still
  reads "A skip is not an assessment: schedule.Fold would read a recorded skip as a
  miss and demote a word the learner was honest about" above a test whose body
  (:75-84) asserts that an UNGRADED key does not advance - the skip claim it
  describes is now tested two functions down in TestSkippedVerdictRecordsNothing.
  BR-6's fix renamed the function and left the comment.

## Open findings

- **BR-7** [Minor] `unrecorded-scope-decision` Form 2.1 has no skip key, so Verdict.Skipped is unreachable in production, and no artifact records the decision
- **BR-12** [Important] `claim-without-failing-test` Two Done-when rows are ticked with no test: the audio block has measured zero coverage and -count is never exercised
- **BR-13** [Important] `dead-entry-path` The --play and -count flag wiring at main.go:369 has zero test coverage, the class #21 already shipped twice
- **BR-17** [Minor] `io-not-injected` playSession re-enters raw mode on os.Stdin rather than the file runPlay was handed
- **BR-18** [Minor] `probabilistic-regression-test` The cancel-before-select fix is pinned only probabilistically - roughly 1 run in 4 reddens
- **BR-19** [Minor] `duplicate-sentinel-spelling` anyTime spells time.Time{} as store.Word{}.FirstSeen, while reflect.go:250 writes the plain form
- **BR-20** [Minor] `unrecorded-scope-decision` toInput reserves Enter, space, d and D from every form, qualifying the second-form Done-when, and no artifact records it
- **BR-21** [Important] `claim-without-failing-test` The rule from round 3 was written down and then broken by the commit that closed round 3 — four unpinned claims remain and I measured that three are pinnable today
- **BR-22** [Important] `plan-checkboxes-not-ticked` The M1 row is ticked although the M1 boundary review blocked with four Importants that are still open, and no verdict trailer or close line exists for it
- **BR-23** [Minor] `duplicate-initialisation` main.go:417 opens the store a second time on the --play path — measured at 2 calls, against 1 on every other path
- **BR-24** [Minor] `inconsistent-exit-status` Losing the terminal mid-session reports to stderr and exits 0, while failing to enter raw mode at the start exits 1
- **BR-25** [Minor] `budget-counted-before-filter` A deck word the dictionary no longer knows has already spent a -count slot before it is skipped, and that branch has no test
- **BR-26** [Important] `mode-silently-ignores-argument` Mode flags are guarded against a word but not against each other, and "define -raw --play" runs a full session that records nothing
- **BR-27** [Minor] `artifact-revised-without-revision-entry` The durable plan was rewritten in place at 6442c6a with no "## Revisions" section
- **BR-28** [Minor] `docs-not-updated-for-new-surface` README's new --play section never mentions that the pronunciation plays on every reveal, or that -no-audio applies to a session
- **BR-29** [Important] `plan-table-contradicts-code` play runs TWO purity guards, but question.go, purity_test.go and the atlas all still claim three
- **BR-31** [Minor] `probabilistic-regression-test` The cancel-before-select test now claims determinism, and measurement contradicts it
- **BR-32** [Minor] `docs-not-updated-for-new-surface` The atlas does not record Outcome.SessionDone, the puretest.T seam, or the testdata fixture convention this window introduced
- **BR-33** [Minor] `production-code-used-as-test-fixture` puretest's known-GOOD fixture is the live play package, so an unrelated change to play reddens puretest's own suite
- **BR-34** [Minor] `commit-not-attributable-to-issue` 2a2c837 "wip" carries 520 lines of M2 work with no issue reference, no body and no Co-Authored-By trailer
- **BR-35** [Important] `plan-table-contradicts-code` The plan files puretest under "Pure entities" but it execs `go list` and reads the filesystem
- **BR-36** [Important] `unreadable-artifact-edit` The issue Log's round-6 entry silently lost seven inline code spans, making BR-30's own accounting unreadable
- **BR-37** [Important] `artifact-revised-without-revision-entry` The durable plan was substantively revised this window with no `## Revisions` entry
- **BR-38** [Important] `plan-checkboxes-not-ticked` The plan ticks its two gate-invocation steps and the issue ticks M2, though neither gate has produced a verdict trailer or close line
- **BR-39** [Minor] `comment-detached-from-its-declaration` playSession's doc comment is now attached to `type rawTerm`, leaving playSession undocumented
- **BR-40** [Minor] `test-name-contradicts-assertion` TestAnUngradedKeyDoesNotAdvance kept the skip-rule doc comment from the name it replaced
