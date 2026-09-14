# Boundary Review — tools#36 (whole-issue close)

| field | value |
|-------|-------|
| issue | 36 — a question has no progress indicator, and it is the one slow path |
| repo | tools |
| issue file | workshop/issues/000036-ask-progress.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7c887c4b90cd889bb2e1f8cdb5ee9c7d8f2d1c3..6aafa5e471358930dfb399403d3e60b8d059fe60 |
| command | sdlc close --issue 36 |
| reviewer | codex |
| timestamp | 2026-09-14T00:56:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the issue’s approved specification and plan. Foreground LLM calls share one activity lifecycle, cleanup precedes answer output, and terminal suppression and model provenance are preserved. No blocking findings.

1. **Strengths**
   - `activity.go` centralizes cancellation, ticker teardown, and synchronous worker joining.
   - Both display hosts enforce lease ownership, preventing stale updates or cleanup from erasing a replacement activity.
   - `llm_activity.go` covers discovery and stops before forwarding the first nonempty delta.
   - Screen tests exercise prompt geometry, footer hit-testing, resize, and transcript preservation. Relevant README and atlas updates are present.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   
   Independently passed:
   - Full `cmd/define` and `internal/llm/...` package tests.
   - Focused activity, screen, and fake-barrier race tests.
   - Strict PTY activity response/cancellation conformance.
   - Shared conformance checks and pinned-range `git diff --check`.

6. **Architecture**
   - **ARCH-DRY — pass:** Shared lifecycle, painter, and write-error recorder.
   - **ARCH-PURE — pass:** Listed pure entities perform no IO; display effects remain behind hosts.
   - **ARCH-PURPOSE — pass:** All current foreground consumers are covered, including harvest agreement.
   - **ARCH-MOCK — pass:** Stateful HTTP fake, controlled barriers, terminal oracle, and PTY conformance exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** Fixed 80 ms cadence and existing screen throttle bound animation work.
   - **ARCH-SECURE — pass:** Spinner content is fixed; credentials and provider responses do not enter activity state.
   - **ARCH-ORDER — pass:** Explicit lease replacement and synchronous stop prevent stale output; tests control event ordering.
   - **ARCH-FUNERAL — pass:** Timers, workers, and overlays are released; activity creates no durable records.

7. **Plan revision recommendations:** None. Concept declarations and documented revisions match the implementation.

```findings
{}
```
