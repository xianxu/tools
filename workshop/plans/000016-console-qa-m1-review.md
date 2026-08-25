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

---

## Re-review — 2026-08-23T16:05:39-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aab71caeb1303528c1cd8db9641e83402612a4ed^..2378c33746c5df6016002aed11f71620107eb55e |
| command | sdlc milestone-close --issue 16 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-23T16:05:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All eight round-1 findings are genuinely disposed, and I confirmed the two that mattered by mutation rather than by reading the diff: deleting either loop's forced-`?` branch now reddens `TestLineLoopRoutesAForcedQuestion`/`TestEditorLoopRoutesAForcedQuestion` (BR-1), and removing `hist.Add` at either of its two sites reddens the matching subtest of `TestAQuestionIsRecalledByUpArrow` (BR-1/BR-2). BR-4's exhaustive one-shot dispatch is real and pinned; BR-5 and BR-6 I verified against the actual byte stream (the note now carries `eraseLine`; forced and unforced asks both emit exactly `\r\n\r\n` before `draw()`). What keeps this off a clean SHIP is that the round's own new surface repeats two of the families it just closed: the `-raw` rule the round pulled forward from M2 was applied to the unforced route only — `define -raw "?x"`, `echo "?x" | define -raw` and the raw editor under `-raw` all reach the ask path, so README's flat "`-raw` never asks" is false at 3 of the 6 ask entry points; and BR-4's own test injects a `countingCapturer` that is never installed, so the half of BR-4 that was about junk in the append-only log asserts nothing. Both are cheap. `go vet ./...` and `go test ./...` are green.

## 1. Strengths

- **The forced-branch tests do exactly what round 1 asked and nothing more.** `askroute_test.go:38` and `:52` use `?why`, where `why` *is* a headword — so only the hatch can make it a question, which makes them tests of the branch rather than of the classifier. I deleted `case cmdAsk:` from `repl.go:219` and the whole `if cmd.kind == cmdAsk` block from `replraw.go:202-215` and got exactly the two intended failures.
- **`TestAQuestionIsRecalledByUpArrow` (askroute_test.go:145) now asserts against the injected `History`,** and the forced/unforced split means each of the two `hist.Add` sites has its own red. Reverting `replraw.go:207` reddens `forced`; reverting `replraw.go:299` reddens `unforced`. This is the fix the round-1 note said it was.
- **BR-4 was fixed exhaustively rather than by special-casing `?`.** `main.go:346`'s `if oneShot.kind != cmdDefine` closes the whole class — `define " "` and any future kind land there too — and removing it reddens `TestOneShotRejectsAHatchWithNothingAfterIt` with both the exit code and the empty-word-to-dictionary assertions.
- **BR-6's fix is the right shape.** Moving the leading `\r\n` out of `askInSession` and into each caller (`replraw.go:129-132`) is what makes one shared closure actually mean one rendering; I measured both routes producing an identical `…\r\n\r\n` + redraw tail.
- **`readsAsQuestion` stayed pure and its containment argument survived the round** — `route_test.go` drives the real `parseREPLLine` → `lookupAndRender` path against the committed corpus through `loadFakeDictionary`, so the "PURE" entities are never propped up by a stubbed `Lookup`.

## 2. Critical findings

None.

## 3. Important findings

**N-1 — `-raw` still asks on the forced route, in all three entry modes (cmd/define/main.go:336, cmd/define/repl.go:219, cmd/define/replraw.go:202).**
**This is the 2nd finding in family `doc-overstates-code`.** Round 1 fixed the instance (the atlas sentence about single-word lines). Do not fix this instance either — state the rule and sweep it.

The rule: **a claim stated as an absolute in README/atlas or in a test's name must name the enumeration it quantifies over, and every cell of that enumeration must be guarded and asserted.** `-raw never asks` quantifies over `{forced, unforced} × {one-shot, piped loop, raw editor}` — six cells. The `!opt.raw` guard was placed at `main.go:425`, which is on the unforced path only, so the three forced cells are unguarded. Measured with the built binary and with `runEditor`/`replLines` directly:

```
define -raw "?what is X"          → define: no model configured; `what is X` is not a word   (exit 1)
echo "?what is X" | define -raw   → define: no model configured; `what is X` is not a word   (exit 1)
runEditor(opt.raw=true, "?why\r") → define: no model configured; `why` is not a word
```

`TestRawNeverAsks` (askroute_test.go:202) names the class and pins one cell; it stays green with all three above. README.md:79 states the absolute (`-raw never asks — it is the scripting form, so a miss stays a miss`), and the code comment at main.go:419-424 names the cost this leaves open: in M2 a network call for a line a script piped in. Same family, same round, smaller prevalence: README.md:69 and :76 quote the miss message as `` `not found` `` where the program prints `no dictionary entry`. Fix: decide once whether an explicit `?` overrides `-raw`, put that predicate in one place both the forced and unforced dispatches consult, and add the missing rows.

**N-2 — the `countingCapturer` in `TestOneShotRejectsAHatchWithNothingAfterIt` is never installed, so BR-4's log-junk assertion is dead (cmd/define/askroute_test.go:176-194).**
**This is the 2nd finding in family `test-asserts-nothing`.** Round 1 fixed the instance (`TestAQuestionIsRecalledByUpArrow` asserting on an echoing stdout). Do not fix this instance — state the rule.

The rule: **a test that injects a double must assert the injection took effect, or inject at the seam production actually reads.** Here the test sets `d.newStore` (askroute_test.go:178) on a `deps` that `testDeps` already gave a non-nil `capture` (main_test.go:14), and `withStore` only fills nils (`main.go:96`: `if d.capture == nil`), so `cap` is discarded and `cap.calls` is always empty — the `for _, w := range cap.calls` loop never iterates. Verified by probe: panicking inside `countingCapturer.Capture` leaves this test green while `TestAQuestionIsNotCaptured` (which sets `rig.deps.capture` directly) panics immediately; and under the BR-4 mutation the exit-code and dictionary assertions fire while the capture one stays silent. Measured prevalence: 1 of 1 `d.newStore` injection in the package's tests is dead. So the specific harm BR-4 named — `ReviewEvent{Word: ""}` that `store/event.go`'s `complete()` discards at read time — has no falsifiable test. Cheapest fix: set `d.capture = cap` directly, as `TestAQuestionIsNotCaptured` does; the durable fix is a guard in the injection helper (or a `withStore` that prefers an explicitly supplied `newStore`) so a discarded double is a failure rather than a silence.

## 4. Minor findings

- **This is the 3rd finding in family `raw-mode-message-placement`.** Neither BR-5's nor BR-6's fix is pinned: with `eraseLine` removed from replraw.go:223 *and* a second `\r\n` re-added to `askInSession`, `go test ./cmd/define/` is fully green. Do not re-fix the sites — state the rule: **every message class the raw loop writes gets an assertion on the emitted byte stream**, the way `TestEditorLoopUsesCarriageReturnsInRawMode` (editorloop_test.go:143) already does for the definition path. Measured prevalence: 3 of 3 raw-mode message placements introduced by #16 (the bare-`?` note, the forced ask, the unforced ask) are unasserted, which is why the family recurs.
- cmd/define/replraw.go:299 vs :207 — the `\` hatch is dropped from editor recall while `?` is kept. Measured: typing `\how so` records `how so` in `History` (submitLine adds `cmd.word`, post-strip), so Up-arrow + Enter now *asks* the line you had just forced to a lookup; `?why` records `?why` and stays forced. Family `forced-route-enumeration` (new): a behaviour specified for the unforced route is silently inherited by the forced one.
- cmd/define/ask.go:18 — same family: `askUnavailable` tells a forced question about a real headword that it "is not a word" (`?why` → ``no model configured; `why` is not a word``). The sentence was written for the unforced route, where it is true by construction.
- workshop/plans/000016-console-qa-plan.md — 0 of 60 step checkboxes ticked while Tasks 1–6 are complete and committed. #14 and #15's archived plans are fully ticked; #11's are not, so the convention is soft, but leaving M1's boxes open makes "where does M2 resume" ambiguous.

## 5. Test coverage notes

Mutation results, all run in a scratch copy of the tree (the working tree was not touched):

| mutation | result |
|---|---|
| delete `case cmdAsk:` from `replLines` | RED — `TestLineLoopRoutesAForcedQuestion` |
| delete the `cmdAsk` block from `runEditor` | RED — `TestEditorLoopRoutesAForcedQuestion`, `…IsRecalledByUpArrow/forced` |
| `hist.Add(line)` → no-op in `submitLine` | RED — `…IsRecalledByUpArrow/unforced`, `TestEditorLoopFeedsHistory` |
| delete the `oneShot.kind != cmdDefine` guard | RED — `TestOneShotRejectsAHatchWithNothingAfterIt` (both rows) |
| delete the `?` arm from `parseREPLLine` | RED — 6 tests / 10 subtests across parser, both loops, one-shot and the decision table |
| drop `!opt.raw` from `lookupAndRender` | RED — `TestRawNeverAsks` |
| drop `eraseLine` from the bare-`?` note (BR-5) | **GREEN** |
| re-add `askInSession`'s own `\r\n` (BR-6) | **GREEN** |
| panic inside `countingCapturer.Capture` | **GREEN** for `TestOneShotRejectsAHatchWithNothingAfterIt` (N-2) |

Both hatches are now pinned by tests that fail when the hatch is removed, so the issue's second Done-when row is honestly ticked — at the parser *and* at both loop shells, which is what round 1 said it was not. The gap that remains is the one N-1 and the family finding describe: the enumerations `{forced, unforced} × {entry mode}` for `-raw`, and raw-mode message placement, have no rows at all.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one note.** `askUnavailable` is the single degradation message; `parseREPLLine` is the single decision table; `lookupAndRender` is still the single capture site; `session` holds what three variables used to. The note: BR-4's fix gave `cmdNothing` a *third* reporting site (main.go:347, repl.go:194, replraw.go:219), each formatting `define: %s` with its own line ending and default text. Three is where this repo's own history says a rule starts being believed once and written thrice — worth collapsing before M2 adds a fourth caller.
- **ARCH-PURE — pass.** `readsAsQuestion`, `openerStem`, `truncateQuestion`, `parseREPLLine` and `session` are deterministic, and their tests run with no fake at all. `route_test.go` goes through the real dictionary seam against a committed corpus rather than a stub. Nothing "pure" here needs a mock to run.
- **ARCH-PURPOSE — flagged (N-1).** M1's purpose — the console knows a question from a word, both hatches, all three entry modes — is delivered and now verified. The flag is on the round's *own* additions: `-raw never asks` was named as a class in the commit that added it and swept at one site. That is the same instance-not-class shape as BR-1, one round later, which is what makes it worth stating as a rule rather than patching.
- **ARCH-MOCK — pass for M1.** No new external binary or service. The dictionary stays behind `d.dict` with a committed-corpus fake plus `dict_conformance_test.go` for live drift; production and test share the boundary. M2 introduces `llm.Client.Stream`, where the plan already names `llmtest`'s wire-level httptest fake — hold the line that tests hit the server, not a stubbed `Client`.
- **M2 watch item — the ask entry points are now six, not three.** `askUnavailable` is reached from `run` (main.go:337 and :362), `replLines`' `ask` closure (both arms), and `runEditor`'s `askInSession` (both arms). When `runAsk` replaces it, that enumeration is the thing to wire once — and it is the same enumeration N-1 says is currently half-guarded, so fixing N-1 now buys M2 the table it needs.
- **M2 watch item — `session.entry` is written and read nowhere.** `sawLookup` sets it (session.go:27) for `askContext.CurrentEntry`; nothing consumes it until Task 9. Fine as declared intent, but it is exactly the "field set, never read" shape that reads as protection — worth a consuming assertion the moment M2 lands rather than after.

## 7. Plan revision recommendations

One `## Revisions` entry in `workshop/plans/000016-console-qa-plan.md`, because the plan no longer describes what the code does:

> ### 2026-08-23 — M1 boundary review round 1: `-raw` never asks, pulled forward from M2
>
> **Reason:** M1 boundary review round 1 took the reviewer's M2 watch item and implemented it in M1. D3/D4 and Task 3 describe the miss-branch route condition as `!literal && readsAsQuestion(word)`; the code is `!cmd.literal && !opt.raw && readsAsQuestion(word)` (`main.go:425`).
>
> **Delta:** D4 gains a fourth suppression alongside `\`: `-raw` is the scripting form and a miss under it stays a miss. Task 3's snippet is updated to match. The rule the condition implies is written out as an enumeration — `{forced, unforced} × {one-shot, piped loop, raw editor}`, six cells — with the note that as shipped only the three unforced cells are guarded, and that `TestRawNeverAsks` covers one of them (round-1 finding N-1). Task 11's `TestForcedAndUnforcedAsksShareOneWiring` inherits that table.

Also worth a second entry if the `\`-in-recall behaviour is left as-is: Task 4's contract table says "does add the line to editor recall (`hist.Add`)" without saying *what* is added, and the two hatches now differ (`?why` recalls forced, `\how so` recalls unforced). Either state the intended asymmetry in the table or make both record `submitted.String()`.

No contradiction between the plan's Core-concepts table and the tree at this boundary: `readsAsQuestion`, `truncateQuestion`, `parseREPLLine`/`replCommand` and `session` exist at their stated paths with their stated status, and every absent row (`askContext`, `renderAskPrompt`, `recentTurns`, `exchange`, `crlfWriter`, `ReviewEvent`/`complete`, `gatherAskContext`, `runAsk`, `Store.UserModel`, `interrupter`, `deps.notifySignals`) is explicitly M2's.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: deleting either loop's cmdAsk branch now reddens the matching forced test; hist.Add(submitted) is pinned too.
  - id: BR-2
    disposition: addressed
    note: |
      Asserted against an injected memHistory; reverting hist.Add(line) in submitLine reddens the unforced subtest.
  - id: BR-3
    disposition: addressed
    note: |
      README documents both hatches, routing, the model requirement and the exit codes; two of its claims are inexact — see the new doc-overstates-code finding.
  - id: BR-4
    disposition: addressed
    note: |
      Exhaustive dispatch at main.go:346; removing it reddens the test. The test's capture half is dead, raised separately under test-asserts-nothing.
  - id: BR-5
    disposition: addressed
    note: |
      eraseLine present at replraw.go:223 and measured on the byte stream; unpinned by any test, folded into the family finding.
  - id: BR-6
    disposition: addressed
    note: |
      Measured: forced and unforced asks both emit exactly one leading CRLF plus askInSession's, identical tails; unpinned, folded into the family finding.
  - id: BR-7
    disposition: addressed
    note: |
      atlas/define.md now qualifies the sentence and names "why?" explicitly.
  - id: BR-8
    disposition: addressed
    note: |
      question.go:62 uses slices.Contains; the hand-rolled contains() is gone.
findings:
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      "-raw never asks" is guarded on the unforced route only, so 3 of 6 ask entry points ignore it
    detail: |
      This is the 2nd finding in family doc-overstates-code, so the deliverable
      is the rule, not the site. Rule - an absolute stated in README/atlas or in
      a test name must name the enumeration it quantifies over, and every cell
      must be guarded and asserted. Here the enumeration is
      {forced, unforced} x {one-shot, piped loop, raw editor}; the !opt.raw
      guard sits at main.go:425 on the unforced path only. Measured -
      `define -raw "?what is X"` and `echo "?what is X" | define -raw` both
      print the no-model message and exit 1, and runEditor with opt.raw=true on
      "?why" does the same, while TestRawNeverAsks (askroute_test.go:202) stays
      green through all three. README.md:79 states the absolute; main.go:419-424
      names the M2 cost (a network call for a line a script piped in). Same
      family, smaller prevalence - README.md:69 and :76 quote the miss message
      as `not found` where the program prints `no dictionary entry`.
  - id: new
    severity: Important
    family: test-asserts-nothing
    title: |
      BR-4's test injects a countingCapturer that withStore discards, so its log-junk assertion is dead
    detail: |
      This is the 2nd finding in family test-asserts-nothing, so state the rule -
      a test that injects a double must assert the injection took effect, or
      inject at the seam production reads. askroute_test.go:178 sets d.newStore
      on a deps whose capture testDeps already filled (main_test.go:14), and
      withStore only fills nils (main.go:96), so cap is discarded and
      `for _, w := range cap.calls` never iterates. Probe-verified - a panic
      inside countingCapturer.Capture leaves TestOneShotRejectsAHatchWithNothingAfterIt
      green while TestAQuestionIsNotCaptured panics at once. So the harm BR-4
      actually named, a ReviewEvent with an empty Word that complete() discards
      at read time, has no falsifiable test. Measured prevalence - 1 of 1
      d.newStore injection in the package is dead.
  - id: new
    severity: Minor
    family: raw-mode-message-placement
    title: |
      neither BR-5's nor BR-6's fix is pinned - both can be reverted with the suite green
    detail: |
      This is the 3rd finding in family raw-mode-message-placement. Do not
      re-fix the sites. Rule - every message class the raw loop writes gets an
      assertion on the emitted byte stream, as TestEditorLoopUsesCarriageReturnsInRawMode
      (editorloop_test.go:143) already does for the definition path. Verified -
      removing eraseLine from replraw.go:223 and re-adding askInSession's own
      "\r\n" together leave go test ./cmd/define/ fully green. Measured
      prevalence - 3 of 3 raw-mode message placements introduced by this issue
      (the bare-"?" note, the forced ask, the unforced ask) are unasserted,
      which is why the family recurs.
  - id: new
    severity: Minor
    family: forced-route-enumeration
    title: |
      the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
    detail: |
      Two instances of one rule - a behaviour specified for the unforced route
      is silently inherited by the forced one. (a) submitLine adds cmd.word
      (replraw.go:299), which is post-strip, so typing `\how so` records
      "how so"; Up-arrow then Enter asks the line you had just forced to a
      lookup. The cmdAsk branch records submitted.String() (replraw.go:207), so
      "?why" stays forced. Measured through runEditor with an injected
      memHistory. (b) askUnavailable (ask.go:18) tells a forced question about a
      real headword that it "is not a word" - "?why" prints
      "no model configured; `why` is not a word". The sentence is true by
      construction only on the unforced route.
  - id: new
    severity: Minor
    family: plan-bookkeeping
    title: |
      the plan's step checkboxes are 0 of 60 ticked while M1's tasks are complete and committed
    detail: |
      workshop/plans/000016-console-qa-plan.md - Tasks 1-6 shipped in five
      commits, but no step box is ticked. The two most recent archived plans set
      the convention (000015 39/39, 000014 33/33; 000011 did not), so this is
      soft, but leaving M1's boxes open makes "where does M2 resume" ambiguous
      when the plan is picked up in a fresh session.
```

---

## Re-review — 2026-08-23T16:29:26-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aab71caeb1303528c1cd8db9641e83402612a4ed^..a2a9f9422791c6893eec5f5a64e507c8bb284f3f |
| command | sdlc milestone-close --issue 16 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-23T16:29:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Three of the five open findings are genuinely closed and I confirmed each by mutation rather than by reading the diff: `mayAsk` is now consulted inside `ask` itself, so deleting that guard reddens exactly the three forced cells of `TestRawNeverAsks` (BR-9); the capturer injection is live and `TestTheCapturerInjectionIsLive` is a real control that goes red the moment `withStore` starts overwriting a supplied double (BR-10); and both round-1 placement fixes are now pinned on the emitted byte stream — removing `eraseLine` reddens `TestRawLoopMessagePlacement/the_bare-?_note`, and a *faithful* BR-6 revert (leading `\r\n` back inside `askInSession`, removed from the forced caller) reddens `TestForcedAndUnforcedAsksRenderAtTheSameHeight` (BR-11). What keeps this off SHIP is that the round's own claim — "every fix mutation-verified" — is measurably false for **BR-12**: both of its halves can be reverted with `go test ./cmd/define/` fully green (`hist.Add(cmd.recallLine())` → `hist.Add(line)` restores the `\how so` → `how so` inversion; the entire `q.forced` branch in `ask` can be *deleted*). The behaviour is right, the wiring is unpinned, so by this gate's own rule that is `not-addressed`. Two new Important findings: `assertDidNotAsk` cannot distinguish "did not ask" from "asked and printed the other message", so 2 of `TestRawNeverAsks`'s 6 cells pass under the mutation they exist to catch; and `echo '?why' | define -raw` exits **1** where `ask` computed 2 and README states 2, because the ask branch collapses the code into `anyFailed` — precisely the BR-16 defect the `cmdCode` variable 50 lines above exists to prevent. `go vet ./...` and `go test ./...` are green.

## 1. Strengths

- **The `-raw` fix went to the point the thing happens, not to a route.** `mayAsk` (cmd/define/ask.go:28) is consulted inside `ask` *and* at cmd/define/main.go:425, and the enumeration is real where it counts: deleting the `!mayAsk` block from `ask` reddens `one-shot/forced`, `piped/forced` and `editor/forced` and leaves the unforced three green — exactly the three cells BR-9 named.
- **`TestTheCapturerInjectionIsLive` (askroute_test.go:203) is a control that actually controls.** I made `withStore` fill `d.capture` unconditionally; it goes red, along with four subtests of `TestCaptureArityIsOnePerLookup`. The BR-10 rule is now enforced by a test rather than by a comment.
- **Both hatches are pinned far deeper than the parser.** Deleting the `?` arm from `parseREPLLine` reddens 6 tests / 10 subtests spanning the parser, both loop shells, the one-shot, the decision table and message placement; deleting the `\` arm reddens 3 tests / 5 subtests. The issue's ticked Done-when row "each is exercised by a test that fails when the hatch is removed" is honestly ticked now.
- **`recallLine` (repl.go:91) is the right shape for the fix**, and its round-trip assertion (`first.kind != again.kind || first.literal != again.literal`, askroute_test.go:328) is the assertion that makes "same meaning" falsifiable rather than "same string" — it just isn't reached from the wiring.
- **BR-13 closed cleanly:** Chunk 1 is 29/29 ticked and Chunk 2 is 0/31, so "where does M2 resume" reads off the plan without inference.

## 2. Critical findings

None.

## 3. Important findings

**N-1 — `assertDidNotAsk` passes for the failure it was written to catch, in 2 of 6 cells (cmd/define/askroute_test.go:269).**
**This is the 3rd finding in family `test-asserts-nothing`.** Do not fix the two cells — state the rule.

The rule: **an assertion that pins "X did not happen" must assert the positive observable that distinguishes X from every other outcome, not the absence of one string.** `assertDidNotAsk` checks only that stderr lacks `"no model configured"` (plus, in 2 of 6 cells, an exit code). Measured — remove `mayAsk(opt)` from the miss branch at main.go:425 (a plausible cleanup: `ask` guards too, and the comment there flags the redundancy) and:

```
one-shot/unforced  RED   (via the exit code, 2 != 1 — not the message)
piped/unforced     GREEN → stderr = `define: -raw does not ask; drop -raw, or drop the "?"`
editor/unforced    GREEN → same
```

Both green cells are printing advice to "drop the `?`" for a line containing no `?` at all, and the assertion is satisfied. atlas/define.md:538 and the test's own comment both state the enumeration "covers every cell"; measured, 4 of 6 cells can fail and 2 cannot. The distinguishing observable is available and cheap: assert stderr *contains* `no dictionary entry` (the miss the scripting contract promises), not merely that it lacks the ask message.

**N-2 — `echo '?why' | define -raw` exits 1; `ask` computed 2 and README states 2 (cmd/define/repl.go:189).**
**This is the 3rd finding in family `doc-overstates-code`.** Do not patch the one site — state the rule.

The rule: **a non-zero code a dispatch computes inside `replLines` must survive the loop, and every exit-code absolute in README must be measured across `{one-shot, piped, editor}` before it is written.** `askHere` (repl.go:189-193) collapses `ask`'s return into `anyFailed`, discarding the 2. The `cmdCommand` branch 45 lines below does it correctly (repl.go:236-240), and the comment introducing `cmdCode` at repl.go:180 names this exact defect by number: *"Collapsing it into anyFailed made `echo /histry | define` exit 1 where dispatchCommand computes 2 and the README documents 2 for a usage error (BR-16)."* Measured against the built binary:

```
define -raw '?why'              exit 2   ✓ README.md:81-83
echo '?why' | define -raw       exit 1   ✗ (ask returns 2; anyFailed swallows it)
echo '/histry' | define         exit 2   ✓ (cmdCode path)
```

README.md:81 states the claim unqualified ("on either route … a usage error (exit `2`)"), and `TestRawNeverAsks`'s piped cells pass `wantCode 0`, so no cell asserts it. The one-line fix is to give `askHere` the `cmdCode` treatment; the rule is the sweep of README's four exit-code absolutes across the three entry modes (I measured the other three: bare `?`/`\` one-shot both exit 2 ✓; bare `\` piped exits 0 with the generic "type a word" message; bare `?` piped exits 0 — defensible under README's separate piped sentence, worth a deliberate call rather than an accident).

## 4. Minor findings

- **This is the 4th finding in family `raw-mode-message-placement`.** Do not re-fix the site. The rule BR-11 stated — every message class the raw loop writes gets an assertion on the emitted byte stream — is executed for 1 of the 3 classes `TestRawLoopMessagePlacement` enumerates. The two ask rows assert only `assertNoBareNewline(stdout)`, but `ask` writes to **stderr**, so those rows never touch the message. Verified: removing `cooked(...)` from `askInSession` (replraw.go:134) — which in production is what makes the ask message's `\n` translate at all — leaves the suite fully green. The rig's `cooked` is a no-op closure (editorloop_test.go:34), so a recording `cooked` is the missing observable.
- cmd/define/repl.go:53 — a bare `\` with a current word parses to `cmdReplay`, so typing `\` and Enter at the prompt replays the pronunciation, while a bare `?` gets its note. `TestParseREPLLine`'s `"a bare backslash is blank"` row only covers `hasCurrent=false`. Asymmetric between the two hatches; harmless today.
- cmd/define/main.go:347, repl.go:217, replraw.go:219 — `cmdNothing`'s note is reported at three sites, each with its own line ending and default text. Round 2 noted this; still three. Worth collapsing before M2 adds a fourth.
- atlas/define.md:517 writes the ask entry as `ask(opt, w, question)`; the signature is `ask(opt options, errOut io.Writer, q question)`.

## 5. Test coverage notes

All mutations run in a scratch `git archive` of HEAD, working tree untouched:

| mutation | result |
|---|---|
| delete `!mayAsk` guard inside `ask` | RED — `TestRawNeverAsks` {one-shot,piped,editor}/forced |
| drop `mayAsk(opt)` from `lookupAndRender` | **RED in 1 of 3** unforced cells (N-1) |
| `withStore` overwrites a supplied capturer | RED — `TestTheCapturerInjectionIsLive` + 4 arity subtests |
| drop `eraseLine` from the bare-`?` note | RED — `TestRawLoopMessagePlacement/the_bare-?_note` |
| faithful BR-6 revert (leading CRLF back in `askInSession`) | RED — `TestForcedAndUnforcedAsksRenderAtTheSameHeight` |
| delete the `?` arm from `parseREPLLine` | RED — 6 tests / 10 subtests |
| delete the `\` arm from `parseREPLLine` | RED — 3 tests / 5 subtests |
| `submitLine`: `hist.Add(cmd.recallLine())` → `hist.Add(line)` | **GREEN** — probe confirms `\how so` again records `how so` |
| delete `ask`'s entire `q.forced` branch | **GREEN** |
| `askInSession` stops running the ask cooked | **GREEN** |

The two green rows against BR-12 are why it is disposed `not-addressed`: the behaviour is correct (probed — `\how so` → `["\how so"]`, `?why` → `["?why"]`, `/history 7` → `["/history 7"]`, `hot  dog` → `["hot dog"]`; `?why` prints ``cannot answer `why` `` and the unforced route prints ``is not a word``), but neither half has a test that fails without it. `TestRecallPreservesWhatALineMeant` pins `recallLine` as a pure function only; `TestAQuestionIsRecalledByUpArrow/unforced` uses a question whose `recallLine()` equals its `word`, so it cannot distinguish the two. One row typed `\how so` through `runEditor` against the injected `memHistory`, and one assertion on the forced message text, close both.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass**, one Minor. `ask` is one function reached from four call sites covering all six cells (main.go:337, :362; repl.go:190; replraw.go:134); `mayAsk` is one predicate; `recallLine` collapsed three recall sites; `session` collapsed three `current` declarations. Only the three `cmdNothing` note sites are flagged.
- **ARCH-PURE — pass.** `readsAsQuestion`, `openerStem`, `truncateQuestion`, `parseREPLLine`, `recallLine` and `session` are deterministic and unit-tested with no fake. `route_test.go` drives the real `d.dict` seam against the committed corpus via `loadFakeDictionary`, not a stubbed `Lookup`. `ask` is a writer-only shell with the terminal-mode wiring kept in the raw loop's closure — the right seam for M2's stream.
- **ARCH-PURPOSE — flagged (BR-12 not-addressed, N-1).** Both are the same shape one level up from where the family sits: the *rule* was stated and the *enumeration* was written into a test's name, and the sweep stopped at the statement. BR-12 fixed both named sites and pinned neither; BR-9's enumeration names six cells and falsifies four. The class here is not "recall" or "-raw" — it is **a fix is not swept until the revert is run**, which is what the issue `## Log` claims was done and measurably was not for three of this round's reverts.
- **ARCH-MOCK — pass for M1.** No external binary or service introduced. The dictionary stays behind `d.dict` with a committed corpus plus `dict_conformance_test.go` for live drift; production and test share that boundary. M2 brings `llm.Client.Stream` — hold the line that tests hit `llmtest`'s httptest server, not a stubbed `Client`.
- **M2 watch — `session.entry` and `lookupOutcome.entry` are written at one site and read nowhere** (session.go:27 is the only reference). Declared M2 intent, but it is the "field set, never read" shape that reads as protection; give it a consuming assertion the moment Task 9 lands.
- **M2 watch — `askInSession` discards `ask`'s return code entirely** (replraw.go:134). Fine while the code is only a diagnostic; when `runAsk` returns a real taxonomy (`ErrTruncated`, `ErrRequest`), that closure needs to do something with it.

## 7. Plan revision recommendations

**This is the 2nd finding in family `plan-bookkeeping`.** Round 2 recommended a `## Revisions` entry and none was written; the plan's last entry is still `plan-quality round 2`. The rule rather than the site: **when a boundary round changes what the code does relative to the plan, the plan gets the entry in the same commit as the code** — otherwise the next session designs M2 against a plan that describes a different M1. Concretely, `workshop/plans/000016-console-qa-plan.md` needs one entry covering everything M1 shipped that the plan does not contain:

> ### 2026-08-23 — M1 boundary rounds 1–2: what M1 actually shipped
>
> **Reason:** two boundary-review rounds (ledger: `000016-console-qa-close-gate.md`) changed the code away from the plan. The plan is now stale in four places and was cited as current by round 2's own recommendation.
>
> **Delta:**
> - **`-raw` never asks, pulled forward from M2.** D3/D4 and Task 3's snippet give the miss-branch condition as `!literal && readsAsQuestion(word)`; the code is `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)` (main.go:425), with a second `mayAsk` guard inside `ask` covering the forced route. The enumeration `{forced, unforced} × {one-shot, piped, raw editor}` is added to D4, with the note that an explicit `?` under `-raw` is a **usage error** (exit 2 one-shot; see the open finding on the piped cell).
> - **`askUnavailable(stderr, question)` → `ask(opt, errOut, question)`** (Task 4's closing paragraph), and `question` carries `forced` because the honest sentence differs by route.
> - **Core concepts gains four M1 rows** the table does not list: `lookupOutcome` (main.go, new — described in Task 3 prose only), `recallLine` (repl.go, new), `question` / `mayAsk` / `ask` (ask.go, new). `ask.go` exists at M1, not M2 as the Integration table implies.
> - **Task 4's contract row** "does add the line to editor recall (`hist.Add`)" gains *what* is added: `cmd.recallLine()` at all three sites, because a stripped prefix re-submits to the opposite meaning.

No contradiction between the Core-concepts table and the tree for the rows it *does* list: `readsAsQuestion`, `truncateQuestion`, `parseREPLLine`/`replCommand` and `session` all exist at their stated paths with their stated status, and every absent row (`askContext`, `renderAskPrompt`, `recentTurns`, `exchange`, `crlfWriter`, `ReviewEvent`/`complete`, `gatherAskContext`, `runAsk`, `Store.UserModel`, `interrupter`, `deps.notifySignals`) is explicitly M2's.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      mayAsk moved into ask(); deleting that guard reddens exactly the three forced cells. Assertion strength raised separately.
  - id: BR-10
    disposition: addressed
    note: |
      Injection moved to d.capture and probe-verified live; making withStore overwrite it reddens TestTheCapturerInjectionIsLive.
  - id: BR-11
    disposition: addressed
    note: |
      Both fixes now redden on a faithful revert — eraseLine removal and the BR-6 CRLF move each go red.
  - id: BR-12
    disposition: not-addressed
    note: |
      Behaviour is correct but neither half is pinned - submitLine's recallLine and ask's whole q.forced branch both revert with the suite green.
  - id: BR-13
    disposition: addressed
    note: |
      Chunk 1 is 29/29 ticked and Chunk 2 is 0/31, so the M2 resume point is unambiguous.
findings:
  - id: new
    severity: Important
    family: test-asserts-nothing
    title: |
      assertDidNotAsk passes for the failure it exists to catch, in 2 of TestRawNeverAsks's 6 cells
    detail: |
      This is the 3rd finding in family test-asserts-nothing, so the deliverable
      is the rule. Rule - an assertion that pins "X did not happen" must assert
      the positive observable that distinguishes X from every other outcome, not
      the absence of one string. assertDidNotAsk (askroute_test.go:269) checks
      only that stderr lacks "no model configured". Measured - removing
      mayAsk(opt) from the miss branch at main.go:425 (a plausible cleanup; the
      comment there flags the redundancy with ask's own guard) reddens
      one-shot/unforced via its EXIT CODE only, while piped/unforced and
      editor/unforced stay green printing
      `define: -raw does not ask; drop -raw, or drop the "?"` for a line
      containing no "?" at all. atlas/define.md:538 and the test's own comment
      both claim the enumeration "covers every cell"; 4 of 6 can fail, 2 cannot.
      The distinguishing observable is cheap - assert stderr CONTAINS
      `no dictionary entry`, the miss the scripting contract promises.
  - id: new
    severity: Important
    family: doc-overstates-code
    title: |
      the piped loop discards the usage code ask() computes, so -raw plus "?" exits 1 where README states 2
    detail: |
      This is the 3rd finding in family doc-overstates-code, so state the rule -
      a non-zero code a dispatch computes inside replLines must survive the loop,
      and every exit-code absolute in README must be measured across
      {one-shot, piped, editor} before it is written. askHere (repl.go:189-193)
      collapses ask's return into anyFailed, discarding the 2; the cmdCommand
      branch 45 lines below feeds cmdCode correctly (repl.go:236-240) and the
      comment at repl.go:180 names this defect by number - BR-16, "Collapsing it
      into anyFailed made echo /histry | define exit 1 where dispatchCommand
      computes 2". Measured against the built binary -
      `define -raw '?why'` exits 2, `echo '?why' | define -raw` exits 1,
      `echo '/histry' | define` exits 2. README.md:81 states the claim
      unqualified ("on either route ... a usage error (exit 2)") and
      TestRawNeverAsks passes wantCode 0 for both piped cells, so nothing
      asserts it. Sweep the other three absolutes too - bare "?" and bare "\"
      exit 2 one-shot but 0 piped.
  - id: new
    severity: Minor
    family: raw-mode-message-placement
    title: |
      the two ask rows of TestRawLoopMessagePlacement assert nothing about the ask message
    detail: |
      This is the 4th finding in family raw-mode-message-placement. Do not
      re-fix the site. BR-11's rule was executed for 1 of the 3 message classes
      the test enumerates - the ask rows assert only
      assertNoBareNewline(stdout), but ask writes to STDERR, so they never touch
      the message. Verified - removing cooked(...) from askInSession
      (replraw.go:134), which in production is the only reason the ask message's
      "\n" translates at all, leaves go test ./cmd/define/ fully green. The
      rig's cooked is a no-op closure (editorloop_test.go:34), so a recording
      cooked is the missing observable.
  - id: new
    severity: Minor
    family: plan-bookkeeping
    title: |
      the plan still describes an M1 the code no longer implements, despite round 2 recommending the Revisions entry
    detail: |
      workshop/plans/000016-console-qa-plan.md - last Revisions entry is
      "plan-quality round 2"; nothing records the two boundary rounds. Stale in
      four places - Task 3's snippet gives the miss condition as
      `!literal && readsAsQuestion(word)` where the code is
      `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; "-raw" and
      "mayAsk" appear nowhere in the plan; Task 4 names askUnavailable where the
      code has ask(opt, errOut, question); and Core concepts lists none of
      lookupOutcome, recallLine, question or mayAsk, while placing ask.go in M2.
      Rule - when a boundary round changes what the code does relative to the
      plan, the plan gets the entry in the same commit as the code.
  - id: new
    severity: Minor
    family: forced-route-enumeration
    title: |
      a bare "\" with a current word replays audio while a bare "?" gets its note
    detail: |
      repl.go:53 - the "\" arm strips the prefix and falls into the
      empty-word test, so `\` alone with hasCurrent returns cmdReplay.
      TestParseREPLLine's "a bare backslash is blank" row only covers
      hasCurrent=false, so the asymmetry between the two hatches at the prompt
      is unasserted. Harmless today; worth one row.
```

---

## Re-review — 2026-08-23T16:50:25-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 16 — free-form Q&A in the console: input classification + the directory as context |
| repo | tools |
| issue file | workshop/issues/000016-console-qa.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aab71caeb1303528c1cd8db9641e83402612a4ed^..507a48ab16c749222d62ed1e5093de3612bc5dc8 |
| command | sdlc milestone-close --issue 16 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-23T16:50:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All mutations measured. Working tree is green (`go vet` clean, `go test ./...` passes). Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Four of the six open findings are genuinely closed and I confirmed each by revert rather than by reading the diff: `assertDidNotAsk` now asserts the positive observable, so dropping `mayAsk(opt)` from the miss branch reddens **all three** unforced cells instead of one (BR-14); the `fail(code)` sink makes ask's 2 survive the piped loop, and I measured all 8 exit-code cells on the built binary against README — 8/8 match (BR-15); removing `cooked(...)` from `askInSession` now reddens both ask rows of `TestRawLoopMessagePlacement` (BR-16); and the bare-`\` fix reddens two parser rows (BR-18). What keeps this off SHIP is that **BR-12 is unchanged for the third consecutive round** — `hist.Add(cmd.recallLine())` → `hist.Add(line)` and deleting `ask`'s entire `q.forced` branch both leave `go test ./cmd/define/` fully green, exactly as round 3 measured — and **BR-17 is unchanged for the second**, the plan still carrying three `## Revisions` entries none of which is a boundary round. The one new Important is the same rule one level down: no test anywhere asserts the *text* a message delivers, and as a direct consequence a wrong message shipped — `noteEmptyLiteral` is a Go raw string containing `\\`, so a bare `\` prints `define: type a word after "\\"`, and the only assertion compares the production constant to itself.

## 1. Strengths

- **BR-14's fix went to the observable, not the cell.** `assertDidNotAsk` (askroute_test.go:269) now requires stderr to *contain* `no dictionary entry` or the refusal. Measured: the mutation round 3 used to expose 1 of 3 unforced cells now reddens `one-shot/unforced`, `piped/unforced` and `editor/unforced` together.
- **BR-15 was swept, not patched.** `fail(code)` (repl.go:186-195) is one sink all four branches feed, and I measured the full grid on the built binary — `define -raw '?why'` 2, `echo '?why' | define -raw` 2 (was 1), bare `?`/`\` 2 both one-shot and piped (were 0 piped), `echo '/histry' | define` 2. Every absolute README states now holds in every mode it quantifies over.
- **The recording `cooked` is a real observable** (askroute_test.go:~365). The rig's `cooked` is a no-op, so capturing `errb` deltas across the closure is the only thing that can see raw-vs-cooked placement — and it works: dropping `cooked(...)` reddens both ask rows.
- **ARCH-PURE holds cleanly.** `readsAsQuestion`, `openerStem`, `truncateQuestion`, `parseREPLLine`, `recallLine`, `nothingSays`, `mayAsk` and `session` are all deterministic and unit-tested with no fake; `route_test.go` drives the real `d.dict` seam against the committed corpus via `loadFakeDictionary`, never a stubbed `Lookup`.
- **The hatches are pinned deep.** Dropping the `requestVerbs` arm reddens `TestReadsAsQuestion` *and* `TestConsoleDecisionTable`; the `\` and `?` arms each redden across parser, both loop shells and the one-shot.

## 2. Critical findings

None.

## 3. Important findings

**N-1 — no test asserts what any of #16's messages actually say, and a wrong one shipped (cmd/define/repl.go:44).**

**This is the 4th finding in family `test-asserts-nothing`.** Do not fix the instance — state the rule.

The rule: **an assertion on a message must compare the bytes the user receives against a literal expectation written in the test. Referencing the production constant asserts only that a branch was selected, not that its text is right.**

`repl_test.go:43,45` assert `note: noteEmptyLiteral` — the constant compared to itself. `noteEmptyLiteral` is a Go **raw** string literal, so its `\\` is two characters:

```
$ define '\'
define: type a word after "\\"      # both one-shot and piped
$ define '?'
define: type a question after "?"   # the sibling, correct
```

Measured prevalence — 4 of the 5 message/code behaviours this milestone introduced are unasserted at the point of delivery, and the family recurs because each round pinned a *placement* and never a *text*:

| behaviour | mutation | result |
|---|---|---|
| the bare-`?`/`\` note text | delete `nothingSays`'s `if c.note != ""` early return | **GREEN** |
| the bare-hatch piped exit 2 | delete `if cmd.note != "" { fail(2) }` (repl.go:261) | **GREEN** |
| truncation in ask's two messages | `truncateQuestion(q.text)` → `q.text` | **GREEN** |
| the note's raw-mode placement | drop `eraseLine` | RED (BR-11's fix) |

The second row is BR-15's own sweep: the code is correct in all 8 cells, but README's new `a bare ? or \ with nothing after it` → exit 2 has no assertion on the piped route. The grid I measured above is the table this wants — one test over `{?, \} × {one-shot, piped}` asserting code *and* stderr text, plus a literal-text assertion for the two notes, closes all four rows at once.

## 4. Minor findings

- **This is the 4th finding in family `doc-overstates-code`.** Do not re-fix the site. The rule: **a consolidation claimed as "the ONE place" must be verified by enumerating the copies it consolidated — a claim about uniqueness is an absolute, and it quantifies over the whole tree.** `nothingSays`'s doc comment (repl.go:100-104) and atlas/define.md:553 both say it is "the one place that answers 'this line meant nothing — why'". `replayInPlace` (replraw.go:~272) still holds a byte-identical copy of that function's replay sentence, and it is the **live** path for the raw editor's bare Enter with nothing current (pinned by `TestEditorLoopBareEnterWithNoCurrentWord`, editorloop_test.go:235). Second half of the same rule: `nothingSays(cmd, true)` at repl.go:257 hardcodes `true` where `sess.hasCurrent()` is in scope — and a note-less `cmdNothing` is reachable *only* when `hasCurrent` is false (a blank line with a current word parses to `cmdReplay`), so that call site's `canReplay=true` branch selects the replay message in the one state where there is definitionally nothing to replay. The text is pre-#16 and faithfully ported, but the parameter introduced to distinguish the two cases is fed a literal at both loop sites.
- cmd/define/replraw.go:195, :211, :241, :250 — four byte-identical `finish(); Fprintf("define: lost the terminal: %v"); return 1` blocks, two of them added by this diff (2 before, 4 now). M2's Task 11 adds streaming inside `askInSession` and is positioned to add a fifth. One helper (ARCH-DRY), the same argument `nothingSays` was extracted on.
- atlas/define.md — the free-form section is otherwise accurate against the code; I spot-checked the decision table, the three `readsAsQuestion` arms including the `why?` qualification, "all three recall sites use it" (3 confirmed), and the exit-code paragraph.

## 5. Test coverage notes

All mutations run in a scratch `git archive` of HEAD with its own git init; the working tree was never modified. Baseline green, `go vet ./...` clean.

| mutation | result |
|---|---|
| drop `mayAsk(opt)` from `lookupAndRender`'s miss branch | RED — 3 of 3 unforced cells (BR-14 ✓) |
| `askHere` collapses ask's code into `anyFailed` | RED — `TestRawNeverAsks/piped/forced` (BR-15 ✓) |
| `askInSession` stops running the ask cooked | RED — both ask rows (BR-16 ✓) |
| bare `\` falls through to the empty-word test | RED — 2 parser rows (BR-18 ✓) |
| `recallLine` cmdCommand drops its args | RED — `TestRecallPreservesWhatALineMeant` |
| `readsAsQuestion` drops the `requestVerbs` arm | RED — 3 subtests across 2 tests |
| **`submitLine`: `hist.Add(cmd.recallLine())` → `hist.Add(line)`** | **GREEN** — BR-12 |
| **delete `ask`'s entire `q.forced` branch** | **GREEN** — BR-12 |
| **delete `nothingSays`'s note early-return** | **GREEN** — N-1 |
| **delete `fail(2)` for a bare hatch in `replLines`** | **GREEN** — N-1 |
| **`truncateQuestion(q.text)` → `q.text` in ask** | **GREEN** — N-1 |

BR-12's two green rows are why it is disposed `not-addressed` again rather than as a fresh finding: the behaviour is correct — `recallLine` round-trips all four kinds and `?why` prints ``cannot answer `why` `` while the unforced route prints ``is not a word`` — but `TestRecallPreservesWhatALineMeant` pins `recallLine` as a pure function only, and `TestAQuestionIsRecalledByUpArrow/unforced` uses a question whose `recallLine()` equals its `word`, so neither can distinguish the two. One row typing `\how so` through `runEditor` against the injected `memHistory`, and one assertion on the forced message's text, close both.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, two Minors.** `ask` is one function reached from four call sites covering all six cells; `mayAsk` one predicate; `fail` one exit-code sink; `recallLine` one canonical form; `session` collapsed three `current` declarations. Flagged: `replayInPlace` duplicates `nothingSays`'s replay sentence, and the four `lost the terminal` blocks.
- **ARCH-PURE — pass.** Every entity the plan calls PURE is deterministic and its test runs with no fake. `ask` is a writer-only shell with the terminal-mode wiring held in the raw loop's closure — the correct seam for M2's stream.
- **ARCH-PURPOSE — flagged (BR-12 3rd round, BR-17 2nd round, N-1).** All three are the same shape: the *class* is named in a comment or a rule and the *verification* is the part that stops at the instance. BR-12's behaviour was fixed at both named sites and pinned at neither, twice; N-1 shows the message surface fixed for placement three times and never for text. The class here is **a fix is not delivered until the revert is red** — which the issue's own `## Log` claims ("Every fix mutation-verified") and which is measurably false for BR-12 in two consecutive rounds. Before M2, make the revert the acceptance step rather than the review's discovery.
- **ARCH-MOCK — pass for M1.** No external binary or service introduced; the dictionary stays behind `d.dict` with a committed corpus plus `dict_conformance_test.go`. M2 brings `llm.Client.Stream` — hold the line that tests hit `llmtest`'s httptest server, not a stubbed `Client`.
- **M2 watch — `lookupOutcome.entry` / `session.entry` are written at one site and read nowhere.** Declared M2 intent (Task 9), but it is the "field set, never read" shape; give it a consuming assertion the moment Task 9 lands.
- **M2 watch — `askInSession` discards `ask`'s return code** (replraw.go:134). Harmless while the code is only a diagnostic; when `runAsk` returns `ErrTruncated`/`ErrRequest`, that closure needs to act on it.

## 7. Plan revision recommendations

BR-17 stands, so the recommendation is unchanged and I will not restate it at length: `workshop/plans/000016-console-qa-plan.md` still has exactly three `## Revisions` entries (`the classifier has three arms`, `plan-quality round 1`, `plan-quality round 2`) and none for the three boundary rounds. Verified stale in four places — Task 3's snippet gives the miss condition as `!literal && readsAsQuestion(word)` where the code is `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; the strings `-raw` and `mayAsk` appear nowhere in the file; Tasks 4 and 10 name `askUnavailable(stderr, question)` where the code has `ask(opt options, errOut io.Writer, q question)`; and the Core-concepts tables list `ask.go` only under M2 integration points (lines 219-220) while omitting `lookupOutcome`, `recallLine`, `question`, `mayAsk`, `ask` and `nothingSays`, all of which shipped in M1. One entry covering those four deltas, in the same commit as the fixes for this round.

Everything else in the Core-concepts cross-check is consistent: `readsAsQuestion`, `truncateQuestion`, `parseREPLLine`/`replCommand` and `session` exist at their stated paths with their stated status and are PURE with IO-free tests; every absent row (`askContext`, `renderAskPrompt`, `recentTurns`, `exchange`, `crlfWriter`, `ReviewEvent`/`complete`, `gatherAskContext`, `runAsk`, `Store.UserModel`, `interrupter`, `deps.notifySignals`) is explicitly M2's.

```findings
dispose:
  - id: BR-12
    disposition: not-addressed
    note: |
      Third round unchanged - hist.Add(cmd.recallLine())->hist.Add(line) and deleting ask's whole q.forced branch both revert with the suite green.
  - id: BR-14
    disposition: addressed
    note: |
      assertDidNotAsk asserts the positive observable; dropping the miss-branch mayAsk now reddens 3 of 3 unforced cells, not 1.
  - id: BR-15
    disposition: addressed
    note: |
      fail(code) sink reddens piped/forced on revert; measured 8 of 8 exit-code cells on the built binary against README. Coverage of the swept cells raised in N-1.
  - id: BR-16
    disposition: addressed
    note: |
      Recording cooked is a live observable - removing cooked(...) from askInSession reddens both ask rows.
  - id: BR-17
    disposition: not-addressed
    note: |
      Plan still has three Revisions entries, none a boundary round; askUnavailable still named, -raw/mayAsk absent, Core concepts still places ask.go in M2.
  - id: BR-18
    disposition: addressed
    note: |
      Reverting the bare-backslash arm reddens both hasCurrent rows of TestParseREPLLine.
findings:
  - id: new
    severity: Important
    family: test-asserts-nothing
    title: |
      no test asserts what any of #16's messages say, and a doubled backslash shipped as a result
    detail: |
      This is the 4th finding in family test-asserts-nothing, so the deliverable
      is the rule. Rule - an assertion on a message must compare the bytes the
      user receives against a literal expectation written in the test;
      referencing the production constant asserts only that a branch was
      selected, not that its text is right. Shipped defect - noteEmptyLiteral
      (repl.go:44) is a Go RAW string literal containing `\\`, so both
      `define '\'` and `echo '\' | define` print
      `define: type a word after "\\"` with a doubled backslash, while its
      sibling bare-"?" note is correct. The only assertions are repl_test.go:43
      and :45, which compare note to noteEmptyLiteral - the constant to itself.
      Measured prevalence, 4 of 5 message/code behaviours this milestone
      introduced are unasserted at the point of delivery, all GREEN on revert -
      deleting nothingSays's `if c.note != ""` early return; deleting
      `if cmd.note != "" { fail(2) }` at repl.go:261 (BR-15's own sweep, correct
      in code but unpinned, so README's "a bare ? or \ with nothing after it"
      exit 2 has no piped assertion); and replacing truncateQuestion(q.text)
      with q.text in ask's two messages. Only the placement fix (eraseLine) goes
      red, which is why the family recurs - each round pinned a placement and
      never a text. One table over {?, \} x {one-shot, piped} asserting exit code
      AND literal stderr text closes all four rows.
  - id: new
    severity: Minor
    family: doc-overstates-code
    title: |
      nothingSays is documented as "the ONE place" while replayInPlace holds a live duplicate of its replay sentence
    detail: |
      This is the 4th finding in family doc-overstates-code. Do not re-fix the
      site. Rule - a consolidation claimed as "the ONE place" must be verified by
      enumerating the copies it consolidated; a uniqueness claim is an absolute
      and it quantifies over the whole tree. nothingSays's doc comment
      (repl.go:100-104) and atlas/define.md:553 both assert it is the one place
      that answers "this line meant nothing - why", but replayInPlace
      (replraw.go, `case current == ""`) still holds a byte-identical copy of
      that sentence, and it is the LIVE path for the raw editor's bare Enter
      with nothing current - pinned by TestEditorLoopBareEnterWithNoCurrentWord
      (editorloop_test.go:235). Same rule, second half - nothingSays(cmd, true)
      at repl.go:257 hardcodes true where sess.hasCurrent() is in scope, and a
      note-less cmdNothing is reachable ONLY when hasCurrent is false (a blank
      line with a current word parses to cmdReplay), so that site selects "press
      return to replay the last one" in the one state where there is nothing to
      replay. Measured on the built binary. The text is pre-#16 and faithfully
      ported; the parameter introduced to distinguish the two cases is what is
      fed a literal.
  - id: new
    severity: Minor
    family: stdlib-reuse
    title: |
      four byte-identical "lost the terminal" blocks in runEditor, two added by this diff
    detail: |
      replraw.go:195, :211, :241, :250 - the same
      finish(); Fprintf("define: lost the terminal: %v"); return 1 three-liner,
      up from 2 copies before this window. M2's Task 11 adds streaming inside
      askInSession and is positioned to add a fifth. One helper, on the same
      argument nothingSays was extracted on (ARCH-DRY).
```
