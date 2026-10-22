---
id: 000086
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# llm: prompt-cache breakpoints on System, Block and Tool

## Problem

The public `llm` seam (tools#85) has no way to mark a prompt-cache breakpoint. echo orders stable parts first
(system prompt, tool definitions, older history) so they cache, and its acceptance asks for cache controls to
pass through. Without a neutral surface the only route is authoring provider `Raw`, which the seam forbids. The
chat MVP doesn't need it; it matters for voice latency and cost. The echo TL scoped it out of #85 for a later
v0.x (review: ops-echo `teams/echo/projects/echo-control-board/reviews/tools-85-plan.md`).
