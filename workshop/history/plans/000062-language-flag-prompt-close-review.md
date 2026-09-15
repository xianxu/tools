# Boundary Review — tools#62 (whole-issue close)

| field | value |
|-------|-------|
| issue | 62 — define: show language flag before prompt |
| repo | tools |
| issue file | workshop/issues/000062-language-flag-prompt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 62c5bbfca9df730725bc94f99fdb4bd0d1b63886..fe1ef33601ec8cb59ed40a95f6330ca9ba289b75 |
| command | sdlc close --issue 62 |
| reviewer | codex |
| timestamp | 2026-09-15T13:06:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches tools#62’s Spec and Plan. Language indicators follow effective session language across startup, switching, submitted lines, and both prompt paths. Flag geometry and fallback behavior have meaningful regression coverage. No blocking findings.

## 1. Strengths

- `language_prompt.go` centralizes mapping, normalization, and safe fallback.
- `replraw.go` derives each prompt from current language and terminal width; submitted commands retain their previous language.
- `display_unit.go` supplies shared flag boundaries across measurement, wrapping, clipping, and selection.
- Geometry tests assert literal rows, cursor controls, and copied text independently of the display helper.
- The command README and atlas document the new flag, behavior, and terminal limitations.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Independently passed:

- Focused language, flag, and prompt tests.
- Race-enabled prompt, screen, selection, rendering, and highlighting tests.
- Strict PTY language-prompt conformance, including flag/code modes and raw editing.
- Shared conformance checks and pinned-range whitespace validation.

The full repository suite and reported mutation checks were not independently repeated. Host-font glyph width remains an explicitly documented manual check.

## 6. Architectural notes

- **ARCH-DRY — pass:** One prompt policy and shared display-unit reader; relevant consumers use them.
- **ARCH-PURE — pass:** Policy and geometry remain pure; terminal inspection stays in integration callers. Core-concept classifications match implementation.
- **ARCH-PURPOSE — pass:** All planned prompt paths and language transitions are covered.
- **ARCH-MOCK — pass:** Stateful screen/store seams complement actual PTY conformance.
- **ARCH-CONSTRAINTS — pass:** Prefixes are bounded; display scans retain linear complexity and narrow-width fallback.
- **ARCH-SECURE — pass:** Language validation prevents malformed fallback text from injecting controls.
- **ARCH-ORDER — pass:** Existing language transitions remain authoritative; rendering adds no asynchronous state.
- **ARCH-FUNERAL — pass:** No new durable runtime artifact or background worker.

## 7. Plan revision recommendations

None. The remaining visual check is already recorded.

```findings
{}
```
