---
gate: plan-quality
issue: 23
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-28T11:00:44-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: /lang has no stated mechanism, and lang-as-constructor-param conflicts with the store graph openStore builds once
          detail: |-
            openStore derives history, capture, deck and vocab from ONE *YAML
            (cmd/define/main.go:202-212), pinned by TestOpenStoreSharesOneHighlightSet
            (cmd/define/vocab_test.go:148). repl takes d deps by value (repl.go:181) and
            rebuilds commandCtx per dispatch (repl.go:322), and commandCtx is
            deliberately narrow (command.go:145-167). A mid-session /lang es must
            re-derive all four and re-Load the vocabulary; Task 4 Step 2 says only "a
            row in command.go's table". Name the mechanism (a setLang closure over the
            loop's deps, on the setTimes precedent at command.go:162-163) or say the
            store reads the setting through, and reconcile whichever with the plan's
            stated "read once at the boundary" architecture.
          family: derived-state-rebuild
          round: 1
        - id: PQ-2
          severity: Important
          title: -lang overlaps the existing -locale flag and the plan never states the precedence
          detail: |-
            main.go:272 already defines -locale ("us or gb", default "us"), threaded via
            speak (main.go:611) into AudioCandidates (audiourl.go:21). Task 5's own test
            asserts voice{Lang "es", Locale "es"} but nothing in the plan derives that
            Locale; with today's default the mode builds Locale "us" and requests
            madrugar_es_us_1.mp3, a form the plan never measured. State the M1 interim
            rule now, since the issue that owns locale policy is blocked on this one.
          family: overlapping-input-precedence
          round: 1
        - id: PQ-3
          severity: Important
          title: MigrateFlatDeck's target language is ambiguous and its collision semantics are undefined
          detail: |-
            The test calls MigrateFlatDeck(dir, DefaultLang) while Step 2 prose says it
            moves into "words/<lang>/" called at the boundary — if that is the ACTIVE
            language, one "define -lang es" first run files an entire English deck under
            words/es/. Separately, "skipping anything already in a subdirectory" does not
            define the case where words/mesa.yaml and words/en/mesa.yaml both exist; the
            plan's own assertion that the flat file must not survive then either loses
            data or reddens the test. Deck() already skips directories
            (store/yaml.go:103-115), so pinning both rules is cheap.
          family: irreplaceable-artifact-guard
          round: 1
        - id: PQ-4
          severity: Important
          title: M2's dictionary seam signature and the Spanish fixture capture path are unnamed (ARCH-MOCK)
          detail: |-
            Dictionary is Lookup(word string) (dict.go:14-16) and systemDictionary()
            takes no argument (dict_darwin.go); the plan never says how the language
            reaches the seam. loadFakeDictionary globs a FLAT testdata/entries/*.txt
            keyed by basename and hard-fails on an empty corpus (dict_fake_test.go:24-49),
            and testdata/capture.sh captures through DCSCopyTextDefinition(NULL, ...).
            "the fake models a SET of dictionaries" therefore requires a corpus layout
            change, a capture.sh that captures through a chosen DCSDictionaryRef, and
            real Spanish captures for mesa and sycophantic. Task 9 Step 1 is one line.
          family: external-seam-fake-incomplete
          round: 1
        - id: PQ-5
          severity: Important
          title: RuntimeFiles reaches only .gitignore though the plan claims RuntimeDirs parity, and its entry shadows a tracked fixture
          detail: |-
            store/yaml.go:34-41 says a runtime name must reach .gitignore, the index
            guard and the history guard; the plan repeats that framing but writes only
            the .gitignore guard. The reason it cannot mirror the other two belongs in
            the plan: cmd/define/testdata/golden/user-model.md is tracked and reachable
            from HEAD, so pointing repo_guard_test.go:200-218 and :240-260 at
            RuntimeFiles by basename fails on contact. The un-anchored user-model.md
            line also matches that golden path — harmless while tracked, silently
            un-addable after a git rm.
          family: single-source-consumer-gap
          round: 1
        - id: PQ-6
          severity: Minor
          title: The plan reproduces roughly 250 lines of the diff instead of one strategy line per risky function
          detail: |-
            Full ParseLang, NewYAML and wordsDir bodies, complete test functions with
            enumerated case tables, and a call-site file inventory. The strategy that
            matters is real but buried: traversal input for ParseLang, the three named
            mutations for chooseDictionary, idempotence for the migration. Compress to
            those lines; the code will state the rest better within the hour.
          family: plan-restates-diff
          round: 1
        - id: PQ-7
          severity: Minor
          title: '"English, because that is what every existing deck holds" is false of this tree'
          detail: |-
            The live deck here is words/ligament.yaml and words/madrugar.yaml. The
            language-blind migration files madrugar — this issue's own headline Spanish
            example — into words/en/, where the Spanish mode cannot see it and where its
            audio request stays the _en_us_ form the issue calls a 404. Say the migration
            is language-blind by design and print what it moved.
          family: unbacked-existing-behavior-claim
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-28T11:09:11-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: D1 names setLang + deckDeps, and catches replraw.go:79's captured voc that the finding did not.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: 'D2 pins the interim locale rule and routes it into atlas/define.md for #27.'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: MigrateFlatDeck loses the lang param; collision and idempotence rules each get a test.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: systemDictionary(lang) with Lookup unchanged; per-language corpus and the capture path are Task 9 Step 1.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: RuntimeFiles reaches all three guards; the squatting golden is renamed first, and lang.txt is justified.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Diff reproduction replaced by one strategy line per risky decision.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Migration is now stated as language-blind, prints what it moved, and names the mv remedy for madrugar.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-08-28T11:18:03-07:00"
      agent: claude
      findings:
        - id: PQ-8
          severity: Minor
          title: The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
          detail: |-
            Task 7 Step 3 and Task 10 Step 3 call milestone-close/close, but the issue's
            Plan holds a single non-Mx row. close.go:554 matches the Mx checkbox against
            the ISSUE body and only warns on a miss (close.go:560), and
            findMilestonesMissingVerdict (close.go:1717) reads that same section — so at
            the full close the "was M1 reviewed" guard finds zero milestones and passes
            vacuously. Add the two Mx rows to the issue's Plan before starting M1.
          family: declared-boundary-untracked
          round: 3
      blocked: false
content_hash: 064dceb1778d8c3e5c9e32ba176d97ce18a2842ab74c95f58166181d5e1c5b41
---

# Gate ledger — tools#23 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T11:00:44-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `derived-state-rebuild` /lang has no stated mechanism, and lang-as-constructor-param conflicts with the store graph openStore builds once
  openStore derives history, capture, deck and vocab from ONE *YAML
  (cmd/define/main.go:202-212), pinned by TestOpenStoreSharesOneHighlightSet
  (cmd/define/vocab_test.go:148). repl takes d deps by value (repl.go:181) and
  rebuilds commandCtx per dispatch (repl.go:322), and commandCtx is
  deliberately narrow (command.go:145-167). A mid-session /lang es must
  re-derive all four and re-Load the vocabulary; Task 4 Step 2 says only "a
  row in command.go's table". Name the mechanism (a setLang closure over the
  loop's deps, on the setTimes precedent at command.go:162-163) or say the
  store reads the setting through, and reconcile whichever with the plan's
  stated "read once at the boundary" architecture.
- **PQ-2** [Important] `overlapping-input-precedence` -lang overlaps the existing -locale flag and the plan never states the precedence
  main.go:272 already defines -locale ("us or gb", default "us"), threaded via
  speak (main.go:611) into AudioCandidates (audiourl.go:21). Task 5's own test
  asserts voice{Lang "es", Locale "es"} but nothing in the plan derives that
  Locale; with today's default the mode builds Locale "us" and requests
  madrugar_es_us_1.mp3, a form the plan never measured. State the M1 interim
  rule now, since the issue that owns locale policy is blocked on this one.
- **PQ-3** [Important] `irreplaceable-artifact-guard` MigrateFlatDeck's target language is ambiguous and its collision semantics are undefined
  The test calls MigrateFlatDeck(dir, DefaultLang) while Step 2 prose says it
  moves into "words/<lang>/" called at the boundary — if that is the ACTIVE
  language, one "define -lang es" first run files an entire English deck under
  words/es/. Separately, "skipping anything already in a subdirectory" does not
  define the case where words/mesa.yaml and words/en/mesa.yaml both exist; the
  plan's own assertion that the flat file must not survive then either loses
  data or reddens the test. Deck() already skips directories
  (store/yaml.go:103-115), so pinning both rules is cheap.
- **PQ-4** [Important] `external-seam-fake-incomplete` M2's dictionary seam signature and the Spanish fixture capture path are unnamed (ARCH-MOCK)
  Dictionary is Lookup(word string) (dict.go:14-16) and systemDictionary()
  takes no argument (dict_darwin.go); the plan never says how the language
  reaches the seam. loadFakeDictionary globs a FLAT testdata/entries/*.txt
  keyed by basename and hard-fails on an empty corpus (dict_fake_test.go:24-49),
  and testdata/capture.sh captures through DCSCopyTextDefinition(NULL, ...).
  "the fake models a SET of dictionaries" therefore requires a corpus layout
  change, a capture.sh that captures through a chosen DCSDictionaryRef, and
  real Spanish captures for mesa and sycophantic. Task 9 Step 1 is one line.
- **PQ-5** [Important] `single-source-consumer-gap` RuntimeFiles reaches only .gitignore though the plan claims RuntimeDirs parity, and its entry shadows a tracked fixture
  store/yaml.go:34-41 says a runtime name must reach .gitignore, the index
  guard and the history guard; the plan repeats that framing but writes only
  the .gitignore guard. The reason it cannot mirror the other two belongs in
  the plan: cmd/define/testdata/golden/user-model.md is tracked and reachable
  from HEAD, so pointing repo_guard_test.go:200-218 and :240-260 at
  RuntimeFiles by basename fails on contact. The un-anchored user-model.md
  line also matches that golden path — harmless while tracked, silently
  un-addable after a git rm.
- **PQ-6** [Minor] `plan-restates-diff` The plan reproduces roughly 250 lines of the diff instead of one strategy line per risky function
  Full ParseLang, NewYAML and wordsDir bodies, complete test functions with
  enumerated case tables, and a call-site file inventory. The strategy that
  matters is real but buried: traversal input for ParseLang, the three named
  mutations for chooseDictionary, idempotence for the migration. Compress to
  those lines; the code will state the rest better within the hour.
- **PQ-7** [Minor] `unbacked-existing-behavior-claim` "English, because that is what every existing deck holds" is false of this tree
  The live deck here is words/ligament.yaml and words/madrugar.yaml. The
  language-blind migration files madrugar — this issue's own headline Spanish
  example — into words/en/, where the Spanish mode cannot see it and where its
  audio request stays the _en_us_ form the issue calls a 404. Say the migration
  is language-blind by design and print what it moved.

## Round 2 — 2026-08-28T11:09:11-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — D1 names setLang + deckDeps, and catches replraw.go:79's captured voc that the finding did not.
- PQ-2 — addressed — D2 pins the interim locale rule and routes it into atlas/define.md for #27.
- PQ-3 — addressed — MigrateFlatDeck loses the lang param; collision and idempotence rules each get a test.
- PQ-4 — addressed — systemDictionary(lang) with Lookup unchanged; per-language corpus and the capture path are Task 9 Step 1.
- PQ-5 — addressed — RuntimeFiles reaches all three guards; the squatting golden is renamed first, and lang.txt is justified.
- PQ-6 — addressed — Diff reproduction replaced by one strategy line per risky decision.
- PQ-7 — addressed — Migration is now stated as language-blind, prints what it moved, and names the mv remedy for madrugar.

## Round 3 — 2026-08-28T11:18:03-07:00 (claude) — passed

### Raised

- **PQ-8** [Minor] `declared-boundary-untracked` The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
  Task 7 Step 3 and Task 10 Step 3 call milestone-close/close, but the issue's
  Plan holds a single non-Mx row. close.go:554 matches the Mx checkbox against
  the ISSUE body and only warns on a miss (close.go:560), and
  findMilestonesMissingVerdict (close.go:1717) reads that same section — so at
  the full close the "was M1 reviewed" guard finds zero milestones and passes
  vacuously. Add the two Mx rows to the issue's Plan before starting M1.

## Open findings

- **PQ-8** [Minor] `declared-boundary-untracked` The plan declares M1/M2 boundaries but the issue's Plan section has no Mx rows for the binary to tick or enumerate
