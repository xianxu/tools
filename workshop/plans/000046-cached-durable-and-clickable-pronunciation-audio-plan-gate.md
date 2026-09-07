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
    - "n": 2
      timestamp: "2026-09-07T13:10:28-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Key is now a digest of the candidate list and the record carries `from`; the un-swept restatements are raised separately.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: mergeRegions states precedence and the disjoint-ascending property is checked and mutation-swept.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Wrap stays in the loop and the guard became the derived thing; the guard's own derivation is raised separately.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: surfaceOf(q) switches on q.Form() over docSyncForms, and the board row now asserts a reachable fact.
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
        - id: PQ-6
          disposition: addressed
          round: 2
        - id: PQ-7
          disposition: addressed
          round: 2
        - id: PQ-8
          disposition: addressed
          round: 2
      findings:
        - id: PQ-9
          severity: Critical
          title: 'Task 1''s guard derives the wrong set: `run()`''s deps-taking callees are seven non-loops, and the two functions that actually wrap are not among them'
          detail: |-
            This is the 2nd finding in family `seam-wrap-site`. Do NOT just fix this
            instance. The rule: every statement about where the wrap lives must be
            checked against the real call graph, and the derivation validated in BOTH
            directions. Measured prevalence — 2 of 2 such statements this plan has made
            have been wrong: round 1's `realDeps` traversal, and now "the loops are
            exactly the functions run() dispatches to that take a deps". `run()`
            dispatches to forgetWord (main.go:1092), runPlay (play_loop.go:24),
            runReflect (reflect.go:338), runHarvest (harvest.go:109), repl
            (repl.go:203), ask (ask.go:80), defineOnce (main.go:775) and newCommandCtx
            (command.go:211); the wraps live in replLines (repl.go:263) and runEditor
            (replraw.go:264), reached only via repl.go:232/235. So Step 2's "FAIL,
            naming runPlay and nothing else" would name six, and the guard never
            inspects either function that wraps today. The `len(...) < 2` floor and
            Step 5's delete-and-confirm sweep both only test under-derivation, so
            neither catches this. "Anything reaching d.audio" over-derives too:
            defineOnce reaches it through speak (main.go:1058) and correctly does not
            wrap. Name a predicate that actually separates a replay loop from a
            one-shot, and add the complementary sweep — the guard must be shown NOT to
            name a non-loop.
          family: seam-wrap-site
          round: 2
        - id: PQ-10
          severity: Important
          title: Round 1's key fix was applied at the site the finding named; four other statements of the superseded on-disk shape survive, and Task 2 now contradicts itself
          detail: |-
            This is the 2nd finding in family `cache-key-narrower-than-seam`. Do NOT
            fix the four sites one by one. The rule: the artifact's key and on-disk
            shape are stated ONCE, in `audioKey`/`audioRecord`, and every other mention
            references that definition instead of restating it. Measured prevalence —
            after round 1, 4 of 7 mentions still carry the superseded design: the
            entity table names `audioName`/`audioVerdict` (plan:75-76), types absent
            from the rest of the document; the `perWordDir` bullet says "it is
            `<slug>.mp3` or `<slug>.none`" (plan:201-202); Task 2 Step 3 repeats the
            `.mp3`/`.none` pair (plan:388-389) that Step 4 contradicts eight lines
            later and that `audioRecord` argues against by name (plan:173); and Task 2
            Step 2 still justifies `scoped: true` with the exact sentence PQ-1 quoted
            as false (plan:382) — `audioDir` would scope on `y.lang`, like its siblings
            at store/yaml.go:178-179. An implementer working Task 2 in order builds the
            shape the plan already rejected.
          family: cache-key-narrower-than-seam
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-07T13:17:19-07:00"
      agent: claude
      dispose:
        - id: PQ-9
          disposition: not-addressed
          note: Idempotence fixes over-derivation, but the predicate still excludes runEditor (replraw.go:253), a wrap site driven directly by ~55 tests; plan:290 and plan:752 still name it `replRaw`.
          round: 3
        - id: PQ-10
          disposition: addressed
          note: All four restatements now defer to audioKey/audioRecord, and the scoped:true justification matches store/yaml.go:177-178.
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-07T13:22:37-07:00"
      agent: claude
      dispose:
        - id: PQ-9
          disposition: addressed
          note: 'Verified: replLines (repl.go:249), runEditor (replraw.go:253) and runPlay (play_loop.go:24) all take deps+options, so both guard halves cover the real sites.'
          round: 4
      findings:
        - id: PQ-11
          severity: Important
          title: '"That is ~8 functions" is the 4th wrong prose statement of the wrap set — the real membership is 24, and Step 4 writes the line into functions that never touch audio'
          detail: |-
            This is the 3rd finding in family `seam-wrap-site`; measured prevalence is 4 of 4 prose
            statements about the wrap set being wrong (round 1 `realDeps`, round 2 `run()`'s callees,
            round 3 `replRaw`, now "~8"). Do not fix the instance by writing "24". The rule: the plan
            must carry NO prose statement of the set's membership or size — run the predicate, paste the
            computed set in, and decide before implementation what Step 4 does with the members that
            never read `d.audio` (`vocabularyFor` vocab.go:191, `clozeAsk` cloze.go:189, `todaysQuestions`
            play_loop.go:896, `newCommandCtx` command.go:211, `playRegion` replraw.go:587, `submitLine`
            replraw.go:642) and whether `*deps`/`*options` members are in (`sessionSetLang` command.go:352,
            `applyLang` command.go:400). As written, Step 4 produces ~22 copies of
            `d.audio = newCachingAudioSource(d.audio)`, contradicting the plan's own Architecture line
            ("one construction site") and violating ARCH-DRY: the guard then checks that a token appears
            in a body, not that a caller's source is cached. Name which shape M1 takes — every member
            wraps, only members reaching `d.audio` wrap, or `d.audio` becomes unreachable except through
            an accessor — since that is a seam decision and is hard to reverse once 24 sites carry it.
          family: seam-wrap-site
          round: 4
      blocked: false
content_hash: 8f2924ca524326f69c1d4ee3492b7341e50aa710c0cd67be3411095ab19f8ac8
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

## Round 2 — 2026-09-07T13:10:28-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Key is now a digest of the candidate list and the record carries `from`; the un-swept restatements are raised separately.
- PQ-2 — addressed — mergeRegions states precedence and the disjoint-ascending property is checked and mutation-swept.
- PQ-3 — addressed — Wrap stays in the loop and the guard became the derived thing; the guard's own derivation is raised separately.
- PQ-4 — addressed — surfaceOf(q) switches on q.Form() over docSyncForms, and the board row now asserts a reachable fact.
- PQ-5 — addressed
- PQ-6 — addressed
- PQ-7 — addressed
- PQ-8 — addressed

### Raised

- **PQ-9** [Critical] `seam-wrap-site` Task 1's guard derives the wrong set: `run()`'s deps-taking callees are seven non-loops, and the two functions that actually wrap are not among them
  This is the 2nd finding in family `seam-wrap-site`. Do NOT just fix this
  instance. The rule: every statement about where the wrap lives must be
  checked against the real call graph, and the derivation validated in BOTH
  directions. Measured prevalence — 2 of 2 such statements this plan has made
  have been wrong: round 1's `realDeps` traversal, and now "the loops are
  exactly the functions run() dispatches to that take a deps". `run()`
  dispatches to forgetWord (main.go:1092), runPlay (play_loop.go:24),
  runReflect (reflect.go:338), runHarvest (harvest.go:109), repl
  (repl.go:203), ask (ask.go:80), defineOnce (main.go:775) and newCommandCtx
  (command.go:211); the wraps live in replLines (repl.go:263) and runEditor
  (replraw.go:264), reached only via repl.go:232/235. So Step 2's "FAIL,
  naming runPlay and nothing else" would name six, and the guard never
  inspects either function that wraps today. The `len(...) < 2` floor and
  Step 5's delete-and-confirm sweep both only test under-derivation, so
  neither catches this. "Anything reaching d.audio" over-derives too:
  defineOnce reaches it through speak (main.go:1058) and correctly does not
  wrap. Name a predicate that actually separates a replay loop from a
  one-shot, and add the complementary sweep — the guard must be shown NOT to
  name a non-loop.
- **PQ-10** [Important] `cache-key-narrower-than-seam` Round 1's key fix was applied at the site the finding named; four other statements of the superseded on-disk shape survive, and Task 2 now contradicts itself
  This is the 2nd finding in family `cache-key-narrower-than-seam`. Do NOT
  fix the four sites one by one. The rule: the artifact's key and on-disk
  shape are stated ONCE, in `audioKey`/`audioRecord`, and every other mention
  references that definition instead of restating it. Measured prevalence —
  after round 1, 4 of 7 mentions still carry the superseded design: the
  entity table names `audioName`/`audioVerdict` (plan:75-76), types absent
  from the rest of the document; the `perWordDir` bullet says "it is
  `<slug>.mp3` or `<slug>.none`" (plan:201-202); Task 2 Step 3 repeats the
  `.mp3`/`.none` pair (plan:388-389) that Step 4 contradicts eight lines
  later and that `audioRecord` argues against by name (plan:173); and Task 2
  Step 2 still justifies `scoped: true` with the exact sentence PQ-1 quoted
  as false (plan:382) — `audioDir` would scope on `y.lang`, like its siblings
  at store/yaml.go:178-179. An implementer working Task 2 in order builds the
  shape the plan already rejected.

## Round 3 — 2026-09-07T13:17:19-07:00 (claude) — BLOCKED

### Disposed

- PQ-9 — not-addressed — Idempotence fixes over-derivation, but the predicate still excludes runEditor (replraw.go:253), a wrap site driven directly by ~55 tests; plan:290 and plan:752 still name it `replRaw`.
- PQ-10 — addressed — All four restatements now defer to audioKey/audioRecord, and the scoped:true justification matches store/yaml.go:177-178.

## Round 4 — 2026-09-07T13:22:37-07:00 (claude) — passed

### Disposed

- PQ-9 — addressed — Verified: replLines (repl.go:249), runEditor (replraw.go:253) and runPlay (play_loop.go:24) all take deps+options, so both guard halves cover the real sites.

### Raised

- **PQ-11** [Important] `seam-wrap-site` "That is ~8 functions" is the 4th wrong prose statement of the wrap set — the real membership is 24, and Step 4 writes the line into functions that never touch audio
  This is the 3rd finding in family `seam-wrap-site`; measured prevalence is 4 of 4 prose
  statements about the wrap set being wrong (round 1 `realDeps`, round 2 `run()`'s callees,
  round 3 `replRaw`, now "~8"). Do not fix the instance by writing "24". The rule: the plan
  must carry NO prose statement of the set's membership or size — run the predicate, paste the
  computed set in, and decide before implementation what Step 4 does with the members that
  never read `d.audio` (`vocabularyFor` vocab.go:191, `clozeAsk` cloze.go:189, `todaysQuestions`
  play_loop.go:896, `newCommandCtx` command.go:211, `playRegion` replraw.go:587, `submitLine`
  replraw.go:642) and whether `*deps`/`*options` members are in (`sessionSetLang` command.go:352,
  `applyLang` command.go:400). As written, Step 4 produces ~22 copies of
  `d.audio = newCachingAudioSource(d.audio)`, contradicting the plan's own Architecture line
  ("one construction site") and violating ARCH-DRY: the guard then checks that a token appears
  in a body, not that a caller's source is cached. Name which shape M1 takes — every member
  wraps, only members reaching `d.audio` wrap, or `d.audio` becomes unreachable except through
  an accessor — since that is a seam decision and is hard to reverse once 24 sites carry it.

## Open findings

- **PQ-11** [Important] `seam-wrap-site` "That is ~8 functions" is the 4th wrong prose statement of the wrap set — the real membership is 24, and Step 4 writes the line into functions that never touch audio
