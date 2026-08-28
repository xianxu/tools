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

---

## Re-review — 2026-08-27T21:26:18-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 25 — conformance guard: the strict inversion was unpinned, and it broke a test that had pinned it |
| repo | tools |
| issue file | workshop/issues/000025-conformance-guard-unpinned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27b6f1023770b6f76bc74426b6b90b94efeb97b8..de370f3790daafa737c64552b95cc41dc238660c |
| command | sdlc close --issue 25 |
| reviewer | claude |
| timestamp | 2026-08-27T21:26:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code half of this boundary is solid and I verified it rather than reading it: `go test ./...` and `CONFORMANCE_STRICT=1 go test ./...` are both fully green (not just the two packages — the whole tree), all four mutations in the Log reproduce the exact named tests recorded, and BR-1's fix is reachable — deleting `t.Setenv` from `substituteT` reds `TestStrictTurnsAnUnreachableServiceIntoAFailure` in default mode. What blocks SHIP is two things the same commit left behind. First, a real coverage gap: `message()` is fully pinned but its *wiring* is not — I mutated `SkipOrFail` to call `t.Fatal(reason)`/`t.Skip(reason)`, dropping the cause and the `CONFORMANCE_STRICT is set` annotation from every real failure, and the entire suite stayed green. Second, 05dee92 (the BR-1/BR-2 fix) touched no artifact at all, so the `## Done when` enumeration table and the `## Plan`'s ARCH-DRY paragraph both still describe a tree that the extraction deleted — including a greppable command the issue offers as its own proof, which now returns 2 where it says 7.

### 1. Strengths

- **`internal/conformance/conformance.go:71`** — the split is the right ARCH-PURE move, not a test convenience. `message()` is a genuinely pure function (no IO, no clock, no env), `Strict()` is the one-line env read, and `SkipOrFail` is four lines of `Fatal`/`Skip` glue. `TestMessage` runs with zero IO, which is what PURE is supposed to buy.
- **`internal/llm/llmtest/substitute_test.go:33`** — BR-1 was answered as the *class*, not the instance. Making the mode a required parameter rather than a convention means the third site cannot be forgotten the way it was; all five goroutine sites in `package llmtest` route through it, verified by grep (`testing.T{}` now appears exactly twice repo-wide).
- **`internal/conformance/skiporfail_test.go:14`** — both directions × err/nil is the honest 2×2, and the `nil`-err rows are the ones that would have been skipped by a less careful author.
- **The Log's mutation table reproduces exactly.** I ran all four mutations against `05dee92` semantics with the recorded command; the reddened test names match row for row, including M4's seven-test blast radius. BR-3's fix is real.
- **`internal/llm/llmtest/golden_test.go:46`** — the comment that says the `""` there "buys uniformity rather than correctness" is exactly the kind of honesty that stops the next reader from cargo-culting it.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — `SkipOrFail` can stop calling `message()` and nothing reds.** `internal/conformance/conformance.go:73-76`.

Verified by mutation:

```go
if Strict() { t.Fatal(reason) }   // was: t.Fatal(message(reason, err, true))
t.Skip(reason)                    // was: t.Skip(message(reason, err, false))
```

→ `ok internal/conformance`, `ok internal/llm/llmtest`, and `go test ./...` green in both env states. Every real failure loses its cause *and* the `(CONFORMANCE_STRICT is set)` annotation — the thing the function's own doc comment calls "the load-bearing part" — and no test notices. `TestMessage` pins the string builder at 100%; `TestSkipOrFailBothDirections` pins only `Skipped()`/`Failed()`. The seam between them is unpinned.

This is the general shape of "extract it so it can be asserted": the extraction moves the assertable part out and leaves the call site unobservable. Fix, cheapest first: (a) record the boundary explicitly in the Done-when — the row currently reads as if the reader-visible text is pinned, and it is only the *builder* that is; the issue already argues persuasively that shelling out to `go test` is past its stopping point, so this is a defensible answer if it is *stated*. Or (b) if you want it actually pinned, the `GO_WANT_HELPER_PROCESS` re-exec idiom (~25 lines) is the standard Go answer for helpers that end a test, and it would catch this mutation.

**I-2 — the `## Done when` enumeration table and its greppable proof no longer reproduce.** `workshop/issues/000025-conformance-guard-unpinned.md:62-78`.

> **This is the 2nd finding in family `unreproducible-evidence-claim`.** Earlier rounds fixed instances (BR-3, the mutation red-counts). Do NOT fix this instance alone.

Measured now:

| site | recorded | actual |
|---|---|---|
| `reachable_test.go` | 3 | 0 |
| `skiporfail_test.go` | 1 | 1 |
| `golden_test.go` | 3 | 0 |
| `substitute_test.go` | — (absent from the table) | 1 |
| **total / the `# 7` comment** | **7** | **2** |

The rule that covers both this and BR-3, and the one to fix instead of the instance: **a fix that changes the code must sweep, in the same commit, every artifact claim the change invalidates.** `05dee92` extracted `substituteT` and touched zero artifact files; `de370f3` then updated only the `## Log`. Measured prevalence of that one commit's un-swept wake: **three** false claims across two sections — the per-file table (:67-71), the `# 7` grep (:74), and the Plan's ARCH-DRY paragraph (BR-2 below). AGENTS.md already states the rule ("Revising a plan artifact mid-stream: append a `## Revisions` section"), and this issue has no `## Revisions` section. So the rule-level fix is one `## Revisions` entry covering all three at once, not a patched number.

**BR-2 — not addressed.** `workshop/issues/000025-conformance-guard-unpinned.md:106`. The `## Log` narrates the correction, but the artifact BR-2 was actually about is unchanged. The Plan still reads "**ARCH-DRY, acknowledged and NOT extracted**", still says a shared helper "would make `golden_test.go` … depend on the conformance package" (the premise BR-2 showed is false — both files are `package llmtest`), and still says "eight sites across three files". The code now *does* extract, into a helper in exactly the place the paragraph argues is impossible. A reader of the Plan meets the opposite of the decision that shipped. Same `## Revisions` entry as I-2.

### 4. Minor findings

- **`internal/llm/llmtest/substitute_test.go:35`** — `substituteT`'s `t.Setenv` is scoped to the *parent test*, not to `fn`, so the mode leaks into everything after the call. **This is the 2nd finding in family `test-inherits-ambient-env`**; measured prevalence is **0 affected sites today** (all five callers use one mode per test, and only `TestStrictTurns…` passes `"1"`, with nothing after it). Stating the rule rather than patching: *a helper that sets env on behalf of a callee should restore it when the callee returns, or the mode must be isolated in its own `t.Run`.* Cheapest expression of that rule is one line of doc on the helper saying the mode outlives `fn`; a caller mixing modes in one test body is the failure it prevents.
- `internal/conformance/skiporfail_test.go:57` — `TestStrictTreatsAnEmptyValueAsOff` iterates four rows without `t.Run`, so all four share one test name in the mutation table. The `t.Errorf` does print the value, so it is diagnosable; sub-tests would make the table row precise.

### 5. Test coverage notes

- The inversion itself is now pinned at both the rule (`skiporfail_test.go`) and a real call site (`reachable_test.go:53`) — that was the point of the issue and it is delivered.
- Mutation-verified by me, independently: M1–M4 reproduce the Log's named tests exactly; the `substituteT`-drops-`Setenv` mutation reds `TestStrictTurnsAnUnreachableServiceIntoAFailure`. The one mutation that survives is I-1.
- `gofmt -l` and `go vet` clean over `./internal ./cmd`. BR-4 (`wantFail := !tc.wantSkip`, `skiporfail_test.go:44`) and BR-5 (the mis-grouped import, removed from `reachable_test.go` entirely by the extraction) are both genuinely fixed.
- Docs gate: no finding. The diff adds no user-facing surface — `message` and `substituteT` are both unexported, and `CONFORMANCE_STRICT` was already documented in `atlas/define.md:1015-1041` by #24. The issue's plan to close with `--no-atlas` is the right call.

### 6. Architectural notes

- **ARCH-DRY — pass (code), flagged (record).** Five sites collapsed to one helper; the single remaining inline copy in `internal/conformance/skiporfail_test.go:36` is genuinely cross-package and exporting a test helper from a production package would be the worse trade. The *record* of that reasoning is wrong — BR-2.
- **ARCH-PURE — pass, and the best thing here.** `message()` is the pure core, `Strict()` the thin env read, `SkipOrFail` four lines of glue. No mocks needed to test the pure part. Note that I-1 is the characteristic *cost* of this shape: pushing logic into a pure function leaves the glue unobserved, so the glue has to be small enough that a reader can verify it by eye — which it now is.
- **ARCH-PURPOSE — pass (code), flagged (artifacts).** BR-1 was answered as the class, which is exactly right. But the class sweep stopped at the `.go` files; the enumeration the issue itself calls "stated rather than swept" was the thing not swept (I-2).
- **ARCH-MOCK — pass.** No new external dependency. `httptest.NewServer` and the closed port at `127.0.0.1:1` are the two states of the seam, exercised at the seam; the live conformance checks under `-tags conformance` remain the drift detector.
- **Forward-looking:** the deferred MIRROR half (an absent dependency written as an unconditional `Fatal`) remains invisible to `TestEverySkipIsRoutedOrWaived`, whose regex is `\bt\.Skipf?\(|\bt\.SkipNow\(`. Correctly tracked to #24's Risks. Worth noting that the same enumerate-then-restate hazard applies there: whatever enforces it should be a consumer of the source, not a grep someone re-runs.

### 7. Plan revision recommendations

One `## Revisions` entry on `workshop/issues/000025-conformance-guard-unpinned.md`, timestamped, covering all three claims that `05dee92` invalidated at once (this is the rule-level fix for I-2 and BR-2, not three separate patches):

> `## Revisions`
> **2026-08-27 — BR-1's fix extracted the idiom, which invalidated three recorded claims.**
> - `## Done when`, the enumeration table (:67-71) and its `# 7` grep (:74): the substitute-`T` sites are now **two** — `substitute_test.go` 1, `skiporfail_test.go` 1 — because `substituteT` collapsed the five `package llmtest` sites into one helper. The invariant the table was stating is unchanged and now *stronger* (the mode is a parameter, not a convention); restate it as "all five goroutine sites route through `substituteT`, which takes the mode", and re-derive the grep against the current tree.
> - `## Plan`, the ARCH-DRY paragraph (:106): reverse it. The helper WAS extracted, the "would force `golden_test.go` to import `internal/conformance`" premise was false (same package), and the count was 5 sites and not 8. Keep the surviving half of the decision — `internal/conformance`'s own test stays inline because it is genuinely cross-package.
> - `## Done when`, the failure-TEXT row: record the pinning boundary — `message()` is asserted directly, but nothing pins that `SkipOrFail` calls it (verified: bypassing `message()` leaves `go test ./...` green in both modes). Either state that as the deliberate stopping point, consistent with the existing non-goal paragraph, or pin it with a re-exec helper-process test.
