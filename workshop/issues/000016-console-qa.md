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
- [ ] A full Q&A round trip runs against the fake, with recent words and
      `user-model.md` visible in the recorded prompt.
- [ ] A follow-up question resolves against the previous exchange.
- [ ] Ctrl-C mid-stream returns to the prompt with the session intact.
- [ ] With the seam unavailable, a question produces the explanatory message and a
      lookup still works.
- [ ] Driven through the raw TUI loop, not only the piped loop — a wiring only a
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
- [ ] M2 — the answer: `askContext` + pure prompt, `runAsk` streaming through
      `internal/llm`, `Store.UserModel`, the `asked` event, scoped Ctrl-C.

## Log

### 2026-08-22

Created from the operator conversation that broadened `define-learn` to an adaptive
program. See the project file's `## Log` scope event for the surrounding decisions.

### 2026-08-23

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
