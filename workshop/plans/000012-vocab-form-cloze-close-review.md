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
