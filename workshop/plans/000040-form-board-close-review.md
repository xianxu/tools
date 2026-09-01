# Boundary Review — tools#40 (whole-issue close)

| field | value |
|-------|-------|
| issue | 40 — form 2.5: the board — grid triage for mature words |
| repo | tools |
| issue file | workshop/issues/000040-form-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | eb9f1698d6800ff5ad22683f49e2c36e970d36f3..c6ac1ac1d4170ef01f93b23a61cfc30c7b6e9211 |
| command | sdlc close --issue 40 |
| reviewer | claude |
| timestamp | 2026-09-01T12:47:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The board is a genuinely well-built form: `Board` is pure and imports nothing, the session stays form-agnostic behind five capability interfaces, `Form()` is compiler-enforced on the `Question` interface and stamped in exactly one place, and `FooterRowAt` is the right seam — it answers the footer *entry* index and refuses rows `fitFooter` dropped, so a click can never name a word that was not painted. The test suite is unusually strong (real end-to-end measurement, a grep guard that can actually fail, a yaml round-trip that catches the `yaml:"-"` class). What blocks SHIP is one confirmed correctness bug I reproduced by execution: **space on a board is not the no-op the plan's D14 and the board's own doc both promise** — it emits `OutcomeReveal`, which makes the loop write a blank "reveal" into the append-only buffer and play the pronunciation of an arbitrary cell. D12 enumerated four `Apply` paths that must consult `Batch`; `InputReveal` is a fifth that was missed, and no test covers it. Secondarily, `fitsABoard` charges the keys prompt a hard-coded one row when that line is 76 columns wide, so on any terminal narrower than 76 the live edge is under-budgeted and `fitFooter` silently drops the bar, the panel, and — at width ≤ 25 — the mode toggle, which is the *one owner* of which mark is live while irreversible marks land. Both are cheap fixes with obvious tests.

## 1. Strengths

- **`Apply` wraps `apply` and stamps `Form` once** (`cmd/define/play/session.go:186-203`). Three record-building sites, one stamp; combined with `Form()` being on the `Question` interface rather than an optional capability, a new form cannot ship an unattributable promotion. `TestYAMLRoundTripsTheFormThatAsked` is the right pin — it catches the `yaml:"-"` failure that an in-memory double would have shipped green.
- **`FooterRowAt` returns the entry index, not a row offset, and is answered from the last paint** (`cmd/define/screen.go:214-243`). Recording `s.footer` *after* `fitFooter` has trimmed it is what makes "a row that was not drawn is a click on nothing" true by construction rather than by a guard. `TestFooterRowAtNamesTheEntryUnderAClick` covers the wrapped-entry and dropped-entry cases.
- **Two redundant bounds found by mutation and removed, with the property pinned separately from the mechanism** (`play_loop.go:496-501`, `play/board.go:463-471`, `TestTheToggleAndPanelRowsAreNotCells` over eight shapes × every column). This is the right response to a surviving mutant, and it is rarer than it should be.
- **`TestABoardCostsFarLessPerWordThanMeaningChoice` is a real measurement** — two scripted sittings through `playSession`, transcript lines counted, summary charged to both. R5 then moved the claim to what the numbers support instead of finding a framing where the proxy passed.
- **`boardFits` builds a probe board and asks it its height** (`play_loop.go:658-667`) rather than re-deriving the layout in the loop. One owner of the geometry, which is the same reason `CellAt` lives on the form.

## 2. Critical findings

**`cmd/define/play/session.go:319` — space on a board emits `OutcomeReveal`; the plan says it must do nothing.**

D14: *"Only a `Batch` form distinguishes them: `InputFinish` spends it, `InputReveal` does nothing, because a board has nothing to reveal."* `atlas/define.md` repeats it. The `InputReveal` case has no `Batch`/`Grid` guard, so it sets `s.Revealed = true` and returns `Outcome{Kind: OutcomeReveal, Word: q.Word()}`. Reproduced:

```
revealed=true done=false index=0
out[0] kind=2 word="alpha"      # OutcomeReveal, cells[0] — no cell was marked
```

The loop then (`play_loop.go:389-430`) writes `"\n" + asked.Reveal() + "\n"` — two blank lines into the append-only buffer — and, because `opt.playsAudio()` is true by default, calls `playAnnounced` on `out.Word`: the pronunciation of `cells[b.last]`, which is cell 0 before any mark and the last-marked word after. Space is the natural key to press here (it means "reveal" on both other forms and the board's prompt does not mention it), so this is reachable in the first real sitting.

*Fix sketch:* in `apply`'s `InputReveal` case, before `s.Revealed = true`, return `[]Outcome{{Kind: OutcomeNone}}` when `batchOf(q) != nil` (or when the form is a `Grid` — pick the capability that names the reason: a form holding many words has no single hidden word).

*Fix the class, not the instance:* D12 wrote down four `Apply` paths that must consult `Batch` and this is the fifth. The enumeration is small and finite — add a table test that drives **every** `InputKind` at a batch form and asserts the outcome set, so the next input kind added cannot skip the question. `TestEnterSpendsABatchFormAndSpaceDoesNot` currently asserts only that space does not spend or record, which is why this passed. (ARCH-PURPOSE)

## 3. Important findings

**`cmd/define/play_loop.go:506` — `boardChromeRows = 2` charges the keys prompt one row, but that line is 76 columns and the board is offered from width 20.**

`gradePrompt(board)` is `"a word's key or a click = mark, Tab = switch, Enter = finish, Ctrl-C to stop"` — 76 visible columns. `boardFits` only requires `opt.width >= minWrapWidth` (20), so on any terminal narrower than 76 the prompt occupies 2–4 rows while `fitsABoard` budgets 1. `Paint` gives `fitFooter` `termRows - promptRows`, and `fitFooter` drops from the end. Measured against the real functions:

```
w=40 r=11: boardRows=9  promptRows=2 entries=10 fitted=9 → dropped: the bar
w=36 r=12: boardRows=9  promptRows=3 entries=10 fitted=9 → dropped: the bar
w=24 r=13: boardRows=11 promptRows=4 entries=12 fitted=9 → dropped: the TOGGLE, the panel, the bar
```

At width 24 the board is offered and drawn without the toggle row — the single owner of which mark is live (`board.go:277`, *"the mode is drawn by the footer's toggle row, and that row is its one owner"*) — while every mark is irreversible. That is exactly D15's *"a board that cannot be drawn whole is not a board"* failing on its own terms. The frame stays consistent (no scroll, no misplaced click), which is why it is Important rather than Critical.

`TestABoardsPromptDoesNotOfferTheDropKey:2255` already names this failure mode — *"which wraps at 80 and makes the frame a row taller than the board was offered for"* — but only guards it at 80 columns, and `TestFitsABoardCountsTheWholeLiveEdge` only exercises width 80.

*Fix sketch:* charge the measured height, not a constant — `fitsABoard(termRows, boardRows, promptRows)` with `promptRows = displayRows(gradePrompt(probe), opt.width)`, or fold the whole computation into `boardFits` where the probe already exists. Add a width axis to `TestFitsABoardCountsTheWholeLiveEdge` (20/24/40/80) so a constant cannot come back. (ARCH-DRY: the `1` inside `boardChromeRows` is a second, implicit owner of a height `displayRows` already computes. ARCH-CONSTRAINTS: the declared envelope — *offered only when the terminal can hold it whole* — is not enforced across the width range the code accepts.)

**`workshop/plans/000040-form-board-plan.md:459,461,465,362` — the plan's Done-when table cites four tests that do not exist, and an `(R3)` with no revision entry.**

`TestEnterCommitsTheUnmarkedAsNo`, `TestSpaceDoesNotCommitABoard`, `TestANoMarkDoesNotEndTheBoard` and `TestAReviewEventNamesItsForm` are named as the pins for rows 3, 5 and 9 and are in no file; the real pins (`TestEnterSpendsABatchFormAndSpaceDoesNot`, `TestABoardRunsThroughTheSession`, `TestEveryRecordNamesItsForm`) appear only in the issue's `## Log` sweep table. Line 362 cites `(R3)` and the `## Revisions` section runs R1, R2, R4, R5.

This is the same class `TestPlanTableStatusMatchesTheChangeWindow` was built for, and the issue's own Log records the lesson one round earlier: *"a hand-sweep of a plan's tables does not hold, and this repo already knew it."* The guard reads the `name | file.go | status` rows only; the `pinned by` column is unenforced.

*Fix sketch:* correct the four names and add the missing R3 entry (or renumber), **and** extend the guard to the `pinned by` column — every backticked `Test*` identifier in an active plan must resolve to a `func Test…(` in the tree. That is the class; fixing only the four names is the instance. (ARCH-PURPOSE)

## 4. Minor findings

- `cmd/define/play/board.go:257` — *"Colour is what makes the marks pop, and the loop adds it"* is false: `boardFooter` (`play_loop.go:495`) splits `Prompt()` and appends `sittingBar` with no styling anywhere. The comment asserts behaviour that does not exist; either implement it or state it as deferred.
- `cmd/define/play_loop_test.go:2692` — `if len(boardKeys) > len(words)` cannot fail: `boardKeys` is built by `for i := range words`. The keystroke half of Done-when 13's pin asserts the test's own construction, not the board's behaviour. (The reading-cost half is real and does carry the claim.)
- `cmd/define/play/board.go:198` — `Rows()` disagrees with `Prompt()` for an empty board: `gridRows()` is 0 but `Prompt()` still yields four entries (`"", "", toggle, panel`), so `Rows()` returns 3. Unreachable today (`todaysQuestions:773` guards `len(cells) > 0`) but `Word()` and `panelLine()` both defend the empty case, so the invariant is inconsistently held. `TestNoGridLineExceedsTheWidth`'s `Rows()` assertion never sees it.
- `cmd/define/play/board.go:453` — a click in a cell's trailing padding marks that cell, including past the visible end of a `trimRight`-trimmed last cell on a row. It marks the column's own word rather than a neighbour, so it does not contradict *"a click that is not clearly on a word must not mark its neighbour"*, but clicking apparently blank space and getting a permanent mark is worth a line in the doc if it is intended.
- Ctrl-C on a board does write the relearn line (`Current()` returns nil once `Done`, so `s.Current() != asked` holds) — correct, but it is load-bearing on `Current()`'s `s.Done` check and nothing pins it. `TestCtrlCCancelsABoardWithoutMovingUnmarkedWords` asserts the fold, not the transcript line.

## 5. Test coverage notes

- **`go test ./...` is green** (107s for `cmd/define`), as is `go build ./...`.
- **`go test -tags conformance ./cmd/define/ -run TestPTYPlayBoardIsDrawnAndClickable` SKIPS here**: `no pty available: operation not permitted`, even with the sandbox off in this session. Done-when 14 therefore rests entirely on the implementor's out-of-band unsandboxed run recorded in the `## Log`; I could not independently confirm it. The test itself reads the click's row and column off the paint rather than computing them, which is the right shape for that row.
- The gap that shipped the Critical: `TestEnterSpendsABatchFormAndSpaceDoesNot/space leaves it alone` checks only `Index`/`Done` and the absence of `OutcomeRecord`. Asserting the full outcome slice (`OutcomeNone`, nothing else) would have reddened it. The general form — every `InputKind` × a batch form — is the coverage this diff needs.
- No test exercises a board at any terminal width but 80, which is why the fit under-budget survived.

## 6. Architectural notes

- **ARCH-DRY — pass, with one flag.** `gradedPrompt` now derives from `sessionKeys`, `reservedKeys` is the single owner of the reserved half, `boardFooter` is one line because the form owns its own rendering, and `spent`/`batchOf` keep four call sites reading as one question. The flag is `boardChromeRows`'s hidden `1` for the prompt (Important #1 above) — a second owner of a height `displayRows` already computes.
- **ARCH-PURE — pass.** `Board`, `Cell`, `Mark`, `Grid`, `Moded`, `Batch` are pure and import nothing; `puretest` enforces it and `purity_test.go` gained a real grep guard with a premise check. IO stays in the loop and the `display` seam; `FooterRowAt` is injected, and the editor's double answers "none".
- **ARCH-PURPOSE — one flag.** The single-source shadow-sweep passes: `Form()` is on the interface so every consumer (`Recall`, `Choice`, `Board`) derives, `doc_sync_test` makes the README derive the prompt lines rather than restate them, and the atlas section is new rather than a copy. The flag is D12's enumeration stopping at four paths when the matrix is `InputKind × Batch` — the Critical is the fifth cell, and the deliverable is the enumeration, not the one guard.
- **ARCH-MOCK — pass.** No new external dependency. The loop's click tests script `FooterRowAt` on `recordDisplay`, but the object that *joins* the screen and the form is exercised against a real pinned screen in `TestAClickOnABoardMarksIt` (premise read off the paint, not off the click map) and against a real terminal in the pty row, which is the correct application of #30's "a double may not stand in for the joining object".
- **ARCH-CONSTRAINTS — one flag.** The declared envelope (keystroke/redraw, O(16 cells) of string building per frame, `boardsFor` O(deck) once per sitting) is met; `refresh()` is still called only from the two outcomes that change it, so no O(deck) walk per keystroke was introduced. The flag is the height/width envelope not being enforced below 76 columns (Important #1).
- **For upcoming work:** `Grid` is the first capability the *loop* asks rather than `Apply`, and the loop now does the row subtraction between two answers. That is correct today because the grid is drawn as the first footer entries — a fact `boardFooter`'s comment names as load-bearing but nothing tests. A second live-edge form, or anything that wants a row above the grid, will silently shift every cell. Worth a pin (`boardFooter`'s first `Rows()` entries are the form's, in order) before the next consumer arrives.

## 7. Plan revision recommendations

Add to `workshop/plans/000040-form-board-plan.md` `## Revisions`:

- **(R6) — the Done-when table named four tests that were never written.** Rows 3, 5 and 9 cite `TestEnterCommitsTheUnmarkedAsNo`, `TestSpaceDoesNotCommitABoard`, `TestANoMarkDoesNotEndTheBoard` and `TestAReviewEventNamesItsForm`; the shipped pins are `TestEnterSpendsABatchFormAndSpaceDoesNot`, `TestABoardRunsThroughTheSession` and `TestEveryRecordNamesItsForm`. Correct the cells, and record that `TestPlanTableStatusMatchesTheChangeWindow` was extended to the `pinned by` column so a plan cannot again cite a test that does not exist.
- **(R7) — D14's "`InputReveal` does nothing on a batch form" was not implemented.** State what shipped (space emits `OutcomeReveal`, so the loop wrote a blank reveal and played an arbitrary cell's pronunciation), what the fix is, and that D12's four-path enumeration is now a per-`InputKind` table test rather than a list in prose.
- **(R8) — D15's fit condition is measured, not constant.** `boardChromeRows` charged the keys prompt one row; the prompt is 76 columns and the board is offered from 20, so `fitFooter` dropped the bar (< 76 cols), the panel (≤ 38) and the toggle (≤ 25). Record the corrected computation and the width axis added to the fit test.
- **Numbering:** the Core-concepts table cites `(R3)` at line 362 and no R3 entry exists — either add it (the `Form()`-on-the-interface decision it refers to) or renumber the citation.

```findings
findings:
  - id: new
    severity: Critical
    family: batch-capability-not-asked-on-every-path
    title: |
      Space on a board emits OutcomeReveal, so the loop plays an arbitrary cell's audio and writes a blank reveal into the buffer
    detail: |
      cmd/define/play/session.go:319 — the InputReveal case has no Batch/Grid guard, so it sets s.Revealed and returns Outcome{Kind: OutcomeReveal, Word: q.Word()} (cells[0] before any mark). Reproduced by execution. play_loop.go then writes "\n"+Reveal()+"\n" into the append-only buffer and calls playAnnounced on that word. D14 and the board's own doc both state InputReveal must do nothing for a form holding many words. D12 enumerated four Apply paths that must consult Batch; this is the fifth. Fix the class: a table test driving every InputKind at a batch form.
  - id: new
    severity: Important
    family: frame-budget-hardcoded-not-measured
    title: |
      fitsABoard charges the 76-column keys prompt one row, so a board narrower than 76 columns is drawn with its toggle, panel or bar dropped
    detail: |
      cmd/define/play_loop.go:506 — boardChromeRows = 2 assumes a one-row prompt, but boardFits only requires width >= minWrapWidth (20). Measured against the real functions: w=40 r=11 drops the bar; w=36 r=12 drops the bar; w=24 r=13 drops the TOGGLE and panel — the single owner of which mark is live, while every mark is irreversible. That is D15's own condition failing. Charge displayRows(gradePrompt(probe), width) instead of a constant, and add a width axis to TestFitsABoardCountsTheWholeLiveEdge, which only exercises 80.
  - id: new
    severity: Important
    family: plan-citations-unenforced
    title: |
      The plan's Done-when table pins three rows on four tests that do not exist, and cites an (R3) revision that was never written
    detail: |
      workshop/plans/000040-form-board-plan.md:459,461,465 name TestEnterCommitsTheUnmarkedAsNo, TestSpaceDoesNotCommitABoard, TestANoMarkDoesNotEndTheBoard and TestAReviewEventNamesItsForm; none exist. Line 362 cites (R3) and the Revisions section runs R1, R2, R4, R5. TestPlanTableStatusMatchesTheChangeWindow guards only the status column, so the pinned-by column drifted — the exact class the issue's own Log recorded one round earlier. Fix the cells AND extend the guard so every backticked Test* in an active plan must resolve to a func in the tree.
  - id: new
    severity: Minor
    family: comment-asserts-absent-behaviour
    title: |
      board.go claims "the loop adds colour" to the marks; no styling exists anywhere on the board path
    detail: |
      cmd/define/play/board.go:257. boardFooter (play_loop.go:495) splits Prompt() and appends sittingBar with no colour. D10's decision table chose option D partly because it "keeps colour"; nothing ships it. Either implement it or mark it deferred in the comment.
  - id: new
    severity: Minor
    family: assertion-cannot-fail
    title: |
      The keystroke half of Done-when 13's pin is a tautology over the test's own fixture
    detail: |
      cmd/define/play_loop_test.go:2692 — `if len(boardKeys) > len(words)` cannot fail, because boardKeys is built by `for i := range words`. The reading-cost ratio in the same test is real and does carry the claim; the keystroke floor its red-when names is unpinned.
  - id: new
    severity: Minor
    family: degenerate-input-breaks-derived-count
    title: |
      Rows() returns 3 for an empty board while Prompt() yields four lines
    detail: |
      cmd/define/play/board.go:198 — gridRows() is 0, so Rows() is chromeRows (3), but Prompt() emits "" + "\n\n" + toggle + "\n" + panel, which splits to four entries. Unreachable today (todaysQuestions:773 guards len(cells) > 0), but Word() and panelLine() both defend the empty case, so the invariant is held inconsistently and TestNoGridLineExceedsTheWidth's Rows() assertion never sees it.
  - id: new
    severity: Minor
    family: click-target-wider-than-drawn
    title: |
      A click in a cell's trailing padding marks it, including past the visible end of a trimmed last cell
    detail: |
      cmd/define/play/board.go:453 — CellAt accepts any column within labelWidth+wordCells, while Prompt trims the row's trailing blanks. It marks the column's own word rather than a neighbour, so the stated rule holds, but clicking apparently blank space and getting a permanent mark deserves a line in CellAt's doc if intended.
```

---

## Re-review — 2026-09-01T13:14:12-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 40 — form 2.5: the board — grid triage for mature words |
| repo | tools |
| issue file | workshop/issues/000040-form-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | eb9f1698d6800ff5ad22683f49e2c36e970d36f3..04b6bd466fefc3c0a31723a2ada61b58be42e6ed |
| command | sdlc close --issue 40 |
| reviewer | claude |
| timestamp | 2026-09-01T13:14:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All seven prior findings are genuinely fixed, and I verified six of them by reverting the fix in a scratch copy and watching the test go red (BR-4 is a comment-only change with nothing to pin). BR-1's fix is the right shape: `numInputKinds` is a real sentinel and `TestEveryInputKindIsAnsweredForABatchForm` fails without the guard; BR-2's `fitsABoard(termRows, boardRows, promptRows)` now measures the prompt and both fit tests gained a width axis; `TestPlanCitesTestsThatExist` fails on a fabricated citation. `go test ./...` is green (108s for `cmd/define`). What blocks SHIP is one new Critical I reproduced by execution: **a board's layout is fixed at `NewBoard(cells, opt.width)` and the terminal's is not.** After a narrowing resize the loop redraws the same board through `view.Resize` + `show()`, the 74-column grid rows wrap, and two things break at once — `FooterRowAt` correctly returns the same entry for both physical rows of a wrapped grid line while `formCell` hands `CellAt` the raw physical column, so a click on the continuation row marks a *different, permanent* mark on the wrong word; and at 24×12 `fitFooter` drops the blank, the toggle, the panel and the bar, which is BR-2's exact harm re-entering through a path `fitsABoard` cannot see. Both are the failure D15 was written against, arriving through the one door D15 only half-closed.

## 1. Strengths

- **BR-1's fix went at the enumeration, not the site.** `numInputKinds` (`cmd/define/play/session.go:63`) is the same move `choice.go`'s `numAxes` established, and `TestEveryInputKindIsAnsweredForABatchForm` asserts the full outcome slice *plus* `Revealed`/`Graded`/advance for all seven kinds — so a new kind arrives with no expectation and fails. Reverting the `InputReveal` guard reddens it immediately.
- **BR-2's fix keeps exactly one constant, and says why.** `barRows = 1` as a *minimum* because `fitFooter` drops from the end and the bar is last, while `promptRows` is charged because the prompt takes its share off the top (`play_loop.go:503-534`). That asymmetry is argued rather than asserted, and `TestFitsABoardCountsTheWholeLiveEdge:2297-2313` reaches through `boardFits` at width 40 with a `t.Fatalf` premise check so the case cannot go vacuous.
- **`TestRowsIsWhatPromptDraws` (`board_test.go:652`) makes BR-6 an invariant by construction.** `Prompt()` builds a `[]string` and `joinLines` it, so `len(lines)` *is* `Rows()` across 0–16 words × three widths. Restoring the concatenation reddens all three widths at n=0.
- **BR-5's replacement is a real predicate.** `len(boardKeys) != len(words)` is now a `t.Fatalf` on the fixture and `board.Spent()` is the claim (`play_loop_test.go:2723-2732`) — the keystroke floor is finally asserted against the board rather than against the script.
- **`TestFooterRowAtNamesTheEntryUnderAClick` (`screen_test.go:1182`) covers pinned, unpinned, wrapped, dropped and unpainted.** The dropped-entry and unpainted cases are the ones that keep "a click on a row that was not drawn is a click on nothing" true by construction.

## 2. Critical findings

**`cmd/define/play/board.go:136` + `cmd/define/play_loop.go:245,562` — a narrowing resize under a live board marks the wrong word and drops the toggle.**

`board.go:136` states the premise: *"the width it is derived from cannot change: a resize below the board's height leaves the current board drawn as it was."* D15 says the same (`plan:315`, *"its rows are already budgeted"*). Both are about **height**. The loop's resize case (`play_loop.go:245`) calls `view.Resize(sz.rows, sz.cols)` then `show()`, which redraws `boardFooter(q, fig)` — a board laid out for the *old* width — at the new one. `Paint` clips the prompt and the buffer but writes footer entries raw (`screen.go:494-496`), so a grid row wider than the new `cols` wraps, and `fitFooter` budgets its wrapped height.

Measured against the real functions, board of 8 words built at 80:

```
grid row 0 (74 cols): "[0] arrondissement  [1] bailiwick       [2] sycophantic     [3] obsequious"
displayRows(grid row 0, 40) = 2          # two physical rows after narrowing to 40
CellAt(0, 4) = 0   CellAt(0, 44) = 2     # col 44 is "[2] sycophantic"
```

`FooterRowAt` returns entry 0 for **both** physical rows (its own `screen_test.go:1220` subtest pins that a wrapped entry owns every row it occupies). `formCell` then does `g.CellAt(row, k.Col)` with the raw physical column (`play_loop.go:562`) — so a click at column 4 of the *second* physical row, where the terminal is showing `sycophantic`, marks `arrondissement`. The mark is irreversible and the event is written immediately.

The same resize also re-opens BR-2. Painting that board on a 24×12 terminal produces:

```
a word's key or a click = mark, Tab = switch, Enter = finish, Ctrl-C to stop
[0] arrondissement  [1] bailiwick       [2] sycophantic     [3] obsequious
[4] potassium       [5] ligament        [6] concrete        [7] parrot
```

— the blank, the TOGGLE, the panel and the bar are all gone. The toggle is the one owner of which mark is live.

> **This is the 2nd finding in family `frame-budget-hardcoded-not-measured`.** BR-2 fixed the instance where the prompt's height was a constant. Do not fix this instance alone. **The rule:** *every quantity the board's fit and its click map depend on must be read from the terminal as it is at draw/click time, never from a value fixed when the form was chosen.* The enumeration that rule implies, swept in one round:
> 1. prompt height at selection — **fixed** (BR-2);
> 2. the board's own layout width — **open**: `NewBoard` freezes `cols`/`cell`/`wordCells` and nothing relayouts on `Resize`;
> 3. the fit decision itself — **open**: `fitsABoard` runs once in `boardsFor`, never again;
> 4. the column translation for a wrapped footer entry — **open**: `formCell` passes a physical column into a function that expects an entry-relative one, and the screen already models entries as wrapping.
>
> *Fix sketch:* the cheapest correct shape is for the board to carry its build width and for the loop to invalidate on mismatch — either a `Board.Relayout(width)` that preserves `marks`/`mode`/`last` (cell indices are stable, so this is a `layout()` re-run) called from the resize case after `view.Resize`, plus a re-run of `fitsABoard` that falls back to a plain "this board no longer fits" state; or, minimally, `formCell` refusing every click while the drawn width differs from the build width, so a stale board is read-only rather than mis-clickable. Either way item 4 wants a real pin: `FooterRowAt` should report the column offset within the entry, or `formCell` should refuse a row that is not the entry's first. Pin with a loop-level test that drives a resize under a board and asserts the mark lands on the word the frame actually shows at that row/column — no test in the diff exercises a board at any width other than the one it was built for.

## 3. Important findings

**`cmd/define/repo_guard_test.go:1175-1178` — `TestPlanCitesTestsThatExist` silently disarms itself on exactly the document shape that produced BR-3.**

The fix round found the real mechanism and wrote it down (`plan:448-455`, issue Log): `currentTruthOnly` cuts a plan at its **first** `## Revisions`/`## Log`, this plan had two, and both plan guards had been reading a truncated file and passing. The round then fixed the *plan* — merging the revisions into one appended section — and left the truncation unguarded. Verified by execution: reinserting a `## Revisions` heading above `## Core concepts` and planting `TestSpaceDoesNotCommitABoard` in the Done-when table leaves `TestPlanCitesTestsThatExist` **PASS** (its `checked == 0` branch is a `t.Skip`), where without the truncation it fails with the right message.

Measured prevalence: `currentTruthOnly` has **8 call sites** in `repo_guard_test.go` (590, 688, 856, 1010, 1082, 1166, 1423); **four** of the guards reading it end in a `checked == 0` → `t.Skip` (729, 881, 1106, 1178); **none** asserts that the surviving text still contains the sections it is supposed to check. One misplaced heading disarms all four for that file, and the only reason it was caught last round was luck while mutation-checking an unrelated guard.

> **This is the 2nd finding in family `plan-citations-unenforced`.** Do not fix this instance. **The rule:** *a guard that reads a filtered view of an artifact must assert its own premise about that view and FAIL — never Skip — when the filter removed the thing it exists to check.* The repo already applies this rule elsewhere: lines 167, 322, 355, 445, 531, 598 and 1156 all `t.Fatal("…would pass vacuously")`. The four plan guards are the ones that don't. Concretely: have `currentTruthOnly` (or its callers) fail when a `## Revisions`/`## Log` heading precedes `## Done when` or `## Core concepts` in an active plan, and turn the `checked == 0` skips into failures for a plan that demonstrably contains the section being checked.

## 4. Minor findings

- `cmd/define/play/session.go:474` — `Batch`'s doc still says *"It is consulted at FOUR points… the other three were found by measurement"* and lists three (miss branch, `InputDrop`, `InputFinish`), omitting the `InputReveal` path added twelve lines of comment earlier in the same commit. `workshop/plans/000040-form-board-plan.md:378` (*"the FOUR places `Apply` consults it"*), and the Core-concepts rows at `:326` (`Batch` — "asked at FIVE points") and `:327` (`Apply` — "advance, miss-on-hidden, drop, Enter") say the same. **This is the 2nd finding in family `comment-asserts-absent-behaviour`.** BR-4 fixed the colour comment. The rule that covers both: *prose that restates a set the code owns is a second owner and drifts — state the count once, or derive it.* Here the derivation already exists (`numInputKinds` + the table test); the four prose sites should point at it rather than re-count. Records (the issue Log, the close-review sidecar) legitimately keep "four" and need no change.
- `cmd/define/play/session_test.go:863` — the expectation struct declares `reveals bool`, which no table entry sets and nothing reads. It reads as a checked dimension that isn't one; `next.Revealed` is asserted unconditionally instead. Family `test-declares-unchecked-expectation` (new).
- `cmd/define/play/board.go:407` — `Grade` scans all 16 `boardLabels` regardless of `len(cells)`; harmless, and `Mark` bounds-checks, but the loop bound is the alphabet rather than the board.

## 5. Test coverage notes

- `go test ./...` green (exit 0; `cmd/define` 107.6s). `go build ./...` clean.
- `go test -tags conformance ./cmd/define/ -run TestPTYPlayBoardIsDrawnAndClickable` **SKIPS here**: `no pty available: operation not permitted`, sandbox off. Same as round 1. Done-when 14 still rests on the implementor's out-of-band run recorded in the `## Log`; I could not independently confirm it. That is `#37`'s subject.
- Mutation-verified this round (fix reverted in a scratch copy, test observed red): BR-1 → `TestEveryInputKindIsAnsweredForABatchForm`; BR-2 → `TestFitsABoardCountsTheWholeLiveEdge` **and** `TestAShortTerminalGetsMeaningChoiceNotAClippedBoard/very narrow`; BR-3 → `TestPlanCitesTestsThatExist`; BR-6 → `TestRowsIsWhatPromptDraws`; BR-7 → `TestACellsPaddingBelongsToIt` (both halves: gutter-becomes-target and label-only-target).
- **The gap that ships the Critical:** every board test builds the board at the same width the screen is given. There is no test in the diff where those two numbers differ, and no test drives `con.resizes` while a board is current. That single axis is what the Critical needs.

## 6. Architectural notes

- **ARCH-DRY — pass, one flag.** `reservedKeys` is the single owner of the reserved half, `gradedPrompt` derives from `sessionKeys`, `boardFooter` is one line, and `boardFits` now asks `displayRows` instead of re-deriving. `joinLines` is hand-rolled but justified — the package's purity guard declares an empty import allowlist. The flag is the stale four/five prose (Minor #1): four sites restating a set the code now derives.
- **ARCH-PURE — pass.** `Board`, `Cell`, `Mark`, `Grid`, `Moded`, `Batch` import nothing; `puretest` enforces it and `TestTheSessionNamesNoForm` is a grep with a real premise check on the five capabilities. `FooterRowAt` is injected through the `display` seam and the editor's double answers "none".
- **ARCH-PURPOSE — flag.** Two of the three blockers were answered at the class: BR-1 with a derived enumeration, BR-2 with a measured budget. The third stopped at the instance — the `currentTruthOnly` truncation is named in the plan as a convention note and left unguarded (Important above), which is a hand-maintained restatement standing in for enforcement. The shadow-sweep on `Form()` still passes: it is on the interface, all three forms derive, `doc_sync_test` makes the README derive, `Fold` ignoring it is pinned.
- **ARCH-MOCK — pass.** No new external dependency. The click seam's joining object is exercised against a real pinned screen with the premise read off the paint (`TestAClickOnABoardMarksIt:2156-2165`), which is the correct application of #30's rule; the live conformance check exists but cannot run here.
- **ARCH-CONSTRAINTS — flag, and it is the Critical.** The declared envelope — *offered only when the terminal can draw it whole* — is enforced once, in `boardsFor`, and never re-asserted. The terminal is an input that changes at runtime and the loop already has a handler for that change; the board is the one thing on screen whose geometry is frozen against it. Representative measurement covers exactly one width.
- **For upcoming work:** `boardFooter`'s "the form's rows come first, in order" is now pinned (`TestBoardFooterPutsTheFormsOwnRowsFirst`), which is good. The next thing a second live-edge form will hit is the column half of the same contract: `formCell` assumes one footer entry occupies exactly one physical row, and the screen has never promised that. Fixing it as part of the Critical buys the next form the invariant for free.

## 7. Plan revision recommendations

- **(R9) — D15's fit is a selection-time decision, and the terminal is not.** Record that `NewBoard` freezes the layout while `view.Resize` changes the width it was frozen against; that a narrowing resize wraps the grid rows, which makes `formCell`'s column translation wrong and lets `fitFooter` drop the toggle; and what shipped instead (relayout, or clicks refused while the width is stale). D15's current sentence — *"A resize below the minimum mid-sitting leaves the current board drawn as it was — its rows are already budgeted"* — is true of height and false of width, and should say so.
- **Core concepts / T1 — five, not four.** `:326` (`Batch`, "asked at FIVE points"), `:327` (`Apply`, the four-path list) and `:378` (T1, "the FOUR places") all predate R7's fifth path. Correct them, and point at `numInputKinds` + `TestEveryInputKindIsAnsweredForABatchForm` as the owner of the count so the cells stop being a second one.
- **Amend R6.** It claims the class was fixed by `TestPlanCitesTestsThatExist`. Record that the guard is defeated by the very truncation the section's own preamble describes, and what was done about it — either the premise check, or an explicit statement that the class remains open with the blockquote as its only defence.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by reverting the InputReveal batch guard: TestEveryInputKindIsAnsweredForABatchForm goes red on kind 1 and on Revealed.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by restoring the constant: TestFitsABoardCountsTheWholeLiveEdge and TestAShortTerminalGetsMeaningChoiceNotAClippedBoard/very-narrow both go red.
  - id: BR-3
    disposition: addressed
    note: |
      All 25 cited tests resolve; planting a fake citation reddens TestPlanCitesTestsThatExist. The guard's own vacuity is raised separately.
  - id: BR-4
    disposition: addressed
    note: |
      board.go:255 now says NO COLOUR YET and marks it deferred; comment-only, nothing to pin.
  - id: BR-5
    disposition: addressed
    note: |
      The fixture equality is now a t.Fatalf and the claim is board.Spent(), which can fail.
  - id: BR-6
    disposition: addressed
    note: |
      Verified by restoring the concatenation: TestRowsIsWhatPromptDraws goes red at n=0 for all three widths.
  - id: BR-7
    disposition: addressed
    note: |
      CellAt's doc states the padding rule and TestACellsPaddingBelongsToIt reddens under both a wider and a narrower hit box.
findings:
  - id: new
    severity: Critical
    family: frame-budget-hardcoded-not-measured
    title: |
      A narrowing resize under a live board marks the wrong word on a click and drops the toggle, panel and bar
    detail: |
      SECOND finding in family frame-budget-hardcoded-not-measured — do not fix this instance alone. board.go:136 claims "the width it is derived from cannot change"; play_loop.go:245 changes it via view.Resize and redraws the same board. Measured: a board built at 80 has 74-column grid rows, so displayRows(row, 40) = 2; FooterRowAt returns the same entry for both physical rows (screen_test.go:1220 pins that), and formCell (play_loop.go:562) passes the raw physical column into CellAt — CellAt(0,4)=cell 0 while column 44 of that line is "[2] sycophantic", so a click on the continuation row lands a permanent mark on the wrong word. Painting the same board at 24x12 drops the blank, the TOGGLE, the panel and the bar, which is BR-2's harm through a door fitsABoard cannot see. The rule: every quantity the board's fit and click map depend on must be read from the terminal as it is at draw/click time, never fixed at selection time. Enumeration - prompt height (fixed), board layout width (open), the fit re-check after resize (open), the column translation for a wrapped footer entry (open). No test in the diff draws a board at a width other than the one it was built for.
  - id: new
    severity: Important
    family: plan-citations-unenforced
    title: |
      TestPlanCitesTestsThatExist skips silently when currentTruthOnly truncates the plan, which is the shape that produced BR-3
    detail: |
      SECOND finding in family plan-citations-unenforced — do not fix this instance. The fix round diagnosed the real mechanism (currentTruthOnly cuts at the FIRST "## Revisions", this plan had two, both plan guards read a truncated file and passed) and then fixed only this plan's layout. Verified by execution: reinserting a "## Revisions" heading above "## Core concepts" and planting a fabricated Test name in the Done-when table leaves TestPlanCitesTestsThatExist PASS via its checked==0 t.Skip at repo_guard_test.go:1178. Measured prevalence - currentTruthOnly has 8 call sites in repo_guard_test.go; four of the guards reading it end in checked==0 t.Skip (729, 881, 1106, 1178); none checks that the surviving text still contains the section it exists to check, while seven other guards in the same file do Fatal on vacuity. The rule: a guard reading a filtered view of an artifact must assert its premise about that view and FAIL, never Skip, when the filter removed the thing it checks.
  - id: new
    severity: Minor
    family: comment-asserts-absent-behaviour
    title: |
      The Batch doc still says the capability is consulted at FOUR points and lists three, omitting the InputReveal path added in the same commit
    detail: |
      SECOND finding in family comment-asserts-absent-behaviour — state the rule rather than patching the site. cmd/define/play/session.go:474 says "It is consulted at FOUR points… the other three were found by measurement" and enumerates the miss branch, InputDrop and InputFinish; InputReveal is now a fifth. The plan repeats it at :378 ("the FOUR places Apply consults it") and in the Core-concepts rows at :326 and :327. This is the same prose enumeration whose incompleteness caused BR-1, left stating the wrong number by the commit that declared prose the culprit. The rule: prose restating a set the code owns is a second owner and drifts — point at numInputKinds and the table test instead of re-counting. Records (issue Log, close-review sidecar) legitimately keep "four".
  - id: new
    severity: Minor
    family: test-declares-unchecked-expectation
    title: |
      The InputKind table's expectation struct declares a `reveals` field that no case sets and nothing reads
    detail: |
      cmd/define/play/session_test.go:863 — `reveals bool` sits beside `kinds` and `advances` and reads as a third checked dimension. Revealed is in fact asserted unconditionally for every kind, so the field is dead; either drop it or make the reveal expectation per-case.
```
