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

---

## Re-review — 2026-08-29T13:00:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 31 — curate the Italian dictionary so /lang it is a real mode (French and German split to #34) |
| repo | tools |
| issue file | workshop/issues/000031-curate-dictionaries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83134e464ba38dce607296341a01fa9cc5d343f7..3593ebb86b65d980929c3e9f27381776d49a998a |
| command | sdlc close --issue 31 |
| reviewer | claude |
| timestamp | 2026-08-29T13:00:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Five of the nine prior findings are genuinely fixed, and I confirmed each by mutation rather than by reading: the widened strong invariant reddens only `TestRenderLosesNothing/it/*` when I made `Render` drop an Italian-only marker (`s.f.`), the consolidated no-notation table reddens on an injected IPA-bearing `it` fixture, the atlas half of the docs guard reddens on a mangled bullet, and the built binary now prints `-lang … en, es, it`. `go test ./...`, `go vet` and `gofmt -l` are clean. What blocks SHIP is the same thing that blocked it last round, one level up: commit `19b5ea1` states the class answer — *"every surface that enumerates the curated languages derives from `curated`"* — and delivers two of at least six sites. BR-2's own third named site is untouched, and I found three more the previous round did not name, one of which is factually wrong today (`atlas/define.md:1297` says the Spanish corpus is six fixtures; it is five) and one of which the previous round explicitly predicted would survive (`atlas/define.md:1271`'s "**Three curated languages**" sits four lines *above* the `<!-- curated-languages -->` span, so the new guard cannot see it). This is the 4th round of `curated-consumer-unpinned`, and per ARCH-PURPOSE the deliverable now is the written enumeration plus the sweep, not a sixth instance patch. Verification limit, unchanged from round 1: DictionaryServices resolves nothing for this process (`en` skips too), so `TestSelectedDictionaryAnswersInItsOwnLanguage/it` and `TestFixturesMatchLiveDictionary/it` could not be executed here.

## 1. Strengths

- **BR-5's widening is real and I proved it.** Making `Render` strip `s.f.` (an Italian-only string) reddens exactly `TestRenderLosesNothing/it/{casa,pizza,acqua}` with `alnum count 4915 rendered vs 4917 raw`, and leaves `en`/`es` green. The weak sibling would have stayed green through the same mutation. `invariant_test.go:169` and `render_test.go:326` now both sweep `capturedLanguages`, and six Italian word-subtests genuinely run.
- **BR-3 and BR-4 answered as one change, correctly.** `render_test.go:364` is one table with a per-row `why`, and the `why` is not decorative — the failure message it produced under my injected fixture carries the full Devoto-Oli syllabification argument `#30` reads this pair for. Collapsing the two functions removed the misattached comment as a side effect rather than shuffling it.
- **BR-2's atlas half is load-bearing.** Renaming the atlas row to `Itaian` fails with `../../atlas/define.md's marked span never names Italian (it)`. Note my first mutation (`Italiano`) passed — that is BR-6, still open, reproduced.
- **`langHelp` (`voice.go:158`) is the right shape.** Derived from `slices.Sorted(maps.Keys(curated))`, and its comment explains both the sort (map iteration order) and the scope (it names what a *dictionary* is curated for; `ParseLang` still accepts `fr`, which degrades per `#23 M2`). Confirmed in the built binary's `-h`.
- **The corpus matches its documented rationale.** `ciao.txt` really carries `A. inter. … B. s.m. … ACCRESCITIVO … ETIMOLOGIA … DATA 1905.`; `parlare.txt` really carries the homograph number before the syllabification (`parlare 1 (par·là·re)`); `essere.txt` is 11.6 KB. The words prove what `capture.sh` says they prove.

## 2. Critical findings

None.

## 3. Important findings

**BR-2 (not-addressed) — the third site it named is untouched.** `dict_conformance_test.go:184-227` is a hand-written `{lang, shared, marker, absent}` table with no cross-check against `curated`. `#34` can add `fr`/`de` rows to `curated` and acquire no live own-language check at all. The atlas half is fixed and verified; this half is not.

**NEW (Important, `curated-consumer-unpinned`) — the enumeration was still never written.**

> **This is the 4th finding in family `curated-consumer-unpinned`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

The rule is already written down twice in this repo — `workshop/lessons.md:2390` ("A doc that ENUMERATES something must derive from it") and `:2176` ("Give a count one producer, or delete the count") — and fired again anyway. That, not any single site, is the finding: **no site may state a per-language fact by hand; every per-language enumeration ranges over `curated`, and every language-keyed table is cross-checked against `curated` in both directions** — the mechanism `TestCaptureScriptUsesTheCuratedDictionaries` and `TestDocsNameEveryCuratedLanguage` already implement. Measured prevalence at HEAD, the enumeration the class fix owes:

| # | site | state |
|---|---|---|
| 1 | `dict_conformance_test.go:184` own-language table | no `curated` cross-check (BR-2's third site) |
| 2 | `dictselect_test.go:157` `[]store.Lang{"en", "es"}` | Italian not order-tested. **Measured:** changing it to `for lang := range curated` fails today — `it: no choice from the measured set` |
| 3 | `dictselect_test.go:23` `installedOnThisMachine()` | the modelled installed set never gained the Italian books `#31` measured as installed, so `TestChooseDictionaryPicksTheCuratedItalian:399` re-declares `Devoto-Oli` and `OxfordItalian` locally instead (ARCH-MOCK: the fake diverges from the measured dependency; ARCH-DRY: two declarations of one fixture fact). This is what blocks site 2 |
| 4 | `atlas/define.md:1271` "**Three curated languages as of `#31`**" | a hand-maintained COUNT sitting **four lines above** `<!-- curated-languages -->` (:1275), so the new guard cannot see it — round 1 predicted this exact sentence would survive `#34` |
| 5 | `atlas/define.md:1297` "`es` and `it` … six fixtures each" | **wrong today**: `entries/es/` holds five |
| 6 | `main.go:408` `fs.String("lang", "", langHelp)` | BR-1's derivation is delivered and observed at the binary, but nothing pins the *registration* — reverting it to a literal is invisible to `go test ./...` |

Sites 4 and 5 are answered by naming the list rather than counting it, per the lesson already on file. Site 6 wants a flagset-usage assertion, not a restatement of `langHelp`.

## 4. Minor findings

- **BR-6 (not-addressed)** — `dictselect_test.go:497` is still `strings.Contains(span, name)`. Reproduced: renaming the atlas row to `Italiano` keeps it green, because the name is a substring.
- **BR-7 (not-addressed)** — `dictselect_test.go:428` still re-globs instead of calling `loadFakeDictionary`, which additionally rejects zero-byte fixtures (`dict_fake_test.go:49`).
- **BR-8 (not-addressed)** — `atlas/define.md:1303-1304`, still no blank line before `## Source pronunciation`; and the same insertion added a *double* blank line at `:1269-1270`.
- **BR-9 (not-addressed)** — no `## Revisions` entry for this round landed in the plan; `plan:148`'s Files line still names `doc_sync_test.go` and `dict_fake_test.go` and still omits `dictselect_test.go` and `invariant_test.go`. Additionally `langHelp` is a new PURE entity in `cmd/define/voice.go` with no Core-concepts row — the "table catches stale rows but not absent ones" gap `#27` recorded.
- **NEW (Minor, `one-predicate-two-spellings`)** — `TestRenderLosesNothingInEveryCapturedLanguage` (`dict_fake_test.go:297`) is now strictly subsumed by the widened `TestRenderLosesNothing`, and its doc comment at `:290` ("`TestRenderLosesNothing` goes through `testDict(t)`, which is English by definition, so the Spanish captures … were never run through the parser and renderer at all") is now false.

  > **This is the 3rd finding in family `one-predicate-two-spellings`.** Do NOT fix this instance — the rule: **when a check is widened to the dimension its narrower sibling existed to cover, the sibling is deleted in the same commit.** A weaker duplicate left behind is a test that can never fail plus prose that is now wrong. Enumerated at HEAD: one site (`dict_fake_test.go:290-315`); the raw-notation widening created none and the no-notation change consolidated rather than duplicated.

## 5. Test coverage notes

- Mutation-verified this round: Italian content drop → strong invariant red (`it` only); injected IPA fixture → no-notation table red; atlas row mangled → docs guard red; `curated`-derived order loop → red for `it`. Green-on-mutation reproduced for BR-6 (`Italiano`).
- `TestRenderLosesNothing/it` runs all six Italian words; `noRawNotationIn` and `renderLosesNothingIn` are both non-vacuous because `loadFakeDictionary` refuses an empty corpus.
- Unrun here: the live half. `TestFixturesMatchLiveDictionary/{en,es,it}` and `TestSelectedDictionaryAnswersInItsOwnLanguage/{es,it}` all SKIP — `en` skips too, so it is the harness, not the change. `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define/` from a real session is still the only place `s.f.` and `sycophantic → ErrNoEntry` are exercised.
- `langHelp` has no test. I judged BR-1 addressed on the derivation plus the observed `-h` output rather than on a test, because a test asserting `langHelp` contains every `curated` key restates the implementation; the pin worth having is on the *registration* (site 6 above).

## 6. Architectural notes

- **ARCH-DRY — flag.** Site 3 (`installedOnThisMachine()` vs the locally re-declared Italian `dictMeta` literals) and the subsumed weak sibling. The conformance table, `noRawNotationIn`, `renderLosesNothingIn` and the two-row no-notation table are all correct consolidations.
- **ARCH-PURE — pass.** `chooseDictionary` / `monolingualIn` remain pure over `dictMeta`; `langHelp` is a pure init-time computation over a package var (Go's initialization-order analysis makes the `curated` dependency safe). Test-side file reads are the correct seam, not business logic in IO.
- **ARCH-PURPOSE — flag, and this is the blocking axis.** The shadow-sweep of `curated`'s consumers now passes on `chooseDictionary`, `capture.sh`, README, the corpus, and the flag help; it fails on the six sites tabulated above. A commit message that names the class and a diff that closes two sites is the instance-not-class shape this principle describes, and the ledger's repeated `curated-consumer-unpinned` slug is the signal. `workshop/lessons.md` gained the checkout-trap sharpening this round but nothing about the rule that fired here despite being written down twice — per AGENTS.md §4 that entry is owed.
- **ARCH-MOCK — pass on the seam, flag on the model.** Italian enters through the existing `Dictionary` seam with `fakeDictionary` behind it and `TestFixturesMatchLiveDictionary` as the live half; `capture.sh` captures through the identifier `chooseDictionary` selects and says why. The gap is site 3: `installedOnThisMachine()` is the *fake of the host's dictionary set*, `#31` measured six more books installed (`it.Devoto-Oli`, `OxfordItalian`, `fr.Multi`, `de.DDDSI`, `OxfordFrench`, `OxfordGerman`), and none reached the fixture — so `TestChooseDictionaryIgnoresDictionariesThatDoNotIndexTheLanguage:178`'s rationale ("nothing installed indexes French") is now measurably false about this machine.
- **For `#34`:** closing sites 1-5 is what makes adding `fr`/`de` one row plus a red build that lists everything owing an update. Leaving them means `#34` repeats this sweep a fifth time.

## 7. Plan revision recommendations

Still owed in full — round 1's four recommendations are all unlanded — plus two new ones. One `## Revisions` entry in `workshop/plans/000031-curate-dictionaries-plan.md` covering:

1. **Task 5's `**Files:**` line (`plan:148`)** names `doc_sync_test.go` and `dict_fake_test.go`, neither modified; the guards landed in `cmd/define/dictselect_test.go`, and this round also touched `cmd/define/invariant_test.go` and `cmd/define/voice.go`.
2. **Task 5's first bullet is ticked but its "fix both comments" instruction was correctly not followed** — widening made both `capture.sh:174` and `rawnotation_test.go` true. Amend the task text so no reader hunts for an edit that should not exist.
3. **D5 needs a scope correction.** It reasons about "the README's book list" and concluded with a README-only guard; the boundary also added an atlas table (now covered) and the `-lang` flag help (a consumer D5 did not consider at all, since fixed as `langHelp`).
4. **The Done-when coverage table's row 3** should now cite `TestRenderLosesNothing` (widened this round, `invariant_test.go:169`) as the strong pin, and drop the claim that `TestRenderLosesNothingInEveryCapturedLanguage` is what covers no-data-loss.
5. **New — add a Core-concepts row for `langHelp`** (`cmd/define/voice.go`, PURE, new). The table guard catches stale rows, not absent ones.
6. **New — record the class that remains open**, with the six-site enumeration above, so `#34` inherits the list rather than rediscovering it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      langHelp derives from slices.Sorted(maps.Keys(curated)); built binary prints "en, es, it". No test pins the registration — folded into the new enumeration as site 6.
  - id: BR-2
    disposition: not-addressed
    note: |
      Atlas half fixed and verified red on mutation; the third site it named (dict_conformance_test.go:184 own-language table, no curated cross-check) is untouched.
  - id: BR-3
    disposition: addressed
    note: |
      Consolidated into one table at render_test.go:364; both comments now describe what they sit above.
  - id: BR-4
    disposition: addressed
    note: |
      One predicate, two rows, per-language why preserved; verified red on an injected IPA-bearing it fixture.
  - id: BR-5
    disposition: addressed
    note: |
      Verified by mutation — dropping an Italian-only marker in Render reddens only TestRenderLosesNothing/it/*.
  - id: BR-6
    disposition: not-addressed
    note: |
      Still strings.Contains(span, name); reproduced green with the atlas row renamed to "Italiano".
  - id: BR-7
    disposition: not-addressed
    note: |
      dictselect_test.go:428 still re-globs rather than calling loadFakeDictionary.
  - id: BR-8
    disposition: not-addressed
    note: |
      atlas/define.md:1303 still has no blank line before the heading; the same insertion also left a double blank line at :1269.
  - id: BR-9
    disposition: not-addressed
    note: |
      No Revisions entry landed; plan:148 Files line unchanged; langHelp also has no Core-concepts row.
findings:
  - id: new
    severity: Important
    family: curated-consumer-unpinned
    title: |
      the class was declared closed in 19b5ea1 but the enumeration was never written; six sites remain, two wrong today
    detail: |
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
  - id: new
    severity: Minor
    family: one-predicate-two-spellings
    title: |
      the widened strong invariant subsumes TestRenderLosesNothingInEveryCapturedLanguage, whose comment is now false
    detail: |
      3rd in family. Do NOT just delete this one — the rule: when a check is widened to the
      dimension its narrower sibling existed to cover, the sibling is deleted in the SAME commit.
      dict_fake_test.go:297 asserts only a non-empty render over capturedLanguages, which
      TestRenderLosesNothing (invariant_test.go:169) now covers with exact alnum counts over the
      same set, so it can no longer fail first; and its doc comment at :290 still claims
      TestRenderLosesNothing "goes through testDict(t), which is English by definition, so the
      Spanish captures were never run through the parser and renderer at all", which this round
      made false. Enumerated at HEAD: this is the only site.
```

---

## Re-review — 2026-08-29T13:26:29-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 31 — curate the Italian dictionary so /lang it is a real mode (French and German split to #34) |
| repo | tools |
| issue file | workshop/issues/000031-curate-dictionaries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83134e464ba38dce607296341a01fa9cc5d343f7..9b32bfbc52d1d0caf35d6f59ba103ceeff147e1d |
| command | sdlc close --issue 31 |
| reviewer | claude |
| timestamp | 2026-08-29T13:26:29-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3. The tree is behaviourally correct and the Done-when rows are genuinely pinned: I re-verified five guards by mutation in a scratch worktree of `9b32bfb` and watched each redden (dropping the Devoto-Oli record from `installedOnThisMachine()` fails `TestTheMeasuredSetModelsEveryCuratedLanguage`, `TestChooseDictionaryDoesNotDependOnOrder` *and* `TestChooseDictionaryPicksTheCuratedItalian`; renaming the atlas row to `Italiano` fails `TestDocsNameEveryCuratedLanguage`; deleting the `it` row from the own-language table fails `TestSelectedDictionaryAnswersInItsOwnLanguage`; reverting `fs.String("lang", "", langHelp)` to the old literal fails `TestTheLangFlagRegistersTheDerivedHelp`). `go test ./...`, `go vet ./...` and `gofmt -l` are clean. What stops SHIP is two measured residues, both of the class this gate has been circling: (a) BR-10's site 6 pinned *delivery* of the derivation, not the derivation — replacing `langHelp`'s body with the stale literal `"…en, es…"` and dropping three imports leaves the whole suite green (measured); and (b) this issue renamed one test and deleted another without feeding `retiredSymbolNames`, so `atlas/define.md:1181` and the plan now name tests the tree does not declare — the repo's own mechanism for exactly this was in place and unfed. BR-6 is also still open as measured.

## 1. Strengths

- **The `installedOnThisMachine()` cross-check is the highest-leverage guard in the diff** (`cmd/define/dictselect_test.go:59`). Removing the Devoto-Oli record reddens three tests at once, which is what "a language curated but unmodelled silently drops out of all of them" was supposed to buy — and it does.
- **The own-language conformance check became a table with a both-directions cross-check** (`cmd/define/dict_conformance_test.go:184-215`). Verified red when the `it` row is removed while `curated` keeps it. `#34` cannot add `fr`/`de` and acquire no live check.
- **The atlas half of `TestDocsNameEveryCuratedLanguage` is real, not decorative** (`dictselect_test.go:515`, `:545`). The word-boundary regex genuinely rejects `Italiano` — I reproduced the near-miss and it fails.
- **BR-11 was disposed the right way.** `TestRenderLosesNothingInEveryCapturedLanguage` is deleted with a tombstone and a subsumption argument (`dict_fake_test.go:291`), and the widened `TestRenderLosesNothing` really does emit `en`/`es`/`it` subtests (verified with `-v`) — so the sibling was not merely retired, its coverage moved.
- **The corpus matches its stated rationale.** `entries/it/parlare.txt` really opens `parlare 1 (par·là·re) s.m.` (homograph number before the syllabification, as `capture.sh` claims), `essere.txt` is 11.6 KB, and `pizza.txt` carries `s.f.` — the marker the live conformance row asserts.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `atlas/define.md:1181` and the plan name tests the tree no longer declares; the `retiredSymbolNames` row was never added.** This round renamed `TestSpanishEntriesCarryNoPronunciationNotation` → `TestNonEnglishEntriesCarryNoPronunciationNotation` and deleted `TestRenderLosesNothingInEveryCapturedLanguage`. `cmd/define/repo_guard_test.go:719` exists precisely to sweep those restatements, and its own comment says "*a rename cannot be detected automatically — only the person doing it knows the old name … Adding the row is the whole discipline*". No row was added, so the guard stayed green over stale prose. **Measured:** adding the two rows in a scratch copy turns `TestNoArtifactNamesARetiredSymbol` red on `atlas/define.md` and twice on `workshop/plans/000031-curate-dictionaries-plan.md`. A third stale name, `TestItalianEntriesCarryNoPronunciationNotation` (plan Done-when row 4), never existed at any commit. `workshop/issues/000030-clickable-regions.md:69` also cites the old name as its evidence row — the guard exempts issues as records, but #30 is the live consumer this issue exists to unblock, so a reader will grep for a test that is gone. Fix sketch: add the two `retiredSymbolNames` rows, sweep the three names out of `atlas/define.md` + the plan's current-truth sections. **Rule, since the human half has now failed twice** (`#27` is recorded in that same map): make it mechanical — a guard that reads removed `^func Test…`/`^func …` declarations out of the review window with `git` (the machinery is already in this file, `TestPlanTableStatusMatchesTheChangeWindow` shells to git) and requires either a `retiredSymbolNames` row or zero current-truth mentions.

**I-2 — the class is still enforced per-site, and two hand-written enumerations remain. *This is the 5th finding in family `curated-consumer-unpinned`.*** Earlier rounds fixed instances; per the escalation, do **not** patch these two — state the rule and build it. Full shadow-sweep of `curated`'s consumers at HEAD (ARCH-PURPOSE), which is the enumeration the class claim needs:

| # | consumer | derives? | pinned against `curated`? |
|---|---|---|---|
| 1 | `chooseDictionary` (`dictselect.go:134`) | yes | yes — order test + conformance policy check |
| 2 | `langHelp` (`voice.go:158`) | yes | **delivery only** — see BR-10 disposition |
| 3 | README span | n/a | yes (weak — see BR-6) |
| 4 | atlas span | n/a | yes, verified |
| 5 | `capture.sh` ids | n/a | yes, both directions |
| 6 | `installedOnThisMachine()` | n/a | yes, verified |
| 7 | own-language conformance table | n/a | yes, verified |
| 8 | `testdata/entries/<lang>/` | n/a | yes |
| 9 | `TestNonEnglishEntriesCarryNoPronunciationNotation`'s `{es,it}` table (`render_test.go:366`) | no | **no** — `#34` can curate `fr` and this language-keyed table silently gains no row |
| — | the doc list `{README, atlas}`, spelled 3× (`dictselect_test.go:515`, `doc_sync_test.go:125`, `:142`) | no | **no** — a third doc is covered only by whichever test its author remembered |

The rule: **`curated` gets ONE registry of the surfaces obliged to name every curated language, and the registry supplies each surface's TEXT.** Then the assertion is `for surface in registry { for lang in curated { assert lang appears in surface.text } }` — README span, atlas span and `langHelp` all become rows, replacing a literal derivation fails (which today it does not), and the only hand-written list left is "which surfaces exist". Row 9 needs the second half of the rule: a language-keyed predicate table is cross-checked against `curated` **with an explicit exempt list** (German will need one — its Duden field is real, so an `IPA == ""` row would be wrong), because "no row" and "deliberately no row" are currently indistinguishable.

## 4. Minor findings

- `dictselect_test.go:475` — a bare `{ … }` block around a single `t.Errorf`; the whole thing reads as `if err != nil || len(d.entries) == 0 { … }`.
- `atlas/define.md:1269-1270` — double blank line; and the standing-limitation paragraph is raggedly re-wrapped ("…held five. The live / ratchet in `live_property_test.go`…").
- `atlas/define.md:1288` — "**the raw-notation ratchet is ENGLISH-ONLY**" is contradicted by its own next sentence ("sweeps every captured language since `#31`"); the English-only one is the *live* `TestRenderLosesNothingOverLiveEntries`, so name it.
- `TestDocsNameEveryCuratedLanguage` lives in `dictselect_test.go` while the doc-sync family it models itself on lives in `doc_sync_test.go`.

## 5. Test coverage notes

- **Live rows could not be executed here.** DictionaryServices is unreachable from this review process — even `en` reports "no known en dictionary is installed" — so `TestSelectedDictionaryAnswersInItsOwnLanguage/it` and `TestFixturesMatchLiveDictionary/it` both SKIP. Everything reachable off the fake was verified. The operator's `--verified` should carry the unsandboxed `-tags conformance` evidence (and ideally `CONFORMANCE_STRICT=1`, since a skip is otherwise green).
- The per-language sweep harness is now spelled three times (`invariant_test.go:170`, `render_test.go:242`, `:326`) — `for lang := range capturedLanguages(t) { t.Run(string(lang), func(t){ xIn(t, testDictFor(t, lang)) }) }`. Two of the three are new in this diff. One `forEachCapturedLanguage(t, fn)` helper collapses them (ARCH-DRY).
- Coverage of the bug class this diff could ship (a new corpus checked by the weak invariant only) is genuinely closed: the strong alnum-count invariant now runs `it` subtests.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** Three copies of the sweep harness and three copies of the `{README, atlas}` doc list (see I-2 and §5).
- **ARCH-PURE — pass.** `langHelp` is a pure derivation over `curated`; the widened sweeps are pure over committed fixtures; DictionaryServices stays behind the `Dictionary` seam. No business logic moved into IO.
- **ARCH-PURPOSE — flag (I-2).** The purpose was "every consumer derives from `curated`". Nine consumers enumerated, seven enforced. A guard that pins *delivery* of a derivation while leaving the derivation replaceable by a literal is the "easy subset" shape this axis names — and the plan's own Revisions table already claimed site 6 closed.
- **ARCH-MOCK — pass.** No new external dependency. `capture.sh` → fixtures → `fakeDictionary` behind the same seam production uses, with `TestFixturesMatchLiveDictionary/it` and the own-language row as the live conformance half. `#34` should note that adding `fr`/`de` costs a corpus, a fixture record, an own-language row and a notation-table decision — all four are now enforced, which is the point.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000031-curate-dictionaries-plan.md`:

- **Stale test names.** The Architecture paragraph, D3, D4, Task 1 Step 3, Task 5 bullets 1–2 and Done-when rows 3–4 name `TestRenderLosesNothingInEveryCapturedLanguage` (deleted), `TestSpanishEntriesCarryNoPronunciationNotation` (renamed to `TestNonEnglishEntriesCarryNoPronunciationNotation`) and `TestItalianEntriesCarryNoPronunciationNotation` (never existed — it landed as a subtest of the merged table). Row 3's no-data-loss pin is now `TestRenderLosesNothing` (widened); row 4's is `TestNonEnglishEntriesCarryNoPronunciationNotation/it`.
- **Task 5 bullet 1** is ticked while instructing a `capture.sh` / `rawnotation_test.go` comment fix that was correctly *not* performed (widening made both comments true). The implementation-notes Revisions entry says so; the bullet should point at it.
- **Site 6 of the round-2 table** ("all six closed") overstates: `langHelp`'s registration is pinned, its derivation is not. Record what was actually closed.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      TestDocsNameEveryCuratedLanguage now loops README + atlas (verified red on the atlas row), and the own-language table gained its cross-check (verified red).
  - id: BR-6
    disposition: not-addressed
    note: |
      Measured at HEAD: renaming the README bullet label to "- **Itaian** —" still passes, because the same bullet's next sentence says "Italian"; the word-boundary change only closed the "Italiano" near-miss.
  - id: BR-7
    disposition: addressed
    note: |
      TestEveryCuratedLanguageHasACorpus now goes through loadFakeDictionary, which also rejects zero-byte fixtures.
  - id: BR-8
    disposition: addressed
    note: |
      Blank line present before "## Source pronunciation"; a new double-blank appeared four lines earlier (raised as doc-formatting).
  - id: BR-9
    disposition: addressed
    note: |
      Task 5's Files line now names the files that actually changed, and the Revisions entry records why the comment fix was not performed.
  - id: BR-10
    disposition: not-addressed
    note: |
      Five of six sites verified red on removal; site 6 is half-closed — replacing langHelp's body with the stale literal "en, es" and dropping the three imports leaves go test ./... fully green (measured), which is the exact regression the finding described.
  - id: BR-11
    disposition: addressed
    note: |
      The subsumed test is deleted with a tombstone, and TestRenderLosesNothing verifiably emits en/es/it subtests.
findings:
  - id: new
    severity: Important
    family: retired-symbol-unswept
    title: |
      the rename/deletion this round performed never got its retiredSymbolNames row, so the atlas and the plan name tests the tree does not declare
    detail: |
      atlas/define.md:1181 names TestSpanishEntriesCarryNoPronunciationNotation (renamed to
      TestNonEnglishEntriesCarryNoPronunciationNotation); the plan names it plus the deleted
      TestRenderLosesNothingInEveryCapturedLanguage and a TestItalianEntriesCarryNoPronunciationNotation
      that never existed. Measured: adding the two rows to retiredSymbolNames
      (repo_guard_test.go:719) turns TestNoArtifactNamesARetiredSymbol red on atlas/define.md and
      twice on the plan. The map's own comment records this human half failing once before (#27),
      so the rule to build is mechanical: derive removed func declarations from the review window
      with git — TestPlanTableStatusMatchesTheChangeWindow already shells to git in this file — and
      require either a retiredSymbolNames row or zero current-truth mentions.
  - id: new
    severity: Important
    family: curated-consumer-unpinned
    title: |
      the class is still enforced per-site; two hand-written enumerations remain and the mechanism should be one surface registry
    detail: |
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
  - id: new
    severity: Minor
    family: one-predicate-two-spellings
    title: |
      the per-captured-language sweep harness is now spelled three times, two of them new in this diff
    detail: |
      This is the 4th finding in this family, so the fix is the helper, not the site. Measured
      prevalence: invariant_test.go:170, render_test.go:242, render_test.go:326 all read
      `for lang := range capturedLanguages(t) { t.Run(string(lang), func(t){ xIn(t, testDictFor(t, lang)) }) }`.
      One forEachCapturedLanguage(t, func(*testing.T, *fakeDictionary)) collapses all three, and a
      fourth sweep then cannot be written narrower than the corpus by accident.
  - id: new
    severity: Minor
    family: doc-formatting
    title: |
      atlas/define.md:1269-1270 has a double blank line and a raggedly re-wrapped paragraph
    detail: |
      This is the 2nd finding in this family (BR-8 was a missing blank line before a heading), so
      state the rule rather than patching the line: nothing in the repo checks markdown block
      spacing or wrap in atlas/ and README.md, and both instances were introduced by hand-editing
      prose. Either run a markdown formatter over the touched docs as part of the boundary, or
      accept the class explicitly. Also at :1288 the ragged rewrap left "…held five. The live /
      ratchet in live_property_test.go…".
  - id: new
    severity: Minor
    family: redundant-syntax
    title: |
      dictselect_test.go:475 wraps a single t.Errorf in a bare block instead of inverting the guard
    detail: |
      `if err == nil && len(d.entries) > 0 { return }` followed by `{ t.Errorf(...) }` reads as
      `if err != nil || len(d.entries) == 0 { t.Errorf(...) }`.
```

---

## Re-review — 2026-08-29T13:54:07-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 31 — curate the Italian dictionary so /lang it is a real mode (French and German split to #34) |
| repo | tools |
| issue file | workshop/issues/000031-curate-dictionaries.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83134e464ba38dce607296341a01fa9cc5d343f7..dcbb683a3144231363c888b1b11af4564b91a919 |
| command | sdlc close --issue 31 |
| reviewer | claude |
| timestamp | 2026-08-29T13:54:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's actual purpose is delivered and genuinely pinned: `"it": {"com.apple.dictionary.it.Devoto-Oli"}` lands, a real 6-entry Devoto-Oli corpus (27.9 KB) is committed, `define -h` prints `en, es, it`, and I verified the two round-3 mechanisms by mutation in a scratch clone rather than taking the commit messages at face value — replacing `langHelp`'s body with the stale literal reddens `TestEverySurfaceNamesEveryCuratedLanguage/the -lang flag help` (BR-10's residual site 6, closed), and injecting a stale test name into `atlas/define.md` reddens `TestARemovedDeclarationIsSweptOrRetired` (BR-12's mechanism, real). `go test ./...`, `go vet ./...` and `gofmt -l` are clean. What keeps this off SHIP is that the *fix commits themselves* shipped three new instances of the two families this gate has been circling: the new retired-name guard matches with `strings.Contains` where its sibling uses `\b`, so a scratch rename of the 3-letter helper `ids` produced seven false failures including one on the substring "forbids"; and both helpers introduced to collapse duplication — `derivedDocs` and `currentTruthFiles` — have zero and one call site respectively, while the spellings they were meant to replace are still in the tree, under comments asserting they were consolidated. None of this touches user-facing behaviour; all of it is a few lines.

## 1. Strengths

- **The registry is the right mechanism and it is load-bearing.** `curatedSurfaces()` (`cmd/define/dictselect_test.go:527`) supplying each surface's own *text* and its own *spelling* is what finally distinguishes delivery from derivation — verified: hardcoding `langHelp` to `"…en, es…"` keeps `TestTheLangFlagRegistersTheDerivedHelp` green and fails the registry row, exactly as `a9bbf97` claims.
- **`TestARemovedDeclarationIsSweptOrRetired` (`repo_guard_test.go:1085`) does what BR-12 asked for.** Splitting the *trigger* (git knows what disappeared) from the *mapping* (only the author knows the new name) is the correct decomposition, and it fires — with the two `retiredSymbolNames` rows removed and a stale mention re-injected, it goes red with a specific, actionable message.
- **`notationExempt` (`render_test.go:395`) makes "deliberately no row" legible.** Carrying the *reason* as the map value rather than using a bare set is the detail that will actually stop `#34` from adding German to a table asserting the opposite.
- **The corpus is real and chosen for structure, not vocabulary.** `essere` at 11.6 KB as the blob ceiling and `parlare 1` as number-before-syllabification are the two fixtures a sense-splitting regression would surface in; `capture.sh:170-186` documents each choice.
- **The `#34` handoff is written as measurement, not as a promise.** `dictselect.go:81-89` and `atlas/define.md:1284-1294` both carry the 245/6/5 sense counts and the 2,811/4,404-rune blobs, so the next issue starts from data rather than from "one line each" a second time.

## 2. Critical findings

None. No production correctness defect; the only production change is one map row and a derived help string, both verified against the built binary.

## 3. Important findings

**I-1 — `repo_guard_test.go:1120` matches removed declarations by substring, so an ordinary rename produces false failures, and the remedy it suggests poisons its sibling.** `strings.Contains(currentTruthOnly(string(b)), name)` — while `TestNoArtifactNamesARetiredSymbol:791` uses `regexp.MustCompile(`\b`+QuoteMeta(old)+`\b`)` for the same job. **Measured:** in a scratch clone I renamed the test helper `func ids` → `idsOfMetas` and committed; the guard failed on seven files, including `atlas/define.md`, which has **zero** word-boundary occurrences of `ids` — the hit was "for**bids**". The failure message then instructs the author to add `"ids": "idsOfMetas"` to `retiredSymbolNames`, which would make `TestNoArtifactNamesARetiredSymbol` permanently red on every file containing the word. Fix: use the same word-boundary regex, and restrict `gone` to declarations whose name is plausibly unique (e.g. `Test*`/exported, or ≥ some length) or the guard will keep tripping on short helpers.

**I-2 — the round-3 consolidation introduced two helpers that consolidate nothing.** *This is the 5th finding in family `one-predicate-two-spellings`.* Do NOT patch the two sites individually — the rule is: **a consolidation is complete only when the new helper's call-site count equals the number of spellings it replaced; grep for callers before disposing the finding.** Measured prevalence at HEAD:
- `derivedDocs` (`dictselect_test.go:487`), commented "*the doc set … named ONCE … It was spelled three times — here and twice in doc_sync_test.go*", has **zero** references anywhere in the tree. `doc_sync_test.go:125` and `:142` still spell `[]string{"../../README.md", "../../atlas/define.md"}` literally, and `curatedSurfaces()` spells the two paths separately at `:538` and `:543`. Package-level vars are exempt from Go's unused check, so this passes every suite while doing nothing — BR-13's sub-item (b) is not addressed.
- `currentTruthFiles` (`repo_guard_test.go:1134`), commented "*Shared with TestNoArtifactNamesARetiredSymbol so the two guards cannot disagree about what 'current truth' means*", has one caller — the new test. `TestNoArtifactNamesARetiredSymbol:759-780` still carries a byte-identical `binds` closure and `ls-files` loop. **The two copies have already diverged:** the inline one asserts `if seen == 0 { t.Fatal("…this test would pass vacuously") }`; the extracted one has no such assertion, so if `binds` ever stopped matching, `TestARemovedDeclarationIsSweptOrRetired` would pass over an empty file set. They also disagree on the matching rule (I-1). Fix: have `TestNoArtifactNamesARetiredSymbol` call `currentTruthFiles`, move the vacuity assertion into the helper, and either use `derivedDocs` at all four sites or delete it.

**I-3 — the own-language cross-check is behind `//go:build darwin && conformance`, so the pure half never runs in the default gate.** *This is the 6th finding in family `curated-consumer-unpinned`.* Do NOT add a row anywhere — the rule is: **the pure data half of a cross-check must live in an untagged file; only the live half needs the tag.** `dict_conformance_test.go:202`'s `for lang := range curated { … no row for it … }` needs no dictionary, no network and no macOS — it compares two in-memory tables — yet it is compiled only under `-tags conformance`, which the file's own header says "a CI runner does not" satisfy, and which `.github/workflows/merge-check.yml` does not pass. So the obligation "a curated language has a live own-language check" is enforced only for someone who both remembers to run the tagged suite and reads it. Fix: move the `rows` table and its cross-check into an untagged file (next to the registry), leaving only the `t.Run` lookup half tagged.

## 4. Minor findings

- **`dictselect_test.go:52`** — "*EVERY curated language is modelled in the fixture above, in BOTH directions*", but the loop below only checks `curated` → `installedOnThisMachine()`. The reverse is deliberately absent (the fixture carries bilingual decoys), so the comment, not the code, is what should change.
- **`capture.sh:11` and `:177`** — `mkdir -p entries/en entries/es entries/it` and the `echo`/`wc -c` summary are hand-maintained per-language enumerations that no registry row covers. `TestCaptureScriptUsesTheCuratedDictionaries` pins the *identifiers* in both directions but not the summary, so `#34` can add `fr` and get a summary that under-reports — the exact failure the new comment at `:172` says the summary exists to prevent. A whole-file token check won't work (`it` is an English word all over the script); a marked span or an explicit exemption would.
- **`repo_guard_test.go:1111`** — the skip message says "removed no top-level **declaration**" while the regex matches `^-func` only; a removed `var`/`const`/`type` is invisible. Narrow the message or widen the regex.

## 5. Test coverage notes

- Every Done-when row is pinned by a named test I saw emit real subtests: `TestRenderLosesNothing` and `TestNoRawPronunciationNotationSurvives` both now run `en`/`es`/`it` (6 Italian entries each), `TestEveryCuratedLanguageHasACorpus` runs all three through `loadFakeDictionary` (BR-7's zero-byte rejection included), and `TestNonEnglishEntriesCarryNoPronunciationNotation` runs `es`/`it` untagged.
- **Verification limit, same as prior rounds:** DictionaryServices is unreachable from this review process — `TestSelectedDictionaryAnswersInItsOwnLanguage/{es,it}` both `SkipOrFail` with "chooseDictionary fell back to the NULL search", so the live Italian rows (`pizza` differs from NOAD, carries `s.f.`, `sycophantic` returns `ErrNoEntry`) could not be executed here. Everything reachable off the fake was run.
- The kind of bug this diff could ship — a Devoto-Oli shape (`•` sub-senses, `A./B.` blocks, `ETIMOLOGIA`/`DATA`) silently dropping content — is now covered by the strong alnum-count invariant over `it`, which is the right widening.

## 6. Architectural notes

- **ARCH-DRY — flag.** I-2 (two helpers with zero/one callers while their duplicates remain) and BR-14 (three verbatim `for lang := range capturedLanguages(t) { t.Run(…) }` harnesses at `invariant_test.go:170`, `render_test.go:243`, `render_test.go:327`). The bodies were extracted; the harness was not.
- **ARCH-PURE — pass.** `langHelp` is a pure derivation of `curated` evaluated at init with no IO; `curated`/`chooseDictionary`/`monolingualIn` stay pure and are unit-tested without mocks. The one wrinkle is I-3: a *pure* assertion placed behind an IO-gated build tag.
- **ARCH-PURPOSE — mostly pass.** The shadow-sweep over `curated`'s consumers now finds nine, of which the registry or a direct cross-check enforces eight; the residuals are `capture.sh`'s summary (Minor above) and I-3's tag-gating. The purpose — `/lang it` as a real mode — is delivered, not deferred.
- **ARCH-MOCK — pass.** No new external dependency: DictionaryServices stays behind `fakeDictionary`, `TestFixturesMatchLiveDictionary` is the live conformance half with a documented on-demand cadence, and the Italian books were added to `installedOnThisMachine()` rather than re-declared locally, which is the right direction. `TestTheMeasuredSetModelsEveryCuratedLanguage` now prevents the fake from silently modelling a machine production no longer targets.

## 7. Plan revision recommendations

The plan matches the code at HEAD — Core-concepts rows (`curated` modified, `chooseDictionary`/`monolingualIn` unchanged, `langHelp` new in `voice.go`) all check out, and `TestPlanTableStatusMatchesTheChangeWindow` passes over them. One entry is worth appending once I-1/I-2/I-3 are dealt with:

> **### 2026-08-29 — close review round 4.** Round 3's own fixes carried three instances of the families they were closing: the new retired-name trigger matched by substring rather than word boundary (seven false hits on a scratch rename of `ids`, one of them on "forbids"), and both consolidation helpers — `derivedDocs` and `currentTruthFiles` — landed with their old spellings still in the tree and comments claiming otherwise. The rule this adds: **a consolidation is not disposed `addressed` until the helper's call-site count equals the number of spellings it replaced**, and a guard's *matching rule* is part of what two guards sharing a scope must share.

```findings
dispose:
  - id: BR-6
    disposition: not-addressed
    note: |
      Re-measured at HEAD: renaming the README bullet to "- **Itaian** —" leaves TestEverySurfaceNamesEveryCuratedLanguage green, because the same bullet's next sentence says "Italian". Minor, non-blocking.
  - id: BR-10
    disposition: addressed
    note: |
      Site 6 closed by the registry row: replacing langHelp's body with the literal "en, es" now reddens TestEverySurfaceNamesEveryCuratedLanguage/the -lang flag help (verified by revert in a scratch clone).
  - id: BR-12
    disposition: addressed
    note: |
      Rows added and all three artifacts swept (incl. workshop/issues/000030). Mechanism verified live: removing the two rows and re-injecting a stale mention turns TestARemovedDeclarationIsSweptOrRetired red. Its matching rule is defective — raised separately.
  - id: BR-13
    disposition: not-addressed
    note: |
      The registry (part a) is delivered and real; part (b) is not — derivedDocs at dictselect_test.go:487 has zero references while doc_sync_test.go:125 and :142 still spell the doc list literally.
  - id: BR-14
    disposition: not-addressed
    note: |
      Bodies were extracted but the harness is still spelled three times verbatim: invariant_test.go:170, render_test.go:243, render_test.go:327. No forEachCapturedLanguage helper exists.
  - id: BR-15
    disposition: not-addressed
    note: |
      Instances mostly repaired (double blank gone), but no rule was stated and none accepted; atlas/define.md:1273 (94 cols) and :1303 (87) still break the file's ~80-col wrap.
  - id: BR-16
    disposition: addressed
    note: |
      dictselect_test.go:472 now reads `if err != nil || len(d.entries) == 0 {`.
findings:
  - id: new
    severity: Important
    family: retired-symbol-unswept
    title: |
      the new removed-declaration guard matches by substring, so an ordinary rename fires seven false failures and its suggested remedy poisons the sibling guard
    detail: |
      repo_guard_test.go:1120 uses strings.Contains, while TestNoArtifactNamesARetiredSymbol:791
      uses a word-boundary regex for the same question — the two guards disagree on the matching
      rule that currentTruthFiles' own comment says they cannot disagree about. Measured in a
      scratch clone: renaming the helper `func ids` to `idsOfMetas` and committing turns the guard
      red on seven paths including atlas/define.md, which has ZERO word-boundary occurrences of
      "ids" (the hit is "forbids"). The message then tells the author to add a retiredSymbolNames
      row for "ids", which would make TestNoArtifactNamesARetiredSymbol permanently red on every
      file containing that word. Fix: share the word-boundary regex, and bound `gone` to names
      unlikely to collide (Test*/exported, or a length floor).
  - id: new
    severity: Important
    family: one-predicate-two-spellings
    title: |
      both helpers introduced to collapse duplication have zero and one call site while the spellings they replace remain, under comments claiming the consolidation happened
    detail: |
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
  - id: new
    severity: Important
    family: curated-consumer-unpinned
    title: |
      the own-language table's curated cross-check is pure data but sits behind the conformance build tag, so it never runs in go test ./... or CI
    detail: |
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
  - id: new
    severity: Minor
    family: curated-consumer-unpinned
    title: |
      capture.sh's mkdir line and closing summary enumerate languages by hand and no registry row covers them
    detail: |
      capture.sh:11 (`mkdir -p entries/en entries/es entries/it`) and :177 (the echo/wc summary)
      are per-language enumerations. TestCaptureScriptUsesTheCuratedDictionaries pins the
      IDENTIFIERS in both directions but not these, so a fourth language yields a summary that
      under-reports — the exact failure the comment added at :172 says the summary exists to
      prevent. A whole-file token check cannot work ("it" is an English word throughout the
      script), so this needs a marked span or an explicit exemption with its reason, the way
      notationExempt makes "deliberately no row" legible.
  - id: new
    severity: Minor
    family: comment-overclaims-code
    title: |
      dictselect_test.go:52 says the fixture is cross-checked "in BOTH directions" but only curated to fixture is implemented
    detail: |
      TestTheMeasuredSetModelsEveryCuratedLanguage loops curated and asserts each id is modelled;
      nothing checks the reverse. The reverse is deliberately absent — the fixture carries
      bilingual and thesaurus decoys — so the comment should change, not the code. Same shape as
      the two false structural claims in the Important finding above ("named ONCE", "Shared
      with"), all three introduced by the rounds that were closing findings about exactly this.
  - id: new
    severity: Minor
    family: retired-symbol-unswept
    title: |
      the removed-declaration guard's skip message says "top-level declaration" but the regex matches only ^-func
    detail: |
      repo_guard_test.go:1111 reports "this window removed no top-level declaration" while the
      regex at :1093 matches `^-func` only, so a removed var/const/type is invisible. Narrow the
      message or widen the regex.
```
