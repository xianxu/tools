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

## Open findings

- **BR-1** [Important] `test-inherits-ambient-env` TestDoesNotSkipWhenTheServiceAnswersWithAnError exercises a conformance-routed helper without controlling CONFORMANCE_STRICT
- **BR-2** [Minor] `decision-record-premise-wrong` The Plan's ARCH-DRY reason for not extracting the substitute-T idiom does not hold for the same-package majority
- **BR-3** [Minor] `unreproducible-evidence-claim` Mutation red-counts recorded in the issue Log do not reproduce
- **BR-4** [Minor] `assertion-readability` The Failed() assertion in skiporfail_test.go is expressed inversely
- **BR-5** [Minor] `import-grouping` The conformance import in reachable_test.go sits inside the stdlib group
