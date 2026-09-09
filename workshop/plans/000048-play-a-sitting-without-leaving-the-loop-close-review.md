# Boundary Review — tools#48 (whole-issue close)

| field | value |
|-------|-------|
| issue | 48 — /play: a sitting without leaving the loop |
| repo | tools |
| issue file | workshop/issues/000048-play-a-sitting-without-leaving-the-loop.md |
| boundary | whole-issue close |
| milestone | — |
| window | adcbcf1841ed6139fd0cc72f5adce4ad258e9c2a..2dc2100ab103dbcf89c94bf007caadd27f0247f3 |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-09-08T16:52:21-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The command half of `/play` is exactly the shape the plan argued for — `runPlayCommand` records and returns, `runEditor` performs, and both doors provably reach one `playSession`. `liveScreen.suspend`/`resume` is a clean, minimal capability and I mutation-verified both halves of its test (dropping `l.suspended` from `repaint`'s gate and dropping `resume`'s `repaint()` each redden `TestASuspendedScreenPaintsNothingAndResumesWhereItWas`). What blocks SHIP is the third borrowed resource: the sitting borrows the REPL's `resizes` channel and *consumes* SIGWINCH shapes without ever handing them back, so a terminal resized during `/play` leaves the editor's screen painting at the pre-sitting geometry for the rest of the session — I reproduced this with a scratch test. The plan's ARCH-ORDER section and `screen.go:899`'s comment both assert the opposite. Compounding it, the behaviour this issue exists to protect is unpinned: I gutted all three of `runPlayCommand`'s refusals **and** removed the `interrupts.Set(cancel)`/`defer restore()` pair together, and the entire suite stayed green — the three unit tests the plan's Chunk 1 demanded, the pty test, and the Ctrl-C test were never written, yet every Plan and Done-when box is ticked.

## 1. Strengths

- **The borrow/take distinction is real and derived, not asserted.** `play_cmd_test.go:14-97` walks `sittingInPlace`'s callees transitively, fails closed on the walk size, *and* checks that `runPlay` would trip it — so the guard can't quietly become vacuous. Routing `/play` through `runPlay` reddens it as claimed.
- **`suspend`/`resume` is gated in exactly one place.** `screen.go:918-926` puts both flags in `repaint`, the only writer to the tty, rather than growing a second "may I paint" answer. Mutation-confirmed load-bearing (both directions).
- **The comment correction in `suspend` (`screen.go:872-878`)** is the right kind of honesty: the timer disarm is documented as hygiene after the sweep showed removing it reddened nothing, instead of keeping a claim the code doesn't earn.
- **The summary-goes-up decision** (`play_cmd.go:110-117`) correctly avoids `handBack`'s cooked-terminal precondition; `over()` is on every exit path in `playSession`, so `sitting.Stop()` always runs and nothing of the sitting's outlives the call — the plan's EXTENT claim holds.
- **`runPlayCommand` is genuinely pure** and records-not-performs, matching `cc.replay`'s precedent at `replraw.go:516-522`.

## 2. Critical findings

**C1 — `cmd/define/play_cmd.go:86` (with `screen.go:897-909`): a resize during a sitting is swallowed, and the REPL screen stays stale for the rest of the session. `ARCH-ORDER`.**

`sittingInPlace` borrows `resizes` (`replraw.go:103`), and `playSession`'s resize case applies the new shape to `con.view` — the *sitting's* screen (`play_loop.go:404` region, `view.Resize(sz.rows, sz.cols)`). Nothing writes it back to `repl`. `resume()` repaints with `l.rows/l.cols` untouched, and `watchResize` only emits on the *next* SIGWINCH, so the editor paints every subsequent frame at the pre-sitting geometry: too many rows scrolls the terminal and moves every row the screen believes it placed, wrong `cols` clips and mis-wraps, and click coordinates go off. This is the corruption class the issue was filed to avoid.

`screen.go:899` states the opposite in a doc comment — *"A resize that arrived meanwhile is already in rows/cols — the loop sets them"* — which is false precisely while the loop is inside `sittingInPlace`, the only situation `resume` exists for.

Reproduction (run in a scratch copy; fails today, 24x80 vs 40x120):

```go
d, opt, _ := playRig(t, "sycophantic")
var ttyBuf bytes.Buffer
repl := newLiveScreen(&ttyBuf, 24, 80)
resizes := make(chan winSize, 1)
resizes <- winSize{rows: 40, cols: 120}
keys := make(chan Key, 1)
go func() { time.Sleep(150 * time.Millisecond); keys <- Key{Kind: KeyInterrupt}; close(keys) }()
sittingInPlace(t.Context(), d, opt, keys, &interrupter{}, repl, resizes, &ttyBuf, &bytes.Buffer{})
r, c := repl.Size() // 24, 80 — want 40, 120
```

Fix sketch: hand the shape back before resuming —

```go
defer func() {
    r, c := sitting.Size()   // the sitting tracked every resize it consumed
    repl.Resize(r, c)        // Resize does not paint, so this is safe while suspended
    repl.resume()
}()
```

and correct `screen.go:899` to say the *caller* must restore the shape before `resume`. Ship the test above with it.

## 3. Important findings

**I1 — `cmd/define/play_cmd.go:23-49`: all three of `runPlayCommand`'s refusals are unpinned. `ARCH-PURE`.**
Mutation-proven: I replaced the whole body with `if c.startSitting == nil { return 0 }; c.startSitting(); return 0` — dropping the argument refusal, the no-terminal sentence, the no-deck sentence and both non-zero exit codes — and the full `./cmd/define` suite stayed green (only the git-dependent repo guards fail in a `.git`-less scratch copy, and they fail on the pristine copy too). Grep confirms no test file names `runPlayCommand` or sets `startSitting`. The plan's Chunk 1 Task 1 Step 1 named exactly these three tests; none exist, and the step is ticked. This is the one PURE entity in the change and the cheapest thing in the diff to test.

**I2 — `cmd/define/play_cmd.go:97-98`: the scoped interrupt — the issue's headline Done-when — has no test at all. `ARCH-ORDER`.**
In the same mutation run I also deleted `restore := interrupts.Set(cancel)` and `defer restore()`. Nothing reddened. Without that pair, Ctrl-C during a sitting fires the *loop's* cancel and kills `define` — the exact regression Done-when 2 exists to prevent — and the suite reports success. Plan Task 2 Step 2 (pty: `/play`, answer, Ctrl-C, back at the prompt), Step 4 (Ctrl-C does not cancel the loop's context) and Verification item 4 all claim this; `pty_conformance_test.go` is untouched in the window, and `sittingInPlace` and the `runEditor` dispatch at `replraw.go:521-531` have zero coverage. The only evidence is the operator's manual smoke test recorded in the issue Log. At minimum, add a non-pty test driving `sittingInPlace` with a scripted interrupt that asserts (a) the outer ctx is not cancelled and (b) `interrupts.Fire()` after the call reaches the loop's cancel again.

**I3 — `cmd/define/play_cmd_test.go:145-183`: `TestASittingRecordsTheSameEventFromEitherDoor` never exercises either door.**
The `answer` closure calls `playSession` directly; `runPlay`, `sittingInPlace` and `runPlayCommand` appear nowhere in it. It runs one function twice with identical inputs and compares the results to each other — a determinism check, not a two-door comparison. The Done-when it is cited against ("Everything a sitting records is recorded identically from either entry point... asserted through the store") is therefore carried entirely by the AST guard at `play_cmd_test.go:101-142`. Either rename it to what it pins (`playSession` records a reviewed event carrying a `Form`) or parameterise it over the two doors so the comparison is real.

**I4 — `atlas/define.md`: atlas update appears missing for the new surface (AGENTS.md §8).**
Only the `/play` command-table row was added (`atlas/define.md:1119`). Not recorded:
- `console.newSitting` — the type table at `atlas/define.md:273` still reads `console  display + resizes + finish + stdout + stderr`, and is now incomplete.
- `liveScreen.suspend`/`resume` — a genuinely new capability ("two screens over one terminal") with no entry beside `handBack`/`Stop` at `atlas/define.md:270-275`.
- `sittingInPlace` as a second door into a sitting, and the fact that the *"`newConsole` is the shared builder both loops now call"* claim (`atlas/define.md:2562`, `2579`) now has a deliberate exception — a console assembled by hand, precisely the "second way to draw" that section warns against. The *reason* it is right here belongs in the atlas, not only in a function comment.

README is fine — `cmd/define/README.md:724-730` covers the new user-facing surface including the Ctrl-C rule.

## 4. Minor findings

- `screen.go:899` — the `resume` doc asserts the resize is already applied; false (see C1). Fix with C1.
- Plan tables never gained a `suspend`/`resume` row, although the PQ-5 revision claims *"It has all three now… an entity row"*; and the `sittingInPlace` row's signature (`…, sess, keys, …`) is stale — the shipped one takes `repl`/`resizes`/`tty` and no `sess`, which is better than planned but no longer matches. `TestPlanTablesNameEntitiesThatExist` passes because it checks only one direction.
- `replraw.go:100-104` — `newSitting` is installed on *every* console `newConsole` builds, including `--play`'s and `#40`'s board, where a nested sitting cannot run; the field's own doc (`replraw.go:165-168`) says "nil where one cannot run". Unreachable today, but the invariant the doc states is already false.
- `play_cmd.go:44-47` vs `play_loop.go:29-31` — `/play` on a deckless directory prints `noDeckMessage` to **stderr** and returns **1**; `--play` prints a different sentence to **stdout** and returns **0**. Two doors, two answers to the same question.
- `play_cmd_test.go:172-183` re-implements `reviewEvents` (`play_loop_test.go:81`) inline in the same package, and the AST-parse preamble is duplicated between the two guards at `play_cmd_test.go:23-58` and `:104-119` (`ARCH-DRY`).
- `replraw.go:527-530` — the `/play` branch omits the `fmt.Fprint(stdout, "\r\n")` that both the plain-command and `/pron` paths write before `draw()`. Cosmetic; the transcript happens to end in a newline.
- The estimate's `cross-cutting-refactor 0.03/0.16` row is described as *"extracting `runPlay`'s shared half"*; no extraction happened — both doors independently call `todaysQuestions` then `playSession`. That is fine as a design (the shared halves are already functions), but the row describes work not done.

## 5. Test coverage notes

- Verified good: the suspend/resume pair is mutation-tight in both directions, and `TestResumeDoesNotReviveAStoppedScreen` correctly pins that `resume` cannot un-`Stop` a screen.
- Verified absent: `runPlayCommand` (all three behaviours), `sittingInPlace` (any behaviour), the `runEditor` `/play` dispatch, the scoped interrupt, and any resize-during-sitting case. One combined mutation removing five behaviours left the suite green.
- The pty conformance suite is the live-conformance check for real-terminal behaviour (`ARCH-MOCK`); `/play` is the first thing in this binary to run two screens over one tty and it was not added to it. That is where the resize interleaving in C1 would have shown up.
- `ARCH-ORDER` at-review lens: the shipped tests observe exactly one interleaving of the sitting (none, in fact — no test starts one). There is no seam for injecting arrival order, which is why the resize case was invisible; the scratch test in C1 shows a `resizes` channel pre-loaded before the call is a perfectly good seam and costs ~20 lines.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass in production code (one `playSession`, one `todaysQuestions`, one `repaint` gate); flagged only in tests.
- **ARCH-PURE** — pass structurally: `runPlayCommand` is pure, `sittingInPlace` is the thin shell, `runEditor` still has no `rawSession`. The payoff went unclaimed — the pure entity is the untested one (I1).
- **ARCH-PURPOSE** — the *runtime* purpose is delivered; the *test* surface the plan committed to was reduced to its cheap subset (three static AST guards + one screen unit test) and the checkboxes ticked anyway. The class here is broader than any single missing test: `lessons.md`'s own new entry says *"enumerate what the ordinary constructor ACQUIRES and answer each one."* `newConsole` acquires **four** things, not three — `finish`, `watchResize`, `enterMouse`, and *the loop's exclusive read of the resize channel*. C1 is the fourth acquisition, unanswered. Worth amending the lesson.
- **ARCH-MOCK** — no new external dependency; existing seams reused. Gap is the missing pty conformance row.
- **ARCH-CONSTRAINTS** — pass. `todaysQuestions` is per-`/play`, not per-keystroke; the suspend gate keeps the throttled painter off the tty; nothing unbounded is spawned.
- **ARCH-SECURE** — N/A: no new untrusted input, no credentials, no new persisted artifact.
- **ARCH-ORDER** — flagged (C1, I2). For `#7`'s form and any third full-screen surface: the "borrow the loop's channel" pattern needs a stated rule about *returning* what was consumed, or the next borrower repeats this.

## 7. Plan revision recommendations

`workshop/plans/000048-play-from-the-loop-plan.md` needs a `## Revisions` entry covering:

1. **The ARCH-ORDER resize claim is wrong.** The plan states *"the suspended screen takes the new shape on `resume`, which repaints unconditionally anyway"* and *"A resize during a sitting therefore arrives on that one channel… the suspended screen takes the new shape on `resume`."* It does not; `resume` repaints at the stale shape. Record the correction and the shape hand-back.
2. **Tests named but not written.** Chunk 1 Task 1 Step 1's three unit tests, Chunk 2 Task 2 Step 2's pty test, Step 4's Ctrl-C test, and Step 7's *"Remove the `Set`/`restore` pair and confirm the Ctrl-C row reddens"* sweep — there is no Ctrl-C row to redden. Verification items 2 and 4 claim them. Either write them or record what shipped instead and why.
3. **The entity tables are stale.** Add the `suspend`/`resume` row PQ-5's revision claims is already there, and correct `sittingInPlace`'s signature (no `sess`; takes `repl`, `resizes`, `tty`).
4. **The estimate's `cross-cutting-refactor` row** describes an extraction that did not happen — note that both doors reach the shared halves directly, so no helper was needed.

```findings
findings:
  - id: new
    severity: Critical
    family: borrowed-channel-swallows-owners-events
    title: |
      a resize consumed during a sitting is never handed back, so the REPL screen paints at a stale shape for the rest of the session
    detail: |
      sittingInPlace borrows the loop's resizes channel (play_cmd.go:71, replraw.go:103) and playSession
      applies each shape to the sitting's screen only. Nothing writes it back to repl, and resume()
      (screen.go:901) repaints with rows/cols untouched; watchResize only emits on the NEXT SIGWINCH, so the
      editor is stuck at the pre-sitting geometry - too many rows scrolls the terminal, wrong cols mis-wraps
      and puts click coordinates off. screen.go:899 asserts the opposite in a doc comment. Reproduced: a
      scratch test that pre-loads resizes with 40x120, runs sittingInPlace to a scripted interrupt, and reads
      repl.Size() gets 24x80. Fix: capture sitting.Size() and repl.Resize(r, c) before repl.resume(), and
      correct the comment.
  - id: new
    severity: Important
    family: plan-named-test-not-written
    title: |
      all three of runPlayCommand's refusals are unpinned - the plan's Chunk 1 tests were never written
    detail: |
      Mutation-proven: replacing runPlayCommand's body (play_cmd.go:23-49) with a bare nil check plus
      c.startSitting() - dropping the argument refusal, the no-terminal sentence, the no-deck sentence and
      both non-zero exit codes - leaves the whole ./cmd/define suite green. No test file names runPlayCommand
      or sets startSitting. Chunk 1 Task 1 Step 1 named exactly these three tests and the step is ticked.
  - id: new
    severity: Important
    family: plan-named-test-not-written
    title: |
      the scoped interrupt has no test - removing Set/restore reddens nothing
    detail: |
      In the same mutation run, deleting restore := interrupts.Set(cancel) and defer restore()
      (play_cmd.go:97-98) reddened nothing. Without them Ctrl-C during a sitting fires the loop's cancel and
      kills define, which is the regression Done-when 2 exists to prevent. pty_conformance_test.go is
      untouched in this window; sittingInPlace and the runEditor dispatch (replraw.go:521-531) have zero
      coverage. Plan Task 2 Steps 2, 4 and 7 and Verification item 4 all claim this.
  - id: new
    severity: Important
    family: test-name-overclaims-what-it-pins
    title: |
      TestASittingRecordsTheSameEventFromEitherDoor never invokes either door
    detail: |
      play_cmd_test.go:145-183 calls playSession directly twice with identical inputs and compares the two
      results - a determinism check, not a comparison of entry points. runPlay, sittingInPlace and
      runPlayCommand appear nowhere in it, so the Done-when it is cited against rests entirely on the AST
      guard at play_cmd_test.go:101-142. Rename it to what it pins, or parameterise it over the two doors.
  - id: new
    severity: Important
    family: atlas-stale-for-new-surface
    title: |
      atlas update appears missing for console.newSitting, liveScreen.suspend/resume and sittingInPlace
    detail: |
      Only the /play command-table row was added (atlas/define.md:1119). The type table at atlas/define.md:273
      still lists console as display + resizes + finish + stdout + stderr; suspend/resume gets no entry beside
      handBack/Stop; and the "newConsole is the shared builder both loops now call" claim at
      atlas/define.md:2562 and 2579 now has a deliberate hand-assembled exception whose rationale lives only
      in a function comment.
  - id: new
    severity: Minor
    family: plan-table-stale
    title: |
      the plan's entity tables have no suspend/resume row and a stale sittingInPlace signature
    detail: |
      PQ-5's revision claims suspend/resume now has "an entity row"; neither the Pure entities nor the
      Integration points table contains one. The sittingInPlace row still reads (ctx, d, opt, sess, keys,
      interrupts, stdout, stderr); the shipped function takes repl, resizes and tty and no sess.
      TestPlanTablesNameEntitiesThatExist passes because it only checks one direction.
  - id: new
    severity: Minor
    family: capability-wider-than-its-doc
    title: |
      newSitting is installed on every console newConsole builds, including --play's and the board's
    detail: |
      replraw.go:100-104 sets con.newSitting unconditionally, so --play's pinned console and #40's board carry
      a non-nil sitting factory where a nested sitting cannot run - contradicting the field's own doc at
      replraw.go:165-168 ("nil where one cannot run"). Unreachable today; only runEditor reads it.
  - id: new
    severity: Minor
    family: refusal-wording-diverges
    title: |
      --play and /play give different sentences, streams and exit codes for a deckless directory
    detail: |
      play_loop.go:29-31 prints "no deck in this directory, so there is nothing to review" to stdout and
      returns 0; play_cmd.go:44-47 prints noDeckMessage to stderr and returns 1. Two doors, two answers to the
      same question.
  - id: new
    severity: Minor
    family: test-helper-duplicated
    title: |
      play_cmd_test.go re-implements reviewEvents and duplicates the AST-parse preamble
    detail: |
      The inline closure at play_cmd_test.go:172-183 is reviewEvents (play_loop_test.go:81) in the same
      package, and the ParseDir + FuncDecl walk is copied between play_cmd_test.go:23-58 and :104-119
      (ARCH-DRY).
```
