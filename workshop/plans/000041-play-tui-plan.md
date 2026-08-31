# Play-Mode Frames Implementation Plan (`#41`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `--play` draws whole frames through the same `screen`/`display` seam the editor uses, so it owns coordinates — which buys a pinned status bar, paging over a long reveal, and the ground `#40`'s grid needs.

**Architecture:** No new abstraction. `#30` built `screen` (a pure line buffer plus viewport) and `liveScreen` (the only part that touches a terminal) for the editor loop, and `atlas/define.md` already records `--play`'s divergence from it as predicted. This issue makes `--play` the second consumer: it builds a `console` exactly as `replRaw` does, writes the question into the buffer once per question rather than appending it per keystroke, and passes the grading keys as the prompt and the status bar as the footer.

**Tech Stack:** Go 1.26. No new dependency.

---

## Decisions

**D1 — `--play` reuses `console`, and adds nothing.** `replraw.go` already builds `console{view, resizes, finish, stdout, stderr}` from `newLiveScreen(stdout, terminalRows, terminalCols)`. That construction is what `--play` adopts. If anything in it turns out not to fit, the honest move is to widen the shared seam rather than grow a parallel one — a second way to draw is the thing this issue exists to remove, not to add.

**D2 — `Paint`'s `menu` parameter is RENAMED `footer`, and that is not cosmetic.** The concept `Paint` actually implements is *"rows below the prompt, which give up whole rows before the prompt does"*. The editor's use of that is a command menu; play's is a status bar; calling the parameter `menu` would make play's call site read as something it is not, and would invite a future third consumer to add a third parameter for its own bottom rows. One name, one budget, two consumers.

**D3 — the grading keys are the PROMPT and the status bar is the FOOTER, which gets the sacrifice order right for free.** `Paint` documents its own order of value: *"The prompt is the line you are typing and survives first… The menu is a dropdown and gives up whole rows next. The buffer is scrollable, so it takes what is left."* For a review sitting that ordering is already correct — a learner who cannot see the grading keys cannot answer at all, while a learner who cannot see their daily load loses nothing this minute. So no new layout logic, and the visual order (buffer, keys, bar) is the one wanted.

**D4 — the question goes into the BUFFER once; the keys and the bar are the live edge.** This is the change of model, and the place a naive port breaks. `draw()` today writes the prompt, the reveal and the keys on EVERY call, which is correct for a scrolling terminal and would, against a line buffer, append a copy of the question per keystroke.

So the loop tracks which question it has already written and writes on transition:

- the current question's `Prompt()` is written when the index changes,
- its `Reveal()` is written when `OutcomeReveal` is performed — which the loop already handles, since that is where the pronunciation plays,
- everything else is `view.Draw(keys, footer)` on every frame.

The state is one `int` and it lives in the loop, which is where "perform the outcomes" already lives.

**D5 — the transcript surviving exit is REUSED, not re-decided.** `#30` M1.2b is *"Paint draws a whole frame, and the transcript survives the alt screen"*, and `handBack`/`onceHandBack` in `replraw.go` own the order. A learner who quits a sitting should still see the words they just reviewed; that is more true of a review than of an editor session. Whatever the editor does here, `--play` does the same call.

**D6 — paging is inherited, and it is the user-visible fix that justifies this issue on its own.** Form 2.3's reveal shows the whole rendered entry, which for `run` or `bank` is several screenfuls, and today the question scrolls off the top. `#30` already decodes the wheel and PageUp/PageDown into `Scroll`/`Page`, and `screen` already has the viewport. The loop's `toInput` gains those cases; nothing else moves.

**D7 — the bar's numbers are computed ONCE PER ANSWER, never per frame.** `schedule.DailyLoad` needs the deck and the folded log — two disk reads. `Draw` is called on every keystroke, and a frame that reads the disk would put IO on the keystroke path, which `#30` spent a milestone getting off it.

So the loop holds the figures and refreshes them only where they can have changed: after an `OutcomeRecord`, which is the only thing that moves a box. On failure the previous figures stand rather than the bar disappearing — a stale count is better than a flickering one, and the sitting is not the place to report that a disk read failed.

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
| `schedule.DailyLoad` needs the deck AND the fold | `cmd/define/schedule/load.go` | true — two disk reads, which is why D7 caches |
| `finish` already prints the load and names the `-count` assumption | `cmd/define/play_loop.go` | true — `#39` T7 |
| the atlas already records this divergence as predicted | `atlas/define.md` | true — *"play_loop.go … therefore owns no coordinates. #30 D5a predicted this seam"* |

---

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Paint` | `cmd/define/screen.go` | modified | PURE — `menu` renamed `footer`; one concept, two consumers (D2) |
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
| `playSession` | `cmd/define/play_loop.go` | modified | takes a `console`; tracks the written question and the cached figures |
| `toInput` | `cmd/define/play_loop.go` | modified | gains the paging keys (D6) |

**ARCH-MOCK.** No new external dependency. The display seam is an INTERFACE (`display`) that the editor's tests already fake, so `--play`'s tests take the same double; the pty conformance suite covers the real terminal, and `#7` already added a form-2.3 pty check that this issue extends with a bar assertion.

**ARCH-CONSTRAINTS.** The interaction path is a keystroke and a redraw. A frame is O(visible rows) of string building and is already throttled to 16ms with a trailing flush (`paintInterval`), which this issue inherits rather than re-tunes. THE ONE NEW COST IS THE BAR'S FIGURES, and D7 keeps it off that path entirely: `DailyLoad` is two disk reads and is recomputed only after an `OutcomeRecord` — at most once per answered question, so at most `-count` times a sitting, against the dictionary lookups the sitting already pays at startup. Nothing here grows with deck size on a per-frame basis; the per-answer recompute is linear in the deck, which for a 5,000-word deck is a map walk of microseconds.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [ ] **T1 — `menu` becomes `footer`** (D2). Rename the parameter and `fitMenu`, update the editor's call sites and the `display` interface's doc. No behaviour change; the test suite is the proof.
- [ ] **T2 — `sittingBar`** (D8). A pure formatter in `cmd/define/playbar.go`, sharing its wording with `finish()`. Table test including the degenerate cases: nothing due, zero budget, a load of zero.
- [ ] **T3 — `--play` builds a `console`** (D1). Mirror `replRaw`'s construction, including `handBack` (D5). `playSession` takes the console instead of a raw writer.
- [ ] **T4 — the question is written once** (D4). Track the written index; write `Prompt()` on transition and `Reveal()` on `OutcomeReveal`. Test that N keystrokes on one question leave ONE copy of it in the buffer — the assertion a naive port fails.
- [ ] **T5 — the live edge** (D3). `draw` computes the grading keys and the bar and calls `Draw`; the frame's shape is `Paint`'s business.
- [ ] **T6 — the figures are cached** (D7). Recompute after `OutcomeRecord` only; keep the previous values on a read failure. Test with a counting store that a sitting of N answers reads the deck at most N+1 times.
- [ ] **T7 — paging** (D6). `toInput` maps the wheel and PageUp/PageDown to `Scroll`/`Page`. Test that a reveal taller than the viewport keeps the prompt word on screen after a page.
- [ ] **T8 — SIGWINCH** (D1). The resize case redraws through the console, as the editor's does.
- [ ] **T9 — pty conformance + docs.** Extend `#7`'s form-2.3 pty test to assert the bar is present and updates; atlas and README.

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
| 6 | the bar's figures are computed at most once per ANSWER | `TestTheBarDoesNotReadTheDiskPerFrame` — a counting store, N answers, ≤N+1 reads | the figures move onto the frame path |
| 7 | a failed read keeps the previous figures rather than blanking the bar | `TestTheBarSurvivesAFailedRead` | the bar disappears on a transient error |
| 8 | the `-count` assumption is worded once | `TestTheBarAndTheSummaryAgree` — same formatter | the bar and `finish()` spell it differently |
| 9 | SIGWINCH repaints mid-sitting | `TestPlayRepaintsOnResize` | the resize case is not wired |
| 10 | the real terminal shows the bar and updates it | the `#7` pty test, extended | it works in-process and not on a tty |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then on a real terminal with a deck of a dozen words: `define --play`, confirm the bar is pinned at the bottom and its counts move as answers land; reveal a long entry (`run`, `bank`) and page back to the question; resize the window mid-sitting; quit and confirm the sitting is still on screen.

**Close:** one boundary, one `sdlc close`, one publish.
