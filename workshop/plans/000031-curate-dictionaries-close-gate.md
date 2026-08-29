---
gate: boundary-review
issue: 31
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-29T12:32:30-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: the -lang flag help still enumerates "en, es" and does not derive from curated
          detail: |-
            cmd/define/main.go:408 hardcodes "language for this invocation: en, es"; the built
            binary prints it, so `define -h` denies a language the README now documents. This is
            the only curated consumer that is wrong rather than merely unpinned. Build a langHelp
            constant from slices.Sorted(maps.Keys(curated)), following the pronHelp/localeHelp
            pattern the repo already owns (ARCH-PURPOSE, ARCH-DRY).
          family: curated-consumer-unpinned
          round: 1
        - id: BR-2
          severity: Important
          title: TestDocsNameEveryCuratedLanguage pins README only; the atlas table this diff added is unchecked
          detail: |-
            dictselect_test.go:470 reads only ../../README.md, while atlas/define.md:1271 gained a
            hand-maintained "Three curated languages" table in the same commit. Its two model tests,
            TestDocsQuoteThePronHelp and TestDocsQuoteTheLocaleHelp, both loop over README AND atlas
            with a comment naming this exact half-fix. Third unpinned site: the own-language table at
            dict_conformance_test.go:190 has no guard requiring a curated language to have a row.
          family: curated-consumer-unpinned
          round: 1
        - id: BR-3
          severity: Important
          title: the Italian test was inserted inside the Spanish test's doc comment, so both are misdocumented
          detail: |-
            render_test.go:335-355 ("A Spanish entry carries NO pronunciation notation… orthography is
            phonemic…") is now the godoc of TestItalianEntriesCarryNoPronunciationNotation at :358 and
            contradicts the Italian paragraph appended to it. TestSpanishEntriesCarryNoPronunciationNotation
            at :383 keeps only the orphaned "SCOPED DELIBERATELY… the claim above" tail, whose antecedent
            is now the Italian test. Move the Italian function and its own comment below the Spanish one.
          family: doc-comment-misattached
          round: 1
        - id: BR-4
          severity: Important
          title: the Italian no-notation test is a verbatim second spelling of the Spanish one
          detail: |-
            render_test.go:358-376 and :383-398 share the same empty guard, the same redundant
            d.Lookup(word) over a map whose value is already raw, and the same ParseEntry(raw).IPA
            assertion. Task 3 generalised the sibling conformance check to a table in this very
            boundary while warning that this family's second instance lands in the commit that fixes
            the first. Consolidate to noNotationIn(t, lang, why) driven by a two-row table (ARCH-DRY).
          family: one-predicate-two-spellings
          round: 1
        - id: BR-5
          severity: Important
          title: the strong no-data-loss invariant is still English-only; widening it is measured green
          detail: |-
            TestRenderLosesNothing (invariant_test.go:161) does the alnum-count + subsequenceGap check
            and takes testDict(t). Italian gets only TestRenderLosesNothingInEveryCapturedLanguage,
            which asserts a non-empty render and would pass while most of essere's 11.6KB was dropped —
            and the Devoto-Oli shapes (• sub-senses, A./B. blocks, ETIMOLOGIA/DATA) are exactly where
            loss is likely. Verified at HEAD: the strong invariant passes over all captured languages,
            so widening it costs ~6 lines and is what Done-when 3 reads as.
          family: check-narrower-than-corpus
          round: 1
        - id: BR-6
          severity: Minor
          title: the README guard is free-text containment inside the span, so a broken bullet label survives
          detail: |-
            Verified: renaming the bullet "- **Italian** —" to "- **Itaian** —" keeps
            TestDocsNameEveryCuratedLanguage green, because a later sentence in the same bullet says
            "Italian". Narrower than whole-file containment, but the same class the test's own comment
            condemns. Anchor on the bold label rather than anywhere in the span.
          family: curated-consumer-unpinned
          round: 1
        - id: BR-7
          severity: Minor
          title: TestEveryCuratedLanguageHasACorpus re-globs instead of calling loadFakeDictionary
          detail: |-
            dictselect_test.go:431 duplicates the corpus glob. loadFakeDictionary("testdata/entries", lang)
            already answers the same question and additionally rejects zero-byte fixtures, with a message
            that names capture.sh.
          family: one-predicate-two-spellings
          round: 1
        - id: BR-8
          severity: Minor
          title: atlas/define.md:1301 has no blank line before the "## Source pronunciation" heading
          family: doc-formatting
          round: 1
        - id: BR-9
          severity: Minor
          title: Task 5's Files line names doc_sync_test.go and dict_fake_test.go, neither of which changed
          detail: |-
            Both new guards landed in cmd/define/dictselect_test.go. Separately, Task 5's first bullet is
            ticked while instructing a comment fix in capture.sh and rawnotation_test.go that was
            correctly not performed (widening made both comments true). The Core concepts table itself
            matches the code; only the per-task Files line and that bullet need a Revisions entry.
          family: plan-table-drifts-from-code
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-29T13:00:13-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: langHelp derives from slices.Sorted(maps.Keys(curated)); built binary prints "en, es, it". No test pins the registration — folded into the new enumeration as site 6.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Atlas half fixed and verified red on mutation; the third site it named (dict_conformance_test.go:184 own-language table, no curated cross-check) is untouched.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Consolidated into one table at render_test.go:364; both comments now describe what they sit above.
          round: 2
        - id: BR-4
          disposition: addressed
          note: One predicate, two rows, per-language why preserved; verified red on an injected IPA-bearing it fixture.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Verified by mutation — dropping an Italian-only marker in Render reddens only TestRenderLosesNothing/it/*.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: Still strings.Contains(span, name); reproduced green with the atlas row renamed to "Italiano".
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: dictselect_test.go:428 still re-globs rather than calling loadFakeDictionary.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: atlas/define.md:1303 still has no blank line before the heading; the same insertion also left a double blank line at :1269.
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: No Revisions entry landed; plan:148 Files line unchanged; langHelp also has no Core-concepts row.
          round: 2
      findings:
        - id: BR-10
          severity: Important
          title: the class was declared closed in 19b5ea1 but the enumeration was never written; six sites remain, two wrong today
          detail: |-
            4th round of this family. Do NOT patch a sixth instance — the rule is already in
            workshop/lessons.md:2390 and :2176 and fired anyway. Rule to enforce: every per-language
            enumeration ranges over curated, and every language-keyed table is cross-checked against
            curated in both directions. Measured enumeration at HEAD: (1) dict_conformance_test.go:184
            own-language table, no cross-check; (2) dictselect_test.go:157 hardcodes {"en","es"} —
            changing it to range over curated fails today with "it: no choice from the measured set";
            (3) dictselect_test.go:23 installedOnThisMachine() never gained the Italian books #31
            measured as installed, so dictselect_test.go:399 re-declares them locally (ARCH-MOCK,
            ARCH-DRY) — this is what blocks site 2; (4) atlas/define.md:1271 "Three curated languages"
            sits four lines ABOVE the guarded span, exactly as round 1 predicted; (5) atlas/define.md:1297
            says es and it are "six fixtures each" — entries/es holds FIVE, wrong today; (6) main.go:408's
            flag registration is unpinned, so reverting langHelp to a literal is invisible to go test.
            Sites 4 and 5 are answered by naming the list rather than counting it. ARCH-PURPOSE.
          family: curated-consumer-unpinned
          round: 2
        - id: BR-11
          severity: Minor
          title: the widened strong invariant subsumes TestRenderLosesNothingInEveryCapturedLanguage, whose comment is now false
          detail: |-
            3rd in family. Do NOT just delete this one — the rule: when a check is widened to the
            dimension its narrower sibling existed to cover, the sibling is deleted in the SAME commit.
            dict_fake_test.go:297 asserts only a non-empty render over capturedLanguages, which
            TestRenderLosesNothing (invariant_test.go:169) now covers with exact alnum counts over the
            same set, so it can no longer fail first; and its doc comment at :290 still claims
            TestRenderLosesNothing "goes through testDict(t), which is English by definition, so the
            Spanish captures were never run through the parser and renderer at all", which this round
            made false. Enumerated at HEAD: this is the only site.
          family: one-predicate-two-spellings
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-29T13:26:29-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: TestDocsNameEveryCuratedLanguage now loops README + atlas (verified red on the atlas row), and the own-language table gained its cross-check (verified red).
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: 'Measured at HEAD: renaming the README bullet label to "- **Itaian** —" still passes, because the same bullet''s next sentence says "Italian"; the word-boundary change only closed the "Italiano" near-miss.'
          round: 3
        - id: BR-7
          disposition: addressed
          note: TestEveryCuratedLanguageHasACorpus now goes through loadFakeDictionary, which also rejects zero-byte fixtures.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Blank line present before "## Source pronunciation"; a new double-blank appeared four lines earlier (raised as doc-formatting).
          round: 3
        - id: BR-9
          disposition: addressed
          note: Task 5's Files line now names the files that actually changed, and the Revisions entry records why the comment fix was not performed.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: Five of six sites verified red on removal; site 6 is half-closed — replacing langHelp's body with the stale literal "en, es" and dropping the three imports leaves go test ./... fully green (measured), which is the exact regression the finding described.
          round: 3
        - id: BR-11
          disposition: addressed
          note: The subsumed test is deleted with a tombstone, and TestRenderLosesNothing verifiably emits en/es/it subtests.
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: the rename/deletion this round performed never got its retiredSymbolNames row, so the atlas and the plan name tests the tree does not declare
          detail: |-
            atlas/define.md:1181 names TestSpanishEntriesCarryNoPronunciationNotation (renamed to
            TestNonEnglishEntriesCarryNoPronunciationNotation); the plan names it plus the deleted
            TestRenderLosesNothingInEveryCapturedLanguage and a TestItalianEntriesCarryNoPronunciationNotation
            that never existed. Measured: adding the two rows to retiredSymbolNames
            (repo_guard_test.go:719) turns TestNoArtifactNamesARetiredSymbol red on atlas/define.md and
            twice on the plan. The map's own comment records this human half failing once before (#27),
            so the rule to build is mechanical: derive removed func declarations from the review window
            with git — TestPlanTableStatusMatchesTheChangeWindow already shells to git in this file — and
            require either a retiredSymbolNames row or zero current-truth mentions.
          family: retired-symbol-unswept
          round: 3
        - id: BR-13
          severity: Important
          title: the class is still enforced per-site; two hand-written enumerations remain and the mechanism should be one surface registry
          detail: |-
            This is the 5th finding in this family, so do NOT patch the two sites. Shadow-sweep at HEAD:
            nine consumers of curated, seven enforced. Unenforced — (a) the {es,it} table in
            TestNonEnglishEntriesCarryNoPronunciationNotation (render_test.go:366), which is a
            language-keyed table with no cross-check, so #34 can curate fr and gain no row; (b) the doc
            list {README, atlas} spelled three times (dictselect_test.go:515, doc_sync_test.go:125, :142),
            so a third doc is covered only by whichever test its author remembered. Rule: curated gets ONE
            registry of surfaces obliged to name every curated language, and each row supplies the
            surface's TEXT (README span, atlas span, langHelp) so the assertion is "every curated language
            appears in that text" — which makes replacing a derivation with a literal fail, the gap BR-10
            leaves open. Language-keyed predicate tables get a cross-check with an EXPLICIT exempt list,
            because German's Duden field is real and an IPA=="" row for de would be wrong; today "no row"
            and "deliberately no row" are indistinguishable. ARCH-PURPOSE, ARCH-DRY.
          family: curated-consumer-unpinned
          round: 3
        - id: BR-14
          severity: Minor
          title: the per-captured-language sweep harness is now spelled three times, two of them new in this diff
          detail: |-
            This is the 4th finding in this family, so the fix is the helper, not the site. Measured
            prevalence: invariant_test.go:170, render_test.go:242, render_test.go:326 all read
            `for lang := range capturedLanguages(t) { t.Run(string(lang), func(t){ xIn(t, testDictFor(t, lang)) }) }`.
            One forEachCapturedLanguage(t, func(*testing.T, *fakeDictionary)) collapses all three, and a
            fourth sweep then cannot be written narrower than the corpus by accident.
          family: one-predicate-two-spellings
          round: 3
        - id: BR-15
          severity: Minor
          title: atlas/define.md:1269-1270 has a double blank line and a raggedly re-wrapped paragraph
          detail: |-
            This is the 2nd finding in this family (BR-8 was a missing blank line before a heading), so
            state the rule rather than patching the line: nothing in the repo checks markdown block
            spacing or wrap in atlas/ and README.md, and both instances were introduced by hand-editing
            prose. Either run a markdown formatter over the touched docs as part of the boundary, or
            accept the class explicitly. Also at :1288 the ragged rewrap left "…held five. The live /
            ratchet in live_property_test.go…".
          family: doc-formatting
          round: 3
        - id: BR-16
          severity: Minor
          title: dictselect_test.go:475 wraps a single t.Errorf in a bare block instead of inverting the guard
          detail: |-
            `if err == nil && len(d.entries) > 0 { return }` followed by `{ t.Errorf(...) }` reads as
            `if err != nil || len(d.entries) == 0 { t.Errorf(...) }`.
          family: redundant-syntax
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-29T13:54:07-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: not-addressed
          note: 'Re-measured at HEAD: renaming the README bullet to "- **Itaian** —" leaves TestEverySurfaceNamesEveryCuratedLanguage green, because the same bullet''s next sentence says "Italian". Minor, non-blocking.'
          round: 4
        - id: BR-10
          disposition: addressed
          note: 'Site 6 closed by the registry row: replacing langHelp''s body with the literal "en, es" now reddens TestEverySurfaceNamesEveryCuratedLanguage/the -lang flag help (verified by revert in a scratch clone).'
          round: 4
        - id: BR-12
          disposition: addressed
          note: 'Rows added and all three artifacts swept (incl. workshop/issues/000030). Mechanism verified live: removing the two rows and re-injecting a stale mention turns TestARemovedDeclarationIsSweptOrRetired red. Its matching rule is defective — raised separately.'
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: The registry (part a) is delivered and real; part (b) is not — derivedDocs at dictselect_test.go:487 has zero references while doc_sync_test.go:125 and :142 still spell the doc list literally.
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: 'Bodies were extracted but the harness is still spelled three times verbatim: invariant_test.go:170, render_test.go:243, render_test.go:327. No forEachCapturedLanguage helper exists.'
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: Instances mostly repaired (double blank gone), but no rule was stated and none accepted; atlas/define.md:1273 (94 cols) and :1303 (87) still break the file's ~80-col wrap.
          round: 4
        - id: BR-16
          disposition: addressed
          note: dictselect_test.go:472 now reads `if err != nil || len(d.entries) == 0 {`.
          round: 4
      findings:
        - id: BR-17
          severity: Important
          title: the new removed-declaration guard matches by substring, so an ordinary rename fires seven false failures and its suggested remedy poisons the sibling guard
          detail: |-
            repo_guard_test.go:1120 uses strings.Contains, while TestNoArtifactNamesARetiredSymbol:791
            uses a word-boundary regex for the same question — the two guards disagree on the matching
            rule that currentTruthFiles' own comment says they cannot disagree about. Measured in a
            scratch clone: renaming the helper `func ids` to `idsOfMetas` and committing turns the guard
            red on seven paths including atlas/define.md, which has ZERO word-boundary occurrences of
            "ids" (the hit is "forbids"). The message then tells the author to add a retiredSymbolNames
            row for "ids", which would make TestNoArtifactNamesARetiredSymbol permanently red on every
            file containing that word. Fix: share the word-boundary regex, and bound `gone` to names
            unlikely to collide (Test*/exported, or a length floor).
          family: retired-symbol-unswept
          round: 4
        - id: BR-18
          severity: Important
          title: both helpers introduced to collapse duplication have zero and one call site while the spellings they replace remain, under comments claiming the consolidation happened
          detail: |-
            This is the 5th finding in this family, so the fix is the rule, not the two sites: a
            consolidation is complete only when the new helper's call-site count equals the number of
            spellings it replaced — grep for callers before disposing the finding. Measured at HEAD:
            (a) derivedDocs (dictselect_test.go:487, commented "named ONCE") has zero references
            anywhere; doc_sync_test.go:125 and :142 still spell the two-doc list literally and
            curatedSurfaces() spells each path separately at :538/:543 — package-level vars escape Go's
            unused check, so it passes every suite while doing nothing. (b) currentTruthFiles
            (repo_guard_test.go:1134, commented "Shared with TestNoArtifactNamesARetiredSymbol so the
            two guards cannot disagree") has one caller; TestNoArtifactNamesARetiredSymbol:759-780
            keeps a byte-identical binds closure and ls-files loop, and the two copies HAVE ALREADY
            DIVERGED — the inline one Fatals on `seen == 0`, the extracted one has no vacuity
            assertion, so the new guard would pass over an empty file set. ARCH-DRY.
          family: one-predicate-two-spellings
          round: 4
        - id: BR-19
          severity: Important
          title: the own-language table's curated cross-check is pure data but sits behind the conformance build tag, so it never runs in go test ./... or CI
          detail: |-
            This is the 6th finding in this family, so state the rule rather than adding a row: the PURE
            half of a cross-check belongs in an untagged file; only the live lookup half needs the tag.
            dict_conformance_test.go:202's `for lang := range curated { … no row for it … }` compares two
            in-memory tables and needs no dictionary, network or macOS, yet the file carries
            `//go:build darwin && conformance` — which its own header says a CI runner does not satisfy,
            and which .github/workflows/merge-check.yml does not pass. So the obligation "a curated
            language has a live own-language check" is enforced only for whoever remembers to run the
            tagged suite and read it, which is the per-site enforcement BR-13 set out to end. Fix: move
            the rows table and the cross-check to an untagged file beside the registry, leaving the
            t.Run lookup tagged. ARCH-PURPOSE.
          family: curated-consumer-unpinned
          round: 4
        - id: BR-20
          severity: Minor
          title: capture.sh's mkdir line and closing summary enumerate languages by hand and no registry row covers them
          detail: |-
            capture.sh:11 (`mkdir -p entries/en entries/es entries/it`) and :177 (the echo/wc summary)
            are per-language enumerations. TestCaptureScriptUsesTheCuratedDictionaries pins the
            IDENTIFIERS in both directions but not these, so a fourth language yields a summary that
            under-reports — the exact failure the comment added at :172 says the summary exists to
            prevent. A whole-file token check cannot work ("it" is an English word throughout the
            script), so this needs a marked span or an explicit exemption with its reason, the way
            notationExempt makes "deliberately no row" legible.
          family: curated-consumer-unpinned
          round: 4
        - id: BR-21
          severity: Minor
          title: dictselect_test.go:52 says the fixture is cross-checked "in BOTH directions" but only curated to fixture is implemented
          detail: |-
            TestTheMeasuredSetModelsEveryCuratedLanguage loops curated and asserts each id is modelled;
            nothing checks the reverse. The reverse is deliberately absent — the fixture carries
            bilingual and thesaurus decoys — so the comment should change, not the code. Same shape as
            the two false structural claims in the Important finding above ("named ONCE", "Shared
            with"), all three introduced by the rounds that were closing findings about exactly this.
          family: comment-overclaims-code
          round: 4
        - id: BR-22
          severity: Minor
          title: the removed-declaration guard's skip message says "top-level declaration" but the regex matches only ^-func
          detail: |-
            repo_guard_test.go:1111 reports "this window removed no top-level declaration" while the
            regex at :1093 matches `^-func` only, so a removed var/const/type is invisible. Narrow the
            message or widen the regex.
          family: retired-symbol-unswept
          round: 4
      blocked: false
---

# Gate ledger — tools#31 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T12:32:30-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `curated-consumer-unpinned` the -lang flag help still enumerates "en, es" and does not derive from curated
  cmd/define/main.go:408 hardcodes "language for this invocation: en, es"; the built
  binary prints it, so `define -h` denies a language the README now documents. This is
  the only curated consumer that is wrong rather than merely unpinned. Build a langHelp
  constant from slices.Sorted(maps.Keys(curated)), following the pronHelp/localeHelp
  pattern the repo already owns (ARCH-PURPOSE, ARCH-DRY).
- **BR-2** [Important] `curated-consumer-unpinned` TestDocsNameEveryCuratedLanguage pins README only; the atlas table this diff added is unchecked
  dictselect_test.go:470 reads only ../../README.md, while atlas/define.md:1271 gained a
  hand-maintained "Three curated languages" table in the same commit. Its two model tests,
  TestDocsQuoteThePronHelp and TestDocsQuoteTheLocaleHelp, both loop over README AND atlas
  with a comment naming this exact half-fix. Third unpinned site: the own-language table at
  dict_conformance_test.go:190 has no guard requiring a curated language to have a row.
- **BR-3** [Important] `doc-comment-misattached` the Italian test was inserted inside the Spanish test's doc comment, so both are misdocumented
  render_test.go:335-355 ("A Spanish entry carries NO pronunciation notation… orthography is
  phonemic…") is now the godoc of TestItalianEntriesCarryNoPronunciationNotation at :358 and
  contradicts the Italian paragraph appended to it. TestSpanishEntriesCarryNoPronunciationNotation
  at :383 keeps only the orphaned "SCOPED DELIBERATELY… the claim above" tail, whose antecedent
  is now the Italian test. Move the Italian function and its own comment below the Spanish one.
- **BR-4** [Important] `one-predicate-two-spellings` the Italian no-notation test is a verbatim second spelling of the Spanish one
  render_test.go:358-376 and :383-398 share the same empty guard, the same redundant
  d.Lookup(word) over a map whose value is already raw, and the same ParseEntry(raw).IPA
  assertion. Task 3 generalised the sibling conformance check to a table in this very
  boundary while warning that this family's second instance lands in the commit that fixes
  the first. Consolidate to noNotationIn(t, lang, why) driven by a two-row table (ARCH-DRY).
- **BR-5** [Important] `check-narrower-than-corpus` the strong no-data-loss invariant is still English-only; widening it is measured green
  TestRenderLosesNothing (invariant_test.go:161) does the alnum-count + subsequenceGap check
  and takes testDict(t). Italian gets only TestRenderLosesNothingInEveryCapturedLanguage,
  which asserts a non-empty render and would pass while most of essere's 11.6KB was dropped —
  and the Devoto-Oli shapes (• sub-senses, A./B. blocks, ETIMOLOGIA/DATA) are exactly where
  loss is likely. Verified at HEAD: the strong invariant passes over all captured languages,
  so widening it costs ~6 lines and is what Done-when 3 reads as.
- **BR-6** [Minor] `curated-consumer-unpinned` the README guard is free-text containment inside the span, so a broken bullet label survives
  Verified: renaming the bullet "- **Italian** —" to "- **Itaian** —" keeps
  TestDocsNameEveryCuratedLanguage green, because a later sentence in the same bullet says
  "Italian". Narrower than whole-file containment, but the same class the test's own comment
  condemns. Anchor on the bold label rather than anywhere in the span.
- **BR-7** [Minor] `one-predicate-two-spellings` TestEveryCuratedLanguageHasACorpus re-globs instead of calling loadFakeDictionary
  dictselect_test.go:431 duplicates the corpus glob. loadFakeDictionary("testdata/entries", lang)
  already answers the same question and additionally rejects zero-byte fixtures, with a message
  that names capture.sh.
- **BR-8** [Minor] `doc-formatting` atlas/define.md:1301 has no blank line before the "## Source pronunciation" heading
- **BR-9** [Minor] `plan-table-drifts-from-code` Task 5's Files line names doc_sync_test.go and dict_fake_test.go, neither of which changed
  Both new guards landed in cmd/define/dictselect_test.go. Separately, Task 5's first bullet is
  ticked while instructing a comment fix in capture.sh and rawnotation_test.go that was
  correctly not performed (widening made both comments true). The Core concepts table itself
  matches the code; only the per-task Files line and that bullet need a Revisions entry.

## Round 2 — 2026-08-29T13:00:13-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — langHelp derives from slices.Sorted(maps.Keys(curated)); built binary prints "en, es, it". No test pins the registration — folded into the new enumeration as site 6.
- BR-2 — not-addressed — Atlas half fixed and verified red on mutation; the third site it named (dict_conformance_test.go:184 own-language table, no curated cross-check) is untouched.
- BR-3 — addressed — Consolidated into one table at render_test.go:364; both comments now describe what they sit above.
- BR-4 — addressed — One predicate, two rows, per-language why preserved; verified red on an injected IPA-bearing it fixture.
- BR-5 — addressed — Verified by mutation — dropping an Italian-only marker in Render reddens only TestRenderLosesNothing/it/*.
- BR-6 — not-addressed — Still strings.Contains(span, name); reproduced green with the atlas row renamed to "Italiano".
- BR-7 — not-addressed — dictselect_test.go:428 still re-globs rather than calling loadFakeDictionary.
- BR-8 — not-addressed — atlas/define.md:1303 still has no blank line before the heading; the same insertion also left a double blank line at :1269.
- BR-9 — not-addressed — No Revisions entry landed; plan:148 Files line unchanged; langHelp also has no Core-concepts row.

### Raised

- **BR-10** [Important] `curated-consumer-unpinned` the class was declared closed in 19b5ea1 but the enumeration was never written; six sites remain, two wrong today
  4th round of this family. Do NOT patch a sixth instance — the rule is already in
  workshop/lessons.md:2390 and :2176 and fired anyway. Rule to enforce: every per-language
  enumeration ranges over curated, and every language-keyed table is cross-checked against
  curated in both directions. Measured enumeration at HEAD: (1) dict_conformance_test.go:184
  own-language table, no cross-check; (2) dictselect_test.go:157 hardcodes {"en","es"} —
  changing it to range over curated fails today with "it: no choice from the measured set";
  (3) dictselect_test.go:23 installedOnThisMachine() never gained the Italian books #31
  measured as installed, so dictselect_test.go:399 re-declares them locally (ARCH-MOCK,
  ARCH-DRY) — this is what blocks site 2; (4) atlas/define.md:1271 "Three curated languages"
  sits four lines ABOVE the guarded span, exactly as round 1 predicted; (5) atlas/define.md:1297
  says es and it are "six fixtures each" — entries/es holds FIVE, wrong today; (6) main.go:408's
  flag registration is unpinned, so reverting langHelp to a literal is invisible to go test.
  Sites 4 and 5 are answered by naming the list rather than counting it. ARCH-PURPOSE.
- **BR-11** [Minor] `one-predicate-two-spellings` the widened strong invariant subsumes TestRenderLosesNothingInEveryCapturedLanguage, whose comment is now false
  3rd in family. Do NOT just delete this one — the rule: when a check is widened to the
  dimension its narrower sibling existed to cover, the sibling is deleted in the SAME commit.
  dict_fake_test.go:297 asserts only a non-empty render over capturedLanguages, which
  TestRenderLosesNothing (invariant_test.go:169) now covers with exact alnum counts over the
  same set, so it can no longer fail first; and its doc comment at :290 still claims
  TestRenderLosesNothing "goes through testDict(t), which is English by definition, so the
  Spanish captures were never run through the parser and renderer at all", which this round
  made false. Enumerated at HEAD: this is the only site.

## Round 3 — 2026-08-29T13:26:29-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — TestDocsNameEveryCuratedLanguage now loops README + atlas (verified red on the atlas row), and the own-language table gained its cross-check (verified red).
- BR-6 — not-addressed — Measured at HEAD: renaming the README bullet label to "- **Itaian** —" still passes, because the same bullet's next sentence says "Italian"; the word-boundary change only closed the "Italiano" near-miss.
- BR-7 — addressed — TestEveryCuratedLanguageHasACorpus now goes through loadFakeDictionary, which also rejects zero-byte fixtures.
- BR-8 — addressed — Blank line present before "## Source pronunciation"; a new double-blank appeared four lines earlier (raised as doc-formatting).
- BR-9 — addressed — Task 5's Files line now names the files that actually changed, and the Revisions entry records why the comment fix was not performed.
- BR-10 — not-addressed — Five of six sites verified red on removal; site 6 is half-closed — replacing langHelp's body with the stale literal "en, es" and dropping the three imports leaves go test ./... fully green (measured), which is the exact regression the finding described.
- BR-11 — addressed — The subsumed test is deleted with a tombstone, and TestRenderLosesNothing verifiably emits en/es/it subtests.

### Raised

- **BR-12** [Important] `retired-symbol-unswept` the rename/deletion this round performed never got its retiredSymbolNames row, so the atlas and the plan name tests the tree does not declare
  atlas/define.md:1181 names TestSpanishEntriesCarryNoPronunciationNotation (renamed to
  TestNonEnglishEntriesCarryNoPronunciationNotation); the plan names it plus the deleted
  TestRenderLosesNothingInEveryCapturedLanguage and a TestItalianEntriesCarryNoPronunciationNotation
  that never existed. Measured: adding the two rows to retiredSymbolNames
  (repo_guard_test.go:719) turns TestNoArtifactNamesARetiredSymbol red on atlas/define.md and
  twice on the plan. The map's own comment records this human half failing once before (#27),
  so the rule to build is mechanical: derive removed func declarations from the review window
  with git — TestPlanTableStatusMatchesTheChangeWindow already shells to git in this file — and
  require either a retiredSymbolNames row or zero current-truth mentions.
- **BR-13** [Important] `curated-consumer-unpinned` the class is still enforced per-site; two hand-written enumerations remain and the mechanism should be one surface registry
  This is the 5th finding in this family, so do NOT patch the two sites. Shadow-sweep at HEAD:
  nine consumers of curated, seven enforced. Unenforced — (a) the {es,it} table in
  TestNonEnglishEntriesCarryNoPronunciationNotation (render_test.go:366), which is a
  language-keyed table with no cross-check, so #34 can curate fr and gain no row; (b) the doc
  list {README, atlas} spelled three times (dictselect_test.go:515, doc_sync_test.go:125, :142),
  so a third doc is covered only by whichever test its author remembered. Rule: curated gets ONE
  registry of surfaces obliged to name every curated language, and each row supplies the
  surface's TEXT (README span, atlas span, langHelp) so the assertion is "every curated language
  appears in that text" — which makes replacing a derivation with a literal fail, the gap BR-10
  leaves open. Language-keyed predicate tables get a cross-check with an EXPLICIT exempt list,
  because German's Duden field is real and an IPA=="" row for de would be wrong; today "no row"
  and "deliberately no row" are indistinguishable. ARCH-PURPOSE, ARCH-DRY.
- **BR-14** [Minor] `one-predicate-two-spellings` the per-captured-language sweep harness is now spelled three times, two of them new in this diff
  This is the 4th finding in this family, so the fix is the helper, not the site. Measured
  prevalence: invariant_test.go:170, render_test.go:242, render_test.go:326 all read
  `for lang := range capturedLanguages(t) { t.Run(string(lang), func(t){ xIn(t, testDictFor(t, lang)) }) }`.
  One forEachCapturedLanguage(t, func(*testing.T, *fakeDictionary)) collapses all three, and a
  fourth sweep then cannot be written narrower than the corpus by accident.
- **BR-15** [Minor] `doc-formatting` atlas/define.md:1269-1270 has a double blank line and a raggedly re-wrapped paragraph
  This is the 2nd finding in this family (BR-8 was a missing blank line before a heading), so
  state the rule rather than patching the line: nothing in the repo checks markdown block
  spacing or wrap in atlas/ and README.md, and both instances were introduced by hand-editing
  prose. Either run a markdown formatter over the touched docs as part of the boundary, or
  accept the class explicitly. Also at :1288 the ragged rewrap left "…held five. The live /
  ratchet in live_property_test.go…".
- **BR-16** [Minor] `redundant-syntax` dictselect_test.go:475 wraps a single t.Errorf in a bare block instead of inverting the guard
  `if err == nil && len(d.entries) > 0 { return }` followed by `{ t.Errorf(...) }` reads as
  `if err != nil || len(d.entries) == 0 { t.Errorf(...) }`.

## Round 4 — 2026-08-29T13:54:07-07:00 (claude) — passed

### Disposed

- BR-6 — not-addressed — Re-measured at HEAD: renaming the README bullet to "- **Itaian** —" leaves TestEverySurfaceNamesEveryCuratedLanguage green, because the same bullet's next sentence says "Italian". Minor, non-blocking.
- BR-10 — addressed — Site 6 closed by the registry row: replacing langHelp's body with the literal "en, es" now reddens TestEverySurfaceNamesEveryCuratedLanguage/the -lang flag help (verified by revert in a scratch clone).
- BR-12 — addressed — Rows added and all three artifacts swept (incl. workshop/issues/000030). Mechanism verified live: removing the two rows and re-injecting a stale mention turns TestARemovedDeclarationIsSweptOrRetired red. Its matching rule is defective — raised separately.
- BR-13 — not-addressed — The registry (part a) is delivered and real; part (b) is not — derivedDocs at dictselect_test.go:487 has zero references while doc_sync_test.go:125 and :142 still spell the doc list literally.
- BR-14 — not-addressed — Bodies were extracted but the harness is still spelled three times verbatim: invariant_test.go:170, render_test.go:243, render_test.go:327. No forEachCapturedLanguage helper exists.
- BR-15 — not-addressed — Instances mostly repaired (double blank gone), but no rule was stated and none accepted; atlas/define.md:1273 (94 cols) and :1303 (87) still break the file's ~80-col wrap.
- BR-16 — addressed — dictselect_test.go:472 now reads `if err != nil || len(d.entries) == 0 {`.

### Raised

- **BR-17** [Important] `retired-symbol-unswept` the new removed-declaration guard matches by substring, so an ordinary rename fires seven false failures and its suggested remedy poisons the sibling guard
  repo_guard_test.go:1120 uses strings.Contains, while TestNoArtifactNamesARetiredSymbol:791
  uses a word-boundary regex for the same question — the two guards disagree on the matching
  rule that currentTruthFiles' own comment says they cannot disagree about. Measured in a
  scratch clone: renaming the helper `func ids` to `idsOfMetas` and committing turns the guard
  red on seven paths including atlas/define.md, which has ZERO word-boundary occurrences of
  "ids" (the hit is "forbids"). The message then tells the author to add a retiredSymbolNames
  row for "ids", which would make TestNoArtifactNamesARetiredSymbol permanently red on every
  file containing that word. Fix: share the word-boundary regex, and bound `gone` to names
  unlikely to collide (Test*/exported, or a length floor).
- **BR-18** [Important] `one-predicate-two-spellings` both helpers introduced to collapse duplication have zero and one call site while the spellings they replace remain, under comments claiming the consolidation happened
  This is the 5th finding in this family, so the fix is the rule, not the two sites: a
  consolidation is complete only when the new helper's call-site count equals the number of
  spellings it replaced — grep for callers before disposing the finding. Measured at HEAD:
  (a) derivedDocs (dictselect_test.go:487, commented "named ONCE") has zero references
  anywhere; doc_sync_test.go:125 and :142 still spell the two-doc list literally and
  curatedSurfaces() spells each path separately at :538/:543 — package-level vars escape Go's
  unused check, so it passes every suite while doing nothing. (b) currentTruthFiles
  (repo_guard_test.go:1134, commented "Shared with TestNoArtifactNamesARetiredSymbol so the
  two guards cannot disagree") has one caller; TestNoArtifactNamesARetiredSymbol:759-780
  keeps a byte-identical binds closure and ls-files loop, and the two copies HAVE ALREADY
  DIVERGED — the inline one Fatals on `seen == 0`, the extracted one has no vacuity
  assertion, so the new guard would pass over an empty file set. ARCH-DRY.
- **BR-19** [Important] `curated-consumer-unpinned` the own-language table's curated cross-check is pure data but sits behind the conformance build tag, so it never runs in go test ./... or CI
  This is the 6th finding in this family, so state the rule rather than adding a row: the PURE
  half of a cross-check belongs in an untagged file; only the live lookup half needs the tag.
  dict_conformance_test.go:202's `for lang := range curated { … no row for it … }` compares two
  in-memory tables and needs no dictionary, network or macOS, yet the file carries
  `//go:build darwin && conformance` — which its own header says a CI runner does not satisfy,
  and which .github/workflows/merge-check.yml does not pass. So the obligation "a curated
  language has a live own-language check" is enforced only for whoever remembers to run the
  tagged suite and read it, which is the per-site enforcement BR-13 set out to end. Fix: move
  the rows table and the cross-check to an untagged file beside the registry, leaving the
  t.Run lookup tagged. ARCH-PURPOSE.
- **BR-20** [Minor] `curated-consumer-unpinned` capture.sh's mkdir line and closing summary enumerate languages by hand and no registry row covers them
  capture.sh:11 (`mkdir -p entries/en entries/es entries/it`) and :177 (the echo/wc summary)
  are per-language enumerations. TestCaptureScriptUsesTheCuratedDictionaries pins the
  IDENTIFIERS in both directions but not these, so a fourth language yields a summary that
  under-reports — the exact failure the comment added at :172 says the summary exists to
  prevent. A whole-file token check cannot work ("it" is an English word throughout the
  script), so this needs a marked span or an explicit exemption with its reason, the way
  notationExempt makes "deliberately no row" legible.
- **BR-21** [Minor] `comment-overclaims-code` dictselect_test.go:52 says the fixture is cross-checked "in BOTH directions" but only curated to fixture is implemented
  TestTheMeasuredSetModelsEveryCuratedLanguage loops curated and asserts each id is modelled;
  nothing checks the reverse. The reverse is deliberately absent — the fixture carries
  bilingual and thesaurus decoys — so the comment should change, not the code. Same shape as
  the two false structural claims in the Important finding above ("named ONCE", "Shared
  with"), all three introduced by the rounds that were closing findings about exactly this.
- **BR-22** [Minor] `retired-symbol-unswept` the removed-declaration guard's skip message says "top-level declaration" but the regex matches only ^-func
  repo_guard_test.go:1111 reports "this window removed no top-level declaration" while the
  regex at :1093 matches `^-func` only, so a removed var/const/type is invisible. Narrow the
  message or widen the regex.

## Open findings

- **BR-6** [Minor] `curated-consumer-unpinned` the README guard is free-text containment inside the span, so a broken bullet label survives
- **BR-13** [Important] `curated-consumer-unpinned` the class is still enforced per-site; two hand-written enumerations remain and the mechanism should be one surface registry
- **BR-14** [Minor] `one-predicate-two-spellings` the per-captured-language sweep harness is now spelled three times, two of them new in this diff
- **BR-15** [Minor] `doc-formatting` atlas/define.md:1269-1270 has a double blank line and a raggedly re-wrapped paragraph
- **BR-17** [Important] `retired-symbol-unswept` the new removed-declaration guard matches by substring, so an ordinary rename fires seven false failures and its suggested remedy poisons the sibling guard
- **BR-18** [Important] `one-predicate-two-spellings` both helpers introduced to collapse duplication have zero and one call site while the spellings they replace remain, under comments claiming the consolidation happened
- **BR-19** [Important] `curated-consumer-unpinned` the own-language table's curated cross-check is pure data but sits behind the conformance build tag, so it never runs in go test ./... or CI
- **BR-20** [Minor] `curated-consumer-unpinned` capture.sh's mkdir line and closing summary enumerate languages by hand and no registry row covers them
- **BR-21** [Minor] `comment-overclaims-code` dictselect_test.go:52 says the fixture is cross-checked "in BOTH directions" but only curated to fixture is implemented
- **BR-22** [Minor] `retired-symbol-unswept` the removed-declaration guard's skip message says "top-level declaration" but the regex matches only ^-func
