---
gate: boundary-review
issue: 55
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-13T13:01:57-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Wrapped highlighted phrases lose styling when their opening row leaves the viewport.
          detail: cmd/define/answerwrap.go:107 inserts a newline without restoring active SGR state on the continuation row. screen.Paint paints only visible rows (cmd/define/screen.go:530), so a viewport starting at that continuation displays it without its highlight, violating the styling-preservation contract (ARCH-PURPOSE). Reuse sgrState to preserve styling independently across wrapped rows and add a viewport-paint regression covering a multiword highlighted phrase.
          family: viewport-independent-styling
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T13:09:45-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: cmd/define/answerwrap.go:91 and :111 replay shared SGR state after explicit and inserted newlines. TestAnswerWrapContinuationStyle at cmd/define/answerwrap_test.go:74 covers actual highlighted phrases, enclosing styles, and closing resets through viewport painting. All four cases pass at HEAD and fail when both replay calls are removed in a scratch copy.
          round: 2
      blocked: false
---

# Gate ledger — tools#55 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T13:01:57-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `viewport-independent-styling` Wrapped highlighted phrases lose styling when their opening row leaves the viewport.
  cmd/define/answerwrap.go:107 inserts a newline without restoring active SGR state on the continuation row. screen.Paint paints only visible rows (cmd/define/screen.go:530), so a viewport starting at that continuation displays it without its highlight, violating the styling-preservation contract (ARCH-PURPOSE). Reuse sgrState to preserve styling independently across wrapped rows and add a viewport-paint regression covering a multiword highlighted phrase.

## Round 2 — 2026-09-13T13:09:45-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — cmd/define/answerwrap.go:91 and :111 replay shared SGR state after explicit and inserted newlines. TestAnswerWrapContinuationStyle at cmd/define/answerwrap_test.go:74 covers actual highlighted phrases, enclosing styles, and closing resets through viewport painting. All four cases pass at HEAD and fail when both replay calls are removed in a scratch copy.

## Open findings

(none — every finding has been disposed)
