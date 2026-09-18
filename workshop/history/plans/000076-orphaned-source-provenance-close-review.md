# Boundary Review — tools#76 (whole-issue close)

| field | value |
|-------|-------|
| issue | 76 — define: delete or re-use #66's orphaned source-provenance chain |
| repo | tools |
| issue file | workshop/issues/000076-orphaned-source-provenance.md |
| boundary | whole-issue close |
| milestone | — |
| window | fe28b01d0a9cde14d716aa4c611f580789001035..5b7ed8fc1cab83c501034006d9388048b1aeb37b |
| command | sdlc close --issue 76 |
| reviewer | claude |
| timestamp | 2026-09-18T13:59:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

A clean, fully-swept deletion. I independently reproduced every load-bearing claim: `cmd/define/parse.go` at HEAD is byte-identical to `62c6a66^` (empty diff) **and** `git log 62c6a66^..fe28b01 -- cmd/define/parse.go` shows #65 was the only commit ever to touch it since, so the file-level revert provably loses nothing else; the absence grep over `cmd atlas README.md` is empty (exit 1); all six commits build and vet clean including `-tags conformance`; and both mutations reproduce the plan's measured kill sets exactly (M-join → only `TestDisplayProjectionOwnsOnlyMatchingGlyphs/same-length_substitution`; M-break → the two inserted-whitespace subtests plus `TestPracticeLongRunOwnershipFollowsPhysicalRows`). The full suite at HEAD is green apart from three pty tests (`TestLanguagePromptStartup`, `TestLanguageTintInvocation`, `TestSavedSchemeGovernsALookup`), which fail identically at the base commit `fe28b01` in a scratch worktree with `pty.Open: operation not permitted` — environmental, pre-existing, not this window. The plan's Core-concepts and Tests tables match the code row for row; no plan revision is needed. Nothing here blocks the boundary; the four findings are all Minor.

## 1. Strengths

- **The revert is provably safe, and the plan said so up front.** `workshop/plans/000076-…-plan.md` named "the check is an empty diff" as the criterion before doing it, rather than asserting equivalence after. Verified independently, plus the single-commit history that makes the revert lossless.
- **The new test is real regression evidence, not a restatement.** `cmd/define/language_text_test.go:16` documents *which* case reaches the glyph comparison and why the length guards catch the other two — and the mutation confirms it: with `!strings.HasPrefix(...)` replaced by `false`, only `same-length_substitution` reddens. The comment's "Measured 2026-09-18" claim held up.
- **The side-quest fix is reachable and pinned in both directions.** Dropping `change.cuts = append(...)` in `parseHunks` (`cmd/define/repo_guard_test.go:1573`) reddens both field cases of `TestWindowChangeSeesADeletionInsideADeclaration` **and** the live `TestPlanTableStatusMatchesTheChangeWindow`. The neighbour cases (`@@ -8,2 +7,0 @@`, `@@ -7,2 +6,0 @@`) pin the other half — the blank-line ambiguity that would otherwise have made the fix overshoot.
- **`bilingualDocument.source` collapsed from `languageText` to `string`** (`cmd/define/bilingual_layout.go:19`) — "a `languageText` whose spans nobody sets is a string wearing a type". That is the sweep going one level past the issue's list, and it dropped the `store` import from the file entirely.
- **`ro.Tint = tintPolicy{}` removal is grep-verified safe:** `\.Tint\b` over production files hits only `definitions.go:109`, which reads `opt.Tint` (section paint), never the per-entry `ro`. `assertDictionaryTint` decodes background escapes byte-by-byte, so `TestDictionaryDefinitionsSectionTintKeepsRegions` would still catch a fragment-level tint if one ever reappeared.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- `cmd/define/dictionary_language_test.go:1` — the test file outlives the production file it is named for. `dictionary_language.go` was deleted precisely because it "would hold nothing dictionary-related"; its test file keeps the name while holding only definition-tint tests. Fold into `definitions_output_test.go` or rename.
- `cmd/define/dictionary_language_test.go:45` — `TestDictionaryMonolingualOriginAndDisabledTint` now asserts only "`Color:false` emits no ANSI". Nothing about monolingual origin survives; #70 gutted half and this window removed the last `Language:` reason for the rest.
- `atlas/define.md:2350` — "`renderDefinitionOutput` retains section language ownership through rendering and region offsets" was the prose for `ro.Language = section.language`, now deleted. What remains is `ro.Vocab = nil` plus role-based row paint; a one-clause edit would stop the sentence pointing at a removed field.
- `cmd/define/language_text.go:17` — `projectDisplayText`'s `owner()` advances `spanIndex` monotonically and never rewinds, so it silently requires `source.spans` to ascend and not overlap. That precondition is written nowhere on `languageText`, and `FuzzDisplayProjection` only ever builds a sorted two-span partition, so it is never explored. Both producers (`language_decode.go:123`, `practice_output.go:40`) do append in order, so nothing is broken — but this diff made `projectDisplayText` the type's only consumer and moved it next to the type, which is the moment to state the invariant.
- Process: the working tree carries unrelated uncommitted changes (`.github/workflows/merge-check.yml`, `Makefile` type change, `bootstrap.sh`, deleted `scripts/issue-sync.sh`, untracked `scripts/merge-checks.d/40-duplicate-issue-id.sh`). None is #76's; keep them out of the close commit.

## 5. Test coverage notes

- **Mutation evidence reproduced, not taken on faith.** M-join and M-break both behaved exactly as the plan's `## Revisions` claims, and M-cut against the guard reddens the real plan-table check. This is the strongest part of the boundary.
- **Deleted tests are accounted for.** `TestDictionaryProjectionExactOccurrenceAndFallback` / `DoesNotJoinWords` lost only their strict-mode halves; their surviving cases are present as `changed-length word` and `joined words`. `TestBilingualNativeLanguageOwnership` tested a property that no longer exists, and the installed-source path still has `TestBilingualNativeRendirLayout`, `…Direction`, `…Limits`, `…SystemDictionary`.
- **One claim I could not independently reproduce:** this review environment has no Dictionary Services access — all four native conformance tests SKIP here, and both before/after binaries exit 1 with "Spanish dictionary unavailable" (identically, which is a weak signal at best). The issue `## Log`'s byte-identical 30-file and 16-file pty comparisons rest on the implementor's run. The fixture-driven Oxford render tests that *do* run here (`TestOxfordNativeTreeAndLeafConservation`, `TestOxfordCorpusConservation`, `TestDictionaryDefinitionsSectionTintKeepsRegions`, `TestTerminalSerializationPreservesOverlongDictionarySource`) all pass, which covers the same render path against captured records.
- Per-commit *suite* greenness was not re-run (the plan guards read the git window, so a detached-HEAD run reports a different window); build + vet + conformance-vet at each of the six commits was verified instead.

## 6. Architectural notes

- **ARCH-DRY — pass.** Two projection modes became one function beside the type it projects; `dictionary_language.go` is gone rather than left holding one indirection.
- **ARCH-PURE — pass.** Everything touched is a pure string function; `language_text_test.go` runs with no exec, net, or fs.
- **ARCH-PURPOSE — pass.** Shadow-sweep run: the surviving consumers of `languageText`/`languageSpan` are `language_decode.go` (producer) and `practice_output.go:51-52` (consumer), both live on the practice path. No hand-maintained restatement of the deleted model remains in `cmd/`, `atlas/`, or `README.md`. The sweep took five members the issue never listed, which is the class rather than the instance.
- **ARCH-MOCK — pass.** `fakeRecordSource` and the `recordSource` seam are untouched; the one deleted live check tested a property that ceased to exist.
- **ARCH-CONSTRAINTS — pass.** Net removal of per-record and per-sense work on the interactive lookup path; no new fan-out or repeated work.
- **ARCH-SECURE — pass.** `parseBilingualDocument` keeps every check on untrusted native HTML (size bound, depth, identity, UTF-8, Text correspondence), and the rewritten comment at `bilingual_layout.go:22` names them accurately. Leaf offsets into `doc.source` are in range by construction, and `FuzzOxfordDocument` still asserts `n.end <= len(doc.source)` and full consumption.
- **ARCH-ORDER — pass.** Nothing touched carries state between events; the only cursor is `spanIndex`, local to one call (see the Minor on its unwritten precondition).
- **ARCH-FUNERAL — pass.** Pure deletion; the only new durable artifacts are the plan and gate ledger, which archive with the issue. Unrelated: `git worktree list` shows four stale/prunable scratch worktrees under `/private/tmp/claude-501` from earlier sessions — worth a `git worktree prune` at some point, but not this window's residue.

## 7. Plan revision recommendations

None. The plan's Core-concepts and Tests tables match the code at every row I checked, the three `## Revisions` entries accurately record what changed and why, and the side-quest is recorded in both the plan and `workshop/lessons.md`.

```findings
findings:
  - id: new
    severity: Minor
    family: name-outlives-referent
    title: |
      dictionary_language_test.go is named for a production file this window deleted
    detail: |
      dictionary_language.go was deleted because it "would hold nothing
      dictionary-related", but its test file keeps the name while holding only
      definition-tint tests (assertDictionaryTint and three TestDictionary*
      cases). Fold into definitions_output_test.go or rename.
  - id: new
    severity: Minor
    family: name-outlives-referent
    title: |
      TestDictionaryMonolingualOriginAndDisabledTint asserts neither origin nor tint
    detail: |
      cmd/define/dictionary_language_test.go:45 now only checks that Color:false
      emits no ANSI. #70 removed the tint half; this window removed the last
      Language: literal that gave "MonolingualOrigin" any meaning. Rename to what
      it checks, or delete it if another no-color assertion already covers it.
  - id: new
    severity: Minor
    family: atlas-describes-removed-surface
    title: |
      atlas/define.md:2350 still describes the deleted ro.Language write
    detail: |
      "renderDefinitionOutput retains section language ownership through
      rendering and region offsets" was the prose for `ro.Language =
      section.language`. What remains is `ro.Vocab = nil` for non-primary
      sections plus role-based row paint; a one-clause edit keeps the sentence
      true without naming a field that no longer exists.
  - id: new
    severity: Minor
    family: unwritten-precondition
    title: |
      projectDisplayText requires ascending non-overlapping spans, stated nowhere and never fuzzed
    detail: |
      owner() in cmd/define/language_text.go:32 advances spanIndex monotonically
      and never rewinds, so out-of-order or overlapping source.spans silently
      lose ownership rather than failing visibly. languageText's doc comment
      (language_text.go:17) does not state the invariant, and
      FuzzDisplayProjection only builds a sorted two-span partition, so it never
      explores a violation. Both current producers do append in order; this diff
      made projectDisplayText the type's only consumer and moved it beside the
      type, which is the moment to write the precondition down.
  - id: new
    severity: Minor
    family: unrelated-changes-in-close-window
    title: |
      Working tree carries uncommitted changes unrelated to this issue
    detail: |
      .github/workflows/merge-check.yml, a Makefile type change, bootstrap.sh, a
      deleted scripts/issue-sync.sh and an untracked
      scripts/merge-checks.d/40-duplicate-issue-id.sh are modified but belong to
      no #76 commit. Keep them out of the close commit.
```
