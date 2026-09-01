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
