---
id: 000022
status: open
deps: [tools#21]
github_issue:
created: 2026-08-26
updated: 2026-08-26
estimate_hours:
---

# active learning set: words graduate out of highlighting

## Problem

#21 highlights every word in the deck, everywhere it appears. That is the right
v1 — one rule, no surprising suppression — but it has a built-in decay: the deck
only grows, so the screen gets greener forever and the signal thins out. A
learner who has looked up 400 words does not need 400 words flagged; they need
the twenty they are still working on.

Operator's framing, which is the point of the issue:

> I imagine as user use this program and some of the words become theirs, and
> those highlighting would stop.

Highlighting should mark *effort still owed*, not *history of contact*.

## Spec

Not designed yet. The seam is already in place: #21 puts every highlight
decision behind `Vocabulary.Has(word)`, so this issue is about what fills that
set, not about the rendering. Nothing downstream of the seam should need to
change.

Open questions for the brainstorm:

- **What graduates a word?** Candidates: N successful reviews (#6's review
  events), elapsed time without a re-lookup, an explicit "I know this" gesture,
  or a spaced-repetition interval crossing a threshold. #5 (scheduling) and #6
  (play) are the natural sources; this issue may want to wait for them.
- **Does a graduated word ever come back?** Looking it up again is decent
  evidence the graduation was wrong.
- **Is graduation visible?** A word going quiet is a reward; a silent change the
  learner cannot see or dispute is a mystery.
- **Does `--stats` (#8) report the active set?** "You are working on 18 words"
  is a better number than "your deck has 412".

## Done when

- [ ] Highlighting reflects an *active* set, not the whole deck.
- [ ] The graduation rule is explainable in one sentence to the learner.
- [ ] `Vocabulary`'s consumers (typed line, definitions, answers) are unchanged.

## Plan

- [ ] Brainstorm after #5/#6 land — the graduation signal probably lives there.

## Log

### 2026-08-26

Filed from the operator's request while specifying #21, explicitly as the later
half: "later we should have the concept of words user is actively learning and
only highlight those."
