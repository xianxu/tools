---
gate: plan-quality
issue: 41
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-31T13:35:22-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: D1/D5 adopt the alt screen but the reveal's restore/enterRaw dance drops it and never re-enters
          detail: |-
            play_loop.go:163-198 calls raw.sess.restore() on every OutcomeReveal to play cooked, then
            *raw.sess = *again with a fresh rawSession whose alt/mouse flags are false (rawterm.go:47);
            restore() already did leaveAlt/leaveMouse (rawterm.go:50-66). From the second reveal on, frames
            paint onto the NORMAL buffer and the wheel reverts to arrow keys. D5 cites #30 D4, whose premise
            (no mid-session restore) does not hold here. Every existing --play pty test passes --no-audio,
            so nothing in the tree or the Done-when table would catch it. Decide explicitly: delete the
            dance for --play, or re-enter alt+mouse after playback; pin it with an audio-enabled pty row.
          family: terminal-state-reentry
          round: 1
        - id: PQ-2
          severity: Important
          title: ARCH-CONSTRAINTS states the bar's cost as "two disk reads, microseconds"; it is O(deck files + days of log) per answer
          detail: |-
            store/yaml.go:208 reads one file and parses one YAML per deck word; store/yaml.go:286 reads one
            file per day of history, parses every event ever, and sorts. That runs synchronously between the
            grading keystroke and the next frame. todaysQuestions (play_loop.go:237-249) ALREADY holds deck
            and the folded progress and discards them — carry them forward and apply a pure delta per answer
            (ARCH-PURE, ARCH-DRY), so the bar costs no IO and D7's cache is unnecessary.
          family: cost-basis-unverified
          round: 1
        - id: PQ-3
          severity: Important
          title: D6 routes paging through toInput, which would teach the pure play package about a viewport
          detail: |-
            toInput returns play.Input (play_loop.go:216). The editor deliberately dispatches viewport
            gestures in the loop BEFORE Apply, documented at replraw.go:361-377 ("a viewport gesture never
            reaches Apply"). Handle them ahead of toInput and extract the four-case switch plus wheelLines
            into one helper both loops call, rather than a second copy of the policy. Also note D6's
            "nothing else moves" omits enterMouse, whose cost is that drag-select needs Option/Shift
            (rawterm.go:175-190, /help at command.go:267) — a review sitting now inherits that.
          family: viewport-gesture-layering
          round: 1
        - id: PQ-4
          severity: Minor
          title: 'T9 extends only #7''s pty test; three other --play pty tests scrape the output shape this issue changes'
          detail: |-
            TestPTYPlayRendersEveryLineAtColumnZero (pty_conformance_test.go:367),
            TestPTYPlayCorrectAnswerNeverRevealsIt (:467) and TestPTYPlayGradeFirst (:490) assert over
            --play's raw byte stream, which becomes cursorHome+eraseDown whole frames. Say what happens
            to each — especially the column-zero check, which is --play's shipped-defect net.
          family: existing-assertions-unaccounted
          round: 1
        - id: PQ-5
          severity: Minor
          title: Done-when 8 requires finish() to share sittingBar's formatter, but finish appears in no table as modified
          detail: |-
            finish (play_loop.go:381) owns the current wording of the load line and the -count assumption.
            List it in the integration-points table as modified so the DRY claim has an owner.
          family: touched-site-unlisted
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-31T13:40:54-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: D5a deletes the dance; premises verified (screen.go:64-78 honours eraseLine, replraw.go:517 already plays in raw mode).
          round: 2
        - id: PQ-2
          disposition: addressed
          note: D7 carries deck+prog forward and applies schedule.Answer in memory; T6 pins it with a counting store.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Layering half fixed; the shared page/scroll helper (replraw.go:366-376 + wheelLines) and the enterMouse drag-select cost are still absent.
          round: 2
        - id: PQ-4
          disposition: addressed
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
      findings:
        - id: PQ-6
          severity: Important
          title: D3 claims the bar pins to the bottom for free, but Paint never pads a short buffer
          detail: |-
            screen.go:394-396 budgets s.rows, and visible() (screen.go:211-221) returns s.lines
            unpadded when the buffer is shorter — so early in a sitting the footer floats under
            the last question rather than on the terminal's last row. Padding Paint changes the
            EDITOR's frames too (a shared seam), so the plan must state which way it goes, and
            Done-when 3 needs a predicate over row position, since "a state forgets to pass the
            footer" passes either way (ARCH-PURPOSE).
          family: inherited-mechanism-unverified
          round: 2
        - id: PQ-7
          severity: Minor
          title: Round 1's reversals were folded into the Decisions but left standing in the tables
          detail: |-
            Two enumerable instances: the verified-claims row still says DailyLoad is "two disk
            reads, which is why D7 caches" (the exact basis PQ-2 corrected, and D7 no longer
            caches), and the integration-points row still says toInput "gains the paging keys
            (D6)" against a D6 that says it is untouched. The rule, not the two edits: a
            Revision that reverses a decision re-reads every table row and task that cited it.
          family: reversed-decision-not-swept
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-31T13:44:55-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: Layering fixed; the shared gesture helper (replraw.go:359-385 + wheelLines at :80) and enterMouse — without which KeyWheel* is never produced (key.go:318-320) — are both still absent.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: D3a names newPinnedScreen and Done-when 11/12 give row-position and no-REPL-regression predicates.
          round: 3
        - id: PQ-7
          disposition: addressed
          note: Both rows corrected and the sweep rule stated in Revision round 2.
          round: 3
      findings:
        - id: PQ-8
          severity: Important
          title: D7's "one deck read and one log read" per sitting is contradicted by finish(), and T6's test fails against it
          detail: |-
            This is the 2nd finding in family cost-basis-unverified, so the deliverable is the rule, not the
            sentence: every IO claim names the call sites that pay it and is checked against all of them,
            including sites the plan does not modify. finish() reads Deck() and Events(anyTime) again at
            play_loop.go:384-388, documented at :372-375 as deliberate correctness, so the sitting pays two of
            each and T6's "reads the deck ONCE, not N times" fails as written. Write the enumeration
            (todaysQuestions :238/:242, finish :384/:388, the new per-answer walk) and decide there whether
            finish consumes D7's in-memory prog, since D8 already makes it share the formatter.
          family: cost-basis-unverified
          round: 3
        - id: PQ-9
          severity: Minor
          title: Done-when rows 6 and 7 still encode the per-answer-read model D7 reversed
          detail: |-
            This is the 2nd finding in family reversed-decision-not-swept. The rule is already written in
            Revision round 2; what is missing is executing the enumeration it implies across all twelve
            Done-when rows and nine tasks. Measured prevalence in this instance: row 6's bound of N+1 reads is
            satisfied by the design D7 rejected and disagrees with T6's "ONCE", and row 7's "keeps the previous
            figures" presupposes a repeated read that D7 removed, where play_loop.go:246 already degrades a
            failed log read to empty progress.
          family: reversed-decision-not-swept
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-31T13:49:01-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: Routing before toInput is fixed; the shared helper and enterMouse are still unstated.
          round: 4
        - id: PQ-8
          disposition: addressed
          note: Four-site enumeration is complete — I confirmed no other Deck()/Events() caller in the play path.
          round: 4
        - id: PQ-9
          disposition: addressed
          note: Rows 6 and 7 now match D7, and row 7's claim verifies against play_loop.go:242-246.
          round: 4
      findings:
        - id: PQ-10
          severity: Minor
          title: Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
          detail: |-
            The Done-when preamble requires every pin be a named test or a grep for a property, never
            "file X is unchanged", and row 7 is the one row that breaks its own rule. Grepping the
            message at play_loop.go:246 finds no test asserting the degraded path. T6 changes
            todaysQuestions' signature so that path must now hand the loop a usable empty progress
            rather than merely continuing locally, which is precisely when an uncovered branch is worth
            a test.
          family: pin-without-predicate
          round: 4
      blocked: false
content_hash: 1af18d02b97caa3d67fbd669db50ba4df96d31726d1cce55bef6f30b85bbd645
---

# Gate ledger — tools#41 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T13:35:22-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `terminal-state-reentry` D1/D5 adopt the alt screen but the reveal's restore/enterRaw dance drops it and never re-enters
  play_loop.go:163-198 calls raw.sess.restore() on every OutcomeReveal to play cooked, then
  *raw.sess = *again with a fresh rawSession whose alt/mouse flags are false (rawterm.go:47);
  restore() already did leaveAlt/leaveMouse (rawterm.go:50-66). From the second reveal on, frames
  paint onto the NORMAL buffer and the wheel reverts to arrow keys. D5 cites #30 D4, whose premise
  (no mid-session restore) does not hold here. Every existing --play pty test passes --no-audio,
  so nothing in the tree or the Done-when table would catch it. Decide explicitly: delete the
  dance for --play, or re-enter alt+mouse after playback; pin it with an audio-enabled pty row.
- **PQ-2** [Important] `cost-basis-unverified` ARCH-CONSTRAINTS states the bar's cost as "two disk reads, microseconds"; it is O(deck files + days of log) per answer
  store/yaml.go:208 reads one file and parses one YAML per deck word; store/yaml.go:286 reads one
  file per day of history, parses every event ever, and sorts. That runs synchronously between the
  grading keystroke and the next frame. todaysQuestions (play_loop.go:237-249) ALREADY holds deck
  and the folded progress and discards them — carry them forward and apply a pure delta per answer
  (ARCH-PURE, ARCH-DRY), so the bar costs no IO and D7's cache is unnecessary.
- **PQ-3** [Important] `viewport-gesture-layering` D6 routes paging through toInput, which would teach the pure play package about a viewport
  toInput returns play.Input (play_loop.go:216). The editor deliberately dispatches viewport
  gestures in the loop BEFORE Apply, documented at replraw.go:361-377 ("a viewport gesture never
  reaches Apply"). Handle them ahead of toInput and extract the four-case switch plus wheelLines
  into one helper both loops call, rather than a second copy of the policy. Also note D6's
  "nothing else moves" omits enterMouse, whose cost is that drag-select needs Option/Shift
  (rawterm.go:175-190, /help at command.go:267) — a review sitting now inherits that.
- **PQ-4** [Minor] `existing-assertions-unaccounted` T9 extends only #7's pty test; three other --play pty tests scrape the output shape this issue changes
  TestPTYPlayRendersEveryLineAtColumnZero (pty_conformance_test.go:367),
  TestPTYPlayCorrectAnswerNeverRevealsIt (:467) and TestPTYPlayGradeFirst (:490) assert over
  --play's raw byte stream, which becomes cursorHome+eraseDown whole frames. Say what happens
  to each — especially the column-zero check, which is --play's shipped-defect net.
- **PQ-5** [Minor] `touched-site-unlisted` Done-when 8 requires finish() to share sittingBar's formatter, but finish appears in no table as modified
  finish (play_loop.go:381) owns the current wording of the load line and the -count assumption.
  List it in the integration-points table as modified so the DRY claim has an owner.

## Round 2 — 2026-08-31T13:40:54-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — D5a deletes the dance; premises verified (screen.go:64-78 honours eraseLine, replraw.go:517 already plays in raw mode).
- PQ-2 — addressed — D7 carries deck+prog forward and applies schedule.Answer in memory; T6 pins it with a counting store.
- PQ-3 — not-addressed — Layering half fixed; the shared page/scroll helper (replraw.go:366-376 + wheelLines) and the enterMouse drag-select cost are still absent.
- PQ-4 — addressed
- PQ-5 — addressed

### Raised

- **PQ-6** [Important] `inherited-mechanism-unverified` D3 claims the bar pins to the bottom for free, but Paint never pads a short buffer
  screen.go:394-396 budgets s.rows, and visible() (screen.go:211-221) returns s.lines
  unpadded when the buffer is shorter — so early in a sitting the footer floats under
  the last question rather than on the terminal's last row. Padding Paint changes the
  EDITOR's frames too (a shared seam), so the plan must state which way it goes, and
  Done-when 3 needs a predicate over row position, since "a state forgets to pass the
  footer" passes either way (ARCH-PURPOSE).
- **PQ-7** [Minor] `reversed-decision-not-swept` Round 1's reversals were folded into the Decisions but left standing in the tables
  Two enumerable instances: the verified-claims row still says DailyLoad is "two disk
  reads, which is why D7 caches" (the exact basis PQ-2 corrected, and D7 no longer
  caches), and the integration-points row still says toInput "gains the paging keys
  (D6)" against a D6 that says it is untouched. The rule, not the two edits: a
  Revision that reverses a decision re-reads every table row and task that cited it.

## Round 3 — 2026-08-31T13:44:55-07:00 (claude) — BLOCKED

### Disposed

- PQ-3 — not-addressed — Layering fixed; the shared gesture helper (replraw.go:359-385 + wheelLines at :80) and enterMouse — without which KeyWheel* is never produced (key.go:318-320) — are both still absent.
- PQ-6 — addressed — D3a names newPinnedScreen and Done-when 11/12 give row-position and no-REPL-regression predicates.
- PQ-7 — addressed — Both rows corrected and the sweep rule stated in Revision round 2.

### Raised

- **PQ-8** [Important] `cost-basis-unverified` D7's "one deck read and one log read" per sitting is contradicted by finish(), and T6's test fails against it
  This is the 2nd finding in family cost-basis-unverified, so the deliverable is the rule, not the
  sentence: every IO claim names the call sites that pay it and is checked against all of them,
  including sites the plan does not modify. finish() reads Deck() and Events(anyTime) again at
  play_loop.go:384-388, documented at :372-375 as deliberate correctness, so the sitting pays two of
  each and T6's "reads the deck ONCE, not N times" fails as written. Write the enumeration
  (todaysQuestions :238/:242, finish :384/:388, the new per-answer walk) and decide there whether
  finish consumes D7's in-memory prog, since D8 already makes it share the formatter.
- **PQ-9** [Minor] `reversed-decision-not-swept` Done-when rows 6 and 7 still encode the per-answer-read model D7 reversed
  This is the 2nd finding in family reversed-decision-not-swept. The rule is already written in
  Revision round 2; what is missing is executing the enumeration it implies across all twelve
  Done-when rows and nine tasks. Measured prevalence in this instance: row 6's bound of N+1 reads is
  satisfied by the design D7 rejected and disagrees with T6's "ONCE", and row 7's "keeps the previous
  figures" presupposes a repeated read that D7 removed, where play_loop.go:246 already degrades a
  failed log read to empty progress.

## Round 4 — 2026-08-31T13:49:01-07:00 (claude) — passed

### Disposed

- PQ-3 — not-addressed — Routing before toInput is fixed; the shared helper and enterMouse are still unstated.
- PQ-8 — addressed — Four-site enumeration is complete — I confirmed no other Deck()/Events() caller in the play path.
- PQ-9 — addressed — Rows 6 and 7 now match D7, and row 7's claim verifies against play_loop.go:242-246.

### Raised

- **PQ-10** [Minor] `pin-without-predicate` Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
  The Done-when preamble requires every pin be a named test or a grep for a property, never
  "file X is unchanged", and row 7 is the one row that breaks its own rule. Grepping the
  message at play_loop.go:246 finds no test asserting the degraded path. T6 changes
  todaysQuestions' signature so that path must now hand the loop a usable empty progress
  rather than merely continuing locally, which is precisely when an uncovered branch is worth
  a test.

## Open findings

- **PQ-3** [Important] `viewport-gesture-layering` D6 routes paging through toInput, which would teach the pure play package about a viewport
- **PQ-10** [Minor] `pin-without-predicate` Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
