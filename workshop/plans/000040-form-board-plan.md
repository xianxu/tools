# Form 2.5: The Board Implementation Plan (`#40`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mature words are triaged sixteen at a time on a grid — one keystroke each — so a large deck stays affordable, and the scheduler rather than the learner decides which words get the cheap form.

**Architecture:** `Board` is a third `play.Question`, pure, and the session learns nothing new about forms. It holds more than one word, which the session discovers by ASKING — a third optional capability interface beside `SelfRated` and `Missed`, not a type switch. `--play` already owns coordinates since `#41`, so the grid is a `Prompt()` string like any other.

**Tech Stack:** Go 1.26. No new dependency, no new package.

---

## What this plan asserts about the existing tree, verified

| claim | verified at | status |
|---|---|---|
| `Question` is one word, one verdict | `play/question.go:54-91` | true — `Word()`, `Grade(rune) (Verdict, bool)` |
| the session already asks forms about CAPABILITIES rather than switching on type | `play/session.go:287` (`Missed`), `:311` (`SelfRated`) | true — two precedents, both optional interfaces |
| `advance` unconditionally moves to the next question | `play/session.go:261-267` | true — `s.Index++` on every graded answer |
| a self-rated form can never earn the two-rung promotion | `play/session.go:317-323` | true — `unaidedNow` returns false when `IsSelfRated()` |
| `GradeCorrect` is `+1` at or above MaxBox and `+2` below it | `schedule/progress.go:79-86` | true — the express lane back after a lapse |
| the queue's budget is in WORDS | `play_loop.go`, `schedule.Queue(deck, prog, now, budget)` | true — returns `budget` keys, so packing them into boards does not change how many words a sitting asks about |
| the bar's total is the SLOT count | `play_loop.go`, `fig.total = len(s.Questions)` | true — which is why T6 exists |
| `MasteredBox` is 9, and the Spec's board threshold is 8 | `schedule/progress.go:31`, the issue's Spec | true — one apart, and D6 decides what that means |

---

## Decisions

> **Revised 2026-09-01 from the operator's sketch and two follow-ups.** The
> original Spec's board was a maturity-triage form driven by the box, marked with
> a keystroke per word, carrying three marks. It is now a CLICK-driven grid over
> all of today's words, with two marks. Four of the first draft's decisions were
> deleted by that; what they were and why they went is recorded below, because a
> plan that silently loses a decision cannot be argued with later.

**D1 — TWO OF THE SPEC'S REQUIREMENTS ARE ALREADY IN THE TREE.**

The Spec asks for a Revision to `#39` saying `+2` should come from *"a correct
answer in form 2.3 with no reveal first"*. That is `unaidedNow`
(`play/session.go:317`), which computes exactly `v == Correct && !s.Revealed &&
!IsSelfRated()`, and `schedule.Answer` already promotes `GradeUnaided` by two.
**`#39` shipped this; it needs no revision.**

And a `Board` implementing `SelfRated` cannot produce `GradeUnaided`, so *"Yes is
not a confident answer"* is structural rather than a rule anyone must remember.
No new grade, no exception in `Answer`.

One nuance the Spec's table does not state and the code does: `GradeCorrect` is
`+2` when `Box < MaxBox` — the express lane back after a lapse. A `Yes` on a
lapsed word climbs two. Correct and consistent, and written down here because the
table reads `box + 1` flatly.

**D2 — the session discovers that a form holds many words by ASKING.**

`advance` moves on after every graded answer (`session.go:263`). A board must
stay current while sixteen marks land, and the session must not learn what a
board is (`#6`'s Done-when, `TestSessionIsFormAgnostic`). So the question goes to
the FORM — the third instance of a pattern `play` already has twice:

```go
// Batch is implemented by forms that hold more than one word.
type Batch interface {
	// Spent reports whether every word this form holds has been answered.
	Spent() bool
	// Rest answers every word still unmarked and returns them, which is what
	// Enter means to a form holding many.
	Rest(v Verdict) []string
}
```

`Missed` (`session.go:287`) and `SelfRated` (`:311`) exist because *"a type switch
on `*Choice` would be the thing Done-when 7 forbids"*. This is the same sentence
about `*Board`.

**D3 — ENTER COMMITS, CTRL-C CANCELS, and the cancel is free.**

Operator, 2026-09-01: *"ctrl-C means nothing is changed from that form (board).
already clicked words can still be recorded, but unmarked words, are just
unmarked, no state change for them."*

That behaviour already exists and needs no code. Every click emits its own
`OutcomeRecord`, which `CaptureReview` writes immediately — the property
`play_loop.go` states as *"recorded NOW, before the next question is drawn… what
makes Ctrl-C lossless by construction rather than by a flush"*. `Apply` on
`InputQuit` emits only `OutcomeDone`. So an interrupted board leaves clicked
words written and unmarked words with NO event, `Fold` leaves their boxes where
they were, and they are due again tomorrow.

**Enter is the half that needs code**, which is the inversion worth noticing: the
DESTRUCTIVE path carries the logic and the cancel path is the absence of it.
Enter reaches `Apply` as `InputReveal`, which a board has nothing to do with — it
reveals nothing — so `Apply` asks the `Batch` capability and, for a form that has
one, spends it: `Rest(Wrong)` marks every unmarked word, one `OutcomeRecord`
each, and the session advances.

**Sixteen demotions from one keystroke is deliberate** — it means *"I am out of
time, ask me all of these again"* — and it is the only expensive-to-undo action
on this surface, which is why the alternative had to exist before it shipped.

**D4 — THE BOX PICKS THE FORM, AND THE BOARD IS THE FIRST FORM IT HAS EVER
PICKED.**

Worth stating plainly because it is easy to get backwards: **form selection today
is a CAPABILITY question, not a scheduling one.** `todaysQuestions` calls
`choiceFor`, which returns form 2.3 when it can pick two usable distractors and
falls back to 2.1 when it cannot — a young deck, an entry that is only
cross-references, an entry that defines a different word. Nothing consults a box.
The board introduces box-based selection.

The Spec set the line at 8. Operator, 2026-09-01: *"not removing box as selector,
I guess just allow form-box to be used earlier."*

**And the risk is not shaped the way the Spec assumed.** The comparison that
matters is not "days added" but *how much longer than a real test would have
allowed*: a wrong `Yes` sends the word to `box+1`, where form 2.3 would have
caught it and sent it to `box/2`.

| box | 0 | 2 | 4 | 6 | 8 | 10 |
|---|---|---|---|---|---|---|
| days a wrong `Yes` buys | **0** | 3 | 8 | 22 | 62 | 165 |

Delay grows steeply with the box while the CHANCE of a wrong `Yes` falls with it —
a box-10 word has been recalled ten times. The product peaks in the middle,
around **boxes 5–7**, where the board buys 15–20 days on a word that is not
secure. Both ends are cheap, which is the opposite of the Spec's assumption that
high boxes are the safe place.

Boxes 0 and 1 are literally free: `box.go`'s ladder waits one day at both, so a
wrongly promoted new word returns tomorrow regardless.

**The threshold is box ≥ 3** — past the two free rungs, a 4-day interval, roughly
three recalls of history, and a wrong `Yes` buys four days. It is a starting
number rather than a derived one, and D4a is what makes it revisable from
evidence instead of from argument.

The operator's two candidate remedies — the board promotes more slowly, or it is
used less often — are both DEFERRED, to be chosen later from the log and
eventually made per-learner.

**D4a — WHICH MEANS THE LOG MUST RECORD THE FORM, and today it does not.**

`ReviewEvent` carries the word, the verdict, the axis, `Unaided` and the time
(`store/event.go:23-51`). Nothing says which form asked. Without it the log
cannot answer *"do board-promoted words lapse more than 2.3-promoted ones?"* —
so neither remedy can be chosen from evidence, and the configuration the operator
wants to learn later has nothing to be fitted against.

This is not one of the deferred limits. It is the instrument that makes deferring
them safe, and it matters because **the damage is silent and delayed**: a wrongly
promoted word vanishes for weeks, and when it is eventually forgotten that is
indistinguishable from ordinary forgetting. One field on the event, one line at
the capture site.

**CONFIRMED by the operator**, 2026-09-01: *"do record. I think it's a matter of
data mining: using this scheme, did user fail beyond reasonable in real recall
test later. but do add telemetry you need."* That names the query, which is what
decides the field: the analysis joins a board-promoted word to its NEXT real
test, so the form must be on the event that PROMOTED it, not on a summary
elsewhere. `ReviewEvent.Form` it is.

**D5 — CLICKS AND LABELLED KEYS, not a cursor.** The learner clicks a word to
mark it with the active mode; Tab swaps the mode between Yes and No. This is
`#38`'s affordance finding its second consumer, and it deletes the first draft's
D3 entirely — no cursor, no auto-advance, and no pressure on `play.Input` to grow
a movement kind.

**Every cell also carries a KEY, printed beside it**: `[0] arrondissement  [1]
bailiwick …`. Operator, 2026-09-01: *"let's use 0-9, and a-f."* Without it a
terminal reporting no mouse has no way to mark anything — Enter still works, so
the board would silently degrade to "everything is No", which is wrong rather
than merely limited, and `#38`'s pty rows exist precisely because a mouse-less
terminal must keep working.

**`d` IS NOT AVAILABLE, and the labels are `0`–`9` then `a b c e f g`.**
`toInput` intercepts `'d'` and `'D'` as *drop from deck* before any form sees the
key, and `Question.Grade`'s own doc states the rule: *"The session RESERVES some
keys before a form ever sees them… A form must not build its answer set from
those."* Sixteen labels skipping `d`. The gap is visible rather than surprising,
because the labels are PRINTED — nobody has to know the sequence.

**D6 — the toggle and the feedback panel are the LIVE EDGE, not buffer lines.**
The buffer is append-only, which is what makes a click's coordinates exact
(`#30` D1). Anything that changes in place is the prompt or the footer, and
`Paint` already takes a multi-row footer that gives up whole rows before the
prompt does. So the definition panel is the footer and costs nothing.

**AND SO IS THE TOGGLE.** The sketch put "Do you remember? [Yes] No" above the
grid, which would have required `screen` to grow a HEADER — rows above the buffer
that every frame-budget calculation would have to learn about, for one line.
Operator, 2026-09-01: *"it's fine to move it to the footer."* So the whole live
edge is at the bottom: the toggle, the definition panel, the keys, the bar — and
`screen` is untouched by this issue.

**D7 — `unsure` is DELETED.** Operator: *"I guess unsure means no."* Three marks
collapse to two, `Yes → GradeCorrect` and `No → GradeWrong`, and the first draft's
entire `EventUnsure` mechanism — a new event kind, a `Fold` exemption, and a
form-selection rule reading it — goes with it.

**D8 — the bar counts WORDS, not slots.** `fig.total = len(s.Questions)` is the
slot count, and a board is one slot holding up to sixteen words; a 20-word sitting
would read "0 of 2". The budget is unaffected — `schedule.Queue` returns `budget`
KEYS — so this is display, but it is the number the load argument is about.

**D9 — NOT in scope:** any per-learner configuration of the two deferred
remedies. D4a ships the instrument; fitting anything to it is a later issue.

**D10 — THE GRID IS THE LIVE EDGE, and that is the whole shape of this issue.**

The first draft put the grid in the buffer as a `Prompt()` string "like any other
form". It cannot be: `show()` writes a question into the buffer once per index
and the buffer is APPEND-ONLY — `addRegions` calls that load-bearing, because
*"nothing afterwards can move a line that is already written"* is what makes a
click's coordinates exact (`#30` D1). A grid written there is frozen at the
moment it is written, and marks that toggle green and red are impossible.

Four ways out were put to the operator; **D was chosen** (2026-09-01: *"D for
now. let's keep things simple before going full screen TUI"*):

| | keeps colour | keeps clicking | cost |
|---|---|---|---|
| A — grid in the buffer, marks only in the panel | ✗ | ✓ | loses the feature |
| B — grid in the footer | ✓ | ✗ | keyboard-only |
| C — `screen` rewrites the last N buffer lines | ✓ | ✓ | breaks the invariant clicks rest on |
| **D — grid in the footer, and `display` reports WHICH FOOTER ROW was clicked** | ✓ | ✓ | one question on the seam |

**D is the option that does not start a layout system.** The footer already
redraws every frame, already has a row budget (`fitFooter`), and `Paint` already
computes where it begins — it simply does not report it. A full-screen TUI means
arbitrary redrawable regions, focus and a component tree; C plus a header is the
first step down that road, and nothing here needs it yet.

**Two consequences, named rather than inherited:**

- **The footer's sacrifice order is now load-bearing.** `fitFooter` drops rows
  from the END, so `[grid, toggle, panel, bar]` loses the bar first and the panel
  next, which is the right order by luck. **Grid rows must never drop** — half a
  board is unusable — so the fit needs a floor rather than the plain tail-drop.
- **The board is NOT in the exit transcript**, because the live edge is ephemeral
  by design (`#41` D4). Right for triage, wrong for the outcome: the words marked
  `No` are the ones worth keeping, so the board writes ONE buffer line as it
  closes — the relearn list — and nothing else.

**D11 — A CLICK MEANS "MARK" ON A BOARD AND "PLAY" EVERYWHERE ELSE, so the loop
asks the form which it is.**

`#38` shipped the invariant *"a click ACTS and never answers"*, pinned by
`TestPlayClickActsAndIsNotAnAnswer`: a click stops before `toInput` so
`play.Apply` never sees it. A board reverses that — its clicks ARE answers.

The loop therefore offers a click to the current form first and falls through to
`playRegion` when the form does not take it. The invariant survives, restated
honestly: **a click never answers a form that did not ask for it.** Every
existing form declines, so `#38`'s row stays green unchanged, which is the proof
the seam widened rather than branched.

**D12 — `Spent()` IS NOT ENOUGH; three other `Apply` paths assume one word.**

Measured, not reasoned:

- **A `No` mark would FREEZE the board.** It reaches the miss-on-a-hidden-word
  branch (`session.go`), which sets `s.Revealed, s.Graded = true, true` — so the
  next keystroke means "any key = next word" and the board ends after one mark.
- **`InputDrop`** advances with `Skipped` and drops `q.Word()`. On a board there
  is no single current word, so `d` has nothing to name. It is REFUSED on a
  board rather than given a guess.
- **`livePrompt`** returns `gradedPrompt` when `s.Graded`, which a board never is.

So the `Batch` capability is consulted at each of those points, not only in
`advance`. That is more surface than the first draft claimed, and it is the
honest cost of a form that holds many words.

**D13 — Tab reaches nothing today.** `key.go` decodes `KeyTab`, and `toInput` has
no case for it, so it is dropped before `play` sees it. The mode toggle needs it,
which means one row in `toInput` and one `play.Input` kind — and unlike the
paging keys (`#41` D6) this one is legitimately about WHAT IS BEING ANSWERED
rather than what is being looked at, so it belongs in `play` rather than being
intercepted by the loop.

**D14 — ENTER COMMITS AND SPACE MUST NOT, so the merged pair splits.**

`toInput` maps BOTH Enter and space to `InputReveal`, and `Apply`'s own comment
calls that deliberate: *"Enter and space both land on this kind, which toInput
maps to the same kind."* D3 spends the board on `InputReveal` — so **space would
commit it**, taking every unmarked word as `No`. A casual keystroke would fire
the one action on this surface that is expensive to undo.

So `Enter` gets its own kind, `InputFinish`, and space keeps `InputReveal`. For
every existing form `Apply` treats `InputFinish` exactly as `InputReveal`, so
2.1 and 2.3 are unchanged and their tests pass untouched — that equivalence is
the proof the pair split without the forms noticing. Only a `Batch` form
distinguishes them: `InputFinish` spends it, `InputReveal` does nothing, because
a board has nothing to reveal.

**D15 — A BOARD THAT DOES NOT FIT IS NOT OFFERED, which is what keeps
`fitFooter`'s budget invariant true.**

`fitFooter` guarantees `footerRows <= avail` today, and `Paint` rests on it:
`s.rows` is computed from it and so is the cursor walk-back. `Paint`'s own
comment states the consequence of breaking it — *"the terminal then SCROLLS to
fit it, which moves every row the app believes it placed, and a click at viewport
row R stops meaning buffer line R+offset"* — which lands squarely on
`FooterRowAt`, the seam this issue adds. **A floor that simply refuses to drop
grid rows would violate it on a short terminal**, and the failure would be a
click on the wrong word.

So the floor is not a floor. The board is offered only when the terminal can hold
it: grid rows + the toggle + the prompt + the bar. Below that height those words
go to form 2.3 for that sitting, decided in `boardsFor` where the form is chosen
— **a board that cannot be drawn whole is not a board**, and 2.3 is a complete
answer rather than a degraded one.

`fitFooter` is therefore UNCHANGED, which also retires D10's second consequence:
there is no sacrifice-order problem, because the board is never in a footer that
has to sacrifice. A resize below the minimum mid-sitting leaves the current board
drawn as it was — its rows are already budgeted — and the next question is
chosen at the new height.

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Board` | `cmd/define/play/board.go` | new | PURE — form 2.5: N words, two marks each, a mode. Implements `Question`, `SelfRated`, `Batch`, `Grid` and `Moded`. Draws its OWN live edge — the grid and the panel — so the loop composes nothing (R1); the mode rides on `Keys()`, the row that survives a short terminal (R11) |
| `Cell` | `cmd/define/play/board.go` | new | PURE — one word and the one-line gloss the panel shows when it is marked (R1) |
| `Grid` | `cmd/define/play/session.go` | new | PURE — the capability "I am drawn as cells you click". The first capability the LOOP asks rather than `Apply` (D11), and the methods are the type's rather than this row's to list (R9 added `Resize`) |
| `Moded` | `cmd/define/play/session.go` | new | PURE — the capability "Tab means something to me" (D13) |
| `Mark` | `cmd/define/play/board.go` | new | PURE — `Yes`/`No` and the `Verdict` each maps to. TWO marks: `unsure` is deleted (D7) |
| `Batch` | `cmd/define/play/session.go` | new | PURE — the capability "I hold more than one word". The set of paths that consult it is `InputKind × Batch`, DERIVED from `numInputKinds` by `TestEveryInputKindIsAnsweredForABatchForm` rather than counted in prose (R7), plus `Words()` for the bar (D8) |
| `Apply` | `cmd/define/play/session.go` | modified | PURE — consults `Batch` wherever an input can mean something different to a form holding many words, and stamps every record's `Form` in one place (D2, D3, D12, D4a, R7) |
| `livePrompt` | `cmd/define/play_loop.go` | unchanged | PURE — D12 predicted a guard here and none was needed: a board is never `Graded`, so the graded prompt cannot fire (R2) |
| `gradePrompt` | `cmd/define/play_loop.go` | modified | PURE — asks `reservedKeys` rather than naming the constant, because `d` is refused on a board; the mode rides on it as of R11, via the form's own `Keys()` (R2, R11) |
| `reservedKeys` | `cmd/define/play_loop.go` | new | PURE — the session's reserved keys FOR THIS FORM. `d` is not among them for a form holding many (D12, R2) |
| `boardFooter` | `cmd/define/play_loop.go` | new | PURE — the board's own rows, then the bar. One line, because the form owns the rest (R1) |
| `fitsABoard` | `cmd/define/play_loop.go` | new | PURE — is this terminal tall enough to draw the board whole (D15) |
| `boardsFor` | `cmd/define/play_loop.go` | new | PURE — partitions the day's keys into boards and single questions at box ≥ 3 (D4) |
| `fitFooter` | `cmd/define/screen.go` | unchanged | PURE — D15 retired D10's floor: a board that does not fit is not OFFERED, which keeps this function's budget invariant true rather than negotiating with it (R2) |
| `sittingWords` | `cmd/define/play_loop.go` | new | PURE — the bar's `total` becomes the WORD count. In the loop, where `fig.total` is set, rather than in `sittingFigures`, which never knew about questions at all (D8, R4) |
| `relearnLine` | `cmd/define/play_loop.go` | new | PURE — the one buffer line a board leaves as it closes, built from the outcomes the loop RECORDED so the transcript cannot disagree with the log (D10) |
| `boardFits` | `cmd/define/play_loop.go` | new | PURE — builds a probe board and asks it how tall it is, because the form owns its own layout (D15) |

- **`Board`** — up to sixteen words, marked by click or by labelled key.
  - **Relationships:** 1:N with words (N ≤ 16); occupies ONE `Session.Questions` slot.
  - **DRY rationale:** `Word()` returns the word a mark is landing on, so `advance`'s existing `Outcome{Word: q.Word()}` records the right word with no change at the call site.
  - **Future extensions:** a second many-word form is another `Batch` implementer and nothing else moves.

- **`Batch`** — the session's question, never the form's announcement.
  - **DRY rationale:** third instance of the pattern. `Missed` and `SelfRated` exist because *"a type switch on `*Choice` would be the thing Done-when 7 forbids"*; this is that sentence about `*Board`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `display` | `cmd/define/replraw.go` | modified | the seam gains ONE question: which footer row a click landed on. The editor answers "none", which is the whole of its involvement (D10) |
| `FooterRowAt` | `cmd/define/screen.go` | new | the terminal — `Paint` already computes the footer's origin and simply does not report it. This is that report, and it is what makes the live edge clickable |
| `toInput` | `cmd/define/play_loop.go` | modified | the keyboard — a row for Tab, which is dropped today (D13), and Enter split from space (D14) |
| `formCell` | `cmd/define/play_loop.go` | new | the mouse — asks the screen which footer entry and the form which cell, and does the subtraction between two answers it is not qualified to give itself (D11) |
| the loop's click branch | `cmd/define/play_loop.go` | modified | the mouse — offers a click to the form first, falls through to `playRegion` (D11) |
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | the store — packs box ≥ 3 keys into boards (D4) |
| `ReviewEvent.Form` | `cmd/define/store/event.go` | new | the event log — which form asked, so the deferred remedies can be chosen from evidence (D4a) |
| `Question` | `cmd/define/play/question.go` | modified | gains `Form() string`. ON the interface, not an optional capability: a form that could forget to name itself would write an empty string beside a real promotion (R3) |
| `CaptureReview` | `cmd/define/capture.go` | modified | the store — writes the form on every review event |
| `reservedKeys` | `cmd/define/play_loop.go` | new | the learner — `d` is refused on a board, so the prompt stops offering it. `TestREADMEQuotesThePromptsTheLoopActuallyPrints` now covers all three forms (T12) |

**ARCH-MOCK.** No new external dependency. `Board` is pure and unit-tested with no IO; the loop's tests use the recorder `#30` built; the pty suite covers the real terminal, and the mouse-less path is exactly what `#38`'s rows exist for.

**ARCH-CONSTRAINTS.** Keystroke-and-redraw, inherited from `#41` unchanged: 16ms throttle, O(visible rows) per frame. The board moves work from the buffer to the FOOTER, which is repainted every frame — so a board's grid is rebuilt per keystroke rather than written once. That is O(16 cells) of string building at human typing rates, the same order as form 2.3's four options, and it is the price of marks that change colour. `boardsFor` is O(deck) once per sitting.

**ARCH-PURE.** `Board` and `Mark` import nothing; `cmd/define/puretest` enforces it.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [x] **T1 — `Batch`, and the places `Apply` consults it** (D2, D12; the count is derived, not listed — R7). The interface, then: `advance` moves on only when `Spent()`; the miss-on-hidden branch must not set `Graded` for a batch form; `InputDrop` is refused; Enter spends the board via `Rest(Wrong)`. Existing forms implement none of it and are unaffected — the whole `play` suite and `TestSessionIsFormAgnostic` pass untouched, which is what proves the seam widened rather than branched.
- [x] **T2 — `Board` and `Mark`** (D5, D7). Two marks. `Prompt()` renders the labelled grid; `Grade` takes a cell label; `Mark(i, v)` takes a click; `Spent()`; `Rest(v)`; `IsSelfRated() → true`; `Reveal()` is empty. Table test including a board of three (D5) and the sixteenth mark. **Labels are `0`–`9` then `a b c e f g`** — `d` is reserved by `toInput` before a form sees it.
- [x] **T3 — Tab, and Enter split from space** (D13, D14). Enter becomes `InputFinish`; `Apply` treats it as `InputReveal` for every non-batch form, so 2.1 and 2.3 are untouched and their tests prove it. One row in `toInput`, one `play.Input` kind, and the board's mode flips. It belongs in `play` because it is about what is being ANSWERED, unlike the paging keys.
- [x] **T4 — `display.FooterRowAt`** (D10). `Paint` already computes the footer's origin; `liveScreen` records it and answers which footer row a viewport row is. The editor's screen answers "none", which is the whole of its involvement.
- [x] **T5 — the loop offers a click to the form first** (D11). Falls through to `playRegion` when the form declines. `#38`'s `TestPlayClickActsAndIsNotAnAnswer` must pass UNTOUCHED — every existing form declines.
- [x] **T6 — the footer carries the board** (D10, D15). The board's own rows, then the bar — the form first, which is what `formCell` reads an entry index back as. `fitFooter` is UNCHANGED; `boardsFor` asks `fitsABoard` and sends the words to 2.3 when the terminal is too short. The mode is NOT in the footer (R11).
- [x] **T7 — `ReviewEvent.Form`** (D4a). The field, `CaptureReview` writing it, and `Fold` ignoring it — it is telemetry, not assessment. **Operator-requested and the instrument the deferred remedies depend on.**
- [x] **T8 — form selection** (D4). `boardsFor` partitions today's keys at box ≥ 3 and packs the eligible ones sixteen at a time.
- [x] **T9 — the relearn line** (D10). As a board closes it writes ONE buffer line naming the words marked `No`, so the transcript keeps the outcome even though the grid was ephemeral.
- [x] **T10 — the bar counts words** (D8).
- [x] **T11 — the load claim, measured** (Done-when 7). **The measurement moved the claim — see R5.**
- [x] **T13 — pty conformance.** Done-when 14, which had a row and no task. `#37` records that pty rows need `-tags conformance` AND a real pty — *"in the review environment every one reports 'no pty available'"* — so they are run here or they are run nowhere.
- [x] **T12 — docs.** `cmd/define/README.md`, `atlas/define.md`'s forms section, the `--help` key table — **and the two in-tree forward references D7 falsifies**: `schedule/progress.go` says this issue extends the `Grade` seam with `GradeUnsure`, and `play/session.go` describes the mark as "firm".

---

## Done when

Every row's pin is a PREDICATE OVER BEHAVIOUR. **Every `red when` cell is EXECUTED as a mutation at the boundary and the result recorded per row** (`#38` BR-16: a row that survives its own mutation pins nothing).

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | a sitting of eligible words presents them as a grid | `TestASittingOfDueWordsIsABoard` | `boardsFor` sends them to 2.3 one at a time |
| 2 | every mark reaches the log as it happens | `TestEveryMarkOnABoardIsRecordedImmediately` — counting store, N marks, N events before the sitting ends | marks are batched to the end, losing them to Ctrl-C |
| 3 | **Enter takes the unmarked as `No`, and SPACE DOES NOTHING AT ALL** | `TestEnterSpendsABatchFormAndSpaceDoesNot`, `TestToInputSplitsEnterFromSpaceAndCarriesTab`, `TestEveryInputKindIsAnsweredForABatchForm` (R7) | the merged Enter/space kind spends the board, so a casual keystroke demotes sixteen words — or space is merely harmless rather than inert, and files a blank reveal while pronouncing an arbitrary cell |
| 4 | **Ctrl-C leaves unmarked words UNTOUCHED, and marked ones recorded** | `TestCtrlCCancelsABoardWithoutMovingUnmarkedWords` — fold the log after, boxes unchanged | a board writes its marks at the end instead of as they land |
| 5 | a `No` mark does not freeze the board | `TestABoardRunsThroughTheSession`, and `TestEveryInputKindIsAnsweredForABatchForm` asserts NO kind sets `Graded` | the miss-on-hidden branch sets `Graded` for a batch form |
| 6 | **the mouse-less path works, and `d` still drops** | `TestABoardIsMarkableByKeyAlone`, `TestDOnABoardIsNotACellLabel` | labels include `d`, or the keyboard path is missing and the board degrades to "everything is No" |
| 7 | a click marks on a board and plays everywhere else | `TestAClickOnABoardMarksIt`, and `#38`'s `TestPlayClickActsAndIsNotAnAnswer` UNCHANGED | the loop learns what a board is instead of asking |
| 8 | the session still learns nothing about which form is asking | `TestSessionIsFormAgnostic` unchanged, plus `TestTheSessionNamesNoForm` — a grep for every concrete form in `session.go`'s code, with a premise check that the capabilities are there | `Apply` type-switches instead of consulting `Batch` |
| 9 | **every review event records the form that asked** | `TestEveryRecordNamesItsForm` over all three forms and all three record-building paths, `TestAReviewEventNamesItsFormOnDisk`, `TestYAMLRoundTripsTheFormThatAsked` | the field is written for one form and defaulted for the others, which is worse than absent |
| 10 | **a board is never drawn clipped** | `TestAShortTerminalGetsMeaningChoiceNotAClippedBoard` and `TestFitsABoardCountsTheWholeLiveEdge`, both across a WIDTH axis (R8), and `TestPaintFitsTheTerminalAndParksTheCursor` UNCHANGED | `fitFooter` is given a floor, so `footerRows` exceeds `termRows - promptRows`, the terminal scrolls, and a click lands on the wrong word |
| 11 | the outcome survives the sitting | `TestABoardLeavesItsRelearnListInTheTranscript`, `TestABoardWithNothingToRelearnWritesNoLine`, `TestCtrlCOnABoardStillLeavesItsRelearnList` | the board is live edge and vanishes whole |
| 12 | the bar counts WORDS | `TestTheBarCountsWordsNotSlots` | `total` stays `len(s.Questions)` and a 20-word sitting reads "0 of 2" |
| 13 | the board is materially cheaper per word, **measured as what the learner READS** | `TestABoardCostsFarLessPerWordThanMeaningChoice` — one keystroke per word AND at least ten to one on transcript lines (R5) | the grid asks for more than one keystroke per word, or the board's rows reach the buffer and the reading cost collapses to 2.3's |
| 14 | the real terminal draws and clicks it | `TestPTYPlayBoardIsDrawnAndClickable`, extending `#41`'s and `#38`'s pty rows | it works in-process and not on a tty |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then on a real terminal with a deck holding mature words: `define --play`, confirm the grid appears for them and single questions for the fragile ones, sweep a board, and check the event log holds one event per mark.

**Close:** one boundary, one `sdlc close`, one publish.

## Open for the operator

All settled as of 2026-09-01. Recorded here so the list is visibly closed rather
than quietly dropped:

| question | answer |
|---|---|
| Enter vs Ctrl-C | Enter commits the unmarked as No; Ctrl-C cancels, and cancelling is free (D3) |
| the box threshold | the box stays the selector, lowered to ≥ 3 (D4) |
| `unsure` | deleted — "unsure means no" (D7) |
| telemetry | record the form on every review event (D4a) |
| the toggle's position | the footer, so `screen` grows no header (D6) — and later the prompt row, so a short terminal cannot drop it (R11) |
| a mouse-less terminal | labelled keys `0`–`9`, `a b c e f g`, printed beside each word (D5) |

One thing I chose rather than asked, flagged because it is the only number in
here without an argument behind it: **box ≥ 3** is a starting point, not a
derived threshold. D4a exists so it can be replaced by evidence.

## Revisions

> APPENDED, ONCE, AT THE END — and the placement is load-bearing rather than
> tidiness. `currentTruthOnly` (repo_guard_test.go) truncates a plan at its first
> `## Revisions`, so a revisions section inserted mid-document hides everything
> below it from every guard that reads "current truth only". This plan had two,
> the first above `## Done when`, and both `TestPlanTableStatusMatchesTheChangeWindow`
> and the new `TestPlanCitesTestsThatExist` were silently reading a truncated
> file. Found while mutation-checking the second guard, which is the only reason
> it was found at all.

### 2026-09-01 (R1) — the form draws its own live edge, and the panel is the last mark's gloss

D10's footer order names a **panel** and nothing in the plan said what it showed.
Put to the operator with the board on screen: *the last-marked word's
definition*. That is the feedback moment, in the place a definition sits on every
other form.

It carries two consequences the plan did not have.

- **`NewBoard` takes `[]Cell`, not `[]string`** — a word and a ONE-LINE gloss.
  Finished content, exactly as `Choice` takes rendered `Option`s: extracting a
  sense needs the dictionary and this package imports nothing. One line rather
  than the whole entry, because a board is offered only when the terminal can
  draw it whole and a twenty-row panel would mean a board nobody's terminal fits.
- **`Prompt()` returns the WHOLE live edge** — grid, blank, toggle, panel — and
  `boardFooter` is one line that appends the bar. The first cut had the loop
  assembling it from `Mode()` and a gloss, which made the board's appearance a
  thing two files agree about, on the surface where disagreeing marks the wrong
  word. A form owns how it looks; `Choice` owns its option layout for the same
  reason. `Moded` therefore needs only `Toggle()`, and `Rows()` counts the whole
  edge rather than the grid — which is the number `fitsABoard` is actually about.

The keys line stays ABOVE the grid (operator, 2026-09-01), where `Paint` puts the
prompt: it reads as a header for the grid, and it is the one row `Paint` protects
from clipping.

### 2026-09-01 (R2) — three table rows that described work that did not happen

Caught by `TestPlanTableStatusMatchesTheChangeWindow`, which is the mechanism
that exists because hand-sweeping keeps failing — and it failed here in exactly
the way the issue's own Revisions section had already written down.

- **`fitFooter` — `modified` → `unchanged`.** D15 retired D10's floor in the
  prose ("`fitFooter` is therefore UNCHANGED") and the Pure-entities table kept
  the old claim. The plan's own sweep missed the plan's own table.
- **`livePrompt` — `modified` → `unchanged`.** D12 predicted a change that turned
  out unnecessary: `s.Graded` is never true for a board, so the graded prompt
  cannot fire and the function needs no guard. What actually changed is
  `gradePrompt`, which now asks `reservedKeys` — because `d` is refused on a
  board (D12) and the prompt was offering it anyway.
- **The rows that were missing entirely** are listed above: `Cell`, `Grid`,
  `Moded`, `formCell`, `boardFooter`, `fitsABoard`, `reservedKeys`. A plan that
  names four new symbols and ships eleven is as wrong as one whose status column
  lies.

### 2026-09-01 (R3) — `Form()` is on the `Question` interface, not an optional capability

Cited by the Core-concepts table and never written down, which is its own small
instance of the class R6 is about.

`#40` adds four OPTIONAL capabilities — `Batch`, `Moded`, `Grid`, beside the
existing `Missed` and `SelfRated` — so the obvious shape for D4a's telemetry was
a sixth. It is not, and the reason is the plan's own red-when for Done-when 9:
*"the field is written for one form and defaulted for the others, which is worse
than absent"*. An optional capability defaults to the empty string, and an event
that says "some form" looks like data. On the interface, a new form cannot
compile without naming itself — the same argument that put `Keys()` there.

### 2026-09-01 (R4) — the bar's total is set in the loop, not in `sittingFigures`

The Pure-entities table named `sittingFigures` as the thing that changes for D8.
It does not: `sittingFigures` is a struct of numbers and never knew what a
question was. What changes is `fig.total = len(s.Questions)` in the loop's
`refresh`, which is now `sittingWords(s.Questions)`. Corrected, and the new
helpers T8-T10 shipped are listed in the table above.

### 2026-09-01 (R5) — the keystroke proxy was the wrong instrument, and the measurement said so

Done-when 13 asked for "materially fewer KEYSTROKES than form 2.3". Measured over
eight words:

| | keystrokes/word | transcript lines/word |
|---|---|---|
| board, one mode throughout | 1.00 | 0.4 |
| board, two of eight missed | 1.12 | 0.6 |
| form 2.3, none missed | 1.00 | **8.8** |
| form 2.3, two of eight missed | 1.25 | **23.1** |

**Keystrokes are a wash.** Both forms cost one per word in the good case; 2.3
costs a second on every miss — the definition goes up and any key moves on — and
a board costs one per mode switch. Nothing there is "material".

**What is material is how much the learner has to READ**, by a factor of twenty
to forty. Form 2.3 writes a whole rendered entry per word into the transcript;
a board writes one line for the entire sweep. And that is the Spec's own claim,
which was never a typing claim: *"a hundred mature words swept in a grid cost
what ten fragile ones cost in multiple choice"* is a ten-to-one ratio, and it is
reading cost, for a form whose whole argument is that a large deck becomes
unaffordable to maintain.

So the row now pins both — the keystroke floor its own red-when names, and the
reading ratio that carries the claim. The issue's Done-when is swept to match.

**The rule this leaves: a Done-when that names a PROXY is a hypothesis, and
measuring it can falsify the proxy rather than the claim.** The right response is
to say which measurement the claim actually rests on, not to find a framing under
which the proxy passes.

### 2026-09-01 (R6) — the Done-when table cited four tests that were never written

Rows 3, 5 and 9 named `TestEnterCommitsTheUnmarkedAsNo`,
`TestSpaceDoesNotCommitABoard`, `TestANoMarkDoesNotEndTheBoard` and
`TestAReviewEventNamesItsForm`. None exists; the shipped pins are
`TestEnterSpendsABatchFormAndSpaceDoesNot`, `TestABoardRunsThroughTheSession` and
`TestEveryRecordNamesItsForm`. Corrected above.

**The class, not the instance.** `TestPlanTableStatusMatchesTheChangeWindow`
exists because hand-sweeping a plan's tables does not hold — and it reads only
the `name | file.go | status` rows, so the `pinned by` column was unguarded and a
plan could cite a test that does not exist. Extended:
`TestPlanCitesTestsThatExist` resolves every backticked `Test*` identifier in an
active plan to a `func Test…(` in the tree.

### 2026-09-01 (R7) — D14's "`InputReveal` does nothing on a batch form" was not implemented

Space set `Revealed` and returned an `OutcomeReveal` carrying `q.Word()` — which
on a grid is the last-marked cell, or the first cell before any mark — so the
loop filed a blank reveal in the append-only buffer and played the pronunciation
of a word nobody asked about. Space is the natural key to press: it reveals on
both other forms and the board's own prompt does not mention it.

**D12's enumeration was the defect, not the missing guard.** It wrote down four
`Apply` paths that must consult `Batch`; the real shape is `InputKind × Batch`
and `InputReveal` was a fifth cell. An enumeration that lives in prose fails
silently the next time the set grows. `numInputKinds` is now a sentinel and
`TestEveryInputKindIsAnsweredForABatchForm` ranges over it, so a kind added later
arrives with no expectation and fails.

### 2026-09-01 (R8) — D15's fit condition is MEASURED, not constant

`boardChromeRows` charged the keys prompt a hard-coded one row. That line is 76
columns and a board is offered from 20, so below 76 the live edge was
under-budgeted and `fitFooter` dropped rows from the end: the bar under 76
columns, the panel at 38 or less, and the TOGGLE at 25 or less — the one owner of
which mark is live, on a surface where every mark is irreversible.

`fitsABoard(termRows, boardRows, promptRows)` now takes the measured height, and
`boardFits` computes it from the same `gradePrompt` the loop will draw. The bar
keeps a one-row MINIMUM rather than a measurement, and the asymmetry is the
point: `fitFooter` drops from the END and the board's own rows come first, so a
bar that turns out to need three rows is dropped instead of costing the board
anything. The prompt sits above the footer and takes its share off the top, so
its real height has to be charged.

Both fit tests gained a width axis.

### 2026-09-01 (R9) — the board is laid out for the width it is DRAWN at, not the one it was chosen at

D15 said a resize below the board's height *"leaves the current board drawn as it
was — its rows are already budgeted"*, and `Board`'s own field comment said the
width *"cannot change"*. Both were wrong, and the boundary review measured the
consequence: a board laid out for eighty columns has 74-column rows, so at forty
the terminal wraps each into two. A footer entry stops being one physical row —
`FooterRowAt` correctly answers the same entry for both — and `formCell` handed
the raw physical column to `CellAt`, so column 4 of a continuation row marked
cell 0 while the text at that column belonged to another word. **The mark is
permanent.** Painting the same board at 24x12 also dropped the toggle, the panel
and the bar, which is R8's harm through a door `fitsABoard` cannot see.

**The rule, which is bigger than the bug:** every quantity the board's fit and
its click map depend on must be read from the terminal AS IT IS at draw and click
time, never fixed at selection time. Four quantities, all now closed:

| quantity | how it is read |
|---|---|
| prompt height | measured at the current width (R8) |
| board layout width | `Board.Resize(cols)`, called from the loop's resize case with `sz.cols` — the width the SCREEN paints at, not `opt.width`, which is a wrap policy that answers 0 on a narrow terminal |
| the fit after a resize | the board relays out, so its rows keep fitting; if the terminal is then too short, `fitFooter` drops trailing entries, which are grid rows that are neither drawn nor clickable. No scroll, no wrong-word click, and the marks already landed are already in the log |
| a wrapped entry's columns | `FooterRowAt` now reports WHICH of an entry's physical rows was hit, and `formCell` refuses any but the first |

The marks SURVIVE a relayout, which is why it is a relayout and not a new board:
those answers are already in the event log and cannot be retracted. Only the
geometry moves.

**Two defences, deliberately.** The relayout closes it at the root; the
continuation-row refusal closes it at the seam, for any future multi-row footer
entry and for the day someone forgets to pass a resize on. A click is
irreversible, and "should never happen" is not something to bet one on.

### 2026-09-01 (R10) — a guard reading a filtered view must FAIL when the filter swallowed its subject

R6 diagnosed the mechanism — `currentTruthOnly` truncates at the FIRST
`## Revisions`, this plan had grown two, and every guard reading it saw a file
that stopped before the section it existed to check — and then fixed only this
plan's layout. The review's second round called that correctly: the shape can
come back tomorrow, in this plan or any other, and four of the eight guards
reading `currentTruthOnly` end in `checked == 0 → t.Skip`, so it comes back as
SUCCESS.

`currentTruthOnly` now takes the test and FAILS when a record section is not the
last top-level section, naming the section it swallowed. One place, all eight
call sites. And `TestPlanCitesTestsThatExist` asserts its own premise: a plan
with a `## Done when` table that cites no test at all is an error, not a skip.

**The rule: a guard that reports success about a file it never saw is worse than
no guard.** It is the same shape as the estimate gate refusing rather than
guessing — evidence you cannot produce must be reported as absent, never as
satisfied.

### 2026-09-01 (R11) — the mode moves to the prompt row, because that is the row that survives

D6 put the toggle in the footer, on the reasoning that the live edge is where
things that change belong. R9's resize measured what that costs: `fitFooter`
drops footer rows from the END, so a terminal too short after a narrowing dropped
the panel and then the TOGGLE — leaving a board on screen with no statement of
what the next click would mean, while every mark is irreversible.

`Paint` clips the PROMPT last, and only when the prompt alone exceeds the
terminal. So the mode lives on `Keys()` now: *"marking [yes] no, Tab switches,
click or key marks, Enter ends"*. Still ONE owner — the toggle row is gone, not
duplicated — and what changed is which row it is, chosen for which row survives.

**And D15's "a board is never in a footer that has to drop anything" was measured
wrong.** It holds at SELECTION, which is what `boardsFor` enforces; a resize
afterwards is a shape nobody chose. The board is NOT re-selected then — its marks
are already in the log, so sending those words to 2.3 would mean re-asking words
already answered. What the order of the footer buys instead is that the losses
are harmless in sequence: the bar (a figure), the panel (cosmetic), then grid
rows — which are conspicuously absent and, never having been painted, are not
clickable either. The one thing that must not go is now not in the footer at all.

### 2026-09-01 (R12) — the resize test asserted over an event set it never produced

`TestANarrowingResizeKeepsTheBoardsClickMapHonest` ranged over
`reviewEvents(...)` with no count check, and the click never landed — so it
passed with `formCell` stubbed to return false. The rule it broke is one this
issue's own Log already records for T13: **read the click's row and column off
the PAINT, never compute them.** The driver derived a row by splitting the
emitted frame, which is logical writes, while `FooterRowAt` works in physical
rows — and the loop repaints after a resize, so the frame that was read and the
frame the click resolved against were different ones.

The row now comes from the screen's own paint record (`FooterRowAt`, which is
what production asks) and the column from the board's current layout. The premise
is asserted before the claim: **a test whose subject is an event must assert the
event happened.**

**And it HUNG rather than failed under its own mutation**, which is the same
defect wearing a second coat. `waitFor`'s `t.Fatal` runs on the helper goroutine;
`FailNow` there is a `Goexit`, so the goroutine died without closing the key
channel and `playSession` blocked forever. The driver closes with `defer` now and
gives up quietly, leaving the main body to fail. A test that hangs on the defect
certifies about as much as one that passes on it, and takes longer to say so.

### 2026-09-01 (R13) — the filter's OTHER discarding rule got the same premise assertion

R10 gave `currentTruthOnly`'s truncation rule a premise assertion and left its
closed-section rule discarding silently. Splitting on `\n### ` means a closed
section runs to the next `### `, which can swallow a following top-level `## `
section whole — the same failure through the other door. No live instance in the
tree, which is why this is the rule written down rather than a defect fixed: **a
filter with two ways to discard needs the assertion on both, or the next one
silently gets none.** Verified by planting a closed section above
`## Core concepts`: it now fails, naming what was swallowed.

### 2026-09-01 (R14) — the premise assertion found a live instance the review believed did not exist, and a guard blind for months

R13 was written as "the rule applied to every rule the filter has", with the
review noting no live instance in the tree. There was one, and it was worse than
a plan's layout: **`atlas/repo-guards.md` documents `currentTruthOnly` and
therefore spells `**closed:**` in its prose**, so `strings.Contains` matched it
and the guard inventory — the whole top of the page — had been discarded from
every guard reading current truth.

Two of them were blind over it. Unblinded, `TestNoArtifactNamesARetiredSymbol`
immediately found FIVE retired identifiers on that page, the tenth artifact in
the nine-recurrence family the page itself is about.

Fixed at both ends. `closedSection` matches the marker at the START of a line,
because a marker is written by whatever wrote the record and prose ABOUT a marker
is not one. And the page now counts its examples rather than naming them — this
guard's own rule applied to its own documentation, since names on a page are
exactly what rots.

**The lesson is about the ORDER of the two changes.** The assertion was added on
the reviewer's word that it had no present instance; it took thirty seconds to
write and found a months-old hole immediately. A filter that discards silently
cannot be audited by reading it — only by making it speak.
