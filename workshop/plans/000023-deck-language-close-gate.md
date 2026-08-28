---
gate: boundary-review
issue: 23
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-28T12:12:35-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
          detail: |-
            Task 7 Step 3 and Task 10 Step 3 call milestone-close/close, but the issue's
            Plan holds a single non-Mx row. close.go:554 matches the Mx checkbox against
            the ISSUE body and only warns on a miss (close.go:560), and
            findMilestonesMissingVerdict (close.go:1717) reads that same section — so at
            the full close the "was M1 reviewed" guard finds zero milestones and passes
            vacuously. Add the two Mx rows to the issue's Plan before starting M1.
            (carried from plan-quality PQ-8, deferred to the boundary review)
          family: declared-boundary-untracked
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-28T12:12:35-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-08-28T12:31:12-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: The issue's Plan now carries `- [ ] M1 —` and `- [ ] M2 —` rows; I checked them against the binary's own regexes (close.go:554 tick pattern and milestonePlanRE at close.go:1667) and both match, so the milestone-verdict guard will no longer pass vacuously.
          round: 3
      findings:
        - id: BR-2
          severity: Important
          title: user-model.md is derived from the language-scoped deck but stored unscoped, so a --reflect in one language replaces the other language's model
          detail: 'reflect.go:318 reads the language-scoped d.deck.Deck() while reflect.go:397 writes the unscoped userModelFile() (store/yaml.go:88), and ask.go:263 reads that one file in every language. Reproduced with the existing reflectRig: --reflect over an English deck, then over a Spanish deck in the same directory, leaves the English session reading "A2 — Spanish beginner / Read off: madrugar". This is the same enumeration class C1 named, one member further out — a persisted artifact derived from the language. atlas/define.md, shipped in this range, justifies leaving it unscoped with the events/ argument, which does not transfer.'
          family: language-derived-state-unscoped
          round: 3
        - id: BR-3
          severity: Minor
          title: applyLang assigns opt.lang, contradicting that field's own documented meaning
          detail: command.go:367 writes opt.lang = l, while main.go:329-333 documents options.lang as "the -lang FLAG, empty when it was not given — not the language in effect". Nothing reads it after withStore, so the behaviour is fine; the comment is false after a switch.
          family: comment-contract-drift
          round: 3
        - id: BR-4
          severity: Minor
          title: the .tmp-* shadows writeBytesAtomic leaves beside a runtime FILE are covered by neither .gitignore nor the basename guards
          detail: store/yaml.go:310 creates .tmp-* in the target's directory. For words/ and usage/ that directory is itself ignored; for lang.txt and user-model.md it is the working-directory root, where .gitignore has no .tmp-* pattern and isRuntimeFile cannot match a random name. Pre-existing for user-model.md, but it is the part of the RuntimeFiles class the class fix does not reach.
          family: runtime-artifact-guard-coverage
          round: 3
        - id: BR-5
          severity: Minor
          title: ParseLang's stated rationale argues against a whitelist, not for the two-letter limit it actually imposes
          detail: store/lang.go:41 requires exactly two ASCII letters, refusing pt-br, zh-hans and ISO 639-3 tags. The comment explains only why there is no list of known languages. One sentence naming the CDN's _xx_yy_ path shape as what fixes the length would make the constraint a decision rather than an artifact.
          family: comment-contract-drift
          round: 3
      boundary: M1
      blocked: true
---

# Gate ledger — tools#23 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T12:12:35-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `declared-boundary-untracked` The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
  Task 7 Step 3 and Task 10 Step 3 call milestone-close/close, but the issue's
  Plan holds a single non-Mx row. close.go:554 matches the Mx checkbox against
  the ISSUE body and only warns on a miss (close.go:560), and
  findMilestonesMissingVerdict (close.go:1717) reads that same section — so at
  the full close the "was M1 reviewed" guard finds zero milestones and passes
  vacuously. Add the two Mx rows to the issue's Plan before starting M1.
  (carried from plan-quality PQ-8, deferred to the boundary review)

## Round 2 — 2026-08-28T12:12:35-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-08-28T12:31:12-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — The issue's Plan now carries `- [ ] M1 —` and `- [ ] M2 —` rows; I checked them against the binary's own regexes (close.go:554 tick pattern and milestonePlanRE at close.go:1667) and both match, so the milestone-verdict guard will no longer pass vacuously.

### Raised

- **BR-2** [Important] `language-derived-state-unscoped` user-model.md is derived from the language-scoped deck but stored unscoped, so a --reflect in one language replaces the other language's model
  reflect.go:318 reads the language-scoped d.deck.Deck() while reflect.go:397 writes the unscoped userModelFile() (store/yaml.go:88), and ask.go:263 reads that one file in every language. Reproduced with the existing reflectRig: --reflect over an English deck, then over a Spanish deck in the same directory, leaves the English session reading "A2 — Spanish beginner / Read off: madrugar". This is the same enumeration class C1 named, one member further out — a persisted artifact derived from the language. atlas/define.md, shipped in this range, justifies leaving it unscoped with the events/ argument, which does not transfer.
- **BR-3** [Minor] `comment-contract-drift` applyLang assigns opt.lang, contradicting that field's own documented meaning
  command.go:367 writes opt.lang = l, while main.go:329-333 documents options.lang as "the -lang FLAG, empty when it was not given — not the language in effect". Nothing reads it after withStore, so the behaviour is fine; the comment is false after a switch.
- **BR-4** [Minor] `runtime-artifact-guard-coverage` the .tmp-* shadows writeBytesAtomic leaves beside a runtime FILE are covered by neither .gitignore nor the basename guards
  store/yaml.go:310 creates .tmp-* in the target's directory. For words/ and usage/ that directory is itself ignored; for lang.txt and user-model.md it is the working-directory root, where .gitignore has no .tmp-* pattern and isRuntimeFile cannot match a random name. Pre-existing for user-model.md, but it is the part of the RuntimeFiles class the class fix does not reach.
- **BR-5** [Minor] `comment-contract-drift` ParseLang's stated rationale argues against a whitelist, not for the two-letter limit it actually imposes
  store/lang.go:41 requires exactly two ASCII letters, refusing pt-br, zh-hans and ISO 639-3 tags. The comment explains only why there is no list of known languages. One sentence naming the CDN's _xx_yy_ path shape as what fixes the length would make the constraint a decision rather than an artifact.

## Open findings

- **BR-2** [Important] `language-derived-state-unscoped` user-model.md is derived from the language-scoped deck but stored unscoped, so a --reflect in one language replaces the other language's model
- **BR-3** [Minor] `comment-contract-drift` applyLang assigns opt.lang, contradicting that field's own documented meaning
- **BR-4** [Minor] `runtime-artifact-guard-coverage` the .tmp-* shadows writeBytesAtomic leaves beside a runtime FILE are covered by neither .gitignore nor the basename guards
- **BR-5** [Minor] `comment-contract-drift` ParseLang's stated rationale argues against a whitelist, not for the two-letter limit it actually imposes
