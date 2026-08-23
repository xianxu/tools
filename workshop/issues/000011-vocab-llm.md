---
id: 000011
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-22
estimate_hours: 7.98
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
familiarity: 1.5
item: greenfield-go-module   design=1.5  impl=0.32
item: api-integration        design=1.0  impl=0.60
item: greenfield-go-module   design=0.75 impl=0.28
item: smaller-go-module      design=0.2  impl=0.14
item: real-api-discovery     design=0.0  impl=0.18
item: milestone-review       design=0.0  impl=0.20
item: milestone-review       design=0.0  impl=0.20
item: milestone-review       design=0.0  impl=0.20
item: atlas-docs             design=0.15 impl=0.08
design-buffer: 0.30
total: 7.98
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

**Revised upward from 5.78 after the estimate-quality judge, which raised six
findings that all pointed the same way.** The first derivation passed the gate;
that is not a reason to keep a number I now believe is low, because a knowingly-low
estimate pollutes the calibration ledger exactly as much as a guessed one. What
changed and why:

- **`familiarity: 1.0 → 1.5`.** The block's own closing paragraph said an
  unexercised SDK streaming path or `output_config` pass-through through a
  third-party proxy "could double" the integration impl — which is a verbatim
  description of v2 Step 5's *novel-but-bounded* case. Naming a risk in prose and
  then declining to price it with the lever the model provides is having it both
  ways.
- **Three review boundaries, not two, at band-max hours.** The plan's own
  `### Issue close` dispatches a review, so M1 + M2 + close is three. And the
  primitive covers "one chunk" of review — dispatch and reading, not fixing. Checked
  against this repo's most recent comparable rather than guessed: #15's boundary
  remediation was `e37543a` (17:24) → `e5719f9` (17:45) → `8f04434` (18:07), ~43
  minutes across three commits, plus `d9d26ce` for M1's five findings. 0.20 each
  (band max) is still probably light for a greenfield transport.
- **`design-buffer: 0.15 → 0.30`.** v2.1's Step 6 rule of thumb is explicit: the
  +15% exists to avoid double-counting a ×0.2 spec discount, so *"if Step 3 was
  ×0.5 or ×1.0, keep the v2 +30%"*. Step 3 ran at ×1.0 here (see below), so taking
  the discount-adjacent buffer without the discount was reading the model
  selectively in the direction that lowered the number.
- **`api-integration` impl to band max (0.60).** The `## Robustness bar` puts work
  inside this primitive that its definition ("batch + retry + tests") does not
  cover: an idle-timeout stall detector, `Progress`/`OnSlow` across four phases, a
  header-dribbling test server, and thinking-block byte-preservation. There is no
  headroom left inside the band; if it overruns, the honest fix is a Method B
  sketch rather than a higher pick here.
- **The two `greenfield-go-module` impl figures were inverted.** The core carries
  Tasks 1, 2, 3, 9, 10, 11; `llmtest` carries 4, 6 and part of 11. Core 0.24 → 0.32,
  fake 0.32 → 0.28.
- **`atlas-docs` absorbs Task 7**, the `AGENTS.local.md` carve-out, which had no
  line of its own.

**Step 2.5 (library availability) applied, and it moved two numbers.**
`api-integration` design halved 2.0 → 1.0: the official `anthropic-sdk-go` exists,
is fetched and vetted at v1.66.0, and collapses the wire-format, retry and
SSE-parsing design dialogue the primitive's range assumes. `llmtest` design halved
1.5 → 0.75: `net/http/httptest` is stdlib and this repo already carries the pattern
to mirror (`fakeCDN`, `cmd/define/fetch_fake_test.go:18`). The `internal/llm` core
keeps full design hours — no library supplies a provider-independent contract or
the degradation taxonomy, which is where the decisions actually were.

**Step 3 (spec-quality discount) deliberately NOT applied.** The ×0.2 credits a
spec that *pre-existed* the work; here the plan was authored inside the measurement
window, because `sdlc claim` ran before any design. So today's brainstorm, four
plan-quality rounds and the prior-art study are all inside what `sdlc actual`
measures. `sdlc actual --issue 11` already reads 2.45h with no code committed —
against a ×0.2 discount that would have budgeted 0.71h for all design, the row
would be 3× under before a line of implementation exists.

**Reconciliation.** Σdesign 3.60 × 1.30 = 4.68; Σimpl 2.20 × 1.5 = 3.30;
total 7.98. Impl values are written at v3.1's 40% of the v2 table, per the model's
instruction not to carry a separate scale field.

**Where this is most likely wrong, now.** The design column: 3.60h before buffer
assumes in-implementation design dialogue across 14 tasks under a plan that
pre-resolves most decisions. If the plan holds up, design lands near the 2.45h
already spent and this row reads over. The judge's own advisory is worth recording
too — `sdlc actual` sums concurrent subagent spans additively, so the measured
number can exceed wall clock, and this window already does.

## Plan

Design: `workshop/plans/000011-vocab-llm-plan.md` (authored 2026-08-22 via
`superpowers-writing-plans`, after `sdlc start-plan`).

Two review boundaries — each closes with its own `sdlc milestone-close`.

- [x] M1 — transport, contract, wire fake. `internal/llm` contract; error
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

### 2026-08-22 — M1 built

Seven tasks: contract, taxonomy, config, wire fake, real client, obligation
suite, `AGENTS.local.md` carve-out. Full repo suite green (`go test ./...`).

Three things the build discovered that the plan did not predict:

1. **A malformed SSE frame is fatal, and the Lua lesson does not transfer.** The
   SDK's `ssestream` decoder owns framing, so a frame it refuses ends the stream
   and there is no skipping it from above. What transfers is the other half: once
   a frame has arrived the service is demonstrably reachable, so a stream dying
   mid-reply returns its partial text with `ErrTruncated` rather than
   `ErrUnavailable` — kbench's `is_outage` line, and the two need opposite
   responses from a caller.
2. **The proxy answers 502 for an unknown model**, where `api.anthropic.com`
   answers 400. So an our-bug-class error arrives as a 5xx and is absorbed as
   `ErrUnavailable` instead of staying loud as `ErrRequest`. The fake models the
   502 (a fake that answered 400 would pass here and fail live), and the suite
   asserts only that a typo errors rather than which member it lands in. Known
   limitation, not a defect to fix here: `--llm-check` (M2) is where a bad model
   should be caught loudly.
3. **One test could not fail, and only mutation found it.** `TestStreamDoes
   NotForwardThinkingDeltas` compared `onDelta`'s output against the thinking
   block's text — and the capture's `thinking_delta` carries `""` even under
   `display:summarized`, so there was nothing to leak and the assertion never
   ran. Forwarding thinking deltas left it GREEN. Rewritten to count calls
   against the capture's own frame counts, it now reports `6 want 5`. Filed to
   `workshop/lessons.md`.

Also: a stalled-stream fake that `time.Sleep`s blocks `httptest.Server.Close`,
which turned every stall test into a 30-second cleanup hang — the package suite
ran 65s instead of 6s. It waits on a cleanup channel now.

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
