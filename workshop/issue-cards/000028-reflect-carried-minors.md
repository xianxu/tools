---
id: 000028
status: open
created: 2026-08-28
updated: 2026-08-28
estimate_hours:
github_issue:
---

# reflect: eight carried Minors from #17's close

## Problem

`#17` (learner model) closed at M1 after eight gate rounds. Eight Minor findings
were recorded past the round cap and never disposed — none is a correctness
defect in shipped behaviour, which is why the issue converged without them, but
archiving them with `#17` would have dropped them silently.

They are carried here verbatim so they are a tracked list rather than a footnote
in `workshop/history/`. `dep: tools#17` records where they came from; the issue
is closed, so nothing blocks.

**Do not treat this as a batch to grind through.** Two have teeth and the rest
are cheap; the sensible trigger is *"`--reflect` is being touched again"*, at
which point the whole list costs little. Picking them off in isolation is likely
to cost more in gate rounds than the defects are worth — which is the lesson
`#17` itself paid 33.46h against an 8.39h estimate to learn.
