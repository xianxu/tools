---
id: 000038
status: working
deps: []
github_issue:
created: 2026-08-30
updated: 2026-08-30
estimate_hours:
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

## Plan

- [ ] Decide Option A vs B, and the two questions above.
- [ ] Design if it is Option A — it is more than wiring, and it touches `#32`.

## Log

### 2026-08-30

Filed from the operator's screenshot immediately after `#30` closed. Measured
while filing: `Prompt()` returns the bare word, so the region is trivial; the
whole cost is that `play_loop.go` writes through `crlfWriter` to a scrolling
terminal and therefore owns no coordinates. `#30` D5a predicted this seam and
`#32` is filed against the same divergence.
