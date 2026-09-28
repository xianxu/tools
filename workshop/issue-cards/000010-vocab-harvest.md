---
id: '000010'
status: done
started: 2026-09-04T09:38:50-07:00
created: 2026-08-20
updated: 2026-09-07
estimate_hours: 7.94
actual_hours: 12.77
---

# authored practice items: level-tagged words, and stems the model writes offline

## Problem

A multiple-choice question is only as good as its wrong answers, and asking a
model to invent them is how a distractor ends up being *also* correct. The fix is
to stop GENERATING distractors and start SELECTING them from real words at a
known level.

That needs two things this program does not have. **Words have no level**, so
"at the learner's band, or one below" is not yet arithmetic on anything — `#17`
assigns the LEARNER a band and nothing assigns one to a word. And **there are no
items to select into**: a raw sentence is not a question. Measured against the
live feed for `sycophantic` (2026-08-22, 100 items): blanking *"Sycophantic AI
decreases prosocial intentions"* produces a question its own sentence does not
entail, and 10 of the first 14 matching headlines were about AI chatbots, so the
learner would acquire the collocation rather than the word.

So the material has to be AUTHORED — read across usages, write a stem that
entails its answer — and authored **ahead of time**, because a review sitting
must stay instant, free and offline.
