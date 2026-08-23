---
gate: boundary-review
issue: 11
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-22T19:21:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: A stalled stream discards accumulated answer text and returns ErrUnavailable, contradicting the sawEvent rule the same function documents
          detail: |-
            internal/llm/anthropic.go:154 returns Response{} on stall, throwing away every
            byte msg accumulated, and classifies as ErrUnavailable. Fifteen lines below,
            anthropic.go:168 does the opposite for the same situation, and atlas/llm.md:60
            states the preserving behaviour as the contract. Verified with a peer that
            replays the recorded frames through the first text_delta then goes silent:
            onDelta saw "**Obsequious.**\n\nThe clue after" while resp.Text was "" and
            errors.Is(err, ErrTruncated) was false. A Stream caller passing onDelta == nil
            loses the whole partial answer, and a mid-answer hang tells the caller to stop
            trying rather than to skip the question.
          family: stream-failure-classification
          round: 1
        - id: BR-2
          severity: Critical
          title: No test can exercise a stall that occurs after answer text has arrived
          detail: |-
            Fake.serveStream triggers Stall on strings.Contains(fr, "content_block_delta")
            at internal/llm/llmtest/fake.go:328, and the first such frame in
            stream-sample.sse is the thinking_delta at index 0. So TestStreamStallIsDetected
            only ever runs stall-before-any-text, where returning Response{} is
            coincidentally correct. Gate on text_delta the way JunkFrame already does at
            fake.go:322, and assert the partial text survives.
          family: fake-cannot-reach-the-branch
          round: 1
        - id: BR-3
          severity: Important
          title: The fake rejects exactly one model name and it is the test's own fixture string
          detail: |-
            internal/llm/llmtest/fake.go:128 accepts any model with a known prefix unless it
            contains the literal "not-a-real". Verified: changing the fixture to
            claude-opus-6 makes TestUnknownModelIsRejectedLikeTheProxyDoes fail with
            "status = 200, want 502". The fake therefore answers 200 where the measured
            proxy answers 502 for every typo but one, and llmtest.Suite's unknown-model
            obligation passes only because the fixture carries the magic substring
            (ARCH-MOCK). Hold a small set of real model ids instead.
          family: fake-tuned-to-its-fixture
          round: 1
        - id: BR-4
          severity: Important
          title: New defaults Timeout and MaxTokens but not StallAfter, Model or Effort
          detail: |-
            internal/llm/anthropic.go:31. A consumer writing llm.New(llm.Config{BaseURL: u,
            APIKey: k}) silently gets no stall detection, so a hung stream blocks for the
            full 5-minute deadline. StallAfter is the one bound the SDK does not supply and
            the one the constructor omits. Config is this package's whole public input with
            five consumers queued behind it: default all five in New, or none and make
            Resolve the only documented constructor path.
          family: partial-constructor-defaults
          round: 1
        - id: BR-5
          severity: Important
          title: atlas and suite doc comments state a -tags conformance live run that has no file
          detail: |-
            atlas/llm.md:98 and internal/llm/llmtest/suite.go:20 both say Suite runs against
            the live proxy under -tags conformance, and internal/llm/suite_test.go:12 names
            a conformance_test.go. The only conformance-tagged file in the tree is
            cmd/define/fetch_conformance_test.go; the llm one is plan Task 12, deferred to
            M2. The plan is consistent, the atlas is not, and the atlas is the
            current-state map (ARCH-MOCK: the live half of the fake's contract is asserted
            but absent).
          family: docs-claim-absent-surface
          round: 1
        - id: BR-6
          severity: Important
          title: Nothing in go test enforces the captures' required shapes, and the script that does cannot run offline
          detail: |-
            internal/llm/llmtest/testdata/README.md tabulates each capture's required shape
            as "enforced by llm-probe.sh verify", but nothing runs that script and it cannot
            run without the proxy config: K="$(key)" executes unconditionally at
            scripts/llm-probe.sh:20, so CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify
            dies on FileNotFoundError even though verify needs neither key nor network.
            Given fake-models-unobserved-shape produced three plan-gate findings, the
            durable fix is a Go test; llmtest already embeds every capture.
          family: enforcement-not-pinned-by-a-test
          round: 1
        - id: BR-7
          severity: Minor
          title: go mod tidy was never run; the directly-imported SDK is still marked indirect
          detail: |-
            go.mod:11 lists github.com/anthropics/anthropic-sdk-go as // indirect although
            internal/llm/anthropic.go imports it. Plan Task 1 Step 1 said tidy runs at the
            end of Task 5. Verified: tidy promotes it to the direct require block and
            changes nothing else.
          family: stale-module-metadata
          round: 1
        - id: BR-8
          severity: Minor
          title: TestThinkingSignatureIsPreserved discards the error then indexes Blocks[0]
          detail: |-
            internal/llm/anthropic_test.go:90 does got, _ := ...Complete(...) and then
            got.Blocks[0] at :91. When the call errors this panics with index out of range,
            aborting the whole package run and masking every other result. Observed while
            mutation-testing. Check the error.
          family: test-panics-instead-of-failing
          round: 1
        - id: BR-9
          severity: Minor
          title: var _ = http.StatusOK props up an otherwise-unused net/http import
          detail: internal/llm/anthropic_test.go:381. Drop the line and the import.
          family: dead-code
          round: 1
        - id: BR-10
          severity: Minor
          title: response() has an unused receiver, so the package's only transformation logic is reachable only through httptest
          detail: |-
            internal/llm/anthropic.go:185 never references a. Promoting it to a package-level
            function would let block assembly and Text-joining be unit-tested directly from a
            capture via json.Unmarshal, with no server (ARCH-PURE).
          family: pure-logic-on-an-io-type
          round: 1
        - id: BR-11
          severity: Minor
          title: serveStream ignores Reply.Text and Reply.Stop entirely
          detail: |-
            internal/llm/llmtest/fake.go:305. f.Script("x", Reply{Text: "…", Stop: "refusal"})
            silently does nothing on a streaming request — it always replays the capture.
            Either honour them or fail loudly on the combination.
          family: fake-silently-ignores-inputs
          round: 1
        - id: BR-12
          severity: Minor
          title: Capture calls t.Fatalf from the fake's server goroutine
          detail: |-
            internal/llm/llmtest/fake.go:311. A missing capture, or a request arriving after
            cleanup, becomes a hang or a "log after test completed" panic rather than a clean
            failure. Resolve captures at Script/ServeRecorded time instead.
          family: test-helper-fatal-off-goroutine
          round: 1
        - id: BR-13
          severity: Minor
          title: config.go references --llm-check, which does not exist until M2
          detail: |-
            internal/llm/config.go:99. Also io := []byte(...) shadows the io package at
            internal/llm/anthropic_test.go:349.
          family: comment-references-future-surface
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-08-22T19:45:37-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Mutation-verified — reverting the stall branch to Response{}+ErrUnavailable turns TestStreamStallAfterTextSalvagesAndTruncates red.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Mutation-verified — re-gating Stall on content_block_delta turns the same test red at "no deltas were delivered".
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Behavior verified: claude-opus-6 / -typo / -sonnet-9 all get 502. Not pinned by a test — rolled into the enforcement-not-pinned-by-a-test repeat.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: New now defaults all six fields. Side effects (unpinned; "Zero disables it" now unreachable) raised as new findings.
          round: 2
        - id: BR-5
          disposition: addressed
          note: conformance_test.go exists, vets under -tags conformance, and skips cleanly with no key configured.
          round: 2
        - id: BR-6
          disposition: addressed
          note: captures_test.go runs offline in go test; CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify exits 0.
          round: 2
        - id: BR-7
          disposition: addressed
          note: SDK is in the direct require block; go mod tidy produces no diff.
          round: 2
        - id: BR-8
          disposition: addressed
          note: Both tests now check the error and length-check before indexing Blocks.
          round: 2
        - id: BR-9
          disposition: addressed
          note: var _ = http.StatusOK and the net/http import are gone from anthropic_test.go.
          round: 2
        - id: BR-10
          disposition: addressed
          note: buildResponse is package-level with no receiver. The doc comment's claim of a direct unit test is false — rolled into the docs-claim repeat.
          round: 2
        - id: BR-11
          disposition: addressed
          note: serveStream fails loudly; verified reachable with a scratch test. No fixture enters it — rolled into the enforcement repeat.
          round: 2
        - id: BR-12
          disposition: addressed
          note: Both capture-miss paths return 500 instead of Fatalf. A new off-goroutine Fatalf was added by BR-11's fix — raised separately.
          round: 2
        - id: BR-13
          disposition: addressed
          note: config.go:99 cleared; llm.go:110 and errors.go:54 still carry the reference — raised as the family repeat.
          round: 2
      findings:
        - id: BR-14
          severity: Important
          title: An exhausted 408 classifies as ErrRequest, the loud "our bug" arm, though the SDK retries it as transient
          detail: |-
            internal/llm/errors.go:44-52 routes 429 and >=500 to ErrUnavailable and everything
            else to default/ErrRequest, so 408 Request Timeout and 409 Conflict land loud.
            Measured end-to-end through the real client against the fake: status 408 ->
            "llm: bad request", ErrUnavailable=false, after 3 attempts — the SDK retried it,
            which is the transport itself calling it transient, and then the taxonomy calls it
            our bad schema/model/body. errors_test.go covers neither status. Add
            http.StatusRequestTimeout to the ErrUnavailable arm, decide 409 deliberately, and
            add both rows to the table.
          family: unclassified-failure-mode
          round: 2
        - id: BR-15
          severity: Important
          title: Three of this round's fixes survive full reversion with a green suite, or are entered by no fixture
          detail: |-
            This is the 2nd finding in family enforcement-not-pinned-by-a-test. Do NOT fix these
            three instances — the rule is: a boundary fix that changes behavior is not complete
            until a test fails against the pre-fix code, and the fixture must not encode the
            pre-fix implementation's escape hatch. Measured prevalence 3, all verified this
            round. BR-3 (fake.go:148): reverting knownModel to the prefix + "not-a-real"
            substring rule leaves go test ./internal/llm/... green, because the fixture
            claude-not-a-real-model still carries the magic substring. BR-4 (anthropic.go:39):
            deleting the StallAfter default from New leaves the suite green — every stall test
            sets it explicitly. BR-11 (fake.go:342): no test scripts Reply{Text} against a
            streaming request, so the loud branch is reachable but unexercised.
          family: enforcement-not-pinned-by-a-test
          round: 2
        - id: BR-16
          severity: Important
          title: Four doc claims outrun the tree, two of them written by this round's own fixes
          detail: |-
            This is the 2nd finding in family docs-claim-absent-surface. Do NOT fix these four
            sites individually — the rule is: a claim in a doc comment or atlas page is a claim
            about the tree, and the sweep must check each against the tree rather than against
            the intent of the change that wrote it. Measured prevalence 4. (1) anthropic.go:203
            says buildResponse "is now unit-tested directly from a committed capture with
            json.Unmarshal and no server at all" — grep finds no caller outside anthropic.go,
            and it is unexported while anthropic_test.go is package llm_test. (2) config.go:43
            says StallAfter "Zero disables it", but New — the only constructor — overrides zero
            with 90s; the plan repeats the sentence at plan:837. (3) llm.go:130-136 documents
            Progress phases connect|waiting|streaming|done, but only waiting and streaming are
            ever emitted, and Progress.Bytes is assigned nowhere. (4) fake.go:104 says
            Reply.Capture "wins over the rest", but serve checks reply.Status first at
            fake.go:256.
          family: docs-claim-absent-surface
          round: 2
        - id: BR-17
          severity: Minor
          title: BR-11's fix added a t.Fatalf on the fake's server goroutine, ten lines above the comment forbidding it
          detail: |-
            This is the 2nd finding in family test-helper-fatal-off-goroutine. Do NOT fix this
            instance — state the rule (no t.Fatalf/FailNow from a handler goroutine; resolve at
            Script time or answer with a 500) and fix that. fake.go:343 calls f.t.Fatalf inside
            serveStream while fake.go:352 says "Same reason as serveJSON: no Fatalf off the test
            goroutine". Verified: it does record the failure today, but a request arriving after
            cleanup still panics with "log after test completed".
          family: test-helper-fatal-off-goroutine
          round: 2
        - id: BR-18
          severity: Minor
          title: --llm-check was removed from config.go and left in two sibling files
          detail: |-
            This is the 2nd finding in family comment-references-future-surface. Do NOT fix this
            instance — state the rule covering forward references to unbuilt surface and sweep
            the enumeration. Remaining: llm.go:110 and errors.go:54. Measured prevalence 2 after
            the round that closed the one site the finding named (ARCH-PURPOSE: instance, not
            class).
          family: comment-references-future-surface
          round: 2
        - id: BR-19
          severity: Minor
          title: captures_test.go indexes into unchecked type assertions, so a malformed capture panics rather than fails
          detail: |-
            This is the 2nd finding in family test-panics-instead-of-failing. Do NOT fix this
            instance — state the rule and sweep. internal/llm/llmtest/captures_test.go:29-31 does
            m["content"].([]any) and c.(map[string]any)["type"].(string), and :40 does
            cm["text"].(string). A capture missing content aborts the whole package run. This
            instance was introduced by the fix round that closed BR-8.
          family: test-panics-instead-of-failing
          round: 2
        - id: BR-20
          severity: Minor
          title: orDefault, orInt64 and orDuration re-implement stdlib cmp.Or
          detail: |-
            anthropic.go:294-306 and config.go:96. cmp.Or[T comparable](vals ...T) T has been in
            the standard library since Go 1.22 and the module targets 1.26, so one import
            replaces three near-identical helpers (ARCH-DRY).
          family: stdlib-reimplemented
          round: 2
        - id: BR-21
          severity: Minor
          title: Production comments increasingly narrate prior review rounds, duplicating the gate ledger
          detail: |-
            The fix commit added 110 comment lines against 230 code lines. Examples:
            anthropic.go:32-37 and :164-169, fake.go:117-121 and :142-147 all recount what "the
            first version" did. That history already lives in the gate ledger and the issue Log,
            and it means a behavior change now needs three edits to stay honest. llm.go is 53%
            comment, errors.go 48%. Round 1 flagged density as worth watching; it rose.
          family: review-archaeology-in-code
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-08-22T20:04:10-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: Fixed at the cause — transient() derives from the SDK retry policy; reverting reddens the 408 and 409 rows.
          round: 3
        - id: BR-15
          disposition: addressed
          note: All three reversions now red; TestNewDefaultsEveryConfigField verified against two different deleted defaults, not just StallAfter.
          round: 3
        - id: BR-16
          disposition: not-addressed
          note: 'Three claims still false: fake.go:107 "wins over the rest" (Status wins, measured), plan:837, plan:345-346 — two cited by line in the finding.'
          round: 3
        - id: BR-17
          disposition: addressed
          note: No t.Fatalf remains on any handler goroutine; the rule is stated in serveStream and both capture-miss paths answer with a status.
          round: 3
        - id: BR-18
          disposition: addressed
          note: Swept to zero — no --llm-check reference anywhere in internal/, cmd/, atlas/ or scripts/.
          round: 3
        - id: BR-19
          disposition: addressed
          note: captures_test.go checks every assertion and t.Fatalf's; the rule is stated at the top of the helper.
          round: 3
        - id: BR-20
          disposition: addressed
          note: cmp.Or replaces all three helpers. Two hand-rolled contains helpers survived the sweep — raised as the family repeat.
          round: 3
        - id: BR-21
          disposition: addressed
          note: Density fell (llm.go 53->48%, errors.go 48->39%); only three archaeology sites remain and all three are rationale.
          round: 3
      findings:
        - id: BR-22
          severity: Important
          title: Three of the transport's own named invariants can be deleted outright with the whole suite still green
          detail: |-
            This is the 2nd finding in family fake-cannot-reach-the-branch. Do NOT fix these
            four instances — the rule is: a behaviour the code singles out as load-bearing
            needs a fixture that separates it from the alternative it warns against, and
            where no committed capture exhibits that shape the fixture is constructed,
            because block count and header presence are transport shape rather than
            judgment. Measured prevalence 4, three verified by reversion against
            go test ./internal/llm/... this round. (1) anthropic.go:218-228 — all three
            captures carry exactly one text block, so writing only the first leaves the
            suite green, though "every text block JOINED, not Content[0].Text" is what the
            atlas, the package doc and Response.Text all single out. (2) anthropic.go:62-64
            — deleting the r.System to p.System plumbing leaves the suite green;
            Recorded.System() exists at fake.go:59 for exactly this assertion and has zero
            callers, and M2's Task[T] sets System on every call. (3) anthropic.go:262-284 —
            neutering watch() so OnSlow never fires leaves the suite green; grep for
            OnSlow or Progress across _test.go returns nothing, and plan:1383 lists
            TestOnSlowNamesThePhase undelivered. (4) Response.StopDetails has zero test
            references and the fake's refusal reply emits no stop_details, so the populated
            branch is unreachable from either side of the seam.
          family: fake-cannot-reach-the-branch
          round: 3
        - id: BR-23
          severity: Minor
          title: Two hand-rolled contains helpers survived the round that replaced the or* helpers with cmp.Or
          detail: |-
            This is the 2nd finding in family stdlib-reimplemented. Do NOT fix these two
            instances — state the rule (a helper duplicating a stdlib function is removed,
            and the sweep covers _test.go as well as production files) and fix that.
            Measured prevalence 2. errors_test.go:91 re-implements strings.Contains with an
            index loop; captures_test.go:130 re-implements slices.Contains. Both are in the
            same two packages as the cmp.Or fix and both were skipped because the sweep
            scoped itself to production code.
          family: stdlib-reimplemented
          round: 3
        - id: BR-24
          severity: Minor
          title: Fake.t is assigned in NewFake and read by nothing after BR-17 removed its last consumer
          detail: |-
            This is the 2nd finding in family dead-code. Do NOT fix this instance — the rule
            is: when a fix removes the last reader of a field, import or helper, the same
            edit removes the field. internal/llm/llmtest/fake.go:174 declares t *testing.T
            and fake.go:184 assigns it; grep for f.t across the package returns nothing now
            that serveStream answers with a status instead of calling Fatalf. go vet does
            not flag unused struct fields, so nothing else will catch this class.
          family: dead-code
          round: 3
        - id: BR-25
          severity: Minor
          title: An unknown-model request pops the matcher queue and then discards the reply it drew
          detail: |-
            This is the 2nd finding in family fake-silently-ignores-inputs. Do NOT fix this
            instance — the rule is: the fake must not mutate scripted state on a path that
            does not serve the scripted reply. internal/llm/llmtest/fake.go:233 calls
            f.next(rec.Prompt()) before the unknown-model check at fake.go:243 returns 502,
            so a queued Reply is consumed and thrown away. Harmless today because no test
            scripts a sequence against a bad model; a trap for M2's cassette sequences,
            where the next call would silently draw the wrong queue entry.
          family: fake-silently-ignores-inputs
          round: 3
        - id: BR-26
          severity: Minor
          title: The plan was corrected in place at the M1 boundary with no appended Revisions entry
          detail: |-
            workshop/plans/000011-vocab-llm-plan.md carries four dated Revisions entries,
            none for the boundary-review rounds, while the Robustness bar was rewritten
            in place ("Corrected in place, M1 boundary review (C1)") and measured fact 2
            likewise. AGENTS.md section 1 requires appending a Revisions entry with
            timestamp, reason and delta rather than overwriting, so the pre-review claim
            survives. Round 2 recommended this in its advisory section, where it was never
            tracked as a finding and therefore never disposed.
          family: plan-revision-not-appended
          round: 3
      boundary: M1
      blocked: true
    - "n": 4
      timestamp: "2026-08-22T20:26:51-07:00"
      agent: claude
      dispose:
        - id: BR-16
          disposition: addressed
          note: All four cited sites plus round 3's three re-cited sites verified correct against the tree; one uncited instance survives at plan:516 and is raised as the family repeat.
          round: 4
        - id: BR-22
          disposition: addressed
          note: All four instances pinned by constructed fixtures; each verified red by reversion this round against go test ./internal/llm/...
          round: 4
        - id: BR-23
          disposition: addressed
          note: Both hand-rolled contains helpers gone; errors_test.go:87 and captures_test.go:70,123 now use strings.Contains / slices.Contains.
          round: 4
        - id: BR-24
          disposition: addressed
          note: Fake.t is gone from the struct and from NewFake; grep for a testing.T field in fake.go returns only the Capture and NewFake parameters.
          round: 4
        - id: BR-25
          disposition: addressed
          note: f.next() now runs after the unknown-model check, with the rationale recorded at the call site.
          round: 4
        - id: BR-26
          disposition: addressed
          note: A fifth Revisions entry ("M1 boundary review, three rounds") is appended at plan:1986 with the rules the rounds produced.
          round: 4
      findings:
        - id: BR-27
          severity: Important
          title: A negative SlowEvery panics on the watcher goroutine, where no caller can recover
          detail: |-
            This is the 2nd finding in family partial-constructor-defaults. Do NOT fix
            this site — the rule is: New must give every Config field a defined meaning
            for every value it can hold, and sibling duration fields must not disagree
            about what a sentinel means. Measured prevalence 3, all verified this round.
            llm.New(Config{BaseURL, APIKey, SlowEvery: -1, OnSlow: f}) panics
            "non-positive interval for NewTicker" at anthropic.go:270 on a goroutine the
            caller cannot recover from, killing the process. Negative Timeout instead
            makes every call return ErrUnavailable instantly — silently absorbed rather
            than reported as misconfiguration. Only StallAfter has a documented negative
            meaning (config.go:52-55 says disabling REQUIRES a negative value), which is
            what invites a caller to try the same on SlowEvery, documented at config.go:59
            as "Zero takes the default" and nothing more. TestNewDefaultsEveryConfigField
            asserts non-zero, so -1 passes it.
          family: partial-constructor-defaults
          round: 4
        - id: BR-28
          severity: Important
          title: A .json capture scripted against a streaming request is silently replaced by stream-sample.sse
          detail: |-
            This is the 3rd finding in family fake-silently-ignores-inputs. Do NOT fix
            this site — the rule is: every Reply field that cannot be served on the path
            a request took is a caller mistake and answers with a 400 naming the field,
            and the sweep covers the Reply struct field by field rather than the one
            branch a finding named. internal/llm/llmtest/fake.go:403 falls back to
            stream-sample.sse for any Capture not ending in .sse. Measured: scripting
            Reply{Capture: "message-truncated.json"} against Stream returns err=nil,
            Stop="end_turn" and the Obsequious text from stream-sample.sse — a different
            capture with a different stop_reason than the one asked for. This is the
            un-swept half of BR-11's own enumeration: the adjacent Reply{Text/Stop}
            branch was made loud this round at fake.go:381, ten lines above. M2's
            cassette sequences run through this path.
          family: fake-silently-ignores-inputs
          round: 4
        - id: BR-29
          severity: Minor
          title: TestNegativeStallAfterDisablesTheBound survives full reversion of the behaviour it names
          detail: |-
            This is the 3rd finding in family enforcement-not-pinned-by-a-test. Do NOT
            fix this site — the rule the family never enumerated is: reversion-check the
            TEST you add, not only the fix it pins, because a bound dominated by a second
            bound distinguishes nothing. Verified: changing New to
            "if c.StallAfter <= 0 { c.StallAfter = defaultStallAfter }" reddens only
            TestNewPreservesADisabledStallBound; anthropic_test.go:444 stays green,
            because its assertion (elapsed >= 1s against Timeout 2s) holds whether the
            stall bound is disabled or defaulted to 90s — the total deadline ends the
            call at ~2s either way. Its comment claims it makes the documented escape
            hatch reachable and tested. Prevalence 1 of the 6 tests added this round; the
            other 5 were each verified red.
          family: enforcement-not-pinned-by-a-test
          round: 4
        - id: BR-30
          severity: Minor
          title: splitInto chops on byte offsets, so any multibyte scripted text round-trips corrupted
          detail: |-
            This is the 2nd finding in family fake-tuned-to-its-fixture. Do NOT fix this
            site — the rule is: a constructed fixture helper must be correct for the
            inputs the committed captures actually contain, not only for the ASCII string
            its first caller passes. internal/llm/llmtest/fake.go:162 slices on len(s)/n,
            so a rune spanning a boundary is cut and json.Marshal substitutes U+FFFD.
            Measured: Reply{Text: "obsequieux — tres flagorneur, vraiment", SplitText: 3}
            comes back with the em-dash replaced by three U+FFFD, so Response.Text is not
            what was scripted. TestTextJoinsEveryTextBlock passes only because its
            fixture is ASCII; every committed capture contains em-dashes. Introduced by
            this round's own BR-22.1 fix.
          family: fake-tuned-to-its-fixture
          round: 4
        - id: BR-31
          severity: Minor
          title: The plan's Task 1 contract block still declares four Progress phases and a Bytes field
          detail: |-
            This is the 3rd finding in family docs-claim-absent-surface. Do NOT fix this
            site — the rule needs widening: the sweep's enumeration includes embedded
            code blocks in plan artifacts, not only prose and production files.
            workshop/plans/000011-vocab-llm-plan.md:516-518 declares
            Phase string // "connect" | "waiting" | "streaming" | "done" and Bytes int,
            which llm.go corrected to two phases with Bytes removed and which the same
            document's own Revisions entry at plan:2040 declares gone. Round 3's sweep
            fixed the plan's prose (plan:837, plan:345-346, both verified correct now)
            and stopped before its code blocks — which Task 1 Step 3 explicitly warns
            "get pasted verbatim", the reason a stale one is a hazard rather than a typo.
          family: docs-claim-absent-surface
          round: 4
        - id: BR-32
          severity: Minor
          title: Reply.Body and Reply.NoThinking are documented knobs that no fixture in the tree turns
          detail: |-
            This is the 3rd finding in family dead-code. Do NOT fix these two instances —
            the rule needs widening from "a field whose last reader a fix removed" to
            "the sweep enumerates the fake's exported surface, and a knob no fixture
            turns is dead the same as an unread field." internal/llm/llmtest/fake.go:114
            (Body) and :118 (NoThinking) each gate a production branch in the fake
            (fake.go:289 and :339) and neither is set anywhere in the tree, so both
            branches are reachable but unexercised — the shape BR-15c named. Round 3
            identified both in its prose section 5 and never raised them, so they were
            never tracked or disposed.
          family: dead-code
          round: 4
      boundary: M1
      blocked: false
    - "n": 5
      timestamp: "2026-08-23T00:06:09-07:00"
      agent: claude
      findings:
        - id: BR-33
          severity: Critical
          title: decode returns a partially populated T with a nil error when a required field is missing
          detail: |-
            Measured against the shipped code: decode[answer](`{"fits":true}`) returns {Fits:true Reason:""} with err=nil; `{}` and `null` return the zero value with err=nil. task.go:59 states "it never returns a partially populated value with a nil error", atlas/llm.md:112 states "reject missing ones", and plan Task 10 states "require every schema-required field present". SchemaFor already emits "required":["fits","reason"] — the single source declares it and decode ignores it. #12's veto would read Fits:false from `{}` and silently drop a distractor rather than skip the question.
            THIS IS THE 4TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not fix only this site. The RULE: an invariant stated in a doc comment must be asserted by a test that goes red when it is violated — including the SUCCESS branch of a property/fuzz target, which is where FuzzDecode returns early (`if err == nil { return }`) and where its own seed `"{}"` is a violating input the target waves through. THE ENUMERATION to sweep in this round: grep every "never", "always", "must", "either ... or" claim in the doc comments added by this window (task.go decode, task.go Run, render.go renderRequest/renderSchema, schema.go SchemaFor, cassette.go Cassette/Stream, golden.go AssertGolden, fake.go next) and for each confirm a test that fails when the claim is broken; where none exists, either write it or delete the claim.
          family: enforcement-not-pinned-by-a-test
          round: 5
        - id: BR-34
          severity: Important
          title: Four doc claims in this window assert properties the code does not hold
          detail: |-
            Measured prevalence, 4 instances in one milestone. (1) atlas/llm.md:112 "allow unknown fields, reject missing ones" — decode does not reject missing (see the Critical). (2) atlas/llm.md:114 "Its invariant ... is held by a fuzz target" — FuzzDecode asserts only the error half. (3) atlas/llm.md:171 "Every failure names scripts/llm-probe.sh record" and capture_conformance_test.go:22 "On drift the failure names the fix" — 2 of 7 failure messages in that file name it; the unknown-block-type, no-text-block, output_config-decode, no-deltas and preamble messages do not. (4) internal/llm/schema.go:15 "llmtest.Golden snapshots it, so a struct field added without thought shows up in a diff" — no golden file exists anywhere in the tree, and `llmtest.Golden` is not an identifier (it is `AssertGolden`).
            THIS IS THE 4TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not patch the four sentences. The RULE: a doc claim of UNIVERSAL form ("every", "always", "never", "is held by") is a claim about an enumeration, so it may only be written after enumerating the sites and checking each — and a doc claim naming a code identifier or an on-disk artifact must be grep-verified against the tree in the same edit. Sweep: for each universal claim in atlas/llm.md and in the doc comments of render.go, schema.go, task.go, cassette.go and golden.go, run the enumeration it implies and either make it true or weaken it to what is true.
          family: docs-claim-absent-surface
          round: 5
        - id: BR-35
          severity: Important
          title: Plan Task 9's schema golden was never written, so AssertGolden ships with zero committed artifacts
          detail: 'internal/llm/testdata/ does not exist; `find` returns no golden/ or cassettes/ directory in the repo; the only callers of AssertGolden and Cassettes are their own self-tests against t.TempDir(). Task 9 required a snapshot AND a byte-identical assertion, and schema.go:15 cites that snapshot as the property that justifies reflecting the schema instead of hand-writing it. ARCH-PURPOSE shadow-sweep on the SchemaFor[T] single source: 4 consumers, 2 derive (Request.Schema on the wire, RequestHash), 2 do not (no golden; decode ignores the required list).'
          family: single-source-consumer-not-derived
          round: 5
        - id: BR-36
          severity: Important
          title: README.md is not updated for the new --llm-check flag or its exit code
          detail: '`grep -n "llm" README.md` returns nothing. README documents --sound/-times, -locale, -raw, --forget and DEFINE_NO_CAPTURE, and its exit-code paragraph enumerates what `1` means ("no dictionary entry, or --forget found nothing to remove") — --llm-check now also exits 1, for a third reason. main.go''s usage text was updated; README was not.'
          family: user-surface-undocumented
          round: 5
        - id: BR-37
          severity: Important
          title: The cassette double replaces llm.Client, bypassing the SDK path the plan places it beneath
          detail: 'Plan Task 4 Step 3 specifies `func Cassette(t *testing.T, r llm.Request) Reply` — a body served through the wire Fake. Shipped is `func (c *Cassette) Client(live llm.Client) llm.Client` (cassette.go:47): replay reads the file and returns rec.Response without serialising a Request, without the SDK, without SSE. llmtest''s own package doc (fake.go:5) argues that a stubbed Client cannot see a mis-serialized output_config, a dropped anthropic-version header, or a retry that re-sends a consumed body — a consumer test written against a cassette now cannot see any of them. cassetteClient.Stream (cassette.go:101) does not stream: it calls Complete and fires a single delta. ARCH-MOCK: production flow and test flow no longer share the same boundary on this path.'
          family: double-above-the-seam
          round: 5
        - id: BR-38
          severity: Important
          title: Cassette replay collapses a recorded ErrRequest into ErrUnavailable
          detail: 'recorded.err() (cassette.go:122) stores the error as a string and re-derives the taxonomy from Stop. A recorded 400 — bad schema or unknown model, the class errors.go:20 says must stay LOUD — carries Stop:"" , so ErrorForStop returns nil and the replay falls through to `fmt.Errorf("%w: %s", llm.ErrUnavailable, r.Err)`, the quiet class every consumer degrades on. Recording the taxonomy member itself (or the HTTP status) instead of re-deriving it removes the second classification path.'
          family: double-rederives-error-taxonomy
          round: 5
        - id: BR-39
          severity: Important
          title: The capture-drift conformance suite reports drift when the proxy is merely unreachable
          detail: 'capture_conformance_test.go:31 skips only when llm.Resolve fails (no key). With a key set and the proxy stopped, Complete returns ErrUnavailable and every subtest t.Fatalf''s — telling the operator "the model may no longer think by default ... Re-record: scripts/llm-probe.sh record" when nothing drifted. Plan Task 8 Step 2 asked for skip-not-fail explicitly. THIS IS THE 2ND FINDING IN FAMILY `unclassified-failure-mode`: the rule is that a check must distinguish "dependency unreachable" from "dependency changed" before reporting either, and internal/llm/conformance_test.go:30 has the same gap, so fix both.'
          family: unclassified-failure-mode
          round: 5
        - id: BR-40
          severity: Important
          title: Both test doubles added this window discard the llm.Request entirely
          detail: |-
            checkClient (cmd/define/llmcheck_test.go:14) and stubLive (internal/llm/llmtest/cassette_test.go:17) both take `context.Context, llm.Request` and name neither parameter. Consequence: nothing asserts that --llm-check's request is well-formed — MaxTokens:2048, Task:"llm-check" and the PONG prompt could all be dropped and the four llmcheck tests stay green, while the real flag 400s. The repo already has a wire fake (llmtest.NewFake + llm.New) that would catch it and is importable from cmd/define.
            THIS IS THE 4TH FINDING IN FAMILY `fake-silently-ignores-inputs`. The RULE: a double must either record its input for assertion or be replaced by the wire-level fake that already exists for that dependency; a double whose method signature discards its request parameter cannot fail for any reason related to what was asked. Sweep every type in the tree implementing llm.Client, cmd/define's fetch seam, and the store seams, and confirm each records or asserts its input.
          family: fake-silently-ignores-inputs
          round: 5
        - id: BR-41
          severity: Important
          title: The plan still describes an M2 design that was not built, with no Revisions entry
          detail: |-
            Undeclared deltas: Cassette's seam, API and storage path all changed (see the double-above-the-seam finding); the flag is -update, not -record; Task 9's golden snapshot was dropped; the Integration points table names `llmtest.Golden` where the identifier is `AssertGolden` and has no row for llmtest.Cassette or for renderRequest/RequestHash (internal/llm/render.go), both delivered.
            THIS IS THE 2ND FINDING IN FAMILY `plan-revision-not-appended`. The rule per AGENTS.md section 1: any divergence from a plan artifact discovered during implementation is appended as a timestamped `## Revisions` delta in the SAME commit that diverges — so the sweep is not "add one entry now" but "diff the plan's Core concepts and Task lists against the tree at each milestone close and append what moved".
          family: plan-revision-not-appended
          round: 5
        - id: BR-42
          severity: Minor
          title: '`_ = llmtest.Capture` exists only to keep an import alive'
          detail: |-
            capture_conformance_test.go:132. The comment calls it "the committed artifacts this run is checking", but the statement checks nothing — llmtest is otherwise unused in the file.
            THIS IS THE 4TH FINDING IN FAMILY `dead-code`. The RULE: a statement whose only effect is to satisfy the compiler is not documentation — drop the import and put the sentence in the doc comment, or make the reference load-bearing (here: read the capture and compare a field against the live response, which is what the file claims to do).
          family: dead-code
          round: 5
        - id: BR-43
          severity: Minor
          title: TestAQueueStillAdvancesWhileItHasEntries duplicates TestQueueServesInOrder
          detail: 'fake_test.go:163 vs fake_test.go:80 — same script shape (429 then a text reply), same two assertions, different string literals. Verified: with the sticky change reverted, the new test still passes, so it pins nothing the older one does not. ARCH-DRY.'
          family: redundant-test-duplicates-existing
          round: 5
        - id: BR-44
          severity: Minor
          title: '`define -llm-check <word>` silently ignores the word'
          detail: 'main.go:279 returns before the arity switch. The comment immediately above calls --llm-check "a mode, like --forget", but --forget has an explicit guard (main.go:288: "-forget takes the word to remove; do not also pass one") added because "silently honouring one of them is how -raw came to mean two different things in #2". Same guard, same reason.'
          family: mode-flag-arity-guard
          round: 5
        - id: BR-45
          severity: Minor
          title: --llm-check hardcodes MaxTokens 2048 instead of the resolved cfg.MaxTokens
          detail: llmcheck.go:44. config.go:23 records that an under-budgeted max_tokens let adaptive thinking consume the whole allowance and returned an answer cut mid-rune — the committed message-truncated.json. A truncated PONG surfaces as ErrTruncated and exits 1, so the diagnostic would report a healthy configuration as broken.
          family: diagnostic-ignores-config
          round: 5
        - id: BR-46
          severity: Minor
          title: additionalProperties:false is set unconditionally, including on non-object schemas
          detail: 'schema.go:66. Measured: SchemaFor[string]() -> {"type":"string","additionalProperties":false}; SchemaFor[map[string]string]() -> {"type":"object","additionalProperties":false}, an object that permits no keys at all. The adjacent comment justifies stripping $schema/$id because "the provider rejects a schema carrying JSON Schema metadata it does not use" — the same argument applies to additionalProperties on a string or array.'
          family: schema-metadata-applied-blindly
          round: 5
        - id: BR-47
          severity: Minor
          title: Cassette.Client's t.Fatalf fires from whatever goroutine a consumer calls Complete on
          detail: |-
            cassette.go:60/66/74/80/84 call c.store.t.Fatalf from inside an llm.Client, a value designed to be handed to arbitrary consumer code including concurrent authoring loops. fake.go:326 states the opposing rule for the same package ("NOT t.Fatalf: this runs on the server's goroutine, where Fatalf becomes a hang or a 'log after test completed' panic").
            THIS IS THE 3RD FINDING IN FAMILY `test-helper-fatal-off-goroutine`. The RULE: a helper may call t.Fatalf only if it is structurally guaranteed to run on the test goroutine; anything returned to a caller as a value (a Client, a handler, a callback) must return an error instead. Enumerate every exported llmtest constructor that captures *testing.T and classify each by that criterion.
          family: test-helper-fatal-off-goroutine
          round: 5
        - id: BR-48
          severity: Minor
          title: TestCassetteReplaysTheTaxonomy sets *update without a defer
          detail: cassette_test.go:107-112 does `*update = true` ... `*update = false` inline; a Fatal in the Complete call between them leaks -update into every subsequent test in the package, turning AssertGolden from a comparator into a writer. recordThenReplay (cassette_test.go:34) uses defer correctly; this site does not.
          family: test-flag-mutation-leaks
          round: 5
        - id: BR-49
          severity: Minor
          title: A cassette records the answer but not the question
          detail: recorded (cassette.go:117) stores only Response and an error string; the filename carries task plus a 12-hex hash. TestCassetteOnDiskIsReadable asserts the ANSWER is legible in a diff, but a reviewer cannot tell what was asked without recomputing the hash. Storing llm.RenderRequest(r) alongside would make the artifact self-describing, and is free — the miss message already renders it.
          family: artifact-omits-the-question
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-08-23T00:28:07-07:00"
      agent: claude
      dispose:
        - id: BR-33
          disposition: addressed
          note: 'Reversion-verified: removing requireSchemaFields reddens 3 table cases and FuzzDecode seed#1. See new finding for the depth-1 limit.'
          round: 6
        - id: BR-34
          disposition: addressed
          note: All four claims now hold; 7 of 7 drift failures name the script. render.go's claims are re-raised as a new finding, not this one.
          round: 6
        - id: BR-35
          disposition: addressed
          note: testdata/golden/schema-veto-verdict.txt committed; adding a struct field to vetoVerdict reddens TestSchemaGoldenIsStable.
          round: 6
        - id: BR-36
          disposition: addressed
          note: README has a "Checking the model connection" section and the exit-code paragraph names --llm-check.
          round: 6
        - id: BR-37
          disposition: addressed
          note: Rebuilt as http.RoundTripper under Config.Transport; replay against 127.0.0.1:1 proves the SDK path still runs.
          round: 6
        - id: BR-38
          disposition: addressed
          note: 'Reversion-verified: hardcoding jsonResponse(200, …) reddens both subtests of TestCassetteReplaysTheTaxonomyFromTheStatus.'
          round: 6
        - id: BR-39
          disposition: not-addressed
          note: capture_conformance_test.go fixed; internal/llm/conformance_test.go:30, named in the finding, is unchanged.
          round: 6
        - id: BR-40
          disposition: addressed
          note: 'Reversion-verified: mutating MaxTokens to 512 and the PONG prompt each redden a distinct assertion.'
          round: 6
        - id: BR-41
          disposition: addressed
          note: Revisions entry appended and both tables corrected; Task 11's prose was missed and is covered by the new coupling finding.
          round: 6
        - id: BR-42
          disposition: not-addressed
          note: '`_ = llmtest.Capture` still present at capture_conformance_test.go:142.'
          round: 6
        - id: BR-43
          disposition: not-addressed
          note: TestAQueueStillAdvancesWhileItHasEntries still present and still the same shape as TestQueueServesInOrder (fake_test.go:80).
          round: 6
        - id: BR-44
          disposition: not-addressed
          note: main.go still returns on *llmCheck before the arity switch; `define -llm-check hello` ignores the word.
          round: 6
        - id: BR-45
          disposition: not-addressed
          note: 'llmcheck.go:40 still hardcodes MaxTokens: 2048 rather than reading cfg.MaxTokens.'
          round: 6
        - id: BR-46
          disposition: not-addressed
          note: schema.go:71 still sets additionalProperties:false unconditionally; SchemaFor[string]() still returns it on a string schema.
          round: 6
        - id: BR-47
          disposition: addressed
          note: No t.Fatalf remains in cassette.go; the enumeration holds — every other Fatalf in llmtest is on the test goroutine.
          round: 6
        - id: BR-48
          disposition: addressed
          note: Defer added; the restore-to-literal residual is raised as a new finding in the same family rather than re-raised here.
          round: 6
        - id: BR-49
          disposition: addressed
          note: exchange.Request stores the wire body; TestCassetteOnDiskIsSelfDescribing asserts the question is legible.
          round: 6
      findings:
        - id: BR-50
          severity: Important
          title: The cassette no longer derives from renderRequest, leaving RequestHash dead and five artifacts asserting a coupling that is measurably false
          detail: |-
            Measured: two Requests differing only in Task recorded to ONE cassette file (f3f94dfd72ba.json) while RequestHash(a)=c94ee5919d42, RequestHash(b)=d1d164764fc2 and RenderRequest differs — the second recording silently overwrote the first. cassette.go:51 keys on sha256(wire body minus max_tokens); renderRequest is no longer its input. RequestHash has zero non-test callers yet render.go:57 says "RequestHash keys a cassette"; render.go:15 says "llmtest.Golden prints it … and llmtest.Cassette hashes it to key a recording" (llmtest.Golden is still not an identifier, in the very file BR-34's rule named for grep-verification); golden.go:28, atlas/llm.md:140 and plan Task 11 repeat the claim; and TestGoldenAndCassetteKeyMoveTogether plus TestEveryMeaningfulFieldReachesTheHash now pin a function nothing uses, so the real cassette key has no field-coverage test.
            THIS IS THE 2ND FINDING IN FAMILY `single-source-consumer-not-derived`. Do not patch the five sentences. The RULE: when a refactor moves a consumer off a declared single source, the source is either re-wired to that consumer or deleted in the SAME change — an exported function with no caller plus docs asserting its role is a source that has quietly become documentation. THE ENUMERATION to sweep: for every "the same X that Y" / "single source" / "exactly one renderer" claim in internal/llm and internal/llm/llmtest, grep that Y actually calls X, and for every exported identifier in internal/llm confirm a non-test caller exists or the export is justified in its doc.
          family: single-source-consumer-not-derived
          round: 6
        - id: BR-51
          severity: Important
          title: requireSchemaFields checks only top-level required fields, so a nested object still decodes to a partial value with a nil error
          detail: |-
            Measured against the shipped code with type outer{Fits bool; Inner inner} where inner{Score int; Detail string}: SchemaFor emits "required":["score","detail"] on the nested object, and decode[outer](`{"fits":true,"inner":{}}`) returns {Fits:true Inner:{Score:0 Detail:""}} with err=nil; `{"fits":true,"inner":{"score":3}}` likewise. task.go:61 still claims "require every field the schema marks REQUIRED to be present" and task.go:75 still claims "it never returns a partially populated value with a nil error"; atlas/llm.md repeats both. #10's authoring result is the first consumer likely to be nested. Also unchecked: an object inside an array, and an explicit null for a required object field (present, so it passes, and zero-fills).
            THIS IS THE 5TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not fix only the nested case. The RULE: a check written to satisfy a finding must be written against the SHAPE the invariant quantifies over, not against the example the finding used — here the invariant quantifies over the whole schema tree, so the check must walk it (or the doc must state the depth limit, and then the limit needs its own test). THE ENUMERATION to sweep in this round: for each universal quantifier in the doc comments of task.go, schema.go, render.go, cassette.go and golden.go, write down the set it ranges over and confirm a test exists at every point of that set — depth for schemas, both branches for the fuzz target, every method for a seam double.
          family: enforcement-not-pinned-by-a-test
          round: 6
        - id: BR-52
          severity: Important
          title: A cassette cannot record a streaming exchange, and the failure is delivered as ErrUnavailable
          detail: 'exchange.Response is json.RawMessage (cassette.go:44), so an SSE body cannot be marshalled. Measured, recording a Stream call through the transport against the wire fake: `llm: unavailable: Post ".../v1/messages": json: error calling MarshalJSON for type json.RawMessage: invalid character ''e'' looking for beginning of value`. Half the Client interface is unrecordable, and a harness marshalling bug arrives wearing the class every consumer is designed to absorb silently — the collapse BR-38 existed to prevent, now on a different path. cassette.go:23 justifies the transport placement on the grounds that replay "parses an SSE frame", describing a path recording cannot produce. Replay itself would work: ssestream.NewDecoder ignores the application/json content-type jsonResponse hardcodes. Fix: store the body as bytes plus the recorded Content-Type, and return a harness error that is not in the dependency''s absorbable class.'
          family: double-covers-partial-seam
          round: 6
        - id: BR-53
          severity: Minor
          title: withUpdate restores *update to the literal false rather than its prior value, silently cancelling a real -update run
          detail: |-
            cassette_test.go:31. Measured: `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` FAILS ("a changed prompt passed its golden"); the full-package run only passes because the record test runs first and clobbers the flag back to false before the golden tests see it. So the documented refresh mechanism is order-dependent and self-cancelling in the package that defines it.
            THIS IS THE 2ND FINDING IN FAMILY `test-flag-mutation-leaks`. The RULE: a helper that mutates process-global state must capture the prior value and restore THAT — `defer func(prev bool) { *update = prev }(*update)` — because restoring to a constant is indistinguishable from a leak whenever the constant is not what the operator passed.
          family: test-flag-mutation-leaks
          round: 6
        - id: BR-54
          severity: Minor
          title: SchemaFor returns the memoised map by reference, so any consumer mutation poisons the cache process-wide
          detail: schema.go:27 returns the cached map[string]any itself. A consumer doing `s, _ := llm.SchemaFor[T](); s["description"] = "..."` permanently changes what every later Run[T] sends on the wire and what requireSchemaFields reads. Either clone on read or document the value as read-only and return it through a type that says so; this is a new internal package five downstream issues will consume.
          family: memoised-value-is-caller-mutable
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-08-23T07:45:26-07:00"
      agent: claude
      dispose:
        - id: BR-39
          disposition: not-addressed
          note: Both sites now call SkipIfUnreachable, but the helper is untested and skips on a renamed model (502 to ErrUnavailable) — the drift it exists to catch.
          round: 7
        - id: BR-42
          disposition: not-addressed
          note: '`_ = llmtest.Capture` still present at capture_conformance_test.go:142.'
          round: 7
        - id: BR-43
          disposition: not-addressed
          note: TestAQueueStillAdvancesWhileItHasEntries unchanged at fake_test.go:167, still the shape of TestQueueServesInOrder (fake_test.go:82).
          round: 7
        - id: BR-44
          disposition: not-addressed
          note: main.go:278 still returns on *llmCheck before the arity switch; `define -llm-check hello` ignores the word.
          round: 7
        - id: BR-45
          disposition: not-addressed
          note: llmcheck.go:41 still hardcodes MaxTokens 2048 rather than reading cfg.MaxTokens.
          round: 7
        - id: BR-46
          disposition: not-addressed
          note: 'Measured on the shipped tree: SchemaFor[string]() = {type:string, additionalProperties:false}; SchemaFor[map[string]string]() = an object permitting no keys.'
          round: 7
        - id: BR-50
          disposition: addressed
          note: 'Reversion-verified: removing the context lookup in key() reddens TestTheKeyDerivesFromTheRequestNotTheBody with the collision printed. See the new finding for the pre-defaulting residual.'
          round: 7
        - id: BR-51
          disposition: not-addressed
          note: Objects and explicit nulls are walked; array items are not — decode[arrOuter](`{"fits":true,"list":[{}]}`) still returns a partial value with err=nil, and task.go:129 now claims "EVERY depth".
          round: 7
        - id: BR-52
          disposition: addressed
          note: 'Reversion-verified: restoring json.RawMessage reproduces the exact ErrUnavailable marshalling failure in TestCassetteRecordsAndReplaysAStream.'
          round: 7
        - id: BR-53
          disposition: not-addressed
          note: cassette_test.go:32 still does `defer func() { *update = false }()` — the literal, not the prior value.
          round: 7
        - id: BR-54
          disposition: addressed
          note: 'Reversion-verified with the clone removed from BOTH return paths: all three assertions of TestSchemaIsolationAcrossCallers redden, including the nested one.'
          round: 7
      findings:
        - id: BR-55
          severity: Important
          title: The cassette key is taken before the Request is defaulted, so two different effective models collide into one recording
          detail: |-
            Measured on the clean tree: llm.Task[T]{Name:"veto", Prompt:"near-synonym?"} — the documented shape, since task.go:27 says a zero Model means "use the Config default" — recorded under Config{Model:"claude-opus-5"} and then Config{Model:"claude-sonnet-5"} wrote to ONE file, 4ea325a60ca6.json, whose on-disk request.model is claude-sonnet-5; the opus recording is gone. renderRequest hashes r.Model/r.Effort as DECLARED while params() (anthropic.go:66) resolves them from a.cfg below the hash. Two further consequences: TestEveryMeaningfulFieldReachesTheHash (render_test.go:57) passes while the property it names is false on the only production path, and a consumer's golden and its cassette describe different requests — golden_schema_test.go:31 hand-writes Model/Effort (hash a11b39ae49fe) where Run[T] sends them empty (hash fae0974f64b3).
            THIS IS THE 3RD FINDING IN FAMILY `single-source-consumer-not-derived`. Do not fix Model alone. The RULE, already stated at BR-50 and now failing on a different axis: where two derivations of one fact exist, one must derive from the other — here renderRequest is a restatement of params() that omits defaulting. Resolve Model/Effort/System onto the Request before withRequest(ctx, r) in Complete and Stream, so RenderRequest and the wire body agree by construction, and drive the field-coverage table through Run[T] end to end rather than over a hand-populated struct.
          family: single-source-consumer-not-derived
          round: 7
        - id: BR-56
          severity: Minor
          title: ErrorForStop is an exported function with zero references whose doc still asserts the role the rework removed
          detail: |-
            grep for ErrorForStop across the tree returns only its own definition and comment (errors.go:88-92). It was added this window so "a replayed recording reconstructs the same taxonomy member", and BR-38's fix replaced that mechanism with status replay. Also in this class: `c := llm.New(...)` followed by `_ = c` in withReq (cassette_test.go:217), and BR-42's `_ = llmtest.Capture`.
            THIS IS THE 5TH FINDING IN FAMILY `dead-code`. Do not delete only this function. The RULE is BR-50's own enumeration, which was run for RequestHash and RenderRequest and not for the rest: for every exported identifier in internal/llm, confirm a non-test caller exists or the export is justified in its doc — and for every `_ = X` statement, confirm it does something other than satisfy the compiler. Run that grep and paste it, rather than fixing the three sites this finding happens to name.
          family: dead-code
          round: 7
        - id: BR-57
          severity: Minor
          title: Round 6 changed the design again with no Revisions entry, and two artifacts still name llmtest.Golden
          detail: |-
            The plan's last `## Revisions` entry is the round-5 one. Undeclared since: the cassette key moved to a context-carried RequestHash, `exchange` gained ContentType and became text, and llmtest.SkipIfUnreachable is new exported surface with no row in the Integration points table. Plan line 1638 and issue line 146 both still say `llmtest.Golden`, which is not an identifier — the same string BR-34 and BR-50 each named for grep-verification.
            THIS IS THE 3RD FINDING IN FAMILY `plan-revision-not-appended`. The RULE stated at BR-41 was "diff the plan's Core concepts and Task lists against the tree at each boundary and append what moved" — it was run once, at round 5, and not at round 6. Make the diff a step of the boundary itself rather than a response to a finding, and grep the plan and issue for every identifier they name in the same pass.
          family: plan-revision-not-appended
          round: 7
        - id: BR-58
          severity: Minor
          title: jsonEscape double-escapes the request the cassette miss message exists to show
          detail: |-
            cassette.go:206. The envelope is built with json.Marshal, which escapes already, so the operator sees `request was: {\\\"max_tokens\\\":8192,...\\\\\\\"obsequious\\\\\\\"...}` — measured. The message's whole purpose is to be readable without recomputing a hash.
            THIS IS THE 3RD FINDING IN FAMILY `stdlib-reimplemented`. The RULE: before hand-rolling a string transform, check whether the encoder that consumes the value already performs it — a manual escape applied to a value that is later marshalled is always a double-escape, never a no-op. Sweep the diff for hand-written quoting/escaping helpers and confirm each sits outside an encoder, not inside one.
          family: stdlib-reimplemented
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-08-23T08:43:06-07:00"
      agent: claude
      dispose:
        - id: BR-39
          disposition: not-addressed
          note: 'Helper fixed and tested, but capture_conformance_test.go:42''s inline closure still skips on ErrUnavailable — measured: a renamed model answers 502, classifies ErrUnavailable, and all four drift subtests SKIP.'
          round: 8
        - id: BR-42
          disposition: addressed
          note: '`_ = llmtest.Capture` is gone; the llmtest import is now load-bearing via SkipIfUnreachable.'
          round: 8
        - id: BR-43
          disposition: not-addressed
          note: 'Re-verified by reversion: with sticky reverted, TestTheLastScriptedReplyIsSticky reddens while TestAQueueStillAdvancesWhileItHasEntries (fake_test.go:166) stays green.'
          round: 8
        - id: BR-44
          disposition: not-addressed
          note: 'Measured: `define -llm-check hello` runs the check and discards the word; main.go:278 still returns before the arity switch.'
          round: 8
        - id: BR-45
          disposition: not-addressed
          note: llmcheck.go:41 still hardcodes MaxTokens 2048 rather than reading cfg.MaxTokens.
          round: 8
        - id: BR-46
          disposition: not-addressed
          note: 'Measured: SchemaFor[string]() = {type:string, additionalProperties:false}; a top-level slice type now also gets it on a type:array schema.'
          round: 8
        - id: BR-51
          disposition: addressed
          note: 'Reversion-verified for arrays: removing the []any branch reddens all three assertions of TestDecodeRequiresFieldsAtEveryShape. Map values are a new finding, not this one re-raised.'
          round: 8
        - id: BR-53
          disposition: not-addressed
          note: 'Re-measured: `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still FAILS; cassette_test.go:32 restores the literal false.'
          round: 8
        - id: BR-55
          disposition: addressed
          note: 'Reversion-verified: dropping a.effective(r) reddens TestRequestInContextCarriesTheEffectiveModel ("the context carried an unresolved model"). Residual duplication raised separately.'
          round: 8
        - id: BR-56
          disposition: not-addressed
          note: 'Ran the enumeration: ErrorForStop still has 0 non-test refs (errors.go:88), and `_ = c` is still at cassette_test.go:242 guarding an unused llm.New at :231.'
          round: 8
        - id: BR-57
          disposition: not-addressed
          note: Revisions entry appended and llmtest.Golden corrected in plan and issue, but the Integration points table (plan:152) still has no SkipIfUnreachable row — one of the three deltas the finding named.
          round: 8
        - id: BR-58
          disposition: addressed
          note: jsonEscape removed; measured miss message now carries only the single escaping JSON-in-JSON requires.
          round: 8
      findings:
        - id: BR-59
          severity: Important
          title: The required-field walk skips additionalProperties, so a map-valued object still decodes to a partial value with a nil error
          detail: |-
            Measured on the clean tree with type zzMapOuter{Fits bool; By map[string]zzItem} where zzItem{Score int; Detail string}: SchemaFor emits by.additionalProperties = {type:object, required:[score,detail]}, and decode(`{"fits":true,"by":{"a":{}}}`) returns {Fits:true By:map[a:{Score:0 Detail:""}]} with err=nil; `{"fits":true,"by":{"a":{"score":1}}}` likewise. missingRequired recurses through schema["properties"] (task.go:154) and schema["items"] (task.go:169) and never through additionalProperties, which is what jsonschema.Reflector emits for a Go map. task.go:126 claims the walk covers "the WHOLE schema tree: object properties, array items, and non-object payloads" — an enumeration of three where there are four — and atlas/llm.md:112 claims "require every field the schema marks required". Separately, task.go:108 still says "Only object payloads are checked", which round 7's own array and scalar branches falsified.
            THIS IS THE 6TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not add an additionalProperties case and stop — that is the fourth instance-fix in a row (top-level BR-33, nested object BR-51, array item BR-51 re-raise, map value here). The RULE: the walk must be driven by the schema GENERATOR's nesting vocabulary, not by the payload shapes a finding happened to name. THE ENUMERATION, mechanically available: the subschema-bearing keywords jsonschema.Reflector can emit for a Go type under this configuration — properties, items, additionalProperties, and (latent under DoNotReference:true) $defs/$ref and oneOf/anyOf. Write that list into the doc comment, cover each arm, and pin each. The test-side half of the same rule: FuzzDecode's success branch asserts got.Reason != "" — a named field of one fixture type — so it ranges over `answer` alone, which is why every vehicle has had to be found by a reviewer. Replace it with a reflective check driven over a fixture list of struct / nested struct / slice-of-struct / map-of-struct / pointer-to-struct / top-level-slice.
          family: enforcement-not-pinned-by-a-test
          round: 8
        - id: BR-60
          severity: Minor
          title: The Revisions entry written to establish grep-verification claims a table row that does not exist
          detail: |-
            workshop/plans/000011-vocab-llm-plan.md:2138 states "`llmtest.SkipIfUnreachable` is new exported surface (added to the Integration points table above)". The table at plan:152 has eight rows and none of them is SkipIfUnreachable; llm.RequestFromContext is likewise absent. The claim sits inside the entry whose stated purpose is that the boundary "greps every identifier the plan and the issue name in the same pass".
            THIS IS THE 5TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not just add the row. The RULE, first stated at BR-34 and unchanged: a doc claim naming a code identifier or an on-disk artifact must be grep-verified against the tree in the SAME edit that writes it — and that applies to a claim about the document being edited, not only to claims about code. A revision entry asserting "added to X" is a claim about X's current contents; check X. The cheap enforcement is to write the table row first and the sentence second, so the sentence describes a state that already exists.
          family: docs-claim-absent-surface
          round: 8
        - id: BR-61
          severity: Minor
          title: effective() and params() are now two independent defaulting implementations of the same three fields
          detail: |-
            anthropic.go:70 resolves Model/Effort/MaxTokens against a.cfg for the context-carried Request; anthropic.go:77 params() independently re-does cmp.Or on the same three fields for the wire body. They agree today, so nothing is currently broken — but the fix for BR-55 added a third derivation of "what was asked" rather than collapsing to one, so a fourth Config-defaulted field added to params() would silently move the wire body without moving the cassette key, and no test would catch it because the field-coverage table (render_test.go:57) still runs over a hand-populated struct rather than through Run[T].
            THIS IS THE 4TH FINDING IN FAMILY `single-source-consumer-not-derived`. Do not add a fourth field to both functions when that day comes. The RULE, stated at BR-50 and again at BR-55: where two derivations of one fact exist, one must derive from the other. Here that is one line — `r = a.effective(r)` at the top of Complete and Stream, with params() reading the already-resolved values and its cmp.Or calls deleted — after which the wire body and RenderRequest agree by construction rather than by coincidence.
          family: single-source-consumer-not-derived
          round: 8
      boundary: M2
      blocked: false
    - "n": 9
      timestamp: "2026-08-23T10:03:37-07:00"
      agent: claude
      dispose:
        - id: BR-27
          disposition: not-addressed
          note: SlowEvery clamped; the class not swept — negative Timeout still returns ErrUnavailable in 1ms, negative MaxTokens still reaches the wire as -5 with err=nil.
          round: 9
        - id: BR-28
          disposition: not-addressed
          note: Stream path swept field by field; JSON path not swept at all — Stall/StallEarly/JunkFrame silently ignored on Complete, and an .sse capture on Complete surfaces as ErrUnavailable.
          round: 9
        - id: BR-29
          disposition: addressed
          note: The vacuous test is gone from the tree; only TestNewPreservesADisabledStallBound remains, and it reddens when New stops preserving a negative StallAfter.
          round: 9
        - id: BR-30
          disposition: addressed
          note: Reversion-verified — reverting splitInto to []byte reddens TestTextJoinsMultibyteBlocksWithoutCorruption while the ASCII test stays green.
          round: 9
        - id: BR-31
          disposition: addressed
          note: plan:518-520 now declares two phases and no Bytes, matching llm.go.
          round: 9
        - id: BR-32
          disposition: addressed
          note: Reply.Body and Reply.NoThinking are both gone from the struct and from every branch that read them.
          round: 9
        - id: BR-39
          disposition: addressed
          note: Measured live — with the proxy up and a renamed model, all four drift subtests now FAIL where round 8 measured them skipping. No ErrUnavailable skip remains in any conformance file.
          round: 9
        - id: BR-43
          disposition: not-addressed
          note: Re-verified by reversion — with sticky reverted, TestAQueueStillAdvancesWhileItHasEntries (fake_test.go:166) stays green while TestTheLastScriptedReplyIsSticky reddens.
          round: 9
        - id: BR-44
          disposition: not-addressed
          note: Re-measured — `define -llm-check hello` runs the check and discards the word, exit 0.
          round: 9
        - id: BR-45
          disposition: not-addressed
          note: llmcheck.go:41 still hardcodes MaxTokens 2048. Folded into the diagnostic-ignores-config class finding this round.
          round: 9
        - id: BR-46
          disposition: not-addressed
          note: schema.go:100 still unconditional; SchemaFor[string]() still returns additionalProperties:false on a string schema.
          round: 9
        - id: BR-53
          disposition: not-addressed
          note: Re-measured — `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still fails; cassette_test.go:32 restores the literal false.
          round: 9
        - id: BR-56
          disposition: not-addressed
          note: Ran the enumeration — ErrorForStop still 0 non-test refs (errors.go:92); `_ = c` still at cassette_test.go:242 guarding an unused llm.New at :231.
          round: 9
        - id: BR-57
          disposition: not-addressed
          note: Table at plan:152 still lacks SkipIfUnreachable and RequestFromContext, and the new task_conformance_test.go surface is undeclared in the plan too.
          round: 9
        - id: BR-59
          disposition: addressed
          note: Reversion-verified twice — deleting the additionalProperties arm reddens TestRequiredWalkCoversEveryNestingKeyword AND FuzzDecode/seed#8 via an independent oracle.
          round: 9
        - id: BR-60
          disposition: not-addressed
          note: plan:2138 still claims SkipIfUnreachable was "added to the Integration points table above"; the eight-row table at plan:152 does not contain it.
          round: 9
        - id: BR-61
          disposition: not-addressed
          note: Now measured stronger — making params() send a different model than effective() hashes leaves the ENTIRE suite green.
          round: 9
      findings:
        - id: BR-62
          severity: Important
          title: The live typed-task suite fails on ordinary model variation, asserting judgment its own doc comment disclaims
          detail: |-
            Measured against the live proxy: 3 failures in 18 runs (~17%), two distinct modes.
            "task_conformance_test.go:97: option[3] = {Word: Why:}: a required field came back
            empty" (the model returned four options where the prompt asks for three, the fourth
            with empty strings) and "task_conformance_test.go:88: stem has no blank:
            \"placeholder\"" with "no options returned" (a degenerate stub). In both, Run[T]
            returned err=nil correctly — an empty array and an empty string both satisfy JSON
            Schema required. Three artifacts claim the test asserts shape only: the doc comment
            at :23, atlas/llm.md:207, and the commit message of e4c0364. Contains(stem,"___")
            is a formatting convention and non-emptiness is not a shape the layer guarantees.
            Two messages compound it by misattributing to the walk: :56 says "the required-field
            check should have rejected this" and :95 says "Every required field at every depth —
            the property the walk exists for", but missingRequired enforces key PRESENCE, not
            value non-emptiness, so both send the next reader hunting in a function that behaved
            correctly.
            THIS IS THE 3RD FINDING IN FAMILY `unclassified-failure-mode`. Do not fix these two
            assertions. The RULE, stated at BR-39 as "a check must distinguish dependency
            unreachable from dependency changed before reporting either", needs its third
            bucket: a live check must ALSO distinguish "the dependency changed" from "the
            dependency behaved normally but differently", and may only assert properties the
            layer under test guarantees. THE ENUMERATION, mechanically available: for each
            assertion in the three -tags conformance files, name which side guarantees it —
            the transport/decode layer (assert) or the model (log). Contains(stem,"___"),
            TrimSpace(o.Word)!="" and TrimSpace(got.Reason)!="" are all model-side and belong
            beside the verdicts already in t.Logf; the layer-side properties this test should
            assert and does not are err==nil, that the struct decoded, and that the schema
            reached the wire. The same sweep covers mustCall at capture_conformance_test.go:52,
            which Fatalfs a mid-run 429/529 as drift.
          family: unclassified-failure-mode
          round: 9
        - id: BR-63
          severity: Important
          title: define --llm-check discards run's signal context, so Ctrl-C is swallowed for the full five-minute timeout
          detail: |-
            main.go:154 builds signal.NotifyContext with the comment "Ctrl-C now cancels the
            context ... This changes the one-shot path too, deliberately", and run() threads
            that ctx into defineOnce and repl. main.go:278 drops it —
            runLLMCheck(os.Getenv, llm.New, stdout, stderr) takes no ctx — and llmcheck.go:35
            starts from context.Background() with cfg.Timeout, five minutes. Measured against a
            server that accepts and never answers: the process survived three SIGINTs over six
            seconds. Repeated Ctrl-C does not help, because NotifyContext leaves signal.Notify
            registered until stop(), so the default terminate behaviour is never restored and
            the process is uninterruptible until the deadline.
            THIS IS THE 2ND FINDING IN FAMILY `diagnostic-ignores-config`. Do not just add a ctx
            parameter. The RULE that covers both: the diagnostic path must use the inputs the
            caller already resolved, never reconstruct them. Measured prevalence 2 —
            llmcheck.go:41 hardcodes MaxTokens 2048 instead of cfg.MaxTokens (BR-45, still
            open), and llmcheck.go:35 reconstructs a context instead of deriving from run's.
            THE ENUMERATION is runLLMCheck's argument list against what run() already holds:
            ctx, the resolved cfg, and the writers — two of the three are reconstructed.
          family: diagnostic-ignores-config
          round: 9
        - id: BR-64
          severity: Minor
          title: atlas/llm.md says "Two tagged suites" above a list of three, and the third bullet's claim is measurably false
          detail: |-
            atlas/llm.md:200 reads "Two tagged suites, both on-demand" and is followed by three
            bullets; e4c0364 inserted TestTypedTaskAgainstTheLiveService without touching the
            count. The same inserted bullet asserts "Asserts shape, never the model's judgment",
            which the ~17% live failure rate falsifies.
            THIS IS THE 6TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not just change
            "Two" to "Three". The RULE, unchanged since BR-34: a doc claim of universal or
            COUNTING form is a claim about an enumeration, so it may only be written after
            running that enumeration — and inserting a member into a list is an edit to every
            counting sentence that governs the list. The cheap enforcement is to grep the
            enclosing section for a cardinal before adding a bullet, and to re-read the sentence
            you are inserting under, not only the ones you wrote.
          family: docs-claim-absent-surface
          round: 9
      blocked: true
---

# Gate ledger — tools#11 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-22T19:21:28-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `stream-failure-classification` A stalled stream discards accumulated answer text and returns ErrUnavailable, contradicting the sawEvent rule the same function documents
  internal/llm/anthropic.go:154 returns Response{} on stall, throwing away every
  byte msg accumulated, and classifies as ErrUnavailable. Fifteen lines below,
  anthropic.go:168 does the opposite for the same situation, and atlas/llm.md:60
  states the preserving behaviour as the contract. Verified with a peer that
  replays the recorded frames through the first text_delta then goes silent:
  onDelta saw "**Obsequious.**\n\nThe clue after" while resp.Text was "" and
  errors.Is(err, ErrTruncated) was false. A Stream caller passing onDelta == nil
  loses the whole partial answer, and a mid-answer hang tells the caller to stop
  trying rather than to skip the question.
- **BR-2** [Critical] `fake-cannot-reach-the-branch` No test can exercise a stall that occurs after answer text has arrived
  Fake.serveStream triggers Stall on strings.Contains(fr, "content_block_delta")
  at internal/llm/llmtest/fake.go:328, and the first such frame in
  stream-sample.sse is the thinking_delta at index 0. So TestStreamStallIsDetected
  only ever runs stall-before-any-text, where returning Response{} is
  coincidentally correct. Gate on text_delta the way JunkFrame already does at
  fake.go:322, and assert the partial text survives.
- **BR-3** [Important] `fake-tuned-to-its-fixture` The fake rejects exactly one model name and it is the test's own fixture string
  internal/llm/llmtest/fake.go:128 accepts any model with a known prefix unless it
  contains the literal "not-a-real". Verified: changing the fixture to
  claude-opus-6 makes TestUnknownModelIsRejectedLikeTheProxyDoes fail with
  "status = 200, want 502". The fake therefore answers 200 where the measured
  proxy answers 502 for every typo but one, and llmtest.Suite's unknown-model
  obligation passes only because the fixture carries the magic substring
  (ARCH-MOCK). Hold a small set of real model ids instead.
- **BR-4** [Important] `partial-constructor-defaults` New defaults Timeout and MaxTokens but not StallAfter, Model or Effort
  internal/llm/anthropic.go:31. A consumer writing llm.New(llm.Config{BaseURL: u,
  APIKey: k}) silently gets no stall detection, so a hung stream blocks for the
  full 5-minute deadline. StallAfter is the one bound the SDK does not supply and
  the one the constructor omits. Config is this package's whole public input with
  five consumers queued behind it: default all five in New, or none and make
  Resolve the only documented constructor path.
- **BR-5** [Important] `docs-claim-absent-surface` atlas and suite doc comments state a -tags conformance live run that has no file
  atlas/llm.md:98 and internal/llm/llmtest/suite.go:20 both say Suite runs against
  the live proxy under -tags conformance, and internal/llm/suite_test.go:12 names
  a conformance_test.go. The only conformance-tagged file in the tree is
  cmd/define/fetch_conformance_test.go; the llm one is plan Task 12, deferred to
  M2. The plan is consistent, the atlas is not, and the atlas is the
  current-state map (ARCH-MOCK: the live half of the fake's contract is asserted
  but absent).
- **BR-6** [Important] `enforcement-not-pinned-by-a-test` Nothing in go test enforces the captures' required shapes, and the script that does cannot run offline
  internal/llm/llmtest/testdata/README.md tabulates each capture's required shape
  as "enforced by llm-probe.sh verify", but nothing runs that script and it cannot
  run without the proxy config: K="$(key)" executes unconditionally at
  scripts/llm-probe.sh:20, so CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify
  dies on FileNotFoundError even though verify needs neither key nor network.
  Given fake-models-unobserved-shape produced three plan-gate findings, the
  durable fix is a Go test; llmtest already embeds every capture.
- **BR-7** [Minor] `stale-module-metadata` go mod tidy was never run; the directly-imported SDK is still marked indirect
  go.mod:11 lists github.com/anthropics/anthropic-sdk-go as // indirect although
  internal/llm/anthropic.go imports it. Plan Task 1 Step 1 said tidy runs at the
  end of Task 5. Verified: tidy promotes it to the direct require block and
  changes nothing else.
- **BR-8** [Minor] `test-panics-instead-of-failing` TestThinkingSignatureIsPreserved discards the error then indexes Blocks[0]
  internal/llm/anthropic_test.go:90 does got, _ := ...Complete(...) and then
  got.Blocks[0] at :91. When the call errors this panics with index out of range,
  aborting the whole package run and masking every other result. Observed while
  mutation-testing. Check the error.
- **BR-9** [Minor] `dead-code` var _ = http.StatusOK props up an otherwise-unused net/http import
  internal/llm/anthropic_test.go:381. Drop the line and the import.
- **BR-10** [Minor] `pure-logic-on-an-io-type` response() has an unused receiver, so the package's only transformation logic is reachable only through httptest
  internal/llm/anthropic.go:185 never references a. Promoting it to a package-level
  function would let block assembly and Text-joining be unit-tested directly from a
  capture via json.Unmarshal, with no server (ARCH-PURE).
- **BR-11** [Minor] `fake-silently-ignores-inputs` serveStream ignores Reply.Text and Reply.Stop entirely
  internal/llm/llmtest/fake.go:305. f.Script("x", Reply{Text: "…", Stop: "refusal"})
  silently does nothing on a streaming request — it always replays the capture.
  Either honour them or fail loudly on the combination.
- **BR-12** [Minor] `test-helper-fatal-off-goroutine` Capture calls t.Fatalf from the fake's server goroutine
  internal/llm/llmtest/fake.go:311. A missing capture, or a request arriving after
  cleanup, becomes a hang or a "log after test completed" panic rather than a clean
  failure. Resolve captures at Script/ServeRecorded time instead.
- **BR-13** [Minor] `comment-references-future-surface` config.go references --llm-check, which does not exist until M2
  internal/llm/config.go:99. Also io := []byte(...) shadows the io package at
  internal/llm/anthropic_test.go:349.

## Round 2 — 2026-08-22T19:45:37-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified — reverting the stall branch to Response{}+ErrUnavailable turns TestStreamStallAfterTextSalvagesAndTruncates red.
- BR-2 — addressed — Mutation-verified — re-gating Stall on content_block_delta turns the same test red at "no deltas were delivered".
- BR-3 — addressed — Behavior verified: claude-opus-6 / -typo / -sonnet-9 all get 502. Not pinned by a test — rolled into the enforcement-not-pinned-by-a-test repeat.
- BR-4 — addressed — New now defaults all six fields. Side effects (unpinned; "Zero disables it" now unreachable) raised as new findings.
- BR-5 — addressed — conformance_test.go exists, vets under -tags conformance, and skips cleanly with no key configured.
- BR-6 — addressed — captures_test.go runs offline in go test; CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify exits 0.
- BR-7 — addressed — SDK is in the direct require block; go mod tidy produces no diff.
- BR-8 — addressed — Both tests now check the error and length-check before indexing Blocks.
- BR-9 — addressed — var _ = http.StatusOK and the net/http import are gone from anthropic_test.go.
- BR-10 — addressed — buildResponse is package-level with no receiver. The doc comment's claim of a direct unit test is false — rolled into the docs-claim repeat.
- BR-11 — addressed — serveStream fails loudly; verified reachable with a scratch test. No fixture enters it — rolled into the enforcement repeat.
- BR-12 — addressed — Both capture-miss paths return 500 instead of Fatalf. A new off-goroutine Fatalf was added by BR-11's fix — raised separately.
- BR-13 — addressed — config.go:99 cleared; llm.go:110 and errors.go:54 still carry the reference — raised as the family repeat.

### Raised

- **BR-14** [Important] `unclassified-failure-mode` An exhausted 408 classifies as ErrRequest, the loud "our bug" arm, though the SDK retries it as transient
  internal/llm/errors.go:44-52 routes 429 and >=500 to ErrUnavailable and everything
  else to default/ErrRequest, so 408 Request Timeout and 409 Conflict land loud.
  Measured end-to-end through the real client against the fake: status 408 ->
  "llm: bad request", ErrUnavailable=false, after 3 attempts — the SDK retried it,
  which is the transport itself calling it transient, and then the taxonomy calls it
  our bad schema/model/body. errors_test.go covers neither status. Add
  http.StatusRequestTimeout to the ErrUnavailable arm, decide 409 deliberately, and
  add both rows to the table.
- **BR-15** [Important] `enforcement-not-pinned-by-a-test` Three of this round's fixes survive full reversion with a green suite, or are entered by no fixture
  This is the 2nd finding in family enforcement-not-pinned-by-a-test. Do NOT fix these
  three instances — the rule is: a boundary fix that changes behavior is not complete
  until a test fails against the pre-fix code, and the fixture must not encode the
  pre-fix implementation's escape hatch. Measured prevalence 3, all verified this
  round. BR-3 (fake.go:148): reverting knownModel to the prefix + "not-a-real"
  substring rule leaves go test ./internal/llm/... green, because the fixture
  claude-not-a-real-model still carries the magic substring. BR-4 (anthropic.go:39):
  deleting the StallAfter default from New leaves the suite green — every stall test
  sets it explicitly. BR-11 (fake.go:342): no test scripts Reply{Text} against a
  streaming request, so the loud branch is reachable but unexercised.
- **BR-16** [Important] `docs-claim-absent-surface` Four doc claims outrun the tree, two of them written by this round's own fixes
  This is the 2nd finding in family docs-claim-absent-surface. Do NOT fix these four
  sites individually — the rule is: a claim in a doc comment or atlas page is a claim
  about the tree, and the sweep must check each against the tree rather than against
  the intent of the change that wrote it. Measured prevalence 4. (1) anthropic.go:203
  says buildResponse "is now unit-tested directly from a committed capture with
  json.Unmarshal and no server at all" — grep finds no caller outside anthropic.go,
  and it is unexported while anthropic_test.go is package llm_test. (2) config.go:43
  says StallAfter "Zero disables it", but New — the only constructor — overrides zero
  with 90s; the plan repeats the sentence at plan:837. (3) llm.go:130-136 documents
  Progress phases connect|waiting|streaming|done, but only waiting and streaming are
  ever emitted, and Progress.Bytes is assigned nowhere. (4) fake.go:104 says
  Reply.Capture "wins over the rest", but serve checks reply.Status first at
  fake.go:256.
- **BR-17** [Minor] `test-helper-fatal-off-goroutine` BR-11's fix added a t.Fatalf on the fake's server goroutine, ten lines above the comment forbidding it
  This is the 2nd finding in family test-helper-fatal-off-goroutine. Do NOT fix this
  instance — state the rule (no t.Fatalf/FailNow from a handler goroutine; resolve at
  Script time or answer with a 500) and fix that. fake.go:343 calls f.t.Fatalf inside
  serveStream while fake.go:352 says "Same reason as serveJSON: no Fatalf off the test
  goroutine". Verified: it does record the failure today, but a request arriving after
  cleanup still panics with "log after test completed".
- **BR-18** [Minor] `comment-references-future-surface` --llm-check was removed from config.go and left in two sibling files
  This is the 2nd finding in family comment-references-future-surface. Do NOT fix this
  instance — state the rule covering forward references to unbuilt surface and sweep
  the enumeration. Remaining: llm.go:110 and errors.go:54. Measured prevalence 2 after
  the round that closed the one site the finding named (ARCH-PURPOSE: instance, not
  class).
- **BR-19** [Minor] `test-panics-instead-of-failing` captures_test.go indexes into unchecked type assertions, so a malformed capture panics rather than fails
  This is the 2nd finding in family test-panics-instead-of-failing. Do NOT fix this
  instance — state the rule and sweep. internal/llm/llmtest/captures_test.go:29-31 does
  m["content"].([]any) and c.(map[string]any)["type"].(string), and :40 does
  cm["text"].(string). A capture missing content aborts the whole package run. This
  instance was introduced by the fix round that closed BR-8.
- **BR-20** [Minor] `stdlib-reimplemented` orDefault, orInt64 and orDuration re-implement stdlib cmp.Or
  anthropic.go:294-306 and config.go:96. cmp.Or[T comparable](vals ...T) T has been in
  the standard library since Go 1.22 and the module targets 1.26, so one import
  replaces three near-identical helpers (ARCH-DRY).
- **BR-21** [Minor] `review-archaeology-in-code` Production comments increasingly narrate prior review rounds, duplicating the gate ledger
  The fix commit added 110 comment lines against 230 code lines. Examples:
  anthropic.go:32-37 and :164-169, fake.go:117-121 and :142-147 all recount what "the
  first version" did. That history already lives in the gate ledger and the issue Log,
  and it means a behavior change now needs three edits to stay honest. llm.go is 53%
  comment, errors.go 48%. Round 1 flagged density as worth watching; it rose.

## Round 3 — 2026-08-22T20:04:10-07:00 (claude) — BLOCKED

### Disposed

- BR-14 — addressed — Fixed at the cause — transient() derives from the SDK retry policy; reverting reddens the 408 and 409 rows.
- BR-15 — addressed — All three reversions now red; TestNewDefaultsEveryConfigField verified against two different deleted defaults, not just StallAfter.
- BR-16 — not-addressed — Three claims still false: fake.go:107 "wins over the rest" (Status wins, measured), plan:837, plan:345-346 — two cited by line in the finding.
- BR-17 — addressed — No t.Fatalf remains on any handler goroutine; the rule is stated in serveStream and both capture-miss paths answer with a status.
- BR-18 — addressed — Swept to zero — no --llm-check reference anywhere in internal/, cmd/, atlas/ or scripts/.
- BR-19 — addressed — captures_test.go checks every assertion and t.Fatalf's; the rule is stated at the top of the helper.
- BR-20 — addressed — cmp.Or replaces all three helpers. Two hand-rolled contains helpers survived the sweep — raised as the family repeat.
- BR-21 — addressed — Density fell (llm.go 53->48%, errors.go 48->39%); only three archaeology sites remain and all three are rationale.

### Raised

- **BR-22** [Important] `fake-cannot-reach-the-branch` Three of the transport's own named invariants can be deleted outright with the whole suite still green
  This is the 2nd finding in family fake-cannot-reach-the-branch. Do NOT fix these
  four instances — the rule is: a behaviour the code singles out as load-bearing
  needs a fixture that separates it from the alternative it warns against, and
  where no committed capture exhibits that shape the fixture is constructed,
  because block count and header presence are transport shape rather than
  judgment. Measured prevalence 4, three verified by reversion against
  go test ./internal/llm/... this round. (1) anthropic.go:218-228 — all three
  captures carry exactly one text block, so writing only the first leaves the
  suite green, though "every text block JOINED, not Content[0].Text" is what the
  atlas, the package doc and Response.Text all single out. (2) anthropic.go:62-64
  — deleting the r.System to p.System plumbing leaves the suite green;
  Recorded.System() exists at fake.go:59 for exactly this assertion and has zero
  callers, and M2's Task[T] sets System on every call. (3) anthropic.go:262-284 —
  neutering watch() so OnSlow never fires leaves the suite green; grep for
  OnSlow or Progress across _test.go returns nothing, and plan:1383 lists
  TestOnSlowNamesThePhase undelivered. (4) Response.StopDetails has zero test
  references and the fake's refusal reply emits no stop_details, so the populated
  branch is unreachable from either side of the seam.
- **BR-23** [Minor] `stdlib-reimplemented` Two hand-rolled contains helpers survived the round that replaced the or* helpers with cmp.Or
  This is the 2nd finding in family stdlib-reimplemented. Do NOT fix these two
  instances — state the rule (a helper duplicating a stdlib function is removed,
  and the sweep covers _test.go as well as production files) and fix that.
  Measured prevalence 2. errors_test.go:91 re-implements strings.Contains with an
  index loop; captures_test.go:130 re-implements slices.Contains. Both are in the
  same two packages as the cmp.Or fix and both were skipped because the sweep
  scoped itself to production code.
- **BR-24** [Minor] `dead-code` Fake.t is assigned in NewFake and read by nothing after BR-17 removed its last consumer
  This is the 2nd finding in family dead-code. Do NOT fix this instance — the rule
  is: when a fix removes the last reader of a field, import or helper, the same
  edit removes the field. internal/llm/llmtest/fake.go:174 declares t *testing.T
  and fake.go:184 assigns it; grep for f.t across the package returns nothing now
  that serveStream answers with a status instead of calling Fatalf. go vet does
  not flag unused struct fields, so nothing else will catch this class.
- **BR-25** [Minor] `fake-silently-ignores-inputs` An unknown-model request pops the matcher queue and then discards the reply it drew
  This is the 2nd finding in family fake-silently-ignores-inputs. Do NOT fix this
  instance — the rule is: the fake must not mutate scripted state on a path that
  does not serve the scripted reply. internal/llm/llmtest/fake.go:233 calls
  f.next(rec.Prompt()) before the unknown-model check at fake.go:243 returns 502,
  so a queued Reply is consumed and thrown away. Harmless today because no test
  scripts a sequence against a bad model; a trap for M2's cassette sequences,
  where the next call would silently draw the wrong queue entry.
- **BR-26** [Minor] `plan-revision-not-appended` The plan was corrected in place at the M1 boundary with no appended Revisions entry
  workshop/plans/000011-vocab-llm-plan.md carries four dated Revisions entries,
  none for the boundary-review rounds, while the Robustness bar was rewritten
  in place ("Corrected in place, M1 boundary review (C1)") and measured fact 2
  likewise. AGENTS.md section 1 requires appending a Revisions entry with
  timestamp, reason and delta rather than overwriting, so the pre-review claim
  survives. Round 2 recommended this in its advisory section, where it was never
  tracked as a finding and therefore never disposed.

## Round 4 — 2026-08-22T20:26:51-07:00 (claude) — passed

### Disposed

- BR-16 — addressed — All four cited sites plus round 3's three re-cited sites verified correct against the tree; one uncited instance survives at plan:516 and is raised as the family repeat.
- BR-22 — addressed — All four instances pinned by constructed fixtures; each verified red by reversion this round against go test ./internal/llm/...
- BR-23 — addressed — Both hand-rolled contains helpers gone; errors_test.go:87 and captures_test.go:70,123 now use strings.Contains / slices.Contains.
- BR-24 — addressed — Fake.t is gone from the struct and from NewFake; grep for a testing.T field in fake.go returns only the Capture and NewFake parameters.
- BR-25 — addressed — f.next() now runs after the unknown-model check, with the rationale recorded at the call site.
- BR-26 — addressed — A fifth Revisions entry ("M1 boundary review, three rounds") is appended at plan:1986 with the rules the rounds produced.

### Raised

- **BR-27** [Important] `partial-constructor-defaults` A negative SlowEvery panics on the watcher goroutine, where no caller can recover
  This is the 2nd finding in family partial-constructor-defaults. Do NOT fix
  this site — the rule is: New must give every Config field a defined meaning
  for every value it can hold, and sibling duration fields must not disagree
  about what a sentinel means. Measured prevalence 3, all verified this round.
  llm.New(Config{BaseURL, APIKey, SlowEvery: -1, OnSlow: f}) panics
  "non-positive interval for NewTicker" at anthropic.go:270 on a goroutine the
  caller cannot recover from, killing the process. Negative Timeout instead
  makes every call return ErrUnavailable instantly — silently absorbed rather
  than reported as misconfiguration. Only StallAfter has a documented negative
  meaning (config.go:52-55 says disabling REQUIRES a negative value), which is
  what invites a caller to try the same on SlowEvery, documented at config.go:59
  as "Zero takes the default" and nothing more. TestNewDefaultsEveryConfigField
  asserts non-zero, so -1 passes it.
- **BR-28** [Important] `fake-silently-ignores-inputs` A .json capture scripted against a streaming request is silently replaced by stream-sample.sse
  This is the 3rd finding in family fake-silently-ignores-inputs. Do NOT fix
  this site — the rule is: every Reply field that cannot be served on the path
  a request took is a caller mistake and answers with a 400 naming the field,
  and the sweep covers the Reply struct field by field rather than the one
  branch a finding named. internal/llm/llmtest/fake.go:403 falls back to
  stream-sample.sse for any Capture not ending in .sse. Measured: scripting
  Reply{Capture: "message-truncated.json"} against Stream returns err=nil,
  Stop="end_turn" and the Obsequious text from stream-sample.sse — a different
  capture with a different stop_reason than the one asked for. This is the
  un-swept half of BR-11's own enumeration: the adjacent Reply{Text/Stop}
  branch was made loud this round at fake.go:381, ten lines above. M2's
  cassette sequences run through this path.
- **BR-29** [Minor] `enforcement-not-pinned-by-a-test` TestNegativeStallAfterDisablesTheBound survives full reversion of the behaviour it names
  This is the 3rd finding in family enforcement-not-pinned-by-a-test. Do NOT
  fix this site — the rule the family never enumerated is: reversion-check the
  TEST you add, not only the fix it pins, because a bound dominated by a second
  bound distinguishes nothing. Verified: changing New to
  "if c.StallAfter <= 0 { c.StallAfter = defaultStallAfter }" reddens only
  TestNewPreservesADisabledStallBound; anthropic_test.go:444 stays green,
  because its assertion (elapsed >= 1s against Timeout 2s) holds whether the
  stall bound is disabled or defaulted to 90s — the total deadline ends the
  call at ~2s either way. Its comment claims it makes the documented escape
  hatch reachable and tested. Prevalence 1 of the 6 tests added this round; the
  other 5 were each verified red.
- **BR-30** [Minor] `fake-tuned-to-its-fixture` splitInto chops on byte offsets, so any multibyte scripted text round-trips corrupted
  This is the 2nd finding in family fake-tuned-to-its-fixture. Do NOT fix this
  site — the rule is: a constructed fixture helper must be correct for the
  inputs the committed captures actually contain, not only for the ASCII string
  its first caller passes. internal/llm/llmtest/fake.go:162 slices on len(s)/n,
  so a rune spanning a boundary is cut and json.Marshal substitutes U+FFFD.
  Measured: Reply{Text: "obsequieux — tres flagorneur, vraiment", SplitText: 3}
  comes back with the em-dash replaced by three U+FFFD, so Response.Text is not
  what was scripted. TestTextJoinsEveryTextBlock passes only because its
  fixture is ASCII; every committed capture contains em-dashes. Introduced by
  this round's own BR-22.1 fix.
- **BR-31** [Minor] `docs-claim-absent-surface` The plan's Task 1 contract block still declares four Progress phases and a Bytes field
  This is the 3rd finding in family docs-claim-absent-surface. Do NOT fix this
  site — the rule needs widening: the sweep's enumeration includes embedded
  code blocks in plan artifacts, not only prose and production files.
  workshop/plans/000011-vocab-llm-plan.md:516-518 declares
  Phase string // "connect" | "waiting" | "streaming" | "done" and Bytes int,
  which llm.go corrected to two phases with Bytes removed and which the same
  document's own Revisions entry at plan:2040 declares gone. Round 3's sweep
  fixed the plan's prose (plan:837, plan:345-346, both verified correct now)
  and stopped before its code blocks — which Task 1 Step 3 explicitly warns
  "get pasted verbatim", the reason a stale one is a hazard rather than a typo.
- **BR-32** [Minor] `dead-code` Reply.Body and Reply.NoThinking are documented knobs that no fixture in the tree turns
  This is the 3rd finding in family dead-code. Do NOT fix these two instances —
  the rule needs widening from "a field whose last reader a fix removed" to
  "the sweep enumerates the fake's exported surface, and a knob no fixture
  turns is dead the same as an unread field." internal/llm/llmtest/fake.go:114
  (Body) and :118 (NoThinking) each gate a production branch in the fake
  (fake.go:289 and :339) and neither is set anywhere in the tree, so both
  branches are reachable but unexercised — the shape BR-15c named. Round 3
  identified both in its prose section 5 and never raised them, so they were
  never tracked or disposed.

## Round 5 — 2026-08-23T00:06:09-07:00 (claude) — BLOCKED

### Raised

- **BR-33** [Critical] `enforcement-not-pinned-by-a-test` decode returns a partially populated T with a nil error when a required field is missing
  Measured against the shipped code: decode[answer](`{"fits":true}`) returns {Fits:true Reason:""} with err=nil; `{}` and `null` return the zero value with err=nil. task.go:59 states "it never returns a partially populated value with a nil error", atlas/llm.md:112 states "reject missing ones", and plan Task 10 states "require every schema-required field present". SchemaFor already emits "required":["fits","reason"] — the single source declares it and decode ignores it. #12's veto would read Fits:false from `{}` and silently drop a distractor rather than skip the question.
  THIS IS THE 4TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not fix only this site. The RULE: an invariant stated in a doc comment must be asserted by a test that goes red when it is violated — including the SUCCESS branch of a property/fuzz target, which is where FuzzDecode returns early (`if err == nil { return }`) and where its own seed `"{}"` is a violating input the target waves through. THE ENUMERATION to sweep in this round: grep every "never", "always", "must", "either ... or" claim in the doc comments added by this window (task.go decode, task.go Run, render.go renderRequest/renderSchema, schema.go SchemaFor, cassette.go Cassette/Stream, golden.go AssertGolden, fake.go next) and for each confirm a test that fails when the claim is broken; where none exists, either write it or delete the claim.
- **BR-34** [Important] `docs-claim-absent-surface` Four doc claims in this window assert properties the code does not hold
  Measured prevalence, 4 instances in one milestone. (1) atlas/llm.md:112 "allow unknown fields, reject missing ones" — decode does not reject missing (see the Critical). (2) atlas/llm.md:114 "Its invariant ... is held by a fuzz target" — FuzzDecode asserts only the error half. (3) atlas/llm.md:171 "Every failure names scripts/llm-probe.sh record" and capture_conformance_test.go:22 "On drift the failure names the fix" — 2 of 7 failure messages in that file name it; the unknown-block-type, no-text-block, output_config-decode, no-deltas and preamble messages do not. (4) internal/llm/schema.go:15 "llmtest.Golden snapshots it, so a struct field added without thought shows up in a diff" — no golden file exists anywhere in the tree, and `llmtest.Golden` is not an identifier (it is `AssertGolden`).
  THIS IS THE 4TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not patch the four sentences. The RULE: a doc claim of UNIVERSAL form ("every", "always", "never", "is held by") is a claim about an enumeration, so it may only be written after enumerating the sites and checking each — and a doc claim naming a code identifier or an on-disk artifact must be grep-verified against the tree in the same edit. Sweep: for each universal claim in atlas/llm.md and in the doc comments of render.go, schema.go, task.go, cassette.go and golden.go, run the enumeration it implies and either make it true or weaken it to what is true.
- **BR-35** [Important] `single-source-consumer-not-derived` Plan Task 9's schema golden was never written, so AssertGolden ships with zero committed artifacts
  internal/llm/testdata/ does not exist; `find` returns no golden/ or cassettes/ directory in the repo; the only callers of AssertGolden and Cassettes are their own self-tests against t.TempDir(). Task 9 required a snapshot AND a byte-identical assertion, and schema.go:15 cites that snapshot as the property that justifies reflecting the schema instead of hand-writing it. ARCH-PURPOSE shadow-sweep on the SchemaFor[T] single source: 4 consumers, 2 derive (Request.Schema on the wire, RequestHash), 2 do not (no golden; decode ignores the required list).
- **BR-36** [Important] `user-surface-undocumented` README.md is not updated for the new --llm-check flag or its exit code
  `grep -n "llm" README.md` returns nothing. README documents --sound/-times, -locale, -raw, --forget and DEFINE_NO_CAPTURE, and its exit-code paragraph enumerates what `1` means ("no dictionary entry, or --forget found nothing to remove") — --llm-check now also exits 1, for a third reason. main.go's usage text was updated; README was not.
- **BR-37** [Important] `double-above-the-seam` The cassette double replaces llm.Client, bypassing the SDK path the plan places it beneath
  Plan Task 4 Step 3 specifies `func Cassette(t *testing.T, r llm.Request) Reply` — a body served through the wire Fake. Shipped is `func (c *Cassette) Client(live llm.Client) llm.Client` (cassette.go:47): replay reads the file and returns rec.Response without serialising a Request, without the SDK, without SSE. llmtest's own package doc (fake.go:5) argues that a stubbed Client cannot see a mis-serialized output_config, a dropped anthropic-version header, or a retry that re-sends a consumed body — a consumer test written against a cassette now cannot see any of them. cassetteClient.Stream (cassette.go:101) does not stream: it calls Complete and fires a single delta. ARCH-MOCK: production flow and test flow no longer share the same boundary on this path.
- **BR-38** [Important] `double-rederives-error-taxonomy` Cassette replay collapses a recorded ErrRequest into ErrUnavailable
  recorded.err() (cassette.go:122) stores the error as a string and re-derives the taxonomy from Stop. A recorded 400 — bad schema or unknown model, the class errors.go:20 says must stay LOUD — carries Stop:"" , so ErrorForStop returns nil and the replay falls through to `fmt.Errorf("%w: %s", llm.ErrUnavailable, r.Err)`, the quiet class every consumer degrades on. Recording the taxonomy member itself (or the HTTP status) instead of re-deriving it removes the second classification path.
- **BR-39** [Important] `unclassified-failure-mode` The capture-drift conformance suite reports drift when the proxy is merely unreachable
  capture_conformance_test.go:31 skips only when llm.Resolve fails (no key). With a key set and the proxy stopped, Complete returns ErrUnavailable and every subtest t.Fatalf's — telling the operator "the model may no longer think by default ... Re-record: scripts/llm-probe.sh record" when nothing drifted. Plan Task 8 Step 2 asked for skip-not-fail explicitly. THIS IS THE 2ND FINDING IN FAMILY `unclassified-failure-mode`: the rule is that a check must distinguish "dependency unreachable" from "dependency changed" before reporting either, and internal/llm/conformance_test.go:30 has the same gap, so fix both.
- **BR-40** [Important] `fake-silently-ignores-inputs` Both test doubles added this window discard the llm.Request entirely
  checkClient (cmd/define/llmcheck_test.go:14) and stubLive (internal/llm/llmtest/cassette_test.go:17) both take `context.Context, llm.Request` and name neither parameter. Consequence: nothing asserts that --llm-check's request is well-formed — MaxTokens:2048, Task:"llm-check" and the PONG prompt could all be dropped and the four llmcheck tests stay green, while the real flag 400s. The repo already has a wire fake (llmtest.NewFake + llm.New) that would catch it and is importable from cmd/define.
  THIS IS THE 4TH FINDING IN FAMILY `fake-silently-ignores-inputs`. The RULE: a double must either record its input for assertion or be replaced by the wire-level fake that already exists for that dependency; a double whose method signature discards its request parameter cannot fail for any reason related to what was asked. Sweep every type in the tree implementing llm.Client, cmd/define's fetch seam, and the store seams, and confirm each records or asserts its input.
- **BR-41** [Important] `plan-revision-not-appended` The plan still describes an M2 design that was not built, with no Revisions entry
  Undeclared deltas: Cassette's seam, API and storage path all changed (see the double-above-the-seam finding); the flag is -update, not -record; Task 9's golden snapshot was dropped; the Integration points table names `llmtest.Golden` where the identifier is `AssertGolden` and has no row for llmtest.Cassette or for renderRequest/RequestHash (internal/llm/render.go), both delivered.
  THIS IS THE 2ND FINDING IN FAMILY `plan-revision-not-appended`. The rule per AGENTS.md section 1: any divergence from a plan artifact discovered during implementation is appended as a timestamped `## Revisions` delta in the SAME commit that diverges — so the sweep is not "add one entry now" but "diff the plan's Core concepts and Task lists against the tree at each milestone close and append what moved".
- **BR-42** [Minor] `dead-code` `_ = llmtest.Capture` exists only to keep an import alive
  capture_conformance_test.go:132. The comment calls it "the committed artifacts this run is checking", but the statement checks nothing — llmtest is otherwise unused in the file.
  THIS IS THE 4TH FINDING IN FAMILY `dead-code`. The RULE: a statement whose only effect is to satisfy the compiler is not documentation — drop the import and put the sentence in the doc comment, or make the reference load-bearing (here: read the capture and compare a field against the live response, which is what the file claims to do).
- **BR-43** [Minor] `redundant-test-duplicates-existing` TestAQueueStillAdvancesWhileItHasEntries duplicates TestQueueServesInOrder
  fake_test.go:163 vs fake_test.go:80 — same script shape (429 then a text reply), same two assertions, different string literals. Verified: with the sticky change reverted, the new test still passes, so it pins nothing the older one does not. ARCH-DRY.
- **BR-44** [Minor] `mode-flag-arity-guard` `define -llm-check <word>` silently ignores the word
  main.go:279 returns before the arity switch. The comment immediately above calls --llm-check "a mode, like --forget", but --forget has an explicit guard (main.go:288: "-forget takes the word to remove; do not also pass one") added because "silently honouring one of them is how -raw came to mean two different things in #2". Same guard, same reason.
- **BR-45** [Minor] `diagnostic-ignores-config` --llm-check hardcodes MaxTokens 2048 instead of the resolved cfg.MaxTokens
  llmcheck.go:44. config.go:23 records that an under-budgeted max_tokens let adaptive thinking consume the whole allowance and returned an answer cut mid-rune — the committed message-truncated.json. A truncated PONG surfaces as ErrTruncated and exits 1, so the diagnostic would report a healthy configuration as broken.
- **BR-46** [Minor] `schema-metadata-applied-blindly` additionalProperties:false is set unconditionally, including on non-object schemas
  schema.go:66. Measured: SchemaFor[string]() -> {"type":"string","additionalProperties":false}; SchemaFor[map[string]string]() -> {"type":"object","additionalProperties":false}, an object that permits no keys at all. The adjacent comment justifies stripping $schema/$id because "the provider rejects a schema carrying JSON Schema metadata it does not use" — the same argument applies to additionalProperties on a string or array.
- **BR-47** [Minor] `test-helper-fatal-off-goroutine` Cassette.Client's t.Fatalf fires from whatever goroutine a consumer calls Complete on
  cassette.go:60/66/74/80/84 call c.store.t.Fatalf from inside an llm.Client, a value designed to be handed to arbitrary consumer code including concurrent authoring loops. fake.go:326 states the opposing rule for the same package ("NOT t.Fatalf: this runs on the server's goroutine, where Fatalf becomes a hang or a 'log after test completed' panic").
  THIS IS THE 3RD FINDING IN FAMILY `test-helper-fatal-off-goroutine`. The RULE: a helper may call t.Fatalf only if it is structurally guaranteed to run on the test goroutine; anything returned to a caller as a value (a Client, a handler, a callback) must return an error instead. Enumerate every exported llmtest constructor that captures *testing.T and classify each by that criterion.
- **BR-48** [Minor] `test-flag-mutation-leaks` TestCassetteReplaysTheTaxonomy sets *update without a defer
  cassette_test.go:107-112 does `*update = true` ... `*update = false` inline; a Fatal in the Complete call between them leaks -update into every subsequent test in the package, turning AssertGolden from a comparator into a writer. recordThenReplay (cassette_test.go:34) uses defer correctly; this site does not.
- **BR-49** [Minor] `artifact-omits-the-question` A cassette records the answer but not the question
  recorded (cassette.go:117) stores only Response and an error string; the filename carries task plus a 12-hex hash. TestCassetteOnDiskIsReadable asserts the ANSWER is legible in a diff, but a reviewer cannot tell what was asked without recomputing the hash. Storing llm.RenderRequest(r) alongside would make the artifact self-describing, and is free — the miss message already renders it.

## Round 6 — 2026-08-23T00:28:07-07:00 (claude) — BLOCKED

### Disposed

- BR-33 — addressed — Reversion-verified: removing requireSchemaFields reddens 3 table cases and FuzzDecode seed#1. See new finding for the depth-1 limit.
- BR-34 — addressed — All four claims now hold; 7 of 7 drift failures name the script. render.go's claims are re-raised as a new finding, not this one.
- BR-35 — addressed — testdata/golden/schema-veto-verdict.txt committed; adding a struct field to vetoVerdict reddens TestSchemaGoldenIsStable.
- BR-36 — addressed — README has a "Checking the model connection" section and the exit-code paragraph names --llm-check.
- BR-37 — addressed — Rebuilt as http.RoundTripper under Config.Transport; replay against 127.0.0.1:1 proves the SDK path still runs.
- BR-38 — addressed — Reversion-verified: hardcoding jsonResponse(200, …) reddens both subtests of TestCassetteReplaysTheTaxonomyFromTheStatus.
- BR-39 — not-addressed — capture_conformance_test.go fixed; internal/llm/conformance_test.go:30, named in the finding, is unchanged.
- BR-40 — addressed — Reversion-verified: mutating MaxTokens to 512 and the PONG prompt each redden a distinct assertion.
- BR-41 — addressed — Revisions entry appended and both tables corrected; Task 11's prose was missed and is covered by the new coupling finding.
- BR-42 — not-addressed — `_ = llmtest.Capture` still present at capture_conformance_test.go:142.
- BR-43 — not-addressed — TestAQueueStillAdvancesWhileItHasEntries still present and still the same shape as TestQueueServesInOrder (fake_test.go:80).
- BR-44 — not-addressed — main.go still returns on *llmCheck before the arity switch; `define -llm-check hello` ignores the word.
- BR-45 — not-addressed — llmcheck.go:40 still hardcodes MaxTokens: 2048 rather than reading cfg.MaxTokens.
- BR-46 — not-addressed — schema.go:71 still sets additionalProperties:false unconditionally; SchemaFor[string]() still returns it on a string schema.
- BR-47 — addressed — No t.Fatalf remains in cassette.go; the enumeration holds — every other Fatalf in llmtest is on the test goroutine.
- BR-48 — addressed — Defer added; the restore-to-literal residual is raised as a new finding in the same family rather than re-raised here.
- BR-49 — addressed — exchange.Request stores the wire body; TestCassetteOnDiskIsSelfDescribing asserts the question is legible.

### Raised

- **BR-50** [Important] `single-source-consumer-not-derived` The cassette no longer derives from renderRequest, leaving RequestHash dead and five artifacts asserting a coupling that is measurably false
  Measured: two Requests differing only in Task recorded to ONE cassette file (f3f94dfd72ba.json) while RequestHash(a)=c94ee5919d42, RequestHash(b)=d1d164764fc2 and RenderRequest differs — the second recording silently overwrote the first. cassette.go:51 keys on sha256(wire body minus max_tokens); renderRequest is no longer its input. RequestHash has zero non-test callers yet render.go:57 says "RequestHash keys a cassette"; render.go:15 says "llmtest.Golden prints it … and llmtest.Cassette hashes it to key a recording" (llmtest.Golden is still not an identifier, in the very file BR-34's rule named for grep-verification); golden.go:28, atlas/llm.md:140 and plan Task 11 repeat the claim; and TestGoldenAndCassetteKeyMoveTogether plus TestEveryMeaningfulFieldReachesTheHash now pin a function nothing uses, so the real cassette key has no field-coverage test.
  THIS IS THE 2ND FINDING IN FAMILY `single-source-consumer-not-derived`. Do not patch the five sentences. The RULE: when a refactor moves a consumer off a declared single source, the source is either re-wired to that consumer or deleted in the SAME change — an exported function with no caller plus docs asserting its role is a source that has quietly become documentation. THE ENUMERATION to sweep: for every "the same X that Y" / "single source" / "exactly one renderer" claim in internal/llm and internal/llm/llmtest, grep that Y actually calls X, and for every exported identifier in internal/llm confirm a non-test caller exists or the export is justified in its doc.
- **BR-51** [Important] `enforcement-not-pinned-by-a-test` requireSchemaFields checks only top-level required fields, so a nested object still decodes to a partial value with a nil error
  Measured against the shipped code with type outer{Fits bool; Inner inner} where inner{Score int; Detail string}: SchemaFor emits "required":["score","detail"] on the nested object, and decode[outer](`{"fits":true,"inner":{}}`) returns {Fits:true Inner:{Score:0 Detail:""}} with err=nil; `{"fits":true,"inner":{"score":3}}` likewise. task.go:61 still claims "require every field the schema marks REQUIRED to be present" and task.go:75 still claims "it never returns a partially populated value with a nil error"; atlas/llm.md repeats both. #10's authoring result is the first consumer likely to be nested. Also unchecked: an object inside an array, and an explicit null for a required object field (present, so it passes, and zero-fills).
  THIS IS THE 5TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not fix only the nested case. The RULE: a check written to satisfy a finding must be written against the SHAPE the invariant quantifies over, not against the example the finding used — here the invariant quantifies over the whole schema tree, so the check must walk it (or the doc must state the depth limit, and then the limit needs its own test). THE ENUMERATION to sweep in this round: for each universal quantifier in the doc comments of task.go, schema.go, render.go, cassette.go and golden.go, write down the set it ranges over and confirm a test exists at every point of that set — depth for schemas, both branches for the fuzz target, every method for a seam double.
- **BR-52** [Important] `double-covers-partial-seam` A cassette cannot record a streaming exchange, and the failure is delivered as ErrUnavailable
  exchange.Response is json.RawMessage (cassette.go:44), so an SSE body cannot be marshalled. Measured, recording a Stream call through the transport against the wire fake: `llm: unavailable: Post ".../v1/messages": json: error calling MarshalJSON for type json.RawMessage: invalid character 'e' looking for beginning of value`. Half the Client interface is unrecordable, and a harness marshalling bug arrives wearing the class every consumer is designed to absorb silently — the collapse BR-38 existed to prevent, now on a different path. cassette.go:23 justifies the transport placement on the grounds that replay "parses an SSE frame", describing a path recording cannot produce. Replay itself would work: ssestream.NewDecoder ignores the application/json content-type jsonResponse hardcodes. Fix: store the body as bytes plus the recorded Content-Type, and return a harness error that is not in the dependency's absorbable class.
- **BR-53** [Minor] `test-flag-mutation-leaks` withUpdate restores *update to the literal false rather than its prior value, silently cancelling a real -update run
  cassette_test.go:31. Measured: `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` FAILS ("a changed prompt passed its golden"); the full-package run only passes because the record test runs first and clobbers the flag back to false before the golden tests see it. So the documented refresh mechanism is order-dependent and self-cancelling in the package that defines it.
  THIS IS THE 2ND FINDING IN FAMILY `test-flag-mutation-leaks`. The RULE: a helper that mutates process-global state must capture the prior value and restore THAT — `defer func(prev bool) { *update = prev }(*update)` — because restoring to a constant is indistinguishable from a leak whenever the constant is not what the operator passed.
- **BR-54** [Minor] `memoised-value-is-caller-mutable` SchemaFor returns the memoised map by reference, so any consumer mutation poisons the cache process-wide
  schema.go:27 returns the cached map[string]any itself. A consumer doing `s, _ := llm.SchemaFor[T](); s["description"] = "..."` permanently changes what every later Run[T] sends on the wire and what requireSchemaFields reads. Either clone on read or document the value as read-only and return it through a type that says so; this is a new internal package five downstream issues will consume.

## Round 7 — 2026-08-23T07:45:26-07:00 (claude) — BLOCKED

### Disposed

- BR-39 — not-addressed — Both sites now call SkipIfUnreachable, but the helper is untested and skips on a renamed model (502 to ErrUnavailable) — the drift it exists to catch.
- BR-42 — not-addressed — `_ = llmtest.Capture` still present at capture_conformance_test.go:142.
- BR-43 — not-addressed — TestAQueueStillAdvancesWhileItHasEntries unchanged at fake_test.go:167, still the shape of TestQueueServesInOrder (fake_test.go:82).
- BR-44 — not-addressed — main.go:278 still returns on *llmCheck before the arity switch; `define -llm-check hello` ignores the word.
- BR-45 — not-addressed — llmcheck.go:41 still hardcodes MaxTokens 2048 rather than reading cfg.MaxTokens.
- BR-46 — not-addressed — Measured on the shipped tree: SchemaFor[string]() = {type:string, additionalProperties:false}; SchemaFor[map[string]string]() = an object permitting no keys.
- BR-50 — addressed — Reversion-verified: removing the context lookup in key() reddens TestTheKeyDerivesFromTheRequestNotTheBody with the collision printed. See the new finding for the pre-defaulting residual.
- BR-51 — not-addressed — Objects and explicit nulls are walked; array items are not — decode[arrOuter](`{"fits":true,"list":[{}]}`) still returns a partial value with err=nil, and task.go:129 now claims "EVERY depth".
- BR-52 — addressed — Reversion-verified: restoring json.RawMessage reproduces the exact ErrUnavailable marshalling failure in TestCassetteRecordsAndReplaysAStream.
- BR-53 — not-addressed — cassette_test.go:32 still does `defer func() { *update = false }()` — the literal, not the prior value.
- BR-54 — addressed — Reversion-verified with the clone removed from BOTH return paths: all three assertions of TestSchemaIsolationAcrossCallers redden, including the nested one.

### Raised

- **BR-55** [Important] `single-source-consumer-not-derived` The cassette key is taken before the Request is defaulted, so two different effective models collide into one recording
  Measured on the clean tree: llm.Task[T]{Name:"veto", Prompt:"near-synonym?"} — the documented shape, since task.go:27 says a zero Model means "use the Config default" — recorded under Config{Model:"claude-opus-5"} and then Config{Model:"claude-sonnet-5"} wrote to ONE file, 4ea325a60ca6.json, whose on-disk request.model is claude-sonnet-5; the opus recording is gone. renderRequest hashes r.Model/r.Effort as DECLARED while params() (anthropic.go:66) resolves them from a.cfg below the hash. Two further consequences: TestEveryMeaningfulFieldReachesTheHash (render_test.go:57) passes while the property it names is false on the only production path, and a consumer's golden and its cassette describe different requests — golden_schema_test.go:31 hand-writes Model/Effort (hash a11b39ae49fe) where Run[T] sends them empty (hash fae0974f64b3).
  THIS IS THE 3RD FINDING IN FAMILY `single-source-consumer-not-derived`. Do not fix Model alone. The RULE, already stated at BR-50 and now failing on a different axis: where two derivations of one fact exist, one must derive from the other — here renderRequest is a restatement of params() that omits defaulting. Resolve Model/Effort/System onto the Request before withRequest(ctx, r) in Complete and Stream, so RenderRequest and the wire body agree by construction, and drive the field-coverage table through Run[T] end to end rather than over a hand-populated struct.
- **BR-56** [Minor] `dead-code` ErrorForStop is an exported function with zero references whose doc still asserts the role the rework removed
  grep for ErrorForStop across the tree returns only its own definition and comment (errors.go:88-92). It was added this window so "a replayed recording reconstructs the same taxonomy member", and BR-38's fix replaced that mechanism with status replay. Also in this class: `c := llm.New(...)` followed by `_ = c` in withReq (cassette_test.go:217), and BR-42's `_ = llmtest.Capture`.
  THIS IS THE 5TH FINDING IN FAMILY `dead-code`. Do not delete only this function. The RULE is BR-50's own enumeration, which was run for RequestHash and RenderRequest and not for the rest: for every exported identifier in internal/llm, confirm a non-test caller exists or the export is justified in its doc — and for every `_ = X` statement, confirm it does something other than satisfy the compiler. Run that grep and paste it, rather than fixing the three sites this finding happens to name.
- **BR-57** [Minor] `plan-revision-not-appended` Round 6 changed the design again with no Revisions entry, and two artifacts still name llmtest.Golden
  The plan's last `## Revisions` entry is the round-5 one. Undeclared since: the cassette key moved to a context-carried RequestHash, `exchange` gained ContentType and became text, and llmtest.SkipIfUnreachable is new exported surface with no row in the Integration points table. Plan line 1638 and issue line 146 both still say `llmtest.Golden`, which is not an identifier — the same string BR-34 and BR-50 each named for grep-verification.
  THIS IS THE 3RD FINDING IN FAMILY `plan-revision-not-appended`. The RULE stated at BR-41 was "diff the plan's Core concepts and Task lists against the tree at each boundary and append what moved" — it was run once, at round 5, and not at round 6. Make the diff a step of the boundary itself rather than a response to a finding, and grep the plan and issue for every identifier they name in the same pass.
- **BR-58** [Minor] `stdlib-reimplemented` jsonEscape double-escapes the request the cassette miss message exists to show
  cassette.go:206. The envelope is built with json.Marshal, which escapes already, so the operator sees `request was: {\\\"max_tokens\\\":8192,...\\\\\\\"obsequious\\\\\\\"...}` — measured. The message's whole purpose is to be readable without recomputing a hash.
  THIS IS THE 3RD FINDING IN FAMILY `stdlib-reimplemented`. The RULE: before hand-rolling a string transform, check whether the encoder that consumes the value already performs it — a manual escape applied to a value that is later marshalled is always a double-escape, never a no-op. Sweep the diff for hand-written quoting/escaping helpers and confirm each sits outside an encoder, not inside one.

## Round 8 — 2026-08-23T08:43:06-07:00 (claude) — passed

### Disposed

- BR-39 — not-addressed — Helper fixed and tested, but capture_conformance_test.go:42's inline closure still skips on ErrUnavailable — measured: a renamed model answers 502, classifies ErrUnavailable, and all four drift subtests SKIP.
- BR-42 — addressed — `_ = llmtest.Capture` is gone; the llmtest import is now load-bearing via SkipIfUnreachable.
- BR-43 — not-addressed — Re-verified by reversion: with sticky reverted, TestTheLastScriptedReplyIsSticky reddens while TestAQueueStillAdvancesWhileItHasEntries (fake_test.go:166) stays green.
- BR-44 — not-addressed — Measured: `define -llm-check hello` runs the check and discards the word; main.go:278 still returns before the arity switch.
- BR-45 — not-addressed — llmcheck.go:41 still hardcodes MaxTokens 2048 rather than reading cfg.MaxTokens.
- BR-46 — not-addressed — Measured: SchemaFor[string]() = {type:string, additionalProperties:false}; a top-level slice type now also gets it on a type:array schema.
- BR-51 — addressed — Reversion-verified for arrays: removing the []any branch reddens all three assertions of TestDecodeRequiresFieldsAtEveryShape. Map values are a new finding, not this one re-raised.
- BR-53 — not-addressed — Re-measured: `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still FAILS; cassette_test.go:32 restores the literal false.
- BR-55 — addressed — Reversion-verified: dropping a.effective(r) reddens TestRequestInContextCarriesTheEffectiveModel ("the context carried an unresolved model"). Residual duplication raised separately.
- BR-56 — not-addressed — Ran the enumeration: ErrorForStop still has 0 non-test refs (errors.go:88), and `_ = c` is still at cassette_test.go:242 guarding an unused llm.New at :231.
- BR-57 — not-addressed — Revisions entry appended and llmtest.Golden corrected in plan and issue, but the Integration points table (plan:152) still has no SkipIfUnreachable row — one of the three deltas the finding named.
- BR-58 — addressed — jsonEscape removed; measured miss message now carries only the single escaping JSON-in-JSON requires.

### Raised

- **BR-59** [Important] `enforcement-not-pinned-by-a-test` The required-field walk skips additionalProperties, so a map-valued object still decodes to a partial value with a nil error
  Measured on the clean tree with type zzMapOuter{Fits bool; By map[string]zzItem} where zzItem{Score int; Detail string}: SchemaFor emits by.additionalProperties = {type:object, required:[score,detail]}, and decode(`{"fits":true,"by":{"a":{}}}`) returns {Fits:true By:map[a:{Score:0 Detail:""}]} with err=nil; `{"fits":true,"by":{"a":{"score":1}}}` likewise. missingRequired recurses through schema["properties"] (task.go:154) and schema["items"] (task.go:169) and never through additionalProperties, which is what jsonschema.Reflector emits for a Go map. task.go:126 claims the walk covers "the WHOLE schema tree: object properties, array items, and non-object payloads" — an enumeration of three where there are four — and atlas/llm.md:112 claims "require every field the schema marks required". Separately, task.go:108 still says "Only object payloads are checked", which round 7's own array and scalar branches falsified.
  THIS IS THE 6TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not add an additionalProperties case and stop — that is the fourth instance-fix in a row (top-level BR-33, nested object BR-51, array item BR-51 re-raise, map value here). The RULE: the walk must be driven by the schema GENERATOR's nesting vocabulary, not by the payload shapes a finding happened to name. THE ENUMERATION, mechanically available: the subschema-bearing keywords jsonschema.Reflector can emit for a Go type under this configuration — properties, items, additionalProperties, and (latent under DoNotReference:true) $defs/$ref and oneOf/anyOf. Write that list into the doc comment, cover each arm, and pin each. The test-side half of the same rule: FuzzDecode's success branch asserts got.Reason != "" — a named field of one fixture type — so it ranges over `answer` alone, which is why every vehicle has had to be found by a reviewer. Replace it with a reflective check driven over a fixture list of struct / nested struct / slice-of-struct / map-of-struct / pointer-to-struct / top-level-slice.
- **BR-60** [Minor] `docs-claim-absent-surface` The Revisions entry written to establish grep-verification claims a table row that does not exist
  workshop/plans/000011-vocab-llm-plan.md:2138 states "`llmtest.SkipIfUnreachable` is new exported surface (added to the Integration points table above)". The table at plan:152 has eight rows and none of them is SkipIfUnreachable; llm.RequestFromContext is likewise absent. The claim sits inside the entry whose stated purpose is that the boundary "greps every identifier the plan and the issue name in the same pass".
  THIS IS THE 5TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not just add the row. The RULE, first stated at BR-34 and unchanged: a doc claim naming a code identifier or an on-disk artifact must be grep-verified against the tree in the SAME edit that writes it — and that applies to a claim about the document being edited, not only to claims about code. A revision entry asserting "added to X" is a claim about X's current contents; check X. The cheap enforcement is to write the table row first and the sentence second, so the sentence describes a state that already exists.
- **BR-61** [Minor] `single-source-consumer-not-derived` effective() and params() are now two independent defaulting implementations of the same three fields
  anthropic.go:70 resolves Model/Effort/MaxTokens against a.cfg for the context-carried Request; anthropic.go:77 params() independently re-does cmp.Or on the same three fields for the wire body. They agree today, so nothing is currently broken — but the fix for BR-55 added a third derivation of "what was asked" rather than collapsing to one, so a fourth Config-defaulted field added to params() would silently move the wire body without moving the cassette key, and no test would catch it because the field-coverage table (render_test.go:57) still runs over a hand-populated struct rather than through Run[T].
  THIS IS THE 4TH FINDING IN FAMILY `single-source-consumer-not-derived`. Do not add a fourth field to both functions when that day comes. The RULE, stated at BR-50 and again at BR-55: where two derivations of one fact exist, one must derive from the other. Here that is one line — `r = a.effective(r)` at the top of Complete and Stream, with params() reading the already-resolved values and its cmp.Or calls deleted — after which the wire body and RenderRequest agree by construction rather than by coincidence.

## Round 9 — 2026-08-23T10:03:37-07:00 (claude) — BLOCKED

### Disposed

- BR-27 — not-addressed — SlowEvery clamped; the class not swept — negative Timeout still returns ErrUnavailable in 1ms, negative MaxTokens still reaches the wire as -5 with err=nil.
- BR-28 — not-addressed — Stream path swept field by field; JSON path not swept at all — Stall/StallEarly/JunkFrame silently ignored on Complete, and an .sse capture on Complete surfaces as ErrUnavailable.
- BR-29 — addressed — The vacuous test is gone from the tree; only TestNewPreservesADisabledStallBound remains, and it reddens when New stops preserving a negative StallAfter.
- BR-30 — addressed — Reversion-verified — reverting splitInto to []byte reddens TestTextJoinsMultibyteBlocksWithoutCorruption while the ASCII test stays green.
- BR-31 — addressed — plan:518-520 now declares two phases and no Bytes, matching llm.go.
- BR-32 — addressed — Reply.Body and Reply.NoThinking are both gone from the struct and from every branch that read them.
- BR-39 — addressed — Measured live — with the proxy up and a renamed model, all four drift subtests now FAIL where round 8 measured them skipping. No ErrUnavailable skip remains in any conformance file.
- BR-43 — not-addressed — Re-verified by reversion — with sticky reverted, TestAQueueStillAdvancesWhileItHasEntries (fake_test.go:166) stays green while TestTheLastScriptedReplyIsSticky reddens.
- BR-44 — not-addressed — Re-measured — `define -llm-check hello` runs the check and discards the word, exit 0.
- BR-45 — not-addressed — llmcheck.go:41 still hardcodes MaxTokens 2048. Folded into the diagnostic-ignores-config class finding this round.
- BR-46 — not-addressed — schema.go:100 still unconditional; SchemaFor[string]() still returns additionalProperties:false on a string schema.
- BR-53 — not-addressed — Re-measured — `go test ./internal/llm/llmtest -update -run TestGoldenDetectsAChangedPrompt` still fails; cassette_test.go:32 restores the literal false.
- BR-56 — not-addressed — Ran the enumeration — ErrorForStop still 0 non-test refs (errors.go:92); `_ = c` still at cassette_test.go:242 guarding an unused llm.New at :231.
- BR-57 — not-addressed — Table at plan:152 still lacks SkipIfUnreachable and RequestFromContext, and the new task_conformance_test.go surface is undeclared in the plan too.
- BR-59 — addressed — Reversion-verified twice — deleting the additionalProperties arm reddens TestRequiredWalkCoversEveryNestingKeyword AND FuzzDecode/seed#8 via an independent oracle.
- BR-60 — not-addressed — plan:2138 still claims SkipIfUnreachable was "added to the Integration points table above"; the eight-row table at plan:152 does not contain it.
- BR-61 — not-addressed — Now measured stronger — making params() send a different model than effective() hashes leaves the ENTIRE suite green.

### Raised

- **BR-62** [Important] `unclassified-failure-mode` The live typed-task suite fails on ordinary model variation, asserting judgment its own doc comment disclaims
  Measured against the live proxy: 3 failures in 18 runs (~17%), two distinct modes.
  "task_conformance_test.go:97: option[3] = {Word: Why:}: a required field came back
  empty" (the model returned four options where the prompt asks for three, the fourth
  with empty strings) and "task_conformance_test.go:88: stem has no blank:
  \"placeholder\"" with "no options returned" (a degenerate stub). In both, Run[T]
  returned err=nil correctly — an empty array and an empty string both satisfy JSON
  Schema required. Three artifacts claim the test asserts shape only: the doc comment
  at :23, atlas/llm.md:207, and the commit message of e4c0364. Contains(stem,"___")
  is a formatting convention and non-emptiness is not a shape the layer guarantees.
  Two messages compound it by misattributing to the walk: :56 says "the required-field
  check should have rejected this" and :95 says "Every required field at every depth —
  the property the walk exists for", but missingRequired enforces key PRESENCE, not
  value non-emptiness, so both send the next reader hunting in a function that behaved
  correctly.
  THIS IS THE 3RD FINDING IN FAMILY `unclassified-failure-mode`. Do not fix these two
  assertions. The RULE, stated at BR-39 as "a check must distinguish dependency
  unreachable from dependency changed before reporting either", needs its third
  bucket: a live check must ALSO distinguish "the dependency changed" from "the
  dependency behaved normally but differently", and may only assert properties the
  layer under test guarantees. THE ENUMERATION, mechanically available: for each
  assertion in the three -tags conformance files, name which side guarantees it —
  the transport/decode layer (assert) or the model (log). Contains(stem,"___"),
  TrimSpace(o.Word)!="" and TrimSpace(got.Reason)!="" are all model-side and belong
  beside the verdicts already in t.Logf; the layer-side properties this test should
  assert and does not are err==nil, that the struct decoded, and that the schema
  reached the wire. The same sweep covers mustCall at capture_conformance_test.go:52,
  which Fatalfs a mid-run 429/529 as drift.
- **BR-63** [Important] `diagnostic-ignores-config` define --llm-check discards run's signal context, so Ctrl-C is swallowed for the full five-minute timeout
  main.go:154 builds signal.NotifyContext with the comment "Ctrl-C now cancels the
  context ... This changes the one-shot path too, deliberately", and run() threads
  that ctx into defineOnce and repl. main.go:278 drops it —
  runLLMCheck(os.Getenv, llm.New, stdout, stderr) takes no ctx — and llmcheck.go:35
  starts from context.Background() with cfg.Timeout, five minutes. Measured against a
  server that accepts and never answers: the process survived three SIGINTs over six
  seconds. Repeated Ctrl-C does not help, because NotifyContext leaves signal.Notify
  registered until stop(), so the default terminate behaviour is never restored and
  the process is uninterruptible until the deadline.
  THIS IS THE 2ND FINDING IN FAMILY `diagnostic-ignores-config`. Do not just add a ctx
  parameter. The RULE that covers both: the diagnostic path must use the inputs the
  caller already resolved, never reconstruct them. Measured prevalence 2 —
  llmcheck.go:41 hardcodes MaxTokens 2048 instead of cfg.MaxTokens (BR-45, still
  open), and llmcheck.go:35 reconstructs a context instead of deriving from run's.
  THE ENUMERATION is runLLMCheck's argument list against what run() already holds:
  ctx, the resolved cfg, and the writers — two of the three are reconstructed.
- **BR-64** [Minor] `docs-claim-absent-surface` atlas/llm.md says "Two tagged suites" above a list of three, and the third bullet's claim is measurably false
  atlas/llm.md:200 reads "Two tagged suites, both on-demand" and is followed by three
  bullets; e4c0364 inserted TestTypedTaskAgainstTheLiveService without touching the
  count. The same inserted bullet asserts "Asserts shape, never the model's judgment",
  which the ~17% live failure rate falsifies.
  THIS IS THE 6TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not just change
  "Two" to "Three". The RULE, unchanged since BR-34: a doc claim of universal or
  COUNTING form is a claim about an enumeration, so it may only be written after
  running that enumeration — and inserting a member into a list is an edit to every
  counting sentence that governs the list. The cheap enforcement is to grep the
  enclosing section for a cardinal before adding a bullet, and to re-read the sentence
  you are inserting under, not only the ones you wrote.

## Open findings

- **BR-27** [Important] `partial-constructor-defaults` A negative SlowEvery panics on the watcher goroutine, where no caller can recover
- **BR-28** [Important] `fake-silently-ignores-inputs` A .json capture scripted against a streaming request is silently replaced by stream-sample.sse
- **BR-43** [Minor] `redundant-test-duplicates-existing` TestAQueueStillAdvancesWhileItHasEntries duplicates TestQueueServesInOrder
- **BR-44** [Minor] `mode-flag-arity-guard` `define -llm-check <word>` silently ignores the word
- **BR-45** [Minor] `diagnostic-ignores-config` --llm-check hardcodes MaxTokens 2048 instead of the resolved cfg.MaxTokens
- **BR-46** [Minor] `schema-metadata-applied-blindly` additionalProperties:false is set unconditionally, including on non-object schemas
- **BR-53** [Minor] `test-flag-mutation-leaks` withUpdate restores *update to the literal false rather than its prior value, silently cancelling a real -update run
- **BR-56** [Minor] `dead-code` ErrorForStop is an exported function with zero references whose doc still asserts the role the rework removed
- **BR-57** [Minor] `plan-revision-not-appended` Round 6 changed the design again with no Revisions entry, and two artifacts still name llmtest.Golden
- **BR-60** [Minor] `docs-claim-absent-surface` The Revisions entry written to establish grep-verification claims a table row that does not exist
- **BR-61** [Minor] `single-source-consumer-not-derived` effective() and params() are now two independent defaulting implementations of the same three fields
- **BR-62** [Important] `unclassified-failure-mode` The live typed-task suite fails on ordinary model variation, asserting judgment its own doc comment disclaims
- **BR-63** [Important] `diagnostic-ignores-config` define --llm-check discards run's signal context, so Ctrl-C is swallowed for the full five-minute timeout
- **BR-64** [Minor] `docs-claim-absent-surface` atlas/llm.md says "Two tagged suites" above a list of three, and the third bullet's claim is measurably false
