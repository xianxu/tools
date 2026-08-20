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

## Done when

- [ ] Four options, exactly one correct, drawn from the local deck.
- [ ] Deterministic under a fixed seed.
- [ ] Degrades sensibly when the deck has fewer than four words.
- [ ] Works with the network off.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.
