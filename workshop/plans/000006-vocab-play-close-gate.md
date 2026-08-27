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

## Open findings

- **BR-1** [Important] `signature-cannot-express-contract` CaptureReview(word, correct bool, opt) cannot represent Skipped, yet the plan tests the skip against it
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
