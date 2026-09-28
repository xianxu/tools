---
id: 000068
status: blocked
started: 2026-09-17T10:56:31-07:00
created: 2026-09-16
updated: 2026-09-17
estimate_hours:
github_issue:
---

# define: use real sentences as cloze material — finish the `usage/` thread

## Problem

`define` caches real sentences for a word and never reads them.

`usage/<slug>.yaml` holds `store.NewsItem` records from Google News RSS, and
`entryUsages` supplies NOAD's own examples; `bothSources` (`usage.go:153`) joins
them behind one seam and is constructed into `deps` at `main.go:280` and `:389`.
**`Usages` has no caller** (`usage.go:161` — the only references are the interface,
the implementation and its own helper). The atlas says plainly: *"Nothing
user-facing consumes it yet, deliberately. #10's authoring step is the consumer."*
That consumer was never built.

Meanwhile `--harvest` pays a model call to AUTHOR a sentence for every deck word
(`renderAuthorPrompt`, `harvest_item.go:351`), then pays another to entailment-judge
it (`harvest.go:369`), for words that may already have real sentences sitting in
`usage/`.

#67 surfaced a third source: a passage the learner actually read and marked a word
in. It was deliberately kept out of #67 so this issue can serve all three sources
with ONE consumer rather than accumulating a second stalled producer.
