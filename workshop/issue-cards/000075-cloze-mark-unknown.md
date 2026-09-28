---
id: 000075
status: open
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
github_issue:
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
