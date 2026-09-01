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

**D4 — NO BOX THRESHOLD. The board draws from all of today's words.**

This reverses the Spec, deliberately. Operator, 2026-09-01: *"words should come
from all today's practice words. this is a weaker form of recall, but faster… I
feel we should not have those two limits at start, and see how things work."*

The Spec bound the form to the box on two arguments. The **load** argument
survives but does not need a threshold: a box-8 word costs `1/42 ≈ 0.024`
reviews/day against a box-0 word's `1.0`, so the tail is where the volume is
whether or not a rule says so. The **correctness** argument — a grid is
self-report without retrieval, and the illusion of knowing runs that way — is the
one being consciously accepted for now.

**And the risk is not shaped the way the Spec assumed.** Extra days before a
wrongly-promoted word returns:

| from box | 0→1 | 2→3 | 4→5 | 6→7 | 8→9 |
|---|---|---|---|---|---|
| days added | **0** | 2 | 4 | 10 | 26 |

The ladder's first two rungs are both one day — `box.go` calls that duplicate
*"the most valuable rung"* — so a brand-new word marked `Yes` in error **comes
back tomorrow regardless**. The error is free at the bottom. Absolute delay grows
with the box while pedagogical damage is worst where forgetting is steepest,
which is the bottom; the two run opposite and cross around boxes 5–7. **A hard
cut at 8 aimed at neither end of that.**

The operator's two candidate remedies — the board promotes more slowly, or it is
used less often — are both DEFERRED, to be chosen later from evidence and
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

**D5 — CLICKS, not a cursor.** The learner clicks a word to mark it with the
active mode; Tab swaps the mode between Yes and No. This is `#38`'s affordance
finding its second consumer, and it deletes the first draft's D3 entirely — no
cursor, no auto-advance, and no pressure on `play.Input` to grow a movement kind.

**D6 — the toggle and the feedback panel are the LIVE EDGE, not buffer lines.**
The buffer is append-only, which is what makes a click's coordinates exact
(`#30` D1). Anything that changes in place is the prompt or the footer, and
`Paint` already takes a multi-row footer that gives up whole rows before the
prompt does. So the definition panel is the footer and costs nothing.

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

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Board` | `cmd/define/play/board.go` | new | PURE — form 2.5: N words, a cursor, one mark each. Implements `Question`, `SelfRated` and `Batch` |
| `Mark` | `cmd/define/play/board.go` | new | PURE — `firm`/`unsure`/`no idea`, and the `Verdict` each maps to |
| `Batch` | `cmd/define/play/session.go` | new | PURE — the capability "I hold more than one word", asked by `advance`. Third of its kind beside `Missed` and `SelfRated` |
| `advance` | `cmd/define/play/session.go` | modified | PURE — moves on only when the current form is `Spent()` |
| `boardsFor` | `cmd/define/play_loop.go` | new | PURE — partitions the day's keys into boards and single questions by `Mastered` |
| `sittingFigures` | `cmd/define/playbar.go` | modified | PURE — `total` becomes the WORD count (D7) |

- **`Board`** — sixteen mature words, marked one keystroke each.
  - **Relationships:** 1:N with words (N ≤ 16); occupies ONE `Session.Questions` slot.
  - **DRY rationale:** `Word()` returns the CURSOR's word, so `advance`'s existing `Outcome{Word: q.Word()}` records the right word with no change at the call site. The board does not get its own record path.
  - **Future extensions:** `#12`'s cloze and `#13`'s free sentence are single-word forms and need none of this; a future "N at a time" form is a second `Batch` implementer and nothing else moves.

- **`Batch`** — the session's question, not the form's announcement.
  - **DRY rationale:** third instance of the optional-capability pattern. `Missed` (`session.go:287`) and `SelfRated` (`:311`) both exist because *"a type switch on `*Choice` would be the thing Done-when 7 forbids"*; this is the same sentence about `*Board`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | the store — packs mature keys into boards (D6) |
| `EventUnsure` | `cmd/define/store/event.go` | new | the event log — `unsure`'s durable home (D4) |
| `CaptureReview` | `cmd/define/capture.go` | modified | the store — writes `EventUnsure` for the third mark |
| `Fold` | `cmd/define/schedule/progress.go` | modified | — ignores `EventUnsure`, as it ignores lookups |

**ARCH-MOCK.** No new external dependency. `Board` is pure and unit-tested with no IO; the loop's tests use the recorder `#30` built and `#41` extended; the pty suite covers the real terminal.

**ARCH-CONSTRAINTS.** The interaction path is a keystroke and a redraw, inherited from `#41` unchanged: a frame is O(visible rows), throttled at 16ms. A board's `Prompt()` is O(16) string building per frame — the same order as form 2.3's four options. `boardsFor` is O(deck) once per sitting, beside the walk `#41` already does. **The load claim this issue exists for is the thing to measure**, and Done-when 7 makes it a test rather than an assertion: N mature words through the board must cost materially fewer keystrokes than N through 2.3.

**ARCH-PURE.** `Board` and `Mark` import nothing — `play`'s purity guard (`cmd/define/puretest`) enforces it mechanically, and a board that reached for a width or a terminal would fail the build.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [ ] **T1 — `Batch`, and `advance` asks it** (D2). The interface, plus `spent(q)` beside `missedAxis(q)` and `unaidedNow(...)`. Existing forms are unaffected because they do not implement it: `TestSessionIsFormAgnostic` and the whole `play` suite pass untouched, which is what proves the seam widened rather than branched.
- [ ] **T2 — `Board` and `Mark`** (D3, D5). `Prompt()` renders the grid with the cursor; `Grade` marks and steps; `Spent()`; `IsSelfRated() → true`; `Reveal()` is the empty string, because triage shows nothing. Table test including a board of three (D5) and the sixteenth mark.
- [ ] **T3 — `EventUnsure`** (D4). The kind, `CaptureReview` writing it, and `Fold` IGNORING it. Counting test: a folded log containing an unsure leaves the word's box exactly where it was.
- [ ] **T4 — form selection** (D6). `boardsFor` partitions the day's keys by `Mastered`, packs the mature ones sixteen at a time, and sends a word with a recent `unsure` to 2.3 whatever its box. Test over a deck spanning both sides of the threshold.
- [ ] **T5 — the loop draws it.** Nothing should be needed here: a board is a `Question`, `#41` gave `--play` coordinates, and `show()` writes `Prompt()` once per question. **If this task needs code, that is the finding** — it means `Batch` leaked into the loop, and the fix is in T1 rather than here.
- [ ] **T6 — the bar counts words** (D7). `sittingFigures.total` from the word count; `done` from marks and answers together.
- [ ] **T7 — the load claim, measured.** Done-when 7: N mature words through the board versus through 2.3, counted in keystrokes.
- [ ] **T8 — `/board`** (D8). The manual override, and the first thing to cut.
- [ ] **T9 — docs.** `cmd/define/README.md`'s review-loop section, `atlas/define.md`'s forms section, and the `--help` key table.

---

## Done when

Every row's pin is a PREDICATE OVER BEHAVIOUR — a named test or a grep for a property — never "file X is unchanged". **Every `red when` cell is EXECUTED as a mutation at the boundary**, and the result recorded per row (`#38` BR-16: a row that survives its own mutation pins nothing).

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | a sitting of mature words presents them as a grid, sixteen at a time | `TestASittingOfMatureWordsIsABoard` | `boardsFor` sends them to 2.3 one at a time |
| 2 | every word in the grid can be marked, and each mark reaches the log as it happens | `TestEveryMarkOnABoardIsRecordedImmediately` — a counting store, N marks, N events before the sitting ends | marks are batched to the end, losing them to Ctrl-C |
| 3 | the scheduler chooses the form, not the learner | `TestTheBoxChoosesTheForm` over a deck spanning the threshold | the form is a flag or a mood |
| 4 | `unsure` leaves the box where it was AND sends the word to 2.3 next time | `TestUnsureDoesNotMoveTheBoxAndForcesARealTest` | it is folded as a miss, or the word meets the board again |
| 5 | `firm` can never earn the two-rung promotion | `TestABoardIsSelfRated` — and `TestSelfRatedFormsNeverEarnUnaided`, which already exists and must not need changing | `Board` stops implementing `SelfRated` |
| 6 | the session still learns nothing about which form is asking | `TestSessionIsFormAgnostic`, unchanged, plus a grep for `*Board` in `session.go` finding nothing | `advance` type-switches instead of asking `Batch` |
| 7 | the board is materially cheaper per word | `TestABoardCostsFewerKeystrokesThanMeaningChoice` | the grid asks for a keystroke per word plus navigation |
| 8 | the bar counts WORDS | `TestTheBarCountsWordsNotSlots` | `total` stays `len(s.Questions)` and a 20-word sitting reads "0 of 2" |
| 9 | a short board is normal | `TestALastBoardTakesWhatIsLeft` | words are held back until sixteen accumulate |
| 10 | the real terminal draws it | a pty row extending `#41`'s | it works in-process and not on a tty |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then on a real terminal with a deck holding mature words: `define --play`, confirm the grid appears for them and single questions for the fragile ones, sweep a board, and check the event log holds one event per mark.

**Close:** one boundary, one `sdlc close`, one publish.

## Open for the operator

Three of the original five are settled above (D3 Enter/Ctrl-C, D4 the threshold,
D7 `unsure`). Two remain, and both are structural:

1. **The toggle sits at the TOP of your sketch, and the live edge is at the
   bottom.** The buffer is append-only, so "Do you remember? [Yes] No" cannot be a
   buffer line — it changes as you Tab. Either it moves down beside the keys and
   the bar, or `screen` grows a HEADER: rows above the buffer, which nothing has
   needed until now and which every frame-budget calculation would have to learn
   about. The header is the truer rendering of your sketch and the larger change.

2. **A terminal reporting no mouse has no way to mark a word.** Enter still
   works, so the board would silently degrade to "everything is No" — wrong
   rather than merely limited, and `#38`'s own pty rows exist because a
   mouse-less terminal must keep working. Cheapest keyboard path is
   `1`–`9`/`a`–`g` as the sixteen cells, which also gives the mouse users a
   faster option. Or is mouse-only acceptable, with the board simply not offered
   where the mouse is absent?

3. **And one I would like your instinct on, since D4 removed the box as the
   selector:** if the schedule no longer picks the form, what does? A flag or
   `/board` returns the choice to the learner, which the Spec argued against. The
   alternative your mock hints at — **sweep the board first over today's words,
   and the ones marked No become the sitting's real work in form 2.3** — keeps
   the speed, makes "No" the route to a real test rather than only a demotion,
   and needs no selector at all.
