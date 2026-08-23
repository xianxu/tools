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

## Open findings

- **BR-14** [Important] `unclassified-failure-mode` An exhausted 408 classifies as ErrRequest, the loud "our bug" arm, though the SDK retries it as transient
- **BR-15** [Important] `enforcement-not-pinned-by-a-test` Three of this round's fixes survive full reversion with a green suite, or are entered by no fixture
- **BR-16** [Important] `docs-claim-absent-surface` Four doc claims outrun the tree, two of them written by this round's own fixes
- **BR-17** [Minor] `test-helper-fatal-off-goroutine` BR-11's fix added a t.Fatalf on the fake's server goroutine, ten lines above the comment forbidding it
- **BR-18** [Minor] `comment-references-future-surface` --llm-check was removed from config.go and left in two sibling files
- **BR-19** [Minor] `test-panics-instead-of-failing` captures_test.go indexes into unchecked type assertions, so a malformed capture panics rather than fails
- **BR-20** [Minor] `stdlib-reimplemented` orDefault, orInt64 and orDuration re-implement stdlib cmp.Or
- **BR-21** [Minor] `review-archaeology-in-code` Production comments increasingly narrate prior review rounds, duplicating the gate ledger
