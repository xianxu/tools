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

## Open findings

- **PQ-1** [Critical] `mechanism-adopted-without-its-obligations` the reveal-playback restore/re-enter dance tears down the alternate screen and mouse reporting
- **PQ-2** [Important] `existing-seam-not-reused` writeRendered is already the region seam, and D2's claim that draw does not change is false
- **PQ-3** [Important] `mechanism-adopted-without-its-obligations` no task wires scroll, page or resize, which the alternate screen makes the loop's job
- **PQ-4** [Important] `two-callers-one-behavior` --play's y/n/Enter playback is playAnnounced, not replayInPlace, so the click becomes a second caller
- **PQ-5** [Minor] `entity-table-mislabels-purity` replayRegion is listed as a PURE entity but performs audio playback and writes to stdout/stderr
- **PQ-6** [Minor] `issue-plan-not-synced` the issue's ## Plan still says "decide Option A vs B / design it", which this plan file completes
- **PQ-7** [Minor] `milestone-tag-without-a-boundary` M1 is the only milestone and the plan says one sdlc close, so the Mx tag buys a redundant boundary
