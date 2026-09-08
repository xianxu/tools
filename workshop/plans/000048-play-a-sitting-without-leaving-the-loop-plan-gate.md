---
gate: plan-quality
issue: 48
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-07T23:57:31-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: newConsole's finish restores the shared rawSession, so a sitting started from the loop hands the terminal back mid-REPL
          detail: |-
            newConsole sets finish to onceHandBack(live, sess, stdout); handBack runs
            live.Stop(), sess.restore() (rawterm.go:50 — leaves mouse, leaves alt
            screen, term.Restore) and prints the transcript to the real stdout.
            playSession calls con.finish() from over(), on every exit path. Reusing
            newConsole over the REPL's live session therefore un-raws the terminal
            when the sitting ends and runEditor keeps painting frames into a cooked
            normal buffer. enterAlt's idempotency (rawterm.go:147-150) is real and is
            not this hazard. Say who owns the hand-back and where the sitting's
            summary lands — the Spec's "What the REPL looks like on return" decision
            is currently unanswered.
          family: terminal-ownership-unstated
          round: 1
        - id: PQ-2
          severity: Critical
          title: the performing half has neither the rawSession nor the real tty in scope inside runEditor
          detail: |-
            runEditor (replraw.go:253) takes only ctx, keys, interrupts, d, opt, con;
            console carries view/resizes/finish/stdout/stderr and its stdout IS the
            editor's liveScreen. The *rawSession and the real stdout are locals of
            replRaw (replraw.go:25-33), and runEditor already binds `sess` to a
            `session` at replraw.go:277. Passing con.stdout into newConsole makes
            terminalCols fall back to defaultCols (main.go:1270-1281), painting the
            sitting at a fabricated 80x24 into the editor's own buffer. The plan must
            state which seam changes: replRaw builds the startSitting closure, or
            console gains the tty and session.
          family: terminal-ownership-unstated
          round: 1
        - id: PQ-3
          severity: Important
          title: the nil-capability refusal omits the absent-deck case, which todaysQuestions dereferences
          detail: |-
            The plan's rule covers one-shot, pipe and line-mode. todaysQuestions calls
            d.deck.Deck() on its first line (play_loop.go:903), which is why runPlay
            guards d.deck == nil at play_loop.go:29. /stats — landed in ff962e5 —
            already has the sibling shape at stats.go:207 using noDeckMessage
            (main.go:1215). Name the case and reuse that helper rather than reaching a
            panic or writing a fifth wording (ARCH-DRY).
          family: refusal-enumeration-incomplete
          round: 1
        - id: PQ-4
          severity: Important
          title: the ARCH-ORDER "no concurrency" N/A is wrong — newConsole starts a resize watcher and each screen carries a paint timer
          detail: |-
            watchResize (rawterm.go:230-242) spawns a goroutine bound to the ctx it is
            given and is deliberately never signal.Stop'd, and liveScreen repaints
            from its own timer goroutine (screen.go:~628). A sitting built through
            newConsole therefore leaks a SIGWINCH watcher per /play if it takes the
            loop's ctx, and puts two liveScreens over one tty whose timers can both
            fire. State the extent: who is still running when sittingInPlace returns,
            which ctx the sitting's console gets, and which screen owns the tty while
            the other is suspended.
          family: unstated-goroutine-extent
          round: 1
      blocked: true
---

# Gate ledger — tools#48 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T23:57:31-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `terminal-ownership-unstated` newConsole's finish restores the shared rawSession, so a sitting started from the loop hands the terminal back mid-REPL
  newConsole sets finish to onceHandBack(live, sess, stdout); handBack runs
  live.Stop(), sess.restore() (rawterm.go:50 — leaves mouse, leaves alt
  screen, term.Restore) and prints the transcript to the real stdout.
  playSession calls con.finish() from over(), on every exit path. Reusing
  newConsole over the REPL's live session therefore un-raws the terminal
  when the sitting ends and runEditor keeps painting frames into a cooked
  normal buffer. enterAlt's idempotency (rawterm.go:147-150) is real and is
  not this hazard. Say who owns the hand-back and where the sitting's
  summary lands — the Spec's "What the REPL looks like on return" decision
  is currently unanswered.
- **PQ-2** [Critical] `terminal-ownership-unstated` the performing half has neither the rawSession nor the real tty in scope inside runEditor
  runEditor (replraw.go:253) takes only ctx, keys, interrupts, d, opt, con;
  console carries view/resizes/finish/stdout/stderr and its stdout IS the
  editor's liveScreen. The *rawSession and the real stdout are locals of
  replRaw (replraw.go:25-33), and runEditor already binds `sess` to a
  `session` at replraw.go:277. Passing con.stdout into newConsole makes
  terminalCols fall back to defaultCols (main.go:1270-1281), painting the
  sitting at a fabricated 80x24 into the editor's own buffer. The plan must
  state which seam changes: replRaw builds the startSitting closure, or
  console gains the tty and session.
- **PQ-3** [Important] `refusal-enumeration-incomplete` the nil-capability refusal omits the absent-deck case, which todaysQuestions dereferences
  The plan's rule covers one-shot, pipe and line-mode. todaysQuestions calls
  d.deck.Deck() on its first line (play_loop.go:903), which is why runPlay
  guards d.deck == nil at play_loop.go:29. /stats — landed in ff962e5 —
  already has the sibling shape at stats.go:207 using noDeckMessage
  (main.go:1215). Name the case and reuse that helper rather than reaching a
  panic or writing a fifth wording (ARCH-DRY).
- **PQ-4** [Important] `unstated-goroutine-extent` the ARCH-ORDER "no concurrency" N/A is wrong — newConsole starts a resize watcher and each screen carries a paint timer
  watchResize (rawterm.go:230-242) spawns a goroutine bound to the ctx it is
  given and is deliberately never signal.Stop'd, and liveScreen repaints
  from its own timer goroutine (screen.go:~628). A sitting built through
  newConsole therefore leaks a SIGWINCH watcher per /play if it takes the
  loop's ctx, and puts two liveScreens over one tty whose timers can both
  fire. State the extent: who is still running when sittingInPlace returns,
  which ctx the sitting's console gets, and which screen owns the tty while
  the other is suspended.

## Open findings

- **PQ-1** [Critical] `terminal-ownership-unstated` newConsole's finish restores the shared rawSession, so a sitting started from the loop hands the terminal back mid-REPL
- **PQ-2** [Critical] `terminal-ownership-unstated` the performing half has neither the rawSession nor the real tty in scope inside runEditor
- **PQ-3** [Important] `refusal-enumeration-incomplete` the nil-capability refusal omits the absent-deck case, which todaysQuestions dereferences
- **PQ-4** [Important] `unstated-goroutine-extent` the ARCH-ORDER "no concurrency" N/A is wrong — newConsole starts a resize watcher and each screen carries a paint timer
