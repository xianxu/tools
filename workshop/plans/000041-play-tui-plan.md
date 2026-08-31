# Play-Mode Frames Implementation Plan (`#41`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `--play` draws whole frames through the same `screen`/`display` seam the editor uses, so it owns coordinates — which buys a pinned status bar, paging over a long reveal, and the ground `#40`'s grid needs.

**Architecture:** No new abstraction. `#30` built `screen` (a pure line buffer plus viewport) and `liveScreen` (the only part that touches a terminal) for the editor loop, and `atlas/define.md` already records `--play`'s divergence from it as predicted. This issue makes `--play` the second consumer: it builds a `console` exactly as `replRaw` does, writes the question into the buffer once per question rather than appending it per keystroke, and passes the grading keys as the prompt and the status bar as the footer.

**Tech Stack:** Go 1.26. No new dependency.

---

## Decisions

**D1 — `--play` reuses `console`, and adds nothing.** `replraw.go` already builds `console{view, resizes, finish, stdout, stderr}` from `newLiveScreen(stdout, terminalRows, terminalCols)`. That construction is what `--play` adopts. If anything in it turns out not to fit, the honest move is to widen the shared seam rather than grow a parallel one — a second way to draw is the thing this issue exists to remove, not to add.

**D2 — `Paint`'s `menu` parameter is RENAMED `footer`, and that is not cosmetic.** The concept `Paint` actually implements is *"rows below the prompt, which give up whole rows before the prompt does"*. The editor's use of that is a command menu; play's is a status bar; calling the parameter `menu` would make play's call site read as something it is not, and would invite a future third consumer to add a third parameter for its own bottom rows. One name, one budget, two consumers.

**D3a — "PINNED TO THE BOTTOM" IS NOT FREE: `Paint` does not pad a short buffer.** It writes the visible frame, then the prompt, then the footer (`screen.go:400-411`) — so with a five-line question on a forty-row terminal the bar sits at row seven with thirty-three blank rows beneath it. For the editor that is correct behaviour, and deliberately so: a REPL prompt belongs directly under the last output, not stranded at the screen's edge.

So the buffer region must be able to occupy its FULL height, and that is a property of the screen rather than of a paint call. Two named constructors instead of a boolean at a call site that already takes two integers:

```go
newLiveScreen(tty, rows, cols)    // editor: the footer follows the content
newPinnedScreen(tty, rows, cols)  // --play: the buffer fills, the footer sits at the bottom
```

The padding is blank rows emitted at PAINT time, never lines appended to the buffer — the transcript and the click map must not gain rows that exist only because the terminal is tall. Done-when 11 pins the distinction by asserting the buffer's line count is unchanged by a resize.

**D3 — the grading keys are the PROMPT and the status bar is the FOOTER, which gets the sacrifice order right for free.** `Paint` documents its own order of value: *"The prompt is the line you are typing and survives first… The menu is a dropdown and gives up whole rows next. The buffer is scrollable, so it takes what is left."* For a review sitting that ordering is already correct — a learner who cannot see the grading keys cannot answer at all, while a learner who cannot see their daily load loses nothing this minute. So no new layout logic, and the visual order (buffer, keys, bar) is the one wanted.

**D4 — the question goes into the BUFFER once; the keys and the bar are the live edge.** This is the change of model, and the place a naive port breaks. `draw()` today writes the prompt, the reveal and the keys on EVERY call, which is correct for a scrolling terminal and would, against a line buffer, append a copy of the question per keystroke.

So the loop tracks which question it has already written and writes on transition:

- the current question's `Prompt()` is written when the index changes,
- its `Reveal()` is written when `OutcomeReveal` is performed — which the loop already handles, since that is where the pronunciation plays,
- everything else is `view.Draw(keys, footer)` on every frame.

The state is one `int` and it lives in the loop, which is where "perform the outcomes" already lives.

**D5 — the transcript surviving exit is REUSED, not re-decided.** `#30` M1.2b is *"Paint draws a whole frame, and the transcript survives the alt screen"*, and `handBack`/`onceHandBack` in `replraw.go` own the order. A learner who quits a sitting should still see the words they just reviewed; that is more true of a review than of an editor session. Whatever the editor does here, `--play` does the same call.

**D5a — ADOPTING FRAMES DELETES THE PLAYBACK DANCE, and that is the resolution of the only Critical this design had.** Today every reveal calls `raw.sess.restore()`, plays the pronunciation in cooked mode, and calls `enterRaw` again (`play_loop.go:177-199`). `enterAlt` is an OPT-IN call on `rawSession` and `restore` leaves the alternate screen, while `enterRaw` returns a fresh session with `alt` false — so a frame-drawing `--play` would lose the alternate screen on its first reveal and never re-enter it. Everything after that would paint into the normal screen over the user's scrollback.

The fix is not to re-enter it. **Playback stops leaving raw mode at all.** The dance exists because *"the indicator and any warning are written for a human to read"* — that is, they need newline translation — and under the frame model the indicator is a frame write like any other. `screen.Write` already honours the `\r\x1b[K` erase-line sequence by dropping the OPEN line, which is exactly how `#30` M1.2b kept `♫ playing 3×` out of the exit transcript.

So the whole `restore`/`enterRaw` pair goes, along with its error branch — the *"lost the terminal after playback"* path that exits 1. That branch is not being weakened: it exists only because the terminal was handed back and might not come back, and nothing is handed back any more. A Critical resolved by deleting code rather than working around it is the strongest sign the seam was the right one to adopt.

**D6 — paging is inherited, and it is handled BEFORE `toInput`, never inside it.** Form 2.3's reveal shows the whole rendered entry, which for `run` or `bank` is several screenfuls, and today the question scrolls off the top. `#30` already decodes the wheel and PageUp/PageDown, and `screen` already has the viewport.

The first draft of this plan routed those keys through `toInput`, which was wrong in the way that matters: `toInput` converts a `main.Key` into a `play.Input`, and a viewport is not something the session has any business knowing. `play` is mechanically guarded pure and its whole design is that *"main owns the terminal and knows Ctrl-C is 0x03; play must not"* — a `play.Input` kind for "page up" would put a display concept inside the pure package and every future form would inherit it.

So the loop intercepts the paging keys and calls `view.Page`/`view.Scroll` directly, exactly as the editor does, and only what is left becomes a `play.Input`. The session never learns a viewport exists.

**D7 — the bar's numbers touch the disk ONCE PER SITTING, and the per-answer update is in memory.** The first draft said "once per answer, two disk reads, microseconds" and both halves were wrong. `Deck()` reads ONE FILE PER WORD and `Events(anyTime)` reads ONE FILE PER DAY OF HISTORY, so recomputing per answer is `O(deck files + log days)` — for a 5,000-word deck with two years of log that is roughly 5,700 file reads for every question answered, on a path a person is waiting on.

So the loop reads them ONCE, at the start of the sitting, and then keeps `prog` in memory and applies the SAME transition the fold would:

```go
prog[key] = schedule.Answer(prog[key], grade, now)
```

That is not an approximation of `Fold` — it is the function `Fold` applies, so the in-memory figures cannot drift from what the next sitting will derive. `DailyLoad` is then a walk over the deck slice in memory: a few thousand iterations of at most twenty integer multiplications, which is microseconds, and it is charged per ANSWER rather than per frame only because there is no reason to redo it more often.

**THE FULL IO ENUMERATION, because a cost claim has to name every site that pays it — including the ones this plan does not modify.** Four exist today or are added here:

| site | reads today | after this issue |
|---|---|---|
| `todaysQuestions` `:238`, `:242` | `Deck()` + `Events()` | unchanged — the one read, now RETURNED rather than discarded |
| `finish` `:384`, `:388` | `Deck()` + `Events()` again | **removed** — takes the figures the loop already holds |
| the per-answer refresh | — | in memory, no IO |
| the frame paint | — | no IO |

So a sitting reads the deck once and the log once, whatever its length. **`finish` re-reading was `#39` T7's deliberate choice** — *"the figure a learner should see is the one AFTER today"* — and that reasoning survives while its mechanism does not: the loop's in-memory `prog` is updated by the SAME `schedule.Answer` the fold applies, so it already IS the after-today figure. `#39` had no in-memory copy to use; this issue creates one, which is what makes the re-read redundant rather than wrong.

The deck also changes mid-sitting when a word is dropped, and the loop already sees that as `OutcomeDrop` — so it drops from its in-memory copy too, and the bar cannot disagree with the deck the learner just curated.

**A failed read is unchanged and not a new concern:** `play_loop.go:246` already degrades a failed log read to empty progress and carries on, which is right — the reviews are recorded as they happen, and a sitting must not end because a summary could not be computed.

**D8 — the bar states what it assumes.** `~14 reviews/day · 0.9 new/day at 20 a sitting · 7 of 18 done`. The `-count` assumption is already spelled out in `finish()` (`#39` D11) and the bar uses the same wording, because two spellings of one assumption is how they drift.

**D9 — NOT in scope: any new question form.** This issue changes how `--play` DRAWS and nothing about what it asks. `#40` is the first consumer of the ground it lays. A diff here that touches `play/` beyond what the display needs has grown a second issue.

---

## What this plan asserts about the existing tree, verified

| claim | verified at | status |
|---|---|---|
| `--play` writes lines through `crlfWriter`, appending | `cmd/define/play_loop.go:304` | true — `draw` `Fprintf`s the prompt and keys on every call |
| the editor builds a `console` from `newLiveScreen` | `cmd/define/replraw.go:45` | true — rows and cols from `terminalRows`/`terminalCols` |
| `display` already has `Draw`, `Page`, `Scroll`, `Resize` | `cmd/define/replraw.go:114` | true |
| `Paint` budgets every component in DISPLAY ROWS and states its order of sacrifice | `cmd/define/screen.go:371` | true — prompt clipped last, menu drops whole rows, buffer takes the rest |
| the paint throttle is already 16ms | `cmd/define/screen.go:495` | true — `paintInterval`, with a trailing flush |
| `Deck()` reads a file per WORD and `Events()` a file per DAY | `cmd/define/store/yaml.go` | true — which is why D7 keeps the refresh in memory rather than re-reading |
| `finish` already prints the load and names the `-count` assumption | `cmd/define/play_loop.go` | true — `#39` T7 |
| the atlas already records this divergence as predicted | `atlas/define.md` | true — *"play_loop.go … therefore owns no coordinates. #30 D5a predicted this seam"* |

---

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Paint` | `cmd/define/screen.go` | modified | PURE — `menu` renamed `footer`; pads the buffer region when the screen is pinned (D2, D3a) |
| `newPinnedScreen` | `cmd/define/screen.go` | new | the `--play` constructor: buffer fills, footer at the bottom. A named constructor rather than a bool at a call site already taking two ints |
| `fitMenu` | `cmd/define/screen.go` | modified | PURE — renamed `fitFooter` with it, so the pair does not disagree |
| `sittingBar` | `cmd/define/playbar.go` | new | PURE — figures + progress → the bar's text. Takes numbers, never a deck |

- **`sittingBar`** — the footer's text for one moment in a sitting.
  - **DRY rationale:** `finish()` already renders the same figures at the end of a sitting (`#39` T7). One formatter means the bar and the summary cannot describe the same deck differently, and the `-count` assumption is worded once.
  - **Future extensions:** `#40`'s board needs a "16 of 40 marked" variant; the shape is the same and the row count is data.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `playConsole` | `cmd/define/play_loop.go` | new | the terminal — mirrors `replRaw`'s construction (D1) |
| `draw` | `cmd/define/play_loop.go` | modified | becomes "compute the live edge", not "append lines" (D4) |
| `finish` | `cmd/define/play_loop.go` | modified | shares `sittingBar`'s formatter, so the bar and the summary cannot word the `-count` assumption differently (D8) |
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | returns the deck and the folded progress it already computes, instead of discarding them (D7) |
| `playSession` | `cmd/define/play_loop.go` | modified | takes a `console`; tracks the written question and the cached figures |
| `toInput` | `cmd/define/play_loop.go` | unchanged | listed because D6 REVERSED an earlier plan to widen it: the loop intercepts paging before this is reached, and `play` learns no viewport |

**ARCH-MOCK.** No new external dependency. The display seam is an INTERFACE (`display`) that the editor's tests already fake, so `--play`'s tests take the same double; the pty conformance suite covers the real terminal, and `#7` already added a form-2.3 pty check that this issue extends with a bar assertion.

**ARCH-CONSTRAINTS.** The interaction path is a keystroke and a redraw. A frame is O(visible rows) of string building, already throttled to 16ms with a trailing flush (`paintInterval`), which this issue inherits rather than re-tunes.

**The bar's figures are the one new cost, and the first draft of this plan mis-stated them as "two disk reads, microseconds".** They are not: `Deck()` reads one file per word and `Events(anyTime)` one file per day of history, so a per-answer recompute is `O(deck files + log days)` — about 5,700 file reads per question on a 5,000-word deck with two years of log, with a person waiting. D7 moves it off that path entirely: the reads happen ONCE per sitting (and are already paid by `todaysQuestions`), the progress map is updated in memory with the same `schedule.Answer` the fold applies, and `DailyLoad` becomes a walk over the deck slice — a few thousand iterations of at most twenty integer multiplications.

| path | frequency | cost |
|---|---|---|
| frame paint | per keystroke, ≤16ms apart | O(visible rows), no IO |
| figure refresh | per answered question | O(deck), in memory, no IO |
| deck + log read | once per sitting | already paid by `todaysQuestions` |

Overload behaviour: a deck large enough for the per-answer walk to be felt would already have made the sitting's startup lookups untenable, so the bound that binds is the one `#39` made visible rather than anything this issue adds.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [ ] **T1 — `menu` becomes `footer`** (D2). Rename the parameter and `fitMenu`, update the editor's call sites and the `display` interface's doc. No behaviour change; the test suite is the proof.
- [ ] **T2 — `sittingBar`** (D8). A pure formatter in `cmd/define/playbar.go`, sharing its wording with `finish()`. Table test including the degenerate cases: nothing due, zero budget, a load of zero.
- [ ] **T3 — `--play` builds a `console`** (D1, D5a). Mirror `replRaw`'s construction, including `handBack` (D5), and DELETE the reveal's `restore`/`enterRaw` pair and its error branch — playback no longer leaves raw mode. `playSession` takes the console instead of a raw writer.
- [ ] **T4 — the question is written once** (D4). Track the written index; write `Prompt()` on transition and `Reveal()` on `OutcomeReveal`. Test that N keystrokes on one question leave ONE copy of it in the buffer — the assertion a naive port fails.
- [ ] **T5 — the live edge** (D3, D3a). `newPinnedScreen`, and `Paint` pads the buffer region to its full height when pinned — blank rows at paint time, never lines in the buffer. `draw` computes the grading keys and the bar and calls `Draw`; the frame's shape is `Paint`'s business.
- [ ] **T6 — the figures are in memory, and `finish` stops re-reading** (D7). `todaysQuestions` returns the deck and progress it already computes; the loop applies `schedule.Answer` on each record, drops on `OutcomeDrop`, and recomputes `DailyLoad` from memory. `finish` takes the figures instead of reading — superseding `#39` T7's re-read, whose REASONING survives (the after-today figure) while its mechanism becomes redundant. Counting store: a sitting of N answers calls `Deck()` exactly once and `Events()` exactly once.
- [ ] **T7 — paging** (D6). The LOOP intercepts the wheel and PageUp/PageDown and calls `view.Scroll`/`view.Page`; `toInput` is untouched and `play` learns nothing. Test that a reveal taller than the viewport keeps the prompt word on screen after a page, and that `play.Input` gained no kind.
- [ ] **T8 — SIGWINCH** (D1). The resize case redraws through the console, as the editor's does.
- [ ] **T9 — pty conformance + docs.** Three existing pty tests assert over `--play`'s RAW BYTE STREAM, which becomes whole frames, so each is re-examined rather than assumed:

  | test | what it asserts | expectation under frames |
  |---|---|---|
  | `TestPTYPlayRendersEveryLineAtColumnZero` | no bare `\n` reaches the terminal | must still HOLD — `Paint` emits CRLF — and it stays the shipped-defect net |
  | `TestPTYPlayCorrectAnswerNeverRevealsIt` | the definition is absent before answering | holds: the buffer carries the question only until `OutcomeReveal` |
  | `TestPTYPlayGradeFirst` | the keys are offered up front, definition hidden | holds, but the keys now arrive inside a frame, so `unstyled()` must still find them |

  All three re-run on real hardware, not reasoned about. Then extend `#7`'s form-2.3 pty test to assert the bar is present and updates; atlas and README.

---

## Done when

Every row's pin is a PREDICATE OVER BEHAVIOUR — a named test or a grep for a property — never "file X is unchanged".

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | `--play` paints whole frames; no path appends bare lines | `TestPlayDrawsThroughTheDisplay` | a `Fprintln` to the tty returns |
| 2 | one question leaves ONE copy in the buffer however many keys are pressed | `TestRepeatedKeystrokesDoNotDuplicateTheQuestion` | the naive port (D4) |
| 3 | the bar is pinned across question, reveal and resize | `TestTheBarSurvivesEveryState` | a state forgets to pass the footer |
| 4 | a reveal taller than the viewport PAGES; the word stays on screen | `TestALongRevealPagesRatherThanScrollingTheWordAway` | paging is not wired |
| 5 | the transcript survives exit | `TestPlayTranscriptSurvivesExit`, the shape `#30` used | `handBack` is skipped |
| 6 | a whole sitting reads the deck ONCE and the log ONCE, whatever its length | `TestASittingReadsTheDeckOnce` — a counting store, N answers, exactly one `Deck()` and one `Events()` | any of the four sites in D7's enumeration reads again |
| 7 | a failed log read degrades to empty progress and the sitting still runs | the existing behaviour at `play_loop.go:246`, unchanged | a read failure ends a sitting whose reviews are already recorded |
| 8 | the `-count` assumption is worded once | `TestTheBarAndTheSummaryAgree` — same formatter | the bar and `finish()` spell it differently |
| 9 | SIGWINCH repaints mid-sitting | `TestPlayRepaintsOnResize` | the resize case is not wired |
| 10 | the real terminal shows the bar and updates it | the `#7` pty test, extended | it works in-process and not on a tty |
| 11 | the bar sits at the TERMINAL'S bottom on a short question, and padding never enters the buffer | `TestAShortQuestionStillPinsTheBar` and `TestPaddingNeverReachesTheTranscript` | the bar floats under the content, or blank rows appear in `Lines()` |
| 12 | the editor's footer still FOLLOWS its content | the editor's existing frame tests, unchanged | `newPinnedScreen`'s padding leaks into the REPL |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then on a real terminal with a deck of a dozen words: `define --play`, confirm the bar is pinned at the bottom and its counts move as answers land; reveal a long entry (`run`, `bank`) and page back to the question; resize the window mid-sitting; quit and confirm the sitting is still on screen.

**Close:** one boundary, one `sdlc close`, one publish.

## Revisions

### 2026-08-31 — plan-quality round 1

- **PQ-1 (Critical) — the reveal's `restore`/`enterRaw` pair would have dropped
  the alternate screen on the first reveal and never re-entered it**, since
  `enterAlt` is opt-in on `rawSession` and `enterRaw` returns a fresh one.
  Resolved by DELETING the dance: playback no longer leaves raw mode, because
  under the frame model the indicator is a frame write and `screen.Write`
  already handles it. The error branch goes with it, and is not weakened — it
  exists only because the terminal was handed back, and nothing is handed back
  any more.
- **PQ-2 — the cost was stated as "two disk reads, microseconds" and it is
  `O(deck files + log days)` per answer.** ~5,700 file reads per question on a
  realistic deck. The figures now refresh in memory through the same
  `schedule.Answer` the fold applies, so they cannot drift from what the next
  sitting derives, and the reads happen once per sitting — already paid.
- **PQ-3 — routing paging through `toInput` would have taught the pure `play`
  package about a viewport.** `toInput` converts a terminal key into a session
  intent, and `play`'s design is that main owns the terminal. The loop
  intercepts the paging keys and calls `view.Page`/`view.Scroll` directly;
  `play.Input` gains no kind.
- **Minor — three pty tests assert over the raw byte stream**, which becomes
  whole frames. T9 now names each and what is expected of it, rather than
  leaving a reviewer to discover that `--play`'s column-zero defect net was
  silently in scope.
- **Minor — `finish` and `todaysQuestions` were touched but unlisted.** Both
  are now in the integration table, which is what gives D8's DRY claim and D7's
  "already paid" claim an owner.

### 2026-08-31 — plan-quality round 2

- **PQ-3 was still open because I revised the DECISIONS and left the tables.**
  Two rows still cited the reversed reasoning: the verified-claims row repeating
  "two disk reads, which is why D7 caches" (the exact basis PQ-2 corrected) and
  the integration row saying `toInput` "gains the paging keys" against a D6 that
  now says it is untouched.

  **The rule, which is the deliverable rather than the two edits: a Revision that
  reverses a decision re-reads every table row and task that cited it.** A plan's
  prose and its tables are written at different moments and read by different
  people — the judge reads the tables, a human reads the prose — so a correction
  applied to one and not the other leaves the artifact arguing with itself, and
  the half that is wrong is the half a machine is checking.

- **PQ-6 — D3 claimed the bar pins to the bottom "for free" and `Paint` never
  pads a short buffer.** The sacrifice ORDER was free; the PINNING is not. D3a
  adds `newPinnedScreen`, and Done-when 12 pins that the editor's footer keeps
  following its content — this must not become a change to the REPL's
  appearance, which would be a second issue wearing this one's clothes.

### 2026-08-31 — plan-quality round 3

- **PQ-9 (and what kept PQ-3 alive) — I swept the tables in round 2 and left the
  DONE-WHEN ROWS.** Rows 6 and 7 still encoded the per-answer-read model D7 had
  reversed: "computed at most once per ANSWER" and "a failed read keeps the
  previous figures", the second of which presupposes a repeated read that no
  longer exists.

  Round 2 stated the rule — *a Revision that reverses a decision re-reads every
  table row and task that cited it* — and then applied it to two of the three
  surfaces a plan has. **The surfaces are: the decision prose, the entity
  tables, and the Done-when rows**, and a sweep that stops at two is how the same
  finding returns with a new number. Applied to all three now, and the rule is
  restated with the enumeration rather than as an instruction to be thorough.

- **PQ-8 — "one deck read and one log read per sitting" was contradicted by a
  site I wrote a day ago.** `finish` reads both again (`#39` T7), so the sitting
  pays two of each and T6's test would have failed as written.

  2nd finding in `cost-basis-unverified`, so the rule: **every IO claim names the
  call sites that pay it and is checked against all of them, including sites the
  plan does not modify.** The enumeration is now a table in ARCH-CONSTRAINTS with
  four rows, and the decision it forced is a real simplification — `finish` stops
  re-reading, because the loop now holds an in-memory `prog` updated by the same
  transition the fold applies, which `#39` did not have available.
