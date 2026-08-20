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

## Open findings

- **PQ-1** [Critical] Plan and Done-when claim `echo word | define` works today; it exits 2 with usage
- **PQ-2** [Important] No injectable TTY seam, so Task 5's "TTY-ish stdin" test is unwritable
- **PQ-3** [Important] Session temp dir contradicts the verbatim extraction of speak
- **PQ-4** [Important] Cancellation contract unstated, and NotifyContext changes the shipped one-shot path
- **PQ-5** [Important] cachingAudioSource wired only in realDeps, so no test covers the production wiring
- **PQ-6** [Important] No stated non-goals for a REPL, where scope creep is line editing
- **PQ-7** [Important] Reader-goroutine strategy names no adversarial input class or mechanical guard
- **PQ-8** [Minor] Task 2, 3 and 4 Step 1 enumerate test cases in prose
- **PQ-9** [Minor] Prompt rendering and flag behaviour inside the loop are unspecified
