---
gate: boundary-review
issue: 46
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-07T15:13:07-07:00"
      agent: sdlc
      findings:
        - id: BR-1
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
            (carried from plan-quality PQ-11, deferred to the boundary review)
          family: seam-wrap-site
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-07T15:13:07-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Critical
          title: A /lang switch strands the audio cache on the previous language's shelf, and --forget reports success while the file survives
          detail: |-
            withStore (cmd/define/main.go:187) installs diskAudioCache over d.deck; applyLang
            (cmd/define/command.go:416) replaces the whole d.langDeps including deck but never
            rebuilds d.audio. Verified by driving run() with "/lang es\nsycophantic\n": the word
            lands in words/es/sycophantic.yaml while the record lands in
            audio/en/sycophantic--35084a048a8eddfe.yaml, and a following -forget sycophantic
            prints "removed sycophantic" with that file still on disk. Two Done-when rows fail on
            this path, and applyLang's own doc claims it owns the re-derivation enumeration.
            ARCH-ORDER: this is the external event the plan's enumeration did not name.
          family: lang-switch-derivation
          round: 2
        - id: BR-3
          severity: Critical
          title: Deleting the entire production wiring of diskAudioCache from withStore leaves the suite green
          detail: |-
            Replacing the block at cmd/define/main.go:187-190 with a no-op produces a failure set
            byte-identical to the unmutated baseline (18 git-dependent guards in both runs, nothing
            else). Every disk-cache test hand-wires newAudioSeam(newDiskAudioCache(...)), so
            audiodisk.go's own claim that "the line the tests exercise is the line production runs"
            is untrue of the line that installs the layer — the same shape as the wordFiler
            signature bug the issue Log already records. ARCH-MOCK: the fake is good, but no test
            runs the stack through the production boundary.
          family: production-wiring-unpinned
          round: 2
        - id: BR-4
          severity: Critical
          title: The suite is red at the review head — TestPlanTableStatusMatchesTheChangeWindow fails
          detail: |-
            go test ./cmd/define/ at 46c77c6 with a clean worktree: the plan calls RenderOpts
            "modified" but the window does not touch its declaration (cmd/define/render.go:12-41).
            The plan's own bullet says "no field changes; Vocab STAYS", so the table row contradicts
            the prose beside it. Change the status to "unchanged" or drop the row.
          family: red-at-boundary
          round: 2
        - id: BR-5
          severity: Important
          title: perWordDir's new `many` axis is declared but no guard checks it, and the export helper still says "both axes"
          detail: |-
            PerWordDirsForTest (cmd/define/store/export_test.go:11) exposes only Path and Scoped, so
            TestPerWordDirsCoverEveryRuntimeDir cannot see `many`. A future directory where a word
            owns several files can be declared many:false and Forget will silently leave them — the
            exact drift the second axis was added to stop, and what yaml.go:731's own comment claims
            the guard prevents.
          family: unguarded-classification-axis
          round: 2
        - id: BR-6
          severity: Important
          title: The degrade-never-fail rows are half-delivered and failingStore.Audio claims coverage with zero call sites
          detail: |-
            Plan Task 3 Step 5 commits to pinning an unwritable directory, a corrupt file and a
            store that could not open. Only "no store" and "no word" are pinned. The new
            failingStore.Audio/SetAudio (cmd/define/history_store_test.go:100-106) carry a comment
            saying they drive that property, but failingStore appears in no audio test, and
            YAML.Audio's corrupt-record and missing-blob branches are reached by nothing.
          family: claimed-coverage-absent
          round: 2
        - id: BR-7
          severity: Important
          title: TestASittingFetchesARecordingOnce never calls runPlay, so the M1 regression is not pinned where it lived
          detail: |-
            Plan Task 1 Steps 5-6 specify a behavioural pin at the loop through the CDN recorder —
            "fakeCDN + runPlay, two questions on one word". The delivered test
            (cmd/define/play_loop_test.go:4328) calls seam.Fetch three times, which
            TestAudioSeamServesRepeatsFromMemory already asserts. The trailing field-type assertion
            is a real guard, but it is not the loop-level behaviour the plan named.
          family: pin-not-at-the-loop
          round: 2
        - id: BR-8
          severity: Important
          title: README's and the atlas's working-directory listings do not mention audio/, the new (and first binary) runtime artifact
          detail: |-
            .gitignore derived the new directory from RuntimeDirs, but the two hand-maintained
            restatements did not: cmd/define/README.md:501-520 and atlas/define.md:605-616 both
            enumerate every other runtime artifact and omit audio/<lang>/. The atlas gained prose
            about the cache elsewhere; the enumeration a reader greps did not move. ARCH-PURPOSE
            shadow-sweep.
          family: runtime-artifact-undocumented
          round: 2
        - id: BR-9
          severity: Important
          title: The plan's historical Revisions prose was overwritten by a blind symbol substitution and no longer reads as English
          detail: |-
            Satisfying the retired-symbol guard rewrote completed revision entries in place, against
            AGENTS.md's append-don't-overwrite rule, leaving text such as "the decorator it
            replaced's doc comment records", "the taxonomy the decorator it replaced already
            documents", "every non-test the wrap that used to be remembered call must be a subset",
            and "the wrap that used to be remembered becomes idempotent". The record of what rounds
            2-4 decided is now unreadable.
          family: revision-overwritten
          round: 2
        - id: BR-10
          severity: Minor
          title: audioNameSep's claim that a slug cannot contain "--" is false, so --forget re deletes re-'s recordings
          detail: |-
            Probe: Slug("re-") == "re--ddf427", Slug("anti-") == "anti--0aa4e1",
            Slug("well- known") == "well--known-885c00". removeByPrefix globs "<slug>--", so
            forgetting "re" reaches "re-"'s files. Cost is one wasted refetch, but the invariant
            written into the comment is not the one Slug provides.
          family: prefix-separator-invariant
          round: 2
        - id: BR-11
          severity: Minor
          title: The cached blob is read with an unbounded os.ReadFile where the network path caps at maxAudioBytes
          detail: |-
            cmd/define/store/yaml.go:833 loads the whole file into memory. ARCH-SECURE: a persisted
            artifact that an older version, another process, or a hand edit may have written is not
            trustworthy just because this program produced it.
          family: unbounded-input-read
          round: 2
        - id: BR-12
          severity: Minor
          title: SetAudio with Missing set skips the blob write but does not remove an existing .mp3
          detail: |-
            A key that was a hit and later becomes a verdict leaves an orphan blob until Forget.
            Not served — the record gates it — but it is debris in a directory with no eviction.
          family: stale-blob-on-verdict
          round: 2
        - id: BR-13
          severity: Minor
          title: 'The plan''s entity tables drift from the code: audioKey/audioRecord vs AudioKey/AudioRecord, no store/audio_test.go, mergeRegions has no row'
          detail: |-
            The types are exported because they cross the package boundary; the named colocated
            test file was never created and its rows landed in storetest/suite.go instead, which is
            arguably the better home but is not what the plan says.
          family: plan-table-drift
          round: 2
        - id: BR-14
          severity: Minor
          title: TestClozeOptionsAreClickableButNotColoured asserts only colour and never a click target
          detail: |-
            It checks admitsColour() on three form names. The clickability half is in
            TestADeckSurfaceIsClickableButUncoloured, so the name promises coverage the body does
            not contain.
          family: test-name-overclaims
          round: 2
        - id: BR-15
          severity: Minor
          title: The plan file has zero ticked steps while the issue's Plan ticks M1
          detail: |-
            Both chunks of workshop/plans/000046-audio-cache-and-deck-words-plan.md remain
            unchecked after the work landed, so the plan's own checklist cannot be used to see what
            this boundary delivered.
          family: tracking-in-two-places
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-09-07T16:22:33-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'The third shape shipped: deps.audio is *audioSeam, both wrap lines are gone, and the retired-symbol guard now forbids the old constructor.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: audio/ is flat; re-scoping it by language reddens TestForgetTakesARecordingFetchedInAnotherLanguage and the scoped guard (mutation-verified).
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'Mutation-verified: deleting the withStore block reddens TestWithStorePutsTheDiskCacheUnderTheMemo with the right message.'
          round: 3
        - id: BR-4
          disposition: addressed
          note: Full suite green at HEAD under default and conformance tags; gofmt -l and go vet ./... clean.
          round: 3
        - id: BR-5
          disposition: addressed
          note: 'Mutation-verified: many true to false reddens three storetest rows on both twins plus the language-switch test.'
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: failingStore now has a call site, but the corrupt-file and unwritable-directory rows and YAML.Audio's two defensive branches remain unpinned.
          round: 3
        - id: BR-7
          disposition: addressed
          note: Renamed to what it asserts and the argument recorded; note that playSession IS drivable in-process, so the "a sitting cannot be driven" claim is broader than true.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Both enumerations now list audio/ — but with the superseded filing scheme; raised separately below.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: No Revisions entry was appended for this round and the body was edited in place again, introducing a fresh break at plan lines 365-366.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Directory-per-word removes the separator claim entirely; the re/re- conformance row plants a fixture that reaches the branch on both twins.
          round: 3
        - id: BR-11
          disposition: addressed
          note: Blob read now capped by readCapped; the sibling record read and the doc/behaviour mismatch are raised below in the same family.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: 'Mutation-verified: reverting the os.Remove(blob) leaves the store package fully green — the test asserts only through Audio, which gates on the record.'
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: audioKey/audioRecord, store/audio_test.go and the missing mergeRegions row all persist, and new drift arrived with the reversal.
          round: 3
        - id: BR-14
          disposition: addressed
          note: The test now drives writeWords and asserts two RegionWord click targets as well as the absence of colour.
          round: 3
        - id: BR-15
          disposition: not-addressed
          note: Still 0 of 45 plan checkboxes ticked while the issue's Plan ticks M1 and M2.
          round: 3
      findings:
        - id: BR-16
          severity: Important
          title: The audio filing reversal was swept through the code paths and left ten restatements of the superseded scheme, two of them doc comments on the field it changed
          detail: |-
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
          family: runtime-artifact-undocumented
          round: 3
        - id: BR-17
          severity: Minor
          title: applyLang's enumeration of what a language switch re-derives still does not account for d.audio, which holds a store bound to the pre-switch language
          detail: |-
            This is the 2nd finding in family `lang-switch-derivation`. The behaviour is correct now
            only because audioDir is flat, so d.audio's stale store reference is harmless — but
            applyLang (cmd/define/command.go:397-399) explicitly lists its deliberate exclusions
            ("Deliberately NOT here: d.history") and d.audio is in neither list. The rule, not the
            line: every deps member holding a language-derived object appears in applyLang's
            enumeration or in its exclusion clause, with the reason — here, that audio/ carries no
            language shelf.
          family: lang-switch-derivation
          round: 3
        - id: BR-18
          severity: Minor
          title: Only the blob was bounded — the record YAML beside it is still read with an unbounded os.ReadFile, readCapped truncates where its doc says it refuses, and no test pins the cap
          detail: |-
            This is the 2nd finding in family `unbounded-input-read`, and the sibling read is in the
            same function as the one the first finding named: yaml.go:838 reads the record with
            os.ReadFile while yaml.go:843 caps the blob. readCapped's doc says "refusing anything past
            max" but io.LimitReader truncates, which matches the network path's behaviour and not the
            comment. Do not just cap the record. The rule: state once, at the store boundary, which
            persisted reads are bounded and why the rest (words/, facts/, items/) are not — a per-site
            fix is what left the sibling in place here.
          family: unbounded-input-read
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-09-07T16:42:38-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 5
      timestamp: "2026-09-07T17:01:41-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: All four degrade branches mutation-verified red-without-the-fix; failingStore now has a real call site at audiodisk_test.go:152.
          round: 5
        - id: BR-9
          disposition: addressed
          note: A Revisions entry was appended for this round; the substitution artifacts and the "go, along / along with" break are gone (grep clean).
          round: 5
        - id: BR-12
          disposition: addressed
          note: 'Mutation-verified: reverting os.Remove(blob) reddens TestAVerdictDeletesTheRecordingItSupersedes by name.'
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: mergeRegions, AudioKey/AudioRecord and Task 2's Create list are fixed; plan lines 217, 232, 255 and 346 still drift from the code.
          round: 5
        - id: BR-15
          disposition: addressed
          note: 45 of 45 plan checkboxes ticked, none left unchecked.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: Code paths, README, atlas and the lessons rule all landed; the prescribed grep is line-oriented and misses three wrapped restatements.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: cmd/define/command.go is not in this window at all; d.audio is still in neither applyLang's enumeration nor its exclusion clause.
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: The cap is now pinned (mutation-verified), but the sibling record read is still unbounded and the boundary-level statement was not written.
          round: 5
      findings:
        - id: BR-19
          severity: Important
          title: An empty recording is written to disk and served as a permanent, never-expiring hit — the outcome the new missing-blob branch exists to prevent
          detail: |-
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
          family: degenerate-payload-trusted
          round: 5
        - id: BR-20
          severity: Minor
          title: Two comments justify the 4MB cap and the .mp3 extension by citing a README passage that does not exist
          detail: |-
            readCapped says "The directory is documented as inspectable and hand-editable"
            (cmd/define/store/yaml.go:826-827) and audioBlobExt says "for a human browsing the
            directory, which the README documents as an invited workflow" (yaml.go:886-887).
            cmd/define/README.md describes what define writes and never invites inspection or
            editing; grepping it for hand-edit/editable/inspect/browse returns nothing. The claim is
            load-bearing — it is the entire ARCH-SECURE argument for bounding the read. The rule: a
            comment that justifies a decision by citing another document must be checkable against
            it. Either state it once in the README and cite that, or drop the citation and keep the
            cap on its own merits.
          family: unsourced-cross-reference
          round: 5
      boundary: M1
      blocked: false
    - "n": 6
      timestamp: "2026-09-07T17:31:05-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: not-addressed
          note: Table names and the mergeRegions row are fixed; the plan still cites `store/audio_test.go`, which does not exist, and TestPlanCitesTestsThatExist checks only Test* names.
          round: 6
        - id: BR-16
          disposition: addressed
          note: yaml.go/README/atlas swept, grep + rule recorded in lessons.md; the plan body is superseded by the appended Revisions entry per AGENTS.md §1.
          round: 6
        - id: BR-17
          disposition: not-addressed
          note: command.go is not in the change window; applyLang's exclusion clause still names only d.history, and d.audio appears in neither list.
          round: 6
        - id: BR-18
          disposition: not-addressed
          note: The cap is now pinned by a test, but the sibling record read at yaml.go:859 is still an unbounded os.ReadFile and no single statement at the store boundary says which persisted reads are bounded.
          round: 6
        - id: BR-19
          disposition: addressed
          note: 'Mutation-verified: dropping the read guard reddens the truncated-blob subtest, dropping the write guard reddens the filesystem subtest, dropping both reddens TestAnEmptyResponseIsNotCachedAsAHit.'
          round: 6
        - id: BR-20
          disposition: addressed
          note: README.md:535 now carries "Look at any of it, and edit it if you like" plus the untrusted-input paragraph, and both comments cite that passage.
          round: 6
      findings:
        - id: BR-21
          severity: Important
          title: Passing nil for the vocabulary at all three writeWords call sites leaves the whole suite green, so M2's production wiring is unpinned
          detail: 'Probed in a scratch checkout of cd89212: replacing deckVocabulary(d) with nil at play_loop.go:229, play_loop.go:478 and main.go:877 keeps go test ./cmd/define/ green (only the git-dependent repo guards fail, identically at unmutated baseline). Every writeWords test passes a literal deckOf(...), so nothing observes the capability arriving from deps — M2 could be silently dead exactly as forWord''s interface mismatch was. This is the 2nd finding in family production-wiring-unpinned (BR-3 was the same rule at M1). Do NOT just fixture the three sites: state the rule — a capability handed from deps into a write door is pinned by a test that observes it arriving from deps, not from a literal — and write the enumeration (the writeWords call sites, deckVocabulary''s callers) so a fourth site is covered by construction. TestWithStorePutsTheDiskCacheUnderTheMemo is the shape that worked. Related: deckwords_test.go:264-266 claims in prose that it "fails if a call site stops passing the vocabulary", which it cannot.'
          family: production-wiring-unpinned
          round: 6
        - id: BR-22
          severity: Important
          title: The memo serves a zero-byte 200 as a hit for the whole sitting, with a from URL reportVoice prints as the voice that answered
          detail: 'audioSeam.fetch (cmd/define/fetch.go:206) stores c.hits[key] = cachedAudio{data, from} with no emptiness predicate. Probed with one seam and two FetchFor calls against a CDN serving an empty body: one request, both calls return bytes=0 from="http://.../a.mp3" err=nil. speak then writes a 0-byte mp3 for afplay and reportVoice prints a record naming a URL that carried no recording. This is the 2nd finding in family degenerate-payload-trusted. Round 4 satisfied BR-19''s literal ask (both store twins plus the write) but not the class it named — the payload cannot be a recording. Do not patch the memo alone: enumerate every layer that can produce, remember, persist or play a fetch result (httpAudioSource.Fetch, audioSeam.hits, diskAudioCache.put, both SetAudio implementations, speak) and put the predicate where the payload is first judged, so a 0-byte 200 is not an answer and every layer below is correct by construction. The store.Store interface doc (store.go:88-91) is part of that sweep: it documents the verdict form of SetAudio and is silent that a non-Missing call with empty data is refused.'
          family: degenerate-payload-trusted
          round: 6
        - id: BR-23
          severity: Minor
          title: The Done-when row names TestAPromptRegionCoversTheTextItClaims as exercising RegionWord, but that test can only see promptRegions
          detail: 'workshop/issues/000046-...md:275-277 claims numRegionKinds, TestEveryRegionKindIsActionable, TestAtlasDescribesEveryRegionKind and TestAPromptRegionCoversTheTextItClaims all exercise the new kind unedited. The last one loops over promptRegions(q) (play_loop_test.go:4276), which only ever yields RegionHeadword. The invariant is genuinely pinned — by TestWordRegionsCoverTheTextTheyClaim — so this is a wrong citation rather than a gap. 2nd finding in family claimed-coverage-absent: the rule is that a Done-when row names the guard whose assertion would go red, verified by asking which assertion that is, not by which test name sounds adjacent.'
          family: claimed-coverage-absent
          round: 6
        - id: BR-24
          severity: Minor
          title: The 4MB audio ceiling is stated twice, in two packages, with nothing holding them equal
          detail: maxAudioBytes (cmd/define/fetch.go:35) and maxAudioBlobBytes (cmd/define/store/yaml.go:892) are the same number written twice; the comment justifies it ("the store cannot import main") but nothing detects drift. Exporting one and reading it from the other costs a line.
          family: two-statements-one-fact
          round: 6
        - id: BR-25
          severity: Minor
          title: deckSpan.Word is documented as the deck key but holds the display text
          detail: cmd/define/deckwords.go:26 says "Word is the deck key, which is what a click plays"; deckSpans assigns sp.text, so a sentence-initial or differently-spaced match carries the on-screen spelling. Harmless today because AudioCandidates lower-cases and store.Key normalises before filing, but the doc is a claim a caller would rely on.
          family: doc-claims-unbacked
          round: 6
      blocked: true
    - "n": 7
      timestamp: "2026-09-07T17:50:46-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: not-addressed
          note: Entity-table names and the mergeRegions row are right, but the plan still cites `store/audio_test.go` at lines 217 and 421 and no such file exists.
          round: 7
        - id: BR-17
          disposition: not-addressed
          note: cmd/define/command.go is untouched by the window; applyLang's exclusion clause (command.go:397) still names only d.history and d.audio is in neither list.
          round: 7
        - id: BR-18
          disposition: not-addressed
          note: yaml.go:859 still reads the record with an unbounded os.ReadFile beside the capped blob at :871; readCapped truncates where its doc says it refuses, and yaml_test.go's own subtest is named "is truncated"; no single statement at the store boundary says which persisted reads are bounded.
          round: 7
        - id: BR-21
          disposition: not-addressed
          note: 'Mutation-verified at HEAD: `nil` at a call site now reddens the new guard, but swapping all three sites to `vocabularyFor(d, opt)` leaves the whole cmd/define suite green — the guard reads the argument''s source token, not the capability''s arrival from deps, and deckwords_test.go:268-269''s false prose is unchanged.'
          round: 7
        - id: BR-22
          disposition: not-addressed
          note: The memo layer is fixed and pinned, but httpAudioSource.Fetch (fetch.go:70) still returns a zero-byte 200 as success and stops the candidate walk there, and store.go:90-93 is still silent that a non-Missing SetAudio with empty data is refused — both named explicitly by the finding.
          round: 7
        - id: BR-23
          disposition: not-addressed
          note: The issue row at :275-277 is unchanged, and the same citation appears at plan:623 and plan:731 — where Task 5 Step 4 is ticked for an extension of TestAPromptRegionCoversTheTextItClaims that did not happen.
          round: 7
        - id: BR-24
          disposition: not-addressed
          note: maxAudioBytes (fetch.go:35) and maxAudioBlobBytes (yaml.go:892) are both unchanged; nothing holds them equal.
          round: 7
        - id: BR-25
          disposition: addressed
          note: deckwords.go:25-32 now says "the span's text AS WRITTEN, not the normalised deck key", which matches deckSpans assigning sp.text, and names why it is harmless.
          round: 7
      findings:
        - id: BR-26
          severity: Minor
          title: The "every deck word is clickable" Done-when row now carries two false statements, one of them created by this round's own fix
          detail: 'workshop/issues/000046-...md:279-285 says TestEveryDeckWordInASittingIsClickable "drives it through writeWords rather than through the rule, so it fails if a call site stops passing the vocabulary" — it passes a literal deckOf(...) and cannot — and then concedes "A new writeWords call site is covered by review, not by construction — carried to #30", which round 6''s TestEveryWriteWordsCallSitePassesAVocabulary made stale in the same commit that left the first clause standing. This is the 3rd finding in family claimed-coverage-absent (BR-6, BR-23 preceding it). Do NOT fix this row. State the rule and sweep the enumeration it implies: a row, a ticked plan step, or a test doc comment that names a guard must name the guard whose assertion would go red, verified by asking which assertion that is — and when a round ADDS a guard, the concessions that said it did not exist are part of the same commit''s sweep. The enumeration is the Done-when rows, the ticked plan steps (Task 5 Step 4 and Verification row 6 are both wrong for the same reason), and the doc comments on the tests this window added.'
          family: claimed-coverage-absent
          round: 7
        - id: BR-27
          severity: Minor
          title: The seam reversal never got the grep the filing reversal got, and left a dead double plus three present-tense claims about it
          detail: noAudioSource (cmd/define/main_test.go:20) is declared and instantiated nowhere in the package after deps.audio became *audioSeam. play_loop_test.go:64 states "playRig deliberately installs noAudioSource AND noAudio:true" when playRig (:41) does newAudioSeam(nil); play_loop_test.go:1444 calls it "the package's other double"; fetch.go:123 refers to "the tests that used to write noAudioSource{}" while the type itself was left behind. Separately, the plan's writeWords bullet says "NEW, beside writeRendered rather than replacing it. The plain door still serves callers with nothing to mark" — writeRendered has exactly one caller in the tree, writeWords itself. This is the 3rd finding in family runtime-artifact-undocumented (BR-8, BR-16). Do NOT patch these four sites. The rule is already in lessons.md for the FILING reversal — "write the grep before the fix, run it after" — and M1 contained a SECOND reversal (cachingAudioSource -> audioSeam) that never got one. Write the grep for the retired seam vocabulary, run it across cmd/, atlas/, README and the plan, and add "one grep per reversal, not one per issue" to the rule.
          family: runtime-artifact-undocumented
          round: 7
        - id: BR-28
          severity: Minor
          title: AudioRecord.From is read back from a hand-editable file and printed as the record of which voice answered, with no check that it is one of the URLs asked for
          detail: 'diskAudioCache.Fetch (cmd/define/audiodisk.go:88) returns rec.From straight from audio/<slug>/<digest>.yaml. speak passes it to reportVoice, whose spokeSource decides by membership in sourceCandidates() and whose own doc says the report "survives on a pipe and cannot be taken back — and a record has to be true". README.md:535 now explicitly invites editing that directory, and this same round added readCapped and the empty-payload predicate on exactly that reasoning — the payload was hardened and the provenance field beside it was not. A hand-edited From flips the fallback announcement in either direction: absent when it should fire, or fired when the source recording really did answer. ARCH-SECURE: an input crossing a process boundary is untrusted even when this program wrote it. Cheapest fix is one membership check where the record is read — if rec.From is not in urls, treat the entry as nothing cached rather than as evidence.'
          family: persisted-field-unvalidated
          round: 7
      blocked: true
    - "n": 8
      timestamp: "2026-09-07T18:14:42-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: not-addressed
          note: Table fixed (AudioKey/AudioRecord, mergeRegions row added); the Test-surface paragraph at :216-219 still names a store/audio_test.go that does not exist, and :203/:449/:456/:827 still spell the types lowercase.
          round: 8
        - id: BR-17
          disposition: not-addressed
          note: applyLang (command.go) still lists neither d.audio among its members nor among its "Deliberately NOT here" exclusions.
          round: 8
        - id: BR-18
          disposition: not-addressed
          note: yaml.go:838 still reads the record with an unbounded os.ReadFile beside the capped blob at :843, and readCapped's doc still says "refusing" where io.LimitReader truncates. The blob cap IS pinned (yaml_test.go, "a blob larger than the cap is truncated") — that half of the finding was inaccurate.
          round: 8
        - id: BR-21
          disposition: addressed
          note: 'Mutation-verified: replacing d with deps{} at play_loop.go:229 reddens TestEveryWriteWordsCallSitePassesAVocabulary by name; nil no longer compiles.'
          round: 8
        - id: BR-22
          disposition: addressed
          note: Mutation-verified separately at both layers; the store.Store interface doc now states the empty-payload refusal.
          round: 8
        - id: BR-23
          disposition: not-addressed
          note: TestAPromptRegionCoversTheTextItClaims still loops promptRegions(q) only (play_loop_test.go:4276); the Done-when row still cites it.
          round: 8
        - id: BR-24
          disposition: not-addressed
          note: maxAudioBytes and maxAudioBlobBytes still independent; news.go:22 makes it three statements of the same number.
          round: 8
        - id: BR-26
          disposition: not-addressed
          note: Both clauses of the Done-when row are unchanged, as are Task 5 Step 4 and Verification row 6.
          round: 8
        - id: BR-27
          disposition: not-addressed
          note: noAudioSource still declared at main_test.go:20 with zero instantiations; the three present-tense claims stand; no grep for the seam reversal and no "one grep per reversal" rule in lessons.md. A THIRD reversal is now visible in the same artifact — the plan's audioDir bullet still says "the per-language audio directory".
          round: 8
        - id: BR-28
          disposition: not-addressed
          note: audiodisk.go:88 still returns rec.From unchecked against urls, and README.md:538 now claims persisted input is treated as untrusted.
          round: 8
      findings:
        - id: BR-29
          severity: Important
          title: A meaning question now paints the word under test deck-green, which is the one word on screen where the discovery rule says it should not
          detail: surfaceOf("meaning") is surfaceProse and Choice.Prompt() is word + "\n\n" + options, so colourOutside runs over the headword line too. Probed at HEAD with a deck holding the headword the prompt writes "\n\x1b[1;32mephemeral\x1b[0m\n\n1  ...". New behaviour — the prompt was written through writeRendered with no colour before. TestChoiceOptionGlossesAreColoured cannot see it because its vocabulary omits the headword, and the operator's pty pass covered the cloze and reveal frames only. Either classify the headword line out of the colour pass or pin the decision with a deck containing the headword.
          family: discovery-colour-scope
          round: 8
        - id: BR-30
          severity: Important
          title: README's and the atlas's working-directory listings restate store.RuntimeDirs by hand with no guard, and README already omits usage/
          detail: 'This is the 4th finding in family runtime-artifact-undocumented (BR-8, BR-16, BR-27). Do NOT fix another instance. RuntimeDirs has four consumers; the two derived ones are guarded (TestGitignoreCoversRuntimeDirs, TestPerWordDirsCoverEveryRuntimeDir) and the two documentation ones are not. Measured prevalence: README.md contains no occurrence of "usage/" at all, so the listing is missing a runtime directory nobody noticed. The class fix is TestStoreLayoutDocsNameEveryRuntimeDir, mirroring TestStoreLayoutDocsNameEveryEventKind at doc_sync_test.go:622 and reusing its layoutBlockIn helper over both markers.'
          family: runtime-artifact-undocumented
          round: 8
        - id: BR-31
          severity: Important
          title: The durable plan has no Revisions entry for close rounds 3-7, and its writeWords bullet states a signature the head commit removed
          detail: 'This is the 2nd finding in family plan-table-drift. The plan''s last Revisions entry is "M1 boundary review, rounds 1 and 2"; five commits since then changed the design. plan:690 says writeWords "takes the vocabulary and the surface" — a1d339b removed that argument, in the commit that left the sentence standing. plan:257 still calls audioDir "the per-language audio directory", falsified by the flat-shelf reversal whose vocabulary BR-16''s grep could not match. State the rule: a plan artifact''s entity description is swept in the same commit that changes the entity''s shape, by an APPENDED Revisions entry per AGENTS.md section 1, and each reversal earns its own grep. Enumeration for this round: the writeWords signature, the audioDir shelf, the empty-payload predicate''s three layers, the two AST guards.'
          family: plan-table-drift
          round: 8
        - id: BR-32
          severity: Minor
          title: Two guard clauses added in this window cannot fire — AudioKey.ok's Word check is a tautology and the AST guard's nil branch does not compile
          detail: 'This is the 4th finding in family claimed-coverage-absent (BR-6, BR-23, BR-26). Earlier rounds fixed prose instances; this is the same rule in code. Do NOT fix these two sites — state the rule (every guard clause needs a witness input that makes it fire; write the witness or delete the clause) and sweep the clauses this window added. Instances: store/audio.go:63, where Slug never returns "" (Slug("") = "w-e3b0c4"), so ok() is effectively just Digest != "" — probed, an AudioKey with an empty Word files a readable hit under audio/w-e3b0c4/ that Forget("") cannot remove, reachable through the newly exported NewAudioKey/SetAudio surface though not from production. And deckwords_test.go:409, where arg.Name == "nil" is dead because nil is not assignable to the struct-typed deps parameter (verified: "cannot use nil as deps value"), so the package would not build and the test would never run.'
          family: claimed-coverage-absent
          round: 8
        - id: BR-33
          severity: Minor
          title: newHTTPAudioSource2 is a rename wrapper around cdn.source() whose name reads as a second constructor
          detail: fetch_test.go:295 defines newHTTPAudioSource2(c *fakeCDN) *httpAudioSource { return c.source() } beside the real newHTTPAudioSource. Inline the call.
          family: two-statements-one-fact
          round: 8
      blocked: true
---

# Gate ledger — tools#46 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T15:13:07-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `seam-wrap-site` "That is ~8 functions" is the 4th wrong prose statement of the wrap set — the real membership is 24, and Step 4 writes the line into functions that never touch audio
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
  (carried from plan-quality PQ-11, deferred to the boundary review)

## Round 2 — 2026-09-07T15:13:07-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `lang-switch-derivation` A /lang switch strands the audio cache on the previous language's shelf, and --forget reports success while the file survives
  withStore (cmd/define/main.go:187) installs diskAudioCache over d.deck; applyLang
  (cmd/define/command.go:416) replaces the whole d.langDeps including deck but never
  rebuilds d.audio. Verified by driving run() with "/lang es\nsycophantic\n": the word
  lands in words/es/sycophantic.yaml while the record lands in
  audio/en/sycophantic--35084a048a8eddfe.yaml, and a following -forget sycophantic
  prints "removed sycophantic" with that file still on disk. Two Done-when rows fail on
  this path, and applyLang's own doc claims it owns the re-derivation enumeration.
  ARCH-ORDER: this is the external event the plan's enumeration did not name.
- **BR-3** [Critical] `production-wiring-unpinned` Deleting the entire production wiring of diskAudioCache from withStore leaves the suite green
  Replacing the block at cmd/define/main.go:187-190 with a no-op produces a failure set
  byte-identical to the unmutated baseline (18 git-dependent guards in both runs, nothing
  else). Every disk-cache test hand-wires newAudioSeam(newDiskAudioCache(...)), so
  audiodisk.go's own claim that "the line the tests exercise is the line production runs"
  is untrue of the line that installs the layer — the same shape as the wordFiler
  signature bug the issue Log already records. ARCH-MOCK: the fake is good, but no test
  runs the stack through the production boundary.
- **BR-4** [Critical] `red-at-boundary` The suite is red at the review head — TestPlanTableStatusMatchesTheChangeWindow fails
  go test ./cmd/define/ at 46c77c6 with a clean worktree: the plan calls RenderOpts
  "modified" but the window does not touch its declaration (cmd/define/render.go:12-41).
  The plan's own bullet says "no field changes; Vocab STAYS", so the table row contradicts
  the prose beside it. Change the status to "unchanged" or drop the row.
- **BR-5** [Important] `unguarded-classification-axis` perWordDir's new `many` axis is declared but no guard checks it, and the export helper still says "both axes"
  PerWordDirsForTest (cmd/define/store/export_test.go:11) exposes only Path and Scoped, so
  TestPerWordDirsCoverEveryRuntimeDir cannot see `many`. A future directory where a word
  owns several files can be declared many:false and Forget will silently leave them — the
  exact drift the second axis was added to stop, and what yaml.go:731's own comment claims
  the guard prevents.
- **BR-6** [Important] `claimed-coverage-absent` The degrade-never-fail rows are half-delivered and failingStore.Audio claims coverage with zero call sites
  Plan Task 3 Step 5 commits to pinning an unwritable directory, a corrupt file and a
  store that could not open. Only "no store" and "no word" are pinned. The new
  failingStore.Audio/SetAudio (cmd/define/history_store_test.go:100-106) carry a comment
  saying they drive that property, but failingStore appears in no audio test, and
  YAML.Audio's corrupt-record and missing-blob branches are reached by nothing.
- **BR-7** [Important] `pin-not-at-the-loop` TestASittingFetchesARecordingOnce never calls runPlay, so the M1 regression is not pinned where it lived
  Plan Task 1 Steps 5-6 specify a behavioural pin at the loop through the CDN recorder —
  "fakeCDN + runPlay, two questions on one word". The delivered test
  (cmd/define/play_loop_test.go:4328) calls seam.Fetch three times, which
  TestAudioSeamServesRepeatsFromMemory already asserts. The trailing field-type assertion
  is a real guard, but it is not the loop-level behaviour the plan named.
- **BR-8** [Important] `runtime-artifact-undocumented` README's and the atlas's working-directory listings do not mention audio/, the new (and first binary) runtime artifact
  .gitignore derived the new directory from RuntimeDirs, but the two hand-maintained
  restatements did not: cmd/define/README.md:501-520 and atlas/define.md:605-616 both
  enumerate every other runtime artifact and omit audio/<lang>/. The atlas gained prose
  about the cache elsewhere; the enumeration a reader greps did not move. ARCH-PURPOSE
  shadow-sweep.
- **BR-9** [Important] `revision-overwritten` The plan's historical Revisions prose was overwritten by a blind symbol substitution and no longer reads as English
  Satisfying the retired-symbol guard rewrote completed revision entries in place, against
  AGENTS.md's append-don't-overwrite rule, leaving text such as "the decorator it
  replaced's doc comment records", "the taxonomy the decorator it replaced already
  documents", "every non-test the wrap that used to be remembered call must be a subset",
  and "the wrap that used to be remembered becomes idempotent". The record of what rounds
  2-4 decided is now unreadable.
- **BR-10** [Minor] `prefix-separator-invariant` audioNameSep's claim that a slug cannot contain "--" is false, so --forget re deletes re-'s recordings
  Probe: Slug("re-") == "re--ddf427", Slug("anti-") == "anti--0aa4e1",
  Slug("well- known") == "well--known-885c00". removeByPrefix globs "<slug>--", so
  forgetting "re" reaches "re-"'s files. Cost is one wasted refetch, but the invariant
  written into the comment is not the one Slug provides.
- **BR-11** [Minor] `unbounded-input-read` The cached blob is read with an unbounded os.ReadFile where the network path caps at maxAudioBytes
  cmd/define/store/yaml.go:833 loads the whole file into memory. ARCH-SECURE: a persisted
  artifact that an older version, another process, or a hand edit may have written is not
  trustworthy just because this program produced it.
- **BR-12** [Minor] `stale-blob-on-verdict` SetAudio with Missing set skips the blob write but does not remove an existing .mp3
  A key that was a hit and later becomes a verdict leaves an orphan blob until Forget.
  Not served — the record gates it — but it is debris in a directory with no eviction.
- **BR-13** [Minor] `plan-table-drift` The plan's entity tables drift from the code: audioKey/audioRecord vs AudioKey/AudioRecord, no store/audio_test.go, mergeRegions has no row
  The types are exported because they cross the package boundary; the named colocated
  test file was never created and its rows landed in storetest/suite.go instead, which is
  arguably the better home but is not what the plan says.
- **BR-14** [Minor] `test-name-overclaims` TestClozeOptionsAreClickableButNotColoured asserts only colour and never a click target
  It checks admitsColour() on three form names. The clickability half is in
  TestADeckSurfaceIsClickableButUncoloured, so the name promises coverage the body does
  not contain.
- **BR-15** [Minor] `tracking-in-two-places` The plan file has zero ticked steps while the issue's Plan ticks M1
  Both chunks of workshop/plans/000046-audio-cache-and-deck-words-plan.md remain
  unchecked after the work landed, so the plan's own checklist cannot be used to see what
  this boundary delivered.

## Round 3 — 2026-09-07T16:22:33-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — The third shape shipped: deps.audio is *audioSeam, both wrap lines are gone, and the retired-symbol guard now forbids the old constructor.
- BR-2 — addressed — audio/ is flat; re-scoping it by language reddens TestForgetTakesARecordingFetchedInAnotherLanguage and the scoped guard (mutation-verified).
- BR-3 — addressed — Mutation-verified: deleting the withStore block reddens TestWithStorePutsTheDiskCacheUnderTheMemo with the right message.
- BR-4 — addressed — Full suite green at HEAD under default and conformance tags; gofmt -l and go vet ./... clean.
- BR-5 — addressed — Mutation-verified: many true to false reddens three storetest rows on both twins plus the language-switch test.
- BR-6 — not-addressed — failingStore now has a call site, but the corrupt-file and unwritable-directory rows and YAML.Audio's two defensive branches remain unpinned.
- BR-7 — addressed — Renamed to what it asserts and the argument recorded; note that playSession IS drivable in-process, so the "a sitting cannot be driven" claim is broader than true.
- BR-8 — addressed — Both enumerations now list audio/ — but with the superseded filing scheme; raised separately below.
- BR-9 — not-addressed — No Revisions entry was appended for this round and the body was edited in place again, introducing a fresh break at plan lines 365-366.
- BR-10 — addressed — Directory-per-word removes the separator claim entirely; the re/re- conformance row plants a fixture that reaches the branch on both twins.
- BR-11 — addressed — Blob read now capped by readCapped; the sibling record read and the doc/behaviour mismatch are raised below in the same family.
- BR-12 — not-addressed — Mutation-verified: reverting the os.Remove(blob) leaves the store package fully green — the test asserts only through Audio, which gates on the record.
- BR-13 — not-addressed — audioKey/audioRecord, store/audio_test.go and the missing mergeRegions row all persist, and new drift arrived with the reversal.
- BR-14 — addressed — The test now drives writeWords and asserts two RegionWord click targets as well as the absence of colour.
- BR-15 — not-addressed — Still 0 of 45 plan checkboxes ticked while the issue's Plan ticks M1 and M2.

### Raised

- **BR-16** [Important] `runtime-artifact-undocumented` The audio filing reversal was swept through the code paths and left ten restatements of the superseded scheme, two of them doc comments on the field it changed
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
- **BR-17** [Minor] `lang-switch-derivation` applyLang's enumeration of what a language switch re-derives still does not account for d.audio, which holds a store bound to the pre-switch language
  This is the 2nd finding in family `lang-switch-derivation`. The behaviour is correct now
  only because audioDir is flat, so d.audio's stale store reference is harmless — but
  applyLang (cmd/define/command.go:397-399) explicitly lists its deliberate exclusions
  ("Deliberately NOT here: d.history") and d.audio is in neither list. The rule, not the
  line: every deps member holding a language-derived object appears in applyLang's
  enumeration or in its exclusion clause, with the reason — here, that audio/ carries no
  language shelf.
- **BR-18** [Minor] `unbounded-input-read` Only the blob was bounded — the record YAML beside it is still read with an unbounded os.ReadFile, readCapped truncates where its doc says it refuses, and no test pins the cap
  This is the 2nd finding in family `unbounded-input-read`, and the sibling read is in the
  same function as the one the first finding named: yaml.go:838 reads the record with
  os.ReadFile while yaml.go:843 caps the blob. readCapped's doc says "refusing anything past
  max" but io.LimitReader truncates, which matches the network path's behaviour and not the
  comment. Do not just cap the record. The rule: state once, at the store boundary, which
  persisted reads are bounded and why the rest (words/, facts/, items/) are not — a per-site
  fix is what left the sibling in place here.

## Round 4 — 2026-09-07T16:42:38-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 5 — 2026-09-07T17:01:41-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — All four degrade branches mutation-verified red-without-the-fix; failingStore now has a real call site at audiodisk_test.go:152.
- BR-9 — addressed — A Revisions entry was appended for this round; the substitution artifacts and the "go, along / along with" break are gone (grep clean).
- BR-12 — addressed — Mutation-verified: reverting os.Remove(blob) reddens TestAVerdictDeletesTheRecordingItSupersedes by name.
- BR-13 — not-addressed — mergeRegions, AudioKey/AudioRecord and Task 2's Create list are fixed; plan lines 217, 232, 255 and 346 still drift from the code.
- BR-15 — addressed — 45 of 45 plan checkboxes ticked, none left unchecked.
- BR-16 — not-addressed — Code paths, README, atlas and the lessons rule all landed; the prescribed grep is line-oriented and misses three wrapped restatements.
- BR-17 — not-addressed — cmd/define/command.go is not in this window at all; d.audio is still in neither applyLang's enumeration nor its exclusion clause.
- BR-18 — not-addressed — The cap is now pinned (mutation-verified), but the sibling record read is still unbounded and the boundary-level statement was not written.

### Raised

- **BR-19** [Important] `degenerate-payload-trusted` An empty recording is written to disk and served as a permanent, never-expiring hit — the outcome the new missing-blob branch exists to prevent
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
- **BR-20** [Minor] `unsourced-cross-reference` Two comments justify the 4MB cap and the .mp3 extension by citing a README passage that does not exist
  readCapped says "The directory is documented as inspectable and hand-editable"
  (cmd/define/store/yaml.go:826-827) and audioBlobExt says "for a human browsing the
  directory, which the README documents as an invited workflow" (yaml.go:886-887).
  cmd/define/README.md describes what define writes and never invites inspection or
  editing; grepping it for hand-edit/editable/inspect/browse returns nothing. The claim is
  load-bearing — it is the entire ARCH-SECURE argument for bounding the read. The rule: a
  comment that justifies a decision by citing another document must be checkable against
  it. Either state it once in the README and cite that, or drop the citation and keep the
  cap on its own merits.

## Round 6 — 2026-09-07T17:31:05-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — not-addressed — Table names and the mergeRegions row are fixed; the plan still cites `store/audio_test.go`, which does not exist, and TestPlanCitesTestsThatExist checks only Test* names.
- BR-16 — addressed — yaml.go/README/atlas swept, grep + rule recorded in lessons.md; the plan body is superseded by the appended Revisions entry per AGENTS.md §1.
- BR-17 — not-addressed — command.go is not in the change window; applyLang's exclusion clause still names only d.history, and d.audio appears in neither list.
- BR-18 — not-addressed — The cap is now pinned by a test, but the sibling record read at yaml.go:859 is still an unbounded os.ReadFile and no single statement at the store boundary says which persisted reads are bounded.
- BR-19 — addressed — Mutation-verified: dropping the read guard reddens the truncated-blob subtest, dropping the write guard reddens the filesystem subtest, dropping both reddens TestAnEmptyResponseIsNotCachedAsAHit.
- BR-20 — addressed — README.md:535 now carries "Look at any of it, and edit it if you like" plus the untrusted-input paragraph, and both comments cite that passage.

### Raised

- **BR-21** [Important] `production-wiring-unpinned` Passing nil for the vocabulary at all three writeWords call sites leaves the whole suite green, so M2's production wiring is unpinned
  Probed in a scratch checkout of cd89212: replacing deckVocabulary(d) with nil at play_loop.go:229, play_loop.go:478 and main.go:877 keeps go test ./cmd/define/ green (only the git-dependent repo guards fail, identically at unmutated baseline). Every writeWords test passes a literal deckOf(...), so nothing observes the capability arriving from deps — M2 could be silently dead exactly as forWord's interface mismatch was. This is the 2nd finding in family production-wiring-unpinned (BR-3 was the same rule at M1). Do NOT just fixture the three sites: state the rule — a capability handed from deps into a write door is pinned by a test that observes it arriving from deps, not from a literal — and write the enumeration (the writeWords call sites, deckVocabulary's callers) so a fourth site is covered by construction. TestWithStorePutsTheDiskCacheUnderTheMemo is the shape that worked. Related: deckwords_test.go:264-266 claims in prose that it "fails if a call site stops passing the vocabulary", which it cannot.
- **BR-22** [Important] `degenerate-payload-trusted` The memo serves a zero-byte 200 as a hit for the whole sitting, with a from URL reportVoice prints as the voice that answered
  audioSeam.fetch (cmd/define/fetch.go:206) stores c.hits[key] = cachedAudio{data, from} with no emptiness predicate. Probed with one seam and two FetchFor calls against a CDN serving an empty body: one request, both calls return bytes=0 from="http://.../a.mp3" err=nil. speak then writes a 0-byte mp3 for afplay and reportVoice prints a record naming a URL that carried no recording. This is the 2nd finding in family degenerate-payload-trusted. Round 4 satisfied BR-19's literal ask (both store twins plus the write) but not the class it named — the payload cannot be a recording. Do not patch the memo alone: enumerate every layer that can produce, remember, persist or play a fetch result (httpAudioSource.Fetch, audioSeam.hits, diskAudioCache.put, both SetAudio implementations, speak) and put the predicate where the payload is first judged, so a 0-byte 200 is not an answer and every layer below is correct by construction. The store.Store interface doc (store.go:88-91) is part of that sweep: it documents the verdict form of SetAudio and is silent that a non-Missing call with empty data is refused.
- **BR-23** [Minor] `claimed-coverage-absent` The Done-when row names TestAPromptRegionCoversTheTextItClaims as exercising RegionWord, but that test can only see promptRegions
  workshop/issues/000046-...md:275-277 claims numRegionKinds, TestEveryRegionKindIsActionable, TestAtlasDescribesEveryRegionKind and TestAPromptRegionCoversTheTextItClaims all exercise the new kind unedited. The last one loops over promptRegions(q) (play_loop_test.go:4276), which only ever yields RegionHeadword. The invariant is genuinely pinned — by TestWordRegionsCoverTheTextTheyClaim — so this is a wrong citation rather than a gap. 2nd finding in family claimed-coverage-absent: the rule is that a Done-when row names the guard whose assertion would go red, verified by asking which assertion that is, not by which test name sounds adjacent.
- **BR-24** [Minor] `two-statements-one-fact` The 4MB audio ceiling is stated twice, in two packages, with nothing holding them equal
  maxAudioBytes (cmd/define/fetch.go:35) and maxAudioBlobBytes (cmd/define/store/yaml.go:892) are the same number written twice; the comment justifies it ("the store cannot import main") but nothing detects drift. Exporting one and reading it from the other costs a line.
- **BR-25** [Minor] `doc-claims-unbacked` deckSpan.Word is documented as the deck key but holds the display text
  cmd/define/deckwords.go:26 says "Word is the deck key, which is what a click plays"; deckSpans assigns sp.text, so a sentence-initial or differently-spaced match carries the on-screen spelling. Harmless today because AudioCandidates lower-cases and store.Key normalises before filing, but the doc is a claim a caller would rely on.

## Round 7 — 2026-09-07T17:50:46-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — not-addressed — Entity-table names and the mergeRegions row are right, but the plan still cites `store/audio_test.go` at lines 217 and 421 and no such file exists.
- BR-17 — not-addressed — cmd/define/command.go is untouched by the window; applyLang's exclusion clause (command.go:397) still names only d.history and d.audio is in neither list.
- BR-18 — not-addressed — yaml.go:859 still reads the record with an unbounded os.ReadFile beside the capped blob at :871; readCapped truncates where its doc says it refuses, and yaml_test.go's own subtest is named "is truncated"; no single statement at the store boundary says which persisted reads are bounded.
- BR-21 — not-addressed — Mutation-verified at HEAD: `nil` at a call site now reddens the new guard, but swapping all three sites to `vocabularyFor(d, opt)` leaves the whole cmd/define suite green — the guard reads the argument's source token, not the capability's arrival from deps, and deckwords_test.go:268-269's false prose is unchanged.
- BR-22 — not-addressed — The memo layer is fixed and pinned, but httpAudioSource.Fetch (fetch.go:70) still returns a zero-byte 200 as success and stops the candidate walk there, and store.go:90-93 is still silent that a non-Missing SetAudio with empty data is refused — both named explicitly by the finding.
- BR-23 — not-addressed — The issue row at :275-277 is unchanged, and the same citation appears at plan:623 and plan:731 — where Task 5 Step 4 is ticked for an extension of TestAPromptRegionCoversTheTextItClaims that did not happen.
- BR-24 — not-addressed — maxAudioBytes (fetch.go:35) and maxAudioBlobBytes (yaml.go:892) are both unchanged; nothing holds them equal.
- BR-25 — addressed — deckwords.go:25-32 now says "the span's text AS WRITTEN, not the normalised deck key", which matches deckSpans assigning sp.text, and names why it is harmless.

### Raised

- **BR-26** [Minor] `claimed-coverage-absent` The "every deck word is clickable" Done-when row now carries two false statements, one of them created by this round's own fix
  workshop/issues/000046-...md:279-285 says TestEveryDeckWordInASittingIsClickable "drives it through writeWords rather than through the rule, so it fails if a call site stops passing the vocabulary" — it passes a literal deckOf(...) and cannot — and then concedes "A new writeWords call site is covered by review, not by construction — carried to #30", which round 6's TestEveryWriteWordsCallSitePassesAVocabulary made stale in the same commit that left the first clause standing. This is the 3rd finding in family claimed-coverage-absent (BR-6, BR-23 preceding it). Do NOT fix this row. State the rule and sweep the enumeration it implies: a row, a ticked plan step, or a test doc comment that names a guard must name the guard whose assertion would go red, verified by asking which assertion that is — and when a round ADDS a guard, the concessions that said it did not exist are part of the same commit's sweep. The enumeration is the Done-when rows, the ticked plan steps (Task 5 Step 4 and Verification row 6 are both wrong for the same reason), and the doc comments on the tests this window added.
- **BR-27** [Minor] `runtime-artifact-undocumented` The seam reversal never got the grep the filing reversal got, and left a dead double plus three present-tense claims about it
  noAudioSource (cmd/define/main_test.go:20) is declared and instantiated nowhere in the package after deps.audio became *audioSeam. play_loop_test.go:64 states "playRig deliberately installs noAudioSource AND noAudio:true" when playRig (:41) does newAudioSeam(nil); play_loop_test.go:1444 calls it "the package's other double"; fetch.go:123 refers to "the tests that used to write noAudioSource{}" while the type itself was left behind. Separately, the plan's writeWords bullet says "NEW, beside writeRendered rather than replacing it. The plain door still serves callers with nothing to mark" — writeRendered has exactly one caller in the tree, writeWords itself. This is the 3rd finding in family runtime-artifact-undocumented (BR-8, BR-16). Do NOT patch these four sites. The rule is already in lessons.md for the FILING reversal — "write the grep before the fix, run it after" — and M1 contained a SECOND reversal (cachingAudioSource -> audioSeam) that never got one. Write the grep for the retired seam vocabulary, run it across cmd/, atlas/, README and the plan, and add "one grep per reversal, not one per issue" to the rule.
- **BR-28** [Minor] `persisted-field-unvalidated` AudioRecord.From is read back from a hand-editable file and printed as the record of which voice answered, with no check that it is one of the URLs asked for
  diskAudioCache.Fetch (cmd/define/audiodisk.go:88) returns rec.From straight from audio/<slug>/<digest>.yaml. speak passes it to reportVoice, whose spokeSource decides by membership in sourceCandidates() and whose own doc says the report "survives on a pipe and cannot be taken back — and a record has to be true". README.md:535 now explicitly invites editing that directory, and this same round added readCapped and the empty-payload predicate on exactly that reasoning — the payload was hardened and the provenance field beside it was not. A hand-edited From flips the fallback announcement in either direction: absent when it should fire, or fired when the source recording really did answer. ARCH-SECURE: an input crossing a process boundary is untrusted even when this program wrote it. Cheapest fix is one membership check where the record is read — if rec.From is not in urls, treat the entry as nothing cached rather than as evidence.

## Round 8 — 2026-09-07T18:14:42-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — not-addressed — Table fixed (AudioKey/AudioRecord, mergeRegions row added); the Test-surface paragraph at :216-219 still names a store/audio_test.go that does not exist, and :203/:449/:456/:827 still spell the types lowercase.
- BR-17 — not-addressed — applyLang (command.go) still lists neither d.audio among its members nor among its "Deliberately NOT here" exclusions.
- BR-18 — not-addressed — yaml.go:838 still reads the record with an unbounded os.ReadFile beside the capped blob at :843, and readCapped's doc still says "refusing" where io.LimitReader truncates. The blob cap IS pinned (yaml_test.go, "a blob larger than the cap is truncated") — that half of the finding was inaccurate.
- BR-21 — addressed — Mutation-verified: replacing d with deps{} at play_loop.go:229 reddens TestEveryWriteWordsCallSitePassesAVocabulary by name; nil no longer compiles.
- BR-22 — addressed — Mutation-verified separately at both layers; the store.Store interface doc now states the empty-payload refusal.
- BR-23 — not-addressed — TestAPromptRegionCoversTheTextItClaims still loops promptRegions(q) only (play_loop_test.go:4276); the Done-when row still cites it.
- BR-24 — not-addressed — maxAudioBytes and maxAudioBlobBytes still independent; news.go:22 makes it three statements of the same number.
- BR-26 — not-addressed — Both clauses of the Done-when row are unchanged, as are Task 5 Step 4 and Verification row 6.
- BR-27 — not-addressed — noAudioSource still declared at main_test.go:20 with zero instantiations; the three present-tense claims stand; no grep for the seam reversal and no "one grep per reversal" rule in lessons.md. A THIRD reversal is now visible in the same artifact — the plan's audioDir bullet still says "the per-language audio directory".
- BR-28 — not-addressed — audiodisk.go:88 still returns rec.From unchecked against urls, and README.md:538 now claims persisted input is treated as untrusted.

### Raised

- **BR-29** [Important] `discovery-colour-scope` A meaning question now paints the word under test deck-green, which is the one word on screen where the discovery rule says it should not
  surfaceOf("meaning") is surfaceProse and Choice.Prompt() is word + "\n\n" + options, so colourOutside runs over the headword line too. Probed at HEAD with a deck holding the headword the prompt writes "\n\x1b[1;32mephemeral\x1b[0m\n\n1  ...". New behaviour — the prompt was written through writeRendered with no colour before. TestChoiceOptionGlossesAreColoured cannot see it because its vocabulary omits the headword, and the operator's pty pass covered the cloze and reveal frames only. Either classify the headword line out of the colour pass or pin the decision with a deck containing the headword.
- **BR-30** [Important] `runtime-artifact-undocumented` README's and the atlas's working-directory listings restate store.RuntimeDirs by hand with no guard, and README already omits usage/
  This is the 4th finding in family runtime-artifact-undocumented (BR-8, BR-16, BR-27). Do NOT fix another instance. RuntimeDirs has four consumers; the two derived ones are guarded (TestGitignoreCoversRuntimeDirs, TestPerWordDirsCoverEveryRuntimeDir) and the two documentation ones are not. Measured prevalence: README.md contains no occurrence of "usage/" at all, so the listing is missing a runtime directory nobody noticed. The class fix is TestStoreLayoutDocsNameEveryRuntimeDir, mirroring TestStoreLayoutDocsNameEveryEventKind at doc_sync_test.go:622 and reusing its layoutBlockIn helper over both markers.
- **BR-31** [Important] `plan-table-drift` The durable plan has no Revisions entry for close rounds 3-7, and its writeWords bullet states a signature the head commit removed
  This is the 2nd finding in family plan-table-drift. The plan's last Revisions entry is "M1 boundary review, rounds 1 and 2"; five commits since then changed the design. plan:690 says writeWords "takes the vocabulary and the surface" — a1d339b removed that argument, in the commit that left the sentence standing. plan:257 still calls audioDir "the per-language audio directory", falsified by the flat-shelf reversal whose vocabulary BR-16's grep could not match. State the rule: a plan artifact's entity description is swept in the same commit that changes the entity's shape, by an APPENDED Revisions entry per AGENTS.md section 1, and each reversal earns its own grep. Enumeration for this round: the writeWords signature, the audioDir shelf, the empty-payload predicate's three layers, the two AST guards.
- **BR-32** [Minor] `claimed-coverage-absent` Two guard clauses added in this window cannot fire — AudioKey.ok's Word check is a tautology and the AST guard's nil branch does not compile
  This is the 4th finding in family claimed-coverage-absent (BR-6, BR-23, BR-26). Earlier rounds fixed prose instances; this is the same rule in code. Do NOT fix these two sites — state the rule (every guard clause needs a witness input that makes it fire; write the witness or delete the clause) and sweep the clauses this window added. Instances: store/audio.go:63, where Slug never returns "" (Slug("") = "w-e3b0c4"), so ok() is effectively just Digest != "" — probed, an AudioKey with an empty Word files a readable hit under audio/w-e3b0c4/ that Forget("") cannot remove, reachable through the newly exported NewAudioKey/SetAudio surface though not from production. And deckwords_test.go:409, where arg.Name == "nil" is dead because nil is not assignable to the struct-typed deps parameter (verified: "cannot use nil as deps value"), so the package would not build and the test would never run.
- **BR-33** [Minor] `two-statements-one-fact` newHTTPAudioSource2 is a rename wrapper around cdn.source() whose name reads as a second constructor
  fetch_test.go:295 defines newHTTPAudioSource2(c *fakeCDN) *httpAudioSource { return c.source() } beside the real newHTTPAudioSource. Inline the call.

## Open findings

- **BR-13** [Minor] `plan-table-drift` The plan's entity tables drift from the code: audioKey/audioRecord vs AudioKey/AudioRecord, no store/audio_test.go, mergeRegions has no row
- **BR-17** [Minor] `lang-switch-derivation` applyLang's enumeration of what a language switch re-derives still does not account for d.audio, which holds a store bound to the pre-switch language
- **BR-18** [Minor] `unbounded-input-read` Only the blob was bounded — the record YAML beside it is still read with an unbounded os.ReadFile, readCapped truncates where its doc says it refuses, and no test pins the cap
- **BR-23** [Minor] `claimed-coverage-absent` The Done-when row names TestAPromptRegionCoversTheTextItClaims as exercising RegionWord, but that test can only see promptRegions
- **BR-24** [Minor] `two-statements-one-fact` The 4MB audio ceiling is stated twice, in two packages, with nothing holding them equal
- **BR-26** [Minor] `claimed-coverage-absent` The "every deck word is clickable" Done-when row now carries two false statements, one of them created by this round's own fix
- **BR-27** [Minor] `runtime-artifact-undocumented` The seam reversal never got the grep the filing reversal got, and left a dead double plus three present-tense claims about it
- **BR-28** [Minor] `persisted-field-unvalidated` AudioRecord.From is read back from a hand-editable file and printed as the record of which voice answered, with no check that it is one of the URLs asked for
- **BR-29** [Important] `discovery-colour-scope` A meaning question now paints the word under test deck-green, which is the one word on screen where the discovery rule says it should not
- **BR-30** [Important] `runtime-artifact-undocumented` README's and the atlas's working-directory listings restate store.RuntimeDirs by hand with no guard, and README already omits usage/
- **BR-31** [Important] `plan-table-drift` The durable plan has no Revisions entry for close rounds 3-7, and its writeWords bullet states a signature the head commit removed
- **BR-32** [Minor] `claimed-coverage-absent` Two guard clauses added in this window cannot fire — AudioKey.ok's Word check is a tautology and the AST guard's nil branch does not compile
- **BR-33** [Minor] `two-statements-one-fact` newHTTPAudioSource2 is a rename wrapper around cdn.source() whose name reads as a second constructor
