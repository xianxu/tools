---
id: '000056'
status: done
started: 2026-09-17T09:04:42-07:00
created: 2026-09-13
updated: 2026-09-17
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
