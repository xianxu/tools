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

---

## Re-review — 2026-08-22T19:45:37-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b5d50ea29e4041e5681696fd47bc4e973c8f6c10^..a9fc896c18bc95a1fc6a1107514fac39ae269529 |
| command | sdlc milestone-close --issue 11 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-22T19:45:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I have everything I need. Verdict and findings:

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Criticals from round 1 are genuinely fixed and — unusually — genuinely pinned: I reverted each in the working tree and confirmed the test goes red (BR-1 → `err = llm: unavailable…, want ErrTruncated`; BR-2 → `no deltas were delivered; this test is not exercising the after-text case`), then restored byte-exactly. All four Importants are disposed `addressed` with independent verification: `go mod tidy` now produces no diff, `CLIPROXY_CONFIG=/nonexistent scripts/llm-probe.sh verify` exits 0, `conformance_test.go` exists/vets/skips cleanly, and every plausible typo model (`claude-opus-6`, `claude-opus-5-typo`, `claude-sonnet-9`) now gets 502. Full suite green including `-race`. What keeps this from a clean SHIP is one new correctness gap in the milestone's own deliverable — an exhausted 408 lands in `ErrRequest`, the loud "our bug" arm, despite being in the SDK's retry set — plus two family repeats the escalation protocol says to answer with a rule rather than an instance: three of this round's fixes changed behavior that no test would catch regressing, and four doc claims now outrun the tree.

## 1. Strengths

- **The two Critical fixes are pinned, not asserted.** `internal/llm/anthropic.go:159-177` routes a stall through the same `sawEvent` discriminator as any other mid-stream failure, and `TestStreamStallAfterTextSalvagesAndTruncates` (anthropic_test.go:307) fails without it. The test even guards its own premise (`if seen.Len() == 0 { t.Fatal("this test is not exercising the after-text case") }`), which is what caught the C2 reversion — a rare and correct instinct.
- **I1 was answered at the class, not the instance.** `knownModels` (fake.go:148) is an exact measured set, not the fixture patch the finding could have been read as asking for. Verified positively: three unrelated typos all get 502.
- **I4 became a Go test rather than a better script.** `captures_test.go` runs offline on every `go test`, and `llm-probe.sh`'s key read is now lazy (`need_key`, scripts/llm-probe.sh:24) so `verify` runs without the proxy config. That is the durable form of the fix.
- **The plan's Robustness bar was corrected in place where the code contradicted it** (plan §"Does: detect a stalled stream"), rather than leaving prose that the implementation had already diverged from.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `internal/llm/errors.go:44-52`: an exhausted 408 is classified as `ErrRequest`, the arm a caller must not absorb.**

`classifyStatus` routes 429 and ≥500 to `ErrUnavailable` and everything else to `ErrRequest`. 408 Request Timeout and 409 Conflict fall to `default`. Measured end-to-end through the real client against the fake:

```
status 408 -> err=llm: bad request: … 408 Request Timeout | ErrUnavailable=false ErrRequest=true (requests=3)
status 409 -> err=llm: bad request: … 409 Conflict         | ErrUnavailable=false ErrRequest=true (requests=3)
```

The `requests=3` is the point: the SDK retried it twice, because 408 is in its retry set — `anthropic.go:18-19` says so in the doc comment. So the transport treats 408 as transient and then the taxonomy calls it our bad schema/model/body. A consumer degrading on `ErrUnavailable` and staying loud on `ErrRequest` will surface "the proxy timed out" as a defect report. `errors_test.go:14-25` covers neither status, so nothing catches it. Fix: add `status == http.StatusRequestTimeout` (and decide 409 deliberately) to the `ErrUnavailable` arm, and add both rows to the table.

**I2 — three of this round's fixes are not pinned by any test.** *This is the 2nd finding in family `enforcement-not-pinned-by-a-test`.* Earlier rounds fixed instances. Do NOT fix these three instances — state the rule that covers all of them and fix that. The rule: **a boundary fix that changes behavior is not complete until a test fails against the pre-fix code, and the fixture must not encode the pre-fix implementation's escape hatch.** Measured prevalence 3, all verified this round:

- BR-3 (`fake.go:148`): reverting `knownModel` to the prefix + `"not-a-real"` substring rule leaves `go test ./internal/llm/...` fully green, because `TestUnknownModelIsRejectedLikeTheProxyDoes` still uses the fixture `claude-not-a-real-model`, which carries the magic substring. The rule can silently regress.
- BR-4 (`anthropic.go:39`): deleting `c.StallAfter = orDuration(...)` from `New` leaves the suite green. Every stall test sets `StallAfter` explicitly, so no fixture depends on the default.
- BR-11 (`fake.go:342`): no test in the tree scripts `Reply{Text}` against a streaming request, so the loud branch is entered by nothing. (I confirmed it *works* with a scratch test — it is reachable, just unexercised.)

**I3 — four doc claims now outrun the tree.** *This is the 2nd finding in family `docs-claim-absent-surface`.* Do NOT fix these four sites individually — the rule is: **a claim in a doc comment or atlas page is a claim about the tree, and this round's sweep must check each one against the tree rather than against the intent of the change that wrote it.** Measured prevalence 4:

1. `anthropic.go:203-204` — "It is now unit-tested directly from a committed capture with `json.Unmarshal` and no server at all." `grep -rn buildResponse --include="*.go"` returns only `anthropic.go`; it is unexported and `anthropic_test.go` is `package llm_test`, so no such test exists or can exist there. The receiver removal (BR-10) is real; the test is not.
2. `config.go:43` — "StallAfter bounds SILENCE inside a stream. **Zero disables it.**" `New` (the only constructor) now overrides zero with 90s, so the documented way to disable stall detection is unreachable. This was introduced *by* the BR-4 fix. The plan carries the same sentence at `workshop/plans/000011-vocab-llm-plan.md:837`.
3. `llm.go:130-136` — `Progress.Phase` documents `"connect" | "waiting" | "streaming" | "done"`; only `"waiting"` and `"streaming"` are ever emitted (`anthropic.go:82,116`). `Progress.Bytes` is never assigned anywhere in the tree.
4. `fake.go:104` — `Reply.Capture` "committed capture served verbatim; **wins over the rest**". `serve` checks `reply.Status != 0` at fake.go:256 before reaching `serveJSON`, so `Status` wins over `Capture`.

## 4. Minor findings

- `fake.go:343` — *2nd finding in family `test-helper-fatal-off-goroutine`.* BR-11's fix added `f.t.Fatalf` on the server goroutine, ten lines above the comment "Same reason as serveJSON: no Fatalf off the test goroutine" that BR-12's fix wrote. It happens to record the failure today; a request arriving after cleanup still panics with "log after test completed". State the rule (no `t.Fatalf`/`FailNow` from a handler goroutine — resolve at `Script` time or answer with a 500) rather than patching this site.
- `llm.go:110`, `errors.go:54` — *2nd finding in family `comment-references-future-surface`.* BR-13 removed the `--llm-check` reference from `config.go:99` and left two siblings in the same package. Instance fixed, class not (ARCH-PURPOSE).
- `captures_test.go:29-31,36-40` — *2nd finding in family `test-panics-instead-of-failing`.* `m["content"].([]any)`, `c.(map[string]any)["type"].(string)` and `cm["text"].(string)` are unchecked; a capture missing `content` panics and aborts the package run rather than failing. Introduced by the fix round closing BR-8.
- `anthropic.go:294-306` + `config.go:96` — `orInt64`, `orDuration` and `orDefault` re-implement stdlib `cmp.Or[T comparable](vals ...T) T`, available since Go 1.22 (module targets 1.26). One import replaces three helpers (ARCH-DRY).
- Comment archaeology is growing in production code. The fix commit added 110 comment lines against 230 code lines, much of it narrating prior review rounds (`anthropic.go:32-37`, `:164-169`; `fake.go:117-121`, `:142-147`) — history that already lives in the gate ledger and the issue Log. `llm.go` is 53% comment, `errors.go` 48%. Round 1 flagged density as "worth watching"; it went up.

## 5. Test coverage notes

- **Verified green under mutation:** BR-1 and BR-2 both confirmed red-on-revert. Full suite `go test ./...` passes in 8.6s; `-race` clean on both llm packages.
- **`OnSlow` / `Progress` has zero tests.** `grep -rn "OnSlow\|Progress" --include="*_test.go"` returns nothing. The plan's Task 5 explicitly lists `func TestOnSlowNamesThePhase(t *testing.T) { /* waiting vs streaming */ }` (plan:1383) and it was not delivered. A test would have caught the phase-vocabulary and `Bytes` gaps in I3.3.
- **`Response.StopDetails` is never asserted.** `TestRefusalBecomesErrRefused` checks the error but not the field PQ-9 added specifically so `ErrRefused` could say *why*. The fake's scripted refusal emits no `stop_details`, so the populated case is unexercised on both sides.
- **The 408/409 rows are absent from the `classifyStatus` table** (I1) — the one gap where missing coverage maps directly to a shipped misclassification.

## 6. Architectural notes

- **ARCH-DRY — pass, one flag.** `buildResponse` and `watch` remain shared between `Complete` and `Stream`, so the two paths cannot fork on `Text` assembly or phase vocabulary. Flagged: the three `or*` helpers duplicate `cmp.Or` (Minor above).
- **ARCH-PURE — pass, one flag.** `buildResponse` is now a package-level pure function with no receiver, which was the substance of BR-10, and `Resolve` remains pure over the lookup with `TestResolveIsPureOverTheLookup` proving it. Flagged: the promotion did not come with the direct capture-driven unit test its own doc comment claims (I3.1) — the logic is *structurally* pure and still only reachable through httptest.
- **ARCH-PURPOSE — pass, one flag.** The round mostly answered classes rather than instances: I1 became an exact model set rather than a fixture edit, and I4 became a Go test rather than a script tweak. Flagged: BR-13 fixed the one site the finding named and left two enumerable siblings — the instance, not the class.
- **ARCH-MOCK — pass, two flags.** The fake stays on the wire, models the *measured* 502 rather than the direct API's 400, and the live half of its contract now exists as a file that vets under the tag and skips cleanly unconfigured. Flagged: the fake's exact-model-set invariant regresses silently (I2), and the live conformance run could not be verified here (no key on this machine — it skipped, which is the correct shape).

**For M2:** `Run[T]` must call `classifyStop` before `decode`, and the test for it should feed `message-truncated.json` and fail against a decode-first implementation. Fix I1 before five consumers branch on the taxonomy — a 408 misclassification is much cheaper now than after `#12`/`#13` build degradation logic on it. And `renderRequest` must marshal the schema `map[string]any` with sorted keys or every cassette will miss at random.

## 7. Plan revision recommendations

1. **Append a `## Revisions` entry for the M1 boundary-review round.** The plan has four dated entries, none for the BR round; the C1 correction was made in place with an inline note. AGENTS.md §1 requires an appended entry (timestamp + reason + delta), not an in-place rewrite, so the record of *what the code did before the review* survives.
2. **Correct `Zero disables it` at plan:837** to match whatever `New` is decided to do (I3.2) — the plan and `config.go` currently agree with each other and disagree with the code.
3. **Retract or deliver `TestOnSlowNamesThePhase` (plan:1383)**, and reconcile the plan's `phases connect|waiting|streaming|done` claim (plan:346) with the two phases the code emits.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified — reverting the stall branch to Response{}+ErrUnavailable turns TestStreamStallAfterTextSalvagesAndTruncates red.
  - id: BR-2
    disposition: addressed
    note: |
      Mutation-verified — re-gating Stall on content_block_delta turns the same test red at "no deltas were delivered".
  - id: BR-3
    disposition: addressed
    note: |
      Behavior verified: claude-opus-6 / -typo / -sonnet-9 all get 502. Not pinned by a test — rolled into the enforcement-not-pinned-by-a-test repeat.
  - id: BR-4
    disposition: addressed
    note: |
      New now defaults all six fields. Side effects (unpinned; "Zero disables it" now unreachable) raised as new findings.
  - id: BR-5
    disposition: addressed
    note: |
      conformance_test.go exists, vets under -tags conformance, and skips cleanly with no key configured.
  - id: BR-6
    disposition: addressed
    note: |
      captures_test.go runs offline in go test; CLIPROXY_CONFIG=/nonexistent llm-probe.sh verify exits 0.
  - id: BR-7
    disposition: addressed
    note: |
      SDK is in the direct require block; go mod tidy produces no diff.
  - id: BR-8
    disposition: addressed
    note: |
      Both tests now check the error and length-check before indexing Blocks.
  - id: BR-9
    disposition: addressed
    note: |
      var _ = http.StatusOK and the net/http import are gone from anthropic_test.go.
  - id: BR-10
    disposition: addressed
    note: |
      buildResponse is package-level with no receiver. The doc comment's claim of a direct unit test is false — rolled into the docs-claim repeat.
  - id: BR-11
    disposition: addressed
    note: |
      serveStream fails loudly; verified reachable with a scratch test. No fixture enters it — rolled into the enforcement repeat.
  - id: BR-12
    disposition: addressed
    note: |
      Both capture-miss paths return 500 instead of Fatalf. A new off-goroutine Fatalf was added by BR-11's fix — raised separately.
  - id: BR-13
    disposition: addressed
    note: |
      config.go:99 cleared; llm.go:110 and errors.go:54 still carry the reference — raised as the family repeat.
findings:
  - id: new
    severity: Important
    family: unclassified-failure-mode
    title: |
      An exhausted 408 classifies as ErrRequest, the loud "our bug" arm, though the SDK retries it as transient
    detail: |
      internal/llm/errors.go:44-52 routes 429 and >=500 to ErrUnavailable and everything
      else to default/ErrRequest, so 408 Request Timeout and 409 Conflict land loud.
      Measured end-to-end through the real client against the fake: status 408 ->
      "llm: bad request", ErrUnavailable=false, after 3 attempts — the SDK retried it,
      which is the transport itself calling it transient, and then the taxonomy calls it
      our bad schema/model/body. errors_test.go covers neither status. Add
      http.StatusRequestTimeout to the ErrUnavailable arm, decide 409 deliberately, and
      add both rows to the table.
  - id: new
    severity: Important
    family: enforcement-not-pinned-by-a-test
    title: |
      Three of this round's fixes survive full reversion with a green suite, or are entered by no fixture
    detail: |
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
  - id: new
    severity: Important
    family: docs-claim-absent-surface
    title: |
      Four doc claims outrun the tree, two of them written by this round's own fixes
    detail: |
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
  - id: new
    severity: Minor
    family: test-helper-fatal-off-goroutine
    title: |
      BR-11's fix added a t.Fatalf on the fake's server goroutine, ten lines above the comment forbidding it
    detail: |
      This is the 2nd finding in family test-helper-fatal-off-goroutine. Do NOT fix this
      instance — state the rule (no t.Fatalf/FailNow from a handler goroutine; resolve at
      Script time or answer with a 500) and fix that. fake.go:343 calls f.t.Fatalf inside
      serveStream while fake.go:352 says "Same reason as serveJSON: no Fatalf off the test
      goroutine". Verified: it does record the failure today, but a request arriving after
      cleanup still panics with "log after test completed".
  - id: new
    severity: Minor
    family: comment-references-future-surface
    title: |
      --llm-check was removed from config.go and left in two sibling files
    detail: |
      This is the 2nd finding in family comment-references-future-surface. Do NOT fix this
      instance — state the rule covering forward references to unbuilt surface and sweep
      the enumeration. Remaining: llm.go:110 and errors.go:54. Measured prevalence 2 after
      the round that closed the one site the finding named (ARCH-PURPOSE: instance, not
      class).
  - id: new
    severity: Minor
    family: test-panics-instead-of-failing
    title: |
      captures_test.go indexes into unchecked type assertions, so a malformed capture panics rather than fails
    detail: |
      This is the 2nd finding in family test-panics-instead-of-failing. Do NOT fix this
      instance — state the rule and sweep. internal/llm/llmtest/captures_test.go:29-31 does
      m["content"].([]any) and c.(map[string]any)["type"].(string), and :40 does
      cm["text"].(string). A capture missing content aborts the whole package run. This
      instance was introduced by the fix round that closed BR-8.
  - id: new
    severity: Minor
    family: stdlib-reimplemented
    title: |
      orDefault, orInt64 and orDuration re-implement stdlib cmp.Or
    detail: |
      anthropic.go:294-306 and config.go:96. cmp.Or[T comparable](vals ...T) T has been in
      the standard library since Go 1.22 and the module targets 1.26, so one import
      replaces three near-identical helpers (ARCH-DRY).
  - id: new
    severity: Minor
    family: review-archaeology-in-code
    title: |
      Production comments increasingly narrate prior review rounds, duplicating the gate ledger
    detail: |
      The fix commit added 110 comment lines against 230 code lines. Examples:
      anthropic.go:32-37 and :164-169, fake.go:117-121 and :142-147 all recount what "the
      first version" did. That history already lives in the gate ledger and the issue Log,
      and it means a behavior change now needs three edits to stay honest. llm.go is 53%
      comment, errors.go 48%. Round 1 flagged density as worth watching; it rose.
```

---

## Re-review — 2026-08-22T20:04:10-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b5d50ea29e4041e5681696fd47bc4e973c8f6c10^..97a9b674e681f1ae22caacde42acfffdb0149389 |
| command | sdlc milestone-close --issue 11 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-22T20:04:10-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's two hardest findings were genuinely closed at the class, and I verified that by reverting rather than by reading: BR-14's fix derives transience from the SDK's own retry policy instead of adding two cases (reverting reddens the 408/409 rows), and BR-15's three unpinned fixes are now each pinned by a test that fails against pre-fix code — I confirmed `TestNewDefaultsEveryConfigField` reddens on *two different* deleted defaults, not just the one the finding named. Full suite green, `-race` clean, `go mod tidy` a no-op, `-tags conformance` vets, `llm-probe.sh verify` runs offline at exit 0. What keeps this from SHIP is two Importants, neither a shipped defect: BR-16's docs sweep is **not-addressed** — three claims it was scoped to fix are still false in the tree, two of them cited by line number *inside the finding itself* — and a measured class of load-bearing behaviour that no fixture distinguishes. `Request.System` can be deleted from the wire and the entire suite stays green; so can the multi-text-block join that `Response.Text`'s own comment calls "the thing most likely to drift."

## 1. Strengths

- **BR-14 was fixed structurally, not additively.** `internal/llm/errors.go:42-51` derives the transient set from the transport's retry policy rather than keeping a second list — which is the actual cause of the contradiction, not the two missing cases. Mutation-verified: reverting reddens `classifyStatus(408)` and `(409)`.
- **BR-15's rule was applied to the class.** `internal/llm/response_test.go:114` asserts on the *constructed config* in-package, so it fails when **any** default is dropped. I deleted `StallAfter` and then `Effort` — both red, with the field named in the message. That is the class, not the field.
- **The unknown-model fixtures no longer carry the escape hatch.** `internal/llm/llmtest/fake_test.go:135-139` uses three ordinary typos. Reverting `knownModel` to the prefix rule now reddens both `TestUnknownModelIsRejectedLikeTheProxyDoes` *and* `llmtest.Suite`'s unknown-model obligation — the exact regression BR-15a said was silent.
- **`buildResponse`'s doc comment finally describes the tree.** `response_test.go` is `package llm`, reads the capture with `os.ReadFile`, and calls `buildResponse` directly — no server. The claim that was false in round 2 is true now, and four tests rest on it.
- **BR-18 was swept to zero.** `grep -- "--llm-check"` over `internal/ cmd/ atlas/ scripts/` returns nothing. Comment density also genuinely fell (`llm.go` 53%→48%, `errors.go` 48%→39%), and only three archaeology sites survive, all of them rationale for the current shape rather than narration of a prior round.

## 2. Critical findings

None.

## 3. Important findings

**I1 — BR-16's sweep is incomplete; three claims it covered are still false, two named by line in the finding.** *Re-raised as `not-addressed`, not as a new id.*

1. `internal/llm/llmtest/fake.go:107` — `Capture string // committed capture served verbatim; wins over the rest`. Measured with a scratch test: `Reply{Capture: "message-thinking.json", Status: 429}` serves **429**, because `serve` checks `reply.Status != 0` at fake.go:249 before `serveJSON` ever reads `Capture`. This is verbatim the sub-item BR-16 listed as (4).
2. `workshop/plans/000011-vocab-llm-plan.md:837` — `// seconds. Zero disables it.` BR-16 said "the plan repeats the sentence at plan:837." `config.go` was corrected; the plan was not.
3. `workshop/plans/000011-vocab-llm-plan.md:345-346` — `Progress{Phase, Elapsed, Bytes}` … phases `connect|waiting|streaming|done`. `llm.go` was corrected to the two phases the tree emits and `Bytes` was removed; the plan still claims four phases and a field that no longer exists.

The rule BR-16 stated is right and stays the deliverable: a doc claim is a claim about the tree, checked against the tree. What it did not get is the **enumeration** — the sweep ran over `internal/llm/*.go` and stopped there, so the plan (which the finding explicitly reached into) and one struct-field comment survived. ARCH-PURPOSE: the instances that were easy to see, not the class.

**I2 — Four load-bearing behaviours are entered by no fixture; the suite stays green when each is removed.** *This is the 2nd finding in family `fake-cannot-reach-the-branch`.* Earlier rounds fixed instances (BR-2: the fake stalled on a thinking delta, so stall-after-text was unreachable). Do NOT fix these four instances — state the rule and sweep the enumeration it implies. The rule: **a behaviour the code singles out as load-bearing needs a fixture that separates it from the alternative it warns against; where no committed capture exhibits that shape, the fixture is constructed, because block *count* and header *presence* are transport shape rather than judgment.** Measured prevalence 4, three verified by reversion:

- `internal/llm/anthropic.go:218-228` — "Text is every text block JOINED … not `Content[0].Text`". All three captures carry exactly **one** text block, so `if text.Len() == 0 { … }` (first-block-only) leaves `go test ./internal/llm/...` fully green. This is the property the atlas, the package doc, and `Response.Text`'s comment each single out.
- `internal/llm/anthropic.go:62-64` — deleting the `r.System` → `p.System` plumbing leaves the suite green. `Recorded.System()` exists at fake.go:59 for exactly this assertion and has **zero callers**. M2's `Task[T]` sets `System` on every call; a silently dropped system prompt looks identical to a working one — the same shape as round 1's "the credential never reaches an assertion," which was fixed by asserting `x-api-key` on the wire.
- `internal/llm/anthropic.go:262-284` — neutering `watch()` so `OnSlow` never fires leaves the suite green. `grep -rn "OnSlow\|Progress" --include='*_test.go'` returns nothing. The plan lists `TestOnSlowNamesThePhase` at plan:1383 and it was not delivered; round 2 asked for it to be retracted or delivered and neither happened. A test here would have caught I1.3 on its own.
- `Response.StopDetails` — zero test references. The field exists so `ErrRefused` can say *why*, and the fake's scripted refusal (`Reply{Stop: "refusal"}`) emits no `stop_details`, so the populated branch is unreachable from any fixture on either side of the seam.

## 4. Minor findings

- `internal/llm/errors_test.go:91` and `internal/llm/llmtest/captures_test.go:130` — *2nd finding in family `stdlib-reimplemented`.* Two hand-rolled `contains` helpers survived the round that replaced `orDefault`/`orInt64`/`orDuration` with `cmp.Or`: one re-implements `strings.Contains`, the other `slices.Contains`. State the rule (a helper that duplicates a stdlib function is removed, and the sweep covers `_test.go` too) rather than patching these two.
- `internal/llm/llmtest/fake.go:174` — *2nd finding in family `dead-code`.* `Fake.t` is assigned in `NewFake` and read by nothing; BR-17's fix removed its last consumer. The rule: when a fix removes the last reader of a field, import, or helper, the same edit removes the field.
- `internal/llm/llmtest/fake.go:233` vs `:243` — *2nd finding in family `fake-silently-ignores-inputs`.* `f.next()` pops the matcher queue **before** the unknown-model check returns 502, so a scripted reply is consumed and discarded on that path. Harmless today (no test scripts a sequence against a bad model); a trap for M2's cassette sequences.
- The plan was edited in place at the M1 boundary (`*Corrected in place, M1 boundary review (C1)*` in the Robustness bar; measured fact 2 likewise) with no appended `## Revisions` entry for the boundary-review round. AGENTS.md §1: append, don't overwrite. Round 2 recommended this in section 7, where it was advisory and therefore never tracked or disposed.

## 5. Test coverage notes

Verified green under reversion this round: BR-14 (both statuses), BR-15a (`knownModel`, reddens two tests), BR-15b (two independent defaults), BR-15c (the loud unstreamable-`Reply` branch). Full suite 9.5s, `-race` clean, `go vet` clean including `-tags conformance`.

The gap is I2's enumeration, and it is worth stating as one number: **three of the transport's own named invariants can be deleted outright without a single test failing.** Beyond those, `Reply.NoThinking` and `Reply.Body` are fake features with zero callers, and `Usage.InputTokens`/`CacheCreationTokens` are never asserted individually (only via `PreambleTokens()`, which sums them — so a swap of the two fields is invisible). The right shape for the fix is one hand-built multi-block fixture plus a `Recorded.System()` assertion in `TestCompleteRoundTrip`, not four separate tests.

## 6. Architectural notes

- **ARCH-DRY — pass, one flag.** `transient()` at errors.go:42 is a genuine win: one answer to "is this worth retrying," derived from the transport rather than restated beside it, and the comment says why. `buildResponse`/`watch` stay shared across both paths. Flagged: the two hand-rolled `contains` helpers (Minor).
- **ARCH-PURE — pass.** `buildResponse` is package-level, side-effect free, and now genuinely tested with no server; `Resolve` remains pure over the lookup with `TestResolveIsPureOverTheLookup` proving it against a real `t.Setenv`. Nothing in the diff buries logic in IO.
- **ARCH-PURPOSE — flag.** The round answered BR-14 and BR-15 at the class, which is the right instinct and visibly better than round 1. BR-16 did not: the sweep covered the package's `.go` files and stopped, leaving two sites the finding had cited by line number. Naming a class and then enumerating only the part of it that is convenient to grep is the same failure one level up.
- **ARCH-MOCK — flag.** Placement stays exemplary — the double is on the wire, the fake models the *measured* 502 rather than the direct API's 400, and the live half exists, vets under the tag, and skips cleanly unconfigured (I could not exercise it: no key on this machine, which is the correct shape). Flagged: the fake's fixture set does not exhibit the multi-text-block shape the transport's own contract is written around, and the one accessor built to assert `System` on the wire has no caller — so two things the seam exists to protect are protected by nothing.

**For M2:** `Run[T]` must call `classifyStop` before `decode`, with `message-truncated.json` as the fixture and a test that fails against a decode-first implementation. `renderRequest` must marshal the schema `map[string]any` with sorted keys or cassettes miss at random. And `--llm-check` is the first production caller of `Resolve`/`Redact` — both are test-only today, so the whole config path lands unexercised by production flow until then.

## 7. Plan revision recommendations

1. **Append a `## Revisions` entry for the M1 boundary-review rounds.** The plan has four dated entries, none for BR-1…BR-21; the C1 correction and measured fact 2 were both rewritten in place. AGENTS.md §1 requires an appended entry (timestamp + reason + delta) so the pre-review claim survives.
2. **Correct plan:837** (`Zero disables it`) and **plan:345-346** (`Progress{Phase, Elapsed, Bytes}`, `connect|waiting|streaming|done`) to match the tree. Both were named by BR-16 and by round 2's section 7.
3. **Retract or deliver `TestOnSlowNamesThePhase` (plan:1383).** Asked for last round; still neither.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      Fixed at the cause — transient() derives from the SDK retry policy; reverting reddens the 408 and 409 rows.
  - id: BR-15
    disposition: addressed
    note: |
      All three reversions now red; TestNewDefaultsEveryConfigField verified against two different deleted defaults, not just StallAfter.
  - id: BR-16
    disposition: not-addressed
    note: |
      Three claims still false: fake.go:107 "wins over the rest" (Status wins, measured), plan:837, plan:345-346 — two cited by line in the finding.
  - id: BR-17
    disposition: addressed
    note: |
      No t.Fatalf remains on any handler goroutine; the rule is stated in serveStream and both capture-miss paths answer with a status.
  - id: BR-18
    disposition: addressed
    note: |
      Swept to zero — no --llm-check reference anywhere in internal/, cmd/, atlas/ or scripts/.
  - id: BR-19
    disposition: addressed
    note: |
      captures_test.go checks every assertion and t.Fatalf's; the rule is stated at the top of the helper.
  - id: BR-20
    disposition: addressed
    note: |
      cmp.Or replaces all three helpers. Two hand-rolled contains helpers survived the sweep — raised as the family repeat.
  - id: BR-21
    disposition: addressed
    note: |
      Density fell (llm.go 53->48%, errors.go 48->39%); only three archaeology sites remain and all three are rationale.
findings:
  - id: new
    severity: Important
    family: fake-cannot-reach-the-branch
    title: |
      Three of the transport's own named invariants can be deleted outright with the whole suite still green
    detail: |
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
  - id: new
    severity: Minor
    family: stdlib-reimplemented
    title: |
      Two hand-rolled contains helpers survived the round that replaced the or* helpers with cmp.Or
    detail: |
      This is the 2nd finding in family stdlib-reimplemented. Do NOT fix these two
      instances — state the rule (a helper duplicating a stdlib function is removed,
      and the sweep covers _test.go as well as production files) and fix that.
      Measured prevalence 2. errors_test.go:91 re-implements strings.Contains with an
      index loop; captures_test.go:130 re-implements slices.Contains. Both are in the
      same two packages as the cmp.Or fix and both were skipped because the sweep
      scoped itself to production code.
  - id: new
    severity: Minor
    family: dead-code
    title: |
      Fake.t is assigned in NewFake and read by nothing after BR-17 removed its last consumer
    detail: |
      This is the 2nd finding in family dead-code. Do NOT fix this instance — the rule
      is: when a fix removes the last reader of a field, import or helper, the same
      edit removes the field. internal/llm/llmtest/fake.go:174 declares t *testing.T
      and fake.go:184 assigns it; grep for f.t across the package returns nothing now
      that serveStream answers with a status instead of calling Fatalf. go vet does
      not flag unused struct fields, so nothing else will catch this class.
  - id: new
    severity: Minor
    family: fake-silently-ignores-inputs
    title: |
      An unknown-model request pops the matcher queue and then discards the reply it drew
    detail: |
      This is the 2nd finding in family fake-silently-ignores-inputs. Do NOT fix this
      instance — the rule is: the fake must not mutate scripted state on a path that
      does not serve the scripted reply. internal/llm/llmtest/fake.go:233 calls
      f.next(rec.Prompt()) before the unknown-model check at fake.go:243 returns 502,
      so a queued Reply is consumed and thrown away. Harmless today because no test
      scripts a sequence against a bad model; a trap for M2's cassette sequences,
      where the next call would silently draw the wrong queue entry.
  - id: new
    severity: Minor
    family: plan-revision-not-appended
    title: |
      The plan was corrected in place at the M1 boundary with no appended Revisions entry
    detail: |
      workshop/plans/000011-vocab-llm-plan.md carries four dated Revisions entries,
      none for the boundary-review rounds, while the Robustness bar was rewritten
      in place ("Corrected in place, M1 boundary review (C1)") and measured fact 2
      likewise. AGENTS.md section 1 requires appending a Revisions entry with
      timestamp, reason and delta rather than overwriting, so the pre-review claim
      survives. Round 2 recommended this in its advisory section, where it was never
      tracked as a finding and therefore never disposed.
```

---

## Re-review — 2026-08-22T20:26:51-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b5d50ea29e4041e5681696fd47bc4e973c8f6c10^..78dedc02fea1a5f50873473e3972524b4024ec4b |
| command | sdlc milestone-close --issue 11 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-22T20:26:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Every open finding from round 3 is genuinely closed, and I verified the hard one by reversion rather than by reading: all four BR-22 instances now go red when deleted (`TestTextJoinsEveryTextBlock`, `TestSystemPromptReachesTheWire`, `TestOnSlowReportsThePhase`, `TestRefusalCarriesItsReason` — six reversions run this round, six red). BR-16's cited sites are all corrected, `Fake.t` is gone, `next()` moved below the unknown-model check, both hand-rolled `contains` helpers are stdlib now, and the plan carries an appended `## Revisions` entry for the boundary rounds. Full suite green including `-race`, `go mod tidy` a no-op, `-tags conformance` vets, `llm-probe.sh verify` exits 0 offline. What keeps this from SHIP is one shipped latent panic — a negative `SlowEvery` with `OnSlow` set panics `non-positive interval for NewTicker` on a background goroutine, unrecoverable, in a package whose *sibling* duration field documents negative as "disabled" — plus a fake gap where the branch adjacent to the one this round made loud is still silent. Worth naming plainly: **every finding below is a family repeat.** The code is converging; the enumerations are not.

## 1. Strengths

- **The BR-22 fixtures are constructed where captures couldn't reach, and each one is load-bearing.** `Reply.SplitText` (fake.go:127) exists solely so "Text is every text block joined" can be told apart from "Text takes the first" — I deleted the join and `TestTextJoinsEveryTextBlock` went red. Same for the system-prompt plumbing, `watch()`, and the fake's refusal `stop_details`. That is the rule applied, not the instance patched.
- **`TestNewDefaultsEveryConfigField` (response_test.go:114) asserts on the constructed config, in-package.** It fails naming whichever default was dropped, so it covers the class rather than the field the finding happened to name — and its own comment explains why the behavioural version couldn't work.
- **`TestScriptedTextOnAStreamingRequestFailsLoudly` finally enters BR-15c's branch**, and reverting the guard reddens it. The choice of 400 over 500 (fake.go:388) is right and the comment says why: 5xx is in the SDK's retry set, so a retried complaint would draw the next queue entry and swallow itself.
- **`transient()` (errors.go:42) derives the retryable set from the transport's own policy** instead of restating it — the structural answer to BR-14, not two added cases.
- **`AGENTS.local.md:15-25` amends the rule in the same change that breaks it**, and bounds the carve-out ("If a would-be `internal/` package could be described without naming an external service, the first-consumer rule still applies"). That is how an exception stays an exception.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `internal/llm/anthropic.go:270`: a negative `SlowEvery` panics on a goroutine the caller cannot recover from.** *2nd in family `partial-constructor-defaults`.*

Measured:

```
panic: non-positive interval for NewTicker
time.NewTicker(0x0?)
github.com/xianxu/tools/internal/llm.(*anthropicClient).watch.func2()
	/Users/xianxu/workspace/tools/internal/llm/anthropic.go:270
```

from `llm.New(llm.Config{BaseURL: …, APIKey: …, SlowEvery: -1, OnSlow: f})`. It fires on the watcher goroutine, so no `recover` at the call site can catch it — the process dies.

This is invited by the package's own API. `config.go:52-55` documents `StallAfter` as *"disabling requires a NEGATIVE value"*; `config.go:59` documents `SlowEvery` only as *"Zero takes the default"*. Two duration fields on one struct, opposite conventions, one of them fatal.

Do NOT fix this site. The rule BR-4 needed and never got: **`New` must give every `Config` field a defined meaning for every value it can hold, and sibling fields must not disagree about what a sentinel means.** Measured prevalence 3, all three verified:

| field | negative value does | documented? |
|---|---|---|
| `StallAfter` | disables the bound | yes |
| `SlowEvery` | **panics** off-goroutine | no |
| `Timeout` | every call returns `ErrUnavailable: context deadline exceeded` instantly — silently absorbed, not a config error | no |

`TestNewDefaultsEveryConfigField` cannot catch any of these: it asserts non-zero, and `-1` is non-zero. The sweep is the four numeric `Config` fields, and the fix is one rule applied in `New` (clamp or reject), not a guard at line 270.

**I2 — `internal/llm/llmtest/fake.go:403`: a `.json` capture scripted against a streaming request is silently replaced by a different capture.** *3rd in family `fake-silently-ignores-inputs`.*

```go
name := reply.Capture
if name == "" || !strings.HasSuffix(name, ".sse") {
    name = "stream-sample.sse"
}
```

Measured — `f.Script("x", Reply{Capture: "message-truncated.json"})` then `Stream`:

```
err  = <nil>
Text = "**Obsequious.**\n\nThe clue after the dash — public praise for"
Stop = "end_turn"   blocks=2
```

The test asked for the `max_tokens` truncation specimen and got the healthy `end_turn` stream, with no error. A test written to pin truncation-during-stream would pass while asserting nothing about its own fixture — the same shape as `lessons.md`'s new "an assertion whose subject is empty cannot fail" entry, one level up.

This is the un-swept half of BR-11's own enumeration. BR-11's rule was *"either honour them or fail loudly on the combination"*; this round made `Reply{Text/Stop}` loud (fake.go:381) and left `Reply{Capture}` silent, ten lines above. Do NOT fix this site — state the rule (**every `Reply` field that cannot be served on the path a request took is a caller mistake and answers with a 400 naming the field; silence is never the alternative**) and sweep the `Reply` struct field by field. M2's cassette sequences run through this path.

## 4. Minor findings

- `internal/llm/anthropic_test.go:444` — *3rd in family `enforcement-not-pinned-by-a-test`.* `TestNegativeStallAfterDisablesTheBound` survives full reversion: I changed `New` to `if c.StallAfter <= 0 { c.StallAfter = defaultStallAfter }` and only `TestNewPreservesADisabledStallBound` went red. Its assertion (`elapsed >= 1s` against `Timeout: 2s`) cannot distinguish "stall disabled, deadline ends it at 2s" from "stall defaulted to 90s, deadline ends it at 2s" — the 2s bound dominates either way. Its comment claims otherwise. The rule the family never enumerated: **reversion-check the test you add, not only the fix it pins; a bound dominated by a second bound distinguishes nothing.** Prevalence 1 of the 6 tests added this round (the other 5 verified red).
- `internal/llm/llmtest/fake.go:162` — *2nd in family `fake-tuned-to-its-fixture`.* `splitInto` chops on byte offsets, so multibyte text is cut mid-rune and `json.Marshal` substitutes U+FFFD. Measured: `Reply{Text: "obséquieux — très flagorneur, vraiment", SplitText: 3}` round-trips as `"obséquieux ��� très flagorneur, vraiment"`. `TestTextJoinsEveryTextBlock` passes only because its fixture is ASCII, and every committed capture contains em-dashes. Introduced by this round's own BR-22.1 fix.
- `internal/llm/llmtest/fake.go:114,118` — *3rd in family `dead-code`.* `Reply.Body` and `Reply.NoThinking` have zero setters anywhere in the tree; `NoThinking` gates fake.go:339 and `Body` gates fake.go:289, so both are documented knobs no fixture turns. Round 3 named them in prose (§5) and never raised them, so they were never disposed. The rule needs widening from "a field whose last reader a fix removed" to "**the sweep enumerates the fake's exported surface, and a knob no fixture turns is dead the same as an unread field.**"
- `workshop/plans/000011-vocab-llm-plan.md:516-518` — *3rd in family `docs-claim-absent-surface`.* The Task 1 contract block still declares `Phase string // "connect" | "waiting" | "streaming" | "done"` and `Bytes int`, which the same document's Revisions entry at :2040 says are gone and the tree agrees are gone. Round 3's sweep covered the plan's prose (:837, :345-346, both now correct) and stopped at its embedded code blocks — which Task 1 Step 3 explicitly warns "get pasted verbatim."

## 5. Test coverage notes

Six reversions run this round, six results, all recorded above. Coverage of the transport's named invariants is now genuinely complete: text joining, system plumbing, `OnSlow`/`Progress`, `StopDetails`, the unknown-model set, the loud unstreamable-`Reply` branch, and both `New` defaulting behaviours all fail against pre-fix code.

Remaining gaps, in order of how likely they are to bite M2:

- **Out-of-range `Config` values are untested entirely** (I1). `TestNewDefaultsEveryConfigField` covers zero; nothing covers negative, and one of the three is a panic.
- **`Usage.InputTokens` and `CacheCreationTokens` are never asserted individually** — only via `PreambleTokens()`, which sums `CacheCreation + CacheRead`, so swapping the two field assignments in `buildResponse` is invisible to the suite. Round 3 noted this in prose; still open, still untracked.
- **The `.sse` capture path has one fixture.** `serveStream` accepts a `Capture` name but only `stream-sample.sse` exists, so I2's silent substitution has never had a second file to substitute *away from*.
- **Live conformance is unexercised here** — no key on this machine; it skipped, which is the correct shape.

## 6. Architectural notes

- **ARCH-DRY — pass.** `transient()` is one answer to "is this worth retrying," derived rather than restated. `buildResponse` and `watch` stay shared across both paths so `Text` assembly and the phase vocabulary cannot fork. `cmp.Or` replaced the three `or*` helpers and the two `contains` helpers are gone from `_test.go` as well. No new duplication in the diff.
- **ARCH-PURE — pass.** `buildResponse` is package-level, side-effect free, and now tested from a capture with `os.ReadFile` and no server — the claim its doc comment makes is true this round. `Resolve` stays pure over the lookup. `splitInto`, `knownModel`, `joinBlocks`, `classifyStop`, `classifyStatus`, `transient` are all pure; nothing in the diff buries logic in IO.
- **ARCH-PURPOSE — flag.** BR-22 was answered at the class with constructed fixtures, which is the right instinct and reversion-proven. But six findings this round, six family repeats, and three of them (I2, `dead-code`, `enforcement-not-pinned-by-a-test`) are the *adjacent half* of an enumeration a prior fix wrote too narrowly: `Reply{Text}` made loud and `Reply{Capture}` left silent; a field whose reader vanished removed while two fields no fixture ever sets stayed; the fix reversion-checked while the test added to pin it was not. The ledger is reporting that "name the class" is being read as "name the sites the finding listed."
- **ARCH-MOCK — flag.** Placement stays exemplary — the double is on the wire, stateful across calls, models the *measured* 502 rather than the direct API's 400, and the live half exists and vets under the tag. Two flags, both new: the fake silently serves a capture other than the one scripted (I2), and its one constructed-content helper corrupts non-ASCII (`splitInto`). A fake that answers a question different from the one asked is the failure mode the shared obligation suite exists to prevent, one layer down.

**For M2:** `Run[T]` must call `classifyStop` before `decode`, with `message-truncated.json` as the fixture and a test that fails against a decode-first implementation. `renderRequest` must marshal the schema `map[string]any` with sorted keys or cassettes miss at random. And `--llm-check` is the first *production* caller of `Resolve`/`Redact` — both are test-only today, so the whole config path lands unexercised by production flow until then, and I1's negative-`SlowEvery` panic sits on that path.

## 7. Plan revision recommendations

1. **Append a `## Revisions` entry recording that M1 over-delivered the M1/M2 split.** Three items the issue (`workshop/issues/000011-vocab-llm.md`) and the plan both assign to M2 shipped in M1: the atlas page (Task 14, landed at `4505160`), live conformance behind the build tag (Task 12, `internal/llm/conformance_test.go`), and the recorded SSE sample (Task 8, `testdata/stream-sample.sse`). Both artifacts now under-describe what M1 delivered, and M2's checklist claims work already done.
2. **Correct plan:516-518** — the `Progress` struct in Task 1 Step 2 still declares four phases and `Bytes`, contradicting the tree and the plan's own Revisions entry at :2040 (Minor finding above).
3. **Record the `SlowEvery` sentinel decision in the Robustness bar.** The bar specifies `StallAfter`'s semantics precisely and says nothing about `SlowEvery`'s, which is how the two fields came to disagree (I1). State what a negative value means for every duration field, so `New`'s clamp has a spec to satisfy.

```findings
dispose:
  - id: BR-16
    disposition: addressed
    note: |
      All four cited sites plus round 3's three re-cited sites verified correct against the tree; one uncited instance survives at plan:516 and is raised as the family repeat.
  - id: BR-22
    disposition: addressed
    note: |
      All four instances pinned by constructed fixtures; each verified red by reversion this round against go test ./internal/llm/...
  - id: BR-23
    disposition: addressed
    note: |
      Both hand-rolled contains helpers gone; errors_test.go:87 and captures_test.go:70,123 now use strings.Contains / slices.Contains.
  - id: BR-24
    disposition: addressed
    note: |
      Fake.t is gone from the struct and from NewFake; grep for a testing.T field in fake.go returns only the Capture and NewFake parameters.
  - id: BR-25
    disposition: addressed
    note: |
      f.next() now runs after the unknown-model check, with the rationale recorded at the call site.
  - id: BR-26
    disposition: addressed
    note: |
      A fifth Revisions entry ("M1 boundary review, three rounds") is appended at plan:1986 with the rules the rounds produced.
findings:
  - id: new
    severity: Important
    family: partial-constructor-defaults
    title: |
      A negative SlowEvery panics on the watcher goroutine, where no caller can recover
    detail: |
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
  - id: new
    severity: Important
    family: fake-silently-ignores-inputs
    title: |
      A .json capture scripted against a streaming request is silently replaced by stream-sample.sse
    detail: |
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
  - id: new
    severity: Minor
    family: enforcement-not-pinned-by-a-test
    title: |
      TestNegativeStallAfterDisablesTheBound survives full reversion of the behaviour it names
    detail: |
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
  - id: new
    severity: Minor
    family: fake-tuned-to-its-fixture
    title: |
      splitInto chops on byte offsets, so any multibyte scripted text round-trips corrupted
    detail: |
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
  - id: new
    severity: Minor
    family: docs-claim-absent-surface
    title: |
      The plan's Task 1 contract block still declares four Progress phases and a Bytes field
    detail: |
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
  - id: new
    severity: Minor
    family: dead-code
    title: |
      Reply.Body and Reply.NoThinking are documented knobs that no fixture in the tree turns
    detail: |
      This is the 3rd finding in family dead-code. Do NOT fix these two instances —
      the rule needs widening from "a field whose last reader a fix removed" to
      "the sweep enumerates the fake's exported surface, and a knob no fixture
      turns is dead the same as an unread field." internal/llm/llmtest/fake.go:114
      (Body) and :118 (NoThinking) each gate a production branch in the fake
      (fake.go:289 and :339) and neither is set anywhere in the tree, so both
      branches are reachable but unexercised — the shape BR-15c named. Round 3
      identified both in its prose section 5 and never raised them, so they were
      never tracked or disposed.
```
