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
    - "n": 2
      timestamp: "2026-09-01T13:14:12-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by reverting the InputReveal batch guard: TestEveryInputKindIsAnsweredForABatchForm goes red on kind 1 and on Revealed.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Verified by restoring the constant: TestFitsABoardCountsTheWholeLiveEdge and TestAShortTerminalGetsMeaningChoiceNotAClippedBoard/very-narrow both go red.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: All 25 cited tests resolve; planting a fake citation reddens TestPlanCitesTestsThatExist. The guard's own vacuity is raised separately.
          round: 2
        - id: BR-4
          disposition: addressed
          note: board.go:255 now says NO COLOUR YET and marks it deferred; comment-only, nothing to pin.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The fixture equality is now a t.Fatalf and the claim is board.Spent(), which can fail.
          round: 2
        - id: BR-6
          disposition: addressed
          note: 'Verified by restoring the concatenation: TestRowsIsWhatPromptDraws goes red at n=0 for all three widths.'
          round: 2
        - id: BR-7
          disposition: addressed
          note: CellAt's doc states the padding rule and TestACellsPaddingBelongsToIt reddens under both a wider and a narrower hit box.
          round: 2
      findings:
        - id: BR-8
          severity: Critical
          title: A narrowing resize under a live board marks the wrong word on a click and drops the toggle, panel and bar
          detail: 'SECOND finding in family frame-budget-hardcoded-not-measured — do not fix this instance alone. board.go:136 claims "the width it is derived from cannot change"; play_loop.go:245 changes it via view.Resize and redraws the same board. Measured: a board built at 80 has 74-column grid rows, so displayRows(row, 40) = 2; FooterRowAt returns the same entry for both physical rows (screen_test.go:1220 pins that), and formCell (play_loop.go:562) passes the raw physical column into CellAt — CellAt(0,4)=cell 0 while column 44 of that line is "[2] sycophantic", so a click on the continuation row lands a permanent mark on the wrong word. Painting the same board at 24x12 drops the blank, the TOGGLE, the panel and the bar, which is BR-2''s harm through a door fitsABoard cannot see. The rule: every quantity the board''s fit and click map depend on must be read from the terminal as it is at draw/click time, never fixed at selection time. Enumeration - prompt height (fixed), board layout width (open), the fit re-check after resize (open), the column translation for a wrapped footer entry (open). No test in the diff draws a board at a width other than the one it was built for.'
          family: frame-budget-hardcoded-not-measured
          round: 2
        - id: BR-9
          severity: Important
          title: TestPlanCitesTestsThatExist skips silently when currentTruthOnly truncates the plan, which is the shape that produced BR-3
          detail: 'SECOND finding in family plan-citations-unenforced — do not fix this instance. The fix round diagnosed the real mechanism (currentTruthOnly cuts at the FIRST "## Revisions", this plan had two, both plan guards read a truncated file and passed) and then fixed only this plan''s layout. Verified by execution: reinserting a "## Revisions" heading above "## Core concepts" and planting a fabricated Test name in the Done-when table leaves TestPlanCitesTestsThatExist PASS via its checked==0 t.Skip at repo_guard_test.go:1178. Measured prevalence - currentTruthOnly has 8 call sites in repo_guard_test.go; four of the guards reading it end in checked==0 t.Skip (729, 881, 1106, 1178); none checks that the surviving text still contains the section it exists to check, while seven other guards in the same file do Fatal on vacuity. The rule: a guard reading a filtered view of an artifact must assert its premise about that view and FAIL, never Skip, when the filter removed the thing it checks.'
          family: plan-citations-unenforced
          round: 2
        - id: BR-10
          severity: Minor
          title: The Batch doc still says the capability is consulted at FOUR points and lists three, omitting the InputReveal path added in the same commit
          detail: 'SECOND finding in family comment-asserts-absent-behaviour — state the rule rather than patching the site. cmd/define/play/session.go:474 says "It is consulted at FOUR points… the other three were found by measurement" and enumerates the miss branch, InputDrop and InputFinish; InputReveal is now a fifth. The plan repeats it at :378 ("the FOUR places Apply consults it") and in the Core-concepts rows at :326 and :327. This is the same prose enumeration whose incompleteness caused BR-1, left stating the wrong number by the commit that declared prose the culprit. The rule: prose restating a set the code owns is a second owner and drifts — point at numInputKinds and the table test instead of re-counting. Records (issue Log, close-review sidecar) legitimately keep "four".'
          family: comment-asserts-absent-behaviour
          round: 2
        - id: BR-11
          severity: Minor
          title: The InputKind table's expectation struct declares a `reveals` field that no case sets and nothing reads
          detail: cmd/define/play/session_test.go:863 — `reveals bool` sits beside `kinds` and `advances` and reads as a third checked dimension. Revealed is in fact asserted unconditionally for every kind, so the field is dead; either drop it or make the reveal expectation per-case.
          family: test-declares-unchecked-expectation
          round: 2
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

## Round 2 — 2026-09-01T13:14:12-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by reverting the InputReveal batch guard: TestEveryInputKindIsAnsweredForABatchForm goes red on kind 1 and on Revealed.
- BR-2 — addressed — Verified by restoring the constant: TestFitsABoardCountsTheWholeLiveEdge and TestAShortTerminalGetsMeaningChoiceNotAClippedBoard/very-narrow both go red.
- BR-3 — addressed — All 25 cited tests resolve; planting a fake citation reddens TestPlanCitesTestsThatExist. The guard's own vacuity is raised separately.
- BR-4 — addressed — board.go:255 now says NO COLOUR YET and marks it deferred; comment-only, nothing to pin.
- BR-5 — addressed — The fixture equality is now a t.Fatalf and the claim is board.Spent(), which can fail.
- BR-6 — addressed — Verified by restoring the concatenation: TestRowsIsWhatPromptDraws goes red at n=0 for all three widths.
- BR-7 — addressed — CellAt's doc states the padding rule and TestACellsPaddingBelongsToIt reddens under both a wider and a narrower hit box.

### Raised

- **BR-8** [Critical] `frame-budget-hardcoded-not-measured` A narrowing resize under a live board marks the wrong word on a click and drops the toggle, panel and bar
  SECOND finding in family frame-budget-hardcoded-not-measured — do not fix this instance alone. board.go:136 claims "the width it is derived from cannot change"; play_loop.go:245 changes it via view.Resize and redraws the same board. Measured: a board built at 80 has 74-column grid rows, so displayRows(row, 40) = 2; FooterRowAt returns the same entry for both physical rows (screen_test.go:1220 pins that), and formCell (play_loop.go:562) passes the raw physical column into CellAt — CellAt(0,4)=cell 0 while column 44 of that line is "[2] sycophantic", so a click on the continuation row lands a permanent mark on the wrong word. Painting the same board at 24x12 drops the blank, the TOGGLE, the panel and the bar, which is BR-2's harm through a door fitsABoard cannot see. The rule: every quantity the board's fit and click map depend on must be read from the terminal as it is at draw/click time, never fixed at selection time. Enumeration - prompt height (fixed), board layout width (open), the fit re-check after resize (open), the column translation for a wrapped footer entry (open). No test in the diff draws a board at a width other than the one it was built for.
- **BR-9** [Important] `plan-citations-unenforced` TestPlanCitesTestsThatExist skips silently when currentTruthOnly truncates the plan, which is the shape that produced BR-3
  SECOND finding in family plan-citations-unenforced — do not fix this instance. The fix round diagnosed the real mechanism (currentTruthOnly cuts at the FIRST "## Revisions", this plan had two, both plan guards read a truncated file and passed) and then fixed only this plan's layout. Verified by execution: reinserting a "## Revisions" heading above "## Core concepts" and planting a fabricated Test name in the Done-when table leaves TestPlanCitesTestsThatExist PASS via its checked==0 t.Skip at repo_guard_test.go:1178. Measured prevalence - currentTruthOnly has 8 call sites in repo_guard_test.go; four of the guards reading it end in checked==0 t.Skip (729, 881, 1106, 1178); none checks that the surviving text still contains the section it exists to check, while seven other guards in the same file do Fatal on vacuity. The rule: a guard reading a filtered view of an artifact must assert its premise about that view and FAIL, never Skip, when the filter removed the thing it checks.
- **BR-10** [Minor] `comment-asserts-absent-behaviour` The Batch doc still says the capability is consulted at FOUR points and lists three, omitting the InputReveal path added in the same commit
  SECOND finding in family comment-asserts-absent-behaviour — state the rule rather than patching the site. cmd/define/play/session.go:474 says "It is consulted at FOUR points… the other three were found by measurement" and enumerates the miss branch, InputDrop and InputFinish; InputReveal is now a fifth. The plan repeats it at :378 ("the FOUR places Apply consults it") and in the Core-concepts rows at :326 and :327. This is the same prose enumeration whose incompleteness caused BR-1, left stating the wrong number by the commit that declared prose the culprit. The rule: prose restating a set the code owns is a second owner and drifts — point at numInputKinds and the table test instead of re-counting. Records (issue Log, close-review sidecar) legitimately keep "four".
- **BR-11** [Minor] `test-declares-unchecked-expectation` The InputKind table's expectation struct declares a `reveals` field that no case sets and nothing reads
  cmd/define/play/session_test.go:863 — `reveals bool` sits beside `kinds` and `advances` and reads as a third checked dimension. Revealed is in fact asserted unconditionally for every kind, so the field is dead; either drop it or make the reveal expectation per-case.

## Open findings

- **BR-8** [Critical] `frame-budget-hardcoded-not-measured` A narrowing resize under a live board marks the wrong word on a click and drops the toggle, panel and bar
- **BR-9** [Important] `plan-citations-unenforced` TestPlanCitesTestsThatExist skips silently when currentTruthOnly truncates the plan, which is the shape that produced BR-3
- **BR-10** [Minor] `comment-asserts-absent-behaviour` The Batch doc still says the capability is consulted at FOUR points and lists three, omitting the InputReveal path added in the same commit
- **BR-11** [Minor] `test-declares-unchecked-expectation` The InputKind table's expectation struct declares a `reveals` field that no case sets and nothing reads
