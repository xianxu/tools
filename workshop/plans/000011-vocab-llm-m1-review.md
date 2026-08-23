# Boundary Review — tools#11 (milestone M1)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b5d50ea29e4041e5681696fd47bc4e973c8f6c10^..d00a1d81faff930b4a46d1d1a496370a9325c09c |
| command | sdlc milestone-close --issue 11 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-22T19:21:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the plan committed for this boundary — the provider-independent contract, the five-member taxonomy, pure config resolution, a wire-level stateful fake, the real SDK client, and one obligation suite — and the whole repo suite is green (`go test ./...`, plus `-race` on both llm packages). The test double's placement is right and the captures-not-literals discipline is real, not claimed: I mutation-checked the one claimed fix in the Log (forwarding thinking deltas turns `TestStreamDoesNotForwardThinkingDeltas` red at `6 want 5`) and it is genuinely pinned. What blocks a clean SHIP is one correctness defect in `Stream`: a stalled stream discards every byte of answer text already accumulated and classifies as `ErrUnavailable`, directly contradicting the `sawEvent` rule the same function documents fifteen lines below and that `atlas/llm.md` states as the contract. Two supporting gaps let it through — the fake stalls at a `thinking_delta`, so the stall-after-text case never runs, and the fake's unknown-model rejection is keyed to a literal test string, so its most load-bearing obligation passes for the wrong reason.

## 1. Strengths

- **The fake is where the plan said it should be, and it earns the placement.** `internal/llm/llmtest/fake.go` is an `httptest` server speaking the Anthropic wire protocol, so the SDK's serialization, retry, and SSE parsing all run under test. `TestRetriesOn429` asserting `reqs[1].Prompt() == "hello"` (anthropic_test.go:141) catches the consumed-body class of bug a stubbed `Client` cannot see.
- **Content comes from committed captures, and the captures disagree with each other on purpose.** `[thinking,text]`, `[text]`, `[thinking,text,thinking]` across three files means no test can bake in a block sequence. `TestBlocksPreserveArrivalOrder` (anthropic_test.go:112) is the assertion that would have caught the shape the earlier plan rounds got wrong twice.
- **`Resolve` is pure over the lookup and proves it.** `TestResolveIsPureOverTheLookup` (config_test.go:89) sets a real env var and asserts `Resolve` ignores it — the assertion most people write as a comment.
- **`ErrTruncated` is a real distinction, backed by a preserved specimen.** `message-truncated.json` decodes to `{"verdict":"yes","reason":": Ā"}` with every required field present; only `stop_reason` catches it, and `response()` returns the built response *with* the error so a caller can still inspect it (anthropic.go:216).
- **The stalled-fake waits on a cleanup channel rather than sleeping** (fake.go:333) — the fix for `httptest.Server.Close` blocking, and the reason the package suite runs in 8s not 65s.

## 2. Critical findings

**C1 — `internal/llm/anthropic.go:154`: a stalled stream throws away the answer that already arrived, and calls a reachable service unreachable.**

The stall branch returns `Response{}`, discarding everything `msg` accumulated, and classifies as `ErrUnavailable`. Fifteen lines below, the salvage branch does the opposite for the same situation (anthropic.go:168), and `atlas/llm.md:60` states the rule as contract: *"A stream that dies mid-reply returns its partial text with `ErrTruncated` rather than `ErrUnavailable`: once a frame has arrived the service is demonstrably reachable."*

Verified with a scratch peer that replays the recorded frames through the first `text_delta` then goes silent:

```
onDelta saw "**Obsequious.**\n\nThe clue after"
resp.Text  = ""
err        = llm: unavailable: stream went silent for 500ms (phase streaming)
ErrTruncated = false
```

Two consequences for the consumers that branch on this: a `Stream` caller passing `onDelta == nil` (the shape `llmtest.Suite` itself uses) loses the whole partial answer, and a single mid-answer hang tells the caller "stop trying for a while" when the correct instruction is "skip this question" — the exact inversion kbench's `is_outage` line is cited for in the plan's Robustness bar.

Fix sketch: in the `if s` branch, build the response before returning and route it through the same discriminator —

```go
if s {
    partial, _ := a.response(&msg, time.Since(start))
    if sawEvent {
        return partial, fmt.Errorf("%w: stream went silent for %s after %d bytes of text",
            ErrTruncated, a.cfg.StallAfter, len(partial.Text))
    }
    return Response{}, fmt.Errorf("%w: stream went silent for %s before any frame (phase streaming)",
        ErrUnavailable, a.cfg.StallAfter)
}
```

If instead the intent is that a stall is *always* `ErrUnavailable` (the plan's Robustness bar says so), then the classification is a deliberate exception and `atlas/llm.md:58-62` plus the `sawEvent` comment at anthropic.go:116 both need to say so — but the partial-text discard is a defect either way.

**C2 — no test can reach the case in C1.** `Fake.serveStream` triggers `Stall` on `strings.Contains(fr, "content_block_delta")` (fake.go:328), and the first such frame in `stream-sample.sse` is the `thinking_delta` at index 0. So `TestStreamStallIsDetected` only ever exercises stall-before-any-text, where `Response{}` is coincidentally correct. Add a `StallAfterText bool` (or gate on `text_delta`, as `JunkFrame` already does at fake.go:322) and assert the partial text survives.

## 3. Important findings

**I1 — `internal/llm/llmtest/fake.go:128` (ARCH-MOCK): the fake rejects exactly one model name, and it is the test's own fixture.**

```go
if strings.HasPrefix(m, p) && !strings.Contains(m, "not-a-real") {
```

The comment above it claims "the point is to reject a typo, not to be a registry that goes stale" — but the prefix check accepts every typo that starts with `claude-`. Verified: changing the fixture to `claude-opus-6` makes `TestUnknownModelIsRejectedLikeTheProxyDoes` fail with `status = 200, want 502`. So the fake answers 200 where the measured proxy answers 502, and `llmtest.Suite`'s *"an unknown model errors without panicking"* obligation passes only because the fixture string carries the magic substring. This is the fake tuned to its test rather than to the dependency — precisely the divergence the shared obligation suite exists to catch. Fix: hold a small set of real model ids (a handful from the measured 31 is enough) and reject anything not in it.

**I2 — `internal/llm/anthropic.go:31`: `New` defaults `Timeout` and `MaxTokens` but not `StallAfter`, `Model`, or `Effort`.** A caller who writes `llm.New(llm.Config{BaseURL: u, APIKey: k})` — the natural first thing a downstream consumer types — gets no stall detection at all, so a hung stream blocks for the full 5-minute deadline. The stall bound is the one protection the SDK does not supply, and it is the one the constructor silently omits. Since `Config` is this package's whole public input and five consumers are queued behind it, default all five in `New` (or none, and make `Resolve` the only documented way to build one).

**I3 — `atlas/llm.md:98` and `internal/llm/llmtest/suite.go:20` state a live conformance run that does not exist.** Both say `Suite` runs "against the live proxy under `-tags conformance`", and `suite_test.go:12` names a `conformance_test.go`. The only conformance-tagged file in the tree is `cmd/define/fetch_conformance_test.go`; the llm one is Task 12, deferred to M2. The plan is consistent — the atlas is not, and the atlas is supposed to be the current-state map. Either mark it as M2-pending or add the file at this boundary.

**I4 — nothing in `go test` enforces the captures' required shapes.** `testdata/README.md` tabulates them as "enforced by `llm-probe.sh verify`", but nothing runs that script, and it can't be run without the proxy config: `K="$(key)"` executes unconditionally at scripts/llm-probe.sh:20, so `CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify` dies on a `FileNotFoundError` even though `verify` needs neither key nor network. Given `fake-models-unobserved-shape` produced three plan-gate findings, the durable form of that fix is a Go test — `llmtest` already embeds every capture, so `TestCapturesExhibitTheirShapes` is a dozen lines and runs on every `go test`. Separately, move `K="$(key)"` into `post`/the `record` arm so `verify` is offline-runnable.

## 4. Minor findings

- `go.mod:11` — `go mod tidy` was never run; `anthropic-sdk-go` is imported directly by `anthropic.go` but still marked `// indirect`. Plan Task 1 Step 1 explicitly said tidy runs at the end of Task 5. Verified: tidy promotes it to the direct block and changes nothing else.
- `internal/llm/anthropic_test.go:90` — `got, _ := ...Complete(...)` then `got.Blocks[0]` at :91. When the call errors this **panics** (`index out of range [0] with length 0`) instead of failing, which aborts the whole package run and masks every other result. Check the error.
- `internal/llm/anthropic_test.go:381` — `var _ = http.StatusOK`, a dead line propping up an otherwise-unused `net/http` import. Drop both.
- `internal/llm/anthropic.go:185` — `response()`'s receiver `a` is unused; it is a pure function bolted to the IO type (ARCH-PURE). Promoting it to package level would let block assembly and `Text`-joining be unit-tested directly from a capture via `json.Unmarshal`, with no httptest server.
- `internal/llm/config.go:99` — comment references `--llm-check`, which does not exist until M2.
- `internal/llm/llmtest/fake.go:305` — `serveStream` ignores `Reply.Text` and `Reply.Stop` entirely, so `f.Script("x", Reply{Text: "…", Stop: "refusal"})` silently does nothing on a streaming request. Either honour them or make the fake fail loudly on the combination.
- `internal/llm/llmtest/fake.go:311` — `Capture(f.t, name)` calls `t.Fatalf` from the server's goroutine. A missing capture (or a request arriving after cleanup) becomes a hang or a "log after test completed" panic rather than a clean failure.
- `internal/llm/anthropic_test.go:349` — `io := []byte(...)` shadows the package name `io`.
- Comment density runs 48–54% of lines in `llm.go`/`errors.go`. Much of it is verbatim-duplicated across `llm.go`, `anthropic.go`, and `atlas/llm.md` (e.g. the thinking-deltas rationale appears three times), so a behavior change now needs three edits to stay honest. Not worth churning now; worth watching as M2 lands.

## 5. Test coverage notes

Coverage is strong where it counts, and I confirmed two mutations go red as claimed: forwarding thinking deltas breaks `TestStreamDoesNotForwardThinkingDeltas` (`6 want 5`), and deleting `option.WithAPIKey` breaks three tests. Gaps:

- **Stall-after-text is unreachable** (C2) — the case where the C1 defect is observable.
- **The credential never reaches an assertion.** Deleting `WithAPIKey` is caught only by the SDK's own client-side credential validation, not by the wire. The fake never inspects `x-api-key`, so a key sent under the wrong header — or dropped on the retry — would pass everything. `TestCompleteRoundTrip` already checks `anthropic-version`; add the key alongside it, and check it on `reqs[1]` in `TestRetriesOn429` too.
- **Capture shape drift** (I4) is unguarded by the Go suite.
- `Redact` is tested as a pure function, but nothing asserts the key is absent from a rendered *error* — the path the doc comment claims ("every diagnostic path goes through it"). Cheap to add now, before `--llm-check` exists to leak it.

## 6. Architectural notes

- **ARCH-DRY — pass.** `response()` and `watch()` are deliberately shared between `Complete` and `Stream` so the two paths cannot disagree about `Text` assembly or phase vocabulary, which is the right instinct at exactly the right seam. `orDefault`/`orInt64` are a legitimate type split, not duplication. Only prose duplicates (Minor, above).
- **ARCH-PURE — pass with one flag.** `Resolve` taking `getenv` rather than calling it is the model answer, and the taxonomy is pure and table-tested with no IO. Flagged: `response()`'s unused receiver (Minor) — the one piece of real transformation logic in the package is reachable only through an HTTP server.
- **ARCH-PURPOSE — pass.** The boundary delivers the transport, not the easy subset: streaming shipped now rather than deferred to #16 (deferring would have changed the interface, the fake, and the suite at once), and `Blocks`/`Signature`/`Raw` are preserved for #16's byte-for-byte echo rather than flattened. The issue's testable Done-when (`ErrUnavailable` for no key / no network / 429 / 5xx, distinguishable from a loud `ErrRequest`) is delivered and pinned. The measured 502-for-unknown-model limitation is recorded honestly in the Log and the atlas rather than papered over — the right call, since `--llm-check` (M2) is where it should be caught loudly. C1 is the one place the *taxonomy itself* — the actual deliverable of this milestone — is internally inconsistent, which is why it is worth fixing before five consumers branch on it.
- **ARCH-MOCK — flag.** The placement is exemplary: production flow and test flow share the same wire boundary, the fake is stateful across calls (ordered matchers with per-key queues, an in-order request log), and it models the proxy's *measured* 502 rather than the direct API's 400. Three flags: the `not-a-real` special case (I1) means the fake diverges from the modeled dependency for every typo but one; the live conformance half is asserted but absent (I3); and capture-shape drift is guarded only by a script nothing runs and that cannot run offline (I4).

**For M2:** `Run[T]` must call `classifyStop` before `decode` — the plan says so and `message-truncated.json` exists to prove why; make that ordering a test that fails against a decode-first implementation. `renderRequest` landing before both `Golden` and `Cassette` is the right sequencing; keep the schema `map[string]any` marshalled with sorted keys or every cassette will miss at random.

## 7. Plan revision recommendations

The plan's Core concepts table matches the code for every M1 row (`Request`/`Response`/`Usage`/`Client` in `llm.go`, `Config`+`Resolve`+`Redact` in `config.go`, taxonomy in `errors.go`, `anthropicClient` in `anthropic.go`, `Fake` and `Suite` in `llmtest/`); the remaining rows are M2 and correctly absent. No contradiction to record there.

One revision is warranted, and only if C1 is resolved in the direction of keeping `ErrUnavailable`:

- **`## Revisions` entry — "the stall bound is an exception to the sawEvent rule"**: the Robustness bar states both *"no delta for `StallAfter` and the call aborts as `ErrUnavailable`"* and kbench finding 5 (*"an outage is not a stall"*), which point opposite ways for a stall that occurs after frames have arrived. Record which one governs, and why, so M2's consumers are not left to infer it from the code. If instead the code changes to `ErrTruncated`-with-partial-text, no plan revision is needed — only `atlas/llm.md` stays as written and the plan's `StallAfter` sentence should be corrected in place.

```findings
findings:
  - id: new
    severity: Critical
    family: stream-failure-classification
    title: |
      A stalled stream discards accumulated answer text and returns ErrUnavailable, contradicting the sawEvent rule the same function documents
    detail: |
      internal/llm/anthropic.go:154 returns Response{} on stall, throwing away every
      byte msg accumulated, and classifies as ErrUnavailable. Fifteen lines below,
      anthropic.go:168 does the opposite for the same situation, and atlas/llm.md:60
      states the preserving behaviour as the contract. Verified with a peer that
      replays the recorded frames through the first text_delta then goes silent:
      onDelta saw "**Obsequious.**\n\nThe clue after" while resp.Text was "" and
      errors.Is(err, ErrTruncated) was false. A Stream caller passing onDelta == nil
      loses the whole partial answer, and a mid-answer hang tells the caller to stop
      trying rather than to skip the question.
  - id: new
    severity: Critical
    family: fake-cannot-reach-the-branch
    title: |
      No test can exercise a stall that occurs after answer text has arrived
    detail: |
      Fake.serveStream triggers Stall on strings.Contains(fr, "content_block_delta")
      at internal/llm/llmtest/fake.go:328, and the first such frame in
      stream-sample.sse is the thinking_delta at index 0. So TestStreamStallIsDetected
      only ever runs stall-before-any-text, where returning Response{} is
      coincidentally correct. Gate on text_delta the way JunkFrame already does at
      fake.go:322, and assert the partial text survives.
  - id: new
    severity: Important
    family: fake-tuned-to-its-fixture
    title: |
      The fake rejects exactly one model name and it is the test's own fixture string
    detail: |
      internal/llm/llmtest/fake.go:128 accepts any model with a known prefix unless it
      contains the literal "not-a-real". Verified: changing the fixture to
      claude-opus-6 makes TestUnknownModelIsRejectedLikeTheProxyDoes fail with
      "status = 200, want 502". The fake therefore answers 200 where the measured
      proxy answers 502 for every typo but one, and llmtest.Suite's unknown-model
      obligation passes only because the fixture carries the magic substring
      (ARCH-MOCK). Hold a small set of real model ids instead.
  - id: new
    severity: Important
    family: partial-constructor-defaults
    title: |
      New defaults Timeout and MaxTokens but not StallAfter, Model or Effort
    detail: |
      internal/llm/anthropic.go:31. A consumer writing llm.New(llm.Config{BaseURL: u,
      APIKey: k}) silently gets no stall detection, so a hung stream blocks for the
      full 5-minute deadline. StallAfter is the one bound the SDK does not supply and
      the one the constructor omits. Config is this package's whole public input with
      five consumers queued behind it: default all five in New, or none and make
      Resolve the only documented constructor path.
  - id: new
    severity: Important
    family: docs-claim-absent-surface
    title: |
      atlas and suite doc comments state a -tags conformance live run that has no file
    detail: |
      atlas/llm.md:98 and internal/llm/llmtest/suite.go:20 both say Suite runs against
      the live proxy under -tags conformance, and internal/llm/suite_test.go:12 names
      a conformance_test.go. The only conformance-tagged file in the tree is
      cmd/define/fetch_conformance_test.go; the llm one is plan Task 12, deferred to
      M2. The plan is consistent, the atlas is not, and the atlas is the
      current-state map (ARCH-MOCK: the live half of the fake's contract is asserted
      but absent).
  - id: new
    severity: Important
    family: enforcement-not-pinned-by-a-test
    title: |
      Nothing in go test enforces the captures' required shapes, and the script that does cannot run offline
    detail: |
      internal/llm/llmtest/testdata/README.md tabulates each capture's required shape
      as "enforced by llm-probe.sh verify", but nothing runs that script and it cannot
      run without the proxy config: K="$(key)" executes unconditionally at
      scripts/llm-probe.sh:20, so CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify
      dies on FileNotFoundError even though verify needs neither key nor network.
      Given fake-models-unobserved-shape produced three plan-gate findings, the
      durable fix is a Go test; llmtest already embeds every capture.
  - id: new
    severity: Minor
    family: stale-module-metadata
    title: |
      go mod tidy was never run; the directly-imported SDK is still marked indirect
    detail: |
      go.mod:11 lists github.com/anthropics/anthropic-sdk-go as // indirect although
      internal/llm/anthropic.go imports it. Plan Task 1 Step 1 said tidy runs at the
      end of Task 5. Verified: tidy promotes it to the direct require block and
      changes nothing else.
  - id: new
    severity: Minor
    family: test-panics-instead-of-failing
    title: |
      TestThinkingSignatureIsPreserved discards the error then indexes Blocks[0]
    detail: |
      internal/llm/anthropic_test.go:90 does got, _ := ...Complete(...) and then
      got.Blocks[0] at :91. When the call errors this panics with index out of range,
      aborting the whole package run and masking every other result. Observed while
      mutation-testing. Check the error.
  - id: new
    severity: Minor
    family: dead-code
    title: |
      var _ = http.StatusOK props up an otherwise-unused net/http import
    detail: |
      internal/llm/anthropic_test.go:381. Drop the line and the import.
  - id: new
    severity: Minor
    family: pure-logic-on-an-io-type
    title: |
      response() has an unused receiver, so the package's only transformation logic is reachable only through httptest
    detail: |
      internal/llm/anthropic.go:185 never references a. Promoting it to a package-level
      function would let block assembly and Text-joining be unit-tested directly from a
      capture via json.Unmarshal, with no server (ARCH-PURE).
  - id: new
    severity: Minor
    family: fake-silently-ignores-inputs
    title: |
      serveStream ignores Reply.Text and Reply.Stop entirely
    detail: |
      internal/llm/llmtest/fake.go:305. f.Script("x", Reply{Text: "…", Stop: "refusal"})
      silently does nothing on a streaming request — it always replays the capture.
      Either honour them or fail loudly on the combination.
  - id: new
    severity: Minor
    family: test-helper-fatal-off-goroutine
    title: |
      Capture calls t.Fatalf from the fake's server goroutine
    detail: |
      internal/llm/llmtest/fake.go:311. A missing capture, or a request arriving after
      cleanup, becomes a hang or a "log after test completed" panic rather than a clean
      failure. Resolve captures at Script/ServeRecorded time instead.
  - id: new
    severity: Minor
    family: comment-references-future-surface
    title: |
      config.go references --llm-check, which does not exist until M2
    detail: |
      internal/llm/config.go:99. Also io := []byte(...) shadows the io package at
      internal/llm/anthropic_test.go:349.
```
