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
    - "n": 6
      timestamp: "2026-08-26T16:39:10-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: addressed
          note: Verified by revert — stubbing tokenStillOpen to false reddens all four rows of TestHighlightWriterByteAtATimeMatchesOneCall.
          round: 6
        - id: BR-14
          disposition: addressed
          note: Verified by revert — deleting d.vocab.Load() reddens all three entry-path rows plus TestEditorLoopHighlightsADeckWordOnScreen.
          round: 6
        - id: BR-15
          disposition: addressed
          note: Verified by mutation — routing the head token through opt.prose reddens TestTheHeadwordLineIsNotHighlighted.
          round: 6
        - id: BR-16
          disposition: not-addressed
          note: Core-concepts rows and contract rule 4 are corrected; Task 5/6/7 Files blocks, the main.go:488 injection point, the fuzz target name and Task 7 Step 2's invariant location still name what does not exist.
          round: 6
        - id: BR-17
          disposition: addressed
          note: stripEscapes and onlyPhraseGap are gone; phraseGapOrEmpty delegates to phraseGap; a sweep of cmd/define's function list finds no remaining near-duplicate pair.
          round: 6
        - id: BR-18
          disposition: not-addressed
          note: The crlfWriter line was fixed but the window sweep it demanded was not run — atlas/define.md:498 now claims highlighting wraps the rendered string rather than reaching into Render, the opposite of what this commit built.
          round: 6
        - id: BR-19
          disposition: addressed
          note: Working tree is clean; no zz_probe file in cmd/define.
          round: 6
        - id: BR-20
          disposition: not-addressed
          note: The colour-ON half landed; the hits-greater-than-zero vacuity guard did not — measured 6 of 32 corpus entries highlight at HEAD.
          round: 6
        - id: BR-21
          disposition: addressed
          note: highlightwriter_test.go:287 now asserts absence of any escape byte.
          round: 6
        - id: BR-22
          disposition: addressed
          note: newHighlightWriter nils the vocabulary when on is empty, so a direct M3 caller inherits the guard.
          round: 6
      findings:
        - id: BR-23
          severity: Important
          title: decidedEnd's no-token branch still releases bytes that can grow, so a chunk splitting a word-initial multi-byte rune loses the match
          detail: |-
            This is the 2nd finding in family `release-only-what-cannot-change` — do NOT fix
            only this instance. THE RULE: every release path in decidedEnd must consult the
            same "can the tail still grow?" predicate. There are two, and only one calls
            tokenStillOpen: when wordRuns(region) is empty (highlightwriter.go:187) the
            function returns len(region) for anything that is not a pure phrase gap, so a
            region holding only non-word bytes plus an incomplete rune is released.
            Measured at HEAD: writeChunks(vocab("uber-with-umlaut"), "!<word>") highlights,
            writeChunks(..., "!\xc3", "\xbcber") does not; likewise "hi!"|" "|"\xc3"|"\xbcber now".
            Bytes survive, the match does not, and the property that should quantify over
            this is blind because fuzzDeck reproduces the tokenizer's character CLASSES but
            not their POSITIONS — cafe is the only multi-byte entry and its multi-byte rune
            is word-final, so no exec count reaches a word-initial one. The enumeration to
            write is class x position (initial / medial / final) for each word-character
            class, applied to both the derived deck and the byte-at-a-time table.
          family: release-only-what-cannot-change
          round: 6
        - id: BR-24
          severity: Important
          title: Four behaviours added this window are pinned by nothing, including the M2 Done-when's own enclosing-style resume
          detail: |-
            This is the 7th finding in family `behaviour-claimed-without-a-failing-test` —
            do NOT fix only this instance. Measured survivals at HEAD: (1) render.go:175,
            replacing the p.ex base with "" passes the whole suite while changing production
            bytes — correct output is \x1b[3;32m"an \x1b[1;32mobsequious\x1b[0m\x1b[3;32m
            smile followed"\x1b[0m and the mutant drops the resume, leaving the rest of the
            example unstyled; sgrState.base dies with the same mutation. (2) sgr.go:14,
            maxOpenSGR has no test at all. (3) vocab.go:157, deleting !opt.color from
            vocabularyFor passes — the whole deck is read under -no-color, which is exactly
            M1 round 3's io-for-a-disabled-feature finding, now unpinned again after the
            refactor moved it. THE RULE, in the shape this window needs: the prior
            enumerations covered hops that change OUTPUT, and every survivor here is either a
            wiring argument (base) or a guard whose only effect is the ABSENCE of work. So
            the enumeration is: for each behaviour this diff's comments claim, name the
            observation that would falsify it — output bytes at the production site for
            wiring, and a counting/spying double for guards. The package already has
            countingDeck (vocab_test.go:199) doing precisely this for "read the deck once";
            the colour gate can reuse it verbatim.
          family: behaviour-claimed-without-a-failing-test
          round: 6
        - id: BR-25
          severity: Minor
          title: highlightText has zero call sites after the per-region refactor, while atlas and plan both name it as the definition path
          detail: |-
            highlightwriter.go:257. RenderOpts.prose -> highlightRegion replaced it; go vet
            does not flag unused functions, so it survives silently. Same rule as the earlier
            dead-scaffolding instances, widened to production code: when a refactor replaces a
            helper, the helper goes with it, and so do the docs that name it.
          family: dead-test-scaffolding
          round: 6
        - id: BR-26
          severity: Minor
          title: admitsHighlight's region table lists 10 regions; Render produces more
          detail: |-
            This is the 3rd finding in family `feature-leaks-across-namespace` — do NOT fix
            only this instance. render.go:47's table omits HeadHomograph, HeadOther (which
            carries real prose, e.g. "read verb (past and past participle read | red |)") and
            the block label at render.go:130. All three are withheld by construction today, so
            behaviour is correct; the enumeration that is supposed to force a decision is a
            subset of the regions, and nothing makes a newly added region declare itself. THE
            RULE: the table is only a decision procedure if it is complete and something
            fails when a region is missing from it — enumerate from Render's emit sites, and
            pin the withholds with one assertion that a highlight appears only inside admitted
            regions.
          family: feature-leaks-across-namespace
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-26T17:05:08-07:00"
      agent: claude
      dispose:
        - id: BR-16
          disposition: not-addressed
          note: 'Plan Task 5/6 still say Test: cmd/define/highlight_test.go, and Task 6 Step 2 still promises caller-unit counts.'
          round: 7
        - id: BR-18
          disposition: addressed
          note: atlas rule 4 now reads "crlfWriter, which M3 will nest this inside"; swept the rest of this window's atlas/README claims, all true at HEAD except the region count raised below.
          round: 7
        - id: BR-20
          disposition: not-addressed
          note: Colour-ON half fixed (Color:true+Vocab; 4 of 32 entries exercise resume); the hits>0 guard is still absent — an unmatchable deck leaves the test green.
          round: 7
        - id: BR-23
          disposition: addressed
          note: Verified by revert — both the targeted test and the byte-at-a-time table redden; 2.47M-exec fuzz plus an independent 284k-exec differential property clean.
          round: 7
        - id: BR-24
          disposition: addressed
          note: All three mutations verified red — p.ex base to "", maxOpenSGR cap disabled, !opt.color deleted.
          round: 7
        - id: BR-25
          disposition: addressed
          note: highlightText deleted; zero references anywhere in the tree, docs corrected.
          round: 7
        - id: BR-26
          disposition: addressed
          note: Table complete and derived-test enforced; leak mutations reproduce the claimed 59/29/11 exactly.
          round: 7
      findings:
        - id: BR-27
          severity: Minor
          title: admitsHighlight enumerates 15 regions; render.go, atlas and lessons.md say thirteen or ten
          detail: |-
            This is the 3rd finding in family `atlas-claims-unbuilt-surface`. Do NOT fix
            only this instance. render.go:48 says "Render emits thirteen", atlas/define.md:498
            says "thirteen regions", lessons.md:1138 says "the ten-region admit/withhold table";
            the table has 15 rows (the round-2 correction also SPLIT two rows, which the
            arithmetic missed). THE RULE: a count written in prose beside an enumeration is a
            second source of truth that nothing checks and that drifts on the next edit — the
            derived test is the record, so the number should be deleted rather than corrected.
          family: atlas-claims-unbuilt-surface
          round: 7
        - id: BR-28
          severity: Minor
          title: A highlight resumes base after a reset that arrived inside the region, restyling neighbouring plain text
          detail: |-
            sgr.go:48 clears `open` on a reset but never `base`, so a region carrying its own
            reset (what prettyPronunciations emits inside an example) resumes to the enclosing
            style for text that is unstyled without highlighting. Measured: highlightRegion of
            "foo \x1b[35m/aI/\x1b[0m bar known baz" with base \x1b[3;32m returns
            "... \x1b[1;32mknown\x1b[0m\x1b[3;32m baz", where " baz" is plain in the no-highlight
            render. Escape-stripped-equal, so TestHighlightingLosesNothing cannot see it. Decide
            once whether an inner reset also clears base, and pin it — M3's model output will
            carry resets routinely.
          family: sgr-resume-outlives-an-inner-reset
          round: 7
        - id: BR-29
          severity: Minor
          title: fuzzDeck has no phrase longer than two tokens, so decidedEnd's hold arithmetic is never fuzzed at maxWords >= 3
          detail: |-
            highlightwriter_test.go:207. The class-x-position table added this round covers
            character classes and their positions but not phrase LENGTH, which is the input
            MaxPhraseWords feeds straight into `k := len(toks) - maxWords` and the straddle
            pull-back. Not a live bug — I ran the axis independently (decks derived from each
            text's own 1..4-token windows, byte-at-a-time vs one-call, 284k execs) and it is
            clean — but it is the same "the deck is input too" rule one notch wider, and
            `in spite of` is a realistic entry. One line in fuzzDeck plus a seed.
          family: fuzz-fixture-axis-missing
          round: 7
      boundary: M2
      blocked: false
    - "n": 8
      timestamp: "2026-08-26T17:41:46-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: not-addressed
          note: 'Two of three seams wrap warnTo; capture.go:132 still writes "define: " itself, and warnTo''s own comment (vocab.go:130) claims to be the one place it is written.'
          round: 8
        - id: BR-10
          disposition: addressed
          note: Verified by revert — dropping !opt.color from vocabularyFor reddens TestNoColourReadsNoDeck and TestStreamedAnswerCarriesNoEscapesWithoutColour.
          round: 8
        - id: BR-11
          disposition: addressed
          note: nil is the single representation; no memVocabulary{} fallback remains in production and the unreachable replraw.go guard is gone.
          round: 8
        - id: BR-12
          disposition: addressed
          note: Verified by revert — neutering highlightSetFor's parseCommandLine test reddens TestACommandLineIsNotHighlighted.
          round: 8
        - id: BR-16
          disposition: not-addressed
          note: 4th round. plan:270/:282 still name highlight_test.go, plan:286 still promises caller-unit counts, plan:70 still says ask.go:160, and Task 8 Steps 4 and 6 are ticked while undelivered (2 of 5 paths tabled; the Step 6 mutation reddens nothing).
          round: 8
        - id: BR-20
          disposition: not-addressed
          note: 3rd round for the vacuity guard — measured 6 of 32 corpus entries highlight, and swapping the deck for an unmatchable word leaves TestHighlightingLosesNothing green.
          round: 8
        - id: BR-27
          disposition: addressed
          note: Counts deleted from render.go and atlas/define.md; lessons.md:1139's "ten-region" is the lesson quoting its own mistake, which is the record rather than a restatement.
          round: 8
        - id: BR-28
          disposition: addressed
          note: Verified by revert — leaving base set on a reset reddens TestAnInnerResetClearsTheEnclosingStyle and TestHighlightAfterAnInnerResetDoesNotRestylePlainText.
          round: 8
        - id: BR-29
          disposition: addressed
          note: fuzzDeck.MaxPhraseWords() measured at 3 with seeds; 1.12M execs clean this round.
          round: 8
      findings:
        - id: BR-30
          severity: Important
          title: M3 added a render surface and the entry-path enumeration written to prevent exactly this was not widened
          detail: |-
            This is the 8th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
            fix only this instance. Measured at HEAD: replacing vocabularyFor(d, opt) in runAsk
            (ask.go:171) with an inline equivalent that keeps the nil/colour gate but drops
            d.vocab.Load() passes the ENTIRE suite. In production that mutant means a streamed
            answer never highlights on piped stdin or one-shot, since only runEditor loads —
            BR-14's shipped Critical, one surface over. It survives because every askhighlight
            test injects a pre-filled memVocabulary and so begins after the hop that fills it.
            THE RULE, widened: the enumeration's axis is entry path x RENDER SURFACE, not entry
            path alone. TestEveryEntryPathHighlightsDefinitions (vocab_test.go:242) covers 3 of
            6 cells while atlas/define.md:517 calls it the guard for "every render path". Each
            surface needs at least one row driven with the dependency in its real initial state;
            verified in both directions that swapping vocab(word) for an unloaded
            newStoreVocabulary passes on HEAD and kills the mutant.
          family: behaviour-claimed-without-a-failing-test
          round: 8
        - id: BR-31
          severity: Minor
          title: An assertion guarded on the run's own output never fires, and two comments claim what it does not pin
          detail: |-
            This is the 9th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
            fix only this instance. askhighlight_test.go:143's only check sits behind
            `tc.cancel && got != ""`, and with an already-cancelled context runAsk returns before
            any delta, so out is empty (measured: code=0 out="" len=0) and the "interrupted
            mid-stream" row asserts nothing — while the test's own comment says the table pins
            that no path leaves text dangling. Same file, :163: "the capture's final characters,
            which only a flush can emit" is refuted by mutation — deleting `defer hw.Flush()`
            leaves that test green because the trailing Fprintln emits them. THE RULE the family
            had not yet named: an assertion guarded on the run's OWN output is not an assertion
            until something proves the guard fires. Measured prevalence: 4 output-conditional
            assertions in cmd/define; three guard on a fixture the author controls and are
            legitimate, this one is the only vacuous one. The package already owns the fix idiom
            at 5 sites (highlightwriter_test.go:328/433/483, invariant_test.go:79,
            dict_fake_test.go:69) — a t.Fatal when the observable is empty.
          family: behaviour-claimed-without-a-failing-test
          round: 8
        - id: BR-32
          severity: Minor
          title: func max in askhighlight_test.go shadows the Go builtin across the whole package's test build
          detail: |-
            This is the 3rd finding in family `copy-pasted-helper`. Do NOT fix only this
            instance. askhighlight_test.go:66 redeclares max(a, b int); deleting it leaves
            `go vet ./cmd/define/` clean (verified), so it is pure shadowing on Go 1.26. Eight
            other call sites now resolve to it — invariant_test.go:43,95, live_property_test.go
            :51,81,101, render_test.go:235, editorloop_test.go:155 — while min beside them still
            resolves to the builtin, so the package's two halves of the same idiom now come from
            different places. THE RULE needs one word: the language's own builtins are part of
            what you grep before adding a helper.
          family: copy-pasted-helper
          round: 8
        - id: BR-33
          severity: Minor
          title: Every error highlightWriter is designed to report is discarded by its only production caller
          detail: |-
            ask.go:171. `fmt.Fprint(out, delta)` ignores its error and `defer hw.Flush()`
            discards its return, while the writer poisons on first failure by contract
            (highlightwriter.go:120) — so one downstream failure silently drops the REST of an
            answer with no message, where before M3 it would have lost one delta. Practically
            unreachable with an os.File, but it is exactly the question M2 round 3 left as a note
            for Task 8 Step 4 ("what does the user see when the writer is already poisoned") and
            nothing in the window records a decision. Same defer, related: on the `default:` path
            the error line reaches errOut BEFORE the deferred flush pushes held text to out, so
            the last partial word lands after the error rather than before it.
          family: contract-error-unread-by-consumer
          round: 8
        - id: BR-34
          severity: Minor
          title: The atlas highlight section never mentions highlightSetFor, the command-namespace withhold
          detail: |-
            highlight.go:173 withholds the vocabulary on a command line, so a deck word named
            like a command is not green inside "/history 7" — a real user-visible rule shipped at
            M1 round 3. atlas/define.md's "Highlighting the words you are learning" section
            documents every other rule in the feature and omits this one. New slug rather than
            atlas-claims-unbuilt-surface: that family names prose asserting surface that does not
            exist; this is the inverse, and the two need different slugs to match a recurrence.
          family: atlas-omits-a-shipped-rule
          round: 8
      blocked: true
    - "n": 9
      timestamp: "2026-08-26T18:07:12-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: not-addressed
          note: 'Unchanged since round 8 — capture.go:132 still writes "define: " itself while warnTo''s comment (vocab.go:130) claims to be the one place it is written.'
          round: 9
        - id: BR-16
          disposition: not-addressed
          note: 5th round. Files blocks fixed; plan:70 still says ask.go:160 (actual 171), plan:286 still promises caller-unit counts, and Task 8 Step 6's mutation claim is false — deleting the defer reddens nothing.
          round: 9
        - id: BR-20
          disposition: not-addressed
          note: 4th round for the vacuity guard — measured 6 of 32 corpus entries highlight, and swapping the deck for an unmatchable word leaves TestHighlightingLosesNothing green.
          round: 9
        - id: BR-30
          disposition: addressed
          note: Verified by mutation — an inline gate keeping nil/colour but dropping Load reddens TestEveryEntryPathHighlightsAnswers on exactly the one-shot and piped-stdin rows.
          round: 9
        - id: BR-31
          disposition: addressed
          note: wantEmpty is asserted unconditionally with a t.Fatal on empty output; probed that the surviving output-conditional guard fires on the clean row; the refuted comment is corrected in place.
          round: 9
        - id: BR-32
          disposition: addressed
          note: No func max/min remains in cmd/define; max(0, len(deltas)-1) resolves to the Go 1.26 builtin; vet, build and suite clean.
          round: 9
        - id: BR-33
          disposition: not-addressed
          note: The code change is right but nothing pins it — reverting to `defer hw.Flush()` leaves the ENTIRE suite green (full run). A 12-line test with the package's existing shortWriter kills the mutant, verified both ways. The errOut-before-flush ordering half is also unchanged and unrecorded.
          round: 9
        - id: BR-34
          disposition: not-addressed
          note: atlas/define.md:422-545 still never mentions highlightSetFor or the command-namespace withhold; the close commit added a second omission in the same family by recording the poisoned-writer decision only in the plan's Revisions, not in the atlas Flush paragraph where it belongs.
          round: 9
      findings:
        - id: BR-35
          severity: Minor
          title: The atlas sentence the close commit itself edited is malformed prose
          detail: |-
            atlas/define.md:513-515 now reads `One function now owns "loaded, and only with
            colour", and` / `The enumeration that` / `guards it is **entry path x render
            surface**...` — a dangling conjunction followed by an orphaned capitalised clause,
            broken mid-phrase across lines. The text it replaced was well-formed; the
            replacement was written but not read back. New slug rather than
            atlas-claims-unbuilt-surface or atlas-omits-a-shipped-rule: those two name what the
            prose CLAIMS (surface that does not exist, a rule left out); this is a mechanically
            malformed edit, and would recur in any doc file with any content.
          family: doc-edit-not-read-back
          round: 9
        - id: BR-36
          severity: Minor
          title: A third-party import sits inside the stdlib group, and nothing in the repo enforces import grouping
          detail: |-
            This is the 2nd finding in family `unformatted-source`. Do NOT fix only this
            instance. askhighlight_test.go:3-13 puts github.com/xianxu/tools/cmd/define/store
            between encoding/json and strings, and splits the stdlib block for no reason; every
            other file in the package groups stdlib then third-party (vocab_test.go:3,
            highlightwriter_test.go:3). gofmt does not catch this and goimports would. THE RULE:
            both instances of this family were caught by a reviewer typing the command by hand —
            verified that no Makefile, Makefile.workflow, scripts/ or CI target runs gofmt or
            goimports anywhere in the repo. Put `gofmt -l` and `goimports -l` in a gate; fixing
            the file again leaves the next instance to the next reviewer.
          family: unformatted-source
          round: 9
        - id: BR-37
          severity: Minor
          title: The interrupt-with-held-text exit path lost its row when BR-31's fix renamed it, and nothing noticed
          detail: |-
            This is the 10th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
            fix only this instance. The row `interrupted mid-stream` became `cancelled before any
            delta` — honest, and strictly weaker: an already-cancelled context returns before any
            delta, so held text and cancellation never interact. The path Task 8 Step 4 names by
            name (askScoped cancels mid-stream; ctx.Err() != nil with answer.Len() > 0, writing
            Fprintln through the held writer) is now exercised only at options{} — colour off,
            vocabularyFor nil, nothing held — by askrun_test.go:590. Measured with color:true and
            Stall:true it is CORRECT (43 bytes, ends in a newline, highlight intact), so this is
            coverage rather than a bug, and it is ~20 lines using syncBuf and Stall, both already
            in the package. THE RULE the family had not yet named: a fix that makes a test HONEST
            can also make it NARROWER, and the narrowing is silent because every remaining
            assertion still passes. When a row is renamed or a guard tightened, state what it
            stopped covering. Measured prevalence this window: 1 of 3 rewritten rows lost a case.
          family: behaviour-claimed-without-a-failing-test
          round: 9
        - id: BR-38
          severity: Minor
          title: MaxPhraseWords is inflated by deck keys that can never match, and M3 made that streaming latency
          detail: |-
            vocab.go:76. phraseGap admits only spaces and tabs, so a key with internal
            punctuation is structurally unmatchable — TestAPunctuatedKeyIsNotMatchable pins
            exactly that — yet Add still counts its tokens into the look-ahead budget
            decidedEnd holds against. Measured on "the quick brown fox jumps": single-word deck
            holds 5 bytes, adding the unmatchable `e.g.` holds 9, adding the unmatchable
            `rock 'n' roll` holds 15, so the reader waits on words for a match that cannot
            occur. The comment at vocab.go:76 says such a key "is currently UNMATCHABLE
            whichever way it is counted" — true at M1, false since M3 wired the stream, and
            nothing revisited it. THE RULE: a bound derived from an input set must be derived
            from the subset that can actually exercise it — bump maxWords only for keys whose
            own tokens rejoin.
          family: bound-includes-unusable-input
          round: 9
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

## Round 6 — 2026-08-26T16:39:10-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — addressed — Verified by revert — stubbing tokenStillOpen to false reddens all four rows of TestHighlightWriterByteAtATimeMatchesOneCall.
- BR-14 — addressed — Verified by revert — deleting d.vocab.Load() reddens all three entry-path rows plus TestEditorLoopHighlightsADeckWordOnScreen.
- BR-15 — addressed — Verified by mutation — routing the head token through opt.prose reddens TestTheHeadwordLineIsNotHighlighted.
- BR-16 — not-addressed — Core-concepts rows and contract rule 4 are corrected; Task 5/6/7 Files blocks, the main.go:488 injection point, the fuzz target name and Task 7 Step 2's invariant location still name what does not exist.
- BR-17 — addressed — stripEscapes and onlyPhraseGap are gone; phraseGapOrEmpty delegates to phraseGap; a sweep of cmd/define's function list finds no remaining near-duplicate pair.
- BR-18 — not-addressed — The crlfWriter line was fixed but the window sweep it demanded was not run — atlas/define.md:498 now claims highlighting wraps the rendered string rather than reaching into Render, the opposite of what this commit built.
- BR-19 — addressed — Working tree is clean; no zz_probe file in cmd/define.
- BR-20 — not-addressed — The colour-ON half landed; the hits-greater-than-zero vacuity guard did not — measured 6 of 32 corpus entries highlight at HEAD.
- BR-21 — addressed — highlightwriter_test.go:287 now asserts absence of any escape byte.
- BR-22 — addressed — newHighlightWriter nils the vocabulary when on is empty, so a direct M3 caller inherits the guard.

### Raised

- **BR-23** [Important] `release-only-what-cannot-change` decidedEnd's no-token branch still releases bytes that can grow, so a chunk splitting a word-initial multi-byte rune loses the match
  This is the 2nd finding in family `release-only-what-cannot-change` — do NOT fix
  only this instance. THE RULE: every release path in decidedEnd must consult the
  same "can the tail still grow?" predicate. There are two, and only one calls
  tokenStillOpen: when wordRuns(region) is empty (highlightwriter.go:187) the
  function returns len(region) for anything that is not a pure phrase gap, so a
  region holding only non-word bytes plus an incomplete rune is released.
  Measured at HEAD: writeChunks(vocab("uber-with-umlaut"), "!<word>") highlights,
  writeChunks(..., "!\xc3", "\xbcber") does not; likewise "hi!"|" "|"\xc3"|"\xbcber now".
  Bytes survive, the match does not, and the property that should quantify over
  this is blind because fuzzDeck reproduces the tokenizer's character CLASSES but
  not their POSITIONS — cafe is the only multi-byte entry and its multi-byte rune
  is word-final, so no exec count reaches a word-initial one. The enumeration to
  write is class x position (initial / medial / final) for each word-character
  class, applied to both the derived deck and the byte-at-a-time table.
- **BR-24** [Important] `behaviour-claimed-without-a-failing-test` Four behaviours added this window are pinned by nothing, including the M2 Done-when's own enclosing-style resume
  This is the 7th finding in family `behaviour-claimed-without-a-failing-test` —
  do NOT fix only this instance. Measured survivals at HEAD: (1) render.go:175,
  replacing the p.ex base with "" passes the whole suite while changing production
  bytes — correct output is \x1b[3;32m"an \x1b[1;32mobsequious\x1b[0m\x1b[3;32m
  smile followed"\x1b[0m and the mutant drops the resume, leaving the rest of the
  example unstyled; sgrState.base dies with the same mutation. (2) sgr.go:14,
  maxOpenSGR has no test at all. (3) vocab.go:157, deleting !opt.color from
  vocabularyFor passes — the whole deck is read under -no-color, which is exactly
  M1 round 3's io-for-a-disabled-feature finding, now unpinned again after the
  refactor moved it. THE RULE, in the shape this window needs: the prior
  enumerations covered hops that change OUTPUT, and every survivor here is either a
  wiring argument (base) or a guard whose only effect is the ABSENCE of work. So
  the enumeration is: for each behaviour this diff's comments claim, name the
  observation that would falsify it — output bytes at the production site for
  wiring, and a counting/spying double for guards. The package already has
  countingDeck (vocab_test.go:199) doing precisely this for "read the deck once";
  the colour gate can reuse it verbatim.
- **BR-25** [Minor] `dead-test-scaffolding` highlightText has zero call sites after the per-region refactor, while atlas and plan both name it as the definition path
  highlightwriter.go:257. RenderOpts.prose -> highlightRegion replaced it; go vet
  does not flag unused functions, so it survives silently. Same rule as the earlier
  dead-scaffolding instances, widened to production code: when a refactor replaces a
  helper, the helper goes with it, and so do the docs that name it.
- **BR-26** [Minor] `feature-leaks-across-namespace` admitsHighlight's region table lists 10 regions; Render produces more
  This is the 3rd finding in family `feature-leaks-across-namespace` — do NOT fix
  only this instance. render.go:47's table omits HeadHomograph, HeadOther (which
  carries real prose, e.g. "read verb (past and past participle read | red |)") and
  the block label at render.go:130. All three are withheld by construction today, so
  behaviour is correct; the enumeration that is supposed to force a decision is a
  subset of the regions, and nothing makes a newly added region declare itself. THE
  RULE: the table is only a decision procedure if it is complete and something
  fails when a region is missing from it — enumerate from Render's emit sites, and
  pin the withholds with one assertion that a highlight appears only inside admitted
  regions.

## Round 7 — 2026-08-26T17:05:08-07:00 (claude) — passed

### Disposed

- BR-16 — not-addressed — Plan Task 5/6 still say Test: cmd/define/highlight_test.go, and Task 6 Step 2 still promises caller-unit counts.
- BR-18 — addressed — atlas rule 4 now reads "crlfWriter, which M3 will nest this inside"; swept the rest of this window's atlas/README claims, all true at HEAD except the region count raised below.
- BR-20 — not-addressed — Colour-ON half fixed (Color:true+Vocab; 4 of 32 entries exercise resume); the hits>0 guard is still absent — an unmatchable deck leaves the test green.
- BR-23 — addressed — Verified by revert — both the targeted test and the byte-at-a-time table redden; 2.47M-exec fuzz plus an independent 284k-exec differential property clean.
- BR-24 — addressed — All three mutations verified red — p.ex base to "", maxOpenSGR cap disabled, !opt.color deleted.
- BR-25 — addressed — highlightText deleted; zero references anywhere in the tree, docs corrected.
- BR-26 — addressed — Table complete and derived-test enforced; leak mutations reproduce the claimed 59/29/11 exactly.

### Raised

- **BR-27** [Minor] `atlas-claims-unbuilt-surface` admitsHighlight enumerates 15 regions; render.go, atlas and lessons.md say thirteen or ten
  This is the 3rd finding in family `atlas-claims-unbuilt-surface`. Do NOT fix
  only this instance. render.go:48 says "Render emits thirteen", atlas/define.md:498
  says "thirteen regions", lessons.md:1138 says "the ten-region admit/withhold table";
  the table has 15 rows (the round-2 correction also SPLIT two rows, which the
  arithmetic missed). THE RULE: a count written in prose beside an enumeration is a
  second source of truth that nothing checks and that drifts on the next edit — the
  derived test is the record, so the number should be deleted rather than corrected.
- **BR-28** [Minor] `sgr-resume-outlives-an-inner-reset` A highlight resumes base after a reset that arrived inside the region, restyling neighbouring plain text
  sgr.go:48 clears `open` on a reset but never `base`, so a region carrying its own
  reset (what prettyPronunciations emits inside an example) resumes to the enclosing
  style for text that is unstyled without highlighting. Measured: highlightRegion of
  "foo \x1b[35m/aI/\x1b[0m bar known baz" with base \x1b[3;32m returns
  "... \x1b[1;32mknown\x1b[0m\x1b[3;32m baz", where " baz" is plain in the no-highlight
  render. Escape-stripped-equal, so TestHighlightingLosesNothing cannot see it. Decide
  once whether an inner reset also clears base, and pin it — M3's model output will
  carry resets routinely.
- **BR-29** [Minor] `fuzz-fixture-axis-missing` fuzzDeck has no phrase longer than two tokens, so decidedEnd's hold arithmetic is never fuzzed at maxWords >= 3
  highlightwriter_test.go:207. The class-x-position table added this round covers
  character classes and their positions but not phrase LENGTH, which is the input
  MaxPhraseWords feeds straight into `k := len(toks) - maxWords` and the straddle
  pull-back. Not a live bug — I ran the axis independently (decks derived from each
  text's own 1..4-token windows, byte-at-a-time vs one-call, 284k execs) and it is
  clean — but it is the same "the deck is input too" rule one notch wider, and
  `in spite of` is a realistic entry. One line in fuzzDeck plus a seed.

## Round 8 — 2026-08-26T17:41:46-07:00 (claude) — BLOCKED

### Disposed

- BR-9 — not-addressed — Two of three seams wrap warnTo; capture.go:132 still writes "define: " itself, and warnTo's own comment (vocab.go:130) claims to be the one place it is written.
- BR-10 — addressed — Verified by revert — dropping !opt.color from vocabularyFor reddens TestNoColourReadsNoDeck and TestStreamedAnswerCarriesNoEscapesWithoutColour.
- BR-11 — addressed — nil is the single representation; no memVocabulary{} fallback remains in production and the unreachable replraw.go guard is gone.
- BR-12 — addressed — Verified by revert — neutering highlightSetFor's parseCommandLine test reddens TestACommandLineIsNotHighlighted.
- BR-16 — not-addressed — 4th round. plan:270/:282 still name highlight_test.go, plan:286 still promises caller-unit counts, plan:70 still says ask.go:160, and Task 8 Steps 4 and 6 are ticked while undelivered (2 of 5 paths tabled; the Step 6 mutation reddens nothing).
- BR-20 — not-addressed — 3rd round for the vacuity guard — measured 6 of 32 corpus entries highlight, and swapping the deck for an unmatchable word leaves TestHighlightingLosesNothing green.
- BR-27 — addressed — Counts deleted from render.go and atlas/define.md; lessons.md:1139's "ten-region" is the lesson quoting its own mistake, which is the record rather than a restatement.
- BR-28 — addressed — Verified by revert — leaving base set on a reset reddens TestAnInnerResetClearsTheEnclosingStyle and TestHighlightAfterAnInnerResetDoesNotRestylePlainText.
- BR-29 — addressed — fuzzDeck.MaxPhraseWords() measured at 3 with seeds; 1.12M execs clean this round.

### Raised

- **BR-30** [Important] `behaviour-claimed-without-a-failing-test` M3 added a render surface and the entry-path enumeration written to prevent exactly this was not widened
  This is the 8th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
  fix only this instance. Measured at HEAD: replacing vocabularyFor(d, opt) in runAsk
  (ask.go:171) with an inline equivalent that keeps the nil/colour gate but drops
  d.vocab.Load() passes the ENTIRE suite. In production that mutant means a streamed
  answer never highlights on piped stdin or one-shot, since only runEditor loads —
  BR-14's shipped Critical, one surface over. It survives because every askhighlight
  test injects a pre-filled memVocabulary and so begins after the hop that fills it.
  THE RULE, widened: the enumeration's axis is entry path x RENDER SURFACE, not entry
  path alone. TestEveryEntryPathHighlightsDefinitions (vocab_test.go:242) covers 3 of
  6 cells while atlas/define.md:517 calls it the guard for "every render path". Each
  surface needs at least one row driven with the dependency in its real initial state;
  verified in both directions that swapping vocab(word) for an unloaded
  newStoreVocabulary passes on HEAD and kills the mutant.
- **BR-31** [Minor] `behaviour-claimed-without-a-failing-test` An assertion guarded on the run's own output never fires, and two comments claim what it does not pin
  This is the 9th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
  fix only this instance. askhighlight_test.go:143's only check sits behind
  `tc.cancel && got != ""`, and with an already-cancelled context runAsk returns before
  any delta, so out is empty (measured: code=0 out="" len=0) and the "interrupted
  mid-stream" row asserts nothing — while the test's own comment says the table pins
  that no path leaves text dangling. Same file, :163: "the capture's final characters,
  which only a flush can emit" is refuted by mutation — deleting `defer hw.Flush()`
  leaves that test green because the trailing Fprintln emits them. THE RULE the family
  had not yet named: an assertion guarded on the run's OWN output is not an assertion
  until something proves the guard fires. Measured prevalence: 4 output-conditional
  assertions in cmd/define; three guard on a fixture the author controls and are
  legitimate, this one is the only vacuous one. The package already owns the fix idiom
  at 5 sites (highlightwriter_test.go:328/433/483, invariant_test.go:79,
  dict_fake_test.go:69) — a t.Fatal when the observable is empty.
- **BR-32** [Minor] `copy-pasted-helper` func max in askhighlight_test.go shadows the Go builtin across the whole package's test build
  This is the 3rd finding in family `copy-pasted-helper`. Do NOT fix only this
  instance. askhighlight_test.go:66 redeclares max(a, b int); deleting it leaves
  `go vet ./cmd/define/` clean (verified), so it is pure shadowing on Go 1.26. Eight
  other call sites now resolve to it — invariant_test.go:43,95, live_property_test.go
  :51,81,101, render_test.go:235, editorloop_test.go:155 — while min beside them still
  resolves to the builtin, so the package's two halves of the same idiom now come from
  different places. THE RULE needs one word: the language's own builtins are part of
  what you grep before adding a helper.
- **BR-33** [Minor] `contract-error-unread-by-consumer` Every error highlightWriter is designed to report is discarded by its only production caller
  ask.go:171. `fmt.Fprint(out, delta)` ignores its error and `defer hw.Flush()`
  discards its return, while the writer poisons on first failure by contract
  (highlightwriter.go:120) — so one downstream failure silently drops the REST of an
  answer with no message, where before M3 it would have lost one delta. Practically
  unreachable with an os.File, but it is exactly the question M2 round 3 left as a note
  for Task 8 Step 4 ("what does the user see when the writer is already poisoned") and
  nothing in the window records a decision. Same defer, related: on the `default:` path
  the error line reaches errOut BEFORE the deferred flush pushes held text to out, so
  the last partial word lands after the error rather than before it.
- **BR-34** [Minor] `atlas-omits-a-shipped-rule` The atlas highlight section never mentions highlightSetFor, the command-namespace withhold
  highlight.go:173 withholds the vocabulary on a command line, so a deck word named
  like a command is not green inside "/history 7" — a real user-visible rule shipped at
  M1 round 3. atlas/define.md's "Highlighting the words you are learning" section
  documents every other rule in the feature and omits this one. New slug rather than
  atlas-claims-unbuilt-surface: that family names prose asserting surface that does not
  exist; this is the inverse, and the two need different slugs to match a recurrence.

## Round 9 — 2026-08-26T18:07:12-07:00 (claude) — BLOCKED

### Disposed

- BR-9 — not-addressed — Unchanged since round 8 — capture.go:132 still writes "define: " itself while warnTo's comment (vocab.go:130) claims to be the one place it is written.
- BR-16 — not-addressed — 5th round. Files blocks fixed; plan:70 still says ask.go:160 (actual 171), plan:286 still promises caller-unit counts, and Task 8 Step 6's mutation claim is false — deleting the defer reddens nothing.
- BR-20 — not-addressed — 4th round for the vacuity guard — measured 6 of 32 corpus entries highlight, and swapping the deck for an unmatchable word leaves TestHighlightingLosesNothing green.
- BR-30 — addressed — Verified by mutation — an inline gate keeping nil/colour but dropping Load reddens TestEveryEntryPathHighlightsAnswers on exactly the one-shot and piped-stdin rows.
- BR-31 — addressed — wantEmpty is asserted unconditionally with a t.Fatal on empty output; probed that the surviving output-conditional guard fires on the clean row; the refuted comment is corrected in place.
- BR-32 — addressed — No func max/min remains in cmd/define; max(0, len(deltas)-1) resolves to the Go 1.26 builtin; vet, build and suite clean.
- BR-33 — not-addressed — The code change is right but nothing pins it — reverting to `defer hw.Flush()` leaves the ENTIRE suite green (full run). A 12-line test with the package's existing shortWriter kills the mutant, verified both ways. The errOut-before-flush ordering half is also unchanged and unrecorded.
- BR-34 — not-addressed — atlas/define.md:422-545 still never mentions highlightSetFor or the command-namespace withhold; the close commit added a second omission in the same family by recording the poisoned-writer decision only in the plan's Revisions, not in the atlas Flush paragraph where it belongs.

### Raised

- **BR-35** [Minor] `doc-edit-not-read-back` The atlas sentence the close commit itself edited is malformed prose
  atlas/define.md:513-515 now reads `One function now owns "loaded, and only with
  colour", and` / `The enumeration that` / `guards it is **entry path x render
  surface**...` — a dangling conjunction followed by an orphaned capitalised clause,
  broken mid-phrase across lines. The text it replaced was well-formed; the
  replacement was written but not read back. New slug rather than
  atlas-claims-unbuilt-surface or atlas-omits-a-shipped-rule: those two name what the
  prose CLAIMS (surface that does not exist, a rule left out); this is a mechanically
  malformed edit, and would recur in any doc file with any content.
- **BR-36** [Minor] `unformatted-source` A third-party import sits inside the stdlib group, and nothing in the repo enforces import grouping
  This is the 2nd finding in family `unformatted-source`. Do NOT fix only this
  instance. askhighlight_test.go:3-13 puts github.com/xianxu/tools/cmd/define/store
  between encoding/json and strings, and splits the stdlib block for no reason; every
  other file in the package groups stdlib then third-party (vocab_test.go:3,
  highlightwriter_test.go:3). gofmt does not catch this and goimports would. THE RULE:
  both instances of this family were caught by a reviewer typing the command by hand —
  verified that no Makefile, Makefile.workflow, scripts/ or CI target runs gofmt or
  goimports anywhere in the repo. Put `gofmt -l` and `goimports -l` in a gate; fixing
  the file again leaves the next instance to the next reviewer.
- **BR-37** [Minor] `behaviour-claimed-without-a-failing-test` The interrupt-with-held-text exit path lost its row when BR-31's fix renamed it, and nothing noticed
  This is the 10th finding in family `behaviour-claimed-without-a-failing-test`. Do NOT
  fix only this instance. The row `interrupted mid-stream` became `cancelled before any
  delta` — honest, and strictly weaker: an already-cancelled context returns before any
  delta, so held text and cancellation never interact. The path Task 8 Step 4 names by
  name (askScoped cancels mid-stream; ctx.Err() != nil with answer.Len() > 0, writing
  Fprintln through the held writer) is now exercised only at options{} — colour off,
  vocabularyFor nil, nothing held — by askrun_test.go:590. Measured with color:true and
  Stall:true it is CORRECT (43 bytes, ends in a newline, highlight intact), so this is
  coverage rather than a bug, and it is ~20 lines using syncBuf and Stall, both already
  in the package. THE RULE the family had not yet named: a fix that makes a test HONEST
  can also make it NARROWER, and the narrowing is silent because every remaining
  assertion still passes. When a row is renamed or a guard tightened, state what it
  stopped covering. Measured prevalence this window: 1 of 3 rewritten rows lost a case.
- **BR-38** [Minor] `bound-includes-unusable-input` MaxPhraseWords is inflated by deck keys that can never match, and M3 made that streaming latency
  vocab.go:76. phraseGap admits only spaces and tabs, so a key with internal
  punctuation is structurally unmatchable — TestAPunctuatedKeyIsNotMatchable pins
  exactly that — yet Add still counts its tokens into the look-ahead budget
  decidedEnd holds against. Measured on "the quick brown fox jumps": single-word deck
  holds 5 bytes, adding the unmatchable `e.g.` holds 9, adding the unmatchable
  `rock 'n' roll` holds 15, so the reader waits on words for a match that cannot
  occur. The comment at vocab.go:76 says such a key "is currently UNMATCHABLE
  whichever way it is counted" — true at M1, false since M3 wired the stream, and
  nothing revisited it. THE RULE: a bound derived from an input set must be derived
  from the subset that can actually exercise it — bump maxWords only for keys whose
  own tokens rejoin.

## Open findings

- **BR-9** [Minor] `copy-pasted-helper` A third near-identical warnf, with the "define: " prefix now written in three places (ARCH-DRY)
- **BR-16** [Important] `plan-record-not-updated` Plan file layout, contract rule 4 and two test names no longer match the code
- **BR-20** [Minor] `behaviour-claimed-without-a-failing-test` The no-data-loss invariant runs only over the colour-OFF render
- **BR-33** [Minor] `contract-error-unread-by-consumer` Every error highlightWriter is designed to report is discarded by its only production caller
- **BR-34** [Minor] `atlas-omits-a-shipped-rule` The atlas highlight section never mentions highlightSetFor, the command-namespace withhold
- **BR-35** [Minor] `doc-edit-not-read-back` The atlas sentence the close commit itself edited is malformed prose
- **BR-36** [Minor] `unformatted-source` A third-party import sits inside the stdlib group, and nothing in the repo enforces import grouping
- **BR-37** [Minor] `behaviour-claimed-without-a-failing-test` The interrupt-with-held-text exit path lost its row when BR-31's fix renamed it, and nothing noticed
- **BR-38** [Minor] `bound-includes-unusable-input` MaxPhraseWords is inflated by deck keys that can never match, and M3 made that streaming latency
