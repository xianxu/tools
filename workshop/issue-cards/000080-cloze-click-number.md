---
id: '000080'
status: done
started: 2026-09-23T16:48:56-07:00
created: 2026-09-20
updated: 2026-09-27
actual_hours: 5.31
---

# define: click a cloze option's number to answer it

## Problem

Operator, 2026-09-20:

> create a task to support click to select from the cloze answer choices, by
> clicking on the [1] [2] numbering. the words themselves have underscore so
> clicking means to pronounce.

A cloze question is answered from the keyboard only: press `1`–`4`. The mouse is
already live on that same prompt — each option word is underlined and clicking it
**pronounces** it — so a learner with a hand on the mouse has no way to *answer*
with it, and the one thing the pointer can do on those rows is the thing that must
not answer.

So the gesture is split by target: **the number answers; the word speaks.** ("Underscore"
in the request is read as the underline `markClickable` splices under a deck word.)

What exists today, so the change is measured against it:

- The option rows are printed `1  keel` — `optionLine` (`play/choice.go:195`) is the
  digit and two spaces, **no brackets**. `[1]` in the request is taken as shorthand
  for "the number", not a glyph to add; whether to draw brackets is a design
  question below.
- The option *words* are `RegionWord` regions from `deckSpans` (`deckwords.go`),
  found by walking the same text that decides colour. A click on one plays the word
  through `playRegion`; `TestClozeOptionsAreClickableButNotColoured` pins that they
  are clickable and uncoloured.
- The digit is not in any region, so a click on it is *ordinary text* — nothing,
  by design ("A click on ordinary text is NOTHING: no beep, no message").
- Answering by pointer exists for exactly one form. `formCell` (`play_loop.go:819`)
  offers a click to the question first and only a `play.Grid` (the board) takes it,
  becoming `InputMark`. Everything else declines and falls through to `playRegion`.
  The atlas states the invariant: *a click never answers a form that did not ask
  for it* (D8).
