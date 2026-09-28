---
id: 000037
status: open
created: 2026-08-30
updated: 2026-08-30
estimate_hours:
github_issue:
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
