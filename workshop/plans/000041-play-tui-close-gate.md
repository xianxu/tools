---
gate: boundary-review
issue: 41
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-31T16:09:53-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Important
          title: D6 routes paging through toInput, which would teach the pure play package about a viewport
          detail: |-
            toInput returns play.Input (play_loop.go:216). The editor deliberately dispatches viewport
            gestures in the loop BEFORE Apply, documented at replraw.go:361-377 ("a viewport gesture never
            reaches Apply"). Handle them ahead of toInput and extract the four-case switch plus wheelLines
            into one helper both loops call, rather than a second copy of the policy. Also note D6's
            "nothing else moves" omits enterMouse, whose cost is that drag-select needs Option/Shift
            (rawterm.go:175-190, /help at command.go:267) — a review sitting now inherits that.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: viewport-gesture-layering
          round: 1
        - id: BR-2
          severity: Minor
          title: Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
          detail: |-
            The Done-when preamble requires every pin be a named test or a grep for a property, never
            "file X is unchanged", and row 7 is the one row that breaks its own rule. Grepping the
            message at play_loop.go:246 finds no test asserting the degraded path. T6 changes
            todaysQuestions' signature so that path must now hand the loop a usable empty progress
            rather than merely continuing locally, which is precisely when an uncovered branch is worth
            a test.
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: pin-without-predicate
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-31T16:09:53-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: --play takes the alternate screen without consulting opt.tty, so -no-color no longer disables cursor control
          detail: |-
            runPlay gates the full-screen surface on stdinIsTerminal alone, where repl uses
            `terminalUI := interactive && opt.tty` (repl.go:228) and falls back to replLines.
            playConsole then calls enterAlt/enterMouse and paints ANSI frames unconditionally,
            so `define --play -no-color` emits the escapes that flag exists to suppress
            (main.go:527) and `define --play > file` writes them into the file at a fabricated
            80 columns. Before this window --play emitted no escapes at all.
          family: terminal-ui-gate
          round: 2
        - id: BR-4
          severity: Important
          title: Narrowing the window mid-sitting clips option glosses — the same class as the operator's finding, swept only at the startup width
          detail: |-
            Measured: a form-2.3 prompt built by choiceFor at width 80 and painted at cols 40
            loses the tail of every option line. The resize case (play_loop.go:196-213)
            deliberately does not re-derive opt.width, which was true before the last commit
            made the wrap width a per-question fact. Same rule as the revision that closed the
            startup instance; this is the enumerable sibling (ARCH-PURPOSE).
          family: frame-clips-unwrapped-text
          round: 2
        - id: BR-5
          severity: Important
          title: TestDroppingAWordLowersTheCostTheBarShows passes only because slices.DeleteFunc aliases the caller's backing array
          detail: |-
            sittingDeck is passed to playSession by value and the test asserts on its own copy;
            it observes the drop solely through in-place compaction of the shared array.
            Rewriting dropped() to build a new slice — behaviour-identical for the loop — turns
            the test red. It also never inspects a drawn footer, so it does not pin the claim
            the finding is about.
          family: test-asserts-aliasing
          round: 2
        - id: BR-6
          severity: Important
          title: crlfWriter has no production caller after this window, and the diff adds three claims that it does
          detail: |-
            grep finds crlfWriter only in crlf.go and crlf_test.go. The new atlas prose says it
            "survives only on the piped path" (atlas/define.md:285-290 and :893-899), which is
            false; replraw.go:307-308 still says "--play keeps its own crlfWriter", the opposite
            of what D5a landed; askhighlight_test.go:245 repeats it. Third recurrence of this
            family per play_loop.go:459-462.
          family: doc-sweep-incomplete
          round: 2
        - id: BR-7
          severity: Important
          title: playConsole duplicates replRaw's console construction verbatim except for the screen constructor
          detail: |-
            play_loop.go:73-104 and replraw.go:32-70 differ in one token (newPinnedScreen vs
            newLiveScreen) across six statements. D1 committed to widening the shared seam
            rather than growing a parallel one, and the function's own comment asserts "Not a
            parallel construction". #40 will be the third caller (ARCH-DRY).
          family: parallel-construction
          round: 2
        - id: BR-8
          severity: Minor
          title: refresh() hand-copies two fields out of figures(), so a future field goes stale silently
          detail: |-
            play_loop.go:130-141 assigns only fig.load and fig.fresh from a fresh figures() call.
            Assigning the struct and re-applying total/done cannot rot.
          family: partial-copy-refresh
          round: 2
        - id: BR-9
          severity: Minor
          title: Done-when 7 (failed log read degrades to empty progress) has no test, contradicting the plan's own rule
          detail: |-
            Its pin is "the existing behaviour at play_loop.go:246, unchanged", which the
            Done-when header forbids. The behaviour also changed: the degraded progress now
            drives the bar and the summary, which previously re-read the log.
          family: unpinned-done-when
          round: 2
        - id: BR-10
          severity: Minor
          title: twiceNumberedOption ranges a map, so a tie picks the "guaranteed miss" at random
          detail: |-
            pty_conformance_test.go:905-915. If two digits both appear twice in the frame, the
            returned answer depends on map iteration order and the deliberate miss can become a
            correct answer. Pick the lowest matching digit or fail on a tie.
          family: nondeterministic-test-read
          round: 2
        - id: BR-11
          severity: Minor
          title: choiceFor wraps to opt.width (0 on a non-terminal or below 20 cols) while the frame clips at terminalCols (never 0)
          detail: |-
            The wrap width is a policy answer with a sentinel; the clip width is a measurement.
            Where they disagree the gloss arrives unwrapped into a frame that cuts it —
            TestUnwrappedWidthLeavesTheGlossAlone pins the sentinel, which is correct for the
            pipe and wrong for a sitting.
          family: frame-clips-unwrapped-text
          round: 2
        - id: BR-12
          severity: Minor
          title: --play -raw records nothing but the bar still applies the transition
          detail: |-
            decideCapture returns captureNothing under opt.raw (capture.go:30), so
            CaptureReview writes no event while held.answered still advances the in-memory
            progress. On that path D7's "cannot drift from what the next sitting derives" does
            not hold.
          family: figures-drift
          round: 2
        - id: BR-13
          severity: Minor
          title: TestTheBarCountsAnswersAsTheyLand slices on strings.Index without checking for -1
          detail: |-
            play_loop_test.go:558 — a bar format change panics the test instead of failing it
            with the message the assertion was written to give.
          family: test-fragility
          round: 2
        - id: BR-14
          severity: Minor
          title: sittingDeck has pointer-receiver mutators but is passed by value, sharing its map and half-sharing its slice
          detail: |-
            Nothing production depends on the aliasing today, but it is what makes I-3's test
            pass and it is a landmine for #40's second consumer. A *sittingDeck parameter
            removes the question.
          family: value-receiver-shared-state
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-31T16:42:01-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: viewportGesture exists (replraw.go:109) but has ONE caller (replraw.go:413); play_loop.go:235-248 still holds the verbatim four-case switch. The enterMouse-cost half IS addressed (replraw.go:63-66).
          round: 3
        - id: BR-2
          disposition: addressed
          note: TestAFailedLogReadStillRunsTheSitting drives the degraded path through a real bar draw via logRefusingStore.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Both gates land before enterRaw; deleting them reddens TestPlayRefusesWhenStdoutIsNotATerminal. The -no-color half is pinned only by a pty row that skips here.
          round: 3
        - id: BR-4
          disposition: addressed
          note: 'Verified by mutation: removing the resize''s opt.width update reddens TestANarrowedSittingWrapsTheRestOfItself. Option lines only; see the new definition-body finding.'
          round: 3
        - id: BR-5
          disposition: addressed
          note: 'Mutation-verified both ways: an alias-free dropped() stays green, a no-op dropped() goes red on the drawn bar.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: crlf.go and crlf_test.go deleted, shortWriter re-homed. Residual mentions of the deleted FILE names roll into the new doc-sweep finding.
          round: 3
        - id: BR-7
          disposition: addressed
          note: newConsole(ctx, d, sess, stdout, newScreen) is called from replraw.go:33 and play_loop.go:96.
          round: 3
        - id: BR-8
          disposition: addressed
          note: refresh() assigns the whole figures() struct then re-applies total and done (play_loop.go:122-132).
          round: 3
        - id: BR-9
          disposition: addressed
          note: Same fix as BR-2.
          round: 3
        - id: BR-10
          disposition: addressed
          note: twiceNumberedOption now scans '1'..'9' in order rather than ranging the map (pty_conformance_test.go:944-949).
          round: 3
        - id: BR-11
          disposition: addressed
          note: choiceFor no longer wraps; the write-time wrap uses the live opt.width, and the non-terminal case is now refused outright. The sub-20-column sentinel remains as an explicit policy — see plan revision 3.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: decideCapture still returns captureNothing under opt.raw while held.answered advances the in-memory progress. Minor; never blocks.
          round: 3
        - id: BR-13
          disposition: addressed
          note: play_loop_test.go:558 now Fatalf's on a missing separator instead of slicing.
          round: 3
        - id: BR-14
          disposition: addressed
          note: sittingDeck travels by pointer from todaysQuestions through playSession; confirmed by the alias-free mutation staying green.
          round: 3
      findings:
        - id: BR-15
          severity: Important
          title: A reveal written after a narrowing resize carries a definition wrapped to the STARTUP width, and the frame clips it
          detail: |-
            This is the 3rd finding in family `frame-clips-unwrapped-text`. Earlier rounds fixed
            instances (the operator's startup-width option gloss, then BR-4's resize sibling). Do NOT
            fix this instance alone.

            THE RULE that covers all of them: `--play` RENDERS its text at queue-build time and WRITES
            it much later, so every pre-rendered artifact carries a width that may already be wrong when
            the frame clips it. Anything the loop writes into the buffer must be wrapped to the width in
            force at the moment of WRITING, not at the moment of rendering. `wrapOptionLines` applies
            that rule to exactly one line-kind; `todaysQuestions:436` calls `Render(..., Width: opt.width)`
            once per sitting for every definition, and `play_loop.go:323` writes those lines after any
            number of resizes.

            MEASURED prevalence (scratch probe, `playRig` with sycophantic/ephemeral/quokka/mesa,
            opt.width=100, resize 100->40, then one reveal): 7 buffer lines written AFTER the narrow
            exceed 40 columns, worst 88 cells. README.md:148 now claims "a definition longer than the
            window is scrolled rather than lost", which this contradicts.

            The enumeration the rule implies, to be swept in ONE round: q.Prompt(), asked.Reveal()
            (option lines AND the rendered definition body), the "removed %q from the deck" notice,
            finish()'s summary lines, and con.stderr diagnostics. Two candidate mechanisms: render the
            entry lazily at write time against the live opt.width, or keep the unwrapped source on the
            Question so a re-wrap is possible.
          family: frame-clips-unwrapped-text
          round: 3
        - id: BR-16
          severity: Important
          title: Five current-truth artifacts name symbols the tree does not have, and the new refusal surface reaches neither README nor atlas
          detail: |-
            This is the 2nd finding in family `doc-sweep-incomplete`. Do NOT fix the instances one by
            one — that is what failed in rounds 1 and 2.

            THE RULE, and it is mechanisable because the repo already built most of it: every name a
            current-truth artifact cites must be DECLARED in the tree. `TestARemovedDeclarationIsSweptOrRetired`
            (repo_guard_test.go) only sees `-func` lines whose names pass `isCitableName` (exported or
            Test*), so it structurally cannot see a removed TYPE (`crlfWriter`), a removed unexported
            func (`playConsole`, `draw`, `fitMenu`), a deleted FILE path (`crlf_test.go`), or a name that
            was never declared at all. Widening it to (a) removed type/const/unexported declarations,
            (b) deleted file paths, and (c) a forward check that every `Test[A-Z]\w+` cited in a
            current-truth artifact is declared, turns this whole family into a build failure.

            MEASURED prevalence at HEAD, all introduced by this window:
              - cmd/define/play_loop.go:278 cites `TestAMissIsRecordedBeforeItIsRevealed`; the tree
                declares `TestAMissRecordsBeforeItPlays`. Introduced by 4d3b53a; never existed.
              - cmd/define/highlightwriter.go:60 says "crlf_test.go defends that" — file deleted here.
              - cmd/define/highlightwriter_test.go:158 cites "crlf_test.go's fixture", 400 lines above
                the fixture's new home in the same file.
              - atlas/define.md:2006 says "`playConsole` is `replRaw`'s construction"; the symbol was
                renamed to `newConsole` in the final commit.
              - workshop/plans/000038-play-clickable-plan.md:101, :128, :237 name `playConsole`.
              - Two prose breaks left by the sweep: atlas/define.md:942-943 ("It used to wrap the raw
                loop's / the raw loop's line-ending writer") and highlightwriter_test.go:142-144
                ("...right answer here while / A writer that can be written to again...").

            THE DOCS GATE, same rule at the behaviour level: this window added two user-facing refusals
            (`define --play > file` and `define --play -no-color` now print a message and exit 1,
            play_loop.go:67-74). Neither README.md nor atlas/define.md records them, and the atlas's
            "Degrading is the absence of input" paragraph (:470-476) still describes only the editor's
            degrade-by-routing. `newConsole`, `viewportGesture` and `wrapOptionLines` — three new shared
            surfaces `#40` will consume — appear in neither.
          family: doc-sweep-incomplete
          round: 3
        - id: BR-17
          severity: Important
          title: The plan's Core concepts table describes two entities the tree does not have, and the guard checks only the status column
          detail: |-
            `workshop/plans/000041-play-tui-plan.md:118` says `choiceFor` "takes the terminal width and
            wraps each gloss through `wrapText`" — BR-4 reverted exactly that, and `optionpool.go:228`
            declares `choiceFor(word, rendered string, e Entry, pool []play.Candidate, seed uint64)`.
            `:126` says `viewportGesture` is "PURE dispatch — the paging keys, for both loops"; it has
            one caller. The plan's BR-4 revision also asserts "all three are closed" when site (c),
            `opt.width == 0`, is a deliberate policy exception at cols < 20.

            `TestPlanTableStatusMatchesTheChangeWindow` passes both rows because it only judges the
            `modified`/`unchanged`/`new` column against `git diff` — the DESCRIPTION is unguarded. The
            plan's own rounds 2 and 3 stated the rule ("a Revision that reverses a decision re-reads the
            decision prose, the entity tables, AND the Done-when rows"); this is the fourth time a
            reversal reached the prose and not the table. `#40` reads this table as the record of what
            landed.
          family: plan-table-vs-tree
          round: 3
        - id: BR-18
          severity: Minor
          title: The surface gate runs after todaysQuestions has already read the deck and written to the non-terminal stdout
          detail: |-
            This is the 2nd finding in family `terminal-ui-gate`. THE RULE: every precondition for
            owning the terminal is settled in ONE place, before the command does any work or writes any
            byte — the same rule main.go:542 states for usage errors ("settled BEFORE a store is
            opened"). `runPlay` calls `todaysQuestions` at :33 and only checks `isTerminal(stdout)` and
            `opt.tty` at :67-74, so `define --play > file` on an empty deck writes "the deck is empty"
            into the file and exits 0 without ever reaching the refusal, and on a non-empty deck it pays
            the deck+log reads first. Moving the two gates above the `todaysQuestions` call closes both.
          family: terminal-ui-gate
          round: 3
        - id: BR-19
          severity: Minor
          title: workshop/lessons.md is untouched across a window that ran two review rounds and 14 findings
          detail: |-
            AGENTS.md section 4: "When you run code review, add rules to workshop/lessons.md that
            prevent the mistakes you found." Three families repeated across rounds
            (`frame-clips-unwrapped-text`, `doc-sweep-incomplete`, `parallel-construction`) and the
            rules the plan wrote for itself live only in that plan's Revisions, which is archived at
            close. The two durable ones — "adopting an existing seam inherits its behaviour on inputs
            the previous consumer never sent it" and "re-examine the tests that assert over the surface
            this issue changes MEANS enumerating them" — belong in lessons.md, where the next issue
            reads them.
          family: lessons-not-recorded
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-31T17:13:53-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: viewportGesture called from replraw.go:413 and play_loop.go:247; stubbing the play call reddens TestPagingIsNotAnAnswer and TestALongRevealPagesRatherThanScrollingTheWordAway. The enterMouse cost is in the source comment only — folded into BR-16's docs gate.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: 'Unchanged at HEAD: decideCapture still returns captureNothing under opt.raw (capture.go:30, :135) while held.answered advances the in-memory progress; no --play/-raw conflict guard exists in main.go.'
          round: 4
        - id: BR-15
          disposition: addressed
          note: 'Mutation-verified: making wrapWritten a no-op reddens TestANarrowedSittingWrapsTheRestOfItself with 7 overwide lines (worst 97 cells), covering option lines, the rendered definition body and the summary. The helper-write gap is raised separately as the family''s 4th.'
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: Instances swept, but the closing commit added a new one and the mechanism was not built. See the finding detail below.
          round: 4
        - id: BR-17
          disposition: addressed
          note: 'All three named contradictions verified fixed against the tree: choiceFor is unchanged and its row says so, viewportGesture now has two callers, and site (c) is stated as a policy exception. The unguarded-column half is re-raised as the family''s 2nd with new evidence.'
          round: 4
        - id: BR-18
          disposition: not-addressed
          note: 'The gates did move above todaysQuestions (play_loop.go:52-72), but nothing pins it: moving them back below the call in a scratch worktree leaves the whole cmd/define suite green (107s, ok). The same round moved TestEmptyQueueExitsZero off runPlay onto todaysQuestions, deleting the only runPlay-on-empty-deck coverage.'
          round: 4
        - id: BR-19
          disposition: addressed
          note: 'lessons.md gains three rules (seam adoption, enumerate the class, a plan naming an anti-pattern is not protection), each with the #41 evidence.'
          round: 4
      findings:
        - id: BR-20
          severity: Important
          title: playAnnounced writes into the clipping frame without wrapWritten, so a playback diagnostic is cut mid-sitting
          detail: |-
            This is the 4th finding in family `frame-clips-unwrapped-text`. Do NOT fix this instance —
            the rule is already stated at playbar.go:66-87; what is missing is that it is enforced at
            call sites the LOOP owns, so any write made by a helper the loop calls escapes it.
            play_loop.go:354 calls playAnnounced, whose stderr is the screen; main.go:856 writes
            `define: %s` unwrapped, and main.go:844/:866 and reportVoice do the same.
            MEASURED (scratch probe: playRig, opt.width=40, newPinnedScreen(24,40), a player returning
            a network error, one miss then a reveal): one transcript line of 156 cells in a 40-column
            terminal, which Paint clips. atlas/define.md's new section already claims the loop routes
            "its diagnostics" through the one function; only the drop error at play_loop.go:312 is.
            Fix the class: wrap at the SEAM — the pinned screen's or the console's Write, against its
            own cols, with the sub-20-column policy in one place — so a helper cannot write around it,
            and extend the narrowing test to drive a failing playback so the predicate covers it.
          family: frame-clips-unwrapped-text
          round: 4
        - id: BR-21
          severity: Minor
          title: The initial sittingFigures hand-copies refresh()'s first two lines
          detail: |-
            This is the 2nd finding in family `partial-copy-refresh`. BR-8 fixed refresh() copying
            named fields out of figures(); play_loop.go:129-130 is now the site that does it, so a
            third adjustment added to refresh() would be stale on the first frame, before any answer.
            The rule: one expression builds the figures and every site derives from it. refresh() is
            exactly equivalent at init because s.Right+s.Wrong is 0 there, so calling it is the fix.
          family: partial-copy-refresh
          round: 4
        - id: BR-22
          severity: Minor
          title: Three pure entities are listed as Integration points and an IO constructor as Pure
          detail: |-
            This is the 2nd finding in family `plan-table-vs-tree`. Do NOT fix only these rows.
            workshop/plans/000041-play-tui-plan.md:125-127 lists viewportGesture (whose own text says
            "PURE dispatch"), wrapWritten (string to string, playbar.go) and livePrompt (play.Session
            to string) under "Integration points", whose column header is "Wraps"; none wraps an
            external dependency and none is injected. Conversely newPinnedScreen (:109), which takes a
            tty and builds the IO shell, sits under "Pure entities".
            THE RULE: a table row asserts path, status, and kind/description, and
            TestPlanTableStatusMatchesTheChangeWindow judges only status. Either mechanise what can be
            — every backticked Name is DECLARED at the stated path, and an Integration row names a
            wrapped external dependency — or stop asserting the unguarded columns in the table `#40`
            reads as the record of what landed.
          family: plan-table-vs-tree
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-08-31T17:41:30-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: not-addressed
          note: 'Nothing at HEAD touches it: no usage gate rejects `--play -raw`, decideCapture still returns captureNothing (capture.go:30), and held.answered/refresh still advance the bar.'
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: 'repo_guard_test.go is untouched in this window, so neither the removed-type/unexported/deleted-file widening nor the forward "cited Test name is declared" check exists; the class was filed to #33 while this round added two NEW instances — highlightwriter.go:60 cites TestHighlightWriterShortWriteContract (never existed; the real name is TestHighlightWriterTreatsAShortWriteAsAnError) and playbar.go:122-137 lost isOptionLine''s doc comment to the inserted minWrapWidth const — plus two survivors: plan:311 still cites TestAMissIsRecordedBeforeItIsRevealed, and atlas/define.md:2054 still describes the deleted per-site wrap and the deleted resize-time opt.width update.'
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: The code is right (gates at play_loop.go:56-77, above todaysQuestions at :79) but nothing pins it — I moved the three gates back below todaysQuestions in a scratch copy and `go test ./cmd/define/` stayed green (107s); TestPlayRefusesWhenStdoutIsNotATerminal only checks exit 1, absence of ESC, and a stderr substring, none of which a "the deck is empty" line to stdout would break.
          round: 5
        - id: BR-20
          disposition: addressed
          note: 'The wrap moved to liveScreen.Write for pinned screens (screen.go:511) and playAnnounced''s plain-text `define: %s` warning is now wrapped; pinned in both directions by TestThePinnedScreenWrapsWhateverIsWrittenToIt — but the same commit introduced a new hole in the seam, raised below.'
          round: 5
        - id: BR-21
          disposition: addressed
          note: play_loop.go:146 calls refresh() at init; behaviour-identical to the hand-copy, so accepted as a structural fix with no possible failing test.
          round: 5
        - id: BR-22
          disposition: addressed
          note: 'viewportGesture/wrapWritten/livePrompt moved to Pure entities and newPinnedScreen to Integration points; #33 gained the widening spec in its own Revisions section.'
          round: 5
      findings:
        - id: BR-23
          severity: Critical
          title: wrapWritten skips every escape-carrying line, and --play can only run with colour on, so a reveal's rendered definition is never wrapped
          detail: |-
            This is the 5th finding in family `frame-clips-unwrapped-text`, and it was INTRODUCED by the commit that closed the fourth. Do NOT fix this instance.
            playbar.go:105-109 returns any line containing 0x1b untouched. runPlay refuses unless isTerminal(stdout) and opt.tty, and main.go:523/527 define color and tty as the same expression, so every real sitting renders with Color:true and every styled line bypasses the seam. The same commit also deleted the resize case's opt.width maintenance that had been the prior mitigation.
            MEASURED (scratch: playRig + opt.color=true, opt.width=100, newPinnedScreen(24,40), one reveal): 4 buffer lines of 55-87 cells in a 40-column terminal, e.g. "    ephemerality \x1b[35m/..../\x1b[0m noun ephemerally ..." at 87 cells. Deleting the three-line skip takes that to 0 and leaves `go test ./cmd/define/...` green, so nothing defends it; wrapText already measures with visibleCells and strings.Fields cannot split inside an escape sequence.
            THE RULE, one step past "wrap at the seam": a seam covers every write only when (a) every path into the buffer goes through it and (b) its predicate is exercised in the configuration production actually runs in. The enumeration is two-by-two and belongs in the plan: paths = liveScreen.Write (wraps, except escapes) and liveScreen.WriteRegions (screen.go:551, calls l.s.Write directly and does NOT wrap — latent for --play, live for #40's board, and already contradicting screen.go:503's "Here nothing can write around it"); line classes = plain (wraps) and escape-carrying (does not).
            Fix the class: route both methods through one private writeBuffer(text) that applies the wrap when pinned, narrow the guard to the eraseLine gesture rather than "contains any escape", and drive TestANarrowedSittingWrapsTheRestOfItself with opt.color = true.
          family: frame-clips-unwrapped-text
          round: 5
        - id: BR-24
          severity: Important
          title: playRig hardcodes color:false, a configuration --play now refuses to run in, so every in-process sitting test drives an unreachable state
          detail: |-
            play_loop_test.go:39 returns options{color: false, width: 0, ...}. Since BR-3, runPlay exits 1 unless opt.tty, and opt.tty and opt.color are the identical expression (main.go:523/527) — so no in-process test has ever exercised the only configuration a sitting can be in. That is why the Critical above shipped: Done-when 0b's two pins (TestANarrowedSittingWrapsTheRestOfItself, TestALongOptionGlossWrapsRatherThanBeingCut) both pass with colour off. The pty rows do run coloured, but TestPTYPlayResizeRepaints narrows ROWS 24->10 and never COLS, so no row narrows the width of a coloured sitting.
            THE RULE: a rig's default options have to be reachable from the flag parse of the command under test. Set color/tty true in playRig (or derive both from one helper, as main.go does) and add a column-narrowing pty row.
          family: rig-config-not-production
          round: 5
      blocked: true
---

# Gate ledger — tools#41 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T16:09:53-07:00 (sdlc) — passed

### Raised

- **BR-1** [Important] `viewport-gesture-layering` D6 routes paging through toInput, which would teach the pure play package about a viewport
  toInput returns play.Input (play_loop.go:216). The editor deliberately dispatches viewport
  gestures in the loop BEFORE Apply, documented at replraw.go:361-377 ("a viewport gesture never
  reaches Apply"). Handle them ahead of toInput and extract the four-case switch plus wheelLines
  into one helper both loops call, rather than a second copy of the policy. Also note D6's
  "nothing else moves" omits enterMouse, whose cost is that drag-select needs Option/Shift
  (rawterm.go:175-190, /help at command.go:267) — a review sitting now inherits that.
  (carried from plan-quality PQ-3, deferred to the boundary review)
- **BR-2** [Minor] `pin-without-predicate` Done-when row 7 is pinned by "the existing behaviour, unchanged" and no test covers that behaviour today
  The Done-when preamble requires every pin be a named test or a grep for a property, never
  "file X is unchanged", and row 7 is the one row that breaks its own rule. Grepping the
  message at play_loop.go:246 finds no test asserting the degraded path. T6 changes
  todaysQuestions' signature so that path must now hand the loop a usable empty progress
  rather than merely continuing locally, which is precisely when an uncovered branch is worth
  a test.
  (carried from plan-quality PQ-10, deferred to the boundary review)

## Round 2 — 2026-08-31T16:09:53-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `terminal-ui-gate` --play takes the alternate screen without consulting opt.tty, so -no-color no longer disables cursor control
  runPlay gates the full-screen surface on stdinIsTerminal alone, where repl uses
  `terminalUI := interactive && opt.tty` (repl.go:228) and falls back to replLines.
  playConsole then calls enterAlt/enterMouse and paints ANSI frames unconditionally,
  so `define --play -no-color` emits the escapes that flag exists to suppress
  (main.go:527) and `define --play > file` writes them into the file at a fabricated
  80 columns. Before this window --play emitted no escapes at all.
- **BR-4** [Important] `frame-clips-unwrapped-text` Narrowing the window mid-sitting clips option glosses — the same class as the operator's finding, swept only at the startup width
  Measured: a form-2.3 prompt built by choiceFor at width 80 and painted at cols 40
  loses the tail of every option line. The resize case (play_loop.go:196-213)
  deliberately does not re-derive opt.width, which was true before the last commit
  made the wrap width a per-question fact. Same rule as the revision that closed the
  startup instance; this is the enumerable sibling (ARCH-PURPOSE).
- **BR-5** [Important] `test-asserts-aliasing` TestDroppingAWordLowersTheCostTheBarShows passes only because slices.DeleteFunc aliases the caller's backing array
  sittingDeck is passed to playSession by value and the test asserts on its own copy;
  it observes the drop solely through in-place compaction of the shared array.
  Rewriting dropped() to build a new slice — behaviour-identical for the loop — turns
  the test red. It also never inspects a drawn footer, so it does not pin the claim
  the finding is about.
- **BR-6** [Important] `doc-sweep-incomplete` crlfWriter has no production caller after this window, and the diff adds three claims that it does
  grep finds crlfWriter only in crlf.go and crlf_test.go. The new atlas prose says it
  "survives only on the piped path" (atlas/define.md:285-290 and :893-899), which is
  false; replraw.go:307-308 still says "--play keeps its own crlfWriter", the opposite
  of what D5a landed; askhighlight_test.go:245 repeats it. Third recurrence of this
  family per play_loop.go:459-462.
- **BR-7** [Important] `parallel-construction` playConsole duplicates replRaw's console construction verbatim except for the screen constructor
  play_loop.go:73-104 and replraw.go:32-70 differ in one token (newPinnedScreen vs
  newLiveScreen) across six statements. D1 committed to widening the shared seam
  rather than growing a parallel one, and the function's own comment asserts "Not a
  parallel construction". #40 will be the third caller (ARCH-DRY).
- **BR-8** [Minor] `partial-copy-refresh` refresh() hand-copies two fields out of figures(), so a future field goes stale silently
  play_loop.go:130-141 assigns only fig.load and fig.fresh from a fresh figures() call.
  Assigning the struct and re-applying total/done cannot rot.
- **BR-9** [Minor] `unpinned-done-when` Done-when 7 (failed log read degrades to empty progress) has no test, contradicting the plan's own rule
  Its pin is "the existing behaviour at play_loop.go:246, unchanged", which the
  Done-when header forbids. The behaviour also changed: the degraded progress now
  drives the bar and the summary, which previously re-read the log.
- **BR-10** [Minor] `nondeterministic-test-read` twiceNumberedOption ranges a map, so a tie picks the "guaranteed miss" at random
  pty_conformance_test.go:905-915. If two digits both appear twice in the frame, the
  returned answer depends on map iteration order and the deliberate miss can become a
  correct answer. Pick the lowest matching digit or fail on a tie.
- **BR-11** [Minor] `frame-clips-unwrapped-text` choiceFor wraps to opt.width (0 on a non-terminal or below 20 cols) while the frame clips at terminalCols (never 0)
  The wrap width is a policy answer with a sentinel; the clip width is a measurement.
  Where they disagree the gloss arrives unwrapped into a frame that cuts it —
  TestUnwrappedWidthLeavesTheGlossAlone pins the sentinel, which is correct for the
  pipe and wrong for a sitting.
- **BR-12** [Minor] `figures-drift` --play -raw records nothing but the bar still applies the transition
  decideCapture returns captureNothing under opt.raw (capture.go:30), so
  CaptureReview writes no event while held.answered still advances the in-memory
  progress. On that path D7's "cannot drift from what the next sitting derives" does
  not hold.
- **BR-13** [Minor] `test-fragility` TestTheBarCountsAnswersAsTheyLand slices on strings.Index without checking for -1
  play_loop_test.go:558 — a bar format change panics the test instead of failing it
  with the message the assertion was written to give.
- **BR-14** [Minor] `value-receiver-shared-state` sittingDeck has pointer-receiver mutators but is passed by value, sharing its map and half-sharing its slice
  Nothing production depends on the aliasing today, but it is what makes I-3's test
  pass and it is a landmine for #40's second consumer. A *sittingDeck parameter
  removes the question.

## Round 3 — 2026-08-31T16:42:01-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — viewportGesture exists (replraw.go:109) but has ONE caller (replraw.go:413); play_loop.go:235-248 still holds the verbatim four-case switch. The enterMouse-cost half IS addressed (replraw.go:63-66).
- BR-2 — addressed — TestAFailedLogReadStillRunsTheSitting drives the degraded path through a real bar draw via logRefusingStore.
- BR-3 — addressed — Both gates land before enterRaw; deleting them reddens TestPlayRefusesWhenStdoutIsNotATerminal. The -no-color half is pinned only by a pty row that skips here.
- BR-4 — addressed — Verified by mutation: removing the resize's opt.width update reddens TestANarrowedSittingWrapsTheRestOfItself. Option lines only; see the new definition-body finding.
- BR-5 — addressed — Mutation-verified both ways: an alias-free dropped() stays green, a no-op dropped() goes red on the drawn bar.
- BR-6 — addressed — crlf.go and crlf_test.go deleted, shortWriter re-homed. Residual mentions of the deleted FILE names roll into the new doc-sweep finding.
- BR-7 — addressed — newConsole(ctx, d, sess, stdout, newScreen) is called from replraw.go:33 and play_loop.go:96.
- BR-8 — addressed — refresh() assigns the whole figures() struct then re-applies total and done (play_loop.go:122-132).
- BR-9 — addressed — Same fix as BR-2.
- BR-10 — addressed — twiceNumberedOption now scans '1'..'9' in order rather than ranging the map (pty_conformance_test.go:944-949).
- BR-11 — addressed — choiceFor no longer wraps; the write-time wrap uses the live opt.width, and the non-terminal case is now refused outright. The sub-20-column sentinel remains as an explicit policy — see plan revision 3.
- BR-12 — not-addressed — decideCapture still returns captureNothing under opt.raw while held.answered advances the in-memory progress. Minor; never blocks.
- BR-13 — addressed — play_loop_test.go:558 now Fatalf's on a missing separator instead of slicing.
- BR-14 — addressed — sittingDeck travels by pointer from todaysQuestions through playSession; confirmed by the alias-free mutation staying green.

### Raised

- **BR-15** [Important] `frame-clips-unwrapped-text` A reveal written after a narrowing resize carries a definition wrapped to the STARTUP width, and the frame clips it
  This is the 3rd finding in family `frame-clips-unwrapped-text`. Earlier rounds fixed
  instances (the operator's startup-width option gloss, then BR-4's resize sibling). Do NOT
  fix this instance alone.
  
  THE RULE that covers all of them: `--play` RENDERS its text at queue-build time and WRITES
  it much later, so every pre-rendered artifact carries a width that may already be wrong when
  the frame clips it. Anything the loop writes into the buffer must be wrapped to the width in
  force at the moment of WRITING, not at the moment of rendering. `wrapOptionLines` applies
  that rule to exactly one line-kind; `todaysQuestions:436` calls `Render(..., Width: opt.width)`
  once per sitting for every definition, and `play_loop.go:323` writes those lines after any
  number of resizes.
  
  MEASURED prevalence (scratch probe, `playRig` with sycophantic/ephemeral/quokka/mesa,
  opt.width=100, resize 100->40, then one reveal): 7 buffer lines written AFTER the narrow
  exceed 40 columns, worst 88 cells. README.md:148 now claims "a definition longer than the
  window is scrolled rather than lost", which this contradicts.
  
  The enumeration the rule implies, to be swept in ONE round: q.Prompt(), asked.Reveal()
  (option lines AND the rendered definition body), the "removed %q from the deck" notice,
  finish()'s summary lines, and con.stderr diagnostics. Two candidate mechanisms: render the
  entry lazily at write time against the live opt.width, or keep the unwrapped source on the
  Question so a re-wrap is possible.
- **BR-16** [Important] `doc-sweep-incomplete` Five current-truth artifacts name symbols the tree does not have, and the new refusal surface reaches neither README nor atlas
  This is the 2nd finding in family `doc-sweep-incomplete`. Do NOT fix the instances one by
  one — that is what failed in rounds 1 and 2.
  
  THE RULE, and it is mechanisable because the repo already built most of it: every name a
  current-truth artifact cites must be DECLARED in the tree. `TestARemovedDeclarationIsSweptOrRetired`
  (repo_guard_test.go) only sees `-func` lines whose names pass `isCitableName` (exported or
  Test*), so it structurally cannot see a removed TYPE (`crlfWriter`), a removed unexported
  func (`playConsole`, `draw`, `fitMenu`), a deleted FILE path (`crlf_test.go`), or a name that
  was never declared at all. Widening it to (a) removed type/const/unexported declarations,
  (b) deleted file paths, and (c) a forward check that every `Test[A-Z]\w+` cited in a
  current-truth artifact is declared, turns this whole family into a build failure.
  
  MEASURED prevalence at HEAD, all introduced by this window:
    - cmd/define/play_loop.go:278 cites `TestAMissIsRecordedBeforeItIsRevealed`; the tree
      declares `TestAMissRecordsBeforeItPlays`. Introduced by 4d3b53a; never existed.
    - cmd/define/highlightwriter.go:60 says "crlf_test.go defends that" — file deleted here.
    - cmd/define/highlightwriter_test.go:158 cites "crlf_test.go's fixture", 400 lines above
      the fixture's new home in the same file.
    - atlas/define.md:2006 says "`playConsole` is `replRaw`'s construction"; the symbol was
      renamed to `newConsole` in the final commit.
    - workshop/plans/000038-play-clickable-plan.md:101, :128, :237 name `playConsole`.
    - Two prose breaks left by the sweep: atlas/define.md:942-943 ("It used to wrap the raw
      loop's / the raw loop's line-ending writer") and highlightwriter_test.go:142-144
      ("...right answer here while / A writer that can be written to again...").
  
  THE DOCS GATE, same rule at the behaviour level: this window added two user-facing refusals
  (`define --play > file` and `define --play -no-color` now print a message and exit 1,
  play_loop.go:67-74). Neither README.md nor atlas/define.md records them, and the atlas's
  "Degrading is the absence of input" paragraph (:470-476) still describes only the editor's
  degrade-by-routing. `newConsole`, `viewportGesture` and `wrapOptionLines` — three new shared
  surfaces `#40` will consume — appear in neither.
- **BR-17** [Important] `plan-table-vs-tree` The plan's Core concepts table describes two entities the tree does not have, and the guard checks only the status column
  `workshop/plans/000041-play-tui-plan.md:118` says `choiceFor` "takes the terminal width and
  wraps each gloss through `wrapText`" — BR-4 reverted exactly that, and `optionpool.go:228`
  declares `choiceFor(word, rendered string, e Entry, pool []play.Candidate, seed uint64)`.
  `:126` says `viewportGesture` is "PURE dispatch — the paging keys, for both loops"; it has
  one caller. The plan's BR-4 revision also asserts "all three are closed" when site (c),
  `opt.width == 0`, is a deliberate policy exception at cols < 20.
  
  `TestPlanTableStatusMatchesTheChangeWindow` passes both rows because it only judges the
  `modified`/`unchanged`/`new` column against `git diff` — the DESCRIPTION is unguarded. The
  plan's own rounds 2 and 3 stated the rule ("a Revision that reverses a decision re-reads the
  decision prose, the entity tables, AND the Done-when rows"); this is the fourth time a
  reversal reached the prose and not the table. `#40` reads this table as the record of what
  landed.
- **BR-18** [Minor] `terminal-ui-gate` The surface gate runs after todaysQuestions has already read the deck and written to the non-terminal stdout
  This is the 2nd finding in family `terminal-ui-gate`. THE RULE: every precondition for
  owning the terminal is settled in ONE place, before the command does any work or writes any
  byte — the same rule main.go:542 states for usage errors ("settled BEFORE a store is
  opened"). `runPlay` calls `todaysQuestions` at :33 and only checks `isTerminal(stdout)` and
  `opt.tty` at :67-74, so `define --play > file` on an empty deck writes "the deck is empty"
  into the file and exits 0 without ever reaching the refusal, and on a non-empty deck it pays
  the deck+log reads first. Moving the two gates above the `todaysQuestions` call closes both.
- **BR-19** [Minor] `lessons-not-recorded` workshop/lessons.md is untouched across a window that ran two review rounds and 14 findings
  AGENTS.md section 4: "When you run code review, add rules to workshop/lessons.md that
  prevent the mistakes you found." Three families repeated across rounds
  (`frame-clips-unwrapped-text`, `doc-sweep-incomplete`, `parallel-construction`) and the
  rules the plan wrote for itself live only in that plan's Revisions, which is archived at
  close. The two durable ones — "adopting an existing seam inherits its behaviour on inputs
  the previous consumer never sent it" and "re-examine the tests that assert over the surface
  this issue changes MEANS enumerating them" — belong in lessons.md, where the next issue
  reads them.

## Round 4 — 2026-08-31T17:13:53-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — viewportGesture called from replraw.go:413 and play_loop.go:247; stubbing the play call reddens TestPagingIsNotAnAnswer and TestALongRevealPagesRatherThanScrollingTheWordAway. The enterMouse cost is in the source comment only — folded into BR-16's docs gate.
- BR-12 — not-addressed — Unchanged at HEAD: decideCapture still returns captureNothing under opt.raw (capture.go:30, :135) while held.answered advances the in-memory progress; no --play/-raw conflict guard exists in main.go.
- BR-15 — addressed — Mutation-verified: making wrapWritten a no-op reddens TestANarrowedSittingWrapsTheRestOfItself with 7 overwide lines (worst 97 cells), covering option lines, the rendered definition body and the summary. The helper-write gap is raised separately as the family's 4th.
- BR-16 — not-addressed — Instances swept, but the closing commit added a new one and the mechanism was not built. See the finding detail below.
- BR-17 — addressed — All three named contradictions verified fixed against the tree: choiceFor is unchanged and its row says so, viewportGesture now has two callers, and site (c) is stated as a policy exception. The unguarded-column half is re-raised as the family's 2nd with new evidence.
- BR-18 — not-addressed — The gates did move above todaysQuestions (play_loop.go:52-72), but nothing pins it: moving them back below the call in a scratch worktree leaves the whole cmd/define suite green (107s, ok). The same round moved TestEmptyQueueExitsZero off runPlay onto todaysQuestions, deleting the only runPlay-on-empty-deck coverage.
- BR-19 — addressed — lessons.md gains three rules (seam adoption, enumerate the class, a plan naming an anti-pattern is not protection), each with the #41 evidence.

### Raised

- **BR-20** [Important] `frame-clips-unwrapped-text` playAnnounced writes into the clipping frame without wrapWritten, so a playback diagnostic is cut mid-sitting
  This is the 4th finding in family `frame-clips-unwrapped-text`. Do NOT fix this instance —
  the rule is already stated at playbar.go:66-87; what is missing is that it is enforced at
  call sites the LOOP owns, so any write made by a helper the loop calls escapes it.
  play_loop.go:354 calls playAnnounced, whose stderr is the screen; main.go:856 writes
  `define: %s` unwrapped, and main.go:844/:866 and reportVoice do the same.
  MEASURED (scratch probe: playRig, opt.width=40, newPinnedScreen(24,40), a player returning
  a network error, one miss then a reveal): one transcript line of 156 cells in a 40-column
  terminal, which Paint clips. atlas/define.md's new section already claims the loop routes
  "its diagnostics" through the one function; only the drop error at play_loop.go:312 is.
  Fix the class: wrap at the SEAM — the pinned screen's or the console's Write, against its
  own cols, with the sub-20-column policy in one place — so a helper cannot write around it,
  and extend the narrowing test to drive a failing playback so the predicate covers it.
- **BR-21** [Minor] `partial-copy-refresh` The initial sittingFigures hand-copies refresh()'s first two lines
  This is the 2nd finding in family `partial-copy-refresh`. BR-8 fixed refresh() copying
  named fields out of figures(); play_loop.go:129-130 is now the site that does it, so a
  third adjustment added to refresh() would be stale on the first frame, before any answer.
  The rule: one expression builds the figures and every site derives from it. refresh() is
  exactly equivalent at init because s.Right+s.Wrong is 0 there, so calling it is the fix.
- **BR-22** [Minor] `plan-table-vs-tree` Three pure entities are listed as Integration points and an IO constructor as Pure
  This is the 2nd finding in family `plan-table-vs-tree`. Do NOT fix only these rows.
  workshop/plans/000041-play-tui-plan.md:125-127 lists viewportGesture (whose own text says
  "PURE dispatch"), wrapWritten (string to string, playbar.go) and livePrompt (play.Session
  to string) under "Integration points", whose column header is "Wraps"; none wraps an
  external dependency and none is injected. Conversely newPinnedScreen (:109), which takes a
  tty and builds the IO shell, sits under "Pure entities".
  THE RULE: a table row asserts path, status, and kind/description, and
  TestPlanTableStatusMatchesTheChangeWindow judges only status. Either mechanise what can be
  — every backticked Name is DECLARED at the stated path, and an Integration row names a
  wrapped external dependency — or stop asserting the unguarded columns in the table `#40`
  reads as the record of what landed.

## Round 5 — 2026-08-31T17:41:30-07:00 (claude) — BLOCKED

### Disposed

- BR-12 — not-addressed — Nothing at HEAD touches it: no usage gate rejects `--play -raw`, decideCapture still returns captureNothing (capture.go:30), and held.answered/refresh still advance the bar.
- BR-16 — not-addressed — repo_guard_test.go is untouched in this window, so neither the removed-type/unexported/deleted-file widening nor the forward "cited Test name is declared" check exists; the class was filed to #33 while this round added two NEW instances — highlightwriter.go:60 cites TestHighlightWriterShortWriteContract (never existed; the real name is TestHighlightWriterTreatsAShortWriteAsAnError) and playbar.go:122-137 lost isOptionLine's doc comment to the inserted minWrapWidth const — plus two survivors: plan:311 still cites TestAMissIsRecordedBeforeItIsRevealed, and atlas/define.md:2054 still describes the deleted per-site wrap and the deleted resize-time opt.width update.
- BR-18 — not-addressed — The code is right (gates at play_loop.go:56-77, above todaysQuestions at :79) but nothing pins it — I moved the three gates back below todaysQuestions in a scratch copy and `go test ./cmd/define/` stayed green (107s); TestPlayRefusesWhenStdoutIsNotATerminal only checks exit 1, absence of ESC, and a stderr substring, none of which a "the deck is empty" line to stdout would break.
- BR-20 — addressed — The wrap moved to liveScreen.Write for pinned screens (screen.go:511) and playAnnounced's plain-text `define: %s` warning is now wrapped; pinned in both directions by TestThePinnedScreenWrapsWhateverIsWrittenToIt — but the same commit introduced a new hole in the seam, raised below.
- BR-21 — addressed — play_loop.go:146 calls refresh() at init; behaviour-identical to the hand-copy, so accepted as a structural fix with no possible failing test.
- BR-22 — addressed — viewportGesture/wrapWritten/livePrompt moved to Pure entities and newPinnedScreen to Integration points; #33 gained the widening spec in its own Revisions section.

### Raised

- **BR-23** [Critical] `frame-clips-unwrapped-text` wrapWritten skips every escape-carrying line, and --play can only run with colour on, so a reveal's rendered definition is never wrapped
  This is the 5th finding in family `frame-clips-unwrapped-text`, and it was INTRODUCED by the commit that closed the fourth. Do NOT fix this instance.
  playbar.go:105-109 returns any line containing 0x1b untouched. runPlay refuses unless isTerminal(stdout) and opt.tty, and main.go:523/527 define color and tty as the same expression, so every real sitting renders with Color:true and every styled line bypasses the seam. The same commit also deleted the resize case's opt.width maintenance that had been the prior mitigation.
  MEASURED (scratch: playRig + opt.color=true, opt.width=100, newPinnedScreen(24,40), one reveal): 4 buffer lines of 55-87 cells in a 40-column terminal, e.g. "    ephemerality \x1b[35m/..../\x1b[0m noun ephemerally ..." at 87 cells. Deleting the three-line skip takes that to 0 and leaves `go test ./cmd/define/...` green, so nothing defends it; wrapText already measures with visibleCells and strings.Fields cannot split inside an escape sequence.
  THE RULE, one step past "wrap at the seam": a seam covers every write only when (a) every path into the buffer goes through it and (b) its predicate is exercised in the configuration production actually runs in. The enumeration is two-by-two and belongs in the plan: paths = liveScreen.Write (wraps, except escapes) and liveScreen.WriteRegions (screen.go:551, calls l.s.Write directly and does NOT wrap — latent for --play, live for #40's board, and already contradicting screen.go:503's "Here nothing can write around it"); line classes = plain (wraps) and escape-carrying (does not).
  Fix the class: route both methods through one private writeBuffer(text) that applies the wrap when pinned, narrow the guard to the eraseLine gesture rather than "contains any escape", and drive TestANarrowedSittingWrapsTheRestOfItself with opt.color = true.
- **BR-24** [Important] `rig-config-not-production` playRig hardcodes color:false, a configuration --play now refuses to run in, so every in-process sitting test drives an unreachable state
  play_loop_test.go:39 returns options{color: false, width: 0, ...}. Since BR-3, runPlay exits 1 unless opt.tty, and opt.tty and opt.color are the identical expression (main.go:523/527) — so no in-process test has ever exercised the only configuration a sitting can be in. That is why the Critical above shipped: Done-when 0b's two pins (TestANarrowedSittingWrapsTheRestOfItself, TestALongOptionGlossWrapsRatherThanBeingCut) both pass with colour off. The pty rows do run coloured, but TestPTYPlayResizeRepaints narrows ROWS 24->10 and never COLS, so no row narrows the width of a coloured sitting.
  THE RULE: a rig's default options have to be reachable from the flag parse of the command under test. Set color/tty true in playRig (or derive both from one helper, as main.go does) and add a column-narrowing pty row.

## Open findings

- **BR-12** [Minor] `figures-drift` --play -raw records nothing but the bar still applies the transition
- **BR-16** [Important] `doc-sweep-incomplete` Five current-truth artifacts name symbols the tree does not have, and the new refusal surface reaches neither README nor atlas
- **BR-18** [Minor] `terminal-ui-gate` The surface gate runs after todaysQuestions has already read the deck and written to the non-terminal stdout
- **BR-23** [Critical] `frame-clips-unwrapped-text` wrapWritten skips every escape-carrying line, and --play can only run with colour on, so a reveal's rendered definition is never wrapped
- **BR-24** [Important] `rig-config-not-production` playRig hardcodes color:false, a configuration --play now refuses to run in, so every in-process sitting test drives an unreachable state
