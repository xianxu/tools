---
gate: boundary-review
issue: 21
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-26T14:52:43-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: The editor loop's use of the vocabulary is pinned by no test — highlighting can be switched fully off and the suite stays green
          detail: |-
            Two mutations survive the entire suite: replacing `voc` with `nil` at both
            RenderLine call sites (replraw.go:138, :216), and deleting voc.Load()
            (replraw.go:82). Nothing drives runEditor with a vocabulary and asserts a
            green byte reaches stdout, so the M1 Done-when has no proof. This is also
            Plan Task 4 Step 7's own stated test ("assert it highlights on the next
            render without a reload"), which was replaced by a capturer-to-set unit
            test. editorRig already supplies color:true and runEditor already takes a
            scripted key channel plus an out buffer, so the fix is ~10 lines.
          family: behaviour-claimed-without-a-failing-test
          round: 1
        - id: BR-2
          severity: Important
          title: Capture's "must not claim a word the deck rejected" is a comment with no test behind it
          detail: |-
            capture.go:96-104 documents that vocab.Add runs only after Upsert
            succeeds, but moving the Add above the error check survives the whole
            suite. The fixture already exists — failingStore{} at capture_test.go:62 —
            so the pin costs four lines. Same rule as the editor-loop finding: sweep
            the diff for behaviours stated only in comments rather than fixing these
            two sites alone.
          family: behaviour-claimed-without-a-failing-test
          round: 1
        - id: BR-3
          severity: Minor
          title: storeVocabulary's once-only Load guard survives removal; the test asserting it cannot fail
          detail: |-
            vocab_test.go:74 comments "second Load is free and must not double
            anything", but Add is idempotent and maxWords is a max, so deleting the
            `loaded` guard at vocab.go:107 keeps every assertion green. A counting
            store double would make the assertion bite.
          family: behaviour-claimed-without-a-failing-test
          round: 1
        - id: BR-4
          severity: Minor
          title: cmd/define/capture.go is not gofmt-clean
          detail: |-
            The new `vocab Vocabulary` field at capture.go:70 is misaligned; gofmt -l
            flags this file and no other in the package. No CI gate catches it.
          family: unformatted-source
          round: 1
        - id: BR-5
          severity: Minor
          title: isWordRune admits apostrophe and hyphen at token edges, so 'obsequious' and word--word never match
          detail: |-
            highlight.go:25 treats ' and - as word runes unconditionally, so a quoted
            or double-dashed occurrence tokenizes into a run whose store.Key differs
            from the deck key. Harmless on the prompt line; definition bodies in M2
            routinely quote and dash. Relatedly, MaxPhraseWords counts with
            strings.Fields (vocab.go:70), which disagrees with wordRuns for keys
            holding other punctuation (e.g., 9/11) — such entries are unmatchable.
          family: tokenizer-edge-admits-punctuation
          round: 1
        - id: BR-6
          severity: Minor
          title: atlas/define.md describes the definition path in the present tense before M2 builds it
          detail: |-
            Line 444 says wordRuns is "the single tokenizer both the prompt and the
            definition path use". The M2/M3 forward references elsewhere in the same
            section are correctly marked as future; this one is not.
          family: atlas-claims-unbuilt-surface
          round: 1
        - id: BR-7
          severity: Minor
          title: All 26 M1 plan steps remain unchecked while the issue marks M1 complete
          detail: |-
            workshop/plans/000021-highlight-learned-plan.md Tasks 1-4 are entirely
            `- [ ]` at HEAD. The plan also needs Revisions entries for Task 4 Step 2
            (its "assert absence of \x1b" instruction is infeasible — eraseLine and
            the cursor park are escapes) and Task 4 Step 7 (test not delivered).
          family: plan-record-not-updated
          round: 1
        - id: BR-8
          severity: Minor
          title: '`_ = unicode.IsLetter` in highlight_test.go:92 exists only to justify an unused import'
          detail: Drop the statement and the `unicode` import together.
          family: dead-test-scaffolding
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-08-26T15:11:28-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-08-26T15:33:16-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by mutation: nil at both RenderLine call sites and deleting voc.Load() each redden TestEditorLoopHighlightsADeckWordOnScreen.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: Hoisting vocab.Add above the Upsert error check reddens TestCaptureDoesNotAddAWordTheDeckRejected via the Upsert-only double.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Removing the loaded guard reddens TestStoreVocabularyReadsTheDeckOnlyOnce; countingDeck counts Deck() reads.
          round: 3
        - id: BR-4
          disposition: addressed
          note: gofmt -l over cmd/ and internal/ reports nothing.
          round: 3
        - id: BR-5
          disposition: addressed
          note: appendTrimmed trims joiners at token edges; MaxPhraseWords counts with wordRuns; the punctuated-key limit is pinned by TestAPunctuatedKeyIsNotMatchable.
          round: 3
        - id: BR-6
          disposition: addressed
          note: atlas/define.md now reads "used by the prompt line today and by the definition and answer paths from M2".
          round: 3
        - id: BR-7
          disposition: addressed
          note: All 26 M1 steps are ticked and three Revisions entries record the Task 4 Step 2 infeasibility, the skipped Step 7, and the wordRuns contract change.
          round: 3
        - id: BR-8
          disposition: addressed
          note: The statement and the unicode import are both gone from highlight_test.go.
          round: 3
      findings:
        - id: BR-9
          severity: Minor
          title: 'A third near-identical warnf, with the "define: " prefix now written in three places (ARCH-DRY)'
          detail: |-
            vocab.go:126 is a copy of history_store.go:82 minus its suffix, and capture.go:125
            is the same shape with a once-per-process flag. The shared part — the nil-writer
            guard and the "define: " prefix — should be one helper the three wrap; today
            renaming the program means finding three string literals.
          family: copy-pasted-helper
          round: 3
        - id: BR-10
          severity: Minor
          title: voc.Load() reads the whole deck even with -no-color, where nothing can consume it
          detail: |-
            replraw.go:82 loads unconditionally, but RenderLine only calls highlightSpans
            inside its `if color` branch, so with colour off the deck walk and YAML parse
            buy nothing. hist.Load() beside it is needed either way, which is what makes
            the unconditional shape look right.
          family: io-for-a-disabled-feature
          round: 3
        - id: BR-11
          severity: Minor
          title: The Vocabulary seam has two absent-representations and four guards, one of them already unreachable
          detail: |-
            nil and an empty memVocabulary both mean "nothing highlighted", defended at
            highlight.go:113, replraw.go:79, main.go:134 and capture.go:102. run() calls
            withStore before repl(), so replraw.go:79 cannot fire in production. M2 adds two
            more consumers of the same seam; picking one representation now keeps this at one
            guard rather than six.
          family: one-absence-representation-per-seam
          round: 3
        - id: BR-12
          severity: Minor
          title: A deck word used as a command name highlights inside the command namespace
          detail: |-
            Confirmed by direct render: with "history" in the deck, typing "/history 7" emits
            "\x1b[1m/\x1b[1;32mhistory\x1b[0m\x1b[1m 7". parseCommandLine's own comment says
            "/" opens a real separate namespace, not a marker on a word, so highlighting
            inside it reads as the vocabulary feature leaking across that boundary. It may be
            intended and harmless; nothing in the Spec's out-of-scope list decides it either way.
          family: feature-leaks-across-namespace
          round: 3
      boundary: M1
      blocked: false
    - "n": 4
      timestamp: "2026-08-26T15:52:11-07:00"
      agent: claude
      boundary: M2
      blocked: false
      protocol_error: no valid findings block
    - "n": 5
      timestamp: "2026-08-26T16:09:13-07:00"
      agent: claude
      findings:
        - id: BR-13
          severity: Critical
          title: highlightWriter loses a match when a chunk splits on a joiner or mid-rune
          detail: |-
            decidedEnd (cmd/define/highlightwriter.go:172) tests the bytes after the
            TRIMMED token end, so a trailing apostrophe/hyphen or an incomplete UTF-8
            sequence reads as "punctuation closed the token" and the region is
            released. Measured: don' + t, hot- + dog, caf\xc3 + \xa9 each highlight in
            one call and are lost when split, violating the writer's chunk-independence
            contract. FuzzHighlightWriterIsChunkIndependent cannot see the class — its
            vocabulary is the constant vocab("obsequious","hot dog","hot"), which holds
            no joiner-bearing or multi-byte entry, so no exec count reaches it. Fix the
            release test to require evidence the last token cannot grow, and derive the
            fuzz deck and seeds from TestWordRuns' own class table.
          family: release-only-what-cannot-change
          round: 5
        - id: BR-14
          severity: Critical
          title: Definitions never highlight on the one-shot or piped-stdin paths
          detail: |-
            4th in this family — do not fix only this instance. Vocabulary.Load() has
            one call site, runEditor (cmd/define/replraw.go:84), but M2 wired
            highlighting into lookupAndRender (main.go:511), which is also reached by
            one-shot `define <word>` and by replLines. Verified: an unloaded
            storeVocabulary holding "obsequious" produces no highlight for
            "sycophantic"; calling Load() first produces one. 2 of 3 entry paths dead,
            including the one README's new sentence describes. THE RULE: the
            production-chain enumeration begins at the process entry points, not at the
            dependency the test injects — TestDefinitionBodyHighlightsADeckWord injects
            a pre-populated memVocabulary and so cannot see the Load hop, the same
            blindness as M1 round 2's withStore merge. The enumeration to write and
            sweep is one row per entry path reaching the render. Any fix must keep M1
            round 3's opt.color gate on Load.
          family: behaviour-claimed-without-a-failing-test
          round: 5
        - id: BR-15
          severity: Important
          title: The headword line is re-styled, which the Spec lists as out of scope
          detail: |-
            2nd in this family — do not fix only this instance. highlightText is
            applied to the whole rendered string (main.go:511), so a deck word looked
            up again renders its own headword green inside the bold cyan:
            "\x1b[1;36m\x1b[1;32msycophantic\x1b[0m\x1b[1;36m\x1b[0m". The Spec's
            out-of-scope list says "Re-styling the headword line". Nothing tests it
            either way. THE RULE: the vocabulary is withheld per region by an explicit
            decision at the boundary — the shape highlightSetFor already has for the
            command namespace — not by wrapping whatever string is at hand. Enumerate
            every region a renderer produces (headword, syllabification, pronunciation,
            part-of-speech, sense body, examples, and for M3 the answer stream), state
            admit or withhold for each, and pin the withholds.
          family: feature-leaks-across-namespace
          round: 5
        - id: BR-16
          severity: Important
          title: Plan file layout, contract rule 4 and two test names no longer match the code
          detail: |-
            2nd in this family — do not fix only this instance. Six divergences:
            sgrState and highlightWriter are in sgr.go/highlightwriter.go not
            highlight.go; Task 5/6 Files blocks name the wrong files; contract rule 4
            promises (n, err) in caller units while Write returns (0, err) and the test
            asserts n==0 — the opposite — and crlfWriter in the same package returns
            caller-unit progress with a test defending exactly that; the fuzz target
            was renamed; Task 7 Step 2's invariant landed as a different property in a
            different file. THE RULE: a deviation from the plan's stated layout or
            contract lands its "## Revisions" entry in the same commit as the
            deviation. Decide the Write count question once, and record it.
          family: plan-record-not-updated
          round: 5
        - id: BR-17
          severity: Important
          title: Two more near-identical helper pairs land in the same package
          detail: |-
            2nd in this family — do not fix only these instances. stripEscapes
            (highlightwriter_test.go:207) duplicates stripANSI (render_test.go:89) in
            the same package; onlyPhraseGap (highlightwriter.go:200) duplicates
            phraseGap (highlight.go:100), differing only on the empty string, so
            admitting another gap character in one and not the other silently drops a
            phrase. THE RULE: before adding a helper to package main, grep the package
            for one with the same job and extend it. M1 round 3 already applied this
            once (warnTo) and two fresh pairs landed in the next milestone, so run the
            mechanical sweep across cmd/define — production and test files both, since
            one of these pairs crosses that line.
          family: copy-pasted-helper
          round: 5
        - id: BR-18
          severity: Minor
          title: atlas says highlightWriter wraps crlfWriter in raw mode; that is M3
          detail: |-
            2nd in this family. atlas/define.md:490 states the crlfWriter wrapping in
            the present tense; it is Task 8 Step 5 and unchecked. The rule was recorded
            in lessons.md after M1 round 2 and a fresh instance landed anyway — sweep
            every present-tense architectural claim added to atlas/ and README in this
            window against the code at HEAD.
          family: atlas-claims-unbuilt-surface
          round: 5
        - id: BR-19
          severity: Minor
          title: cmd/define/zz_probe_test.go left untracked in the working tree
          detail: |-
            2nd in this family. Outside the reviewed commit, but a `git add -A` at
            close will commit it. THE RULE: a scratch probe is deleted in the turn that
            reads its output, never left for a later add to decide.
          family: dead-test-scaffolding
          round: 5
        - id: BR-20
          severity: Minor
          title: The no-data-loss invariant runs only over the colour-OFF render
          detail: |-
            TestHighlightingLosesNothing renders with Color:false, so sgrState.resume —
            the ANSI-nesting logic that is the whole reason definitions and answers
            share a mechanism — is never exercised over the real corpus, though
            production always feeds Color:opt.color. Verified the coloured version
            passes (32/32 entries), so this is a missing assertion, not a live bug. The
            same test also lacks a "hits > 0" guard; measured, only 6 of 32 corpus
            entries currently highlight, so a corpus refresh could leave it green and
            vacuous.
          family: behaviour-claimed-without-a-failing-test
          round: 5
        - id: BR-21
          severity: Minor
          title: The no-colour test asserts absence of green, not absence of escapes
          detail: |-
            highlightwriter_test.go:263 checks for "\x1b[1;32m" only. The plan's own M1
            Task 4 Step 2 rule says assert absence of "\x1b" entirely — "a test that
            only checks for green passes while emitting bold" — and with opt.color
            false the stronger assertion is free here.
          family: behaviour-claimed-without-a-failing-test
          round: 5
        - id: BR-22
          severity: Minor
          title: newHighlightWriter accepts on == "" and emits a bare reset around each match
          detail: |-
            highlightText guards v == nil || on == "" before constructing the writer,
            but the constructor does not, so a direct caller with an empty style emits
            sgrOff around every known span (highlightwriter.go:127). M3 wires the
            stream by constructing the writer directly and will not inherit that guard.
          family: one-absence-representation-per-seam
          round: 5
      boundary: M2
      blocked: true
---

# Gate ledger — tools#21 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T14:52:43-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `behaviour-claimed-without-a-failing-test` The editor loop's use of the vocabulary is pinned by no test — highlighting can be switched fully off and the suite stays green
  Two mutations survive the entire suite: replacing `voc` with `nil` at both
  RenderLine call sites (replraw.go:138, :216), and deleting voc.Load()
  (replraw.go:82). Nothing drives runEditor with a vocabulary and asserts a
  green byte reaches stdout, so the M1 Done-when has no proof. This is also
  Plan Task 4 Step 7's own stated test ("assert it highlights on the next
  render without a reload"), which was replaced by a capturer-to-set unit
  test. editorRig already supplies color:true and runEditor already takes a
  scripted key channel plus an out buffer, so the fix is ~10 lines.
- **BR-2** [Important] `behaviour-claimed-without-a-failing-test` Capture's "must not claim a word the deck rejected" is a comment with no test behind it
  capture.go:96-104 documents that vocab.Add runs only after Upsert
  succeeds, but moving the Add above the error check survives the whole
  suite. The fixture already exists — failingStore{} at capture_test.go:62 —
  so the pin costs four lines. Same rule as the editor-loop finding: sweep
  the diff for behaviours stated only in comments rather than fixing these
  two sites alone.
- **BR-3** [Minor] `behaviour-claimed-without-a-failing-test` storeVocabulary's once-only Load guard survives removal; the test asserting it cannot fail
  vocab_test.go:74 comments "second Load is free and must not double
  anything", but Add is idempotent and maxWords is a max, so deleting the
  `loaded` guard at vocab.go:107 keeps every assertion green. A counting
  store double would make the assertion bite.
- **BR-4** [Minor] `unformatted-source` cmd/define/capture.go is not gofmt-clean
  The new `vocab Vocabulary` field at capture.go:70 is misaligned; gofmt -l
  flags this file and no other in the package. No CI gate catches it.
- **BR-5** [Minor] `tokenizer-edge-admits-punctuation` isWordRune admits apostrophe and hyphen at token edges, so 'obsequious' and word--word never match
  highlight.go:25 treats ' and - as word runes unconditionally, so a quoted
  or double-dashed occurrence tokenizes into a run whose store.Key differs
  from the deck key. Harmless on the prompt line; definition bodies in M2
  routinely quote and dash. Relatedly, MaxPhraseWords counts with
  strings.Fields (vocab.go:70), which disagrees with wordRuns for keys
  holding other punctuation (e.g., 9/11) — such entries are unmatchable.
- **BR-6** [Minor] `atlas-claims-unbuilt-surface` atlas/define.md describes the definition path in the present tense before M2 builds it
  Line 444 says wordRuns is "the single tokenizer both the prompt and the
  definition path use". The M2/M3 forward references elsewhere in the same
  section are correctly marked as future; this one is not.
- **BR-7** [Minor] `plan-record-not-updated` All 26 M1 plan steps remain unchecked while the issue marks M1 complete
  workshop/plans/000021-highlight-learned-plan.md Tasks 1-4 are entirely
  `- [ ]` at HEAD. The plan also needs Revisions entries for Task 4 Step 2
  (its "assert absence of \x1b" instruction is infeasible — eraseLine and
  the cursor park are escapes) and Task 4 Step 7 (test not delivered).
- **BR-8** [Minor] `dead-test-scaffolding` `_ = unicode.IsLetter` in highlight_test.go:92 exists only to justify an unused import
  Drop the statement and the `unicode` import together.

## Round 2 — 2026-08-26T15:11:28-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-08-26T15:33:16-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Verified by mutation: nil at both RenderLine call sites and deleting voc.Load() each redden TestEditorLoopHighlightsADeckWordOnScreen.
- BR-2 — addressed — Hoisting vocab.Add above the Upsert error check reddens TestCaptureDoesNotAddAWordTheDeckRejected via the Upsert-only double.
- BR-3 — addressed — Removing the loaded guard reddens TestStoreVocabularyReadsTheDeckOnlyOnce; countingDeck counts Deck() reads.
- BR-4 — addressed — gofmt -l over cmd/ and internal/ reports nothing.
- BR-5 — addressed — appendTrimmed trims joiners at token edges; MaxPhraseWords counts with wordRuns; the punctuated-key limit is pinned by TestAPunctuatedKeyIsNotMatchable.
- BR-6 — addressed — atlas/define.md now reads "used by the prompt line today and by the definition and answer paths from M2".
- BR-7 — addressed — All 26 M1 steps are ticked and three Revisions entries record the Task 4 Step 2 infeasibility, the skipped Step 7, and the wordRuns contract change.
- BR-8 — addressed — The statement and the unicode import are both gone from highlight_test.go.

### Raised

- **BR-9** [Minor] `copy-pasted-helper` A third near-identical warnf, with the "define: " prefix now written in three places (ARCH-DRY)
  vocab.go:126 is a copy of history_store.go:82 minus its suffix, and capture.go:125
  is the same shape with a once-per-process flag. The shared part — the nil-writer
  guard and the "define: " prefix — should be one helper the three wrap; today
  renaming the program means finding three string literals.
- **BR-10** [Minor] `io-for-a-disabled-feature` voc.Load() reads the whole deck even with -no-color, where nothing can consume it
  replraw.go:82 loads unconditionally, but RenderLine only calls highlightSpans
  inside its `if color` branch, so with colour off the deck walk and YAML parse
  buy nothing. hist.Load() beside it is needed either way, which is what makes
  the unconditional shape look right.
- **BR-11** [Minor] `one-absence-representation-per-seam` The Vocabulary seam has two absent-representations and four guards, one of them already unreachable
  nil and an empty memVocabulary both mean "nothing highlighted", defended at
  highlight.go:113, replraw.go:79, main.go:134 and capture.go:102. run() calls
  withStore before repl(), so replraw.go:79 cannot fire in production. M2 adds two
  more consumers of the same seam; picking one representation now keeps this at one
  guard rather than six.
- **BR-12** [Minor] `feature-leaks-across-namespace` A deck word used as a command name highlights inside the command namespace
  Confirmed by direct render: with "history" in the deck, typing "/history 7" emits
  "\x1b[1m/\x1b[1;32mhistory\x1b[0m\x1b[1m 7". parseCommandLine's own comment says
  "/" opens a real separate namespace, not a marker on a word, so highlighting
  inside it reads as the vocabulary feature leaking across that boundary. It may be
  intended and harmless; nothing in the Spec's out-of-scope list decides it either way.

## Round 4 — 2026-08-26T15:52:11-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 5 — 2026-08-26T16:09:13-07:00 (claude) — BLOCKED

### Raised

- **BR-13** [Critical] `release-only-what-cannot-change` highlightWriter loses a match when a chunk splits on a joiner or mid-rune
  decidedEnd (cmd/define/highlightwriter.go:172) tests the bytes after the
  TRIMMED token end, so a trailing apostrophe/hyphen or an incomplete UTF-8
  sequence reads as "punctuation closed the token" and the region is
  released. Measured: don' + t, hot- + dog, caf\xc3 + \xa9 each highlight in
  one call and are lost when split, violating the writer's chunk-independence
  contract. FuzzHighlightWriterIsChunkIndependent cannot see the class — its
  vocabulary is the constant vocab("obsequious","hot dog","hot"), which holds
  no joiner-bearing or multi-byte entry, so no exec count reaches it. Fix the
  release test to require evidence the last token cannot grow, and derive the
  fuzz deck and seeds from TestWordRuns' own class table.
- **BR-14** [Critical] `behaviour-claimed-without-a-failing-test` Definitions never highlight on the one-shot or piped-stdin paths
  4th in this family — do not fix only this instance. Vocabulary.Load() has
  one call site, runEditor (cmd/define/replraw.go:84), but M2 wired
  highlighting into lookupAndRender (main.go:511), which is also reached by
  one-shot `define <word>` and by replLines. Verified: an unloaded
  storeVocabulary holding "obsequious" produces no highlight for
  "sycophantic"; calling Load() first produces one. 2 of 3 entry paths dead,
  including the one README's new sentence describes. THE RULE: the
  production-chain enumeration begins at the process entry points, not at the
  dependency the test injects — TestDefinitionBodyHighlightsADeckWord injects
  a pre-populated memVocabulary and so cannot see the Load hop, the same
  blindness as M1 round 2's withStore merge. The enumeration to write and
  sweep is one row per entry path reaching the render. Any fix must keep M1
  round 3's opt.color gate on Load.
- **BR-15** [Important] `feature-leaks-across-namespace` The headword line is re-styled, which the Spec lists as out of scope
  2nd in this family — do not fix only this instance. highlightText is
  applied to the whole rendered string (main.go:511), so a deck word looked
  up again renders its own headword green inside the bold cyan:
  "\x1b[1;36m\x1b[1;32msycophantic\x1b[0m\x1b[1;36m\x1b[0m". The Spec's
  out-of-scope list says "Re-styling the headword line". Nothing tests it
  either way. THE RULE: the vocabulary is withheld per region by an explicit
  decision at the boundary — the shape highlightSetFor already has for the
  command namespace — not by wrapping whatever string is at hand. Enumerate
  every region a renderer produces (headword, syllabification, pronunciation,
  part-of-speech, sense body, examples, and for M3 the answer stream), state
  admit or withhold for each, and pin the withholds.
- **BR-16** [Important] `plan-record-not-updated` Plan file layout, contract rule 4 and two test names no longer match the code
  2nd in this family — do not fix only this instance. Six divergences:
  sgrState and highlightWriter are in sgr.go/highlightwriter.go not
  highlight.go; Task 5/6 Files blocks name the wrong files; contract rule 4
  promises (n, err) in caller units while Write returns (0, err) and the test
  asserts n==0 — the opposite — and crlfWriter in the same package returns
  caller-unit progress with a test defending exactly that; the fuzz target
  was renamed; Task 7 Step 2's invariant landed as a different property in a
  different file. THE RULE: a deviation from the plan's stated layout or
  contract lands its "## Revisions" entry in the same commit as the
  deviation. Decide the Write count question once, and record it.
- **BR-17** [Important] `copy-pasted-helper` Two more near-identical helper pairs land in the same package
  2nd in this family — do not fix only these instances. stripEscapes
  (highlightwriter_test.go:207) duplicates stripANSI (render_test.go:89) in
  the same package; onlyPhraseGap (highlightwriter.go:200) duplicates
  phraseGap (highlight.go:100), differing only on the empty string, so
  admitting another gap character in one and not the other silently drops a
  phrase. THE RULE: before adding a helper to package main, grep the package
  for one with the same job and extend it. M1 round 3 already applied this
  once (warnTo) and two fresh pairs landed in the next milestone, so run the
  mechanical sweep across cmd/define — production and test files both, since
  one of these pairs crosses that line.
- **BR-18** [Minor] `atlas-claims-unbuilt-surface` atlas says highlightWriter wraps crlfWriter in raw mode; that is M3
  2nd in this family. atlas/define.md:490 states the crlfWriter wrapping in
  the present tense; it is Task 8 Step 5 and unchecked. The rule was recorded
  in lessons.md after M1 round 2 and a fresh instance landed anyway — sweep
  every present-tense architectural claim added to atlas/ and README in this
  window against the code at HEAD.
- **BR-19** [Minor] `dead-test-scaffolding` cmd/define/zz_probe_test.go left untracked in the working tree
  2nd in this family. Outside the reviewed commit, but a `git add -A` at
  close will commit it. THE RULE: a scratch probe is deleted in the turn that
  reads its output, never left for a later add to decide.
- **BR-20** [Minor] `behaviour-claimed-without-a-failing-test` The no-data-loss invariant runs only over the colour-OFF render
  TestHighlightingLosesNothing renders with Color:false, so sgrState.resume —
  the ANSI-nesting logic that is the whole reason definitions and answers
  share a mechanism — is never exercised over the real corpus, though
  production always feeds Color:opt.color. Verified the coloured version
  passes (32/32 entries), so this is a missing assertion, not a live bug. The
  same test also lacks a "hits > 0" guard; measured, only 6 of 32 corpus
  entries currently highlight, so a corpus refresh could leave it green and
  vacuous.
- **BR-21** [Minor] `behaviour-claimed-without-a-failing-test` The no-colour test asserts absence of green, not absence of escapes
  highlightwriter_test.go:263 checks for "\x1b[1;32m" only. The plan's own M1
  Task 4 Step 2 rule says assert absence of "\x1b" entirely — "a test that
  only checks for green passes while emitting bold" — and with opt.color
  false the stronger assertion is free here.
- **BR-22** [Minor] `one-absence-representation-per-seam` newHighlightWriter accepts on == "" and emits a bare reset around each match
  highlightText guards v == nil || on == "" before constructing the writer,
  but the constructor does not, so a direct caller with an empty style emits
  sgrOff around every known span (highlightwriter.go:127). M3 wires the
  stream by constructing the writer directly and will not inherit that guard.

## Open findings

- **BR-9** [Minor] `copy-pasted-helper` A third near-identical warnf, with the "define: " prefix now written in three places (ARCH-DRY)
- **BR-10** [Minor] `io-for-a-disabled-feature` voc.Load() reads the whole deck even with -no-color, where nothing can consume it
- **BR-11** [Minor] `one-absence-representation-per-seam` The Vocabulary seam has two absent-representations and four guards, one of them already unreachable
- **BR-12** [Minor] `feature-leaks-across-namespace` A deck word used as a command name highlights inside the command namespace
- **BR-13** [Critical] `release-only-what-cannot-change` highlightWriter loses a match when a chunk splits on a joiner or mid-rune
- **BR-14** [Critical] `behaviour-claimed-without-a-failing-test` Definitions never highlight on the one-shot or piped-stdin paths
- **BR-15** [Important] `feature-leaks-across-namespace` The headword line is re-styled, which the Spec lists as out of scope
- **BR-16** [Important] `plan-record-not-updated` Plan file layout, contract rule 4 and two test names no longer match the code
- **BR-17** [Important] `copy-pasted-helper` Two more near-identical helper pairs land in the same package
- **BR-18** [Minor] `atlas-claims-unbuilt-surface` atlas says highlightWriter wraps crlfWriter in raw mode; that is M3
- **BR-19** [Minor] `dead-test-scaffolding` cmd/define/zz_probe_test.go left untracked in the working tree
- **BR-20** [Minor] `behaviour-claimed-without-a-failing-test` The no-data-loss invariant runs only over the colour-OFF render
- **BR-21** [Minor] `behaviour-claimed-without-a-failing-test` The no-colour test asserts absence of green, not absence of escapes
- **BR-22** [Minor] `one-absence-representation-per-seam` newHighlightWriter accepts on == "" and emits a bare reset around each match
