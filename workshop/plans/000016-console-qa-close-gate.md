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
    - "n": 2
      timestamp: "2026-08-23T16:05:39-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: deleting either loop''s cmdAsk branch now reddens the matching forced test; hist.Add(submitted) is pinned too.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Asserted against an injected memHistory; reverting hist.Add(line) in submitLine reddens the unforced subtest.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README documents both hatches, routing, the model requirement and the exit codes; two of its claims are inexact — see the new doc-overstates-code finding.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Exhaustive dispatch at main.go:346; removing it reddens the test. The test's capture half is dead, raised separately under test-asserts-nothing.
          round: 2
        - id: BR-5
          disposition: addressed
          note: eraseLine present at replraw.go:223 and measured on the byte stream; unpinned by any test, folded into the family finding.
          round: 2
        - id: BR-6
          disposition: addressed
          note: 'Measured: forced and unforced asks both emit exactly one leading CRLF plus askInSession''s, identical tails; unpinned, folded into the family finding.'
          round: 2
        - id: BR-7
          disposition: addressed
          note: atlas/define.md now qualifies the sentence and names "why?" explicitly.
          round: 2
        - id: BR-8
          disposition: addressed
          note: question.go:62 uses slices.Contains; the hand-rolled contains() is gone.
          round: 2
      findings:
        - id: BR-9
          severity: Important
          title: '"-raw never asks" is guarded on the unforced route only, so 3 of 6 ask entry points ignore it'
          detail: |-
            This is the 2nd finding in family doc-overstates-code, so the deliverable
            is the rule, not the site. Rule - an absolute stated in README/atlas or in
            a test name must name the enumeration it quantifies over, and every cell
            must be guarded and asserted. Here the enumeration is
            {forced, unforced} x {one-shot, piped loop, raw editor}; the !opt.raw
            guard sits at main.go:425 on the unforced path only. Measured -
            `define -raw "?what is X"` and `echo "?what is X" | define -raw` both
            print the no-model message and exit 1, and runEditor with opt.raw=true on
            "?why" does the same, while TestRawNeverAsks (askroute_test.go:202) stays
            green through all three. README.md:79 states the absolute; main.go:419-424
            names the M2 cost (a network call for a line a script piped in). Same
            family, smaller prevalence - README.md:69 and :76 quote the miss message
            as `not found` where the program prints `no dictionary entry`.
          family: doc-overstates-code
          round: 2
        - id: BR-10
          severity: Important
          title: BR-4's test injects a countingCapturer that withStore discards, so its log-junk assertion is dead
          detail: |-
            This is the 2nd finding in family test-asserts-nothing, so state the rule -
            a test that injects a double must assert the injection took effect, or
            inject at the seam production reads. askroute_test.go:178 sets d.newStore
            on a deps whose capture testDeps already filled (main_test.go:14), and
            withStore only fills nils (main.go:96), so cap is discarded and
            `for _, w := range cap.calls` never iterates. Probe-verified - a panic
            inside countingCapturer.Capture leaves TestOneShotRejectsAHatchWithNothingAfterIt
            green while TestAQuestionIsNotCaptured panics at once. So the harm BR-4
            actually named, a ReviewEvent with an empty Word that complete() discards
            at read time, has no falsifiable test. Measured prevalence - 1 of 1
            d.newStore injection in the package is dead.
          family: test-asserts-nothing
          round: 2
        - id: BR-11
          severity: Minor
          title: neither BR-5's nor BR-6's fix is pinned - both can be reverted with the suite green
          detail: |-
            This is the 3rd finding in family raw-mode-message-placement. Do not
            re-fix the sites. Rule - every message class the raw loop writes gets an
            assertion on the emitted byte stream, as TestEditorLoopUsesCarriageReturnsInRawMode
            (editorloop_test.go:143) already does for the definition path. Verified -
            removing eraseLine from replraw.go:223 and re-adding askInSession's own
            "\r\n" together leave go test ./cmd/define/ fully green. Measured
            prevalence - 3 of 3 raw-mode message placements introduced by this issue
            (the bare-"?" note, the forced ask, the unforced ask) are unasserted,
            which is why the family recurs.
          family: raw-mode-message-placement
          round: 2
        - id: BR-12
          severity: Minor
          title: the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
          detail: |-
            Two instances of one rule - a behaviour specified for the unforced route
            is silently inherited by the forced one. (a) submitLine adds cmd.word
            (replraw.go:299), which is post-strip, so typing `\how so` records
            "how so"; Up-arrow then Enter asks the line you had just forced to a
            lookup. The cmdAsk branch records submitted.String() (replraw.go:207), so
            "?why" stays forced. Measured through runEditor with an injected
            memHistory. (b) askUnavailable (ask.go:18) tells a forced question about a
            real headword that it "is not a word" - "?why" prints
            "no model configured; `why` is not a word". The sentence is true by
            construction only on the unforced route.
          family: forced-route-enumeration
          round: 2
        - id: BR-13
          severity: Minor
          title: the plan's step checkboxes are 0 of 60 ticked while M1's tasks are complete and committed
          detail: |-
            workshop/plans/000016-console-qa-plan.md - Tasks 1-6 shipped in five
            commits, but no step box is ticked. The two most recent archived plans set
            the convention (000015 39/39, 000014 33/33; 000011 did not), so this is
            soft, but leaving M1's boxes open makes "where does M2 resume" ambiguous
            when the plan is picked up in a fresh session.
          family: plan-bookkeeping
          round: 2
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

## Round 2 — 2026-08-23T16:05:39-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: deleting either loop's cmdAsk branch now reddens the matching forced test; hist.Add(submitted) is pinned too.
- BR-2 — addressed — Asserted against an injected memHistory; reverting hist.Add(line) in submitLine reddens the unforced subtest.
- BR-3 — addressed — README documents both hatches, routing, the model requirement and the exit codes; two of its claims are inexact — see the new doc-overstates-code finding.
- BR-4 — addressed — Exhaustive dispatch at main.go:346; removing it reddens the test. The test's capture half is dead, raised separately under test-asserts-nothing.
- BR-5 — addressed — eraseLine present at replraw.go:223 and measured on the byte stream; unpinned by any test, folded into the family finding.
- BR-6 — addressed — Measured: forced and unforced asks both emit exactly one leading CRLF plus askInSession's, identical tails; unpinned, folded into the family finding.
- BR-7 — addressed — atlas/define.md now qualifies the sentence and names "why?" explicitly.
- BR-8 — addressed — question.go:62 uses slices.Contains; the hand-rolled contains() is gone.

### Raised

- **BR-9** [Important] `doc-overstates-code` "-raw never asks" is guarded on the unforced route only, so 3 of 6 ask entry points ignore it
  This is the 2nd finding in family doc-overstates-code, so the deliverable
  is the rule, not the site. Rule - an absolute stated in README/atlas or in
  a test name must name the enumeration it quantifies over, and every cell
  must be guarded and asserted. Here the enumeration is
  {forced, unforced} x {one-shot, piped loop, raw editor}; the !opt.raw
  guard sits at main.go:425 on the unforced path only. Measured -
  `define -raw "?what is X"` and `echo "?what is X" | define -raw` both
  print the no-model message and exit 1, and runEditor with opt.raw=true on
  "?why" does the same, while TestRawNeverAsks (askroute_test.go:202) stays
  green through all three. README.md:79 states the absolute; main.go:419-424
  names the M2 cost (a network call for a line a script piped in). Same
  family, smaller prevalence - README.md:69 and :76 quote the miss message
  as `not found` where the program prints `no dictionary entry`.
- **BR-10** [Important] `test-asserts-nothing` BR-4's test injects a countingCapturer that withStore discards, so its log-junk assertion is dead
  This is the 2nd finding in family test-asserts-nothing, so state the rule -
  a test that injects a double must assert the injection took effect, or
  inject at the seam production reads. askroute_test.go:178 sets d.newStore
  on a deps whose capture testDeps already filled (main_test.go:14), and
  withStore only fills nils (main.go:96), so cap is discarded and
  `for _, w := range cap.calls` never iterates. Probe-verified - a panic
  inside countingCapturer.Capture leaves TestOneShotRejectsAHatchWithNothingAfterIt
  green while TestAQuestionIsNotCaptured panics at once. So the harm BR-4
  actually named, a ReviewEvent with an empty Word that complete() discards
  at read time, has no falsifiable test. Measured prevalence - 1 of 1
  d.newStore injection in the package is dead.
- **BR-11** [Minor] `raw-mode-message-placement` neither BR-5's nor BR-6's fix is pinned - both can be reverted with the suite green
  This is the 3rd finding in family raw-mode-message-placement. Do not
  re-fix the sites. Rule - every message class the raw loop writes gets an
  assertion on the emitted byte stream, as TestEditorLoopUsesCarriageReturnsInRawMode
  (editorloop_test.go:143) already does for the definition path. Verified -
  removing eraseLine from replraw.go:223 and re-adding askInSession's own
  "\r\n" together leave go test ./cmd/define/ fully green. Measured
  prevalence - 3 of 3 raw-mode message placements introduced by this issue
  (the bare-"?" note, the forced ask, the unforced ask) are unasserted,
  which is why the family recurs.
- **BR-12** [Minor] `forced-route-enumeration` the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
  Two instances of one rule - a behaviour specified for the unforced route
  is silently inherited by the forced one. (a) submitLine adds cmd.word
  (replraw.go:299), which is post-strip, so typing `\how so` records
  "how so"; Up-arrow then Enter asks the line you had just forced to a
  lookup. The cmdAsk branch records submitted.String() (replraw.go:207), so
  "?why" stays forced. Measured through runEditor with an injected
  memHistory. (b) askUnavailable (ask.go:18) tells a forced question about a
  real headword that it "is not a word" - "?why" prints
  "no model configured; `why` is not a word". The sentence is true by
  construction only on the unforced route.
- **BR-13** [Minor] `plan-bookkeeping` the plan's step checkboxes are 0 of 60 ticked while M1's tasks are complete and committed
  workshop/plans/000016-console-qa-plan.md - Tasks 1-6 shipped in five
  commits, but no step box is ticked. The two most recent archived plans set
  the convention (000015 39/39, 000014 33/33; 000011 did not), so this is
  soft, but leaving M1's boxes open makes "where does M2 resume" ambiguous
  when the plan is picked up in a fresh session.

## Open findings

- **BR-9** [Important] `doc-overstates-code` "-raw never asks" is guarded on the unforced route only, so 3 of 6 ask entry points ignore it
- **BR-10** [Important] `test-asserts-nothing` BR-4's test injects a countingCapturer that withStore discards, so its log-junk assertion is dead
- **BR-11** [Minor] `raw-mode-message-placement` neither BR-5's nor BR-6's fix is pinned - both can be reverted with the suite green
- **BR-12** [Minor] `forced-route-enumeration` the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
- **BR-13** [Minor] `plan-bookkeeping` the plan's step checkboxes are 0 of 60 ticked while M1's tasks are complete and committed
