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
