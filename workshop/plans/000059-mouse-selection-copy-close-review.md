# Boundary Review — tools#59 (whole-issue close)

| field | value |
|-------|-------|
| issue | 59 — define: mouse text selection and clipboard copy |
| repo | tools |
| issue file | workshop/issues/000059-mouse-selection-copy.md |
| boundary | whole-issue close |
| milestone | — |
| window | eec60c0c0d20cb45c55f656d7c3665e44f7e9973..8f03ea8881a680c10af680af4f8f8eb42b2b416c |
| command | sdlc close --issue 59 |
| reviewer | codex |
| timestamp | 2026-09-14T06:50:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared painter, gesture policy and clipboard worker form a coherent implementation, and the package tests pass. One cancellation gap blocks shipping: some input paths bypass selection invalidation, allowing an unfinished drag to copy after cancellation. Native clipboard verification was unavailable in this environment.

1. **Strengths**

   - Painting and extraction share physical-row geometry, with tests covering Unicode, clipping and cursor preservation.
   - Both consumers validate screen/frame ownership and capture immutable click targets before dispatch.
   - Clipboard writes are serialized, bounded and cancellable; conformance tests use isolated pasteboards.
   - The tool README, help and atlas document the new interaction.

2. **Critical findings**

   **Cancellation must reach selection independently of input admission.** At [selection_input.go:177](/Users/xianxu/workspace/tools/cmd/define/selection_input.go:177), a full type-ahead queue drops ordinary keys before `router.route` can cancel the gesture. Once `saturated` is already true, subsequent drops produce no repaint either. Start a fresh drag while that queue remains full, type or scroll, then release: the gesture survives and can copy, contrary to the cancellation contract.

   The same omission exists for scoped SIGINT: [interrupt.go:85](/Users/xianxu/workspace/tools/cmd/define/interrupt.go:85) calls `interrupts.Fire()` without invalidating selection. During model cancellation, release can therefore arrive before the owning loop redraws.

   Route cancellation at event observation, before queue admission, and cover both interrupt transports. Add deterministic tests asserting zero clipboard writes after these events. **ARCH-ORDER / ARCH-PURPOSE.**

3. **Important findings:** None separately.

4. **Minor findings:** None.

5. **Test coverage notes**

   - `go test ./cmd/define -count=1`: passed, 111.498s.
   - Focused selection/clipboard tests and focused race checks: passed.
   - Pinned-range `git diff --check`: passed.
   - Strict native and PTY selection checks failed during isolated pasteboard creation, status `-4960`; native behavior remains unverified here.
   - `TestSelectionInputNeverWaitsBehindTypeahead` asserts interrupt delivery, but never asserts clipboard effects. It cannot catch the cancellation gap.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared layout, routing and click validation. |
   | ARCH-PURE | Pass: gesture and extraction logic remain directly testable without IO. |
   | ARCH-PURPOSE | Flag: cancellation contract is incomplete across input paths. |
   | ARCH-MOCK | Pass structurally: stateful clipboard/process fixtures and native conformance seam; live verification unavailable here. |
   | ARCH-CONSTRAINTS | Pass: explicit frame, payload, queue and subprocess bounds. |
   | ARCH-SECURE | Pass: validated literal input, shell-free execution and isolated test targets. |
   | ARCH-ORDER | Flag: admission and interrupt transport can bypass gesture cancellation. |
   | ARCH-FUNERAL | Pass: shutdown cancels and joins workers; helper watchdog bounds orphan lifetime. |

7. **Plan revision recommendation**

   Append a `## Revisions` entry enumerating cancellation ingress: admitted and rejected keyboard/page/wheel events, byte Ctrl-C and scoped SIGINT. Specify cancellation before admission or asynchronous completion, with clipboard-effect regression tests.

```findings
findings:
  - id: new
    severity: Critical
    family: cancellation-ingress-completeness
    title: |
      Selection cancellation is bypassed by saturated input and scoped SIGINT
    detail: |
      cmd/define/selection_input.go:177-182 drops keys before router cancellation; after saturation is established, typing or scrolling during a fresh drag leaves it active and release can copy. Scoped SIGINT likewise reaches interrupts.Fire at cmd/define/interrupt.go:85 without selection invalidation, allowing release before the cancelled operation redraws. Apply cancellation at observation across all ingress paths and add deterministic zero-copy regression tests (ARCH-ORDER, ARCH-PURPOSE).
```
