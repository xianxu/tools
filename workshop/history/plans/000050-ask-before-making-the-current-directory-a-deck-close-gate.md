---
gate: boundary-review
issue: 50
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-10T17:56:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: '"nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open'
          detail: |-
            Verified by mutation in a scratch worktree: making SetItems write to g.disk AND
            route through g.creating() leaves the entire suite green while a declined
            directory gains an items/ tree. A scratch test listing the directory after a
            denied SetItems went red under the mutation and green after revert. Fix as a
            class: iterate storeInterfaceMethods (not the createsOnDisk map) with a
            sampleArgs table and assert the directory is byte-identical under denial.
          family: guard-fails-open
          round: 1
        - id: BR-2
          severity: Important
          title: the denied session's fallback store is per-wrapper, so a /lang rebuild discards it
          detail: |-
            newGatedStore allocates store.NewMem() per wrapper (gated_store.go:32) and
            newLangDeps rebuilds the wrapper (main.go:339). Measured: allowed keeps 1 word
            across a rebuild, denied drops to 0 — contradicting "a denied session still
            recalls itself". This is PQ-2's rule applied to the decision but not to the
            store it swaps in. Key the fallback by (dir, lang), memoized in openStore.
          family: process-scoped-state-per-wrapper
          round: 1
        - id: BR-3
          severity: Important
          title: M2 surface (-here, deckAsker) landed at the M1 boundary, inert and untested
          detail: |-
            -here prints in --help as "make this directory a deck without asking" but has
            no production reader (main.go:414,474,601); deckAsker (deckperm.go:132) has no
            caller and no committed test (deckasker_test.go is untracked). The new atlas
            section describes the question and the decline as current truth, which the
            committed binary cannot produce. M1 is stated to be a true no-op.
          family: unshipped-surface-claimed-as-current
          round: 1
        - id: BR-4
          severity: Important
          title: Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
          detail: |-
            Done-when says "every write-shaped method consults the gate" — PQ-8 made that
            false on purpose (Forget). The Plan row says "8 gated writes, 7 ungated reads";
            the code ships 7 gated / 8 ungated. The plan's table lists store.IsDeck as PURE
            though it calls os.ReadDir and its tests need a mutable filesystem. No
            "## Revisions" section exists in either artifact.
          family: artifact-claims-what-code-does-not
          round: 1
        - id: BR-5
          severity: Important
          title: README update missing for -here, and the plan has no README step at all
          detail: |-
            cmd/define/README.md is the binding user documentation and gains no mention of
            -here, which the Spec calls "the only path automation has". Task 10 Step 7
            names only the atlas, so nothing remaining in M2 will catch this before the
            merge-time specs judge.
          family: readme-gate
          round: 1
        - id: BR-6
          severity: Minor
          title: deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
          detail: |-
            deckperm.go:149 may consume up to 4 KiB past the answer and discard it. M2 Task
            8 resolves the permission immediately before the REPL reads the same stdin, so
            a pasted "y\ndog\n" would lose the word. Read byte-wise to '\n', or share the
            loop's reader.
          family: stdin-over-read
          round: 1
        - id: BR-7
          severity: Minor
          title: the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
          detail: |-
            The AST guard forces every method into a bucket and the consultation test forces
            the bucket to match the code, but moving a method to the wrong bucket AND
            switching creating() to reading() together passes everything. The class-level
            denied-directory test above closes this as a side effect.
          family: guard-fails-open
          round: 1
        - id: BR-8
          severity: Minor
          title: storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
          detail: |-
            Only m.Names is read, so an embedded interface contributes zero methods. Deriving
            the floor from reflect.TypeOf((*store.Store)(nil)).Elem().NumMethod() would make
            it self-maintaining.
          family: derivation-under-derives
          round: 1
        - id: BR-9
          severity: Minor
          title: IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
          detail: |-
            Early exit does not help: ReadDir materialises and sorts the full list first. In a
            home directory or monorepo root that is a needless O(n log n) on the startup path.
            f.ReadDir(-1) unsorted avoids the sort. ARCH-CONSTRAINTS, once per process.
          family: startup-path-cost
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-10T23:26:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Re-measured: BR-1''s own mutation leaves the whole suite green; zero-value args make 5 of 7 subtests vacuous and the positive control waives per-method checking.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Revert-verified — restoring the per-wrapper store.NewMem() reddens gated_store_test.go:405 with its own message.
          round: 2
        - id: BR-3
          disposition: addressed
          note: deckasker_test.go is committed, -here has production readers, and --here creates events/ unasked on the real binary.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Done-when still says "write-shaped", the Plan row still says 8/7, IsDeck is still listed PURE, and neither artifact has a "## Revisions" section.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Both READMEs document the question, the decline and --here; --help is a separate surface, raised below.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: readLineUnbuffered is the right code but no test fails without it; one assertion on the unread remainder closes it.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: The class test iterates createsOnDisk, not storeInterfaceMethods, so a coordinated bucket-move plus creating()->reading() is still invisible.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: storeInterfaceMethods still reads only m.Names and the floor is still a hand-typed "< 15".
          round: 2
        - id: BR-9
          disposition: not-addressed
          note: Still os.ReadDir, and settleQuietly now sits on the read path so it is no longer once per process while undecided (measured 5 evaluations for 5 reads; 1 in a real one-shot lookup).
          round: 2
      findings:
        - id: BR-10
          severity: Important
          title: third round of guard-fails-open — the rule is that an absence or ordering claim needs a per-instance control, not an aggregate one
          detail: |-
            This is the 3rd+ finding in family guard-fails-open (BR-1, BR-7, and now the
            loop-shell ordering claim). Do NOT fix the instances. Measured at the pinned head
            in a scratch worktree: (1) BR-1's dual-write SetItems mutation leaves the entire
            cmd/define suite green, exit 0; (2) of the 7 createsOnDisk methods only
            AppendEvent and SetUserModel write anything under callStoreMethod's zero-value
            args, so 5 of 7 subtests of TestNoCreatingMethodTouchesADeclinedDirectory are
            no-ops, and TestCreatingMethodsDoReachAnAllowedDirectory explicitly waives the
            per-method check that would have caught it; (3) moving repl.go:245's
            resolve() below both shells leaves TestBothLoopShellsResolveBeforeReading and the
            full 111s suite green, so PQ-3's ordering claim is pinned for 0 of 2 shells. The
            rule: every assertion of an absence or an ordering must be paired with a mutation
            that makes it false and shown to redden THAT assertion, per instance. Write the
            enumeration (a sampleArgs entry per creating method with a per-method positive
            assertion, and a Read-recording seam for the ordering) in one pass.
          family: guard-fails-open
          round: 2
        - id: BR-11
          severity: Important
          title: second round of readme-gate — --help still promises unconditional recording, and the README quotes a prompt string nothing keeps in step
          detail: |-
            This is the 2nd finding in family readme-gate. Do NOT fix only the instance.
            main.go:528-531's usage prose still says define "records what you look up under
            words/ and events/ in the CURRENT DIRECTORY … A word that was found is added to
            the deck", which is false in the third state; --help is the first surface a user
            types. The rule: when behaviour changes, enumerate every place the old behaviour
            is asserted and sweep them in one round — README.md, cmd/define/README.md,
            atlas/define.md, the fs.Usage prose, the issue's Done-when, the plan. Cheap
            structural half: cmd/define/README.md:52 quotes the literal string deckperm.go:195
            prints, and this package already owns doc_sync_test.go for exactly that class.
          family: readme-gate
          round: 2
        - id: BR-12
          severity: Important
          title: deckPolicy and deckAsker independently encode the same three-way precedence
          detail: |-
            deckperm.go:149-157 and :181-194 each implement already-a-deck / --here /
            no-terminal. They agree today and nothing makes them. Observable consequence
            already present: only deckAsker prints the "nothing will be saved (use --here)"
            explanation, so whether a piped user is told depends on which encoding settles the
            state first — confirmed on the real binary (echo word | define prints it,
            define --forget cat piped does not). ARCH-DRY: deckAsker should switch on
            deckPolicy and own only the deckUndecided arm.
          family: policy-restated-not-derived
          round: 2
        - id: BR-13
          severity: Minor
          title: renderStats gained a bare positional bool, read at six call sites as a literal true
          detail: |-
            stats.go:101's `saving bool` appears as renderStats(s, now, true) in stats_test.go
            and deckasker_test.go. A named type or a field makes the call sites self-describing.
          family: positional-bool-parameter
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-10T23:56:17-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by mutation: a dual-writing SetItems now reddens TestCreatingMethodsWriteWhenAllowedAndNotWhenDenied/SetItems.'
          round: 3
        - id: BR-4
          disposition: addressed
          note: Both artifacts gained "## Revisions"; the residual Pure-entities bullet for IsDeck is a new Minor.
          round: 3
        - id: BR-6
          disposition: addressed
          note: 'Verified by reversion to bufio: the remainder assertion goes red.'
          round: 3
        - id: BR-7
          disposition: addressed
          note: 'Verified: moving SetItems to doesNotCreate plus routing via reading() reddens, because sampleCalls is independent of the buckets.'
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: Still only m.Names, and the floor is still a hand-typed "< 15".
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: 'Still os.ReadDir. Measured cost: 2 calls per terminal one-shot, 3 per terminal --stats, 1 piped.'
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: |-
            The rule holds for the 7 creating methods only. Measured at the pinned head in a scratch
            worktree: (1) moving repl.go:245's resolve() into replLines leaves the entire suite green,
            so the "both loop shells" ordering claim is still pinned for 0 of 1 raw shells — both
            subtests of TestBothLoopShellsResolveBeforeReading reach replLines, the "raw" one via
            replRaw's non-file-stdin fallback, as its own name concedes; (2) making YAML.Forget call
            MkdirAll(wordsDir) leaves the entire suite green while a declined directory grows words/en/,
            so the doesNotCreate bucket's absence claim has no control at all — only "does not consult
            the permission" is asserted. The enumeration BR-1 asked for was storeInterfaceMethods, and
            what shipped enumerates sampleCalls, whose completeness is checked only against createsOnDisk.
            Write it once, for the class: a sample per INTERFACE method with a per-method positive
            control and a byte-identical denied directory, and a raw-path ordering observation in the
            existing pty conformance seam.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: |-
            --help was fixed, the sweep was not. cmd/define/README.md:586-587 still reads "Every
            successful lookup - one-shot, piped, or in the editor - records the word where you started
            define, so your deck and history build themselves", which is false in the third state and
            names the piped case by name; :702 "-raw records nothing ... and neither does it ask" is now
            contradicted by the program (see the new finding). The test written to close this family
            asserts only that each surface contains the substring "-here", which a document can satisfy
            while asserting the old behaviour elsewhere in the same file. The rule, restated so it can
            catch the next instance: a surface test must assert the OLD claim is GONE, not that a new
            word is present - enumerate the sentences that assert the superseded behaviour and pin their
            absence.
          round: 3
        - id: BR-12
          disposition: not-addressed
          note: |-
            The duplicate encoding is genuinely gone (deckAsker switches on deckPolicy), but the
            consequence the finding cited survives. Measured with the real wiring: `define sycophantic`
            with stdin not a terminal in a fresh directory prints NO explanation (stderr is only the
            pronunciation warning), while `echo sycophantic | define` prints "nothing will be saved (use
            --here to create one)". Traced: the one-shot path has a READ settle the state to deckDeny
            through settleQuietly - silently - so the later write finds it decided and deckAsker, the
            only thing that says why, is never invoked. The effect belongs to the TRANSITION into
            deckDeny, not to one of the paths that can cause it; deckPermission should emit it once when
            it settles to deny, whichever path settles it. The same gap has a second face: only the EMPTY
            --stats screen says nothing is being saved, so a declined session that has looked three words
            up shows ordinary figures with no hint they are session-only.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: Still a bare positional bool; six call sites read renderStats(s, now, true).
          round: 3
      findings:
        - id: BR-14
          severity: Important
          title: -raw prompts to create a deck, and tells piped scripts to use --here, though it writes nothing
          detail: |-
            This is the 2nd finding in family policy-restated-not-derived (BR-12 is the 1st, and it is
            re-raised above). Do NOT fix this instance alone. The rule: a policy this codebase already
            owns in one place must be DERIVED at every new site - including the site that decides whether
            to build the permission at all. decideCapture (capture.go:26-33) is the single source of "this
            invocation writes nothing", and capture.go:192 already states that DEFINE_NO_CAPTURE and -raw
            are one class; main.go:806 restates half of it as `!opt.noCapture`, so -raw falls through into
            the gate. Measured at the pinned head with the real wiring: `define -raw` on a terminal in a
            fresh directory prints "... is not a deck yet. Create one here? [y/N]", answering y creates
            nothing at all, and the question consumes the session's first line of stdin as its answer;
            piped, every run prints "nothing will be saved (use --here to create one)" to stderr,
            recommending a flag that would not make -raw save anything. The Spec forbids exactly this
            ("a prompt for a command that would not have written anything is a false alarm"), and
            cmd/define/README.md:702 already documents -raw as never asking. Enumerate the members of the
            write-nothing class from decideCapture and gate on the class.
          family: policy-restated-not-derived
          round: 3
        - id: BR-15
          severity: Important
          title: Ctrl-C at the deck question does nothing - the prompt blocks with no cancellation arm
          detail: |-
            main.go:401 installs signal.NotifyContext, and repl.go calls detachedInterrupts BEFORE
            repl.go:245's resolve(), so by the time the question is on screen SIGINT is intercepted
            process-wide and delivered to a context nobody is watching. deckAsker (deckperm.go:179) takes
            no context and blocks in readLineUnbuffered until a newline or EOF, so Ctrl-C at the prompt
            leaves the question up; the only exits are Enter, which silently declines, and Ctrl-D. The
            program otherwise owns what Ctrl-C means, which is why a reader will expect it to work here.
            Give the asker the cancellation channel and treat a cancelled context as an explicit,
            printed decline.
          family: blocking-prompt-ignores-cancellation
          round: 3
        - id: BR-16
          severity: Minor
          title: the plan's Core concepts never gained deckPolicy, and still lists IsDeck under Pure entities
          detail: |-
            This is the 2nd finding in family artifact-claims-what-code-does-not (BR-4 was the 1st and is
            disposed addressed). Do NOT fix only this row. The rule: the Core-concepts table is the
            greppable contract a boundary review cross-checks, so any entity the implementation ADDS or
            RECLASSIFIES lands in the table with a "## Revisions" entry in the same commit that adds it.
            Instances: deckPolicy, withQuiet and settleQuietly - introduced in M2 after smoke testing -
            appear in no table; and while BR-4 moved the store.IsDeck table row to Integration points, the
            bullet describing it (plan:41) still sits under the "### Pure entities" heading, so the plan
            files it in both sections.
          family: artifact-claims-what-code-does-not
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-11T00:28:01-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: Still only m.Names and a hand-typed "< 15" floor; latent, since store.Store embeds nothing today.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: Still os.ReadDir, and deckPolicy runs twice on the ask path (quiet, then ask), so two ReadDirs there.
          round: 4
        - id: BR-10
          disposition: addressed
          note: 'Revert-verified: Forget->MkdirAll reddens TestNonCreatingMethodsCreateNothing/Forget; removing resolve() reddens both TestTheLineShellResolvesBeforeReading subtests. The raw-shell pty pin could not run here either (openpty EPERM, unsandboxed), so it has never been observed; run it with CONFORMANCE_STRICT=1 on a pty-capable machine before merge.'
          round: 4
        - id: BR-11
          disposition: addressed
          note: Exact literal reverts of the fs.Usage prose and the README paragraph each redden TestEverySurfaceDescribingCaptureMentionsTheQuestion; deckPrompt is pinned against the README. The /lang surfaces belong to the new confirmation finding.
          round: 4
        - id: BR-12
          disposition: addressed
          note: 'Both explanation mutations (no sayWhy on settle; resolve skipping the quiet half) redden; real binary: redirected and piped lookups each explain once. The non-empty /stats "second face" is outside the Spec, which asked only for the empty message.'
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: Still renderStats(s, now, true) at every call site.
          round: 4
        - id: BR-14
          disposition: addressed
          note: 'Reverting the -raw branch reddens TestRawNeitherAsksNorAdvises; real binary: -raw piped and one-shot print no advice and create nothing. The derivation the finding asked for was not done; raised below as a Minor.'
          round: 4
        - id: BR-15
          disposition: addressed
          note: Reverting readLineCancellable hangs TestInterruptAtTheQuestionDeclines 5s and reddens; reachable, because deckAsker holds run()'s NotifyContext ctx, which SIGINT still cancels. The post-cancel message is part of the new confirmation finding.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: 'Plan untouched since a30cb79: deckPolicy, deckReason, explainDenial, readLineCancellable, withQuiet, settleQuietly absent; IsDeck bullet still under Pure entities; plan''s deckAsker lacks ctx and says four inputs (five now), and deckAsker''s own doc comment says FOUR INPUTS and cites a nonexistent deckPermission.settle.'
          round: 4
      findings:
        - id: BR-17
          severity: Important
          title: '4th guard-fails-open: the issue''s Done-when list is the enumeration never written, and its --stats row is pinned by nothing'
          detail: 'This is the 4th finding in family guard-fails-open. Earlier rounds applied the per-instance rule to components (store methods, buckets, loop shells), and those now hold. The enumeration never written is the issue''s own Done-when list, where the feature''s claims live. Measured at the pinned head: making printStats render the saving screen unconditionally (stats.go:87), or handing both doors a nil permission (stats.go:47 and :260), each leaves the whole cmd/define package green (110s). So the ticked row "--stats in an unsaved directory does not claim a word will join your deck", the exact defect smoke testing found, is pinned by nothing through production wiring. deckperm_e2e_test.go:174 asserts "Nothing yet", which both screens print, and TestStatsIsHonestWhereTheAnswerNeedsNobody computes its own copy of the formula (deckasker_test.go:269) instead of calling printStats. The rule: a ticked Done-when row is pinned only by a test that fails when the row is made false through production wiring. Write a Log table for all seven rows: row, test, wiring mutation, observed red. Sweep result: rows 1, 3, 5 and 6 have discriminating controls; row 7 has none; row 2''s --play and /history half is pinned only structurally (non-nil gatedStore), with no test running either under a declined permission.'
          family: guard-fails-open
          round: 4
        - id: BR-18
          severity: Important
          title: /lang reports a switch and "now on the record" in a declined directory, because persistLang swallows the write its only caller reports
          detail: 'The Spec names this class ("The message that would become a lie") and --stats was fixed; /lang was not. persistLang returns nil when declined (main.go:385-389), and TestPersistLangIsGated pins that nil, so lang_cmd.go:72 and :69 report success. Real binary, fresh directory, stdin redirected: define /lang es prints "now defining in es", exit 0, writes nothing, and the next define /lang says en. define /lang en prints "still defining in en, now on the record". A one-shot /lang has only a durable effect, so in the third state it is a no-op reporting a switch. The pre-#50 binary wrote lang.txt, and the no-directory case already refuses honestly (lang_cmd.go:57). TestLangAfterADeclineWritesNothing never reads stdout. Enumeration: lang_cmd.go:69 and :72 are reachable. The harvest, reflect, forget and play "saved/wrote/removed" messages all need a non-empty deck, which a declined one-shot cannot have. deckperm.go:263 prints "the lookup still works" on Ctrl-C, but in the REPL the same SIGINT fires detachedInterrupts'' default cancel and the session exits with no lookup (scratch test: run returned 0). Docs asserting the old behaviour: cmd/define/README.md:649-651 and atlas/define.md:1128. Fix: persistLang reports a declined sentinel instead of nil; /lang says the language applies to this session only and was not saved; pin it on stdout. While in that README, :590 links the question to the Install anchor instead of the section that describes it.'
          family: confirmation-not-derived-from-effect
          round: 4
        - id: BR-19
          severity: Minor
          title: deckPolicy re-tests opt.raw instead of consuming decideCapture, the single owner of "this invocation writes nothing"
          detail: 'This is the 3rd finding in family policy-restated-not-derived (BR-12, BR-14). The rule: a decision this codebase already owns is consumed by calling its owner, never by re-testing the owner''s inputs at a new site. BR-14''s behaviour is fixed, but deckperm.go:210 re-tests opt.raw instead of asking decideCapture (capture.go:26), which capture.go:198 already consults for the same question. A third captureNothing member would reach the REPL''s up-front question again. Enumerated: this is the only site in the diff restating decideCapture. main.go:806''s noCapture check mirrors openStore''s "anywhere to read at all" and is correct as is. One line: decideCapture(true, opt) == captureNothing, with reasonRaw renamed for the class.'
          family: policy-restated-not-derived
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-09-11T00:52:19-07:00"
      agent: claude
      dispose:
        - id: BR-8
          disposition: not-addressed
          note: gated_store_test.go:54-57 still reads only m.Names, and the floor at :75 is still a literal 15.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: isdeck.go:34 unchanged; and it is not once per process - settleQuietly does not memoize deckUndecided, so every gatedStore.reading() re-runs deckPolicy. Measured 3 ReadDir calls for one declined terminal one-shot.
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: stats.go:101 still takes a bare positional bool, read as a literal at 9 call sites.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: Plan unchanged at the pinned head - no Revisions entry, no deckPolicy/deckReason/errDeckDeclined/withQuiet/settleQuietly rows, and the IsDeck bullet still sits under the Pure entities heading at plan:41.
          round: 5
        - id: BR-17
          disposition: addressed
          note: Both mutations verified red in a scratch worktree, and the Done-when pin table is written. The sibling door it did not sweep is raised separately.
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: The /lang half is fixed and mutation-verified, but the enumeration this finding named is incomplete - the Ctrl-C message residual is measured below, and the two doc sites plus the :590 anchor are untouched.
          round: 5
        - id: BR-19
          disposition: not-addressed
          note: The code change landed and is correct, but reverting deckperm.go:232 to `if opt.raw` leaves the whole cmd/define package green on a full 110s run - nothing pins the derivation.
          round: 5
      findings:
        - id: BR-20
          severity: Important
          title: '5th guard-fails-open: printStats has two doors and only the one the Done-when row named is pinned'
          detail: 'Measured at the pinned head: replacing stats.go:260 with printStats(c.deck, c.clock, "/stats", nil, c.stdout, c.stderr) leaves `go test ./cmd/define/` entirely green (110s), so the in-REPL door into the exact screen this issue exists for is pinned by nothing. Do NOT fix this instance. The rule: when a claim is about a shared renderer or decision, the pin count is DERIVED FROM THAT FUNCTION''S CALLER SET, not from the single entry point a Done-when row happens to name. printStats has exactly two callers (stats.go:47 and :260) and TestStatsInAnUnsavedDirectoryReportsEmpty drives only the first. Behaviour is currently correct - I ran /stats in a declined session and it renders the honest screen - so this is a coverage hole, not a live bug. The same caller-set enumeration applies to --play vs /play and to one-shot vs REPL /lang; write it once and derive the subtests.'
          family: guard-fails-open
          round: 5
        - id: BR-21
          severity: Important
          title: '3rd readme-gate: the superseded-claims enumeration is hand-maintained, so the /lang persistence promise went stale'
          detail: 'Earlier rounds fixed --help, then "Every successful lookup". Do NOT fix these two sentences only. The rule: deckasker_test.go:347''s `superseded` array IS the enumeration, and every behaviour change that narrows a documented promise adds its superseded sentence there in the same commit that narrows it. Prevalence - the array has 2 entries, both added reactively one per round; this round narrowed a third promise and added nothing. Now stale and unconditional: cmd/define/README.md:648-649 "`/lang es` switches, and the setting stays with the directory", :651 "It has to persist", and atlas/define.md:1128 "`/lang` reports when bare and persists when given" - all false in a declined directory, which is the state persistLang now reports with errDeckDeclined. Also cmd/define/README.md:590 links the question to [Install](#install) rather than to "## The directory is the deck, so it asks first", the section that actually describes it.'
          family: readme-gate
          round: 5
      blocked: false
---

# Gate ledger — tools#50 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T17:56:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `guard-fails-open` "nothing reaches a declined directory" is pinned for 2 of 7 creating methods, and the guard fails open
  Verified by mutation in a scratch worktree: making SetItems write to g.disk AND
  route through g.creating() leaves the entire suite green while a declined
  directory gains an items/ tree. A scratch test listing the directory after a
  denied SetItems went red under the mutation and green after revert. Fix as a
  class: iterate storeInterfaceMethods (not the createsOnDisk map) with a
  sampleArgs table and assert the directory is byte-identical under denial.
- **BR-2** [Important] `process-scoped-state-per-wrapper` the denied session's fallback store is per-wrapper, so a /lang rebuild discards it
  newGatedStore allocates store.NewMem() per wrapper (gated_store.go:32) and
  newLangDeps rebuilds the wrapper (main.go:339). Measured: allowed keeps 1 word
  across a rebuild, denied drops to 0 — contradicting "a denied session still
  recalls itself". This is PQ-2's rule applied to the decision but not to the
  store it swaps in. Key the fallback by (dir, lang), memoized in openStore.
- **BR-3** [Important] `unshipped-surface-claimed-as-current` M2 surface (-here, deckAsker) landed at the M1 boundary, inert and untested
  -here prints in --help as "make this directory a deck without asking" but has
  no production reader (main.go:414,474,601); deckAsker (deckperm.go:132) has no
  caller and no committed test (deckasker_test.go is untracked). The new atlas
  section describes the question and the decline as current truth, which the
  committed binary cannot produce. M1 is stated to be a true no-op.
- **BR-4** [Important] `artifact-claims-what-code-does-not` Done-when, the M1 Plan row, and the plan's Core-concepts table each state what the code deliberately does not do
  Done-when says "every write-shaped method consults the gate" — PQ-8 made that
  false on purpose (Forget). The Plan row says "8 gated writes, 7 ungated reads";
  the code ships 7 gated / 8 ungated. The plan's table lists store.IsDeck as PURE
  though it calls os.ReadDir and its tests need a mutable filesystem. No
  "## Revisions" section exists in either artifact.
- **BR-5** [Important] `readme-gate` README update missing for -here, and the plan has no README step at all
  cmd/define/README.md is the binding user documentation and gains no mention of
  -here, which the Spec calls "the only path automation has". Task 10 Step 7
  names only the atlas, so nothing remaining in M2 will catch this before the
  merge-time specs judge.
- **BR-6** [Minor] `stdin-over-read` deckAsker reads the answer through a throwaway bufio.Reader over shared stdin
  deckperm.go:149 may consume up to 4 KiB past the answer and discard it. M2 Task
  8 resolves the permission immediately before the REPL reads the same stdin, so
  a pasted "y\ndog\n" would lose the word. Read byte-wise to '\n', or share the
  loop's reader.
- **BR-7** [Minor] `guard-fails-open` the createsOnDisk/doesNotCreate split is hand-maintained; a coordinated reclassification is invisible
  The AST guard forces every method into a bucket and the consultation test forces
  the bucket to match the code, but moving a method to the wrong bucket AND
  switching creating() to reading() together passes everything. The class-level
  denied-directory test above closes this as a side effect.
- **BR-8** [Minor] `derivation-under-derives` storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
  Only m.Names is read, so an embedded interface contributes zero methods. Deriving
  the floor from reflect.TypeOf((*store.Store)(nil)).Elem().NumMethod() would make
  it self-maintaining.
- **BR-9** [Minor] `startup-path-cost` IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
  Early exit does not help: ReadDir materialises and sorts the full list first. In a
  home directory or monorepo root that is a needless O(n log n) on the startup path.
  f.ReadDir(-1) unsorted avoids the sort. ARCH-CONSTRAINTS, once per process.

## Round 2 — 2026-09-10T23:26:34-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Re-measured: BR-1's own mutation leaves the whole suite green; zero-value args make 5 of 7 subtests vacuous and the positive control waives per-method checking.
- BR-2 — addressed — Revert-verified — restoring the per-wrapper store.NewMem() reddens gated_store_test.go:405 with its own message.
- BR-3 — addressed — deckasker_test.go is committed, -here has production readers, and --here creates events/ unasked on the real binary.
- BR-4 — not-addressed — Done-when still says "write-shaped", the Plan row still says 8/7, IsDeck is still listed PURE, and neither artifact has a "## Revisions" section.
- BR-5 — addressed — Both READMEs document the question, the decline and --here; --help is a separate surface, raised below.
- BR-6 — not-addressed — readLineUnbuffered is the right code but no test fails without it; one assertion on the unread remainder closes it.
- BR-7 — not-addressed — The class test iterates createsOnDisk, not storeInterfaceMethods, so a coordinated bucket-move plus creating()->reading() is still invisible.
- BR-8 — not-addressed — storeInterfaceMethods still reads only m.Names and the floor is still a hand-typed "< 15".
- BR-9 — not-addressed — Still os.ReadDir, and settleQuietly now sits on the read path so it is no longer once per process while undecided (measured 5 evaluations for 5 reads; 1 in a real one-shot lookup).

### Raised

- **BR-10** [Important] `guard-fails-open` third round of guard-fails-open — the rule is that an absence or ordering claim needs a per-instance control, not an aggregate one
  This is the 3rd+ finding in family guard-fails-open (BR-1, BR-7, and now the
  loop-shell ordering claim). Do NOT fix the instances. Measured at the pinned head
  in a scratch worktree: (1) BR-1's dual-write SetItems mutation leaves the entire
  cmd/define suite green, exit 0; (2) of the 7 createsOnDisk methods only
  AppendEvent and SetUserModel write anything under callStoreMethod's zero-value
  args, so 5 of 7 subtests of TestNoCreatingMethodTouchesADeclinedDirectory are
  no-ops, and TestCreatingMethodsDoReachAnAllowedDirectory explicitly waives the
  per-method check that would have caught it; (3) moving repl.go:245's
  resolve() below both shells leaves TestBothLoopShellsResolveBeforeReading and the
  full 111s suite green, so PQ-3's ordering claim is pinned for 0 of 2 shells. The
  rule: every assertion of an absence or an ordering must be paired with a mutation
  that makes it false and shown to redden THAT assertion, per instance. Write the
  enumeration (a sampleArgs entry per creating method with a per-method positive
  assertion, and a Read-recording seam for the ordering) in one pass.
- **BR-11** [Important] `readme-gate` second round of readme-gate — --help still promises unconditional recording, and the README quotes a prompt string nothing keeps in step
  This is the 2nd finding in family readme-gate. Do NOT fix only the instance.
  main.go:528-531's usage prose still says define "records what you look up under
  words/ and events/ in the CURRENT DIRECTORY … A word that was found is added to
  the deck", which is false in the third state; --help is the first surface a user
  types. The rule: when behaviour changes, enumerate every place the old behaviour
  is asserted and sweep them in one round — README.md, cmd/define/README.md,
  atlas/define.md, the fs.Usage prose, the issue's Done-when, the plan. Cheap
  structural half: cmd/define/README.md:52 quotes the literal string deckperm.go:195
  prints, and this package already owns doc_sync_test.go for exactly that class.
- **BR-12** [Important] `policy-restated-not-derived` deckPolicy and deckAsker independently encode the same three-way precedence
  deckperm.go:149-157 and :181-194 each implement already-a-deck / --here /
  no-terminal. They agree today and nothing makes them. Observable consequence
  already present: only deckAsker prints the "nothing will be saved (use --here)"
  explanation, so whether a piped user is told depends on which encoding settles the
  state first — confirmed on the real binary (echo word | define prints it,
  define --forget cat piped does not). ARCH-DRY: deckAsker should switch on
  deckPolicy and own only the deckUndecided arm.
- **BR-13** [Minor] `positional-bool-parameter` renderStats gained a bare positional bool, read at six call sites as a literal true
  stats.go:101's `saving bool` appears as renderStats(s, now, true) in stats_test.go
  and deckasker_test.go. A named type or a field makes the call sites self-describing.

## Round 3 — 2026-09-10T23:56:17-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by mutation: a dual-writing SetItems now reddens TestCreatingMethodsWriteWhenAllowedAndNotWhenDenied/SetItems.
- BR-4 — addressed — Both artifacts gained "## Revisions"; the residual Pure-entities bullet for IsDeck is a new Minor.
- BR-6 — addressed — Verified by reversion to bufio: the remainder assertion goes red.
- BR-7 — addressed — Verified: moving SetItems to doesNotCreate plus routing via reading() reddens, because sampleCalls is independent of the buckets.
- BR-8 — not-addressed — Still only m.Names, and the floor is still a hand-typed "< 15".
- BR-9 — not-addressed — Still os.ReadDir. Measured cost: 2 calls per terminal one-shot, 3 per terminal --stats, 1 piped.
- BR-10 — not-addressed — The rule holds for the 7 creating methods only. Measured at the pinned head in a scratch
worktree: (1) moving repl.go:245's resolve() into replLines leaves the entire suite green,
so the "both loop shells" ordering claim is still pinned for 0 of 1 raw shells — both
subtests of TestBothLoopShellsResolveBeforeReading reach replLines, the "raw" one via
replRaw's non-file-stdin fallback, as its own name concedes; (2) making YAML.Forget call
MkdirAll(wordsDir) leaves the entire suite green while a declined directory grows words/en/,
so the doesNotCreate bucket's absence claim has no control at all — only "does not consult
the permission" is asserted. The enumeration BR-1 asked for was storeInterfaceMethods, and
what shipped enumerates sampleCalls, whose completeness is checked only against createsOnDisk.
Write it once, for the class: a sample per INTERFACE method with a per-method positive
control and a byte-identical denied directory, and a raw-path ordering observation in the
existing pty conformance seam.
- BR-11 — not-addressed — --help was fixed, the sweep was not. cmd/define/README.md:586-587 still reads "Every
successful lookup - one-shot, piped, or in the editor - records the word where you started
define, so your deck and history build themselves", which is false in the third state and
names the piped case by name; :702 "-raw records nothing ... and neither does it ask" is now
contradicted by the program (see the new finding). The test written to close this family
asserts only that each surface contains the substring "-here", which a document can satisfy
while asserting the old behaviour elsewhere in the same file. The rule, restated so it can
catch the next instance: a surface test must assert the OLD claim is GONE, not that a new
word is present - enumerate the sentences that assert the superseded behaviour and pin their
absence.
- BR-12 — not-addressed — The duplicate encoding is genuinely gone (deckAsker switches on deckPolicy), but the
consequence the finding cited survives. Measured with the real wiring: `define sycophantic`
with stdin not a terminal in a fresh directory prints NO explanation (stderr is only the
pronunciation warning), while `echo sycophantic | define` prints "nothing will be saved (use
--here to create one)". Traced: the one-shot path has a READ settle the state to deckDeny
through settleQuietly - silently - so the later write finds it decided and deckAsker, the
only thing that says why, is never invoked. The effect belongs to the TRANSITION into
deckDeny, not to one of the paths that can cause it; deckPermission should emit it once when
it settles to deny, whichever path settles it. The same gap has a second face: only the EMPTY
--stats screen says nothing is being saved, so a declined session that has looked three words
up shows ordinary figures with no hint they are session-only.
- BR-13 — not-addressed — Still a bare positional bool; six call sites read renderStats(s, now, true).

### Raised

- **BR-14** [Important] `policy-restated-not-derived` -raw prompts to create a deck, and tells piped scripts to use --here, though it writes nothing
  This is the 2nd finding in family policy-restated-not-derived (BR-12 is the 1st, and it is
  re-raised above). Do NOT fix this instance alone. The rule: a policy this codebase already
  owns in one place must be DERIVED at every new site - including the site that decides whether
  to build the permission at all. decideCapture (capture.go:26-33) is the single source of "this
  invocation writes nothing", and capture.go:192 already states that DEFINE_NO_CAPTURE and -raw
  are one class; main.go:806 restates half of it as `!opt.noCapture`, so -raw falls through into
  the gate. Measured at the pinned head with the real wiring: `define -raw` on a terminal in a
  fresh directory prints "... is not a deck yet. Create one here? [y/N]", answering y creates
  nothing at all, and the question consumes the session's first line of stdin as its answer;
  piped, every run prints "nothing will be saved (use --here to create one)" to stderr,
  recommending a flag that would not make -raw save anything. The Spec forbids exactly this
  ("a prompt for a command that would not have written anything is a false alarm"), and
  cmd/define/README.md:702 already documents -raw as never asking. Enumerate the members of the
  write-nothing class from decideCapture and gate on the class.
- **BR-15** [Important] `blocking-prompt-ignores-cancellation` Ctrl-C at the deck question does nothing - the prompt blocks with no cancellation arm
  main.go:401 installs signal.NotifyContext, and repl.go calls detachedInterrupts BEFORE
  repl.go:245's resolve(), so by the time the question is on screen SIGINT is intercepted
  process-wide and delivered to a context nobody is watching. deckAsker (deckperm.go:179) takes
  no context and blocks in readLineUnbuffered until a newline or EOF, so Ctrl-C at the prompt
  leaves the question up; the only exits are Enter, which silently declines, and Ctrl-D. The
  program otherwise owns what Ctrl-C means, which is why a reader will expect it to work here.
  Give the asker the cancellation channel and treat a cancelled context as an explicit,
  printed decline.
- **BR-16** [Minor] `artifact-claims-what-code-does-not` the plan's Core concepts never gained deckPolicy, and still lists IsDeck under Pure entities
  This is the 2nd finding in family artifact-claims-what-code-does-not (BR-4 was the 1st and is
  disposed addressed). Do NOT fix only this row. The rule: the Core-concepts table is the
  greppable contract a boundary review cross-checks, so any entity the implementation ADDS or
  RECLASSIFIES lands in the table with a "## Revisions" entry in the same commit that adds it.
  Instances: deckPolicy, withQuiet and settleQuietly - introduced in M2 after smoke testing -
  appear in no table; and while BR-4 moved the store.IsDeck table row to Integration points, the
  bullet describing it (plan:41) still sits under the "### Pure entities" heading, so the plan
  files it in both sections.

## Round 4 — 2026-09-11T00:28:01-07:00 (claude) — BLOCKED

### Disposed

- BR-8 — not-addressed — Still only m.Names and a hand-typed "< 15" floor; latent, since store.Store embeds nothing today.
- BR-9 — not-addressed — Still os.ReadDir, and deckPolicy runs twice on the ask path (quiet, then ask), so two ReadDirs there.
- BR-10 — addressed — Revert-verified: Forget->MkdirAll reddens TestNonCreatingMethodsCreateNothing/Forget; removing resolve() reddens both TestTheLineShellResolvesBeforeReading subtests. The raw-shell pty pin could not run here either (openpty EPERM, unsandboxed), so it has never been observed; run it with CONFORMANCE_STRICT=1 on a pty-capable machine before merge.
- BR-11 — addressed — Exact literal reverts of the fs.Usage prose and the README paragraph each redden TestEverySurfaceDescribingCaptureMentionsTheQuestion; deckPrompt is pinned against the README. The /lang surfaces belong to the new confirmation finding.
- BR-12 — addressed — Both explanation mutations (no sayWhy on settle; resolve skipping the quiet half) redden; real binary: redirected and piped lookups each explain once. The non-empty /stats "second face" is outside the Spec, which asked only for the empty message.
- BR-13 — not-addressed — Still renderStats(s, now, true) at every call site.
- BR-14 — addressed — Reverting the -raw branch reddens TestRawNeitherAsksNorAdvises; real binary: -raw piped and one-shot print no advice and create nothing. The derivation the finding asked for was not done; raised below as a Minor.
- BR-15 — addressed — Reverting readLineCancellable hangs TestInterruptAtTheQuestionDeclines 5s and reddens; reachable, because deckAsker holds run()'s NotifyContext ctx, which SIGINT still cancels. The post-cancel message is part of the new confirmation finding.
- BR-16 — not-addressed — Plan untouched since a30cb79: deckPolicy, deckReason, explainDenial, readLineCancellable, withQuiet, settleQuietly absent; IsDeck bullet still under Pure entities; plan's deckAsker lacks ctx and says four inputs (five now), and deckAsker's own doc comment says FOUR INPUTS and cites a nonexistent deckPermission.settle.

### Raised

- **BR-17** [Important] `guard-fails-open` 4th guard-fails-open: the issue's Done-when list is the enumeration never written, and its --stats row is pinned by nothing
  This is the 4th finding in family guard-fails-open. Earlier rounds applied the per-instance rule to components (store methods, buckets, loop shells), and those now hold. The enumeration never written is the issue's own Done-when list, where the feature's claims live. Measured at the pinned head: making printStats render the saving screen unconditionally (stats.go:87), or handing both doors a nil permission (stats.go:47 and :260), each leaves the whole cmd/define package green (110s). So the ticked row "--stats in an unsaved directory does not claim a word will join your deck", the exact defect smoke testing found, is pinned by nothing through production wiring. deckperm_e2e_test.go:174 asserts "Nothing yet", which both screens print, and TestStatsIsHonestWhereTheAnswerNeedsNobody computes its own copy of the formula (deckasker_test.go:269) instead of calling printStats. The rule: a ticked Done-when row is pinned only by a test that fails when the row is made false through production wiring. Write a Log table for all seven rows: row, test, wiring mutation, observed red. Sweep result: rows 1, 3, 5 and 6 have discriminating controls; row 7 has none; row 2's --play and /history half is pinned only structurally (non-nil gatedStore), with no test running either under a declined permission.
- **BR-18** [Important] `confirmation-not-derived-from-effect` /lang reports a switch and "now on the record" in a declined directory, because persistLang swallows the write its only caller reports
  The Spec names this class ("The message that would become a lie") and --stats was fixed; /lang was not. persistLang returns nil when declined (main.go:385-389), and TestPersistLangIsGated pins that nil, so lang_cmd.go:72 and :69 report success. Real binary, fresh directory, stdin redirected: define /lang es prints "now defining in es", exit 0, writes nothing, and the next define /lang says en. define /lang en prints "still defining in en, now on the record". A one-shot /lang has only a durable effect, so in the third state it is a no-op reporting a switch. The pre-#50 binary wrote lang.txt, and the no-directory case already refuses honestly (lang_cmd.go:57). TestLangAfterADeclineWritesNothing never reads stdout. Enumeration: lang_cmd.go:69 and :72 are reachable. The harvest, reflect, forget and play "saved/wrote/removed" messages all need a non-empty deck, which a declined one-shot cannot have. deckperm.go:263 prints "the lookup still works" on Ctrl-C, but in the REPL the same SIGINT fires detachedInterrupts' default cancel and the session exits with no lookup (scratch test: run returned 0). Docs asserting the old behaviour: cmd/define/README.md:649-651 and atlas/define.md:1128. Fix: persistLang reports a declined sentinel instead of nil; /lang says the language applies to this session only and was not saved; pin it on stdout. While in that README, :590 links the question to the Install anchor instead of the section that describes it.
- **BR-19** [Minor] `policy-restated-not-derived` deckPolicy re-tests opt.raw instead of consuming decideCapture, the single owner of "this invocation writes nothing"
  This is the 3rd finding in family policy-restated-not-derived (BR-12, BR-14). The rule: a decision this codebase already owns is consumed by calling its owner, never by re-testing the owner's inputs at a new site. BR-14's behaviour is fixed, but deckperm.go:210 re-tests opt.raw instead of asking decideCapture (capture.go:26), which capture.go:198 already consults for the same question. A third captureNothing member would reach the REPL's up-front question again. Enumerated: this is the only site in the diff restating decideCapture. main.go:806's noCapture check mirrors openStore's "anywhere to read at all" and is correct as is. One line: decideCapture(true, opt) == captureNothing, with reasonRaw renamed for the class.

## Round 5 — 2026-09-11T00:52:19-07:00 (claude) — passed

### Disposed

- BR-8 — not-addressed — gated_store_test.go:54-57 still reads only m.Names, and the floor at :75 is still a literal 15.
- BR-9 — not-addressed — isdeck.go:34 unchanged; and it is not once per process - settleQuietly does not memoize deckUndecided, so every gatedStore.reading() re-runs deckPolicy. Measured 3 ReadDir calls for one declined terminal one-shot.
- BR-13 — not-addressed — stats.go:101 still takes a bare positional bool, read as a literal at 9 call sites.
- BR-16 — not-addressed — Plan unchanged at the pinned head - no Revisions entry, no deckPolicy/deckReason/errDeckDeclined/withQuiet/settleQuietly rows, and the IsDeck bullet still sits under the Pure entities heading at plan:41.
- BR-17 — addressed — Both mutations verified red in a scratch worktree, and the Done-when pin table is written. The sibling door it did not sweep is raised separately.
- BR-18 — not-addressed — The /lang half is fixed and mutation-verified, but the enumeration this finding named is incomplete - the Ctrl-C message residual is measured below, and the two doc sites plus the :590 anchor are untouched.
- BR-19 — not-addressed — The code change landed and is correct, but reverting deckperm.go:232 to `if opt.raw` leaves the whole cmd/define package green on a full 110s run - nothing pins the derivation.

### Raised

- **BR-20** [Important] `guard-fails-open` 5th guard-fails-open: printStats has two doors and only the one the Done-when row named is pinned
  Measured at the pinned head: replacing stats.go:260 with printStats(c.deck, c.clock, "/stats", nil, c.stdout, c.stderr) leaves `go test ./cmd/define/` entirely green (110s), so the in-REPL door into the exact screen this issue exists for is pinned by nothing. Do NOT fix this instance. The rule: when a claim is about a shared renderer or decision, the pin count is DERIVED FROM THAT FUNCTION'S CALLER SET, not from the single entry point a Done-when row happens to name. printStats has exactly two callers (stats.go:47 and :260) and TestStatsInAnUnsavedDirectoryReportsEmpty drives only the first. Behaviour is currently correct - I ran /stats in a declined session and it renders the honest screen - so this is a coverage hole, not a live bug. The same caller-set enumeration applies to --play vs /play and to one-shot vs REPL /lang; write it once and derive the subtests.
- **BR-21** [Important] `readme-gate` 3rd readme-gate: the superseded-claims enumeration is hand-maintained, so the /lang persistence promise went stale
  Earlier rounds fixed --help, then "Every successful lookup". Do NOT fix these two sentences only. The rule: deckasker_test.go:347's `superseded` array IS the enumeration, and every behaviour change that narrows a documented promise adds its superseded sentence there in the same commit that narrows it. Prevalence - the array has 2 entries, both added reactively one per round; this round narrowed a third promise and added nothing. Now stale and unconditional: cmd/define/README.md:648-649 "`/lang es` switches, and the setting stays with the directory", :651 "It has to persist", and atlas/define.md:1128 "`/lang` reports when bare and persists when given" - all false in a declined directory, which is the state persistLang now reports with errDeckDeclined. Also cmd/define/README.md:590 links the question to [Install](#install) rather than to "## The directory is the deck, so it asks first", the section that actually describes it.

## Open findings

- **BR-8** [Minor] `derivation-under-derives` storeInterfaceMethods ignores embedded interfaces and the "< 15" floor is hand-bumped
- **BR-9** [Minor] `startup-path-cost` IsDeck uses os.ReadDir, which reads and sorts every entry in the working directory
- **BR-13** [Minor] `positional-bool-parameter` renderStats gained a bare positional bool, read at six call sites as a literal true
- **BR-16** [Minor] `artifact-claims-what-code-does-not` the plan's Core concepts never gained deckPolicy, and still lists IsDeck under Pure entities
- **BR-18** [Important] `confirmation-not-derived-from-effect` /lang reports a switch and "now on the record" in a declined directory, because persistLang swallows the write its only caller reports
- **BR-19** [Minor] `policy-restated-not-derived` deckPolicy re-tests opt.raw instead of consuming decideCapture, the single owner of "this invocation writes nothing"
- **BR-20** [Important] `guard-fails-open` 5th guard-fails-open: printStats has two doors and only the one the Done-when row named is pinned
- **BR-21** [Important] `readme-gate` 3rd readme-gate: the superseded-claims enumeration is hand-maintained, so the /lang persistence promise went stale
