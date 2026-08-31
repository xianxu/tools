# Boundary Review — tools#41 (whole-issue close)

| field | value |
|-------|-------|
| issue | 41 — play mode paints frames through screen, and gains a status bar |
| repo | tools |
| issue file | workshop/issues/000041-play-tui.md |
| boundary | whole-issue close |
| milestone | — |
| window | 94f3ad077210c544fe60b8539e2a92def29333c1..d08454bae0eeb76ab1931df6334d2f2947c2a31b |
| command | sdlc close --issue 41 |
| reviewer | claude |
| timestamp | 2026-08-31T16:09:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

`--play` genuinely became the second consumer of `#30`'s `screen`/`console` seam: the question is a buffer line written once per transition, the grading keys are the frame's prompt, the bar is the footer, paging is intercepted in the loop so `play` stays pure, and the playback restore/re-enter dance is gone. I mutation-verified the three load-bearing new behaviours (pinned padding, write-once, in-memory drop) and all reddened correctly; `go test ./...` and `go vet` are green here, and every Done-when row except #7 names a test that exists. What holds up SHIP is a class the last commit opened and closed only at one end — the operator's "a frame clips an unwrapped line" bug is still reachable by narrowing the window mid-sitting (measured, below) — plus a new gate regression: `runPlay` takes the alternate screen and paints ANSI without consulting `opt.tty`, so `-no-color` no longer disables cursor control on the sitting path, which is exactly the "cursor control gated on stdin" family `repl.go:216-228` records as having already appeared three times. Both fixes are small.

I could not run the pty conformance rows: this environment refuses pty allocation (`no pty available: operation not permitted`), so all six `TestPTYPlay*` rows SKIP. D5a's Critical and the sitting's SIGWINCH are pinned **only** there — I verified the code paths by reading, not by measurement.

### 1. Strengths

- **D5a resolved by deletion, and the deletion is complete.** `play_loop.go:319-352` — the reveal no longer calls `restore()`/`enterRaw`, the "lost the terminal after playback" branch is gone with it, and `rawTerm` is gone from the tree entirely. `finish()` (`play_loop.go:512`) lost its failure branches with the reads. That is a Critical closed by removing code.
- **The write-once model is real, not asserted.** `written` sits beside "perform the outcomes" (`play_loop.go:148`), and forcing the write every frame makes `TestRepeatedKeystrokesDoNotDuplicateTheQuestion` report `8 times after 7 keystrokes`. Verified by mutation.
- **`newPinnedScreen` vs `newLiveScreen` is the right shape, and both directions are pinned.** Disabling `if s.pinned` reddens three tests, and `TestTheEditorsFooterFollowsItsContent` guards the REPL from the padding. Verified by mutation.
- **The IO claim is pinned by a counting store rather than a comment.** `TestASittingReadsTheDeckOnce` splits `decks`/`events` (`history_store_test.go:60`) so one extra deck read cannot hide behind one fewer log read — and `finish` genuinely stopped reading.
- **`schedule.GradeOf` is the honest version of the DRY claim.** `gradeOf` is now one line over it (`progress.go:190`), so the bar's transition and `Fold`'s are the same function rather than two that agree today.
- **`TestAMissRecordsBeforeItPlays` re-homes an invariant instead of porting an assertion** — one ordered log written by both the capturer and the player, which still discriminates once the abort branch it used to ride on is deleted.

### 2. Critical findings

None.

### 3. Important findings

**I-1 — `runPlay`/`playConsole` gate the full-screen surface on stdin alone; `-no-color` no longer disables cursor control.** `cmd/define/play_loop.go:41-64`, `:73-77`.
`repl` computes `terminalUI := interactive && opt.tty` (`repl.go:228`) and falls back to `replLines` when it is false, with a comment recording that this exact family has already shipped three times. `runPlay` checks only `stdinIsTerminal`, then `playConsole` calls `sess.enterAlt()` and `sess.enterMouse()` unconditionally and paints `\x1b[H\x1b[J` frames. `opt.tty` is `!*noColor && isTerminal(stdout)` (`main.go:527`), whose own comment says "-no-color means 'emit no ANSI', so it disables cursor control too — the flag exists for terminals that mangle escapes". Before this window `--play` emitted no escapes at all (`git show 94f3ad0:cmd/define/play_loop.go` — `enterRaw` only touches termios). So `define --play -no-color` on such a terminal now prints the alt-screen, mouse-report and cursor sequences as literal garbage, and `define --play > file` writes them into the file at a fabricated 80 columns.
*Fix sketch:* compute the same predicate in `runPlay` and refuse (or degrade) when `!opt.tty` — e.g. alongside the existing `--play needs a terminal` guard, since a sitting the learner cannot see is not a sitting.

**I-2 — the "a frame clips unwrapped text" class was swept only at its startup instance; narrowing the window mid-sitting still cuts the options.** `cmd/define/play_loop.go:196-213`, `cmd/define/optionpool.go:260-263`.
Measured, not reasoned: rendering `quokka`'s form-2.3 prompt through `choiceFor(..., width: 80)` and then painting the same screen at `cols: 40` yields

```
1  a small short-tailed wallaby with a s
   tree-climbing ability, native to West
2  an isolated flat-topped hill with ste
3  a bird with a short hooked bill and b
```

— every option cut, i.e. the operator's screenshot symptom, reachable by resizing. The resize case deliberately does not re-derive `opt.width` ("re-deriving it would change nothing and would claim a re-wrap this loop does not do"), which was true before the last commit made the wrap width a per-question fact. The plan's own revision states the rule ("a frame CLIPS… an unwrapped gloss stopped being ugly and started being missing"); this is the enumerable sibling of the site it names (ARCH-PURPOSE).
*Fix sketch:* either re-wrap the sitting's questions on resize (`choiceFor` already takes the width; the queue is in hand), or state in the resize case why a narrowing sitting is allowed to lose option text and pin the decision with a test.

**I-3 — `TestDroppingAWordLowersTheCostTheBarShows` passes on an aliasing artifact and never observes the bar.** `cmd/define/play_loop_test.go:578-590`.
`held` is passed to `playSession` by value; the test then asserts on *its own* copy. It only sees the drop because `slices.DeleteFunc` compacts the shared backing array in place. Replacing `dropped`'s body with a behaviour-equivalent copy-into-new-slice turns the test red (`the deck cost 4.000 before the drop and 4.000 after`) while the loop's own deck is still correct — so the test pins an implementation detail, not the claim. It also never looks at a drawn footer, which is the thing the finding is about.
*Fix sketch:* assert on `view.menus` before/after the drop the way `TestTheBarCountsAnswersAsTheyLand` does, so the pin is the bar's text rather than the caller's slice header.

**I-4 — `crlfWriter` is now dead production code, and this window adds three claims that it isn't.** `cmd/define/crlf.go:15`, `atlas/define.md:285-290` and `:893-899`, `cmd/define/replraw.go:307-308`, `cmd/define/askhighlight_test.go:245`.
`grep -rn crlfWriter cmd/define/` finds no production instantiation at all after `#41` — only `crlf_test.go`. The new atlas prose says it "survives only on the piped path" and "the writer that motivated the rule is now only on the piped path"; neither is true (`replLines` runs cooked and needs no translation). `replraw.go:307` still says "`--play` keeps its own crlfWriter, because it keeps drawing its own frames (D5a)", the opposite of what D5a landed. This is the `doc-sweep-incomplete` family `play_loop.go:459-462` already records three instances of.
*Fix sketch:* delete `crlf.go` + `crlf_test.go` (or say plainly it is retained unused), and correct the two atlas paragraphs and the two comments.

**I-5 — `playConsole` is a verbatim second copy of `replRaw`'s console construction (ARCH-DRY).** `cmd/define/play_loop.go:73-104` vs `cmd/define/replraw.go:32-70`.
Six statements — `enterAlt`, `enterMouse`, the screen, `watchResize` with the same closure, `onceHandBack`, the `console` literal — differ in exactly one token (`newPinnedScreen` vs `newLiveScreen`). D1 committed to the opposite: "If anything in it turns out not to fit, the honest move is to widen the shared seam rather than grow a parallel one — a second way to draw is the thing this issue exists to remove, not to add." The comment on `playConsole` asserts "Not a parallel construction", which the diff does not support. `#40`'s board will be the third caller.
*Fix sketch:* one `newConsole(ctx, d, sess, stdout, mk func(io.Writer, int, int) *liveScreen) console`, called by both loops with their constructor.

### 4. Minor findings

- `refresh()` (`play_loop.go:130-141`) hand-copies only `load` and `fresh` out of a fresh `figures()`. A field added to `sittingFigures`/`figures` later goes silently stale in the bar; assigning the struct and re-applying `total`/`done` is one line and cannot rot.
- Done-when 7 ("a failed log read degrades to empty progress") is pinned by "the existing behaviour … unchanged", which the plan's own header forbids ("never 'file X is unchanged'"). No test covers `Events` failing on the play path, and the consequence changed this window: the failure now also drives the bar and the summary, which previously re-read.
- `twiceNumberedOption` (`pty_conformance_test.go:905-915`) ranges a map and returns the first digit seen twice; if two digits tie, the "guaranteed miss" is chosen by map iteration order. Pick the lowest matching digit, or fail on a tie.
- `choiceFor` wraps to `opt.width` (a *policy* width — 0 when stdout is not a terminal or `cols < 20`) while the frame clips at `terminalCols` (never 0). `TestUnwrappedWidthLeavesTheGlossAlone` pins the sentinel behaviour, which is right for the pipe and wrong for a frame. Same root as I-1's gate.
- `--play -raw` is `captureNothing` (`capture.go:30`), so no event is written, but `held.answered` still applies the transition — on that one path D7's "cannot drift from what the next sitting derives" does not hold.
- `TestTheBarCountsAnswersAsTheyLand` slices on `strings.Index(footer[0], " ·")` without checking for `-1`; a bar format change panics the test instead of failing it.
- `sittingDeck` has pointer-receiver mutators but is passed by value, so its `prog` map is shared and its `deck` slice half-shared. Nothing depends on that today (I-3 is the only observer), but it is a landmine for `#40`'s second consumer; a `*sittingDeck` parameter would remove the question.

### 5. Test coverage notes

- Mutation-verified this round: `s.pinned` padding (3 tests red), `written != s.Index` (1 red), `dropped` no-op (1 red).
- Not verifiable here: all six `TestPTYPlay*` rows SKIP (`no pty available: operation not permitted`), including the two new ones. `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` is the *only* pin for D5a's Critical, and `playConsole` has no in-process test — a `&rawSession{control: &strings.Builder{}}` plus a buffer would let one assert "the alt-screen sequence is written once and no `restore` follows a reveal" without a pty, which matters because that Critical is the issue's headline risk.
- Nothing asserts mouse reporting is given back on the `--play` path (the editor has `#30` M1.4b's row); `restore()` does it, but the sitting's exit order is now its own code path.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** I-5 (parallel console construction) and I-4 (dead `crlfWriter` with three live doc claims).
- **ARCH-PURE — pass.** `sittingBar`/`costPhrase`/`sittingFigures`, `sittingDeck.{answered,dropped,figures}`, `livePrompt`, `schedule.GradeOf` are all pure and unit-tested with no IO; the `console`/`display` seam is injected and `playSession` is drivable with a scripted key channel and a recorder. `sittingFigures` taking plain numbers rather than a deck is the right cut.
- **ARCH-PURPOSE — flag.** The issue's purpose is delivered (frames, pinned bar, paging, transcript, SIGWINCH, pty). The gap is the *class* behind the last commit: I-2 is the same rule ("a frame clips what the terminal used to wrap") at the resize end, and the Minor above is the same rule at the `width == 0` sentinel end.
- **ARCH-MOCK — pass, with a standing weakness.** No new external dependency; the `display` seam has a stateful recorder that both loops share, and the real-terminal behaviour lives in a pty suite. The weakness is structural rather than introduced here: `conformance.SkipOrFail` means the rows that pin the most dangerous behaviour in this program vanish on any machine without a pty, as they did for this review.
- **ARCH-CONSTRAINTS — pass.** The per-answer refresh is O(deck) in memory and pinned by the counting store; the 16ms throttle is inherited, not re-tuned; paint is O(visible rows) and the pinned padding adds at most `termRows` CRLFs. One note: `figures()` walks the deck twice per refresh (`DailyLoad`, then `SustainableNewWords` → `DailyLoad`), which is free at deck sizes that matter but is worth knowing before `#40` calls it per frame.

### 7. Plan revision recommendations

- **`workshop/plans/000041-play-tui-plan.md`, D1 / the `playConsole` integration row** — record that `playConsole` is a *copy* of `replRaw`'s construction, not a reuse of it, and either commit to extracting the shared builder or state why two copies are acceptable with `#40` inbound. As written the plan claims something the tree does not do (I-5).
- **Same file, the 2026-08-31 "a frame CLIPS" revision** — extend the rule to the enumeration it implies: the sites where an unwrapped line meets a clipping frame are (a) the startup width, (b) a narrowing resize, (c) `opt.width == 0`. (a) is fixed; (b) and (c) are open (I-2 and its Minor sibling). This is the plan's own "ENUMERATING them, not listing the ones already in mind" rule applied to itself.
- **Same file, Done-when row 7** — replace "the existing behaviour at `play_loop.go:246`, unchanged" with a named test, per the section's own header. The behaviour's blast radius grew this window (the bar and the summary now consume the degraded progress), so "unchanged" is no longer accurate either.
- **Same file, ARCH-CONSTRAINTS** — the envelope table does not cover the non-terminal / `-no-color` environment, which is where I-1 lands. A row for "stdout is not a terminal, or the user asked for no ANSI" would have caught it at plan time, as `repl`'s `terminalUI` did for the editor.

```findings
findings:
  - id: new
    severity: Important
    family: terminal-ui-gate
    title: |
      --play takes the alternate screen without consulting opt.tty, so -no-color no longer disables cursor control
    detail: |
      runPlay gates the full-screen surface on stdinIsTerminal alone, where repl uses
      `terminalUI := interactive && opt.tty` (repl.go:228) and falls back to replLines.
      playConsole then calls enterAlt/enterMouse and paints ANSI frames unconditionally,
      so `define --play -no-color` emits the escapes that flag exists to suppress
      (main.go:527) and `define --play > file` writes them into the file at a fabricated
      80 columns. Before this window --play emitted no escapes at all.
  - id: new
    severity: Important
    family: frame-clips-unwrapped-text
    title: |
      Narrowing the window mid-sitting clips option glosses — the same class as the operator's finding, swept only at the startup width
    detail: |
      Measured: a form-2.3 prompt built by choiceFor at width 80 and painted at cols 40
      loses the tail of every option line. The resize case (play_loop.go:196-213)
      deliberately does not re-derive opt.width, which was true before the last commit
      made the wrap width a per-question fact. Same rule as the revision that closed the
      startup instance; this is the enumerable sibling (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: test-asserts-aliasing
    title: |
      TestDroppingAWordLowersTheCostTheBarShows passes only because slices.DeleteFunc aliases the caller's backing array
    detail: |
      sittingDeck is passed to playSession by value and the test asserts on its own copy;
      it observes the drop solely through in-place compaction of the shared array.
      Rewriting dropped() to build a new slice — behaviour-identical for the loop — turns
      the test red. It also never inspects a drawn footer, so it does not pin the claim
      the finding is about.
  - id: new
    severity: Important
    family: doc-sweep-incomplete
    title: |
      crlfWriter has no production caller after this window, and the diff adds three claims that it does
    detail: |
      grep finds crlfWriter only in crlf.go and crlf_test.go. The new atlas prose says it
      "survives only on the piped path" (atlas/define.md:285-290 and :893-899), which is
      false; replraw.go:307-308 still says "--play keeps its own crlfWriter", the opposite
      of what D5a landed; askhighlight_test.go:245 repeats it. Third recurrence of this
      family per play_loop.go:459-462.
  - id: new
    severity: Important
    family: parallel-construction
    title: |
      playConsole duplicates replRaw's console construction verbatim except for the screen constructor
    detail: |
      play_loop.go:73-104 and replraw.go:32-70 differ in one token (newPinnedScreen vs
      newLiveScreen) across six statements. D1 committed to widening the shared seam
      rather than growing a parallel one, and the function's own comment asserts "Not a
      parallel construction". #40 will be the third caller (ARCH-DRY).
  - id: new
    severity: Minor
    family: partial-copy-refresh
    title: |
      refresh() hand-copies two fields out of figures(), so a future field goes stale silently
    detail: |
      play_loop.go:130-141 assigns only fig.load and fig.fresh from a fresh figures() call.
      Assigning the struct and re-applying total/done cannot rot.
  - id: new
    severity: Minor
    family: unpinned-done-when
    title: |
      Done-when 7 (failed log read degrades to empty progress) has no test, contradicting the plan's own rule
    detail: |
      Its pin is "the existing behaviour at play_loop.go:246, unchanged", which the
      Done-when header forbids. The behaviour also changed: the degraded progress now
      drives the bar and the summary, which previously re-read the log.
  - id: new
    severity: Minor
    family: nondeterministic-test-read
    title: |
      twiceNumberedOption ranges a map, so a tie picks the "guaranteed miss" at random
    detail: |
      pty_conformance_test.go:905-915. If two digits both appear twice in the frame, the
      returned answer depends on map iteration order and the deliberate miss can become a
      correct answer. Pick the lowest matching digit or fail on a tie.
  - id: new
    severity: Minor
    family: frame-clips-unwrapped-text
    title: |
      choiceFor wraps to opt.width (0 on a non-terminal or below 20 cols) while the frame clips at terminalCols (never 0)
    detail: |
      The wrap width is a policy answer with a sentinel; the clip width is a measurement.
      Where they disagree the gloss arrives unwrapped into a frame that cuts it —
      TestUnwrappedWidthLeavesTheGlossAlone pins the sentinel, which is correct for the
      pipe and wrong for a sitting.
  - id: new
    severity: Minor
    family: figures-drift
    title: |
      --play -raw records nothing but the bar still applies the transition
    detail: |
      decideCapture returns captureNothing under opt.raw (capture.go:30), so
      CaptureReview writes no event while held.answered still advances the in-memory
      progress. On that path D7's "cannot drift from what the next sitting derives" does
      not hold.
  - id: new
    severity: Minor
    family: test-fragility
    title: |
      TestTheBarCountsAnswersAsTheyLand slices on strings.Index without checking for -1
    detail: |
      play_loop_test.go:558 — a bar format change panics the test instead of failing it
      with the message the assertion was written to give.
  - id: new
    severity: Minor
    family: value-receiver-shared-state
    title: |
      sittingDeck has pointer-receiver mutators but is passed by value, sharing its map and half-sharing its slice
    detail: |
      Nothing production depends on the aliasing today, but it is what makes I-3's test
      pass and it is a landmine for #40's second consumer. A *sittingDeck parameter
      removes the question.
```
