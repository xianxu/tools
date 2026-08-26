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

## Open findings

- **BR-9** [Minor] `copy-pasted-helper` A third near-identical warnf, with the "define: " prefix now written in three places (ARCH-DRY)
- **BR-10** [Minor] `io-for-a-disabled-feature` voc.Load() reads the whole deck even with -no-color, where nothing can consume it
- **BR-11** [Minor] `one-absence-representation-per-seam` The Vocabulary seam has two absent-representations and four guards, one of them already unreachable
- **BR-12** [Minor] `feature-leaks-across-namespace` A deck word used as a command name highlights inside the command namespace
