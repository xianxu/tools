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

## Open findings

- **BR-1** [Important] `done-when-unpinned` The issue's first Done-when — the session does not move — has no automated assertion, in the test named for it
- **BR-2** [Important] `two-loops-one-test` The raw editor's /pron branch is untested, though it carries the design's only real hazard
- **BR-3** [Important] `heading-reparents-prose` atlas/define.md:1170 — the new H2 was inserted mid-section and re-parented four paragraphs and an H3
- **BR-4** [Important] `plan-table-vs-tree` The plan's Core-concepts table calls fakeCDN "unchanged — REUSED" after the diff changed it
- **BR-5** [Minor] `report-must-use-the-normalised-value` reportVoice prints undefaulted Lang fields while AudioCandidates defaults them
- **BR-6** [Minor] `one-predicate-two-spellings` utterance.sourceCandidates and utterance.askedForSource write the same predicate twice, negated
- **BR-7** [Minor] `nondeterministic-fake` fakeDictionary's accent-insensitive fallback iterates a Go map
- **BR-8** [Minor] `doc-sweep-incomplete` README's define cheat-sheet lists every other flag a reader types, but not -pron
- **BR-9** [Minor] `guard-heuristic-too-loose` The plan-guard's new-row exemption matches any status cell containing "new"
- **BR-10** [Minor] `split-out-not-filed` The Spec's split-out dictionary-curation win was never filed as an issue
