# Boundary Review — tools#16 (milestone M1)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aab71caeb1303528c1cd8db9641e83402612a4ed^..bb2c98bac048709deb7ddce0ff99289161032041 |
| command | sdlc milestone-close --issue 16 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-23T15:46:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The classification design is the strongest part of this milestone and it holds up under scrutiny: the dictionary-as-classifier containment is real, `parseREPLLine`/`readsAsQuestion` are genuinely pure, and `TestConsoleDecisionTable` drives the actual production route rather than the two halves separately. I mutation-checked six of the diff's load-bearing guards and five went red as claimed (`out.ask == "" && out.code == 0`, `!cmd.literal`, capture-after-route, the `cmdAsk` usage exemption, the unforced editor-loop branch). Two did not, and that is what blocks a clean SHIP: **the entire forced-`?` branch can be deleted from both REPL loops with the full suite still green**, and `TestAQuestionIsRecalledByUpArrow` passes with history recording removed outright — so two rows of Task 4's contract table, and half of the issue's ticked "each hatch is exercised by a test that fails when the hatch is removed", are claims no test can falsify. None of the findings are correctness bugs in shipped behavior; all are cheap.

## 1. Strengths

- **`lookupAndRender` (cmd/define/main.go:392-431) routes before capture, and it is pinned.** Moving `d.capture.Capture(word, false, opt)` above the route decision reddens `TestAQuestionIsNotCaptured` immediately. D2 is enforced, not documented.
- **`TestConsoleDecisionTable` (cmd/define/route_test.go:16) is the right test.** `routeFor` is a harness that decides nothing itself — it asks `parseREPLLine` then `lookupAndRender` — so the table it asserts is the one the console actually implements. Removing `!cmd.literal` reddens exactly the `\how so` row.
- **The ask-outcome guard in `submitLine` (cmd/define/replraw.go:296).** `out.ask == "" && out.code == 0` is the subtle one, and deleting the `ask` half turns `TestAQuestionDoesNotBecomeTheCurrentWord` red with a precise message. This was PQ-3 caught on paper and it is genuinely wired.
- **`session` (cmd/define/session.go) earns its existence.** Three declarations of "what is this session holding" collapsed to one, which is what makes "an ask touches none of this" a single assignment site rather than a rule repeated three times. Clean ARCH-DRY win.
- **`openerStem` cuts `n't` as a unit before the apostrophe split**, with `when is it used` as the fixture that pins why (cmd/define/question.go:70-80). That is the failure mode a lazier rule would have shipped.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `TestAQuestionIsRecalledByUpArrow` cannot fail (cmd/define/askroute_test.go:95).**
The assertion is `strings.Contains(out.String(), aQuestion)`, but the raw editor re-renders the whole line on every keystroke, so the full question is in stdout from typing alone — before Enter, before Up, before history is consulted. Verified: replacing `hist.Add(line)` at cmd/define/replraw.go:292 with a no-op leaves this test green. The Task-4 contract row "**does** add the line to editor recall (`hist.Add`)" is therefore unpinned.
*Fix sketch:* inject `d.history = &memHistory{}`, run, and assert the recorded lines contain the question — that pins `hist.Add`. To also pin the recall rendering, drive a second run without the `KeyUp` and assert the question appears strictly more times with it.

**I-2 — the forced `?` route is unreachable by any test in either loop (cmd/define/repl.go:219, cmd/define/replraw.go:199-211).**
Verified: deleting `case cmdAsk: ask(cmd.question)` from `replLines` *and* the entire `if cmd.kind == cmdAsk { … }` block from `runEditor` leaves `go test ./cmd/define/` fully green. Every loop-level ask test (`TestLineLoopRoutesAQuestion`, `TestEditorLoopRoutesAQuestion`, `TestAQuestion*`) uses the *unforced* route; `routeFor` returns `"question"` for `cmdAsk` without entering a loop at all; only the one-shot has a forced test. So `hist.Add(submitted.String())` at replraw.go:204 is also unpinned (deleting it alone is green). Without the branch, `?why` at the raw prompt would silently fall through to `replayInPlace`, and in a piped run would do nothing at all — and the suite would not notice. This is lessons.md define #15's rule ("a wiring only a loop shell supplies must be pinned by a test that drives that loop shell") applied to the branch the milestone is named after.
*Fix sketch:* two tests mirroring the existing ones with `"?why\r"` / `"?why\n"` — I confirmed both loops behave correctly today, so they will pass on the first run and go red the moment the branch is touched. Note the issue's Done-when row "Both escape hatches work, and each is exercised by a test that fails when the hatch is removed" is currently ticked on parser-level coverage only.

**I-3 — README.md not updated for the new user-facing surface (README.md:120-131).**
The diff touches `atlas/define.md`, `fs.Usage` and `/help`, but not README, which is where this repo documents exactly this class: the `/`-in-column-1 convention, the interactive key table (README.md:43-50), and the exit-code contract (README.md:111-116). The `?` and `\` hatches, "a question is answered by the model rather than looked up", and the new "question with no model configured" → exit 1 case are all things a reader types or scripts against, and none of them appear there. Per AGENTS.md §8 / the docs gate, this is the gap that otherwise surfaces at the merge-time `specs` judge.

**I-4 — the one-shot sends `cmdNothing`/`cmdReplay` to the dictionary as an empty word (cmd/define/main.go:342).**
`run`'s default branch handles `cmdCommand` and `cmdAsk` explicitly and falls through to `defineOnce(…, oneShot, …)` for everything else — but `parseREPLLine` now returns `cmdNothing` (with `word == ""`) for two inputs it never used to produce: `?` and `\`. Measured: `define "?"` and `define "\"` both print `define: : no dictionary entry`, exit 1, and call `d.capture.Capture("", false, opt)`, which appends a `ReviewEvent{Word: ""}` that `complete()` (cmd/define/store/event.go:30) then discards at read time — a junk record in the append-only log, indistinguishable from a torn one. `cmd.note` is dropped entirely, so the helpful `type a question after "?"` never reaches the one-shot. Pre-#16 this input looked up the literal `"?"`; the empty-word form is new.
*Fix sketch:* in the `default:` arm, handle `cmd.kind == cmdNothing` before `defineOnce` — print `cmd.note` when set, otherwise the existing "type a word" diagnostic — and never hand `lookupAndRender` an empty word.

## 4. Minor findings

- cmd/define/replraw.go:215 — the bare-`?` note is written without `eraseLine`, unlike its sibling at replraw.go:264 (`replayInPlace`); on a real terminal it renders as `› ?define: type a question after "?"` appended to the line the user typed.
- cmd/define/replraw.go:227 + :204 — forced and unforced asks render with different vertical spacing: the unforced path emits the pre-`submitLine` `\r\n` *plus* `askInSession`'s own, so the message sits one blank line lower than the forced one. Two routes the plan wanted visually identical.
- atlas/define.md:488 — "Single-word lines are never questions" contradicts the code: the trailing-`?` arm precedes the `len(fields) < 2` check, so `why?` and `sycophanti?` both classify as questions. Either qualify the sentence ("single-word lines *without a question mark*") or move the arm.
- cmd/define/question.go:83 — hand-rolled `contains()` where `slices.Contains` exists; the module is go 1.26 and the package's tests already import `slices` (ARCH-DRY, stdlib reuse).

## 5. Test coverage notes

Mutation results, run against a scratch copy:

| mutation | result |
|---|---|
| `out.ask == "" && out.code == 0` → `out.code == 0` | RED — `TestAQuestionDoesNotBecomeTheCurrentWord` |
| drop `!cmd.literal` | RED — decision-table `\how so` row |
| capture before the route decision | RED — `TestAQuestionIsNotCaptured` |
| drop `oneShot.kind != cmdAsk` from the usage guard | RED — `TestOneShotForcedQuestionIsNotAUsageError` |
| disable the unforced ask branch in `runEditor` | RED — `TestEditorLoopRoutesAQuestion` |
| **delete both loops' `cmdAsk` branches** | **GREEN** — I-2 |
| **drop `hist.Add(line)` in `submitLine`** | **GREEN** — I-1 |

`TestHelpNamesBothHatches` does pin its claim (pre-existing `/help` output contains none of `?`, `\`, `ask`), though matching bare `"?"` is loose enough that any future summary containing a question mark would mask a regression — matching the literal hatch line would be tighter. `question_test.go`'s table is pure with no fake, and its comment correctly records that every row is a line NOAD already missed. `TestTruncateQuestion`'s UTF-8 row is real coverage, not decoration.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass**, one Minor. `askUnavailable` is the single degradation message; `parseREPLLine` is the single decision table both loops and the one-shot enter; `lookupAndRender` remains the single capture site; `session` collapsed three copies of one idea. The two per-loop closures (`ask`, `askInSession`) are justified — they differ in terminal mode — and both funnel to one function. Only `contains` vs `slices.Contains` is flagged.
- **ARCH-PURE — pass.** `readsAsQuestion`, `parseREPLLine`, `truncateQuestion` and `session` are deterministic and unit-tested with no IO, no exec, no fs. `route_test.go` runs against the committed fixture corpus through `loadFakeDictionary`, not a stubbed `Lookup`, so the "pure" entities are never propped up by mocks.
- **ARCH-PURPOSE — flagged (I-2, I-3).** M1's stated purpose is "the console knows a question from a word … both hatches, in all three entry modes." The routing is delivered in all three; the `?` hatch's *loop* behavior is delivered but not verified, and the README half of the docs obligation is deferred. The class here, not just the instance: any branch reachable only through a loop shell needs a loop-shell test — that enumeration is `{replLines, runEditor} × {forced, unforced}`, and three of four cells are currently empty on the forced side.
- **ARCH-MOCK — pass for M1.** No external binary or service is introduced; the dictionary stays behind `d.dict` with a committed-corpus fake, and production and test share that boundary. M2 brings the real surface (`llm.Client.Stream`), where the plan already names `llmtest`'s wire-level fake — hold the line that tests hit the httptest server rather than a stubbed `Client`.
- **M2 watch item — `-raw` and the ask fallback.** README documents `-raw` as the scripting form that "records nothing, because it is for scripts", but `lookupAndRender` currently routes a `-raw` miss to the ask path on the same terms as an interactive one. Today that costs a different message; in M2 it costs a network call for a line a script piped in. Suppressing the fallback under `opt.raw`, decided next to the `literal` flag at main.go:404, keeps the scripting contract intact and is one condition.
- **M2 watch item — three ask entry points, one `runAsk`.** `askUnavailable` is reached from `run` (main.go:337), `replLines`' `ask` closure, and `runEditor`'s `askInSession`. When M2 replaces it, `runAsk` must stay the single entry the way `askUnavailable` is now — the interrupter and `crlfWriter` belong only on the raw closure, but the resolve/gather/stream/record sequence must not fork.

## 7. Plan revision recommendations

One `## Revisions` entry in `workshop/plans/000016-console-qa-plan.md`, if I-2 is deferred rather than fixed now:

> ### 2026-08-23 — M1 boundary review: the forced route shipped without loop coverage
>
> **Reason:** M1 boundary review, finding I-2. Task 4's Step-1 test list names one test per entry mode, and all three were written against the *unforced* route; the `?`-forced `cmdAsk` branch in `replLines` and `runEditor` can be deleted with the suite green.
>
> **Delta:** Task 4's test list gains two rows — `TestLineLoopRoutesAForcedQuestion` and `TestEditorLoopRoutesAForcedQuestion` — and states the enumeration the rule implies: `{replLines, runEditor} × {forced, unforced}`, four cells, each pinned. Task 11's `TestForcedAndUnforcedAsksShareOneWiring` then verifies the *shared wiring* rather than being the first test to enter the forced branch at all. Task 4's contract row "does add the line to editor recall" gains its assertion target: the injected `History`, not stdout, which the editor's per-keystroke echo already satisfies.

No contradiction between the Core-concepts table and the code at this boundary: `readsAsQuestion`, `truncateQuestion`, `parseREPLLine`/`replCommand` and `session` all exist at their stated paths with the stated status, and every row absent from the tree (`askContext`, `renderAskPrompt`, `recentTurns`, `crlfWriter`, `exchange`, `ReviewEvent`/`complete`, `gatherAskContext`, `runAsk`, `Store.UserModel`, `interrupter`) is explicitly M2's.

```findings
findings:
  - id: new
    severity: Important
    family: loop-shell-branch-untested
    title: |
      deleting both loops' forced-"?" cmdAsk branches leaves the whole suite green
    detail: |
      Verified by mutation: removing `case cmdAsk:` from replLines (repl.go:219)
      and the whole `if cmd.kind == cmdAsk` block from runEditor
      (replraw.go:199-211) passes `go test ./cmd/define/` in full. Every
      loop-level ask test uses the unforced route; routeFor returns "question"
      for cmdAsk without entering a loop. hist.Add(submitted.String()) at
      replraw.go:204 is unpinned by the same gap. The issue's ticked Done-when
      row claims each hatch is exercised by a test that fails when the hatch is
      removed; that holds at the parser only. Add one forced test per loop —
      both pass against today's code, which is the point.
  - id: new
    severity: Important
    family: test-asserts-nothing
    title: |
      TestAQuestionIsRecalledByUpArrow passes with history recording removed
    detail: |
      askroute_test.go:95 asserts stdout contains the question, but the raw
      editor re-renders the line on every keystroke, so the question is in
      stdout from typing alone. Verified: replacing hist.Add(line) at
      replraw.go:292 with a no-op leaves the test green. Task 4's contract row
      "does add the line to editor recall (hist.Add)" has no falsifiable test.
      Assert against an injected History, and pin the recall rendering by
      comparing runs with and without the KeyUp.
  - id: new
    severity: Important
    family: readme-surface-gate
    title: |
      README update appears missing for the "?" and "\" hatches and question routing
    detail: |
      README.md documents exactly this class — the "/"-in-column-1 convention
      (README.md:120), the interactive key table (:43-50), and the exit-code
      contract (:111-116) — and is untouched in the window, while atlas,
      fs.Usage and /help were all updated. New user-facing surface a reader
      types: "?" forces a question, "\" forces a lookup, an unrecognised line
      that reads as a question goes to the model, and a question with no model
      configured exits 1.
  - id: new
    severity: Important
    family: oneshot-kind-coverage
    title: |
      the one-shot sends cmdNothing to the dictionary as an empty word
    detail: |
      main.go:342's default arm handles cmdCommand and cmdAsk and falls through
      to defineOnce for every other kind, but parseREPLLine now returns
      cmdNothing with word=="" for two inputs it never used to produce. Measured:
      `define "?"` and `define "\"` print `define: : no dictionary entry`, exit
      1, and Capture("", false, opt) appends a ReviewEvent{Word: ""} that
      complete() (store/event.go:30) discards at read time — junk in the
      append-only log #8 and #17 fold over. cmd.note is dropped, so the one-shot
      never shows `type a question after "?"`. Handle cmdNothing before
      defineOnce and never pass an empty word to lookupAndRender.
  - id: new
    severity: Minor
    family: raw-mode-message-placement
    title: |
      the bare-"?" note is printed without eraseLine, unlike its sibling branch
    detail: |
      replraw.go:215 writes `define: %s\r\n` with the cursor still on the typed
      prompt line, so it renders as `› ?define: type a question after "?"`.
      replayInPlace (replraw.go:264) prefixes eraseLine for the same class of
      message.
  - id: new
    severity: Minor
    family: raw-mode-message-placement
    title: |
      forced and unforced asks render with different vertical spacing
    detail: |
      The unforced path emits runEditor's pre-submitLine "\r\n" (replraw.go:227)
      in addition to askInSession's own, so its message sits one blank line
      lower than the forced path's. Two routes the plan wanted visually
      identical.
  - id: new
    severity: Minor
    family: doc-overstates-code
    title: |
      atlas says single-word lines are never questions; the trailing-"?" arm says otherwise
    detail: |
      atlas/define.md:488. The "?" arm precedes the len(fields) < 2 check, so
      readsAsQuestion("why?") and readsAsQuestion("sycophanti?") both return
      true (measured). Qualify the sentence or move the arm.
  - id: new
    severity: Minor
    family: stdlib-reuse
    title: |
      hand-rolled contains() where slices.Contains exists
    detail: |
      question.go:83. The module is go 1.26 and the package's tests already
      import slices (ARCH-DRY).
```
