---
gate: boundary-review
issue: 72
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-17T16:11:25-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: the production-chain ordering assertion the Done-when names is not delivered
          detail: |-
            Done-when line 1 and plan row 6 require a test that text reaches the sink
            BEFORE the stream ends, through the production chain, driven by
            Reply{AfterText, FinishRelease} — the barrier PQ-5 was disposed `addressed`
            on. The delivered TestALongPassageReachesTheScreenInPieces
            (cmd/define/ask_language_test.go:139) instead checks largest single write
            after the run completes. Verified by mutation in a scratch worktree: a
            decoder that buffers the passage and releases it rune-by-rune at the close
            marker PASSES that test; only the pure decoder test catches it. So no
            production-chain test asserts the ordering property, and a hold introduced
            downstream of the decoder would ship green. The barrier already exists at
            internal/llm/llmtest/fake.go:115-117 and is used by llm_activity_test.go:26.
          family: envelope-guard-unautomated
          round: 1
        - id: BR-2
          severity: Minor
          title: the e2e streaming test runs at width 0, so the wrap writer is never exercised
          detail: |-
            options{color: true} leaves opt.width at 0, so WriteOwned is a pass-through
            and ownedAnswerWrapWriter's row-commit path — the remaining hold, and the
            subject of the envelope's "first row 0.3 s, ~0.5 s per row at width 100"
            budget — is not on the tested path. Those figures stay Log-only facts
            (ARCH-CONSTRAINTS).
          family: envelope-guard-unautomated
          round: 1
        - id: BR-3
          severity: Minor
          title: splitWordInsideAPassage does not require the passage language to match the session
          detail: |-
            cmd/define/askhighlight_test.go:29 accepts a word in ANY [lang=..] region,
            but TestOwnedPassageHighlightsAWordSplitAcrossDeltas sets d.lang = "en" and
            only an `en` region can highlight (vocabularyFor returns nil otherwise). All
            twelve split-word candidates in stream-language.sse happen to sit in `en`
            regions today, so it passes; a re-record can turn a correct implementation
            into a misleading "was not highlighted" failure. Pass the wanted lang into
            the predicate.
          family: test-helper-underconstrained
          round: 1
        - id: BR-4
          severity: Minor
          title: three copies of the span-offset accumulator, and decodedChunksBounded clones decodedChunks
          detail: |-
            cmd/define/language_decode_test.go:93 is a verbatim copy of :9 plus three
            lines of assertion, and language_conformance_test.go:120 open-codes a third
            copy with merging added. One helper with an optional per-chunk hook and a
            merge flag (ARCH-DRY).
          family: duplicated-test-helper
          round: 1
        - id: BR-5
          severity: Minor
          title: '"dominant passage" is defined twice, over different denominators'
          detail: |-
            assertDominantPassage (cmd/define/ask_language_test.go:174) requires the
            longest region to be at least half the RAW capture including marker bytes;
            the recorder (language_conformance_test.go:143) requires 60 percent of
            DECODED text. The replay guard and the promotion guard can disagree about
            the same capture.
          family: one-rule-two-statements
          round: 1
        - id: BR-6
          severity: Minor
          title: annotatedRegions does not handle nested markers
          detail: |-
            cmd/define/askhighlight_test.go:49 takes the first [/lang] after an open, so
            the nested [lang=en]Sycophant[lang=es][/lang][/lang] in
            stream-long-passage.sse yields a region whose body contains marker bytes.
            Harmless at the current thresholds, wrong as a general helper.
          family: one-rule-two-statements
          round: 1
        - id: BR-7
          severity: Minor
          title: (*languageAnswer).vocabularyFor shares its name with the package-level vocabularyFor
          detail: |-
            cmd/define/answer_language.go:79 versus the package function called at
            ask.go:164. Legal Go, but two unrelated things named the same in one
            package.
          family: shadowed-name
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T16:33:13-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-17T16:47:12-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: a buffer-then-dribble decoder reds both widths at ask_language_test.go:207 ("nothing reached the screen until delta 102 of 161"); production chain via deltaObserver, no clock.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: The terminal subtest runs at width 100; 1742 bytes in 17 writes, largest 135, so the row-commit path is on the tested path.
          round: 3
        - id: BR-3
          disposition: addressed
          note: splitWordInsideAPassage takes the wanted lang and matches it against the region; residual helper/parser divergence raised separately at 4.1.
          round: 3
        - id: BR-4
          disposition: addressed
          note: One spanAccumulator and one decodeChunks with a per-chunk hook; the conformance recorder now uses the shared one. Duplicated doc comment noted at 4.2.
          round: 3
        - id: BR-5
          disposition: addressed
          note: dominantPassage/longestPassage stated once over decoded text, used by both the promotion guard and the replay guard.
          round: 3
        - id: BR-6
          disposition: addressed
          note: A region now ends at a nested open; the specific defect (region body containing marker bytes) is gone, verified against the nested form in stream-long-passage.sse.
          round: 3
        - id: BR-7
          disposition: addressed
          note: The method is runVocabulary, with a comment explaining why it is not vocabularyFor.
          round: 3
      findings:
        - id: BR-8
          severity: Important
          title: 'the shadow-sweep enumerated prose consumers and missed the executable one: the ask prompt still restates the deleted body bound'
          detail: |-
            3rd finding in this family, so per the repeat protocol the deliverable is the RULE, not this site.
            Rule - a quantitative bound in this subsystem has exactly one statement, and every other site derives
            from it or cites it by name; a prompt is a site. Measured prevalence, five bounds in the subsystem -
            header candidate (single-sourced), decoder retention (fixed round 1), dominant passage (fixed round 1),
            passage length and entity candidate (both open). Passage length - sharedLanguageGrammar
            (cmd/define/askctx.go:176) still says "Keep passages below 4000 characters", which is 16000 bytes, the
            UTF-8 worst-case fit inside the deleted languageBodyLimit; git log -S puts the clause and the constant
            in the same commit 62c6a66 (issue 65). It now protects nothing, contradicts the atlas line landed in
            this diff ("the only bounds now are the 64-byte header candidate and a stated total retention"), and
            pushes the model toward the fragmented passage shape TestLongPassageStreamsAgainstLiveService refuses.
            Entity candidate - maxLanguageDecoderRetained is 2*languageHeaderLimit + 2*utf8.UTFMax and so now
            depends on the entity cap being languageHeaderLimit, but the entity check at
            cmd/define/language_decode.go:245 is a bare literal 64; this coupling was introduced by this diff. It
            fails loudly via the fuzz invariant rather than silently, so it is the mild member - sweep it in the
            same pass.
          family: one-rule-two-statements
          round: 3
        - id: BR-9
          severity: Minor
          title: annotatedRegions reports text after a nested open as owned, where the parser has it in recovery
          detail: |-
            2nd in this family, so the deliverable is the rule - a test helper that must know where the parser puts
            a boundary derives that boundary from the parser, not from a second scan. cmd/define/askhighlight_test.go:90
            ends a region at a nested open (BR-6's fix) but then rescans FROM that open, emitting the nested passage
            as a region. Measured in a scratch test: "[lang=en]foo[lang=es]bar[/lang] tail" yields regions en "foo"
            and es "bar", while the decoder yields only en "foo" - "bar" is in recovery and owned by nobody. Harmless
            on today's capture (the nested region there is empty) and the failure mode is a false red, not a false
            green. The derivation is available: run the capture through languageDecoder one delta at a time and record
            the decoded length at each delta boundary, which yields both "split across deltas" and "owned by lang" from
            production, collapsing annotatedRegions, the raw-vs-decoded offset mismatch and BR-3's lang parameter into
            one mechanism.
          family: test-helper-underconstrained
          round: 3
        - id: BR-10
          severity: Minor
          title: the retention-invariant rationale is duplicated verbatim onto decodeChunks, which does not assert it
          detail: |-
            2nd in this family; same one-statement rule as BR-4. The eight-line comment at
            cmd/define/language_decode_test.go:31-40 is character-for-character the one at :138-147, and decodeChunks
            in its decodedChunks form takes a no-op hook and asserts nothing. Delete the copy at :31-40 - the
            invariant's home is decodedChunksBounded. Separately, assertDominantPassage's comment
            (cmd/define/ask_language_test.go:222-225) describes the bug BR-5 fixed in the present tense, inside the fix.
          family: duplicated-test-helper
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-17T17:07:07-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: addressed
          note: 'Both sites fixed and the class swept: the "Keep passages below 4000 characters" clause is gone from sharedLanguageGrammar (cmd/define/askctx.go:180) with both goldens re-recorded and TestRenderAskPrompt/TestRenderPassagePrompt green, and the entity cap at cmd/define/language_decode.go:246 now cites languageHeaderLimit; grep for 4000/16 KiB/languageBodyLimit/d.body/decodeLimit returns only historical comments.'
          round: 4
        - id: BR-9
          disposition: addressed
          note: 'annotatedRegions, splitWordMatching and splitWordInsideAPassage are deleted; splitWordOwnedBy (cmd/define/askhighlight_test.go:29) derives both the delta boundaries and the ownership by running the capture through the real languageDecoder one delta at a time, so the helper cannot disagree with the parser. Verified reachable: the owned-path test still finds "phrase" and still goes red when highlightRegion is restored.'
          round: 4
        - id: BR-10
          disposition: addressed
          note: The duplicated eight-line retention rationale is deleted from decodeChunks (the invariant's only statement is now decodedChunksBounded, cmd/define/language_decode_test.go:132-140), and assertDominantPassage's comment (cmd/define/ask_language_test.go:227-230) no longer describes the fixed bug in the present tense.
          round: 4
      findings:
        - id: BR-11
          severity: Minor
          title: the run highlighter's vocabulary is derived in own() and restated in the constructor
          detail: |-
            This is the 4th finding in family `one-rule-two-statements`. Earlier rounds fixed instances
            (decoder retention, dominant passage, passage length, entity candidate). Do NOT fix these two
            instances alone — state the rule and sweep it.
            Rule: a fact with one authority has exactly one derivation site; every other site calls that
            derivation or names it. Measured prevalence, two open sites. (1) cmd/define/answer_language.go:74
            derives the run's vocabulary as a.runVocabulary(lang); cmd/define/answer_language.go:32 restates
            it as the raw v. They agree today only because runVocabulary("") returns a.vocab unconditionally,
            so if the neutral-run rule ever changes (plausibly in #64) an answer that opens neutral silently
            keeps the old vocabulary for its first run and one that opens inside a passage does not. It is
            also an ARCH-ORDER bypass: own() is the sole transition for the (ownership, highlight) pair and
            the constructor sets it directly. Fix is one line — newHighlightWriter(a, a.runVocabulary(""), knownOn).
            (2) atlas/define.md:2317 says "the only bounds now are the 64-byte header candidate and a stated
            total retention (maxLanguageDecoderRetained)" — it names the constant for one bound and spells
            the number for the other; "the 64-byte header candidate (languageHeaderLimit)" closes it.
          family: one-rule-two-statements
          round: 4
        - id: BR-12
          severity: Minor
          title: splitWordOwnedBy reports a production streaming regression as "re-record the capture"
          detail: |-
            This is the 3rd finding in family `test-helper-underconstrained` (BR-3: the helper could hand back
            a foreign-region word and report a correct implementation as "was not highlighted"; BR-9: the
            helper disagreed with the parser about a boundary). Do NOT fix this instance alone — state the rule.
            Rule: a helper that computes a test's precondition from production code must separate "the fixture
            lacks the case" from "production stopped producing the case", because after BR-9 those two share a
            single derivation and therefore a single failure message. Measured: with the segment buffer restored
            in a scratch worktree at HEAD, cmd/define/askhighlight_test.go:68 fires with "no word in
            stream-language.sse is split across deltas and owned by \"en\" — re-record the capture or pick
            another". The test does go red, so the gate is safe; but the message directs a maintainer to
            re-record a capture that is not the problem. The split is cheap and stays inside BR-9's rule: assert
            first that the decoded capture carries any span owned by lang (a fixture property — still true under
            the buffering mutation, since spans exist, they just arrive at the close), then that a delta boundary
            falls inside such a word (the production property), with a message naming streaming.
          family: test-helper-underconstrained
          round: 4
        - id: BR-13
          severity: Minor
          title: the 0.5 s first-paint budget has no post-change measurement at width 100 and no guard in its own units
          detail: |-
            This is the 3rd finding in family `envelope-guard-unautomated`. Do NOT add a wall-clock assertion —
            the operator already decided at PQ-5 that ordering is the deterministic stand-in and the second
            figure stays a `## Log` fact. State the rule instead.
            Rule: every budget in an `Operating envelope` block names, in the block itself, either the test that
            enforces it or the `## Log` measurement that is its evidence after the change; a budget with neither
            is an assumption, not a bound. Measured prevalence in this envelope: retention -> guarded
            (FuzzLanguageDecoderChunks, verified over 507k execs); piped first-byte -> measured after
            (0.917 s, `## Log`); row cadence -> measured, wrapper untouched; first text visible <= 0.5 s at
            width 100 -> neither. Its stated basis (0.3 s) was measured on the wrap writer in isolation before
            the decoder hold was removed, so it never described the composed path; the only post-change signal
            is TestALongPassageReachesTheScreenInPieces logging "first visible at delta 13 of 161" at width 100
            against delta 2 at width 0, and nothing translates 13 deltas into the envelope's units. The ordering
            guard at cmd/define/ask_language_test.go:206 admits anything up to delta 40, so it does not stand in
            for 0.5 s. Either record the composed width-100 figure in `## Log` beside the piped one, or mark the
            budget as carried by the ordering guard and drop the second-level number.
          family: envelope-guard-unautomated
          round: 4
      blocked: false
---

# Gate ledger — tools#72 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T16:11:25-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `envelope-guard-unautomated` the production-chain ordering assertion the Done-when names is not delivered
  Done-when line 1 and plan row 6 require a test that text reaches the sink
  BEFORE the stream ends, through the production chain, driven by
  Reply{AfterText, FinishRelease} — the barrier PQ-5 was disposed `addressed`
  on. The delivered TestALongPassageReachesTheScreenInPieces
  (cmd/define/ask_language_test.go:139) instead checks largest single write
  after the run completes. Verified by mutation in a scratch worktree: a
  decoder that buffers the passage and releases it rune-by-rune at the close
  marker PASSES that test; only the pure decoder test catches it. So no
  production-chain test asserts the ordering property, and a hold introduced
  downstream of the decoder would ship green. The barrier already exists at
  internal/llm/llmtest/fake.go:115-117 and is used by llm_activity_test.go:26.
- **BR-2** [Minor] `envelope-guard-unautomated` the e2e streaming test runs at width 0, so the wrap writer is never exercised
  options{color: true} leaves opt.width at 0, so WriteOwned is a pass-through
  and ownedAnswerWrapWriter's row-commit path — the remaining hold, and the
  subject of the envelope's "first row 0.3 s, ~0.5 s per row at width 100"
  budget — is not on the tested path. Those figures stay Log-only facts
  (ARCH-CONSTRAINTS).
- **BR-3** [Minor] `test-helper-underconstrained` splitWordInsideAPassage does not require the passage language to match the session
  cmd/define/askhighlight_test.go:29 accepts a word in ANY [lang=..] region,
  but TestOwnedPassageHighlightsAWordSplitAcrossDeltas sets d.lang = "en" and
  only an `en` region can highlight (vocabularyFor returns nil otherwise). All
  twelve split-word candidates in stream-language.sse happen to sit in `en`
  regions today, so it passes; a re-record can turn a correct implementation
  into a misleading "was not highlighted" failure. Pass the wanted lang into
  the predicate.
- **BR-4** [Minor] `duplicated-test-helper` three copies of the span-offset accumulator, and decodedChunksBounded clones decodedChunks
  cmd/define/language_decode_test.go:93 is a verbatim copy of :9 plus three
  lines of assertion, and language_conformance_test.go:120 open-codes a third
  copy with merging added. One helper with an optional per-chunk hook and a
  merge flag (ARCH-DRY).
- **BR-5** [Minor] `one-rule-two-statements` "dominant passage" is defined twice, over different denominators
  assertDominantPassage (cmd/define/ask_language_test.go:174) requires the
  longest region to be at least half the RAW capture including marker bytes;
  the recorder (language_conformance_test.go:143) requires 60 percent of
  DECODED text. The replay guard and the promotion guard can disagree about
  the same capture.
- **BR-6** [Minor] `one-rule-two-statements` annotatedRegions does not handle nested markers
  cmd/define/askhighlight_test.go:49 takes the first [/lang] after an open, so
  the nested [lang=en]Sycophant[lang=es][/lang][/lang] in
  stream-long-passage.sse yields a region whose body contains marker bytes.
  Harmless at the current thresholds, wrong as a general helper.
- **BR-7** [Minor] `shadowed-name` (*languageAnswer).vocabularyFor shares its name with the package-level vocabularyFor
  cmd/define/answer_language.go:79 versus the package function called at
  ask.go:164. Legal Go, but two unrelated things named the same in one
  package.

## Round 2 — 2026-09-17T16:33:13-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-17T16:47:12-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: a buffer-then-dribble decoder reds both widths at ask_language_test.go:207 ("nothing reached the screen until delta 102 of 161"); production chain via deltaObserver, no clock.
- BR-2 — addressed — The terminal subtest runs at width 100; 1742 bytes in 17 writes, largest 135, so the row-commit path is on the tested path.
- BR-3 — addressed — splitWordInsideAPassage takes the wanted lang and matches it against the region; residual helper/parser divergence raised separately at 4.1.
- BR-4 — addressed — One spanAccumulator and one decodeChunks with a per-chunk hook; the conformance recorder now uses the shared one. Duplicated doc comment noted at 4.2.
- BR-5 — addressed — dominantPassage/longestPassage stated once over decoded text, used by both the promotion guard and the replay guard.
- BR-6 — addressed — A region now ends at a nested open; the specific defect (region body containing marker bytes) is gone, verified against the nested form in stream-long-passage.sse.
- BR-7 — addressed — The method is runVocabulary, with a comment explaining why it is not vocabularyFor.

### Raised

- **BR-8** [Important] `one-rule-two-statements` the shadow-sweep enumerated prose consumers and missed the executable one: the ask prompt still restates the deleted body bound
  3rd finding in this family, so per the repeat protocol the deliverable is the RULE, not this site.
  Rule - a quantitative bound in this subsystem has exactly one statement, and every other site derives
  from it or cites it by name; a prompt is a site. Measured prevalence, five bounds in the subsystem -
  header candidate (single-sourced), decoder retention (fixed round 1), dominant passage (fixed round 1),
  passage length and entity candidate (both open). Passage length - sharedLanguageGrammar
  (cmd/define/askctx.go:176) still says "Keep passages below 4000 characters", which is 16000 bytes, the
  UTF-8 worst-case fit inside the deleted languageBodyLimit; git log -S puts the clause and the constant
  in the same commit 62c6a66 (issue 65). It now protects nothing, contradicts the atlas line landed in
  this diff ("the only bounds now are the 64-byte header candidate and a stated total retention"), and
  pushes the model toward the fragmented passage shape TestLongPassageStreamsAgainstLiveService refuses.
  Entity candidate - maxLanguageDecoderRetained is 2*languageHeaderLimit + 2*utf8.UTFMax and so now
  depends on the entity cap being languageHeaderLimit, but the entity check at
  cmd/define/language_decode.go:245 is a bare literal 64; this coupling was introduced by this diff. It
  fails loudly via the fuzz invariant rather than silently, so it is the mild member - sweep it in the
  same pass.
- **BR-9** [Minor] `test-helper-underconstrained` annotatedRegions reports text after a nested open as owned, where the parser has it in recovery
  2nd in this family, so the deliverable is the rule - a test helper that must know where the parser puts
  a boundary derives that boundary from the parser, not from a second scan. cmd/define/askhighlight_test.go:90
  ends a region at a nested open (BR-6's fix) but then rescans FROM that open, emitting the nested passage
  as a region. Measured in a scratch test: "[lang=en]foo[lang=es]bar[/lang] tail" yields regions en "foo"
  and es "bar", while the decoder yields only en "foo" - "bar" is in recovery and owned by nobody. Harmless
  on today's capture (the nested region there is empty) and the failure mode is a false red, not a false
  green. The derivation is available: run the capture through languageDecoder one delta at a time and record
  the decoded length at each delta boundary, which yields both "split across deltas" and "owned by lang" from
  production, collapsing annotatedRegions, the raw-vs-decoded offset mismatch and BR-3's lang parameter into
  one mechanism.
- **BR-10** [Minor] `duplicated-test-helper` the retention-invariant rationale is duplicated verbatim onto decodeChunks, which does not assert it
  2nd in this family; same one-statement rule as BR-4. The eight-line comment at
  cmd/define/language_decode_test.go:31-40 is character-for-character the one at :138-147, and decodeChunks
  in its decodedChunks form takes a no-op hook and asserts nothing. Delete the copy at :31-40 - the
  invariant's home is decodedChunksBounded. Separately, assertDominantPassage's comment
  (cmd/define/ask_language_test.go:222-225) describes the bug BR-5 fixed in the present tense, inside the fix.

## Round 4 — 2026-09-17T17:07:07-07:00 (claude) — passed

### Disposed

- BR-8 — addressed — Both sites fixed and the class swept: the "Keep passages below 4000 characters" clause is gone from sharedLanguageGrammar (cmd/define/askctx.go:180) with both goldens re-recorded and TestRenderAskPrompt/TestRenderPassagePrompt green, and the entity cap at cmd/define/language_decode.go:246 now cites languageHeaderLimit; grep for 4000/16 KiB/languageBodyLimit/d.body/decodeLimit returns only historical comments.
- BR-9 — addressed — annotatedRegions, splitWordMatching and splitWordInsideAPassage are deleted; splitWordOwnedBy (cmd/define/askhighlight_test.go:29) derives both the delta boundaries and the ownership by running the capture through the real languageDecoder one delta at a time, so the helper cannot disagree with the parser. Verified reachable: the owned-path test still finds "phrase" and still goes red when highlightRegion is restored.
- BR-10 — addressed — The duplicated eight-line retention rationale is deleted from decodeChunks (the invariant's only statement is now decodedChunksBounded, cmd/define/language_decode_test.go:132-140), and assertDominantPassage's comment (cmd/define/ask_language_test.go:227-230) no longer describes the fixed bug in the present tense.

### Raised

- **BR-11** [Minor] `one-rule-two-statements` the run highlighter's vocabulary is derived in own() and restated in the constructor
  This is the 4th finding in family `one-rule-two-statements`. Earlier rounds fixed instances
  (decoder retention, dominant passage, passage length, entity candidate). Do NOT fix these two
  instances alone — state the rule and sweep it.
  Rule: a fact with one authority has exactly one derivation site; every other site calls that
  derivation or names it. Measured prevalence, two open sites. (1) cmd/define/answer_language.go:74
  derives the run's vocabulary as a.runVocabulary(lang); cmd/define/answer_language.go:32 restates
  it as the raw v. They agree today only because runVocabulary("") returns a.vocab unconditionally,
  so if the neutral-run rule ever changes (plausibly in #64) an answer that opens neutral silently
  keeps the old vocabulary for its first run and one that opens inside a passage does not. It is
  also an ARCH-ORDER bypass: own() is the sole transition for the (ownership, highlight) pair and
  the constructor sets it directly. Fix is one line — newHighlightWriter(a, a.runVocabulary(""), knownOn).
  (2) atlas/define.md:2317 says "the only bounds now are the 64-byte header candidate and a stated
  total retention (maxLanguageDecoderRetained)" — it names the constant for one bound and spells
  the number for the other; "the 64-byte header candidate (languageHeaderLimit)" closes it.
- **BR-12** [Minor] `test-helper-underconstrained` splitWordOwnedBy reports a production streaming regression as "re-record the capture"
  This is the 3rd finding in family `test-helper-underconstrained` (BR-3: the helper could hand back
  a foreign-region word and report a correct implementation as "was not highlighted"; BR-9: the
  helper disagreed with the parser about a boundary). Do NOT fix this instance alone — state the rule.
  Rule: a helper that computes a test's precondition from production code must separate "the fixture
  lacks the case" from "production stopped producing the case", because after BR-9 those two share a
  single derivation and therefore a single failure message. Measured: with the segment buffer restored
  in a scratch worktree at HEAD, cmd/define/askhighlight_test.go:68 fires with "no word in
  stream-language.sse is split across deltas and owned by \"en\" — re-record the capture or pick
  another". The test does go red, so the gate is safe; but the message directs a maintainer to
  re-record a capture that is not the problem. The split is cheap and stays inside BR-9's rule: assert
  first that the decoded capture carries any span owned by lang (a fixture property — still true under
  the buffering mutation, since spans exist, they just arrive at the close), then that a delta boundary
  falls inside such a word (the production property), with a message naming streaming.
- **BR-13** [Minor] `envelope-guard-unautomated` the 0.5 s first-paint budget has no post-change measurement at width 100 and no guard in its own units
  This is the 3rd finding in family `envelope-guard-unautomated`. Do NOT add a wall-clock assertion —
  the operator already decided at PQ-5 that ordering is the deterministic stand-in and the second
  figure stays a `## Log` fact. State the rule instead.
  Rule: every budget in an `Operating envelope` block names, in the block itself, either the test that
  enforces it or the `## Log` measurement that is its evidence after the change; a budget with neither
  is an assumption, not a bound. Measured prevalence in this envelope: retention -> guarded
  (FuzzLanguageDecoderChunks, verified over 507k execs); piped first-byte -> measured after
  (0.917 s, `## Log`); row cadence -> measured, wrapper untouched; first text visible <= 0.5 s at
  width 100 -> neither. Its stated basis (0.3 s) was measured on the wrap writer in isolation before
  the decoder hold was removed, so it never described the composed path; the only post-change signal
  is TestALongPassageReachesTheScreenInPieces logging "first visible at delta 13 of 161" at width 100
  against delta 2 at width 0, and nothing translates 13 deltas into the envelope's units. The ordering
  guard at cmd/define/ask_language_test.go:206 admits anything up to delta 40, so it does not stand in
  for 0.5 s. Either record the composed width-100 figure in `## Log` beside the piped one, or mark the
  budget as carried by the ordering guard and drop the second-level number.

## Open findings

- **BR-11** [Minor] `one-rule-two-statements` the run highlighter's vocabulary is derived in own() and restated in the constructor
- **BR-12** [Minor] `test-helper-underconstrained` splitWordOwnedBy reports a production streaming regression as "re-record the capture"
- **BR-13** [Minor] `envelope-guard-unautomated` the 0.5 s first-paint budget has no post-change measurement at width 100 and no guard in its own units
