---
gate: plan-quality
issue: 29
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-29T06:54:49-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Task 8's doc sweep omits audiourl.go's "never a search across languages" comment, which this change falsifies
          detail: |-
            cmd/define/audiourl.go:24-26 states the same invariant as the atlas paragraph
            Task 8 rewrites, and Task 4 adds utterance.Candidates() — the cross-language
            walk — to that same file. Enumerate it in Task 8 and in the Step 5 grep, or
            the file ships asserting the opposite of what it does (ARCH-DRY).
          family: doc-sweep-incomplete
          round: 1
        - id: PQ-2
          severity: Important
          title: TestFrenchCoverageIsStillPartial does not match the -run CDN filter Task 9 verifies with
          detail: |-
            fetch_conformance_test.go:13 and Task 9 Step 2 both run `-tags conformance
            -run CDN`; the name contains no "CDN", so the row pinning why the fallback
            exists (hotel/debut 404 on fr_fr, 200 on en_us) is skipped and Step 2 passes
            vacuously. lessons.md:1812 records the same class already. Rename it, or
            state the filter it needs (ARCH-MOCK).
          family: conformance-row-never-runs
          round: 1
        - id: PQ-3
          severity: Minor
          title: AlsoSpellings reads only the first (also …) in a gloss; the plan's own naive fixture has two
          detail: |-
            CutPrefix + Cut(rest, ")") stops at the first alternative. Running ParseEntry
            on the Task 2 fixture gives Blocks[0].Senses[0].Gloss = "(also naïve) (also
            naïveness)" — one gloss, two alternatives. It passes only because the
            diacritic one is written first; the reverse order drops the source spelling
            and degrades to English. Scan every occurrence in the gloss.
          family: first-match-not-all-matches
          round: 1
        - id: PQ-4
          severity: Minor
          title: Headword-first ordering is right 5/8 on the plan's own sample; diacritic-first is right 8/8
          detail: |-
            The Log's table has the accent on the headword for jalapeno/pinata/senor/
            cliche/fiance and on the (also …) for cafe/naive/facade. Ordering by "the
            spelling carrying non-ASCII first" wins on all eight and saves the café class
            two 404s at the ~300-600 ms per miss atlas/define.md:1129-1131 prices.
          family: candidate-order-by-measurement
          round: 1
        - id: PQ-5
          severity: Minor
          title: The one-shot play site is inside defineOnce, shared with the piped loop, and the plan does not say how pron reaches it
          detail: |-
            Task 6 says the one-shot site passes utteranceFor(word, out.entry, pron, opt),
            but that site is main.go:672 inside defineOnce (main.go:660), also called from
            repl.go:332. Say whether defineOnce gains a parameter; a field on `options` is
            the easy path and recreates the session-scoped value D3 refuses.
          family: unstated-seam-threading
          round: 1
        - id: PQ-6
          severity: Minor
          title: The issue's third option — labelling which notation variant is which — is never disposed
          detail: |-
            The Spec called it "worth pricing" (the /əˈrändəsmənt, eˌrändēsˈmäN/ point).
            D2 implies the operator chose the action instead; record that as a non-goal
            with its reason rather than leaving it unanswered at close.
          family: unstated-non-goal
          round: 1
        - id: PQ-7
          severity: Minor
          title: 1131 lines, most of it the pre-written diff and four full test tables
          detail: |-
            The implementations and their doc comments will be rewritten within the hour
            and the case tables are a lossy pre-image of executable artifacts. One
            strategy line per risky function carries the same information — for
            differsOnlyByDiacritics, a predicate over arbitrary NOAD gloss text, that
            line is the fuzz/malformed-input class, not fourteen hand-picked pairs.
          family: plan-restates-the-diff
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-29T07:00:59-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 8 is now a five-site enumeration table including audiourl.go:24-26, with a per-deletion grep in Step 4; all five sites verified to exist at the cited lines.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: All new rows TestCDN-prefixed and the -run CDN cadence comment at fetch_conformance_test.go:13 is rewritten to the unfiltered whole-file form.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: AlsoSpellings scans every (also ...) in a gloss; Task 2 tests both orderings of the two-parenthetical naive gloss.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: SourceSpellings orders non-ASCII-carrying spellings first, measured 9/9 after role/rôle was added.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: pron rides on replCommand beside literal, verified present at repl.go:18-21 and received by defineOnce at main.go:660; the options field is explicitly refused.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Now D6, a stated non-goal with its reason, not a deferral.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: 1131 to 395 lines; contracts plus one test-class line per risky function, with the fuzz/malformed-input class named for differsOnlyByDiacritics.
          round: 2
      blocked: false
content_hash: 42bf650ab587da726952282ddb9c2946278967653110912e687baf347743ec3a
---

# Gate ledger — tools#29 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T06:54:49-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `doc-sweep-incomplete` Task 8's doc sweep omits audiourl.go's "never a search across languages" comment, which this change falsifies
  cmd/define/audiourl.go:24-26 states the same invariant as the atlas paragraph
  Task 8 rewrites, and Task 4 adds utterance.Candidates() — the cross-language
  walk — to that same file. Enumerate it in Task 8 and in the Step 5 grep, or
  the file ships asserting the opposite of what it does (ARCH-DRY).
- **PQ-2** [Important] `conformance-row-never-runs` TestFrenchCoverageIsStillPartial does not match the -run CDN filter Task 9 verifies with
  fetch_conformance_test.go:13 and Task 9 Step 2 both run `-tags conformance
  -run CDN`; the name contains no "CDN", so the row pinning why the fallback
  exists (hotel/debut 404 on fr_fr, 200 on en_us) is skipped and Step 2 passes
  vacuously. lessons.md:1812 records the same class already. Rename it, or
  state the filter it needs (ARCH-MOCK).
- **PQ-3** [Minor] `first-match-not-all-matches` AlsoSpellings reads only the first (also …) in a gloss; the plan's own naive fixture has two
  CutPrefix + Cut(rest, ")") stops at the first alternative. Running ParseEntry
  on the Task 2 fixture gives Blocks[0].Senses[0].Gloss = "(also naïve) (also
  naïveness)" — one gloss, two alternatives. It passes only because the
  diacritic one is written first; the reverse order drops the source spelling
  and degrades to English. Scan every occurrence in the gloss.
- **PQ-4** [Minor] `candidate-order-by-measurement` Headword-first ordering is right 5/8 on the plan's own sample; diacritic-first is right 8/8
  The Log's table has the accent on the headword for jalapeno/pinata/senor/
  cliche/fiance and on the (also …) for cafe/naive/facade. Ordering by "the
  spelling carrying non-ASCII first" wins on all eight and saves the café class
  two 404s at the ~300-600 ms per miss atlas/define.md:1129-1131 prices.
- **PQ-5** [Minor] `unstated-seam-threading` The one-shot play site is inside defineOnce, shared with the piped loop, and the plan does not say how pron reaches it
  Task 6 says the one-shot site passes utteranceFor(word, out.entry, pron, opt),
  but that site is main.go:672 inside defineOnce (main.go:660), also called from
  repl.go:332. Say whether defineOnce gains a parameter; a field on `options` is
  the easy path and recreates the session-scoped value D3 refuses.
- **PQ-6** [Minor] `unstated-non-goal` The issue's third option — labelling which notation variant is which — is never disposed
  The Spec called it "worth pricing" (the /əˈrändəsmənt, eˌrändēsˈmäN/ point).
  D2 implies the operator chose the action instead; record that as a non-goal
  with its reason rather than leaving it unanswered at close.
- **PQ-7** [Minor] `plan-restates-the-diff` 1131 lines, most of it the pre-written diff and four full test tables
  The implementations and their doc comments will be rewritten within the hour
  and the case tables are a lossy pre-image of executable artifacts. One
  strategy line per risky function carries the same information — for
  differsOnlyByDiacritics, a predicate over arbitrary NOAD gloss text, that
  line is the fuzz/malformed-input class, not fourteen hand-picked pairs.

## Round 2 — 2026-08-29T07:00:59-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 8 is now a five-site enumeration table including audiourl.go:24-26, with a per-deletion grep in Step 4; all five sites verified to exist at the cited lines.
- PQ-2 — addressed — All new rows TestCDN-prefixed and the -run CDN cadence comment at fetch_conformance_test.go:13 is rewritten to the unfiltered whole-file form.
- PQ-3 — addressed — AlsoSpellings scans every (also ...) in a gloss; Task 2 tests both orderings of the two-parenthetical naive gloss.
- PQ-4 — addressed — SourceSpellings orders non-ASCII-carrying spellings first, measured 9/9 after role/rôle was added.
- PQ-5 — addressed — pron rides on replCommand beside literal, verified present at repl.go:18-21 and received by defineOnce at main.go:660; the options field is explicitly refused.
- PQ-6 — addressed — Now D6, a stated non-goal with its reason, not a deferral.
- PQ-7 — addressed — 1131 to 395 lines; contracts plus one test-class line per risky function, with the fuzz/malformed-input class named for differsOnlyByDiacritics.

## Open findings

(none — every finding has been disposed)
