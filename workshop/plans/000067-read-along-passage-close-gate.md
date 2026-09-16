---
gate: boundary-review
issue: 67
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-16T15:27:51-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: An unterminated ESC[200~ permanently deafens the input path, swallowing Enter and Ctrl-C
          detail: |-
            Verified through the real readKeys/readInput goroutine: feeding "\x1b[200~" + 1050 'x'
            + "hello\r\x03" yields two inert KeyUnknown keys and nothing else — the carriage return
            and the interrupt never emerge. Under the cap scan returns used==0 forever so readInput
            never advances buf (selection_input.go:178) and later keystrokes join the same buffer;
            over the cap `draining` latches (paste.go:90) and every byte is discarded until a closer
            that never arrives. Raw mode disables ISIG, so Ctrl-C is only reachable via a decoded
            KeyInterrupt and the program cannot be quit from the keyboard. A regression against the
            base, where ESC[200~ was an inert 6-byte KeyUnknown. The plan (Task 1.2b item 1) named
            this case and promised TestAnUnterminatedPasteRecoversAtTheCap; neither the bound, the
            test, nor the atlas note shipped. ARCH-SECURE, ARCH-ORDER.
          family: external-state-needs-bounded-exit
          round: 1
        - id: BR-2
          severity: Critical
          title: Suite is red at HEAD — render.go still names the test this window renamed
          detail: |-
            TestARemovedDeclarationIsSweptOrRetired fails at c1844b3 in a clean checkout:
            cmd/define/render.go:284 names TestEveryEnabledMouseModeIsDecoded, removed by this
            window and absent from retiredSymbolNames. The guard SKIPS at the base commit, so this
            window introduced the failure, and the issue Log's "full suite green" claim does not
            hold. doc_sync_test.go:390 and :427 carry the same stale name, and key_test.go:399's doc
            comment still says "Adding a mode to mouseOn" after the constant became mouseOn+pasteOn.
          family: removed-symbol-unswept
          round: 1
        - id: BR-3
          severity: Important
          title: readInput's long-lived keyDecoder has no test — reverting it leaves the whole suite green
          detail: |-
            In a scratch worktree at c1844b3 I replaced dec.decode(buf) with decodeKey(buf) at
            selection_input.go:178, reverting the stateful wiring this milestone added. The full
            cmd/define suite produced the identical three failures and zero new ones. That revert
            reintroduces exactly what the drain exists to prevent: with a fresh decoder per call the
            held-back tail and every later chunk of an over-cap paste decode as KeyRune. Every paste
            test drives scan/decode directly and simulates re-presentation by hand. The seam already
            exists and is already used by TestReadKeysHandlesSequencesSplitAcrossReads
            (rawterm_test.go:50); add a paste-split-across-reads row and an oversize-paste row there.
            ARCH-ORDER — the oracle observes only the interleaving the author constructed.
          family: production-seam-untested
          round: 1
        - id: BR-4
          severity: Important
          title: The durable plan contradicts the shipped code and carries no Revisions entry
          detail: |-
            workshop/plans/000067-read-along-passage-plan.md has all M1 checkboxes (Tasks 1.1-1.5,
            lines 123-469) still unticked while the issue marks M1 done, and no "## Revisions"
            section at all (AGENTS.md section 1). Its prose describes code that does not exist: Task
            1.2b states "newPassage is the parse boundary" and its tests call newPassage(), but the
            boundary shipped as sanitisePasteBody (paste.go:109); Task 1.4 Step 3 specifies Apply
            returns ActNone for KeyPaste and that runEditor intercepts both kinds, but editor.go:58
            inserts the paste and replraw.go:534 intercepts only KeyPasteRefused. The Pure entities
            table also omits sanitisePasteBody, pasteLineRunes and the keyDecoder type. M2 will be
            read off this plan.
          family: plan-artifact-stale
          round: 1
        - id: BR-5
          severity: Important
          title: cmd/define/README.md not updated for the paste behaviour or the 1000-character refusal
          detail: |-
            That README documents interactive input in detail (Keys table at :113-119, "Select and
            copy text" at :63-74, including the type-ahead-full notice). This window changes what a
            user sees: a multi-line paste now lands as ONE line with newlines and tabs turned into
            spaces (paste.go:143) instead of submitting mid-paste, and a paste over 1000 characters
            is refused with "define: that paste is longer than 1000 characters; paste less"
            (replraw.go:539). Neither appears in the README. atlas/define.md covers the mechanism
            well but also omits pasteLineRunes — the flattening is the one behaviour a user hits and
            it is the documented deviation from the plan.
          family: readme-surface-undocumented
          round: 1
        - id: BR-6
          severity: Important
          title: No test for a paste arriving during a live drag, which the plan required as a decision
          detail: |-
            Task 1.4's preamble: a KeyPaste reaches pointerRouter.route (selection_input.go:28) and,
            being non-pointer and non-KeyUnknown, hits cancelPointerInput(l, k, true) at :47 —
            cancelling a live drag — and "it must be a decision with a test, not an inherited side
            effect." Step 1 lists the test; nothing in the diff exercises it. Same class as the
            missing unterminated-paste pin.
          family: decided-behaviour-unpinned
          round: 1
        - id: BR-7
          severity: Minor
          title: sanitisePasteBody drops Cc controls but lets bidi/format controls through
          detail: |-
            unicode.IsControl (paste.go:128) is category Cc only, so U+202E RLO, ZWJ and U+200B
            survive the boundary. practice_help.go:140 already treats unicode.Bidi_Control as a
            distinct hazard for terminal-bound text. Decide before M2 puts the passage on screen and
            M4 sends it to the model. ARCH-SECURE.
          family: boundary-parses-partial-class
          round: 1
        - id: BR-8
          severity: Minor
          title: Key struct doc still says Raw is an unmodelled sequence to be ignored, not inserted
          detail: |-
            key.go:66. For KeyPaste, Raw is sanitised text that IS inserted (editor.go:64). The
            field now carries two trust levels and the doc asserts one; the KeyPaste const comment
            says the right thing but the struct doc above it does not.
          family: doc-contradicts-type
          round: 1
        - id: BR-9
          severity: Minor
          title: TestTheBoundaryKeepsNewlinesAndDropsOtherControls accepts either outcome, pinning neither
          detail: |-
            paste_test.go:163 passes for both "a\nb\tc de" and "a\nb\tcde", so NUL/BEL handling is
            unpinned. Pick one. Relatedly, TestTheDrainDoesNotCutAStraddlingCloser at :134 hands the
            second scan a fresh buffer rather than the leftover the caller would re-present.
          family: test-accepts-two-outcomes
          round: 1
        - id: BR-10
          severity: Minor
          title: The issue's M1 row says "1000-byte cap"; the code, plan and atlas all say runes
          detail: |-
            workshop/issues/000067-read-along-passage.md, M1 checkbox. Rune-vs-byte is the argued
            point of the cap, so the tracker row inverts it.
          family: issue-row-stale
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — tools#67 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-16T15:27:51-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `external-state-needs-bounded-exit` An unterminated ESC[200~ permanently deafens the input path, swallowing Enter and Ctrl-C
  Verified through the real readKeys/readInput goroutine: feeding "\x1b[200~" + 1050 'x'
  + "hello\r\x03" yields two inert KeyUnknown keys and nothing else — the carriage return
  and the interrupt never emerge. Under the cap scan returns used==0 forever so readInput
  never advances buf (selection_input.go:178) and later keystrokes join the same buffer;
  over the cap `draining` latches (paste.go:90) and every byte is discarded until a closer
  that never arrives. Raw mode disables ISIG, so Ctrl-C is only reachable via a decoded
  KeyInterrupt and the program cannot be quit from the keyboard. A regression against the
  base, where ESC[200~ was an inert 6-byte KeyUnknown. The plan (Task 1.2b item 1) named
  this case and promised TestAnUnterminatedPasteRecoversAtTheCap; neither the bound, the
  test, nor the atlas note shipped. ARCH-SECURE, ARCH-ORDER.
- **BR-2** [Critical] `removed-symbol-unswept` Suite is red at HEAD — render.go still names the test this window renamed
  TestARemovedDeclarationIsSweptOrRetired fails at c1844b3 in a clean checkout:
  cmd/define/render.go:284 names TestEveryEnabledMouseModeIsDecoded, removed by this
  window and absent from retiredSymbolNames. The guard SKIPS at the base commit, so this
  window introduced the failure, and the issue Log's "full suite green" claim does not
  hold. doc_sync_test.go:390 and :427 carry the same stale name, and key_test.go:399's doc
  comment still says "Adding a mode to mouseOn" after the constant became mouseOn+pasteOn.
- **BR-3** [Important] `production-seam-untested` readInput's long-lived keyDecoder has no test — reverting it leaves the whole suite green
  In a scratch worktree at c1844b3 I replaced dec.decode(buf) with decodeKey(buf) at
  selection_input.go:178, reverting the stateful wiring this milestone added. The full
  cmd/define suite produced the identical three failures and zero new ones. That revert
  reintroduces exactly what the drain exists to prevent: with a fresh decoder per call the
  held-back tail and every later chunk of an over-cap paste decode as KeyRune. Every paste
  test drives scan/decode directly and simulates re-presentation by hand. The seam already
  exists and is already used by TestReadKeysHandlesSequencesSplitAcrossReads
  (rawterm_test.go:50); add a paste-split-across-reads row and an oversize-paste row there.
  ARCH-ORDER — the oracle observes only the interleaving the author constructed.
- **BR-4** [Important] `plan-artifact-stale` The durable plan contradicts the shipped code and carries no Revisions entry
  workshop/plans/000067-read-along-passage-plan.md has all M1 checkboxes (Tasks 1.1-1.5,
  lines 123-469) still unticked while the issue marks M1 done, and no "## Revisions"
  section at all (AGENTS.md section 1). Its prose describes code that does not exist: Task
  1.2b states "newPassage is the parse boundary" and its tests call newPassage(), but the
  boundary shipped as sanitisePasteBody (paste.go:109); Task 1.4 Step 3 specifies Apply
  returns ActNone for KeyPaste and that runEditor intercepts both kinds, but editor.go:58
  inserts the paste and replraw.go:534 intercepts only KeyPasteRefused. The Pure entities
  table also omits sanitisePasteBody, pasteLineRunes and the keyDecoder type. M2 will be
  read off this plan.
- **BR-5** [Important] `readme-surface-undocumented` cmd/define/README.md not updated for the paste behaviour or the 1000-character refusal
  That README documents interactive input in detail (Keys table at :113-119, "Select and
  copy text" at :63-74, including the type-ahead-full notice). This window changes what a
  user sees: a multi-line paste now lands as ONE line with newlines and tabs turned into
  spaces (paste.go:143) instead of submitting mid-paste, and a paste over 1000 characters
  is refused with "define: that paste is longer than 1000 characters; paste less"
  (replraw.go:539). Neither appears in the README. atlas/define.md covers the mechanism
  well but also omits pasteLineRunes — the flattening is the one behaviour a user hits and
  it is the documented deviation from the plan.
- **BR-6** [Important] `decided-behaviour-unpinned` No test for a paste arriving during a live drag, which the plan required as a decision
  Task 1.4's preamble: a KeyPaste reaches pointerRouter.route (selection_input.go:28) and,
  being non-pointer and non-KeyUnknown, hits cancelPointerInput(l, k, true) at :47 —
  cancelling a live drag — and "it must be a decision with a test, not an inherited side
  effect." Step 1 lists the test; nothing in the diff exercises it. Same class as the
  missing unterminated-paste pin.
- **BR-7** [Minor] `boundary-parses-partial-class` sanitisePasteBody drops Cc controls but lets bidi/format controls through
  unicode.IsControl (paste.go:128) is category Cc only, so U+202E RLO, ZWJ and U+200B
  survive the boundary. practice_help.go:140 already treats unicode.Bidi_Control as a
  distinct hazard for terminal-bound text. Decide before M2 puts the passage on screen and
  M4 sends it to the model. ARCH-SECURE.
- **BR-8** [Minor] `doc-contradicts-type` Key struct doc still says Raw is an unmodelled sequence to be ignored, not inserted
  key.go:66. For KeyPaste, Raw is sanitised text that IS inserted (editor.go:64). The
  field now carries two trust levels and the doc asserts one; the KeyPaste const comment
  says the right thing but the struct doc above it does not.
- **BR-9** [Minor] `test-accepts-two-outcomes` TestTheBoundaryKeepsNewlinesAndDropsOtherControls accepts either outcome, pinning neither
  paste_test.go:163 passes for both "a\nb\tc de" and "a\nb\tcde", so NUL/BEL handling is
  unpinned. Pick one. Relatedly, TestTheDrainDoesNotCutAStraddlingCloser at :134 hands the
  second scan a fresh buffer rather than the leftover the caller would re-present.
- **BR-10** [Minor] `issue-row-stale` The issue's M1 row says "1000-byte cap"; the code, plan and atlas all say runes
  workshop/issues/000067-read-along-passage.md, M1 checkbox. Rune-vs-byte is the argued
  point of the cap, so the tracker row inverts it.

## Open findings

- **BR-1** [Critical] `external-state-needs-bounded-exit` An unterminated ESC[200~ permanently deafens the input path, swallowing Enter and Ctrl-C
- **BR-2** [Critical] `removed-symbol-unswept` Suite is red at HEAD — render.go still names the test this window renamed
- **BR-3** [Important] `production-seam-untested` readInput's long-lived keyDecoder has no test — reverting it leaves the whole suite green
- **BR-4** [Important] `plan-artifact-stale` The durable plan contradicts the shipped code and carries no Revisions entry
- **BR-5** [Important] `readme-surface-undocumented` cmd/define/README.md not updated for the paste behaviour or the 1000-character refusal
- **BR-6** [Important] `decided-behaviour-unpinned` No test for a paste arriving during a live drag, which the plan required as a decision
- **BR-7** [Minor] `boundary-parses-partial-class` sanitisePasteBody drops Cc controls but lets bidi/format controls through
- **BR-8** [Minor] `doc-contradicts-type` Key struct doc still says Raw is an unmodelled sequence to be ignored, not inserted
- **BR-9** [Minor] `test-accepts-two-outcomes` TestTheBoundaryKeepsNewlinesAndDropsOtherControls accepts either outcome, pinning neither
- **BR-10** [Minor] `issue-row-stale` The issue's M1 row says "1000-byte cap"; the code, plan and atlas all say runes
