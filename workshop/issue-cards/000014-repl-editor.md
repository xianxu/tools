---
id: '000014'
status: done
started: 2026-08-20T16:25:33-07:00
created: 2026-08-20
updated: 2026-08-20
estimate_hours: 3.59
actual_hours: 1.27
---

# REPL line editor: history, prefix search, inline autosuggestion

## Problem

`#2` shipped the REPL on `bufio.Scanner`, which reads whole lines and sees no
keystrokes. Up-arrow prints `^[[A`. Retyping `sycophantic` for the fourth time is
the actual daily friction.

**This deliberately lifts a non-goal.** `#2` ruled out line editing, history and
completion, and said: *"the moment this needs a line editor it needs a dependency,
and that is a separate decision."* This issue is that decision (operator,
2026-08-20). Recording it so the reversal reads as intent, not drift.
