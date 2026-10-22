---
id: 000086
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd3172d3ba64a57e77b5a07ea0fb77a6a666de934' # card fields mirrored from issue-cards; edit via sdlc
---

# llm: prompt-cache breakpoints on System, Block and Tool

## Problem

The public `llm` seam (tools#85) has no way to mark a prompt-cache breakpoint. echo orders stable parts first
(system prompt, tool definitions, older history) so they cache, and its acceptance asks for cache controls to
pass through. Without a neutral surface the only route is authoring provider `Raw`, which the seam forbids. The
chat MVP doesn't need it; it matters for voice latency and cost. The echo TL scoped it out of #85 for a later
v0.x (review: ops-echo `teams/echo/projects/echo-control-board/reviews/tools-85-plan.md`).

## Spec

A sketch from #85's design discussion, to revisit:
- `Request.CacheSystem bool`, `Block.CacheBreakpoint bool`, `Tool.CacheBreakpoint bool`. Each means
  "everything up to here is a stable prefix", and each becomes `cache_control: {"type":"ephemeral"}` on that
  element. A marked `Raw` block gets the key merged into a copy of its bytes.
- At most 4 per request (Anthropic's limit). A fifth is refused with `ErrRequest`, never dropped silently.
- `renderRequest` shows breakpoints only when set, so existing hashes stay byte-identical.
- Advisory: no hit is promised, and a non-Claude provider behind the proxy may ignore the marker.
  `Usage.CacheReadTokens` reports what happened.
- TTL stays the provider default until a consumer needs one.

## Done when

- [ ] A consumer can mark cache breakpoints on System, a Block and a Tool without authoring `Raw`.
- [ ] The fake records breakpoints (`Recorded.CacheBreakpoints()`), and wire tests pin the `cache_control` shape.
- [ ] Live conformance shows a second identical call reading from cache on direct Claude.
- [ ] Released as a v0.x minor bump.

## Plan

- [ ]

## Log

### 2026-10-21
