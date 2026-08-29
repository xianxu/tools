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

## Open findings

- **BR-6** [Minor] `curated-consumer-unpinned` the README guard is free-text containment inside the span, so a broken bullet label survives
- **BR-10** [Important] `curated-consumer-unpinned` the class was declared closed in 19b5ea1 but the enumeration was never written; six sites remain, two wrong today
- **BR-12** [Important] `retired-symbol-unswept` the rename/deletion this round performed never got its retiredSymbolNames row, so the atlas and the plan name tests the tree does not declare
- **BR-13** [Important] `curated-consumer-unpinned` the class is still enforced per-site; two hand-written enumerations remain and the mechanism should be one surface registry
- **BR-14** [Minor] `one-predicate-two-spellings` the per-captured-language sweep harness is now spelled three times, two of them new in this diff
- **BR-15** [Minor] `doc-formatting` atlas/define.md:1269-1270 has a double blank line and a raggedly re-wrapped paragraph
- **BR-16** [Minor] `redundant-syntax` dictselect_test.go:475 wraps a single t.Errorf in a bare block instead of inverting the guard
