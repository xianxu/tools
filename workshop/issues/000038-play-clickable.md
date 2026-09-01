---
id: 000038
status: punt
deps: []
github_issue:
created: 2026-08-30
updated: 2026-08-30
estimate_hours: 2.83
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
item: cross-cutting-refactor   design=0.05 impl=0.12
item: cross-cutting-refactor   design=0.05 impl=0.24
item: smaller-go-module        design=0.02 impl=0.08
item: greenfield-go-module     design=0.06 impl=0.28
item: smaller-go-module        design=0.01 impl=0.08
item: atlas-docs               design=0.02 impl=0.06
item: ux-rename-iteration      design=0.30 impl=0.05
item: milestone-review         design=0.00 impl=0.30
item: milestone-review         design=0.00 impl=0.32
design-buffer: 0.15
total: 2.83
```

| item | task | why this primitive |
|---|---|---|
| `issue-spec` 0.55/0.08 | the design carrier | the issue, the operator's two decisions, FOUR plan-quality rounds — plus two design passes after the park: the revision recording what `#41` landed, and the re-read against the tree it left. Up from 0.45 for those two |
| `cross-cutting-refactor` 0.05/0.12 | **T0 — DONE** (`9168bf2`) | `options.playsAudio()` inside `playAnnounced`, replacing four hand-copies in two spellings. Kept in the block: it is real work inside the window `sdlc actual` measures, and removing it would understate the issue |
| `cross-cutting-refactor` 0.05/0.24 | T1 lift the click registry | the one remaining task that touches a working loop. Priced against `#41`'s `todaysQuestions` signature-widening (0.24), which is its shape — not its `menu`→`footer` rename (0.12), which is not |
| `smaller-go-module` 0.02/0.08 | T4 the prompt word is a region | line 0, column 0 of the write `show()` already makes. Design up 0.01 because `draw` is gone and the site had to be re-found |
| `greenfield-go-module` 0.06/0.28 | T5 the revealed definition carries its regions | **the row the re-read repriced.** Was 0.02/0.10 for "call `writeRendered` and the regions ride along"; a pinned screen's `WriteRegions` now WRAPS, so a region computed before the wrap underlines one word and answers for another — silently, on any entry long enough to wrap. `greenfield` because `cross-cutting-refactor`'s band tops out at 0.20 and this row was sitting on the ceiling while the prose called it the issue's real risk |
| `smaller-go-module` 0.01/0.08 | T7 a click never answers | one guard and one assertion |
| `atlas-docs` 0.02/0.06 | T8 | the README's review-loop section and the atlas's clickable-regions section |
| `ux-rename-iteration` 0.30/0.05 | the TUI iteration round | unchanged from the post-T0 derivation, and `#41` confirmed the cost is real: the operator found a clipped option line on the first real sitting |
| `milestone-review` 0.00/0.30 | the boundary: run + **manual verification** | the close checklist needs an unsandboxed conformance run AND a real-terminal session — click and HEAR the word, `n`, click the headword and `ORIGIN`, page back, quit, check the transcript. The audio step is verifiable no other way |
| `milestone-review` 0.00/0.32 | the boundary: remediation | |

Derivation notes.

**RE-DERIVED 2026-08-31, against the tree `#41` left — and it lands back on 2.83
for entirely different reasons, which is the one thing a reader of this
calibration row has to know.** Three priced rows died and the survivors grew by
almost exactly as much. The mapping table above is what makes that auditable;
without it the arithmetic looks like nothing happened.

- **T2, T3 and T6 are DEAD, ~0.54h.** `#41` landed the playback-dance deletion,
  the screen adoption and the viewport while this branch was parked, and its close
  verified two of this plan's Done-when rows on real hardware under the exact test
  names this plan predicted, plus row 4c's replacement pin built to this plan's
  round-4 specification. Removed rather than re-labelled.

- **T5 absorbs most of what they gave back**, and is the honest reason the total
  held. `#41` BR-25 wrote the constraint down for whoever arrived first: a
  region's column is relative to the text it was computed from, so a wrap that
  moves a word moves what a click there means. That failure is SILENT and appears
  only on entries long enough to wrap, which is most of them.

- **The T3 premium is gone with T3.** The post-T0 note said the optimistic row was
  T3, "where `--play`'s five return paths meet `onceHandBack`". That risk was
  `#41`'s, and it materialised there.

- **T1 0.12 → 0.24**, from the estimate-quality gate: it is shaped like `#41`'s
  `todaysQuestions` widening, not like a rename.

- **The two `milestone-review` rows go 0.16/0.30 → 0.30/0.32.** See the deviation
  note below — this is outside the model, deliberately.

**ONE DELIBERATE DEVIATION FROM v3.1, named rather than buried.** v2's table gives
milestone review `0.2–0.5` and v3.1 writes `impl=` at 40% of that, so the scaled
ceiling is **0.20**. These rows are 1.5–1.6× above it and are 22% of the total.
That is an evidence-based OVERRIDE of the primitive table, not the model applied
as written: `#41` ran FIVE close rounds against these same files days ago and
measured 4.77 against 3.31, with roughly half the total being boundary work.
Pricing this boundary at the model's ceiling would encode a number the adjacent
row already falsified.

**THE PREDICTION, restated.** The local ledger here is four rows wide: `#30`
3.4×, `#7` 1.70×, `#39` 0.54×, `#41` 1.44× — a spread, not a bias, and every
overrun in it was boundary rounds rather than building. The remaining scope is
small and the design has been through six passes, so **if this misses, it misses
at the boundary, and T5 is the row most likely to put it there**: a silent offset
bug is exactly what a review round finds and sends back. On that reading the
actual lands near **3–4h**. The estimate is NOT padded toward it.

--- the post-T0 derivation, kept because the rows it argued for survive ---



- **`issue-spec` design 0.45 is mostly already spent**: the issue, the operator's
  two decisions, and FOUR plan-quality rounds. Lower than `#30`'s 0.50 because
  the architecture was decided there — this issue adopts it rather than choosing
  it. The rounds were not cheap, though, and three of them found real defects:
  a Critical the first draft would have shipped, a guard that scoped wrong, and a
  replacement pin that did not discriminate.

- **TWO `cross-cutting-refactor`s, and the first draft mislabelled one.** T1
  lifts the click registry out of `runEditor`'s closure; T0 replaces the
  audio-off predicate at four sites in four files and changes `playAnnounced`
  itself. The table's definition is literally "multi-file rename / language
  pivot", and all four of T0's sites are in working loops — calling it a
  `smaller-go-module` under-reported how much of this issue touches shipped code.
  The hours barely move (the scaled bands overlap), which is the point: it was a
  labelling defect, and a labelling defect is what makes a ledger row unreadable
  later.

- **A `ux-rename-iteration` round, which the first draft omitted.** This is a TUI
  feature filed from an operator screenshot, and v2.1's own Known Limitations
  flag that case: "UX iteration round count (3–5 typical for TUI features, not
  1)". One round is budgeted, not three — the operator's request was specific and
  the design is already settled — but zero was not defensible. `#30` took two
  such rounds mid-flight (the duplicated prompt, the wheel).

- **No `greenfield-go-module` and no `TUI screen` primitive**, which is the whole
  shape of this issue: `#30` built the screen, the click map, the region
  registry, the viewport keys, `watchResize`, `onceHandBack` and `console`. This
  is a second consumer adopting them (D10). If any row here is optimistic it is
  T3, where `--play`'s five return paths meet `onceHandBack`.

- **Two `milestone-review`s for ONE boundary**, priced 0.16 to run and 0.30 to
  remediate. Not a hedge — this session's measured rate: `#30`'s M1 took six
  rounds, M2 five, its close two, and EVERY one returned at least an Important.
  Pricing remediation at zero is the single thing the record rules out.

  **The remediation row also carries the MANUAL TERMINAL PASS**, which the first
  draft dropped and `#30` priced separately at 0.10. It matters more here, not
  less: `#37` measured today that every `TestPTY*` row reports "no pty available"
  where the work runs, and FOUR of this issue's Done-when rows (5, 6b, 7, 8) rest
  on pty tests. So the only place several of these claims can be checked at all
  is an operator at a real terminal, clicking. There is no vocabulary slug for
  that, so it rides here rather than being invented.

- **THE PREDICTION, on the record so the close can tell a miss from a
  confirmation.** `#30` estimated 3.19 and measured 10.91 — **3.4×** — and the
  overrun was almost entirely boundary rounds rather than building. This issue is
  smaller and its design is already through four rounds, so the same multiplier
  should not apply; but if the boundary behaves as `#30`'s did, the actual lands
  near **4–5h** rather than 2.83.

- **The design column is already spent**, and the block should be read that way:
  0.68 × 1.15 ≈ 0.78h of it went on the issue, the operator's decisions and four
  plan-quality rounds before T0 begins. What 2.83 actually asserts is that nine
  tasks, a boundary and a manual pass land in the remaining ~2h. The chain is
  also strictly sequential — T4 through T7 have no screen to write into until T3
  lands — so there is no fan-out for parallelism to compress.
  **The estimate is NOT padded toward that.** v3.1 is applied as written, or the
  calibration row means nothing — which is exactly what `#30`'s own estimate note
  said before being wrong in the same direction.

## Plan

- [x] Decide Option A vs B, and the two questions above — the operator chose the
      screen, and chose to mark a revealed definition too.
- [x] Design: `workshop/plans/000038-play-clickable-plan.md` (single-pass, one
      boundary; four plan-quality rounds).
- [x] T0 — one audio-off predicate, applied inside `playAnnounced`.
- [ ] T1 — lift the click registry into `playRegion`, shared by both loops.
- [x] T2 — delete the playback dance, and re-home the outcome-ORDER pin it strands. **Landed by `#41` T3/T4**; the replacement pin is `TestAMissRecordsBeforeItPlays`, built to this plan's round-4 specification.
- [x] T3 — `--play` writes into a `liveScreen`. **Landed by `#41` T3** as `newConsole(…, newPinnedScreen)`.
- [ ] T4 — the prompt word is a region.
- [ ] T5 — the revealed definition carries its regions.
- [x] T6 — the viewport: scroll, wheel and resize. **Landed by `#41` T7/T8** as the shared `viewportGesture` plus the loop's resize case.
- [ ] T7 — a click acts and never answers.
- [ ] T8 — docs: the README's review-loop section and the atlas's.

## Log

### 2026-08-31 — resumed; `#41` landed three of the nine tasks

`#41` merged (PR #25) and took T2, T3 and T6 with it, so this issue resumes at
T1 with T0 already done from the 2026-08-30 session. The plan carries the
re-read; the estimate is re-derived above and lands back on 2.83 for a different
set of reasons, which the mapping table makes auditable.

**What a resumer must not lose, updated.** The 2026-08-30 note said it was the
playback dance. That is done. The live one is `#41` BR-25's: a pinned screen's
`WriteRegions` WRAPS, and a region's column is relative to the text it was
computed from — so T5 has to wrap BEFORE computing regions, or the underline and
the hit test land on different words. Silently, and only on entries long enough
to wrap.

**A process note, because it cost real work.** Resuming, I read this issue's file
on `main` and re-derived its estimate from a 2.18 block — while the branch
carried a considered 2.83 from the post-T0 session that `main` had never seen,
because the branch was unpublished. **A parked branch's issue file is the current
one; `main`'s copy is as old as the park.** Check out the branch before reading
the record.


### 2026-08-30 — PARKED after T0, deliberately

Operator moved to the review-forms work. Parked rather than abandoned: the plan
cleared four rounds of plan-quality and is the durable part, so resuming is
`sdlc claim --issue 38` and picking up at T1.

**State when parked.** T0 is DONE and merged into the branch
`000038-play-clickable` — `options.playsAudio()` is one predicate applied inside
`playAnnounced`, replacing four hand-copies in two spellings, with the row none
of the existing audio tests could be (they all go through callers that stop
first). That task stands on its own and would be worth keeping even if the rest
of this issue never happens.

T1–T8 are unstarted. The branch is unpublished, so nothing is on `main`.

**The one thing a resumer must not lose**, because it is the reason the plan took
four rounds: `--play`'s playback restores raw mode, and since `#30` folded the
alternate screen and mouse reporting into `rawSession`, `restore()` tears both
down while `enterRaw()` returns a session with neither. Adopting the screen
without deleting that dance (T2) breaks the sitting at the first reveal, silently.

### 2026-08-30

Filed from the operator's screenshot immediately after `#30` closed. Measured
while filing: `Prompt()` returns the bare word, so the region is trivial; the
whole cost is that `play_loop.go` writes through `crlfWriter` to a scrolling
terminal and therefore owns no coordinates. `#30` D5a predicted this seam and
`#32` is filed against the same divergence.
