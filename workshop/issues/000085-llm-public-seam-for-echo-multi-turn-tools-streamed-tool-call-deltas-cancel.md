---
id: 000085
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5f2d777fa8c172a5f137768f41848fa7326bf563' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T20:24:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: tools:1
    worktree: /Users/xianxu/workspace/worktree/tools-slot1/tools
    repository: github.com/xianxu/tools
---

# llm: public seam for echo — multi-turn, tools, streamed tool-call deltas, cancel

## Problem

echo (ops-echo, echo#4 gateway actor) needs a provider-neutral LLM seam and
tools already owns one: `internal/llm`. Go's `internal/` rule keeps it out of
reach of any other module, and the seam today covers only what `define` uses:
single-turn completion with streamed text. echo needs multi-turn message
history, tool definitions, streamed tool-call deltas interleaved with text
deltas, and cancellation, behind the same seam. Dispatched by the echo TL
(ops-echo:0); echo consumes the result as an ordinary Go module at v0.1.0.

## Spec

Durable plan: `workshop/plans/000085-llm-public-seam-plan.md`. Its **Public API** section is the
contract under review by ops:0 (tools owner) and echo:1 (echo#4 gateway). In short: the package moves to
`github.com/xianxu/tools/llm` (+ `llm/llmtest`, + `conformance`, which `llmtest` imports). `Client` keeps two
methods, and `Stream` takes `func(Delta)` (text / tool-call start / input fragment / end). `Request` gains
`Messages`, `Tools` and `ToolChoice`; `Response` gains `ToolCalls` and `Message()`. A new `ErrCanceled` covers
caller cancellation. Retries stay in the transport, and `Config.BeforeAttempt` runs once per physical attempt.

## Done when

- [ ] `internal/llm` is a public package (e.g. `github.com/xianxu/tools/llm`)
      and `define` builds, tests and runs on the new import path.
- [ ] The seam carries multi-turn message history, tool definitions, streamed
      tool-call deltas alongside text deltas, and cancel, provider-neutral.
- [ ] The httptest wire fake, cassettes, goldens and live conformance suite
      grow with each new capability, so the fake still behaves like the real
      provider.
- [ ] The public API section was reviewed by ops:0 (tools owner) and the plan
      sent to ops-echo:0 before change-code.
- [ ] `v0.1.0` is tagged on the landed main commit and resolves via
      `go get github.com/xianxu/tools@v0.1.0`.

## Plan

- [ ] M1 — Publish `llm`, `llm/llmtest`, `conformance`; imports, rule text, atlas; no behavior change.
- [ ] M2 — Widen the seam (Messages, Tools, ToolCalls, Delta, ErrCanceled, BeforeAttempt, Attempts); fake + Suite grow; define on `TextOnly`.
- [ ] M3 — Real tool-use/multi-turn captures, live conformance, atlas rewrite, land, tag `v0.1.0`.

## Log

### 2026-10-21

- Dispatched by ops-echo:0 (echo TL); echo:1 (echo#4) coordinates on the seam directly.
- `start-plan` refused twice on a dirty resting branch. The weave catch-up's substrate step failed in the
  sandbox (a symlink under `.claude/skills` is denied; weave asks for `weave refresh --substrate` from the
  operator's shell) and left `.github/workflows/merge-check.yml` modified and `local-gates-guard.yml`
  untracked. Both are set aside in stash `tools85-weave-substrate-workflows` and reported to the hub; they
  overlap #84. The `.gitattributes` weave block is committed here, as the dispatch allowed.
- Found while planning: `llmtest/reachable.go` imports `internal/conformance`, so a public `llmtest` needs
  `conformance` public too (otherwise echo can't compile against it).
- SDK v1.66.0 `requestconfig.go`: the middleware chain runs inside the retry loop, once per attempt, which
  makes `BeforeAttempt` possible without a second retry loop.
- ops:0 approved the public API (all 5 points) with markups: refuse a Tool named auto/any/none (and an unknown ToolChoice) with ErrRequest; package doc states the v0.x break rule; tag only after landing. Folded into the plan's Revisions.
- echo:1 consumer review: added cache breakpoints (CacheSystem / Block / Tool, max 4), defined Usage.Attempts as dispatched attempts on every return path, documented and tested Delta/Response value ownership. Additive; sent to ops:0 for sight.
