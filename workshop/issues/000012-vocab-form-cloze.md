---
id: 000012
status: open
deps: ["tools#6", "tools#10", "tools#11"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# review form 2.2: cloze from current news with curated distractors

## Problem

A word met in a real, current sentence sticks better than one met in a
dictionary. This is the form the whole news pipeline exists for.

## Spec

Form 2.2: a real news sentence with the word blanked, and four options.

```
"Critics called the memo ______, a transparent attempt to flatter the board."
  1) sycophantic   2) ephemeral   3) defenestrate   4) obsequious
```

**Distractors are selected, never invented.** The candidate pool is:

1. words harvested from current news at the learner's level (#10), and
2. words already in the learner's deck (#4),

filtered to those whose meaning is **substantially different** from the answer.
Selecting from real words removes the failure mode where a generated distractor
happens to be correct — the option set is drawn from a pool whose members are
known words with known definitions.

Semantic distance, cheapest first: different part of speech, no shared
definition terms, different NOAD domain label. The model (#11) is used only to
**veto** a candidate that would also fit the blank — a yes/no check on a
concrete pair, which is far more reliable than open generation, and skippable
when the seam is unavailable.

Note the example above: `obsequious` is a *near-synonym* of `sycophantic` and
must be rejected by the distance filter. That case belongs in the tests.

- Sentence comes from the cache (#9/#10), so the question is offline once
  harvested.
- Bad-question keypress (#6) records the flag with the full option set, so a
  filter failure is diagnosable rather than anecdotal.

## Done when

- [ ] Options are drawn from the pool + deck, never model-generated.
- [ ] A near-synonym of the answer is rejected as a distractor — asserted with
      `sycophantic`/`obsequious`.
- [ ] The form works with the LLM seam unavailable (veto step skipped).
- [ ] The blanked sentence never leaks the answer (stem, plural, hyphenation).
- [ ] Deterministic under a fixed seed.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-08-22 — the stem is authored; the options are still selected

**Reason.** Operator, 2026-08-22: real usage is raw material, not a question — the
model has to process and rephrase before it is usable. Measured support in #10's
revision.

**Delta.**

- **The stem is authored by the model** (#10, offline, ahead of time) rather than
  being a real sentence with a word blanked. The example in the Spec above is now
  what the *output* looks like, not what the input looks like.
- **The form reads a finished item** from the store instead of assembling one at
  question time.

**Explicitly unchanged — the load-bearing rule.** *Distractors are selected, never
invented.* The options still come from the level-matched pool and the learner's own
deck, and the model's only role in the option set is to **veto** a candidate that
would also fit the blank. The 2026-08-20 decision stands: selecting from real words
with known definitions makes "the generated wrong answer is also right" impossible
by construction rather than by a check that has to hold.

`obsequious` must still be rejected as a distractor for `sycophantic`. That test
does not move.

**Added.** Distractor *domain* now follows the learner (#17): for someone whose
lookups are 34% judicial, the interesting confusion is `dicta` against `holding`,
not `dicta` against `ephemeral`.

### 2026-08-28 — same domain, band or below, and the veto becomes load-bearing

**Reason.** The project's 2026-08-28 decision replaces *"filtered for
substantial semantic difference"* with **same domain (or general vocabulary), at
the learner's CEFR band or one below**. Rationale in the project's
`## Decisions`.

**Delta.**

- **The pool changes source.** No longer *"words harvested from current news at
  the learner's level (#10)"* — it is a static level-and-domain-tagged
  vocabulary plus the learner's deck. *Selected, never invented* is unchanged and
  still holds: the words are real, from a real list.
- **The selection rule changes shape.** Semantic distance is no longer the
  primary filter; domain and band are. Distance survives only as the
  near-synonym guard below.
- **The model veto is now LOAD-BEARING, not a nicety.** Same-domain, same-band
  words are likelier to also fit the blank — that is exactly what makes the item
  good and exactly what raises the "the wrong answer is also right" failure mode
  the original rule was written to kill. This form can afford plausible
  distractors *because* it has the veto; that is what earns it.
  - The Done-when row *"the form works with the LLM seam unavailable (veto step
    skipped)"* now carries a cost it did not before: without the veto, a
    same-domain distractor may genuinely fit. Either the offline path widens the
    domain filter, or it accepts a rarer ambiguous item. Decide it when building.
- **`sycophantic`/`obsequious` stays the test case.** A near-synonym must still
  be rejected, and under the new rule it is *more* likely to be selected, not
  less — same domain, same band. The guard matters more.
