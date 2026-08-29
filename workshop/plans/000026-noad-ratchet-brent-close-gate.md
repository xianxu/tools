---
gate: boundary-review
issue: 26
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-28T18:33:24-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-08-28T18:54:15-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: causeUnclassified cannot fire for oracle-tripping input, so the totality assertion is dead and new shapes are absorbed
          detail: |-
            Both branches of classifyRawNotation (cmd/define/rawnotation_test.go:95-111) are catch-alls, so every
            oracle survivor lands in a named bucket by construction and live_property_test.go:142 can never fire.
            Probed empirically: the issue's own "| AmE brent, BrE brent |" shape, plus an invented unknown shape,
            both classify as prose-numeral. Make causeProseNumeral positive (a digit adjacent to the pipe) so the
            residue falls to causeUnclassified.
          family: attribution-not-mechanized
          round: 2
        - id: BR-2
          severity: Important
          title: atlas/define.md:129 says the two oracles "measure 0 over the live sample" — contradicted by the 26 added at :148
          detail: |-
            An undated standing claim about the exact property this boundary re-measured, left in place in the
            section the boundary rewrote, twenty lines above the marked count that falsifies it.
          family: unbacked-behavior-claim
          round: 2
        - id: BR-3
          severity: Important
          title: The DERIVED-or-DATED rule was applied to the named sites, not the class — four more remain in the two edited files
          detail: |-
            atlas/define.md:174 ("all 71,427 reachable entries", a third undated width against the dated 70,886 at :90),
            :175 ("~32 entries"), :178 ("up to ~600 entries") and cmd/define/parse.go:93 ("13.8% of entries") are all
            neither derived nor dated. atlas/define.md:103 reads "at full width it was 27" as a current value.
          family: doc-sweep-incomplete
          round: 2
        - id: BR-4
          severity: Important
          title: The atlas per-cause breakdown is hand-restated, not derived — a compensating swap keeps the whole suite green
          detail: |-
            Verified by mutation in a scratch copy: swapping causeProseNumeral 7 to 8 and causePhrasePronunciation
            8 to 7 leaves every test passing while atlas/define.md:149 and :161 become wrong. This falsifies the
            Log claim "Moving a cause count reddens the atlas doc-sync by name". Add per-cause marker spans and
            loop TestAtlasQuotesTheRawNotationCount over rawCauses.
          family: restatement-not-consumer
          round: 2
        - id: BR-5
          severity: Important
          title: The dormant AmE/BrE shape's resurrection condition is written nowhere the next person adding a dictionary will meet it
          detail: |-
            Done-when row 1 is ticked, but grep over cmd/, atlas/ and workshop/targets/ finds nothing. The curated
            map at cmd/define/dictselect.go:76 carries no warning, and its comment at :54 lists ODE as an installed
            English candidate without noting the cost. The disposition survives only in the issue, which archives
            to workshop/history/.
          family: record-not-at-point-of-use
          round: 2
        - id: BR-6
          severity: Important
          title: testdata/rawnotation has no capture path in capture.sh and no live conformance check (ARCH-MOCK)
          detail: |-
            capture.sh builds only entries/en and entries/es; TestFixturesMatchLiveDictionary iterates
            capturedLanguages over testdata/entries alone. rawnotation_test.go:157 tells the reader to "re-capture"
            with no mechanism, and a macOS upgrade would leave the corpus a silent fossil.
          family: fake-without-conformance
          round: 2
        - id: BR-7
          severity: Important
          title: The two-oracle disjunction is restated at four sites and strayStress's body is copy-pasted into the classifier (ARCH-DRY)
          detail: |-
            rawnotation_test.go:93-94 reproduces invariant_test.go:43-44 verbatim; the trip predicate appears in
            three spellings at live_property_test.go:92, rawnotation_test.go:156 and render_test.go:235/241.
            Extract strayStressAt(out) int and rawNotationNear(out) (string, bool).
          family: oracle-restated-not-shared
          round: 2
        - id: BR-8
          severity: Minor
          title: classifyRawNotation's word parameter is read at zero sites, and the Plan specified a one-argument signature
          family: dead-parameter
          round: 2
        - id: BR-9
          severity: Minor
          title: capture.sh:45 still claims "there is no public API to select one" while the same file selects by identifier
          family: doc-sweep-incomplete
          round: 2
        - id: BR-10
          severity: Minor
          title: TestClassifyRawNotationIsTotal asserts only "not literal-pipe" for the both-oracles case rather than the exact cause
          family: assertion-by-exclusion
          round: 2
        - id: BR-11
          severity: Minor
          title: headwordBlockBytes = 128 has no boundary test; only exemplars at bytes 27 and 2436 constrain it
          family: constant-not-pinned
          round: 2
        - id: BR-12
          severity: Minor
          title: Three of the four literal-pipe entries rest entirely on the "the symbol |" substring, checked only by the conformance run
          family: family-pinned-by-one-member
          round: 2
        - id: BR-13
          severity: Minor
          title: workshop/projects/define-learn.md:788 quotes the old failure text "lower knownRawNotationEntries"; the message now says knownRawByCause
          family: doc-sweep-incomplete
          round: 2
      blocked: true
---

# Gate ledger — tools#26 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T18:33:24-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-08-28T18:54:15-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `attribution-not-mechanized` causeUnclassified cannot fire for oracle-tripping input, so the totality assertion is dead and new shapes are absorbed
  Both branches of classifyRawNotation (cmd/define/rawnotation_test.go:95-111) are catch-alls, so every
  oracle survivor lands in a named bucket by construction and live_property_test.go:142 can never fire.
  Probed empirically: the issue's own "| AmE brent, BrE brent |" shape, plus an invented unknown shape,
  both classify as prose-numeral. Make causeProseNumeral positive (a digit adjacent to the pipe) so the
  residue falls to causeUnclassified.
- **BR-2** [Important] `unbacked-behavior-claim` atlas/define.md:129 says the two oracles "measure 0 over the live sample" — contradicted by the 26 added at :148
  An undated standing claim about the exact property this boundary re-measured, left in place in the
  section the boundary rewrote, twenty lines above the marked count that falsifies it.
- **BR-3** [Important] `doc-sweep-incomplete` The DERIVED-or-DATED rule was applied to the named sites, not the class — four more remain in the two edited files
  atlas/define.md:174 ("all 71,427 reachable entries", a third undated width against the dated 70,886 at :90),
  :175 ("~32 entries"), :178 ("up to ~600 entries") and cmd/define/parse.go:93 ("13.8% of entries") are all
  neither derived nor dated. atlas/define.md:103 reads "at full width it was 27" as a current value.
- **BR-4** [Important] `restatement-not-consumer` The atlas per-cause breakdown is hand-restated, not derived — a compensating swap keeps the whole suite green
  Verified by mutation in a scratch copy: swapping causeProseNumeral 7 to 8 and causePhrasePronunciation
  8 to 7 leaves every test passing while atlas/define.md:149 and :161 become wrong. This falsifies the
  Log claim "Moving a cause count reddens the atlas doc-sync by name". Add per-cause marker spans and
  loop TestAtlasQuotesTheRawNotationCount over rawCauses.
- **BR-5** [Important] `record-not-at-point-of-use` The dormant AmE/BrE shape's resurrection condition is written nowhere the next person adding a dictionary will meet it
  Done-when row 1 is ticked, but grep over cmd/, atlas/ and workshop/targets/ finds nothing. The curated
  map at cmd/define/dictselect.go:76 carries no warning, and its comment at :54 lists ODE as an installed
  English candidate without noting the cost. The disposition survives only in the issue, which archives
  to workshop/history/.
- **BR-6** [Important] `fake-without-conformance` testdata/rawnotation has no capture path in capture.sh and no live conformance check (ARCH-MOCK)
  capture.sh builds only entries/en and entries/es; TestFixturesMatchLiveDictionary iterates
  capturedLanguages over testdata/entries alone. rawnotation_test.go:157 tells the reader to "re-capture"
  with no mechanism, and a macOS upgrade would leave the corpus a silent fossil.
- **BR-7** [Important] `oracle-restated-not-shared` The two-oracle disjunction is restated at four sites and strayStress's body is copy-pasted into the classifier (ARCH-DRY)
  rawnotation_test.go:93-94 reproduces invariant_test.go:43-44 verbatim; the trip predicate appears in
  three spellings at live_property_test.go:92, rawnotation_test.go:156 and render_test.go:235/241.
  Extract strayStressAt(out) int and rawNotationNear(out) (string, bool).
- **BR-8** [Minor] `dead-parameter` classifyRawNotation's word parameter is read at zero sites, and the Plan specified a one-argument signature
- **BR-9** [Minor] `doc-sweep-incomplete` capture.sh:45 still claims "there is no public API to select one" while the same file selects by identifier
- **BR-10** [Minor] `assertion-by-exclusion` TestClassifyRawNotationIsTotal asserts only "not literal-pipe" for the both-oracles case rather than the exact cause
- **BR-11** [Minor] `constant-not-pinned` headwordBlockBytes = 128 has no boundary test; only exemplars at bytes 27 and 2436 constrain it
- **BR-12** [Minor] `family-pinned-by-one-member` Three of the four literal-pipe entries rest entirely on the "the symbol |" substring, checked only by the conformance run
- **BR-13** [Minor] `doc-sweep-incomplete` workshop/projects/define-learn.md:788 quotes the old failure text "lower knownRawNotationEntries"; the message now says knownRawByCause

## Open findings

- **BR-1** [Important] `attribution-not-mechanized` causeUnclassified cannot fire for oracle-tripping input, so the totality assertion is dead and new shapes are absorbed
- **BR-2** [Important] `unbacked-behavior-claim` atlas/define.md:129 says the two oracles "measure 0 over the live sample" — contradicted by the 26 added at :148
- **BR-3** [Important] `doc-sweep-incomplete` The DERIVED-or-DATED rule was applied to the named sites, not the class — four more remain in the two edited files
- **BR-4** [Important] `restatement-not-consumer` The atlas per-cause breakdown is hand-restated, not derived — a compensating swap keeps the whole suite green
- **BR-5** [Important] `record-not-at-point-of-use` The dormant AmE/BrE shape's resurrection condition is written nowhere the next person adding a dictionary will meet it
- **BR-6** [Important] `fake-without-conformance` testdata/rawnotation has no capture path in capture.sh and no live conformance check (ARCH-MOCK)
- **BR-7** [Important] `oracle-restated-not-shared` The two-oracle disjunction is restated at four sites and strayStress's body is copy-pasted into the classifier (ARCH-DRY)
- **BR-8** [Minor] `dead-parameter` classifyRawNotation's word parameter is read at zero sites, and the Plan specified a one-argument signature
- **BR-9** [Minor] `doc-sweep-incomplete` capture.sh:45 still claims "there is no public API to select one" while the same file selects by identifier
- **BR-10** [Minor] `assertion-by-exclusion` TestClassifyRawNotationIsTotal asserts only "not literal-pipe" for the both-oracles case rather than the exact cause
- **BR-11** [Minor] `constant-not-pinned` headwordBlockBytes = 128 has no boundary test; only exemplars at bytes 27 and 2436 constrain it
- **BR-12** [Minor] `family-pinned-by-one-member` Three of the four literal-pipe entries rest entirely on the "the symbol |" substring, checked only by the conformance run
- **BR-13** [Minor] `doc-sweep-incomplete` workshop/projects/define-learn.md:788 quotes the old failure text "lower knownRawNotationEntries"; the message now says knownRawByCause
