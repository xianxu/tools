---
gate: boundary-review
issue: 41
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-31T16:09:53-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: D6 routes paging through toInput, which would teach the pure play package about a viewport
          detail: |-
            toInput returns play.Input (play_loop.go:216). The editor deliberately dispatches viewport
            gestures in the loop BEFORE Apply, documented at replraw.go:361-377 ("a viewport gesture never
            reaches Apply"). Handle them ahead of toInput and extract the four-case switch plus wheelLines
            into one helper both loops call, rather than a second copy of the policy. Also note D6's
            "nothing else moves" omits enterMouse, whose cost is that drag-select needs Option/Shift
            (rawterm.go:175-190, /help at command.go:267) — a review sitting now inherits that.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: viewport-gesture-layering
          round: 1
        - id: BR-2
          severity: Minor
          title: Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
          detail: |-
            The Done-when preamble requires every pin be a named test or a grep for a property, never
            "file X is unchanged", and row 7 is the one row that breaks its own rule. Grepping the
            message at play_loop.go:246 finds no test asserting the degraded path. T6 changes
            todaysQuestions' signature so that path must now hand the loop a usable empty progress
            rather than merely continuing locally, which is precisely when an uncovered branch is worth
            a test.
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: pin-without-predicate
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-31T16:09:53-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: --play takes the alternate screen without consulting opt.tty, so -no-color no longer disables cursor control
          detail: |-
            runPlay gates the full-screen surface on stdinIsTerminal alone, where repl uses
            `terminalUI := interactive && opt.tty` (repl.go:228) and falls back to replLines.
            playConsole then calls enterAlt/enterMouse and paints ANSI frames unconditionally,
            so `define --play -no-color` emits the escapes that flag exists to suppress
            (main.go:527) and `define --play > file` writes them into the file at a fabricated
            80 columns. Before this window --play emitted no escapes at all.
          family: terminal-ui-gate
          round: 2
        - id: BR-4
          severity: Important
          title: Narrowing the window mid-sitting clips option glosses — the same class as the operator's finding, swept only at the startup width
          detail: |-
            Measured: a form-2.3 prompt built by choiceFor at width 80 and painted at cols 40
            loses the tail of every option line. The resize case (play_loop.go:196-213)
            deliberately does not re-derive opt.width, which was true before the last commit
            made the wrap width a per-question fact. Same rule as the revision that closed the
            startup instance; this is the enumerable sibling (ARCH-PURPOSE).
          family: frame-clips-unwrapped-text
          round: 2
        - id: BR-5
          severity: Important
          title: TestDroppingAWordLowersTheCostTheBarShows passes only because slices.DeleteFunc aliases the caller's backing array
          detail: |-
            sittingDeck is passed to playSession by value and the test asserts on its own copy;
            it observes the drop solely through in-place compaction of the shared array.
            Rewriting dropped() to build a new slice — behaviour-identical for the loop — turns
            the test red. It also never inspects a drawn footer, so it does not pin the claim
            the finding is about.
          family: test-asserts-aliasing
          round: 2
        - id: BR-6
          severity: Important
          title: crlfWriter has no production caller after this window, and the diff adds three claims that it does
          detail: |-
            grep finds crlfWriter only in crlf.go and crlf_test.go. The new atlas prose says it
            "survives only on the piped path" (atlas/define.md:285-290 and :893-899), which is
            false; replraw.go:307-308 still says "--play keeps its own crlfWriter", the opposite
            of what D5a landed; askhighlight_test.go:245 repeats it. Third recurrence of this
            family per play_loop.go:459-462.
          family: doc-sweep-incomplete
          round: 2
        - id: BR-7
          severity: Important
          title: playConsole duplicates replRaw's console construction verbatim except for the screen constructor
          detail: |-
            play_loop.go:73-104 and replraw.go:32-70 differ in one token (newPinnedScreen vs
            newLiveScreen) across six statements. D1 committed to widening the shared seam
            rather than growing a parallel one, and the function's own comment asserts "Not a
            parallel construction". #40 will be the third caller (ARCH-DRY).
          family: parallel-construction
          round: 2
        - id: BR-8
          severity: Minor
          title: refresh() hand-copies two fields out of figures(), so a future field goes stale silently
          detail: |-
            play_loop.go:130-141 assigns only fig.load and fig.fresh from a fresh figures() call.
            Assigning the struct and re-applying total/done cannot rot.
          family: partial-copy-refresh
          round: 2
        - id: BR-9
          severity: Minor
          title: Done-when 7 (failed log read degrades to empty progress) has no test, contradicting the plan's own rule
          detail: |-
            Its pin is "the existing behaviour at play_loop.go:246, unchanged", which the
            Done-when header forbids. The behaviour also changed: the degraded progress now
            drives the bar and the summary, which previously re-read the log.
          family: unpinned-done-when
          round: 2
        - id: BR-10
          severity: Minor
          title: twiceNumberedOption ranges a map, so a tie picks the "guaranteed miss" at random
          detail: |-
            pty_conformance_test.go:905-915. If two digits both appear twice in the frame, the
            returned answer depends on map iteration order and the deliberate miss can become a
            correct answer. Pick the lowest matching digit or fail on a tie.
          family: nondeterministic-test-read
          round: 2
        - id: BR-11
          severity: Minor
          title: choiceFor wraps to opt.width (0 on a non-terminal or below 20 cols) while the frame clips at terminalCols (never 0)
          detail: |-
            The wrap width is a policy answer with a sentinel; the clip width is a measurement.
            Where they disagree the gloss arrives unwrapped into a frame that cuts it —
            TestUnwrappedWidthLeavesTheGlossAlone pins the sentinel, which is correct for the
            pipe and wrong for a sitting.
          family: frame-clips-unwrapped-text
          round: 2
        - id: BR-12
          severity: Minor
          title: --play -raw records nothing but the bar still applies the transition
          detail: |-
            decideCapture returns captureNothing under opt.raw (capture.go:30), so
            CaptureReview writes no event while held.answered still advances the in-memory
            progress. On that path D7's "cannot drift from what the next sitting derives" does
            not hold.
          family: figures-drift
          round: 2
        - id: BR-13
          severity: Minor
          title: TestTheBarCountsAnswersAsTheyLand slices on strings.Index without checking for -1
          detail: |-
            play_loop_test.go:558 — a bar format change panics the test instead of failing it
            with the message the assertion was written to give.
          family: test-fragility
          round: 2
        - id: BR-14
          severity: Minor
          title: sittingDeck has pointer-receiver mutators but is passed by value, sharing its map and half-sharing its slice
          detail: |-
            Nothing production depends on the aliasing today, but it is what makes I-3's test
            pass and it is a landmine for #40's second consumer. A *sittingDeck parameter
            removes the question.
          family: value-receiver-shared-state
          round: 2
      blocked: true
---

# Gate ledger — tools#41 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T16:09:53-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `viewport-gesture-layering` D6 routes paging through toInput, which would teach the pure play package about a viewport
  toInput returns play.Input (play_loop.go:216). The editor deliberately dispatches viewport
  gestures in the loop BEFORE Apply, documented at replraw.go:361-377 ("a viewport gesture never
  reaches Apply"). Handle them ahead of toInput and extract the four-case switch plus wheelLines
  into one helper both loops call, rather than a second copy of the policy. Also note D6's
  "nothing else moves" omits enterMouse, whose cost is that drag-select needs Option/Shift
  (rawterm.go:175-190, /help at command.go:267) — a review sitting now inherits that.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-2** [Minor] `pin-without-predicate` Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
  The Done-when preamble requires every pin be a named test or a grep for a property, never
  "file X is unchanged", and row 7 is the one row that breaks its own rule. Grepping the
  message at play_loop.go:246 finds no test asserting the degraded path. T6 changes
  todaysQuestions' signature so that path must now hand the loop a usable empty progress
  rather than merely continuing locally, which is precisely when an uncovered branch is worth
  a test.
  (carried from plan-quality PQ-10, deferred to the boundary review)

## Round 2 — 2026-08-31T16:09:53-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `terminal-ui-gate` --play takes the alternate screen without consulting opt.tty, so -no-color no longer disables cursor control
  runPlay gates the full-screen surface on stdinIsTerminal alone, where repl uses
  `terminalUI := interactive && opt.tty` (repl.go:228) and falls back to replLines.
  playConsole then calls enterAlt/enterMouse and paints ANSI frames unconditionally,
  so `define --play -no-color` emits the escapes that flag exists to suppress
  (main.go:527) and `define --play > file` writes them into the file at a fabricated
  80 columns. Before this window --play emitted no escapes at all.
- **BR-4** [Important] `frame-clips-unwrapped-text` Narrowing the window mid-sitting clips option glosses — the same class as the operator's finding, swept only at the startup width
  Measured: a form-2.3 prompt built by choiceFor at width 80 and painted at cols 40
  loses the tail of every option line. The resize case (play_loop.go:196-213)
  deliberately does not re-derive opt.width, which was true before the last commit
  made the wrap width a per-question fact. Same rule as the revision that closed the
  startup instance; this is the enumerable sibling (ARCH-PURPOSE).
- **BR-5** [Important] `test-asserts-aliasing` TestDroppingAWordLowersTheCostTheBarShows passes only because slices.DeleteFunc aliases the caller's backing array
  sittingDeck is passed to playSession by value and the test asserts on its own copy;
  it observes the drop solely through in-place compaction of the shared array.
  Rewriting dropped() to build a new slice — behaviour-identical for the loop — turns
  the test red. It also never inspects a drawn footer, so it does not pin the claim
  the finding is about.
- **BR-6** [Important] `doc-sweep-incomplete` crlfWriter has no production caller after this window, and the diff adds three claims that it does
  grep finds crlfWriter only in crlf.go and crlf_test.go. The new atlas prose says it
  "survives only on the piped path" (atlas/define.md:285-290 and :893-899), which is
  false; replraw.go:307-308 still says "--play keeps its own crlfWriter", the opposite
  of what D5a landed; askhighlight_test.go:245 repeats it. Third recurrence of this
  family per play_loop.go:459-462.
- **BR-7** [Important] `parallel-construction` playConsole duplicates replRaw's console construction verbatim except for the screen constructor
  play_loop.go:73-104 and replraw.go:32-70 differ in one token (newPinnedScreen vs
  newLiveScreen) across six statements. D1 committed to widening the shared seam
  rather than growing a parallel one, and the function's own comment asserts "Not a
  parallel construction". #40 will be the third caller (ARCH-DRY).
- **BR-8** [Minor] `partial-copy-refresh` refresh() hand-copies two fields out of figures(), so a future field goes stale silently
  play_loop.go:130-141 assigns only fig.load and fig.fresh from a fresh figures() call.
  Assigning the struct and re-applying total/done cannot rot.
- **BR-9** [Minor] `unpinned-done-when` Done-when 7 (failed log read degrades to empty progress) has no test, contradicting the plan's own rule
  Its pin is "the existing behaviour at play_loop.go:246, unchanged", which the
  Done-when header forbids. The behaviour also changed: the degraded progress now
  drives the bar and the summary, which previously re-read the log.
- **BR-10** [Minor] `nondeterministic-test-read` twiceNumberedOption ranges a map, so a tie picks the "guaranteed miss" at random
  pty_conformance_test.go:905-915. If two digits both appear twice in the frame, the
  returned answer depends on map iteration order and the deliberate miss can become a
  correct answer. Pick the lowest matching digit or fail on a tie.
- **BR-11** [Minor] `frame-clips-unwrapped-text` choiceFor wraps to opt.width (0 on a non-terminal or below 20 cols) while the frame clips at terminalCols (never 0)
  The wrap width is a policy answer with a sentinel; the clip width is a measurement.
  Where they disagree the gloss arrives unwrapped into a frame that cuts it —
  TestUnwrappedWidthLeavesTheGlossAlone pins the sentinel, which is correct for the
  pipe and wrong for a sitting.
- **BR-12** [Minor] `figures-drift` --play -raw records nothing but the bar still applies the transition
  decideCapture returns captureNothing under opt.raw (capture.go:30), so
  CaptureReview writes no event while held.answered still advances the in-memory
  progress. On that path D7's "cannot drift from what the next sitting derives" does
  not hold.
- **BR-13** [Minor] `test-fragility` TestTheBarCountsAnswersAsTheyLand slices on strings.Index without checking for -1
  play_loop_test.go:558 — a bar format change panics the test instead of failing it
  with the message the assertion was written to give.
- **BR-14** [Minor] `value-receiver-shared-state` sittingDeck has pointer-receiver mutators but is passed by value, sharing its map and half-sharing its slice
  Nothing production depends on the aliasing today, but it is what makes I-3's test
  pass and it is a landmine for #40's second consumer. A *sittingDeck parameter
  removes the question.

## Open findings

- **BR-1** [Important] `viewport-gesture-layering` D6 routes paging through toInput, which would teach the pure play package about a viewport
- **BR-2** [Minor] `pin-without-predicate` Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
- **BR-3** [Important] `terminal-ui-gate` --play takes the alternate screen without consulting opt.tty, so -no-color no longer disables cursor control
- **BR-4** [Important] `frame-clips-unwrapped-text` Narrowing the window mid-sitting clips option glosses — the same class as the operator's finding, swept only at the startup width
- **BR-5** [Important] `test-asserts-aliasing` TestDroppingAWordLowersTheCostTheBarShows passes only because slices.DeleteFunc aliases the caller's backing array
- **BR-6** [Important] `doc-sweep-incomplete` crlfWriter has no production caller after this window, and the diff adds three claims that it does
- **BR-7** [Important] `parallel-construction` playConsole duplicates replRaw's console construction verbatim except for the screen constructor
- **BR-8** [Minor] `partial-copy-refresh` refresh() hand-copies two fields out of figures(), so a future field goes stale silently
- **BR-9** [Minor] `unpinned-done-when` Done-when 7 (failed log read degrades to empty progress) has no test, contradicting the plan's own rule
- **BR-10** [Minor] `nondeterministic-test-read` twiceNumberedOption ranges a map, so a tie picks the "guaranteed miss" at random
- **BR-11** [Minor] `frame-clips-unwrapped-text` choiceFor wraps to opt.width (0 on a non-terminal or below 20 cols) while the frame clips at terminalCols (never 0)
- **BR-12** [Minor] `figures-drift` --play -raw records nothing but the bar still applies the transition
- **BR-13** [Minor] `test-fragility` TestTheBarCountsAnswersAsTheyLand slices on strings.Index without checking for -1
- **BR-14** [Minor] `value-receiver-shared-state` sittingDeck has pointer-receiver mutators but is passed by value, sharing its map and half-sharing its slice
