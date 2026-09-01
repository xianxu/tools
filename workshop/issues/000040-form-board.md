---
id: 000040
status: working
deps: ["tools#39", "tools#41"]
github_issue:
created: 2026-08-31
updated: 2026-09-01
estimate_hours: 4.46
started: 2026-09-01T08:54:58-07:00
---

# form 2.5: the board — grid triage for mature words

## Problem

The ladder's cost is dominated by MATURE words. Daily load is
`Σ 1/IntervalDays(box)` over the deck, and with `#39`'s unbounded ladder most of
a grown deck sits in the long tail — hundreds of words each costing a few reviews
a year, which adds up to most of the day's work.

One-at-a-time multiple choice costs the same per word whatever its box. So a deck
large enough to be worth having becomes a deck too expensive to maintain, and the
learner's only lever is to stop adding words — which is the wrong lever.

Separately, `#39`'s ladder distinguishes a confident answer from a merely correct
one, and no form reports confidence for a word it did not test.

## Spec

**Form 2.5 is a grid: sixteen words at once, one keystroke each.** The learner
marks each word `firm` / `unsure` / `no idea`. It is triage, not practice, and it
exists to make a large deck affordable: a hundred mature words swept in a grid
cost what ten fragile ones cost in multiple choice.

**The SCHEDULER picks the form, not the learner's mood.**

| box | state | form |
|---|---|---|
| 0–3 | fragile, being acquired | 2.3 `/meaning` — real retrieval |
| 4–7 | consolidating | 2.3 |
| 8+ | mature | 2.5 the board |

This is the load argument made concrete, and it is also a correctness argument. A
grid is SELF-REPORT WITHOUT RETRIEVAL, and the illusion of knowing runs exactly
that direction — a familiar-looking word feels known. If the learner chooses the
form, they will choose the cheap one, and the whole ladder ends up driven by
overconfidence. Binding the form to the box means the cheap form is only used
where being wrong is cheap: a box-10 word marked `firm` in error costs one missed
retrieval on a word already recalled ten times.

`/board` stays available manually as an explicit "I am short on time today"
escape. It must not be the default path.

**The mapping to `#39`'s transitions is deliberately CONSERVATIVE:**

| mark | effect |
|---|---|
| `firm` | `box + 1` — the same as a correct answer, NOT `+2` |
| `unsure` | box unchanged, re-asked sooner, and next time through form 2.3 |
| `no idea` | `box / 2`, exactly as a wrong answer |

**`firm` earns `+1` and not `+2`, and the reason belongs in the code.** `#39`
reserves `+2` for a confident answer, and self-report is not that. The real
producer of confidence is already available and objective: `Apply` knows
`s.Revealed` at grading time, so *a correct answer in form 2.3 with no reveal
first* means the learner knew it cold. That is measured rather than claimed, and
it is where `+2` should come from. This issue should carry a Revision to `#39`
saying so.

**`unsure` promotes a word back to a real test.** It is the signal that triage
was the wrong instrument for that word, so the answer is to test it properly, not
to guess at a box change.

**Selection is a cursor over a grid**, which is why this depends on `#41`: a grid
cannot be drawn by appending lines.

## Done when

- [ ] A sitting containing eligible words presents them as a grid, sixteen at a time.
- [ ] Every word in the grid can be marked — by click OR by its printed key, so a mouse-less terminal is not stuck — and the marks reach the event log with the same "recorded as it happens" guarantee a single answer has.
- [ ] The scheduler chooses 2.5 at box ≥ 3 and 2.3 below, pinned by a test over a deck spanning both, asserting BOTH sides.
- [ ] Enter takes every unmarked word as `No`; Ctrl-C cancels, leaving marked words recorded and unmarked ones with no event at all.
- [ ] Every review event names the form that asked it, so the two deferred remedies can later be chosen from the log rather than from argument.
- [ ] The `Question` interface is unchanged, or the change is form-agnostic — the session still learns nothing about which form is asking (`#6`'s Done-when, `TestSessionIsFormAgnostic`).
- [ ] Measured: a sitting of N words through the board takes materially fewer keystrokes than the same N through form 2.3.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source`, so the per-primitive hours are provisional.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.50 impl=0.08
item: cross-cutting-refactor   design=0.05 impl=0.24
item: greenfield-go-module     design=0.06 impl=0.28
item: smaller-go-module        design=0.01 impl=0.12
item: cross-cutting-refactor   design=0.04 impl=0.16
item: smaller-go-module        design=0.03 impl=0.12
item: smaller-go-module        design=0.03 impl=0.14
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.01 impl=0.08
item: smaller-go-module        design=0.01 impl=0.08
item: smaller-go-module        design=0.02 impl=0.10
item: atlas-docs               design=0.05 impl=0.08
item: smaller-go-module        design=0.02 impl=0.16
item: ux-rename-iteration      design=0.55 impl=0.10
item: milestone-review         design=0.00 impl=0.40
item: milestone-review         design=0.00 impl=0.45
design-buffer: 0.15
total: 4.46
```

| item | task | why this primitive |
|---|---|---|
| `issue-spec` 0.50/0.08 | the design carrier | the issue, the operator's sketch and FOUR follow-ups that each deleted a decision, two plan-quality rounds, and a rendered mock |
| `cross-cutting-refactor` 0.05/0.24 | T1 `Batch` and the four `Apply` points | it edits the state machine every form runs through — advance, the miss-on-hidden branch, drop, and Enter |
| `greenfield-go-module` 0.06/0.28 | T2 `Board` and `Mark` | a new form: grid layout, labels, two marks, a mode, and four interfaces to satisfy |
| `smaller-go-module` 0.01/0.12 | T3 Tab, and Enter split from space | two `Input` kinds, and `Apply` must treat `InputFinish` as `InputReveal` for every non-batch form so 2.1 and 2.3 do not notice |
| `cross-cutting-refactor` 0.04/0.16 | T4 `display.FooterRowAt` | widens the seam BOTH loops take, and the editor has to answer it too |
| `smaller-go-module` 0.03/0.12 | T5 the click asks the form first | `#38`'s row must stay green untouched, which is the constraint rather than the code |
| `smaller-go-module` 0.03/0.14 | T6 the footer carries the board | plus `fitFooter`'s floor, so grid rows are never dropped |
| `smaller-go-module` 0.02/0.12 | T7 `ReviewEvent.Form` | a store field, the capture site, and `Fold` ignoring it |
| `smaller-go-module` 0.02/0.12 | T8 `boardsFor` | partition and pack |
| `smaller-go-module` 0.01/0.08 | T9 the relearn line | one buffer write as the board closes. At the primitive's FLOOR — 0.06 was under it |
| `smaller-go-module` 0.01/0.08 | T10 the bar counts words | at the floor, as above |
| `smaller-go-module` 0.02/0.10 | T11 the load claim, measured | |
| `atlas-docs` 0.05/0.08 | T12 | README, atlas, `--help`, and the two in-tree forward references D7 falsifies |
| `ux-rename-iteration` 0.55/0.10 | the TUI iteration ROUNDS, plural | a visual form gets looked at and adjusted. `baseline-v2.1.md` says *"plan for 3–5 rounds per TUI-heavy milestone, not 1"*, and this is the most TUI-heavy thing in the project: a grid, a toggle, a panel, a colour scheme and click targets. `#38` bought ONE round at the floor and still missed 1.50×; `#41`'s operator found a clipped option line on the first real sitting. Priced mid-range rather than at the bottom |
| `smaller-go-module` 0.02/0.16 | **T13 pty conformance** | Done-when 14 had a row and no task — and `#37` records that pty rows need `-tags conformance` and a real pty, which the review environment does not have, so they must be run here |
| `milestone-review` 0.00/0.40 | the boundary: run + manual verification + the fourteen-row mutation sweep | |
| `milestone-review` 0.00/0.45 | the boundary: remediation | |

**TWO DELIBERATE DEVIATIONS FROM v3.1, both named, because the sentence claiming
fidelity is what makes an unnamed one dishonest.**

1. **The two `milestone-review` rows are priced 0.40 and 0.45** against v3.1's
   scaled ceiling of 0.20. An evidence-based override: `#41` ran five close rounds
   and `#38` four, against this same machinery, in the last two days — and
   crucially **both priced this pair at 0.30/0.32 and both still missed by half**.
   Pricing at the neighbours' *failed* number would be the same optimism twice.
2. **T1 is `impl=0.24`** against `cross-cutting-refactor`'s scaled ceiling of
   0.20. House convention — `#38` and `#41` both used it — and it edits four
   paths in the state machine every form runs through.

**AND THE ASYMMETRY THE ESTIMATE-QUALITY GATE CAUGHT, corrected rather than
argued with.** The first derivation overrode the table UPWARD on the boundary
rows using neighbour evidence, then sat at the FLOOR on `ux-rename-iteration` —
the row most about that same evidence. `baseline-v2.1.md` says *"plan for 3–5
rounds per TUI-heavy milestone, not 1"*, and this is the most TUI-heavy thing in
the project. Applying the model in both directions moved the total 3.60 → 4.46
**without padding a single row toward the prediction**: three floors raised
(`T9`, `T10`, `atlas-docs` design), one missing task priced (T13's pty rows,
which had a Done-when row and no item), and the TUI row moved off the bottom of
its range.

**THE PREDICTION, on the record so the close can tell a miss from a
confirmation.** The ledger here is five rows wide: `#30` 3.4×, `#7` 1.70×, `#39`
0.54×, `#41` 1.44×, `#38` 1.50×. The last two are the nearest neighbours — same
files, same boundary machinery, days old — and both overran by half, entirely at
the boundary rather than in the building.

This issue is **larger and more structural than either**: it widens the `display`
seam, edits four paths in the state machine every form runs through, reverses an
invariant `#38` shipped and pinned yesterday, and adds a field to the store's
vocabulary. Two plan-quality rounds already found three Criticals in the design,
which is a leading indicator rather than a comfort — those were the ones caught
before code.

So the honest prediction is **5–6h**, and the estimate is NOT padded toward it —
the two deviations above are the only places the table is overridden, and both
are argued from measured neighbours rather than from wanting a bigger number. If it misses, it misses at the boundary, and
**T4 and T5 are the rows most likely to put it there** — one widens a seam two
loops share, and the other changes what a gesture means without being allowed to
change what it means anywhere else.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-31

Filed from a design conversation. The operator proposed the 4x4 board as "a
faster way to glimpse through today's recall work"; the analysis that turned it
into a scheduler decision rather than a user preference is in the Spec — the
grid is the maintenance form, and binding it to the box is what keeps the
illusion of knowing from driving the ladder.

`/synonym` and `/acronym` were discussed and are DEFERRED to their own issues.
`/synonym` has a real data source — `com.apple.dictionary.OAWT` is installed and
active, and `dictselect.go` already names it while deliberately filtering
thesauruses out of the curated general-dictionary list — so that issue starts
with a measurement of OAWT's output shape, not a design.

### 2026-09-01 — T2: `Board` and `Mark`

Landed with sixteen mutations executed against the new rows; all sixteen were
caught, including the two T1 rows in `Apply` that a board is the first form to
exercise for real.

**Three open questions from the resume, settled by the operator.** Box ≥ 3
STAYS, to be replaced by evidence from `ReviewEvent.Form` (T7) rather than by
argument — the plan is unchanged. `#40` is finished before `#10` is claimed.
`~/play41` is removed; the manual verification step builds a fresh deck.

**`Mark` takes no verdict — `Mark(i)`, not the plan's `Mark(i, v)`.** THE MODE
DECIDES, and the mode is the board's own state: it is drawn in the footer's
toggle and flipped by Tab. A caller passing a verdict in would be a second owner
of a fact this form already renders, and the two would disagree the first time a
frame was drawn between the toggle and the click. This is the session's own
lesson applied one level down — when two things must agree about a measurement,
one owns it and the other asks.

**A CELL IS MARKED ONCE, which the plan did not say and the log requires.** Every
mark emits its `OutcomeRecord` as it lands — that is what makes Ctrl-C lossless
(D3) — and the price of writing immediately is that nothing can be taken back.
`Fold` would read a re-marked cell as two reviews of one word on one day. So a
second mark is refused, and the refusal is made VISIBLE rather than silent: the
mark stands where the key was (`[y]` in place of `[3]`), which says both "this is
answered" and "this key no longer does anything". A key that stops working
without saying so is the kind of thing a learner blames themselves for.

The same reasoning made the gutter a non-target in `CellAt`. A forgiving hit box
is the usual kindness and is wrong here: a mark cannot be taken back, so a click
that is not clearly on a word does nothing rather than marking its neighbour.

**`Prompt()` renders the grid and T6 puts it in the FOOTER.** T2's task line and
D10 read as contradictory — one says `Prompt()` renders the labelled grid, the
other says the grid cannot be a buffer line. Both are true: the form renders it,
and the loop calls it per frame for the live edge instead of writing it into the
buffer once. Nothing in `play` changes for that; it is entirely T6's wiring.

**The board owns its geometry.** `NewBoard(words, width)` takes the terminal
width — the same seam `Choice` sits on, where prose and terminals belong to the
caller and a form takes finished dimensions — and answers `Rows()` and
`CellAt(row, col)`. The alternative, exporting the column count and cell width so
the loop could do the arithmetic, is two owners of one measurement whose failure
mode is a click that marks the word next to the one under the pointer. `Rows()`
is also what D15's fit test needs: the board is the only thing that knows how
tall it is.

Width buys the guarantee D15 actually rests on: **no line `Prompt()` produces is
wider than the width it was built for**, pinned across four boards and nine
widths. Below `labelWidth + 1` a single cell cannot be drawn at all, which is far
under the terminal size a board is ever offered at.

**ASCII marks, not `✓`/`✗`.** Those runes are East Asian Ambiguous, so some
terminals give them two columns — and a cell one column wider than the board
believes is exactly the failure D15 is written against, a click landing on the
wrong word. Colour is what will make the marks pop, and it is T6's to add:
keeping the grid in the live edge is what bought colour in the first place.

**Found, for T6: `sessionKeys` LIES on a board.** `gradePrompt` appends
*"d = remove from deck, Ctrl-C to stop"* to every form's `Keys()`, on the grounds
that it is true whatever form is asking. D12 made `d` REFUSED on a batch form, so
the learner is now told to press a key that does nothing — which is the exact bug
`gradePrompt` was created to fix when the keys line was a const spelling form
2.1's y/n. `Keys()` here names Tab and Enter for the same reason: neither is true
whatever form is asking, so neither can live in `sessionKeys`.

### 2026-09-01 — T3–T6: Tab, the click seam, and the footer that carries the board

Twenty-six mutations executed across the four tasks; every one caught, two of
them only after the code was corrected (below). `#38`'s
`TestPlayClickActsAndIsNotAnAnswer` is green and UNTOUCHED, which is the proof
the click seam widened rather than branched.

**T3.** `toInput` maps Enter to `InputFinish` and Tab to a new `InputToggle`;
`Apply` asks a fourth capability, `Moded`, and no-ops for a form that has no
mode. Tab is deliberately NOT part of "any key = next word": a board is never
`Graded`, so the only forms it could advance there are 2.1 and 2.3, where it
would be an accident-prone extra way to scroll a definition away mid-read.

**T4.** `screen` records the footer it DREW and the viewport row it began at, and
`FooterRowAt` answers which footer ENTRY a row is showing — the index, not a
display-row offset, because an entry that wraps owns several rows and it is the
screen that wrapped it. A row `fitFooter` dropped answers none: inventing an
entry for a row that was never painted would mark a word that is not on screen.

**T5, and a bound that had two owners.** The loop asks the screen which footer
entry, then the form which cell — and the first draft added `row >= g.Rows()`
between them. A mutation showed it changed nothing: `CellAt` already refuses a
row past the grid, because the grid knows how tall it is. Removed. Two owners of
one bound is how the toggle row becomes a cell on the day one of them is edited.

**And a guard that is unobservable today and load-bearing tomorrow.** `Apply`
returns `OutcomeNone` for a refused mark rather than `advance(Skipped)`. For a
board the two are identical — an unspent form does not advance — so the mutation
survived. It survives only because `Grid` and `Batch` are separate capabilities:
a grid form holding ONE word is spent by definition, and without the guard a
click on nothing would step past the question. `fakeGrid` (a Grid that is not a
Batch) is the pin, and it is the contract being tested rather than the board.

**T6.** `show()` routes a `Grid` form to the footer and writes nothing to the
buffer. `boardFooter` is grid, blank, toggle, bar — the grid FIRST, which is
load-bearing rather than aesthetic, because `formCell` reads a footer index
straight back as a grid row. `fitFooter` is unchanged, as D15 promised; the fit
is `fitsABoard`, and a test pins its chrome count against what `boardFooter`
actually draws so the two cannot drift into half a board on screen.

**`sessionKeys` no longer lies, and `gradedPrompt` now derives.** `d` is refused
on a batch form (D12), so `reservedKeys(q)` drops it there — the exact bug
`gradePrompt` was created to fix, one form later. `gradedPrompt` is built from
`sessionKeys` instead of restating it, so the pair cannot drift; `doc_sync_test`
still finds both strings in the README verbatim.

**The mode has ONE owner: the footer's toggle row.** `Board.Keys()` named it too
for a while, which was the same fact drawn twice on the surface where two rulers
mark the wrong word. `Keys()` is now mode-free and its length is pinned under
eighty columns — a prompt that wraps is a frame one row taller than the board was
offered for.

**Open, and NOT invented:** D10's footer order names a **panel** between the
toggle and the bar, and nothing in the plan says what it shows or when. It has no
Done-when row either. Grid, toggle and bar are implemented; the panel is left for
the operator rather than guessed at, because giving the board definitions would
change `NewBoard`'s signature for a feature nobody specified.

### 2026-09-01 — the panel, and a plan guard that caught the sweep failing again

The operator chose **the last-marked word's definition** for D10's unspecified
panel, and **left the keys line above the grid**. Both are recorded as `R1` in
the plan's new `## Revisions`.

**The panel made the board draw its own live edge, which is the better shape
anyway.** The first cut had the loop assembling grid + toggle + panel from
`Mode()` and a gloss; that made the board's appearance a thing two files agree
about, on the surface where disagreeing marks the wrong word. Now `Prompt()`
returns the whole edge and `boardFooter` is one line that appends the bar. Two
things fell out: `Moded` needs only `Toggle()` — nothing outside the form ever
reads the mode — and `Rows()` counts the whole edge, which is the number
`fitsABoard` was always actually about.

**`NewBoard` takes `[]Cell`** — a word and a one-line gloss — the same seam
`Choice` sits on, where the caller does the dictionary work and the form takes
finished content. `targetCandidate` (optionpool.go) already produces exactly that
gloss, so T8 has it to hand.

**An empty panel is still a row.** A row that appeared with the first mark would
shift the grid up by one and move every word under a pointer already resting on
it.

**A SECOND redundant bound, found the same way as T5's.** `CellAt` guarded
`row >= gridRows()` and the mutation could not be made to fail: a row below the
grid indexes past the last cell by construction, so `i >= len(cells)` was already
answering. Removed, on this codebase's own precedent — *"the guard that was here
shipped as dead code and would have hidden that"*. The PROPERTY (the toggle and
panel rows are not clickable) is now pinned apart from the mechanism, over eight
board shapes and every column, so a rewrite of the arithmetic has to keep it.

**`TestPlanTableStatusMatchesTheChangeWindow` caught three stale plan rows**, and
the failure is the lesson this issue already wrote down, one round later:

- `fitFooter` still said `modified` though **D15 retired that in the prose** —
  the plan's own sweep missed the plan's own table.
- `livePrompt` said `modified` and needed no change at all: a board is never
  `Graded`, so the graded prompt cannot fire. What changed is `gradePrompt`.
- Seven new symbols were shipped that no table row named.

The rule that leaves: **a hand-sweep of a plan's tables does not hold, and this
repo already knew it** — the guard exists because #29 got four of twenty rows
wrong across two review rounds. Trust the guard, and run the full suite before
believing a plan.

### 2026-09-01 — T7: `ReviewEvent.Form`, and a field that cannot be forgotten

`Form()` is on the `Question` INTERFACE rather than an optional capability, and
that is the whole design of this task. The plan's own red-when is *"the field is
written for one form and defaulted for the others, which is worse than absent"* —
so the only acceptable shape is one where a new form cannot compile without
naming itself. `Keys()` is on the interface for the same reason.

**And it is stamped in ONE place.** `Apply` now wraps the state machine and
stamps every `OutcomeRecord` with `q.Form()` on the way out. Writing
`Form: q.Form()` at the three sites that build a record — the ordinary advance,
the miss-on-a-hidden-word branch, and the Enter that spends a board — would be
three chances to ship a promotion the log cannot attribute, and the failure would
be silent: an event with an empty form looks like data. What this field exists to
watch is already silent and delayed enough.

**Names, not the project's form numbers.** `recall`, `meaning`, `board`. A log is
read years later by a script or a person, and `board` needs no atlas to decode
while `2.5` does; `meaning` is also what the learner types to reach form 2.3.

**A survivor worth the round it cost: `yaml:"-"` shipped green.** The loop's
tests read through `store.Mem`, which keeps events in memory — so the field
reached every assertion without ever reaching a file, and a field that reaches
only memory answers nothing the query needs. Pinned now by a real
`store.NewYAML` round trip, plus the `at:`-stays-last rule, which this field
tests for the first time since it sits immediately before `At`.

The rule that leaves: **an in-memory double cannot pin a claim about
persistence.** It is the same shape as #30's rule about doubles standing in for
the object that joins two halves — here the two halves are the struct and the
file, and the thing between them is the yaml tag.

## Revisions

### 2026-09-01 — the operator's sketch redesigned the interaction; the Spec above predates it

The `## Spec` describes a keystroke-per-word maturity triage with three marks,
chosen by the box at ≥ 8. The operator sketched something different and answered
four follow-ups; the design now lives in
`workshop/plans/000040-form-board-plan.md`. The Spec is kept as filed — it is the
record of what was asked for — and the deltas are here.

**What changed:**

- **Clicks and labelled keys, not a cursor.** Each cell prints its key
  (`0`–`9` then `a b c e f g` — `d` is reserved by `toInput` as *drop from deck*
  before any form sees it). Clicking is `#38`'s affordance finding its second
  consumer.
- **Two marks, not three.** *"I guess unsure means no."* That deletes the entire
  `EventUnsure` mechanism the first plan drafted — a new event kind, a `Fold`
  exemption, and a form-selection rule reading it.
- **Box ≥ 3, not ≥ 8.** *"Words should come from all today's practice words…
  allow form-board to be used earlier."* The risk runs opposite to the Spec's
  assumption: a wrong `Yes` buys ZERO extra days at box 0 (the ladder's first two
  rungs both wait a day) and 165 at box 10, while the chance of being wrong falls
  as the box rises — so the product peaks in the MIDDLE, not at the top.
- **Both remedies for the form's weakness — promote slower, or use it less often
  — are DEFERRED**, to be chosen from evidence and eventually made per-learner.
  Which is why **every review event now records the form that asked**: the query
  the operator named ("did the user fail beyond reasonable in a real recall test
  later") joins a promotion to a later test, and nothing in the log can answer it
  today.
- **Enter commits, Ctrl-C cancels.** Ctrl-C needs no code: clicks record as they
  happen and `InputQuit` emits no record, so unmarked words keep their boxes.
- **The grid is the LIVE EDGE, in the footer.** The buffer is append-only —
  that is what makes a click's coordinates exact — so a grid written there could
  never change colour. `display` gains one question: which footer row a click
  landed on. Chosen over rewriting the buffer because it does not start a layout
  system.
- **`/board` is a NON-GOAL, with a reason.** It existed as the "I am short on
  time" escape while the board was confined to mature words. With the schedule
  offering it from box 3, there is much less to escape to, and a manual override
  reintroduces exactly the learner-picks-the-cheap-form problem the Spec argued
  against. Filed separately if it is still wanted after real use.

**The rule this revision leaves**, because the plan-quality gate had to find it
twice: **a revision sweeps every artifact that restates the decision, and the
issue's own Spec and Done-when are artifacts.** The plan's prose, its tables and
its Done-when rows are three surfaces; the issue file is a fourth, and stopping
at the plan is how a Done-when row survives naming a threshold nothing uses.
