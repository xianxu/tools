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

## Open findings

- **BR-1** [Important] `behaviour-claimed-without-a-failing-test` The editor loop's use of the vocabulary is pinned by no test — highlighting can be switched fully off and the suite stays green
- **BR-2** [Important] `behaviour-claimed-without-a-failing-test` Capture's "must not claim a word the deck rejected" is a comment with no test behind it
- **BR-3** [Minor] `behaviour-claimed-without-a-failing-test` storeVocabulary's once-only Load guard survives removal; the test asserting it cannot fail
- **BR-4** [Minor] `unformatted-source` cmd/define/capture.go is not gofmt-clean
- **BR-5** [Minor] `tokenizer-edge-admits-punctuation` isWordRune admits apostrophe and hyphen at token edges, so 'obsequious' and word--word never match
- **BR-6** [Minor] `atlas-claims-unbuilt-surface` atlas/define.md describes the definition path in the present tense before M2 builds it
- **BR-7** [Minor] `plan-record-not-updated` All 26 M1 plan steps remain unchecked while the issue marks M1 complete
- **BR-8** [Minor] `dead-test-scaffolding` `_ = unicode.IsLetter` in highlight_test.go:92 exists only to justify an unused import
