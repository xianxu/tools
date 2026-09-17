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
    - "n": 2
      timestamp: "2026-09-16T15:55:32-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: stubbing indexPasteAbandon to never match reddens both rows of TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt. Residual, acceptable: a quiet unterminated paste still swallows Enter until Ctrl-C/Ctrl-D, which is now the documented exit.'
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: 'Suite is green at HEAD, but 3 of the 4 named sites remain: key_test.go:399 still says "Adding a mode to mouseOn", doc_sync_test.go:390 and :427 still name TestEveryEnabledMouseModeIsDecoded. currentTruthFiles (repo_guard_test.go:1738) excludes *_test.go, so no guard can ever see them.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Mutation-verified: replacing dec.decode(buf) with decodeKey(buf) at selection_input.go:178 reddens TestReadKeysCarriesTheDrainAcrossReads with "a drained body byte reached the line as the rune ''x''".'
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: The Revisions section landed and spansToStyled is gone, but every M1 checkbox (Tasks 1.1-1.5, including the atlas item) is still unticked, and the Pure entities table still omits sanitisePasteBody, pasteLineRunes, indexPasteAbandon and keyDecoder.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: The cap is now documented, but README.md:80-83 describes M2 routing that does not exist at HEAD, and the behaviour BR-5 actually named — pasteLineRunes flattening newlines and tabs to spaces — is still absent from both README and atlas, as is the new abandon rule.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: 'TestAPasteCancelsALiveDrag calls cancelPointerInput directly, which cancels for every kind but KeyUnknown. Mutation-verified: adding "&& k.Kind != KeyPaste" to route''s condition at selection_input.go:47 leaves the suite green. Assert through pointerRouter.route.'
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: paste.go:128 is still unicode.IsControl (category Cc only); no decision recorded in the plan's Revisions or the atlas.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: key.go:66 is byte-identical to the base; the struct doc still says Raw is an unmodelled sequence to be ignored.
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: 'First half fixed — TestTheBoundaryKeepsNewlinesAndDropsOtherControls now pins "a\nb\tcde" exactly. Second half open: TestTheDrainDoesNotCutAStraddlingCloser (paste_test.go:134) still hands each scan a fresh buffer instead of the leftover the caller re-presents.'
          round: 2
        - id: BR-10
          disposition: addressed
          note: The M1 row now reads "the 1000-RUNE cap (runes, not bytes ...)", matching paste.go, the plan and the atlas.
          round: 2
      findings:
        - id: BR-11
          severity: Important
          title: Prose committed in this window asserts M2 behaviour and names symbols the tree does not declare
          detail: |-
            This is the 2nd finding in family `doc-contradicts-type` (BR-8 is still open as the
            1st), so do NOT fix these instances alone. The rule that covers all of them: prose
            committed in a window may describe only what that window's tree contains — every
            identifier it names must be declared at HEAD, and every behavioural claim must have a
            test. Measured prevalence in this window, 3 forward-reference sites plus the 3
            backward ones BR-2 left: README.md:80-83 claims a 4+-word or multi-line paste "becomes
            the passage" (pasteIsPassage is not declared at HEAD; every paste goes into the line);
            paste_test.go:344-346 cites pasteIsPassage and TestThePassageSurvivesALookup, neither
            declared; paste.go:126-131 justifies the boundary by "the footer" and "the mark
            painting", which M3 builds. The enumeration is mechanical and the repo already owns
            half of it — TestPlanCitesTestsThatExist and TestPlanTablesNameEntitiesThatExist do
            exactly this derivation for plan files, and TestNoArtifactNamesARetiredSymbol does the
            backward half for currentTruthFiles. Extending that derivation to README, atlas and Go
            doc comments (and dropping the *_test.go exemption for Test* names, which is what hid
            BR-2's residual) closes both directions at once. ARCH-PURPOSE — the class, not the site.
          family: doc-contradicts-type
          round: 2
        - id: BR-12
          severity: Important
          title: The rune cap is evaluated on a possibly-truncated buffer, so a legal 1000-rune CJK paste is refused
          detail: |-
            paste.go:97 decides drain-vs-wait with utf8.RuneCount over a body that may end mid-rune.
            Reproduced at HEAD: a 1000-rune CJK paste split so the first scan sees 2999 body bytes
            counts 1001 (999 complete runes plus 2 orphan bytes each counted as RuneError), latches
            draining, and the closer then returns KeyPasteRefused for a paste that fits — a false
            refusal on exactly the decks /lang exists for, which is the argued point of the rune
            cap. The structural cause is that one predicate serves two purposes: the no-closer
            branch is a MEMORY bound and belongs in bytes (len(body) > maxPasteRunes*utf8.UTFMax),
            while the SEMANTIC cap belongs only where the text is complete, at the closer where it
            already runs. TestPasteScannerCapsInRunesNotBytes uses a single unsplit scan, so it
            cannot see this even though paste.go:41 documents multi-read as the normal path; the
            regression row is a cap-sized CJK paste split at body length 2999. ARCH-CONSTRAINTS.
          family: decision-on-incomplete-input
          round: 2
        - id: BR-13
          severity: Minor
          title: enterPaste/leavePaste is the third copy of the same terminal-mode pair, and restore's ordering is still undeclared
          detail: |-
            rawterm.go:105-192 now holds three near-identical enter/leave pairs differing only in
            (flag field, on-string, off-string), and rawSession carries three independent bools —
            8 representable combinations for about 4 legal ones. A single ordered table of
            {flag, on, off} collapses the duplication (ARCH-DRY) and, more usefully, makes
            restore()'s teardown ORDER a declared list rather than three hand-written calls;
            that order is the thing rawterm_test.go:144-151 asserts and the one thing a fourth
            mode's author will not see in the enterPaste template they copy (ARCH-ORDER).
          family: repeated-shape-not-extracted
          round: 2
        - id: BR-14
          severity: Minor
          title: The issue Log records no boundary-review round and its "full suite green" claim was false at the commit it describes
          detail: |-
            This is the 2nd finding in family `issue-row-stale`, so state the rule rather than
            patching the line: every claim in the issue that asserts a property of the tree is
            verification evidence and must be re-stated at the boundary that re-verified it.
            The "M1 implemented" entry (issue:937) claims "full suite green" for c1844b3, where
            BR-2 proved the suite was red; it is uncorrected, and there is no ## Log entry for
            boundary-review round 1 at all, which AGENTS.md section 3 requires alongside the
            Review-Verdict trailer. Prevalence in this issue: 2 of 2 tree-asserting claims were
            wrong at some point (the M1 checkbox's "1000-byte cap", fixed as BR-10, and this one).
            The enumeration is short — the Log's verification claims, the Plan checkboxes, and the
            Estimate block's actuals — and appending one Log entry per gate round covers it.
          family: issue-row-stale
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-09-16T16:47:28-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: Suite green at HEAD and the guard RUNS (--- PASS, 0.20s, not SKIP); render.go:284 now names TestEveryEnabledInputModeIsDecoded and the three test-file sites are swept — no occurrence of the old name remains anywhere under cmd/ or atlas/.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: The Revisions section and the four missing Pure-entities rows landed, but every M1 task checkbox (Tasks 1.1-1.5, plan lines 126-472) is still unticked — 0 ticked boxes in the whole file — which is exactly what round 2 named.
          round: 3
        - id: BR-5
          disposition: addressed
          note: README.md:76-92 now documents the single-insertion behaviour, newline/tab flattening, the 1000-character refusal, escape stripping and the abandon rule; atlas/define.md covers pasteLineRunes. Prose-only repair verified against the pinned diff and the behaviour's existing tests.
          round: 3
        - id: BR-6
          disposition: addressed
          note: 'Mutation-verified here: adding "&& k.Kind != KeyPaste" to route''s condition at selection_input.go:48 reddens TestAPasteCancelsALiveDrag (paste_test.go:505). The test now drives pointerRouter.route, not cancelPointerInput.'
          round: 3
        - id: BR-7
          disposition: addressed
          note: 'The decision is recorded at the site (paste.go:155-160): Cc removed because it is what a terminal acts on, Cf deliberately kept because stripping it would alter words in scripts that need it. No behaviour change, so no regression test is owed — though nothing pins that Cf survives.'
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: key.go:66-68 is still byte-identical to the base; the struct doc says Raw is an unmodelled sequence to be ignored, while editor.go:63 inserts it for KeyPaste.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: 'First half stayed fixed. Second half open: TestTheDrainDoesNotCutAStraddlingCloser (paste_test.go:136-149) still hands each scan a fresh buffer rather than the leftover readInput would re-present.'
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: 'README and atlas were repaired, but 3 sites remain — paste.go:145-148 and paste_test.go:150-154 still say the body is "bound for the footer" and would "defeat the mark painting" (it goes into the LINE at HEAD), and paste_test.go:348-349 still cites pasteIsPassage and TestThePassageSurvivesALookup in the past tense. The class fix was not written: repo_guard_test.go is byte-identical across the entire window, so currentTruthFiles (:1732-1746) still exempts *_test.go and no forward-direction guard exists.'
          round: 3
        - id: BR-12
          disposition: addressed
          note: 'Mutation-verified here: restoring the single "utf8.RuneCount(body) <= maxPasteRunes" predicate reddens TestALegalCJKPasteIsNotRefusedAtAnySplit at split 3011. The split into maxPasteRunes/maxPasteBytes is the structural fix the finding asked for.'
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: rawterm.go:96-198 is unchanged since c1844b3 — still three hand-written enter/leave pairs, three independent bools, and a hand-written teardown order in restore(). I-2 below approaches the same shape from the test side.
          round: 3
        - id: BR-14
          disposition: not-addressed
          note: 'The issue file changed only by b801efb (one checkbox). There is still no ## Log entry for either boundary-review round, and issue:937 still claims "full suite green" for c1844b3, where BR-2 proved it red.'
          round: 3
      findings:
        - id: BR-15
          severity: Important
          title: A paste during a review sitting is silently dropped — toInput has no case for KeyPaste, and this window turned mode 2004 on for that surface too
          detail: 'newConsole calls sess.enterPaste() (replraw.go:89) and BOTH sitting paths run through it: --play via runPlay (play_loop.go:116) and /play via sittingInPlace, which borrows the REPL''s keys channel (replraw.go:620). toInput (play_loop.go:565) has no case for KeyPaste or KeyPasteRefused — I confirmed it returns ok=false for both — and play_loop.go:370 continues on !ok. Before this window 2004 was off and a pasted answer arrived as KeyRunes and typed; now it vanishes with no message, and a refused paste produces no notice either because the report lives in runEditor (replraw.go:534), which is not running during a sitting. No test covers it. The durable fix is the guard, not the case: KeyKind has no sentinel (key.go:12-63), so nothing forced the question. The repo already owns the move at render.go:281 (numRegionKinds plus the guards derived from it).'
          family: enum-grows-past-consumers
          round: 3
        - id: BR-16
          severity: Important
          title: sess.enterPaste() is production wiring no test exercises, and mode 2004 has no live conformance row while the mouse does
          detail: 'This is the 2nd finding in family `production-seam-untested` (BR-3 was the 1st), so do not fix this site alone. Evidence: deleting replraw.go:89 leaves `go test ./cmd/define/ -run ''Paste|Raw|Editor|Console|Repl|Input''` green (ok, 15.9s) — with it gone the terminal never brackets and the whole milestone silently reverts to a pasted newline submitting mid-paste. rawterm_test.go:119-152 calls r.enterPaste() itself, so it pins the METHOD and never the call site, and it asserts into a bytes.Buffer, so it cannot show the sequence reaching a terminal. pty_conformance_test.go:664 is the exact missing precedent, and rawterm.go:163-166 states paste carries the identical no-reset-reflex hazard. The rule that covers the class - every terminal mode the program enables is asserted at the place it is ENABLED and given back on a real terminal, DERIVED from the enable-constant set rather than hand-written per mode. The enumeration is already owned by TestEveryEnabledInputModeIsDecoded: altScreenOn, mouseOn, pasteOn. Sweep it in this round - one in-process assertion per constant that newConsole writes it, one PTY row per constant that it reaches and leaves a real terminal. TestRestoreHandsBackEveryTerminalState hand-writes all three today, which is BR-13''s duplication seen from the test side. ARCH-MOCK, ARCH-DRY.'
          family: production-seam-untested
          round: 3
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

## Round 2 — 2026-09-16T15:55:32-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: stubbing indexPasteAbandon to never match reddens both rows of TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt. Residual, acceptable: a quiet unterminated paste still swallows Enter until Ctrl-C/Ctrl-D, which is now the documented exit.
- BR-2 — not-addressed — Suite is green at HEAD, but 3 of the 4 named sites remain: key_test.go:399 still says "Adding a mode to mouseOn", doc_sync_test.go:390 and :427 still name TestEveryEnabledMouseModeIsDecoded. currentTruthFiles (repo_guard_test.go:1738) excludes *_test.go, so no guard can ever see them.
- BR-3 — addressed — Mutation-verified: replacing dec.decode(buf) with decodeKey(buf) at selection_input.go:178 reddens TestReadKeysCarriesTheDrainAcrossReads with "a drained body byte reached the line as the rune 'x'".
- BR-4 — not-addressed — The Revisions section landed and spansToStyled is gone, but every M1 checkbox (Tasks 1.1-1.5, including the atlas item) is still unticked, and the Pure entities table still omits sanitisePasteBody, pasteLineRunes, indexPasteAbandon and keyDecoder.
- BR-5 — not-addressed — The cap is now documented, but README.md:80-83 describes M2 routing that does not exist at HEAD, and the behaviour BR-5 actually named — pasteLineRunes flattening newlines and tabs to spaces — is still absent from both README and atlas, as is the new abandon rule.
- BR-6 — not-addressed — TestAPasteCancelsALiveDrag calls cancelPointerInput directly, which cancels for every kind but KeyUnknown. Mutation-verified: adding "&& k.Kind != KeyPaste" to route's condition at selection_input.go:47 leaves the suite green. Assert through pointerRouter.route.
- BR-7 — not-addressed — paste.go:128 is still unicode.IsControl (category Cc only); no decision recorded in the plan's Revisions or the atlas.
- BR-8 — not-addressed — key.go:66 is byte-identical to the base; the struct doc still says Raw is an unmodelled sequence to be ignored.
- BR-9 — not-addressed — First half fixed — TestTheBoundaryKeepsNewlinesAndDropsOtherControls now pins "a\nb\tcde" exactly. Second half open: TestTheDrainDoesNotCutAStraddlingCloser (paste_test.go:134) still hands each scan a fresh buffer instead of the leftover the caller re-presents.
- BR-10 — addressed — The M1 row now reads "the 1000-RUNE cap (runes, not bytes ...)", matching paste.go, the plan and the atlas.

### Raised

- **BR-11** [Important] `doc-contradicts-type` Prose committed in this window asserts M2 behaviour and names symbols the tree does not declare
  This is the 2nd finding in family `doc-contradicts-type` (BR-8 is still open as the
  1st), so do NOT fix these instances alone. The rule that covers all of them: prose
  committed in a window may describe only what that window's tree contains — every
  identifier it names must be declared at HEAD, and every behavioural claim must have a
  test. Measured prevalence in this window, 3 forward-reference sites plus the 3
  backward ones BR-2 left: README.md:80-83 claims a 4+-word or multi-line paste "becomes
  the passage" (pasteIsPassage is not declared at HEAD; every paste goes into the line);
  paste_test.go:344-346 cites pasteIsPassage and TestThePassageSurvivesALookup, neither
  declared; paste.go:126-131 justifies the boundary by "the footer" and "the mark
  painting", which M3 builds. The enumeration is mechanical and the repo already owns
  half of it — TestPlanCitesTestsThatExist and TestPlanTablesNameEntitiesThatExist do
  exactly this derivation for plan files, and TestNoArtifactNamesARetiredSymbol does the
  backward half for currentTruthFiles. Extending that derivation to README, atlas and Go
  doc comments (and dropping the *_test.go exemption for Test* names, which is what hid
  BR-2's residual) closes both directions at once. ARCH-PURPOSE — the class, not the site.
- **BR-12** [Important] `decision-on-incomplete-input` The rune cap is evaluated on a possibly-truncated buffer, so a legal 1000-rune CJK paste is refused
  paste.go:97 decides drain-vs-wait with utf8.RuneCount over a body that may end mid-rune.
  Reproduced at HEAD: a 1000-rune CJK paste split so the first scan sees 2999 body bytes
  counts 1001 (999 complete runes plus 2 orphan bytes each counted as RuneError), latches
  draining, and the closer then returns KeyPasteRefused for a paste that fits — a false
  refusal on exactly the decks /lang exists for, which is the argued point of the rune
  cap. The structural cause is that one predicate serves two purposes: the no-closer
  branch is a MEMORY bound and belongs in bytes (len(body) > maxPasteRunes*utf8.UTFMax),
  while the SEMANTIC cap belongs only where the text is complete, at the closer where it
  already runs. TestPasteScannerCapsInRunesNotBytes uses a single unsplit scan, so it
  cannot see this even though paste.go:41 documents multi-read as the normal path; the
  regression row is a cap-sized CJK paste split at body length 2999. ARCH-CONSTRAINTS.
- **BR-13** [Minor] `repeated-shape-not-extracted` enterPaste/leavePaste is the third copy of the same terminal-mode pair, and restore's ordering is still undeclared
  rawterm.go:105-192 now holds three near-identical enter/leave pairs differing only in
  (flag field, on-string, off-string), and rawSession carries three independent bools —
  8 representable combinations for about 4 legal ones. A single ordered table of
  {flag, on, off} collapses the duplication (ARCH-DRY) and, more usefully, makes
  restore()'s teardown ORDER a declared list rather than three hand-written calls;
  that order is the thing rawterm_test.go:144-151 asserts and the one thing a fourth
  mode's author will not see in the enterPaste template they copy (ARCH-ORDER).
- **BR-14** [Minor] `issue-row-stale` The issue Log records no boundary-review round and its "full suite green" claim was false at the commit it describes
  This is the 2nd finding in family `issue-row-stale`, so state the rule rather than
  patching the line: every claim in the issue that asserts a property of the tree is
  verification evidence and must be re-stated at the boundary that re-verified it.
  The "M1 implemented" entry (issue:937) claims "full suite green" for c1844b3, where
  BR-2 proved the suite was red; it is uncorrected, and there is no ## Log entry for
  boundary-review round 1 at all, which AGENTS.md section 3 requires alongside the
  Review-Verdict trailer. Prevalence in this issue: 2 of 2 tree-asserting claims were
  wrong at some point (the M1 checkbox's "1000-byte cap", fixed as BR-10, and this one).
  The enumeration is short — the Log's verification claims, the Plan checkboxes, and the
  Estimate block's actuals — and appending one Log entry per gate round covers it.

## Round 3 — 2026-09-16T16:47:28-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — Suite green at HEAD and the guard RUNS (--- PASS, 0.20s, not SKIP); render.go:284 now names TestEveryEnabledInputModeIsDecoded and the three test-file sites are swept — no occurrence of the old name remains anywhere under cmd/ or atlas/.
- BR-4 — not-addressed — The Revisions section and the four missing Pure-entities rows landed, but every M1 task checkbox (Tasks 1.1-1.5, plan lines 126-472) is still unticked — 0 ticked boxes in the whole file — which is exactly what round 2 named.
- BR-5 — addressed — README.md:76-92 now documents the single-insertion behaviour, newline/tab flattening, the 1000-character refusal, escape stripping and the abandon rule; atlas/define.md covers pasteLineRunes. Prose-only repair verified against the pinned diff and the behaviour's existing tests.
- BR-6 — addressed — Mutation-verified here: adding "&& k.Kind != KeyPaste" to route's condition at selection_input.go:48 reddens TestAPasteCancelsALiveDrag (paste_test.go:505). The test now drives pointerRouter.route, not cancelPointerInput.
- BR-7 — addressed — The decision is recorded at the site (paste.go:155-160): Cc removed because it is what a terminal acts on, Cf deliberately kept because stripping it would alter words in scripts that need it. No behaviour change, so no regression test is owed — though nothing pins that Cf survives.
- BR-8 — not-addressed — key.go:66-68 is still byte-identical to the base; the struct doc says Raw is an unmodelled sequence to be ignored, while editor.go:63 inserts it for KeyPaste.
- BR-9 — not-addressed — First half stayed fixed. Second half open: TestTheDrainDoesNotCutAStraddlingCloser (paste_test.go:136-149) still hands each scan a fresh buffer rather than the leftover readInput would re-present.
- BR-11 — not-addressed — README and atlas were repaired, but 3 sites remain — paste.go:145-148 and paste_test.go:150-154 still say the body is "bound for the footer" and would "defeat the mark painting" (it goes into the LINE at HEAD), and paste_test.go:348-349 still cites pasteIsPassage and TestThePassageSurvivesALookup in the past tense. The class fix was not written: repo_guard_test.go is byte-identical across the entire window, so currentTruthFiles (:1732-1746) still exempts *_test.go and no forward-direction guard exists.
- BR-12 — addressed — Mutation-verified here: restoring the single "utf8.RuneCount(body) <= maxPasteRunes" predicate reddens TestALegalCJKPasteIsNotRefusedAtAnySplit at split 3011. The split into maxPasteRunes/maxPasteBytes is the structural fix the finding asked for.
- BR-13 — not-addressed — rawterm.go:96-198 is unchanged since c1844b3 — still three hand-written enter/leave pairs, three independent bools, and a hand-written teardown order in restore(). I-2 below approaches the same shape from the test side.
- BR-14 — not-addressed — The issue file changed only by b801efb (one checkbox). There is still no ## Log entry for either boundary-review round, and issue:937 still claims "full suite green" for c1844b3, where BR-2 proved it red.

### Raised

- **BR-15** [Important] `enum-grows-past-consumers` A paste during a review sitting is silently dropped — toInput has no case for KeyPaste, and this window turned mode 2004 on for that surface too
  newConsole calls sess.enterPaste() (replraw.go:89) and BOTH sitting paths run through it: --play via runPlay (play_loop.go:116) and /play via sittingInPlace, which borrows the REPL's keys channel (replraw.go:620). toInput (play_loop.go:565) has no case for KeyPaste or KeyPasteRefused — I confirmed it returns ok=false for both — and play_loop.go:370 continues on !ok. Before this window 2004 was off and a pasted answer arrived as KeyRunes and typed; now it vanishes with no message, and a refused paste produces no notice either because the report lives in runEditor (replraw.go:534), which is not running during a sitting. No test covers it. The durable fix is the guard, not the case: KeyKind has no sentinel (key.go:12-63), so nothing forced the question. The repo already owns the move at render.go:281 (numRegionKinds plus the guards derived from it).
- **BR-16** [Important] `production-seam-untested` sess.enterPaste() is production wiring no test exercises, and mode 2004 has no live conformance row while the mouse does
  This is the 2nd finding in family `production-seam-untested` (BR-3 was the 1st), so do not fix this site alone. Evidence: deleting replraw.go:89 leaves `go test ./cmd/define/ -run 'Paste|Raw|Editor|Console|Repl|Input'` green (ok, 15.9s) — with it gone the terminal never brackets and the whole milestone silently reverts to a pasted newline submitting mid-paste. rawterm_test.go:119-152 calls r.enterPaste() itself, so it pins the METHOD and never the call site, and it asserts into a bytes.Buffer, so it cannot show the sequence reaching a terminal. pty_conformance_test.go:664 is the exact missing precedent, and rawterm.go:163-166 states paste carries the identical no-reset-reflex hazard. The rule that covers the class - every terminal mode the program enables is asserted at the place it is ENABLED and given back on a real terminal, DERIVED from the enable-constant set rather than hand-written per mode. The enumeration is already owned by TestEveryEnabledInputModeIsDecoded: altScreenOn, mouseOn, pasteOn. Sweep it in this round - one in-process assertion per constant that newConsole writes it, one PTY row per constant that it reaches and leaves a real terminal. TestRestoreHandsBackEveryTerminalState hand-writes all three today, which is BR-13's duplication seen from the test side. ARCH-MOCK, ARCH-DRY.

## Open findings

- **BR-4** [Important] `plan-artifact-stale` The durable plan contradicts the shipped code and carries no Revisions entry
- **BR-8** [Minor] `doc-contradicts-type` Key struct doc still says Raw is an unmodelled sequence to be ignored, not inserted
- **BR-9** [Minor] `test-accepts-two-outcomes` TestTheBoundaryKeepsNewlinesAndDropsOtherControls accepts either outcome, pinning neither
- **BR-11** [Important] `doc-contradicts-type` Prose committed in this window asserts M2 behaviour and names symbols the tree does not declare
- **BR-13** [Minor] `repeated-shape-not-extracted` enterPaste/leavePaste is the third copy of the same terminal-mode pair, and restore's ordering is still undeclared
- **BR-14** [Minor] `issue-row-stale` The issue Log records no boundary-review round and its "full suite green" claim was false at the commit it describes
- **BR-15** [Important] `enum-grows-past-consumers` A paste during a review sitting is silently dropped — toInput has no case for KeyPaste, and this window turned mode 2004 on for that surface too
- **BR-16** [Important] `production-seam-untested` sess.enterPaste() is production wiring no test exercises, and mode 2004 has no live conformance row while the mouse does
