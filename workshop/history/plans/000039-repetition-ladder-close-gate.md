---
gate: boundary-review
issue: 39
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-31T12:30:23-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: T7's load number has no named path from the deck and fold to finish()
          detail: |-
            2nd in family — the rule, not the instance: every number T7 prints must name
            its full input path. D11 sourced the budget to opt.count; the load's inputs
            are still unrouted. finish is finish(w io.Writer, s play.Session) int
            (play_loop.go:365) with five callers in playSession; todaysQuestions computes
            deck and Fold(events) at play_loop.go:249 and returns neither, and runPlay
            discards them at :31. Say whether T7 widens todaysQuestions' return and
            threads the values (four test call sites move too) or re-reads inside finish
            (ARCH-PURE-wrong). The estimate's 0.02/0.08 "one line, one test" assumes the
            former is free.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: unsourced-input
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-31T12:30:23-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
---

# Gate ledger — tools#39 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T12:30:23-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `unsourced-input` T7's load number has no named path from the deck and fold to finish()
  2nd in family — the rule, not the instance: every number T7 prints must name
  its full input path. D11 sourced the budget to opt.count; the load's inputs
  are still unrouted. finish is finish(w io.Writer, s play.Session) int
  (play_loop.go:365) with five callers in playSession; todaysQuestions computes
  deck and Fold(events) at play_loop.go:249 and returns neither, and runPlay
  discards them at :31. Say whether T7 widens todaysQuestions' return and
  threads the values (four test call sites move too) or re-reads inside finish
  (ARCH-PURE-wrong). The estimate's 0.02/0.08 "one line, one test" assumes the
  former is free.
  (carried from plan-quality PQ-8, deferred to the boundary review)

## Round 2 — 2026-08-31T12:30:23-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-1** [Minor] `unsourced-input` T7's load number has no named path from the deck and fold to finish()
