# Boundary Review — tools#16 (whole-issue close)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | whole-issue close |
| milestone | — |
| window | ba0162169f48d72983ff7f21d3a6479c9e6535a3..eb624f311fa6fb98f8248c54a3932c8e30a2c4a4 |
| command | sdlc close --issue 16 |
| reviewer | claude |
| timestamp | 2026-08-24T20:42:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need. Measurements complete.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Nine rounds in, the code is in genuinely good shape and I confirmed it by reversion rather than by reading: the ask path is one `ask()` reached from all six cells, `askScoped` is one owner demonstrably reachable from *both* loops (omitting `interrupts.Set` reddens 4 tests / 10 counting subtests, spanning `replLines` and `runEditor`), BR-19's message-text rule is applied with all four cells verified red, BR-52's cancel-path fix reddens its row, and `nothingSays`/`lostTerminal` are each now a single site in the whole tree. `go vet ./...`, `go vet -tags conformance`, `go test ./...` and `go test -race ./cmd/define/...` are all clean. What keeps this off SHIP is two things, both cheap and both created by the round that fixed the findings it was answering. `TestCtrlCQuitsAgainOnceTheAnswerIsOver` — the *only* test defending BR-51's restore cell — is racy: it waits for the last text delta and then requires the scope to already be handed back, while a disk-backed `CaptureAsk` still runs inside the scope; measured **12 failures in 30 isolated runs** on unmutated HEAD. And `askScoped`'s new doc comment, whose whole thesis is "the deliverable is the enumeration," records `omit defer qcancel() → 1 test red` where the measured answer is **0 red and `go vet` silent** — one of four measured doc claims in this window that the code contradicts.

## 1. Strengths

- **`askScoped` (cmd/define/ask.go:50) is a real consolidation, not a tidy-up.** Replacing `interrupts.Set(qcancel)` with a no-op reddens `TestEditorCtrlCMidStreamReturnsToThePrompt` (both transports), `TestForcedAndUnforcedAsksShareOneWiring` (both routes), `TestAKeyTypedBeforeCtrlCDoesNotBlockTheReader`, and `TestTheAskWiringTable/{piped,editor}/the interrupt is scoped` — 10 cells across both loop shells from one function.
- **BR-19's rule is applied, not restated.** All four cells verified red individually: reintroducing the raw-string `\\` in `noteEmptyLiteral` reddens `TestWhatTheMessagesSay/a_bare_backslash/{one-shot,piped}`; deleting `nothingSays`'s note branch reddens all four rows; deleting `fail(2)` at cmd/define/repl.go:302 reddens both piped rows; disabling `truncateQuestion`'s elision reddens `TestWhatTheAskMessagesSay/a_long_question_is_elided`.
- **BR-20 and BR-21 closed at the class, and the tree proves it.** `grep 'press return to replay'` finds exactly one production occurrence (cmd/define/repl.go:118) — `replayInPlace` routes through `nothingSays` — and `grep 'lost the terminal'` finds exactly one (cmd/define/replraw.go:135). Uniqueness claims verified by enumeration, which is what the family asked for.
- **BR-52's fix is pinned where three earlier "fixes" were not.** Dropping `sess.recordExchange` from the cancel path reddens `TestAnAnswerTheUserReadSurvivesHowItEnded/stopped_by_the_user` — and the row test covers all three ways an answer can end.
- **ARCH-MOCK is closed for the newest interface method.** `Store.SetUserModel` lets `Mem` hold the state `YAML` holds, so `storetest.Suite`'s user-model row is falsifiable for both implementations. `llmtest.Fake` remains a wire-level httptest server, `askRig` uses a real YAML store in a temp dir, and the pty suite builds its own binary from the tree.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `TestCtrlCQuitsAgainOnceTheAnswerIsOver` fails 12 of 30 runs on unmutated HEAD** (cmd/define/askrun_test.go:831).

**This is the 2nd finding in family `unsynchronised-test-observation`.** BR-25 fixed the instance (data races reading a `bytes.Buffer`); do not fix only this test — the rule is broader.

**The rule: a test must synchronise on the state it asserts, not on a proxy that merely precedes it.** Here the proxy is `waitFor(... Contains(out.String(), "insincerely"))` — the text of the *last* delta in `stream-sample.sse` (line 35 of 45). The test then immediately writes `\x03` and requires the scope to already be restored. Everything between those two points is unsynchronised: the remaining SSE frames parse, `Stream` returns, `recordExchange` runs, and then **`runAsk`'s deferred `CaptureAsk` does a real `AppendEvent` to disk** (cmd/define/ask.go:147-151) — all inside `askScoped`, before `restore()`. Lose that race and `readKeys` swallows the `\x03` as scope-consumed, `quit` never closes, and the test fails at its 5s deadline with the exact message it prints for the real defect.

Measured: `-count=30` isolated → **12 failures**; under full-package load it is rarer (3 of 3 clean in a dedicated sweep, plus one spontaneous failure observed across ~10 full-package runs during this review). It is the only test in the file that drives a *completing* stream and then asserts post-scope state — the eight `Stall: true` tests never leave the scope, so none of them can race.

This matters beyond the noise: it is the sole defender of the cell BR-51 named, and the close's `--verified` evidence is a `go test ./...` run.

*Fix sketch:* wait on a happens-after marker for `restore()` rather than on the answer text. `askInSession` (cmd/define/replraw.go:164-165) writes `"\r\n"` then `draw()` **strictly after** `askScoped` returns, so polling for the post-answer prompt redraw is exact. Worth considering separately: moving the `CaptureAsk` disk write outside the scope would also shrink the window in which a user's real Ctrl-C is silently swallowed.

**I2 — four measured doc claims in this window contradict the code, all created by the last two rounds.**

**This is the 11th finding in family `doc-overstates-code`.** The rule has been stated five times; do not patch the four lines.

**The rule, in the shape this window needs it: a measured claim written into a comment is a claim, and it is falsified the same way any other absolute is — by running the mutation it names.** Measured:

1. **cmd/define/ask.go:43** — `omit defer qcancel()  1 test red — the question's context leaks`. Measured: `go test ./cmd/define/` → `ok  58.078s`, **0 tests red**, and `go vet ./cmd/define/` silent (exit 0) — `lostcancel` cannot fire because `qcancel` is passed as a value to `interrupts.Set`. This is exactly what BR-51's own detail measured and reported; the commit that answered it wrote the opposite into the table. The cell is a real (if small) context leak per question and it is undefended.
2. **cmd/define/ask.go:38 vs cmd/define/askrun_test.go:800** — the same commit gives two numbers for the same mutation: `omit interrupts.Set   10 tests red` and "omitting `interrupts.Set` reddens five tests". I measure 4 top-level / 10 including subtests. Two numbers in one commit is the signature of a recollection rather than a measurement.
3. **atlas/define.md:602-603** — "Each loop wraps that one call in its own closure — the raw loop's is where M2's streaming writer and **scoped interrupt** hang". Round 3 moved the scope out of both closures into `askScoped`, and cmd/define/replraw.go:142-143 says so explicitly: *"the interrupt SCOPE does not hang here — askScoped owns that."* The code comment was corrected and the atlas paragraph that owns the claim was left standing — BR-28/BR-44's rule, verbatim, a third time.
4. **cmd/define/store/mem.go:18** — "`SetUserModel` is #17's to add, and the conformance suite only needs 'absent reads as empty' until then". `SetUserModel` is implemented 73 lines below it (cmd/define/store/mem.go:91) by the same commit, and the suite writes through it.

A fifth, adjacent: a **configured but unreachable** model prints `define: no model configured; …` and exits 1 while the question **is** recorded as an `asked` event (probed against `http://127.0.0.1:1`: `code=1`, one event). README:97-101 keys "not recorded" on "no model configured", so the message and the record disagree in that cell, and it is the one cell `TestAQuestionIsRecordedWhateverBecameOfTheAnswer` does not enumerate (it covers answered / 400 / cancelled / no-seam).

## 4. Minor findings

- `carriedCR` (cmd/define/crlf.go:55) is unpinned: `c.lastWasCR = carriedCR(out, n, entryWasCR)` → `c.lastWasCR = lastWasCR` leaves the whole package green. The close commit changed `crlf.go` without touching `crlf_test.go`. The behaviour is correct — I probed it: a `"\r\nz"` short-written at 1 byte then retried with `p[1:]` yields `"\r\nz"`, not `"\r\r\nz"`.
- `submitLine`'s recall wiring is unpinned: `hist.Add(cmd.recallLine())` → `hist.Add(line)` (cmd/define/replraw.go:314) leaves the full suite green on two consecutive runs, restoring BR-12's `\how so` → `how so` inversion. `TestRecallPreservesWhatALineMeant` pins the pure function only, and `TestAQuestionIsRecalledByUpArrow/unforced` uses a line whose `recallLine()` equals its `word`.
- `Store.SetUserModel` has **zero production call sites** — its only caller is `storetest/suite.go:56`. Legitimate (it is BR-45's fix), but interface surface added for a test should be recorded as such.
- The five PTY conformance rows all **SKIP** here (`no pty available: operation not permitted`), so `TestPTYCtrlCMidAnswerKeepsTheSession`'s claim still rests on the implementor's note.
- `ReviewEvent.Found` has no `omitempty`, so every `asked` event carries `found: false` — a field with no meaning for a question, in a log `#17` folds over. Harmless today (both consumers filter on `EventLookedUp`).

## 5. Test coverage notes

Everything below was measured in a scratch `git archive` of `eb624f3`; the working tree was never modified and is clean.

| mutation | result |
|---|---|
| `askScoped`: omit `interrupts.Set` | RED — 4 tests / 10 subtests, both loops |
| `askScoped`: omit `defer restore()` | RED — `TestCtrlCQuitsAgainOnceTheAnswerIsOver` |
| `askScoped`: omit `defer qcancel()` | **GREEN**, `go vet` silent — I2(1) |
| cancel path: drop `recordExchange` | RED — `…SurvivesHowItEnded/stopped_by_the_user` |
| `noteEmptyLiteral` doubled backslash | RED — 2 rows |
| drop `nothingSays` note branch | RED — 4 rows |
| drop piped `fail(2)` for a bare hatch | RED — 2 piped rows |
| disable `truncateQuestion` elision | RED — 2 tests |
| delete `unavailable`'s `q.forced` branch | RED — `TestWhatTheAskMessagesSay/forced…` |
| `crlf`: revert `carriedCR` | **GREEN** |
| `submitLine`: `recallLine()` → `line` | **GREEN** (×2) |

`go test ./...` green; `go test -race ./cmd/define/...` green; `go vet ./...` and `go vet -tags conformance ./cmd/define/` clean.

## 6. Architectural notes

- **ARCH-DRY — flag.** The consolidations are real and I verified each is load-bearing: `askScoped`, `fail(code)`, `nothingSays`, `lostTerminal`, `recallLine`, `syncBuf`, `writeBytesAtomic`, and `capture.go` as the only non-test `AppendEvent` caller. The flag is BR-53's stated rule, unapplied: `crlfWriter.Write` materialises the translation into `out` and `consumed` (cmd/define/crlf.go:68) still re-derives it independently — two derivations of one fact, which is how they disagreed in the first place. `carriedCR` reads `out`; `consumed` does not.
- **ARCH-PURE — pass.** `readsAsQuestion`, `openerStem`, `truncateQuestion`, `parseREPLLine`, `recallLine`, `nothingSays`, `recentTurns`, `recentDeck`, `renderAskPrompt`, `consumed`, `carriedCR` and `session` are all deterministic and tested with no store, socket or terminal. `gatherAskContext` is a genuinely thin read whose only judgement is the nil-deck check.
- **ARCH-PURPOSE — flag on the class/instance axis.** The shadow-sweep over "the directory is the context" passes: one-shot, piped and raw editor all derive from the store and session, and all three are pinned by `TestTheAskWiringTable`. The flag is that two class-fixes stopped at the instance again — BR-47's enumeration was re-run and again missed the entities the same commit created, and BR-53's rule ("one derivation") was answered with a site fix.
- **ARCH-MOCK — pass.** Wire-level `llmtest.Fake`, real YAML store in a temp dir, `storetest.Suite` over both implementations with a setter so the fake can hold the real one's state, `deps.notifySignals` seaming `signal.Notify`, and a pty suite that builds what it tests. The only gap is environmental (pty unavailable here), not structural.

## 7. Plan revision recommendations

Three `## Revisions` entries are owed in `workshop/plans/000016-console-qa-plan.md`:

1. **The entity enumeration, re-run against the staged tree** (BR-47, third consecutive miss). The command the plan itself records lists `Store.SetUserModel` (created by `baee679`) and `carriedCR` (created by `eb624f3`); neither appears anywhere in the file — `grep -c SetUserModel` and `grep -c carriedCR` both return 0. Add both, and record the correction the misses imply: run the enumeration against the **staged tree**, not `base..previous-HEAD`, or an entity created by the same commit falls outside it every time.
2. **Boundary rounds 2–4** (BR-48). Still no entry for round 2's four forks (`TestThePipedLoopsAskWiring` folded into `TestTheAskWiringTable`; `replLines` gaining the interrupt scope the plan gave only to the raw loop; the pty suite building its own binary), for round 6's outstanding Task 11 item ("SIGINT through `repl`'s watcher **during a scoped stream**" is still unasserted), for round 3's `askScoped` consolidation and `Store.SetUserModel`, or for the close round.
3. **What M1 actually shipped** (BR-17, unchanged since round 3). Task 3's snippet at line 632 still reads `if !literal && readsAsQuestion(word)` where the code is `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; `askUnavailable(stderr, question)` is still named at lines 686 and 745 where the code has `unavailable(errOut, q)`; and `mayAsk`, `recallLine`, `nothingSays` and `lostTerminal` appear zero times.

Bookkeeping otherwise checks out: 60/60 plan checkboxes ticked, every issue Done-when row ticked, and `workshop/projects/define-learn.md` carries per-milestone rows with `actual:`/`closed:` for both M1 and M2.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Half (b) is now pinned; half (a)'s wiring still reverts green — submitLine's hist.Add(cmd.recallLine()) -> hist.Add(line) leaves the full suite green on two runs.
  - id: BR-17
    disposition: not-addressed
    note: |
      Unchanged verbatim — Task 3's snippet at line 632, askUnavailable at 686/745, and zero mentions of mayAsk, recallLine, nothingSays or lostTerminal.
  - id: BR-19
    disposition: addressed
    note: |
      All four cells verified red individually — the doubled backslash, nothingSays's note branch, the piped fail(2), and truncateQuestion's elision.
  - id: BR-20
    disposition: addressed
    note: |
      replayInPlace routes through nothingSays; one production occurrence of the replay sentence in the whole tree, and the parameter is inSession where true is correct.
  - id: BR-21
    disposition: addressed
    note: |
      One "lost the terminal" site in cmd/, via lostTerminal. The doc comment above it still predicts a fifth call site — that belongs to BR-50.
  - id: BR-47
    disposition: not-addressed
    note: |
      Third round, same failure mode — the enumeration was re-run and again missed what the same commit created: Store.SetUserModel and carriedCR are both absent from the plan.
  - id: BR-48
    disposition: not-addressed
    note: |
      Still no Revisions entry for round 2's four forks, round 6's Task 11 item, round 3's askScoped/SetUserModel, or the close round.
  - id: BR-49
    disposition: not-addressed
    note: |
      Unchanged — replRaw still passes the seam straight to readKeys with no policy, and 44 test call sites still pass nil.
  - id: BR-50
    disposition: not-addressed
    note: |
      All four residues present verbatim — replraw.go:132's "adds a fifth", capture.go's CaptureAsk guard, repl_test.go's t.Context() failure arm, ask.go's per-question Deck().
  - id: BR-51
    disposition: addressed
    note: |
      The restore cell is genuinely pinned — omitting defer restore() reddens TestCtrlCQuitsAgainOnceTheAnswerIsOver (verified). Two residuals raised separately: that test is racy, and the enumeration's qcancel row is false.
  - id: BR-52
    disposition: addressed
    note: |
      Dropping the cancel-path recordExchange reddens TestAnAnswerTheUserReadSurvivesHowItEnded/stopped_by_the_user (verified).
  - id: BR-53
    disposition: not-addressed
    note: |
      The site defect is fixed and behaviourally verified, but no test fails without carriedCR, and the rule the finding asked for is unapplied — Write and consumed are still two derivations.
findings:
  - id: new
    severity: Important
    family: unsynchronised-test-observation
    title: |
      TestCtrlCQuitsAgainOnceTheAnswerIsOver fails 12 of 30 runs on unmutated HEAD
    detail: |
      This is the 2nd finding in family unsynchronised-test-observation. BR-25
      fixed the instance (data races on a bytes.Buffer); state the rule instead.
      Rule - a test must synchronise on the state it ASSERTS, not on a proxy that
      merely precedes it. askrun_test.go:831 waits for "insincerely", the LAST
      text delta of stream-sample.sse (line 35 of 45), then immediately writes
      \x03 and requires the scope to already be restored. Everything between is
      unsynchronised: the remaining SSE frames parse, Stream returns,
      recordExchange runs, and runAsk's deferred CaptureAsk does a real
      AppendEvent to DISK (ask.go:147-151) - all inside askScoped, before
      restore(). Lose that race and readKeys swallows the \x03 as
      scope-consumed, quit never closes, and the test fails at its 5s deadline
      printing the message it reserves for the real defect. Measured on
      unmutated HEAD: `-count=30` isolated gives 12 failures; under full-package
      load it is rarer (3 of 3 clean in a dedicated sweep, one spontaneous
      failure across ~10 full-package runs during this review). It is the only
      test in the file that drives a COMPLETING stream and then asserts
      post-scope state - the eight `Stall: true` tests never leave the scope - and
      it is the sole defender of the cell BR-51 named, while the close's
      --verified evidence is a `go test ./...` run. Fix: wait on a happens-after
      marker for restore() rather than on answer text; askInSession
      (replraw.go:164-165) writes "\r\n" then draw() strictly after askScoped
      returns. Consider separately moving the CaptureAsk disk write outside the
      scope, which also shrinks the window where a user's real Ctrl-C is
      silently swallowed.
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      Four measured doc claims contradict the code, all created by the last two rounds
    detail: |
      This is the 11th finding in this family; the rule has been stated five
      times, so do NOT patch the four lines. Rule, in the shape this window
      needs it - a measured claim written into a comment is a claim, and it is
      falsified the same way any other absolute is: by running the mutation it
      names. Measured: (1) ask.go:43 records
      `omit defer qcancel()  1 test red - the question's context leaks`;
      measured, `go test ./cmd/define/` is ok 58.078s with 0 tests red and
      `go vet ./cmd/define/` silent, because qcancel is passed as a value to
      interrupts.Set so lostcancel never fires - which is exactly what BR-51's
      own detail reported, so the commit answering it wrote the opposite into
      the table whose thesis is "the deliverable is the enumeration"; the cell
      is a real per-question context leak and is undefended. (2) ask.go:38 says
      `omit interrupts.Set  10 tests red` while askrun_test.go:800, same commit,
      says "reddens five tests"; I measure 4 top-level / 10 with subtests - two
      numbers in one commit is the signature of a recollection, not a
      measurement. (3) atlas/define.md:602-603 still says the raw loop's closure
      "is where M2's streaming writer and scoped interrupt hang", which round 3
      falsified by moving the scope into askScoped - replraw.go:142-143 says so
      explicitly, so the code comment was corrected and the atlas paragraph that
      OWNS the claim was left standing, BR-28/BR-44's rule a third time.
      (4) store/mem.go:18 says "SetUserModel is #17's to add ... until then"
      while mem.go:91, same commit, implements it and the suite writes through
      it. A fifth, adjacent: a CONFIGURED but unreachable model prints
      "no model configured" and exits 1 while the question IS recorded as an
      asked event (probed against http://127.0.0.1:1), where README:97-101 keys
      "not recorded" on "no model configured" - and it is the one outcome cell
      TestAQuestionIsRecordedWhateverBecameOfTheAnswer does not enumerate.
```
