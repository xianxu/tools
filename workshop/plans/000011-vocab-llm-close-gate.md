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

## Open findings

- **BR-1** [Critical] `stream-failure-classification` A stalled stream discards accumulated answer text and returns ErrUnavailable, contradicting the sawEvent rule the same function documents
- **BR-2** [Critical] `fake-cannot-reach-the-branch` No test can exercise a stall that occurs after answer text has arrived
- **BR-3** [Important] `fake-tuned-to-its-fixture` The fake rejects exactly one model name and it is the test's own fixture string
- **BR-4** [Important] `partial-constructor-defaults` New defaults Timeout and MaxTokens but not StallAfter, Model or Effort
- **BR-5** [Important] `docs-claim-absent-surface` atlas and suite doc comments state a -tags conformance live run that has no file
- **BR-6** [Important] `enforcement-not-pinned-by-a-test` Nothing in go test enforces the captures' required shapes, and the script that does cannot run offline
- **BR-7** [Minor] `stale-module-metadata` go mod tidy was never run; the directly-imported SDK is still marked indirect
- **BR-8** [Minor] `test-panics-instead-of-failing` TestThinkingSignatureIsPreserved discards the error then indexes Blocks[0]
- **BR-9** [Minor] `dead-code` var _ = http.StatusOK props up an otherwise-unused net/http import
- **BR-10** [Minor] `pure-logic-on-an-io-type` response() has an unused receiver, so the package's only transformation logic is reachable only through httptest
- **BR-11** [Minor] `fake-silently-ignores-inputs` serveStream ignores Reply.Text and Reply.Stop entirely
- **BR-12** [Minor] `test-helper-fatal-off-goroutine` Capture calls t.Fatalf from the fake's server goroutine
- **BR-13** [Minor] `comment-references-future-surface` config.go references --llm-check, which does not exist until M2
