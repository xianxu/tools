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

---

## Re-review — 2026-09-07T16:22:33-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..9ce223759a0c2aede487ccff39d3104e8b82f2a9 |
| command | sdlc milestone-close --issue 46 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T16:22:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three Criticals from the previous round are genuinely closed, and I verified each by mutation rather than by reading the commit message: deleting the `withStore` block reddens `TestWithStorePutsTheDiskCacheUnderTheMemo`; flipping `many: true → false` reddens three store rows on both twins; re-scoping `audioDir` by language reddens `TestForgetTakesARecordingFetchedInAnotherLanguage` and the `scoped` guard. The suite is green at HEAD under the default and `conformance` tags, with `gofmt -l` and `go vet ./...` clean. Making `audio/` flat is the right call and the argument for it (the digest already carries the voice, so a language shelf is a second answer to one question) is better than the bug it fixes. What remains is one class of defect, repeated: **the reversal was applied to the code paths and not to the restatements of the superseded shape** — README, atlas, two doc comments in the very file that changed, `perWordDir`'s own type doc, and five sites in the plan all still describe `<slug>--<digest>` filed flat in one directory and removed by prefix glob. Nothing here blocks the boundary; all of it is cheap.

## 1. Strengths

- **`audioDir` flat, argued rather than patched** (`cmd/define/store/yaml.go:181-194`). The fix could have been "rebuild `d.audio` in `applyLang`". Choosing instead to remove the shelf makes the whole `/lang` ordering question disappear, and the comment says why in terms of the model rather than the bug.
- **A directory per word replaces the separator claim** (`cmd/define/store/audio.go:45-58`). "The fix is not a rarer separator — any separator has to be reasoned about against `Slug`'s alphabet, and that reasoning is what was wrong" is the class-level answer, and the `re`/`re-` conformance row plants the fixture that actually reaches the branch.
- **`TestWithStorePutsTheDiskCacheUnderTheMemo`** (`cmd/define/audiodisk_test.go:171-197`) constructs no `diskAudioCache` at all — it builds `deps` the way `run()` does, calls `withStore`, and counts requests. Mutation-verified: the production line is now unremovable in silence.
- **The `many`-axis pin was moved to where the naming convention is known** (`cmd/define/store/yaml_test.go:724-737`). The comment explains why the obvious guard (plant files, see what `Forget` takes) is self-fulfilling, and the behavioural pin in `storetest` reddens under mutation on both twins. That reasoning is worth more than the fix.
- **`workshop/lessons.md:3782-3831`** generalises correctly: "a mutation that does not compile is not a passing mutation", the compile-time assertion for optional-capability interfaces, and "what exact input reaches the line I am claiming to pin?" all name families rather than instances.

## 2. Critical findings

None.

## 3. Important findings

**N1 — the design reversal was swept through the code and not through the ten restatements of the shape it replaced.** The shipped layout is `audio/<slug>/<digest>.mp3` + `<digest>.yaml`, removed by `os.RemoveAll` on an exact directory name (probed). Still describing the superseded scheme:
- `cmd/define/store/yaml.go:725-726` — "so it is globbed by prefix rather than removed by name"
- `cmd/define/store/yaml.go:741-742` — `many`'s own field doc, "Forget globs its prefix instead of removing one name" — 60 lines above `removeWordTree`'s "AN EXACT NAME, not a prefix"
- `cmd/define/store/yaml.go:731-737` — `perWordDir`'s type doc still says "on BOTH axes … TWO axes", the same drift the commit fixed in `PerWordDirsForTest`
- `cmd/define/store/yaml_test.go:727` — repeats the prefix claim
- `cmd/define/README.md:515` — `audio/sycophantic--51f6…mp3`, a path that does not exist
- `atlas/define.md:614,618` — `audio/<slug>--<digest>.mp3` and "Forget globs the word's prefix"
- `workshop/plans/…-plan.md:153-155, 202-204, 254, 450, 456-460` — the per-language dir, the prefix glob, the filename

Fix: don't patch the seven sites. Write the enumeration the sweep needs and run it — `grep -rn -- 'slug>--\|<digest>\|per-language audio\|glob' cmd/define/README.md atlas/ cmd/define/store/ workshop/plans/000046-*` — and add the rule to `workshop/lessons.md`: *a design reversal is landed only when that grep is clean in the same commit.* This is the 2nd finding in family `runtime-artifact-undocumented`, and the third time this issue has paid for the same rule (PQ-10 recorded it, `workshop/targets/derived-restatement.md` exists for it). ARCH-PURPOSE shadow-sweep: the derived consumer (`.gitignore`, from `RuntimeDirs`) is right; every hand-maintained one is wrong.

**BR-6 (still open) — the corrupt-file and unwritable-directory rows, and `YAML.Audio`'s two defensive branches.** The `failingStore` half is properly closed (`cmd/define/audiodisk_test.go:158-161` is a real call site). But Plan Task 3 Step 5 still reads "An unwritable directory, a corrupt file, a store that could not open… Pin each", and `cmd/define/store/yaml.go:846-848` (corrupt record → `warnf`) and `:851-857` (record present, blob gone) are reached by no test. Both are ~10 lines to pin: write garbage into the record `.yaml`; delete the `.mp3` and leave the record. Either pin them or revise Step 5 to say which rows the design dropped and why.

**BR-9 (still open) — the plan was edited in place again, with no `## Revisions` entry for this round.** `## Revisions` still ends at plan-quality round 4; the whole M1 remediation delta (the `RenderOpts` row removed, Step 6 rewritten, the layering argument changed) is invisible in the record, against AGENTS.md §1's append-don't-overwrite rule. The blind-substitution damage was partly repaired but the same edit introduced a fresh break at `workshop/plans/…-plan.md:365-366`: "`repl.go:257` and `replraw.go:264` go, along / along with the decorator itself". The retired-symbol guard genuinely forbids restoring the old name verbatim — so the answer is an appended entry saying the historical text now spells the retired symbol as "the memo decorator", not another in-place pass.

## 4. Minor findings

- **BR-12 (still open)** — reverting the `os.Remove(blob)` at `cmd/define/store/yaml.go:882-884` leaves `go test ./cmd/define/store/` fully green (mutation-verified). "A verdict replaces the recording it supersedes" asserts only through `Audio`, which gates on the record and therefore cannot see the orphan. The fix is real and the pin is not.
- **BR-13 (still open)** — the pure table still says `audioKey`/`audioRecord` (code exports `AudioKey`/`AudioRecord`); `store/audio_test.go` is still cited at plan lines 216 and 414 and was never created; `mergeRegions` still has no row; the `audioSeam` sketch declares `misses map[string]missRecord` where the code has `map[string]struct{}`; the `diskAudioCache` bullet says "the outermost decorator" where the memo is outermost.
- **BR-15 (still open)** — 0 of 45 plan checkboxes ticked, while the issue's `## Plan` ticks M1 and M2.
- **N2** — `applyLang`'s doc (`cmd/define/command.go:397-399`) enumerates what is re-derived and what is "deliberately NOT here"; `d.audio` — which holds a `store.Store` bound to the pre-switch language — appears in neither. Correct today only because `audioDir` is flat. 2nd in family `lang-switch-derivation`; the rule, not the line: *every `deps` member holding a language-derived object appears in that list or in its exclusion clause, with the reason.*
- **N3** — `readCapped` (`yaml.go:806-817`) truncates via `LimitReader` where its doc says "refusing anything past max"; the record `.yaml` beside the blob is still read with an unbounded `os.ReadFile` (`yaml.go:838`); no test pins the cap. 2nd in family `unbounded-input-read`; the rule: *state once, at the store boundary, which persisted reads are bounded and why the rest are not* — the sibling read in the same function is the evidence that fixing the named instance did not settle it.
- `deckSpan.Word`'s doc says "the deck key" but is assigned `sp.text` (`cmd/define/deckwords.go:60-66`), so `Word` and `Text` are always the same value — two fields, one fact.

## 5. Test coverage notes

The store layer is now held properly: six conformance rows run against `Mem` and `YAML` alike, and three of the boundary's four fixes are mutation-verified red-without-the-fix. The gaps are all one shape — **a fix whose only witness is the code that implements it**: the orphan-blob removal (BR-12), the 4MB read cap (N3), and `YAML.Audio`'s corrupt-record and missing-blob branches (BR-6). Each is cheap and each is the class the issue's own Log already names twice (the `wordFiler` signature, the production wiring).

Two smaller notes. `TestEveryFormHasASurface` greps `deckwords.go` for `"<form>"`, so a form name appearing anywhere in the file — including a comment — satisfies it; the Task 6 Step 5 mutation proving it reddens for an unclassified form is still not recorded. And the claim at `cmd/define/play_loop_test.go:4325-4327` that "a sitting cannot be driven in-process anyway" is broader than true: `runPlay` cannot (real `isTerminal` syscall), but `playSession` is driven directly by several tests and reaches `playRegion` at `play_loop.go:351`, so a two-clicks-one-request pin at the sitting level is available if it is ever wanted.

## 6. Architecture

- **ARCH-DRY — flag (N1).** One span walk feeds colour and clicks; `vocabularyFor`/`deckVocabulary` is the right split. The flag is the seven hand-maintained restatements of one on-disk fact.
- **ARCH-PURE — pass.** `AudioKey`, `AudioRecord`, `deckSpans`, `wordRegions`, `mergeRegions`, `surfaceOf` are pure and tested with no IO; `diskAudioCache` and `YAML` hold the filesystem. `readCapped` is correctly on the IO side.
- **ARCH-PURPOSE — flag (N1, BR-6).** The purpose is delivered — a recording survives the process, `Forget` takes it on every voice and from every language. The flag is the answer-the-class rule: the reversal fixed the code sites it named and left the enumeration unswept.
- **ARCH-MOCK — pass.** `fakeCDN` at the wire plus a real `t.TempDir()` store, and after BR-3 production flow and test flow share the boundary. That is the condition this principle sets, and it is now met.
- **ARCH-CONSTRAINTS — pass.** No eviction, decided with a number. `writeWords` is gated on `written != s.Index`, so the span walk is per question, not per frame. The disk write on the play path is ~30KB atomic — below any perceptible budget.
- **ARCH-SECURE — pass with a note (N3).** The record parse degrades visibly and the blob is now capped; the record file beside it is not, and `readCapped`'s doc overstates what it does.
- **ARCH-ORDER — pass.** Process death mid-write (blob before record), two processes in one directory, and the `/lang` event are all now answered — the last by removing the state that made ordering matter, which is the stronger answer. `forWord` returns a copy, so no shared mutable state escapes the seam.

For the close boundary: M2's Done-when "every deck word is clickable wherever it is written" is ticked with body text saying the guard it asked for is **not** what shipped and is carried to `#30`. That is honest, but a ticked row whose own text says it was not delivered is what the close's plan-check gate exists to catch — settle it before `sdlc close` rather than at it.

## 7. Plan revision recommendations

Append **one** `## Revisions` entry to `workshop/plans/000046-audio-cache-and-deck-words-plan.md` (do not edit the body again in place) recording:

- **`audioDir` is FLAT, not per-language**, and why — superseding line 254, Task 2 Step 2 (lines 440-444), and the `audio/<lang>/…` paths at lines 153-155.
- **A word owns a DIRECTORY, not a filename prefix** — superseding the `perWordDir` bullet (lines 199-204), Task 2 Step 4 (lines 456-460) and Task 3 Step 3's prefix glob (line 450).
- **`audioKey`/`audioRecord` shipped exported**, and `store/audio_test.go` was never created — its rows are `storetest` conformance rows; drop it from "Test surface" (line 216) and from Task 2's Create list (line 414).
- **`mergeRegions` needs a row** in the pure-entity table.
- **Task 3 Step 5 is partially delivered** — name which of "unwritable directory" and "corrupt file" are being pinned now and which are dropped, with the reason.
- **The `audioSeam` sketch drifted** — `misses` is `map[string]struct{}`; `diskAudioCache` is beneath the memo, not "outermost".
- **A note that the completed round-2/3/4 entries spell the retired symbol as "the memo decorator"** because `retiredSymbolNames` forbids the literal name — so the next reader does not read the substitution as damage, and repair the "go, along / along with" break at lines 365-366.
- **Tick the delivered steps** (currently 0 of 45), so the plan's checklist can show what this boundary closed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The third shape shipped: deps.audio is *audioSeam, both wrap lines are gone, and the retired-symbol guard now forbids the old constructor.
  - id: BR-2
    disposition: addressed
    note: |
      audio/ is flat; re-scoping it by language reddens TestForgetTakesARecordingFetchedInAnotherLanguage and the scoped guard (mutation-verified).
  - id: BR-3
    disposition: addressed
    note: |
      Mutation-verified: deleting the withStore block reddens TestWithStorePutsTheDiskCacheUnderTheMemo with the right message.
  - id: BR-4
    disposition: addressed
    note: |
      Full suite green at HEAD under default and conformance tags; gofmt -l and go vet ./... clean.
  - id: BR-5
    disposition: addressed
    note: |
      Mutation-verified: many true to false reddens three storetest rows on both twins plus the language-switch test.
  - id: BR-6
    disposition: not-addressed
    note: |
      failingStore now has a call site, but the corrupt-file and unwritable-directory rows and YAML.Audio's two defensive branches remain unpinned.
  - id: BR-7
    disposition: addressed
    note: |
      Renamed to what it asserts and the argument recorded; note that playSession IS drivable in-process, so the "a sitting cannot be driven" claim is broader than true.
  - id: BR-8
    disposition: addressed
    note: |
      Both enumerations now list audio/ — but with the superseded filing scheme; raised separately below.
  - id: BR-9
    disposition: not-addressed
    note: |
      No Revisions entry was appended for this round and the body was edited in place again, introducing a fresh break at plan lines 365-366.
  - id: BR-10
    disposition: addressed
    note: |
      Directory-per-word removes the separator claim entirely; the re/re- conformance row plants a fixture that reaches the branch on both twins.
  - id: BR-11
    disposition: addressed
    note: |
      Blob read now capped by readCapped; the sibling record read and the doc/behaviour mismatch are raised below in the same family.
  - id: BR-12
    disposition: not-addressed
    note: |
      Mutation-verified: reverting the os.Remove(blob) leaves the store package fully green — the test asserts only through Audio, which gates on the record.
  - id: BR-13
    disposition: not-addressed
    note: |
      audioKey/audioRecord, store/audio_test.go and the missing mergeRegions row all persist, and new drift arrived with the reversal.
  - id: BR-14
    disposition: addressed
    note: |
      The test now drives writeWords and asserts two RegionWord click targets as well as the absence of colour.
  - id: BR-15
    disposition: not-addressed
    note: |
      Still 0 of 45 plan checkboxes ticked while the issue's Plan ticks M1 and M2.
findings:
  - id: new
    severity: Important
    family: runtime-artifact-undocumented
    title: |
      The audio filing reversal was swept through the code paths and left ten restatements of the superseded scheme, two of them doc comments on the field it changed
    detail: |
      This is the 2nd finding in family `runtime-artifact-undocumented`, and the third time this
      issue has paid for the rule (PQ-10 recorded it; workshop/targets/derived-restatement.md
      exists for it). Do NOT patch the sites. The shipped layout is audio/<slug>/<digest>.mp3
      plus <digest>.yaml, removed by RemoveAll on an exact directory name (probed). Still
      describing <slug>--<digest> filed flat and removed by prefix glob: yaml.go:725-726,
      yaml.go:741-742 (the `many` field's own doc, 60 lines above removeWordTree's "AN EXACT
      NAME, not a prefix"), yaml.go:731-737 (perWordDir's type doc still says "on BOTH axes …
      TWO axes" — the same drift this commit fixed in PerWordDirsForTest), yaml_test.go:727,
      README.md:515, atlas/define.md:614 and :618, and plan lines 153-155, 202-204, 254, 450,
      456-460. The rule: a design reversal is landed only when the greppable enumeration of
      restatements is swept in the same commit — write the grep, run it, and put the rule in
      lessons.md. ARCH-PURPOSE shadow-sweep: the derived consumer (.gitignore, from RuntimeDirs)
      is right; every hand-maintained one is wrong.
  - id: new
    severity: Minor
    family: lang-switch-derivation
    title: |
      applyLang's enumeration of what a language switch re-derives still does not account for d.audio, which holds a store bound to the pre-switch language
    detail: |
      This is the 2nd finding in family `lang-switch-derivation`. The behaviour is correct now
      only because audioDir is flat, so d.audio's stale store reference is harmless — but
      applyLang (cmd/define/command.go:397-399) explicitly lists its deliberate exclusions
      ("Deliberately NOT here: d.history") and d.audio is in neither list. The rule, not the
      line: every deps member holding a language-derived object appears in applyLang's
      enumeration or in its exclusion clause, with the reason — here, that audio/ carries no
      language shelf.
  - id: new
    severity: Minor
    family: unbounded-input-read
    title: |
      Only the blob was bounded — the record YAML beside it is still read with an unbounded os.ReadFile, readCapped truncates where its doc says it refuses, and no test pins the cap
    detail: |
      This is the 2nd finding in family `unbounded-input-read`, and the sibling read is in the
      same function as the one the first finding named: yaml.go:838 reads the record with
      os.ReadFile while yaml.go:843 caps the blob. readCapped's doc says "refusing anything past
      max" but io.LimitReader truncates, which matches the network path's behaviour and not the
      comment. Do not just cap the record. The rule: state once, at the store boundary, which
      persisted reads are bounded and why the rest (words/, facts/, items/) are not — a per-site
      fix is what left the sibling in place here.
```

---

## Re-review — 2026-09-07T16:42:38-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..26f0b9b8a00b497b44bcd9a5bf4a901c0fb31af1 |
| command | sdlc milestone-close --issue 46 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T16:42:38-07:00 |
| verdict | unknown |

## Review

Failed to authenticate: OAuth session expired and could not be refreshed

---

## Re-review — 2026-09-07T17:01:41-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..26f0b9b8a00b497b44bcd9a5bf4a901c0fb31af1 |
| command | sdlc milestone-close --issue 46 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-07T17:01:41-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three fixes this round *claims* are real and I verified each by reverting it in a scratch clone rather than by reading the commit message: reverting `os.Remove(blob)` reddens `TestAVerdictDeletesTheRecordingItSupersedes` (BR-12); making the corrupt-record branch return an error, making a record-with-no-blob read as a hit, and swapping `readCapped` back to `os.ReadFile` each redden their own subtest of `TestAudioDegradesOnEveryDamagedFile` (BR-6, and the cap clause of BR-18); deleting the `withStore` block still reddens `TestWithStorePutsTheDiskCacheUnderTheMemo`. The suite is green at HEAD (`go test ./...`, plus `-tags conformance` on `cmd/define/...`), `gofmt -l` and `go vet ./...` (both tag sets) clean, and 45 of 45 plan checkboxes are now ticked (BR-15). What does not survive scrutiny is the sweep BR-16 asked for: the rule was written into `lessons.md` but the enumeration it prescribes is a **line-oriented grep**, and this codebase wraps comments at ~80 columns, so it structurally cannot see a restatement that spans a line break — three present-tense restatements of the superseded filing scheme are still in the tree, two of them in the store package the sweep was about, one of them thirty lines above the doc comment that contradicts it in the same file. Separately, one new Important: an empty recording is written to disk and served as a permanent, never-expiring hit, which is the exact outcome the diff's own new missing-blob branch calls "a lie the caller cannot detect" — fixed at the instance (`readCapped` errored) rather than at the class (`len(data) == 0`). Nothing here blocks; all of it is cheap.

## 1. Strengths

- **The degrade rows are pinned at the layer that can actually produce the damage** (`cmd/define/store/yaml_test.go:797-921`). Corrupt record, vanished blob, unwritable path and oversized blob are driven against a real `t.TempDir()`, with `DigestForTest` added for the one thing a test needs and nothing more. All four mutation-verified red-without-the-fix.
- **`TestAVerdictDeletesTheRecordingItSupersedes` asserts through the filesystem, and says why** (`yaml_test.go:889-897`). The comment names the reason the previous pin could not fail — `Audio` reads the record first, so it cannot see the orphan — which is worth more than the assertion.
- **`perWordDir`'s type doc is now the argument, not the label** (`cmd/define/store/yaml.go:731-742`). Each of the three axes is introduced by the bug the previous set could not see. That is the shape a classification should carry.
- **`removeWordTree`'s doc keeps the false claim as history rather than deleting it** (`yaml.go:806-812`). "The fix is not a rarer separator, because any separator has to be reasoned about against `Slug`'s alphabet and that reasoning is what was wrong" is the class-level answer; the `re`/`re-` conformance fixture plants a case that actually reaches the branch.
- **`lessons.md:3833-3860` distinguishes historical mentions from present-tense drift explicitly.** That distinction is what makes the rule usable rather than a grep that fires on every comment explaining why the code looks as it does — and it is the sentence that identifies the residual sites below as real.

## 2. Critical findings

None.

## 3. Important findings

**N1 — an empty recording is cached durably and served as a permanent hit, which is the failure the sibling branch was written to prevent.** `YAML.Audio` (`cmd/define/store/yaml.go:870-877`) guards the missing-blob case on `readCapped` returning an *error*; a blob that reads back as zero bytes returns `data=[]`, the full record, and `nil`. `diskAudioCache.Fetch` (`cmd/define/audiodisk.go:80-84`) then treats it as a hit, and by design a hit never expires ("bytes that answered once are still the right bytes" — `store/audio.go:95-97`), so the word can never play again from that directory until `--forget`. Two reachable inputs: a hand-truncated `.mp3` (the code twice calls this directory untrusted, hand-editable input), and a `200` with an empty body — `httpAudioSource.Fetch` (`cmd/define/fetch.go:64-72`) returns `data, u, nil` on any 200 regardless of length, and `SetAudio` has no emptiness guard (`yaml.go:917`). Probe against the production layering:

```
first run:  0 bytes, err=<nil>
second run: 0 bytes, from="http://127.0.0.1:.../a.mp3" err=<nil>
requests:   [/a.mp3]                     ← one request, ever
store says: 0 bytes, rec={From:… At:… Missing:false}
```

This is behaviour drift from the method's own stated contract — `Audio`'s doc says "every way this can go wrong … reads as 'nothing cached'", and the branch eight lines above calls exactly this outcome "an EMPTY recording, which plays as silence and reads as 'this word has no audio' — a lie the caller cannot detect". The fix answered the instance (`readCapped` errored) and not the class (the payload cannot be a recording). **Fix sketch:** treat `!rec.Missing && len(data) == 0` as "nothing cached" in both twins (`YAML.Audio`, `Mem.Audio`), and refuse the write in `SetAudio` rather than storing it; add a `storetest` row so both twins are held, and one `audiodisk_test.go` row proving a 200-with-empty-body is re-asked next run. ARCH-SECURE: the failure path currently substitutes a fabricated value that downstream reads as evidence — `reportVoice` prints a record naming the URL that "answered".

**BR-16 (still open) — the sweep landed but its enumeration cannot see wrapped comments, and three present-tense restatements survive.** See the disposition below; the fix is to the grep and the lesson, not to the three sites.

**BR-17, BR-18, BR-13 (still open)** — see dispositions.

## 4. Minor findings

- Two comments justify a decision by citing the README for something it does not say: `readCapped`'s "The directory is documented as inspectable and hand-editable" (`yaml.go:826-827`) and `audioBlobExt`'s "which the README documents as an invited workflow" (`yaml.go:886-887`). `cmd/define/README.md` describes what `define` writes; it never invites inspection or editing. The claim is load-bearing — it is the whole ARCH-SECURE argument for the 4MB cap — so either say it once in the README and cite that, or drop the citation and keep the cap on its own merits.
- `diskAudioCache.keyFor` gates on `k.Word != ""` (`audiodisk.go:109`) where the store gates on `AudioKey.ok()`. Harmless today (`Slug` never returns `""`), but it is a second, weaker statement of one predicate across a package boundary — ARCH-DRY.
- Plan Task 3 Step 6 ("`--forget` end to end … the file is gone") is ticked, but the delivered pin is at the store (`TestForgetTakesARecordingFetchedInAnotherLanguage`, `storetest`), not through `run()`/`forgetWord`. The store-level coverage is the substantive half and I would not spend a test on the thin caller — but the step should say where it landed.
- `atlas/define.md:613` inserts `audio/` between `usage/` and `facts/`; `RuntimeDirs` order puts it last. Cosmetic, but the block reads as the tail-append rule it documents.

## 5. Test coverage notes

The store layer is now held where it matters and the pins fail for the right reasons — I confirmed four of them by mutation, and each error message named the actual consequence rather than the assertion. The remaining gap is the one N1 names and it is the same shape the issue's `## Log` already records twice (the `wordFiler` signature, the production wiring): **a guard written against the error path of an operation rather than against the validity of its result.** `TestAudioDegradesOnEveryDamagedFile` covers *damaged* files thoroughly and does not cover a file that reads back cleanly as nothing. One `storetest` row (`an empty recording is not a hit`) closes it on both twins.

Two smaller notes carried forward and still true: `TestEveryFormHasASurface` (`deckwords_test.go:152-159`) greps `deckwords.go` for `"<form>"`, so a form name appearing anywhere in the file — including a comment — satisfies it, and the Task 6 Step 5 mutation proving it reddens for an unclassified form is still not recorded. And nothing drives two clicks on one word through `playSession` to assert one request; `playSession` *is* drivable in-process (`runPlay` is not), so that pin is available if the close boundary wants it.

## 6. Architecture

- **ARCH-DRY — flag (BR-16, BR-13).** One span walk feeds colour and clicks; `vocabularyFor`/`deckVocabulary` is the right split and `deckSpans` correctly refuses to teach `highlightSpans` about ANSI. The flag is the one on-disk fact restated in five hand-maintained places, three of them still wrong.
- **ARCH-PURE — pass.** `AudioKey`, `AudioRecord`, `deckSpans`, `wordRegions`, `mergeRegions`, `surfaceOf` are pure and tested with no IO; `readCapped`, `diskAudioCache` and `YAML` hold the filesystem. No "pure" entity needs a mock to run.
- **ARCH-PURPOSE — flag (N1, BR-16).** The milestone's purpose is delivered: a recording survives the process, an unrecorded word is asked once, `Forget` takes every voice from every language. Both flags are the instance-vs-class rule — the missing-blob guard fixed the error path and not the payload, and the reversal sweep fixed the sites the finding listed while the enumeration that would have found the rest cannot see them.
- **ARCH-MOCK — pass.** `fakeCDN` at the wire plus a real `t.TempDir()` store, and `TestWithStorePutsTheDiskCacheUnderTheMemo` constructs no `diskAudioCache` at all — mutation-verified this round: deleting the `withStore` block reddens it with the right message. Production flow and test flow share the boundary, which is the condition this principle sets.
- **ARCH-CONSTRAINTS — pass.** No eviction, decided with a number rather than omitted; the disk read is capped at 4MB and the cap is now pinned; `writeWords` runs per question, not per keystroke.
- **ARCH-SECURE — flag (N1, BR-18).** `removeWordTree` is safe by construction (`Slug` yields exactly one path element, and `RemoveAll` takes an exact name). The blob read is bounded and its degrade path is now driven. What is not held: an empty payload is trusted as evidence, and the record `.yaml` beside the capped blob is still read with an unbounded `os.ReadFile` in the same function.
- **ARCH-ORDER — pass.** Process death mid-write (blob before record), two processes in one directory, and the `/lang` event are all answered — the last by removing the state that made ordering matter, which is the stronger answer. `forWord` returns a copy, so no shared mutable state escapes the seam; the memo's lock is held only around the map, which is correct for a synchronous fetch path (and `#45`'s async playback is the change that would need this revisited).

**For the close boundary:** the M2 Done-when "every deck word is clickable wherever it is written" is ticked with body text saying the guard it asked for is *not* what shipped and is carried to `#30`. That is honest, but a ticked row whose own text says it was not delivered is what the close's plan-check gate exists to catch — settle the row's wording before `sdlc close` rather than at it.

## 7. Plan revision recommendations

Append **one** further `## Revisions` entry to `workshop/plans/000046-audio-cache-and-deck-words-plan.md` (do not edit the body again in place beyond the four factual corrections below):

- **Four residual drifts in the body**, which the round-2 entry was supposed to supersede and does not reach: line 217 still names `store/audio_test.go` as the colocated test surface (never created); line 232 still calls `diskAudioCache` "the outermost decorator" (the memo is outermost, and `deps.audio` is no longer a decorator chain); line 255 still calls `audioDir` "the per-language audio directory" (it is FLAT); line 346's `audioSeam` sketch declares `misses map[string]missRecord` where the code has `map[string]struct{}`.
- **The reversal sweep's enumeration is line-oriented and this codebase wraps comments** — record the corrected form (fold line breaks and comment markers before matching) and note the three sites it recovers, so `lessons.md`'s rule is usable next time rather than merely written.
- **An empty payload is not a hit** — add it to Task 2's record/blob contract, beside the missing-blob rule it belongs with, and to Task 2 Step 7's `storetest` row list.
- **Task 3 Step 6 landed at the store, not end to end** — say so, with the reason (`forgetWord` is a thin caller of `Store.Forget`, which is where the removal lives).

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      All four degrade branches mutation-verified red-without-the-fix; failingStore now has a real call site at audiodisk_test.go:152.
  - id: BR-9
    disposition: addressed
    note: |
      A Revisions entry was appended for this round; the substitution artifacts and the "go, along / along with" break are gone (grep clean).
  - id: BR-12
    disposition: addressed
    note: |
      Mutation-verified: reverting os.Remove(blob) reddens TestAVerdictDeletesTheRecordingItSupersedes by name.
  - id: BR-13
    disposition: not-addressed
    note: |
      mergeRegions, AudioKey/AudioRecord and Task 2's Create list are fixed; plan lines 217, 232, 255 and 346 still drift from the code.
  - id: BR-15
    disposition: addressed
    note: |
      45 of 45 plan checkboxes ticked, none left unchecked.
  - id: BR-16
    disposition: not-addressed
    note: |
      Code paths, README, atlas and the lessons rule all landed; the prescribed grep is line-oriented and misses three wrapped restatements.
  - id: BR-17
    disposition: not-addressed
    note: |
      cmd/define/command.go is not in this window at all; d.audio is still in neither applyLang's enumeration nor its exclusion clause.
  - id: BR-18
    disposition: not-addressed
    note: |
      The cap is now pinned (mutation-verified), but the sibling record read is still unbounded and the boundary-level statement was not written.
findings:
  - id: new
    severity: Important
    family: degenerate-payload-trusted
    title: |
      An empty recording is written to disk and served as a permanent, never-expiring hit — the outcome the new missing-blob branch exists to prevent
    detail: |
      YAML.Audio (cmd/define/store/yaml.go:870-877) guards the missing-blob case on readCapped
      returning an ERROR; a blob that reads back as zero bytes returns empty data with the full
      record and nil, and diskAudioCache.Fetch (cmd/define/audiodisk.go:80-84) serves it as a
      hit. Hits never expire by design (store/audio.go:95-97), so the word can never play again
      from that directory until --forget. Two reachable inputs: a hand-truncated .mp3 (the code
      twice calls this directory untrusted, hand-editable input) and a 200 with an empty body,
      which httpAudioSource.Fetch (cmd/define/fetch.go:64-72) returns as success and SetAudio
      (yaml.go:917) stores without an emptiness guard. Probe against the production layering:
      first run 0 bytes err=nil, second run 0 bytes err=nil, one CDN request ever, record on
      disk with Missing:false. This is drift from Audio's own contract ("every way this can go
      wrong reads as nothing cached") and from the sibling branch eight lines above, which
      calls exactly this "an EMPTY recording ... a lie the caller cannot detect". The fix
      answered the instance (readCapped errored) and not the class (the payload cannot be a
      recording): treat !rec.Missing and len(data)==0 as nothing-cached in BOTH twins, refuse
      the write in SetAudio, and add a storetest row plus one audiodisk row so an empty 200 is
      re-asked next run. ARCH-SECURE: the failure path substitutes a fabricated value that
      reportVoice then prints as the URL that answered.
  - id: new
    severity: Minor
    family: unsourced-cross-reference
    title: |
      Two comments justify the 4MB cap and the .mp3 extension by citing a README passage that does not exist
    detail: |
      readCapped says "The directory is documented as inspectable and hand-editable"
      (cmd/define/store/yaml.go:826-827) and audioBlobExt says "for a human browsing the
      directory, which the README documents as an invited workflow" (yaml.go:886-887).
      cmd/define/README.md describes what define writes and never invites inspection or
      editing; grepping it for hand-edit/editable/inspect/browse returns nothing. The claim is
      load-bearing — it is the entire ARCH-SECURE argument for bounding the read. The rule: a
      comment that justifies a decision by citing another document must be checkable against
      it. Either state it once in the README and cite that, or drop the citation and keep the
      cap on its own merits.
```
