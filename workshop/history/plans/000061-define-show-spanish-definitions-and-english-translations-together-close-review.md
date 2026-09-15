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

---

## Re-review — 2026-09-15T10:00:10-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 61 — define: show Spanish definitions and English translations together |
| repo | tools |
| issue file | workshop/issues/000061-define-show-spanish-definitions-and-english-translations-together.md |
| boundary | whole-issue close |
| milestone | — |
| window | b47aba3f734c71408e56890111839e516fe3c6c2..7752bddce9f1f99eb6d22e393ce1c682c580f294 |
| command | sdlc close --issue 61 |
| reviewer | codex |
| timestamp | 2026-09-15T10:00:10-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the issue and its approved scope revisions. Both prior findings are addressed. Review found no blocking correctness, integration, or documentation gaps.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      bilingual_sources.go preserves unknown availability separately from confirmed absence. TestBilingualFactoryMetadataDiagnostic passes on HEAD and fails in both unknown-metadata cases when the fix is removed through a temporary overlay.
  - id: BR-2
    disposition: addressed
    note: |
      The plan's Core concepts table now classifies PURE and INTEGRATION entities and separates parsers/renderers from factory, command, and persistence effects. The cited implementation paths support those classifications.
```

### 1. Strengths

- Spanish-source selection rejects wrong-direction records; native conformance verifies ambiguous `red`, accents, and inflections.
- Shared definition rendering preserves successful sections, primary-only raw output, and language-specific click regions.
- Practice assistance translates displayed material, validates cached answers, and supplies Choice help for every option or none.
- README and atlas document setup, toggle persistence, assistance preparation, and offline behavior.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage

Passed:

- `go test ./cmd/define/... ./internal/conformance -count=1`
- Focused bilingual and assistance regressions.
- Native direction, limits, and assembled-factory conformance.
- `go vet ./cmd/define/...`

BR-1’s regression demonstrably fails without its fix. Twenty warm native lookups completed in approximately 22 ms.

Range whitespace inspection reported trailing spaces in the new prompt golden’s empty metadata fields; no behavioral defect identified. Live model conformance was inspected but not rerun.

### 6. Architecture

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — shared rendering and translation preparation. |
| ARCH-PURE | Pass — selection and validation separated from IO. |
| ARCH-PURPOSE | Pass — lookup, reveals, and revised pre-answer assistance delivered. |
| ARCH-MOCK | Pass — injected dictionary/model fakes and conformance checks. |
| ARCH-CONSTRAINTS | Pass — bounded native results, batches, deadline, and disk cache. |
| ARCH-SECURE | Pass — metadata uncertainty preserved; external and cached translations validated. |
| ARCH-ORDER | Pass — preparation precedes playback; interrupt scope covers preparation. |
| ARCH-FUNERAL | Pass — settings replaced atomically; durable cache evicted within limits; session memory ends with process. |

### 7. Plan revisions

None required.
