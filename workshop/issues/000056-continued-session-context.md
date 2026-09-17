---
id: 000056
status: codecomplete
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T09:04:42-07:00
actual_hours: 0.84
---

# define: keep questions in a continued session conversation

## Problem

The operator experiences `define` LLM submissions as isolated questions rather
than a continued conversation. Exploring shades of meaning among synonyms
should not require restating the words and previous discussion with each input.

Operator request (2026-09-13): previous questions and answers should form the
core context of the session, so abbreviated follow-up questions can refer to
that context naturally.

Initial inspection shows that context is not entirely absent: `session.turns`
records Q&A pairs, `gatherAskContext` passes them into `askContext`, and
`renderAskPrompt` includes the latest six exchanges as a textual
"Earlier in this conversation" section. Investigate why this existing mechanism
does not deliver the desired continuity before choosing an implementation.
The operator's explanation that each input is submitted alone is a hypothesis,
not an established diagnosis.

## Spec

- Treat LLM questions within one running `define` session as turns in a shared
  conversation. Prior user questions and model answers are the primary context
  for understanding the next turn.
- Allow short follow-ups such as "which is more negative?", "can the second
  one be sincere?", and "give me examples of both" after a discussion of
  `unctuous` and `obsequious`, without retyping those words.
- Keep conversational order and distinguish user questions from model answers.
  Dictionary entries, looked-up words, and learner information supplement the
  conversation; they should not displace the referents established by it.
- Preserve continuity while looking up related words within the same session.
  Investigate the six-exchange cutoff and define a bounded context policy that
  preserves the ongoing topic through longer synonym explorations.
- Scope is continuity within a session. Persistence across application restarts
  is not requested. Choose the request representation and any context-budget
  mechanism during design, after tracing the current path.

## Done when

- A synonym discussion supports multiple abbreviated follow-ups that clearly
  refer to earlier questions and answers.
- The next outgoing request demonstrably carries the relevant prior questions
  and answers in order; tests inspect the real request through the LLM fake.
- Regression coverage includes intervening dictionary lookups, the current
  six-exchange boundary, and a fresh session with no previous conversation.
- The context retention/budget policy and its limits are documented; a
  repeatable semantic evaluation checks that referents are actually understood,
  beyond merely asserting that a transcript string was included.

## Plan

Replaced 2026-09-17 — see the second entry under `## Revisions`. The rows below
are the narrowed scope, not the original investigation.

- [x] raise `maxTurns` from 6 to 20 in `cmd/define/askctx.go`
- [x] pin the window in `TestRecentTurnsBoundsTheTranscript` with an expectation
      that does not read `maxTurns`
- [x] check whether any documentation states the limit

## Log

### 2026-09-13

- Captured at the operator's request as a separate feature task from #55
  (answer wrapping). Left open; implementation has not started.
- Initial code pointers: `cmd/define/session.go` (`recordExchange`),
  `cmd/define/ask.go` (`gatherAskContext`), and `cmd/define/askctx.go`
  (`recentTurns`, `maxTurns = 6`, `renderAskPrompt`, `askSystem`).
- ARCH-DRY: build on or repair the existing session transcript rather than
  introduce a second conversation store. ARCH-PURPOSE: evaluate natural
  follow-up understanding, not just the presence of history in a request.

### 2026-09-17
- 2026-09-17: closed — maxTurns 6→20 at askctx.go:59; TestRecentTurnsBoundsTheTranscript rewritten to pin the window independently of the constant — verified RED at 6 ("kept 6 turns, want 20") and green at 20; full cmd/define suite green (the two operation-not-permitted path tests are sandbox artifacts, re-run unsandboxed and passing). --no-atlas: no new architectural surface — the window was documented nowhere outside the code comment, and the number was deliberately NOT added to atlas/define.md to avoid a prose count drifting against the constant. --no-judge: operator waived the boundary review, one-constant change.; review verdict: not-run

Closed. `maxTurns` 6 → 20; the suite is green (the two `operation not permitted`
failures under the sandbox are exec-based path tests, re-run outside it and
passing — not this change).

The real work was the test, not the constant. `TestRecentTurnsBoundsTheTranscript`
derived EVERY assertion from `maxTurns`, including its loop bound `maxTurns + 3`
— so exactly three were always evicted and even the literal `"d"` was invariant.
It passed at 6 and at 20 alike, pinning nothing. Confirmed by measurement before
touching it: set the constant to 20 against the unmodified test, whole suite
green, and `git log -S` shows the probe never reached a commit. The rewritten
test reads `wantTurns = 20`, with `extra` kept independent so the two numbers
cannot move together and cancel out; verified RED at 6 ("kept 6 turns, want 20")
and green at 20. ARCH-DRY was the temptation to resist here — sharing the
constant between the code and its test is exactly what made the check consult
the thing it checks.

## Revisions

### 2026-09-13 — Narrow scope to twenty exchanges

The operator tested a follow-up ("tell me synonyms in this group") and confirmed
that existing conversational context works. The task's earlier premise is
superseded: no new conversation mechanism or failure investigation is required.
The desired change is to send the latest **20 complete question-and-answer
pairs**, up from six; the operator does not expect discussions longer than that.

This replaces the broader Spec, Done-when, and Plan above with the following:

- Change the existing `maxTurns` retention limit in `cmd/define/askctx.go` from
  6 to 20; retain chronological order, the current prompt format, and session
  scope. No summaries, cross-restart persistence, or new conversation store.
- Verify that all exchanges reach the prompt through turn 20, and that the 21st
  stored exchange evicts only the oldest from the next request. A shorter
  session remains unchanged. Use an independent expected count in the boundary
  test so deriving the test expectation from `maxTurns` cannot mask a bad limit.
- Update the documented limit and run the relevant context/request tests.
- Prior user questions and model answers remain the context for abbreviated
  follow-ups; the change only lengthens the existing window.

### 2026-09-17 — Plan rows replaced, and the narrowed scope executed

**Reason.** The 2026-09-13 revision said it replaced the Spec, Done-when and
Plan, but only the prose was replaced — the `## Plan` checkboxes were left
describing the superseded three-step investigation ("reproduce the continuity
gap", "design the smallest change"). Closing against them would have ticked
boxes for work nobody did, and the close gate's plan check would have been
asserting the wrong thing.

**Delta.**

- `## Plan` now carries the three narrowed steps that were actually performed.
  The original rows are preserved in this file's history, not rewritten in place.
- `maxTurns` 6 → 20 (`cmd/define/askctx.go:59`).
- The comment above `maxTurns` STAYS as written. Operator decision, 2026-09-17:
  20 is still bounded, so "short enough that a long session does not grow the
  prompt without limit" continues to hold. Recorded because the alternative was
  considered and declined, not overlooked — the per-turn size is unbounded
  (`recordExchange` stores the whole answer), so the policy is twenty TURNS
  rather than a token budget.
- **Documentation: no change needed, deliberately.** The six-turn window was
  stated nowhere outside that code comment — not in `cmd/define/README.md`, not
  in `atlas/define.md`, which describes the context as "the session's own earlier
  exchanges" and gives no count. The number was NOT added to the atlas: a count
  written in prose beside a constant is a second source of truth that nothing
  checks, which is the drift `render.go:57` already records having been bitten by
  twice. The atlas keeps the shape; the constant keeps the number.

