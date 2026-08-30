---
id: 000038
status: working
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
item: issue-spec               design=0.45 impl=0.08
item: cross-cutting-refactor   design=0.05 impl=0.12
item: cross-cutting-refactor   design=0.05 impl=0.12
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.03 impl=0.16
item: smaller-go-module        design=0.01 impl=0.08
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.02 impl=0.14
item: smaller-go-module        design=0.01 impl=0.08
item: atlas-docs               design=0.02 impl=0.06
item: ux-rename-iteration      design=0.30 impl=0.05
item: milestone-review         design=0.00 impl=0.16
item: milestone-review         design=0.00 impl=0.30
design-buffer: 0.15
total: 2.83
```

Derivation notes.

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
- [ ] T2 — delete the playback dance, and re-home the outcome-ORDER pin it strands.
- [ ] T3 — `--play` writes into a `liveScreen`.
- [ ] T4 — the prompt word is a region.
- [ ] T5 — the revealed definition carries its regions.
- [ ] T6 — the viewport: scroll, wheel and resize.
- [ ] T7 — a click acts and never answers.
- [ ] T8 — docs: the README's review-loop section and the atlas's.

## Log

### 2026-08-30

Filed from the operator's screenshot immediately after `#30` closed. Measured
while filing: `Prompt()` returns the bare word, so the region is trivial; the
whole cost is that `play_loop.go` writes through `crlfWriter` to a scrolling
terminal and therefore owns no coordinates. `#30` D5a predicted this seam and
`#32` is filed against the same divergence.
