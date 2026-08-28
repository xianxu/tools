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
    - "n": 4
      timestamp: "2026-08-28T12:55:52-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: userModelFile() is per-language and MigrateToLanguages moves the flat file; verified by reverting yaml.go:115 in a scratch copy of e393f2d9 — TestTheUserModelIsPerLanguage goes red with the exact cross-language clobber.
          round: 4
        - id: BR-3
          disposition: addressed
          note: applyLang no longer assigns opt.lang and command.go:371-374 states why a switch does not retroactively make the flag present.
          round: 4
        - id: BR-4
          disposition: addressed
          note: .tmp-* is in RuntimeFiles, .gitignore and both basename guards; git check-ignore -v matches .tmp-abc123 and still misses the golden fixture. See BR-7 for the remaining pin gap, which is a different rule.
          round: 4
        - id: BR-5
          disposition: addressed
          note: store/lang.go:39-43 now names the CDN's _<lang>_<locale>_ shape and the path segment as what fixes the length, with pt-br and ISO 639-3 refused deliberately.
          round: 4
      findings:
        - id: BR-6
          severity: Important
          title: The user-model rename swept 3 of ~12 restatements, including --reflect's success line, which names a file it did not write
          detail: '3rd finding in this family — do NOT fix the instance. The rule, which this commit already stated for itself: no string, comment, README line, atlas line or plan line may spell a runtime artifact''s filename or a code-owned symbol name; name the artifact, or derive the name. Enforceable with the legacyRuntimeFilePaths exact-set ratchet idiom. Measured prevalence at HEAD: reflect.go:405 prints "wrote user-model.md" after writing user-model.<lang>.md; store/yaml.go:89 says userModelFile is NOT per-language, 26 lines above yaml.go:115; comments at askctx.go:37, store/store.go:25, store/mem.go:17; README 142/148/218 contradict README 182; atlas/define.md 808/821/908/961 contradict 220. Same rule, symbol form: main.go:248 and atlas/define.md:238 name MigrateFlatDeck, which the tree does not export (MigrateToLanguages does; migrateFlatDeck is unexported), and the plan names it at :80, :143, :186, :258, :296 — round 1''s I4 (deckDeps) recurring one function further out.'
          family: comment-contract-drift
          round: 4
        - id: BR-7
          severity: Important
          title: RuntimeFiles coverage is asserted from hand-typed literals, so changing the atomic-write temp prefix escapes every guard green
          detail: '2nd finding in this family — do NOT fix the .tmp- instance. The rule: a runtime artifact''s name has exactly one producing function; guards, migrations and tests derive from it and never restate it. langFile/langFileName is already the model. Mutation-verified on a scratch copy of e393f2d9: os.CreateTemp(filepath.Dir(path), "tmp-*") at store/yaml.go:337 leaves go test ./cmd/define/store/ fully green, because store/lang_test.go:142 appends the literal ".tmp-123456" rather than observing the writer — while atlas/repo-guards.md:96-100 documents that test as asserting the real output of the writing functions. The shadow beside user-model.<lang>.md and lang.txt in the working-directory root then matches no .gitignore pattern. Same shape at store/migrate.go:117, which hand-writes "user-model."+DefaultLang+".md" instead of calling userModelFile() in its own package, with migrate_test.go:186 asserting the same literal. Prevalence: 3 writer names, 1 derived. Fix: a shared tmpPattern const consumed by os.CreateTemp and RuntimeFiles, migrateUserModel deriving its destination, and no literals left in TestRuntimeFilePatternsCoverWhatWeWrite.'
          family: runtime-artifact-guard-coverage
          round: 4
        - id: BR-8
          severity: Minor
          title: The project file records M1's actual and closed date from before three rounds of boundary-review fixes
          detail: workshop/projects/define-learn.md:405-406 carry actual 1.03h and closed 2026-08-28, written in 30ee9e2 before the C1/I1-I4 and BR-2..BR-5 fix commits. Re-measure at the real close rather than leaving a number that understates by the whole review cost.
          family: project-ledger-lag
          round: 4
        - id: BR-9
          severity: Minor
          title: README never states that -locale is English-only, three lines above the new -lang example
          detail: README.md:37 shows `define -locale gb colour`; D2's rule (honoured for English only, with a diagnostic otherwise) is documented in atlas/define.md:1084-1091 and nowhere in README. Belongs to the same sweep as the other finding in this family.
          family: comment-contract-drift
          round: 4
        - id: BR-10
          severity: Minor
          title: The PATTERNS explanation at store/yaml.go:75-87 is a detached comment godoc attaches to nothing
          detail: Blank line before it and after it, so it documents neither RuntimeFiles nor wordsDir. Merge it into the RuntimeFiles doc comment. Same for store/migrate.go:11, whose opening line says MigrateToLanguages moves a DIRECTORY when it moves two named artifacts.
          family: comment-contract-drift
          round: 4
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

## Round 4 — 2026-08-28T12:55:52-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — userModelFile() is per-language and MigrateToLanguages moves the flat file; verified by reverting yaml.go:115 in a scratch copy of e393f2d9 — TestTheUserModelIsPerLanguage goes red with the exact cross-language clobber.
- BR-3 — addressed — applyLang no longer assigns opt.lang and command.go:371-374 states why a switch does not retroactively make the flag present.
- BR-4 — addressed — .tmp-* is in RuntimeFiles, .gitignore and both basename guards; git check-ignore -v matches .tmp-abc123 and still misses the golden fixture. See BR-7 for the remaining pin gap, which is a different rule.
- BR-5 — addressed — store/lang.go:39-43 now names the CDN's _<lang>_<locale>_ shape and the path segment as what fixes the length, with pt-br and ISO 639-3 refused deliberately.

### Raised

- **BR-6** [Important] `comment-contract-drift` The user-model rename swept 3 of ~12 restatements, including --reflect's success line, which names a file it did not write
  3rd finding in this family — do NOT fix the instance. The rule, which this commit already stated for itself: no string, comment, README line, atlas line or plan line may spell a runtime artifact's filename or a code-owned symbol name; name the artifact, or derive the name. Enforceable with the legacyRuntimeFilePaths exact-set ratchet idiom. Measured prevalence at HEAD: reflect.go:405 prints "wrote user-model.md" after writing user-model.<lang>.md; store/yaml.go:89 says userModelFile is NOT per-language, 26 lines above yaml.go:115; comments at askctx.go:37, store/store.go:25, store/mem.go:17; README 142/148/218 contradict README 182; atlas/define.md 808/821/908/961 contradict 220. Same rule, symbol form: main.go:248 and atlas/define.md:238 name MigrateFlatDeck, which the tree does not export (MigrateToLanguages does; migrateFlatDeck is unexported), and the plan names it at :80, :143, :186, :258, :296 — round 1's I4 (deckDeps) recurring one function further out.
- **BR-7** [Important] `runtime-artifact-guard-coverage` RuntimeFiles coverage is asserted from hand-typed literals, so changing the atomic-write temp prefix escapes every guard green
  2nd finding in this family — do NOT fix the .tmp- instance. The rule: a runtime artifact's name has exactly one producing function; guards, migrations and tests derive from it and never restate it. langFile/langFileName is already the model. Mutation-verified on a scratch copy of e393f2d9: os.CreateTemp(filepath.Dir(path), "tmp-*") at store/yaml.go:337 leaves go test ./cmd/define/store/ fully green, because store/lang_test.go:142 appends the literal ".tmp-123456" rather than observing the writer — while atlas/repo-guards.md:96-100 documents that test as asserting the real output of the writing functions. The shadow beside user-model.<lang>.md and lang.txt in the working-directory root then matches no .gitignore pattern. Same shape at store/migrate.go:117, which hand-writes "user-model."+DefaultLang+".md" instead of calling userModelFile() in its own package, with migrate_test.go:186 asserting the same literal. Prevalence: 3 writer names, 1 derived. Fix: a shared tmpPattern const consumed by os.CreateTemp and RuntimeFiles, migrateUserModel deriving its destination, and no literals left in TestRuntimeFilePatternsCoverWhatWeWrite.
- **BR-8** [Minor] `project-ledger-lag` The project file records M1's actual and closed date from before three rounds of boundary-review fixes
  workshop/projects/define-learn.md:405-406 carry actual 1.03h and closed 2026-08-28, written in 30ee9e2 before the C1/I1-I4 and BR-2..BR-5 fix commits. Re-measure at the real close rather than leaving a number that understates by the whole review cost.
- **BR-9** [Minor] `comment-contract-drift` README never states that -locale is English-only, three lines above the new -lang example
  README.md:37 shows `define -locale gb colour`; D2's rule (honoured for English only, with a diagnostic otherwise) is documented in atlas/define.md:1084-1091 and nowhere in README. Belongs to the same sweep as the other finding in this family.
- **BR-10** [Minor] `comment-contract-drift` The PATTERNS explanation at store/yaml.go:75-87 is a detached comment godoc attaches to nothing
  Blank line before it and after it, so it documents neither RuntimeFiles nor wordsDir. Merge it into the RuntimeFiles doc comment. Same for store/migrate.go:11, whose opening line says MigrateToLanguages moves a DIRECTORY when it moves two named artifacts.

## Open findings

- **BR-6** [Important] `comment-contract-drift` The user-model rename swept 3 of ~12 restatements, including --reflect's success line, which names a file it did not write
- **BR-7** [Important] `runtime-artifact-guard-coverage` RuntimeFiles coverage is asserted from hand-typed literals, so changing the atomic-write temp prefix escapes every guard green
- **BR-8** [Minor] `project-ledger-lag` The project file records M1's actual and closed date from before three rounds of boundary-review fixes
- **BR-9** [Minor] `comment-contract-drift` README never states that -locale is English-only, three lines above the new -lang example
- **BR-10** [Minor] `comment-contract-drift` The PATTERNS explanation at store/yaml.go:75-87 is a detached comment godoc attaches to nothing
