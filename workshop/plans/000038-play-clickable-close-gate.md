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
    - "n": 3
      timestamp: "2026-08-31T22:34:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: code side fixed (Line 1 and the screen-width ruler are both mutation-pinned; Word:key is present but unpinned), but T4/T5 in `## Tasks` are byte-identical to before — the commit's "All three are rows in T5 now" did not happen.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: the current-truth table was not touched; 8 of 10 anchors I re-measured at HEAD point at unrelated lines, and two rows are false in substance.
          round: 3
        - id: BR-3
          disposition: addressed
          note: mutation-verified twice in a scratch copy — deleting the call and substituting a fixed 80 for l.cols each redden two tests.
          round: 3
        - id: BR-4
          disposition: addressed
          note: atlas/define.md and README.md both rewritten to the per-line rule and the one-ruler rule; one new inaccuracy in the README raised separately.
          round: 3
        - id: BR-5
          disposition: addressed
          note: row 3a renamed and restated, row 3b added; TestPlanNamedTestsExist passes on the plan once the task boxes are ticked.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: replraw.go:554-563 still runs into playRegion's comment with no blank line; replayInPlace's own doc at :613 does not name it.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: play_loop.go still initialises held.marks at the constructor, builds a second local map, and assigns it over the first.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: play_loop.go:272 still passes entry "" — and this is the issue's stated parity promise, not an extension (ARCH-PURPOSE).
          round: 3
        - id: BR-9
          disposition: addressed
          note: both sites now use fmt.Fprintln; the \r\n spelling is gone from this path.
          round: 3
        - id: BR-10
          disposition: addressed
          note: 'issue frontmatter is status: working; the wider sweep it was one slot of is raised as a new finding.'
          round: 3
      findings:
        - id: BR-11
          severity: Critical
          title: the plan's Core-concepts PURE table names playRegions and promptRegionFor, neither of which the tree declares
          detail: |-
            This is the 2nd finding in family `plan-table-vs-tree`. Earlier rounds fixed
            instances (BR-5 renamed one Done-when cell). Do NOT fix this instance alone —
            the rule is that EVERY identifier a plan states as current truth is checked
            against the tree in one sweep at the boundary, and the repo already owns the
            enumerator: TestPlanTablesNameEntitiesThatExist for Core-concepts cells,
            TestPlanNamedTestsExist for backticked test names. Both are currently
            suppressed on this plan by its unticked task boxes.
            Measured: the delivered entities are `clickable` (play_loop.go:519) and
            `(*sittingDeck).marksIn` (play_loop.go:581); the prompt region is built inline
            in `show()` and has no named function. Reproduced in a scratch copy — with the
            boxes as they are the guard PASSES; tick them (which `sdlc close`'s
            plan-unchecked gate requires) and it fails on both names. So the close cannot
            be recorded without either a red suite or a corrected table.
          family: plan-table-vs-tree
          round: 3
        - id: BR-12
          severity: Important
          title: the plan's Tasks still show T0/T1/T4/T5/T6/T7/T8 unticked while the issue ticks all nine and the code has landed
          detail: |-
            This is the 2nd finding in family `tracker-state-stale`. BR-10 fixed one
            instance (issue frontmatter status). Do NOT fix this instance alone — the rule:
            an issue's completion state lives in FOUR slots and is swept as one enumeration
            at the boundary: issue frontmatter `status:`, the issue's `## Plan` boxes, the
            plan's `## Tasks` boxes, and the referencing project row. Measured at HEAD: slot
            1 fixed by BR-10; slot 3 wrong (plan lines 120-136, including T6 which the issue
            marks landed by `#41`); slot 4 never checked — workshop/projects/define-learn.md
            carries a `[tools#41]` row and no `[tools#38]` row at all. 2 of 4 wrong after a
            round that named one of them.
            This is also what hides the Critical above: TestPlanTablesNameEntitiesThatExist
            exempts `new` rows while a plan has unticked steps, and TestPlanNamedTestsExist
            skips the document entirely ("no finished unit of work names a test").
          family: tracker-state-stale
          round: 3
        - id: BR-13
          severity: Minor
          title: Choice.Prompt's doc still says the region is line 0 and that 38 is PARKED, and the README says links follow the text as it re-wraps
          detail: |-
            This is the 2nd finding in family `docs-lag-behavior-change`. BR-4 fixed the
            atlas and README paragraphs for the wrap rule. Do NOT fix these two instances
            alone — the rule: a behaviour change sweeps every prose site that NAMES the
            behaviour, located by grepping the changed symbol and the issue number, not by
            recalling which doc mentioned it. Measured this round: play/choice.go:113-114
            states "computes its region as line 0, column 0" and "#38 is PARKED" (it is line
            1 and the issue is working) — a comment written specifically to protect this
            invariant, so a stale one defeats its only purpose; and README.md:159 "Narrow
            the window and the links follow the text as it re-wraps" is false, since Resize
            only updates l.rows/l.cols and neither existing buffer lines nor existing
            regions move — only writes made after the resize are affected.
          family: docs-lag-behavior-change
          round: 3
        - id: BR-14
          severity: Minor
          title: three copies of the audio-off guard plus its identical message, where D11 justified the copies by claiming callers keep their own
          detail: |-
            replraw.go:584 (playRegion), replraw.go:616 (replayInPlace) and repl.go:313
            (replayPiped) each spell `if !opt.playsAudio() { Fprintln(stderr,
            nothingToReplay) }`. D11 left the message with the callers on the grounds that
            "callers keep their own MESSAGES; only the condition is shared" — but all three
            messages are the same constant, so the premise does not hold for these three.
            This window added the third. ARCH-DRY.
          family: duplicated-guard-and-message
          round: 3
        - id: BR-15
          severity: Minor
          title: the RenderOpts.Word obligation is fixed in code but no test fails without it
          detail: |-
            Deleting `Word: key` from the Render call in todaysQuestions leaves the whole
            cmd/define suite green — I ran it with that mutation and the only failures were
            the git-dependent repo guards (the scratch copy has no .git). regionsIn falls
            back to e.Headword(), so the divergence the fix commit calls "a live defect — a
            click on a normalised deck word would have fetched the wrong recording" is
            unpinned. A row asserting Region.Word == the deck key when key and headword
            differ (jalapeno / jalapeño) would close it. Same family as BR-1 because it is
            the same obligation, one step further on: adopted, but not made falsifiable.
          family: mechanism-adopted-without-its-obligations
          round: 3
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

## Round 3 — 2026-08-31T22:34:51-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — code side fixed (Line 1 and the screen-width ruler are both mutation-pinned; Word:key is present but unpinned), but T4/T5 in `## Tasks` are byte-identical to before — the commit's "All three are rows in T5 now" did not happen.
- BR-2 — not-addressed — the current-truth table was not touched; 8 of 10 anchors I re-measured at HEAD point at unrelated lines, and two rows are false in substance.
- BR-3 — addressed — mutation-verified twice in a scratch copy — deleting the call and substituting a fixed 80 for l.cols each redden two tests.
- BR-4 — addressed — atlas/define.md and README.md both rewritten to the per-line rule and the one-ruler rule; one new inaccuracy in the README raised separately.
- BR-5 — addressed — row 3a renamed and restated, row 3b added; TestPlanNamedTestsExist passes on the plan once the task boxes are ticked.
- BR-6 — not-addressed — replraw.go:554-563 still runs into playRegion's comment with no blank line; replayInPlace's own doc at :613 does not name it.
- BR-7 — not-addressed — play_loop.go still initialises held.marks at the constructor, builds a second local map, and assigns it over the first.
- BR-8 — not-addressed — play_loop.go:272 still passes entry "" — and this is the issue's stated parity promise, not an extension (ARCH-PURPOSE).
- BR-9 — addressed — both sites now use fmt.Fprintln; the \r\n spelling is gone from this path.
- BR-10 — addressed — issue frontmatter is status: working; the wider sweep it was one slot of is raised as a new finding.

### Raised

- **BR-11** [Critical] `plan-table-vs-tree` the plan's Core-concepts PURE table names playRegions and promptRegionFor, neither of which the tree declares
  This is the 2nd finding in family `plan-table-vs-tree`. Earlier rounds fixed
  instances (BR-5 renamed one Done-when cell). Do NOT fix this instance alone —
  the rule is that EVERY identifier a plan states as current truth is checked
  against the tree in one sweep at the boundary, and the repo already owns the
  enumerator: TestPlanTablesNameEntitiesThatExist for Core-concepts cells,
  TestPlanNamedTestsExist for backticked test names. Both are currently
  suppressed on this plan by its unticked task boxes.
  Measured: the delivered entities are `clickable` (play_loop.go:519) and
  `(*sittingDeck).marksIn` (play_loop.go:581); the prompt region is built inline
  in `show()` and has no named function. Reproduced in a scratch copy — with the
  boxes as they are the guard PASSES; tick them (which `sdlc close`'s
  plan-unchecked gate requires) and it fails on both names. So the close cannot
  be recorded without either a red suite or a corrected table.
- **BR-12** [Important] `tracker-state-stale` the plan's Tasks still show T0/T1/T4/T5/T6/T7/T8 unticked while the issue ticks all nine and the code has landed
  This is the 2nd finding in family `tracker-state-stale`. BR-10 fixed one
  instance (issue frontmatter status). Do NOT fix this instance alone — the rule:
  an issue's completion state lives in FOUR slots and is swept as one enumeration
  at the boundary: issue frontmatter `status:`, the issue's `## Plan` boxes, the
  plan's `## Tasks` boxes, and the referencing project row. Measured at HEAD: slot
  1 fixed by BR-10; slot 3 wrong (plan lines 120-136, including T6 which the issue
  marks landed by `#41`); slot 4 never checked — workshop/projects/define-learn.md
  carries a `[tools#41]` row and no `[tools#38]` row at all. 2 of 4 wrong after a
  round that named one of them.
  This is also what hides the Critical above: TestPlanTablesNameEntitiesThatExist
  exempts `new` rows while a plan has unticked steps, and TestPlanNamedTestsExist
  skips the document entirely ("no finished unit of work names a test").
- **BR-13** [Minor] `docs-lag-behavior-change` Choice.Prompt's doc still says the region is line 0 and that 38 is PARKED, and the README says links follow the text as it re-wraps
  This is the 2nd finding in family `docs-lag-behavior-change`. BR-4 fixed the
  atlas and README paragraphs for the wrap rule. Do NOT fix these two instances
  alone — the rule: a behaviour change sweeps every prose site that NAMES the
  behaviour, located by grepping the changed symbol and the issue number, not by
  recalling which doc mentioned it. Measured this round: play/choice.go:113-114
  states "computes its region as line 0, column 0" and "#38 is PARKED" (it is line
  1 and the issue is working) — a comment written specifically to protect this
  invariant, so a stale one defeats its only purpose; and README.md:159 "Narrow
  the window and the links follow the text as it re-wraps" is false, since Resize
  only updates l.rows/l.cols and neither existing buffer lines nor existing
  regions move — only writes made after the resize are affected.
- **BR-14** [Minor] `duplicated-guard-and-message` three copies of the audio-off guard plus its identical message, where D11 justified the copies by claiming callers keep their own
  replraw.go:584 (playRegion), replraw.go:616 (replayInPlace) and repl.go:313
  (replayPiped) each spell `if !opt.playsAudio() { Fprintln(stderr,
  nothingToReplay) }`. D11 left the message with the callers on the grounds that
  "callers keep their own MESSAGES; only the condition is shared" — but all three
  messages are the same constant, so the premise does not hold for these three.
  This window added the third. ARCH-DRY.
- **BR-15** [Minor] `mechanism-adopted-without-its-obligations` the RenderOpts.Word obligation is fixed in code but no test fails without it
  Deleting `Word: key` from the Render call in todaysQuestions leaves the whole
  cmd/define suite green — I ran it with that mutation and the only failures were
  the git-dependent repo guards (the scratch copy has no .git). regionsIn falls
  back to e.Headword(), so the divergence the fix commit calls "a live defect — a
  click on a normalised deck word would have fetched the wrong recording" is
  unpinned. A row asserting Region.Word == the deck key when key and headword
  differ (jalapeno / jalapeño) would close it. Same family as BR-1 because it is
  the same obligation, one step further on: adopted, but not made falsifiable.

## Open findings

- **BR-1** [Important] `mechanism-adopted-without-its-obligations` T4 and T5 specify region coordinates against premises the screen and Render do not hold
- **BR-2** [Minor] `citation-does-not-point-at-the-claim` ten of eighteen line anchors in the plan's current-truth sections are wrong after 41 landed
- **BR-6** [Minor] `doc-comment-attachment` replayInPlace's doc comment now heads playRegion, leaving replayInPlace undocumented
- **BR-7** [Minor] `redundant-duplicate-state` todaysQuestions builds two marks maps where one would do
- **BR-8** [Minor] `available-context-discarded` a sitting's ORIGIN-language click always passes an empty entry, degrading to the headword fallback
- **BR-11** [Critical] `plan-table-vs-tree` the plan's Core-concepts PURE table names playRegions and promptRegionFor, neither of which the tree declares
- **BR-12** [Important] `tracker-state-stale` the plan's Tasks still show T0/T1/T4/T5/T6/T7/T8 unticked while the issue ticks all nine and the code has landed
- **BR-13** [Minor] `docs-lag-behavior-change` Choice.Prompt's doc still says the region is line 0 and that 38 is PARKED, and the README says links follow the text as it re-wraps
- **BR-14** [Minor] `duplicated-guard-and-message` three copies of the audio-off guard plus its identical message, where D11 justified the copies by claiming callers keep their own
- **BR-15** [Minor] `mechanism-adopted-without-its-obligations` the RenderOpts.Word obligation is fixed in code but no test fails without it
