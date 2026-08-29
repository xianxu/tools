# Boundary Review — tools#31 (whole-issue close)

| field | value |
|-------|-------|
| issue | 31 — curate the Italian dictionary so /lang it is a real mode (French and German split to #34) |
| repo | tools |
| issue file | workshop/issues/000031-curate-dictionaries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83134e464ba38dce607296341a01fa9cc5d343f7..1d7627e750709098e764b476fff95863ecb174a7 |
| command | sdlc close --issue 31 |
| reviewer | claude |
| timestamp | 2026-08-29T12:32:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue delivers what it narrowed to: `"it": {"com.apple.dictionary.it.Devoto-Oli"}` lands, a real six-entry Devoto-Oli corpus is committed, and the four guards Task 5 promised are genuinely load-bearing — I reverted each in a scratch copy of `1d7627e` and watched it redden (emptying `entries/it/` fails `TestEveryCuratedLanguageHasACorpus`; deleting the Italian README bullet fails `TestDocsNameEveryCuratedLanguage`; reverting the regex hyphen fails `TestCaptureScriptUsesTheCuratedDictionaries` in both directions; dropping the `curated` row fails `TestChooseDictionaryPicksTheCuratedItalian`; injecting an IPA-bearing entry into `entries/it/` fails `TestItalianEntriesCarryNoPronunciationNotation`). `go test ./...`, `go vet ./...` and `gofmt -l` are all clean. What blocks SHIP is the shadow-sweep (ARCH-PURPOSE): the plan itself named `doc-sweep-incomplete` and said Task 5 would close it as a class, but it closed two of five `curated` consumers — the `-lang` flag help a user actually reads still says **`en, es`**, and the atlas table this very diff added is unpinned even though the two tests `TestDocsNameEveryCuratedLanguage` was modelled on both loop over README *and* atlas for exactly that reason. Note one verification limit: DictionaryServices is unreachable from this review process (even `en` skipped), so the live conformance rows — `TestSelectedDictionaryAnswersInItsOwnLanguage/it` and `TestFixturesMatchLiveDictionary/it` — could not be executed here; I verified everything reachable off the fake.

## 1. Strengths

- **The measurement is the deliverable, and it changed the scope honestly.** `dictselect.go:80-89` replaces an invitation ("each is one line") with the numbers that make it false for two of three books, and points at `#34`. That comment is now the thing that stops the next person repeating `#29`'s mistake.
- **`TestEveryCuratedLanguageHasACorpus` (`dictselect_test.go:428`) closes a direction nothing checked.** `capturedLanguages` derives from the directory and guards only `len >= 2`, so deleting `entries/it/` outright was previously invisible. Verified red on removal.
- **The own-language conformance check became a table rather than a second copy** (`dict_conformance_test.go:180-225`), with the `{shared, marker, absent}` triple documented per column. That is the right generalisation and a fourth language really is one row.
- **The hyphen regex defect was found by the data, and its comment says so** (`dictselect_test.go:356-362`). Reverting it reproduces exactly the two-directional failure described — the comment is not a story about the fix, it is the fix's failure mode.
- **Fixtures are real captured output, not approximations.** `ciao.txt` carries the full `A. inter. … B. s.m. … ACCRESCITIVO … ETIMOLOGIA … DATA 1905.` shape the plan claimed it would; `essere.txt` is 11.6 KB. ARCH-MOCK holds — production and test share the `Dictionary` seam, and `capture.sh` captures through the same identifier `chooseDictionary` selects.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `cmd/define/main.go:408`: the `-lang` help still enumerates `en, es`.**
```
-lang string
    language for this invocation: en, es (default: the directory's setting)
```
Confirmed by running the built binary. This is the shipped, user-facing enumeration of curated languages, it is now factually wrong, and it is the only `curated` consumer that is wrong rather than merely unpinned. The repo already owns the mechanism: `pronHelp` and `localeHelp` are production constants that the docs consume through marked spans (`doc_sync_test.go:123-152`) — there is no `langHelp`, just an inline literal. Fix sketch: introduce `langHelp` built from `slices.Sorted(maps.Keys(curated))` so the flag help *derives* from `curated` instead of restating it, and it can never go stale again (ARCH-PURPOSE, ARCH-DRY).

**I-2 — `cmd/define/dictselect_test.go:470`: the new guard reads only `README.md`, while this same diff added a curated-language table to `atlas/define.md:1271`.**
`TestDocsQuoteTheLocaleHelp` (`doc_sync_test.go:137-141`) carries the comment *"EVERY doc that states the policy, not just the first one wired up. Fixing the README alone left atlas/define.md as the next copy to go stale, which is the same half-fix this family keeps producing."* That is this diff. When `#34` adds `fr`/`de`, `atlas/define.md` will still read "**Three curated languages**" beside a paragraph saying French and German were *not* curated — actively false, and nothing red. Fix: mark a span in `atlas/define.md` and loop `TestDocsNameEveryCuratedLanguage` over `[]string{"../../README.md", "../../atlas/define.md"}`, exactly as its two model tests do. The third remaining site is the `dict_conformance_test.go:190-196` table itself: nothing requires a curated language to have an own-language row, so `#34`'s languages can enter with no live check at all — a `for lang := range curated` guard (excluding `en`, the reference side) closes it.

**I-3 — `cmd/define/render_test.go:335-381`: the new Italian test was inserted *inside* the Spanish test's doc comment, so both comments now document the wrong function.**
The block at :335 ("A Spanish entry carries NO pronunciation notation… Spanish orthography is phonemic… the recording is load-bearing…") is now the godoc of `TestItalianEntriesCarryNoPronunciationNotation` at :358, and it contradicts the Italian paragraph appended below it ("Devoto-Oli DOES write something"). `TestSpanishEntriesCarryNoPronunciationNotation` at :383 is left with only the orphaned tail ("SCOPED DELIBERATELY… the claim above"), whose antecedent is now the Italian test. Task 5 says a corrected test with stale prose beside it is how the next reader is misled; this is that, produced by the fix. Fix: move `TestItalianEntriesCarryNoPronunciationNotation` and its own comment *below* `TestSpanishEntriesCarryNoPronunciationNotation`, leaving :335-355 attached to the Spanish function.

**I-4 — ARCH-DRY: `TestItalianEntriesCarryNoPronunciationNotation` is a second spelling of the Spanish one, added in the commit that removed a second spelling.**
`render_test.go:358-376` and `:383-398` are byte-for-byte the same loop — same empty guard, same redundant `d.Lookup(word)` round-trip over a map whose value is already `raw`, same `ParseEntry(raw).IPA != ""` assertion — differing only in `lang` and message. Plan D4 justified the copy because the *reason* differs, but the reason lives in prose; the *predicate* is identical, and Task 3's own text warns this is the `one-predicate-two-spellings` family where "the second landed in the commit that fixed the first." Fix: one `noNotationIn(t, lang, why string)` helper driven by a two-row table, with the per-language rationale as the per-row comment — the shape Task 3 just adopted two files over.

**I-5 — the strong no-data-loss invariant is still English-only, and widening it is free.**
`TestRenderLosesNothing` (`invariant_test.go:161`) does the real work — exact alnum count plus `subsequenceGap`. It takes `testDict(t)`. What Italian gets instead is `TestRenderLosesNothingInEveryCapturedLanguage`, which asserts only that the render is non-empty; it would pass while 90% of `essere`'s 11.6 KB was dropped. Devoto-Oli's `•` sub-senses, `A./B.` blocks and unknown `ETIMOLOGIA`/`DATA`/`ACCRESCITIVO` sections are precisely where the parser is most likely to lose content, so this is the kind of bug this diff could ship. **Measured:** I ran the strong invariant over `capturedLanguages` at HEAD and all six Italian and five Spanish entries pass. It is ~6 lines, it is green today, and it is the strongest pin the corpus can give the new language — which is what issue Done-when #3 ("the no-data-loss invariant cover it") reads as.

## 4. Minor findings

- `dictselect_test.go:485` — the README guard is free-text containment *within* the span, so it survives a broken bullet: mangling `- **Italian** —` to `- **Itaian** —` still passes, because the same bullet later says "Italian has no recordings". Narrower than the whole file, but the same class the test's own comment condemns. Anchoring on the bullet's bold label would close it.
- `dictselect_test.go:431` — the guard re-implements the corpus glob instead of calling `loadFakeDictionary("testdata/entries", lang)`, which already reports both "no fixtures" and "empty fixture" with a better message (ARCH-DRY).
- `atlas/define.md:1301` — no blank line between the new limitation paragraph and `## Source pronunciation (#29)`.
- `dict_conformance_test.go:194` — the `it` row's `absent` word is `sycophantic`, copied from the `es` row. Fine today, but a per-row `absent` that is always the same value invites the column being read as a constant.

## 5. Test coverage notes

- Every guard this boundary claims was verified by reverting, not by reading: five mutations, five reds, plus the two documented green-on-row-removal cases (dropping the `curated` row leaves both `curated`-derived guards green — correct, since uncurating withdraws the obligation; the plan's "Verification before close" already states this and it checks out).
- `TestNoRawPronunciationNotationSurvives` genuinely enumerates `en`/`es`/`it` subtests (47 words); the widening is real, not a field set at zero call sites. Both comments it was widened to satisfy (`rawnotation_test.go:148`, `capture.sh:156-158`) are now true as written.
- Uncovered here: the live half. `TestSelectedDictionaryAnswersInItsOwnLanguage/it` and `TestFixturesMatchLiveDictionary/it` both skipped in this environment because DictionaryServices resolves nothing for this process (`en` skipped too, so it is the harness, not the change). The operator should re-run `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define/` from a real session before the close verdict is recorded — that is the only place the `s.f.` marker and the `sycophantic → ErrNoEntry` absence are actually exercised.
- I-5 is the one coverage gap where the missing test is both cheap and measured-green.

## 6. Architectural notes

- **ARCH-DRY — flag.** I-4 (two spellings of the no-notation predicate) and the Minor glob duplication. Everything else consolidated well: the conformance table, `noRawNotationIn`, and reusing `rawNotationNear` rather than a fourth spelling.
- **ARCH-PURE — pass.** `chooseDictionary`/`monolingualIn` remain pure functions over `dictMeta`, unit-testable with no dictionaries installed; `TestChooseDictionaryPicksTheCuratedItalian` constructs metadata literals and needs no IO. The new guards touch the filesystem only to read committed artifacts, which is the correct test-side seam, not business logic in IO.
- **ARCH-PURPOSE — flag.** Shadow sweep of `curated`'s consumers: `chooseDictionary` derives ✓; `capture.sh` pinned ✓; README pinned ✓ (new); corpus pinned ✓ (new); **`-lang` help — hand-maintained and now wrong** (I-1); **atlas table — hand-maintained, unpinned** (I-2); **own-language conformance table — hand-maintained, unpinned** (I-2). Task 4 wrote "that is the `doc-sweep-incomplete` shape #29 closed twice… Task 5 does the same here" — the enumeration was named but never written out, so the sweep stopped at the two sites someone happened to look at. Writing the enumeration (grep `curated` + grep the language names) is what turns this into the class.
- **ARCH-MOCK — pass.** No new external dependency; the Devoto-Oli enters through the existing `Dictionary` seam with `fakeDictionary` behind it and `TestFixturesMatchLiveDictionary` as the live conformance half, both of which pick Italian up from the directory automatically. `capture.sh` captures through the identifier `chooseDictionary` selects, not by name match, and states why. The only seam-level gap is I-2's third site: the live half has no guard forcing a curated language to acquire a row.
- **For `#34`:** the three fixes above are what make adding `fr`/`de` a row rather than a five-site sweep. Landing I-1 and I-2 now means `#34` adds one `curated` entry and gets a red build listing every doc and check that must catch up — which is the outcome this issue was trying to build.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000031-curate-dictionaries-plan.md` covering:

1. **Task 5's `**Files:**` line is wrong.** It names `cmd/define/doc_sync_test.go` and `cmd/define/dict_fake_test.go`; neither was modified. `TestDocsNameEveryCuratedLanguage` and `TestEveryCuratedLanguageHasACorpus` landed in `cmd/define/dictselect_test.go`. (The Core-concepts table itself is accurate — every row checks out against the diff.)
2. **Task 5's first bullet is ticked but its instruction was not followed, correctly.** It says "then **fix both comments** — `capture.sh` and `rawnotation_test.go`"; neither was edited, because widening the check made both true. The Revisions section already records this; the task text should be amended in place so a reader does not go looking for a comment edit that should not exist.
3. **D5 needs a scope correction.** It reasons about "the README's book list" and concludes with a README-only guard, but the same boundary added a curated-language table to `atlas/define.md`. Record that the decision covers *every doc that restates `curated`*, and that the `-lang` flag help is a consumer D5 did not consider at all.
4. **The "Done-when coverage" table's row 3** cites `TestRenderLosesNothingInEveryCapturedLanguage` as the no-data-loss pin; state plainly that this is the non-empty check, and that the alnum/subsequence invariant (`invariant_test.go:161`) remains English-only — either widen it (measured green, per I-5) or record the narrower coverage as the decision, the way D3 records the dictionary-width gap.

```findings
findings:
  - id: new
    severity: Important
    family: curated-consumer-unpinned
    title: |
      the -lang flag help still enumerates "en, es" and does not derive from curated
    detail: |
      cmd/define/main.go:408 hardcodes "language for this invocation: en, es"; the built
      binary prints it, so `define -h` denies a language the README now documents. This is
      the only curated consumer that is wrong rather than merely unpinned. Build a langHelp
      constant from slices.Sorted(maps.Keys(curated)), following the pronHelp/localeHelp
      pattern the repo already owns (ARCH-PURPOSE, ARCH-DRY).
  - id: new
    severity: Important
    family: curated-consumer-unpinned
    title: |
      TestDocsNameEveryCuratedLanguage pins README only; the atlas table this diff added is unchecked
    detail: |
      dictselect_test.go:470 reads only ../../README.md, while atlas/define.md:1271 gained a
      hand-maintained "Three curated languages" table in the same commit. Its two model tests,
      TestDocsQuoteThePronHelp and TestDocsQuoteTheLocaleHelp, both loop over README AND atlas
      with a comment naming this exact half-fix. Third unpinned site: the own-language table at
      dict_conformance_test.go:190 has no guard requiring a curated language to have a row.
  - id: new
    severity: Important
    family: doc-comment-misattached
    title: |
      the Italian test was inserted inside the Spanish test's doc comment, so both are misdocumented
    detail: |
      render_test.go:335-355 ("A Spanish entry carries NO pronunciation notation… orthography is
      phonemic…") is now the godoc of TestItalianEntriesCarryNoPronunciationNotation at :358 and
      contradicts the Italian paragraph appended to it. TestSpanishEntriesCarryNoPronunciationNotation
      at :383 keeps only the orphaned "SCOPED DELIBERATELY… the claim above" tail, whose antecedent
      is now the Italian test. Move the Italian function and its own comment below the Spanish one.
  - id: new
    severity: Important
    family: one-predicate-two-spellings
    title: |
      the Italian no-notation test is a verbatim second spelling of the Spanish one
    detail: |
      render_test.go:358-376 and :383-398 share the same empty guard, the same redundant
      d.Lookup(word) over a map whose value is already raw, and the same ParseEntry(raw).IPA
      assertion. Task 3 generalised the sibling conformance check to a table in this very
      boundary while warning that this family's second instance lands in the commit that fixes
      the first. Consolidate to noNotationIn(t, lang, why) driven by a two-row table (ARCH-DRY).
  - id: new
    severity: Important
    family: check-narrower-than-corpus
    title: |
      the strong no-data-loss invariant is still English-only; widening it is measured green
    detail: |
      TestRenderLosesNothing (invariant_test.go:161) does the alnum-count + subsequenceGap check
      and takes testDict(t). Italian gets only TestRenderLosesNothingInEveryCapturedLanguage,
      which asserts a non-empty render and would pass while most of essere's 11.6KB was dropped —
      and the Devoto-Oli shapes (• sub-senses, A./B. blocks, ETIMOLOGIA/DATA) are exactly where
      loss is likely. Verified at HEAD: the strong invariant passes over all captured languages,
      so widening it costs ~6 lines and is what Done-when 3 reads as.
  - id: new
    severity: Minor
    family: curated-consumer-unpinned
    title: |
      the README guard is free-text containment inside the span, so a broken bullet label survives
    detail: |
      Verified: renaming the bullet "- **Italian** —" to "- **Itaian** —" keeps
      TestDocsNameEveryCuratedLanguage green, because a later sentence in the same bullet says
      "Italian". Narrower than whole-file containment, but the same class the test's own comment
      condemns. Anchor on the bold label rather than anywhere in the span.
  - id: new
    severity: Minor
    family: one-predicate-two-spellings
    title: |
      TestEveryCuratedLanguageHasACorpus re-globs instead of calling loadFakeDictionary
    detail: |
      dictselect_test.go:431 duplicates the corpus glob. loadFakeDictionary("testdata/entries", lang)
      already answers the same question and additionally rejects zero-byte fixtures, with a message
      that names capture.sh.
  - id: new
    severity: Minor
    family: doc-formatting
    title: |
      atlas/define.md:1301 has no blank line before the "## Source pronunciation" heading
  - id: new
    severity: Minor
    family: plan-table-drifts-from-code
    title: |
      Task 5's Files line names doc_sync_test.go and dict_fake_test.go, neither of which changed
    detail: |
      Both new guards landed in cmd/define/dictselect_test.go. Separately, Task 5's first bullet is
      ticked while instructing a comment fix in capture.sh and rawnotation_test.go that was
      correctly not performed (widening made both comments true). The Core concepts table itself
      matches the code; only the per-task Files line and that bullet need a Revisions entry.
```
