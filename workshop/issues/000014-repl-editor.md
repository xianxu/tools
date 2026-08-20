---
id: 000014
status: open
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
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

## Spec

zsh-style line editing in the REPL.

1. **History navigation.** Up/Down walk previously entered words, newest first.
2. **Prefix search.** With text already typed, Up walks only entries starting with
   it — zsh's `history-beginning-search-backward`, not a plain walk.
3. **Inline autosuggestion.** As you type, the most recent history entry with that
   prefix appears ahead of the cursor in grey; Right or End accepts it, any other
   key ignores it. Modelled on `zsh-autosuggestions`.

### The architectural decision this turns on

All three need **raw mode** — reading keystrokes rather than lines — which is a
different input layer from `#2`'s scanner, and drags in ANSI cursor control that
`#2` also listed as a non-goal.

Two ways, and the plan must pick one explicitly:

- **A dependency** (`chzyer/readline`, `peterh/liner`, `charmbracelet/bubbletea`).
  Cheap, and the first third-party runtime dependency in this repo — worth naming
  as such, since the repo currently ships with `x/term` and nothing else.
- **Own it.** A key-event decoder plus an editor state machine. More code, but the
  interesting part is small and this repo's whole test posture depends on being
  able to drive things from a string.

**Whichever is chosen, the editor state must be a pure state machine:**
`(state, keyEvent) → (state, render)`. Then history, prefix search and
autosuggestion are unit tests over key sequences with no terminal anywhere, and a
pty test covers only the raw-mode plumbing (ARCH-PURE). A design where these are
testable only through a pty should be rejected at plan review.

### Constraints carried from #2

- **Ctrl-C must still exit 0 with no temp files.** In raw mode Ctrl-C arrives as
  byte `0x03`, *not* a signal — so `#2`'s `signal.NotifyContext` contract does not
  fire and the equivalent behaviour has to be re-established deliberately. This is
  the single most likely regression.
- Ctrl-D on an empty line ends the session, as it does today at EOF.
- A bare return still replays audio and **writes nothing to stdout**.
- Non-TTY stdin keeps the current line-reading path unchanged — `echo w | define`
  must not enter raw mode. Two input layers, one loop.
- Grey suggestion text degrades on terminals without colour, and must never be
  submitted when the user presses return without accepting it.

### Still out of scope

Multi-line editing, kill-ring, incremental reverse search (Ctrl-R), vi mode.

## Done when

- [ ] Up/Down walk history; with a prefix typed, they walk only matching entries.
- [ ] The grey suggestion is never submitted unless explicitly accepted.
- [ ] History persists across sessions (see `#15` for the shared source).
- [ ] Ctrl-C exits 0 leaving no temp files, verified through a real pty — the
      raw-mode regression risk.
- [ ] `echo word | define` behaves exactly as it does today.
- [ ] The editor's behaviour is unit-tested over key sequences, not only via pty.
- [ ] Any new third-party dependency is named and justified in the plan.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
