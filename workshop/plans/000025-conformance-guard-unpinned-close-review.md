# Boundary Review — tools#25 (whole-issue close)

| field | value |
|-------|-------|
| issue | 25 — conformance guard: the strict inversion was unpinned, and it broke a test that had pinned it |
| repo | tools |
| issue file | workshop/issues/000025-conformance-guard-unpinned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27b6f1023770b6f76bc74426b6b90b94efeb97b8..ba321d1a39a29464d381bec921246acde05f2d95 |
| command | sdlc close --issue 25 |
| reviewer | claude |
| timestamp | 2026-08-27T21:10:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff does what the issue says: `message(reason, err, strict)` is split out of `SkipOrFail` (conformance.go:92) so the text a reader of a red log sees is assertable for the first time, both directions are pinned for both the `err` and `nil` paths, and the actual regression — `TestSkipsOnlyWhenNothingIsListening` inheriting the ambient variable — is fixed at reachable_test.go:23. I verified the claims rather than reading them: I reproduced the Done-when greps (7 substitute-`T` sites, `golden.go` conformance refs = 0), ran the suites unsandboxed in both env states, ran `-race`, `gofmt -l`, and `go vet -tags conformance ./...` clean, and ran five mutations in a scratch copy including reverting the fix itself — deleting the single `t.Setenv` line reds `TestSkipsOnlyWhenNothingIsListening` under `CONFORMANCE_STRICT=1` and stays green in default mode, which is precisely the two-mode disagreement the issue exists to close. Nothing blocks. One Important remains: the Done-when asserts "**every** test that exercises a conformance-routed helper controls `CONFORMANCE_STRICT` itself", and one member of the enumerated class doesn't — `TestDoesNotSkipWhenTheServiceAnswersWithAnError` (reachable_test.go:38) exercises `SkipIfUnreachable` with the variable uncontrolled. It is invariant today only because a successful dial never reaches the guard, and the fix is one line.

### 1. Strengths

- **conformance.go:92 — the extraction is the right shape, and it didn't drift the output.** `message` is a pure function with no IO, tested directly (ARCH-PURE); `SkipOrFail` is now thin `Fatal`/`Skip` glue over `Strict()` + `*testing.T`. I checked the byte-faithfulness concern the swap invites: `t.Fatal(s)` routes through `Sprintln` where the old code used `Fatalf`, but `common.decorate` strips the trailing newline, so a strict failure still prints `dict_conformance_test.go:41: system dictionary unreachable: no dictionary entry (CONFORMANCE_STRICT is set)` — identical, `t.Helper()` still attributing the caller's line.
- **reachable_test.go:23 — the fix is pinned by a test that fails without it, verified by revert.** This is the check the claimed-fixes rule exists for, and it passes: mutation E (delete the `t.Setenv`) → green default, red strict; restore → green in both.
- **conformance/message_test.go:14 — PQ-1 is genuinely addressed, not restated.** The strict row asserts the literal `"… (CONFORMANCE_STRICT is set)"` rather than comparing `StrictEnv` with itself, so the tautology the plan gate caught cannot reappear. Mutations B (drop the variable name) and C (drop the cause) each red it.
- **The Done-when enumeration is measured, not asserted.** Both greps reproduce exactly. And the "no mechanical cross-mode enforcement" non-goal is stated with its reason instead of a one-shot measurement being dressed up as a standing invariant — that is the honest form.
- **skiporfail_test.go:31 — `t.Setenv` per subtest** means the guard's own test no longer inherits what it is testing, which is the same defect class one level up.

### 2. Critical findings

None.

### 3. Important findings

**`internal/llm/llmtest/reachable_test.go:38` — one member of the swept class still inherits the ambient variable (ARCH-PURPOSE).**
`TestDoesNotSkipWhenTheServiceAnswersWithAnError` calls `SkipIfUnreachable(fake, srv.URL)` — a conformance-routed helper — without controlling `CONFORMANCE_STRICT`, while its two neighbours in the same file now do. The Done-when's universal ("every test that exercises a conformance-routed helper controls `CONFORMANCE_STRICT` itself") is therefore false as written. It is harmless today only because a successful dial never reaches `conformance.SkipOrFail`; any future change that routes the "answered with an error" path through the guard makes this test's verdict env-dependent again, silently. Fix: add `t.Setenv(conformance.StrictEnv, "")` at the top of the function, matching line 23, and it becomes a property of the source rather than of the current control flow.

### 4. Minor findings

- **`workshop/issues/000025-…md`, Plan's ARCH-DRY note — the recorded reason for not extracting is wrong for the same-package majority.** It says a shared helper "would need a test-support package that all three import, which would make `golden_test.go` … depend on the conformance package." But `golden_test.go` and `reachable_test.go` are both `package llmtest`, so a 5-line unexported `runOnFakeT(func(*testing.T)) *testing.T` in an llmtest test file covers 4 of the 5 goroutine+done sites with no new package and no conformance import. Only `skiporfail_test.go` (in `package conformance_test`) is genuinely cross-package. The note also repeats PQ-3's count of "eight sites" that the same issue's Done-when corrects to seven — and the full goroutine+done idiom is actually 5 sites, not 8 (`golden_test.go:28` and `:35` are bare `&testing.T{}`, no goroutine). The decision to skip the extraction is still defensible; the record shouldn't rest on a premise that doesn't hold.
- **`## Log` mutation counts don't reproduce.** The Log records "(4 red), (4), (7), (9)"; measured with `go test -v` and counting `--- FAIL` lines I get 4 / 3 / 3 / 8 for the same four mutations, with no stated command or counting convention to reconcile them. Record the *named tests* each mutation reds (which is what the Plan row actually promises) rather than a bare count.
- **`internal/conformance/skiporfail_test.go:47`** — `if got := fake.Failed(); got == tc.wantSkip` with `want` printed as `!tc.wantSkip` is correct but reads backwards; a `wantFail := !tc.wantSkip` local would say it plainly.
- **`internal/llm/llmtest/reachable_test.go:4`** — the `internal/conformance` import sits inside the stdlib group. `gofmt -l` is clean and it matches `reachable.go`'s existing style, but `skiporfail_test.go` groups correctly and `goimports` would split this one.

### 5. Test coverage notes

Coverage is the strong part of this boundary and I verified it independently rather than trusting the table. Five mutations in a scratch copy, each reddening a named test:

| mutation | reds |
|---|---|
| strict branch deleted from `SkipOrFail` | `TestSkipOrFailBothDirections` (2 rows) + `TestStrictTurnsAnUnreachableServiceIntoAFailure` |
| strict message drops the variable name | `TestMessage` (2 rows) |
| cause (`err`) dropped from the message | `TestMessage` (2 rows) |
| `Strict()` inverted | `TestSkipOrFailBothDirections`, `TestStrictTreatsAnEmptyValueAsOff`, `TestSkipsOnlyWhenNothingIsListening`, `TestStrictTurnsAnUnreachableServiceIntoAFailure` |
| `t.Setenv` removed from `TestSkipsOnlyWhenNothingIsListening` | that test, **under strict only** — the exact shipped defect |

Suites green in both env states (`internal/conformance`, `internal/llm/llmtest`, `internal/llm`); `-race` clean. Under `-tags conformance` with `CONFORMANCE_STRICT=1` the only failures on this machine are genuine absent-dependency ones (no pty, no dictionary, no `afplay`, no API key) — the mode behaving as designed, with each message correctly naming the variable and the caller's line.

The one uncovered thing is what the issue explicitly declares a non-goal, so I am not raising it as a finding — see below.

### 6. Architectural notes for upcoming work

- **ARCH-PURE — pass.** `message` is the pure core, `SkipOrFail` the thin seam, `Strict()` the single env read. Exactly the split PQ-1 asked for, and the internal-test choice (not exporting `message` for a test's convenience) is right.
- **ARCH-MOCK — pass.** The substitute `*testing.T` is the real type used as a stateful recorder (`Skipped()`/`Failed()` persist across the call), and production and test flow share the same `SkipOrFail` boundary. No new external-binary call outside the seam.
- **ARCH-DRY — flagged, Minor above.**
- **ARCH-PURPOSE — flagged, Important above.**
- **The declared non-goal deserves a second look later, not now.** "Two modes agree is a property of a RUN, not of the source" is true of the general invariant, but the specific class this issue swept *is* a source property, and `guard_test.go` already demonstrates the enforcement shape: a walk that flags any file referencing `conformance` which constructs a substitute `*testing.T` in a test function that never calls `t.Setenv(conformance.StrictEnv, …)`. That would have caught the Important above mechanically. It was gate-cleared as a deliberate deferral so I am not re-opening it here, but it is a cheaper option than the "run the suite twice in CI and diff" the non-goal contemplates, and worth an issue.

### 7. Plan revision recommendations

Add one `## Revisions` entry to `workshop/issues/000025-conformance-guard-unpinned.md` covering both corrections, since the Plan's ARCH-DRY paragraph is the artifact that will be read next time this question comes up:

> **Revisions — <timestamp> — ARCH-DRY rationale corrected.** The recorded reason for not extracting the substitute-`T` idiom said a shared helper would need a package all three files import, forcing `golden_test.go` to depend on `internal/conformance`. That is wrong for 4 of the 5 goroutine+done sites: `golden_test.go` and `reachable_test.go` are both `package llmtest`, so an unexported helper in an llmtest test file needs no new package and no conformance import. Only `skiporfail_test.go` (`package conformance_test`) is cross-package. The decision to keep the duplication stands on "5-line idiom, one cross-package holdout"; the earlier reason did not. Also corrects "eight sites" to the measured counts: 7 substitute-`T` sites, of which 5 use the full goroutine+done shape.

If the Important is fixed, the Done-when's enumeration table needs no change (the site is already inside the counted 3 for `reachable_test.go`) — but if it is instead consciously left, the Done-when's "every test that exercises a conformance-routed helper controls `CONFORMANCE_STRICT` itself" must be narrowed to the tests that can reach the guard branch, rather than left as a universal the tree contradicts.

```findings
findings:
  - id: new
    severity: Important
    family: test-inherits-ambient-env
    title: |
      TestDoesNotSkipWhenTheServiceAnswersWithAnError exercises a conformance-routed helper without controlling CONFORMANCE_STRICT
    detail: |
      internal/llm/llmtest/reachable_test.go:38 calls SkipIfUnreachable on a substitute T
      while its two neighbours in the same file now neutralize the variable, so the
      Done-when's universal claim is false as written (ARCH-PURPOSE). It is invariant
      today only because a successful dial never reaches conformance.SkipOrFail; that is
      a property of current control flow, not of the test. Fix: add
      t.Setenv(conformance.StrictEnv, "") at the top, matching line 23.
  - id: new
    severity: Minor
    family: decision-record-premise-wrong
    title: |
      The Plan's ARCH-DRY reason for not extracting the substitute-T idiom does not hold for the same-package majority
    detail: |
      The note says a shared helper would force golden_test.go to import
      internal/conformance, but golden_test.go and reachable_test.go are both
      package llmtest, so an unexported helper there covers 4 of the 5 goroutine+done
      sites with no new package and no conformance coupling; only skiporfail_test.go is
      genuinely cross-package. The note also repeats PQ-3's "eight sites" that the same
      issue's Done-when corrects to seven (and the full idiom is 5 sites, not 8).
      The decision may still be right; the recorded reason is not.
  - id: new
    severity: Minor
    family: unreproducible-evidence-claim
    title: |
      Mutation red-counts recorded in the issue Log do not reproduce
    detail: |
      The Log records "(4 red), (4), (7), (9)" for the four mutations; counting
      "--- FAIL" lines from go test -v gives 4 / 3 / 3 / 8, with no command or counting
      convention stated to reconcile them. The substantive claim (each mutation reds a
      named test) holds and I verified it. Record the named tests per row, which is what
      the Plan item promises, rather than a bare count.
  - id: new
    severity: Minor
    family: assertion-readability
    title: |
      The Failed() assertion in skiporfail_test.go is expressed inversely
    detail: |
      internal/conformance/skiporfail_test.go:47 reads
      `if got := fake.Failed(); got == tc.wantSkip` and prints want as `!tc.wantSkip`.
      Correct, but a `wantFail := !tc.wantSkip` local would state the intent directly.
  - id: new
    severity: Minor
    family: import-grouping
    title: |
      The conformance import in reachable_test.go sits inside the stdlib group
    detail: |
      internal/llm/llmtest/reachable_test.go:4 places
      github.com/xianxu/tools/internal/conformance ahead of net/http in the same block.
      gofmt -l is clean and it matches reachable.go's existing style, but the new
      skiporfail_test.go groups correctly and goimports would split this one.
```
