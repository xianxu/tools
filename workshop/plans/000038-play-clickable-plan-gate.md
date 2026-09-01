---
gate: plan-quality
issue: 38
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-30T16:24:53-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: the reveal-playback restore/re-enter dance tears down the alternate screen and mouse reporting
          detail: |-
            play_loop.go:173-175 calls raw.sess.restore(), which is leaveMouse + leaveAlt
            (rawterm.go:57-61), and play_loop.go:182-197 replaces the session wholesale with a
            fresh enterRaw result whose alt and mouse flags are false. After the first miss the
            sitting is off the alt screen with clicks unreported, and the re-entered session's
            control writer becomes the liveScreen. The plan must decide the flap's fate (deleting
            it is #30 D4's continuous-raw answer) and say what happens to
            TestLosingTheTerminalAfterPlaybackExitsOne (play_loop_test.go:529) and BR-24's exit-1
            contract, which that path is the only source of.
          family: mechanism-adopted-without-its-obligations
          round: 1
        - id: PQ-2
          severity: Important
          title: writeRendered is already the region seam, and D2's claim that draw does not change is false
          detail: |-
            writeRendered (main.go:801) fills the region seam by type-assert so callers do not
            branch (#30 D6). Changing playSession to take a display duplicates that and breaks the
            fifteen call sites passing bytes.Buffer (play_loop_test.go:109-715). M1.3 and M1.4 both
            require the prompt line and the reveal to carry regions, and draw is the only writer of
            them, so draw does change — to call writeRendered while keeping io.Writer.
          family: existing-seam-not-reused
          round: 1
        - id: PQ-3
          severity: Important
          title: no task wires scroll, page or resize, which the alternate screen makes the loop's job
          detail: |-
            toInput (play_loop.go:213-229) drops everything but Interrupt/EOF/Enter/Rune, so
            KeyPageUp, KeyWheelUp and KeyWheelDown are inert; with native scrollback gone, content
            above the viewport becomes unreachable mid-sitting and the plan's own manual step
            ("scroll back with PageUp") has no implementing task. watchResize (replraw.go:52-56) is
            likewise unmentioned, and a frame drawn for a stale height scrolls the terminal, which
            is precisely the click-coordinate correctness this issue rests on.
          family: mechanism-adopted-without-its-obligations
          round: 1
        - id: PQ-4
          severity: Important
          title: --play's y/n/Enter playback is playAnnounced, not replayInPlace, so the click becomes a second caller
          detail: |-
            play_loop.go:179 calls playAnnounced with defaultIndicator(opt); replayInPlace
            (replraw.go:537-538) calls it with indicator{show:true, erase:eraseLine}. Routing the
            click through the lifted replayRegion therefore adds a second playback caller inside
            --play with different output, contradicting the issue's Done-when that a third caller
            must not appear. Move the reveal path onto replayInPlace or state why the divergence
            stands.
          family: two-callers-one-behavior
          round: 1
        - id: PQ-5
          severity: Minor
          title: replayRegion is listed as a PURE entity but performs audio playback and writes to stdout/stderr
          detail: |-
            It dispatches into replayInPlace, which plays a recording. Labelling it PURE misplaces
            where its test lives — it needs the player fake, not a colocated pure unit test.
          family: entity-table-mislabels-purity
          round: 1
        - id: PQ-6
          severity: Minor
          title: 'the issue''s ## Plan still says "decide Option A vs B / design it", which this plan file completes'
          detail: |-
            The close gate's plan-unchecked guard reads the issue's ## Plan. Its two rows describe
            work this plan file has already done, so the executable steps live only in the plan
            file and the issue records nothing tickable about the implementation.
          family: issue-plan-not-synced
          round: 1
        - id: PQ-7
          severity: Minor
          title: M1 is the only milestone and the plan says one sdlc close, so the Mx tag buys a redundant boundary
          detail: |-
            AGENTS.md section 3: an Mx tag is a review boundary that commits to its own
            milestone-close. Single-pass work takes plain checkboxes; M1.1-M1.6 can stay as the
            task list without the milestone framing.
          family: milestone-tag-without-a-boundary
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-30T16:33:43-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: D2 and T2 delete the dance outright; verified restore/enterRaw at rawterm.go:50 and :42 behave as D2 describes.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: D3 names writeRendered as the seam and keeps draw on io.Writer; the residual playSession shape moves to the family finding below.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: T6 wires page and wheel; watchResize and the stale-shape case named in the finding are still unmentioned.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: D4 moves the shared action to playAnnounced; the guard that sits above it is the family finding below.
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Revisions claims the issue's Plan was updated; the issue file still has the two decide/design rows.
          round: 2
        - id: PQ-7
          disposition: addressed
          round: 2
      findings:
        - id: PQ-8
          severity: Important
          title: playAnnounced sits below the audio-off guard, so playRegion converging there silently drops it
          detail: |-
            2nd in family. replayInPlace guards on opt.noAudio || opt.times <= 0 and prints
            nothingToReplay (replraw.go:534); playAnnounced (main.go:835) does not, and speak
            fetches unconditionally. So T1's "NO behaviour change" is false under --no-audio.
            Fix the rule, not playRegion - the predicate is hand-copied at repl.go:312,
            replraw.go:534, play_loop.go:161 and main.go:783 (prevalence 4), and playAnnounced's
            own comment at main.go:829 names this guard as a divergence it was meant to end.
            Make it one named predicate applied inside playAnnounced so no caller can be below it.
          family: two-callers-one-behavior
          round: 2
        - id: PQ-9
          severity: Important
          title: 'console, onceHandBack and watchResize are the carriers #30 built for this loop shape'
          detail: |-
            2nd in family. Round 2 reused writeRendered; the class has three members and two are
            still re-derived. playSession is planned to take "a display and the region map" where
            console (replraw.go:94) already bundles view/resizes/finish/stdout/stderr for exactly
            this reason, and T3 names handBack where onceHandBack (replraw.go:140) is the
            once-wrapper - playSession has five return paths (four after T2) plus runPlay's
            defer sess.restore() at :53, so a bare handBack can print the transcript twice.
            State the rule: adopting #30's loop shape means adopting its carriers.
          family: existing-seam-not-reused
          round: 2
        - id: PQ-10
          severity: Minor
          title: several line anchors in the verified-claims table do not point at the code they cite
          detail: |-
            restore leaving mouse/alt is cited rawterm.go:40-47, actual 50-61; enterRaw cited
            :28-34, actual 42-48; the viewport keys cited replraw.go:330, actual 366-380;
            writeRendered cited main.go:806, actual 801. Every claim is substantively true, but
            the table exists to be audited and the anchors send the auditor to the wrong lines.
          family: citation-does-not-point-at-the-claim
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-30T16:38:25-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: addressed
          note: T6 names scroll, wheel and watchResize (rawterm.go:230); Done-when rows 6 and 6b pin each half.
          round: 3
        - id: PQ-6
          disposition: not-addressed
          note: 'The issue''s ## Plan on disk still reads "Decide Option A vs B"; the round-1 Revisions entry claims otherwise.'
          round: 3
        - id: PQ-8
          disposition: not-addressed
          note: D11 guards playRegion — a 5th hand-copy — leaving the enumerable class of 4 unswept.
          round: 3
        - id: PQ-9
          disposition: addressed
          note: D10 states the carriers rule and names console, onceHandBack and watchResize with verified anchors.
          round: 3
        - id: PQ-10
          disposition: addressed
          note: All twelve table anchors re-measured against the tree; every one lands on the cited code.
          round: 3
      findings:
        - id: PQ-11
          severity: Important
          title: T2 deletes the re-entry path and with it the only test pinning the outcome-ORDER obligation
          detail: |-
            play_loop.go:126-141 enumerates three consumer obligations and names
            TestLosingTheTerminalAfterPlaybackExitsOne (play_loop_test.go:529) as the pin for
            `order`, recording that reversing the iteration once left the whole suite green
            (BR-13). That test works by handing playSession a rawTerm whose file is /dev/null so
            the post-playback enterRaw fails — a premise T2's deletion makes unreachable. T2 lists
            three other pins, none of which observe the order. The rule: a task that deletes code
            must re-home every invariant whose ONLY pin lives in that code, in the same task.
            Here the replacement is cheap — playAnnounced still blocks on speak, so a miss driven
            with a cancelled context or a failing player asserts the record was written first.
          family: deletion-drops-an-invariants-only-pin
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-30T16:42:15-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: not-addressed
          note: 'the issue''s ## Plan on disk still reads "Decide Option A vs B" / "Design if it is Option A"'
          round: 4
        - id: PQ-8
          disposition: addressed
          note: D11 + T0 move the guard inside playAnnounced as options.playsAudio(); the class of four is confirmed
          round: 4
        - id: PQ-11
          disposition: not-addressed
          note: the named replacement is green under a reversed outs iteration, so order is still unpinned
          round: 4
      blocked: false
    - "n": 5
      timestamp: "2026-08-31T18:28:30-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: addressed
          note: 'The issue''s ## Plan now carries T0–T8 as tickable rows.'
          round: 5
        - id: PQ-11
          disposition: addressed
          note: TestAMissRecordsBeforeItPlays exists at play_loop_test.go:1113 and play_loop.go:272 names it as the order pin.
          round: 5
      findings:
        - id: PQ-12
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
          family: mechanism-adopted-without-its-obligations
          round: 5
        - id: PQ-13
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
          family: citation-does-not-point-at-the-claim
          round: 5
      blocked: false
    - "n": 6
      timestamp: "2026-08-31T18:35:17-07:00"
      agent: claude
      dispose:
        - id: PQ-12
          disposition: not-addressed
          note: Plan unchanged since round 5 (last edit 3e19b95, 18:23; round 5 at 18:28). All three premises re-measured true at HEAD; a fifth row, the KeyClick dispatch, is unassigned.
          round: 6
        - id: PQ-13
          disposition: not-addressed
          note: The current-truth tables still anchor by line number; unchanged since the round that raised it.
          round: 6
      blocked: false
content_hash: e2c1aa752b2c57f1b325790f8141762fe6882dff22b4f57c75c171066273561a
---

# Gate ledger — tools#38 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T16:24:53-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `mechanism-adopted-without-its-obligations` the reveal-playback restore/re-enter dance tears down the alternate screen and mouse reporting
  play_loop.go:173-175 calls raw.sess.restore(), which is leaveMouse + leaveAlt
  (rawterm.go:57-61), and play_loop.go:182-197 replaces the session wholesale with a
  fresh enterRaw result whose alt and mouse flags are false. After the first miss the
  sitting is off the alt screen with clicks unreported, and the re-entered session's
  control writer becomes the liveScreen. The plan must decide the flap's fate (deleting
  it is #30 D4's continuous-raw answer) and say what happens to
  TestLosingTheTerminalAfterPlaybackExitsOne (play_loop_test.go:529) and BR-24's exit-1
  contract, which that path is the only source of.
- **PQ-2** [Important] `existing-seam-not-reused` writeRendered is already the region seam, and D2's claim that draw does not change is false
  writeRendered (main.go:801) fills the region seam by type-assert so callers do not
  branch (#30 D6). Changing playSession to take a display duplicates that and breaks the
  fifteen call sites passing bytes.Buffer (play_loop_test.go:109-715). M1.3 and M1.4 both
  require the prompt line and the reveal to carry regions, and draw is the only writer of
  them, so draw does change — to call writeRendered while keeping io.Writer.
- **PQ-3** [Important] `mechanism-adopted-without-its-obligations` no task wires scroll, page or resize, which the alternate screen makes the loop's job
  toInput (play_loop.go:213-229) drops everything but Interrupt/EOF/Enter/Rune, so
  KeyPageUp, KeyWheelUp and KeyWheelDown are inert; with native scrollback gone, content
  above the viewport becomes unreachable mid-sitting and the plan's own manual step
  ("scroll back with PageUp") has no implementing task. watchResize (replraw.go:52-56) is
  likewise unmentioned, and a frame drawn for a stale height scrolls the terminal, which
  is precisely the click-coordinate correctness this issue rests on.
- **PQ-4** [Important] `two-callers-one-behavior` --play's y/n/Enter playback is playAnnounced, not replayInPlace, so the click becomes a second caller
  play_loop.go:179 calls playAnnounced with defaultIndicator(opt); replayInPlace
  (replraw.go:537-538) calls it with indicator{show:true, erase:eraseLine}. Routing the
  click through the lifted replayRegion therefore adds a second playback caller inside
  --play with different output, contradicting the issue's Done-when that a third caller
  must not appear. Move the reveal path onto replayInPlace or state why the divergence
  stands.
- **PQ-5** [Minor] `entity-table-mislabels-purity` replayRegion is listed as a PURE entity but performs audio playback and writes to stdout/stderr
  It dispatches into replayInPlace, which plays a recording. Labelling it PURE misplaces
  where its test lives — it needs the player fake, not a colocated pure unit test.
- **PQ-6** [Minor] `issue-plan-not-synced` the issue's ## Plan still says "decide Option A vs B / design it", which this plan file completes
  The close gate's plan-unchecked guard reads the issue's ## Plan. Its two rows describe
  work this plan file has already done, so the executable steps live only in the plan
  file and the issue records nothing tickable about the implementation.
- **PQ-7** [Minor] `milestone-tag-without-a-boundary` M1 is the only milestone and the plan says one sdlc close, so the Mx tag buys a redundant boundary
  AGENTS.md section 3: an Mx tag is a review boundary that commits to its own
  milestone-close. Single-pass work takes plain checkboxes; M1.1-M1.6 can stay as the
  task list without the milestone framing.

## Round 2 — 2026-08-30T16:33:43-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — D2 and T2 delete the dance outright; verified restore/enterRaw at rawterm.go:50 and :42 behave as D2 describes.
- PQ-2 — addressed — D3 names writeRendered as the seam and keeps draw on io.Writer; the residual playSession shape moves to the family finding below.
- PQ-3 — not-addressed — T6 wires page and wheel; watchResize and the stale-shape case named in the finding are still unmentioned.
- PQ-4 — addressed — D4 moves the shared action to playAnnounced; the guard that sits above it is the family finding below.
- PQ-5 — addressed
- PQ-6 — not-addressed — Revisions claims the issue's Plan was updated; the issue file still has the two decide/design rows.
- PQ-7 — addressed

### Raised

- **PQ-8** [Important] `two-callers-one-behavior` playAnnounced sits below the audio-off guard, so playRegion converging there silently drops it
  2nd in family. replayInPlace guards on opt.noAudio || opt.times <= 0 and prints
  nothingToReplay (replraw.go:534); playAnnounced (main.go:835) does not, and speak
  fetches unconditionally. So T1's "NO behaviour change" is false under --no-audio.
  Fix the rule, not playRegion - the predicate is hand-copied at repl.go:312,
  replraw.go:534, play_loop.go:161 and main.go:783 (prevalence 4), and playAnnounced's
  own comment at main.go:829 names this guard as a divergence it was meant to end.
  Make it one named predicate applied inside playAnnounced so no caller can be below it.
- **PQ-9** [Important] `existing-seam-not-reused` console, onceHandBack and watchResize are the carriers #30 built for this loop shape
  2nd in family. Round 2 reused writeRendered; the class has three members and two are
  still re-derived. playSession is planned to take "a display and the region map" where
  console (replraw.go:94) already bundles view/resizes/finish/stdout/stderr for exactly
  this reason, and T3 names handBack where onceHandBack (replraw.go:140) is the
  once-wrapper - playSession has five return paths (four after T2) plus runPlay's
  defer sess.restore() at :53, so a bare handBack can print the transcript twice.
  State the rule: adopting #30's loop shape means adopting its carriers.
- **PQ-10** [Minor] `citation-does-not-point-at-the-claim` several line anchors in the verified-claims table do not point at the code they cite
  restore leaving mouse/alt is cited rawterm.go:40-47, actual 50-61; enterRaw cited
  :28-34, actual 42-48; the viewport keys cited replraw.go:330, actual 366-380;
  writeRendered cited main.go:806, actual 801. Every claim is substantively true, but
  the table exists to be audited and the anchors send the auditor to the wrong lines.

## Round 3 — 2026-08-30T16:38:25-07:00 (claude) — BLOCKED

### Disposed

- PQ-3 — addressed — T6 names scroll, wheel and watchResize (rawterm.go:230); Done-when rows 6 and 6b pin each half.
- PQ-6 — not-addressed — The issue's ## Plan on disk still reads "Decide Option A vs B"; the round-1 Revisions entry claims otherwise.
- PQ-8 — not-addressed — D11 guards playRegion — a 5th hand-copy — leaving the enumerable class of 4 unswept.
- PQ-9 — addressed — D10 states the carriers rule and names console, onceHandBack and watchResize with verified anchors.
- PQ-10 — addressed — All twelve table anchors re-measured against the tree; every one lands on the cited code.

### Raised

- **PQ-11** [Important] `deletion-drops-an-invariants-only-pin` T2 deletes the re-entry path and with it the only test pinning the outcome-ORDER obligation
  play_loop.go:126-141 enumerates three consumer obligations and names
  TestLosingTheTerminalAfterPlaybackExitsOne (play_loop_test.go:529) as the pin for
  `order`, recording that reversing the iteration once left the whole suite green
  (BR-13). That test works by handing playSession a rawTerm whose file is /dev/null so
  the post-playback enterRaw fails — a premise T2's deletion makes unreachable. T2 lists
  three other pins, none of which observe the order. The rule: a task that deletes code
  must re-home every invariant whose ONLY pin lives in that code, in the same task.
  Here the replacement is cheap — playAnnounced still blocks on speak, so a miss driven
  with a cancelled context or a failing player asserts the record was written first.

## Round 4 — 2026-08-30T16:42:15-07:00 (claude) — passed

### Disposed

- PQ-6 — not-addressed — the issue's ## Plan on disk still reads "Decide Option A vs B" / "Design if it is Option A"
- PQ-8 — addressed — D11 + T0 move the guard inside playAnnounced as options.playsAudio(); the class of four is confirmed
- PQ-11 — not-addressed — the named replacement is green under a reversed outs iteration, so order is still unpinned

## Round 5 — 2026-08-31T18:28:30-07:00 (claude) — passed

### Disposed

- PQ-6 — addressed — The issue's ## Plan now carries T0–T8 as tickable rows.
- PQ-11 — addressed — TestAMissRecordsBeforeItPlays exists at play_loop_test.go:1113 and play_loop.go:272 names it as the order pin.

### Raised

- **PQ-12** [Important] `mechanism-adopted-without-its-obligations` T4 and T5 specify region coordinates against premises the screen and Render do not hold
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
- **PQ-13** [Minor] `citation-does-not-point-at-the-claim` ten of eighteen line anchors in the plan's current-truth sections are wrong after 41 landed
  Second in this family, so the rule rather than the instance: anchor a current-truth
  claim by SYMBOL or test name, which TestPlanTablesNameEntitiesThatExist and
  TestPlanNamedTestsExist guard mechanically and which pass at HEAD, instead of by
  line number, which nothing guards. Measured at HEAD: play_loop.go 262-265, 174-198,
  178-181, 161 and replraw.go 264, 170, 140, 94, 534, 366-380 all moved or vanished;
  two of those rows are now false in substance, since the playback dance is deleted
  and viewportGesture is already shared with --play.

## Round 6 — 2026-08-31T18:35:17-07:00 (claude) — passed

### Disposed

- PQ-12 — not-addressed — Plan unchanged since round 5 (last edit 3e19b95, 18:23; round 5 at 18:28). All three premises re-measured true at HEAD; a fifth row, the KeyClick dispatch, is unassigned.
- PQ-13 — not-addressed — The current-truth tables still anchor by line number; unchanged since the round that raised it.

## Open findings

- **PQ-12** [Important] `mechanism-adopted-without-its-obligations` T4 and T5 specify region coordinates against premises the screen and Render do not hold
- **PQ-13** [Minor] `citation-does-not-point-at-the-claim` ten of eighteen line anchors in the plan's current-truth sections are wrong after 41 landed
