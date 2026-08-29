---
gate: plan-quality
issue: 27
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-28T22:21:26-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Task 1 and the Risks section specify opposite policies for the same function
          detail: |-
            Task 1 requires a closed per-language valid-pair table that refuses an
            invalid pair by name; Risks says such a table is "deliberately not
            validated" and specifies warn-and-404 instead, naming the same es plus gb
            case. The Risks section is a pre-23 artifact the rewrite did not sweep.
            Pick one and delete the other before implementation starts.
          family: contradictory-design-statement
          round: 1
        - id: PQ-2
          severity: Important
          title: Risks describes current behavior that the tree contradicts and a function that was never built
          detail: |-
            Risks claims "-locale gb with -lang es produces es_gb, which 404s".
            cmd/define/voice.go:47-56 returns defaultLocale plus a complaint, so es_gb
            is never built; cmd/define/audiourl_test.go:112 pins it and
            atlas/define.md:1136 documents it. The same section reasons about a ~900ms
            fallback and a future voices() that the plan itself records as deleted.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-3
          severity: Important
          title: The per-language locale table is closed over en and es, but ParseLang admits any two-letter tag
          detail: |-
            cmd/define/store/lang.go:33 deliberately accepts any well-formed tag and
            its doc comment refuses a known-language list on principle. The plan does
            not say what "-lang de -locale gb" does under the new table, and it does
            not acknowledge that a closed table diverges from ParseLang's stated
            stance. The existing test already carries a de row, so the gap is live.
          family: closed-set-over-open-domain
          round: 1
        - id: PQ-4
          severity: Important
          title: The locale policy is restated in four places with no derivation (ARCH-DRY, ARCH-PURPOSE)
          detail: |-
            localeFor, the flag help at cmd/define/main.go:409, README.md:196-198 and
            atlas/define.md:1136-1141 will each independently state the policy. The
            docs task asks for a manual sweep. cmd/define/doc_sync_test.go already
            implements derivation for this exact recurring family; the help text
            should be the source const and the README asserted against it.
          family: hand-maintained-restatement
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-28T22:23:28-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 1 and Risks now both specify no whitelist, warn-and-404, same es plus gb case.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Risks now describes post-change behavior that follows from deleting the guard; the 900ms and voices() reasoning is gone.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The closed table is withdrawn in favour of ParseLang's open-domain stance, quoted accurately from store/lang.go:27-31.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Help string becomes the source const with the README deriving via doc_sync_test.go, the mechanism that file already implements.
          round: 2
      blocked: false
content_hash: 9fe5ce16cfac73816c946996711e831ff37fdc4c057169bd6c2f58d50e28c6fc
---

# Gate ledger — tools#27 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T22:21:26-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `contradictory-design-statement` Task 1 and the Risks section specify opposite policies for the same function
  Task 1 requires a closed per-language valid-pair table that refuses an
  invalid pair by name; Risks says such a table is "deliberately not
  validated" and specifies warn-and-404 instead, naming the same es plus gb
  case. The Risks section is a pre-23 artifact the rewrite did not sweep.
  Pick one and delete the other before implementation starts.
- **PQ-2** [Important] `unbacked-existing-behavior-claim` Risks describes current behavior that the tree contradicts and a function that was never built
  Risks claims "-locale gb with -lang es produces es_gb, which 404s".
  cmd/define/voice.go:47-56 returns defaultLocale plus a complaint, so es_gb
  is never built; cmd/define/audiourl_test.go:112 pins it and
  atlas/define.md:1136 documents it. The same section reasons about a ~900ms
  fallback and a future voices() that the plan itself records as deleted.
- **PQ-3** [Important] `closed-set-over-open-domain` The per-language locale table is closed over en and es, but ParseLang admits any two-letter tag
  cmd/define/store/lang.go:33 deliberately accepts any well-formed tag and
  its doc comment refuses a known-language list on principle. The plan does
  not say what "-lang de -locale gb" does under the new table, and it does
  not acknowledge that a closed table diverges from ParseLang's stated
  stance. The existing test already carries a de row, so the gap is live.
- **PQ-4** [Important] `hand-maintained-restatement` The locale policy is restated in four places with no derivation (ARCH-DRY, ARCH-PURPOSE)
  localeFor, the flag help at cmd/define/main.go:409, README.md:196-198 and
  atlas/define.md:1136-1141 will each independently state the policy. The
  docs task asks for a manual sweep. cmd/define/doc_sync_test.go already
  implements derivation for this exact recurring family; the help text
  should be the source const and the README asserted against it.

## Round 2 — 2026-08-28T22:23:28-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 1 and Risks now both specify no whitelist, warn-and-404, same es plus gb case.
- PQ-2 — addressed — Risks now describes post-change behavior that follows from deleting the guard; the 900ms and voices() reasoning is gone.
- PQ-3 — addressed — The closed table is withdrawn in favour of ParseLang's open-domain stance, quoted accurately from store/lang.go:27-31.
- PQ-4 — addressed — Help string becomes the source const with the README deriving via doc_sync_test.go, the mechanism that file already implements.

## Open findings

(none — every finding has been disposed)
