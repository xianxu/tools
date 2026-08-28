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

---

## Re-review — 2026-08-27T21:47:04-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 25 — conformance guard: the strict inversion was unpinned, and it broke a test that had pinned it |
| repo | tools |
| issue file | workshop/issues/000025-conformance-guard-unpinned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27b6f1023770b6f76bc74426b6b90b94efeb97b8..514014cadc6f20a0882461a60d944c88f5697337 |
| command | sdlc close --issue 25 |
| reviewer | claude |
| timestamp | 2026-08-27T21:47:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All five prior findings are genuinely addressed and I verified each by mutation rather than by reading the commit messages: reverting `t.Setenv` out of `substituteT` reds `TestStrictTurnsAnUnreachableServiceIntoAFailure` in default mode and `TestSkipsOnlyWhenNothingIsListening` under strict (BR-1 is reachable, not decorative); all four rows of the Log's mutation table reproduce **exactly** at the baseline the Log names, `05dee92` (BR-3); the ARCH-DRY paragraph is reversed with the false premise removed (BR-2); `wantFail` is a local (BR-4); and the mis-grouped import is gone with the file it lived in (BR-5). The new `wiring_test.go` closes the real gap round 2 found — bypassing `message()` inside `SkipOrFail` now reds both its subtests, where before it left `go test ./...` green in both modes. What keeps this off SHIP is one residual half of that same gap: the default-mode row asserts a string that is a **prefix** of the strict message, so the mutation `t.Skip(message(reason, err, true))` — every offline skip falsely announcing `(CONFORMANCE_STRICT is set)` — leaves both packages and the whole tree green in both env states. I ran it. That is the issue's own defect class (`check-that-cannot-fail-reads-as-green`) surviving inside the commit that claims to have closed it, and the fix is one line.

Verified independently: whole tree green under `go test ./...` and `CONFORMANCE_STRICT=1 go test ./...`; `-race` clean in both modes; `gofmt -l` and `go vet` clean; both Done-when derivation greps reproduce (2 substitute-`T` constructions, 6 `substituteT` call sites); the printed message is byte-identical to the pre-change `Fatalf` output (`wiring_test.go:33: network unavailable: dial refused (CONFORMANCE_STRICT is set)`), with `t.Helper()` still attributing the caller's line — no drift from the `Fatalf`→`Fatal` swap.

### 1. Strengths

- **`internal/conformance/wiring_test.go:27` — the re-exec is the right answer and it is genuinely load-bearing.** This is the standard Go helper-process idiom, and it is the only way to observe text a substitute `*testing.T` structurally cannot expose. Mutation-confirmed: bypassing `message()` reds both subtests; `t.Fatal(message(reason, err, false))` reds the strict subtest.
- **`internal/llm/llmtest/substitute_test.go:33` — BR-1 answered as the class, and the class is the right one.** Making the mode a *parameter* rather than a convention is what makes "cannot be forgotten" true; the doc comment saying so is honest about the fact that the convention had already been forgotten once, inside the fix.
- **`internal/conformance/conformance.go:71` — ARCH-PURE done properly.** `message` is pure (no IO, clock or env), `Strict()` is the single env read, `SkipOrFail` is four lines of glue. `TestMessage` runs with zero IO, and the internal-test choice avoids widening the API for a test's convenience.
- **The Log's mutation table now records named tests plus the exact command and baseline commit.** I ran all four against `05dee92` and every reddened test matches row for row, including M4's seven-test blast radius. This is the shape evidence claims should take in this repo.
- **`internal/llm/llmtest/golden_test.go:46`** — the comment stating that the `""` there "buys uniformity rather than correctness" stops the next reader from cargo-culting an env-set that isn't load-bearing.

### 2. Critical findings

None.

### 3. Important findings

**`internal/conformance/wiring_test.go:45` — the default-mode assertion cannot fail for the mode it names, because the expected string is a prefix of the wrong behavior's output.**

`want` for the default row is `"network unavailable: dial refused"`, asserted with `strings.Contains`. The strict message is `"network unavailable: dial refused (CONFORMANCE_STRICT is set)"` — a superstring. So the row passes whenever the child prints *either* message. Verified by mutation on the clean head:

```go
t.Skip(message(reason, err, true))   // was: message(reason, err, false)
```

→ `ok internal/conformance`, `ok internal/llm/llmtest`, and `go test ./...` green in **both** env states. Every default-mode skip in the tree would then read `master is not a terminal on this platform (CONFORMANCE_STRICT is set)` — announcing a mode that is off — and nothing notices. The asymmetry is real, not theoretical: the mirror mutation (`t.Fatal(message(reason, err, false))`) *does* red the strict subtest, so exactly one of the two directions this issue exists to pin is actually pinned at the call site.

The same weakness makes the subtest unable to detect that its own env override failed: if `CONFORMANCE_STRICT=` did not beat the inherited `=1`, the child would run strict and the row would still pass.

Fix, one line — add a negative assertion for the default row so the check excludes the alternative rather than merely including the expected:

```go
{name: "default: …", strict: "", want: "network unavailable: dial refused",
 unwanted: conformance.StrictEnv + " is set"},
```
and `if tc.unwanted != "" && strings.Contains(string(out), tc.unwanted) { t.Errorf(...) }`. (Asserting the whole decorated line, `wiring_test.go:NN: <msg>\n`, would also do it, at the cost of a line-number-coupled fixture — the negative assertion is cheaper and states the intent.)

Done-when row 3 currently claims "**the WIRING is pinned, not just the text**" without qualification. Either the fix above, or narrow the row to "pinned in the strict direction; the default direction is pinned only against dropping `message()` entirely."

### 4. Minor findings

- **`workshop/issues/000025-conformance-guard-unpinned.md:74, :119, :267, :289` — "five" is wrong in four places (family `unreproducible-evidence-claim`, 3rd instance).** Measured at `ba321d1`: `package llmtest` had **six** substitute-`T` sites (`golden_test.go:28,35,49`, `reachable_test.go:26,47,70`), of which **four** used the goroutine idiom; the fifth goroutine site is `skiporfail_test.go`, a different package. So "collapsed five `package llmtest` sites" (:74, :289) is six, and "all five goroutine sites in `package llmtest`" (:119, :267) is four goroutine sites / six routed sites. `:126`'s cross-package "five" is correct. Do not patch the number — see §7 for the rule-level fix.
- **`internal/llm/llmtest/substitute_test.go:10` — the doc says the mode is set "for the duration", but `t.Setenv` is scoped to the *parent test*, not to `fn` (family `test-inherits-ambient-env`, 2nd instance).** Measured prevalence: 0 affected sites (each caller uses one mode, and every call re-sets the variable). See §7 for the rule.
- **`internal/conformance/wiring_test.go:59` — `out, _ := cmd.CombinedOutput()` discards a spawn failure, and the failure message then misdiagnoses it.** If the child cannot start, `out` is empty and the test reports "SkipOrFail is not routing through message()" — a confident, wrong cause. Capture the error and include it in the `t.Errorf`.
- `internal/conformance/wiring_test.go:32` — an ambient `CONFORMANCE_WIRING_HELPER` in a developer's environment silently turns the parent test into a skip. Not worth guarding, but the marker name is close enough to `CONFORMANCE_STRICT` to be worth a word in the comment.

### 5. Test coverage notes

- **Mutation-verified by me, on the clean head and on `05dee92`, not read from the Log.** `SkipOrFail` bypasses `message()` → both wiring subtests red. Strict branch formats with `strict=false` → strict subtest red. `substituteT` drops `t.Setenv` → `TestStrictTurnsAnUnreachableServiceIntoAFailure` red in default mode, `TestSkipsOnlyWhenNothingIsListening` red under strict. Log rows M1–M4 reproduce their named tests exactly at the stated baseline. The one surviving mutation is the Important above.
- Note that the Log's table is measured against `05dee92`, which predates `wiring_test.go`; re-run at head, M1 and M2 also red `TestSkipOrFailPrintsTheMessage`. The Log names its baseline, so it reproduces as written — worth a parenthetical, not a finding.
- Whole tree green in both env states; `-race` clean in both; `gofmt -l` and `go vet` clean over `./internal ./cmd`. Both Done-when greps reproduce against the current tree.
- **Docs gate: no finding.** Both new symbols are unexported (`message`, `substituteT`), the re-exec is internal, and `CONFORMANCE_STRICT` was already documented at `README.md:302-308` and `atlas/define.md:1017-1040` by #24. Closing with `--no-atlas` is the right call, and the Estimate section already says so with its reason.

### 6. Architectural notes for upcoming work

- **ARCH-PURE — pass, and the best thing in the diff.** Pure core (`message`), one-line env read (`Strict()`), thin glue (`SkipOrFail`), and the IO-shell test (`wiring_test.go`, which execs) is correctly isolated in its own file instead of being mixed into the pure `TestMessage`. Worth naming the general cost this diff paid for that shape: extraction moves the assertable part out and leaves the *call site* unobserved — I-1 last round, and the residual half of it in §3 this round. When you extract for assertability, budget the wiring test in the same step.
- **ARCH-DRY — pass.** Six sites collapsed into one helper; the single remaining inline copy (`internal/conformance/skiporfail_test.go:36`) is genuinely cross-package and exporting a test helper from a production package is the worse trade. The re-exec idiom is a single site, so no premature helper. Only the *record's* count is wrong (§4).
- **ARCH-PURPOSE — pass on the code, flagged on the claim.** BR-1 was fixed as the class, not the instance, which is exactly the principle. The gap is the Done-when's universal about the wiring, which the tree half-supports (§3).
- **ARCH-MOCK — pass.** The substitute `*testing.T` is the real type used as a stateful recorder across the call; the re-exec child is this same binary, not an external dependency outside a seam; `httptest.NewServer` and the closed port at `127.0.0.1:1` are the two states of the reachability seam, exercised at the seam. Production and test flow share the `SkipOrFail` boundary, and `-tags conformance` remains the live drift detector.
- **Forward-looking, not a finding.** `guard_test.go`'s own comment states this repo's hardest-won rule — "A grep cannot fail a build; this can … make the claim a CONSUMER of the thing it claims about" — and this issue's Done-when backs its invariant with two greps a human re-runs. The issue's non-goal paragraph disclaims *cross-mode agreement*, correctly, as a property of a RUN; but "every `package llmtest` goroutine site routes through `substituteT`" is a property of the SOURCE, enforceable by a walk shaped exactly like `TestEverySkipIsRoutedOrWaived`. That was gate-cleared as a deferral so I am not reopening it, but it is cheaper than the CI double-run the non-goal contemplates and it would have caught BR-1 mechanically. Worth an issue.

### 7. Plan revision recommendations

Both entries append to the existing `## Revisions` section.

> **2026-08-27 — the count came back, in the sentence explaining why counts don't belong here.** `## Done when` already states the rule — *a count in a durable artifact is a restatement of a fact the code owns, and it drifts* — and then four sentences in this issue restate a site count anyway (:74, :119, :267, :289), all of them wrong: `package llmtest` had six substitute-`T` sites, four of them goroutine sites; "five" was the cross-package goroutine count borrowed from the round-1 review. Applied as the rule rather than the instance: **every remaining count of code sites in this issue is replaced by the invariant plus its derivation command, the way the Done-when row already does it** — "all `package llmtest` sites route through `substituteT`, derived by `grep -rn 'substituteT(t,' internal/`" — rather than by changing five to six. This is the third instance of `unreproducible-evidence-claim` on this issue; the rule was written down at one site and not applied at the other four, which is the `instance-not-class` shape one level up.

> **2026-08-27 — the wiring row's scope, stated.** `## Done when` row 3 claims the wiring is pinned. Measured: the strict direction is pinned (`t.Fatal(message(reason, err, false))` reds `TestSkipOrFailPrintsTheMessage/strict`), the default direction is not — `t.Skip(message(reason, err, true))` leaves `go test ./...` green in both env states, because the default row's expected string is a prefix of the strict message. Either add the negative assertion and keep the row as written, or narrow the row to the direction actually pinned.

Also worth one line on `substitute_test.go:10`, as the rule for the 2nd `test-inherits-ambient-env` instance: **a helper that sets an environment variable on a callee's behalf must say whose scope it restores at, because `t.Setenv` binds to the test, not to the callback** — cheapest expression is fixing "for the duration" to "for the remainder of the calling test", so a future caller that mixes modes in one test body knows it must use `t.Run`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Fixed as the class via substituteT(t, strict, fn); verified by revert — removing t.Setenv reds TestStrictTurnsAnUnreachableServiceIntoAFailure (default) and TestSkipsOnlyWhenNothingIsListening (strict).
  - id: BR-2
    disposition: addressed
    note: |
      Plan's ARCH-DRY paragraph reversed, false same-package premise removed; the residual wrong count in the correction is raised separately as an unreproducible-evidence-claim instance, not as BR-2.
  - id: BR-3
    disposition: addressed
    note: |
      Table now records named tests, command and baseline; I ran all four mutations at 05dee92 and every row reproduces exactly.
  - id: BR-4
    disposition: addressed
    note: |
      wantFail := !tc.wantSkip at internal/conformance/skiporfail_test.go:44.
  - id: BR-5
    disposition: addressed
    note: |
      The extraction removed the mis-grouped import from reachable_test.go entirely; substitute_test.go and wiring_test.go both group correctly.
findings:
  - id: new
    severity: Important
    family: check-that-cannot-fail-reads-as-green
    title: |
      wiring_test.go's default-mode row cannot fail for the mode it names — its want string is a prefix of the strict message
    detail: |
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
  - id: new
    severity: Minor
    family: unreproducible-evidence-claim
    title: |
      "five" is wrong in four places, including the sentence that explains why counts do not belong in this issue
    detail: |
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
  - id: new
    severity: Minor
    family: test-inherits-ambient-env
    title: |
      substituteT's doc says the mode is set "for the duration", but t.Setenv binds to the calling test, not to fn
    detail: |
      This is the 2nd finding in family test-inherits-ambient-env (BR-1 was the missed
      reachable_test.go site). Do NOT special-case a caller. internal/llm/llmtest/substitute_test.go:35
      calls t.Setenv on the parent t, so the mode outlives fn and persists to the end of the
      test. Measured prevalence: 0 affected sites today — each caller uses one mode and every
      call re-sets the variable. The rule that covers it: a helper that sets an environment
      variable on a callee's behalf must state whose scope it restores at, so a caller mixing
      modes in one test body knows it needs a t.Run. Cheapest expression is the doc line at
      substitute_test.go:10 — "for the remainder of the calling test", not "for the duration".
  - id: new
    severity: Minor
    family: swallowed-error-misattributes-cause
    title: |
      wiring_test.go discards CombinedOutput's error, so a spawn failure is reported as a wiring bug
    detail: |
      internal/conformance/wiring_test.go:59 does out, _ := cmd.CombinedOutput(). The discard
      is right for the exit status (non-zero under strict, by design) but also swallows a
      genuine spawn failure; out is then empty and the assertion at :62 reports "SkipOrFail is
      not routing through message()" — a confident wrong cause for a reader debugging CI.
      Capture the error and include it in the t.Errorf alongside the transcript.
```

---

## Re-review — 2026-08-27T22:03:56-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 25 — conformance guard: the strict inversion was unpinned, and it broke a test that had pinned it |
| repo | tools |
| issue file | workshop/issues/000025-conformance-guard-unpinned.md |
| boundary | whole-issue close |
| milestone | — |
| window | 27b6f1023770b6f76bc74426b6b90b94efeb97b8..82a5679e8ac6335f97bdb7691ff028e122e5bd04 |
| command | sdlc close --issue 25 |
| reviewer | claude |
| timestamp | 2026-08-27T22:03:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four open findings from round 3 are genuinely addressed, and I verified the load-bearing one by mutation rather than by reading the commit message: mutating `SkipOrFail` so every offline skip announces `(CONFORMANCE_STRICT is set)` now reddens `TestSkipOrFailPrintsTheMessage/default:…` in **both** env states, where round 3 measured it green. I independently reproduced the Log's `Strict()`-inverted mutation row and it named exactly the tests recorded. The suite is green in both env states under `-race`, `go vet` is clean, and the Done-when's derivation greps produce the invariant they claim (6 `substituteT` sites, 1 goroutine in `package llmtest`, 2 substitute-`T` constructions). What holds this back from SHIP is one measured gap of exactly the kind this issue exists to close: `SkipOrFail`'s `t.Helper()` is verified by having read output once and by nothing else — deleting it moves every conformance red-log line from the caller to `conformance.go:75` across all 18 call sites, and leaves `internal/conformance`, `internal/llm/llmtest` and `go test ./...` green in both modes.

### 1. Strengths

- **`message_test.go` and `wiring_test.go` split the assertion correctly.** `message_test.go:31` hardcodes the literal `"…(CONFORMANCE_STRICT is set)"` so the variable *name* is pinned, while `wiring_test.go:47` derives `strictSuffix` from `conformance.StrictEnv` so the *shape* is pinned without re-implementing the format. That split avoids the PQ-1 vacuity in one direction and the brittleness in the other; it is the right decomposition, not an accident.
- **The BR-6 fix is real and reddens where it should.** `wiring_test.go:86` asserting the suffix `ABSENT` by default is what turns a prefix-containment check into a two-directional one. Verified: the round-3 mutation now fails in both env states, which also proves the `CONFORMANCE_STRICT=""` child override beats an inherited `=1` (Go dedups `Cmd.Env` last-wins).
- **`substituteT` makes the mode a parameter rather than a convention** (`substitute_test.go:38`). The BR-1 defect — two of three sites given `t.Setenv` by hand — is structurally unrepeatable now, and the grep-derived invariant in the Done-when confirms it holds today with zero remaining raw goroutine sites in `package llmtest`.
- **BR-9's fix is reachable, not decorative.** `exec: <nil>` appeared in every mutation transcript I produced, so `runErr` is genuinely on the reporting path.
- **The `## Estimate` remediation rows** are an honest structural correction — pricing *fixing* what a review finds, not just *conducting* it — rather than a fudge factor, and the refusal to carry #24's 0.67 ratio as a multiplier is the right call.

### 2. Critical findings

None.

### 3. Important findings

**`internal/conformance/conformance.go:72` — `t.Helper()` is the one property of `SkipOrFail` still verified by reading output, and nothing asserts it.**

Measured on a scratch copy of clean head: deleting `t.Helper()` leaves `internal/conformance`, `internal/llm/llmtest` and `go test ./...` **green in both env states**. The effect is observable and one `strings.Contains` away — the child transcript changes from

```
wiring_test.go:33: network unavailable: dial refused     (with t.Helper)
conformance.go:75: network unavailable: dial refused     (without)
```

`message()`'s own doc says the reader "needs to know that the DEPENDENCY was missing, not the code broken" — and *which file:line the log names* is half of how they learn that. Lose it and all 18 `SkipOrFail` call sites point at the guard. `TestSkipOrFailPrintsTheMessage` is the only place in the tree where this is observable, and it already reads the transcript. Fix sketch: add `wantCallSite: "wiring_test.go:"` to the table (or assert `!strings.Contains(transcript, "conformance.go:")`), one row each direction.

### 4. Minor findings

- **`internal/conformance/wiring_test.go:70` — 2nd in family `swallowed-error-misattributes-cause`.** The `-test.run=^TestSkipOrFailPrintsTheMessage$` selector is a string literal, unrelated to the function name. I renamed the selector target on a scratch copy: the child ran zero tests, exited 0, and the parent reported `SkipOrFail is not routing through message()` with `exec: <nil>` — BR-9's failure mode reached through a different door. Per the escalation, the rule rather than the instance: *a child-process assertion must first establish that the child DID the work, before reading its output as evidence about the code under test.* Measured prevalence: 1 (`grep -rn 'os.Args\[0\]' --include='*.go'` returns only this line; `cmd/define/pty_conformance_test.go` builds a separate binary and is a different shape). Cheapest expression covering both doors: capture `name := t.Name()` once at the top, build the selector from it, and assert the transcript contains `"--- SKIP: "+name` / `"--- FAIL: "+name`.
- **`internal/conformance/conformance.go:65` — `CONFORMANCE_STRICT=0` turns strict ON, and this boundary pins it as intended without saying so anywhere a reader looks.** `skiporfail_test.go:66` newly asserts `{"0", true}`, but the test's name advertises only the empty case, and `Strict()`'s doc, the package doc, `README.md:302,308` and `atlas/define.md:1015` all say only "set". The test's own comment reasons that `CONFORMANCE_STRICT=` "is a plausible way to try to turn the mode off" — `=0` is the *more* common way. One line on `Strict()`: "any non-empty value, including `0`, means strict; only unset or empty is off."
- `workshop/issues/000025-…md:71-73` — "Three versions of this row carried a number — eight, then seven" lists two numbers for three versions.

### 5. Test coverage notes

Independently run, all against clean head:

| check | result |
|---|---|
| `go test ./...`, default and `CONFORMANCE_STRICT=1` | green both; no test flips verdict on the variable |
| `go test -race ./internal/conformance/ ./internal/llm/llmtest/`, both modes | green |
| `go vet` on both packages | clean |
| BR-6 mutation (`t.Skip(message(…, true))`) | reddens `TestSkipOrFailPrintsTheMessage/default:…` in both modes ✅ |
| `Strict()` inverted | reddens all 4 `TestSkipOrFailBothDirections` rows + `TestStrictTreatsAnEmptyValueAsOff` + `TestSkipsOnlyWhenNothingIsListening` + `TestStrictTurnsAnUnreachableServiceIntoAFailure` — exactly as the Log's table records ✅ |
| `t.Helper()` removed | **green in both modes** ❌ (finding above) |

One further unpinned direction, below the bar for a finding but worth knowing: swapping `t.Fatal` → `t.Error` in the strict branch keeps everything green (`Failed()` is still true, the transcript still carries the text), so the *abort* semantics — which is what stops a conformance test from continuing against an absent dependency — rest on `Fatal` being read, not asserted. It is genuinely awkward to pin through a substitute `T`; the child transcript's `--- SKIP:`/`--- FAIL:` line is the cheap seam if the `t.Helper()` fix goes in anyway.

### 6. Architectural notes

- **ARCH-DRY — pass.** Exactly two substitute-`T` constructions remain (`substitute_test.go:44`, `skiporfail_test.go:36`), and the recorded reason for keeping the second inline is now the *correct* one after BR-2: `internal/conformance`'s test is genuinely cross-package, and exporting a test helper from a production package to save ~8 lines is the worse trade. I checked the premise rather than the prose — `golden_test.go` and `reachable_test.go` are both `package llmtest`, so the helper couples nothing.
- **ARCH-PURE — pass.** `message()` is the pure core and is unit-tested with no IO; `SkipOrFail` is thin env-read + `Fatal`/`Skip` glue. `wiring_test.go`'s subprocess is IO by necessity, not by leakage — the printed text is only observable in a real test binary, and the comment says so.
- **ARCH-PURPOSE — pass, with the Important above as the residue.** Shadow-sweep run: `Strict()` is the single source, its only consumer is `SkipOrFail`, `message()` takes the mode as a parameter, and every unit test of a routed helper (`SkipIfUnreachable`, the only one — `grep 'conformance.SkipOrFail'` shows all other sites are conformance suites that *should* track the variable) now states its mode. No hand-maintained restatement of the model survives. The declared non-goal (mechanical cross-mode enforcement is a property of a RUN, not of the source) is an honest stopping point, correctly argued.
- **ARCH-MOCK — pass, not applicable.** No new external binary or service seam; the re-exec target is this test binary itself, and `llmtest`'s stateful fake at the wire boundary is untouched.
- Forward-looking: the deferred MIRROR half — an absent dependency written as an unconditional `Fatal`, invisible to `guard_test.go`'s `t.Skipf?\(|t.SkipNow\(` pattern — remains the largest real gap in this rule and is correctly tracked in #24's Risks rather than smuggled in here.

### 7. Plan revision recommendations

- If the `t.Helper()` finding is taken, add a `## Revisions` entry extending the failure-TEXT Done-when row: what a red log shows is the message *and* the site it is attributed to, and the second half was unasserted until now. If it is deliberately declined, it belongs in the non-goal paragraph, not unstated.
- `## Log` → "Verified UNSANDBOXED in both env states" reads as covering the `llm` conformance suites, but those sit behind `//go:build conformance` and are absent from a bare `go test ./...`. Naming the flag (`-tags conformance`) makes the bullet reproducible in one step; I needed two to work out which run it described. Precision note, not a new finding — the family's rule is already stated in the Done-when.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Verified by mutation, not by the commit message: the strict-suffix mutation now reddens the named default row in BOTH env states.
  - id: BR-7
    disposition: addressed
    note: |
      Code-site counts removed; surviving "five"s are lines-of-code and estimate rows. I ran both derivation greps and the invariant holds.
  - id: BR-8
    disposition: addressed
    note: |
      substitute_test.go:8-14 now states the real t.Setenv scope and the two-subtest remedy.
  - id: BR-9
    disposition: addressed
    note: |
      runErr captured and reachable at both assertions — it printed as "exec: <nil>" in every mutation transcript I produced.
findings:
  - id: new
    severity: Important
    family: verified-by-reading-not-by-a-test
    title: |
      t.Helper() in SkipOrFail is unpinned; deleting it leaves both packages and go test ./... green in both env states
    detail: |
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
  - id: new
    severity: Minor
    family: swallowed-error-misattributes-cause
    title: |
      the re-exec selector is a literal unrelated to the test name, so a child that runs zero tests is reported as a wiring defect
    detail: |
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
  - id: new
    severity: Minor
    family: contract-pinned-only-in-a-test
    title: |
      CONFORMANCE_STRICT=0 turns strict ON, newly pinned by a test whose name advertises only the empty case and documented nowhere
    detail: |
      skiporfail_test.go:66 newly asserts {"0", true}, deciding a surprising contract. Its
      test name (TestStrictTreatsAnEmptyValueAsOff) covers one of four rows, and neither
      Strict()'s doc at conformance.go:64, the package doc, README.md:302,308 nor
      atlas/define.md:1015 says that any non-empty value counts. The test's own comment
      argues that "CONFORMANCE_STRICT=" is a plausible way to try to turn the mode off —
      "=0" is the more common way, and it silently turns it on. The rule: when a test is
      the only place a surprising contract is decided, the contract belongs in the doc
      comment of the symbol that owns it. One line on Strict().
```
