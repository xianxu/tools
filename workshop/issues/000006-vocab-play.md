---
id: 000006
status: working
deps: ["tools#3", "tools#5"]
github_issue:
created: 2026-08-20
updated: 2026-08-27
estimate_hours: 6.15
started: 2026-08-27T08:15:55-07:00
---

# define --play: review loop + form 2.1 quick pass

## Problem

The deck and the schedule are inert without a way to sit down and review.

## Spec

`define --play` runs today's queue.

- Form 2.1 (this issue): show the word, let the learner recall, reveal the
  definition, self-rate right/wrong. No question generation, no network.
- A `Question` interface every later form implements, so forms drop in without
  touching the loop.
- Terminal-shaped and single-threaded; interruptible at any point. Events are
  written **as they happen**, so Ctrl-C mid-session keeps what was reviewed —
  free from the append-only log (#3).
- Playing the pronunciation during review is on by default; `--no-audio` applies.
- A bad-question keypress records a flag against the question (groundwork for
  the generated forms, which need this feedback loop).

## Done when

- [ ] A full session runs against fake store + fake clock, recording one event
      per answer.
- [ ] A SKIP records nothing: it is not an assessment, and `Fold` would read a
      recorded skip as a miss and demote the word. Asserted on the SESSION (no
      record outcome is emitted), not on `CaptureReview` — whose `bool` cannot
      represent a skip precisely because one is never passed to it.
- [ ] Audio plays before reveal by default, and `--no-audio` silences it.
- [ ] `-count` bounds the session; it defaults to 20.
- [ ] An empty queue prints a line and exits 0 — "nothing due today" is the
      expected state most days, not an error.
- [ ] With NO DECK, `--play` prints one line and exits 0 rather than running —
      guarded on `deck == nil`, not on the env var, because `openStore`'s `Getwd`
      failure path also returns no deck and keying on the flag would hand a nil
      store to the queue builder and panic. (The first draft of this row said a
      session runs and records nothing under `DEFINE_NO_CAPTURE` — unsatisfiable,
      since that branch returns no deck at all.)
- [ ] Interrupting mid-session preserves already-recorded events.
- [x] Adding a second form requires no change to the loop — asserted by driving the same table through a fake form with entirely different keys, and by checking that 2.1's own keys mean nothing to it.
- [ ] **A full session runs with the LLM seam unavailable**, falling back to the
      forms that need neither key nor network (2.1 here, 2.3 in #7) rather than
      failing. Relocated from #11 on 2026-08-22: it names `--play`, so it belongs
      to the issue that owns `--play`. Asserted with a client returning
      `llm.ErrUnavailable`, not by unsetting an env var — the point is that the
      loop degrades, not that config resolution does.

## Plan

Durable plan: `workshop/plans/000006-vocab-play-plan.md` (two milestones; each
`Mx` row is its own review boundary).

- [x] M1 — `Question`, `Recall`, `Session`/`Apply`, the purity guards, the atlas
- [ ] M2 — `CaptureReview`, `runPlay`, the `--play` flag, the README

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), one item per task plus the
process work inside the measured window. Design carries v2's ×0.2 spec-quality
discount on every code item — the plan pre-resolves the `Input` translation, the
single skip filter, the budget flag, the no-deck guard and the guard extraction.
Implementation is v3.1's 40% of the v2 table. Familiarity 1.0: `#5` just built
the sibling package and `#14`'s editor is the shape `Apply` copies.

**Round prices follow the split `#5` established and its outcome confirms.** #5
priced plan rounds at 0.22 and boundary rounds at 0.35, budgeted three per
boundary, and closed at 3.0h against 5.56 — over-estimated, but by less than the
two before it. FOUR plan rounds happened here (the gate cap was reached), so they
are counted as spent rather than guessed.

**The one real unknown is the terminal loop, and it is priced as such.** `runPlay`
is the first new raw-mode surface since `#16`, and `#16`'s streaming loop is the
row that overran worst in this repo's history. `#20` and `#21` both touched that
code and came in short, so the machinery is now well understood — but `runPlay`
creates a new FILE and a new loop, which is `greenfield-go-module`, not a
refactor of an existing one. That is where the honest uncertainty sits.

**Corrected at the estimate gate, and the correction is the point.** The first
draft raised boundary rounds to 0.30 impl *and* under-priced `runPlay` as a
refactor — two errors that partially cancelled to a plausible total. `#5`'s own
block wrote the rule this violates: *"A number that is right by accident teaches
the ledger nothing."* Rounds are back at `#5`'s measured `0.15/0.20`, `runPlay`
is greenfield, and a fifth plan round (this one) is counted. Every number is now
defensible on its own, which is what makes the resulting row usable evidence.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=0.50 impl=0.08
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: cross-cutting-refactor design=0.12 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
design-buffer: 0.15
total: 6.15
```

Item-to-task map. Process: spec+plan, then the four plan rounds (all spent — two
Criticals in round 1, and round 3 caught me stating the skip filter in two places
without choosing). **M1** — `smaller` = Task 1's `Question`/`Recall`;
`greenfield` = Task 2's `Session`/`Apply` with the `fakeForm` property;
`cross-cutting` = extracting `#5`'s purity guards into a shared helper touching
both packages; `atlas-docs` = M1's atlas; then three M1 boundary rounds. **M2** —
`cross-cutting` ×2 = Task 3's `CaptureReview` across every implementation and
double, and Task 4's `runPlay` with the raw-mode wiring; `smaller` = the `--play`
and `-count` flags; `atlas-docs` = M2's atlas plus the README; then three close
rounds.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-27

Claimed and planned. Three decisions worth recording before implementation:

- **A skip is a third verdict, not a wrong answer.** Recording a skip as a miss
  would demote the word through `#5`'s `Answer`, so a learner who skips a word
  they half-know would be punished for honesty. It records NOTHING, which leaves
  `#8` unable to distinguish "skipped" from "never reviewed" — a deliberate hole,
  and the alternative (a fourth event kind) would make `Fold` learn to ignore
  something.
- **`Grade` lives on the form, not in the loop.** The Done-when says a second form
  must need no loop change, and that is a property of the interface rather than a
  promise — so key semantics stay inside the form (2.1's `y/n`, 2.3's `1/2/3/4`),
  and the loop only asks "did that key mean anything to you".
- **Reviews record through `Capturer`, not `store.AppendEvent`.** `capture.go`
  states the rule: capture is the only thing that records, and a second appender
  beside it is how that stops being true unnoticed. `#16` added `CaptureAsk` the
  same way.

  A fourth, added at the plan gate: **the Spec's bad-question keypress is
  deferred to `#12`/`#13`.** Form 2.1's question is the learner's own deck word
  plus NOAD's definition — nothing is generated, so there is nothing to flag as
  bad, and the feature would ship with no consumer and no way to exercise it. The
  Spec itself calls it "groundwork for the generated forms"; this records that the
  groundwork lands with the forms that need it rather than being silently dropped.

- 2026-08-27: M1 — `Question`, `Recall` (form 2.1), and the `Session`/`Apply`
  state machine, in a second pure package.
  The purity guards were EXTRACTED rather than copied (`cmd/define/puretest`),
  which the plan gate asked for and which pays for itself immediately: `#7`,
  `#12` and `#13` each add a form package, so copying would have meant five sets
  to keep in agreement. It also closes the gap I recorded at `#5`'s close — every
  "the mutant reddens it" claim there was verified in a scratch copy and thrown
  away, and now both callers' guards are exercised from one body with the
  negative cases verified in the tree.
  One correction to the extracted helper: its vacuity check fatalled on ZERO
  imports, which fired on `play` — a package so pure it imports nothing at all,
  the strongest version of the claim being tested. A vacuity guard has to
  distinguish "the measurement failed" from "the answer is legitimately empty".
