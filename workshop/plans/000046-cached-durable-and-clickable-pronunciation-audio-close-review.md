# Boundary Review — tools#46 (whole-issue close)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..cd89212d69ddb0e490267141b6351d73f329794a |
| command | sdlc close --issue 46 |
| reviewer | claude |
| timestamp | 2026-09-07T17:31:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The window ships a durable, per-voice audio cache under `audio/<slug>/<digest>` and a single span walk that feeds both colour and clicks, and the M1 half is now genuinely well pinned — I mutation-verified the round-4 fix (dropping either half of the emptiness guard reddens a named test; dropping both reddens the audiodisk row). Suite is green at HEAD (`go test -count=1 ./...`), `go vet` and `gofmt` clean. What keeps this from SHIP is one class the diff repeats from its own M1 review: **M2's production wiring is unpinned** — passing `nil` for the vocabulary at all three `writeWords` call sites leaves the entire `cmd/define` suite green, so every deck-word click and every colour decision is asserted only against hand-passed fixtures; and the **empty-payload predicate stops one layer short** — the memo still serves a zero-byte 200 as a hit for the whole sitting, with a `from` URL that `reportVoice` prints as a record. Three prior Minors (BR-13, BR-17, BR-18) are still open at HEAD; BR-16, BR-19 and BR-20 are genuinely closed.

**1. Strengths**

- `cd89212`'s emptiness guard is the rare fix that is pinned where only it can answer: `yaml_test.go:929-968` reaches the write half through the filesystem and the read half through a hand-truncated blob, and reverting either one alone goes red by name (verified in a scratch checkout). `lessons.md:3861` states the rule that made that necessary.
- `deckSpans` (`cmd/define/deckwords.go:47`) is the right seam — it borrows `visibleIndex` + the existing `highlightSpans` instead of teaching either about the other, so the escape-awareness is structural rather than a rule someone must remember. `TestASpanWalkSkipsEscapeSequences` compares the coloured walk against the plain one, which is the assertion that cannot pass vacuously.
- `mergeRegions` (`deckwords.go:96`) writes down `markClickable`'s previously unwritten precondition and turns it into a checked property, with the precedence argument (`RegionHeadword` carries the lookup key) stated where the code enforces it.
- `store/audio.go:20-44` keying on the candidate list rather than the word, with `Forget` reachability kept as a separate concern (`dir()`/`stem()`), is the correct decomposition; `storetest`'s `re`/`re-` row drives the exact pair that broke the superseded scheme, on both twins.
- `audioDir` being flat, with the reason (`the digest already carries the voice`) rather than just the bug, is a better answer than the one it replaced.

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/define/play_loop.go:229`, `:478`, `cmd/define/main.go:877` — the write door's production wiring is unpinned.** I replaced `deckVocabulary(d)` with `nil` at all three `writeWords` call sites in a scratch checkout: `go test ./cmd/define/` stayed green (the only failures were the git-dependent repo guards, which fail identically at unmutated baseline in a non-git tree). Every `writeWords` test passes a literal `deckOf(...)`; nothing observes the vocabulary arriving from `deps`. So M2 could be entirely dead in production — the same silent-nothing this issue already shipped once (`forWord`'s interface mismatch) and the same class as BR-3 one milestone earlier. **This is the 2nd finding in family `production-wiring-unpinned`.** Do not just add a fixture for the three sites: state the rule — *a capability passed from `deps` into a write door must be pinned by a test that observes it arriving from `deps`, not from a literal* — and write the enumeration it implies (the `writeWords` call sites and `deckVocabulary`'s callers) so a fourth site is covered by construction. `TestWithStorePutsTheDiskCacheUnderTheMemo` is the shape that worked for M1. Note also that `deckwords_test.go:264-266` claims the opposite in prose ("so it fails if a call site stops passing the vocabulary") — it cannot, and the issue's own Done-when row already concedes the site-level guard did not ship.
- **`cmd/define/fetch.go:206` — an empty 200 is remembered as a hit for the whole sitting.** `audioSeam.fetch` stores `c.hits[key] = cachedAudio{data, from}` with no emptiness predicate, so one CDN response with a zero-byte body is served as a hit to every replay in the process, `speak` writes a 0-byte `.mp3` for `afplay`, and `reportVoice` prints the URL as the voice that answered. Probed: one seam, two `FetchFor` calls, one request, `bytes=0 from="http://…/a.mp3" err=<nil>` both times. **This is the 2nd finding in family `degenerate-payload-trusted`.** Round 4 fixed the two store twins and the write, which is what BR-19 literally asked for, but the class it named — *the payload cannot be a recording* — has more members than the store. Do not patch the memo alone: enumerate every layer that can produce, remember, persist or play a fetch result (`httpAudioSource.Fetch`, `audioSeam.hits`, `diskAudioCache.put`, both `SetAudio`s, `speak`) and put the predicate at the point where the payload is first judged — a 0-byte 200 is not an answer, so `httpAudioSource` continues to the next candidate — which makes every layer below correct by construction instead of by four separate guards. The `store.Store` interface doc (`store.go:88-91`) belongs in that sweep too: it documents the verdict form of `SetAudio` and is silent that a non-`Missing` call with empty data is refused, so a third implementation would not know the contract.

**4. Minor findings**

- `workshop/issues/000046-…md:275-277` — the Done-when row says `TestAPromptRegionCoversTheTextItClaims` exercises the new kind "without being edited to know about it"; that test loops over `promptRegions(q)` only (`play_loop_test.go:4276`), which never yields a `RegionWord`. The invariant *is* pinned, by `TestWordRegionsCoverTheTextTheyClaim`. **2nd finding in family `claimed-coverage-absent`** — the rule: a Done-when row names the guard that can reach the property, and a row is verified by asking which assertion would go red, not by which test's name sounds adjacent.
- `cmd/define/store/yaml.go:892` / `cmd/define/fetch.go:35` — the 4MB ceiling is stated twice with nothing holding the two equal (family `two-statements-one-fact`); the comment justifies the duplication but the drift is unguarded. Exporting one and having the other read it costs a line.
- `cmd/define/deckwords.go:26` — `deckSpan.Word`'s doc says "the deck key", but it is assigned `sp.text` (display text). Harmless today because `AudioCandidates` lower-cases and `store.Key` normalises before filing, but the field doc is a claim a caller would rely on (family `doc-claims-unbacked`).

**5. Test coverage notes**

`storetest`'s six new audio rows hold both twins to one contract and are the right home for the round-trip, collision, TTL and Forget properties; `yaml_test.go`'s degrade table drives all four damaged-file branches including the oversized blob, which closes BR-18's "no test pins the cap" half. What is missing is at the two ends: nothing observes M2's capability reaching the door from production (finding above), and nothing exercises `audioSeam`'s hit path with a degenerate payload — the audiodisk row deliberately uses two separate seams, which is exactly the layer where the empty is still remembered.

**6. Architectural notes**

- **ARCH-DRY** — flag (Minor): the 4MB ceiling stated in two packages. Otherwise a strong pass: `deckSpans` grew a consumer instead of a second walk, `audioSeam.fetch` takes the resolved source as a parameter so `FetchFor` cannot create a second memo, and `memVocabulary` stays the one set implementation.
- **ARCH-PURE** — pass. `deckwords.go` is pure and unit-tested with no IO (`TestEveryFormHasASurface` reads source, which is a guard, not the entity); the IO sits in `diskAudioCache` and `YAML`, both injected.
- **ARCH-PURPOSE** — flag. The shadow-sweep on the empty-payload class stops at the store (finding above). The docs sweep is otherwise complete: `.gitignore` derives from `RuntimeDirs`, and README/atlas were corrected rather than left as hand-maintained restatements.
- **ARCH-MOCK** — pass, and notably good: `fakeCDN` is at the wire with a request recorder, `diskRig` uses a real temp directory rather than a map, `failingStore.Audio` now has a call site, and `storetest` runs both Store implementations through one suite.
- **ARCH-CONSTRAINTS** — pass. Eviction is a decision with a number (plan:240, ~30MB for a 1000-word deck), the verdict TTL is 30 days with the reason, both read paths are capped, and `Load` is idempotent so the per-write vocabulary lookup is one map read.
- **ARCH-SECURE** — mostly pass: `Slug` guarantees a single safe path element, so `RemoveAll` cannot escape `audio/`; the blob read is capped; the README now actually carries the hand-editability claim the comments cite. The remaining gap is BR-18's: the record YAML beside the blob is still read with an unbounded `os.ReadFile` (`yaml.go:859`) in the same function that caps the blob at `:871`.
- **ARCH-ORDER** — pass. `SetAudio` writes the blob before the record, and a verdict removes the blob first, so every crash window costs a refetch rather than silence; `Audio` treats a missing/empty blob as nothing cached, which is the state that makes the ordering safe.

**7. Plan revision recommendations**

- `workshop/plans/000046-audio-cache-and-deck-words-plan.md` — append a Revisions entry for round 4: *"Test surface: `store/audio_test.go` was never written. `AudioKey`/`AudioRecord` are held by `storetest`'s conformance rows and `yaml_test.go`, which is the better home because the properties are cross-implementation — but the plan claimed a file that does not exist."* (This is BR-13's remaining half; `TestPlanCitesTestsThatExist` only checks backticked `Test*` names, so a cited test FILE is unguarded — worth extending that guard to backticked `*_test.go` paths.)
- Same file — append: *"An empty payload is not a recording at any layer that can remember it,"* recording the layer enumeration once the second Important finding is swept, so the next reader does not re-derive which of the five layers hold the predicate.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      Table names and the mergeRegions row are fixed; the plan still cites `store/audio_test.go`, which does not exist, and TestPlanCitesTestsThatExist checks only Test* names.
  - id: BR-16
    disposition: addressed
    note: |
      yaml.go/README/atlas swept, grep + rule recorded in lessons.md; the plan body is superseded by the appended Revisions entry per AGENTS.md §1.
  - id: BR-17
    disposition: not-addressed
    note: |
      command.go is not in the change window; applyLang's exclusion clause still names only d.history, and d.audio appears in neither list.
  - id: BR-18
    disposition: not-addressed
    note: |
      The cap is now pinned by a test, but the sibling record read at yaml.go:859 is still an unbounded os.ReadFile and no single statement at the store boundary says which persisted reads are bounded.
  - id: BR-19
    disposition: addressed
    note: |
      Mutation-verified: dropping the read guard reddens the truncated-blob subtest, dropping the write guard reddens the filesystem subtest, dropping both reddens TestAnEmptyResponseIsNotCachedAsAHit.
  - id: BR-20
    disposition: addressed
    note: |
      README.md:535 now carries "Look at any of it, and edit it if you like" plus the untrusted-input paragraph, and both comments cite that passage.
findings:
  - id: new
    severity: Important
    family: production-wiring-unpinned
    title: |
      Passing nil for the vocabulary at all three writeWords call sites leaves the whole suite green, so M2's production wiring is unpinned
    detail: |
      Probed in a scratch checkout of cd89212: replacing deckVocabulary(d) with nil at play_loop.go:229, play_loop.go:478 and main.go:877 keeps go test ./cmd/define/ green (only the git-dependent repo guards fail, identically at unmutated baseline). Every writeWords test passes a literal deckOf(...), so nothing observes the capability arriving from deps — M2 could be silently dead exactly as forWord's interface mismatch was. This is the 2nd finding in family production-wiring-unpinned (BR-3 was the same rule at M1). Do NOT just fixture the three sites: state the rule — a capability handed from deps into a write door is pinned by a test that observes it arriving from deps, not from a literal — and write the enumeration (the writeWords call sites, deckVocabulary's callers) so a fourth site is covered by construction. TestWithStorePutsTheDiskCacheUnderTheMemo is the shape that worked. Related: deckwords_test.go:264-266 claims in prose that it "fails if a call site stops passing the vocabulary", which it cannot.
  - id: new
    severity: Important
    family: degenerate-payload-trusted
    title: |
      The memo serves a zero-byte 200 as a hit for the whole sitting, with a from URL reportVoice prints as the voice that answered
    detail: |
      audioSeam.fetch (cmd/define/fetch.go:206) stores c.hits[key] = cachedAudio{data, from} with no emptiness predicate. Probed with one seam and two FetchFor calls against a CDN serving an empty body: one request, both calls return bytes=0 from="http://.../a.mp3" err=nil. speak then writes a 0-byte mp3 for afplay and reportVoice prints a record naming a URL that carried no recording. This is the 2nd finding in family degenerate-payload-trusted. Round 4 satisfied BR-19's literal ask (both store twins plus the write) but not the class it named — the payload cannot be a recording. Do not patch the memo alone: enumerate every layer that can produce, remember, persist or play a fetch result (httpAudioSource.Fetch, audioSeam.hits, diskAudioCache.put, both SetAudio implementations, speak) and put the predicate where the payload is first judged, so a 0-byte 200 is not an answer and every layer below is correct by construction. The store.Store interface doc (store.go:88-91) is part of that sweep: it documents the verdict form of SetAudio and is silent that a non-Missing call with empty data is refused.
  - id: new
    severity: Minor
    family: claimed-coverage-absent
    title: |
      The Done-when row names TestAPromptRegionCoversTheTextItClaims as exercising RegionWord, but that test can only see promptRegions
    detail: |
      workshop/issues/000046-...md:275-277 claims numRegionKinds, TestEveryRegionKindIsActionable, TestAtlasDescribesEveryRegionKind and TestAPromptRegionCoversTheTextItClaims all exercise the new kind unedited. The last one loops over promptRegions(q) (play_loop_test.go:4276), which only ever yields RegionHeadword. The invariant is genuinely pinned — by TestWordRegionsCoverTheTextTheyClaim — so this is a wrong citation rather than a gap. 2nd finding in family claimed-coverage-absent: the rule is that a Done-when row names the guard whose assertion would go red, verified by asking which assertion that is, not by which test name sounds adjacent.
  - id: new
    severity: Minor
    family: two-statements-one-fact
    title: |
      The 4MB audio ceiling is stated twice, in two packages, with nothing holding them equal
    detail: |
      maxAudioBytes (cmd/define/fetch.go:35) and maxAudioBlobBytes (cmd/define/store/yaml.go:892) are the same number written twice; the comment justifies it ("the store cannot import main") but nothing detects drift. Exporting one and reading it from the other costs a line.
  - id: new
    severity: Minor
    family: doc-claims-unbacked
    title: |
      deckSpan.Word is documented as the deck key but holds the display text
    detail: |
      cmd/define/deckwords.go:26 says "Word is the deck key, which is what a click plays"; deckSpans assigns sp.text, so a sentence-initial or differently-spaced match carries the on-screen spelling. Harmless today because AudioCandidates lower-cases and store.Key normalises before filing, but the doc is a claim a caller would rely on.
```

---

## Re-review — 2026-09-07T17:50:46-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..83bab010d5cdec5e5c9bac30d09e29cfeaa10c00 |
| command | sdlc close --issue 46 |
| reviewer | claude |
| timestamp | 2026-09-07T17:50:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 7 of this gate. The window ships a durable per-voice audio cache under `audio/<slug>/<digest>` and one span walk feeding both colour and clicks; the suite is green at HEAD (`go test -count=1 ./...`, 9 packages), `go vet` clean under default/`pty`/`conformance`, `gofmt` clean. Round 6's two Importants are both materially better — the memo no longer serves a zero-byte 200 as a session-long hit, and nilling the vocabulary at a `writeWords` site now reddens a named test (I confirmed both by mutation). Neither is *closed*, though, and both fall short in the same way the findings predicted: BR-22 fixed the layer the finding pointed at (the memo) and left `httpAudioSource` still answering with an empty 200 — and stopping the candidate walk there — plus the `Store` interface doc still silent on the refusal it explicitly named; BR-21's guard reads the argument's *source token* rather than observing the capability arrive from `deps`, so swapping all three sites back to `vocabularyFor(d, opt)` — the exact regression the issue Log records as near-miss #2 — leaves the entire `cmd/define` suite green. Nothing here is a user-facing defect: production is wired correctly today and playback degrades safely. Everything remaining is pin quality and stale prose, which is why this is FIX-THEN-SHIP rather than a seventh blocking round.

**1. Strengths**

- `cd89212`+`83bab01`'s emptiness guards are pinned where only they can answer, and I re-verified it: `yaml_test.go:TestBothHalvesOfTheEmptinessGuardAreReached` reaches the write half through the filesystem (above `MkdirAll`, so no debris directory makes it pass for the wrong reason) and the read half through a hand-truncated blob. `fetch_test.go:TestAZeroByteResponseIsNotAHit` adds the third layer and also asserts the *non*-miss — that the next play re-asks — which is the half a lazier fix would have got wrong.
- `deckSpans` (`cmd/define/deckwords.go:56`) is the right seam: `visibleIndex` + the existing `highlightSpans`, with `at` tracked against `highlightSpans`' documented "spans concatenate to the input exactly" invariant. Escape-awareness is structural, not remembered. `TestASpanWalkSkipsEscapeSequences` compares the coloured walk against the plain one, which cannot pass vacuously.
- `mergeRegions` (`deckwords.go:97`) writes down `markClickable`'s previously unwritten one-cursor precondition and turns it into a checked property, with the precedence argument stated where it is enforced.
- `store/audio.go` keying on the candidate list with `dir()`/`stem()` kept as a separate filing concern is the correct decomposition, and `storetest`'s `re`/`re-` row drives the exact pair that broke the superseded scheme — on both twins.
- `TestWithStorePutsTheDiskCacheUnderTheMemo` (`audiodisk_test.go:169`) is the shape that actually works for wiring: it constructs no `diskAudioCache`, drives `withStore` from a real cwd twice, and asserts through the CDN's request recorder.

**2. Critical findings**

None.

**3. Important findings** (both re-raised under their original ids, not as new)

- **`cmd/define/deckwords_test.go:358` — BR-21's guard catches the token, not the capability.** Mutation-verified in a scratch worktree of HEAD: replacing `deckVocabulary(d)` with `nil` at `play_loop.go:230` reddens `TestEveryWriteWordsCallSitePassesAVocabulary` by name (good), but replacing it with `vocabularyFor(d, opt)` at all three sites (`play_loop.go:230`, `play_loop.go:479`, `main.go:881`) leaves `go test ./cmd/define/` **fully green** — 110s, no failures. That swap is not hypothetical: it is discovery #2 in the issue's own Log ("Clicks were about to depend on colour"), and under `-no-color` it silently removes every click target on every surface. The guard the finding asked for was *a test that observes the capability arriving from `deps`*; what shipped inspects `call.Args[3]` for the identifier `nil`, which is the "a token appears in a body" shape BR-1/PQ-11 rejected for the seam one milestone earlier. `TestClicksSurviveNoColour` cannot cover the gap — it passes a literal `v`. The related false prose BR-21 named (`deckwords_test.go:268-269`, "so it fails if a call site stops passing the vocabulary") is unchanged.
- **`cmd/define/fetch.go:44` — BR-22's sweep stopped one layer above where the payload is first judged.** The memo now refuses an empty body, which closes the user-visible symptom. But `httpAudioSource.Fetch` still `return data, u, nil` for a zero-byte 200 (`fetch.go:70`), which BR-22 named first in its enumeration and which the round-6 review spelled out as "a 0-byte 200 is not an answer, so `httpAudioSource` continues to the next candidate". Consequence today: a word whose *first* candidate serves an empty 200 reports `ErrNoAudio` even when candidate 2 holds a real recording — the walk never gets there. And `cmd/define/store/store.go:90-93` still documents only the verdict form of `SetAudio` and is silent that a non-`Missing` call with empty data is refused, which BR-22 called out by line; a third `Store` implementation would not know the contract (`storetest` would catch it, the doc would not have told it). The predicate now exists at five sites instead of one.

**4. Minor findings**

- `workshop/issues/000046-…md:279-285` — the "every deck word is clickable" Done-when row now carries two false statements, one of them *created* by round 6: it says `TestEveryDeckWordInASittingIsClickable` "fails if a call site stops passing the vocabulary" (it passes a literal), and then concedes "A new `writeWords` call site is covered by review, not by construction — carried to `#30`", which the new AST guard has made stale. **3rd finding in family `claimed-coverage-absent`** — see the findings block for the rule and enumeration.
- Stale restatements left by the *seam* reversal (the filing reversal got a grep; this one did not): `noAudioSource` (`main_test.go:20`) is declared and instantiated nowhere in the package; `play_loop_test.go:64` says "playRig deliberately installs `noAudioSource`" when `playRig` (`:41`) does `newAudioSeam(nil)`; `play_loop_test.go:1444` describes it as "the package's other double"; and the plan's `writeWords` bullet says "the plain door still serves callers with nothing to mark" while `writeRendered` has exactly one caller in the tree — `writeWords` itself. **3rd finding in family `runtime-artifact-undocumented`.**
- `cmd/define/audiodisk.go:88` returns `rec.From` read back from `audio/<slug>/<digest>.yaml` — a file the README now explicitly invites editing — straight into `spokeSource`/`reportVoice`, which the code calls "a RECORD … it survives on a pipe and cannot be taken back". Nothing checks `From` is a member of the `urls` the caller just passed. ARCH-SECURE: the blob got a cap and a payload predicate in this same sweep; the provenance field beside it got neither.

**5. Test coverage notes**

`storetest`'s eight audio rows hold both twins to one contract and are the right home for round-trip, collision, TTL, `re`/`re-` and Forget. `yaml_test.go`'s degrade table drives all four damaged-file branches. What is still unpinned is at the two ends: nothing observes M2's capability reaching the write door *from production* (BR-21, mutation-confirmed above), and no test reaches `httpAudioSource`'s empty-200 branch at the layer where the candidate walk lives — `TestAZeroByteResponseIsNotAHit` uses a single candidate, so the "stops the walk" behaviour is invisible to it. Note also that `yaml_test.go`'s own subtest is named "a blob larger than the cap is **truncated**", which is BR-18's point stated by the test that closes its other half: `readCapped`'s doc says it refuses, and a 4MB-truncated mp3 is served as a hit.

**6. Architectural notes**

- **ARCH-DRY** — flag (Minor, BR-24 open): `maxAudioBytes` / `maxAudioBlobBytes`, one number in two packages with nothing holding them equal. Otherwise strong: one span walk with two consumers, `audioSeam.fetch` taking the resolved source as a parameter so `FetchFor` cannot spawn a second memo, `memVocabulary` still the only set.
- **ARCH-PURE** — pass. `deckwords.go` is pure and unit-tested with no IO; the AST/source-reading tests are guards, not entity tests. IO sits in `diskAudioCache` and `YAML`, both injected.
- **ARCH-PURPOSE** — flag. The shadow-sweep on the empty-payload class stops above `httpAudioSource` and the `Store` doc (BR-22); the class BR-21 named ("pinned by arrival from deps") was answered with a token check. Docs sweep is otherwise complete — `.gitignore` derives from `RuntimeDirs`, README and atlas both corrected.
- **ARCH-MOCK** — pass, and good: `fakeCDN` at the wire with a request recorder, `diskRig` on a real temp directory, `failingStore.Audio` now has a call site, `storetest` runs both `Store` implementations through one suite. No live conformance check exists for the CDN, but that predates this window.
- **ARCH-CONSTRAINTS** — pass. Eviction is a decision with a number, the verdict TTL is 30 days with the reason, `deckSpans` is once-per-write (never per keystroke) with the bound stated, `Load` is guarded so the per-write vocabulary is one map read.
- **ARCH-SECURE** — flag. `Slug` keeps `RemoveAll` inside `audio/`, the blob is capped, the empty payload is refused. Remaining: BR-18's unbounded `os.ReadFile` on the record at `yaml.go:859` in the same function that caps the blob at `:871`, and the unvalidated `From` above.
- **ARCH-ORDER** — pass. Blob-before-record, verdict-removes-blob, `writeBytesAtomic` on both, `Audio` treating a missing/empty blob as nothing cached. `audioSeam`'s mutex is released across `inner.Fetch`, which is correct: a duplicate concurrent fetch is idempotent, and the key is the candidate list.

**7. Plan revision recommendations**

- `workshop/plans/000046-audio-cache-and-deck-words-plan.md` — append a Revisions entry recording that **`store/audio_test.go` was never written** (still cited at plan lines 217 and 421); `AudioKey`/`AudioRecord` are held by `storetest`'s conformance rows and `yaml_test.go`, which is the better home because the properties are cross-implementation. `TestPlanCitesTestsThatExist` only checks backticked `Test*` names, so a cited test *file* is unguarded — worth extending it to backticked `*_test.go` paths.
- Same file — append: **Task 5 Step 4 is ticked for work that did not happen.** `TestAPromptRegionCoversTheTextItClaims` (`play_loop_test.go:4270`) still walks `promptRegions(q)` only; the invariant for `RegionWord` is held by the separate `TestWordRegionsCoverTheTextTheyClaim`. Correct the step and the Verification row 6 to name the guard that actually goes red.
- Same file — append: **the seam reversal's restatement sweep.** The filing reversal got a grep and a lessons.md rule; the `cachingAudioSource` → `audioSeam` reversal did not, and left `noAudioSource` dead plus two comments describing it as installed. Record the grep alongside the filing one.
- Same file — correct the `writeWords` bullet: `writeRendered` no longer serves any caller but `writeWords`.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      Entity-table names and the mergeRegions row are right, but the plan still cites `store/audio_test.go` at lines 217 and 421 and no such file exists.
  - id: BR-17
    disposition: not-addressed
    note: |
      cmd/define/command.go is untouched by the window; applyLang's exclusion clause (command.go:397) still names only d.history and d.audio is in neither list.
  - id: BR-18
    disposition: not-addressed
    note: |
      yaml.go:859 still reads the record with an unbounded os.ReadFile beside the capped blob at :871; readCapped truncates where its doc says it refuses, and yaml_test.go's own subtest is named "is truncated"; no single statement at the store boundary says which persisted reads are bounded.
  - id: BR-21
    disposition: not-addressed
    note: |
      Mutation-verified at HEAD: `nil` at a call site now reddens the new guard, but swapping all three sites to `vocabularyFor(d, opt)` leaves the whole cmd/define suite green — the guard reads the argument's source token, not the capability's arrival from deps, and deckwords_test.go:268-269's false prose is unchanged.
  - id: BR-22
    disposition: not-addressed
    note: |
      The memo layer is fixed and pinned, but httpAudioSource.Fetch (fetch.go:70) still returns a zero-byte 200 as success and stops the candidate walk there, and store.go:90-93 is still silent that a non-Missing SetAudio with empty data is refused — both named explicitly by the finding.
  - id: BR-23
    disposition: not-addressed
    note: |
      The issue row at :275-277 is unchanged, and the same citation appears at plan:623 and plan:731 — where Task 5 Step 4 is ticked for an extension of TestAPromptRegionCoversTheTextItClaims that did not happen.
  - id: BR-24
    disposition: not-addressed
    note: |
      maxAudioBytes (fetch.go:35) and maxAudioBlobBytes (yaml.go:892) are both unchanged; nothing holds them equal.
  - id: BR-25
    disposition: addressed
    note: |
      deckwords.go:25-32 now says "the span's text AS WRITTEN, not the normalised deck key", which matches deckSpans assigning sp.text, and names why it is harmless.
findings:
  - id: new
    severity: Minor
    family: claimed-coverage-absent
    title: |
      The "every deck word is clickable" Done-when row now carries two false statements, one of them created by this round's own fix
    detail: |
      workshop/issues/000046-...md:279-285 says TestEveryDeckWordInASittingIsClickable "drives it through writeWords rather than through the rule, so it fails if a call site stops passing the vocabulary" — it passes a literal deckOf(...) and cannot — and then concedes "A new writeWords call site is covered by review, not by construction — carried to #30", which round 6's TestEveryWriteWordsCallSitePassesAVocabulary made stale in the same commit that left the first clause standing. This is the 3rd finding in family claimed-coverage-absent (BR-6, BR-23 preceding it). Do NOT fix this row. State the rule and sweep the enumeration it implies: a row, a ticked plan step, or a test doc comment that names a guard must name the guard whose assertion would go red, verified by asking which assertion that is — and when a round ADDS a guard, the concessions that said it did not exist are part of the same commit's sweep. The enumeration is the Done-when rows, the ticked plan steps (Task 5 Step 4 and Verification row 6 are both wrong for the same reason), and the doc comments on the tests this window added.
  - id: new
    severity: Minor
    family: runtime-artifact-undocumented
    title: |
      The seam reversal never got the grep the filing reversal got, and left a dead double plus three present-tense claims about it
    detail: |
      noAudioSource (cmd/define/main_test.go:20) is declared and instantiated nowhere in the package after deps.audio became *audioSeam. play_loop_test.go:64 states "playRig deliberately installs noAudioSource AND noAudio:true" when playRig (:41) does newAudioSeam(nil); play_loop_test.go:1444 calls it "the package's other double"; fetch.go:123 refers to "the tests that used to write noAudioSource{}" while the type itself was left behind. Separately, the plan's writeWords bullet says "NEW, beside writeRendered rather than replacing it. The plain door still serves callers with nothing to mark" — writeRendered has exactly one caller in the tree, writeWords itself. This is the 3rd finding in family runtime-artifact-undocumented (BR-8, BR-16). Do NOT patch these four sites. The rule is already in lessons.md for the FILING reversal — "write the grep before the fix, run it after" — and M1 contained a SECOND reversal (cachingAudioSource -> audioSeam) that never got one. Write the grep for the retired seam vocabulary, run it across cmd/, atlas/, README and the plan, and add "one grep per reversal, not one per issue" to the rule.
  - id: new
    severity: Minor
    family: persisted-field-unvalidated
    title: |
      AudioRecord.From is read back from a hand-editable file and printed as the record of which voice answered, with no check that it is one of the URLs asked for
    detail: |
      diskAudioCache.Fetch (cmd/define/audiodisk.go:88) returns rec.From straight from audio/<slug>/<digest>.yaml. speak passes it to reportVoice, whose spokeSource decides by membership in sourceCandidates() and whose own doc says the report "survives on a pipe and cannot be taken back — and a record has to be true". README.md:535 now explicitly invites editing that directory, and this same round added readCapped and the empty-payload predicate on exactly that reasoning — the payload was hardened and the provenance field beside it was not. A hand-edited From flips the fallback announcement in either direction: absent when it should fire, or fired when the source recording really did answer. ARCH-SECURE: an input crossing a process boundary is untrusted even when this program wrote it. Cheapest fix is one membership check where the record is read — if rec.From is not in urls, treat the entry as nothing cached rather than as evidence.
```
