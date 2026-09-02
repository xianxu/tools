---
id: 000042
status: blocked
deps: [tools#40, tools#44]
github_issue:
created: 2026-09-01
updated: 2026-09-02
estimate_hours:
started: 2026-09-02T12:31:27-07:00
---

# retire form 2.1: the board is the fallback when a real test cannot be built

## Problem

**Form 2.1 and the board are the same instrument, and the board is sixteen times
cheaper.** Both implement `SelfRated`: `y` means "I knew it" and nothing checked.
Neither retrieves anything. 2.1 spends a whole screen and a keystroke per word to
collect a claim the board collects sixteen at a time.

Operator, 2026-09-01, from a real sitting — with the board on screen and form 2.1
having just asked about `comports`: *"with the better recall form, this is not
useful anymore I think?"*

**Its remaining population is already small and is the worst place for
self-report.** `#40`'s `boardsFor` partitions on the box FIRST, so form 2.1 now
fires only for a word that is BOTH box ≤ 2 AND one the deck cannot build
distractors for — a young deck's first sittings. Self-report is least reliable
exactly there, because the word has not been learned yet and the illusion of
knowing is strongest.

**The one thing 2.1 does that the board does not is REVEAL THE WHOLE ENTRY on a
miss.** `Recall.Reveal()` returns the full rendered entry; the board's panel is
one line, deliberately — a twenty-row panel would mean a board no terminal fits
(`#40` R1). That is the trade this issue accepts.

## Spec

**Delete form 2.1. A word that cannot be tested is triaged instead.** The form
set becomes legible in one sentence: **2.3 tests you, the board triages you.**

Selection becomes a single rule, and it is a WIDENING of `#40` D4 rather than a
replacement:

| word | form |
|---|---|
| box ≥ 3 | the board (unchanged — `#40` D4) |
| box ≤ 2, distractors available | form 2.3 (unchanged) |
| box ≤ 2, distractors NOT available | **the board** (was 2.1) |

**Operator chose this over the alternative, which was to keep the fallback as a
STUDY CARD** — word and definition shown together, any key to move on, no y/n at
all, recording `Skipped` and moving no box. That option kept the full reveal and
conceded the same point (the self-check is the part that stopped being useful).
It was declined in favour of one fewer form. Recorded because a deleted option is
worth arguing with later: if the young-word reveal turns out to matter, the study
card is the thing to build, not 2.1 restored.

**The risk is bounded by the ladder, which is why this is affordable.** `#40` D4's
table: boxes 0 and 1 are literally free — `box.go` waits one day at both — so a
wrongly-promoted young word returns tomorrow regardless. Box 2 buys one day. The
words moving from a self-rated form to a self-rated form are the ones where being
wrong costs least.

### What this touches

- **`todaysQuestions` must decide AFTER the lookup.** Today `boardsFor` runs on
  keys before anything is looked up, because the box is all it consults. "Can
  this word have distractors" is only knowable once the entry is parsed, so the
  partition moves after the render loop.
- **`fallbackReasons`** (`optionpool.go`) is a DECLARED list — it exists because
  the same enumeration had drifted across the code, the README and the atlas. It
  stops being "why a word goes to form 2.1" and becomes "why a word is triaged
  rather than tested". `TestREADMENamesEveryFallbackReason` is its pin.
- **`play.Recall` and `recall_test.go` are deleted.** Thirty-odd test references
  use it as a convenient one-word `Question`; those want a fake form, not a
  shipped one. `TestSessionIsFormAgnostic` already has the right shape.
- **A board that cannot be DRAWN is the new edge.** `#40` D15 sends a board the
  terminal cannot hold whole to form 2.3 — which is exactly the form that was
  unavailable here. A single-word board needs five rows; below that the word is
  skipped for the sitting with a note, like a word the dictionary cannot find.
- **Docs:** the README's "Recall, on a young deck" section, the atlas's forms
  section, and `schedule`/`play` doc comments that name 2.1 as the fallback that
  always works.

## Done when

- [ ] A word with no usable distractors reaches a BOARD, at every box, pinned by a test over a deck that spans the threshold and asserts both sides.
- [ ] `play.Recall` does not exist, and no test substitutes a shipped form where it wants a double.
- [ ] The declared fallback reasons still derive into the README, and still name three — the reasons did not change, only what they select.
- [ ] A word that can be neither tested nor drawn is skipped with a message that names the cause, and the sitting continues.
- [ ] Measured: the sitting a young deck gets is no longer than it was — the board packs, so this must not add screens.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-09-01

Filed from `#40`'s first operator sitting. The screenshot that prompted it showed
`comports` under form 2.1 — an "entry defines a different word" fallback, since
NOAD sends `comports` to `comport`.

**Depends on `#40`** rather than being part of it: `#40` has been through four
boundary-review rounds and still has an open Critical (a narrowing resize under a
live board). Widening a closing issue to delete a form is the scope creep the
constitution warns about, and the selection rule this changes is `#40`'s own D4.
