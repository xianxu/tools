---
id: 000017
status: working
deps: [tools#3, tools#11]
github_issue:
created: 2026-08-22
updated: 2026-08-25
estimate_hours: 5.23
started: 2026-08-25T16:45:07-07:00
---

# learner model: batch analysis into a durable user-model.md

## Problem

Nothing in the trainer knows who it is teaching. `--stats` (#8) counts answers;
it does not know that this learner reads judicial opinions for pleasure, sits at
C1–C2, and consistently mistakes contemptuous praise words for neutral ones.

Without that, every generated item is generic — and generic is exactly what a
personal tool has no excuse to be. The words a person looks up *are* a signal a
vocabulary app cannot buy.

## Spec

One durable artifact, `user-model.md`, in the same working directory as `words/`
and `events/`. Batch-generated, human-correctable, and read by every authoring
prompt.

### Why a document, not a table

It is meant to be read and argued with. A learner who disagrees ("I read these for
pleasure, not for the bar exam") must be able to say so and have it stick — which
is also the escape hatch for a batch analysis that draws the wrong conclusion.

### Shape

```markdown
---
type: user-model
learner: <name>
updated: <ISO date>
window: <from>..<to>          # N lookups, M reviews
generated_by: define --reflect (<model>)
---

## Level
Working band, anchored on evidence from the deck rather than asserted.

## Domains they read in
Inferred from what gets looked up — never asked. Table of domain, share, and the
words that are the evidence. Carries the authoring directive that follows from it.

## Weaknesses
Error kinds with counts, each naming the events it was derived from.

## Corrections          ← human-owned
```

### Rules

- **Batch, never per-answer.** `define --reflect` folds the event log and the deck
  into this file on demand. No model call ever sits inside the review loop or the
  lookup path; review stays instant and works offline.
- **`## Corrections` is never rewritten.** Regeneration replaces everything above
  it and appends nothing to it. Corrections are authoritative over anything
  inferred, and the authoring prompt is told so explicitly.
- **Every claim names its evidence.** "34% of lookups are legal — `certiorari`,
  `dicta`, `estoppel`" is checkable; "you like law" is not. A claim that cannot
  name the events behind it does not go in the file (`lessons.md`: a claim must not
  outrun the width it was measured at).
- **Regeneration is idempotent under a fixed clock, fake seam and fixed store.**
  Same inputs, same file — which is what makes it testable at all.
- **Absent file is normal.** Authoring works without it, just generically; nothing
  blocks on it existing.

### Milestones

- **M1 — from lookups.** Level and domain need only the deck and the lookup events,
  both of which exist today (#3, #4). Lands before any review events do, so the very
  first authored item is already learner-aware.
- **M2 — weaknesses.** The error taxonomy needs review events from #6, and the
  misses classified by kind (near-synonym collapse, connotation, register, domain,
  right-meaning-wrong-usage). Feeds back into which words surface and what the next
  item is authored to probe.

## Done when

M1:
- [ ] `--reflect` writes a `user-model.md` whose level and domain claims each name
      the deck words they were derived from.
- [ ] Regeneration is idempotent against a fake seam, fixed clock and fixed store.
- [ ] A hand-written `## Corrections` section survives regeneration byte-for-byte,
      asserted by a test that fails when the preservation is removed.
- [ ] Authoring (#10) reads the file, and its absence degrades to generic authoring
      rather than an error.
- [ ] Domain inference is checked against a held-out sample of deck words, not
      asserted — the same bar #10 sets for level bucketing.

M2:
- [ ] Misses classify into a fixed, enumerated error taxonomy; an unrecognised
      classification is dropped with a warning, not admitted as a new kind.
- [ ] A weakness claim names the events behind it, and the count is reproducible
      from the log.
- [ ] The model demonstrably changes what is authored: same deck, two different
      `user-model.md` files, different distractor domains — asserted, not asserted-ish.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), against
`workshop/plans/000017-user-model-plan.md` — one item per task, plus the process
work that sits inside the measured window.

**Costed with #16's overrun as evidence, not with its estimate.** #16 estimated
6.49 and measured **17.63** (0.4×). The feature itself came in at ~6.3h, almost
exactly as costed; the other ~11h was twelve review rounds, and per-milestone
actuals under-report because those rounds land at and after the boundary they
measure. So the review rounds are line items here rather than a hope: two plan
rounds (already spent) and four boundary rounds, which is what #16 needed for
each of its milestones.

Design carries v2's ×0.2 spec-quality discount where the plan pre-resolves the
decision — it does for the pure entities, and does not for the plan authoring
itself or for the review rounds, which are the design work rather than a
beneficiary of it. Implementation is v3.1's 40% of the v2 table. Familiarity 1.0:
same repo, same files, `summariseLookups` and the store seams all shipped.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=1.00 impl=0.08
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.02 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
design-buffer: 0.15
total: 5.23
```

| item | what |
|---|---|
| issue-spec | plan authoring — inside the measured window, as #16 established |
| milestone-review ×2 | the plan-quality rounds (PQ-1…PQ-5), already spent |
| smaller-go-module | T1 `foldLookups`, reusing `summariseLookups` |
| smaller-go-module | T2 `learnerModel` + `checkEvidence` |
| smaller-go-module | T3 `renderUserModel` + its golden |
| greenfield-go-module | T4 `spliceCorrections` + the fuzz property |
| greenfield-go-module | T5 `reflectTask` — the prompt, and prompts are design |
| greenfield-go-module | T6 `runReflect`, the flag, seven wiring tests |
| smaller-go-module | T7 the live conformance check for domain inference |
| milestone-review ×4 | boundary rounds — #16 needed four per milestone |
| milestone-review | the M1 close itself |
| atlas-docs | the `--reflect` atlas section and the project row |

**What would make this wrong in the other direction:** M1 has no terminal work,
no interrupt semantics and no new store implementation — the three things that
generated most of #16's second-order findings. If the review rounds converge in
two rather than four, this lands nearer 4.3.

## Plan

Design: [`workshop/plans/000017-user-model-plan.md`](../plans/000017-user-model-plan.md)
— M1 only; M2 needs review events #6 does not yet produce.

- [x] Design via `sdlc start-plan` before implementing.
- [ ] M1 — the model from lookups: `foldLookups`, a typed `learnerModel` whose
      evidence is CHECKED against the deck, `renderUserModel` +
      `spliceCorrections`, and `--reflect` as a mode beside `--llm-check`.
- [ ] M2 — weaknesses. Blocked on #6's review events; planned when they exist.

## Log

### 2026-08-22

Created from the operator conversation that broadened `define-learn` to an adaptive
program: *"keep one user model of where user is, what area/words they seem to know
more… if the words appear more in supreme court cases, then the comparables and
wrong options would be drawn from those areas more. the whole thing should be
adaptive."*

Batch analysis over accumulated errors was specified in preference to diagnosing
each miss as it happens.

### 2026-08-25

`sdlc start-plan` run; durable plan at `workshop/plans/000017-user-model-plan.md`,
**M1 only** — M2's error taxonomy needs review events #6 does not yet produce, and
planning against a data shape nobody has seen is how a plan becomes fiction.

The decision worth surfacing before code: **the model's evidence is checked, not
trusted.** The typed answer carries the deck words behind each claim and
`checkEvidence` drops any claim citing a word the deck does not hold. That is
this project's existing rule — *distractors are selected, never invented* — applied
to the learner model: the model may READ the deck and may not ADD to it. Without
it, "every claim names its evidence" is a formatting convention that a plausible
hallucination satisfies, and the file's whole promise is that a claim can be
checked.

Two smaller ones: a floor of 12 deck words, because a model built from four
lookups is noise that would then steer authoring (absence already degrades
cleanly — #16's `gatherAskContext` handles it); and `## Corrections` is spliced
by scanning outside fenced code blocks, because the file documents its own format
in a fence that contains the marker.

The fourth M1 Done-when row names #10, which does not exist. #16's ask path reads
`user-model.md` today and degrades on absence, so the row's substance has a live
consumer already; Task 8 verifies that rather than assuming it.
