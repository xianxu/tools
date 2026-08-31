---
id: 000039
status: working
deps: []
github_issue:
created: 2026-08-31
updated: 2026-08-31
estimate_hours: 2.94
started: 2026-08-31T11:01:05-07:00
---

# spaced repetition: one unbounded ladder, confidence-driven promotion, and a visible daily budget

## Problem

`define`'s schedule is a Leitner ladder of `1, 3, 7, 14, 30, 90` days with a
mastery bar of seven consecutive correct answers. Three things are wrong with it,
and they were found by using the tool rather than by reading the code.

**It starts backing off immediately.** The first three rungs are 1, 3 and 7 days,
which assumes the word is already learned and only needs protecting. A word met
once yesterday is not learned. Pimsleur's graduated interval recall and Anki's
FSRS both spend heavily in the first days and then back off; this ladder never
spends.

**It has a ceiling, and a ceiling is unsustainable at any admission rate.** In
steady state, with `a` new words per day over `N` rungs:

```
daily reviews = a × N            (climbing: every word passes each rung once)
              + stock / I_top    (everything parked at the top)
```

The second term grows LINEARLY forever. At 7 new words/day with a 90-day top
rung, the accumulated stock alone costs ~55 reviews/day after two years and there
is no budget left for anything new. The ladder needs no ceiling: if intervals
keep growing geometrically, a word of age τ is reviewed at roughly `1/τ` per day
and the total load integrates to `a × ln(T)` — logarithmic, so a fixed daily
budget supports a nearly constant new-word rate indefinitely.

**Mastery costs 235 days and one slip near the end costs six months.** Seven
CONSECUTIVE correct on a ladder whose top rung is 90 days means five promotions
to climb (55 days) plus two confirmations at 90 days each. A wrong answer resets
the streak to zero.

There is also no way for the learner to see what any of this costs them. The
number that should govern how many new words they take on — reviews per day — is
not computed or shown anywhere.

## Spec

**One ladder. No phases, no terminus.** "Acquisition" and "maintenance" are the
early and late rungs of a single curve, not two systems. The interval is
computed, not tabled:

```
IntervalDays(box) = max(1, floor(1.6 ** box))
```

| box | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| wait (days) | 1 | 1 | 2 | 4 | 6 | 10 | 16 | 26 | 42 | 68 | 109 | 175 | 281 |
| reviewed on day | 1 | 2 | 4 | 8 | 14 | 24 | 40 | 66 | 108 | 176 | 285 | 460 | 741 |

**The two 1-day rungs are the point, not a rounding artifact.** A new word is
seen on day 1 and again on day 2, which is when the forgetting curve is steepest.
`floor(1.6**(box+1))` would remove the duplicate and with it the most valuable
rung in the ladder; the duplicate is what gives the curve its acquisition
density, and it costs one extra review per word, once.

**No cap.** Box 20 is a 12-year interval and costs nothing to keep. A word never
leaves the system; it just gets cheap.

**Why 1.6 rather than 2.** Cepeda et al.'s spacing meta-analysis finds the
optimal gap shrinks as a proportion of the target retention interval, which
argues for a ratio under 2. The review rate goes as `r/(r-1)`, so 1.6 costs about
35% more reviews than doubling — roughly 6 new words/day on a 50-review budget
where doubling would allow 8. That is retention bought with review slots, and it
is the trade this issue chooses. Revisit if `--stats` shows retention is fine.

### Transitions

| answer | effect |
|---|---|
| correct | `box + 1`, or `box + 2` while `box < MaxBox` |
| confident (a form that can say) | `box + 2` |
| wrong | `box → box / 2`, and `MaxBox → MaxBox - 1` |

**Demotion halves rather than stepping.** One sentence, and it scales: box 12
(281 days) falls to box 6 (16 days), which is a real relearning interval, while
box 2 falls to box 1, which is barely a nudge. Harsh where harshness is warranted
and gentle where it is not — which the current fixed `-1` cannot be.

**`MaxBox` is the express lane back.** A word you took to box 12 and lapsed is not
a word you have never seen: storage strength survives even when retrieval
strength does not, which is why relearning is faster than learning (Ebbinghaus's
savings). So while `box < MaxBox`, a correct answer climbs two rungs instead of
one. Worked example, from box 10 (175 days):

```
lapse   → box 5  (10d),  MaxBox 9
correct → box 7  (26d)   [5 < 9, so +2]
correct → box 9  (68d)   [7 < 9, so +2]
correct → box 10         [9 = 9, so +1]
```

Three reviews instead of five, and the first retest lands 10 days after the
failure, which is where the relearning actually happens.

**`MaxBox` erodes by one on every lapse**, so a word that keeps failing
gradually loses its express lane and is eventually relearned properly rather than
being waved back up forever. Like `Box` and `Streak` it is DERIVED by `Fold` from
the event log — no new stored state, per `store/event.go`'s rule that the log is
the only record.

**These two are a PAIR and neither works alone.** A gentle `-2` demotion needs no
express lane because it never travels far; a halving demotion without one would
make a single slip cost most of a year. Do not adopt one without the other.

### Mastery

`Mastered = Box >= 9`, reached on day 108 with nine correct recalls, the last of
which came after a 42-day gap. The streak requirement goes away: reaching box 9
already requires a clean-enough run, and the current rule's real cost was never
the streak but the two extra 90-day waits it forced.

**Mastery stays a LABEL, never a removal**, and `queue.go` already states why:
*"a word never offered can never be answered wrong, so it could never be
demoted, and the learner's mastered count could only ever grow while their actual
recall decayed."* A mastered word keeps being reviewed — at box 9 that is three
times a year, which rounds to nothing — and keeps serving as a distractor.

### Admission and overflow

**Every looked-up word is admitted.** No gate, no second-lookup rule, no
friction: removal is one keystroke and should be reachable from every surface
where a word appears.

This is safe because `schedule.Queue` ALREADY handles overflow by ordering:
reviewed words are ranked before fresh ones, so `--play` only reaches new
material once the existing backlog fits in the budget. The learning rate
self-throttles to what the learner can afford, while the deck grows freely and
`Lookups` ranks which unstudied word surfaces first — a personal frequency
distribution measured from what they actually read, which is strictly better for
this tool than the corpus frequency list `#10` retired.

**One fix is needed to make that true: rank overdue RELATIVE to interval.**
`queue.go` currently ranks by absolute days overdue, which systematically favours
high boxes — a box-12 word 50 days late (18% over) outranks a box-1 word 4 days
late (400% over), though the second is in real danger and the first being
slightly late is harmless or even beneficial. Ranking by `overdue / interval`
puts the fragile words first, which is what makes always-admit safe under a
backlog.

### The budget, made visible

Reviews per day is a pure function of `Progress` and needs no new data:

```
daily reviews = Σ 1 / IntervalDays(box)   over the deck
```

`--stats` (`#8`) should report it, plus the sustainable new-word rate at the
current mix — the number that should govern how many new words the learner takes
on, rather than being discovered as a growing backlog.

## Done when

- [x] `IntervalDays` is computed from the ratio, unbounded, with `1, 1, 2, 4, 6, 10, 16, 26…` pinned by a table test.
- [x] Correct promotes one rung, or two while below `MaxBox`; wrong halves the box and erodes `MaxBox`.
- [x] `MaxBox` is derived by `Fold` from the log, stored nowhere.
- [x] `Mastered` is `Box >= 9` and excludes nothing from the queue.
- [x] `Queue` ranks reviewed words by overdue RELATIVE to interval, pinned by a test where a low-box word beats a more-absolutely-overdue high-box one.
- [x] Daily-review load and the sustainable new-word rate are computable from `Progress` alone, and reported.
- [x] The recovery path is pinned end to end: a word at box 10 that lapses is back at box 10 in three reviews, not five.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.40 impl=0.06
item: smaller-go-module        design=0.05 impl=0.20
item: greenfield-go-module     design=0.06 impl=0.32
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.03 impl=0.14
item: greenfield-go-module     design=0.05 impl=0.24
item: cross-cutting-refactor   design=0.05 impl=0.28
item: smaller-go-module        design=0.02 impl=0.08
item: atlas-docs               design=0.03 impl=0.06
item: milestone-review         design=0.0  impl=0.30
item: milestone-review         design=0.0  impl=0.32
design-buffer: 0.15
total: 2.94
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source`, so the per-primitive hours are provisional.

| item | task | why this primitive |
|---|---|---|
| `issue-spec` 0.40/0.06 | the design carrier | the Spec, this plan, two plan-quality rounds |
| `smaller-go-module` 0.05/0.20 | T1 the computed ladder | a table becomes a function, plus a 21-row exactness test and the clamp argument |
| `greenfield-go-module` 0.06/0.32 | T2 `Grade`, transitions, `MaxBox` | the algorithmic heart: a new enum, three transitions, the express lane and its erosion — four Done-when rows |
| `smaller-go-module` 0.02/0.12 | T3 `Mastered`, delete `Streak` | touches `Answer`, `Fold`, `Mastered` and their tests |
| `smaller-go-module` 0.03/0.14 | T4 relative overdue | one comparator, plus a test that must be RED on today's code |
| `greenfield-go-module` 0.05/0.24 | T5 the load functions | a new file, three functions, one derived from the ladder |
| `cross-cutting-refactor` 0.05/0.28 | T6 `unaided` to the log | four files — `store`, `play`, `capture`, the loop — and the `advance` trap (D14) |
| `smaller-go-module` 0.02/0.08 | T7 `finish` reports | one line, one test |
| `atlas-docs` 0.03/0.06 | T8 | atlas, README, project row |
| `milestone-review` 0.0/0.30 | the boundary: run + the manual verification pass | folded into one line because this plan commits to ONE boundary; the manual pass is verification rather than review, and `#127` should read it as such |
| `milestone-review` 0.0/0.32 | the boundary: remediation | priced at the TOP of the range on measured evidence — see below |

**Two corrections were applied, both derived and both per-primitive.**

*Remediation is priced at the top of its range.* `#7` closed one day ago and took
**eight review rounds** surfacing three Criticals. Its estimate priced a boundary
at 0.16 to run plus 0.12 to remediate; the rounds alone plainly cost more than
0.28h. Correcting a primitive against a measured row in this repo is what the
primitive table is for.

*The task lines were raised after the first draft priced them BELOW `#7`'s.* The
first version summed 1.16h of task implementation against `#7`'s 1.48h — for more
scope (8 tasks to 6, 15 Done-when rows to 9, spanning `schedule`, `play`,
`store`, `capture.go` and `play_loop.go`). Claiming a neutral hold while pricing
lower than the measured neighbour is not neutral; it is a correction in the wrong
direction. Each line was re-derived against what it actually does, not nudged
toward a target.

**The standing calibration question, unchanged.** Two closed rows: `#30` at 3.4×
and `#7` at 1.70×. Both overran, which points at the v3.1 impl scale being too
aggressive for `define`-sized work (`#127`). **A calibration-adjusted reading of
this issue is therefore ~4.5-5h, and that number is recorded here deliberately
while the block's own total stays 2.94.** The block reports what the model says;
the paragraph reports what I expect. Folding the second into the first would
corrupt the ledger that exists to measure the gap between them. If this lands
near 1.7× again, that is three consecutive rows and the SCALE should move rather
than each estimate quietly compensating.

## Plan

- [x] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-31

Filed from a design conversation with the operator. The shape was reached by
working the arithmetic rather than by preference — the ceiling problem, the
`a × ln(T)` result and the relative-overdue flaw all came out of asking what a
fixed daily budget can actually buy.

Two things were found in the existing code during that conversation and are
recorded here because they change what this issue must NOT do:

- `Queue` already orders reviewed before fresh, which is the admission control
  this issue would otherwise have had to build. It needs the relative-overdue
  fix and nothing else.
- `Queue` already documents that mastered words must not be excluded, with the
  absorbing-state argument. Mastery must stay presentational.

**Open, deliberately not decided here:** the ratio 1.6 is a judgment call the
operator made against the alternative of 2.0, and `--stats` is what would
eventually say whether it is right. The confident/+2 transition has no producer
until a form can report confidence — see the review-modes issue for `/board`.

## Revisions

### 2026-08-31 — the `+2 confident` transition has an objective producer

**Reason.** The Spec above names a `confident → box + 2` transition and says its
producer is "a form that can say". While filing `#40` it became clear that the
obvious candidate — a `firm` mark in the grid — is the WORST source for it, and
that a better one already exists.

**Delta.**

- **`+2` is earned by a correct answer in form 2.3 given WITHOUT a reveal.**
  `Apply` already knows `s.Revealed` at grading time, so "answered correctly
  without needing to see the answer first" is available today, costs nothing, and
  is MEASURED rather than claimed. A learner who peeked and then picked right has
  demonstrated recognition; one who picked right cold has demonstrated recall.
  That is exactly the distinction `+2` wants.
- **Self-report does NOT earn `+2`.** `#40`'s grid is triage without retrieval,
  and the illusion of knowing runs in the direction of overconfidence, so a
  `firm` mark earns `+1` — the same as an ordinary correct answer — and no more.
  The reasoning lives in `#40`'s Spec.
- **Nothing else changes.** The transition table stands; only its producer is
  now named, and named as something the session can observe rather than something
  the learner asserts.

## Log

### 2026-08-31 — built in one pass

**The `advance` trap was the interesting part**, and the plan-quality gate found
it before any code existed. `advance` zeroes `s.Revealed` before it builds the
Record outcome, so the obvious implementation — computing `unaided` where the
outcome is constructed — marks EVERY correct answer unaided and runs the ladder
at double speed. It would have looked like a working feature. The discriminating
case is reveal-then-answer-correctly, which is why it became a Done-when row
rather than a line in the manual verification block.

**`SelfRated` exists because form 2.1's `y` is self-report**, which the first
draft of the plan missed: it granted the two-rung promotion to `Recall`, one form
to the left of the board the issue had just denied it to.

**The halving and the express lane are pinned as a PAIR.** Removing the lane
turns the recovery walk red at step 1; demoting one box instead of halving fails
seven tests. A test asserting only the end state would have passed on either.

**Manual verification on a real terminal**, with the real dictionary, two
sittings on fresh decks:

| run | reviewed | correct | `unaided: true` |
|---|---|---|---|
| answered cold | 8 | 4 | **4** |
| revealed first | 6 | 2 | **0** |

Every cold correct answer earned the flag and no revealed one did — the
behaviour `TestARevealDisqualifiesUnaided` pins, confirmed outside the test
harness.

**Two repo guards were fixed as side-quests.**
`TestPlanTablesNameEntitiesThatExist` required a `deleted` row's entity to still
EXIST, so the one status word that asserts an absence could never pass; it now
checks the opposite, mutation-verified. And the plan named two tests that did not
exist under those names — the plan's names were better, so the tests were renamed
to match rather than the plan edited to describe whatever I had typed.

**A migration effect worth knowing:** no migration runs, but `Fold` re-derives
every box under the new transition, so the first sitting after this ships will be
LARGE for a mature deck. `TestReFoldingAnOldLogIsSafe` pins the direction — below
box 10 words become due sooner, never later, so nobody loses reviews they had
earned.
