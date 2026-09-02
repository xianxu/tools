---
id: 000040
status: codecomplete
deps: ["tools#39", "tools#41"]
github_issue:
created: 2026-08-31
updated: 2026-09-01
estimate_hours: 4.46
started: 2026-09-01T08:54:58-07:00
actual_hours: 7.89
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

- [x] A sitting containing eligible words presents them as a grid, sixteen at a time.
- [x] Every word in the grid can be marked — by click OR by its printed key, so a mouse-less terminal is not stuck — and the marks reach the event log with the same "recorded as it happens" guarantee a single answer has.
- [x] The scheduler chooses 2.5 at box ≥ 3 and 2.3 below, pinned by a test over a deck spanning both, asserting BOTH sides.
- [x] Enter takes every unmarked word as `No` — and is HELD while the window cannot show the whole board, because it would otherwise demote words that were never drawn (R17); Ctrl-C cancels, leaving marked words recorded and unmarked ones with no event at all.
- [x] Every review event names the form that asked it, so the two deferred remedies can later be chosen from the log rather than from argument.
- [x] The `Question` interface is unchanged, or the change is form-agnostic — it GAINED `Form()`, which every form answers and the session never branches on — the session still learns nothing about which form is asking (`#6`'s Done-when, `TestSessionIsFormAgnostic`).
- [x] Measured: a sitting of N words through the board costs materially less per word than the same N through form 2.3 — **one keystroke per word, and at least ten to one on what the learner has to read**. (Swept by plan revision R5: keystrokes alone are a wash at 1.00 against 1.00; the measured ratio on transcript lines is 20-40x, and that is the Spec's own ten-to-one claim.)

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

- [x] Design via `sdlc start-plan` before implementing.

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
- 2026-09-01: closed — go test ./... green; go test ./cmd/define/ -race green; go test -tags conformance -run PTY ./cmd/define/ green (154s, UNSANDBOXED — in-sandbox every pty row reports "no pty available", the #37 state where a row certifies nothing).; review verdict: FIX-THEN-SHIP

ROUND 4 left ONE blocker, BR-8, with rows 2 and 3 of its own enumeration open. Both closed as R17, and they were the same mistake in two costumes: a fact about the terminal read ONCE where it had to be read every frame.

Row 2 — a board that becomes current AFTER a resize was never told. g.Resize was called from the resize case on s.Current(), which fixes the board on screen at that instant and no other; the next board was built by todaysQuestions at the old width, painted with rows too wide, wrapped, and brought the wrong-word click back. show() is the ONE place that draws, so it is the only place that can promise this for every board — and it asks the SCREEN for the terminal shape, through a new Size on the display seam, rather than the loop keeping a copy that would be right until the first SIGWINCH it missed. Resize is idempotent, so this costs one comparison per frame.

Row 3 — Enter took words the window never drew. fitFooter drops trailing footer rows, so a shrunken terminal does not paint some grid rows, and Enter records Wrong for every unmarked word including those: boxes halved on one keystroke, for words the learner had no chance to look at. Enter is now HELD while the board is not whole and the prompt says why; the LOOP refuses rather than the session, because what was DRAWN is the terminal business and play is guarded pure (D6 applied to a destructive key). Marking still works and Ctrl-C is still free.

AND THE REASON I MISSED ROW 3 FOR TWO ROUNDS, recorded as a lesson: I checked "dropped rows are harmless" against the CLICK MAP — FooterRowAt answers nothing for a row never painted, so a click cannot reach one — and then wrote "harmless" into three places. A sweep does not go through the click map at all. The word travelled from the mechanism it was verified against to a different one without being re-checked, which is the same failure as a stale prose enumeration. A safety claim names the path it was checked on.

Five mutations against R17, all caught. The doc-comment guard also caught me inserting Size between FooterRowAt comment and its function, which is exactly the class it was built for.

ALSO IN THIS WINDOW — the operator drove a real sitting and returned four corrections, recorded as R16. A marked cell is PAINTED and keeps its key (the mark used to stand where the key was; the key is how a mouse-less terminal reaches the cell and how a learner reads the grid back). play.Palette carries the sequences in from main, which owns newPalette, and padding sits outside the style so the click map is untouched — this is the first feature to spend what D10 bought, since a grid in the append-only buffer could never repaint a cell. `d` is a LABEL now: the gap protected a key that did nothing on that screen because D12 had already refused the drop for a form holding many words, so the rule is the sharper one it always was — d is reserved for forms that HAVE a current word. One blank buffer line as a board opens. BoardLabels is exported, because restoring d broke three restatements of the old sequence.

The lesson from that: a review checks the code does what the plan says; only a sitting checks whether the plan said the right thing. Filed #42 from the same sitting (retire form 2.1 — it and the board are the same instrument and the board is sixteen times cheaper), deliberately NOT folded into this issue.

Rounds 1-3 all remain fixed and pinned. All 14 Done-when rows have a named pin that RESOLVES (TestPlanCitesTestsThatExist enforces it) and every red-when was executed as a mutation; the per-row table is in the issue Log. #38 TestPlayClickActsAndIsNotAnAnswer green and UNTOUCHED throughout. Plan revisions R1-R17. workshop/lessons.md gains sixteen rules across the four rounds plus the sitting.

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

### 2026-09-01 — T8, T9, T10: the box picks the form

**T8 — and this is the first thing in the program to consult a box when choosing
HOW to ask.** Selection was a capability question until now: form 2.3 when the
deck could supply distractors, 2.1 when it could not, and nothing looked at a box
at all.

**Singles first, then boards, and that is a choice.** Packing is the point of the
form — sixteen words for sixteen keystrokes — and packing cannot preserve the
queue's interleaving, because a board built from words scattered through it has
to sit somewhere. So retrieval gets the freshest attention and the maintenance
sweep comes after. The counter-argument is real and now MEASURABLE: a tired
learner marks everything yes, which is the illusion-of-knowing the Spec worries
about, and `ReviewEvent.Form` is what will eventually say whether it happens.

**`opt.rows`, and why it is not `opt.width`.** Width is a WRAP POLICY and may be
zero — a pipe wants no baked-in breaks. Height has no such policy; a frame is as
tall as the terminal is. Read at flag-parse time because form selection needs it
before the screen exists.

**And the trap the continuation warned about twice, caught before it bit.**
`playRig` returns an `options` with no `rows`, which is zero, which makes
`fitsABoard` false for every board — so `#40`'s entire form would never have been
offered in any test while every test stayed green. `rows: defaultRows` now, with
the same comment `width: defaultCols` carries. The mutation that removes it is
caught by `TestASittingOfDueWordsIsABoard`.

**T9 — the relearn line is built from the OUTCOMES, not the board's marks.** The
two are the same by construction, which is exactly why the choice matters: a line
assembled from the form could one day disagree with the events, and a transcript
that disagrees with the log is worse than no transcript. An all-yes board writes
nothing — a bare `relearn:` reads as a list that failed to render.

**T10 — `Batch` gains `Words()`.** `fig.done` was already a word count (every
mark scores), so a board made the bar compare words against slots and a
twenty-word sitting read "0 of 2". The budget was never affected: `schedule.Queue`
returns that many keys whatever they are packed into.

### 2026-09-01 — T11, T13, T12: the measurement, the real terminal, the docs

**T11 is recorded above as plan revision R5** — the keystroke proxy was a wash
and the reading cost is 20-40x, so the row now pins both and the claim rests on
the one the numbers support.

**T13 — and the pty rows had to be run OUTSIDE the sandbox.** In-sandbox they
report `no pty available: operation not permitted`, which is exactly the state
`#37` records for the review environment: a conformance row that skips is a row
that certifies nothing. Run with the sandbox off, the board draws whole in a
24x80 window, a real SGR mouse report marks the cell under the pointer, Tab flips
the toggle on a real terminal, the relearn line survives into the exit
transcript, and the log names `form: board` four times.

The click's ROW and COLUMN are read off the paint rather than computed. The
frame and the click map are the two things that have to agree, so a premise taken
from one of them could not catch the two disagreeing — and the first run failed
on the frame's last row, which carries the cursor walk-back and the reprinted
prompt appended to it (`unstyled` strips colour, not motion).

**T12 — and the "`--help` key table" the plan names does not exist.** `--play`'s
flag help is one line, and `/help` is the REPL's command list. The key table is
the README's, which now has the board's keys, the click, Tab, the split meaning
of Enter, and the note that `d` is not offered on a board.

**The two forward references D7 falsified are corrected rather than deleted.**
`schedule/progress.go` said this issue would extend the `Grade` seam with
`GradeUnsure`; it now records that the extension did NOT happen and why, because
"the seam is still the right one to extend" is worth keeping and a silent
deletion would lose it. `play/session.go` described the board's mark as `firm`,
which was the deleted three-mark draft's word.

**And the board joined `doc_sync_test.go`'s forms slice**, so its prompt line is
now checked against the README by the build. That row earns its place twice: the
board's line differs in the RESERVED half too, so it is the only one that can
catch `reservedKeys` regressing and offering `d` again.

### 2026-09-01 — the Done-when sweep: fourteen rows, every `red when` executed

The plan's preamble commits to it: *"Every `red when` cell is EXECUTED as a
mutation at the boundary and the result recorded per row"* — because a row that
survives its own mutation pins nothing (`#38` BR-16).

| # | claim | mutation run | caught by |
|---|---|---|---|
| 1 | a sitting of eligible words is a grid | `todaysQuestions` never builds a board | `TestASittingOfDueWordsIsABoard` |
| 2 | every mark reaches the log as it happens | `advance` holds the record until the form is spent | `TestEveryMarkOnABoardIsRecordedImmediately`, `TestCtrlCCancels…`, `TestABoardIsMarkableByKeyAlone` |
| 3 | Enter takes the unmarked as No, space does not | `InputFinish` loses its `fallthrough`; `toInput` maps Enter to `InputReveal` again | `TestEnterSpendsABatchFormAndSpaceDoesNot`, `TestToInputSplitsEnterFromSpaceAndCarriesTab`, `TestEnterStillRevealsOnASingleWordForm` |
| 4 | Ctrl-C leaves unmarked words untouched | `InputQuit` spends the batch and emits a record per word | `TestCtrlCCancelsABoardWithoutMovingUnmarkedWords` |
| 5 | a No does not freeze the board | the `batchOf` guard removed from the miss-on-hidden branch | `TestABoardRunsThroughTheSession` |
| 6 | the mouse-less path works, `d` is not a label | `boardLabels` becomes `…abcdef`; `InputDrop`'s batch refusal removed | `TestABoardIsMarkableByKeyAlone`, `TestBoardLabelsSkipTheReservedD`, `TestDOnABoardIsNotACellLabel` |
| 7 | a click marks here and plays elsewhere | the loop never offers the click to the form; the footer row is used as the cell; the viewport row is used unsubtracted | `TestAClickOnABoardMarksIt`, `TestFormCellAsksTheScreenAndTheForm`, and `#38`'s row green UNTOUCHED |
| 8 | the session names no form | `batchOf` type-switches on `*Board` | `TestTheSessionNamesNoForm` |
| 9 | every event names its form | the stamp removed; moved to the wrong outcome kind; the capture site drops it; `yaml:"-"` | `TestEveryRecordNamesItsForm`, `TestAReviewEventNamesItsFormOnDisk`, `TestYAMLRoundTripsTheFormThatAsked` |
| 10 | a board is never drawn clipped | `fitsABoard` returns true always; the chrome under-counted; the width floor removed | `TestAShortTerminalGetsMeaningChoiceNotAClippedBoard`, `TestFitsABoardCountsTheWholeLiveEdge`; `TestPaintFitsTheTerminalAndParksTheCursor` UNCHANGED |
| 11 | the outcome survives the sitting | the relearn write removed; it names every marked word; a bare line on an all-yes board | `TestABoardLeavesItsRelearnListInTheTranscript`, `TestABoardWithNothingToRelearnWritesNoLine` |
| 12 | the bar counts words | `fig.total` back to `len(s.Questions)`; a board counts as one | `TestTheBarCountsWordsNotSlots` |
| 13 | the board is materially cheaper | the grid goes into the buffer like any prompt | `TestABoardCostsFarLessPerWordThanMeaningChoice` (see R5 — the claim moved to what is READ) |
| 14 | the real terminal draws and clicks it | run on a real pty, unsandboxed | `TestPTYPlayBoardIsDrawnAndClickable` |

**Fifty-eight mutations across the thirteen tasks. Three survived and all three
were real**, which is the whole argument for running them:

1. **A `row >= g.Rows()` bound in `formCell`** that `CellAt` already enforced.
   Removed — two owners of one bound is how the toggle row becomes a cell.
2. **A `row >= gridRows()` bound inside `CellAt`** that `i >= len(cells)`
   already enforced, one layer down from the first. Removed, and the PROPERTY
   pinned separately from the mechanism over eight board shapes and every column.
3. **`yaml:"-"` on `ReviewEvent.Form` left the whole suite green**, because the
   loop's tests read through `store.Mem` and it keeps events in memory. A field
   that reaches only memory answers nothing the query it exists for needs.

A fourth "survivor" was an equivalent mutant and worth naming as one: marking a
batch's rest inside `InputQuit` without emitting outcomes changes nothing,
because the form is discarded and the LOG is what the row asserts on. The
stronger mutation — emitting a record per word — reddens it.

### 2026-09-01 — boundary review round 1: REWORK, and what the three blockers were really about

Seven findings, three blocking. Every one was real, and each fix went at the
CLASS rather than the site. Sidecar:
`workshop/plans/000040-form-board-close-review.md`.

**BR-1 Critical — space on a board emitted `OutcomeReveal`.** Reproduced by the
reviewer through execution: the loop filed a blank reveal into the append-only
buffer and called `playAnnounced` on `q.Word()`, which on a grid is the
last-marked cell or cell 0 before any mark. D14 and the board's own doc both say
`InputReveal` must do nothing for a form holding many words. Space is the natural
key to press — it reveals on both other forms and the board's prompt does not
mention it — so this was reachable in the first real sitting.

**The defect is D12's enumeration, not the missing guard.** It named FOUR `Apply`
paths that must consult `Batch`, found by measurement, and the plan treated the
list as the deliverable. The real shape is `InputKind × Batch` and `InputReveal`
was a fifth cell. `numInputKinds` is a sentinel now and
`TestEveryInputKindIsAnsweredForABatchForm` ranges over it, so the next kind
added arrives with no expectation and fails. `choice.go`'s `numAxes` had
established that exact pattern in this package — *"the guard derives the set from
this, never from a list"* — and the list got written down anyway.

**Why the existing test missed it:** `TestEnterSpendsABatchFormAndSpaceDoesNot`
asserted only that space did not spend or record. "Harmless" and "inert" are
different claims, and only the second is what D14 promised.

**BR-2 Important — a constant standing in for a measurement.**
`boardChromeRows = 2` charged the keys prompt one row; that line is 76 columns
and a board was offered from 20. Below 76 the live edge was under-budgeted and
`fitFooter` dropped from the end: the bar under 76 columns, the panel at 38 or
less, the TOGGLE at 25 or less — the one owner of which mark is live, while every
mark is irreversible. `displayRows` already answers "how tall is this line at
this width"; the constant was a second, implicit owner of it.

The fix keeps ONE constant on purpose. `barRows = 1` is a minimum rather than a
measurement, because `fitFooter` drops from the END and the bar is last, so a bar
that needs three rows is dropped instead of costing the board anything. **A
budget only has to measure what it can be squeezed by.**

**BR-3 Important — the Done-when table cited four tests that were never
written**, and an `(R3)` that did not exist. Corrected, and the class guarded:
`TestPlanCitesTestsThatExist` resolves every backticked `Test*` in an active plan
to a real `func Test…(`.

**And mutation-checking THAT guard found something worse.** Citing a fake test
left it green: `currentTruthOnly` cuts a plan at its FIRST `## Revisions`, this
plan had grown two, and the first sat above `## Done when` — so both plan guards
had been reading a truncated file and passing on it, including every earlier
sweep this Log records as clean. The revisions are one appended section now.
**A guard added without a mutation is a guard nobody has seen fail**, and this
one would have shipped certifying nothing.

**Four Minors, all fixed:** `Rows()` and `Prompt()` now agree BY CONSTRUCTION
(built as a slice) rather than by arithmetic that happened to match everywhere
but the empty board; the keystroke half of Done-when 13 asserted its own fixture
(`len(boardKeys) > len(words)` cannot fail) and now asserts the board is SPENT; a
cell's trailing padding is documented and pinned as its own click target; the
comment claiming the loop adds colour described an intention rather than the
code, and now says deferred. The fifth — the relearn line on the Ctrl-C path,
load-bearing on `Current()` returning nil once `Done` — has its own test.

**One architectural note taken up:** `boardFooter`'s "the form's rows come first,
in order" was called load-bearing and tested by nothing.
`TestBoardFooterPutsTheFormsOwnRowsFirst` is that pin, and it is what a second
live-edge form will hit first.

**Reviewer's coverage note, accepted:** the pty row SKIPPED in their environment
(`no pty available`), so Done-when 14 rested on this session's out-of-band run.
Re-run after the fixes, unsandboxed: `TestPTYPlayBoardIsDrawnAndClickable`,
`TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` and `TestPTYPlayGradeFirst` all
pass. That the row cannot be independently confirmed in a sandboxed review is
`#37`'s subject, not this issue's.

### 2026-09-01 — boundary review round 2: REWORK again, and why "fix the class" was not yet what I did

Four new findings, **three repeat families**, and the gate's verdict on the round
was the useful part: *"Not converging: fix rules, not instances."* It was right.

**BR-8 Critical, family `frame-budget-hardcoded-not-measured` — SECOND
occurrence.** Round 1's BR-2 fixed the prompt's height and left the board's
LAYOUT WIDTH fixed at selection time. The reviewer measured what a resize does: a
board laid out for eighty columns has 74-column rows, at forty the terminal wraps
each into two, a footer entry stops being one physical row, and `formCell` handed
the raw physical column to `CellAt` — column 4 of a continuation marking cell 0
while the word drawn there is another. **Permanent**, because the mark is already
in the log. The same resize at 24x12 drops the toggle, the panel and the bar,
which is BR-2's exact harm through a door `fitsABoard` cannot see.

D15 had said this was safe (*"leaves the current board drawn as it was — its rows
are already budgeted"*) and `Board`'s field comment said the width *"cannot
change"*. Both wrong, both now corrected in R9 with the enumeration written out:
prompt height, layout width, the fit after a resize, and a wrapped entry's
columns.

**BR-9 Important, family `plan-citations-unenforced` — SECOND occurrence.** Round
1 DIAGNOSED the mechanism correctly — `currentTruthOnly` cuts at the first
`## Revisions`, this plan had two, both guards read a truncated file — and then
fixed this plan's layout. The reviewer's point: the shape returns tomorrow in any
artifact, and it returns as SUCCESS, because four of the eight guards reading
that filter end in `checked == 0 → t.Skip`. Measured: none of the four asserts a
premise about the filtered view, while seven other guards in the same file Fatal
on vacuity. R10.

**BR-10 Minor, family `comment-asserts-absent-behaviour` — SECOND occurrence.**
The `Batch` doc still said *"consulted at FOUR points"* and listed three — inside
the commit that declared prose enumerations the culprit. The number was never the
point; it now points at `numInputKinds` and the table test.

**The lesson, and it is the one worth keeping from this issue.** Round 1 fixed
each finding at what I believed was the class: a sentinel and a matrix test for
the stale enumeration, a measured height for the constant, a new guard for the
uncited tests. Every one of those was a real improvement and none of them was the
rule. **When a review names a family, enumerate every member before fixing one** —
the question is not "where else does this exact bug appear" but *"what else is
this quantity read from, and when"*. Writing the enumeration INTO the fix, as a
derived set or a measured value or a table in the revision, is what stops round
three.

Verification after the fixes: `go test ./...` green, `-race` green, and the pty
rows that touch resize — `TestPTYResizeRepaints`, `TestPTYPlayResizeRepaints`,
`TestPTYPlayBoardIsDrawnAndClickable` — all pass unsandboxed. Four mutations run
against R9's fixes and four caught; the `currentTruthOnly` fix was mutated with
the exact shape that produced BR-3 (a `## Revisions` above `## Core concepts`
plus a fabricated test name) and now fails naming the swallowed section.

### 2026-09-01 — the operator's first sitting: four corrections

Recorded in full as plan revision R16. Three boundary-review rounds found real
defects and none of them found these, because all four are correct code that
reads wrong to the person using it.

- **A marked cell is painted and KEEPS its key.** The mark used to stand where
  the key was. The key is how a mouse-less terminal reaches the cell and how a
  learner reads the grid back — the wrong half to spend on saying "answered".
  `play.Palette` brings the sequences in from `main`; padding sits outside the
  style so the click map is untouched. It is also the first thing to actually
  spend what D10 bought: a grid in the append-only buffer could never repaint.
- **`d` is a label.** The hole at `d` protected a key that did nothing on that
  screen, because D12 had already refused the drop for a form holding many words.
  The rule is sharper now: `d` is reserved for forms that HAVE a current word.
- **A blank buffer line as the board opens**, since a board writes nothing else
  there and its grid began flush against the previous question.
- **`BoardLabels` is exported**, because restoring `d` broke two tests and a pty
  row that each restated the old sequence.

**The lesson: a boundary review checks that the thing does what the plan says;
only a sitting checks whether the plan said the right thing.**

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
