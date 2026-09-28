---
id: '000048'
status: done
started: 2026-09-07T23:50:42-07:00
created: 2026-09-07
updated: 2026-09-08
estimate_hours: 3.60
actual_hours: 3.40
---

# /play: a sitting without leaving the loop

## Problem

**You have to leave the program to review.** `define --play` is a mode: it runs
from a shell, takes the terminal, and exits. So a session that starts as "look a
few things up" and turns into "actually, let me review" costs a quit, a
re-invocation, and — when the sitting ends — another invocation to get back to
looking things up.

The loop is where a learner already is. The sitting should be reachable from it.
