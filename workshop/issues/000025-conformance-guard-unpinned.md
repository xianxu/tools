---
id: 000025
status: codecomplete
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 3.64
started: 2026-08-27T20:36:42-07:00
actual_hours: 4.32
---

# conformance guard: the strict inversion was unpinned, and it broke a test that had pinned it

## Problem

#24 built `internal/conformance`: an absent external dependency SKIPS by default
and FAILS under `CONFORMANCE_STRICT`, so "green" cannot silently mean "did not
run". The behaviour was verified by running whole suites and reading their
output — and the package itself shipped with **no test of the inversion**.

It broke something immediately, and #24 did not see it:
`internal/llm/llmtest.SkipIfUnreachable` was routed through the guard, and its
own unit test `TestSkipsOnlyWhenNothingIsListening` asserts that an unreachable
service *skips*. Under strict it now fails instead, so that test **passed under
`go test` and failed under `CONFORMANCE_STRICT=1`, against unchanged code**.

#24 never caught it because its verification ran the conformance suite in a
sandbox, where the llm suites are absent-dependency skips either way. Running it
unsandboxed is what surfaced it.

The deeper fault is the same one in both places: **a test whose result depends on
ambient environment it does not control**, and **a rule with two modes pinned at
neither.**

## Spec

1. `TestSkipsOnlyWhenNothingIsListening` neutralizes `CONFORMANCE_STRICT` with
   `t.Setenv`, so it asserts the helper's reachability contract rather than
   whatever the environment happens to be.
2. The strict direction gets its own test at that call site — the half nothing
   pinned.
3. `internal/conformance` gets unit tests for `SkipOrFail` and `Strict` in both
   directions, including the `nil` error path and the empty-value-is-off case.

Out of scope, tracked below: the MIRROR half of the rule — an absent dependency
written as an unconditional `Fatal` — is still enforced by reading sites, not by
a test. `TestEverySkipIsRoutedOrWaived` sees `t.Skip*` only. #24's plan Risks
records this; designing the enforcement is a separate piece of thinking, not a
drive-by.

## Done when

- [x] `SkipOrFail` is pinned in BOTH directions — skip by default, fail under
      strict — for both the `err` and `nil` paths, mutation-verified.
- [x] The failure TEXT is asserted directly, not through a substitute
      `*testing.T`. (A substitute records *that* a test failed and exposes no
      reader for *why*, which is how the first version of this row degenerated
      into comparing `StrictEnv` with its own literal — PQ-1.)
- [x] **And the WIRING is pinned, not just the text.** Extracting `message()` made
      the string assertable and left a new seam uncovered: bypassing it inside
      `SkipOrFail` and formatting inline left `go test ./...` green in BOTH modes.
      `TestSkipOrFailPrintsTheMessage` re-execs the test binary and reads what
      `go test -v` actually printed — the only place the text is observable —
      and that bypass now reddens it. An extraction that makes a thing testable
      is not the same as testing it.
- [x] `Strict()` treats an empty value as off, pinned.
- [x] **The enumeration, stated rather than swept.** Every test that exercises a
      conformance-routed helper states which mode it asserts, so none inherits the
      ambient variable.

      **Stated as an invariant with its derivation, NOT as a count.** Three
      versions of this row carried a number — eight, then seven — and the number
      went stale twice: once because a table-driven test collapsed four rows into
      one substitute-`T`, and once because BR-1's own fix collapsed the
      `package llmtest` sites into `substituteT`. A count in a durable artifact is
      a restatement of a fact the code owns, and it drifts exactly like the prose
      and the line-number citations this repo has already learned about. Derive it:

      ```sh
      grep -rn "testing.T{}" internal/          # every substitute-T construction
      grep -rn "substituteT(t," internal/       # every site routed through the helper
      ```

      The invariant, which does not change when the counts do: **`package
      llmtest`'s goroutine sites all route through `substituteT(t, strict, fn)`,
      which takes the mode as a parameter** — so it cannot be forgotten, which a
      hand-written `t.Setenv` convention demonstrably could (BR-1 missed one of
      three). `internal/conformance`'s own test keeps the idiom inline because it
      is genuinely cross-package, and exporting a test helper from a production
      package to save five lines is the worse trade.

**Explicit non-goal: mechanical cross-mode enforcement.** Nothing here makes it
*impossible* to add a ninth site that flips verdict on the variable. "The two
modes agree" is a property of a RUN, not of the source, so enforcing it means
running the suite twice and diffing — which belongs in CI, not in a test that
would have to shell out to `go test` to assert it. Stating the class and its
boundary is the honest stopping point; pretending a one-shot two-mode read is a
guarded invariant is the thing this issue exists to stop doing.

## Plan

Single boundary — no `Mx` tags.

- [x] Split `message(reason, err, strict) string` out of `SkipOrFail`, leaving it
      thin `Fatal`/`Skip` glue. This is what makes the text assertable at all, and
      it also collapses the four-branch message assembly the #24 close review
      flagged.
- [x] `internal/conformance/message_test.go` (INTERNAL test — exporting
      `message` purely for a test would widen the API for a test's convenience):
      four rows, err × strict.
- [x] `internal/conformance/skiporfail_test.go`: both directions × err/nil, plus
      `Strict()`'s empty-value case.
- [x] `t.Setenv(conformance.StrictEnv, "")` in `TestSkipsOnlyWhenNothingIsListening`,
      and `TestStrictTurnsAnUnreachableServiceIntoAFailure` beside it.
- [x] Mutation table, each row reddening a named test.
- [x] Verify unsandboxed in both env states.

**ARCH-DRY — EXTRACTED, reversing what this section first recorded.** The
substitute-`T` idiom is now `substituteT` in `package llmtest`, covering every
goroutine site there (derive with `grep -c 'substituteT(t,' internal/llm/llmtest/*_test.go`).

The original note said a shared helper would force `golden_test.go` to import
`internal/conformance`, and that premise was simply false: `golden_test.go` and
`reachable_test.go` are the same package, so an unexported helper couples nothing
(BR-2). It also repeated PQ-3's "eight sites" that this issue's own Done-when had
already corrected to seven — and both numbers were wrong for the idiom itself. A decision
record whose reason is wrong is worse than none, because the next person inherits
the reasoning rather than re-deriving it.

The surviving half of the decision stands: `internal/conformance`'s own test keeps
the idiom inline, because it is genuinely cross-package and exporting a test
helper from a production package to save five lines is the worse trade.


## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187). Familiarity **1.0** — this
package was built in #24, in this same session, so `SkipOrFail`, the four
classes and the llmtest seam are all warm. No greenfield item: the package
exists; this adds a split function and tests.

Spec-quality ×0.2 on every code item — the plan pre-resolves the `message()`
signature, which test file each assertion lives in, the internal-vs-external
test choice, and the enumeration table. Implementation at v3.1's 40% of the
v2/v2.1 table. Design buffer **+15%**, per v2.1's rule of thumb: the ×0.2
discount is applied across every primitive, so the full +30% would double-count
the spec's thoroughness.

**TWO plan-quality rounds counted as SPENT, not budgeted.** Round 1 returned two
Importants — PQ-1 (the strict-message assertion could not fail, because a
substitute `*testing.T` exposes no reader for the text) and PQ-2 (a standing
invariant backed by a one-time two-mode read) — plus a Minor. Round 2 cleared.

**THREE close rounds budgeted, and that is where the uncertainty sits.** The diff
is small — one extracted function and three test files — but #24 measured that
the boundary review's thoroughness is a property of the GATE, not of the diff: it
budgeted four rounds and used four, on findings that were mostly about rules
rather than volume. Budgeting one round here because the change is small would
price the outcome I want rather than the one the evidence shows. Three, not four,
because #24's rounds were driven by a doc-sweep family and a widened contract
that this issue does not have.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=0.50 impl=0.08
item: milestone-review        design=0.10 impl=0.12
item: milestone-review        design=0.10 impl=0.12
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: milestone-review        design=0.15 impl=0.20
item: milestone-review        design=0.15 impl=0.20
item: milestone-review        design=0.15 impl=0.20
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
design-buffer: 0.15
total: 3.64
```
Item-to-task map, one row per task. `issue-spec` = the Problem/Spec/Done-when,
including the measured enumeration table — priced at the slug's 0.50 floor, as
#24 used, not the 0.25 a first draft wrote below the floor with no multiplier to
justify it. The two `design=0.10` rows are the plan-quality rounds actually
spent. **SIX** `smaller-go-module` rows for the six Plan tasks — the first draft
mapped five rows to six checkboxes and left "verify unsandboxed in both env
states" unpriced, which is precisely the step the Problem section indicts #24 for
skipping. The three `design=0.15` rows are the budgeted close rounds.

**The last two rows price REMEDIATION, and they are the honest lesson from #24.**
`milestone-review` prices *conducting* a round, not *fixing* what it finds — and
#24's four close rounds produced a repo-wide package extraction, an env-var
rename across the tree, a new pty test and three Minors, all real code, unpriced
in any review row. That is the most plausible source of #24's overrun (est 4.67,
actual 7.01, ratio 0.67). Budgeting two rounds' worth of remediation here is a
structural fix for a missing item, not a fudge factor.

**On carrying #24's ratio forward: deliberately NOT applied wholesale.** Every
per-primitive constant here is #24's, and #24 came in 50% under with the same
constants, same session, same model. Scaling 2.80 by that ratio gives ~4.2. I
have not done that, because v3.1 is a fleet calibration and one row is not a
mandate to rescale it — the calibration ledger is where that decision belongs,
across repos. What the row DOES justify is finding the missing item, which is the
remediation rows above; that moves the number to 3.64 for a stated reason instead
of a multiplier. If this issue also lands ~50% under, that is two rows pointing
at the same gap and the fleet constants should be revisited rather than patched
per-issue.

No `atlas-docs` row: #24 already documented this package's rule in
`atlas/define.md`, and this issue adds no surface a reader of the atlas would
need — the close's atlas gate should be waived with `--no-atlas`, not silently
satisfied.

## Log

### 2026-08-27
- 2026-08-27: closed — Close round 4. All four round-3 findings fixed. BR-6 (Important): the wiring test added in round 2 to pin that SkipOrFail routes through message() could not fail in one direction — its default-row want string is a PREFIX of the strict message, so the row passed whatever mode the child ran in. Measured on clean head, mutating SkipOrFail so every offline skip announces (CONFORMANCE_STRICT is set) left both packages and go test ./... green in BOTH env states; the same weakness meant the row could not detect its own CONFORMANCE_STRICT= override failing to beat an inherited =1, which is the ambient-environment defect this issue exists to fix, reappearing inside the test written to prove it fixed. Now asserted in both directions — suffix PRESENT under strict, ABSENT by default — and the mutation reddens the named row (verified: default mode 4 FAIL, go test ./... 3 FAIL, naming TestSkipOrFailPrintsTheMessage/default). BR-7: five was wrong in four places (the idiom covers six sites), including the sentence explaining why counts do not belong in this issue — fourth stale count here; the numbers are removed and the derivation commands remain. BR-8: substituteT doc claimed the mode is set for the duration of fn, but t.Setenv binds to the calling test and restores at its cleanup — doc corrected with the real scope. BR-9: CombinedOutput error was discarded so a spawn failure would report as a wiring bug — captured and reported with the transcript. Verification: gofmt clean, go vet both tag sets, go test ./internal/... all ok. --no-atlas: #24 already documented this package rule in atlas/define.md; gate considered and waived, not forgotten.; review verdict: FIX-THEN-SHIP

- 2026-08-27: shipped. Plan-quality cleared in 2 rounds (PQ-1: the strict-message
  assertion could not fail, because a substitute `*testing.T` exposes no reader
  for the text — the same vacuous shape #24's PQ-6 caught one issue earlier;
  PQ-2: a standing invariant backed by a one-time two-mode read; PQ-3: the
  substitute-`T` idiom, acknowledged and deliberately not extracted).
- Estimate revised 2.80 → 3.64 after the estimate-quality judge found three real
  gaps: a sixth Plan task unpriced (and it was the *verification* task, the very
  step this issue indicts #24 for skipping), `issue-spec` written below its 0.50
  floor with no multiplier, and — the substantive one — `milestone-review` prices
  CONDUCTING a review round, not REMEDIATING what it finds. #24's four close
  rounds produced a package extraction, a tree-wide env rename and a new pty
  test, all unpriced. Two remediation rows added. #24's 0.67 ratio deliberately
  NOT carried forward as a multiplier: one row is not a mandate to rescale a
  fleet calibration, but it did point at the missing item.
- Verified UNSANDBOXED in both env states, which is the method this issue exists
  to establish:
  - default: `internal/conformance`, `internal/llm`, `internal/llm/llmtest` all ok.
  - `CONFORMANCE_STRICT=1`: `llmtest` still **ok** (it FAILED here before this
    fix, against unchanged code) and the only failures are the three genuinely
    absent-dependency llm suites — no API key. No test changes verdict on the
    variable alone.
- **Mutation table, recorded by NAME.** The first version recorded bare red-counts
  (`4 / 4 / 7 / 9`) which did not reproduce — counting `--- FAIL` lines gives
  `4 / 3 / 3 / 8` depending on whether parent tests and `-v` subtests are counted,
  and no convention was stated (BR-3). The repo's own consolidated lesson says
  *read the FAILURE, not the count*, and this Log had recorded counts. Command:
  `go test ./internal/conformance/ ./internal/llm/llmtest/ -v`, restoring with
  `git checkout HEAD --` against the committed baseline `05dee92`:

  | mutation | tests reddened |
  |---|---|
  | strict message drops the variable name | `TestMessage/strict_names_the_variable…`, `TestMessage/no_cause_to_report,_strict` |
  | the cause is dropped from the message | `TestMessage/default,_with_a_cause`, `TestMessage/strict_names_the_variable…` |
  | strict never fails | `TestSkipOrFailBothDirections/strict_fails…`, `…/a_nil_err_still_fails_under_strict`, `TestStrictTurnsAnUnreachableServiceIntoAFailure` |
  | `Strict()` inverted | all four `TestSkipOrFailBothDirections` rows, `TestStrictTreatsAnEmptyValueAsOff`, `TestSkipsOnlyWhenNothingIsListening`, `TestStrictTurnsAnUnreachableServiceIntoAFailure` |

- Close round 1 (FIX-THEN-SHIP) raised BR-1..BR-5; all taken.
  - **BR-1** was this issue's own defect one level in: the Done-when states that
    every test exercising a conformance-routed helper controls the variable, and
    `reachable_test.go` had three such sites — the fix gave `t.Setenv` to two. The
    third is invariant TODAY only because a successful dial never reaches
    `SkipOrFail`, which is a property of control flow, not of the test. Fixed as
    the class: `substituteT(t, strict, fn)` bundles the goroutine with the env, so
    the mode is a parameter that cannot be forgotten rather than a convention that
    can. Every goroutine site in `package llmtest` routes through it.
  - **BR-2** caught the recorded reason for not extracting as simply WRONG — I
    wrote that a helper would force `golden_test.go` to import
    `internal/conformance`, but both files are `package llmtest`. Only
    `internal/conformance`'s own test is genuinely cross-package.
  - **BR-4** `wantFail` stated directly; **BR-5** resolved by the extraction, which
    removed the mis-grouped import from `reachable_test.go` altogether.
- Deferred, tracked in #24's plan Risks: the MIRROR half — an absent dependency
  written as an unconditional `Fatal` — is still enforced by reading sites.
  `guard_test.go` matches `t.Skipf?\(|t.SkipNow\(` and structurally cannot see a
  `Fatal`.

## Revisions

**2026-08-27 — BR-1's fix extracted the idiom, invalidating three recorded
claims at once.** Recorded as one entry rather than three patches, because they
share a cause: `substituteT` changed the shape the artifacts described, and every
claim written as a COUNT or a premise about package structure went stale with it.

- **The Done-when enumeration table and its `# 7` grep.** Rewritten as an
  invariant plus its derivation commands. The substitute-`T` constructions are
  now two (`substitute_test.go`, `skiporfail_test.go`) because the helper
  collapsed them; the invariant the table was stating is unchanged and now
  *stronger* — the mode is a parameter rather than a convention. Third stale
  count in this issue, which is why the row no longer carries one.
- **The ARCH-DRY paragraph.** Reversed. The helper WAS extracted, the
  "would force `golden_test.go` to import `internal/conformance`" premise was
  false (same package). The surviving half
  — `internal/conformance`'s test stays inline as genuinely cross-package — is
  kept and restated.
- **The failure-TEXT row.** Extended: `message()` being asserted did not pin that
  `SkipOrFail` CALLS it, and the review verified the bypass left `go test ./...`
  green in both modes. `TestSkipOrFailPrintsTheMessage` closes it by re-execing
  the test binary and reading the printed transcript. This was the deliberate
  stopping point the non-goal paragraph would otherwise have had to claim; it is
  cheaper to pin than to justify.

**2026-08-27 — close round 3 (BR-6 … BR-9).** The wiring test written in round 2
to fix a check that could not fail was itself a check that could not fail in one
direction.

- **BR-6 (Important).** `wiring_test.go`'s default row asserted
  `Contains(out, "network unavailable: dial refused")`, which is a PREFIX of the
  strict message — so the row passed whatever mode the child ran in. Measured on
  clean head: mutating `SkipOrFail` so every offline skip announces
  `(CONFORMANCE_STRICT is set)` left both packages and `go test ./...` green in
  both env states. It also meant the row could not detect its own
  `CONFORMANCE_STRICT=""` override failing to beat an inherited `=1` — the
  ambient-environment bug this entire issue is about, inside the test written to
  prove it fixed. Now asserted in both directions: the suffix must be PRESENT
  under strict and ABSENT by default. The mutation reddens the named row.
- **BR-7 (Minor).** "five" was wrong in four places — the idiom covers six sites —
  including the sentence explaining why counts do not belong in this issue. That
  is the FOURTH stale count here. The numbers are gone; the derivation commands
  remain. The surviving "five"s refer to lines of code and estimate rows, not
  sites.
- **BR-8 (Minor).** `substituteT`'s doc claimed the mode is set "for the
  duration"; `t.Setenv` binds to the CALLING test and restores at its cleanup,
  not when `fn` returns. Doc corrected with the real scope and what a two-mode
  test should do instead (two subtests).
- **BR-9 (Minor).** `CombinedOutput`'s error was discarded, so a spawn failure
  would surface as "SkipOrFail is not routing through message()" — a confident
  wrong cause. Captured and reported alongside the transcript.

**2026-08-27 — close round 4 (BR-10 … BR-12), taken under the FIX-THEN-SHIP
protocol before committing the close.** The gate converged; these three were
recorded past the round cap.

- **BR-10 (Important).** `t.Helper()` in `SkipOrFail` was unpinned — deleting it
  left both packages and `go test ./...` green in both modes, so the file:line a
  reader needs in order to find which check bailed was verified by reading and
  never by a test. Measured: with the call the transcript says
  `wiring_test.go:33`, without it `conformance.go:75`. Both directions now
  asserted, and the mutation reddens both rows.
- **BR-11 (Minor).** The re-exec selector was a literal copy of the test name, so
  a rename would make the child match zero tests and the assertion would report a
  wiring defect — the same misattribution BR-9 fixed for spawn failures. Derived
  from `t.Name()` now, and a child that ran nothing fails as a harness fault
  rather than as a wiring bug.
- **BR-12 (Minor).** `CONFORMANCE_STRICT=0` turns strict ON — set-vs-unset, the
  ordinary shell convention, but exactly what someone gets wrong when they mean to
  switch the mode OFF. It was pinned by a test whose name advertised only the
  empty case and documented nowhere. Now named in the test
  (`TestStrictIsSetVsUnsetSoEvenZeroTurnsItOn`) and tabulated on `StrictEnv`
  where a caller reads it.

