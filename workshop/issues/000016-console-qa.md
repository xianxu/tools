---
id: 000016
status: working
deps: [tools#3, tools#11]
github_issue:
created: 2026-08-22
updated: 2026-08-23
estimate_hours: 6.49
started: 2026-08-23T13:35:58-07:00
---

# free-form Q&A in the console: input classification + the directory as context

## Problem

You look up `sycophantic`, and the next thing you want is *"what's the difference
to obsequious?"* — with examples. Today that means leaving the tool.

The wider point (operator, 2026-08-22): the software should meet the learner where
they are, not require them to conform to a way of using it. Free-form learning is
the third verb, alongside definition and pronunciation — not a mode you switch into.

## Spec

Typing a question at the prompt answers it. There is no mode, no prefix to
remember, and no ceremony.

### Input classification — one decision table

`parseREPLLine` already splits `/command` from lookup. This adds a third outcome,
and it must stay ONE parser: two loops re-implementing "what does this line mean"
is a mistake this repo has already paid for twice (`lessons.md`, define #14).

Word count cannot be the signal — `hot dog` is a two-word headword and
`defenestrate` is one word. The signal that works is free, offline and already
present: **ask NOAD first.**

| input | classified as |
|---|---|
| `/history 7` | command (unchanged — `/` in column 1) |
| `sycophantic`, `hot dog` | lookup — NOAD has an entry |
| `what's the difference to obsequious?` | question — no entry, reads interrogative |
| `sycophanti` | not-found (unchanged) — no entry, does not read interrogative |

The interrogative test is a pure function over the line, table-tested against a
fixture set. Both directions of failure need an escape hatch, and both must be
cheap: a way to force a question, and a way to force a lookup. Neither may be the
only way to reach its outcome.

### The directory is the context

Not a chat history. `define` was started in a directory that holds `words/`,
`events/` and (once #17 lands) `user-model.md`. The question goes to the model with:

- the last few words of this session, and the recent deck from the store;
- the current word's NOAD entry, when the question follows a lookup;
- `user-model.md`, so the answer pitches at the right level and register.

Three consequences fall out, and they are the reason for this shape: a fresh
process answers just as well as a long-running one; the context is inspectable as
files rather than trapped in memory; and the answer is adaptive for the same reason
the generated items are.

### Shape

- Streams, because a paragraph arriving all at once after four seconds reads as a
  hang. Interruptible with Ctrl-C like everything else in the loop (raw mode: the
  key reader owns cancellation — see `lessons.md`, define #14).
- Multi-turn within a session: a follow-up *"give me three more examples"* resolves
  against the previous exchange.
- Q&A exchanges are recorded as events, so #17 can see what the learner asked about
  — a question is a strong signal of what they are working on.
- Seam unavailable → the classifier still runs and says so plainly (`define: no
  model configured; \`what's the difference…\` is not a word`), rather than
  attempting a dictionary lookup of a sentence.

## Done when

- [x] Classification is a pure function with a fixture table; every row in the
      table above is asserted, including `hot dog` as a lookup and a typo as
      not-found.
- [x] Both escape hatches work, and each is exercised by a test that fails when the
      hatch is removed.
- [x] A full Q&A round trip runs against the fake, with recent words and
      `user-model.md` visible in the recorded prompt.
- [x] A follow-up question resolves against the previous exchange.
- [x] Ctrl-C mid-stream returns to the prompt with the session intact.
- [x] With the seam unavailable, a question produces the explanatory message and a
      lookup still works.
- [x] Driven through the raw TUI loop, not only the piped loop — a wiring only a
      loop shell supplies must be pinned by a test that drives that loop shell
      (`lessons.md`, define #15).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), against
`workshop/plans/000016-console-qa-plan.md`'s task decomposition — one item per
task, plus the two review boundaries and the two atlas updates the milestones
commit to.

Design hours carry v2's ×0.2 spec-quality discount: the plan pre-resolves the
decision table, both hatches, the outcome contract, the interrupt enumeration and
the prompt's shape, so what is left at design time is reading rather than
deciding. Implementation hours are written at v3.1's 40% of the v2 primitive
table. Familiarity is 1.0 — same repo, same files, `internal/llm` and both REPL
loops shipped in the last two issues. Step 2.5 **is** satisfied and the credit
is taken at the slug rather than as a halving: #11 already built the transport,
the wire fake and the conformance suite, so T10 is priced `greenfield-go-module`
(0.47) rather than `api-integration` (0.80).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=1.00 impl=0.08
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: milestone-review       design=0.02 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: greenfield-go-module   design=0.25 impl=0.22
item: scope-pivot            design=0.35 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: greenfield-go-module   design=0.25 impl=0.22
item: cross-cutting-refactor design=0.12 impl=0.14
item: smaller-go-module      design=0.02 impl=0.10
item: milestone-review       design=0.02 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
design-buffer: 0.15
total: 6.49
```

| item | plan task |
|---|---|
| issue-spec | plan authoring — `000016-console-qa-plan.md`, **inside the measured window** |
| milestone-review ×3 | the three plan-quality gate rounds (PQ-1…PQ-6), also inside it |
| smaller-go-module | T1 `question.go` — `readsAsQuestion`, `truncateQuestion` |
| smaller-go-module | T2 `parseREPLLine` — `cmdAsk`, the `?` and `\` hatches |
| cross-cutting-refactor | T3 `lookupOutcome` through three call sites + the decision table |
| cross-cutting-refactor | T4a `session` replaces three `current` declarations |
| cross-cutting-refactor | T4b ask routing — two switches, `askInSession`, the one-shot dispatch **and** its usage guard, six named tests |
| smaller-go-module | T5 `/help` and the messages |
| milestone-review | M1 boundary review |
| atlas-docs | M1 atlas update |
| greenfield-go-module | T7 `crlfWriter`, `interrupter`, the signal seam, the detach |
| scope-pivot | T7's design has already moved twice on paper (PQ-1 → PQ-6 → the relocation); budgeting zero for a third move is the optimistic read |
| smaller-go-module | T8 store — `UserModel`, the `asked` event, suite rows |
| greenfield-go-module | T9 `askContext` + the pure prompt + golden |
| greenfield-go-module | T10 `runAsk` — seams, streaming, taxonomy, wire-fake tests |
| cross-cutting-refactor | T11 Ctrl-C through `readKeys`/`repl`/`replRaw` + pty row |
| smaller-go-module | T12 the hand-driven session check — operator-at-the-terminal, not autonomous impl |
| milestone-review | close review |
| atlas-docs | close — atlas + the project row |

### Revision — 2026-08-23, after estimate-quality

First derivation was **3.72**, and it priced only the tasks. The estimate-quality
judge pointed at the two nearest neighbours in this repo — #15 (1.82 est / 6.77
actual, 3.7×) and #11 (7.98 / 12.38, 1.55×) — whose `## Log`s name the same cause
this one inherited: *plan authoring and the plan-gate rounds sit inside what
`sdlc actual` measures*. This issue was claimed at 13:35 and the plan cleared at
15:04 after three gate rounds; none of that was in the first block, so the first
block was measuring a different thing than the actual will.

Five changes, four of them the judge's findings:

- **+`issue-spec` and +3×`milestone-review`** for the plan and its gate rounds —
  design-time work already spent inside the window. No ×0.2 discount: the
  discount credits a plan for collapsing later design, and this **is** that plan
  being written.
- **T4b `smaller-go-module` → `cross-cutting-refactor`.** It creates a file,
  changes three signatures, adds two switch cases, a closure, the one-shot
  dispatch and its usage guard, and ships six tests — it was priced at the same
  0.17 as T5, which adds one `/help` line.
- **+`scope-pivot` for T7.** Its design opened as Critical (PQ-1), reopened
  (PQ-6), and moved a third time; the primitive for "this shifts again" exists
  and was unused.
- **+`smaller-go-module` (0.02/0.10) for T12's hand-driven check**, which was
  folded into a 0.08 `atlas-docs` item alongside two other jobs.
- **Step 2.5 wording corrected.** The credit for "#11 already built the
  transport" *was* taken — at the slug: T10 is priced `greenfield-go-module`
  (0.47) rather than `api-integration` (0.80). The prose said the step found
  nothing, which named a satisfied condition and then skipped it.

Not taken: the judge's advisory that fanning M1's independent tasks out to
subagents compresses wall-clock below a sequential sum. Real, but it is a
within-session parallelism effect on the same unit — recorded here for #117's
ledger rather than netted against the findings above.

## Plan

Design: [`workshop/plans/000016-console-qa-plan.md`](../plans/000016-console-qa-plan.md).

- [x] Design via `sdlc start-plan` before implementing.
- [x] M1 — the console knows a question from a word: `readsAsQuestion`, the `?`
      and `\` hatches on `parseREPLLine`, routing from the NOAD miss branch, and
      the honest "no model configured" message in all three entry modes.
- [x] M2 — the answer: `askContext` + pure prompt, `runAsk` streaming through
      `internal/llm`, `Store.UserModel`, the `asked` event, scoped Ctrl-C.

## Log

### 2026-08-22

Created from the operator conversation that broadened `define-learn` to an adaptive
program. See the project file's `## Log` scope event for the surrounding decisions.

### 2026-08-23
- 2026-08-23: closed M1 — go test ./... + go vet green. Boundary rounds 1-3 (BR-1..BR-17) all fixed as rules with their enumerations, none deferred. Round 3: (BR-14) assertDidNotAsk now asserts the POSITIVE observable per cell — "no dictionary entry" unforced, the refusal forced — so the miss-branch guard removal reddens 3 unforced cells instead of 1; the two mayAsk guards are documented as producing different observables rather than as redundant. (BR-15) one fail(code) sink so a per-line dispatch code survives the loop (the BR-16 defect #15 fixed for commands, reintroduced by #16 for questions); READMEs four exit-code absolutes measured 8/8 on the built binary across {one-shot,piped}. Also: a recording cooked() proves the ask message is written inside cooked mode (the placement family had passed 4 rounds unasserted); both hatches symmetric in both session states; one nothingSays for what had been three note sites. Every rule mutation-verified individually.; review verdict: FIX-THEN-SHIP

`sdlc start-plan` run; durable plan written to
`workshop/plans/000016-console-qa-plan.md`. Three decisions worth surfacing
before code: the NOAD lookup is **not** repeated — routing happens inside
`lookupAndRender` after its single `dict.Lookup`, which also keeps it the one
capture site (ARCH-DRY); a question is routed **before** capture, so it never
lands in the event log as a not-found lookup (#8/#17 fold over that log); and
Ctrl-C mid-stream has to mean something narrower than everywhere else, so the raw
key reader's cancel becomes a swappable sink rather than the session cancel
(ARCH-PURE keeps the swap testable without a terminal).

Three open questions for the operator are listed at the foot of the plan: the
choice of `\` for the force-literal hatch, whether the ≥5-word arm of the
classifier is too generous, and whether `Store.UserModel` belongs here or in #17.

### 2026-08-23 — residual estimate risk, recorded not repriced

`sdlc change-code` cleared plan-quality after three rounds (PQ-1…PQ-6, ledger in
`workshop/plans/000016-console-qa-plan-gate.md`) and estimate-quality passed the
revised block as a derivation. Two risk signals it raised were **not** folded into
the number, deliberately — a second repricing round before any code exists is the
optimism it was warning about, in a different shape. Recorded here so that when
`sdlc actual` measures this, the calibration has the hypotheses rather than just
the miss:

- **T11 is the item to doubt.** `impl=0.14` buys ~8 minutes for three tests, a pty
  row, and threading `interrupts.Set` through `readKeys`/`repl`/`replRaw` — the
  wiring the gate caught wrong twice on paper. `scope-pivot` insures T7's design
  churn; nothing insures T11 executing that contract against a real terminal.
- **No UX-iteration allowance on a feature that is entirely surface.** The
  `user-driven UX iteration round` primitive is unused, while everything the
  operator touches here is wording and rendering — the degradation message, the
  `\` hatch, how a streamed paragraph lands through `crlfWriter`. v2.1's own
  Known Limitations names this hole (3–5 rounds typical for TUI features).
  Design produced three operator questions before any code; budgeting zero
  rounds after code is the same bet.

Counterweight the judge itself supplied: #14 — the closest analogue, raw-mode
editor work — came in at **0.35×** its estimate. So the exposure is real but not
one-directional.

### 2026-08-23 — M1 closed: the console knows a question from a word

Five tasks, five commits, in plan order. The decision table is asserted
end-to-end by `TestConsoleDecisionTable` through the real route rather than
half-by-half, which is the only version that can catch the two halves disagreeing.

Three things worth keeping:

- **The ask outcome carries exit code 0**, because a question is not a failed
  lookup — which makes `if out.code == 0 { current = word }` exactly wrong. The
  guard is `out.ask == "" && out.code == 0`; mutation-checked by deleting the ask
  half, which reddens `TestAQuestionDoesNotBecomeTheCurrentWord`. This was PQ-3,
  found at the plan gate rather than in code.
- **`git checkout <file>` after a deliberate mutation restores HEAD, not the
  working tree**, so it silently discarded Task 4's uncommitted edits. Recovered
  from a copy made before the mutation. `lessons.md`'s "check `git status` after a
  mutation experiment" now has a second shape: *make the backup first, and restore
  from the backup, not from git.*
- **The sandbox blocks CoreServices**, so a piped run of the built binary reports
  "no dictionary entry" for every word and looks exactly like a regression.
  Verified unsandboxed instead.

Verified by hand against the real dictionary (not the fake): `sycophantic` and
`hot dog` define, `sycophanti` misses, `how so` routes to the model, `\how so`
stays a miss, `?why` asks despite `why` being a headword. Event log after six
lines: four events, neither question among them.

### 2026-08-23 — M1 boundary review round 1: BR-1…BR-4 + 4 Minor, all fixed

Verdict FIX-THEN-SHIP, 8 findings, and two of them were mutations the reviewer
ran that I had claimed were covered. Worth recording as a pair, because they are
the same mistake in two shapes — **a test that cannot fail**:

- **BR-1 / I-2 — the forced `?` branch was deletable from BOTH loops with the
  suite green.** Every loop-level ask test took the *unforced* route, and
  `routeFor` answers "question" for `cmdAsk` without entering a loop at all. The
  class, not the site: any branch reachable only through a loop shell needs a
  loop-shell test, and the enumeration that implies is
  `{replLines, runEditor} × {forced, unforced}` — four cells, three empty. Both
  new tests use `?why`, where `why` **is** a headword, so only the hatch can make
  it a question.
- **BR-2 / I-1 — `TestAQuestionIsRecalledByUpArrow` passed with `hist.Add`
  removed.** It asserted on stdout, and the raw editor re-renders the line on
  every keystroke, so the question was there from *typing* — before Enter, before
  Up. In a loop that echoes, an assertion on stdout is satisfied by the echo. Now
  asserted against the injected `History`, both routes.

All three mutations the reviewer found green now go red; re-verified after the
fixes.

- **BR-4** — the one-shot fell through to `defineOnce` for kinds it did not
  handle, and #16 gave the parser a new one: a bare `?` or `\` produced
  `cmdNothing` with an empty word, printed `define: : no dictionary entry`, and
  appended an event with no word that the log then discards at read time as
  indistinguishable from a torn record. Fixed exhaustively — the branch now
  handles everything that is not `cmdDefine` — rather than by special-casing `?`.
- **BR-3** — README documents this class (the `/` marker, the key table, the
  exit-code contract) and had none of it. Added, and the two behavioural claims
  it now makes are pinned by tests rather than asserted in prose.

Taken from the reviewer's M2 watch list and fixed **now**, since it is one
condition and shipped behaviour: **`-raw` never asks.** It is the scripting form
("records nothing, because it is for scripts"); routing a `-raw` miss to the
model costs a different message today and a network call for a piped line in M2.
Decided beside the `literal` flag, because both answer the same question — may
this miss fall back to a question?

Four Minor findings fixed in the same round rather than carried: the bare-`?`
note now erases the line like its siblings, the forced and unforced asks render
at the same height (the shared closure no longer emits a second `\r\n`), the
atlas sentence that claimed single-word lines are never questions is qualified
(`why?` is one), and `contains` is `slices.Contains`.

### 2026-08-23 — M1 boundary round 2: two repeat families, fixed as rules

Round 1's eight findings all disposed; round 2 returned two Important, and the
gate's own verdict on them was the useful part — *3 repeat families, not
converging: fix rules, not instances.* Both were second occurrences of families I
had "fixed" the instance of:

- **BR-9, family `doc-overstates-code`.** I put the `-raw` guard on the unforced
  fallback and wrote "`-raw` never asks" in the README. Measured: three of six
  cells still asked — `{forced} × {one-shot, piped, editor}` — because the forced
  route skips `lookupAndRender` entirely. **The rule: a claim stated as an
  absolute must name the enumeration it quantifies over, and every cell must be
  guarded and asserted.** `mayAsk` now lives with the ask rather than with either
  dispatch, and `TestRawNeverAsks` runs all six cells. Decided while fixing it:
  an explicit `?` under `-raw` is contradictory input, so it is a **usage error**
  (exit 2), matching this repo's `-forget`-with-a-word and `--sound`-with-`-times`
  precedents rather than guessing which flag was meant.
- **BR-10, family `test-asserts-nothing`.** BR-4's test injected a
  `countingCapturer` through `d.newStore`, but `withStore` only fills nils and
  `testDeps` had already supplied one — so the double was discarded and every
  assertion over `cap.calls` ranged over an empty slice. (The pre-existing
  convention is explicit: `TestNoCaptureWritesNothingToDisk` sets
  `deps.capture = nil // force the real wiring`.) **The rule: a test that injects
  a double must inject where production reads, or assert the injection took
  effect.** Done both — the injection moved to `d.capture`, and
  `TestTheCapturerInjectionIsLive` is the control that goes red if it is ever
  discarded again.

Two more families closed the same way rather than at their sites:

- **`raw-mode-message-placement`, 3rd occurrence.** Round 1's two placement fixes
  were unpinned — both could be reverted with the suite green. The rule: every
  message class the raw loop writes gets an assertion on the emitted byte stream.
  Three classes, three cells, all asserted. Getting this right needed one
  correction of my own assertion: the loop's final newline is written *after*
  `finish()` restores cooked mode, so a bare `\n` there is correct.
- **`forced-route-enumeration`, new.** `\how so` was recorded in history as
  `how so`, so Up-arrow + Enter re-submitted it as a **question** — the opposite
  of what the hatch forced. The rule: what recall stores must re-submit to the
  same meaning. `recallLine` is now the one canonical form and all three recall
  sites use it. Same family: `askUnavailable` told `?why` that "`why` is not a
  word", a claim nothing had checked, since the forced route never consults the
  dictionary. `question` now carries its route, and the sentence is only used
  where it is true by construction.

Every fix mutation-verified: reverting each of the four rules reddens its test,
and the `-raw` revert reddens exactly the three forced cells.

### 2026-08-23 — M1 boundary round 3: assertions that pass for the failure they exist to catch

Two Important, both third-or-later occurrences of families I had already
"fixed" — which is itself the finding worth keeping.

- **BR-14, family `test-asserts-nothing` (3rd).** `assertDidNotAsk` checked only
  that stderr LACKED "no model configured". Measured by the reviewer: remove the
  miss-branch `mayAsk` guard — a plausible cleanup, since `ask` guards too — and
  2 of the 6 cells stay GREEN, because the unforced miss then reaches `ask` and
  gets refused with advice to *drop the "?"* for a line containing no `?`. No ask
  message, assertion satisfied, scripting contract broken. **The rule: an
  assertion that pins "X did not happen" must assert the positive observable that
  distinguishes X from every other outcome, not the absence of one string.** Each
  cell now asserts what DID happen — `no dictionary entry` for the unforced
  cells, the refusal for the forced ones — and the mutation reddens all three
  unforced cells instead of one. The two guards are also no longer described as
  redundant in the comment that invited the cleanup: they produce different
  observables, and deleting either changes what a script sees.
- **BR-15, family `doc-overstates-code` (3rd).** `echo '?why' | define -raw`
  exited 1 where `ask` computed 2 and README stated 2: the piped loop collapsed
  the code into `anyFailed`. The comment 45 lines above names this exact defect
  by number (BR-16, #15, for commands) — so #16 reintroduced a fixed bug in a new
  branch. **The rule: one sink for a per-line dispatch's exit code, and every
  exit-code absolute in README measured across `{one-shot, piped}` before it is
  written.** All four branches now feed one `fail(code)`. Swept and measured on
  the built binary: 8 of 8 cells match README.

Round 3's Minor findings, fixed rather than carried:

- **4th occurrence of `raw-mode-message-placement`.** My own byte-stream
  assertion executed for 1 of the 3 classes it enumerated: `ask` writes to
  **stderr**, and the two ask rows asserted only on stdout. Deleting `cooked(...)`
  from `askInSession` — in production the thing that makes the message's `\n`
  translate — left the suite green, because the rig's `cooked` is a no-op. Fixed
  with a **recording `cooked`**: the ask rows now assert the message was written
  inside cooked mode.
- A bare `\` with a current word parsed to `cmdReplay` and replayed audio, while
  a bare `?` got its note. Both hatches are now symmetric in both session states,
  with rows for all four.
- `cmdNothing`'s note had three reporting sites with three defaults; M2 would
  have added a fourth. One `nothingSays`.

The pattern across three rounds is worth naming: **every one of my "class" fixes
was itself an instance until the test asserted the positive observable.** Stating
a rule in a comment and enumerating cells in a table are not the same as having
each cell fail on its own.

### 2026-08-23 — M1 round 4: the message nobody read

The gate cleared (round cap reached), and demoted BR-19 past it with a warning
worth more than the pass: *no later gate picks this up.* It was right to say so —
it had found a **shipped bug**.

`define '\'` printed ``type a word after "\\"``, two backslashes. `noteEmptyLiteral`
is a Go **raw** string, so the escape I wrote survived into the output; and
`repl_test.go` asserted `note: noteEmptyLiteral` — the constant compared to
itself, which proves a branch was selected and nothing about what the user reads.
Four rounds had pinned every message's *placement* and none its *text*.

Its grid measured three more behaviours as unasserted, all now covered by
literal-text assertions and each mutation-verified: the note text, the bare-hatch
piped exit code, and the elision in both ask messages (`truncateQuestion(q.text)`
→ `q.text` was green).

Two Minor findings of the same shape, fixed rather than carried:

- **"the ONE place" is an absolute too.** `nothingSays`'s own doc comment and the
  atlas both claimed uniqueness while `replayInPlace` held a byte-identical copy
  of its replay sentence — and that copy is the *live* path for a bare Enter with
  nothing current. Routed through the one function.
- **The parameter I introduced to distinguish two cases was fed a literal at both
  loop sites.** `canReplay: true` where a note-less `cmdNothing` is reachable only
  when nothing IS current (a blank line with a current word parses to
  `cmdReplay`), so it named a condition it could not mean. Renamed to `inSession`,
  which is what actually varies: the one-shot has no loop to press return in.
- The `lost the terminal` report went from two copies to four in this diff, with
  M2's streaming positioned to add a fifth. One `lostTerminal` helper.

Verified on the built binary, not just in tests: `define '\'` and `echo '\' |
define` both print ``type a word after "\"`` and exit 2.

`workshop/lessons.md` gains the rules these four rounds cost, under *"A rule
stated in a comment is not a rule the suite enforces"*.

### 2026-08-23 — M2: the answer

Five tasks in plan order. Two things the plan did not predict, both found by
running the thing rather than by reading it:

- **`ask()` short-circuited the FORCED route.** M1 left the degradation message
  as an early return, so `?why` printed "no model configured" and never reached
  the seam — even with one configured. Every M1 test still passed, because M1 had
  no seam to reach. Found by Task 11's test timing out waiting for a stream that
  was never requested. The forced/unforced distinction decides what can honestly
  be SAID when there is no model, never whether the question is asked.
- **The interrupt swallow belongs in the READER, not the loop.** The first draft
  had the loop reading keys while the answer streamed, which worked and ate
  type-ahead: keys typed during a long answer vanished. `interrupter.Fire` now
  reports whether a scope consumed the interrupt and `readKeys` drops it when one
  did. That made the key channel's buffering load-bearing — the loop stops
  reading during a stream, and on an unbuffered channel the reader blocks on the
  first key typed and never decodes the Ctrl-C behind it.

Verified against the LIVE proxy, not only the fake. With a two-line
`user-model.md` ("B2, reads business news, weak on near-synonym distinctions"),
`sycophantic` then *"what is the difference to obsequious?"* returned an answer
carrying a **"Business-news nuance"** paragraph and **"Related near-synonyms in
your range"**, and quoting the entry back: *"the dictionary definition you looked
up actually contains both"*. The follow-up *"give me two more examples"* resolved
against it — same word, same near-synonym thread. The log held `kind: asked`
records carrying their questions, and no answers.

**Filed rather than fixed: [tools#19](000019-llm-overloaded.md).** Mid-verification
the proxy began answering **HTTP 200 with `{"type":"error","type":"overloaded_error"}`**,
which `classifyStatus` reads as `ErrRequest` — *our* bug, stay loud — where an
overloaded service is what `ErrUnavailable` exists for. So a transient overload
reaches the user as a raw JSON blob and exit 1. The defect is in `internal/llm`'s
taxonomy, which has its own fake and conformance obligations; folding it into this
close would have changed #11's contract without its own review boundary.

### 2026-08-23 — M2 boundary review: REWORK, 16 findings, two Critical

The harshest round of the issue, and it found two REAL BUGS that every test I
had written passed over. Both were mine, and both were the same shape — a
comment asserting the opposite of what the code did:

- **BR-22 (Critical) — "Recently in the deck" carried the twelve OLDEST words.**
  `Deck()` is documented newest-first; I took `lastN`. My own comment beside it
  said *"the newest are the ones worth keeping"*. The section labelled *recent*
  had been sending the words the learner had touched least recently.
- **BR-23 (Critical) — the one-shot put question TEXT into the word field.**
  `&session{current: oneShot.word}` — for an unforced question the word IS the
  whole line, so it reached "## The word on screen" and the event log's `word:`.
  That is exactly the pollution routing-before-capture exists to prevent, arriving
  through a door D2 did not cover.

The meta-finding is the one worth keeping. `test-asserts-nothing` reached its
**5th** occurrence, and the rule that finally names it is sharper than the one I
had been applying:

> An assertion that a value reached an output must use a value **only that
> source can supply**. A fixture shared with another source makes the assertion
> vacuous.

`TestAskStreamsAnAnswerWithTheDirectoryAsContext` asserted `"sycophantic"` for
the session-words list — a string `CurrentWord` also supplies — so deleting
`SessionWords` entirely left it green. Four claims in the diff survived the
mutation they existed to catch, each for that reason. Every context source now
has a fixture only it can supply.

Fixed, all mutation-verified individually:

- Both Criticals, plus tests using deck-only and one-shot-only observables.
- **I1** — `UserModel`'s error was discarded at its only call site, defeating the
  reason `store/yaml.go` returns one. Warned now, like a store that cannot open.
- **I2** — `go test -race` failed with eleven reports: two new tests read a
  `bytes.Buffer` while the code under test wrote it. One `syncBuf`, and
  `ptyOut` folded onto it — the package already owned that shape.
- **I3** — the plan's `askCapturer` was not built; events appended through a
  second write path. `Capturer` gained `CaptureAsk`, so the log has one writer.
- **I4/I5** — README did not say questions are persisted; `atlas` still stated
  the pre-M2 cancellation model in the paragraph that OWNS it while the new
  section sat below. The rule: a corrected fact is swept at its restatements,
  not appended beside them.
- **I6** — the piped loop's ask wiring was entirely unpinned (both the answer
  destination and the session were deletable green).
- **I8** — `assertNoBareNewline` excused a trailing newline, which made it
  unfalsifiable for a single-line message: vacuous for 3 of 3 rows. Replaced by
  `assertCRLFTerminated`, which counts the terminator itself.
- **I9** — 31 unticked plan steps and no `## Revisions` entry for any boundary
  round or M2 departure. Both now recorded, including M1's four rounds.
- Minors: `crlf.Write` reported 0 on a short write, `askInSession`'s error branch
  was unreachable, `keysFor` was dead, `bytesReader` was `strings.NewReader`
  renamed, and `DEFINE_NO_CAPTURE` silently un-adapts answers (documented).

### 2026-08-23 — M2 round 2: the fixes that were right and unpinned

Verdict improved to FIX-THEN-SHIP, both Criticals disposed, five findings open —
and the shape of them is the lesson. Two were marked **not-addressed** on code I
had genuinely fixed:

- **BR-24** — the `UserModel` warn was *"correct and reachable, but no test fails
  without it"*, and `failingStore.UserModel`, which I added FOR it, sat at zero
  call sites. Fixing a finding is not disposing of it; the disposal is the test.
- **BR-30** — three of its four claims were pinned, and the fourth was my own
  test naming a transport it never touched: the "signal transport" row called
  `interrupts.Fire()`, which is the sink itself, so deleting `repl`'s signal
  watcher left it green. It now drives `d.notifySignals`.

Three findings were created by the previous round's fixes — worth stating
plainly, since "I fixed it" was the claim each time:

- **`consumed()` ignored the `out` it was handed** and re-seeded the translation
  from `false`, getting exactly the case `lastWasCR` exists for backwards.
- **`newestFirst` returned oldest-first**, sat in the IO shell, and was reachable
  only through a real store and a wire fake — the shape BR-22's defect hid in.
  Now `recentDeck`, beside `recentTurns`, table-tested.
- **The pty suite ran whatever `bin/define` was on disk**, with no staleness
  check. Measured: the binary was 34 minutes older than the commit under test, so
  a conformance run would have validated pre-fix code and reported ok. It builds
  from the tree now — verified by mutating source and watching it fail with no
  rebuild step.

**BR-39 found a real behavioural gap in a claim I wrote.** README said "Ctrl-C
stops the answer rather than the session"; measured, that was true in ONE of the
two loops. `replLines` passed its own ctx, which the default sink cancels, so an
interrupt during an answer ended the session there. The interrupt sink is
supposed to be the single answer to what Ctrl-C means — a loop that streams an
answer and does not scope it is a loop where the sink is not the answer after
all. Both loops scope it now.

**BR-39 also caught a claim that was true in 1 of 4 cells**: "every question is
recorded". Measured 0 events for an errored ask, an unwired seam, and a cancelled
one. Rather than qualifying the sentence away, the question is now recorded on
every path where a request was actually SENT — a cancelled or refused question is
still what the learner wanted to know — and NOT when no model was ever reached.
Four cells, four assertions.

**BR-40 is the one I would not have thought to write.** `question:` is the first
free-form user text this log has ever held; every value before it was a single
dictionary headword. The reader's record boundary is a literal top-level `- `,
and nothing pinned that user text cannot reach column 0. The reviewer verified
today's behaviour is correct — including a question whose text is a complete
forged event record — and filed it anyway, because a regression there corrupts an
append-only log irreversibly. The conformance suite now round-trips five
adversarial questions against both store implementations.

**BR-38 was refused a third round of one-off tests**, correctly: *"write the table
as one fixture-driven test whose rows ARE the cells"*. `TestTheAskWiringTable` is
{one-shot, piped, editor} × {answer destination, session carried, interrupt
scoped}, and all three previously-green mutations now redden.

### 2026-08-24 — M2 round 3: four findings, all about the previous round's fixes
- 2026-08-24: closed M2 — go test ./... + go vet + go test -race + pty conformance all green. M2 boundary rounds 1-3, every finding fixed, none deferred. Round 3 disposed all five of round 2s and raised four more, each about round 2s own fixes: the line-loop scoping fix had made a SECOND copy of the five-step scope sequence (one askScoped owns it now, and since reversing its order reddens nothing — the window is too narrow to test without a flaky race — defers LIFO enforces the order rather than a comment that cannot fail); countingCapturer.asked was appended at one site and read at zero (now read at the seam, with a wired fake so the question genuinely reaches a model); the UserModel conformance row asserted the only value Mem could produce because the method arrived as a getter with no writer anywhere (Store.SetUserModel closes it, atomically for YAML, and #17 inherits the setter it needs); and the atlas store section plus the plans entity tables are now resolved against enumerations — yaml.go/event.go/storetest against the three normative blocks, and git diff base..HEAD +func/+type against Core concepts — rather than patched at the lines a finding named.; review verdict: FIX-THEN-SHIP

Round 2's five findings all disposed; four new, and every one of them is about
scaffolding or duplication that round 2's own fixes created. That is the pattern
of this whole issue in miniature, and it converged this time because the review
refused to accept a fix that could not fail.

- **BR-46** — my BR-39 fix (scoping the interrupt in the line loop) created a
  SECOND copy of the five-step scope sequence, and only the raw loop's copy
  carried the ordering rationale. One `askScoped` owns it now; the writers and
  the post-answer redraw stay per-loop, since those are what legitimately differ.
  Then, probing it: **reversing the order reddens nothing** — the window where it
  matters is an instant too narrow to test without a flaky race. A rationale no
  test can defend is exactly the scaffolding BR-45 names, so the sequence is
  written with `defer`'s LIFO instead: it can no longer be written in the wrong
  order.
- **BR-45** — two things I added last round *read as protection and could not
  fail*: `countingCapturer.asked`/`askedWord` were appended at one site and read
  at zero (the seam-level assertion now reads them, with a wired fake so the
  question genuinely reaches a model), and the `UserModel` conformance row
  asserted the only value `Mem` could produce, because the method arrived as a
  getter with **no writer anywhere in the tree**. `Store.SetUserModel` closes
  that: a fake that cannot hold the real one's state is the gap `storetest`
  exists to close, and #17 inherits the setter it needs anyway.
- **BR-44** — three claims in the atlas's store section, in the paragraphs that
  OWN them: the artifact block still listed two files of three, "carries every
  field" contradicted the completeness rule this window generalised, and the
  conformance claim was false for the newest method. Swept against `yaml.go`,
  `event.go` and `storetest/suite.go` rather than patched line by line.
- **BR-47** — the Core-concepts tables were fixed by eye last round and drifted
  again. Now resolved mechanically against
  `git diff <boundary>..HEAD -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) '`,
  which lists 29 additions here; `askScoped` and `consumed` went in by the same
  pass rather than waiting to be named by a round 4.

### 2026-08-24 — M2 round 4: the gate passed, and found a bug I would have shipped

FIX-THEN-SHIP with no open blockers; three findings, all fixed before the close
commit per #174. Two of them matter:

- **BR-48 corrected my own reasoning from round 3.** I had probed ONE mutation of
  `askScoped` (reversing the two defers), found it unobservable, and concluded
  the mechanism could not be tested. `askScoped` has FOUR mutations, and the
  enumeration is the deliverable — not the verdict on the one probed. Measured:
  omitting `interrupts.Set` reddens ten tests; **omitting `defer restore()` left
  the entire suite green while making the session unquittable** — the sink keeps
  pointing at the dead question's cancel, so `readKeys` swallows every later
  `\x03` and Ctrl-C does nothing for the rest of the session; omitting
  `defer qcancel()` leaks the question's context; only the reordering is
  genuinely unobservable. Every existing test asserted the sink DURING an answer
  and none asserted it AFTER one. The enumeration now sits in `askScoped`'s doc
  comment so nobody re-derives it.
- **BR-49 — an answer the user READ was dropped when they stopped it.** README
  states two halves in adjacent paragraphs: Ctrl-C stops the answer rather than
  the session, and a follow-up resolves against the answer before it. The cancel
  path returned before `recordExchange`, so the claim was true in three of four
  cells and the false one was the flow this milestone is named after. A stopped
  answer is now recorded — the user read it, which is the same reason a truncated
  one is kept — with a row test over how the answer ended.

Also: `crlfWriter` advanced its `lastWasCR` carry over bytes the underlying
writer never took, so after a short write the retry's first newline was judged
against a carriage return the terminal never saw.

### 2026-08-24 — issue close review: a flaky test, and a measurement I got wrong

Twelve findings disposed, three open. Two of the three are about the round that
was supposed to fix this family, which is the honest summary of this issue.

- **BR-54 — the test I wrote last round is FLAKY**, failing 12 of 30 runs on
  unmutated HEAD. It waited for the last text delta to appear, then sent `\x03`
  and required the scope to already be back. Everything between is unsynchronised:
  the remaining SSE frames parse, `Stream` returns, `recordExchange` runs, and the
  deferred `CaptureAsk` writes to DISK — all inside `askScoped`, before
  `restore()`. Lose that race and it fails printing the message it reserves for
  the real defect, which is worse than no test: it trains you to re-run. **A test
  must synchronise on the state it ASSERTS, not on a proxy that merely precedes
  it.** It now waits for the prompt to be redrawn, which happens strictly after
  `askScoped` returns: 0 of 30 clean, 10 of 10 mutated.
- **BR-55 caught a measurement I reported wrongly.** The comment I added to
  `askScoped` claims `omit defer qcancel() → 1 test red`. There were 0; the "1" I
  read came from the flaky test failing beside it. The cell was UNDEFENDED and I
  had written the opposite into the table whose whole thesis is that the
  enumeration is the deliverable. `TestAskScopedHandsTheScopeBackAndCleansUp`
  now asserts all three observable cells directly against `askScoped` — no
  stream, no timing — including that the question's context is cancelled when it
  returns.
- **BR-47, third time.** The entity enumeration was re-run and again missed what
  the same commit created (`Store.SetUserModel`, `carriedCR`). Re-running the
  command was never the fix: **a mechanical check placed before the last edit is
  a check of a tree nobody shipped.** It runs LAST now, reconciled in full.

One user-visible fix fell out of BR-55: `ErrUnavailable` covers both "no key" and
"configured but did not answer", and the message said *no model configured* for
both — while the question IS recorded in the second case and is NOT in the first,
which is exactly what README keys on. The two now say different things, and the
outcome enumeration gained the cell it was missing.

### 2026-08-24 — close review round 2: the fix from the round before was wrong

Nine disposed, two open, and the first is a defect in the previous round's fix —
which by now is the recognisable rhythm of this issue.

- **BR-56.** I split `ErrUnavailable` into "no model configured" and "the model
  did not answer". It has **three** producers in `internal/llm`'s
  `classifyStatus`: never reached, retryable-and-gave-up, and **401/403 — a wrong
  or expired credential**, which that function's own comment calls *"an operator
  configuration state"*. So the split told the one user who CAN fix their problem
  to wait for it to pass. The message now reports the taxonomy's word and hands
  over the underlying error, which names the cause — the conclusion `--llm-check`
  reaches by not deciding either. **A taxonomy branch must be exhaustive over what
  PRODUCES the error it dispatches on, and that enumeration is greppable in the
  producer rather than guessable in the consumer.**

  The test cell I added for the split **dialled past the wire fake**
  (`127.0.0.1:1`), so it exercised only the never-reached arm and could not touch
  the states the fake models (ARCH-MOCK). A row per producer now, through the
  fake.
- **BR-47, fourth time.** The command was never wrong. Two things were: "last"
  has to mean against the **working tree** — run against `..HEAD` it lists what
  earlier commits added, and it showed me a function this change had renamed —
  and the entry must carry **no count**, because a count is a measured claim that
  drifts silently. Same rule as BR-55's mutation table and BR-56's message: name
  the check, not the number it produced once.

Also swept, and it is the sharpest instance of the family in the issue: my own
test comment still asserted *"omitting restore leaves the whole suite GREEN"* —
a measured fact that the very fix beneath it had falsified, left standing because
nobody re-ran it.

### 2026-08-24 — close review round 3: docs only

Nine disposed, two open, both documentation.

- **BR-47, fifth round.** The timing fix from last round was right and not
  enough: I ran the enumeration against the working tree and then *read* its
  output against the tables by eye, missing three symbols — one (`ask`) present
  since M1. Reconciliation is now an actual **set difference** with word
  boundaries (a substring test reports `ask` as present because `askContext`
  contains it), run to empty before the commit. The check, not its output, is
  what the plan records.
- **BR-58** is the one worth keeping. The atlas said *"the FOUR exit-code
  absolutes"*, and this window's own message split added exit sites without
  touching it — while **the same commit**, twenty lines away, wrote *"the entry
  must not carry a count: a count is a measured claim that drifts the moment
  anything is added, and it drifts silently."* Stating a rule and breaking it in
  the same change is the most compact version of this issue's whole pattern.

Both fixes are documentation. No code, no tests, no new symbols — so the
enumeration is empty by construction rather than by inspection.
