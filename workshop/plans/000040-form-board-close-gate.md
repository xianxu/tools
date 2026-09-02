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
    - "n": 3
      timestamp: "2026-09-01T13:42:55-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: wrong-word click closed and pinned; the layout-width and fit-after-resize rows of its own enumeration are still open — measured
          round: 3
        - id: BR-9
          disposition: addressed
          note: 'verified by planting BR-3''s shape in a scratch plan: three guards now FAIL naming the swallowed section'
          round: 3
        - id: BR-10
          disposition: addressed
          note: the Batch doc points at numInputKinds and the table test; remaining "four" mentions are records or the true batchOf call-site count
          round: 3
        - id: BR-11
          disposition: addressed
          note: the reveals field is gone and Revealed is asserted unconditionally for every kind
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: TestANarrowingResizeKeepsTheBoardsClickMapHonest asserts over an empty event set, so R9's click-map claim is unpinned at the loop
          detail: |-
            2nd finding in family assertion-cannot-fail. The rule, not the instance: a test whose subject is an
            EVENT must assert the event happened — `for _, e := range events` with no count check certifies nothing.
            Measured: instrumented and run three times, the test records 0 review events every time, and it still
            passes when formCell is stubbed to return false unconditionally. The click never lands because the
            goroutine derives its row from strings.Split(frame, "\r\n") — logical writes — while FooterRowAt works in
            physical rows, and the 76-column keys prompt wraps to two at width 40. Same lesson the issue's Log
            records for T13 ("the click's ROW and COLUMN are read off the paint rather than computed"), unapplied
            one test over. The relayout and row-width halves of the test are real and do fail without the fix.
          family: assertion-cannot-fail
          round: 3
        - id: BR-13
          severity: Minor
          title: 'Two live prose enumerations restate sets the code owns: the plan''s Grid row omits Resize, and boardFooter plus atlas claim a board is never in a footer that drops rows'
          detail: |-
            3rd finding in family. The rule: prose restating a set the code owns is a second owner and drifts —
            point at the type. plan.md:327 lists Rows, CellAt, Mark for an interface that has had four methods since
            R9 added Resize. play_loop.go:509-511 and atlas/define.md:2068 assert "a board is never IN a footer that
            has to drop anything", which measurement contradicts after a resize (toggle dropped at 10x40 and 12x24).
            Neither is guarded: TestPlanTableStatusMatchesTheChangeWindow reads the name|file|status cells only.
          family: comment-asserts-absent-behaviour
          round: 3
        - id: BR-14
          severity: Minor
          title: currentTruthOnly has two discarding rules and R10 gave a premise assertion to only one
          detail: |-
            3rd finding in family. The closed-section rule (repo_guard_test.go:640-646) still discards silently, and
            because it splits on "\n### " a closed section swallows everything up to the NEXT "### " — which can
            include a following top-level "## " section. No live instance in the tree today (define-learn.md's closed
            sections already sit below the ## Log truncation), so this is the rule R10 wrote applied to every rule the
            filter has, rather than a present defect.
          family: plan-citations-unenforced
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-01T14:45:56-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: 'Rows 2 and 3 of its own enumeration are still open and now measured: a board that becomes current after a resize is painted with 73-column rows into a 40-column terminal (g.Resize is called at play_loop.go:257 for s.Current() only), and after a shrinking resize Enter records Wrong for every word fitFooter dropped (12x24 -> 8 of 16 undrawn, all swept). The plan, the atlas and boardFooter''s comment now record those losses as harmless.'
          round: 4
        - id: BR-12
          disposition: addressed
          note: 'Verified twice by revert: stubbing formCell to false reddens the test in 0.4s, removing g.Resize(sz.cols) reddens it in 2.4s, and the defer close(keys) means it fails rather than hangs. 10/10 stable, 5/5 under -race.'
          round: 4
        - id: BR-13
          disposition: addressed
          note: The plan's Grid row points at the type instead of listing methods; boardFooter's comment and atlas/define.md now say a board CAN end up in a footer that drops rows. Five NEW stale toggle-row comments raised separately as the family's 4th finding.
          round: 4
        - id: BR-14
          disposition: addressed
          note: Verified by revert - the closed-section premise assertion fires on the real atlas/repo-guards.md under the old strings.Contains spelling, and with closedSection's line-start match a planted retired symbol on that page's head section is now caught where it previously was invisible.
          round: 4
      findings:
        - id: BR-15
          severity: Minor
          title: Five code comments still describe the deleted footer toggle row, and two docs claims contradict measurement
          detail: 4th finding in this family - do not patch the sites. Instances - board.go:239 (Rows() "counts the toggle and the panel too", it counts a blank and the panel, contradicting chromeRows twenty lines below), board.go:453-456 (Mark - "drawn in the footer's toggle"), board.go:501 (CellAt - "the blank, the toggle, the panel"), play_loop.go:573 ("a footer row below Rows() is the toggle or the bar"), play_loop.go:605; plus atlas/define.md:2065 asserting every quantity is read at draw time (false, see BR-8) and README.md:101 saying "recalled three times or more" where boardBox is a BOX, so a word recalled five times and missed twice is below the threshold. The rule is already written down; what is missing is the trigger. TestARemovedDeclarationIsSweptOrRetired skips toggleLine because isCitableName filters unexported names, and no guard can catch prose naming a CONCEPT rather than a symbol - so the enumerable half is that deleting a drawn element makes `grep -w <its word>` over currentTruthFiles the sweep set, run in the same commit. TestTheToggleAndPanelRowsAreNotCells is itself named after the removed row.
          family: comment-asserts-absent-behaviour
          round: 4
        - id: BR-16
          severity: Minor
          title: The issue's Log stops at boundary review round 2; round 3's verdict and its R11-R15 work are recorded everywhere except the tracker
          detail: workshop/issues/000040-form-board.md:591 is the last Log section. Rounds 1 and 2 each got an entry naming the findings and the lesson; round 3 (FIX-THEN-SHIP) and the fixes for it appear only in the plan's Revisions, the close-review sidecar, the gate ledger and workshop/lessons.md. AGENTS.md section 3 puts the boundary outcome in the issue's own Log, which is the surface a reader reaches first.
          family: gate-round-outcome-unlogged
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-09-01T18:13:09-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: addressed
          note: Verified by revert in a scratch worktree — both remaining rows redden their own test.
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: All five code comments unchanged (board.go:276/510/558, play_loop.go:677/709); README:101 unchanged; atlas:2069's claim became true only because the CODE moved, while atlas:2061 became false in the same commit.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: The issue's Log still ends at the operator's sitting; rounds 3, 4 and R17 are unrecorded there.
          round: 5
      findings:
        - id: BR-17
          severity: Important
          title: R17 changed Enter's contract and moved Board.Resize, and swept only the plan's Revisions — seven live artifacts still state the old behaviour
          detail: 'FIFTH finding in this family — do NOT patch the seven sites. Measured set, all current: play_loop.go:289-298 ("AND THE FORM, if it lays itself out (R9)... `sz.cols` rather than `opt.width`") sits directly above `// NO Resize HERE` with no separator, and neither a Resize nor sz.cols reaches the board there any more; play_loop.go:624-632 and atlas/define.md:2057 still call the dropped rows "harmless in sequence", the exact sentence R17''s own commit message identifies as the error; atlas/define.md:2061-2062 names "the loop''s resize case" as Board.Resize''s call site; atlas and README say nothing about display.Size() or the Enter refusal; README:133 and :149 state Enter''s contract unconditionally; plan:411 and issue:82 do the same and neither R17 test is cited anywhere in the plan. THE RULE: a decision that changes a drawn or keyed contract has an enumerable sweep set that is the same every time — the comment at the site the mechanism moved FROM, atlas/define.md, README prose + key table + example block, and both Done-when tables. Write that enumeration into R18 and run it in the same commit. TestPlanCitesTestsThatExist catches an unresolvable citation; nothing catches shipped behaviour with no citation, which is what happened here.'
          family: comment-asserts-absent-behaviour
          round: 5
        - id: BR-18
          severity: Important
          title: The README's only picture of the board renders the design R16 deleted, and nothing derives it
          detail: 'SIXTH in the family, and a distinct trigger from the one above. cmd/define/README.md:107-113 shows `[y] keel`, `[n] quokka`, `[n] sycophantic`, `[y] run` — the mark standing where the key was — and a label row `[c] [e] [f] [g]`, the old sequence with its hole at `d`. R16 deleted both after the operator''s sitting, and the README''s own prose contradicts the picture eight lines below (`:124` "0-9 then a-f, in order and with no gaps"; `:128` "keeps its key"). doc_sync_test.go pins the prompt LINE only, so the grid block is a hand-maintained restatement with no consumer and no guard. The enforceable fix is to make it derive: build a play.Board over the block''s own words, mark the cells the block shows marked, and assert the fenced block''s cell lines equal board.Prompt()''s grid rows.'
          family: comment-asserts-absent-behaviour
          round: 5
        - id: BR-19
          severity: Minor
          title: boardWhole is measured against gradePrompt while the frame draws boardPrompt, and the two differ by a row at some widths
          detail: 'THIRD in this family — do not special-case the widths. play_loop.go:203 charges the frame `displayRows(gradePrompt(q), termCols)` and then draws `boardPrompt(q, boardWhole)`. Measured: gradePrompt is 78 visible columns and the refusal row is 79, so displayRows disagrees at cols 78 (1 vs 2), 39 (2 vs 3) and 26 (3 vs 4) — the drawn prompt is one row taller than the budget charged and one extra footer row is dropped. Bounded to cosmetics: Enter is already held in that state and an unpainted row is unclickable, and because the refusal is never shorter the error is always in the safe direction. The rule is the family''s own — the budget must measure the string that will actually be drawn, so compute the prompt first and measure THAT, rather than measuring a sibling of it.'
          family: frame-budget-hardcoded-not-measured
          round: 5
        - id: BR-20
          severity: Minor
          title: The board's fit formula is spelled twice — at selection in boardFits and at draw in show() — and only one copy enforces minWrapWidth
          detail: 'play_loop.go:833 (boardFits) and play_loop.go:203 (show) both spell `fitsABoard(rows, board.Rows(), displayRows(gradePrompt(board), cols))`. ARCH-DRY: one helper taking (form, rows, cols) asked at both moments. The divergence is already visible — boardFits refuses `opt.width < minWrapWidth` and the draw-time copy does not, so a board narrowed below minWrapWidth by resize can still report whole. Not reachable as harm today (the row arithmetic makes boardWhole false well before the words become unreadable), which is exactly why it should be consolidated before it is.'
          family: one-measurement-two-owners
          round: 5
        - id: BR-21
          severity: Minor
          title: atlas/repo-guards.md's guard inventory does not list TestPlanCitesTestsThatExist, added in this window
          detail: 'The table at atlas/repo-guards.md:107-114 inventories the repo guards and includes TestPlanTablesNameEntitiesThatExist, the new guard''s direct sibling. TestPlanCitesTestsThatExist shipped in 80a4044 as BR-3''s class fix and has no row. An inventory that under-states is the same failure as prose that over-states: the next reader cannot tell what is guarded.'
          family: comment-asserts-absent-behaviour
          round: 5
      blocked: false
    - "n": 6
      timestamp: "2026-09-01T20:52:25-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
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

## Round 3 — 2026-09-01T13:42:55-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — wrong-word click closed and pinned; the layout-width and fit-after-resize rows of its own enumeration are still open — measured
- BR-9 — addressed — verified by planting BR-3's shape in a scratch plan: three guards now FAIL naming the swallowed section
- BR-10 — addressed — the Batch doc points at numInputKinds and the table test; remaining "four" mentions are records or the true batchOf call-site count
- BR-11 — addressed — the reveals field is gone and Revealed is asserted unconditionally for every kind

### Raised

- **BR-12** [Important] `assertion-cannot-fail` TestANarrowingResizeKeepsTheBoardsClickMapHonest asserts over an empty event set, so R9's click-map claim is unpinned at the loop
  2nd finding in family assertion-cannot-fail. The rule, not the instance: a test whose subject is an
  EVENT must assert the event happened — `for _, e := range events` with no count check certifies nothing.
  Measured: instrumented and run three times, the test records 0 review events every time, and it still
  passes when formCell is stubbed to return false unconditionally. The click never lands because the
  goroutine derives its row from strings.Split(frame, "\r\n") — logical writes — while FooterRowAt works in
  physical rows, and the 76-column keys prompt wraps to two at width 40. Same lesson the issue's Log
  records for T13 ("the click's ROW and COLUMN are read off the paint rather than computed"), unapplied
  one test over. The relayout and row-width halves of the test are real and do fail without the fix.
- **BR-13** [Minor] `comment-asserts-absent-behaviour` Two live prose enumerations restate sets the code owns: the plan's Grid row omits Resize, and boardFooter plus atlas claim a board is never in a footer that drops rows
  3rd finding in family. The rule: prose restating a set the code owns is a second owner and drifts —
  point at the type. plan.md:327 lists Rows, CellAt, Mark for an interface that has had four methods since
  R9 added Resize. play_loop.go:509-511 and atlas/define.md:2068 assert "a board is never IN a footer that
  has to drop anything", which measurement contradicts after a resize (toggle dropped at 10x40 and 12x24).
  Neither is guarded: TestPlanTableStatusMatchesTheChangeWindow reads the name|file|status cells only.
- **BR-14** [Minor] `plan-citations-unenforced` currentTruthOnly has two discarding rules and R10 gave a premise assertion to only one
  3rd finding in family. The closed-section rule (repo_guard_test.go:640-646) still discards silently, and
  because it splits on "\n### " a closed section swallows everything up to the NEXT "### " — which can
  include a following top-level "## " section. No live instance in the tree today (define-learn.md's closed
  sections already sit below the ## Log truncation), so this is the rule R10 wrote applied to every rule the
  filter has, rather than a present defect.

## Round 4 — 2026-09-01T14:45:56-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — Rows 2 and 3 of its own enumeration are still open and now measured: a board that becomes current after a resize is painted with 73-column rows into a 40-column terminal (g.Resize is called at play_loop.go:257 for s.Current() only), and after a shrinking resize Enter records Wrong for every word fitFooter dropped (12x24 -> 8 of 16 undrawn, all swept). The plan, the atlas and boardFooter's comment now record those losses as harmless.
- BR-12 — addressed — Verified twice by revert: stubbing formCell to false reddens the test in 0.4s, removing g.Resize(sz.cols) reddens it in 2.4s, and the defer close(keys) means it fails rather than hangs. 10/10 stable, 5/5 under -race.
- BR-13 — addressed — The plan's Grid row points at the type instead of listing methods; boardFooter's comment and atlas/define.md now say a board CAN end up in a footer that drops rows. Five NEW stale toggle-row comments raised separately as the family's 4th finding.
- BR-14 — addressed — Verified by revert - the closed-section premise assertion fires on the real atlas/repo-guards.md under the old strings.Contains spelling, and with closedSection's line-start match a planted retired symbol on that page's head section is now caught where it previously was invisible.

### Raised

- **BR-15** [Minor] `comment-asserts-absent-behaviour` Five code comments still describe the deleted footer toggle row, and two docs claims contradict measurement
  4th finding in this family - do not patch the sites. Instances - board.go:239 (Rows() "counts the toggle and the panel too", it counts a blank and the panel, contradicting chromeRows twenty lines below), board.go:453-456 (Mark - "drawn in the footer's toggle"), board.go:501 (CellAt - "the blank, the toggle, the panel"), play_loop.go:573 ("a footer row below Rows() is the toggle or the bar"), play_loop.go:605; plus atlas/define.md:2065 asserting every quantity is read at draw time (false, see BR-8) and README.md:101 saying "recalled three times or more" where boardBox is a BOX, so a word recalled five times and missed twice is below the threshold. The rule is already written down; what is missing is the trigger. TestARemovedDeclarationIsSweptOrRetired skips toggleLine because isCitableName filters unexported names, and no guard can catch prose naming a CONCEPT rather than a symbol - so the enumerable half is that deleting a drawn element makes `grep -w <its word>` over currentTruthFiles the sweep set, run in the same commit. TestTheToggleAndPanelRowsAreNotCells is itself named after the removed row.
- **BR-16** [Minor] `gate-round-outcome-unlogged` The issue's Log stops at boundary review round 2; round 3's verdict and its R11-R15 work are recorded everywhere except the tracker
  workshop/issues/000040-form-board.md:591 is the last Log section. Rounds 1 and 2 each got an entry naming the findings and the lesson; round 3 (FIX-THEN-SHIP) and the fixes for it appear only in the plan's Revisions, the close-review sidecar, the gate ledger and workshop/lessons.md. AGENTS.md section 3 puts the boundary outcome in the issue's own Log, which is the surface a reader reaches first.

## Round 5 — 2026-09-01T18:13:09-07:00 (claude) — passed

### Disposed

- BR-8 — addressed — Verified by revert in a scratch worktree — both remaining rows redden their own test.
- BR-15 — not-addressed — All five code comments unchanged (board.go:276/510/558, play_loop.go:677/709); README:101 unchanged; atlas:2069's claim became true only because the CODE moved, while atlas:2061 became false in the same commit.
- BR-16 — not-addressed — The issue's Log still ends at the operator's sitting; rounds 3, 4 and R17 are unrecorded there.

### Raised

- **BR-17** [Important] `comment-asserts-absent-behaviour` R17 changed Enter's contract and moved Board.Resize, and swept only the plan's Revisions — seven live artifacts still state the old behaviour
  FIFTH finding in this family — do NOT patch the seven sites. Measured set, all current: play_loop.go:289-298 ("AND THE FORM, if it lays itself out (R9)... `sz.cols` rather than `opt.width`") sits directly above `// NO Resize HERE` with no separator, and neither a Resize nor sz.cols reaches the board there any more; play_loop.go:624-632 and atlas/define.md:2057 still call the dropped rows "harmless in sequence", the exact sentence R17's own commit message identifies as the error; atlas/define.md:2061-2062 names "the loop's resize case" as Board.Resize's call site; atlas and README say nothing about display.Size() or the Enter refusal; README:133 and :149 state Enter's contract unconditionally; plan:411 and issue:82 do the same and neither R17 test is cited anywhere in the plan. THE RULE: a decision that changes a drawn or keyed contract has an enumerable sweep set that is the same every time — the comment at the site the mechanism moved FROM, atlas/define.md, README prose + key table + example block, and both Done-when tables. Write that enumeration into R18 and run it in the same commit. TestPlanCitesTestsThatExist catches an unresolvable citation; nothing catches shipped behaviour with no citation, which is what happened here.
- **BR-18** [Important] `comment-asserts-absent-behaviour` The README's only picture of the board renders the design R16 deleted, and nothing derives it
  SIXTH in the family, and a distinct trigger from the one above. cmd/define/README.md:107-113 shows `[y] keel`, `[n] quokka`, `[n] sycophantic`, `[y] run` — the mark standing where the key was — and a label row `[c] [e] [f] [g]`, the old sequence with its hole at `d`. R16 deleted both after the operator's sitting, and the README's own prose contradicts the picture eight lines below (`:124` "0-9 then a-f, in order and with no gaps"; `:128` "keeps its key"). doc_sync_test.go pins the prompt LINE only, so the grid block is a hand-maintained restatement with no consumer and no guard. The enforceable fix is to make it derive: build a play.Board over the block's own words, mark the cells the block shows marked, and assert the fenced block's cell lines equal board.Prompt()'s grid rows.
- **BR-19** [Minor] `frame-budget-hardcoded-not-measured` boardWhole is measured against gradePrompt while the frame draws boardPrompt, and the two differ by a row at some widths
  THIRD in this family — do not special-case the widths. play_loop.go:203 charges the frame `displayRows(gradePrompt(q), termCols)` and then draws `boardPrompt(q, boardWhole)`. Measured: gradePrompt is 78 visible columns and the refusal row is 79, so displayRows disagrees at cols 78 (1 vs 2), 39 (2 vs 3) and 26 (3 vs 4) — the drawn prompt is one row taller than the budget charged and one extra footer row is dropped. Bounded to cosmetics: Enter is already held in that state and an unpainted row is unclickable, and because the refusal is never shorter the error is always in the safe direction. The rule is the family's own — the budget must measure the string that will actually be drawn, so compute the prompt first and measure THAT, rather than measuring a sibling of it.
- **BR-20** [Minor] `one-measurement-two-owners` The board's fit formula is spelled twice — at selection in boardFits and at draw in show() — and only one copy enforces minWrapWidth
  play_loop.go:833 (boardFits) and play_loop.go:203 (show) both spell `fitsABoard(rows, board.Rows(), displayRows(gradePrompt(board), cols))`. ARCH-DRY: one helper taking (form, rows, cols) asked at both moments. The divergence is already visible — boardFits refuses `opt.width < minWrapWidth` and the draw-time copy does not, so a board narrowed below minWrapWidth by resize can still report whole. Not reachable as harm today (the row arithmetic makes boardWhole false well before the words become unreadable), which is exactly why it should be consolidated before it is.
- **BR-21** [Minor] `comment-asserts-absent-behaviour` atlas/repo-guards.md's guard inventory does not list TestPlanCitesTestsThatExist, added in this window
  The table at atlas/repo-guards.md:107-114 inventories the repo guards and includes TestPlanTablesNameEntitiesThatExist, the new guard's direct sibling. TestPlanCitesTestsThatExist shipped in 80a4044 as BR-3's class fix and has no row. An inventory that under-states is the same failure as prose that over-states: the next reader cannot tell what is guarded.

## Round 6 — 2026-09-01T20:52:25-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-15** [Minor] `comment-asserts-absent-behaviour` Five code comments still describe the deleted footer toggle row, and two docs claims contradict measurement
- **BR-16** [Minor] `gate-round-outcome-unlogged` The issue's Log stops at boundary review round 2; round 3's verdict and its R11-R15 work are recorded everywhere except the tracker
- **BR-17** [Important] `comment-asserts-absent-behaviour` R17 changed Enter's contract and moved Board.Resize, and swept only the plan's Revisions — seven live artifacts still state the old behaviour
- **BR-18** [Important] `comment-asserts-absent-behaviour` The README's only picture of the board renders the design R16 deleted, and nothing derives it
- **BR-19** [Minor] `frame-budget-hardcoded-not-measured` boardWhole is measured against gradePrompt while the frame draws boardPrompt, and the two differ by a row at some widths
- **BR-20** [Minor] `one-measurement-two-owners` The board's fit formula is spelled twice — at selection in boardFits and at draw in show() — and only one copy enforces minWrapWidth
- **BR-21** [Minor] `comment-asserts-absent-behaviour` atlas/repo-guards.md's guard inventory does not list TestPlanCitesTestsThatExist, added in this window
