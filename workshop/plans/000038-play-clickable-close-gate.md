---
gate: boundary-review
issue: 38
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-31T20:24:15-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: T4 and T5 specify region coordinates against premises the screen and Render do not hold
          detail: |-
            Third finding in this family, so the deliverable is the enumeration, not the
            instance. Rule: a task adopting an existing mechanism carries that mechanism's
            HEAD-documented obligations as checkable items. For WriteRegions/Render that is
            four rows, and the plan currently misses three. Measured: the prompt and reveal
            writes both lead with a newline (play_loop.go:167, :317) while addRegions anchors
            at the render's start (screen.go:149-168), so the specified Line 0 lands on the
            blank line above the word; todaysQuestions passes no Word (play_loop.go:430-432)
            though RenderOpts.Word documents empty as "no click map wanted" (render.go:15-25);
            and regions built at queue-build time against startup opt.width cannot survive the
            pinned screen's write-time wrap (screen.go:591-595) across a resize. None of the
            three reddens the pins named in Done-when 1 and 3, which no test is required to
            drive through a real screen — the repo's own joint rule is editorloop_test.go:891-902.
            (carried from plan-quality PQ-12, deferred to the boundary review)
          family: mechanism-adopted-without-its-obligations
          round: 1
        - id: BR-2
          severity: Minor
          title: ten of eighteen line anchors in the plan's current-truth sections are wrong after 41 landed
          detail: |-
            Second in this family, so the rule rather than the instance: anchor a current-truth
            claim by SYMBOL or test name, which TestPlanTablesNameEntitiesThatExist and
            TestPlanNamedTestsExist guard mechanically and which pass at HEAD, instead of by
            line number, which nothing guards. Measured at HEAD: play_loop.go 262-265, 174-198,
            178-181, 161 and replraw.go 264, 170, 140, 94, 534, 366-380 all moved or vanished;
            two of those rows are now false in substance, since the playback dance is deleted
            and viewportGesture is already shared with --play.
            (carried from plan-quality PQ-13, deferred to the boundary review)
          family: citation-does-not-point-at-the-claim
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-31T20:24:15-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Critical
          title: writeClickable is handed opt.width while the screen wraps at its own cols, so a resize MISPLACES the click map instead of dropping it
          detail: |-
            play_loop.go:179 and :358 pass opt.width, fixed at startup by terminalWidth
            and never updated; liveScreen.writeBuffer wraps at l.cols, which Resize does
            update. Reproduced in the real loop (start 80 cols, Resize to 40, reveal):
            all three regions land on lines that do not contain their text - a headword
            region on a blank line, the ORIGIN "French" region on a quotation line. That
            is the wrong-click failure wrapMovedRegions' own comment, plan Done-when 3a,
            atlas and README all promise cannot happen, and the loop's own resize comment
            says a second width here "would be a second answer to the same question".
            Fix by moving the region-moving into liveScreen.WriteRegions, which already
            holds l.cols and already routes through writeBuffer, and add a loop-level
            resize-then-reveal test asserting every region still underlines its own text.
          family: wrap-width-single-owner
          round: 2
        - id: BR-4
          severity: Important
          title: atlas and README still describe the all-or-nothing wrap rule that commit 70b18a5 replaced
          detail: |-
            atlas/define.md:2055-2062 and cmd/define/README.md:159-160 both say the map is
            passed along "only if it changed nothing" and that the underlines stop after a
            narrowing resize. The rule is now per line, and the resize claim is false as
            written. 70b18a5 updated playbar.go, the tests, lessons.md and the plan prose
            but neither doc.
          family: docs-lag-behavior-change
          round: 2
        - id: BR-5
          severity: Important
          title: Done-when row 3a names TestClickMapIsDroppedRatherThanMisplacedByAWrap, which does not exist
          detail: |-
            The test in the tree is TestAWrapMovesTheClickMapRatherThanDroppingIt, and the
            row's claim ("DROPPED rather than misplaced") no longer states the rule either.
            Same family the plan flags against itself, in the document whose latest revision
            is titled "the Done-when names the tests that exist".
          family: plan-table-vs-tree
          round: 2
        - id: BR-6
          severity: Minor
          title: replayInPlace's doc comment now heads playRegion, leaving replayInPlace undocumented
          detail: |-
            replraw.go:554-563 runs straight into playRegion's comment with no blank line,
            so godoc reads "replayInPlace speaks the current word again..." as playRegion's
            doc.
          family: doc-comment-attachment
          round: 2
        - id: BR-7
          severity: Minor
          title: todaysQuestions builds two marks maps where one would do
          detail: |-
            play_loop.go:447 initialises held.marks, :465 builds a second local map, :500
            assigns it over the first. Build into held.marks directly.
          family: redundant-duplicate-state
          round: 2
        - id: BR-8
          severity: Minor
          title: a sitting's ORIGIN-language click always passes an empty entry, degrading to the headword fallback
          detail: |-
            play_loop.go:270 passes entry "", so utteranceFor gets ParseEntry("") and plays
            the English spelling in the origin voice. todaysQuestions has the raw text at
            :475 and discards it; one field on clickable would close it. The README claims
            the reveal is clickable "exactly as in the interactive session", where the
            current entry's source spellings are used.
          family: available-context-discarded
          round: 2
        - id: BR-9
          severity: Minor
          title: playRegion writes nothingToReplay with \n while replayInPlace writes the same constant with \r\n
          detail: |-
            replraw.go:584 vs :612. Invisible today because both stream to the screen,
            which normalises, but the constant now has two spellings of its terminator.
          family: line-ending-single-owner
          round: 2
        - id: BR-10
          severity: Minor
          title: 'the issue is still status: punt with every Plan row ticked'
          detail: |-
            workshop/issues/000038-play-clickable.md:3 - the resume never moved the status
            off punt.
          family: tracker-state-stale
          round: 2
      blocked: true
---

# Gate ledger — tools#38 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T20:24:15-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `mechanism-adopted-without-its-obligations` T4 and T5 specify region coordinates against premises the screen and Render do not hold
  Third finding in this family, so the deliverable is the enumeration, not the
  instance. Rule: a task adopting an existing mechanism carries that mechanism's
  HEAD-documented obligations as checkable items. For WriteRegions/Render that is
  four rows, and the plan currently misses three. Measured: the prompt and reveal
  writes both lead with a newline (play_loop.go:167, :317) while addRegions anchors
  at the render's start (screen.go:149-168), so the specified Line 0 lands on the
  blank line above the word; todaysQuestions passes no Word (play_loop.go:430-432)
  though RenderOpts.Word documents empty as "no click map wanted" (render.go:15-25);
  and regions built at queue-build time against startup opt.width cannot survive the
  pinned screen's write-time wrap (screen.go:591-595) across a resize. None of the
  three reddens the pins named in Done-when 1 and 3, which no test is required to
  drive through a real screen — the repo's own joint rule is editorloop_test.go:891-902.
  (carried from plan-quality PQ-12, deferred to the boundary review)
- **BR-2** [Minor] `citation-does-not-point-at-the-claim` ten of eighteen line anchors in the plan's current-truth sections are wrong after 41 landed
  Second in this family, so the rule rather than the instance: anchor a current-truth
  claim by SYMBOL or test name, which TestPlanTablesNameEntitiesThatExist and
  TestPlanNamedTestsExist guard mechanically and which pass at HEAD, instead of by
  line number, which nothing guards. Measured at HEAD: play_loop.go 262-265, 174-198,
  178-181, 161 and replraw.go 264, 170, 140, 94, 534, 366-380 all moved or vanished;
  two of those rows are now false in substance, since the playback dance is deleted
  and viewportGesture is already shared with --play.
  (carried from plan-quality PQ-13, deferred to the boundary review)

## Round 2 — 2026-08-31T20:24:15-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Critical] `wrap-width-single-owner` writeClickable is handed opt.width while the screen wraps at its own cols, so a resize MISPLACES the click map instead of dropping it
  play_loop.go:179 and :358 pass opt.width, fixed at startup by terminalWidth
  and never updated; liveScreen.writeBuffer wraps at l.cols, which Resize does
  update. Reproduced in the real loop (start 80 cols, Resize to 40, reveal):
  all three regions land on lines that do not contain their text - a headword
  region on a blank line, the ORIGIN "French" region on a quotation line. That
  is the wrong-click failure wrapMovedRegions' own comment, plan Done-when 3a,
  atlas and README all promise cannot happen, and the loop's own resize comment
  says a second width here "would be a second answer to the same question".
  Fix by moving the region-moving into liveScreen.WriteRegions, which already
  holds l.cols and already routes through writeBuffer, and add a loop-level
  resize-then-reveal test asserting every region still underlines its own text.
- **BR-4** [Important] `docs-lag-behavior-change` atlas and README still describe the all-or-nothing wrap rule that commit 70b18a5 replaced
  atlas/define.md:2055-2062 and cmd/define/README.md:159-160 both say the map is
  passed along "only if it changed nothing" and that the underlines stop after a
  narrowing resize. The rule is now per line, and the resize claim is false as
  written. 70b18a5 updated playbar.go, the tests, lessons.md and the plan prose
  but neither doc.
- **BR-5** [Important] `plan-table-vs-tree` Done-when row 3a names TestClickMapIsDroppedRatherThanMisplacedByAWrap, which does not exist
  The test in the tree is TestAWrapMovesTheClickMapRatherThanDroppingIt, and the
  row's claim ("DROPPED rather than misplaced") no longer states the rule either.
  Same family the plan flags against itself, in the document whose latest revision
  is titled "the Done-when names the tests that exist".
- **BR-6** [Minor] `doc-comment-attachment` replayInPlace's doc comment now heads playRegion, leaving replayInPlace undocumented
  replraw.go:554-563 runs straight into playRegion's comment with no blank line,
  so godoc reads "replayInPlace speaks the current word again..." as playRegion's
  doc.
- **BR-7** [Minor] `redundant-duplicate-state` todaysQuestions builds two marks maps where one would do
  play_loop.go:447 initialises held.marks, :465 builds a second local map, :500
  assigns it over the first. Build into held.marks directly.
- **BR-8** [Minor] `available-context-discarded` a sitting's ORIGIN-language click always passes an empty entry, degrading to the headword fallback
  play_loop.go:270 passes entry "", so utteranceFor gets ParseEntry("") and plays
  the English spelling in the origin voice. todaysQuestions has the raw text at
  :475 and discards it; one field on clickable would close it. The README claims
  the reveal is clickable "exactly as in the interactive session", where the
  current entry's source spellings are used.
- **BR-9** [Minor] `line-ending-single-owner` playRegion writes nothingToReplay with \n while replayInPlace writes the same constant with \r\n
  replraw.go:584 vs :612. Invisible today because both stream to the screen,
  which normalises, but the constant now has two spellings of its terminator.
- **BR-10** [Minor] `tracker-state-stale` the issue is still status: punt with every Plan row ticked
  workshop/issues/000038-play-clickable.md:3 - the resume never moved the status
  off punt.

## Open findings

- **BR-1** [Important] `mechanism-adopted-without-its-obligations` T4 and T5 specify region coordinates against premises the screen and Render do not hold
- **BR-2** [Minor] `citation-does-not-point-at-the-claim` ten of eighteen line anchors in the plan's current-truth sections are wrong after 41 landed
- **BR-3** [Critical] `wrap-width-single-owner` writeClickable is handed opt.width while the screen wraps at its own cols, so a resize MISPLACES the click map instead of dropping it
- **BR-4** [Important] `docs-lag-behavior-change` atlas and README still describe the all-or-nothing wrap rule that commit 70b18a5 replaced
- **BR-5** [Important] `plan-table-vs-tree` Done-when row 3a names TestClickMapIsDroppedRatherThanMisplacedByAWrap, which does not exist
- **BR-6** [Minor] `doc-comment-attachment` replayInPlace's doc comment now heads playRegion, leaving replayInPlace undocumented
- **BR-7** [Minor] `redundant-duplicate-state` todaysQuestions builds two marks maps where one would do
- **BR-8** [Minor] `available-context-discarded` a sitting's ORIGIN-language click always passes an empty entry, degrading to the headword fallback
- **BR-9** [Minor] `line-ending-single-owner` playRegion writes nothingToReplay with \n while replayInPlace writes the same constant with \r\n
- **BR-10** [Minor] `tracker-state-stale` the issue is still status: punt with every Plan row ticked
