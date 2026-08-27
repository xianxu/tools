---
gate: plan-quality
issue: 24
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-27T15:43:38-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: the new miss-before-reveal branch never increments s.Wrong, so the session tally reports 0 wrong
          detail: |-
            Task 2 Step 3's snippet returns the record+reveal outcomes directly instead
            of going through advance, and advance is where s.Wrong++ lives
            (cmd/define/play/session.go:158-163). The follow-up key then calls
            advance(s, q, Skipped), which increments nothing. finish
            (cmd/define/play_loop.go:261) prints "N right, 0 wrong" for a session of
            misses. No planned or existing test covers it — session_test.go:118 checks
            the tally only on the reveal-then-grade path.
          family: branch-bypasses-shared-bookkeeping
          round: 1
        - id: PQ-2
          severity: Critical
          title: Enter and space are dead in the new Graded state, while the prompt promises "any key = next word"
          detail: |-
            toInput (cmd/define/play_loop.go:174-190) maps Enter and space to
            InputReveal before they reach the InputRune arm, and the InputReveal arm
            returns OutcomeNone when s.Revealed is set (session.go:128-133). After a
            miss both Revealed and Graded are true, so the two keys every existing user
            has muscle memory for do nothing, contradicting Task 7's own prompt copy.
            The plan must handle InputReveal in the Graded state and the pty test must
            press the move-on key, or the defect ships exactly the way BR-45's did.
          family: new-state-unhandled-input-kinds
          round: 1
        - id: PQ-3
          severity: Important
          title: the plan names tests that do not break and omits the one that does
          detail: |-
            Task 5 Step 2 claims TestUngradedKeyNeverReachesTheCapturer
            (play_loop_test.go:203, scripts "\rz^") and "any test pressing Enter before
            y" assert the old flow; all of these stay green. The test that actually
            encodes the reversed premise is TestGradingBeforeRevealIsIgnored at
            cmd/define/play/session_test.go:61 with its comment at :59, and no task
            mentions it — nor does Task 9's grep pattern reach it or atlas
            define.md:1284. Task 1's "seven call sites" is nine.
          family: unbacked-claim-about-existing-code
          round: 1
        - id: PQ-4
          severity: Important
          title: Task 4's "after a miss" subtest uses 'n' against fakeForm, which grades only digits
          detail: |-
            fakeForm.Grade (cmd/define/play/session_test.go:212-222) returns
            (Skipped, false) for 'n', so the precondition yields OutcomeNone, Graded is
            never set, and the subtest silently re-runs the "before anything" case
            under a name claiming to cover the new state. Use '2'.
          family: test-precondition-never-fires
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-27T15:47:59-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: score split from advance; miss branch scores without advancing, pinned by next.Wrong == 1.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: InputReveal now has a Graded arm ahead of the Revealed short-circuit; pty test presses space.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: TestGradingBeforeRevealIsIgnored named and deleted at session_test.go:61; grep reaches atlas:1284; nine call sites confirmed.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: The "after a miss" subtest drives twoQuestions()/Recall, whose Grade('n') returns (Wrong, true).
          round: 2
      findings:
        - id: PQ-5
          severity: Minor
          title: Task 9's grep cannot reach README:41-42, where the stale phrase is split across a line break
          detail: |-
            2nd in this family (2 of 2 rounds). The rule, not the site: prove a doc-sweep
            grep hits every already-known stale site before trusting it as the authority
            for the site list — line-based patterns miss line-wrapped prose. Here
            "Enter or / space reveals the definition" at README.md:41-42 matches none of
            the four patterns; Task 9's Files list naming README.md:41-58 covers it anyway.
          family: unbacked-claim-about-existing-code
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-08-27T15:52:55-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: addressed
          note: Grep is now single-word plus a proof sub-grep; I ran it and README.md:42 and atlas/define.md:1284 both hit.
          round: 3
      findings:
        - id: PQ-6
          severity: Important
          title: Task 6's no-audio assertion cannot fail — playRig installs noAudioSource, so d.player is never reached
          detail: |-
            playRig sets d.audio = noAudioSource{} (play_loop_test.go:33) and Fetch always
            returns ErrNoAudio (main_test.go:19), while speak fetches before playing
            (main.go:636). fp.Played is empty under old and new flow alike, so Step 3's
            mutant survives and Done-when bullet 1 proves nothing. Set d.audio = okAudio{}
            (play_loop_test.go:474, which exists for exactly this — ARCH-DRY/ARCH-MOCK).
            2nd in family. RULE, not the site: a negative assertion is meaningful only if
            the rig can produce the thing being denied; name the double that produces it
            and confirm the rig installs it. Enumeration run over the plan's 8 new tests —
            1 of 8 fails. Put that enumeration in the plan and extend Step 3's bite-check
            from one test to every negative assertion.
          family: test-precondition-never-fires
          round: 3
        - id: PQ-7
          severity: Minor
          title: The issue's Plan row says seven Apply call sites in session_test.go; grep counts nine
          detail: |-
            grep -c "Apply(" cmd/define/play/session_test.go returns 9, matching the plan
            doc and the Estimate; only the issue checklist row says seven. 3rd in family.
            RULE, not the site: a count or line-range in a plan must be produced by a
            command the plan shows, not typed from memory — the same discipline Task 9's
            grep-proof already applies to the doc sweep, extended to counts.
          family: unbacked-claim-about-existing-code
          round: 3
      blocked: true
content_hash: 5c05864bf55827ee8c2ef3913dd0ecbe5bd997b78de9b268ca725c253ed8d1de
---

# Gate ledger — tools#24 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T15:43:38-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `branch-bypasses-shared-bookkeeping` the new miss-before-reveal branch never increments s.Wrong, so the session tally reports 0 wrong
  Task 2 Step 3's snippet returns the record+reveal outcomes directly instead
  of going through advance, and advance is where s.Wrong++ lives
  (cmd/define/play/session.go:158-163). The follow-up key then calls
  advance(s, q, Skipped), which increments nothing. finish
  (cmd/define/play_loop.go:261) prints "N right, 0 wrong" for a session of
  misses. No planned or existing test covers it — session_test.go:118 checks
  the tally only on the reveal-then-grade path.
- **PQ-2** [Critical] `new-state-unhandled-input-kinds` Enter and space are dead in the new Graded state, while the prompt promises "any key = next word"
  toInput (cmd/define/play_loop.go:174-190) maps Enter and space to
  InputReveal before they reach the InputRune arm, and the InputReveal arm
  returns OutcomeNone when s.Revealed is set (session.go:128-133). After a
  miss both Revealed and Graded are true, so the two keys every existing user
  has muscle memory for do nothing, contradicting Task 7's own prompt copy.
  The plan must handle InputReveal in the Graded state and the pty test must
  press the move-on key, or the defect ships exactly the way BR-45's did.
- **PQ-3** [Important] `unbacked-claim-about-existing-code` the plan names tests that do not break and omits the one that does
  Task 5 Step 2 claims TestUngradedKeyNeverReachesTheCapturer
  (play_loop_test.go:203, scripts "\rz^") and "any test pressing Enter before
  y" assert the old flow; all of these stay green. The test that actually
  encodes the reversed premise is TestGradingBeforeRevealIsIgnored at
  cmd/define/play/session_test.go:61 with its comment at :59, and no task
  mentions it — nor does Task 9's grep pattern reach it or atlas
  define.md:1284. Task 1's "seven call sites" is nine.
- **PQ-4** [Important] `test-precondition-never-fires` Task 4's "after a miss" subtest uses 'n' against fakeForm, which grades only digits
  fakeForm.Grade (cmd/define/play/session_test.go:212-222) returns
  (Skipped, false) for 'n', so the precondition yields OutcomeNone, Graded is
  never set, and the subtest silently re-runs the "before anything" case
  under a name claiming to cover the new state. Use '2'.

## Round 2 — 2026-08-27T15:47:59-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — score split from advance; miss branch scores without advancing, pinned by next.Wrong == 1.
- PQ-2 — addressed — InputReveal now has a Graded arm ahead of the Revealed short-circuit; pty test presses space.
- PQ-3 — addressed — TestGradingBeforeRevealIsIgnored named and deleted at session_test.go:61; grep reaches atlas:1284; nine call sites confirmed.
- PQ-4 — addressed — The "after a miss" subtest drives twoQuestions()/Recall, whose Grade('n') returns (Wrong, true).

### Raised

- **PQ-5** [Minor] `unbacked-claim-about-existing-code` Task 9's grep cannot reach README:41-42, where the stale phrase is split across a line break
  2nd in this family (2 of 2 rounds). The rule, not the site: prove a doc-sweep
  grep hits every already-known stale site before trusting it as the authority
  for the site list — line-based patterns miss line-wrapped prose. Here
  "Enter or / space reveals the definition" at README.md:41-42 matches none of
  the four patterns; Task 9's Files list naming README.md:41-58 covers it anyway.

## Round 3 — 2026-08-27T15:52:55-07:00 (claude) — BLOCKED

### Disposed

- PQ-5 — addressed — Grep is now single-word plus a proof sub-grep; I ran it and README.md:42 and atlas/define.md:1284 both hit.

### Raised

- **PQ-6** [Important] `test-precondition-never-fires` Task 6's no-audio assertion cannot fail — playRig installs noAudioSource, so d.player is never reached
  playRig sets d.audio = noAudioSource{} (play_loop_test.go:33) and Fetch always
  returns ErrNoAudio (main_test.go:19), while speak fetches before playing
  (main.go:636). fp.Played is empty under old and new flow alike, so Step 3's
  mutant survives and Done-when bullet 1 proves nothing. Set d.audio = okAudio{}
  (play_loop_test.go:474, which exists for exactly this — ARCH-DRY/ARCH-MOCK).
  2nd in family. RULE, not the site: a negative assertion is meaningful only if
  the rig can produce the thing being denied; name the double that produces it
  and confirm the rig installs it. Enumeration run over the plan's 8 new tests —
  1 of 8 fails. Put that enumeration in the plan and extend Step 3's bite-check
  from one test to every negative assertion.
- **PQ-7** [Minor] `unbacked-claim-about-existing-code` The issue's Plan row says seven Apply call sites in session_test.go; grep counts nine
  grep -c "Apply(" cmd/define/play/session_test.go returns 9, matching the plan
  doc and the Estimate; only the issue checklist row says seven. 3rd in family.
  RULE, not the site: a count or line-range in a plan must be produced by a
  command the plan shows, not typed from memory — the same discipline Task 9's
  grep-proof already applies to the doc sweep, extended to counts.

## Open findings

- **PQ-6** [Important] `test-precondition-never-fires` Task 6's no-audio assertion cannot fail — playRig installs noAudioSource, so d.player is never reached
- **PQ-7** [Minor] `unbacked-claim-about-existing-code` The issue's Plan row says seven Apply call sites in session_test.go; grep counts nine
