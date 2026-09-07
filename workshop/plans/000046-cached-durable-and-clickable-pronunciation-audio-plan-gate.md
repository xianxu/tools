---
gate: plan-quality
issue: 46
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-07T13:03:28-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: The durable audio key drops locale, source voice and `from`, so it collides where the in-memory memo does not
          detail: |-
            Task 2 keys the cache as `audio/<lang>/<slug>.{mp3,none}` and justifies
            `scoped: true` with "AudioCandidates keys on the voice's Lang and Locale".
            `audioDir` would scope on the STORE's `y.lang` (yaml.go:137,177), not the
            voice's, and locale is absent from the path entirely. The seam's real key
            is the whole joined candidate list (fetch.go:109), which
            `utterance.Candidates()` (audiourl.go:220-231) builds from a source voice,
            a session voice and `u.Spellings` — so `-pron en red` and plain `red` in
            one Spanish directory, and `-locale gb` vs `-locale us`, all map to one
            file and serve the wrong recording. `<slug>.mp3` also drops `from`, which
            `spokeSource` (audiourl.go:237) and `reportVoice` (main.go:1014) depend on,
            so a cache hit makes that RECORD silent or false. Decide the on-disk key
            against the seam's key, not against the word.
          family: cache-key-narrower-than-seam
          round: 1
        - id: PQ-2
          severity: Important
          title: Appending `wordRegions` to existing regions creates duplicate spans that silently kill underlines to their right
          detail: |-
            A Choice prompt already gets `RegionHeadword` at Line 1 Col 0
            (play_loop.go:1220), and the headword is a deck word, so `wordRegions`
            emits a second region on the same cells. `markClickable` (screen.go:359)
            sorts by Col and advances one `next` cursor, so after the first span
            closes the duplicate at Col 0 can never match again and every later region
            on that line loses its underline. `RegionAt` (screen.go:192) resolves by
            append order, not by narrowest span. State the precedence rule and add a
            property guard that a line's regions are disjoint and ascending — that is
            markClickable's unwritten precondition.
          family: region-overlap-precedence
          round: 1
        - id: PQ-3
          severity: Important
          title: Moving the wrap into `realDeps` leaves every loop test on an uncached source
          detail: |-
            Task 1 Step 3 claims "a test driving any loop through realDeps now gets the
            same source production gets", but no test calls `realDeps()` — `run()`
            takes deps and the suite builds `deps{...}` literals at 21 sites
            (main_test.go:57). Deleting repl.go:257 and replraw.go:264 therefore
            reverses the #2 I-1 reason repl.go:256 states, and the planned
            `TestASittingFetchesARecordingOnce` over `runPlay` would have to hand-wrap
            to work. Wrap at a normalisation point both production and the loop tests
            traverse, or keep the loop wraps and let the guard be the derived thing.
          family: seam-wrap-site
          round: 1
        - id: PQ-4
          severity: Important
          title: '`surface` is a property of the form, not the write site — and a board never reaches `writeRendered` at all'
          detail: |-
            There is one prompt write site (play_loop.go:226) serving both Choice and
            Cloze, so it cannot distinguish them; the surface must be derived from the
            form. A board returns at play_loop.go:217 before reaching it, with cells
            going through `boardFooter` (play_loop.go:672) into the footer — so
            "surfaceDeck for a cloze prompt and a board" is unexecutable and
            `TestBoardCellsAreNotColoured` is a pin that cannot fail. The Step 5 guard
            checks only that a surface is passed, not that it is right; the derived
            extent to hang the classification on is `docSyncForms` /
            `TestEveryFormIsEnrolled` (doc_sync_test.go:104,126).
          family: surface-extent-axis
          round: 1
        - id: PQ-5
          severity: Minor
          title: '`TestTheRawAudioSourceIsConstructedOnceAndWrapped` is green before the change'
          detail: |-
            `newHTTPAudioSource` already has exactly one non-test call site
            (main.go:98), so the AST guard passes today and could never have caught the
            bug. Step 2's "Expected: FAIL" is true only of
            `TestRealDepsCarriesACachingAudioSource`. Mutation-sweep it the way Task 4
            Step 5 sweeps its own, or drop it.
          family: guard-cannot-fail
          round: 1
        - id: PQ-6
          severity: Minor
          title: Task 2's file list omits `.gitignore` and `store/store.go`
          detail: |-
            Adding "audio" to RuntimeDirs reddens `TestGitignoreCoversRuntimeDirs` —
            .gitignore lists words/ events/ usage/ facts/ items/ and single-sources
            from RuntimeDirs. The new audio methods also have to join the `Store`
            interface (store/store.go:10) that both Mem and storetest satisfy, which
            the file list does not mention.
          family: runtimedirs-fanout
          round: 1
        - id: PQ-7
          severity: Minor
          title: No ARCH-ORDER statement for the first binary artifact written to the working directory
          detail: |-
            M1 adds durable state with external events and says nothing about them.
            Name process-death-mid-write and two concurrent `define` processes in one
            directory, and say the bytes go through `writeBytesAtomic` — whose `.tmp-*`
            shadow RuntimeFiles already covers. A bare omission is what an author who
            cannot see the ordering writes.
          family: durable-write-ordering
          round: 1
        - id: PQ-8
          severity: Minor
          title: M2 carries no operating envelope while M1 carries a good one
          detail: |-
            M1 states 10-30KB per recording, ~30MB per 1000-word deck, and a 30-day TTL
            with its cost. M2 says nothing about `deckSpans`: one sentence that it runs
            per write (per question, per reveal) rather than per keystroke, bounded
            against a full-entry reveal, closes ARCH-CONSTRAINTS for the walk.
          family: constraints-envelope-missing
          round: 1
      blocked: true
---

# Gate ledger — tools#46 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T13:03:28-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `cache-key-narrower-than-seam` The durable audio key drops locale, source voice and `from`, so it collides where the in-memory memo does not
  Task 2 keys the cache as `audio/<lang>/<slug>.{mp3,none}` and justifies
  `scoped: true` with "AudioCandidates keys on the voice's Lang and Locale".
  `audioDir` would scope on the STORE's `y.lang` (yaml.go:137,177), not the
  voice's, and locale is absent from the path entirely. The seam's real key
  is the whole joined candidate list (fetch.go:109), which
  `utterance.Candidates()` (audiourl.go:220-231) builds from a source voice,
  a session voice and `u.Spellings` — so `-pron en red` and plain `red` in
  one Spanish directory, and `-locale gb` vs `-locale us`, all map to one
  file and serve the wrong recording. `<slug>.mp3` also drops `from`, which
  `spokeSource` (audiourl.go:237) and `reportVoice` (main.go:1014) depend on,
  so a cache hit makes that RECORD silent or false. Decide the on-disk key
  against the seam's key, not against the word.
- **PQ-2** [Important] `region-overlap-precedence` Appending `wordRegions` to existing regions creates duplicate spans that silently kill underlines to their right
  A Choice prompt already gets `RegionHeadword` at Line 1 Col 0
  (play_loop.go:1220), and the headword is a deck word, so `wordRegions`
  emits a second region on the same cells. `markClickable` (screen.go:359)
  sorts by Col and advances one `next` cursor, so after the first span
  closes the duplicate at Col 0 can never match again and every later region
  on that line loses its underline. `RegionAt` (screen.go:192) resolves by
  append order, not by narrowest span. State the precedence rule and add a
  property guard that a line's regions are disjoint and ascending — that is
  markClickable's unwritten precondition.
- **PQ-3** [Important] `seam-wrap-site` Moving the wrap into `realDeps` leaves every loop test on an uncached source
  Task 1 Step 3 claims "a test driving any loop through realDeps now gets the
  same source production gets", but no test calls `realDeps()` — `run()`
  takes deps and the suite builds `deps{...}` literals at 21 sites
  (main_test.go:57). Deleting repl.go:257 and replraw.go:264 therefore
  reverses the #2 I-1 reason repl.go:256 states, and the planned
  `TestASittingFetchesARecordingOnce` over `runPlay` would have to hand-wrap
  to work. Wrap at a normalisation point both production and the loop tests
  traverse, or keep the loop wraps and let the guard be the derived thing.
- **PQ-4** [Important] `surface-extent-axis` `surface` is a property of the form, not the write site — and a board never reaches `writeRendered` at all
  There is one prompt write site (play_loop.go:226) serving both Choice and
  Cloze, so it cannot distinguish them; the surface must be derived from the
  form. A board returns at play_loop.go:217 before reaching it, with cells
  going through `boardFooter` (play_loop.go:672) into the footer — so
  "surfaceDeck for a cloze prompt and a board" is unexecutable and
  `TestBoardCellsAreNotColoured` is a pin that cannot fail. The Step 5 guard
  checks only that a surface is passed, not that it is right; the derived
  extent to hang the classification on is `docSyncForms` /
  `TestEveryFormIsEnrolled` (doc_sync_test.go:104,126).
- **PQ-5** [Minor] `guard-cannot-fail` `TestTheRawAudioSourceIsConstructedOnceAndWrapped` is green before the change
  `newHTTPAudioSource` already has exactly one non-test call site
  (main.go:98), so the AST guard passes today and could never have caught the
  bug. Step 2's "Expected: FAIL" is true only of
  `TestRealDepsCarriesACachingAudioSource`. Mutation-sweep it the way Task 4
  Step 5 sweeps its own, or drop it.
- **PQ-6** [Minor] `runtimedirs-fanout` Task 2's file list omits `.gitignore` and `store/store.go`
  Adding "audio" to RuntimeDirs reddens `TestGitignoreCoversRuntimeDirs` —
  .gitignore lists words/ events/ usage/ facts/ items/ and single-sources
  from RuntimeDirs. The new audio methods also have to join the `Store`
  interface (store/store.go:10) that both Mem and storetest satisfy, which
  the file list does not mention.
- **PQ-7** [Minor] `durable-write-ordering` No ARCH-ORDER statement for the first binary artifact written to the working directory
  M1 adds durable state with external events and says nothing about them.
  Name process-death-mid-write and two concurrent `define` processes in one
  directory, and say the bytes go through `writeBytesAtomic` — whose `.tmp-*`
  shadow RuntimeFiles already covers. A bare omission is what an author who
  cannot see the ordering writes.
- **PQ-8** [Minor] `constraints-envelope-missing` M2 carries no operating envelope while M1 carries a good one
  M1 states 10-30KB per recording, ~30MB per 1000-word deck, and a 30-day TTL
  with its cost. M2 says nothing about `deckSpans`: one sentence that it runs
  per write (per question, per reveal) rather than per keystroke, bounded
  against a full-entry reveal, closes ARCH-CONSTRAINTS for the walk.

## Open findings

- **PQ-1** [Critical] `cache-key-narrower-than-seam` The durable audio key drops locale, source voice and `from`, so it collides where the in-memory memo does not
- **PQ-2** [Important] `region-overlap-precedence` Appending `wordRegions` to existing regions creates duplicate spans that silently kill underlines to their right
- **PQ-3** [Important] `seam-wrap-site` Moving the wrap into `realDeps` leaves every loop test on an uncached source
- **PQ-4** [Important] `surface-extent-axis` `surface` is a property of the form, not the write site — and a board never reaches `writeRendered` at all
- **PQ-5** [Minor] `guard-cannot-fail` `TestTheRawAudioSourceIsConstructedOnceAndWrapped` is green before the change
- **PQ-6** [Minor] `runtimedirs-fanout` Task 2's file list omits `.gitignore` and `store/store.go`
- **PQ-7** [Minor] `durable-write-ordering` No ARCH-ORDER statement for the first binary artifact written to the working directory
- **PQ-8** [Minor] `constraints-envelope-missing` M2 carries no operating envelope while M1 carries a good one
