# Boundary Review — tools#12 (whole-issue close)

| field | value |
|-------|-------|
| issue | 12 — review form 2.2: cloze from current news with curated distractors |
| repo | tools |
| issue file | workshop/issues/000012-vocab-form-cloze.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..46a769bbd94aa04be3c028813a5911b4eff5c937 |
| command | sdlc close --issue 12 |
| reviewer | claude |
| timestamp | 2026-09-07T11:22:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three remaining Done-when rows are genuinely delivered and genuinely pinned — I reverted six separate properties in a scratch copy (`CaptureFlag`'s event kind, the every-occurrence loop, the word-run extension, the shuffle, the equal-distractor guard, the graded-state flag offer) and every one reddened a named test with a legible message. `go test ./...`, `go vet ./...`, `go vet -tags conformance ./...` and `gofmt -l` are clean, and a 45s `FuzzBlankStem` run found nothing new. Nothing here is Critical. What holds it back from SHIP is a cluster of cheap gaps: the `?` gesture works but is named in no on-screen prompt (and the graded prompt now actively mis-describes it), `apply` hands *any* rune to `Grade` on an already-graded question so a stray digit re-picks the answer on a Cloze, a store read error in `clozeAsk` degrades silently where its neighbour prints, and the issue's `## Log` carries no record of the implementation at all — including no evidence for the hand-run sitting the plan's Verification commits to and the estimate priced as a named deviation.

## 1. Strengths

- **The mutation sweep caught the gate's own finding unpinned, and the fix asserts through the consumer** — `cmd/define/capture_test.go:694` folds a flag-only log through `schedule.Fold` rather than reading `Kind`. I verified by reverting: changing `CaptureFlag` to write `EventReviewed` reddens both assertions, naming the demoted word. This is exactly the check the "claimed fixes" rule exists for, and it was run without being asked.
- **`blankStem` is specified by its leak table, then by the property, then by fuzz** (`cmd/define/cloze_test.go:20,74`, `cloze_fuzz_test.go`). Reverting to first-occurrence-only and to `at + n` each reddened three distinct rows. The two fuzz findings (the `RuneError` hang, the `_` non-word answer) are seeded into the committed corpus rather than merely fixed.
- **`blankOut` delegating to `blankStem`** (`harvest_judge.go:177`) is the right correction of a DRY argument that had been settled on a wrong cost estimate, and the unmoved `testdata/golden/veto-prompt.txt` is the cheap proof it is the same intent.
- **`optionSet` was extracted with `#7`'s suite untouched** — `play/choice.go`'s diff is deletion plus delegation, and no test file moved.
- **`EventFlagged` is inert across every consumer.** I checked all five readers (`schedule/progress.go:155`, `history_cmd.go:107`, `history_store.go:65`, `reflect.go:87`, `summariseLookups`) and each filters on `Kind` before touching `Found`/`Word`, so the new kind cannot be miscounted as a lookup or a review.
- **ARCH-SECURE holds end to end for the new log field**: `oneLine` runs at both `SetItems` and YAML `Items()` (`store/item.go:169`, `yaml.go:662`), so a distractor cannot carry the newline that would forge a column-0 `- ` record in the day log's `splitRecords` — and the `storetest` row round-trips it through the real file, not just `Mem`.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `?` is named in no prompt line, and the graded prompt contradicts it.** `cmd/define/play/cloze.go:131` returns `"1-4 = pick the word"`; `play_loop.go:1259` returns the constant `"any key = next word, d = remove from deck, …"` once graded. So the only in-TUI documentation of the keys never mentions `?`, and after answering it tells the learner "any key = next word" when `?` does something else entirely. `gradePrompt`'s own doc comment (`play_loop.go:1277`) records the prior instance of this exact class — a keys line disagreeing with what the form actually does, "a bug no test could see". Fix: `keysFor` gains the flag clause when the form can flag (or `Cloze.Keys()` appends `, ? = bad question`), and `livePrompt`'s graded branch derives from the form via `canFlag(q)` instead of the flat constant. `doc_sync_test.go` should pin the new wording the way it pins the existing two.

**I-2 — the graded branch offers *any* rune to `Grade`, so a digit re-picks the answer.** `play/session.go:277` guards on `canFlag(q)`, not on the gesture, and its own comment claims the opposite ("re-grading it would move a pick on a question already answered"). I confirmed with a probe test: after a miss, pressing `3` moves `Cloze.chosen` from 0 to 2. It is inert today only because the Cloze is already spent and `Reveal()` is never called again — but the atlas says "any form can flag; the board is the obvious next one", and the guard as written is what a future flagging form will inherit. Fix: `in.Rune == FlagKey && canFlag(q)` (the const is the package's, so this needs no type switch on a concrete form), or give `Flagging` the gesture (`Flag(k rune) bool`) so the session never re-grades.

**I-3 — `clozeAsk` swallows a store read error where its neighbour reports one.** `cmd/define/cloze.go:192` treats `err != nil` and `len(items) == 0` identically and returns silently, four lines away from `play_loop.go:993`, which prints `define: skipping %q: %v` for a dictionary failure. A corrupt or unreadable `items/<lang>/<key>.yaml` therefore makes the cloze form vanish for that word forever with no signal — for material that cost a model call to author. Fix: split the branches and warn on the error (`c.warnf`'s pattern already exists in `capture.go`), keeping the fallback to 2.3 either way.

**I-4 — the issue's `## Log` records nothing this boundary did.** `workshop/issues/000012-vocab-form-cloze.md` still ends at `### 2026-08-20 / Created as part of the define-learn project`, while all four Plan rows and all three Done-when rows are ticked in the working tree. The 19-property sweep and the unpinned-finding discovery exist only in `46a769b`'s commit body, and the plan's Verification row **"A real sitting, run by hand against the batch `#10`'s checkpoint generated"** has no evidence anywhere in the tree — which matters because the estimate books it as named deviation 3 precisely to stop it being priced as a formality, and because it is what would have surfaced I-1. Fix: log the sweep result, the hand-run outcome (or say it was not run), and the review verdict before closing.

## 4. Minor findings

- `cmd/define/capture.go:149` — the flag outcome carries `Form` (set at `session.go:415` and `:514`) and `ReviewEvent` has a `Form` field, but `CaptureFlag` drops it. Set at two sites, read at zero; pass `Form: out.Form` so a future flagging form is diagnosable.
- `cmd/define/cloze.go:197` vs `play_loop.go:975` — the `Render` + `marks[key] = clickable{…}` block is copy-pasted between `clozeAsk` and `ask`, and a word that *has* items but no usable cloze item renders its entry twice (ARCH-DRY; the plan's own words are "one place renders"). Extract a `renderInto(marks, key, entry)` helper, or move the render below `clozeFor`'s nil check.
- `cmd/define/cloze.go:99` — `strings.TrimSpace(it.Answer) == ""` is unreachable: `hasLetterOrDigit` at line 95 already returned false for it. Dead guard.
- `workshop/projects/define-learn.md:75` — the "where a model is used" table still reads `veto a distractor | #12`, which the issue's 2026-09-04 revision moved to `#10` and this plan's own scope note repudiates ("this issue builds no selection and no model call").
- `Cloze.Flagged()` records the option set but not the stem. The Spec's literal requirement ("the flag with the full option set") is met; a stem would make a flagged item locatable on disk without a timestamp join. Note for `#13`.

## 5. Test coverage notes

Coverage is strong and, unusually, self-verified: the sweep's own record shows one property that came back green and was re-pinned. My independent reverts confirm the pins reach the implementation rather than a fake — `TestCaptureFlagWritesAFlagAndNotAReview` drives the *real* `storeCapturer`, which is what the earlier end-to-end-with-a-fake test could not. `TestTheFormSelectionRule` deliberately uses a six-word deck so the two fallback rows fail for the rule under test rather than for `#42`'s reason — that is the kind of rig error most selection tests ship with. Gaps: the graded-state flag is pinned in `play` (`TestAFlagIsHeardInEverySessionState/after_answering`) but never driven through `playSession`, so nothing pins that a flag *after a miss* reaches `CaptureFlag` in the loop; and I-2's stray-digit-after-grading cell has no test at all. Both are one test each. `optionset_test.go` was planned and not written, which is coherent — Task 1's oracle was `#7`'s untouched suite — but `keysFor`, `correctIndex` and `wrongPick` are now covered only indirectly through the two forms.

## 6. Architectural notes

- **ARCH-DRY — pass, with one flag.** `optionSet`, the `blankOut` delegation and `wordRunEnd` reusing `highlight.go`'s `isWordRune` are three consolidations in one diff. The flag is the duplicated render/marks block (Minor above).
- **ARCH-PURE — pass.** `play` stays import-guarded pure; `blankStem`, `usableItem` and `clozeFor` are pure and unit-tested with no store, no dictionary and no model; `clozeAsk` is the thin IO seam and holds the only `Items()` call.
- **ARCH-PURPOSE — flag.** The code fulfils the purpose; the *record* under-delivers it (I-4), and a gesture the learner cannot discover in the TUI (I-1) is the purpose half-shipped.
- **ARCH-MOCK — pass.** No new external dependency. The new store verb landed as a `storetest` row held by both implementations rather than in one implementation's test — which is `#10`'s BR-2 lesson applied forward.
- **ARCH-CONSTRAINTS — pass.** No model or network call, asserted with the seam made to *panic* rather than nil. One `Items()` read per due word, same order as the existing `Deck()`/`Events()` reads. The only waste is the double `Render` (Minor).
- **ARCH-SECURE — pass.** The on-disk item is parsed into a typed floor at the boundary (`usableItem`), the floor is enumerated rather than exemplified, and the fuzz-derived `hasLetterOrDigit` closed a hole one predicate over from the first fix. Model text reaching the day log cannot forge a record. The one degradation that is *not* visible is I-3.
- **ARCH-ORDER — flag.** The three states `?` can arrive in are enumerated, routed with the `InputDrop`/`InputQuit` precedent, and pinned; `Flagged()` is one-shot by contract with a test that fails on a second read. The unwritten cell is `(Graded, non-flag rune)` — I-2. Also worth noting for future work: the session's carried state is still a bool constellation (`Graded`, `Revealed`, plus the form's `flagged`/`chosen`), and this diff adds the fourth flag to it; a tagged enum over `asking | revealed | graded` is what would make cells like I-2 unrepresentable rather than merely untested. Pre-existing, not this boundary's to fix.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000012-vocab-form-cloze-plan.md` recording that the code diverged from the plan in three places, none of them substantive but all of them currently claims the plan makes and the code does not honour:

1. **`ReviewEvent.Flagged` shipped as `ReviewEvent.Options`** (`store/event.go:93`). The Integration points table and its prose both name `Flagged`; the field that exists is `Options`, with the "flagged" fact carried by `Kind`. The code is better than the plan here — record the rename rather than leave the table naming a field that does not exist.
2. **Task 1's `cmd/define/play/optionset_test.go` was not created**, deliberately: Task 1 Step 4 requires no test file to change, and `#7`'s suite is the oracle. Say so, so a reader does not go looking.
3. **Tasks 5 and 6 name `cmd/define/play_loop_test.go`; the tests landed in `cmd/define/cloze_test.go`** (same package, `TestTheFormSelectionRule`, `TestAClozeSittingNeverReachesForTheModel`, `TestFlaggingAQuestionRecordsItWithoutScoringIt`). Point the rows at where the pins actually live.

```findings
findings:
  - id: new
    severity: Important
    family: prompt-line-matches-live-keys
    title: |
      the flag key `?` is named in no on-screen prompt, and the graded prompt contradicts it
    detail: |
      play/cloze.go:131 returns "1-4 = pick the word" and play_loop.go:1259 returns the
      constant "any key = next word, ..." once graded, so the only in-TUI documentation of
      the keys never mentions `?` and actively mis-describes it after answering. gradePrompt's
      own doc comment records the prior instance of this class. Fix in keysFor when the form
      can flag, derive livePrompt's graded branch from canFlag(q), and extend doc_sync_test.go.
  - id: new
    severity: Important
    family: capability-guard-too-wide
    title: |
      the graded branch hands ANY rune to Grade, so a stray digit re-picks the answer on a Cloze
    detail: |
      play/session.go:277 guards on canFlag(q) rather than on the gesture, while its own comment
      claims re-grading is what the guard prevents. Verified with a probe: after a miss, pressing
      `3` moves Cloze.chosen from 0 to 2. Inert today only because the question is already spent
      and Reveal() is not called again. Guard with `in.Rune == FlagKey && canFlag(q)`, or move the
      gesture onto the Flagging interface so the session never re-grades.
  - id: new
    severity: Important
    family: silent-degradation-unreported
    title: |
      clozeAsk swallows an Items() read error where the adjacent lookup failure is reported
    detail: |
      cmd/define/cloze.go:192 treats `err != nil` and `len(items) == 0` identically and returns
      silently, four lines from play_loop.go:993 which prints a stderr diagnostic for a dictionary
      failure. A corrupt or unreadable items file makes the cloze form vanish for that word with no
      signal, for material that cost a model call to author. Split the branches and warn on the error.
  - id: new
    severity: Important
    family: boundary-record-unwritten
    title: |
      the issue's Log records nothing this boundary did, including the hand-run sitting
    detail: |
      workshop/issues/000012-vocab-form-cloze.md still ends at the 2026-08-20 creation note while
      every Plan and Done-when row is ticked. The 19-property mutation sweep and its finding exist
      only in 46a769b's commit body, and the plan's Verification row "a real sitting, run by hand"
      has no evidence in the tree — the item the estimate books as named deviation 3, and the one
      that would have surfaced the missing `?` prompt.
  - id: new
    severity: Minor
    family: outcome-field-unread
    title: |
      the flag outcome carries Form and CaptureFlag drops it
    detail: |
      Set at session.go:415 and :514, and ReviewEvent has a Form field, but capture.go:149 does not
      write it. Set at two sites, read at zero. Pass `Form: out.Form`.
  - id: new
    severity: Minor
    family: one-place-renders
    title: |
      clozeAsk duplicates ask's render+marks block, and renders twice for an unusable item
    detail: |
      cmd/define/cloze.go:197 copies play_loop.go:975; a word holding items but no usable cloze item
      renders its entry once in clozeAsk and again in ask. Extract a shared helper, or move the render
      below clozeFor's nil check (ARCH-DRY).
  - id: new
    severity: Minor
    family: dead-guard
    title: |
      usableItem's TrimSpace(Answer) == "" check is unreachable
    detail: |
      cmd/define/cloze.go:99 cannot fire: hasLetterOrDigit at line 95 already returned false for a
      blank answer.
  - id: new
    severity: Minor
    family: stale-artifact-restatement
    title: |
      the project's model-task table still attributes the distractor veto to #12
    detail: |
      workshop/projects/define-learn.md:75 reads "veto a distractor | #12", which the issue's
      2026-09-04 revision moved to #10 and this plan's scope note repudiates.
  - id: new
    severity: Minor
    family: stale-artifact-restatement
    title: |
      the plan's Integration points table names ReviewEvent.Flagged; the code ships ReviewEvent.Options
    detail: |
      store/event.go:93 is Options, with the flagged fact carried by Kind — better than the plan, but
      the table now names a field that does not exist. Same entry should note optionset_test.go was
      deliberately not created and that Tasks 5/6's tests landed in cmd/define/cloze_test.go.
```

---

## Re-review — 2026-09-07T11:41:55-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 12 — review form 2.2: cloze from current news with curated distractors |
| repo | tools |
| issue file | workshop/issues/000012-vocab-form-cloze.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..0698b277f1ca6cc9f2e1365d89493daaa075f6fd |
| command | sdlc close --issue 12 |
| reviewer | claude |
| timestamp | 2026-09-07T11:41:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four blocking Importants from round 1 are genuinely fixed and genuinely pinned — I reverted each fix in a scratch copy and every one reddened a named test with a legible message (BR-2's `TestAStrayDigitAfterAnsweringDoesNotRePick` names the moved pick; BR-1 reddens twice, once for `Keys()` and once for `gradedPromptFor`; BR-3's `TestAnUnreadableItemsFileIsReported` reports the silent read). `go test ./...`, `go vet ./...`, `go vet -tags conformance ./...` and `gofmt -l` are clean at HEAD, and the BR-2 fix is a strictly better design than the one I sketched: moving the gesture onto `Flagging` as `Flag(k rune)` removed the interception, the one-shot field and the carried state together. What holds it back from SHIP is that the *class* behind BR-1 was not swept: this repo already owns the mechanism that exists to stop a keys line drifting from the README (`doc_sync_test.go`), its own comment names the human residual, and the very next form added walked straight through it — neither of Cloze's two prompt lines appears in README.md, and the key table never lists `?`. Three prior families also recur, one of them as a regression the deleted code's own comment argued against.

## 1. Strengths

- **The BR-2 fix is better than the finding asked for, and it is pinned.** `Flagging.Flag(k rune)` (`play/session.go:556`) makes "is this a flag" and "is this an answer" different questions about a keystroke, so `Grade` never sees `?` at all. That deletes the `flagged` field, the one-shot contract, and the `(Graded, non-flag rune)` cell in one move — ARCH-ORDER's preferred shape. `TestAStrayDigitAfterAnsweringDoesNotRePick` (`play/cloze_test.go:232`) goes red the moment `q.Grade(in.Rune)` is put back into the graded branch; I checked.
- **BR-3's fix splits the branches at the right seam and adds the row that keeps it from becoming noise.** `cmd/define/cloze.go:190` warns on the error; `TestAWordWithNoItemsSaysNothing` (`cloze_test.go:494`) pins that an unharvested word stays silent. Both directions, which is what a "report it" fix usually skips.
- **BR-1's fix derives both lines from the form** — `Cloze.Keys()` and `gradedPromptFor(q)` (`play_loop.go:1291`) — rather than adding a second constant, and the non-flagging negative is asserted in the same test (`cloze_test.go:452`).
- **The Log now records the boundary honestly, including the part that went wrong.** The issue's `2026-09-07` entry says plainly that the first "hand-run" was a programmatic render, that neither party pressed `?`, and that booking the deviation did not prevent the failure it was booked to prevent. The pty transcript, the flag output, and the "no `correct:` field" check are all in the record.
- **Independently confirmed the graded-state flag works end to end.** A probe driving a wrong pick then `?` through `playSession` records exactly one flag with the full option set and one review — so the restructured graded branch is correct, only unpinned at the loop level.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `Cloze` was never enrolled in `doc_sync_test.go`, the mechanism that exists to catch exactly the bug BR-1 was.** `cmd/define/doc_sync_test.go:56` holds a `forms` slice of `Choice` and `Board`, with a comment reading *"TWO FORMS, not three"* and a stated residual: *"a form added to play and not added to this slice is not checked here. That half is human."* This diff added the third shipped form and the human half did not fire. Measured, not inferred — a probe enrolling `Cloze` fails today on both lines:

```
README.md does not contain "1-4 = pick the word, ? = bad question, d = remove from deck, Ctrl-C to stop"
README.md does not contain "any key = next word, ? = bad question, d = remove from deck, Ctrl-C to stop"
```

The README's cloze section (`cmd/define/README.md:41-65`) shows the four options with **no keys line at all**, unlike the 2.3 example at :84 which quotes one; the key table at :187-196 lists `1`–`4` as *"multiple choice: pick the definition"* and never mentions `?`; and the only graded prompt the README quotes (:205) is the non-flagging one, which is now the wrong line for the form the README says the tool *prefers*. `?` is documented in prose at :62, so this is not "undocumented" — it is "not derived, and the lines that are quoted are now incomplete."

> **This is the 2nd finding in family `prompt-line-matches-live-keys`.** Round 1 fixed the instance (the on-screen prompt). Do NOT fix this instance by hand-pasting two lines into README.md — state the rule and fix that. The rule is stateable: **every shipped `play` form's keys line and graded prompt are README consumers, enforced mechanically, not by remembering.** The concrete fix is to close the residual the test's own comment confesses to — enumerate the shipped forms in one place `todaysQuestions`-side code and `doc_sync_test.go` both derive from (or a guard that fails when a `Question` implementation in `play` is absent from the slice), add `Cloze` and `gradedPromptFor` to the pinned set, and update the README key table so the `?` row derives from `play.FlagKey`. `gradePrompt`'s doc comment already records the first instance of this class and `doc_sync_test.go`'s header records three more; this is the fourth, and it is the one where the general fix existed and was not extended.

## 4. Minor findings

- **`stale-artifact-restatement`, 3rd finding — three more restatements now contradict the code.** (a) `play/session.go:262-264` still says *"The keystroke reaches the form through Grade, which returns false for it … and the flag itself comes back out of advance beside the drop"* — both halves are false as of `0698b27`: `Grade` never sees `?`, and `advance` no longer emits `OutcomeFlag`. (b) `workshop/plans/000012-vocab-form-cloze-plan.md:211` declares `Flagging { Flagged() ([]string, bool) }`; the code ships `Flag(k rune)`. (c) `cmd/define/README.md:486` says `items/` — *"Nothing writes this yet — authoring is the next milestone"* — three paragraphs below the README's own *"It then writes the practice items."* (introduced by `#10 M1`, falsified by `#10 M2`, both inside this window). With BR-8 and BR-9 still open that is **five open instances in one family**. Do NOT fix these five one at a time: the rule is that a prose or comment restatement of a fact the code owns must either derive from it or be swept at the boundary that changed it. The enumerable class here is small and worth writing down as a close-time sweep list — the issue's own artifacts (`issue`, `plan`, `project`), the README, the atlas, and the doc comments on every symbol the diff renamed or re-shaped.
- **`one-place-renders`, 2nd finding — a regression, and the code's surviving comment argues against it.** `0698b27` deleted the `flaggedBy` block from `advance` and open-coded the identical six-line `OutcomeFlag` literal at `play/session.go:273-279` and `:409-415`. The deleted block's comment said it lived in `advance` because *"this is the one place both mark paths meet, so the question is put once rather than at two call sites that could drift"* — and the drop's version of that same comment is still there at `:498-503` (#42). So this is not a nit: the rule is written in the file, next to the code that now breaks it. Combined with BR-6's still-open `Render`+`marks` duplication, the rule to state and sweep is **one constructor per outcome/render, called from N sites — never N constructions**. A `flagOutcome(s, q, opts)` helper and BR-6's `renderInto(marks, key, entry)` are the same fix twice.
- **`boundary-record-unwritten`, 2nd finding — `workshop/lessons.md` carries nothing from close-review round 1.** AGENTS.md §4 says a code review adds rules to `lessons.md`. `46a769b` added the `#12` section (the fuzz, the unreachable pin, the small-rig row) but `0698b27` — which fixed two user-facing bugs and whose own commit body states the transferable rule (*"a programmatic render cannot see a prompt line"*, and the finding-under-the-finding that booking a deviation is not running it) — added none. The rule covering this and BR-4: **a boundary's durable record has more than one home** — the issue `## Log`, `lessons.md` when a review found something, the plan's `## Revisions`, and the project file — and "the record" is not written until all of them are. BR-8/BR-9 are the same rule seen from the artifact side, which is why this family and `stale-artifact-restatement` keep alternating.

## 5. Test coverage notes

The reverts hold: I independently reddened BR-1 (twice), BR-2 and BR-3 by undoing each fix in a scratch tree, so none of the three is a test written to assert whatever the fix happened to do. `TestFlagAnswersAboutTheRune` (`play/cloze_test.go:204`) is a good replacement for the deleted one-shot test — it asserts statelessness *and* that `Grade(FlagKey)` leaves `chosen` untouched, which is the property BR-2 was actually about. Two gaps, both one test each and neither hiding a defect (I verified both by probe): the flag **after a miss** is pinned in `play` (`TestAFlagIsHeardInEverySessionState/after_answering`) but still never driven through `playSession`, so the graded branch's second `OutcomeFlag` construction site has no loop-level pin — and it is now a *second* site, which is exactly when that matters; and `gradedPromptFor` is asserted only against its own wording, never against the README (I-1). `optionset_test.go` remains deliberately unwritten, which is coherent given Task 1's oracle was `#7`'s untouched suite — but it should be said in the plan rather than left to be discovered.

## 6. Architectural notes

- **ARCH-DRY — flag.** `optionSet`, the `blankOut → blankStem` delegation and `wordRunEnd` reusing `isWordRune` are three real consolidations. Against them, this round *added* a duplication (`OutcomeFlag` at two sites) and left BR-6's open. Both are the same rule; fix them together.
- **ARCH-PURE — pass.** `play` is still import-guarded pure; `blankStem`, `usableItem`, `clozeFor` and the two prompt helpers are pure and tested with no store, dictionary or model. `clozeAsk` is the only `Items()` call and is the thin seam; `warn io.Writer` was threaded rather than reaching for `os.Stderr`, which keeps it injectable.
- **ARCH-PURPOSE — flag.** The gesture is now discoverable *in the TUI*, which is the purpose. But the class behind the finding was not swept: the mechanism that makes keys lines derive (`doc_sync_test.go`) was left un-extended, and five restatement instances remain across the plan, project, README and a doc comment. Fixing the site a finding names while enumerable siblings stay in the tree is the instance, not the class.
- **ARCH-MOCK — pass.** No new external dependency. `CaptureFlag` landed as a `storetest` row held by both `Mem` and `YAML`, and `TestCaptureFlagWritesAFlagAndNotAReview` drives the *real* `storeCapturer` through `schedule.Fold` — the consumer, not the field. The offline claim is asserted with the model seam made to **panic**, not nil.
- **ARCH-CONSTRAINTS — pass.** One `Items()` read per due word, no model or network call on any sitting path, no new concurrency or unbounded loop. `blankStem`'s termination is now bounded by construction (`if end <= at { end = at + n }`) with the fuzz corpus committed, which is the right answer to a loop whose termination depended on the property under test. The only waste is the double `Render` (BR-6).
- **ARCH-SECURE — pass, improved this round.** `usableItem` is the typed floor at the on-disk boundary and enumerates the class rather than an example; `oneLine` at both `SetItems` and YAML `Items()` stops model-authored text forging a record in the day log; and BR-3's fix is precisely the ARCH-SECURE ask that a parse failure "degrade visibly rather than substituting a fabricated value." Nothing in the new warn path reaches a credential.
- **ARCH-ORDER — pass, and this round improved it.** Making the gesture a function of the rune rather than a remembered field removed a state from the constellation instead of adding one, and closed the `(Graded, non-flag rune)` cell by construction rather than by a test. What remains is pre-existing and not this boundary's: `Session` still carries `Graded`/`Revealed` as independent bools with the legal combinations unwritten; a tagged enum over `asking | revealed | graded` is what would make the next cell of this shape unrepresentable. Worth naming in `#13`'s plan, since it adds the fourth form.

## 7. Plan revision recommendations

Add one `## Revisions` entry to `workshop/plans/000012-vocab-form-cloze-plan.md` recording the four places the plan now claims something the code does not do:

1. **`ReviewEvent.Flagged` shipped as `ReviewEvent.Options`** (`store/event.go:93`), with the flagged fact carried by `Kind`. The code is better than the plan; record the rename so the Integration points table stops naming a field that does not exist. (BR-9, still open.)
2. **`Flagging` ships as `Flag(k rune) ([]string, bool)`, not `Flagged() ([]string, bool)`** — the plan's Task 3 Step 2 snippet at line 211 is the pre-review shape. Record *why*: the `Flagged()` shape forced the session to hand every rune to `Grade` to discover a flag, which re-picked the answer on a graded question (close review I-2). Same entry should note that the flag outcome is now built in `apply` rather than in `advance`, and that this is the ARCH-DRY debt above rather than a decision.
3. **`cmd/define/play/optionset_test.go` was deliberately not created** — Task 1 Step 4 requires no test file to change, so `#7`'s untouched suite is the oracle. Say so, so a reader does not go looking for it.
4. **Tasks 5 and 6 name `cmd/define/play_loop_test.go`; the pins landed in `cmd/define/cloze_test.go`** (`TestTheFormSelectionRule`, `TestAClozeSittingNeverReachesForTheModel`, `TestFlaggingAQuestionRecordsItWithoutScoringIt`, and the two prompt/warn rows added this round). Point the rows at where the tests actually live.

Also correct `workshop/projects/define-learn.md:75` (BR-8) as part of the family sweep rather than on its own — the row still attributes the distractor veto to `#12` after the 2026-09-04 revision moved it to `#10`, and line 257's `form 2.2` task is still unticked (the close gate ticks it, so that one needs no manual edit).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Keys() and gradedPromptFor both derive from the form; reverting each independently
      reddens a named test. The class-level gap (doc_sync enrollment) is raised separately.
  - id: BR-2
    disposition: addressed
    note: |
      Flagging now takes the rune, so Grade never sees the key; reverting reddens
      TestAStrayDigitAfterAnsweringDoesNotRePick, which names the moved pick.
  - id: BR-3
    disposition: addressed
    note: |
      err and len==0 split; reverting the warn reddens TestAnUnreadableItemsFileIsReported,
      and TestAWordWithNoItemsSaysNothing pins the silent ordinary case.
  - id: BR-4
    disposition: addressed
    note: |
      The 2026-09-07 Log entry records the sweep, the pty transcript, and that the first
      "hand-run" was a programmatic render. See the new lessons.md finding for the sibling.
  - id: BR-5
    disposition: not-addressed
    note: |
      capture.go:149 still omits Form: out.Form; set at two sites, read at zero.
  - id: BR-6
    disposition: not-addressed
    note: |
      cloze.go:207 still copies play_loop.go:975 and still renders above clozeFor's nil check.
  - id: BR-7
    disposition: not-addressed
    note: |
      cloze.go:101 TrimSpace(Answer)=="" is still unreachable behind hasLetterOrDigit.
  - id: BR-8
    disposition: not-addressed
    note: |
      define-learn.md:75 still reads "veto a distractor | #12".
  - id: BR-9
    disposition: not-addressed
    note: |
      No Revisions entry was added; the plan still names ReviewEvent.Flagged.
findings:
  - id: new
    severity: Important
    family: prompt-line-matches-live-keys
    title: |
      Cloze was never enrolled in doc_sync_test's forms slice, so neither of its prompt lines is a README consumer
    detail: |
      2nd in this family — round 1 fixed the instance, not the class. doc_sync_test.go:56
      pins Choice and Board and its own comment names the residual ("a form added to play
      and not added to this slice is not checked here. That half is human"); this diff added
      the third shipped form and skipped it. Probed: README.md contains neither
      "1-4 = pick the word, ? = bad question, ..." nor "any key = next word, ? = bad question,
      ...", and the key table at README.md:187 never lists `?`. Do not hand-paste the lines —
      enroll every shipped form by construction and make the key table's `?` row derive from
      play.FlagKey.
  - id: new
    severity: Minor
    family: stale-artifact-restatement
    title: |
      three more restatements now contradict the code, bringing the open family to five
    detail: |
      3rd in this family. play/session.go:262-264 still says the keystroke reaches the form
      through Grade and the flag comes out of advance — both false since 0698b27. The plan's
      Flagging snippet (line 211) still declares Flagged(). README.md:486 says items/ has
      "Nothing writes this yet" three paragraphs below the README's own "It then writes the
      practice items". With BR-8 and BR-9 open that is five instances. Fix the rule, not the
      five: a restatement of a fact the code owns must derive from it or be swept at the
      boundary that changed it — write the close-time sweep list (issue, plan, project,
      README, atlas, and the doc comments on every symbol the diff re-shaped).
  - id: new
    severity: Minor
    family: one-place-renders
    title: |
      the OutcomeFlag literal is now built at two call sites, against the rule the surviving drop comment states
    detail: |
      2nd in this family, and a regression. 0698b27 deleted flaggedBy from advance and
      open-coded the identical six-line OutcomeFlag at play/session.go:273-279 and :409-415.
      The deleted block's comment justified the single site ("the one place both mark paths
      meet ... rather than at two call sites that could drift"), and the drop's version of
      that comment is still at :498-503. With BR-6 open the rule to state is: one constructor
      per outcome/render called from N sites, never N constructions — a flagOutcome helper
      and BR-6's renderInto are the same fix twice.
  - id: new
    severity: Minor
    family: boundary-record-unwritten
    title: |
      lessons.md carries nothing from close-review round 1, whose commit body states the rule
    detail: |
      2nd in this family. 46a769b added the #12 lessons section; 0698b27 fixed two
      user-facing bugs and added none, though its own body states the transferable rule
      ("a programmatic render cannot see a prompt line"; booking a deviation is not running
      it). AGENTS.md section 4 requires it. The rule covering this and BR-4: a boundary's
      durable record has more than one home — issue Log, lessons.md when a review found
      something, the plan's Revisions, the project file — and it is not written until all of
      them are.
```
