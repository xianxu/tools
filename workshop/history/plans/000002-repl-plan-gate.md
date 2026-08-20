---
gate: plan-quality
issue: 2
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-20T12:46:58-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Plan and Done-when claim `echo word | define` works today; it exits 2 with usage
          detail: |-
            run() never reads stdin (no os.Stdin anywhere in cmd/) and cmd/define/main.go:60
            returns 2 on NArg() != 1 — verified by running bin/define. Task 5's "piped stdin
            to the one-shot path" is therefore underdetermined: preserve exit 2 (and fix the
            Done-when wording), or add reading the word from stdin as new declared scope with
            its own tests and a decision on TestRunNoArgsIsUsageError (main_test.go:87).
          round: 1
        - id: PQ-2
          severity: Important
          title: No injectable TTY seam, so Task 5's "TTY-ish stdin" test is unwritable
          detail: |-
            isTerminal (main.go:116-119) type-asserts *os.File and calls term.IsTerminal, and
            run (main.go:39) takes no stdin parameter. Name the seam and the run() signature
            change explicitly — this is a boundary change, not an implementation detail (ARCH-PURE).
          round: 1
        - id: PQ-3
          severity: Important
          title: Session temp dir contradicts the verbatim extraction of speak
          detail: |-
            Chunk 1 says repl "owns the session temp dir" and Task 4 Step 3 says one MkdirTemp
            per session, but speak creates and removes its own per call (main.go:100-104) and
            Task 1 extracts verbatim. Task 4's Files list omits main.go. Drop the session dir
            or declare speak's signature change and add the file.
          round: 1
        - id: PQ-4
          severity: Important
          title: Cancellation contract unstated, and NotifyContext changes the shipped one-shot path
          detail: |-
            run passes context.Background() today (main.go:87). With signal.NotifyContext in
            main(), a Ctrl-C during one-shot playback makes playN return ctx.Err()
            (player.go:52-55), which speak wraps and run prints on stderr (main.go:88). State
            what Ctrl-C prints and returns for both the loop and the one-shot path.
          round: 1
        - id: PQ-5
          severity: Important
          title: cachingAudioSource wired only in realDeps, so no test covers the production wiring
          detail: |-
            realDeps() takes no arguments (main.go:24), so "for the REPL only" needs a second
            constructor and the e2e test must hand-wrap the decorator — a regression dropping
            it from production stays green while the "no second CDN request" Done-when appears
            covered. Apply it inside repl, or build test deps through the production constructor.
          round: 1
        - id: PQ-6
          severity: Important
          title: No stated non-goals for a REPL, where scope creep is line editing
          detail: |-
            Say explicitly that history, arrow keys, tab completion, readline and cross-session
            persistence are out, and why. The Tech Stack lists golang.org/x/term while the
            reader is bufio.Scanner, which reads as an unresolved intention.
          round: 1
        - id: PQ-7
          severity: Important
          title: Reader-goroutine strategy names no adversarial input class or mechanical guard
          detail: |-
            bufio.Scanner returns false with ErrTooLong past 64 KB, so "EOF means return 0"
            would silently quit the REPL on a large paste. Name the guard - scanner.Err()
            distinguished from EOF, and go test -race for the goroutine sharing the current
            word with the loop, which also stays blocked on stdin after repl returns.
          round: 1
        - id: PQ-8
          severity: Minor
          title: Task 2, 3 and 4 Step 1 enumerate test cases in prose
          detail: |-
            Chunk 1's Test surface paragraph already states the strategy correctly. Keep only
            the bullets that are design decisions - a failed fetch is not cached, an unknown
            word leaves the current word unchanged - and move them into the Spec.
          round: 1
        - id: PQ-9
          severity: Minor
          title: Prompt rendering and flag behaviour inside the loop are unspecified
          detail: |-
            The Spec's prompt glyph appears in no task and no stream is named for it, and
            nothing says whether -times, -locale, -no-audio and -raw apply inside the loop -
            `define -times 1` with no word also has NArg() == 0.
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-20T12:50:56-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Spec contradiction fully fixed; the demanded decision on TestRunNoArgsIsUsageError (main_test.go:87) is still absent, and Task 1 Step 3's "a test that needs editing is the signal" norm now points the wrong way for Task 5.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: TTY seam named well (stdinIsTerminal on deps); run()'s signature change is still unstated — it needs both a stdin io.Reader and a ctx, since run manufactures context.Background() at main.go:87.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Chunk 1 line 80 now says repl owns no temp dir, but Task 4 Step 3 (line 139) still instructs "One os.MkdirTemp for the session" — the plan contradicts itself.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Dedicated Cancellation contract section; states exit 0 and that the shipped one-shot path changes intentionally. The stderr line Ctrl-C now emits is left to the manual check.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Decorator applied inside repl, not realDeps, so test and production wiring are the same line.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Non-goals section rules out readline/history/completion/multi-line/pager; x/term is now justified as the stdin TTY probe.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: ErrTooLong and the outliving goroutine named as adversarial classes; -race promoted into Done-when and Risks.
          round: 2
        - id: PQ-8
          disposition: not-addressed
          note: Task 4's list was compressed to design decisions, but Task 2 (five cases) and Task 3 (three cases) still enumerate in prose. Minor — does not block.
          round: 2
        - id: PQ-9
          disposition: addressed
          note: Flags-inside-the-loop section plus the prompt glyph routed to stdout only when interactive.
          round: 2
      findings:
        - id: PQ-10
          severity: Minor
          title: The ErrTooLong guard as written cannot "report it and continue"
          detail: |-
            Once bufio.Scanner returns ErrTooLong, every subsequent Scan() returns false, so
            Task 4 Step 1's "check scanner.Err() separately, report it, and continue" either
            spins or exits the loop anyway. The mechanical guard is scanner.Buffer(buf, max)
            to raise the cap, or recreating the reader after the error.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-20T12:53:17-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Declared as new scope in Spec, Done-when, and Task 5; TestRunNoArgsIsUsageError rewritten, not deleted.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: stdinIsTerminal is a deps field and the run() signature change is stated explicitly with both additions justified.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Session temp dir dropped; speak keeps its per-call dir, so Task 1 stays verbatim and main.go is correctly absent from Task 4.
          round: 3
        - id: PQ-8
          disposition: addressed
          note: Task 4 compressed to design decisions plus two named adversarial classes; the Task 2/3 residue is the keep-worthy kind.
          round: 3
        - id: PQ-10
          disposition: addressed
          note: scanner.Buffer cap landed in Step 3; drop Step 1's leftover "report it, and continue" clause when writing the test.
          round: 3
      blocked: false
content_hash: a8a258f9aec0e2fa0d1af44c7002ffe283933ca44c327467afd9953bbf41c9c6
---

# Gate ledger — tools#2 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-20T12:46:58-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] Plan and Done-when claim `echo word | define` works today; it exits 2 with usage
  run() never reads stdin (no os.Stdin anywhere in cmd/) and cmd/define/main.go:60
  returns 2 on NArg() != 1 — verified by running bin/define. Task 5's "piped stdin
  to the one-shot path" is therefore underdetermined: preserve exit 2 (and fix the
  Done-when wording), or add reading the word from stdin as new declared scope with
  its own tests and a decision on TestRunNoArgsIsUsageError (main_test.go:87).
- **PQ-2** [Important] No injectable TTY seam, so Task 5's "TTY-ish stdin" test is unwritable
  isTerminal (main.go:116-119) type-asserts *os.File and calls term.IsTerminal, and
  run (main.go:39) takes no stdin parameter. Name the seam and the run() signature
  change explicitly — this is a boundary change, not an implementation detail (ARCH-PURE).
- **PQ-3** [Important] Session temp dir contradicts the verbatim extraction of speak
  Chunk 1 says repl "owns the session temp dir" and Task 4 Step 3 says one MkdirTemp
  per session, but speak creates and removes its own per call (main.go:100-104) and
  Task 1 extracts verbatim. Task 4's Files list omits main.go. Drop the session dir
  or declare speak's signature change and add the file.
- **PQ-4** [Important] Cancellation contract unstated, and NotifyContext changes the shipped one-shot path
  run passes context.Background() today (main.go:87). With signal.NotifyContext in
  main(), a Ctrl-C during one-shot playback makes playN return ctx.Err()
  (player.go:52-55), which speak wraps and run prints on stderr (main.go:88). State
  what Ctrl-C prints and returns for both the loop and the one-shot path.
- **PQ-5** [Important] cachingAudioSource wired only in realDeps, so no test covers the production wiring
  realDeps() takes no arguments (main.go:24), so "for the REPL only" needs a second
  constructor and the e2e test must hand-wrap the decorator — a regression dropping
  it from production stays green while the "no second CDN request" Done-when appears
  covered. Apply it inside repl, or build test deps through the production constructor.
- **PQ-6** [Important] No stated non-goals for a REPL, where scope creep is line editing
  Say explicitly that history, arrow keys, tab completion, readline and cross-session
  persistence are out, and why. The Tech Stack lists golang.org/x/term while the
  reader is bufio.Scanner, which reads as an unresolved intention.
- **PQ-7** [Important] Reader-goroutine strategy names no adversarial input class or mechanical guard
  bufio.Scanner returns false with ErrTooLong past 64 KB, so "EOF means return 0"
  would silently quit the REPL on a large paste. Name the guard - scanner.Err()
  distinguished from EOF, and go test -race for the goroutine sharing the current
  word with the loop, which also stays blocked on stdin after repl returns.
- **PQ-8** [Minor] Task 2, 3 and 4 Step 1 enumerate test cases in prose
  Chunk 1's Test surface paragraph already states the strategy correctly. Keep only
  the bullets that are design decisions - a failed fetch is not cached, an unknown
  word leaves the current word unchanged - and move them into the Spec.
- **PQ-9** [Minor] Prompt rendering and flag behaviour inside the loop are unspecified
  The Spec's prompt glyph appears in no task and no stream is named for it, and
  nothing says whether -times, -locale, -no-audio and -raw apply inside the loop -
  `define -times 1` with no word also has NArg() == 0.

## Round 2 — 2026-08-20T12:50:56-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Spec contradiction fully fixed; the demanded decision on TestRunNoArgsIsUsageError (main_test.go:87) is still absent, and Task 1 Step 3's "a test that needs editing is the signal" norm now points the wrong way for Task 5.
- PQ-2 — not-addressed — TTY seam named well (stdinIsTerminal on deps); run()'s signature change is still unstated — it needs both a stdin io.Reader and a ctx, since run manufactures context.Background() at main.go:87.
- PQ-3 — not-addressed — Chunk 1 line 80 now says repl owns no temp dir, but Task 4 Step 3 (line 139) still instructs "One os.MkdirTemp for the session" — the plan contradicts itself.
- PQ-4 — addressed — Dedicated Cancellation contract section; states exit 0 and that the shipped one-shot path changes intentionally. The stderr line Ctrl-C now emits is left to the manual check.
- PQ-5 — addressed — Decorator applied inside repl, not realDeps, so test and production wiring are the same line.
- PQ-6 — addressed — Non-goals section rules out readline/history/completion/multi-line/pager; x/term is now justified as the stdin TTY probe.
- PQ-7 — addressed — ErrTooLong and the outliving goroutine named as adversarial classes; -race promoted into Done-when and Risks.
- PQ-8 — not-addressed — Task 4's list was compressed to design decisions, but Task 2 (five cases) and Task 3 (three cases) still enumerate in prose. Minor — does not block.
- PQ-9 — addressed — Flags-inside-the-loop section plus the prompt glyph routed to stdout only when interactive.

### Raised

- **PQ-10** [Minor] The ErrTooLong guard as written cannot "report it and continue"
  Once bufio.Scanner returns ErrTooLong, every subsequent Scan() returns false, so
  Task 4 Step 1's "check scanner.Err() separately, report it, and continue" either
  spins or exits the loop anyway. The mechanical guard is scanner.Buffer(buf, max)
  to raise the cap, or recreating the reader after the error.

## Round 3 — 2026-08-20T12:53:17-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Declared as new scope in Spec, Done-when, and Task 5; TestRunNoArgsIsUsageError rewritten, not deleted.
- PQ-2 — addressed — stdinIsTerminal is a deps field and the run() signature change is stated explicitly with both additions justified.
- PQ-3 — addressed — Session temp dir dropped; speak keeps its per-call dir, so Task 1 stays verbatim and main.go is correctly absent from Task 4.
- PQ-8 — addressed — Task 4 compressed to design decisions plus two named adversarial classes; the Task 2/3 residue is the keep-worthy kind.
- PQ-10 — addressed — scanner.Buffer cap landed in Step 3; drop Step 1's leftover "report it, and continue" clause when writing the test.

## Open findings

(none — every finding has been disposed)
