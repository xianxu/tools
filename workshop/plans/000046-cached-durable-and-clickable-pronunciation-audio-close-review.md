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

---

## Re-review — 2026-09-07T18:14:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..a1d339b3ddf1c19d0360a51632133f600f382484 |
| command | sdlc close --issue 46 |
| reviewer | claude |
| timestamp | 2026-09-07T18:14:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Important findings from the prior round are genuinely fixed and I mutation-verified each: passing `deps{}` at `play_loop.go:229` reddens `TestEveryWriteWordsCallSitePassesAVocabulary` by name, and deleting either the memo's or `httpAudioSource`'s zero-byte predicate reddens a differently-named row (`TestAZeroByteResponseIsNotAHit`, `TestAnEmptyBodyDoesNotStopTheCandidateWalk`/`TestAnEmptyResponseIsAVerdictNotAHit`). The suite is green at HEAD (`go test -count=1 ./...`, `go vet ./...`, `gofmt -l` all clean). Nothing blocks the boundary. What I am raising is one user-visible behaviour this window introduced and nothing pins — the word under test in a *meaning* question is now painted with the deck-known green, which is the one place on screen the Spec's own "colour is a discovery signal" criterion says it should not be — plus two class-level gaps: the README/atlas working-directory listings are still hand-maintained restatements of `store.RuntimeDirs` with no derived guard (and the README already omits `usage/` today), and the durable plan has had no `## Revisions` entry since M1 round 2 while five commits changed the design under it. Eight Minor findings from prior rounds remain open and unaddressed.

## 1. Strengths

- **The seam-not-wrap move is the right answer and the atlas records why.** `deps.audio` as `*audioSeam` (`main.go:23-26`) makes "which loop must remember to wrap" unaskable rather than answering it wrongly a fifth time, and `TestOneWordCostsOneFetchHoweverOftenItIsPlayed` (`play_loop_test.go:4351`) asserts the field's *type* as the guarantee rather than asserting a habit.
- **`storetest/suite.go:640-880` holds both twins to the same audio contract**, including the `re`/`re-` prefix pair that only reddens under the superseded filing scheme. `Mem.Audio`/`Mem.SetAudio` (`mem.go:195-232`) carry the same emptiness predicate as YAML, so the fake cannot be more permissive than the real thing (ARCH-MOCK).
- **The write-door guard was fixed at the right level.** Removing the `Vocabulary` argument (`main.go:939`) so `nil` no longer compiles is a strictly better fix than tightening a token check — verified: `writeWords(..., nil, ...)` fails to build, `deps{}` fires the AST guard.
- **`readCapped` + the empty-payload predicate at three layers, each swept separately** (`lessons.md:3860-3880`), is the correct response to "defence in depth is not two pins"; the write half asserts through the filesystem, the read half plants a truncated file.
- **`deckSpans` reuses `highlightSpans` over `visibleIndex`'s plain text** (`deckwords.go:53-77`) rather than teaching the matcher about ANSI — one matcher, two consumers, and the escape-awareness falls out (ARCH-DRY, ARCH-PURE).

## 2. Critical findings

None.

## 3. Important findings

**a. `cmd/define/play_loop.go:229` — the word under test in a *meaning* question is now painted deck-green.**
`surfaceOf("meaning")` is `surfaceProse`, so `colourOutside` runs over the whole prompt — and `Choice.Prompt()` (`play/choice.go:109-118`) is `word + "\n\n" + options`. Probed at HEAD: with a deck holding the headword, the prompt writes `"\n\x1b[1;32mephemeral\x1b[0m\n\n1  …"`. Every meaning question in a sitting now marks its own asked word as one you know, which is the clearest possible case of the Spec's "where finding one is a discovery" failing — and the same token renders in `Render`'s headword style one frame later in the reveal. `TestChoiceOptionGlossesAreColoured` cannot see it: its vocabulary is `deckOf("keel")` and the headword is `sycophantic`. The operator's pty pass checked the cloze frame and the reveal frame, not this one. Fix sketch: either classify the prompt's headword line out of the colour pass (the form knows which part of its prompt is the deck — the same asymmetry `promptRegions` already encodes), or accept it and add the case to `TestChoiceOptionGlossesAreColoured` with a deck containing the headword so the decision is pinned rather than incidental.

**b. `cmd/define/README.md:501` and `atlas/define.md:608` — the working-directory listings restate `store.RuntimeDirs` by hand, unguarded. This is the 4th finding in family `runtime-artifact-undocumented`** (BR-8, BR-16, BR-27 preceding it). Earlier rounds fixed instances — BR-8 added the `audio/` lines to both docs. Do NOT fix another instance; the rule is already written in this repo and half-implemented. `RuntimeDirs` has four consumers: `.gitignore` (derived, guarded by `TestGitignoreCoversRuntimeDirs`), `perWordDirs` (derived, guarded by `TestPerWordDirsCoverEveryRuntimeDir`), and the two doc listings — hand-maintained, nothing checking them. The measured prevalence: **README.md never mentions `usage/` at all**, so the listing is already missing a runtime directory that no one has noticed. The class fix is ~12 lines and the shape exists in the same file: `TestStoreLayoutDocsNameEveryEventKind` (`doc_sync_test.go:622-644`) already walks both docs' fenced layout blocks via `layoutBlockIn`. Add `TestStoreLayoutDocsNameEveryRuntimeDir` over `store.RuntimeDirs` using the same markers, then fix whatever it reddens.

**c. `workshop/plans/000046-audio-cache-and-deck-words-plan.md:690` — the plan has no `## Revisions` entry for close rounds 3–7, and its integration table now states two things the head commit falsified. This is the 2nd finding in family `plan-table-drift`.** The `writeWords` bullet says *"It takes the vocabulary and the surface"* — a1d339b removed the vocabulary argument for exactly the reason BR-21 gave, in the same commit that left this sentence standing. The `audioDir` bullet (`:257`) still says *"the per-language audio directory"*, which the flat-directory reversal falsified in round 2; BR-16's grep was written for the *filing* reversal's vocabulary (`slug>--`, `prefix glob`, `BOTH axes`) and structurally could not match this one — which is precisely what BR-27 asked for ("one grep per reversal, not one per issue") and did not get. State the rule: **a plan artifact's entity description is swept in the same commit that changes the entity's shape, by an APPENDED `## Revisions` entry (AGENTS.md §1), and each reversal earns its own grep.** The enumeration for this round: the `writeWords` signature, the `audioDir` shelf, the empty-payload predicate's three layers, and the two AST guards — none of which the plan records.

## 4. Minor findings

- **Two assertions added in this window cannot fire. This is the 4th finding in family `claimed-coverage-absent`.** Earlier rounds fixed instances (BR-6, BR-23, BR-26 — all prose naming a guard that could not go red); this is the same rule in code. Do NOT fix these two sites — state the rule (*every guard clause needs a witness input that makes it fire; write the witness or delete the clause*) and sweep the clauses this window added. Measured instances: (1) `store/audio.go:63` — `ok()`'s `Slug(k.Word) != ""` is a tautology, because `Slug` never returns `""` (`Slug("")` = `"w-e3b0c4"`). Probed: `SetAudio(NewAudioKey("", urls), …)` files a readable hit under `audio/w-e3b0c4/` that `Forget("")` returns `(false, nil)` for — unreachable from production only because `diskAudioCache.keyFor` happens to check `c.word == ""` separately. (2) `deckwords_test.go:409-413` — the `arg.Name == "nil"` branch is dead: `nil` is not assignable to the struct-typed `deps` parameter, so the package fails to build and the test never runs (verified: `cannot use nil as deps value`). Only the `default` branch does any work.
- `cmd/define/fetch_test.go:295` — `newHTTPAudioSource2(c *fakeCDN) *httpAudioSource { return c.source() }` is a rename wrapper whose name reads as a second constructor beside `newHTTPAudioSource`. Inline `cdn.source()`.
- `cmd/define/store/yaml.go:892` and `cmd/define/news.go:22` — `maxFeedBytes`' comment says it mirrors `maxAudioBytes`, so the 4MB ceiling is now stated in **three** places, not the two BR-24 counted.

## 5. Test coverage notes

- Mutation-verified this round: BR-21's AST guard (fires on `deps{}`), the memo's zero-byte predicate, and `httpAudioSource`'s empty-body skip (reddens two independently-named rows). All three are real pins.
- `TestPlaybackSurvivesAnUnusableCache`'s `failingStore` row now has a real `Audio`/`SetAudio` implementation with a call site, so it is no longer interface padding.
- The gap the diff could still ship: the meaning-prompt colouring above. No test drives `writeWords` with a vocabulary that contains the prompt's headword, so the whole class of "the prompt's own word is marked" is invisible to the suite.
- `TestAPromptRegionCoversTheTextItClaims` still walks `promptRegions(q)` only, so the new kind's coverage invariant on a real prompt is unpinned — the invariant itself holds via `TestWordRegionsCoverTheTextTheyClaim` on synthetic text (see BR-23, still open).

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (Minor above): the 4MB ceiling is three statements of one fact. Otherwise clean; `deckSpans` reusing `highlightSpans` is the diff's best DRY decision.
- **ARCH-PURE** — pass. `deckwords.go` and `store/audio.go` are pure and tested without IO; `diskAudioCache` is the thin shell with `now` injected; the store's degrade branches are driven against a real `t.TempDir()` rather than a mock.
- **ARCH-PURPOSE** — flag (Important b): the shadow-sweep over `RuntimeDirs` finds two hand-maintained consumers, and one of them is already stale for a *different* directory.
- **ARCH-MOCK** — pass. `fakeCDN` is at the wire, `storetest` holds both store twins to one contract, and `fetch_conformance_test.go` is a live check run unfiltered. Note for later: the new "an empty 200 means no recording here" behaviour is a modelled assumption with no conformance row; it is defensive rather than a claim about Google's CDN, so a row is optional — but if the fallback walk ever depends on it, it earns one.
- **ARCH-CONSTRAINTS** — pass. Eviction is declined with a number (10–30KB × 1000 words ≈ 30MB, `plan:240`); `deckSpans` is bounded to one walk per write, not per keystroke; the blob read is capped.
- **ARCH-SECURE** — flag: BR-28 is still open and README.md:538 now makes it worse, claiming *"The program treats what it reads back as UNTRUSTED"* in the same block that invites editing `audio/`. `AudioRecord.From` is read straight off a hand-editable file and printed by `reportVoice` as the voice that answered. The payload was hardened this round; the provenance field beside it was not.
- **ARCH-ORDER** — pass, with a note. `audioSeam` carries two maps (`hits`, `misses`) for one key, declaring four states where three are legal; concurrent first-fetches can land a key in both (hits wins, so behaviour is correct). If `#45`'s async playback makes concurrent fetches routine, collapse them into one map to a tagged `hit | verdict` value before adding a third state.

## 7. Plan revision recommendations

Append one `## Revisions` entry to `workshop/plans/000046-audio-cache-and-deck-words-plan.md` covering close-gate rounds 3–7, and within it:

- **`writeWords` no longer takes a vocabulary.** Correct the integration-table bullet (`:690`): it takes the `deps` and derives the vocabulary itself, because a guard over an argument can only check the argument's source token.
- **`audioDir` is FLAT, not per-language.** Correct `:257`, and record that the language-shelf reversal never got its own grep — the filing reversal's grep could not match it.
- **`store/audio_test.go` was never created.** The "Test surface" paragraph (`:216-219`) still names it; its rows live in `storetest/suite.go`, which is the better home. Say so, and drop the lowercase `audioKey`/`audioRecord` spellings still in the prose at `:203`, `:449`, `:456`, `:827` (the entity table itself is now correct).
- **Task 6 Step 3 and Verification row 6 are wrong for the same reason as the Done-when rows** — Step 3 says "give `writeRendered` the vocabulary and the surface", and row 6 says `TestAPromptRegionCoversTheTextItClaims` was "widened", which it was not.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      Table fixed (AudioKey/AudioRecord, mergeRegions row added); the Test-surface paragraph at :216-219 still names a store/audio_test.go that does not exist, and :203/:449/:456/:827 still spell the types lowercase.
  - id: BR-17
    disposition: not-addressed
    note: |
      applyLang (command.go) still lists neither d.audio among its members nor among its "Deliberately NOT here" exclusions.
  - id: BR-18
    disposition: not-addressed
    note: |
      yaml.go:838 still reads the record with an unbounded os.ReadFile beside the capped blob at :843, and readCapped's doc still says "refusing" where io.LimitReader truncates. The blob cap IS pinned (yaml_test.go, "a blob larger than the cap is truncated") — that half of the finding was inaccurate.
  - id: BR-21
    disposition: addressed
    note: |
      Mutation-verified: replacing d with deps{} at play_loop.go:229 reddens TestEveryWriteWordsCallSitePassesAVocabulary by name; nil no longer compiles.
  - id: BR-22
    disposition: addressed
    note: |
      Mutation-verified separately at both layers; the store.Store interface doc now states the empty-payload refusal.
  - id: BR-23
    disposition: not-addressed
    note: |
      TestAPromptRegionCoversTheTextItClaims still loops promptRegions(q) only (play_loop_test.go:4276); the Done-when row still cites it.
  - id: BR-24
    disposition: not-addressed
    note: |
      maxAudioBytes and maxAudioBlobBytes still independent; news.go:22 makes it three statements of the same number.
  - id: BR-26
    disposition: not-addressed
    note: |
      Both clauses of the Done-when row are unchanged, as are Task 5 Step 4 and Verification row 6.
  - id: BR-27
    disposition: not-addressed
    note: |
      noAudioSource still declared at main_test.go:20 with zero instantiations; the three present-tense claims stand; no grep for the seam reversal and no "one grep per reversal" rule in lessons.md. A THIRD reversal is now visible in the same artifact — the plan's audioDir bullet still says "the per-language audio directory".
  - id: BR-28
    disposition: not-addressed
    note: |
      audiodisk.go:88 still returns rec.From unchecked against urls, and README.md:538 now claims persisted input is treated as untrusted.
findings:
  - id: new
    severity: Important
    family: discovery-colour-scope
    title: |
      A meaning question now paints the word under test deck-green, which is the one word on screen where the discovery rule says it should not
    detail: |
      surfaceOf("meaning") is surfaceProse and Choice.Prompt() is word + "\n\n" + options, so colourOutside runs over the headword line too. Probed at HEAD with a deck holding the headword the prompt writes "\n\x1b[1;32mephemeral\x1b[0m\n\n1  ...". New behaviour — the prompt was written through writeRendered with no colour before. TestChoiceOptionGlossesAreColoured cannot see it because its vocabulary omits the headword, and the operator's pty pass covered the cloze and reveal frames only. Either classify the headword line out of the colour pass or pin the decision with a deck containing the headword.
  - id: new
    severity: Important
    family: runtime-artifact-undocumented
    title: |
      README's and the atlas's working-directory listings restate store.RuntimeDirs by hand with no guard, and README already omits usage/
    detail: |
      This is the 4th finding in family runtime-artifact-undocumented (BR-8, BR-16, BR-27). Do NOT fix another instance. RuntimeDirs has four consumers; the two derived ones are guarded (TestGitignoreCoversRuntimeDirs, TestPerWordDirsCoverEveryRuntimeDir) and the two documentation ones are not. Measured prevalence: README.md contains no occurrence of "usage/" at all, so the listing is missing a runtime directory nobody noticed. The class fix is TestStoreLayoutDocsNameEveryRuntimeDir, mirroring TestStoreLayoutDocsNameEveryEventKind at doc_sync_test.go:622 and reusing its layoutBlockIn helper over both markers.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      The durable plan has no Revisions entry for close rounds 3-7, and its writeWords bullet states a signature the head commit removed
    detail: |
      This is the 2nd finding in family plan-table-drift. The plan's last Revisions entry is "M1 boundary review, rounds 1 and 2"; five commits since then changed the design. plan:690 says writeWords "takes the vocabulary and the surface" — a1d339b removed that argument, in the commit that left the sentence standing. plan:257 still calls audioDir "the per-language audio directory", falsified by the flat-shelf reversal whose vocabulary BR-16's grep could not match. State the rule: a plan artifact's entity description is swept in the same commit that changes the entity's shape, by an APPENDED Revisions entry per AGENTS.md section 1, and each reversal earns its own grep. Enumeration for this round: the writeWords signature, the audioDir shelf, the empty-payload predicate's three layers, the two AST guards.
  - id: new
    severity: Minor
    family: claimed-coverage-absent
    title: |
      Two guard clauses added in this window cannot fire — AudioKey.ok's Word check is a tautology and the AST guard's nil branch does not compile
    detail: |
      This is the 4th finding in family claimed-coverage-absent (BR-6, BR-23, BR-26). Earlier rounds fixed prose instances; this is the same rule in code. Do NOT fix these two sites — state the rule (every guard clause needs a witness input that makes it fire; write the witness or delete the clause) and sweep the clauses this window added. Instances: store/audio.go:63, where Slug never returns "" (Slug("") = "w-e3b0c4"), so ok() is effectively just Digest != "" — probed, an AudioKey with an empty Word files a readable hit under audio/w-e3b0c4/ that Forget("") cannot remove, reachable through the newly exported NewAudioKey/SetAudio surface though not from production. And deckwords_test.go:409, where arg.Name == "nil" is dead because nil is not assignable to the struct-typed deps parameter (verified: "cannot use nil as deps value"), so the package would not build and the test would never run.
  - id: new
    severity: Minor
    family: two-statements-one-fact
    title: |
      newHTTPAudioSource2 is a rename wrapper around cdn.source() whose name reads as a second constructor
    detail: |
      fetch_test.go:295 defines newHTTPAudioSource2(c *fakeCDN) *httpAudioSource { return c.source() } beside the real newHTTPAudioSource. Inline the call.
```

---

## Re-review — 2026-09-07T18:31:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 46 — cached, durable and clickable pronunciation audio |
| repo | tools |
| issue file | workshop/issues/000046-cached-durable-and-clickable-pronunciation-audio.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3effb6462ee74c0e1d45f8b6b5a49de939b2531e..a1f04291e9054f6da8515cf17f4fd8d35522f1eb |
| command | sdlc close --issue 46 |
| reviewer | claude |
| timestamp | 2026-09-07T18:31:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 8's two Important findings are genuinely fixed and I mutation-verified both: reverting the `withoutWord` hold-out reddens `TestAQuestionDoesNotColourTheWordItAsks` by name, and renaming `usage/` in README or `audio/` in the atlas each reddens `TestStoreLayoutDocsNameEveryRuntimeDir` — the guard fires on both documents, so the derived-restatement class fix is real rather than decorative. `newHTTPAudioSource2` is gone (BR-33). The suite is green at HEAD (`go test ./cmd/define/...`, 110s), `go vet -tags conformance` is clean, `gofmt` is clean. What keeps this from SHIP is that the BR-29 fix repeats the exact hole the window's own lessons entry was written about: dropping `q.Word()` at `play_loop.go:229` leaves the **whole** `cmd/define` suite green (mutation-verified), so the production wiring of a fix motivated by production behaviour is pinned by nothing. Alongside that, `AudioKey.Digest` reaches a filesystem path element with no validation — probe-verified writing outside the store directory — while `ok()`'s doc says it reports "whether the key is usable as a filename", and the M2 write door / surface classification is still absent from the atlas while README's colour rule now has an undocumented exception. Eight of the thirteen carried findings remain untouched by this window and are re-raised as `not-addressed`.

### 1. Strengths

- **`exceptWord` is a decorator, not a filter in the walk** (`cmd/define/deckwords.go:229-245`) — the deck is loaded once and shared, so copying it per question to remove one word would be per-frame work for a set that changes once a sitting. The doc says exactly that, and the hold-out applies to `Has` only while `wordRegions` keeps the full vocabulary, so the click target survives by construction rather than by a second rule.
- **The reveal exemption is stated where it is taken and pinned separately** (`play_loop.go:474-481`, `TestARevealStillMarksTheWordItRevealed`) — "the word has been shown and its sentence restored" is a real distinction, and asserting it stops the hold-out from being over-applied later.
- **`TestAQuestionDoesNotColourTheWordItAsks` cannot pass by colouring nothing** (`deckwords_test.go:456-459`) — the second assertion requires `keel` to still be green. That is the assertion the sibling row was missing, and its absence is why the bug was invisible.
- **`TestStoreLayoutDocsNameEveryRuntimeDir` reuses `layoutBlockIn`** (`doc_sync_test.go:663-700`) rather than scanning whole documents, so prose naming a directory in passing cannot satisfy it — and it `t.Fatal`s on an empty extent, so it cannot certify nothing.
- **The empty-payload predicate now sits at every layer that decides it** (`fetch.go:72-82`, `fetch.go:216-228`, `yaml.go:911-922`, `yaml.go:869-880`), and `httpAudioSource` keeps walking the remaining candidates instead of stopping — the finding that produced this named the class and the class is what shipped.

### 2. Critical findings

None.

### 3. Important findings

**I-1. `play_loop.go:229` — the subject argument is production policy that no guard reaches.**
Mutation-verified: replacing `q.Word()` with `""` at `cmd/define/play_loop.go:229` leaves the entire `cmd/define` suite green (full run, not `-run` filtered). **This is the 3rd finding in family `production-wiring-unpinned`** (BR-3: deleting the disk cache's wiring left the suite green; BR-21: nilling the vocabulary at all three sites left the suite green). Do not pin this instance. The rule the family keeps producing: *a behaviour proved at the door is unproven until the site obliged to obey it is derived* — and the window's own `lessons.md` states the corollary ("a parameter with one correct value is a parameter that will eventually be given another"), then the head commit added a seventh parameter whose correct value differs per site and is checked by nothing. Aggravating: `subject` and `already` are adjacent bare `string`s behind five other arguments, so swapping them at the reveal site compiles and silently re-colours the embedded render. Fix sketch, in the direction the guard already goes: give the prompt site one derivation — `writePrompt(stdout, q, d, opt)` deriving text, `promptRegions(q)`, `surfaceOf(q.Form())` and `q.Word()` from the single `q` — so the four cannot disagree, and let `TestEveryWriteWordsCallSitePassesAVocabulary`'s extent shrink to the non-prompt sites. Alternatively give `isTerminal(stdout)` (`play_loop.go:60`) the seam `stdinIsTerminal` already has, which is what currently makes an in-process sitting undrivable and forces every prompt guard to be a door guard.

**I-2. `cmd/define/store/audio.go:63` — `ok()` says "usable as a filename" and lets `Digest` out of the store directory.**
Probe-verified: `SetAudio(AudioKey{Word: "keel", Digest: "../../../../pwned"}, …)` wrote `pwned.mp3` and `pwned.yaml` four levels above the store root and returned `nil`. `Word` is laundered through `Slug` (which is what `safeElement` exists for); `Digest` reaches `filepath.Join` verbatim via `stem()`. Not reachable from production — `NewAudioKey` yields hex — but `AudioKey` is newly **exported** with exported fields and `SetAudio` is on the exported `store.Store` interface, so this is the new API's surface, not an internal detail. **This is the 2nd finding in family `persisted-field-unvalidated`** (BR-28 is the read half: `rec.From` returned as provenance with no membership check). Do not patch `Digest` alone. The class: *every field of an exported store type is validated by the predicate its USE requires — path elements against `safeElement`, provenance against the candidate list — never by non-emptiness and never by "this program wrote it"*. The enumeration is three fields: `AudioKey.Word` (validated, via `Slug`), `AudioKey.Digest` (not), `AudioRecord.From` (not, BR-28). One `ok()` covering the first two and one membership check at `audiodisk.go:88` covering the third closes the family.

**I-3. Docs gate — README's colour rule now has an exception it does not state, and the atlas never got M2's write door.**
`cmd/define/README.md:314-328` says words you have looked up "show in green — in the line you type, in definitions, and in answers", and states the one exception as "green stops where the text IS your deck". A meaning question's headword is prose on a colour-admitting surface and is now deliberately plain, which the README says should be green. The atlas is thinner: `atlas/define.md`'s "Highlighting the words you are learning" (:917-945) still describes two layers (`highlightSpans`, `wordRuns`) and never mentions `deckSpans` as the locator, `writeWords` as the one write door, or the `surface` classification (`surfaceProse`/`surfaceDeck`/`surfaceBoard`, `admitsColour`) — the whole architectural surface M2 introduced, present only as one bullet in the region registry at :2603. **This is the 5th finding in family `runtime-artifact-undocumented`** (BR-8, BR-16, BR-27, BR-30). Do not patch the two passages. The rule the family has now paid for five times: *the commit that changes a behaviour sweeps every present-tense statement of that behaviour, by a grep written before the change* — and the derived consumer was right every time while every hand-maintained one was wrong. Enumeration for this instance: README's "The words you already know" section, the atlas's highlighting section (three layers, the door, the surface enum), and the atlas's playback section which describes the seam but not the write path.

### 4. Minor findings

- **`doc_sync_test.go:625-629` vs `:684-687`** — the `{README.md, "events/2026-08-21.yaml"}, {../../atlas/define.md, "events/YYYY-MM-DD.yaml"}` table is now hand-copied into two guards; a third layout document reaches one and not the other. **3rd finding in family `two-statements-one-fact`** (BR-24 is the 4MB ceiling). One package-level `layoutDocs` var both range over is the whole fix, and it is the same shape as exporting one of the two audio ceilings.
- `writeWords`' two adjacent `string` parameters (`subject, already`) are swappable at every call site with no compile error — a named two-field struct or a `subjectOf`/`renderedAs` pair of typed strings removes the class. Folded into I-1's sketch.

### 5. Test coverage notes

- Both claimed fixes are pinned by tests that go red without them — verified by reverting each in a scratch worktree, not by reading the commit message. `TestStoreLayoutDocsNameEveryRuntimeDir` fires on both documents independently.
- The gap is the same one the whole issue keeps re-finding: door-level guards with no site-level derivation (I-1). `TestARevealStillMarksTheWordItRevealed` has the same shape — it proves the door's behaviour for `subject == ""`, not that the reveal site passes `""`.
- BR-32's dead clause is still there and still dead: `Slug("")` returns `w-e3b0c4`, so `ok()`'s `Slug(k.Word) != ""` is a tautology and `deckwords_test.go:409`'s `arg.Name == "nil"` branch is unreachable because `nil` is not assignable to `deps`.

### 6. Architectural notes

- **ARCH-DRY** — flag, minor: the two layout-doc guards duplicate their extent table (§4). Elsewhere the window is disciplined — `perWordDirs` derives from `RuntimeDirs`, `.gitignore` and the layout blocks now both derive.
- **ARCH-PURE** — pass. `deckSpans`/`highlightSpans`/`exceptWord`/`surfaceOf` are pure and tested with no IO; `writeWords` is the thin shell. `withoutWord` returning `v` unchanged for a nil vocabulary or empty subject keeps the one representation of "nothing to highlight".
- **ARCH-PURPOSE** — flag: I-3. The issue's purpose includes the discovery rule being legible to a learner; the rule now has a third case that only the code states. I-1 is the same axis on the test side — the finding named the colour pass, the class is "the site that must obey the rule".
- **ARCH-MOCK** — pass. `fakeCDN` is stateful and wire-level, the store has a `mem`/`YAML` twin pair driven by one `storetest` suite, `fetch_conformance_test.go` carries the live check, and production and tests share `newAudioSeam(newDiskAudioCache(st, http))`. The `var _ wordFiler = (*diskAudioCache)(nil)` line is the right response to the silent-capability failure.
- **ARCH-CONSTRAINTS** — pass. Both read paths that matter are bounded or justified (blob capped at 4MB; memo → disk → network ordering documented and asserted by request counts); no fan-out, no blocking optional work on the keystroke path. The record YAML read remains unbounded (BR-18, still open).
- **ARCH-SECURE** — flag: I-2, plus the still-open BR-18 and BR-28. The window hardened the payload and left both fields beside it — the blob is capped and empty-checked, while `From` (read back) and `Digest` (written through) are trusted.
- **ARCH-ORDER** — pass. `audioSeam` holds one mutex over two maps, releases it across the inner fetch (a duplicate concurrent fetch, not a corrupt state), spawns nothing that outlives its call, and `SetAudio` writes the blob before the record so the crash window costs a refetch rather than silence.

### 7. Plan revision recommendations

The appended `### 2026-09-07 — close rounds 3-7` entry is the right mechanism and covers the `writeWords` signature, the three-layer empty-payload predicate and BR-30. Two items from BR-31's enumeration are still missing and one is a live contradiction:

- **`## Revisions` — the `audioDir` shelf.** `plan:255` still calls it "the per-language audio directory" and `plan:446-447`'s ticked Step 2 still says `{path: y.audioDir(), scoped: true}` … "language-scoped **the way its siblings are**: `audioDir()` scopes on the STORE's language, exactly like `factsDir` and `itemsDir`". The code is `{path: y.audioDir(), scoped: false, many: true}` (`yaml.go:727`) with `audioDir` documented as "FLAT — not per language" (`yaml.go:180`). Append the reversal and the reason it was made.
- **`## Revisions` — the two AST guards' extent.** The entry mentions the clause `TestEveryWriteWordsCallSitePassesAVocabulary` grew; it should also record that the guard checks argument 3 only, so the surface and subject arguments are outside it (I-1).
- **BR-13's remainder.** `plan:217` and `plan:421` still cite a `store/audio_test.go` that does not exist (the rows live in `storetest/suite.go`), and `plan:203/449/456/827` still spell the types `audioKey`/`audioRecord` where the code exports `AudioKey`/`AudioRecord`.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      plan:217 and plan:421 still cite store/audio_test.go (no such file; the rows are in storetest/suite.go), and :203/:449/:456/:827 still spell the types lowercase.
  - id: BR-17
    disposition: not-addressed
    note: |
      command.go:397-399 unchanged; d.audio appears in neither applyLang's member list nor its "Deliberately NOT here" clause.
  - id: BR-18
    disposition: not-addressed
    note: |
      yaml.go:858 still reads the record with an unbounded os.ReadFile beside the capped blob at :869; readCapped's doc still says "refusing" where io.LimitReader truncates; no single statement at the store boundary says which persisted reads are bounded.
  - id: BR-23
    disposition: not-addressed
    note: |
      play_loop_test.go:4270-4291 still loops promptRegions(q) only; the issue row at :274-277 and plan:623/:731 still cite it for RegionWord.
  - id: BR-24
    disposition: not-addressed
    note: |
      fetch.go:35 and yaml.go:892 are still two independent 4 << 20 constants with nothing holding them equal.
  - id: BR-26
    disposition: not-addressed
    note: |
      The issue file has not changed since cd89212; both clauses of the row at :261-267 stand, as do Task 5 Step 4 and Verification row 6.
  - id: BR-27
    disposition: not-addressed
    note: |
      noAudioSource is still declared at main_test.go:20 with zero instantiations; play_loop_test.go:64/:1444 and fetch.go:134 still describe it in the present tense; lessons.md has the filing-reversal grep but no "one grep per reversal" rule.
  - id: BR-28
    disposition: not-addressed
    note: |
      audiodisk.go:80 still returns rec.From with no membership check against urls. Now the read half of the class named in this round's I-2.
  - id: BR-29
    disposition: addressed
    note: |
      Mutation-verified: reverting withoutWord at main.go:943 reddens TestAQuestionDoesNotColourTheWordItAsks by name; the reveal exemption is pinned separately. The production wiring is a new finding, not this one.
  - id: BR-30
    disposition: addressed
    note: |
      Mutation-verified on both halves: renaming usage/ in README or audio/ in the atlas each reddens TestStoreLayoutDocsNameEveryRuntimeDir, and README's missing usage/ row was added.
  - id: BR-31
    disposition: not-addressed
    note: |
      The appended Revisions entry covers the writeWords signature, the three-layer predicate and BR-30, but not the audioDir shelf: plan:255 and the ticked Step 2 at plan:446-447 still say audio/ is language-scoped "exactly like factsDir and itemsDir", contradicted by yaml.go:180 and yaml.go:727.
  - id: BR-32
    disposition: not-addressed
    note: |
      store/audio.go is untouched this window; Slug("") = "w-e3b0c4" so ok()'s Word clause is still a tautology, and deckwords_test.go:409's arg.Name == "nil" branch is still unreachable.
  - id: BR-33
    disposition: addressed
    note: |
      newHTTPAudioSource2 is deleted and the call inlined to cdn.source() at fetch_test.go:282.
findings:
  - id: new
    severity: Important
    family: production-wiring-unpinned
    title: |
      Dropping the subject at the sitting's prompt site leaves the whole suite green, so the BR-29 fix is pinned at the door and nowhere else
    detail: |
      Mutation-verified at HEAD: replacing q.Word() with "" at cmd/define/play_loop.go:229 passes the
      full cmd/define suite. This is the 3rd finding in family production-wiring-unpinned (BR-3, BR-21).
      Do NOT pin this instance. The rule: a behaviour proved at the door is unproven until the SITE
      obliged to obey it is derived — the window's own lessons.md states the corollary ("a parameter
      with one correct value is a parameter that will eventually be given another") and the head commit
      then added a seventh parameter whose value differs per site and is checked by nothing. Aggravating:
      subject and already are adjacent bare strings behind five arguments, so swapping them at the reveal
      site compiles and silently re-colours the embedded render. Sketch: one writePrompt(stdout, q, d, opt)
      deriving text, promptRegions(q), surfaceOf(q.Form()) and q.Word() from the single q, so the four
      cannot disagree; or give isTerminal(stdout) at play_loop.go:60 the seam stdinIsTerminal already has,
      which is what makes an in-process sitting undrivable and forces every prompt guard to be a door guard.
  - id: new
    severity: Important
    family: persisted-field-unvalidated
    title: |
      AudioKey.Digest reaches a filesystem path element unvalidated, and ok() — whose doc says "usable as a filename" — only checks that it is non-empty
    detail: |
      Probe-verified: SetAudio(AudioKey{Word: "keel", Digest: "../../../../pwned"}, []byte("ID3"), rec)
      wrote pwned.mp3 and pwned.yaml four levels above the store root and returned nil. Word is laundered
      through Slug (which is what safeElement exists for); Digest reaches filepath.Join verbatim via stem()
      (store/audio.go:60, yaml.go:925-943). Not reachable from production — NewAudioKey yields hex — but
      AudioKey is newly EXPORTED with exported fields and SetAudio is on the exported store.Store interface.
      This is the 2nd finding in family persisted-field-unvalidated (BR-28 is the read half, rec.From).
      Do NOT patch Digest alone. The class: every field of an exported store type is validated by the
      predicate its USE requires — path elements against safeElement, provenance against the candidate
      list — never by non-emptiness and never by "this program wrote it". Enumeration is three fields:
      AudioKey.Word (validated), AudioKey.Digest (not), AudioRecord.From (not). One ok() covering the
      first two plus one membership check at audiodisk.go:88 closes the family.
  - id: new
    severity: Important
    family: runtime-artifact-undocumented
    title: |
      README states the colour rule with an exception it does not mention, and the atlas never got M2's write door, surface classification or locator layer
    detail: |
      README.md:314-328 says looked-up words "show in green — in the line you type, in definitions, and
      in answers" with one stated exception ("green stops where the text IS your deck"); a meaning
      question's headword is prose on a colour-admitting surface and is now deliberately plain.
      atlas/define.md's "Highlighting the words you are learning" (:917-945) still describes two layers
      and never names deckSpans as the locator, writeWords as the one write door, or the surface enum
      (surfaceProse/surfaceDeck/surfaceBoard, admitsColour) — M2's whole architectural surface, present
      only as one region-registry bullet at :2603. This is the 5th finding in family
      runtime-artifact-undocumented (BR-8, BR-16, BR-27, BR-30). Do NOT patch the two passages. The rule,
      now paid for five times: the commit that changes a behaviour sweeps every present-tense statement of
      it, by a grep written before the change — the derived consumer was right every time and every
      hand-maintained one was wrong. Enumeration: README's "The words you already know" section, the
      atlas's highlighting section (three layers, the door, the surface enum), the atlas's playback section.
  - id: new
    severity: Minor
    family: two-statements-one-fact
    title: |
      The layout-document extent table is now hand-copied into two guards, so a third document reaches one and not the other
    detail: |
      doc_sync_test.go:625-629 and :684-687 both spell {README.md, "events/2026-08-21.yaml"} and
      {../../atlas/define.md, "events/YYYY-MM-DD.yaml"}. 3rd finding in family two-statements-one-fact
      (BR-24 is the 4MB ceiling); measured prevalence 2 sites. One package-level layoutDocs var both
      range over is the whole fix, and it is the same move as exporting one of the two audio ceilings.
```
