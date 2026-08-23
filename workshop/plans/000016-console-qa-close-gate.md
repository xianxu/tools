---
gate: boundary-review
issue: 16
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-23T15:46:30-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: deleting both loops' forced-"?" cmdAsk branches leaves the whole suite green
          detail: |-
            Verified by mutation: removing `case cmdAsk:` from replLines (repl.go:219)
            and the whole `if cmd.kind == cmdAsk` block from runEditor
            (replraw.go:199-211) passes `go test ./cmd/define/` in full. Every
            loop-level ask test uses the unforced route; routeFor returns "question"
            for cmdAsk without entering a loop. hist.Add(submitted.String()) at
            replraw.go:204 is unpinned by the same gap. The issue's ticked Done-when
            row claims each hatch is exercised by a test that fails when the hatch is
            removed; that holds at the parser only. Add one forced test per loop —
            both pass against today's code, which is the point.
          family: loop-shell-branch-untested
          round: 1
        - id: BR-2
          severity: Important
          title: TestAQuestionIsRecalledByUpArrow passes with history recording removed
          detail: |-
            askroute_test.go:95 asserts stdout contains the question, but the raw
            editor re-renders the line on every keystroke, so the question is in
            stdout from typing alone. Verified: replacing hist.Add(line) at
            replraw.go:292 with a no-op leaves the test green. Task 4's contract row
            "does add the line to editor recall (hist.Add)" has no falsifiable test.
            Assert against an injected History, and pin the recall rendering by
            comparing runs with and without the KeyUp.
          family: test-asserts-nothing
          round: 1
        - id: BR-3
          severity: Important
          title: README update appears missing for the "?" and "\" hatches and question routing
          detail: |-
            README.md documents exactly this class — the "/"-in-column-1 convention
            (README.md:120), the interactive key table (:43-50), and the exit-code
            contract (:111-116) — and is untouched in the window, while atlas,
            fs.Usage and /help were all updated. New user-facing surface a reader
            types: "?" forces a question, "\" forces a lookup, an unrecognised line
            that reads as a question goes to the model, and a question with no model
            configured exits 1.
          family: readme-surface-gate
          round: 1
        - id: BR-4
          severity: Important
          title: the one-shot sends cmdNothing to the dictionary as an empty word
          detail: |-
            main.go:342's default arm handles cmdCommand and cmdAsk and falls through
            to defineOnce for every other kind, but parseREPLLine now returns
            cmdNothing with word=="" for two inputs it never used to produce. Measured:
            `define "?"` and `define "\"` print `define: : no dictionary entry`, exit
            1, and Capture("", false, opt) appends a ReviewEvent{Word: ""} that
            complete() (store/event.go:30) discards at read time — junk in the
            append-only log #8 and #17 fold over. cmd.note is dropped, so the one-shot
            never shows `type a question after "?"`. Handle cmdNothing before
            defineOnce and never pass an empty word to lookupAndRender.
          family: oneshot-kind-coverage
          round: 1
        - id: BR-5
          severity: Minor
          title: the bare-"?" note is printed without eraseLine, unlike its sibling branch
          detail: |-
            replraw.go:215 writes `define: %s\r\n` with the cursor still on the typed
            prompt line, so it renders as `› ?define: type a question after "?"`.
            replayInPlace (replraw.go:264) prefixes eraseLine for the same class of
            message.
          family: raw-mode-message-placement
          round: 1
        - id: BR-6
          severity: Minor
          title: forced and unforced asks render with different vertical spacing
          detail: |-
            The unforced path emits runEditor's pre-submitLine "\r\n" (replraw.go:227)
            in addition to askInSession's own, so its message sits one blank line
            lower than the forced path's. Two routes the plan wanted visually
            identical.
          family: raw-mode-message-placement
          round: 1
        - id: BR-7
          severity: Minor
          title: atlas says single-word lines are never questions; the trailing-"?" arm says otherwise
          detail: |-
            atlas/define.md:488. The "?" arm precedes the len(fields) < 2 check, so
            readsAsQuestion("why?") and readsAsQuestion("sycophanti?") both return
            true (measured). Qualify the sentence or move the arm.
          family: doc-overstates-code
          round: 1
        - id: BR-8
          severity: Minor
          title: hand-rolled contains() where slices.Contains exists
          detail: |-
            question.go:83. The module is go 1.26 and the package's tests already
            import slices (ARCH-DRY).
          family: stdlib-reuse
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — tools#16 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-23T15:46:30-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `loop-shell-branch-untested` deleting both loops' forced-"?" cmdAsk branches leaves the whole suite green
  Verified by mutation: removing `case cmdAsk:` from replLines (repl.go:219)
  and the whole `if cmd.kind == cmdAsk` block from runEditor
  (replraw.go:199-211) passes `go test ./cmd/define/` in full. Every
  loop-level ask test uses the unforced route; routeFor returns "question"
  for cmdAsk without entering a loop. hist.Add(submitted.String()) at
  replraw.go:204 is unpinned by the same gap. The issue's ticked Done-when
  row claims each hatch is exercised by a test that fails when the hatch is
  removed; that holds at the parser only. Add one forced test per loop —
  both pass against today's code, which is the point.
- **BR-2** [Important] `test-asserts-nothing` TestAQuestionIsRecalledByUpArrow passes with history recording removed
  askroute_test.go:95 asserts stdout contains the question, but the raw
  editor re-renders the line on every keystroke, so the question is in
  stdout from typing alone. Verified: replacing hist.Add(line) at
  replraw.go:292 with a no-op leaves the test green. Task 4's contract row
  "does add the line to editor recall (hist.Add)" has no falsifiable test.
  Assert against an injected History, and pin the recall rendering by
  comparing runs with and without the KeyUp.
- **BR-3** [Important] `readme-surface-gate` README update appears missing for the "?" and "\" hatches and question routing
  README.md documents exactly this class — the "/"-in-column-1 convention
  (README.md:120), the interactive key table (:43-50), and the exit-code
  contract (:111-116) — and is untouched in the window, while atlas,
  fs.Usage and /help were all updated. New user-facing surface a reader
  types: "?" forces a question, "\" forces a lookup, an unrecognised line
  that reads as a question goes to the model, and a question with no model
  configured exits 1.
- **BR-4** [Important] `oneshot-kind-coverage` the one-shot sends cmdNothing to the dictionary as an empty word
  main.go:342's default arm handles cmdCommand and cmdAsk and falls through
  to defineOnce for every other kind, but parseREPLLine now returns
  cmdNothing with word=="" for two inputs it never used to produce. Measured:
  `define "?"` and `define "\"` print `define: : no dictionary entry`, exit
  1, and Capture("", false, opt) appends a ReviewEvent{Word: ""} that
  complete() (store/event.go:30) discards at read time — junk in the
  append-only log #8 and #17 fold over. cmd.note is dropped, so the one-shot
  never shows `type a question after "?"`. Handle cmdNothing before
  defineOnce and never pass an empty word to lookupAndRender.
- **BR-5** [Minor] `raw-mode-message-placement` the bare-"?" note is printed without eraseLine, unlike its sibling branch
  replraw.go:215 writes `define: %s\r\n` with the cursor still on the typed
  prompt line, so it renders as `› ?define: type a question after "?"`.
  replayInPlace (replraw.go:264) prefixes eraseLine for the same class of
  message.
- **BR-6** [Minor] `raw-mode-message-placement` forced and unforced asks render with different vertical spacing
  The unforced path emits runEditor's pre-submitLine "\r\n" (replraw.go:227)
  in addition to askInSession's own, so its message sits one blank line
  lower than the forced path's. Two routes the plan wanted visually
  identical.
- **BR-7** [Minor] `doc-overstates-code` atlas says single-word lines are never questions; the trailing-"?" arm says otherwise
  atlas/define.md:488. The "?" arm precedes the len(fields) < 2 check, so
  readsAsQuestion("why?") and readsAsQuestion("sycophanti?") both return
  true (measured). Qualify the sentence or move the arm.
- **BR-8** [Minor] `stdlib-reuse` hand-rolled contains() where slices.Contains exists
  question.go:83. The module is go 1.26 and the package's tests already
  import slices (ARCH-DRY).

## Open findings

- **BR-1** [Important] `loop-shell-branch-untested` deleting both loops' forced-"?" cmdAsk branches leaves the whole suite green
- **BR-2** [Important] `test-asserts-nothing` TestAQuestionIsRecalledByUpArrow passes with history recording removed
- **BR-3** [Important] `readme-surface-gate` README update appears missing for the "?" and "\" hatches and question routing
- **BR-4** [Important] `oneshot-kind-coverage` the one-shot sends cmdNothing to the dictionary as an empty word
- **BR-5** [Minor] `raw-mode-message-placement` the bare-"?" note is printed without eraseLine, unlike its sibling branch
- **BR-6** [Minor] `raw-mode-message-placement` forced and unforced asks render with different vertical spacing
- **BR-7** [Minor] `doc-overstates-code` atlas says single-word lines are never questions; the trailing-"?" arm says otherwise
- **BR-8** [Minor] `stdlib-reuse` hand-rolled contains() where slices.Contains exists
