---
id: 000037
status: open
deps: []
github_issue:
created: 2026-08-30
updated: 2026-08-30
estimate_hours:
---

# fuzz targets and pty rows run in nothing automated, so the tests that would catch a Critical never do

## Problem

Raised as `BR-56` at `#30`'s M2 boundary, and the evidence is that issue's own
Critical.

**`BR-51` was a reachable panic that this repo's own fuzzer finds in under a
second** — `Render` crashed on a blank or single-space entry — and it shipped
through eleven review rounds. It was found by a reviewer typing `-fuzz` by hand.
Worse, the fix for it added a fifteenth fuzz target with exactly the same
property: nothing runs it either.

Measured 2026-08-30:

- `go test ./...` exercises a fuzz target against its SEED CORPUS only. A
  `f.Fuzz` body never explores unless `-fuzz` names it, one target per run.
- No `-fuzz` invocation exists in `Makefile`, `Makefile.local`,
  `Makefile.workflow`, `scripts/`, or `.github/workflows/merge-check.yml`.
- The 12 `TestPTY*` rows need `-tags conformance` AND a real pty. In the review
  environment every one reports "no pty available: operation not permitted", so
  `#30`'s Done-when rows 6 and 8 were certified by their in-process counterparts
  alone.

So two whole categories of test exist, are well written, and defend nothing on
any automated path.

## Spec

Not designed. The shape is a scheduling question, not a testing one — the tests
are already there.

**Fuzzing is not a suite step.** `-fuzz` runs ONE target and runs it until told
to stop, so it cannot simply join `go test ./...`. The options are a periodic job
(a fixed budget per target, corpus committed so findings become seeds), a
merge-gate step with a short budget, or a `make fuzz` an operator runs
deliberately. The corpus growing on disk is the durable half either way — a
finding that becomes a seed is a regression test forever after.

**The pty rows are a different failure.** They are correctly written to
`SkipOrFail`, so an environment without a pty degrades rather than lying. But
that means the honest signal — "these rows did not run" — is currently invisible
at the moment it matters, which is a boundary review. `CONFORMANCE_STRICT`
already exists to turn a skip into a failure; the question is where it is set.

**A guard is probably the deliverable**, in this repo's own idiom: a test that
enumerates fuzz targets and asserts each is named by whatever runs them, so a
sixteenth cannot be added into the same silence. That is the same move
`TestEveryEnabledMouseModeIsDecoded` and `TestAtlasDescribesEveryRegionKind`
make — the SET has one owner and the check derives from it.

## Done when

- [ ] Every `Fuzz*` target is exercised beyond its seed corpus by something that
      runs without a human remembering to type a flag.
- [ ] Adding a new fuzz target that nothing runs FAILS, rather than joining the
      silence — derived from the set of targets, not a list someone maintains.
- [ ] The 12 pty rows either run somewhere automated, or their not-running is
      loud at the boundary rather than reported as a skip nobody reads.
- [ ] A finding becomes a committed seed, so it is a regression test rather than
      a story about one afternoon.

## Plan

- [ ] Decide where fuzzing runs and on what budget.
- [ ] Decide where `CONFORMANCE_STRICT` is set, and what a boundary review should
      see when a pty is unavailable.

## Log

### 2026-08-30

Filed from `#30`'s M2 boundary. The specific sequence worth keeping: the repo has
fifteen fuzz targets written with real care — `FuzzDecodeMouseIsBounded` alone
found two genuine decoder defects when run by hand this session — and a Critical
still shipped because none of them run on their own. The tests were not missing;
the schedule was.
