---
id: 000005
status: open
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# spaced-repetition scheduling engine (Leitner, pure)

## Problem

Reviewing everything every day does not scale and does not work. The schedule
decides what is worth the learner's attention today.

## Spec

Leitner boxes with fixed intervals (1, 3, 7, 14, 30, 90 days).

- Chosen over SM-2 deliberately: SM-2's ease factors cannot be explained in a
  stats screen, and for a personal tool "why is this due?" should be answerable
  in one sentence.
- Pure: `Due(word, now)`, `Promote/Demote(word, correct)`, `Mastered` = final box
  reached with N consecutive correct answers.
- Selection is also pure: given a deck, a clock and a daily budget, return
  today's queue — ordered by overdue-ness, then by lookup count (#4).
- No IO whatsoever; every test sets the clock explicitly.

## Done when

- [ ] Promotion/demotion and due-dates verified across a simulated multi-week
      schedule with a fake clock.
- [ ] A daily budget never returns more than the budget, and never starves an
      overdue word in favour of a fresh one.
- [ ] `Mastered` has one definition, used by both `--play` and `--stats`.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
