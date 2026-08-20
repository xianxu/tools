---
id: 000014
status: working
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours: 3.59
started: 2026-08-20T16:25:33-07:00
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

### Inherits a cooked-mode workaround

`#2` ended up doing cursor arithmetic in **cooked mode** to place the "♫ playing
N×" indicator over the prompt: the terminal echoes Enter onto a new line, so the
loop steps back over that echo and over the prompt before drawing, then erases
and redraws. It works and is tested, but it is arithmetic against a terminal that
has already moved the cursor.

**It is also only correct while the user waits.** `eraseLine` acts on the line the
cursor is on *now*; the loop blocks inside playback for seconds with ECHO on, so
a second impatient Return is echoed by the driver, moves the cursor, and the
post-playback erase clears the wrong line — stranding the indicator. Cooked-mode
echo is what makes the arithmetic breakable, and no amount of care inside the
loop fixes it.

In raw mode this disappears: nothing is echoed, so the frame is simply rendered
with the indicator where the prompt would be. **Expect to delete
`eraseLineAndStepBack` and the `skipPrompt` bookkeeping here** rather than port
them — if this issue keeps them, the render model is wrong.

### Still out of scope

Multi-line editing, kill-ring, incremental reverse search (Ctrl-R), vi mode.

## Done when

- [x] Up/Down walk history; with a prefix typed, they walk only matching entries.
- [x] The grey suggestion is never submitted unless explicitly accepted.
- [x] History is consumed through a `History` seam, so `#3`'s store satisfies
      persistence without this issue inventing a private history file.
- [x] Ctrl-C at the prompt exits 0 with the terminal restored to cooked mode.
- [x] Ctrl-C **during playback** cancels and prints nothing — `#2`'s contract,
      which `signal.NotifyContext` can no longer deliver because raw mode makes
      Ctrl-C a byte rather than a signal.
- [x] `echo word | define` behaves exactly as it does today.
- [x] The editor's behaviour is unit-tested over key sequences, not only via pty.
- [x] No new **runtime** dependency; `github.com/creack/pty` is test-only and
      named as such.
- [x] History records every submitted line with a found flag — Up-arrow recalls
      typos (that is when you want to edit and retry), while `#15`'s `/history`
      filters to found.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.3  impl=0.04
item: greenfield-go-module     design=0.4  impl=0.4
item: smaller-go-module        design=0.2  impl=0.28
item: tui-screen               design=0.4  impl=0.6
item: real-api-discovery       design=0.0  impl=0.24
item: milestone-review         design=0.0  impl=0.12
item: milestone-review         design=0.0  impl=0.12
item: milestone-review         design=0.0  impl=0.12
item: atlas-docs               design=0.05 impl=0.12
design-buffer: 0.15
total: 3.59
```

Derivation notes:

- **greenfield-go-module** is the editor state machine + key decoder: genuinely
  new, single concern, no prior art in this repo. Design takes the ×0.2
  spec-quality discount (2.0 → 0.4) — the plan fixes the `Key` vocabulary, the
  `Apply` signature and the `Action` split.
- **tui-screen** is the right primitive for the raw-mode half: terminal state,
  a frame renderer, and a key loop. Design **not** discounted — raw-mode restore
  on every exit path is the part that bites, and the plan can specify it but not
  retire it.
- **smaller-go-module** covers deleting `#2`'s cooked-mode workaround and rewiring
  the loop: well-specced, mirror-shaped.
- **Two `milestone-review` items**, not one. `#1` closed at 4.67h against 2.75
  and `#2` at 2.22 against 0.81, both from review rework that the model prices at
  one pass. This is the first estimate in this repo to budget a second round
  explicitly rather than discover it.
- `familiarity: 1.0` — same package and fakes; raw mode is new but `x/term` is
  already in use.

**Revised up after the estimate-quality judge (post-gate, recorded rather than
hidden).** It argued 2.57 priced three things as mechanical that are not:
`smaller-go-module` covers moving cancellation into a reader goroutine and
keeping `signal.NotifyContext` alive for the non-raw paths — the issue's own
"single most likely regression", not a port; and nothing budgeted the raw-mode
and pty discovery a first cgo-free terminal integration needs. Added
`real-api-discovery` (the terminal *is* the external surface here) and lifted
`smaller-go-module` off its floor.

Also recording, since a later recalibration cannot otherwise tell it happened:
**the library-availability check ran** — readline, liner and bubbletea were each
considered and rejected with reasons in the plan, so v2.1's from-scratch clause
applies and no halving is taken.

Σdesign 1.35 × 1.15 = 1.5525; Σimpl 2.04 × 1.0 = 2.04; total **3.59**. Both prior v3.1 rows in this repo
under-estimated (`#1` 2.75→4.67, `#2` 0.81→2.22); this is the first to correct
*before* the fact rather than after.

## Plan

See `workshop/plans/000014-repl-editor-plan.md`.

- [x] `decodeKey` — the escape-sequence vocabulary, pure, fuzzed.
- [x] `Editor`/`Apply` — the state machine, no IO.
- [x] History walk + prefix search behind a `History` seam.
- [x] Inline autosuggestion (Enter must never submit it).
- [x] `RenderLine` — a whole frame, so `#2`'s cursor arithmetic can be deleted.
- [x] Raw mode + loop rewire; delete `eraseLineAndStepBack` and `skipPrompt`.

## Revisions

### 2026-08-20 — Done-when narrowed before implementation

"History persists across sessions" now reads as "history is consumed through a
`History` seam". The persistent implementation is `#3`'s store, and `#15` states
that `/history` must read the store rather than a private history file — so
satisfying persistence *here* would mean building the thing `#15` forbids. The
seam is the deliverable; `#3` fills it. Sequencing is therefore **#14 → #3 → #15**.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-20 — implementation notes

**The plan gate's Critical was real and my first implementation still failed it.**
PQ-1 said raw mode kills the Ctrl-C-during-playback path. I moved cancellation
into the key reader as instructed — and it still hung, because I restored *cooked*
mode around the whole lookup. Playback is the part that blocks for seconds, so
during it the key reader saw nothing and the byte was swallowed by the line
discipline. Fixed by splitting `lookupAndRender` out of `defineOnce`: **render
cooked, play raw.** Measured on a pty: hang → exit 0 in 0.61s.

**The stdin/stdout family bit a fourth time, during this issue.** Restoring the
fallback loop's prompt, I gated it on `interactive` — stdin — while writing to
stdout, and `TestREPLPromptRequiresBothStreams` caught it immediately. `replLines`
now takes `pipedInput` and `showPrompt` as separate parameters with a comment
explaining why they must never be one flag.

**`creack/pty` is a test-only dependency**, used solely by
`pty_conformance_test.go`. The runtime dependency set is unchanged: `x/term`.
