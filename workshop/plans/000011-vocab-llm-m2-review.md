# Boundary Review — tools#11 (milestone M2)

| field | value |
|-------|-------|
| issue | 11 — LLM seam: Anthropic client, stateful fake, offline degradation |
| repo | tools |
| issue file | workshop/issues/000011-vocab-llm.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 85f6c9531b32c7798f2aaefe850f50a6ed0705fd..ae52feedd76ed13731566b2e8bbe2a912bb207b4 |
| command | sdlc milestone-close --issue 11 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-23T00:06:09-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

M2 delivers the typed-task layer with real craft — the single `renderRequest` feeding both the golden printer and the cassette key is exactly right and is *asserted* rather than trusted; the stop-reason-before-decode ordering is pinned by a genuine specimen (`message-truncated.json` really does decode to `{"verdict":"yes","reason":": Ā"}` with both fields populated); and I reverted three of the milestone's fixes and confirmed each goes red. What blocks SHIP is that the headline primitive does not hold the invariant it advertises: `decode` returns a **partially populated `T` with a nil error** when a schema-required field is missing, contradicting `task.go`'s own doc comment, `atlas/llm.md`, and plan Task 10 — and the fuzz target that claims to hold the invariant asserts only its error half, so nothing could have caught it. Alongside that, plan Task 9's schema golden was never written (so `AssertGolden` ships with zero committed artifacts and the DRY justification in `schema.go` is not in force), README has no `--llm-check`, and the cassette shipped at a different seam than the plan specified with no `## Revisions` entry.

## 1. Strengths

- **`internal/llm/render.go:12` — one renderer, two consumers, and the coupling is a test.** `TestGoldenAndCassetteKeyMoveTogether` (`golden_test.go:60`) and `TestAPromptEditMovesBothTheRenderAndTheHash` (`render_test.go:41`) assert the exact drift the design exists to prevent. `TestEveryMeaningfulFieldReachesTheHash` (`render_test.go:57`) is a table over every hash-bearing field — that is the rare test that catches a *future* field being forgotten, not just today's.
- **`render_test.go:26` — `TestRenderIsStableUnderMapIterationOrder` rebuilds the fixture each iteration.** Re-hashing the same map instance would prove nothing (Go randomises per-map, not per-read); the comment says so and the code does it. This is the property everything else rests on and it is genuinely pinned.
- **`anthropic_test.go:684` — the ordering test uses a real truncated capture, not a constructed one.** I verified the payload independently: `stop_reason: max_tokens`, text ` {"verdict":"yes", "reason":": Ā"}`. It decodes cleanly, so only the stop reason can catch it, and the test asserts both `errors.Is(ErrTruncated)` *and* `!errors.Is(ErrMalformed)`. That second assertion is what makes it a real discriminator.
- **`fake.go:300` — the sticky-last-reply change is correct and reversion-verified.** I reverted it to `m.queue = m.queue[1:]`: `TestTheLastScriptedReplyIsSticky` goes red, `TestQueueServesInOrder` stays green. The trap it removes (SDK retries a scripted 503 into the fallback's success) is real.
- **`cmd/define/llmcheck_test.go:118` — the wiring test drives `run()`.** I removed the dispatch from `main.go:279`; only `TestLLMCheckIsReachableFromTheFlag` failed, the three helper-level tests stayed green. That is precisely the lessons.md #15 rule working.

## 2. Critical findings

**C1 — `internal/llm/task.go:66` `decode` returns a half-populated `T` with a nil error.**

Measured directly against the shipped code:

```
{"fits":true}   -> {Fits:true  Reason:""} err=<nil>     ← partial value, nil error
{}              -> {Fits:false Reason:""} err=<nil>
null            -> {Fits:false Reason:""} err=<nil>
```

Three artefacts state the opposite. `task.go:59`: *"It never panics, and it never returns a partially populated value with a nil error."* `atlas/llm.md:112`: *"allow unknown fields, reject missing ones."* Plan Task 10: *"require every schema-required field present."* And `SchemaFor[vetoResult]` already emits `"required":["fits","reason"]` — the single source declares it and the decoder ignores it.

Consequence for the consumers this milestone exists to serve: `#12`'s veto reads `Fits:false` from `{}` and cannot distinguish it from a real "no" — a silently dropped distractor, not a skipped question. This is the issue's Done-when *"structured responses are parsed defensively"*, so that box must not be ticked at issue close until this is fixed.

Fix sketch — decode twice and derive the required set from the one source:
```go
var present map[string]json.RawMessage
if err := json.Unmarshal([]byte(body), &present); err == nil {   // object payloads only
    if schema, serr := SchemaFor[T](); serr == nil {
        for _, k := range schema["required"].([]any) { ... require present[k.(string)] ... }
    }
}
```
Then add `{}`, `{"fits":true}` and `null` to `TestDecodeRejectsWithoutPartialResults`, and make `FuzzDecode`'s success branch assert the populated half instead of `return`.

## 3. Important findings

**I1 — plan Task 9's schema golden was never written; `AssertGolden` ships with no committed artifact.** `internal/llm/testdata/` does not exist; `find` turns up no `golden/` or `cassettes/` directory anywhere in the tree; the only callers of `AssertGolden`/`Cassettes` are their own self-tests. Task 9 asked for a snapshot *and* a byte-identical assertion, precisely so "a struct field added without thought shows up in a diff." ARCH-PURPOSE shadow-sweep on the `SchemaFor[T]` single source: 4 consumers, 2 derive (the wire request, the request hash), 2 do not (the golden — absent; `decode` — C1).

**I2 — four doc claims in this diff assert properties the code does not hold** (4th in `docs-claim-absent-surface`; details in the findings block — the rule, not the four patches, is the deliverable).

**I3 — `README.md` is not updated for `--llm-check`.** README documents every other flag (`--sound`, `-locale`, `-raw`, `--forget`) and enumerates exit codes; `grep -n "llm" README.md` returns nothing, and the exit-code paragraph still says `1` means "no dictionary entry, or `--forget` found nothing to remove" — `--llm-check` now also exits `1`.

**I4 — the cassette double sits *above* the seam, reversing M1's central design decision, with no plan revision.** Plan Task 4 Step 3 specifies `func Cassette(t *testing.T, r llm.Request) Reply` — a body served *through* the wire `Fake`. What shipped is `func (c *Cassette) Client(live llm.Client) llm.Client` (`cassette.go:47`), which replaces `llm.Client` outright: replay reads a file and returns `rec.Response` without serialising a request, without the SDK, without SSE. `llmtest`'s own package doc (`fake.go:5`) argues at length that a stubbed `Client` cannot see a mis-serialized `output_config` or a dropped header — and a consumer test written against a cassette now cannot. `cassetteClient.Stream` (`cassette.go:101`) does not stream at all; it calls `Complete` and fires one delta. ARCH-MOCK.

**I5 — cassette replay collapses `ErrRequest` into `ErrUnavailable`.** `recorded.err()` (`cassette.go:122`) records the error as a *string* and re-derives the taxonomy from `Stop`. A recorded 400 (bad schema, bad model — the class `errors.go:20` says must stay loud) has `Stop: ""`, so `ErrorForStop` returns nil and it replays as `ErrUnavailable`, the quiet class. That is the one collapse the taxonomy exists to prevent. Record the taxonomy member alongside the text.

**I6 — the capture-drift suite reports drift when the proxy is merely unreachable.** `capture_conformance_test.go:31` skips only when `llm.Resolve` fails. With a key set and the proxy stopped, `Complete` returns `ErrUnavailable` and the test `t.Fatalf`s — telling the operator "the model may no longer think by default… Re-record" when nothing drifted. Plan Task 8 Step 2 asked explicitly for skip-not-fail. (2nd in `unclassified-failure-mode`.)

**I7 — both new test doubles discard the `llm.Request` entirely** (4th in `fake-silently-ignores-inputs`; rule stated in the findings block).

**I8 — the plan still describes a design M2 did not build,** and AGENTS.md §1 requires a `## Revisions` entry rather than a silent divergence (2nd in `plan-revision-not-appended`).

## 4. Minor findings

- `capture_conformance_test.go:132` — `_ = llmtest.Capture` is a dead statement whose only job is to keep an import alive (4th in `dead-code`; see rule in block).
- `fake_test.go:163` — `TestAQueueStillAdvancesWhileItHasEntries` is `TestQueueServesInOrder` with different strings; I confirmed it passes under the reverted implementation, so it pins nothing the older test doesn't. ARCH-DRY.
- `main.go:279` — `define -llm-check hello` silently ignores `hello`. The comment two lines above calls it "a mode, like `--forget`", but `--forget` gets an explicit arity guard (`main.go:288`) added because *"silently honouring one of them is how -raw came to mean two different things in #2."*
- `llmcheck.go:44` — hardcodes `MaxTokens: 2048` instead of `cfg.MaxTokens` (8192). `config.go:23` records that a small budget let thinking consume the answer and produced `message-truncated.json`; a truncated `PONG` would make the diagnostic report exit 1 on a healthy configuration.
- `schema.go:66` — `out["additionalProperties"] = false` is applied unconditionally: `SchemaFor[string]()` yields `{"type":"string","additionalProperties":false}` and `SchemaFor[map[string]string]()` yields an object that permits no keys at all.
- `cassette_test.go:107` — `*update = true` … `*update = false` without `defer`; a Fatal in between leaks the flag into every later test in the package, turning `AssertGolden` into a writer.
- `cassette.go:60` — `c.store.t.Fatalf` fires from whatever goroutine a consumer calls `Complete` on; `fake.go:326` states the opposing rule 260 lines away ("NOT t.Fatalf: this runs on the server's goroutine") (3rd in `test-helper-fatal-off-goroutine`).
- `cassette.go:128` — the artifact records the *answer* but not the *question*; the filename carries only `task-<12hex>`. `TestCassetteOnDiskIsReadable` asserts the answer is legible; storing `llm.RenderRequest(r)` in the record would make the diff self-describing.
- `Usage.Duration` has no production reader — `llmcheck.go:60` re-measures latency with its own `took()`; and `Updating()` vs `*update` are two readers of one flag inside one package.

## 5. Test coverage notes

Reversion-verified as genuinely red-on-revert: the sticky-reply fix, the `decode` clean-EOF fix (only `trailing_prose_after_the_object` went red — exactly the case the log claims), and the `-llm-check` flag dispatch. `go test ./...` and `go test -race` are green; `-tags conformance` compiles clean.

The gap the suite would not catch is C1's shape, and the reason is structural: `FuzzDecode` (`task_test.go:100`) states a two-sided invariant in prose and asserts one side — `if err == nil { return }`. Its own seed corpus contains `"{}"`, which is a *violating* input the target waves through. `TestDecodeRejectsWithoutPartialResults` covers eight malformed-syntax cases and zero missing-field cases. Also unpinned: `--llm-check` has never been driven through the real SDK path (a `llmtest.NewFake` + `llm.New` variant of `TestLLMCheckReportsAHealthyConfiguration` would cost ten lines and pin the request shape), and no test exercises `Run[T]` through a cassette.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one flag.** `renderRequest` as sole renderer is the model case, and `ErrorForStop` reuses `classifyStop` rather than restating it. Flagged: `recorded.err()` (I5) is a *second*, lossy classification path beside `classifyStatus`, and `truncate` reuse from `command.go:250` shows the right instinct that `llmcheck.go:60`'s `took()`/`Usage.Duration` duplication misses.
- **ARCH-PURE — pass.** `renderRequest`, `RequestHash`, `decode`, `stripFence`, `excerpt`, `SchemaFor` are all pure and unit-tested with no server; `render_test.go` and `task_test.go` need no IO at all. `runLLMCheck` is a thin shell over injected `getenv`/`newClient`. Every plan-table PURE row holds.
- **ARCH-PURPOSE — flag.** The shadow-sweep on the `SchemaFor[T]` single source finds two consumers not deriving from it (I1, C1). The pattern to watch as #10/#12/#13 land: the mechanism shipped, the artifact that makes the mechanism bind did not.
- **ARCH-MOCK — flag.** The wire fake, the two live conformance suites, and the named-remediation drift check are strong. But M2 introduced a *second* seam for the same dependency at a different depth (I4) and a lossy error reconstruction on it (I5). Before #12 writes its first cassette, settle whether a cassette is a `Reply` served by the `Fake` (plan) or a `Client` replacement (shipped) — that choice decides what every consumer test can see, and it should not be settled by whichever one exists.
- **For the consumers next up:** the `Task[T]`/`Run[T]` surface is stable and correctly shaped (package function over a two-method un-generic `Client`). The one thing to fix before it has five callers is C1, because each consumer will write its own "did the model answer?" guard otherwise.

## 7. Plan revision recommendations

Append one `## Revisions` entry to `workshop/plans/000011-vocab-llm-plan.md` covering:

- **Task 4 Step 3 / Core concepts:** `Cassette` shipped as an `llm.Client` wrapper (`Cassettes(t, dir).Client(live)`), not `func Cassette(t, r) Reply` served through `Fake`; storage moved from `testdata/cassettes/<hash>.json` to `<consumer-dir>/cassettes/<task>-<hash>.json`; the flag is `-update`, not `-record`. State why the seam moved, or move it back.
- **Integration points table:** add rows for `llmtest.Cassette` (`internal/llm/llmtest/cassette.go`, new) and `renderRequest`/`RequestHash` (`internal/llm/render.go`, new) — both delivered, neither in the table. Rename the `llmtest.Golden` row to `AssertGolden`, the actual identifier.
- **Task 9:** either deliver the schema golden or record the decision to drop it — the plan currently claims an artifact the tree does not contain.
- **Task 8 Step 2:** unavailable-vs-drift is not yet distinguished; either implement the skip or amend the step.

```findings
findings:
  - id: new
    severity: Critical
    family: enforcement-not-pinned-by-a-test
    title: |
      decode returns a partially populated T with a nil error when a required field is missing
    detail: |
      Measured against the shipped code: decode[answer](`{"fits":true}`) returns {Fits:true Reason:""} with err=nil; `{}` and `null` return the zero value with err=nil. task.go:59 states "it never returns a partially populated value with a nil error", atlas/llm.md:112 states "reject missing ones", and plan Task 10 states "require every schema-required field present". SchemaFor already emits "required":["fits","reason"] — the single source declares it and decode ignores it. #12's veto would read Fits:false from `{}` and silently drop a distractor rather than skip the question.
      THIS IS THE 4TH FINDING IN FAMILY `enforcement-not-pinned-by-a-test`. Do not fix only this site. The RULE: an invariant stated in a doc comment must be asserted by a test that goes red when it is violated — including the SUCCESS branch of a property/fuzz target, which is where FuzzDecode returns early (`if err == nil { return }`) and where its own seed `"{}"` is a violating input the target waves through. THE ENUMERATION to sweep in this round: grep every "never", "always", "must", "either ... or" claim in the doc comments added by this window (task.go decode, task.go Run, render.go renderRequest/renderSchema, schema.go SchemaFor, cassette.go Cassette/Stream, golden.go AssertGolden, fake.go next) and for each confirm a test that fails when the claim is broken; where none exists, either write it or delete the claim.
  - id: new
    severity: Important
    family: docs-claim-absent-surface
    title: |
      Four doc claims in this window assert properties the code does not hold
    detail: |
      Measured prevalence, 4 instances in one milestone. (1) atlas/llm.md:112 "allow unknown fields, reject missing ones" — decode does not reject missing (see the Critical). (2) atlas/llm.md:114 "Its invariant ... is held by a fuzz target" — FuzzDecode asserts only the error half. (3) atlas/llm.md:171 "Every failure names scripts/llm-probe.sh record" and capture_conformance_test.go:22 "On drift the failure names the fix" — 2 of 7 failure messages in that file name it; the unknown-block-type, no-text-block, output_config-decode, no-deltas and preamble messages do not. (4) internal/llm/schema.go:15 "llmtest.Golden snapshots it, so a struct field added without thought shows up in a diff" — no golden file exists anywhere in the tree, and `llmtest.Golden` is not an identifier (it is `AssertGolden`).
      THIS IS THE 4TH FINDING IN FAMILY `docs-claim-absent-surface`. Do not patch the four sentences. The RULE: a doc claim of UNIVERSAL form ("every", "always", "never", "is held by") is a claim about an enumeration, so it may only be written after enumerating the sites and checking each — and a doc claim naming a code identifier or an on-disk artifact must be grep-verified against the tree in the same edit. Sweep: for each universal claim in atlas/llm.md and in the doc comments of render.go, schema.go, task.go, cassette.go and golden.go, run the enumeration it implies and either make it true or weaken it to what is true.
  - id: new
    severity: Important
    family: single-source-consumer-not-derived
    title: |
      Plan Task 9's schema golden was never written, so AssertGolden ships with zero committed artifacts
    detail: |
      internal/llm/testdata/ does not exist; `find` returns no golden/ or cassettes/ directory in the repo; the only callers of AssertGolden and Cassettes are their own self-tests against t.TempDir(). Task 9 required a snapshot AND a byte-identical assertion, and schema.go:15 cites that snapshot as the property that justifies reflecting the schema instead of hand-writing it. ARCH-PURPOSE shadow-sweep on the SchemaFor[T] single source: 4 consumers, 2 derive (Request.Schema on the wire, RequestHash), 2 do not (no golden; decode ignores the required list).
  - id: new
    severity: Important
    family: user-surface-undocumented
    title: |
      README.md is not updated for the new --llm-check flag or its exit code
    detail: |
      `grep -n "llm" README.md` returns nothing. README documents --sound/-times, -locale, -raw, --forget and DEFINE_NO_CAPTURE, and its exit-code paragraph enumerates what `1` means ("no dictionary entry, or --forget found nothing to remove") — --llm-check now also exits 1, for a third reason. main.go's usage text was updated; README was not.
  - id: new
    severity: Important
    family: double-above-the-seam
    title: |
      The cassette double replaces llm.Client, bypassing the SDK path the plan places it beneath
    detail: |
      Plan Task 4 Step 3 specifies `func Cassette(t *testing.T, r llm.Request) Reply` — a body served through the wire Fake. Shipped is `func (c *Cassette) Client(live llm.Client) llm.Client` (cassette.go:47): replay reads the file and returns rec.Response without serialising a Request, without the SDK, without SSE. llmtest's own package doc (fake.go:5) argues that a stubbed Client cannot see a mis-serialized output_config, a dropped anthropic-version header, or a retry that re-sends a consumed body — a consumer test written against a cassette now cannot see any of them. cassetteClient.Stream (cassette.go:101) does not stream: it calls Complete and fires a single delta. ARCH-MOCK: production flow and test flow no longer share the same boundary on this path.
  - id: new
    severity: Important
    family: double-rederives-error-taxonomy
    title: |
      Cassette replay collapses a recorded ErrRequest into ErrUnavailable
    detail: |
      recorded.err() (cassette.go:122) stores the error as a string and re-derives the taxonomy from Stop. A recorded 400 — bad schema or unknown model, the class errors.go:20 says must stay LOUD — carries Stop:"" , so ErrorForStop returns nil and the replay falls through to `fmt.Errorf("%w: %s", llm.ErrUnavailable, r.Err)`, the quiet class every consumer degrades on. Recording the taxonomy member itself (or the HTTP status) instead of re-deriving it removes the second classification path.
  - id: new
    severity: Important
    family: unclassified-failure-mode
    title: |
      The capture-drift conformance suite reports drift when the proxy is merely unreachable
    detail: |
      capture_conformance_test.go:31 skips only when llm.Resolve fails (no key). With a key set and the proxy stopped, Complete returns ErrUnavailable and every subtest t.Fatalf's — telling the operator "the model may no longer think by default ... Re-record: scripts/llm-probe.sh record" when nothing drifted. Plan Task 8 Step 2 asked for skip-not-fail explicitly. THIS IS THE 2ND FINDING IN FAMILY `unclassified-failure-mode`: the rule is that a check must distinguish "dependency unreachable" from "dependency changed" before reporting either, and internal/llm/conformance_test.go:30 has the same gap, so fix both.
  - id: new
    severity: Important
    family: fake-silently-ignores-inputs
    title: |
      Both test doubles added this window discard the llm.Request entirely
    detail: |
      checkClient (cmd/define/llmcheck_test.go:14) and stubLive (internal/llm/llmtest/cassette_test.go:17) both take `context.Context, llm.Request` and name neither parameter. Consequence: nothing asserts that --llm-check's request is well-formed — MaxTokens:2048, Task:"llm-check" and the PONG prompt could all be dropped and the four llmcheck tests stay green, while the real flag 400s. The repo already has a wire fake (llmtest.NewFake + llm.New) that would catch it and is importable from cmd/define.
      THIS IS THE 4TH FINDING IN FAMILY `fake-silently-ignores-inputs`. The RULE: a double must either record its input for assertion or be replaced by the wire-level fake that already exists for that dependency; a double whose method signature discards its request parameter cannot fail for any reason related to what was asked. Sweep every type in the tree implementing llm.Client, cmd/define's fetch seam, and the store seams, and confirm each records or asserts its input.
  - id: new
    severity: Important
    family: plan-revision-not-appended
    title: |
      The plan still describes an M2 design that was not built, with no Revisions entry
    detail: |
      Undeclared deltas: Cassette's seam, API and storage path all changed (see the double-above-the-seam finding); the flag is -update, not -record; Task 9's golden snapshot was dropped; the Integration points table names `llmtest.Golden` where the identifier is `AssertGolden` and has no row for llmtest.Cassette or for renderRequest/RequestHash (internal/llm/render.go), both delivered.
      THIS IS THE 2ND FINDING IN FAMILY `plan-revision-not-appended`. The rule per AGENTS.md section 1: any divergence from a plan artifact discovered during implementation is appended as a timestamped `## Revisions` delta in the SAME commit that diverges — so the sweep is not "add one entry now" but "diff the plan's Core concepts and Task lists against the tree at each milestone close and append what moved".
  - id: new
    severity: Minor
    family: dead-code
    title: |
      `_ = llmtest.Capture` exists only to keep an import alive
    detail: |
      capture_conformance_test.go:132. The comment calls it "the committed artifacts this run is checking", but the statement checks nothing — llmtest is otherwise unused in the file.
      THIS IS THE 4TH FINDING IN FAMILY `dead-code`. The RULE: a statement whose only effect is to satisfy the compiler is not documentation — drop the import and put the sentence in the doc comment, or make the reference load-bearing (here: read the capture and compare a field against the live response, which is what the file claims to do).
  - id: new
    severity: Minor
    family: redundant-test-duplicates-existing
    title: |
      TestAQueueStillAdvancesWhileItHasEntries duplicates TestQueueServesInOrder
    detail: |
      fake_test.go:163 vs fake_test.go:80 — same script shape (429 then a text reply), same two assertions, different string literals. Verified: with the sticky change reverted, the new test still passes, so it pins nothing the older one does not. ARCH-DRY.
  - id: new
    severity: Minor
    family: mode-flag-arity-guard
    title: |
      `define -llm-check <word>` silently ignores the word
    detail: |
      main.go:279 returns before the arity switch. The comment immediately above calls --llm-check "a mode, like --forget", but --forget has an explicit guard (main.go:288: "-forget takes the word to remove; do not also pass one") added because "silently honouring one of them is how -raw came to mean two different things in #2". Same guard, same reason.
  - id: new
    severity: Minor
    family: diagnostic-ignores-config
    title: |
      --llm-check hardcodes MaxTokens 2048 instead of the resolved cfg.MaxTokens
    detail: |
      llmcheck.go:44. config.go:23 records that an under-budgeted max_tokens let adaptive thinking consume the whole allowance and returned an answer cut mid-rune — the committed message-truncated.json. A truncated PONG surfaces as ErrTruncated and exits 1, so the diagnostic would report a healthy configuration as broken.
  - id: new
    severity: Minor
    family: schema-metadata-applied-blindly
    title: |
      additionalProperties:false is set unconditionally, including on non-object schemas
    detail: |
      schema.go:66. Measured: SchemaFor[string]() -> {"type":"string","additionalProperties":false}; SchemaFor[map[string]string]() -> {"type":"object","additionalProperties":false}, an object that permits no keys at all. The adjacent comment justifies stripping $schema/$id because "the provider rejects a schema carrying JSON Schema metadata it does not use" — the same argument applies to additionalProperties on a string or array.
  - id: new
    severity: Minor
    family: test-helper-fatal-off-goroutine
    title: |
      Cassette.Client's t.Fatalf fires from whatever goroutine a consumer calls Complete on
    detail: |
      cassette.go:60/66/74/80/84 call c.store.t.Fatalf from inside an llm.Client, a value designed to be handed to arbitrary consumer code including concurrent authoring loops. fake.go:326 states the opposing rule for the same package ("NOT t.Fatalf: this runs on the server's goroutine, where Fatalf becomes a hang or a 'log after test completed' panic").
      THIS IS THE 3RD FINDING IN FAMILY `test-helper-fatal-off-goroutine`. The RULE: a helper may call t.Fatalf only if it is structurally guaranteed to run on the test goroutine; anything returned to a caller as a value (a Client, a handler, a callback) must return an error instead. Enumerate every exported llmtest constructor that captures *testing.T and classify each by that criterion.
  - id: new
    severity: Minor
    family: test-flag-mutation-leaks
    title: |
      TestCassetteReplaysTheTaxonomy sets *update without a defer
    detail: |
      cassette_test.go:107-112 does `*update = true` ... `*update = false` inline; a Fatal in the Complete call between them leaks -update into every subsequent test in the package, turning AssertGolden from a comparator into a writer. recordThenReplay (cassette_test.go:34) uses defer correctly; this site does not.
  - id: new
    severity: Minor
    family: artifact-omits-the-question
    title: |
      A cassette records the answer but not the question
    detail: |
      recorded (cassette.go:117) stores only Response and an error string; the filename carries task plus a 12-hex hash. TestCassetteOnDiskIsReadable asserts the ANSWER is legible in a diff, but a reviewer cannot tell what was asked without recomputing the hash. Storing llm.RenderRequest(r) alongside would make the artifact self-describing, and is free — the miss message already renders it.
```
