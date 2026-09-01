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
| `fitFooter` | `cmd/define/screen.go` | modified | PURE — was `fitMenu`; renamed with `Paint`'s parameter so the pair cannot disagree about what it fits |
| `OptionIndent` | `cmd/define/play/choice.go` | new | PURE — how many columns `optionLine` prepends. Exported because the caller pre-wraps the gloss and must know what goes in front of its first line |
| `choiceFor` | `cmd/define/optionpool.go` | unchanged | listed because BR-4 REVERSED a first fix that gave it the terminal width: rendering-time is the wrong place to wrap, since the sitting writes what it rendered much later. The wrap lives in `wrapWritten`, at the moment of writing |
| `sittingBar` | `cmd/define/playbar.go` | new | PURE — figures + progress → the bar's text. Takes numbers, never a deck |
| `viewportGesture` | `cmd/define/replraw.go` | new | PURE dispatch over the `display` seam — the paging keys, for BOTH loops (D6, BR-1). A second copy of the policy is how two loops come to disagree about which direction a page goes |
| `wrapWritten` | `cmd/define/playbar.go` | new | PURE — string to string. Wraps whatever is written into a clipping frame, at the width in force then (BR-4, BR-15, BR-20). One line-kind at a time is what produced FOUR findings in one family |
| `livePrompt` | `cmd/define/play_loop.go` | new | PURE — what `draw`'s last two lines became: the frame's PROMPT for one state, returned rather than printed. The question and the reveal are buffer writes the LOOP owns, because it is the loop that knows they are transitions |
| `sittingDeck` | `cmd/define/play_loop.go` | new | the deck ONE sitting holds in memory, and the home of D7's claim: `answered` applies the same transition `Fold` does, `dropped` keeps it agreeing with the deck the learner just curated, `figures` walks it with no IO |
| `GradeOf` | `cmd/define/schedule/progress.go` | new | PURE — the rule turning `(correct, unaided)` into a rung. Exported because D7 gave it a second caller, and two spellings of one rule is how the bar's figures would drift from the log's |

- **`sittingBar`** — the footer's text for one moment in a sitting.
  - **DRY rationale:** `finish()` already renders the same figures at the end of a sitting (`#39` T7). One formatter means the bar and the summary cannot describe the same deck differently, and the `-count` assumption is worded once.
  - **Future extensions:** `#40`'s board needs a "16 of 40 marked" variant; the shape is the same and the row count is data.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `newPinnedScreen` | `cmd/define/screen.go` | new | the terminal, for `--play`: the buffer region fills so the footer sits on the bottom row, and its `Write` WRAPS, which is the seam a helper cannot write around (D3a, BR-20) |
| `newConsole` | `cmd/define/replraw.go` | new | the terminal, for BOTH loops — the screen constructor is its one parameter (D1, BR-7). The first cut of this was `playConsole`, a verbatim copy of `replRaw`'s six statements differing in one token; D1 had committed to the opposite |
| `draw` | `cmd/define/play_loop.go` | deleted | it wrote the question, the reveal AND the keys on every call; those three have different lifetimes and a scrolling terminal could not express the difference (D4) |
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

- [x] **T1 — `menu` becomes `footer`** (D2). Rename the parameter and `fitMenu`, update the editor's call sites and the `display` interface's doc. No behaviour change; the test suite is the proof.
- [x] **T2 — `sittingBar`** (D8). A pure formatter in `cmd/define/playbar.go`, sharing its wording with `finish()`. Table test including the degenerate cases: nothing due, zero budget, a load of zero.
- [x] **T3 — `--play` builds a `console`** (D1, D5a). Mirror `replRaw`'s construction, including `handBack` (D5), and DELETE the reveal's `restore`/`enterRaw` pair and its error branch — playback no longer leaves raw mode. `playSession` takes the console instead of a raw writer.
- [x] **T4 — the question is written once** (D4). Track the written index; write `Prompt()` on transition and `Reveal()` on `OutcomeReveal`. Test that N keystrokes on one question leave ONE copy of it in the buffer — the assertion a naive port fails.
- [x] **T5 — the live edge** (D3, D3a). `newPinnedScreen`, and `Paint` pads the buffer region to its full height when pinned — blank rows at paint time, never lines in the buffer. `draw` computes the grading keys and the bar and calls `Draw`; the frame's shape is `Paint`'s business.
- [x] **T6 — the figures are in memory, and `finish` stops re-reading** (D7). `todaysQuestions` returns the deck and progress it already computes; the loop applies `schedule.Answer` on each record, drops on `OutcomeDrop`, and recomputes `DailyLoad` from memory. `finish` takes the figures instead of reading — superseding `#39` T7's re-read, whose REASONING survives (the after-today figure) while its mechanism becomes redundant. Counting store: a sitting of N answers calls `Deck()` exactly once and `Events()` exactly once.
- [x] **T7 — paging** (D6). The LOOP intercepts the wheel and PageUp/PageDown and calls `view.Scroll`/`view.Page`; `toInput` is untouched and `play` learns nothing. Test that a reveal taller than the viewport keeps the prompt word on screen after a page, and that `play.Input` gained no kind.
- [x] **T8 — SIGWINCH** (D1). The resize case redraws through the console, as the editor's does.
- [x] **T9 — pty conformance + docs.** Three existing pty tests assert over `--play`'s RAW BYTE STREAM, which becomes whole frames, so each is re-examined rather than assumed:

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
| 0a | the full-screen surface is gated on STDOUT, not on stdin alone | `TestPlayRefusesWhenStdoutIsNotATerminal`, `TestPTYPlayRefusesWithNoColor` | `define --play > file` writes frames into it, or `-no-color` still takes the alternate screen |
| 0b | an over-wide option line WRAPS, at the width in force when it is written | `TestALongOptionGlossWrapsRatherThanBeingCut`, `TestANarrowedSittingWrapsTheRestOfItself` | the wrap is baked at queue-build time, so narrowing the window clips the rest of the sitting |
| 1 | `--play` paints whole frames; no path appends bare lines | `TestPlayDrawsThroughTheDisplay` | a `Fprintln` to the tty returns |
| 2 | one question leaves ONE copy in the buffer however many keys are pressed | `TestRepeatedKeystrokesDoNotDuplicateTheQuestion` | the naive port (D4) |
| 3 | the bar is pinned across question, reveal and resize | `TestTheBarSurvivesEveryState` | a state forgets to pass the footer |
| 4 | a reveal taller than the viewport PAGES; the word stays on screen | `TestALongRevealPagesRatherThanScrollingTheWordAway` | paging is not wired |
| 5 | the transcript survives exit | `TestPlayTranscriptSurvivesExit`, the shape `#30` used | `handBack` is skipped |
| 6 | a whole sitting reads the deck ONCE and the log ONCE, whatever its length | `TestASittingReadsTheDeckOnce` — a counting store, N answers, exactly one `Deck()` and one `Events()` | any of the four sites in D7's enumeration reads again |
| 7 | a failed log read degrades to empty progress and the sitting still runs | `TestAFailedLogReadStillRunsTheSitting` | a read failure ends a sitting whose reviews are already recorded, or the bar is drawn from a nil map |
| 8 | the `-count` assumption is worded once | `TestTheBarAndTheSummaryAgree` — same formatter | the bar and `finish()` spell it differently |
| 9 | SIGWINCH repaints mid-sitting | `TestPlayRepaintsOnResize` | the resize case is not wired |
| 10 | the real terminal shows the bar and updates it | the `#7` pty test, extended | it works in-process and not on a tty |
| 11 | the bar sits at the TERMINAL'S bottom on a short question, and padding never enters the buffer | `TestAShortQuestionStillPinsTheBar` and `TestPaddingNeverReachesTheTranscript` | the bar floats under the content, or blank rows appear in `Lines()` |
| 12 | the editor's footer still FOLLOWS its content | `TestTheEditorsFooterFollowsItsContent` and `TestOnlyThePinnedConstructorPads`, plus the editor's existing frame tests unchanged | `newPinnedScreen`'s padding leaks into the REPL |

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

### 2026-08-31 — T3/T4 landed: `draw` split by lifetime rather than modified

The integration table said `draw` | **modified** | *"becomes 'compute the live
edge', not 'append lines'"*. Implementing it showed the row was one entity where
there are two: `draw` wrote three things with three different lifetimes, and the
change is not that it computes differently but that the three go to different
places. The keys are the frame's prompt (`livePrompt`, pure, returned); the
question and the reveal are buffer writes, and they belong to the LOOP because
only the loop knows which keystroke was a transition — the write-once state
(`written`) has to sit beside "perform the outcomes" or it is tracking something
its owner cannot see.

So `draw` is **deleted** and `livePrompt` is **new**, which is what the code
does. A `modified` row over a name that no longer exists is the plan claiming a
shape the tree does not have.

Two consequences worth recording, both of which are the plan's own tests being
made real rather than departures from it:

- **`editorConsole` is now `recordingConsole`.** `console` has a second consumer,
  so a sitting's test that built an "editorConsole" would name the wrong loop.
- **`TestLosingTheTerminalAfterPlaybackExitsOne` is replaced, not just deleted.**
  D5a deletes the branch it drove, but that test was ALSO the pin for the outcome
  ORDER (record before reveal) that `session.go` calls load-bearing — BR-13 found
  that reversing the loop's iteration left the whole suite green. The replacement
  is `TestAMissIsRecordedBeforeItIsRevealed`, which observes the store from
  INSIDE playback via a hooked player: the order stays pinned by something that
  does not depend on a failure branch existing. **The rule: deleting the code a
  test drove is not deleting the test's claim — check what else that test was the
  only pin for.**

### 2026-08-31 — T5/T6 landed: two entities the plan implied but did not name

D7 said the loop "keeps `prog` in memory and applies the SAME transition the fold
would". Implementing it turned up that the sentence names a THING — a deck the
sitting holds, with three operations on it — and the plan had it as a phrase.
`sittingDeck` is that thing, and it is where the claim can actually be defended:
`answered` is the only place the transition is applied, `dropped` is the only
place the in-memory deck and the deck on disk can disagree, and `figures` is the
only walk. Left as loose locals in the loop, each of those would have been a
place to forget.

The second is smaller and the same shape. "The SAME function `Fold` applies" was
not literally true: `Fold` reaches the rule through `gradeOf(store.ReviewEvent)`,
which the loop has no event to call it with. So `schedule.GradeOf(correct,
unaided bool)` is the rule, `gradeOf` is one line over it, and the loop is the
second caller. **A DRY claim in a decision has to name the function, or it is a
claim about two pieces of code that happen to agree today.**

Both rows are now in Core concepts.

### 2026-08-31 — T9: three pty rows held, a fourth did not, and that was the trip's value

The plan named THREE existing pty rows to re-examine and predicted all three
would hold. They did, on real hardware. The plan did not name the fourth —
`TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis`, `#7`'s form-2.3 row — and that
is the one frames broke.

It read the correct option out of *"the chunk the reveal produced"*, which is a
premise only an APPEND-ONLY surface has. A frame redraws the question and the
reveal together, so "the first option line in the chunk" became option 1 every
time and the test's deliberate miss silently stopped being deliberate — a wrong
guess is still a legal answer, so the failure surfaced as a stall rather than as
a wrong assertion. Measured on a real terminal rather than reasoned about: after
a reveal the options are scrolled off the screen entirely.

The fix uses this issue's own behaviour as the instrument — page to the top, then
read the option digit that appears TWICE — and the row now also asserts the bar
is present and counts, which is T9's other half.

**The rule the plan should have applied to itself: "re-examine the tests that
assert over the surface this issue changes" means ENUMERATING them, not listing
the ones already in mind.** `grep -l TestPTYPlay` was the enumeration, and it has
four rows. Two more were added rather than re-examined —
`TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` for D5a's Critical and
`TestPTYPlayResizeRepaints` for the sitting's SIGWINCH, both mutation-verified.

### 2026-08-31 — operator found it: a frame CLIPS, and option glosses were never wrapped

Reported from a real sitting, with a screenshot: `ligament`'s option line ran off
the right edge and was **cut**, not wrapped.

The plan never considered it, and the reason it did not is worth writing down.
Every claim about width in this plan is about the LIVE EDGE — `Paint` clips the
prompt, `fitFooter` drops footer rows — because those are the components #41
adds. The BUFFER's clipping is `#30`'s, inherited and unexamined: *"buffer lines
are clipped to the width — clipped at PAINT time, so the transcript and the click
map keep the whole text."* True, and it silently changed what an unwrapped line
does. Before this issue the terminal wrapped an over-wide option line — at the
column, with no indent, which `#7` recorded as a KNOWN ROUGH EDGE and left. A
frame cannot allow that wrap (a wrapped line is a frame one row too tall, and the
terminal then scrolls every placed row), so it clips, and the rough edge became
information loss.

`#7` had already named the fix: *"if it becomes annoying the honest fix is to
wrap in main and pass pre-wrapped option text, the same way the definition
already arrives pre-rendered."* It is not annoying any more, it is lossy. So
`choiceFor` takes the width and wraps each gloss through `wrapText` — the same
function that wraps the definition — to `play.OptionIndent`, which is exported
for exactly this reason: the caller pre-wraps and therefore has to know what the
form will prepend.

One forced consequence: `Choice.Reveal` put the label inline (`"you chose 3  …"`),
which pushes a pre-wrapped first line ten columns past its width. The label now
gets its own line. Wrapping every option ten columns narrower to buy room for one
line in one state is the worse trade.

**The rule: adopting an existing seam inherits its behaviour on inputs the
previous consumer never sent it.** `#30`'s clipping was correct for a REPL, whose
lines are all pre-wrapped by `Render`. `--play` had one line that was not, and
nothing in "adopt the editor's screen" prompted anyone to ask which.

### 2026-08-31 — boundary review: FIX-THEN-SHIP, six Importants, and the one rule behind three of them

The fresh-eyes review at `sdlc close` returned six blocking findings. Three of
them are the same rule, and it is the rule this plan had already written down
about itself one revision earlier and then not applied:

> **Adopting an existing seam inherits its behaviour on inputs the previous
> consumer never sent it.**

- **BR-3 — the full-screen surface was gated on STDIN alone.** `repl` computes
  `terminalUI := interactive && opt.tty` and falls back to the line loop, with a
  comment recording that gating cursor control on the wrong stream has already
  shipped three times here. `--play` checking only `stdinIsTerminal` was harmless
  while it appended lines and emitted no escapes at all; the moment it took the
  alternate screen, `define --play > file` wrote frames into the file at a
  fabricated 80 columns and `-no-color` — a flag whose whole purpose is
  terminals that mangle escapes — stopped meaning anything. It now REFUSES, with
  the two causes told apart because their fixes differ.
- **BR-4 — the wrap was fixed at the startup width, so the same defect was one
  resize away.** The revision above closed the operator's instance and stated the
  rule; the review measured the sibling. The wrap moved from `choiceFor` (queue
  build) to `wrapOptionLines` at WRITE time, and the resize case keeps
  `opt.width` current — which also retires the `opt.width == 0` sentinel
  mismatch, since the loop now wraps against the width it is actually painting
  at.
- **BR-6 — `crlfWriter` had no production caller left, and this window added
  three claims that it did.** Deleted, with `crlf.go`, `crlf_test.go` and eight
  stale mentions swept; `shortWriter` re-homed to `highlightwriter_test.go`,
  where the short-write contract it proves actually lives.

The other three are structural:

- **BR-7 / BR-1 — `playConsole` was a verbatim second copy of `replRaw`'s
  construction, and the viewport switch a second copy of the editor's.** D1 said
  *"the honest move is to widen the shared seam rather than grow a parallel
  one"*, and the first cut did the opposite twice. Now `newConsole(…, newScreen)`
  and `viewportGesture(view, k)`, both called by both loops. **A plan that names
  the anti-pattern is not protection against writing it; the diff is.**
- **BR-5 — a test passed on an aliasing artifact.** `sittingDeck` was passed by
  value with pointer-receiver mutators, so `TestDropping…` saw the drop only
  because `slices.DeleteFunc` compacts the shared backing array in place. It now
  asserts on the DRAWN BAR, which is what the claim was about, and `sittingDeck`
  travels by pointer so the question cannot arise for `#40`.
- **BR-2 — Done-when row 7 was pinned by "the existing behaviour, unchanged"**,
  which this section's own header forbids. Now `TestAFailedLogReadStillRunsTheSitting`,
  and it needed to be: the degraded progress feeds the bar and the summary now,
  where before `finish` re-read the log.

**The enumeration BR-4 forced, recorded because the plan's own T9 rule says to
enumerate rather than list what comes to mind.** An unwrapped line meets a
clipping frame at three sites: (a) the startup width, (b) a narrowing resize,
(c) `opt.width == 0` — and, found a round later, (d) the rendered DEFINITION a
reveal writes, which is not an option line at all. (a), (b) and (d) are closed by
wrapping EVERYTHING the loop writes at the width in force then. (c) is not a bug
to close: `terminalWidth` returns 0 below 20 columns because nothing can be
broken that narrowly and stay readable, and `Render` is given the same answer. What remains, stated rather than discovered later: **the question
already on screen keeps the wrapping it was written with**, exactly as the
editor's scrollback does (`#30`) — and nothing is lost by it, because clipping
happens at PAINT and the whole text is still in the buffer if the window widens.

### 2026-08-31 — review rounds 2 and 3: the same family four times, and the seam that ends it

`frame-clips-unwrapped-text` produced FOUR findings across an operator report and
three review rounds. Each fix was correct and each was an instance:

1. the operator's option gloss, wrapped at the **queue build**;
2. BR-4's narrowing resize, so the wrap moved to the **loop's writes**;
3. BR-15's rendered definition — a line-kind the loop's wrap did not match;
4. BR-20's `playAnnounced` warning, measured at 156 cells in a 40-column
   terminal — written by a HELPER the loop calls, so no rule enforced at the
   loop's call sites could ever have caught it.

The fix is the pinned screen's own `Write`. **A rule every caller must remember
is a rule with a caller who will not** — and the callers are not even all in this
package. The sub-20-column policy moved with it, so there is one number and one
place rather than `opt.width` and `cols` answering the same question differently.

The loop's per-site wraps are DELETED with it, and so is the resize case's
`opt.width` update: the screen's own `cols` is what `Resize` maintains, and a
second width would be a second answer.

Also from these rounds:

- **BR-1 was open for two rounds because I extracted the helper and gave it to
  one loop.** `viewportGesture` existed, the editor called it, and `--play` kept
  the copy the finding was about. Extracting a shared function is half the fix;
  the other half is deleting what it replaced.
- **BR-22 — three pure functions were filed under "Integration points", whose
  column header is "Wraps", and the IO constructor under "Pure entities".** The
  status column is guarded and the KIND column is not, so the table `#40` reads
  as the record of what landed can be wrong in the half nothing checks. Rows
  corrected; the mechanisation belongs to `#33`, which is filed for exactly this.
- **BR-21 — the initial figures hand-copied `refresh()`'s first lines.** Calling
  `refresh()` is exactly equivalent at init, so a third field added later cannot
  be stale on the first frame.

### 2026-08-31 — round 4: a Critical I introduced, and the rig that hid it

`wrapWritten` skipped every line carrying an escape, to protect the `♫ playing
3×` indicator's `\r\x1b[K` marker from being scattered by a wrap. Since BR-3,
`--play` refuses to run with `-no-color` — so every rendered definition line
carries colour, and the skip exempted **exactly the lines the wrap exists for**.
Now only the erase gesture is exempt; SGR wraps, because `visibleCells` measures
styled text correctly and an attribute persists across a break to its own reset.

**BR-24 is why it shipped, and it is the more useful finding.** `playRig`
returned `options{color: false}` — a configuration `--play` now REFUSES — so no
in-process sitting has ever run in the only state production can produce. Both of
Done-when 0b's pins were green over uncoloured text. The rig now sets
`color: true, tty: true`, and `unstyled` moved out of the conformance file
because in-process assertions over frame text now meet escapes for the first
time.

**THE RULE: a rig's default options must be reachable from the flag parse of the
command under test.** A default that production cannot produce is a suite testing
a state that does not exist, and it fails silently — everything passes.

And the second-order one, which is why this took four rounds: **a guard added to
protect a special case must name the case, not the mechanism it happens to use.**
"Skip escape-carrying lines" was a guess at what needed protecting; "skip lines
carrying the erase gesture" is the actual case, and it is one `strings.Contains`
away.
