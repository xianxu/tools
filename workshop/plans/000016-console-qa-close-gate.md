---
gate: boundary-review
issue: 16
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-23T15:46:30-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: deleting both loops' forced-"?" cmdAsk branches leaves the whole suite green
          detail: |-
            Verified by mutation: removing `case cmdAsk:` from replLines (repl.go:219)
            and the whole `if cmd.kind == cmdAsk` block from runEditor
            (replraw.go:199-211) passes `go test ./cmd/define/` in full. Every
            loop-level ask test uses the unforced route; routeFor returns "question"
            for cmdAsk without entering a loop. hist.Add(submitted.String()) at
            replraw.go:204 is unpinned by the same gap. The issue's ticked Done-when
            row claims each hatch is exercised by a test that fails when the hatch is
            removed; that holds at the parser only. Add one forced test per loop —
            both pass against today's code, which is the point.
          family: loop-shell-branch-untested
          round: 1
        - id: BR-2
          severity: Important
          title: TestAQuestionIsRecalledByUpArrow passes with history recording removed
          detail: |-
            askroute_test.go:95 asserts stdout contains the question, but the raw
            editor re-renders the line on every keystroke, so the question is in
            stdout from typing alone. Verified: replacing hist.Add(line) at
            replraw.go:292 with a no-op leaves the test green. Task 4's contract row
            "does add the line to editor recall (hist.Add)" has no falsifiable test.
            Assert against an injected History, and pin the recall rendering by
            comparing runs with and without the KeyUp.
          family: test-asserts-nothing
          round: 1
        - id: BR-3
          severity: Important
          title: README update appears missing for the "?" and "\" hatches and question routing
          detail: |-
            README.md documents exactly this class — the "/"-in-column-1 convention
            (README.md:120), the interactive key table (:43-50), and the exit-code
            contract (:111-116) — and is untouched in the window, while atlas,
            fs.Usage and /help were all updated. New user-facing surface a reader
            types: "?" forces a question, "\" forces a lookup, an unrecognised line
            that reads as a question goes to the model, and a question with no model
            configured exits 1.
          family: readme-surface-gate
          round: 1
        - id: BR-4
          severity: Important
          title: the one-shot sends cmdNothing to the dictionary as an empty word
          detail: |-
            main.go:342's default arm handles cmdCommand and cmdAsk and falls through
            to defineOnce for every other kind, but parseREPLLine now returns
            cmdNothing with word=="" for two inputs it never used to produce. Measured:
            `define "?"` and `define "\"` print `define: : no dictionary entry`, exit
            1, and Capture("", false, opt) appends a ReviewEvent{Word: ""} that
            complete() (store/event.go:30) discards at read time — junk in the
            append-only log #8 and #17 fold over. cmd.note is dropped, so the one-shot
            never shows `type a question after "?"`. Handle cmdNothing before
            defineOnce and never pass an empty word to lookupAndRender.
          family: oneshot-kind-coverage
          round: 1
        - id: BR-5
          severity: Minor
          title: the bare-"?" note is printed without eraseLine, unlike its sibling branch
          detail: |-
            replraw.go:215 writes `define: %s\r\n` with the cursor still on the typed
            prompt line, so it renders as `› ?define: type a question after "?"`.
            replayInPlace (replraw.go:264) prefixes eraseLine for the same class of
            message.
          family: raw-mode-message-placement
          round: 1
        - id: BR-6
          severity: Minor
          title: forced and unforced asks render with different vertical spacing
          detail: |-
            The unforced path emits runEditor's pre-submitLine "\r\n" (replraw.go:227)
            in addition to askInSession's own, so its message sits one blank line
            lower than the forced path's. Two routes the plan wanted visually
            identical.
          family: raw-mode-message-placement
          round: 1
        - id: BR-7
          severity: Minor
          title: atlas says single-word lines are never questions; the trailing-"?" arm says otherwise
          detail: |-
            atlas/define.md:488. The "?" arm precedes the len(fields) < 2 check, so
            readsAsQuestion("why?") and readsAsQuestion("sycophanti?") both return
            true (measured). Qualify the sentence or move the arm.
          family: doc-overstates-code
          round: 1
        - id: BR-8
          severity: Minor
          title: hand-rolled contains() where slices.Contains exists
          detail: |-
            question.go:83. The module is go 1.26 and the package's tests already
            import slices (ARCH-DRY).
          family: stdlib-reuse
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-08-23T16:05:39-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: deleting either loop''s cmdAsk branch now reddens the matching forced test; hist.Add(submitted) is pinned too.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Asserted against an injected memHistory; reverting hist.Add(line) in submitLine reddens the unforced subtest.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README documents both hatches, routing, the model requirement and the exit codes; two of its claims are inexact — see the new doc-overstates-code finding.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Exhaustive dispatch at main.go:346; removing it reddens the test. The test's capture half is dead, raised separately under test-asserts-nothing.
          round: 2
        - id: BR-5
          disposition: addressed
          note: eraseLine present at replraw.go:223 and measured on the byte stream; unpinned by any test, folded into the family finding.
          round: 2
        - id: BR-6
          disposition: addressed
          note: 'Measured: forced and unforced asks both emit exactly one leading CRLF plus askInSession''s, identical tails; unpinned, folded into the family finding.'
          round: 2
        - id: BR-7
          disposition: addressed
          note: atlas/define.md now qualifies the sentence and names "why?" explicitly.
          round: 2
        - id: BR-8
          disposition: addressed
          note: question.go:62 uses slices.Contains; the hand-rolled contains() is gone.
          round: 2
      findings:
        - id: BR-9
          severity: Important
          title: '"-raw never asks" is guarded on the unforced route only, so 3 of 6 ask entry points ignore it'
          detail: |-
            This is the 2nd finding in family doc-overstates-code, so the deliverable
            is the rule, not the site. Rule - an absolute stated in README/atlas or in
            a test name must name the enumeration it quantifies over, and every cell
            must be guarded and asserted. Here the enumeration is
            {forced, unforced} x {one-shot, piped loop, raw editor}; the !opt.raw
            guard sits at main.go:425 on the unforced path only. Measured -
            `define -raw "?what is X"` and `echo "?what is X" | define -raw` both
            print the no-model message and exit 1, and runEditor with opt.raw=true on
            "?why" does the same, while TestRawNeverAsks (askroute_test.go:202) stays
            green through all three. README.md:79 states the absolute; main.go:419-424
            names the M2 cost (a network call for a line a script piped in). Same
            family, smaller prevalence - README.md:69 and :76 quote the miss message
            as `not found` where the program prints `no dictionary entry`.
          family: doc-overstates-code
          round: 2
        - id: BR-10
          severity: Important
          title: BR-4's test injects a countingCapturer that withStore discards, so its log-junk assertion is dead
          detail: |-
            This is the 2nd finding in family test-asserts-nothing, so state the rule -
            a test that injects a double must assert the injection took effect, or
            inject at the seam production reads. askroute_test.go:178 sets d.newStore
            on a deps whose capture testDeps already filled (main_test.go:14), and
            withStore only fills nils (main.go:96), so cap is discarded and
            `for _, w := range cap.calls` never iterates. Probe-verified - a panic
            inside countingCapturer.Capture leaves TestOneShotRejectsAHatchWithNothingAfterIt
            green while TestAQuestionIsNotCaptured panics at once. So the harm BR-4
            actually named, a ReviewEvent with an empty Word that complete() discards
            at read time, has no falsifiable test. Measured prevalence - 1 of 1
            d.newStore injection in the package is dead.
          family: test-asserts-nothing
          round: 2
        - id: BR-11
          severity: Minor
          title: neither BR-5's nor BR-6's fix is pinned - both can be reverted with the suite green
          detail: |-
            This is the 3rd finding in family raw-mode-message-placement. Do not
            re-fix the sites. Rule - every message class the raw loop writes gets an
            assertion on the emitted byte stream, as TestEditorLoopUsesCarriageReturnsInRawMode
            (editorloop_test.go:143) already does for the definition path. Verified -
            removing eraseLine from replraw.go:223 and re-adding askInSession's own
            "\r\n" together leave go test ./cmd/define/ fully green. Measured
            prevalence - 3 of 3 raw-mode message placements introduced by this issue
            (the bare-"?" note, the forced ask, the unforced ask) are unasserted,
            which is why the family recurs.
          family: raw-mode-message-placement
          round: 2
        - id: BR-12
          severity: Minor
          title: the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
          detail: |-
            Two instances of one rule - a behaviour specified for the unforced route
            is silently inherited by the forced one. (a) submitLine adds cmd.word
            (replraw.go:299), which is post-strip, so typing `\how so` records
            "how so"; Up-arrow then Enter asks the line you had just forced to a
            lookup. The cmdAsk branch records submitted.String() (replraw.go:207), so
            "?why" stays forced. Measured through runEditor with an injected
            memHistory. (b) askUnavailable (ask.go:18) tells a forced question about a
            real headword that it "is not a word" - "?why" prints
            "no model configured; `why` is not a word". The sentence is true by
            construction only on the unforced route.
          family: forced-route-enumeration
          round: 2
        - id: BR-13
          severity: Minor
          title: the plan's step checkboxes are 0 of 60 ticked while M1's tasks are complete and committed
          detail: |-
            workshop/plans/000016-console-qa-plan.md - Tasks 1-6 shipped in five
            commits, but no step box is ticked. The two most recent archived plans set
            the convention (000015 39/39, 000014 33/33; 000011 did not), so this is
            soft, but leaving M1's boxes open makes "where does M2 resume" ambiguous
            when the plan is picked up in a fresh session.
          family: plan-bookkeeping
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-23T16:29:26-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: mayAsk moved into ask(); deleting that guard reddens exactly the three forced cells. Assertion strength raised separately.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Injection moved to d.capture and probe-verified live; making withStore overwrite it reddens TestTheCapturerInjectionIsLive.
          round: 3
        - id: BR-11
          disposition: addressed
          note: Both fixes now redden on a faithful revert — eraseLine removal and the BR-6 CRLF move each go red.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: Behaviour is correct but neither half is pinned - submitLine's recallLine and ask's whole q.forced branch both revert with the suite green.
          round: 3
        - id: BR-13
          disposition: addressed
          note: Chunk 1 is 29/29 ticked and Chunk 2 is 0/31, so the M2 resume point is unambiguous.
          round: 3
      findings:
        - id: BR-14
          severity: Important
          title: assertDidNotAsk passes for the failure it exists to catch, in 2 of TestRawNeverAsks's 6 cells
          detail: |-
            This is the 3rd finding in family test-asserts-nothing, so the deliverable
            is the rule. Rule - an assertion that pins "X did not happen" must assert
            the positive observable that distinguishes X from every other outcome, not
            the absence of one string. assertDidNotAsk (askroute_test.go:269) checks
            only that stderr lacks "no model configured". Measured - removing
            mayAsk(opt) from the miss branch at main.go:425 (a plausible cleanup; the
            comment there flags the redundancy with ask's own guard) reddens
            one-shot/unforced via its EXIT CODE only, while piped/unforced and
            editor/unforced stay green printing
            `define: -raw does not ask; drop -raw, or drop the "?"` for a line
            containing no "?" at all. atlas/define.md:538 and the test's own comment
            both claim the enumeration "covers every cell"; 4 of 6 can fail, 2 cannot.
            The distinguishing observable is cheap - assert stderr CONTAINS
            `no dictionary entry`, the miss the scripting contract promises.
          family: test-asserts-nothing
          round: 3
        - id: BR-15
          severity: Important
          title: the piped loop discards the usage code ask() computes, so -raw plus "?" exits 1 where README states 2
          detail: |-
            This is the 3rd finding in family doc-overstates-code, so state the rule -
            a non-zero code a dispatch computes inside replLines must survive the loop,
            and every exit-code absolute in README must be measured across
            {one-shot, piped, editor} before it is written. askHere (repl.go:189-193)
            collapses ask's return into anyFailed, discarding the 2; the cmdCommand
            branch 45 lines below feeds cmdCode correctly (repl.go:236-240) and the
            comment at repl.go:180 names this defect by number - BR-16, "Collapsing it
            into anyFailed made echo /histry | define exit 1 where dispatchCommand
            computes 2". Measured against the built binary -
            `define -raw '?why'` exits 2, `echo '?why' | define -raw` exits 1,
            `echo '/histry' | define` exits 2. README.md:81 states the claim
            unqualified ("on either route ... a usage error (exit 2)") and
            TestRawNeverAsks passes wantCode 0 for both piped cells, so nothing
            asserts it. Sweep the other three absolutes too - bare "?" and bare "\"
            exit 2 one-shot but 0 piped.
          family: doc-overstates-code
          round: 3
        - id: BR-16
          severity: Minor
          title: the two ask rows of TestRawLoopMessagePlacement assert nothing about the ask message
          detail: |-
            This is the 4th finding in family raw-mode-message-placement. Do not
            re-fix the site. BR-11's rule was executed for 1 of the 3 message classes
            the test enumerates - the ask rows assert only
            assertNoBareNewline(stdout), but ask writes to STDERR, so they never touch
            the message. Verified - removing cooked(...) from askInSession
            (replraw.go:134), which in production is the only reason the ask message's
            "\n" translates at all, leaves go test ./cmd/define/ fully green. The
            rig's cooked is a no-op closure (editorloop_test.go:34), so a recording
            cooked is the missing observable.
          family: raw-mode-message-placement
          round: 3
        - id: BR-17
          severity: Minor
          title: the plan still describes an M1 the code no longer implements, despite round 2 recommending the Revisions entry
          detail: |-
            workshop/plans/000016-console-qa-plan.md - last Revisions entry is
            "plan-quality round 2"; nothing records the two boundary rounds. Stale in
            four places - Task 3's snippet gives the miss condition as
            `!literal && readsAsQuestion(word)` where the code is
            `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; "-raw" and
            "mayAsk" appear nowhere in the plan; Task 4 names askUnavailable where the
            code has ask(opt, errOut, question); and Core concepts lists none of
            lookupOutcome, recallLine, question or mayAsk, while placing ask.go in M2.
            Rule - when a boundary round changes what the code does relative to the
            plan, the plan gets the entry in the same commit as the code.
          family: plan-bookkeeping
          round: 3
        - id: BR-18
          severity: Minor
          title: a bare "\" with a current word replays audio while a bare "?" gets its note
          detail: |-
            repl.go:53 - the "\" arm strips the prefix and falls into the
            empty-word test, so `\` alone with hasCurrent returns cmdReplay.
            TestParseREPLLine's "a bare backslash is blank" row only covers
            hasCurrent=false, so the asymmetry between the two hatches at the prompt
            is unasserted. Harmless today; worth one row.
          family: forced-route-enumeration
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-23T16:50:25-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: Third round unchanged - hist.Add(cmd.recallLine())->hist.Add(line) and deleting ask's whole q.forced branch both revert with the suite green.
          round: 4
        - id: BR-14
          disposition: addressed
          note: assertDidNotAsk asserts the positive observable; dropping the miss-branch mayAsk now reddens 3 of 3 unforced cells, not 1.
          round: 4
        - id: BR-15
          disposition: addressed
          note: fail(code) sink reddens piped/forced on revert; measured 8 of 8 exit-code cells on the built binary against README. Coverage of the swept cells raised in N-1.
          round: 4
        - id: BR-16
          disposition: addressed
          note: Recording cooked is a live observable - removing cooked(...) from askInSession reddens both ask rows.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: Plan still has three Revisions entries, none a boundary round; askUnavailable still named, -raw/mayAsk absent, Core concepts still places ask.go in M2.
          round: 4
        - id: BR-18
          disposition: addressed
          note: Reverting the bare-backslash arm reddens both hasCurrent rows of TestParseREPLLine.
          round: 4
      findings:
        - id: BR-19
          severity: Important
          title: 'no test asserts what any of #16''s messages say, and a doubled backslash shipped as a result'
          detail: |-
            This is the 4th finding in family test-asserts-nothing, so the deliverable
            is the rule. Rule - an assertion on a message must compare the bytes the
            user receives against a literal expectation written in the test;
            referencing the production constant asserts only that a branch was
            selected, not that its text is right. Shipped defect - noteEmptyLiteral
            (repl.go:44) is a Go RAW string literal containing `\\`, so both
            `define '\'` and `echo '\' | define` print
            `define: type a word after "\\"` with a doubled backslash, while its
            sibling bare-"?" note is correct. The only assertions are repl_test.go:43
            and :45, which compare note to noteEmptyLiteral - the constant to itself.
            Measured prevalence, 4 of 5 message/code behaviours this milestone
            introduced are unasserted at the point of delivery, all GREEN on revert -
            deleting nothingSays's `if c.note != ""` early return; deleting
            `if cmd.note != "" { fail(2) }` at repl.go:261 (BR-15's own sweep, correct
            in code but unpinned, so README's "a bare ? or \ with nothing after it"
            exit 2 has no piped assertion); and replacing truncateQuestion(q.text)
            with q.text in ask's two messages. Only the placement fix (eraseLine) goes
            red, which is why the family recurs - each round pinned a placement and
            never a text. One table over {?, \} x {one-shot, piped} asserting exit code
            AND literal stderr text closes all four rows.
          family: test-asserts-nothing
          round: 4
        - id: BR-20
          severity: Minor
          title: nothingSays is documented as "the ONE place" while replayInPlace holds a live duplicate of its replay sentence
          detail: |-
            This is the 4th finding in family doc-overstates-code. Do not re-fix the
            site. Rule - a consolidation claimed as "the ONE place" must be verified by
            enumerating the copies it consolidated; a uniqueness claim is an absolute
            and it quantifies over the whole tree. nothingSays's doc comment
            (repl.go:100-104) and atlas/define.md:553 both assert it is the one place
            that answers "this line meant nothing - why", but replayInPlace
            (replraw.go, `case current == ""`) still holds a byte-identical copy of
            that sentence, and it is the LIVE path for the raw editor's bare Enter
            with nothing current - pinned by TestEditorLoopBareEnterWithNoCurrentWord
            (editorloop_test.go:235). Same rule, second half - nothingSays(cmd, true)
            at repl.go:257 hardcodes true where sess.hasCurrent() is in scope, and a
            note-less cmdNothing is reachable ONLY when hasCurrent is false (a blank
            line with a current word parses to cmdReplay), so that site selects "press
            return to replay the last one" in the one state where there is nothing to
            replay. Measured on the built binary. The text is pre-#16 and faithfully
            ported; the parameter introduced to distinguish the two cases is what is
            fed a literal.
          family: doc-overstates-code
          round: 4
        - id: BR-21
          severity: Minor
          title: four byte-identical "lost the terminal" blocks in runEditor, two added by this diff
          detail: |-
            replraw.go:195, :211, :241, :250 - the same
            finish(); Fprintf("define: lost the terminal: %v"); return 1 three-liner,
            up from 2 copies before this window. M2's Task 11 adds streaming inside
            askInSession and is positioned to add a fifth. One helper, on the same
            argument nothingSays was extracted on (ARCH-DRY).
          family: stdlib-reuse
          round: 4
      boundary: M1
      blocked: false
    - "n": 5
      timestamp: "2026-08-23T23:03:30-07:00"
      agent: claude
      findings:
        - id: BR-22
          severity: Critical
          title: '"Recently in the deck" sends the twelve OLDEST deck words, unreversed'
          detail: |-
            cmd/define/ask.go:180-187. Store.Deck() is newest-first (store.go:14,
            sortDeck at mem.go:62-69) and lastN takes the tail, so a deck larger than
            maxContextWords sends the oldest twelve and drops the most recent. Measured
            with a 15-word deck: the section renders "wl, wk, ... wa" and the three
            newest words never reach the model. The code's own comment states both
            intentions it fails ("reads better oldest-first", "the newest are the ones
            worth keeping"), and README:82 plus atlas/define.md:522 both promise "the
            recent deck". The selection is a PURE decision inlined in the IO gatherer,
            which is why no test above the bound exists (ARCH-PURE) — fix by extracting
            recentDeck() beside recentTurns and table-testing it.
          family: policy-in-io-shell
          round: 5
        - id: BR-23
          severity: Critical
          title: One-shot unforced ask puts the question text into session.current and the event log's word field
          detail: |-
            This is the 3rd finding in family forced-route-enumeration; earlier rounds
            fixed instances, so fix the CLASS. cmd/define/main.go:387 passes
            &session{current: oneShot.word}, where oneShot.word is the line the
            dictionary just missed — the question. Measured with a live seam and a real
            YAML store: the prompt renders "## The word on screen\n<the question>" with
            no entry, and recordAsked (ask.go:150) writes ReviewEvent{Word: "<the
            question>"} into the append-only log #17 folds over. That violates plan D2
            and Task 4's contract table ("does not set sess.current"). The forced
            one-shot cell passes &session{} and is correct, so the two routes diverge.
            The rule: every fact the ask path carries is quantified over {forced,
            unforced} x {one-shot, piped, editor} and each cell asserted for THAT fact.
            M2 added three such facts (which session, where the answer is written,
            whether the interrupt is scoped); measured coverage is 2 of 18 cells.
          family: forced-route-enumeration
          round: 5
        - id: BR-24
          severity: Important
          title: gatherAskContext discards UserModel's error, defeating the reason it returns one
          detail: |-
            cmd/define/ask.go:172-174 writes `if model, err := d.deck.UserModel(); err
            == nil`. store/yaml.go:44-48 justifies returning an error precisely to
            prevent this: "silently answering \"\" for a model that exists would make
            every answer pitched at the wrong level with no way to tell." Warn on
            stderr (the storeCapturer/storeHistory precedent) or delete the
            justification.
          family: silent-error-swallow
          round: 5
        - id: BR-25
          severity: Important
          title: go test -race ./cmd/define/ now fails; two new tests read a bytes.Buffer concurrently
          detail: |-
            Verified: base 993fccc is race-clean, HEAD 686913c reports eleven races and
            two failures. askrun_test.go:147-151 spins on out.Len() while runAsk writes
            out; askrun_test.go:267,280 read out.String() via waitFor while runEditor
            writes it. The package already owns the correct shape — ptyOut's
            mutex-guarded buffer at pty_conformance_test.go:88-124 — so this is also
            ARCH-DRY. Extract one syncBuf and use it in all three places.
          family: unsynchronised-test-observation
          round: 5
        - id: BR-26
          severity: Important
          title: The plan's askCapturer integration point was not built; events append through a second write path
          detail: |-
            Core concepts -> Integration points lists askCapturer on Capturer in
            capture.go wrapping the event append. The code appends directly via
            d.deck.AppendEvent from ask.go:145-152. Behaviour is safe (DEFINE_NO_CAPTURE
            leaves sd.deck nil so recordAsked no-ops), but it adds a second writer to
            the event log beside capture and makes main.go:31-34 stale ("capture is the
            only thing that RECORDS lookups. deck ... --forget deletes through it").
            Either route it through the seam or add a "## Revisions" entry and fix the
            comment.
          family: plan-contract-drift
          round: 5
        - id: BR-27
          severity: Important
          title: README does not say that question text is persisted to events/
          detail: |-
            This is the 2nd finding in family readme-surface-gate, so state the rule
            rather than adding one line. README:95-105 is the block documenting what
            define writes to the working directory; M2 makes every question's text
            persist into events/*.yaml and that block does not say so. The rule: any
            new record type or field written into the working directory is named in
            that block in the same change. Enumeration: {words/, events/,
            user-model.md} x {looked-up, reviewed, asked}; README currently names 2 of
            3 files and 0 of 3 kinds.
          family: readme-surface-gate
          round: 5
        - id: BR-28
          severity: Important
          title: atlas/define.md:182-186 still states the pre-M2 cancellation model the diff disproved
          detail: |-
            This is the 5th finding in family doc-overstates-code; do NOT fix only this
            paragraph. It claims Ctrl-C is "not a signal", that the piped path "still
            relies on" signal.NotifyContext, and that the key reader calls cancel().
            All three are false after this diff: the pty suite measured \x03 arriving
            as SIGINT (D5's premise), repl.go:180 detaches with WithoutCancel, and
            readKeys calls interrupts.Fire(). The diff corrected the code comment at
            rawterm.go:44-46 and added a NEW atlas section at :544 while leaving the
            old statement standing. The rule: when a diff corrects a claim in a code
            comment, every restatement of that fact elsewhere is a consumer of the same
            source and must be swept in the same change (ARCH-PURPOSE shadow-sweep) —
            grep the fact, do not append beside the stale copy. Measured: 1 of 2
            NotifyContext mentions in atlas/ is now false.
          family: doc-overstates-code
          round: 5
        - id: BR-29
          severity: Important
          title: The piped loop's ask wiring is unpinned — answer destination and session both deletable green
          detail: |-
            This is the 2nd finding in family loop-shell-branch-untested; state the
            rule. Two mutations at repl.go:261 each leave go test ./cmd/define/ fully
            green: replacing stdout with io.Discard (the answer goes nowhere), and
            replacing &sess with &session{} (the piped loop loses multi-turn and
            session words). lessons.md define #15 already states the rule; M2 obeyed it
            for runEditor and skipped it for replLines and the one-shot. That skipped
            cell is where the one-shot Critical lives — the same six-cell enumeration
            fixes both.
          family: loop-shell-branch-untested
          round: 5
        - id: BR-30
          severity: Important
          title: Four claims in this diff survive the mutation they exist to catch
          detail: |-
            This is the 5th finding in family test-asserts-nothing; do NOT fix the four
            instances, state the rule. Measured individually, all green: (1) the
            key-channel buffering rawterm.go:49-54 calls "load-bearing" — make(chan
            Key) instead of make(chan Key, 256); (2) session words reaching the prompt,
            an issue Done-when row — SessionWords: nil in gatherAskContext; (3)
            recordExchange's empty-answer guard, session.go:44-46 — delete it; (4)
            askrun_test.go:293-296 claims to pin "the interrupt scoping and the CRLF
            writer" for both routes but passes nil for interrupts, so it can only pin
            the writer. Row (2) is the BR-2 shape exactly: the test asserts
            "sycophantic" is in the prompt, but CurrentWord supplies that string, so
            the SessionWords assertion is satisfied by a different source. The rule: an
            assertion that a value reached an output must use a value only that source
            can supply. Also applies to
            TestEditorCtrlCMidStreamReturnsToThePrompt/the_signal_transport, which
            calls interrupter.Fire() directly and never touches d.notifySignals.
          family: test-asserts-nothing
          round: 5
        - id: BR-31
          severity: Important
          title: assertNoBareNewline is vacuous for all three rows of TestRawLoopMessagePlacement
          detail: |-
            This is the 5th finding in family raw-mode-message-placement; state the
            rule. The helper does strings.TrimSuffix(s, "\n") at askroute_test.go:395
            to excuse the loop's post-finish() newline, and all three rows write
            exactly one line to stderr — so the excused newline is the only one under
            test. Measured: unwrapping stderr from crlfWriter at replraw.go:141 leaves
            the suite green, and changing the bare-? note's "\r\n" to "\n" at
            replraw.go:226 leaves TestRawLoopMessagePlacement green. The test's own
            header claims "EVERY message this loop writes ... carries its own carriage
            returns" — vacuous for 3 of 3 rows. The rule (BR-14's rule applied to
            placement): a placement assertion asserts the POSITIVE observable, that the
            message's own terminator IS "\r\n", never the absence of a bare "\n".
          family: raw-mode-message-placement
          round: 5
        - id: BR-32
          severity: Important
          title: 31 unticked M2 plan steps and no Revisions entry for any boundary round or M2 design departure
          detail: |-
            This is the 3rd finding in family plan-bookkeeping; state the rule.
            workshop/plans/000016-console-qa-plan.md Chunk 2 has 31 unticked steps and
            0 ticked, while the issue's ## Plan and the project row both mark M2 [x].
            The plan still carries three ## Revisions entries, none of which is a
            boundary round (BR-17, raised in M1, never disposed), and none covering
            M2's three design departures. The rule: the plan artifact's state is part
            of the milestone deliverable — the checkbox sweep plus a ## Revisions entry
            for every fork taken differently from the plan happen in the
            milestone-close commit, not at issue close.
          family: plan-bookkeeping
          round: 5
        - id: BR-33
          severity: Minor
          title: runAsk's `parent := ctx` is a bare alias with no derived context
          detail: |-
            ask.go:100-102 copies llmcheck.go:45-47's idiom, but llmcheck derives a
            WithTimeout child so parent and ctx genuinely differ. Here they are the
            same value and the comment describes a distinction that does not exist. The
            observed cancel path also returns ErrTruncated, not ErrUnavailable, so the
            failure mode the comment names is the pre-text case only.
          family: doc-overstates-code
          round: 5
        - id: BR-34
          severity: Minor
          title: Core concepts table cites the wrong file for `exchange` and `interrupter`
          detail: |-
            `exchange` is in askctx.go:9-12, not session.go; `interrupter` is in the
            new interrupt.go, not rawterm.go. Both entities exist, are pure and are
            tested, so this is a stale table rather than a missing deliverable —
            correct the rows in the plan revision.
          family: plan-contract-drift
          round: 5
        - id: BR-35
          severity: Minor
          title: bytesReader is strings.NewReader under a new name; keysFor is dead
          detail: |-
            This is the 3rd finding in family stdlib-reuse. interrupt_test.go:98
            defines `func bytesReader(s string) io.Reader { return
            strings.NewReader(s) }` for one call site. The rule (same as
            contains->slices.Contains two rounds ago): before adding a helper, check
            whether the stdlib already names it. Separately, askrun_test.go:334-347's
            keysFor is defined and never called, with a doc comment describing a use
            case that does not exist.
          family: stdlib-reuse
          round: 5
        - id: BR-36
          severity: Minor
          title: DEFINE_NO_CAPTURE now also suppresses READING user-model.md and the deck
          detail: |-
            openStore leaves deck nil under noCapture, so gatherAskContext (ask.go:167)
            skips both the learner model and the deck. README documents the variable as
            "write nothing in this directory"; answers silently become un-adapted. Also
            note unavailable() reports "no model configured" for a configured but
            unreachable model — plan-specified, but the same misdiagnosis surface as
            the already-filed tools#19.
          family: doc-overstates-code
          round: 5
        - id: BR-37
          severity: Minor
          title: askInSession always returns nil, so lostTerminal is unreachable at both ask call sites
          detail: |-
            replraw.go:130-155 no longer calls cooked(), so the closure cannot fail;
            the `if err := askInSession(...); err != nil { return lostTerminal(err) }`
            guards at both call sites are dead. Also crlf.go:28 returns 0 on a write
            error rather than the bytes consumed, and reports len(p) on a short
            underlying write; and ask.go:114 prints a newline on cancel even when no
            delta ever arrived.
          family: dead-branch
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-08-23T23:40:16-07:00"
      agent: claude
      dispose:
        - id: BR-22
          disposition: addressed
          note: newestFirst takes the head and reverses; reverting to lastN reddens TestTheDeckSectionCarriesTheNewestWords (verified).
          round: 6
        - id: BR-23
          disposition: addressed
          note: 'main.go:393 passes &session{}; restoring current: oneShot.word reddens TestOneShotQuestionHasNoCurrentWord (verified).'
          round: 6
        - id: BR-24
          disposition: not-addressed
          note: The warn is correct and reachable, but no test fails without it — and failingStore.UserModel, added by this same diff, is at zero call sites.
          round: 6
        - id: BR-25
          disposition: addressed
          note: go test -race ./cmd/define/... is ok at HEAD; syncBuf is used in all three places and ptyOut folds onto it.
          round: 6
        - id: BR-26
          disposition: addressed
          note: capture.go is the only non-test AppendEvent caller; a no-op CaptureAsk reddens three tests (verified). Plan Revisions records the departure.
          round: 6
        - id: BR-27
          disposition: addressed
          note: README:99-103 names events kinds, user-model.md and "answers are NOT stored"; pinned by TestTheEventLogHoldsQuestionsAndNotAnswers.
          round: 6
        - id: BR-28
          disposition: addressed
          note: atlas/define.md:182-191 rewritten; the surviving NotifyContext mention at :676 is about the one-shot path and is still true. The family recurs elsewhere — raised separately.
          round: 6
        - id: BR-29
          disposition: addressed
          note: Both repl.go:261 mutations now redden TestThePipedLoopsAskWiring (verified). The sibling cells the finding did not name are raised separately.
          round: 6
        - id: BR-30
          disposition: not-addressed
          note: Four of the five named items are fixed and mutation-verified red; the fifth is untouched — the "signal transport" subtest still calls interrupts.Fire() directly and never reaches d.notifySignals, and no Revisions entry records it.
          round: 6
        - id: BR-31
          disposition: addressed
          note: assertCRLFTerminated counts terminators; both the unwrapped-stderr and the bare-newline mutation now redden TestRawLoopMessagePlacement (verified).
          round: 6
        - id: BR-32
          disposition: addressed
          note: 60 of 60 plan checkboxes ticked, and a "M2's design departures, and the M1 rounds" Revisions entry now exists.
          round: 6
        - id: BR-33
          disposition: addressed
          note: The parent alias is gone; runAsk asks ctx.Err() directly.
          round: 6
        - id: BR-34
          disposition: addressed
          note: Core concepts now cites askctx.go for exchange and interrupt.go for interrupter.
          round: 6
        - id: BR-35
          disposition: addressed
          note: Neither bytesReader nor keysFor appears anywhere in cmd/define.
          round: 6
        - id: BR-36
          disposition: addressed
          note: README:118-120 documents that DEFINE_NO_CAPTURE also suppresses reading the deck and user-model.md.
          round: 6
        - id: BR-37
          disposition: addressed
          note: askInSession returns nothing and lostTerminal still has two live call sites; crlf reports caller-unit progress; the cancel newline is guarded by answer.Len(). The new consumed() helper has its own defect, raised separately.
          round: 6
      findings:
        - id: BR-38
          severity: Important
          title: Three cells of the ask-wiring table still survive their mutation with the full suite green
          detail: |-
            This is the 3rd finding in family loop-shell-branch-untested, and the 2nd
            round in which the enumeration BR-23 wrote out was answered with targeted
            tests instead of the table. Measured against the whole ./cmd/define/ suite,
            not a -run subset: replraw.go:161 &sess -> &session{} leaves it green (ok,
            28.160s) — the raw editor is the primary UI and multi-turn is a Done-when
            row; main.go:366 and main.go:393 stdout -> io.Discard, applied together,
            leave it green (ok, 28.391s) — `define "what is the difference to
            obsequious?"` printing nothing is caught by nothing. Coverage of the
            {one-shot, piped, editor} x {which session, answer destination, interrupt
            scope} table is 4 of 7 applicable cells. Do NOT add three more one-off
            tests: write the table as one fixture-driven test whose rows ARE the cells,
            the way TestRawNeverAsks already does for its six.
          family: loop-shell-branch-untested
          round: 6
        - id: BR-39
          severity: Important
          title: Four measured doc claims contradict the code, three of them created by this window
          detail: |-
            This is the 8th finding in this family; the rule has been stated three
            times (BR-9's absolute-names-its-enumeration, BR-28's shadow-sweep) and
            keeps recurring, so the escalation must be mechanical. Measured:
            (1) replraw.go:143 "It runs COOKED for the same reason a command does",
            five lines above the code and comment saying it streams RAW through
            crlfWriter; (2) atlas/define.md:574 restates ask's pre-M2 signature `ask(opt
            options, errOut io.Writer, q question)` — 1 of the 3 quoted Go signatures in
            that file is stale and it is the one this diff changed; (3) README:78
            "Ctrl-C stops the answer rather than the session" — measured through the
            injected signal transport with stdin a tty and stdout redirected, the LINE
            loop ends the session, because replLines' askHere (repl.go:261) passes the
            loop's own ctx that the default sink cancels; true in 1 of 2 interactive
            loops; (4) README:97 "Every question you ask is recorded too, by its text" —
            measured 0 events for an errored (400) ask, 0 for an unwired seam, and 0
            events plus 0 turns for a cancelled ask; true in 1 of 4 outcome cells. The
            mechanical fix: atlas and README stop restating signatures and unqualified
            absolutes, and each surviving absolute gets a named-enumeration row test
            like TestRawNeverAsks.
          family: doc-overstates-code
          round: 6
        - id: BR-40
          severity: Important
          title: The event log's first free-form user field has no test defending the record-boundary invariant
          detail: |-
            Until this window every value written to events/*.yaml was a single
            dictionary headword; `question:` is now arbitrary text the user typed. The
            reader's record boundary is a literal top-level "- " (yaml.go:307) and the
            only thing keeping user text off column 0 is that yaml.Marshal indents
            block scalars. I verified today's behaviour is correct — five adversarial
            questions round-trip intact, including one whose text is a complete forged
            "- word: injected / kind: looked-up / at: ..." record — but
            storetest/suite.go:52 round-trips one plain question and nothing pins the
            invariant. A regression corrupts an append-only log #17 folds over,
            irreversibly; BR-4 is the precedent for an event that looked whole being
            silently discarded at read time. Add a suite row with a newline, a leading
            "- " and an embedded "at:".
          family: user-text-in-record-format
          round: 6
        - id: BR-41
          severity: Minor
          title: crlf.go's consumed() ignores the `out` it is handed and re-derives the translation from a wrong seed
          detail: |-
            consumed(p, out, n) at crlf.go:41 never reads `out` — a dead parameter —
            and restarts the translation with `lastWasCR := false` instead of the
            writer's carried entry state. Write "a\r" then "\nb" with a 1-byte short
            write and it reports 0 consumed where 1 byte was written, so a retry
            duplicates the newline: the exact defect the fix's own comment says it
            prevents, surviving in the one case lastWasCR exists for.
            TestCRLFWriterReportsProgressOnAShortWrite covers only the fresh-state
            cell. Derive from `out`, or capture the entry flag before the loop.
          family: second-implementation-drifts
          round: 6
        - id: BR-42
          severity: Minor
          title: newestFirst returns oldest-first, sits in the IO shell, and is only reachable through a store-backed test
          detail: |-
            This is the 2nd finding in family policy-in-io-shell; BR-22's defect itself
            is genuinely gone. What remains is the shape BR-22 named: a pure ordering
            policy living in ask.go rather than beside recentTurns in askctx.go, whose
            only test (TestTheDeckSectionCarriesTheNewestWords) needs a real YAML store
            and a wire-level fake to exercise a slice reversal. The name also says
            newest-first while the function returns oldest-first.
          family: policy-in-io-shell
          round: 6
        - id: BR-43
          severity: Minor
          title: The pty conformance suite runs whatever bin/define is on disk, with no staleness check
          detail: |-
            startDefine (pty_conformance_test.go:63) skips when ../../bin/define is
            absent but never checks it is current. The binary here is timestamped
            22:45, 34 minutes before the round-5 fix commit at 23:19, so a `go test
            -tags conformance` run right now would validate pre-fix code and report ok
            — a live conformance check that cannot fail on the change it exists to
            check. Compare against the newest source mtime, or build in TestMain.
          family: test-asserts-nothing
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-24T14:08:16-07:00"
      agent: claude
      dispose:
        - id: BR-24
          disposition: addressed
          note: Reverting the warn to a discarded error reddens TestAnUnreadableUserModelIsReported; failingStore.UserModel now has a live call site.
          round: 7
        - id: BR-30
          disposition: addressed
          note: Deleting repl's signal watcher now reddens TestBothInterruptTransportsReachTheSink and TestThePipedLoopStillExitsOnASignal (verified); the subtest still bridges sigs itself, recorded as a plan revision rather than re-raised.
          round: 7
        - id: BR-38
          disposition: addressed
          note: TestTheAskWiringTable is the table, not a fourth one-off; all three named mutations plus the piped session cell redden against the full suite (verified).
          round: 7
        - id: BR-39
          disposition: addressed
          note: All four measured claims fixed and the two behavioural gaps pinned by enumeration tests; the surviving stale restatements are elsewhere in atlas and raised as the 9th in the family.
          round: 7
        - id: BR-40
          disposition: addressed
          note: The hostile-question suite row reddens when AppendEvent is replaced with a naive hand-written record (verified), so it is not vacuous.
          round: 7
        - id: BR-41
          disposition: addressed
          note: The out parameter is gone and entryWasCR is captured before the loop; re-seeding from false reddens TestCRLFWriterProgressAcrossACarriedCR (verified).
          round: 7
        - id: BR-42
          disposition: addressed
          note: recentDeck is pure, lives beside recentTurns in askctx.go and is table-tested without a store; un-reversing it reddens three rows (verified).
          round: 7
        - id: BR-43
          disposition: addressed
          note: builtBinary compiles from the tree per run; bin/define survives only in a comment. Staleness is impossible by construction.
          round: 7
      findings:
        - id: BR-44
          severity: Important
          title: Three claims in atlas's store section were falsified by this window, in the paragraphs that own them
          detail: |-
            This is the 9th finding in this family; the rule has now been stated four
            times, so do NOT patch the three lines. Measured: (1) atlas/define.md:210-213,
            the normative artifact block, lists words/ and events/ while the code
            (store/yaml.go:37) and README both say user-model.md is the third — the atlas
            names it only in M2 prose at :527, so 2 of 3 restatements were swept;
            (2) atlas/define.md:252 "a whole record ... carries every field", contradicted
            by complete()'s generalisation to "has a SUBJECT" (store/event.go:45) that
            this window shipped precisely so a wordless asked record is whole — a reader
            following the atlas discards exactly the events it was generalised to keep;
            (3) atlas/define.md:576 "the fake behaves like the real thing is a test rather
            than an assumption", false for the newest interface method (see the
            test-asserts-nothing finding). The mechanical fix: a normative block in atlas
            is a CONSUMER of the code, so a change to what the code enumerates sweeps
            every block that enumerates it in the same commit. The enumeration is three
            blocks in the store section, resolved against yaml.go, event.go and
            storetest/suite.go mechanically, the way PQ-5 resolved every line citation.
          family: doc-overstates-code
          round: 7
        - id: BR-45
          severity: Important
          title: Two pieces of scaffolding added this round read as protection and cannot fail
          detail: |-
            This is the 7th finding in this family, and both instances are the shape
            BR-24's own disposal note named — "the disposal is the test". Measured, 2 of
            the 14 test helpers and fake fields this window added: (1) capture_test.go:85
            countingCapturer.asked and askedWord are appended at one site and read at
            ZERO, while their comment claims "so a test can assert that a question was
            NOT counted as one"; no test reads either field (the distinction is asserted
            at the store level instead, so this is dead scaffolding). (2)
            storetest/suite.go:38 "UserModel is empty before anything writes one" runs
            against both implementations, but store/mem.go:20 userModel has no setter and
            no writer anywhere in the tree, so for Mem the row asserts the only value the
            type can produce — unfalsifiable, and ARCH-MOCK: the fake cannot hold the
            state the real one holds for the method Store gained in this window. The
            rule: a fixture, field or suite row added to defend a finding must have a
            read site that can fail in the same commit, and a fake added to a conformance
            suite must be able to hold the state the row asserts about.
          family: test-asserts-nothing
          round: 7
        - id: BR-46
          severity: Important
          title: The scoped-ask wiring is now written twice, and only one copy carries the ordering rationale
          detail: |-
            This is the 2nd finding in family second-implementation-drifts, so state the
            rule rather than fixing one site. repl.go:273-279 and replraw.go:148-168 each
            contain the identical five-step sequence — derive qctx, interrupts.Set,
            call ask, restore, qcancel — and the ordering is load-bearing in a way that
            is silent when wrong: only replraw.go:163-166 records why restore precedes
            qcancel. The second copy was created by round 2's BR-39 fix, while
            replraw.go:140-143 and atlas/define.md:568 both still say a second copy of
            this wiring would be a second copy of Ctrl-C's meaning (ARCH-DRY). The rule,
            same shape as fail(code) and nothingSays: one function owns the scope and
            both loops call it — askScoped(ctx, interrupts, run) — with the writers and
            the post-answer redraw staying in each loop's closure, since those are what
            legitimately differ.
          family: second-implementation-drifts
          round: 7
        - id: BR-47
          severity: Important
          title: The Core concepts tables do not describe the entities this boundary's rounds created
          detail: |-
            This is the 3rd finding in this family; BR-34 fixed two rows by eye and the
            next round created three more discrepancies, so fix the enumeration rather
            than the rows. Measured against workshop/plans/000016-console-qa-plan.md:
            (1) recentDeck (askctx.go:86) is a NEW pure entity with its own table test,
            extracted by round 1 in answer to a Critical, and is absent from the
            Pure-entities table entirely; (2) Integration points names "askCapturer (on
            Capturer)" where the code's method is CaptureAsk (capture.go:49) — the
            Revisions entry says so, the table does not; (3) the session bullet lists
            current, entry, turns while the code also has words (session.go:20), the
            field an issue Done-when row quantifies over. 3 of 3 entity changes made by
            the two boundary rounds are unrecorded or wrong. The rule, mechanised: at a
            boundary close the enumeration is
            git diff base..HEAD -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ',
            resolved against the Core concepts tables — a row fixed because a finding
            named it is the instance again.
          family: plan-contract-drift
          round: 7
        - id: BR-48
          severity: Minor
          title: Boundary round 2 has no Revisions entry, including the one the previous round asked for
          detail: |-
            This is the 4th finding in family plan-bookkeeping. The round-1 entry states
            the rule ("a Revisions entry is owed for every fork taken differently from
            the plan, in the milestone-close commit") and round 2 then took four forks
            and recorded none: newestFirst became recentDeck and moved to askctx.go,
            TestThePipedLoopsAskWiring was folded into TestTheAskWiringTable, replLines
            gained the interrupt scope the plan gave only to the raw loop, and the pty
            suite now builds its own binary. Round 6's explicit recommendation — Task
            11's test decomposition, and that "SIGINT through repl's watcher DURING a
            scoped stream" is still unasserted — is also absent. Checkboxes are 60 of 60,
            so this is the entry half of the rule only.
          family: plan-bookkeeping
          round: 7
        - id: BR-49
          severity: Minor
          title: The interrupter seam has three consumers and two nil policies; one of them dereferences
          detail: |-
            runEditor (replraw.go:56) and replLines (repl.go:226) substitute
            &interrupter{} when handed nil, while replRaw (replraw.go:15) passes it
            straight to readKeys, which dereferences it on the first Ctrl-C. Not
            production-reachable — repl is replRaw's only caller and always supplies one
            — but 44 test call sites now pass nil, exercising a configuration production
            never has, and the two policies will diverge again when #17 consumes the
            seam. Either the seam is required (tests pass &interrupter{}) or optional
            (one policy applied at every entry that consumes it).
          family: nil-seam-policy
          round: 7
        - id: BR-50
          severity: Minor
          title: 'Four small residues: a stale prediction, an unreachable guard, a hanging test arm, and a per-question full deck read'
          detail: |-
            This is the 2nd finding in family dead-branch, so these are recorded rather
            than individually fixed. (1) replraw.go:132 "M2's streaming adds a fifth"
            predicts a lostTerminal call site that M2 deliberately did not create; two
            live sites remain — belongs to the atlas sweep above. (2) capture.go:97's
            decideCapture guard in CaptureAsk is deletable with the suite green and is
            unreachable in production, since -raw refuses in ask() before runAsk and
            noCapture yields a noopCapturer; correct as a mirror of Capture's guard, but
            not a tested policy. (3) repl_test.go:174 uses t.Context().Done() as its
            failure arm, so a loop that never returns HANGS to the package timeout
            instead of failing — measured at 600s during a mutation; its sibling
            assertSignalEndsTheLoop uses a 5s time.After. (4) gatherAskContext calls
            Deck() on every question, reading every file under words/ to pick twelve —
            negligible beside a network round-trip today, worth remembering when #10
            reuses askContext.
          family: dead-branch
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-08-24T14:30:32-07:00"
      agent: claude
      dispose:
        - id: BR-44
          disposition: addressed
          note: All three atlas claims swept at the paragraphs that own them; the conformance sentence is now true for the newest method because SetUserModel landed.
          round: 8
        - id: BR-45
          disposition: addressed
          note: Both verified by reversion — a no-op Mem.SetUserModel reddens the suite row, and mutating the captured word reddens countingCapturer.askedWord.
          round: 8
        - id: BR-46
          disposition: addressed
          note: One askScoped, called by both loops; removing the scope reddens 7 cells across both. The residual — 3 of its 4 mutations undefended — is raised separately.
          round: 8
        - id: BR-47
          disposition: not-addressed
          note: The three named rows are fixed, but the enumeration the finding demanded was run in the commit that stated it and missed Store.SetUserModel, which that same commit created.
          round: 8
        - id: BR-48
          disposition: not-addressed
          note: Still no Revisions entry for boundary round 2's four forks or round 6's Task 11 item; and Task 12's `sdlc close` checkbox is ticked while the issue is status:working at a milestone-close gate.
          round: 8
        - id: BR-49
          disposition: not-addressed
          note: Unchanged — replRaw (replraw.go:15) still passes the seam straight to readKeys with no policy, and 44 test call sites still pass nil.
          round: 8
        - id: BR-50
          disposition: not-addressed
          note: All four residues present verbatim — replraw.go:132's "adds a fifth", capture.go:96's guard, repl_test.go:199's hanging arm, ask.go:199's per-question Deck().
          round: 8
      findings:
        - id: BR-51
          severity: Important
          title: askScoped's sequence has four mutations and one is defended; omitting restore makes the session unquittable, suite green
          detail: |-
            This is the 8th finding in this family, so the rule rather than the site.
            cmd/define/ask.go:40-45. The commit that created askScoped probed ONE
            mutation (reversing the two defers), found it unobservable, and concluded
            "a rationale no test can defend is scaffolding". Measured, all four:
            omitting interrupts.Set reddens 5 tests across both loops; omitting
            `defer restore()` leaves the FULL suite green; omitting `defer qcancel()`
            leaves it green AND go vet silent, because qcancel is used as a value so
            lostcancel never fires; reordering is genuinely unobservable. The restore
            cell is user-visible: interrupter.scoped stays true and fn stays the dead
            question cancel, so readKeys (rawterm.go:70) swallows every subsequent
            \x03 and Ctrl-C at the prompt does nothing after the first question. I
            wrote the test — drive runEditor through the real readKeys, ask ?why, let
            the answer COMPLETE, send \x03, require the loop to return — and it fails
            mutated and passes unmutated in 0.67s. Every existing test asserts the
            sink DURING an answer; none asserts it AFTER one. The rule: when a probe
            finds a mutation untestable, the deliverable is the ENUMERATION of
            mutations to that mechanism, not the verdict on the one probed.
          family: test-asserts-nothing
          round: 8
        - id: BR-52
          severity: Important
          title: An interrupted answer is dropped from the transcript, so the follow-up README promises resolves against nothing
          detail: |-
            This is the 10th finding in this family; the rule has been stated five
            times, so do NOT weaken the sentence. cmd/define/ask.go:140-145 returns 0
            on the cancel path BEFORE sess.recordExchange, so an answer the user read
            and then stopped leaves no trace — not the partial answer, not even the
            question. README:79-88 states both halves in adjacent paragraphs:
            "Ctrl-C stops the answer rather than the session" and "the earlier
            questions in this session — so a follow-up like `give me two more
            examples` resolves against the answer before it". Measured against the
            wire fake: after 32 bytes streamed and cancelled, sess.turns is empty and
            the follow-up prompt contains only "## The word on screen / sycophantic"
            and the new question. Across the cells the claim quantifies over —
            answered, ErrTruncated, ErrUnavailable-with-partial, cancelled — it is
            true in 3 of 4, and the false cell is the flow the milestone is named
            after. The fix is to record the partial exchange (the user READ it, the
            same reason ask.go:153 keeps a truncated one) and give the claim a
            named-enumeration row test, the way TestRawNeverAsks does its six.
          family: doc-overstates-code
          round: 8
        - id: BR-53
          severity: Minor
          title: crlfWriter.Write advances lastWasCR over bytes the underlying writer never took
          detail: |-
            This is the 3rd finding in this family, so recorded rather than fixed at
            the site. crlf.go:19-42: entryWasCR is captured for consumed()'s benefit,
            but c.lastWasCR is still advanced across the WHOLE buffer even when only
            part of it was written, so the state carried into a retry describes bytes
            that never reached the terminal. Verified: Write("\r\nz") short at 1 byte
            returns n=1 correctly (BR-41's fix), and the retry with p[1:] = "\nz" then
            inserts a carriage return already on the wire — measured "\r\r\nz", the
            exact doubling lastWasCR exists to prevent and the one consumed()'s
            comment claims it prevents. BR-41 corrected the return value and left the
            carried state. Low reach: nothing in the tree retries and runAsk discards
            fmt.Fprint's error, so the symptom is one stray \r garbling a line on an
            already-degraded path. The rule: a translation materialised once must not
            be re-derived — Write and consumed are still two derivations of it.
          family: second-implementation-drifts
          round: 8
      boundary: M2
      blocked: false
    - "n": 9
      timestamp: "2026-08-24T20:42:05-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: Half (b) is now pinned; half (a)'s wiring still reverts green — submitLine's hist.Add(cmd.recallLine()) -> hist.Add(line) leaves the full suite green on two runs.
          round: 9
        - id: BR-17
          disposition: not-addressed
          note: Unchanged verbatim — Task 3's snippet at line 632, askUnavailable at 686/745, and zero mentions of mayAsk, recallLine, nothingSays or lostTerminal.
          round: 9
        - id: BR-19
          disposition: addressed
          note: All four cells verified red individually — the doubled backslash, nothingSays's note branch, the piped fail(2), and truncateQuestion's elision.
          round: 9
        - id: BR-20
          disposition: addressed
          note: replayInPlace routes through nothingSays; one production occurrence of the replay sentence in the whole tree, and the parameter is inSession where true is correct.
          round: 9
        - id: BR-21
          disposition: addressed
          note: One "lost the terminal" site in cmd/, via lostTerminal. The doc comment above it still predicts a fifth call site — that belongs to BR-50.
          round: 9
        - id: BR-47
          disposition: not-addressed
          note: 'Third round, same failure mode — the enumeration was re-run and again missed what the same commit created: Store.SetUserModel and carriedCR are both absent from the plan.'
          round: 9
        - id: BR-48
          disposition: not-addressed
          note: Still no Revisions entry for round 2's four forks, round 6's Task 11 item, round 3's askScoped/SetUserModel, or the close round.
          round: 9
        - id: BR-49
          disposition: not-addressed
          note: Unchanged — replRaw still passes the seam straight to readKeys with no policy, and 44 test call sites still pass nil.
          round: 9
        - id: BR-50
          disposition: not-addressed
          note: All four residues present verbatim — replraw.go:132's "adds a fifth", capture.go's CaptureAsk guard, repl_test.go's t.Context() failure arm, ask.go's per-question Deck().
          round: 9
        - id: BR-51
          disposition: addressed
          note: 'The restore cell is genuinely pinned — omitting defer restore() reddens TestCtrlCQuitsAgainOnceTheAnswerIsOver (verified). Two residuals raised separately: that test is racy, and the enumeration''s qcancel row is false.'
          round: 9
        - id: BR-52
          disposition: addressed
          note: Dropping the cancel-path recordExchange reddens TestAnAnswerTheUserReadSurvivesHowItEnded/stopped_by_the_user (verified).
          round: 9
        - id: BR-53
          disposition: not-addressed
          note: The site defect is fixed and behaviourally verified, but no test fails without carriedCR, and the rule the finding asked for is unapplied — Write and consumed are still two derivations.
          round: 9
      findings:
        - id: BR-54
          severity: Important
          title: TestCtrlCQuitsAgainOnceTheAnswerIsOver fails 12 of 30 runs on unmutated HEAD
          detail: |-
            This is the 2nd finding in family unsynchronised-test-observation. BR-25
            fixed the instance (data races on a bytes.Buffer); state the rule instead.
            Rule - a test must synchronise on the state it ASSERTS, not on a proxy that
            merely precedes it. askrun_test.go:831 waits for "insincerely", the LAST
            text delta of stream-sample.sse (line 35 of 45), then immediately writes
            \x03 and requires the scope to already be restored. Everything between is
            unsynchronised: the remaining SSE frames parse, Stream returns,
            recordExchange runs, and runAsk's deferred CaptureAsk does a real
            AppendEvent to DISK (ask.go:147-151) - all inside askScoped, before
            restore(). Lose that race and readKeys swallows the \x03 as
            scope-consumed, quit never closes, and the test fails at its 5s deadline
            printing the message it reserves for the real defect. Measured on
            unmutated HEAD: `-count=30` isolated gives 12 failures; under full-package
            load it is rarer (3 of 3 clean in a dedicated sweep, one spontaneous
            failure across ~10 full-package runs during this review). It is the only
            test in the file that drives a COMPLETING stream and then asserts
            post-scope state - the eight `Stall: true` tests never leave the scope - and
            it is the sole defender of the cell BR-51 named, while the close's
            --verified evidence is a `go test ./...` run. Fix: wait on a happens-after
            marker for restore() rather than on answer text; askInSession
            (replraw.go:164-165) writes "\r\n" then draw() strictly after askScoped
            returns. Consider separately moving the CaptureAsk disk write outside the
            scope, which also shrinks the window where a user's real Ctrl-C is
            silently swallowed.
          family: unsynchronised-test-observation
          round: 9
        - id: BR-55
          severity: Important
          title: Four measured doc claims contradict the code, all created by the last two rounds
          detail: |-
            This is the 11th finding in this family; the rule has been stated five
            times, so do NOT patch the four lines. Rule, in the shape this window
            needs it - a measured claim written into a comment is a claim, and it is
            falsified the same way any other absolute is: by running the mutation it
            names. Measured: (1) ask.go:43 records
            `omit defer qcancel()  1 test red - the question's context leaks`;
            measured, `go test ./cmd/define/` is ok 58.078s with 0 tests red and
            `go vet ./cmd/define/` silent, because qcancel is passed as a value to
            interrupts.Set so lostcancel never fires - which is exactly what BR-51's
            own detail reported, so the commit answering it wrote the opposite into
            the table whose thesis is "the deliverable is the enumeration"; the cell
            is a real per-question context leak and is undefended. (2) ask.go:38 says
            `omit interrupts.Set  10 tests red` while askrun_test.go:800, same commit,
            says "reddens five tests"; I measure 4 top-level / 10 with subtests - two
            numbers in one commit is the signature of a recollection, not a
            measurement. (3) atlas/define.md:602-603 still says the raw loop's closure
            "is where M2's streaming writer and scoped interrupt hang", which round 3
            falsified by moving the scope into askScoped - replraw.go:142-143 says so
            explicitly, so the code comment was corrected and the atlas paragraph that
            OWNS the claim was left standing, BR-28/BR-44's rule a third time.
            (4) store/mem.go:18 says "SetUserModel is #17's to add ... until then"
            while mem.go:91, same commit, implements it and the suite writes through
            it. A fifth, adjacent: a CONFIGURED but unreachable model prints
            "no model configured" and exits 1 while the question IS recorded as an
            asked event (probed against http://127.0.0.1:1), where README:97-101 keys
            "not recorded" on "no model configured" - and it is the one outcome cell
            TestAQuestionIsRecordedWhateverBecameOfTheAnswer does not enumerate.
          family: doc-overstates-code
          round: 9
      blocked: true
---

# Gate ledger — tools#16 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-23T15:46:30-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `loop-shell-branch-untested` deleting both loops' forced-"?" cmdAsk branches leaves the whole suite green
  Verified by mutation: removing `case cmdAsk:` from replLines (repl.go:219)
  and the whole `if cmd.kind == cmdAsk` block from runEditor
  (replraw.go:199-211) passes `go test ./cmd/define/` in full. Every
  loop-level ask test uses the unforced route; routeFor returns "question"
  for cmdAsk without entering a loop. hist.Add(submitted.String()) at
  replraw.go:204 is unpinned by the same gap. The issue's ticked Done-when
  row claims each hatch is exercised by a test that fails when the hatch is
  removed; that holds at the parser only. Add one forced test per loop —
  both pass against today's code, which is the point.
- **BR-2** [Important] `test-asserts-nothing` TestAQuestionIsRecalledByUpArrow passes with history recording removed
  askroute_test.go:95 asserts stdout contains the question, but the raw
  editor re-renders the line on every keystroke, so the question is in
  stdout from typing alone. Verified: replacing hist.Add(line) at
  replraw.go:292 with a no-op leaves the test green. Task 4's contract row
  "does add the line to editor recall (hist.Add)" has no falsifiable test.
  Assert against an injected History, and pin the recall rendering by
  comparing runs with and without the KeyUp.
- **BR-3** [Important] `readme-surface-gate` README update appears missing for the "?" and "\" hatches and question routing
  README.md documents exactly this class — the "/"-in-column-1 convention
  (README.md:120), the interactive key table (:43-50), and the exit-code
  contract (:111-116) — and is untouched in the window, while atlas,
  fs.Usage and /help were all updated. New user-facing surface a reader
  types: "?" forces a question, "\" forces a lookup, an unrecognised line
  that reads as a question goes to the model, and a question with no model
  configured exits 1.
- **BR-4** [Important] `oneshot-kind-coverage` the one-shot sends cmdNothing to the dictionary as an empty word
  main.go:342's default arm handles cmdCommand and cmdAsk and falls through
  to defineOnce for every other kind, but parseREPLLine now returns
  cmdNothing with word=="" for two inputs it never used to produce. Measured:
  `define "?"` and `define "\"` print `define: : no dictionary entry`, exit
  1, and Capture("", false, opt) appends a ReviewEvent{Word: ""} that
  complete() (store/event.go:30) discards at read time — junk in the
  append-only log #8 and #17 fold over. cmd.note is dropped, so the one-shot
  never shows `type a question after "?"`. Handle cmdNothing before
  defineOnce and never pass an empty word to lookupAndRender.
- **BR-5** [Minor] `raw-mode-message-placement` the bare-"?" note is printed without eraseLine, unlike its sibling branch
  replraw.go:215 writes `define: %s\r\n` with the cursor still on the typed
  prompt line, so it renders as `› ?define: type a question after "?"`.
  replayInPlace (replraw.go:264) prefixes eraseLine for the same class of
  message.
- **BR-6** [Minor] `raw-mode-message-placement` forced and unforced asks render with different vertical spacing
  The unforced path emits runEditor's pre-submitLine "\r\n" (replraw.go:227)
  in addition to askInSession's own, so its message sits one blank line
  lower than the forced path's. Two routes the plan wanted visually
  identical.
- **BR-7** [Minor] `doc-overstates-code` atlas says single-word lines are never questions; the trailing-"?" arm says otherwise
  atlas/define.md:488. The "?" arm precedes the len(fields) < 2 check, so
  readsAsQuestion("why?") and readsAsQuestion("sycophanti?") both return
  true (measured). Qualify the sentence or move the arm.
- **BR-8** [Minor] `stdlib-reuse` hand-rolled contains() where slices.Contains exists
  question.go:83. The module is go 1.26 and the package's tests already
  import slices (ARCH-DRY).

## Round 2 — 2026-08-23T16:05:39-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: deleting either loop's cmdAsk branch now reddens the matching forced test; hist.Add(submitted) is pinned too.
- BR-2 — addressed — Asserted against an injected memHistory; reverting hist.Add(line) in submitLine reddens the unforced subtest.
- BR-3 — addressed — README documents both hatches, routing, the model requirement and the exit codes; two of its claims are inexact — see the new doc-overstates-code finding.
- BR-4 — addressed — Exhaustive dispatch at main.go:346; removing it reddens the test. The test's capture half is dead, raised separately under test-asserts-nothing.
- BR-5 — addressed — eraseLine present at replraw.go:223 and measured on the byte stream; unpinned by any test, folded into the family finding.
- BR-6 — addressed — Measured: forced and unforced asks both emit exactly one leading CRLF plus askInSession's, identical tails; unpinned, folded into the family finding.
- BR-7 — addressed — atlas/define.md now qualifies the sentence and names "why?" explicitly.
- BR-8 — addressed — question.go:62 uses slices.Contains; the hand-rolled contains() is gone.

### Raised

- **BR-9** [Important] `doc-overstates-code` "-raw never asks" is guarded on the unforced route only, so 3 of 6 ask entry points ignore it
  This is the 2nd finding in family doc-overstates-code, so the deliverable
  is the rule, not the site. Rule - an absolute stated in README/atlas or in
  a test name must name the enumeration it quantifies over, and every cell
  must be guarded and asserted. Here the enumeration is
  {forced, unforced} x {one-shot, piped loop, raw editor}; the !opt.raw
  guard sits at main.go:425 on the unforced path only. Measured -
  `define -raw "?what is X"` and `echo "?what is X" | define -raw` both
  print the no-model message and exit 1, and runEditor with opt.raw=true on
  "?why" does the same, while TestRawNeverAsks (askroute_test.go:202) stays
  green through all three. README.md:79 states the absolute; main.go:419-424
  names the M2 cost (a network call for a line a script piped in). Same
  family, smaller prevalence - README.md:69 and :76 quote the miss message
  as `not found` where the program prints `no dictionary entry`.
- **BR-10** [Important] `test-asserts-nothing` BR-4's test injects a countingCapturer that withStore discards, so its log-junk assertion is dead
  This is the 2nd finding in family test-asserts-nothing, so state the rule -
  a test that injects a double must assert the injection took effect, or
  inject at the seam production reads. askroute_test.go:178 sets d.newStore
  on a deps whose capture testDeps already filled (main_test.go:14), and
  withStore only fills nils (main.go:96), so cap is discarded and
  `for _, w := range cap.calls` never iterates. Probe-verified - a panic
  inside countingCapturer.Capture leaves TestOneShotRejectsAHatchWithNothingAfterIt
  green while TestAQuestionIsNotCaptured panics at once. So the harm BR-4
  actually named, a ReviewEvent with an empty Word that complete() discards
  at read time, has no falsifiable test. Measured prevalence - 1 of 1
  d.newStore injection in the package is dead.
- **BR-11** [Minor] `raw-mode-message-placement` neither BR-5's nor BR-6's fix is pinned - both can be reverted with the suite green
  This is the 3rd finding in family raw-mode-message-placement. Do not
  re-fix the sites. Rule - every message class the raw loop writes gets an
  assertion on the emitted byte stream, as TestEditorLoopUsesCarriageReturnsInRawMode
  (editorloop_test.go:143) already does for the definition path. Verified -
  removing eraseLine from replraw.go:223 and re-adding askInSession's own
  "\r\n" together leave go test ./cmd/define/ fully green. Measured
  prevalence - 3 of 3 raw-mode message placements introduced by this issue
  (the bare-"?" note, the forced ask, the unforced ask) are unasserted,
  which is why the family recurs.
- **BR-12** [Minor] `forced-route-enumeration` the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
  Two instances of one rule - a behaviour specified for the unforced route
  is silently inherited by the forced one. (a) submitLine adds cmd.word
  (replraw.go:299), which is post-strip, so typing `\how so` records
  "how so"; Up-arrow then Enter asks the line you had just forced to a
  lookup. The cmdAsk branch records submitted.String() (replraw.go:207), so
  "?why" stays forced. Measured through runEditor with an injected
  memHistory. (b) askUnavailable (ask.go:18) tells a forced question about a
  real headword that it "is not a word" - "?why" prints
  "no model configured; `why` is not a word". The sentence is true by
  construction only on the unforced route.
- **BR-13** [Minor] `plan-bookkeeping` the plan's step checkboxes are 0 of 60 ticked while M1's tasks are complete and committed
  workshop/plans/000016-console-qa-plan.md - Tasks 1-6 shipped in five
  commits, but no step box is ticked. The two most recent archived plans set
  the convention (000015 39/39, 000014 33/33; 000011 did not), so this is
  soft, but leaving M1's boxes open makes "where does M2 resume" ambiguous
  when the plan is picked up in a fresh session.

## Round 3 — 2026-08-23T16:29:26-07:00 (claude) — BLOCKED

### Disposed

- BR-9 — addressed — mayAsk moved into ask(); deleting that guard reddens exactly the three forced cells. Assertion strength raised separately.
- BR-10 — addressed — Injection moved to d.capture and probe-verified live; making withStore overwrite it reddens TestTheCapturerInjectionIsLive.
- BR-11 — addressed — Both fixes now redden on a faithful revert — eraseLine removal and the BR-6 CRLF move each go red.
- BR-12 — not-addressed — Behaviour is correct but neither half is pinned - submitLine's recallLine and ask's whole q.forced branch both revert with the suite green.
- BR-13 — addressed — Chunk 1 is 29/29 ticked and Chunk 2 is 0/31, so the M2 resume point is unambiguous.

### Raised

- **BR-14** [Important] `test-asserts-nothing` assertDidNotAsk passes for the failure it exists to catch, in 2 of TestRawNeverAsks's 6 cells
  This is the 3rd finding in family test-asserts-nothing, so the deliverable
  is the rule. Rule - an assertion that pins "X did not happen" must assert
  the positive observable that distinguishes X from every other outcome, not
  the absence of one string. assertDidNotAsk (askroute_test.go:269) checks
  only that stderr lacks "no model configured". Measured - removing
  mayAsk(opt) from the miss branch at main.go:425 (a plausible cleanup; the
  comment there flags the redundancy with ask's own guard) reddens
  one-shot/unforced via its EXIT CODE only, while piped/unforced and
  editor/unforced stay green printing
  `define: -raw does not ask; drop -raw, or drop the "?"` for a line
  containing no "?" at all. atlas/define.md:538 and the test's own comment
  both claim the enumeration "covers every cell"; 4 of 6 can fail, 2 cannot.
  The distinguishing observable is cheap - assert stderr CONTAINS
  `no dictionary entry`, the miss the scripting contract promises.
- **BR-15** [Important] `doc-overstates-code` the piped loop discards the usage code ask() computes, so -raw plus "?" exits 1 where README states 2
  This is the 3rd finding in family doc-overstates-code, so state the rule -
  a non-zero code a dispatch computes inside replLines must survive the loop,
  and every exit-code absolute in README must be measured across
  {one-shot, piped, editor} before it is written. askHere (repl.go:189-193)
  collapses ask's return into anyFailed, discarding the 2; the cmdCommand
  branch 45 lines below feeds cmdCode correctly (repl.go:236-240) and the
  comment at repl.go:180 names this defect by number - BR-16, "Collapsing it
  into anyFailed made echo /histry | define exit 1 where dispatchCommand
  computes 2". Measured against the built binary -
  `define -raw '?why'` exits 2, `echo '?why' | define -raw` exits 1,
  `echo '/histry' | define` exits 2. README.md:81 states the claim
  unqualified ("on either route ... a usage error (exit 2)") and
  TestRawNeverAsks passes wantCode 0 for both piped cells, so nothing
  asserts it. Sweep the other three absolutes too - bare "?" and bare "\"
  exit 2 one-shot but 0 piped.
- **BR-16** [Minor] `raw-mode-message-placement` the two ask rows of TestRawLoopMessagePlacement assert nothing about the ask message
  This is the 4th finding in family raw-mode-message-placement. Do not
  re-fix the site. BR-11's rule was executed for 1 of the 3 message classes
  the test enumerates - the ask rows assert only
  assertNoBareNewline(stdout), but ask writes to STDERR, so they never touch
  the message. Verified - removing cooked(...) from askInSession
  (replraw.go:134), which in production is the only reason the ask message's
  "\n" translates at all, leaves go test ./cmd/define/ fully green. The
  rig's cooked is a no-op closure (editorloop_test.go:34), so a recording
  cooked is the missing observable.
- **BR-17** [Minor] `plan-bookkeeping` the plan still describes an M1 the code no longer implements, despite round 2 recommending the Revisions entry
  workshop/plans/000016-console-qa-plan.md - last Revisions entry is
  "plan-quality round 2"; nothing records the two boundary rounds. Stale in
  four places - Task 3's snippet gives the miss condition as
  `!literal && readsAsQuestion(word)` where the code is
  `!cmd.literal && mayAsk(opt) && readsAsQuestion(word)`; "-raw" and
  "mayAsk" appear nowhere in the plan; Task 4 names askUnavailable where the
  code has ask(opt, errOut, question); and Core concepts lists none of
  lookupOutcome, recallLine, question or mayAsk, while placing ask.go in M2.
  Rule - when a boundary round changes what the code does relative to the
  plan, the plan gets the entry in the same commit as the code.
- **BR-18** [Minor] `forced-route-enumeration` a bare "\" with a current word replays audio while a bare "?" gets its note
  repl.go:53 - the "\" arm strips the prefix and falls into the
  empty-word test, so `\` alone with hasCurrent returns cmdReplay.
  TestParseREPLLine's "a bare backslash is blank" row only covers
  hasCurrent=false, so the asymmetry between the two hatches at the prompt
  is unasserted. Harmless today; worth one row.

## Round 4 — 2026-08-23T16:50:25-07:00 (claude) — passed

### Disposed

- BR-12 — not-addressed — Third round unchanged - hist.Add(cmd.recallLine())->hist.Add(line) and deleting ask's whole q.forced branch both revert with the suite green.
- BR-14 — addressed — assertDidNotAsk asserts the positive observable; dropping the miss-branch mayAsk now reddens 3 of 3 unforced cells, not 1.
- BR-15 — addressed — fail(code) sink reddens piped/forced on revert; measured 8 of 8 exit-code cells on the built binary against README. Coverage of the swept cells raised in N-1.
- BR-16 — addressed — Recording cooked is a live observable - removing cooked(...) from askInSession reddens both ask rows.
- BR-17 — not-addressed — Plan still has three Revisions entries, none a boundary round; askUnavailable still named, -raw/mayAsk absent, Core concepts still places ask.go in M2.
- BR-18 — addressed — Reverting the bare-backslash arm reddens both hasCurrent rows of TestParseREPLLine.

### Raised

- **BR-19** [Important] `test-asserts-nothing` no test asserts what any of #16's messages say, and a doubled backslash shipped as a result
  This is the 4th finding in family test-asserts-nothing, so the deliverable
  is the rule. Rule - an assertion on a message must compare the bytes the
  user receives against a literal expectation written in the test;
  referencing the production constant asserts only that a branch was
  selected, not that its text is right. Shipped defect - noteEmptyLiteral
  (repl.go:44) is a Go RAW string literal containing `\\`, so both
  `define '\'` and `echo '\' | define` print
  `define: type a word after "\\"` with a doubled backslash, while its
  sibling bare-"?" note is correct. The only assertions are repl_test.go:43
  and :45, which compare note to noteEmptyLiteral - the constant to itself.
  Measured prevalence, 4 of 5 message/code behaviours this milestone
  introduced are unasserted at the point of delivery, all GREEN on revert -
  deleting nothingSays's `if c.note != ""` early return; deleting
  `if cmd.note != "" { fail(2) }` at repl.go:261 (BR-15's own sweep, correct
  in code but unpinned, so README's "a bare ? or \ with nothing after it"
  exit 2 has no piped assertion); and replacing truncateQuestion(q.text)
  with q.text in ask's two messages. Only the placement fix (eraseLine) goes
  red, which is why the family recurs - each round pinned a placement and
  never a text. One table over {?, \} x {one-shot, piped} asserting exit code
  AND literal stderr text closes all four rows.
- **BR-20** [Minor] `doc-overstates-code` nothingSays is documented as "the ONE place" while replayInPlace holds a live duplicate of its replay sentence
  This is the 4th finding in family doc-overstates-code. Do not re-fix the
  site. Rule - a consolidation claimed as "the ONE place" must be verified by
  enumerating the copies it consolidated; a uniqueness claim is an absolute
  and it quantifies over the whole tree. nothingSays's doc comment
  (repl.go:100-104) and atlas/define.md:553 both assert it is the one place
  that answers "this line meant nothing - why", but replayInPlace
  (replraw.go, `case current == ""`) still holds a byte-identical copy of
  that sentence, and it is the LIVE path for the raw editor's bare Enter
  with nothing current - pinned by TestEditorLoopBareEnterWithNoCurrentWord
  (editorloop_test.go:235). Same rule, second half - nothingSays(cmd, true)
  at repl.go:257 hardcodes true where sess.hasCurrent() is in scope, and a
  note-less cmdNothing is reachable ONLY when hasCurrent is false (a blank
  line with a current word parses to cmdReplay), so that site selects "press
  return to replay the last one" in the one state where there is nothing to
  replay. Measured on the built binary. The text is pre-#16 and faithfully
  ported; the parameter introduced to distinguish the two cases is what is
  fed a literal.
- **BR-21** [Minor] `stdlib-reuse` four byte-identical "lost the terminal" blocks in runEditor, two added by this diff
  replraw.go:195, :211, :241, :250 - the same
  finish(); Fprintf("define: lost the terminal: %v"); return 1 three-liner,
  up from 2 copies before this window. M2's Task 11 adds streaming inside
  askInSession and is positioned to add a fifth. One helper, on the same
  argument nothingSays was extracted on (ARCH-DRY).

## Round 5 — 2026-08-23T23:03:30-07:00 (claude) — BLOCKED

### Raised

- **BR-22** [Critical] `policy-in-io-shell` "Recently in the deck" sends the twelve OLDEST deck words, unreversed
  cmd/define/ask.go:180-187. Store.Deck() is newest-first (store.go:14,
  sortDeck at mem.go:62-69) and lastN takes the tail, so a deck larger than
  maxContextWords sends the oldest twelve and drops the most recent. Measured
  with a 15-word deck: the section renders "wl, wk, ... wa" and the three
  newest words never reach the model. The code's own comment states both
  intentions it fails ("reads better oldest-first", "the newest are the ones
  worth keeping"), and README:82 plus atlas/define.md:522 both promise "the
  recent deck". The selection is a PURE decision inlined in the IO gatherer,
  which is why no test above the bound exists (ARCH-PURE) — fix by extracting
  recentDeck() beside recentTurns and table-testing it.
- **BR-23** [Critical] `forced-route-enumeration` One-shot unforced ask puts the question text into session.current and the event log's word field
  This is the 3rd finding in family forced-route-enumeration; earlier rounds
  fixed instances, so fix the CLASS. cmd/define/main.go:387 passes
  &session{current: oneShot.word}, where oneShot.word is the line the
  dictionary just missed — the question. Measured with a live seam and a real
  YAML store: the prompt renders "## The word on screen\n<the question>" with
  no entry, and recordAsked (ask.go:150) writes ReviewEvent{Word: "<the
  question>"} into the append-only log #17 folds over. That violates plan D2
  and Task 4's contract table ("does not set sess.current"). The forced
  one-shot cell passes &session{} and is correct, so the two routes diverge.
  The rule: every fact the ask path carries is quantified over {forced,
  unforced} x {one-shot, piped, editor} and each cell asserted for THAT fact.
  M2 added three such facts (which session, where the answer is written,
  whether the interrupt is scoped); measured coverage is 2 of 18 cells.
- **BR-24** [Important] `silent-error-swallow` gatherAskContext discards UserModel's error, defeating the reason it returns one
  cmd/define/ask.go:172-174 writes `if model, err := d.deck.UserModel(); err
  == nil`. store/yaml.go:44-48 justifies returning an error precisely to
  prevent this: "silently answering \"\" for a model that exists would make
  every answer pitched at the wrong level with no way to tell." Warn on
  stderr (the storeCapturer/storeHistory precedent) or delete the
  justification.
- **BR-25** [Important] `unsynchronised-test-observation` go test -race ./cmd/define/ now fails; two new tests read a bytes.Buffer concurrently
  Verified: base 993fccc is race-clean, HEAD 686913c reports eleven races and
  two failures. askrun_test.go:147-151 spins on out.Len() while runAsk writes
  out; askrun_test.go:267,280 read out.String() via waitFor while runEditor
  writes it. The package already owns the correct shape — ptyOut's
  mutex-guarded buffer at pty_conformance_test.go:88-124 — so this is also
  ARCH-DRY. Extract one syncBuf and use it in all three places.
- **BR-26** [Important] `plan-contract-drift` The plan's askCapturer integration point was not built; events append through a second write path
  Core concepts -> Integration points lists askCapturer on Capturer in
  capture.go wrapping the event append. The code appends directly via
  d.deck.AppendEvent from ask.go:145-152. Behaviour is safe (DEFINE_NO_CAPTURE
  leaves sd.deck nil so recordAsked no-ops), but it adds a second writer to
  the event log beside capture and makes main.go:31-34 stale ("capture is the
  only thing that RECORDS lookups. deck ... --forget deletes through it").
  Either route it through the seam or add a "## Revisions" entry and fix the
  comment.
- **BR-27** [Important] `readme-surface-gate` README does not say that question text is persisted to events/
  This is the 2nd finding in family readme-surface-gate, so state the rule
  rather than adding one line. README:95-105 is the block documenting what
  define writes to the working directory; M2 makes every question's text
  persist into events/*.yaml and that block does not say so. The rule: any
  new record type or field written into the working directory is named in
  that block in the same change. Enumeration: {words/, events/,
  user-model.md} x {looked-up, reviewed, asked}; README currently names 2 of
  3 files and 0 of 3 kinds.
- **BR-28** [Important] `doc-overstates-code` atlas/define.md:182-186 still states the pre-M2 cancellation model the diff disproved
  This is the 5th finding in family doc-overstates-code; do NOT fix only this
  paragraph. It claims Ctrl-C is "not a signal", that the piped path "still
  relies on" signal.NotifyContext, and that the key reader calls cancel().
  All three are false after this diff: the pty suite measured \x03 arriving
  as SIGINT (D5's premise), repl.go:180 detaches with WithoutCancel, and
  readKeys calls interrupts.Fire(). The diff corrected the code comment at
  rawterm.go:44-46 and added a NEW atlas section at :544 while leaving the
  old statement standing. The rule: when a diff corrects a claim in a code
  comment, every restatement of that fact elsewhere is a consumer of the same
  source and must be swept in the same change (ARCH-PURPOSE shadow-sweep) —
  grep the fact, do not append beside the stale copy. Measured: 1 of 2
  NotifyContext mentions in atlas/ is now false.
- **BR-29** [Important] `loop-shell-branch-untested` The piped loop's ask wiring is unpinned — answer destination and session both deletable green
  This is the 2nd finding in family loop-shell-branch-untested; state the
  rule. Two mutations at repl.go:261 each leave go test ./cmd/define/ fully
  green: replacing stdout with io.Discard (the answer goes nowhere), and
  replacing &sess with &session{} (the piped loop loses multi-turn and
  session words). lessons.md define #15 already states the rule; M2 obeyed it
  for runEditor and skipped it for replLines and the one-shot. That skipped
  cell is where the one-shot Critical lives — the same six-cell enumeration
  fixes both.
- **BR-30** [Important] `test-asserts-nothing` Four claims in this diff survive the mutation they exist to catch
  This is the 5th finding in family test-asserts-nothing; do NOT fix the four
  instances, state the rule. Measured individually, all green: (1) the
  key-channel buffering rawterm.go:49-54 calls "load-bearing" — make(chan
  Key) instead of make(chan Key, 256); (2) session words reaching the prompt,
  an issue Done-when row — SessionWords: nil in gatherAskContext; (3)
  recordExchange's empty-answer guard, session.go:44-46 — delete it; (4)
  askrun_test.go:293-296 claims to pin "the interrupt scoping and the CRLF
  writer" for both routes but passes nil for interrupts, so it can only pin
  the writer. Row (2) is the BR-2 shape exactly: the test asserts
  "sycophantic" is in the prompt, but CurrentWord supplies that string, so
  the SessionWords assertion is satisfied by a different source. The rule: an
  assertion that a value reached an output must use a value only that source
  can supply. Also applies to
  TestEditorCtrlCMidStreamReturnsToThePrompt/the_signal_transport, which
  calls interrupter.Fire() directly and never touches d.notifySignals.
- **BR-31** [Important] `raw-mode-message-placement` assertNoBareNewline is vacuous for all three rows of TestRawLoopMessagePlacement
  This is the 5th finding in family raw-mode-message-placement; state the
  rule. The helper does strings.TrimSuffix(s, "\n") at askroute_test.go:395
  to excuse the loop's post-finish() newline, and all three rows write
  exactly one line to stderr — so the excused newline is the only one under
  test. Measured: unwrapping stderr from crlfWriter at replraw.go:141 leaves
  the suite green, and changing the bare-? note's "\r\n" to "\n" at
  replraw.go:226 leaves TestRawLoopMessagePlacement green. The test's own
  header claims "EVERY message this loop writes ... carries its own carriage
  returns" — vacuous for 3 of 3 rows. The rule (BR-14's rule applied to
  placement): a placement assertion asserts the POSITIVE observable, that the
  message's own terminator IS "\r\n", never the absence of a bare "\n".
- **BR-32** [Important] `plan-bookkeeping` 31 unticked M2 plan steps and no Revisions entry for any boundary round or M2 design departure
  This is the 3rd finding in family plan-bookkeeping; state the rule.
  workshop/plans/000016-console-qa-plan.md Chunk 2 has 31 unticked steps and
  0 ticked, while the issue's ## Plan and the project row both mark M2 [x].
  The plan still carries three ## Revisions entries, none of which is a
  boundary round (BR-17, raised in M1, never disposed), and none covering
  M2's three design departures. The rule: the plan artifact's state is part
  of the milestone deliverable — the checkbox sweep plus a ## Revisions entry
  for every fork taken differently from the plan happen in the
  milestone-close commit, not at issue close.
- **BR-33** [Minor] `doc-overstates-code` runAsk's `parent := ctx` is a bare alias with no derived context
  ask.go:100-102 copies llmcheck.go:45-47's idiom, but llmcheck derives a
  WithTimeout child so parent and ctx genuinely differ. Here they are the
  same value and the comment describes a distinction that does not exist. The
  observed cancel path also returns ErrTruncated, not ErrUnavailable, so the
  failure mode the comment names is the pre-text case only.
- **BR-34** [Minor] `plan-contract-drift` Core concepts table cites the wrong file for `exchange` and `interrupter`
  `exchange` is in askctx.go:9-12, not session.go; `interrupter` is in the
  new interrupt.go, not rawterm.go. Both entities exist, are pure and are
  tested, so this is a stale table rather than a missing deliverable —
  correct the rows in the plan revision.
- **BR-35** [Minor] `stdlib-reuse` bytesReader is strings.NewReader under a new name; keysFor is dead
  This is the 3rd finding in family stdlib-reuse. interrupt_test.go:98
  defines `func bytesReader(s string) io.Reader { return
  strings.NewReader(s) }` for one call site. The rule (same as
  contains->slices.Contains two rounds ago): before adding a helper, check
  whether the stdlib already names it. Separately, askrun_test.go:334-347's
  keysFor is defined and never called, with a doc comment describing a use
  case that does not exist.
- **BR-36** [Minor] `doc-overstates-code` DEFINE_NO_CAPTURE now also suppresses READING user-model.md and the deck
  openStore leaves deck nil under noCapture, so gatherAskContext (ask.go:167)
  skips both the learner model and the deck. README documents the variable as
  "write nothing in this directory"; answers silently become un-adapted. Also
  note unavailable() reports "no model configured" for a configured but
  unreachable model — plan-specified, but the same misdiagnosis surface as
  the already-filed tools#19.
- **BR-37** [Minor] `dead-branch` askInSession always returns nil, so lostTerminal is unreachable at both ask call sites
  replraw.go:130-155 no longer calls cooked(), so the closure cannot fail;
  the `if err := askInSession(...); err != nil { return lostTerminal(err) }`
  guards at both call sites are dead. Also crlf.go:28 returns 0 on a write
  error rather than the bytes consumed, and reports len(p) on a short
  underlying write; and ask.go:114 prints a newline on cancel even when no
  delta ever arrived.

## Round 6 — 2026-08-23T23:40:16-07:00 (claude) — BLOCKED

### Disposed

- BR-22 — addressed — newestFirst takes the head and reverses; reverting to lastN reddens TestTheDeckSectionCarriesTheNewestWords (verified).
- BR-23 — addressed — main.go:393 passes &session{}; restoring current: oneShot.word reddens TestOneShotQuestionHasNoCurrentWord (verified).
- BR-24 — not-addressed — The warn is correct and reachable, but no test fails without it — and failingStore.UserModel, added by this same diff, is at zero call sites.
- BR-25 — addressed — go test -race ./cmd/define/... is ok at HEAD; syncBuf is used in all three places and ptyOut folds onto it.
- BR-26 — addressed — capture.go is the only non-test AppendEvent caller; a no-op CaptureAsk reddens three tests (verified). Plan Revisions records the departure.
- BR-27 — addressed — README:99-103 names events kinds, user-model.md and "answers are NOT stored"; pinned by TestTheEventLogHoldsQuestionsAndNotAnswers.
- BR-28 — addressed — atlas/define.md:182-191 rewritten; the surviving NotifyContext mention at :676 is about the one-shot path and is still true. The family recurs elsewhere — raised separately.
- BR-29 — addressed — Both repl.go:261 mutations now redden TestThePipedLoopsAskWiring (verified). The sibling cells the finding did not name are raised separately.
- BR-30 — not-addressed — Four of the five named items are fixed and mutation-verified red; the fifth is untouched — the "signal transport" subtest still calls interrupts.Fire() directly and never reaches d.notifySignals, and no Revisions entry records it.
- BR-31 — addressed — assertCRLFTerminated counts terminators; both the unwrapped-stderr and the bare-newline mutation now redden TestRawLoopMessagePlacement (verified).
- BR-32 — addressed — 60 of 60 plan checkboxes ticked, and a "M2's design departures, and the M1 rounds" Revisions entry now exists.
- BR-33 — addressed — The parent alias is gone; runAsk asks ctx.Err() directly.
- BR-34 — addressed — Core concepts now cites askctx.go for exchange and interrupt.go for interrupter.
- BR-35 — addressed — Neither bytesReader nor keysFor appears anywhere in cmd/define.
- BR-36 — addressed — README:118-120 documents that DEFINE_NO_CAPTURE also suppresses reading the deck and user-model.md.
- BR-37 — addressed — askInSession returns nothing and lostTerminal still has two live call sites; crlf reports caller-unit progress; the cancel newline is guarded by answer.Len(). The new consumed() helper has its own defect, raised separately.

### Raised

- **BR-38** [Important] `loop-shell-branch-untested` Three cells of the ask-wiring table still survive their mutation with the full suite green
  This is the 3rd finding in family loop-shell-branch-untested, and the 2nd
  round in which the enumeration BR-23 wrote out was answered with targeted
  tests instead of the table. Measured against the whole ./cmd/define/ suite,
  not a -run subset: replraw.go:161 &sess -> &session{} leaves it green (ok,
  28.160s) — the raw editor is the primary UI and multi-turn is a Done-when
  row; main.go:366 and main.go:393 stdout -> io.Discard, applied together,
  leave it green (ok, 28.391s) — `define "what is the difference to
  obsequious?"` printing nothing is caught by nothing. Coverage of the
  {one-shot, piped, editor} x {which session, answer destination, interrupt
  scope} table is 4 of 7 applicable cells. Do NOT add three more one-off
  tests: write the table as one fixture-driven test whose rows ARE the cells,
  the way TestRawNeverAsks already does for its six.
- **BR-39** [Important] `doc-overstates-code` Four measured doc claims contradict the code, three of them created by this window
  This is the 8th finding in this family; the rule has been stated three
  times (BR-9's absolute-names-its-enumeration, BR-28's shadow-sweep) and
  keeps recurring, so the escalation must be mechanical. Measured:
  (1) replraw.go:143 "It runs COOKED for the same reason a command does",
  five lines above the code and comment saying it streams RAW through
  crlfWriter; (2) atlas/define.md:574 restates ask's pre-M2 signature `ask(opt
  options, errOut io.Writer, q question)` — 1 of the 3 quoted Go signatures in
  that file is stale and it is the one this diff changed; (3) README:78
  "Ctrl-C stops the answer rather than the session" — measured through the
  injected signal transport with stdin a tty and stdout redirected, the LINE
  loop ends the session, because replLines' askHere (repl.go:261) passes the
  loop's own ctx that the default sink cancels; true in 1 of 2 interactive
  loops; (4) README:97 "Every question you ask is recorded too, by its text" —
  measured 0 events for an errored (400) ask, 0 for an unwired seam, and 0
  events plus 0 turns for a cancelled ask; true in 1 of 4 outcome cells. The
  mechanical fix: atlas and README stop restating signatures and unqualified
  absolutes, and each surviving absolute gets a named-enumeration row test
  like TestRawNeverAsks.
- **BR-40** [Important] `user-text-in-record-format` The event log's first free-form user field has no test defending the record-boundary invariant
  Until this window every value written to events/*.yaml was a single
  dictionary headword; `question:` is now arbitrary text the user typed. The
  reader's record boundary is a literal top-level "- " (yaml.go:307) and the
  only thing keeping user text off column 0 is that yaml.Marshal indents
  block scalars. I verified today's behaviour is correct — five adversarial
  questions round-trip intact, including one whose text is a complete forged
  "- word: injected / kind: looked-up / at: ..." record — but
  storetest/suite.go:52 round-trips one plain question and nothing pins the
  invariant. A regression corrupts an append-only log #17 folds over,
  irreversibly; BR-4 is the precedent for an event that looked whole being
  silently discarded at read time. Add a suite row with a newline, a leading
  "- " and an embedded "at:".
- **BR-41** [Minor] `second-implementation-drifts` crlf.go's consumed() ignores the `out` it is handed and re-derives the translation from a wrong seed
  consumed(p, out, n) at crlf.go:41 never reads `out` — a dead parameter —
  and restarts the translation with `lastWasCR := false` instead of the
  writer's carried entry state. Write "a\r" then "\nb" with a 1-byte short
  write and it reports 0 consumed where 1 byte was written, so a retry
  duplicates the newline: the exact defect the fix's own comment says it
  prevents, surviving in the one case lastWasCR exists for.
  TestCRLFWriterReportsProgressOnAShortWrite covers only the fresh-state
  cell. Derive from `out`, or capture the entry flag before the loop.
- **BR-42** [Minor] `policy-in-io-shell` newestFirst returns oldest-first, sits in the IO shell, and is only reachable through a store-backed test
  This is the 2nd finding in family policy-in-io-shell; BR-22's defect itself
  is genuinely gone. What remains is the shape BR-22 named: a pure ordering
  policy living in ask.go rather than beside recentTurns in askctx.go, whose
  only test (TestTheDeckSectionCarriesTheNewestWords) needs a real YAML store
  and a wire-level fake to exercise a slice reversal. The name also says
  newest-first while the function returns oldest-first.
- **BR-43** [Minor] `test-asserts-nothing` The pty conformance suite runs whatever bin/define is on disk, with no staleness check
  startDefine (pty_conformance_test.go:63) skips when ../../bin/define is
  absent but never checks it is current. The binary here is timestamped
  22:45, 34 minutes before the round-5 fix commit at 23:19, so a `go test
  -tags conformance` run right now would validate pre-fix code and report ok
  — a live conformance check that cannot fail on the change it exists to
  check. Compare against the newest source mtime, or build in TestMain.

## Round 7 — 2026-08-24T14:08:16-07:00 (claude) — BLOCKED

### Disposed

- BR-24 — addressed — Reverting the warn to a discarded error reddens TestAnUnreadableUserModelIsReported; failingStore.UserModel now has a live call site.
- BR-30 — addressed — Deleting repl's signal watcher now reddens TestBothInterruptTransportsReachTheSink and TestThePipedLoopStillExitsOnASignal (verified); the subtest still bridges sigs itself, recorded as a plan revision rather than re-raised.
- BR-38 — addressed — TestTheAskWiringTable is the table, not a fourth one-off; all three named mutations plus the piped session cell redden against the full suite (verified).
- BR-39 — addressed — All four measured claims fixed and the two behavioural gaps pinned by enumeration tests; the surviving stale restatements are elsewhere in atlas and raised as the 9th in the family.
- BR-40 — addressed — The hostile-question suite row reddens when AppendEvent is replaced with a naive hand-written record (verified), so it is not vacuous.
- BR-41 — addressed — The out parameter is gone and entryWasCR is captured before the loop; re-seeding from false reddens TestCRLFWriterProgressAcrossACarriedCR (verified).
- BR-42 — addressed — recentDeck is pure, lives beside recentTurns in askctx.go and is table-tested without a store; un-reversing it reddens three rows (verified).
- BR-43 — addressed — builtBinary compiles from the tree per run; bin/define survives only in a comment. Staleness is impossible by construction.

### Raised

- **BR-44** [Important] `doc-overstates-code` Three claims in atlas's store section were falsified by this window, in the paragraphs that own them
  This is the 9th finding in this family; the rule has now been stated four
  times, so do NOT patch the three lines. Measured: (1) atlas/define.md:210-213,
  the normative artifact block, lists words/ and events/ while the code
  (store/yaml.go:37) and README both say user-model.md is the third — the atlas
  names it only in M2 prose at :527, so 2 of 3 restatements were swept;
  (2) atlas/define.md:252 "a whole record ... carries every field", contradicted
  by complete()'s generalisation to "has a SUBJECT" (store/event.go:45) that
  this window shipped precisely so a wordless asked record is whole — a reader
  following the atlas discards exactly the events it was generalised to keep;
  (3) atlas/define.md:576 "the fake behaves like the real thing is a test rather
  than an assumption", false for the newest interface method (see the
  test-asserts-nothing finding). The mechanical fix: a normative block in atlas
  is a CONSUMER of the code, so a change to what the code enumerates sweeps
  every block that enumerates it in the same commit. The enumeration is three
  blocks in the store section, resolved against yaml.go, event.go and
  storetest/suite.go mechanically, the way PQ-5 resolved every line citation.
- **BR-45** [Important] `test-asserts-nothing` Two pieces of scaffolding added this round read as protection and cannot fail
  This is the 7th finding in this family, and both instances are the shape
  BR-24's own disposal note named — "the disposal is the test". Measured, 2 of
  the 14 test helpers and fake fields this window added: (1) capture_test.go:85
  countingCapturer.asked and askedWord are appended at one site and read at
  ZERO, while their comment claims "so a test can assert that a question was
  NOT counted as one"; no test reads either field (the distinction is asserted
  at the store level instead, so this is dead scaffolding). (2)
  storetest/suite.go:38 "UserModel is empty before anything writes one" runs
  against both implementations, but store/mem.go:20 userModel has no setter and
  no writer anywhere in the tree, so for Mem the row asserts the only value the
  type can produce — unfalsifiable, and ARCH-MOCK: the fake cannot hold the
  state the real one holds for the method Store gained in this window. The
  rule: a fixture, field or suite row added to defend a finding must have a
  read site that can fail in the same commit, and a fake added to a conformance
  suite must be able to hold the state the row asserts about.
- **BR-46** [Important] `second-implementation-drifts` The scoped-ask wiring is now written twice, and only one copy carries the ordering rationale
  This is the 2nd finding in family second-implementation-drifts, so state the
  rule rather than fixing one site. repl.go:273-279 and replraw.go:148-168 each
  contain the identical five-step sequence — derive qctx, interrupts.Set,
  call ask, restore, qcancel — and the ordering is load-bearing in a way that
  is silent when wrong: only replraw.go:163-166 records why restore precedes
  qcancel. The second copy was created by round 2's BR-39 fix, while
  replraw.go:140-143 and atlas/define.md:568 both still say a second copy of
  this wiring would be a second copy of Ctrl-C's meaning (ARCH-DRY). The rule,
  same shape as fail(code) and nothingSays: one function owns the scope and
  both loops call it — askScoped(ctx, interrupts, run) — with the writers and
  the post-answer redraw staying in each loop's closure, since those are what
  legitimately differ.
- **BR-47** [Important] `plan-contract-drift` The Core concepts tables do not describe the entities this boundary's rounds created
  This is the 3rd finding in this family; BR-34 fixed two rows by eye and the
  next round created three more discrepancies, so fix the enumeration rather
  than the rows. Measured against workshop/plans/000016-console-qa-plan.md:
  (1) recentDeck (askctx.go:86) is a NEW pure entity with its own table test,
  extracted by round 1 in answer to a Critical, and is absent from the
  Pure-entities table entirely; (2) Integration points names "askCapturer (on
  Capturer)" where the code's method is CaptureAsk (capture.go:49) — the
  Revisions entry says so, the table does not; (3) the session bullet lists
  current, entry, turns while the code also has words (session.go:20), the
  field an issue Done-when row quantifies over. 3 of 3 entity changes made by
  the two boundary rounds are unrecorded or wrong. The rule, mechanised: at a
  boundary close the enumeration is
  git diff base..HEAD -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ',
  resolved against the Core concepts tables — a row fixed because a finding
  named it is the instance again.
- **BR-48** [Minor] `plan-bookkeeping` Boundary round 2 has no Revisions entry, including the one the previous round asked for
  This is the 4th finding in family plan-bookkeeping. The round-1 entry states
  the rule ("a Revisions entry is owed for every fork taken differently from
  the plan, in the milestone-close commit") and round 2 then took four forks
  and recorded none: newestFirst became recentDeck and moved to askctx.go,
  TestThePipedLoopsAskWiring was folded into TestTheAskWiringTable, replLines
  gained the interrupt scope the plan gave only to the raw loop, and the pty
  suite now builds its own binary. Round 6's explicit recommendation — Task
  11's test decomposition, and that "SIGINT through repl's watcher DURING a
  scoped stream" is still unasserted — is also absent. Checkboxes are 60 of 60,
  so this is the entry half of the rule only.
- **BR-49** [Minor] `nil-seam-policy` The interrupter seam has three consumers and two nil policies; one of them dereferences
  runEditor (replraw.go:56) and replLines (repl.go:226) substitute
  &interrupter{} when handed nil, while replRaw (replraw.go:15) passes it
  straight to readKeys, which dereferences it on the first Ctrl-C. Not
  production-reachable — repl is replRaw's only caller and always supplies one
  — but 44 test call sites now pass nil, exercising a configuration production
  never has, and the two policies will diverge again when #17 consumes the
  seam. Either the seam is required (tests pass &interrupter{}) or optional
  (one policy applied at every entry that consumes it).
- **BR-50** [Minor] `dead-branch` Four small residues: a stale prediction, an unreachable guard, a hanging test arm, and a per-question full deck read
  This is the 2nd finding in family dead-branch, so these are recorded rather
  than individually fixed. (1) replraw.go:132 "M2's streaming adds a fifth"
  predicts a lostTerminal call site that M2 deliberately did not create; two
  live sites remain — belongs to the atlas sweep above. (2) capture.go:97's
  decideCapture guard in CaptureAsk is deletable with the suite green and is
  unreachable in production, since -raw refuses in ask() before runAsk and
  noCapture yields a noopCapturer; correct as a mirror of Capture's guard, but
  not a tested policy. (3) repl_test.go:174 uses t.Context().Done() as its
  failure arm, so a loop that never returns HANGS to the package timeout
  instead of failing — measured at 600s during a mutation; its sibling
  assertSignalEndsTheLoop uses a 5s time.After. (4) gatherAskContext calls
  Deck() on every question, reading every file under words/ to pick twelve —
  negligible beside a network round-trip today, worth remembering when #10
  reuses askContext.

## Round 8 — 2026-08-24T14:30:32-07:00 (claude) — passed

### Disposed

- BR-44 — addressed — All three atlas claims swept at the paragraphs that own them; the conformance sentence is now true for the newest method because SetUserModel landed.
- BR-45 — addressed — Both verified by reversion — a no-op Mem.SetUserModel reddens the suite row, and mutating the captured word reddens countingCapturer.askedWord.
- BR-46 — addressed — One askScoped, called by both loops; removing the scope reddens 7 cells across both. The residual — 3 of its 4 mutations undefended — is raised separately.
- BR-47 — not-addressed — The three named rows are fixed, but the enumeration the finding demanded was run in the commit that stated it and missed Store.SetUserModel, which that same commit created.
- BR-48 — not-addressed — Still no Revisions entry for boundary round 2's four forks or round 6's Task 11 item; and Task 12's `sdlc close` checkbox is ticked while the issue is status:working at a milestone-close gate.
- BR-49 — not-addressed — Unchanged — replRaw (replraw.go:15) still passes the seam straight to readKeys with no policy, and 44 test call sites still pass nil.
- BR-50 — not-addressed — All four residues present verbatim — replraw.go:132's "adds a fifth", capture.go:96's guard, repl_test.go:199's hanging arm, ask.go:199's per-question Deck().

### Raised

- **BR-51** [Important] `test-asserts-nothing` askScoped's sequence has four mutations and one is defended; omitting restore makes the session unquittable, suite green
  This is the 8th finding in this family, so the rule rather than the site.
  cmd/define/ask.go:40-45. The commit that created askScoped probed ONE
  mutation (reversing the two defers), found it unobservable, and concluded
  "a rationale no test can defend is scaffolding". Measured, all four:
  omitting interrupts.Set reddens 5 tests across both loops; omitting
  `defer restore()` leaves the FULL suite green; omitting `defer qcancel()`
  leaves it green AND go vet silent, because qcancel is used as a value so
  lostcancel never fires; reordering is genuinely unobservable. The restore
  cell is user-visible: interrupter.scoped stays true and fn stays the dead
  question cancel, so readKeys (rawterm.go:70) swallows every subsequent
  \x03 and Ctrl-C at the prompt does nothing after the first question. I
  wrote the test — drive runEditor through the real readKeys, ask ?why, let
  the answer COMPLETE, send \x03, require the loop to return — and it fails
  mutated and passes unmutated in 0.67s. Every existing test asserts the
  sink DURING an answer; none asserts it AFTER one. The rule: when a probe
  finds a mutation untestable, the deliverable is the ENUMERATION of
  mutations to that mechanism, not the verdict on the one probed.
- **BR-52** [Important] `doc-overstates-code` An interrupted answer is dropped from the transcript, so the follow-up README promises resolves against nothing
  This is the 10th finding in this family; the rule has been stated five
  times, so do NOT weaken the sentence. cmd/define/ask.go:140-145 returns 0
  on the cancel path BEFORE sess.recordExchange, so an answer the user read
  and then stopped leaves no trace — not the partial answer, not even the
  question. README:79-88 states both halves in adjacent paragraphs:
  "Ctrl-C stops the answer rather than the session" and "the earlier
  questions in this session — so a follow-up like `give me two more
  examples` resolves against the answer before it". Measured against the
  wire fake: after 32 bytes streamed and cancelled, sess.turns is empty and
  the follow-up prompt contains only "## The word on screen / sycophantic"
  and the new question. Across the cells the claim quantifies over —
  answered, ErrTruncated, ErrUnavailable-with-partial, cancelled — it is
  true in 3 of 4, and the false cell is the flow the milestone is named
  after. The fix is to record the partial exchange (the user READ it, the
  same reason ask.go:153 keeps a truncated one) and give the claim a
  named-enumeration row test, the way TestRawNeverAsks does its six.
- **BR-53** [Minor] `second-implementation-drifts` crlfWriter.Write advances lastWasCR over bytes the underlying writer never took
  This is the 3rd finding in this family, so recorded rather than fixed at
  the site. crlf.go:19-42: entryWasCR is captured for consumed()'s benefit,
  but c.lastWasCR is still advanced across the WHOLE buffer even when only
  part of it was written, so the state carried into a retry describes bytes
  that never reached the terminal. Verified: Write("\r\nz") short at 1 byte
  returns n=1 correctly (BR-41's fix), and the retry with p[1:] = "\nz" then
  inserts a carriage return already on the wire — measured "\r\r\nz", the
  exact doubling lastWasCR exists to prevent and the one consumed()'s
  comment claims it prevents. BR-41 corrected the return value and left the
  carried state. Low reach: nothing in the tree retries and runAsk discards
  fmt.Fprint's error, so the symptom is one stray \r garbling a line on an
  already-degraded path. The rule: a translation materialised once must not
  be re-derived — Write and consumed are still two derivations of it.

## Round 9 — 2026-08-24T20:42:05-07:00 (claude) — BLOCKED

### Disposed

- BR-12 — not-addressed — Half (b) is now pinned; half (a)'s wiring still reverts green — submitLine's hist.Add(cmd.recallLine()) -> hist.Add(line) leaves the full suite green on two runs.
- BR-17 — not-addressed — Unchanged verbatim — Task 3's snippet at line 632, askUnavailable at 686/745, and zero mentions of mayAsk, recallLine, nothingSays or lostTerminal.
- BR-19 — addressed — All four cells verified red individually — the doubled backslash, nothingSays's note branch, the piped fail(2), and truncateQuestion's elision.
- BR-20 — addressed — replayInPlace routes through nothingSays; one production occurrence of the replay sentence in the whole tree, and the parameter is inSession where true is correct.
- BR-21 — addressed — One "lost the terminal" site in cmd/, via lostTerminal. The doc comment above it still predicts a fifth call site — that belongs to BR-50.
- BR-47 — not-addressed — Third round, same failure mode — the enumeration was re-run and again missed what the same commit created: Store.SetUserModel and carriedCR are both absent from the plan.
- BR-48 — not-addressed — Still no Revisions entry for round 2's four forks, round 6's Task 11 item, round 3's askScoped/SetUserModel, or the close round.
- BR-49 — not-addressed — Unchanged — replRaw still passes the seam straight to readKeys with no policy, and 44 test call sites still pass nil.
- BR-50 — not-addressed — All four residues present verbatim — replraw.go:132's "adds a fifth", capture.go's CaptureAsk guard, repl_test.go's t.Context() failure arm, ask.go's per-question Deck().
- BR-51 — addressed — The restore cell is genuinely pinned — omitting defer restore() reddens TestCtrlCQuitsAgainOnceTheAnswerIsOver (verified). Two residuals raised separately: that test is racy, and the enumeration's qcancel row is false.
- BR-52 — addressed — Dropping the cancel-path recordExchange reddens TestAnAnswerTheUserReadSurvivesHowItEnded/stopped_by_the_user (verified).
- BR-53 — not-addressed — The site defect is fixed and behaviourally verified, but no test fails without carriedCR, and the rule the finding asked for is unapplied — Write and consumed are still two derivations.

### Raised

- **BR-54** [Important] `unsynchronised-test-observation` TestCtrlCQuitsAgainOnceTheAnswerIsOver fails 12 of 30 runs on unmutated HEAD
  This is the 2nd finding in family unsynchronised-test-observation. BR-25
  fixed the instance (data races on a bytes.Buffer); state the rule instead.
  Rule - a test must synchronise on the state it ASSERTS, not on a proxy that
  merely precedes it. askrun_test.go:831 waits for "insincerely", the LAST
  text delta of stream-sample.sse (line 35 of 45), then immediately writes
  \x03 and requires the scope to already be restored. Everything between is
  unsynchronised: the remaining SSE frames parse, Stream returns,
  recordExchange runs, and runAsk's deferred CaptureAsk does a real
  AppendEvent to DISK (ask.go:147-151) - all inside askScoped, before
  restore(). Lose that race and readKeys swallows the \x03 as
  scope-consumed, quit never closes, and the test fails at its 5s deadline
  printing the message it reserves for the real defect. Measured on
  unmutated HEAD: `-count=30` isolated gives 12 failures; under full-package
  load it is rarer (3 of 3 clean in a dedicated sweep, one spontaneous
  failure across ~10 full-package runs during this review). It is the only
  test in the file that drives a COMPLETING stream and then asserts
  post-scope state - the eight `Stall: true` tests never leave the scope - and
  it is the sole defender of the cell BR-51 named, while the close's
  --verified evidence is a `go test ./...` run. Fix: wait on a happens-after
  marker for restore() rather than on answer text; askInSession
  (replraw.go:164-165) writes "\r\n" then draw() strictly after askScoped
  returns. Consider separately moving the CaptureAsk disk write outside the
  scope, which also shrinks the window where a user's real Ctrl-C is
  silently swallowed.
- **BR-55** [Important] `doc-overstates-code` Four measured doc claims contradict the code, all created by the last two rounds
  This is the 11th finding in this family; the rule has been stated five
  times, so do NOT patch the four lines. Rule, in the shape this window
  needs it - a measured claim written into a comment is a claim, and it is
  falsified the same way any other absolute is: by running the mutation it
  names. Measured: (1) ask.go:43 records
  `omit defer qcancel()  1 test red - the question's context leaks`;
  measured, `go test ./cmd/define/` is ok 58.078s with 0 tests red and
  `go vet ./cmd/define/` silent, because qcancel is passed as a value to
  interrupts.Set so lostcancel never fires - which is exactly what BR-51's
  own detail reported, so the commit answering it wrote the opposite into
  the table whose thesis is "the deliverable is the enumeration"; the cell
  is a real per-question context leak and is undefended. (2) ask.go:38 says
  `omit interrupts.Set  10 tests red` while askrun_test.go:800, same commit,
  says "reddens five tests"; I measure 4 top-level / 10 with subtests - two
  numbers in one commit is the signature of a recollection, not a
  measurement. (3) atlas/define.md:602-603 still says the raw loop's closure
  "is where M2's streaming writer and scoped interrupt hang", which round 3
  falsified by moving the scope into askScoped - replraw.go:142-143 says so
  explicitly, so the code comment was corrected and the atlas paragraph that
  OWNS the claim was left standing, BR-28/BR-44's rule a third time.
  (4) store/mem.go:18 says "SetUserModel is #17's to add ... until then"
  while mem.go:91, same commit, implements it and the suite writes through
  it. A fifth, adjacent: a CONFIGURED but unreachable model prints
  "no model configured" and exits 1 while the question IS recorded as an
  asked event (probed against http://127.0.0.1:1), where README:97-101 keys
  "not recorded" on "no model configured" - and it is the one outcome cell
  TestAQuestionIsRecordedWhateverBecameOfTheAnswer does not enumerate.

## Open findings

- **BR-12** [Minor] `forced-route-enumeration` the "\" hatch is dropped from editor recall while "?" is kept, and the no-model message calls a headword "not a word"
- **BR-17** [Minor] `plan-bookkeeping` the plan still describes an M1 the code no longer implements, despite round 2 recommending the Revisions entry
- **BR-47** [Important] `plan-contract-drift` The Core concepts tables do not describe the entities this boundary's rounds created
- **BR-48** [Minor] `plan-bookkeeping` Boundary round 2 has no Revisions entry, including the one the previous round asked for
- **BR-49** [Minor] `nil-seam-policy` The interrupter seam has three consumers and two nil policies; one of them dereferences
- **BR-50** [Minor] `dead-branch` Four small residues: a stale prediction, an unreachable guard, a hanging test arm, and a per-question full deck read
- **BR-53** [Minor] `second-implementation-drifts` crlfWriter.Write advances lastWasCR over bytes the underlying writer never took
- **BR-54** [Important] `unsynchronised-test-observation` TestCtrlCQuitsAgainOnceTheAnswerIsOver fails 12 of 30 runs on unmutated HEAD
- **BR-55** [Important] `doc-overstates-code` Four measured doc claims contradict the code, all created by the last two rounds
