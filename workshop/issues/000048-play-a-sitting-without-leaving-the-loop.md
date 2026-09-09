---
id: 000048
status: working
deps: ["tools#6"]
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours: 3.60
started: 2026-09-07T23:50:42-07:00
---

# /play: a sitting without leaving the loop

## Problem

**You have to leave the program to review.** `define --play` is a mode: it runs
from a shell, takes the terminal, and exits. So a session that starts as "look a
few things up" and turns into "actually, let me review" costs a quit, a
re-invocation, and — when the sitting ends — another invocation to get back to
looking things up.

The loop is where a learner already is. The sitting should be reachable from it.

## Spec

**`/play` runs today's sitting and returns to the prompt.** Ctrl-C ends the
sitting the way it always has, and lands back at the definition prompt rather
than at the shell. So does finishing the queue.

### What makes this more than a command-table row

Three things are already true, and they decide the shape:

- **`playSession` is already separable from `runPlay`.** `runPlay`
  (`play_loop.go:24`) does the guards, `enterRaw` and the console, then calls
  `playSession(ctx, d, opt, session, held, keys, console)` (`play_loop.go:105`).
  That call is the reusable half, and `/play` wants it rather than `runPlay`.
- **BOTH loops call `enterRaw`** — `play_loop.go:85` and `replraw.go:25`. A
  `/play` that called `runPlay` would put an already-raw terminal into raw mode
  and take a second alternate screen inside the first. The command must reuse the
  REPL's live `rawSession`, not open its own.
- **The scoped interrupt already exists.** `interrupter.Set`
  (`interrupt.go:34`) exists so something narrower than the session can own
  Ctrl-C and hand it back — built in `#16` for a streaming answer. A sitting is
  the second thing that wants it, and "Ctrl-C ends the sitting, not the program"
  is exactly what `Set`/`restore` say.

So this is not new machinery. It is three existing seams meeting, and the risk is
that a fourth path through terminal setup gets written instead.

### The decisions the plan owns

- **Where the sitting draws.** The REPL is a scrolling loop with a pinned editor;
  `--play` paints full frames. Does `/play` take the alternate screen inside the
  REPL's, or draw into the same one? A wrong answer here is visible as corruption
  rather than as a silent bug, which is a mercy.
- **What the REPL looks like on return.** The sitting's summary is worth keeping
  on screen; the frames are not. `--play` already draws its summary into the
  buffer before handing the terminal back (`play_loop.go`'s exit comment), and
  that ordering is the precedent.
- **The line-mode REPL.** `replLines` has no raw terminal at all, and a sitting
  needs one. `/play` there should refuse with a sentence — the same shape
  `--play` uses when stdout is not a terminal — rather than degrade into
  something unusable.
- **`--play` stays.** A learner who wants only to review should not have to enter
  a REPL to do it, and scripts use the flag. Two entry points, one
  `playSession`.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source`, so the per-primitive hours are provisional; derived
against `#8` (3.71/2.98) and `#46` (7.02/5.91), the two most recent closes.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.35 impl=0.06
item: smaller-go-module        design=0.03 impl=0.14
item: smaller-go-module        design=0.02 impl=0.12
item: greenfield-go-module     design=0.04 impl=0.20
item: smaller-go-module        design=0.02 impl=0.12
item: cross-cutting-refactor   design=0.03 impl=0.16
item: smaller-go-module        design=0.02 impl=0.14
item: smaller-go-module        design=0.02 impl=0.12
item: atlas-docs               design=0.02 impl=0.06
item: smaller-go-module        design=0.0  impl=0.20
item: ux-rename-iteration      design=0.0  impl=0.20
item: milestone-review         design=0.0  impl=0.60
item: milestone-review         design=0.0  impl=0.85
design-buffer: 0.15
total: 3.60
```

| row | the work |
|---|---|
| `issue-spec` 0.35/0.06 | the Spec plus FOUR plan rounds, two of them Criticals about terminal ownership. Level with `#12`'s 0.35 and below `#8`'s 0.30-equivalent inflated by rounds: the design was reshaped rather than merely evidenced. |
| `smaller-go-module` 0.03/0.14 | `liveScreen.suspend`/`resume` — the one genuinely new capability, one flag in `repaint`'s existing guard plus a state model. |
| `smaller-go-module` 0.02/0.12 | `console.newSitting`, the factory built where `sess` is in scope so `runEditor`'s signature stays untouched. |
| `greenfield-go-module` 0.04/0.20 | `play_cmd.go`: `runPlayCommand` and `sittingInPlace`. |
| `smaller-go-module` 0.02/0.12 | the command row, the two refusals (no terminal, no deck), the atlas's derived list. |
| `cross-cutting-refactor` 0.03/0.16 | extracting `runPlay`'s shared half so both entry points reach one `playSession` — the Done-when that stops a change applying to only one. |
| `smaller-go-module` 0.02/0.14 | the pty test: `/play`, answer, Ctrl-C, back at the prompt. |
| `smaller-go-module` 0.02/0.12 | the `enterRaw` guard, and the store assertion that both doors record alike. |
| `atlas-docs` 0.02/0.06 | the README's per-command paragraph and the atlas. |
| `smaller-go-module` 0.0/0.20 | the mutation sweeps: the `Set`/`restore` pair, the `enterRaw` guard, suspend-is-not-Stop. |
| `ux-rename-iteration` 0.0/0.20 | the hand-run. Above `#8`'s 0.15 because a TERMINAL is what is being changed, and a wrong answer shows as corruption a test cannot see. |
| `milestone-review` 0.0/0.60 + 0.0/0.85 | the close boundary, run + remediation — the house pair. |

**Reconciliation.** Σdesign = 0.55, Σimpl = 2.97.
0.55 × 1.15 + 2.97 × 1.0 = **3.60**.

**Read against the trailing record.** This repo's ledger: `#42` 0.36, `#44` 0.47,
`#10` 0.62, `#12` 1.23, `#46` 1.18, `#8` 1.24 — median ≈ 0.62, which at 3.60
predicts roughly 6h actual. That gap is the within-session parallelism `#117`'s
ledger instruments; multiplying the rows to meet it would destroy the only signal
it carries.

**Why this is close to `#8` despite looking harder.** The variable impl is 1.32
against `#8`'s 1.22 — the fixed tail (sweeps, hand-run, docs, two review rows) is
1.91 in both. The terminal work is genuinely small ONCE NAMED: one flag in an
existing guard, one factory, one struct assembled by hand. What was expensive was
finding out that it had to be, which is design and is priced in `issue-spec`.

## Done when

- [x] `/play` runs today's sitting from inside the loop and returns to the
      definition prompt, both on finishing the queue and on Ctrl-C.
- [x] Ctrl-C during a sitting ends the SITTING, not the program — through
      `interrupter.Set`, not a second interrupt path.
- [x] The terminal is entered ONCE. A guard, not a comment: nothing may call
      `enterRaw` while a session is live, and the check derives its set of call
      sites rather than listing them.
- [x] `--play` and `/play` reach the same `playSession`, so a change to the
      sitting cannot apply to only one of them.
- [x] `/play` on the line-mode REPL refuses with a sentence naming the cause.
- [x] The command appears in `/help` (which reads the `commands` registry, so it
      follows from the row) and in the ATLAS's command list, which is the derived
      one — `TestDocsQuoteTheCommandList` checks `atlas/define.md`, not the
      README. The README's own prose is hand-written and swept, not derived.
- [x] Everything a sitting records is recorded identically from either entry
      point — one capture path, asserted through the store rather than through a
      fake.

## Plan

Durable design: `workshop/plans/000048-play-from-the-loop-plan.md`, on this
branch — a plan is checked against the code in its window by guards that walk
every plan in the tree (`repo_guard_test.go:1306`), so it travels with the branch
that implements it rather than ahead of it (#8 BR-12).

Single-pass: one boundary, plain checkboxes (AGENTS.md §3).

- [x] The command — /play records the intent, refuses where it cannot run.
- [x] The sitting — sittingInPlace on the terminal the loop already holds, with
      Ctrl-C scoped through interrupter.Set.

## Log

### 2026-09-07 — filed, IN the MVP

Operator request: *"we should add another /action in the TUI program, /play to
trigger today's play. after play is finished, or ctrl-c to exit play mode, we go
back to definition mode. this way user can always be in the TUI program."*

Added to `define-learn`'s `mvp_scope` on the operator's instruction — see that
project's scope event of the same date.

**Checked before filing**, because the interesting part is what already exists:
`playSession` is already the separable half of `runPlay`; both loops already call
`enterRaw`, so nesting is the hazard; and `interrupter.Set` was built in `#16`
for exactly the "something narrower than the session owns Ctrl-C" case a sitting
now needs. The work is joining three seams, and the failure mode is writing a
fourth path through terminal setup instead.

### 2026-09-08 — built, and smoke-tested by the operator

**Operator smoke test PASSED** on a freshly restaged deck (ten words, five
authored cloze items, no events), including the Ctrl-C path the first run had
missed.

**What the four plan rounds bought.** The issue was filed as three existing seams
meeting, and the operator and I both read it as straightforward. The command half
was. The terminal half needed three things, none of which existed:

1. **`liveScreen.suspend`/`resume`** — two screens now share one terminal, and
   each carries a throttled painter that fires on its own goroutine. `Stop` is
   one-way; it is the end of a screen's life, not a pause. One flag, checked in
   `repaint`'s existing gate, which is the only thing that writes to the tty.
2. **A console that BORROWS.** `newConsole` acquires three things a borrower must
   not: a `finish` that restores the SHARED session and prints to a cooked
   terminal, a second `watchResize` goroutine, and a second `enterMouse`. The
   sitting assembles its console by hand and borrows the loop's resize channel,
   so there is one watcher for the process's life.
3. **`console.newSitting`**, built where `sess` and the real stdout are in scope,
   so `runEditor`'s signature stays what its own doc calls the point: "the editor
   loop with the terminal factored out".

**The summary goes UP, not out.** `handBack` prints the transcript AFTER
restoring to cooked mode, and the order is the precondition — the first revision
proposed a no-op restorer, which keeps the order and removes what it was for.
The sitting writes its transcript into the editor's buffer instead, where it is
in the scrollback when the prompt returns.

**Three guards, each mutation-checked with the real regression:**
`TestASittingFromTheLoopNeverEntersRawMode` walks `sittingInPlace`'s callees
transitively and names the path (routing `/play` through `runPlay` reddens it as
`sittingInPlace → runPlay → enterRaw`);
`TestBothEntryPointsReachOnePlaySession` is the Done-when that stops a change
applying to only one door; and
`TestASuspendedScreenPaintsNothingAndResumesWhereItWas` carries the clause that
separates suspend from `Stop` — after resume, the frame comes back.

**One comment corrected mid-build.** `suspend` disarms the timer, and the comment
said the frame stayed off the terminal because of it. The sweep proved otherwise:
removing the disarm reddened nothing, because `repaint`'s gate already stops the
flush. It is hygiene, and the comment says so now rather than claiming a
load-bearing role it does not have.

### 2026-09-08 — the root cause, and what it found

The operator asked for the root cause rather than another surface fix. Reading
the fourteen findings across four plan rounds and four close rounds, they are not
fourteen mistakes. They are three, and arguably one.

**1. I check one side of a relationship.** `enterAlt` is idempotent — but
`restore` is shared (PQ-1). The sitting gets its own `finish` — but the entity
row still called `newConsole` (PQ-8). Two painters were handled — but so were two
resize watchers (PQ-4). The screen's shape was handed back — but not `opt.width`
(BR-5, BR-14). The consumer was pinned — but not the producer (BR-19). The seam
was pinned — but not the site (BR-17).

Every one is a design that creates a RELATIONSHIP — borrower/owner,
producer/consumer, seam/site, acquire/release — where I reasoned about the half I
was building and not the half it implies.

**2. I test the property as I conceive it.** Four wrong interrupt tests (BR-7),
each passing; a test named "either door" that invoked neither (BR-8); three
refusals ticked and unwritten (BR-6). And when I swept, I chose the mutation that
matched MY model of the break — removing `restore` — rather than the one a
careless edit would make, which was removing `Set` and `restore` together.

**3. I write prose in the same motion as code, and prose has no compiler.** A
false Go rule in the rule-store (BR-18); a plan ticked for work that did not ship
(BR-15); four wrong claims about existing code in `#8`.

**Underneath all three: I validate my model of the system instead of the system.**
The counter-practice is not "be more careful" — it is that every check that
actually worked this session was an EXECUTION. The compiler found
`Progress.FirstSeen`. A four-line program found the false defer rule. A probe
found the Havana midnight. The reviewer's mutation found what mine missed. The
operator's smoke test found `words/day 0.0`. Reasoning found none of them.

**Applying it, before this round's reviewer could.** If the rule is "a sitting
mutates state the loop caches, and the loop must be told", then the shape was two
of three: **the DECK is the third.** A sitting can drop a word (`d`), and the
loop caches the highlight set in a local — `replraw.go` says so explicitly,
because `/lang` had to solve the same problem. Before `#48` a word could only
leave the deck in a one-shot `--forget`, where a stale set died with the process.

So: look a word up, `/play`, drop it, come back — and it was still painted green
at the prompt. Found by applying the rule rather than by a review. `Vocabulary`
gains `Forget`, symmetric with the `Add` that `Capture` already does, with the
phrase-width recount that is easy to miss.

**And it turned up a second thing.** While trying to drive the drop path in a
test, the `d` gesture would not reach the arm — and `TestDropRecordsNoReview`
(`play_loop_test.go:1408`) passes for the same reason. It asserts that dropping
records no review, which is ALSO true when the drop never happens. That is a
pre-existing vacuous test, not this issue's to fix, and it is recorded here so it
is not lost: the drop wiring is guarded from the source instead, which catches the
regression that actually occurred.
