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
    - "n": 3
      timestamp: "2026-08-28T20:44:19-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Verified by reverting in a scratch copy — restoring the catch-alls reddens all three reachability cases.
          round: 3
        - id: BR-2
          disposition: addressed
          note: atlas/define.md:126-133 replaced; no standing "measure 0" claim remains.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Re-ran the enumeration over atlas/define.md and cmd/define/*.go — every residue is dated, derived or past-tense.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Mechanism verified fixed (the compensating swap now reddens by cause name), but the second site the finding named, atlas/define.md:170 "7 of those entries", is still an unmarked restatement that strings.Contains cannot see.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Comment now sits above var curated; but its behavioral promise is wrong — see the new finding.
          round: 3
        - id: BR-6
          disposition: addressed
          note: capture.sh RAW_WORDS loop plus TestRawNotationExemplarsMatchLiveDictionary; compiles under -tags conformance, unrunnable on this host.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: Three sites converted; rawnotation_test.go:180 still spells the disjunction by hand, and stressWindow is a fourth copy of strayStress's window with a different radius.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Signature is classifyRawNotation(rendered string) at all three call sites.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: capture.sh:44-45 unchanged in this window.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: rawnotation_test.go:233 still asserts by exclusion; the input actually yields headword-pronunciation.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: No boundary case added; the two justifying figures are correct but asserted nowhere.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: Still four exemplars, and the catch-all-to-positive change made sibling misclassification newly possible.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: workshop/projects/define-learn.md:788 unchanged in this window.
          round: 3
      findings:
        - id: BR-14
          severity: Important
          title: 'The headword branch is still a position-only catch-all, so the stress-marked form of #26''s own dormant shape is absorbed'
          detail: |-
            This is the 2nd finding in family attribution-not-mechanized. Do NOT fix the instance. The RULE: every
            cause branch's predicate must be a positive signature of the SHAPE it names; a predicate that tests
            position or context rather than shape is a catch-all and absorbs novel input. The enumeration is the four
            branches of classifyRawNotation, and measured prevalence is 1 of 4 still failing it —
            cmd/define/rawnotation_test.go:96 `case i < headwordBlockBytes` claims any stray stress mark in the first
            128 bytes. Probed: "hundred | AmE ˈhəndrəd, BrE ˈhʌndrəd |" -> headword-pronunciation,
            "laboratory | AmE ˈlabrəˌtôrē, BrE ləˈbɒrət(ə)ri |" -> headword-pronunciation, and an invented leak at
            byte 44 -> headword-pronunciation. The captured brent case reaches the residue only because it is a
            monosyllable with no stress mark. Consequently cmd/define/dictselect.go:75, the comment added for BR-5,
            promises "the live ratchet reports the new entries as unclassified" for a book whose entries mostly will
            not. Give the branch a shape tell (position AND the glued-gloss "(…/" run), extend the reachability table
            with the polysyllabic dual-locale case, and correct the dictselect.go promise.
          family: attribution-not-mechanized
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-28T20:59:57-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: Mutation-verified — setting the second marked occurrence at atlas/define.md:170 to 8 reddens the marker-count check by cause name.
          round: 4
        - id: BR-7
          disposition: addressed
          note: rawNotationNear is the sole trip predicate at five call sites and strayStressWindow is the one windowing; the fourth copy is gone.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: cmd/define/testdata/capture.sh:45 unchanged for a third round — still "there is no public API to select one" while :89 selects by identifier.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: Still asserts by exclusion, and the input now yields unclassified rather than a stress cause, so the comment's "a pronunciation leak that happens to contain a pipe" overstates what is demonstrated.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: No boundary case, and aggravated this round — the justifying figures (27 of 1828, 2436 of 4116) were removed from the classifier comment, so rawnotation_test.go:139 "Measured, not guessed — see classifyRawNotation" now points at a measurement that is nowhere in the code.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: Still 4 exemplars for 26 members, and narrowing headword-pronunciation to a parenthesis tell makes the six unpinned siblings depend on a signature verified against hundred alone.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: workshop/projects/define-learn.md:788 still quotes "lower knownRawNotationEntries"; live_property_test.go:149 says knownRawByCause.
          round: 4
        - id: BR-14
          disposition: addressed
          note: Verified by revert in a scratch worktree — restoring `case i < headwordBlockBytes` reddens both dual-locale reachability rows.
          round: 4
      findings:
        - id: BR-15
          severity: Important
          title: Two of the four cause predicates key on tells shared with a sibling cause, so novel input is still absorbed
          detail: |-
            This is the 3rd finding in family attribution-not-mechanized. Do NOT fix the instance. The rule BR-14 stated
            needs its missing half: a cause predicate must be a signature that is DISJOINT from every sibling's shape,
            not merely positive — a tell a sibling shape also carries makes whichever branch runs second the catch-all.
            Measured prevalence: 2 of 4 branches fail it. rawnotation_test.go:113 keys phrase-pronunciation on a leftover
            "/" within 60 bytes, which the glued-headword shape also carries, so the two causes are separated by POSITION
            alone — the thing BR-14 rejected. Probed: the verbatim "(a-stress-hundred/)" string at byte 303 classifies
            as phrase-pronunciation, and an invented leak carrying any leftover slash classifies as phrase-pronunciation
            rather than reaching the residue. rawnotation_test.go:106 keys headword-pronunciation on parenthesisation,
            which NOAD also applies to inflection lists: "hundred (plural hundreds | AmE ..., BrE ... |) cardinal"
            classifies as headword-pronunciation, so the dual-locale shape this issue is named after is still absorbed in
            the form ODE would write it, and dictselect.go:76 still promises "unclassified" for it. The enumeration to
            write is a pairwise-disjointness table over the four tells: assert each captured exemplar matches exactly one
            predicate independent of branch order, and add a residue row placing each tell in a context belonging to
            another cause. TestUnclassifiedIsReachableForOracleTrippingInput has no row carrying a stray slash, which is
            why this class was invisible.
          family: attribution-not-mechanized
          round: 4
        - id: BR-16
          severity: Minor
          title: Exemplar-corpus membership is stated three times by hand, so a fifth fixture is silently untested offline
          detail: |-
            capture.sh:89 RAW_WORDS, the hand table at rawnotation_test.go:162-168, and the glob at
            dict_conformance_test.go:224 are three independent statements of the same set. The rule the family already
            carries applies unchanged: the corpus directory is the source and the tests derive from it — have the offline
            classifier test glob testdata/rawnotation and require every fixture to declare a cause, so adding an exemplar
            cannot leave the offline pin behind.
          family: restatement-not-consumer
          round: 4
      blocked: false
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

## Round 3 — 2026-08-28T20:44:19-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by reverting in a scratch copy — restoring the catch-alls reddens all three reachability cases.
- BR-2 — addressed — atlas/define.md:126-133 replaced; no standing "measure 0" claim remains.
- BR-3 — addressed — Re-ran the enumeration over atlas/define.md and cmd/define/*.go — every residue is dated, derived or past-tense.
- BR-4 — not-addressed — Mechanism verified fixed (the compensating swap now reddens by cause name), but the second site the finding named, atlas/define.md:170 "7 of those entries", is still an unmarked restatement that strings.Contains cannot see.
- BR-5 — addressed — Comment now sits above var curated; but its behavioral promise is wrong — see the new finding.
- BR-6 — addressed — capture.sh RAW_WORDS loop plus TestRawNotationExemplarsMatchLiveDictionary; compiles under -tags conformance, unrunnable on this host.
- BR-7 — not-addressed — Three sites converted; rawnotation_test.go:180 still spells the disjunction by hand, and stressWindow is a fourth copy of strayStress's window with a different radius.
- BR-8 — addressed — Signature is classifyRawNotation(rendered string) at all three call sites.
- BR-9 — not-addressed — capture.sh:44-45 unchanged in this window.
- BR-10 — not-addressed — rawnotation_test.go:233 still asserts by exclusion; the input actually yields headword-pronunciation.
- BR-11 — not-addressed — No boundary case added; the two justifying figures are correct but asserted nowhere.
- BR-12 — not-addressed — Still four exemplars, and the catch-all-to-positive change made sibling misclassification newly possible.
- BR-13 — not-addressed — workshop/projects/define-learn.md:788 unchanged in this window.

### Raised

- **BR-14** [Important] `attribution-not-mechanized` The headword branch is still a position-only catch-all, so the stress-marked form of #26's own dormant shape is absorbed
  This is the 2nd finding in family attribution-not-mechanized. Do NOT fix the instance. The RULE: every
  cause branch's predicate must be a positive signature of the SHAPE it names; a predicate that tests
  position or context rather than shape is a catch-all and absorbs novel input. The enumeration is the four
  branches of classifyRawNotation, and measured prevalence is 1 of 4 still failing it —
  cmd/define/rawnotation_test.go:96 `case i < headwordBlockBytes` claims any stray stress mark in the first
  128 bytes. Probed: "hundred | AmE ˈhəndrəd, BrE ˈhʌndrəd |" -> headword-pronunciation,
  "laboratory | AmE ˈlabrəˌtôrē, BrE ləˈbɒrət(ə)ri |" -> headword-pronunciation, and an invented leak at
  byte 44 -> headword-pronunciation. The captured brent case reaches the residue only because it is a
  monosyllable with no stress mark. Consequently cmd/define/dictselect.go:75, the comment added for BR-5,
  promises "the live ratchet reports the new entries as unclassified" for a book whose entries mostly will
  not. Give the branch a shape tell (position AND the glued-gloss "(…/" run), extend the reachability table
  with the polysyllabic dual-locale case, and correct the dictselect.go promise.

## Round 4 — 2026-08-28T20:59:57-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — Mutation-verified — setting the second marked occurrence at atlas/define.md:170 to 8 reddens the marker-count check by cause name.
- BR-7 — addressed — rawNotationNear is the sole trip predicate at five call sites and strayStressWindow is the one windowing; the fourth copy is gone.
- BR-9 — not-addressed — cmd/define/testdata/capture.sh:45 unchanged for a third round — still "there is no public API to select one" while :89 selects by identifier.
- BR-10 — not-addressed — Still asserts by exclusion, and the input now yields unclassified rather than a stress cause, so the comment's "a pronunciation leak that happens to contain a pipe" overstates what is demonstrated.
- BR-11 — not-addressed — No boundary case, and aggravated this round — the justifying figures (27 of 1828, 2436 of 4116) were removed from the classifier comment, so rawnotation_test.go:139 "Measured, not guessed — see classifyRawNotation" now points at a measurement that is nowhere in the code.
- BR-12 — not-addressed — Still 4 exemplars for 26 members, and narrowing headword-pronunciation to a parenthesis tell makes the six unpinned siblings depend on a signature verified against hundred alone.
- BR-13 — not-addressed — workshop/projects/define-learn.md:788 still quotes "lower knownRawNotationEntries"; live_property_test.go:149 says knownRawByCause.
- BR-14 — addressed — Verified by revert in a scratch worktree — restoring `case i < headwordBlockBytes` reddens both dual-locale reachability rows.

### Raised

- **BR-15** [Important] `attribution-not-mechanized` Two of the four cause predicates key on tells shared with a sibling cause, so novel input is still absorbed
  This is the 3rd finding in family attribution-not-mechanized. Do NOT fix the instance. The rule BR-14 stated
  needs its missing half: a cause predicate must be a signature that is DISJOINT from every sibling's shape,
  not merely positive — a tell a sibling shape also carries makes whichever branch runs second the catch-all.
  Measured prevalence: 2 of 4 branches fail it. rawnotation_test.go:113 keys phrase-pronunciation on a leftover
  "/" within 60 bytes, which the glued-headword shape also carries, so the two causes are separated by POSITION
  alone — the thing BR-14 rejected. Probed: the verbatim "(a-stress-hundred/)" string at byte 303 classifies
  as phrase-pronunciation, and an invented leak carrying any leftover slash classifies as phrase-pronunciation
  rather than reaching the residue. rawnotation_test.go:106 keys headword-pronunciation on parenthesisation,
  which NOAD also applies to inflection lists: "hundred (plural hundreds | AmE ..., BrE ... |) cardinal"
  classifies as headword-pronunciation, so the dual-locale shape this issue is named after is still absorbed in
  the form ODE would write it, and dictselect.go:76 still promises "unclassified" for it. The enumeration to
  write is a pairwise-disjointness table over the four tells: assert each captured exemplar matches exactly one
  predicate independent of branch order, and add a residue row placing each tell in a context belonging to
  another cause. TestUnclassifiedIsReachableForOracleTrippingInput has no row carrying a stray slash, which is
  why this class was invisible.
- **BR-16** [Minor] `restatement-not-consumer` Exemplar-corpus membership is stated three times by hand, so a fifth fixture is silently untested offline
  capture.sh:89 RAW_WORDS, the hand table at rawnotation_test.go:162-168, and the glob at
  dict_conformance_test.go:224 are three independent statements of the same set. The rule the family already
  carries applies unchanged: the corpus directory is the source and the tests derive from it — have the offline
  classifier test glob testdata/rawnotation and require every fixture to declare a cause, so adding an exemplar
  cannot leave the offline pin behind.

## Open findings

- **BR-9** [Minor] `doc-sweep-incomplete` capture.sh:45 still claims "there is no public API to select one" while the same file selects by identifier
- **BR-10** [Minor] `assertion-by-exclusion` TestClassifyRawNotationIsTotal asserts only "not literal-pipe" for the both-oracles case rather than the exact cause
- **BR-11** [Minor] `constant-not-pinned` headwordBlockBytes = 128 has no boundary test; only exemplars at bytes 27 and 2436 constrain it
- **BR-12** [Minor] `family-pinned-by-one-member` Three of the four literal-pipe entries rest entirely on the "the symbol |" substring, checked only by the conformance run
- **BR-13** [Minor] `doc-sweep-incomplete` workshop/projects/define-learn.md:788 quotes the old failure text "lower knownRawNotationEntries"; the message now says knownRawByCause
- **BR-15** [Important] `attribution-not-mechanized` Two of the four cause predicates key on tells shared with a sibling cause, so novel input is still absorbed
- **BR-16** [Minor] `restatement-not-consumer` Exemplar-corpus membership is stated three times by hand, so a fifth fixture is silently untested offline
