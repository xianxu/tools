---
id: 000085
status: done
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 3.59
github_issue:
started: 2026-10-21T20:24:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: tools:1
    worktree: /Users/xianxu/workspace/worktree/tools-slot1/tools
    repository: github.com/xianxu/tools
actual_hours: 2.11
tracker:
    version: 1
    completion:
        token: close-a6134277bc25
        repository: github.com/xianxu/tools
        reviewed_head: 74ef0b692d07aac73f4ef9c2e01c6f38dc92fb38
        evidence_commit: 0a395de5c15a64016e721287592eda4414b07f4b
        landed_commit: dc8cf67a691401e8595ee65c5a77a08c39ee9003
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
