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

