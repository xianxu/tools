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

---

## Re-review — 2026-08-24T21:05:46-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | whole-issue close |
| milestone | — |
| window | ba0162169f48d72983ff7f21d3a6479c9e6535a3..3ae98350bdb8bbce4af6cccb5948dc22c3e8b0bc |
| command | sdlc close --issue 16 |
| reviewer | claude |
| timestamp | 2026-08-24T21:05:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Ten rounds in, the code is genuinely solid and I confirmed it by reversion rather than by reading. BR-54's fix is real: `TestCtrlCQuitsAgainOnceTheAnswerIsOver` now passes **30 of 30** isolated runs (was 12 failures in 30) and still reddens under `omit defer restore()`, because it now waits on the prompt redraw — an effect that happens strictly after `askScoped` returns — instead of on answer text. BR-55's fix is real too, and better than claimed: all three observable cells of `askScoped` now redden the exact subtest the doc comment names them by. `go test ./...`, `go test -race ./cmd/define/...`, `go vet ./...` and `go vet -tags conformance` are clean, and I exercised the built binary end to end — every README exit-code absolute holds (`?`/`\` → 2 in both modes, `-raw '?why'` → 2, unforced/forced question with no key → 1) and the event log after six lines carries only the lookups, no question text and no empty-word records. Two things keep this off SHIP. The close commit introduced `unreachable` to split ErrUnavailable into "set it up" vs "wait it out", but ErrUnavailable has a third producer: measured through the wire fake, **401 and 403 now both say "the model did not answer"** — the one cell the user can actually fix, told to wait. And BR-47 fails a **fourth** time in the identical way: the plan's own Revisions entry says the enumeration "runs LAST" and reports 34 additions, which is the count at `993fccc..eb624f3` — it was run against the *previous* commit, and the close commit then created `unreachable` and `sayUnavailable`, neither of which is in the tables.

## 1. Strengths

- **`askScoped`'s enumeration is now the deliverable it claims to be** (`cmd/define/ask.go:41-51`). Each of the three observable cells reddens exactly the subtest its row names: `omit interrupts.Set` → `/the_sink_is_scoped_to_the_question_WHILE_it_runs`; `omit defer restore()` → `/the_sink_is_handed_back_when_it_returns` **and** `TestCtrlCQuitsAgainOnceTheAnswerIsOver`; `omit defer qcancel()` → `/the_question's_context_is_cancelled_when_it_returns`. Replacing counts with test names was the right call — a name is checkable, and I checked all three.
- **BR-54's fix removed the race rather than papering it.** The rewritten wait (`cmd/define/askrun_test.go:860-871`) synchronises on `prompt` appearing *after* `"insincerely"`, which `askInSession` writes strictly after `askScoped` returns. 30/30 clean, and each run dropped from timing-dependent to ~0.01s.
- **The ask-wiring table still holds after two more rounds of edits.** I re-ran two of its cells against the full suite: `replraw.go:161` `&sess` → `&session{}` reddens `TestTheAskWiringTable` + `TestAQuestionIsNotCaptured`; `main.go:366`/`:393` `stdout` → `io.Discard` reddens `TestTheAskWiringTable`. The fixture-driven table survived the churn a pile of one-offs would not have.
- **Routing-before-capture is verifiable outside the test suite.** Six one-shot runs against the built binary in a temp directory produced two `looked-up` events and nothing else — no question text, no empty-word record, correct UTC day-file naming.
- **The Core concepts rows that exist all resolve.** Every one of the 21 listed entities is at its stated path with its stated status, and every PURE row's test runs with no store, socket or terminal (`askctx_test.go`, `crlf_test.go`, `question_test.go`, `TestParseREPLLine`). The gap is missing rows, not wrong ones.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `unreachable` splits ErrUnavailable into two halves; it has three producers, and the credential one lands on the wrong side** (`cmd/define/ask.go:115`, reached from `ask.go:194`).

**This is the 12th finding in family `doc-overstates-code`.** The rule has been stated six times; do not patch the one message.

**The rule, in the shape this window needs it: a taxonomy branch must be exhaustive over what *produces* the error it dispatches on, not over the halves the author had in mind — and the enumeration is greppable in the producer, not guessable in the consumer.** `unreachable`'s own doc comment states the contract: *"the two need opposite responses — one is something to set up, the other something to wait out."* `internal/llm`'s `classifyStatus` produces `ErrUnavailable` from three distinct places, and its own comment on the third says the opposite of what the new message says: *"A missing or wrong credential is an operator configuration state, not a crash."*

Measured through `llmtest.Fake` (a scratch probe, four statuses):

```
status 401 -> code=1 stderr="define: the model did not answer; `how so` is not a word"
status 403 -> code=1 stderr="define: the model did not answer; `how so` is not a word"
status 503 -> code=1 stderr="define: the model did not answer; `how so` is not a word"
status 429 -> code=1 stderr="define: the model did not answer; `how so` is not a word"
```

So an expired or wrong `DEFINE_LLM_API_KEY` — the single ErrUnavailable cell a user can actually fix — is told to wait it out. `--llm-check` gets this right by not deciding: `llmcheck.go:69` prints `llm check failed after %s: %v`, surfacing the underlying error verbatim. The ask path now discards it entirely and substitutes a fixed sentence, so it gives strictly *less* information than the diagnostic surface beside it, and points the wrong way.

**Why the gap survived:** the cell added for `unreachable` (`askrun_test.go:619-645`) reaches around the wire fake to a raw socket — `envFor("http://127.0.0.1:1")`. That exercises only `classifyStatus`'s `status == 0` arm, so the 5xx and 401/403 arms the fake models perfectly well were never driven. **ARCH-MOCK**: the fake is the seam, and a cell that dials past it cannot cover the states the fake holds. `llmtest.Reply{Status: 401}` would have.

*Fix sketch, two honest options — pick one and say which:* cheapest is to include the underlying error the way `llmcheck` does (`define: the model did not answer (%v); …`), which restores the information without touching `internal/llm`'s contract. The principled fix is a credential sentinel in `internal/llm` — but that is `#11`'s taxonomy and `tools#19` already owns exactly this misdiagnosis surface, so deferring it there is defensible *provided* the deferral is recorded rather than left implicit. Either way, re-drive the cell through `llmtest.Reply{Status: …}` so the enumeration `{no key, refused, 5xx, 401/403}` is the one the test covers.

**Second instance of the same rule, in the comment the last round's finding named.** `askrun_test.go:828-831` still reads *"omitting restore leaves the whole suite GREEN … omitting qcancel leaves the suite green AND go vet silent"*. Both are now false — I measured each reddening a named subtest — and they were falsified by the **same commit** that edited the adjacent clause on line 828 (`"reddens five tests"` → `"reddens the scoped-during-the-answer cell"`). The ragged wrap left at `828-829` is the signature: a line-level patch where the finding asked for a sweep of the paragraph that owns the claim (BR-28/BR-44's rule, fourth appearance).

## 4. Minor findings

- **This is the 13th finding in family `doc-overstates-code`** — `openerStem` (`cmd/define/question.go:71-80`) claims contractions are matched on the stem "so the set stays the vocabulary rather than its inflections". Measured, that holds for regular negatives and fails for all three irregular ones: `can't`→`"ca"`, `won't`→`"wo"`, `shan't`→`"sha"`, while `can`, `will` and `shall` are all in `questionOpeners`. So `can't you use it in a sentence` and `won't that sound rude` both answer "no dictionary entry". `question_test.go`'s negation row uses `isn't` — a regular form — so the table's enumeration covers the representative, not the class. Same rule as I1; low harm (`?` is the recovery), worth one row and three stems.
- The atlas paragraph that owns the recording rule (*"The log records the question, not the answer"*, `atlas/define.md:585`) does not state the recorded/not-recorded boundary that README:99-102 now keys on ("whether a request was actually sent"), and mentions neither degradation message. An omission rather than a contradiction, but it is the paragraph that owns the fact.
- README:164-166's exit-1 list (`no dictionary entry, a question with no model configured, --forget…, --llm-check…`) does not include the two exit-1 causes this window added: a configured model that did not answer, and `ErrRequest`/`ErrMalformed` from `runAsk`'s `default:` arm.
- The five PTY conformance rows all **SKIP** here (`no pty available: operation not permitted`), so `TestPTYCtrlCMidAnswerKeepsTheSession`'s claim still rests on the implementor's note. `builtBinary` does run, so staleness is impossible by construction — that half is structurally sound.

## 5. Test coverage notes

All mutations run in scratch `git archive` trees under `$TMPDIR`; the working tree was never modified and is clean. Scratch trees are not git repos, so `TestNoTrackedRuntimeState` / `TestNoRuntimeStateInHistory` / `TestNoBinariesInHistory` fail there as artifacts — those are excluded from the readings below, and no other test failed in the green rows.

| mutation | result |
|---|---|
| `askScoped`: omit `interrupts.Set` | RED — `…CleansUp/the_sink_is_scoped_to_the_question_WHILE_it_runs` |
| `askScoped`: omit `defer restore()` | RED — `…/the_sink_is_handed_back…` **and** `TestCtrlCQuitsAgainOnceTheAnswerIsOver` |
| `askScoped`: omit `defer qcancel()` | RED — `…/the_question's_context_is_cancelled_when_it_returns` |
| `TestCtrlCQuitsAgainOnceTheAnswerIsOver` `-count=30` | **30/30 PASS** (was 12 failures in 30) |
| `replraw.go:161` `&sess` → `&session{}` | RED — `TestTheAskWiringTable`, `TestAQuestionIsNotCaptured` |
| `main.go:366`+`:393` `stdout` → `io.Discard` | RED — `TestTheAskWiringTable` |
| `crlf.go:41` revert `carriedCR` → `lastWasCR` | **GREEN** — BR-53 |
| `replraw.go:314` `recallLine()` → `line` | **GREEN** — BR-12(a), 4th round |
| `capture.go:102` delete `CaptureAsk`'s `decideCapture` guard | **GREEN** — BR-50(2) |

`go test ./...` ok (`cmd/define` 60.6s); `go test -race ./cmd/define/...` ok (61.8s); `go vet ./...` and `go vet -tags conformance ./cmd/define/` clean. Built-binary probe in a temp directory: `?`→2, `\`→2 (both one-shot and piped, correct single-backslash text), `-raw '?why'`→2, unforced question→1 with `is not a word`, forced question→1 with `cannot answer`; event log held exactly the two lookups.

## 6. Architectural notes

- **ARCH-DRY — flag, unchanged from round 9.** The consolidations are real and load-bearing: `askScoped` (7 cells across both loops), `fail(code)`, `nothingSays` (one production occurrence tree-wide), `lostTerminal` (one helper, two call sites), `recallLine`, `syncBuf`, `writeBytesAtomic`, `capture.go` as the sole non-test `AppendEvent` caller. The flag is BR-53's stated rule, still unapplied: `crlfWriter.Write` materialises the translation into `out`, `carriedCR` reads `out`, and `consumed` re-derives the same translation from `p` — two derivations of one fact, which is precisely how they disagreed in the first place.
- **ARCH-PURE — pass.** Every entity the plan calls PURE is deterministic and its test runs without a store, socket, clock or terminal. `gatherAskContext` remains a thin read whose only judgement is the nil-deck check; `askScoped` is control flow with no policy in it.
- **ARCH-PURPOSE — flag on the class/instance axis, twice.** The shadow-sweep over "the directory is the context" passes: one-shot, piped and raw editor all derive from the store and session, all three pinned by one table, and I verified the one-shot end to end against the binary. The flags are that two class-fixes again stopped at the instance — the entity enumeration was re-run and again missed what the same commit created (I1's sibling, and BR-47), and the `unreachable` split named the two halves the author had in mind rather than the three the producer emits.
- **ARCH-MOCK — flag (new, small).** `llmtest.Fake` is a wire-level httptest server, `askRig` uses a real YAML store in a temp dir, `deps.notifySignals` seams `signal.Notify`, `storetest.Suite` runs both implementations with a setter so the fake holds the real one's state, and the pty suite builds what it tests. The flag is the one cell that reaches *past* the fake to `http://127.0.0.1:1` — which is exactly why I1's 401/403 gap was never driven. Production flow and test flow must share the boundary; here they briefly do not.
- **For `#17` and `#10`:** `askContext` is a stable surface now that every selection policy in it is pure. Two things to hand over deliberately — `repl`'s `context.WithoutCancel` (`repl.go:180`) means the loop honours no caller cancellation, which is correct while `main` is the only caller and a trap for the next; and `gatherAskContext` reads the whole deck per question (`ask.go:238`), negligible beside a round-trip today and not once `#10` reuses the struct in a loop.

## 7. Plan revision recommendations

Three `## Revisions` entries are owed, and the first is the one that matters:

1. **The enumeration, actually run last (BR-47, 4th occurrence).** The existing entry at `workshop/plans/000016-console-qa-plan.md:1396` states the rule and reports "34 additions". Measured: 34 is the count at `993fccc..eb624f3` — the *previous* commit. At the close boundary (`ba01621..3ae9835`) it is 50; `eb624f3..3ae9835` adds exactly `unreachable` and `sayUnavailable`, and neither appears anywhere in the tables. Add them, and add the rows the enumeration lists that no round has ever reconciled: `ask`, `mayAsk`, `unavailable`, `question` (the type), `lookupOutcome`, `recallLine`, `nothingSays`. Then correct the rule the entry encodes, because "run it last" has now failed as stated: **the check has to run against the staged tree of the commit that will carry it, and its output pasted into that same commit** — a checklist item placed "last" still precedes the edits the fix itself makes.
2. **Boundary round 2's forks and round 6's outstanding item (BR-48).** Six `###` entries exist and none records: `TestThePipedLoopsAskWiring` folded into `TestTheAskWiringTable`; `replLines` gaining the interrupt scope the plan gave only to the raw loop; the pty suite building its own binary instead of using `bin/define`. Grep confirms all three strings appear zero times in the plan. Round 6's explicit item is also still absent — **"SIGINT through `repl`'s watcher *during a scoped stream*" remains unasserted**; `TestBothInterruptTransportsReachTheSink` covers the unscoped session only.
3. **What M1 actually shipped (BR-17, unchanged since round 3).** Task 3's snippet at line 632 still reads `if !literal && readsAsQuestion(word)` where the code is `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; `askUnavailable(stderr, question)` is still named at lines 687 and 746 where the code has `unavailable`/`unreachable`/`sayUnavailable`; and `mayAsk`, `recallLine`, `nothingSays` and `lostTerminal` appear zero times.

Bookkeeping otherwise checks out: 60/60 plan checkboxes ticked, every issue Done-when row ticked and traceable to a named test, and `workshop/projects/define-learn.md` carries per-milestone rows with `actual:`/`closed:` for M1 and M2.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Fourth round unchanged - submitLine's hist.Add(cmd.recallLine()) -> hist.Add(line) leaves the suite green; only the pure recallLine is pinned.
  - id: BR-17
    disposition: not-addressed
    note: |
      Unchanged verbatim - Task 3's snippet at line 632, askUnavailable at 687/746, and zero mentions of mayAsk, recallLine, nothingSays or lostTerminal.
  - id: BR-47
    disposition: not-addressed
    note: |
      Fourth round, same failure mode - the entry says the enumeration "runs LAST" and reports 34, which is the count at 993fccc..eb624f3; the close commit then added unreachable and sayUnavailable, neither in the tables.
  - id: BR-48
    disposition: not-addressed
    note: |
      Six Revisions entries, still none for round 2's four forks or round 6's Task 11 item; TestThePipedLoopsAskWiring, "builds its own binary" and "scoped stream" appear zero times in the plan.
  - id: BR-49
    disposition: not-addressed
    note: |
      Unchanged - replRaw (replraw.go:29) still passes the seam straight to readKeys with no policy, two nil guards elsewhere, and 47 test call sites now pass nil.
  - id: BR-50
    disposition: not-addressed
    note: |
      All four residues present verbatim; residue (1) is now doubly wrong - one lostTerminal helper with two call sites, and M2's streaming created no fifth. Residue (2) mutation-verified green.
  - id: BR-53
    disposition: not-addressed
    note: |
      The site fix stands but carriedCR reverts green (measured), and the rule is unapplied - Write materialises the translation into out while consumed re-derives it from p.
  - id: BR-54
    disposition: addressed
    note: |
      Measured 30 of 30 clean on unmutated HEAD (was 12 failures in 30), and it still reddens under omit defer restore(); the wait is now on the post-askScoped prompt redraw.
  - id: BR-55
    disposition: addressed
    note: |
      All four claims corrected and the qcancel cell genuinely defended - each of askScoped's three observable mutations reddens the subtest its row now names. The new unavailable/unreachable split misroutes 401/403; raised separately.
findings:
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      unreachable splits ErrUnavailable into two halves; it has three producers and the credential one gets the wrong half
    detail: |
      This is the 12th finding in this family; the rule has been stated six
      times, so do NOT patch the one message. Rule, in the shape this window
      needs it - a taxonomy branch must be exhaustive over what PRODUCES the
      error it dispatches on, and that enumeration is greppable in the producer
      rather than guessable in the consumer. ask.go:115's own comment states the
      contract ("one is something to set up, the other something to wait out"),
      while internal/llm's classifyStatus emits ErrUnavailable from three places
      and says of the third that "a missing or wrong credential is an operator
      configuration state, not a crash". Measured through llmtest.Fake at
      statuses 401, 403, 503 and 429: all four print
      `define: the model did not answer; ...` and exit 1, so an expired or wrong
      DEFINE_LLM_API_KEY - the one ErrUnavailable cell a user can fix - is told
      to wait. llmcheck.go:69 gets this right by not deciding, printing the
      underlying error verbatim; the ask path now gives strictly less
      information than the diagnostic beside it and points the wrong way. The
      gap survived because the cell added for this split
      (askrun_test.go:619-645) reaches past the wire fake to
      envFor("http://127.0.0.1:1"), exercising only the status==0 arm - ARCH-MOCK:
      a cell that dials past the fake cannot cover the states the fake models.
      Second instance of the same rule, in the comment the previous finding
      named: askrun_test.go:828-831 still says "omitting restore leaves the whole
      suite GREEN" and "omitting qcancel leaves the suite green AND go vet
      silent", both measured false, both falsified by the SAME commit that edited
      the adjacent clause on line 828 - a line-level patch where the finding
      asked for a sweep of the paragraph that owns the claim.
  - id: new
    severity: Minor
    family: doc-overstates-code
    title: |
      openerStem cannot reach can't, won't or shan't, while can, will and shall are all openers
    detail: |
      This is the 13th finding in this family, so it is recorded as an instance
      of the rule above rather than fixed at the site. question.go:71-80 claims
      negated auxiliaries are matched "on the stem before n't ... so the set
      stays the vocabulary rather than its inflections". Measured: isn't->is,
      don't->do, didn't->did and couldn't->could all reach questionOpeners, but
      the three irregular English negatives do not - can't->"ca", won't->"wo",
      shan't->"sha" - so `can't you use it in a sentence` and `won't that sound
      rude` both answer "no dictionary entry". question_test.go's negation row
      uses isn't, a regular form, so the table covers the representative rather
      than the class. Low harm, since "?" is the documented recovery; worth one
      row and three stems when the arm is next touched.
```

---

## Re-review — 2026-08-24T21:20:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | whole-issue close |
| milestone | — |
| window | ba0162169f48d72983ff7f21d3a6479c9e6535a3..e6dec7cda5ad120833035ab1f3d625df1536cb77 |
| command | sdlc close --issue 16 |
| reviewer | claude |
| timestamp | 2026-08-24T21:20:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All measurements complete. Race-clean, vet-clean, suite green.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Eleven rounds in, and this round's two claimed fixes are both real — I confirmed them by reversion, not by reading. BR-56's split is genuinely retired: `unavailableAfterSending` now hands over the underlying error, and I measured all four `ErrUnavailable` producers through the wire fake (`401 Unauthorized`, `403 Forbidden`, `429`, `503` each reach stderr verbatim), with a fixed-message mutation reddening `TestAQuestionIsRecordedWhateverBecameOfTheAnswer`. BR-54's flake is gone: `TestCtrlCQuitsAgainOnceTheAnswerIsOver` is **30/30** clean (was 12 failures in 30), and each of `askScoped`'s three observable cells reddens exactly the subtest its doc comment names it by — `omit interrupts.Set` → `/the_sink_is_scoped_to_the_question_WHILE_it_runs` (plus 8 more cells), `omit defer restore()` → `/the_sink_is_handed_back_when_it_returns` **and** the end-to-end row, `omit defer qcancel()` → `/the_question's_context_is_cancelled_when_it_returns`. `go test ./...`, `go test -race ./cmd/define/...`, `go vet ./...` and `go vet -tags conformance` are all clean. What keeps this off SHIP is that the closing commit repeated, in the same commit, the two rules it wrote down: the atlas still says the ask path has **"four exit-code absolutes"** twenty lines from the section that commit edited, while this window's own message split made that count wrong — and the plan's entity enumeration, whose new Revisions entry declares it "runs against the working tree", still omits `ask`, `sayUnavailable` and `lastN`, two of them from the file the closing commit rewrote. Three prior findings (BR-12, BR-50(2), BR-53) remain behaviourally correct and mutation-green for the 5th, 3rd and 3rd round.

## 1. Strengths

- **BR-56's fix is the right shape and it is pinned.** `cmd/define/ask.go:122` reports the taxonomy's word and hands over the cause, the conclusion `llmcheck.go` reaches by not deciding. Measured through `llmtest.Fake`: a wrong key now prints `the model is unavailable (llm: unavailable: POST …: 401 Unauthorized …)` instead of being told to wait. Replacing it with a fixed sentence reddens the test.
- **The `askScoped` enumeration is now honest, and checkable.** Swapping counts for test *names* was the correct correction — I opened each name and ran each mutation, and all three land on the cell they claim. `omit interrupts.Set` reddens 5 top-level tests spanning both loop shells.
- **BR-54's fix removed the race rather than widening a timeout.** Waiting for `prompt` to reappear *after* `"insincerely"` (`askrun_test.go:860`) synchronises on an effect `askInSession` produces strictly after `askScoped` returns. Each run now costs ~0.01s instead of being timing-dependent.
- **The wire-fake discipline held.** The new rows drive `llmtest.Reply{Status: …}` through the fake rather than dialling `127.0.0.1:1`, and the never-reached producer keeps its own row — so all three `ErrUnavailable` producers are covered at the seam production uses (ARCH-MOCK).
- **The route/capture ordering still holds end to end.** `recentDeck`, `renderAskPrompt`, `recentTurns`, `consumed`, `parseREPLLine` and `session` are all pure and tested without a store, socket or terminal; `TestTheAskWiringTable` still reddens on `&sess → &session{}` and `stdout → io.Discard`.

## 2. Critical findings

None.

## 3. Important findings

**I1 — the atlas states a COUNT for the ask path's exit codes, and this window made it wrong** (`atlas/define.md:622`, `README.md:164`).

**This is the 14th finding in family `doc-overstates-code`.** The rule has been stated seven times. Do NOT patch the two lines.

`atlas/define.md:622` reads *"The **four** exit-code absolutes README states are measured across `{one-shot, piped}`, which is the enumeration they quantify over."* Measured, the ask path alone now has three `return 1` sites — `ask.go:129`, `ask.go:132` (`sayUnavailable`'s two route branches) and `ask.go:209` (`runAsk`'s `default:` arm for `ErrRequest`/`ErrMalformed`) — and README's exit-`1` clause (`no dictionary entry, a question with no model configured, --forget…, --llm-check…`) names one of them. Probed against the fake: a 400 exits **1** with `define: llm: bad request: … 400 Bad Request …`; a 401 exits **1** with `the model is unavailable (…)`. Neither appears in either document, and `atlas/define.md` has **zero** occurrences of `no model configured`, `unavailable` or `did not answer` anywhere — while it devotes a paragraph to *which* of the degradation sentences each route selects.

The sharp part is not the gap; it is where the gap came from. The closing commit's own plan Revisions entry states the rule it broke:

> **The entry must not carry a count.** A count is a measured claim that drifts the moment anything is added, and it drifts silently. Same rule BR-56 states for taxonomy messages…

That entry was written in the same commit that created `unavailableAfterSending` and `sayUnavailable`, twenty lines above the atlas paragraph that carries the count, in a section the previous commit had just edited.

**The rule this window's evidence actually supports** — and it spans this family and `plan-contract-drift` both: two families here have now been answered with *"here is a greppable command to run at the boundary"* (BR-44's normative-block sweep, BR-47's `git diff | grep '^+\(func\|type\)'`), and **both recurred in the very commit that wrote the command down** — BR-47 five times. The command was never the problem; reconciling its output *by eye, against prose, inside a commit still being edited* is. This repo already knows the working shape: `builtBinary` retired the pty-staleness class by making staleness impossible rather than checkable, and `TestTheCapturerInjectionIsLive` is a control that goes red. A checklist item that must be remembered at the exact moment attention is lowest is not a fix. If a claim is worth writing down, make something fail when it stops being true — or write it without a count and without an enumeration it does not own.

*Cheap version if the sweep is deferred:* drop `four` from `atlas/define.md:622`, and add the two ask-path causes to README's exit-`1` clause. Note that block was already partial before this window (`lostTerminal`, the `reading input` failure), so the deliverable is the sweep, not the clause.

## 4. Minor findings

- **This is the 9th finding in family `test-asserts-nothing`.** `askrun_test.go:658`'s `configured but never reached` row is inside `TestAQuestionIsRecordedWhateverBecameOfTheAnswer` — a test whose *name* is the claim — and it is the only one of its nine rows that never reads `st.Events`. Its eight siblings all assert `len(ev) != 1`. Behaviour is correct (probed: `code=1`, `events=1`), so this is a covered cell that isn't checked. The rule, and it is new: **a row added to a table whose name is an absolute inherits that table's assertions; a row asserting less than its siblings is a cell that is present in the enumeration but absent from the coverage.** Diffing a new row against its siblings' assertion set is mechanical and would have caught it.
- `sayUnavailable` now embeds a full HTTP error — URL plus JSON body, ~200 chars — into a message whose *other* half is bounded to 40 runes by `truncateQuestion`, and it renders at an interactive prompt through `crlfWriter`. `--llm-check` prints verbatim because a user ran a diagnostic on purpose; this fires mid-session. The trade is deliberate and documented, but the two halves of one sentence now disagree about brevity.
- `Store.SetUserModel` still has zero production call sites — its only caller is `storetest/suite.go`. Legitimate (it is BR-45's fix), unrecorded as test-only surface.
- All five PTY conformance rows **SKIP** here (`no pty available: operation not permitted`), so `TestPTYCtrlCMidAnswerKeepsTheSession`'s claim still rests on the implementor's note. `builtBinary` does compile from the tree, so staleness is impossible by construction.

## 5. Test coverage notes

Every mutation ran in a scratch `git archive` of `e6dec7c` under `$TMPDIR`; the working tree was never modified and is clean. Scratch baseline: `ok 62.3s`.

| mutation | result |
|---|---|
| `askScoped`: omit `interrupts.Set` | RED — `…/the_sink_is_scoped_to_the_question_WHILE_it_runs` + 4 tests / 8 subtests |
| `askScoped`: omit `defer restore()` | RED — `…/the_sink_is_handed_back_when_it_returns`, `TestCtrlCQuitsAgainOnceTheAnswerIsOver` |
| `askScoped`: omit `defer qcancel()` | RED — `…/the_question's_context_is_cancelled_when_it_returns` |
| `unavailableAfterSending` → fixed sentence | RED — `TestAQuestionIsRecordedWhateverBecameOfTheAnswer` |
| `TestCtrlCQuitsAgainOnceTheAnswerIsOver` `-count=30` | **30/30 PASS** (was 12 failures in 30) |
| `submitLine`: `recallLine()` → `line` | **GREEN** — BR-12, 5th round |
| `crlf.go:41`: `carriedCR(...)` → `lastWasCR` | **GREEN** — BR-53, 3rd round |
| delete `CaptureAsk`'s `decideCapture` guard | **GREEN** — BR-50(2), 3rd round |

`go test ./...` ok; `go test -race ./cmd/define/...` ok (64.2s); `go vet ./...` and `go vet -tags conformance ./cmd/define/` silent. Probed behaviour: `401/403/429/503` → exit 1, event recorded, cause carried; connection-refused → exit 1, event recorded, `connection refused` carried; `400` → exit 1, `llm: bad request`; unwired seam → exit 1, `no model configured`, **no** event. `openerStem`: `isn't→is`, `don't→do`, `didn't→did`, `couldn't→could`, but `can't→"ca"`, `won't→"wo"`, `shan't→"sha"` (BR-57 unchanged). `recallLine`: `\how so`→`\how so`, `?why`→`?why` (BR-12 behaviour correct, wiring unpinned).

## 6. Architectural notes

- **ARCH-DRY — pass, one standing flag.** `askScoped` is one owner reachable from both loops and I re-proved it (7 cells); `sayUnavailable` correctly collapses the two degradation messages onto one route-aware sentence rather than duplicating the `q.forced` branch — the right instinct, applied the round after the family that produced BR-46. The flag is unchanged: `crlfWriter.Write` materialises the translation into `out` while `consumed` re-derives it from `p` — two derivations of one fact, which is how they disagreed originally (BR-53's stated rule, still unapplied).
- **ARCH-PURE — pass.** Every entity the plan calls PURE is deterministic and its test runs with no store, socket, clock or terminal. `askScoped` is control flow with no policy; `gatherAskContext` is a thin read whose only judgement is the nil-deck check. The three new `TestAskScopedHandsTheScopeBackAndCleansUp` subtests are the principle paying off — three timing-free unit assertions replacing what two rounds tried to measure through a stream.
- **ARCH-PURPOSE — flag on the class/instance axis, twice.** The shadow-sweep over "the directory is the context" passes: one-shot, piped and editor all derive from the store and session, all three pinned by one table. The flags are that two class-fixes again landed as instances — the exit-code/message enumeration (I1) and the entity enumeration (BR-47, fifth round) — and in both cases the commit that stated the rule is the commit that broke it.
- **ARCH-MOCK — pass, and this round closed the last gap.** The `ErrUnavailable` rows now drive `llmtest.Fake` at the statuses the fake models rather than dialling past it, so production flow and test flow share the boundary for all three producers. `askRig` uses a real YAML store in a temp dir, `storetest.Suite` runs both implementations with a setter, `deps.notifySignals` seams `signal.Notify`, and the pty suite builds what it tests. The only remaining gap is environmental (no pty here), not structural.
- **For `#17` and `#10`:** `askContext` is stable now that every selection policy in it is pure. Hand over deliberately: `repl`'s `context.WithoutCancel` (`repl.go:180`) means the loop honours no caller cancellation — correct while `main` is the only caller, a trap for the next; and `gatherAskContext` reads the whole deck per question (`ask.go:244`), negligible beside a round-trip today and not once `#10` loops on it.

## 7. Plan revision recommendations

Three `## Revisions` entries are owed in `workshop/plans/000016-console-qa-plan.md`:

1. **The entity enumeration, fifth attempt — and a different kind of fix (BR-47).** I ran the plan's own command against the boundary: 35 entities, and `ask`, `sayUnavailable` and `lastN` appear nowhere in the file (`grep -c` returns 0 for each). Two of the three are in `ask.go`, the file the closing commit rewrote, and `sayUnavailable` was created one commit before the pass that added its two siblings' rows. `ask` — *"the ONE entry into the question path, from all six cells"*, the function the issue is about — has been missing since M1. The entry should record that the timing correction ("run against the working tree") was applied and the enumeration still missed three, so the remaining variable is the by-eye reconciliation, not the moment: either drop the tables to a generated list, or accept that they are illustrative and stop claiming they are the enumeration.
2. **Boundary round 2's forks and round 6's outstanding item (BR-48).** Six entries exist; `TestThePipedLoopsAskWiring`, `builds its own binary` and `scoped stream` all appear **zero** times. Still unrecorded: the piped-wiring test folded into `TestTheAskWiringTable`, `replLines` gaining the interrupt scope the plan gave only to the raw loop, the pty suite building its own binary — and round 6's item, that **"SIGINT through `repl`'s watcher *during a scoped stream*" remains unasserted** (`TestBothInterruptTransportsReachTheSink` covers the unscoped session only).
3. **What M1 actually shipped (BR-17, unchanged since round 3).** Task 3's snippet at line 637 still reads `if !literal && readsAsQuestion(word)` where the code is `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; `askUnavailable(stderr, question)` is still named at lines 691 and 750 where the code has `unavailable`/`unavailableAfterSending`/`sayUnavailable`; `-raw` and `lostTerminal` appear **zero** times. The Core-concepts tables were swept for `mayAsk`/`recallLine`/`nothingSays`/`lookupOutcome`/`question`; the task prose was not.

Bookkeeping otherwise checks out: 60/60 plan checkboxes ticked, every issue Done-when row ticked and traceable to a named test, and `workshop/projects/define-learn.md` carries per-milestone rows with `actual:`/`closed:` for M1 and M2.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Fifth round unchanged - behaviour correct (recallLine round-trips all four kinds, measured), but submitLine's hist.Add(cmd.recallLine()) -> hist.Add(line) leaves the full suite green.
  - id: BR-17
    disposition: not-addressed
    note: |
      Task 3's snippet at line 637 and askUnavailable at 691/750 are verbatim unchanged; "-raw" and "lostTerminal" appear zero times. Only the Core concepts half was swept.
  - id: BR-47
    disposition: not-addressed
    note: |
      Fifth round. The timing fix was applied and the enumeration still misses ask, sayUnavailable and lastN - two of them in the file the closing commit rewrote, and ask since M1.
  - id: BR-48
    disposition: not-addressed
    note: |
      Six Revisions entries, still none for round 2's forks or round 6's Task 11 item; TestThePipedLoopsAskWiring, "builds its own binary" and "scoped stream" all appear zero times.
  - id: BR-49
    disposition: not-addressed
    note: |
      Unchanged - replRaw still passes the seam straight to readKeys with no policy, two nil guards elsewhere, and 47 test call sites still pass nil.
  - id: BR-50
    disposition: not-addressed
    note: |
      All four residues verbatim; residue (1) still predicts a fifth lostTerminal site that has two, and residue (2) is mutation-verified green again this round.
  - id: BR-53
    disposition: not-addressed
    note: |
      The site fix stands but carriedCR still reverts green (measured), and the rule is unapplied - Write materialises the translation into out while consumed re-derives it from p.
  - id: BR-56
    disposition: addressed
    note: |
      Measured through the fake at 401/403/429/503 - all four now carry the underlying cause and none says "did not answer"; a fixed-message revert reddens the test.
  - id: BR-57
    disposition: not-addressed
    note: |
      Unchanged and re-measured - can't->"ca", won't->"wo", shan't->"sha" all miss questionOpeners while isn't/don't/didn't/couldn't reach it.
findings:
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      atlas states a COUNT for the ask path's exit codes, and this window's own message split made it wrong
    detail: |
      This is the 14th finding in this family; the rule has been stated seven
      times, so do NOT patch the two lines. atlas/define.md:622 says "The FOUR
      exit-code absolutes README states are measured across {one-shot, piped}".
      Measured: the ask path alone has three return-1 sites - ask.go:129 and
      ask.go:132 (sayUnavailable's two route branches) and ask.go:209 (runAsk's
      default arm) - and README:164's exit-1 clause names one of them. Probed
      through the fake: a 400 exits 1 with `define: llm: bad request: ...`, a 401
      exits 1 with `the model is unavailable (...)`; neither appears in either
      document, and atlas/define.md has ZERO occurrences of "no model
      configured", "unavailable" or "did not answer" while devoting a paragraph
      to which degradation sentence each route selects. The count is the sharp
      part: the SAME commit that created unavailableAfterSending and
      sayUnavailable wrote a plan Revisions entry stating "the entry must not
      carry a count - a count is a measured claim that drifts the moment
      anything is added, and it drifts silently", twenty lines above the atlas
      paragraph carrying one. The rule the evidence supports, spanning this
      family and plan-contract-drift both: two families here have been answered
      with "here is a greppable command to run at the boundary" (BR-44's
      normative-block sweep, BR-47's entity enumeration) and BOTH recurred in
      the commit that wrote the command down - BR-47 five times. The command was
      never the problem; reconciling its output by eye, against prose, inside a
      commit still being edited is. The repo already knows the working shape -
      builtBinary made pty staleness impossible rather than checkable, and
      TestTheCapturerInjectionIsLive is a control that goes red. If a claim is
      worth writing down, make something FAIL when it stops being true, or write
      it without a count and without an enumeration it does not own. Note the
      README block was already partial before this window (lostTerminal, the
      reading-input failure), so the deliverable is the sweep, not the clause.
  - id: new
    severity: Minor
    family: test-asserts-nothing
    title: |
      The row added this round to the recording table is the only one of nine that never reads the event log
    detail: |
      This is the 9th finding in this family. askrun_test.go:658's "configured
      but never reached" row sits inside
      TestAQuestionIsRecordedWhateverBecameOfTheAnswer - a test whose NAME is the
      claim - and asserts only that stderr carries the connection failure. Its
      eight siblings (answered, refused-by-the-service, cut-off, and the four
      status rows added by the same commit) all assert len(ev) != 1. Behaviour is
      correct: I probed it and the event IS recorded (code=1, events=1), so this
      is a cell present in the enumeration and absent from the coverage. The
      rule, which is new for this family: a row added to a table whose name is an
      absolute INHERITS that table's assertions, and a row asserting less than
      its siblings is exactly the shape a coverage gap hides in - diffing a new
      row's assertion set against its siblings' is mechanical and would have
      caught it. Adjacent and cheap: the same row is the one README's recording
      sentence is loosest about, since a connection-refused sends no request yet
      records.
```
