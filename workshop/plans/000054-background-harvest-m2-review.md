# Boundary Review — tools#54 (milestone M2)

| field | value |
|-------|-------|
| issue | 54 — the TUI keeps practice material current in the background, so new words get cloze questions without a command |
| repo | tools |
| issue file | workshop/issues/000054-background-harvest.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 63a1a379b29f1f3ead275a08e526aa9ad23171b5..8295817728267f2fb55f56f208b5967b32b33a4c |
| command | sdlc milestone-close --issue 54 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-13T00:54:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M2 delivers what the issue's Spec and the plan's Task 2.x rows promise, and the code holds up under inspection rather than under the commit messages. The reflect core is a clean extraction with `runReflect`'s guards and prints unchanged, the job writes the learner model before the harvest that reads it, both triggers (none at the 12-word floor, refresh when deck lookups double) are pure table-tested functions, and the M1 round-2 Minors are fixed as rules (`markUnfinished`, `stopMeans`, `errDeckIO`) rather than as sites. I independently re-ran 17 mutations in a scratch worktree at head; every one turned its named test red, including the harvest-before-reflect reorder and the miss-counting guard in the loop. `go build`, `go vet`, `gofmt`, the full `cmd/define` suite, and a `-race` run over the session tests are all green. Nothing blocks the boundary; the findings below are Minor and two of them are plan-prose drift.

**Inspection log.** stat, name-status, full patch on all 11 files, the issue's Spec/Plan/Log, the plan's Core concepts, state table, envelope and Revisions, `replraw.go` wiring at the three `applyBg` sites, `deckperm.go` methods, `foldLookups`/`DeckLookups`, `renderUserModel`'s window line, and the test rigs. Mutations (all red): `reflectDue` drops the factor; `reflectDue` makes a zero-lookup model due; `stopMeans` drops the `errDeckIO` case; `stepBackground` ignores `deckErr`; the job ignores `reflectFailed`; the runner forgets `reflectFailed`; `markUnfinished` counts a budget cut; `modelLookups` accepts an unterminated frontmatter; harvest `WordFacts` error untyped; harvest `SetWordFacts` error untyped; reflect `SetUserModel` error untyped; the job reflects only when no model exists (no refresh); the job reads a loud store; the reflected notice dropped; the job harvests before it reflects; the loop counts a miss.

### 1. Strengths

- `cmd/define/reflect.go:387-452` — `reflectDeck` starts at the model call and `runReflect` keeps every guard, message and order; the existing reflect tests are the byte-faithfulness guard and they pass unchanged. Good ARCH-DRY: one core for the flag and the job.
- `cmd/define/background.go:398-424` — `modelLookups` refuses a number outside a terminated frontmatter and the fuzz asserts the count it reports is one the `window:` line states. This is ARCH-SECURE done right for a file the README invites the user to hand-edit: an unreadable model means no paid call, never a fabricated count.
- `cmd/define/background.go:381-391` — the "doubling nothing is no growth" row (`lookups > recorded`) is exactly the kind of cliff a refresh factor hides, and it is pinned in `TestReflectDue`.
- `cmd/define/harvest.go:270-280` and `background.go:435-446` — `markUnfinished` and `stopMeans` answer the round-2 families with a rule each rather than a site each. Every give-up site in `harvestDeck`/`runAuthoring` goes through the one function, and I checked the two `errBudget` breaks still precede the returns that would mark.
- `cmd/define/background_loop_test.go:326-359` — the end-to-end reflect test asserts ordering by request index against the fake, not by a stubbed trigger. That is the Done-when bullet, verified as written.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- `background.go:127-135` — `bgNoticeFor` returns early on `noModel` or `deckErr`, so a job that wrote the learner model and then hit a 429 on its first band call, or a store error in `pendingWords`, hides "learner model updated" even though the file is on disk. 2nd in `notice-wording-matches-cause`; rule stated below.
- Plan prose and tables lag the code in four places: the state-machine table has no `deckErr` row; the `bgJobResult` bullet lacks `deckErr`/`reflectFailed`; the `runBackgroundJob(ctx, d, skip map[string]bool)` and `bgRunner` "tried set" prose predate `bgMemory`; the envelope's reflect-input row still reads as per reflect while the Revisions entry says per check. 2nd in `plan-table-classification-matches-code`.
- The plan's ARCH-ORDER "second actor" row covers items and facts but not the learner model, which is now written by a background actor while the README invites the operator to edit it by hand. The consequence is bounded (splice keeps whatever Corrections are on disk; an editor save that clobbers a refresh leaves the old count, so the next check refreshes again, one wasted call) but the row should say so. 2nd in `shared-state-across-the-job-boundary`.
- `background.go:203-215` — once the deck is at the floor, every check reads the whole event log to decide "due", and a reflecting job reads `UserModel()` twice (job, then `reflectDeck`). Background-only and acknowledged in Revisions, but the Events read is wasted whenever the model exists and `modelLookups` is unknown. 2nd in `repeated-reads-per-job`.

### 5. Test coverage notes

- Every rule introduced this boundary has a mutation that turns it red; I reproduced the plan's Task 2.4 claims plus five of my own. The `-race` run over the session tests is clean.
- `TestAStoreErrorStopsTheSessionOnce` pins "once" via `stepBackground` rather than a session run; that is fine because `bgOff` ignoring events is already pinned in the table test.
- Not pinned: the notice-drop case in Minor 1 (a job that both reflected and then stopped). A one-row addition to `TestStepBackgroundTransitions` or a `bgNoticeFor` table would cover it.
- `markUnfinished`'s budget guard is unreachable from production because every site breaks on `errBudget` first; it is defense in depth for the rule, and the unit pin is the right level for it.

### 6. Architectural notes

- ARCH-DRY: pass. One reflect core; one give-up rule; one stop-kind rule; `deckIO` marks at the store boundary only.
- ARCH-PURE: pass. `reflectDue`, `modelLookups`, `stopMeans`, `markUnfinished`, `stepBackground` are IO-free and table-tested; `runBackgroundJob` is the thin glue.
- ARCH-PURPOSE: pass. M2's stated purpose, the session writing the model at the floor and refreshing it rarely, ahead of the harvest that reads it, is delivered end to end; no deferred follow-up is the point.
- ARCH-MOCK: pass. Same seam (`deps.newLLM`, `store.YAML` on a temp dir); no new call shape, so the existing reflect conformance check still covers it.
- ARCH-CONSTRAINTS: pass with the envelope-row drift in Minor 2. One extra model call per reflect inside the same one-job-at-a-time bound.
- ARCH-SECURE: pass. The hand-editable model is parsed defensively and fuzzed; unknown degrades to "not due", never to a paid call.
- ARCH-ORDER: pass with Minor 3. `deckErr` is a table row and a pinned transition; reflect-before-harvest is pinned by request index; the runner still copies memory into the goroutine so nothing is shared. The learner-model second-actor row is the one missing cell.
- ARCH-FUNERAL: pass. No new artifact family; the model is one file rewritten in place; `bgMemory` dies with the session; the atlas now names the sweep for a word forgotten mid-job.

**Rules for the repeat families.** Notices: `bgNoticeFor` is a fold over the result, one line per effect in job order followed by the stop line, with no early return (prevalence: both stop branches drop `reflected`). Plan tables: a Revisions entry that changes a behavior edits every table row and prose signature that describes it in the same commit, and the pre-gate grep pass covers signatures, not just symbols (prevalence: 4 sites). Shared state: every artifact the job writes is listed with all its other writers, including the operator's editor, and the resolution named (prevalence: 1 of 3 artifacts missing). Reads: a job reads each store surface once and the envelope prices the read per check (prevalence: Events per check, UserModel twice).

### 7. Plan revision recommendations

Add one `## Revisions` entry, "M2 boundary review, round 1", that: adds the `running | job done, deckErr | off | one notice` row to the state-machine table; extends the `bgJobResult` bullet with `deckErr` and `reflectFailed`; rewrites the `runBackgroundJob` and `bgRunner` prose to `bgMemory`; changes the envelope's reflect-input row to "Events reads every day file per check once the deck is at the floor"; and adds the learner-model row to the ARCH-ORDER second-actor list with the bounded consequence above.

```findings
findings:
  - id: new
    severity: Minor
    family: notice-wording-matches-cause
    title: |
      bgNoticeFor's early return on noModel/deckErr hides a learner model the same job wrote
    detail: |
      This is the 2nd finding in family notice-wording-matches-cause. Rule: bgNoticeFor is a fold over the result, one line per effect in job order then the stop line, no early return. Both stop branches (background.go:127-135) drop reflected today; a job that reflects and then meets a 429 on its first band call says only that the model did not answer.
  - id: new
    severity: Minor
    family: plan-table-classification-matches-code
    title: |
      Plan tables and prose lag the M2 code in four places (deckErr row, bgJobResult fields, bgMemory signatures, envelope reflect row)
    detail: |
      This is the 2nd finding in family plan-table-classification-matches-code. Rule: a Revisions entry that changes a behavior edits every table row and prose signature describing it in the same commit; the pre-gate grep pass covers signatures, not only symbols. Prevalence: 4 sites. See section 7 for the Revisions entry.
  - id: new
    severity: Minor
    family: shared-state-across-the-job-boundary
    title: |
      The learner model is now written by a background actor while the README invites hand-editing it, and the plan's second-actor list does not name it
    detail: |
      This is the 2nd finding in family shared-state-across-the-job-boundary. Rule: every artifact the job writes is listed with all its other writers, including the operator's editor, and the resolution named. Prevalence: 1 of the 3 artifacts the job writes is missing. Consequence is bounded: the splice keeps Corrections from disk, and a clobbered refresh leaves the old count so the next check refreshes again, one wasted call. Plan-only change.
  - id: new
    severity: Minor
    family: repeated-reads-per-job
    title: |
      Once the deck is at the floor every check reads the whole event log, and a reflecting job reads UserModel twice
    detail: |
      This is the 2nd finding in family repeated-reads-per-job. Rule: a job reads each store surface once, and the envelope prices the read per check. Reading UserModel before Events would skip the log read when modelLookups is unknown; background-only, acknowledged in Revisions, envelope row not yet updated.
```
