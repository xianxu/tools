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

---

## Re-review — 2026-08-31T16:42:01-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 41 — play mode paints frames through screen, and gains a status bar |
| repo | tools |
| issue file | workshop/issues/000041-play-tui.md |
| boundary | whole-issue close |
| milestone | — |
| window | 94f3ad077210c544fe60b8539e2a92def29333c1..b7acbdc1d0126b255505c5751694354e1e0a30d4 |
| command | sdlc close --issue 41 |
| reviewer | claude |
| timestamp | 2026-08-31T16:42:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The window delivers the issue's Spec: `--play` really does draw whole frames through the same `console`/`display` seam the editor uses, the bar is pinned and counted, paging works, the transcript survives exit, and every Done-when row has a named test that exists. I verified four of the prior round's claimed fixes by mutation in a scratch copy (`dropped()` rewritten without aliasing stays green; a no-op `dropped()` goes red; deleting the resize's `opt.width` update reddens `TestANarrowedSittingWrapsTheRestOfItself`; deleting the stdout gate reddens `TestPlayRefusesWhenStdoutIsNotATerminal`), and `go test ./...` is green (107s). What blocks a clean SHIP is one claimed fix that did not land — **BR-1's shared `viewportGesture` has exactly one caller, and `play_loop.go:235-248` still holds the verbatim copy the finding was about**, while the commit message, the atlas and the plan all assert otherwise — plus one measured sibling of the operator's own clipping bug that BR-4's enumeration missed: after a narrowing resize the reveal's *definition body* is written at the startup width and clipped (7 over-wide lines measured on a 4-word deck at 100→40).

### 1. Strengths

- **`newConsole` is a genuine consolidation, not a rename.** `replraw.go:49-94` is called by both loops (`replraw.go:33`, `play_loop.go:96`) with the screen constructor as its one parameter — that is exactly what D1 committed to, and `#40` gets it free.
- **`newPinnedScreen` is pinned in both directions.** `TestOnlyThePinnedConstructorPads` and `TestTheEditorsFooterFollowsItsContent` (screen_test.go:461, :443) mean the padding cannot leak into the REPL, and `TestPaddingNeverReachesTheTranscript` proves it is rows-at-paint rather than lines-in-buffer.
- **`TestASittingReadsTheDeckOnce` (play_loop_test.go:600) is the right shape for the cost claim** — a counting store is the only observer that can see the difference D7 argues about, and `finish` genuinely stopped reading.
- **BR-5's rewrite is the model answer.** The test moved from the caller's own struct to the *drawn footer* (`loadIn`, play_loop_test.go:652); I confirmed it survives an alias-free `dropped()` and dies on a no-op one. That is a test pinning behaviour rather than an implementation detail.
- **`twiceNumberedOption` (pty_conformance_test.go:933) now scans digits in order** and the form-2.3 row uses this issue's own paging as the instrument — a nice use of the feature to fix the test the feature broke.

### 2. Critical findings

None.

### 3. Important findings

**(a) `viewportGesture` has one caller; `--play` still owns a second copy of the policy.** `cmd/define/play_loop.go:235-248`. `grep -n 'viewportGesture'` returns three hits, all in `replraw.go` (definition at :109, call at :413). The play loop's four-case switch plus `wheelLines` is unchanged from before the fix. *Fix:* replace lines 235-248 with `if viewportGesture(view, k) { continue }`. This is disposed as `BR-1: not-addressed` below rather than raised anew.

**(b) The reveal's definition body is clipped after a narrowing resize** — `cmd/define/play_loop.go:323` / `todaysQuestions:436`. 3rd finding in `frame-clips-unwrapped-text`; see the rule statement in the findings block. Measured with a scratch probe: 4-word deck, `opt.width=100`, resize to 40, then reveal → 7 buffer lines exceed 40 columns (worst: `"    ephemerality /əˌfem(ə)ˈralədē/ noun ephemerally …"`, 88 cells). README now claims "a definition longer than the window is scrolled rather than lost", which this contradicts.

**(c) The docs gate: the new refusal surface is undocumented, and five current-truth artifacts name symbols the tree does not have.** 2nd in `doc-sweep-incomplete`; rule and full enumeration in the findings block.

**(d) The `#41` plan's Core concepts table describes two entities the tree does not have.** `workshop/plans/000041-play-tui-plan.md:118` (`choiceFor` — "takes the terminal width and wraps each gloss through `wrapText`"; it takes no width, `optionpool.go:228`) and `:126` (`viewportGesture` — "for both loops"; one loop). `TestPlanTableStatusMatchesTheChangeWindow` checks only the *status* column, so descriptions are unguarded.

### 4. Minor findings

- `runPlay` settles the terminal gates *after* `todaysQuestions` has read the deck and written to stdout — `define --play > file` on an empty deck writes "the deck is empty" into the file and exits 0, never reaching the refusal (`play_loop.go:33` vs `:67`). 2nd in `terminal-ui-gate`.
- `--play -raw` / `DEFINE_NO_CAPTURE`: `decideCapture` returns `captureNothing` (capture.go:29, :135) while `held.answered` still advances the in-memory progress, so D7's "cannot drift from what the next sitting derives" does not hold on that path. (BR-12, still open.)
- `sittingDeck.figures` walks the deck twice per answer — `DailyLoad` directly and again inside `SustainableNewWords` (`schedule/load.go:59`). Not hot enough to matter; noted for `#40`.
- `playbar.go:38` says the bar is "ONE ROW, always"; below ~55 columns `displayRows` charges it two, or `fitFooter` drops it. The arithmetic is correct; the comment overstates.
- `workshop/lessons.md` is untouched across the whole window, after two review rounds and 14 findings (AGENTS.md §4).

### 5. Test coverage notes

- Every Done-when row (0a, 0b, 1-12) resolves to a test that exists and runs; I spot-checked all fifteen names against the tree.
- **The `-no-color` half of BR-3 has no runnable pin here.** `TestPTYPlayRefusesWithNoColor` is the only test for `!opt.tty`, and all `TestPTYPlay*` rows SKIP in this environment (`no pty available: operation not permitted`), as they did last round. The issue Log claims the conformance suite ran green in 124s; I could not reproduce that and am taking it on report. Extracting the two gates into a pure `func playSurfaceRefusal(stdoutIsTTY, tty bool) (string, bool)` would give both halves an in-process pin.
- The `viewportGesture` extraction has no test that would have caught (a): `TestPagingIsNotAnAnswer` asserts on `view.pages`/`view.lines`, which the duplicate switch satisfies identically. A table test driving `viewportGesture` with both loops' dispatch would.
- No test covers a definition (as opposed to an option line) written after a narrowing resize — that gap is finding (b).

### 6. Architectural notes

- **ARCH-DRY — flag.** `viewportGesture` (finding a). `newConsole` passes.
- **ARCH-PURE — pass.** `sittingBar`/`costPhrase`/`wrapOptionLines`/`isOptionLine`/`livePrompt`/`schedule.GradeOf` are pure and unit-tested with no IO; `screen.Paint` stays pure and `liveScreen` is the only terminal toucher; `playSession` is a thin loop over injected `console`. No "pure" entity needs a mock to run.
- **ARCH-PURPOSE — flag.** The `--play` half of the paging policy was the *class* BR-1 named, and only the editor half landed — the instance-not-the-class pattern. Finding (b) is the same shape one level down: BR-4 enumerated three sites for *option lines* when the enumerable class is "everything `--play` writes into the buffer at a width fixed earlier".
- **ARCH-MOCK — pass.** No new external dependency; `display` is faked in-process, `store.Mem`/`countingStore`/`logRefusingStore` are stateful doubles behind the real seam, and the pty suite is the live conformance check (unrunnable here, not absent).
- **ARCH-CONSTRAINTS — pass.** The declared envelope is enforced where it is checkable: the per-answer refresh is O(deck) in memory and the once-per-sitting IO claim is pinned by a counting store. The 16ms paint throttle is inherited unchanged.
- For `#40`: `sittingDeck` travelling by pointer and `newConsole` taking the screen constructor are both the right shapes to build the board on. `wrapOptionLines`' shape-matching on `optionLine`'s output is the one seam that will not generalise — a grid form's lines will not look like `"1  gloss"`, which is another reason to move the wrap to a per-form concern rather than a regex over rendered text.

### 7. Plan revision recommendations

`workshop/plans/000041-play-tui-plan.md` needs a `## Revisions` entry recording:

1. **The `choiceFor` Core-concepts row is stale.** BR-4 moved the wrap to `wrapOptionLines` at write time and `choiceFor` reverted to its old signature; the row still describes the reverted design. Restate it as `unchanged` or delete it.
2. **The `viewportGesture` row's "for both loops" is not true of the tree.** Either land the `--play` call site or change the row to say the editor is the only caller and `--play` keeps a copy.
3. **The BR-4 revision's "All three are closed" overclaims site (c).** At `cols < 20` the loop sets `opt.width = 0`, `wrapOptionLines` returns the text unchanged, and the frame still clips — that is a deliberate policy exception, not a closed site, and it should be written as one.
4. **The enumeration BR-4 forced was scoped to option lines.** Record the widened enumeration — everything the loop writes to the buffer (`q.Prompt()`, `asked.Reveal()`, the drop notice, `finish`'s lines, stderr diagnostics) — and which of them derive their width at write time.

`workshop/plans/000038-play-clickable-plan.md:101,128,237` name `playConsole`, which this window deleted; those rows should say `newConsole` before `#38` starts.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      viewportGesture exists (replraw.go:109) but has ONE caller (replraw.go:413); play_loop.go:235-248 still holds the verbatim four-case switch. The enterMouse-cost half IS addressed (replraw.go:63-66).
  - id: BR-2
    disposition: addressed
    note: |
      TestAFailedLogReadStillRunsTheSitting drives the degraded path through a real bar draw via logRefusingStore.
  - id: BR-3
    disposition: addressed
    note: |
      Both gates land before enterRaw; deleting them reddens TestPlayRefusesWhenStdoutIsNotATerminal. The -no-color half is pinned only by a pty row that skips here.
  - id: BR-4
    disposition: addressed
    note: |
      Verified by mutation: removing the resize's opt.width update reddens TestANarrowedSittingWrapsTheRestOfItself. Option lines only; see the new definition-body finding.
  - id: BR-5
    disposition: addressed
    note: |
      Mutation-verified both ways: an alias-free dropped() stays green, a no-op dropped() goes red on the drawn bar.
  - id: BR-6
    disposition: addressed
    note: |
      crlf.go and crlf_test.go deleted, shortWriter re-homed. Residual mentions of the deleted FILE names roll into the new doc-sweep finding.
  - id: BR-7
    disposition: addressed
    note: |
      newConsole(ctx, d, sess, stdout, newScreen) is called from replraw.go:33 and play_loop.go:96.
  - id: BR-8
    disposition: addressed
    note: |
      refresh() assigns the whole figures() struct then re-applies total and done (play_loop.go:122-132).
  - id: BR-9
    disposition: addressed
    note: |
      Same fix as BR-2.
  - id: BR-10
    disposition: addressed
    note: |
      twiceNumberedOption now scans '1'..'9' in order rather than ranging the map (pty_conformance_test.go:944-949).
  - id: BR-11
    disposition: addressed
    note: |
      choiceFor no longer wraps; the write-time wrap uses the live opt.width, and the non-terminal case is now refused outright. The sub-20-column sentinel remains as an explicit policy — see plan revision 3.
  - id: BR-12
    disposition: not-addressed
    note: |
      decideCapture still returns captureNothing under opt.raw while held.answered advances the in-memory progress. Minor; never blocks.
  - id: BR-13
    disposition: addressed
    note: |
      play_loop_test.go:558 now Fatalf's on a missing separator instead of slicing.
  - id: BR-14
    disposition: addressed
    note: |
      sittingDeck travels by pointer from todaysQuestions through playSession; confirmed by the alias-free mutation staying green.
findings:
  - id: new
    severity: Important
    family: frame-clips-unwrapped-text
    title: |
      A reveal written after a narrowing resize carries a definition wrapped to the STARTUP width, and the frame clips it
    detail: |
      This is the 3rd finding in family `frame-clips-unwrapped-text`. Earlier rounds fixed
      instances (the operator's startup-width option gloss, then BR-4's resize sibling). Do NOT
      fix this instance alone.

      THE RULE that covers all of them: `--play` RENDERS its text at queue-build time and WRITES
      it much later, so every pre-rendered artifact carries a width that may already be wrong when
      the frame clips it. Anything the loop writes into the buffer must be wrapped to the width in
      force at the moment of WRITING, not at the moment of rendering. `wrapOptionLines` applies
      that rule to exactly one line-kind; `todaysQuestions:436` calls `Render(..., Width: opt.width)`
      once per sitting for every definition, and `play_loop.go:323` writes those lines after any
      number of resizes.

      MEASURED prevalence (scratch probe, `playRig` with sycophantic/ephemeral/quokka/mesa,
      opt.width=100, resize 100->40, then one reveal): 7 buffer lines written AFTER the narrow
      exceed 40 columns, worst 88 cells. README.md:148 now claims "a definition longer than the
      window is scrolled rather than lost", which this contradicts.

      The enumeration the rule implies, to be swept in ONE round: q.Prompt(), asked.Reveal()
      (option lines AND the rendered definition body), the "removed %q from the deck" notice,
      finish()'s summary lines, and con.stderr diagnostics. Two candidate mechanisms: render the
      entry lazily at write time against the live opt.width, or keep the unwrapped source on the
      Question so a re-wrap is possible.
  - id: new
    severity: Important
    family: doc-sweep-incomplete
    title: |
      Five current-truth artifacts name symbols the tree does not have, and the new refusal surface reaches neither README nor atlas
    detail: |
      This is the 2nd finding in family `doc-sweep-incomplete`. Do NOT fix the instances one by
      one — that is what failed in rounds 1 and 2.

      THE RULE, and it is mechanisable because the repo already built most of it: every name a
      current-truth artifact cites must be DECLARED in the tree. `TestARemovedDeclarationIsSweptOrRetired`
      (repo_guard_test.go) only sees `-func` lines whose names pass `isCitableName` (exported or
      Test*), so it structurally cannot see a removed TYPE (`crlfWriter`), a removed unexported
      func (`playConsole`, `draw`, `fitMenu`), a deleted FILE path (`crlf_test.go`), or a name that
      was never declared at all. Widening it to (a) removed type/const/unexported declarations,
      (b) deleted file paths, and (c) a forward check that every `Test[A-Z]\w+` cited in a
      current-truth artifact is declared, turns this whole family into a build failure.

      MEASURED prevalence at HEAD, all introduced by this window:
        - cmd/define/play_loop.go:278 cites `TestAMissIsRecordedBeforeItIsRevealed`; the tree
          declares `TestAMissRecordsBeforeItPlays`. Introduced by 4d3b53a; never existed.
        - cmd/define/highlightwriter.go:60 says "crlf_test.go defends that" — file deleted here.
        - cmd/define/highlightwriter_test.go:158 cites "crlf_test.go's fixture", 400 lines above
          the fixture's new home in the same file.
        - atlas/define.md:2006 says "`playConsole` is `replRaw`'s construction"; the symbol was
          renamed to `newConsole` in the final commit.
        - workshop/plans/000038-play-clickable-plan.md:101, :128, :237 name `playConsole`.
        - Two prose breaks left by the sweep: atlas/define.md:942-943 ("It used to wrap the raw
          loop's / the raw loop's line-ending writer") and highlightwriter_test.go:142-144
          ("...right answer here while / A writer that can be written to again...").

      THE DOCS GATE, same rule at the behaviour level: this window added two user-facing refusals
      (`define --play > file` and `define --play -no-color` now print a message and exit 1,
      play_loop.go:67-74). Neither README.md nor atlas/define.md records them, and the atlas's
      "Degrading is the absence of input" paragraph (:470-476) still describes only the editor's
      degrade-by-routing. `newConsole`, `viewportGesture` and `wrapOptionLines` — three new shared
      surfaces `#40` will consume — appear in neither.
  - id: new
    severity: Important
    family: plan-table-vs-tree
    title: |
      The plan's Core concepts table describes two entities the tree does not have, and the guard checks only the status column
    detail: |
      `workshop/plans/000041-play-tui-plan.md:118` says `choiceFor` "takes the terminal width and
      wraps each gloss through `wrapText`" — BR-4 reverted exactly that, and `optionpool.go:228`
      declares `choiceFor(word, rendered string, e Entry, pool []play.Candidate, seed uint64)`.
      `:126` says `viewportGesture` is "PURE dispatch — the paging keys, for both loops"; it has
      one caller. The plan's BR-4 revision also asserts "all three are closed" when site (c),
      `opt.width == 0`, is a deliberate policy exception at cols < 20.

      `TestPlanTableStatusMatchesTheChangeWindow` passes both rows because it only judges the
      `modified`/`unchanged`/`new` column against `git diff` — the DESCRIPTION is unguarded. The
      plan's own rounds 2 and 3 stated the rule ("a Revision that reverses a decision re-reads the
      decision prose, the entity tables, AND the Done-when rows"); this is the fourth time a
      reversal reached the prose and not the table. `#40` reads this table as the record of what
      landed.
  - id: new
    severity: Minor
    family: terminal-ui-gate
    title: |
      The surface gate runs after todaysQuestions has already read the deck and written to the non-terminal stdout
    detail: |
      This is the 2nd finding in family `terminal-ui-gate`. THE RULE: every precondition for
      owning the terminal is settled in ONE place, before the command does any work or writes any
      byte — the same rule main.go:542 states for usage errors ("settled BEFORE a store is
      opened"). `runPlay` calls `todaysQuestions` at :33 and only checks `isTerminal(stdout)` and
      `opt.tty` at :67-74, so `define --play > file` on an empty deck writes "the deck is empty"
      into the file and exits 0 without ever reaching the refusal, and on a non-empty deck it pays
      the deck+log reads first. Moving the two gates above the `todaysQuestions` call closes both.
  - id: new
    severity: Minor
    family: lessons-not-recorded
    title: |
      workshop/lessons.md is untouched across a window that ran two review rounds and 14 findings
    detail: |
      AGENTS.md section 4: "When you run code review, add rules to workshop/lessons.md that
      prevent the mistakes you found." Three families repeated across rounds
      (`frame-clips-unwrapped-text`, `doc-sweep-incomplete`, `parallel-construction`) and the
      rules the plan wrote for itself live only in that plan's Revisions, which is archived at
      close. The two durable ones — "adopting an existing seam inherits its behaviour on inputs
      the previous consumer never sent it" and "re-examine the tests that assert over the surface
      this issue changes MEANS enumerating them" — belong in lessons.md, where the next issue
      reads them.
```

---

## Re-review — 2026-08-31T17:13:53-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 41 — play mode paints frames through screen, and gains a status bar |
| repo | tools |
| issue file | workshop/issues/000041-play-tui.md |
| boundary | whole-issue close |
| milestone | — |
| window | 94f3ad077210c544fe60b8539e2a92def29333c1..4653c0f8543dbbe7463a74b326c6323d78dcd43f |
| command | sdlc close --issue 41 |
| reviewer | claude |
| timestamp | 2026-08-31T17:13:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's substance landed and it is well pinned: `--play` really does draw through `console`/`display`, the bar is pinned by frame-geometry tests, paging/SIGWINCH/write-once/one-read-per-sitting all have predicates, and the two headline round-2 fixes survive mutation (`wrapWritten` no-op → `TestANarrowedSittingWrapsTheRestOfItself` reddens with 7 overwide lines; removing the play loop's `viewportGesture` call reddens two tests). `go test ./...`, `go test ./cmd/define -race` are green here; the pty rows skip in this environment. What keeps it from SHIP is that two of the five open findings are disposed by the tree but not by a predicate or a mechanism: BR-16's `doc-sweep-incomplete` closing commit swept five stale citations and introduced a sixth (`TestHighlightWriterShortWriteContract`, which no file declares), and BR-18's gate move is real but reverting it leaves the entire suite green. One new measured instance of `frame-clips-unwrapped-text` remains — `playAnnounced`'s diagnostic writes into the same clipping frame without going through `wrapWritten` (probe: 156 cells in a 40-column terminal).

## 1. Strengths

- **`newConsole` + `viewportGesture` are the right consolidation, and both are genuinely reachable from both loops** (`replraw.go:44`/`:109`, called at `replraw.go:33`, `replraw.go:413`, `play_loop.go:247`). The screen constructor as the single parameter is exactly the "widen the seam" move D1 committed to.
- **`wrapWritten` states the class rather than the instance** (`playbar.go:88`), and its pin is a predicate over *every* line written after a narrowing resize (`play_loop_test.go:810-820`) rather than one assertion per line-kind. I confirmed by mutation that it covers option lines, the rendered definition body, and the summary in one test.
- **Done-when 11/12 are pinned in both directions** — `TestAShortQuestionStillPinsTheBar` and `TestOnlyThePinnedConstructorPads` (`screen_test.go:413`, `:471`) assert the padding exists for `--play` *and* does not leak into the editor, with `TestPaddingNeverReachesTheTranscript` checking `Lines()` across two heights.
- **`schedule.GradeOf` makes D7's DRY claim structural** (`schedule/progress.go:163`): `gradeOf` is now one line over it, so the bar's transition and the fold's cannot diverge.
- **`twiceNumberedOption`'s determinism fix is real** (`pty_conformance_test.go:918-950`): scanning `'1'..'9'` in order instead of ranging a map removes the tie-breaking nondeterminism BR-10 named.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `playAnnounced` writes into the clipping frame without `wrapWritten` (`cmd/define/play_loop.go:354` → `cmd/define/main.go:856`).**
**This is the 4th finding in family `frame-clips-unwrapped-text`.** Earlier rounds fixed instances. Do NOT fix this instance — the rule is already written down (`playbar.go:66-87`, `lessons.md`); what is missing is that it is enforced at *call sites the loop owns*, so any write made by a shared helper the loop calls escapes it. Measured with a scratch probe (`playRig`, `opt.width=40`, a player returning a network error, one miss → reveal): the transcript gains a 156-cell line in a 40-column terminal, which `Paint` clips. `atlas/define.md` already claims the loop routes "its diagnostics" through the one function; it routes only the drop error (`play_loop.go:312`). Two other unwrapped sites on the same path: the `♫ playing N×` indicator and `reportVoice`'s fallback line. *Fix sketch:* move the wrap to the **seam** — have the pinned screen (or a `console` stdout/stderr wrapper) wrap on `Write` against its own `cols`, keeping the sub-20-column policy in one place — so a future helper cannot write around it; then extend `TestANarrowedSittingWrapsTheRestOfItself` (or a sibling) to drive a failing playback so the predicate covers the helper path. ARCH-PURPOSE: the enumeration in `playbar.go:82` lists five loop-owned sites and stops at the loop's boundary; that is the instance, not the class.

**I-2 — BR-16's sweep introduced a fresh instance of the family it was closing, and the mechanism it asked for was not built.**
`cmd/define/highlightwriter.go:60` (added by `4653c0f`) says "`TestHighlightWriterShortWriteContract` in this file's neighbour defends that". No file declares that test; the nearest real one is `TestHighlightWriterTreatsAShortWriteAsAnError` (`highlightwriter_test.go:157`), which defends the *highlight* writer's short-write handling, not the deleted writer's caller-unit-progress contract the sentence attributes to it. `repo_guard_test.go` is untouched across the whole window, so BR-16's widening — (a) removed types/unexported decls, (b) deleted file paths, (c) a forward check that every `Test[A-Z]\w+` cited in a current-truth artifact is declared — was not built. A scan for (c) across `cmd/` finds a pre-existing second live instance outside this window: `TestTheClampIsUnreachable`, cited as the pin at `schedule/box.go:48` and `atlas/define.md:1837`, is declared nowhere. Detail in the disposition below.

## 4. Minor findings

- **M-1 — the initial `fig` hand-copies `refresh()`'s first two lines** (`play_loop.go:129-141`). **2nd finding in family `partial-copy-refresh`** — BR-8 fixed `refresh()` copying fields out of `figures()`; the initializer is now the site that does it, so a third field added to `refresh` is stale on the first frame. The rule: one expression builds the figures and every site derives from it. `refresh()` is exactly equivalent at init (`s.Right+s.Wrong == 0`), so the fix is to call it.
- **M-2 — the plan's Integration-points table holds three pure entities, and its Pure table holds an IO constructor** (`workshop/plans/000041-play-tui-plan.md:124-127`, `:112`). `viewportGesture` (self-labelled "PURE dispatch"), `wrapWritten` (string→string in `playbar.go`) and `livePrompt` (`play.Session`→string) wrap no external dependency; `newPinnedScreen`, which takes a tty and builds the IO shell, sits under Pure entities. **2nd finding in family `plan-table-vs-tree`** — the rule: a row makes three claims (path, status, kind/description) and `TestPlanTableStatusMatchesTheChangeWindow` guards one. Either mechanise what can be mechanised (every backticked Name is declared at the stated path; an Integration row names a wrapped external dependency) or stop asserting the unguarded columns in the table `#40` reads as the record of what landed.
- **M-3 —** `--play`'s inherited `enterMouse` cost (drag-select needs Option/Shift) is recorded in `replraw.go:63-66` and `lessons.md`, but the README's `--play` section and the atlas's new "The sitting is a frame" section do not mention it; `/help` documents it for the editor only. Folded into the I-2 docs gate.

## 5. Test coverage notes

- All seven `TestPTYPlay*` rows **SKIP** here (`no pty available: operation not permitted`), including `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` — the only pin for D5a's Critical — and `TestPTYPlayRefusesWithNoColor`, the only pin for the `-no-color` refusal. The issue log reports the tagged suite green on real hardware (124s); I could not reproduce that in this environment.
- `-no-color` is unreachable in-process because it is only checked after `isTerminal(stdout)` passes. Extracting the three-way refusal into a pure `refuseReason(stdinTTY, stdoutTTY, tty bool)` would make all three causes table-testable without a pty and leave only the detection in the IO shell (ARCH-PURE).
- The wrap tests all run `color: false`; production runs colour on. I probed the coloured path (`opt.color = true`, resize 100→40, reveal): **0 overwide lines**, so `visibleCells`/`escapeLen` handle it. Worth a case in the table so it stays true.
- `go test ./...` green; `go test ./cmd/define -race -count=1` green (120s).

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass**, with M-1 as the one residue. `newConsole`, `viewportGesture`, `costPhrase`, `GradeOf`, `OptionIndent` all have two real callers, each verified.
- **ARCH-PURE — pass.** `screen` stays pure with `liveScreen` as the only terminal-touching part; `sittingBar`/`costPhrase`/`wrapWritten`/`livePrompt`/`sittingDeck` are unit-tested with no IO. Flag only that the plan's tables label three of them as integration points (M-2), which is what `#40` will read.
- **ARCH-PURPOSE — flagged (I-1).** The single-source shadow-sweep passes for the prompt lines (`doc_sync_test.go` derives the README), the bar/summary (one `costPhrase`), the grade rule (`GradeOf`) and the option indent (`OptionIndent`). It fails for the wrap: `wrapWritten` is a rule applied by convention at five call sites rather than enforced at the seam, and the one writer outside the loop's body does not derive.
- **ARCH-MOCK — pass with a caveat.** No new external dependency; `display`/`console` is the seam and `recordingConsole` the double both loops share, with `fakePlayer` and a counting store behind it. The live conformance check exists but cannot run here.
- **ARCH-CONSTRAINTS — pass.** The declared envelope is enforced: `refresh()` is charged per answered question rather than per frame, `TestASittingReadsTheDeckOnce` holds the one-`Deck()`/one-`Events()` bound with a counting store, and the 16ms throttle is inherited untouched. `figures()` is O(deck) in memory as stated.
- For `#40`: `newConsole` takes the screen constructor and `viewportGesture` owns the key policy, so the board is a third caller of both without new parameters. If I-1 is fixed at the seam, `#40` inherits wrapping for free rather than re-enumerating its own write sites.

## 7. Plan revision recommendations

- **`## Revisions` — "boundary review round 3"**: record that the Integration-points table lists `viewportGesture`, `wrapWritten` and `livePrompt`, which are pure, and that `newPinnedScreen` sits under Pure entities though it builds the IO shell; state the rule (a row asserts path + status + kind, and only status is guarded) and say which of the three the guard will now check.
- **Same entry**: correct `wrapWritten`'s row — it wraps everything *the loop's own body* writes, not "EVERYTHING the loop writes"; `playAnnounced`'s diagnostics reach the buffer unwrapped. Update the `frame-clips-unwrapped-text` enumeration in the round-2 revision to include site (e), writes made by helpers the loop calls.
- **Same entry**: note that BR-16's mechanism (widening `TestARemovedDeclarationIsSweptOrRetired`) was not built, and that the sweep introduced `TestHighlightWriterShortWriteContract` at `highlightwriter.go:60` — the family's own evidence that instance-fixing is what keeps failing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      viewportGesture called from replraw.go:413 and play_loop.go:247; stubbing the play call reddens TestPagingIsNotAnAnswer and TestALongRevealPagesRatherThanScrollingTheWordAway. The enterMouse cost is in the source comment only — folded into BR-16's docs gate.
  - id: BR-12
    disposition: not-addressed
    note: |
      Unchanged at HEAD: decideCapture still returns captureNothing under opt.raw (capture.go:30, :135) while held.answered advances the in-memory progress; no --play/-raw conflict guard exists in main.go.
  - id: BR-15
    disposition: addressed
    note: |
      Mutation-verified: making wrapWritten a no-op reddens TestANarrowedSittingWrapsTheRestOfItself with 7 overwide lines (worst 97 cells), covering option lines, the rendered definition body and the summary. The helper-write gap is raised separately as the family's 4th.
  - id: BR-16
    disposition: not-addressed
    note: |
      Instances swept, but the closing commit added a new one and the mechanism was not built. See the finding detail below.
  - id: BR-17
    disposition: addressed
    note: |
      All three named contradictions verified fixed against the tree: choiceFor is unchanged and its row says so, viewportGesture now has two callers, and site (c) is stated as a policy exception. The unguarded-column half is re-raised as the family's 2nd with new evidence.
  - id: BR-18
    disposition: not-addressed
    note: |
      The gates did move above todaysQuestions (play_loop.go:52-72), but nothing pins it: moving them back below the call in a scratch worktree leaves the whole cmd/define suite green (107s, ok). The same round moved TestEmptyQueueExitsZero off runPlay onto todaysQuestions, deleting the only runPlay-on-empty-deck coverage.
  - id: BR-19
    disposition: addressed
    note: |
      lessons.md gains three rules (seam adoption, enumerate the class, a plan naming an anti-pattern is not protection), each with the #41 evidence.
findings:
  - id: new
    severity: Important
    family: frame-clips-unwrapped-text
    title: |
      playAnnounced writes into the clipping frame without wrapWritten, so a playback diagnostic is cut mid-sitting
    detail: |
      This is the 4th finding in family `frame-clips-unwrapped-text`. Do NOT fix this instance —
      the rule is already stated at playbar.go:66-87; what is missing is that it is enforced at
      call sites the LOOP owns, so any write made by a helper the loop calls escapes it.
      play_loop.go:354 calls playAnnounced, whose stderr is the screen; main.go:856 writes
      `define: %s` unwrapped, and main.go:844/:866 and reportVoice do the same.
      MEASURED (scratch probe: playRig, opt.width=40, newPinnedScreen(24,40), a player returning
      a network error, one miss then a reveal): one transcript line of 156 cells in a 40-column
      terminal, which Paint clips. atlas/define.md's new section already claims the loop routes
      "its diagnostics" through the one function; only the drop error at play_loop.go:312 is.
      Fix the class: wrap at the SEAM — the pinned screen's or the console's Write, against its
      own cols, with the sub-20-column policy in one place — so a helper cannot write around it,
      and extend the narrowing test to drive a failing playback so the predicate covers it.
  - id: new
    severity: Minor
    family: partial-copy-refresh
    title: |
      The initial sittingFigures hand-copies refresh()'s first two lines
    detail: |
      This is the 2nd finding in family `partial-copy-refresh`. BR-8 fixed refresh() copying
      named fields out of figures(); play_loop.go:129-130 is now the site that does it, so a
      third adjustment added to refresh() would be stale on the first frame, before any answer.
      The rule: one expression builds the figures and every site derives from it. refresh() is
      exactly equivalent at init because s.Right+s.Wrong is 0 there, so calling it is the fix.
  - id: new
    severity: Minor
    family: plan-table-vs-tree
    title: |
      Three pure entities are listed as Integration points and an IO constructor as Pure
    detail: |
      This is the 2nd finding in family `plan-table-vs-tree`. Do NOT fix only these rows.
      workshop/plans/000041-play-tui-plan.md:125-127 lists viewportGesture (whose own text says
      "PURE dispatch"), wrapWritten (string to string, playbar.go) and livePrompt (play.Session
      to string) under "Integration points", whose column header is "Wraps"; none wraps an
      external dependency and none is injected. Conversely newPinnedScreen (:109), which takes a
      tty and builds the IO shell, sits under "Pure entities".
      THE RULE: a table row asserts path, status, and kind/description, and
      TestPlanTableStatusMatchesTheChangeWindow judges only status. Either mechanise what can be
      — every backticked Name is DECLARED at the stated path, and an Integration row names a
      wrapped external dependency — or stop asserting the unguarded columns in the table `#40`
      reads as the record of what landed.
```

---

## Re-review — 2026-08-31T17:41:30-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 41 — play mode paints frames through screen, and gains a status bar |
| repo | tools |
| issue file | workshop/issues/000041-play-tui.md |
| boundary | whole-issue close |
| milestone | — |
| window | 94f3ad077210c544fe60b8539e2a92def29333c1..6201f4d1b57cd7f4eafb5121e9c21b8f0f31b120 |
| command | sdlc close --issue 41 |
| reviewer | claude |
| timestamp | 2026-08-31T17:41:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The issue's Spec is delivered and delivered well — `--play` genuinely draws through the editor's `console`/`display`, the playback dance is gone rather than worked around, the bar is pinned in both directions, and the four prior rounds' structural findings (BR-1/BR-5/BR-7) were closed by extraction rather than by patching the instance. What blocks SHIP is that the final commit's own fix re-opened the family it claimed to close: `wrapWritten` now returns any line containing an escape untouched, and `--play` can only run with `opt.color == true` (it refuses unless `opt.tty`, which is the same expression as `opt.color`). I measured a coloured sitting at 40 columns and got four buffer lines of 55–87 cells that `Paint` clips — the exact defect of the operator's original report, restored for the only configuration production has. No in-process test can see it because `playRig` hardcodes `color: false`, and the one pty resize row changes rows (24→10) but never columns. Separately, BR-16 stays open with two *new* stale citations introduced this round, and BR-18's fix is correct in the code but unpinned — I moved the terminal gates back below `todaysQuestions` and the whole package stayed green.

## 1. Strengths

- **D5a resolved by deletion, not by workaround.** `play_loop.go:325-345` — the `restore`/`enterRaw` pair and its exit-1 branch are gone because nothing is handed back any more. The comment states why the branch is not being weakened. This is the strongest thing in the window.
- **BR-1/BR-7 closed at the seam, both halves.** `newConsole` (`replraw.go:37`) takes the screen constructor as its one parameter, and `viewportGesture` (`replraw.go:96`) is called by *both* loops — `play_loop.go:245` deletes the copy rather than leaving it beside the extraction, which is the half that was missing a round earlier.
- **The non-drift claim is enforced by one function.** `schedule.GradeOf` (`schedule/progress.go:174`) with `gradeOf(e) = GradeOf(e.Correct, e.Unaided)` as a one-liner means the bar's transition and the fold's transition cannot disagree — a DRY claim that names the function instead of asserting two pieces of code agree.
- **The padding decision is pinned in the direction that usually gets skipped.** `TestOnlyThePinnedConstructorPads` and `TestTheEditorsFooterFollowsItsContent` (`screen_test.go:507`, `:441`) assert the editor's appearance did *not* change, which is what keeps this from being a second issue wearing this one's clothes.
- **`workshop/lessons.md:2769+`** gained three rules with their why, including the `grep -l` enumeration lesson that produced the fourth pty row.

## 2. Critical findings

**`cmd/define/playbar.go:105-109` — the escape-skip disables the wrap in the only configuration `--play` can run in.**

`wrapWritten` skips any line containing `0x1b`. `runPlay` refuses unless `isTerminal(stdout)` and `opt.tty`, and `main.go:523/527` define `color` and `tty` as the *same* expression — so every real sitting has `opt.color == true`, and `todaysQuestions` (`play_loop.go:431`) renders with `Color: opt.color`. Every styled line of a reveal therefore bypasses the seam.

Measured (scratch test, `playRig` + `opt.color = true`, `opt.width = 100`, `newPinnedScreen(24, 40)`, one reveal):

```
line is 87 cells in a 40-col terminal: "    ephemerality \x1b[35m/əˌfem(ə)ˈralədē/\x1b[0m noun ephemerally …"
line is 85 cells in a 40-col terminal: "        \x1b[3;32m“chickweed is an ephemeral weed, producing …”\x1b[0m"
4 over-wide lines
```

Deleting the three-line skip makes that go to 0 over-wide lines and leaves `go test ./cmd/define/...` green (106s) — so nothing in the suite defends the skip, and the stated hazard is over-broad: `wrapText` measures with `visibleCells` and splits on `strings.Fields`, and no escape sequence contains whitespace. The `♫ playing 3×` line and `\r\x1b[K` are short and return unchanged from the width check anyway.

Two things make this the fifth finding in the family rather than a fresh bug: the skip was *added* by the commit that claimed to close the family, and the same commit deleted the resize-time `opt.width` maintenance that had been the prior mitigation.

**This is the 5th finding in family `frame-clips-unwrapped-text`.** Do not fix this instance. The rule the four earlier instances were reaching for is now one step further than "wrap at the seam":

> A seam covers every write only when (a) every path into the buffer goes through it, and (b) its predicate is exercised in the configuration production actually runs in.

Both halves are enumerable, and the enumeration is small enough to write down:

| path into `screen.lines` | wraps today |
|---|---|
| `liveScreen.Write` (`screen.go:511`) | yes, unless the line carries an escape |
| `liveScreen.WriteRegions` (`screen.go:551`) | **no** — calls `l.s.Write` directly |

| line class | wraps today |
|---|---|
| plain | yes |
| escape-carrying (i.e. all styled output) | **no** |

`WriteRegions` is latent today (`--play` passes no regions, `play_loop.go:429`) and live for `#40`, which is the clickable board. The comment at `screen.go:503` — *"Here nothing can write around it"* — is already false for it.

Fix sketch: route both methods through one private `l.writeBuffer(text)` that applies the wrap when `pinned`; replace `strings.ContainsRune(line, 0x1b)` with a guard on the `eraseLine` gesture specifically; and make the predicate in `TestANarrowedSittingWrapsTheRestOfItself` run with `opt.color = true`.

## 3. Important findings

**`cmd/define/play_loop_test.go:39` — `playRig` returns `options{color: false, …}`, a configuration `--play` cannot be in.**

Since BR-3, `runPlay` exits 1 unless `opt.tty`, and `opt.tty` and `opt.color` are the identical expression. Every in-process `--play` test therefore drives a state production refuses to enter, and the tests that *would* have caught the Critical above — `TestANarrowedSittingWrapsTheRestOfItself` (Done-when 0b) and `TestALongOptionGlossWrapsRatherThanBeingCut` — pass for that reason. The pty rows do run with colour, but `TestPTYPlayResizeRepaints` (`pty_conformance_test.go:664`) narrows rows 24→10 and never touches columns, so no row narrows the width on a coloured sitting.

The rule: **a rig's defaults have to be reachable from the flag parse of the command under test.** Set `color: true, tty: true` in `playRig` (or derive them from one helper as `main.go` does) and add a column-narrowing pty row.

## 4. Minor findings

- `cmd/define/playbar.go:122-137` — the final commit spliced `minWrapWidth`'s comment and declaration into the middle of `isOptionLine`'s doc block; `isOptionLine` now has no doc comment and `minWrapWidth`'s doc opens with a paragraph about option lines. (Counted under BR-16, not raised separately.)
- `wrapWritten` lives in `playbar.go` — "what a sitting tells the learner about its own cost" — while its only caller is `screen.go`. It should sit beside the seam it defends.
- `screen.pinned` now governs two orthogonal properties: pad the buffer region, *and* wrap writes. `#40` will want a pinned board; a future editor change might want wrapping. Two fields or a small policy struct.
- `TestUnwrappedWidthLeavesTheGlossAlone` (`optionpool_test.go:~466`) pins `width == 0`, which the seam can no longer produce — `terminalCols` floors at 80 and `l.cols` is never 0. Still a valid unit contract; just no longer a path.

## 5. Test coverage notes

- `go test ./...` green at HEAD (106s). `go test -tags conformance ./cmd/define/` fails only on `TestReflectAgainstTheLiveService` (expired OAuth token in this environment, unrelated to the window).
- **All 16 `TestPTY*` rows SKIP here** (`no pty available`), including `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal`, which is the only pin for D5a's Critical, and `TestPTYPlayRefusesWithNoColor`, which is Done-when 0a's second pin. Same gap the previous round reported; I could not verify them on real hardware and am taking the issue Log's claim as unverified rather than false.
- BR-18's fix is unpinned: I moved the three terminal gates back below `todaysQuestions` and `go test ./cmd/define/` stayed green (107s). `TestPlayRefusesWhenStdoutIsNotATerminal` asserts exit 1, no escapes in stdout, and a stderr substring — an empty-deck message contains no escapes, so it passes either way. The pin wants `out.Len() == 0` plus a counting store showing zero `Deck()`/`Events()` calls.
- BR-21's fix (`refresh()` at init) is behaviour-identical to the hand-copy it replaced, so no test can fail without it. Accepted as structural.

## 6. Architectural notes

- **ARCH-DRY — pass.** `newConsole`, `viewportGesture` and `costPhrase` each have two real callers; the test-side `recordingConsole`/`playbackConsole` share one double. The only DRY-adjacent nit is `wrapWritten`'s file placement.
- **ARCH-PURE — pass.** `sittingBar`/`costPhrase`/`wrapWritten`/`isOptionLine`/`livePrompt`/`GradeOf`/`screen.Paint` are all unit-tested with no IO; `liveScreen` remains the only terminal-toucher; `playSession` is a thin loop over an injected `console`. `viewportGesture` needs a `display` double to test — it is a thin dispatcher rather than strictly pure, which the plan's row now hedges as "PURE dispatch"; not worth a third `plan-table-vs-tree` finding, but `#33`'s widening should decide the column's meaning.
- **ARCH-PURPOSE — flag.** The family was declared closed at the seam in the same commit that opened a new hole in it. The finding named one site (`playAnnounced`); the class is "every path into the buffer × every line class", and that enumeration was never written — which is precisely the pattern the family escalation exists to surface. Same axis on BR-16: the class was correctly identified and then *deferred to `#33`* while this window added two more instances to the tree.
- **ARCH-MOCK — pass.** No new external dependency. `display` is the seam, the doubles run the whole loop, and production and test share the boundary. The pty suite is the live conformance check; it just cannot run here.
- **ARCH-CONSTRAINTS — pass.** `refresh()` is charged per answer, not per frame, and `TestASittingReadsTheDeckOnce` pins the one-`Deck()`/one-`Events()` claim with a counting store. The 16ms throttle is inherited rather than re-tuned. `wrapWritten` is now O(text) per write on a pinned screen — off the keystroke path, since the sitting writes only on transitions.

## 7. Plan revision recommendations

1. **`atlas/define.md:2044-2058`** — the wrap paragraph describes the *third* version: *"the loop routes the question, the reveal, the drop notice, the summary and its diagnostics through one function, and the resize case keeps `opt.width` current."* At HEAD the loop routes nothing (the pinned screen's `Write` does it) and `play_loop.go:205-215` explicitly refuses to set a second width. Rewrite for the seam, and state the escape exception once it is decided.
2. **`workshop/plans/000041-play-tui-plan.md:311`** — cites `TestAMissIsRecordedBeforeItIsRevealed`; the tree declares `TestAMissRecordsBeforeItPlays`. This is the same never-existed name `#33`'s new Revisions section quotes as the example, still live in a current-truth artifact.
3. **A `## Revisions` entry for the 5th family instance**, recording that the seam fix introduced it, and carrying the two-axis enumeration above so the next round has a predicate rather than an instance.
4. **Done-when 0b** should say its predicate runs with colour on; as written it is satisfied by a test in a state production refuses.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Nothing at HEAD touches it: no usage gate rejects `--play -raw`, decideCapture still returns captureNothing (capture.go:30), and held.answered/refresh still advance the bar.
  - id: BR-16
    disposition: not-addressed
    note: |
      repo_guard_test.go is untouched in this window, so neither the removed-type/unexported/deleted-file widening nor the forward "cited Test name is declared" check exists; the class was filed to #33 while this round added two NEW instances — highlightwriter.go:60 cites TestHighlightWriterShortWriteContract (never existed; the real name is TestHighlightWriterTreatsAShortWriteAsAnError) and playbar.go:122-137 lost isOptionLine's doc comment to the inserted minWrapWidth const — plus two survivors: plan:311 still cites TestAMissIsRecordedBeforeItIsRevealed, and atlas/define.md:2054 still describes the deleted per-site wrap and the deleted resize-time opt.width update.
  - id: BR-18
    disposition: not-addressed
    note: |
      The code is right (gates at play_loop.go:56-77, above todaysQuestions at :79) but nothing pins it — I moved the three gates back below todaysQuestions in a scratch copy and `go test ./cmd/define/` stayed green (107s); TestPlayRefusesWhenStdoutIsNotATerminal only checks exit 1, absence of ESC, and a stderr substring, none of which a "the deck is empty" line to stdout would break.
  - id: BR-20
    disposition: addressed
    note: |
      The wrap moved to liveScreen.Write for pinned screens (screen.go:511) and playAnnounced's plain-text `define: %s` warning is now wrapped; pinned in both directions by TestThePinnedScreenWrapsWhateverIsWrittenToIt — but the same commit introduced a new hole in the seam, raised below.
  - id: BR-21
    disposition: addressed
    note: |
      play_loop.go:146 calls refresh() at init; behaviour-identical to the hand-copy, so accepted as a structural fix with no possible failing test.
  - id: BR-22
    disposition: addressed
    note: |
      viewportGesture/wrapWritten/livePrompt moved to Pure entities and newPinnedScreen to Integration points; #33 gained the widening spec in its own Revisions section.
findings:
  - id: new
    severity: Critical
    family: frame-clips-unwrapped-text
    title: |
      wrapWritten skips every escape-carrying line, and --play can only run with colour on, so a reveal's rendered definition is never wrapped
    detail: |
      This is the 5th finding in family `frame-clips-unwrapped-text`, and it was INTRODUCED by the commit that closed the fourth. Do NOT fix this instance.
      playbar.go:105-109 returns any line containing 0x1b untouched. runPlay refuses unless isTerminal(stdout) and opt.tty, and main.go:523/527 define color and tty as the same expression, so every real sitting renders with Color:true and every styled line bypasses the seam. The same commit also deleted the resize case's opt.width maintenance that had been the prior mitigation.
      MEASURED (scratch: playRig + opt.color=true, opt.width=100, newPinnedScreen(24,40), one reveal): 4 buffer lines of 55-87 cells in a 40-column terminal, e.g. "    ephemerality \x1b[35m/..../\x1b[0m noun ephemerally ..." at 87 cells. Deleting the three-line skip takes that to 0 and leaves `go test ./cmd/define/...` green, so nothing defends it; wrapText already measures with visibleCells and strings.Fields cannot split inside an escape sequence.
      THE RULE, one step past "wrap at the seam": a seam covers every write only when (a) every path into the buffer goes through it and (b) its predicate is exercised in the configuration production actually runs in. The enumeration is two-by-two and belongs in the plan: paths = liveScreen.Write (wraps, except escapes) and liveScreen.WriteRegions (screen.go:551, calls l.s.Write directly and does NOT wrap — latent for --play, live for #40's board, and already contradicting screen.go:503's "Here nothing can write around it"); line classes = plain (wraps) and escape-carrying (does not).
      Fix the class: route both methods through one private writeBuffer(text) that applies the wrap when pinned, narrow the guard to the eraseLine gesture rather than "contains any escape", and drive TestANarrowedSittingWrapsTheRestOfItself with opt.color = true.
  - id: new
    severity: Important
    family: rig-config-not-production
    title: |
      playRig hardcodes color:false, a configuration --play now refuses to run in, so every in-process sitting test drives an unreachable state
    detail: |
      play_loop_test.go:39 returns options{color: false, width: 0, ...}. Since BR-3, runPlay exits 1 unless opt.tty, and opt.tty and opt.color are the identical expression (main.go:523/527) — so no in-process test has ever exercised the only configuration a sitting can be in. That is why the Critical above shipped: Done-when 0b's two pins (TestANarrowedSittingWrapsTheRestOfItself, TestALongOptionGlossWrapsRatherThanBeingCut) both pass with colour off. The pty rows do run coloured, but TestPTYPlayResizeRepaints narrows ROWS 24->10 and never COLS, so no row narrows the width of a coloured sitting.
      THE RULE: a rig's default options have to be reachable from the flag parse of the command under test. Set color/tty true in playRig (or derive both from one helper, as main.go does) and add a column-narrowing pty row.
```

---

## Re-review — 2026-08-31T18:10:11-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 41 — play mode paints frames through screen, and gains a status bar |
| repo | tools |
| issue file | workshop/issues/000041-play-tui.md |
| boundary | whole-issue close |
| milestone | — |
| window | 94f3ad077210c544fe60b8539e2a92def29333c1..13828b3ecc4473ce0efea90a2d449960322d2e28 |
| command | sdlc close --issue 41 |
| reviewer | claude |
| timestamp | 2026-08-31T18:10:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The window delivers what the Spec and the plan promise: `--play` draws whole frames through the same `console`/`display` seam the editor uses, the bar is pinned and counted, paging works, the transcript survives, SIGWINCH repaints, and the deck+log are read exactly once per sitting. `go test ./cmd/define/...` is green (107s), and `-race` on the new play/screen rows is green. I mutation-verified three of the claimed fixes rather than reading the commit messages: reverting round 4's `eraseLine` narrowing reddens `TestWrapWrittenWrapsStyledTextButNotTheEraseGesture` (BR-23 genuinely fixed); reverting `playRig` to `color:false, tty:false` leaves the suite green (BR-24 fixed but unpinned); moving the three terminal gates back below `todaysQuestions` leaves the suite green (BR-18's code is right, its claim is undefended). What keeps this from SHIP is two cheap structural items that `#40` will consume directly — `liveScreen.WriteRegions` still writes around the wrap seam whose own comment says nothing can, and `minWrapWidth` was introduced as "the sub-20 policy in one place" while two other spellings of `< 20` survive untouched — plus a live doc instance this window's own sweep introduced. Caveat on my evidence: the pty conformance rows all SKIP in this environment ("no pty available: operation not permitted"), so I verified `-tags conformance` compiles and vets but could not exercise real hardware; the issue Log's claim that the full 124s pty suite ran green is unverified by me.

## 1. Strengths

- **The seam fix is the right shape, and the round-4 correction is real.** `liveScreen.Write` wrapping on the pinned screen (`cmd/define/screen.go:571-585`) is the version that finally covers helpers the loop doesn't own, and narrowing the exemption from "any escape" to `strings.Contains(line, eraseLine)` (`cmd/define/playbar.go:118`) is exactly the "name the case, not the mechanism" correction. Mutation-verified red without it.
- **`newConsole` and `viewportGesture` are genuinely shared, not extracted-and-abandoned.** `replraw.go:56` and `play_loop.go:107` both call `newConsole`; `replraw.go:413` and `play_loop.go:240` both call `viewportGesture`, and the editor's four-case switch is deleted rather than left beside it. That is the half BR-1 was open two rounds for.
- **`GradeOf` closes D7's DRY claim with a named function.** `schedule/progress.go:171-181` is the single rule; `gradeOf` is one line over it (`:192`) and `sittingDeck.answered` is the second caller. Grepping `GradeWrong|GradeCorrect|GradeUnaided` outside tests finds no third spelling.
- **The docs gate is met properly, not perfunctorily.** README gains the paging keys, the bar sample, and both refusals (`cmd/define/README.md:105,132-159`); atlas gains a whole "The sitting is a frame" section covering `newConsole`, `newPinnedScreen`, `viewportGesture`, `wrapWritten`, `sittingDeck` and the refusal policy.
- **`TestANarrowedSittingWrapsTheRestOfItself` asserts the class, not an instance** (`play_loop_test.go:817-834`): every line written after the resize, with a "nothing wrapped, so this test asserts nothing" guard so it can't pass vacuously.

## 2. Critical findings

None.

## 3. Important findings

**(a) `liveScreen.WriteRegions` writes around the wrap seam — `cmd/define/screen.go:626-632`.**
**This is the 6th finding in family `frame-clips-unwrapped-text`.** Do not fix it as an instance. BR-23 stated the rule as a two-by-two enumeration — paths (`Write`, `WriteRegions`) × line classes (plain, escape-carrying) — and round 4 closed the line-class axis only. `WriteRegions` calls `l.s.Write([]byte(text))` directly, so on a pinned screen it bypasses `wrapWritten` entirely, and `screen.go:566`'s "Here nothing can write around it" is now literally false. No production caller hits it today (`main.go:803` reaches it only through the editor's `newLiveScreen`), which is why the suite is green — it is latent for `--play` and live for `#40`, whose board writes regions into a pinned screen. Fix the class: one private `writeBuffer(text string)` on `liveScreen` that applies the pinned wrap, with both `Write` and `WriteRegions` routed through it, and a test that writes an over-wide region into a `newPinnedScreen` and asserts no buffer line exceeds `cols`.

**(b) `minWrapWidth` is a third spelling of the sub-20 policy, introduced by a comment claiming it is the only one — `cmd/define/playbar.go:96-101,143-146`.**
**This is the 2nd finding in family `parallel-construction`.** The rule BR-1's disposition already stated is the one that covers it: *extracting a shared owner is half the fix; the other half is deleting what it replaced.* Round 3's plan revision claims "The sub-20-column policy moved with it, so there is one number and one place", and `minWrapWidth`'s own comment says "THE SUB-20 POLICY, in one place" — but nothing moved. Measured at HEAD, three unlinked spellings: `main.go:995` (`sz.cols < 20` in `terminalWidth`), `replraw.go:399` (`if sz.cols < 20` in the editor's resize case), `playbar.go:146` (`const minWrapWidth = 20`). The enumeration this window owes, run: of the six "one place" claims it makes — `newConsole`, `viewportGesture`, `GradeOf`, `costPhrase`, `wrapWritten`, `minWrapWidth` — four are clean, `wrapWritten` is (a) above, and `minWrapWidth` is this. Fix: have `terminalWidth` and the editor's resize case both compare against `minWrapWidth` (or hoist it somewhere both can see), so the number has one owner.

## 4. Minor findings

- `atlas/define.md:2016` says `wrapWritten` is one of "three surfaces SHARED with the editor"; it is called only from the pinned branch of `liveScreen.Write`, and the same section correctly says the editor's screen does not wrap.
- `play_loop_test.go:766` still comments "the resize case keeps `opt.width` current" — that maintenance was deleted in `6201f4d`.
- `wrapText` (`render.go:623`) rebuilds through `strings.Fields`, so an over-wide line loses its internal whitespace runs (e.g. the headword line's double space). Only fires on lines that would otherwise be clipped, so it's a trade rather than a bug — worth a sentence where `wrapWritten` documents itself.

## 5. Test coverage notes

- **BR-24's rig fix is not pinned.** Reverting `playRig` (`play_loop_test.go:43`) to `options{color: false, tty: false, …}` leaves the whole `cmd/define` package green. The rule BR-24 stated — "a rig's default options must be reachable from the flag parse of the command under test" — is cheap to enforce: a subtest that runs `runPlay` with the rig's options against a fake terminal and asserts it does *not* refuse. Without it, the state that hid a Critical for a round can silently return.
- **BR-18's ordering is not pinned.** Mutation-verified: moving the stdin/stdout/`-no-color` gates below the `todaysQuestions` call leaves the package green (`TestPlayRefusesWhenStdoutIsNotATerminal` only checks exit 1, absence of `\x1b`, and a stderr substring, none of which an empty-deck stdout line breaks). A `runPlay`-on-empty-deck-to-a-non-terminal row would redden it — the coverage that existed until `TestEmptyQueueExitsZero` moved down to `todaysQuestions`.
- **No pty row narrows COLUMNS.** `TestPTYPlayResizeRepaints` (`pty_conformance_test.go:668`) narrows 24→10 rows at a fixed 80 columns, so the wrap-on-narrowing behaviour is pinned in-process only. BR-24 asked for this row; it is still absent.
- **The README's bar and summary samples are hand-maintained restatements of `costPhrase`.** `doc_sync_test.go` already does exactly this derivation for the prompt lines; extending it to `sittingBar`/`sittingSummary` would close the one remaining place where README can drift from the `-count` wording D8 exists to single-source.

## 6. Architectural notes

- **ARCH-DRY — flag.** Finding (b), and finding (a) as its second instance. Everything else consolidated well.
- **ARCH-PURE — pass.** `sittingBar`, `costPhrase`, `wrapWritten`, `livePrompt`, `GradeOf` are string/value functions unit-tested with no IO; `sittingDeck.figures` walks memory. The IO is confined to `newConsole`, `liveScreen.Paint` and the store. One note for `#33` rather than a finding: the plan files `viewportGesture` under "Pure entities" while it mutates a `display`; it is injected, so the arrangement is right, but the kind cell isn't.
- **ARCH-PURPOSE — pass, with (a) as the exception.** Shadow-sweep of the single-source claims is in finding (b). The one real deferral — mechanising `doc-sweep-incomplete` — is filed to `#33` with the full four-part widening spec written out, which is the class named rather than the instance fixed. That is the right home; `#41`'s purpose is the TUI.
- **ARCH-MOCK — pass.** `display` is faked in-process (`recordingConsole`, `paintInto`), the store is `store.Mem`, audio is `fakePlayer`/`okAudio`, and the real terminal is covered by the pty suite behind a build tag with `SkipOrFail`. Production and test flows share the `console` boundary.
- **ARCH-CONSTRAINTS — pass.** The keystroke path stays O(visible rows) behind the 16ms throttle; `refresh()` is charged per answer, not per frame; `TestASittingReadsTheDeckOnce` pins the one-`Deck()`-one-`Events()` bound with a counting store. `wrapWritten` adds O(bytes written) on a per-transition path.

## 7. Plan revision recommendations

1. **`## Revisions` → "review rounds 2 and 3"** — the sentence *"The sub-20-column policy moved with it, so there is one number and one place"* is not what the tree does. Either land finding (b) and leave the sentence, or correct it to record that `minWrapWidth` is a third spelling.
2. **Core concepts, `newPinnedScreen` row** — *"its `Write` WRAPS, which is the seam a helper cannot write around"* overstates it while `WriteRegions` exists. Land finding (a), or qualify the row.
3. **`## Revisions` → "T3/T4 landed"**, plan line 311 — names `TestAMissIsRecordedBeforeItIsRevealed`; the test is `TestAMissRecordsBeforeItPlays`. Exempt from `TestPlanNamedTestsExist` because `currentTruthOnly` strips `## Revisions`, so nothing will ever catch it.
4. **Not the plan but the same gate: `atlas/define.md:2052-2055`** still describes the per-site wrap and the resize-time `opt.width` maintenance that `6201f4d` deleted, in the document AGENTS.md §8 defines as "current state of codebase".

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Unchanged at HEAD — no `--play`/-raw usage guard in main.go, decideCapture still returns captureNothing under opt.raw (capture.go:30), and held.answered/refresh still advance the bar. Minor; never blocks.
  - id: BR-16
    disposition: not-addressed
    note: |
      Docs gate DELIVERED (README refusals+paging, atlas "The sitting is a frame") and the class correctly filed to #33 with the widening spec; but highlightwriter.go:60 now cites TestHighlightWriterShortWriteContract, a name no test has ever declared, introduced by this window's own sweep, and atlas/define.md:2052-2055 still describes the per-site wrap and resize-time opt.width update that 6201f4d deleted.
  - id: BR-18
    disposition: not-addressed
    note: |
      Code is right (gates at play_loop.go:56-77, above todaysQuestions at :79) but mutation-verified unpinned — I moved the three gates below the call in a scratch copy and the whole cmd/define package stayed green.
  - id: BR-23
    disposition: addressed
    note: |
      Mutation-verified: reverting playbar.go:118 to ContainsRune(line, 0x1b) reddens TestWrapWrittenWrapsStyledTextButNotTheEraseGesture. The path axis of BR-23's own enumeration (WriteRegions) is raised separately below.
  - id: BR-24
    disposition: addressed
    note: |
      playRig now returns color:true, tty:true (play_loop_test.go:43). Two residuals in the test-coverage notes: reverting the rig leaves the suite green, and TestPTYPlayResizeRepaints still narrows only rows.
findings:
  - id: new
    severity: Important
    family: frame-clips-unwrapped-text
    title: |
      liveScreen.WriteRegions bypasses the pinned wrap seam whose own comment says nothing can write around it
    detail: |
      6th in this family, and the path axis of BR-23's own two-by-two enumeration — round 4 closed the line-class axis (escape-carrying vs plain) and left the path axis (Write vs WriteRegions) open. screen.go:626-632 calls l.s.Write directly, so on a pinned screen it skips wrapWritten and screen.go:566's "Here nothing can write around it" is false. No production caller reaches it today (main.go:803 goes through the editor's unpinned screen), so it is latent for --play and live for #40's board. Fix the class: one private writeBuffer(text) on liveScreen applying the pinned wrap, with both Write and WriteRegions routed through it, plus a test that writes an over-wide region into a newPinnedScreen.
  - id: new
    severity: Important
    family: parallel-construction
    title: |
      minWrapWidth is a third spelling of the sub-20 policy, introduced by a comment claiming it is the only one
    detail: |
      2nd in this family, and the same rule BR-1's disposition stated: extracting a shared owner is half the fix, deleting what it replaced is the other half. playbar.go:96 says "THE SUB-20 POLICY, in one place" and the plan's round-3 revision says "there is one number and one place", but nothing moved — main.go:995 (sz.cols < 20 in terminalWidth) and replraw.go:399 (if sz.cols < 20 in the editor's resize case) are untouched by this window. Enumeration of the six "one place" claims this window makes — newConsole, viewportGesture, GradeOf, costPhrase, wrapWritten, minWrapWidth: four clean, wrapWritten is the finding above, minWrapWidth is this one. Have terminalWidth and the editor's resize case compare against the constant.
  - id: new
    severity: Minor
    family: lessons-not-recorded
    title: |
      Rounds 3 and 4 produced two transferable rules that reached the plan's Revisions but not workshop/lessons.md
    detail: |
      2nd in this family. AGENTS.md section 4 asks for the rule in lessons.md, and this window did add three entries — but they stop at round 2. The two rules the last two rounds actually bought are absent: "a guard added to protect a special case must name the case, not the mechanism it happens to use" (BR-23, the escape-vs-erase-gesture skip) and "a rig's default options must be reachable from the flag parse of the command under test" (BR-24). The second is repo-general and will recur outside cmd/define. The rule behind the family: a review round's rule lands in lessons.md, not only in the plan's Revisions, because the plan is archived with the issue and lessons.md is what the next session reads.
```
