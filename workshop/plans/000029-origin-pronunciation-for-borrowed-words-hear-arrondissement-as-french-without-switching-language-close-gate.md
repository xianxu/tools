---
gate: boundary-review
issue: 29
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-29T08:13:51-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: The issue's first Done-when — the session does not move — has no automated assertion, in the test named for it
          detail: |-
            cmd/define/main_test.go:320 TestPronFetchesTheSourceRecordingWithoutMovingTheSession
            asserts the first CDN URL, the walk length, the printed entry and empty stderr, and
            nothing about the deck, the dictionary or d.lang. The rig wires noopCapturer, so it
            could not have recorded one. Task 6 Step 1(c) promised exactly that assertion. NOAD
            is unreachable from this process context, so the plan's manual `ls words/` check is
            also not reproducible at this gate. Use countingCapturer (capture_test.go:73) or a
            store-backed capturer as askrun_test.go:34 does.
          family: done-when-unpinned
          round: 1
        - id: BR-2
          severity: Important
          title: The raw editor's /pron branch is untested, though it carries the design's only real hazard
          detail: |-
            cmd/define/replraw.go:223-251 is the record-in-cooked / perform-in-raw split that
            lessons.md's "render cooked, play raw" exists for. Task 7 named cmd/define/repl_test.go
            and it was not touched. The repo already pairs TestLineLoopDispatchesCommands with
            TestRawEditorDispatchesCommands for this reason, and editorRig/scriptKeys make it a
            ~20-line test; the cooked-block property is assertable by counting CDN requests inside
            the cooked callback. I verified by scratch test that the path works today, so this is
            coverage rather than correctness.
          family: two-loops-one-test
          round: 1
        - id: BR-3
          severity: Important
          title: atlas/define.md:1170 — the new H2 was inserted mid-section and re-parented four paragraphs and an H3
          detail: |-
            "## Source pronunciation (#29)" sits between the locale-help block and the paragraph
            explaining it, so the localeHelp derivation note (:1249), the -locale English-only
            history (:1254), the playN note (:1258), "A missing recording is not a failed lookup"
            (:1263) and the H3 "### The dictionary follows the language (#23 M2)" (:1266) now read
            as part of the source-pronunciation section. Move the new section to just before
            "## Conformance" (:1329).
          family: heading-reparents-prose
          round: 1
        - id: BR-4
          severity: Important
          title: The plan's Core-concepts table calls fakeCDN "unchanged — REUSED" after the diff changed it
          detail: |-
            workshop/plans/000029-origin-pronunciation-plan.md:80 claims fakeCDN was reused
            untouched; the diff changed its keying from r.URL.Path to r.URL.EscapedPath, which is
            load-bearing. fakeDictionary and rebasedSource were also modified and appear in no
            row. The issue's Log records all three honestly; the plan does not. Fix with a
            "## Revisions" entry, not code.
          family: plan-table-vs-tree
          round: 1
        - id: BR-5
          severity: Minor
          title: reportVoice prints undefaulted Lang fields while AudioCandidates defaults them
          detail: |-
            cmd/define/main.go:856 formats u.Source.Lang and u.Session.Lang raw, while
            AudioCandidates (audiourl.go:117) substitutes store.DefaultLang for an empty Lang. With
            a zero opt.voice the record reads "played the  one" — observed in a scratch run.
            Production always calls applyVoice, so it is latent, but this line is the record the
            design insists must be true.
          family: report-must-use-the-normalised-value
          round: 1
        - id: BR-6
          severity: Minor
          title: utterance.sourceCandidates and utterance.askedForSource write the same predicate twice, negated
          detail: |-
            audiourl.go:180 guards on `u.Source.Lang == "" || u.Source == u.Session`; audiourl.go:233
            returns its negation. The latter's comment says the two must agree — make it so by
            construction: sourceCandidates should call askedForSource() (ARCH-DRY).
          family: one-predicate-two-spellings
          round: 1
        - id: BR-7
          severity: Minor
          title: fakeDictionary's accent-insensitive fallback iterates a Go map
          detail: |-
            cmd/define/dict_fake_test.go:79 — with two entries that both differ from the query only
            by diacritics, the entry returned is nondeterministic. Iterate sorted keys.
          family: nondeterministic-fake
          round: 1
        - id: BR-8
          severity: Minor
          title: README's define cheat-sheet lists every other flag a reader types, but not -pron
          detail: |-
            README.md:29-41 shows --sound, -no-audio, -locale, -lang, -raw and -no-color; Task 8
            Step 3 named "README flag table". The prose section further down does document -pron,
            so this is completeness rather than a gap in the docs gate.
          family: doc-sweep-incomplete
          round: 1
        - id: BR-9
          severity: Minor
          title: The plan-guard's new-row exemption matches any status cell containing "new"
          detail: |-
            cmd/define/repo_guard_test.go:593 scans the whole document for "- [ ] " and then
            exempts a row whose status cell lower-cases to contain "new" — "renewed" or "newly"
            would exempt too. Low risk while gated on in-progress, but an anchored match is one
            character more.
          family: guard-heuristic-too-loose
          round: 1
        - id: BR-10
          severity: Minor
          title: The Spec's split-out dictionary-curation win was never filed as an issue
          detail: |-
            #29's Spec says the fr.Multi / it.Devoto-Oli / de.DDDSI curation is "split out so the
            cheap win is not blocked on this design", but no issue exists; it survives only as
            prose in #30:92-97 ("nobody has taken yet"). Correctly deferred — it is not this
            issue's purpose — but a promised split with no tracker item evaporates.
          family: split-out-not-filed
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-29T08:42:14-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-08-29T09:17:02-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-08-29T09:48:45-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: applyVoice(&opt, pron) after the session''s derivation reddens main_test.go:368 on the countingCapturer deck assertion.'
          round: 4
        - id: BR-2
          disposition: addressed
          note: TestRawEditorPronPlaysOutsideTheCookedBlock counts CDN requests inside the cooked callback and requires zero.
          round: 4
        - id: BR-3
          disposition: addressed
          note: 'The H2 now sits at atlas/define.md:1270, after the #23 M2 H3 and before ## Conformance; nothing is re-parented.'
          round: 4
        - id: BR-4
          disposition: addressed
          note: Revisions entry plus three added rows, and the claim is now mechanically checked at declaration level. See M-1 for the family's remaining holes.
          round: 4
        - id: BR-5
          disposition: addressed
          note: voice.langOrDefault is one accessor with both readers deriving from it, pinned by TestTheVoiceReportNamesALanguageEvenWithAZeroVoice.
          round: 4
        - id: BR-6
          disposition: addressed
          note: sourceCandidates now calls askedForSource() rather than spelling its negation.
          round: 4
        - id: BR-7
          disposition: addressed
          note: dict_fake_test.go:85 iterates slices.Sorted(maps.Keys(d.entries)).
          round: 4
        - id: BR-8
          disposition: addressed
          note: README.md:39 adds the -pron line to the define cheat-sheet.
          round: 4
        - id: BR-9
          disposition: addressed
          note: planStatus is a controlled vocabulary failing loudly outside it; mutation-verified — removing the emphasis Trim reddens four fixture cells.
          round: 4
        - id: BR-10
          disposition: addressed
          note: workshop/issues/000031-curate-dictionaries.md exists.
          round: 4
      findings:
        - id: BR-11
          severity: Minor
          title: The status guard fails open on untouched files, and nothing checks the table is complete
          detail: |-
            3rd in family — state the rule, do not fix the two instances. repo_guard_test.go:855
            skips any row whose FILE this window did not touch, so a `modified` row over an
            untouched file is unchecked: I added `| crlfWriter | cmd/define/crlf.go | modified |`
            to the #29 plan in a scratch copy and both plan guards stayed green. And nothing
            checks tree-to-row: voice.langOrDefault, isASCIIOnly and nothingToReplay are new in
            this window with no row, while the AudioCandidates row names langOrDefault in its own
            status text. The rule: the table is a projection of the diff in BOTH directions, and
            the touched==nil skip belongs inside checkPlanStatus as inWindow=false. Its two
            t.Errorf branches also have no fixture, which is the same green-when-removed state
            round 3 found in planStatus.
          family: plan-table-vs-tree
          round: 4
        - id: BR-12
          severity: Minor
          title: reportVoice writes a bare \n to a stderr that is a raw terminal on the /pron path
          detail: |-
            main.go:857 uses "\n" while every sibling write in replraw.go (:274, :325, :327)
            spells "\r\n", and stderr is wrapped in crlfWriter only for the ask path (replraw.go:172)
            and the review loop (play_loop.go:65) — not for replayInPlace. Confirmed by scratch
            test: runEditor driven with "jalapeno\r/pron es\r" against an English-only CDN gives
            stderr = "define: no es recording for jalapeno; played the en one\n". Masked today
            because runEditor writes "\r\n" right after replayInPlace returns, hence Minor. The
            sibling defect pre-exists on playAnnounced's error line (main.go:824), so the fix
            belongs at the seam, not on this line.
          family: raw-mode-bare-newline
          round: 4
        - id: BR-13
          severity: Minor
          title: The NOAD-backed live rows SKIP in every review environment, fourth round running
          detail: |-
            TestLiveDictionaryResolvesAnUnaccentedQuery skips all five subtests here with "NOAD
            unavailable: no dictionary entry", as does TestFixturesMatchLiveDictionary. So the
            chain the feature IS — typed jalapeno reaching jalapeño_es_es via NOAD's headword —
            has never run against both real dependencies at a gate; only the fake models the
            dictionary half. The issue's Log shows the implementor ran the chain against the live
            dictionary during design, so this is an evidence-recording gap. Ask: --verified should
            name a run of the unfiltered conformance suite AND the plan's CLI script from a
            context where NOAD answers, on this final HEAD.
          family: conformance-row-never-runs
          round: 4
      blocked: false
---

# Gate ledger — tools#29 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T08:13:51-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `done-when-unpinned` The issue's first Done-when — the session does not move — has no automated assertion, in the test named for it
  cmd/define/main_test.go:320 TestPronFetchesTheSourceRecordingWithoutMovingTheSession
  asserts the first CDN URL, the walk length, the printed entry and empty stderr, and
  nothing about the deck, the dictionary or d.lang. The rig wires noopCapturer, so it
  could not have recorded one. Task 6 Step 1(c) promised exactly that assertion. NOAD
  is unreachable from this process context, so the plan's manual `ls words/` check is
  also not reproducible at this gate. Use countingCapturer (capture_test.go:73) or a
  store-backed capturer as askrun_test.go:34 does.
- **BR-2** [Important] `two-loops-one-test` The raw editor's /pron branch is untested, though it carries the design's only real hazard
  cmd/define/replraw.go:223-251 is the record-in-cooked / perform-in-raw split that
  lessons.md's "render cooked, play raw" exists for. Task 7 named cmd/define/repl_test.go
  and it was not touched. The repo already pairs TestLineLoopDispatchesCommands with
  TestRawEditorDispatchesCommands for this reason, and editorRig/scriptKeys make it a
  ~20-line test; the cooked-block property is assertable by counting CDN requests inside
  the cooked callback. I verified by scratch test that the path works today, so this is
  coverage rather than correctness.
- **BR-3** [Important] `heading-reparents-prose` atlas/define.md:1170 — the new H2 was inserted mid-section and re-parented four paragraphs and an H3
  "## Source pronunciation (#29)" sits between the locale-help block and the paragraph
  explaining it, so the localeHelp derivation note (:1249), the -locale English-only
  history (:1254), the playN note (:1258), "A missing recording is not a failed lookup"
  (:1263) and the H3 "### The dictionary follows the language (#23 M2)" (:1266) now read
  as part of the source-pronunciation section. Move the new section to just before
  "## Conformance" (:1329).
- **BR-4** [Important] `plan-table-vs-tree` The plan's Core-concepts table calls fakeCDN "unchanged — REUSED" after the diff changed it
  workshop/plans/000029-origin-pronunciation-plan.md:80 claims fakeCDN was reused
  untouched; the diff changed its keying from r.URL.Path to r.URL.EscapedPath, which is
  load-bearing. fakeDictionary and rebasedSource were also modified and appear in no
  row. The issue's Log records all three honestly; the plan does not. Fix with a
  "## Revisions" entry, not code.
- **BR-5** [Minor] `report-must-use-the-normalised-value` reportVoice prints undefaulted Lang fields while AudioCandidates defaults them
  cmd/define/main.go:856 formats u.Source.Lang and u.Session.Lang raw, while
  AudioCandidates (audiourl.go:117) substitutes store.DefaultLang for an empty Lang. With
  a zero opt.voice the record reads "played the  one" — observed in a scratch run.
  Production always calls applyVoice, so it is latent, but this line is the record the
  design insists must be true.
- **BR-6** [Minor] `one-predicate-two-spellings` utterance.sourceCandidates and utterance.askedForSource write the same predicate twice, negated
  audiourl.go:180 guards on `u.Source.Lang == "" || u.Source == u.Session`; audiourl.go:233
  returns its negation. The latter's comment says the two must agree — make it so by
  construction: sourceCandidates should call askedForSource() (ARCH-DRY).
- **BR-7** [Minor] `nondeterministic-fake` fakeDictionary's accent-insensitive fallback iterates a Go map
  cmd/define/dict_fake_test.go:79 — with two entries that both differ from the query only
  by diacritics, the entry returned is nondeterministic. Iterate sorted keys.
- **BR-8** [Minor] `doc-sweep-incomplete` README's define cheat-sheet lists every other flag a reader types, but not -pron
  README.md:29-41 shows --sound, -no-audio, -locale, -lang, -raw and -no-color; Task 8
  Step 3 named "README flag table". The prose section further down does document -pron,
  so this is completeness rather than a gap in the docs gate.
- **BR-9** [Minor] `guard-heuristic-too-loose` The plan-guard's new-row exemption matches any status cell containing "new"
  cmd/define/repo_guard_test.go:593 scans the whole document for "- [ ] " and then
  exempts a row whose status cell lower-cases to contain "new" — "renewed" or "newly"
  would exempt too. Low risk while gated on in-progress, but an anchored match is one
  character more.
- **BR-10** [Minor] `split-out-not-filed` The Spec's split-out dictionary-curation win was never filed as an issue
  #29's Spec says the fr.Multi / it.Devoto-Oli / de.DDDSI curation is "split out so the
  cheap win is not blocked on this design", but no issue exists; it survives only as
  prose in #30:92-97 ("nobody has taken yet"). Correctly deferred — it is not this
  issue's purpose — but a promised split with no tracker item evaporates.

## Round 2 — 2026-08-29T08:42:14-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-08-29T09:17:02-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-08-29T09:48:45-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Mutation-verified: applyVoice(&opt, pron) after the session's derivation reddens main_test.go:368 on the countingCapturer deck assertion.
- BR-2 — addressed — TestRawEditorPronPlaysOutsideTheCookedBlock counts CDN requests inside the cooked callback and requires zero.
- BR-3 — addressed — The H2 now sits at atlas/define.md:1270, after the #23 M2 H3 and before ## Conformance; nothing is re-parented.
- BR-4 — addressed — Revisions entry plus three added rows, and the claim is now mechanically checked at declaration level. See M-1 for the family's remaining holes.
- BR-5 — addressed — voice.langOrDefault is one accessor with both readers deriving from it, pinned by TestTheVoiceReportNamesALanguageEvenWithAZeroVoice.
- BR-6 — addressed — sourceCandidates now calls askedForSource() rather than spelling its negation.
- BR-7 — addressed — dict_fake_test.go:85 iterates slices.Sorted(maps.Keys(d.entries)).
- BR-8 — addressed — README.md:39 adds the -pron line to the define cheat-sheet.
- BR-9 — addressed — planStatus is a controlled vocabulary failing loudly outside it; mutation-verified — removing the emphasis Trim reddens four fixture cells.
- BR-10 — addressed — workshop/issues/000031-curate-dictionaries.md exists.

### Raised

- **BR-11** [Minor] `plan-table-vs-tree` The status guard fails open on untouched files, and nothing checks the table is complete
  3rd in family — state the rule, do not fix the two instances. repo_guard_test.go:855
  skips any row whose FILE this window did not touch, so a `modified` row over an
  untouched file is unchecked: I added `| crlfWriter | cmd/define/crlf.go | modified |`
  to the #29 plan in a scratch copy and both plan guards stayed green. And nothing
  checks tree-to-row: voice.langOrDefault, isASCIIOnly and nothingToReplay are new in
  this window with no row, while the AudioCandidates row names langOrDefault in its own
  status text. The rule: the table is a projection of the diff in BOTH directions, and
  the touched==nil skip belongs inside checkPlanStatus as inWindow=false. Its two
  t.Errorf branches also have no fixture, which is the same green-when-removed state
  round 3 found in planStatus.
- **BR-12** [Minor] `raw-mode-bare-newline` reportVoice writes a bare \n to a stderr that is a raw terminal on the /pron path
  main.go:857 uses "\n" while every sibling write in replraw.go (:274, :325, :327)
  spells "\r\n", and stderr is wrapped in crlfWriter only for the ask path (replraw.go:172)
  and the review loop (play_loop.go:65) — not for replayInPlace. Confirmed by scratch
  test: runEditor driven with "jalapeno\r/pron es\r" against an English-only CDN gives
  stderr = "define: no es recording for jalapeno; played the en one\n". Masked today
  because runEditor writes "\r\n" right after replayInPlace returns, hence Minor. The
  sibling defect pre-exists on playAnnounced's error line (main.go:824), so the fix
  belongs at the seam, not on this line.
- **BR-13** [Minor] `conformance-row-never-runs` The NOAD-backed live rows SKIP in every review environment, fourth round running
  TestLiveDictionaryResolvesAnUnaccentedQuery skips all five subtests here with "NOAD
  unavailable: no dictionary entry", as does TestFixturesMatchLiveDictionary. So the
  chain the feature IS — typed jalapeno reaching jalapeño_es_es via NOAD's headword —
  has never run against both real dependencies at a gate; only the fake models the
  dictionary half. The issue's Log shows the implementor ran the chain against the live
  dictionary during design, so this is an evidence-recording gap. Ask: --verified should
  name a run of the unfiltered conformance suite AND the plan's CLI script from a
  context where NOAD answers, on this final HEAD.

## Open findings

- **BR-11** [Minor] `plan-table-vs-tree` The status guard fails open on untouched files, and nothing checks the table is complete
- **BR-12** [Minor] `raw-mode-bare-newline` reportVoice writes a bare \n to a stderr that is a raw terminal on the /pron path
- **BR-13** [Minor] `conformance-row-never-runs` The NOAD-backed live rows SKIP in every review environment, fourth round running
