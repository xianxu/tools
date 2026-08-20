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
