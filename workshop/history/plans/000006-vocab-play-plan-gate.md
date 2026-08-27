---
gate: plan-quality
issue: 6
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-27T08:21:39-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: The play package's central interface names Key, which lives in package main and is unreachable from a sub-package
          detail: |-
            cmd/define/key.go:1 declares package main; Key/KeyKind are used by rawterm.go:48, editor.go:52,
            repl.go, replraw.go, complete.go and commandloop. The plan's Grade(k Key) and Apply(s Session, k Key)
            sit in cmd/define/play/, which cannot import main. Every resolution is a layer-boundary decision the
            plan must make explicitly: hoist Key/KeyKind/decodeKey to a shared package (touching six production
            files, their tests, and the new purity import allowlist), duplicate a key type in play with a shim
            in main, or keep Question/Session/Apply in package main and forfeit the purity-guard rationale that
            justifies the separate package. The #14 editor precedent does not settle it: that Apply is in main.
          family: cross-package-type-reachability
          round: 1
        - id: PQ-2
          severity: Critical
          title: DEFINE_NO_CAPTURE leaves deck nil, so the "still RUNS a session" Done-when cannot be satisfied as planned
          detail: |-
            cmd/define/main.go:187-192 — openStore under opt.noCapture returns storeDeps with history, capture,
            usage and clock but NO deck; withStore (main.go:130-133) leaves d.deck nil; command.go:164-166
            documents it as "the deck was never opened". runPlay needs deck.Deck() and deck.Events() to build a
            queue, so under the opt-out the session nil-panics or falls into the empty-queue branch — the opposite
            of the Done-when. The plan addresses only the write side via decideCapture. State the resolution:
            either split the documented noCapture policy into "write nothing, still read" (a change to a seam
            whose meaning is in --help and the README) or revise the Done-when row.
          family: unverified-wiring-assumption
          round: 1
        - id: PQ-3
          severity: Important
          title: CaptureReview(word, correct bool, opt) cannot represent Skipped, yet the plan tests the skip against it
          detail: |-
            Task 2 Step 1 says a skip emits a record Outcome carrying Skipped; Task 3 Step 1 tests "a SKIP records
            nothing at all" against CaptureReview, whose only verdict input is a bool. Decide which side filters:
            Apply emits no record outcome for a skip (test moves to session_test.go), or CaptureReview takes the
            Verdict — which introduces a main-to-play dependency edge the plan never draws.
          family: signature-cannot-express-contract
          round: 1
        - id: PQ-4
          severity: Important
          title: The daily budget passed to schedule.Queue is never specified — default, constant or flag
          detail: |-
            cmd/define/schedule/queue.go:29 requires budget; queue.go:31-35 returns nil for budget <= 0 and
            queue.go:78 notes "a budget is caller input" — this issue is that caller. Task 4 Step 8 wires --play
            without naming the value or whether it is configurable. A wrong default silently produces "nothing due
            today" forever, indistinguishable from the empty-queue success path the same plan builds.
          family: undefined-required-input
          round: 1
        - id: PQ-5
          severity: Important
          title: The Spec's bad-question keypress is absent from the plan and from any non-goals statement
          detail: |-
            The Spec requires "a bad-question keypress records a flag against the question". No Outcome, no
            Capturer verb, no Done-when row, no test. It is not a checkbox: store.EventKind has exactly three
            values (cmd/define/store/event.go:9-16) and none is a flag, so it is an append-only-log schema
            decision. Deferring it to #7 may be correct, but the plan has a Risks section and no non-goals
            section, so the deferral is invisible (ARCH-PURPOSE at-plan).
          family: scope-subset-of-spec
          round: 1
        - id: PQ-6
          severity: Important
          title: 'Task 2 Step 6 commits to copying ~150 lines of #5''s purity guards rather than extracting a shared helper'
          detail: |-
            cmd/define/schedule/purity_test.go is three test functions — go list invocation, source walk,
            comment-line skipping, three vacuity guards — that differ from play's only in import path and two
            allowlist maps. With #7/#12/#13 each adding a form package, copy two becomes copy four. Extract a
            parameterized guard taking (importPath, allowedImports, allowedStoreSymbols); the storetest.Suite
            precedent at cmd/define/store/storetest/suite.go is the shape to follow (ARCH-DRY).
          family: duplicated-guard-logic
          round: 1
        - id: PQ-7
          severity: Minor
          title: Audio-on-by-default has no Done-when row, no test step, and the runPlay harness omits the player fake
          detail: |-
            The Spec makes pronunciation playback default-on during review and the plan discusses it only in
            Risks. Task 4 Step 1's harness names a fake dictionary, store.Mem and FixedClock but no Player, while
            playAnnounced (cmd/define/main.go:567) shells out to afplay(1). fakePlayer already exists
            (cmd/define/player_fake_test.go, atlas/define.md:30) — name it in the harness and add an acceptance
            row for audio plus --no-audio (ARCH-MOCK).
          family: untested-spec-behavior
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-27T08:25:05-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Grade(r rune) resolved, but Apply(s Session, k Key) is verbatim unchanged and Task 2 Step 1's Ctrl-C row contradicts the stated rune+methods seam.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Verified openStore's opt-out branch returns no deck (main.go:180-186); plan and Done-when both now say print-one-line-and-exit-0.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Prose and Done-when fixed; Task 2 Step 1 ("skip emits a record outcome with Skipped") and Task 3 Step 1 (skip tested against CaptureReview) still carry the superseded text.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: -count, default 20, flag-configurable, with 0 deferring to Queue's nil contract rather than inventing a second meaning.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: 'Explicit deferral to #12/#13 with the reason stated in the plan and mirrored into the issue Log.'
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Step 6 now extracts a parameterised (importPath, allowedImports, allowedStoreSymbols) helper on the storetest.Suite precedent.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: fakePlayer named in the Task 4 Step 1 harness; Step 1b adds the play-before-reveal and --no-audio assertions.
          round: 2
      findings:
        - id: PQ-8
          severity: Minor
          title: The plan names withRawTerminal for reuse; no such symbol exists — the raw-mode API is enterRaw/rawSession.restore
          detail: |-
            This is the 2nd finding in family `unverified-wiring-assumption`. Do not
            just fix this instance. The rule: every EXISTING symbol a plan names for
            reuse must resolve to a file:line before the plan is executable, and the
            enumeration is the set of such names in the plan. I ran that enumeration —
            readKeys (rawterm.go:48), playAnnounced (main.go:567), interrupter
            (interrupt.go:20), fakePlayer (player_fake_test.go), store.NewMem
            (store/mem.go:24), store.FixedClock (store/clock.go:19), storetest.Suite,
            schedule.Queue (schedule/queue.go:29), store.EventReviewed
            (store/event.go:11), llm.ErrUnavailable (internal/llm/errors.go:19),
            deps.newLLM (ask.go:150) all resolve. Measured prevalence: 1 wrong out of
            12, so the family is not systemic and this is a naming slip, not a design
            gap. The actual raw-mode seam is enterRaw(*os.File) (*rawSession, error)
            plus (*rawSession).restore() at cmd/define/rawterm.go:21 and :29.
          family: unverified-wiring-assumption
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-27T08:28:00-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Question.Grade takes a rune, Apply takes play's own Input, main decodes Key at the boundary — key.go:29 verified.
          round: 3
        - id: PQ-3
          disposition: not-addressed
          note: 'The plan now states both filter locations: "the loop drops skips before recording" vs "Apply emits no record outcome for a skip".'
          round: 3
        - id: PQ-8
          disposition: not-addressed
          note: Plan line 60 still names withRawTerminal; the API is enterRaw / rawSession.restore at cmd/define/rawterm.go:21,29.
          round: 3
      findings:
        - id: PQ-9
          severity: Minor
          title: M2 adds --play and runPlay but schedules no atlas update, which its own closing gate refuses
          detail: |-
            Task 4 lists README (Step 10) then `sdlc close` (Step 11). The M2 window adds a
            user-visible flag and a new loop — new surface, so the close gate's atlas guard fires,
            and the plan itself records that #21 and #5 were both refused for exactly this. atlas/
            define.md already carries the Player seam table (line 30) and the render-cooked/play-raw
            narrative; add an M2 step extending it with the play loop, or state why --no-atlas applies.
          family: milestone-omits-gate-obligation
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-27T08:30:48-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: Design half decided (Apply filters); Task 2 Step 1 still instructs asserting the opposite.
          round: 4
        - id: PQ-8
          disposition: addressed
          note: enterRaw/(*rawSession).restore now cited at rawterm.go:21,29 — verified correct.
          round: 4
        - id: PQ-9
          disposition: addressed
          note: Task 4 Step 9b adds the M2 atlas extension before the close.
          round: 4
      findings:
        - id: PQ-10
          severity: Minor
          title: The no-deck exit is specified against DEFINE_NO_CAPTURE, but openStore returns a nil deck on two branches
          detail: |-
            main.go:188 (opt.noCapture) and main.go:194-201 (os.Getwd failure) both return a
            storeDeps with no deck. Keying runPlay's print-and-exit-0 on the env-var flag leaves
            the Getwd path handing a nil store.Store to the queue builder. Guard on deck == nil,
            which is the actual precondition; the flag is only one of its causes.
          family: guard-keyed-on-proxy-condition
          round: 4
      blocked: false
    - "n": 5
      timestamp: "2026-08-27T08:33:18-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: not-addressed
          note: |-
            Design section chose "Apply emits OutcomeNone for Skipped", but Task 2 Step 1 (plan:98) still says
            "a skip emits a record outcome with Skipped", and Task 3 (plan:122) still says "the loop drops it".
          round: 5
        - id: PQ-10
          disposition: addressed
          note: Guard is now deck == nil, with both openStore branches (main.go:186, 194-201) named as its causes.
          round: 5
      blocked: false
content_hash: 3582a3641ee036ed9720a512abc64758eea9d2e008d21f99bf94c856d499005e
---

# Gate ledger — tools#6 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T08:21:39-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `cross-package-type-reachability` The play package's central interface names Key, which lives in package main and is unreachable from a sub-package
  cmd/define/key.go:1 declares package main; Key/KeyKind are used by rawterm.go:48, editor.go:52,
  repl.go, replraw.go, complete.go and commandloop. The plan's Grade(k Key) and Apply(s Session, k Key)
  sit in cmd/define/play/, which cannot import main. Every resolution is a layer-boundary decision the
  plan must make explicitly: hoist Key/KeyKind/decodeKey to a shared package (touching six production
  files, their tests, and the new purity import allowlist), duplicate a key type in play with a shim
  in main, or keep Question/Session/Apply in package main and forfeit the purity-guard rationale that
  justifies the separate package. The #14 editor precedent does not settle it: that Apply is in main.
- **PQ-2** [Critical] `unverified-wiring-assumption` DEFINE_NO_CAPTURE leaves deck nil, so the "still RUNS a session" Done-when cannot be satisfied as planned
  cmd/define/main.go:187-192 — openStore under opt.noCapture returns storeDeps with history, capture,
  usage and clock but NO deck; withStore (main.go:130-133) leaves d.deck nil; command.go:164-166
  documents it as "the deck was never opened". runPlay needs deck.Deck() and deck.Events() to build a
  queue, so under the opt-out the session nil-panics or falls into the empty-queue branch — the opposite
  of the Done-when. The plan addresses only the write side via decideCapture. State the resolution:
  either split the documented noCapture policy into "write nothing, still read" (a change to a seam
  whose meaning is in --help and the README) or revise the Done-when row.
- **PQ-3** [Important] `signature-cannot-express-contract` CaptureReview(word, correct bool, opt) cannot represent Skipped, yet the plan tests the skip against it
  Task 2 Step 1 says a skip emits a record Outcome carrying Skipped; Task 3 Step 1 tests "a SKIP records
  nothing at all" against CaptureReview, whose only verdict input is a bool. Decide which side filters:
  Apply emits no record outcome for a skip (test moves to session_test.go), or CaptureReview takes the
  Verdict — which introduces a main-to-play dependency edge the plan never draws.
- **PQ-4** [Important] `undefined-required-input` The daily budget passed to schedule.Queue is never specified — default, constant or flag
  cmd/define/schedule/queue.go:29 requires budget; queue.go:31-35 returns nil for budget <= 0 and
  queue.go:78 notes "a budget is caller input" — this issue is that caller. Task 4 Step 8 wires --play
  without naming the value or whether it is configurable. A wrong default silently produces "nothing due
  today" forever, indistinguishable from the empty-queue success path the same plan builds.
- **PQ-5** [Important] `scope-subset-of-spec` The Spec's bad-question keypress is absent from the plan and from any non-goals statement
  The Spec requires "a bad-question keypress records a flag against the question". No Outcome, no
  Capturer verb, no Done-when row, no test. It is not a checkbox: store.EventKind has exactly three
  values (cmd/define/store/event.go:9-16) and none is a flag, so it is an append-only-log schema
  decision. Deferring it to #7 may be correct, but the plan has a Risks section and no non-goals
  section, so the deferral is invisible (ARCH-PURPOSE at-plan).
- **PQ-6** [Important] `duplicated-guard-logic` Task 2 Step 6 commits to copying ~150 lines of #5's purity guards rather than extracting a shared helper
  cmd/define/schedule/purity_test.go is three test functions — go list invocation, source walk,
  comment-line skipping, three vacuity guards — that differ from play's only in import path and two
  allowlist maps. With #7/#12/#13 each adding a form package, copy two becomes copy four. Extract a
  parameterized guard taking (importPath, allowedImports, allowedStoreSymbols); the storetest.Suite
  precedent at cmd/define/store/storetest/suite.go is the shape to follow (ARCH-DRY).
- **PQ-7** [Minor] `untested-spec-behavior` Audio-on-by-default has no Done-when row, no test step, and the runPlay harness omits the player fake
  The Spec makes pronunciation playback default-on during review and the plan discusses it only in
  Risks. Task 4 Step 1's harness names a fake dictionary, store.Mem and FixedClock but no Player, while
  playAnnounced (cmd/define/main.go:567) shells out to afplay(1). fakePlayer already exists
  (cmd/define/player_fake_test.go, atlas/define.md:30) — name it in the harness and add an acceptance
  row for audio plus --no-audio (ARCH-MOCK).

## Round 2 — 2026-08-27T08:25:05-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Grade(r rune) resolved, but Apply(s Session, k Key) is verbatim unchanged and Task 2 Step 1's Ctrl-C row contradicts the stated rune+methods seam.
- PQ-2 — addressed — Verified openStore's opt-out branch returns no deck (main.go:180-186); plan and Done-when both now say print-one-line-and-exit-0.
- PQ-3 — not-addressed — Prose and Done-when fixed; Task 2 Step 1 ("skip emits a record outcome with Skipped") and Task 3 Step 1 (skip tested against CaptureReview) still carry the superseded text.
- PQ-4 — addressed — -count, default 20, flag-configurable, with 0 deferring to Queue's nil contract rather than inventing a second meaning.
- PQ-5 — addressed — Explicit deferral to #12/#13 with the reason stated in the plan and mirrored into the issue Log.
- PQ-6 — addressed — Step 6 now extracts a parameterised (importPath, allowedImports, allowedStoreSymbols) helper on the storetest.Suite precedent.
- PQ-7 — addressed — fakePlayer named in the Task 4 Step 1 harness; Step 1b adds the play-before-reveal and --no-audio assertions.

### Raised

- **PQ-8** [Minor] `unverified-wiring-assumption` The plan names withRawTerminal for reuse; no such symbol exists — the raw-mode API is enterRaw/rawSession.restore
  This is the 2nd finding in family `unverified-wiring-assumption`. Do not
  just fix this instance. The rule: every EXISTING symbol a plan names for
  reuse must resolve to a file:line before the plan is executable, and the
  enumeration is the set of such names in the plan. I ran that enumeration —
  readKeys (rawterm.go:48), playAnnounced (main.go:567), interrupter
  (interrupt.go:20), fakePlayer (player_fake_test.go), store.NewMem
  (store/mem.go:24), store.FixedClock (store/clock.go:19), storetest.Suite,
  schedule.Queue (schedule/queue.go:29), store.EventReviewed
  (store/event.go:11), llm.ErrUnavailable (internal/llm/errors.go:19),
  deps.newLLM (ask.go:150) all resolve. Measured prevalence: 1 wrong out of
  12, so the family is not systemic and this is a naming slip, not a design
  gap. The actual raw-mode seam is enterRaw(*os.File) (*rawSession, error)
  plus (*rawSession).restore() at cmd/define/rawterm.go:21 and :29.

## Round 3 — 2026-08-27T08:28:00-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Question.Grade takes a rune, Apply takes play's own Input, main decodes Key at the boundary — key.go:29 verified.
- PQ-3 — not-addressed — The plan now states both filter locations: "the loop drops skips before recording" vs "Apply emits no record outcome for a skip".
- PQ-8 — not-addressed — Plan line 60 still names withRawTerminal; the API is enterRaw / rawSession.restore at cmd/define/rawterm.go:21,29.

### Raised

- **PQ-9** [Minor] `milestone-omits-gate-obligation` M2 adds --play and runPlay but schedules no atlas update, which its own closing gate refuses
  Task 4 lists README (Step 10) then `sdlc close` (Step 11). The M2 window adds a
  user-visible flag and a new loop — new surface, so the close gate's atlas guard fires,
  and the plan itself records that #21 and #5 were both refused for exactly this. atlas/
  define.md already carries the Player seam table (line 30) and the render-cooked/play-raw
  narrative; add an M2 step extending it with the play loop, or state why --no-atlas applies.

## Round 4 — 2026-08-27T08:30:48-07:00 (claude) — passed

### Disposed

- PQ-3 — not-addressed — Design half decided (Apply filters); Task 2 Step 1 still instructs asserting the opposite.
- PQ-8 — addressed — enterRaw/(*rawSession).restore now cited at rawterm.go:21,29 — verified correct.
- PQ-9 — addressed — Task 4 Step 9b adds the M2 atlas extension before the close.

### Raised

- **PQ-10** [Minor] `guard-keyed-on-proxy-condition` The no-deck exit is specified against DEFINE_NO_CAPTURE, but openStore returns a nil deck on two branches
  main.go:188 (opt.noCapture) and main.go:194-201 (os.Getwd failure) both return a
  storeDeps with no deck. Keying runPlay's print-and-exit-0 on the env-var flag leaves
  the Getwd path handing a nil store.Store to the queue builder. Guard on deck == nil,
  which is the actual precondition; the flag is only one of its causes.

## Round 5 — 2026-08-27T08:33:18-07:00 (claude) — passed

### Disposed

- PQ-3 — not-addressed — Design section chose "Apply emits OutcomeNone for Skipped", but Task 2 Step 1 (plan:98) still says
"a skip emits a record outcome with Skipped", and Task 3 (plan:122) still says "the loop drops it".
- PQ-10 — addressed — Guard is now deck == nil, with both openStore branches (main.go:186, 194-201) named as its causes.

## Open findings

- **PQ-3** [Important] `signature-cannot-express-contract` CaptureReview(word, correct bool, opt) cannot represent Skipped, yet the plan tests the skip against it
