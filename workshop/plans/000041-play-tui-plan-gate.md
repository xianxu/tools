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

## Open findings

- **PQ-1** [Critical] `terminal-state-reentry` D1/D5 adopt the alt screen but the reveal's restore/enterRaw dance drops it and never re-enters
- **PQ-2** [Important] `cost-basis-unverified` ARCH-CONSTRAINTS states the bar's cost as "two disk reads, microseconds"; it is O(deck files + days of log) per answer
- **PQ-3** [Important] `viewport-gesture-layering` D6 routes paging through toInput, which would teach the pure play package about a viewport
- **PQ-4** [Minor] `existing-assertions-unaccounted` T9 extends only #7's pty test; three other --play pty tests scrape the output shape this issue changes
- **PQ-5** [Minor] `touched-site-unlisted` Done-when 8 requires finish() to share sittingBar's formatter, but finish appears in no table as modified
