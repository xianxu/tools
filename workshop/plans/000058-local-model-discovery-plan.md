# Local provider discovery implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let define select an advertised model from the user's configured local provider: Claude Opus, otherwise Codex, otherwise Antigravity Flash before Pro.

**Architecture:** A pure catalog selector feeds a lazy client wrapper in `internal/llm`. The wrapper discovers and pins a concrete Anthropic-wire client on first model use, preserving existing consumers and the HTTP transport seam. Diagnostics and reflection inspect the chosen model after use.

**Tech Stack:** Go, net/http, existing Anthropic Go SDK, existing stateful llmtest fake.

**Spec:** `workshop/issues/000058-local-model-discovery.md`. One atomic issue-close boundary; task numbers below are not milestones.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ModelInfo, ModelSelection, SelectModel | internal/llm/models.go | new |
| Config and Resolve | internal/llm/config.go | modified |
| effective request rendering | internal/llm/anthropic.go, internal/llm/render.go | modified |

`ModelInfo` retains only ID and owner from discovery. `ModelSelection` is the
chosen owner/ID pair. `SelectModel` performs provider-first numeric ranking,
independent of HTTP, clocks, or configuration files. Colocated table tests cover
each supported grammar and ordering. The selector is the only preference table.

`Config` gains `AutoModel bool`, set by pure Resolve only when the model env var
is empty and BaseURL equals the existing default exactly. Config.Model retains
the existing concrete default for compatibility; AutoModel takes precedence only
inside New. Manually constructed configs remain pinned unless AutoModel is true.
Explicit request Model bypasses discovery for that request and does not replace
the client's eventual automatic selection. No new environment variable.

The selected concrete client's config carries an internal provider classification.
Its pure effective renderer derives non-Claude JSON instructions from the same
Request.Schema passed to the SDK, and enables adaptive thinking. Effective
request hashing must include the new wire-changing fields, including thinking mode;
never mutate the caller's schema map. Add an internal effective-thinking field
to Request if necessary and include it in renderRequest. Existing fixed/direct
Claude configs retain their current rendering. Preserve the existing deliberate
MaxTokens hash exclusion; emit the new thinking line only when enabled so old
direct-Claude recordings and goldens remain valid.
Both withRequest and SDK params must consume the same effective Request.
Currently Complete/Stream hash effective(r) but pass the original r to params;
change that data flow so generated JSON instructions reach the wire too.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| discoverModels | internal/llm/discovery.go | new | authenticated GET /v1/models |
| autoClient | internal/llm/auto.go | new | lazy discovery, mutex/channel, concrete Client |
| SelectionOf | internal/llm/auto.go | new | read-only optional selection reporter on Client |
| llmtest catalog state | internal/llm/llmtest/fake.go, internal/llm/llmtest/models.go | modified/new | stateful HTTP fake |
| diagnostic/provenance consumers | cmd/define/llmcheck.go, cmd/define/reflect.go | modified | terminal and stored learner model |

`New` returns autoClient when requested, otherwise the existing concrete client.
The wrapper's Complete/Stream both resolve once then delegate. SelectionOf uses
a private optional interface rather than expanding the public Client interface;
unrecognized test/custom clients return an empty selection. Pinned production
clients report their configured model; auto clients report empty until selected.
Reflection retains the client used for llm.Run and inspects it afterward,
falling back to cfg.Model only for custom clients without a reporter. Diagnostics
print `auto` before discovery, then the actual provider/model after selection,
including when inference fails after selection. An explicit request override is
per-call and must not be misreported as the cached default.

## Operating envelope and boundaries

- Default loopback endpoint only; use the existing resolved key unchanged. No
  proxy login, credential-file reads, management API access, or process repair.
- Discovery: GET /v1/models with bearer auth; use Config.Transport. Reject
  redirects, do not return raw server bodies or transport errors containing auth.
  Parse one complete JSON object with a data array; reject missing/null data,
  trailing JSON, malformed entries, and oversized input. Ignore unknown fields.
  Empty arrays are valid catalogs but have no selectable model.
- Five-second discovery cap, 1 MiB body cap, 4096 entries. Model IDs/owners must
  be printable ASCII without control characters; impose 256-byte ID and 64-byte
  owner limits. Unknown well-formed owners/models are skipped by the selector.
  Return ErrUnavailable for auth/network/empty/unsupported catalog; malformed,
  oversized, and unexpected discovery statuses receive a sanitized error with
  existing appropriate taxonomy, explicitly tested. Do not retry discovery.
- Wrapper sets the existing total request deadline before discovery; the
  delegate inherits that context so discovery cannot add five seconds beyond it.
- No IO from Resolve/New, no probes on dictionary/cache-only paths. Success is
  cached only for a client lifetime; a fresh client discovers afresh.
- No model switch after an inference failure. Existing SDK retries remain.
- HTTP fake owns catalog contents, request counts, auth expectations, controlled
  release channels and response errors. Tests use fake URLs through injected
  transport/config and must never depend on the real loopback service.

### Selection state transitions (ARCH-ORDER)

State is one tagged outcome under a mutex: unselected, discovering(flight), or
selected(selection, delegate). A flight owns a done channel and result/error.

| State/event | Action/result |
|---|---|
| Unselected / auto request | Install flight; caller synchronously owns discovery |
| Discovering / another auto request | Wait on flight.done or own ctx.Done |
| Discovering / waiting caller cancels | Return to that caller; preserve active flight |
| Discovering / owner succeeds | Publish immutable selection/delegate; close done |
| Discovering / owner fails or cancels | Publish flight error to attached callers; clear state; close done |
| Unselected after failure / later request | New bounded discovery attempt |
| Selected / request | Delegate using pinned model; no discovery |
| Any / explicit request model | Use pinned override client without changing auto state |

No detached goroutines: the initiating caller owns discovery. Cancelling that
owner fails its attached flight; other waiters receive that flight error, and a
later call may retry. Cancelling a waiting caller does not cancel the owner.
All waiting and network work is context-bounded. Race tests orchestrate each
interleaving with channels, not sleeps or repeated probabilistic runs.

ARCH-DRY/PURE: one selector and effective renderer shared by both request modes.
ARCH-PURPOSE: cover all define consumers, including selected-model provenance.
ARCH-MOCK: stateful fake at the existing wire seam, plus optional live conformance.
ARCH-CONSTRAINTS: explicit size/time limits above; no startup latency added.
ARCH-SECURE: authenticated loopback boundary, bounded untrusted catalog parsing,
no redirects/auth disclosure, validated identifiers before terminal output.
ARCH-FUNERAL: no files or global cache; client state ends with its owner.

## Chunk 1: Selection through consumption

### Task 1: Pure policy and config intent

**Files:** create `internal/llm/models.go`, `internal/llm/models_test.go`; modify
`internal/llm/config.go`, `internal/llm/config_test.go`.

- [ ] Write table tests first: direct Claude wins over Codex, which wins over
  Antigravity; Antigravity Claude IDs cannot masquerade as direct Claude;
  Flash beats Pro even when Pro has a newer version; numeric 3.10 beats 3.9;
  GPT 5.6 beats 6, which beats other recognized versions; canonical GPT beats
  its -codex sibling; malformed/unknown/image/agent IDs are not auto-selected.
  Cover empty inputs and all permutations of representative duplicate entries.
- [ ] Run `go test ./internal/llm -run 'TestSelectModel|TestResolve' -count=1`;
  confirm new assertions fail before implementing.
- [ ] Implement anchored, bounded grammars for direct Opus numeric versions
  with optional dated suffix, GPT numeric versions with optional -codex, and
  Gemini numeric versions plus flash/pro and documented high/low/lite suffixes.
  Use numeric tuple comparison and final exact-ID lexical tie-break. Implement
  AutoModel config intent without performing IO or changing key precedence.
- [ ] Repeat the focused command and confirm success; commit task changes with
  an issue-referencing message and Co-Authored-By trailer.

### Task 2: Bounded discovery and fake catalog

**Files:** create `internal/llm/discovery.go`, `internal/llm/discovery_test.go`,
`internal/llm/llmtest/models.go`; modify `internal/llm/llmtest/fake.go` and add
`internal/llm/llmtest/models_test.go`.

- [ ] Add stateful fake catalog configuration and GET history separately from
  message counts. Preserve existing fake default behavior; explicitly configured
  catalog models become valid completion targets. Inspect outgoing schema/prompt
  and model against expectations rather than answering every request blindly.
- [ ] Add failing tests for valid/empty/unknown catalog, auth rejection, missing
  or malformed data, trailing JSON, each size limit, bounded cancellation, and
  a redirect target that must receive zero authenticated requests.
- [ ] Run `go test ./internal/llm/... -run 'TestDiscover|TestCatalog' -count=1`,
  implement the bounded HTTP parser using Config.Transport, and repeat to pass.
- [ ] Commit the tested discovery/fake changes.

### Task 3: Lazy client and cross-provider request contract

**Files:** create `internal/llm/auto.go`, `internal/llm/auto_test.go`; modify
`internal/llm/anthropic.go`, `internal/llm/anthropic_test.go`, `internal/llm/llm.go`,
`internal/llm/render.go`, `internal/llm/render_test.go`.

- [ ] Write failing integration tests for no IO at construction, one successful
  catalog fetch per client, Complete/Stream sharing selection, explicit config
  and request model bypass, and every concurrency transition above. Capture the
  HTTP request to prove selected IDs reach the wire, auth remains unchanged, and
  errors cannot silently trigger a different model.
- [ ] Add provider contract tests for schema-derived JSON instructions and
  adaptive effort on auto-selected Codex/Antigravity; direct Claude unchanged.
  Assert schema bytes/structure appear in the actual request, malformed answers
  still fail the existing decoder, and effective-request hashes differ whenever
  model, schema instructions, effort or thinking mode changes.
- [ ] Run `go test ./internal/llm/... -count=1` and confirm new failures;
  implement the wrapper/state transitions and shared effective renderer.
- [ ] Run `go test -race ./internal/llm/... -count=1`; confirm deterministic
  concurrency assertions pass, then commit.

### Task 4: Diagnostics, provenance, docs and closure

**Files:** modify `cmd/define/llmcheck.go`, `cmd/define/llmcheck_test.go`,
`cmd/define/reflect.go`, `cmd/define/reflect_run_test.go`,
`cmd/define/askrun_test.go`, `cmd/define/harvest_test.go`,
`internal/llm/conformance_test.go`, `cmd/define/README.md`, `atlas/llm.md`.

- [ ] Add failing tests proving diagnostics display selected owner/ID, reflection
  persists selected ID, ask/harvest inherit provider selection, and ordinary
  dictionary plus fully cached harvest paths send no discovery requests.
  Retain the same Client instance for selection inspection after typed Run.
- [ ] Run `go test ./cmd/define -run 'TestLLMCheck|TestReflect|TestHarvest|TestAsk' -count=1`;
  implement reporting only at existing consumer seams and rerun to pass.
- [ ] Extend opt-in conformance to record the discovered owner/ID and exercise
  plain, streaming and schema-shaped answers for an available provider. Missing
  real providers skip with an explicit reason; never report a skipped case as
  verified compatibility. No inference calls are required to finish a unit test.
- [ ] Update README/atlas with provider-first order, Flash before Pro, explicit
  override, lazy discovery, auth and catalog limitations, selection-only fallback,
  and local JSON validation rather than upstream schema enforcement. Update
  atlas/index.md only if a new atlas page is introduced (none planned).
- [ ] Run `go test ./internal/llm/... ./cmd/define/... -count=1`,
  `go test -race ./internal/llm/... -count=1`, and `git diff --check`.
  Run `bash scripts/run-merge-checks.sh BASE_SHA HEAD` over this issue's
  implementation range using its actual branch-point SHA;
  record commands/results, including live cases unavailable on this machine.
- [ ] Tick issue/plan tasks, record evidence, and use
  `sdlc close --issue 58 --verified '<actual evidence>'`. Let that boundary
  dispatch its review; fix required findings, update lessons for code-review
  mistakes, then ship through `sdlc pr` and `sdlc merge`.

## Review and approval

The user approved provider-first selection and corrected Antigravity to Flash
before Pro. This implementation plan requires review and approval under AGENTS.md
Section 2 before `sdlc change-code`. Estimate is derived only after plan-quality
acceptance, as required by the gate.

Fresh-context plan review: approved, with the effective-request data-flow
detail above called out explicitly. No blocking findings remain.
