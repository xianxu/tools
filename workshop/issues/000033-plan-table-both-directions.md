---
id: 000033
status: open
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
---

# the plan-status guard checks rows against the tree but not the tree against rows

## Problem

Carried from `#29`'s close review (BR-11, Minor, third in family
`plan-table-vs-tree`).

`#29` added `TestPlanTableStatusMatchesTheChangeWindow` so a Core-concepts row
claiming `unchanged`/`modified` is checked against `git merge-base main HEAD` at
declaration level. It works, and it caught four wrong rows. It is also only half
a projection.

**Two holes, both demonstrated by the reviewer rather than argued:**

1. **A row naming an UNTOUCHED file is not checked at all.** `changedLines`
   returns nil for such a file and the loop `continue`s past the whole row, so
   adding `| crlfWriter | cmd/define/crlf.go | modified |` to the `#29` plan left
   both plan guards green. The `touched == nil` case should reach
   `checkPlanStatus` as `inWindow = false`, where `modified` is already an error.
2. **Nothing checks tree → row.** `voice.langOrDefault`, `isASCIIOnly` and
   `nothingToReplay` are all new in `#29`'s window with no table row, while the
   `AudioCandidates` row names `langOrDefault` in its own status text. A plan
   whose table omits half the symbols it introduced is as misleading as one whose
   rows are wrong — and `superpowers-writing-plans` calls the table "the
   load-bearing surface".

**And the guard's own error branches have no fixtures** — the same
green-when-removed state round 3 found in `planStatus`, one level up. `#29`'s
close review widened the rule to cover exactly this: *a finding-fix with no test
that reddens without it is not addressed.*

## Spec

Not designed. The obvious shape — enumerate every top-level declaration the
window adds and require a row — needs a decision about what a table is FOR
before it is built:

- **Every new symbol, or every new CONCEPT?** `#29` introduced `isASCIIOnly` as a
  three-line helper beside `SourceSpellings`. A table with a row for it is
  noisier and no more informative; a table without it is incomplete by the rule
  above. The `superpowers-writing-plans` guidance says entities, not symbols, and
  a mechanical check cannot tell the difference — so the answer is probably a
  waiver marker rather than a stricter rule, and that is a design question.
- **Unexported helpers vs. exported surface?** `nothingToReplay` is a const that
  exists purely to stop two loops drifting. That is arguably exactly what a table
  should record.

## Done when

- [ ] A row naming a file this window did not touch is CHECKED, not skipped —
      `modified` on an untouched declaration fails.
- [ ] The tree → row direction exists, with a decided and documented answer for
      helpers that do not deserve a row.
- [ ] Every error branch in the guard is entered by a fixture, so removing it
      reddens something.
- [ ] The rule lands once: `TestPlanTablesNameEntitiesThatExist` and
      `TestPlanTableStatusMatchesTheChangeWindow` should not grow a third
      spelling of the same parse (they already shared one locator only after a
      review found the second copy had diverged).

## Plan

- [ ] Claim, then design via `sdlc start-plan`.

## Log

### 2026-08-29

Filed from `#29`'s close review round 4. Third finding in this family across one
issue; the first two were fixed as instances and as a mechanism respectively, and
this is the mechanism's own gap.
