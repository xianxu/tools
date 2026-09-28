---
id: '000054'
status: done
started: 2026-09-12T17:16:34-07:00
created: 2026-09-12
updated: 2026-09-14
estimate_hours: 2.75
actual_hours: N/A
---

# the TUI keeps practice material current in the background, so new words get cloze questions without a command

## Problem

Cloze questions exist only for words `define --harvest` has written a practice
sentence for, and nothing runs `--harvest` for you: `runHarvest` is reachable only
from its flag (`main.go:860`). The learner model is the same, written only by
`define --reflect` (`main.go:845`). A user who never learns those commands exist,
which is the likely case, sees only multiple choice and boards forever. A user who
does learn them has to remember to run them after every batch of lookups.
Operator, 2026-09-12: "user won't remember this."
