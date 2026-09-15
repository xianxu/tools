# Boundary Review — tools#58 (whole-issue close)

| field | value |
|-------|-------|
| issue | 58 — define: discover local proxy models and select by preference |
| repo | tools |
| issue file | workshop/issues/000058-local-model-discovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6017df4345a342454b8a00e7ed22bef2a3a5e1a..f5e1cda3980a5033160ded04563f6a459190fa5f |
| command | sdlc close --issue 58 |
| reviewer | codex |
| timestamp | 2026-09-13T22:49:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation largely matches the Spec and Plan, including provider-first selection, lazy bounded discovery, cancellation, and selected-model provenance. Focused suites pass, but the new conformance test bypasses the repository’s strict-mode policy and breaks an existing repository-wide guard.

1. **Strengths**
   - Pure selection policy has numeric-ordering, permutation, duplicate, and unsupported-model coverage.
   - Discovery bounds and sanitizes external input; redirects cannot forward authentication.
   - Channel-controlled tests verify shared discovery, independent waiter cancellation, and retry after owner cancellation.
   - Consumer tests inspect wire model IDs, persisted reflection provenance, and zero-network cache/dictionary paths. README and atlas cover the new surface.

2. **Critical findings:** None.

3. **Important findings**
   - [discovery_conformance_test.go:18](/Users/xianxu/workspace/tools/internal/llm/discovery_conformance_test.go:18), also lines 21 and 26 — **Conformance skips bypass shared policy (ARCH-DRY, ARCH-MOCK).** `TestEverySkipIsRoutedOrWaived` fails on all three sites. With `CONFORMANCE_STRICT=1` and missing credentials, the new test still reports `SKIP` and exits successfully. Route absent dependencies through `conformance.SkipOrFail`; explicitly classify and document any truly inapplicable configuration branch using the existing waiver convention.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed: LLM unit suite, LLM race suite, full define suite, release-stamp merge check, and pinned-range whitespace check.
   - Failed: `go test ./internal/conformance -run TestEverySkipIsRoutedOrWaived -count=1`.
   - Reproduced strict-mode false success without network access using a custom endpoint with both API-key variables empty.
   - Live provider inference was not independently rerun.

6. **Architectural notes**
   - **ARCH-DRY — flag:** reuse the existing conformance policy helper.
   - **ARCH-PURE — pass:** selector, configuration resolution, and effective rendering have direct pure tests; IO remains at transport boundaries.
   - **ARCH-PURPOSE — pass:** ask, harvest, reflection, and diagnostics consume discovery.
   - **ARCH-MOCK — flag:** stateful fake coverage is sound, but strict live verification can silently skip.
   - **ARCH-CONSTRAINTS — pass:** size, entry, and deadline bounds are implemented and tested.
   - **ARCH-SECURE — pass:** catalog validation, redirect refusal, and sanitized discovery errors protect the boundary.
   - **ARCH-ORDER — pass:** explicit flight states and controlled interleavings cover publication and cancellation.
   - **ARCH-FUNERAL — pass:** selection lives only with its client; no durable discovery cache or detached worker is introduced.

7. **Plan revision recommendation**
   - Append a `## Revisions` entry requiring conformance skip classification through the existing shared policy and include the repository-wide skip guard in closure verification.

```findings
findings:
  - id: new
    severity: Important
    family: conformance-skip-policy
    title: |
      New discovery conformance test bypasses strict-mode policy and fails the repository guard
    detail: |
      internal/llm/discovery_conformance_test.go:18,21,26 directly skip instead of using the existing conformance policy. TestEverySkipIsRoutedOrWaived fails on all three sites, and a missing-credential run still exits successfully under CONFORMANCE_STRICT=1. Route absent dependencies through conformance.SkipOrFail and explicitly justify any inapplicable configuration waiver; rerun the guard and strict/default behavior checks (ARCH-DRY, ARCH-MOCK).
```

---

## Re-review — 2026-09-13T22:54:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 58 — define: discover local proxy models and select by preference |
| repo | tools |
| issue file | workshop/issues/000058-local-model-discovery.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6017df4345a342454b8a00e7ed22bef2a3a5e1a..0cd6e0482f11b056795b3fae4b9ea797d3cd864f |
| command | sdlc close --issue 58 |
| reviewer | codex |
| timestamp | 2026-09-13T22:54:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the Spec and Plan: provider-first selection, lazy bounded discovery, shared request rendering, and selected-model provenance are delivered. BR-1 is addressed with verified regression evidence. No blocking findings remain.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      All three branches use conformance.SkipOrFail. In a disposable scratch copy, TestEverySkipIsRoutedOrWaived fails on all three pre-fix sites and passes with the pinned fix. Missing credentials and non-auto configuration skip by default and fail under CONFORMANCE_STRICT=1.
findings:
  - id: new
    severity: Minor
    family: documentation-inventory-current
    title: |
      Atlas conformance inventory omits the new discovery suite
    detail: |
      atlas/llm.md:248 still says “Three tagged suites” and omits TestConformanceAutoSelection. Update the count and list the new test, including its provider-specific verification scope.
```

1. **Strengths**
   - `internal/llm/models.go`: deterministic, owner-based selection with numeric ranking, anchored grammars, and permutation tests.
   - `internal/llm/auto.go`: explicit flight ownership and cancellation behavior, tested with controlled interleavings.
   - `internal/llm/discovery.go`: bounded parsing, redirect refusal, and sanitized discovery errors.
   - Provider wire tests and define integration tests verify schema instructions, offline paths, and persisted provenance.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** Atlas conformance inventory, as recorded above.

5. **Test coverage**
   - Passed `go test ./internal/llm/... ./internal/conformance -count=1`.
   - Passed `go test -race ./internal/llm/... -count=1`.
   - Passed `go test ./cmd/define/... -count=1`.
   - Passed pinned-range `git diff --check`.
   - Verified BR-1’s pre-fix failure and post-fix success.
   - Live provider inference was not rerun; this review confirms fake-backed behavior and strict/default policy.

6. **Architecture**
   - **ARCH-DRY — pass:** shared selector, renderer, and conformance policy.
   - **ARCH-PURE — pass:** policy/config/rendering have direct pure tests; IO remains at integration boundaries.
   - **ARCH-PURPOSE — pass:** ask, harvest, reflect, and diagnostics inherit discovery.
   - **ARCH-MOCK — pass:** stateful catalog fake shares the production transport boundary; opt-in live check exists.
   - **ARCH-CONSTRAINTS — pass:** size limits, discovery cap, and shared total deadline are enforced and tested.
   - **ARCH-SECURE — pass:** catalog validation and sanitized errors protect the new boundary.
   - **ARCH-ORDER — pass:** documented transitions match code and deterministic cancellation tests.
   - **ARCH-FUNERAL — pass:** selection dies with the client; no durable cache or detached discovery worker.

7. **Plan revision recommendations:** None; core-concept classifications and implemented behavior agree.
