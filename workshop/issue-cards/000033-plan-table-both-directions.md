---
id: 000033
status: open
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
github_issue:
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
