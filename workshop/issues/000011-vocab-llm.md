---
id: 000011
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-22
estimate_hours: 5.78
started: 2026-08-22T17:11:23-07:00
---

# LLM seam: Anthropic client, stateful fake, offline degradation

## Problem

Two review forms need judgment a local rule cannot supply: grading a sentence
a person wrote, and confirming that a candidate distractor really is wrong.

## Spec

An LLM seam with a stateful fake.

- Anthropic API behind a narrow interface — one method per task, not a general
  chat call, so each prompt is a testable unit.
- **Degradation is a requirement, not a nicety.** With no key or no network,
  `--play` silently falls back to forms 2.1 and 2.3 (which need neither) rather
  than failing. The learner's review is never blocked on a third party.
- The model's job stays **narrow and checkable**: it verifies or ranks candidates
  that were selected locally (#10, #12); it does not invent the option set.
- Fake records prompts and returns canned completions, so prompt regressions are
  visible in a diff. Live conformance behind the build tag, like every other seam.
- Cost and latency are per-question and user-visible; budget them explicitly.

## Done when

- [x] ~~`--play` runs a full session with the seam unavailable, using local
      forms.~~ **Relocated 2026-08-22** — `--play` does not exist until #6, so
      this could only ever have been ticked dishonestly here. Now carried by #6
      (the loop degrades), #12 (veto skipped — it already held this row) and #13
      (form skipped). What stays here is the property they all rest on, below.
- [ ] `ErrUnavailable` is returned for no key, no network, 429 and 5xx, and is
      distinguishable from `ErrRequest`, which stays loud. This is the half of
      the degradation contract that IS testable in this issue.
- [ ] Every prompt has a fake-backed test; no test hits the live API by default.
- [ ] Structured responses are parsed defensively — a malformed reply degrades to
      "skip this question", never a crash.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=1.5  impl=0.24
item: api-integration        design=1.0  impl=0.48
item: greenfield-go-module   design=0.75 impl=0.32
item: smaller-go-module      design=0.2  impl=0.14
item: real-api-discovery     design=0.0  impl=0.18
item: milestone-review       design=0.0  impl=0.14
item: milestone-review       design=0.0  impl=0.14
item: atlas-docs             design=0.1  impl=0.06
design-buffer: 0.15
total: 5.78
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**What each item is.** `greenfield-go-module` ×2 — the `internal/llm` core
(contract, config, taxonomy, `Task[T]`/`Run[T]`, schema, `renderRequest`) and
`llmtest` (wire fake, captures, cassettes, obligation suite, golden), which are
separate concerns with separate test surfaces. `api-integration` — the SDK client
with retry, streaming, stall detection and the error mapping; the slug's own
definition is "API integration with batch + retry + tests", which is this exactly.
`smaller-go-module` — `define --llm-check`, extending an existing command surface.
`real-api-discovery` — the per-external-API budget: proxy shape, captures,
conformance. `milestone-review` ×2 — M1 and M2 are two real boundaries, so two.

**Step 2.5 (library availability) applied, and it moved two numbers.**
`api-integration` design halved 2.0 → 1.0: the official `anthropic-sdk-go` exists,
is already fetched and vetted, and collapses the wire-format, retry and SSE-parsing
design dialogue that the primitive's range assumes. `llmtest` design halved
1.5 → 0.75: `net/http/httptest` is stdlib and this repo already carries the pattern
to mirror (`fakeCDN` in `cmd/define/fetch_fake_test.go`). The `internal/llm` core
keeps full design hours — no library supplies a provider-independent contract or
the degradation taxonomy, which is where the actual decisions were.

**Step 3 (spec-quality discount) deliberately NOT applied — the honest call.**
The ×0.2 discount credits a spec that *pre-existed* the work. Here the plan was
authored inside the measurement window: `sdlc claim` ran before any design, so
today's brainstorm, the four plan-quality rounds and the prior-art study are all
inside what `sdlc actual` will measure. Discounting design to ~0.7h would produce a
row that reads 4× over for a reason that is an artifact of the method, not of the
work. The +15% design buffer *is* applied, since a thorough plan doc now exists for
the implementation half.

**Reconciliation.** Σdesign 3.55 × 1.15 = 4.0825; Σimpl 1.70 × 1.0 = 1.70;
total 5.78. Impl values are already written at v3.1's 40% of the v2 table, per the
model's instruction not to carry a separate scale field.

**Where this is most likely wrong.** The two `greenfield-go-module` design figures
are the soft numbers — if the wire fake turns out to be mostly mechanical once the
captures are in hand (they are already recorded and verified), design lands lower
and the row reads over. Conversely `api-integration` impl assumes the SDK behaves
as read; a surprise in streaming or `output_config` pass-through through the proxy
is the one thing that could double it.


## Plan

Design: `workshop/plans/000011-vocab-llm-plan.md` (authored 2026-08-22 via
`superpowers-writing-plans`, after `sdlc start-plan`).

Two review boundaries — each closes with its own `sdlc milestone-close`.

- [ ] M1 — transport, contract, wire fake. `internal/llm` contract; error
      taxonomy; config resolution pure over an env lookup; the stateful
      Anthropic-shaped `httptest` fake; the real client over `anthropic-sdk-go`
      driven at that fake; the obligation suite; the `AGENTS.local.md` carve-out.
- [ ] M2 — typed tasks, goldens, conformance. Recorded live SSE sample;
      `SchemaFor[T]` with a golden snapshot; `Task[T]`/`Run[T]` with defensive
      decode; `llmtest.Golden`; live conformance behind the build tag;
      `define --llm-check`; atlas page.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-08-22 — from a narrow seam to the base of a harness

**Reason.** The operator broadened `define-learn` to an adaptive program in which
the model authors material, classifies level, models the learner and answers
free-form questions — six tasks rather than the two this issue was scoped for. And
this is the first thing in `tools/` that talks to a model at all, so what gets built
here is what every later tool inherits.

**Delta.**

- **Scope: `internal/llm`, the transport and the contract.** Per-task prompts do NOT
  live here — they live with their consumers (#10 authoring, #12 veto, #13 grading,
  #16 Q&A, #17 reflection). This issue owns one transport, the fake, the response
  contract and the conformance check.
- **Operator override on `AGENTS.local.md`.** That file says `internal/` is earned
  on the *second* consumer, never the first. This creates it for the first,
  deliberately: a transport carrying auth, retries, a stateful fake and a live
  conformance check is exactly what the next tool would otherwise copy. Recorded
  loudly rather than done quietly; `AGENTS.local.md` gets the carve-out in the same
  change.
- **Access path.** Official `anthropic-sdk-go` speaking the Messages API, base URL
  and auth from config, defaulting to the local `cli-proxy-api` (measured running on
  `127.0.0.1:8317`) against a subscription plan, with a direct API key as fallback.
  One seam, so the fake and the conformance check do not fork.
- **Streaming is required, not optional.** #16 answers a question at a prompt; a
  paragraph that arrives all at once after four seconds reads as a hang.
- **Cost stops being a design constraint** (operator, 2026-08-22) — the goal is the
  best material achievable. "Cost and latency are per-question and user-visible;
  budget them explicitly" in the Spec above is superseded: still *report* them, no
  longer *budget* against them. Frontier models by default.
- **Degradation survives unchanged, and matters more.** With the seam unavailable
  the definition and the pronunciation still work, `--play` falls back to the local
  forms, and #16 says so plainly instead of guessing.

**Unchanged.** One method per task rather than a general chat call; the fake records
prompts so prompt regressions show up in a diff; structured responses parsed
defensively; live conformance behind the build tag.

### 2026-08-22 — one Done-when relocated (it cannot be satisfied here)

**Reason.** The Done-when *"`--play` runs a full session with the seam
unavailable, using local forms"* names a command that does not exist yet: `--play`
is #6, and forms 2.1/2.3 are #6/#7. Left here it would be either un-ticked
forever or ticked dishonestly.

**Delta.** That obligation moves to the issues that own the surface — #6 (the loop
degrades) and #12/#13 (the forms skip). What stays here is the property those
depend on and that IS testable now: `ErrUnavailable` is returned for no key, no
network, 429 and 5xx, and is distinguishable from `ErrRequest`, which stays loud.
Pinned by `errors_test.go` and by the obligation suite.

**Added, to keep the seam from being unexercised in a real binary until #16:**
`define --llm-check` — a diagnostic that runs one task through the real transport
and reports base URL, model, latency and usage. It also answers the operator's
"did my proxy config take" question, which is currently open (see the plan's
`## Open question`).
