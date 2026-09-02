---
gate: plan-quality
issue: 44
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-02T13:01:16-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: the sitting's reveal playback keeps defaultIndicator, so the buffer still grows a blank line per question
          detail: |-
            play_loop.go:507-508 passes defaultIndicator(opt) into the same screen-backed
            stdout as the click path the plan fixes, so `before: "\n"` still commits one
            blank buffer line per audio reveal — the default path. The plan's claim that
            there are "two spellings of the indicator inside a screen" is wrong: there are
            five screen-hosted sites. Enumerate them in the plan and sweep them this round.
          family: sweep-every-site-of-the-rule
          round: 1
        - id: PQ-2
          severity: Important
          title: the editor's two other screen-hosted indicators are neither swept nor declared a non-goal
          detail: |-
            replraw.go:651-652 passes `before: "\r\n"`, which screen.Write normalizes to
            "\n" (screen.go:93) and therefore commits a blank line per audio lookup — the
            same live bug, in a file the plan already edits. replraw.go:633-634 is a fourth
            copy of the literal screenIndicator() exists to own. Either adopt the
            constructor at both, or record an explicit non-goal with the reason.
          family: sweep-every-site-of-the-rule
          round: 1
        - id: PQ-3
          severity: Important
          title: chromeGap is charged to the budget but emitted unconditionally, so a short terminal scrolls
          detail: |-
            Step 4 subtracts s.gap from s.rows, but the existing clamp at screen.go:468
            absorbs the shortfall while the gap rows are still written. At termRows == promptRows
            (a 76-column keys row on a narrow window) the frame becomes termRows+1 rows and the
            terminal scrolls, which does not happen today. Paint documents an order of sacrifice;
            say where the gap sits in it and pin the boundary height.
          family: frame-budget-completeness
          round: 1
        - id: PQ-4
          severity: Important
          title: Verification runs go test ./... which cannot compile the pty conformance suite that checks footerTop
          detail: |-
            pty_conformance_test.go is `//go:build darwin && conformance`, and its SGR-1006 click
            is the only end-to-end proof that a real terminal's click maps to the intended cell
            after footerTop moves. The plan calls that arithmetic the one that must not be wrong.
            Add `go test -tags conformance ./cmd/define` to the Verification section.
          family: verification-covers-the-declared-risk
          round: 1
        - id: PQ-5
          severity: Minor
          title: fitsABoard hardcodes chromeGap while the screen carries it as the field s.gap
          detail: |-
            Two owners of one quantity, in the function whose own comment says it exists because
            "a constant here was a second owner of a height displayRows already computes". Safe
            today because newPinnedScreen has one production caller (play_loop.go:106); the plan
            should state that rather than leave it implied.
          family: one-owner-per-quantity
          round: 1
        - id: PQ-6
          severity: Minor
          title: Task 3 leaves the non-board bar at play_loop.go:247 unstyled and changes boardFooter's signature silently
          detail: |-
            "the two view.Draw calls and sittingBar's result in boardFooter" covers the board's bar
            but not the inline `[]string{sittingBar(fig)}` at play_loop.go:247, which is the common
            sitting. Threading a palette into boardFooter also alters a function whose comment
            documents a load-bearing entry ordering (play_loop.go:635); name that edit.
          family: sweep-every-site-of-the-rule
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-02T13:06:57-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Reveal at play_loop.go:507 is now a named site in the table and in the sweep; the behaviour test drives the reveal path.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: replraw.go:651 and :633 are both in the five-site sweep, with the "\r\n" normalisation traced in the Problem section.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Step 4 makes the gap conditional on a surviving buffer row and Step 4a sweeps heights; the sum closes against the clamp at screen.go:468.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: go test -tags conformance ./cmd/define is in Verification and in Done-when.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Step 5 states the single-production-caller rationale rather than leaving it implied.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: The four-site table covers play_loop.go:247 and the boardFooter signature change is declared with its ordering caveat.
          round: 2
      findings:
        - id: PQ-7
          severity: Important
          title: the guard's predicate is scoped to playAnnounced calls, so it fails on playRegion's forwarder and is blind to the click path
          detail: |-
            4th finding in this family, so the fix is the RULE not the instance. play_loop.go:359
            and replraw.go:343 call playRegion (replraw.go:584), which forwards ind at
            replraw.go:605 — so "every playAnnounced( call passes screenIndicator()" fails at the
            forwarder and never sees the sitting click the issue is named for. Predicate that covers
            the class: every call whose corresponding parameter has type indicator must pass
            screenIndicator(), with a bare identifier permitted when the enclosing function itself
            takes an indicator. Prevalence: 5 supplying sites / 2 entry functions / 1 forwarder,
            40% invisible to the stated walk. Alternative worth weighing — playRegion's only
            production callers are screen-hosted, so deleting its ind parameter removes two sites
            rather than guarding them. Widen the matching Done-when row too; as phrased it can pass
            while play_loop.go:359 regresses.
          family: sweep-every-site-of-the-rule
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-02T13:09:35-07:00"
      agent: claude
      dispose:
        - id: PQ-7
          disposition: addressed
          note: Guard predicate is now by argument type, and playRegion's ind parameter is deleted, so the forwarder and its two supplying sites are gone structurally rather than guarded.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-09-02T13:14:51-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 1 Step 1 drives the reveal path (play_loop.go:507, confirmed passing defaultIndicator) and says why.
          round: 4
        - id: PQ-2
          disposition: addressed
          note: replraw.go:633 and the submit call (actually :651, plan says :653) are both in Task 1's sweep.
          round: 4
        - id: PQ-3
          disposition: addressed
          note: The gap is now granted only when avail >= s.gap+1, with Step 4a sweeping a range of heights.
          round: 4
        - id: PQ-4
          disposition: addressed
          note: Verification runs go test -tags conformance ./cmd/define; the file is //go:build darwin && conformance.
          round: 4
        - id: PQ-5
          disposition: addressed
          note: chromeGap is the constant, screen.gap the field; the fitsABoard asymmetry is stated and justified.
          round: 4
        - id: PQ-6
          disposition: addressed
          note: Four styling sites tabulated including play_loop.go:247's bar; boardFooter's new parameter is declared.
          round: 4
        - id: PQ-7
          disposition: addressed
          note: The guard predicate is by argument type, which reaches playRegion's forwarder at replraw.go:584.
          round: 4
      findings:
        - id: PQ-8
          severity: Minor
          title: fitsABoard is also the draw-time boardWhole predicate, so charging chromeGap there fires a false refusal at one height
          detail: |-
            This is the 2nd finding in family `frame-budget-completeness`; PQ-3 fixed the
            charge-vs-emission mismatch inside Paint. The RULE that covers both: chromeGap's
            charge and its emission must agree at every place either is consulted, and every
            consumer of fitsABoard must be named when a term is added to it. Prevalence is 2
            consumers, both live: boardFitsIn (play_loop.go:591-601) is asked at SELECTION and
            at every DRAW, and only the selection role is reasoned about in the plan. At
            termRows == boardRows+promptRows+barRows (the existing table row
            play_loop_test.go:2306, {8,6,1,true}) Paint computes avail=0, declines the gap and
            draws the board whole, while the new fitsABoard returns false so boardPrompt swaps
            in boardRefusal and Enter is held over a whole board. Safe direction, so a note.
          family: frame-budget-completeness
          round: 4
        - id: PQ-9
          severity: Minor
          title: the AST-guard step points at dict_symbols_darwin_test.go, which parses no Go source
          detail: |-
            dict_symbols_darwin_test.go compares a C resolver's symbol list against a Go list
            and imports no go/ast. The package's real precedent is repo_guard_test.go:7-8
            (go/ast + go/parser), which also already carries the Fatal-never-Skip discipline
            the plan wants (repo_guard_test.go:46-48). Reuse that walker rather than a new one.
          family: cite-the-code-you-claim
          round: 4
      blocked: false
content_hash: 65c3e24befe0fa3a58bb00c4d5decfae6a86e612a664e3494b3e8920190094c8
---

# Gate ledger — tools#44 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T13:01:16-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `sweep-every-site-of-the-rule` the sitting's reveal playback keeps defaultIndicator, so the buffer still grows a blank line per question
  play_loop.go:507-508 passes defaultIndicator(opt) into the same screen-backed
  stdout as the click path the plan fixes, so `before: "\n"` still commits one
  blank buffer line per audio reveal — the default path. The plan's claim that
  there are "two spellings of the indicator inside a screen" is wrong: there are
  five screen-hosted sites. Enumerate them in the plan and sweep them this round.
- **PQ-2** [Important] `sweep-every-site-of-the-rule` the editor's two other screen-hosted indicators are neither swept nor declared a non-goal
  replraw.go:651-652 passes `before: "\r\n"`, which screen.Write normalizes to
  "\n" (screen.go:93) and therefore commits a blank line per audio lookup — the
  same live bug, in a file the plan already edits. replraw.go:633-634 is a fourth
  copy of the literal screenIndicator() exists to own. Either adopt the
  constructor at both, or record an explicit non-goal with the reason.
- **PQ-3** [Important] `frame-budget-completeness` chromeGap is charged to the budget but emitted unconditionally, so a short terminal scrolls
  Step 4 subtracts s.gap from s.rows, but the existing clamp at screen.go:468
  absorbs the shortfall while the gap rows are still written. At termRows == promptRows
  (a 76-column keys row on a narrow window) the frame becomes termRows+1 rows and the
  terminal scrolls, which does not happen today. Paint documents an order of sacrifice;
  say where the gap sits in it and pin the boundary height.
- **PQ-4** [Important] `verification-covers-the-declared-risk` Verification runs go test ./... which cannot compile the pty conformance suite that checks footerTop
  pty_conformance_test.go is `//go:build darwin && conformance`, and its SGR-1006 click
  is the only end-to-end proof that a real terminal's click maps to the intended cell
  after footerTop moves. The plan calls that arithmetic the one that must not be wrong.
  Add `go test -tags conformance ./cmd/define` to the Verification section.
- **PQ-5** [Minor] `one-owner-per-quantity` fitsABoard hardcodes chromeGap while the screen carries it as the field s.gap
  Two owners of one quantity, in the function whose own comment says it exists because
  "a constant here was a second owner of a height displayRows already computes". Safe
  today because newPinnedScreen has one production caller (play_loop.go:106); the plan
  should state that rather than leave it implied.
- **PQ-6** [Minor] `sweep-every-site-of-the-rule` Task 3 leaves the non-board bar at play_loop.go:247 unstyled and changes boardFooter's signature silently
  "the two view.Draw calls and sittingBar's result in boardFooter" covers the board's bar
  but not the inline `[]string{sittingBar(fig)}` at play_loop.go:247, which is the common
  sitting. Threading a palette into boardFooter also alters a function whose comment
  documents a load-bearing entry ordering (play_loop.go:635); name that edit.

## Round 2 — 2026-09-02T13:06:57-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Reveal at play_loop.go:507 is now a named site in the table and in the sweep; the behaviour test drives the reveal path.
- PQ-2 — addressed — replraw.go:651 and :633 are both in the five-site sweep, with the "\r\n" normalisation traced in the Problem section.
- PQ-3 — addressed — Step 4 makes the gap conditional on a surviving buffer row and Step 4a sweeps heights; the sum closes against the clamp at screen.go:468.
- PQ-4 — addressed — go test -tags conformance ./cmd/define is in Verification and in Done-when.
- PQ-5 — addressed — Step 5 states the single-production-caller rationale rather than leaving it implied.
- PQ-6 — addressed — The four-site table covers play_loop.go:247 and the boardFooter signature change is declared with its ordering caveat.

### Raised

- **PQ-7** [Important] `sweep-every-site-of-the-rule` the guard's predicate is scoped to playAnnounced calls, so it fails on playRegion's forwarder and is blind to the click path
  4th finding in this family, so the fix is the RULE not the instance. play_loop.go:359
  and replraw.go:343 call playRegion (replraw.go:584), which forwards ind at
  replraw.go:605 — so "every playAnnounced( call passes screenIndicator()" fails at the
  forwarder and never sees the sitting click the issue is named for. Predicate that covers
  the class: every call whose corresponding parameter has type indicator must pass
  screenIndicator(), with a bare identifier permitted when the enclosing function itself
  takes an indicator. Prevalence: 5 supplying sites / 2 entry functions / 1 forwarder,
  40% invisible to the stated walk. Alternative worth weighing — playRegion's only
  production callers are screen-hosted, so deleting its ind parameter removes two sites
  rather than guarding them. Widen the matching Done-when row too; as phrased it can pass
  while play_loop.go:359 regresses.

## Round 3 — 2026-09-02T13:09:35-07:00 (claude) — passed

### Disposed

- PQ-7 — addressed — Guard predicate is now by argument type, and playRegion's ind parameter is deleted, so the forwarder and its two supplying sites are gone structurally rather than guarded.

## Round 4 — 2026-09-02T13:14:51-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 1 Step 1 drives the reveal path (play_loop.go:507, confirmed passing defaultIndicator) and says why.
- PQ-2 — addressed — replraw.go:633 and the submit call (actually :651, plan says :653) are both in Task 1's sweep.
- PQ-3 — addressed — The gap is now granted only when avail >= s.gap+1, with Step 4a sweeping a range of heights.
- PQ-4 — addressed — Verification runs go test -tags conformance ./cmd/define; the file is //go:build darwin && conformance.
- PQ-5 — addressed — chromeGap is the constant, screen.gap the field; the fitsABoard asymmetry is stated and justified.
- PQ-6 — addressed — Four styling sites tabulated including play_loop.go:247's bar; boardFooter's new parameter is declared.
- PQ-7 — addressed — The guard predicate is by argument type, which reaches playRegion's forwarder at replraw.go:584.

### Raised

- **PQ-8** [Minor] `frame-budget-completeness` fitsABoard is also the draw-time boardWhole predicate, so charging chromeGap there fires a false refusal at one height
  This is the 2nd finding in family `frame-budget-completeness`; PQ-3 fixed the
  charge-vs-emission mismatch inside Paint. The RULE that covers both: chromeGap's
  charge and its emission must agree at every place either is consulted, and every
  consumer of fitsABoard must be named when a term is added to it. Prevalence is 2
  consumers, both live: boardFitsIn (play_loop.go:591-601) is asked at SELECTION and
  at every DRAW, and only the selection role is reasoned about in the plan. At
  termRows == boardRows+promptRows+barRows (the existing table row
  play_loop_test.go:2306, {8,6,1,true}) Paint computes avail=0, declines the gap and
  draws the board whole, while the new fitsABoard returns false so boardPrompt swaps
  in boardRefusal and Enter is held over a whole board. Safe direction, so a note.
- **PQ-9** [Minor] `cite-the-code-you-claim` the AST-guard step points at dict_symbols_darwin_test.go, which parses no Go source
  dict_symbols_darwin_test.go compares a C resolver's symbol list against a Go list
  and imports no go/ast. The package's real precedent is repo_guard_test.go:7-8
  (go/ast + go/parser), which also already carries the Fatal-never-Skip discipline
  the plan wants (repo_guard_test.go:46-48). Reuse that walker rather than a new one.

## Open findings

- **PQ-8** [Minor] `frame-budget-completeness` fitsABoard is also the draw-time boardWhole predicate, so charging chromeGap there fires a false refusal at one height
- **PQ-9** [Minor] `cite-the-code-you-claim` the AST-guard step points at dict_symbols_darwin_test.go, which parses no Go source
