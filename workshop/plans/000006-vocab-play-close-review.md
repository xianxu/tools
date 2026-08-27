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
