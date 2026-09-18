# #76 Delete the orphaned source-provenance chain: implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete every piece of `define`'s per-fragment source-language provenance that no production path reads, along with the tests that pin only it. Output stays byte-identical.

**Architecture:** This is a deletion only, with no new behaviour. #65 threaded source offsets through the NOAD parser and language spans through the Oxford parser so that each fragment could be tinted by its own language. #70 deleted that tint, the chain's only reader. Everything that fed it is now dead data. The one live piece caught up in it is `projectLanguageText`, which `practice_output.go` still uses in its display mode. It stays and loses its strict-mode flag.

**Tech Stack:** Go 1.x, `cmd/define` (package `main`), `go test`, the `conformance` build tag for live macOS dictionary checks.

**Decision (operator, 2026-09-18):** delete, don't re-use. The operator's words: "generally, I'd like to delete unused features." No open issue needs per-fragment ownership, and git history keeps it (`62c6a66`, `77a5cfe`).

---

## Core concepts

### Pure entities

Everything here is pure: parse and render functions over strings, with no IO.

| Name | Lives in | Status |
|------|----------|--------|
| `projectDictionaryText` | `cmd/define/dictionary_language.go` | deleted |
| `bilingualLanguageText` | `cmd/define/dictionary_language.go` | deleted |
| `projectLanguageText` | `cmd/define/dictionary_language.go` | deleted |
| `sourceOffset` | `cmd/define/parse.go` | deleted |
| `advanceSource` | `cmd/define/parse.go` | deleted |
| `RenderOpts.Language` | `cmd/define/render.go` | deleted |
| `Entry.source` | `cmd/define/parse.go` | deleted |
| `Sense.sourceAt` | `cmd/define/parse.go` | deleted |
| `Sense.sourceKnown` | `cmd/define/parse.go` | deleted |
| `Example.sourceAt` | `cmd/define/parse.go` | deleted |
| `Example.sourceKnown` | `cmd/define/parse.go` | deleted |
| `definitionSection.source` | `cmd/define/definitions.go` | deleted |
| `bilingualDocument.native` | `cmd/define/bilingual_layout.go` | deleted |
| `bilingualNode.lang` | `cmd/define/bilingual_layout.go` | deleted |
| `projectDisplayText` | `cmd/define/language_text.go` | new (moved from `dictionary_language.go`, strict mode folded away) |
| `RenderOpts` | `cmd/define/render.go` | modified |
| `Entry` | `cmd/define/parse.go` | modified |
| `Sense` | `cmd/define/parse.go` | modified |
| `Example` | `cmd/define/parse.go` | modified |
| `ParseEntry` | `cmd/define/parse.go` | modified |
| `parseBlocks` | `cmd/define/parse.go` | modified |
| `parseSenses` | `cmd/define/parse.go` | modified |
| `newSense` | `cmd/define/parse.go` | modified |
| `newExample` | `cmd/define/parse.go` | modified |
| `definitionSection` | `cmd/define/definitions.go` | modified |
| `renderDefinitionOutput` | `cmd/define/definitions.go` | modified |
| `spanishDefinitions.supplement` | `cmd/define/definitions.go` | modified |
| `bilingualNode` | `cmd/define/bilingual_layout.go` | modified |
| `bilingualDocument` | `cmd/define/bilingual_layout.go` | modified |
| `parseBilingualDocument` | `cmd/define/bilingual_layout.go` | modified |
| `renderBilingualDocument` | `cmd/define/bilingual_layout.go` | modified |

**The whole chain, reader-first.** No production code reads `Entry.source`, `Sense.sourceAt`/`sourceKnown` or `Example.sourceAt`/`sourceKnown`. Their only reader was `RenderOpts.dictionaryText`, which #70 deleted. Everything upstream exists only to fill them:

- **NOAD parser.** `ParseEntry` → `parseBlocks` → `newBlock` → `parseSenses` → `newSense` → `newExample` carry a variadic `sourceBase ...int` plus `sourceOffset`/`advanceSource`. This is **every** change `parse.go` has had since `62c6a66^`: `diff <(git show 62c6a66^:cmd/define/parse.go) cmd/define/parse.go` shows only offset lines (measured 2026-09-18; the only other lines are gofmt realigning the widened structs). So the parser half is restoring that file exactly, and the check is an empty diff.
- **Oxford parser.** In `parseBilingualDocument`, a class→language switch sets `bilingualNode.lang`. The leaves feed `bilingualDocument.source.spans`, `projectDictionaryText` turns those into `bilingualDocument.native`, and `native` goes to `definitionSection.source`, then to `Entry.source`, and stops there. `renderBilingualDocument` reads only `doc.source.text` and the node classes, never `.lang` or `.spans`. So `bilingualDocument.source` becomes a plain `string`, because a `languageText` whose spans nobody sets is a string wearing a type (lessons: *A declaration nothing reads is a comment with a type*).
- **`definitions.go`.** `ro.Language = section.language` writes a field nothing reads. `ro.Tint = tintPolicy{}` zeroed the tint so the deleted fragment painter could not fire, and nothing under `Render` or `renderBilingualDocument` reads `.Tint` now (`git grep '\.Tint\b'` over production files finds only `renderDefinitionOutput`'s own section-level read of `opt.Tint`, measured 2026-09-18). Both lines go.

The issue listed six members. This table has more because ARCH-PURPOSE asks for the class, not the listed instances. The extras are `bilingualNode.lang` and its switch, `doc.source`'s spans, `projectLanguageText`'s strict flag, the parser's variadics and helpers, and the dead `ro.Tint` write.

- **`projectDisplayText`** is the survivor. `renderPracticeOutput` (`practice_output.go:51-52`) projects ownership and answer exclusions through the wrap with it.
  - **Relationships:** 1 caller file, 2 call sites.
  - **DRY rationale:** `projectLanguageText(…, displayWhitespace bool)` had two modes because it had two callers. With the strict caller gone, the flag is always `true`, so the `begin == i && !displayWhitespace` refusal is dead. It is folded into one function, `projectDisplayText(source languageText, rendered string) languageText`, which moves to `language_text.go` beside the type it projects (ARCH-DRY). `dictionary_language.go` would hold nothing dictionary-related, so it is deleted.
  - **Future extensions:** none planned.

### Integration points

None changes. The one seam nearby is the native Oxford record source (`recordSource`, implemented by `nativeSpanishEnglishSource`, with its stateful fake `fakeRecordSource` and its live `conformance` checks). This work narrows what the parser extracts from a record. It does not touch the seam, the fake, or any validation that runs on untrusted native HTML: bounds, identity, and Text correspondence all stay.

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `nativeSpanishEnglishSource` | `cmd/define/bilingual_darwin.go` | unchanged | macOS Dictionary Services |

### Tests

| Name | Lives in | Status |
|------|----------|--------|
| `TestDisplayProjectionOwnsOnlyMatchingGlyphs` | `cmd/define/language_text_test.go` | new |
| `FuzzDisplayProjection` | `cmd/define/language_text_test.go` | new |
| `TestDictionaryDefinitionsSectionTintKeepsRegions` | `cmd/define/dictionary_language_test.go` | new |
| `TestDictionaryDuplicateRecordSelectionIsDeterministic` | `cmd/define/dictionary_language_test.go` | new |
| `TestDictionarySourceProvenanceCorpus` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryParserSourceOffsets` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryProjectionExactOccurrenceAndFallback` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryProjectionDoesNotJoinWords` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryUnprovenSectionAlignmentStaysNeutral` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryDefinitionsRetainSourceAndRegions` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryDuplicateRecordOwnershipIsDeterministic` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestBilingualNativeLanguageOwnership` | `cmd/define/bilingual_conformance_test.go` | deleted |
| `TestDictionaryMonolingualOriginAndDisabledTint` | `cmd/define/dictionary_language_test.go` | modified |
| `TestLockedDictionaryPreservesSupplementalCapability` | `cmd/define/dict_test.go` | modified |
| `TestDefinitionUnknownSourceDoesNotInheritStudyLanguage` | `cmd/define/dictionary_source_test.go` | modified |
| `TestDictionaryAssemblyOwnsVerifiedPrimaryLanguage` | `cmd/define/dictionary_source_test.go` | modified |
| `TestTerminalSerializationPreservesOverlongDictionarySource` | `cmd/define/terminal_serialization_test.go` | modified |
| `TestOxfordNativeTreeAndLeafConservation` | `cmd/define/bilingual_layout_test.go` | modified |
| `TestOxfordRejectsUnprovenRecords` | `cmd/define/bilingual_layout_test.go` | modified |
| `FuzzOxfordDocument` | `cmd/define/bilingual_layout_test.go` | modified |

What happens to each test, and why (lessons: *Deleting a test needs the same evidence as writing one*):

- **Deleted, because they pin only the chain.** These are `TestDictionarySourceProvenanceCorpus`, `TestDictionaryParserSourceOffsets`, `TestDictionaryUnprovenSectionAlignmentStaysNeutral` (it pins the `section.source[i].text == raw` guard) and `TestBilingualNativeLanguageOwnership` (a live check that ownership ranges exist). The installed-Oxford parse keeps its live check, `TestBilingualNativeRendirLayout`.
- **Deleted, with the surviving half ported.** `TestDictionaryProjectionExactOccurrenceAndFallback` and `TestDictionaryProjectionDoesNotJoinWords` call `projectDictionaryText`. Their "changed word → neutral" and "joined words → neutral" cases run through the shared loop that `projectDisplayText` keeps. No test pins that loop's glyph-mismatch branch in either mode. Measured 2026-09-18: with M-join (Task 3) applied to today's code, the only reds are the three sandbox pty tests and `TestPlanCitesTestsThatExist`, and that one is red only because this untracked plan exists. Both cases are caught by the length guards, not by the glyph branch. Those cases move to `TestDisplayProjectionOwnsOnlyMatchingGlyphs`, along with the same-length substitution that does reach the glyph branch and the display-only case: an inserted break keeps ownership. That last case is the branch the fold makes unconditional. `TestPracticeLongRunOwnershipFollowsPhysicalRows` already reddens under M-break (measured 2026-09-18). The unit case pins the same branch at the function's own level. The dictionary-mode "inserted space → neutral" case retires with the strict mode.
- **Renamed, because the name described the deleted concept.**
  - `TestDictionaryDefinitionsRetainSourceAndRegions` becomes `TestDictionaryDefinitionsSectionTintKeepsRegions`. The body is kept minus `Language:`, and its failure text changes from "provenance changed region behavior" to "tint changed region behavior".
  - `TestDictionaryDuplicateRecordOwnershipIsDeterministic` becomes `TestDictionaryDuplicateRecordSelectionIsDeterministic`. The body is kept. Which duplicate is selected still matters, because the layout styles by class (`n.has("ex")`).
  - Nothing outside the file cites either old name (`git grep`, 2026-09-18). That makes a `retiredSymbolNames` row unnecessary. `TestARemovedDeclarationIsSweptOrRetired` asks for a row only while a current-truth file still names a removed citable declaration (read 2026-09-18, `repo_guard_test.go`), and nothing will.
- **Modified.**
  - Drop `Language:` from the `RenderOpts` literals, since the field is gone.
  - Drop the `bilingualLanguageText` half of `TestOxfordRejectsUnprovenRecords`. The parse-rejection half stays.
  - Drop `FuzzOxfordDocument`'s `doc.native` span loop.
  - Change `doc.source.text` to `doc.source` in the Oxford tests.

## Operating envelope and principles

- **ARCH-PURE:** everything touched is pure, and no IO moves.
- **ARCH-DRY:** two projection modes become one function (see `projectDisplayText` above).
- **ARCH-PURPOSE:** the sweep covers the whole chain, not the six members the issue listed.
- **ARCH-MOCK:** N/A. No dependency is added, and the native seam and its fake don't change.
- **ARCH-CONSTRAINTS:** interactive lookup path. The work only removes steps (one `projectDictionaryText` pass per Oxford record, and offset arithmetic per sense), so no budget moves.
- **ARCH-SECURE:** `parseBilingualDocument` keeps every check it runs on untrusted native HTML (size bound, identity, depth, UTF-8, Text correspondence). The only trust claim this work removes is ownership, which nothing consumed.
- **ARCH-ORDER:** holds no state between events. Every function touched is a single-shot parse or render over its arguments.
- **ARCH-FUNERAL:** this issue is the chain's end, and it creates nothing durable.

## Tasks

Single-pass atomic work: plain checkboxes, one `sdlc close`.

**Every commit must build and pass the full suite.** `CHECK` observes that after each commit; it is not predicted here. The order is what makes it possible:
1. tests that are valid today;
2. every writer and reader of the chain's data;
3. the declarations and producers.

The table's rows follow the same split. A `modified` row is checked by `TestPlanTableStatusMatchesTheChangeWindow` once this window touches its file, so each task touches every row in the files it opens. Rows marked `deleted` or `new` are not checked mid-plan (read 2026-09-18 in `repo_guard_test.go`). `currentTruthOnly` strips `deleted` names, `TestPlanTablesNameEntitiesThatExist` skips `new` rows while a `- [ ] ` remains, and `TestPlanTableStatusMatchesTheChangeWindow` skips `new`, `deleted`, and files the window doesn't touch.

**The check after every commit is the same** (`CHECK` below):
- `go build ./... && go vet ./... && go vet -tags conformance ./cmd/define/ && go test -count=1 ./...`, green;
- the three pty tests re-run outside the Bash sandbox (see the baseline in the issue Log).

Run `CHECK` after the commit, because two guards read the commit window.

### Task 1: Tests that are valid today

**Files:** create `cmd/define/language_text_test.go`; modify `dictionary_language_test.go`, `dict_test.go`, `dictionary_source_test.go`, `terminal_serialization_test.go`.

- [x] `TestDisplayProjectionOwnsOnlyMatchingGlyphs` covers `projectDisplayText`'s two directions. It needs one case per branch:
  - a glyph that differs from the source gives no ownership. The case that reaches the glyph branch is a **same-length substitution** (`"red red"`→`"red bed"`). A changed-length word (`"red red"`→`"red green"`) and joined words (`"an other"`→`"another"`) stay as cases for the length guards (`i >= len(source.text)` and the trailing-leftover check), which are what catch them. Measured 2026-09-18 on a mutant copy of today's body;
  - generated whitespace inside a source word keeps ownership (an inserted break, `"another"`→`"an other"`).
- [x] `FuzzDisplayProjection` is seeded with those cases. For any source and rendered text, it asserts the contract and nothing about the input's form (lessons: *A fuzz property may assert only YOUR contract*):
  - `result.text == rendered`;
  - spans lie within `rendered`, ascend, and don't overlap.
- [x] Rename `TestDictionaryDefinitionsRetainSourceAndRegions` to `TestDictionaryDefinitionsSectionTintKeepsRegions`, whose failure text becomes "tint changed region behavior". Rename `TestDictionaryDuplicateRecordOwnershipIsDeterministic` to `TestDictionaryDuplicateRecordSelectionIsDeterministic`.
- [x] Drop `Language:` from every `RenderOpts` literal in the tests. It is optional and unread, so this compiles today. The literals are in:
  - `TestDictionaryMonolingualOriginAndDisabledTint`
  - `TestDictionaryDefinitionsSectionTintKeepsRegions`
  - `TestDictionaryUnprovenSectionAlignmentStaysNeutral`
  - `TestLockedDictionaryPreservesSupplementalCapability`
  - `TestDefinitionUnknownSourceDoesNotInheritStudyLanguage`
  - `TestDictionaryAssemblyOwnsVerifiedPrimaryLanguage`
  - `TestTerminalSerializationPreservesOverlongDictionarySource`
- [x] Commit (`#76: pin the display projection; drop the unread Language from test literals`), then run `CHECK`. **This plan file lands in the same commit.** `TestPlanCitesTestsThatExist` has no in-progress exemption, so on its own the plan is red: measured 2026-09-18, it fails on exactly the three test names this task writes. The new tests describe a path that survives, so they must pass against today's code. `CHECK` is where that gets observed.

### Task 2: Remove every writer and reader of the chain's data

**Files:** `cmd/define/parse.go`, `cmd/define/definitions.go`; tests in `dictionary_language_test.go` and `bilingual_conformance_test.go`.

- [x] `git checkout 62c6a66^ -- cmd/define/parse.go`, which removes `Entry.source` and the offset plumbing.
- [x] `definitions.go`:
  - remove `definitionSection.source`;
  - in `renderDefinitionOutput`, remove `ro.Language = …`, `ro.Tint = tintPolicy{}` and the `entry.source` block;
  - in `spanishDefinitions.supplement`, remove both `section.source` appends.
- [x] Delete the tests that use what this task removes:
  - `TestDictionaryParserSourceOffsets`;
  - `TestDictionaryUnprovenSectionAlignmentStaysNeutral`, which builds a `definitionSection{source: …}`;
  - `TestBilingualNativeLanguageOwnership`, conformance-tagged, which writes `entry.source`.
- [x] `git diff 62c6a66^ -- cmd/define/parse.go` is empty.
- [ ] Commit (`#76: stop writing source provenance — parse.go is its pre-#65 self`), then run `CHECK`.

### Task 3: Delete the declarations and producers

**Files:** `render.go`, `bilingual_layout.go`, `language_text.go`; `dictionary_language.go` is deleted. Tests in `dictionary_language_test.go` and `bilingual_layout_test.go`.

- [ ] `render.go`: remove `RenderOpts.Language`.
- [ ] `bilingual_layout.go`:
  - remove `bilingualNode.lang` with its class→language switch and span append, and `bilingualDocument.native` with its `projectDictionaryText` call;
  - change `bilingualDocument.source` to `string`, and `renderBilingualDocument` reads `doc.source[n.start:n.end]`;
  - rewrite the `parseBilingualDocument` and `bilingualNode` comments to say what remains (structure, bounds, identity, Text correspondence).
- [ ] `language_text.go`: `projectDisplayText(source languageText, rendered string) languageText` takes `projectLanguageText`'s body without the `displayWhitespace` parameter and its refusal. The risk is dropping any other neutral fallback. Then delete `dictionary_language.go`.
- [ ] Tests:
  - delete `TestDictionarySourceProvenanceCorpus`, `TestDictionaryProjectionExactOccurrenceAndFallback` and `TestDictionaryProjectionDoesNotJoinWords`;
  - drop the `bilingualLanguageText` half of `TestOxfordRejectsUnprovenRecords` and `FuzzOxfordDocument`'s `doc.native` loop;
  - change `doc.source.text` to `doc.source` in `TestOxfordNativeTreeAndLeafConservation` and `FuzzOxfordDocument`.
- [ ] Commit (`#76: delete the language-ownership chain nothing reads`), then run `CHECK`.
- [ ] Mutation checks against that commit, with `-count=1`. Confirm each mutation applied (`git diff` shows it) and compiled, then restore with `git checkout -- cmd/define/language_text.go`; nothing else is uncommitted at that point.
  - **M-join:** replace the non-space mismatch condition `!strings.HasPrefix(source.text[i:], rendered[j:j+n])` with `false`. `TestDisplayProjectionOwnsOnlyMatchingGlyphs` must go red on the same-length substitution `"red red"`→`"red bed"`. Measured 2026-09-18 against a mutant copy of today's body, that is the only case M-join kills. The changed-length and joined-word cases stay neutral under it, because the length guards catch them.
  - **M-break:** reinstate `if begin == i { return languageText{text: rendered} }`. The inserted-break case must go red. Measured 2026-09-18 on a mutant copy, `"another"`→`"an other"` loses its ownership under it.

### Task 4: Sweep the prose and prove the absence

**Files:** `atlas/define.md`, `workshop/issues/000076-orphaned-source-provenance.md`.

- [ ] `atlas/define.md`:
  - delete the `RenderOpts.Language` row. `TestAtlasDescribesEveryRenderOpt` reflects over the struct and checks only that each field's qualified name appears (read 2026-09-18, `doc_sync_test.go`), so it needs no edit;
  - rewrite the `parseBilingualDocument` paragraph ("structure and ownership parsing … validated before ownership is trusted") to say it parses structure and validates identity and Text correspondence before the layout is trusted.
- [ ] Commit, then run `CHECK`.
- [ ] Absence proof. This command must print nothing:

  ```
  git grep -n -E 'projectDictionaryText|bilingualLanguageText|projectLanguageText|sourceOffset|advanceSource|sourceAt|sourceKnown|sourceBase|displayWhitespace|RenderOpts\.Language|ro\.Language|ro\.Tint|doc\.native|entry\.source|section\.source|leaf\.lang|node\.lang' -- cmd atlas README.md
  ```
- [ ] Live checks on this Mac: `go test -count=1 -tags conformance -run 'Bilingual|Oxford' ./cmd/define/`. Record which ran and which skipped, by name.
- [ ] Run the program before and after. Build `main` and HEAD binaries into `$TMPDIR`, then run each with `DEFINE_NO_CAPTURE=1` and `-no-audio`, piped, over `red`, `rendir`, `mesa`, `record` and `bank`, including the Spanish/bilingual path. Required: `diff` is empty.
- [ ] Log each run's scope and result in the issue's `## Log`, with the mutations named. Then tick.

## Done when

| criterion | pinned by |
|---|---|
| No reader and no writer of any deleted member remains | the Task 4 `git grep` (empty) |
| The surviving projection keeps its contract in both directions | `TestDisplayProjectionOwnsOnlyMatchingGlyphs`: its `"red red"`→`"red bed"` case must redden under M-join and its inserted-break case under M-break. The kill set was measured 2026-09-18 on a mutant copy of today's body, and Task 3 re-measures it against the test. `FuzzDisplayProjection` covers arbitrary input |
| Output unchanged, and every commit green | `CHECK` after each commit; the before/after binary diff is empty |
| The NOAD parser is exactly its pre-#65 self | `git diff 62c6a66^ -- cmd/define/parse.go` is empty |

## Revisions

### 2026-09-18: plan gate round 1 (PQ-1, PQ-2, PQ-3)

- **PQ-1** (every intermediate commit must build). Task 2 removed `Entry.source`,
  while its writer in `definitions.go` and the conformance-tagged
  `TestBilingualNativeLanguageOwnership` waited for Task 3. The tasks are
  reordered so the split is the class rule itself: tests valid today, then every
  writer and reader, then the declarations. `go vet -tags conformance` is now
  part of each task's `CHECK`, so the tagged files are compiled too.
- **PQ-2** (claims about guard behaviour must be measured). The predicted
  between-task red set was wrong in both directions: `deleted` rows are never
  checked, because `currentTruthOnly` strips their names, and Task 1's renames
  would have reddened an untouched `modified` row. The prediction is removed.
  Every commit is now green, and the check is the same for all of them.
  `projectDisplayText`'s row is `new` at `language_text.go`: that declaration is
  new in that file, and a `modified` row there was false until Task 3.
- **PQ-3** (Minor). The hand-written cases become one strategy line, and
  `FuzzDisplayProjection` adds a property check over arbitrary input.

### 2026-09-18: plan gate round 2 advisory (PQ-4, 2nd in `unbacked-existing-behavior-claim`)

- **Instance.** M-join survived both cases the plan named. Measured on a mutant
  copy of today's body: `"red red"`→`"red green"` and `"an other"`→`"another"`
  stay neutral under it. The length guards catch them (`i >= len(source.text)`
  and the trailing-leftover check), not the glyph branch. Only a same-length
  substitution (`"red red"`→`"red bed"`) kills M-join. Task 1's strategy line,
  Task 3's M-join expectation, and the Done-when row now name that case.
- **Family rule, applied to the whole plan, not only the one instance.** A
  predicted test or guard outcome carries a measurement beside it, or it goes.
  Swept:
  - the parse.go diff, the `.Tint` grep, the `retiredSymbolNames` claim, the
    mid-plan row exemptions and the `TestAtlasDescribesEveryRenderOpt` claim.
    Each is now marked measured or read on 2026-09-18;
  - "no display-mode test pins them" was measured by running M-join against
    today's suite. The claim was understated: no test pins the glyph branch in
    **either** mode. M-break is already killed by
    `TestPracticeLongRunOwnershipFollowsPhysicalRows`;
  - "every commit passes", "the new tests pass today" and "Expected: diff empty"
    were predictions. Each is now a requirement that `CHECK` or the run observes.
