# Boundary Review — tools#54 (milestone M1)

| field | value |
|-------|-------|
| issue | 54 — the TUI keeps practice material current in the background, so new words get cloze questions without a command |
| repo | tools |
| issue file | workshop/issues/000054-background-harvest.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c4123297dc2a28907227958e2ccfe13d8e94f4f7..b32d5e48c63d2ec51705fd27ba24c66666460885 |
| command | sdlc milestone-close --issue 54 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-12T23:36:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the Spec and Plan promise for this boundary: the store is the only count (`pendingWords`), one written transition table decides when a job runs, the job runs off the loop through the same `harvestDeck` core the CLI uses, every screen write stays on the loop between prompts, the dictionary is locked at the one builder both startup and `/lang` go through, and quitting cancels with a bounded wait. I ran `go vet` and the whole `cmd/define` package under `-race` at HEAD (`ok 137.5s`, exit 0); every test named in the plan exists and pins real behaviour through the wire-level fake or a blocking stub, not a stubbed trigger. Nothing blocks SHIP. Three Important items are worth fixing before the gate because each is cheap: band-refused words escape the per-session retry bound the plan reasoned about for authoring; the store's warning writer lets a background read print to the raw terminal off-loop; and the plan table labels `pendingWords` PURE while its test runs against the in-memory store.

**1. Strengths**

- `cmd/define/background.go:86-119` — `stepBackground` is a genuine `(state, event) -> (state, effects)` table, and `TestStepBackgroundTransitions` enumerates every row including the unreachable "stray result while idle" one. ARCH-ORDER done as the principle asks.
- `cmd/define/background.go:222-233` — the runner's send selects on `ctx.Done()`, and `TestTheRunnerNeverBlocksAfterStop` pre-fills the channel so the test only passes if that select exists. Extent is lexically bounded by `defer bg.stop(2s)` at `replraw.go:336`.
- `cmd/define/harvest.go:167-170, 316-318` — one `batchFilter` serves both halves; `TestABacklogDrainsOnBothHalves` cannot pass under the old band-everything-first order, which is the PQ-1 fix pinned by a failing-without-it test.
- `cmd/define/dict.go:52-60` + `TestProductionDictionariesAreLocked` — wrapping the single builder in `realDeps` rather than each construction site, with a test that reads what production actually hands out.
- `background_loop_test.go:76-105` — the blocking stub fixes interleavings by releasing a channel, so the never-waits and quitting tests observe a chosen ordering rather than whichever one the scheduler produced.
- The unobservable budget mutation is recorded honestly in the plan, the test comment, and a `lessons.md` entry instead of being claimed as a pin.

**2. Critical findings**

None.

**3. Important findings**

- **Band-refused words never join `tried`, so the job re-spends band calls on them at every check** — `cmd/define/harvest.go:213-220` counts a non-CEFR answer as `refused` and `continue`s without appending to `o.failed`; `runBackgroundJob` (`background.go:189`) forwards only `failed` into the runner's `tried`. A word the model persistently mis-bands stays `!Harvested()`, stays pending, and if it is among the ten newest occupies a batch slot every job: up to ten wasted calls per check, and with ten such words the older backlog is never reached. This is the retry-loop class decision 7 and PQ-1 were about, applied to the banding half. Fix sketch: report refused words as data (a `refused []string` on `harvestOutcome`, or fold them into the same "ran and kept nothing" list) and have `received` add them to `tried`; pin with a test that scripts `{"band":"Z9"}` for one word and asserts the second job makes no band call for it. (ARCH-PURPOSE: the class was named; the sweep missed one member. ARCH-ORDER: a retry-forever transition not on the table.)
- **A background store read can write to the raw terminal's stderr, off-loop** — the session's stores are built in `run` with the process `stderr` as their `warn` writer (`main.go:347,355` via `withStore(opt, stderr)`), before the console exists; `store.YAML.Deck` (`yaml.go:262-269`) prints `define: skipping …` for an unreadable or text-less word file. The job calls `Deck()` three times per check from its own goroutine, so a stray temp/hand-edited file under `words/` produces a write into the live frame at an arbitrary moment. The Spec's "nothing reaches stderr while the TUI owns the terminal" is exactly this. The seam predates #54, but the loop's own reads happen between prompts; the job's do not. Fix sketch: give the session's stores a warn writer the console can redirect to the screen (or that the job's deps copy points at `io.Discard`), and a test that plants a bad `words/*.yaml` and asserts no write lands while a prompt is shown. (ARCH-SECURE: a file under the deck directory is untrusted input; its failure path should degrade visibly on the loop, not into the frame.)
- **Plan table classifies `pendingWords` as PURE; it reads through `store.Store` and its test runs on `store.NewMem()`** — `workshop/plans/000054-background-harvest-plan.md` Core concepts, Pure entities row. The code is fine as an INTEGRATION entity (the store is a stateful fake behind the production seam, which is the ARCH-MOCK shape), but the table contradicts it. Either move the row to the integration table or extract the predicate (`needsWork(facts, items) bool`) as the pure row and list `pendingWords` as integration. Plan revision below. (ARCH-PURE.)

**4. Minor findings**

- `background_loop_test.go:228-241` — `assertNoJob` proves absence with a 700 ms poll. A `newLLM` that `t.Fatal`s if called plus `end()` (whose deferred `bg.stop` waits for the goroutine) makes it deterministic, the same shape `TestRunBackgroundJobHarvestsOnlyPastTheThreshold` already uses. (ARCH-ORDER.)
- `background.go:194` — `ErrUnavailable` covers 429 and every 5xx (`errors.go:39-49`), so one rate-limit burst on a paid key turns background work off for the whole session with a "no model answered" notice. Consistent with the CLI's stop, but the wording overstates it; consider "the model did not answer".
- `background.go:172` repeats the nil guard `backgroundEnabled` already applies (`:270`); one is enough, or name why the job re-checks.
- Per job the deck is read three times (`pendingWords`, `runBackgroundJob`, `runAuthoring`) and facts twice; passing the deck into `pendingWords` removes one full pass. Background only, so not hot-path. (ARCH-CONSTRAINTS, within the declared envelope.)
- `harvest.go:104-106` — the rewritten comment leaves one line at ~130 columns.

**5. Test coverage notes**

- Every plan-named test exists and passes under `-race`. The pure table, the batch semantics, the typed stop, the retry-once set, the non-blocking send, the permission and off-switch gates, the one-notice rule, and the between-prompts rule each have a test that fails without the code it pins (verified by inspection of what each asserts, not by re-running the 13 mutations).
- Gap: no test for a band-refused word across two jobs (finding 1). No test for a store warning during a job (finding 2).
- The budget test is honestly marked unobservable; when an item's cost can exceed six calls, revisit it.

**6. Architectural notes for upcoming work (M2)**

- ARCH-DRY pass; ARCH-PURE flag (table row, above); ARCH-PURPOSE flag (refused words); ARCH-MOCK pass (model via `d.newLLM`, dictionary via `d.dict`, real store on tmpdir, existing conformance checks cover the call shapes); ARCH-CONSTRAINTS pass (envelope declared and enforced: one goroutine, capacity-1 channel, 60-call budget, 2 s stop); ARCH-SECURE flag (store warn writer, above); ARCH-ORDER flag (refused-word loop, plus the poll-based absence test); ARCH-FUNERAL pass (nothing new durable; `tried` dies with the session; facts/items are existing per-word families).
- M2 adds `reflectDeck` to the same job. `reflectOutcome.stopped` must feed the same `noModel` typing, and the reflect write should be atomic like the store's, since the 2 s stop can cut it.
- When M2 lands `modelLookups`, keep the parse frontmatter-only as planned; the same "file under a shared directory is untrusted" lens as finding 2.

**7. Plan revision recommendations**

- `## Revisions` — M1 review: move `pendingWords` from the Pure entities table to the Integration points table (wraps: the store), or add a pure `needsWork` row and keep `pendingWords` as its integration caller.
- `## Revisions` — M1 review: `harvestOutcome` gains the refused-word list and `tried` absorbs it; add the mutation row "a refused band is dropped from `tried`" with its test.
- `## Revisions` — M1 review: note the store warn-writer seam under ARCH-ORDER's "events the loop cannot block" as "a store warning during a job", with the chosen fix.

```findings
findings:
  - id: new
    severity: Important
    family: retry-bound-covers-every-failure-kind
    title: |
      Band-refused words never enter the runner's tried set, so every check re-spends band calls on them and can starve the batch
    detail: |
      harvest.go:213-220 counts a non-CEFR band as refused and continues without recording the word; background.go:189 forwards only failed into tried. Such a word stays unbanded, stays pending, and if among the ten newest occupies a batch slot every job. Report refused words as data and add them to tried; pin with a two-job test scripting a bad band for one word.
  - id: new
    severity: Important
    family: background-never-writes-outside-the-loop
    title: |
      A background Deck() read can print a store warning to the raw terminal's stderr from the job goroutine
    detail: |
      The session's stores are built with run's process stderr as warn (main.go:347,355) before the console exists; yaml.go:262-269 prints "define: skipping ..." for a bad words/*.yaml. The job reads the deck three times per check off-loop, so the write lands in the frame at an arbitrary moment. Redirect the store's warn writer through the console, or discard it in the job's deps copy, and add a test planting a bad word file.
  - id: new
    severity: Important
    family: plan-table-classification-matches-code
    title: |
      Plan table lists pendingWords as PURE, but it reads through store.Store and its test runs on store.NewMem
    detail: |
      Move the row to the Integration points table (wraps the store) or extract the no-band-or-no-item predicate as the pure row. Code needs no change; the plan needs a Revisions entry.
  - id: new
    severity: Minor
    family: tests-fix-the-interleaving
    title: |
      assertNoJob proves absence with a 700 ms poll instead of a fail-if-called newLLM plus end()
    detail: |
      background_loop_test.go:228-241. A newLLM that t.Fatal's if called, with end() waiting on the deferred bg.stop, observes the absence deterministically.
  - id: new
    severity: Minor
    family: notice-wording-matches-cause
    title: |
      A transient 429/5xx (ErrUnavailable) turns the session's background work off with a "no model answered" notice
    detail: |
      background.go:194 with errors.go:39-49. Consistent with the CLI's stop, but the wording overstates a rate-limit; consider "the model did not answer".
  - id: new
    severity: Minor
    family: repeated-guard
    title: |
      runBackgroundJob repeats the nil-deps guard backgroundEnabled already applies
    detail: |
      background.go:172 and :270. Keep one, or say why the job re-checks.
  - id: new
    severity: Minor
    family: repeated-reads-per-job
    title: |
      One job reads the deck three times and every word's facts twice
    detail: |
      pendingWords, runBackgroundJob and runAuthoring each call Deck(); passing the deck into pendingWords removes one pass. Background only, inside the declared envelope.
```

---

## Re-review — 2026-09-13T00:11:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 54 — the TUI keeps practice material current in the background, so new words get cloze questions without a command |
| repo | tools |
| issue file | workshop/issues/000054-background-harvest.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | c4123297dc2a28907227958e2ccfe13d8e94f4f7..631e6e399912adb61a2e7c086b25a5a4e27398af |
| command | sdlc milestone-close --issue 54 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-13T00:11:07-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The M1 window delivers what the issue's Spec and Plan claim for this boundary, and every prior finding is genuinely closed. The whole `cmd/define` suite is green at the pinned head, `go vet ./...` is clean, and the background and harvest tests pass under the race detector. I verified each of the three Important round-1 fixes by reverting it in a scratch worktree: the refused-band fix, the quiet-store fix, and the gated-store branch of `quietStore` each turn their named test red, as does the stopped-word pin. The production deck store is the `*gatedStore` over `*store.YAML` that `quietStore` unwraps, so the BR-2 fix is reachable, not decorative. What remains is a handful of Minors, none of which touch the boundary's contracts, plus the operator smoke test the plan lists as Task 1.7 Step 3, which is unticked and outside what a reviewer can run.

**Strengths**

- `cmd/define/background.go:84` — the state machine is a real tagged enum plus one counter, with the transition table in the doc comment and every row pinned by `TestStepBackgroundTransitions`. This is ARCH-ORDER done right.
- `cmd/define/background_loop_test.go:88` — `bgStub` gives the loop tests a seam to hold a model call, so "never waits" and "quitting cancels" observe a chosen interleaving rather than a lucky one.
- `cmd/define/background_test.go:263` — `TestTheJobWritesNothingToTheTerminal` includes a control read that must warn, so the test cannot pass vacuously, and it covers both the bare and gated store shapes.
- `cmd/define/harvest.go:169` — one core serves `--harvest` and the job; the existing harvest output tests pass unchanged, which is the byte-faithfulness guard for the CLI.
- `cmd/define/dict.go:38` — one package-level mutex wraps the one builder both the startup dictionary and `/lang` go through, and `TestProductionDictionariesAreLocked` reads what `realDeps` actually hands out.

**Critical findings**

None.

**Important findings**

None.

**Minor findings**

- `cmd/define/harvest.go:345` and `:369` — a budget cut at the author or entail call marks the word `failed`, so the session skips it, while the same cut at the veto call (`:395`) leaves it pending. Unreachable at today's constants, but it contradicts the comment at `:152`.
- `cmd/define/deckperm.go:42` — the "not safe for concurrent use, every caller is on the command's own path" invariant is now false in letter: the job goroutine reads it through `gatedStore.reading()`. It is safe only because the runner exists solely once the state is decided and every method is read-only on a decided state. Write that down.
- `cmd/define/background.go:189` and `harvest.go:290` — a store read error in a job returns an empty result with no notice, so a persistently unreadable facts file silently disables authoring for the session where `--harvest` would say so.
- `cmd/define/replraw.go:652` — mutating the lookup event to fire on any exit code leaves the suite green; no test pins that a miss does not count. Cadence only, since the store is the count.
- A word forgotten mid-job can get its facts and items re-created with no sweep; the plan argues it, but the only removal path is a second `--forget`.

**Test coverage notes**

Every claimed fix has a test that fails without it, verified by revert. The batch, threshold, budget, off switch, permission gate, and notice placement all have pins. The gaps are the two above: the hit-versus-miss event filter, and the errBudget classification at the author and entail sites.

**Architectural notes**

- ARCH-DRY pass. ARCH-PURE pass; `pendingWords` is honestly an integration row now. ARCH-PURPOSE pass for M1; M2 is a declared second boundary, not a deferred purpose. ARCH-MOCK pass; the job adds no new call shape and the conformance tests exist. ARCH-CONSTRAINTS pass; the envelope is enforced and the lesson about the cap that cannot bind is recorded. ARCH-SECURE pass; N/A beyond the existing model config. ARCH-ORDER pass, with the permission-invariant note above. ARCH-FUNERAL pass, with the orphan-residue note.
- For M2, `runAuthoring` now has seven return sites each spelling `append(failed, c.Word)`. When reflect gets the same treatment, classify "unfinished" by error kind in one helper rather than at each site; that also fixes the errBudget inconsistency.

**Plan revision recommendations**

None required if the errBudget sites are fixed to match the comment. If they are left as is, add a Revisions line saying a mid-item budget cut at the author or entail call counts as unfinished for the session.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Reverting the refused-band append turns TestABandRefusalIsRetriedOncePerSession red.
  - id: BR-2
    disposition: addressed
    note: |
      Reverting quietStore, and separately the gated branch, turns TestTheJobWritesNothingToTheTerminal red; production deck is *gatedStore over *store.YAML.
  - id: BR-3
    disposition: addressed
    note: |
      pendingWords is in the Integration points table with wraps: the store.
  - id: BR-4
    disposition: addressed
    note: |
      assertNoJob counts clients built and asserts after end(); no poll remains.
  - id: BR-5
    disposition: addressed
    note: |
      Notice, README and atlas say the model did not answer.
  - id: BR-6
    disposition: addressed
    note: |
      hasModelSeam is the one guard; the job's re-check is explained.
  - id: BR-7
    disposition: addressed
    note: |
      Deck() is read once per job and passed to pendingWords, harvestDeck and runAuthoring.
findings:
  - id: new
    severity: Minor
    family: retry-bound-covers-every-failure-kind
    title: |
      A budget cut at the author or entail call marks the word failed, unlike the veto path
    detail: |
      This is the 2nd finding in this family. harvest.go:345 and :369 return append(failed, c.Word) when runWithin yields errBudget, while :395 treats the same cut as truncated and leaves the word pending; the comment at :152 says a budget cut is not a failure. Unreachable at bgBudget 60 over 10 words. The rule: membership in failed is decided by error kind at one site, excluding errBudget, rather than at each return.
  - id: new
    severity: Minor
    family: shared-state-across-the-job-boundary
    title: |
      deckPermission's single-goroutine invariant is now stale
    detail: |
      deckperm.go:42 says every caller is on the command's own path; the job goroutine reads it via gatedStore.reading(). Safe today because the runner is built only once saving() is decided and no method writes on a decided state. Record that invariant so a later re-ask path does not race.
  - id: new
    severity: Minor
    family: silent-failure-has-a-signal
    title: |
      A store read error inside a job leaves no trace in the session
    detail: |
      background.go:189 and harvest.go:290 return an empty result with no failed or stopped, so a persistently unreadable facts file silently stops authoring for the session where --harvest reports it.
  - id: new
    severity: Minor
    family: trigger-event-pinned
    title: |
      No test pins that a missed lookup does not count toward the check
    detail: |
      Mutating replraw.go:652 to fire bgLookedUp on any code leaves the suite green. Cadence only, since the store is the count.
  - id: new
    severity: Minor
    family: residue-names-its-sweep
    title: |
      Facts and items re-created for a word forgotten mid-job have no removal path
    detail: |
      The plan argues they are harmless; the only cleanup is a second --forget of the same word. Rare, but per ARCH-FUNERAL the sweep should be named.
```
