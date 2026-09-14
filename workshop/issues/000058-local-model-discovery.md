---
id: 000058
status: codecomplete
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 2.634
started: 2026-09-13T22:21:55-07:00
actual_hours: 0.84
---

# define: discover local proxy models and select by preference

## Problem

`define` assumes `claude-opus-5` even when a user's local CLIProxyAPI has
only Codex or Antigravity configured. A Homebrew installation should use the
providers available through that user's proxy without requiring a model export.
The preceding invalid-key report is separate: model discovery itself requires
a key accepted by the proxy.

## Spec

### Provider-first selection

When `DEFINE_LLM_MODEL` is unset and the endpoint is the default local proxy,
query its authenticated `GET /v1/models` on first model use. Select a provider
from advertised `owned_by` metadata, then a preferred advertised model within
that provider. Do not infer the configured provider from a `claude-` or `gpt-`
model prefix: Antigravity can itself advertise Claude models.

Approved preference order (including the user's Flash-first correction):

1. Direct Claude (`anthropic`): newest advertised text Opus version.
2. Codex (`openai`): prefer recognized advertised GPT version 5.6, then version
   6, then newest recognized general text GPT version. The user's Sol/Astra
   labels express these version preferences, not literal API ID spellings.
   Initially recognize canonical `gpt-N[.N[.N]]` and its `-codex` variant;
   prefer canonical over `-codex` at the same version. Other aliases require
   an explicit override until their spelling/meaning is verified.
3. Antigravity (`antigravity`): newest advertised Gemini Flash text model, then
   newest Gemini Pro text model. At the same version/tier, prefer `high`,
   then unsuffixed, then `low`, then `lite`; use exact ID lexical order as the
   last deterministic tie-break. Exclude image, audio, and embedding models.
   Exclude agent variants as a product policy for this initial general-text
   selector, not as a claim that their client request contract differs.

Selection is deterministic under catalog reordering and duplicates. Version
comparison is numeric, not lexical. Exact IDs are sent unchanged. The selector
must recognize only documented/observed model grammars; do not manufacture IDs
from the user's display names or select arbitrary unknown aliases. If a provider
has no eligible model, consider the next provider. Unknown aliases remain usable
through the explicit override.

The public catalog is an advertised routing view, not an exhaustive account
inventory: multiple providers registering one ID may collapse into one entry.
Use the owner returned by the proxy; do not claim to identify every login, select
a specific account, or override proxy-side routing. Reading credential files or
requiring a management API key is outside this feature.

An explicit nonempty `DEFINE_LLM_MODEL` bypasses discovery and retains existing
behavior. Custom endpoints retain their current explicit-model/default behavior;
automatic discovery in this issue is scoped to `http://127.0.0.1:8317`.
Key precedence remains `DEFINE_LLM_API_KEY`, then `ANTHROPIC_API_KEY`, then the
existing local default. Do not retry authentication with different credentials.

### Timing, errors, and observability

- `Resolve` and client construction remain free of IO. Ordinary dictionary
  lookup, startup, and harvest runs requiring no model calls do no discovery.
- Discover only when the first actual Complete/Stream operation needs an
  automatic model. Reuse a successful selection for that client lifetime;
  a newly constructed client discovers again. No global or on-disk cache.
- Bound discovery to five seconds or the caller's remaining request deadline,
  whichever is shorter. Limit response reads to 1 MiB and 4096 catalog entries.
  Discovery and completion share the existing total request deadline.
- Concurrent callers share one active discovery. Each waiting caller can cancel
  independently; a cancelled or failed discovery must not permanently poison the
  client. The implementation plan will enumerate the exact state transitions.
- Empty lists/no recognized eligible providers report model unavailability.
  Authentication errors name the proxy/key configuration, without exposing key
  material. Malformed/oversized catalogs report a discovery error. No failed
  discovery falls back to the previously hardcoded Opus model.
- The selected provider/model is visible in `--llm-check`. Learner-model
  provenance records the selected model rather than `auto` or the old default.
- This is selection fallback only. A selected model's quota, availability,
  malformed answer, or mid-stream failure keeps the existing request/error
  behavior; it does not trigger another model call.

### Architecture and verification

Keep provider classification, model eligibility, numeric ordering, and selection
as a pure core in `internal/llm` (ARCH-PURE). A thin discovery shell uses the
existing Config.Transport seam; both Complete and Stream consume the same
selection mechanism (ARCH-DRY). All define LLM consumers inherit the behavior,
including ask, harvest, reflect, and diagnostics (ARCH-PURPOSE).

Extend the existing stateful `llmtest` HTTP fake with catalog state and discovery
request history, auth, delay, errors, and selected-model validation (ARCH-MOCK).
Tests must prove provider precedence, Antigravity-served Claude classification,
empty/unknown/duplicate/shuffled catalogs, numeric versions, override bypass,
auth failures, bounded response parsing, deadline/cancellation races, client
reuse, no-LLM paths with zero discovery requests, selected provenance, and
streaming/structured calls for each supported provider.

CLIProxyAPI translates `/v1/messages` for other providers, but its current
Claude-source Codex/Gemini/Antigravity translators omit the JSON-schema output
setting. For auto-selected non-Claude providers, the shared request renderer
must also supply the existing request schema as explicit JSON-only instructions
alongside the domain prompt; continue using the existing typed decoder to reject
malformed/incomplete results. Derive those instructions from Request.Schema,
never maintain a parallel schema. Direct Claude keeps its existing renderer.
For those non-Claude requests, translate the existing effort setting through
the proxy's supported adaptive-thinking contract, with deterministic wire tests.
Do not claim upstream schema enforcement or live compatibility without a live
test. Eligible IDs must be supported by source inspection or runtime discovery,
especially the requested Sol/Astra display names; unknown mappings are never
guessed.

ARCH-CONSTRAINTS: the budgets above bound the interactive IO path.
ARCH-ORDER: selection is unselected/discovering/selected, with cancellation and
retry covered by controlled interleaving tests. ARCH-SECURE: treat catalog fields
as untrusted; never follow redirects carrying proxy auth, display raw errors
containing secrets, or print unsanitized model metadata to the terminal.
ARCH-FUNERAL: selection state dies with its owning client; no durable state is
introduced by discovery.

## Done when

- A default installation selects a usable advertised model from the user's
  configured provider, following the approved provider/model preference policy.
- Explicit model configuration bypasses discovery; ordinary dictionary use
  remains offline; request failures do not silently switch models.
- Selection and failures are diagnosable without exposing credentials, and
  learner-model provenance identifies the model actually selected.
- Pure selector tests and stateful HTTP integration tests cover the cases above;
  README and atlas describe the verified contract and limits.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.24
item: api-integration design=0.20 impl=0.40
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.20
item: real-api-discovery design=0 impl=0.16
design-buffer: 0.15
total: 2.634
```

Method A decomposition: selector (v2 1.0 design / 0.6 impl), discovery API
(2.0 / 1.0), lazy client (1.0 / 0.8), renderer (0.3 / 0.5), consumer wiring
(0.3 / 0.5), docs (0.1 / 0.1), one close review (0.1 / 0.5), and API discovery
(0 / 0.4). net/http and existing SDK/fake halve discovery design before the
thorough-spec x0.2 design discount. All other design uses x0.2. Each impl value
is v2 x0.4 per v3.1; existing Go/HTTP patterns give familiarity 1.0. No library
replaces this provider-ranking policy or its lazy selection lifecycle. Design
sum 0.76 x1.15 plus impl sum 1.76 = 2.634 hours. Calibration is marked stale,
so these values remain provisional.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Plan

- [x] Approve the provider-first spec and land a reviewed durable implementation
  plan at `workshop/plans/000058-local-model-discovery-plan.md`.
- [x] Implement selection and discovery with regression tests through the shared
  transport, including diagnostics and provenance.
- [x] Verify and update docs/atlas. SDLC close review follows these deliverables.

## Log

### 2026-09-13
- 2026-09-13: closed — Full define/llm suites and llm race tests passed; conformance guard passes after BR-1 fix; absent dependency skips by default and fails strict mode; live Claude plain/stream/structured passed; all provider routes fake-verified; release-stamp merge checks passed.; review verdict: SHIP

- Created and claimed at design start. User confirmed discovery-based selection
  with explicit DEFINE_LLM_MODEL override, then clarified that the primary
  decision is which provider the user has configured, not a global model rank.
- Live read-only probe of local `GET /v1/models` with the existing local default
  key returned HTTP-success JSON with `data: []`; this validates the empty-catalog
  case, not provider inference or model completion.
- Source exploration found provider ownership metadata and a distinction between
  direct Claude and Antigravity-served Claude. The checkout's embedded model
  catalog does not establish Sol/Astra IDs. No implementation code changed.
- Fresh-context spec review approved implementation planning after resolving
  schema transport, Gemini tie-breaks, and display-name/API-ID ambiguity.
  Awaiting user approval of the proposed provider/model policy before the
  durable implementation plan and change-code gate.
- User corrected Antigravity preference to Flash before Pro; proceeding to
  implementation planning with that policy. The other provider/override decisions
  stand.
- Durable implementation plan written and approved by fresh-context review at
  `workshop/plans/000058-local-model-discovery-plan.md`. Review emphasized that
  both request hashing and SDK parameters must consume the same effective
  request, now explicit in the plan. Awaiting implementation-plan approval under
  AGENTS.md Section 2; no product code changed.

- Implementation completed on the issue branch. Selector/config, bounded HTTP
  discovery/stateful fake, lazy client and provider rendering are shared in
  internal/llm. Reflection and diagnostics report selected-model provenance.
- TDD evidence: selector/config and discovery initially failed on missing APIs;
  renderer wire assertions failed for absent JSON instructions/adaptive thinking;
  consumer tests failed by reporting/storing claude-opus-5 despite selecting
  GPT/Gemini. All now pass. Agents' changes integrated and inspected locally.
- Verification: `go test ./internal/llm/... -count=1` passed; full
  `go test ./cmd/define/... -count=1` passed (define 108.629s); final
  `go test -race ./internal/llm/... -count=1` passed (13.150s/10.147s).
  Selector, catalog parser and schema-rendering short fuzz runs passed.
- Live `TestConformanceAutoSelection` passed plain, streaming, and schema-shaped
  requests with anthropic/claude-opus-5. Catalog availability changed since the
  initial empty probe. Codex/Antigravity are verified with the stateful HTTP fake,
  not live. Concurrent discovery/cancellation and inherited deadlines are pinned
  by deterministic tests. Commits group the independently tested tasks together
  after integration rather than committing interdependent partial builds.

- Close review round 1 returned REWORK for BR-1: three live-test skips bypassed
  the shared conformance policy. Reproduced with TestEverySkipIsRoutedOrWaived;
  routed every new skip through conformance.SkipOrFail. Conformance package tests
  now pass; absent credentials skip by default and fail under strict mode.
  Added the prevention rule to lessons. Merge-check release stamping also passed.

- Close round 2 approved SHIP with no blocking findings. Addressed its advisory
  atlas inventory omission by listing TestConformanceAutoSelection and the shared
  strict/default policy. Full `go test ./... -count=1` passed, including
  internal/conformance; all code is verified and only final publication remains.

## Revisions

- 2026-09-13: User requested Flash before Pro for Antigravity; reversed that
  provider's tier order. Claude then Codex then Antigravity is unchanged.

- 2026-09-13: Proxy source review established that Claude-source translators drop
  schema output configuration for Codex/Gemini/Antigravity. Added schema-derived
  JSON instructions plus local validation to the proposed non-Claude request
  contract so selection also supports define's structured tasks.
- 2026-09-13: Spec review requested concrete Gemini tie-breaks and cautioned
  against inventing Sol/Astra API names. Added suffix ordering, identified agent
  exclusion as a product policy, and expressed Codex preferences as recognized
  advertised numeric versions with unknown aliases left to the explicit override.
