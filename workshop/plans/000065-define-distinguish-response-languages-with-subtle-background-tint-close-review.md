# Boundary Review — tools#65 (whole-issue close)

| field | value |
|-------|-------|
| issue | 65 — define: distinguish response languages with subtle background tint |
| repo | tools |
| issue file | workshop/issues/000065-define-distinguish-response-languages-with-subtle-background-tint.md |
| boundary | whole-issue close |
| milestone | — |
| window | 635dde9f94d53de2aaa9a5de42119a83c22ebb39..62c6a6671793afa1b1f93d43fa49dded9770541c |
| command | sdlc close --issue 65 |
| reviewer | codex |
| timestamp | 2026-09-15T14:25:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation covers the main presentation paths and passes the `define` test suite. One correctness gap blocks shipping: dictionary rendering treats the selected study language as verified source ownership, even when dictionary selection falls back to searching every active dictionary.

## 1. Strengths

- Dictionary provenance uses source offsets and validated HTML ownership, preserving repeated-word distinctions.
- Annotation decoding bounds buffering, removes terminal controls, and flushes partial answers before recording history.
- Practice text and ownership share layout builders; answer markings retain precedence.
- Tool README and atlas document the new flag, rendering flow, and conformance seam.

## 2. Critical findings

**Source language is inferred from `/lang` on fallback dictionary paths — ARCH-PURPOSE, ARCH-SECURE.**

[main.go:1076](/Users/xianxu/workspace/tools/cmd/define/main.go:1076) passes `Language: d.lang`; [dictionary_language.go:176](/Users/xianxu/workspace/tools/cmd/define/dictionary_language.go:176) then assigns that language to primary prose without source metadata.

However, `dictionaryFor` explicitly falls back to every active dictionary when selection is unavailable or no matching dictionary exists. `systemDictionary` implements that fallback through `noadDictionary`. For example, `/lang it` without an Italian dictionary can return English prose and tint it as Italian.

The same assumption appears in [cloze.go:236](/Users/xianxu/workspace/tools/cmd/define/cloze.go:236) and [play_loop.go:1040](/Users/xianxu/workspace/tools/cmd/define/play_loop.go:1040).

**Fix:** Carry verified source ownership separately from the study language. Unknown fallback sources must remain neutral. Cover lookup and both practice-reveal paths with regressions that fail under the current implementation.

## 3. Important findings

None separate from the regression coverage required above.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed independently:

- `go test ./cmd/define/... -count=1` — all packages passed; main package took 127.427s.
- Focused dictionary-selection, monolingual-rendering, invocation, and practice tests.
- Pinned-range `git diff --check`.

Existing tests cover fallback selection and tinting separately, but miss their incorrect composition. Live model/native conformance was inspected, not rerun.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass: shared composer and producer-owned presentation builders. |
| ARCH-PURE | Pass: parsing, projection, and style logic remain separate from IO. |
| ARCH-PURPOSE | **Flag:** fallback rendering violates unknown-language neutrality. |
| ARCH-MOCK | Pass: existing stateful transport fake and captured records are reused. |
| ARCH-CONSTRAINTS | Pass: bounded annotation/control handling; no new background workload. |
| ARCH-SECURE | **Flag:** source ownership is asserted without evidence on fallback paths. |
| ARCH-ORDER | Pass: explicit decoder transitions and termination flushing are exercised. |
| ARCH-FUNERAL | Pass: ownership metadata is transient; no new growing runtime artifact. |

## 7. Plan revision recommendations

Append a `## Revisions` entry defining verified dictionary source language independently of `/lang`, enumerating the three rendering consumers and their fallback regression tests.

```findings
findings:
  - id: new
    severity: Critical
    family: source-ownership-requires-provenance
    title: |
      Dictionary fallback output inherits the study language without source evidence
    detail: |
      main.go:1076, cloze.go:236, and play_loop.go:1040 pass d.lang as source ownership, which dictionary_language.go:176 applies to primary prose. dictionaryFor/systemDictionary can instead search every active dictionary, so an Italian session can tint an English fallback definition as Italian. Carry verified source ownership separately, leave unknown fallback text neutral, and add failing regressions across lookup and practice reveals. ARCH-PURPOSE, ARCH-SECURE.
```
