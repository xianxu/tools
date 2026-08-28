---
gate: boundary-review
issue: 25
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-27T21:10:33-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: TestDoesNotSkipWhenTheServiceAnswersWithAnError exercises a conformance-routed helper without controlling CONFORMANCE_STRICT
          detail: |-
            internal/llm/llmtest/reachable_test.go:38 calls SkipIfUnreachable on a substitute T
            while its two neighbours in the same file now neutralize the variable, so the
            Done-when's universal claim is false as written (ARCH-PURPOSE). It is invariant
            today only because a successful dial never reaches conformance.SkipOrFail; that is
            a property of current control flow, not of the test. Fix: add
            t.Setenv(conformance.StrictEnv, "") at the top, matching line 23.
          family: test-inherits-ambient-env
          round: 1
        - id: BR-2
          severity: Minor
          title: The Plan's ARCH-DRY reason for not extracting the substitute-T idiom does not hold for the same-package majority
          detail: |-
            The note says a shared helper would force golden_test.go to import
            internal/conformance, but golden_test.go and reachable_test.go are both
            package llmtest, so an unexported helper there covers 4 of the 5 goroutine+done
            sites with no new package and no conformance coupling; only skiporfail_test.go is
            genuinely cross-package. The note also repeats PQ-3's "eight sites" that the same
            issue's Done-when corrects to seven (and the full idiom is 5 sites, not 8).
            The decision may still be right; the recorded reason is not.
          family: decision-record-premise-wrong
          round: 1
        - id: BR-3
          severity: Minor
          title: Mutation red-counts recorded in the issue Log do not reproduce
          detail: |-
            The Log records "(4 red), (4), (7), (9)" for the four mutations; counting
            "--- FAIL" lines from go test -v gives 4 / 3 / 3 / 8, with no command or counting
            convention stated to reconcile them. The substantive claim (each mutation reds a
            named test) holds and I verified it. Record the named tests per row, which is what
            the Plan item promises, rather than a bare count.
          family: unreproducible-evidence-claim
          round: 1
        - id: BR-4
          severity: Minor
          title: The Failed() assertion in skiporfail_test.go is expressed inversely
          detail: |-
            internal/conformance/skiporfail_test.go:47 reads
            `if got := fake.Failed(); got == tc.wantSkip` and prints want as `!tc.wantSkip`.
            Correct, but a `wantFail := !tc.wantSkip` local would state the intent directly.
          family: assertion-readability
          round: 1
        - id: BR-5
          severity: Minor
          title: The conformance import in reachable_test.go sits inside the stdlib group
          detail: |-
            internal/llm/llmtest/reachable_test.go:4 places
            github.com/xianxu/tools/internal/conformance ahead of net/http in the same block.
            gofmt -l is clean and it matches reachable.go's existing style, but the new
            skiporfail_test.go groups correctly and goimports would split this one.
          family: import-grouping
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-27T21:26:18-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-08-27T21:47:04-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Fixed as the class via substituteT(t, strict, fn); verified by revert — removing t.Setenv reds TestStrictTurnsAnUnreachableServiceIntoAFailure (default) and TestSkipsOnlyWhenNothingIsListening (strict).
          round: 3
        - id: BR-2
          disposition: addressed
          note: Plan's ARCH-DRY paragraph reversed, false same-package premise removed; the residual wrong count in the correction is raised separately as an unreproducible-evidence-claim instance, not as BR-2.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Table now records named tests, command and baseline; I ran all four mutations at 05dee92 and every row reproduces exactly.
          round: 3
        - id: BR-4
          disposition: addressed
          note: wantFail := !tc.wantSkip at internal/conformance/skiporfail_test.go:44.
          round: 3
        - id: BR-5
          disposition: addressed
          note: The extraction removed the mis-grouped import from reachable_test.go entirely; substitute_test.go and wiring_test.go both group correctly.
          round: 3
      findings:
        - id: BR-6
          severity: Important
          title: wiring_test.go's default-mode row cannot fail for the mode it names — its want string is a prefix of the strict message
          detail: |-
            internal/conformance/wiring_test.go:45 asserts strings.Contains(out, "network
            unavailable: dial refused") for the default row, which is a substring of the strict
            message. Verified on clean head: mutating SkipOrFail to t.Skip(message(reason, err,
            true)) — so every offline skip falsely announces "(CONFORMANCE_STRICT is set)" —
            leaves internal/conformance, internal/llm/llmtest and go test ./... green in BOTH env
            states. The mirror mutation on the strict branch DOES red, so exactly one of the two
            directions is pinned at the call site while Done-when row 3 claims the wiring is
            pinned without qualification. The same weakness means the row cannot detect that its
            own CONFORMANCE_STRICT= override failed to beat an inherited =1. Fix: assert the
            default row does NOT contain conformance.StrictEnv+" is set", or narrow the Done-when
            row to the direction actually pinned (ARCH-PURPOSE).
          family: check-that-cannot-fail-reads-as-green
          round: 3
        - id: BR-7
          severity: Minor
          title: '"five" is wrong in four places, including the sentence that explains why counts do not belong in this issue'
          detail: |-
            This is the 3rd finding in family unreproducible-evidence-claim (BR-3 was the mutation
            red-counts). Do NOT patch five to six. Measured at ba321d1: package llmtest had SIX
            substitute-T sites (golden_test.go:28,35,49; reachable_test.go:26,47,70), FOUR of them
            goroutine sites; the fifth goroutine site is skiporfail_test.go, a different package.
            So issue lines 74 and 289 ("collapsed five package llmtest sites") are six, and lines
            119 and 267 ("all five goroutine sites in package llmtest") are four goroutine / six
            routed; line 126's cross-package five is correct. Measured prevalence four sites. The
            rule is already written verbatim in the Done-when — a count in a durable artifact is a
            restatement of a fact the code owns, and it drifts — and was applied at exactly one
            row. Rule-level fix: replace every remaining code-site count in the issue with the
            invariant plus its derivation grep, the way that row already does.
          family: unreproducible-evidence-claim
          round: 3
        - id: BR-8
          severity: Minor
          title: substituteT's doc says the mode is set "for the duration", but t.Setenv binds to the calling test, not to fn
          detail: |-
            This is the 2nd finding in family test-inherits-ambient-env (BR-1 was the missed
            reachable_test.go site). Do NOT special-case a caller. internal/llm/llmtest/substitute_test.go:35
            calls t.Setenv on the parent t, so the mode outlives fn and persists to the end of the
            test. Measured prevalence: 0 affected sites today — each caller uses one mode and every
            call re-sets the variable. The rule that covers it: a helper that sets an environment
            variable on a callee's behalf must state whose scope it restores at, so a caller mixing
            modes in one test body knows it needs a t.Run. Cheapest expression is the doc line at
            substitute_test.go:10 — "for the remainder of the calling test", not "for the duration".
          family: test-inherits-ambient-env
          round: 3
        - id: BR-9
          severity: Minor
          title: wiring_test.go discards CombinedOutput's error, so a spawn failure is reported as a wiring bug
          detail: |-
            internal/conformance/wiring_test.go:59 does out, _ := cmd.CombinedOutput(). The discard
            is right for the exit status (non-zero under strict, by design) but also swallows a
            genuine spawn failure; out is then empty and the assertion at :62 reports "SkipOrFail is
            not routing through message()" — a confident wrong cause for a reader debugging CI.
            Capture the error and include it in the t.Errorf alongside the transcript.
          family: swallowed-error-misattributes-cause
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-27T22:03:56-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: 'Verified by mutation, not by the commit message: the strict-suffix mutation now reddens the named default row in BOTH env states.'
          round: 4
        - id: BR-7
          disposition: addressed
          note: Code-site counts removed; surviving "five"s are lines-of-code and estimate rows. I ran both derivation greps and the invariant holds.
          round: 4
        - id: BR-8
          disposition: addressed
          note: substitute_test.go:8-14 now states the real t.Setenv scope and the two-subtest remedy.
          round: 4
        - id: BR-9
          disposition: addressed
          note: 'runErr captured and reachable at both assertions — it printed as "exec: <nil>" in every mutation transcript I produced.'
          round: 4
      findings:
        - id: BR-10
          severity: Important
          title: t.Helper() in SkipOrFail is unpinned; deleting it leaves both packages and go test ./... green in both env states
          detail: |-
            Measured on a scratch copy of clean head: removing t.Helper() at
            internal/conformance/conformance.go:72 keeps internal/conformance,
            internal/llm/llmtest and go test ./... green in default AND strict mode, while the
            child transcript changes from "wiring_test.go:33: network unavailable: dial refused"
            to "conformance.go:75: ...". message()'s own doc says the reader must learn that the
            DEPENDENCY was missing rather than the code broken, and the attributed file:line is
            half of how they learn it — lost, all 18 SkipOrFail call sites point at the guard.
            This is the same shape as the defect the issue exists to fix: a property of the guard
            established by having read output once and by no test. TestSkipOrFailPrintsTheMessage
            already reads the transcript, so the fix is a wantCallSite column in the existing
            table (or asserting the transcript does NOT contain "conformance.go:"), one row per
            direction (ARCH-PURPOSE).
          family: verified-by-reading-not-by-a-test
          round: 4
        - id: BR-11
          severity: Minor
          title: the re-exec selector is a literal unrelated to the test name, so a child that runs zero tests is reported as a wiring defect
          detail: |-
            This is the 2nd finding in family swallowed-error-misattributes-cause. Do NOT fix the
            instance. BR-9 closed the spawn-failure door; this is the child-ran-nothing door.
            Reproduced on a scratch copy by pointing wiring_test.go:70's
            -test.run=^TestSkipOrFailPrintsTheMessage$ at a renamed target: the child exits 0
            with "testing: warning: no tests to run", and the parent reports "SkipOrFail is not
            routing through message()" with exec: <nil> — a confident wrong cause. The rule that
            covers both doors: a child-process assertion must first establish that the child DID
            the work before reading its output as evidence about the code under test. Measured
            prevalence 1 — grep -rn 'os.Args\[0\]' --include='*.go' returns only this line;
            cmd/define/pty_conformance_test.go builds a separate binary and is a different shape.
            Cheapest expression: capture name := t.Name() once, build the selector from it (so it
            cannot drift), and assert the transcript contains "--- SKIP: "+name / "--- FAIL: "+name.
          family: swallowed-error-misattributes-cause
          round: 4
        - id: BR-12
          severity: Minor
          title: CONFORMANCE_STRICT=0 turns strict ON, newly pinned by a test whose name advertises only the empty case and documented nowhere
          detail: |-
            skiporfail_test.go:66 newly asserts {"0", true}, deciding a surprising contract. Its
            test name (TestStrictTreatsAnEmptyValueAsOff) covers one of four rows, and neither
            Strict()'s doc at conformance.go:64, the package doc, README.md:302,308 nor
            atlas/define.md:1015 says that any non-empty value counts. The test's own comment
            argues that "CONFORMANCE_STRICT=" is a plausible way to try to turn the mode off —
            "=0" is the more common way, and it silently turns it on. The rule: when a test is
            the only place a surprising contract is decided, the contract belongs in the doc
            comment of the symbol that owns it. One line on Strict().
          family: contract-pinned-only-in-a-test
          round: 4
      blocked: false
---

# Gate ledger — tools#25 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T21:10:33-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `test-inherits-ambient-env` TestDoesNotSkipWhenTheServiceAnswersWithAnError exercises a conformance-routed helper without controlling CONFORMANCE_STRICT
  internal/llm/llmtest/reachable_test.go:38 calls SkipIfUnreachable on a substitute T
  while its two neighbours in the same file now neutralize the variable, so the
  Done-when's universal claim is false as written (ARCH-PURPOSE). It is invariant
  today only because a successful dial never reaches conformance.SkipOrFail; that is
  a property of current control flow, not of the test. Fix: add
  t.Setenv(conformance.StrictEnv, "") at the top, matching line 23.
- **BR-2** [Minor] `decision-record-premise-wrong` The Plan's ARCH-DRY reason for not extracting the substitute-T idiom does not hold for the same-package majority
  The note says a shared helper would force golden_test.go to import
  internal/conformance, but golden_test.go and reachable_test.go are both
  package llmtest, so an unexported helper there covers 4 of the 5 goroutine+done
  sites with no new package and no conformance coupling; only skiporfail_test.go is
  genuinely cross-package. The note also repeats PQ-3's "eight sites" that the same
  issue's Done-when corrects to seven (and the full idiom is 5 sites, not 8).
  The decision may still be right; the recorded reason is not.
- **BR-3** [Minor] `unreproducible-evidence-claim` Mutation red-counts recorded in the issue Log do not reproduce
  The Log records "(4 red), (4), (7), (9)" for the four mutations; counting
  "--- FAIL" lines from go test -v gives 4 / 3 / 3 / 8, with no command or counting
  convention stated to reconcile them. The substantive claim (each mutation reds a
  named test) holds and I verified it. Record the named tests per row, which is what
  the Plan item promises, rather than a bare count.
- **BR-4** [Minor] `assertion-readability` The Failed() assertion in skiporfail_test.go is expressed inversely
  internal/conformance/skiporfail_test.go:47 reads
  `if got := fake.Failed(); got == tc.wantSkip` and prints want as `!tc.wantSkip`.
  Correct, but a `wantFail := !tc.wantSkip` local would state the intent directly.
- **BR-5** [Minor] `import-grouping` The conformance import in reachable_test.go sits inside the stdlib group
  internal/llm/llmtest/reachable_test.go:4 places
  github.com/xianxu/tools/internal/conformance ahead of net/http in the same block.
  gofmt -l is clean and it matches reachable.go's existing style, but the new
  skiporfail_test.go groups correctly and goimports would split this one.

## Round 2 — 2026-08-27T21:26:18-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-08-27T21:47:04-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Fixed as the class via substituteT(t, strict, fn); verified by revert — removing t.Setenv reds TestStrictTurnsAnUnreachableServiceIntoAFailure (default) and TestSkipsOnlyWhenNothingIsListening (strict).
- BR-2 — addressed — Plan's ARCH-DRY paragraph reversed, false same-package premise removed; the residual wrong count in the correction is raised separately as an unreproducible-evidence-claim instance, not as BR-2.
- BR-3 — addressed — Table now records named tests, command and baseline; I ran all four mutations at 05dee92 and every row reproduces exactly.
- BR-4 — addressed — wantFail := !tc.wantSkip at internal/conformance/skiporfail_test.go:44.
- BR-5 — addressed — The extraction removed the mis-grouped import from reachable_test.go entirely; substitute_test.go and wiring_test.go both group correctly.

### Raised

- **BR-6** [Important] `check-that-cannot-fail-reads-as-green` wiring_test.go's default-mode row cannot fail for the mode it names — its want string is a prefix of the strict message
  internal/conformance/wiring_test.go:45 asserts strings.Contains(out, "network
  unavailable: dial refused") for the default row, which is a substring of the strict
  message. Verified on clean head: mutating SkipOrFail to t.Skip(message(reason, err,
  true)) — so every offline skip falsely announces "(CONFORMANCE_STRICT is set)" —
  leaves internal/conformance, internal/llm/llmtest and go test ./... green in BOTH env
  states. The mirror mutation on the strict branch DOES red, so exactly one of the two
  directions is pinned at the call site while Done-when row 3 claims the wiring is
  pinned without qualification. The same weakness means the row cannot detect that its
  own CONFORMANCE_STRICT= override failed to beat an inherited =1. Fix: assert the
  default row does NOT contain conformance.StrictEnv+" is set", or narrow the Done-when
  row to the direction actually pinned (ARCH-PURPOSE).
- **BR-7** [Minor] `unreproducible-evidence-claim` "five" is wrong in four places, including the sentence that explains why counts do not belong in this issue
  This is the 3rd finding in family unreproducible-evidence-claim (BR-3 was the mutation
  red-counts). Do NOT patch five to six. Measured at ba321d1: package llmtest had SIX
  substitute-T sites (golden_test.go:28,35,49; reachable_test.go:26,47,70), FOUR of them
  goroutine sites; the fifth goroutine site is skiporfail_test.go, a different package.
  So issue lines 74 and 289 ("collapsed five package llmtest sites") are six, and lines
  119 and 267 ("all five goroutine sites in package llmtest") are four goroutine / six
  routed; line 126's cross-package five is correct. Measured prevalence four sites. The
  rule is already written verbatim in the Done-when — a count in a durable artifact is a
  restatement of a fact the code owns, and it drifts — and was applied at exactly one
  row. Rule-level fix: replace every remaining code-site count in the issue with the
  invariant plus its derivation grep, the way that row already does.
- **BR-8** [Minor] `test-inherits-ambient-env` substituteT's doc says the mode is set "for the duration", but t.Setenv binds to the calling test, not to fn
  This is the 2nd finding in family test-inherits-ambient-env (BR-1 was the missed
  reachable_test.go site). Do NOT special-case a caller. internal/llm/llmtest/substitute_test.go:35
  calls t.Setenv on the parent t, so the mode outlives fn and persists to the end of the
  test. Measured prevalence: 0 affected sites today — each caller uses one mode and every
  call re-sets the variable. The rule that covers it: a helper that sets an environment
  variable on a callee's behalf must state whose scope it restores at, so a caller mixing
  modes in one test body knows it needs a t.Run. Cheapest expression is the doc line at
  substitute_test.go:10 — "for the remainder of the calling test", not "for the duration".
- **BR-9** [Minor] `swallowed-error-misattributes-cause` wiring_test.go discards CombinedOutput's error, so a spawn failure is reported as a wiring bug
  internal/conformance/wiring_test.go:59 does out, _ := cmd.CombinedOutput(). The discard
  is right for the exit status (non-zero under strict, by design) but also swallows a
  genuine spawn failure; out is then empty and the assertion at :62 reports "SkipOrFail is
  not routing through message()" — a confident wrong cause for a reader debugging CI.
  Capture the error and include it in the t.Errorf alongside the transcript.

## Round 4 — 2026-08-27T22:03:56-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — Verified by mutation, not by the commit message: the strict-suffix mutation now reddens the named default row in BOTH env states.
- BR-7 — addressed — Code-site counts removed; surviving "five"s are lines-of-code and estimate rows. I ran both derivation greps and the invariant holds.
- BR-8 — addressed — substitute_test.go:8-14 now states the real t.Setenv scope and the two-subtest remedy.
- BR-9 — addressed — runErr captured and reachable at both assertions — it printed as "exec: <nil>" in every mutation transcript I produced.

### Raised

- **BR-10** [Important] `verified-by-reading-not-by-a-test` t.Helper() in SkipOrFail is unpinned; deleting it leaves both packages and go test ./... green in both env states
  Measured on a scratch copy of clean head: removing t.Helper() at
  internal/conformance/conformance.go:72 keeps internal/conformance,
  internal/llm/llmtest and go test ./... green in default AND strict mode, while the
  child transcript changes from "wiring_test.go:33: network unavailable: dial refused"
  to "conformance.go:75: ...". message()'s own doc says the reader must learn that the
  DEPENDENCY was missing rather than the code broken, and the attributed file:line is
  half of how they learn it — lost, all 18 SkipOrFail call sites point at the guard.
  This is the same shape as the defect the issue exists to fix: a property of the guard
  established by having read output once and by no test. TestSkipOrFailPrintsTheMessage
  already reads the transcript, so the fix is a wantCallSite column in the existing
  table (or asserting the transcript does NOT contain "conformance.go:"), one row per
  direction (ARCH-PURPOSE).
- **BR-11** [Minor] `swallowed-error-misattributes-cause` the re-exec selector is a literal unrelated to the test name, so a child that runs zero tests is reported as a wiring defect
  This is the 2nd finding in family swallowed-error-misattributes-cause. Do NOT fix the
  instance. BR-9 closed the spawn-failure door; this is the child-ran-nothing door.
  Reproduced on a scratch copy by pointing wiring_test.go:70's
  -test.run=^TestSkipOrFailPrintsTheMessage$ at a renamed target: the child exits 0
  with "testing: warning: no tests to run", and the parent reports "SkipOrFail is not
  routing through message()" with exec: <nil> — a confident wrong cause. The rule that
  covers both doors: a child-process assertion must first establish that the child DID
  the work before reading its output as evidence about the code under test. Measured
  prevalence 1 — grep -rn 'os.Args\[0\]' --include='*.go' returns only this line;
  cmd/define/pty_conformance_test.go builds a separate binary and is a different shape.
  Cheapest expression: capture name := t.Name() once, build the selector from it (so it
  cannot drift), and assert the transcript contains "--- SKIP: "+name / "--- FAIL: "+name.
- **BR-12** [Minor] `contract-pinned-only-in-a-test` CONFORMANCE_STRICT=0 turns strict ON, newly pinned by a test whose name advertises only the empty case and documented nowhere
  skiporfail_test.go:66 newly asserts {"0", true}, deciding a surprising contract. Its
  test name (TestStrictTreatsAnEmptyValueAsOff) covers one of four rows, and neither
  Strict()'s doc at conformance.go:64, the package doc, README.md:302,308 nor
  atlas/define.md:1015 says that any non-empty value counts. The test's own comment
  argues that "CONFORMANCE_STRICT=" is a plausible way to try to turn the mode off —
  "=0" is the more common way, and it silently turns it on. The rule: when a test is
  the only place a surprising contract is decided, the contract belongs in the doc
  comment of the symbol that owns it. One line on Strict().

## Open findings

- **BR-10** [Important] `verified-by-reading-not-by-a-test` t.Helper() in SkipOrFail is unpinned; deleting it leaves both packages and go test ./... green in both env states
- **BR-11** [Minor] `swallowed-error-misattributes-cause` the re-exec selector is a literal unrelated to the test name, so a child that runs zero tests is reported as a wiring defect
- **BR-12** [Minor] `contract-pinned-only-in-a-test` CONFORMANCE_STRICT=0 turns strict ON, newly pinned by a test whose name advertises only the empty case and documented nowhere
