---
gate: boundary-review
issue: 23
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-28T12:12:35-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
          detail: |-
            Task 7 Step 3 and Task 10 Step 3 call milestone-close/close, but the issue's
            Plan holds a single non-Mx row. close.go:554 matches the Mx checkbox against
            the ISSUE body and only warns on a miss (close.go:560), and
            findMilestonesMissingVerdict (close.go:1717) reads that same section — so at
            the full close the "was M1 reviewed" guard finds zero milestones and passes
            vacuously. Add the two Mx rows to the issue's Plan before starting M1.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: declared-boundary-untracked
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-28T12:12:35-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-08-28T12:31:12-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: The issue's Plan now carries `- [ ] M1 —` and `- [ ] M2 —` rows; I checked them against the binary's own regexes (close.go:554 tick pattern and milestonePlanRE at close.go:1667) and both match, so the milestone-verdict guard will no longer pass vacuously.
          round: 3
      findings:
        - id: BR-2
          severity: Important
          title: user-model.md is derived from the language-scoped deck but stored unscoped, so a --reflect in one language replaces the other language's model
          detail: 'reflect.go:318 reads the language-scoped d.deck.Deck() while reflect.go:397 writes the unscoped userModelFile() (store/yaml.go:88), and ask.go:263 reads that one file in every language. Reproduced with the existing reflectRig: --reflect over an English deck, then over a Spanish deck in the same directory, leaves the English session reading "A2 — Spanish beginner / Read off: madrugar". This is the same enumeration class C1 named, one member further out — a persisted artifact derived from the language. atlas/define.md, shipped in this range, justifies leaving it unscoped with the events/ argument, which does not transfer.'
          family: language-derived-state-unscoped
          round: 3
        - id: BR-3
          severity: Minor
          title: applyLang assigns opt.lang, contradicting that field's own documented meaning
          detail: command.go:367 writes opt.lang = l, while main.go:329-333 documents options.lang as "the -lang FLAG, empty when it was not given — not the language in effect". Nothing reads it after withStore, so the behaviour is fine; the comment is false after a switch.
          family: comment-contract-drift
          round: 3
        - id: BR-4
          severity: Minor
          title: the .tmp-* shadows writeBytesAtomic leaves beside a runtime FILE are covered by neither .gitignore nor the basename guards
          detail: store/yaml.go:310 creates .tmp-* in the target's directory. For words/ and usage/ that directory is itself ignored; for lang.txt and user-model.md it is the working-directory root, where .gitignore has no .tmp-* pattern and isRuntimeFile cannot match a random name. Pre-existing for user-model.md, but it is the part of the RuntimeFiles class the class fix does not reach.
          family: runtime-artifact-guard-coverage
          round: 3
        - id: BR-5
          severity: Minor
          title: ParseLang's stated rationale argues against a whitelist, not for the two-letter limit it actually imposes
          detail: store/lang.go:41 requires exactly two ASCII letters, refusing pt-br, zh-hans and ISO 639-3 tags. The comment explains only why there is no list of known languages. One sentence naming the CDN's _xx_yy_ path shape as what fixes the length would make the constraint a decision rather than an artifact.
          family: comment-contract-drift
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-28T12:55:52-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: userModelFile() is per-language and MigrateToLanguages moves the flat file; verified by reverting yaml.go:115 in a scratch copy of e393f2d9 — TestTheUserModelIsPerLanguage goes red with the exact cross-language clobber.
          round: 4
        - id: BR-3
          disposition: addressed
          note: applyLang no longer assigns opt.lang and command.go:371-374 states why a switch does not retroactively make the flag present.
          round: 4
        - id: BR-4
          disposition: addressed
          note: .tmp-* is in RuntimeFiles, .gitignore and both basename guards; git check-ignore -v matches .tmp-abc123 and still misses the golden fixture. See BR-7 for the remaining pin gap, which is a different rule.
          round: 4
        - id: BR-5
          disposition: addressed
          note: store/lang.go:39-43 now names the CDN's _<lang>_<locale>_ shape and the path segment as what fixes the length, with pt-br and ISO 639-3 refused deliberately.
          round: 4
      findings:
        - id: BR-6
          severity: Important
          title: The user-model rename swept 3 of ~12 restatements, including --reflect's success line, which names a file it did not write
          detail: '3rd finding in this family — do NOT fix the instance. The rule, which this commit already stated for itself: no string, comment, README line, atlas line or plan line may spell a runtime artifact''s filename or a code-owned symbol name; name the artifact, or derive the name. Enforceable with the legacyRuntimeFilePaths exact-set ratchet idiom. Measured prevalence at HEAD: reflect.go:405 prints "wrote user-model.md" after writing user-model.<lang>.md; store/yaml.go:89 says userModelFile is NOT per-language, 26 lines above yaml.go:115; comments at askctx.go:37, store/store.go:25, store/mem.go:17; README 142/148/218 contradict README 182; atlas/define.md 808/821/908/961 contradict 220. Same rule, symbol form: main.go:248 and atlas/define.md:238 name MigrateFlatDeck, which the tree does not export (MigrateToLanguages does; migrateFlatDeck is unexported), and the plan names it at :80, :143, :186, :258, :296 — round 1''s I4 (deckDeps) recurring one function further out.'
          family: comment-contract-drift
          round: 4
        - id: BR-7
          severity: Important
          title: RuntimeFiles coverage is asserted from hand-typed literals, so changing the atomic-write temp prefix escapes every guard green
          detail: '2nd finding in this family — do NOT fix the .tmp- instance. The rule: a runtime artifact''s name has exactly one producing function; guards, migrations and tests derive from it and never restate it. langFile/langFileName is already the model. Mutation-verified on a scratch copy of e393f2d9: os.CreateTemp(filepath.Dir(path), "tmp-*") at store/yaml.go:337 leaves go test ./cmd/define/store/ fully green, because store/lang_test.go:142 appends the literal ".tmp-123456" rather than observing the writer — while atlas/repo-guards.md:96-100 documents that test as asserting the real output of the writing functions. The shadow beside user-model.<lang>.md and lang.txt in the working-directory root then matches no .gitignore pattern. Same shape at store/migrate.go:117, which hand-writes "user-model."+DefaultLang+".md" instead of calling userModelFile() in its own package, with migrate_test.go:186 asserting the same literal. Prevalence: 3 writer names, 1 derived. Fix: a shared tmpPattern const consumed by os.CreateTemp and RuntimeFiles, migrateUserModel deriving its destination, and no literals left in TestRuntimeFilePatternsCoverWhatWeWrite.'
          family: runtime-artifact-guard-coverage
          round: 4
        - id: BR-8
          severity: Minor
          title: The project file records M1's actual and closed date from before three rounds of boundary-review fixes
          detail: workshop/projects/define-learn.md:405-406 carry actual 1.03h and closed 2026-08-28, written in 30ee9e2 before the C1/I1-I4 and BR-2..BR-5 fix commits. Re-measure at the real close rather than leaving a number that understates by the whole review cost.
          family: project-ledger-lag
          round: 4
        - id: BR-9
          severity: Minor
          title: README never states that -locale is English-only, three lines above the new -lang example
          detail: README.md:37 shows `define -locale gb colour`; D2's rule (honoured for English only, with a diagnostic otherwise) is documented in atlas/define.md:1084-1091 and nowhere in README. Belongs to the same sweep as the other finding in this family.
          family: comment-contract-drift
          round: 4
        - id: BR-10
          severity: Minor
          title: The PATTERNS explanation at store/yaml.go:75-87 is a detached comment godoc attaches to nothing
          detail: Blank line before it and after it, so it documents neither RuntimeFiles nor wordsDir. Merge it into the RuntimeFiles doc comment. Same for store/migrate.go:11, whose opening line says MigrateToLanguages moves a DIRECTORY when it moves two named artifacts.
          family: comment-contract-drift
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-08-28T13:18:45-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: Rule stated and mechanised for Go; ratchet verified reachable by planting a spelling in reflect.go. Residual prose scope raised as a new finding.
          round: 5
        - id: BR-7
          disposition: addressed
          note: 'Mutation-verified twice on scratch copies: tmpPattern and userModelPrefix changes each redden TestGitignoreCoversRuntimeFiles by name.'
          round: 5
        - id: BR-8
          disposition: addressed
          note: 2.15h now recorded; consistent with sdlc actual at HEAD (7.05h) minus the 4.92h pre-implementation baseline the issue Log documents.
          round: 5
        - id: BR-9
          disposition: addressed
          note: README.md:36 marks the example English-only and a new paragraph states the rule.
          round: 5
        - id: BR-10
          disposition: addressed
          note: PATTERNS comment merged into the RuntimeFiles doc; MigrateToLanguages' doc names the two artifacts it moves.
          round: 5
      findings:
        - id: BR-11
          severity: Important
          title: BR-6's rule binds prose but is enforced over *.go only, and the hand-swept half left a live false claim in the project file
          detail: |-
            6th finding in this family — do NOT fix the instance. The rule is already
            written and correct; what is missing is that its enforcement stops at
            `git ls-files '*.go'`, so the half the rule was actually raised about
            (README/atlas/plan lines) is still swept by hand. Measured at HEAD:
            README.md and atlas/ are clean, but workshop/projects/define-learn.md
            carries 6 live restatements (lines 5, 61, 140, 316, 349, 360, 494) plus 2
            historical ones, and :140 "A single user-model.md, batch-generated,
            human-correctable" and :494 "One user-model.md" are now FALSE — BR-2 made
            the model one file per language. :5 is the project's done_when frontmatter.
            A second live instance of the same family, pre-existing rather than
            introduced here: store/yaml.go:478-488 runs Forget's doc comment straight
            into the newsFile comment with no blank line, so godoc attaches "Forget
            removes one word file" to `type newsFile` — the same shape as BR-10, in
            the file BR-10 named. Class fix: extend the ratchet to markdown with an
            explicit allowlist (workshop/history/, the *-gate.md / *-review.md
            ledgers, and "## Revisions" sections, which legitimately record what was
            once true), and state that scope in the rule. For the orphaned comment the
            mechanical form is one check over top-level decls — a doc comment whose
            first word is not the declared identifier is attached to the wrong thing.
          family: comment-contract-drift
          round: 5
        - id: BR-12
          severity: Minor
          title: Two test fixtures/paths are hand-restated outside the store package; one is now stranded at the pre-#23 flat layout
          detail: |-
            3rd finding in this family — do NOT fix the words/en/ instance. The rule is
            the family's own: a path or name a test asserts against must come from the
            function that produces it. store/yaml_test.go:45 still plants
            ".tmp-halfwritten" at dir/words/, while wordsDir() is now words/en/, so
            TestYAMLIgnoresInterruptedWrites no longer exercises Deck()'s temp-file
            skip at all — the sibling test 25 lines below had its path corrected with a
            comment about exactly this. Honest caveat: removing Deck()'s .yaml suffix
            check leaves the whole suite green at BOTH base and HEAD (the fixture's
            unparseable body masks it), so this is a pre-existing weak pin that this
            range made structurally unreachable, not a regression in detection.
            Second instance: cmd/define/lang_cmd_test.go:180 restates "lang.txt" in a
            NEGATIVE assertion, which passes vacuously after a rename (:159's positive
            one would redden). store.UserModelName was exported so consumers could
            derive; there is no equivalent for the language file. Class fix: move the
            temp-file test into package store (internal, as lang_test.go already is) so
            it plants via y.wordsDir() and newTempFile(), and export a producer for the
            language filename so package main's tests derive it too.
          family: runtime-artifact-guard-coverage
          round: 5
      boundary: M1
      blocked: false
    - "n": 6
      timestamp: "2026-08-28T14:14:26-07:00"
      agent: claude
      dispose:
        - id: BR-11
          disposition: addressed
          note: TestProseDoesNotSpellStaleRuntimeArtifactNames binds README/atlas/workshop/projects with a currentTruthOnly shape-based record filter; I planted the exact false sentence at define-learn.md:150 and it went red, and appending it after "## Log" correctly did not. Forget's doc comment is reattached at store/yaml.go:534. Residual symbol-half scope raised as a new finding.
          round: 6
        - id: BR-12
          disposition: addressed
          note: TestDeckIgnoresInterruptedWrites moved to package store (atomic_internal_test.go) deriving paths from y.wordsDir() and newTempFile(); reverting Deck()'s .yaml suffix check turns it red with the half-written word in the deck. store.LangFileName() exported and used at lang_cmd_test.go:159 and :182.
          round: 6
      findings:
        - id: BR-13
          severity: Important
          title: applyLang does not re-derive d.usage, so D6's news gate holds at the boundary but not across a mid-session /lang
          detail: '2nd finding in this family — do NOT fix the instance by adding one line to applyLang. newsFeedFor is applied only in openStore (main.go:316) and sessionUsage (main.go:250), while applyLang (command.go:388) excludes d.usage as "not language-scoped" — true before this range, false after M2 made the feed''s presence a function of the language. Verified with a scratch test: after applyLang(&d,&opt,"es",...) the session still holds the English cachingFeed (bs2.news != nil); symmetrically a session started in es keeps news==nil after /lang en and silently loses the feed. Latent rather than live only because UsageSource.Usages has no production caller at HEAD (grep: d.usage is referenced only at main.go:182-183); it becomes silent wrong-language data the moment #10 wires it. The class: applyLang''s own doc states the generating rule ("anything derived from the language BEFORE a switch must be re-derived BY it") and the very next member added violated it, so an enumeration in a comment is not enforcement. Structural fix: extend the single newDeck builder to construct everything that is a function of the language (including the gated usage source) and have applyLang call exactly that builder, so a boundary-only derivation is unspellable. Pin at vocab_test.go:405, which already asserts d.history identity across the switch.'
          family: language-derived-state-unscoped
          round: 6
        - id: BR-14
          severity: Important
          title: The artifact-name rule's SYMBOL/MODEL half is still unenforced; 11 live restatements measured at HEAD
          detail: '7th finding in this family — do NOT fix these instances. Both ratchets count exactly one string, "user-model.", while the rule they enforce binds "a runtime artifact''s filename OR A SYMBOL THE CODE OWNS". The unmechanised half has now recurred four times (I4 deckDeps, BR-6 MigrateFlatDeck, and two fresh ones). Measured at HEAD - (1) dict_darwin.go:296 "the nine symbols this file resolves"; (2) dict_conformance_test.go:69 "Nine undocumented symbols"; (3) atlas/define.md:1145; (4) workshop/projects/define-learn.md:464 - dcs_resolve resolves THREE. (5) vocab.go:160 claims warnTo is "the one place" the "define: " prefix is written while dict_darwin.go:359 is a byte-identical second (ARCH-DRY; also a package-level func inside a darwin-only build tag). (6) README.md:331-333 is a stranded pre-M2 paragraph duplicating the one four lines above and falsely saying enabling Chinese dictionaries affects formatting, which the curated L to L selection now prevents. (7) atlas/index.md:9 still says "NOAD word lookup". (8) plan :116/:132 name dictChoice, absent. (9) plan :147/:153 name dcsDictionaries, absent. (10) plan :279 ticked, "assert all nine symbols resolve". (11) plan D6''s Bonus claims a live Google News request per Spanish lookup, which no code path makes. Class fix, cheap because the enumeration already exists - one test asserting every Name cell of a plan''s Core-concepts table resolves to a declared identifier at the stated path would have caught 8, 9 and both prior recurrences; and give the symbol COUNT one producer (a dcsSymbols slice the resolver and the docs both read) instead of four hand-typed copies.'
          family: comment-contract-drift
          round: 6
        - id: BR-15
          severity: Important
          title: systemDictionary's two fallback branches have no automated test on any platform, and the Done-when row is ticked on a manual experiment
          detail: 'dict_darwin.go:327 fuses the three-outcome policy and its user-facing warning text to installedDictionaries()'' cgo IO, so neither "the private surface is gone" nor "nothing curated matches" is reachable from a test. noadDictionary and everyActiveDictionary appear in no non-conformance test. The Done-when row "the seam FALLS BACK to today''s NULL behaviour if any is missing" and plan Task 9 Step 4 are both ticked on a manual symbol-misspelling run. ARCH-PURE (extract dictionaryFor(installed []dictMeta, lang) (ids []string, name string) as pure and leave the cgo shell thin) and ARCH-MOCK (with the metadata source injected, production and test finally share the selection boundary — the standard the M1 review sidecar set at 000023-deck-language-m1-review.md:289 and this milestone did not meet). Not hypothetical: I measured this machine''s shell context returning a single dictionary (com.apple.dictionary.Wikipedia), so ./define -lang es mesa takes the untested branch on every run here.'
          family: policy-inside-io-shell
          round: 6
        - id: BR-16
          severity: Important
          title: atlas/repo-guards.md does not name the two artifact-name ratchets this range added
          detail: 'repo-guards.md is the atlas catalogue of repo guards and was updated for RuntimeFiles, TestRuntimeFilePatternsCoverWhatWeWrite and legacyRuntimeFilePaths, but never names TestRuntimeArtifactNamesAreSpelledOnceInSource or TestProseDoesNotSpellStaleRuntimeArtifactNames — the two guards workshop/lessons.md calls this range''s class fix, and the ones a contributor most needs to find before adding a doc line. One table row each plus the records-versus-current-truth scope rule (## Revisions / ## Log / a block carrying **closed:**), so the deliberate scope decision is discoverable outside the test''s own comment.'
          family: atlas-lags-new-surface
          round: 6
        - id: BR-17
          severity: Minor
          title: selectedDictionary.Lookup lets a later ErrNoEntry overwrite an earlier "dictionary unavailable", the collapse its own C comment forbids
          detail: dict_darwin.go:270 assigns lastErr on every iteration, so status 3 on NOAD followed by status 1 on AppleDictionary reports ErrNoEntry — "this word does not exist in English" when the truth is "the primary dictionary vanished". dcs_lookup_in's comment at :118 states these two must not collapse. Keep the first non-ErrNoEntry error instead of the last error seen.
          family: comment-contract-drift
          round: 6
        - id: BR-18
          severity: Minor
          title: installedDictionaries indexes position i across N separately-copied CFSets, whose enumeration order the file itself calls unspecified
          detail: dict_darwin.go:189 calls dcs_describe(i) once per dictionary and each call re-invokes f_copy_available() and CFSetGetValues on a fresh copy. dcs_describe's own comment states iteration order is UNSPECIFIED; if two copies ever enumerate differently the returned slice gains duplicates and drops entries, and a dropped com.apple.dictionary.es.DGLEV silently degrades Spanish to the NULL search. Copy the set once and describe all N from that copy. Performance is not the concern (measured 46us warm).
          family: policy-inside-io-shell
          round: 6
        - id: BR-19
          severity: Minor
          title: The English fixture corpus is captured through the NULL search while production selects the curated identifiers
          detail: testdata/capture.sh:60 captures entries/en through DCSCopyTextDefinition(NULL) — "the host's ACTIVE dictionaries" — while systemDictionary(en) now selects com.apple.dictionary.NOAD and com.apple.dictionary.AppleDictionary. Capture path and production path therefore differ for English; they agree today only because this host's active set happens to match. Detectable (TestFixturesMatchLiveDictionary compares them, and did go red once), so this is a note rather than a defect — but passing the curated ids to capture.py would make the two agree by construction (ARCH-MOCK).
          family: policy-inside-io-shell
          round: 6
      blocked: true
    - "n": 7
      timestamp: "2026-08-28T14:44:27-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: addressed
          note: Verified by mutation — dropping d.usage from applyLang's adoption reddens vocab_test.go:463 in both directions. See new finding on the residual hand-enumeration.
          round: 7
        - id: BR-14
          disposition: not-addressed
          note: Class guard landed but has a comment-admitting loophole and three of the eleven measured instances are still live in the plan.
          round: 7
        - id: BR-15
          disposition: addressed
          note: dictionaryFor is pure and all three outcomes plus the empty-set case are unit-tested.
          round: 7
        - id: BR-16
          disposition: addressed
          note: Both ratchets are now table rows in atlas/repo-guards.md with the records-vs-current-truth scope rule.
          round: 7
        - id: BR-17
          disposition: not-addressed
          note: The fix does not fire — case 1 still assigns lastErr unconditionally, so status 3 then status 1 still returns ErrNoEntry.
          round: 7
        - id: BR-18
          disposition: addressed
          note: dcs_describe_all copies the set once and describes all N in a single pass.
          round: 7
        - id: BR-19
          disposition: addressed
          note: capture.sh walks EN_DICTS in curated order; capture.py exits 1 on no-entry so the first-hit-wins loop matches selectedDictionary.Lookup.
          round: 7
      findings:
        - id: BR-20
          severity: Important
          title: TestPlanTablesNameEntitiesThatExist passes on a COMMENT mention, and a stale `newDeck` row proves the hole
          detail: |-
            9th finding in this family — do NOT fix the newDeck row alone. repo_guard_test.go:595 accepts
            `strings.Contains(string(src), name+" ")` as evidence a symbol is declared, so any mention anywhere
            in the file satisfies it. The plan's Core-concepts row `newDeck` at cmd/define/main.go
            (plan:144) is stale — the tree renamed it to newLangDeps in this very commit — and the guard is
            green only because main.go:269 still says "newDeck stays nil on both of these paths". Mutation
            proof: rewording that one comment turns the test RED naming newDeck; removing the fallback clause
            flags exactly that one row across all active plans and nothing else, so no legitimate row needs it.
            The rule the family keeps failing: a guard may not accept prose as evidence about code. Delete the
            fallback (declared/assigned already cover every current row), then sweep newDeck from plan:144,
            :61, :84, :191, :196, main.go:52, :69, :269 and atlas/define.md:465.
          family: comment-contract-drift
          round: 7
        - id: BR-21
          severity: Important
          title: atlas/define.md still describes the pre-BR-13 /lang design, including the exact claim BR-13 disproved
          detail: |-
            2nd finding in this family — the rule is that the atlas is updated in the SAME commit as the
            surface it maps, not the same range. atlas/define.md:463-476 says the rebuild goes through
            `newDeck(lang)` (renamed to newLangDeps), that "usage reads usage/, so re-deriving them would put
            a second, unloaded History beside the one runEditor already Load()ed" — false, usage IS re-derived
            now and only history is not — and enumerates applyLang's members as d.lang, opt.voice, the deck
            triple and voc, with usage absent. The commit edited atlas/define.md but only around line 1142.
            The one current-truth document describing this invariant now contradicts the code on it.
          family: atlas-lags-new-surface
          round: 7
        - id: BR-22
          severity: Important
          title: langDeps is adopted field-by-field at two sites, and d.dict — a member since M1 — has no test at all
          detail: |-
            3rd finding in this family — do NOT fix by adding one assertion. The rule the fixes keep half-applying
            is: every member of the language-derived set must be (a) adopted as a WHOLE from the one builder and
            (b) pinned by a test that reddens when its re-derivation is removed. Neither half holds. Adoption is
            `d.deck, d.capture, d.vocab, d.usage = ld.deck, ld.capture, ld.vocab, ld.usage` (command.go:388) and
            the same four enumerated again into storeDeps (main.go:325-329), so a fifth langDeps field is
            forgettable at both sites — embedding langDeps in deps/storeDeps makes it one assignment and closes
            (a). For (b): d.newDict is set at ZERO call sites in any test, so deleting BOTH the boundary
            derivation (main.go:597-599) and the /lang re-derivation (command.go:388-390) leaves the whole
            cmd/define suite green (measured, 94s) while production would dereference a nil Dictionary. That is
            the milestone's headline Done-when row, and it is the same class as C1 and BR-13.
          family: language-derived-state-unscoped
          round: 7
        - id: BR-23
          severity: Important
          title: Two pieces of pure logic still live inside the darwin cgo shell; one shipped an inoperative fix, the other broke GOOS=linux
          detail: |-
            4th finding in this family — do NOT fix either instance alone. The rule: nothing that is a decision
            or a parse over data may live inside dict_darwin.go's build-tagged shell; dictselect.go is where it
            goes. The enumeration for this file is three items and only one is done. (1) selection policy →
            dictionaryFor, extracted ✓. (2) status-folding policy in selectedDictionary.Lookup — still inline,
            untestable, and consequently BR-17's fix shipped broken with no test to catch it. Extract
            foldLookupStatuses([]int, []string) error as pure. (3) parseDictRecords / parseLangPairs / baseLang —
            their own comments say "Pure, so the whole cgo boundary's format is testable without CoreServices",
            but they sit behind //go:build darwin while TestParseLangPairs is in the untagged dict_fake_test.go:203.
            Measured: `GOOS=linux go vet ./cmd/define/` now fails with `undefined: parseLangPairs`; it exited 0 at
            base 2e929fc5 (checked in a worktree), so this range regressed it, and dict_stub.go:13 still claims
            the stub keeps `go vet ./...` green off darwin.
          family: policy-inside-io-shell
          round: 7
        - id: BR-24
          severity: Important
          title: dcsPrivateSymbols is documented as the ONE producer, but dcs_resolve hand-writes the same three names in C
          detail: |-
            4th finding in this family — do NOT fix by syncing the two lists. Identical shape to BR-7: a guard
            asserting coverage from a hand-typed restatement. dict_darwin.go:200 declares the slice and its doc
            says "ONE producer for the list", while dcs_resolve at dict_darwin.go:43-45 spells
            DCSCopyAvailableDictionaries / DCSDictionaryGetIdentifier / DCSDictionaryGetLanguages again as C
            string literals. TestPrivateDictionarySurfaceStillResolves walks the Go copy, so adding or renaming a
            dlsym in the C preamble leaves the conformance check reporting "all present" while the resolver needs
            a symbol nobody verifies — and this check IS the whole mitigation for M2's stated OS-version risk.
            The rule: a list the code owns has one producer, and where a second language forces a restatement, a
            ratchet asserts the two agree. Cheap here — one test regexing dlsym("...") out of the file's own
            source and comparing the set to dcsPrivateSymbols, in the same shape as the existing ratchets.
          family: runtime-artifact-guard-coverage
          round: 7
        - id: BR-25
          severity: Minor
          title: Two doc comments stranded onto the wrong declaration by this commit's insertions
          detail: |-
            main.go:127-135 — storeDeps' doc ("the trio openStore produces") now runs into `type langDeps` with
            no blank line, so godoc attaches it to langDeps and storeDeps is undocumented. main.go:212-231 —
            openStore's doc runs into `func newsFeedFor` the same way, leaving openStore undocumented. Both
            declarations were inserted directly beneath an existing doc comment. Same shape as BR-10, twice, in
            the commit that closed it; a ratchet asserting a doc comment's first word matches the declaration it
            precedes is the mechanical form.
          family: comment-contract-drift
          round: 7
        - id: BR-26
          severity: Minor
          title: --help still says define looks words up in "macOS's active dictionaries", which is now the fallback path
          detail: |-
            main.go:435. After M2 the normal path is the curated per-language selection and the whole-active-set
            search is the degradation. README.md:314 was updated for this; the binary's own usage text, which the
            same commit edited to add the /lang paragraph, was not.
          family: comment-contract-drift
          round: 7
        - id: BR-27
          severity: Minor
          title: The private-surface conformance check routes SHAPE drift through SkipOrFail, so "says loudly" is a skip by default
          detail: |-
            dict_conformance_test.go:91. internal/conformance's own four-class rule puts an absent external
            dependency in SkipOrFail and "the dependency's surface moved" under SHAPE drift, which "ALWAYS
            fail[s] ... the very thing these suites exist to report". A vanished private symbol is the second
            class, and the issue's Done-when row asks it to say so LOUDLY. Under a plain
            `go test -tags conformance` it prints a skip, which the package doc itself says reads as green.
          family: comment-contract-drift
          round: 7
      blocked: true
---

# Gate ledger — tools#23 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T12:12:35-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `declared-boundary-untracked` The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
  Task 7 Step 3 and Task 10 Step 3 call milestone-close/close, but the issue's
  Plan holds a single non-Mx row. close.go:554 matches the Mx checkbox against
  the ISSUE body and only warns on a miss (close.go:560), and
  findMilestonesMissingVerdict (close.go:1717) reads that same section — so at
  the full close the "was M1 reviewed" guard finds zero milestones and passes
  vacuously. Add the two Mx rows to the issue's Plan before starting M1.
  (carried from plan-quality PQ-8, deferred to the boundary review)

## Round 2 — 2026-08-28T12:12:35-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-08-28T12:31:12-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — The issue's Plan now carries `- [ ] M1 —` and `- [ ] M2 —` rows; I checked them against the binary's own regexes (close.go:554 tick pattern and milestonePlanRE at close.go:1667) and both match, so the milestone-verdict guard will no longer pass vacuously.

### Raised

- **BR-2** [Important] `language-derived-state-unscoped` user-model.md is derived from the language-scoped deck but stored unscoped, so a --reflect in one language replaces the other language's model
  reflect.go:318 reads the language-scoped d.deck.Deck() while reflect.go:397 writes the unscoped userModelFile() (store/yaml.go:88), and ask.go:263 reads that one file in every language. Reproduced with the existing reflectRig: --reflect over an English deck, then over a Spanish deck in the same directory, leaves the English session reading "A2 — Spanish beginner / Read off: madrugar". This is the same enumeration class C1 named, one member further out — a persisted artifact derived from the language. atlas/define.md, shipped in this range, justifies leaving it unscoped with the events/ argument, which does not transfer.
- **BR-3** [Minor] `comment-contract-drift` applyLang assigns opt.lang, contradicting that field's own documented meaning
  command.go:367 writes opt.lang = l, while main.go:329-333 documents options.lang as "the -lang FLAG, empty when it was not given — not the language in effect". Nothing reads it after withStore, so the behaviour is fine; the comment is false after a switch.
- **BR-4** [Minor] `runtime-artifact-guard-coverage` the .tmp-* shadows writeBytesAtomic leaves beside a runtime FILE are covered by neither .gitignore nor the basename guards
  store/yaml.go:310 creates .tmp-* in the target's directory. For words/ and usage/ that directory is itself ignored; for lang.txt and user-model.md it is the working-directory root, where .gitignore has no .tmp-* pattern and isRuntimeFile cannot match a random name. Pre-existing for user-model.md, but it is the part of the RuntimeFiles class the class fix does not reach.
- **BR-5** [Minor] `comment-contract-drift` ParseLang's stated rationale argues against a whitelist, not for the two-letter limit it actually imposes
  store/lang.go:41 requires exactly two ASCII letters, refusing pt-br, zh-hans and ISO 639-3 tags. The comment explains only why there is no list of known languages. One sentence naming the CDN's _xx_yy_ path shape as what fixes the length would make the constraint a decision rather than an artifact.

## Round 4 — 2026-08-28T12:55:52-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — userModelFile() is per-language and MigrateToLanguages moves the flat file; verified by reverting yaml.go:115 in a scratch copy of e393f2d9 — TestTheUserModelIsPerLanguage goes red with the exact cross-language clobber.
- BR-3 — addressed — applyLang no longer assigns opt.lang and command.go:371-374 states why a switch does not retroactively make the flag present.
- BR-4 — addressed — .tmp-* is in RuntimeFiles, .gitignore and both basename guards; git check-ignore -v matches .tmp-abc123 and still misses the golden fixture. See BR-7 for the remaining pin gap, which is a different rule.
- BR-5 — addressed — store/lang.go:39-43 now names the CDN's _<lang>_<locale>_ shape and the path segment as what fixes the length, with pt-br and ISO 639-3 refused deliberately.

### Raised

- **BR-6** [Important] `comment-contract-drift` The user-model rename swept 3 of ~12 restatements, including --reflect's success line, which names a file it did not write
  3rd finding in this family — do NOT fix the instance. The rule, which this commit already stated for itself: no string, comment, README line, atlas line or plan line may spell a runtime artifact's filename or a code-owned symbol name; name the artifact, or derive the name. Enforceable with the legacyRuntimeFilePaths exact-set ratchet idiom. Measured prevalence at HEAD: reflect.go:405 prints "wrote user-model.md" after writing user-model.<lang>.md; store/yaml.go:89 says userModelFile is NOT per-language, 26 lines above yaml.go:115; comments at askctx.go:37, store/store.go:25, store/mem.go:17; README 142/148/218 contradict README 182; atlas/define.md 808/821/908/961 contradict 220. Same rule, symbol form: main.go:248 and atlas/define.md:238 name MigrateFlatDeck, which the tree does not export (MigrateToLanguages does; migrateFlatDeck is unexported), and the plan names it at :80, :143, :186, :258, :296 — round 1's I4 (deckDeps) recurring one function further out.
- **BR-7** [Important] `runtime-artifact-guard-coverage` RuntimeFiles coverage is asserted from hand-typed literals, so changing the atomic-write temp prefix escapes every guard green
  2nd finding in this family — do NOT fix the .tmp- instance. The rule: a runtime artifact's name has exactly one producing function; guards, migrations and tests derive from it and never restate it. langFile/langFileName is already the model. Mutation-verified on a scratch copy of e393f2d9: os.CreateTemp(filepath.Dir(path), "tmp-*") at store/yaml.go:337 leaves go test ./cmd/define/store/ fully green, because store/lang_test.go:142 appends the literal ".tmp-123456" rather than observing the writer — while atlas/repo-guards.md:96-100 documents that test as asserting the real output of the writing functions. The shadow beside user-model.<lang>.md and lang.txt in the working-directory root then matches no .gitignore pattern. Same shape at store/migrate.go:117, which hand-writes "user-model."+DefaultLang+".md" instead of calling userModelFile() in its own package, with migrate_test.go:186 asserting the same literal. Prevalence: 3 writer names, 1 derived. Fix: a shared tmpPattern const consumed by os.CreateTemp and RuntimeFiles, migrateUserModel deriving its destination, and no literals left in TestRuntimeFilePatternsCoverWhatWeWrite.
- **BR-8** [Minor] `project-ledger-lag` The project file records M1's actual and closed date from before three rounds of boundary-review fixes
  workshop/projects/define-learn.md:405-406 carry actual 1.03h and closed 2026-08-28, written in 30ee9e2 before the C1/I1-I4 and BR-2..BR-5 fix commits. Re-measure at the real close rather than leaving a number that understates by the whole review cost.
- **BR-9** [Minor] `comment-contract-drift` README never states that -locale is English-only, three lines above the new -lang example
  README.md:37 shows `define -locale gb colour`; D2's rule (honoured for English only, with a diagnostic otherwise) is documented in atlas/define.md:1084-1091 and nowhere in README. Belongs to the same sweep as the other finding in this family.
- **BR-10** [Minor] `comment-contract-drift` The PATTERNS explanation at store/yaml.go:75-87 is a detached comment godoc attaches to nothing
  Blank line before it and after it, so it documents neither RuntimeFiles nor wordsDir. Merge it into the RuntimeFiles doc comment. Same for store/migrate.go:11, whose opening line says MigrateToLanguages moves a DIRECTORY when it moves two named artifacts.

## Round 5 — 2026-08-28T13:18:45-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — Rule stated and mechanised for Go; ratchet verified reachable by planting a spelling in reflect.go. Residual prose scope raised as a new finding.
- BR-7 — addressed — Mutation-verified twice on scratch copies: tmpPattern and userModelPrefix changes each redden TestGitignoreCoversRuntimeFiles by name.
- BR-8 — addressed — 2.15h now recorded; consistent with sdlc actual at HEAD (7.05h) minus the 4.92h pre-implementation baseline the issue Log documents.
- BR-9 — addressed — README.md:36 marks the example English-only and a new paragraph states the rule.
- BR-10 — addressed — PATTERNS comment merged into the RuntimeFiles doc; MigrateToLanguages' doc names the two artifacts it moves.

### Raised

- **BR-11** [Important] `comment-contract-drift` BR-6's rule binds prose but is enforced over *.go only, and the hand-swept half left a live false claim in the project file
  6th finding in this family — do NOT fix the instance. The rule is already
  written and correct; what is missing is that its enforcement stops at
  `git ls-files '*.go'`, so the half the rule was actually raised about
  (README/atlas/plan lines) is still swept by hand. Measured at HEAD:
  README.md and atlas/ are clean, but workshop/projects/define-learn.md
  carries 6 live restatements (lines 5, 61, 140, 316, 349, 360, 494) plus 2
  historical ones, and :140 "A single user-model.md, batch-generated,
  human-correctable" and :494 "One user-model.md" are now FALSE — BR-2 made
  the model one file per language. :5 is the project's done_when frontmatter.
  A second live instance of the same family, pre-existing rather than
  introduced here: store/yaml.go:478-488 runs Forget's doc comment straight
  into the newsFile comment with no blank line, so godoc attaches "Forget
  removes one word file" to `type newsFile` — the same shape as BR-10, in
  the file BR-10 named. Class fix: extend the ratchet to markdown with an
  explicit allowlist (workshop/history/, the *-gate.md / *-review.md
  ledgers, and "## Revisions" sections, which legitimately record what was
  once true), and state that scope in the rule. For the orphaned comment the
  mechanical form is one check over top-level decls — a doc comment whose
  first word is not the declared identifier is attached to the wrong thing.
- **BR-12** [Minor] `runtime-artifact-guard-coverage` Two test fixtures/paths are hand-restated outside the store package; one is now stranded at the pre-#23 flat layout
  3rd finding in this family — do NOT fix the words/en/ instance. The rule is
  the family's own: a path or name a test asserts against must come from the
  function that produces it. store/yaml_test.go:45 still plants
  ".tmp-halfwritten" at dir/words/, while wordsDir() is now words/en/, so
  TestYAMLIgnoresInterruptedWrites no longer exercises Deck()'s temp-file
  skip at all — the sibling test 25 lines below had its path corrected with a
  comment about exactly this. Honest caveat: removing Deck()'s .yaml suffix
  check leaves the whole suite green at BOTH base and HEAD (the fixture's
  unparseable body masks it), so this is a pre-existing weak pin that this
  range made structurally unreachable, not a regression in detection.
  Second instance: cmd/define/lang_cmd_test.go:180 restates "lang.txt" in a
  NEGATIVE assertion, which passes vacuously after a rename (:159's positive
  one would redden). store.UserModelName was exported so consumers could
  derive; there is no equivalent for the language file. Class fix: move the
  temp-file test into package store (internal, as lang_test.go already is) so
  it plants via y.wordsDir() and newTempFile(), and export a producer for the
  language filename so package main's tests derive it too.

## Round 6 — 2026-08-28T14:14:26-07:00 (claude) — BLOCKED

### Disposed

- BR-11 — addressed — TestProseDoesNotSpellStaleRuntimeArtifactNames binds README/atlas/workshop/projects with a currentTruthOnly shape-based record filter; I planted the exact false sentence at define-learn.md:150 and it went red, and appending it after "## Log" correctly did not. Forget's doc comment is reattached at store/yaml.go:534. Residual symbol-half scope raised as a new finding.
- BR-12 — addressed — TestDeckIgnoresInterruptedWrites moved to package store (atomic_internal_test.go) deriving paths from y.wordsDir() and newTempFile(); reverting Deck()'s .yaml suffix check turns it red with the half-written word in the deck. store.LangFileName() exported and used at lang_cmd_test.go:159 and :182.

### Raised

- **BR-13** [Important] `language-derived-state-unscoped` applyLang does not re-derive d.usage, so D6's news gate holds at the boundary but not across a mid-session /lang
  2nd finding in this family — do NOT fix the instance by adding one line to applyLang. newsFeedFor is applied only in openStore (main.go:316) and sessionUsage (main.go:250), while applyLang (command.go:388) excludes d.usage as "not language-scoped" — true before this range, false after M2 made the feed's presence a function of the language. Verified with a scratch test: after applyLang(&d,&opt,"es",...) the session still holds the English cachingFeed (bs2.news != nil); symmetrically a session started in es keeps news==nil after /lang en and silently loses the feed. Latent rather than live only because UsageSource.Usages has no production caller at HEAD (grep: d.usage is referenced only at main.go:182-183); it becomes silent wrong-language data the moment #10 wires it. The class: applyLang's own doc states the generating rule ("anything derived from the language BEFORE a switch must be re-derived BY it") and the very next member added violated it, so an enumeration in a comment is not enforcement. Structural fix: extend the single newDeck builder to construct everything that is a function of the language (including the gated usage source) and have applyLang call exactly that builder, so a boundary-only derivation is unspellable. Pin at vocab_test.go:405, which already asserts d.history identity across the switch.
- **BR-14** [Important] `comment-contract-drift` The artifact-name rule's SYMBOL/MODEL half is still unenforced; 11 live restatements measured at HEAD
  7th finding in this family — do NOT fix these instances. Both ratchets count exactly one string, "user-model.", while the rule they enforce binds "a runtime artifact's filename OR A SYMBOL THE CODE OWNS". The unmechanised half has now recurred four times (I4 deckDeps, BR-6 MigrateFlatDeck, and two fresh ones). Measured at HEAD - (1) dict_darwin.go:296 "the nine symbols this file resolves"; (2) dict_conformance_test.go:69 "Nine undocumented symbols"; (3) atlas/define.md:1145; (4) workshop/projects/define-learn.md:464 - dcs_resolve resolves THREE. (5) vocab.go:160 claims warnTo is "the one place" the "define: " prefix is written while dict_darwin.go:359 is a byte-identical second (ARCH-DRY; also a package-level func inside a darwin-only build tag). (6) README.md:331-333 is a stranded pre-M2 paragraph duplicating the one four lines above and falsely saying enabling Chinese dictionaries affects formatting, which the curated L to L selection now prevents. (7) atlas/index.md:9 still says "NOAD word lookup". (8) plan :116/:132 name dictChoice, absent. (9) plan :147/:153 name dcsDictionaries, absent. (10) plan :279 ticked, "assert all nine symbols resolve". (11) plan D6's Bonus claims a live Google News request per Spanish lookup, which no code path makes. Class fix, cheap because the enumeration already exists - one test asserting every Name cell of a plan's Core-concepts table resolves to a declared identifier at the stated path would have caught 8, 9 and both prior recurrences; and give the symbol COUNT one producer (a dcsSymbols slice the resolver and the docs both read) instead of four hand-typed copies.
- **BR-15** [Important] `policy-inside-io-shell` systemDictionary's two fallback branches have no automated test on any platform, and the Done-when row is ticked on a manual experiment
  dict_darwin.go:327 fuses the three-outcome policy and its user-facing warning text to installedDictionaries()' cgo IO, so neither "the private surface is gone" nor "nothing curated matches" is reachable from a test. noadDictionary and everyActiveDictionary appear in no non-conformance test. The Done-when row "the seam FALLS BACK to today's NULL behaviour if any is missing" and plan Task 9 Step 4 are both ticked on a manual symbol-misspelling run. ARCH-PURE (extract dictionaryFor(installed []dictMeta, lang) (ids []string, name string) as pure and leave the cgo shell thin) and ARCH-MOCK (with the metadata source injected, production and test finally share the selection boundary — the standard the M1 review sidecar set at 000023-deck-language-m1-review.md:289 and this milestone did not meet). Not hypothetical: I measured this machine's shell context returning a single dictionary (com.apple.dictionary.Wikipedia), so ./define -lang es mesa takes the untested branch on every run here.
- **BR-16** [Important] `atlas-lags-new-surface` atlas/repo-guards.md does not name the two artifact-name ratchets this range added
  repo-guards.md is the atlas catalogue of repo guards and was updated for RuntimeFiles, TestRuntimeFilePatternsCoverWhatWeWrite and legacyRuntimeFilePaths, but never names TestRuntimeArtifactNamesAreSpelledOnceInSource or TestProseDoesNotSpellStaleRuntimeArtifactNames — the two guards workshop/lessons.md calls this range's class fix, and the ones a contributor most needs to find before adding a doc line. One table row each plus the records-versus-current-truth scope rule (## Revisions / ## Log / a block carrying **closed:**), so the deliberate scope decision is discoverable outside the test's own comment.
- **BR-17** [Minor] `comment-contract-drift` selectedDictionary.Lookup lets a later ErrNoEntry overwrite an earlier "dictionary unavailable", the collapse its own C comment forbids
  dict_darwin.go:270 assigns lastErr on every iteration, so status 3 on NOAD followed by status 1 on AppleDictionary reports ErrNoEntry — "this word does not exist in English" when the truth is "the primary dictionary vanished". dcs_lookup_in's comment at :118 states these two must not collapse. Keep the first non-ErrNoEntry error instead of the last error seen.
- **BR-18** [Minor] `policy-inside-io-shell` installedDictionaries indexes position i across N separately-copied CFSets, whose enumeration order the file itself calls unspecified
  dict_darwin.go:189 calls dcs_describe(i) once per dictionary and each call re-invokes f_copy_available() and CFSetGetValues on a fresh copy. dcs_describe's own comment states iteration order is UNSPECIFIED; if two copies ever enumerate differently the returned slice gains duplicates and drops entries, and a dropped com.apple.dictionary.es.DGLEV silently degrades Spanish to the NULL search. Copy the set once and describe all N from that copy. Performance is not the concern (measured 46us warm).
- **BR-19** [Minor] `policy-inside-io-shell` The English fixture corpus is captured through the NULL search while production selects the curated identifiers
  testdata/capture.sh:60 captures entries/en through DCSCopyTextDefinition(NULL) — "the host's ACTIVE dictionaries" — while systemDictionary(en) now selects com.apple.dictionary.NOAD and com.apple.dictionary.AppleDictionary. Capture path and production path therefore differ for English; they agree today only because this host's active set happens to match. Detectable (TestFixturesMatchLiveDictionary compares them, and did go red once), so this is a note rather than a defect — but passing the curated ids to capture.py would make the two agree by construction (ARCH-MOCK).

## Round 7 — 2026-08-28T14:44:27-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — addressed — Verified by mutation — dropping d.usage from applyLang's adoption reddens vocab_test.go:463 in both directions. See new finding on the residual hand-enumeration.
- BR-14 — not-addressed — Class guard landed but has a comment-admitting loophole and three of the eleven measured instances are still live in the plan.
- BR-15 — addressed — dictionaryFor is pure and all three outcomes plus the empty-set case are unit-tested.
- BR-16 — addressed — Both ratchets are now table rows in atlas/repo-guards.md with the records-vs-current-truth scope rule.
- BR-17 — not-addressed — The fix does not fire — case 1 still assigns lastErr unconditionally, so status 3 then status 1 still returns ErrNoEntry.
- BR-18 — addressed — dcs_describe_all copies the set once and describes all N in a single pass.
- BR-19 — addressed — capture.sh walks EN_DICTS in curated order; capture.py exits 1 on no-entry so the first-hit-wins loop matches selectedDictionary.Lookup.

### Raised

- **BR-20** [Important] `comment-contract-drift` TestPlanTablesNameEntitiesThatExist passes on a COMMENT mention, and a stale `newDeck` row proves the hole
  9th finding in this family — do NOT fix the newDeck row alone. repo_guard_test.go:595 accepts
  `strings.Contains(string(src), name+" ")` as evidence a symbol is declared, so any mention anywhere
  in the file satisfies it. The plan's Core-concepts row `newDeck` at cmd/define/main.go
  (plan:144) is stale — the tree renamed it to newLangDeps in this very commit — and the guard is
  green only because main.go:269 still says "newDeck stays nil on both of these paths". Mutation
  proof: rewording that one comment turns the test RED naming newDeck; removing the fallback clause
  flags exactly that one row across all active plans and nothing else, so no legitimate row needs it.
  The rule the family keeps failing: a guard may not accept prose as evidence about code. Delete the
  fallback (declared/assigned already cover every current row), then sweep newDeck from plan:144,
  :61, :84, :191, :196, main.go:52, :69, :269 and atlas/define.md:465.
- **BR-21** [Important] `atlas-lags-new-surface` atlas/define.md still describes the pre-BR-13 /lang design, including the exact claim BR-13 disproved
  2nd finding in this family — the rule is that the atlas is updated in the SAME commit as the
  surface it maps, not the same range. atlas/define.md:463-476 says the rebuild goes through
  `newDeck(lang)` (renamed to newLangDeps), that "usage reads usage/, so re-deriving them would put
  a second, unloaded History beside the one runEditor already Load()ed" — false, usage IS re-derived
  now and only history is not — and enumerates applyLang's members as d.lang, opt.voice, the deck
  triple and voc, with usage absent. The commit edited atlas/define.md but only around line 1142.
  The one current-truth document describing this invariant now contradicts the code on it.
- **BR-22** [Important] `language-derived-state-unscoped` langDeps is adopted field-by-field at two sites, and d.dict — a member since M1 — has no test at all
  3rd finding in this family — do NOT fix by adding one assertion. The rule the fixes keep half-applying
  is: every member of the language-derived set must be (a) adopted as a WHOLE from the one builder and
  (b) pinned by a test that reddens when its re-derivation is removed. Neither half holds. Adoption is
  `d.deck, d.capture, d.vocab, d.usage = ld.deck, ld.capture, ld.vocab, ld.usage` (command.go:388) and
  the same four enumerated again into storeDeps (main.go:325-329), so a fifth langDeps field is
  forgettable at both sites — embedding langDeps in deps/storeDeps makes it one assignment and closes
  (a). For (b): d.newDict is set at ZERO call sites in any test, so deleting BOTH the boundary
  derivation (main.go:597-599) and the /lang re-derivation (command.go:388-390) leaves the whole
  cmd/define suite green (measured, 94s) while production would dereference a nil Dictionary. That is
  the milestone's headline Done-when row, and it is the same class as C1 and BR-13.
- **BR-23** [Important] `policy-inside-io-shell` Two pieces of pure logic still live inside the darwin cgo shell; one shipped an inoperative fix, the other broke GOOS=linux
  4th finding in this family — do NOT fix either instance alone. The rule: nothing that is a decision
  or a parse over data may live inside dict_darwin.go's build-tagged shell; dictselect.go is where it
  goes. The enumeration for this file is three items and only one is done. (1) selection policy →
  dictionaryFor, extracted ✓. (2) status-folding policy in selectedDictionary.Lookup — still inline,
  untestable, and consequently BR-17's fix shipped broken with no test to catch it. Extract
  foldLookupStatuses([]int, []string) error as pure. (3) parseDictRecords / parseLangPairs / baseLang —
  their own comments say "Pure, so the whole cgo boundary's format is testable without CoreServices",
  but they sit behind //go:build darwin while TestParseLangPairs is in the untagged dict_fake_test.go:203.
  Measured: `GOOS=linux go vet ./cmd/define/` now fails with `undefined: parseLangPairs`; it exited 0 at
  base 2e929fc5 (checked in a worktree), so this range regressed it, and dict_stub.go:13 still claims
  the stub keeps `go vet ./...` green off darwin.
- **BR-24** [Important] `runtime-artifact-guard-coverage` dcsPrivateSymbols is documented as the ONE producer, but dcs_resolve hand-writes the same three names in C
  4th finding in this family — do NOT fix by syncing the two lists. Identical shape to BR-7: a guard
  asserting coverage from a hand-typed restatement. dict_darwin.go:200 declares the slice and its doc
  says "ONE producer for the list", while dcs_resolve at dict_darwin.go:43-45 spells
  DCSCopyAvailableDictionaries / DCSDictionaryGetIdentifier / DCSDictionaryGetLanguages again as C
  string literals. TestPrivateDictionarySurfaceStillResolves walks the Go copy, so adding or renaming a
  dlsym in the C preamble leaves the conformance check reporting "all present" while the resolver needs
  a symbol nobody verifies — and this check IS the whole mitigation for M2's stated OS-version risk.
  The rule: a list the code owns has one producer, and where a second language forces a restatement, a
  ratchet asserts the two agree. Cheap here — one test regexing dlsym("...") out of the file's own
  source and comparing the set to dcsPrivateSymbols, in the same shape as the existing ratchets.
- **BR-25** [Minor] `comment-contract-drift` Two doc comments stranded onto the wrong declaration by this commit's insertions
  main.go:127-135 — storeDeps' doc ("the trio openStore produces") now runs into `type langDeps` with
  no blank line, so godoc attaches it to langDeps and storeDeps is undocumented. main.go:212-231 —
  openStore's doc runs into `func newsFeedFor` the same way, leaving openStore undocumented. Both
  declarations were inserted directly beneath an existing doc comment. Same shape as BR-10, twice, in
  the commit that closed it; a ratchet asserting a doc comment's first word matches the declaration it
  precedes is the mechanical form.
- **BR-26** [Minor] `comment-contract-drift` --help still says define looks words up in "macOS's active dictionaries", which is now the fallback path
  main.go:435. After M2 the normal path is the curated per-language selection and the whole-active-set
  search is the degradation. README.md:314 was updated for this; the binary's own usage text, which the
  same commit edited to add the /lang paragraph, was not.
- **BR-27** [Minor] `comment-contract-drift` The private-surface conformance check routes SHAPE drift through SkipOrFail, so "says loudly" is a skip by default
  dict_conformance_test.go:91. internal/conformance's own four-class rule puts an absent external
  dependency in SkipOrFail and "the dependency's surface moved" under SHAPE drift, which "ALWAYS
  fail[s] ... the very thing these suites exist to report". A vanished private symbol is the second
  class, and the issue's Done-when row asks it to say so LOUDLY. Under a plain
  `go test -tags conformance` it prints a skip, which the package doc itself says reads as green.

## Open findings

- **BR-14** [Important] `comment-contract-drift` The artifact-name rule's SYMBOL/MODEL half is still unenforced; 11 live restatements measured at HEAD
- **BR-17** [Minor] `comment-contract-drift` selectedDictionary.Lookup lets a later ErrNoEntry overwrite an earlier "dictionary unavailable", the collapse its own C comment forbids
- **BR-20** [Important] `comment-contract-drift` TestPlanTablesNameEntitiesThatExist passes on a COMMENT mention, and a stale `newDeck` row proves the hole
- **BR-21** [Important] `atlas-lags-new-surface` atlas/define.md still describes the pre-BR-13 /lang design, including the exact claim BR-13 disproved
- **BR-22** [Important] `language-derived-state-unscoped` langDeps is adopted field-by-field at two sites, and d.dict — a member since M1 — has no test at all
- **BR-23** [Important] `policy-inside-io-shell` Two pieces of pure logic still live inside the darwin cgo shell; one shipped an inoperative fix, the other broke GOOS=linux
- **BR-24** [Important] `runtime-artifact-guard-coverage` dcsPrivateSymbols is documented as the ONE producer, but dcs_resolve hand-writes the same three names in C
- **BR-25** [Minor] `comment-contract-drift` Two doc comments stranded onto the wrong declaration by this commit's insertions
- **BR-26** [Minor] `comment-contract-drift` --help still says define looks words up in "macOS's active dictionaries", which is now the fallback path
- **BR-27** [Minor] `comment-contract-drift` The private-surface conformance check routes SHAPE drift through SkipOrFail, so "says loudly" is a skip by default
