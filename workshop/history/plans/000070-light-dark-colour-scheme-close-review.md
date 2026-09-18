# Boundary Review — tools#70 (whole-issue close)

| field | value |
|-------|-------|
| issue | 70 — define: switch between a light and a dark colour scheme |
| repo | tools |
| issue file | workshop/issues/000070-light-dark-colour-scheme.md |
| boundary | whole-issue close |
| milestone | — |
| window | 75370a2905a22a70f2935efec78e96d1f1b7c9e5..5d6ebbe34f713ff7574735cd1f814cfd05414f09 |
| command | sdlc close --issue 70 |
| reviewer | claude |
| timestamp | 2026-09-18T11:19:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All nine open findings were checked against the code at `5d6ebbe`. Eight are fixed. BR-16 is not: its fix went too far the other way. Its sentence is a one-line Minor that doesn't block.

On the code:
- There is one atomic scheme holder, and every production writer to it goes through `choose`, `forget` or `detect`.
- Every painter gets the shade from that holder when it paints. The screen frame, the exit transcript, `serializeOutput`, the answer writer and the practice writers all do.
- The terminal's reply is read by a decoder capped at 64 bytes that swallows it first and parses it second.
- The only two loops that read keys use one shared rule, `terminalReport`, to recognise a reply.

Test results:
- `go vet` is clean for both the default and the conformance builds.
- `go test ./cmd/define/...` passes except for 15 pty-backed subtests, which all fail with EPERM. This environment can't open a pty at all, even outside the sandbox: `os.openpty()` gives "Operation not permitted". So the pty tests were **not run here**. `TestLanguagePromptStartup`, which #70 didn't write, fails the same way, which points to the environment rather than the code. The issue Log records the full tagged suite run elsewhere.

**Strengths**
- The state shape is now right. `schemeState{choice *schemeChoice; detected store.Scheme}` (`cmd/define/scheme.go:57`) plus `choiceSource` means a choice can't claim to be detected or default. `schemeArg{}` means auto, and `loopKind` replaced the two booleans.
- Every consumer of `KeyBackground` is listed and each one is tested on what it outputs:
  - the editor's frame and the sitting's frame (`TestASittingRepaintsOnABackgroundReply`)
  - the editor's frame after a sitting ends
  - a drag in progress (`TestAReplyMidDragKeepsTheSelection`)
  - the full-channel drop, where `TestADroppedReplyIsSilent` checks no notice appears and the next real key still gets one.
- `decodeOSC` (`key.go:481`) is correct at every edge I traced:
  - it waits on a partial prefix
  - Ctrl-C, Enter, DEL, 0x80+ and a stray ESC abort in the same pass, and the input then decodes exactly as it did before #70
  - the cap holds when an ESC-backslash terminator would run past 64 bytes.
- `paintLanguageRow` now works out whether its ink is on from `filled && !coloured` instead of storing it. That's correct because `unfill()` runs before every change to `coloured`.

**Critical:** none.

**Important:** none.

**Minor**
- BR-16 again: `README.md:359` says "Every full-screen session asks the terminal". The code asks only when `wantsBackground` holds: `opt.tty && opt.color && opt.tintOn && !opt.raw` (`rawterm.go:162`). The full-screen editor opens whenever `terminalUI && opt.tty` (`repl.go:297`), so `define -language-tint off` and `define -raw` get a full-screen session that never asks.

**Test coverage notes**
- The pty tests (`TestPTYBackgroundDetection`, `TestPTYNoQueryWithoutATint`, `TestSavedSchemeGovernsALookup`, `TestLanguageTintInvocation`) could not run in this environment. Their in-process counterparts all pass.

**Architecture**
- **ARCH-DRY:** pass. There is one parser for the flag and the command, one SGR parse (`sourceColours`) and one report rule.
- **ARCH-PURE:** pass. `schemeState`, `applyScheme` (which takes a persister), `describeScheme`, `initialSchemeState`, `parseBackgroundColour` and `decodeOSC` are all pure.
- **ARCH-PURPOSE:** pass. Every consumer reads the shade from the one holder, and no painter hard-codes a shade in production.
- **ARCH-MOCK:** pass.
  - The persister has a stateful fake, and the store code is tested on temp directories.
  - The terminal is modelled in-process and through the pty.
  - The live check is half done: a real terminal read `dark (detected)`. The light-terminal check is owned by #77, which is on `origin/main` and cited in the atlas.
- **ARCH-CONSTRAINTS:** pass. The query is 8 bytes, nothing waits for the reply, the decoder holds at most 64 bytes, and it repaints only when the shade changes.
- **ARCH-SECURE:** pass.
  - The saved file is read with a 64-byte cap and parsed into the two-value enum.
  - Only absolute config directories count.
  - Test deps have no config directory, and the pty harness is isolated.
  - Only `realDeps()` tests read the config path, and they write nothing.
- **ARCH-ORDER:** pass. The immutable state is written only through transitions, there is a single writer, and the event-sequence tests cover a late reply and a duplicate reply.
- **ARCH-FUNERAL:** pass. `/scheme auto` removes the file and the directory if it is empty; nothing else is left behind.

**Plan revision recommendations:** none. The plan's Revisions already cover every shape change.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      schemeState.choice is *schemeChoice{value, by choiceSource}; choiceSource admits only flag/saved/session (scheme.go:30-58).
  - id: BR-3
    disposition: addressed
    note: |
      boardFooter and practiceChrome are gone; both tests call paintedBoardFooterForTest; the rationale now sits on boardFooterOutput (practice_output.go:155-189).
  - id: BR-4
    disposition: addressed
    note: |
      At close every source and transition the paragraph describes has shipped; atlas/define.md:512-531 matches scheme.go and screen.go.
  - id: BR-5
    disposition: addressed
    note: |
      The issue Log's M1 review entry states the renderers were moved and the fragment-tint assertions dropped, with the live halves ported; the Done-when evidence list cites that reconciliation.
  - id: BR-7
    disposition: addressed
    note: |
      commandCtx uses loopKind {oneShot, piped, editor} (command.go:257-267); schemeArg is one field whose empty value means auto (scheme.go:164-166).
  - id: BR-9
    disposition: addressed
    note: |
      Detection shipped in M3, so the atlas sentence about a loop asking the terminal is now true of loopEditor; no later-milestone claims are left.
  - id: BR-15
    disposition: addressed
    note: |
      inking is removed; unfill writes inkOff when !coloured, which is correct because unfill runs before every change to coloured (language_row.go:24-38).
  - id: BR-16
    disposition: not-addressed
    note: |
      Both named sentences are fixed, but the replacement at README.md:359 ("Every full-screen session asks") claims more than the code does: wantsBackground (rawterm.go:162) also requires colour on, -language-tint on and not -raw, while the editor opens on terminalUI && opt.tty alone (repl.go:297). So define -language-tint off and define -raw are full-screen sessions that never ask. Fix: state the predicate as the atlas does ("unless -language-tint off or -raw"), and change lessons.md "no narrower conditions than the code has" to "the code's condition exactly, neither narrower nor broader". Prevalence in the family: 4.
  - id: BR-17
    disposition: addressed
    note: |
      #77 ("confirm light detection in a real light terminal") is committed on origin/main (9b9f691), and atlas/define.md:555 cites it.
```
