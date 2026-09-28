---
id: '000024'
status: done
started: 2026-08-27T15:35:13-07:00
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 4.67
actual_hours: 7.01
---

# grade before reveal: y advances, n shows the definition

## Problem

Every word costs two keystrokes, and one of them carries no information.

The session shows a word, waits for Enter or space to reveal, and only then
accepts `y`/`n`. So a word the learner knows cold still costs a reveal they did
not need — and the reveal is the slow step, because it fetches and plays the
pronunciation and prints the whole definition.

The operator's words: *"when a word is displayed, we should display the
following choices first, and when user chose `n`, run the definition. one less
step and smoother."*

