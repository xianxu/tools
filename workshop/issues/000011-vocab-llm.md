---
id: 000011
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-22
estimate_hours:
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

- [ ] `--play` runs a full session with the seam unavailable, using local forms.
- [ ] Every prompt has a fake-backed test; no test hits the live API by default.
- [ ] Structured responses are parsed defensively — a malformed reply degrades to
      "skip this question", never a crash.

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
