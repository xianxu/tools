---
id: 000075
status: open
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
---

# define: mark unknown words inside a cloze question

## Problem

A review question is pass/fail on the answer word, and the learner has no way to
say that what stopped them was a DIFFERENT word in the stem.

`store.Item` carries a stem, an answer and its distractors; the sitting records
`Correct` on the answer. So a learner who cannot read the stem — a named
institution, a second hard word, an idiom — either guesses or fails, and in both
cases the program learns nothing about the word that actually blocked them. They
also cannot get it explained without leaving the sitting.

#67 built exactly this interaction for a pasted passage: click or drag the words
you cannot follow, press Enter, and they are explained. A cloze stem is a short
passage, and the same gesture is missing from it.

## Spec

### The gesture

Mark words in the stem the way #67 marks a passage — click for a word, drag for a
span — and Enter with marks triggers the explain flow. Each mark records a
`marked` event for its own word, which `CaptureMarked` already does.

### The scoring rule

Operator decision, 2026-09-17: **the item counts as wrong only when a marked word
IS the answer or one of its distractors.** Otherwise the item's outcome is
whatever the learner answers, and the marks stand as observations about their own
words.

The reason is mechanical rather than philosophical. `schedule.Progress` is keyed
per word, so scoring the item wrong moves the ANSWER word's schedule on the basis
of ignorance of a different word — and a learner who marks a hard name in the
stem would have the answer word they knew perfectly well pushed back up the
queue. Comparing the marked word against the answer and the distractors is one
comparison, and it keeps both schedules honest.

Accepted consequence, stated so it is not mistaken for an oversight: a learner who
knows the answer and marks another word purely out of curiosity is not penalised.
That is the intended behaviour, not a loophole.

### It is the strongest sense signal available

A word marked inside a stem is disambiguated by that stem — the program already
knows which sense the item was built from. That makes this the best signal for
#74's sense mask, better than a lookup (which carries no sense) and stronger than
a wrong answer (which is free but unacknowledged). The mark is an explicit "I do
not know this", with the meaning pinned.

### The premise to check before sizing

**Unverified:** #67's marking operates on a `passage` rendered into the screen's
footer, with its own coordinate space (`wordAtCell`, `cellRangesFor`) and its own
ownership rules (`ownsBufferLine`). A cloze question is drawn by the board in
`play_loop.go`, a different surface with a different lifecycle. Whether the
mark, drag and explain machinery composes onto the board — or needs the board
re-expressed as something markable — is unknown, and it is the difference between
a small issue and a large one. First plan step, before any estimate.

## Done when

- Clicking a word in a cloze stem marks it; dragging marks a span; Enter with
  marks explains them — the #67 gestures, asserted through the same pty
  conformance shape #67 used, not through a unit test of the handler.
- A mark on a word that is neither the answer nor a distractor leaves the item's
  outcome as answered, and leaves the ANSWER word's schedule untouched —
  asserted against the queue, since that is the whole reason for the rule.
- A mark on the answer, or on any distractor, scores the item wrong.
- Every mark records a `marked` event for its own word regardless of scoring.
- The sense the stem was built from is recorded with the mark, so #74 can read it.
- Marks clear when the question is left, on every exit path — #67's "marks clear
  iff an answer reached the reader" predicate is the precedent, and its lesson was
  that collapsing the exit paths into one predicate is what keeps them correct.
- Ctrl-C mid-explain returns to the sitting with it intact.
- `atlas/define.md` records the gesture, the scoring rule and its reason.

## Plan

- [ ] verify whether #67's mark/drag machinery composes onto the board, or
      whether the board needs re-expressing — this sizes the issue
- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] the gesture on the board, with the pty conformance rows
- [ ] the scoring rule and its queue assertion
- [ ] the sense recorded with the mark, for #74
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-17

Filed out of #68's brainstorm, from the operator's observation that the sense
problem needs a signal and that a cloze question is a place the learner already
knows they are stuck. The scoring rule was narrowed during that discussion from
"a mark makes the question wrong" to the answer-or-distractor test, because
`Progress` is per word and the broad rule would move the wrong word's schedule.
