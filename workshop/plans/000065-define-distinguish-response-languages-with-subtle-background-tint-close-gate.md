---
gate: boundary-review
issue: 65
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-15T14:25:15-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Dictionary fallback output inherits the study language without source evidence
          detail: main.go:1076, cloze.go:236, and play_loop.go:1040 pass d.lang as source ownership, which dictionary_language.go:176 applies to primary prose. dictionaryFor/systemDictionary can instead search every active dictionary, so an Italian session can tint an English fallback definition as Italian. Carry verified source ownership separately, leave unknown fallback text neutral, and add failing regressions across lookup and practice reveals. ARCH-PURPOSE, ARCH-SECURE.
          family: source-ownership-requires-provenance
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-15T14:38:03-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Selected-ID metadata establishes source ownership; wrappers preserve it and renderDefinitions applies it independently of the study language. Regression tests cover lookup, Choice/Cloze reveals, and practice glosses. Replacing section ownership with the study language compiled and failed the lookup and both reveal regressions.
          round: 2
      blocked: false
---

# Gate ledger — tools#65 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-15T14:25:15-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `source-ownership-requires-provenance` Dictionary fallback output inherits the study language without source evidence
  main.go:1076, cloze.go:236, and play_loop.go:1040 pass d.lang as source ownership, which dictionary_language.go:176 applies to primary prose. dictionaryFor/systemDictionary can instead search every active dictionary, so an Italian session can tint an English fallback definition as Italian. Carry verified source ownership separately, leave unknown fallback text neutral, and add failing regressions across lookup and practice reveals. ARCH-PURPOSE, ARCH-SECURE.

## Round 2 — 2026-09-15T14:38:03-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Selected-ID metadata establishes source ownership; wrappers preserve it and renderDefinitions applies it independently of the study language. Regression tests cover lookup, Choice/Cloze reveals, and practice glosses. Replacing section ownership with the study language compiled and failed the lookup and both reveal regressions.

## Open findings

(none — every finding has been disposed)
