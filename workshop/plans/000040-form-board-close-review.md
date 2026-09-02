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

---

## Re-review — 2026-09-01T13:42:55-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 40 — form 2.5: the board — grid triage for mature words |
| repo | tools |
| issue file | workshop/issues/000040-form-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | eb9f1698d6800ff5ad22683f49e2c36e970d36f3..ffdb93580efd301d6819d2f2054d6289c1108374 |
| command | sdlc close --issue 40 |
| reviewer | claude |
| timestamp | 2026-09-01T13:42:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The board is a well-built form and round 2's two blockers were fixed at the seam, not the site: I verified both by revert-mutation — removing `g.Resize(sz.cols)` reddens `TestANarrowingResizeKeepsTheBoardsClickMapHonest`, removing the `offset != 0` refusal reddens `TestFormCellAsksTheScreenAndTheForm/a wrapped entry's continuation row`, no-op'ing `Board.Resize` reddens `TestABoardRelaysOutForTheWidthItIsDrawnAt`, and the R10 `currentTruthOnly` guard fails loudly when I plant BR-3's exact shape (a `## Revisions` above `## Core concepts` plus a fabricated `Test*` name). BR-8's Critical half — a permanent mark on the wrong word — is genuinely closed. What is not closed is the rest of BR-8's own enumeration, and I measured it: the relayout is applied **at resize time to the current form only**, so a board that becomes current later is drawn at its selection-time width (73 columns on a 40-column terminal, reproduced end-to-end through `playSession`); and after a shortening resize `fitFooter` still drops the toggle — the one owner of which mark is live — at 10x40 and 12x24, which the plan (R9), the atlas and `boardFooter`'s own comment all say cannot happen. Full suite green (`go test ./...`, 108s); pty rows SKIP here (`no pty available`, #37).

## 1. Strengths

- **Two defences, both real and both pinned.** `Board.Resize` closes it at the root and `formCell`'s `offset != 0` closes it at the seam (`cmd/define/play_loop.go:576`), and each has a test that goes red without it — I checked all three by reverting. The comment at `play_loop.go:568-575` explaining why the seam guard exists *even though it should never fire* is the right instinct for an irreversible action.
- **`FooterRowAt` grew the offset without widening its contract** (`cmd/define/screen.go:239-251`). The index still belongs to the screen (which wrapped the entries) and the caller gets exactly the one extra fact it needs to refuse a column it cannot place. `TestFooterRowAtNamesTheEntryUnderAClick/a wrapped entry owns every row it occupies, and says WHICH` pins the offset per row.
- **The R10 fix went to the filter, not the file** (`repo_guard_test.go:624-647`). One check covers all eight call sites; I planted the swallowing shape in a scratch plan and three separate guards failed naming the swallowed section. `TestPlanCitesTestsThatExist` now distinguishes "nothing to check" from "I was handed nothing" (`:1197-1215`).
- **The `numInputKinds` matrix test is the shape D12 should have had** (`play/session_test.go:857-915`): the `len(want) != numInputKinds` fatal means a new kind arrives with no expectation and fails, and `Revealed`/`Graded` are asserted unconditionally rather than per case — which is also how BR-11 got closed properly instead of by deleting a field.
- **Docs are complete for the surface**: README gains the board's frame, its keys, the click, Tab, the split Enter and the `d`-is-not-offered note; `atlas/define.md` gains a full form 2.5 section including the five capabilities and the measured cost. `doc_sync_test.go` now drives the board's prompt line against the README, which is the one row that can catch `reservedKeys` regressing.

## 2. Critical findings

None new. BR-8's Critical mechanism (a click marking the wrong word) is closed and verified; see the disposition below for the half that is not.

## 3. Important findings

**BR-8, disposed `not-addressed` — the enumeration it demanded is two rows short.** BR-8's rule was *"every quantity the board's fit and click map depend on must be read from the terminal as it is at draw/click time, never fixed at selection time"*, with four rows: prompt height (fixed), layout width, the fit re-check, the wrapped-entry column. Measured against the shipped code:

- **Layout width is read at RESIZE time, for `s.Current()` only** (`cmd/define/play_loop.go:256-258`). A board later in the queue never hears about it. Reproduced end-to-end through `playSession` with a `[Recall, Board]` queue and a `winSize{24,40}` on the resize channel: after the single is answered, the board is drawn with rows of **73–75 columns on a 40-column terminal**. Singles come first by design (`boardsFor`), so *any* resize during the retrieval phase leaves every board in the sitting stale. Harm is bounded by the seam guard — no wrong-word mark — but half of every grid row is drawn and silently unclickable, on a form whose own doc says a key that stops working without saying so is what a learner blames themselves for.
- **The fit is not re-checked after a resize, and the argument for that is measurably wrong.** The plan's R9 table (`workshop/plans/000040-form-board-plan.md:624`) says *"`fitFooter` drops trailing entries, which are grid rows that are neither drawn nor clickable"*. `boardFooter` puts the form's rows FIRST, so the trailing entries are the bar, the panel and then the **toggle**. Painted through the real screen with the loop's own sequence (`view.Resize` → `b.Resize` → `Draw(gradePrompt, boardFooter)`): at **12x40** the panel and bar drop; at **10x40** and **12x24** the toggle drops — and at 12x24 only 8 of 16 words are drawn at all, while the undrawn ones are still markable blind and still taken as `no` by Enter. That is BR-2's harm arriving through the door BR-8 named.

*Fix sketch — one place, which is what makes it the rule rather than the site:* keep the current terminal size in `playSession` (initialised from `opt.rows`/`opt.width`, updated in the resize case) and have `show()` relayout the current form from it before drawing — `if g, ok := q.(play.Grid); ok { g.Resize(cols) }` — **replacing** the resize-case call rather than joining it, so there is one owner. Then decide the short-terminal case explicitly instead of by fitFooter's tail-drop: either the board trims its own grid rows to an available height it is told (keeping `boardFooter`'s index→grid-row identity intact, which trailing-drop preserves), or the loop re-checks `fitsABoard` at draw time and the plan says what happens when it fails. Either way the claim in `play_loop.go:509-511`, `atlas/define.md:2068` and R9 has to match what fitFooter actually drops. (ARCH-PURPOSE, ARCH-CONSTRAINTS)

**`cmd/define/play_loop_test.go:3101-3106` — the loop-level pin for R9's click map asserts over an empty set, always.** *This is the 2nd finding in family `assertion-cannot-fail`.* The rule, not the instance: **a test whose subject is an event must assert the event happened; `for _, e := range events` with no count check certifies nothing.** Measured — instrumented with a count and run three times, the test records **0 review events every time**, and it still passes when I stub `formCell` to return `false` unconditionally. The click never lands because the goroutine derives its row from `strings.Split(frame, "\r\n")`, i.e. from logical writes, while `FooterRowAt` works in *physical* rows — the 76-column keys prompt wraps to two rows at width 40, so every row sent is one short and `FooterRowAt` answers `false`. This is the same lesson the issue's Log already records for T13 (*"the click's ROW and COLUMN are read off the paint rather than computed"*), unapplied one test over. What the test does pin (the relayout, the row widths) is real and does fail without the fix; only the mark assertion is dead.

*Fix sketch:* walk the physical rows of the last frame (the frame is `cursorHome+eraseDown` delimited and each written row is `\r\n`-terminated, so wrapped entries must be expanded by `displayRows` — or simply probe `live.FooterRowAt(r)` for the row whose entry is grid row 0 and take the column from the drawn text), then assert `len(reviewEvents(t, st)) == 1` before checking the word. Also drop the dead `_ = i` and the `HasPrefix ||` disjunct subsumed by `Contains` at `:3070-3077`.

## 4. Minor findings

- *3rd finding in family `comment-asserts-absent-behaviour`* — the rule: **prose restating a set the code owns is a second owner and drifts; point at the type instead.** Two live instances: the plan's Core-concepts `Grid` row (`plan.md:327`) still enumerates `Rows`, `CellAt`, `Mark` while the interface has four methods since R9 added `Resize`; and `boardFooter`'s comment (`play_loop.go:509-511`) plus `atlas/define.md:2068` still assert *"a board is never IN a footer that has to drop anything"*, which the measurement above contradicts. Neither is guarded — `TestPlanTableStatusMatchesTheChangeWindow` reads the `name | file | status` cells only.
- *3rd finding in family `plan-citations-unenforced`* — `currentTruthOnly` has **two** discarding rules and R10 gave a premise assertion to one of them. The second (`repo_guard_test.go:640-646`, dropping any `### ` section containing `**closed:**`) still discards silently, and because it splits on `"\n### "` a closed section swallows everything up to the *next* `### ` — which can include a following `## ` top-level section. No live instance in the tree today (`workshop/projects/define-learn.md` is the only artifact with `**closed:**`, and its closed sections are already below the `## Log` truncation), so this is a note rather than a defect: the rule R10 wrote applies to every rule the filter has, not just the one that bit.
- `Board.toggleLine()` is a fixed 19 columns, so after `Resize(cols)` below 19 the board's own "no line wider than the width" invariant is false for the toggle row. Harmless today (grid rows come first, so nothing shifts) and unreachable at selection time (`minWrapWidth`), but `TestABoardRelaysOutForTheWidthItIsDrawnAt` exercises 24 as its narrowest width, so the invariant is unpinned exactly where it stops holding.
- README: *"A word you have recalled three times or more is swept on a grid"* — `boardBox` is a box, not a recall count; a lapsed word can be at box 3 with many more than three recalls behind it.

## 5. Test coverage notes

- Round 2's three code fixes each have a test that fails without them; I verified all three by reverting rather than by reading. The R10 guard fix I verified by planting BR-3's shape in a scratch plan — three guards failed naming the swallowed section, where round 1's version passed.
- The gap is the composition: nothing pins **a click on a relaid-out board through a real screen**. `TestABoardRelaysOutForTheWidthItIsDrawnAt` pins `CellAt` against the board's own text, `TestFormCellAsksTheScreenAndTheForm` pins the refusal against a fake view, and the loop-level test that was supposed to join them is vacuous (above). `TestPTYPlayBoardIsDrawnAndClickable` does read the click off the paint, but it does not resize and it SKIPS in this environment (`no pty available: operation not permitted` for all 20 pty rows — #37's subject, not this issue's), so Done-when 14 again rests on the implementor's out-of-band run.
- Nothing draws a board through `screen.Paint` after a size change and asserts what survived. That one test would have caught the dropped toggle, and it is the pin the fit-after-resize row needs.

## 6. Architectural notes

- **ARCH-DRY — pass.** No duplicated layout arithmetic: `boardFits` builds a probe and asks it, `fitsABoard` takes a measured `promptRows`, and the two redundant bounds found by mutation were removed rather than kept "for safety". One forward-looking caution: if the draw-time relayout is *added* beside the resize-case call rather than replacing it, the width will have two owners — the failure mode this issue keeps paying for.
- **ARCH-PURE — pass.** `board.go` imports nothing (empty allowlist, `puretest`), `Apply`/`apply` stay a pure state machine, and the new mutation (`Resize`) is state on the form driven from the thin loop. `TestTheSessionNamesNoForm` reads `session.go` off disk and strips comments — a grep guard that can actually fail, with its own premise check.
- **ARCH-PURPOSE — flag.** The shadow sweep is where this boundary is short. R9 wrote the enumeration out (four quantities) which is exactly the right move; the sweep then closed two rows and argued the other two away. A finding's deliverable is the class, and the class here is enumerated and small — closing it means the draw seam, not the resize event.
- **ARCH-MOCK — pass.** No new external dependency. The `store.NewYAML` round trip is the right pin for a persistence claim (the `yaml:"-"` class), and `seedMature` writes the fixture through the production writer rather than by hand. The live conformance check exists (`TestPTYPlayBoardIsDrawnAndClickable`) but cannot run in a review environment, which is a known, filed gap.
- **ARCH-CONSTRAINTS — flag.** The declared envelope is *"a board that cannot be drawn whole is not a board"*. It is enforced at selection time and, after R9, horizontally at resize time; it is **not** enforced vertically after a resize, and the measurements above are the envelope being silently exceeded (toggle gone at 10x40; half the words undrawn at 12x24) with no bounded behaviour declared for it. Per-frame cost is unchanged and fine — `Prompt()` is O(16 cells) at keystroke rates.

## 7. Plan revision recommendations

Add an `## Revisions` entry (R11) covering:

1. **R9's "the fit after a resize" row is wrong about what drops.** `boardFooter` puts the form's rows first, so `fitFooter` drops the bar, then the panel, then the toggle, then the blank, and only then grid rows. Replace the row with the measurement (12x40 → panel + bar; 10x40 → toggle, panel, bar; 12x24 → everything below the 8th grid row) and state the chosen bounded behaviour.
2. **R9's "board layout width" row overstates where the relayout happens.** It says the width is read at draw time; the code reads it in the resize case for `s.Current()` only. Either say that, or move the call to `show()` and say *that*.
3. **Core-concepts `Grid` row (`:327`)** must list `Resize` — the interface gained a fourth method in R9.
4. **The two prose claims that now contradict measurement**: `play_loop.go:509-511` and `atlas/define.md:2068` (*"a board is never IN a footer that has to drop anything"*).

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      wrong-word click closed and pinned; the layout-width and fit-after-resize rows of its own enumeration are still open — measured
  - id: BR-9
    disposition: addressed
    note: |
      verified by planting BR-3's shape in a scratch plan: three guards now FAIL naming the swallowed section
  - id: BR-10
    disposition: addressed
    note: |
      the Batch doc points at numInputKinds and the table test; remaining "four" mentions are records or the true batchOf call-site count
  - id: BR-11
    disposition: addressed
    note: |
      the reveals field is gone and Revealed is asserted unconditionally for every kind
findings:
  - id: new
    severity: Important
    family: assertion-cannot-fail
    title: |
      TestANarrowingResizeKeepsTheBoardsClickMapHonest asserts over an empty event set, so R9's click-map claim is unpinned at the loop
    detail: |
      2nd finding in family assertion-cannot-fail. The rule, not the instance: a test whose subject is an
      EVENT must assert the event happened — `for _, e := range events` with no count check certifies nothing.
      Measured: instrumented and run three times, the test records 0 review events every time, and it still
      passes when formCell is stubbed to return false unconditionally. The click never lands because the
      goroutine derives its row from strings.Split(frame, "\r\n") — logical writes — while FooterRowAt works in
      physical rows, and the 76-column keys prompt wraps to two at width 40. Same lesson the issue's Log
      records for T13 ("the click's ROW and COLUMN are read off the paint rather than computed"), unapplied
      one test over. The relayout and row-width halves of the test are real and do fail without the fix.
  - id: new
    severity: Minor
    family: comment-asserts-absent-behaviour
    title: |
      Two live prose enumerations restate sets the code owns: the plan's Grid row omits Resize, and boardFooter plus atlas claim a board is never in a footer that drops rows
    detail: |
      3rd finding in family. The rule: prose restating a set the code owns is a second owner and drifts —
      point at the type. plan.md:327 lists Rows, CellAt, Mark for an interface that has had four methods since
      R9 added Resize. play_loop.go:509-511 and atlas/define.md:2068 assert "a board is never IN a footer that
      has to drop anything", which measurement contradicts after a resize (toggle dropped at 10x40 and 12x24).
      Neither is guarded: TestPlanTableStatusMatchesTheChangeWindow reads the name|file|status cells only.
  - id: new
    severity: Minor
    family: plan-citations-unenforced
    title: |
      currentTruthOnly has two discarding rules and R10 gave a premise assertion to only one
    detail: |
      3rd finding in family. The closed-section rule (repo_guard_test.go:640-646) still discards silently, and
      because it splits on "\n### " a closed section swallows everything up to the NEXT "### " — which can
      include a following top-level "## " section. No live instance in the tree today (define-learn.md's closed
      sections already sit below the ## Log truncation), so this is the rule R10 wrote applied to every rule the
      filter has, rather than a present defect.
```

---

## Re-review — 2026-09-01T14:45:56-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 40 — form 2.5: the board — grid triage for mature words |
| repo | tools |
| issue file | workshop/issues/000040-form-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | eb9f1698d6800ff5ad22683f49e2c36e970d36f3..0a9cc48fcfb04a9aa26ee6a4348f88d4f71daee0 |
| command | sdlc close --issue 40 |
| reviewer | claude |
| timestamp | 2026-09-01T14:45:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3's three closable findings are genuinely closed and I verified each by reverting the fix and watching a test go red: stubbing `formCell` to return false, or removing `g.Resize(sz.cols)`, both redden `TestANarrowingResizeKeepsTheBoardsClickMapHonest` (fast, and it no longer hangs — the driver's `defer close(keys)` is what fixed that); reverting `closedSection` to `strings.Contains` makes `currentTruthOnly`'s new premise assertion fire on the real `atlas/repo-guards.md`, which is how the months-old blindness was found, and with the fix in place a planted `deckDeps` on that page's head section is now caught by `TestNoArtifactNamesARetiredSymbol` where it previously was not. `go test ./...` is green (107s for `cmd/define`); `-race` green on the board rows; the resize test is stable over 10 runs. Moving the mode onto the prompt row is the right call and is argued from `Paint`'s clipping order rather than asserted. What keeps this from SHIP is that **BR-8's own enumeration is still two rows short, and the round closed those two rows in prose instead of in code**: I reproduced, end-to-end through `playSession`, a board that becomes current *after* a resize being painted with 73-column grid rows into a 40-column terminal, and — at 12×24 — a board whose grid shows only cells `[0]`–`[7]` while `Enter` records `Wrong` for all eight words the frame does not show. The plan (R11), `atlas/define.md` and `boardFooter`'s comment all now say those losses are harmless; the measurement says one of them halves a box for a word the learner cannot see.

## 1. Strengths

- **The mode moved to the row that survives, and the reason is mechanical rather than aesthetic** (`cmd/define/play/board.go:376-400`, `chromeRows` at `:261`). `Paint` clips the prompt only when it alone exceeds the terminal (`screen.go:466-470`) while `fitFooter` drops from the end, so "what will my next click mean" is now the last thing lost rather than the first. One owner either way — `board_test.go:170-175` fails if `Prompt()` says `marking` a second time.
- **The resize test now asserts its own premise and reads everything off the screen** (`play_loop_test.go:3110-3145`). Detecting the relayout by *entry count* rather than by frame text is the right instrument: the 40-column grid row is a prefix of the 80-column one, so a text match could not distinguish the stale state — which is the test's own subject one layer up. `len(evs) != 1` at `:3157` is the check whose absence made the previous version certify nothing.
- **The driver closes its channel with `defer`** (`play_loop_test.go:3078`). A helper goroutine that `t.Fatal`s is a `Goexit`; without the close, `playSession` blocks forever and the test hangs on exactly the defect it exists to catch. I confirmed both mutations now fail in under 3s.
- **R13's premise assertion was added on the reviewer's word that no live instance existed, and immediately found one** (`repo_guard_test.go:641-651`, `closedSection` at `:675`). `atlas/repo-guards.md` documents the `**closed:**` marker, so `strings.Contains` matched its prose and discarded the guard inventory from every guard reading current truth. Matching the marker at line start is the correct narrowing, and the page now counts its examples instead of naming them.
- **The pty row derives its expectation from the form** (`pty_conformance_test.go:1001-1007`, `:1075-1079`). A conformance row's subject is that the board reached a real terminal, not how its prompt is phrased; the phrasing is pinned once against the README by `TestREADMEQuotesThePromptsTheLoopActuallyPrints`.

## 2. Critical findings

None new. See BR-8's disposition below for the half that is still open — its Critical mechanism (a permanent mark on the wrong word) remains closed and I re-verified both defences.

## 3. Important findings

**BR-8, disposed `not-addressed` — the relayout is bound to the resize *event*, not to the *draw*, and no bounded behaviour was chosen for a board that stops fitting.**

*This is the 3rd finding in family `frame-budget-hardcoded-not-measured`.* The rule was already written correctly in R9 — *every quantity the board's fit and click map depend on must be read from the terminal as it is at draw and click time, never fixed at selection time* — and two of its four rows are closed. Do not patch either instance below on its own; the two share one cause and one fix site.

- **Row 2 (layout width) — open.** `g.Resize(sz.cols)` is called at `cmd/define/play_loop.go:257`, inside the resize case, on `s.Current()` only. A board still in the queue never hears about it. Reproduced through `playSession` with a `[Recall, Board]` queue and a `winSize{24, 40}` while the recall is current: after the single is answered, the board's row 0 is `"[0] arrondissement  [1] sycophantic     [2] defenestrate    [3] ephemeral"` — **73 columns, painted into a 40-column terminal**, confirmed by finding that exact string in the emitted frames. `boardsFor` puts singles first by design, so *any* resize during the retrieval phase leaves every board in the sitting stale. No wrong-word mark — `formCell`'s `offset != 0` refusal (`play_loop.go:599`) holds, and I checked that a click on the visible first physical row still resolves correctly — so the harm is that half of every grid row is drawn and silently unclickable, on a form whose own doc says a key that stops working without saying so is what a learner blames themselves for.
- **Row 3 (the fit after a resize) — a decision was made, but not this one.** R11 chose "no re-selection", which is right and well argued. What was not decided is what happens to grid rows `fitFooter` drops. Measured through the real screen and loop: a 16-word board resized to **12×24** paints cells `[0]`–`[7]` only; **14×24** paints `[0]`–`[9]`. Pressing `Enter` then spends the board and records `Wrong` — `box/2` — for every word the frame does not show. In the compounded case (resize to 12×24 while a single is current, board becomes current afterwards at its stale 80-column layout) those eight words are never drawn on the grid at all and appear on screen for the first time in the relearn line, *after* their boxes have been halved. `play_loop.go:521-524`, `atlas/define.md:2049-2056` and plan R11 all state these losses are harmless in sequence; this is the one that is not.

*Fix sketch — one site, which is what makes it the rule rather than the instance:* keep the current size on the loop (initialised from `opt.rows`/`opt.width`, updated in the resize case) and relayout in `show()` before drawing — `if g, ok := q.(play.Grid); ok { g.Resize(cols) }` — **replacing** the resize-case call rather than joining it, so the width keeps one owner. Then declare the vertical bound explicitly: either the board is told an available height and trims its own grid (which preserves `boardFooter`'s entry-index → grid-row identity, since trailing drop already does), or `Rest`/`InputFinish` refuses to sweep a word the last paint did not draw. Pin it with a loop-level test that resizes under a board and asserts that every word the sweep records was on the frame. (ARCH-PURPOSE — the enumeration is the deliverable, not the two rows it was cheap to close; ARCH-CONSTRAINTS — the declared envelope is enforced at selection and at resize-for-the-current-form, and nowhere else.)

## 4. Minor findings

- **Five in-code comments still describe the footer toggle row R11 deleted, and two of them are enumerations of the drawn rows** — `board.go:239` (`Rows()`: *"it counts the toggle and the panel too"* — it counts a blank and the panel), `board.go:453-456` (`Mark`: *"the mode … is drawn in the footer's toggle"*), `board.go:501` (`CellAt`: *"the blank, the toggle, the panel"*), `play_loop.go:573` (*"a footer row below `Rows()` is the toggle or the bar"* — it is the panel or the bar), `play_loop.go:605`. `board.go:239` contradicts `board.go:255-261` twenty lines below it. Plus two docs claims: `atlas/define.md:2065` now asserts *"Every quantity the fit and the click map depend on is read from the terminal as it is at draw and click time, never fixed at selection"*, which the Important above measures false; and `README.md:101` says *"A word you have recalled three times or more is swept on a grid"* when `boardBox` is a **box** (`play_loop.go:665`) — a word recalled five times and then missed twice sits below the threshold. **This is the 4th finding in family `comment-asserts-absent-behaviour`.** Do not patch the sites. The rule holds and is already written down — *prose restating a set the code owns is a second owner* — and what is missing is the trigger: `TestARemovedDeclarationIsSweptOrRetired` skips `toggleLine` because `isCitableName` filters unexported names, and no guard can catch prose that names a *concept* rather than a symbol. So the enumerable half is: when a window deletes a drawn element, `grep -w <its word>` over `currentTruthFiles` is the sweep set, and it is run in the same commit. `TestTheToggleAndPanelRowsAreNotCells` (`board_test.go:644`) is itself named after the removed row.
- The round-3 gate outcome and the R11–R15 work are recorded in the plan, the sidecar, the gate ledger and `workshop/lessons.md`, but the issue's `## Log` still ends at *"boundary review round 2"* (`workshop/issues/000040-form-board.md:591`). Rounds 1 and 2 each got an entry; AGENTS.md §3 puts the boundary outcome there.
- `pty_conformance_test.go:1022` asserts `strings.Contains(keys, "[yes]")` where `keys` came from `play.NewBoard(...).Keys()` in default mode. It can only fail if `Keys()` stops naming the mode, which `board_test.go:150` already pins; the frame-level claim is carried transitively by the `Contains(first, keys)` check above it. Harmless, but it reads as a screen assertion and is not one.

## 5. Test coverage notes

- `go build ./...` clean; `go test ./...` exit 0 (`cmd/define` 107.2s). `go test -race ./cmd/define/ -run 'Board|Resize|FormCell|FitsA|Grid'` green. `TestANarrowingResizeKeepsTheBoardsClickMapHonest` passed 10/10 plain and 5/5 under `-race`.
- Revert-verified this round: `formCell` stubbed to `false` → resize test red in 0.4s; `g.Resize(sz.cols)` removed → red in 2.4s naming the relayout; `closedSection` reverted to `strings.Contains` → the new premise assertion fires on the real `atlas/repo-guards.md`; a closed `### ` block planted above a `## ` heading in the active plan → five guards fail naming the swallowed section.
- `go test -tags conformance ./cmd/define/ -run TestPTYPlayBoardIsDrawnAndClickable` **SKIPS** here (`no pty available: operation not permitted`), as in all three prior rounds. Done-when 14 rests on the implementor's out-of-band unsandboxed run; that is `#37`'s subject, not this issue's.
- **The gap that ships the Important:** every board test either resizes while the board is current, or never resizes. Nothing drives a resize with a board *later in the queue*, and nothing asserts what `Enter` records against what the last frame actually drew. Those are the two tests the fix needs, and either one would have reddened on the measurements above.

## 6. Architectural notes

- **ARCH-DRY — pass.** The mode has exactly one owner (`Keys()`), enforced by a `Prompt()` scan; `closedSection` is extracted rather than inlined twice; `boardFits` still asks a probe rather than re-deriving. The one caution R15 itself raises stands: if the draw-time relayout is added *beside* `play_loop.go:257` rather than replacing it, the width gets two owners, which is the failure this issue keeps paying for.
- **ARCH-PURE — pass.** `board.go` imports nothing (`puretest` empty allowlist), `Resize` is state on the form driven from the thin loop, `TestTheSessionNamesNoForm` reads `session.go` off disk with its own premise check, and the new `Grid.Resize` is on the capability the loop already asks.
- **ARCH-PURPOSE — flag.** Three of round 3's four findings were answered at the class (a filter-wide premise assertion; a screen-driven test driver; prose that points at the type). The fourth stopped at the instance and then *recorded* the class as closed in the plan, the atlas and a code comment — which is the more expensive failure, because the next reader has no reason to re-measure.
- **ARCH-MOCK — pass.** No new external dependency. `seedMature` writes fixtures through `store.NewYAML`, the production writer; the live conformance check exists and cannot run in a review environment.
- **ARCH-CONSTRAINTS — flag, and it is the Important.** Per-frame cost is unchanged and fine (`Prompt()` is O(16 cells) at keystroke rates). The envelope — *"a board that cannot be drawn whole is not a board"* — is enforced at selection, and horizontally at resize time for one form. It is not enforced at draw time and has no declared bounded behaviour when exceeded; the measurements above are the envelope being silently exceeded while three artifacts say it cannot be.
- **For upcoming work:** `formCell`'s contract is now "one footer entry, first physical row, board's own coordinates". A second live-edge form inherits that for free only if the relayout moves to the draw seam — bound to the resize event, each new form has to remember to be current when the terminal changes.

## 7. Plan revision recommendations

- **Amend R9's table, rows 2 and 3.** Row 2 says the layout width is read via `Board.Resize(cols)` "called from the loop's resize case"; say that this covers `s.Current()` only and that a board later in the queue is drawn at its selection-time width — or move the call to `show()` and say *that*. Row 3 says a too-short terminal loses "grid rows, which are neither drawn nor clickable"; add that `Enter` still records `Wrong` for them, and state the chosen bounded behaviour.
- **Amend R11.** *"The losses are harmless in sequence"* is true of the bar and the panel and false of grid rows. Record the measurement (12×24 → cells `[0]`–`[7]` only, eight `Wrong` events for undrawn words; 14×24 → six) and what was done about it.
- **Correct `atlas/define.md:2065`** — the absolute *"never fixed at selection"* is the claim the measurement contradicts; it should say which quantity is read where, or the enumeration should be closed first.
- **Log round 3 in the issue's `## Log`** alongside rounds 1 and 2, before the close.

```findings
dispose:
  - id: BR-8
    disposition: not-addressed
    note: |
      Rows 2 and 3 of its own enumeration are still open and now measured: a board that becomes current after a resize is painted with 73-column rows into a 40-column terminal (g.Resize is called at play_loop.go:257 for s.Current() only), and after a shrinking resize Enter records Wrong for every word fitFooter dropped (12x24 -> 8 of 16 undrawn, all swept). The plan, the atlas and boardFooter's comment now record those losses as harmless.
  - id: BR-12
    disposition: addressed
    note: |
      Verified twice by revert: stubbing formCell to false reddens the test in 0.4s, removing g.Resize(sz.cols) reddens it in 2.4s, and the defer close(keys) means it fails rather than hangs. 10/10 stable, 5/5 under -race.
  - id: BR-13
    disposition: addressed
    note: |
      The plan's Grid row points at the type instead of listing methods; boardFooter's comment and atlas/define.md now say a board CAN end up in a footer that drops rows. Five NEW stale toggle-row comments raised separately as the family's 4th finding.
  - id: BR-14
    disposition: addressed
    note: |
      Verified by revert - the closed-section premise assertion fires on the real atlas/repo-guards.md under the old strings.Contains spelling, and with closedSection's line-start match a planted retired symbol on that page's head section is now caught where it previously was invisible.
findings:
  - id: new
    severity: Minor
    family: comment-asserts-absent-behaviour
    title: |
      Five code comments still describe the deleted footer toggle row, and two docs claims contradict measurement
    detail: |
      4th finding in this family - do not patch the sites. Instances - board.go:239 (Rows() "counts the toggle and the panel too", it counts a blank and the panel, contradicting chromeRows twenty lines below), board.go:453-456 (Mark - "drawn in the footer's toggle"), board.go:501 (CellAt - "the blank, the toggle, the panel"), play_loop.go:573 ("a footer row below Rows() is the toggle or the bar"), play_loop.go:605; plus atlas/define.md:2065 asserting every quantity is read at draw time (false, see BR-8) and README.md:101 saying "recalled three times or more" where boardBox is a BOX, so a word recalled five times and missed twice is below the threshold. The rule is already written down; what is missing is the trigger. TestARemovedDeclarationIsSweptOrRetired skips toggleLine because isCitableName filters unexported names, and no guard can catch prose naming a CONCEPT rather than a symbol - so the enumerable half is that deleting a drawn element makes `grep -w <its word>` over currentTruthFiles the sweep set, run in the same commit. TestTheToggleAndPanelRowsAreNotCells is itself named after the removed row.
  - id: new
    severity: Minor
    family: gate-round-outcome-unlogged
    title: |
      The issue's Log stops at boundary review round 2; round 3's verdict and its R11-R15 work are recorded everywhere except the tracker
    detail: |
      workshop/issues/000040-form-board.md:591 is the last Log section. Rounds 1 and 2 each got an entry naming the findings and the lesson; round 3 (FIX-THEN-SHIP) and the fixes for it appear only in the plan's Revisions, the close-review sidecar, the gate ledger and workshop/lessons.md. AGENTS.md section 3 puts the boundary outcome in the issue's own Log, which is the surface a reader reaches first.
```

---

## Re-review — 2026-09-01T18:13:09-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 40 — form 2.5: the board — grid triage for mature words |
| repo | tools |
| issue file | workshop/issues/000040-form-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | eb9f1698d6800ff5ad22683f49e2c36e970d36f3..0d075085fa4b5a2c8486cc490de418d221f240c6 |
| command | sdlc close --issue 40 |
| reviewer | claude |
| timestamp | 2026-09-01T18:13:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-8 — the one Critical carried across four rounds — is genuinely closed, and I verified it the way the claimed-fixes rule asks rather than by reading the commit message: reverting `g.Resize(termCols)` out of `show()` reddens `TestABoardBuiltBeforeAResizeIsStillLaidOutForTheTerminal` ("the second board's row 0 is 73 columns in a 40-column window"), and deleting the `InputFinish && !boardWhole` guard reddens `TestEnterIsHeldWhileTheBoardIsNotDrawnInFull` (16 events where 1 is wanted). `go test ./...` and `go test ./cmd/define/... -race` are both green. What blocks a clean SHIP is not correctness — it is the fifth round of the same prose family: R17 changed two contracts (Enter is refused; `Board.Resize` moved call sites) and swept the plan's Revisions and Core-concepts rows only, leaving seven live artifacts restating the pre-R17 behaviour, including the resize case's own comment sitting directly above the line that contradicts it. Separately, the README's only picture of the board still renders the design R16 deleted.

## 1. Strengths

- **The fix is at the root, and it is the right root.** `show()` is the one place that draws, so it is the only place that can promise the layout for *every* board — `play_loop.go:196-203`. The old resize-case call could only ever fix the board that happened to be on screen. Confirmed by revert.
- **The fit check errs in the safe direction.** `fitsABoard` charges `barRows = 1` on top of the board's rows, while `fitFooter` drops the bar *first* — so `boardWhole == true` strictly implies every grid row was painted. The false-negative case (bar dropped, board whole) costs a held Enter, never a wrong mark.
- **Two independent defences on the click map, and neither is decorative.** `screen.go:499` records the *fitted* footer, so `FooterRowAt` cannot resolve a row `fitFooter` dropped; `formCell` (`play_loop.go:703`) separately refuses `offset != 0`. Belt and seam, on a path where the mistake is permanent.
- **The Enter hold went in the loop, not in `play`** (`play_loop.go:375-390`) — "what was DRAWN is the terminal's business" is D6's rule applied correctly, and `play`'s purity guard still passes with an empty import allowlist.
- **`closedSection` (`repo_guard_test.go:665`) found a live guard failure, not a hypothetical one:** matching `**closed:**` as a substring meant `atlas/repo-guards.md`'s own prose about the marker discarded the entire guard inventory from every current-truth check. That is the premise-assertion discipline paying for itself.

## 2. Critical findings

None.

## 3. Important findings

**(a) R17's contract change reached the plan's Revisions and nothing else.** Measured sweep set, all still stating the pre-R17 behaviour:

| artifact | what it still says |
|---|---|
| `cmd/define/play_loop.go:289-298` | "AND THE FORM, if it lays itself out (R9)… `sz.cols` rather than `opt.width`" — immediately above `// NO Resize HERE`, with no separator. No `Resize` happens in that case and `sz.cols` no longer reaches the board. |
| `cmd/define/play_loop.go:624-632` | dropped rows are "harmless in sequence" — the exact sentence R17's own commit message identifies as wrong ("harmless to a CLICK… and not to a sweep"). |
| `atlas/define.md:2057` | same "harmless in sequence" claim. |
| `atlas/define.md:2061-2062` | `Board.Resize` "called from the loop's resize case" — false as of `f2186d7`. |
| `atlas/define.md` | no mention of `display.Size()` or of the Enter refusal anywhere. |
| `cmd/define/README.md:133` and `:149` | "`Enter` takes everything still unmarked as 'no'" / "board: finish, taking everything unmarked as 'no'" — unconditional. |
| `workshop/plans/…-plan.md:411`, `workshop/issues/…-form-board.md:82` | both Done-when rows state Enter's contract unconditionally; neither R17 test is cited anywhere in the plan. |

Fix sketch — the rule, not the seven sites: a decision that changes a **drawn or keyed** contract has an enumerable sweep set, and it is the same set every time: the comment at the site the mechanism moved *from*, `atlas/define.md`, the README prose + key table + example block, and both Done-when tables. Write that enumeration into R18 and run it in the commit that makes the change. `TestPlanCitesTestsThatExist` catches a citation that does not resolve; nothing catches shipped behaviour with no citation, which is what happened here — so R18 should also add the two Done-when rows naming `TestEnterIsHeldWhileTheBoardIsNotDrawnInFull` and `TestABoardBuiltBeforeAResizeIsStillLaidOutForTheTerminal`.

**(b) `cmd/define/README.md:107-113` renders the design R16 deleted.** The example block shows `[y] keel`, `[n] quokka`, `[n] sycophantic`, `[y] run` — the mark standing *where the key was* — and a label row `[c] [e] [f] [g]`, the old sequence with a hole at `d`. Both were removed by R16 after the operator's sitting, and the README's own prose contradicts the picture eight lines later (`:124` "`0`–`9` then `a`–`f`, in order and with no gaps", `:128` "keeps its key"). `doc_sync_test.go` pins the prompt *line* only, so the grid is a hand-maintained restatement with no consumer. Fix sketch (ARCH-PURPOSE — make the consumer derive): extend the doc-sync guard to build a `play.Board` over the README block's own words, mark the cells the block shows as marked, and assert the fenced block's cell lines equal `board.Prompt()`'s grid rows.

## 4. Minor findings

- `play_loop.go:203` charges the frame `displayRows(gradePrompt(q), termCols)` but draws `boardPrompt(q, boardWhole)`. Measured: the two are 78 and 79 columns, so at `cols` ∈ {78, 39, 26, …} the drawn prompt is one row taller than the budget charged and one extra footer row is dropped. Bounded to cosmetics (Enter is already held; undrawn rows are unclickable), and the error is always in the safe direction — but it is the family's own rule, measure the string you draw.
- `boardFits` (`play_loop.go:833`) and `show()` each spell the fit formula `fitsABoard(rows, board.Rows(), displayRows(gradePrompt(board), cols))`, and only the selection copy enforces `minWrapWidth`. ARCH-DRY: one helper, asked at both moments.
- `atlas/repo-guards.md`'s guard inventory (`:107-114`) does not list `TestPlanCitesTestsThatExist`, added in this window beside `TestPlanTablesNameEntitiesThatExist`, which is in the table.
- `board_test.go:167` — "the toggle below it legitimately moves when the mode flips" — one more member of BR-15's set, in the test file.

## 5. Test coverage notes

- Both R17 fixes are pinned by tests that fail without them; I ran both reverts in a scratch worktree. `TestABoardBuiltBeforeAResizeIsStillLaidOutForTheTerminal`'s silent-return paths are covered by its own premise assertion, and a lost resize fails it rather than passing it.
- `go test ./...` green; `go test ./cmd/define/... -race` green (115s).
- `go test -tags conformance ./cmd/define/ -run TestPTY` still reports `no pty available: operation not permitted` and **skips every row** in this environment, so Done-when 14 continues to rest on the implementor's out-of-band run. Unchanged from rounds 1–4 and `#37`'s subject, not this issue's.
- Nothing pins the refusal prompt's width or its row count against the budget it is drawn into (finding 3).

## 6. Architectural notes

- **ARCH-DRY** — flag, Minor above: the fit formula has two owners.
- **ARCH-PURE** — pass. `Board`, `Mark`, `Palette` import nothing; `Resize`/`Rows`/`CellAt` are pure arithmetic; the "was it drawn" question is answered in the IO shell and never enters `play`.
- **ARCH-PURPOSE** — flag. T12's docs half is the deferred purpose: the code contracts moved twice (R16, R17) and the artifacts that restate them did not. A README example that no consumer derives is a deferred consumer, not a finished one.
- **ARCH-MOCK** — pass with note. The terminal is behind one seam (`display`), the stateful double (`recordDisplay`) now answers `Size()` with a real default rather than a zero, and live conformance exists behind `-tags conformance`. The conformance rows cannot run in a sandboxed review; that gap is tracked in `#37`.
- **ARCH-CONSTRAINTS** — pass. `Resize` returns immediately when the width is unchanged, so the per-frame cost is one comparison plus one mutex-guarded `Size()`; `displayRows` over a 78-column string is negligible at keystroke rates, and the plan's declared envelope (O(visible rows) per frame) still holds.

## 7. Plan revision recommendations

- **R18 — the sweep R17 owed.** Name the enumeration (site-of-move comment, `atlas/define.md`, README prose + key table + example block, both Done-when tables) and record that it was run. Amend plan Done-when row 3 to state the refusal, and add two rows citing the R17 tests.
- **Issue-file sweep:** `workshop/issues/000040-form-board.md:82` (Done-when row 4) states Enter's contract unconditionally — the issue's own Revisions section already declares that "the issue's own Spec and Done-when are artifacts".
- **`## Log`:** the issue's Log still ends at the operator's sitting. Rounds 3, 4 and R17 are recorded in the plan, the sidecar, the ledger and `lessons.md` — every surface except the tracker a reader reaches first.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      Verified by revert in a scratch worktree — both remaining rows redden their own test.
  - id: BR-15
    disposition: not-addressed
    note: |
      All five code comments unchanged (board.go:276/510/558, play_loop.go:677/709); README:101 unchanged; atlas:2069's claim became true only because the CODE moved, while atlas:2061 became false in the same commit.
  - id: BR-16
    disposition: not-addressed
    note: |
      The issue's Log still ends at the operator's sitting; rounds 3, 4 and R17 are unrecorded there.
findings:
  - id: new
    severity: Important
    family: comment-asserts-absent-behaviour
    title: |
      R17 changed Enter's contract and moved Board.Resize, and swept only the plan's Revisions — seven live artifacts still state the old behaviour
    detail: |
      FIFTH finding in this family — do NOT patch the seven sites. Measured set, all current: play_loop.go:289-298 ("AND THE FORM, if it lays itself out (R9)... `sz.cols` rather than `opt.width`") sits directly above `// NO Resize HERE` with no separator, and neither a Resize nor sz.cols reaches the board there any more; play_loop.go:624-632 and atlas/define.md:2057 still call the dropped rows "harmless in sequence", the exact sentence R17's own commit message identifies as the error; atlas/define.md:2061-2062 names "the loop's resize case" as Board.Resize's call site; atlas and README say nothing about display.Size() or the Enter refusal; README:133 and :149 state Enter's contract unconditionally; plan:411 and issue:82 do the same and neither R17 test is cited anywhere in the plan. THE RULE: a decision that changes a drawn or keyed contract has an enumerable sweep set that is the same every time — the comment at the site the mechanism moved FROM, atlas/define.md, README prose + key table + example block, and both Done-when tables. Write that enumeration into R18 and run it in the same commit. TestPlanCitesTestsThatExist catches an unresolvable citation; nothing catches shipped behaviour with no citation, which is what happened here.
  - id: new
    severity: Important
    family: comment-asserts-absent-behaviour
    title: |
      The README's only picture of the board renders the design R16 deleted, and nothing derives it
    detail: |
      SIXTH in the family, and a distinct trigger from the one above. cmd/define/README.md:107-113 shows `[y] keel`, `[n] quokka`, `[n] sycophantic`, `[y] run` — the mark standing where the key was — and a label row `[c] [e] [f] [g]`, the old sequence with its hole at `d`. R16 deleted both after the operator's sitting, and the README's own prose contradicts the picture eight lines below (`:124` "0-9 then a-f, in order and with no gaps"; `:128` "keeps its key"). doc_sync_test.go pins the prompt LINE only, so the grid block is a hand-maintained restatement with no consumer and no guard. The enforceable fix is to make it derive: build a play.Board over the block's own words, mark the cells the block shows marked, and assert the fenced block's cell lines equal board.Prompt()'s grid rows.
  - id: new
    severity: Minor
    family: frame-budget-hardcoded-not-measured
    title: |
      boardWhole is measured against gradePrompt while the frame draws boardPrompt, and the two differ by a row at some widths
    detail: |
      THIRD in this family — do not special-case the widths. play_loop.go:203 charges the frame `displayRows(gradePrompt(q), termCols)` and then draws `boardPrompt(q, boardWhole)`. Measured: gradePrompt is 78 visible columns and the refusal row is 79, so displayRows disagrees at cols 78 (1 vs 2), 39 (2 vs 3) and 26 (3 vs 4) — the drawn prompt is one row taller than the budget charged and one extra footer row is dropped. Bounded to cosmetics: Enter is already held in that state and an unpainted row is unclickable, and because the refusal is never shorter the error is always in the safe direction. The rule is the family's own — the budget must measure the string that will actually be drawn, so compute the prompt first and measure THAT, rather than measuring a sibling of it.
  - id: new
    severity: Minor
    family: one-measurement-two-owners
    title: |
      The board's fit formula is spelled twice — at selection in boardFits and at draw in show() — and only one copy enforces minWrapWidth
    detail: |
      play_loop.go:833 (boardFits) and play_loop.go:203 (show) both spell `fitsABoard(rows, board.Rows(), displayRows(gradePrompt(board), cols))`. ARCH-DRY: one helper taking (form, rows, cols) asked at both moments. The divergence is already visible — boardFits refuses `opt.width < minWrapWidth` and the draw-time copy does not, so a board narrowed below minWrapWidth by resize can still report whole. Not reachable as harm today (the row arithmetic makes boardWhole false well before the words become unreadable), which is exactly why it should be consolidated before it is.
  - id: new
    severity: Minor
    family: comment-asserts-absent-behaviour
    title: |
      atlas/repo-guards.md's guard inventory does not list TestPlanCitesTestsThatExist, added in this window
    detail: |
      The table at atlas/repo-guards.md:107-114 inventories the repo guards and includes TestPlanTablesNameEntitiesThatExist, the new guard's direct sibling. TestPlanCitesTestsThatExist shipped in 80a4044 as BR-3's class fix and has no row. An inventory that under-states is the same failure as prose that over-states: the next reader cannot tell what is guarded.
```

---

## Re-review — 2026-09-01T20:52:25-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 40 — form 2.5: the board — grid triage for mature words |
| repo | tools |
| issue file | workshop/issues/000040-form-board.md |
| boundary | whole-issue close |
| milestone | — |
| window | eb9f1698d6800ff5ad22683f49e2c36e970d36f3..691868992fe9106973b9bece3b38ac1613311848 |
| command | sdlc close --issue 40 |
| reviewer | claude |
| timestamp | 2026-09-01T20:52:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Every round-5 blocker is genuinely closed and I verified each by revert in a scratch worktree — the Enter hold, the draw-time relayout, the single fit formula, the refusal-row budget, and the README board block that now *derives* from a real `play.Board` all redden their own test when undone. The shipped binary is correct: I found no crash, no silent swallow, and no behaviour drift from the Spec. What blocks a clean SHIP is one class the close commit itself named and then only half-ran. R18 wrote down the seven-surface sweep set for "a change to what is drawn or what a key does" — and ran it over R17 only. The *other* drawn-and-keyed change in the same window, R16 (`d` became a cell label), was never swept: two board.go doc comments still say `d` never reaches the form, the plan's Done-when row 6 lists the shipped design as its failure condition, the issue's row 6 cites a test that was deleted three commits later, and `TestDOnABoardIsNotACellLabel` asserts the *deleted* contract thirty lines below a test that proves the opposite — passing only because its fixture has four cells and `d` is index 13. BR-15's five stale toggle comments are also untouched for a second round running.

## 1. Strengths

- **`boardFitsIn` is a real consolidation, not a rename.** `cmd/define/play_loop.go:594` is now the one fit formula and `boardFits` delegates to it (`play_loop.go:865`). Removing the `minWrapWidth` clause reddens `TestSelectionAndDrawAskTheSameFitQuestion` at 19 columns — verified.
- **The Enter hold is put in the right place and pinned.** `play_loop.go:369-386` refuses in the *loop*, not the session, so `play` stays terminal-ignorant (D6 applied to a destructive key). Deleting the four-line guard reddens `TestEnterIsHeldWhileTheBoardIsNotDrawnInFull` — verified.
- **The layout moved to the one place that draws.** `play_loop.go:196` reads `view.Size()` per frame instead of the loop keeping a copy. Stubbing out `g.Resize(termCols)` reddens `TestABoardBuiltBeforeAResizeIsStillLaidOutForTheTerminal` with the exact symptom ("row 0 is 73 columns in a 40-column window") — verified.
- **The README's board picture now derives.** `cmd/define/doc_sync_test.go:364` builds a real `play.Board` and asserts the fenced rows. Planting `[y] sycophantic` back into the block reddens it twice over — verified. This is the right shape for BR-18: a picture with a consumer.
- **ARCH-MOCK is honoured on the widened seam.** `recordDisplay` (`editorloop_test.go:159-193`) is a *stateful* fake for `FooterRowAt`/`Size` with a non-degenerate default, `newPinnedScreen` runs the real screen against a buffer, and `pty_conformance_test.go` is the live conformance check on a real tty.

## 2. Critical findings

None.

## 3. Important findings

**(a) R16 made `d` a cell label; the tests and both Done-when tables still assert the design it deleted.**
**This is the 7th finding in family `comment-asserts-absent-behaviour`.** Do **not** patch the sites — the rule is what is missing. Measured, all current:

- `cmd/define/play/board.go:493` — `Grade`'s doc: *"`d` and `D` never arrive: toInput takes them first, and boardLabels has no cell for them either way."* Both clauses are false. `Apply`'s `InputDrop` case (`session.go:262`) hands the rune to `q.Grade` for any `Batch`, and `boardLabels` is `"0123456789abcdef"`. Executed: `Apply(board16, {InputDrop, 'd'})` returns `{OutcomeRecord, Word:"a14", Form:"board"}` and marks cell 13.
- `cmd/define/play/board.go:452-454` — `Keys`'s doc: *"It has a hole at `d` … spelling `0-9 a-c e-g` here would be a second owner."* The hole was removed by R16; `boardLabels`'s own doc 380 lines above says "Sixteen labels, no gap".
- `cmd/define/play_loop_test.go:2996-2999` — the doc comment says *"`d` IS NOT A CELL LABEL … on a board Apply refuses it too"*, directly contradicting `TestABoardIsMarkableByKeyAlone` (`:2961`), which sends `BoardLabels[13]` and asserts 16 events.
- `cmd/define/play_loop_test.go:2999` — `TestDOnABoardIsNotACellLabel` asserts `0 review events after pressing d twice`. It passes only because its fixture is four words, so index 13 is out of range. With sixteen it fails. A test that certifies the opposite of the shipped contract, kept green by a fixture accident, is the worst kind of pin: a future reader "fixing the code to match" would undo the operator's own correction.
- `workshop/plans/000040-form-board-plan.md:414` — row 6's `red when` is *"labels include `d`"*, which is now the shipped design.
- `workshop/issues/000040-form-board.md:508` — row 6's claim is *"`d` is not a label"* and its mutation is *"`boardLabels` becomes `…abcdef`"*, i.e. the current state.

**THE RULE.** R18's checklist (`plan.md` R18) is right and was run forward over R17 only. Two things it needs before it works: (1) it must be run over **every** drawn-or-keyed change already in the window, not the one that produced the finding — R16 and R17 are both in this window and only one was swept (ARCH-PURPOSE: the class, not the instance); and (2) it needs an **8th row — the TEST that pins the old contract**, because a test's *name* and its *fixture* are restatements of the contract exactly as prose is, and this one survived while its own subject was inverted. The mechanical half is cheap: when a contract changes, the sweep set includes `grep -rn <the old claim's word>` over `*_test.go`, and any test whose name asserts the negated contract is a rename-or-delete, not a pass.

**(b) The citation guards cannot see the tracker, and cannot see a symbol born and buried inside one window.**
**This is the 4th finding in family `plan-citations-unenforced`.** Do not patch the instance. Two independent holes, both measured:

- `cmd/define/repo_guard_test.go:1183` globs `workshop/plans/*-plan.md` only, and `currentTruthFiles` (`:1508-1520`) excludes `workshop/issues/`. Measured over the tree: active issues carry **45** backticked `Test*` citations in their Done-when and mutation tables, of which **2 do not resolve** — `TestBoardLabelsSkipTheReservedD` (issue `#40:508`) and `TestAMissIsRecordedBeforeItIsRevealed` (issue `#33`). The issue file is the surface a reader reaches *first*; a `pinned by` cell there makes the same claim a plan's does.
- `TestARemovedDeclarationIsSweptOrRetired` (`:1436`) diffs `base..HEAD`, so a declaration **added and removed inside the same window is invisible**. `TestBoardLabelsSkipTheReservedD` was added in `816a0cb` and deleted in `4949778`, both inside this window: `git diff eb9f169..6918689 -- ':/*.go' | grep -c TestBoardLabelsSkipTheReservedD` returns 0.

**THE RULE.** A guard over citations must span (i) every artifact class that carries a citation — plans *and* issues, since both use the same `| claim | pinned by |` table — and (ii) the whole window commit-by-commit, not just its endpoints, because a window is not a single diff. Both are one-line scope changes; the finding is that the scope was set from the artifact that produced BR-3 rather than from the set of artifacts that make the claim.

## 4. Minor findings

- `cmd/define/play/board.go:276` — `Rows()`'s doc still says it "counts the toggle and the panel", contradicting `chromeRows` twenty lines below (BR-15, unfixed).
- `cmd/define/play/board.go:467`, `:510`, `:558` and `cmd/define/play_loop.go:708`, `:740` — four more live references to the footer toggle row deleted by R11 (BR-15, unfixed).
- `cmd/define/README.md:100` — "recalled three times or more" describes `boardBox` as a recall count; it is a **box**, so a word recalled five times and missed twice is below the threshold (BR-15, unfixed).
- `workshop/plans/000040-form-board-plan.md:333-350` — the Core-concepts table omits `Palette` (a **new exported type** on `play`'s public API), `boardFitsIn` and `boardRefusal`. Don't hand-add the rows: `TestPlanTablesNameEntitiesThatExist` only checks table→tree, and **tools#33** is the open issue for the missing direction. Same family as BR-21 (an inventory that under-states).
- `workshop/issues/000040-form-board.md:202-218` — the close summary was inserted *under* the `### 2026-09-01 — T2` heading, above T2's own text, so the Log's chronology now starts with its ending. Content is fine; placement makes the first `###` a reader reaches read as T2's record.
- `cmd/define/play/purity_test.go:52` — the comment stripper cuts at the first `//` on a line, so a string literal containing `//` would truncate the code it is grepping. Not reachable in `session.go` today.

## 5. Test coverage notes

- Green: `go test ./cmd/define/...` passes at HEAD (107s).
- Every round-5 claim was verified by revert, not read: refusal string (79 vs 78 cols → 3 failures), `minWrapWidth` in `boardFitsIn`, the `[y]` cell in the README block, the Enter-hold branch, and `g.Resize(termCols)` in `show`. All five redden. No false "fixed in <sha>".
- The gap the diff could ship and does not cover: **a full sixteen-cell board driven by `d`** through `TestDOnABoardIsNotACellLabel`'s own path. `TestABoardIsMarkableByKeyAlone` covers it through the label loop, which is why the contradiction is survivable — but the two tests disagree and only one is right.
- `TestSelectionAndDrawAskTheSameFitQuestion`'s first half is now structurally equal (`boardFits` delegates to `boardFitsIn`); the assertion that carries the claim is the `minWrapWidth-1` case at `:3448`, which is real. Worth knowing when reading it.

## 6. Architectural notes

- **ARCH-DRY — pass.** BR-20's consolidation landed and is pinned. `gradePrompt` is built twice per frame (`boardFitsIn` and `boardPrompt`); trivial, not worth a change. `boardPalette` spells `\x1b[1;32m` literally, which is `highlight.go:15`'s `knownOn` — different facts, and consistent with `editor.go`'s existing two `sgrOff` spellings, so no finding.
- **ARCH-PURE — pass.** `play` keeps its no-imports/no-clock guard; `Board` is unit-tested with no IO (`board_test.go`, 836 lines). The Enter refusal was deliberately kept in the loop so the pure package never learns what "drawn" means — the right side of the seam.
- **ARCH-PURPOSE — flag.** See Important (a). The enumeration was written and applied to the finding's own site; the sibling in the same window was left. This is the *fourth* round in which a stated rule outlived its own application, which is the signal that the rule needs a mechanical trigger, not another restatement.
- **ARCH-MOCK — pass.** Production and test flow share the `display` boundary; the fake is stateful; a live pty conformance row exists and was run unsandboxed.
- **ARCH-CONSTRAINTS — pass.** `show` is on the keystroke path and does one mutex-guarded `Size()` plus an idempotent `Resize`; `boardFitsIn` is O(cells) with cells ≤ 16; `todaysQuestions` still pays exactly one `Deck()` and one `Events()` per sitting, documented at `play_loop.go:875`. No unbounded fan-out or repeated expensive work introduced.

## 7. Plan revision recommendations

- **R20 — the sweep set applies to the window, not to the finding.** Record that R18's checklist was run over R17 and not over R16, name the six R16 sites above, and add the 8th row: *the test that pins the old contract — its name and its fixture*. State the mechanical trigger (`grep` the old claim's word over `*_test.go` in the same commit) so the next contract change does not need the checklist to be remembered.
- **R21 — the citation guards' scope.** Record both measured holes (issues excluded from the glob and from `currentTruthFiles`; `base..HEAD` blind to a within-window add-then-delete), with the measured prevalence (45 issue citations, 2 unresolvable), and the rule: scope a guard to the set of artifacts that make the claim, over the whole window.
- **Correct Done-when row 6** in the plan (`:414`) and the issue (`:508`) as part of R20 — the `red when` currently names the shipped design, and the issue's `caught by` cell cites a deleted test.
