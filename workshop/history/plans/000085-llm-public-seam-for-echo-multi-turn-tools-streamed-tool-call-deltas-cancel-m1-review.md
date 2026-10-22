# Boundary Review — tools#85 (milestone M1)

| field | value |
|-------|-------|
| issue | 85 — llm: public seam for echo — multi-turn, tools, streamed tool-call deltas, cancel |
| repo | tools |
| issue file | workshop/issues/000085-llm-public-seam-for-echo-multi-turn-tools-streamed-tool-call-deltas-cancel.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 2ff2d9d484a5b709721375699cf377ffb199dd43..12082073a2925d1c8cbf8086d62118446152a51b |
| command | sdlc milestone-close --issue 85 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-21T20:54:19-07:00 |
| verdict | SHIP |

## Review

Mostly a package move. Let me read the plan's M1 scope.

I'm checking whether two `cmd/define` test failures were already there at base or came from this move.

Embedded, good. Running the external consumer with the correct wiring.

```verdict
verdict: SHIP
confidence: high
```

**VERDICT: SHIP.** M1 is a clean mechanical move: `internal/llm` became `llm`, `internal/llm/llmtest` became `llm/llmtest`, and `internal/conformance` became `conformance`, with no behavior change. I filtered the whole Go diff for lines other than the import rewrite. What's left is comment path updates, the guard's router path (`conformance/guard_test.go`, `const router = "conformance/conformance.go"`), and one `gofmt` spacing fix (`questionsFor(t, d, opt)`). The checks I ran:
- `go build ./...` and `go vet ./...` pass.
- `go test ./llm/... ./conformance/...` passes.
- `gofmt -l` is clean.
- The plan's outside-consumer check holds. A scratch module under `$TMPDIR`, using `replace github.com/xianxu/tools => <checkout>` with `GOPROXY=off`, imports `llm` and `llmtest`. It ran `llmtest.Suite` against `NewFake` and passed. Its dependency list has only `llm`, `llm/llmtest` and `conformance`, so no `internal/` package leaks.
- `grep -rn "internal/llm\|internal/conformance"`, run outside `workshop/`, finds only the intended historical sentence in `AGENTS.local.md:20`.

`go test ./cmd/define/` fails, but **I ran the same tests at base 2ff2d9d and they fail identically there**, so this range didn't cause them:
- **Sandbox "operation not permitted":** `TestLanguagePromptStartup`, `TestLanguageTintInvocation`, `TestSavedSchemeGovernsALookup`.
- **`TestPlanCitesTestsThatExist`:** an earlier issue's plan (000082) cites a test, `TestPTY`, that doesn't exist.

Nothing blocks SHIP.

1. **Strengths**
   - The `conformance` move is justified by Go's cross-module `internal/` rule, and it keeps one owner for `SkipOrFail` instead of copying it (plan §Import paths). ARCH-DRY is preserved.
   - The skip guard keeps enforcing after the move: the router path in `conformance/guard_test.go` was updated, and `TestEverySkipIsRoutedOrWaived` passes. A missed path here would have made the guard flag its own router.
   - Captures are embedded (`llm/llmtest/fake.go:34`), so an outside consumer needs no copy of `testdata`. I confirmed this by running the suite from outside the repo.
   - The carve-out in `AGENTS.local.md` is reworded with a dated operator decision (tools#85) instead of being silently stretched. `.gitattributes` and `scripts/llm-probe.sh` paths are updated too, so SSE byte preservation and capture recording keep working.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - Historical accounts now name a path that didn't exist when the events happened. This affects `conformance/conformance.go` package doc ("Measured, `… ./llm/...` reported ok"), `conformance/guard_test.go` round 4, and `atlas/define.md:2772`. Past measurements should keep `internal/llm` or say "(then `internal/llm`)".
   - The milestone checkbox `- [ ] M1` in the plan's `## Milestones` is still unticked, while the issue has `- [x] M1` and Task 1's steps are all `[x]`.
   - In `llm/conformance_test.go`, the `github.com/xianxu/tools/conformance` import sits in the standard-library group. `gofmt` accepts this, but `goimports` grouping would move it.

5. **Test coverage notes:** A move has no new behavior to test. The existing suites plus the outside-consumer build are the right evidence, and both pass. The golden hash is unchanged: `llm/testdata/golden/schema-veto-verdict.txt` was moved with no content change and `llm` tests pass.

6. **Architecture notes for upcoming work (all nine principles pass for M1):**
   - **ARCH-DRY, pass:** no duplication; `conformance` was moved rather than copied.
   - **ARCH-PURE, pass:** no logic changed.
   - **ARCH-PURPOSE, pass:** M1 delivers what it claims. The issue's purpose (Messages, Tools, Delta, ErrCanceled, BeforeAttempt) is planned for M2/M3, not deferred out of scope.
   - **ARCH-MOCK, pass:** the fake, cassettes and live conformance suite moved intact, and the fake is usable from outside the repo.
   - **ARCH-CONSTRAINTS, N/A:** no runtime path changed.
   - **ARCH-SECURE, pass:** no new input or credential handling; `llm.Redact` is still used in the conformance log.
   - **ARCH-ORDER, N/A for M1:** no state between events changed. M2's `callState` and the cancel/`BeforeAttempt` ordering will need sequence tests with injectable ordering.
   - **ARCH-FUNERAL, pass:** the move creates nothing durable.
   - **ARCH-EVOLVE, pass:** no persisted format changed; cassette hashes stay stable.
   - **Public surface:** `conformance` now publicly exports `StrictEnv`, `Strict` and `SkipOrFail`, which become v0.x API at the `v0.1.0` tag. The `llm` package doc doesn't yet state the planned "v0.x may break at a minor bump" policy. That belongs to M2/M3, but it must land before the tag.

7. **Plan revision recommendations:** none needed for code drift. Just tick `M1` in the plan's `## Milestones` at milestone close.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: historical-prose-rewritten-by-mechanical-rename
    title: |
      Path rewrite changed historical accounts to name `llm` where the past measurement was against `internal/llm`
    detail: |
      conformance/conformance.go package doc ("Measured, ... ./llm/... reported ok"), conformance/guard_test.go round-4 comment, and atlas/define.md:2772 now describe past events with a path that did not exist then; keep `internal/llm` (or annotate "then internal/llm") in historical narrative.
  - id: new
    severity: Minor
    kind: defect
    family: tracker-checkbox-state-diverges
    title: |
      Plan's Milestones list still shows `- [ ] M1` while the issue marks M1 done and every Task 1 step is checked
    detail: |
      workshop/plans/000085-llm-public-seam-plan.md line 369; tick it at milestone close so plan and issue agree.
  - id: new
    severity: Minor
    kind: defect
    family: import-grouping
    title: |
      llm/conformance_test.go places the repo-local conformance import inside the stdlib import group
    detail: |
      gofmt-clean but not goimports-grouped; move it into the github.com/xianxu/tools group.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cec2c4eb-6a5c-4bab-9f90-62516561042d","repository":"github.com/xianxu/tools","issue_id":"000085","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/0a791a0ee029577e","reviewed_head":"12082073a2925d1c8cbf8086d62118446152a51b","slot":"tools:1","requested_at":"2026-10-22T03:47:14.017665Z"},"phase":"finished","pid":95561,"started_at":"2026-10-22T03:47:14.192099Z","finished_at":"2026-10-22T03:54:18.655298Z","session_id":"aabeb59d-dd8b-421e-b6ed-b66bc7cc7503","outcome":"returned"}] -->
