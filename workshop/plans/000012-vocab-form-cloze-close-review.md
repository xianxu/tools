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

---

## Re-review — 2026-09-07T12:02:57-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 12 — review form 2.2: cloze from current news with curated distractors |
| repo | tools |
| issue file | workshop/issues/000012-vocab-form-cloze.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..188f269e67e98f7f25b34f84c1096d2587e8d038 |
| command | sdlc close --issue 12 |
| reviewer | claude |
| timestamp | 2026-09-07T12:02:57-07:00 |
| verdict | REWORK |

## Review

Scratch worktree removed, tree clean at the pinned head.

```verdict
verdict: REWORK
confidence: high
```

The form itself is well built — `optionSet` is a real DRY win, `blankStem` is fuzzed and its leak table is a property rather than a row list, the flag's whole signal path is pinned end-to-end through a real sitting, and round 2's BR-10 was fixed as the class (`TestEveryFormIsEnrolled` derives the extent from `play/*.go`; I mutation-verified both halves — un-enrolling `Cloze` reddens it by name, and dropping the flag from `Cloze.Keys()` reddens the README guard). What blocks SHIP is a confirmed answer leak the diff's own doc comment predicted and did not act on: `play_loop.go:234` registers a `RegionHeadword` click region at line 1, col 0, width `visibleCells(q.Word())` for **every** non-board form, on a premise only `Choice.Prompt()` satisfies. On a cloze that span covers the first 11 cells of the blanked sentence — I ran it: the region comes back `{Kind:headword Word:sycophantic Line:1 Col:0 Width:11}`, the tty is painted `\x1b[4mThe Times d\x1b[24mismissed the interviews as ___.`, and a click there plays the answer's pronunciation. Done-when says *"the blanked sentence never leaks the answer"*; the text does not leak it, the affordances around the text do. Six of round 2's nine Minors are also still open, untouched by the last commit.

## 1. Strengths

- **`optionSet` (`play/optionset.go`)** — the extraction is behaviour-preserving and `#7`'s suite is untouched, which was the refactor's declared oracle. The contract (digit past the end is a stray key, not a wrong answer) now lives once.
- **`blankStem` (`cloze.go:40`)** — the loop-must-advance guard at :66 with the `RuneError` explanation is the right shape, and `FuzzBlankStem` states the leak as a property rather than rows. Both fuzz corpus entries are committed as regressions.
- **`blankOut` now delegates (`harvest_judge.go`)** — one blanker, and the doc comment records that the original DRY argument was decided on a wrong cost estimate. That is the useful half of the lesson.
- **The flag's negative property is asserted through its consumer** — `capture_test.go:695` runs `schedule.Fold` over a flag-only log and demands zero words, which is what PQ-2 was actually about. I mutation-checked the graded-state arm: deleting the block at `session.go:271-280` reddens `TestAFlagIsHeardInEverySessionState/after_answering` by name.
- **BR-10's fix is the class, not the instance** — `docSyncForms(t)` is one source for both guards, and `TestEveryFormIsEnrolled` regexes the extent out of `play/*.go`. Verified red on un-enrolment.

## 2. Critical findings

**`cmd/define/play_loop.go:234` — the loop registers a headword click region over the cloze prompt, so clicking the blanked sentence speaks the answer.**

`writeRendered(stdout, "\n"+q.Prompt()+"\n", []Region{{Kind: RegionHeadword, Text: q.Word(), Word: q.Word(), Line: 1, Col: 0, Width: visibleCells(q.Word())}})` runs for every form the board branch did not return on. `addRegions` (`screen.go:165`) stores it without checking that `Text` is at those coordinates. `Cloze.Prompt()`'s own doc comment at `play/cloze.go:57-60` states the premise does not hold here — and nothing acts on that.

Confirmed by running it: region `{Kind:headword Word:sycophantic Line:1 Col:0 Width:11}`; tty paints `\x1b[4mThe Times d\x1b[24mismissed the interviews as ___.`; the click produced `♫ playing 1×` and one `fakePlayer` call. So three consequences, before the learner has answered: the answer is spoken aloud, an underline marks a span exactly as wide as the answer (the length leak `Blank`'s comment at `cloze.go:13-17` says `___` exists to prevent), and the underline sits on arbitrary text — which `playbar.go:228` already calls *"worse than no underline"*.

**This is the 2nd finding in family `capability-guard-too-wide`.** BR-2 was the graded branch guarding on `CanFlag(q)` rather than on the gesture; this is the same rule one layer out — a guard whose condition (`not a board`) is wider than the property it needs (`this prompt begins with its headword`). Do not fix the instance by special-casing `*Cloze`. The rule is already written down in this repo, at `play_loop.go:1170`: *"Located rather than counted from a formula: the form owns its own layout, and a formula here would be a second copy of it that a new form silently invalidates."* `marksIn` obeys it for the reveal; the prompt path is the copy that never got it, and `Cloze` is the new form that invalidated it. Fix: locate the span (`strings.Index` into the prompt, as `marksIn` does) or let the form declare its own prompt regions, and add the derived guard the rule implies — over `docSyncForms(t)`, assert every registered `Region.Text` actually occupies its claimed `Line`/`Col`/`Width` in the written text. That guard is what makes the enumeration mechanical instead of one more form-by-form sweep.

## 3. Important findings

**`cmd/define/store/item.go:181` — item free text reaches the raw terminal with only whitespace collapsed.**

`oneLine` is `strings.Join(strings.Fields(s), " ")`, which removes newlines but not ESC, BEL, or any other C0/C1 control. `Cloze.Prompt()` is the first path that renders `Item.Stem`/`Answer`/`Distractors` to a terminal (before #12 they only reached YAML files and LLM prompts). Probed through the real store: `SetItems` then `Items` returns `"The aide was sycophantic\x1b[2J\x1b[H to a fault."` and `"ephemeral\a"` unchanged, and `clozeFor` renders `"The aide was ___\x1b[2J\x1b[H to a fault.\n\n1  keel\n2  ephemeral\a\n..."` — `\x1b[2J\x1b[H` clears the alt screen mid-sitting. The provenance is model output plus a directory the README documents as inspectable and hand-editable, so this is ARCH-SECURE's "input from outside the process treated as well-formed". `event.go:103` states the options "are neutralised at the store's write (sanitiseItem)", which is true of newlines and of nothing else. Fix in `oneLine` (drop `unicode.IsControl` runes before joining), which is the one place `sanitiseItem`'s own comment says every consumer shares, and pin it with a store-level test so `Mem` and `YAML` are both held.

**`cmd/define/README.md:199-209` — the key table still never lists `?`.**

**This is the 3rd finding in family `prompt-line-matches-live-keys`.** BR-1 fixed the two prompt lines; BR-10 fixed the enrolment that checks them; the table a reader consults for "what can I press" is the third home of the same fact and is still hand-maintained — its `1`–`4` row also says only "pick the definition" now that a second form grades digits. Do not hand-paste a `?` row. The rule: **every place that enumerates live keys must derive from the code that owns them, and a new key is not shipped until every such enumeration derives.** The prompt lines already derive via `gradePrompt`/`gradedPromptFor`; the table does not, and it is the enumeration the README's own "| key | does |" header promises is complete. Make the row derive from `play.FlagKey` + `CanFlag`, the same move `TestREADMENamesEveryFallbackReason` makes for `fallbackReasons`.

## 4. Minor findings

Six of round 2's Minors are re-raised unchanged below in the `dispose` block (BR-5, BR-6, BR-7, BR-8, BR-9, BR-11, BR-12) — nothing new to add beyond BR-11's fourth site: `README.md:487` still lists the event kinds as "looked-up, asked, reviewed", and `flagged` is now a fourth.

## 5. Test coverage notes

- `go test ./cmd/define/...` green; `go vet ./...` and `go vet -tags conformance ./...` clean; `gofmt -l` clean.
- Two mutations confirmed the round-2 claim and the round-1 pin (recorded in Strengths). A third — deleting the graded-state flag block — reddens by name.
- **The hole the Critical fell through:** no test observes the *prompt's* registered regions for any form. `play_loop_test.go:1027` asserts `live.RegionAtRow(1, 0)` is offered, but only for a `Choice`, and only that *something* is there. The derived guard proposed above closes it for every enrolled form at once.
- No test asserts that item text reaching the TUI is control-character free; the store's sanitisation tests cover whitespace only.

## 6. Architectural notes

- **ARCH-DRY** — flag, twice, both already open: `clozeAsk` duplicates `ask`'s render+marks block (BR-6) and the `OutcomeFlag` literal is constructed at `session.go:273` and `:409` (BR-12), against the rule the surviving drop comment at `:498` states. Wins on the other side: `optionSet`, and `blankOut` delegating.
- **ARCH-PURE** — pass. `play` stays mechanically pure; `blankStem`/`usableItem`/`clozeFor` are pure and unit-tested with no store, dictionary or model; `clozeAsk` is the thin seam.
- **ARCH-PURPOSE** — flag. The shadow-sweep for "never leaks the answer" enumerates the channels the answer can reach the learner through: the stem text (swept, fuzzed), the reveal (correct), the click affordance (**leaks — audio**), and the underline width (**leaks — length**). Only the first was swept.
- **ARCH-MOCK** — pass. No new external dependency; `TestAClozeSittingNeverReachesForTheModel` makes the seam panic rather than nil, and the flag's store promise landed in `storetest/suite.go` so `Mem` and `YAML` are both held.
- **ARCH-CONSTRAINTS** — pass with a note. No model or network call, asserted. The plan calls the per-word `Items()` read "the same order as the existing `Deck()` and `Events()` reads"; it is O(due words) file opens against O(1), bounded by `-count`, so the conclusion holds even though the stated reason does not.
- **ARCH-SECURE** — flag; see the `oneLine` finding.
- **ARCH-ORDER** — pass. The three states `?` can arrive in are enumerated in the plan, implemented outside the any-key-advances branch for the reason `InputDrop` sits outside it, and each state has a named test row. The flag advances via `advance(Skipped)` so no in-flight effect is dropped.

## 7. Plan revision recommendations

- BR-9's entry (still owed): the Integration points table names `ReviewEvent.Flagged`; the code ships `ReviewEvent.Options` with the flagged fact carried by `Kind`. Same entry should record that `optionset_test.go` was deliberately not created and that Tasks 5/6's tests landed in `cmd/define/cloze_test.go`, not `play_loop_test.go`.
- BR-11's entry (still owed): the `Flagging` snippet at plan line 211 still declares `Flagged()`; the shipped interface is `Flag(k rune)`.
- New: Task 2's `Cloze` entry should record that a form whose prompt does not begin with its headword invalidates the loop's prompt-region formula — the plan named the reveal's region handling and never the prompt's, which is how the Critical shipped with a doc comment describing it.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      capture.go:145-149 still omits Form; Outcome.Form is set at session.go:277 and :412 and read at zero sites.
  - id: BR-6
    disposition: not-addressed
    note: |
      cloze.go:208 still renders before clozeFor's nil check at :213, so an items-holding word with no usable cloze item renders twice.
  - id: BR-7
    disposition: not-addressed
    note: |
      cloze.go:101 still unreachable behind hasLetterOrDigit at :98.
  - id: BR-8
    disposition: not-addressed
    note: |
      workshop/projects/define-learn.md:75 unchanged; the project file received no edit at all in this window.
  - id: BR-9
    disposition: not-addressed
    note: |
      plan lines 92 and 101 still name ReviewEvent.Flagged; store/event.go:93 ships Options.
  - id: BR-10
    disposition: addressed
    note: |
      Mutation-verified both halves; the key table's residual `?` row is raised separately as a rule-level finding.
  - id: BR-11
    disposition: not-addressed
    note: |
      All three sites stand (session.go:262-264, plan:211, README:498), plus a fourth: README:487 lists three event kinds and flagged is now a fourth.
  - id: BR-12
    disposition: not-addressed
    note: |
      The six-line OutcomeFlag literal is still built at session.go:273-279 and :409-415.
  - id: BR-13
    disposition: addressed
    note: |
      lessons.md now carries round 2 and the multi-home rule; that rule's own project-file home is still unwritten, which BR-8 tracks.
findings:
  - id: new
    severity: Critical
    family: capability-guard-too-wide
    title: |
      the loop registers a headword click region over the cloze prompt, so clicking the blanked sentence speaks the answer and underlines a span as wide as it
    detail: |
      play_loop.go:234 registers RegionHeadword at Line 1, Col 0, Width visibleCells(q.Word()) for
      every form the board branch did not return on, on a premise only Choice.Prompt() satisfies;
      addRegions (screen.go:165) does not check Text is at those coordinates. Ran it: region
      {Kind:headword Word:sycophantic Line:1 Col:0 Width:11}, tty paints
      "\x1b[4mThe Times d\x1b[24mismissed the interviews as ___.", the click yields "♫ playing 1×"
      and one fakePlayer call. Three leaks before answering: the answer spoken, its length shown as
      an underline (what Blank's comment at cloze.go:13-17 exists to prevent), and the underline on
      arbitrary text (playbar.go:228 calls that worse than none). play/cloze.go:57-60 states the
      premise fails here and nothing acts on it. 2nd in this family after BR-2 — same rule, a guard
      wider than the property it needs. Do not special-case *Cloze: the rule is already at
      play_loop.go:1170 ("the form owns its own layout, and a formula here would be a second copy
      that a new form silently invalidates"). Locate the span as marksIn does, or let the form
      declare its prompt regions, and add the derived guard over docSyncForms(t) asserting every
      registered Region.Text occupies its claimed Line/Col/Width.
  - id: new
    severity: Important
    family: untrusted-text-reaches-output
    title: |
      item free text reaches the raw terminal with only whitespace collapsed, and the event log's comment claims otherwise
    detail: |
      store/item.go:181 oneLine is strings.Join(strings.Fields(s), " ") — newlines go, ESC and BEL
      do not. Cloze.Prompt() is the first path putting Item.Stem/Answer/Distractors on a terminal.
      Probed through the real store: Items() returns
      "The aide was sycophantic\x1b[2J\x1b[H to a fault." and "ephemeral\a" unchanged, and clozeFor
      renders both into the prompt; \x1b[2J\x1b[H clears the alt screen mid-sitting. Provenance is
      model output plus a directory the README documents as inspectable and editable, so ARCH-SECURE
      applies. store/event.go:103 asserts the options are "neutralised at the store's write
      (sanitiseItem)", true of newlines only. Fix in oneLine (drop unicode.IsControl runes) — the one
      place sanitiseItem's own comment says every consumer shares — and pin it in storetest so Mem
      and YAML are both held.
  - id: new
    severity: Important
    family: prompt-line-matches-live-keys
    title: |
      the README key table still lists no `?`, and its digit row still says only "pick the definition"
    detail: |
      README.md:199-209 is the table a reader consults for what they can press, and its header
      promises completeness. 3rd in this family: BR-1 fixed the prompt lines, BR-10 fixed the
      enrolment that checks them, and this third home of the same fact is still hand-maintained.
      Do not hand-paste a row. The rule: every enumeration of live keys must derive from the code
      that owns them, and a new key is not shipped until every such enumeration derives — the prompt
      lines already do via gradePrompt/gradedPromptFor, this does not. Derive the `?` row from
      play.FlagKey + CanFlag and widen the digit row now that a second form grades digits, the same
      move TestREADMENamesEveryFallbackReason makes for fallbackReasons.
```

---

## Re-review — 2026-09-07T12:23:35-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 12 — review form 2.2: cloze from current news with curated distractors |
| repo | tools |
| issue file | workshop/issues/000012-vocab-form-cloze.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..2c67482163c1752ddac57dfd663406c526f0ebe0 |
| command | sdlc close --issue 12 |
| reviewer | claude |
| timestamp | 2026-09-07T12:23:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3's three findings are genuinely fixed and genuinely pinned — I reverted each in a scratch copy at HEAD and watched the named test go red (BR-14: `TestAPromptRegionCoversTheTextItClaims` reddens for `*play.Cloze` **and** `*play.Board`; BR-15: both `TestMemConformance` and `TestYAMLConformance` redden on `\x1b`/`\a`; BR-16: `TestREADMEKeyTableNamesEveryLiveKey` reddens by name). `go build`, `go vet`, `gofmt -l` and `go test ./...` are clean. What blocks SHIP is not new code: the seven Minors carried since round 1 are still open verbatim, and two of them are the *only* remaining instances of families the prior rounds asked to be closed as a class. Beyond that I found one new Important: the mechanical extent that BR-10's and BR-14's fixes both rest on is a regex over source text that fails **open** — I defeated it two independent ways in a scratch copy (a two-letter receiver name; a three-line `Form()` body), each leaving all four doc/region guards green with `Cloze` unenrolled. That is BR-10's exact damage, restored by a formatting choice.

## 1. Strengths

- **`promptRegions` is the right shape for the class, not a special case for the form that broke.** `cmd/define/play_loop.go:1204-1224` issues the region only when its own claim is true (`line 0 begins with the headword`), and `TestAPromptRegionCoversTheTextItClaims` (`play_loop_test.go:4269`) reads every enrolled form's coordinates back out of the text actually written via the new cell-accurate `cellSlice` (`render.go:642`). The comment's refusal to "just search the prompt for the word" is correct and load-bearing — a cloze prompt contains its answer among the options.
- **The guard found a second instance the moment it existed.** My revert showed it reddens for `*play.Board` too, which is the evidence that it was sized to the class rather than to the reported bug.
- **BR-15 was fixed at the one place `sanitiseItem`'s own comment says every consumer shares** (`store/item.go:200-207`), and the storetest row now asserts over `unicode.IsControl` rather than over `"\r\n"` — with a companion assertion that `a\nb` still collapses to `a b` rather than `ab`, which is the trap the obvious fix falls into.
- **The `optionSet` extraction is behaviour-faithful.** `git diff` on `choice.go` shows `Grade`, `Keys`, `Reveal` and `MissedAxis` re-expressed over `correctIndex`/`wrongPick`/`keysFor` with no test file edited — the plan's stated oracle for Task 1 held.
- **The leak Done-when is pinned three ways, not one**: the six-row table (`cloze_test.go:22`), the standalone property (`:74`), and `FuzzBlankStem` with the hang and the non-word-answer regressions committed as seeds.

## 2. Critical findings

None.

## 3. Important findings

**I-A — `TestEveryFormIsEnrolled`'s derivation fails open; three guards silently lose a form.**
`cmd/define/doc_sync_test.go:136` derives the extent with
`regexp.MustCompile("func \\([a-z] \\*[A-Za-z]+\\) Form\\(\\) string \\{ return \"([a-z]+)\" \\}")`.
That requires four incidental things at once: a *single-letter* receiver name, a *pointer* receiver, a *one-line* body, and an all-lowercase return literal. Any of them failing drops the form from `declared`, and the test only asserts `declared ⊆ enrolled` — so a missing member is silence, not a failure. Measured in a scratch copy at HEAD:

- rename `func (c *Cloze) Form()` → `func (cz *Cloze) Form()` **and** delete `Cloze` from `docSyncForms` → `TestEveryFormIsEnrolled`, `TestREADMEQuotesThePromptsTheLoopActuallyPrints`, `TestREADMEKeyTableNamesEveryLiveKey` and `TestAPromptRegionCoversTheTextItClaims` are all **green**;
- reformatting `Form()` onto three lines (still `gofmt`-clean) does the same.

Un-enrolling alone *does* redden, so the issue Log's mutation claim is accurate — the weak link is the derivation, not the enrolment. This matters more than an ordinary doc guard because BR-14's Critical fix now hangs off the same extent. Fix sketch: replace the regex with `go/parser` + `ast.Inspect` over `play/*.go`, matching any `FuncDecl` with a receiver, `Name == "Form"`, and a single `string` result, taking the returned `BasicLit` — receiver name, pointer-ness and body layout all stop mattering. `numRegionKinds` is the model this is reaching for, and a sentinel is airtight where a source scrape is not.

**I-B — the close-time sweep list BR-11 asked for was not written, and the family grew from 5 instances to 11.**
**This is the 4th finding in family `stale-artifact-restatement`.** Earlier rounds fixed instances; BR-8, BR-9 and BR-11 are all still open verbatim at HEAD, and this round's own commits added three more. Do not fix these eleven — write the rule and the enumeration it implies. The rule BR-11 already stated is right: *a restatement of a fact the code owns must derive from it, or be swept at the boundary that changed it.* What is missing is the enumeration — the close-time sweep list over **issue, plan, project, README, atlas, and the doc comment on every symbol the diff reshaped**. Measured prevalence at `2c67482`, so the list has something to be checked against:

| # | site | what it still claims | prior |
|---|---|---|---|
| 1 | `play/session.go:262-264` | the keystroke reaches the form "through `Grade`" and the flag "comes back out of `advance`" — both false since `0698b27` | BR-11 |
| 2 | `plans/000012-…-plan.md:211` | `Flagging` declares `Flagged() ([]string, bool)`; the code ships `Flag(k rune)` | BR-11 |
| 3 | `cmd/define/README.md:499` | `items/` — "Nothing writes this yet — authoring is the next milestone" | BR-11 |
| 4 | `projects/define-learn.md:75` | "veto a distractor \| `#12`" — moved to `#10` by the 2026-09-04 revision | BR-8 |
| 5 | `plans/000012-…-plan.md:92,101` | `ReviewEvent.Flagged`; the code ships `ReviewEvent.Options` | BR-9 |
| 6 | `atlas/define.md:2579-2581` | "**The prompt word is line 1, column 0** … Both forms put the headword on their first line" — this is the *exact* premise `2c67482` deleted from the code, left standing in the atlas by that same commit, and "both forms" is now three | **new** |
| 7 | `cmd/define/README.md:491` | `events/` — "kinds: looked-up, asked, reviewed"; `#12` shipped `flagged` and an `options:` field, and this block is the only doc a human reading the log has | **new** |
| 8 | `atlas/define.md:611` | store layout — "kinds: looked-up, asked"; missing `reviewed`, `flagged`, and `#10`'s `facts/` and `items/` dirs | **new** |
| 9 | `play/session.go:158-160` | `Outcome.Form` is set "on every Record outcome … Set in ONE place — see `Apply`"; the two flag sites set it directly on a non-Record kind | **new** |
| 10 | `play/optionset.go:40` | "`#38`, which **marks** the option lines clickable" — the pre-`#12` text read "which **will** mark"; the Task 1 extraction flipped a plan into a false statement of fact, and only `RegionHeadword`/`RegionOriginLang` exist | **new** |
| 11 | `play/choice.go:106` | "`recall.go:29` is its premise" — `play/recall.go` was deleted with form 2.1 | pre-existing |

Rows 6, 9 and 10 were introduced or left by this round's own commits, which is the measurement that says the sweep is not happening by attention. Rows 6, 7 and 8 are also the **docs gate**: the atlas restates a premise the diff removed, and the README's file-format block does not document the new `flagged` kind or `options:` field that `#12` persists.

## 4. Minor findings

All seven carried Minors are re-raised unchanged; see the `dispose` block. In brief: `CaptureFlag` drops `Outcome.Form` (BR-5 — and `ReviewEvent.Form`'s own doc says an absent form means "some earlier form", so every flagged event now reads as a lie by that comment's rule); `clozeAsk` duplicates `ask`'s render+marks block and renders twice for a word whose only item is unusable (BR-6); `usableItem`'s `TrimSpace(Answer) == ""` at `cloze.go:101` is unreachable behind `hasLetterOrDigit` (BR-7); BR-8/BR-9/BR-11 fold into I-B above; the `OutcomeFlag` literal is still open-coded at `play/session.go:273-279` and `:409-415`, against the rule the surviving `advance` comment at `:498-503` still states (BR-12).

## 5. Test coverage notes

Coverage is the strongest part of this boundary. The flag is driven end-to-end through a real `playSession` rather than by calling the verb (`cloze_test.go:376`), the no-model-call pin makes the seam **panic** rather than nil, the three session states `?` can arrive in are each exercised (`play/cloze_test.go:135`), and `TestAStrayDigitAfterAnsweringDoesNotRePick` pins BR-2's regression directly. Two gaps worth naming, neither blocking:

- **The extent behind the doc/region guards is not itself pinned** (I-A). A `TestTheFormExtentDerivationSeesEveryForm`-style check — e.g. assert `len(declared) == len(docSyncForms(t))` — would fail closed and cost one line.
- **Reveal regions have no counterpart to `TestAPromptRegionCoversTheTextItClaims`.** `marksIn` *locates* rather than computes, so it is structurally sound and no leak follows (the reveal shows the word anyway), but the rule BR-14 established — a region must cover the text it claims — is currently enforced on only one of the two region paths.

## 6. Architectural notes

- **ARCH-DRY — flag.** `promptRegions` and `oneLine` each consolidate correctly, but BR-6 and BR-12 are both still open and are the same fix twice: one constructor per outcome, one helper per render, called from N sites rather than N constructions. `play/session.go:498-503` still carries the comment arguing for exactly that, four hundred lines below the two call sites that violate it.
- **ARCH-PURE — pass.** `Cloze`, `optionSet`, `blankStem`, `usableItem`, `clozeFor` and `hasLetterOrDigit` are pure and unit-tested with no store, dictionary or model; `play`'s empty-allowlist purity guard still holds; `clozeAsk` is the thin IO seam and reports its one read failure rather than swallowing it.
- **ARCH-PURPOSE — flag.** The issue's purpose is delivered. The failure is on the finding axis: BR-11 named the class and asked for the enumeration; the round fixed BR-14/15/16 as classes (well) and answered BR-11 with nothing, so the family grew (I-B). Note the contrast — BR-16's fix *did* sweep its class properly; I grepped `atlas/define.md` for key-line restatements and found none, so the README table and the prompt lines really are the whole enumeration for keys.
- **ARCH-MOCK — pass.** The flagged-event promise is in the `storetest` suite, so `Mem` and `YAML` are both held; BR-15's fix reddens both. The cloze sitting asserts no model call with a panicking seam rather than a nil one.
- **ARCH-CONSTRAINTS — pass.** The plan's envelope (one `Items()` read per due word, no model, no network) is what the code does. `promptRegions` runs once per question, not per frame. The only repeated work is BR-6's double `Render`.
- **ARCH-SECURE — pass, with one named coupling.** Provenance is stated correctly (model output in a directory the README documents as editable), the fix is at the store boundary, and both `Mem` (write) and `YAML` (read) sanitise. The residual: `AppendEvent` does not neutralise `ReviewEvent.Options` at its own boundary — safety comes from the items surface upstream. True today, and `store/event.go:100-102` names the dependency, so this is a documented coupling rather than a hole.
- **ARCH-ORDER — pass.** The flag enters `apply` as an explicit `(state, event) -> (state, effects)` arm in each of the three states, asked *before* the graded any-key rule for the same stated reason `InputDrop` and `InputQuit` sit outside it, and the tests observe all three interleavings rather than one. Nothing concurrent was added.

## 7. Plan revision recommendations

- **`## Revisions` — round 3 of the close gate.** The plan has entries for plan-quality round 1 and close round 2 but nothing for round 3. Add one recording that a click region issued from a form-specific layout formula was a Critical the plan never anticipated, and that "the form owns its own layout" (already stated for `marksIn`) now governs prompt regions too.
- **Core concepts → Integration points table.** `ReviewEvent.Flagged` must become `ReviewEvent.Options`, with a note that the flagged *fact* is carried by `Kind` — better than the plan, but the table currently names a field that does not exist (BR-9).
- **Task 3's `Flagging` snippet.** `Flagged() ([]string, bool)` → `Flag(k rune) ([]string, bool)`, with the reason the shipped shape is better (asking the rune keeps `Grade` from ever seeing the gesture).
- **Task 1's file list.** `optionset_test.go` was deliberately not created; Tasks 5/6's tests landed in `cmd/define/cloze_test.go`. Say so rather than leaving the plan promising a file.
- **Task 7.** Record that the doc-guard extent is *derived* rather than written, and that the derivation must fail closed (I-A) — the round-2 revision states the first half and not the second.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      capture.go:148-150 still writes Word/Kind/Found/Options/At; Outcome.Form is set at session.go:277 and :413 and read nowhere, so every flagged event has an empty form — which ReviewEvent.Form's own doc says means "some earlier form".
  - id: BR-6
    disposition: not-addressed
    note: |
      cloze.go:208 still renders and sets marks[key] before clozeFor's nil check, duplicating play_loop.go:962.
  - id: BR-7
    disposition: not-addressed
    note: |
      cloze.go:101's TrimSpace(it.Answer) == "" is still unreachable behind hasLetterOrDigit at :99.
  - id: BR-8
    disposition: not-addressed
    note: |
      projects/define-learn.md:75 still reads "veto a distractor | #12". Rolled into the family finding below.
  - id: BR-9
    disposition: not-addressed
    note: |
      plan lines 92 and 101 still name ReviewEvent.Flagged; store/event.go:103 ships Options. Rolled into the family finding below.
  - id: BR-11
    disposition: not-addressed
    note: |
      All three instances stand (session.go:262-264, plan:211, README.md:499) and the close-time sweep list the finding asked for was not written; the family measured 11 instances at HEAD.
  - id: BR-12
    disposition: not-addressed
    note: |
      The six-line OutcomeFlag literal is still open-coded at session.go:273-279 and :409-415.
  - id: BR-14
    disposition: addressed
    note: |
      Verified by revert: deleting the HasPrefix check in promptRegions reddens TestAPromptRegionCoversTheTextItClaims for both *play.Cloze and *play.Board, plus TestAClozePromptOffersNoHeadwordToClick.
  - id: BR-15
    disposition: addressed
    note: |
      Verified by revert: removing the strings.Map from oneLine reddens TestMemConformance and TestYAMLConformance on the ESC/BEL fixtures in both implementations.
  - id: BR-16
    disposition: addressed
    note: |
      Verified by revert: restoring the old digit row and dropping the `?` row reddens TestREADMEKeyTableNamesEveryLiveKey by name for both of Cloze's pairs.
findings:
  - id: new
    severity: Important
    family: derived-extent-fails-open
    title: |
      the forms extent is scraped by a regex that silently under-derives, so BR-10's and BR-14's guards both lose a form to a formatting choice
    detail: |
      doc_sync_test.go:136 requires a single-letter receiver, a pointer receiver, a
      one-line body and an all-lowercase return literal, all at once; the test only
      asserts declared is a subset of enrolled, so a member the regex misses is
      silence. Measured at HEAD in a scratch copy: renaming the receiver to `cz`
      (or reformatting Form() onto three lines, still gofmt-clean) AND removing
      Cloze from docSyncForms leaves TestEveryFormIsEnrolled,
      TestREADMEQuotesThePromptsTheLoopActuallyPrints,
      TestREADMEKeyTableNamesEveryLiveKey and TestAPromptRegionCoversTheTextItClaims
      all green — BR-10's damage restored, now also carrying BR-14's Critical guard.
      Un-enrolling alone does redden, so the enrolment is fine and the derivation is
      the weak link. Fix: derive with go/parser + ast.Inspect (any FuncDecl with a
      receiver, Name == "Form", one string result, returning a BasicLit), or make it
      fail closed by asserting len(declared) == len(docSyncForms(t)).
  - id: new
    severity: Important
    family: stale-artifact-restatement
    title: |
      the close-time sweep list BR-11 asked for was never written, and the family grew from 5 open instances to 11 — three of them added by this round's own commits
    detail: |
      This is the 4th finding in family stale-artifact-restatement. Earlier rounds
      fixed instances; BR-8, BR-9 and BR-11 all stand verbatim at HEAD. Do NOT fix
      these eleven sites. The rule BR-11 stated is correct — a restatement of a fact
      the code owns must derive from it or be swept at the boundary that changed it —
      and what is missing is the ENUMERATION: a close-time sweep list over issue,
      plan, project, README, atlas, and the doc comment on every symbol the diff
      reshaped. Measured at 2c67482: (1) session.go:262-264 keystroke-through-Grade;
      (2) plan:211 Flagged(); (3) README.md:499 "Nothing writes this yet";
      (4) projects/define-learn.md:75 veto attributed to #12; (5) plan:92,101
      ReviewEvent.Flagged; (6) NEW atlas/define.md:2579-2581 restates the exact
      "prompt word is line 1, column 0 / both forms put the headword on their first
      line" premise that 2c67482 deleted from the code, and there are now three forms;
      (7) NEW README.md:491 events block still reads "kinds: looked-up, asked,
      reviewed" with no `flagged` and no `options:` field, and it is the only doc a
      human reading the log has; (8) NEW atlas/define.md:611 store layout reads
      "kinds: looked-up, asked", missing reviewed, flagged, facts/ and items/;
      (9) NEW session.go:158-160 says Outcome.Form is set "in ONE place — see Apply"
      while :277 and :413 set it directly on a non-Record kind; (10) NEW
      play/optionset.go:40 — the pre-#12 text read "#38, which WILL mark the option
      lines clickable" and the Task 1 extraction flipped it to "which marks", a
      false statement of fact (only RegionHeadword and RegionOriginLang exist);
      (11) pre-existing play/choice.go:106 cites recall.go:29, deleted with form 2.1.
      Rows 6, 7 and 8 are also the docs gate: the atlas restates a removed premise and
      the README does not document the flagged kind or options: field #12 persists.
```
