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

- [ ] Design via `sdlc start-plan` before implementing.

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
