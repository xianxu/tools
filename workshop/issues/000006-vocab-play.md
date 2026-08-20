---
id: 000006
status: open
deps: ["tools#3", "tools#5"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# define --play: review loop + form 2.1 quick pass

## Problem

The deck and the schedule are inert without a way to sit down and review.

## Spec

`define --play` runs today's queue.

- Form 2.1 (this issue): show the word, let the learner recall, reveal the
  definition, self-rate right/wrong. No question generation, no network.
- A `Question` interface every later form implements, so forms drop in without
  touching the loop.
- Terminal-shaped and single-threaded; interruptible at any point. Events are
  written **as they happen**, so Ctrl-C mid-session keeps what was reviewed —
  free from the append-only log (#3).
- Playing the pronunciation during review is on by default; `--no-audio` applies.
- A bad-question keypress records a flag against the question (groundwork for
  the generated forms, which need this feedback loop).

## Done when

- [ ] A full session runs against fake store + fake clock, recording one event
      per answer.
- [ ] Interrupting mid-session preserves already-recorded events.
- [ ] Adding a second form requires no change to the loop.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
