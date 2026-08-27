---
id: 000005
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-26
estimate_hours: 5.56
started: 2026-08-26T22:20:11-07:00
---

# spaced-repetition scheduling engine (Leitner, pure)

## Problem

Reviewing everything every day does not scale and does not work. The schedule
decides what is worth the learner's attention today.

## Spec

Leitner boxes with fixed intervals (1, 3, 7, 14, 30, 90 days).

- Chosen over SM-2 deliberately: SM-2's ease factors cannot be explained in a
  stats screen, and for a personal tool "why is this due?" should be answerable
  in one sentence.
- Pure: `Due(word, now)`, `Promote/Demote(word, correct)`, `Mastered` = final box
  reached with N consecutive correct answers.
- Selection is also pure: given a deck, a clock and a daily budget, return
  today's queue — ordered by overdue-ness, then by lookup count (#4).
- No IO whatsoever; every test sets the clock explicitly.

## Done when

- [x] Promotion/demotion and due-dates verified across a simulated multi-week
      schedule with a fake clock.
- [x] A daily budget never returns more than the budget, and never starves an
      overdue word in favour of a fresh one — asserted on a fixture holding BOTH
      an overdue reviewed word and a fresh one, so the two orderings differ.
- [x] `Mastered` has one definition, exported for `--play` and `--stats`. It
      cannot be fully closed here: `#8` is FILED and open
      (`workshop/issues/000008-vocab-stats.md`) but unbuilt, so this issue
      delivers the single definition and `#8` is where "used by both" becomes
      true.
- [x] Schedule state is DERIVED from the event log, never stored beside it —
      `store.Word` gains no fields.
- [x] The package compiles against `store` and pure stdlib only; no test in it needs
      a fake.

## Plan

Durable plan: `workshop/plans/000005-vocab-schedule-plan.md` (two milestones;
each `Mx` row is its own review boundary).

- [x] M1 — `Box`, `Progress`, `Fold`, `Due`, `Answer`, `Mastered`
- [x] M2 — `Queue`, the two tiers, and the atlas

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), one item per task in
`workshop/plans/000005-vocab-schedule-plan.md` plus the process work inside the
measured window. Design carries v2's ×0.2 spec-quality discount on every code
item — the plan pre-resolves the time model, the derived-not-stored decision, the
two queue tiers, `masteryStreak`'s value and the helper's ownership, so what is
left at design time is reading. Implementation is v3.1's 40% of the v2 table.
Familiarity 1.0: same repo, and the only unfamiliar thing is that this is the
first subpackage under `cmd/define` besides `store`, which is itself the pattern
being copied.

`issue-spec` is 0.50 — the low end of its band. The Spec was written in August and
needed no revision; this window authored a ~190-line plan, against #21's 1.50 for
646 lines and #9's 0.75 for 351.

**Review rounds drop to 0.35h, and this time the derivation is arithmetic rather
than a feeling.** #9 measured 3.80h TOTAL across five review rounds *plus* all of
its implementation — three new files, a store interface widening, two
conformance suites and a live check. Even attributing only ~2h to the code, the
rounds cannot have averaged more than ~0.35h. #9's own block priced them at 0.5h
and came in at 2.16×; the round before that priced them at 0.3h and I corrected
it UP to 0.5h on the grounds that halving a measured value was not a derivation.
That correction was right in method and wrong in fact, and the ledger has now
said so twice.

**The recent rows agree on direction and I am following them rather than the
older ones.** tools#21 est 9.28 / actual 6.00 (1.55×) and tools#9 est 8.22 /
actual 3.80 (2.16×), both trusted, both over-estimated. The older rows that say
the opposite — #16 at 0.37×, #11 at 0.64× — predate this working rhythm by a
week and two issues, and #16's overrun was twelve review rounds on a milestone
structure this project no longer uses. Weighting the two most recent trusted
rows over four older ones is a judgement, and it is the one the evidence
supports; if #5 closes near 5h the drift is corrected, and if it closes near 2.5h
the round price is still too high and the next block should say so.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=0.50 impl=0.08
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: cross-cutting-refactor design=0.12 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: greenfield-go-module   design=0.25 impl=0.22
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
design-buffer: 0.15
total: 5.56
```

**PLAN rounds and BOUNDARY rounds are priced differently, and the first draft of
this block used one number for both.** A plan round reads a document and edits a
document; a boundary round reads a diff, and its fixes are code with tests and
mutation checks. Counting from the gate file there were FOUR plan rounds, priced
0.22h each — and boundary rounds stay 0.35h, now budgeted THREE per boundary
rather than one-plus-a-fix, which is what #21 (3, 3) and #9 (2, 3) actually
needed. The first draft's 4.99 was the right order of magnitude by cancellation:
under-counting plan rounds and under-budgeting boundary rounds happened to
offset. A number that is right by accident teaches the ledger nothing.

Item-to-task map. Process: plan authoring, then the four plan rounds (all spent —
round 2 caught three false claims about existing code, round 3 caught the residue
of that sweep, round 4 was the estimate). **M1** — `cross-cutting-refactor` = Task 0, which adds
`store.StartOfDay` and collapses the two existing day-boundary encodings in
`history_cmd.go`; `smaller` = Task 1's `Box` and interval table; `greenfield` =
Task 2's `Progress`/`Fold`/`Due`/`Answer`/`Mastered` with its fuzz target;
`smaller` = the import guard; then three M1 boundary rounds. **M2** —
`greenfield` = Task 3's `Queue` with the two tiers; `atlas-docs` = the atlas
section; then three close-boundary rounds.

**Measurement note for the close:** `sdlc actual` currently reads 0.00h with
attribution shared across `#5` and `#9`, because the plan and its four gate
rounds are still uncommitted at the time of writing. Whether the ~1.6h already
spent on design converges depends on the plan commit landing inside the window.
Worth a line in `## Log` at close so the ledger row is readable rather than
mysteriously small — the same defect recorded on `#20` and `#21`.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-26

Claimed and planned, ahead of `#10` — see the project's scope event: the operator
asked for a loop they can actually practise with before the authoring
integrations, and the dependency graph already allowed it.

Two decisions worth recording before implementation:

- **Schedule state is DERIVED from the event log, not stored on the word.**
  `event.go` already states the rule for the whole store: the log is "deliberately
  the ONLY record of activity ... storing counters alongside would create a second
  source of truth that drifts." A `Box` field on `store.Word` would be exactly
  that, and it would drift silently — a hand-edited or partially-written deck file
  disagreeing with the events that produced it. So `Fold` is a pass over the log,
  and `store` gains nothing.
- **Its own package.** `cmd/define` is `package main` and this is the first thing
  here with no reason to touch IO. A subpackage makes the purity claim checkable
  from outside rather than asserted in a comment: `schedule` imports `store` and
  `time`, and a test needing a fake would not compile.

- 2026-08-26: M1 — `store.StartOfDay`/`DaysBetween` (Task 0), then the box ladder,
  `Progress`, `Fold`, `Due`, `Answer` and `Mastered`.
  Task 0 was a genuine ARCH-DRY fix that this issue merely forced into the open:
  `#15` needed "which local day is this" twice for `/history` and wrote it inline
  twice in two DIFFERENT shapes — one building a local midnight, one projecting
  onto a UTC day index. `#5` needing it a third time is what moved it into
  `store` beside the `Clock` that owns the time model. `/history`'s own tests
  were the regression net and passed unchanged.
  The purity claim is now ENFORCED rather than asserted: the plan's first draft
  said "a test needing a fake would not compile", which is false, and
  `TestScheduleImportsOnlyStoreAndTime` reads the package's import set and fails
  when anything outside the pure allowlist appears — verified by adding `os`.
  FuzzFold 11.1M execs on box-in-range, non-negative streak and idempotence.
  Permutation-independence is deliberately NOT asserted: two reviews sharing a
  timestamp with different outcomes fold differently by order and `ReviewEvent`
  has no tiebreaker, so `Fold`'s contract is the order `store.Events` returns.
  Five mutations run; four died first time. The survivor was "Fold counts every
  event kind" — my fixture put the lookup and the question BEFORE any promotion,
  where their spurious demotions clamp at box 0 and vanish, so both
  implementations gave the same answer. Moved them after two correct reviews and
  it dies.

- 2026-08-26: M2 — `Queue`, with the two tiers the Done-when's starvation clause
  forces. Ranking everything on one "how long since we saw it" axis is the
  tempting simplification and is exactly the bug: a word first seen months ago
  and never reviewed outranks one reviewed last week and three days overdue. The
  mutation that collapses the tiers reddens two named tests.
  The purity guard earned its keep twice in one milestone, and the first time it
  was WRONG in a useful way: it fired on `sort`, because my allowlist expressed
  "exactly two imports" when the claim is "no IO and no hidden clock". A guard
  that reddens on correct code invites deleting the guard, so the rule was
  rewritten rather than the code — and it gained the check an import list cannot
  make, that `time.Now` appears nowhere, since `time` is legitimately imported
  for its type.

- 2026-08-26: close boundary round 1 — REWORK, one Critical and five Importants,
  all real. The Critical is the one worth keeping: `store.DaysBetween` was WRONG
  for arguments in different locations, so `Due` reported a word due on the day it
  was reviewed. Store stamps carry fixed offsets (yaml.v3) while `now` comes from
  `time.Local`, so mixed zones are the NORMAL state for half the year — and no
  test crossed zones, so it was green. Reproduced empirically before fixing:
  same-local-day gave 1, and a DST-crossing pair gave 7 where the answer is 6.
  The deeper lesson is about the consolidation that introduced it: `#15` had
  written the idea twice, and I kept the shape that read more cleanly and dropped
  the one whose comment explained why it was shaped that way. `/history`'s tests
  passed as a regression net because `relativeDay` normalises its arguments
  first — a refactor's net only covers what the OLD callers did, and `Due`'s
  pairing of a stored stamp against the system clock did not exist before.
  Also: BR-6, mastery was ABSORBING — `Queue` excluded mastered words, so they
  could never be answered wrong, so the count could only grow while recall
  decayed; it also contradicted `progress.go`'s own comment that `--play` decides
  what to stop offering. Mastery is now a reporting status and the 90-day interval
  does the rarity. BR-2, three of six intervals were asserted by nothing. BR-3,
  the "store and time and nothing else" claim went stale in FIVE artifacts when I
  widened the guard. BR-4, the clock guard grepped one spelling and missed
  `time.Since` and four others. BR-5, `LastBox` was an exported mutable var; it is
  a const over an array literal now.

- 2026-08-26: close round 2 — two findings, both mine. BR-11: the only two tests
  pinning this round's Critical both began with `t.Skipf("no tzdata")`, so on a
  machine without the system timezone database a real correctness fix would have
  had no check at all and reported green. The repo had already solved that one
  file away — `history_cmd_test.go` imports `_ "time/tzdata"` — and I wrote the
  skip instead. Both tests now embed it and `t.Fatal` on failure, and the
  zone-regression mutant reddens both. BR-3, second round open: I "swept" the
  stale import claim, ran a residue check that came back empty, and reported it
  done — while three of five sites stood, because I grepped one exact wording and
  the others said the same thing differently. The residue check inherited the
  sweep's blind spot. Swept by the concept this time.

- 2026-08-26: close round 3 — three findings. BR-12 was a genuine hole the earlier
  guards left wide: the import allowlist admits `store` WHOLESALE, and `store` is
  where the disk lives, so `store.NewYAML(dir, w)` inside `schedule` would open,
  read and write files while passing both guards — the purity claim false with
  everything green. Two guards agreeing is not two independent checks when they
  share a blind spot: both reasoned about NAMES, neither about what the named
  thing does. A third guard now lists the eight pure store symbols this package
  may use, and the constructor mutant reddens it.
  BR-3 and BR-11 were both open a further round because my own fixes minted new
  instances of the very classes they named: the atlas still said "a mastered word
  leaves the rotation", which is the behaviour BR-6 removed; and I added the
  tzdata import to one of two files while writing the comment claiming it into
  BOTH, so the second file asserted an import it did not have. A fix is not done
  until it is verified in every artifact it touched.
