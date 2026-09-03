---
id: 000042
status: working
deps: [tools#40, tools#44]
github_issue:
created: 2026-09-01
updated: 2026-09-02
estimate_hours: 3.89
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

### The board learns to DROP, because it is about to become the only form some words see

**Widened 2026-09-02, operator's call — and the premise it was decided on was
partly WRONG, which is recorded here rather than quietly repaired.**

What is true: on a board `d` is cell 13, not a drop (`#40` D12 — a grid has no
single current word, so the session hands the key to the form). So sending
untestable young words to the board removes the IN-SITTING drop for exactly the
population most likely to need it: a typo'd capture, a word looked up once. On a
deck of one to three words that is every word.

What was claimed and is false: that this is the *only* drop path. **`define
--forget <word>` exists** (`main.go:425`, dispatched as a mode at `:568`, calling
`store.Forget` at `:1032`), so `store.Forget` has TWO callers and a learner can
always remove a word from the command line. The decision to widen this issue was
put to the operator on the stronger, false version. The honest framing is
narrower: **`--play` would lose the ability to curate the deck from inside the
sitting, for words the sitting itself surfaces** — which is where you notice you
do not want a word, and leaving is a context switch — not that curation becomes
impossible.

**Proposed mechanism: `Tab` cycles THREE modes — yes, no, drop.** The board
already owns a mode, already draws it on the prompt row, and already lands it with
the same keys and clicks; a third value costs no new gesture and no new key. The
prompt row is also the last thing a short window gives up (`#40` R11), so the live
mode is always legible.

- **Rejected: a modifier or an uppercase key.** `Grade` folds case deliberately,
  so `D` already means cell 13; un-folding it to mean "drop d" would make the one
  key this issue is about mean two things depending on shift.
- **Rejected: a prefix gesture** (`x` then a cell). Non-sticky, which is safer,
  but it is a second input grammar on a form whose whole argument is that one
  keystroke does one thing.
- **The risk a mode carries is stickiness**, and it is bounded rather than
  dismissed: `Forget` does NOT remove events (`store.go`), so a dropped word
  returns to the deck with its history intact the next time it is looked up. A
  board therefore closes with a `dropped: …` line beside its `relearn: …` one, so
  the action is in the transcript and the recovery is a lookup.

**Also in scope, because this issue is about that key:** `Board.Grade`'s doc still
says *"`d` and `D` never arrive: toInput takes them first, and boardLabels has no
cell for them either way."* Both halves are false since `#40` put `d` back in the
label alphabet. A comment describing the opposite of the code, on the key this
issue changes, is fixed here.

## Done when

- [ ] A word with no usable distractors reaches a BOARD, at every box, pinned by a test over a deck that spans the threshold and asserts both sides.
- [ ] `play.Recall` does not exist, and no test substitutes a shipped form where it wants a double.
- [ ] The declared fallback reasons still derive into the README, and still name three — the reasons did not change, only what they select.
- [ ] A word that can be neither tested nor drawn is skipped with a message that names the cause, and the sitting continues.
- [ ] Measured: the sitting a young deck gets is no longer than it was — the board packs, so this must not add screens.
- [ ] A word can be dropped from a board, by key and by click, and the drop reaches `store.Forget` — pinned end to end rather than at the form, since the form is not what removes anything.
- [ ] The live mode is unambiguous on the prompt row in all three states, and a board closes naming what it dropped.
- [ ] `Board.Grade`'s doc no longer claims `d` never arrives.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**Revised upward from 3.33 after the estimate-quality judge, before any code was
written.** The gate PASSED at 3.33; the revision is recorded rather than quietly
made, because an estimate known to be light pollutes the calibration ledger it
feeds, and the ledger is the only thing that can catch this repo's drift.

**TWO NAMED DEVIATIONS from v3.1's primitive units**, stated because a block that
claims fidelity while quietly deviating is the dishonest kind:

1. **Two `milestone-review` rows for a single-boundary issue.** The primitive's
   unit is one review of one chunk, and the Plan is explicitly single-pass. Booking
   two prices ROUNDS, not boundaries — `#44` shipped on this exact surface against
   one booked review and took FIVE, closing est 2.16 / actual 4.60.
2. **`ux-rename-iteration` at `#40`'s price (0.55/0.1), not one round's.** The
   record already shows two operator rounds on this issue, and the second — the
   drop mode — landed after `started:`, so it is inside the measured window
   already. `baseline-v2.1.md` says plan for 3–5 rounds per TUI-heavy milestone,
   not 1.

**Design discounting is NOT uniform, which the earlier prose wrongly implied.**
The v2.1 thorough-plan discount applies to the two `tui-screen` rows and the
`cross-cutting-refactor` row, whose decisions the plan resolved. It does NOT apply
to `ux-rename-iteration` — a plan cannot pre-resolve a sitting — and the small
rows sit at table midpoints because a discount below their floor would be noise.

**One thing this estimate does NOT do**, and it is deliberate: the repo's four most
recent rows (`#38` 0.67, `#40` 0.48, `#41` 0.69, `#44` 0.47) are same-direction
misses, and at that trailing ratio 3.89 predicts 6–8h. Multiplying the total to
meet it would be back-fitting — the primitives are the method, and `#117`'s
calibration ledger is where a systematic ratio belongs. Recorded here so the next
reader knows the gap was seen rather than missed.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module        design=0.15 impl=0.18
item: tui-screen               design=0.35 impl=0.34
item: cross-cutting-refactor   design=0.05 impl=0.18
item: tui-screen               design=0.35 impl=0.36
item: smaller-go-module        design=0.05 impl=0.1
item: smaller-go-module        design=0.02 impl=0.16
item: smaller-go-module        design=0.02 impl=0.1
item: atlas-docs               design=0.1  impl=0.08
item: milestone-review         design=0.0  impl=0.2
item: milestone-review         design=0.0  impl=0.2
item: ux-rename-iteration      design=0.55 impl=0.1
design-buffer: 0.15
total: 3.89
```

| row | what it is |
|---|---|
| `smaller-go-module` #1 | `optionsFor` split out of `choiceFor`, and `formFor` |
| `tui-screen` #1 | `packBoards`, the board-else-2.3-else-skip order, `todaysQuestions`'s single loop |
| `cross-cutting-refactor` | deleting `play.Recall` and re-pointing every borrower at a double |
| `tui-screen` #2 | the board's drop: `Mark`, `Toggle`, `Dropping`, `Apply`, the palette |
| `smaller-go-module` #2 | re-cutting the keys row inside its width budget, and its two pins |
| `smaller-go-module` #3 | the pty conformance run — it needs `-tags conformance` and a real pty, so it is work here rather than in review (`#37`, and `#40` priced it the same way) |
| `smaller-go-module` #4 | the sitting-length measurement (Done-when 5), itemized as `#40` itemized its load claim |
| `atlas-docs` | README's form section + the atlas sweep over the tree |
| `milestone-review` ×2 | deviation 1 above |
| `ux-rename-iteration` | deviation 2 above |

## Plan

Durable design: `workshop/plans/000042-retire-recall-form-plan.md`.

Single-pass: one review boundary, so plain checkboxes rather than `Mx` tags.

- [ ] One rule picks the form, and it picks AFTER the lookup — `formFor` over an already-parsed entry, with `optionsFor` split out of `choiceFor` so a triaged word is never rendered.
- [ ] `packBoards` shrinks a chunk to what the terminal can draw instead of sending it to a form that is unavailable; a word that can be neither tested nor drawn is skipped naming both causes.
- [ ] `play.Recall` deleted; the ~30 tests that borrowed it as a one-word `Question` take a double, and any that needed a SELF-RATED form take one.
- [ ] The board gains a `Dropped` mark, a `Dropping` capability (not a type switch), a third palette colour, and a `dropped:` line as it closes.
- [ ] `Board.Grade`'s doc stops claiming `d` never arrives.
- [ ] README + atlas swept over the TREE; the three passages that are history are kept as history.

## Log

### 2026-09-01

Filed from `#40`'s first operator sitting. The screenshot that prompted it showed
`comports` under form 2.1 — an "entry defines a different word" fallback, since
NOAD sends `comports` to `comport`.

**Depends on `#40`** rather than being part of it: `#40` has been through four
boundary-review rounds and still has an open Critical (a narrowing resize under a
live board). Widening a closing issue to delete a form is the scope creep the
constitution warns about, and the selection rule this changes is `#40`'s own D4.


## Revisions

### 2026-09-02 — the board gains a drop, and the issue is wider for it

**Reason:** the Spec sent untestable young words to the board without noticing
that `d` is the only drop path in the program and does not drop on a board. On a
young deck that is every word, so `#42` as filed would have removed the only way
to delete a mistaken capture from exactly the learner most likely to have made
one.

**Delta:** `## Spec` gains "The board learns to DROP"; three Done-when rows added.
Scope grew by one mode value, one transcript line and a stale doc comment.

**Alternatives put to the operator and declined:** accept the loss and record it;
add a `/forget <word>` command first as a separate issue (decoupling deck
management from the review forms); revive the study card declined when this issue
was filed. The operator chose to keep it inside the board.

**Correction, same day:** the question was put on a false premise — that `d` in a
sitting was the only drop path. `define --forget <word>` already exists, so the
loss is a convenience inside the sitting rather than a capability. The operator's
choice stands unless they revisit it; it is recorded this way so the choice is not
later read as having been made on facts it was not.
