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

---

## Re-review — 2026-09-08T18:33:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 48 — /play: a sitting without leaving the loop |
| repo | tools |
| issue file | workshop/issues/000048-play-a-sitting-without-leaving-the-loop.md |
| boundary | whole-issue close |
| milestone | — |
| window | adcbcf1841ed6139fd0cc72f5adce4ad258e9c2a..15ecac4d633944a95ea21fbf1dd776ee212bd200 |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-09-08T18:33:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's Critical (BR-5) is genuinely fixed and I proved it by mutation — deleting `repl.Resize(sitting.Size())` reddens `TestTheEditorScreenTakesTheShapeTheSittingEndedWith` with the exact sentence it was written for. So is BR-6: gutting `runPlayCommand` to a bare nil check reddens three of four subtests. The suite is green under default and `pty`, `-race` is clean on all eight new tests, `go vet`/`gofmt` clean, and the four refusal paths behave as documented when the real binary is run. What blocks SHIP is that **BR-7's claimed fix does not hold**: with `restore := interrupts.Set(cancel)` and `defer restore()` both deleted, `TestASittingHandsTheInterruptBack` still passes — because `keysFor("^")` feeds `Key{KeyInterrupt}` straight into the channel, which is the exact flaw the test's own comment says version one had, and because the test installs the loop's cancel itself so `Fire()`'s `consumed` is true regardless. Only the `restore()` half is pinned (dropping it alone does redden). Alongside that, BR-5's fix hands back one of the two effects the owner's resize handler applies: `view.Resize` yes, `opt.width` no — so a SIGWINCH consumed by a sitting leaves every definition looked up afterwards laid out at the pre-sitting width. Five prior findings (BR-1, BR-2, BR-3, BR-10, BR-11, BR-12, BR-13) got no change at all this round, and the durable plan is unchanged and 0/15 ticked.

## 1. Strengths

- **`TestASittingFromTheLoopNeverEntersRawMode`** (`cmd/define/play_cmd_test.go:29`) is the best test in the window: a transitive callee walk that fails closed on the walk's size (`len(seen) < 5`) *and* carries an explicit non-vacuity check that `runPlay` would still trip it. Derived, not listed, and it knows it could certify nothing.
- **BR-5's fix is real.** `play_cmd.go:103-104` + `play_cmd_test.go:195` — mutation-verified red. The test forces the ordering (resize consumed before the interrupt) rather than hoping for it.
- **BR-6's fix is real.** `TestSlashPlayRefusals` (`play_cmd_test.go:284`) — mutation-verified: three of four subtests redden when the refusals are removed, on both the sentence and the exit code.
- **One gate decides whether a frame goes out.** `screen.go:922` puts `stopped` and `suspended` in the single `repaint` guard, and `TestResumeDoesNotReviveAStoppedScreen` (`screen_test.go:1625`) pins the `stopped ∧ suspended` corner. This is the ARCH-ORDER-correct shape: two flags, one transition point, no unwritten legal combinations.
- **`sittingInPlace` takes its terminal as parameters** (`play_cmd.go:70-72`), which is exactly what let BR-8's fix drive the real door over a `bytes.Buffer` and assert through the store. The thin-shell boundary paid off immediately.
- **Refusal shape matches the house convention** — exit 2 + `"it takes no arguments, not %q"` mirrors `/stats` (`stats.go:232-234`).

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `opt.width` is the second effect of the borrowed resize, and it is not handed back.** `play_cmd.go:103`

*This is the 2nd finding in family `borrowed-channel-swallows-owners-events`.* Per the escalation rule I am not asking for this instance to be patched — state the rule and sweep the enumeration.

The rule: **when a borrower consumes an owner's event stream, it owes back every effect the owner's handler would have applied — enumerate that handler's body and mirror it, or hand the raw event back instead of one derived value.** "Borrow the channel" reads as sharing and is not; the lesson already written at `workshop/lessons.md:4048` says so, and then hands back one of two.

The enumeration is small and exact. The owner's handler is `replraw.go:404-430` and it applies **two** pieces of state: the wrap policy (`opt.width = sz.cols`, clamped to 0 below `minWrapWidth`, lines 424-427) and the screen shape (`view.Resize(sz.rows, sz.cols)`, line 428). `sittingInPlace` hands back the second (`repl.Resize(sitting.Size())`) and not the first — and it cannot, because `opt` reaches it by value through `con.newSitting(ctx, d, opt, …)` (`replraw.go:532`) and again by value into `playSession`. Consequence: after a SIGWINCH consumed by a sitting, `lookupAndRender(d, opt, …)` (`replraw.go:685`) renders every subsequent definition at the pre-sitting width and `newCommandCtx(d, opt, …)` (`replraw.go:495`) lays out every subsequent command's output at it, for the rest of the session. On a narrowed terminal `Paint`'s `clipVisible` then cuts the over-wide lines at the right edge; on a widened one the text stays needlessly narrow. Either way it persists until the next SIGWINCH.

Fix sketch for the rule, not the row: give `newSitting` a way to return the shape it ended with (or take `*options`), and make the hand-back derive from the owner's handler rather than restate a piece of it — e.g. factor `replraw.go:424-428` into an `applyResize(sz, &opt, view)` that both the owner's case and the sitting's hand-back call, so a third effect added later is handed back by construction.

**I-2 — the durable plan is 0/15 ticked at close, which also exempts its two new-entity rows from the guard that would have caught BR-3 and BR-10.** `workshop/plans/000048-play-from-the-loop-plan.md`

*This is the 2nd finding in family `plan-table-stale`.* State the rule rather than editing the rows.

The rule: **at a close boundary the durable plan is reconciled with what shipped, in one sweep — checkboxes, entity rows, signatures and test names — because guards that read the plan key off "is this plan still in progress".** `repo_guard_test.go:797` reads `inProgress := strings.Contains(body, "- [ ] ")` and then `if inProgress && status == "new" { continue }`. The plan has 15 unticked steps and 0 ticked, so **both** `new` rows — `runPlayCommand` and `sittingInPlace` — are skipped outright. That is the mechanism by which BR-10's stale signature and BR-3's stale row survived a green suite; they were never checked, not checked-and-passed.

Same sweep, same rule, three more artifacts still contradicting the code: the `sittingInPlace` row's signature is `(ctx, d, opt, sess, keys, interrupts, stdout, stderr)` against a shipped `(ctx, d, opt, keys, interrupts, repl, resizes, tty, stderr)`; there is still no entity row for `suspend`/`resume` despite PQ-5's revision claiming one; and the Verification section now backticks **no** test names at all, having been de-backticked in `adcbcf1` "until the tests land" — the tests have landed, and `TestPlanCitesTestsThatExist`'s own error text says "write it, or cite the one that shipped."

**I-3 — BR-7 re-raised: the interrupt scope is still unpinned.** See §Dispositions; detail below under prior findings.

## 4. Minor findings

- **M-1 — the `console.newSitting` seam has a redundant parameter and a name that does not describe it.** `replraw.go:101-104,168-169`. The closure captures `live`, and `runEditor` passes `stderr := con.stderr`, which *is* `live` — so the sixth parameter is always the value already captured. And `newSitting` is `func(...) int` that runs a whole sitting and returns an exit code; the plan specified `newSitting func() console`, a constructor. A `new*` name for a verb that performs is the one thing a reader of `runEditor:532` cannot guess. Rename to `runSitting` and drop the `stderr` parameter.

## 5. Test coverage notes

- **Mutation results this round.** `repl.Resize(sitting.Size())` removed → red (good). `runPlayCommand` body gutted → red (good). `interrupts.Set(cancel)` + `defer restore()` both removed → **green** (bad). `defer restore()` alone removed → red. So the coverage is exactly half of the property: the hand-back is pinned, the taking is not.
- **The pty test the plan names still does not exist.** `pty_conformance_test.go` has 17 `TestPTY*` functions and none mentions `/play` from the loop; the plan's Task 2 Step 2 and Verification item 4 both claim it. What actually validated the end-to-end path is the operator's hand-run, which is Verification item 6.
- **No test drives `runEditor`'s `/play` dispatch** (`replraw.go:521-534`) — the `if con.newSitting != nil` gate, the `sitting` flag and the post-sitting `draw(); continue` are all uncovered. That block is where I-1 lives.
- `-race -count=2` on all eight new tests is clean.
- Nit: `play_cmd_test.go:214-220` busy-polls `len(resizes)` at 1ms for up to 2s. It works, but it is a wall-clock loop in a suite that already takes 110s.

## 6. Architectural notes

- **ARCH-DRY — flag.** Three open instances, all prior findings: `reviewEvents` re-implemented inline at `play_cmd_test.go:169-181` while the identical helper sits at `play_loop_test.go:81` in the same package; the `ParseDir`+`FuncDecl` preamble copied between `play_cmd_test.go:31-63` and `:106-131`; and one refusal condition with two sentences (`play_loop.go:30` vs `play_cmd.go:45`).
- **ARCH-PURE — pass.** `runPlayCommand` is pure over `commandCtx` and its tests run with `io.Discard`; the IO is in `sittingInPlace`, which takes the terminal as parameters rather than reaching for it. That separation is what made BR-8's fix possible without a mock.
- **ARCH-PURPOSE — flag.** The shadow-sweep on "one refusal condition, one sentence from one helper": six sites derive from `noDeckMessage`, one hand-rolls it, and the round that added the seventh consumer did not sweep the one hand-maintained restatement. The class was named in BR-1; only the instance the plan happened to touch derives.
- **ARCH-MOCK — pass.** `store.NewMem()` is the stateful fake, `fakePlayer` stands in for `afplay(1)`, and the sitting runs over a `bytes.Buffer` tty through the same `console` boundary production uses. No new external dependency; nothing shells out outside the seam.
- **ARCH-CONSTRAINTS — pass.** The sitting adds one console build and one `todaysQuestions` (two reads) per invocation, nothing per keystroke; `suspend`/`resume` is O(1) and disarms the throttle timer. Envelope respected.
- **ARCH-SECURE — N/A, stated.** No untrusted input is parsed, no credential is touched, and the tests use in-memory stores and `t.Context()` rather than real user state.
- **ARCH-ORDER — flag, and it is the highest-leverage one here.** `TestASittingHandsTheInterruptBack` can only observe one interleaving, and it is the wrong transport: the interrupt arrives as a synthesised `Key` on the channel rather than through `interrupter.Fire()`, which is how `readKeys` actually delivers it and how `consumed` suppresses the key. There is no seam that drives Ctrl-C the way production does, so the green run is a sample of size one from a path the code does not take. The state modelling itself is good (one gate, both flags, the `stopped ∧ suspended` corner pinned) — it is the *oracle* that is missing.

## 7. Plan revision recommendations

Add one `## Revisions` entry, **2026-09-08 — close boundary, round 3**, containing:

1. **The plan is reconciled with what shipped.** Tick the 15 steps that landed and say plainly which did not: Task 2 Step 2's pty test was never written (the operator's hand-run covered it instead), and Step 1's "extract what `runPlay` and this share" was answered by both doors calling the existing `playSession` rather than by a new helper — `play_loop.go` is unmodified.
2. **The `sittingInPlace` row's signature** is corrected to `(ctx, d, opt, keys, interrupts, repl *liveScreen, resizes <-chan winSize, tty, stderr io.Writer) int` — no `sess`, no bare `stdout`.
3. **A `liveScreen.suspend`/`resume` row** is added to the entity tables (`cmd/define/screen.go`, modified), which PQ-5's revision claimed and never delivered.
4. **The Verification section re-backticks the tests that shipped** — `TestSlashPlayRefusals`, `TestASittingHandsTheInterruptBack`, `TestTheSittingDoorRecordsWhatItAnswers`, `TestTheEditorScreenTakesTheShapeTheSittingEndedWith`, `TestASittingFromTheLoopNeverEntersRawMode`, `TestBothEntryPointsReachOnePlaySession`, `TestASuspendedScreenPaintsNothingAndResumesWhereItWas`, `TestResumeDoesNotReviveAStoppedScreen` — and drops the "deliberately un-backticked" note, which was true before the code landed and is now the thing making the plan cite nothing.
5. **Step 8 names its surface**: `cmd/define/README.md`, the per-command paragraphs at :721-760 — this repo's own convention at `doc_sync_test.go:376-380` (BR-2's rule).
6. **The borrowed-channel rule** from I-1 is written into the ARCH-ORDER section: the effects the owner's resize handler applies are enumerated, and the hand-back mirrors all of them.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      play_loop.go:30 still hand-rolls the sentence; verified by running both doors — --play prints "no deck in this directory, so there is nothing to review" to stdout and exits 0, /play prints noDeckMessage to stderr and exits 1. No rule stated anywhere.
  - id: BR-2
    disposition: not-addressed
    note: |
      README.md:724-730 did get its paragraph, but the plan's Step 8 is still the bare "README + atlas" — the rule the finding asked for was never stated.
  - id: BR-3
    disposition: not-addressed
    note: |
      The plan is unmodified in this window; the Integration-points row still reads (ctx, d, opt, sess, keys, interrupts, stdout, stderr). The code got it right, so the predicted defect did not ship — only the artifact contradicts it.
  - id: BR-4
    disposition: addressed
    note: |
      Fixed in the window's base commit adcbcf1; TestPlanCitesTestsThatExist is green. The inverse is now true and folded into the new plan-table-stale finding: the plan cites no shipped test name at all.
  - id: BR-5
    disposition: addressed
    note: |
      Mutation-verified: deleting play_cmd.go:103 reddens TestTheEditorScreenTakesTheShapeTheSittingEndedWith with its own sentence. Only the screen shape is handed back, though — see the new opt.width finding.
  - id: BR-6
    disposition: addressed
    note: |
      Mutation-verified: replacing runPlayCommand's body with a bare nil check plus c.startSitting() reddens three of TestSlashPlayRefusals' four subtests, on sentence and exit code.
  - id: BR-7
    disposition: not-addressed
    note: |
      Mutation re-run: with BOTH restore := interrupts.Set(cancel) and defer restore() deleted, TestASittingHandsTheInterruptBack still PASSES. keysFor("^") puts Key{KeyInterrupt} straight on the channel, bypassing the interrupter — the flaw the test's own comment attributes to version one — and the test installs the loop's cancel itself, so Fire()'s consumed is true either way. Only the restore half is pinned (dropping defer restore() alone does redden). The pty test the plan names at Task 2 Step 2 and Verification item 4 still does not exist; pty_conformance_test.go has no /play case.
  - id: BR-8
    disposition: addressed
    note: |
      Renamed to TestTheSittingDoorRecordsWhatItAnswers and now drives sittingInPlace over a buffer with a scripted key channel, asserting through store.Events rather than a fake.
  - id: BR-9
    disposition: addressed
    note: |
      atlas/define.md:2605-2626 now names sittingInPlace, the three acquisitions a borrower must not take, suspend/resume, the shape hand-back and the scoped interrupt. Residual: the console block at atlas/define.md:268-274 still enumerates five fields and omits newSitting.
  - id: BR-10
    disposition: not-addressed
    note: |
      Plan unmodified. Root cause now measured: repo_guard_test.go:797 skips status "new" rows while any "- [ ] " remains, and the plan is 0/15 ticked — so both new rows are exempt rather than passing. Rolled into the new plan-table-stale finding.
  - id: BR-11
    disposition: not-addressed
    note: |
      replraw.go:101-104 still sets con.newSitting unconditionally, so --play's console (play_loop.go:106) carries a factory the field's own doc at replraw.go:166-169 says is nil where a sitting cannot run.
  - id: BR-12
    disposition: not-addressed
    note: |
      Confirmed by running the built binary in a deckless directory: --play says "no deck in this directory, so there is nothing to review" on stdout, exit 0; /play says noDeckMessage on stderr, exit 1, and names DEFINE_NO_CAPTURE where --play does not.
  - id: BR-13
    disposition: not-addressed
    note: |
      Both duplications survive: play_cmd_test.go:169-181 re-implements reviewEvents (play_loop_test.go:81), and the ParseDir + FuncDecl preamble is still copied between play_cmd_test.go:31-63 and :106-131.
findings:
  - id: new
    severity: Important
    family: borrowed-channel-swallows-owners-events
    title: |
      the borrowed resize hands back the screen shape but not opt.width, so entries looked up after a sitting wrap at the pre-sitting width
    detail: |
      This is the 2nd finding in family borrowed-channel-swallows-owners-events, so the rule
      rather than the instance. The rule: a borrower that consumes an owner's event stream owes
      back EVERY effect the owner's handler applies - enumerate that handler and mirror it, or
      hand the raw event back instead of one derived value. The enumeration here is exact and
      small. The owner's handler (replraw.go:404-430) applies two pieces of state: the wrap
      policy (opt.width = sz.cols, clamped to 0 below minWrapWidth, lines 424-427) and the
      screen shape (view.Resize, line 428). BR-5's fix hands back the second only, and cannot
      hand back the first - opt reaches sittingInPlace by value through con.newSitting
      (replraw.go:532) and again by value into playSession. So a SIGWINCH consumed by a sitting
      leaves lookupAndRender (replraw.go:685) and newCommandCtx (replraw.go:495) using the
      pre-sitting width for the rest of the session; on a narrowed terminal Paint's clipVisible
      then cuts the over-wide lines at the right edge. Fix the rule: factor lines 424-428 into
      one applyResize(sz, &opt, view) that both the owner's case and the sitting's hand-back
      call, so a third effect added later is handed back by construction.
  - id: new
    severity: Important
    family: plan-table-stale
    title: |
      the durable plan is 0/15 ticked at close, which exempts its two new-entity rows from the guard that would have checked them
    detail: |
      This is the 2nd finding in family plan-table-stale, so the rule rather than the rows. The
      rule: at a close boundary the durable plan is reconciled with what shipped in ONE sweep -
      checkboxes, entity rows, signatures, test names - because the guards that read plans key
      off "is this plan still in progress". Measured: repo_guard_test.go:797 does
      `if inProgress && status == "new" { continue }`, and inProgress is
      `strings.Contains(body, "- [ ] ")`. The plan has 15 unticked steps and 0 ticked, so BOTH
      new rows (runPlayCommand, sittingInPlace) are skipped outright - that is how BR-3's and
      BR-10's stale rows survived a green suite. Same sweep, same rule: the sittingInPlace row's
      signature still names sess and stdout against a shipped (ctx, d, opt, keys, interrupts,
      repl, resizes, tty, stderr); there is still no suspend/resume entity row despite PQ-5's
      revision claiming one; and the Verification section now backticks no test name at all,
      de-backticked in adcbcf1 "until the tests land" - they have landed, and
      TestPlanCitesTestsThatExist's own message says "write it, or cite the one that shipped".
      Also record what did not ship: the pty test at Task 2 Step 2, and Step 1's shared-half
      extraction (play_loop.go is unmodified).
  - id: new
    severity: Minor
    family: new-seam-surface-unshaped
    title: |
      console.newSitting takes a parameter it already captures, and is named like a constructor while running a whole sitting
    detail: |
      replraw.go:101-104 builds the closure over `live`, and runEditor passes stderr := con.stderr
      (replraw.go:532), which IS `live` - so the sixth parameter is always the value already
      captured and can be dropped. Separately, the field is `func(...) int` that performs a
      sitting and returns its exit code, while the plan specified `newSitting func() console`, a
      constructor; a `new*` name on a verb is the one thing a reader of the dispatch at
      replraw.go:532 cannot guess. Rename to runSitting and drop the redundant writer. This is a
      newly-introduced internal seam that downstream work will consume, so the surface is worth
      settling now.
```

---

## Re-review — 2026-09-08T18:56:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 48 — /play: a sitting without leaving the loop |
| repo | tools |
| issue file | workshop/issues/000048-play-a-sitting-without-leaving-the-loop.md |
| boundary | whole-issue close |
| milestone | — |
| window | adcbcf1841ed6139fd0cc72f5adce4ad258e9c2a..c2e593a4ac48cb143671988fd1d96e920f2c7d97 |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-09-08T18:56:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The terminal-ownership work is genuinely good and I mutation-verified three of its fixes: BR-5 (`repl.Resize(r, c)` removed → `TestTheEditorScreenTakesTheShapeTheSittingEndedWith` reddens), BR-14 (`applyShape` bypassed → `TestBothShapeRoutesGoThroughOnePlace` reddens on both clauses), BR-7 (`Set`/`restore` deleted → `TestASittingScopesTheInterruptAndHandsItBack` reddens at "the sitting did not end"). What blocks SHIP is a regression in the closing commit itself: `c2e593a` **deleted `TestSlashPlayRefusals`**, the test written one commit earlier to close BR-6. `runPlayCommand` now has zero references from any test, and I confirmed by mutation that deleting the `c.startSitting == nil` refusal — which turns the next line into a nil-func call — leaves the entire suite green. A second mutation showed the same for the loop half: replacing `cc.startSitting = func() { sitting = true }` (replraw.go:540) with a no-op makes `/play` do nothing at the prompt, and nothing reddens. So the issue's headline surface — the command row, its three refusals, and the record-then-perform wiring — ships unpinned while the plan's 15/15 ticked boxes and the issue's ticked Done-when rows claim otherwise. The repo's guard that would normally catch a vanished `Test*` (`repo_guard_test.go:1560`) diffs `merge-base..HEAD`, so a test born and killed inside the same window is invisible to it.

## 1. Strengths

- **`applyShape` (replraw.go:50-55) is the right answer to BR-14** — not "hand back one more field" but one function that *is* what a shape means, with `TestBothShapeRoutesGoThroughOnePlace` deriving the route set from the AST so a third route is covered on arrival. Both clauses of that guard redden under mutation. This is ARCH-DRY done as a class fix rather than an instance fix.
- **The interrupt test finally pins the property honestly** (play_cmd_test.go:246-300). Ordering is forced through an observable channel read (`len(resizes)`) rather than a sleep or a buffer race, and both halves assert — scope consumed *and* restored. Clean under `-race -count=3`.
- **`suspend`/`resume` is a real capability with a real spec** (screen.go:857-909). One gate in `repaint`, `pending` preserved, resume paints unconditionally, and `TestResumeDoesNotReviveAStoppedScreen` pins the one illegal transition. The comment correcting itself about the timer being hygiene rather than the guard (screen.go:869-875) is exactly the discipline this repo asks for.
- **`sittingInPlace`'s signature landed right** (play_cmd.go:70-72): `repl *liveScreen`, borrowed `resizes`, and the real `tty` — not the `sess`/`stdout` pair the plan's row still names. The fabricated-80x24 hazard PQ-2/BR-3 warned about did not materialise.
- **The atlas entry (atlas/define.md:2605-2626) is the borrow design stated once, correctly**, including the non-obvious "takes the sitting's final shape before resuming".

## 2. Critical findings

None. The shipped code is correct as written; the failures are in what pins it and what the artifacts claim.

## 3. Important findings

**I-1 — the `/play` dispatch in `runEditor` is unpinned; `/play` can be made a no-op with a green suite** (`cmd/define/replraw.go:539-553`).
Measured, not inferred: replacing line 540 with `_ = sitting` leaves every non-git-dependent test passing. No test enters that branch, so the record-then-perform wiring — the thing the issue is about — has no coverage at any interleaving.

> **This is the 3rd finding in family `plan-named-test-not-written`.** Do not fix this instance. The rule: **a plan step is not ticked until the test it names exists and a named mutation of the code it covers reddens it.** The enumeration for this plan is exact and small — Verification lists 6 items and the Test surface 3 rows. Delivered and mutation-checked: items 3, 5 (one door), 6, plus the shape rows. Not delivered: item 2 (the refusals — deleted, see BR-6) and item 4 (the pty row: `/play`, answer, Ctrl-C, back at the prompt; `pty_conformance_test.go` is untouched in this window). Two of the plan's own obligations are unmet while 15/15 boxes read done. Adopt the rule at the tick, and record the two unmet obligations in `## Revisions` rather than ticking over them.

**I-2 — `workshop/lessons.md` records a Go semantics claim that is false, and a superseded lesson as "what worked"** (`workshop/lessons.md:4048-4073`, mirrored at `cmd/define/play_cmd.go:75-78`).
Line 4069: *"`return code, shape` evaluates `shape` before the defer runs, so it hands back the pre-sitting size."* With **named** results — which `sittingInPlace` has — that is not true; I ran it: `return code, shape` still yields the deferred mutation. The real requirement is that the results be *named*, which they are; a plain `return` versus `return code, shape` makes no difference. Separately, lines 4048-4060 teach the round-1 approach ("narrow the claim to the RESTORE… the end-to-end path is covered by the pty run") that BR-7 rejected and `c2e593a` replaced with a test asserting both halves — and there is no pty run for `/play`. §4 makes `lessons.md` the rule-store; a wrong rule there is worse than no rule.
*Fix sketch:* rewrite 4069-4073 to "results settled by a defer must be **named**"; rewrite 4056-4060 to the lesson that actually held (force the ordering through an observable channel read, and assert both halves).

## 4. Minor findings

- **M-1 — `TestTheSittingDoorRecordsWhatItAnswers` (play_cmd_test.go:172-178) re-implements `reviewEvents`** (`play_loop_test.go:81`, same package), and `parser.ParseDir` + the `FuncDecl` walk is copied between play_cmd_test.go:31-58 and :107-119. This is BR-13 unchanged; disposed `not-addressed` below rather than re-raised.

## 5. Test coverage notes

- Mutation-confirmed live: BR-5, BR-7, BR-14 fixes each redden a named test when reverted. Suspend/resume covers the discriminating clause (frame returns after resume) plus the stopped-screen case.
- Mutation-confirmed dead: `runPlayCommand`'s refusals (nothing reddens), the `runEditor` `/play` dispatch (nothing reddens).
- `TestBothEntryPointsReachOnePlaySession` carries Done-when 4 structurally (AST), and `TestTheSittingDoorRecordsWhatItAnswers` carries the store half for one door only. That combination is defensible, but the Done-when's wording — "recorded identically from either entry point" — is stronger than what the store assertion proves; the AST guard is doing the "identically" work.
- `-race -count=3` clean over all nine new/changed tests. `go vet` clean under default, `pty`, and `conformance`; `gofmt -l` clean; full suite green (`cmd/define` 110s).

## 6. Architectural notes

- **ARCH-DRY — pass, with M-1 outstanding.** `applyShape` is the model consolidation. The test-file duplication is the one open instance.
- **ARCH-PURE — pass.** `runPlayCommand` is pure over `commandCtx`; `applyShape` is pure over `*options` + `display`; `sittingInPlace` is a thin shell driven in-process over `bytes.Buffer` and scripted channels with no pty and no mocks. Good separation.
- **ARCH-PURPOSE — flag (see I-1).** The shadow-sweep: `sittingInPlace` (reachable without a terminal) is thoroughly pinned; the command half (the row, the refusals, the wiring) is not. That is the easy subset of the purpose, and it is where the deleted test was.
- **ARCH-MOCK — pass.** No new external binary or service. The terminal dependency is consumed at the same seam production uses (`repl`, `resizes`, `tty`), so test flow and production flow share the boundary. Note for future: the live-conformance surface (`pty_conformance_test.go`) gained no `/play` row, so the one behaviour only a real terminal can confirm is covered by the operator's hand-run alone.
- **ARCH-CONSTRAINTS — pass.** Nothing per-keystroke added; `suspend` disarms the throttle; the sitting spawns no goroutine — it borrows the single `watchResize` and the single `readKeys`. Extent is stated in the plan and matches `sittingInPlace`'s defers.
- **ARCH-SECURE — N/A with reason.** No credentials and no persisted artifact parsed here. The one input from outside the process is the SIGWINCH-derived `winSize`, and it degrades visibly rather than crashing: `cols < minWrapWidth` turns wrapping off while the frame still fits the real columns (`applyShape`, pinned by `TestApplyShapeSetsBothWidths`).
- **ARCH-ORDER — pass on the sitting, flag on the dispatch.** `(stopped, suspended)` is two booleans but the legal set is `{live, suspended, stopped}` read off one gate in `repaint` (screen.go:918-924), with the illegal transition pinned. The interrupt test now has a genuine ordering seam. The flag is I-1: the `var sitting bool` → `if sitting` transition at replraw.go:539-543 is observed at *zero* interleavings — a sample of size zero, which is the case this principle calls the highest-leverage flag.
- **For upcoming work:** `TestARemovedDeclarationIsSweptOrRetired` diffs `merge-base..HEAD`, so a `Test*` added and removed inside one window is invisible to it. That is how this round's regression passed. Worth a separate issue: compare against the previous *boundary* as well as the branch point, or have the boundary review diff round-to-round.

## 7. Plan revision recommendations

One `## Revisions` entry, dated 2026-09-08, covering all of:

1. **The `sittingInPlace` row is still stale** (line 144): it reads `(ctx, d, opt, sess, keys, interrupts, stdout, stderr) int` against a shipped `(ctx, d, opt, keys, interrupts, repl *liveScreen, resizes <-chan winSize, tty, stderr io.Writer) (code int, shape winSize)`. The previous revision *claims* this was updated ("the entity table gained… `sittingInPlace`'s signature returning the shape via NAMED returns"); it was not.
2. **There is still no `suspend`/`resume` entity row**, in either the Pure or the Integration table, despite PQ-5's revision claiming one.
3. **Line 208-209 still says "the suspended screen takes the new shape on `resume`"** — the statement the BR-5/BR-14 revision identifies as false. Correct it in the body (a `~~struck~~` line with a pointer to the revision keeps the append-don't-overwrite rule).
4. **Task 2 Step 1 is ticked and `play_loop.go` is unmodified** — no shared half was extracted. Either untick it or record that the shared half is `todaysQuestions` + `play.NewSession` + `playSession` called from both, with no helper needed.
5. **Task 2 Step 2 is ticked and no pty test was written.** Untick, or record the deferral explicitly.
6. **Verification still backticks no test name** ("un-backticked until the tests land" — they landed). The plan has no `## Done when` heading, so `TestPlanCitesTestsThatExist` skips it rather than enforcing; cite the shipped names by hand.
7. **Step 8 remains the bare "README + atlas"** — name the file and section (`cmd/define/README.md`, the per-command paragraphs) per BR-2's rule, even though the sweep itself landed.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      play_loop.go is unmodified in this window and no artifact states the rule; runPlay:30 still hand-rolls the sentence.
  - id: BR-2
    disposition: addressed
    note: |
      The named surface shipped at cmd/define/README.md:724-730; Step 8's wording is folded into the plan-revision list.
  - id: BR-3
    disposition: addressed
    note: |
      The shipped signature takes repl *liveScreen and the real tty, not sess/stdout; the stale plan ROW is BR-10/BR-15's.
  - id: BR-6
    disposition: not-addressed
    note: |
      TestSlashPlayRefusals was written in 15ecac4 and DELETED in c2e593a; removing the nil-capability refusal now reddens nothing.
  - id: BR-7
    disposition: addressed
    note: |
      Verified by reverting Set/restore in a scratch copy - the test fails at "the sitting did not end when its own interrupt fired".
  - id: BR-10
    disposition: not-addressed
    note: |
      Still no suspend/resume entity row, and line 144's sittingInPlace signature is unchanged.
  - id: BR-11
    disposition: not-addressed
    note: |
      replraw.go:123 still sets con.newSitting unconditionally, so --play's console and the board carry it.
  - id: BR-12
    disposition: not-addressed
    note: |
      play_loop.go is unmodified; the two doors still differ in sentence, stream and exit code.
  - id: BR-13
    disposition: not-addressed
    note: |
      reviewEvents is still re-implemented at play_cmd_test.go:172-178 and ParseDir is still copied at :31 and :107.
  - id: BR-14
    disposition: addressed
    note: |
      Verified by bypassing applyShape on the hand-back route - both clauses of TestBothShapeRoutesGoThroughOnePlace fail.
  - id: BR-15
    disposition: not-addressed
    note: |
      Boxes ticked but two of them falsely (Task 2 Steps 1 and 2); signature row, suspend/resume row, Verification backticks and the did-not-ship record are all still missing.
  - id: BR-16
    disposition: not-addressed
    note: |
      Still named newSitting for a verb, and still takes the stderr it already captures as live.
findings:
  - id: new
    severity: Important
    family: plan-named-test-not-written
    title: |
      the /play dispatch in runEditor is unpinned - making the command a no-op leaves the suite green
    detail: |
      This is the 3rd finding in family plan-named-test-not-written, so the rule rather
      than the instance. Measured: replacing cc.startSitting = func() { sitting = true }
      (replraw.go:540) with a no-op leaves every non-git-dependent test passing, so /play
      typed at the prompt can silently do nothing. The rule: a plan step is not ticked
      until the test it names exists AND a named mutation of the code it covers reddens
      it. The enumeration is exact - Verification lists 6 items plus 3 Test-surface rows;
      delivered and mutation-checked are items 3, 5 (one door), 6 and the shape rows;
      unmet are item 2 (the refusals, deleted in c2e593a - see BR-6) and item 4 (the pty
      row, pty_conformance_test.go untouched in this window). Two obligations unmet while
      15/15 boxes read done. Adopt the rule at the tick and record the two unmet
      obligations in the plan's Revisions instead of ticking over them.
  - id: new
    severity: Important
    family: unverified-mechanism-claim
    title: |
      lessons.md states a Go semantics rule that is false, and records a superseded approach as what worked
    detail: |
      workshop/lessons.md:4069 says "return code, shape evaluates shape before the defer
      runs, so it hands back the pre-sitting size". With NAMED results - which
      sittingInPlace has - that is not so; I ran it, and the deferred mutation still wins
      for both a bare return and return code, shape. The real requirement is that the
      results be named, which they are. The same wrong claim is in the code comment at
      play_cmd.go:75-78. Separately, lines 4056-4060 record the round-1 approach ("narrow
      the claim to the RESTORE... the end-to-end path is covered by the pty run") that
      BR-7 rejected and c2e593a replaced, and there is no pty run for /play. AGENTS.md
      section 4 makes lessons.md the rule-store, so a wrong rule there misdirects future
      work; rewrite both entries to what actually held.
```

---

## Re-review — 2026-09-08T19:32:27-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 48 — /play: a sitting without leaving the loop |
| repo | tools |
| issue file | workshop/issues/000048-play-a-sitting-without-leaving-the-loop.md |
| boundary | whole-issue close |
| milestone | — |
| window | adcbcf1841ed6139fd0cc72f5adce4ad258e9c2a..ea74e04387b419859457fe6b56bd937e98cc32f6 |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-09-08T19:32:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The terminal work is genuinely good and I'd ship the *code* as-is: `suspend`/`resume` is the right shape with the right clause (`resume` repaints unconditionally, and `repaint` is the single gate both flags live in), `applyShape` collapses the two shape routes into one policy with a derived guard, the borrowed-channel design is now documented in `atlas/define.md`, and the interrupt test finally asserts both halves against an ordering the test can actually observe. What blocks SHIP is not correctness — it is that two production sites introduced by this issue are provably unpinned, and the plan asserts otherwise. I mutation-measured all three of this window's wiring sites in a scratch checkout of `ea74e04`: the loop's `/play` dispatch is now pinned (BR-17's instance is genuinely fixed), but gutting **all three** of `runPlayCommand`'s refusals leaves the `./cmd/define` failure set byte-identical to baseline, and making `newConsole`'s `newSitting` closure stop calling `sittingInPlace` altogether *also* leaves it byte-identical — production `/play` would run no sitting at all with a green suite. Meanwhile `c2e593a` **ticked** Chunk 1 Step 1 ("Write the failing tests. Three…") in the same commit whose `## Revisions` entry records "BR-6 — the three refusals this plan named in Chunk 1 were never written", and the same window added to `workshop/lessons.md`: *"Ticking a plan step you did not do is worse than leaving it open."* That is cheap to fix (three table-driven unit tests, one AST assertion, one untick) and must not cross the boundary as a ticked-but-undone claim — especially since BR-6 was already disposed `addressed` once in round 3 and was not.

## 1. Strengths

- **`screen.go:857-909` — suspend/resume is specified, not improvised.** `resume` repaints *unconditionally* with the reason written down ("whatever ran while this screen was quiet has overwritten every cell"), both verbs are idempotent, and `repaint`'s gate (`screen.go:919-922`) is the one place either flag is consulted. The comment even retracts its own earlier claim — the timer disarm is hygiene, and the sweep proved the gate is what stops the frame. `TestASuspendedScreenPaintsNothingAndResumesWhereItWas` + `TestResumeDoesNotReviveAStoppedScreen` pin the distinction from `Stop` in both directions.
- **`applyShape` (`replraw.go:49-56`) is the right answer to BR-14, not a second patch.** One function *is* what a shape means; both routes call it; `TestBothShapeRoutesGoThroughOnePlace` (`play_cmd_test.go:319`) derives that from the source and `TestApplyShapeSetsBothWidths` pins the below-the-floor policy behaviourally. And `play_cmd.go:99-121` spells out the ordering constraint (shape *before* `resume`, because resume repaints).
- **`TestASittingFromTheLoopNeverEntersRawMode` (`play_cmd_test.go:30`) checks its own premise.** It fails closed on the walk size *and* asserts `runPlay` still reaches `enterRaw`, so the guard cannot pass by having nothing to distinguish. That is the non-vacuity clause most derived guards omit.
- **`TestASittingScopesTheInterruptAndHandsItBack` (`play_cmd_test.go:254`)** asserts both halves and orders the fire through a channel read the test filled — and `workshop/lessons.md` records all four failed attempts, which is the durable half.
- **Docs gate satisfied in-window**: `atlas/define.md` gained both the derived command row and a full borrow-design section; `cmd/define/README.md:724-731` gained the per-command paragraph in the hand-swept section BR-2 named.
- **The Go-semantics correction is correct.** I verified independently: with named results the defer's mutation wins for both `return a, b` and a bare `return`; with unnamed results it cannot win at all. `lessons.md` and `play_cmd.go:75-81` now say exactly that.

## 2. Critical findings

None. No correctness defect, crash path, or contract drift found in the shipped code.

## 3. Important findings

**I-1 — `newConsole`'s `newSitting` closure is unpinned; the whole suite survives it doing nothing (`cmd/define/replraw.go:123`).**
**This is the 4th finding in family `plan-named-test-not-written`** (BR-6, BR-7, BR-17 precede it). Per the escalation rule I am not asking for this instance to be fixed — here is the rule.

> **A capability delivered through an injected field is pinned at BOTH ends: the consumer, and the production assembly that supplies the real implementation.** A test that installs its own double for field `F` proves the consumer and *nothing* about the producer. When the producer is a closure inside a constructor, the cheap pin is the AST derivation this file already uses three times.

The enumeration is exact and small — this issue introduced three wiring sites, and I mutation-measured each against `ea74e04` in a scratch checkout:

| site | mutation | result |
|---|---|---|
| `runPlayCommand`'s 3 refusals (`play_cmd.go:23-49`) | body → `if c.startSitting == nil { return 0 }; c.startSitting(); return 0` | suite green (BR-6) |
| loop dispatch (`replraw.go:540`) | `cc.startSitting = func() {}` | **reddens** `TestTypingSlashPlayRunsASitting`, `TestTheLoopAppliesTheShapeASittingHandsBack` ✓ |
| `newConsole`'s closure (`replraw.go:123-126`) | body → `return 0, winSize{}`, never calling `sittingInPlace` | suite green |

Two of three unpinned after four rounds of this family. `TestTypingSlashPlayRunsASitting` (`play_cmd_test.go:401`) sets `con.newSitting` itself, so it can never see the producer; `TestASittingFromTheLoopNeverEntersRawMode` and `TestBothEntryPointsReachOnePlaySession` walk `sittingInPlace` by name whether or not anything reaches it.
*Fix sketch:* extend `TestBothEntryPointsReachOnePlaySession` (or a sibling in the same AST pass) with a third assertion — `newConsole`'s body must reach `sittingInPlace` — so the production wiring is derived rather than doubled. Same technique, ~10 lines.

**I-2 — see BR-6 and BR-15 below; both are re-raised as `not-addressed` rather than as new ids.**

## 4. Minor findings

- `replraw.go:123` sets `con.newSitting` on *every* console `newConsole` builds, including `--play`'s (`play_loop.go:106`), contradicting the field's own doc at `replraw.go:187-191` ("or is nil where one cannot run"). Unreachable today. *(BR-11, still open.)*
- `--play` on a deckless directory prints `define: no deck in this directory, so there is nothing to review` to **stdout** and returns **0** (`play_loop.go:29-31`); `/play` prints `noDeckMessage(...)` to **stderr** and returns **1** (`play_cmd.go:44-47`). Two doors, two answers. *(BR-1/BR-12, still open.)*
- `play_cmd_test.go:170-179` re-implements `reviewEvents` (`play_loop_test.go:81`, same package, 20+ call sites); the `ParseDir` + `FuncDecl` preamble is copied between `play_cmd_test.go:32` and `:108` (ARCH-DRY). *(BR-13, still open.)*
- `console.newSitting` is a `new*` name on a verb that runs a whole sitting and returns its exit code, and its `stderr` parameter is always `con.stderr` == `live`, the value the closure already captures (`replraw.go:114`, `125`, `296`, `548`). *(BR-16, still open.)*
- **New, `guard-scope-narrower-than-claim`:** `TestBothShapeRoutesGoThroughOnePlace` (`play_cmd_test.go:319-321`) parses only `replraw.go`, but a second bare `repl.Resize(r, c)` outside `applyShape` lives at `play_cmd.go:116` — precisely the shape the guard's own message forbids ("a shape applied outside applyShape is a shape whose width policy was forgotten"). Benign today because `runEditor` calls `applyShape` immediately after, but the guard's doc claims a rule its parse scope cannot enforce.

## 5. Test coverage notes

- **`runPlayCommand`: zero tests.** No test file names it. Measured green under full gutting.
- **`newConsole` → `sittingInPlace`: zero tests.** Measured green under a no-op closure (I-1).
- **`opt.width` after a sitting is AST-only.** `TestTheLoopAppliesTheShapeASittingHandsBack` asserts `con.view.Size()` and explicitly declines to observe `opt.width` (`opt` is the loop's value copy). The behavioural half of BR-14's class rests entirely on `TestBothShapeRoutesGoThroughOnePlace`.
- **No pty/e2e for `/play`.** Task 2 Step 2 now records this honestly ("NOT a pty test… the pty was used once, by hand"), but **Verification item 4 still reads "The pty test: `/play`, answer, Ctrl-C, back at the prompt"** — the same document contradicting itself.
- `TestPlanCitesTestsThatExist` **skips** on this plan: the plan has no `## Done when` heading, so `donewhen == 0 && checked == 0` → `t.Skip`. It reports nothing either way, which is why the un-backticked Verification names went unnoticed.
- `TestPlanTablesNameEntitiesThatExist` still exempts both `new` rows: one unticked box makes `inProgress` true (`repo_guard_test.go:770`, `797`). The rows are correct now — but they are correct by hand, not by guard.
- Verified green in-window: `go test ./...` (all packages ok, 111s), `go vet` under default / `pty` / `conformance`, `gofmt` clean, `go build ./...` clean, working tree clean.

## 6. Architectural notes

- **ARCH-DRY — flag.** `reviewEvents` re-implemented and the AST preamble copied (BR-13); the `runPlay`/`sittingInPlace` questions-and-console preamble is written twice (recorded honestly as Chunk 2 Step 1 DID NOT SHIP, which is the right way to record it).
- **ARCH-PURE — pass.** `runPlayCommand` is pure over `commandCtx`; `applyShape` takes a `display` interface; `suspend`/`resume` unit-test against a `bytes.Buffer` with a real timer and no IO; `sittingInPlace` takes its terminal as parameters, which is exactly why the loop, the dispatch, the interrupt and the hand-back are all drivable in-process. The plan's claim that a pty was never needed is borne out by the code.
- **ARCH-PURPOSE — flag.** Shadow-sweep of "one sitting, two doors": `playSession` is single-sourced and derived-guarded ✓, the atlas command list derives ✓, `/help` derives ✓. But the *deckless refusal* still reads two ways by door (BR-1/BR-12) — against Done-when's "a change to the sitting cannot apply to only one of them" — and BR-17's escalated **rule** was answered by fixing the **instance**: the dispatch got its test while the enumerable siblings (I-1's table) did not.
- **ARCH-MOCK — pass.** No new external binary or service. The terminal is injected as `io.Writer` + channels and tests run the real stack (real `liveScreen`, real store) over a buffer, not a function-call mock. No live conformance is *owed* here, but note that the one manual pty run is the only end-to-end evidence and nothing in the suite carries it.
- **ARCH-CONSTRAINTS — pass.** Nothing runs per keystroke; the sitting is bounded by `opt.count`; `suspend` disarms the throttle timer; extent is bounded — no goroutine is created by `sittingInPlace`, which is the whole point of borrowing the resize channel.
- **ARCH-SECURE — N/A, stated.** No new untrusted-input parse, no credentials, no artifact format change.
- **ARCH-ORDER — pass with a note.** `suspended`/`stopped` is two booleans with three legal states, both verbs idempotent, and the illegal combination (`resume` on a stopped screen) is pinned by its own test — a tagged enum would be over-engineering here. The interrupt ordering is made deterministic through a channel the test filled, which is the right seam and the right lesson. The note: the shape hand-back is applied in **two** places with **two** completenesses (`play_cmd.go:116` rows/cols only, `replraw.go:551` full policy), and the guard forbidding exactly that reads only one of the two files.

## 7. Plan revision recommendations

Append a single `## Revisions` entry dated 2026-09-08, "close review rounds 3-5", covering:

1. **Untick Chunk 1 Step 1.** `c2e593a` ticked it while the same commit's Revisions entry recorded "BR-6 — the three refusals this plan named in Chunk 1 were never written", and the same window added the rule to `lessons.md`. Replace with the Step-2-style honest form: `- [ ] **Step 1: … DID NOT SHIP.**` plus one sentence — or write the three tests and leave it ticked.
2. **Fix Verification item 4.** It still names "The pty test: `/play`, answer, Ctrl-C, back at the prompt" while Task 2 Step 2 says the opposite. Replace with the in-process rows that actually shipped.
3. **Backtick the tests that shipped.** The de-backticking note is now stale — the names exist. Cite `TestASittingFromTheLoopNeverEntersRawMode`, `TestBothEntryPointsReachOnePlaySession`, `TestTheSittingDoorRecordsWhatItAnswers`, `TestASittingScopesTheInterruptAndHandsItBack`, `TestBothShapeRoutesGoThroughOnePlace`, `TestApplyShapeSetsBothWidths`, `TestTypingSlashPlayRunsASitting`, `TestTheLoopAppliesTheShapeASittingHandsBack`, `TestASuspendedScreenPaintsNothingAndResumesWhereItWas`, `TestResumeDoesNotReviveAStoppedScreen`. Note that `TestPlanCitesTestsThatExist` currently **skips** on this plan (no `## Done when` heading), so it will not catch a future omission here either.
4. **Correct "The three things that must be built" §2.** It still specifies `newSitting func() console`, a constructor; what shipped is `func(context.Context, deps, options, <-chan Key, *interrupter, io.Writer) (int, winSize)`, a verb that returns an exit code and a shape.
5. **Soften or re-scope Step 5's tick.** It claims "a review answered via `/play` and via `--play` produce the same event"; the shipped test drives one door and asserts properties, with the other door covered by the AST guard. Say that.
6. **Record the class, not just the instances**, for the family that has now produced four findings: the rule in I-1, plus the enumeration table, so the next injected-seam design pins the producer as well as the consumer.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      play_loop.go untouched in this window; runPlay:29-31 still hand-rolls the sentence to stdout with exit 0, and noDeckMessage's doc (main.go:1209) is unchanged. Neither the sweep nor the rule-statement happened.
  - id: BR-6
    disposition: not-addressed
    note: |
      Mutation-measured on a scratch checkout of ea74e04: gutting all three refusals leaves the ./cmd/define failure set byte-identical to baseline. No test names runPlayCommand. Worse, c2e593a TICKED Chunk 1 Step 1 in the same commit whose Revisions entry records this finding.
  - id: BR-10
    disposition: addressed
    note: |
      Integration points now carries the liveScreen.suspend/resume row, and sittingInPlace's signature matches the shipped one exactly.
  - id: BR-11
    disposition: not-addressed
    note: |
      replraw.go:123 still assigns unconditionally; play_loop.go:106 builds --play's console through newConsole, so it carries a non-nil factory the field doc at replraw.go:187-191 says must be nil.
  - id: BR-12
    disposition: not-addressed
    note: |
      Unchanged - stdout/exit 0 versus stderr/exit 1 for the same deckless directory.
  - id: BR-13
    disposition: not-addressed
    note: |
      play_cmd_test.go:170-179 still re-implements reviewEvents (play_loop_test.go:81); the ParseDir preamble is still copied between play_cmd_test.go:32 and :108.
  - id: BR-15
    disposition: not-addressed
    note: |
      Half swept - entity rows, signature, Step 1's DID-NOT-SHIP note, Step 2's rewrite. Not swept, and one moved backwards - Chunk 1 Step 1 was ticked in c2e593a with none of its three tests written; Verification item 4 still claims a pty test Step 2 says was not written; Verification still backticks no test name though ten landed (TestPlanCitesTestsThatExist SKIPS here, no Done-when heading); "The three things that must be built" 2 still specifies newSitting func() console.
  - id: BR-16
    disposition: not-addressed
    note: |
      Field still named newSitting while returning (int, winSize); stderr param is still always con.stderr == live, the value the closure already captures.
  - id: BR-17
    disposition: not-addressed
    note: |
      The INSTANCE is fixed and I verified it - no-opping cc.startSitting at replraw.go:540 reddens TestTypingSlashPlayRunsASitting and TestTheLoopAppliesTheShapeASittingHandsBack. The escalated RULE was not adopted - the same window ticked Chunk 1 Step 1 with none of its named tests written, and a third site of the same class (newConsole's closure) is unpinned.
  - id: BR-18
    disposition: addressed
    note: |
      lessons.md now states the correct rule and records the earlier error; the round-1 narrowing is recorded as also wrong; play_cmd.go:75-81 corrected. I re-verified the Go semantics independently - named results, defer wins for both bare and explicit return.
findings:
  - id: new
    severity: Important
    family: plan-named-test-not-written
    title: |
      newConsole's newSitting closure can stop calling sittingInPlace entirely and the whole suite stays green
    detail: |
      4th in family - so the rule, not the instance. A capability delivered
      through an injected field is pinned at BOTH ends: the consumer, and the
      production assembly that supplies the real implementation. A test that
      installs its own double for field F proves the consumer and nothing about
      the producer. Enumeration is exact - this issue added three wiring sites,
      all mutation-measured against ea74e04 in a scratch checkout: the loop
      dispatch (replraw.go:540) reddens two tests; runPlayCommand's refusals and
      newConsole's closure (replraw.go:123-126, body replaced with
      "return 0, winSize{}") each leave the failure set byte-identical to
      baseline. Two of three unpinned. TestTypingSlashPlayRunsASitting sets
      con.newSitting itself so it can never see the producer; the two AST guards
      walk sittingInPlace by name whether or not anything reaches it. Cheap
      enforcement already exists in this file - add to
      TestBothEntryPointsReachOnePlaySession that newConsole's body reaches
      sittingInPlace, derived from the source.
  - id: new
    severity: Minor
    family: guard-scope-narrower-than-claim
    title: |
      the one-place-for-a-shape guard parses only replraw.go while a second bare Resize lives in play_cmd.go
    detail: |
      TestBothShapeRoutesGoThroughOnePlace (play_cmd_test.go:319-321) calls
      parser.ParseFile on "replraw.go" alone, and its message says "a shape
      applied outside applyShape is a shape whose width policy was forgotten" -
      but play_cmd.go:116 does exactly that, repl.Resize(r, c) with no opt.width,
      one file over. Benign today because runEditor calls applyShape immediately
      after, so the class the guard names is enforced only where it happens to
      look.
```

---

## Re-review — 2026-09-08T21:39:28-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 48 — /play: a sitting without leaving the loop |
| repo | tools |
| issue file | workshop/issues/000048-play-a-sitting-without-leaving-the-loop.md |
| boundary | whole-issue close |
| milestone | — |
| window | adcbcf1841ed6139fd0cc72f5adce4ad258e9c2a..116d0f326d5bb4d1d3531b6fcfe5dd098070ab40 |
| command | sdlc close --issue 48 |
| reviewer | claude |
| timestamp | 2026-09-08T21:39:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Two of the three open Importants are genuinely closed this round and I mutation-verified both: gutting `newConsole`'s `newSitting` closure reddens `TestTheProducedSittingCapabilityCallsTheRealThing`, and no-op'ing the loop's `cc.startSitting` reddens `TestTypingSlashPlayRunsASitting` and `TestTheLoopAppliesTheShapeASittingHandsBack`. `applyShape`, `suspend`/`resume`'s single `repaint` gate, and the deck-shrink sweep (`d.deck.Forget` has exactly two call sites; only the loop-outliving one needed telling) are all solid. What stops a clean SHIP is one live defect and one measured coverage gap. The defect: `memVocabulary.Forget` recounts `maxWords` with a *different rule* than `Add` maintains it with — I ran it, and after a `Forget` a deliberately-unmatchable key raises `MaxPhraseWords` from 1 to 2, re-introducing the wider-lookahead cost `Add`'s own comment says was measured and rejected. The gap: I removed `repl.suspend()`/`repl.resume()` from `sittingInPlace`, and separately gutted its `finish`, and the whole `./cmd/define` suite stayed green both times — so the issue's headline capability and the README's promised summary-in-the-scrollback are both unpinned, and BR-19's hand-written enumeration of "exactly three wiring sites" was wrong.

## 1. Strengths

- **The producer guard is real, not decorative.** `play_cmd_test.go:452-500` reads `newConsole`'s installed closure from the source; I replaced its body with `return 0, winSize{}` and it reddened with its own message. That is BR-19's rule actually enforced, and it is confirmed-good ground to build on.
- **The dispatch pin holds under mutation.** `replraw.go:540` → `TestTypingSlashPlayRunsASitting` (`play_cmd_test.go:398`): a no-op `startSitting` reddens two tests, not one.
- **`applyShape` (`replraw.go:36-53`) is ARCH-DRY done right** — one function that *is* what a shape means, both replraw routes derived from the source by `TestBothShapeRoutesGoThroughOnePlace`, plus `TestApplyShapeSetsBothWidths` on the policy itself.
- **One gate for "may I paint" (`screen.go:918-922`).** Both `stopped` and `suspended` live in `repaint`, the only writer to the tty, and `TestResumeDoesNotReviveAStoppedScreen` pins the interaction of the two flags rather than each alone.
- **The deck-shrink shadow-sweep is complete.** `d.deck.Forget` has exactly two call sites — `main.go:1196` (one-shot `--forget`, stale set dies with the process) and `play_loop.go:451` (the sitting, which the loop outlives). Only the second needs telling, and it is told. ARCH-PURPOSE passes on this axis.
- **`Task 2 Step 1` left unticked with "DID NOT SHIP" written into it** is the right artifact discipline, and the lessons entry that generalises it is well earned.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `memVocabulary.Forget`'s recount uses a different rule than `Add`, so `MaxPhraseWords` can rise above what `Add` permits** (`cmd/define/vocab.go:101-106`).

`Add` (`vocab.go:129-131`) raises `maxWords` only when `phraseRunsJoin(key, runs)` — a key holding other punctuation (`e.g.`, `9/11`) is permanently unmatchable, and the comment records the measured cost of counting it anyway ("a single-word deck holds 5 bytes and adding the unmatchable `e.g.` holds 9"). `Forget`'s recount drops that filter:

```go
v.maxWords = 0
for w := range v.words {
    if n := len(wordRuns(w)); n > v.maxWords { v.maxWords = n }
}
```

Executed, not reasoned: `Add("keel"); Add("e.g."); Add("junk")` → `MaxPhraseWords()==1`; `Forget("junk")` → `MaxPhraseWords()==2`. Any `/play` drop, on a deck containing one punctuated entry, widens the streaming renderer's lookahead window for the rest of the session. No wrong highlight results (`phraseGap` still refuses the candidate), so the blast radius is latency on the streaming path — ARCH-CONSTRAINTS, and ARCH-DRY for the duplicated maintenance rule.

Fix sketch: one helper both sides call — `v.recount()` applying `phraseRunsJoin`, or in `Forget` `if n := len(runs); n > v.maxWords && phraseRunsJoin(w, runs)`. Extend `TestAWordDroppedInASittingLeavesTheHighlightSet` with a punctuated key; the current fixture (`keel`, `hot dog`) was written from the same mental model as the fix and cannot see this.

**I-2 — `suspend`/`resume` and the sitting's `finish` are unpinned at the site; the enumeration that was supposed to prevent this was hand-written and wrong.**

**This is the 5th finding in family `plan-named-test-not-written`.** Earlier rounds fixed instances (BR-6, BR-17, BR-19). Per the escalation rule I am not asking for these two instances to be patched — the rule is what needs fixing.

Measured in a scratch checkout of `116d0f3`, full `./cmd/define` suite each time:

| mutation | result |
|---|---|
| delete `repl.suspend()` (`play_cmd.go:97`) **and** `repl.resume()` (`play_cmd.go:117`) | **green** |
| replace `finish:` body (`play_cmd.go:137-140`) with `func() {}` | **green** |
| gut `runPlayCommand` (`play_cmd.go:23-49`) to a bare nil check + `c.startSitting()` | **green** (BR-6, still) |

So the one genuinely new capability this issue exists for — the editor's screen going quiet so its throttled painter cannot land inside the sitting's frame — is pinned as a *unit* (`TestASuspendedScreenPaintsNothingAndResumesWhereItWas`) and not at the site obliged to call it. Same for the summary-goes-up decision the plan spends two revisions arguing for and the README promises the user ("with the session's summary in the scrollback above you", `README.md:724`).

**The rule:** BR-19 stated "an injected capability is pinned at BOTH ends" and then *hand-listed* the sites — "the enumeration is exact — this issue added three wiring sites". It was not exact; it missed three more, including the headline one. **An enumeration of "sites obliged to obey" must be derived from the source, not listed from memory** — the same discipline `TestASittingFromTheLoopNeverEntersRawMode` already applies to callees and `TestBothShapeRoutesGoThroughOnePlace` to routes. Concretely: one AST guard over `sittingInPlace`'s body asserting it calls each of `suspend`, `resume` and writes `Transcript()` into `repl`, derived the way the drop-arm guard (`play_cmd_test.go:557-616`) already is. That guard is cheap and it covers the class rather than the three instances I happened to mutate.

**I-3 — BR-6 remains open and measurably so.** Disposed `not-addressed` below rather than re-raised; noted here only because it is the Done-when "`/play` on the line-mode REPL refuses with a sentence naming the cause", ticked `[x]` in the issue, with no test naming `runPlayCommand` anywhere in the tree.

**I-4 — BR-15 remains partly open, and I verified the exemption mechanism it named.** In a scratch copy I renamed the `runPlayCommand` entity row to `noSuchEntityAtAll` and `TestPlanTablesNameEntitiesThatExist` stayed **green** — because the plan still contains one `- [ ] ` (Task 2 Step 1), `inProgress` is true, and `repo_guard_test.go:797` skips every `new` row. All three `new` rows (`runPlayCommand`, `sittingInPlace`, `applyShape`) are unchecked at this close. Details in the disposition and in §7.

## 4. Minor findings

- **BR-1 / BR-12 unchanged:** `play_loop.go:30` still prints `"define: no deck in this directory, so there is nothing to review"` to **stdout** and returns **0**; `play_cmd.go:44-45` prints `noDeckMessage` to **stderr** and returns **1**. Same condition, two doors, two answers — and no rule was written down either way.
- **BR-11 unchanged:** `replraw.go:123` sets `con.newSitting` unconditionally, so `--play`'s pinned-screen console carries a sitting factory. Unreachable (only `runEditor` reads it), but the field's doc at `replraw.go:189-193` says nil is what a console gets where a sitting cannot run.
- **BR-16 unchanged:** the sixth parameter of `newSitting` (`replraw.go:193`) is always `con.stderr`, which in production *is* the `live` screen the closure already captures (`replraw.go:296` → `:548`). Still named `newSitting` while running a sitting and returning an exit code.
- **BR-13 unchanged:** `play_cmd_test.go:171-180` re-implements `reviewEvents` (`play_loop_test.go:81`) in the same package; the `ParseDir`+`FuncDecl` preamble is now duplicated at `:32` and `:108`, with two more `ParseFile` copies at `:321` and `:458` (ARCH-DRY).
- **BR-20 unchanged:** `TestBothShapeRoutesGoThroughOnePlace` (`play_cmd_test.go:321`) parses `replraw.go` alone while `play_cmd.go:116` does a bare `repl.Resize(r, c)` — benign (the caller runs `applyShape` immediately after) but the guard's own message claims a class it does not police.
- `Stop()` while suspended silently drops its final flush, because `repaint`'s gate now includes `suspended`. Unreachable today (the sitting is synchronous inside the dispatch, so `runEditor` cannot return mid-suspension) — noted for whoever makes a sitting asynchronous.
- The plan's `## Verification` section has two items numbered `6`.

## 5. Test coverage notes

- `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .` all clean at `116d0f3`.
- Mutation-verified live this round: the producer guard, the loop dispatch, the shape hand-back, and the drop-arm `Forget` guard all redden under the mutation they claim to catch.
- Mutation-verified **absent**: `runPlayCommand`'s three refusals, `repl.suspend()`/`repl.resume()`, the sitting console's `finish`.
- `TestAWordDroppedInASittingLeavesTheHighlightSet` is a fixture written from the fix's own model — it asserts the recount goes *down* and never asks whether it may go *up* past `Add`'s rule. That is the gap I-1 lives in.
- `TestTheDropArmUpdatesTheHighlightSet` honestly documents that it reads the source rather than driving the arm, and records the pre-existing vacuous `TestDropRecordsNoReview` in the issue Log rather than hiding it. That is the right call for this boundary.

## 6. Architectural notes

- **ARCH-DRY — flag.** I-1 (two maintenance rules for one derived aggregate) and BR-13 (test helper re-implemented, AST preamble ×4).
- **ARCH-PURE — pass.** `runPlayCommand` is pure over `commandCtx`; `applyShape` and `Forget` are pure; `sittingInPlace` is the thin IO shell and takes its terminal as parameters, which is what let this round's tests drive the loop in-process at all.
- **ARCH-PURPOSE — pass, with one note.** The shadow-sweep on deck-shrink is complete (two `d.deck.Forget` sites, correct one wired). The `/play`-vs-`--play` refusal divergence (BR-1/BR-12) is the one place a "one sitting, two doors" purpose is still delivered as two.
- **ARCH-MOCK — pass.** `TestTheSittingDoorRecordsWhatItAnswers` asserts through `store.Mem`, the stateful fake, not a call-counting capturer; production and test share the `console`/key-channel boundary.
- **ARCH-CONSTRAINTS — flag (I-1).** Otherwise sound: a sitting is bounded by `opt.count`, nothing added runs per keystroke, and the borrow means one resize watcher for the process's life.
- **ARCH-SECURE — N/A, stated.** No new untrusted input, no credentials; `Forget`'s argument comes from the deck this process just read.
- **ARCH-ORDER — pass.** `suspend`/`resume` is a reversible flag distinct from the one-way `stopped`, both read at one gate, and `TestResumeDoesNotReviveAStoppedScreen` pins the only interesting combination — two booleans here do not need a tagged enum. The interrupt is `Set` + `defer restore()`, and the tests make ordering *observable* (poll the borrowed `resizes` channel, then fire) rather than sampling one interleaving. Extent is lexically bounded: the sitting's screen is stopped and no goroutine outlives the call. The one gap is that no test observes the *interleaving between the two screens* — which is exactly what I-2's missing site-guard would cover cheaply.

## 7. Plan revision recommendations

Add a `## Revisions` entry dated 2026-09-08 covering close-review rounds 3–6, and in the same sweep:

1. **Delete or rewrite the `newSitting func() console` code block** (`plan:95`). The shipped field is `func(ctx, d, opt, keys, interrupts, stderr) (int, winSize)` — a verb returning an exit code and a shape, not a constructor returning a console. The plan has claimed the wrong signature for six rounds.
2. **Reconcile the pty claims.** `plan:180` ("`sittingInPlace` — needs a terminal, so the **pty test** under the `pty` tag") and Verification item 4 (`plan:312`) both still promise a pty test that `Task 2 Step 2` (`plan:278`) explicitly records as not shipped. One of the three has to change.
3. **Add the entity rows for what the last commit shipped:** `Vocabulary.Forget` (`cmd/define/vocab.go`, modified — the interface gained a method) and `memVocabulary.Forget` (new). The deck-as-third-field work has no row and no revision entry anywhere in the plan.
4. **Back-tick the tests that shipped in `## Verification`.** The un-backticked convention was adopted "until the tests land"; they have landed, and `TestPlanCitesTestsThatExist`'s own message asks for the name that shipped.
5. **Resolve the `inProgress` exemption.** While Task 2 Step 1 stays `- [ ]`, every `new` entity row in this plan is skipped by `TestPlanTablesNameEntitiesThatExist` — I confirmed a bogus row passes. Either tick it with the "shipped instead:" sentence inline, or move the not-shipped record into the Revisions section so the plan can close as complete and its rows become checkable.
6. Renumber the duplicated Verification item `6`.

```findings
dispose:
  - id: BR-17
    disposition: addressed
    note: |
      Mutation-verified: no-op'ing cc.startSitting at replraw.go:540 reddens TestTypingSlashPlayRunsASitting and TestTheLoopAppliesTheShapeASittingHandsBack.
  - id: BR-19
    disposition: addressed
    note: |
      Mutation-verified: replacing newConsole's newSitting closure body with `return 0, winSize{}` reddens TestTheProducedSittingCapabilityCallsTheRealThing.
  - id: BR-6
    disposition: not-addressed
    note: |
      Re-measured at 116d0f3: gutting runPlayCommand to a bare nil check plus c.startSitting() leaves the whole ./cmd/define suite green. No test names runPlayCommand.
  - id: BR-15
    disposition: not-addressed
    note: |
      Entity rows and the DID-NOT-SHIP record are fixed; the plan still carries `newSitting func() console`, two pty claims contradicting Task 2 Step 2, no Forget row, no revision for rounds 3-6, and no backticked test names. I confirmed by execution that a bogus `new` row still passes the guard while the plan is inProgress.
  - id: BR-1
    disposition: not-addressed
    note: |
      play_loop.go:30 still hand-rolls the sentence; no rule stated in code, plan or lessons.md.
  - id: BR-12
    disposition: not-addressed
    note: |
      Unchanged - play_loop.go:30 stdout/exit-0 versus play_cmd.go:44-45 stderr/exit-1.
  - id: BR-11
    disposition: not-addressed
    note: |
      replraw.go:123 still assigns con.newSitting unconditionally; the field doc at replraw.go:189-193 is unchanged since 2dc2100.
  - id: BR-13
    disposition: not-addressed
    note: |
      play_cmd_test.go:171-180 still re-implements reviewEvents; the AST preamble is now duplicated four ways (:32, :108, :321, :458).
  - id: BR-16
    disposition: not-addressed
    note: |
      Signature and name unchanged; runEditor still passes con.stderr, which is the `live` screen the closure already captures.
  - id: BR-20
    disposition: not-addressed
    note: |
      Guard still parses replraw.go alone; play_cmd.go:116 still calls repl.Resize directly.
findings:
  - id: new
    severity: Important
    family: inverse-op-diverges-from-its-pair
    title: |
      memVocabulary.Forget recounts maxWords without Add's phraseRunsJoin filter, so a drop can widen the phrase window past what Add allows
    detail: |
      vocab.go:101-106 recomputes maxWords as max(len(wordRuns(w))) over every key, while Add
      (vocab.go:129-131) raises it only when phraseRunsJoin(key, runs) - because a key holding
      other punctuation is permanently unmatchable and counting it makes every stream hold a
      wider window for a match that cannot happen, a cost Add's comment records as measured.
      Executed at 116d0f3: Add("keel"); Add("e.g."); Add("junk") gives MaxPhraseWords()==1, and
      Forget("junk") raises it to 2. Any /play drop on a deck holding one punctuated entry
      widens the streaming renderer's lookahead for the rest of the session. No wrong highlight
      results, so the blast radius is latency (ARCH-CONSTRAINTS), but the invariant is broken
      and the two maintainers of one derived aggregate should be one helper (ARCH-DRY). The
      existing test uses `keel` and `hot dog` and cannot see it - written from the fix's own
      model.
  - id: new
    severity: Important
    family: plan-named-test-not-written
    title: |
      suspend/resume and the sitting's finish are unpinned at the call site, and BR-19's hand-written enumeration of wiring sites was wrong
    detail: |
      This is the 5th finding in family plan-named-test-not-written. Not asking for these
      instances to be patched - the rule is the deliverable. Measured at 116d0f3, full
      ./cmd/define suite per mutation: deleting repl.suspend() (play_cmd.go:97) and
      repl.resume() (play_cmd.go:117) leaves it GREEN; replacing the sitting console's finish
      body (play_cmd.go:137-140) with a no-op leaves it GREEN; gutting runPlayCommand leaves it
      GREEN (BR-6). So the issue's headline capability - the editor's throttled painter going
      quiet so it cannot land inside the sitting's frame - and the README's promised
      summary-in-the-scrollback are both unpinned. BR-19 stated the both-ends rule and then
      HAND-LISTED the sites ("the enumeration is exact - three wiring sites"); it missed three.
      The rule: an enumeration of sites obliged to obey a capability must be DERIVED from the
      source, not listed from memory - the discipline TestASittingFromTheLoopNeverEntersRawMode
      already applies to callees and TestBothShapeRoutesGoThroughOnePlace to routes. One AST
      guard over sittingInPlace's body (calls suspend, calls resume, writes Transcript() into
      repl), shaped like the drop-arm guard at play_cmd_test.go:557-616, covers the class.
```
