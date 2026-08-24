# Boundary Review — tools#16 (milestone M2)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 993fccc020f3a003c792154917585f998ecbbe7c..686913cc5543b1b9c8e9b721382836a397cfdd50 |
| command | sdlc milestone-close --issue 16 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-23T23:03:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 lands the shape the plan designed and lands it well: the prompt is a pure renderer over a data struct with a golden that snapshots the real `llm.Request`, the interrupt has one sink both transports feed, the swallow lives in the reader and is genuinely pinned (I reverted it — `TestEditorCtrlCMidStreamReturnsToThePrompt` goes red), and the cancel-guard-asks-the-context decision is pinned too. What blocks SHIP is two verified correctness bugs in the milestone's headline feature — the "recent deck" section sends the **oldest** twelve words, and the one-shot unforced route writes the question text into `session.current`, which both mislabels the prompt and lands a non-word in the `word` field of the append-only event log `#17` will fold over — plus `go test -race ./cmd/define/` failing on two new tests where the base commit is race-clean. Underneath those sits the repeat pattern this issue has been fighting for five rounds: I measured four separate claims in this diff that survive the mutation they exist to catch, and the two Criticals are both in cells no test drives.

## 1. Strengths

- `cmd/define/askctx.go:78-119` — `renderAskPrompt` returning an `llm.Request` (not two strings, as the plan sketched) is the right call: `llmtest.AssertGolden` renders through `llm.RenderRequest`, the same function `RequestHash` uses, so the golden and the cassette key cannot drift apart. Confirmed at `internal/llm/llmtest/golden.go:29-33`.
- `cmd/define/rawterm.go:70-78` — the swallow. Verified by mutation: `if interrupts.Fire() && false` reddens `TestEditorCtrlCMidStreamReturnsToThePrompt/the_byte_transport` with a real `readKeys` over a real pipe. This is the one wiring the milestone is named after and it is honestly pinned.
- `cmd/define/ask.go:113-117` — the `parent.Err()`-first guard. Deleting it prints `define: the answer was cut off` at the user's own Ctrl-C; the test catches it.
- `cmd/define/store/event.go:46-51` + `yaml_test.go:151-181` — the `at:`-stays-last rule is asserted against the bytes on disk, not just described. The three new torn-record rows cut the `asked` shape at each new field.
- `summariseLookups` (`history_cmd.go:108`) and `storeHistory.Load` (`history_store.go:52`) both filter on `EventLookedUp`, so the new kind cannot pollute `/history` or Up-arrow recall. Checked, not assumed.

## 2. Critical findings

**C1 — `cmd/define/ask.go:180-187`: the deck context sends the oldest words under a "Recently in the deck" header.**

`Store.Deck()` is documented and implemented as **newest-first** (`store.go:14`, `sortDeck` at `mem.go:62-69`). `lastN(words, 12)` takes the **tail**, which on a newest-first list is the twelve *oldest* — and it does not reverse, so the comment's two stated intentions ("the prompt reads better oldest-first, and the newest are the ones worth keeping") are both unmet.

Measured with a 15-word deck (`wa` oldest … `wo` newest):

```
## Recently in the deck
wl, wk, wj, wi, wh, wg, wf, we, wd, wc, wb, wa
```

The three most recent words are exactly the ones dropped. README:82 ("your recent deck") and `atlas/define.md:522` ("the recent deck") both state the opposite. Fix: `first := deck[:min(len(deck), maxContextWords)]`, then reverse — and move the selection into a pure helper so it is table-tested (see §6).

**C2 — `cmd/define/main.go:387`: the one-shot unforced ask puts the QUESTION into `session.current`.**

> **This is the 3rd finding in family `forced-route-enumeration`.** Earlier rounds fixed instances (`\how so` recalled as `how so`; `askUnavailable` claiming `why` is not a word). Do NOT fix only this site — state the rule and sweep the enumeration.

`&session{current: oneShot.word}` is handed the line the dictionary just missed, i.e. the question itself. Measured, with a live seam and a real YAML store:

```
PROMPT:
## The word on screen
what's the difference to obsequious?

## The question
what's the difference to obsequious?

EVENT: kind="asked" word="what's the difference to obsequious?" question="…"
```

Two consequences. The model is told a question is "the word on screen" with no entry beneath it — a claim nothing checked, in the same shape as BR-11. And `recordAsked` (`ask.go:150`) writes that text into `ReviewEvent.Word` in the append-only log, which is precisely what D2 exists to prevent and what Task 4's contract table forbids ("does **not** set `sess.current`"). The forced one-shot cell passes `&session{}` and is correct; the two routes diverge.

**The rule the family needs:** every fact the ask path carries is quantified over `{forced, unforced} × {one-shot, piped, editor}` — six cells — and each cell is asserted for *that fact*, not for the path existing. The facts M2 added are: which session the ask carries, where the answer is written, and whether the interrupt is scoped. Measured coverage today is 2/18 (the editor's answer destination and interrupt scope). Write that table and assert it; C2 and I6 both fall out of the same sweep.

## 3. Important findings

**I1 — `cmd/define/ask.go:172-174`: `UserModel`'s error is silently discarded at its only call site.**

`store/yaml.go:44-48` states the reason it returns one: *"an UNREADABLE file is [an error], because silently answering `""` for a model that exists would make every answer pitched at the wrong level with no way to tell."* `gatherAskContext` then writes `if model, err := d.deck.UserModel(); err == nil { c.UserModel = model }` — the exact outcome the store's comment says must not happen. Either warn on stderr (the `storeCapturer`/`storeHistory` precedent) or delete the justification.

**I2 — `go test -race ./cmd/define/` fails; the base commit is race-clean.**

Verified in a worktree at `993fccc`: `ok`. At `686913c`: two failures, eleven race reports.
- `askrun_test.go:147-151` — `for out.Len() == 0 {}` reads a `bytes.Buffer` from a goroutine while `runAsk` writes it (also a bare spin loop with no yield).
- `askrun_test.go:267,280` — `waitFor` reads `out.String()` while `runEditor` writes it.

This package already owns the right shape: `ptyOut` (`pty_conformance_test.go:88-124`) is a mutex-guarded buffer with a snapshot accessor. The new tests hand-rolled an unsynchronised version instead (ARCH-DRY). Extract one `syncBuf` and use it in all three.

**I3 — the plan's `askCapturer` integration point was not delivered, and the departure is undeclared.**

Core concepts → Integration points lists `askCapturer` on `Capturer`, in `cmd/define/capture.go`, wrapping the event append. The code appends directly through `d.deck.AppendEvent` from `ask.go:145-152`, creating a second write path into the event log beside `capture`. It happens to be safe — `DEFINE_NO_CAPTURE` leaves `sd.deck` nil so `recordAsked` no-ops — but `main.go:31-34` now lies (*"capture is the only thing that RECORDS lookups. deck below is the other way the store is mutated: `--forget` deletes through it"*). Either route it through the seam or add a `## Revisions` entry saying why not, and fix that comment.

**I4 — README does not say that questions are written to disk.** (`readme-surface-gate`, 2nd finding.)

> Earlier rounds fixed one instance. State the rule, don't just add a line.

README:95-105 is the block that documents what `define` persists to the working directory — it names `words/` and `events/` and says *"Every successful lookup … records the word"*. M2 makes the tool persist **the text of every question you ask** into `events/*.yaml`. A reader of that block cannot learn this. **The rule:** any new record type or field the tool writes into the working directory is named in that block in the same change; the enumeration is `{words/, events/, user-model.md} × {looked-up, reviewed, asked}` and README currently names 2 of 3 files and 0 of 3 kinds.

**I5 — `atlas/define.md:182-186` still states the cancellation model this diff disproved.** (`doc-overstates-code`, 5th finding.)

> Earlier rounds fixed instances (the `-raw` absolute, the exit-code absolute, the `nothingSays` uniqueness claim). Do NOT fix only this paragraph.

It reads: *"Ctrl-C arrives as byte `0x03`, not a signal, so `signal.NotifyContext` — which the one-shot and piped paths still rely on — never fires. The key reader owns cancellation instead, calling `cancel()`…"*. Three claims, all now false: the pty suite measured a `\x03` arriving as SIGINT (which is D5's whole premise); the piped path no longer relies on `NotifyContext` at all (`repl.go:180` detaches with `WithoutCancel`); and the reader calls `interrupts.Fire()`, not `cancel()`. The diff corrected the *code comment* at `rawterm.go:44-46` and added a new atlas section at :544 — but left the atlas's own statement of the old model standing.

**The rule:** when a diff corrects a claim in a code comment, the same fact's restatements elsewhere are consumers of that source and must be swept in the same change — grep the fact, don't append a new section beside the stale one (ARCH-PURPOSE's shadow-sweep). Measured prevalence here: 1 of 2 `NotifyContext` mentions in `atlas/` is now false; the surviving one (`:672`, the one-shot path) is still correct.

**I6 — the piped loop's ask wiring is entirely unpinned.** (`loop-shell-branch-untested`, 2nd finding.)

> State the rule, don't just add one test.

Two mutations, both leaving `go test ./cmd/define/` fully green:
- `repl.go:261`: `ask(ctx, d, opt, &sess, stdout, …)` → `io.Discard` — the answer goes nowhere.
- `repl.go:261`: `&sess` → `&session{}` — the piped loop loses multi-turn and session words.

`lessons.md`'s define #15 rule already covers this ("a wiring only a loop shell supplies must be pinned by a test that drives that loop shell"); M2 obeyed it for `runEditor` and skipped it for `replLines` and the one-shot. That skipped cell is where C2 lives. The rule and the enumeration are the same as C2's — fix once.

**I7 — four claims in this diff survive the mutation they exist to catch.** (`test-asserts-nothing`, 5th finding.)

> Earlier rounds fixed four instances. Do NOT fix these four — state the rule.

Measured, each individually:

| claim | mutation | suite |
|---|---|---|
| `rawterm.go:49-54` "the buffering is load-bearing… the reader would block and never decode the Ctrl-C behind it" | `make(chan Key, 256)` → `make(chan Key)` | **green** |
| `ask.go:174` session words reach the prompt (issue Done-when: *"recent words … visible in the recorded prompt"*) | `SessionWords: lastN(sess.words, …)` → `nil` | **green** |
| `session.go:44-46` an empty answer is not recorded as a turn | delete the guard | **green** |
| `askrun_test.go:293-296` "Both routes … so the interrupt scoping and the CRLF writer exist once" | n/a — the test passes `nil` for `interrupts`, so it can only pin the writer | — |

The SessionWords row is the BR-2 shape exactly: `TestAskStreamsAnAnswerWithTheDirectoryAsContext` asserts `"sycophantic"` is in the prompt, and that string is supplied by `CurrentWord`, so the assertion is satisfied by a different source than the one it names. **The rule:** an assertion that a value reached an output must use a value *only that source can supply* — a fixture shared with another source makes the assertion vacuous. Apply it to the three rows above and to `TestEditorCtrlCMidStreamReturnsToThePrompt/the_signal_transport`, which calls `interrupts.Fire()` directly and so never touches `d.notifySignals` despite its name.

**I8 — `assertNoBareNewline` is vacuous for every row of `TestRawLoopMessagePlacement`.** (`raw-mode-message-placement`, 5th finding.)

> Four rounds have now fixed placement instances. State the rule.

The helper does `s = strings.TrimSuffix(s, "\n")` (`askroute_test.go:395`) to excuse the loop's post-`finish()` newline. All three rows write exactly **one** line to stderr, so the excused newline is the only one under test. Measured:
- `replraw.go:141` → `&crlfWriter{w: stdout}, stderr` (stderr unwrapped during a raw-mode stream): **green**.
- `replraw.go:226` `"%sdefine: %s\r\n"` → `"…\n"` (the bare-`?` note): `TestRawLoopMessagePlacement` **green**; only its `eraseLine` check has teeth.

The test's own header says *"EVERY message this loop writes is written in RAW mode, so every one carries its own carriage returns"* — asserted vacuously for 3 of 3 rows. **The rule (this is BR-14's rule applied to placement):** a placement assertion asserts the POSITIVE observable — that the message's own terminator is `\r\n` — rather than the absence of a bare `\n`, because a helper that excuses a trailing newline makes absence unfalsifiable for any single-line message.

**I9 — plan bookkeeping.** (`plan-bookkeeping`, 3rd finding.)

> State the rule.

`workshop/plans/000016-console-qa-plan.md` Chunk 2: **31 unticked steps, 0 ticked**, while the issue's `## Plan` and the project row both mark M2 `[x]`. The plan also still carries three `## Revisions` entries, **none** for a boundary round — the same gap BR-17 raised in M1 and which was never disposed — and none for M2's three design departures (I3, and the two location moves in §4). **The rule:** the plan artifact's state is part of the milestone deliverable — the checkbox sweep and a `## Revisions` entry for every design fork taken differently from the plan happen in the milestone-close commit, not at issue close.

## 4. Minor findings

- `ask.go:100-102` — `parent := ctx` is a bare alias; unlike `llmcheck.go:45-47` there is no derived context here, so the comment describes a distinction that does not exist.
- Core concepts table vs code: `exchange` is in `askctx.go:9-12`, not `session.go`; `interrupter` is in `interrupt.go`, not `rawterm.go`. (Not filed Critical despite the checklist's default: both entities exist, are pure, and are tested — this is a stale table, not a missing deliverable. Plan revision in §7.)
- `askrun_test.go:334-347` — `keysFor` is defined and never called; its doc comment describes a use case that does not exist.
- `interrupt_test.go:98` — `bytesReader(s)` is `strings.NewReader(s)` under a new name. (`stdlib-reuse`, 3rd finding — the rule: before adding a helper, check whether the stdlib already names it; `contains`→`slices.Contains` was the same rule two rounds ago.)
- `DEFINE_NO_CAPTURE=1` now also suppresses *reading* `user-model.md` and the deck (`ask.go:167`, since `openStore` leaves `deck` nil). README documents it as write-suppression only; answers silently become un-adapted.
- `unavailable()` prints "no model configured" for a *configured but unreachable* model, since `ErrUnavailable` covers both. Plan-specified, so not drift — but it is the same misdiagnosis surface as the already-filed tools#19.
- `askInSession` (`replraw.go:130-155`) always returns `nil`; the `lostTerminal(err)` branch at both call sites is now unreachable.
- `crlf.go:28` returns `0, err` on a write error and reports `len(p)` on a short underlying write.
- `ask.go:114` prints a newline on cancel even when no delta ever arrived.

## 5. Test coverage notes

- **Pinned and verified by reversion:** the reader's swallow; the `parent.Err()` cancel guard; the deck and user-model reaching the prompt; the `asked` event round-trip and its torn shapes (through `storetest.Suite`, so both `Mem` and `YAML`); `at:`-last on disk; the shared `askInSession` closure's CRLF behaviour for both routes.
- **Unpinned (measured, all green under mutation):** session words in the prompt; key-channel buffering; `recordExchange`'s empty-answer guard; the stderr `crlfWriter`; all three raw-mode message placements; both halves of the piped loop's ask wiring; the entire one-shot ask path with a live seam (every one-shot test runs with no model configured, which is why C2 shipped).
- **Not exercisable here:** `TestPTYCtrlCMidAnswerKeepsTheSession` skips in this environment (`pty.Start`: operation not permitted). Its claim — that unscoping the interrupt reddens it through a real pty — rests on the implementor's note; I could not check it. `go vet -tags conformance ./cmd/define/` is clean.
- `store.Mem.userModel` is set at zero call sites, so `storetest.Suite`'s user-model row can only assert the empty case for either implementation (ARCH-MOCK: the fake cannot hold the state the real one has). Honest for #16, but #17 should add `SetUserModel` and a non-empty conformance row rather than inherit a field that does nothing.

## 6. Architectural notes

- **ARCH-DRY — flag.** Two instances. The unsynchronised test buffers (I2) duplicate `ptyOut`'s mutex-guarded one in the same package. `bytesReader` duplicates `strings.NewReader`. On the positive side, `askInSession` as the single closure both ask routes enter is real and pinned — that half of the principle held.
- **ARCH-PURE — flag, and it is the cause of C1.** `renderAskPrompt` and `recentTurns` are properly pure and table-tested. But the *deck's* ordering-and-truncation policy — a pure decision, identical in kind to `recentTurns` — was inlined into `gatherAskContext`, the IO gatherer, where the only way to test it is to stand up a store. That is why a 15-word deck was never exercised and why the bug is invisible. Move it next to `recentTurns` as `recentDeck([]Word) []string` and give it the same table `TestRecentTurnsBoundsTheTranscript` has. `#10`'s harvest is going to reuse this struct; the policy should be pure before it does.
- **ARCH-PURPOSE — flag.** The stated purpose is "the directory is the context, so a fresh process answers as well as a long-running one". The shadow-sweep over consumers: the raw editor derives correctly; the piped loop derives correctly but is untested; the **one-shot derives wrongly** (C2) — and the one-shot is precisely the "fresh process" the purpose names. Also: the diff corrected `rawterm.go`'s comment about the interrupt model but left `atlas/define.md`'s restatement of the old model standing (I5) — a hand-maintained restatement that no longer derives from the source.
- **ARCH-MOCK — pass, with one note.** `deps.notifySignals` puts the signal transport behind an injected seam; `llmtest.Fake` is a wire-level httptest server, not a stubbed `Client`, so `askRig` exercises serialisation, headers and SSE parsing; the store fake runs the same conformance suite as YAML; the pty suite is the live conformance check for the terminal. The note is `store.Mem.userModel` above — a fake field that cannot be given a value is not yet modelling the dependency's state.
- **For upcoming work:** `askContext` is a good stable surface for `#10`/`#13` to reuse *once the selection policy moves out of the IO half*. And `repl`'s `context.WithoutCancel` means the loop no longer honours any caller cancellation — correct today because `main` is the only caller, but it is a trap for the next one; consider taking an explicit `stop <-chan struct{}` or documenting the contract at the signature.

## 7. Plan revision recommendations

Add a `## Revisions` section entry (2026-08-23, M2 close) covering:

1. **`askCapturer` was not built.** The `asked` event is appended directly through `d.deck` from `ask.go:145`, not through a `Capturer` wrapper in `capture.go`. State why (the event is not a lookup and `decideCapture`'s policy does not apply), and record that `DEFINE_NO_CAPTURE` still suppresses it via a nil `deck` rather than via the capture seam — plus the consequence that `main.go:31-34`'s "capture is the only thing that RECORDS" comment needs updating.
2. **Entity locations moved.** `exchange` landed in `askctx.go` beside `askContext`, not in `session.go`; `interrupter` landed in its own `interrupt.go`, not in `rawterm.go`. Correct both rows in Core concepts.
3. **Task 11's test decomposition changed.** The plan specified `TestEditorCtrlCMidStreamReturnsToThePrompt` *and* `TestEditorSignalMidStreamReturnsToThePrompt`; the code ships one test with two subtests, and the "signal" subtest calls `interrupter.Fire()` directly rather than driving `d.notifySignals`. Record it, and record that the combination "SIGINT through `repl`'s watcher during a *scoped* stream" is therefore still unasserted.
4. **The M2 checkboxes.** Tick the 31 delivered steps in Chunk 2 (or mark the ones that were not, e.g. Task 7's "run the pty suite unchanged" and Task 12's atlas/index review).
5. **Boundary rounds.** Per BR-17, still open from M1: the plan needs a `## Revisions` entry per boundary review round, not only per plan-gate round.

```findings
findings:
  - id: new
    severity: Critical
    family: policy-in-io-shell
    title: |
      "Recently in the deck" sends the twelve OLDEST deck words, unreversed
    detail: |
      cmd/define/ask.go:180-187. Store.Deck() is newest-first (store.go:14,
      sortDeck at mem.go:62-69) and lastN takes the tail, so a deck larger than
      maxContextWords sends the oldest twelve and drops the most recent. Measured
      with a 15-word deck: the section renders "wl, wk, ... wa" and the three
      newest words never reach the model. The code's own comment states both
      intentions it fails ("reads better oldest-first", "the newest are the ones
      worth keeping"), and README:82 plus atlas/define.md:522 both promise "the
      recent deck". The selection is a PURE decision inlined in the IO gatherer,
      which is why no test above the bound exists (ARCH-PURE) — fix by extracting
      recentDeck() beside recentTurns and table-testing it.
  - id: new
    severity: Critical
    family: forced-route-enumeration
    title: |
      One-shot unforced ask puts the question text into session.current and the event log's word field
    detail: |
      This is the 3rd finding in family forced-route-enumeration; earlier rounds
      fixed instances, so fix the CLASS. cmd/define/main.go:387 passes
      &session{current: oneShot.word}, where oneShot.word is the line the
      dictionary just missed — the question. Measured with a live seam and a real
      YAML store: the prompt renders "## The word on screen\n<the question>" with
      no entry, and recordAsked (ask.go:150) writes ReviewEvent{Word: "<the
      question>"} into the append-only log #17 folds over. That violates plan D2
      and Task 4's contract table ("does not set sess.current"). The forced
      one-shot cell passes &session{} and is correct, so the two routes diverge.
      The rule: every fact the ask path carries is quantified over {forced,
      unforced} x {one-shot, piped, editor} and each cell asserted for THAT fact.
      M2 added three such facts (which session, where the answer is written,
      whether the interrupt is scoped); measured coverage is 2 of 18 cells.
  - id: new
    severity: Important
    family: silent-error-swallow
    title: |
      gatherAskContext discards UserModel's error, defeating the reason it returns one
    detail: |
      cmd/define/ask.go:172-174 writes `if model, err := d.deck.UserModel(); err
      == nil`. store/yaml.go:44-48 justifies returning an error precisely to
      prevent this: "silently answering \"\" for a model that exists would make
      every answer pitched at the wrong level with no way to tell." Warn on
      stderr (the storeCapturer/storeHistory precedent) or delete the
      justification.
  - id: new
    severity: Important
    family: unsynchronised-test-observation
    title: |
      go test -race ./cmd/define/ now fails; two new tests read a bytes.Buffer concurrently
    detail: |
      Verified: base 993fccc is race-clean, HEAD 686913c reports eleven races and
      two failures. askrun_test.go:147-151 spins on out.Len() while runAsk writes
      out; askrun_test.go:267,280 read out.String() via waitFor while runEditor
      writes it. The package already owns the correct shape — ptyOut's
      mutex-guarded buffer at pty_conformance_test.go:88-124 — so this is also
      ARCH-DRY. Extract one syncBuf and use it in all three places.
  - id: new
    severity: Important
    family: plan-contract-drift
    title: |
      The plan's askCapturer integration point was not built; events append through a second write path
    detail: |
      Core concepts -> Integration points lists askCapturer on Capturer in
      capture.go wrapping the event append. The code appends directly via
      d.deck.AppendEvent from ask.go:145-152. Behaviour is safe (DEFINE_NO_CAPTURE
      leaves sd.deck nil so recordAsked no-ops), but it adds a second writer to
      the event log beside capture and makes main.go:31-34 stale ("capture is the
      only thing that RECORDS lookups. deck ... --forget deletes through it").
      Either route it through the seam or add a "## Revisions" entry and fix the
      comment.
  - id: new
    severity: Important
    family: readme-surface-gate
    title: |
      README does not say that question text is persisted to events/
    detail: |
      This is the 2nd finding in family readme-surface-gate, so state the rule
      rather than adding one line. README:95-105 is the block documenting what
      define writes to the working directory; M2 makes every question's text
      persist into events/*.yaml and that block does not say so. The rule: any
      new record type or field written into the working directory is named in
      that block in the same change. Enumeration: {words/, events/,
      user-model.md} x {looked-up, reviewed, asked}; README currently names 2 of
      3 files and 0 of 3 kinds.
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      atlas/define.md:182-186 still states the pre-M2 cancellation model the diff disproved
    detail: |
      This is the 5th finding in family doc-overstates-code; do NOT fix only this
      paragraph. It claims Ctrl-C is "not a signal", that the piped path "still
      relies on" signal.NotifyContext, and that the key reader calls cancel().
      All three are false after this diff: the pty suite measured \x03 arriving
      as SIGINT (D5's premise), repl.go:180 detaches with WithoutCancel, and
      readKeys calls interrupts.Fire(). The diff corrected the code comment at
      rawterm.go:44-46 and added a NEW atlas section at :544 while leaving the
      old statement standing. The rule: when a diff corrects a claim in a code
      comment, every restatement of that fact elsewhere is a consumer of the same
      source and must be swept in the same change (ARCH-PURPOSE shadow-sweep) —
      grep the fact, do not append beside the stale copy. Measured: 1 of 2
      NotifyContext mentions in atlas/ is now false.
  - id: new
    severity: Important
    family: loop-shell-branch-untested
    title: |
      The piped loop's ask wiring is unpinned — answer destination and session both deletable green
    detail: |
      This is the 2nd finding in family loop-shell-branch-untested; state the
      rule. Two mutations at repl.go:261 each leave go test ./cmd/define/ fully
      green: replacing stdout with io.Discard (the answer goes nowhere), and
      replacing &sess with &session{} (the piped loop loses multi-turn and
      session words). lessons.md define #15 already states the rule; M2 obeyed it
      for runEditor and skipped it for replLines and the one-shot. That skipped
      cell is where the one-shot Critical lives — the same six-cell enumeration
      fixes both.
  - id: new
    severity: Important
    family: test-asserts-nothing
    title: |
      Four claims in this diff survive the mutation they exist to catch
    detail: |
      This is the 5th finding in family test-asserts-nothing; do NOT fix the four
      instances, state the rule. Measured individually, all green: (1) the
      key-channel buffering rawterm.go:49-54 calls "load-bearing" — make(chan
      Key) instead of make(chan Key, 256); (2) session words reaching the prompt,
      an issue Done-when row — SessionWords: nil in gatherAskContext; (3)
      recordExchange's empty-answer guard, session.go:44-46 — delete it; (4)
      askrun_test.go:293-296 claims to pin "the interrupt scoping and the CRLF
      writer" for both routes but passes nil for interrupts, so it can only pin
      the writer. Row (2) is the BR-2 shape exactly: the test asserts
      "sycophantic" is in the prompt, but CurrentWord supplies that string, so
      the SessionWords assertion is satisfied by a different source. The rule: an
      assertion that a value reached an output must use a value only that source
      can supply. Also applies to
      TestEditorCtrlCMidStreamReturnsToThePrompt/the_signal_transport, which
      calls interrupter.Fire() directly and never touches d.notifySignals.
  - id: new
    severity: Important
    family: raw-mode-message-placement
    title: |
      assertNoBareNewline is vacuous for all three rows of TestRawLoopMessagePlacement
    detail: |
      This is the 5th finding in family raw-mode-message-placement; state the
      rule. The helper does strings.TrimSuffix(s, "\n") at askroute_test.go:395
      to excuse the loop's post-finish() newline, and all three rows write
      exactly one line to stderr — so the excused newline is the only one under
      test. Measured: unwrapping stderr from crlfWriter at replraw.go:141 leaves
      the suite green, and changing the bare-? note's "\r\n" to "\n" at
      replraw.go:226 leaves TestRawLoopMessagePlacement green. The test's own
      header claims "EVERY message this loop writes ... carries its own carriage
      returns" — vacuous for 3 of 3 rows. The rule (BR-14's rule applied to
      placement): a placement assertion asserts the POSITIVE observable, that the
      message's own terminator IS "\r\n", never the absence of a bare "\n".
  - id: new
    severity: Important
    family: plan-bookkeeping
    title: |
      31 unticked M2 plan steps and no Revisions entry for any boundary round or M2 design departure
    detail: |
      This is the 3rd finding in family plan-bookkeeping; state the rule.
      workshop/plans/000016-console-qa-plan.md Chunk 2 has 31 unticked steps and
      0 ticked, while the issue's ## Plan and the project row both mark M2 [x].
      The plan still carries three ## Revisions entries, none of which is a
      boundary round (BR-17, raised in M1, never disposed), and none covering
      M2's three design departures. The rule: the plan artifact's state is part
      of the milestone deliverable — the checkbox sweep plus a ## Revisions entry
      for every fork taken differently from the plan happen in the
      milestone-close commit, not at issue close.
  - id: new
    severity: Minor
    family: doc-overstates-code
    title: |
      runAsk's `parent := ctx` is a bare alias with no derived context
    detail: |
      ask.go:100-102 copies llmcheck.go:45-47's idiom, but llmcheck derives a
      WithTimeout child so parent and ctx genuinely differ. Here they are the
      same value and the comment describes a distinction that does not exist. The
      observed cancel path also returns ErrTruncated, not ErrUnavailable, so the
      failure mode the comment names is the pre-text case only.
  - id: new
    severity: Minor
    family: plan-contract-drift
    title: |
      Core concepts table cites the wrong file for `exchange` and `interrupter`
    detail: |
      `exchange` is in askctx.go:9-12, not session.go; `interrupter` is in the
      new interrupt.go, not rawterm.go. Both entities exist, are pure and are
      tested, so this is a stale table rather than a missing deliverable —
      correct the rows in the plan revision.
  - id: new
    severity: Minor
    family: stdlib-reuse
    title: |
      bytesReader is strings.NewReader under a new name; keysFor is dead
    detail: |
      This is the 3rd finding in family stdlib-reuse. interrupt_test.go:98
      defines `func bytesReader(s string) io.Reader { return
      strings.NewReader(s) }` for one call site. The rule (same as
      contains->slices.Contains two rounds ago): before adding a helper, check
      whether the stdlib already names it. Separately, askrun_test.go:334-347's
      keysFor is defined and never called, with a doc comment describing a use
      case that does not exist.
  - id: new
    severity: Minor
    family: doc-overstates-code
    title: |
      DEFINE_NO_CAPTURE now also suppresses READING user-model.md and the deck
    detail: |
      openStore leaves deck nil under noCapture, so gatherAskContext (ask.go:167)
      skips both the learner model and the deck. README documents the variable as
      "write nothing in this directory"; answers silently become un-adapted. Also
      note unavailable() reports "no model configured" for a configured but
      unreachable model — plan-specified, but the same misdiagnosis surface as
      the already-filed tools#19.
  - id: new
    severity: Minor
    family: dead-branch
    title: |
      askInSession always returns nil, so lostTerminal is unreachable at both ask call sites
    detail: |
      replraw.go:130-155 no longer calls cooked(), so the closure cannot fail;
      the `if err := askInSession(...); err != nil { return lostTerminal(err) }`
      guards at both call sites are dead. Also crlf.go:28 returns 0 on a write
      error rather than the bytes consumed, and reports len(p) on a short
      underlying write; and ask.go:114 prints a newline on cancel even when no
      delta ever arrived.
```

---

## Re-review — 2026-08-23T23:40:16-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 993fccc020f3a003c792154917585f998ecbbe7c..99dfd89ef58d48eda3f7780a45e59df0445281dc |
| command | sdlc milestone-close --issue 16 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-23T23:40:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 5's two Criticals are genuinely dead: I reverted each fix in a scratch worktree and both tests went red (`TestTheDeckSectionCarriesTheNewestWords`, `TestOneShotQuestionHasNoCurrentWord`), and the same held for eight more claimed fixes I mutation-checked — the piped loop's two wirings, the key-channel buffering, `SessionWords`, the empty-answer guard, the shared-wiring interrupt scope, and both halves of `assertCRLFTerminated`. `go test ./...` and `go test -race ./cmd/define/...` are both green (the base was race-clean and HEAD now is too), `go vet ./...` and `go vet -tags conformance ./cmd/define/` are clean, and the plan is fully ticked with a Revisions entry. This is the strongest round of the issue. What keeps it from SHIP is that the round fixed the *cells the review named* rather than the *table the review asked for*: with the full suite green I can still hand the raw editor's ask a fresh `&session{}` and send both one-shot answers to `io.Discard`. Underneath that, `doc-overstates-code` reaches its 8th occurrence with four measured instances, one of them a comment that contradicts the code five lines below it inside the function this milestone rewrote.

## 1. Strengths

- **The fixes are pinned, not asserted.** Ten reversions, ten reds. `cmd/define/askrun_test.go:98` uses `deckword00..19` — a fixture only the deck can supply — which is the BR-30 rule actually applied rather than restated.
- `cmd/define/syncbuf_test.go:13` folds `ptyOut` onto one mutex-guarded buffer instead of adding a third copy (`pty_conformance_test.go:91`). ARCH-DRY done right, and it turned a real `-race` failure green.
- `cmd/define/capture.go:49` — `CaptureAsk` on `Capturer`. `AppendEvent` now has exactly one non-test caller file (`capture.go`), so `deps`' "capture is the only thing that RECORDS" comment is true again.
- `cmd/define/askroute_test.go:366` — `assertCRLFTerminated` counts terminators instead of excusing one. Both mutations the old helper survived now redden it.
- **The store is robust against the new free-form field.** I round-tripped five adversarial questions through `store.YAML` — including one whose text is a complete forged `- word: injected` record — and all five survived intact: `yaml.Marshal` emits an indented block scalar, so `splitRecords` (`yaml.go:300`) never sees a top-level `- `. Correct by construction, just unasserted (finding I3).

## 2. Critical findings

None.

## 3. Important findings

**I1 — three cells of the ask-wiring table still survive their mutation, full suite green.** *(3rd in family `loop-shell-branch-untested`.)*

BR-23 and BR-29 both asked for the enumeration to be written as a table with each cell asserted. Three targeted tests were added instead. Measured against the **whole** `./cmd/define/` suite, not a `-run` subset:

| fact \ mode | one-shot | piped | raw editor |
|---|---|---|---|
| which session the ask carries | ✓ `TestOneShotQuestionHasNoCurrentWord` | ✓ `TestThePipedLoopsAskWiring` | **GREEN** |
| where the answer is written | **GREEN** (both routes) | ✓ | ✓ |
| whether the interrupt is scoped | n/a | see I2 | ✓ |

- `cmd/define/replraw.go:161` — `&sess` → `&session{}`: `ok github.com/xianxu/tools/cmd/define 28.160s`. The raw editor is the primary UI, and "a follow-up resolves against the previous exchange" is a Done-when row.
- `cmd/define/main.go:366` and `main.go:393` — `stdout` → `io.Discard` on both one-shot routes, together: `ok … 28.391s`. `define "what's the difference to obsequious?"` printing nothing at all is caught by no test.

Do NOT add three more one-off tests. The deliverable is the table above, written once as a fixture-driven test whose rows are the cells — the repo already has the working precedent in `TestRawNeverAsks`, which names its six cells and covers every one.

**I2 — `doc-overstates-code`, 8th occurrence: four measured instances, three of them created by this window.**

The rule has now been stated three times (BR-9's absolute-names-its-enumeration, BR-28's grep-the-fact shadow-sweep) and keeps recurring, so the escalation has to be mechanical rather than another sentence. Measured:

1. `cmd/define/replraw.go:143` — *"It runs COOKED for the same reason a command does"*, five lines above `replraw.go:157`'s *"Streamed in RAW mode … rather than under `cooked()`"* and the code that does exactly that. The comment block contradicts itself; commit `1f28d54` rewrote the body and left the header.
2. `atlas/define.md:574` — *"all six go through one `ask(opt options, errOut io.Writer, q question)`"*. That signature became false in this window (`ask.go:41` is now seven parameters). Enumeration is greppable: 3 quoted Go signatures in `atlas/define.md`, 1 stale, and it is the one this diff changed.
3. `README.md:78` — *"Ctrl-C stops the answer rather than the session"*, stated unconditionally. Measured by driving `repl` with the injected signal transport, stdin a tty and stdout redirected: **the line loop ends the session** (`replLines`' `askHere` at `repl.go:261` passes the loop's own `ctx`, which the default sink cancels). True in 1 of 2 interactive loops.
4. `README.md:97` — *"**Every question you ask is recorded too, by its text**"*. Measured: an errored ask (HTTP 400) → **0 events**; an unwired seam → **0 events**; a cancelled ask → **0 events and 0 turns**. True in 1 of 4 outcome cells.

The mechanical fix that retires the class: **atlas and README stop restating a signature or an absolute** — name the function and let the code be the source; and every absolute that survives gets the `TestRawNeverAsks` treatment (its enumeration named in the test, one row per cell). Instances 3 and 4 are each one row.

**I3 — the event log gained its first user-controlled free-form field and nothing pins the format invariant.** `cmd/define/store/storetest/suite.go:52` round-trips one plain question. Until #16 every value written to `events/*.yaml` was a single dictionary headword; `question:` is now arbitrary text the user typed. The reader's record boundary is a literal top-level `- ` (`yaml.go:307`), and the only thing keeping user text off column 0 is that `yaml.Marshal` indents block scalars. I verified today's behaviour is correct (see Strengths) — but that invariant is defended by nothing, and a regression corrupts an append-only log #17 folds over, irreversibly. BR-4's precedent is exactly this shape: an event that looked whole was silently discarded at read time. Add a suite row with a question containing a newline, a leading `- `, and an embedded `at:`, asserting count and content survive.

## 4. Minor findings

- `cmd/define/crlf.go:41` — `consumed(p, out, n)` never reads `out`, and re-derives the translation with `lastWasCR := false` instead of the writer's carried state. Split `"a\r"` then `"\nb"` with a 1-byte short write and it reports 0 consumed where 1 was written — the duplicate-on-retry the fix exists to prevent, surviving in the one case `lastWasCR` exists for. `TestCRLFWriterReportsProgressOnAShortWrite` covers only the fresh-state cell. Derive from `out`, or save the entry-state flag.
- `cmd/define/ask.go:176` — `newestFirst` returns oldest-first, and lives in the IO shell rather than beside `recentTurns` in `askctx.go`; its only test needs a real YAML store plus a wire fake to exercise a pure ordering decision. (2nd in `policy-in-io-shell`; BR-22's defect itself is gone.)
- `cmd/define/pty_conformance_test.go:63` — `startDefine` skips when `bin/define` is absent but never checks it is current. The binary on disk is timestamped 22:45, **34 minutes before** the round-5 fix commit, so a `-tags conformance` run right now validates pre-fix code and reports `ok`. Compare against the source mtime, or build in `TestMain`.
- `cmd/define/ask.go:113` — a Ctrl-C'd partial answer is printed to the user but dropped from `sess.turns`, so the follow-up the feature is built for cannot refer to what was just read.
- `cmd/define/main.go:366` / `main.go:393` — the two one-shot `ask(...)` call sites differ only in `forced` and could be one expression.

## 5. Test coverage notes

- **Verified red under reversion:** the deck ordering; the one-shot session; `CaptureAsk`; both piped-loop wirings; the key-channel buffering (fails via timeout, ~35s); `SessionWords`; the empty-answer guard; the shared-wiring interrupt scope; `assertCRLFTerminated` for both the unwrapped-stderr and the bare-`\n` mutation; `YAML.UserModel`'s file read.
- **Unpinned:** the three cells in I1; the `UserModel` warn (I1 of round 5 — see the disposition); the format invariant in I3.
- **`store.Mem.userModel` still has zero setters** (`mem.go:19`), so the conformance suite can only assert the empty case for the fake, while YAML's non-empty read is pinned elsewhere. ARCH-MOCK: the fake does not model the state the real one holds. #17 should add `SetUserModel` and a non-empty suite row rather than inherit a field that does nothing.
- `TestPTYCtrlCMidAnswerKeepsTheSession` **skips here** (`no pty available: operation not permitted`), so its claim rests on the implementor's note — and see the staleness Minor above.

## 6. Architectural notes

- **ARCH-DRY — pass, one flag.** `syncBuf` collapsing `ptyOut`, one `askInSession`, one `fail(code)` sink, one `AppendEvent` caller: all real. Flagged: `consumed()` is a second implementation of a translation the first one already materialised into `out`, and the two copies already disagree.
- **ARCH-PURE — pass, one note.** `renderAskPrompt` and `recentTurns` are pure and table-tested; `gatherAskContext` is a genuinely thin IO step. `newestFirst` is now a named pure function — the defect is gone — but its location and its store-backed-only test keep the smell BR-22 named.
- **ARCH-PURPOSE — flag.** Shadow-sweep over the consumers of "the directory is the context": raw editor, piped loop and one-shot all derive correctly today. The flag is the principle's own second clause — *a finding names one instance; the deliverable is the CLASS.* BR-23 and BR-29 each named the table; the round fixed the cells and skipped the table, and 3 of 7 applicable cells are still green under mutation. `family:` repeating across rounds is the ledger reporting the enumeration was never written.
- **ARCH-MOCK — pass, two notes.** `llmtest.Fake` is a wire-level httptest server (so serialisation, headers and SSE parsing are exercised), `askRig` uses a real YAML store in a temp dir, `deps.notifySignals` seams the signal transport, and the pty suite is the live conformance check. Notes: `Mem.userModel`'s missing setter, and the pty suite's unversioned binary.
- **For upcoming work (#17, #10):** `askContext` is a good stable surface to reuse — move `newestFirst` beside `recentTurns` before a second consumer copies its location. And `repl`'s `context.WithoutCancel` means the loop honours no caller cancellation at all; correct while `main` is the only caller, but the next caller will not expect it — say so at the signature or take an explicit stop channel.

## 7. Plan revision recommendations

The plan now matches the code on the two entity rows and carries an M2 departures entry. One item from round 5's §7 was not recorded — add it to `## Revisions`:

> **Task 11's test decomposition changed.** The plan specified `TestEditorCtrlCMidStreamReturnsToThePrompt` *and* `TestEditorSignalMidStreamReturnsToThePrompt`; the code ships one test with two subtests, and the "signal transport" subtest calls `interrupter.Fire()` directly (`askrun_test.go:377`) rather than driving `d.notifySignals` through `repl`'s watcher. Record that the cell "SIGINT through `repl`'s watcher **during a scoped stream**" is therefore still unasserted — `TestBothInterruptTransportsReachTheSink` covers the unscoped session only.

```findings
dispose:
  - id: BR-22
    disposition: addressed
    note: |
      newestFirst takes the head and reverses; reverting to lastN reddens TestTheDeckSectionCarriesTheNewestWords (verified).
  - id: BR-23
    disposition: addressed
    note: |
      main.go:393 passes &session{}; restoring current: oneShot.word reddens TestOneShotQuestionHasNoCurrentWord (verified).
  - id: BR-24
    disposition: not-addressed
    note: |
      The warn is correct and reachable, but no test fails without it — and failingStore.UserModel, added by this same diff, is at zero call sites.
  - id: BR-25
    disposition: addressed
    note: |
      go test -race ./cmd/define/... is ok at HEAD; syncBuf is used in all three places and ptyOut folds onto it.
  - id: BR-26
    disposition: addressed
    note: |
      capture.go is the only non-test AppendEvent caller; a no-op CaptureAsk reddens three tests (verified). Plan Revisions records the departure.
  - id: BR-27
    disposition: addressed
    note: |
      README:99-103 names events kinds, user-model.md and "answers are NOT stored"; pinned by TestTheEventLogHoldsQuestionsAndNotAnswers.
  - id: BR-28
    disposition: addressed
    note: |
      atlas/define.md:182-191 rewritten; the surviving NotifyContext mention at :676 is about the one-shot path and is still true. The family recurs elsewhere — raised separately.
  - id: BR-29
    disposition: addressed
    note: |
      Both repl.go:261 mutations now redden TestThePipedLoopsAskWiring (verified). The sibling cells the finding did not name are raised separately.
  - id: BR-30
    disposition: not-addressed
    note: |
      Four of the five named items are fixed and mutation-verified red; the fifth is untouched — the "signal transport" subtest still calls interrupts.Fire() directly and never reaches d.notifySignals, and no Revisions entry records it.
  - id: BR-31
    disposition: addressed
    note: |
      assertCRLFTerminated counts terminators; both the unwrapped-stderr and the bare-newline mutation now redden TestRawLoopMessagePlacement (verified).
  - id: BR-32
    disposition: addressed
    note: |
      60 of 60 plan checkboxes ticked, and a "M2's design departures, and the M1 rounds" Revisions entry now exists.
  - id: BR-33
    disposition: addressed
    note: |
      The parent alias is gone; runAsk asks ctx.Err() directly.
  - id: BR-34
    disposition: addressed
    note: |
      Core concepts now cites askctx.go for exchange and interrupt.go for interrupter.
  - id: BR-35
    disposition: addressed
    note: |
      Neither bytesReader nor keysFor appears anywhere in cmd/define.
  - id: BR-36
    disposition: addressed
    note: |
      README:118-120 documents that DEFINE_NO_CAPTURE also suppresses reading the deck and user-model.md.
  - id: BR-37
    disposition: addressed
    note: |
      askInSession returns nothing and lostTerminal still has two live call sites; crlf reports caller-unit progress; the cancel newline is guarded by answer.Len(). The new consumed() helper has its own defect, raised separately.
findings:
  - id: new
    severity: Important
    family: loop-shell-branch-untested
    title: |
      Three cells of the ask-wiring table still survive their mutation with the full suite green
    detail: |
      This is the 3rd finding in family loop-shell-branch-untested, and the 2nd
      round in which the enumeration BR-23 wrote out was answered with targeted
      tests instead of the table. Measured against the whole ./cmd/define/ suite,
      not a -run subset: replraw.go:161 &sess -> &session{} leaves it green (ok,
      28.160s) — the raw editor is the primary UI and multi-turn is a Done-when
      row; main.go:366 and main.go:393 stdout -> io.Discard, applied together,
      leave it green (ok, 28.391s) — `define "what is the difference to
      obsequious?"` printing nothing is caught by nothing. Coverage of the
      {one-shot, piped, editor} x {which session, answer destination, interrupt
      scope} table is 4 of 7 applicable cells. Do NOT add three more one-off
      tests: write the table as one fixture-driven test whose rows ARE the cells,
      the way TestRawNeverAsks already does for its six.
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      Four measured doc claims contradict the code, three of them created by this window
    detail: |
      This is the 8th finding in this family; the rule has been stated three
      times (BR-9's absolute-names-its-enumeration, BR-28's shadow-sweep) and
      keeps recurring, so the escalation must be mechanical. Measured:
      (1) replraw.go:143 "It runs COOKED for the same reason a command does",
      five lines above the code and comment saying it streams RAW through
      crlfWriter; (2) atlas/define.md:574 restates ask's pre-M2 signature `ask(opt
      options, errOut io.Writer, q question)` — 1 of the 3 quoted Go signatures in
      that file is stale and it is the one this diff changed; (3) README:78
      "Ctrl-C stops the answer rather than the session" — measured through the
      injected signal transport with stdin a tty and stdout redirected, the LINE
      loop ends the session, because replLines' askHere (repl.go:261) passes the
      loop's own ctx that the default sink cancels; true in 1 of 2 interactive
      loops; (4) README:97 "Every question you ask is recorded too, by its text" —
      measured 0 events for an errored (400) ask, 0 for an unwired seam, and 0
      events plus 0 turns for a cancelled ask; true in 1 of 4 outcome cells. The
      mechanical fix: atlas and README stop restating signatures and unqualified
      absolutes, and each surviving absolute gets a named-enumeration row test
      like TestRawNeverAsks.
  - id: new
    severity: Important
    family: user-text-in-record-format
    title: |
      The event log's first free-form user field has no test defending the record-boundary invariant
    detail: |
      Until this window every value written to events/*.yaml was a single
      dictionary headword; `question:` is now arbitrary text the user typed. The
      reader's record boundary is a literal top-level "- " (yaml.go:307) and the
      only thing keeping user text off column 0 is that yaml.Marshal indents
      block scalars. I verified today's behaviour is correct — five adversarial
      questions round-trip intact, including one whose text is a complete forged
      "- word: injected / kind: looked-up / at: ..." record — but
      storetest/suite.go:52 round-trips one plain question and nothing pins the
      invariant. A regression corrupts an append-only log #17 folds over,
      irreversibly; BR-4 is the precedent for an event that looked whole being
      silently discarded at read time. Add a suite row with a newline, a leading
      "- " and an embedded "at:".
  - id: new
    severity: Minor
    family: second-implementation-drifts
    title: |
      crlf.go's consumed() ignores the `out` it is handed and re-derives the translation from a wrong seed
    detail: |
      consumed(p, out, n) at crlf.go:41 never reads `out` — a dead parameter —
      and restarts the translation with `lastWasCR := false` instead of the
      writer's carried entry state. Write "a\r" then "\nb" with a 1-byte short
      write and it reports 0 consumed where 1 byte was written, so a retry
      duplicates the newline: the exact defect the fix's own comment says it
      prevents, surviving in the one case lastWasCR exists for.
      TestCRLFWriterReportsProgressOnAShortWrite covers only the fresh-state
      cell. Derive from `out`, or capture the entry flag before the loop.
  - id: new
    severity: Minor
    family: policy-in-io-shell
    title: |
      newestFirst returns oldest-first, sits in the IO shell, and is only reachable through a store-backed test
    detail: |
      This is the 2nd finding in family policy-in-io-shell; BR-22's defect itself
      is genuinely gone. What remains is the shape BR-22 named: a pure ordering
      policy living in ask.go rather than beside recentTurns in askctx.go, whose
      only test (TestTheDeckSectionCarriesTheNewestWords) needs a real YAML store
      and a wire-level fake to exercise a slice reversal. The name also says
      newest-first while the function returns oldest-first.
  - id: new
    severity: Minor
    family: test-asserts-nothing
    title: |
      The pty conformance suite runs whatever bin/define is on disk, with no staleness check
    detail: |
      startDefine (pty_conformance_test.go:63) skips when ../../bin/define is
      absent but never checks it is current. The binary here is timestamped
      22:45, 34 minutes before the round-5 fix commit at 23:19, so a `go test
      -tags conformance` run right now would validate pre-fix code and report ok
      — a live conformance check that cannot fail on the change it exists to
      check. Compare against the newest source mtime, or build in TestMain.
```
