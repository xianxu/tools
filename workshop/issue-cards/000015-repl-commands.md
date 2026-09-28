---
id: '000015'
status: done
started: 2026-08-21T13:55:32-07:00
created: 2026-08-20
updated: 2026-08-21
estimate_hours: 1.82
actual_hours: 6.77
---

# REPL command mode: /-prefixed commands with type-ahead, starting with /history

## Problem

Once the REPL has an editor, it needs a way to do things that are not "define
this word" — and a namespace that cannot collide with a word being looked up.
