---
gate: boundary-review
issue: 17
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-25T20:47:40-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
          detail: |-
            Third instance in this family (PQ-1 duplicated fold, PQ-4 two count sources, PQ-6 test restating D7),
            so the deliverable is the rule, not the site: a `## Decisions` entry is the sole statement of its fact
            and task bodies cite it (D1/D7) rather than restate it. D1's sentence needs the prune-then-drop-if-empty
            semantics that `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually specify,
            then the task bodies can cite D1 instead of paraphrasing it.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: single-source-of-truth
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-25T20:47:40-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: TestReflectIsIdempotent and TestReflectPreservesCorrections pass when runReflect fails and writes nothing
          detail: |-
            Neither test checks the exit code (reflect_run_test.go:79-95, :117-140). Verified in a
            scratch copy: injecting an early `return 1` for any run that finds an existing model
            leaves both PASSING, and breaking the model call outright (renaming the evidence_words
            json tag) leaves TestReflectIsIdempotent PASSING while its three siblings fail — the
            "idempotency" asserted is that two failed runs both wrote nothing. This is the exact
            vacuity the issue's Log records catching by hand. Assert code == 0 on every runReflect
            call, and assert the generated half actually changed between the two runs in the
            corrections test.
          family: vacuous-verification
          round: 2
        - id: BR-3
          severity: Important
          title: checkEvidence's usability arm covers domain name and directive but not level rationale, band, or share
          detail: |-
            reflect.go:161-191. Verified: checkEvidence with Level{Band:"x", Rationale:"placeholder",
            EvidenceWords:["certiorari"]} returns it KEPT with dropped=[] — the same stub shape the
            domain arm was written from a live failure to kill. An empty rationale renders
            "**C1** — " with a dangling em dash, and Share:42 renders "4200%", which the system
            prompt makes plausible since its own example reads "Law, 42%". Enumerate the fields a
            reader or authoring depends on and sweep them in one predicate, rather than adding a
            third hand-written arm when M2's weakness claims land.
          family: fix-the-class-not-the-instance
          round: 2
        - id: BR-4
          severity: Important
          title: The prompt golden renders a request with no schema while the wire request carries one
          detail: |-
            renderReflectPrompt returns an llm.Request without Schema, so the golden's schema section
            reads "(none)" (testdata/golden/reflect-prompt.txt:42); runReflect discards that Request
            except for .Prompt and rebuilds Name/System in llm.Task, where Run attaches the real
            schema. So reflect.go:93-95's claim that a learnerModel field change "shows up in the
            golden's diff" is false, no artifact covers SchemaFor[learnerModel] — the schema being
            exactly what produced this milestone's evidence/evidence_words bug — and
            TestReflectRequestNamesItsTask asserts a field production never sends. Set Schema in
            renderReflectPrompt and have runReflect derive Name/System/Prompt from that Request.
          family: golden-not-the-sent-request
          round: 2
        - id: BR-5
          severity: Important
          title: README's exit-code table is not swept for --reflect, which adds four producers of 1 and one of 2
          detail: |-
            README.md:184-186 introduces the table as "enumerated rather than sampled, because a list
            of examples goes stale the moment a new one is added and nothing says so". --reflect exits
            1 on a deck below the floor, on no model configured, on nothing surviving the check, and
            on store read/write failure; it exits 2 when combined with a word. None are listed. The
            same sweep misses README.md:122, where user-model.md is still annotated "optional, yours
            to write".
          family: docs-enumeration-not-swept
          round: 2
        - id: BR-6
          severity: Minor
          title: foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
          detail: |-
            reflect.go:59-89 never reads `now`; :57-58 explains it as what makes the window a table
            row rather than a timing test. Drop the parameter or use it (clamping To to now).
          family: comment-outruns-code
          round: 2
        - id: BR-7
          severity: Minor
          title: The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
          detail: |-
            reflect.go:225 prints len(ev.Words) — deck intersected with the found-lookup log — as
            "%d words in the deck", so a day-file that Events warned past silently shrinks the number
            a learner is told. reflect.go:168 renders "dropped level : cites nothing, none of which is
            in the deck" when the model made no level claim at all, which is the unactionable shape
            that message exists to avoid.
          family: message-states-what-it-measures
          round: 2
        - id: BR-8
          severity: Minor
          title: spliceCorrections discards an existing file with no out-of-fence marker without saying so
          detail: |-
            usermodel.go:120-122 returns `generated` whole when firstMarkerOutsideAFence finds nothing
            — reachable if a learner deletes the heading or leaves an unterminated fence above it.
            D3's contract does not promise preservation there, but the destruction should not be
            silent; one line to errOut is enough.
          family: silent-data-loss
          round: 2
        - id: BR-9
          severity: Minor
          title: modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
          detail: |-
            usermodel.go:19 defines `type modelMeta`, which the plan's own cited command
            (git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ') still
            reports. The plan's final Revisions entry names eight symbols and says "Reconciled to
            empty before this commit". lessons.md:941 is this rule.
          family: entity-table-completeness
          round: 2
        - id: BR-10
          severity: Minor
          title: --reflect combined with another mode silently honours one of them
          detail: |-
            Verified: `define --forget nonexistent --reflect` runs forget only and `define --llm-check
            --reflect` runs llm-check only, neither mentioning the ignored flag — while --reflect plus
            a word is correctly rejected at main.go:327-331 on the reasoning that two commands on one
            line must not be silently resolved. The llm-check/forget pair predates this diff; this
            change adds two more. A mode-count guard in the same switch covers all three.
          family: fix-the-class-not-the-instance
          round: 2
        - id: BR-11
          severity: Minor
          title: The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
          detail: |-
            usermodel.go:47-53 emits type/updated/window/generated_by; the issue Spec's shape and plan
            Task 3 ("Frontmatter exactly as the issue's Spec shows") both include `learner: <name>`.
            Omitting it is defensible — nothing supplies a name — but the plan's Revisions entry says
            "Two departures from the plan as written" and this is a third.
          family: spec-drift-undocumented
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-25T21:04:42-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: D1 and the Task 2 body are unchanged at HEAD; the plan still says "drops any claim citing a word the deck does not contain".
          round: 3
        - id: BR-2
          disposition: addressed
          note: Verified by injecting a failure into runReflect — both tests now redden via mustReflect; the "assert it changed" half was correctly rejected in a comment.
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'Verified by reverting both arms — all three TestCheckEvidenceHoldsTheLevelToTheSameBar subtests redden; residual: Band is checked non-empty but not against A2..C2.'
          round: 3
        - id: BR-4
          disposition: addressed
          note: Verified by renaming the evidence_words json tag — TestRenderReflectPrompt reddens; schema now derived from the same SchemaFor[learnerModel] llm.Run uses.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: The exit-code table is swept and correct, but README.md:122 — named in the same finding — still reads "optional, yours to write".
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: reflect.go:63 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: 'Both original sites unchanged, and the BR-3 fix added two more — citedOrNothing over a one-element slice can never return "nothing" (measured: "dropped level : no band or no rationale").'
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: usermodel.go:120-122 unchanged; a marker-less existing file is still discarded without a word.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: No modelMeta row was added; the plan's Revisions entry still claims the enumeration reconciled to empty.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: main.go still has no mode-count guard; --reflect with --forget or --llm-check silently honours one.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: Frontmatter still omits `learner:`, the Revisions entry still says "Two departures", and `M reviews` shipped as `M questions` (a fourth).
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: Model free text is rendered verbatim into a markdown table and a marker-delimited file
          detail: |-
            usermodel.go:56-70 embeds Band, Rationale, Name and Directive unmodified; checkEvidence
            only checks them for emptiness. Verified in a scratch copy: a directive containing a
            newline garbles the table row, and a field containing a line-start "## Corrections"
            permanently freezes everything below it — I ran run-1, a learner edit, then run-2 with a
            different answer and run-1's domain table survived into the new file and will never
            regenerate, silently breaking D3's "everything above the marker is replaced". Paragraph
            -length directives are exactly what reflect.go:255-257 raised MaxTokens for.
            FuzzSpliceCorrections cannot catch it: it asserts preservation below the marker, never
            regeneration above it. Collapse newlines and escape `|` in the four model-supplied
            fields at render time — collapsing newlines closes the injection outright, since every
            rendered line is prefixed by `**`, `Read off: ` or `| `.
          family: model-text-unconstrained-by-format
          round: 3
        - id: BR-13
          severity: Minor
          title: D5's dispatch site survives being moved before withStore with the whole suite green
          detail: |-
            Verified: moving `if *reflect` from main.go:352 to just before `d = d.withStore(...)`
            passes go test ./cmd/define/ entirely, while in production --reflect would then refuse
            every run with "define: no deck in this directory". The wider mis-siting is caught by
            TestReflectWithAWordIsAUsageError; the narrow one is not, and D5 was a blocking
            plan-gate finding (PQ-2). One happy-path run(ctx, []string{"--reflect"}, ...) test in a
            temp dir pins it.
          family: decision-unpinned-by-test
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-25T23:46:16-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: plan.md:37-43 unchanged; D1 still says "drops any claim citing a word the deck does not contain".
          round: 4
        - id: BR-5
          disposition: addressed
          note: Both sites swept — exit-code table's five new cells verified against the code, and README.md:122 now says "written by --reflect".
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: reflect.go:59 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: All four sites unchanged — reflect.go:240 counts deck-intersect-log as "in the deck", and :168/:190 pass one-element slices to citedOrNothing.
          round: 4
        - id: BR-8
          disposition: not-addressed
          note: usermodel.go:104-108 unchanged; a marker-less existing file is still discarded without a word.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: No modelMeta row, and 9bbfd4b added a ninth unrostered symbol (oneLine); the Revisions entry still claims the enumeration reconciled to empty.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: main.go still has no mode-count guard; --reflect with --forget or --llm-check silently honours one.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: Frontmatter still omits `learner:`, `M reviews` still ships as `M questions`, and the Revisions entry still says "Two departures".
          round: 4
        - id: BR-12
          disposition: addressed
          note: Code fix verified by full revert (test reddens) and by escape-pipes-only revert (also reddens); but its serious half is unpinned — see the new vacuous-verification finding.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: 'Re-verified: moving `if *reflect` before withStore leaves the whole suite green, and the resulting binary refuses every --reflect run with "no deck in this directory".'
          round: 4
      findings:
        - id: BR-14
          severity: Important
          title: The forged-marker assertion searches only the region before the first marker, where a forged marker cannot be
          detail: |-
            usermodel_test.go:253-259 computes i as the offset of the FIRST out-of-fence marker, then
            asserts no marker exists in got[:i] — marker-free by construction. Measured: with oneLine
            reverted the render holds 2 markers (offsets 336, 385) and the shipped expression yields
            j = -1. Decisive experiment: removing oneLine from the Level section only — reinstating
            BR-12's serious half exactly — leaves go test ./cmd/define/ fully green, that test
            included. So the milestone's worst defect is guarded only by the pipe-count arm. 2nd in
            this family after BR-2; the rule, not the site: an assertion that a violation is absent
            must run over the region the violation would occupy, and the positive control (construct
            the violation, confirm the assertion fires) is written before the assertion is believed.
            Add it to lessons.md's mechanical list beside the before/after-comparison entry.
          family: vacuous-verification
          round: 4
        - id: BR-15
          severity: Important
          title: EvidenceWords are model-supplied and are the two render sites oneLine was not applied to
          detail: |-
            usermodel.go:57 and :67 render EvidenceWords through joinWords, which does not sanitise.
            Verified end to end: deck {"hot dog"}, model cites "hot\ndog" — store.Key collapses it so
            checkEvidence KEEPS it verbatim (reflect.go:151-158 returns w, not the key) — and
            joinWords emits a backticked token containing a newline, splitting the "Read off:" line
            and breaking the table row mid-cell. `hot dog` is this repo's own canonical multi-word
            headword. 2nd in this family: do not patch the two call sites — enforce the rule that
            every model-supplied string reaching the file passes one sanitiser at one boundary (put
            oneLine inside joinWords, or normalise citations to store.Key in checkEvidence's
            supported). The sprinkle-at-call-sites shape is what let two of six sites be missed
            (ARCH-DRY). Same site: oneLine does not strip backticks, and
            reflect_conformance_test.go:145-149 treats every backticked token above the marker as an
            evidence claim, so a rationale quoting a word would fail the live check spuriously.
          family: model-text-unconstrained-by-format
          round: 4
        - id: BR-16
          severity: Important
          title: atlas/define.md was not swept for the round-2 rule and still asserts a property this window disproved
          detail: |-
            Commit 9bbfd4b touched README, code, tests, the issue and two ledgers — not the atlas. The
            neutralisation rule and its learner/model asymmetry, the one new architectural idea this
            round, appear nowhere in atlas/define.md. Worse, atlas/define.md:697-699 still tells the
            reader the fuzz property "asserts the one thing that must hold", which the issue's own Log
            now records as false. 3rd in this family — round 3 stated the rule as "grep for the surface
            name across README.md AND atlas/ before closing", and this round swept README only. Apply
            the stated rule as a closing step rather than patching the paragraph, and replace "the one
            thing that must hold" with the two-directions invariant.
          family: docs-enumeration-not-swept
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-08-26T09:13:34-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: Verified by the decisive experiment — removing Level sanitisation alone now reddens the test via countMarkers; the requested lessons.md mechanical-list line is still absent, but the adjacent class entry was written.
          round: 5
        - id: BR-15
          disposition: addressed
          note: Structural fix verified by two independent mutations (evidence-word arm, pipe escape); residual named in the finding — oneLine still does not strip backticks, so the live check can fail spuriously on a quoted non-deck word.
          round: 5
        - id: BR-16
          disposition: addressed
          note: atlas/define.md:693-701 now states the two-directions invariant and the learner/model asymmetry; the disproved "one thing that must hold" claim is gone.
          round: 5
        - id: BR-1
          disposition: not-addressed
          note: plan.md:37-43 unchanged; D1 still says "drops any claim citing a word the deck does not contain".
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: reflect.go:59 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
          round: 5
        - id: BR-7
          disposition: not-addressed
          note: Half fixed — cite() means an empty band now prints "level nothing" — but reflect.go:260 and :329 still call deck-intersect-log "words in the deck".
          round: 5
        - id: BR-8
          disposition: not-addressed
          note: usermodel.go:114-118 unchanged; a marker-less existing file is still discarded without a word.
          round: 5
        - id: BR-9
          disposition: not-addressed
          note: Now seven rows short (modelMeta, oneLine, oneLineAll, sanitiseModel, sanitiseMeta, cite, citeAll) and one row stale (citedOrNothing, deleted in 692ec09); the Revisions entry still claims closure.
          round: 5
        - id: BR-10
          disposition: not-addressed
          note: main.go still has no mode-count guard; --llm-check short-circuits at :308 and --forget dispatches at :347, both before `if *reflect` at :352.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: Frontmatter still omits `learner:`, `M reviews` still ships as `M questions`, and the Revisions entry still says "Two departures".
          round: 5
        - id: BR-13
          disposition: not-addressed
          note: 'Re-verified on the current code: moving `if *reflect` before withStore leaves the suite green, and the built binary then refuses every --reflect run with "no deck in this directory".'
          round: 5
      findings:
        - id: BR-17
          severity: Important
          title: Four neutralisation sites have no positive control — deleting sanitiseMeta's body leaves the whole suite green
          detail: |-
            3rd in this family after BR-12 and BR-15, so the deliverable is the rule, not these four sites.
            Measured against the full ./cmd/define/ suite, each mutation applied in a scratch copy: deleting
            sanitiseMeta's body (usermodel.go:230-233) — green; unsanitising cite(d.Name) at reflect.go:210
            and :216 and cite(m.Level.Band) at :188 — green. TestDroppedClaimDiagnosticsCannotForgeALine
            reaches only two of checkEvidence's five diagnostic arms, and sampleMeta().Model carries no
            injection, so the frontmatter sanitiser was never exercised. The rule lessons.md:993 already
            states for the file and the code contradicts for the diagnostics: a sink that renders untrusted
            text has ONE neutralising constructor plus ONE positive-control test that constructs the
            violation and confirms the assertion fires. checkEvidence sanitises at five call sites, which is
            the list-that-drifts renderUserModel was just refactored away from (ARCH-DRY); make `dropped`
            carry a struct and give it one formatter that calls oneLine on the subject, so no arm can forget
            and one injection test covers all five by construction. Same site: sanitiseModel's comment
            (usermodel.go:210-211) claims adding a learnerModel field "makes this function fail to compile" —
            it does not, and M2's Weaknesses would silently bypass it.
          family: model-text-unconstrained-by-format
          round: 5
      boundary: M1
      blocked: false
    - "n": 6
      timestamp: "2026-08-27T22:29:11-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: plan.md D1 unchanged — still "drops any claim citing a word the deck does not contain"; the code prunes and drops only when nothing survives.
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: reflect.go:59 still takes `now`; lines 60-88 never read it and the doc comment at :57-58 still justifies it as what makes the window a table row.
          round: 6
        - id: BR-7
          disposition: not-addressed
          note: 'reflect.go:260 and :329 still call deck-intersect-log "words in the deck"; reproduced live — 13 word files and no events prints "define: 0 words in the deck".'
          round: 6
        - id: BR-8
          disposition: not-addressed
          note: usermodel.go:114-118 unchanged; an existing file with no out-of-fence marker is still returned over without a word to errOut.
          round: 6
        - id: BR-9
          disposition: not-addressed
          note: Re-ran the plan's own cited command — seven symbols still have no row (modelMeta, cite, citeAll, oneLine, oneLineAll, sanitiseModel, sanitiseMeta) and citedOrNothing is stale.
          round: 6
        - id: BR-10
          disposition: not-addressed
          note: Re-verified with the built binary — `--forget nonexistent --reflect` runs forget only, `--llm-check --reflect` runs llm-check only, neither mentioning the ignored flag; --play is now a fourth mode with no count guard.
          round: 6
        - id: BR-11
          disposition: not-addressed
          note: usermodel.go:49-55 still omits `learner:` and still renders "M questions" where the Spec shows "M reviews"; the Revisions entry still says "Two departures".
          round: 6
        - id: BR-13
          disposition: not-addressed
          note: Re-verified by mutation on current code — moving `if *reflect` above withStore leaves the whole ./cmd/define/ suite green while the binary would refuse every run.
          round: 6
        - id: BR-17
          disposition: not-addressed
          note: Render half landed and is mutation-verified (deleting sanitiseMeta's body now reddens); the diagnostic half did not — unsanitising cite() at reflect.go:188, :210 and :216 leaves the full suite GREEN, and the one-formatter fix was not made.
          round: 6
      findings:
        - id: BR-18
          severity: Important
          title: The M2 descope is recorded in two issue files and in none of the four places the project still calls it planned work
          detail: |-
            This is the 3rd finding in family `docs-enumeration-not-swept`, so the deliverable is the
            enumeration, not the four lines. Rule: at a boundary that changes a fact, grep for every
            artifact that restates it and reconcile each hit before the verdict is recorded. Measured for
            this close: `grep -rn '#17|tools#17' workshop/ atlas/ README.md` returns 24 hits;
            workshop/projects/define-learn.md:171-173, :194, :383-386, :563 and :613 all still describe
            #17 M2 as planned or blocked-on-#6 work, and :385's "blocked — needs review events from
            tools#6" is now false. The remaining hits are still true. Note that sdlc close auto-ticks
            referencing project rows, so :194 risks being marked delivered.
          family: docs-enumeration-not-swept
          round: 6
        - id: BR-19
          severity: Important
          title: M1 Done-when row 5 is ticked and the held-out property is a t.Logf, not an assertion
          detail: |-
            This is the 3rd finding in family `vacuous-verification` after BR-2 and BR-14, so the
            deliverable is the rule: a Done-when row is ticked only when a named test FAILS if the
            property is removed. I ran that enumeration over M1's five rows — rows 1, 2, 3 are pinned by
            named tests, row 4 is pinned (mutation-verified: setting c.UserModel aside in ask.go:270
            reddens TestAskStreamsAnAnswerWithTheDirectoryAsContext), and row 5 is the only unpinned one.
            In reflect_conformance_test.go the held-out word is in the deck, in the prompt AND in the
            representation loop at :95, and the only statement about it is a t.Logf at :125. Cheapest
            real fix: count `seen` over c.words only.
          family: vacuous-verification
          round: 6
        - id: BR-20
          severity: Minor
          title: The M2 Plan row is ticked while its own text says "not delivered", and the M2 Done-when rows stay unticked
          detail: |-
            000017-user-model.md:212 marks `- [x] M2 — weaknesses. DESCOPED at close, not delivered.`
            while :98-105 leave the M2 Done-when rows `- [ ]`, so one file answers "was M2 done" both
            ways and a `grep '\- \[ \]'` over the tracker no longer means what it did. The plan-unchecked
            gate has `--no-plan-check` for precisely this case — leave the box unticked and put the
            descope reasoning in `--verified`.
          family: gate-satisfied-not-met
          round: 6
      blocked: true
---

# Gate ledger — tools#17 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-25T20:47:40-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `single-source-of-truth` D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
  Third instance in this family (PQ-1 duplicated fold, PQ-4 two count sources, PQ-6 test restating D7),
  so the deliverable is the rule, not the site: a `## Decisions` entry is the sole statement of its fact
  and task bodies cite it (D1/D7) rather than restate it. D1's sentence needs the prune-then-drop-if-empty
  semantics that `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually specify,
  then the task bodies can cite D1 instead of paraphrasing it.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-08-25T20:47:40-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `vacuous-verification` TestReflectIsIdempotent and TestReflectPreservesCorrections pass when runReflect fails and writes nothing
  Neither test checks the exit code (reflect_run_test.go:79-95, :117-140). Verified in a
  scratch copy: injecting an early `return 1` for any run that finds an existing model
  leaves both PASSING, and breaking the model call outright (renaming the evidence_words
  json tag) leaves TestReflectIsIdempotent PASSING while its three siblings fail — the
  "idempotency" asserted is that two failed runs both wrote nothing. This is the exact
  vacuity the issue's Log records catching by hand. Assert code == 0 on every runReflect
  call, and assert the generated half actually changed between the two runs in the
  corrections test.
- **BR-3** [Important] `fix-the-class-not-the-instance` checkEvidence's usability arm covers domain name and directive but not level rationale, band, or share
  reflect.go:161-191. Verified: checkEvidence with Level{Band:"x", Rationale:"placeholder",
  EvidenceWords:["certiorari"]} returns it KEPT with dropped=[] — the same stub shape the
  domain arm was written from a live failure to kill. An empty rationale renders
  "**C1** — " with a dangling em dash, and Share:42 renders "4200%", which the system
  prompt makes plausible since its own example reads "Law, 42%". Enumerate the fields a
  reader or authoring depends on and sweep them in one predicate, rather than adding a
  third hand-written arm when M2's weakness claims land.
- **BR-4** [Important] `golden-not-the-sent-request` The prompt golden renders a request with no schema while the wire request carries one
  renderReflectPrompt returns an llm.Request without Schema, so the golden's schema section
  reads "(none)" (testdata/golden/reflect-prompt.txt:42); runReflect discards that Request
  except for .Prompt and rebuilds Name/System in llm.Task, where Run attaches the real
  schema. So reflect.go:93-95's claim that a learnerModel field change "shows up in the
  golden's diff" is false, no artifact covers SchemaFor[learnerModel] — the schema being
  exactly what produced this milestone's evidence/evidence_words bug — and
  TestReflectRequestNamesItsTask asserts a field production never sends. Set Schema in
  renderReflectPrompt and have runReflect derive Name/System/Prompt from that Request.
- **BR-5** [Important] `docs-enumeration-not-swept` README's exit-code table is not swept for --reflect, which adds four producers of 1 and one of 2
  README.md:184-186 introduces the table as "enumerated rather than sampled, because a list
  of examples goes stale the moment a new one is added and nothing says so". --reflect exits
  1 on a deck below the floor, on no model configured, on nothing surviving the check, and
  on store read/write failure; it exits 2 when combined with a word. None are listed. The
  same sweep misses README.md:122, where user-model.md is still annotated "optional, yours
  to write".
- **BR-6** [Minor] `comment-outruns-code` foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
  reflect.go:59-89 never reads `now`; :57-58 explains it as what makes the window a table
  row rather than a timing test. Drop the parameter or use it (clamping To to now).
- **BR-7** [Minor] `message-states-what-it-measures` The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
  reflect.go:225 prints len(ev.Words) — deck intersected with the found-lookup log — as
  "%d words in the deck", so a day-file that Events warned past silently shrinks the number
  a learner is told. reflect.go:168 renders "dropped level : cites nothing, none of which is
  in the deck" when the model made no level claim at all, which is the unactionable shape
  that message exists to avoid.
- **BR-8** [Minor] `silent-data-loss` spliceCorrections discards an existing file with no out-of-fence marker without saying so
  usermodel.go:120-122 returns `generated` whole when firstMarkerOutsideAFence finds nothing
  — reachable if a learner deletes the heading or leaves an unterminated fence above it.
  D3's contract does not promise preservation there, but the destruction should not be
  silent; one line to errOut is enough.
- **BR-9** [Minor] `entity-table-completeness` modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
  usermodel.go:19 defines `type modelMeta`, which the plan's own cited command
  (git diff <boundary> -- 'cmd/**/*.go' ':!*_test.go' | grep -E '^\+(func|type) ') still
  reports. The plan's final Revisions entry names eight symbols and says "Reconciled to
  empty before this commit". lessons.md:941 is this rule.
- **BR-10** [Minor] `fix-the-class-not-the-instance` --reflect combined with another mode silently honours one of them
  Verified: `define --forget nonexistent --reflect` runs forget only and `define --llm-check
  --reflect` runs llm-check only, neither mentioning the ignored flag — while --reflect plus
  a word is correctly rejected at main.go:327-331 on the reasoning that two commands on one
  line must not be silently resolved. The llm-check/forget pair predates this diff; this
  change adds two more. A mode-count guard in the same switch covers all three.
- **BR-11** [Minor] `spec-drift-undocumented` The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
  usermodel.go:47-53 emits type/updated/window/generated_by; the issue Spec's shape and plan
  Task 3 ("Frontmatter exactly as the issue's Spec shows") both include `learner: <name>`.
  Omitting it is defensible — nothing supplies a name — but the plan's Revisions entry says
  "Two departures from the plan as written" and this is a third.

## Round 3 — 2026-08-25T21:04:42-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — D1 and the Task 2 body are unchanged at HEAD; the plan still says "drops any claim citing a word the deck does not contain".
- BR-2 — addressed — Verified by injecting a failure into runReflect — both tests now redden via mustReflect; the "assert it changed" half was correctly rejected in a comment.
- BR-3 — addressed — Verified by reverting both arms — all three TestCheckEvidenceHoldsTheLevelToTheSameBar subtests redden; residual: Band is checked non-empty but not against A2..C2.
- BR-4 — addressed — Verified by renaming the evidence_words json tag — TestRenderReflectPrompt reddens; schema now derived from the same SchemaFor[learnerModel] llm.Run uses.
- BR-5 — not-addressed — The exit-code table is swept and correct, but README.md:122 — named in the same finding — still reads "optional, yours to write".
- BR-6 — not-addressed — reflect.go:63 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
- BR-7 — not-addressed — Both original sites unchanged, and the BR-3 fix added two more — citedOrNothing over a one-element slice can never return "nothing" (measured: "dropped level : no band or no rationale").
- BR-8 — not-addressed — usermodel.go:120-122 unchanged; a marker-less existing file is still discarded without a word.
- BR-9 — not-addressed — No modelMeta row was added; the plan's Revisions entry still claims the enumeration reconciled to empty.
- BR-10 — not-addressed — main.go still has no mode-count guard; --reflect with --forget or --llm-check silently honours one.
- BR-11 — not-addressed — Frontmatter still omits `learner:`, the Revisions entry still says "Two departures", and `M reviews` shipped as `M questions` (a fourth).

### Raised

- **BR-12** [Important] `model-text-unconstrained-by-format` Model free text is rendered verbatim into a markdown table and a marker-delimited file
  usermodel.go:56-70 embeds Band, Rationale, Name and Directive unmodified; checkEvidence
  only checks them for emptiness. Verified in a scratch copy: a directive containing a
  newline garbles the table row, and a field containing a line-start "## Corrections"
  permanently freezes everything below it — I ran run-1, a learner edit, then run-2 with a
  different answer and run-1's domain table survived into the new file and will never
  regenerate, silently breaking D3's "everything above the marker is replaced". Paragraph
  -length directives are exactly what reflect.go:255-257 raised MaxTokens for.
  FuzzSpliceCorrections cannot catch it: it asserts preservation below the marker, never
  regeneration above it. Collapse newlines and escape `|` in the four model-supplied
  fields at render time — collapsing newlines closes the injection outright, since every
  rendered line is prefixed by `**`, `Read off: ` or `| `.
- **BR-13** [Minor] `decision-unpinned-by-test` D5's dispatch site survives being moved before withStore with the whole suite green
  Verified: moving `if *reflect` from main.go:352 to just before `d = d.withStore(...)`
  passes go test ./cmd/define/ entirely, while in production --reflect would then refuse
  every run with "define: no deck in this directory". The wider mis-siting is caught by
  TestReflectWithAWordIsAUsageError; the narrow one is not, and D5 was a blocking
  plan-gate finding (PQ-2). One happy-path run(ctx, []string{"--reflect"}, ...) test in a
  temp dir pins it.

## Round 4 — 2026-08-25T23:46:16-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — plan.md:37-43 unchanged; D1 still says "drops any claim citing a word the deck does not contain".
- BR-5 — addressed — Both sites swept — exit-code table's five new cells verified against the code, and README.md:122 now says "written by --reflect".
- BR-6 — not-addressed — reflect.go:59 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
- BR-7 — not-addressed — All four sites unchanged — reflect.go:240 counts deck-intersect-log as "in the deck", and :168/:190 pass one-element slices to citedOrNothing.
- BR-8 — not-addressed — usermodel.go:104-108 unchanged; a marker-less existing file is still discarded without a word.
- BR-9 — not-addressed — No modelMeta row, and 9bbfd4b added a ninth unrostered symbol (oneLine); the Revisions entry still claims the enumeration reconciled to empty.
- BR-10 — not-addressed — main.go still has no mode-count guard; --reflect with --forget or --llm-check silently honours one.
- BR-11 — not-addressed — Frontmatter still omits `learner:`, `M reviews` still ships as `M questions`, and the Revisions entry still says "Two departures".
- BR-12 — addressed — Code fix verified by full revert (test reddens) and by escape-pipes-only revert (also reddens); but its serious half is unpinned — see the new vacuous-verification finding.
- BR-13 — not-addressed — Re-verified: moving `if *reflect` before withStore leaves the whole suite green, and the resulting binary refuses every --reflect run with "no deck in this directory".

### Raised

- **BR-14** [Important] `vacuous-verification` The forged-marker assertion searches only the region before the first marker, where a forged marker cannot be
  usermodel_test.go:253-259 computes i as the offset of the FIRST out-of-fence marker, then
  asserts no marker exists in got[:i] — marker-free by construction. Measured: with oneLine
  reverted the render holds 2 markers (offsets 336, 385) and the shipped expression yields
  j = -1. Decisive experiment: removing oneLine from the Level section only — reinstating
  BR-12's serious half exactly — leaves go test ./cmd/define/ fully green, that test
  included. So the milestone's worst defect is guarded only by the pipe-count arm. 2nd in
  this family after BR-2; the rule, not the site: an assertion that a violation is absent
  must run over the region the violation would occupy, and the positive control (construct
  the violation, confirm the assertion fires) is written before the assertion is believed.
  Add it to lessons.md's mechanical list beside the before/after-comparison entry.
- **BR-15** [Important] `model-text-unconstrained-by-format` EvidenceWords are model-supplied and are the two render sites oneLine was not applied to
  usermodel.go:57 and :67 render EvidenceWords through joinWords, which does not sanitise.
  Verified end to end: deck {"hot dog"}, model cites "hot\ndog" — store.Key collapses it so
  checkEvidence KEEPS it verbatim (reflect.go:151-158 returns w, not the key) — and
  joinWords emits a backticked token containing a newline, splitting the "Read off:" line
  and breaking the table row mid-cell. `hot dog` is this repo's own canonical multi-word
  headword. 2nd in this family: do not patch the two call sites — enforce the rule that
  every model-supplied string reaching the file passes one sanitiser at one boundary (put
  oneLine inside joinWords, or normalise citations to store.Key in checkEvidence's
  supported). The sprinkle-at-call-sites shape is what let two of six sites be missed
  (ARCH-DRY). Same site: oneLine does not strip backticks, and
  reflect_conformance_test.go:145-149 treats every backticked token above the marker as an
  evidence claim, so a rationale quoting a word would fail the live check spuriously.
- **BR-16** [Important] `docs-enumeration-not-swept` atlas/define.md was not swept for the round-2 rule and still asserts a property this window disproved
  Commit 9bbfd4b touched README, code, tests, the issue and two ledgers — not the atlas. The
  neutralisation rule and its learner/model asymmetry, the one new architectural idea this
  round, appear nowhere in atlas/define.md. Worse, atlas/define.md:697-699 still tells the
  reader the fuzz property "asserts the one thing that must hold", which the issue's own Log
  now records as false. 3rd in this family — round 3 stated the rule as "grep for the surface
  name across README.md AND atlas/ before closing", and this round swept README only. Apply
  the stated rule as a closing step rather than patching the paragraph, and replace "the one
  thing that must hold" with the two-directions invariant.

## Round 5 — 2026-08-26T09:13:34-07:00 (claude) — passed

### Disposed

- BR-14 — addressed — Verified by the decisive experiment — removing Level sanitisation alone now reddens the test via countMarkers; the requested lessons.md mechanical-list line is still absent, but the adjacent class entry was written.
- BR-15 — addressed — Structural fix verified by two independent mutations (evidence-word arm, pipe escape); residual named in the finding — oneLine still does not strip backticks, so the live check can fail spuriously on a quoted non-deck word.
- BR-16 — addressed — atlas/define.md:693-701 now states the two-directions invariant and the learner/model asymmetry; the disproved "one thing that must hold" claim is gone.
- BR-1 — not-addressed — plan.md:37-43 unchanged; D1 still says "drops any claim citing a word the deck does not contain".
- BR-6 — not-addressed — reflect.go:59 still takes `now` and never reads it; the doc comment at :57-58 still justifies it.
- BR-7 — not-addressed — Half fixed — cite() means an empty band now prints "level nothing" — but reflect.go:260 and :329 still call deck-intersect-log "words in the deck".
- BR-8 — not-addressed — usermodel.go:114-118 unchanged; a marker-less existing file is still discarded without a word.
- BR-9 — not-addressed — Now seven rows short (modelMeta, oneLine, oneLineAll, sanitiseModel, sanitiseMeta, cite, citeAll) and one row stale (citedOrNothing, deleted in 692ec09); the Revisions entry still claims closure.
- BR-10 — not-addressed — main.go still has no mode-count guard; --llm-check short-circuits at :308 and --forget dispatches at :347, both before `if *reflect` at :352.
- BR-11 — not-addressed — Frontmatter still omits `learner:`, `M reviews` still ships as `M questions`, and the Revisions entry still says "Two departures".
- BR-13 — not-addressed — Re-verified on the current code: moving `if *reflect` before withStore leaves the suite green, and the built binary then refuses every --reflect run with "no deck in this directory".

### Raised

- **BR-17** [Important] `model-text-unconstrained-by-format` Four neutralisation sites have no positive control — deleting sanitiseMeta's body leaves the whole suite green
  3rd in this family after BR-12 and BR-15, so the deliverable is the rule, not these four sites.
  Measured against the full ./cmd/define/ suite, each mutation applied in a scratch copy: deleting
  sanitiseMeta's body (usermodel.go:230-233) — green; unsanitising cite(d.Name) at reflect.go:210
  and :216 and cite(m.Level.Band) at :188 — green. TestDroppedClaimDiagnosticsCannotForgeALine
  reaches only two of checkEvidence's five diagnostic arms, and sampleMeta().Model carries no
  injection, so the frontmatter sanitiser was never exercised. The rule lessons.md:993 already
  states for the file and the code contradicts for the diagnostics: a sink that renders untrusted
  text has ONE neutralising constructor plus ONE positive-control test that constructs the
  violation and confirms the assertion fires. checkEvidence sanitises at five call sites, which is
  the list-that-drifts renderUserModel was just refactored away from (ARCH-DRY); make `dropped`
  carry a struct and give it one formatter that calls oneLine on the subject, so no arm can forget
  and one injection test covers all five by construction. Same site: sanitiseModel's comment
  (usermodel.go:210-211) claims adding a learnerModel field "makes this function fail to compile" —
  it does not, and M2's Weaknesses would silently bypass it.

## Round 6 — 2026-08-27T22:29:11-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — plan.md D1 unchanged — still "drops any claim citing a word the deck does not contain"; the code prunes and drops only when nothing survives.
- BR-6 — not-addressed — reflect.go:59 still takes `now`; lines 60-88 never read it and the doc comment at :57-58 still justifies it as what makes the window a table row.
- BR-7 — not-addressed — reflect.go:260 and :329 still call deck-intersect-log "words in the deck"; reproduced live — 13 word files and no events prints "define: 0 words in the deck".
- BR-8 — not-addressed — usermodel.go:114-118 unchanged; an existing file with no out-of-fence marker is still returned over without a word to errOut.
- BR-9 — not-addressed — Re-ran the plan's own cited command — seven symbols still have no row (modelMeta, cite, citeAll, oneLine, oneLineAll, sanitiseModel, sanitiseMeta) and citedOrNothing is stale.
- BR-10 — not-addressed — Re-verified with the built binary — `--forget nonexistent --reflect` runs forget only, `--llm-check --reflect` runs llm-check only, neither mentioning the ignored flag; --play is now a fourth mode with no count guard.
- BR-11 — not-addressed — usermodel.go:49-55 still omits `learner:` and still renders "M questions" where the Spec shows "M reviews"; the Revisions entry still says "Two departures".
- BR-13 — not-addressed — Re-verified by mutation on current code — moving `if *reflect` above withStore leaves the whole ./cmd/define/ suite green while the binary would refuse every run.
- BR-17 — not-addressed — Render half landed and is mutation-verified (deleting sanitiseMeta's body now reddens); the diagnostic half did not — unsanitising cite() at reflect.go:188, :210 and :216 leaves the full suite GREEN, and the one-formatter fix was not made.

### Raised

- **BR-18** [Important] `docs-enumeration-not-swept` The M2 descope is recorded in two issue files and in none of the four places the project still calls it planned work
  This is the 3rd finding in family `docs-enumeration-not-swept`, so the deliverable is the
  enumeration, not the four lines. Rule: at a boundary that changes a fact, grep for every
  artifact that restates it and reconcile each hit before the verdict is recorded. Measured for
  this close: `grep -rn '#17|tools#17' workshop/ atlas/ README.md` returns 24 hits;
  workshop/projects/define-learn.md:171-173, :194, :383-386, :563 and :613 all still describe
  #17 M2 as planned or blocked-on-#6 work, and :385's "blocked — needs review events from
  tools#6" is now false. The remaining hits are still true. Note that sdlc close auto-ticks
  referencing project rows, so :194 risks being marked delivered.
- **BR-19** [Important] `vacuous-verification` M1 Done-when row 5 is ticked and the held-out property is a t.Logf, not an assertion
  This is the 3rd finding in family `vacuous-verification` after BR-2 and BR-14, so the
  deliverable is the rule: a Done-when row is ticked only when a named test FAILS if the
  property is removed. I ran that enumeration over M1's five rows — rows 1, 2, 3 are pinned by
  named tests, row 4 is pinned (mutation-verified: setting c.UserModel aside in ask.go:270
  reddens TestAskStreamsAnAnswerWithTheDirectoryAsContext), and row 5 is the only unpinned one.
  In reflect_conformance_test.go the held-out word is in the deck, in the prompt AND in the
  representation loop at :95, and the only statement about it is a t.Logf at :125. Cheapest
  real fix: count `seen` over c.words only.
- **BR-20** [Minor] `gate-satisfied-not-met` The M2 Plan row is ticked while its own text says "not delivered", and the M2 Done-when rows stay unticked
  000017-user-model.md:212 marks `- [x] M2 — weaknesses. DESCOPED at close, not delivered.`
  while :98-105 leave the M2 Done-when rows `- [ ]`, so one file answers "was M2 done" both
  ways and a `grep '\- \[ \]'` over the tracker no longer means what it did. The plan-unchecked
  gate has `--no-plan-check` for precisely this case — leave the box unticked and put the
  descope reasoning in `--verified`.

## Open findings

- **BR-1** [Minor] `single-source-of-truth` D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
- **BR-6** [Minor] `comment-outruns-code` foldLookups takes a `now` parameter it never uses, and the doc comment justifies it
- **BR-7** [Minor] `message-states-what-it-measures` The floor message says "words in the deck" but counts evidence words, and a dropped empty level prints a blank band
- **BR-8** [Minor] `silent-data-loss` spliceCorrections discards an existing file with no out-of-fence marker without saying so
- **BR-9** [Minor] `entity-table-completeness` modelMeta has no Core-concepts row, and the plan claims the enumeration reconciled to empty
- **BR-10** [Minor] `fix-the-class-not-the-instance` --reflect combined with another mode silently honours one of them
- **BR-11** [Minor] `spec-drift-undocumented` The rendered frontmatter omits the Spec's `learner:` field and the Revisions entry does not say so
- **BR-13** [Minor] `decision-unpinned-by-test` D5's dispatch site survives being moved before withStore with the whole suite green
- **BR-17** [Important] `model-text-unconstrained-by-format` Four neutralisation sites have no positive control — deleting sanitiseMeta's body leaves the whole suite green
- **BR-18** [Important] `docs-enumeration-not-swept` The M2 descope is recorded in two issue files and in none of the four places the project still calls it planned work
- **BR-19** [Important] `vacuous-verification` M1 Done-when row 5 is ticked and the held-out property is a t.Logf, not an assertion
- **BR-20** [Minor] `gate-satisfied-not-met` The M2 Plan row is ticked while its own text says "not delivered", and the M2 Done-when rows stay unticked
