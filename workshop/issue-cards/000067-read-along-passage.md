---
id: '000067'
status: done
started: 2026-09-16T12:50:20-07:00
created: 2026-09-16
updated: 2026-09-17
estimate_hours: 7.78
actual_hours: 11.11
---

# define: read-along — paste a passage, click or drag what is opaque

## Problem

The operator's use of `define` has outgrown the word as the unit. Reading
astronomical prose they looked up `zenith`, `nadir`, `meridian`, then asked
*"what's progression in this context"* — a question about the passage, which
`define` cannot see.

The answer (observed 2026-09-16) opens with *"I can't see the sentence you're
reading, so this is a best guess from what's around it"* and then spends three of
four paragraphs hedging across an astronomical reading, an ordinary-use reading,
and a guess keyed off the word on screen. One line of the actual source would
collapse all of it into a single confident sentence.

The session's lookups are only a SHADOW of the passage. `askContext` carries the
deck, the learner model, the current entry and this session's words — everything
except the text that prompted the question.

Two further observations from the same answer:

- It highlighted four deck words (`equinox`, `synodic`, `granulation`,
  `epithelium`) inside its own prose, because `highlightWriter` wraps the answer
  stream (atlas §Highlighting, M2/M3). Each is an incidental retrieval event.
  Explanation-in-context is ALREADY a reinforcement surface; it is simply aimed
  at the wrong context.
- It flagged `progression` vs `precession` as a probable misreading. Sense and
  form disambiguation against a real sentence is the thing a dictionary app
  structurally cannot do, and it is the value this issue is for.
