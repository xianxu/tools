# Boundary Review — tools#6 (whole-issue close)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..12716049eeaeef2ac7358f0b1496826fceaf309c |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-08-27T10:40:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M2 code itself is good work — `play` is the purest package in the tree (`go list` returns an empty import set), the skip filter lives in exactly one place and I confirmed by inspection that `TestSkippedVerdictRecordsNothing` reddens without it, the CRLF fix is pinned by a byte-level assertion, and the drop key is modelled as a session-level `InputKind` rather than something a form grades, which genuinely does give `#7`/`#12`/`#13` the behaviour for free. What blocks the close is not the code, it is the evidence. Ten of the eleven M1 findings arrive at this gate **unchanged** — `puretest` still reports `[no test files]` while `000006-vocab-play.md:183` still claims its negative cases are "verified in the tree"; the plan's Core-concepts table and Task 2/Task 3 steps still instruct the contract the code correctly rejects; every plan checkbox is still `- [ ]`. On top of that, two Done-when rows are ticked `[x]` with nothing behind them: I measured coverage and `play_loop.go:131–144` (the entire audio block) is **0**, and `playRig` hard-codes `noAudio: true`, so "Audio plays before reveal by default, and `--no-audio` silences it" is asserted by no test at all; `-count` is likewise unexercised. The `--play` flag→`runPlay` wiring at `main.go:369` has zero coverage in a repo whose own `news_test.go:209` records `#21` shipping two dead entry paths for exactly this reason. And README:50 plus `atlas/define.md:1215` both describe a prompt line and a key mapping the code stopped printing three commits ago. AGENTS.md §5 is "NEVER mark done without proof"; this close is the gate that enforces it.

## 1. Strengths

- **`play` imports literally nothing** (`purity_test.go:21` passes an empty allowlist and passes). ARCH-PURE in its strongest form — `recall.go:14-21` takes the *already-rendered* definition specifically so `Render`/`RenderOpts`/the dictionary stay outside. Confirmed against `go list`.
- **The skip filter is in one place and stayed there.** `session.go:161-165` emits `OutcomeNone`; `play_loop.go:114-119` records on `OutcomeRecord` without ever reading `out.Verdict`. The plan's round-3 "one rule, two places or neither" worry is genuinely resolved.
- **`TestUngradedKeyNeverReachesTheCapturer` (`play_loop_test.go:185`) is the right test, for the reason the Log gives.** A spy on the capturer is the only discriminating observable — an `OutcomeNone` carries an empty word that `CaptureReview` drops anyway, so the event log cannot tell the two implementations apart. That is a real mutation lesson correctly acted on.
- **`TestSessionOutputIsAllCRLF` (`play_loop_test.go:267`) asserts the invariant, not the incident.** "In raw mode there is no such thing as a bare newline" is the assertable form of a defect a byte capture structurally cannot see, and it includes a vacuity guard plus a check that the definition actually reached the writer.
- **`TestDropRemovesFromDeckButKeepsEvents` (`play_loop_test.go:293`) seeds the lookup event first**, with a comment explaining that without it the preservation assertion would range over an empty log and pass vacuously. That is the failure mode of this exact test shape, caught.

## 2. Critical findings

None. The shipped feature works; the operator ran a real session against it.

## 3. Important findings

**A. Two Done-when rows are ticked with no test in the tree — 2nd in family `claim-without-failing-test`.**
`000006-vocab-play.md` ticks "Audio plays before reveal by default, and `--no-audio` silences it" and "`-count` bounds the session; it defaults to 20". I ran `go test -coverprofile` over the play tests: `play_loop.go` lines `131.37–142.7` — the whole `!opt.noAudio && opt.times > 0` block including `raw.restore()`, `playAnnounced` and the re-entry — report **count 0**. `playRig` (`play_loop_test.go:33`) returns `noAudio: true` and no test overrides it, so `d.player = &fakePlayer{}` at line 31 is installed and never asked anything (**ARCH-MOCK**: the stack is never run against the fake at the one seam this feature added). `-count` is passed as 20 against 2-word decks, so bounding is never exercised and the default is never read by a test. Plan Task 4 Step 1b demanded exactly this row.

Per the family escalation: **do not just add the audio test.** The rule is *no Done-when row is ticked without a named test in the tree that fails without it*, and the enumeration is the Done-when list itself. I walked it — rows 1, 2, 5, 6, 7, 8, 9, 10, 11 each have a pinning test; rows 3 (audio) and 4 (`-count`) have none. Write the enumeration into the issue (row → test name) and close both gaps in this round, alongside BR-2's identical instance in `puretest`.

**B. `main.go:369` — the `--play`/`-count` flag wiring has zero coverage, which is the dead-entry-path class this repo has already been bitten by.**
No test calls `run()` with `--play`; every play test enters at `runPlay`/`playSession` with a hand-built `deps`. Plan Task 4 Step 8 committed to "the same entry-path enumeration `#21` needed — one row per process entry that can start a session", and `news_test.go:209-212` records the precedent verbatim: *"#21 shipped two dead entry paths because its tests injected a pre-filled…"*. Nothing today would catch `opt.count` not being threaded, `withStore` being dropped, or the dispatch landing after the argument-count switch. Fix: one table test driving `run(ctx, []string{"--play"}, …)` and `run(ctx, []string{"--play", "-count", "3"}, …)` through production wiring with a non-terminal stdin, asserting the loop was reached and the budget arrived.

**C. `main.go:369` — `define --play sycophantic` silently runs a session and ignores the word, contradicting a rule this file states three times.**
`--forget` and `--reflect` both reject a mode-plus-word line with exit 2, and `main.go:389-392` gives the reason: *"A mode plus a word is two commands on one line, and silently honouring one of them is how -raw came to mean two things in #2"*, pinned by `TestReflectWithAWordIsAUsageError`. `--play` is dispatched above that switch and can never reach it. `--llm-check` is the same shape, but it is read-only; `--play` writes events, so the user who typed a word and got a full session has had state changed under a misread intent. Fix: add `case *playFlag && fs.NArg() != 0` to the switch and move the dispatch below it (it already needs `withStore`, which is there anyway).

**D. Docs describe behaviour the code stopped exhibiting three commits ago — the drop key ships undocumented.**
`eba07e0` added `d`/`D` → `InputDrop`, which **permanently removes a word from the deck on one keystroke**. `README.md:41-51` still says only "Enter or space reveals… `y` and `n` say whether you had it. Ctrl-C stops", and its sample transcript prints `Enter or space to reveal, Ctrl-C to stop` — a line `draw()` (`play_loop.go:233`) no longer emits. `atlas/define.md:1213-1215` is now factually wrong: *"Ctrl-C and EOF become `InputQuit`, Enter and space become `InputReveal`, **everything else is a rune for the form to grade**"* — `d`/`D` do not. Neither `InputDrop`/`OutcomeDrop`, nor the "session output goes through `crlfWriter`" rule, nor the check-cancellation-before-select rule appear in the atlas, though all three are this window's durable design decisions. Both gates in the checklist (atlas, README) fire.

**E. `play_loop.go:139-142` — the raw-mode re-entry error is swallowed entirely — 2nd in family `discarded-error-detail`.**
```go
if again, err := enterRaw(os.Stdin); err == nil { *raw = *again }
```
If `enterRaw` fails after audio playback, the terminal stays cooked for the remainder of the session: `readKeys` becomes line-buffered, so every subsequent keystroke appears to do nothing until the learner presses Enter, and nothing is written to stderr. The session looks frozen. Per the family escalation, the rule is *an error is either acted on or reported; it is never dropped at the point it is available*. I enumerated the window's production sites: this one (dropped entirely) and `puretest.go:43,123` (BR-10 — `.Output()` discards `exitErr.Stderr`). `play_loop.go:184-188` is the correct shape and should be the model — report and degrade. Sweep both, not one.

## 4. Minor findings

- `play_loop.go:140` — `playSession` re-enters raw mode on `os.Stdin` rather than the file `runPlay` was handed, which is *why* the whole audio path has 0 coverage (**ARCH-PURE**: the one hard-wired IO reference in an otherwise-injected loop). Pass the `*os.File` through and finding A becomes testable.
- `play_loop_test.go:218` — `TestCancelledContextEndsTheSession` pins the select-race fix only probabilistically: with the guard removed, both `<-ctx.Done()` and a buffered key are ready, so the test reddens roughly 1 run in 4. A deterministic form (assert the first scripted key is still buffered on return) would fail every time.
- `play_loop.go:217` — `var anyTime = store.Word{}.FirstSeen` is an obscure spelling of `time.Time{}`, and `reflect.go:250` already writes the plain form for the same "everything" sentinel (ARCH-DRY).
- `play_loop.go:157-172` — `toInput` reserves Enter, space, `d` and `D` from *every* form, which qualifies the "adding a second form requires no change to the loop" Done-when. **2nd in family `unrecorded-scope-decision`** (with BR-7). Don't just note this key: the rule is *a decision that constrains a future consumer is recorded in the Log or plan in the same round*, and the open enumeration is exactly two — form 2.1 having no skip key (BR-7) and the loop's reserved key set. `#7` will hit the second one when it picks digits.

## 5. Test coverage notes

- Measured, not assumed: `play_loop.go` `38.2–73.50` (interrupt wiring, terminal check, `enterRaw`, `readKeys`, `crlfWriter` construction) and `131.37–142.7` (audio) are all **count 0**. Everything from the flag to `playSession` is verified only by the manual pty smoke test recorded in the Log — which is real evidence, but it is not in the tree and does not survive the next refactor.
- `cmd/define/puretest` remains `[no test files]`. Gutting `t.Errorf` at `puretest.go:52` and `:76` still leaves `play` and `schedule` green. Five packages are scheduled to depend on this body.
- What *is* well covered: the pure `play` package (session, recall, form-agnosticism, drop, reveal idempotence) and `CaptureReview` (verdict, normalisation, no-capture, deck-untouched). Those are the right tests at the right layer.

## 6. Architectural notes

- **ARCH-DRY — pass, with one nit.** The purity-guard extraction is the exemplar: `schedule/purity_test.go` goes 193 → 38 lines and `eachSourceFile` is shared rather than copied, at exactly the right moment (two callers, five coming). Nit: `anyTime` (minor 3); test-local duplication BR-8 still open.
- **ARCH-PURE — pass on the core, flag on the shell.** `play` imports nothing and `Apply` is a clean state-in/state-out machine with every effect named as an `Outcome`. The flag is `playSession`'s hard-wired `os.Stdin` (minor 1), which is the single reason the loop's most side-effecting branch cannot be driven.
- **ARCH-PURPOSE — flag.** The purpose of a close gate is that the issue's claims are true when it closes. Ten prior findings arrive undisposed; two Done-when rows are ticked with nothing behind them; the plan M2 was executed from still instructs a superseded contract. Each is individually small and the class is one thing: claims outrunning verification. Fix the class in this round.
- **ARCH-MOCK — flag.** `fakePlayer` exists and is wired but never exercised (finding A) — a fake that no test drives satisfies nothing. `puretest` shells out to the `go` binary with `*testing.T` welded into the signature, which is precisely what blocks BR-2's fix; a minimal `reporter` interface (`Helper/Errorf/Fatalf`) is the seam, and a deliberately-impure `testdata/` subject is the stateful double.

## 7. Plan revision recommendations

Add a `## Revisions` section to `workshop/plans/000006-vocab-play-plan.md` recording, as one entry dated 2026-08-27:

1. **Core concepts table (BR-4)** — move `Verdict` to `cmd/define/play/question.go`; add rows for `Input`/`InputKind` (incl. `InputDrop`) and `Outcome`/`OutcomeKind` (incl. `OutcomeDrop`); add a row for the `cmd/define/puretest` package naming its kind (it shells out to `go`, so it is neither pure nor a production integration point); add the M2 helpers `todaysQuestions`, `toInput`, `playSession`, `draw`, `finish` to the integration table.
2. **Line 26 (BR-4)** — strike "`play` imports `store`, `schedule` and pure stdlib"; `play` imports nothing. Line 48 — strike the promise of the store SYMBOL guard and record why `purity_test.go:13-19` deliberately omits it (an empty allowlist would make the guard vacuous).
3. **Task 2 Step 1 (BR-5/PQ-3)** — replace "a skip emits a record outcome with `Skipped`" with "a skip emits `OutcomeNone`", and strike "the last answer emits quit" (it emits `OutcomeRecord` and sets `Done`). Task 3 Step 1 — replace "the loop drops it" with "`Apply` drops it", so the file stops contradicting its own "ONE filter, in `Apply`".
4. **Checkboxes (BR-11)** — tick Chunk 1 Steps 1-9 and Chunk 2 Steps 1-10 to match what shipped, and mark Task 4 Step 1b (audio) and Step 8 (entry-path enumeration) as **not delivered**, since findings A and B show they were not.
5. **New entry for the operator round** — the `d` drop key, the reserved key set (`Enter`, space, `d`, `D`) and its cost to future forms, and the cancellation-before-select rule. None of these appear in the plan at all.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Apply filters (session.go:161-165); the test lives at session_test.go:88 and reddens without the branch.
  - id: BR-2
    disposition: not-addressed
    note: |
      cmd/define/puretest still reports "[no test files]"; issue:183 still claims the negative cases are in the tree.
  - id: BR-3
    disposition: not-addressed
    note: |
      No Outcome.SessionDone and no doc on Apply/Session.Done/OutcomeDone; playSession happens to loop on !s.Done.
  - id: BR-4
    disposition: not-addressed
    note: |
      plan:21, plan:26 and plan:48 unchanged; still no puretest row, and now no Input/Outcome rows either.
  - id: BR-5
    disposition: not-addressed
    note: |
      plan:98 and plan:122 are byte-identical to the round-2 text; PQ-3 is still open.
  - id: BR-6
    disposition: not-addressed
    note: |
      session_test.go:74 still named TestSkipAdvancesButRecordsNothing while asserting Index == 0.
  - id: BR-7
    disposition: not-addressed
    note: |
      recall.go:33 unchanged; no Log or plan entry records the no-skip-key decision.
  - id: BR-8
    disposition: not-addressed
    note: |
      skipForm still at session_test.go:220 beside fakeForm's identical Grade('3') capability.
  - id: BR-9
    disposition: not-addressed
    note: |
      question.go:45 doc comment unchanged; Verdict's zero value is still silently Skipped.
  - id: BR-10
    disposition: not-addressed
    note: |
      puretest.go:43 and :123 still use .Output(); exitErr.Stderr is still discarded.
  - id: BR-11
    disposition: not-addressed
    note: |
      Every step in both chunks is still "- [ ]", now including all of Chunk 2.
findings:
  - id: new
    severity: Important
    family: claim-without-failing-test
    title: |
      Two Done-when rows are ticked with no test: the audio block has measured zero coverage and -count is never exercised
    detail: |
      2nd in this family (with BR-2), so fix the RULE, not the instance. go test -coverprofile shows
      play_loop.go 131.37-142.7 at count 0; playRig (play_loop_test.go:33) hard-codes noAudio true, so
      fakePlayer is installed and never asked anything (ARCH-MOCK). -count is passed as 20 against 2-word
      decks, so neither the bound nor the default is read by a test. Plan Task 4 Step 1b demanded the audio
      row explicitly. The rule: no Done-when row is ticked without a named test in the tree that fails
      without it. I walked the enumeration - rows 3 (audio) and 4 (-count) are the only two with no pinning
      test. Write that row-to-test map into the issue and close both in this round, with BR-2.
  - id: new
    severity: Important
    family: dead-entry-path
    title: |
      The --play and -count flag wiring at main.go:369 has zero test coverage, the class #21 already shipped twice
    detail: |
      No test calls run() with --play; every play test enters at runPlay or playSession with a hand-built
      deps. Plan Task 4 Step 8 committed to "the same entry-path enumeration #21 needed", and
      news_test.go:209-212 records the precedent in this repo's own words. Nothing today catches opt.count
      not being threaded, withStore being dropped, or the dispatch moving below the argument-count switch.
      Fix: drive run(ctx, []string{"--play"}, ...) and run(ctx, []string{"--play","-count","3"}, ...)
      through production wiring with a non-terminal stdin.
  - id: new
    severity: Important
    family: mode-silently-ignores-argument
    title: |
      define --play sycophantic runs a full session and ignores the word, unlike --forget and --reflect
    detail: |
      main.go:389-392 states the rule and pins it for --reflect (TestReflectWithAWordIsAUsageError):
      a mode plus a word is two commands on one line, and silently honouring one is how -raw came to mean
      two things in #2. --play is dispatched at main.go:369, above that switch, so it can never reach it.
      Unlike --llm-check, which shares the position, --play writes events - so state changes under a
      misread intent. Fix: add a case for *playFlag and fs.NArg() != 0, and move the dispatch below the
      switch (it already needs withStore, which sits there).
  - id: new
    severity: Important
    family: docs-not-updated-for-new-surface
    title: |
      The d drop key ships undocumented and both README and atlas describe a prompt line the code no longer prints
    detail: |
      eba07e0 added d/D to InputDrop, which permanently removes a word from the deck on one keystroke.
      README.md:41-51 never mentions it and its sample transcript prints "Enter or space to reveal, Ctrl-C
      to stop", which draw() (play_loop.go:233) stopped emitting. atlas/define.md:1215 is now false:
      "everything else is a rune for the form to grade" - d and D are not. Neither InputDrop/OutcomeDrop,
      nor the session-output-through-crlfWriter rule, nor the check-cancellation-before-select rule appear
      in the atlas, though all three are this window's durable decisions. Both the atlas and README gates fire.
  - id: new
    severity: Important
    family: discarded-error-detail
    title: |
      play_loop.go:140 swallows the raw-mode re-entry error, leaving the terminal cooked and the session apparently frozen
    detail: |
      2nd in this family (with BR-10), so fix the RULE, not the instance. "if again, err := enterRaw(os.Stdin);
      err == nil" drops the error entirely: after a failed re-entry readKeys is line-buffered, so every
      keystroke appears to do nothing until Enter, and nothing reaches stderr. The rule: an error is acted on
      or reported, never dropped where it is available. Production sites in this window - play_loop.go:140
      (dropped entirely) and puretest.go:43,123 (exitErr.Stderr discarded). play_loop.go:184-188 is the
      correct shape and should be the model. Sweep both.
  - id: new
    severity: Minor
    family: io-not-injected
    title: |
      playSession re-enters raw mode on os.Stdin rather than the file runPlay was handed
    detail: |
      play_loop.go:140 - the one hard-wired IO reference in an otherwise fully injected loop (ARCH-PURE),
      and the reason the audio branch cannot be driven at all. Thread the *os.File through and the
      audio Done-when becomes testable.
  - id: new
    severity: Minor
    family: probabilistic-regression-test
    title: |
      The cancel-before-select fix is pinned only probabilistically - roughly 1 run in 4 reddens
    detail: |
      play_loop_test.go:218 - with the guard removed, ctx.Done() and a buffered key are both ready and
      select picks at random, so recording an event needs two coin flips to go the wrong way. A
      deterministic form (assert the first scripted key is still buffered on return) fails every time.
  - id: new
    severity: Minor
    family: duplicate-sentinel-spelling
    title: |
      anyTime spells time.Time{} as store.Word{}.FirstSeen, while reflect.go:250 writes the plain form
    detail: |
      play_loop.go:217 ties an "everything since the beginning" sentinel to an unrelated struct field's
      type. ARCH-DRY: one spelling for one fact.
  - id: new
    severity: Minor
    family: unrecorded-scope-decision
    title: |
      toInput reserves Enter, space, d and D from every form, qualifying the second-form Done-when, and no artifact records it
    detail: |
      2nd in this family (with BR-7), so fix the RULE, not the instance. The rule: a decision that
      constrains a future consumer is recorded in the Log or plan in the same round. Open enumeration is
      exactly two - form 2.1 having no skip key (BR-7), and the loop's reserved key set at
      play_loop.go:157-172. #7 hits the second when it picks its keys.
```

---

## Re-review — 2026-08-27T10:56:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..787824810e8a371c3f2c7c7e7899943d91dfbb39 |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-08-27T10:56:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The window ships a genuinely well-designed feature — `play` is a real pure core (it imports *nothing*), the skip filter lives in exactly one place, and `puretest` is a textbook single-source extraction with both consumers derived and no third copy anywhere in the tree. The head commit's three claimed fixes all hold up: I reverted both halves of the `--play` usage guard independently in a scratch copy and `TestPlayWithAWordIsAUsageError` reddened each time, the error sweep is complete at both production sites, and the README/atlas now describe what `draw()` actually prints. What blocks SHIP is that the round-3 commit fixed **three** findings and left **thirteen** untouched — including all four Importants raised at the **M1** boundary in round 2, which have now been open for three rounds while M2 was built on top of them. Worse, round 3 explicitly wrote the rule (`no Done-when row is ticked without a named test in the tree that fails without it`) and the very next commit shipped a fourth unpinned claim. I measured the two untested Done-when rows: both are pinnable **today** with fakes this package already owns — I wrote them in a scratch copy and the audio row passed with `player calls = 1`, no terminal involved. That is a ~20-line fix that has been deferred twice.

## 1. Strengths

- **`cmd/define/puretest` is the ARCH-DRY exemplar of this window.** One guard body, two callers, and I grepped the whole tree to confirm there is no surviving copy: `schedule/purity_test.go` collapsed from 193 lines to a 38-line call, and `play/purity_test.go:19` is 8. The reasoning for *why three guards* (an import allowlist can't see `time.Since`; allowlisting `store` grants the disk with it) is recorded at `puretest.go:85-91`.
- **The skip rule is enforced in exactly one place and the type system backs it up.** `session.go:161-165` filters in `advance`, and `capture.go:57-64` explains that `CaptureReview`'s `bool` *cannot* express a skip because one never reaches it. A signature that refuses the wrong call is better than a comment asking for it.
- **`TestSessionOutputIsAllCRLF` (play_loop_test.go:267)** asserts the one thing bytes *can* prove about raw mode, and the comment records why the pty smoke test missed the cascade (Python renders a bare `\n` at column 0). That reasoning is worth more than the test.
- **`TestUngradedKeyNeverReachesTheCapturer` (play_loop_test.go:185)** spies on the capturer rather than the event log — the discriminating observable, correctly identified after a mutation survived the log-based version.
- **The BR-14 fix is real, not plausible.** Removing the `case` → test red; keeping the case but moving the dispatch back above the switch → test red. Both halves are load-bearing.

## 2. Critical findings

None. Nothing in this window crashes, corrupts the log, or drifts from a stated contract.

## 3. Important findings

- **`claim-without-failing-test`, 3rd in family** — the rule was stated in round 3 and violated by the commit that closed round 3. Enumeration and measurement in the findings block below; I verified 4 of the 5 sites are pinnable with existing seams.
- **`plan-checkboxes-not-ticked`, 2nd in family** — `workshop/issues/000006-vocab-play.md:69` ticks M1, but M1's boundary review blocked with four Importants, no commit carries a `Review-Verdict:` trailer for it, and no `closed M1` line exists in `## Log`.
- Six prior Importants (**BR-2, BR-3, BR-4, BR-5, BR-12, BR-13**) remain open and untouched — see dispositions.

## 4. Minor findings

- `main.go:417` — `--play` opens the store **twice**; measured, every other path opens it once.
- `play_loop.go:147` — a lost terminal reports to stderr and exits **0**, while the sibling failure at `play_loop.go:58` exits 1.
- `play_loop.go:206-211` — a deck word the dictionary no longer knows has already consumed a `-count` slot before it is skipped; branch untested.
- Prior Minors BR-6…BR-9, BR-11, BR-17…BR-20 all still open, unchanged since round 3.

## 5. Test coverage notes

`go test ./...` is green (94.7s for `cmd/define`); `go vet` clean. The Done-when row-to-test map, walked in full:

| row | pinning test | |
|---|---|---|
| full session, one event per answer | `TestFullSessionRecordsOneEventPerAnswer` | ✓ |
| skip records nothing | `TestSkippedVerdictRecordsNothing` + `TestUngradedKeyNeverReachesTheCapturer` | ✓ |
| audio on by default, `--no-audio` silences | — | ✗ |
| `-count` bounds, defaults to 20 | — | ✗ |
| every newline is CRLF | `TestSessionOutputIsAllCRLF` | ✓ |
| `d` drops, keeps events | `TestDropRemovesFromDeckButKeepsEvents` | ✓ |
| empty queue exits 0 | `TestEmptyQueueExitsZero` | ✓ |
| no deck exits 0 | `TestNoDeckExitsZeroWithAMessage` | ✓ |
| interrupt preserves events | `TestInterruptPreservesRecordedEvents` | ✓ |
| second form needs no loop change | `TestSessionIsFormAgnostic` | ✓ |
| runs with the LLM seam unavailable | `TestSessionRunsWithTheModelUnavailable` | ~ weak |

The LLM row is a weak pin: the loop never calls the model, so the test degenerates to "a one-word session records one event" unless a future loop starts reaching for it. Defensible as a regression guard; worth one sentence in the test saying so.

## 6. Architectural notes

- **ARCH-DRY — pass, with two flags.** `puretest` is the model. Flagged: the duplicate `withStore` (measured), `anyTime` spelled as `store.Word{}.FirstSeen` (BR-19), `skipForm` duplicating `fakeForm` (BR-8).
- **ARCH-PURE — pass for `play`, one flag at the seam.** The package imports nothing at all and the guard enforces it. `playSession` re-enters raw mode on `os.Stdin` rather than the file it was handed (`play_loop.go:140`) — the single hard-wired IO reference in an otherwise fully injected loop, and the *only* reason the re-entry error report cannot be pinned (BR-17).
- **ARCH-PURPOSE — flag.** Round 3 named the class and handed over the enumeration; the commit fixed three named instances and swept none of the class. The `puretest` extraction is complete on the consumer axis but its own claim is still false in two artifacts (BR-2).
- **ARCH-MOCK — pass at the seam, flag at the wiring.** `fakePlayer`, `fakeCDN`, `store.Mem` and `fakeDictionary` all sit on the boundary production uses — I drove the audio path end-to-end through `playSession` with `raw == nil` and got `player calls = 1`. The fake is correct; `playRig` (play_loop_test.go:33) just hard-codes `noAudio: true` and never asks it anything.

## 7. Plan revision recommendations

The plan still has **no `## Revisions` section** and zero ticked steps across both chunks. It needs one entry covering: `Verdict` lives in `question.go`, not `session.go`; `play` imports **nothing** (not "store, schedule and pure stdlib") and deliberately omits `StoreSymbolsOnly`; `puretest` needs a row in the pure-entities table and `Input`/`Outcome`/`InputDrop` need rows; Task 2 Step 1's "a skip emits a record outcome with `Skipped`" and "the last answer emits quit" are both superseded; Task 3 Step 1's "the loop drops it" contradicts "ONE filter, in `Apply`". Then tick Chunk 1 Steps 1–9 and Chunk 2 Steps 1–10.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      go test still reports "cmd/define/puretest [no test files]"; issue:183 and define-learn.md:527 both still claim the negative cases are verified in the tree.
  - id: BR-3
    disposition: not-addressed
    note: |
      session.go untouched since round 3 — no Outcome.SessionDone, no doc on Apply/Session.Done, OutcomeDone still says only "the session is over".
  - id: BR-4
    disposition: not-addressed
    note: |
      plan:21/26/48 byte-identical; still no puretest, Input or Outcome rows, and Verdict is still filed under session.go.
  - id: BR-5
    disposition: not-addressed
    note: |
      plan:98 and plan:122 unchanged; the plan still instructs the superseded skip contract that M2 was executed from.
  - id: BR-6
    disposition: not-addressed
    note: |
      session_test.go:74 still named TestSkipAdvancesButRecordsNothing while asserting Index == 0.
  - id: BR-7
    disposition: not-addressed
    note: |
      recall.go:33 unchanged; no Log or plan entry records that form 2.1 has no skip key.
  - id: BR-8
    disposition: not-addressed
    note: |
      skipForm still at session_test.go:220 beside fakeForm.Grade('3') at :213.
  - id: BR-9
    disposition: not-addressed
    note: |
      question.go:45 and the Verdict doc at :15-21 unchanged; the zero value is still silently Skipped.
  - id: BR-10
    disposition: addressed
    note: |
      stderrOf added (puretest.go:126) and used at both go list sites; unpinned only because puretest still has no test file (BR-2).
  - id: BR-11
    disposition: not-addressed
    note: |
      grep -c '^- \[x\]' on the plan returns 0; both chunks are entirely unticked and there is still no "## Revisions" section.
  - id: BR-12
    disposition: not-addressed
    note: |
      playRig still hard-codes noAudio true and count 20; I verified both rows are pinnable today with the existing fakeCDN/fakePlayer seams.
  - id: BR-13
    disposition: not-addressed
    note: |
      run() is now driven with "--play sycophantic", but that returns 2 at the usage switch and never reaches the dispatch, withStore or count threading.
  - id: BR-14
    disposition: addressed
    note: |
      Mutation-verified twice in a scratch copy — removing the case reddens the test, and so does moving the dispatch back above the switch.
  - id: BR-15
    disposition: addressed
    note: |
      README gains a key table naming d and -count and its transcript now matches draw()'s actual line; atlas records InputDrop, crlfWriter and cancel-before-select.
  - id: BR-16
    disposition: addressed
    note: |
      Swept as the class — play_loop.go:141-148 reports and ends, puretest uses stderrOf at both sites; NOT pinned by a test, which the new claim-without-failing-test finding names.
  - id: BR-17
    disposition: not-addressed
    note: |
      play_loop.go:140 still calls enterRaw(os.Stdin); this is now also the sole blocker on pinning BR-16's fix.
  - id: BR-18
    disposition: not-addressed
    note: |
      play_loop_test.go:218 unchanged; two coin flips still have to land wrong for the missing guard to redden it.
  - id: BR-19
    disposition: not-addressed
    note: |
      play_loop.go:224 still spells the sentinel store.Word{}.FirstSeen.
  - id: BR-20
    disposition: not-addressed
    note: |
      The atlas now describes the key mapping, but no artifact records it as a CONSTRAINT on the forms in #7/#12/#13, and BR-7's sibling is still unrecorded.
findings:
  - id: new
    severity: Important
    family: claim-without-failing-test
    title: |
      The rule from round 3 was written down and then broken by the commit that closed round 3 — four unpinned claims remain and I measured that three are pinnable today
    detail: |
      This is the 3rd finding in family claim-without-failing-test (BR-2, BR-12). Do NOT fix this
      instance — the rule is: a claim in an artifact (a Done-when tick, a Log sentence, a fix
      note) is complete only when a named test in the tree fails without it. The enumeration,
      walked in full and measured, is five sites. (1) puretest's three guards, zero tests, both
      the issue Log and the project entry still assert otherwise. (2) the audio Done-when row.
      (3) the -count Done-when row. (4) the --play dispatch through run(). (5) the raw-mode
      re-entry report added by BR-16's own fix. I built a scratch copy and proved 1-4 need no
      structural change: swapping playRig's noAudioSource for the existing newAudioRig fake made
      "audio plays before reveal" pass with player calls = 1 and raw == nil, and a 3-word deck
      with count 2 offered exactly 2 questions. Only site 5 needs a change, and it is exactly
      BR-17 — enterRaw is not injected, so the failure cannot be simulated. Fix the rule: write
      the row-to-test map into the issue, close 1-4 in this round, and either inject enterRaw or
      state in the artifact that site 5 is verified by reading.
  - id: new
    severity: Important
    family: plan-checkboxes-not-ticked
    title: |
      The M1 row is ticked although the M1 boundary review blocked with four Importants that are still open, and no verdict trailer or close line exists for it
    detail: |
      This is the 2nd finding in family plan-checkboxes-not-ticked (BR-11). Do NOT fix this
      instance — state the rule: a checkbox in a tracker artifact asserts an EVENT, and it is
      ticked only when the evidence for that event exists. BR-11 is the same rule with the sign
      flipped (work done, box unticked). Measured: issue:69 reads "- [x] M1", but the gate ledger
      records round 2 (boundary M1) as blocked:true with BR-2/3/4/5 raised; git log over the whole
      window carries exactly one Review-Verdict trailer (REWORK, on HEAD); and grep for "closed
      M1" in the issue returns nothing. AGENTS.md 3 says an Mx row commits to its own
      milestone-close producing both. So M2 was built on top of a boundary that never cleared,
      which is how four M1 findings reached round 4 alive.
  - id: new
    severity: Minor
    family: duplicate-initialisation
    title: |
      main.go:417 opens the store a second time on the --play path — measured at 2 calls, against 1 on every other path
    detail: |
      Moving the dispatch below the switch to fix BR-14 left the branch's own
      d = d.withStore(opt, stderr) in place, three lines below the unconditional one at main.go:408.
      I probed it with a counting newStore: --play calls it twice, a plain lookup once. withStore's
      nil guards discard the second storeDeps, so nothing is corrupted, but openStore re-runs Getwd
      and builds a second store.NewYAML plus a second vocabulary set — and on the Getwd-failure path
      it prints "define: no working directory" twice. Delete line 417. Note the entry-path test
      BR-13 asks for is exactly what caught this.
  - id: new
    severity: Minor
    family: inconsistent-exit-status
    title: |
      Losing the terminal mid-session reports to stderr and exits 0, while failing to enter raw mode at the start exits 1
    detail: |
      play_loop.go:146-147 prints "define: lost the terminal after playback" and then returns
      finish(), which is always 0; play_loop.go:57-59 returns 1 for the same class of failure one
      screen earlier. Defensible (the session ran and its events are recorded) but the two should
      agree or the difference should be stated in the comment that already explains why ending is
      honest.
  - id: new
    severity: Minor
    family: budget-counted-before-filter
    title: |
      A deck word the dictionary no longer knows has already spent a -count slot before it is skipped, and that branch has no test
    detail: |
      play_loop.go:204-211 asks schedule.Queue for opt.count keys and then drops any the dictionary
      cannot resolve, so a learner with three stale entries and -count 20 gets a 17-word sitting
      with no explanation beyond a stderr line. I hit this accidentally while probing -count: three
      deck words, budget 2, one question. Either re-fill from the queue or say in the comment that
      a stale entry costs a slot. No test enters the branch today.
```

---

## Re-review — 2026-08-27T11:44:19-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..6442c6a850476addaf97d345aef5c9aae1cc98b4 |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-08-27T11:44:19-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

This round fixed three of the twenty open findings and left the rest untouched. BR-2 and BR-3 are genuinely closed — I verified both by reverting in a scratch copy (`ImportsOnly`'s `t.Errorf` gutted → `TestImportsOnlyRejectsAnIOImport` reddens; `SessionDone: s.Done` dropped from `advance` → `TestTheOutcomeThatEndsTheSessionSaysSo` reddens), and BR-4's plan-table corrections are all present. What blocks SHIP is that the round answered the *instances* while the *rules* those findings stated were written down and then not swept: BR-21 named a five-site enumeration and closed one site; BR-22 said a ticked checkbox asserts an event, and the M1 row is still `- [x]` with no `Review-Verdict` trailer and no `closed M1` log line, while every checkbox in the durable plan (Chunk 1 Steps 1–9, Chunk 2 Steps 1–11) is still `- [ ]`. Measured with `go test -coverprofile`: the audio block (`play_loop.go:131.37,151`) is at **count 0**, and the `--play` dispatch body (`main.go:416.15,419.3`) is at **count 0** — so two ticked Done-when rows and the flag wiring still have no test that fails without them. `go build`, `go vet` and `go test ./cmd/define/...` are all green (90.5% statement coverage in `cmd/define`).

## 1. Strengths

- **`puretest` is now a guard that has been seen to fail.** `cmd/define/puretest/puretest_test.go` with the `T` interface + `recorder`, and the committed known-bad fixtures — `testdata/impure` (imports `os`, calls `store.NewYAML`) and `testdata/clocky` (imports only `time`, calls `time.Since`, the case an import list structurally cannot catch). `TestGuardsRefuseToPassVacuously` even pins the fatal-on-nothing-to-check path. This is the correct answer to the gap and it is the strongest thing in the window.
- **The extraction itself (ARCH-DRY).** `schedule/purity_test.go` went from 193 lines of inline guard to a 38-line call site, and `play` reuses the same body. With `#7`/`#12`/`#13` each adding a form package, this is the right call made at the right time.
- **`play` is genuinely pure and provably so** — `purity_test.go:21` passes an *empty* allowlist and it holds. The `Grade`-on-the-form design is pinned by `TestSessionIsFormAgnostic` (session_test.go:174), which drives the same table through a fake using digits *and* asserts 2.1's own `y` means nothing there. That is the honest version of "a second form needs no loop change".
- **`TestSessionOutputIsAllCRLF` (play_loop_test.go:267)** asserts the property a byte capture *can* prove, and its comment records exactly why the pty smoke test missed the original defect. Also good: it fails loudly if there are zero newlines, so it cannot pass vacuously.
- **`atlas/define.md`** gained 84 lines covering `Question`, the `main.Key` boundary, the one-place skip filter, `InputDrop`, `crlfWriter`, and check-cancel-before-select. Genuinely useful map, not a changelog.

## 2. Critical findings

None.

## 3. Important findings

**BR-5 (still open) — `workshop/plans/000006-vocab-play-plan.md:102,126`.** `git diff d008145 HEAD -- <plan>` shows only two hunks, both in Core concepts. Task 2 Step 1 still reads *"a skip emits a record outcome with `Skipped`"* and *"the last answer emits quit"* — both superseded (`advance` emits `OutcomeNone` for `Skipped`, and `OutcomeRecord{SessionDone:true}` for the last answer). Task 3 Step 1 still reads *"the loop drops it"*, contradicting plan:70's "ONE filter, in `Apply`". This is the sixth round PQ-3/BR-5 has survived.

**BR-12 / BR-13 / BR-21 (still open) — the row-to-test map was never written.** Measured this round: `play_loop.go:131.37,142.7` and everything through `:151` sit at coverage 0 because `playRig` (play_loop_test.go:33) hard-codes `noAudio: true` and installs `noAudioSource{}`, so `fakePlayer` is wired to the seam and never asked anything (ARCH-MOCK: a fake that production and test share only nominally). `-count` is passed as 20 against 1–2 word decks, so neither the bound nor the default is read. `main.go:416.15,419.3` is at 0 — `TestPlayWithAWordIsAUsageError` returns 2 at the switch, three lines above the dispatch, so nothing exercises `opt.count` threading or `withStore`. `runPlay`'s terminal check (`play_loop.go:49–60`) is also uncovered and is trivially reachable with a non-`*os.File` stdin.

**BR-22 (still open, as a class) — the rule is stated, the enumeration is unswept.** `workshop/lessons.md` gained the right rule (*"A ticked `Mx` with no `Review-Verdict:` trailer and no close line in the Log is a claim with no evidence"*), and the issue gained a Revisions entry. But `git log --format=%B` over the whole window carries exactly one `Review-Verdict` trailer (REWORK, on 7878248), `grep "closed M1"` on the issue returns nothing, and issue:69 is still `- [x] M1`. BR-11's sibling — every plan checkbox — is still `- [ ]`. Writing the rule and not applying it to the sites the finding already enumerated is the ARCH-PURPOSE failure the rule was about.

**NEW (Important) — mode flags are only guarded against a *word*, not against each other; `define -raw --play` runs a full session and records nothing.** *This is the 2nd finding in family `mode-silently-ignores-argument`.* Do not fix the instance — state the rule: **when two flags on one line cannot both be honoured, the binary refuses with a usage error rather than silently picking one.** BR-14 fixed one pairing (mode + word); the enumeration over the current flag surface is: `--play`+word ✅, `--reflect`+word ✅, `-forget`+word ✅ — and `--play`+`-raw` ❌, `--play`+`--reflect` ❌ (main.go:416 wins), `--play`+`-forget` ❌ (main.go:410 wins), `--reflect`+`-forget` ❌, `--llm-check`+anything ❌ (main.go:362 returns first). The `-raw` one has teeth: `openStore` (main.go:188) branches only on `noCapture`, so with `-raw` the deck is non-nil, `runPlay`'s `d.deck == nil` guard does not fire, a full session runs, and every `CaptureReview` returns at `decideCapture(...) == captureNothing` (capture.go:124) with no message — the learner reviews twenty words, sees "18 right, 2 wrong", and nothing is on disk. The plan's own reasoning at plan:80 (*"Reviewing a deck it cannot read would be theatre"*) enumerated two producers of that state and missed the third.

## 4. Minor findings

- **BR-6** still open — `session_test.go:74` is still `TestSkipAdvancesButRecordsNothing` while asserting `s.Index != 0`. A comment was added; the name still says the opposite of the assertion.
- **BR-7 / BR-20** still open — nothing in the issue, plan or atlas records that form 2.1 has no skip key, or that `toInput` (play_loop.go:164–180) reserves Enter, space, `d` and `D` from every future form. `#7` picks its keys against this constraint.
- **BR-8** still open — `skipForm` (session_test.go:220) duplicates `fakeForm.Grade('3')` (session_test.go:214).
- **BR-9** still open — `Verdict`'s doc (question.go:15–20) still says nothing about `Skipped` being the zero value returned alongside `ok == false`.
- **BR-17** still open — `play_loop.go:140` re-enters raw mode on `os.Stdin`, not the `*os.File` `runPlay` was handed. The one hard-wired IO reference in an otherwise injected loop, and the reason BR-21's site 5 cannot be pinned.
- **BR-18** still open — `play_loop_test.go:218` reddens only ~1 run in 4 with the guard removed.
- **BR-19** still open — `anyTime = store.Word{}.FirstSeen` (play_loop.go:224) vs the plain `time.Time{}` spelling at reflect.go:250.
- **BR-23** still open — `main.go:417` opens the store a second time on the `--play` path, three lines below the unconditional `d = d.withStore(opt, stderr)` at main.go:408.
- **BR-24** still open — `play_loop.go:146` returns 0 for a lost terminal; `play_loop.go:58` returns 1 for the same failure one screen earlier.
- **BR-25** still open — `play_loop.go:204–211` spends a `-count` slot on a word the dictionary cannot resolve, then drops it. Related: `todaysQuestions` prints the *same* "nothing due today" line at :199 and :218, so a queue where every word failed lookup reports "nothing due" when the truth is "nothing resolvable". Both branches uncovered.
- **NEW (Minor, family `artifact-revised-without-revision-entry`)** — the durable plan was rewritten in place at 6442c6a (Core concepts table, Test-surface paragraph) with no `## Revisions` section. AGENTS.md §1 requires append-with-timestamp-and-delta for `plan` artifacts; the issue file got its Revisions entries, the plan did not.
- **NEW (Minor)** — *This is the 2nd finding in family `docs-not-updated-for-new-surface`.* Do not fix the instance — the rule is: **every user-observable behaviour a window introduces is described where the user would look for it, in the same round.** Enumerating this window's user-observable surface: `--play` ✅, the key table ✅, `-count` ✅, no-deck line ✅, nothing-due line ✅ — and *audio during a session* ❌. README:41–62 never mentions that the pronunciation plays on every reveal or that `-no-audio` applies to a session; README:35 documents `-no-audio` as a lookup flag only. This is the Spec's headline default-on behaviour and the same row BR-12 shows has no test.

## 5. Test coverage notes

- `cmd/define` is at 90.5%; the uncovered set in `play_loop.go` is coherent and small: `runPlay`'s terminal setup (38–73), the audio branch (131–151), the space/`d` arms of `toInput` (172, 179), both error branches of `todaysQuestions` (186–195, 206–210), and the all-words-unresolvable path (217–219).
- `capture_test.go`'s four new `CaptureReview` tests are well-layered — the verdict, normalisation, the `noCapture` gate, and "must not touch the deck" (which pins a real hazard: a review that upserted would inflate the lookup count `schedule.Queue` orders by).
- `TestUngradedKeyNeverReachesTheCapturer` / `TestGradedKeyReachesTheCapturerExactlyOnce` (play_loop_test.go:185, 203) are a good pair — the second is what stops the first passing on a loop that records nothing. The Log's account of *why* the spy replaced the event-log assertion is correct and worth keeping.
- `TestSessionRunsWithTheModelUnavailable` asserts the Done-when but would also pass on a loop that called the model and swallowed the error. Acceptable for the row as written; noting it because the row's claim is "never reaches for it".

## 6. Architectural notes

- **ARCH-DRY — flag.** Pass on the `puretest` extraction, which is the window's best structural work. Flagged at BR-8 (duplicate double), BR-19 (two spellings of the zero-time sentinel), BR-23 (duplicate `withStore`), and the duplicated "nothing due today" string.
- **ARCH-PURE — flag.** `play` is a real pure core with an enforced boundary, and `Apply`/`advance` keep the skip filter in exactly one place. The single violation is BR-17: `playSession` reaches `os.Stdin` directly instead of the file it was handed, which is both an injection break and the reason one claim in this issue cannot be pinned.
- **ARCH-PURPOSE — flag, and this is the boundary's main problem.** Three findings (BR-12, BR-21, BR-22) each named a class and supplied its enumeration. This round closed the sites of BR-2/BR-3/BR-4 and wrote two excellent `lessons.md` entries, but swept none of the three enumerations. A rule recorded and not applied to the sites the finding already listed is the instance-not-the-class pattern one level up.
- **ARCH-MOCK — flag.** `fakePlayer`, `noAudioSource`, `store.Mem` and `FixedClock` are all proper seams, but `playRig` sets `noAudio: true`, so production flow and test flow do **not** share the audio boundary — the fake is installed and never consulted. `enterRaw` (a terminal dependency) has no seam at all. Conversely, `puretest` shelling out to the real `go list` against committed fixtures is the right call for a test-only helper and reads as a live conformance check — no finding there.

## 7. Plan revision recommendations

1. **`## Revisions` — 2026-08-27, close round 5: the superseded skip contract, finally.** Task 2 Step 1: replace *"the last answer emits quit; … a skip emits a record outcome with `Skipped` and does NOT demote"* with "the last answer emits `OutcomeRecord` with `SessionDone`; a skip emits `OutcomeNone` and does not demote". Task 3 Step 1: replace *"the loop drops it"* with "`Apply` emits no record outcome for it". Reason: PQ-3/BR-5, open six rounds; M2 was executed from these steps.
2. **`## Revisions` — the Core-concepts table was corrected in place at 6442c6a without an entry.** Record the delta (Verdict moved to `question.go`, `Input`/`Outcome` rows added, `puretest` row added, "imports NOTHING" replacing "imports store and schedule", store-symbol guard dropped from the Test-surface note) as a Revisions entry, per AGENTS.md §1.
3. **Tick what is done, in the plan.** Chunk 1 Steps 1–8 and Chunk 2 Steps 1–4, 5–7, 9b, 10 are all delivered and all still `- [ ]`. Steps 1b (audio row), 8 (entry-path enumeration) and 9 (mutation-check of the no-capture gate) are **not** delivered and must stay unticked — which is the point: the plan should distinguish them.
4. **Add the row-to-test map to the issue.** One line per Done-when row naming the test that fails without it. On today's tree, rows 3 (audio) and 4 (`-count`) have no such test, and neither does the `--play` dispatch.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Verified by revert in a scratch copy: gutting ImportsOnly's t.Errorf reddens TestImportsOnlyRejectsAnIOImport.
  - id: BR-3
    disposition: addressed
    note: |
      Verified by revert: dropping SessionDone from advance reddens TestTheOutcomeThatEndsTheSessionSaysSo.
  - id: BR-4
    disposition: addressed
    note: |
      Plan table corrected at 6442c6a - Verdict at question.go, Input/Outcome rows, puretest row, "imports NOTHING".
  - id: BR-5
    disposition: not-addressed
    note: |
      plan:102 still says a skip emits a record outcome and the last answer emits quit; plan:126 still says the loop drops it.
  - id: BR-6
    disposition: not-addressed
    note: |
      session_test.go:74 name unchanged; a clarifying comment was added but the name still contradicts the assertion.
  - id: BR-7
    disposition: not-addressed
    note: |
      No record of the no-skip-key decision in the issue, plan or atlas.
  - id: BR-8
    disposition: not-addressed
    note: |
      skipForm still at session_test.go:220 beside fakeForm.Grade('3') at :214.
  - id: BR-9
    disposition: not-addressed
    note: |
      question.go:15-20 still says nothing about Skipped being the zero value.
  - id: BR-11
    disposition: not-addressed
    note: |
      Every checkbox in the durable plan, both chunks, is still "- [ ]".
  - id: BR-12
    disposition: not-addressed
    note: |
      Measured: play_loop.go:131.37,151 at coverage 0; playRig still noAudio true + noAudioSource; -count still unexercised.
  - id: BR-13
    disposition: not-addressed
    note: |
      Measured: main.go:416.15,419.3 at coverage 0; only the usage-error path enters run() with --play.
  - id: BR-17
    disposition: not-addressed
    note: |
      play_loop.go:140 still re-enters raw mode on os.Stdin rather than the file runPlay was handed.
  - id: BR-18
    disposition: not-addressed
    note: |
      play_loop_test.go:218 unchanged; still reddens probabilistically with the guard removed.
  - id: BR-19
    disposition: not-addressed
    note: |
      play_loop.go:224 still spells the zero time as store.Word{}.FirstSeen.
  - id: BR-20
    disposition: not-addressed
    note: |
      No artifact records that toInput reserves Enter, space, d and D from every form.
  - id: BR-21
    disposition: not-addressed
    note: |
      Rule written to lessons.md, enumeration unswept - site 1 closed, sites 2-5 unchanged, no row-to-test map in the issue.
  - id: BR-22
    disposition: not-addressed
    note: |
      Rule stated in lessons.md and issue Revisions, but issue:69 is still "- [x] M1" with no Review-Verdict trailer and no "closed M1" log line.
  - id: BR-23
    disposition: not-addressed
    note: |
      main.go:417 still calls withStore a second time, nine lines below the unconditional call at :408.
  - id: BR-24
    disposition: not-addressed
    note: |
      play_loop.go:146 returns 0, play_loop.go:58 returns 1, for the same class of failure.
  - id: BR-25
    disposition: not-addressed
    note: |
      play_loop.go:204-211 unchanged; the branch is still uncovered and the fall-through prints a misleading "nothing due today".
findings:
  - id: new
    severity: Important
    family: mode-silently-ignores-argument
    title: |
      Mode flags are guarded against a word but not against each other, and "define -raw --play" runs a full session that records nothing
    detail: |
      2nd in this family (with BR-14), so fix the RULE, not the instance. The rule: when two flags on one
      line cannot both be honoured, the binary refuses with a usage error rather than silently picking one.
      I walked the enumeration over the current flag surface. Guarded: --play+word, --reflect+word,
      -forget+word. NOT guarded: --play+-raw, --play+--reflect (main.go:416 wins), --play+-forget
      (main.go:410 wins), --reflect+-forget, --llm-check+anything (main.go:362 returns first). The -raw
      pairing has teeth - openStore (main.go:188) branches only on noCapture, so with -raw the deck is
      non-nil, runPlay's deck==nil guard at play_loop.go:26 does not fire, a full session runs, and every
      CaptureReview returns at decideCapture(...)==captureNothing (capture.go:124) with no message. The
      learner reviews twenty words, sees the tally, and nothing reaches disk. plan:80 enumerated two
      producers of "there is no deck to review" and missed this third route to the same end state
      (ARCH-PURPOSE). No test covers any mode-plus-mode combination.
  - id: new
    severity: Minor
    family: artifact-revised-without-revision-entry
    title: |
      The durable plan was rewritten in place at 6442c6a with no "## Revisions" section
    detail: |
      git diff d008145 HEAD on workshop/plans/000006-vocab-play-plan.md shows two substantive hunks - the
      Core concepts table and the Test-surface paragraph - both overwriting prior text. AGENTS.md section 1
      requires a plan artifact revised mid-stream to append a Revisions entry (timestamp + reason + delta)
      rather than overwrite. The issue file received its Revisions entries this round; the plan did not, so
      the record of what the table used to claim survives only in the boundary ledger.
  - id: new
    severity: Minor
    family: docs-not-updated-for-new-surface
    title: |
      README's new --play section never mentions that the pronunciation plays on every reveal, or that -no-audio applies to a session
    detail: |
      2nd in this family (with BR-15), so fix the RULE, not the instance. The rule: every user-observable
      behaviour a window introduces is described where the user would look for it, in the same round. I
      walked this window's user-observable surface - --play (README:41), the key table (:54-59), -count
      (:62), the no-deck line, the nothing-due line are all documented; audio-during-a-session is the one
      that is not. README:35 documents -no-audio as a lookup flag only, and the --play section's closing
      line ("No key and no network") reads as if a session is silent. This is the Spec's default-on
      behaviour and the same row BR-12 measures as having zero test coverage.
```

---

## Re-review — 2026-08-27T13:04:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..aae338eb0c4f4dbfb5ac0c4d2ea73de900dd5ea8 |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-08-27T13:04:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shipped feature is sound: `go build ./...` and `go test ./cmd/define/...` are green, `play` is a genuinely pure package with enforced guards, the session state machine is clean, and I revert-verified three of this round's headline claims (mutating `playAnnounced` away reddens the audio test; hardcoding the budget reddens `TestCountBoundsTheSession`; removing the pre-select cancellation guard reddens `TestCancelledContextEndsTheSession` at round 0). Fifteen of twenty-four open findings are genuinely addressed, including every process defect this round targeted — the `[~]` milestone rows, the plan's `## Revisions` section, the `puretest` KIND reclassification, and the issue Log's stripped code spans. What blocks SHIP is that four of the nine survivors are repeats whose *exact* fix was written out verbatim in a prior round's disposition note and still is not in the tree: `atlas/define.md:1179` still says "`#6` needs the same three" seven lines above its own correction (BR-29, named verbatim at round 7); `puretest` still has no zero-import fixture, which I measured by revert — restoring a vacuity fatal leaves `go test ./cmd/define/puretest/` **green** and reddens only `cmd/define/play` (BR-41); `main.go:416.15,422.3` — the entire `--play` dispatch — is at coverage 0 in the profile (BR-13/BR-21); and `-raw --play` still runs a full twenty-word session, prints a tally, and writes nothing to disk with no guard and no test (BR-26). Every one of these is under thirty lines of work, but the pattern the gate exists to catch — instance fixed, named sibling left standing — is measurably still running.

## 1. Strengths

- **`cmd/define/play/session.go:151-176` — the skip filter lives in exactly one place, and the code says so.** `advance` owns the `Skipped → OutcomeNone` rule; the loop at `play_loop.go:125-131` never inspects a verdict. This is the PQ-3 decision the plan agonised over, and it landed correctly.
- **`cmd/define/play_loop_test.go:355-368` is the right fix for `-count`, not the plausible one.** The comment records that the *first* version of this test used a word the fake dictionary lacked, so the bound came from the corpus rather than the flag — a test that read as passing while exercising nothing. Confirmed by mutation: hardcoding `20` in place of `opt.count` reddens it.
- **`cmd/define/puretest/puretest.go:28-32` (the `T` interface) plus the owned `testdata/` fixtures.** Taking a minimal interface instead of `*testing.T` is what makes a guard assertable at all, and `testdata/clocky` (imports only `time`, calls `time.Since`) is precisely the hazard an import allowlist structurally cannot see. `TestGuardsRefuseToPassVacuously` is the part most suites skip.
- **`cmd/define/play_loop.go:63-73` — one `crlfWriter` over every byte, including `Render`'s.** The fix is at the right altitude: the first version put `\r\n` in its own format strings and forgot that most of a session's newlines come from `Render`. `TestSessionOutputIsAllCRLF` counts them rather than sampling.
- **`cmd/define/schedule/box.go:12-19` now states the guard count *per package* with the reason.** That is the durable form of BR-29's rule — the count is a fact about the package, not about the guards. The sweep just missed one artifact.

## 2. Critical findings

None.

## 3. Important findings

**BR-26 (repeat, `mode-silently-ignores-argument`) — `define -raw --play` runs a full session that records nothing.** `cmd/define/main.go:372-398`. The usage switch guards mode-plus-word three times and mode-plus-mode zero times. Measured chain: `openStore` (`main.go:184`) branches only on `opt.noCapture`, so under `-raw` the deck is non-nil and `runPlay`'s `deck == nil` guard at `play_loop.go:26` never fires; every `CaptureReview` then returns at `decideCapture(...) == captureNothing` (`capture.go:29`) with no message. `--play`+`--reflect` (`main.go:416` wins), `--play`+`-forget` (`main.go:410` wins), `--reflect`+`-forget` and `--llm-check`+anything are the rest of the enumeration. `capture_test.go:562` covers only `noCapture: true`, never `raw: true`. Fix the rule, not the site: add a mode-count check before the switch that refuses two modes on one line, and one table test over the pairs.

**BR-13 / BR-21 (repeat, `claim-without-failing-test`) — the `--play` dispatch has zero test coverage, and the row-to-test map two rounds asked for does not exist.** Measured: `go test ./cmd/define/ -coverprofile` gives `main.go:416.15,422.3 1 0`. The only `run()` test on this path, `TestPlayWithAWordIsAUsageError`, returns at `main.go:393` before reaching the dispatch. Nothing catches `opt.count` not being threaded, or the dispatch moving back above the argument switch — the regression BR-14 already shipped once. `grep -n "row-to-test"` on the issue and plan returns nothing. Fix: `run(ctx, []string{"--play"}, ...)` and `run(ctx, []string{"--play","-count","3"}, ...)` through production wiring, plus the Done-when-row → named-test table written into the issue.

**BR-41 (repeat, `production-code-used-as-test-fixture`) — `ImportsOnly`'s zero-import PASS is pinned by `cmd/define/play`, not by `puretest`'s own suite.** `cmd/define/puretest/puretest.go:41-47` documents zero imports as a deliberate pass. I reverted it in a scratch copy (added a `seen == 0 → Fatalf` vacuity guard): `go test ./cmd/define/puretest/` stayed **green**, and only `cmd/define/play`'s `TestPlayPurity/imports` went red — naming the wrong package. `testdata/pure` imports `sort`, so it cannot cover this. The trigger is already named in `puretest_test.go`'s own comment: the moment `#7` gives `play` a legitimate import, this coverage disappears silently. Fix: `testdata/nothing` with no imports + `TestImportsOnlyAcceptsAPackageWithNoImports`.

**BR-42 (repeat, `claim-without-failing-test`) — `store/yaml.go`'s Stat error branch is still at coverage 0, and rounds 7-9 produced no coverage enumeration at all.** Profile entry unchanged: `yaml.go:163.16,165.3 1 0`. The operable rule BR-42 recorded was "before closing a round, run coverage over the CHANGED lines and classify EVERY fix landing in a zero-coverage branch as pinned-this-round or unpinnable-and-why, produced FROM the profile." The issue's `## Log` stops at the round-6 entry; there is no round-7/8/9 entry and no such list. The branch itself is pinnable — a closed `*os.File` makes `f.Stat()` fail — or declare it unpinnable in the artifact, but the classification has to exist.

**BR-29 (repeat, `plan-table-contradicts-code`) — one artifact in the greppable enumeration still asserts the wrong guard count.** `atlas/define.md:1179-1180`: "`#5` wrote three purity guards inline; `#6` needs the same three…", contradicted at `atlas/define.md:1187` ("`schedule` takes all three; `play` takes two"). This is the exact line round 7's disposition note named. `box.go`, `question.go`, `play/purity_test.go`, `puretest.go` and `atlas:1142` were all corrected; this one was not. `grep -n "same three" atlas/define.md` returns exactly this line.

## 4. Minor findings

- **BR-25** (`budget-counted-before-filter`) — `play_loop.go:215-234` still asks `schedule.Queue` for `opt.count` keys and then drops unresolvable ones; three stale entries with `-count 20` silently yields a 17-word sitting. The comment at :225-227 still does not say a stale entry costs a slot. The all-fail branch got a test; the partial case did not.
- **BR-28** (`docs-not-updated-for-new-surface`) — `README.md:62` still closes the `--play` section with "No key and no network", never mentioning that the pronunciation plays on every reveal or that `-no-audio` applies to a session. `README.md:35` documents `-no-audio` as a lookup flag only.
- **BR-43** (`duplicated-guard-logic`) — `play_loop_test.go:523-529`'s `missingDict` duplicates what `dict_fake_test.go:51-56` already does for any word outside the corpus; `playRig(t, "zzznotaword")` writes the same test with no new double and exercises the corpus-backed fake instead (ARCH-MOCK).

## 5. Test coverage notes

- Measured this round: `cmd/define` 91.3%, `cmd/define/store` 84.7%. Both suites green, `go build ./...` clean.
- Zero-coverage blocks in the window's new code: `play_loop.go:38-73` (all of `runPlay`'s terminal setup — expected, needs a pty), `play_loop.go:167` (`*raw.sess = *again`, the successful re-entry — same reason), `play_loop.go:204-213` (deck/log read errors — injectable, not injected), `main.go:416-422` (the `--play` dispatch — BR-13), `yaml.go:163-165` (BR-42).
- Revert-verified as genuinely load-bearing: the audio-on-reveal test, the `-count` bound, the pre-select cancellation guard. Revert-verified as **not** load-bearing: `ImportsOnly`'s zero-import pass (BR-41).
- `capture_test.go:531-614` is the strongest new block in the window — four tests pinning event shape, key normalisation, the no-capture gate and the deck-must-not-move invariant, each against `store.Mem` with an injected clock.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, one flag.** `puretest` is the exemplar of this window: one guard body, `schedule/purity_test.go` down from 193 lines to 38, `play/purity_test.go` at 8, and no surviving copy in the tree. Flagged only at BR-43's test double.
- **ARCH-PURE — pass.** `play` imports nothing at all and the claim is enforced rather than asserted. `playSession` takes its writers, key channel and terminal as parameters; `runPlay` is the only thing that touches a descriptor. The one residual seam is `enterRaw`, still called directly rather than injected — the issue Log declares that honestly as verified-by-reading, which is the right disposition for it.
- **ARCH-PURPOSE — flag.** Same axis as the findings above: the guard-count sweep, the entry-path enumeration and the coverage classification each fixed the instance named and left an enumerable sibling the previous round had spelled out. When `#7` adds form 2.3, the reserved-key set (`play_loop.go:182-198`) and the `testdata/` known-bad-fixture convention are the two things it inherits — both are now recorded in `Question`'s doc comment and the atlas, which is the right place for them.
- **ARCH-MOCK — pass, with one deliberate exception worth stating.** `afplay` sits behind `fakePlayer`, the dictionary behind the captured-corpus `fakeDictionary`, the store behind `store.Mem`, the terminal behind `rawTerm`+`os.DevNull`. `puretest` shells out to the real `go list` with no fake — and that is correct here, not a gap: a faked `go list` would make the purity guards assert nothing, since the real package graph *is* the measurement. Worth a sentence in `puretest.go`'s doc so `#7`/`#12`/`#13` don't "fix" it.

## 7. Plan revision recommendations

The plan's Core-concepts tables now match the code — I checked every row against `grep -n "^type \|^func " cmd/define/play/*.go`, and `puretest` is correctly filed under Integration points with "Wraps: `go list` + the filesystem". No table revision is needed. Two additions:

- Append to `## Revisions`: the round 8-9 entry currently narrates the milestone collapse but not the *rule* it produced. Add the Done-when-row → named-test map BR-12/BR-21 asked for, as a table, so `#7`'s plan inherits the obligation rather than re-deriving it.
- `plan:56` (the Test-surface paragraph) should state the `testdata/` known-bad-**and**-known-good fixture obligation explicitly, including the zero-import case — that convention is the durable output of BR-2 and BR-33, and BR-41 is what happens when it is left implicit.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Both rows now pinned and measured — mutating playAnnounced away reddens TestRevealPlaysThePronunciationByDefault; hardcoding the budget reddens TestCountBoundsTheSession.
  - id: BR-13
    disposition: not-addressed
    note: |
      Coverage profile still shows main.go:416.15,422.3 at count 0 — no test drives run() down the --play dispatch.
  - id: BR-17
    disposition: addressed
    note: |
      rawTerm carries the descriptor; play_loop.go:153 re-enters on raw.f.
  - id: BR-18
    disposition: addressed
    note: |
      40 pre-revealed rounds; revert-verified red at round 0, and the comment states the measured 119/400 rather than claiming determinism.
  - id: BR-19
    disposition: addressed
    note: |
      play_loop.go:245-249 spells it as a plain time.Time with the reason recorded.
  - id: BR-20
    disposition: addressed
    note: |
      Reserved-key set recorded at question.go:67-70, plan:30-34 and atlas:1241.
  - id: BR-21
    disposition: not-addressed
    note: |
      Sites 1-3 and 5 are pinned or honestly declared; site 4 (--play through run()) is at coverage 0, and the row-to-test map is absent from both issue and plan.
  - id: BR-22
    disposition: addressed
    note: |
      issue:69-83 is now [~] with the unclosed-boundary history written down.
  - id: BR-23
    disposition: addressed
    note: |
      main.go:417 is a comment; the single withStore call is at :408.
  - id: BR-24
    disposition: addressed
    note: |
      play_loop.go:163-165 returns 1, pinned by TestLosingTheTerminalAfterPlaybackExitsOne.
  - id: BR-25
    disposition: not-addressed
    note: |
      play_loop.go:215-234 unchanged — no re-fill, and the comment still does not say a stale entry costs a -count slot.
  - id: BR-26
    disposition: not-addressed
    note: |
      No mode-vs-mode guard added; -raw --play still runs a full session recording nothing, and no test covers any mode pairing.
  - id: BR-27
    disposition: addressed
    note: |
      plan:178-203 carries a Revisions section.
  - id: BR-28
    disposition: not-addressed
    note: |
      README:62 still closes the --play block with "No key and no network"; session audio and -no-audio-in-a-session are undocumented.
  - id: BR-29
    disposition: not-addressed
    note: |
      Five artifacts corrected, but atlas/define.md:1179-1180 still reads "#6 needs the same three" — the exact site round 7's note named.
  - id: BR-35
    disposition: addressed
    note: |
      puretest moved to the Integration points table at plan:67, wrapping "go list + the filesystem".
  - id: BR-36
    disposition: addressed
    note: |
      issue:348-376 reads cleanly; all seven identifiers restored.
  - id: BR-37
    disposition: addressed
    note: |
      Same fix as BR-27 — the plan now has a Revisions section covering the nine rounds.
  - id: BR-38
    disposition: addressed
    note: |
      plan:121 and plan:165 are [ ]; issue:69-70 are [~] with the reason stated.
  - id: BR-39
    disposition: addressed
    note: |
      playSession's doc block moved below rawTerm at play_loop.go:87-91.
  - id: BR-40
    disposition: addressed
    note: |
      session_test.go:72-76 now describes the ungraded-key claim its body asserts.
  - id: BR-41
    disposition: not-addressed
    note: |
      Revert-measured: restoring the vacuity fatal leaves puretest's own suite GREEN and reddens only cmd/define/play. No testdata/nothing fixture exists.
  - id: BR-42
    disposition: not-addressed
    note: |
      yaml.go:163.16,165.3 still at count 0, and the issue Log has no round-7/8/9 entry and no coverage-derived pinned/unpinnable classification.
  - id: BR-43
    disposition: not-addressed
    note: |
      play_loop_test.go:523-529 unchanged; fakeDictionary already returns ErrNoEntry for any word outside the corpus.
```

---

## Re-review — 2026-08-27T13:25:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..1346a8ae95758b463d68f5a07385c10e4be3e337 |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-08-27T13:25:06-07:00 |
| verdict | REWORK |

## Review

I've completed the review. Cleaning up the scratch copy is done; here are the results.

```verdict
verdict: REWORK
confidence: high
```

The code in this window is genuinely strong — `play` is a real pure core with enforced guards, `puretest` is the right ARCH-DRY extraction, and the session state machine is tested to a standard most of this repo does not reach. Two of the three findings this round claimed to dispose actually hold up: I revert-measured BR-41 (restoring the vacuity fatal in a scratch copy reddens `TestImportsOnlyAcceptsAPackageWithNoImports` by name, and `go test ./cmd/define/puretest/` goes red rather than some other package), and BR-43's rebuttal is technically sound and now documented. What blocks SHIP is that six of the nine open findings are unchanged in the tree, and one of them is not a docs nit: I measured `-raw --play` and it runs a full session, prints `1 right, 0 wrong`, records **zero** review events, and the `d` key **permanently removes a word from the deck** — under the flag whose own contract at `capture.go:30-31` is "must not mutate the deck it happens to be standing in." Every remaining item is cheap; round 10 can be the last if all seven are taken together.

## 1. Strengths

- **`cmd/define/puretest` is the ARCH-DRY exemplar.** One guard body, two callers; `schedule/purity_test.go` collapsed 193 → 38 lines and `play/purity_test.go` is 8. The `T` interface at `puretest.go:34-38` is the minimal change that makes a guard assertable at all, and `TestGuardsRefuseToPassVacuously` covers the case most suites skip.
- **BR-41's fix is real, not plausible.** `testdata/nothing` is an owned fixture, and I confirmed by revert that the new test is what stands behind `ImportsOnly`'s zero-imports-is-a-pass argument — not `cmd/define/play` happening to import nothing.
- **`TestSessionIsFormAgnostic` (`play/session_test.go:170`) is the honest way to test the Done-when.** Driving the same table through `fakeForm`'s digits *and* asserting `y` means nothing to it is what makes "a second form needs no loop change" a property rather than a promise.
- **`TestUngradedKeyNeverReachesTheCapturer` spies on the capturer, not the log.** The comment names exactly why the log cannot discriminate (an `OutcomeNone` carries an empty word and `CaptureReview` drops those anyway) — that is a mutation that survived being turned into a test that bites.
- **`TestSessionOutputIsAllCRLF` states what bytes *can* prove.** "In raw mode there is no such thing as a bare newline" is the right assertion for a defect a byte-capture smoke test structurally could not see.

## 2. Critical findings

None new.

## 3. Important findings

All four are prior findings re-raised by disposition, not new ids — see the block below. The one new finding:

- **`workshop/projects/define-learn.md:506-510`** — the `### tools#6 M1` entry carries `**closed:** 2026-08-27` and `**actual:** 0.8h (M1)` for a boundary the issue's own Plan says in bold never closed ("neither boundary closed on its own"). This is the 4th in `plan-checkboxes-not-ticked`; the rule, not the instance, is what needs fixing — see the finding detail. Fix sketch: either drop `closed:`/`actual:` from the entry and retitle it as a work note, or keep it and add a line saying the boundary folded into the issue close, so the whole-issue actual the gate is about to measure is not double-counted against a hand-typed 0.8h.

## 4. Minor findings

- `toInput`'s space-reveals branch (`play_loop.go:190-191`) is at coverage 0 while README's key table documents it — folded into BR-21's note rather than raised.
- `--play` and `-count` appear only in the new prose section, not in README's ```sh flag block, though plan Task 4 Step 10 said "`--play` in the flag list" — folded into BR-28.
- `workshop/projects/define-learn.md:487` still describes `schedule` as "enforced by two guards"; true when written, three now. Historical, so noted only.

## 5. Test coverage notes

I ran per-package coverage profiles and merged them against the window's changed production lines. Uncovered changed lines:

| file | uncovered changed lines | read |
|---|---|---|
| `main.go` | **416-422** | the `--play` dispatch — BR-13, and a fix landing in a zero-coverage branch that neither half of the round-6 enumeration classifies |
| `play_loop.go` | 38-45, 49-73, 109-114, 119-120, 136-138, 167, 190-191, 197, 204-213 | `runPlay`'s terminal setup (needs a pty — fine), the `Forget` error path, space-reveal, the deck/log read error paths |
| `capture.go` | 131-133, 136-138, 174-175 | empty-key guard, `AppendEvent` warn path, `noopCapturer` — defensive |
| `play/session.go` | 109-112, 148 | unreachable defensive returns |
| `store/yaml.go` | 163-165 | BR-42's branch, now declared unpinnable in writing |

The BR-13 row is the one that matters: `run(ctx, []string{"--play","-count","0"}, …)` → "nothing due today"/0 and `run(ctx, []string{"--play"}, …)` → "needs a terminal"/1 pin both the dispatch and the count threading with a `newStore`-injected Mem deck, no pty needed.

## 6. Architectural notes

- **ARCH-DRY — pass, with one flag.** `puretest` is the model case. The flag is in `main.go:373-393`: three hand-written copies of the same "a mode plus a word is two commands on one line" rule, each guarding one flag against a *word* and none against another *flag*. The enumeration of pairs is what BR-26 asks for.
- **ARCH-PURE — pass.** `play` imports nothing and is guarded; `runPlay` is the thin shell. Minor note: `todaysQuestions` (`play_loop.go:202-243`) interleaves the budget decision with the dictionary IO, which is exactly why BR-25's spent-slot behaviour has nowhere pure to live.
- **ARCH-PURPOSE — flag.** The shadow-sweep: `puretest`'s consumers both derive from the single body ✓. But the guard-*count* claim still has a hand-maintained restatement that contradicts the source — `atlas/define.md:1179-1180` says "`#6` needs the same three" five lines above its own correct "`play` takes two". Rounds 7 and 8 both named that exact site; round 9 did not touch `atlas/`. Same axis on BR-42: the class-level rule landed in `lessons.md`, but the enumeration the rule prescribes was not run over this window, and `main.go:416-422` is the item it would have found.
- **ARCH-MOCK — pass.** `fakePlayer`, `okAudio`, `store.Mem`, `fakeDictionary` all sit behind the production seams; `TestLosingTheTerminalAfterPlaybackExitsOne` drives a genuinely-failing `enterRaw` through `os.DevNull` rather than simulating it. `puretest` execs `go list` without a fake, but the dependency is the toolchain `go test` already requires and the `testdata/` fixtures are the portable state — acceptable.

## 7. Plan revision recommendations

- **Task 4 Step 8 is ticked `[x]` and not delivered.** It commits to "Assert the flag reaches the loop through the same entry-path enumeration `#21` needed — one row per process entry that can start a session." Measured: `main.go:416.15,422.3` at count 0; the only `run()`-level `--play` test exits at the usage switch. Either untick it or land the two rows above; a ticked step whose content is a coverage claim is the same shape BR-38 caught on the gate steps.
- No other contradiction between the Core-concepts tables and the code — I verified every row's path and status, including `puretest`'s relocation to Integration points.

```findings
dispose:
  - id: BR-41
    disposition: addressed
    note: |
      Revert-verified: restoring the vacuity fatal reddens TestImportsOnlyAcceptsAPackageWithNoImports in puretest's own suite, not play's.
  - id: BR-43
    disposition: addressed
    note: |
      Rebuttal is technically sound (fakeDictionary keys on testdata/entries), and the why is now in the comment plus a lessons.md rule.
  - id: BR-13
    disposition: not-addressed
    note: |
      Measured again: main.go:416.15,422.3 at coverage 0. Only TestPlayWithAWordIsAUsageError enters run() with --play, and it returns 2 at the usage switch.
  - id: BR-21
    disposition: not-addressed
    note: |
      Sites 1-3 and 5 are pinned or honestly declared; site 4 (--play through run()) is still coverage 0, and no row-to-test map exists in issue or plan. README's "Enter or space" is a fifth unpinned claim - toInput's space branch (play_loop.go:190-191) is coverage 0.
  - id: BR-25
    disposition: not-addressed
    note: |
      play_loop.go:215-234 unchanged - no re-fill from the queue, and the comment at 225-226 still does not say a stale deck entry costs a -count slot.
  - id: BR-26
    disposition: not-addressed
    note: |
      No mode-vs-mode guard. MEASURED this round: with opt.raw set, a full session runs, prints "1 right, 0 wrong", records 0 review events, and `d` permanently removed sycophantic from the deck - under the flag whose contract at capture.go:30-31 is "must not mutate the deck it happens to be standing in". Severity is understated at Important.
  - id: BR-28
    disposition: not-addressed
    note: |
      README:62 still closes the --play block with "No key and no network", which is also false - speak() fetches the recording over the network, and audio is on by default. Session audio and -no-audio-in-a-session remain undocumented.
  - id: BR-29
    disposition: not-addressed
    note: |
      atlas/define.md:1179-1180 still reads "#6 needs the same three", contradicted by its own line at 1184. Rounds 7 and 8 both named this exact site; round 9's commit does not touch atlas/.
  - id: BR-42
    disposition: not-addressed
    note: |
      Half landed - the yaml.go judgment is written down and the class rule is in lessons.md. The operable step was not executed: I ran the coverage-over-changed-lines sweep myself and main.go:416-422 (the BR-14 dispatch move) is a fix in a zero-coverage branch classified in neither half, again.
findings:
  - id: new
    severity: Important
    family: plan-checkboxes-not-ticked
    title: |
      The project's tools#6 M1 entry records a close date and a hand-typed actual for a boundary the issue says never closed
    detail: |
      This is the 4th finding in family `plan-checkboxes-not-ticked` (BR-22, BR-38, and
      the M2 row). Do NOT fix only this instance. The rule, restated so it covers all
      four: a field whose whole content is a gate outcome — a ticked `Mx`, a
      `closed:` date, an `actual:` figure — may only be written by the gate that
      produced it, and when a boundary is abandoned every artifact carrying that
      boundary's outcome fields must be swept, not just the one a finding named.
      BR-22's fix changed the issue's Plan rows to `[~]` and wrote a paragraph
      explaining the collapse; the enumerable sibling in the OTHER artifact was not
      touched. Measured: workshop/projects/define-learn.md:506-510 carries
      `**actual:** 0.8h (M1)` and `**closed:** 2026-08-27` for tools#6 M1, while
      workshop/issues/000006-vocab-play.md:70-73 states in bold that neither
      milestone closed on its own and both are folded into the issue close. No
      `Review-Verdict:` trailer or `closed M1` Log line exists. The 0.8h is
      therefore hand-typed rather than measured by `sdlc actual`, and it will
      double-count against the whole-issue actual this close is about to adopt —
      which is precisely the velocity-ledger pollution the close gate's actual
      guard exists to prevent. The enumeration to sweep is small and greppable:
      `grep -rn "tools#6" workshop/projects/` returns the mvp_scope line, the task
      row (correctly unticked), and this entry.
```

---

## Re-review — 2026-08-27T13:40:58-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 6 — define --play: review loop + form 2.1 quick pass |
| repo | tools |
| issue file | workshop/issues/000006-vocab-play.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4cac0b95d83d54eab57b8747d40d09a2196bca1e..1859f7aa59d4b21860649449844ac95fa002d0e3 |
| command | sdlc close --issue 6 |
| reviewer | claude |
| timestamp | 2026-08-27T13:40:58-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The code this window ships is genuinely good — `play` imports *nothing* (measured: `go list` returns empty), the state machine is exhaustively table-tested, the skip filter lives in exactly one place, and `puretest` is a real ARCH-DRY win with its own known-bad fixtures that I revert-verified. Two of the eight open findings are properly addressed (BR-41 and BR-44 both check out under measurement). What blocks SHIP is that **six of the eight remain open, four of them Important, and none of the six was touched this round** — the last two commits are documentation and ledger work. In particular BR-26 is a live behavioural defect (`define -raw --play` runs a full twenty-word session and records nothing, silently) that has been open since round 3; BR-29's last artifact still contradicts itself three lines above its own correction; and BR-13's entry-path gap is measurable at `go tool cover` = 0 across the whole of `runPlay`'s body. All six fixes are cheap. One round should clear them.

## 1. Strengths

- **`cmd/define/play` imports nothing at all**, verified independently (`go list -f '{{join .Imports "\n"}}' .../play` → empty). That is the strongest available form of the purity claim, and `play/purity_test.go:19-26` enforces it from a shared body rather than a copy.
- **BR-41's fix is honest and I revert-measured it.** Reinstating a zero-import vacuity fatal in `ImportsOnly` reddens `TestImportsOnlyAcceptsAPackageWithNoImports` by name (`zero imports must PASS, got [no imports found in .../testdata/nothing]`). `testdata/nothing` is a real fixture, not a field set at zero call sites.
- **BR-44's fix swept the artifact rather than the sentence.** `workshop/projects/define-learn.md:506-510` now carries `**actual:** — folded into tools#6` and `**closed:** — this boundary never closed on its own`, and `grep -rn "tools#6" workshop/projects/` returns exactly three sites, all consistent (mvp_scope, an unticked task row at :188, this entry).
- **`play_loop_test.go:186-216` is the right pair of tests.** Spying on the capturer rather than the event log is the discriminating observable — the log genuinely cannot separate "records everything" from "records only graded answers", and the paired positive test stops the negative one passing vacuously.
- **`cmd/define/puretest/testdata/pure/pure.go:4-7`** states the reason a known-good case must be a fixture and not a production package, and that reasoning was then correctly carried to the second case in BR-41's fix.

## 2. Critical findings

None. Nothing in the window crashes, corrupts state, or drifts from a byte-faithful contract.

## 3. Important findings

**BR-26 (still open, `mode-silently-ignores-argument`) — `-raw --play` runs a whole session that records nothing, and now also mutates the deck while doing it.** `cmd/define/main.go:387` guards `--play` against a *word* but against no other mode. `capture.go:29` returns `captureNothing` on `opt.raw`, and `openStore` branches only on `noCapture`, so with `-raw` the deck is non-nil, `play_loop.go:26`'s guard does not fire, and every `CaptureReview` returns silently. New measurement this round: the session is not uniformly inert — `play_loop.go:136` calls `d.deck.Forget` *directly*, outside `decideCapture`, so under `-raw` the `d` key still deletes words from disk while the reviews are dropped. That is one flag honoured and one ignored inside a single session, which is exactly the `-raw`-means-two-things failure `main.go:381` documents. Unguarded pairings remain: `--play`+`-raw`, `--play`+`--reflect` (`:416` wins), `--play`+`-forget` (`:410` wins), `--reflect`+`-forget`.

**BR-13 (still open, `dead-entry-path`) — the `--play` dispatch and `-count` threading are at measured coverage 0.** `go tool cover` puts `runPlay` at 26.1%, with block `play_loop.go:38.2,73.50` — the interrupter, the terminal check, `enterRaw`, and the `playSession` call — entirely uncovered. The only `run()`-level test, `play_loop_test.go:440`, returns at the usage guard (`main.go:387`) and never reaches the dispatch at `main.go:416`. This is cheap now: `todaysQuestions` runs *before* the terminal check, so `run(ctx, []string{"--play"}, …)` with a non-`*os.File` stdin exits 1 with `--play needs a terminal`, and `run(ctx, []string{"--play","-count","0"}, …)` exits 0 with the empty-queue line — two observables that pin dispatch position and flag threading.

**BR-21 (still open, `claim-without-failing-test`) — the enumeration is 4/5 done and the row-to-test map was never written.** Sites 1 (`puretest` tests), 2 (audio, `play_loop_test.go:321,338`), 3 (`-count`, `:355`) and 5 (raw re-entry report, `:498`) are all pinned. Site 4 is BR-13 and is open. The third ask — write the Done-when-row→test map into the issue — has not happened: `grep -n "Test[A-Z]" workshop/issues/000006-vocab-play.md` returns two incidental mentions, no map. Measured cost of its absence this round: `toInput`'s space→`InputReveal` branch (`play_loop.go:190-191`) is at coverage 0 while README:55 documents space as a key, and the Log's "SMOKE-TESTED FOR REAL through a pty" paragraph has no committed artifact behind it. A map built from the coverage profile would have surfaced both.

**BR-29 (still open, `plan-table-contradicts-code`) — one artifact in the greppable enumeration still asserts the wrong guard count.** `atlas/define.md:1179-1180`: "`#5` wrote three purity guards inline; `#6` needs the **same three**", contradicted at `atlas/define.md:1187` ("`schedule` takes all three; `play` takes two"). `box.go`, `question.go`, `play/purity_test.go`, `puretest.go` and `atlas:1142` were all corrected; this is the same line round 9's disposition note named by number. Secondary residue from that same edit: `atlas:1142` now says "ENFORCED by three guards" and then enumerates only two in the following clause, with the store-symbol guard appearing 40 lines later.

**NEW — `--play` has no live pty conformance check, though the repo owns the harness and the defect that shipped was terminal-only (ARCH-MOCK).** `cmd/define/pty_conformance_test.go` already provides `startDefine(t, args...)` over `creack/pty` and pins "raw mode was really entered" and "the terminal is left cooked" for the editor; `dict`, `fetch`, `news`, `player` and `reflect` each have their own. `--play` is the newest raw-mode surface in the binary and the *only* defect it shipped — the diagonal CRLF cascade — was invisible to every in-process test and to a byte-capture smoke run, and was found by the operator looking at a real terminal (`play_loop_test.go:283-293` says so in its own comment). The in-process CRLF counter is a good proxy but cannot see raw-mode entry, key decoding through a real tty, or restoration on exit. This also overlaps `claim-without-failing-test`: the issue Log's "SMOKE-TESTED FOR REAL through a pty" is the scratch-run-then-discard pattern `workshop/lessons.md` names, applied to the one behaviour only a pty can observe.

## 4. Minor findings

- **BR-25 (still open, `budget-counted-before-filter`)** — `play_loop.go:215` asks `schedule.Queue` for `opt.count` keys and `:222-229` then drops unresolvable ones; three stale entries with `-count 20` silently yields a 17-word sitting. The comment at `:225-227` still does not say a stale entry costs a slot, and no test enters the partial-failure branch (the all-fail branch has one).
- **BR-28 (still open, `docs-not-updated-for-new-surface`)** — `README.md:41-62` never mentions that the pronunciation plays on every reveal or that `-no-audio` applies to a session; `README.md:35` documents `-no-audio` as a lookup flag, and `:62`'s closing "No key and no network" reads as though a session is silent. The atlas has it; the README does not.
- **NEW (`budget-counted-before-filter`, 2nd in family)** — `define --play -count 0` prints `define: nothing due today` and exits 0, naming the schedule as the cause when the budget produced the empty queue. `-count` also accepts negatives silently, while `-times`/`-sound` reject them at `main.go:326` with a usage error. This is the sibling of the branch BR-30's fix already corrected for dictionary failures.
- **NEW (`duplicated-guard-logic`, 3rd in family)** — `play_loop.go:38-47` is a verbatim copy of `repl.go:192-201` (detach-from-parent + install the interrupter sink + the signal goroutine). The comment names the original ("Same interrupt shape as repl (#16 D5)") without saying why it was copied rather than shared, and `repl.go:184-190` records at length the PQ-6 reasoning that a third copy would have to re-derive (ARCH-DRY).

## 5. Test coverage notes

- Measured with `go test ./cmd/define/ -coverprofile`: `runPlay` 26.1%, `playSession` 85.7%, `todaysQuestions` 87.5%, `toInput` 75.0%, `CaptureReview` 71.4%. Zero-coverage blocks in the diff: `play_loop.go:38-73` (all of runPlay's terminal setup), `:119-120` (unmapped key kind), `:136-138` (`Forget` error), `:167` (successful raw re-entry), `:190-191` (space→reveal), `:197`, `:204-207` (deck read error), `:209-213` (events read error).
- Full suite is green (`go test ./cmd/define/...` — 95s, dominated by pre-existing 30s stream-flush tests, not this window).
- `TestCancelledContextEndsTheSession` now states its probabilistic nature honestly with the measured numbers, which is the right resolution of BR-31 — the comment no longer claims something measurement contradicts.
- The `play` package's tests run with no IO at all, matching the plan's PURE classification; `puretest` is correctly filed under Integration points and its tests exec the real `go` toolchain against committed fixtures, which is the right shape for a guard.

## 6. Architectural notes

- **ARCH-DRY — flag.** `puretest` is the exemplar (one body, two callers, 193→38 lines in `schedule/purity_test.go`). Against that, the interrupt-setup block is copied verbatim into the second interactive surface; extract `detachedInterrupts(ctx, d) (context.Context, *interrupter, context.CancelFunc)` before a third caller exists.
- **ARCH-PURE — pass.** Zero imports in `play`, enforced not asserted; `Render` stays in the caller so `Recall` receives a finished string; `Apply` is input-in/state-out with every effect performed by the loop. No test needs a mock to run a pure entity.
- **ARCH-PURPOSE — flag.** Three enumerations named in earlier rounds are still one site short each: BR-29 (atlas:1180), BR-21 (site 4 + the row-to-test map), BR-26 (the mode-pair enumeration was written in round 3 and never swept). The pattern the Log itself diagnoses — "the enumeration gets written from memory rather than from the diff" — recurred once more this window in the round that sharpened the rule.
- **ARCH-MOCK — flag.** The dictionary, player, CDN, store and LLM all sit behind seams with stateful doubles and live conformance checks. The *terminal* has a conformance harness that `--play` does not use, and it is the dependency whose real behaviour this feature is most exposed to. Everything else here passes: `fakePlayer` keeps `afplay(1)` out of the suite, `store.Mem` boots from nothing, `fakeDictionary` is corpus-backed, and `missingDict` now documents why the near-fit was rejected.

## 7. Plan revision recommendations

None. The plan's Core-concepts table now matches the tree on every row I checked (`Question`/`Verdict` in `question.go`, `Recall`, `Input`/`Outcome`/`Session`/`Apply` in `session.go`, `runPlay` in `play_loop.go`, `CaptureReview` in `capture.go`, `--play` in `main.go`, `puretest` under Integration points), Steps 9 and 11 are correctly unticked, and the `## Revisions` section records the milestone collapse. The residual contradiction is in `atlas/define.md`, not the plan.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      Still no run()-level test reaching main.go:416; go tool cover puts play_loop.go:38.2,73.50 at 0.
  - id: BR-21
    disposition: not-addressed
    note: |
      Sites 1/2/3/5 pinned, site 4 still open, and the row-to-test map was never written into the issue.
  - id: BR-25
    disposition: not-addressed
    note: |
      play_loop.go:225-227 still does not say a stale entry costs a slot, and no test enters the partial-failure branch.
  - id: BR-26
    disposition: not-addressed
    note: |
      No mode-pair guard exists; measured additionally that -raw --play still deletes via play_loop.go:136 while dropping every review.
  - id: BR-28
    disposition: not-addressed
    note: |
      README.md:41-62 still never mentions playback during a session or that -no-audio applies to one.
  - id: BR-29
    disposition: not-addressed
    note: |
      atlas/define.md:1180 still says "#6 needs the same three", contradicted at :1187; atlas:1142 says three and enumerates two.
  - id: BR-42
    disposition: addressed
    note: |
      The yaml.go Stat branch is now classified unpinnable-and-why in the Log, and the rule is in lessons.md.
  - id: BR-44
    disposition: addressed
    note: |
      Project entry now carries no close date and no hand-typed actual; grep over workshop/projects/ returns three consistent sites.
findings:
  - id: new
    severity: Important
    family: missing-live-conformance-check
    title: |
      --play is the newest raw-terminal surface and the only one with no pty conformance test, though the harness already exists
    detail: |
      cmd/define/pty_conformance_test.go already provides startDefine(t, args...) over creack/pty and pins raw-mode entry
      and terminal restoration for the editor; dict, fetch, news, player and reflect each have their own conformance file.
      The single defect --play shipped was the CRLF cascade, invisible to every in-process test and to a byte-capture smoke
      run, and found only by the operator on a real terminal. The issue Log's "SMOKE-TESTED FOR REAL through a pty" is a
      hand run that was discarded, which is the scratch-verify pattern lessons.md names. ARCH-MOCK: the terminal is an
      external dependency we depend on, and this consumer has no live check at that seam.
  - id: new
    severity: Minor
    family: budget-counted-before-filter
    title: |
      define --play -count 0 reports "nothing due today", naming the schedule for an empty queue the budget produced
    detail: |
      This is the 2nd finding in family budget-counted-before-filter. Do NOT fix only this instance. The rule that covers
      it and BR-25 and the already-fixed all-lookups-fail branch: whenever the sitting offers fewer words than the deck
      made due, the message names the actual cause — budget exhausted, entries unresolvable, or genuinely nothing due —
      and "nothing due today" is reserved for the last of those. schedule.Queue returns nil for budget <= 0
      (queue.go:30), so play_loop.go:217 fires with words outstanding. Separately, -count accepts negatives silently while
      -times/-sound reject them at main.go:326.
  - id: new
    severity: Minor
    family: duplicated-guard-logic
    title: |
      play_loop.go:38-47 is a verbatim copy of repl.go:192-201, the detach-plus-interrupter-sink block
    detail: |
      This is the 3rd finding in family duplicated-guard-logic. Do NOT fix only this instance. The rule covering it,
      BR-8 and BR-43: before writing a body that does what an existing body does, reuse it; if you write a near-copy
      anyway, the comment states why the existing one was rejected, and if it cannot, use the existing one. Here the
      comment names the original ("Same interrupt shape as repl (#16 D5)") without saying why it was copied. repl.go:184-190
      records the PQ-6 reasoning a third copy would have to re-derive. Consolidation: detachedInterrupts(ctx, d)
      returning (ctx, *interrupter, cancel). ARCH-DRY.
```
