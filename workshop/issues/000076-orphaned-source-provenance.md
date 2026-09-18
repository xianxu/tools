---
id: 000076
status: codecomplete
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 1.92
started: 2026-09-18T12:02:43-07:00
flow: {kind: full, provenance: inferred}
actual_hours: 1.32
---

# define: delete or re-use #66's orphaned source-provenance chain

## Problem

#70 deleted the per-fragment language tint, which production never reached
(`definitions.go` zeroed the tint before `Render`; production tints whole
SECTIONS as row paint). That left #66's source-provenance data with no
production reader:

- `RenderOpts.Language` — written at `definitions.go:80`, read nowhere.
- `Entry.source`, set from `definitionSection.source` (`definitions.go:86-87`).
- `definitionSection.source`, filled from `bilingualDocument.native`
  (`definitions.go:142-144`).
- `bilingualDocument.native` (`bilingual_layout.go:105`) and its only producer,
  `projectDictionaryText`.
- `Sense.sourceAt`/`sourceKnown` and `Example.sourceAt`/`sourceKnown`
  (`parse.go:187-202`, written at `:760`, `:813`).
- `bilingualLanguageText` — no production caller.

`bilingualDocument.source` is NOT part of this: the bilingual layout reads it.
Go does not report unused struct fields, so nothing forces the question.

Tests that check only this residue, to go with it if it goes:
`TestDictionaryParserSourceOffsets`, `TestDictionarySourceProvenanceCorpus`, the
`projectDictionaryText` half of `TestDictionaryProjectionExactOccurrenceAndFallback`,
and the provenance check in `TestBilingualNativeLanguageOwnership` (tagged).

## Spec

Decide: DELETE the chain and its tests, or give it a consumer (a feature that
needs per-fragment language ownership). Default: delete — it is data no output
reads. Found during #70 M1 (see #70's Log, 2026-09-17, "M1 Task 3").

**Decided 2026-09-18: delete.** The operator said: "generally, I'd like to delete
unused features." Tracing the chain from its readers finds more than the list
above, and the sweep takes all of it (ARCH-PURPOSE):

- the parser's `sourceBase ...int` variadics and `sourceOffset`/`advanceSource`.
  These are every change `parse.go` has had since `62c6a66^`, so the file goes
  back to exactly that;
- the Oxford parser's `bilingualNode.lang` class→language switch and the spans
  on `bilingualDocument.source`, which becomes a plain string. The layout reads
  only its text;
- the dead `ro.Tint = tintPolicy{}` write in `renderDefinitionOutput`;
- `projectLanguageText`'s strict mode. `projectDisplayText`, used by practice
  output, is the one live function in the chain's file. It keeps the display
  mode and moves to `language_text.go`, and `dictionary_language.go` goes.

Output does not change. Durable plan:
`workshop/plans/000076-orphaned-source-provenance-plan.md`.

## Done when

- Either every member listed above is gone with its tests (grep proves no
  reader and no writer remains), or a consumer reads it and a test pins that.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=0.75 impl=0.1
item: smaller-go-module       design=0.05 impl=0.15
item: smaller-go-module       design=0.0  impl=0.1
item: cross-cutting-refactor  design=0.1  impl=0.2
item: atlas-docs              design=0.05 impl=0.08
item: milestone-review        design=0.0  impl=0.2
design-buffer: 0.15
total: 1.92
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* (`sdlc estimate-source` flags the
calibration doc stale, so the numbers are provisional.)

Derivation, row by row. The v2 table gives the ranges, and each `impl=` is
written at 40% of its range, per v3.1:

- **issue-spec design=0.75**, lower-middle of 0.5–1.5. There was no brainstorm
  dialogue, because the operator gave the decision in one line. The hours went
  into three things: tracing the chain reader-first, which found five members
  the issue didn't list; two plan-gate rounds; and measuring PQ-4 by running
  the two mutants against the full suite.
- **Task 1: `smaller-go-module`.** A new test file with a unit test and a fuzz,
  two renames, and seven literal edits. Design is ×0.2 of 0–0.3, since the plan
  names every case.
- **Task 2: `smaller-go-module`, design 0.** `parse.go` is a `git checkout` with
  an empty-diff check. `definitions.go` loses a handful of lines.
- **Task 3: `cross-cutting-refactor`, impl at the top of its range.** Four
  production files (`render.go`, `bilingual_layout.go` and `language_text.go`
  edited, `dictionary_language.go` deleted), the projection fold, and test edits
  under two tag sets, then two mutation runs of about 135s each.
- **Task 4: `atlas-docs`.** Two atlas edits. The absence grep, the conformance
  run and the before/after binary diff are verification inside the same row.
- **`milestone-review`** is the one `sdlc close` boundary review. It's
  single-pass, with no Mx rows.
- **Step 2.5** doesn't apply, because nothing here is greenfield or on a novel
  stack. **Step 3** gives ×0.2 design on every implementation row, because the
  plan resolves them to the line. So the buffer is +15%, not +30%.
  **Familiarity 1.0**: the same files as #65 and #70.
- The arithmetic: design is 0.95, which ×1.15 gives 1.09. Impl is 0.83. The
  total is 1.92.

## Plan

Single pass, one `sdlc close`. Task detail lives in the durable plan.

- [x] Task 1: add tests that are valid today. That means a unit test and a fuzz
  pinning `projectDisplayText` in both directions, renaming the two tests whose
  names describe the deleted concept, and dropping the unread `Language:` from
  test literals.
- [x] Task 2: remove every writer and reader. Restore `parse.go` to `62c6a66^`
  (an empty diff proves it), strip `definitions.go`, and delete the tests of
  what was removed.
- [x] Task 3: delete the declarations and producers. Then run mutation checks
  M-join and M-break.
- [x] Task 4: sweep the atlas, grep for absence, run the live conformance checks
  and diff the binary's output before and after. Log each result.

Every commit builds, including under `-tags conformance`, and passes the full
suite.

The tests the Problem section names, deleted here:

| test | file | status |
|---|---|---|
| `TestDictionaryParserSourceOffsets` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionarySourceProvenanceCorpus` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryProjectionExactOccurrenceAndFallback` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestBilingualNativeLanguageOwnership` | `cmd/define/bilingual_conformance_test.go` | deleted |

## Log

### 2026-09-17

### 2026-09-18
- 2026-09-18: closed — orphaned source-provenance chain deleted: absence git grep over cmd/atlas/README (17 member names) prints nothing at HEAD; parse.go byte-identical to 62c6a66^; go build + vet (both tag sets) + full test suite green at HEAD 5b7ed8f (3 pty tests pass unsandboxed); projectDisplayText pinned by TestDisplayProjectionOwnsOnlyMatchingGlyphs + FuzzDisplayProjection, M-join/M-break mutations killed; live -tags conformance Bilingual|Oxford green unsandboxed; before/after binaries byte-identical over 30 piped + 16 pty runs; review verdict: SHIP

- Claimed. Operator decision: delete ("generally, I'd like to delete unused
  features"). Traced the chain from its readers rather than from the listed
  members. That added the parser variadics/helpers, `bilingualNode.lang`,
  `doc.source`'s spans, `projectLanguageText`'s strict flag, and the dead
  `ro.Tint` zeroing. `diff 62c6a66^:cmd/define/parse.go` shows offsets are
  parse.go's only change since #65.
- Baseline on `main` (`go build`, `go vet`, `go test -count=1 ./...`): all green
  except `TestLanguagePromptStartup`, `TestLanguageTintInvocation` and
  `TestSavedSchemeGovernsALookup`. Those failed with "operation not permitted"
  under the Bash sandbox (pty), and passed when re-run outside it. So `main`
  is green.
- Durable plan written (full flow: five code files is outside the quick shell).
  I first wrote here that the plan's `deleted` rows are red-first pins under
  `TestPlanTablesNameEntitiesThatExist`. That was wrong: the plan gate (PQ-2)
  measured it. `currentTruthOnly` strips every name a `| … | deleted |` row
  carries, so those rows are never checked.
- Plan gate round 1: PQ-1 (Task 2 would not build) and PQ-2 (the predicted red
  set was unmeasured). The fix is the class rule rather than the two sites:
  tasks are ordered tests → writers/readers → declarations, so every commit is
  green. PQ-3 added `FuzzDisplayProjection`. Recorded in the plan's
  `## Revisions`.
- Plan gate round 2 passed with one advisory, PQ-4 (Minor, 2nd in
  `unbacked-existing-behavior-claim`): M-join survives both cases the plan
  named. Measured on mutant copies of today's body: only `"red red"`→`"red bed"`
  kills M-join, and `"another"`→`"an other"` kills M-break. Ran both mutants
  against today's committed code with the full suite (`-count=1`, each applied,
  compiled, then restored). M-join reddens nothing: no test pins the glyph branch
  in either mode. M-break reddens `TestPracticeLongRunOwnershipFollowsPhysicalRows`.
  Applied the family rule plan-wide, not only to the PQ-4 instance. Every
  predicted guard or test outcome is now marked measured/read 2026-09-18, or
  restated as a requirement that `CHECK` observes. Recorded in the plan's
  `## Revisions`.
- Estimate derived: v3.1 Method A, 1.92h (see `## Estimate`).
- `change-code` gates: plan-quality round 3 was CLEAN, with PQ-4 addressed.
  Estimate-quality returned INFO, which does not block. Its points are kept here
  rather than re-costing the estimate after the gate:
  - Task 4's verification (two binaries × 5 words, the conformance run) is not
    priced in `atlas-docs`;
  - the four per-commit `CHECK` runs (about 135s each, plus the pty re-runs) and
    the risk of a guard-driven fix-and-recommit are not priced either;
  - Task 3's basis said five production files. It is four, and the Estimate text
    now says so.
  If the actual overruns, these are the first place to look.
- Implementation, one commit per task, each followed by `CHECK`: `go build`,
  `go vet` with and without `-tags conformance`, and `go test -count=1 ./...`.
  Every run was green apart from the three sandbox pty tests, and those passed
  when re-run outside the sandbox each time.
  - Task 1 (`4304070`): `TestDisplayProjectionOwnsOnlyMatchingGlyphs` and
    `FuzzDisplayProjection`, plus the two renames and the literal edits. The
    fuzz ran 30s and 20s (about 216k execs), with no failure.
  - Side-quest (`5de02aa`, ordered before Task 2): Task 2's first `CHECK`
    reddened `TestPlanTableStatusMatchesTheChangeWindow` on the correct
    `Entry … modified` row. The guard read only new-side hunk lines, so a pure
    deletion was invisible. The same gap skipped `definitions.go`'s rows
    entirely, because that file's whole diff was deletions. Fixed to record
    deletion points, pinned by `TestWindowChangeSeesADeletionInsideADeclaration`.
    Mutations, each applied, compiled and run with `-count=1`:
    - M-cut (drop the recorded cuts): two unit cases red, plus the `Entry`
      guard red;
    - M-body (count a cut anywhere in the region): the neighbour case red;
    - a planted `renderDefinitionOutput … unchanged` row: red. The guard could
      not see that row before the fix.
    The suite at `5de02aa` ran in a scratch worktree and was green. Plan
    `## Revisions` and lessons record it.
  - Task 2 (`2604140`): `git diff 62c6a66^ -- cmd/define/parse.go` is empty.
    `section.language` stays, because line 58's section tint role reads it.
  - Task 3 (`1396205`): mutations against the commit, full `define` suite,
    `-count=1`, each applied and compiled. Apart from the sandbox pty trio:
    - M-join reddens only
      `TestDisplayProjectionOwnsOnlyMatchingGlyphs/same-length_substitution`;
    - M-break reddens its `inserted_space…` and `inserted_newline…` cases and
      `TestPracticeLongRunOwnershipFollowsPhysicalRows`.
    Both match the plan's measured kill sets.
- Task 4:
  - **Absence grep:** the plan's `git grep` over `cmd atlas README.md` prints
    nothing (exit 1).
  - **Live conformance**, `-tags conformance -run 'Bilingual|Oxford'`.
    Sandboxed, four native tests skipped with "Oxford Spanish dictionary
    inaccessible": `TestBilingualNativeDirection`, `TestBilingualNativeLimits`,
    `TestBilingualNativeRendirLayout` and `TestBilingualNativeSystemDictionary`.
    Outside the sandbox all four ran and passed, including the six Direction
    subtests. No skips and no failures.
  - **Before/after binaries** (`fe28b01` against `1396205`), with
    `DEFINE_NO_CAPTURE=1 DEFINE_NO_BACKGROUND=1 -no-audio` and stdin from
    `/dev/null`:
    - piped, over `red rendir mesa record bank` × `-lang en|es`: 30 files
      (stdout, stderr, exit code) are byte-identical. The Oxford section is
      present in the Spanish `red`, `rendir`, `mesa` and `record` runs;
    - under a pty (`script`) with `TERM=xterm-256color`, `-scheme dark|light`
      and `-lang es`, over `red rendir mesa record`: 16 files are byte-identical,
      the section tint is painted (62 to 322 background escapes per run), and
      every run exits 0. This is the run that covers the dropped
      `ro.Tint = tintPolicy{}`.
    - An earlier pty run without an explicit `TERM` painted no tint. Its
      environment wasn't recorded, so the cause isn't established. A re-run with
      the tool shell's `TERM=pair-vt-256color` does paint it, so the atlas's
      "disabled by `TERM=dumb`" stands.
- Close: the boundary review (window `fe28b01..5b7ed8fc`) returned **SHIP**,
  with no Critical or Important findings and five Minors. The reviewer
  reproduced the empty `parse.go` diff, the absence grep, build + vet for all
  six commits, and both mutation kill sets. Fixed in the close commit, per the
  post-verdict protocol:
  - BR-1 and BR-2 (`name-outlives-referent`) and BR-3
    (`atlas-describes-removed-surface`) are one rule: a deletion sweep also
    takes the concept's names and prose. `dictionary_language_test.go` is
    deleted. Its two live tests and helper move to `definitions_output_test.go`
    and `bilingual_test.go` unchanged.
    `TestDictionaryMonolingualOriginAndDisabledTint` is deleted, because
    `TestRenderColorOnlyWhenAsked` covers it. Measured: with `newPalette`
    ignoring `on`, both tests go red. The atlas sentence now describes the
    region offsets and primary-only deck-word actions. A vocabulary sweep
    (`provenance`, `ownership`, `origin` over file names, test names and prose)
    found no other residue; the remaining hits describe live concepts.
  - BR-4: `languageSpan`'s comment states that spans ascend and never overlap.
  - BR-5: the unrelated working-tree changes stay out of the commit.
  - The plan's `## Revisions` and `workshop/lessons.md` (*Sweep a deleted
    concept's vocabulary, not only its identifiers*) record it.
  - Measured actual 1.32h against the 1.92h estimate (1.5× under).
- `gofmt -l` flags `cmd/define/play/presentation.go` and
  `selection_paths_test.go`. Both were last touched in #70 (`60321f0`), not in
  this window, so they are left alone.
