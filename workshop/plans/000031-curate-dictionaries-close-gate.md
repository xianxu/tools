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

## Open findings

- **BR-1** [Important] `curated-consumer-unpinned` the -lang flag help still enumerates "en, es" and does not derive from curated
- **BR-2** [Important] `curated-consumer-unpinned` TestDocsNameEveryCuratedLanguage pins README only; the atlas table this diff added is unchecked
- **BR-3** [Important] `doc-comment-misattached` the Italian test was inserted inside the Spanish test's doc comment, so both are misdocumented
- **BR-4** [Important] `one-predicate-two-spellings` the Italian no-notation test is a verbatim second spelling of the Spanish one
- **BR-5** [Important] `check-narrower-than-corpus` the strong no-data-loss invariant is still English-only; widening it is measured green
- **BR-6** [Minor] `curated-consumer-unpinned` the README guard is free-text containment inside the span, so a broken bullet label survives
- **BR-7** [Minor] `one-predicate-two-spellings` TestEveryCuratedLanguageHasACorpus re-globs instead of calling loadFakeDictionary
- **BR-8** [Minor] `doc-formatting` atlas/define.md:1301 has no blank line before the "## Source pronunciation" heading
- **BR-9** [Minor] `plan-table-drifts-from-code` Task 5's Files line names doc_sync_test.go and dict_fake_test.go, neither of which changed
