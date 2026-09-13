# Boundary Review — tools#54 (whole-issue close)

| field | value |
|-------|-------|
| issue | 54 — the TUI keeps practice material current in the background, so new words get cloze questions without a command |
| repo | tools |
| issue file | workshop/issues/000054-background-harvest.md |
| boundary | whole-issue close |
| milestone | — |
| window | c4123297dc2a28907227958e2ccfe13d8e94f4f7..4677e40e6f9359f98415c30bd4defad54a878887 |
| command | sdlc close --issue 54 |
| reviewer | claude |
| timestamp | 2026-09-13T01:12:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The whole-issue window delivers what the Spec and both milestone rows promise, and it holds up against the code rather than the commit messages. I inspected the stat, name-status and full patch for every file, read the issue Spec/Plan/Log, the plan's Core concepts, state table, envelope and Revisions, the gate ledger, and the store gate the job reads through off-loop. Every prior open finding (BR-8 through BR-16) is disposed by evidence: I re-applied six mutations in a throwaway worktree at head (budget cut counted as unfinished, store error unmarked, a miss counted toward the check, an early return in `bgNoticeFor`, the log read before the model, and the permission gate replaced by `allowed()`), and each turned its named test red; the unmutated tree is green. `gofmt`, `go vet`, `go build ./...` are clean, the full `cmd/define` suite passed (120.6 s), and a `-race` run over the session, runner and dictionary tests passed. Nothing blocks SHIP. Three new Minors follow, two of them repeats in existing families with the rule stated rather than the instance.

### 1. Strengths

- `cmd/define/background.go:91-124` with `:130-149` — the transition table is the whole state, the notice list is a fold, and `TestStepBackgroundTransitions` carries a row with both an effect and a stop. This is ARCH-ORDER done as the principle asks: two fields, a written table, every row pinned.
- `cmd/define/background.go:306-314` and `gated_store.go:59-64` — the job never asks the deck question. `backgroundEnabled` reads `saving()`, which cannot prompt, and `repl.go:245` resolves before `replRaw` starts, so the job's later `allowed()` calls hit an already-decided state. `TestNoJobWhereTheDeckWasNotAgreedTo/undecided` uses an `ask` that would say yes, so replacing `saving()` with `allowed()` goes red (verified).
- `cmd/define/harvest.go:270-280` and `background.go:435-449` — `markUnfinished` and `stopMeans` are the round-2 families answered as one rule each; every give-up site in both loops routes through the first, and the `errBudget` breaks precede the returns that would mark.
- `cmd/define/background_loop_test.go:85-115, 157-217` — the never-waits and quitting tests fix the interleaving with a blocking stub released by the test, not with sleeps; the reflect-before-harvest test asserts by request index against the fake.
- `cmd/define/background.go:401-425` with `FuzzModelLookups` — a hand-editable file is parsed to "unknown means not due", and the fuzz asserts a reported count is one the frontmatter states. ARCH-SECURE for a paid decision.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- `workshop/plans/000054-background-harvest-plan.md` Task 1.7 Step 3 and the issue's ten `Done when` boxes are unticked at close. The Log says twice the operator TUI smoke test is "still pending". Every Done-when bullet has a pin, so the boxes are tickable; the smoke test is either done or explicitly waived in `--verified`. New family `checklist-state-matches-delivery`.
- `cmd/define/background.go:327-337` — `quietStore` is a type switch over two concrete shapes; any other `store.Store` wrapper in production returns unchanged and reintroduces the round-1 hazard silently. **This is the 2nd finding in family `background-never-writes-outside-the-loop`.** Rule: the job's silence is a property of the seam, not of a list of known types. Make quieting an optional interface (`Quiet() store.Store`) each wrapper forwards, with a guard test that every `store.Store` implementation in the package implements it, so a new wrapper fails compile-time or the guard rather than the frame.
- `cmd/define/background.go:264-286` — `bgMemory` is keyed by word text and one `reflectFailed` bit, and it survives `/lang`. A word that failed in `en` is skipped in `es` this session if both decks hold it, and a failed `en` reflect stops the `es` model from being written this session. **This is the 3rd finding in family `shared-state-across-the-job-boundary`.** Rule: session memory that outlives a store switch is keyed by the store it was learned against (lang), or reset when the switch happens. Prevalence: both fields of `bgMemory`; the `since` cadence is unaffected because it is not a claim about the store.

### 5. Test coverage notes

- Mutations re-run at head, each anchor matched once, all red: `markUnfinished` counts a budget cut; `pendingWords` error unmarked; a miss counts; `bgNoticeFor` early-returns on `noModel`; `reflectIfDue` reads Events before `reflectCouldBeDue`; `backgroundEnabled` uses `allowed()`; the runner drops `failed`. Control green.
- The budget cap remains unobservable (batch bounds it first); recorded in the test and lessons, not claimed as a pin. Acceptable.
- No test pins the `/lang`-mid-job memory behavior either way (Minor 3); a two-language rig would.
- `TestTheJobWritesNothingToTheTerminal` pins the production shape (gated over YAML) with a loud control; it would not catch a third wrapper (Minor 2).

### 6. Architectural notes for upcoming work

- ARCH-DRY: pass. One harvest core, one reflect core, one `batchFilter`, one lock.
- ARCH-PURE: pass. `stepBackground`, `bgNoticeFor`, `reflectDue`, `modelLookups`, `stopMeans`, `markUnfinished` are IO-free and table-tested; `runBackgroundJob` is thin glue.
- ARCH-PURPOSE: pass. Every Done-when bullet has an end-to-end or job-level pin against the fake, not a stubbed trigger. The only undelivered item is the operator smoke test (Minor 1).
- ARCH-MOCK: pass. Same seam (`deps.newLLM`, `store.YAML` on a temp dir, `fakeDictionary`); the job adds no new call shape, so the existing conformance checks cover it. `-no-capture` yields no deck, so the job cannot spend calls on a `store.Mem`.
- ARCH-CONSTRAINTS: pass. Envelope prices the per-check reads; one goroutine, capacity-1 channel, 2 s stop.
- ARCH-SECURE: pass. Hand-edited model degrades to "not due"; the deck-error notice carries an OS error string on screen, no credential.
- ARCH-ORDER: pass with Minor 3. Table explicit, interleavings injectable via the stub, extent bounded by the session context and the deferred stop.
- ARCH-FUNERAL: pass. No new durable family; `bgMemory` dies with the session; the forgotten-word sweep is named and `YAML.Forget` does remove facts and items for a word outside the deck (`yaml.go:787-810`).

### 7. Plan revision recommendations

Add one `## Revisions` entry, "close review": (a) the plan's Task 1.7 Step 3 is ticked or recorded as waived with the reason; (b) the ARCH-ORDER "`/lang` mid-job" row says the session's `bgMemory` carries across the switch and what that costs, or that it is keyed by language once Minor 3 is applied.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      markUnfinished is the one site; errBudget breaks precede every marking return; mutation to count a budget cut turns TestMarkUnfinishedLeavesABudgetCutPending red.
  - id: BR-9
    disposition: addressed
    note: |
      deckperm.go:42-49 records the invariant; settleQuietly and resolve both return on a decided state, so the job's reads race nothing.
  - id: BR-10
    disposition: addressed
    note: |
      Store errors are marked errDeckIO where returned; dropping the mark in pendingWords turns TestAStoreErrorStopsTheSessionOnce red.
  - id: BR-11
    disposition: addressed
    note: |
      Making every submit count turns TestAMissedLookupDoesNotCountTowardTheCheck/misses red.
  - id: BR-12
    disposition: addressed
    note: |
      Atlas names the sweep; YAML.Forget removes facts and items whether or not the deck holds the word.
  - id: BR-13
    disposition: addressed
    note: |
      bgNoticeFor is a fold; an inserted early return turns TestBgNoticeForSaysEveryEffectInJobOrder red.
  - id: BR-14
    disposition: addressed
    note: |
      deckErr row, bgJobResult fields, bgMemory signatures and the per-check envelope row all match the code.
  - id: BR-15
    disposition: addressed
    note: |
      The learner-model row names the operator's editor and --reflect with the splice as the resolution.
  - id: BR-16
    disposition: addressed
    note: |
      reflectIfDue reads the model first and the log only when it could be due; reading the log first turns TestAHandEditedModelCostsNoLogRead red.
findings:
  - id: new
    severity: Minor
    family: checklist-state-matches-delivery
    title: |
      The operator smoke test (Task 1.7 Step 3) and the issue's ten Done-when boxes are unticked at close
    detail: |
      The Log says twice the smoke test is still pending. Rule: a checkbox the plan or issue carries is ticked when its evidence exists, or explicitly waived in --verified, before the boundary; every Done-when bullet has a pin, so the boxes are tickable now.
  - id: new
    severity: Minor
    family: background-never-writes-outside-the-loop
    title: |
      quietStore is a type switch over two concrete shapes, so any other production store wrapper reads loudly again
    detail: |
      This is the 2nd finding in family background-never-writes-outside-the-loop. Rule: the job's silence is a property of the seam, not a list of known types. Make quieting an optional interface each wrapper forwards, with a guard that every store.Store implementation in the package implements it. background.go:327-337.
  - id: new
    severity: Minor
    family: shared-state-across-the-job-boundary
    title: |
      bgMemory survives /lang, so a word or reflect that failed in one language is skipped in the next this session
    detail: |
      This is the 3rd finding in family shared-state-across-the-job-boundary. Rule: session memory that outlives a store switch is keyed by the store it was learned against, or reset on the switch. Prevalence: both fields of bgMemory (background.go:233-236, merged at :281-286); no test covers a two-language session.
```
