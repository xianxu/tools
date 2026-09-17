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
    - "n": 4
      timestamp: "2026-09-16T17:27:39-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: |-
            Behaviour is correct at HEAD, but the over-the-cap half has no regression test: adding
            `&& !s.draining` to paste.go:88 restores the unquittable-program defect and the whole
            suite stays green. The subtest named "over the byte bound, draining" never drains —
            indexPasteAbandon preempts the byte-bound branch (logged draining=false after the first
            decode). Fix is one row, or better, scan's seven exits as a transition table.
          round: 4
        - id: BR-4
          disposition: addressed
          note: |-
            Revisions section appended at plan.md:1147-1185, all M1 checkboxes ticked, Pure-entities
            table now carries sanitisePasteBody/indexPasteAbandon/pasteLineRunes/keyDecoder and drops
            spansToStyled; superseded in-body prose is covered by the Revisions entry per AGENTS.md 1.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: |-
            key.go:65-67 is byte-identical to the base — the struct doc still says Raw is an unmodelled
            sequence to be ignored, while editor.go:64 inserts it as text.
          round: 4
        - id: BR-9
          disposition: addressed
          note: |-
            paste_test.go:174 now pins exactly "a\nb\tcde". Residual from the finding's second clause:
            TestTheDrainDoesNotCutAStraddlingCloser:145 still builds a fresh buffer for its third scan.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: |-
            Instances repaired and the BACKWARD half of the class landed (currentTruthFiles now binds
            *_test.go; 7 stale mentions swept). The FORWARD half was not written: no guard derives
            "every identifier prose names must be declared at HEAD" over README/atlas/Go comments, and
            repo_guard_test.go's only change this window is the exemption lift. Live prevalence at HEAD
            is now 3 — key.go:65-67 (BR-8), README.md:88-90 ("the keyboard keeps working" is false; I
            measured zero keys emerging after an unterminated paste plus hello\r), and the subtest name
            in the Critical above. Recommend scoping the forward guard as its own issue rather than a
            fourth round here.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: |-
            rawterm.go:105-192 still holds three hand-written enter/leave pairs and restore() still
            hand-orders three calls; nothing in the window changes the shape.
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: |-
            The issue's "full suite green" claim for c1844b3 is uncorrected at issue:937, and there is
            still no Log entry for any boundary-review round.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: |-
            The case landed at play_loop.go:593 and reads well, but deleting it leaves
            TestAPasteDuringASittingIsIgnored green — the test asserts the pre-existing default, so it
            is documentation, not a regression test. The durable ask is untouched: key.go:12-63 still
            has no numKeyKinds sentinel, so the next KeyKind meets the same silence.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: |-
            Commenting out replraw.go:89 leaves the full suite green (mutation-verified at HEAD).
            TestNewConsoleEnablesBracketedPaste (paste_test.go:533) never calls newConsole — it builds a
            bare rawSession and calls the three enters by hand, pinning what rawterm_test.go already
            pinned. The PTY row does cover the call site but is darwin+conformance-gated and SKIPPED
            here ("no pty available: operation not permitted"), so I could not verify it. The derived
            enumeration is still hand-written in two places: key_test.go:436 and rawterm_test.go:119-152.
          round: 4
      findings:
        - id: BR-17
          severity: Important
          title: The paste-refusal notice writes into a standing prompt, against this loop's own pinned invariant
          detail: |-
            replraw.go:539 writes to stderr and continues with no frame clear and no redraw. In
            production stdout and stderr are the same liveScreen (replraw.go:127), so the message
            repaints around the live edge and lands inside the prompt row, and the frame stays wrong
            until the next keystroke. Verified: TestNothingIsWrittenWhileAPromptIsShown
            (editorloop_test.go:553), copied verbatim and driven with one KeyPasteRefused, fails with
            'write 1 of 1 landed with a prompt on the frame'. The precedent three cases above it is
            correct — case res := <-bgResults: does view.Draw("", nil), then writes, then draw()
            (replraw.go:504-510), with a comment naming this exact rule. TestEditorLoopReportsARefusedPaste
            cannot see it because it passes a separate bytes.Buffer for stderr. Fix: clear, write,
            redraw, and drive KeyPasteRefused through the invariant test. ARCH-ORDER.
          family: notice-bypasses-frame-protocol
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-09-16T21:51:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified at HEAD: adding "&& !s.draining" to the abandon condition (paste.go:113) reddens TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt/after_the_drain_has_started ("Enter never emerged"); the pasteExit enumeration plus TestEveryPasteExitIsExercised closes round 4''s untested-drain residual. Residual noted as Minor: the comment claims "a control byte" while indexPasteAbandon matches only 0x03/0x04.'
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: key.go:66-68 is still byte-identical to the base — the struct doc says Raw carries an unmodelled sequence "so it can be ignored rather than inserted as garbage" while editor.go:64 inserts it for KeyPaste.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: 'The named instances are repaired and the BACKWARD half of the class landed (currentTruthFiles now binds *_test.go, repo_guard_test.go:1738-1755). The FORWARD half was not written, and prevalence at HEAD is 3 new sites committed in this window: passage.go:16-22 ("It is CHROME, not scrollback ... It lives in the footer" — it is written to the buffer by replraw.go:605), session.go:24 ("pinned in the footer"), and README.md:76-82 ("goes in at the cursor", false for any 4-word or multi-line paste).'
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: rawterm.go:105-192 still holds three hand-written enter/leave pairs and three independent bools, and restore() still hand-orders the three leaves; nothing in this window changes the shape. Newly-raised I-2 is the same rule one altitude up (duplicated prompt policy), which is worth fixing together.
          round: 5
        - id: BR-14
          disposition: not-addressed
          note: 'Still no "## Log" entry for any boundary-review round (the rounds appear only in Revisions), and issue:937 still claims "full suite green" for c1844b3 where BR-2 proved it red. A third instance now: the Plan checkbox still claims "the passage re-rendering green", which the 2026-09-16 "passage is a record" revision explicitly dropped.'
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: 'Mutation-verified at HEAD: deleting the "case KeyPaste, KeyPasteRefused" arm from toInput (play_loop.go:593) leaves go test ./cmd/define/ -run ''Sitting|Play|Paste|Input'' green, because TestAPasteDuringASittingIsIgnored asserts the pre-existing default. The durable ask is untouched — grep shows no numKeyKinds sentinel in key.go, so the next KeyKind meets the same silence.'
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: 'Mutation-verified at HEAD: commenting out sess.enterPaste() (replraw.go:89) leaves the full in-process suite green. TestNewConsoleEnablesBracketedPaste (paste_test.go:612) never calls newConsole — it builds a bare rawSession and calls the three enters by hand, pinning what rawterm_test.go already pinned. The PTY row does cover the call site but SKIPS here ("no pty available: operation not permitted"), verified with -v, so no runnable test covers it. The enumeration is still hand-written in two places (key_test.go:436, rawterm_test.go:119-152) rather than derived from the enable constants.'
          round: 5
        - id: BR-17
          disposition: addressed
          note: 'Mutation-verified: removing the view.Draw("", nil) before the notice (replraw.go:614) reddens TestTheRefusalNoticeDoesNotLandInsideThePrompt with "the loop drew 1 prompts; the notice did not clear the frame". Clear-write-redraw now matches the bgResults precedent.'
          round: 5
      findings:
        - id: BR-18
          severity: Critical
          title: A superseded passage's regions stay clickable and resolve against the CURRENT passage, marking an unrelated word
          detail: 'passageRegions stores Region.Line relative to its own passage; addRegions keys the screen map by absolute buffer line but leaves that field alone; nothing removes an old passage''s regions on the next paste; and passageSpanOf (passage.go:367) resolves r.Line/r.Col against sess.passage with no identity check. Measured at HEAD with two passages: clicking "alpha" in the old one resolves to "zulu" in the current one, "beta" to "yankee", "epsilon" to "whiskey", and "gamma" to nothing at all (a silent no-op with no message). The mark then paints at passageBase+line, i.e. on the new passage, and the next bare Enter brackets and admits a word the reader never marked. Fix: stamp a passage generation into the Region (or resolve via the absolute buffer line and require 0 <= line-passageBase < lineCount) and reject a click that does not belong to the live passage; pin it with a two-paste regression through runEditor. ARCH-ORDER.'
          family: observation-outlives-its-subject
          round: 5
        - id: BR-19
          severity: Critical
          title: A drag marks each word separately, so the decided single span exists only in marksForDrag, which production never calls
          detail: 'Production drags go passageWordsInLocked -> passageSpanOf -> toggle per Region (replraw.go:568-576), so a drag over the issue''s own example yields four marks and the prompt reads "he stopped [sel]at[/sel] [sel]the[/sel] [sel]zenith[/sel] [sel]of[/sel] the arc" (measured). The Spec decided the opposite: "both gestures produce a SPAN, differing only in how the span is derived". Consequences: admitMarkedWords (ask.go:302) runs the dictionary per word, so with NOAD installed "at", "the" and "of" are hits and enter the deck as EventMarked — durable state, undoable only via /forget — inverting the Done-when "a dragged phrase with no dictionary entry stays out of the deck". Meanwhile marksForDrag, passageCell, firstWordFrom, lastWordTo and wrappedColumn have zero production callers, p.raw() has none at all, and marksForDrag''s correct single span ("at the zenith of") is SILENTLY DROPPED by markedPassageText, which matches a mark only when it equals a whole word run — so the three tests at marks_test.go:70-109 certify behaviour the program does not have and could not send. Same decision, second site: the plan specified "a drag whose ANCHOR ROW is a passage row", but passageWordsInLocked tests overlap across every covered row, so a drag from an answer into the passage loses its copy and marks passage words instead. Fix the class in one pass: pick one representation, wire it, and let no entity in the plan''s table keep a zero-consumer test. ARCH-DRY, ARCH-PURPOSE.'
          family: tested-entity-not-wired
          round: 5
        - id: BR-20
          severity: Important
          title: A tab survives the paste boundary but counts as one cell, so clicks land on the wrong word; an unbreakable token overflows the wrap
          detail: 'This is the 2nd finding in family `boundary-parses-partial-class` (BR-7 was the 1st), so do NOT fix the tab alone. The rule that covers both: every character class sanitisePasteBody ADMITS must be representable by every downstream consumer of the passage''s cell arithmetic. Measured: newPassage("\tthe slow precession", 0) — a click at display column 8, where the terminal draws "the", resolves to "slow", because cellWidth gives a tab 1 cell and the terminal gives it 8; pasteLineRunes flattens tabs on the LINE path and the passage keeps them, so the same input has two meanings. Second instance of the same rule: wrapPassageLines cannot break a long token, so a 70-cell URL stays one line at width 30 and clipVisible truncates it — the tail is neither readable nor clickable, which is the operator-reported bug the wrapping fix was for. The enumeration is short: the classes the boundary admits (newline, tab, Cf, wide glyphs, combining marks) x the consumers (wrapPassageLines, byteAtCell, spanCells, paintMarks), as one table test.'
          family: boundary-parses-partial-class
          round: 5
        - id: BR-21
          severity: Important
          title: passageSystem restates askSystem's level default, dictionary authority and language grammar instead of composing them
          detail: 'This is the 2nd finding in family `repeated-shape-not-extracted` (BR-13 is the 1st and still open), so do NOT fix this site alone. The rule: a prompt paragraph consumed by more than one task is a named constant composed into each system prompt, and the shared set is derived rather than remembered. renderPassagePrompt replaces req.System wholesale (passageprompt.go:48), so the reversed level default, "Never invent a definition that contradicts a dictionary entry", the [lang=xx] annotation grammar and the &#91;/&#93; escape rule exist twice — visible side by side in testdata/golden/passage-prompt.txt and ask-prompt.txt. The issue forbade exactly this: "One answer to ''what level do we assume,'' stated once — a one-line reversal in askSystem that read-along inherits, not a second default in a second prompt (ARCH-DRY)". The language block is load-bearing for language_decode.go, so drift breaks bilingual rendering of passage answers only. BR-13''s three terminal-mode triples are the same rule at the other altitude; one extraction pattern should serve both.'
          family: repeated-shape-not-extracted
          round: 5
        - id: BR-22
          severity: Important
          title: 'The Done-when audit was not run: the style-after-the-mark regression is mutation-green, and the dragged-phrase row cannot hold as written'
          detail: 'This is the 2nd finding in family `decided-behaviour-unpinned` (BR-6 was the 1st), so do NOT just add the two tests. The rule: a Done-when row is a test obligation, and the close step''s audit is its enumeration — walk the rows and record the pinning test''s name beside each, so a row with no test is visible rather than asserted. Evidence: plan.md:1128 ("Confirm every ## Done when row in the issue has a test naming it") is unchecked while the issue''s Plan is fully ticked, and the row that names its own test — "the token AFTER the mark still carries the style it had ... needs a test that inspects the style after the span: stripping escapes is exactly what hides a lost one" — is unpinned: replacing both "sgrOff + style.resume()" writes in paintMarks with bare sgrOff leaves the whole suite green. The second unpinned row is the dragged-phrase admission rule, which the Critical above shows the code contradicts.'
          family: decided-behaviour-unpinned
          round: 5
        - id: BR-23
          severity: Important
          title: README's paste section is now wrong and the whole read-along surface — click, drag, Enter-asks, the nudge, admission — is undocumented
          detail: 'This is the 2nd finding in family `readme-surface-undocumented` (BR-5 was the 1st), so do NOT patch the paragraph alone. The rule: a gesture or key the window changes is a README row, and the enumeration is derivable — RegionKind x regionPlaysAudio/regionUnderlines for clicks and replKind for Enter, the same derivation TestAtlasDescribesEveryRegionKind already runs against the atlas; extend that guard to README.md. Sites at HEAD: README.md:76-82 says "The whole paste arrives at once and goes in at the cursor; newlines and tabs inside it become spaces", true only for a 1-3 word single-line paste since pasteIsPassage routes everything else to the passage; README.md:439 still says a click "plays the word" everywhere; README.md:441''s Enter row does not mention that marks make Enter an ask. Nothing documents click/drag-to-mark, marks-win-over-replay, noteNothingMarked, marks clearing after an ask, or deck admission — the entire feature the issue is named for.'
          family: readme-surface-undocumented
          round: 5
        - id: BR-24
          severity: Important
          title: The durable plan contradicts the tree in five places, and its unticked boxes switch off the two guards that would have caught two of them
          detail: 'This is the 2nd finding in family `plan-artifact-stale` (BR-4 was the 1st), so do NOT patch the rows one at a time. The rule: a plan''s checkboxes and tables are claims about the tree that this repo''s guards READ, so leaving them behind disables the guards — tick the boxes and append the Revisions entry at the commit that lands the departure, not at the close. Sites: Chunks 2-5 are entirely unticked though the work shipped; the Architecture paragraph still says the passage "is chrome, not scrollback ... it lives in the screen''s existing footer []string channel"; "What this plan does NOT do" still says a passage-word RegionKind was "resolved away in Chunk 3" while RegionPassageWord is shipped; the Core-concepts table claims `selectionFrame.highlightRow | selection_frame.go | modified` for a file this window never touched; Task 2.2 declares TestThePassageSurvivesALookup, which the tree does not have. The guard interaction is the teeth: TestPlanTablesNameEntitiesThatExist exempts `new` rows whenever any "- [ ]" remains (repo_guard_test.go:896), and TestPlanTableStatusMatchesTheChangeWindow skips a row whose file the window did not touch (:1378), so exactly those two claims are invisible.'
          family: plan-artifact-stale
          round: 5
        - id: BR-25
          severity: Important
          title: gatherAskContext runs twice on every passage ask, doubling the deck and learner-model reads and any warning they print
          detail: 'ask.go:165 computes req := renderAskPrompt(gatherAskContext(d, sess, q, errOut)) and throws it away; ask.go:170 calls gatherAskContext again for the passage renderer. gatherAskContext is the IO step — d.deck.UserModel() and d.deck.Deck() — and it warns on failure, so an unreadable deck prints "define: could not read the deck (...); answering without it" twice to the user. Fix: gather once into a local, then choose the renderer.'
          family: work-repeated-on-one-path
          round: 5
        - id: BR-26
          severity: Minor
          title: paintMarks emits a 256-colour SGR pair regardless of opt.color, so marking a word under -no-color produces colour
          detail: draw() calls view.SetMarks unconditionally (replraw.go:425) and layoutSelectionFrame applies paintMarks whenever a row has marks (screen.go:622), with no reference to opt.color. Unlike the selection's inverse video, markOn is "\x1b[48;5;24m\x1b[38;5;231m". passageText already honours the flag, so the two halves of the same surface disagree.
          family: decoration-ignores-color-option
          round: 5
        - id: BR-27
          severity: Minor
          title: Each paste appends one Region per word to screen.regions with no removal path, which the plan's ARCH-FUNERAL note does not cover
          detail: 'replraw.go:605 writes passageRegions into the screen on every passage paste — roughly 170 entries for a 1000-character passage — and screen.regions is only ever appended to. The plan states "The passage and its marks are in-memory, die with the session, and are replaced wholesale by the next paste", which holds for sess.passage but not for the screen''s copy: per-session growth per paste is larger than before this window. Note it and state the bound, or clear the superseded passage''s regions (which would also help the stale-region Critical).'
          family: artifact-family-without-removal
          round: 5
      blocked: true
    - "n": 6
      timestamp: "2026-09-16T23:00:17-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: key.go:74-76 is byte-identical; the Key doc still says Raw is an unmodelled sequence to be ignored while KeyPaste's Raw is sanitised text that Apply inserts.
          round: 6
        - id: BR-11
          disposition: not-addressed
          note: 'The three named sites were repaired, but no forward-direction guard was written and five NEW contradicting sites shipped in this window: passage.go:17-21 ("It is CHROME, not scrollback ... It lives in the footer"), session.go:24 ("pinned in the footer"), replraw.go:558-562 ("the hit test is FooterRowAt plus wordAtCell, and no new RegionKind exists", three lines above code keying on RegionPassageWord), atlas/define.md:313-314 ("Newlines and tabs survive" while sanitisePasteBody turns tabs into spaces and paste_test.go:183 pins that), and the plan''s footer sections. currentTruthFiles now binds *_test.go (the backward half), which is real progress; the forward half — every identifier prose names must be declared at HEAD — does not exist.'
          round: 6
        - id: BR-13
          disposition: not-addressed
          note: 'enabledModes (rawterm.go:183) is the right list but nothing derives from it beyond the two new guards: three hand-written enter/leave pairs over three bools remain, restore() still hand-calls its teardown in an order the list does not express (list is alt/mouse/paste, teardown is mouse/paste/alt), and key_test.go:431 still hand-concatenates `const inputModes = mouseOn + pasteOn`, so a fourth mode is invisible to it.'
          round: 6
        - id: BR-14
          disposition: not-addressed
          note: 'issue:961 still claims "full suite green" for the commit BR-2 proved red, and ## Log still holds exactly one dated section with no per-round boundary-review entry (five rounds have now run). The milestone-collapse Revisions entry mentions "four review rounds" in passing, which is the nearest thing to a record.'
          round: 6
        - id: BR-15
          disposition: not-addressed
          note: 'The case and the declaration landed (play_loop.go:571, :617) and TestAPasteDuringASittingIsIgnored pins the behaviour — but the guard, which was the finding, cannot fail for the failure it names. Measured: adding a KeyKind before numKeyKinds leaves TestEveryKeyKindIsDecidedForASitting GREEN, because keyBecomesASittingInput and toInput both default to false, so a kind wired into neither agrees with itself. numPasteExits and numRegionKinds fail closed in this same window; this one fails open.'
          round: 6
        - id: BR-16
          disposition: addressed
          note: 'Mutation-verified: deleting sess.enterPaste() (replraw.go:89) reddens TestNewConsoleEnablesEveryMode with "newConsole never enabled bracketed paste". The PTY row TestPTYEveryEnabledModeIsAskedForAndGivenBack exists and loops enabledModes; it compiles under -tags conformance (go vet clean) and skips here because pty.Open is denied.'
          round: 6
        - id: BR-18
          disposition: addressed
          note: 'The CLICK path is genuinely fixed and pinned: passageSpanAt gates on ownsBufferLine (session.go:79,99) and TestAStalePassagesRegionsDoNotMarkTheCurrentOne asserts a stale region refuses. The enumerable siblings of the class are raised fresh below rather than re-raised here.'
          round: 6
        - id: BR-19
          disposition: addressed
          note: 'Production drags now go through marksForDrag (replraw.go:573) and yield ONE span; the anchor gate is enforced in passageDragLocked (selection_screen.go:177) so a drag from an answer keeps its copy; markedPassageText matches by position so a phrase reaches the wire as one bracket; TestADraggedPhraseIsAdmittedAsAPhraseOrNotAtAll reads both the deck and the recorded prompt. Residual: passage.raw() still has zero consumers anywhere (raised Minor).'
          round: 6
        - id: BR-20
          disposition: not-addressed
          note: 'The tab instance is fixed and pinned to ONE outcome (paste.go sanitisePasteBody, paste_test.go:168-185). The rule fix was not written: no classes-by-consumers table test exists, and the second instance stands. Measured at HEAD: newPassage("see <67-cell URL> for more", 30) yields a 67-cell line and the frame draws "https://example.com/a/very/lon" — clipVisible truncates because wrapText cannot break an unbreakable token, so the tail is neither readable nor clickable. That is the operator-reported wrapping bug, half fixed.'
          round: 6
        - id: BR-21
          disposition: addressed
          note: sharedLevel / sharedAuthority / sharedLanguageGrammar extracted in askctx.go:147-191 and composed by both askSystem and passageSystem; escLeft/escRight extracted and consumed by both the prompt escape and the answer-direction rule. Both goldens now show the identical clauses.
          round: 6
        - id: BR-22
          disposition: not-addressed
          note: 'Half addressed with real evidence: TestTheTokenAfterAMarkKeepsItsStyle is mutation-red (replacing both "sgrOff + style.resume()" writes with bare sgrOff fails it), and the dragged-phrase row now has a test. But the audit asserts pins that do not exist — TestThePassageIsWrittenToTheBufferNotTheFooter and TestAMarkedDeckWordRendersAsAMarkNotAsADeckWord are in no _test.go file — so two rows are still asserted rather than visible, which is the rule''s own failure mode. No guard reads workshop/issues/; TestPlanCitesTestsThatExist globs workshop/plans/*-plan.md only, which is where the teeth belong.'
          round: 6
        - id: BR-23
          disposition: not-addressed
          note: 'The new "Read along" section (README.md:76-121) is accurate and thorough, and the old paste paragraph is gone. But the derived guard the rule named was not written (TestAtlasDescribesEveryRegionKind still reads only the atlas), and two rows now contradict the new section: README.md:163 "Enter on an empty line | replay the pronunciation" is unconditional, and README.md:169-174 still says "Underlined words are clickable" and "A click on ordinary text does nothing" while passage words are clickable and deliberately NOT underlined.'
          round: 6
        - id: BR-24
          disposition: not-addressed
          note: 'Boxes ticked and three of five sites repaired (Architecture paragraph, RegionPassageWord bullet, selectionFrame.highlightRow row removed). Remaining: Integration points row "passage-as-footer" (plan:80) and its bullet (:91); Task 2.2 title plus Step 3 (:591), Task 2.3, and Task 2.4''s ticked atlas row (:611) all still say footer; Task 3.2 "Widen highlightRow from one range to a set" (:707) is fully ticked though highlightRow is unchanged at selection_frame.go:208 and this window never touches that file; plan:579 and :583 declare TestThePassageRendersWithDeckColour and TestThePassageIsWrittenToTheBufferNotTheFooter, neither in the tree. And the rule''s own second clause was not followed: the footer-to-buffer reversal was applied by OVERWRITING the Architecture paragraph, with no Revisions entry appended.'
          round: 6
        - id: BR-25
          disposition: addressed
          note: 'ask.go:167 gathers once into askCtx and both renderers take it; the double deck/learner-model read and the duplicated warning are gone. Residual (Minor below): renderAskPrompt itself still runs twice on the passage path, purely.'
          round: 6
        - id: BR-26
          disposition: not-addressed
          note: replraw.go:425 calls SetMarks(markCellRanges(...)) and screen.go:622-624 applies paintMarks whenever a row has marks, with no reference to opt.color anywhere on the path; markOn is "\x1b[48;5;24m\x1b[38;5;231m". passageText still honours the flag, so the two halves of the same surface disagree under -no-color.
          round: 6
        - id: BR-27
          disposition: not-addressed
          note: replraw.go:609 still writes passageRegions into screen.regions on every passage paste with no removal path, and the plan's ARCH-FUNERAL paragraph is unchanged — it still says "no removal path needed", which holds for sess.passage and not for the screen's copy. Clearing a superseded passage's regions would also shrink the new Critical's blast radius.
          round: 6
      findings:
        - id: BR-28
          severity: Critical
          title: The live-passage gate is on the click path only — a drag in a superseded passage marks the CURRENT one, and hasPassage never expires
          detail: 'This is the 2nd finding in family `observation-outlives-its-subject`, so do NOT patch the drag call alone. The rule that covers both sites: the session must answer ONE question — "is the live passage reachable at this point?" — and every consumer of passage state routes through it. The enumeration is three long and two are ungated. (1) `passageSpanAt` (session.go:99) gates on `ownsBufferLine` — correct, and BR-18''s fix. (2) `passageCell` (session.go:89) CLAMPS instead of refusing, and `passageDragLocked` (selection_screen.go:177) only checks that the anchor row carries some `RegionPassageWord`, which a superseded passage''s rows still do because `screen.regions` is never pruned. Measured at HEAD with two passages, drag anchored on the old one''s buffer row 10 against a current passage at base 40: `passageCell(10,0) = {line:0 col:0}` and `marksForDrag` returns one span — MARKED "zulu yankee xray" of the current passage. Those words are then bracketed in the next prompt and, on a dictionary hit, admitted to the deck as durable EventMarked records; the copy the reader asked for is swallowed by the `continue` at replraw.go:576. (3) `lineState().hasPassage` (session.go:50) treats "a passage was once pasted" as permanent authority over Enter. Measured: with `current` set a bare Enter is `cmdReplay`; after one paste it is `cmdNothing`/`noteNothingMarked` and never returns to replay for the rest of the session — long after the passage has scrolled away, while the nudge points at something possibly off-screen. That decision was right when the passage was pinned footer chrome; the footer-to-buffer reversal changed the surface''s lifetime and the predicate was not re-derived. Fix: `passageCell` returns `(passageCell, bool)` and refuses a row outside `ownsBufferLine`; `hasPassage` means the live passage is on screen. Pin with a two-passage drag regression through runEditor and one asserting replay returns once the passage is gone. Note `pointerClick.line` cannot serve as the gate as written: resolvePointerLocked (selection_screen.go:116) overwrites it from `click.point`, which is the zero value on the drag path, so it carries row 0''s buffer line and `hasRegion` can be spuriously true — hasRegion/hasDrag/footer/retry/line is a five-field constellation whose legal combinations are unwritten and should collapse into a tagged variant. ARCH-ORDER, ARCH-PURPOSE.'
          family: observation-outlives-its-subject
          round: 6
        - id: BR-29
          severity: Important
          title: Deleting the cmdAskPassage branch in runEditor leaves the suite green, and replKind has no sentinel so replLines silently has no case for it
          detail: 'This is the 3rd finding in family `production-seam-untested` (BR-3, BR-16), so do NOT just add a test for this line. Measured: removing the whole `if cmd.kind == cmdAskPassage { askInSession(...) }` block at replraw.go:667-675 leaves `go test ./cmd/define/...` green apart from the two pty-denied rows — the one line that turns "Enter with marks" into an actual model call is unpinned. Every ask test builds `question{passage: ...}` by hand (passageprompt_test.go:171-340) and TestABlankLineWithMarksAsksAboutThePassage stops at parseREPLLine, so nothing joins the decision to the effect. The rule, which is BR-16''s one altitude up: a decision table''s kinds are a registry, so each kind''s disposition in each loop is DECLARED and DERIVED, not hand-written per loop. Second site proving it is a class: `replLines` (repl.go:384) has no case for cmdAskPassage at all — unreachable today only because the piped loop never decodes a paste — and `replKind` (repl.go:45) has no `numReplKinds`, which is why nothing forced the question. This window makes exactly that move three times (numPasteExits, numRegionKinds, numKeyKinds); make it a fourth and drive both loops from it, with the guard failing CLOSED (see the BR-15 disposition for the failure mode to avoid).'
          family: production-seam-untested
          round: 6
        - id: BR-30
          severity: Minor
          title: passage.raw() has zero consumers anywhere in the tree
          detail: 'passage.go:75. BR-19 named it and it survived the round: no production caller and no test caller. Delete it rather than keeping it for symmetry with lineCount/line/spans/text, all of which are used.'
          family: tested-entity-not-wired
          round: 6
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

## Round 4 — 2026-09-16T17:27:39-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Behaviour is correct at HEAD, but the over-the-cap half has no regression test: adding
`&& !s.draining` to paste.go:88 restores the unquittable-program defect and the whole
suite stays green. The subtest named "over the byte bound, draining" never drains —
indexPasteAbandon preempts the byte-bound branch (logged draining=false after the first
decode). Fix is one row, or better, scan's seven exits as a transition table.
- BR-4 — addressed — Revisions section appended at plan.md:1147-1185, all M1 checkboxes ticked, Pure-entities
table now carries sanitisePasteBody/indexPasteAbandon/pasteLineRunes/keyDecoder and drops
spansToStyled; superseded in-body prose is covered by the Revisions entry per AGENTS.md 1.
- BR-8 — not-addressed — key.go:65-67 is byte-identical to the base — the struct doc still says Raw is an unmodelled
sequence to be ignored, while editor.go:64 inserts it as text.
- BR-9 — addressed — paste_test.go:174 now pins exactly "a\nb\tcde". Residual from the finding's second clause:
TestTheDrainDoesNotCutAStraddlingCloser:145 still builds a fresh buffer for its third scan.
- BR-11 — not-addressed — Instances repaired and the BACKWARD half of the class landed (currentTruthFiles now binds
*_test.go; 7 stale mentions swept). The FORWARD half was not written: no guard derives
"every identifier prose names must be declared at HEAD" over README/atlas/Go comments, and
repo_guard_test.go's only change this window is the exemption lift. Live prevalence at HEAD
is now 3 — key.go:65-67 (BR-8), README.md:88-90 ("the keyboard keeps working" is false; I
measured zero keys emerging after an unterminated paste plus hello\r), and the subtest name
in the Critical above. Recommend scoping the forward guard as its own issue rather than a
fourth round here.
- BR-13 — not-addressed — rawterm.go:105-192 still holds three hand-written enter/leave pairs and restore() still
hand-orders three calls; nothing in the window changes the shape.
- BR-14 — not-addressed — The issue's "full suite green" claim for c1844b3 is uncorrected at issue:937, and there is
still no Log entry for any boundary-review round.
- BR-15 — not-addressed — The case landed at play_loop.go:593 and reads well, but deleting it leaves
TestAPasteDuringASittingIsIgnored green — the test asserts the pre-existing default, so it
is documentation, not a regression test. The durable ask is untouched: key.go:12-63 still
has no numKeyKinds sentinel, so the next KeyKind meets the same silence.
- BR-16 — not-addressed — Commenting out replraw.go:89 leaves the full suite green (mutation-verified at HEAD).
TestNewConsoleEnablesBracketedPaste (paste_test.go:533) never calls newConsole — it builds a
bare rawSession and calls the three enters by hand, pinning what rawterm_test.go already
pinned. The PTY row does cover the call site but is darwin+conformance-gated and SKIPPED
here ("no pty available: operation not permitted"), so I could not verify it. The derived
enumeration is still hand-written in two places: key_test.go:436 and rawterm_test.go:119-152.

### Raised

- **BR-17** [Important] `notice-bypasses-frame-protocol` The paste-refusal notice writes into a standing prompt, against this loop's own pinned invariant
  replraw.go:539 writes to stderr and continues with no frame clear and no redraw. In
  production stdout and stderr are the same liveScreen (replraw.go:127), so the message
  repaints around the live edge and lands inside the prompt row, and the frame stays wrong
  until the next keystroke. Verified: TestNothingIsWrittenWhileAPromptIsShown
  (editorloop_test.go:553), copied verbatim and driven with one KeyPasteRefused, fails with
  'write 1 of 1 landed with a prompt on the frame'. The precedent three cases above it is
  correct — case res := <-bgResults: does view.Draw("", nil), then writes, then draw()
  (replraw.go:504-510), with a comment naming this exact rule. TestEditorLoopReportsARefusedPaste
  cannot see it because it passes a separate bytes.Buffer for stderr. Fix: clear, write,
  redraw, and drive KeyPasteRefused through the invariant test. ARCH-ORDER.

## Round 5 — 2026-09-16T21:51:08-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified at HEAD: adding "&& !s.draining" to the abandon condition (paste.go:113) reddens TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt/after_the_drain_has_started ("Enter never emerged"); the pasteExit enumeration plus TestEveryPasteExitIsExercised closes round 4's untested-drain residual. Residual noted as Minor: the comment claims "a control byte" while indexPasteAbandon matches only 0x03/0x04.
- BR-8 — not-addressed — key.go:66-68 is still byte-identical to the base — the struct doc says Raw carries an unmodelled sequence "so it can be ignored rather than inserted as garbage" while editor.go:64 inserts it for KeyPaste.
- BR-11 — not-addressed — The named instances are repaired and the BACKWARD half of the class landed (currentTruthFiles now binds *_test.go, repo_guard_test.go:1738-1755). The FORWARD half was not written, and prevalence at HEAD is 3 new sites committed in this window: passage.go:16-22 ("It is CHROME, not scrollback ... It lives in the footer" — it is written to the buffer by replraw.go:605), session.go:24 ("pinned in the footer"), and README.md:76-82 ("goes in at the cursor", false for any 4-word or multi-line paste).
- BR-13 — not-addressed — rawterm.go:105-192 still holds three hand-written enter/leave pairs and three independent bools, and restore() still hand-orders the three leaves; nothing in this window changes the shape. Newly-raised I-2 is the same rule one altitude up (duplicated prompt policy), which is worth fixing together.
- BR-14 — not-addressed — Still no "## Log" entry for any boundary-review round (the rounds appear only in Revisions), and issue:937 still claims "full suite green" for c1844b3 where BR-2 proved it red. A third instance now: the Plan checkbox still claims "the passage re-rendering green", which the 2026-09-16 "passage is a record" revision explicitly dropped.
- BR-15 — not-addressed — Mutation-verified at HEAD: deleting the "case KeyPaste, KeyPasteRefused" arm from toInput (play_loop.go:593) leaves go test ./cmd/define/ -run 'Sitting|Play|Paste|Input' green, because TestAPasteDuringASittingIsIgnored asserts the pre-existing default. The durable ask is untouched — grep shows no numKeyKinds sentinel in key.go, so the next KeyKind meets the same silence.
- BR-16 — not-addressed — Mutation-verified at HEAD: commenting out sess.enterPaste() (replraw.go:89) leaves the full in-process suite green. TestNewConsoleEnablesBracketedPaste (paste_test.go:612) never calls newConsole — it builds a bare rawSession and calls the three enters by hand, pinning what rawterm_test.go already pinned. The PTY row does cover the call site but SKIPS here ("no pty available: operation not permitted"), verified with -v, so no runnable test covers it. The enumeration is still hand-written in two places (key_test.go:436, rawterm_test.go:119-152) rather than derived from the enable constants.
- BR-17 — addressed — Mutation-verified: removing the view.Draw("", nil) before the notice (replraw.go:614) reddens TestTheRefusalNoticeDoesNotLandInsideThePrompt with "the loop drew 1 prompts; the notice did not clear the frame". Clear-write-redraw now matches the bgResults precedent.

### Raised

- **BR-18** [Critical] `observation-outlives-its-subject` A superseded passage's regions stay clickable and resolve against the CURRENT passage, marking an unrelated word
  passageRegions stores Region.Line relative to its own passage; addRegions keys the screen map by absolute buffer line but leaves that field alone; nothing removes an old passage's regions on the next paste; and passageSpanOf (passage.go:367) resolves r.Line/r.Col against sess.passage with no identity check. Measured at HEAD with two passages: clicking "alpha" in the old one resolves to "zulu" in the current one, "beta" to "yankee", "epsilon" to "whiskey", and "gamma" to nothing at all (a silent no-op with no message). The mark then paints at passageBase+line, i.e. on the new passage, and the next bare Enter brackets and admits a word the reader never marked. Fix: stamp a passage generation into the Region (or resolve via the absolute buffer line and require 0 <= line-passageBase < lineCount) and reject a click that does not belong to the live passage; pin it with a two-paste regression through runEditor. ARCH-ORDER.
- **BR-19** [Critical] `tested-entity-not-wired` A drag marks each word separately, so the decided single span exists only in marksForDrag, which production never calls
  Production drags go passageWordsInLocked -> passageSpanOf -> toggle per Region (replraw.go:568-576), so a drag over the issue's own example yields four marks and the prompt reads "he stopped [sel]at[/sel] [sel]the[/sel] [sel]zenith[/sel] [sel]of[/sel] the arc" (measured). The Spec decided the opposite: "both gestures produce a SPAN, differing only in how the span is derived". Consequences: admitMarkedWords (ask.go:302) runs the dictionary per word, so with NOAD installed "at", "the" and "of" are hits and enter the deck as EventMarked — durable state, undoable only via /forget — inverting the Done-when "a dragged phrase with no dictionary entry stays out of the deck". Meanwhile marksForDrag, passageCell, firstWordFrom, lastWordTo and wrappedColumn have zero production callers, p.raw() has none at all, and marksForDrag's correct single span ("at the zenith of") is SILENTLY DROPPED by markedPassageText, which matches a mark only when it equals a whole word run — so the three tests at marks_test.go:70-109 certify behaviour the program does not have and could not send. Same decision, second site: the plan specified "a drag whose ANCHOR ROW is a passage row", but passageWordsInLocked tests overlap across every covered row, so a drag from an answer into the passage loses its copy and marks passage words instead. Fix the class in one pass: pick one representation, wire it, and let no entity in the plan's table keep a zero-consumer test. ARCH-DRY, ARCH-PURPOSE.
- **BR-20** [Important] `boundary-parses-partial-class` A tab survives the paste boundary but counts as one cell, so clicks land on the wrong word; an unbreakable token overflows the wrap
  This is the 2nd finding in family `boundary-parses-partial-class` (BR-7 was the 1st), so do NOT fix the tab alone. The rule that covers both: every character class sanitisePasteBody ADMITS must be representable by every downstream consumer of the passage's cell arithmetic. Measured: newPassage("\tthe slow precession", 0) — a click at display column 8, where the terminal draws "the", resolves to "slow", because cellWidth gives a tab 1 cell and the terminal gives it 8; pasteLineRunes flattens tabs on the LINE path and the passage keeps them, so the same input has two meanings. Second instance of the same rule: wrapPassageLines cannot break a long token, so a 70-cell URL stays one line at width 30 and clipVisible truncates it — the tail is neither readable nor clickable, which is the operator-reported bug the wrapping fix was for. The enumeration is short: the classes the boundary admits (newline, tab, Cf, wide glyphs, combining marks) x the consumers (wrapPassageLines, byteAtCell, spanCells, paintMarks), as one table test.
- **BR-21** [Important] `repeated-shape-not-extracted` passageSystem restates askSystem's level default, dictionary authority and language grammar instead of composing them
  This is the 2nd finding in family `repeated-shape-not-extracted` (BR-13 is the 1st and still open), so do NOT fix this site alone. The rule: a prompt paragraph consumed by more than one task is a named constant composed into each system prompt, and the shared set is derived rather than remembered. renderPassagePrompt replaces req.System wholesale (passageprompt.go:48), so the reversed level default, "Never invent a definition that contradicts a dictionary entry", the [lang=xx] annotation grammar and the &#91;/&#93; escape rule exist twice — visible side by side in testdata/golden/passage-prompt.txt and ask-prompt.txt. The issue forbade exactly this: "One answer to 'what level do we assume,' stated once — a one-line reversal in askSystem that read-along inherits, not a second default in a second prompt (ARCH-DRY)". The language block is load-bearing for language_decode.go, so drift breaks bilingual rendering of passage answers only. BR-13's three terminal-mode triples are the same rule at the other altitude; one extraction pattern should serve both.
- **BR-22** [Important] `decided-behaviour-unpinned` The Done-when audit was not run: the style-after-the-mark regression is mutation-green, and the dragged-phrase row cannot hold as written
  This is the 2nd finding in family `decided-behaviour-unpinned` (BR-6 was the 1st), so do NOT just add the two tests. The rule: a Done-when row is a test obligation, and the close step's audit is its enumeration — walk the rows and record the pinning test's name beside each, so a row with no test is visible rather than asserted. Evidence: plan.md:1128 ("Confirm every ## Done when row in the issue has a test naming it") is unchecked while the issue's Plan is fully ticked, and the row that names its own test — "the token AFTER the mark still carries the style it had ... needs a test that inspects the style after the span: stripping escapes is exactly what hides a lost one" — is unpinned: replacing both "sgrOff + style.resume()" writes in paintMarks with bare sgrOff leaves the whole suite green. The second unpinned row is the dragged-phrase admission rule, which the Critical above shows the code contradicts.
- **BR-23** [Important] `readme-surface-undocumented` README's paste section is now wrong and the whole read-along surface — click, drag, Enter-asks, the nudge, admission — is undocumented
  This is the 2nd finding in family `readme-surface-undocumented` (BR-5 was the 1st), so do NOT patch the paragraph alone. The rule: a gesture or key the window changes is a README row, and the enumeration is derivable — RegionKind x regionPlaysAudio/regionUnderlines for clicks and replKind for Enter, the same derivation TestAtlasDescribesEveryRegionKind already runs against the atlas; extend that guard to README.md. Sites at HEAD: README.md:76-82 says "The whole paste arrives at once and goes in at the cursor; newlines and tabs inside it become spaces", true only for a 1-3 word single-line paste since pasteIsPassage routes everything else to the passage; README.md:439 still says a click "plays the word" everywhere; README.md:441's Enter row does not mention that marks make Enter an ask. Nothing documents click/drag-to-mark, marks-win-over-replay, noteNothingMarked, marks clearing after an ask, or deck admission — the entire feature the issue is named for.
- **BR-24** [Important] `plan-artifact-stale` The durable plan contradicts the tree in five places, and its unticked boxes switch off the two guards that would have caught two of them
  This is the 2nd finding in family `plan-artifact-stale` (BR-4 was the 1st), so do NOT patch the rows one at a time. The rule: a plan's checkboxes and tables are claims about the tree that this repo's guards READ, so leaving them behind disables the guards — tick the boxes and append the Revisions entry at the commit that lands the departure, not at the close. Sites: Chunks 2-5 are entirely unticked though the work shipped; the Architecture paragraph still says the passage "is chrome, not scrollback ... it lives in the screen's existing footer []string channel"; "What this plan does NOT do" still says a passage-word RegionKind was "resolved away in Chunk 3" while RegionPassageWord is shipped; the Core-concepts table claims `selectionFrame.highlightRow | selection_frame.go | modified` for a file this window never touched; Task 2.2 declares TestThePassageSurvivesALookup, which the tree does not have. The guard interaction is the teeth: TestPlanTablesNameEntitiesThatExist exempts `new` rows whenever any "- [ ]" remains (repo_guard_test.go:896), and TestPlanTableStatusMatchesTheChangeWindow skips a row whose file the window did not touch (:1378), so exactly those two claims are invisible.
- **BR-25** [Important] `work-repeated-on-one-path` gatherAskContext runs twice on every passage ask, doubling the deck and learner-model reads and any warning they print
  ask.go:165 computes req := renderAskPrompt(gatherAskContext(d, sess, q, errOut)) and throws it away; ask.go:170 calls gatherAskContext again for the passage renderer. gatherAskContext is the IO step — d.deck.UserModel() and d.deck.Deck() — and it warns on failure, so an unreadable deck prints "define: could not read the deck (...); answering without it" twice to the user. Fix: gather once into a local, then choose the renderer.
- **BR-26** [Minor] `decoration-ignores-color-option` paintMarks emits a 256-colour SGR pair regardless of opt.color, so marking a word under -no-color produces colour
  draw() calls view.SetMarks unconditionally (replraw.go:425) and layoutSelectionFrame applies paintMarks whenever a row has marks (screen.go:622), with no reference to opt.color. Unlike the selection's inverse video, markOn is "\x1b[48;5;24m\x1b[38;5;231m". passageText already honours the flag, so the two halves of the same surface disagree.
- **BR-27** [Minor] `artifact-family-without-removal` Each paste appends one Region per word to screen.regions with no removal path, which the plan's ARCH-FUNERAL note does not cover
  replraw.go:605 writes passageRegions into the screen on every passage paste — roughly 170 entries for a 1000-character passage — and screen.regions is only ever appended to. The plan states "The passage and its marks are in-memory, die with the session, and are replaced wholesale by the next paste", which holds for sess.passage but not for the screen's copy: per-session growth per paste is larger than before this window. Note it and state the bound, or clear the superseded passage's regions (which would also help the stale-region Critical).

## Round 6 — 2026-09-16T23:00:17-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — key.go:74-76 is byte-identical; the Key doc still says Raw is an unmodelled sequence to be ignored while KeyPaste's Raw is sanitised text that Apply inserts.
- BR-11 — not-addressed — The three named sites were repaired, but no forward-direction guard was written and five NEW contradicting sites shipped in this window: passage.go:17-21 ("It is CHROME, not scrollback ... It lives in the footer"), session.go:24 ("pinned in the footer"), replraw.go:558-562 ("the hit test is FooterRowAt plus wordAtCell, and no new RegionKind exists", three lines above code keying on RegionPassageWord), atlas/define.md:313-314 ("Newlines and tabs survive" while sanitisePasteBody turns tabs into spaces and paste_test.go:183 pins that), and the plan's footer sections. currentTruthFiles now binds *_test.go (the backward half), which is real progress; the forward half — every identifier prose names must be declared at HEAD — does not exist.
- BR-13 — not-addressed — enabledModes (rawterm.go:183) is the right list but nothing derives from it beyond the two new guards: three hand-written enter/leave pairs over three bools remain, restore() still hand-calls its teardown in an order the list does not express (list is alt/mouse/paste, teardown is mouse/paste/alt), and key_test.go:431 still hand-concatenates `const inputModes = mouseOn + pasteOn`, so a fourth mode is invisible to it.
- BR-14 — not-addressed — issue:961 still claims "full suite green" for the commit BR-2 proved red, and ## Log still holds exactly one dated section with no per-round boundary-review entry (five rounds have now run). The milestone-collapse Revisions entry mentions "four review rounds" in passing, which is the nearest thing to a record.
- BR-15 — not-addressed — The case and the declaration landed (play_loop.go:571, :617) and TestAPasteDuringASittingIsIgnored pins the behaviour — but the guard, which was the finding, cannot fail for the failure it names. Measured: adding a KeyKind before numKeyKinds leaves TestEveryKeyKindIsDecidedForASitting GREEN, because keyBecomesASittingInput and toInput both default to false, so a kind wired into neither agrees with itself. numPasteExits and numRegionKinds fail closed in this same window; this one fails open.
- BR-16 — addressed — Mutation-verified: deleting sess.enterPaste() (replraw.go:89) reddens TestNewConsoleEnablesEveryMode with "newConsole never enabled bracketed paste". The PTY row TestPTYEveryEnabledModeIsAskedForAndGivenBack exists and loops enabledModes; it compiles under -tags conformance (go vet clean) and skips here because pty.Open is denied.
- BR-18 — addressed — The CLICK path is genuinely fixed and pinned: passageSpanAt gates on ownsBufferLine (session.go:79,99) and TestAStalePassagesRegionsDoNotMarkTheCurrentOne asserts a stale region refuses. The enumerable siblings of the class are raised fresh below rather than re-raised here.
- BR-19 — addressed — Production drags now go through marksForDrag (replraw.go:573) and yield ONE span; the anchor gate is enforced in passageDragLocked (selection_screen.go:177) so a drag from an answer keeps its copy; markedPassageText matches by position so a phrase reaches the wire as one bracket; TestADraggedPhraseIsAdmittedAsAPhraseOrNotAtAll reads both the deck and the recorded prompt. Residual: passage.raw() still has zero consumers anywhere (raised Minor).
- BR-20 — not-addressed — The tab instance is fixed and pinned to ONE outcome (paste.go sanitisePasteBody, paste_test.go:168-185). The rule fix was not written: no classes-by-consumers table test exists, and the second instance stands. Measured at HEAD: newPassage("see <67-cell URL> for more", 30) yields a 67-cell line and the frame draws "https://example.com/a/very/lon" — clipVisible truncates because wrapText cannot break an unbreakable token, so the tail is neither readable nor clickable. That is the operator-reported wrapping bug, half fixed.
- BR-21 — addressed — sharedLevel / sharedAuthority / sharedLanguageGrammar extracted in askctx.go:147-191 and composed by both askSystem and passageSystem; escLeft/escRight extracted and consumed by both the prompt escape and the answer-direction rule. Both goldens now show the identical clauses.
- BR-22 — not-addressed — Half addressed with real evidence: TestTheTokenAfterAMarkKeepsItsStyle is mutation-red (replacing both "sgrOff + style.resume()" writes with bare sgrOff fails it), and the dragged-phrase row now has a test. But the audit asserts pins that do not exist — TestThePassageIsWrittenToTheBufferNotTheFooter and TestAMarkedDeckWordRendersAsAMarkNotAsADeckWord are in no _test.go file — so two rows are still asserted rather than visible, which is the rule's own failure mode. No guard reads workshop/issues/; TestPlanCitesTestsThatExist globs workshop/plans/*-plan.md only, which is where the teeth belong.
- BR-23 — not-addressed — The new "Read along" section (README.md:76-121) is accurate and thorough, and the old paste paragraph is gone. But the derived guard the rule named was not written (TestAtlasDescribesEveryRegionKind still reads only the atlas), and two rows now contradict the new section: README.md:163 "Enter on an empty line | replay the pronunciation" is unconditional, and README.md:169-174 still says "Underlined words are clickable" and "A click on ordinary text does nothing" while passage words are clickable and deliberately NOT underlined.
- BR-24 — not-addressed — Boxes ticked and three of five sites repaired (Architecture paragraph, RegionPassageWord bullet, selectionFrame.highlightRow row removed). Remaining: Integration points row "passage-as-footer" (plan:80) and its bullet (:91); Task 2.2 title plus Step 3 (:591), Task 2.3, and Task 2.4's ticked atlas row (:611) all still say footer; Task 3.2 "Widen highlightRow from one range to a set" (:707) is fully ticked though highlightRow is unchanged at selection_frame.go:208 and this window never touches that file; plan:579 and :583 declare TestThePassageRendersWithDeckColour and TestThePassageIsWrittenToTheBufferNotTheFooter, neither in the tree. And the rule's own second clause was not followed: the footer-to-buffer reversal was applied by OVERWRITING the Architecture paragraph, with no Revisions entry appended.
- BR-25 — addressed — ask.go:167 gathers once into askCtx and both renderers take it; the double deck/learner-model read and the duplicated warning are gone. Residual (Minor below): renderAskPrompt itself still runs twice on the passage path, purely.
- BR-26 — not-addressed — replraw.go:425 calls SetMarks(markCellRanges(...)) and screen.go:622-624 applies paintMarks whenever a row has marks, with no reference to opt.color anywhere on the path; markOn is "\x1b[48;5;24m\x1b[38;5;231m". passageText still honours the flag, so the two halves of the same surface disagree under -no-color.
- BR-27 — not-addressed — replraw.go:609 still writes passageRegions into screen.regions on every passage paste with no removal path, and the plan's ARCH-FUNERAL paragraph is unchanged — it still says "no removal path needed", which holds for sess.passage and not for the screen's copy. Clearing a superseded passage's regions would also shrink the new Critical's blast radius.

### Raised

- **BR-28** [Critical] `observation-outlives-its-subject` The live-passage gate is on the click path only — a drag in a superseded passage marks the CURRENT one, and hasPassage never expires
  This is the 2nd finding in family `observation-outlives-its-subject`, so do NOT patch the drag call alone. The rule that covers both sites: the session must answer ONE question — "is the live passage reachable at this point?" — and every consumer of passage state routes through it. The enumeration is three long and two are ungated. (1) `passageSpanAt` (session.go:99) gates on `ownsBufferLine` — correct, and BR-18's fix. (2) `passageCell` (session.go:89) CLAMPS instead of refusing, and `passageDragLocked` (selection_screen.go:177) only checks that the anchor row carries some `RegionPassageWord`, which a superseded passage's rows still do because `screen.regions` is never pruned. Measured at HEAD with two passages, drag anchored on the old one's buffer row 10 against a current passage at base 40: `passageCell(10,0) = {line:0 col:0}` and `marksForDrag` returns one span — MARKED "zulu yankee xray" of the current passage. Those words are then bracketed in the next prompt and, on a dictionary hit, admitted to the deck as durable EventMarked records; the copy the reader asked for is swallowed by the `continue` at replraw.go:576. (3) `lineState().hasPassage` (session.go:50) treats "a passage was once pasted" as permanent authority over Enter. Measured: with `current` set a bare Enter is `cmdReplay`; after one paste it is `cmdNothing`/`noteNothingMarked` and never returns to replay for the rest of the session — long after the passage has scrolled away, while the nudge points at something possibly off-screen. That decision was right when the passage was pinned footer chrome; the footer-to-buffer reversal changed the surface's lifetime and the predicate was not re-derived. Fix: `passageCell` returns `(passageCell, bool)` and refuses a row outside `ownsBufferLine`; `hasPassage` means the live passage is on screen. Pin with a two-passage drag regression through runEditor and one asserting replay returns once the passage is gone. Note `pointerClick.line` cannot serve as the gate as written: resolvePointerLocked (selection_screen.go:116) overwrites it from `click.point`, which is the zero value on the drag path, so it carries row 0's buffer line and `hasRegion` can be spuriously true — hasRegion/hasDrag/footer/retry/line is a five-field constellation whose legal combinations are unwritten and should collapse into a tagged variant. ARCH-ORDER, ARCH-PURPOSE.
- **BR-29** [Important] `production-seam-untested` Deleting the cmdAskPassage branch in runEditor leaves the suite green, and replKind has no sentinel so replLines silently has no case for it
  This is the 3rd finding in family `production-seam-untested` (BR-3, BR-16), so do NOT just add a test for this line. Measured: removing the whole `if cmd.kind == cmdAskPassage { askInSession(...) }` block at replraw.go:667-675 leaves `go test ./cmd/define/...` green apart from the two pty-denied rows — the one line that turns "Enter with marks" into an actual model call is unpinned. Every ask test builds `question{passage: ...}` by hand (passageprompt_test.go:171-340) and TestABlankLineWithMarksAsksAboutThePassage stops at parseREPLLine, so nothing joins the decision to the effect. The rule, which is BR-16's one altitude up: a decision table's kinds are a registry, so each kind's disposition in each loop is DECLARED and DERIVED, not hand-written per loop. Second site proving it is a class: `replLines` (repl.go:384) has no case for cmdAskPassage at all — unreachable today only because the piped loop never decodes a paste — and `replKind` (repl.go:45) has no `numReplKinds`, which is why nothing forced the question. This window makes exactly that move three times (numPasteExits, numRegionKinds, numKeyKinds); make it a fourth and drive both loops from it, with the guard failing CLOSED (see the BR-15 disposition for the failure mode to avoid).
- **BR-30** [Minor] `tested-entity-not-wired` passage.raw() has zero consumers anywhere in the tree
  passage.go:75. BR-19 named it and it survived the round: no production caller and no test caller. Delete it rather than keeping it for symmetry with lineCount/line/spans/text, all of which are used.

## Open findings

- **BR-8** [Minor] `doc-contradicts-type` Key struct doc still says Raw is an unmodelled sequence to be ignored, not inserted
- **BR-11** [Important] `doc-contradicts-type` Prose committed in this window asserts M2 behaviour and names symbols the tree does not declare
- **BR-13** [Minor] `repeated-shape-not-extracted` enterPaste/leavePaste is the third copy of the same terminal-mode pair, and restore's ordering is still undeclared
- **BR-14** [Minor] `issue-row-stale` The issue Log records no boundary-review round and its "full suite green" claim was false at the commit it describes
- **BR-15** [Important] `enum-grows-past-consumers` A paste during a review sitting is silently dropped — toInput has no case for KeyPaste, and this window turned mode 2004 on for that surface too
- **BR-20** [Important] `boundary-parses-partial-class` A tab survives the paste boundary but counts as one cell, so clicks land on the wrong word; an unbreakable token overflows the wrap
- **BR-22** [Important] `decided-behaviour-unpinned` The Done-when audit was not run: the style-after-the-mark regression is mutation-green, and the dragged-phrase row cannot hold as written
- **BR-23** [Important] `readme-surface-undocumented` README's paste section is now wrong and the whole read-along surface — click, drag, Enter-asks, the nudge, admission — is undocumented
- **BR-24** [Important] `plan-artifact-stale` The durable plan contradicts the tree in five places, and its unticked boxes switch off the two guards that would have caught two of them
- **BR-26** [Minor] `decoration-ignores-color-option` paintMarks emits a 256-colour SGR pair regardless of opt.color, so marking a word under -no-color produces colour
- **BR-27** [Minor] `artifact-family-without-removal` Each paste appends one Region per word to screen.regions with no removal path, which the plan's ARCH-FUNERAL note does not cover
- **BR-28** [Critical] `observation-outlives-its-subject` The live-passage gate is on the click path only — a drag in a superseded passage marks the CURRENT one, and hasPassage never expires
- **BR-29** [Important] `production-seam-untested` Deleting the cmdAskPassage branch in runEditor leaves the suite green, and replKind has no sentinel so replLines silently has no case for it
- **BR-30** [Minor] `tested-entity-not-wired` passage.raw() has zero consumers anywhere in the tree
