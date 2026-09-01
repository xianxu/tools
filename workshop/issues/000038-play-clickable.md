---
id: 000038
status: working
deps: []
github_issue:
created: 2026-08-30
updated: 2026-08-30
estimate_hours: 2.18
started: 2026-08-30T16:08:11-07:00
---

# the review loop's words are not clickable, because --play draws its own frames

## Problem

Operator, 2026-08-30, with a screenshot of a `--play` sitting:

> in the --play mode, link the words so user can click on them for
> pronunciation. in the screenshot: link sycophantic, obsequious, constabulary.

`#30` made a rendered entry's tokens clickable in the interactive loop. The
review loop shows a word and asks you about it — and that word is the most
obviously clickable thing in the program, because hearing it is the whole point
of the exercise. It is inert.

**The target is easy; the coordinates are not.** A recall form's prompt IS the
word (`play/recall.go:29`, `Prompt() string { return r.word }`), so there is no
region-finding problem — nothing to search for, no offsets to survive a wrap.

What is missing is the thing `#30` spent a milestone building: **`--play` draws
its own frames.** It writes through `crlfWriter` straight to a terminal that
scrolls underneath it (`play_loop.go:64-65`), so it does not know which row
anything is on. A click reports an absolute row; without an owned screen there
is nothing to resolve it against. That is exactly `#30`'s D1, which chose the
alternate screen so the mapping is exact by construction rather than tracked.

`#30`'s D5a already named this seam and deferred it:

> `#32` keeps its `--play` half, which continues to draw its own frames through
> `crlfWriter` and is untouched by this issue.

So this issue is where that debt comes due. It is ALSO `#30`'s own promise under
test — "the affordance is ONE mechanism with a registry of regions, so a third
consumer is a row rather than a new feature". A third consumer IS a row; a third
FRAME-DRAWER is not. The honest reading is that `--play` has to join the screen,
and then the click is a row.

## Spec

Not designed. The question is how much of `#30`'s machinery `--play` adopts.

**Option A — `--play` gets the screen.** It enters the alternate buffer, writes
through `liveScreen`, and gets scrolling, resize, the exit transcript and the
click map for free. Then a clickable word is `screen.addRegions` plus one entry
in the action registry, which is the "third consumer is a row" outcome.

The cost is that `--play`'s visual behaviour changes wholesale: it currently
leaves its session in the terminal's scrollback as it goes, and the alt screen
would replace that with a transcript printed at the end. That is the same
trade `#30` D3 made deliberately for the interactive loop, and it deserves the
same deliberate decision here rather than being inherited by accident.

It also collapses the `crlfWriter` divergence D5a flagged: one line-ending owner
instead of two, which is what `#32` is filed about.

**Option B — mouse reporting without a screen.** Enable `1000`+`1006`, and on a
click compare the reported row against a remembered cursor position. Cheaper, and
wrong for the reason `#30` D1 gives: the terminal scrolls when the loop writes,
so the app is tracking an offset it never observes. `#30` rejected exactly this
and the rejection is measured, not theoretical.

**Whatever is chosen, the actions are already written.** `RegionHeadword` plays a
word through `replayInPlace`, which is the same path a bare Enter and `/pron`
take — a click here should reach it rather than growing a third caller.

### Two questions a design must settle

1. **Does the revealed DEFINITION become clickable too?** After an `n`, the
   entry is on screen and it is the same `Render` output the interactive loop
   marks — so its headword and `ORIGIN` language would light up for free. That is
   either an obvious win or a distraction during a test of recall. Decide it.
2. **Does clicking count as an answer?** It must not: hearing the word is what
   `y`/`n` are answering ABOUT, and a click that recorded a review would corrupt
   the schedule. The click is a replay and nothing else.

## Done when

- [ ] The word `--play` is asking about can be clicked to hear it.
- [ ] A click is a REPLAY and never an answer: no review is recorded, no schedule
      moves, the sitting does not advance.
- [ ] The click reaches `replayInPlace`, the same path `y`/`n`/Enter reach, so a
      third caller does not appear.
- [ ] A terminal that reports no mouse behaves exactly as `--play` does today,
      and tracking is handed back on exit — the same guarantees `rawSession`
      already carries.
- [ ] Whatever happens to `--play`'s scrollback is a DECISION with its reason
      recorded, not a side effect of adopting the screen.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.55 impl=0.08
item: smaller-go-module        design=0.02 impl=0.10
item: cross-cutting-refactor   design=0.05 impl=0.12
item: smaller-go-module        design=0.02 impl=0.08
item: cross-cutting-refactor   design=0.06 impl=0.20
item: smaller-go-module        design=0.01 impl=0.08
item: atlas-docs               design=0.02 impl=0.06
item: milestone-review         design=0.00 impl=0.30
item: milestone-review         design=0.00 impl=0.32
design-buffer: 0.15
total: 2.18
```

| item | task | why this primitive |
|---|---|---|
| `issue-spec` 0.55/0.08 | the design carrier | the issue, the operator's two decisions, FOUR plan-quality rounds — plus two design passes since: the revision recording what `#41` landed, and the re-read against the tree it left. Up from 0.45 for those two |
| `smaller-go-module` 0.02/0.10 | T0 `options.playsAudio()` | one predicate, applied inside `playAnnounced`, replacing four hand-copies. The seam exists |
| `cross-cutting-refactor` 0.05/0.12 | T1 lift the click registry | the one task that touches a working loop: `runEditor`'s `clicked` closure becomes a call to `playRegion` |
| `smaller-go-module` 0.02/0.08 | T4 the prompt word is a region | line 0, column 0 of the write `show()` already makes. Design up 0.01 because `draw` is gone and the site had to be re-found |
| `cross-cutting-refactor` 0.06/0.20 | T5 the revealed definition carries its regions | **the row the re-read repriced.** Was 0.02/0.10 for "call `writeRendered` and the regions ride along"; a pinned screen's `WriteRegions` now WRAPS, so a region computed before the wrap underlines one word and answers for another. Cross-cutting because the fix reaches `todaysQuestions`, the forms' ownership of their own reveal text, and the write path |
| `smaller-go-module` 0.01/0.08 | T7 a click never answers | one guard and one assertion |
| `atlas-docs` 0.02/0.06 | T8 | the README's review-loop section and the atlas's clickable-regions section |
| `milestone-review` 0.00/0.30 | the boundary: run | |
| `milestone-review` 0.00/0.32 | the boundary: remediation | |

Derivation notes.

**THE TOTAL IS UNCHANGED AT 2.18 AND EVERY ROW MOVED, which is the one thing a
reader of this calibration row has to know.** The plan-quality gate at
`change-code` measured the old block as *"provably stale against the tree it will
be measured on"* and warned that the errors offset: *"landing near 2.18 for a
different set of reasons produces a calibration row nobody can read later."* They
do offset, almost exactly. The mapping table above is what makes the row readable;
without it the arithmetic looks like nothing happened.

- **Three priced rows are DEAD, ~0.54h of the old 2.18.** `#41` landed T2 (the
  playback dance), T3 (the screen) and T6 (the viewport) while this issue was
  parked, and verified two of this plan's Done-when rows on real hardware under
  the exact test names it predicted. Removed rather than re-labelled.

- **T5 absorbs most of what those three gave back**, and it is the honest reason
  the total held. It was priced when T5 meant "call `writeRendered`". `#41` BR-25
  wrote the constraint down for whoever arrived first: a region's column is
  relative to the text it was computed from, so a wrap that moves a word moves
  what a click there means. The failure is SILENT and appears only on entries
  long enough to wrap, which is most of them. Reclassified `cross-cutting-refactor`
  because the fix reaches three places, not one.

- **The T3 premium is gone with T3.** The old note said *"if any row here is
  optimistic it is T3, where `--play`'s five return paths meet `onceHandBack`"* —
  that risk was `#41`'s and it materialised there, not here.

- **The two `milestone-review` rows go 0.16/0.20 → 0.30/0.32**, matching `#41`'s.
  The old pair argued from `#30`'s round counts and then priced at half of what
  the adjacent issue priced off the same evidence. `#41` then ran FIVE close
  rounds against these same files and measured 4.77 against 3.31, with roughly
  half the total being boundary work. One boundary is still one boundary: removing
  T2/T3/T6 does not shrink these.

- **`issue-spec` design 0.45 → 0.55.** v3.1 leaves design hours unscaled, and
  there have been two further design passes since the block was written.

**THE PREDICTION, restated so the close can tell a miss from a confirmation.**
The local ledger in this exact area is now four rows wide: `#30` 3.4×, `#7`
1.70×, `#39` 0.54×, `#41` 1.44×. A spread, not a bias, and every overrun in it
was boundary rounds rather than building. This issue's remaining scope is small
and its design has been through six passes, so the building should be close to
priced; **if it misses, it misses at the boundary, and T5 is the row most likely
to put it there** — a silent offset bug is exactly what a review round finds and
sends back. On that reading the actual lands near **2.5–3.5h**. The estimate is
NOT padded toward it: v3.1 is applied as written, or the calibration row means
nothing.

## Plan

- [x] Decide Option A vs B, and the two questions above — the operator chose the
      screen, and chose to mark a revealed definition too.
- [x] Design: `workshop/plans/000038-play-clickable-plan.md` (single-pass, one
      boundary; four plan-quality rounds).
- [ ] T0 — one audio-off predicate, applied inside `playAnnounced`.
- [ ] T1 — lift the click registry into `playRegion`, shared by both loops.
- [x] T2 — delete the playback dance, and re-home the outcome-ORDER pin it strands. **Landed by `#41` T3/T4**; the replacement pin is `TestAMissRecordsBeforeItPlays`, built to this plan's round-4 specification.
- [x] T3 — `--play` writes into a `liveScreen`. **Landed by `#41` T3** as `newConsole(…, newPinnedScreen)`.
- [ ] T4 — the prompt word is a region.
- [ ] T5 — the revealed definition carries its regions.
- [x] T6 — the viewport: scroll, wheel and resize. **Landed by `#41` T7/T8** as the shared `viewportGesture` plus the loop's resize case.
- [ ] T7 — a click acts and never answers.
- [ ] T8 — docs: the README's review-loop section and the atlas's.

## Log

### 2026-08-30

Filed from the operator's screenshot immediately after `#30` closed. Measured
while filing: `Prompt()` returns the bare word, so the region is trivial; the
whole cost is that `play_loop.go` writes through `crlfWriter` to a scrolling
terminal and therefore owns no coordinates. `#30` D5a predicted this seam and
`#32` is filed against the same divergence.
### 2026-08-30 — read this before resuming: #7 lands first and changes the prompt

Parked at `punt` after T0. While it is parked, `#7` (review form 2.3, meaning
multiple choice) is being built, and it touches this issue in two places — found
by `#7`'s plan-quality gate rather than at a merge conflict.

1. **`todaysQuestions` (`play_loop.go:233-265`) is edited by both.** `#7`'s T5
   builds the distractor pool there; this issue's T5 rewrites the same function.
   `#7` lands first, so re-read that function before resuming — do not apply T5
   from the plan as written.

2. **A second form arrives whose prompt is NOT just the word, and this issue's
   region arithmetic assumes it is.** `play/recall.go:29` makes the prompt the
   headword alone, so T4 draws the region at line 0, column 0, width
   `visibleCells(word)`. `Choice`'s prompt is multi-line: the word, a blank, then
   four numbered options.

   `#7` accepted a CONSTRAINT to keep this working — the target word stays alone
   on the first line — so the arithmetic still holds and T4 needs no change. It
   holds because `#7` chose to protect it, not because it is inherent, so if this
   issue ever generalises the region beyond line 0 it should stop depending on
   the constraint and read the form's own declaration instead.

   The upside: `Choice`'s options are additional clickable material this issue
   can mark later, with no new decision needed.
