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
    - "n": 2
      timestamp: "2026-09-08T15:18:01-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: hand-back ownership is answered; where the summary lands is not, and the no-op restorer breaks handBack's cooked-terminal precondition
          round: 2
        - id: PQ-2
          disposition: addressed
          note: console gains newSitting, built in newConsole where sess and the real stdout are in scope; runEditor's signature untouched
          round: 2
        - id: PQ-3
          disposition: addressed
          note: nil deck named as a separate refusal reusing noDeckMessage(c.noCapture)
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: two-painter hazard handled; the extent clause — who is still running when sittingInPlace returns — is still unwritten
          round: 2
      findings:
        - id: PQ-5
          severity: Important
          title: liveScreen suspend/resume is the plan's one new capability and it has no named test, no strategy line, and no state model
          detail: |-
            The plan calls it "the smallest piece that cannot be avoided" yet it appears
            in neither the Pure entities nor the Integration points table, and the Test
            surface paragraph covers only runPlayCommand and sittingInPlace. It is also
            the riskiest piece: repaint gates on l.stopped (screen.go:862), whose doc
            says painting must stop dead once set (screen.go:600-604), and Stop both
            flushes and latches (screen.go:841-852) — so a second boolean beside stopped
            declares four states of which three are legal, the constellation ARCH-ORDER
            exists to catch. Name it as one tagged state (running / suspended / stopped)
            and name the test with its adversarial strategy; the deterministic seam
            already exists, since interval is a field precisely so a test can hold the
            paint window open rather than race the clock (screen.go:637-640).
          family: new-seam-untested
          round: 2
        - id: PQ-6
          severity: Important
          title: the Done-when README row rests on a false claim about existing guards, and Step 8 names no section
          detail: |-
            Done-when says /play must appear in /help and "in the README's command list,
            which are already guarded by derived tests". /help is genuinely derived
            (runHelp lists c.cmds, command.go:152 and 252). The README half is not:
            TestDocsQuoteTheCommandList reads ../../atlas/define.md only
            (doc_sync_test.go:362), and neither cmd/define/README.md nor the root
            README.md has a command list — commands are prose paragraphs
            (cmd/define/README.md:716-768). doc_sync_test.go:376-380 states this repo's
            convention for that surface: the prose sites are swept by hand "and named as
            such in the plan rather than pretending a mechanism covers them". Task 2
            Step 8 is the bare item "README + atlas" and Verification lists no README
            check, so the sweep is neither guarded nor named. Name the section, or fix
            the Done-when row.
          family: hand-swept-surface-unnamed
          round: 2
        - id: PQ-7
          severity: Minor
          title: the two entry points would print different sentences for the same nil deck
          detail: |-
            This is the 2nd finding in family refusal-enumeration-incomplete. Per the
            escalation rule I am not asking you to fix this instance — state the rule.
            The rule: one refusal condition gets one sentence from one helper. The
            enumeration is measurable and small — six sites call noDeckMessage
            (main.go:1193, stats.go:43, stats.go:237, history_cmd.go:220, harvest.go:111,
            reflect.go:340) and exactly one hand-rolls it, runPlay at play_loop.go:30
            ("no deck in this directory, so there is nothing to review"). The plan has
            /play use noDeckMessage, so after this issue the same condition reads two
            ways depending on which entry point you took — against Done-when's "a change
            to the sitting cannot apply to only one of them". Either sweep play_loop.go:30
            into the helper in the same round, or state that noDeckMessage owns the cause
            and a caller may append its own consequence, and make both entry points obey it.
          family: refusal-enumeration-incomplete
          round: 2
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

## Round 2 — 2026-09-08T15:18:01-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — hand-back ownership is answered; where the summary lands is not, and the no-op restorer breaks handBack's cooked-terminal precondition
- PQ-2 — addressed — console gains newSitting, built in newConsole where sess and the real stdout are in scope; runEditor's signature untouched
- PQ-3 — addressed — nil deck named as a separate refusal reusing noDeckMessage(c.noCapture)
- PQ-4 — not-addressed — two-painter hazard handled; the extent clause — who is still running when sittingInPlace returns — is still unwritten

### Raised

- **PQ-5** [Important] `new-seam-untested` liveScreen suspend/resume is the plan's one new capability and it has no named test, no strategy line, and no state model
  The plan calls it "the smallest piece that cannot be avoided" yet it appears
  in neither the Pure entities nor the Integration points table, and the Test
  surface paragraph covers only runPlayCommand and sittingInPlace. It is also
  the riskiest piece: repaint gates on l.stopped (screen.go:862), whose doc
  says painting must stop dead once set (screen.go:600-604), and Stop both
  flushes and latches (screen.go:841-852) — so a second boolean beside stopped
  declares four states of which three are legal, the constellation ARCH-ORDER
  exists to catch. Name it as one tagged state (running / suspended / stopped)
  and name the test with its adversarial strategy; the deterministic seam
  already exists, since interval is a field precisely so a test can hold the
  paint window open rather than race the clock (screen.go:637-640).
- **PQ-6** [Important] `hand-swept-surface-unnamed` the Done-when README row rests on a false claim about existing guards, and Step 8 names no section
  Done-when says /play must appear in /help and "in the README's command list,
  which are already guarded by derived tests". /help is genuinely derived
  (runHelp lists c.cmds, command.go:152 and 252). The README half is not:
  TestDocsQuoteTheCommandList reads ../../atlas/define.md only
  (doc_sync_test.go:362), and neither cmd/define/README.md nor the root
  README.md has a command list — commands are prose paragraphs
  (cmd/define/README.md:716-768). doc_sync_test.go:376-380 states this repo's
  convention for that surface: the prose sites are swept by hand "and named as
  such in the plan rather than pretending a mechanism covers them". Task 2
  Step 8 is the bare item "README + atlas" and Verification lists no README
  check, so the sweep is neither guarded nor named. Name the section, or fix
  the Done-when row.
- **PQ-7** [Minor] `refusal-enumeration-incomplete` the two entry points would print different sentences for the same nil deck
  This is the 2nd finding in family refusal-enumeration-incomplete. Per the
  escalation rule I am not asking you to fix this instance — state the rule.
  The rule: one refusal condition gets one sentence from one helper. The
  enumeration is measurable and small — six sites call noDeckMessage
  (main.go:1193, stats.go:43, stats.go:237, history_cmd.go:220, harvest.go:111,
  reflect.go:340) and exactly one hand-rolls it, runPlay at play_loop.go:30
  ("no deck in this directory, so there is nothing to review"). The plan has
  /play use noDeckMessage, so after this issue the same condition reads two
  ways depending on which entry point you took — against Done-when's "a change
  to the sitting cannot apply to only one of them". Either sweep play_loop.go:30
  into the helper in the same round, or state that noDeckMessage owns the cause
  and a caller may append its own consequence, and make both entry points obey it.

## Open findings

- **PQ-1** [Critical] `terminal-ownership-unstated` newConsole's finish restores the shared rawSession, so a sitting started from the loop hands the terminal back mid-REPL
- **PQ-4** [Important] `unstated-goroutine-extent` the ARCH-ORDER "no concurrency" N/A is wrong — newConsole starts a resize watcher and each screen carries a paint timer
- **PQ-5** [Important] `new-seam-untested` liveScreen suspend/resume is the plan's one new capability and it has no named test, no strategy line, and no state model
- **PQ-6** [Important] `hand-swept-surface-unnamed` the Done-when README row rests on a false claim about existing guards, and Step 8 names no section
- **PQ-7** [Minor] `refusal-enumeration-incomplete` the two entry points would print different sentences for the same nil deck
