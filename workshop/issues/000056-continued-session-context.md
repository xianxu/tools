---
id: 000056
status: open
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
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

- [ ] Reproduce the continuity gap and inspect the actual requests, session
  lifecycle, history cutoff, and prompt instructions.
- [ ] Design the smallest change that makes the conversation the primary
  context, with an explicit bounded retention policy and regression scenarios.
- [ ] Implement, verify request contents and conversational behavior, and
  document the resulting session-context contract.

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
