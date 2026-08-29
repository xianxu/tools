---
id: 000007
status: open
deps: ["tools#6"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# review form 2.3: meaning multiple choice from the local deck

## Problem

Recognition needs testing, not just recall. And the whole review loop should
work with no API key and no network.

## Spec

Form 2.3: given a word, choose its definition from four options.

- **Distractors come from the learner's own deck**, not from a model: offline,
  free, deterministic, and pedagogically better — the wrong answers are words
  they are actually confusing right now.
- Pick distractors that are *far* in meaning from the answer (see #12's
  selection rule), so the question has one defensible answer.
- Pure question construction: deck + target + seed → question. Same seed, same
  question, which is what makes it testable.
- This form plus #6 is a complete trainer needing neither network nor key. That
  is the M1 boundary.

**Record the CHOSEN distractor, not just right/wrong.** This is #17 M2's error
taxonomy arriving for free, and it is the reason M2 was not filed as its own
issue when #17 closed at M1.

The learner model (`user-model.md`, shipped) carries Level and Domains, and #17
reserved a third section — Weaknesses — for misses classified by KIND
(near-synonym collapse, connotation, register, domain). That was blocked on
review events, which now exist; but `store.ReviewEvent` records `Correct bool`,
a binary verdict that cannot carry a kind. Classifying post-hoc would mean a
model call per miss.

In THIS form the classification is already in the learner's hand: the distractor
they picked IS the error kind, since distractors are drawn from their own deck
and selected by semantic distance. Recording which one was chosen costs a field
and no inference. Whatever this form does with the answer, keep the identity of
the chosen option rather than collapsing it to a boolean — the weakness section
is downstream of that one decision.

## Done when

- [ ] Four options, exactly one correct, drawn from the local deck.
- [ ] The CHOSEN option is recorded, not just correctness — see the note above;
      collapsing it to a boolean is what makes the weakness taxonomy expensive
      later.
- [ ] Deterministic under a fixed seed.
- [ ] Degrades sensibly when the deck has fewer than four words.
- [ ] Works with the network off.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-27

- #17 closed at M1. Its M2 (weaknesses, from an error taxonomy) was NOT filed as
  a separate issue: the data it needs does not exist yet, and this form produces
  it for free. Recorded above as a Done-when row rather than a tracked issue, so
  it is met at the moment it is cheap instead of becoming a backlog item whose
  first design question is "wait for #7".


## Revisions

### 2026-08-28 — the same rule, and a tension this form has to settle

**Reason.** The project's 2026-08-28 decision changes the distractor rule to
**same domain (or general vocabulary), at the learner's CEFR band or one below**.
This Spec says *"pick distractors that are far in meaning from the answer (see
#12's selection rule)"*, so it inherits by reference.

**Delta.**

- **Inherits the new rule.** Domain and band replace semantic distance as the
  primary filter. The pool for this form is unchanged — the learner's own deck —
  so nothing here depends on `#10`'s harvest.
- **This form stays no-LLM, and that is now a positive result rather than an
  assumption.** `#12` can afford plausible distractors because it has a model
  veto. This form has none by design. It is safe anyway, because the options here
  are DEFINITIONS: two different words rarely share one, so a same-domain
  distractor does not create a second correct answer the way it does in a cloze
  blank. Domain-matching here only removes the giveaway where three options are
  obviously medical and one is legal.

**An OPEN TENSION this issue must settle when it is built — do not resolve it by
inheriting.** Two claims in this Spec pull against each other, and the new rule
sharpens the conflict rather than causing it:

1. *"Pick distractors far in meaning"* — which the near-synonym guard requires.
2. *"The distractor they picked IS the error kind"*, with the kinds named as
   near-synonym collapse, connotation, register, domain.

If every distractor is far in meaning, choosing one says only *"did not know
it"*, never which KIND of confusion — so the error taxonomy `#17 M2` was folded
into this issue for cannot be read off the choice. Making the option set
diagnostic needs distractors that VARY along those axes, which means deliberately
including a near-synonym — and a near-synonym's definition may plausibly define
the target, which is the ambiguity the guard exists to prevent.

Same-domain selection moves this form toward diagnosis and toward ambiguity at
the same time. Whether the taxonomy survives, or `#17 M2` needs its own path
after all, is a question for this issue's design — it should be answered
deliberately rather than discovered at a review boundary.
