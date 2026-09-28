---
id: '000021'
status: done
started: 2026-08-26T12:40:00-07:00
created: 2026-08-26
updated: 2026-08-26
estimate_hours: 9.28
actual_hours: 6.0
---

# highlight the words you are learning wherever they appear

## Problem

Nothing on screen distinguishes a word the learner has studied from a word they
have never met. A definition of `obsequious` may use `sycophantic` in its own
gloss — the exact connection worth noticing — and it reads the same as every
other word on the line. An LLM answer comparing two words the learner looked up
last week gives no sign that those two are already theirs.

Operator's framing:

> use different color ... for the words user checked ... this makes them easier
> to spot ... such highlighting should appear in definition and LLM responses as
> well, this help reinforce words user are learning.

The reinforcement only works if it is everywhere text appears. Highlighting only
the prompt would mark words at the moment the learner already knows they are
typing them — the least informative moment.
