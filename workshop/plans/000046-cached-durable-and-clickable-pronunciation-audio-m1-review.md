# Boundary Review — tools#46 (milestone M1)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..46c77c62dfe3d735ed00882f79e4e61f4bb273d5 |
| command | sdlc milestone-close --issue 46 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T15:13:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M1 design is the right one — dissolving the decorator into `*audioSeam` makes "which loop must remember to wrap" unaskable, and keying on the candidate list rather than the word closes PQ-1 properly, pinned for both store twins. But three things block the boundary, all verified rather than inferred: the suite is **red at HEAD** (`TestPlanTableStatusMatchesTheChangeWindow`); deleting the entire production wiring of `diskAudioCache` from `withStore` leaves the whole `cmd/define` suite **green** (mutation run: failure sets byte-identical), so the milestone's central integration is pinned by nothing; and a `/lang es` mid-session files the recording under `audio/en/` while the word goes to `words/es/`, after which `--forget` prints `removed sycophantic` and the audio file survives — the exact `#10` BR-45 failure this milestone claims to have closed, one directory over.

## 1. Strengths

- **`audioSeam` (`cmd/define/fetch.go:100`) is the right answer to a question four plan rounds got wrong.** The field type, not a guard, enumerates the construction sites, and the pointer genuinely shares one memo across `deps`' by-value copies. The compiler-driven churn through 20-odd test literals is visible in the diff and is the cheap half.
- **`AudioKey` identity vs filing is clean and both halves are pinned** (`cmd/define/store/audio.go:29`). `storetest/suite.go`'s "two voices of one word do not collide" runs against `Mem` and `YAML` alike, so the Critical that reshaped M1 is closed on both twins rather than on the one that was easy.
- **`var _ wordFiler = (*diskAudioCache)(nil)` (`cmd/define/audiodisk.go:66`)** plus the generalised rule in `workshop/lessons.md` — the silent-optional-interface failure recorded in the Log is now a compile error, and the lesson names the family rather than the instance.
- **Blob-before-record ordering** (`cmd/define/store/yaml.go:873`) is implemented as the plan's ARCH-ORDER section specified, with the failure-mode argument stated at the code rather than only in the plan.
- **`deckSpans` (`cmd/define/deckwords.go:48`) buys escape-awareness without teaching `highlightSpans` about ANSI**, and `TestASpanWalkSkipsEscapeSequences` pins it by comparing the coloured walk against the plain one — a pin that fails for the right reason, not a golden.

## 2. Critical findings

**C1 — `/lang` mid-session strands the cache on the old language's shelf, and `--forget` lies about removing it.** `cmd/define/main.go:187` installs `diskAudioCache` over `d.deck` inside `withStore`; `applyLang` (`cmd/define/command.go:416`) then replaces the whole `d.langDeps` — including `deck` — but never rebuilds `d.audio`, so the cache keeps writing through the pre-switch store. Verified by probe against `run()`:

```
FILE audio/en/sycophantic--35084a048a8eddfe.yaml   ← the verdict
FILE words/es/sycophantic.yaml                     ← the word
requested: .../sycophantic_es_es_1.mp3
```

and then, after `-forget sycophantic`: `out="…removed sycophantic\n"`, `SURVIVED audio/en/sycophantic--35084a048a8eddfe.yaml`. Two Done-when rows fail on this path: `Forget` does not take the recordings, and the next sitting (store now `es`) reads `audio/es/` and re-fetches. `applyLang`'s own doc says "anything derived from the language BEFORE a switch must be re-derived BY it. applyLang owns that enumeration" — audio is neither re-derived nor listed among its deliberate exclusions. **Fix sketch:** re-install the disk layer wherever `d.deck` is replaced — extract the `withStore` block into a `func (d deps) withAudioCache() deps` and call it from both `withStore` and `applyLang`; pin it by running `/lang es` through `run()` and asserting the recording lands under `audio/es/` and that `--forget` takes it. This is the ARCH-ORDER event the plan's enumeration missed: it named process death, two processes and blob/record ordering, but not the in-session event that mutates the state the cache depends on.

**C2 — the production wiring of the disk cache is pinned by nothing.** Replacing the whole `if d.deck != nil && d.audio != nil && …` block in `cmd/define/main.go:187-190` with a no-op leaves `go test ./cmd/define/ -count=1` with an identical failure set (18 git-dependent guards in both runs, nothing else). Every disk-cache test in `audiodisk_test.go` hand-wires `newAudioSeam(newDiskAudioCache(...))`, so the file's own claim — "the line the tests exercise is the line production runs (#2 I-1) is the argument this file leans on twice" — is not true of the line that actually installs the layer. This is the same shape as the failure the Log records (`forWord`'s return type silently disabling the cache); a production-path test would also have caught C1. **Fix sketch:** one test that drives `run()` twice in a `t.TempDir()` against `fakeCDN` and asserts the second run makes no request. ARCH-MOCK: the fake exists and is good; what is missing is a test where production flow and test flow share the boundary.

**C3 — the tree is red at the review head.** `go test ./cmd/define/` at `46c77c6` with a clean worktree: `TestPlanTableStatusMatchesTheChangeWindow` fails — `000046-audio-cache-and-deck-words-plan.md` calls `RenderOpts` **modified**, but the window does not touch its declaration (`cmd/define/render.go:12-41`). The plan's own bullet says "`RenderOpts` *(modified)* — no field changes; `Vocab` STAYS", so the row contradicts the prose beside it. **Fix sketch:** change the row's status to `unchanged` (the guard accepts it, and the declaration genuinely is untouched) or drop the row.

## 3. Important findings

**I1 — the `many` axis is declared but unguarded, and the guard's export still says "both axes".** `perWordDir` gained `many` (`cmd/define/store/yaml.go:731`) with a comment arguing "the classification is what the guard checks" — but `PerWordDirsForTest` (`cmd/define/store/export_test.go:11`) exposes only `Path` and `Scoped`, and `TestPerWordDirsCoverEveryRuntimeDir` checks only those two. The next directory where a word owns several files can be declared `many:false` and `Forget` will silently leave its files, which is precisely the drift the second axis was added to stop. The export helper's doc comment ("on both axes") is now a stale restatement of a three-field struct. **Fix:** expose `Many` and assert it against the directory's actual filename shape (e.g. every file present matches `<slug>.yaml` iff `!many`).

**I2 — the degrade-never-fail row is half-delivered, and a new comment claims coverage that does not exist.** Plan Task 3 Step 5 commits to pinning "an unwritable directory, a corrupt file, a store that could not open". `TestPlaybackSurvivesAnUnusableCache` covers only "no store" and "no word". `failingStore.Audio`/`SetAudio` were added at `cmd/define/history_store_test.go:100-106` with the comment "so the 'playback survives an unusable cache' property is driven against a store that errors rather than one that is merely absent" — `failingStore` appears in no audio test. `YAML.Audio`'s two defensive branches (corrupt record → `warnf`; record present, blob gone → treat as absent) are reached by no test at all; `cmd/define/store/yaml_test.go` contains no audio case.

**I3 — the M1 regression is not pinned at the loop it regressed in.** Plan Task 1 Steps 5-6 are explicit: "`runPlay` needed no wrap line and never will. Pin the behaviour anyway, because the behaviour is what the learner meets" — `fakeCDN + runPlay, two questions on one word`. The delivered `TestASittingFetchesARecordingOnce` (`cmd/define/play_loop_test.go:4328`) never calls `runPlay`; it calls `seam.Fetch` three times, which is what `TestAudioSeamServesRepeatsFromMemory` already asserts. The trailing `any(d.audio).(*audioSeam)` assertion is a real guard against the field type changing back, but it is not the behavioural pin the plan named.

**I4 — README and atlas do not document the new working-directory artifact.** `audio/` is a new runtime directory and the first **binary** artifact `define` writes, and `.gitignore` gained it — but the README's working-directory listing (`cmd/define/README.md:501-520`, which enumerates `words/`, `events/`, `facts/`, `items/`, `lang.txt`, `user-model.*`) and the atlas's file-tree block (`atlas/define.md:605-616`, which additionally lists `usage/`) both went untouched. The atlas gained good prose about the cache in the audio section; the enumeration a reader greps did not. ARCH-PURPOSE shadow-sweep: `RuntimeDirs` has four derived consumers and two hand-maintained restatements, and only the derived ones updated.

**I5 — the plan's historical `## Revisions` prose was overwritten, not appended to.** The retired-symbol guard (`retiredSymbolNames` now holds `cachingAudioSource`) was satisfied by a blind substitution through the plan's completed revision entries, leaving ungrammatical text that destroys the record: "the decorator it replaced's doc comment records", "the taxonomy the decorator it replaced already documents", "every non-test the wrap that used to be remembered call must be a subset of its own membership", "**So the fix is not a better enumeration but an operation that does not need one.** the wrap that used to be remembered becomes idempotent". AGENTS.md is explicit that revisions are appended, not overwritten. **Fix:** rewrite those four sentences so they read as English and preserve what round 2/3/4 actually said (e.g. "the memo decorator's doc comment records…", "every non-test call of the retired constructor…").

## 4. Minor findings

- `audioNameSep`'s safety argument is false: `Slug` **can** emit `--`. Probe: `Slug("re-") == "re--ddf427"`, `Slug("anti-") == "anti--0aa4e1"`, `Slug("well- known") == "well--known-885c00"`. So `--forget re` globs prefix `re--` and deletes `re-`'s recordings. Cost is one wasted refetch, not data loss — but the invariant written in the comment is not the one `Slug` provides.
- `YAML.Audio` reads the blob with an unbounded `os.ReadFile` where the network path caps at `maxAudioBytes` — a hand-edited or truncated-then-grown file in `audio/` is loaded whole (ARCH-SECURE: an artifact is not trustworthy just because this program wrote it).
- `SetAudio` with `rec.Missing` skips the blob write but does not remove a pre-existing `.mp3`, so a hit that later becomes a verdict leaves an orphan blob until `Forget`. Not served (the record gates it); just debris.
- Plan's pure-entity table names `audioKey`/`audioRecord`; the code exports `AudioKey`/`AudioRecord`. `cmd/define/store/audio_test.go`, named in Task 2's Files list and in "Test surface", was never created — its rows landed in `storetest/suite.go` instead, which is arguably the better home but is not what the plan says.
- `TestClozeOptionsAreClickableButNotColoured` asserts only `admitsColour()` on three form names; it never asserts a click target. The clickability half lives in `TestADeckSurfaceIsClickableButUncoloured`. The name over-claims.
- The plan file carries zero ticked steps across both chunks while the issue's `## Plan` ticks `M1`. Tracking lives in two places and only one moved.
- `mergeRegions` is described in a Core-concepts bullet but has no row in either entity table.

## 5. Test coverage notes

The pure layer is genuinely well covered and runs without IO: `deckwords_test.go` pins escape-awareness by differential walk, longest-phrase-wins, per-line coordinates, BR-14 coverage on coloured text, and disjointness with headword precedence. `storetest/suite.go`'s four new rows hold `Mem` and `YAML` to the same contract, including the voice-collision Critical and the TTL. What is missing is at the seams, and it is the same gap twice: **nothing exercises the assembled production object.** C2 is the sharp version (the wiring can be deleted with no red), C1 is what that gap let through, and I2/I3 are the rows the plan named and the diff did not deliver. Plan Task 6 Step 5 also asks for a mutation proving `TestEveryFormHasASurface` reddens when a form arrives unclassified; no such sweep is recorded, and the test's mechanism (grep the file for the form name in quotes) would pass for a name mentioned anywhere in `deckwords.go`, including a comment.

## 6. Architecture

- **ARCH-DRY — pass with a note.** One span walk feeds both consumers; `vocabularyFor`/`deckVocabulary` is the right split. `TestASittingFetchesARecordingOnce` duplicates `TestAudioSeamServesRepeatsFromMemory`'s assertions (see I3), and `diskAudioCache.keyFor`'s `k.Word != ""` is a second, weaker statement of `AudioKey.ok()`.
- **ARCH-PURE — pass.** `deckSpans`, `wordRegions`, `mergeRegions`, `surfaceOf`, `AudioKey`, `AudioRecord` are pure and tested directly; the IO sits in `diskAudioCache` and `YAML`.
- **ARCH-PURPOSE — flag (C1, I4).** The single source is `RuntimeDirs`; its derived consumers updated and its two hand-maintained restatements (README listing, atlas tree) did not. And the Done-when "`Forget` takes a word's recordings with it" holds only on the path with no language switch.
- **ARCH-MOCK — flag (C2).** `fakeCDN` at the wire plus a real `t.TempDir()` store is the right rig, and `audiodisk_test.go` uses it well. But the fake is only ever reached through hand-assembled layering, so production flow and test flow do **not** share the boundary — which is the condition this principle sets.
- **ARCH-CONSTRAINTS — pass with a note.** No eviction, decided with a number (~30MB for a 1000-word deck) rather than omitted; `deckSpans` runs once per write, not per keystroke. The unbounded disk read is the one place the envelope is not enforced.
- **ARCH-SECURE — flag (Minor).** The record parse degrades visibly and correctly; the blob read does not bound its input, and the `--` separator's stated safety property does not hold.
- **ARCH-ORDER — flag (C1).** Process death, two processes and blob/record ordering are enumerated and implemented. The event the enumeration missed is the one the caller cannot make atomic: `/lang` arriving between two fetches, which silently re-points the state the cache is filed against.

For the close boundary: M2's Done-when asks for "a guard [that] walks the surfaces rather than a person listing them, so a new write site is covered by construction". Today `writeRendered` has exactly one production caller (`writeWords`), so the property holds — but nothing prevents a fourth write site from calling `writeRendered` directly. That guard is still owed.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000046-audio-cache-and-deck-words-plan.md` recording:
- **`RenderOpts` is `unchanged`, not `modified`** — the bullet already says "no field changes; `Vocab` STAYS", and the guard measured the contradiction (C3).
- **`audioKey`/`audioRecord` shipped as exported `AudioKey`/`AudioRecord`**, because they cross the package boundary; and `store/audio_test.go` was not created — its rows are `storetest/suite.go` conformance rows instead, held against both twins.
- **`mergeRegions` belongs in an entity table**, not only in a bullet.
- **Task 1 Steps 5-6 were not delivered as written** — the sitting-level pin does not drive `runPlay` (I3).
- **Task 3 Step 5 was partially delivered** — "unwritable directory", "corrupt file" and "store that errors" are still open (I2).
- **The ARCH-ORDER enumeration is incomplete**: add the in-session `/lang` switch as an event, with the answer for what happens to the disk layer (C1).
- Restore the four sentences in the round-2/3/4 revision entries that the retired-symbol substitution mangled (I5).

```findings
findings:
  - id: new
    severity: Critical
    family: lang-switch-derivation
    title: |
      A /lang switch strands the audio cache on the previous language's shelf, and --forget reports success while the file survives
    detail: |
      withStore (cmd/define/main.go:187) installs diskAudioCache over d.deck; applyLang
      (cmd/define/command.go:416) replaces the whole d.langDeps including deck but never
      rebuilds d.audio. Verified by driving run() with "/lang es\nsycophantic\n": the word
      lands in words/es/sycophantic.yaml while the record lands in
      audio/en/sycophantic--35084a048a8eddfe.yaml, and a following -forget sycophantic
      prints "removed sycophantic" with that file still on disk. Two Done-when rows fail on
      this path, and applyLang's own doc claims it owns the re-derivation enumeration.
      ARCH-ORDER: this is the external event the plan's enumeration did not name.
  - id: new
    severity: Critical
    family: production-wiring-unpinned
    title: |
      Deleting the entire production wiring of diskAudioCache from withStore leaves the suite green
    detail: |
      Replacing the block at cmd/define/main.go:187-190 with a no-op produces a failure set
      byte-identical to the unmutated baseline (18 git-dependent guards in both runs, nothing
      else). Every disk-cache test hand-wires newAudioSeam(newDiskAudioCache(...)), so
      audiodisk.go's own claim that "the line the tests exercise is the line production runs"
      is untrue of the line that installs the layer — the same shape as the wordFiler
      signature bug the issue Log already records. ARCH-MOCK: the fake is good, but no test
      runs the stack through the production boundary.
  - id: new
    severity: Critical
    family: red-at-boundary
    title: |
      The suite is red at the review head — TestPlanTableStatusMatchesTheChangeWindow fails
    detail: |
      go test ./cmd/define/ at 46c77c6 with a clean worktree: the plan calls RenderOpts
      "modified" but the window does not touch its declaration (cmd/define/render.go:12-41).
      The plan's own bullet says "no field changes; Vocab STAYS", so the table row contradicts
      the prose beside it. Change the status to "unchanged" or drop the row.
  - id: new
    severity: Important
    family: unguarded-classification-axis
    title: |
      perWordDir's new `many` axis is declared but no guard checks it, and the export helper still says "both axes"
    detail: |
      PerWordDirsForTest (cmd/define/store/export_test.go:11) exposes only Path and Scoped, so
      TestPerWordDirsCoverEveryRuntimeDir cannot see `many`. A future directory where a word
      owns several files can be declared many:false and Forget will silently leave them — the
      exact drift the second axis was added to stop, and what yaml.go:731's own comment claims
      the guard prevents.
  - id: new
    severity: Important
    family: claimed-coverage-absent
    title: |
      The degrade-never-fail rows are half-delivered and failingStore.Audio claims coverage with zero call sites
    detail: |
      Plan Task 3 Step 5 commits to pinning an unwritable directory, a corrupt file and a
      store that could not open. Only "no store" and "no word" are pinned. The new
      failingStore.Audio/SetAudio (cmd/define/history_store_test.go:100-106) carry a comment
      saying they drive that property, but failingStore appears in no audio test, and
      YAML.Audio's corrupt-record and missing-blob branches are reached by nothing.
  - id: new
    severity: Important
    family: pin-not-at-the-loop
    title: |
      TestASittingFetchesARecordingOnce never calls runPlay, so the M1 regression is not pinned where it lived
    detail: |
      Plan Task 1 Steps 5-6 specify a behavioural pin at the loop through the CDN recorder —
      "fakeCDN + runPlay, two questions on one word". The delivered test
      (cmd/define/play_loop_test.go:4328) calls seam.Fetch three times, which
      TestAudioSeamServesRepeatsFromMemory already asserts. The trailing field-type assertion
      is a real guard, but it is not the loop-level behaviour the plan named.
  - id: new
    severity: Important
    family: runtime-artifact-undocumented
    title: |
      README's and the atlas's working-directory listings do not mention audio/, the new (and first binary) runtime artifact
    detail: |
      .gitignore derived the new directory from RuntimeDirs, but the two hand-maintained
      restatements did not: cmd/define/README.md:501-520 and atlas/define.md:605-616 both
      enumerate every other runtime artifact and omit audio/<lang>/. The atlas gained prose
      about the cache elsewhere; the enumeration a reader greps did not move. ARCH-PURPOSE
      shadow-sweep.
  - id: new
    severity: Important
    family: revision-overwritten
    title: |
      The plan's historical Revisions prose was overwritten by a blind symbol substitution and no longer reads as English
    detail: |
      Satisfying the retired-symbol guard rewrote completed revision entries in place, against
      AGENTS.md's append-don't-overwrite rule, leaving text such as "the decorator it
      replaced's doc comment records", "the taxonomy the decorator it replaced already
      documents", "every non-test the wrap that used to be remembered call must be a subset",
      and "the wrap that used to be remembered becomes idempotent". The record of what rounds
      2-4 decided is now unreadable.
  - id: new
    severity: Minor
    family: prefix-separator-invariant
    title: |
      audioNameSep's claim that a slug cannot contain "--" is false, so --forget re deletes re-'s recordings
    detail: |
      Probe: Slug("re-") == "re--ddf427", Slug("anti-") == "anti--0aa4e1",
      Slug("well- known") == "well--known-885c00". removeByPrefix globs "<slug>--", so
      forgetting "re" reaches "re-"'s files. Cost is one wasted refetch, but the invariant
      written into the comment is not the one Slug provides.
  - id: new
    severity: Minor
    family: unbounded-input-read
    title: |
      The cached blob is read with an unbounded os.ReadFile where the network path caps at maxAudioBytes
    detail: |
      cmd/define/store/yaml.go:833 loads the whole file into memory. ARCH-SECURE: a persisted
      artifact that an older version, another process, or a hand edit may have written is not
      trustworthy just because this program produced it.
  - id: new
    severity: Minor
    family: stale-blob-on-verdict
    title: |
      SetAudio with Missing set skips the blob write but does not remove an existing .mp3
    detail: |
      A key that was a hit and later becomes a verdict leaves an orphan blob until Forget.
      Not served — the record gates it — but it is debris in a directory with no eviction.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      The plan's entity tables drift from the code: audioKey/audioRecord vs AudioKey/AudioRecord, no store/audio_test.go, mergeRegions has no row
    detail: |
      The types are exported because they cross the package boundary; the named colocated
      test file was never created and its rows landed in storetest/suite.go instead, which is
      arguably the better home but is not what the plan says.
  - id: new
    severity: Minor
    family: test-name-overclaims
    title: |
      TestClozeOptionsAreClickableButNotColoured asserts only colour and never a click target
    detail: |
      It checks admitsColour() on three form names. The clickability half is in
      TestADeckSurfaceIsClickableButUncoloured, so the name promises coverage the body does
      not contain.
  - id: new
    severity: Minor
    family: tracking-in-two-places
    title: |
      The plan file has zero ticked steps while the issue's Plan ticks M1
    detail: |
      Both chunks of workshop/plans/000046-audio-cache-and-deck-words-plan.md remain
      unchecked after the work landed, so the plan's own checklist cannot be used to see what
      this boundary delivered.
```
