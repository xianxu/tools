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

---

## Re-review — 2026-08-28T20:44:19-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 26 — NOAD ratchet regressed 27 to 32: a British-English pronunciation block renders raw |
| repo | tools |
| issue file | workshop/issues/000026-noad-ratchet-brent.md |
| boundary | whole-issue close |
| milestone | — |
| window | 641f5aad97f8ed79de0fee3b84ec018b4fe3f3fa..33391dfdada0f2e3731b1324cc599c6e8dcc5dd9 |
| command | sdlc close --issue 26 |
| reviewer | claude |
| timestamp | 2026-08-28T20:44:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's headline finding (BR-1) is genuinely fixed and I proved it by reverting: restoring the two catch-all branches in a scratch copy reddens `TestUnclassifiedIsReachableForOracleTrippingInput` on all three cases. BR-4's mechanism is real too — I re-ran the exact compensating swap (prose-numeral 7→8, phrase 8→7) and the doc-sync now fails *by cause name*, twice. BR-2/BR-3/BR-5/BR-6 all landed, and I re-ran BR-3's enumeration over both edited files and found no residue. What stops a clean SHIP is that the *class* BR-1 named was swept at three of four branches: `case i < headwordBlockBytes` is still a position-only catch-all, so any stray stress mark in the first 128 bytes is absorbed into `causeHeadwordPronunciation`. I probed it — the realistic form of this issue's own dormant shape, `| AmE ˈhəndrəd, BrE ˈhʌndrəd |` on a polysyllable, classifies as `headword-pronunciation`, not `unclassified`. The captured `brent` escapes only because it is a monosyllable with no stress mark. The new warning comment at `dictselect.go:75` asserts the opposite ("the live ratchet reports the new entries as `unclassified` — the designed outcome"), so the artifact written to warn the next person is now itself the unbacked claim. Two prior Importants also remain partly open (BR-4's second named site, BR-7's third named site) and five of six Minors were not touched.

## 1. Strengths

- **BR-1's fix is real and non-vacuously pinned.** `cmd/define/rawnotation_test.go:92-124` — every pipe branch is now a positive signature (`proseNumeralPipe` regex, `"the symbol |"`), and reverting the residue to catch-alls reddens all three cases of `TestUnclassifiedIsReachableForOracleTrippingInput` with the exact absorption messages. This is a test that fails without the fix.
- **BR-4's per-cause doc-sync earns its keep** (`cmd/define/doc_sync_test.go:80-92` + `atlas/define.md:152-156`). I ran the reviewer's own mutation: the suite goes red naming `prose-numeral` and `phrase-pronunciation` individually. The Log's falsified claim from the previous round is explicitly retracted in the new Log entry rather than quietly dropped.
- **The prose-numeral positive test keys on the defect's mechanism, not a heuristic.** `proseNumeralPipe = ^\s*\d+\.[^|\n]*\|` matches `"    2. euros for the postcard |"` in `charge` — the line *starts* with the misparsed numeral, which is exactly what the parser bug produces. That is a signature, not a proximity guess.
- **The byte figures in the classifier comment are accurate in the right coordinate space.** I checked: `hundred` is stress at 27 of 1828 *stripped* bytes (1967 raw), `shape` at 2436 of 4116 (4438 raw). The comment quotes the stripped lengths, which is the space `strayStressAt` indexes — a detail that was wrong in an earlier draft and is now right.
- **`strayStressAt` / `rawNotationNear` are the right extraction** (`cmd/define/invariant_test.go:64-85`), and the comment records *why* the stripped coordinate space is load-bearing rather than incidental. `go vet -tags conformance ./cmd/define/` is clean, so the conformance-tagged additions actually compile.

## 2. Critical findings

None. Everything in this window is test-side or documentation; no production code path changed behavior.

## 3. Important findings

**(a) `case i < headwordBlockBytes` is still a catch-all — the class BR-1 named was swept at three branches of four.** `cmd/define/rawnotation_test.go:96-97`. The predicate tests *position*, not shape, so any stray stress mark in the first 128 bytes is claimed by `causeHeadwordPronunciation`. Probed in a scratch copy:

```
brent, as captured (monosyllable, no stress mark)              stressAt=-1    -> unclassified
the same shape on a POLYSYLLABLE (ODE writes a stress mark)    stressAt=19    -> headword-pronunciation
dual-locale on 'laboratory' (the canonical AmE/BrE split)      stressAt=22    -> headword-pronunciation
a brand-new leak 100 bytes in, no slash                        stressAt=44    -> headword-pronunciation
the same leak 200 bytes in, no slash                           stressAt=213   -> unclassified
```

So `TestUnclassifiedIsReachableForOracleTrippingInput`'s third case passes on the one variant of #26's shape that carries no stress mark; the polysyllabic form — which is most of a British dictionary — is absorbed. And `cmd/define/dictselect.go:75` now tells the next person adding a book "What WILL happen is that the live ratchet reports the new entries as `unclassified`", which is false for that majority. *Fix:* give the headword branch a shape signature the way the other three have one (position **and** the glued-gloss tell — a `(` … `/` run, as in `(aˈhəndrədzˈhəndrəd/)`), let the rest fall to the residue, add the polysyllabic dual-locale case to the reachability table, and correct the `dictselect.go` claim to match whatever the branch then does.

**(b) BR-7 is not fully addressed — the third site it named is unchanged.** `cmd/define/rawnotation_test.go:180` still spells the disjunction by hand as `strayStress(out) == "" && !strings.ContainsRune(out, '|')` rather than calling `rawNotationNear`, which is the site the finding listed as "`rawnotation_test.go:156` (`ContainsRune`)". Separately, `stressWindow` (`:135-139`) re-implements `strayStress`'s window with a different radius (60 vs 50) over the same stripped text — a fourth copy of the same three lines. *Fix:* `if _, tripped := rawNotationNear(out); !tripped { t.Fatalf(...) }`, and have `strayStress` call `stressWindow`.

**(c) BR-4 is not fully addressed — the second site it named is still a hand restatement.** `atlas/define.md:170` reads "7 of those entries" for `causeProseNumeral`, outside any marker span. Because `TestAtlasQuotesTheRawNotationCount` uses `strings.Contains`, a second unmarked copy of a pinned number is invisible: after the mutation forces `:153` to 8, `:170` stays at 7 and stays green. *Fix (state the rule, not the site):* a number the code owns appears in the doc **only** inside its marker span — so wrap `:170`'s 7 in a second `raw:prose-numeral` span. The enumeration that makes this checkable is cheap: for each cause, grep the atlas for the pinned value and confirm every occurrence is inside a span.

## 4. Minor findings

- BR-9 untouched: `cmd/define/testdata/capture.sh:44-45` still says "there is no public API to select one" while `:88` selects `EN_DICTS` by identifier — the exact claim Done-when row 6 corrected in `live_property_test.go`.
- BR-10 untouched: `rawnotation_test.go:233` still asserts `got == causeLiteralPipe` is false; the input actually yields `headword-pronunciation`, so `== causeHeadwordPronunciation` would pin it exactly.
- BR-11 untouched: `headwordBlockBytes = 128` has no boundary case. The two figures justifying it (27 of 1828, 2436 of 4116) are correct but hand-restated in a comment and asserted nowhere — both are derivable from committed fixtures in the *normal* suite, so one table test would pin the constant and derive the figures at once.
- BR-12 untouched, and it matters more now: the corpus pins 4 of 26 live members, and the move from catch-all to positive branches made sibling misclassification newly possible (`chargee`, `depth`, `shortish`, `piped`… are constrained only by the live sweep).
- BR-13 untouched: `workshop/projects/define-learn.md:788` quotes `"lower knownRawNotationEntries"` in the present tense; the message at `live_property_test.go:149` now says `knownRawByCause`.
- The `dictselect.go` warning block is inserted mid-paragraph, splitting "A short list, easy to extend…" from "A LIST per language…" — correct content, awkward seam.
- `rawNotationNear` → `strayStress` → `strayStressAt` runs `slashSpan.ReplaceAllString` twice, and the live sweep then runs it again in `classifyRawNotation`: 3–4 regex passes per entry where there was 1. On-demand test path only.

## 5. Test coverage notes

The offline suite is green (`go test ./cmd/define/`, 94s) and `go vet` is clean on both the untagged and `-tags conformance` builds. Both claimed fixes I could mutate held up: reverting BR-1's fix reddens the reachability test; re-running BR-4's compensating swap reddens the doc-sync by cause name. **The live sweep could not be reproduced in this environment** — `TestRenderLosesNothingOverLiveEntries` reports "only 0 live entries reachable (235976 missing)" and `go run ./cmd/define charge` returns "no known en dictionary is installed", so this process has no DictionaryServices access regardless of the harness sandbox. The Log's `7 / 7 / 8 / 4, unclassified 0` after the classifier rewrite is therefore unverified here, and it is the claim most at risk from the catch-all→positive change: the 22 unpinned siblings must each still match a positive signature or the run goes red on both the per-cause pin and the totality assertion. `TestRawNotationExemplarsMatchLiveDictionary` compiles and is correctly gated, but likewise cannot be exercised here.

## 6. Architectural notes

- **ARCH-DRY — flag**, finding (b). Three of four oracle spellings were consolidated; the fourth (`rawnotation_test.go:180`) and a new near-duplicate window helper remain. The extraction itself is well done and the comment on `strayStressAt` explains the coupling properly.
- **ARCH-PURE — pass.** `classifyRawNotation` is `string → rawCause` with no IO; `proseNumeralPipe` and `slashSpan` are package-level compiled regexps; the live sweep is the thin shell and the exemplar test's only IO is `os.ReadFile` over committed fixtures.
- **ARCH-PURPOSE — flag**, finding (a), and (b)/(c) as the smaller instances. The shadow-sweep on this round's own fixes: BR-1's enumeration is "the four cause branches", and three were made positive while the fourth was left as a position test. `attribution-not-mechanized` is now on its second instance at this gate (third counting `PQ-10` at plan-quality), which is the ledger reporting that the enumeration was applied rather than written down. `doc-sweep-incomplete`, by contrast, genuinely closed — I re-ran the enumeration over `atlas/define.md` and `cmd/define/*.go` and every remaining measured number is dated, derived, or past-tense narration.
- **ARCH-MOCK — pass with a note.** BR-6 is properly closed: `capture.sh` gained `RAW_WORDS` through the same `EN_DICTS`, and `TestRawNotationExemplarsMatchLiveDictionary` byte-compares through the `Lookup` seam and re-asserts the oracle trips. The remaining gap is coverage, not provenance — the fake models 4 of the 26 shapes it stands in for (BR-12).

## 7. Plan revision recommendations

The previous round's five recommendations were handled substantively (the code was changed so the claims became true, and the Log explicitly retracts the falsified mutation claim) rather than as `## Revisions` entries — that is the better outcome and I am not asking for them again. Two remain:

1. **Plan row 2's totality clause** — it still reads as though `causeUnclassified` is the residue of *all four* branches. State that `causeHeadwordPronunciation` is currently a position test, or make it a shape test so the clause is true.
2. **Done-when row 1** — the resurrection condition is now recorded at `cmd/define/dictselect.go:66-82`, which satisfies "where the next person will meet it", but the behavior it promises ("reports the new entries as `unclassified`") does not hold for stress-marked dual-locale blocks. Either correct the promise or fix (a); as written the row is ticked against a claim the code does not deliver.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by reverting in a scratch copy — restoring the catch-alls reddens all three reachability cases.
  - id: BR-2
    disposition: addressed
    note: |
      atlas/define.md:126-133 replaced; no standing "measure 0" claim remains.
  - id: BR-3
    disposition: addressed
    note: |
      Re-ran the enumeration over atlas/define.md and cmd/define/*.go — every residue is dated, derived or past-tense.
  - id: BR-4
    disposition: not-addressed
    note: |
      Mechanism verified fixed (the compensating swap now reddens by cause name), but the second site the finding named, atlas/define.md:170 "7 of those entries", is still an unmarked restatement that strings.Contains cannot see.
  - id: BR-5
    disposition: addressed
    note: |
      Comment now sits above var curated; but its behavioral promise is wrong — see the new finding.
  - id: BR-6
    disposition: addressed
    note: |
      capture.sh RAW_WORDS loop plus TestRawNotationExemplarsMatchLiveDictionary; compiles under -tags conformance, unrunnable on this host.
  - id: BR-7
    disposition: not-addressed
    note: |
      Three sites converted; rawnotation_test.go:180 still spells the disjunction by hand, and stressWindow is a fourth copy of strayStress's window with a different radius.
  - id: BR-8
    disposition: addressed
    note: |
      Signature is classifyRawNotation(rendered string) at all three call sites.
  - id: BR-9
    disposition: not-addressed
    note: |
      capture.sh:44-45 unchanged in this window.
  - id: BR-10
    disposition: not-addressed
    note: |
      rawnotation_test.go:233 still asserts by exclusion; the input actually yields headword-pronunciation.
  - id: BR-11
    disposition: not-addressed
    note: |
      No boundary case added; the two justifying figures are correct but asserted nowhere.
  - id: BR-12
    disposition: not-addressed
    note: |
      Still four exemplars, and the catch-all-to-positive change made sibling misclassification newly possible.
  - id: BR-13
    disposition: not-addressed
    note: |
      workshop/projects/define-learn.md:788 unchanged in this window.
findings:
  - id: new
    severity: Important
    family: attribution-not-mechanized
    title: |
      The headword branch is still a position-only catch-all, so the stress-marked form of #26's own dormant shape is absorbed
    detail: |
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
```

---

## Re-review — 2026-08-28T20:59:57-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 26 — NOAD ratchet regressed 27 to 32: a British-English pronunciation block renders raw |
| repo | tools |
| issue file | workshop/issues/000026-noad-ratchet-brent.md |
| boundary | whole-issue close |
| milestone | — |
| window | 641f5aad97f8ed79de0fee3b84ec018b4fe3f3fa..54ebc063af737368dfe45ec742d0926eb6c72353 |
| command | sdlc close --issue 26 |
| reviewer | claude |
| timestamp | 2026-08-28T20:59:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Important findings this round's commits claimed (BR-4's residue, BR-7's residue) and BR-14 are genuinely fixed and genuinely pinned — I verified each by reverting in a scratch worktree and watching the named test go red, which is the check the claimed-fixes protocol asks for. The offline suite is green (`go test ./cmd/define/` 94s) and `go vet -tags conformance` is clean. What stops SHIP is that BR-14's *rule* was applied to the branch BR-14 named and not to the class it enumerated: of the four cause predicates, two now key on tells that are **shared between causes** rather than distinguishing them, so novel input is still absorbed — I probed the verbatim `(aˈhəndrədzˈhəndrəd/)` headword shape at byte 303 landing in `phrase-pronunciation`, an invented leak carrying any leftover `/` landing in `phrase-pronunciation`, and an ODE-shaped dual-locale block inside an inflection paren landing in `headword-pronunciation` — which is the third consecutive round of `attribution-not-mechanized` and, at the last of those, still falsifies the promise at `dictselect.go:76`. Five Minors carried from round 2 remain untouched; none block. The live `7/7/8/4, unclassified 0` claim could not be reproduced here (`go run ./cmd/define charge` → "no known en dictionary is installed"), and it is the claim most exposed by narrowing the headword branch to one offline-pinned member.

## 1. Strengths

- **`stressIsParenthesised` (`cmd/define/invariant_test.go:65`) is the right shape of fix.** It scans outward in the *stripped* coordinate space with `\n` and `)` as hard stops, so it cannot be satisfied by a parenthesis belonging to another span — a window-match would have been. Reverting it to `case i < headwordBlockBytes` reddens both new reachability rows exactly as claimed.
- **The BR-4 residue fix is a real mechanism, not a marked site.** `doc_sync_test.go:95-100` counts `<!-- raw:X -->` opens against fully-valued spans, so a *stale* second occurrence fails. Mutation-verified: setting `atlas/define.md:170` to `8` reddens with "marks prose-numeral 2 time(s) but only 1 carry the pinned value 7".
- **The oracle consolidation landed properly (ARCH-DRY).** `rawNotationNear` is now the sole trip predicate at five call sites (`live_property_test.go:91`, `render_test.go:240`, `rawnotation_test.go:179,220`, `dict_conformance_test.go:250`), and `strayStressWindow` is the one windowing with radius as a parameter — the fourth copy BR-7 named is gone.
- **`knownRawNotationEntries` derived from `knownRawByCause` (`rawnotation_test.go:41-49`)** makes the total and the breakdown structurally incapable of disagreeing, which is the correct answer to "a total is a weak ratchet".
- **ARCH-MOCK is properly closed for provenance**: `capture.sh:89` captures the exemplars through the same `EN_DICTS` production selects, and `TestRawNotationExemplarsMatchLiveDictionary` byte-compares through the `Lookup` seam *and* re-asserts the oracle still trips, so a fixed-upstream shape fails loudly instead of fossilising.

## 2. Critical findings

None. Nothing in this window changes production behaviour — the diff is test files, comments and docs.

## 3. Important findings

**(a) `cmd/define/rawnotation_test.go:106,113` — two of the four cause predicates key on tells that are shared between causes, so the residue is still unreachable for a whole class of novel input.** This is the **3rd finding in family `attribution-not-mechanized`.** Do NOT fix the instance.

The rule BR-14 stated was *"every cause branch's predicate must be a positive signature of the SHAPE it names."* The missing half is **disjointness**: a signature that also matches a sibling cause's shape is not a signature, it is a filter, and whichever branch runs second becomes the catch-all. Measured prevalence after this round's fix: **2 of 4 branches** fail it.

- `phrase-pronunciation` (`:113`) tests `Contains(strayStressWindow(rendered,i,60), "/")`. The glued-headword shape carries that same trailing slash — `(aˈhəndrədzˈhəndrəd/)` — so the branch is separated from `headword-pronunciation` by **position alone**, which is what BR-14 rejected. Probed: `strings.Repeat("x",300) + " (aˈhəndrədzˈhəndrəd/) more"` → `phrase-pronunciation`, and `strings.Repeat("z",500) + " a brand new ˈshape/ nobody described"` → `phrase-pronunciation`. Any novel leak carrying a leftover slash is absorbed.
- `headword-pronunciation` (`:106`) tests parenthesisation, which NOAD also applies to inflection lists. Probed: `"hundred (plural hundreds | AmE ˈhəndrədz, BrE ˈhʌndrədz |) cardinal"` → `headword-pronunciation`. That is the shape #26 is named after, in the form ODE would actually write it, absorbed — so `cmd/define/dictselect.go:76`'s "What WILL happen is that the live ratchet reports the new entries as `unclassified`" is still not true for that variant, and Done-when row 1 is still ticked against it.

The enumeration to write is a **pairwise-disjointness table over the four tells** (paren-scan, leftover-slash, `^\s*\d+\.[^|\n]*\|`, `the symbol |`): for each captured exemplar assert it matches *exactly one* predicate independent of branch order, and for each tell add a residue row placing that tell in a context belonging to another cause. `TestUnclassifiedIsReachableForOracleTrippingInput` currently has no row carrying a stray slash, which is why the class was invisible.

## 4. Minor findings

- `cmd/define/testdata/capture.sh:89` (RAW_WORDS), `rawnotation_test.go:162-168` (the hand table) and `dict_conformance_test.go:224` (a `*.txt` glob) are three hand-maintained statements of the same corpus membership; a fifth exemplar added to the glob is silently uncovered by the offline classifier test. Have the offline test glob and require every fixture to declare a cause.
- `strings.IndexByte(…, '|')` is spelled in both `rawNotationNear` (`invariant_test.go:126`) and `classifyRawNotation` (`rawnotation_test.go:118`) — the last un-shared half-oracle, and cheap to make a `pipeAt(out) int`.
- The `dictselect.go` warning block is still spliced mid-paragraph, separating "A short list, easy to extend…" from "A LIST per language…".
- `atlas/define.md:130-134` still describes the oracle as `strayStress` plus "a bare `strings.IndexByte(out, '|')`"; the code now presents one named `rawNotationNear`.

## 5. Test coverage notes

- Both claimed fixes I could mutate held: reverting `&& stressIsParenthesised(rendered, i)` reddens `dual-locale, polysyllabic` and `dual-locale, longer word`; the marker mutation reddens the doc-sync by cause name. Neither is a test written to agree with its fix.
- **The live sweep is unverifiable in this environment.** `TestRawNotationExemplarsMatchLiveDictionary` skips with "no curated English dictionary is installed", and `go run ./cmd/define charge` reports the same, so DictionaryServices is unavailable to this process regardless of harness sandboxing. The Log's `7 / 7 / 8 / 4, unclassified 0` **after** the BR-14 narrowing is therefore taken on trust, and it is materially more exposed than it was last round: `headword-pronunciation` now demands a parenthesis, and exactly one of its seven live members (`hundred`) is pinned offline. If `million`, `thousand` or the `-fold` forms render their glued pronunciation unparenthesised they will silently move to `phrase-pronunciation` (a slash is in-window) and redden two assertions at once. Re-running the ratchet unsandboxed before the close is the cheap way to convert Plan row 6 from a claim into evidence.
- The offline corpus pins 4 of 26 live members; the other 22 are constrained only by the conformance run.

## 6. Architectural notes

- **ARCH-DRY — pass.** The three-spelling oracle and the fourth window copy are consolidated behind `rawNotationNear` / `strayStressAt` / `strayStressWindow`, and the comment at `invariant_test.go:97-109` records *why* the stripped coordinate space is load-bearing. Only the `IndexByte(…,'|')` half and the corpus-membership restatement remain (Minor above).
- **ARCH-PURE — pass.** `classifyRawNotation` is `string → rawCause`, `slashSpan`/`proseNumeralPipe` are package-level compiled regexps, and the only IO in the offline path is `os.ReadFile` over committed fixtures. The live sweep is the thin shell; the classifier is injected into it rather than embedded.
- **ARCH-PURPOSE — flag**, finding (a). The shadow-sweep on this round's own fix: BR-14 named its enumeration ("the four branches of classifyRawNotation") and the fix changed one of them. `attribution-not-mechanized` is now on its third instance at this gate, which is the ledger reporting that the enumeration was applied rather than written down — the disjointness table above is the form that would close it.
- **ARCH-MOCK — pass with a note.** Provenance is fully closed (capture path + byte-compare + oracle re-assertion through the same seam). The residual is coverage, not seam design: the fake models 4 of the 26 shapes it stands in for, and the catch-all→positive→shape-tell progression has made each unpinned sibling's classification depend on a signature verified against one member (BR-12).

## 7. Plan revision recommendations

1. **Plan row 2's totality clause** still reads as though `causeUnclassified` is the residue of all four branches. It is not: any novel shape carrying a leftover `/` within 60 bytes lands in `phrase-pronunciation`. Append a `## Revisions` entry stating that the branches are exhaustive but **not pairwise disjoint by shape**, and that the residue is reachable only for input matching none of the four tells.
2. **Done-when row 1** is ticked against the `dictselect.go:76` promise that a re-added British dictionary "reports the new entries as `unclassified`". True for the bare `| AmE …, BrE … |` block; false when it sits inside an inflection parenthesis. Either narrow the promise to the forms actually probed, or close finding (a) and leave the row as written.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Mutation-verified — setting the second marked occurrence at atlas/define.md:170 to 8 reddens the marker-count check by cause name.
  - id: BR-7
    disposition: addressed
    note: |
      rawNotationNear is the sole trip predicate at five call sites and strayStressWindow is the one windowing; the fourth copy is gone.
  - id: BR-9
    disposition: not-addressed
    note: |
      cmd/define/testdata/capture.sh:45 unchanged for a third round — still "there is no public API to select one" while :89 selects by identifier.
  - id: BR-10
    disposition: not-addressed
    note: |
      Still asserts by exclusion, and the input now yields unclassified rather than a stress cause, so the comment's "a pronunciation leak that happens to contain a pipe" overstates what is demonstrated.
  - id: BR-11
    disposition: not-addressed
    note: |
      No boundary case, and aggravated this round — the justifying figures (27 of 1828, 2436 of 4116) were removed from the classifier comment, so rawnotation_test.go:139 "Measured, not guessed — see classifyRawNotation" now points at a measurement that is nowhere in the code.
  - id: BR-12
    disposition: not-addressed
    note: |
      Still 4 exemplars for 26 members, and narrowing headword-pronunciation to a parenthesis tell makes the six unpinned siblings depend on a signature verified against hundred alone.
  - id: BR-13
    disposition: not-addressed
    note: |
      workshop/projects/define-learn.md:788 still quotes "lower knownRawNotationEntries"; live_property_test.go:149 says knownRawByCause.
  - id: BR-14
    disposition: addressed
    note: |
      Verified by revert in a scratch worktree — restoring `case i < headwordBlockBytes` reddens both dual-locale reachability rows.
findings:
  - id: new
    severity: Important
    family: attribution-not-mechanized
    title: |
      Two of the four cause predicates key on tells shared with a sibling cause, so novel input is still absorbed
    detail: |
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
  - id: new
    severity: Minor
    family: restatement-not-consumer
    title: |
      Exemplar-corpus membership is stated three times by hand, so a fifth fixture is silently untested offline
    detail: |
      capture.sh:89 RAW_WORDS, the hand table at rawnotation_test.go:162-168, and the glob at
      dict_conformance_test.go:224 are three independent statements of the same set. The rule the family already
      carries applies unchanged: the corpus directory is the source and the tests derive from it — have the offline
      classifier test glob testdata/rawnotation and require every fixture to declare a cause, so adding an exemplar
      cannot leave the offline pin behind.
```
