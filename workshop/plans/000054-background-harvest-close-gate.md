---
gate: boundary-review
issue: 54
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-12T23:36:20-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Band-refused words never enter the runner's tried set, so every check re-spends band calls on them and can starve the batch
          detail: harvest.go:213-220 counts a non-CEFR band as refused and continues without recording the word; background.go:189 forwards only failed into tried. Such a word stays unbanded, stays pending, and if among the ten newest occupies a batch slot every job. Report refused words as data and add them to tried; pin with a two-job test scripting a bad band for one word.
          family: retry-bound-covers-every-failure-kind
          round: 1
        - id: BR-2
          severity: Important
          title: A background Deck() read can print a store warning to the raw terminal's stderr from the job goroutine
          detail: 'The session''s stores are built with run''s process stderr as warn (main.go:347,355) before the console exists; yaml.go:262-269 prints "define: skipping ..." for a bad words/*.yaml. The job reads the deck three times per check off-loop, so the write lands in the frame at an arbitrary moment. Redirect the store''s warn writer through the console, or discard it in the job''s deps copy, and add a test planting a bad word file.'
          family: background-never-writes-outside-the-loop
          round: 1
        - id: BR-3
          severity: Important
          title: Plan table lists pendingWords as PURE, but it reads through store.Store and its test runs on store.NewMem
          detail: Move the row to the Integration points table (wraps the store) or extract the no-band-or-no-item predicate as the pure row. Code needs no change; the plan needs a Revisions entry.
          family: plan-table-classification-matches-code
          round: 1
        - id: BR-4
          severity: Minor
          title: assertNoJob proves absence with a 700 ms poll instead of a fail-if-called newLLM plus end()
          detail: background_loop_test.go:228-241. A newLLM that t.Fatal's if called, with end() waiting on the deferred bg.stop, observes the absence deterministically.
          family: tests-fix-the-interleaving
          round: 1
        - id: BR-5
          severity: Minor
          title: A transient 429/5xx (ErrUnavailable) turns the session's background work off with a "no model answered" notice
          detail: background.go:194 with errors.go:39-49. Consistent with the CLI's stop, but the wording overstates a rate-limit; consider "the model did not answer".
          family: notice-wording-matches-cause
          round: 1
        - id: BR-6
          severity: Minor
          title: runBackgroundJob repeats the nil-deps guard backgroundEnabled already applies
          detail: background.go:172 and :270. Keep one, or say why the job re-checks.
          family: repeated-guard
          round: 1
        - id: BR-7
          severity: Minor
          title: One job reads the deck three times and every word's facts twice
          detail: pendingWords, runBackgroundJob and runAuthoring each call Deck(); passing the deck into pendingWords removes one pass. Background only, inside the declared envelope.
          family: repeated-reads-per-job
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T00:11:07-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Reverting the refused-band append turns TestABandRefusalIsRetriedOncePerSession red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Reverting quietStore, and separately the gated branch, turns TestTheJobWritesNothingToTheTerminal red; production deck is *gatedStore over *store.YAML.
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'pendingWords is in the Integration points table with wraps: the store.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: assertNoJob counts clients built and asserts after end(); no poll remains.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Notice, README and atlas say the model did not answer.
          round: 2
        - id: BR-6
          disposition: addressed
          note: hasModelSeam is the one guard; the job's re-check is explained.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Deck() is read once per job and passed to pendingWords, harvestDeck and runAuthoring.
          round: 2
      findings:
        - id: BR-8
          severity: Minor
          title: A budget cut at the author or entail call marks the word failed, unlike the veto path
          detail: 'This is the 2nd finding in this family. harvest.go:345 and :369 return append(failed, c.Word) when runWithin yields errBudget, while :395 treats the same cut as truncated and leaves the word pending; the comment at :152 says a budget cut is not a failure. Unreachable at bgBudget 60 over 10 words. The rule: membership in failed is decided by error kind at one site, excluding errBudget, rather than at each return.'
          family: retry-bound-covers-every-failure-kind
          round: 2
        - id: BR-9
          severity: Minor
          title: deckPermission's single-goroutine invariant is now stale
          detail: deckperm.go:42 says every caller is on the command's own path; the job goroutine reads it via gatedStore.reading(). Safe today because the runner is built only once saving() is decided and no method writes on a decided state. Record that invariant so a later re-ask path does not race.
          family: shared-state-across-the-job-boundary
          round: 2
        - id: BR-10
          severity: Minor
          title: A store read error inside a job leaves no trace in the session
          detail: background.go:189 and harvest.go:290 return an empty result with no failed or stopped, so a persistently unreadable facts file silently stops authoring for the session where --harvest reports it.
          family: silent-failure-has-a-signal
          round: 2
        - id: BR-11
          severity: Minor
          title: No test pins that a missed lookup does not count toward the check
          detail: Mutating replraw.go:652 to fire bgLookedUp on any code leaves the suite green. Cadence only, since the store is the count.
          family: trigger-event-pinned
          round: 2
        - id: BR-12
          severity: Minor
          title: Facts and items re-created for a word forgotten mid-job have no removal path
          detail: The plan argues they are harmless; the only cleanup is a second --forget of the same word. Rare, but per ARCH-FUNERAL the sweep should be named.
          family: residue-names-its-sweep
          round: 2
      boundary: M1
      blocked: false
    - "n": 3
      timestamp: "2026-09-13T00:54:08-07:00"
      agent: claude
      findings:
        - id: BR-13
          severity: Minor
          title: bgNoticeFor's early return on noModel/deckErr hides a learner model the same job wrote
          detail: 'This is the 2nd finding in family notice-wording-matches-cause. Rule: bgNoticeFor is a fold over the result, one line per effect in job order then the stop line, no early return. Both stop branches (background.go:127-135) drop reflected today; a job that reflects and then meets a 429 on its first band call says only that the model did not answer.'
          family: notice-wording-matches-cause
          round: 3
        - id: BR-14
          severity: Minor
          title: Plan tables and prose lag the M2 code in four places (deckErr row, bgJobResult fields, bgMemory signatures, envelope reflect row)
          detail: 'This is the 2nd finding in family plan-table-classification-matches-code. Rule: a Revisions entry that changes a behavior edits every table row and prose signature describing it in the same commit; the pre-gate grep pass covers signatures, not only symbols. Prevalence: 4 sites. See section 7 for the Revisions entry.'
          family: plan-table-classification-matches-code
          round: 3
        - id: BR-15
          severity: Minor
          title: The learner model is now written by a background actor while the README invites hand-editing it, and the plan's second-actor list does not name it
          detail: 'This is the 2nd finding in family shared-state-across-the-job-boundary. Rule: every artifact the job writes is listed with all its other writers, including the operator''s editor, and the resolution named. Prevalence: 1 of the 3 artifacts the job writes is missing. Consequence is bounded: the splice keeps Corrections from disk, and a clobbered refresh leaves the old count so the next check refreshes again, one wasted call. Plan-only change.'
          family: shared-state-across-the-job-boundary
          round: 3
        - id: BR-16
          severity: Minor
          title: Once the deck is at the floor every check reads the whole event log, and a reflecting job reads UserModel twice
          detail: 'This is the 2nd finding in family repeated-reads-per-job. Rule: a job reads each store surface once, and the envelope prices the read per check. Reading UserModel before Events would skip the log read when modelLookups is unknown; background-only, acknowledged in Revisions, envelope row not yet updated.'
          family: repeated-reads-per-job
          round: 3
      boundary: M2
      blocked: false
---

# Gate ledger — tools#54 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T23:36:20-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `retry-bound-covers-every-failure-kind` Band-refused words never enter the runner's tried set, so every check re-spends band calls on them and can starve the batch
  harvest.go:213-220 counts a non-CEFR band as refused and continues without recording the word; background.go:189 forwards only failed into tried. Such a word stays unbanded, stays pending, and if among the ten newest occupies a batch slot every job. Report refused words as data and add them to tried; pin with a two-job test scripting a bad band for one word.
- **BR-2** [Important] `background-never-writes-outside-the-loop` A background Deck() read can print a store warning to the raw terminal's stderr from the job goroutine
  The session's stores are built with run's process stderr as warn (main.go:347,355) before the console exists; yaml.go:262-269 prints "define: skipping ..." for a bad words/*.yaml. The job reads the deck three times per check off-loop, so the write lands in the frame at an arbitrary moment. Redirect the store's warn writer through the console, or discard it in the job's deps copy, and add a test planting a bad word file.
- **BR-3** [Important] `plan-table-classification-matches-code` Plan table lists pendingWords as PURE, but it reads through store.Store and its test runs on store.NewMem
  Move the row to the Integration points table (wraps the store) or extract the no-band-or-no-item predicate as the pure row. Code needs no change; the plan needs a Revisions entry.
- **BR-4** [Minor] `tests-fix-the-interleaving` assertNoJob proves absence with a 700 ms poll instead of a fail-if-called newLLM plus end()
  background_loop_test.go:228-241. A newLLM that t.Fatal's if called, with end() waiting on the deferred bg.stop, observes the absence deterministically.
- **BR-5** [Minor] `notice-wording-matches-cause` A transient 429/5xx (ErrUnavailable) turns the session's background work off with a "no model answered" notice
  background.go:194 with errors.go:39-49. Consistent with the CLI's stop, but the wording overstates a rate-limit; consider "the model did not answer".
- **BR-6** [Minor] `repeated-guard` runBackgroundJob repeats the nil-deps guard backgroundEnabled already applies
  background.go:172 and :270. Keep one, or say why the job re-checks.
- **BR-7** [Minor] `repeated-reads-per-job` One job reads the deck three times and every word's facts twice
  pendingWords, runBackgroundJob and runAuthoring each call Deck(); passing the deck into pendingWords removes one pass. Background only, inside the declared envelope.

## Round 2 — 2026-09-13T00:11:07-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Reverting the refused-band append turns TestABandRefusalIsRetriedOncePerSession red.
- BR-2 — addressed — Reverting quietStore, and separately the gated branch, turns TestTheJobWritesNothingToTheTerminal red; production deck is *gatedStore over *store.YAML.
- BR-3 — addressed — pendingWords is in the Integration points table with wraps: the store.
- BR-4 — addressed — assertNoJob counts clients built and asserts after end(); no poll remains.
- BR-5 — addressed — Notice, README and atlas say the model did not answer.
- BR-6 — addressed — hasModelSeam is the one guard; the job's re-check is explained.
- BR-7 — addressed — Deck() is read once per job and passed to pendingWords, harvestDeck and runAuthoring.

### Raised

- **BR-8** [Minor] `retry-bound-covers-every-failure-kind` A budget cut at the author or entail call marks the word failed, unlike the veto path
  This is the 2nd finding in this family. harvest.go:345 and :369 return append(failed, c.Word) when runWithin yields errBudget, while :395 treats the same cut as truncated and leaves the word pending; the comment at :152 says a budget cut is not a failure. Unreachable at bgBudget 60 over 10 words. The rule: membership in failed is decided by error kind at one site, excluding errBudget, rather than at each return.
- **BR-9** [Minor] `shared-state-across-the-job-boundary` deckPermission's single-goroutine invariant is now stale
  deckperm.go:42 says every caller is on the command's own path; the job goroutine reads it via gatedStore.reading(). Safe today because the runner is built only once saving() is decided and no method writes on a decided state. Record that invariant so a later re-ask path does not race.
- **BR-10** [Minor] `silent-failure-has-a-signal` A store read error inside a job leaves no trace in the session
  background.go:189 and harvest.go:290 return an empty result with no failed or stopped, so a persistently unreadable facts file silently stops authoring for the session where --harvest reports it.
- **BR-11** [Minor] `trigger-event-pinned` No test pins that a missed lookup does not count toward the check
  Mutating replraw.go:652 to fire bgLookedUp on any code leaves the suite green. Cadence only, since the store is the count.
- **BR-12** [Minor] `residue-names-its-sweep` Facts and items re-created for a word forgotten mid-job have no removal path
  The plan argues they are harmless; the only cleanup is a second --forget of the same word. Rare, but per ARCH-FUNERAL the sweep should be named.

## Round 3 — 2026-09-13T00:54:08-07:00 (claude) — passed

### Raised

- **BR-13** [Minor] `notice-wording-matches-cause` bgNoticeFor's early return on noModel/deckErr hides a learner model the same job wrote
  This is the 2nd finding in family notice-wording-matches-cause. Rule: bgNoticeFor is a fold over the result, one line per effect in job order then the stop line, no early return. Both stop branches (background.go:127-135) drop reflected today; a job that reflects and then meets a 429 on its first band call says only that the model did not answer.
- **BR-14** [Minor] `plan-table-classification-matches-code` Plan tables and prose lag the M2 code in four places (deckErr row, bgJobResult fields, bgMemory signatures, envelope reflect row)
  This is the 2nd finding in family plan-table-classification-matches-code. Rule: a Revisions entry that changes a behavior edits every table row and prose signature describing it in the same commit; the pre-gate grep pass covers signatures, not only symbols. Prevalence: 4 sites. See section 7 for the Revisions entry.
- **BR-15** [Minor] `shared-state-across-the-job-boundary` The learner model is now written by a background actor while the README invites hand-editing it, and the plan's second-actor list does not name it
  This is the 2nd finding in family shared-state-across-the-job-boundary. Rule: every artifact the job writes is listed with all its other writers, including the operator's editor, and the resolution named. Prevalence: 1 of the 3 artifacts the job writes is missing. Consequence is bounded: the splice keeps Corrections from disk, and a clobbered refresh leaves the old count so the next check refreshes again, one wasted call. Plan-only change.
- **BR-16** [Minor] `repeated-reads-per-job` Once the deck is at the floor every check reads the whole event log, and a reflecting job reads UserModel twice
  This is the 2nd finding in family repeated-reads-per-job. Rule: a job reads each store surface once, and the envelope prices the read per check. Reading UserModel before Events would skip the log read when modelLookups is unknown; background-only, acknowledged in Revisions, envelope row not yet updated.

## Open findings

- **BR-8** [Minor] `retry-bound-covers-every-failure-kind` A budget cut at the author or entail call marks the word failed, unlike the veto path
- **BR-9** [Minor] `shared-state-across-the-job-boundary` deckPermission's single-goroutine invariant is now stale
- **BR-10** [Minor] `silent-failure-has-a-signal` A store read error inside a job leaves no trace in the session
- **BR-11** [Minor] `trigger-event-pinned` No test pins that a missed lookup does not count toward the check
- **BR-12** [Minor] `residue-names-its-sweep` Facts and items re-created for a word forgotten mid-job have no removal path
- **BR-13** [Minor] `notice-wording-matches-cause` bgNoticeFor's early return on noModel/deckErr hides a learner model the same job wrote
- **BR-14** [Minor] `plan-table-classification-matches-code` Plan tables and prose lag the M2 code in four places (deckErr row, bgJobResult fields, bgMemory signatures, envelope reflect row)
- **BR-15** [Minor] `shared-state-across-the-job-boundary` The learner model is now written by a background actor while the README invites hand-editing it, and the plan's second-actor list does not name it
- **BR-16** [Minor] `repeated-reads-per-job` Once the deck is at the floor every check reads the whole event log, and a reflecting job reads UserModel twice
