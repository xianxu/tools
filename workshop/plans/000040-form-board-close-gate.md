---
gate: boundary-review
issue: 40
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-01T12:47:23-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Space on a board emits OutcomeReveal, so the loop plays an arbitrary cell's audio and writes a blank reveal into the buffer
          detail: 'cmd/define/play/session.go:319 — the InputReveal case has no Batch/Grid guard, so it sets s.Revealed and returns Outcome{Kind: OutcomeReveal, Word: q.Word()} (cells[0] before any mark). Reproduced by execution. play_loop.go then writes "\n"+Reveal()+"\n" into the append-only buffer and calls playAnnounced on that word. D14 and the board''s own doc both state InputReveal must do nothing for a form holding many words. D12 enumerated four Apply paths that must consult Batch; this is the fifth. Fix the class: a table test driving every InputKind at a batch form.'
          family: batch-capability-not-asked-on-every-path
          round: 1
        - id: BR-2
          severity: Important
          title: fitsABoard charges the 76-column keys prompt one row, so a board narrower than 76 columns is drawn with its toggle, panel or bar dropped
          detail: 'cmd/define/play_loop.go:506 — boardChromeRows = 2 assumes a one-row prompt, but boardFits only requires width >= minWrapWidth (20). Measured against the real functions: w=40 r=11 drops the bar; w=36 r=12 drops the bar; w=24 r=13 drops the TOGGLE and panel — the single owner of which mark is live, while every mark is irreversible. That is D15''s own condition failing. Charge displayRows(gradePrompt(probe), width) instead of a constant, and add a width axis to TestFitsABoardCountsTheWholeLiveEdge, which only exercises 80.'
          family: frame-budget-hardcoded-not-measured
          round: 1
        - id: BR-3
          severity: Important
          title: The plan's Done-when table pins three rows on four tests that do not exist, and cites an (R3) revision that was never written
          detail: workshop/plans/000040-form-board-plan.md:459,461,465 name TestEnterCommitsTheUnmarkedAsNo, TestSpaceDoesNotCommitABoard, TestANoMarkDoesNotEndTheBoard and TestAReviewEventNamesItsForm; none exist. Line 362 cites (R3) and the Revisions section runs R1, R2, R4, R5. TestPlanTableStatusMatchesTheChangeWindow guards only the status column, so the pinned-by column drifted — the exact class the issue's own Log recorded one round earlier. Fix the cells AND extend the guard so every backticked Test* in an active plan must resolve to a func in the tree.
          family: plan-citations-unenforced
          round: 1
        - id: BR-4
          severity: Minor
          title: board.go claims "the loop adds colour" to the marks; no styling exists anywhere on the board path
          detail: cmd/define/play/board.go:257. boardFooter (play_loop.go:495) splits Prompt() and appends sittingBar with no colour. D10's decision table chose option D partly because it "keeps colour"; nothing ships it. Either implement it or mark it deferred in the comment.
          family: comment-asserts-absent-behaviour
          round: 1
        - id: BR-5
          severity: Minor
          title: The keystroke half of Done-when 13's pin is a tautology over the test's own fixture
          detail: cmd/define/play_loop_test.go:2692 — `if len(boardKeys) > len(words)` cannot fail, because boardKeys is built by `for i := range words`. The reading-cost ratio in the same test is real and does carry the claim; the keystroke floor its red-when names is unpinned.
          family: assertion-cannot-fail
          round: 1
        - id: BR-6
          severity: Minor
          title: Rows() returns 3 for an empty board while Prompt() yields four lines
          detail: cmd/define/play/board.go:198 — gridRows() is 0, so Rows() is chromeRows (3), but Prompt() emits "" + "\n\n" + toggle + "\n" + panel, which splits to four entries. Unreachable today (todaysQuestions:773 guards len(cells) > 0), but Word() and panelLine() both defend the empty case, so the invariant is held inconsistently and TestNoGridLineExceedsTheWidth's Rows() assertion never sees it.
          family: degenerate-input-breaks-derived-count
          round: 1
        - id: BR-7
          severity: Minor
          title: A click in a cell's trailing padding marks it, including past the visible end of a trimmed last cell
          detail: cmd/define/play/board.go:453 — CellAt accepts any column within labelWidth+wordCells, while Prompt trims the row's trailing blanks. It marks the column's own word rather than a neighbour, so the stated rule holds, but clicking apparently blank space and getting a permanent mark deserves a line in CellAt's doc if intended.
          family: click-target-wider-than-drawn
          round: 1
      blocked: true
---

# Gate ledger — tools#40 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T12:47:23-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `batch-capability-not-asked-on-every-path` Space on a board emits OutcomeReveal, so the loop plays an arbitrary cell's audio and writes a blank reveal into the buffer
  cmd/define/play/session.go:319 — the InputReveal case has no Batch/Grid guard, so it sets s.Revealed and returns Outcome{Kind: OutcomeReveal, Word: q.Word()} (cells[0] before any mark). Reproduced by execution. play_loop.go then writes "\n"+Reveal()+"\n" into the append-only buffer and calls playAnnounced on that word. D14 and the board's own doc both state InputReveal must do nothing for a form holding many words. D12 enumerated four Apply paths that must consult Batch; this is the fifth. Fix the class: a table test driving every InputKind at a batch form.
- **BR-2** [Important] `frame-budget-hardcoded-not-measured` fitsABoard charges the 76-column keys prompt one row, so a board narrower than 76 columns is drawn with its toggle, panel or bar dropped
  cmd/define/play_loop.go:506 — boardChromeRows = 2 assumes a one-row prompt, but boardFits only requires width >= minWrapWidth (20). Measured against the real functions: w=40 r=11 drops the bar; w=36 r=12 drops the bar; w=24 r=13 drops the TOGGLE and panel — the single owner of which mark is live, while every mark is irreversible. That is D15's own condition failing. Charge displayRows(gradePrompt(probe), width) instead of a constant, and add a width axis to TestFitsABoardCountsTheWholeLiveEdge, which only exercises 80.
- **BR-3** [Important] `plan-citations-unenforced` The plan's Done-when table pins three rows on four tests that do not exist, and cites an (R3) revision that was never written
  workshop/plans/000040-form-board-plan.md:459,461,465 name TestEnterCommitsTheUnmarkedAsNo, TestSpaceDoesNotCommitABoard, TestANoMarkDoesNotEndTheBoard and TestAReviewEventNamesItsForm; none exist. Line 362 cites (R3) and the Revisions section runs R1, R2, R4, R5. TestPlanTableStatusMatchesTheChangeWindow guards only the status column, so the pinned-by column drifted — the exact class the issue's own Log recorded one round earlier. Fix the cells AND extend the guard so every backticked Test* in an active plan must resolve to a func in the tree.
- **BR-4** [Minor] `comment-asserts-absent-behaviour` board.go claims "the loop adds colour" to the marks; no styling exists anywhere on the board path
  cmd/define/play/board.go:257. boardFooter (play_loop.go:495) splits Prompt() and appends sittingBar with no colour. D10's decision table chose option D partly because it "keeps colour"; nothing ships it. Either implement it or mark it deferred in the comment.
- **BR-5** [Minor] `assertion-cannot-fail` The keystroke half of Done-when 13's pin is a tautology over the test's own fixture
  cmd/define/play_loop_test.go:2692 — `if len(boardKeys) > len(words)` cannot fail, because boardKeys is built by `for i := range words`. The reading-cost ratio in the same test is real and does carry the claim; the keystroke floor its red-when names is unpinned.
- **BR-6** [Minor] `degenerate-input-breaks-derived-count` Rows() returns 3 for an empty board while Prompt() yields four lines
  cmd/define/play/board.go:198 — gridRows() is 0, so Rows() is chromeRows (3), but Prompt() emits "" + "\n\n" + toggle + "\n" + panel, which splits to four entries. Unreachable today (todaysQuestions:773 guards len(cells) > 0), but Word() and panelLine() both defend the empty case, so the invariant is held inconsistently and TestNoGridLineExceedsTheWidth's Rows() assertion never sees it.
- **BR-7** [Minor] `click-target-wider-than-drawn` A click in a cell's trailing padding marks it, including past the visible end of a trimmed last cell
  cmd/define/play/board.go:453 — CellAt accepts any column within labelWidth+wordCells, while Prompt trims the row's trailing blanks. It marks the column's own word rather than a neighbour, so the stated rule holds, but clicking apparently blank space and getting a permanent mark deserves a line in CellAt's doc if intended.

## Open findings

- **BR-1** [Critical] `batch-capability-not-asked-on-every-path` Space on a board emits OutcomeReveal, so the loop plays an arbitrary cell's audio and writes a blank reveal into the buffer
- **BR-2** [Important] `frame-budget-hardcoded-not-measured` fitsABoard charges the 76-column keys prompt one row, so a board narrower than 76 columns is drawn with its toggle, panel or bar dropped
- **BR-3** [Important] `plan-citations-unenforced` The plan's Done-when table pins three rows on four tests that do not exist, and cites an (R3) revision that was never written
- **BR-4** [Minor] `comment-asserts-absent-behaviour` board.go claims "the loop adds colour" to the marks; no styling exists anywhere on the board path
- **BR-5** [Minor] `assertion-cannot-fail` The keystroke half of Done-when 13's pin is a tautology over the test's own fixture
- **BR-6** [Minor] `degenerate-input-breaks-derived-count` Rows() returns 3 for an empty board while Prompt() yields four lines
- **BR-7** [Minor] `click-target-wider-than-drawn` A click in a cell's trailing padding marks it, including past the visible end of a trimmed last cell
