---
gate: plan-quality
issue: 14
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-20T16:30:21-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Raw mode kills the Ctrl-C-during-playback cancellation path; plan handles only Ctrl-C at the prompt
          detail: |-
            MakeRaw clears ISIG, so signal.NotifyContext (main.go:44) never fires and afplay is no longer
            killed via exec.CommandContext (player.go:36) — behaviour recorded at atlas/define.md:212-215.
            The plan's ActInterrupt only exists at the prompt; during playback the loop blocks in
            playAnnounced and nobody reads the key channel, so 0x03 is consumed only after audio finishes.
            Name who cancels the context now.
          round: 1
        - id: PQ-2
          severity: Important
          title: Done-when requires a real-pty test; plan delivers only a manual check and names no pty dependency
          detail: |-
            The Architecture section twice asserts a pty test exists, but Task 6 Step 5 is a manual sh block
            and no task creates one. go.mod has only x/term plus x/sys indirect, so the pty approach
            (creack/pty test-only, or /dev/ptmx via x/sys promoted to direct) is an undeclared decision.
            Relatedly ARCH-MOCK - the terminal is a new external surface with three sibling conformance
            tests in this package, and nothing checks decodeKey's vocabulary against a real terminal.
          round: 1
        - id: PQ-3
          severity: Important
          title: The predicate that gates raw mode is never stated, in the exact place this repo has regressed 3x
          detail: |-
            repl.go:74-79 documents three prior bugs about which stream gates UI. The plan covers only piped
            stdin. Gating on `interactive` makes `define > out.txt` type blind with ECHO off and no rendered
            frame; gating on `terminalUI` makes `define -no-color` on a tty lose all editing, since -no-color
            already forces opt.tty false (main.go:99). Pick one explicitly.
          round: 1
        - id: PQ-4
          severity: Important
          title: History seam leaves open the successful-vs-typed question that issue 15 deferred to this issue
          detail: |-
            Issue 15 (workshop/issues/000015-repl-commands.md:38-43) asks whether the deck's successful
            lookups and the editor's typed history are one source with a success flag or two. The plan
            declares Add(word string) but never says when it is called. This issue's Done-when makes the
            seam the deliverable that issue 3 fills, so the Add semantics are a cross-issue contract.
          round: 1
        - id: PQ-5
          severity: Minor
          title: Per-task Obligations lists are enumerated prose test cases; compress to one strategy line each
          detail: |-
            Tasks 1-6 Step 1 enumerate cases that will be rewritten as Go tables within the hour. Task 1's
            fuzz-target line is the right form. Keep the adversarial class and mechanical guard per risky
            function; drop the case lists.
          round: 1
        - id: PQ-6
          severity: Minor
          title: RenderLine placement collides with the existing Entry renderer and its RenderOpts
          detail: |-
            The entity table puts RenderLine in editor.go but Task 5 tests it in render_test.go, which
            already holds 233 lines for the Entry renderer, and RenderOpts is already defined at
            render.go:10 for a different renderer. Say whether the option type is shared or new.
          round: 1
        - id: PQ-7
          severity: Minor
          title: Plan does not name the existing tests that necessarily die with the cooked-mode workaround
          detail: |-
            Risks says issue 2's tests "must stay green rather than be rewritten", but
            TestREPLReplayFlashesThenRestoresThePrompt (repl_test.go:243),
            TestREPLFailedReplayDoesNotEatThePrompt (:316) and TestREPLFailedReplayDoesNotWriteOntoThePrompt
            (:346) pin eraseLineAndStepBack/skipPrompt and are retired by their deletion. Say what replaces
            the coverage they provided.
          round: 1
        - id: PQ-8
          severity: Minor
          title: Apply and Suggestion take the History interface, putting IO inside the declared pure core
          detail: |-
            ARCH-PURE - once issue 3's store fills the seam, a store query runs per keystroke inside the
            state machine, and the "pure" entities need a double to test. Consider having the loop resolve
            candidates and hand Editor a plain snapshot slice; the Apply signature is fixed by this plan and
            cited in the estimate, so it is cheapest to settle now.
          round: 1
      blocked: true
---

# Gate ledger — tools#14 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-20T16:30:21-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] Raw mode kills the Ctrl-C-during-playback cancellation path; plan handles only Ctrl-C at the prompt
  MakeRaw clears ISIG, so signal.NotifyContext (main.go:44) never fires and afplay is no longer
  killed via exec.CommandContext (player.go:36) — behaviour recorded at atlas/define.md:212-215.
  The plan's ActInterrupt only exists at the prompt; during playback the loop blocks in
  playAnnounced and nobody reads the key channel, so 0x03 is consumed only after audio finishes.
  Name who cancels the context now.
- **PQ-2** [Important] Done-when requires a real-pty test; plan delivers only a manual check and names no pty dependency
  The Architecture section twice asserts a pty test exists, but Task 6 Step 5 is a manual sh block
  and no task creates one. go.mod has only x/term plus x/sys indirect, so the pty approach
  (creack/pty test-only, or /dev/ptmx via x/sys promoted to direct) is an undeclared decision.
  Relatedly ARCH-MOCK - the terminal is a new external surface with three sibling conformance
  tests in this package, and nothing checks decodeKey's vocabulary against a real terminal.
- **PQ-3** [Important] The predicate that gates raw mode is never stated, in the exact place this repo has regressed 3x
  repl.go:74-79 documents three prior bugs about which stream gates UI. The plan covers only piped
  stdin. Gating on `interactive` makes `define > out.txt` type blind with ECHO off and no rendered
  frame; gating on `terminalUI` makes `define -no-color` on a tty lose all editing, since -no-color
  already forces opt.tty false (main.go:99). Pick one explicitly.
- **PQ-4** [Important] History seam leaves open the successful-vs-typed question that issue 15 deferred to this issue
  Issue 15 (workshop/issues/000015-repl-commands.md:38-43) asks whether the deck's successful
  lookups and the editor's typed history are one source with a success flag or two. The plan
  declares Add(word string) but never says when it is called. This issue's Done-when makes the
  seam the deliverable that issue 3 fills, so the Add semantics are a cross-issue contract.
- **PQ-5** [Minor] Per-task Obligations lists are enumerated prose test cases; compress to one strategy line each
  Tasks 1-6 Step 1 enumerate cases that will be rewritten as Go tables within the hour. Task 1's
  fuzz-target line is the right form. Keep the adversarial class and mechanical guard per risky
  function; drop the case lists.
- **PQ-6** [Minor] RenderLine placement collides with the existing Entry renderer and its RenderOpts
  The entity table puts RenderLine in editor.go but Task 5 tests it in render_test.go, which
  already holds 233 lines for the Entry renderer, and RenderOpts is already defined at
  render.go:10 for a different renderer. Say whether the option type is shared or new.
- **PQ-7** [Minor] Plan does not name the existing tests that necessarily die with the cooked-mode workaround
  Risks says issue 2's tests "must stay green rather than be rewritten", but
  TestREPLReplayFlashesThenRestoresThePrompt (repl_test.go:243),
  TestREPLFailedReplayDoesNotEatThePrompt (:316) and TestREPLFailedReplayDoesNotWriteOntoThePrompt
  (:346) pin eraseLineAndStepBack/skipPrompt and are retired by their deletion. Say what replaces
  the coverage they provided.
- **PQ-8** [Minor] Apply and Suggestion take the History interface, putting IO inside the declared pure core
  ARCH-PURE - once issue 3's store fills the seam, a store query runs per keystroke inside the
  state machine, and the "pure" entities need a double to test. Consider having the loop resolve
  candidates and hand Editor a plain snapshot slice; the Apply signature is fixed by this plan and
  cited in the estimate, so it is cheapest to settle now.

## Open findings

- **PQ-1** [Critical] Raw mode kills the Ctrl-C-during-playback cancellation path; plan handles only Ctrl-C at the prompt
- **PQ-2** [Important] Done-when requires a real-pty test; plan delivers only a manual check and names no pty dependency
- **PQ-3** [Important] The predicate that gates raw mode is never stated, in the exact place this repo has regressed 3x
- **PQ-4** [Important] History seam leaves open the successful-vs-typed question that issue 15 deferred to this issue
- **PQ-5** [Minor] Per-task Obligations lists are enumerated prose test cases; compress to one strategy line each
- **PQ-6** [Minor] RenderLine placement collides with the existing Entry renderer and its RenderOpts
- **PQ-7** [Minor] Plan does not name the existing tests that necessarily die with the cooked-mode workaround
- **PQ-8** [Minor] Apply and Suggestion take the History interface, putting IO inside the declared pure core
