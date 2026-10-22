---
id: 000085
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '62560ddb25d37a1f72dedecb29641e985bcd644a' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-21
