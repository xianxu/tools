---
gate: plan-quality
issue: 16
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-23T14:54:40-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: D5's scoped interrupter assumes a byte-only Ctrl-C the pty suite measured as a SIGINT
          detail: |-
            rawterm.go:38-42 claims NotifyContext never fires in raw mode, but
            pty_conformance_test.go:13-25 records a mutation-tested measurement that the
            interrupt arrives as a SIGINT and exits through NotifyContext. main.go:154
            installs a process-wide signal context the editor derives from, so a real
            Ctrl-C mid-stream would cancel the root ctx and runEditor's ctx.Done branch
            would end the session regardless of what the interrupter points at, failing
            the Done-when row and Task 11's own pty check.
          family: interrupt-delivery-path
          round: 1
        - id: PQ-2
          severity: Important
          title: the `session` entity is declared and used but no task creates or wires it
          detail: |-
            Core concepts lists session in cmd/define/session.go and Task 10 takes
            `sess *session`, yet no step creates it, defines `exchange`, or migrates
            `current string` in replLines (repl.go:145), runEditor (replraw.go), and
            submitLine's `current *string` (replraw.go:230). The multi-turn Done-when
            row maps to Tasks 9 and 10, neither of which does that wiring.
          family: entity-without-task
          round: 1
        - id: PQ-3
          severity: Important
          title: an ask outcome carries code 0, and the plan never says what the loops do with it
          detail: |-
            repl.go:173 sets `current` on a zero code and replraw.go:242-245 does
            hist.Add plus `*current = line`, so a routed question becomes the current
            word — a bare Enter speaks it and the next askContext.CurrentWord is the
            prior question. main.go:328's one-shot has the same unnamed branch, and the
            unforced raw-loop ask returns out of submitLine while Task 11 wires only the
            cmdAsk branch, risking two copies of the interrupter/crlfWriter wiring
            (ARCH-DRY).
          family: new-outcome-state-contract
          round: 1
        - id: PQ-4
          severity: Important
          title: runAsk's error taxonomy omits the user-cancelled stream that D5 introduces
          detail: |-
            Task 10 enumerates Resolve/ErrUnavailable/ErrRequest but not a stream aborted
            by Ctrl-C, so the answer path would print an error for the user's own
            keypress. Precedent in-tree: playAnnounced's cancelled-context guard in
            main.go and llmcheck.go:62's parent.Err() check. Name it in Task 10 and
            assert it in Task 11.
          family: cancel-is-not-an-error
          round: 1
        - id: PQ-5
          severity: Minor
          title: the plan names test-infra symbols that do not exist
          detail: |-
            Task 10 uses fake.URL(), fake.LastRequestBody(), fake.LastAnswer(); the real
            Fake (internal/llm/llmtest/fake.go:265-322) embeds *httptest.Server (URL is a
            field), scripts replies via Script/ServeRecorded, and exposes Requests()
            []Recorded with Prompt()/System(). D1 cites TestCaptureHappensOncePerLookup;
            the actual test is TestCaptureArityIsOnePerLookup (capture_test.go:89).
          family: stale-api-reference
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-23T15:01:04-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'D5 rewritten: one sink, both transports, WithoutCancel; Task 7 tests both, Task 11 adds the pty row.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Task 4 creates session.go and migrates all three `current` declarations; exchange/turns explicitly M2.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Task 4 states the ask outcome contract as a table with a test per row and one askInSession closure.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: runAsk's taxonomy leads with ctx.Err() != nil → nothing printed, code 0; asks the context, not the error.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: llmtest API and D1 fixed, but Task 3 still names TestCaptureHappensOncePerLookup and repl.go:145 is really :125 — instance fixed, class not swept. Minor.
          round: 2
      findings:
        - id: PQ-6
          severity: Important
          title: D5 detaches the signal context on a branch that serves the piped loop too, leaving replLines with no interrupt transport
          detail: |-
            This is the 2nd finding in family `interrupt-delivery-path`. Do not patch
            the site — state and fix the rule. Rule: the WithoutCancel detach and the
            sink must be installed at the same place, where the loop that owns the
            interrupt is actually chosen, and D5 must enumerate every loop entry path
            against every transport rather than naming transports for replRaw alone.
            Evidence: main.go:314-321's `case 0:` calls repl, and repl.go:96-97 sends
            every non-terminalUI zero-arg run (echo word | define, define < f.txt,
            redirected stdout, and replraw.go:20,25's own fallbacks) to replLines,
            which depends on ctx.Done() at repl.go:132 and gets no watcher under D5
            wiring #2. With main.go:154's NotifyContext still diverting SIGINT from
            default termination, that path becomes uninterruptible — contradicting
            D5's own claim that the piped path keeps the signal context. Measured
            prevalence: 2 of 2 interrupt-path statements in this plan were wrong about
            which transport serves which loop.
          family: interrupt-delivery-path
          round: 2
      blocked: true
---

# Gate ledger — tools#16 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-23T14:54:40-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `interrupt-delivery-path` D5's scoped interrupter assumes a byte-only Ctrl-C the pty suite measured as a SIGINT
  rawterm.go:38-42 claims NotifyContext never fires in raw mode, but
  pty_conformance_test.go:13-25 records a mutation-tested measurement that the
  interrupt arrives as a SIGINT and exits through NotifyContext. main.go:154
  installs a process-wide signal context the editor derives from, so a real
  Ctrl-C mid-stream would cancel the root ctx and runEditor's ctx.Done branch
  would end the session regardless of what the interrupter points at, failing
  the Done-when row and Task 11's own pty check.
- **PQ-2** [Important] `entity-without-task` the `session` entity is declared and used but no task creates or wires it
  Core concepts lists session in cmd/define/session.go and Task 10 takes
  `sess *session`, yet no step creates it, defines `exchange`, or migrates
  `current string` in replLines (repl.go:145), runEditor (replraw.go), and
  submitLine's `current *string` (replraw.go:230). The multi-turn Done-when
  row maps to Tasks 9 and 10, neither of which does that wiring.
- **PQ-3** [Important] `new-outcome-state-contract` an ask outcome carries code 0, and the plan never says what the loops do with it
  repl.go:173 sets `current` on a zero code and replraw.go:242-245 does
  hist.Add plus `*current = line`, so a routed question becomes the current
  word — a bare Enter speaks it and the next askContext.CurrentWord is the
  prior question. main.go:328's one-shot has the same unnamed branch, and the
  unforced raw-loop ask returns out of submitLine while Task 11 wires only the
  cmdAsk branch, risking two copies of the interrupter/crlfWriter wiring
  (ARCH-DRY).
- **PQ-4** [Important] `cancel-is-not-an-error` runAsk's error taxonomy omits the user-cancelled stream that D5 introduces
  Task 10 enumerates Resolve/ErrUnavailable/ErrRequest but not a stream aborted
  by Ctrl-C, so the answer path would print an error for the user's own
  keypress. Precedent in-tree: playAnnounced's cancelled-context guard in
  main.go and llmcheck.go:62's parent.Err() check. Name it in Task 10 and
  assert it in Task 11.
- **PQ-5** [Minor] `stale-api-reference` the plan names test-infra symbols that do not exist
  Task 10 uses fake.URL(), fake.LastRequestBody(), fake.LastAnswer(); the real
  Fake (internal/llm/llmtest/fake.go:265-322) embeds *httptest.Server (URL is a
  field), scripts replies via Script/ServeRecorded, and exposes Requests()
  []Recorded with Prompt()/System(). D1 cites TestCaptureHappensOncePerLookup;
  the actual test is TestCaptureArityIsOnePerLookup (capture_test.go:89).

## Round 2 — 2026-08-23T15:01:04-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — D5 rewritten: one sink, both transports, WithoutCancel; Task 7 tests both, Task 11 adds the pty row.
- PQ-2 — addressed — Task 4 creates session.go and migrates all three `current` declarations; exchange/turns explicitly M2.
- PQ-3 — addressed — Task 4 states the ask outcome contract as a table with a test per row and one askInSession closure.
- PQ-4 — addressed — runAsk's taxonomy leads with ctx.Err() != nil → nothing printed, code 0; asks the context, not the error.
- PQ-5 — not-addressed — llmtest API and D1 fixed, but Task 3 still names TestCaptureHappensOncePerLookup and repl.go:145 is really :125 — instance fixed, class not swept. Minor.

### Raised

- **PQ-6** [Important] `interrupt-delivery-path` D5 detaches the signal context on a branch that serves the piped loop too, leaving replLines with no interrupt transport
  This is the 2nd finding in family `interrupt-delivery-path`. Do not patch
  the site — state and fix the rule. Rule: the WithoutCancel detach and the
  sink must be installed at the same place, where the loop that owns the
  interrupt is actually chosen, and D5 must enumerate every loop entry path
  against every transport rather than naming transports for replRaw alone.
  Evidence: main.go:314-321's `case 0:` calls repl, and repl.go:96-97 sends
  every non-terminalUI zero-arg run (echo word | define, define < f.txt,
  redirected stdout, and replraw.go:20,25's own fallbacks) to replLines,
  which depends on ctx.Done() at repl.go:132 and gets no watcher under D5
  wiring #2. With main.go:154's NotifyContext still diverting SIGINT from
  default termination, that path becomes uninterruptible — contradicting
  D5's own claim that the piped path keeps the signal context. Measured
  prevalence: 2 of 2 interrupt-path statements in this plan were wrong about
  which transport serves which loop.

## Open findings

- **PQ-5** [Minor] `stale-api-reference` the plan names test-infra symbols that do not exist
- **PQ-6** [Important] `interrupt-delivery-path` D5 detaches the signal context on a branch that serves the piped loop too, leaving replLines with no interrupt transport
