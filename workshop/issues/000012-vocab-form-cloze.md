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

**Rewritten 2026-09-06, when `#10 M2` landed** — the trigger the 2026-09-04
Revision below named. Three rows moved to `#10` with the selection they belong
to; they are recorded here as SATISFIED rather than deleted, so a reader can see
where they went instead of building them again.

- [x] Options are drawn from the pool + deck, never model-generated. — **Satisfied
      by `#10 M2`, and more strongly than review-time filtering managed.** Options
      are SELECTED offline from the banded deck and stored finished; `authoredStem`
      has no distractors field at all, so "never model-generated" is structural
      rather than a rule this form has to follow.
- [x] A near-synonym of the answer is rejected as a distractor — asserted with
      `sycophantic`/`obsequious`. — **Satisfied by `#10 M2`'s veto**, which carries
      exactly this pair as its committed known-bad case and fired on it in all
      three live checkpoint batches, in both directions. Under the new selection
      rule the pair is MORE likely to be chosen, not less, which is why the veto
      is what earns the plausibility.
- [x] The form works with the LLM seam unavailable (veto step skipped). —
      **Satisfied by construction, and the cost the 2026-08-30 Revision recorded
      is gone with it.** An item read from disk carries finished, already-vetoed
      options, so there is no degraded offline path to design: the veto ran once,
      when the item was written. There is no "veto step" left to skip.

**What is still this issue's to build**, and it is the whole of the remaining
work:

- [ ] The blanked sentence never leaks the answer (stem, plural, hyphenation).
      `#10`'s `blankOut` is a PROMPT-SHAPING helper for the veto and explicitly
      not this — it is deliberately simple, and here it is the LEARNER who must
      not see the answer.
- [ ] Deterministic under a fixed seed.
- [ ] The bad-question keypress records the flag with the full option set.

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

### 2026-09-04 — selection and the veto MOVE to `#10`; this issue keeps rendering

**Reason.** `#10`'s plan-quality gate (round 1, PQ-3) caught that `#10` was
building distractor selection and the near-synonym veto while three Done-when
rows here still owned them. Declaring the move now rather than discovering the
overlap at one of the two closes.

**Delta.** `#10` authors finished items offline — stem, answer, and distractors
already selected at the learner's band or one below and already vetoed. So:

- *"Options are drawn from the pool + deck, never model-generated"* — **satisfied
  by construction, and more strongly than review-time filtering managed.** Options
  are SELECTED from the banded deck at authoring time; nothing generates them.
- *"The form works with the LLM seam unavailable (veto step skipped)"* — **now
  unconditional, and the cost recorded in the 2026-08-30 revision above
  disappears with it.** A form reading a finished item never reaches for a model,
  so there is no degraded offline path to decide between: the veto already ran,
  once, when the item was written. That was the open question this issue was
  carrying; `#10` answers it by moving the work earlier rather than by widening a
  filter.
- *"A near-synonym of the answer is rejected"* — the `sycophantic`/`obsequious`
  case **moves to `#10`** as the veto's committed known-bad row. Same assertion,
  earlier in time.

**What stays here, and it is still a real issue:** rendering an authored item as
a cloze question — blanking the stem without leaking the answer through stem,
plural or hyphenation — and determinism under a fixed seed. Plus the bad-question
keypress recording the full option set.

**Applied 2026-09-06**, at `#10 M2`'s boundary — see the rewritten Done-when
above. A deferral whose trigger is "when Mx lands" is swept by the issue that
WROTE it, at Mx's boundary, rather than left for the consuming issue to
discover: until this was done, a reader of `#12` would have built all three
again.
