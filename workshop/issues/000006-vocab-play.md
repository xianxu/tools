---
id: 000006
status: working
deps: ["tools#3", "tools#5"]
github_issue:
created: 2026-08-20
updated: 2026-08-27
estimate_hours:
started: 2026-08-27T08:15:55-07:00
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
- [ ] **A full session runs with the LLM seam unavailable**, falling back to the
      forms that need neither key nor network (2.1 here, 2.3 in #7) rather than
      failing. Relocated from #11 on 2026-08-22: it names `--play`, so it belongs
      to the issue that owns `--play`. Asserted with a client returning
      `llm.ErrUnavailable`, not by unsetting an env var — the point is that the
      loop degrades, not that config resolution does.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
