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

**D1 — TWO OF THE SPEC'S REQUIREMENTS ARE ALREADY IN THE TREE, and finding that is most of what this design bought.**

The Spec asks for a Revision to `#39` saying that `+2` should come from *"a correct answer in form 2.3 with no reveal first"*. That is `unaidedNow` (`play/session.go:317`), which already computes exactly `v == Correct && !s.Revealed && !IsSelfRated()`, and `schedule.Answer` already promotes `GradeUnaided` by two. **`#39` needs no revision; it shipped this.**

And the Spec's *"`firm` earns `+1` and not `+2`"* follows for free: a `Board` that implements `SelfRated` cannot produce `GradeUnaided`, so `firm` maps to `GradeCorrect` and the mechanism that would have over-promoted it is structurally unreachable. **No new grade, no new schedule arithmetic, no exception in `Answer`.**

One nuance the Spec's table does not state and the code does: `GradeCorrect` is `+2` when `Box < MaxBox` — the express lane back after a lapse. So `firm` on a mature word that has LAPSED climbs two. That is correct and consistent — the Spec says *"the same as a correct answer"*, and this is what a correct answer does — but it must be written down, because the table reads `box + 1` flatly and a future reader will otherwise call it a bug.

**D2 — the session discovers that a form holds many words by ASKING, which is the third instance of a pattern this package already has twice.**

`advance` moves to the next question on every graded answer (`session.go:263`). A board must stay current while sixteen marks land. The session must not learn what a board is — `#6`'s Done-when and `TestSessionIsFormAgnostic` forbid it — so the question is put to the FORM:

```go
// Batch is implemented by forms that hold more than one word.
type Batch interface {
	// Spent reports whether every word this form holds has been answered.
	Spent() bool
}
```

`advance` consults it exactly as `missedAxis` consults `Missed` and `unaidedNow` consults `SelfRated`: a capability named, never a form. A form that does not implement it is spent after one answer, which is every existing form and is the right default.

**D3 — THE CURSOR ADVANCES ITSELF, and that is what keeps `play.Input` from growing.**

The obvious design gives the board arrow keys. It must not: arrows arrive as `Key` KINDS, `toInput` maps only runes and Enter/space, and widening `play.Input` with a movement kind would put a display concept inside the pure package — the same reasoning that made `#41` D6 intercept paging in the loop instead.

So there is no movement. The cursor starts at the first word and steps left-to-right, top-to-bottom after each mark. Sixteen words, sixteen keystrokes, in order — which is exactly what the Spec asks for (*"one keystroke each"*) and is also the faster interaction: a learner sweeping a grid does not want to steer.

**Consequence, stated rather than discovered:** there is no going back to change a mark. That is acceptable for triage — the marks are conservative in the direction that matters, and `unsure` exists precisely for "I am not sure", which is the answer a mis-key would want anyway.

**D4 — `unsure` needs somewhere to live, and the Spec's three requirements for it cannot all be met by a box change.**

The Spec says `unsure` leaves the box unchanged, re-asks the word sooner, AND sends it to form 2.3 next time. The first two come free from recording nothing: `Fold` leaves `LastReviewed` where it was, so the word is still due and comes back next sitting. The third does not — form selection is by BOX (D6), the box is unchanged, so the word would meet the board again forever.

**So `unsure` is recorded as its own event kind**, `store.EventUnsure`, and form selection reads it: a word whose most recent review event is an `unsure` is asked through 2.3 whatever its box. It is an event rather than a `Progress` field because `progress.go` states the rule for this whole package — *"DERIVED, never stored… storing counters alongside would create a second source of truth that drifts"* — and because the log is append-only, so the signal expires naturally when a real answer lands on top of it.

**`schedule.Fold` must IGNORE it**, exactly as it ignores lookups and questions: an unsure is activity, not assessment, and folding it as a miss would demote the word the Spec says must not move.

**D5 — the board is SIXTEEN WORDS OR FEWER, and a short board is normal.**

A sitting rarely divides by sixteen. The last board takes what is left; a board of three is a board. The alternative — hold words back until sixteen accumulate — would silently drop words from a sitting the schedule asked for, which is the failure `emptyQueueReason` exists to make impossible elsewhere.

**D6 — the threshold is `MasteredBox`, not 8, and this reverses the Spec.**

The Spec's table says box ≥ 8. `MasteredBox` is 9 and is already defined as *"where a word stops being highlighted… reached on day 108 after nine correct recalls, the last of which came after a 42-day gap"*. Two thresholds one apart, both meaning "this word no longer needs real work", is two owners of one judgement — the exact shape `lessons.md` records as drifting.

`Mastered(p)` is already exported and already has two consumers (`#6`'s highlighting, `#8`'s stats). The board is the third, and it is the one that makes the label mean something operationally rather than cosmetically.

**If the operator wants 8 specifically, the honest move is to change `MasteredBox` to 8** and let all three consumers move together — not to add a second constant. Flagged for the operator at plan review; the plan proceeds on `Mastered`.

**D7 — the bar counts WORDS, not slots.**

`fig.total = len(s.Questions)` is the slot count, and a board is one slot holding up to sixteen words. Left alone, a sitting of 20 mature words would read "0 of 2". The budget itself is unaffected — `schedule.Queue` returns `budget` KEYS and packing them into boards does not change how many words are asked about — so this is a display fix, but it is the number the Spec's own load argument is about.

**D8 — NOT in scope:** `/board` as a manual override. The Spec asks for it and it is one command row, but it is the escape hatch rather than the feature, and it cannot be designed until the scheduler path exists to escape from. It gets its own task (T8) and is the first thing to cut if the boundary runs long.

---

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

1. **D6 — the threshold.** The Spec says box ≥ 8; `MasteredBox` is 9 and already means "no longer needs real work", with two consumers. The plan uses `Mastered` rather than adding a second constant one apart. If 8 is the number you want, the honest change is `MasteredBox = 9 → 8`, moving highlighting and stats with it.
2. **D4 — `unsure` becomes an event kind.** It is the only way to meet all three of the Spec's requirements for it, and it adds a row to the store's vocabulary. Cheaper alternative if you would rather not: drop *"next time through form 2.3"* and let `unsure` mean only "box unchanged, ask again" — one less concept, and the word meets the board again.
3. **The Spec's `#39` revision is not needed** — `unaidedNow` already does what it asks for. Confirm you agree before I close that thread.
