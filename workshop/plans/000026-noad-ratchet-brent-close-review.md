# Boundary Review — tools#26 (whole-issue close)

| field | value |
|-------|-------|
| issue | 26 — NOAD ratchet regressed 27 to 32: a British-English pronunciation block renders raw |
| repo | tools |
| issue file | workshop/issues/000026-noad-ratchet-brent.md |
| boundary | whole-issue close |
| milestone | — |
| window | 641f5aad97f8ed79de0fee3b84ec018b4fe3f3fa..ee4f7232594319af3115d99da6311fda9b0b98e1 |
| command | sdlc close --issue 26 |
| reviewer | claude |
| timestamp | 2026-08-28T18:33:24-07:00 |
| verdict | unknown |

## Review

Failed to authenticate. API Error: 401 OAuth access token has been revoked.

---

## Re-review — 2026-08-28T18:54:15-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 26 — NOAD ratchet regressed 27 to 32: a British-English pronunciation block renders raw |
| repo | tools |
| issue file | workshop/issues/000026-noad-ratchet-brent.md |
| boundary | whole-issue close |
| milestone | — |
| window | 641f5aad97f8ed79de0fee3b84ec018b4fe3f3fa..ee4f7232594319af3115d99da6311fda9b0b98e1 |
| command | sdlc close --issue 26 |
| reviewer | claude |
| timestamp | 2026-08-28T18:54:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core deliverable is real and well-built: `knownRawByCause` is a genuine per-cause ratchet with the total *derived* from it, `classifyRawNotation` is pure and untagged so the taxonomy is pinned off-conformance by four real captured exemplars, and the atlas total is now a machine-checked consumer. I verified the whole `./cmd/define` suite green (94s), and I confirmed the Log's second mutation claim by reverting it (disabling the stress-mark branch reddens `hundred`, `shape` **and** the precedence assertion). What blocks a clean SHIP is that the two Done-when rows the issue calls its own point — "the rule, not the six sites" and "written where the next person adding one will meet it" — are ticked `[x]` while the sweep stopped at the sites prior findings named: `atlas/define.md:129` still asserts the oracles "measure **0**" twenty lines above the `26` this commit added, `atlas/define.md:174` still carries an undated 71,427 against the dated 70,886, the resurrection condition for the `| AmE …, BrE … |` shape exists nowhere outside the issue file (which archives to `history/`), and `causeUnclassified` — the mechanism the Plan names as "the failure mode this issue exists to end" — cannot fire for any input that trips the current oracle. I verified that last one empirically: the issue's own headline `brent` shape classifies as `prose-numeral`.

## 1. Strengths

- **`knownRawNotationEntries` derived from `knownRawByCause`** (`cmd/define/rawnotation_test.go:44`) is the right shape — one producer, two consumers, and the two structurally cannot disagree. The untagged-file move (`rawnotation_test.go` vs `live_property_test.go`'s `darwin && conformance`) is what makes the atlas able to consume it at all; that was the real insight of the boundary.
- **The precedence decision is stated *and* pinned.** `TestClassifyRawNotationIsTotal:177` genuinely fails when the stress branch is disabled — I mutated it and it went red with `"literal-pipe"`. That is not a test written to agree with the implementation.
- **The exemplar vacuity guard** (`rawnotation_test.go:156`) is the detail most fixtures skip: it fails loudly if a captured entry stops tripping the oracle rather than passing green against a fossil.
- **`testdata/rawnotation/` deliberately outside `testdata/entries/<lang>/`** (`rawnotation_test.go:120-129`), with the reason written down — `capturedLanguages` walks that tree and `TestNoRawPronunciationNotationSurvives` asserts a hard zero over it. Two corpora, two contracts, correctly separated.
- **`atlas/define.md:90` is a model DATED measurement** — width, non-Latin count and runtime, all in one dated sentence, with the reason the distinction matters stated inline.

## 2. Critical findings

None.

## 3. Important findings

**(a) `causeUnclassified` is unreachable for any input that trips the oracle — `live_property_test.go:142`'s totality assertion cannot fire.**
Both branches of `classifyRawNotation` are catch-alls: a stray stress mark falls to `causePhrasePronunciation`, and any pipe that isn't `"the symbol |"` falls to `causeProseNumeral` (`rawnotation_test.go:105-111`). Since the oracle is exactly `strayStress != "" || contains '|'`, every survivor lands in a named bucket by construction. I probed the issue's own headline shape:

```
"brent\n\n    | AmE brɛnt, BrE brɛnt | noun (British English)…" -> prose-numeral
"frobnicate ⟨totally new shape⟩ | xx | verb"                    -> prose-numeral
```

So when a British dictionary enters `curated["en"]` — the resurrection the Spec documents — the run reports `prose-numeral: 11, pinned at 7` and sends the reader at the sense-numeral parser defect. That is "absorbed into whichever existing cause it happens to resemble", which the Plan names as the failure mode the issue exists to end.
*Fix:* make `causeProseNumeral` positive rather than residual (require a digit adjacent to the pipe, matching the documented cause) and let the residue fall to `causeUnclassified`. Add a unit case asserting an unknown pipe shape classifies unclassified.

**(b) `atlas/define.md:128-129` states "Both now measure **0** over the live sample" — false, and contradicted by line 148 of the same file.**
Twenty lines below, the same file records `<!-- raw-notation-count -->26<!-- /raw-notation-count -->`. This is an undated standing measured claim about the exact property this boundary re-measured, left untouched in the section it rewrote.
*Fix:* replace with the derived/dated form — content loss is 0%; the raw-notation oracles trip on the pinned 26, broken down below.

**(c) The DERIVED-or-DATED rule was applied to the sites prior findings named, not to the class — four more remain in the two files this boundary edited.** `atlas/define.md:174` ("measured over all 71,427 reachable entries" — a third, undated width against the dated 70,886 at :90), `:175` ("~32 entries"), `:178` ("up to ~600 entries"), and `cmd/define/parse.go:93` ("13.8% of entries"). `atlas/define.md:103` ("at full width it was 27") reads as a current value in a historical sentence. ARCH-PURPOSE: the finding named a class and the fix swept the instances.
*Fix:* date each (`measured YYYY-MM-DD: N`) or delete the number as was correctly done for "the 29 captured fixtures" at `:80`.

**(d) The atlas's per-cause breakdown is a hand-maintained restatement, not a consumer — verified by mutation.** In a scratch copy I swapped `causeProseNumeral: 7→8` and `causePhrasePronunciation: 8→7`. Total stays 26, **the entire suite stays green**, and `atlas/define.md:149` ("*prose numeral* 7 … *phrase pronunciation* 8") plus `:161` ("7 of those entries") are both now wrong. This falsifies the `## Log` claim *"Moving a cause count reddens the atlas doc-sync by name"* — it reddens on the total only, and not at all for a compensating move.
*Fix:* per-cause marker spans (`<!-- raw-cause:prose-numeral -->7<!-- /… -->`) and a loop over `rawCauses` in `TestAtlasQuotesTheRawNotationCount`, so the doc-sync test names the cause the way the ratchet does.

**(e) Done-when row 1's "written where the next person adding one will meet it" is not delivered.** `grep -rn "AmE\|BrE\|dormant\|resurrect"` over `cmd/`, `atlas/` and `workshop/targets/` returns nothing about the dual-locale block. The `curated` map at `cmd/define/dictselect.go:76` — the exact line someone edits to add a dictionary — carries no warning, and its own comment at `:54` *lists ODE as an installed English candidate* without noting what adding it costs. The disposition lives only in the issue file, which archives to `workshop/history/` (AGENTS.md §2: "Don't read `workshop/history/*` unless asked").
*Fix:* three lines above `var curated` naming the shape, `#26`, and the ratchet it will move; optionally an `atlas/define.md` Limits bullet.

**(f) ARCH-MOCK: the new fixture corpus has no capture path and no live conformance check.** `testdata/capture.sh` builds only `entries/en` and `entries/es`; `TestFixturesMatchLiveDictionary` (`dict_conformance_test.go:32`) iterates `capturedLanguages(t)` over `testdata/entries` alone. So after a macOS upgrade, `capture.sh` regenerates one corpus and silently leaves `rawnotation/` a fossil — while `rawnotation_test.go:157` instructs the reader to "re-capture" with no mechanism to do so. The Plan's "Captured through the curated identifiers like every other fixture" is true of the bytes but not of the reproducible path.
*Fix:* add a `rawnotation_words=(charge hundred shape pipe)` loop to `capture.sh` through `EN_DICTS`, and extend the conformance test to byte-compare that directory too.

**(g) ARCH-DRY: the two-oracle disjunction is restated at four sites, and `strayStress`'s body is copy-pasted into the classifier.** `rawnotation_test.go:93-94` reproduces `invariant_test.go:43-44` verbatim (`slashSpan.ReplaceAllString` + `IndexAny(rest, "ˈˌ")`) — the comment even admits the coupling. The "did it trip?" predicate appears at `live_property_test.go:92` (`IndexByte >= 0`), `rawnotation_test.go:156` (`ContainsRune`), and `render_test.go:235`+`:241`, in three different spellings.
*Fix:* one `strayStressAt(out) int` that `strayStress` and `classifyRawNotation` both call, and one `rawNotationNear(out) (string, bool)` the four assertion sites consume. Widening the oracle should then be a one-line change, which is precisely the property (a) needs to stay honest.

## 4. Minor findings

- `classifyRawNotation(word, rendered string)` — `word` is read at zero sites; the Plan specified `classifyRawNotation(rendered string) rawCause`. Drop it (or use it), and align the plan row.
- `cmd/define/testdata/capture.sh:45` still says "there is no public API to select one" while lines 88-90 of the same file select by identifier — same false claim Done-when row 6 corrected in `live_property_test.go:69`.
- `TestClassifyRawNotationIsTotal:177` asserts only `got != causeLiteralPipe`; asserting `== causeHeadwordPronunciation` would pin the both-oracles case exactly rather than by exclusion.
- `headwordBlockBytes = 128` has no boundary test — only two exemplars at bytes 27 and 2436 constrain it; a 127/128/129 table case would pin the constant itself.
- Only `pipe` pins `causeLiteralPipe`; `piped`/`pipeful`/`pipeless` (3 of the 4 pinned) rest entirely on the `"the symbol |"` substring holding, checked only by the conformance run.
- `workshop/projects/define-learn.md:788` quotes the old failure text `"lower knownRawNotationEntries"`; the message now says `knownRawByCause`.

## 5. Test coverage notes

Coverage of the *classifier* is good and non-vacuous — I mutation-checked both directions and the exemplar test earns its keep. The gaps are at the edges of the taxonomy rather than its centre: nothing off-conformance pins the residual buckets (a), the threshold constant, or three of the four `literal-pipe` members. The doc-sync test is narrow by design and correctly so, but it currently pins one number where the atlas states five (d). The `go vet` + full-suite run is clean; the conformance sweep was not re-run in this review (host-dependent, ~116s, and the Log records its output).

## 6. Architectural notes

- **ARCH-DRY — flag**, finding (g). The oracle is one fact restated four ways; the issue's entire doc thesis ("one producer, N consumers") applies verbatim to it and wasn't turned inward.
- **ARCH-PURE — pass.** `classifyRawNotation` is a pure `string → rawCause` with no IO; the live sweep is the thin shell; `slashSpan` is a package-level compiled regexp. The exemplar test's only IO is a read-only `os.ReadFile` of committed fixtures, which is the repo's standing fixture idiom, not a mock.
- **ARCH-PURPOSE — flag**, findings (a), (c), (e). Three separate places where the site a prior finding named got fixed and the enumerable class around it did not. `doc-sweep-incomplete` is now on its third instance across this issue's gates; per the principle, that is the ledger reporting the enumeration was never written. The cheap enumeration exists: `grep -rnE '[0-9]{2,}(,[0-9]{3})?\s*(entries|%)' atlas/define.md cmd/define/*.go`.
- **ARCH-MOCK — flag**, finding (f). Production and test do share the `ParseEntry`/`Render` seam, which is the important half; what's missing is the fake's *provenance* — regeneration and drift detection.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/issues/000026-noad-ratchet-brent.md` covering:

1. **Done-when row 1** — untick or scope honestly: the resurrection condition is recorded in the issue and the gate ledger, not at `cmd/define/dictselect.go:76` where a reader adding a dictionary meets it.
2. **Done-when row 5** — untick or record the residue: the DERIVED-or-DATED rule is stated but four sites in the two edited files (`atlas/define.md:174/175/178`, `cmd/define/parse.go:93`) and `atlas/define.md:129` remain neither derived nor dated.
3. **Plan row 2** — the shipped signature is `classifyRawNotation(word, rendered string)`, not the `classifyRawNotation(rendered string)` the row specifies, and `word` is unused.
4. **Plan row 2's totality clause** — correct the claim. `causeUnclassified` is reachable only if the *oracle* widens, not if a new dictionary shape arrives; state that, or change the classifier so the claim becomes true.
5. **`## Log` 2026-08-28** — correct *"Moving a cause count reddens the atlas doc-sync by name"*. A compensating swap leaves the suite green; a non-compensating one reddens on the total and does not name the cause.

```findings
findings:
  - id: new
    severity: Important
    family: attribution-not-mechanized
    title: |
      causeUnclassified cannot fire for oracle-tripping input, so the totality assertion is dead and new shapes are absorbed
    detail: |
      Both branches of classifyRawNotation (cmd/define/rawnotation_test.go:95-111) are catch-alls, so every
      oracle survivor lands in a named bucket by construction and live_property_test.go:142 can never fire.
      Probed empirically: the issue's own "| AmE brent, BrE brent |" shape, plus an invented unknown shape,
      both classify as prose-numeral. Make causeProseNumeral positive (a digit adjacent to the pipe) so the
      residue falls to causeUnclassified.
  - id: new
    severity: Important
    family: unbacked-behavior-claim
    title: |
      atlas/define.md:129 says the two oracles "measure 0 over the live sample" — contradicted by the 26 added at :148
    detail: |
      An undated standing claim about the exact property this boundary re-measured, left in place in the
      section the boundary rewrote, twenty lines above the marked count that falsifies it.
  - id: new
    severity: Important
    family: doc-sweep-incomplete
    title: |
      The DERIVED-or-DATED rule was applied to the named sites, not the class — four more remain in the two edited files
    detail: |
      atlas/define.md:174 ("all 71,427 reachable entries", a third undated width against the dated 70,886 at :90),
      :175 ("~32 entries"), :178 ("up to ~600 entries") and cmd/define/parse.go:93 ("13.8% of entries") are all
      neither derived nor dated. atlas/define.md:103 reads "at full width it was 27" as a current value.
  - id: new
    severity: Important
    family: restatement-not-consumer
    title: |
      The atlas per-cause breakdown is hand-restated, not derived — a compensating swap keeps the whole suite green
    detail: |
      Verified by mutation in a scratch copy: swapping causeProseNumeral 7 to 8 and causePhrasePronunciation
      8 to 7 leaves every test passing while atlas/define.md:149 and :161 become wrong. This falsifies the
      Log claim "Moving a cause count reddens the atlas doc-sync by name". Add per-cause marker spans and
      loop TestAtlasQuotesTheRawNotationCount over rawCauses.
  - id: new
    severity: Important
    family: record-not-at-point-of-use
    title: |
      The dormant AmE/BrE shape's resurrection condition is written nowhere the next person adding a dictionary will meet it
    detail: |
      Done-when row 1 is ticked, but grep over cmd/, atlas/ and workshop/targets/ finds nothing. The curated
      map at cmd/define/dictselect.go:76 carries no warning, and its comment at :54 lists ODE as an installed
      English candidate without noting the cost. The disposition survives only in the issue, which archives
      to workshop/history/.
  - id: new
    severity: Important
    family: fake-without-conformance
    title: |
      testdata/rawnotation has no capture path in capture.sh and no live conformance check (ARCH-MOCK)
    detail: |
      capture.sh builds only entries/en and entries/es; TestFixturesMatchLiveDictionary iterates
      capturedLanguages over testdata/entries alone. rawnotation_test.go:157 tells the reader to "re-capture"
      with no mechanism, and a macOS upgrade would leave the corpus a silent fossil.
  - id: new
    severity: Important
    family: oracle-restated-not-shared
    title: |
      The two-oracle disjunction is restated at four sites and strayStress's body is copy-pasted into the classifier (ARCH-DRY)
    detail: |
      rawnotation_test.go:93-94 reproduces invariant_test.go:43-44 verbatim; the trip predicate appears in
      three spellings at live_property_test.go:92, rawnotation_test.go:156 and render_test.go:235/241.
      Extract strayStressAt(out) int and rawNotationNear(out) (string, bool).
  - id: new
    severity: Minor
    family: dead-parameter
    title: |
      classifyRawNotation's word parameter is read at zero sites, and the Plan specified a one-argument signature
  - id: new
    severity: Minor
    family: doc-sweep-incomplete
    title: |
      capture.sh:45 still claims "there is no public API to select one" while the same file selects by identifier
  - id: new
    severity: Minor
    family: assertion-by-exclusion
    title: |
      TestClassifyRawNotationIsTotal asserts only "not literal-pipe" for the both-oracles case rather than the exact cause
  - id: new
    severity: Minor
    family: constant-not-pinned
    title: |
      headwordBlockBytes = 128 has no boundary test; only exemplars at bytes 27 and 2436 constrain it
  - id: new
    severity: Minor
    family: family-pinned-by-one-member
    title: |
      Three of the four literal-pipe entries rest entirely on the "the symbol |" substring, checked only by the conformance run
  - id: new
    severity: Minor
    family: doc-sweep-incomplete
    title: |
      workshop/projects/define-learn.md:788 quotes the old failure text "lower knownRawNotationEntries"; the message now says knownRawByCause
```
