# Boundary Review — tools#61 (whole-issue close)

| field | value |
|-------|-------|
| issue | 61 — define: show Spanish definitions and English translations together |
| repo | tools |
| issue file | workshop/issues/000061-define-show-spanish-definitions-and-english-translations-together.md |
| boundary | whole-issue close |
| milestone | — |
| window | 602aab61cc5b3de1532d5cb4418e892d59dbb751..24c67621ea1179350a600e27b369be397fccbcef |
| command | sdlc close --issue 61 |
| reviewer | codex |
| timestamp | 2026-09-14T18:00:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation largely delivers the revised scope, and the portable suite passes. One blocking contract mismatch remains: unavailable metadata APIs produce a misleading missing-dictionary diagnosis. Native conformance could not be confirmed because Oxford Spanish–English was unavailable in this review environment.

## 1. Strengths

- Direction selection validates record identity and title, including ambiguous spellings and inflections.
- Shared rendering preserves Spanish-first order and English text selection without Spanish vocabulary actions.
- Tests cover toggle propagation into nested practice, capture counts, and inflected-word audio.
- Settings reuse atomic storage; README and atlas document the new surface.

## 2. Critical findings

**Unknown availability becomes confirmed absence — [dict_darwin.go:295](/Users/xianxu/workspace/tools/cmd/define/dict_darwin.go:295).**

`installedDictionaries()` returns nil when its metadata surface is unavailable. `spanishDictionarySources(nil)` correctly reports “availability unknown,” but the factory constructs the same enable/download-Larousse error used for confirmed absence. This violates the Spec’s distinct failure categories and **ARCH-SECURE**.

Preserve an API/metadata-unavailable result separately. Add factory/composition regression coverage distinguishing nil metadata from a successfully enumerated empty installation; the test must fail with the current constructor.

## 3. Important findings

None.

## 4. Minor findings

The [Core concepts table](/Users/xianxu/workspace/tools/workshop/plans/000061-bilingual-definitions-plan.md:13) omits PURE/INTEGRATION classifications and groups pure parsing with effectful command execution. Referenced entities exist; add the classifications and split mixed rows.

## 5. Test coverage notes

- Passed: focused bilingual, Spanish-parser, and repository-guard tests.
- Passed: `go test ./cmd/define/... ./internal/conformance` (define: 111.680s).
- Failed environmental prerequisite: strict `TestBilingualNative*` checks reported unavailable Oxford dictionaries.
- Existing availability tests check metadata names, but do not exercise the factory’s nil-metadata failure path.
- Code matched pinned HEAD; tracker inspection used committed versions because local tracker edits were present.

## 6. Architecture

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — shared composition and existing storage/region helpers. |
| ARCH-PURE | Pass — selection and rendering separated from native IO. |
| ARCH-PURPOSE | Pass — lookup and both full practice reveals implemented. |
| ARCH-MOCK | Pass structurally — shared fake seam and conformance tests; live verification unavailable here. |
| ARCH-CONSTRAINTS | Pass structurally — bounded records, bytes, and depth; latency check could not run successfully. |
| ARCH-SECURE | **Flag** — unknown metadata becomes asserted absence. |
| ARCH-ORDER | Pass — synchronous composition; persistence precedes session mutation. |
| ARCH-FUNERAL | Pass — native resources released; one overwritten setting ends with deck deletion. |

## 7. Plan revisions

Append a `## Revisions` entry naming the factory failure distinction and its regression test. Add PURE/INTEGRATION classifications to the concept inventory.

```findings
findings:
  - id: new
    severity: Critical
    family: unknown-availability-is-not-absence
    title: |
      Spanish factory reports missing Larousse when metadata availability is unknown
    detail: |
      cmd/define/dict_darwin.go:295 constructs an enable/download error even when installedDictionaries returns nil for unavailable metadata APIs. Preserve that failure separately from confirmed absence and test both through factory composition; this violates the Spec and ARCH-SECURE.
  - id: new
    severity: Minor
    family: explicit-core-concept-classification
    title: |
      Core concepts omit PURE and INTEGRATION classifications
    detail: |
      workshop/plans/000061-bilingual-definitions-plan.md:13 lacks the requested kind column and groups pure parsers with effectful command handlers. Add classifications and split mixed rows.
```
