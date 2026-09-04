---
id: 000042
status: done
deps: [tools#40, tools#44]
github_issue:
created: 2026-09-01
updated: 2026-09-04
estimate_hours: 3.89
started: 2026-09-02T12:31:27-07:00
actual_hours: 10.85
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

- [x] A word with no usable distractors reaches a BOARD, at every box, pinned by a test over a deck that spans the threshold and asserts both sides. — `TestUntestableWordsReachABoardAtEveryBox`: `bases` (every sense a cross-reference) at box 0 reaches a board, a box-5 word still does, and a young TESTABLE word still gets 2.3 — the third assertion is what stops it passing on a build that boarded everything.
- [x] `play.Recall` does not exist, and no test substitutes a shipped form where it wants a double. — deleted with its test file. Three doubles replace it: `fakeQuestion` (loop mechanics), `askableRig` (tests needing a real rendered entry), and `play`'s own `fakeForm`. The one test that genuinely needed a SELF-RATED form takes the board, now its only implementor.
- [x] The declared fallback reasons still derive into the README, and still name three — the reasons did not change, only what they select. — `TestREADMENamesEveryFallbackReason` green; `fallbackReasons` is unchanged and the README carries all three under the board's section.
- [x] A word that can be neither tested nor drawn is skipped with a message that names the cause, and the sitting continues. — `TestAWordNeitherTestableNorDrawableIsSkippedWithItsCause` asserts BOTH causes are named (both must hold) and that the other words are still asked.
- [x] Measured: the sitting a young deck gets is no longer than it was — the board packs, so this must not add screens. — `TestRetiringRecallDoesNotLengthenAYoungSitting`, decks of 1–5, `len(qs) <= len(deck)` against the NAMED baseline (form 2.1 cost one screen per word), plus every due word still reviewed.
- [x] A word can be dropped from a board, by key and by click, and the drop reaches `store.Forget` — pinned end to end rather than at the form. — `TestDroppingAWordOnABoardRemovesItFromTheDeck`, two subtests, asserting the word left the deck, its neighbours did NOT, and no review event was written.
- [x] The live mode is unambiguous on the prompt row in all three states, and a board closes naming what it dropped. — `TestEveryModeSpellingIsTheSameWidthAndFitsEighty`. **Second half revised:** no new line. The loop already reports each removal as it happens (PQ-4), so the planned `dropped:` sibling to `relearnLine` would have been a second owner; `TestADropOnABoardIsReportedOnce` pins exactly one, driving a Tab and a refused click after the drop.
- [x] `Board.Grade`'s doc no longer claims `d` never arrives. — corrected, and so is `Keys()`'s claim that the label set "has a hole at `d`". Both were falsified by `#40`'s own last round.

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

- [x] One rule picks the form, and it picks AFTER the lookup — `formFor` over an already-parsed entry, with `optionsFor` split out of `choiceFor` so a triaged word is never rendered.
- [x] `packBoards` shrinks a chunk to what the terminal can draw instead of sending it to a form that is unavailable; a word that can be neither tested nor drawn is skipped naming both causes.
- [x] `play.Recall` deleted; the ~30 tests that borrowed it as a one-word `Question` take a double, and any that needed a SELF-RATED form take one.
- [x] The board gains a `Dropped` mark, a `Dropping` capability (not a type switch), a third palette colour, and a `dropped:` line as it closes.
- [x] `Board.Grade`'s doc stops claiming `d` never arrives.
- [x] README + atlas swept over the TREE; the three passages that are history are kept as history.

## Log


- 2026-09-03: closed — go test ./... green; go vet + gofmt clean; go test -tags conformance ./cmd/define green (136s). Rounds 1-2 blockers all fixed at the RULE and mutation-verified where mutable: BR-1 the palette check DERIVES from play.Marks() with Palette.For the one owner (both arms reverted, both redden named tests); BR-2 the empty-sitting summary names the cause the code established, not the only cause that used to exist (reverted, reddens); BR-3 five stale routing claims swept plus retiredPhrases rows keyed on the retired CLAIM not the form name; BR-7 the guard that should have caught boardsFor existed and was blind — isCitableName widened to unexported names with an interior capital, red on all six sites immediately; BR-8 five extent restatements now defer to Marks() rather than restating a count; BR-9 both new exported symbols documented in atlas + plan with a Revisions note that they arrived at the boundary. One finding was a CONFLICT between two guards (plan-table requires naming what was deleted, retired-symbol forbids it), settled in currentTruthOnly with an exemption self-limiting to the symbol a deleted-row names. Deliberate non-fix recorded: no retiredSymbolNames row for Recall, since recall is ordinary English this program uses for up-arrow history. BR-4 and BR-5 remain open Minors, declined with reasons in the issue Log. Round 3 was an ABORTED run — a 401 revoked the OAuth token mid-review, so it recorded no findings and no verdict; the ledger names the cause and the round was committed rather than erased. NOT VERIFIED BY ME: the operator sitting the plan Verification asks for — the binary is rebuilt and current but I cannot press keys, so drop-mode legibility on a live prompt row is unconfirmed.; review verdict: FIX-THEN-SHIP
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

### 2026-09-02 — implementation: what the deletion actually cost

**The selection rewrite was the small half.** `formFor`'s rule fits in a
paragraph; what made it a rewrite is that it had to happen AFTER the lookup, so
`todaysQuestions`' two loops (render-then-ask, then boards) became one that
carries parsed entries forward so a word changing hands is not fetched twice.

**Deleting the form broke about fifteen tests, and every one was a test that had
borrowed a shipped form as a double.** That is the finding worth keeping. A
one-word deck produced form 2.1 deterministically, so "give me some single-word
question" was spelled "make a one-word deck" all over the suite — and the form's
`y`/`n` leaked into scripts that were not about it.

The sharpest instance was `gradeKey`, whose doc promised *"the keystroke that
grades q with the wanted verdict, WHICHEVER form q is"* while ending in a
hardcoded `y`/`n`. It had been lying since form 2.3 shipped; deleting 2.1 turned
the lie into silence — a dozen tests stopped grading and still passed their
earlier assertions. It probes the form now.

Three doubles replaced the borrowing: `fakeQuestion` for loop mechanics,
`askableRig` for tests that need a real rendered entry (a deck big enough to build
distractors), and `play`'s existing `fakeForm` inside the package. One test
genuinely needed a self-rated form — `TestSelfRatedFormsNeverEarnUnaided` — and
takes the board, which is the only implementor left.

**The drop landed as designed**, with `advance` the single place the capability is
asked because that is where the printed-key and click paths meet. Every piece is
mutation-verified: `Dropped.Verdict()`, `advance`'s call, the one-shot reset, and
`Toggle`'s third state each redden a named test when reverted.

Also corrected while here, both about the key this issue is named for:
`Board.Grade` claimed *"`d` and `D` never arrive"* and `Keys()` claimed the label
set *"has a hole at `d`"*. `#40`'s own final round falsified both.

### 2026-09-03 — boundary review round 1: one real bug, one false tick, one half-swept class

Verdict FIX-THEN-SHIP. Three blockers, all fixed at the class and
mutation-verified.

- **BR-2 — a real user-visible bug I introduced.** A word the dictionary answers
  fine, for which no test can be built and whose window cannot draw a board, made
  the sitting print *"N words are due but none could be looked up"* — sending the
  learner to check their dictionary about a window that is too narrow. This is
  `PQ-1`'s false-summary concern NARROWED rather than removed, and the surviving
  population is exactly this issue's subject: a young deck on a narrow terminal,
  which ran a full sitting before `#42`. The summary now names the cause the code
  established, pinned in both directions by
  `TestAnEmptySittingNamesWhyItIsEmpty`.
- **BR-1 — I ticked a mutation check I never ran.** The plan named three for the
  drop; I ran two. The palette was the third, and both its halves could be
  deleted with the whole suite green — a dropped cell would have painted
  identically to an untouched one while the README promised it was struck out.
  Fixed at the class rather than by adding one assertion: `play.Marks()` is now
  the extent of the mark set and `Palette.For` its one owner, so a fourth mark
  with no colour fails the day it is declared.
- **BR-3 — the tree-wide sweep was the instance, not the class.** `PQ-7` named
  four files; I swept exactly those four and five more were left claiming a word
  "falls back to form 2.1" as current behaviour. Swept — and `retiredPhrases`
  (the mechanism `#40` built for exactly this) gains rows for the retired ROUTING
  CLAIM. The form's NAME is deliberately not banned: several comments explain why
  something exists by naming the form it was built for, and the atlas keeps the
  argument that produced `Keys()`.

Minor also taken: the plan's cost table claimed a `Render` newly paid on every
mature board word. It pays none — mature words are triaged before any lookup — and
the error runs the flattering way, since the line was added to answer a gate
finding that the first draft named only savings.

Two lessons recorded in `workshop/lessons.md`: a mutation check you did not run is
worse than none, and a scripted revert must never replace an empty string (it
prepends, and it corrupted two source files this round).

### 2026-09-03 — boundary review round 2: the sweep was the instance, again

Six of round 1's findings disposed; three new, and two are the same family one
level out. That is the round's actual content: I had swept the sites each round
named and never written the enumeration the class implied.

- **BR-7 — `boardsFor` was deleted in this window and was still the current
  account of selection in six places**, including two atlas paragraphs — so the
  atlas held two contradictory accounts of the rule this issue exists to change.
  **The guard for this already existed and could not see it:**
  `TestARemovedDeclarationIsSweptOrRetired` gates on `isCitableName`, which
  required an exported or `Test*` name. Widened to unexported names with an
  interior capital — the discriminator that separates cited compounds
  (`boardsFor`, `choiceFor`) from the single lowercase words (`ids`, `paint`) the
  original clause was right to exclude. It went red on all six immediately.
- **BR-8 — five comments restated an extent this window changed**, including
  `Mark`'s own doc saying "TWO marks and an ABSENCE" three lines above the const
  declaring a third. Fixed by deferring to `Marks()` rather than by correcting the
  numbers, which is the same move `Keys()` already makes for the label set.
- **BR-9 — `Marks()` and `Palette.For` were new exported surface with no atlas
  entry and no plan row.** Both, plus a `## Revisions` note saying they were added
  at the close boundary rather than designed.

**One finding was a conflict between two guards, not a defect in the prose.**
`TestPlanTableStatusMatchesTheChangeWindow` requires a plan to name what the window
deleted; `TestNoArtifactNamesARetiredSymbol` forbids a current-truth artifact from
naming a retired symbol. The plan could not both name it and not. Settled in
`currentTruthOnly`, where "is this a record?" is already decided, so both guards
inherit one answer — and the exemption is self-limiting: only a document carrying
a `| deleted |` row gets it, and only for the symbol that row names.

Also recorded: no `retiredSymbolNames` row for `Recall`. "Recall" is ordinary
English this program uses constantly for the editor's up-arrow history, so a row
fires on `history.go`, `repl.go` and half the atlas over text with nothing to do
with the form. What is retired is the routing CLAIM, and `retiredPhrases` carries
that — a phrase, not a word.

## Restart here — state as of 2026-09-03

**Session ended mid-close. Everything is committed; nothing is in flight.**

### Where things are

| | |
|---|---|
| branch | `000042-retire-recall-form` (in-place), clean tree |
| HEAD | `666964f` — "#42: close round 2 — widen the guard that should have caught this" |
| status | `working`, `estimate_hours: 3.89`, no `actual_hours` yet |
| gates | plan-quality PASSED (2 rounds). close: **2 rounds run, not finalized** |
| verification | `go test ./...`, `go vet`, `gofmt` all clean at HEAD; `go test -tags conformance ./cmd/define` green (136s) |

### The one thing to do next

```
sdlc close --issue 42 --verified '<evidence>'
```

A round 3 was launched and **interrupted before it wrote anything** — the gate
ledger has 2 rounds recorded, so re-running is a clean round 3, not a resume.

Reuse the `--verified` text from the round-2 attempt: it is preserved verbatim in
the session transcript, and its substance is the four `### 2026-09-0x` Log entries
above. The one clause that must survive rewriting, because it is the honest limit
of what was checked:

> **NOT VERIFIED BY ME:** the operator sitting the plan's Verification asks for.
> The binary is rebuilt and current, but I cannot press keys — so drop-mode
> legibility on a live prompt row is unconfirmed.

### What the ledger will say

`workshop/plans/000042-retire-recall-form-close-gate.md` lists five open findings.
**Three of them (BR-7, BR-8, BR-9) are already fixed at HEAD** and were fixed
*after* the round that raised them, so the ledger could not dispose of them — the
same situation `workshop/lessons.md` records under "An open ledger row is a
question, not an answer". Re-measure against the tree before treating any as live.

Two are genuinely open, both Minor and both non-blocking, both deliberately
declined with reasons worth keeping:

- **BR-4** — the `Lookup` / skip-message / `ParseEntry` block appears three times
  in `todaysQuestions`. Real duplication. Declined during round 2 because the
  three copies differ in what they do on failure (one skips a due word loudly, one
  is a board cell, one is a leftover retry) and a helper taking a callback for
  that is not obviously better than three short blocks. Worth a fresh look, not
  worth blocking a close.
- **BR-5** — `Board.Dropped()` reads `b.cells[b.last]`, so its correctness rests
  on `b.last` not having moved since the arming `Mark`. No path interleaves `Rest`
  with an armed drop today. The tighter fix is a `dropWord string` field set at
  arming time; it is one line and would remove the ordering dependency entirely.

### If the close passes

`sdlc pr` → merge (interactive; the operator runs it). Then the project file
`workshop/projects/define-learn.md` needs `#42` marked done — it is NOT in
`mvp_scope`, so it is a scope-event note rather than a task tick.

### Then: the MVP is four issues from done

`#10` (authored practice items) is the one the project is actually waiting on —
its Breakdown calls it "the material-quality checkpoint … stop here and read the
output before building the forms that consume it". `#8`, `#12`, `#13` follow.

### 2026-09-03 — close round 4: the rule fixes were themselves unpinned

Round 3 was an ABORTED run — a 401 revoked the OAuth token ~14 minutes into the
review, so it recorded no findings and no verdict, and the gate refused to
finalize. Committed rather than erased: the ledger and sidecar both name the
cause, and deleting a round that was genuinely attempted would falsify the record
to keep the round count down.

Round 4 passed with FIX-THEN-SHIP and demoted three Importants past the round cap.
All three were fixed before this close commit, per `#174`.

- **BR-11 — the fix for BR-7 was green when reverted.** Widening `isCitableName`
  closed a class; nothing pinned the widening, because `boardsFor` had just been
  swept out of the tree so no artifact exercised the new clause. **When a rule's
  triggering input no longer exists, the pin is a fixture table** — the precedent
  was eight lines away in the same file. Both halves now have one, and both
  reverts redden.
- **BR-13 — the count class was closed by re-wording, and the new wording restated
  the count**, inside the sentence forbidding it, while claiming its test derived
  from the cycle when the test read `for range 3`. Fixed in the CODE rather than
  the prose: `Keys()` is a table keyed by mark with a loud default, the spelling
  test walks `Marks()`, and `TestEveryMarkHasASpelling` fails the build for a mark
  with no row. The review's own mutation is what proved it — a fourth mark made
  `Keys()` silently draw the yes row for the wrong mode.
- **BR-12 — seven production comments still said form 2.1 IS, HAS or CANNOT.**
  Round 1 declined a bare ban on the name because ~8 historical mentions are
  legitimate, and that left the class open for three rounds. **Tense is the
  discriminator**: `retiredPhrases` rows keyed on "form 2.1 is/has/cannot" catch
  the claims and spare "was"/"used to"/"before #42" by construction. They found
  three more the hand-grep had missed.

One more worth recording: the first version of the mark-spelling walk HUNG on the
mutation instead of failing, because `for b.Mode() != m { b.Toggle() }` never
terminates for a mark `Toggle` cannot reach — the exact mark the test exists to
catch. Bounded, with a `t.Fatalf` naming the cause.
