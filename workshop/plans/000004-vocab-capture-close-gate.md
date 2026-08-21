---
gate: boundary-review
issue: 4
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-21T11:08:20-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: '"three call sites" is two - defineOnce serves both the one-shot and line paths'
          detail: |-
            main.go:144 and repl.go:144 both call defineOnce, so widening adds one
            invocation site, not two. The design is still right; the count is not.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: unbacked-existing-behavior-claim
          round: 1
        - id: BR-2
          severity: Important
          title: storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
          detail: |-
            This is the 2nd finding in family extraction-strands-behavior (prevalence 2: PQ-4's
            warn-once, now the durability contract itself). The rule: an extraction must enumerate
            every obligation the source component is contracted for — the tests that pin it, the
            warn-once, the durability guarantee — and say where each lands. Plan line 77 says
            storeHistory becomes a pure reader; line 109 says it delegates; the issue checkbox
            agrees with line 109. Under the reader reading, history_store_test.go:17-33, :37-55,
            :86-99, :155-166 and :133-150 all fail, contradicting "green without edits … do not
            edit them to fit".
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: extraction-strands-behavior
          round: 1
        - id: BR-3
          severity: Important
          title: Task 2 Step 3 still names defineOnce as the capture site, contradicting Chunk 1's lookupAndRender
          detail: |-
            This is the 3rd finding in family capture-arity-invariant (prevalence 3: PQ-7 double-capture,
            PQ-9 zero-capture-on-raw, now the executable step disagreeing with the prose). Do not patch
            line 181. The rule, which also covers PQ-6 and PQ-10: each design fact gets exactly ONE
            normative statement in the plan and every other mention references it rather than restating
            it (ARCH-DRY applied to the artifact). "Where capture happens" is currently stated at lines
            7, 24, 72, 116 and 181; three rounds have each fixed one copy and left the rest, which is
            why the family keeps recurring. Rewrite 7, 24, 116, 181 to point at the Chunk 1 statement,
            and give storeHistory's fate the same single-home treatment across lines 7, 24, 133 and the
            issue checkbox.
            (carried from plan-quality PQ-11, deferred to the boundary review)
          family: capture-arity-invariant
          round: 1
        - id: BR-4
          severity: Minor
          title: The plan calls d.capture but never says deps gains the field, nor what a deps literal without it does
          detail: |-
            deps is the injected IO seam (main.go:20) and both test rigs build it as a literal
            (main_test.go:14, main_test.go:41), so an unguarded d.capture.Capture panics the suite.
            The codebase already has the idiom for this at replraw.go:63, where a nil history seam
            falls back to memHistory. State whether capture takes a nil-fallback null object or every
            rig must supply one.
            (carried from plan-quality PQ-12, deferred to the boundary review)
          family: unstated-seam-default
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-08-21T11:08:20-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: The capture-arity test counts Capturer calls, not store writes, so the double-count it names passes
          detail: |-
            cmd/define/capture_test.go:86 injects countingCapturer at the seam, above both decideCapture
            and the writes. Verified by revert: restoring the AppendEvent/Upsert pair in storeHistory.Add
            leaves the entire suite green while the deck records Lookups:2 and two events per lookup. The
            plan's Task 2 Step 1 specified a counting STORE. atlas/define.md:285 claims the invariant is
            pinned; it is not. A store-level test with the real wiring goes red on the double-write variant
            and green at HEAD - verified both directions.
          family: unpinned-invariant
          round: 2
        - id: BR-6
          severity: Important
          title: The Forget traversal guard is asserted by no test, and half of it is unreachable
          detail: |-
            cmd/define/store/yaml.go:306. The issue's Done-when checks "cannot delete outside words/ -
            asserted, not inherited from Slug" and plan Task 3 Step 0 names the class. No test passes a
            traversal key to Forget. Verified by revert: replacing filepath.Base(Slug(k)) plus the
            unsafe-name check with a bare Slug(k) leaves go test ./cmd/define/store/... green. Separately
            name always ends in ".yaml", so the name == "." and name == ".." sub-conditions can never fire.
          family: unpinned-invariant
          round: 2
        - id: BR-7
          severity: Important
          title: openStore and the DEFINE_NO_CAPTURE env wiring have zero coverage, leaving half a Done-when unverified
          detail: |-
            cmd/define/main.go:73. Done-when says the opt-out also drops history to session-only; that
            clause lives entirely in openStore's noCapture return and nothing exercises it, nor the
            os.Getenv to opt.noCapture wiring at main.go:172, nor the Getwd-failure warning.
            TestNoCaptureSuppressesEverything only re-asserts decideCapture's branch through a
            storeCapturer built by hand.
          family: unpinned-invariant
          round: 2
        - id: BR-8
          severity: Important
          title: The raw success path bypasses decideCapture, giving "capture is off" three homes
          detail: |-
            cmd/define/main.go:247 returns before the capture call at :252, so Capture(word, true, opt)
            under -raw is unreachable and capture.go:30's raw branch fires only on the failure path - the
            truth-table row {"raw", true, ...} asserts a combination production never produces. openStore
            (main.go:74) reads opt.noCapture a third time to install noopCapturer. The plan explicitly
            forbids exactly this second home (ARCH-DRY).
          family: single-source-restated-by-hand
          round: 2
        - id: BR-9
          severity: Important
          title: atlas/define.md:300 still describes storeHistory.Add as the writer, contradicting :282 fifteen lines above
          detail: |-
            "History is events, the deck is successes" opens with "storeHistory.Add always appends an event,
            and upserts a word only when the lookup found something" and attributes the warn-once rule to it.
            Both are false since this diff, and the new "Capture: one site, one policy" section directly
            above says so. Same rule the gate raised as PQ-11, recurring in the atlas.
          family: prose-contradicts-code
          round: 2
        - id: BR-10
          severity: Important
          title: main.go:67 claims DEFINE_NO_CAPTURE is documented in --help; fs.Usage never mentions it
          detail: |-
            Confirmed against `go run ./cmd/define -h`. The plan requires the opt-out and its cost be stated
            in --help, the README and the atlas. It is an env var, so PrintDefaults will never surface it,
            and the usage text also never says define now writes to the working directory at all.
          family: prose-contradicts-code
          round: 2
        - id: BR-11
          severity: Important
          title: .gitignore does not ignore words/ or events/, which define now creates in the repo on every lookup
          detail: |-
            This diff makes define write to CWD on every invocation including failed ones, and this repo's
            developers run define from the checkout. Reproduced during review - a failed lookup created an
            untracked events/ in the repo root (removed; tree is clean). The existing .gitignore already
            carries a comment about a build artifact that first got committed this way.
          family: writes-to-cwd-unignored
          round: 2
        - id: BR-12
          severity: Minor
          title: define -forget="" falls through to the REPL instead of erroring
          detail: main.go:188 gates on *forget != "", so an explicitly-empty -forget starts an interactive session.
          family: flag-mode-dispatch
          round: 2
        - id: BR-13
          severity: Minor
          title: -forget under DEFINE_NO_CAPTURE reports "no deck in this directory"
          detail: Verified. There may well be a deck; the user opted out of writes. The message should say that.
          family: misleading-error-text
          round: 2
        - id: BR-14
          severity: Minor
          title: deps.forgetter() is a four-line nil-check wrapper around one field with one caller
          family: needless-indirection
          round: 2
        - id: BR-15
          severity: Minor
          title: newStore's three-return seam plus three nil-merges in run is lumpy; a small struct would collapse it
          family: needless-indirection
          round: 2
        - id: BR-16
          severity: Minor
          title: The issue's Log has no implementation entry and the ticked "Manual check" step records no evidence
          family: undocumented-work-log
          round: 2
        - id: BR-17
          severity: Minor
          title: atlas "Entry modes" table omits define -forget, the fourth invocation this diff adds
          family: prose-contradicts-code
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-21T11:22:26-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 116 unchanged and now wrong twice over - decideCapture has exactly one caller.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: Plan line 181 unchanged; signature also drifted to Capture(word, found, opt).
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Code resolved it (run installs noopCapturer, rigs supply one); the plan still never says so.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Verified by revert - restoring the double write reddens only TestNoDoubleWriteThroughTheRealWiring.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: yaml.go untouched by 6eb36f8 despite the commit message; guard removal still leaves the store suite green.
          round: 3
        - id: BR-7
          disposition: addressed
          note: Verified both directions - severing the env wiring and gutting openStore each redden a distinct test.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Raw branch now calls Capture; behaviour-identical so no test sees it - carried into the N-2 rule finding.
          round: 3
        - id: BR-9
          disposition: addressed
          note: Atlas rewritten correctly; the same sentence survives in history_store.go - raised as N-1.
          round: 3
        - id: BR-10
          disposition: addressed
          note: Confirmed against go run ./cmd/define -h; usage now states cwd writes and DEFINE_NO_CAPTURE.
          round: 3
        - id: BR-11
          disposition: addressed
          note: /words/ and /events/ added with the reason recorded.
          round: 3
        - id: BR-12
          disposition: addressed
          note: Verified with the built binary - define -forget="" exits 2 with "-forget needs a word".
          round: 3
        - id: BR-13
          disposition: addressed
          note: Verified - now reports "DEFINE_NO_CAPTURE is set, so no deck was opened".
          round: 3
        - id: BR-14
          disposition: not-addressed
          note: deps.forgetter() unchanged at main.go:46.
          round: 3
        - id: BR-15
          disposition: not-addressed
          note: newStore still returns a triple with three nil-merges in run.
          round: 3
        - id: BR-16
          disposition: not-addressed
          note: Issue Log still ends at the 2026-08-20 creation line; no implementation entry, no manual-check evidence.
          round: 3
        - id: BR-17
          disposition: not-addressed
          note: Entry-modes table still lists three invocations; the prose above it is now wrong too.
          round: 3
      findings:
        - id: BR-18
          severity: Important
          title: history_store.go:17 still says Add appends events and upserts words, contradicting line 25 of the same comment
          detail: |-
            4th in family (BR-9 atlas, BR-10 --help, BR-17 entry-modes; prevalence 4). Do NOT patch this
            instance. Round 2's I-4 fixed this exact sentence in the atlas and left the code copy eight lines
            above the sentence that refutes it. The rule: a behavioural fact gets exactly one normative home
            and every other mention points at it. Applied here that means DELETING the "Two things are
            deliberately NOT the same here" bullets at history_store.go:15-22 - the atlas owns the split and
            the surviving prose already says everything true - not rewording them into a fifth copy.
          family: prose-contradicts-code
          round: 3
        - id: BR-19
          severity: Important
          title: Two fixes landed this round are revert-green, and BR-6's is unpinnable at the current API
          detail: |-
            4th in family (BR-5, BR-6, BR-7; prevalence 4). Do NOT patch these instances. Removing the entire
            BR-8 fix - d.capture.Capture at main.go:257 - leaves go test ./cmd/define/... green, because the
            arity test has subtests for one-shot, piped, raw editor, replay and failure but none for -raw, the
            one truth-table row production can now produce. openStore's non-noCapture deck return is likewise
            untested; I verified it only by running the binary. The rule: a fix ships with a test whose failure
            you have OBSERVED by deleting the fix - delete the line, run the suite, and if it stays green you
            wrote documentation. BR-5 and BR-7 show the practice works; these show it was applied selectively.
            For BR-6 the rule forces an honest choice: extract the name computation so the guard is exercisable
            at its own level, or drop "asserted, not inherited from Slug" from the Done-when.
          family: unpinned-invariant
          round: 3
        - id: BR-20
          severity: Important
          title: A Done-when box is ticked for a clause that revert-verification shows is not delivered
          detail: |-
            2nd in family (BR-16; prevalence 2). Do NOT just untick this box. Issue line 44 ticks "--forget
            cannot delete outside words/ - asserted, not inherited from Slug"; the assertion does not exist.
            Line 41 ticks "asserted by comparing the directory before and after" against a test that counts
            events instead. Log still ends at the creation line. The rule: a tick claims evidence exists, so
            record the evidence in "## Log" at the moment of ticking, naming the test or pasting the command
            output. Same rule one layer out covers commit 6eb36f8, which describes a yaml.go change the commit
            does not contain.
          family: undocumented-work-log
          round: 3
        - id: BR-21
          severity: Minor
          title: newStoreHistory keeps a dead store.Clock parameter that four call sites construct and pass
          detail: |-
            3rd in family (BR-14, BR-15; prevalence 3). Do NOT patch this instance alone. The rule: when a
            refactor strips a component's responsibilities, strip the surface that served them in the same
            commit - a retained parameter, wrapper or return slot outlives its reason and reads as intentional.
            One pass over deps.forgetter(), the newStore triple and this parameter closes the family.
          family: needless-indirection
          round: 3
        - id: BR-22
          severity: Minor
          title: The committed close-review artifact opens with a harness stderr preamble
          detail: |-
            workshop/plans/000004-vocab-capture-close-review.md:18 carries "Ignoring 6 permissions.allow
            entries from .claude/settings.json..." inside the "## Review" section. Artifact capture should
            take the agent's stdout only, or this recurs on every review run in an untrusted workspace.
          family: generated-artifact-noise
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-21T11:34:57-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 116 unchanged; decideCapture still has exactly one caller.
          round: 4
        - id: BR-2
          disposition: not-addressed
          note: Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
          round: 4
        - id: BR-3
          disposition: not-addressed
          note: Plan line 181 unchanged; the plan file's only edit this window is checkbox ticks.
          round: 4
        - id: BR-4
          disposition: not-addressed
          note: Plan still never states that deps gains capture, deck or newStore, nor the noopCapturer fallback.
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: wordFileName was wired into Upsert, not Forget; Forget's guard is unchanged and still revert-green.
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: deps.forgetter() unchanged at main.go:46.
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: newStore still returns a triple with three nil-merges at main.go:179-193.
          round: 4
        - id: BR-16
          disposition: addressed
          note: Implementation Log entry landed and is substantive; the manual-check evidence residual carries under BR-20.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: atlas/define.md:317-327 unchanged - table still lists three invocations, prose still names defineOnce as the dispatch target.
          round: 4
        - id: BR-18
          disposition: addressed
          note: The restatement was deleted rather than reworded; history_store.go:13-22 now keeps only the locally-owned fact.
          round: 4
        - id: BR-19
          disposition: addressed
          note: Verified by mutation - deleting the raw Capture call and nilling openStore's deck each redden a distinct test.
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: Issue line 45-49, the Log and commit 00f9b94 all state the --forget guard is wordFileName; it is not.
          round: 4
        - id: BR-21
          disposition: not-addressed
          note: newStoreHistory's dead store.Clock parameter unchanged; the one-pass fix the finding specified did not happen.
          round: 4
        - id: BR-22
          disposition: not-addressed
          note: Preamble still at close-review.md:18 and recurred at :251 on the round-3 run, exactly as predicted.
          round: 4
      findings:
        - id: BR-23
          severity: Important
          title: The family fixes closed each finding's titled instance and left the instances enumerated in its body
          detail: |-
            Measured across round 3: BR-18 named 4 instances and 1 closed; BR-19 named 3 and 2 closed;
            BR-21 named 3, said "one pass over all three closes the family", and 0 closed. That is 3 of 10,
            and the three closed are the ones in the titles. A fifth prose-contradicts-code instance also
            arrived unremarked - README.md:85 still says exit 1 means "no dictionary entry", which now also
            means "not in the deck" and "no deck was opened". The rule: an escalated family finding is closed
            only when every instance it enumerates is disposed, and the response states which were fixed and
            which were not. Marking a family finding addressed asserts the family is closed, not that the
            headline site was patched - which is the same fix-the-named-thing substitution the escalation
            mechanism exists to stop. Next round: reply to BR-18, BR-19 and BR-21 instance-by-instance.
          family: family-rule-applied-selectively
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-08-21T11:49:52-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 115-116 unchanged; the plan file's only edit in the whole window is 3ca9ab4's checkbox ticks.
          round: 5
        - id: BR-2
          disposition: not-addressed
          note: Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
          round: 5
        - id: BR-3
          disposition: not-addressed
          note: Lines 7, 24, 116, 181 all unchanged - an escalated family finding that enumerated four sites and closed none.
          round: 5
        - id: BR-4
          disposition: not-addressed
          note: Plan still never states deps gains capture, deck or newStore, nor the noopCapturer fallback.
          round: 5
        - id: BR-6
          disposition: addressed
          note: wordFileName now serves Upsert AND Forget (grep-verified); guard is mutation-RED; Done-when honestly retracts the "asserted" claim with a measured GREEN I reproduced.
          round: 5
        - id: BR-14
          disposition: addressed
          note: deps.forgetter() deleted; forgetWord tests d.deck == nil directly.
          round: 5
        - id: BR-15
          disposition: addressed
          note: Collapsed into a storeDeps value and one withStore call, as the finding specified.
          round: 5
        - id: BR-17
          disposition: addressed
          note: Entry-modes table now has four rows including define -forget, and the prose above it names lookupAndRender.
          round: 5
        - id: BR-20
          disposition: addressed
          note: Both false ticks rewritten to state what is actually asserted; Log carries two substantive entries with a per-instance table.
          round: 5
        - id: BR-21
          disposition: addressed
          note: newStoreHistory's Clock parameter removed along with all four call sites' constructions.
          round: 5
        - id: BR-22
          disposition: not-addressed
          note: Preamble now at close-review.md:18, :251 AND :485 - a third occurrence added by the round-4 run.
          round: 5
        - id: BR-23
          disposition: addressed
          note: Verified all ten enumerated instances across BR-18/BR-19/BR-21 are closed, and the response replies instance-by-instance.
          round: 5
      findings:
        - id: BR-24
          severity: Important
          title: The family rule was applied to every code instance and to no artifact instance
          detail: |-
            2nd in family (BR-23; prevalence 2). Do NOT patch the four plan lines in isolation - BR-3 already
            asked for exactly that and got nothing. Measured this round: code-side families closed 10 of 10
            enumerated instances; artifact-side closed 0 of 4 findings covering at least 7 sites. BR-3 is
            itself an escalated family finding whose body enumerates plan lines 7, 24, 116 and 181 and says
            "rewrite [them] to point at the Chunk 1 statement"; none were touched, and the plan file's only
            edit in the entire window is 3ca9ab4's checkbox ticks. There is still no "## Revisions" section,
            which AGENTS.md section 1 requires and three rounds have recommended. The rule is BR-23's with
            the scope clause it was missing: every open finding gets an instance-by-instance disposition
            regardless of which artifact its instances live in, and where they live in the plan the closing
            move is a "## Revisions" entry, not a checkbox tick.
          family: family-rule-applied-selectively
          round: 5
        - id: BR-25
          severity: Important
          title: This round's own edits left two comments describing code they no longer describe
          detail: |-
            6th in family (BR-9, BR-10, BR-17, BR-18, README exit codes; prevalence 6). Do NOT patch these
            two sites. cmd/define/store/yaml.go:317-320 - Forget's doc still says the guarantee is "asserted
            here rather than inherited: filepath.Base is applied to the slug", but 96adc20 deleted that
            filepath.Base call in the same hunk, and the same commit rewrote the issue Done-when to RETRACT
            "asserted rather than inherited". A false safety claim on the delete path is the exact harm the
            BR-6/BR-20 arc was about. cmd/define/main.go:55-64 - inserting type storeDeps between openStore's
            doc comment and openStore orphaned it; AST-verified, type storeDeps now carries "openStore builds
            the store-backed dependencies..." and openStore has no doc, while the Spec names that comment as
            one of three homes for the opt-out's cost. The rule is round 3's with the missing clause: closing
            a prose-contradicts-code instance means sweeping the file the fix touched, not the line the
            finding named. Both sites are in files this commit edited, and the first is a third restatement of
            a story wordFileName's own doc comment and the Done-when already tell.
          family: prose-contradicts-code
          round: 5
        - id: BR-26
          severity: Minor
          title: withStore's early return silently strands deck when history and capture are both supplied
          detail: |-
            2nd in family (BR-4; prevalence 2). Do NOT patch this instance alone. main.go:77 returns early
            when history and capture are non-nil, so newStore is never consulted and deck stays nil - the
            pre-refactor block filled deck independently. Verified with a probe: "newStore called = false,
            deck = <nil>". Not reachable from production (realDeps supplies neither) and no test hits it, so
            not a live bug, but it is a new seam whose rule no comment states and no test pins, introduced by
            a cleanup whose stated purpose was that "the merge was three chances to forget one". The rule
            covering this and BR-4: every field of an injected seam states its own default and how it is
            filled, and a fill-in helper must not make one field's default depend on another field's presence.
          family: unstated-seam-default
          round: 5
      blocked: false
    - "n": 6
      timestamp: "2026-08-21T12:01:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Revisions section 1 states "There is one call site" explicitly; the AGENTS.md section 1 append-not-overwrite move BR-24 specified.
          round: 6
        - id: BR-2
          disposition: addressed
          note: Line 133 now reads "stops writing entirely - it does not delegate, it does nothing", plus Revisions section 2.
          round: 6
        - id: BR-3
          disposition: addressed
          note: Line 181 rewritten to lookupAndRender with the three-arg signature; lines 7/24/116 covered by Revisions rather than in place.
          round: 6
        - id: BR-4
          disposition: not-addressed
          note: Revisions section 3 names the three new deps fields but never states their defaults - the exact half BR-4 asked for and BR-26 escalated.
          round: 6
        - id: BR-22
          disposition: not-addressed
          note: Fourth occurrence (lines 18, 251, 485, 674). Generator is ariadne/cmd/sdlc, a peer repo - unfixable from tools; needs an ariadne issue.
          round: 6
        - id: BR-24
          disposition: addressed
          note: Plan substantively edited with a Revisions section; 6 of 7 enumerated artifact instances closed, the 7th carried under BR-4/BR-26.
          round: 6
        - id: BR-25
          disposition: addressed
          note: Both sites verified - Forget's doc derives from wordFileName (grep), openStore's doc re-attached to openStore (AST).
          round: 6
        - id: BR-26
          disposition: not-addressed
          note: main.go:67 early return unchanged; no comment states deck's default, no test enters the branch.
          round: 6
      findings:
        - id: BR-27
          severity: Important
          title: Done-when 2 "repeat lookups increment the count" is ticked and no test pins it through the capture path
          detail: |-
            5th in family (BR-5, BR-6, BR-7, BR-19; prevalence 5). Do NOT patch this instance alone. Verified by
            mutation: making storeCapturer skip Upsert on a repeat sighting of the same word leaves the ENTIRE suite
            green (MUTATION_APPLIED, BUILD_OK, ok cmd/define 24.084s). The only Lookups assertions are
            storetest/suite.go:71 (Upsert merge semantics, a different question) and capture_test.go:282 (== 1);
            nothing drives two captures of one word and asserts the count moved. Issue 5 orders by this number.
            The rule is BR-19's with the clause it was missing - the delete-the-line discipline was applied to FIXES
            and never to Done-when TICKS, though a tick is the same kind of behavioural claim. That clause also
            unifies this family with undocumented-work-log, whose rule (BR-20, "a tick claims evidence exists") has
            been chasing the same thing from the other end for four rounds. Cheap close: two lines inside
            TestNoDoubleWriteThroughTheRealWiring, which already has real wiring over a real store.
          family: unpinned-invariant
          round: 6
        - id: BR-28
          severity: Minor
          title: main.go:35 claims no test touches the real filesystem, in the file this commit says it swept
          detail: |-
            7th in family (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25; prevalence 7). Do NOT patch this
            site. deps.newStore's comment says "Tests leave it nil and get in-memory defaults, so no test ever
            touches the real filesystem"; capture_test.go:316 sets rig.deps.newStore = openStore, and four test
            files use t.TempDir(). Both clauses false, ~30 lines above the comment this commit moved. The mechanism
            that defeated BR-25's sweep rule, measured: round 5 stated this defect in its Minor PROSE list and never
            gave it a BR id - 0 of that list's 5 items were addressed while 100 percent of the id'd findings were.
            So the rule needs both halves: the reviewer puts every stated defect in the machine-read findings block
            (done this round for all five), and the sweep becomes mechanical rather than attentional - for each file
            in git diff --stat, read every comment making a claim about another symbol and grep that symbol.
          family: prose-contradicts-code
          round: 6
        - id: BR-29
          severity: Minor
          title: BR-22 cannot be closed from this repo - the generator lives in the ariadne peer
          detail: |-
            Recording this so BR-22 stops recurring undisposed. The close-review artifact is written by sdlc, whose
            source is /Users/xianxu/workspace/ariadne/cmd/sdlc; no commit in tools can change what it captures. The
            preamble is now at lines 18, 251, 485 and 674 - one new occurrence per review round, exactly as predicted
            at rounds 3 and 4. The actionable move is an ariadne issue for "artifact capture takes agent stdout only",
            referenced from this issue's Log, rather than a fifth not-addressed disposition here.
          family: generated-artifact-noise
          round: 6
      blocked: false
    - "n": 7
      timestamp: "2026-08-21T12:15:28-07:00"
      agent: claude
      dispose:
        - id: BR-27
          disposition: addressed
          note: Both mutations re-run independently - skipping Upsert on a repeat and propagating a capture error each redden exactly one test; applied and compiled verified.
          round: 7
        - id: BR-4
          disposition: not-addressed
          note: Revisions section 3 still names the three deps fields with no default stated for any of them.
          round: 7
        - id: BR-26
          disposition: not-addressed
          note: Probe re-verified - a panic in withStore's early return leaves the entire suite green, so no test enters the branch; no comment states deck's default.
          round: 7
        - id: BR-28
          disposition: not-addressed
          note: main.go:36 comment verbatim; capture_test.go:316 still sets newStore = openStore and five test files use t.TempDir().
          round: 7
        - id: BR-22
          disposition: not-addressed
          note: Fifth occurrence (lines 18, 251, 485, 674, 878). Unfixable from tools - see BR-29.
          round: 7
        - id: BR-29
          disposition: not-addressed
          note: No ariadne issue exists (checked ariadne/workshop/issues) and this issue's Log references only ariadne#195, a different mechanism.
          round: 7
      findings:
        - id: BR-30
          severity: Critical
          title: A 9.6MB Mach-O arm64 binary is committed at cmd/define/define in the HEAD commit under review
          detail: |-
            2nd in family (BR-11; prevalence 2). Do NOT just add a fourth anchored .gitignore line - but DO excise the
            blob before merge, because it is the one open item that becomes irreversible. Measured: git ls-files shows
            it tracked, git log --diff-filter=A dates it to 42cc96d, it is 9616546 bytes against a 99336-byte
            next-largest blob, and .git is currently 9.3M so it roughly doubles the repo for every clone forever. It
            is arm64-only, so a Linux or amd64 checkout gets an unrunnable file; and because it is tracked, cd
            cmd/define && go build now dirties git status on every developer build. The rule covering this and BR-11:
            an artifact the developer workflow drops into the working tree gets an ignore pattern covering EVERY
            directory it can be produced in, added when the tool learns to produce it. .gitignore has been patched
            three times by this rule's absence and each patch was anchored to the one place it had already happened
            (bin/, then /define root-only, then /words/ and /events/); go build in a main package writes the binary
            into that package's directory, which none of them match. Second half, since no ignore would have caught
            this one - it was git add-ed before any ignore existed: read git show --stat before committing, where
            "Bin 0 -> 9616546 bytes" is visible at a glance and does appear in this commit's own stat. Fix while the
            branch is unmerged: git rm --cached cmd/define/define, add an un-anchored pattern, git commit --amend.
          family: writes-to-cwd-unignored
          round: 7
        - id: BR-31
          severity: Important
          title: The retracted "a regression in Slug must fail HERE" claim survives in storetest/suite.go and is false by measurement
          detail: |-
            8th in family (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25, BR-28; prevalence 8). Do NOT patch this
            site. storetest/suite.go:140-142 says the traversal guarantee is "asserted here rather than inherited from
            Slug - a regression in Slug must fail HERE, loudly". That is the exact sentence 96adc20 deleted from
            YAML.Forget's doc for BR-25 and the exact claim the same commit rewrote the Done-when to RETRACT; wordFileName's
            own doc now says the opposite. Verified by mutation: regressing Slug entirely (no sanitising) reddens
            TestSlugDoesNotMergeHyphenAndSpace, TestSlugIsAlwaysOneSafePathElement and FuzzSlugIsSafe, and leaves
            Suite/forget cannot escape the words directory GREEN - it discards err, so its assertions pass whether the
            guard is present, absent or bypassed. Second site the same sweep finds: main.go:27, capture.go:50 and
            atlas/define.md:283 each state absolutely that storeCapturer is the only thing that writes to the store,
            while forgetWord's d.deck.Forget at main.go:396 deletes a word file - deps.deck's own comment three lines
            below main.go:27 acknowledges the second mutator. The rule is BR-28's, which is correct and was never
            executed as written: run the mechanical sweep over ALL 14 changed .go/.md files, not the files a finding
            named - storetest/suite.go is in git diff --stat and grepping Slug from that comment lands on the
            contradiction directly.
          family: prose-contradicts-code
          round: 7
        - id: BR-32
          severity: Minor
          title: History.Add's found parameter is now dead in both implementations, and orElse is a generic for two nil checks
          detail: |-
            4th in family (BR-14, BR-15, BR-21; prevalence 4). Do NOT patch these instances. History.Add(line string,
            found bool) at history.go:16 is ignored by BOTH implementations - history.go:28 and history_store.go:50 both
            take _ bool - while the sole caller computes code == 0 to supply it (replraw.go:171). It existed only
            because storeHistory.Add decided event-vs-deck; #4 moved that decision and left the parameter, which is
            BR-21's rule verbatim. It was missed because it lives in history.go, a file the diff never touched, so
            BR-28's file-based sweep cannot reach it - the extension the rule needs is: sweep the changed files AND the
            interface plus callers of every symbol whose responsibilities the diff moved. Weaker second instance,
            reasonably withdrawn: orElse[T comparable] at main.go:83 is a generic helper introduced by the very commit
            that closed this family, replacing two nil checks with six lines.
          family: needless-indirection
          round: 7
        - id: BR-33
          severity: Minor
          title: run builds the store-backed deps before the -forget dispatch, so --forget reads an event log it never uses
          detail: |-
            main.go:212 calls d.withStore before the -forget dispatch at :214, so --forget constructs storeHistory and
            loads the whole event log for a result it never consults. Reproduced against a seeded directory: a -forget
            run printed "define: 2026-08-21.yaml: recovered 0 event(s), dropped 1 torn record(s)" for a log it does not
            touch. Stated in round 6's prose and never given an id, which is the mechanism BR-28 named; recording it so
            it can be disposed rather than restated. The rule: a mode dispatch decides which dependencies are needed,
            so it runs before they are constructed.
          family: eager-dependency-construction
          round: 7
        - id: BR-34
          severity: Minor
          title: 'forgetWord prints a bare "define: %v" where every neighbouring message names the operand'
          detail: |-
            2nd in family (BR-13; prevalence 2). Do NOT patch this instance alone. main.go:398 formats a store error as
            "define: %v" while its neighbours use "define: %s: %v" with the word. The rule covering this and BR-13: a
            user-facing error names both the condition and the operand it failed on - BR-13's message named the wrong
            condition, this one names no operand.
          family: misleading-error-text
          round: 7
      blocked: true
---

# Gate ledger — tools#4 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-21T11:08:20-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `unbacked-existing-behavior-claim` "three call sites" is two - defineOnce serves both the one-shot and line paths
  main.go:144 and repl.go:144 both call defineOnce, so widening adds one
  invocation site, not two. The design is still right; the count is not.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-2** [Important] `extraction-strands-behavior` storeHistory is given two incompatible fates, and Task 1 Step 3's "tests unchanged" is unsatisfiable
  This is the 2nd finding in family extraction-strands-behavior (prevalence 2: PQ-4's
  warn-once, now the durability contract itself). The rule: an extraction must enumerate
  every obligation the source component is contracted for — the tests that pin it, the
  warn-once, the durability guarantee — and say where each lands. Plan line 77 says
  storeHistory becomes a pure reader; line 109 says it delegates; the issue checkbox
  agrees with line 109. Under the reader reading, history_store_test.go:17-33, :37-55,
  :86-99, :155-166 and :133-150 all fail, contradicting "green without edits … do not
  edit them to fit".
  (carried from plan-quality PQ-10, deferred to the boundary review)
- **BR-3** [Important] `capture-arity-invariant` Task 2 Step 3 still names defineOnce as the capture site, contradicting Chunk 1's lookupAndRender
  This is the 3rd finding in family capture-arity-invariant (prevalence 3: PQ-7 double-capture,
  PQ-9 zero-capture-on-raw, now the executable step disagreeing with the prose). Do not patch
  line 181. The rule, which also covers PQ-6 and PQ-10: each design fact gets exactly ONE
  normative statement in the plan and every other mention references it rather than restating
  it (ARCH-DRY applied to the artifact). "Where capture happens" is currently stated at lines
  7, 24, 72, 116 and 181; three rounds have each fixed one copy and left the rest, which is
  why the family keeps recurring. Rewrite 7, 24, 116, 181 to point at the Chunk 1 statement,
  and give storeHistory's fate the same single-home treatment across lines 7, 24, 133 and the
  issue checkbox.
  (carried from plan-quality PQ-11, deferred to the boundary review)
- **BR-4** [Minor] `unstated-seam-default` The plan calls d.capture but never says deps gains the field, nor what a deps literal without it does
  deps is the injected IO seam (main.go:20) and both test rigs build it as a literal
  (main_test.go:14, main_test.go:41), so an unguarded d.capture.Capture panics the suite.
  The codebase already has the idiom for this at replraw.go:63, where a nil history seam
  falls back to memHistory. State whether capture takes a nil-fallback null object or every
  rig must supply one.
  (carried from plan-quality PQ-12, deferred to the boundary review)

## Round 2 — 2026-08-21T11:08:20-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Important] `unpinned-invariant` The capture-arity test counts Capturer calls, not store writes, so the double-count it names passes
  cmd/define/capture_test.go:86 injects countingCapturer at the seam, above both decideCapture
  and the writes. Verified by revert: restoring the AppendEvent/Upsert pair in storeHistory.Add
  leaves the entire suite green while the deck records Lookups:2 and two events per lookup. The
  plan's Task 2 Step 1 specified a counting STORE. atlas/define.md:285 claims the invariant is
  pinned; it is not. A store-level test with the real wiring goes red on the double-write variant
  and green at HEAD - verified both directions.
- **BR-6** [Important] `unpinned-invariant` The Forget traversal guard is asserted by no test, and half of it is unreachable
  cmd/define/store/yaml.go:306. The issue's Done-when checks "cannot delete outside words/ -
  asserted, not inherited from Slug" and plan Task 3 Step 0 names the class. No test passes a
  traversal key to Forget. Verified by revert: replacing filepath.Base(Slug(k)) plus the
  unsafe-name check with a bare Slug(k) leaves go test ./cmd/define/store/... green. Separately
  name always ends in ".yaml", so the name == "." and name == ".." sub-conditions can never fire.
- **BR-7** [Important] `unpinned-invariant` openStore and the DEFINE_NO_CAPTURE env wiring have zero coverage, leaving half a Done-when unverified
  cmd/define/main.go:73. Done-when says the opt-out also drops history to session-only; that
  clause lives entirely in openStore's noCapture return and nothing exercises it, nor the
  os.Getenv to opt.noCapture wiring at main.go:172, nor the Getwd-failure warning.
  TestNoCaptureSuppressesEverything only re-asserts decideCapture's branch through a
  storeCapturer built by hand.
- **BR-8** [Important] `single-source-restated-by-hand` The raw success path bypasses decideCapture, giving "capture is off" three homes
  cmd/define/main.go:247 returns before the capture call at :252, so Capture(word, true, opt)
  under -raw is unreachable and capture.go:30's raw branch fires only on the failure path - the
  truth-table row {"raw", true, ...} asserts a combination production never produces. openStore
  (main.go:74) reads opt.noCapture a third time to install noopCapturer. The plan explicitly
  forbids exactly this second home (ARCH-DRY).
- **BR-9** [Important] `prose-contradicts-code` atlas/define.md:300 still describes storeHistory.Add as the writer, contradicting :282 fifteen lines above
  "History is events, the deck is successes" opens with "storeHistory.Add always appends an event,
  and upserts a word only when the lookup found something" and attributes the warn-once rule to it.
  Both are false since this diff, and the new "Capture: one site, one policy" section directly
  above says so. Same rule the gate raised as PQ-11, recurring in the atlas.
- **BR-10** [Important] `prose-contradicts-code` main.go:67 claims DEFINE_NO_CAPTURE is documented in --help; fs.Usage never mentions it
  Confirmed against `go run ./cmd/define -h`. The plan requires the opt-out and its cost be stated
  in --help, the README and the atlas. It is an env var, so PrintDefaults will never surface it,
  and the usage text also never says define now writes to the working directory at all.
- **BR-11** [Important] `writes-to-cwd-unignored` .gitignore does not ignore words/ or events/, which define now creates in the repo on every lookup
  This diff makes define write to CWD on every invocation including failed ones, and this repo's
  developers run define from the checkout. Reproduced during review - a failed lookup created an
  untracked events/ in the repo root (removed; tree is clean). The existing .gitignore already
  carries a comment about a build artifact that first got committed this way.
- **BR-12** [Minor] `flag-mode-dispatch` define -forget="" falls through to the REPL instead of erroring
  main.go:188 gates on *forget != "", so an explicitly-empty -forget starts an interactive session.
- **BR-13** [Minor] `misleading-error-text` -forget under DEFINE_NO_CAPTURE reports "no deck in this directory"
  Verified. There may well be a deck; the user opted out of writes. The message should say that.
- **BR-14** [Minor] `needless-indirection` deps.forgetter() is a four-line nil-check wrapper around one field with one caller
- **BR-15** [Minor] `needless-indirection` newStore's three-return seam plus three nil-merges in run is lumpy; a small struct would collapse it
- **BR-16** [Minor] `undocumented-work-log` The issue's Log has no implementation entry and the ticked "Manual check" step records no evidence
- **BR-17** [Minor] `prose-contradicts-code` atlas "Entry modes" table omits define -forget, the fourth invocation this diff adds

## Round 3 — 2026-08-21T11:22:26-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan line 116 unchanged and now wrong twice over - decideCapture has exactly one caller.
- BR-2 — not-addressed — Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
- BR-3 — not-addressed — Plan line 181 unchanged; signature also drifted to Capture(word, found, opt).
- BR-4 — not-addressed — Code resolved it (run installs noopCapturer, rigs supply one); the plan still never says so.
- BR-5 — addressed — Verified by revert - restoring the double write reddens only TestNoDoubleWriteThroughTheRealWiring.
- BR-6 — not-addressed — yaml.go untouched by 6eb36f8 despite the commit message; guard removal still leaves the store suite green.
- BR-7 — addressed — Verified both directions - severing the env wiring and gutting openStore each redden a distinct test.
- BR-8 — addressed — Raw branch now calls Capture; behaviour-identical so no test sees it - carried into the N-2 rule finding.
- BR-9 — addressed — Atlas rewritten correctly; the same sentence survives in history_store.go - raised as N-1.
- BR-10 — addressed — Confirmed against go run ./cmd/define -h; usage now states cwd writes and DEFINE_NO_CAPTURE.
- BR-11 — addressed — /words/ and /events/ added with the reason recorded.
- BR-12 — addressed — Verified with the built binary - define -forget="" exits 2 with "-forget needs a word".
- BR-13 — addressed — Verified - now reports "DEFINE_NO_CAPTURE is set, so no deck was opened".
- BR-14 — not-addressed — deps.forgetter() unchanged at main.go:46.
- BR-15 — not-addressed — newStore still returns a triple with three nil-merges in run.
- BR-16 — not-addressed — Issue Log still ends at the 2026-08-20 creation line; no implementation entry, no manual-check evidence.
- BR-17 — not-addressed — Entry-modes table still lists three invocations; the prose above it is now wrong too.

### Raised

- **BR-18** [Important] `prose-contradicts-code` history_store.go:17 still says Add appends events and upserts words, contradicting line 25 of the same comment
  4th in family (BR-9 atlas, BR-10 --help, BR-17 entry-modes; prevalence 4). Do NOT patch this
  instance. Round 2's I-4 fixed this exact sentence in the atlas and left the code copy eight lines
  above the sentence that refutes it. The rule: a behavioural fact gets exactly one normative home
  and every other mention points at it. Applied here that means DELETING the "Two things are
  deliberately NOT the same here" bullets at history_store.go:15-22 - the atlas owns the split and
  the surviving prose already says everything true - not rewording them into a fifth copy.
- **BR-19** [Important] `unpinned-invariant` Two fixes landed this round are revert-green, and BR-6's is unpinnable at the current API
  4th in family (BR-5, BR-6, BR-7; prevalence 4). Do NOT patch these instances. Removing the entire
  BR-8 fix - d.capture.Capture at main.go:257 - leaves go test ./cmd/define/... green, because the
  arity test has subtests for one-shot, piped, raw editor, replay and failure but none for -raw, the
  one truth-table row production can now produce. openStore's non-noCapture deck return is likewise
  untested; I verified it only by running the binary. The rule: a fix ships with a test whose failure
  you have OBSERVED by deleting the fix - delete the line, run the suite, and if it stays green you
  wrote documentation. BR-5 and BR-7 show the practice works; these show it was applied selectively.
  For BR-6 the rule forces an honest choice: extract the name computation so the guard is exercisable
  at its own level, or drop "asserted, not inherited from Slug" from the Done-when.
- **BR-20** [Important] `undocumented-work-log` A Done-when box is ticked for a clause that revert-verification shows is not delivered
  2nd in family (BR-16; prevalence 2). Do NOT just untick this box. Issue line 44 ticks "--forget
  cannot delete outside words/ - asserted, not inherited from Slug"; the assertion does not exist.
  Line 41 ticks "asserted by comparing the directory before and after" against a test that counts
  events instead. Log still ends at the creation line. The rule: a tick claims evidence exists, so
  record the evidence in "## Log" at the moment of ticking, naming the test or pasting the command
  output. Same rule one layer out covers commit 6eb36f8, which describes a yaml.go change the commit
  does not contain.
- **BR-21** [Minor] `needless-indirection` newStoreHistory keeps a dead store.Clock parameter that four call sites construct and pass
  3rd in family (BR-14, BR-15; prevalence 3). Do NOT patch this instance alone. The rule: when a
  refactor strips a component's responsibilities, strip the surface that served them in the same
  commit - a retained parameter, wrapper or return slot outlives its reason and reads as intentional.
  One pass over deps.forgetter(), the newStore triple and this parameter closes the family.
- **BR-22** [Minor] `generated-artifact-noise` The committed close-review artifact opens with a harness stderr preamble
  workshop/plans/000004-vocab-capture-close-review.md:18 carries "Ignoring 6 permissions.allow
  entries from .claude/settings.json..." inside the "## Review" section. Artifact capture should
  take the agent's stdout only, or this recurs on every review run in an untrusted workspace.

## Round 4 — 2026-08-21T11:34:57-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan line 116 unchanged; decideCapture still has exactly one caller.
- BR-2 — not-addressed — Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
- BR-3 — not-addressed — Plan line 181 unchanged; the plan file's only edit this window is checkbox ticks.
- BR-4 — not-addressed — Plan still never states that deps gains capture, deck or newStore, nor the noopCapturer fallback.
- BR-6 — not-addressed — wordFileName was wired into Upsert, not Forget; Forget's guard is unchanged and still revert-green.
- BR-14 — not-addressed — deps.forgetter() unchanged at main.go:46.
- BR-15 — not-addressed — newStore still returns a triple with three nil-merges at main.go:179-193.
- BR-16 — addressed — Implementation Log entry landed and is substantive; the manual-check evidence residual carries under BR-20.
- BR-17 — not-addressed — atlas/define.md:317-327 unchanged - table still lists three invocations, prose still names defineOnce as the dispatch target.
- BR-18 — addressed — The restatement was deleted rather than reworded; history_store.go:13-22 now keeps only the locally-owned fact.
- BR-19 — addressed — Verified by mutation - deleting the raw Capture call and nilling openStore's deck each redden a distinct test.
- BR-20 — not-addressed — Issue line 45-49, the Log and commit 00f9b94 all state the --forget guard is wordFileName; it is not.
- BR-21 — not-addressed — newStoreHistory's dead store.Clock parameter unchanged; the one-pass fix the finding specified did not happen.
- BR-22 — not-addressed — Preamble still at close-review.md:18 and recurred at :251 on the round-3 run, exactly as predicted.

### Raised

- **BR-23** [Important] `family-rule-applied-selectively` The family fixes closed each finding's titled instance and left the instances enumerated in its body
  Measured across round 3: BR-18 named 4 instances and 1 closed; BR-19 named 3 and 2 closed;
  BR-21 named 3, said "one pass over all three closes the family", and 0 closed. That is 3 of 10,
  and the three closed are the ones in the titles. A fifth prose-contradicts-code instance also
  arrived unremarked - README.md:85 still says exit 1 means "no dictionary entry", which now also
  means "not in the deck" and "no deck was opened". The rule: an escalated family finding is closed
  only when every instance it enumerates is disposed, and the response states which were fixed and
  which were not. Marking a family finding addressed asserts the family is closed, not that the
  headline site was patched - which is the same fix-the-named-thing substitution the escalation
  mechanism exists to stop. Next round: reply to BR-18, BR-19 and BR-21 instance-by-instance.

## Round 5 — 2026-08-21T11:49:52-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan line 115-116 unchanged; the plan file's only edit in the whole window is 3ca9ab4's checkbox ticks.
- BR-2 — not-addressed — Plan line 133 "storeHistory delegates" unchanged; it neither delegates nor writes.
- BR-3 — not-addressed — Lines 7, 24, 116, 181 all unchanged - an escalated family finding that enumerated four sites and closed none.
- BR-4 — not-addressed — Plan still never states deps gains capture, deck or newStore, nor the noopCapturer fallback.
- BR-6 — addressed — wordFileName now serves Upsert AND Forget (grep-verified); guard is mutation-RED; Done-when honestly retracts the "asserted" claim with a measured GREEN I reproduced.
- BR-14 — addressed — deps.forgetter() deleted; forgetWord tests d.deck == nil directly.
- BR-15 — addressed — Collapsed into a storeDeps value and one withStore call, as the finding specified.
- BR-17 — addressed — Entry-modes table now has four rows including define -forget, and the prose above it names lookupAndRender.
- BR-20 — addressed — Both false ticks rewritten to state what is actually asserted; Log carries two substantive entries with a per-instance table.
- BR-21 — addressed — newStoreHistory's Clock parameter removed along with all four call sites' constructions.
- BR-22 — not-addressed — Preamble now at close-review.md:18, :251 AND :485 - a third occurrence added by the round-4 run.
- BR-23 — addressed — Verified all ten enumerated instances across BR-18/BR-19/BR-21 are closed, and the response replies instance-by-instance.

### Raised

- **BR-24** [Important] `family-rule-applied-selectively` The family rule was applied to every code instance and to no artifact instance
  2nd in family (BR-23; prevalence 2). Do NOT patch the four plan lines in isolation - BR-3 already
  asked for exactly that and got nothing. Measured this round: code-side families closed 10 of 10
  enumerated instances; artifact-side closed 0 of 4 findings covering at least 7 sites. BR-3 is
  itself an escalated family finding whose body enumerates plan lines 7, 24, 116 and 181 and says
  "rewrite [them] to point at the Chunk 1 statement"; none were touched, and the plan file's only
  edit in the entire window is 3ca9ab4's checkbox ticks. There is still no "## Revisions" section,
  which AGENTS.md section 1 requires and three rounds have recommended. The rule is BR-23's with
  the scope clause it was missing: every open finding gets an instance-by-instance disposition
  regardless of which artifact its instances live in, and where they live in the plan the closing
  move is a "## Revisions" entry, not a checkbox tick.
- **BR-25** [Important] `prose-contradicts-code` This round's own edits left two comments describing code they no longer describe
  6th in family (BR-9, BR-10, BR-17, BR-18, README exit codes; prevalence 6). Do NOT patch these
  two sites. cmd/define/store/yaml.go:317-320 - Forget's doc still says the guarantee is "asserted
  here rather than inherited: filepath.Base is applied to the slug", but 96adc20 deleted that
  filepath.Base call in the same hunk, and the same commit rewrote the issue Done-when to RETRACT
  "asserted rather than inherited". A false safety claim on the delete path is the exact harm the
  BR-6/BR-20 arc was about. cmd/define/main.go:55-64 - inserting type storeDeps between openStore's
  doc comment and openStore orphaned it; AST-verified, type storeDeps now carries "openStore builds
  the store-backed dependencies..." and openStore has no doc, while the Spec names that comment as
  one of three homes for the opt-out's cost. The rule is round 3's with the missing clause: closing
  a prose-contradicts-code instance means sweeping the file the fix touched, not the line the
  finding named. Both sites are in files this commit edited, and the first is a third restatement of
  a story wordFileName's own doc comment and the Done-when already tell.
- **BR-26** [Minor] `unstated-seam-default` withStore's early return silently strands deck when history and capture are both supplied
  2nd in family (BR-4; prevalence 2). Do NOT patch this instance alone. main.go:77 returns early
  when history and capture are non-nil, so newStore is never consulted and deck stays nil - the
  pre-refactor block filled deck independently. Verified with a probe: "newStore called = false,
  deck = <nil>". Not reachable from production (realDeps supplies neither) and no test hits it, so
  not a live bug, but it is a new seam whose rule no comment states and no test pins, introduced by
  a cleanup whose stated purpose was that "the merge was three chances to forget one". The rule
  covering this and BR-4: every field of an injected seam states its own default and how it is
  filled, and a fill-in helper must not make one field's default depend on another field's presence.

## Round 6 — 2026-08-21T12:01:34-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Revisions section 1 states "There is one call site" explicitly; the AGENTS.md section 1 append-not-overwrite move BR-24 specified.
- BR-2 — addressed — Line 133 now reads "stops writing entirely - it does not delegate, it does nothing", plus Revisions section 2.
- BR-3 — addressed — Line 181 rewritten to lookupAndRender with the three-arg signature; lines 7/24/116 covered by Revisions rather than in place.
- BR-4 — not-addressed — Revisions section 3 names the three new deps fields but never states their defaults - the exact half BR-4 asked for and BR-26 escalated.
- BR-22 — not-addressed — Fourth occurrence (lines 18, 251, 485, 674). Generator is ariadne/cmd/sdlc, a peer repo - unfixable from tools; needs an ariadne issue.
- BR-24 — addressed — Plan substantively edited with a Revisions section; 6 of 7 enumerated artifact instances closed, the 7th carried under BR-4/BR-26.
- BR-25 — addressed — Both sites verified - Forget's doc derives from wordFileName (grep), openStore's doc re-attached to openStore (AST).
- BR-26 — not-addressed — main.go:67 early return unchanged; no comment states deck's default, no test enters the branch.

### Raised

- **BR-27** [Important] `unpinned-invariant` Done-when 2 "repeat lookups increment the count" is ticked and no test pins it through the capture path
  5th in family (BR-5, BR-6, BR-7, BR-19; prevalence 5). Do NOT patch this instance alone. Verified by
  mutation: making storeCapturer skip Upsert on a repeat sighting of the same word leaves the ENTIRE suite
  green (MUTATION_APPLIED, BUILD_OK, ok cmd/define 24.084s). The only Lookups assertions are
  storetest/suite.go:71 (Upsert merge semantics, a different question) and capture_test.go:282 (== 1);
  nothing drives two captures of one word and asserts the count moved. Issue 5 orders by this number.
  The rule is BR-19's with the clause it was missing - the delete-the-line discipline was applied to FIXES
  and never to Done-when TICKS, though a tick is the same kind of behavioural claim. That clause also
  unifies this family with undocumented-work-log, whose rule (BR-20, "a tick claims evidence exists") has
  been chasing the same thing from the other end for four rounds. Cheap close: two lines inside
  TestNoDoubleWriteThroughTheRealWiring, which already has real wiring over a real store.
- **BR-28** [Minor] `prose-contradicts-code` main.go:35 claims no test touches the real filesystem, in the file this commit says it swept
  7th in family (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25; prevalence 7). Do NOT patch this
  site. deps.newStore's comment says "Tests leave it nil and get in-memory defaults, so no test ever
  touches the real filesystem"; capture_test.go:316 sets rig.deps.newStore = openStore, and four test
  files use t.TempDir(). Both clauses false, ~30 lines above the comment this commit moved. The mechanism
  that defeated BR-25's sweep rule, measured: round 5 stated this defect in its Minor PROSE list and never
  gave it a BR id - 0 of that list's 5 items were addressed while 100 percent of the id'd findings were.
  So the rule needs both halves: the reviewer puts every stated defect in the machine-read findings block
  (done this round for all five), and the sweep becomes mechanical rather than attentional - for each file
  in git diff --stat, read every comment making a claim about another symbol and grep that symbol.
- **BR-29** [Minor] `generated-artifact-noise` BR-22 cannot be closed from this repo - the generator lives in the ariadne peer
  Recording this so BR-22 stops recurring undisposed. The close-review artifact is written by sdlc, whose
  source is /Users/xianxu/workspace/ariadne/cmd/sdlc; no commit in tools can change what it captures. The
  preamble is now at lines 18, 251, 485 and 674 - one new occurrence per review round, exactly as predicted
  at rounds 3 and 4. The actionable move is an ariadne issue for "artifact capture takes agent stdout only",
  referenced from this issue's Log, rather than a fifth not-addressed disposition here.

## Round 7 — 2026-08-21T12:15:28-07:00 (claude) — BLOCKED

### Disposed

- BR-27 — addressed — Both mutations re-run independently - skipping Upsert on a repeat and propagating a capture error each redden exactly one test; applied and compiled verified.
- BR-4 — not-addressed — Revisions section 3 still names the three deps fields with no default stated for any of them.
- BR-26 — not-addressed — Probe re-verified - a panic in withStore's early return leaves the entire suite green, so no test enters the branch; no comment states deck's default.
- BR-28 — not-addressed — main.go:36 comment verbatim; capture_test.go:316 still sets newStore = openStore and five test files use t.TempDir().
- BR-22 — not-addressed — Fifth occurrence (lines 18, 251, 485, 674, 878). Unfixable from tools - see BR-29.
- BR-29 — not-addressed — No ariadne issue exists (checked ariadne/workshop/issues) and this issue's Log references only ariadne#195, a different mechanism.

### Raised

- **BR-30** [Critical] `writes-to-cwd-unignored` A 9.6MB Mach-O arm64 binary is committed at cmd/define/define in the HEAD commit under review
  2nd in family (BR-11; prevalence 2). Do NOT just add a fourth anchored .gitignore line - but DO excise the
  blob before merge, because it is the one open item that becomes irreversible. Measured: git ls-files shows
  it tracked, git log --diff-filter=A dates it to 42cc96d, it is 9616546 bytes against a 99336-byte
  next-largest blob, and .git is currently 9.3M so it roughly doubles the repo for every clone forever. It
  is arm64-only, so a Linux or amd64 checkout gets an unrunnable file; and because it is tracked, cd
  cmd/define && go build now dirties git status on every developer build. The rule covering this and BR-11:
  an artifact the developer workflow drops into the working tree gets an ignore pattern covering EVERY
  directory it can be produced in, added when the tool learns to produce it. .gitignore has been patched
  three times by this rule's absence and each patch was anchored to the one place it had already happened
  (bin/, then /define root-only, then /words/ and /events/); go build in a main package writes the binary
  into that package's directory, which none of them match. Second half, since no ignore would have caught
  this one - it was git add-ed before any ignore existed: read git show --stat before committing, where
  "Bin 0 -> 9616546 bytes" is visible at a glance and does appear in this commit's own stat. Fix while the
  branch is unmerged: git rm --cached cmd/define/define, add an un-anchored pattern, git commit --amend.
- **BR-31** [Important] `prose-contradicts-code` The retracted "a regression in Slug must fail HERE" claim survives in storetest/suite.go and is false by measurement
  8th in family (BR-9, BR-10, BR-17, BR-18, README exit codes, BR-25, BR-28; prevalence 8). Do NOT patch this
  site. storetest/suite.go:140-142 says the traversal guarantee is "asserted here rather than inherited from
  Slug - a regression in Slug must fail HERE, loudly". That is the exact sentence 96adc20 deleted from
  YAML.Forget's doc for BR-25 and the exact claim the same commit rewrote the Done-when to RETRACT; wordFileName's
  own doc now says the opposite. Verified by mutation: regressing Slug entirely (no sanitising) reddens
  TestSlugDoesNotMergeHyphenAndSpace, TestSlugIsAlwaysOneSafePathElement and FuzzSlugIsSafe, and leaves
  Suite/forget cannot escape the words directory GREEN - it discards err, so its assertions pass whether the
  guard is present, absent or bypassed. Second site the same sweep finds: main.go:27, capture.go:50 and
  atlas/define.md:283 each state absolutely that storeCapturer is the only thing that writes to the store,
  while forgetWord's d.deck.Forget at main.go:396 deletes a word file - deps.deck's own comment three lines
  below main.go:27 acknowledges the second mutator. The rule is BR-28's, which is correct and was never
  executed as written: run the mechanical sweep over ALL 14 changed .go/.md files, not the files a finding
  named - storetest/suite.go is in git diff --stat and grepping Slug from that comment lands on the
  contradiction directly.
- **BR-32** [Minor] `needless-indirection` History.Add's found parameter is now dead in both implementations, and orElse is a generic for two nil checks
  4th in family (BR-14, BR-15, BR-21; prevalence 4). Do NOT patch these instances. History.Add(line string,
  found bool) at history.go:16 is ignored by BOTH implementations - history.go:28 and history_store.go:50 both
  take _ bool - while the sole caller computes code == 0 to supply it (replraw.go:171). It existed only
  because storeHistory.Add decided event-vs-deck; #4 moved that decision and left the parameter, which is
  BR-21's rule verbatim. It was missed because it lives in history.go, a file the diff never touched, so
  BR-28's file-based sweep cannot reach it - the extension the rule needs is: sweep the changed files AND the
  interface plus callers of every symbol whose responsibilities the diff moved. Weaker second instance,
  reasonably withdrawn: orElse[T comparable] at main.go:83 is a generic helper introduced by the very commit
  that closed this family, replacing two nil checks with six lines.
- **BR-33** [Minor] `eager-dependency-construction` run builds the store-backed deps before the -forget dispatch, so --forget reads an event log it never uses
  main.go:212 calls d.withStore before the -forget dispatch at :214, so --forget constructs storeHistory and
  loads the whole event log for a result it never consults. Reproduced against a seeded directory: a -forget
  run printed "define: 2026-08-21.yaml: recovered 0 event(s), dropped 1 torn record(s)" for a log it does not
  touch. Stated in round 6's prose and never given an id, which is the mechanism BR-28 named; recording it so
  it can be disposed rather than restated. The rule: a mode dispatch decides which dependencies are needed,
  so it runs before they are constructed.
- **BR-34** [Minor] `misleading-error-text` forgetWord prints a bare "define: %v" where every neighbouring message names the operand
  2nd in family (BR-13; prevalence 2). Do NOT patch this instance alone. main.go:398 formats a store error as
  "define: %v" while its neighbours use "define: %s: %v" with the word. The rule covering this and BR-13: a
  user-facing error names both the condition and the operand it failed on - BR-13's message named the wrong
  condition, this one names no operand.

## Open findings

- **BR-4** [Minor] `unstated-seam-default` The plan calls d.capture but never says deps gains the field, nor what a deps literal without it does
- **BR-22** [Minor] `generated-artifact-noise` The committed close-review artifact opens with a harness stderr preamble
- **BR-26** [Minor] `unstated-seam-default` withStore's early return silently strands deck when history and capture are both supplied
- **BR-28** [Minor] `prose-contradicts-code` main.go:35 claims no test touches the real filesystem, in the file this commit says it swept
- **BR-29** [Minor] `generated-artifact-noise` BR-22 cannot be closed from this repo - the generator lives in the ariadne peer
- **BR-30** [Critical] `writes-to-cwd-unignored` A 9.6MB Mach-O arm64 binary is committed at cmd/define/define in the HEAD commit under review
- **BR-31** [Important] `prose-contradicts-code` The retracted "a regression in Slug must fail HERE" claim survives in storetest/suite.go and is false by measurement
- **BR-32** [Minor] `needless-indirection` History.Add's found parameter is now dead in both implementations, and orElse is a generic for two nil checks
- **BR-33** [Minor] `eager-dependency-construction` run builds the store-backed deps before the -forget dispatch, so --forget reads an event log it never uses
- **BR-34** [Minor] `misleading-error-text` forgetWord prints a bare "define: %v" where every neighbouring message names the operand
