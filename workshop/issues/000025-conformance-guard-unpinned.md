---
id: 000025
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 3.64
started: 2026-08-27T20:36:42-07:00
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
- [x] `Strict()` treats an empty value as off, pinned.
- [x] **The enumeration, stated rather than swept.** Every test that exercises a
      conformance-routed helper controls `CONFORMANCE_STRICT` itself, so no test
      changes verdict on the ambient variable. The class is the **seven**
      substitute-`*testing.T` sites — COUNTED, not carried over from the finding
      that named the class (which said eight; the table-driven form collapses
      `SkipOrFail`'s four rows into one substitute-`T`):

      | site | count | routed through the guard? |
      |---|---|---|
      | `reachable_test.go` | 3 | YES — `SkipIfUnreachable` |
      | `skiporfail_test.go` | 1 | YES — is the guard |
      | `golden_test.go` | 3 | NO — `golden.go` has no conformance reference |

      ```sh
      grep -h "testing.T{}" internal/llm/llmtest/*_test.go internal/conformance/*_test.go | wc -l   # 7
      ```

      Verified by `grep -c conformance internal/llm/llmtest/golden.go` → 0, so
      the golden sites are outside the invariant rather than unchecked.

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

**ARCH-DRY, acknowledged and NOT extracted.** The goroutine + substitute-`T` +
done-channel idiom reaches eight sites across three files (PQ-3). A shared helper
would need a test-support package that all three import, which would make
`golden_test.go` — whose sites have nothing to do with conformance — depend on
the conformance package for a five-line idiom. The coupling is worse than the
duplication; recorded here so the next person meets a decision rather than an
accident.

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
- Four mutations, each reddening a named test: strict message drops the variable
  name (4 red), the cause is dropped (4), strict never fails (7), `Strict()`
  inverted (9).
- Deferred, tracked in #24's plan Risks: the MIRROR half — an absent dependency
  written as an unconditional `Fatal` — is still enforced by reading sites.
  `guard_test.go` matches `t.Skipf?\(|t.SkipNow\(` and structurally cannot see a
  `Fatal`.
