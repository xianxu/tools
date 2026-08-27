---
id: 000024
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours:
started: 2026-08-27T15:35:13-07:00
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


## Spec

The prompt shown WITH the word becomes the grading prompt:

```
y = got it, n = missed it, d = remove from deck, Ctrl-C to stop
```

- `y` records Correct and moves straight to the next word. No reveal, no audio.
- `n` records Wrong AND reveals — the definition is what a miss earns, and the
  pronunciation plays as it does today.
- `d` and Ctrl-C are unchanged.
- Space/Enter still reveal without grading, for a learner who wants to check
  before rating. Unadvertised in the prompt line but kept, because after a reveal
  the existing `y`/`n` path is exactly what it is today.

**The design position being reversed.** `session.go` currently refuses to grade
before a reveal, and argues it: *"A learner cannot rate what they have not
seen."* That is true of a recognition test and false of a RECALL test, which is
what form 2.1 is. The learner is rating their own recall, which they know before
they check; the definition is FEEDBACK, not stimulus. Getting this backwards is
what put a mandatory step in front of every correct answer.

**The mechanism this needs.** `n`-before-reveal owes the loop two effects —
record the miss, and reveal — while `Apply` returns one `Outcome`, and the loop
is forbidden from inspecting a verdict to infer the second (that ban is load-
bearing: it is what keeps the skip rule in one place). So this is a real change
to the session contract, not a re-ordering of prints.


## Done when

- [ ] `y` on an unrevealed word records Correct and advances, with no reveal and
      no audio — asserted by a player double that fails the test if called.
- [ ] `n` on an unrevealed word records Wrong AND reveals, in that order, and the
      recording happens before the next draw (Ctrl-C stays lossless by
      construction, not by a flush).
- [ ] Space and Enter still reveal without grading; `y`/`n` after a reveal behave
      exactly as they do today.
- [ ] `d` before or after a reveal still drops and records nothing.
- [ ] The loop still never inspects a verdict.
- [ ] The prompt line, README and atlas all show the new keys — verified by grep,
      not by memory (#6 BR-44/BR-48).
- [ ] A pty conformance test drives the new flow on a real terminal (#6 BR-45:
      `--play` shipped its one defect because it had none).


## Plan

- [ ]

## Log

### 2026-08-27
